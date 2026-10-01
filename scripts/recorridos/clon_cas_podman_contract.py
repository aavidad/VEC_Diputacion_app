"""Pure CAS Podman v2 wire contract; validation never grants execution authority.

API: decode(bytes) rejects duplicate JSON keys and non-integer numbers;
canonical(JSON object) returns deterministic UTF-8 bytes; digest(bytes) hashes
original bytes. validate_* returns a detached validated dict. validate dispatches
by kind. bind_result(request, result) also binds every request field. Consumers
must acquire pins independently and keep their execution gates outside this
module. No paths are opened, commands run, receipts acquired or SQL executed.

PG image ID (configuration digest), repository digest (manifest digest) and
container ID are distinct fields. Process dev/ino identifies /proc/PID/ns/net;
mount namespace dev/ino is separate. Mount inventory hashes canonical mounts,
a two-entry projection whose completeness the acquisition layer must prove.
Anonymous tree_sha256 pins independently acquired directory-tree evidence.
receipt_bindings pins original P/A/L receipt bytes and P/L intentions; A has no
invented intention. post is the independently pinned SQL state after L.
SQL observations bracket the CAS readings on the retained nominal sessions.
Their state is acquired externally using the independently pinned H6 bundle:
normalizer_bundle contains original h6_comun.sh and h6_normalizar_pg_dump.py
hashes, and pins.normalizer_sha256 is its canonical aggregate hash. The shell
defines dump commands and datacl SQL; the Python file alone is insufficient.
validate_sql_postimage_observation checks binding and expected state only;
even matching observations do not prove trusted acquisition or grant authority.
Measurement result_bytes_sha256 pins the original helper output separately from
result_canonical_sha256. Pass result_raw to validate_measurement to verify the
original-byte pin; without it validation proves only structure and consistency.
"""
from datetime import datetime
import hashlib
import json
import re

VERSION = 2
REQUEST_KIND = 'cas_podman_helper_request_v2'
PENDING_KIND = 'cas_podman_measurement_pending_v2'
RESULT_KIND = 'cas_podman_helper_result_v2'
MEASUREMENT_KIND = 'cas_podman_namespace_measurement_v2'
MAX_BYTES = 1 << 20
PROFILE = 'podman_rootless_h6_anonymous_rw_pgdata_bind_rw'
CHANNELS = ('identidad', 'contexto', 'autorizacion')
AUTH_USER = 'vec_externo_v3_fuente_autorizacion_desarrollo'
ROLES = {'identidad': 'vec_identidad_sesiones_v1_propietario',
         'contexto': 'vec_contexto_actor_v1_propietario', 'autorizacion': 'none'}
PIN_FIELDS = {'source_sha256', 'alias_sha256', 'restore_sha256', 'journal_sha256',
              'h1_records_sha256', 'aut26_receipt_sha256', 'login_receipt_sha256',
              'hba_sha256', 'acl_sha256', 'rls_sha256', 'normalizer_sha256'}
REQUEST_FIELDS = {'version', 'kind', 'nonce', 'server_name', 'pgid', 'pins',
                  'users', 'tls_hashes', 'targets', 'runtime', 'helper', 'receipt_bindings',
                  'normalizer_bundle'}
RESULT_FIELDS = REQUEST_FIELDS | {'sessions', 'cas_sessions_bound', 'cas_applied',
                                 'rollback_confirmed', 'readings_sha256',
                                 'observations', 'preimages', 'postimage',
                                 'sql_postimage_observations'}
SESSION_FIELDS = {'session_user', 'current_user', 'role', 'transaction_read_only',
                  'transaction_isolation', 'backend_pid', 'database_name',
                  'database_oid', 'system_identifier', 'client_addr', 'server_addr',
                  'server_port', 'login_safe', 'ssl', 'tls_version', 'client_certificate'}
