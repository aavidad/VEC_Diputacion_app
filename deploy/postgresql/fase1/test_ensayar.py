import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

RUTA = Path(__file__).with_name('ensayar.py')
spec = importlib.util.spec_from_file_location('ensayar_fase1', RUTA)
ensayar = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ensayar)


class EnsayoFase1Test(unittest.TestCase):
    def test_ci_acepta_memoria_de_runner_y_local_conserva_34_gib(self):
        with tempfile.TemporaryDirectory(dir='/dev/shm') as temporal:
            espacio = mock.Mock(free=3 * 1024**3)
            with (mock.patch.object(ensayar, 'memoria_disponible', return_value=4 * 1024**3),
                  mock.patch.object(ensayar.shutil, 'disk_usage', return_value=espacio)):
                with mock.patch.dict(os.environ, {'GITHUB_ACTIONS': 'true'}):
                    self.assertEqual(ensayar.comprobar_destino(temporal), Path(temporal))
                with mock.patch.dict(os.environ, {'GITHUB_ACTIONS': ''}):
                    with self.assertRaisesRegex(ValueError, '34 GiB'):
                        ensayar.comprobar_destino(temporal)
            with (mock.patch.dict(os.environ, {'GITHUB_ACTIONS': 'true'}),
                  mock.patch.object(ensayar, 'memoria_disponible', return_value=2 * 1024**3),
                  mock.patch.object(ensayar.shutil, 'disk_usage', return_value=espacio)):
                with self.assertRaisesRegex(ValueError, '3 GiB'):
                    ensayar.comprobar_destino(temporal)

    def test_esquema_corrupto_se_rechaza_antes_de_restaurar(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            sql = b'CREATE SCHEMA ejemplo;\n'
            comprimido = gzip.compress(sql)
            (raiz / 'esquema_hz37.sql.gz').write_bytes(comprimido + b'alterado')
            (raiz / 'esquema_hz37.json').write_text(json.dumps({
                'referencia': 'HZ37', 'postgresql': '18', 'solo_esquema': True,
                'sin_datos': True, 'bytes_sql': len(sql),
                'sha256_sql': hashlib.sha256(sql).hexdigest(),
                'sha256_gzip': hashlib.sha256(comprimido).hexdigest(),
            }))
            with mock.patch.object(ensayar, 'FASE', raiz):
                with self.assertRaisesRegex(ValueError, 'huella.*comprimido'):
                    ensayar.preparar_esquema(raiz)
            self.assertFalse((raiz / 'esquema.sql').exists())

    def test_fallo_de_docker_borra_solo_su_carpeta(self):
        from deploy.principal import copia_local_roles

        with tempfile.TemporaryDirectory() as temporal:
            base = Path(temporal)
            ajeno = base / 'ajeno'
            ajeno.write_text('conservar')

            def esquema(destino):
                ruta = destino / 'esquema.sql'
                ruta.write_text('CREATE SCHEMA ejemplo;')
                return ruta

            with (mock.patch.object(ensayar, 'comprobar_destino', return_value=base),
                  mock.patch.object(ensayar, 'preparar_esquema', side_effect=esquema),
                  mock.patch.object(copia_local_roles, 'extract', return_value=[]),
                  mock.patch.object(ensayar, 'comandos_docker',
                                    side_effect=subprocess.CalledProcessError(1, ['docker'])) as docker,
                  mock.patch.object(ensayar, 'contenedor_propio', return_value=False)):
                with self.assertRaises(subprocess.CalledProcessError):
                    ensayar.ejecutar(solo_base=True)
            self.assertEqual([ruta.name for ruta in base.iterdir()], ['ajeno'])
            self.assertEqual(ajeno.read_text(), 'conservar')
            argumentos = docker.call_args.args
            self.assertIn('--network=none', argumentos)
            self.assertIn('--restart=no', argumentos)
            self.assertIn('PGDATA=/datos', argumentos)

    def test_pgdata_ajeno_al_uid_se_vacia_con_montaje_limitado(self):
        with tempfile.TemporaryDirectory(dir='/dev/shm') as temporal:
            base = Path(temporal)
            propio = base / 'vec-fase1-propio'
            datos = propio / 'data'
            datos.mkdir(parents=True)
            (datos / 'archivo').write_text('dato ficticio')
            ajeno = base / 'ajeno'
            ajeno.write_text('conservar')
            real_rmtree = shutil.rmtree
            primero = True

            def borrar(ruta):
                nonlocal primero
                if Path(ruta) == datos and primero:
                    primero = False
                    raise PermissionError('PGDATA uid 999')
                return real_rmtree(ruta)

            def docker_falso(*argumentos):
                if argumentos[0] == 'run':
                    (datos / 'archivo').unlink()

            with (mock.patch.object(ensayar, 'contenedor_propio', return_value=True),
                  mock.patch.object(ensayar, 'comandos_docker', side_effect=docker_falso) as docker,
                  mock.patch.object(ensayar.shutil, 'rmtree', side_effect=borrar) as rmtree):
                ensayar.limpiar_copia('nombre', str(propio), propio)
            self.assertFalse(propio.exists())
            self.assertEqual(ajeno.read_text(), 'conservar')
            self.assertEqual(docker.call_args_list[0].args[:3], ('rm', '-f', 'nombre'))
            argumentos = docker.call_args_list[1].args
            self.assertIn('--network=none', argumentos)
            self.assertIn('--memory=64m', argumentos)
            self.assertIn(f'type=bind,src={datos},dst=/datos', argumentos)
            self.assertNotIn(str(base), argumentos)
            self.assertEqual([call.args[0] for call in rmtree.call_args_list], [datos, propio])

    def test_base_sin_tablas_no_pasa_la_aceptacion(self):
        with mock.patch.object(ensayar, 'sql', side_effect=[mock.Mock(), mock.Mock(stdout=b'0\n')]):
            with self.assertRaisesRegex(ValueError, 'No se encontraron tablas'):
                ensayar.verificar_tablas_vacias('contenedor')


if __name__ == '__main__':
    unittest.main()
