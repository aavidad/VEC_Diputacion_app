#!/usr/bin/env python3
"""Puerta estructural de search_path para funciones SQL cambiadas en una rama.

Sólo lee ficheros SQL y Git. No conecta con PostgreSQL ni ejecuta migraciones.
"""

from __future__ import annotations

import argparse
import hashlib
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Token:
    kind: str
    value: str
    line: int

    @property
    def upper(self) -> str:
        return self.value.upper() if self.kind == "word" else ""


def tokens(sql: str, first_line: int = 1) -> list[Token]:
    """Tokeniza SQL sin confundir comentarios, identificadores o cuerpos con DDL."""
    found: list[Token] = []
    i, line = 0, first_line
    while i < len(sql):
        c = sql[i]
        if c.isspace():
            line += c == "\n"
            i += 1
            continue
        if sql.startswith("--", i):
            end = sql.find("\n", i)
            i = len(sql) if end < 0 else end
            continue
        if sql.startswith("/*", i):
            start, depth = i, 1
            i += 2
            while i < len(sql) and depth:
                if sql.startswith("/*", i):
                    depth += 1
                    i += 2
                elif sql.startswith("*/", i):
                    depth -= 1
                    i += 2
                else:
                    i += 1
            line += sql[start:i].count("\n")
            continue
        if c == '"':
            start = i
            i += 1
            while i < len(sql):
                if sql.startswith('""', i):
                    i += 2
                elif sql[i] == '"':
                    i += 1
                    break
                else:
                    i += 1
            found.append(Token("identifier", sql[start + 1 : i - 1].replace('""', '"'), line))
            line += sql[start:i].count("\n")
            continue
        if c == "'" or (c in "eE" and i + 1 < len(sql) and sql[i + 1] == "'"
                         and (i == 0 or not (sql[i - 1].isalnum() or sql[i - 1] == "_"))):
            start = i
            escape = c in "eE"
            i += 2 if escape else 1
            value = []
            while i < len(sql):
                if escape and sql[i] == "\\" and i + 1 < len(sql):
                    value.append(sql[i : i + 2])
                    i += 2
                elif sql.startswith("''", i):
                    value.append("'")
                    i += 2
                elif sql[i] == "'":
                    i += 1
                    break
                else:
                    value.append(sql[i])
                    i += 1
            found.append(Token("string", "".join(value), line))
            line += sql[start:i].count("\n")
            continue
        if c == "$":
            match = re.match(r"\$[A-Za-z_][A-Za-z_0-9]*\$|\$\$", sql[i:])
            if match:
                tag = match.group()
                end = sql.find(tag, i + len(tag))
                if end >= 0:
                    body = sql[i + len(tag) : end]
                    found.append(Token("body", body, line))
                    segment = sql[i : end + len(tag)]
                    line += segment.count("\n")
                    i = end + len(tag)
                    continue
        if c.isalpha() or c == "_":
            match = re.match(r"[A-Za-z_][A-Za-z_0-9$]*", sql[i:])
            assert match
            found.append(Token("word", match.group(), line))
            i += len(match.group())
            continue
        found.append(Token("symbol", c, line))
        i += 1
    return found


def statements(stream: list[Token]) -> list[list[Token]]:
    result: list[list[Token]] = []
    current: list[Token] = []
    for token in stream:
        if token.value == ";" and token.kind == "symbol":
            if current:
                result.append(current)
            current = []
        else:
            current.append(token)
    if current:
        result.append(current)
    return result


def keyword_pair(stream: list[Token], first: str, second: str) -> bool:
    return any(a.upper == first and b.upper == second for a, b in zip(stream, stream[1:]))


def is_create_routine(stmt: list[Token]) -> bool:
    words = [token.upper for token in stmt[:5] if token.kind == "word"]
    return bool(words and words[0] == "CREATE" and any(
        word in ("FUNCTION", "PROCEDURE", "ROUTINE") for word in words[1:5]))


def is_alter_routine(stmt: list[Token]) -> bool:
    return len(stmt) >= 2 and stmt[0].upper == "ALTER" and stmt[1].upper in ("FUNCTION", "PROCEDURE", "ROUTINE")