STATE_FIELDS = {'schema_sha256', 'roles_sha256', 'datacl_sha256'}
NORMALIZER_FILES = {'h6_comun.sh', 'h6_normalizar_pg_dump.py'}
SQL_POSTIMAGE_FIELDS = STATE_FIELDS | {'stage', 'nonce', 'request_sha256', 'sessions_sha256',
                                      'pg_container_id', 'pg_image_id', 'system_identifier',
                                      'database_oid', 'normalizer_sha256', 'normalizer_bundle',
                                      'postgres_process', 'network_namespace',
                                      'mount_inventory_sha256'}
PROCESS_FIELDS = {'pid', 'StartedAt', 'starttick', 'dev', 'ino'}
PHYSICAL_FIELDS = {'postgres_process', 'helper_process', 'network_namespace',
                   'postgres_mount_namespace', 'helper_mount_namespace',
                   'uid_map', 'gid_map', 'mounts', 'mount_inventory_sha256'}


class Refused(RuntimeError):
    """Fixed code only: private values and provider errors never escape."""


def require(ok, code):
    if not ok:
        raise Refused(code)


def keys(value, fields, code='cas_v2_fields'):
    require(type(value) is dict and set(value) == set(fields), code)


def fingerprint(value):
    require(type(value) is str and re.fullmatch(r'[a-f0-9]{64}', value) is not None,
            'cas_v2_hash')
    return value


def number(value, maximum=2**64, minimum=0):
    require(type(value) is int and minimum <= value < maximum, 'cas_v2_integer')
    return value


def _json(value, depth=0):
    require(depth <= 24, 'cas_v2_depth')
    if type(value) is dict:
        require(len(value) <= 128 and all(type(k) is str for k in value), 'cas_v2_json_type')
        for k, v in value.items():
            _json(k, depth + 1)
            _json(v, depth + 1)
    elif type(value) is list:
        require(len(value) <= 128, 'cas_v2_array_limit')
        for item in value:
            _json(item, depth + 1)
    elif type(value) is str:
        require(len(value) <= 4096 and not any(0xD800 <= ord(c) <= 0xDFFF for c in value),
                'cas_v2_string')
    else:
        require(value is None or type(value) is bool or type(value) is int,
                'cas_v2_json_type')
        if type(value) is int:
            require(-2**64 < value < 2**64, 'cas_v2_integer')


def decode(raw):
    require(type(raw) is bytes and 0 < len(raw) <= MAX_BYTES, 'cas_v2_size')
    def unique(pairs):
        result = {}
        for field, value in pairs:
            require(field not in result, 'cas_v2_duplicate')
            result[field] = value
        return result
    def invalid(_):
        raise Refused('cas_v2_number')
    try:
        value = json.loads(raw.decode('utf-8'), object_pairs_hook=unique,
                           parse_constant=invalid, parse_float=invalid)
        require(type(value) is dict, 'cas_v2_object')
        _json(value)
        return value
    except (ValueError, UnicodeError, RecursionError):
        raise Refused('cas_v2_json') from None


def canonical(value):
    require(type(value) is dict, 'cas_v2_object')
    _json(value)
    raw = json.dumps(value, sort_keys=True, separators=(',', ':'),
                     ensure_ascii=True, allow_nan=False).encode('utf-8')
    require(len(raw) <= MAX_BYTES, 'cas_v2_size')
    return raw


def digest(raw):
    require(type(raw) is bytes and len(raw) <= MAX_BYTES, 'cas_v2_bytes')
    return hashlib.sha256(raw).hexdigest()


