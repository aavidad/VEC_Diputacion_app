import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import tempfile
import unittest

from copia_local_rrhh import preparar, validar


REPO = Path(__file__).resolve().parents[2]
GENERADOR = REPO / 'scripts/generar_credenciales_desarrollo.sh'
BASE = Path('/dev/shm/vec/pg')


def huella(ruta):
    return hashlib.sha256(ruta.read_bytes()).hexdigest()


class MaterialRRHH(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.carpeta = Path(tempfile.mkdtemp(prefix='vec-rrhh-sintetico-'))
        cls.instancia = BASE / f'l-rrhhtest-{secrets.token_hex(4)}'
        cls.instancia.mkdir(mode=0o700)
        cls.origen = cls.carpeta / 'origen'
        subprocess.run([GENERADOR, cls.origen], check=True, stdout=subprocess.DEVNULL,
                       stderr=subprocess.DEVNULL, env={**os.environ,
                       'VEC_DESARROLLO_VIGENCIA_DIAS': '90'})
        identidad = json.loads((cls.origen / 'identidad/identidad.json').read_text())
        bback = cls.origen / 'identidad/bolsa-bback.json'
        bback.write_text(json.dumps({'version': 1, 'autoridad': 'no_autoritativo',
                          'sujeto': identidad['subject'],
                          'certificado_sha256': identidad['certificate_sha256'],
                          'perfil_ref': 'perfil-sintetico', 'unidad_ref': 'unidad-sintetica',
                          'ambito_ref': 'ambito-sintetico', 'bolsas_ref': ['bolsa-sintetica']}) + '\n')
        bback.chmod(0o600)
        entradas = []
        for nombre, rol in (('centro-solicitante', 'solicitante_centro'),
                            ('centro-ratificador', 'ratificador_centro')):
            clave = cls.origen / 'mtls' / f'{nombre}.key'
            cert = cls.origen / 'mtls' / f'{nombre}.crt'
            solicitud = cls.origen / 'mtls' / f'{nombre}.csr'
            extension = cls.origen / 'mtls' / f'{nombre}.ext'
            subprocess.run(['openssl', 'genpkey', '-algorithm', 'EC', '-pkeyopt',
                            'ec_paramgen_curve:P-256', '-out', clave], check=True,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            subprocess.run(['openssl', 'req', '-new', '-sha256', '-key', clave,
                            '-subj', f'/CN={nombre}', '-out', solicitud], check=True,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            extension.write_text('basicConstraints=critical,CA:FALSE\n'
                                 'keyUsage=critical,digitalSignature\n'
                                 'extendedKeyUsage=clientAuth\n')
            extension.chmod(0o600)
            subprocess.run(['openssl', 'x509', '-req', '-sha256', '-days', '90',
                            '-in', solicitud, '-CA', cls.origen / 'ca/ca.crt',
                            '-CAkey', cls.origen / 'ca/ca.key', '-set_serial',
                            '0x' + secrets.token_hex(16), '-extfile', extension,
                            '-out', cert], check=True, stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL)
            solicitud.unlink()
            extension.unlink()
            clave.chmod(0o600)
            cert.chmod(0o600)
            password = cls.origen / 'mtls' / f'{nombre}.p12.password'
            password.write_text(secrets.token_hex(32) + '\n')
            password.chmod(0o600)
            p12 = cls.origen / 'mtls' / f'{nombre}.p12'
            subprocess.run(['openssl', 'pkcs12', '-export', '-out', p12,
                            '-inkey', clave, '-in', cert,
                            '-certfile', cls.origen / 'ca/ca.crt',
                            '-passout', f'file:{password}'], check=True,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            p12.chmod(0o600)
            certificado_der = subprocess.check_output(['openssl', 'x509', '-in',
                                                        cert, '-outform', 'DER'])
            sujeto = f'desarrollo:{nombre}'
            identidad_centro = cls.origen / 'identidad' / f'{nombre}.json'
            identidad_centro.write_text(json.dumps({
                'version': 1, 'autoridad': 'no_autoritativo', 'subject': sujeto,
                'display_name': nombre, 'roles': [rol],
                'certificate_sha256': hashlib.sha256(certificado_der).hexdigest()}) + '\n')
            identidad_centro.chmod(0o600)
            entrada = {'certificate': f'mtls/{nombre}.crt',
                       'identity': f'identidad/{nombre}.json', 'role': rol,
                       'centro_ref': 'centro-520', 'puesto_ref': 'puesto-prueba'}
            if rol == 'solicitante_centro':
                entrada['ratificador_subject'] = 'desarrollo:centro-ratificador'
            entradas.append(entrada)
        opcional = cls.origen / 'identidad/centros.json'
        opcional.write_text(json.dumps({'version': 1, 'autoridad': 'no_autoritativo',
                                        'entradas': entradas}) + '\n')
        opcional.chmod(0o600)
        for nombre in ('documentos.json', 'usuarios-preferencias-interna.json',
                       'auditoria.json'):
            descriptor = cls.origen / 'identidad' / nombre
            descriptor.write_text('{"dsn":"postgresql://ajeno/base"}\n')
            descriptor.chmod(0o600)
        excluido = cls.origen / 'modulos'
        excluido.mkdir(mode=0o700)
        (excluido / 'documentos.json').write_text('{"dsn":"postgresql://ajeno/base"}\n')
        (excluido / 'documentos.json').chmod(0o600)
        (excluido / 'antiguo.dsn').write_text('postgresql://ajeno/base\n')
        (excluido / 'antiguo.dsn').chmod(0o600)
        (excluido / 'script.antes').write_text('no se copia\n')
        (excluido / 'script.antes').chmod(0o600)
        (excluido / 'enlace-antiguo').symlink_to(opcional)

    @classmethod
    def tearDownClass(cls):
        shutil.rmtree(cls.carpeta)
        shutil.rmtree(cls.instancia)

    def test_conserva_identidad_y_renueva_servidor_con_ca_original(self):
        destino = self.instancia / 'material'
        originales = {nombre: huella(self.origen / nombre) for nombre in (
            'ca/ca.crt', 'ca/ca.key', 'ca/serie', 'mtls/cliente.crt',
            'mtls/cliente.key', 'mtls/cliente.p12', 'mtls/cliente.p12.password',
            'identidad/identidad.json', 'mtls/intervencion.crt',
            'mtls/intervencion.key', 'mtls/intervencion.p12',
            'kms/clave-maestra.bin',
            'tsa/clave-hmac.bin', 'identidad/bolsa-bback.json',
            'identidad/centros.json',
            *(f'mtls/{nombre}.{extension}' for nombre in
              ('centro-solicitante', 'centro-ratificador') for extension in
              ('crt', 'key', 'p12', 'p12.password')),
            'identidad/centro-solicitante.json',
            'identidad/centro-ratificador.json')}
        servidor_original = huella(self.origen / 'tls/servidor.crt')
        preparar(self.origen, destino, REPO, BASE)
        for nombre, esperado in originales.items():
            self.assertEqual(huella(destino / nombre), esperado, nombre)
        self.assertNotEqual(huella(destino / 'tls/servidor.crt'),
                            servidor_original)
        self.assertEqual(huella(self.origen / 'tls/servidor.crt'), servidor_original)
        self.assertFalse((destino / 'modulos').exists())
        for nombre in ('documentos.json', 'usuarios-preferencias-interna.json',
                       'auditoria.json'):
            self.assertFalse((destino / 'identidad' / nombre).exists())
        subprocess.run(['openssl', 'verify', '-purpose', 'sslserver', '-CAfile',
                        destino / 'ca/ca.crt', destino / 'tls/servidor.crt'],
                       check=True, stdout=subprocess.DEVNULL)
        self.assertEqual(subprocess.run(['openssl', 'x509', '-in',
                          destino / 'tls/servidor.crt', '-noout', '-checkend',
                          str(89 * 86400)], stdout=subprocess.DEVNULL).returncode, 0)
        self.assertNotEqual(subprocess.run(['openssl', 'x509', '-in',
                             destino / 'tls/servidor.crt', '-noout', '-checkend',
                             str(91 * 86400)], stdout=subprocess.DEVNULL).returncode, 0)
        manifiesto = json.loads((destino / 'manifiesto.json').read_text())
        self.assertEqual(manifiesto['huella_cliente_sha256'],
                         json.loads((self.origen / 'manifiesto.json').read_text())['huella_cliente_sha256'])
        self.assertNotEqual(manifiesto['huella_servidor_sha256'],
                            json.loads((self.origen / 'manifiesto.json').read_text())['huella_servidor_sha256'])
        entorno = (destino / 'desarrollo.env').read_text()
        self.assertIn(str(destino), entorno)
        self.assertNotIn(str(self.origen), entorno)

    def test_rechaza_enlaces_y_piezas_ausentes_antes_de_publicar(self):
        fuente = self.carpeta / 'invalida'
        shutil.copytree(self.origen, fuente)
        destino = self.carpeta / 'sin-publicar'
        (fuente / 'mtls/cliente.key').unlink()
        (fuente / 'mtls/cliente.key').symlink_to(self.origen / 'mtls/cliente.key')
        with self.assertRaisesRegex(ValueError, 'enlazada'):
            preparar(fuente, destino, REPO, BASE)
        self.assertFalse(destino.exists())
        (fuente / 'mtls/cliente.key').unlink()
        with self.assertRaisesRegex(ValueError, 'Falta una pieza'):
            validar(fuente, REPO, BASE)
        self.assertFalse(destino.exists())

    def test_generador_normal_crea_material_independiente(self):
        otra = self.carpeta / 'normal'
        subprocess.run([GENERADOR, otra], check=True, stdout=subprocess.DEVNULL,
                       stderr=subprocess.DEVNULL)
        self.assertNotEqual(huella(otra / 'ca/ca.crt'), huella(self.origen / 'ca/ca.crt'))
        self.assertTrue((otra / 'manifiesto.json').is_file())

    def test_rechaza_conexion_ajena(self):
        fuente = self.carpeta / 'con-conexion'
        shutil.copytree(self.origen, fuente)
        configuracion = fuente / 'identidad' / 'centros.json'
        configuracion.write_text('{"dsn":"postgresql://otro-servidor/base"}\n')
        configuracion.chmod(0o600)
        with self.assertRaisesRegex(ValueError, 'conexiones ajenas'):
            preparar(fuente, self.carpeta / 'sin-conexion', REPO, BASE)
        self.assertFalse((self.carpeta / 'sin-conexion').exists())

    def test_rechaza_bback_incoherente_o_ausente(self):
        fuente = self.carpeta / 'bback-invalido'
        shutil.copytree(self.origen, fuente)
        bback = fuente / 'identidad/bolsa-bback.json'
        datos = json.loads(bback.read_text())
        datos['sujeto'] = 'otro-sujeto'
        bback.write_text(json.dumps(datos) + '\n')
        destino = self.carpeta / 'sin-bback'
        with self.assertRaisesRegex(ValueError, 'B-BACK'):
            preparar(fuente, destino, REPO, BASE)
        self.assertFalse(destino.exists())
        bback.unlink()
        with self.assertRaisesRegex(ValueError, 'Falta una pieza'):
            validar(fuente, REPO, BASE)

    def test_renovar_servidor_rechaza_fuente_privada_fuera_de_staging(self):
        fuente = self.carpeta / '.material-rrhh-fuente'
        shutil.copytree(self.origen, fuente)
        anterior = huella(fuente / 'tls/servidor.crt')
        resultado = subprocess.run([GENERADOR, '--renovar-servidor', fuente],
                                   capture_output=True)
        self.assertNotEqual(resultado.returncode, 0)
        self.assertEqual(huella(fuente / 'tls/servidor.crt'), anterior)

    def test_renovar_servidor_rechaza_padre_enlazado(self):
        alias = BASE / (self.instancia.name + '-enlace')
        alias.symlink_to(self.instancia, target_is_directory=True)
        try:
            fuente = self.instancia / '.material-rrhh-abcdefgh'
            shutil.copytree(self.origen, fuente)
            anterior = huella(fuente / 'tls/servidor.crt')
            resultado = subprocess.run([GENERADOR, '--renovar-servidor',
                                       alias / fuente.name], capture_output=True)
            self.assertNotEqual(resultado.returncode, 0)
            self.assertEqual(huella(fuente / 'tls/servidor.crt'), anterior)
        finally:
            alias.unlink()
            shutil.rmtree(fuente)


if __name__ == '__main__':
    unittest.main()
