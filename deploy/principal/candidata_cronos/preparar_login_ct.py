#!/usr/bin/env python3
"""Prepara conexiones CT/Bolsa solo en una copia Cronos autorizada."""

import hashlib
import json
import os
from pathlib import Path
import secrets
import shlex
import subprocess
from urllib.parse import parse_qsl, quote, urlencode, urlsplit


CONEXIONES = {
    'VEC_CT_DATABASE_URL': 'vec_ct_o207_runtime',
    'VEC_CT_GOBIERNO_DATABASE_URL': 'vec_ad3_o207_gobierno',
    'VEC_CT_CONFIRMADOR_DATABASE_URL': 'vec_ct_o207_confirmador',
    'VEC_CT_LECTOR_RESULTADO_DATABASE_URL': 'vec_ct_o207_lector',
    'VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL': 'vec_autorizacion_o207_registro',
    'VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL': 'vec_bolsa_llamamientos_desarrollo',
    'VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL': 'vec_bolsa_importacion_convoca_desarrollo',
    'VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL': 'vec_b2_auditoria_frontera_desarrollo',
    'VEC_CALENDARIOS_DATABASE_URL': 'vec_calendarios_lector_desarrollo',
    'VEC_CT_CONSULTAS_RRHH_DATABASE_URL': 'vec_ct_rrhh_consultas_desarrollo',
    'VEC_CT_MOTIVOS_RRHH_DATABASE_URL': 'vec_ct_rrhh_motivos_desarrollo',
    'VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL': 'vec_ct_identidad_registro_desarrollo',
    'VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL': 'vec_ct_identidad_revalidacion_desarrollo',
    'VEC_CT_CONTEXTO_ACTOR_DATABASE_URL': 'vec_ct_contexto_actor_desarrollo',
}


def ejecutar(*args, input=None):
    return subprocess.run(args, input=input, capture_output=True, check=True)


def consultar(contenedor, sql):
    return ejecutar('docker', 'exec', '-u', 'postgres', contenedor,
                   'psql', '-X', '-At', '-U', 'postgres', '-d', 'postgres', '-c', sql).stdout.decode()


