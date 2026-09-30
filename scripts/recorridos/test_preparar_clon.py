from pathlib import Path
import hashlib
import json
import subprocess
import tempfile
import unittest


class PublicacionReadyTests(unittest.TestCase):
    def comprobar_publicacion(self, resultado):
        source = Path(__file__).with_name('preparar_clon.sh').read_text()
        function = source[source.index('publicar_ready() {'):source.index('\nfinalizar_arranque() {')]
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / 'publisher-called'
            script = '''set -e
runtime() { return "$VEC_TEST_VERIFY_EXIT"; }
python3() { touch "$VEC_TEST_PUBLISHER_MARKER"; }
''' + function + '''
if ! publicar_ready; then exit 17; fi
'''
            environment = {'PATH': '/usr/bin:/bin', 'VEC_TEST_VERIFY_EXIT': str(resultado),
                           'VEC_TEST_PUBLISHER_MARKER': str(marker)}
            process = subprocess.run(['bash', '-c', script], env=environment, capture_output=True, timeout=5)
            return process.returncode, marker.exists()

    def test_verificacion_fallida_impide_publicar_incluso_dentro_de_if(self):
        self.assertEqual(self.comprobar_publicacion(7), (17, False))

    def test_verificacion_correcta_permite_la_publicacion(self):
        self.assertEqual(self.comprobar_publicacion(0), (0, True))

    def publicar_fixture(self, *, binario_alterado=False, fuente_alterada=False):
        source = Path(__file__).with_name('preparar_clon.sh').read_text()
        function = source[source.index('publicar_ready() {'):source.index('\nfinalizar_arranque() {')]
        code = function.split('python3 - "$estado" <<\'PY\'\n', 1)[1].split('\nPY\n}', 1)[0]
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory)
            binary = state / 'vec-server'
            binary.write_bytes(b'binario-sintetico')
            manifest = state / 'manifest.json'
            config = state / 'config.json'
            records = {
                'material-manifest.json': {},
                'manifest.json': {'target': {'source_commit': 'app-fijada'}},
                'config.json': {'VEC_PORTAL_PROCESO': 'interno', 'VEC_BOLSA_POLITICA_OFERTAS_ENABLED': 'true'},
                'clon.json': {'contenedor': 'vec-fixture'},
                'sql-journal.json': {'source_ref': 'sql-aprobada', 'current_source_ref': 'app-fijada',
                    'verified_source_ref': 'otra-app' if fuente_alterada else 'app-fijada',
                    'approved_sql_ref': 'sql-aprobada', 'plan_sha': 'plan', 'inventory_sha': 'inventario',
                    'installed': [1, 2]},
                'runtime-process.json': {'container_mode': 'interno', 'runtime_manifest_path': str(manifest),
                    'runtime_config_path': str(config), 'exe': str(binary), 'source_commit': 'app-fijada',
                    'port': 18531, 'pid': 123, 'config_sha256': 'configuracion',
                    'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest()},
            }
            for name, record in records.items():
                (state / name).write_text(json.dumps(record))
            if binario_alterado:
                binary.write_bytes(b'otro-binario')
            # No se usa un certificado ni un almacén NSS real en esta fixture.
            certutil = state / 'certutil'
            certutil.write_text('#!/bin/sh\nexit 0\n')
            certutil.chmod(0o700)
            process = subprocess.run(['/usr/bin/python3', '-c', code, str(state)],
                env={'PATH': str(state) + ':/usr/bin:/bin'}, capture_output=True, timeout=5)
            ready = state / 'READY.json'
            return process.returncode, json.loads(ready.read_text()) if ready.exists() else None

    def test_ready_separa_fuente_app_y_plan_sql(self):
        status, ready = self.publicar_fixture()
        self.assertEqual(status, 0)
        self.assertEqual((ready['commit'], ready['sql_fuente_aprobada']), ('app-fijada', 'sql-aprobada'))
        self.assertEqual((ready['sql_plan_sha256'], ready['sql_inventario_sha256']), ('plan', 'inventario'))
        self.assertEqual(ready['configuracion_sha256'], 'configuracion')

    def test_binario_modificado_impide_ready(self):
        status, ready = self.publicar_fixture(binario_alterado=True)
        self.assertNotEqual(status, 0)
        self.assertIsNone(ready)

    def test_fuente_sql_verificada_distinta_impide_ready(self):
        status, ready = self.publicar_fixture(fuente_alterada=True)
        self.assertNotEqual(status, 0)
        self.assertIsNone(ready)


if __name__ == '__main__':
    unittest.main()
