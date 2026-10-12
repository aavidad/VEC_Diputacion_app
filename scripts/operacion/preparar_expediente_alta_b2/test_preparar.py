import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from preparar import comando_catalogo_faltante, comando_psql, cuerpo_canonico, guardar, cargar, http, lectura_db, serializar_cuerpo


class CanonTrasReinicio(unittest.TestCase):
    def test_comunicacion_conserva_bytes_canonicos_tras_guardar_y_cargar(self):
        cuerpo = {
            "organizacion_ref": "organizacion:desarrollo:dipgra",
            "expediente_ref": "expediente:ct:uno",
            "llamamiento_ref": "llamamiento:uno",
            "version_esperada": 1,
            "prueba_entrega_ref": "recibo:uno",
            "clave_idempotencia": "11111111-1111-4111-8111-111111111111",
        }
        with tempfile.TemporaryDirectory() as folder:
            ruta = Path(folder) / "estado.json"
            guardar(ruta, {"esquema": "vec.preparar-alta-b2.v1", "operaciones": {"comunicacion": {"cuerpo": cuerpo}}})
            cargado = cargar(ruta)["operaciones"]["comunicacion"]["cuerpo"]
        esperado = ('{"clave_idempotencia":"11111111-1111-4111-8111-111111111111",'
                    '"organizacion_ref":"organizacion:desarrollo:dipgra",'
                    '"expediente_ref":"expediente:ct:uno","llamamiento_ref":"llamamiento:uno",'
                    '"version_esperada":1,"prueba_entrega_ref":"recibo:uno"}')
        self.assertEqual(serializar_cuerpo(cuerpo_canonico("comunicacion", cargado)), esperado.encode())

    def test_propuesta_reordena_snapshots_tras_guardar_y_cargar(self):
        snap = {"huella_sha256": "a" * 64, "version": 1, "referencia": "catalogo:uno"}
        cuerpo = {"clave_idempotencia": "11111111-1111-4111-8111-111111111111",
                  "expediente_ref": "expediente:ct:uno", "llamamiento_ref": "llamamiento:uno",
                  "resolucion_llamamiento_aceptada_ref": "resolucion:uno",
                  "recibo_resolucion_aceptada_ref": "recibo:uno", "version_esperada": 6,
                  "tipo_formalizacion": snap, "plantilla": snap, "anexos": [],
                  "politica_firma": snap, "plan_firma": snap}
        with tempfile.TemporaryDirectory() as folder:
            ruta = Path(folder) / "estado.json"
            guardar(ruta, {"esquema": "vec.preparar-alta-b2.v1", "operaciones": {"propuesta": {"cuerpo": cuerpo}}})
            cargado = cargar(ruta)["operaciones"]["propuesta"]["cuerpo"]
        wire = serializar_cuerpo(cuerpo_canonico("propuesta", cargado))
        self.assertIn(b'"tipo_formalizacion":{"referencia":"catalogo:uno","version":1,"huella_sha256":', wire)
        self.assertLess(wire.index(b'"tipo_formalizacion"'), wire.index(b'"anexos"'))
        self.assertLess(wire.index(b'"anexos"'), wire.index(b'"politica_firma"'))

    def test_entorno_hostil_no_cambia_destinos_locales(self):
        with tempfile.TemporaryDirectory() as folder:
            material = Path(folder)
            (material / "ca").mkdir()
            (material / "ca/ca.crt").write_text("CA de prueba")
            args = SimpleNamespace(motor="docker", contenedor="pg-prueba",
                                   material=material, origen="https://127.0.0.1:29445",
                                   estado=material / "privado/estado.json")
            with patch.dict(os.environ, {"CURL_HOME": "/tmp/curl-ajeno",
                                      "DOCKER_HOST": "tcp://remoto:2375", "DOCKER_CONTEXT": "remoto"}):
                self.assertEqual(comando_psql(args)[:6],
                                 ["docker", "--host", "unix:///var/run/docker.sock", "exec", "-i", "pg-prueba"])
                with patch("preparar.subprocess.run", return_value=subprocess.CompletedProcess([], 0, b"BEGIN\nSET\n1\nROLLBACK\n", b"")) as sql_call:
                    self.assertEqual(lectura_db(args, "SELECT 1"), ["1"])
                self.assertNotIn("DOCKER_HOST", sql_call.call_args.kwargs["env"])
                self.assertNotIn("DOCKER_CONTEXT", sql_call.call_args.kwargs["env"])
                with patch("preparar.mTLS", return_value=(material / "actor.crt", material / "actor.key")), \
                     patch("preparar.subprocess.run", return_value=subprocess.CompletedProcess([], 0, b'{"data":{}}\n200', b"")) as call:
                    self.assertEqual(http(args, "RRHH", "GET", "/api/vec/prueba")[0], 200)
                argv = call.call_args.args[0]
                self.assertEqual(argv[:4], ["curl", "-q", "--noproxy", "*"])
                self.assertEqual(argv[-1], "https://127.0.0.1:29445/api/vec/prueba")
                with patch("preparar.mTLS", return_value=(material / "actor.crt", material / "actor.key")):
                    impreso = comando_catalogo_faltante(args, "modalidad:vacante")
                self.assertIn("curl -q --noproxy '*'", impreso)


if __name__ == "__main__":
    unittest.main()
