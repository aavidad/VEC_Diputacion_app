"""Synthetic, pure v2 fixtures and adversarial contract checks; no runtime access."""
from copy import deepcopy
import json
import unittest

try:
    from . import clon_cas_podman_contract as contract
except ImportError:
    import clon_cas_podman_contract as contract

H = 'a' * 64
T = '2026-10-01T00:00:00Z'


def request():
    """Fresh synthetic request for other writers' fixture-only tests."""
    mounts = {
        'anonymous': {'type': 'volume', 'name': 'b' * 64,
                      'source': '/fixture/storage/volumes/anonymous/_data',
                      'destination': '/var/lib/postgresql', 'driver': 'local',
                      'anonymous': True, 'rw': True, 'dev': 10, 'ino': 101,
                      'tree_sha256': H},
        'pgdata': {'type': 'bind', 'source': '/fixture/h6-pgdata',
                   'destination': '/var/lib/postgresql/18/docker',
                   'rw': True, 'dev': 20, 'ino': 201}}
    maps = [{'container_id': 0, 'host_id': 1000, 'size': 1},
            {'container_id': 1, 'host_id': 100000, 'size': 65536}]
    normalizer_bundle = {'h6_comun.sh': '3' * 64, 'h6_normalizar_pg_dump.py': '4' * 64}
    return {
        'version': 2, 'kind': contract.REQUEST_KIND, 'nonce': H,
        'server_name': 'postgres.fixture',
        'pgid': {'pg_container_id': 'c' * 64, 'pg_image': 'postgres:18.4',
                 'pg_image_id': 'sha256:' + 'd' * 64,
                 'pg_repo_digest': 'localhost/fixture-postgres@sha256:' + 'e' * 64,
                 'pgdata_bind_source': '/fixture/h6-pgdata',
                 'pgdata_bind_destination': '/var/lib/postgresql/18/docker',
                 'system_identifier': '10000000001', 'database_name': 'postgres', 'database_oid': 5},
        'pins': {k: contract.digest(contract.canonical(normalizer_bundle)) if k == 'normalizer_sha256'
                 else H for k in contract.PIN_FIELDS},
        'normalizer_bundle': normalizer_bundle,
        'users': {'identidad': 'vec_identidad_fixture', 'contexto': 'vec_contexto_fixture',
                  'autorizacion': contract.AUTH_USER},
        'tls_hashes': {k: H for k in {'ca.pem'} | {c + ext for c in contract.CHANNELS
                                                  for ext in ('.crt', '.key')}},
        'targets': {p: {'cuenta_ref': 'cta_' + x * 16, 'persona_ref': 'per_' + x * 16,
                        'perfil_ref': 'prf_' + x * 16,
                        'provision_ref': ('pce_' if p == 'candidato' else 'pue_') + x * 16}
                    for p, x in (('candidato', 'x'), ('usuarios', 'y'))},
        'runtime': {'profile': contract.PROFILE, 'rootless': True, 'owner_uid': 1000,
                    'owner_gid': 1000, 'uid_map': deepcopy(maps), 'gid_map': deepcopy(maps),
                    'postgres_process': {'pid': 100, 'StartedAt': T, 'starttick': 1234,
                                         'dev': 4, 'ino': 123},
                    'network_namespace': {'dev': 4, 'ino': 123},
                    'postgres_mount_namespace': {'dev': 4, 'ino': 124}, 'mounts': mounts,
                    'mount_inventory_sha256': contract.digest(contract.canonical(mounts))},
        'helper': {'image_id': 'sha256:' + 'f' * 64,
                   'repo_digest': 'localhost/fixture-helper@sha256:' + '1' * 64,
                   'code_sha256': H, 'uid': 10002, 'gid': 10002},
        'receipt_bindings': {'receipt_sha256': {k: H for k in ('p', 'a', 'intent', 'l')},
                             'p_intent_before_effect_sha256': '2' * 64,
                             'intent_before_effect_sha256': H, 'clonado_sha256': H,
                             'sql_func_sha256': H, 'a_physical_binding_sha256': H,
                             'post': {k: H for k in contract.STATE_FIELDS}}}


def sql_observation(req, initial_sessions, stage, state=None):
    """Synthetic external-observer fixture, without reading any SQL state."""
    return {'stage': stage, 'nonce': req['nonce'],
            'request_sha256': contract.digest(contract.canonical(req)),
            'sessions_sha256': contract.digest(contract.canonical(initial_sessions)),
            'normalizer_sha256': req['pins']['normalizer_sha256'],
            'normalizer_bundle': deepcopy(req['normalizer_bundle']),
            **{k: req['pgid'][k] for k in ('pg_container_id', 'pg_image_id',
                                         'system_identifier', 'database_oid')},
            **{k: deepcopy(req['runtime'][k]) for k in
               ('postgres_process', 'network_namespace', 'mount_inventory_sha256')},
            **deepcopy(req['receipt_bindings']['post'] if state is None else state)}


