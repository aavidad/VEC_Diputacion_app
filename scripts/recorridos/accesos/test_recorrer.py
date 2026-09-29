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

    def on(self, *_args):
        pass

    async def screenshot(self, **_kwargs):
        return b"captura sintetica"

    async def goto(self, *_args, **_kwargs):
        return types.SimpleNamespace(status=200, url="https://127.0.0.1:18443/portal-empleado/")

    async def evaluate(self, _script, prueba=None):
        if prueba is None:
            return 0
        if self.fallar:
            self.fallar = False
            return {"estado": 403, "json": True, "comprobado": False, "sinDatos": True}
        return {"estado": prueba["estado"], "json": True, "comprobado": True, "sinDatos": True}


class ContextoFalso:
    def __init__(self, pagina):
        self.pagina = pagina

    async def new_page(self):
        return self.pagina

    async def route(self, *_args):
        pass

    async def route_web_socket(self, *_args):
        pass

    async def cookies(self):
        return []

    async def close(self):
        pass


class NavegadorFalso:
    def __init__(self, pagina):
        self.pagina = pagina
        self.contextos = 0

    async def new_context(self, **_kwargs):
        self.contextos += 1
        return ContextoFalso(self.pagina)

    async def close(self):
        pass


class PlaywrightFalso:
    def __init__(self, navegador):
        async def launch(**_kwargs):
            return navegador
        self.chromium = types.SimpleNamespace(launch=launch)

    async def __aenter__(self):
        return self

    async def __aexit__(self, *_args):
        pass


class RecorridoTest(unittest.TestCase):
    def test_catalogos_tienen_mismas_claves(self):
        carpeta = Path(recorrer.__file__).parent
        es = json.loads((carpeta / "mensajes.es.json").read_text(encoding="utf-8"))
        en = json.loads((carpeta / "mensajes.en.json").read_text(encoding="utf-8"))
        self.assertEqual(set(es), set(en))

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

    def test_rechaza_plan_y_material_en_otro_git(self):
        otro = Path(self.tmp.name) / "otro-trabajo"
        otro.mkdir()
        (otro / ".git").write_text("gitdir: /otro/lugar", encoding="utf-8")
        plan_git = otro / "plan.json"
        plan_git.write_text(json.dumps(self.plan), encoding="utf-8")
        plan_git.chmod(0o600)
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.validar_plan(plan_git)
        certificado_git = otro / "certificado"
        certificado_git.write_bytes(b"certificado de otro worktree")
        certificado_original = self.plan["perfiles"]["rrhh"]["certificado"]
        self.plan["perfiles"]["rrhh"]["certificado"] = str(certificado_git)
        self.escribir()
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.validar_plan(self.ruta)
        clave_git = otro / "clave"
        clave_git.write_bytes(b"clave de otro worktree")
        clave_git.chmod(0o600)
        self.plan["perfiles"]["rrhh"]["certificado"] = certificado_original
        self.plan["perfiles"]["rrhh"]["clave"] = str(clave_git)
        self.escribir()
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.validar_plan(self.ruta)

    def test_perfil_movil_conserva_primer_corte_en_salida_privada(self):
        salida = recorrer.preparar_salida(Path(self.tmp.name) / "evidencia")
        modulo = types.ModuleType("playwright.async_api")
        navegador = NavegadorFalso(PaginaFalsa(fallar=True))
        modulo.async_playwright = lambda: PlaywrightFalso(navegador)
        with patch.dict("sys.modules", {"playwright.async_api": modulo}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(recorrer.recorrer(self.plan, perfil="centro", ancho=390, salida=salida), 1)
        self.assertEqual(navegador.contextos, 1)
        datos = json.loads((salida / "centro-390.json").read_text())
        self.assertEqual(datos["perfil"], "centro")
        self.assertEqual(datos["ancho"], 390)
        self.assertEqual(datos["entrada_http"], 200)
        self.assertEqual(len(datos["pruebas"]), 1)
        self.assertEqual(datos["pruebas"][0]["estado"], 403)
        self.assertEqual(datos["estado"], "prueba")
        for fichero in salida.iterdir():
            self.assertEqual(fichero.stat().st_mode & 0o077, 0)
        self.assertNotIn("certificado", (salida / "centro-390.json").read_text())

    def test_salida_rechaza_git_reutilizacion_y_enlaces(self):
        git = Path(self.tmp.name) / "git"
        git.mkdir()
        (git / ".git").mkdir()
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.preparar_salida(git / "evidencia")
        salida = recorrer.preparar_salida(Path(self.tmp.name) / "salida")
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.preparar_salida(salida)
        enlace = Path(self.tmp.name) / "enlace"
        enlace.symlink_to(salida, target_is_directory=True)
        with self.assertRaises(recorrer.PlanInvalido):
            recorrer.preparar_salida(enlace / "evidencia")

    def test_recorrido_sintetico_cinco_perfiles_y_primer_corte(self):
        modulo = types.ModuleType("playwright.async_api")
        navegador = NavegadorFalso(PaginaFalsa())
        modulo.async_playwright = lambda: PlaywrightFalso(navegador)
        with patch.dict("sys.modules", {"playwright.async_api": modulo}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(recorrer.recorrer(self.plan), 0)
        self.assertEqual(navegador.contextos, 5)
        navegador = NavegadorFalso(PaginaFalsa(fallar=True))
        modulo.async_playwright = lambda: PlaywrightFalso(navegador)
        with patch.dict("sys.modules", {"playwright.async_api": modulo}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(recorrer.recorrer(self.plan), 1)
        self.assertEqual(navegador.contextos, 1)


if __name__ == "__main__":
    unittest.main()
