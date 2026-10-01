#!/usr/bin/env python3
"""Preparatory CAS helper for the Podman v2 profile; operational gate closed.

The private fixture core accepts an in-memory driver for bounded tests. CLI and
run_request always refuse before file acquisition, driver import or connection.
Nominal ACL/RLS and HBA bytes need independently reviewed external pins. The
catalog projection is supplemental; two externally normalized H6 attestations
bind the full SQL state while all nominal sessions are open. The private
observer's backend owns hard timeouts; live acquisition remains unaccredited.
"""
from contextlib import ExitStack
import os
from pathlib import Path
import re
import stat
import sys

try:
    from . import clon_preimagenes_nominales as probe
    from . import clon_cas_podman_contract as contract
except ImportError:
    import clon_preimagenes_nominales as probe
    import clon_cas_podman_contract as contract

Refused = contract.Refused
CHANNELS = ('identidad', 'contexto', 'autorizacion')
EXECUTION_AUTHORITY = None
MODULE_CLOSURE = ('clon_cas_podman_helper.py', 'clon_preimagenes_nominales.py',
                  'clon_cas_podman_contract.py')
ROLES = {'identidad': 'vec_identidad_sesiones_v1_propietario',
         'contexto': 'vec_contexto_actor_v1_propietario', 'autorizacion': 'none'}
AUTH_USER = 'vec_externo_v3_fuente_autorizacion_desarrollo'
INPUT_FILES = {'source_sha256': 'source.json', 'alias_sha256': 'alias.json',
 'restore_sha256': 'h1-restore.json', 'journal_sha256': 'sql-journal.json',
 'h1_records_sha256': 'h1-records.json', 'aut26_receipt_sha256': 'aut26-receipt.json',
 'login_receipt_sha256': 'login-receipt.json', 'hba_sha256': 'pg_hba.conf',
 'acl_sha256': 'acl.json', 'rls_sha256': 'rls.json'}
SESSION_FIELDS = ('session_user', 'current_user', 'role', 'transaction_read_only',
 'transaction_isolation', 'backend_pid', 'database_name', 'database_oid', 'system_identifier',
 'client_addr', 'server_addr', 'server_port', 'login_safe', 'ssl', 'tls_version', 'client_certificate')


def require(ok, code):
    if not ok:
        raise Refused(code)


def rows(cursor, sql, parameters=(), maximum=1):
    try:
        return probe.rows(cursor, sql, parameters, maximum)
    except probe.Refused as error:
        raise Refused(str(error)) from None

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
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles inherited
 WHERE (inherited.rolsuper OR inherited.rolcreatedb OR inherited.rolcreaterole
 OR inherited.rolreplication OR inherited.rolbypassrls
 OR inherited.rolname IN ('pg_read_all_data','pg_write_all_data'))
 AND pg_catalog.pg_has_role(session_user,inherited.oid,'MEMBER'))
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
        values[name] = data if name == 'pg_hba.conf' else contract.decode(data)
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
    for name in ('acl.json', 'rls.json'):
        require(contract.canonical(values[name]) == file_bytes(Path('/inputs') / name),
                'cas_metadata_not_canonical')
    return {'aliases': mapped, 'acl': values['acl.json'], 'rls': values['rls.json']}


