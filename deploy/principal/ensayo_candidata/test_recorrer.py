"""Guardas del guion: ninguna petición con efecto al recuperar, sin secretos."""

import importlib.util
import json
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("hz39_recorrer", ROOT / "recorrer.py")
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class GuionHZ39(unittest.TestCase):
    def test_recuperacion_bloquea_un_acto_antes_de_enviarlo(self):
        class Request:
            method = "POST"
            url = "https://127.0.0.1:9443/api/vec/contratacion-temporal/analisis"

        class Route:
            request = Request()
            aborted = False

            def abort(self):
                self.aborted = True

        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        e.actual = {"red_fallida": []}
        e.solo_lectura = True
        r = Route()
        with patch.object(runner, "responder_sin_redireccion") as enviar:
            e.filtrar(r, "https://127.0.0.1:9443")
            enviar.assert_not_called()
        self.assertTrue(r.aborted)
        self.assertEqual(
            e.actual["red_fallida"][0]["codigo"], "efecto_bloqueado_en_recuperacion"
        )

    def test_intencion_no_guarda_credenciales_del_cuerpo(self):
        class Request:
            method = "POST"
            url = "https://127.0.0.1:9443/api/vec/acto"
            post_data_json = {
                "clave_idempotencia": "clave-123",
                "password": "no_guardar",
                "sesion_token": "no_guardar",
            }
            post_data = json.dumps(post_data_json)

        class Route:
            request = Request()

        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        e.actual = {"actor": "rrhh", "id": "01"}
        with (
            patch.object(runner, "escribir") as guardar,
            patch.object(runner, "responder_sin_redireccion"),
        ):
            e.filtrar(Route(), "https://127.0.0.1:9443")
            contenido = guardar.call_args.args[2]
        self.assertNotIn("no_guardar", json.dumps(contenido))
        self.assertEqual(contenido["clave_idempotencia"], "clave-123")

    def test_firma_parcial_o_sin_verificador_no_cierra_paso(self):
        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        clave = ("POST", "/api/vec/contratacion-temporal/firmas-documento/consultas-v2")
        e.respuestas[clave] = {
            "data": {
                "verificacion_disponible": True,
                "documentos": [
                    {"documento": "resolucion", "completo": True, "pasos": []}
                ],
            }
        }
        self.assertFalse(e.firmas_completas("15"))
        e.respuestas[clave]["data"]["verificacion_disponible"] = False
        self.assertFalse(e.firmas_completas("07b"))

    def test_catalogo_y_referencias_faltantes_no_se_inventan(self):
        self.assertEqual(len(runner.catalogo(ROOT / "pasos_hz39.json")), 28)
        with self.assertRaises(runner.EntradaInvalida):
            runner.resolver("$expediente_ref", {})
        with self.assertRaises(runner.EntradaInvalida):
            runner.ruta_local("//cidonia.cloud/api")


if __name__ == "__main__":
    unittest.main()
