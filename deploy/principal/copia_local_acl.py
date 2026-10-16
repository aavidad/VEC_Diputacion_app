#!/usr/bin/env python3
"""Genera comprobaciones de ACL para los dos anexos oficiales de HZ37."""

import hashlib
import re
import sys
from pathlib import Path


HASH_BASE = 'ab21b92e6f81a838c8c502d431de66454e8c47e7d7fca0becc01882d6d3584aa'
HASH_TIPOS = 'e058a00b9e0c8c9f06c2fc76d33882637ab7fb715618282305237c1ea959ea71'
ROLE = r'"((?:[^"]|"")+)"'
NAME = r'([a-z_][a-z_0-9]*)'
BASE_GRANT = re.compile(rf'GRANT\s+((?:CONNECT|CREATE)(?:\s*,\s*(?:CONNECT|CREATE))?)\s+ON\s+DATABASE\s+postgres\s+TO\s+{ROLE};', re.I)
BASE_REVOKE = re.compile(r'REVOKE\s+ALL\s+ON\s+DATABASE\s+postgres\s+FROM\s+PUBLIC;', re.I)
TYPE_REVOKE = re.compile(rf'REVOKE\s+ALL\s+ON\s+TYPE\s+{NAME}\.{NAME}\s+FROM\s+PUBLIC;', re.I)
B117 = (('vec_bolsa_llamamientos', 'origen_oferta_ct_v1'),
        ('vec_bolsa_llamamientos', 'origen_oferta_ct_outbox_v1'))

def statements(source):
    return [line.strip() for line in source.splitlines() if line.strip() and not line.lstrip().startswith('--')]


def parse_base(source):
    grants, revoked = set(), 0
    lines = statements(source)
    if 'BEGIN;' in lines or 'COMMIT;' in lines:
        if lines[:1] != ['BEGIN;'] or lines[-1:] != ['COMMIT;'] or lines.count('BEGIN;') != 1 or lines.count('COMMIT;') != 1:
            raise ValueError('transacción del anexo incompleta')
        lines = lines[1:-1]
    for line in lines:
        match = BASE_GRANT.fullmatch(line)
        if match:
            name = match[2].replace('""', '"')
            if not name or any(ord(c) < 32 for c in name):
                raise ValueError('rol inválido')
            privileges = [part.strip().upper() for part in match[1].split(',')]
            if len(privileges) != len(set(privileges)) or any((name, part) in grants for part in privileges):
                raise ValueError('privilegio duplicado')
            grants.update((name, part) for part in privileges)
        elif BASE_REVOKE.fullmatch(line):
            revoked += 1
        else:
            raise ValueError('sentencia de base no admitida')
    if revoked != 1 or not grants:
        raise ValueError('anexo de base incompleto')
    return grants


def parse_types(source):
    names = set()
    for line in statements(source):
        match = TYPE_REVOKE.fullmatch(line)
        if not match:
            raise ValueError('sentencia de tipos no admitida')
        names.add((match[1].lower(), match[2].lower()))
    if not names or len(names) != len(statements(source)):
        raise ValueError('anexo de tipos incompleto o duplicado')
    return names


def literal(value):
    return "'" + value.replace("'", "''") + "'"


def render(grants, types, allow_new):
    expected = {('postgres', privilege, 'false') for privilege in ('CONNECT', 'CREATE', 'TEMPORARY')}
    expected.update((name, privilege, 'false') for name, privilege in grants)
    base_rows = ',\n    '.join(f'({literal(name)}, {literal(privilege)}, {grantable})'
                             for name, privilege, grantable in sorted(expected))
    type_rows = ', '.join(literal(f'{schema}.{name}') for schema, name in sorted(types))
    extra_check = '' if allow_new else "IF v_extra THEN RAISE EXCEPTION 'La base tiene concesiones ajenas al anexo HZ37'; END IF;"
    return f'''BEGIN READ ONLY;
SET LOCAL search_path = pg_catalog, pg_temp;
DO $comprobacion$
DECLARE
  v_owner oid;
  v_missing boolean;
  v_extra boolean;
  v_types text[] := ARRAY[{type_rows}];
BEGIN
  SELECT d.datdba INTO v_owner FROM pg_catalog.pg_database d WHERE d.datname = 'postgres';
  IF v_owner IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = 'postgres') THEN
    RAISE EXCEPTION 'El propietario de la base postgres no coincide';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_database d,
      LATERAL pg_catalog.aclexplode(COALESCE(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
    WHERE d.datname = 'postgres' AND a.grantee = 0
  ) THEN
    RAISE EXCEPTION 'PUBLIC conserva privilegios de la base postgres';
  END IF;
  WITH expected(grantee, privilege_type, is_grantable) AS (VALUES
    {base_rows}
  ), actual AS (
    SELECT COALESCE(r.rolname, 'PUBLIC') AS grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_database d
    CROSS JOIN LATERAL pg_catalog.aclexplode(
      COALESCE(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
    LEFT JOIN pg_catalog.pg_roles r ON r.oid = a.grantee
    WHERE d.datname = 'postgres'
  )
  SELECT EXISTS(SELECT * FROM expected EXCEPT ALL SELECT * FROM actual),
         EXISTS(SELECT * FROM actual EXCEPT ALL SELECT * FROM expected)
    INTO v_missing, v_extra;
  IF v_missing THEN RAISE EXCEPTION 'Faltan concesiones de base del anexo HZ37'; END IF;
  {extra_check}
  IF EXISTS (SELECT 1 FROM unnest(v_types) wanted
    WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type t
      JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
      WHERE n.nspname || '.' || t.typname = wanted)) THEN
    RAISE EXCEPTION 'Falta un tipo del anexo HZ37';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_type t
    JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
    LEFT JOIN LATERAL pg_catalog.aclexplode(
      COALESCE(t.typacl, pg_catalog.acldefault('T', t.typowner))) a ON true
    WHERE n.nspname || '.' || t.typname = ANY(v_types)
       OR (n.nspname = 'vec_bolsa_llamamientos' AND t.typname IN
         ('origen_oferta_ct_v1', 'origen_oferta_ct_outbox_v1'))
    GROUP BY t.oid, t.typowner
    HAVING count(a.grantee) <> 1 OR bool_or(a.grantee <> t.typowner
      OR a.privilege_type <> 'USAGE' OR a.is_grantable)) THEN
    RAISE EXCEPTION 'ACL de tipo distinta de USAGE del propietario';
  END IF;
END $comprobacion$;
ROLLBACK;
'''


def main(argv):
    if len(argv) not in (3, 4) or (len(argv) == 4 and argv[3] != '--permitir-nuevas-concesiones'):
        print('Uso: copia_local_acl.py acl-base.sql acl-tipos.sql [--permitir-nuevas-concesiones]', file=sys.stderr)
        return 2
    try:
        raw_base, raw_types = (Path(path).read_bytes() for path in argv[1:3])
        if hashlib.sha256(raw_base).hexdigest() != HASH_BASE or hashlib.sha256(raw_types).hexdigest() != HASH_TIPOS:
            raise ValueError('hash')
        grants = parse_base(raw_base.decode('utf-8'))
        types = parse_types(raw_types.decode('utf-8'))
    except (OSError, UnicodeError, ValueError):
        print('Los anexos HZ37 no coinciden o contienen SQL no admitido.', file=sys.stderr)
        return 1
    sys.stdout.write(render(grants, types, len(argv) == 4))
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
