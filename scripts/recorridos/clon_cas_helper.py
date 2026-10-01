#!/usr/bin/env python3
"""Helper CAS de tres sesiones nominales RO en el namespace PostgreSQL del clon.

La ejecución real permanece cerrada hasta recibir el contrato y los recibos D
aprobados. Las pruebas usan psycopg simulado; no acreditan H6 ni instalación.
La salida conserva CAS mínimos e instantes, nunca alias, HMAC ni filas privadas.
"""
from contextlib import ExitStack
import json
import os
from pathlib import Path
import re
import stat
import sys

try:
    from . import clon_preimagenes_nominales as probe
except ImportError:
    import clon_preimagenes_nominales as probe

Refused = probe.Refused
require = probe.require
keys = probe.keys
CHANNELS = ('identidad', 'contexto', 'autorizacion')
EXECUTION_AUTHORITY = None  # Contrato AUT26/LOGIN/TLS de D todavía pendiente.
MODULE_CLOSURE = ('clon_cas_helper.py', 'clon_preimagenes_nominales.py')
ROLES = {'identidad': 'vec_identidad_sesiones_v1_propietario',
         'contexto': 'vec_contexto_actor_v1_propietario', 'autorizacion': 'none'}
AUTH_USER = 'vec_externo_v3_fuente_autorizacion_desarrollo'
INPUT_FILES = {'source_sha256': 'source.json', 'alias_sha256': 'alias.json',
 'restore_sha256': 'h1-restore.json', 'journal_sha256': 'sql-journal.json',
 'h1_records_sha256': 'h1-records.json', 'aut26_receipt_sha256': 'aut26-receipt.json',
 'login_receipt_sha256': 'login-receipt.json'}
REQUEST_FIELDS = {'version', 'kind', 'nonce', 'server_name', 'pgid', 'pins', 'users', 'tls_hashes', 'targets'}
RESULT_FIELDS = {'version', 'kind', 'nonce', 'pgid', 'pins', 'tls_hashes', 'sessions',
 'cas_sessions_bound', 'cas_applied', 'rollback_confirmed', 'readings_sha256', 'observations', 'preimages'}
SESSION_FIELDS = ('session_user', 'current_user', 'role', 'transaction_read_only',
 'transaction_isolation', 'backend_pid', 'database_name', 'database_oid', 'system_identifier',
 'client_addr', 'server_addr', 'server_port', 'login_safe', 'ssl', 'tls_version', 'client_certificate')
PGID_FIELDS = {'pg_container_id', 'pg_image', 'pg_image_id', 'pg_volume',
               'system_identifier', 'database_name', 'database_oid'}
SESSION_SQL = '''SELECT session_user::text,current_user::text,current_setting('role'),
 current_setting('transaction_read_only'),current_setting('transaction_isolation'),
 pg_backend_pid(),current_database(),
 (SELECT oid::bigint FROM pg_catalog.pg_database WHERE datname=current_database()),
 (SELECT system_identifier::text FROM pg_catalog.pg_control_system()),
 inet_client_addr()::text,inet_server_addr()::text,inet_server_port(),
 (SELECT rolcanlogin AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)
 AND rolconfig IS NULL AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles active
 WHERE active.rolname=current_user AND NOT (active.rolsuper OR active.rolcreatedb
 OR active.rolcreaterole OR active.rolreplication OR active.rolbypassrls) AND active.rolconfig IS NULL)
 FROM pg_catalog.pg_roles WHERE rolname=session_user),
 s.ssl,s.version,s.client_dn IS NOT NULL FROM pg_catalog.pg_stat_ssl s WHERE s.pid=pg_backend_pid()'''
TABLE_SQL = '''SELECT c.relowner=pg_catalog.to_regrole(%s),c.relrowsecurity,c.relforcerowsecurity,
 pg_catalog.has_table_privilege(current_user,c.oid,'SELECT'),
 pg_catalog.row_security_active(c.oid),
 (SELECT count(*)=1 AND bool_and(p.polpermissive AND p.polcmd='*'
 AND p.polroles=ARRAY[c.relowner]::oid[]
 AND pg_catalog.pg_get_expr(p.polqual,p.polrelid)='(CURRENT_USER = ''vec_identidad_sesiones_v1_propietario''::name)'
 AND pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)='(CURRENT_USER = ''vec_identidad_sesiones_v1_propietario''::name)')
 FROM pg_catalog.pg_policy p WHERE p.polrelid=c.oid)
 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass(%s)'''
