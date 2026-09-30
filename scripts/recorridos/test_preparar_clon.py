from pathlib import Path
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


if __name__ == '__main__':
    unittest.main()
