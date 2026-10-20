"""La procedencia de las herramientas se coteja antes y después del efecto."""
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch
from recorrido_copia.comun import Contexto


class FuenteRecorridoTest(unittest.TestCase):
    def contexto(self, carpeta):
        c = Contexto.__new__(Contexto)
        c.revision = 'a' * 40
        c.carpeta = carpeta
        return c

    def test_fuente_alterada_impide_ejecutar_herramienta(self):
        with tempfile.TemporaryDirectory() as d:
            c = self.contexto(Path(d))
            with patch('copia_local.verificar_fuente', side_effect=ValueError('fuente alterada')), \
                    patch('recorrido_copia.comun.subprocess.run') as ejecutar:
                with self.assertRaisesRegex(ValueError, 'fuente alterada'):
                    c.ejecutar(['/usr/bin/true'], 'salida.log')
                ejecutar.assert_not_called()

    def test_cambio_durante_herramienta_conserva_salida_y_no_confirma(self):
        with tempfile.TemporaryDirectory() as d:
            c = self.contexto(Path(d))
            salida = SimpleNamespace(returncode=0, stdout=b'recibo original', stderr=b'')
            with patch('copia_local.verificar_fuente', side_effect=[None, ValueError('fuente alterada')]), \
                    patch('recorrido_copia.comun.subprocess.run', return_value=salida):
                with self.assertRaisesRegex(ValueError, 'fuente alterada'):
                    c.ejecutar(['/usr/bin/true'], 'salida.log')
                self.assertEqual((Path(d) / 'salida.log').read_bytes(), b'recibo original')

    def test_fuente_alterada_impide_sql(self):
        c = self.contexto(Path('/tmp'))
        c.copia = Mock()
        with patch('copia_local.verificar_fuente', side_effect=ValueError('fuente alterada')):
            with self.assertRaisesRegex(ValueError, 'fuente alterada'):
                c.sql(b'SELECT 1;', 'sql.log')
            c.copia.sql_bytes.assert_not_called()
