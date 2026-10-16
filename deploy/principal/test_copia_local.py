"""Rechazos previos a restaurar o borrar una instancia."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('copia', Path(__file__).with_name('copia_local.py'))
copia = importlib.util.module_from_spec(spec)
spec.loader.exec_module(copia)


class CopiaLocalTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.repo = self.raiz / 'fuente'
        self.sql = self.repo / 'deploy/postgresql/modulo/000001.up.sql'
        self.sql.parent.mkdir(parents=True)
        self.sql.write_text('BEGIN; COMMIT;')
        self.lista = self.raiz / 'lista.txt'
        self.lista.write_text('deploy/postgresql/modulo/000001.up.sql\n')

    def test_rechaza_down_y_fuga_fuera_del_repositorio(self):
        for ruta in ['000001.down.sql', '../../../exterior.up.sql']:
            with self.subTest(ruta=ruta):
                fuente = (self.sql.parent / ruta).resolve()
                fuente.parent.mkdir(parents=True, exist_ok=True)
                fuente.write_text('SELECT 1;')
                self.lista.write_text(str(fuente))
                with patch.object(copia, 'REPO', self.repo), self.assertRaises(ValueError):
                    copia.migraciones([self.lista])

    def test_no_omite_duplicado_ni_huella_alterada(self):
        with patch.object(copia, 'REPO', self.repo):
            with self.assertRaises(ValueError):
                copia.migraciones([self.lista, self.lista])
            self.lista.write_text('deploy/postgresql/modulo/000001.up.sql ' + '0' * 64)
            with self.assertRaises(ValueError):
                copia.migraciones([self.lista])

    def test_valida_toda_la_lista_antes_de_aplicar(self):
        with patch.object(copia, 'REPO', self.repo):
            rutas = copia.migraciones([self.lista])
            self.assertEqual(rutas[0][0], self.sql)
            self.assertEqual(len(rutas[0][1]), 64)

    def test_propiedad_impide_usar_estado_ajeno(self):
        with patch.object(copia, 'BASE', self.raiz):
            instancia = copia.Copia('l', 'focal')
            instancia.raiz.mkdir()
            instancia.estado.write_text(json.dumps(dict(nombre='otro', raiz=str(instancia.raiz))))
            with self.assertRaises(ValueError):
                instancia.leer()

    def test_destino_enlazado_no_se_abre(self):
        (self.raiz / 'l-focal').symlink_to(self.repo, target_is_directory=True)
        with patch.object(copia, 'BASE', self.raiz), self.assertRaises(ValueError):
            copia.Copia('l', 'focal')


if __name__ == '__main__':
    unittest.main()
