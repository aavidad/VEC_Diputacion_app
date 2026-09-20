#!/usr/bin/env python3
"""Pruebas SQL B11 sobre PG18.4 desechable, sin tocar bases conservadas.
Dependencias: Docker y Python 3. Solo datos sintéticos. No publica puertos.
000004 se intenta sin modificarlo; si rechaza su manifiesto se informa separado
 y se prueban las dependencias B11 reales 000001/2/3/5.
"""
from pathlib import Path
import hashlib
import os
import subprocess
import time

ROOT = Path(__file__).resolve().parent
NAME = f'vec-b11-000007-test-{os.getpid()}'
IMAGE = 'postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296'
SCHEMA = 'vec_contexto_actor_v1'
B11 = 'vec_contexto_actor_perfil_personal_b11_runtime'
OLD = 'vec_contexto_actor_v1_runtime'
PREFIX = f'{SCHEMA}.'
ACCOUNT = 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa'
PROFILE = 'prf_sintetico_cccccccccccccccccccccccc'
SELECT = 'spp_sintetica_iiiiiiiiiiiiiiiiiiiiiiii'
SOURCE = 'prc_fixture_sintetico_no_corporativo_01'
RESOLVE = PREFIX+'resolver_y_registrar_contexto_actor_perfil_personal_b11_v1'
RECONCILE = PREFIX+'reconciliar_contexto_actor_perfil_personal_b11_v1'
CREDENTIAL = PREFIX+'acreditar_runtime_contexto_actor_perfil_personal_b11_v1()'
LEGACY_CREDENTIAL = PREFIX+'acreditar_runtime_contexto_actor_v1()'
checks = 0
sessions = []


def run(cmd, **kw):
    return subprocess.run(cmd, text=True, capture_output=True, **kw)


def command(user='postgres', app=None):
    cmd = ['docker', 'exec', '-i']
    if app:
        cmd += ['-e', 'PGAPPNAME='+app]
    return cmd+[NAME, 'psql', '-h', '127.0.0.1', '-XqAt', '-v', 'ON_ERROR_STOP=1', '-v', 'VERBOSITY=verbose', '-U', user, '-d', 'b11']


def sql(text, user='postgres', error=None):
    result = run(command(user), input=text, timeout=20)
    if error:
        assert result.returncode and error in result.stderr, (error, result.stdout, result.stderr)
    else:
        assert result.returncode == 0, (result.stdout, result.stderr)
    return result.stdout.strip()


def check(label):
    global checks
    checks += 1
    print(f'OK {checks}: {label}', flush=True)


def file(name, error=None):
    return sql((ROOT/name).read_text(), error=error)


def digest():
    result = run(['docker','exec',NAME,'pg_dump','-U','postgres','-d','b11'], timeout=20)
    assert result.returncode == 0, result.stderr
    return hashlib.sha256('\n'.join(line for line in result.stdout.splitlines()
        if not line.startswith(('\\restrict ', '\\unrestrict '))).encode()).hexdigest()