def instant(value):
    require(type(value) is str and re.fullmatch(
        r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z', value) is not None,
        'cas_v2_time')
    try:
        datetime.fromisoformat(value[:-1] + '+00:00')
    except ValueError:
        raise Refused('cas_v2_time') from None
    # Retain all nine fractional digits for ordering; datetime truncates to six.
    return value[:19] + '.' + value[20:-1].ljust(9, '0') if '.' in value else value[:19] + '.000000000'


def _path(value):
    require(type(value) is str and re.fullmatch(r'/[A-Za-z0-9_./-]{1,1023}', value) is not None
            and all(p not in ('', '.', '..') for p in value[1:].split('/')), 'cas_v2_path')


def _image_id(value):
    require(type(value) is str and re.fullmatch(r'sha256:[a-f0-9]{64}', value) is not None,
            'cas_v2_image_id')


def _repo_digest(value):
    require(type(value) is str and re.fullmatch(
        r'[a-z0-9][a-z0-9./:_-]{0,240}@sha256:[a-f0-9]{64}', value) is not None,
        'cas_v2_repo_digest')


def _namespace(value):
    keys(value, ('dev', 'ino'))
    number(value['dev'])
    number(value['ino'], minimum=1)


def _process(value):
    keys(value, PROCESS_FIELDS)
    number(value['pid'], 2**31, 1)
    instant(value['StartedAt'])
    number(value['starttick'], minimum=1)
    number(value['dev'])
    number(value['ino'], minimum=1)


def mapped_id(rows, inner):
    """Map an inner ID through validated rootless rows; never infer host identity."""
    number(inner, 2**32)
    require(type(rows) is list and 1 <= len(rows) <= 32, 'cas_v2_id_map')
    found = []
    for row in rows:
        keys(row, ('container_id', 'host_id', 'size'))
        start, host, size = row['container_id'], row['host_id'], row['size']
        number(start, 2**32)
        number(host, 2**32, 1)
        number(size, 2**32, 1)
        require(start + size <= 2**32 and host + size <= 2**32, 'cas_v2_id_map')
        if start <= inner < start + size:
            found.append(host + inner - start)
    for index, a in enumerate(rows):
        for b in rows[index + 1:]:
            for field in ('container_id', 'host_id'):
                require(a[field] + a['size'] <= b[field] or b[field] + b['size'] <= a[field],
                        'cas_v2_id_map_overlap')
    require(len(found) == 1, 'cas_v2_id_unmapped')
    return found[0]


def _state(value):
    keys(value, STATE_FIELDS)
    for pin in value.values():
        fingerprint(pin)


def _mounts(mounts, pg):
    keys(mounts, ('anonymous', 'pgdata'))
    parent, bind = mounts['anonymous'], mounts['pgdata']
    keys(parent, ('type', 'name', 'source', 'destination', 'driver', 'anonymous',
                  'rw', 'dev', 'ino', 'tree_sha256'))
    keys(bind, ('type', 'source', 'destination', 'rw', 'dev', 'ino'))
    require(parent['type'] == 'volume' and parent['driver'] == 'local'
            and parent['anonymous'] is True and parent['rw'] is True
            and parent['destination'] == '/var/lib/postgresql', 'cas_v2_parent_mount')
    fingerprint(parent['name'])
    fingerprint(parent['tree_sha256'])
    require(bind['type'] == 'bind' and bind['rw'] is True
            and bind['source'] == pg['pgdata_bind_source']
            and bind['destination'] == pg['pgdata_bind_destination']
            and bind['destination'].startswith(parent['destination'] + '/'), 'cas_v2_pgdata_mount')
    for mount in (parent, bind):
        _path(mount['source'])
        _path(mount['destination'])
        number(mount['dev'])
        number(mount['ino'], minimum=1)
    require(parent['source'] != bind['source']
            and not bind['source'].startswith(parent['source'] + '/')
            and not parent['source'].startswith(bind['source'] + '/'), 'cas_v2_mount_sources')
    require((parent['dev'], parent['ino']) != (bind['dev'], bind['ino']), 'cas_v2_mount_alias')


def _common(value):
    require(type(value['version']) is int and value['version'] == VERSION, 'cas_v2_version')
    fingerprint(value['nonce'])
    require(type(value['server_name']) is str and re.fullmatch(r'[a-z][a-z0-9.-]{0,120}',
            value['server_name']) is not None, 'cas_v2_server_name')
    pg = value['pgid']
    keys(pg, ('pg_container_id', 'pg_image', 'pg_image_id', 'pg_repo_digest',
              'pgdata_bind_source', 'pgdata_bind_destination', 'system_identifier',
              'database_name', 'database_oid'))
    fingerprint(pg['pg_container_id'])
    _image_id(pg['pg_image_id'])
    _repo_digest(pg['pg_repo_digest'])
    require(type(pg['pg_image']) is str and re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9./:@_-]{0,240}',
            pg['pg_image']) is not None, 'cas_v2_image_reference')
    require(type(pg['system_identifier']) is str and re.fullmatch(r'[1-9][0-9]{0,19}',
            pg['system_identifier']) is not None and int(pg['system_identifier']) < 2**64
            and pg['database_name'] == 'postgres', 'cas_v2_sql_identity')
    number(pg['database_oid'], 2**32, 1)
    _path(pg['pgdata_bind_source'])
    _path(pg['pgdata_bind_destination'])
    keys(value['pins'], PIN_FIELDS)
    for pin in value['pins'].values():
        fingerprint(pin)
    keys(value['normalizer_bundle'], NORMALIZER_FILES)
    for pin in value['normalizer_bundle'].values():
        fingerprint(pin)
    require(value['pins']['normalizer_sha256'] == digest(canonical(value['normalizer_bundle'])),
            'cas_v2_normalizer_bundle')
    keys(value['users'], CHANNELS)
    users = value['users']
    require(all(type(user) is str and re.fullmatch(r'vec_[a-z0-9_]{1,59}', user) is not None
                for user in users.values()) and len(set(users.values())) == 3
            and users['autorizacion'] == AUTH_USER
            and not any(user in ROLES.values() for user in users.values()), 'cas_v2_users')
    keys(value['tls_hashes'], {'ca.pem'} | {c + ext for c in CHANNELS for ext in ('.crt', '.key')})
    for pin in value['tls_hashes'].values():
        fingerprint(pin)
    keys(value['targets'], ('candidato', 'usuarios'))
    for population, target in value['targets'].items():
        keys(target, ('cuenta_ref', 'persona_ref', 'perfil_ref', 'provision_ref'))
        for field, prefix in (('cuenta_ref', 'cta'), ('persona_ref', 'per'), ('perfil_ref', 'prf'),
                              ('provision_ref', 'pce' if population == 'candidato' else 'pue')):
            require(type(target[field]) is str and re.fullmatch(prefix + r'_[A-Za-z0-9_-]{16,128}',
                    target[field]) is not None, 'cas_v2_target')
    c, u = value['targets']['candidato'], value['targets']['usuarios']
    require(c['cuenta_ref'] != u['cuenta_ref'] or (c['persona_ref'], c['perfil_ref']) ==
            (u['persona_ref'], u['perfil_ref']), 'cas_v2_shared_account')
    helper = value['helper']
    keys(helper, ('image_id', 'repo_digest', 'code_sha256', 'uid', 'gid'))
    _image_id(helper['image_id'])
    _repo_digest(helper['repo_digest'])
    fingerprint(helper['code_sha256'])
    require(type(helper['uid']) is int and type(helper['gid']) is int
            and helper['uid'] == helper['gid'] == 10002, 'cas_v2_helper_inner_ids')
    runtime = value['runtime']
    keys(runtime, ('profile', 'rootless', 'owner_uid', 'owner_gid', 'uid_map', 'gid_map',
                   'postgres_process', 'network_namespace', 'postgres_mount_namespace',
                   'mounts', 'mount_inventory_sha256'))
    require(runtime['profile'] == PROFILE and runtime['rootless'] is True, 'cas_v2_profile')
    for kind in ('uid', 'gid'):
        owner = number(runtime['owner_' + kind], 2**32, 1)
        require(mapped_id(runtime[kind + '_map'], 0) == owner
                and mapped_id(runtime[kind + '_map'], helper[kind]) != owner, 'cas_v2_rootless_mapping')
    _process(runtime['postgres_process'])
    _namespace(runtime['network_namespace'])
    _namespace(runtime['postgres_mount_namespace'])
    require(all(runtime['postgres_process'][k] == runtime['network_namespace'][k]
                for k in ('dev', 'ino')), 'cas_v2_process_network_namespace')
    _mounts(runtime['mounts'], pg)
    fingerprint(runtime['mount_inventory_sha256'])
    require(runtime['mount_inventory_sha256'] == digest(canonical(runtime['mounts'])),
            'cas_v2_mount_inventory')
    bindings = value['receipt_bindings']
    keys(bindings, ('receipt_sha256', 'p_intent_before_effect_sha256', 'intent_before_effect_sha256',
                    'clonado_sha256', 'sql_func_sha256', 'a_physical_binding_sha256', 'post'))
    keys(bindings['receipt_sha256'], ('p', 'a', 'intent', 'l'))
    for pin in bindings['receipt_sha256'].values():
        fingerprint(pin)
    for field in set(bindings) - {'receipt_sha256', 'post'}:
        fingerprint(bindings[field])
    _state(bindings['post'])
    require(bindings['receipt_sha256']['intent'] == bindings['intent_before_effect_sha256']
            and bindings['receipt_sha256']['a'] == value['pins']['aut26_receipt_sha256']
            and bindings['receipt_sha256']['l'] == value['pins']['login_receipt_sha256'],
            'cas_v2_receipt_binding')


