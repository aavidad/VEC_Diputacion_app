import http.server
import json
from pathlib import Path
import ssl
import subprocess
import sys
import tempfile
import threading
import unittest


class TransporteHTTPTest(unittest.TestCase):
    def test_https_mutuo_omite_connection_y_reconoce_cierre(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            def openssl(*args):
                subprocess.run(['openssl', *args], cwd=raiz, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            openssl('req', '-x509', '-newkey', 'rsa:2048', '-noenc', '-days', '90',
                    '-subj', '/CN=Autoridad local', '-keyout', 'ca.key', '-out', 'ca.pem',
                    '-addext', 'basicConstraints=critical,CA:TRUE',
                    '-addext', 'keyUsage=critical,keyCertSign,cRLSign')
            openssl('req', '-new', '-newkey', 'rsa:2048', '-noenc',
                    '-subj', '/CN=localhost', '-keyout', 'cliente.key', '-out', 'cliente.csr')
            (raiz / 'extensiones').write_text(
                'subjectAltName=DNS:localhost\n'
                'basicConstraints=critical,CA:FALSE\n'
                'keyUsage=critical,digitalSignature,keyEncipherment\n'
                'extendedKeyUsage=serverAuth,clientAuth\n'
                'subjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n')
            openssl('x509', '-req', '-in', 'cliente.csr', '-CA', 'ca.pem',
                    '-CAkey', 'ca.key', '-CAcreateserial', '-days', '90',
                    '-extfile', 'extensiones', '-out', 'cliente.pem')
            observadas = []
            class Receptor(http.server.BaseHTTPRequestHandler):
                def log_message(self, *_args):
                    pass
                def do_POST(self):
                    cuerpo = self.rfile.read(int(self.headers['Content-Length']))
                    observadas.append((self.path, self.headers, cuerpo))
                    b = b'{"error":{"codigo":"servicio_no_disponible"}}'
                    self.send_response(503)
                    self.send_header('Content-Type', 'application/json')
                    self.send_header('Content-Length', str(len(b)))
                    self.end_headers()
                    self.wfile.write(b)
            servidor = http.server.HTTPServer(('127.0.0.1', 0), Receptor)
            tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            tls.load_cert_chain(raiz / 'cliente.pem', raiz / 'cliente.key')
            tls.load_verify_locations(raiz / 'ca.pem')
            tls.verify_mode = ssl.CERT_REQUIRED
            servidor.socket = tls.wrap_socket(servidor.socket, server_side=True)
            hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
            hilo.start()
            entrada = json.dumps(dict.fromkeys(('padre_ref', 'version_padre', 'oferta_ref',
                'numero_plaza', 'secuencia_plaza', 'participacion_ref',
                'recibo_adjudicacion_ref', 'clave_idempotencia'), 'referencia:ficticia')).encode()
            (raiz / 'entrada.json').write_bytes(entrada)
            try:
                salida = subprocess.run([sys.executable, str(Path(__file__).with_name('ensayar_http.py')),
                    '--origen', f'https://localhost:{servidor.server_port}',
                    '--ca', str(raiz / 'ca.pem'), '--certificado', str(raiz / 'cliente.pem'),
                    '--clave', str(raiz / 'cliente.key'), '--entrada', str(raiz / 'entrada.json'),
                    '--salida', str(raiz / 'resultado'), '--esperar-cerrada'],
                    capture_output=True, text=True, timeout=10)
                self.assertEqual(salida.returncode, 0, salida.stderr)
                self.assertEqual(json.loads(salida.stdout)['http'], 503)
                self.assertEqual(len(observadas), 1)
                self.assertEqual(observadas[0][0], '/api/vec/contratacion-temporal/nombramientos/derivaciones')
                self.assertNotIn('Connection', observadas[0][1])
                self.assertEqual(observadas[0][2], entrada)
            finally:
                servidor.shutdown()
                hilo.join(timeout=5)
                servidor.server_close()