def search_path_values(stmt: list[Token]) -> list[list[str]]:
    values: list[list[str]] = []
    following_options = {
        "AS", "LANGUAGE", "SECURITY", "PARALLEL", "IMMUTABLE", "STABLE",
        "VOLATILE", "STRICT", "CALLED", "RETURNS", "LEAKPROOF", "COST",
        "ROWS", "SUPPORT", "WINDOW", "EXTERNAL", "NOT", "TRANSFORM", "SET",
    }
    for i in range(len(stmt) - 2):
        if stmt[i].upper != "SET" or stmt[i + 1].upper != "SEARCH_PATH":
            continue
        j = i + 2
        if stmt[j].value == "=" or stmt[j].upper == "TO":
            j += 1
        else:
            values.append(["opcion_opaca"])
            continue
        names: list[str] = []
        trailing_comma = False
        while j < len(stmt):
            token = stmt[j]
            if token.kind not in ("word", "identifier", "string"):
                break
            names.append(token.value.lower() if token.kind == "word" else token.value)
            j += 1
            if j < len(stmt) and stmt[j].value == ",":
                j += 1
                trailing_comma = True
                continue
            trailing_comma = False
            break
        # Dos literales separados por salto de línea se unen en PostgreSQL.
        # Una vez leídos los dos nombres, sólo puede seguir otra opción de la
        # función o el final de la sentencia; cualquier resto es opaco.
        if trailing_comma or (j < len(stmt) and stmt[j].upper not in following_options):
            names.append("opcion_opaca")
        values.append(names)
    return values


