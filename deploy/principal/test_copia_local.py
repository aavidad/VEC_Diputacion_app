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
        self.historial = patch.object(copia.subprocess, 'check_output',
                side_effect=lambda args, **kw: 'deploy/postgresql/modulo/000001.up.sql\n'
                if 'diff' in args else b'BEGIN; COMMIT;').start()
        self.addCleanup(patch.stopall)

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

    def test_rechaza_sql_del_corte_de_referencia(self):
        self.historial.side_effect = None
        self.historial.return_value = ''
        with patch.object(copia, 'REPO', self.repo), self.assertRaises(ValueError):
            copia.migraciones([self.lista])

    def test_rechaza_sql_modificada_sin_commit(self):
        self.sql.write_text('SELECT 2;')
        with patch.object(copia, 'REPO', self.repo), self.assertRaises(ValueError):
            copia.migraciones([self.lista])

    def test_aplica_los_bytes_validados_aunque_cambie_la_fuente(self):
        with patch.object(copia, 'REPO', self.repo):
            plan = copia.migraciones([self.lista])
        self.sql.write_text('SELECT 2;')
        self.assertEqual(plan[0][2], b'BEGIN; COMMIT;')
        self.assertEqual(copia.hashlib.sha256(plan[0][2]).hexdigest(), plan[0][1])

    def test_interrupcion_de_estado_conserva_el_anterior(self):
        estado = self.raiz / 'estado.json'
        copia.guardar(estado, {'preparada': False})
        with patch.object(copia.os, 'replace', side_effect=OSError), self.assertRaises(OSError):
            copia.guardar(estado, {'preparada': True})
        self.assertFalse(json.loads(estado.read_text())['preparada'])
        self.assertEqual(list(self.raiz.glob('.estado-*')), [])

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

    def test_otro_arbol_no_puede_usar_la_instancia(self):
        with patch.object(copia, 'BASE', self.raiz):
            instancia = copia.Copia('l', 'focal')
            instancia.raiz.mkdir()
            instancia.estado.write_text(json.dumps(dict(nombre=instancia.nombre,
                raiz=str(instancia.raiz), fuente='/otro/arbol')))
            with self.assertRaises(ValueError):
                instancia.leer()

    def test_exportacion_fallida_no_publica_archivo_parcial(self):
        with patch.object(copia, 'BASE', self.raiz):
            instancia = copia.Copia('l', 'focal')
            destino = self.raiz / 'clon.tgz'
            with patch.object(instancia, 'leer', return_value={'origen': '/fuente-privada'}), \
                 patch.object(instancia, 'validar_pgdata'), patch.object(instancia, 'parar'), \
                 patch.object(instancia, 'docker', side_effect=OSError), self.assertRaises(OSError):
                instancia.exportar(str(destino))
            self.assertFalse(destino.exists())
            self.assertEqual(list(self.raiz.glob('.vec-clon-*')), [])


if __name__ == '__main__':
    unittest.main()