FUNCTION_SQL = '''SELECT p.prosecdef,p.proowner=pg_catalog.to_regrole(%s),
 pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE'),
 p.proconfig=ARRAY['search_path=pg_catalog']::text[]
 FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(%s)'''
AUTH_TABLE_SQL = '''SELECT pg_catalog.has_table_privilege(current_user,%s,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'),
 pg_catalog.has_any_column_privilege(current_user,%s,'SELECT,INSERT,UPDATE,REFERENCES')'''
ROLLBACK_SQL = "SELECT current_setting('transaction_read_only'),current_setting('role')"


def decode(raw):
    require(type(raw) is bytes and 0 < len(raw) <= 1 << 20, 'cas_input_size')
    def unique(pairs):
        value = {}
        for key, item in pairs:
            require(key not in value, 'cas_json_duplicate')
            value[key] = item
        return value
    try:
        result = json.loads(raw, object_pairs_hook=unique,
                            parse_constant=lambda _: require(False, 'cas_json_constant'))
    except (ValueError, UnicodeError):
        raise Refused('cas_json_invalid') from None
    require(type(result) is dict, 'cas_json_object')
    return result


def validate_request(request):
    keys(request, REQUEST_FIELDS, 'cas_request_schema')
    require(type(request['version']) is int and request['version'] == 1 and
            request['kind'] == 'cas_helper_request_v1', 'cas_request_version')
    probe.fingerprint(request['nonce'])
    require(type(request['server_name']) is str and
            re.fullmatch(r'[a-z][a-z0-9.-]{0,120}', request['server_name']), 'cas_server_name')
    pg = request['pgid']
    keys(pg, PGID_FIELDS, 'cas_pgid_schema')
    probe.fingerprint(pg['pg_container_id'])
    require(type(pg['pg_image_id']) is str and re.fullmatch(r'sha256:[a-f0-9]{64}', pg['pg_image_id']) and
            pg['pg_image'] == 'postgres:18.4' and pg['database_name'] == 'postgres' and
            type(pg['pg_volume']) is str and re.fullmatch(r'/dev/shm/vec-recorridos-[A-Za-z0-9_-]+', pg['pg_volume']) and
            type(pg['system_identifier']) is str and re.fullmatch(r'[1-9][0-9]{0,19}', pg['system_identifier']) and
            int(pg['system_identifier']) < 2**64, 'cas_pgid_invalid')
    probe.number(pg['database_oid'], 2**32, 1)
    keys(request['pins'], INPUT_FILES, 'cas_pins_schema')
    for pin in request['pins'].values():
        probe.fingerprint(pin)
    users = request['users']
    keys(users, CHANNELS, 'cas_users_schema')
    require(all(type(user) is str and re.fullmatch(r'vec_[a-z0-9_]{1,100}', user) for user in users.values()) and
            len(set(users.values())) == 3 and users['autorizacion'] == AUTH_USER and
            not any(user in ROLES.values() for user in users.values()), 'cas_users_invalid')
    tls = {'ca.pem'} | {channel + ext for channel in CHANNELS for ext in ('.crt', '.key')}
    keys(request['tls_hashes'], tls, 'cas_tls_schema')
    for pin in request['tls_hashes'].values():
        probe.fingerprint(pin)
    targets = request['targets']
    keys(targets, ('candidato', 'usuarios'), 'cas_targets_schema')
    for population, target in targets.items():
        keys(target, ('cuenta_ref', 'persona_ref', 'perfil_ref', 'provision_ref'), 'cas_target_schema')
        for field, prefix in (('cuenta_ref', 'cta'), ('persona_ref', 'per'), ('perfil_ref', 'prf'),
                             ('provision_ref', 'pce' if population == 'candidato' else 'pue')):
            require(type(target[field]) is str and re.fullmatch(prefix + r'_[A-Za-z0-9_-]{16,128}', target[field]),
                    'cas_target_ref')
    c, u = targets['candidato'], targets['usuarios']
    if c['cuenta_ref'] == u['cuenta_ref']:
        require((c['persona_ref'], c['perfil_ref']) == (u['persona_ref'], u['perfil_ref']), 'cas_shared_account')