def validate_request(value):
    value = decode(canonical(value))
    keys(value, REQUEST_FIELDS)
    require(value['kind'] == REQUEST_KIND, 'cas_v2_kind')
    _common(value)
    return value


def _session(value, request, channel):
    keys(value, SESSION_FIELDS)
    role = ROLES[channel]
    require(value['session_user'] == request['users'][channel]
            and value['current_user'] == (request['users'][channel] if role == 'none' else role)
            and value['role'] == role and value['transaction_read_only'] == 'on'
            and value['transaction_isolation'] == 'read committed', 'cas_v2_session_role')
    number(value['backend_pid'], 2**31, 1)
    number(value['database_oid'], 2**32, 1)
    number(value['server_port'], 65536, 1)
    require(all(value[k] == request['pgid'][k] for k in ('database_name', 'database_oid', 'system_identifier'))
            and value['client_addr'] == value['server_addr'] == '127.0.0.1'
            and value['server_port'] == 5432 and value['login_safe'] is True
            and value['ssl'] is True and value['client_certificate'] is True
            and value['tls_version'] == 'TLSv1.3', 'cas_v2_session_identity')


def _preimages(value):
    keys(value, ('candidato', 'usuarios'))
    pairs = (('revision_control_rol', 'huella_control_rol', 2**63),
             ('version_asignacion', 'huella_asignacion', 2**63),
             ('version_contexto', 'huella_contexto', 2**64))
    for population, row in value.items():
        selected = pairs if population == 'candidato' else pairs[-1:]
        fields = {field for pair in selected for field in pair[:2]}
        keys(row, fields | ({'secuencia_motivos'} if population == 'candidato' else set()))
        for field, seal, maximum in selected:
            n = number(row[field], maximum)
            require(type(row[seal]) is str, 'cas_v2_preimage')
            if n == 0:
                require(row[seal] == '', 'cas_v2_preimage')
            else:
                fingerprint(row[seal])
        if population == 'candidato':
            number(row['secuencia_motivos'], 2**62)