def begin(app, text, user='postgres'):
    process = subprocess.Popen(command(user, app), stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    sessions.append(process)
    process.stdin.write(text+'\n')
    process.stdin.flush()
    return process


def state(app, predicate):
    deadline = time.monotonic()+8
    while time.monotonic()<deadline:
        if sql(f"select count(*) from pg_stat_activity where application_name='{app}' and ({predicate});") == '1':
            return
        time.sleep(.025)
    raise AssertionError('sesion no llego al estado '+app+' '+predicate)


def finish(process, text='', error=None):
    if text:
        process.stdin.write(text+'\n')
        process.stdin.flush()
    out, err = process.communicate(timeout=15)
    if error:
        assert process.returncode and error in err, (out,err)
    else:
        assert process.returncode == 0, (out,err)
    return out.strip()


def args(op, timestamp, registro=None):
    return f"'oca_{op:024d}','rca_{(registro if registro is not None else op):024d}','{ACCOUNT}','certificado','alto','{timestamp}'::timestamptz"


def resolve(op, timestamp, registro=None):
    return f'SELECT registro_contexto_ref FROM {RESOLVE}({args(op,timestamp,registro)});'


def reconcile(op, timestamp, registro=None):
    return f'SELECT registro_contexto_ref FROM {RECONCILE}({args(op,timestamp,registro)});'


def publish(version, status='activo', until="clock_timestamp()+interval '1 hour'"):
    return f"""SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SELECT {PREFIX}publicar_seleccion_perfil_personal_b11_v1(
'{SELECT}',{version},'{ACCOUNT}','{PROFILE}','{SOURCE}',1,repeat('1',64),'{status}',
clock_timestamp()-interval '1 hour',{until});
RESET ROLE;"""


try:
    result=run(['docker','run','--detach','--rm','--name',NAME,'-e','POSTGRES_HOST_AUTH_METHOD=trust','-e','POSTGRES_DB=b11',IMAGE], timeout=30)
    assert result.returncode == 0, result.stderr
    for _ in range(100):
        if run(['docker','exec',NAME,'pg_isready','-h','127.0.0.1','-U','postgres','-d','b11']).returncode == 0:
            break
        time.sleep(.1)
    assert sql("SELECT current_setting('server_version_num')")=='180004'
    sql('''CREATE ROLE b11_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
ALTER DATABASE b11 OWNER TO b11_owner;
REVOKE ALL ON DATABASE b11,postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;''')
    for name in ['roles_up.sql','migraciones/000001_contexto_actor_v1.up.sql',
                 'migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql',
                 'roles_contexto_corporativo_rrhh_selector_v1_up.sql',
                 'migraciones/000003_organizacion_corporativa_v1.up.sql']:
        file(name)
    old_four=run(command(),input=(ROOT/'migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql').read_text(),timeout=20)
    if old_four.returncode:
        assert 'manifiesto simbolico del predecesor no acreditado' in old_four.stderr,old_four.stderr
        print('INCIDENCIA HISTORICA 000004 (sin bypass): '+old_four.stderr.strip().replace('\n',' | '),flush=True)
    for name in ['roles_historicos_up.sql','migraciones/000005_lectura_contexto_historico_v2.up.sql','roles_perfil_personal_b11_up.sql']:
        file(name)
    sql(f'''CREATE ROLE b11_legacy LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT {OLD} TO b11_legacy WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;''')
    assert sql('SELECT * FROM '+LEGACY_CREDENTIAL,'b11_legacy') == 'b11_legacy|t'
    guard_before=sql(f"select oid::text||'|'||proacl::text||'|'||proowner::text from pg_proc where oid='{PREFIX}exigir_runtime_contexto_actor_v1()'::regprocedure")
    before=digest()
    file('migraciones/000007_seleccion_perfil_personal_b11_v1.up.sql')
    assert sql(f"select oid::text||'|'||proacl::text||'|'||proowner::text from pg_proc where oid='{PREFIX}exigir_runtime_contexto_actor_v1()'::regprocedure")==guard_before
    assert sql('SELECT * FROM '+LEGACY_CREDENTIAL,'b11_legacy') == 'b11_legacy|t'
    check('UP conserva OID/ACL/propietario y acreditacion runtime historico')
    down=(ROOT/'migraciones/000007_seleccion_perfil_personal_b11_v1.down.sql').read_text()
    confirmation="SET vec.confirmar_retirada_seleccion_perfil_personal_b11_v1='RETIRAR_SELECCION_PERFIL_PERSONAL_B11_V1';\n"
    installed=digest()
    sql(down,error='55000')
    assert digest()==installed
    sql(confirmation+down)
    assert digest()==before
    check('DOWN vacio restaura definicion y catalogo exactos; sin confirmacion rechaza intacto')
    file('roles_perfil_personal_b11_down.sql')
    assert sql(f"select to_regrole('{B11}') is null")=='t'
    file('roles_perfil_personal_b11_up.sql')
    file('roles_perfil_personal_b11_up.sql',error='55000')
    file('migraciones/000007_seleccion_perfil_personal_b11_v1.up.sql')
    file('migraciones/000007_seleccion_perfil_personal_b11_v1.up.sql',error='55000')
    check('ciclo roles/000007 y reentrada rechazada')
    sql(f'''CREATE ROLE b11_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT {B11} TO b11_login WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;''')
    assert sql('SELECT * FROM '+CREDENTIAL,'b11_login') == 'b11_login|t'
    assert sql(f"select count(*) from pg_proc p join pg_namespace n on n.oid=p.pronamespace where n.nspname='{SCHEMA}' and has_function_privilege('b11_login',p.oid,'EXECUTE')")=='3'
    assert sql(f"select count(*) from pg_proc p join pg_namespace n on n.oid=p.pronamespace where n.nspname='{SCHEMA}' and has_function_privilege('b11_legacy',p.oid,'EXECUTE')")=='3'
    assert sql(f"select count(*) from pg_class where relnamespace='{SCHEMA}'::regnamespace and relname like 'seleccion_perfil_personal_b11_%' and relkind='r' and relrowsecurity and relforcerowsecurity")=='3'
    for query,user in [('SELECT * FROM '+LEGACY_CREDENTIAL,'b11_login'),
                       ('SELECT * FROM '+CREDENTIAL,'b11_legacy'),
                       (f'SELECT * FROM {PREFIX}registros_contexto','b11_login'),
                       (f'SELECT {PREFIX}exigir_runtime_exacto_b11(false)','b11_login'),
                       (f'SELECT {PREFIX}publicar_seleccion_perfil_personal_b11_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)','b11_login')]:
        sql(query,user,error='42501')
    check('solo tres funciones por runtime, RLS forzada y denegaciones de tablas/helpers/superficie ajena')
    for damage in [
        'ALTER ROLE b11_login SUPERUSER;',
        'ALTER ROLE b11_login NOBYPASSRLS NOINHERIT;',
        'ALTER ROLE b11_login CREATEDB;',
        'ALTER ROLE b11_login CREATEROLE;',
        'ALTER ROLE b11_login REPLICATION;',
        'ALTER ROLE b11_login BYPASSRLS;',
        "ALTER ROLE b11_login SET statement_timeout='1s';",
        f'ALTER ROLE {B11} LOGIN;',
        f'ALTER ROLE {B11} INHERIT;',
        f'GRANT {B11} TO b11_login WITH ADMIN TRUE;',
        f'GRANT {B11} TO b11_login WITH SET TRUE;',
        f'GRANT {OLD} TO b11_login WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;',
        f'GRANT USAGE ON SCHEMA {SCHEMA} TO b11_login;',
        f'GRANT SELECT ON {PREFIX}registros_contexto TO {B11};',
        f'GRANT EXECUTE ON FUNCTION {PREFIX}referencia_valida(text,text) TO {B11};',
        f'GRANT EXECUTE ON FUNCTION {CREDENTIAL} TO {B11} WITH GRANT OPTION;',
    ]:
        sql('BEGIN;'+damage+'SET SESSION AUTHORIZATION b11_login;SELECT * FROM '+CREDENTIAL,error='42501')
    assert sql('SELECT * FROM '+CREDENTIAL,'b11_login')=='b11_login|t'
    check('16 contaminaciones ACL/atributos/membresias denegadas sin relajar la guarda')
    # Fixture de autoridad simulada exclusivamente para prueba; no fuente real.
    fixture=(ROOT/'pruebas_sql/fixtures_sinteticos.sql').read_text().replace("'no_autoritativa'","'autoridad_maestra_acreditada'")
    sql(fixture)
    sql('BEGIN;'+publish(1)+'COMMIT;')
    generation=sql(f'SELECT generacion FROM {PREFIX}control_generacion_punteros_actuales_v2')
    assert int(generation)>0
    t=sql('select clock_timestamp()')
    receipt=sql('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(1,t)+'COMMIT;','b11_login')
    assert receipt=='rca_000000000000000000000001'
    assert sql('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(1,t,999)+'COMMIT;','b11_login')==receipt
    assert sql(reconcile(1,t),'b11_login')==receipt
    assert sql(reconcile(1,t,999),'b11_login')==''
    assert sql(f'SELECT count(*) FROM {PREFIX}registros_contexto')=='1'
    assert sql(f'SELECT count(*) FROM {PREFIX}seleccion_perfil_personal_b11_usos')=='1'
    check('alta atomica, replay conserva recibo, reconciliacion exacta, sin duplicados')
    for query,error in [
        (resolve(1,t),'25000'),
        ('BEGIN ISOLATION LEVEL SERIALIZABLE;'+reconcile(1,t),'25000'),
        (f"BEGIN ISOLATION LEVEL SERIALIZABLE;SELECT * FROM {RESOLVE}(NULL,NULL,NULL,NULL,NULL,NULL)",'22023'),
        ('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(1,t).replace("'alto'","'bajo'"),'23505'),
    ]:
        sql(query,'b11_login',error=error)
    legacy_t=sql('select clock_timestamp()')
    legacy_args=args(2,legacy_t).replace(f"'{ACCOUNT}',",f"'{ACCOUNT}','{PROFILE}',")
    assert sql(f'BEGIN ISOLATION LEVEL SERIALIZABLE;SELECT registro_contexto_ref FROM {PREFIX}resolver_y_registrar_contexto_actor_v2({legacy_args});COMMIT;','b11_legacy')=='rca_000000000000000000000002'
    sql('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(2,legacy_t),'b11_login',error='23505')
    assert sql(reconcile(2,legacy_t),'b11_login')==''
    check('aislamiento/entrada/colision; B11 no adopta un registro del runtime historico')
    # Cambios actuales vuelven irrecuperable el recibo; la transaccion falla y
    # restaura la fixture para las pruebas posteriores.
    for pointer,versions,key in [('perfil_actual','perfil_versiones','perfil_ref'),('proyeccion_cuenta_actual','proyeccion_cuenta_versiones','cuenta_ref'),('persona_actual','persona_versiones','persona_ref'),('vinculo_contexto_actual','vinculo_contexto_versiones','vinculo_ref'),('vinculo_referencia_actual','vinculo_referencia_versiones','vinculo_ref')]:
        cols=sql(f"select string_agg(attname,',' order by attnum) from pg_attribute where attrelid='{PREFIX}{versions}'::regclass and attnum>0 and not attisdropped")
        expr=','.join('2' if col=='version' else "'revocado'" if col=='estado' else col for col in cols.split(','))
        mutation=f'INSERT INTO {PREFIX}{versions}({cols}) SELECT {expr} FROM {PREFIX}{versions} WHERE version=1;UPDATE {PREFIX}{pointer} SET version=2;'
        sql('BEGIN ISOLATION LEVEL SERIALIZABLE;SET LOCAL ROLE vec_contexto_actor_v1_propietario;'+mutation+'RESET ROLE;SET SESSION AUTHORIZATION b11_login;'+resolve(1,t),error='P0002')
        sql('BEGIN;SET LOCAL ROLE vec_contexto_actor_v1_propietario;'+mutation+'RESET ROLE;SET SESSION AUTHORIZATION b11_login;'+reconcile(1,t),error='P0002')
    check('replay y reconciliacion revalidan cuenta/perfil/persona/contexto/referencias actuales')
    # Commit ambiguo real: la reconciliacion comienza antes de COMMIT y debe
    # esperar el advisory de la escritura y adquirir otro snapshot RC.
    t3=sql('select clock_timestamp()')
    a=begin('b11_pending','BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(3,t3),'b11_login')
    state('b11_pending',"state='idle in transaction'")
    b=begin('b11_reconcile',reconcile(3,t3),'b11_login')
    state('b11_reconcile',"wait_event_type='Lock'")
    observed=finish(a,'COMMIT;')
    reconciled=finish(b)
    assert observed==reconciled=='rca_000000000000000000000003'
    assert sql(f"SELECT count(*) FROM {PREFIX}seleccion_perfil_personal_b11_usos WHERE operacion_ref='oca_000000000000000000000003'")=='1'
    check('reconciliacion concurrente espera COMMIT y devuelve exactamente el recibo durable')
    # DOWN con historia nunca borra: comparar dump completo.
    durable=digest()
    sql(confirmation+down,error='55000')
    assert digest()==durable
    file('roles_perfil_personal_b11_down.sql',error='55000')
    assert digest()==durable
    for mutation in [f'UPDATE {PREFIX}seleccion_perfil_personal_b11_usos SET version=2',
                     f'DELETE FROM {PREFIX}seleccion_perfil_personal_b11_versiones',
                     f'TRUNCATE {PREFIX}seleccion_perfil_personal_b11_usos',
                     f'DELETE FROM {PREFIX}seleccion_perfil_personal_b11_actual']:
        sql('BEGIN;SET LOCAL ROLE vec_contexto_actor_v1_propietario;'+mutation,error='55000')
    check('DOWN/roles e historia append-only rechazados con dump conservado')
    # Revocacion gana el advisory global; el snapshot anterior no puede usar
    # la generacion invisible tras esperar. Nunca acepta seleccion revocada.
    a=begin('b11_revoke','BEGIN;'+publish(2,'revocado'))
    state('b11_revoke',"state='idle in transaction'")
    t4=sql('select clock_timestamp()')
    b=begin('b11_stale','BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(4,t4),'b11_login')
    state('b11_stale',"wait_event_type='Lock'")
    finish(a,'COMMIT;')
    finish(b,error='40001')
    sql('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(4,sql('select clock_timestamp()')),'b11_login',error='P0002')
    assert sql(f"SELECT count(*) FROM {PREFIX}registros_contexto WHERE operacion_ref='oca_000000000000000000000004'")=='0'
    sql('BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(1,t),'b11_login',error='23505')
    assert sql(reconcile(1,t),'b11_login')==''
    check('carrera revocacion/generacion: 40001, retry denegado, recibos antiguos no recuperables')
    # Caducidad durante espera: un unico reloj de negocio posterior a locks.
    sql('BEGIN;'+publish(3,until="clock_timestamp()+interval '1 second'")+'COMMIT;')
    a=begin('b11_expire_barrier',"BEGIN;SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));")
    state('b11_expire_barrier',"state='idle in transaction'")
    t5=sql('select clock_timestamp()')
    b=begin('b11_expire','BEGIN ISOLATION LEVEL SERIALIZABLE;'+resolve(5,t5),'b11_login')
    state('b11_expire',"wait_event_type='Lock'")
    time.sleep(1.1)
    finish(a,'COMMIT;')
    finish(b,error='P0002')
    assert sql(f"SELECT count(*) FROM {PREFIX}registros_contexto WHERE operacion_ref='oca_000000000000000000000005'")=='0'
    check('caducidad tras espera revierte contexto y uso en la misma transaccion')
    print(f'PASS: {checks} grupos de pruebas; PostgreSQL 18.4 desechable.',flush=True)
finally:
    for process in sessions:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
    run(['docker','rm','-f',NAME],timeout=20)