def added_lines(diff: str) -> set[int]:
    result: set[int] = set()
    for start, count in re.findall(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@", diff, re.M):
        # Una supresión pura también modifica la sentencia que queda al lado.
        # Su ancla detecta, por ejemplo, la eliminación de SET search_path.
        length = int(count) if count else 1
        result.update(range(max(1, int(start)), max(1, int(start)) + max(1, length)))
    return result


# Excepción revisada al parche literal de ORQ-1. Cualquier cambio en el DO
# o en sus dos bloques vuelve al rechazo genérico de SQL dinámico.
AD225_PATH = "deploy/postgresql/autorizacion_atestada_v3/migraciones/000225_reanudacion_solicitud_llamamiento.up.sql"
AD225_PREAMBULO = (
    "\\set ON_ERROR_STOP on\n"
    "BEGIN;\n"
    "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;\n"
    "SET LOCAL search_path = pg_catalog;\n"
    "SET LOCAL lock_timeout = '5s';\n"
    "SET LOCAL statement_timeout = '30s';\n"
    "SELECT pg_advisory_xact_lock(hashtextextended('vec:orq1:ad225',0));\n"
    "-- Instalar sólo después de IS17, CA37, AD215, AUT57, AD211, P22, AD175,\n"
    "-- P32, AD180, P34, AD214, AD216 y AD218. Parchea únicamente POST218.\n"
)
AD225_DO_SHA256 = "e1f8592f43312992b7adf7c1b6fe5b5e615ed63fa0b566c7b46b75a70b84e130"
AD225_MARCA_SHA256 = "d511878f1086b13f6e6ece691492a10a1110b9b3b6ef465f8728fd92312da0b7"
AD225_AMPLIACION_SHA256 = "a056298c5d30fbcf2507c67740f378cf348ef6c9d70d8b68b6815b6bdd0afcf4"
AD225_PREIMAGENES = {
    ("c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714",
     "79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5"),
}


def reconstruccion_ad225_revisada(body: str, filename: str | None, sql: str) -> bool:
    """Admite sólo el DO que preserva el núcleo V3 posterior a AD218 byte a byte."""
    if filename != AD225_PATH or not sql.startswith(AD225_PREAMBULO + "DO $parche$") \
            or sql.count("DO $parche$") != 1 \
            or hashlib.sha256(body.encode()).hexdigest() != AD225_DO_SHA256:
        return False
    marca = re.search(r"marca text := \$marca225\$(.*?)\$marca225\$;", body, re.S)
    ampliacion = re.search(r"ampliacion text := \$ampliacion225\$(.*?)\$ampliacion225\$;", body, re.S)
    if not marca or not ampliacion or body.count("$marca225$") != 2 \
            or body.count("$ampliacion225$") != 2:
        return False
    anterior, posterior = marca.group(1), ampliacion.group(1)
    if (hashlib.sha256(anterior.encode()).hexdigest() != AD225_MARCA_SHA256
            or hashlib.sha256(posterior.encode()).hexdigest() != AD225_AMPLIACION_SHA256
            or not posterior.startswith(anterior)
            or re.search(r"\b(?:CREATE|ALTER|SET|RESET|SECURITY)\b", posterior, re.I)):
        return False
    pares = re.findall(r"IF def_sha IS DISTINCT FROM '([0-9a-f]{64})'\s+OR src_sha IS DISTINCT FROM '([0-9a-f]{64})' THEN", body)
    if len(pares) != 1 or set(pares) != AD225_PREIMAGENES:
        return False
    requeridos = (
        "f oid := to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')",
        "p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']",
        "p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole",
        "p.prosecdef AND p.provolatile='v' AND p.proparallel='u'",
        "aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))",
        "a.grantee=p.proowner AND a.grantor=p.proowner",
        "a.privilege_type='EXECUTE' AND NOT a.is_grantable",
        "WHERE p.oid=f)<>1",
        "strpos(original,marca)=0",
        "strpos(substr(original,strpos(original,marca)+length(marca)),marca)<>0",
        "strpos(original,'reanudacion_solicitud_llamamiento')<>0",
        "strpos(original,'contratacion_temporal.llamamiento.reanudar_solicitud')<>0",
        "nuevo:=replace(original,marca,ampliacion);",
        "EXECUTE nuevo;",
        "actual IS DISTINCT FROM nuevo",
        "replace(actual,ampliacion,marca) IS DISTINCT FROM original",
        "(SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta",
        "FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f",
        "FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())",
    )
    return all(piece in body for piece in requeridos) and body.count("pg_get_functiondef(f)") == 2 \
        and body.count("EXECUTE nuevo;") == 1


def inspect_sql(sql: str, changed: set[int], filename: str | None = None) -> list[tuple[int, str]]:
    failures: list[tuple[int, str]] = []
    # El preámbulo se comprueba aunque el diff sólo cambie una línea anterior
    # al DO: de lo contrario el recorrido por sentencias no visitaría su cuerpo.
    if filename == AD225_PATH and changed:
        cuerpos = [t for t in tokens(sql) if t.kind == "body" and
                   "pg_get_functiondef" in t.value and "EXECUTE" in t.value]
        if len(cuerpos) != 1 or not reconstruccion_ad225_revisada(cuerpos[0].value, filename, sql):
            failures.append((cuerpos[0].line if cuerpos else min(changed),
                             "reconstrucción dinámica de función: preámbulo o parche AD225 divergente"))

    def inspect(segment: str, first_line: int, depth: int = 0) -> None:
        if depth > 5:
            failures.append((first_line, "SQL anidado demasiado profundo"))
            return
        for stmt in statements(tokens(segment, first_line)):
            lines = {token.line for token in stmt}
            if not lines.intersection(changed):
                # Un cambio en el interior de un cuerpo también reconstruye su función.
                bodies = [t for t in stmt if t.kind == "body"]
                if not any(any(t.line <= n <= t.line + t.value.count("\n") for n in changed)
                           for t in bodies):
                    continue
            if is_create_routine(stmt):
                # PostgreSQL admite opciones de función después de AS $tag$...$tag$.
                header = [t for t in stmt if t.kind != "body"]
                if keyword_pair(header, "SECURITY", "DEFINER"):
                    paths = search_path_values(header)
                    if paths != [["pg_catalog", "pg_temp"]]:
                        failures.append((stmt[0].line, "SECURITY DEFINER requiere SET search_path = pg_catalog, pg_temp"))
            elif is_alter_routine(stmt):
                if keyword_pair(stmt, "RESET", "SEARCH_PATH") or keyword_pair(stmt, "RESET", "ALL"):
                    failures.append((stmt[0].line, "ALTER FUNCTION/PROCEDURE/ROUTINE no puede retirar search_path con RESET"))
                if keyword_pair(stmt, "SET", "SEARCH_PATH") and search_path_values(stmt) != [["pg_catalog", "pg_temp"]]:
                    failures.append((stmt[0].line, "ALTER FUNCTION/PROCEDURE/ROUTINE requiere search_path = pg_catalog, pg_temp"))
                if keyword_pair(stmt, "SECURITY", "DEFINER"):
                    failures.append((stmt[0].line, "ALTER SECURITY DEFINER requiere revisión de su definición y search_path"))
            # DO y cadenas ejecutadas contienen SQL; los cuerpos de funciones
            # ordinarias son opacos para evitar confundir sus comentarios con DDL.
            for body in (t for t in stmt if t.kind == "body"):
                body_tokens = tokens(body.value, body.line)
                if any(t.upper == "PG_GET_FUNCTIONDEF" for t in body_tokens) and any(
                    t.upper == "EXECUTE" for t in body_tokens
                ) and not reconstruccion_ad225_revisada(body.value, filename, sql):
                    failures.append((body.line, "reconstrucción dinámica de función: exigir definición final explícita y revisión SQL"))
                if any(t.upper == "EXECUTE" for t in body_tokens):
                    literals = [t for t in body_tokens if t.kind in ("string", "body")]
                    routine_ddl = (r"\b(?:CREATE\s+(?:OR\s+REPLACE\s+)?(?:FUNCTION|PROCEDURE|ROUTINE)"
                                   r"|ALTER\s+(?:FUNCTION|PROCEDURE|ROUTINE))\b")
                    joined = "".join(t.value for t in literals)
                    has_ddl_verb = re.search(r"\b(?:CREATE|ALTER)\b", joined, re.I) or any(
                        re.search(r"\b(?:CREATE|ALTER)\b", t.value, re.I) for t in literals)
                    has_routine = re.search(r"\b(?:FUNCTION|PROCEDURE|ROUTINE)\b", joined, re.I) or any(
                        re.search(r"\b(?:FUNCTION|PROCEDURE|ROUTINE)\b", t.value, re.I) for t in literals)
                    if re.search(routine_ddl, joined, re.I) or (has_ddl_verb and has_routine):
                        for inner in statements(body_tokens):
                            for i, token in enumerate(inner):
                                if token.upper != "EXECUTE":
                                    continue
                                expression = inner[i + 1:]
                                direct = bool(expression and (expression[0].kind in ("string", "body") or
                                    (expression[0].upper == "FORMAT" and any(t.kind in ("string", "body") for t in expression[1:]))))
                                concatenated = any(a.value == "|" and b.value == "|"
                                                   for a, b in zip(expression, expression[1:]))
                                format_literal = next((t.value for t in expression if t.kind in ("string", "body")), "")
                                variable_format = expression and expression[0].upper == "FORMAT" and re.search(
                                    r"%(?:\d+\$)?[sL]", format_literal)
                                config_tail = re.search(r"\b(?:SET|RESET)\b(.*)", format_literal, re.I | re.S)
                                variable_config = bool(config_tail and re.search(
                                    r"%(?:\d+\$)?I", config_tail.group(1)))
                                if not direct or concatenated or variable_format or variable_config:
                                    failures.append((token.line, "DDL dinámico de rutina concatenado u opaco: definición final no verificable"))
                        # Si el verbo y el tipo están repartidos entre literales,
                        # ninguna cadena aislada demuestra las opciones finales.
                        if not any(re.search(routine_ddl, t.value, re.I) for t in literals):
                            failures.append((body.line, "DDL dinámico de rutina fragmentado: definición final no verificable"))
                    for literal in literals:
                        if re.search(routine_ddl, literal.value, re.I):
                            inspect(literal.value, literal.line, depth + 1)
            if stmt[0].upper == "DO":
                for body in (t for t in stmt if t.kind == "body"):
                    inspect(body.value, body.line, depth + 1)
            if any(t.upper == "PG_GET_FUNCTIONDEF" for t in stmt) and any(t.upper == "EXECUTE" for t in stmt):
                failures.append((stmt[0].line, "reconstrucción dinámica de función: exigir definición final explícita y revisión SQL"))
            for i, token in enumerate(stmt):
                if token.upper == "EXECUTE" and i + 1 < len(stmt) and stmt[i + 1].kind in ("string", "body"):
                    inspect(stmt[i + 1].value, stmt[i + 1].line, depth + 1)

    inspect(sql, 1)
    return sorted(set(failures))


def git(*args: str) -> str:
    result = subprocess.run(["git", *args], check=True, capture_output=True, text=True)
    return result.stdout.strip()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default=os.environ.get("VEC_SEARCH_PATH_BASE") or
                        f"origin/{os.environ.get('GITHUB_BASE_REF') or 'main'}")
    args = parser.parse_args()
    try:
        ref = git("rev-parse", "--verify", f"{args.base}^{{commit}}")
        base = git("merge-base", ref, "HEAD")
        if base == git("rev-parse", "HEAD") and os.environ.get("CI"):
            base = git("rev-parse", "HEAD^1")
        names = git("diff", "--name-only", "--diff-filter=ACMR", base, "--", "*.sql").splitlines()
    except subprocess.CalledProcessError:
        print(f"PARO clave=base_search_path esperado=referencia_git_valida actual={args.base}", file=sys.stderr)
        return 2
    failures = []
    for name in names:
        path = Path(name)
        if path.suffix != ".sql" or not path.is_file():
            continue
        diff = git("diff", "--unified=0", base, "--", name)
        for line, reason in inspect_sql(path.read_text(encoding="utf-8"), added_lines(diff), filename=name):
            failures.append((name, line, reason))
    for name, line, reason in failures:
        print(f"PARO clave=search_path_definer archivo={name}:{line} esperado=pg_catalog,pg_temp actual={reason}", file=sys.stderr)
    if failures:
        return 1
    print(f"search_path_definer: conforme para {len(names)} SQL modificados desde {base[:12]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
