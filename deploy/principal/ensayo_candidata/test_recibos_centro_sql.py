"""Pruebas del cotejo SQL de recibos originales, sin abrir Docker ni PostgreSQL."""

import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location(
    "hz39_recibos_centro_sql", Path(__file__).with_name("recibos_centro_sql.py")
)
recibos = importlib.util.module_from_spec(spec)
spec.loader.exec_module(recibos)

CONTENEDOR = "vec-codexp-hz39-pg-20261014"
PETICION = "peticion:centro:prueba-123"
FECHA_1 = "2026-10-14T23:00:00Z"
FECHA_2 = "2026-10-14T23:01:00Z"


def originales():
    return {
        "01": {"peticion_ref": PETICION, "recibo_ref": "recibo:centro:uno",
               "version": 1, "registrado_en": FECHA_1},
        "02": {"peticion_ref": PETICION, "recibo_ref": "recibo:centro:dos",
               "version": 2, "registrado_en": FECHA_2},
    }


def filas():
    return [
        {**originales()[paso], "estado": estado, "integridad": True}
        for paso, estado in (("01", "pendiente_ratificacion"), ("02", "ratificada"))
    ]


def previos():
    return {"pasos": [
        {"id": paso, "recibos_efecto": [dict(originales()[paso])]} for paso in ("01", "02")
    ]}


def bandeja(*, referencia=PETICION, version=3, estado="entregada"):
    return {"data": {"peticiones": [
        {"referencia": referencia, "version": version, "estado": estado}
    ]}}


