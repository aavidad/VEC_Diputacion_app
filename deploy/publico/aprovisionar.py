#!/usr/bin/env python3
"""Instalación transaccional en una instancia PostgreSQL pública dedicada."""
import argparse
import configparser
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[2]
LOGIN = "vec_publico_login"
SOURCES = {
    "roles_up.sql": "e3ebb322125e3457928f87ca07f158c5a91afd3d13b401fe6a6fc3db6e54da11",
    "migraciones/000001_proyeccion_publica.up.sql": "1613e3c45cd0e169124dfb2c7b7834e0bf0ff6349350f77e96c909f2f30a7a01",
    "migraciones/000002_proyeccion_bolsas_v1.up.sql": "3260913cd4410da7590dcdc86f8bdf874d490d8be93f64d03d52827b5a62728f",
}
VERSION = "vec-publico-instalacion-v1:" + hashlib.sha256(
    json.dumps(SOURCES, sort_keys=True).encode()).hexdigest()


def private_file(value, limit=65536):
    path = Path(value)
    if not path.is_absolute():
        raise ValueError("private path")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        info = os.fstat(fd)
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
                or info.st_mode & 0o077 or info.st_nlink != 1 or info.st_size > limit):
            raise ValueError("private file")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            return stream.read(limit + 1).decode("utf-8")
    finally:
        os.close(fd)


def literal(value):
    if "\x00" in value:
        raise ValueError("NUL")
    return "'" + value.replace("'", "''") + "'"


def configuration(path):
    cfg = json.loads(private_file(path))
    if set(cfg) != {"service_file", "pass_file", "service", "database",
                    "system_identifier", "reader_password_file", "publisher_password_file"}:
        raise ValueError("configuration")
    if (not re.fullmatch(r"vec_bolsa_publica[a-z0-9_]*", cfg["database"])
            or not re.fullmatch(r"[0-9]{15,20}", cfg["system_identifier"])
            or not re.fullmatch(r"[a-zA-Z0-9_-]+", cfg["service"])):
        raise ValueError("identity")
    service = configparser.ConfigParser(interpolation=None)
    service.read_string(private_file(cfg["service_file"]))
    options = dict(service[cfg["service"]])
    # Sin inclusión de servicios, opciones de sesión ni conexiones alternativas.
    if (set(options) != {"host", "port", "dbname", "user", "sslmode", "sslrootcert", "require_auth"}
            or options["dbname"] != cfg["database"]
            or options["sslmode"] != "verify-full"
            or options["require_auth"] != "scram-sha-256"
            or not options["host"] or "," in options["host"]
            or not options["port"].isdigit()):
        raise ValueError("TLS service")
    if not Path(options["sslrootcert"]).is_absolute():
        raise ValueError("CA path")
    private_file(cfg["pass_file"])
    return cfg


def migrations():
    result = []
    for name, expected in SOURCES.items():
        contents = (ROOT / "deploy/postgresql/bolsa_publica" / name).read_bytes()
        if hashlib.sha256(contents).hexdigest() != expected:
            raise ValueError("source checksum")
        text = contents.decode()
        if len(re.findall(r"^BEGIN;$", text, re.M)) != 1 or len(re.findall(r"^COMMIT;$", text, re.M)) != 1:
            raise ValueError("transaction")
        top = re.sub(r"\$([A-Za-z_][A-Za-z_0-9]*|)\$.*?\$\1\$", "''", text, flags=re.S)
        top = re.sub(r"'(?:[^']|'')*'", "''", top)
        top = re.sub(r"--[^\n]*|/\*.*?\*/|^\\[^\n]*", "", top, flags=re.S | re.M)
        statements = [s.strip().upper() for s in top.split(";") if s.strip()]
        if statements[0] != "BEGIN" or statements[-1] != "COMMIT" or any(
                re.match(r"^(BEGIN|COMMIT|ROLLBACK|START\s+TRANSACTION|SAVEPOINT|RELEASE|END)\b", s)
                for s in statements[1:-1]):
            raise ValueError("nested transaction")
        result.append(re.sub(r"^(?:BEGIN|COMMIT);$", "", text, flags=re.M))
    return result


