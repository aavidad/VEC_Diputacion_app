"""La copia local enciende la cancelación del expediente sin pisar el kit privado."""
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest

spec = importlib.util.spec_from_file_location(
    'cancelacion', Path(__file__).with_name('copia_local_cancelacion.py'))
cancelacion = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cancelacion)
REPO = Path(__file__).resolve().parents[2]


class CancelacionCopiaTest(unittest.TestCase):
    def test_enciende_el_selector_con_los_catalogos_versionados(self):
        entorno = cancelacion.entorno_cancelacion({'VEC_CT_DATABASE_URL': 'x'}, REPO)
        self.assertEqual(entorno['VEC_CT_CANCELACION_ENABLED'], 'true')
        self.assertEqual(entorno['VEC_CT_DATABASE_URL'], 'x')
        for variable, relativa in cancelacion.CATALOGOS.items():
            self.assertEqual(entorno[variable], str(REPO / relativa))
            self.assertTrue(Path(entorno[variable]).is_file())
        reglas = (REPO / cancelacion.CATALOGOS['VEC_CT_REGLAS_SOURCE_PATH']).read_text()
        self.assertIn('c20.cancelacion_expediente', reglas)

    def test_conserva_un_catalogo_que_el_kit_ya_fijo(self):
        entorno = cancelacion.entorno_cancelacion({'VEC_CT_REGLAS_SOURCE_PATH': '/privado/reglas.json'}, REPO)
        self.assertEqual(entorno['VEC_CT_REGLAS_SOURCE_PATH'], '/privado/reglas.json')
        apagada = cancelacion.entorno_cancelacion({'VEC_CT_CANCELACION_ENABLED': 'false'}, REPO)
        self.assertEqual(apagada['VEC_CT_CANCELACION_ENABLED'], 'false')

    def test_sin_catalogo_no_enciende(self):
        with tempfile.TemporaryDirectory() as vacia, self.assertRaises(ValueError):
            cancelacion.entorno_cancelacion({}, vacia)

    def test_encender_guarda_el_entorno_y_su_huella(self):
        with tempfile.TemporaryDirectory() as carpeta:
            raiz = Path(carpeta)
            estado = {'fuente': str(REPO), 'entorno_modulos_sha256': 'antigua'}
            guardados = {}
            copia = SimpleNamespace(raiz=raiz, estado=raiz / 'estado.json', leer=lambda: dict(estado),
                                    entorno_modulos=lambda: {'VEC_CT_DATABASE_URL': 'x'})
            cancelacion.encender(copia, lambda ruta, valor: guardados.__setitem__(ruta, valor),
                                 lambda ruta: 'nueva')
            self.assertEqual(guardados[raiz / 'entorno_modulos.json']['VEC_CT_CANCELACION_ENABLED'], 'true')
            self.assertEqual(guardados[raiz / 'estado.json']['entorno_modulos_sha256'], 'nueva')
            json.dumps(guardados[raiz / 'entorno_modulos.json'])


if __name__ == '__main__':
    unittest.main()
