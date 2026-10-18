#!/usr/bin/env python3
"""Extrae grupos NOLOGIN citados por el volcado y las migraciones validadas.

Uso: python3 copia_local_roles.py esquema.sql acl-base.sql acl-tipos.sql
     [--fuente-roles roles_up.sql] [--migracion delta.up.sql] > roles.sql
Solo prepara roles NOLOGIN; no reproduce credenciales, membresías ni privilegios.
"""

import re
import sys
from dataclasses import dataclass
from pathlib import Path

# F0-31: B119 exige estos nombres antes de instalar. El respaldo HZ37 no
# los cita. Solo se crean sin LOGIN; su activación documental es posterior.
ROLES_B119 = {'vec_externo_documentos_bolsa_datos',
              'vec_interno_documentos_bolsa_rrhh'}


@dataclass(frozen=True)
class Token:
    kind: str
    value: str


class SQLNoAdmitido(ValueError):
    pass


def tokens(sql: str, *, body=False, versioned=False):
    """Separa sentencias sin interpretar el contenido de literales ni comentarios."""
    i = 0
    size = len(sql)
    statement = []
    while i < size:
        char = sql[i]
        if char.isspace():
            i += 1
            continue
        if sql.startswith('--', i):
            end = sql.find('\n', i)
            i = size if end < 0 else end + 1
            continue
        if sql.startswith('/*', i):
            depth = 1
            i += 2
            while depth and i < size:
                if sql.startswith('/*', i):
                    depth += 1
                    i += 2
                elif sql.startswith('*/', i):
                    depth -= 1
                    i += 2
                else:
                    i += 1
            if depth:
                raise SQLNoAdmitido('comentario sin cierre')
            continue
        if char == '\\' and not statement and not sql[sql.rfind('\n', 0, i) + 1:i].strip():
            end = sql.find('\n', i)
            line = sql[i:size if end < 0 else end]
            if not (re.match(r'^\\(?:un)?restrict(?:\s|$)', line)
                    or (versioned and line.strip() == r'\set ON_ERROR_STOP on')):
                raise SQLNoAdmitido('metacomando no admitido')
            i = size if end < 0 else end + 1
            continue
        if char == "'":
            # Las barras de escape corresponden a E'...' o U&'...'.
            escaped = bool(statement and statement[-1].kind == 'word'
                           and statement[-1].value.upper() in {'E', 'U&'})
            i += 1
            start = i
            while i < size:
                if escaped and sql[i] == '\\':
                    i += 2
                elif sql[i] == "'":
                    i += 1
                    if i < size and sql[i] == "'":
                        i += 1
                    else:
                        break
                else:
                    i += 1
            else:
                raise SQLNoAdmitido('literal sin cierre')
            statement.append(Token('literal', sql[start:i - 1]))
            continue
        if char == '"':
            i += 1
            name = []
            while i < size:
                if sql[i] == '"':
                    i += 1
                    if i < size and sql[i] == '"':
                        name.append('"')
                        i += 1
                    else:
                        break
                else:
                    name.append(sql[i])
                    i += 1
            else:
                raise SQLNoAdmitido('identificador sin cierre')
            statement.append(Token('identifier', ''.join(name)))
            continue
        if char == '$':
            match = re.match(r'\$[A-Za-z_][A-Za-z_0-9]*\$|\$\$', sql[i:])
            if match:
                delimiter = match.group()
                end = sql.find(delimiter, i + len(delimiter))
                if end < 0:
                    raise SQLNoAdmitido('cuerpo sin cierre')
                statement.append(Token('body', sql[i + len(delimiter):end]))
                i = end + len(delimiter)
                continue
        if char == ';':
            if statement:
                yield statement
            statement = []
            i += 1
            continue
        if char.isalpha() or char == '_':
            end = i + 1
            while end < size and (sql[end].isalnum() or sql[end] in '_$'):
                end += 1
            statement.append(Token('word', sql[i:end]))
            i = end
            continue
        statement.append(Token('symbol', char))
        i += 1
    if statement:
        if not body:
            raise SQLNoAdmitido('sentencia sin terminador')
        yield statement


def keyword(token, value):
    return token.kind == 'word' and token.value.upper() == value


def role(token):
    if token.kind not in {'word', 'identifier'}:
        raise SQLNoAdmitido('nombre de rol ambiguo')
    name = token.value
    if token.kind == 'word':
        name = name.translate(str.maketrans('ABCDEFGHIJKLMNOPQRSTUVWXYZ',
                                           'abcdefghijklmnopqrstuvwxyz'))
    if (not name or any(ord(char) < 32 or ord(char) == 127 for char in name)
            or len(name.encode('utf-8')) > 63):
        raise SQLNoAdmitido('nombre de rol inválido')
    if token.kind == 'word' and name.upper() in {'CURRENT_USER', 'CURRENT_ROLE', 'SESSION_USER'}:
        raise SQLNoAdmitido('nombre de rol dinámico')
    if token.kind == 'word' and name.upper() in {'WITH', 'GRANTED', 'CASCADE', 'RESTRICT',
                                                  'TO', 'FROM', 'ON', 'BY', 'IN'}:
        raise SQLNoAdmitido('nombre de rol ambiguo')
    if name.lower() == 'public' or name.lower() == 'postgres' or name.lower().startswith('pg_'):
        return None
    return name


