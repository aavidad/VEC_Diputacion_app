#!/usr/bin/env python3
"""Ensayo AD133 en dos clones sintéticos exclusivos; todas las pruebas revierten."""
import json
import re
import subprocess
import sys
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DIR = ROOT / 'deploy/postgresql/autorizacion_atestada_v3'
FIRMA = ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna'
         '(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
HELPER = 'vec_autorizacion_atestada_v3.bytea_igual_constante(bytea,bytea)'


def literal(value):
    return "'" + value.replace("'", "''") + "'"


def sql(container, command):
    return subprocess.run(['docker', 'exec', '-i', container, 'psql', '-U',
                           'postgres', '-d', 'postgres', '-X', '-At',
                           '-v', 'ON_ERROR_STOP=1', '-f', '-'], input=command,
                          text=True, capture_output=True, timeout=180)


def validate(container):
    if not re.fullmatch(r'vec-[a-zA-Z0-9_.-]{1,100}', container):
        raise RuntimeError('Nombre de clon inválido')
    result = subprocess.run(['docker', 'inspect', '--format',
                             '{{json .HostConfig}}', container], text=True,
                            capture_output=True, check=True, timeout=10)
    config = json.loads(result.stdout)
    if (config['NetworkMode'] != 'none' or not config['ReadonlyRootfs']
            or not 0 < config['Memory'] <= 4 * 1024**3
            or not 0 < config['PidsLimit'] <= 256
            or not 0 < config['NanoCpus'] <= 2 * 10**9):
        raise RuntimeError('Se requiere clon aislado, raíz de solo lectura y límites explícitos')
    version = sql(container, 'SHOW server_version_num;')
    if version.returncode or not re.fullmatch(r'18\d{4}\n', version.stdout):
        raise RuntimeError('Se requiere PostgreSQL18')


def snapshot(container):
    result = sql(container, """SELECT encode(sha256(convert_to(coalesce(
      string_agg(to_jsonb(p)::text,'' ORDER BY p.oid),''),'UTF8')),'hex')
      FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec\\_%' ESCAPE '\\';""")
    if result.returncode or not re.fullmatch(r'[a-f0-9]{64}\n', result.stdout):
        raise RuntimeError('No se pudo medir el catálogo de funciones')
    return result.stdout


def source(container):
    result = sql(container, 'SELECT prosrc FROM pg_catalog.pg_proc WHERE oid='
                 'pg_catalog.to_regprocedure(' + literal(FIRMA) + ');')
    if result.returncode or not 0 < len(result.stdout) <= 1024**2:
        raise RuntimeError('Preimagen externa ausente o excesiva')
    # psql añade una nueva línea a la cadena; conservar la representación real.
    return result.stdout[:-1]


def replace_source(value):
    marker = '$ad133_' + uuid.uuid4().hex + '$'
    return ("DO $veneno$ DECLARE p pg_catalog.pg_proc%ROWTYPE; d pg_catalog.text; BEGIN "
            "SELECT * INTO STRICT p FROM pg_catalog.pg_proc WHERE oid="
            "pg_catalog.to_regprocedure(" + literal(FIRMA) + "); "
            "d:=pg_catalog.pg_get_functiondef(p.oid); "
            "EXECUTE pg_catalog.replace(d,p.prosrc," + marker + value + marker + "); "
            "END $veneno$;\n")


def main():
    if len(sys.argv) != 3 or sys.argv[1] == sys.argv[2]:
        raise RuntimeError('Uso: 000133_ensayar_pg18.py clon_historico clon_ampliado')
    containers = sys.argv[1:]
    for container in containers:
        validate(container)
    migration = (DIR / 'migraciones/000133_convergencia_linaje_externo.up.sql').read_text()
    if not migration.endswith('COMMIT;\n') or migration.count('\nBEGIN;\n') != 1:
        raise RuntimeError('Envoltura AD133 inesperada')
    body = migration.replace('BEGIN;\n', '', 1)[:-len('COMMIT;\n')]
    probe = (DIR / 'pruebas_sql/000133_convergencia_linaje_externo.sql').read_text()
    if probe.count('-- AD133-CORRECTIVA-AQUI') != 1:
        raise RuntimeError('Marca de sonda inesperada')
    sources = [source(container) for container in containers]
    if sources[0] == sources[1]:
        raise RuntimeError('Los clones no representan dos linajes diferentes')
    for index, container in enumerate(containers):
        before = snapshot(container)
        positive = sql(container, probe.replace('-- AD133-CORRECTIVA-AQUI', body))
        if positive.returncode or 'AD133-CONSERVACION-17-TIPOS-HISTORIA-OK' not in positive.stdout:
            raise RuntimeError('Sonda positiva fallida: ' + positive.stderr[-1800:])
        cases = {
            'mezcla': replace_source(sources[1-index]),
            'owner': 'ALTER FUNCTION ' + FIRMA + ' OWNER TO postgres;',
            'acl': 'GRANT EXECUTE ON FUNCTION ' + FIRMA + ' TO PUBLIC;',
            'config': 'ALTER FUNCTION ' + FIRMA + ' SET search_path=pg_catalog,public;',
            'cuerpo': replace_source(sources[index] + '\n-- preimagen ajena\n'),
            'definidor': 'ALTER FUNCTION ' + FIRMA + ' SECURITY INVOKER;',
            'cierre_parcial': 'ALTER FUNCTION ' + HELPER + ' SET search_path=pg_catalog,pg_temp;',
            'reaplicacion': body,
        }
        for name, poison in cases.items():
            result = sql(container, '\\set VERBOSITY verbose\nBEGIN;\n'
                         + poison + '\n' + body + '\nROLLBACK;\n')
            if not result.returncode or not re.search(r'55000: AD3-133:', result.stderr):
                raise RuntimeError('AD133 aceptó el caso o falló por otra causa: ' + name
                                   + '\n' + result.stderr[-1800:])
            if snapshot(container) != before:
                raise RuntimeError('AD133 no conservó el catálogo tras ROLLBACK: ' + name)
        print('AD133-ENSAYO-OK linaje=' + ('historico' if index == 0 else 'ampliado')
              + ' firmas=17 negativos=8 historia=conservada efectos=ROLLBACK')


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError, KeyError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