def validate_sql_postimage_observation(observation, request, initial_sessions, stage):
    """Pure typed binding of an external observation; no SQL or acquisition.

stage is a fixed caller barrier, before or after. initial_sessions is the
channel -> initial SESSION_FIELDS map, not the result's before/after pairs.
The acquisition layer must hold those exact three sessions across both barriers
and obtain each observation externally before rollback; JSON is not authority.
"""
    request = validate_request(request)
    observation = decode(canonical(observation))
    initial_sessions = decode(canonical(initial_sessions))
    require(type(stage) is str and stage in ('before', 'after'), 'cas_v2_sql_stage')
    keys(initial_sessions, CHANNELS)
    for channel, session in initial_sessions.items():
        _session(session, request, channel)
    require(len({session['backend_pid'] for session in initial_sessions.values()}) == 3,
            'cas_v2_distinct_sessions')
    keys(observation, SQL_POSTIMAGE_FIELDS)
    require(observation['stage'] == stage, 'cas_v2_sql_stage')
    for field in ('nonce', 'request_sha256', 'sessions_sha256', 'normalizer_sha256',
                  'pg_container_id'):
        fingerprint(observation[field])
    _image_id(observation['pg_image_id'])
    number(observation['database_oid'], 2**32, 1)
    require(observation['nonce'] == request['nonce']
            and observation['request_sha256'] == digest(canonical(request))
            and observation['sessions_sha256'] == digest(canonical(initial_sessions)),
            'cas_v2_sql_observation_binding')
    require(all(observation[field] == request['pgid'][field] for field in
                ('pg_container_id', 'pg_image_id', 'system_identifier', 'database_oid')),
            'cas_v2_sql_observation_identity')
    _process(observation['postgres_process'])
    _namespace(observation['network_namespace'])
    fingerprint(observation['mount_inventory_sha256'])
    require(all(canonical({'value': observation[field]}) ==
                canonical({'value': request['runtime'][field]}) for field in
                ('postgres_process', 'network_namespace', 'mount_inventory_sha256')),
            'cas_v2_sql_observation_physical')
    keys(observation['normalizer_bundle'], NORMALIZER_FILES)
    for pin in observation['normalizer_bundle'].values():
        fingerprint(pin)
    require(observation['normalizer_bundle'] == request['normalizer_bundle']
            and observation['normalizer_sha256'] == request['pins']['normalizer_sha256'],
            'cas_v2_sql_normalizer')
    state = {field: observation[field] for field in STATE_FIELDS}
    _state(state)
    require(state == request['receipt_bindings']['post'], 'cas_v2_sql_postimage_drift')
    return observation