# Catalog projections exclude password hashes, rows and provider diagnostics.
# This projection covers nominal checks, not the complete H6 dump. Function
# bodies, default/column ACLs, database settings and other SQL state require
# the independently pinned H6 normalizer bundle's two external observations.
ACL_SQL = '''SELECT pg_catalog.jsonb_build_object(
 'roles', (SELECT coalesce(jsonb_agg(jsonb_build_object('name',rolname,
 'login',rolcanlogin,'inherit',rolinherit,'super',rolsuper,'createdb',rolcreatedb,
 'createrole',rolcreaterole,'replication',rolreplication,'bypassrls',rolbypassrls,
 'connlimit',rolconnlimit,'validuntil',rolvaliduntil,'config',rolconfig)
 ORDER BY rolname),'[]'::jsonb) FROM pg_catalog.pg_roles),
 'memberships', (SELECT coalesce(jsonb_agg(jsonb_build_object(
 'role',r.rolname,'member',m.rolname,'grantor',g.rolname,'admin',a.admin_option,
 'inherit',a.inherit_option,'set',a.set_option) ORDER BY r.rolname,m.rolname,g.rolname),
 '[]'::jsonb) FROM pg_catalog.pg_auth_members a
 JOIN pg_catalog.pg_roles r ON r.oid=a.roleid JOIN pg_catalog.pg_roles m ON m.oid=a.member
 JOIN pg_catalog.pg_roles g ON g.oid=a.grantor),
 'tables', (SELECT coalesce(jsonb_agg(jsonb_build_object('name',n.nspname||'.'||c.relname,
 'owner',pg_catalog.pg_get_userbyid(c.relowner),'acl',c.relacl::text)
 ORDER BY n.nspname,c.relname),'[]'::jsonb) FROM pg_catalog.pg_class c
 JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=ANY(%s)
 AND c.relkind IN ('r','p','v','m','S')),
 'functions', (SELECT coalesce(jsonb_agg(jsonb_build_object(
 'name',n.nspname||'.'||p.proname||'('||pg_catalog.pg_get_function_identity_arguments(p.oid)||')',
 'owner',pg_catalog.pg_get_userbyid(p.proowner),'acl',p.proacl::text,
 'security_definer',p.prosecdef,'settings',p.proconfig)
 ORDER BY n.nspname,p.proname,p.oid),'[]'::jsonb) FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=ANY(%s)))'''
RLS_SQL = '''SELECT pg_catalog.jsonb_build_object('tables',
 (SELECT coalesce(jsonb_agg(jsonb_build_object('name',n.nspname||'.'||c.relname,
 'owner',pg_catalog.pg_get_userbyid(c.relowner),'enabled',c.relrowsecurity,
 'forced',c.relforcerowsecurity,'policies',
 (SELECT coalesce(jsonb_agg(jsonb_build_object('name',p.polname,
 'permissive',p.polpermissive,'command',p.polcmd::text,
 'roles',(SELECT jsonb_agg(CASE WHEN r=0 THEN 'PUBLIC' ELSE pg_catalog.pg_get_userbyid(r) END ORDER BY r)
 FROM unnest(p.polroles) r),'using',pg_catalog.pg_get_expr(p.polqual,p.polrelid),
 'check',pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)) ORDER BY p.polname),'[]'::jsonb)
 FROM pg_catalog.pg_policy p WHERE p.polrelid=c.oid)) ORDER BY n.nspname,c.relname),'[]'::jsonb)
 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname=ANY(%s) AND c.relkind IN ('r','p')))'''
SCHEMAS = ['vec_identidad_externa_v1', 'vec_contexto_actor_v1', 'vec_autorizacion']


def keys(value, expected, code):
    require(type(value) is dict and set(value) == set(expected), code)


def require_authority(request):
    # Neither approved:true nor an internally consistent receipt is authority.
    require(EXECUTION_AUTHORITY is not None, 'autoridad_ejecucion_pendiente')
    raise Refused('autoridad_ejecucion_pendiente')


def validate_session(request, channel, value):
    keys(value, SESSION_FIELDS, 'cas_session_schema')
    expected = request['users'][channel] if ROLES[channel] == 'none' else ROLES[channel]
    require(value['session_user'] == request['users'][channel] and value['current_user'] == expected and
            value['role'] == ROLES[channel] and value['transaction_read_only'] == 'on' and
            value['transaction_isolation'] == 'read committed', 'cas_session_role')
    probe.number(value['backend_pid'], 2**31, 1)
    require(value['database_name'] == request['pgid']['database_name'] and
            type(value['database_oid']) is int and value['database_oid'] == request['pgid']['database_oid'] and
            value['system_identifier'] == request['pgid']['system_identifier'] and
            value['client_addr'] in ('127.0.0.1', '::1') and value['server_addr'] in ('127.0.0.1', '::1') and
            type(value['server_port']) is int and value['server_port'] == 5432 and
            value['login_safe'] is True and value['ssl'] is True and
            value['tls_version'] == 'TLSv1.3' and value['client_certificate'] is True,
            'cas_session_physical')


def session(request, channel, cursor):
    result = rows(cursor, SESSION_SQL)
    require(len(result) == 1 and len(result[0]) == len(SESSION_FIELDS), 'cas_session_missing')
    value = dict(zip(SESSION_FIELDS, result[0]))
    validate_session(request, channel, value)
    return value


