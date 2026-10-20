import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
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

    def test_espera_al_servidor_definitivo_antes_de_ejecutar_sql(self):
        from deploy.principal import copia_local_roles

        with tempfile.TemporaryDirectory(dir='/dev/shm') as temporal:
            eventos = []

            def esquema(destino):
                ruta = destino / 'esquema.sql'
                ruta.write_bytes(b'CREATE SCHEMA ejemplo;')
                return ruta

            def disponibilidad(argumentos, **_opciones):
                # El servidor temporal acepta el socket Unix, pero todavía
                # no atiende TCP. Un sondeo Unix adelantaría la primera SQL.
                if '-h' not in argumentos:
                    eventos.append('unix_temporal')
                    return mock.Mock(returncode=0)
                if 'tcp_pendiente' not in eventos:
                    eventos.append('tcp_pendiente')
                    return mock.Mock(returncode=1)
                eventos.append('tcp_definitivo')
                return mock.Mock(returncode=0)

            def ejecutar_sql(*_argumentos):
                eventos.append('sql')
                return mock.Mock(stdout=b'')

            with (mock.patch.object(ensayar, 'comprobar_destino', return_value=Path(temporal)),
                  mock.patch.object(ensayar, 'preparar_esquema', side_effect=esquema),
                  mock.patch.object(copia_local_roles, 'extract', return_value=[]),
                  mock.patch.object(ensayar, 'comandos_docker'),
                  mock.patch.object(ensayar, 'contenedor_propio', return_value=True),
                  mock.patch.object(ensayar, 'limpiar_copia'),
                  mock.patch.object(ensayar.subprocess, 'run', side_effect=disponibilidad),
                  mock.patch.object(ensayar.time, 'sleep'),
                  mock.patch.object(ensayar, 'sql', side_effect=ejecutar_sql),
                  mock.patch.object(ensayar, 'verificar_tablas_vacias')):
                ensayar.ejecutar(solo_base=True)
            self.assertEqual(eventos[:3], ['tcp_pendiente', 'tcp_definitivo', 'sql'])

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

    def test_pruebas_usan_bytes_versionados_y_lista_cerrada(self):
        self.assertEqual(len(ensayar.PRUEBAS_SQL), 18)
        self.assertIn(ensayar.CA48_PRUEBA, ensayar.PRUEBAS_SQL)
        with mock.patch.object(ensayar.subprocess, 'check_output', return_value=b'SQL') as git:
            with mock.patch.object(Path, 'read_bytes', return_value=b'SQL'):
                casos = ensayar.pruebas_versionadas()
        self.assertEqual([ruta for ruta, _ in casos], list(ensayar.PRUEBAS_SQL))
        self.assertEqual([contenido for _, contenido in casos], [b'SQL'] * 18)
        self.assertEqual([llamada.args[0][-1] for llamada in git.call_args_list],
                         [f'HEAD:{ruta}' for ruta in ensayar.PRUEBAS_SQL])
        with mock.patch.object(ensayar.subprocess, 'check_output', return_value=b'alterado'):
            with self.assertRaisesRegex(ValueError, 'difiere de la cabeza versionada'):
                ensayar.pruebas_versionadas()

    def test_reconciliacion_hz37_exige_d15_exacto_y_huella(self):
        sql = b'MIGRACION D15'
        fila = f'{ensayar.D15_SQL} {hashlib.sha256(sql).hexdigest()}\n'.encode()
        with mock.patch.object(ensayar, 'contenido_versionado',
                               side_effect=[fila, sql]) as leer:
            self.assertEqual(ensayar.base_hz37_versionada(), (ensayar.D15_SQL, sql))
        self.assertEqual([llamada.args[0] for llamada in leer.call_args_list], [
            'deploy/postgresql/fase1/lista_sql_base_hz37.txt', ensayar.D15_SQL])
        otra = b'deploy/postgresql/otro.sql ' + hashlib.sha256(sql).hexdigest().encode()
        with mock.patch.object(ensayar, 'contenido_versionado', return_value=otra):
            with self.assertRaisesRegex(ValueError, 'no identifica D15'):
                ensayar.base_hz37_versionada()
        with mock.patch.object(ensayar, 'contenido_versionado',
                               side_effect=[fila, b'distinto']):
            with self.assertRaisesRegex(ValueError, 'huella de D15'):
                ensayar.base_hz37_versionada()

    def test_pruebas_siguen_instalacion_y_fallo_sql_sale_con_estado_uno(self):
        from deploy.principal import copia_local, copia_local_roles

        with tempfile.TemporaryDirectory(dir='/dev/shm') as temporal:
            eventos = []

            def esquema(destino):
                ruta = destino / 'esquema.sql'
                ruta.write_bytes(b'ESQUEMA')
                return ruta

            def ejecutar_sql(_nombre, contenido, etapa):
                eventos.append((contenido, etapa))
                if contenido == b'PRUEBA2':
                    raise ValueError('Falló la prueba funcional')
                return mock.Mock(stdout=b'')

            with (mock.patch.object(ensayar, 'comprobar_destino', return_value=Path(temporal)),
                  mock.patch.object(ensayar, 'preparar_esquema', side_effect=esquema),
                  mock.patch.object(ensayar, 'preparacion_versionada', return_value=b'PREPARACION'),
                  mock.patch.object(ensayar, 'base_hz37_versionada',
                                    return_value=(ensayar.D15_SQL, b'D15')),
                  mock.patch.object(ensayar, 'pruebas_versionadas', return_value=[
                      ('primera.sql', b'PRUEBA1'), ('segunda.sql', b'PRUEBA2')]),
                  mock.patch.object(copia_local, 'migraciones', return_value=[
                      (ensayar.RAIZ / 'migracion.sql', 'huella', b'MIGRACION')]),
                  mock.patch.object(copia_local_roles, 'extract', return_value=[]),
                  mock.patch.object(ensayar, 'comandos_docker'),
                  mock.patch.object(ensayar, 'contenedor_propio', return_value=True),
                  mock.patch.object(ensayar.subprocess, 'run',
                                    return_value=mock.Mock(returncode=0)),
                  mock.patch.object(ensayar, 'sql', side_effect=ejecutar_sql),
                  mock.patch.object(ensayar, 'verificar_tablas_vacias'),
                  mock.patch('sys.argv', ['ensayar.py', '--pruebas'])):
                self.assertEqual(ensayar.main(), 1)
            self.assertEqual([etapa for _, etapa in eventos], [
                'la creación de roles sin LOGIN', 'la restauración del esquema HZ37',
                'la preparación versionada de la fase 1',
                f'la reconciliación HZ37 {ensayar.D15_SQL}',
                'la migración migracion.sql',
                'la prueba primera.sql', 'la prueba segunda.sql'])
            self.assertEqual([contenido for contenido, _ in eventos][1:], [
                b'ESQUEMA', b'PREPARACION', b'D15', b'MIGRACION',
                b'PRUEBA1', b'PRUEBA2'])
            self.assertIn('la prueba segunda.sql', eventos[-1][1])
            self.assertEqual(list(Path(temporal).iterdir()), [])

    def test_reinicio_conserva_datos_y_no_reinstala_migraciones(self):
        from deploy.principal import copia_local, copia_local_roles
        from deploy.postgresql.fase1 import concurrencia_contexto, concurrencia_lectura_bolsa

        with tempfile.TemporaryDirectory(dir='/dev/shm') as temporal:
            eventos = []
            ruta = Path(temporal) / 'esquema.sql'
            ruta.write_bytes(b'ESQUEMA')

            def ejecutar_sql(_nombre, contenido, etapa):
                eventos.append((contenido, etapa))
                return mock.Mock(stdout=b'')

            def docker(*argumentos, **_opciones):
                if argumentos[0] == 'restart':
                    eventos.append((b'REINICIO', 'reinicio'))
                return mock.Mock(stdout=b'')

            def concurrencia(_nombre, _etiqueta, contenido):
                self.assertEqual(contenido, b'PRUEBA')
                eventos.append((b'CARRERA', 'concurrencia'))

            def cruce(_nombre, _etiqueta, contenido):
                self.assertEqual(contenido, b'CA48')
                eventos.append((b'CRUCE', 'concurrencia de lectura de Bolsa'))

            with (mock.patch.object(ensayar, 'comprobar_destino', return_value=Path(temporal)),
                  mock.patch.object(ensayar, 'preparar_esquema', return_value=ruta),
                  mock.patch.object(ensayar, 'preparacion_versionada', return_value=b'PREPARACION'),
                  mock.patch.object(ensayar, 'base_hz37_versionada',
                                    return_value=(ensayar.D15_SQL, b'D15')),
                  mock.patch.object(ensayar, 'pruebas_versionadas',
                                    return_value=[(ensayar.CA48_PRUEBA, b'CA48'),
                                                  (ensayar.PRUEBAS_SQL[-1], b'PRUEBA')]),
                  mock.patch.object(copia_local, 'migraciones', return_value=[
                      (ensayar.RAIZ / 'migracion.sql', 'huella', b'MIGRACION')]),
                  mock.patch.object(copia_local_roles, 'extract', return_value=[]),
                  mock.patch.object(ensayar, 'comandos_docker', side_effect=docker),
                  mock.patch.object(ensayar, 'contenedor_propio', return_value=True),
                  mock.patch.object(ensayar, 'esperar_postgresql') as esperar,
                  mock.patch.object(ensayar, 'sql', side_effect=ejecutar_sql),
                  mock.patch.object(ensayar, 'verificar_tablas_vacias'),
                  mock.patch.object(concurrencia_contexto, 'ensayar', side_effect=concurrencia) as carrera,
                  mock.patch.object(concurrencia_lectura_bolsa, 'ensayar', side_effect=cruce) as cruzada,
                  mock.patch.object(ensayar, 'limpiar_copia')):
                ensayar.ejecutar(pruebas=True)
            self.assertEqual(esperar.call_count, 2)
            self.assertEqual([contenido for contenido, _ in eventos][1:], [
                b'ESQUEMA', b'PREPARACION', b'D15', b'MIGRACION',
                b'CA48', b'PRUEBA', b'REINICIO', b'CA48', b'PRUEBA', b'CARRERA', b'CRUCE'])
            self.assertEqual(eventos[-3][1], 'la prueba tras reiniciar ' + ensayar.PRUEBAS_SQL[-1])
            self.assertEqual(carrera.call_count, 1)
            self.assertEqual(cruzada.call_count, 1)

    def test_pruebas_y_solo_base_son_incompatibles(self):
        with mock.patch.object(ensayar, 'comprobar_destino', return_value=Path('/dev/shm')):
            with self.assertRaisesRegex(ValueError, '--solo-base'):
                ensayar.ejecutar(solo_base=True, pruebas=True)

    def test_concurrencia_libera_ambas_sesiones_si_no_observa_bloqueo(self):
        from deploy.postgresql.fase1 import concurrencia_contexto as carrera

        a, b = mock.Mock(), mock.Mock()
        a.esperar.side_effect = [
            carrera.PREIMAGEN.search(b'PREIMAGEN 101 1 ' + b'a' * 64 + b'\n'),
            re.search(rb'A_PUBLICADO\n', b'A_PUBLICADO\n'),
        ]
        b.esperar.return_value = carrera.PREIMAGEN.search(
            b'PREIMAGEN 102 1 ' + b'a' * 64 + b'\n')
        with (mock.patch.object(carrera, 'comprobar_contenedor'),
              mock.patch.object(carrera, 'psql', return_value=mock.Mock(stdout=b'CTX15-PRUEBA-OK')),
              mock.patch.object(carrera, 'Sesion', side_effect=[a, b]),
              mock.patch.object(carrera, 'esperar_bloqueo', side_effect=ValueError('sin bloqueo'))):
            with self.assertRaisesRegex(ValueError, 'sin bloqueo'):
                carrera.ensayar('vec-fase1-012345abcdef', '/dev/shm/propio', b'CTX15-PRUEBA-OK')
        a.terminar.assert_called_once_with()
        b.terminar.assert_called_once_with()

    def test_concurrencia_libera_primera_sesion_si_segunda_no_arranca(self):
        from deploy.postgresql.fase1 import concurrencia_contexto as carrera

        a = mock.Mock()
        with (mock.patch.object(carrera, 'comprobar_contenedor'),
              mock.patch.object(carrera, 'psql', return_value=mock.Mock(stdout=b'CTX15-PRUEBA-OK')),
              mock.patch.object(carrera, 'Sesion', side_effect=[a, OSError('sin descriptor')])):
            with self.assertRaisesRegex(OSError, 'sin descriptor'):
                carrera.ensayar('vec-fase1-012345abcdef', '/dev/shm/propio', b'CTX15-PRUEBA-OK')
        a.terminar.assert_called_once_with()

    def test_concurrencia_rechaza_sesion_sin_bloqueo_real(self):
        from deploy.postgresql.fase1 import concurrencia_contexto as carrera

        sesion = mock.Mock()
        sesion.proceso.poll.return_value = None
        with (mock.patch.object(carrera, 'bloqueado_por', return_value=False),
              mock.patch.object(carrera.time, 'monotonic', side_effect=[0, 0, 0.6]),
              mock.patch.object(carrera.time, 'sleep')):
            with self.assertRaisesRegex(ValueError, 'bloqueo real'):
                carrera.esperar_bloqueo('vec-fase1-012345abcdef', 101, 102, sesion, plazo=0.5)

    def test_concurrencia_lee_hito_de_stderr_sin_esperar_cierre(self):
        from deploy.postgresql.fase1 import concurrencia_contexto as carrera

        salida_r, salida_w = os.pipe()
        error_r, error_w = os.pipe()
        try:
            os.write(error_w, b'PREIMAGEN 101 1 ' + b'a' * 64 + b'\n')
            sesion = carrera.Sesion.__new__(carrera.Sesion)
            sesion.salida, sesion.errores = bytearray(), bytearray()
            sesion.proceso = mock.Mock(stdout=os.fdopen(salida_r, 'rb', buffering=0),
                                      stderr=os.fdopen(error_r, 'rb', buffering=0))
            sesion.proceso.poll.return_value = None
            hito = sesion.esperar(carrera.PREIMAGEN, plazo=0.5)
            self.assertEqual(hito.group(1), b'101')
        finally:
            os.close(salida_w)
            os.close(error_w)
            for tubo in (sesion.proceso.stdout, sesion.proceso.stderr):
                tubo.close()


if __name__ == '__main__':
    unittest.main()