def result(req=None):
    """Fresh helper result, synthetically bound to req; never a real receipt."""
    req = deepcopy(request() if req is None else req)
    value = deepcopy(req)
    value['kind'] = contract.RESULT_KIND
    value.update({'cas_sessions_bound': True, 'cas_applied': False, 'rollback_confirmed': True,
                  'readings_sha256': [H, H], 'observations': {
                      k: {'lectura_1': T, 'lectura_2': '2026-10-01T00:00:01Z'}
                      for k in ('candidato', 'motivos')},
                  'preimages': {'candidato': {'revision_control_rol': 0, 'huella_control_rol': '',
                                               'version_asignacion': 0, 'huella_asignacion': '',
                                               'version_contexto': 0, 'huella_contexto': '',
                                               'secuencia_motivos': 0},
                                'usuarios': {'version_contexto': 0, 'huella_contexto': ''}},
                  'sessions': {}})
    for index, channel in enumerate(contract.CHANNELS):
        role = contract.ROLES[channel]
        row = {'session_user': value['users'][channel],
               'current_user': value['users'][channel] if role == 'none' else role,
               'role': role, 'transaction_read_only': 'on', 'transaction_isolation': 'read committed',
               'backend_pid': 200 + index, 'database_name': 'postgres', 'database_oid': 5,
               'system_identifier': value['pgid']['system_identifier'],
               'client_addr': '127.0.0.1', 'server_addr': '127.0.0.1', 'server_port': 5432,
               'login_safe': True, 'ssl': True, 'tls_version': 'TLSv1.3', 'client_certificate': True}
        value['sessions'][channel] = {'before': deepcopy(row), 'after': deepcopy(row)}
    initial = {channel: pair['before'] for channel, pair in value['sessions'].items()}
    value['sql_postimage_observations'] = {
        stage: sql_observation(req, initial, stage) for stage in ('before', 'after')}
    value['postimage'] = {k: value['sql_postimage_observations']['after'][k]
                         for k in contract.STATE_FIELDS}
    return value


def pending(req=None):
    req = request() if req is None else deepcopy(req)
    return {'version': 2, 'kind': contract.PENDING_KIND, 'request': req,
            'request_sha256': contract.digest(contract.canonical(req))}


def measurement(req=None):
    req = request() if req is None else deepcopy(req)
    physical = {k: deepcopy(req['runtime'][k]) for k in contract.PHYSICAL_FIELDS
                if k not in ('helper_process', 'helper_mount_namespace')}
    physical.update({'helper_process': {'pid': 101, 'StartedAt': T, 'starttick': 5678,
                                       'dev': 4, 'ino': 123},
                     'helper_mount_namespace': {'dev': 4, 'ino': 125}})
    helper_result = result(req)
    raw = json.dumps(helper_result, indent=2).encode()
    return {'version': 2, 'kind': contract.MEASUREMENT_KIND,
            'request_sha256': contract.digest(contract.canonical(req)), 'result': helper_result,
            'result_bytes_sha256': contract.digest(raw),
            'result_canonical_sha256': contract.digest(contract.canonical(helper_result)),
            'physical': {'before': deepcopy(physical), 'after': deepcopy(physical)}}


class StrictJSONTests(unittest.TestCase):
    def test_reject_ambiguous_json(self):
        samples = [b'{"version":2,"version":2}', b'{"outer":{"a":1,"a":2}}',
                   b'{"x":NaN}', b'{"x":Infinity}', b'{"x":-Infinity}',
                   b'{"x":1e999}', b'{"x":2.0}', b'[]', b'null', b'{}{}',
                   b'{"x":"\xff"}', b'{"x":"\\ud800"}', b'']
        for raw in samples:
            with self.subTest(raw=raw), self.assertRaises(contract.Refused):
                contract.decode(raw)

    def test_limits_and_python_types(self):
        for raw in ('{}', bytearray(b'{}'), b' ' * (contract.MAX_BYTES + 1),
                    ('{"x":' * 30 + '{}' + '}' * 30).encode()):
            with self.subTest(type=type(raw)), self.assertRaises(contract.Refused):
                contract.decode(raw)
        for value in ({'x': float('nan')}, {'x': 1.0}, {'x': (1, 2)}, {1: 2},
                      {'x': 2**64}, {'x': [0] * 129}):
            with self.subTest(value=value), self.assertRaises(contract.Refused):
                contract.canonical(value)

    def test_error_never_contains_private_input(self):
        with self.assertRaises(contract.Refused) as caught:
            contract.decode(b'{"private-material":NaN}')
        self.assertNotIn('private-material', str(caught.exception))