def validate_result(value, request=None):
    value = decode(canonical(value))
    keys(value, RESULT_FIELDS)
    require(value['kind'] == RESULT_KIND, 'cas_v2_kind')
    _common(value)
    require(value['cas_sessions_bound'] is True and value['cas_applied'] is False
            and value['rollback_confirmed'] is True, 'cas_v2_result_status')
    keys(value['sessions'], CHANNELS)
    pids = set()
    for channel, pair in value['sessions'].items():
        keys(pair, ('before', 'after'))
        for observation in pair.values():
            _session(observation, value, channel)
        require(pair['before'] == pair['after'], 'cas_v2_session_drift')
        pids.add(pair['before']['backend_pid'])
    require(len(pids) == 3, 'cas_v2_distinct_sessions')
    readings = value['readings_sha256']
    require(type(readings) is list and len(readings) == 2, 'cas_v2_readings')
    for reading in readings:
        fingerprint(reading)
    require(readings[0] == readings[1], 'cas_v2_reading_drift')
    keys(value['observations'], ('candidato', 'motivos'))
    for pair in value['observations'].values():
        keys(pair, ('lectura_1', 'lectura_2'))
        require(instant(pair['lectura_1']) <= instant(pair['lectura_2']), 'cas_v2_observation_order')
    _preimages(value['preimages'])
    keys(value['sql_postimage_observations'], ('before', 'after'))
    embedded_request = {field: value[field] for field in REQUEST_FIELDS}
    embedded_request['kind'] = REQUEST_KIND
    initial_sessions = {channel: pair['before'] for channel, pair in value['sessions'].items()}
    sql_pair = value['sql_postimage_observations']
    for stage, observation in sql_pair.items():
        validate_sql_postimage_observation(observation, embedded_request, initial_sessions, stage)
    require({field: v for field, v in sql_pair['before'].items() if field != 'stage'} ==
            {field: v for field, v in sql_pair['after'].items() if field != 'stage'},
            'cas_v2_sql_postimage_changed')
    _state(value['postimage'])
    require(value['postimage'] == {field: sql_pair['after'][field] for field in STATE_FIELDS},
            'cas_v2_postimage_drift')
    if request is not None:
        request = validate_request(request)
        require(all(value[k] == request[k] for k in REQUEST_FIELDS - {'kind'}), 'cas_v2_request_drift')
    return value