def main():
    os.umask(0o077)
    estado = Path(os.environ['VEC_CRONOS_STATE_DIR'])
    contenedor = os.environ['VEC_CRONOS_CONTAINER']
    puerto = int(os.environ.get('VEC_CRONOS_PORT', '55451'))
    if not estado.is_absolute() or estado.is_symlink() or not estado.is_dir() or \
            (estado.stat().st_mode & 0o777) != 0o700 or \
            not contenedor.startswith('vec-codexc-cronos-pg-') or \
            not (1 <= puerto <= 65535):
        raise SystemExit('Destino privado o contenedor Cronos no válido')
    ambiente = estado / 'arranque-base-ct.env'
    secretos = estado / 'ct-login-local.json'
    if not ambiente.is_file() or secretos.exists() or (estado / 'rotacion-ct-local.sql').exists():
        raise SystemExit('Falta el entorno base o ya se prepararon los LOGIN locales')
    lineas = ambiente.read_text().splitlines()
    if len(lineas) != 7 or not (estado / 'pg-tls/o207-ca.crt').is_file():
        raise SystemExit('El entorno base o la CA local están incompletos')
    auditoria_fichero = (estado / 'runtime/incorporacion-servidor-20260910/'
                        'config/auditoria_frontera.dsn')
    if not auditoria_fichero.is_file():
        raise SystemExit('Falta la conexión nominal de auditoría de frontera')
    auditoria = urlsplit(auditoria_fichero.read_text().strip())
    auditoria_query = dict(parse_qsl(auditoria.query))
    auditoria_rol = 'vec_ct_frontera_6e6498c80895ecb65781d002'
    if auditoria.username != auditoria_rol or not auditoria.password or \
            auditoria.hostname != 'localhost' or auditoria.port != puerto or \
            auditoria.path != '/postgres' or auditoria_query.get('sslmode') != 'verify-full' or \
            auditoria_query.get('sslrootcert') != str(estado / 'pg-tls/o207-ca.crt'):
        raise SystemExit('La conexión de auditoría no coincide con la copia')
    inspect = json.loads(ejecutar('docker', 'inspect', contenedor).stdout)[0]
    enlaces = inspect['HostConfig']['PortBindings'].get('5432/tcp', [])
    if inspect['State']['Running'] is not True or \
            not inspect['HostConfig']['NetworkMode'].startswith('vec-codexc-cronos-net-') or \
            len(enlaces) != 1 or enlaces[0] != {'HostIp': '127.0.0.1', 'HostPort': str(puerto)} or \
            not any(m['Source'] == str(estado / 'pgdata') for m in inspect['Mounts']):
        raise SystemExit('El contenedor no corresponde a esta copia privada')
    if len(CONEXIONES) != 14 or len(set(CONEXIONES.values())) != 14:
        raise SystemExit('Lista de LOGIN local incompleta')
    nombres = sorted(CONEXIONES.values())
    valores = ','.join("('" + nombre + "')" for nombre in nombres)
    base = 'WITH target(name) AS (VALUES ' + valores + ') '
    control = base + "SELECT r.rolname||'|'||r.rolcanlogin||'|'||r.rolsuper||'|'||r.rolcreatedb||'|'||r.rolcreaterole||'|'||r.rolreplication||'|'||r.rolbypassrls||'|'||r.rolinherit||'|'||coalesce(r.rolvaliduntil::text,'')||'|'||coalesce((SELECT string_agg(m.roleid::regrole::text||':'||m.inherit_option||':'||m.set_option||':'||m.admin_option,',' ORDER BY m.roleid) FROM pg_auth_members m WHERE m.member=r.oid),'')||'|'||has_database_privilege(r.oid,'postgres','CONNECT') FROM target t JOIN pg_roles r ON r.rolname=t.name ORDER BY r.rolname"
    verificadores = base + "SELECT a.rolname||'|'||coalesce(a.rolpassword,'') FROM target t JOIN pg_authid a ON a.rolname=t.name ORDER BY a.rolname"
    antes = consultar(contenedor, control)
    antes_claves = consultar(contenedor, verificadores)
    def minimo(linea):
        campos = linea.split('|')
        return len(campos) == 11 and campos[1] == 't' and campos[2:7] == ['f'] * 5 and campos[-1] == 't'
    if len(antes.splitlines()) != 14 or len(antes_claves.splitlines()) != 14 or \
            any(not minimo(linea) for linea in antes.splitlines()):
        raise SystemExit('Falta un LOGIN nominal, CONNECT o un atributo mínimo')
    auditoria_control = consultar(contenedor, "SELECT rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolreplication,rolbypassrls,has_database_privilege(oid,'postgres','CONNECT') FROM pg_roles WHERE rolname='" + auditoria_rol + "'").strip()
    if auditoria_control != 't|f|f|f|f|f|t':
        raise SystemExit('El LOGIN de auditoría carece del contrato mínimo')
    credenciales = {nombre: secrets.token_urlsafe(32) for nombre in nombres}
    secretos.write_text(json.dumps(credenciales, sort_keys=True) + '\n')
    secretos.chmod(0o600)
    sql_file = estado / 'rotacion-ct-local.sql'
    sql_file.write_text('BEGIN;\n' + ''.join(
        "ALTER ROLE " + nombre + " PASSWORD '" + credenciales[nombre] + "';\n"
        for nombre in nombres) + 'COMMIT;\n')
    sql_file.chmod(0o600)
    with sql_file.open('rb') as source, (estado / 'rotacion-ct-local.log').open('wb') as log:
        resultado = subprocess.run(
            ['docker', 'exec', '-i', '-u', 'postgres', contenedor,
             'psql', '-X', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres'],
            stdin=source, stdout=log, stderr=subprocess.STDOUT)
    if resultado.returncode:
        raise SystemExit('Falló la rotación local; revisar el registro privado')
    despues = consultar(contenedor, control)
    despues_claves = consultar(contenedor, verificadores)
    a = dict(line.split('|', 1) for line in antes_claves.splitlines())
    b = dict(line.split('|', 1) for line in despues_claves.splitlines())
    if antes != despues or set(a) != set(nombres) or any(a[n] == b[n] for n in nombres):
        raise SystemExit('La comprobación de LOGIN no pasó; conservar la copia para revisión')
    (estado / 'roles-baseline.sha256').write_text(hashlib.sha256(antes.encode()).hexdigest() + '\n')
    for variable, nombre in CONEXIONES.items():
        dsn = ('postgresql://' + nombre + ':' + quote(credenciales[nombre], safe='') +
               '@localhost:' + str(puerto) + '/postgres?' + urlencode({
                   'sslmode': 'verify-full',
                   'sslrootcert': str(estado / 'pg-tls/o207-ca.crt'),
               }))
        lineas.append(variable + '=' + shlex.quote(dsn))
    lineas.append('VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL=' +
                 shlex.quote(auditoria.geturl()))
    ambiente.write_text('\n'.join(lineas) + '\n')
    ambiente.chmod(0o600)
    sql_file.unlink()
    ca_interna = '/var/lib/postgresql/18/docker/o207-ca.crt'
    probe_interno = '/var/lib/postgresql/18/docker/sonda-14-login.sql'
    ejecutar('docker', 'cp', str(estado / 'pg-tls/o207-ca.crt'), contenedor + ':' + ca_interna)
    ejecutar('docker', 'exec', '-u', '0', contenedor, 'chown', 'postgres:postgres', ca_interna)
    sonda = estado / 'sonda-14-login.sql'
    with sonda.open('w') as output:
        for nombre in nombres:
            dsn = ('postgresql://' + nombre + ':' + quote(credenciales[nombre], safe='') +
                   '@localhost:5432/postgres?' + urlencode({
                       'sslmode': 'verify-full', 'sslrootcert': ca_interna,
                   }))
            output.write('\\connect ' + dsn + '\n')
            output.write("SELECT CASE WHEN ssl THEN 'TLS_VERIFIED' ELSE 'TLS_OFF' END "
                         'FROM pg_stat_ssl WHERE pid=pg_backend_pid();\n')
        auditoria_query['sslrootcert'] = ca_interna
        auditoria_interna = auditoria._replace(
            netloc=auditoria.netloc.rpartition('@')[0] + '@localhost:5432',
            query=urlencode(auditoria_query)).geturl()
        output.write('\\connect ' + auditoria_interna + '\n')
        output.write("SELECT CASE WHEN ssl THEN 'TLS_VERIFIED' ELSE 'TLS_OFF' END "
                     'FROM pg_stat_ssl WHERE pid=pg_backend_pid();\n')
    sonda.chmod(0o600)
    ejecutar('docker', 'cp', str(sonda), contenedor + ':' + probe_interno)
    ejecutar('docker', 'exec', '-u', '0', contenedor, 'chown', 'postgres:postgres', probe_interno)
    with (estado / 'sonda-14-login.log').open('wb') as log:
        resultado = subprocess.run(
            ['docker', 'exec', '-u', 'postgres', contenedor, 'psql', '-X', '-At',
             '-w', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres', '-f', probe_interno],
            stdout=log, stderr=subprocess.STDOUT)
    log = (estado / 'sonda-14-login.log').read_text()
    sonda.unlink()
    ejecutar('docker', 'exec', '-u', '0', contenedor, 'rm', probe_interno)
    if resultado.returncode or log.count('TLS_VERIFIED') != 15:
        raise SystemExit('Una conexión nominal no pasó TLS; revisar el registro privado')
    print('14 LOGIN locales preparados y uno reutilizado; 15 conexiones TLS. Entorno privado: 22 variables.')


if __name__ == '__main__':
    main()
