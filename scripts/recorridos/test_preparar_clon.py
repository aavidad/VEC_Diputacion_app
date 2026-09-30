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

    def test_plan_sin_etapas_no_intenta_archivar_una_referencia_vacia(self):
        source = Path(__file__).with_name('preparar_clon.sh').read_text()
        loop = source[source.index('while IFS= read -r paso_sql; do'):source.index('# La última llamada verifica')]
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / 'git-invocado'
            script = '''set -eu
etapas_sql='[]'
repo='fixture'
estado="$VEC_TEST_STATE"
guiones='fixture'
git() { touch "$VEC_TEST_MARKER"; return 9; }
''' + loop
            process = subprocess.run(['bash', '-c', script], capture_output=True, timeout=5,
                env={'PATH': '/usr/bin:/bin', 'VEC_TEST_STATE': directory, 'VEC_TEST_MARKER': str(marker)})
            self.assertEqual(process.returncode, 0)
            self.assertFalse(marker.exists())
            self.assertFalse((Path(directory) / 'fuente-sql-').exists())

    def argumentos_runtime(self, operation):
        source = Path(__file__).with_name('preparar_clon.sh').read_text()
        function = source[source.index('runtime() {'):source.index('\ncomunicaciones() {')]
        with tempfile.TemporaryDirectory() as directory:
            arguments = Path(directory) / 'arguments'
            script = '''set -eu
marcador='fixture'
repo='fixture'
guiones='fixture'
estado='fixture'
puerto_web=18531
puerto_pg=55531
artefacto='/fixture/vec-server'
artefacto_sha="$(printf 'a%.0s' {1..64})"
artefacto_fuente="$(printf 'b%.0s' {1..40})"
python3() {
  if [[ "$1" == '-' ]]; then printf '%s\n' "$artefacto_fuente";
  else printf '%s\n' "$@" > "$VEC_TEST_ARGUMENTS"; fi
}
''' + function + '\nruntime "$VEC_TEST_OPERATION"\n'
            process = subprocess.run(['bash', '-c', script], capture_output=True, timeout=5,
                env={'PATH': '/usr/bin:/bin', 'VEC_TEST_ARGUMENTS': str(arguments), 'VEC_TEST_OPERATION': operation})
            self.assertEqual(process.returncode, 0)
            return arguments.read_text().splitlines()

    def test_build_transmite_artefacto_huella_y_fuente_juntos(self):
        arguments = self.argumentos_runtime('build')
        self.assertEqual(arguments[-6:], ['--artifact', '/fixture/vec-server', '--artifact-sha256', 'a'*64,
                                        '--artifact-source', 'b'*40])

    def test_verificacion_no_importa_artefactos(self):
        arguments = self.argumentos_runtime('verify')
        self.assertNotIn('--artifact', arguments)
        self.assertNotIn('--artifact-sha256', arguments)
        self.assertNotIn('--artifact-source', arguments)


if __name__ == '__main__':
    unittest.main()
