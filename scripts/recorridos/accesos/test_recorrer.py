"""Prueba focal sintética: no conecta con VEC ni usa certificados reales."""

import contextlib
import hashlib
import io
import json
import os
import sys
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import patch

import recorrer


class PaginaFalsa:
    def __init__(self, fallar=False):
        self.fallar = fallar

    def goto(self, *_args, **_kwargs):
        return types.SimpleNamespace(status=200, url="https://127.0.0.1:18443/portal-empleado/")

    def evaluate(self, _script, prueba=None):
        if prueba is None:
            return 0
        if self.fallar:
            self.fallar = False
            return {"estado": 403, "json": True, "comprobado": False, "sinDatos": True}
        return {"estado": prueba["estado"], "json": True, "comprobado": True, "sinDatos": True}


class ContextoFalso:
    def __init__(self, pagina):
        self.pagina = pagina

    def new_page(self):
        return self.pagina

    def route(self, *_args):
        pass

    def cookies(self):
        return []

    def close(self):
        pass


class NavegadorFalso:
    def __init__(self, pagina):
        self.pagina = pagina
        self.contextos = 0

    def new_context(self, **_kwargs):
        self.contextos += 1
        return ContextoFalso(self.pagina)

    def close(self):
        pass


class PlaywrightFalso:
    def __init__(self, navegador):
        self.chromium = types.SimpleNamespace(launch=lambda **_kwargs: navegador)

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        pass


class RecorridoTest(unittest.TestCase):
    def setUp(self):
        self.enterContext(patch.object(recorrer, "CHROME", Path(sys.executable)))
        self.tmp = tempfile.TemporaryDirectory(prefix="vec-accesos-sintetico-")
        self.addCleanup(self.tmp.cleanup)
        base = Path(self.tmp.name)
        binario = Path(sys.executable).resolve()
        clave = base / "clave"
        clave.write_bytes(b"clave sintetica")
        clave.chmod(0o600)
        prueba_positiva = {"clase": "permiso", "metodo": "GET", "ruta": "/api/vec/session",
                           "estado": 200, "comprobacion": {"campo": "data", "tipo": "objeto"}}
        prueba_negativa = {"clase": "denegacion", "metodo": "GET",
                           "ruta": "/api/vec/contratacion-temporal/peticiones-centro/rrhh", "estado": 403}
        self.plan = {"origen": "https://127.0.0.1:18443",
                     "evidencia": {"clon_h3_h5": True, "binario_en_uso": True,
                                   "binario": str(binario), "pid": os.getpid(),
                                   "sha256_binario": hashlib.sha256(binario.read_bytes()).hexdigest()},
                     "perfiles": {}}
        for nombre in recorrer.PERFILES:
            certificado = base / f"certificado-{nombre}"
            certificado.write_bytes(f"certificado sintetico {nombre}".encode())
            self.plan["perfiles"][nombre] = {"certificado": str(certificado), "clave": str(clave),
                                             "pruebas": [prueba_positiva.copy(), prueba_negativa.copy()]}
        self.ruta = base / "plan.json"
        self.escribir()

    def escribir(self):
        self.ruta.write_text(json.dumps(self.plan), encoding="utf-8")
        self.ruta.chmod(0o600)

    def test_plan_completo_y_puerta_sin_ejecucion(self):
        self.assertEqual(recorrer.validar_plan(self.ruta), self.plan)
        with contextlib.redirect_stdout(io.StringIO()) as salida:
            self.assertEqual(recorrer.main(["--plan", str(self.ruta)]), 2)
        self.assertIn("NO EJECUTADO", salida.getvalue())

    def test_falta_denegacion_falla_cerrado(self):
        self.plan["perfiles"]["rrhh"]["pruebas"] = self.plan["perfiles"]["rrhh"]["pruebas"][:1]
        self.escribir()
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.validar_plan(self.ruta)

    def test_recorrido_sintetico_cinco_perfiles_y_primer_corte(self):
        modulo = types.ModuleType("playwright.sync_api")
        navegador = NavegadorFalso(PaginaFalsa())
        modulo.sync_playwright = lambda: PlaywrightFalso(navegador)
        with patch.dict("sys.modules", {"playwright.sync_api": modulo}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(recorrer.recorrer(self.plan), 0)
        self.assertEqual(navegador.contextos, 5)
        navegador = NavegadorFalso(PaginaFalsa(fallar=True))
        modulo.sync_playwright = lambda: PlaywrightFalso(navegador)
        with patch.dict("sys.modules", {"playwright.sync_api": modulo}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(recorrer.recorrer(self.plan), 1)
        self.assertEqual(navegador.contextos, 1)


if __name__ == "__main__":
    unittest.main()