# No incluye datos ni estadísticas; sí definiciones, propietarios, ACL,
# membresías y ajustes. La marca queda en el comentario de la base dedicada.
FINGERPRINT = """
SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.jsonb_build_object(
    'schemas', (SELECT jsonb_agg(to_jsonb(n) ORDER BY n.oid) FROM pg_namespace n
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'relations', (SELECT jsonb_agg(jsonb_build_array(c.oid,c.relname,c.relnamespace,
      c.relkind,c.relowner,c.relacl,c.reloptions,c.relrowsecurity,c.relforcerowsecurity)
      ORDER BY c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'columns', (SELECT jsonb_agg(to_jsonb(a) ORDER BY a.attrelid,a.attnum)
      FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
      JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname !~ '^pg_'
      AND n.nspname <> 'information_schema' AND a.attnum>0),
    'constraints', (SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_constraint c
      JOIN pg_namespace n ON n.oid=c.connamespace WHERE n.nspname !~ '^pg_'
      AND n.nspname <> 'information_schema'),
    'indexes', (SELECT jsonb_agg(to_jsonb(i) ORDER BY i.indexrelid) FROM pg_index i
      JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'types', (SELECT jsonb_agg(to_jsonb(t) ORDER BY t.oid) FROM pg_type t
      JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname !~ '^pg_'
      AND n.nspname <> 'information_schema'),
    'column_defaults', (SELECT jsonb_agg(to_jsonb(a) ORDER BY a.oid) FROM pg_attrdef a
      JOIN pg_class c ON c.oid=a.adrelid JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'policies', (SELECT jsonb_agg(to_jsonb(p) ORDER BY p.oid) FROM pg_policy p),
    'functions', (SELECT jsonb_agg(to_jsonb(p) ORDER BY p.oid) FROM pg_proc p
      JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname !~ '^pg_'
      AND n.nspname <> 'information_schema'),
    'triggers', (SELECT jsonb_agg(to_jsonb(t) ORDER BY t.oid) FROM pg_trigger t
      JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'rules', (SELECT jsonb_agg(to_jsonb(r) ORDER BY r.oid) FROM pg_rewrite r
      JOIN pg_class c ON c.oid=r.ev_class JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'),
    'defaults', (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.oid) FROM pg_default_acl d),
    'roles', (SELECT jsonb_agg(to_jsonb(r) ORDER BY r.oid) FROM pg_roles r
      WHERE r.rolname LIKE 'vec_%'),
    'database_role_settings', (SELECT jsonb_agg(to_jsonb(s) ORDER BY s.setdatabase,s.setrole)
      FROM pg_db_role_setting s),
    'members', (SELECT jsonb_agg(to_jsonb(m) ORDER BY m.oid) FROM pg_auth_members m
      JOIN pg_roles r ON r.oid=m.member WHERE r.rolname LIKE 'vec_%'),
    'databases', (SELECT jsonb_agg(jsonb_build_array(d.oid,d.datname,d.datdba,
      d.datacl,d.datallowconn) ORDER BY d.oid) FROM pg_database d)
  )::text, 'UTF8')), 'hex') AS fingerprint
"""


