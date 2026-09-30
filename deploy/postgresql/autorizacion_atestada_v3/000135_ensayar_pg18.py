#!/usr/bin/env python3
"""Ensayo AD135 en dos clones propios post-AD133; todos los efectos revierten."""
import hashlib
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
NUCLEO = ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna'
          '(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
HUELLAS_NUCLEO_ANTERIOR = {
    '9d8318c3c32cbe10a35860ab27df042321a8411bce60e8ce9bccf6e494a254c6',
    '4174cd721f27873a6a45c11899ccff553b8d3a8ff6d5841cf156953c0f3f8c73',
}


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
    result = sql(container, """SELECT encode(sha256(convert_to(jsonb_build_object(
      'funciones',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.oid) FROM pg_proc p
        JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname LIKE 'vec\\_%' ESCAPE '\\'),
      'relaciones',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_class c
        JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec\\_%' ESCAPE '\\'),
      'tipos',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.oid) FROM pg_type t
        JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname LIKE 'vec\\_%' ESCAPE '\\'),
      'dependencias',(SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
        d.refclassid,d.refobjid,d.refobjsubid,d.deptype) FROM pg_depend d
        WHERE d.classid='pg_proc'::regclass AND d.objid IN (SELECT p.oid FROM pg_proc p
          JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname LIKE 'vec\\_%' ESCAPE '\\'))
      )::text,'UTF8')),'hex');""")
    if result.returncode or not re.fullmatch(r'[a-f0-9]{64}\n', result.stdout):
        raise RuntimeError('No se pudo medir el catálogo')
    return result.stdout


def source(container):
    result = sql(container, 'SELECT prosrc FROM pg_catalog.pg_proc WHERE oid='
                 'pg_catalog.to_regprocedure(' + literal(FIRMA) + ');')
    if result.returncode or not 0 < len(result.stdout) <= 1024**2:
        raise RuntimeError('Preimagen externa ausente o excesiva')
    return result.stdout[:-1]  # psql añade una nueva línea.


def replace_source(value, signature=FIRMA):
    marker = '$ad135_' + uuid.uuid4().hex + '$'
    return ("DO $veneno$ DECLARE p pg_catalog.pg_proc%ROWTYPE; d pg_catalog.text; BEGIN "
            "SELECT * INTO STRICT p FROM pg_catalog.pg_proc WHERE oid="
            "pg_catalog.to_regprocedure(" + literal(signature) + "); "
            "d:=pg_catalog.pg_get_functiondef(p.oid); "
            "EXECUTE pg_catalog.replace(d,p.prosrc," + marker + value + marker + "); "
            "END $veneno$;\n")


def old_kernel(path):
    source_file = Path(path)
    if not 0 < source_file.stat().st_size <= 1024**2:
        raise RuntimeError('Captura del núcleo anterior ausente o excesiva')
    captured = json.loads(source_file.read_text())
    value = captured.get('prosrc')
    if captured.get('firma') != NUCLEO or not isinstance(value, str):
        raise RuntimeError('Captura del núcleo anterior incompatible')
    digest = hashlib.sha256(value.encode()).hexdigest()
    if digest != captured.get('prosrc_sha256') or digest not in HUELLAS_NUCLEO_ANTERIOR:
        raise RuntimeError('Huella del núcleo anterior incompatible')
    return value


def main():
    if len(sys.argv) != 5 or sys.argv[1] == sys.argv[2]:
        raise RuntimeError('Uso: 000135_ensayar_pg18.py clon_historico clon_ampliado '
                           'captura_nucleo_anterior_historico.json captura_nucleo_anterior_ampliado.json')
    containers = sys.argv[1:3]
    old_kernels = [old_kernel(path) for path in sys.argv[3:]]
    if old_kernels[0] == old_kernels[1]:
        raise RuntimeError('Las capturas no representan los dos núcleos anteriores')
    for container in containers:
        validate(container)
    migration = (DIR / 'migraciones/000135_raiz_externa_linajes_convergentes.up.sql').read_text()
    if not migration.endswith('COMMIT;\n') or migration.count('\nBEGIN;\n') != 1:
        raise RuntimeError('Envoltura AD135 inesperada')
    body = migration.replace('BEGIN;\n', '', 1)[:-len('COMMIT;\n')]
    probe = (DIR / 'pruebas_sql/000135_raiz_externa_linajes_convergentes.sql').read_text()
    if probe.count('-- AD135-CORRECTIVA-AQUI') != 1:
        raise RuntimeError('Marca de sonda inesperada')
    sources = [source(container) for container in containers]
    if sources[0] == sources[1]:
        raise RuntimeError('Los clones no representan dos linajes diferentes')
    for index, container in enumerate(containers):
        before = snapshot(container)
        positive = sql(container, probe.replace('-- AD135-CORRECTIVA-AQUI', body))
        if positive.returncode or 'AD135-RAIZ-LINAJE-CONSERVACION-OK' not in positive.stdout:
            raise RuntimeError('Sonda positiva fallida: ' + positive.stderr[-1800:])
        if snapshot(container) != before:
            raise RuntimeError('La sonda positiva no revirtió el catálogo')
        cases = {
            'mezcla': replace_source(sources[1-index]),
            # Los cuatro lectores coinciden con AD133 nueva, pero el núcleo
            # carece de AD134/136: la guarda global debe rechazarlo antes del DDL.
            'ad133_anterior': replace_source(old_kernels[index], NUCLEO),
            'owner': 'ALTER FUNCTION ' + FIRMA + ' OWNER TO postgres;',
            'acl': 'GRANT EXECUTE ON FUNCTION ' + FIRMA + ' TO PUBLIC;',
            'config': 'ALTER FUNCTION ' + FIRMA + ' SET search_path=pg_catalog,public;',
            'cuerpo': replace_source(sources[index] + '\n-- preimagen ajena\n'),
            'definidor': 'ALTER FUNCTION ' + FIRMA + ' SECURITY INVOKER;',
            'lector_transformado': replace_source(sources[index].replace(
                'vec_autorizacion_atestada_v3.puntero_configuracion_actual',
                'vec_autorizacion_atestada_v3.puntero_configuracion_externa')),
            'tabla_parcial': 'CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision_externa(x int);',
            'tipo_parcial': 'CREATE DOMAIN vec_autorizacion_atestada_v3.checkpoint_gobierno_externo AS int;',
            'funcion_parcial': ('CREATE FUNCTION vec_autorizacion_atestada_v3.material_publico_externo_v1(int) '
                                'RETURNS int LANGUAGE sql AS $$SELECT $1$$;'),
            'reaplicacion': body + '\nRESET ROLE;\n',
        }
        for name, poison in cases.items():
            result = sql(container, '\\set VERBOSITY verbose\nBEGIN;\n'
                         + poison + '\n' + body + '\nROLLBACK;\n')
            if not result.returncode or not re.search(r'55000: AD3-135:', result.stderr):
                raise RuntimeError('AD135 aceptó el caso o falló por otra causa: ' + name
                                   + '\n' + result.stderr[-1800:])
            if snapshot(container) != before:
                raise RuntimeError('AD135 no conservó el catálogo tras ROLLBACK: ' + name)
        print('AD135-ENSAYO-OK linaje=' + ('historico' if index == 0 else 'ampliado')
              + ' lectores=4 negativos=12 gobierno=aislado efectos=ROLLBACK')


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError, KeyError, OSError, ValueError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