def top_level_positions(parts, name):
    depth = 0
    for index, part in enumerate(parts):
        if part.value == '(' and part.kind == 'symbol':
            depth += 1
        elif part.value == ')' and part.kind == 'symbol':
            depth -= 1
            if depth < 0:
                raise SQLNoAdmitido('paréntesis ambiguos')
        elif depth == 0 and keyword(part, name):
            yield index
    if depth:
        raise SQLNoAdmitido('paréntesis sin cierre')


def acl_roles(parts, verb):
    # Un GRANT/REVOKE sin ON es una membresía, ausente en los ACL de la copia.
    on = list(top_level_positions(parts, 'ON'))
    if not on:
        return []
    marker = 'TO' if verb == 'GRANT' else 'FROM'
    choices = [i for i in top_level_positions(parts, marker) if i > on[0]]
    if len(choices) != 1:
        raise SQLNoAdmitido('ACL sin destinatario único')
    index = choices[0] + 1
    found = []
    while index < len(parts):
        current = parts[index]
        if current.kind == 'word' and current.value.upper() in {'WITH', 'GRANTED', 'CASCADE', 'RESTRICT'}:
            break
        if current.kind == 'word' and current.value.upper() == 'GROUP':
            raise SQLNoAdmitido('destinatario GROUP ambiguo')
        found.append(role(current))
        index += 1
        if index == len(parts) or parts[index].value != ',':
            break
        index += 1
        if index == len(parts):
            raise SQLNoAdmitido('lista de roles incompleta')
    if not found:
        raise SQLNoAdmitido('ACL sin destinatario')
    if index < len(parts) and keyword(parts[index], 'WITH'):
        if [p.value.upper() for p in parts[index:index + 3]] != ['WITH', 'GRANT', 'OPTION']:
            raise SQLNoAdmitido('opción ACL no admitida')
        index += 3
    if index < len(parts) and keyword(parts[index], 'GRANTED'):
        if index + 2 >= len(parts) or not keyword(parts[index + 1], 'BY'):
            raise SQLNoAdmitido('otorgante ambiguo')
        found.append(role(parts[index + 2]))
        index += 3
    if index < len(parts) and (keyword(parts[index], 'CASCADE') or keyword(parts[index], 'RESTRICT')):
        index += 1
    if index != len(parts):
        raise SQLNoAdmitido('cola ACL no admitida')
    return found


def statement_roles(parts):
    if keyword(parts[0], 'ALTER'):
        owner = [index for index in top_level_positions(parts, 'OWNER')
                 if index + 1 < len(parts) and keyword(parts[index + 1], 'TO')]
        if owner:
            index = owner[-1]
            if len(owner) != 1 or index + 3 != len(parts):
                raise SQLNoAdmitido('OWNER TO ambiguo')
            return [role(parts[index + 2])]
        if len(parts) >= 3 and keyword(parts[1], 'DEFAULT') and keyword(parts[2], 'PRIVILEGES'):
            # pg_dump puede incluir ACL predeterminadas bajo ALTER DEFAULT PRIVILEGES.
            verbs = [i for i, token in enumerate(parts) if keyword(token, 'GRANT') or keyword(token, 'REVOKE')]
            if len(verbs) != 1:
                raise SQLNoAdmitido('ACL predeterminada ambigua')
            result = acl_roles(parts[verbs[0]:], parts[verbs[0]].value.upper())
            prefix = parts[3:verbs[0]]
            index = 0
            if len(prefix) >= 2 and keyword(prefix[0], 'FOR') and (keyword(prefix[1], 'ROLE') or keyword(prefix[1], 'USER')):
                index = 2
                if index == len(prefix):
                    raise SQLNoAdmitido('rol predeterminado ausente')
                while index < len(prefix):
                    result.append(role(prefix[index]))
                    index += 1
                    if index == len(prefix) or keyword(prefix[index], 'IN'):
                        break
                    if prefix[index].value != ',':
                        raise SQLNoAdmitido('lista de roles predeterminados ambigua')
                    index += 1
                    if index == len(prefix):
                        raise SQLNoAdmitido('lista de roles predeterminados incompleta')
            if index < len(prefix):
                if not (index + 2 < len(prefix) and keyword(prefix[index], 'IN')
                        and keyword(prefix[index + 1], 'SCHEMA')):
                    raise SQLNoAdmitido('ámbito de ACL predeterminada ambiguo')
            return result
    if keyword(parts[0], 'GRANT') or keyword(parts[0], 'REVOKE'):
        return acl_roles(parts, parts[0].value.upper())
    return []


def declared_role(parts, *, require_nologin=True):
    if (len(parts) < 3 or not keyword(parts[0], 'CREATE')
            or not keyword(parts[1], 'ROLE')):
        return None
    name = role(parts[2])
    if require_nologin:
        options = parts[3:]
        if (any(keyword(part, 'LOGIN') for part in options)
                or not any(keyword(part, 'NOLOGIN') for part in options)):
            return None
    return name


