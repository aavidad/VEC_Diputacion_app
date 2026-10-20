"""Cotejos focales de admisiones y configuración con copia sintética aislada."""

from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest

from recorrido_copia import admisiones, configuracion_firma

ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')


class CopiaFalsa:
    nombre = 'l-f131'

    def leer(self):
        return {'sid_copia': '12345678901234567'}


class Contexto:
    def __init__(self, raiz):
        self.raiz = raiz / CopiaFalsa.nombre
        self.carpeta = self.raiz / 'recorrido'
        self.material = self.raiz / 'material'
        self.copia = CopiaFalsa()
        self.parametros = {'organizacion_operativa_ref': 'organizacion:desarrollo:dipgra'}
        self.filas = []
        self.consultas = []
        self.raiz.mkdir(mode=0o700)
        self.carpeta.mkdir(mode=0o700)
        self.material.mkdir(mode=0o700)
        self._semilla()

    @staticmethod
    def sha(ruta):
        return hashlib.sha256(Path(ruta).read_bytes()).hexdigest()

    def leer(self, ruta):
        return json.loads(Path(ruta).read_bytes())

    def guardar(self, ruta, contenido):
        ruta = Path(ruta)
        ruta.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
        b = contenido if isinstance(contenido, bytes) else (json.dumps(
            contenido, ensure_ascii=False, separators=(',', ':')) + '\n').encode()
        if ruta.exists():
            if ruta.read_bytes() != b:
                raise ValueError('No se puede sustituir material anterior')
            return
        ruta.write_bytes(b)
        ruta.chmod(0o600)

    def ahora(self):
        return datetime(2026, 10, 20, 12, tzinfo=timezone.utc)

    def consultar(self, sql):
        self.consultas.append(sql)
        if 'pg_control_system' in sql:
            return '12345678901234567'
        return json.dumps(self.filas)

    def _semilla(self):
        f = self.carpeta / 'firmantes'
        refs, firmantes, rows = {}, [], []
        vence = (self.ahora() + timedelta(days=88)).strftime('%Y-%m-%dT%H:%M:%SZ')
        for n, rol in enumerate(ROLES, 1):
            persona, cuenta, cert = f'per_{n:032x}', f'cta_{n:032x}', f'{n:064x}'
            slots = [f'prf_{n*4+i:032x}' for i in range(4)]
            perfil, consulta, documentos, circuito = slots
            recibo = {'estado': 'vigente', 'version': 1, 'certificado_der_sha256': cert,
                      'organizacion_destino': {'organizacion_ref': 'org_' + 'a' * 32},
                      'vinculo_ref': f'vcc_{n:032x}', 'recibo_ref': f'recibo:{rol}',
                      'decision_ref': f'decision:{rol}', 'auditoria_ref': f'auditoria:{rol}',
                      'estado_replay': 'nueva'}
            self.guardar(self.carpeta / 'actos/recibos' / (rol + '-certificado.json'), {'recibo': recibo})
            refs[rol] = {'persona_ref': persona, 'cuenta_ref': cuenta,
                         'certificado_sha256': cert, 'perfil': {'firma': perfil},
                         'vinculo': {'firma': f'vinculo:{rol}'}}
            firmantes.append({'persona_ref': persona, 'cuenta_ref': cuenta, 'certificado_sha256': cert,
                              'perfil_firma_ref': perfil, 'perfil_consulta_ct_ref': consulta,
                              'perfil_documentos_ref': documentos, 'perfil_consulta_circuito_ref': circuito})
            row = {'rol': rol, 'perfil_ref': perfil, 'asignacion_ref': f'asignacion:{rol}',
                   'asignacion_version': 1, 'asignacion_sha256': 'a' * 64,
                   'rol_version_ref': f'rol:ct_cargo_{rol}_firma_dev:v1', 'rol_sha256': 'b' * 64,
                   'control_revision': 1, 'control_sha256': 'c' * 64,
                   'persona_ref': persona, 'cuenta_ref': cuenta, 'certificado_sha256': cert,
                   'vinculo_certificado_ref': recibo['vinculo_ref'],
                   'recibo_ref': recibo['recibo_ref'], 'decision_ref': recibo['decision_ref'],
                   'auditoria_ref': recibo['auditoria_ref'],
                   'recibo_db': {k: v for k, v in recibo.items() if k != 'estado_replay'}}
            rows.append(row)
        self.filas = rows
        self.guardar(f / 'semilla-final/firmantes-datos.json', {
            'referencias': refs, 'firmantes': firmantes, 'unidad_ref': 'unidad:rrhh',
            'organizacion_ca_ref': 'org_' + 'a' * 32, 'retirada_en': vence,
            'politica_ref': 'pga_' + 'a' * 32, 'politica_sha256': 'd' * 64})
        for prefijo, ids in (('roles', ['ct_cargo_' + rol + '_firma_dev' for rol in ROLES] +
                              ['extra_' + str(i) for i in range(12)]),
                             ('consulta-v2', ['consulta_' + str(i) for i in range(4)])):
            plan = f / (prefijo + '-plan.json')
            acuse = f / (prefijo + '-acuse.json')
            self.guardar(plan, {'cargos': [{'rol_id': i, 'version': 1, 'version_rol_sha256': 'b' * 64}
                                           for i in ids]})
            self.guardar(acuse, {'estado': 'permitido', 'recibo': {
                'auditoria_ref': 'auditoria:aut53', 'plan_sha256': self.sha(plan),
                'cargos': [{'version_rol_ref': f'rol:{i}:v1', 'version_rol_sha256': 'b' * 64,
                            'control_sha256': 'c' * 64} for i in ids]}})
        self.guardar(f / 'certificados/clientes.crl', b'CRL local de prueba')
        self.guardar(f / 'metadata.json', {
            'copia_ref': self.copia.nombre, 'autoridad_hasta': vence,
            'roles_plan': str(f / 'roles-plan.json'), 'roles_acuse': str(f / 'roles-acuse.json'),
            'roles_acuse_sha256': self.sha(f / 'roles-acuse.json'),
            'consulta_v2_plan': str(f / 'consulta-v2-plan.json'),
            'consulta_v2_acuse': str(f / 'consulta-v2-acuse.json'),
            'consulta_v2_acuse_sha256': self.sha(f / 'consulta-v2-acuse.json'),
            'certificados_crl': str(f / 'certificados/clientes.crl')})
        self.guardar(self.carpeta / 'actos/metadata.json', {
            'copia_ref': self.copia.nombre, 'actos_confirmados': 14,
            'recibos': [f'recibo:{i}' for i in range(14)]})
        catalogo = {'id': 'ct.plan.firma.hz39.operativa', 'version': 1, 'revision': 1,
                    'estado': 'publicado', 'publicado_en': '2026-10-20T12:00:00Z',
                    'aprobacion_ref': 'aprobacion:plan',
                    'entradas': [{'atributos': {'organizacion_ref': self.parametros['organizacion_operativa_ref'],
                                               'unidad_ref': 'unidad:rrhh'}} for _ in range(5)]}
        catalogo_raw = json.dumps(catalogo, separators=(',', ':')).encode()
        catalogo_sha = hashlib.sha256(catalogo_raw).hexdigest()
        plan = self.material / 'identidad/firma-recursos/plan-firma-publicado.json'
        self.guardar(plan, {'version_esquema': 1, 'fuente': {'demostracion': False,
                    'origen_sha256': catalogo_sha}, 'catalogo': catalogo})
        recibo = self.carpeta / 'plan/recibos/publicar.json'
        self.guardar(recibo, {'recibo': {'accion': 'vec.catalogos.publicar',
            'catalogo_ref': 'ct.plan.firma.hz39.operativa:1', 'estado': 'publicado',
            'revision': 1, 'publicacion_sha256': catalogo_sha,
            'recibo_ref': 'plan:publicar', 'auditoria_ref': 'auditoria:plan'}})
        self.guardar(self.carpeta / 'plan/metadata.json', {'copia_ref': self.copia.nombre,
            'plan_publicado': True, 'catalogo_sha256': catalogo_sha,
            'paquete_sha256': self.sha(plan), 'recibos': ['plan:crear', 'plan:publicar']})


class ConfiguracionTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = Contexto(Path(self.tmp.name))

    def test_configuracion_cuatro_admisiones_y_replay(self):
        meta_adm = admisiones.preparar(self.c)
        self.assertEqual(meta_adm['filas'], 4)
        meta = configuracion_firma.preparar(self.c)
        self.assertEqual(meta['admisiones_cotejadas'], 4)
        self.assertEqual(configuracion_firma.preparar(self.c), meta)
        self.assertEqual(len(self.c.leer(self.c.material / 'identidad/firma-vec.json')['firmantes']), 4)

    def test_fila_durable_ausente_cierra_configuracion(self):
        self.c.filas.pop()
        with self.assertRaisesRegex(ValueError, 'Cuatro admisiones'):
            admisiones.preparar(self.c)
        self.assertFalse((self.c.material / 'identidad/firma-vec.json').exists())

    def test_recibo_certificado_divergente_falla_cotejo(self):
        self.c.filas[0]['recibo_db']['decision_ref'] = 'otra:decision'
        with self.assertRaisesRegex(ValueError, 'Recibo durable'):
            admisiones.preparar(self.c)
        self.assertFalse((self.c.carpeta / 'admisiones/metadata.json').exists())

    def test_publicacion_plan_divergente_impide_firma_vec(self):
        admisiones.preparar(self.c)
        ruta = self.c.carpeta / 'plan/recibos/publicar.json'
        respuesta = self.c.leer(ruta)
        respuesta['recibo']['publicacion_sha256'] = 'e' * 64
        ruta.write_text(json.dumps(respuesta))
        with self.assertRaisesRegex(ValueError, 'Plan sin publicación'):
            configuracion_firma.preparar(self.c)
        self.assertFalse((self.c.material / 'identidad/firma-vec.json').exists())


if __name__ == '__main__':
    unittest.main()