def require_authority(request):
    # No booleano ni recibo autoconsistente del llamador puede abrir esta puerta.
    require(EXECUTION_AUTHORITY is not None, 'autoridad_ejecucion_pendiente')
    # Se implementará con el contrato exacto revisado de D, nunca con un flag.
    raise Refused('autoridad_ejecucion_pendiente')


def file_bytes(path, maximum=1 << 20):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        before = os.fstat(fd)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                0 < before.st_size <= maximum, 'cas_file_invalid')
        data = bytearray()
        while len(data) <= maximum:
            chunk = os.read(fd, min(65536, maximum + 1 - len(data)))
            if not chunk:
                break
            data.extend(chunk)
        after = os.fstat(fd)
        fields = ('st_dev', 'st_ino', 'st_mode', 'st_uid', 'st_nlink', 'st_size', 'st_mtime_ns', 'st_ctime_ns')
        require(all(getattr(before, k) == getattr(after, k) for k in fields) and
                len(data) == before.st_size, 'cas_file_changed')
        return bytes(data)
    finally:
        os.close(fd)


def read_inputs(request):
    values = {}
    for pin, name in INPUT_FILES.items():
        data = file_bytes(Path('/inputs') / name)
        require(probe.digest(data) == request['pins'][pin], 'cas_private_input_changed')
        values[name] = decode(data)
    for name, pin in request['tls_hashes'].items():
        require(probe.digest(file_bytes(Path('/tls') / name, 65536)) == pin, 'cas_tls_changed')
    source, aliases = values['source.json'], values['alias.json']
    require(source.get('version') == 1 and source.get('datos_sinteticos') is True and
            type(source.get('cuentas')) is list and 1 <= len(source['cuentas']) <= 32, 'cas_source_invalid')
    require(aliases.get('version') == 1 and set(aliases) == {'version', 'cuentas'} and
            type(aliases['cuentas']) is list and 1 <= len(aliases['cuentas']) <= 32, 'cas_alias_schema')
    mapped = {}
    for alias in aliases['cuentas']:
        keys(alias, ('cuenta_ref', 'esquema', 'dominio_ref', 'clave_id', 'clave_version',
                     'cuenta_id_hmac', 'sujeto_id_hmac'), 'cas_alias_schema')
        require(type(alias['cuenta_ref']) is str and alias['cuenta_ref'] not in mapped and
                alias['esquema'] == 'vec.identidad.hmac-sha256.v1' and
                all(type(alias[k]) is str and re.fullmatch(r'[!-~]{1,128}', alias[k])
                    for k in ('dominio_ref', 'clave_id')), 'cas_alias_invalid')
        probe.number(alias['clave_version'], minimum=1)
        probe.fingerprint(alias['cuenta_id_hmac']); probe.fingerprint(alias['sujeto_id_hmac'])
        require(alias['cuenta_id_hmac'] != alias['sujeto_id_hmac'], 'cas_alias_invalid')
        mapped[alias['cuenta_ref']] = alias
    for population, target in request['targets'].items():
        accounts = [a for a in source['cuentas'] if type(a) is dict and a.get('cuenta_ref') == target['cuenta_ref']]
        require(len(accounts) == 1 and accounts[0].get('persona_ref') == target['persona_ref'] and
                population in accounts[0].get('poblaciones', []) and target['cuenta_ref'] in mapped, 'cas_source_target')
    candidate = source.get('candidatoBolsa')
    require(type(candidate) is dict and all(candidate.get(k) == request['targets']['candidato'][k]
            for k in ('cuenta_ref', 'persona_ref', 'perfil_ref')), 'cas_source_candidate')
    journal = values['sql-journal.json']
    require(journal.get('phase') == 'awaiting_ad132' and type(journal.get('installed')) is list and
            len(journal['installed']) == 62, 'cas_sql62_required')
    return mapped