def preflight(cfg):
    db = literal(cfg["database"])
    system = literal(cfg["system_identifier"])
    return f"""
DO $preflight$
BEGIN
 IF current_database() <> {db} OR
    (SELECT system_identifier::text FROM pg_control_system()) <> {system} OR
    current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
    NOT (SELECT rolsuper FROM pg_roles WHERE rolname=current_user) OR
    NOT (SELECT ssl AND version IN ('TLSv1.2','TLSv1.3')
           FROM pg_stat_ssl WHERE pid=pg_backend_pid()) THEN
   RAISE EXCEPTION 'preflight_identity_tls';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_database WHERE datname NOT IN
      ({db},'postgres','template0','template1')) OR
    EXISTS (SELECT 1 FROM pg_namespace WHERE nspname !~ '^pg_' AND nspname NOT IN
      ('information_schema','public','vec_bolsa_publica_datos',
       'vec_bolsa_publica_lectura','vec_bolsa_publica_publicacion')) OR
    EXISTS (SELECT 1 FROM pg_roles WHERE rolname !~ '^pg_' AND rolname <> current_user
      AND rolname NOT IN ('vec_bolsa_publica_propietario',
      'vec_bolsa_publica_publicacion_propietario','vec_bolsa_publica_migrador',
      'vec_bolsa_publica_consulta','vec_bolsa_publica_publicador',
      'vec_bolsa_publica_publicador_login','{LOGIN}')) OR
    EXISTS (SELECT 1 FROM pg_depend d JOIN pg_namespace n ON n.oid=d.refobjid
      WHERE d.refclassid='pg_namespace'::regclass AND n.nspname='public') OR
    EXISTS (SELECT 1 FROM pg_extension WHERE extname<>'plpgsql') OR
    EXISTS (SELECT 1 FROM pg_event_trigger) OR
    EXISTS (SELECT 1 FROM pg_foreign_data_wrapper) OR
    EXISTS (SELECT 1 FROM pg_foreign_server) OR
    current_setting('shared_preload_libraries')<>'' OR
    current_setting('session_preload_libraries')<>'' OR
    current_setting('local_preload_libraries')<>'' THEN
   RAISE EXCEPTION 'preflight_instance_not_dedicated';
 END IF;
END $preflight$;
"""


def install_sql(cfg, apply):
    scripts = migrations()
    db = cfg["database"]  # Identificador validado; sin entrada libre.
    sql = "\\set ON_ERROR_STOP on\n"
    # Conexión de un solo uso: desactivar muestreo antes de BEGIN evita que
    # PostgreSQL marque la transacción completa para registrar sus sentencias.
    sql += "SET debug_print_parse=off; SET debug_print_rewritten=off; SET debug_print_plan=off;\n"
    sql += "SET log_statement='none'; SET log_min_error_statement='panic'; SET log_parameter_max_length_on_error=0; SET log_min_duration_statement=-1; SET log_min_duration_sample=-1; SET log_statement_sample_rate=0; SET log_transaction_sample_rate=0;\n"
    sql += "BEGIN;\nSET LOCAL search_path=pg_catalog;\n"
    sql += "SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='120s';\n"
    sql += "SELECT pg_advisory_xact_lock(hashtextextended('vec-publico-instalar:v1',0));\n"
    sql += preflight(cfg)
    sql += "SELECT shobj_description(oid,'pg_database') IS NOT NULL AS installed FROM pg_database WHERE datname=current_database() \\gset\n"
    sql += "\\if :installed\n" + FINGERPRINT + " \\gset\n"
    sql += "SELECT (SELECT shobj_description(oid,'pg_database') FROM pg_database WHERE datname=current_database()) = " + literal(VERSION + ":") + " || :'fingerprint' AS compatible \\gset\n\\if :compatible\n\\else\nDO $reject$ BEGIN RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='incompatible_installation'; END $reject$;\n\\endif\n"
    sql += "\\else\n"
    sql += "DO $fresh$ BEGIN IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname LIKE 'vec_%') OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname LIKE 'vec_%') THEN RAISE EXCEPTION 'unmarked_installation'; END IF; END $fresh$;\n"
    if apply:
        reader = private_file(cfg["reader_password_file"]).strip()
        publisher = private_file(cfg["publisher_password_file"]).strip()
        if any(not re.fullmatch(r"[A-Za-z0-9_-]{32,128}", p) for p in (reader, publisher)) or reader == publisher:
            raise ValueError("secret material")
        # Sólo esta instancia aprobada y vacía: jamás ACL del clúster interno.
        sql += "SET LOCAL password_encryption='scram-sha-256';\n"
        sql += f"REVOKE ALL ON DATABASE {db},postgres,template0,template1 FROM PUBLIC;\nREVOKE CREATE ON SCHEMA public FROM PUBLIC;\n"
        sql += scripts[0] + "\nSET ROLE vec_bolsa_publica_migrador;\n" + scripts[1]
        sql += "\nSET ROLE vec_bolsa_publica_migrador;\n" + scripts[2] + "\nRESET ROLE;\n"
        sql += f"CREATE ROLE {LOGIN} LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD {literal(reader)};\n"
        sql += f"GRANT vec_bolsa_publica_consulta TO {LOGIN} WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;\n"
        sql += f"ALTER ROLE {LOGIN} SET default_transaction_read_only=on;\nALTER ROLE {LOGIN} SET search_path='pg_catalog,pg_temp';\nALTER ROLE {LOGIN} SET statement_timeout='10s';\nALTER ROLE {LOGIN} SET lock_timeout='2s';\nALTER ROLE {LOGIN} SET idle_in_transaction_session_timeout='10s';\n"
        sql += f"ALTER ROLE vec_bolsa_publica_publicador_login PASSWORD {literal(publisher)};\n"
        sql += FINGERPRINT + " \\gset\n"
        sql += f"SELECT format('COMMENT ON DATABASE %I IS %L',current_database(),{literal(VERSION + ':')} || :'fingerprint') \\gexec\n"
    sql += "\\endif\nCOMMIT;\n"
    return sql


