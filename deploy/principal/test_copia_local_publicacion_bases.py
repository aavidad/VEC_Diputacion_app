"""La semilla de publicación de bases solo declara lo abierto y no repite."""
import importlib.util
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import Mock

spec = importlib.util.spec_from_file_location(
    'publicacion', Path(__file__).with_name('copia_local_publicacion_bases.py'))
publicacion = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publicacion)


class PublicacionBasesTest(unittest.TestCase):
    def test_aplica_la_semilla_versionada_con_su_registro(self):
        copia = SimpleNamespace(sql=Mock())
        publicacion.declarar(copia)
        copia.sql.assert_called_once_with(publicacion.SEMILLA, 'publicacion-bases.log')
        self.assertTrue(publicacion.SEMILLA.is_file())

    def test_la_semilla_es_del_propietario_idempotente_y_solo_con_bc14(self):
        sql = publicacion.SEMILLA.read_text()
        self.assertIn('\\if :bc14', sql)
        self.assertIn('SET LOCAL ROLE vec_bolsa_convocatorias_propietario;', sql)
        # Solo versiones vigentes, con un plazo abierto y sin declaración previa.
        self.assertIn("WHERE v.estado='publicada'", sql)
        self.assertIn('HAVING count(*)=1', sql)
        self.assertIn('NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases d', sql)
        # Dato ficticio reconocible: tablón sin boletín y referencias COPIA.
        self.assertIn("'medio','tablon_edictos_electronico','boletin_numero',NULL", sql)
        self.assertIn("'anuncio_ref','COPIA-'", sql)
        self.assertIn("'clave_idempotencia','copia-publicacion-'", sql)
        self.assertNotIn('UPDATE ', sql)
        self.assertNotIn('DELETE ', sql)


if __name__ == '__main__':
    unittest.main()