class RecibosCentroSQL(unittest.TestCase):
    def test_dos_recibos_originales_se_cotejan_con_sql_de_solo_lectura(self):
        respuesta = b"\n".join(json.dumps(fila).encode() for fila in filas()) + b"\n"
        with patch.object(recibos.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=respuesta)) as ejecutar:
            resultado = recibos.cotejar_recibos(originales(), contenedor=CONTENEDOR)
        self.assertEqual(resultado["01"]["recibo_ref"], "recibo:centro:uno")
        self.assertEqual(resultado["02"]["registrado_en"], FECHA_2)
        orden = ejecutar.call_args.args[0]
        self.assertEqual(orden[:6], ["docker", "exec", "-i", "--user", "postgres", "-e"])
        self.assertIn("PGPASSWORD=", orden)
        self.assertIn("default_transaction_read_only=on", " ".join(orden))
        self.assertEqual(orden[orden.index("-p") + 1], "55639")
        self.assertEqual(orden[-2:], ["-f", "-"])
        self.assertIn("WHERE r.recibo_ref IN (:'recibo_01', :'recibo_02')", ejecutar.call_args.kwargs["input"].decode())
        self.assertIn("r.recibo_json->>'estado'=r.estado", ejecutar.call_args.kwargs["input"].decode())
        self.assertNotIn("recibo:centro:uno", ejecutar.call_args.kwargs["input"].decode())
        self.assertEqual(ejecutar.call_args.kwargs["env"]["DOCKER_HOST"], "unix:///var/run/docker.sock")
        self.assertNotIn("shell", ejecutar.call_args.kwargs)

    def test_entrada_ajena_invalida_no_lanza_proceso(self):
        casos = [
            ({**originales(), "03": originales()["01"]}, CONTENEDOR, 55639),
            (originales(), "vec-otra-cosa-pg-20261014", 55639),
            (originales(), CONTENEDOR, 0),
        ]
        inyeccion = originales()
        inyeccion["01"]["recibo_ref"] = "recibo'; DROP TABLE x; --"
        casos.append((inyeccion, CONTENEDOR, 55639))
        with patch.object(recibos.subprocess, "run") as ejecutar:
            for entrada, contenedor, puerto in casos:
                with self.subTest(contenedor=contenedor, puerto=puerto):
                    with self.assertRaises(recibos.ReciboNoConfirmado):
                        recibos.cotejar_recibos(entrada, contenedor=contenedor, puerto=puerto)
        ejecutar.assert_not_called()

    def test_falta_duplicado_o_divergencia_fallan_cerrado(self):
        casos = [
            filas()[:1],
            [filas()[0], filas()[0]],
            [{**filas()[0], "integridad": False}, filas()[1]],
            [filas()[0], {**filas()[1], "registrado_en": FECHA_1}],
        ]
        with patch.object(recibos.subprocess, "run") as ejecutar:
            for caso in casos:
                ejecutar.return_value = SimpleNamespace(
                    returncode=0,
                    stdout=b"\n".join(json.dumps(fila).encode() for fila in caso) + b"\n",
                )
                with self.assertRaises(recibos.ReciboNoConfirmado):
                    recibos.cotejar_recibos(originales(), contenedor=CONTENEDOR)

    def test_instante_z_y_offset_utc_son_el_mismo_microsegundo(self):
        salida_sql = filas()
        salida_sql[0]["registrado_en"] = "2026-10-14T23:00:00+00:00"
        salida_sql[1]["registrado_en"] = "2026-10-14T23:01:00.000000+00:00"
        respuesta = b"\n".join(json.dumps(fila).encode() for fila in salida_sql) + b"\n"
        with patch.object(recibos.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=respuesta)):
            resultado = recibos.cotejar_recibos(originales(), contenedor=CONTENEDOR)
        self.assertEqual(resultado["01"]["registrado_en"], FECHA_1)
        self.assertEqual(resultado["01"]["registrado_en_sql"], "2026-10-14T23:00:00+00:00")

    def test_un_microsegundo_distinto_o_precision_no_representable_fallan(self):
        un_microsegundo = filas()
        un_microsegundo[0]["registrado_en"] = "2026-10-14T23:00:00.000001+00:00"
        respuesta = b"\n".join(json.dumps(fila).encode() for fila in un_microsegundo) + b"\n"
        with patch.object(recibos.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=respuesta)):
            with self.assertRaises(recibos.ReciboNoConfirmado):
                recibos.cotejar_recibos(originales(), contenedor=CONTENEDOR)
        original = originales()
        original["01"]["registrado_en"] = "2026-10-14T23:00:00.0000001Z"
        with patch.object(recibos.subprocess, "run") as ejecutar:
            with self.assertRaises(recibos.ReciboNoConfirmado):
                recibos.cotejar_recibos(original, contenedor=CONTENEDOR)
            ejecutar.assert_not_called()

    def test_estado_original_y_estado_sql_deben_concordar(self):
        original = originales()
        original["01"]["estado"] = "ratificada"
        with patch.object(recibos.subprocess, "run") as ejecutar:
            with self.assertRaises(recibos.ReciboNoConfirmado):
                recibos.cotejar_recibos(original, contenedor=CONTENEDOR)
            ejecutar.assert_not_called()
        original["01"]["estado"] = "pendiente_ratificacion"
        original["02"]["estado"] = "ratificada"
        respuesta = b"\n".join(json.dumps(fila).encode() for fila in filas()) + b"\n"
        with patch.object(recibos.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=respuesta)):
            self.assertEqual(recibos.cotejar_recibos(original, contenedor=CONTENEDOR)["02"]["estado"], "ratificada")

    def test_preparar_recuperacion_usa_recibos_unicos_y_checkpoint_final(self):
        previos_sin_ref = previos()
        for paso in previos_sin_ref["pasos"]:
            del paso["recibos_efecto"][0]["peticion_ref"]
            paso["recibos_efecto"][0]["estado"] = (
                "pendiente_ratificacion" if paso["id"] == "01" else "ratificada"
            )
        with patch.object(recibos, "cotejar_recibos", return_value={"01": filas()[0], "02": filas()[1]}) as sql:
            preparada = recibos.preparar_recuperacion(
                previos_sin_ref, {"peticion_ref": PETICION}, bandeja(),
                contenedor=CONTENEDOR,
            )
        self.assertEqual(sql.call_count, 1)
        esperados = originales()
        esperados["01"]["estado"] = "pendiente_ratificacion"
        esperados["02"]["estado"] = "ratificada"
        self.assertEqual(sql.call_args.args[0], esperados)
        self.assertEqual(preparada["peticion"],
                         {"referencia": PETICION, "version": 3, "estado": "entregada"})
        self.assertEqual(recibos.cotejar_bandeja(bandeja(), preparada), preparada["peticion"])

    def test_checkpoint_y_bandeja_difieren_en_referencia_version_o_estado(self):
        preparada = {"peticion": {"referencia": PETICION, "version": 3, "estado": "entregada"}}
        for actual in (
            bandeja(referencia="peticion:centro:otra"),
            bandeja(version=2),
            bandeja(estado="ratificada"),
            {"data": {"peticiones": []}},
            {"data": {"peticiones": bandeja()["data"]["peticiones"] * 2}},
        ):
            with self.subTest(actual=actual):
                with self.assertRaises(recibos.ReciboNoConfirmado):
                    recibos.cotejar_bandeja(actual, preparada)

    def test_preparacion_rechaza_ausencia_duplicado_y_checkpoint_divergente_sin_sql(self):
        sin_recibo = previos()
        sin_recibo["pasos"][0]["recibos_efecto"] = []
        duplicado = previos()
        duplicado["pasos"][1]["recibos_efecto"] *= 2
        con_ref_ajena = previos()
        con_ref_ajena["pasos"][0]["recibos_efecto"][0]["peticion_ref"] = "peticion:centro:otra"
        for anterior, checkpoint in (
            (sin_recibo, bandeja()),
            (duplicado, bandeja()),
            (con_ref_ajena, bandeja()),
            (previos(), bandeja(referencia="peticion:centro:otra")),
            (previos(), {"data": {"peticiones": bandeja()["data"]["peticiones"] * 2}}),
            (previos(), bandeja(version=True)),
        ):
            with self.subTest(anterior=anterior, checkpoint=checkpoint):
                with patch.object(recibos, "cotejar_recibos") as sql:
                    with self.assertRaises(recibos.ReciboNoConfirmado):
                        recibos.preparar_recuperacion(
                            anterior, {"peticion_ref": PETICION}, checkpoint,
                            contenedor=CONTENEDOR,
                        )
                    sql.assert_not_called()


if __name__ == "__main__":
    unittest.main()