def access(request, cursors, metadata):
    # Compare the supplemental projections twice; PUBLIC and role edges are included.
    acl = rows(cursors['identidad'], ACL_SQL, (SCHEMAS, SCHEMAS))
    rls = rows(cursors['identidad'], RLS_SQL, (SCHEMAS,))
    require(len(acl) == len(rls) == 1 and len(acl[0]) == len(rls[0]) == 1 and
            contract.canonical(acl[0][0]) == contract.canonical(metadata['acl']) and
            contract.canonical(rls[0][0]) == contract.canonical(metadata['rls']), 'cas_metadata_changed')
    require(len(contract.canonical(metadata['acl'])) <= 1 << 20 and
            len(contract.canonical(metadata['rls'])) <= 1 << 20, 'cas_metadata_size')
    for table in ('cuenta', 'alias_cuenta', 'estado_actual', 'estado_cuenta'):
        require(rows(cursors['identidad'], TABLE_SQL,
                (ROLES['identidad'], 'vec_identidad_externa_v1.' + table)) == [(True,) * 6],
                'cas_identity_permission_rls')
    for channel, owner, signature in (
          ('contexto', ROLES['contexto'], 'vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1(text)'),
          ('autorizacion', 'vec_autorizacion_propietario',
           'vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text)'),
          ('autorizacion', 'vec_autorizacion_propietario',
           'vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()')):
        require(rows(cursors[channel], FUNCTION_SQL, (owner, signature)) == [(True,) * 4],
                'cas_function_permission')
    for table in ('version_rol', 'control_vigencia_version_rol', 'control_vigencia_version_rol_actual',
                  'asignacion_perfil_externa', 'asignacion_perfil_actual_externa', 'motivo_v2_checkpoint_origen',
                  'motivo_v2_evento_origen', 'motivo_v2_catalogo_publicado', 'motivo_v2_retirada', 'motivo_v2_entrada'):
        name = 'vec_autorizacion.' + table
        require(rows(cursors['autorizacion'], AUTH_TABLE_SQL, (name, name)) == [(False, False)],
                'cas_auth_direct_table_permission')


def _observe_sql(stage, request, initial_sessions, connections, driver, state_observer):
    """Private barrier: observe with all three nominal transactions still open.

    Only a backend-owned observer may acquire the H6 state. It must enforce a
    hard deadline itself; errors and timeouts propagate to the core's cleanup.
    The normalizer pin covers h6_comun.sh and h6_normalizar_pg_dump.py together.
    Detached arguments prevent an observer from rewriting the bound request.
    """
    require(set(connections) == set(CHANNELS) and all(
        connection.closed is False and
        connection.info.transaction_status == driver.pq.TransactionStatus.INTRANS
        for connection in connections.values()), 'cas_observer_sessions_closed')
    try:
        observation = state_observer(stage,
            contract.decode(contract.canonical(request)),
            contract.decode(contract.canonical(initial_sessions)))
    except Exception:
        raise Refused('cas_sql_observation_not_accredited') from None
    require(all(connection.closed is False and
        connection.info.transaction_status == driver.pq.TransactionStatus.INTRANS
        for connection in connections.values()), 'cas_observer_sessions_closed')
    return contract.validate_sql_postimage_observation(
        observation, request, initial_sessions, stage)


