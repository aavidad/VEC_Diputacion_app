import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from verificar_i18n import ATRIBUTO_PLANTILLA, COBERTURA, TEXTO_PLANTILLA, WEB, TextosHTML, claves, cubierto  # noqa: E402


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

    def test_noscript_y_copias_ct_no_son_literales_pendientes(self):
        ruta = Path(__file__).resolve().parents[2] / "web/static/ejemplo.html"
        analizador = TextosHTML(ruta)
        analizador.feed('<noscript><p>Active JavaScript</p></noscript>'
                        '<title data-ct-copia="ct_txt_titulo">Contratación</title>'
                        '<img alt="Logotipo" data-ct-copia-alt="ct_txt_logo"><input placeholder="***1234**">')
        self.assertEqual(analizador.fallos, [])

    def test_cobertura_explicita_y_con_exclusiones(self):
        self.assertIn("bolsa/", COBERTURA["literales"])
        self.assertTrue(cubierto(WEB / "bolsa/index.html", COBERTURA["literales"], COBERTURA["literales_excluidos"]))
        self.assertFalse(cubierto(WEB / "bolsa/documentos/bases.html", COBERTURA["literales"], COBERTURA["literales_excluidos"]))
        self.assertFalse(cubierto(WEB / "portal-empleado/modulos/cronos/index.html", COBERTURA["literales"]))


if __name__ == "__main__":
    unittest.main()
