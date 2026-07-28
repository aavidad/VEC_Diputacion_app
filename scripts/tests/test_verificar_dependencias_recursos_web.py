import importlib.util
from pathlib import Path
import tempfile
import unittest


RUTA_VERIFICADOR = (
    Path(__file__).resolve().parents[1] / "verificar_dependencias_recursos_web.py"
)
ESPECIFICACION = importlib.util.spec_from_file_location(
    "verificar_dependencias_recursos_web",
    RUTA_VERIFICADOR,
)
VERIFICADOR = importlib.util.module_from_spec(ESPECIFICACION)
assert ESPECIFICACION.loader is not None
ESPECIFICACION.loader.exec_module(VERIFICADOR)


class VerificadorDependenciasRecursosWebTest(unittest.TestCase):
    def preparar(self, origen: str, contenido: str, dependencia: str | None):
        temporal = tempfile.TemporaryDirectory()
        raiz = Path(temporal.name)
        archivo = raiz / "web" / origen
        archivo.parent.mkdir(parents=True)
        archivo.write_text(contenido, encoding="utf-8")
        manifiesto = raiz / "web" / "prueba.manifest"
        manifiesto.write_text(origen + "\n", encoding="utf-8")
        if dependencia is not None:
            destino = raiz / "web" / dependencia
            destino.parent.mkdir(parents=True, exist_ok=True)
            destino.write_text("recurso", encoding="utf-8")
        return temporal, raiz, manifiesto

    def assert_rechazada(self, origen: str, contenido: str, dependencia: str | None):
        temporal, raiz, manifiesto = self.preparar(
            origen,
            contenido,
            dependencia,
        )
        self.addCleanup(temporal.cleanup)
        errores = VERIFICADOR.comprobar(raiz, manifiesto, None)
        self.assertTrue(errores, "la dependencia local no inventariada fue aceptada")

    def test_rechaza_importacion_estatica(self):
        self.assert_rechazada(
            "static/app.js",
            'import valor from "./dependencia.js";',
            "static/dependencia.js",
        )
        self.assert_rechazada(
            "static/app.js",
            'export { valor } from "./dependencia.js";',
            "static/dependencia.js",
        )

    def test_rechaza_importacion_dinamica_con_backticks_literales(self):
        self.assert_rechazada(
            "static/app.js",
            "import(`./dependencia.js`);",
            "static/dependencia.js",
        )

    def test_rechaza_importacion_css(self):
        self.assert_rechazada(
            "static/app.css",
            '@import "./dependencia.css";',
            "static/dependencia.css",
        )

    def test_rechaza_uso_svg(self):
        self.assert_rechazada(
            "static/icono.svg",
            '<svg><use href="./simbolos.svg#ok"></use></svg>',
            "static/simbolos.svg",
        )

    def test_rechaza_literal_local_ausente(self):
        self.assert_rechazada(
            "static/app.js",
            'import "./ausente.js";',
            None,
        )

    def test_ignora_plantilla_realmente_dinamica(self):
        temporal, raiz, manifiesto = self.preparar(
            "static/app.js",
            "import(`./${nombre}.js`);",
            None,
        )
        self.addCleanup(temporal.cleanup)
        self.assertEqual(VERIFICADOR.comprobar(raiz, manifiesto, None), [])


if __name__ == "__main__":
    unittest.main()