def literal_role(part):
    if part.kind != 'literal':
        return None
    name = part.value.replace("''", "'")
    # Las referencias admitidas son nombres simples de rol, no expresiones SQL.
    if not re.fullmatch(r'[a-z_][a-z_0-9]*', name):
        return None
    return role(Token('word', name))


def literal_role_argument(parts):
    if len(parts) == 1:
        return literal_role(parts[0])
    if (len(parts) == 4 and parts[1].value == ':' and parts[2].value == ':'
            and any(keyword(parts[3], kind) for kind in ('TEXT', 'NAME', 'REGROLE'))):
        return literal_role(parts[0])
    return None


def referenced_roles(parts):
    found = set(name for name in statement_roles(parts) if name is not None)
    for index, part in enumerate(parts):
        if (keyword(part, 'TO_REGROLE') and index + 3 < len(parts)
                and parts[index + 1].value == '('
                and parts[index + 3].value == ')'):
            name = literal_role(parts[index + 2])
            if name:
                found.add(name)
        if (part.kind == 'literal' and index + 3 < len(parts)
                and parts[index + 1].value == ':' and parts[index + 2].value == ':'
                and keyword(parts[index + 3], 'REGROLE')):
            name = literal_role(part)
            if name:
                found.add(name)
        if (keyword(part, 'PG_HAS_ROLE') and index + 4 < len(parts)
                and parts[index + 1].value == '('):
            # Dos argumentos: rol y modo; tres: usuario, rol y modo.
            depth = 0
            arguments = []
            current = []
            for pos in range(index + 2, len(parts)):
                if parts[pos].value == '(':
                    depth += 1
                elif parts[pos].value == ')':
                    if depth == 0:
                        arguments.append(current)
                        break
                    depth -= 1
                if parts[pos].value == ',' and depth == 0:
                    arguments.append(current)
                    current = []
                    continue
                current.append(parts[pos])
            target = arguments[1] if len(arguments) == 3 else arguments[0] if len(arguments) == 2 else []
            name = literal_role_argument(target)
            if name:
                found.add(name)
        if part.kind == 'body':
            for nested in tokens(part.value, body=True):
                found.update(referenced_roles(nested))
    return found


def declarations(paths, *, require_nologin=True):
    names = set()
    for path in paths:
        for statement in tokens(path.read_text(encoding='utf-8'), versioned=True):
            names.update(statement_declarations(statement, require_nologin=require_nologin))
    return names


def statement_declarations(parts, *, require_nologin=True):
    found = set()
    for index in range(len(parts) - 2):
        if keyword(parts[index], 'CREATE') and keyword(parts[index + 1], 'ROLE'):
            name = declared_role(parts[index:], require_nologin=require_nologin)
            if name:
                found.add(name)
    for part in parts:
        if part.kind == 'body':
            for nested in tokens(part.value, body=True):
                found.update(statement_declarations(nested, require_nologin=require_nologin))
    return found


def supplementary_roles(schema_paths, sources, migrations):
    canonical = declarations(sources)
    cited = set()
    for path in [*schema_paths, *migrations]:
        for statement in tokens(path.read_text(encoding='utf-8'), versioned=path in migrations):
            cited.update(referenced_roles(statement))
    return (canonical & cited) - declarations(migrations, require_nologin=False)


def extract(paths):
    names = set()
    for path in paths:
        for statement in tokens(path.read_text(encoding='utf-8')):
            names.update(name for name in statement_roles(statement) if name is not None)
    return sorted(names)


def main(argv):
    import argparse
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('volcados', type=Path, nargs=3)
    parser.add_argument('--fuente-roles', type=Path, action='append', default=[])
    parser.add_argument('--migracion', type=Path, action='append', default=[])
    parser.add_argument('--roles-b119', action='store_true')
    try:
        args = parser.parse_args(argv[1:])
    except SystemExit as exc:
        return exc.code
    if args.migracion and not args.fuente_roles:
        print('Faltan declaraciones canónicas de roles.', file=sys.stderr)
        return 2
    try:
        names = sorted(set(extract(args.volcados))
                       | supplementary_roles(args.volcados, args.fuente_roles, args.migracion)
                       | (ROLES_B119 if args.roles_b119 else set()))
    except (OSError, UnicodeError, SQLNoAdmitido) as exc:
        # Los ficheros de origen pueden ser privados; nunca mostrar ruta ni contenido.
        print(f'No se pudo preparar el SQL de roles ({type(exc).__name__}).', file=sys.stderr)
        return 1
    if names:
        delimiter = '$vec_copia_roles$'
        while any(delimiter in name for name in names):
            delimiter = delimiter[:-1] + '_$'
        print(f'DO {delimiter} BEGIN')
        for name in names:
            quoted = name.replace('"', '""')
            literal = name.replace('\\', '\\\\').replace("'", "''")
            print(f"  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = E'{literal}') THEN")
            print(f'    CREATE ROLE "{quoted}" NOLOGIN;')
            print('  END IF;')
        print(f'END {delimiter};')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
