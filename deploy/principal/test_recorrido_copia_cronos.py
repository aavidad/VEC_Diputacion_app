"""Etapa de Cronos de la persona empleada en la copia (F3-01, corte C1)."""

from datetime import datetime, timezone
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock
from urllib.parse import parse_qs, urlsplit

from recorrido_copia import cronos
from recorrido_copia.comun import Contexto


SID = '12345678901234567'
PERSONA = 'per_' + '1' * 32
CUENTA = 'cta_' + '2' * 32
PERFIL_ORIGEN = 'prf_' + '3' * 32


class EtapaCronosTest(unittest.TestCase):
    def setUp(self):
        temporal = tempfile.TemporaryDirectory()
        self.addCleanup(temporal.cleanup)
        self.raiz = Path(temporal.name)
        self.raiz.chmod(0o700)
        material = self.raiz / 'material'
        for sub in ('ca', 'identidad'):
            (material / sub).mkdir(mode=0o700, parents=True)
        material.chmod(0o700)
        (material / 'ca/ca.crt').write_text('CA ficticia de esta prueba')
        (material / 'ca/ca.crt').chmod(0o600)
        self.material = material
        self._json(material / 'identidad/tecnico-rrhh-adicional-identidad.json', {
            'version': 1, 'autoridad': 'no_autoritativo', 'certificate_sha256': 'a' * 64,
            'subject': 'desarrollo:rrhh:adicional:prueba'})
        self._json(material / 'identidad/tecnico-rrhh-adicional.json', {
            'sujeto': 'desarrollo:rrhh:adicional:prueba', 'cuenta_ref': CUENTA,
            'persona_ref': PERSONA, 'perfil_ref': PERFIL_ORIGEN})
        carpeta = self.raiz / 'recorrido'
        carpeta.mkdir(mode=0o700)
        self.estado = {'recorrido_completo': True, 'sid_copia': SID, 'puerto_pg': 55611}
        copia = Mock(nombre='vec-copia-j-cronos', leer=lambda: self.estado, raiz=self.raiz)
        self.c = Mock(copia=copia, raiz=self.raiz, material=material, carpeta=carpeta,
                      guardar=Contexto.guardar, leer=Contexto.leer, sha=Contexto.sha)
        self.c.ahora.return_value = datetime(2026, 10, 21, tzinfo=timezone.utc)
        self.origenes = []
        self.c.consultar.side_effect = self._consultar

    @staticmethod
    def _json(ruta, datos):
        ruta.write_text(json.dumps(datos))
        ruta.chmod(0o600)

    def _consultar(self, consulta):
        if 'pg_control_system' in consulta:
            return SID
        if 'motivo_v2_catalogo_publicado' in consulta:
            return '\n'.join(sorted({cronos.MOTIVO_CONSULTA, cronos.MOTIVO_ACTUACION, cronos.MOTIVO_DENEGADO}))
        if 'configuracion_origen_consumos_v1' in consulta:
            return '\n'.join(self.origenes)
        if 'jsonb_build_object(\'ref\'' in consulta:
            return json.dumps({'ref': 'prc_' + '4' * 32, 'version': 1, 'huella': '4' * 64,
                               'autoridad': 'autoridad_maestra_acreditada'})
        if 'FROM vec_cronos_v1.calendario_asignacion WHERE empleado_ref=\'emp_cae' in consulta:
            return '[{"anio":2026,"calendario_ref":"calendario:cronos:sintetico.granada.2026"}]'
        if 'FROM vec_cronos_v1.programacion_jornada WHERE' in consulta:
            return '[]'
        if 'resolver_empleado_canonico_persona_v1' in consulta:
            refs = json.loads((self.c.carpeta / 'cronos/semilla.json').read_text())
            return 'empleado|' + refs['empleado_ref']
        if 'count(*)' in consulta:
            return '1'
        raise AssertionError('consulta inesperada: ' + consulta[:80])

    def test_conexiones_ocho_identidades_tls_y_sin_segunda_alta(self):
        valores = cronos.conexiones(self.c)
        self.assertEqual(set(valores), set(cronos.GRUPOS))
        urls = [urlsplit(v) for v in valores.values()]
        self.assertEqual(len({u.username for u in urls}), 8)
        self.assertEqual(len({u.password for u in urls}), 8)
        for u in urls:
            self.assertEqual((u.hostname, u.port, u.path), ('localhost', 55611, '/postgres'))
            self.assertEqual(parse_qs(u.query), {'sslmode': ['verify-full'],
                'sslrootcert': [str(self.material / 'ca/ca.crt')]})
        sentencia = self.c.sql.call_args.args[0].decode()
        self.assertEqual(sentencia.count('CREATE ROLE '), 8)
        self.assertEqual(sentencia.count('WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;'), 8)
        self.assertIn('GRANT vec_cronos_v1_ejecutor TO ', sentencia)
        self.assertEqual(cronos.conexiones(self.c), valores)
        self.c.sql.assert_called_once()

    def test_alta_incierta_no_se_repite(self):
        self.c.sql.side_effect = ValueError('respuesta incierta')
        with self.assertRaisesRegex(ValueError, 'respuesta incierta'):
            cronos.conexiones(self.c)
        with self.assertRaisesRegex(ValueError, 'conciliar'):
            cronos.conexiones(self.c)
        self.c.sql.assert_called_once()

    def test_conexion_ajena_guardada_se_rechaza(self):
        cronos.conexiones(self.c)
        ruta = self.c.carpeta / 'cronos/conexiones.json'
        valores = json.loads(ruta.read_bytes())
        valores['dsn_cronos'] = valores['dsn_cronos'].replace('@localhost:', '@otra:')
        ruta.write_text(json.dumps(valores))
        with self.assertRaisesRegex(ValueError, 'ajena'):
            cronos.conexiones(self.c)

    def test_origenes_ocho_filas_exactas_e_idempotentes(self):
        login = 'vcl_j_cronos_0123456789ab_137'
        esperado = sorted('|'.join((login, a, o, cronos.PROCESO, cronos.CANAL)) for a, o in cronos.CONSUMOS)

        def aplicar(datos, nombre):
            self.origenes[:] = esperado
        self.c.sql.side_effect = aplicar
        cronos.origenes(self.c, login)
        sentencia = self.c.sql.call_args.args[0].decode()
        self.assertEqual(sentencia.count("('" + login + "'"), 8)
        self.assertIn('SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;', sentencia)
        cronos.origenes(self.c, login)
        self.c.sql.assert_called_once()

    def test_origen_divergente_no_se_sobrescribe(self):
        self.origenes[:] = ['otro|vec_cronos_v1.saldo_propio.consultar.v1|x|vec-server|interna_corporativa']
        with self.assertRaisesRegex(ValueError, 'divergente'):
            cronos.origenes(self.c, 'vcl_j_cronos_0123456789ab_137')
        self.c.sql.assert_not_called()

    def test_login_no_admitido(self):
        with self.assertRaisesRegex(ValueError, 'inválido'):
            cronos.origenes(self.c, "vcl_x'; DROP ROLE postgres; --")

    def test_semilla_declarada_y_misma_referencia_al_repetir(self):
        primero = cronos.persona_empleada(self.c)
        sentencia = self.c.sql.call_args.args[0].decode()
        self.assertIn("SET LOCAL ROLE vec_contexto_actor_v1_propietario;", sentencia)
        self.assertIn("SET LOCAL ROLE vec_personal_propietario;", sentencia)
        self.assertIn("SET LOCAL ROLE vec_cronos_v1_propietario;", sentencia)
        self.assertIn("set_config('vec.cronos.empleado_ref'", sentencia)
        self.assertIn("'rol:cronos_empleado_sintetico:v1'", sentencia)
        self.assertIn("'clave','empleado_ref'", sentencia)
        self.assertNotIn('emp_cae2338320061380b9ae00c5f6ab1fed', sentencia.split('vec_autorizacion_propietario')[1])
        semilla = json.loads((self.c.carpeta / 'cronos/semilla.json').read_text())
        self.assertFalse(semilla['gobierno_real_acreditado'])
        self.assertEqual(cronos.persona_empleada(self.c), primero)
        self.assertEqual(primero['cuenta_ref'], CUENTA)
        self.assertEqual(primero['certificado_sha256'], 'a' * 64)

    def test_identidad_incoherente_sin_efectos(self):
        self._json(self.material / 'identidad/tecnico-rrhh-adicional.json', {
            'sujeto': 'otra', 'cuenta_ref': CUENTA, 'persona_ref': PERSONA, 'perfil_ref': PERFIL_ORIGEN})
        with self.assertRaisesRegex(ValueError, 'incoherente'):
            cronos.persona_empleada(self.c)
        self.c.sql.assert_not_called()

    def test_configuracion_motivo_denegado_distinto_y_privada(self):
        empleado = {'certificado_sha256': 'a' * 64, 'sujeto': 's', 'cuenta_ref': CUENTA,
                    'perfil_ref': 'prf_' + '5' * 32}
        dsns = {campo: 'postgresql://u:p@localhost:1/postgres' for campo in cronos.GRUPOS}
        archivo = cronos.configuracion(self.c, empleado, dsns)
        datos = json.loads(archivo.read_text())
        self.assertEqual(archivo.stat().st_mode & 0o777, 0o600)
        denegado = datos['auditoria_intentos']['motivo_denegado']['entrada_clave']
        self.assertEqual(denegado, cronos.MOTIVO_DENEGADO)
        self.assertNotIn(denegado, {m['entrada_clave'] for m in datos['motivos'].values()})
        self.assertEqual(set(datos['motivos']), {'saldo', 'marcaje', 'disponibilidad', 'recuperacion',
                                                 'movimientos', 'correccion', 'permisos', 'permiso'})

    def test_catalogo_ausente_detiene_antes_de_efectos(self):
        self.c.consultar.side_effect = lambda q: SID if 'pg_control_system' in q else ''
        with self.assertRaisesRegex(ValueError, 'CRN18'):
            cronos.preparar(self.c)
        self.c.sql.assert_not_called()

    def test_copia_ajena_detiene_antes_de_efectos(self):
        self.c.consultar.side_effect = lambda q: '98765432109876543'
        with self.assertRaisesRegex(ValueError, 'identificador PostgreSQL'):
            cronos.preparar(self.c)
        self.c.sql.assert_not_called()


if __name__ == '__main__':
    unittest.main()
