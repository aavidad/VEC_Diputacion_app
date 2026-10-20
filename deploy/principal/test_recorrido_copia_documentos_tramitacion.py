"""La técnica original conserva su identidad y obtiene un perfil documental separado."""

import hashlib
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest

from recorrido_copia import documentos_tramitacion as dt
from recorrido_copia.personal import _referencia


SID = '12345678901234567'
HUELLA = hashlib.sha256(b'certificado-original-der').hexdigest()
SUJETO = 'desarrollo:operador-rrhh'
PERSONA = _referencia('per_', SUJETO, HUELLA, 'persona')
CUENTA = _referencia('cta_', SUJETO, HUELLA, 'cuenta')
ORIGINAL = _referencia('prf_', SUJETO, HUELLA, 'perfil-bolsa-bback-v1')
ROL = {'version_rol_ref': dt.ROL, 'version_rol_sha256': 'a' * 64, 'control_sha256': 'b' * 64}


class Contexto:
    def __init__(self, carpeta):
        self.raiz = carpeta
        self.carpeta = carpeta / 'recorrido'
        self.material = carpeta / 'material'
        self.fuente = Path(__file__).resolve().parents[2]
        self.copia = SimpleNamespace(leer=lambda: {'sid_copia': SID, 'recorrido_completo': True})
        self.parametros = {'organizacion_operativa_ref': 'organizacion:desarrollo:dipgra'}
        self.sql_efectos = []
        self.publicada = False
        self.carpeta.mkdir(mode=0o700)
        (self.material / 'identidad').mkdir(parents=True, mode=0o700)
        (self.material / 'mtls').mkdir(mode=0o700)
        self.guardar(self.material / 'identidad/identidad.json', {
            'version': 1, 'autoridad': 'no_autoritativo', 'roles': ['tecnico_rrhh'],
            'subject': SUJETO, 'certificate_sha256': HUELLA})
        self.guardar(self.material / 'identidad/bolsa-bback.json', {
            'sujeto': SUJETO,
            'perfil_ref': ORIGINAL, 'certificado_sha256': HUELLA})
        cert = self.material / 'mtls/cliente.crt'
        cert.write_bytes(b'certificado-local')
        cert.chmod(0o600)
        nominales = []
        for n in range(4):
            nominales.append({'persona_ref': 'per_' + f'{n+6:032x}',
                'cuenta_ref': 'cta_' + f'{n+6:032x}', 'certificado_sha256': f'{n+6:064x}',
                'perfil_firma_ref': 'prf_' + f'{n*4+10:032x}',
                'perfil_consulta_ct_ref': 'prf_' + f'{n*4+11:032x}',
                'perfil_documentos_ref': 'prf_' + f'{n*4+12:032x}',
                'perfil_consulta_circuito_ref': 'prf_' + f'{n*4+13:032x}'})
        self.guardar(self.carpeta / 'firmantes/semilla-final/firmantes-datos.json', {
            'unidad_ref': 'unidad:centro:520', 'retirada_en': '2027-01-16T12:00:00Z',
            'firmantes': nominales})

    @staticmethod
    def sha(ruta):
        return hashlib.sha256(ruta.read_bytes()).hexdigest()

    @staticmethod
    def leer(ruta):
        return json.loads(ruta.read_bytes())

    @staticmethod
    def guardar(ruta, datos):
        ruta.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
        raw = datos if isinstance(datos, bytes) else (json.dumps(datos, separators=(',', ':')) + '\n').encode()
        if ruta.exists():
            if ruta.read_bytes() != raw:
                raise ValueError('Material previo divergente')
            return
        ruta.write_bytes(raw)
        ruta.chmod(0o600)

    def ejecutar(self, argv, nombre_log):
        assert nombre_log == 'documentos-tramitacion-certificado.log'
        assert argv[0:3] == ('/usr/bin/openssl', 'x509', '-in')
        return SimpleNamespace(stdout=b'certificado-original-der')

    def consultar(self, consulta):
        if 'pg_control_system' in consulta:
            return SID
        if consulta.startswith('SELECT jsonb_build_object'):
            return json.dumps({'vinculo': {
                'vinculo_ref': 'vca_' + '4' * 32, 'perfil_ref': ORIGINAL,
                'persona_ref': PERSONA, 'cuenta_ref': CUENTA,
                'procedencia_ref': 'prc_' + '5' * 32,
                'procedencia_huella_sha256': 'c' * 64},
                'perfil': {'persona_ref': PERSONA},
                'cuenta': {'cuenta_ref': CUENTA}, 'persona': {'persona_ref': PERSONA}})
        if consulta.startswith('SELECT count(*) FROM vec_contexto_actor_v1.perfil_actual'):
            return '1' if self.publicada else '0'
        raise AssertionError(consulta[:100])

    def sql(self, raw, log):
        self.sql_efectos.append((raw, log))
        self.publicada = True


class DocumentosTramitacionTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = Contexto(Path(self.tmp.name))

    def test_cuenta_original_nuevo_perfil_y_repeticion_sin_nuevo_efecto(self):
        cuenta = dt.preparar(self.c, rol_documental=ROL, perfiles_reservados={ORIGINAL})
        self.assertEqual(cuenta['certificado_sha256'], HUELLA)
        self.assertEqual(cuenta['cuenta_ref'], CUENTA)
        self.assertNotEqual(cuenta['perfil_ref'], ORIGINAL)
        self.assertEqual(len(self.c.sql_efectos), 1)
        sql = self.c.sql_efectos[0][0].decode()
        self.assertIn('pg_control_system()', sql)
        self.assertIn('rol_administrable_exacto_v1', sql)
        self.assertIn("ra.unidad_requerida IS FALSE", sql)
        self.assertIn("'semilla:ensayo:f131:documentos-tramitacion'", sql)
        self.assertEqual(self.c.leer(self.c.carpeta / 'documentos/tramitacion-acuse.json')['cuenta'], cuenta)
        self.assertEqual(dt.preparar(self.c, rol_documental=ROL, perfiles_reservados={ORIGINAL}), cuenta)
        self.assertEqual(len(self.c.sql_efectos), 1)

    def test_certificado_original_divergente_cierra_sin_sql(self):
        ruta = self.c.material / 'identidad/identidad.json'
        identidad = self.c.leer(ruta)
        identidad['certificate_sha256'] = 'f' * 64
        ruta.write_text(json.dumps(identidad))
        with self.assertRaisesRegex(ValueError, 'identidad documental'):
            dt.preparar(self.c, rol_documental=ROL, perfiles_reservados={ORIGINAL})
        self.assertEqual(self.c.sql_efectos, [])

    def test_perfil_bolsa_de_otra_identidad_cierra_antes_de_semilla(self):
        ruta = self.c.material / 'identidad/bolsa-bback.json'
        manifiesto = self.c.leer(ruta)
        manifiesto['perfil_ref'] = 'prf_' + 'f' * 32
        ruta.write_text(json.dumps(manifiesto))
        with self.assertRaisesRegex(ValueError, 'manifiesto de Bolsa'):
            dt.preparar(self.c, rol_documental=ROL, perfiles_reservados=set())
        self.assertEqual(self.c.sql_efectos, [])
        self.assertFalse((self.c.carpeta / 'documentos/tramitacion-fuente.json').exists())

    def test_sid_ajeno_y_rol_divergente_cierran(self):
        self.c.copia.leer = lambda: {'sid_copia': '9' * 17, 'recorrido_completo': True}
        with self.assertRaisesRegex(ValueError, 'SID exacto'):
            dt.preparar(self.c, rol_documental=ROL, perfiles_reservados={ORIGINAL})
        self.c.copia.leer = lambda: {'sid_copia': SID, 'recorrido_completo': True}
        with self.assertRaisesRegex(ValueError, 'AUT53'):
            dt.preparar(self.c, rol_documental={**ROL, 'control_sha256': 'malo'},
                        perfiles_reservados={ORIGINAL})
        self.assertEqual(self.c.sql_efectos, [])

    def test_sin_postimagen_no_escribe_acuse(self):
        self.c.sql = lambda raw, log: None
        with self.assertRaisesRegex(ValueError, 'sin postimagen'):
            dt.preparar(self.c, rol_documental=ROL, perfiles_reservados={ORIGINAL})
        self.assertFalse((self.c.carpeta / 'documentos/tramitacion-acuse.json').exists())

    def test_postimagen_previa_sin_semilla_rechaza_dos_intentos(self):
        self.c.publicada = True
        semilla = self.c.carpeta / 'documentos/tramitacion-semilla.sql'
        for _ in range(2):
            with self.assertRaisesRegex(ValueError, 'sin semilla declarada'):
                dt.preparar(self.c, rol_documental=ROL, perfiles_reservados=set())
            self.assertFalse(semilla.exists())
        self.assertEqual(self.c.sql_efectos, [])


if __name__ == '__main__':
    unittest.main()