def bind_result(request, result):
    return validate_result(result, request)


def validate_pending(value):
    value = decode(canonical(value))
    keys(value, ('version', 'kind', 'request', 'request_sha256'))
    require(type(value['version']) is int and value['version'] == VERSION
            and value['kind'] == PENDING_KIND, 'cas_v2_pending')
    request = validate_request(value['request'])
    fingerprint(value['request_sha256'])
    require(value['request_sha256'] == digest(canonical(request)), 'cas_v2_request_hash')
    return value


def validate_measurement(value, request=None, *, result_raw=None):
    value = decode(canonical(value))
    keys(value, ('version', 'kind', 'request_sha256', 'result', 'physical',
                 'result_bytes_sha256', 'result_canonical_sha256'))
    require(type(value['version']) is int and value['version'] == VERSION
            and value['kind'] == MEASUREMENT_KIND, 'cas_v2_measurement')
    result = validate_result(value['result'], request)
    embedded = {k: result[k] for k in REQUEST_FIELDS}
    embedded['kind'] = REQUEST_KIND
    fingerprint(value['request_sha256'])
    require(value['request_sha256'] == digest(canonical(embedded)), 'cas_v2_request_hash')
    fingerprint(value['result_bytes_sha256'])
    fingerprint(value['result_canonical_sha256'])
    require(value['result_canonical_sha256'] == digest(canonical(result)), 'cas_v2_result_canonical_hash')
    if result_raw is not None:
        require(digest(result_raw) == value['result_bytes_sha256']
                and canonical(decode(result_raw)) == canonical(result), 'cas_v2_result_original_bytes')
    keys(value['physical'], ('before', 'after'))
    for physical in value['physical'].values():
        keys(physical, PHYSICAL_FIELDS)
        for key in ('network_namespace', 'postgres_mount_namespace', 'helper_mount_namespace'):
            _namespace(physical[key])
        for key in ('postgres_process', 'helper_process'):
            _process(physical[key])
            require(all(physical[key][f] == physical['network_namespace'][f] for f in ('dev', 'ino')),
                    'cas_v2_process_network_namespace')
        require(physical['postgres_process']['pid'] != physical['helper_process']['pid']
                and physical['postgres_mount_namespace'] != physical['helper_mount_namespace'],
                'cas_v2_process_separation')
        require(all(canonical({'value': physical[k]}) == canonical({'value': result['runtime'][k]})
                    for k in PHYSICAL_FIELDS -
                    {'helper_process', 'helper_mount_namespace'}), 'cas_v2_physical_binding')
    require(value['physical']['before'] == value['physical']['after'], 'cas_v2_physical_drift')
    return value


def validate(value):
    keys(value, set(value) if type(value) is dict else (), 'cas_v2_object')
    kind = value.get('kind')
    require(type(kind) is str and kind in (REQUEST_KIND, PENDING_KIND, RESULT_KIND, MEASUREMENT_KIND),
            'cas_v2_kind')
    return {REQUEST_KIND: validate_request, PENDING_KIND: validate_pending,
            RESULT_KIND: validate_result, MEASUREMENT_KIND: validate_measurement}[kind](value)