class ContractTests(unittest.TestCase):
    def rejects(self, value, path, replacement):
        changed = deepcopy(value)
        cursor = changed
        for key in path[:-1]:
            cursor = cursor[key]
        cursor[path[-1]] = replacement
        with self.subTest(path=path, replacement=replacement), self.assertRaises(contract.Refused):
            contract.validate(changed)

    def test_four_kinds_and_detached_values(self):
        for original in (request(), result(), pending(), measurement()):
            accepted = contract.validate(original)
            self.assertEqual(accepted, original)
            self.assertIsNot(accepted, original)
        req = request()
        checked = contract.validate_request(req)
        req['runtime']['postgres_process']['pid'] = 777
        self.assertEqual(checked['runtime']['postgres_process']['pid'], 100)
        self.assertEqual(contract.bind_result(checked, result(checked)), result(checked))

    def test_extra_fields_at_every_object(self):
        def walk(value, path=()):
            if isinstance(value, dict):
                yield path
                for key, item in value.items():
                    yield from walk(item, path + (key,))
            elif isinstance(value, list):
                for index, item in enumerate(value):
                    yield from walk(item, path + (index,))
        for original in (request(), result(), pending(), measurement()):
            for path in walk(original):
                changed = deepcopy(original)
                cursor = changed
                for key in path:
                    cursor = cursor[key]
                cursor['approved'] = True
                with self.subTest(kind=original['kind'], path=path), self.assertRaises(contract.Refused):
                    contract.validate(changed)

    def test_missing_fields_at_every_object(self):
        def walk(value, path=()):
            if isinstance(value, dict):
                for key in value:
                    yield path, key
                for key, item in value.items():
                    yield from walk(item, path + (key,))
            elif isinstance(value, list):
                for index, item in enumerate(value):
                    yield from walk(item, path + (index,))
        for original in (request(), result(), pending(), measurement()):
            for path, key in walk(original):
                changed = deepcopy(original)
                cursor = changed
                for component in path:
                    cursor = cursor[component]
                del cursor[key]
                with self.subTest(kind=original['kind'], path=path, key=key), self.assertRaises(contract.Refused):
                    contract.validate(changed)

    def test_bool_int_substitution_everywhere(self):
        def walk(value, path=()):
            if type(value) in (bool, int):
                yield path, value
            elif isinstance(value, dict):
                for key, item in value.items():
                    yield from walk(item, path + (key,))
            elif isinstance(value, list):
                for index, item in enumerate(value):
                    yield from walk(item, path + (index,))
        for original in (request(), result(), pending(), measurement()):
            for path, value in walk(original):
                self.rejects(original, path, int(value) if type(value) is bool else True)

    def test_physical_profile_mounts_and_maps(self):
        req = request()
        for path, value in [(('runtime', 'profile'), 'docker'), (('runtime', 'rootless'), False),
                            (('runtime', 'owner_uid'), 0), (('helper', 'uid'), 0),
                            (('pgid', 'pg_image_id'), req['pgid']['pg_repo_digest']),
                            (('pgid', 'pg_repo_digest'), req['pgid']['pg_image_id']),
                            (('runtime', 'postgres_process', 'ino'), 999),
                            (('runtime', 'mounts', 'anonymous', 'anonymous'), False),
                            (('runtime', 'mounts', 'anonymous', 'rw'), False),
                            (('runtime', 'mounts', 'pgdata', 'rw'), False),
                            (('runtime', 'mounts', 'pgdata', 'source'), '/other/data'),
                            (('runtime', 'mounts', 'pgdata', 'destination'), '/other/data'),
                            (('pgid', 'pgdata_bind_source'), '/fixture/../escape'),
                            (('runtime', 'mount_inventory_sha256'), '0' * 64),
                            (('runtime', 'uid_map', 1, 'container_id'), 0),
                            (('runtime', 'gid_map', 1, 'host_id'), 1000),
                            (('runtime', 'uid_map', 1, 'size'), 1),
                            (('runtime', 'uid_map', 1, 'host_id'), 2**32 - 1)]:
            self.rejects(req, path, value)
        self.assertEqual(contract.mapped_id(req['runtime']['uid_map'], 10002), 110001)

    def test_original_receipt_intentions_and_postimage(self):
        for path in [('receipt_bindings', 'receipt_sha256', 'a'),
                     ('receipt_bindings', 'receipt_sha256', 'l'),
                     ('receipt_bindings', 'receipt_sha256', 'intent')]:
            self.rejects(request(), path, '0' * 64)
        self.rejects(result(), ('postimage', 'schema_sha256'), '0' * 64)
        # P's original intention is independent of L's; not inferred from A or L.
        self.assertNotEqual(request()['receipt_bindings']['p_intent_before_effect_sha256'],
                            request()['receipt_bindings']['intent_before_effect_sha256'])

    def test_sessions_observations_and_rollback(self):
        res = result()
        for path, value in [(('rollback_confirmed',), False), (('cas_applied',), True),
                            (('cas_sessions_bound',), False), (('readings_sha256', 1), '0' * 64),
                            (('sessions', 'identidad', 'before', 'current_user'), 'postgres'),
                            (('sessions', 'contexto', 'after', 'backend_pid'), 999),
                            (('sessions', 'autorizacion', 'before', 'ssl'), False),
                            (('sessions', 'autorizacion', 'before', 'tls_version'), 'TLSv1.2'),
                            (('observations', 'candidato', 'lectura_1'), '2026-10-01T00:00:02Z'),
                            (('observations', 'motivos', 'lectura_2'), '2026-10-01T00:00:01+00:00'),
                            (('preimages', 'usuarios', 'version_contexto'), 1),
                            (('preimages', 'candidato', 'huella_control_rol'), H)]:
            self.rejects(res, path, value)
        for side in ('before', 'after'):
            res['sessions']['contexto'][side]['backend_pid'] = 200
        with self.assertRaises(contract.Refused):
            contract.validate_result(res)

    def test_submicrosecond_time_order_is_preserved(self):
        res = result()
        res['observations']['candidato'] = {'lectura_1': '2026-10-01T00:00:01.000000002Z',
                                          'lectura_2': '2026-10-01T00:00:01.000000001Z'}
        with self.assertRaises(contract.Refused):
            contract.validate_result(res)

    def test_request_binding_is_complete(self):
        req = request()
        res = result(req)
        for field in ('nonce', 'server_name'):
            changed = deepcopy(res)
            changed[field] = 'b' * 64 if field == 'nonce' else 'other.fixture'
            with self.subTest(field=field), self.assertRaises(contract.Refused):
                contract.bind_result(req, changed)
        self.rejects(pending(), ('request_sha256',), '0' * 64)
        self.rejects(measurement(), ('request_sha256',), '0' * 64)

    def test_original_helper_bytes_are_separate_from_canonical_hash(self):
        value = measurement()
        original = json.dumps(value['result'], indent=2).encode()
        self.assertNotEqual(value['result_bytes_sha256'], value['result_canonical_sha256'])
        contract.validate_measurement(value, result_raw=original)
        for raw in (contract.canonical(value['result']), original + b' ', b'{}'):
            with self.subTest(raw_size=len(raw)), self.assertRaises(contract.Refused):
                contract.validate_measurement(value, result_raw=raw)
        self.rejects(value, ('result_canonical_sha256',), '0' * 64)

    def test_named_physical_identity_drift(self):
        value = measurement()
        for field in ('pid', 'StartedAt', 'starttick', 'dev', 'ino'):
            original = value['physical']['after']['helper_process'][field]
            replacement = '2026-10-01T00:00:01Z' if field == 'StartedAt' else original + 1
            self.rejects(value, ('physical', 'after', 'helper_process', field), replacement)
        for path, replacement in [(('physical', 'before', 'helper_process', 'pid'), 100),
                                  (('physical', 'before', 'helper_mount_namespace'), {'dev': 4, 'ino': 124}),
                                  (('physical', 'before', 'uid_map', 1, 'host_id'), 200000),
                                  (('physical', 'before', 'mounts', 'pgdata', 'ino'), 999)]:
            self.rejects(value, path, replacement)

    def test_sql_state_changes_even_with_matching_before_after(self):
        # Schema body/ACL, role inventory and database ACL each have their own
        # H6 state hash; a consistently altered pair cannot match original L.
        for field in contract.STATE_FIELDS:
            value = result()
            for stage in ('before', 'after'):
                value['sql_postimage_observations'][stage][field] = '0' * 64
            value['postimage'][field] = '0' * 64
            with self.subTest(field=field), self.assertRaises(contract.Refused):
                contract.validate_result(value, request())

    def test_sql_observation_nonce_session_request_and_identity_binding(self):
        value = result()
        for field, changed in [('nonce', '0' * 64), ('request_sha256', '0' * 64),
                               ('sessions_sha256', '0' * 64), ('pg_container_id', '0' * 64),
                               ('pg_image_id', 'sha256:' + '0' * 64),
                               ('system_identifier', '999999999'), ('database_oid', 99),
                               ('normalizer_sha256', '0' * 64),
                               ('normalizer_bundle', {'h6_comun.sh': '0' * 64,
                                                      'h6_normalizar_pg_dump.py': '4' * 64})]:
            for stage in ('before', 'after'):
                self.rejects(value, ('sql_postimage_observations', stage, field), changed)

    def test_sql_stage_order_missing_and_swap(self):
        value = result()
        self.rejects(value, ('sql_postimage_observations', 'before', 'stage'), 'after')
        self.rejects(value, ('sql_postimage_observations', 'after', 'stage'), 'before')
        changed = deepcopy(value)
        changed['sql_postimage_observations'] = {
            'before': value['sql_postimage_observations']['after'],
            'after': value['sql_postimage_observations']['before']}
        with self.assertRaises(contract.Refused):
            contract.validate_result(changed)
        req = request()
        initial = {channel: pair['before'] for channel, pair in value['sessions'].items()}
        for stage in ('unknown', None, True, 1):
            with self.subTest(stage=stage), self.assertRaises(contract.Refused):
                contract.validate_sql_postimage_observation(
                    value['sql_postimage_observations']['before'], req, initial, stage)

    def test_normalizer_bundle_is_pinned_before_observation(self):
        req = request()
        self.rejects(req, ('normalizer_bundle', 'h6_comun.sh'), '0' * 64)
        self.rejects(req, ('normalizer_bundle', 'h6_normalizar_pg_dump.py'), '0' * 64)
        self.rejects(req, ('pins', 'normalizer_sha256'), '0' * 64)
        value = result(req)
        initial = {channel: pair['before'] for channel, pair in value['sessions'].items()}
        observation = value['sql_postimage_observations']['before']
        self.assertEqual(contract.validate_sql_postimage_observation(observation, req, initial, 'before'),
                         observation)
        # A digest of the entire paired-session envelope is not the initial
        # retained-session map required by the fixed barrier.
        forged = deepcopy(observation)
        forged['sessions_sha256'] = contract.digest(contract.canonical(value['sessions']))
        with self.assertRaises(contract.Refused):
            contract.validate_sql_postimage_observation(forged, req, initial, 'before')

    def test_sql_observations_cannot_rewrite_independent_request(self):
        req = request()
        forged_request = deepcopy(req)
        forged_request['receipt_bindings']['post']['schema_sha256'] = '0' * 64
        # This altered JSON can pass internal structural consistency, but
        # binding to the original independently pinned request rejects it.
        forged = result(forged_request)
        with self.assertRaises(contract.Refused):
            contract.bind_result(req, forged)

    def test_sql_observation_process_namespace_and_mount_binding(self):
        value = result()
        for stage in ('before', 'after'):
            for field in contract.PROCESS_FIELDS:
                original = value['sql_postimage_observations'][stage]['postgres_process'][field]
                changed = '2026-10-01T00:00:01Z' if field == 'StartedAt' else original + 1
                self.rejects(value, ('sql_postimage_observations', stage, 'postgres_process', field),
                             changed)
            for field in ('dev', 'ino'):
                self.rejects(value, ('sql_postimage_observations', stage, 'network_namespace', field), 999)
            self.rejects(value, ('sql_postimage_observations', stage, 'mount_inventory_sha256'), '0' * 64)

    def test_body_and_acl_changes_preserve_names_but_change_h6_hash(self):
        # Synthetic normalized pg_dump-like bytes, never passed to PostgreSQL.
        original = b'CREATE FUNCTION f() RETURNS int AS $$ SELECT 1 $$ LANGUAGE sql;\nGRANT EXECUTE ON FUNCTION f() TO reader;\n'
        for changed in (original.replace(b'SELECT 1', b'SELECT 2'),
                        original.replace(b'TO reader', b'TO PUBLIC')):
            req = request()
            req['receipt_bindings']['post']['schema_sha256'] = contract.digest(original)
            value = result(req)
            for stage in ('before', 'after'):
                value['sql_postimage_observations'][stage]['schema_sha256'] = contract.digest(changed)
            value['postimage']['schema_sha256'] = contract.digest(changed)
            with self.subTest(changed=changed), self.assertRaises(contract.Refused):
                contract.bind_result(req, value)


if __name__ == '__main__':
    unittest.main()
