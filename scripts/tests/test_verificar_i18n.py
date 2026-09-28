import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from verificar_i18n import ATRIBUTO_PLANTILLA, TEXTO_PLANTILLA, TextosHTML, claves  # noqa: E402


class I18nGuardTest(unittest.TestCase):
    def test_catalogo_anidado(self):
        self.assertEqual(claves({"area": {"titulo": "Título {total}"}}), {"area.titulo": "Título {total}"})
        with self.assertRaises(ValueError):
            claves({"vacio": ""})

    def test_html_exige_claves_en_texto_y_atributos(self):
        ruta = Path(__file__).resolve().parents[2] / "web/static/ejemplo.html"
        analizador = TextosHTML(ruta)
        analizador.feed('<h1>Título</h1><button aria-label="Abrir">Acción</button>'
                        '<p data-i18n="descripcion">Texto de respaldo</p>')
        self.assertEqual(len(analizador.fallos), 3)
        self.assertTrue(any("atributo aria-label" in fallo for fallo in analizador.fallos))

    def test_plantilla_js_distingue_texto_de_codigo(self):
        self.assertEqual(TEXTO_PLANTILLA.findall('`<p>Sin registros</p>`'), ["Sin registros"])
        self.assertEqual(TEXTO_PLANTILLA.findall('v => String(v).replaceAll("<", "&lt;")'), [])
        self.assertEqual(ATRIBUTO_PLANTILLA.findall('`<button aria-label="Abrir">`'), [('"', 'Abrir')])


if __name__ == "__main__":
    unittest.main()