def validate_session(request, channel, value):
    keys(value, SESSION_FIELDS, 'cas_session_schema')
    expected_current = request['users'][channel] if ROLES[channel] == 'none' else ROLES[channel]
    require(value['session_user'] == request['users'][channel] and value['current_user'] == expected_current and
            value['role'] == ROLES[channel] and value['transaction_read_only'] == 'on' and
            value['transaction_isolation'] == 'read committed', 'cas_session_role')
    probe.number(value['backend_pid'], 2**31, 1)
    require(value['database_name'] == request['pgid']['database_name'] and
            type(value['database_oid']) is int and value['database_oid'] == request['pgid']['database_oid'] and
            value['system_identifier'] == request['pgid']['system_identifier'] and
            value['client_addr'] in ('127.0.0.1', '::1') and value['server_addr'] in ('127.0.0.1', '::1') and
            type(value['server_port']) is int and value['server_port'] == 5432 and
            value['login_safe'] is True and value['ssl'] is True and
            value['tls_version'] in ('TLSv1.2', 'TLSv1.3') and value['client_certificate'] is True,
            'cas_session_physical')


def session(request, channel, cursor):
    result = probe.rows(cursor, SESSION_SQL)
    require(len(result) == 1 and len(result[0]) == len(SESSION_FIELDS), 'cas_session_missing')
    value = dict(zip(SESSION_FIELDS, result[0]))
    validate_session(request, channel, value)
    return value


def access(cursors):
    for table in ('cuenta', 'alias_cuenta', 'estado_actual', 'estado_cuenta'):
        require(probe.rows(cursors['identidad'], TABLE_SQL,
                (ROLES['identidad'], 'vec_identidad_externa_v1.' + table)) == [(True, True, True, True, True, True)],
                'cas_identity_permission_rls')
    functions = (('contexto', 'vec_contexto_actor_v1_propietario',
                 'vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1(text)'),
                ('autorizacion', 'vec_autorizacion_propietario',
                 'vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text)'),
                ('autorizacion', 'vec_autorizacion_propietario',
                 'vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()'))
    for channel, owner, signature in functions:
        require(probe.rows(cursors[channel], FUNCTION_SQL, (owner, signature)) == [(True, True, True, True)],
                'cas_function_permission')
    for table in ('version_rol', 'control_vigencia_version_rol', 'control_vigencia_version_rol_actual',
                  'asignacion_perfil_externa', 'asignacion_perfil_actual_externa', 'motivo_v2_checkpoint_origen',
                  'motivo_v2_evento_origen', 'motivo_v2_catalogo_publicado', 'motivo_v2_retirada', 'motivo_v2_entrada'):
        name = 'vec_autorizacion.' + table
        require(probe.rows(cursors['autorizacion'], AUTH_TABLE_SQL, (name, name)) == [(False, False)],
                'cas_auth_direct_table_permission')


def validate_result(request, result):
    validate_request(request)
    keys(result, RESULT_FIELDS, 'cas_result_schema')
    require(type(result['version']) is int and result['version'] == 1 and result['kind'] == 'cas_helper_result_v1' and
            result['cas_sessions_bound'] is True and result['cas_applied'] is False and
            result['rollback_confirmed'] is True, 'cas_result_status')
    for field in ('nonce', 'pgid', 'pins', 'tls_hashes'):
        require(result[field] == request[field], 'cas_result_pin_changed')
    keys(result['sessions'], CHANNELS, 'cas_result_sessions')
    pids = set()
    for channel, pair in result['sessions'].items():
        keys(pair, ('before', 'after'), 'cas_session_pair')
        for value in pair.values():
            validate_session(request, channel, value)
        require(pair['before'] == pair['after'], 'cas_session_changed')
        pids.add(pair['before']['backend_pid'])
    require(len(pids) == 3, 'cas_sessions_not_distinct')
    require(type(result['readings_sha256']) is list and len(result['readings_sha256']) == 2 and
            result['readings_sha256'][0] == result['readings_sha256'][1], 'cas_readings_changed')
    for value in result['readings_sha256']:
        probe.fingerprint(value)
    keys(result['observations'], ('candidato', 'motivos'), 'cas_observations_schema')
    for pair in result['observations'].values():
        keys(pair, ('lectura_1', 'lectura_2'), 'cas_observations_pair')
        require(probe.instant(pair['lectura_1']) <= probe.instant(pair['lectura_2']), 'cas_observations_time')
    keys(result['preimages'], ('candidato', 'usuarios'), 'cas_preimages_schema')
    for population, value in result['preimages'].items():
        keys(value, probe.CAS_FIELDS if population == 'candidato' else probe.CONTEXT_FIELDS, 'cas_preimage_fields')
        pairs = [('version_contexto', 'huella_contexto', 2**64)]
        if population == 'candidato':
            pairs.extend((('revision_control_rol', 'huella_control_rol', 2**63),
                          ('version_asignacion', 'huella_asignacion', 2**63)))
            probe.number(value['secuencia_motivos'], 2**62)
        for number, seal, maximum in pairs:
            n = probe.number(value[number], maximum)
            require((n == 0 and value[seal] == '') or (n > 0 and type(value[seal]) is str and
                    re.fullmatch(r'[a-f0-9]{64}', value[seal])), 'cas_preimage_pair')


