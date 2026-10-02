"""Fixtures sintéticos: ninguna prueba conecta a PostgreSQL ni acredita H6."""
import copy
import unittest
from decimal import Decimal
import clon_preimagenes_nominales as p

SHA = 'a' * 64
TIME1, TIME2 = '2026-10-01T10:00:00Z', '2026-10-01T10:00:01Z'
TARGET = {'cuenta_ref': 'cta_' + 'a' * 32, 'persona_ref': 'per_' + 'b' * 32,
          'perfil_ref': 'prf_' + 'c' * 32, 'provision_ref': 'pce_' + 'd' * 32}
ALIAS = {'cuenta_ref': TARGET['cuenta_ref'], 'esquema': 'vec.identidad.hmac-sha256.v1',
         'dominio_ref': 'idh_fixture', 'clave_id': 'fixture.key', 'clave_version': 1,
         'cuenta_id_hmac': 'd' * 64, 'sujeto_id_hmac': 'e' * 64}


def auth(when=TIME1, present=False):
    role = {'estado_observacion': 'ausente', 'version': None, 'huella_sha256': None,
            'estado': None, 'versiones': 0, 'version_maxima': None}
    control = {'estado_observacion': 'ausente', 'revision': None, 'huella_sha256': None,
               'estado': None, 'revisiones': 0, 'revision_maxima': None}
    assignment = {'estado_observacion': 'ausente', 'versiones': 0, 'version_maxima': None,
                  'asignacion_ref': None, 'version': None, 'huella_sha256': None, 'estado': None,
                  'vigente_desde': None, 'vigente_hasta': None}
    if present:
        role.update(estado_observacion='publicada', version=1, huella_sha256=SHA,
                    estado='publicada', versiones=1, version_maxima=1)
        control.update(estado_observacion='habilitada', revision=1, huella_sha256=SHA,
                       estado='habilitada', revisiones=1, revision_maxima=1)
        assignment.update(estado_observacion='activa', versiones=1, version_maxima=1,
                          asignacion_ref='asignacion:fixture', version=1, huella_sha256=SHA,
                          estado='activa', vigente_desde='2026-09-01T00:00:00Z',
                          vigente_hasta='2026-11-01T00:00:00Z')
    return {'contrato': 'preimagen_candidato_externo_ro_v1', 'observada_en': when,
            'rol_id': p.ROLE_ID, 'version_rol_ref': p.ROLE_REF,
            'principal_id': TARGET['persona_ref'], 'perfil_activo_ref': TARGET['perfil_ref'],
            'rol': role, 'control_rol': control, 'asignacion': assignment}


def motives(when=TIME1, present=False):
    cp = {'controles': 1, 'ultima_secuencia': 0, 'ultimo_evento_ref': None,
          'ultima_huella_evento_sha256': None, 'actualizado_en': '2026-09-01T00:00:00Z',
          'eventos': 0, 'secuencia_maxima': 0, 'coherente': True}
    catalogs = [{'catalogo_id': key, 'versiones': 0, 'observaciones': []} for key in sorted(p.CATALOGS)]
    if present:
        cp.update(ultima_secuencia=3, eventos=3, secuencia_maxima=3,
                  ultimo_evento_ref='evento:fixture:3', ultima_huella_evento_sha256=SHA)
        for i, cat in enumerate(catalogs, 1):
            cat['versiones'] = 1
            cat['observaciones'] = [{'version': 1, 'huella_publicada_sha256': SHA,
                'publicado_en': '2026-09-01T00:00:00Z', 'evento_publicacion_ref': f'evento:fixture:{i}',
                'secuencia_publicacion': i, 'huella_evento_publicacion_sha256': SHA,
                'entradas': 1, 'entradas_vigentes': 1, 'retirada': False,
                'huella_retirada_sha256': None, 'retirado_en': None, 'evento_retirada_ref': None,
                'secuencia_retirada': None, 'huella_evento_retirada_sha256': None, 'coherente': True}]
    return {'contrato': 'checkpoint_motivos_candidato_externo_ro_v1', 'observada_en': when,
            'checkpoint': cp, 'catalogos': catalogs}