def _fixture_core(request, driver, input_reader, state_observer):
    """Private preparatory core. Tests supply fake cursors and fake input bytes.

    This has no command-line selector and is never used to bypass the real gate.
    The driver must implement connect/close/cursor/rollback and IDLE/INTRANS
    states. The private observer supplies normalized H6 state at two barriers;
    backend deadlines are mandatory before any operational implementation.
    """
    request = contract.validate_request(contract.decode(contract.canonical(request)))
    owned, cursors, initial = {}, {}, {}
    failure = None
    cleanup_failed = False
    try:
        metadata = input_reader(request)
        with ExitStack() as stack:
            # All connections stay open through both observations and rollback.
            for channel in CHANNELS:
                connection = driver.connect(host=request['server_name'], hostaddr='127.0.0.1',
                    port=5432, dbname=request['pgid']['database_name'], user=request['users'][channel],
                    sslmode='verify-full', sslrootcert='/tls/ca.pem', sslcert='/tls/' + channel + '.crt',
                    sslkey='/tls/' + channel + '.key', connect_timeout=5, autocommit=True,
                    options='-c default_transaction_read_only=on -c statement_timeout=5000 -c lock_timeout=1000 '
                            '-c idle_in_transaction_session_timeout=10000 -c search_path=pg_catalog',
                    application_name='vec-clon-cas-podman-nominal-ro-v2')
                owned[channel] = connection
                cursor = connection.cursor()
                stack.callback(cursor.close)
                cursors[channel] = cursor
                cursor.execute('BEGIN ISOLATION LEVEL READ COMMITTED READ ONLY')
                if ROLES[channel] != 'none':
                    # Fixed allowlist, never interpolate a role from request.
                    cursor.execute('SET LOCAL ROLE ' + ROLES[channel])
                initial[channel] = session(request, channel, cursor)
            require(len({s['backend_pid'] for s in initial.values()}) == 3, 'cas_sessions_not_distinct')
            access(request, cursors, metadata)
            sql_before = _observe_sql('before', request, initial, owned, driver, state_observer)
            first = probe.scan(cursors, request['targets'], metadata['aliases'])
            require(contract.canonical(input_reader(request)) == contract.canonical(metadata), 'cas_private_input_changed')
            second = probe.scan(cursors, request['targets'], metadata['aliases'])
            require(probe.canonical(probe.semantic(first)) == probe.canonical(probe.semantic(second)), 'cas_preimages_changed')
            final = {c: session(request, c, cursors[c]) for c in CHANNELS}
            require(initial == final, 'cas_session_changed')
            access(request, cursors, metadata)
            require(contract.canonical(input_reader(request)) == contract.canonical(metadata), 'cas_private_input_changed')
            sql_after = _observe_sql('after', request, initial, owned, driver, state_observer)
            for channel, cursor in cursors.items():
                cursor.execute('ROLLBACK')
                require(owned[channel].info.transaction_status == driver.pq.TransactionStatus.IDLE,
                        'cas_rollback_unconfirmed')
                require(rows(cursor, ROLLBACK_SQL) == [('on', 'none')], 'cas_rollback_unconfirmed')
                require(owned[channel].info.transaction_status == driver.pq.TransactionStatus.IDLE,
                        'cas_rollback_unconfirmed')
        require(contract.canonical(input_reader(request)) == contract.canonical(metadata), 'cas_private_input_changed')
        result = {**request, 'kind': contract.RESULT_KIND,
                  'sessions': {c: {'before': initial[c], 'after': final[c]} for c in CHANNELS},
                  'cas_sessions_bound': True, 'cas_applied': False, 'rollback_confirmed': True,
                  'sql_postimage_observations': {'before': sql_before, 'after': sql_after},
                  'postimage': {field: sql_after[field] for field in contract.STATE_FIELDS},
                  'readings_sha256': [probe.digest(probe.canonical(probe.semantic(v))) for v in (first, second)],
                  'observations': {k: {'lectura_1': first[c]['observada_en'], 'lectura_2': second[c]['observada_en']}
                                   for k, c in (('candidato', 'autorizacion'), ('motivos', 'motivos'))},
                  'preimages': second['preimages']}
        contract.bind_result(request, result)
    except Exception as error:
        failure = error
    finally:
        # Try every rollback/close even if an earlier callback or close fails.
        for connection in reversed(tuple(owned.values())):
            try:
                connection.rollback()
                require(connection.info.transaction_status == driver.pq.TransactionStatus.IDLE,
                        'cas_rollback_unconfirmed')
            except Exception:
                cleanup_failed = True
            try:
                connection.close()
                require(connection.closed is True, 'cas_close_unconfirmed')
            except Exception:
                cleanup_failed = True
    if cleanup_failed:
        raise Refused('cas_cleanup_unconfirmed') from None
    if failure is not None:
        if isinstance(failure, (Refused, probe.Refused)):
            raise Refused(str(failure)) from None
        raise Refused('cas_helper_not_accredited') from None
    return result


def run_request(request):
    """Closed execution API: gate precedes validation, files and driver imports."""
    require_authority(request)
    raise Refused('autoridad_ejecucion_pendiente')


def main():
    try:
        result = run_request(contract.decode(sys.stdin.buffer.read((1 << 20) + 1)))
        sys.stdout.buffer.write(contract.canonical(result) + b'\n')
        return 0
    except Exception:
        sys.stderr.write('cas_helper_refused\n')
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