def run_request(request):
    """API cerrada: sin conexiones, consultas ni callbacks del llamador."""
    try:
        validate_request(request)
        # Copia JSON para impedir que el llamador cambie dicts mientras se leen.
        request = decode(probe.canonical(request))
        require_authority(request)
        aliases = read_inputs(request)
        import psycopg
        with ExitStack() as stack:
            cursors, connections, initial = {}, {}, {}
            for channel in CHANNELS:
                connection = stack.enter_context(psycopg.connect(host=request['server_name'], hostaddr='127.0.0.1',
                    port=5432, dbname='postgres', user=request['users'][channel], sslmode='verify-full',
                    sslrootcert='/tls/ca.pem', sslcert='/tls/' + channel + '.crt', sslkey='/tls/' + channel + '.key',
                    connect_timeout=5, autocommit=True,
                    options='-c default_transaction_read_only=on -c statement_timeout=5000 -c lock_timeout=1000',
                    application_name='vec-clon-cas-nominal-ro'))
                stack.callback(connection.rollback)
                cursor = stack.enter_context(connection.cursor())
                cursor.execute('BEGIN ISOLATION LEVEL READ COMMITTED READ ONLY')
                if channel == 'identidad':
                    cursor.execute('SET LOCAL ROLE vec_identidad_sesiones_v1_propietario')
                elif channel == 'contexto':
                    cursor.execute('SET LOCAL ROLE vec_contexto_actor_v1_propietario')
                connections[channel], cursors[channel] = connection, cursor
                initial[channel] = session(request, channel, cursor)
            require(len({s['backend_pid'] for s in initial.values()}) == 3, 'cas_sessions_not_distinct')
            access(cursors)
            first = probe.scan(cursors, request['targets'], aliases)
            require(read_inputs(request) == aliases, 'cas_private_input_changed')
            second = probe.scan(cursors, request['targets'], aliases)
            require(probe.semantic(first) == probe.semantic(second), 'cas_preimages_changed')
            final = {c: session(request, c, cursors[c]) for c in CHANNELS}
            require(initial == final, 'cas_session_changed')
            access(cursors)
            require(read_inputs(request) == aliases, 'cas_private_input_changed')
            # ROLLBACK con confirmación de estado IDLE y rol restablecido.
            for channel, cursor in cursors.items():
                cursor.execute('ROLLBACK')
                require(connections[channel].info.transaction_status == psycopg.pq.TransactionStatus.IDLE,
                        'cas_rollback_unconfirmed')
                require(probe.rows(cursor, ROLLBACK_SQL) == [('on', 'none')], 'cas_rollback_unconfirmed')
        require(read_inputs(request) == aliases, 'cas_private_input_changed')
        result = {'version': 1, 'kind': 'cas_helper_result_v1',
                  **{k: request[k] for k in ('nonce', 'pgid', 'pins', 'tls_hashes')},
                  'sessions': {c: {'before': initial[c], 'after': final[c]} for c in CHANNELS},
                  'cas_sessions_bound': True, 'cas_applied': False, 'rollback_confirmed': True,
                  'readings_sha256': [probe.digest(probe.canonical(probe.semantic(v))) for v in (first, second)],
                  'observations': {k: {'lectura_1': first[c]['observada_en'], 'lectura_2': second[c]['observada_en']}
                                   for k, c in (('candidato', 'autorizacion'), ('motivos', 'motivos'))},
                  'preimages': second['preimages']}
        validate_result(request, result)
        return result
    except Refused:
        raise
    except Exception:
        raise Refused('cas_helper_not_accredited') from None


def main():
    try:
        result = run_request(decode(sys.stdin.buffer.read((1 << 20) + 1)))
        sys.stdout.buffer.write(probe.canonical(result) + b'\n')
        return 0
    except Exception:
        # No error del driver, parámetros ni material privado en stdout/stderr.
        sys.stderr.write('cas_helper_refused\n')
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