class Cursor:
    def __init__(self, result):
        self.result = result
        self.calls = []
    def execute(self, sql, parameters=()):
        self.calls.append((sql, parameters))
    def fetchmany(self, size):
        return self.result[:size]


class PreimagesTests(unittest.TestCase):
    def test_absence_and_present(self):
        self.assertEqual(p.authorization(auth(), TARGET)['version_asignacion'], 0)
        self.assertEqual(p.authorization(auth(present=True), TARGET)['revision_control_rol'], 1)
        self.assertEqual(p.motives(motives()), 0)
        self.assertEqual(p.motives(motives(present=True)), 3)

    def test_states_and_temporal_barriers(self):
        for channel, field, values in (('rol', 'estado_observacion', ('retirada', 'inconsistente', None)),
              ('control_rol', 'estado_observacion', ('revocada', 'inconsistente', None)),
              ('asignacion', 'estado_observacion', ('revocada', 'no_vigente', 'inconsistente', None)),
              ('asignacion', 'vigente_hasta', (TIME1, 'infinity', None)),
              ('asignacion', 'version_maxima', (2, True)),
              ('rol', 'versiones', (True, 0))):
            for bad in values:
                with self.subTest(channel=channel, field=field, bad=bad):
                    value = auth(present=True)
                    value[channel][field] = bad
                    with self.assertRaises(p.Refused):
                        p.authorization(value, TARGET)

    def test_absence_with_orphan_history_refused(self):
        value = auth()
        value['asignacion']['versiones'] = 1
        with self.assertRaises(p.Refused):
            p.authorization(value, TARGET)

    def test_catalog_null_coherence_and_revocation(self):
        for field, bad in (('coherente', None), ('coherente', False), ('retirada', True),
                            ('entradas_vigentes', 0), ('secuencia_publicacion', 4)):
            value = motives(present=True)
            value['catalogos'][0]['observaciones'][0][field] = bad
            with self.assertRaises(p.Refused):
                p.motives(value)
        value = motives()
        value['checkpoint']['coherente'] = None
        with self.assertRaises(p.Refused):
            p.motives(value)

    def test_only_observation_times_are_ignored(self):
        reading = {'autorizacion': auth(present=True), 'motivos': motives(present=True),
                   'identidad': {'alias': ALIAS}}
        later = copy.deepcopy(reading)
        later['autorizacion']['observada_en'] = later['motivos']['observada_en'] = TIME2
        self.assertEqual(p.semantic(reading), p.semantic(later))
        later['autorizacion']['asignacion']['vigente_hasta'] = '2026-12-01T00:00:00Z'
        self.assertNotEqual(p.semantic(reading), p.semantic(later))
        self.assertEqual(reading['autorizacion']['observada_en'], TIME1)

    def test_identity_alias_collision_partial_and_inactive(self):
        class IdentityCursor(Cursor):
            def execute(self, sql, parameters=()):
                self.result = [(TARGET['cuenta_ref'], 1, state)] if sql == p.ACCOUNT_SQL else [tuple(alias.values())]
        alias = copy.deepcopy(ALIAS)
        state = 'activa'
        self.assertEqual(p.identity(IdentityCursor([]), TARGET, ALIAS)['account'][2], 'activa')
        alias['clave_version'] = 2
        with self.assertRaises(p.Refused):
            p.identity(IdentityCursor([]), TARGET, ALIAS)
        state = 'inactiva'
        with self.assertRaises(p.Refused):
            p.identity(IdentityCursor([]), TARGET, ALIAS)

    def test_context_decimal_and_cardinality(self):
        cursor = Cursor([(Decimal(1), SHA)])
        self.assertEqual(p.context(cursor, TARGET['provision_ref'])['version_contexto'], 1)
        self.assertEqual(cursor.calls[0][1], (TARGET['provision_ref'],))
        for rows in ([(Decimal('1.2'), SHA)], [(True, SHA)], [(1, None)], [(1, SHA), (2, SHA)]):
            with self.assertRaises(p.Refused):
                p.context(Cursor(rows), TARGET['provision_ref'])


if __name__ == '__main__':
    unittest.main()
