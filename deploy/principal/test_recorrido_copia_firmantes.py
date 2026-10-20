"""Guardas de recibos AUT53 y sucesoras sin activar servicios externos."""

import hashlib
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from recorrido_copia import firmantes


class FirmantesTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.raiz.chmod(0o700)

    def poner(self, nombre, dato):
        p = self.raiz / nombre
        p.write_text(json.dumps(dato, separators=(",", ":")))
        p.chmod(0o600)
        return p

    def test_recibo_rechaza_huella_de_plan_y_rol_distintos(self):
        fila = {"version_rol_ref": "rol:ct_cargo_tecnico_consulta_dev:v2",
                "version_rol_sha256": "a" * 64, "control_sha256": "b" * 64}
        plan = self.poner("plan.json", {"cargos": [{"rol_id": "ct_cargo_tecnico_consulta_dev",
                                                  "version": 2, "version_rol_sha256": "a" * 64}]})
        recibo = self.poner("acuse.json", {"estado": "permitido", "recibo": {
            "auditoria_ref": "aud_1", "plan_sha256": hashlib.sha256(plan.read_bytes()).hexdigest(),
            "cargos": [fila]}})
        self.assertIn(fila["version_rol_ref"], firmantes._roles(recibo, plan, 1))
        alterado = json.loads(recibo.read_text())
        alterado["recibo"]["plan_sha256"] = "0" * 64
        self.poner("acuse.json", alterado)
        with self.assertRaisesRegex(ValueError, "recibo AUT53"):
            firmantes._roles(recibo, plan, 1)
        alterado["recibo"]["plan_sha256"] = hashlib.sha256(plan.read_bytes()).hexdigest()
        alterado["recibo"]["cargos"][0]["version_rol_sha256"] = "c" * 64
        self.poner("acuse.json", alterado)
        with self.assertRaisesRegex(ValueError, "huellas de roles"):
            firmantes._roles(recibo, plan, 1)

    def test_cuatro_consultas_exigen_ocho_preimagenes_v1_y_cambian_solo_lectura(self):
        plantilla = json.loads((Path(__file__).parent / "candidata_hz39/firma/semilla_firmantes/cargos-roles.template.json").read_bytes())
        for row in plantilla["cargos"]:
            row["organizacion_ref"] = "organizacion:desarrollo:propia"
            row["vigente_desde"] = "2026-10-20T00:00:00Z"
            row["vigente_hasta"] = "2027-01-16T00:00:00Z"
        inicial = self.raiz / "inicial"
        inicial.mkdir(mode=0o700)
        self.poner("inicial/cargos-firma-fuente.json", plantilla)
        roles = {}
        for tipo in ("firma", "consulta"):
            for role in firmantes.ROLES:
                roles[f"rol:ct_cargo_{role}_{tipo}_dev:v1"] = {"version_rol_sha256": "a" * 64}

        class Contexto:
            def ejecutar(self, argv, log, cwd=None, env=None):
                if argv[0] != sys.executable:
                    raise AssertionError("proceso externo inesperado")
                return subprocess.run(argv, capture_output=True, check=False)

        with patch.object(firmantes, "_aut53", return_value=({}, None, None)) as aut53:
            firmantes._sucesoras(Contexto(), self.raiz, inicial, roles)
        fuente = json.loads((self.raiz / "sucesoras/cargos-consulta-v2.json").read_text())
        self.assertEqual(len(fuente["cargos"]), 4)
        self.assertTrue(all("consultar_detalle_selectores" in x["operaciones_v2"]
                            and "consultar_detalle" not in x["operaciones_v2"]
                            and x["version_anterior_sha256"] == "a" * 64
                            for x in fuente["cargos"]))
        self.assertEqual(aut53.call_args.args[3:], ("consulta-v2", 4))
        del roles["rol:ct_cargo_tecnico_consulta_dev:v1"]
        with self.assertRaisesRegex(ValueError, "preimagen V1"):
            firmantes._sucesoras(Contexto(), self.raiz, inicial, roles)

    def test_recibo_no_sustituye_estado_de_base_de_la_copia(self):
        ref = "rol:ct_cargo_tecnico_consulta_dev:v1"
        recibo = {ref: {"version_rol_sha256": "a" * 64, "control_sha256": "b" * 64}}

        class Consulta:
            def __init__(self, actual):
                self.actual = actual

            def consultar(self, sql):
                self.sql = sql
                return json.dumps(self.actual)

        propia = Consulta({ref: {"huella": "a" * 64, "control": "b" * 64,
                                 "estado": "habilitada"}})
        firmantes._confirmar_db(propia, recibo)
        self.assertIn("control_vigencia_version_rol_actual", propia.sql)
        ajena = Consulta({ref: {"huella": "a" * 64, "control": "c" * 64,
                                "estado": "habilitada"}})
        with self.assertRaisesRegex(ValueError, "divergen del recibo"):
            firmantes._confirmar_db(ajena, recibo)

    def test_certificado_largo_no_prolonga_asignaciones(self):
        ahora = datetime(2026, 10, 20, 12, tzinfo=timezone.utc)
        desde = ahora - timedelta(hours=1)
        hasta = desde + timedelta(days=90)
        retirada = ahora + timedelta(days=88)
        fmt = lambda v: v.strftime("%Y-%m-%dT%H:%M:%SZ")
        prep = {"roles_vigente_desde": fmt(desde), "roles_vigente_hasta": fmt(hasta)}
        sql = (fmt(hasta) + "\n").encode() * 32
        resultado = firmantes._limitar_asignaciones(sql, prep, fmt(retirada), ahora)
        self.assertEqual(resultado.count(fmt(retirada).encode()), 32)
        self.assertNotIn(fmt(hasta).encode(), resultado)
        # La vigencia del X.509 puede ser 397 días; usarla como autoridad falla.
        with self.assertRaisesRegex(ValueError, "60 a 90"):
            firmantes._limitar_asignaciones(sql, prep, fmt(ahora + timedelta(days=397)), ahora)


if __name__ == "__main__":
    unittest.main()