def execute(cfg, sql, psql, user=None, pass_file=None):
    env = {"PATH": os.defpath, "LANG": "C.UTF-8", "PGSERVICEFILE": cfg["service_file"],
           "PGSERVICE": cfg["service"], "PGPASSFILE": pass_file or cfg["pass_file"],
           "PGCONNECT_TIMEOUT": "5", "PGAPPNAME": "vec-publico-instalador"}
    command = [psql, "-XqAt", "--set=VERBOSITY=sqlstate", "--no-password", "--file=-"]
    if user:
        command += ["--username", user]
    result = subprocess.run(command,
                            input=sql, text=True, capture_output=True, env=env, timeout=180)
    if result.returncode:
        # psql puede incluir SQL y secretos en los errores. Nunca se imprimen.
        code = re.search(r"ERROR:\s+([A-Z0-9]{5})\s*$", result.stderr, re.M)
        raise ValueError("database rejected: " + (code[1] if code else "closed"))
    return result.stdout


def provision(cfg, apply, psql):
    execute(cfg, install_sql(cfg, apply), psql)
    if not apply:
        return
    service = configparser.ConfigParser(interpolation=None)
    service.read_string(private_file(cfg["service_file"]))
    options = service[cfg["service"]]
    # El replay no rota secretos. Acreditar las dos conexiones impide informar
    # éxito si se entrega una contraseña distinta de la que conserva la base.
    with tempfile.TemporaryDirectory(prefix=".publico-cred-", dir=Path(cfg["pass_file"]).parent) as directory:
        path = Path(directory) / "pgpass"
        for user, key in ((LOGIN, "reader_password_file"),
                          ("vec_bolsa_publica_publicador_login", "publisher_password_file")):
            password = private_file(cfg[key]).strip()
            values = [options["host"], options["port"], cfg["database"], user, password]
            line = ":".join(v.replace("\\", "\\\\").replace(":", "\\:") for v in values)
            path.write_text(line + "\n")
            path.chmod(0o600)
            execute(cfg, "SELECT 1;", psql, user=user, pass_file=str(path))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--psql", default="psql")
    parser.add_argument("--aplicar-instancia-dedicada", action="store_true")
    args = parser.parse_args()
    try:
        cfg = configuration(args.config)
        provision(cfg, args.aplicar_instancia_dedicada, args.psql)
    except (OSError, ValueError, KeyError, subprocess.SubprocessError, configparser.Error):
        print("aprovisionamiento_publico_rechazado", file=sys.stderr)
        return 1
    print("aprovisionamiento_publico_ok" if args.aplicar_instancia_dedicada else "preflight_publico_ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
