import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import aprovisionar as install
import empaquetar


class InstallationTest(unittest.TestCase):
    def test_sources_are_verified_and_outer_transactions_only_removed(self):
        sources = install.migrations()
        self.assertEqual(len(sources), 3)
        for source in sources:
            self.assertNotIn("\nCOMMIT;", source)
        with patch.object(install, "SOURCES", {"roles_up.sql": "0" * 64}):
            with self.assertRaises(ValueError):
                install.migrations()

    def test_world_readable_or_symlinked_secret_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / "secret"
            path.write_text("synthetic")
            path.chmod(0o644)
            with self.assertRaises(ValueError):
                install.private_file(str(path))
            path.chmod(0o600)
            link = Path(root) / "link"
            link.symlink_to(path)
            with self.assertRaises(OSError):
                install.private_file(str(link))

    def test_database_error_does_not_print_sql_or_password(self):
        cfg = {"service_file": "/private/service", "service": "test", "pass_file": "/private/pass"}
        secret = "never-log-synthetic-secret"
        result = subprocess.CompletedProcess([], 1, "", "ERROR: " + secret)
        with patch.object(install.subprocess, "run", return_value=result) as run:
            with self.assertRaises(ValueError) as error:
                install.execute(cfg, secret, "/usr/bin/psql")
        self.assertNotIn(secret, str(error.exception))
        args, options = run.call_args
        self.assertNotIn(secret, " ".join(args[0]))
        self.assertEqual(options["input"], secret)
        self.assertNotIn("PGPASSWORD", options["env"])


class PackageTest(unittest.TestCase):
    def test_changed_web_or_launcher_is_rejected_before_writing_artifact(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repository = root / "repo"
            repository.mkdir()
            files = {"web/publico.manifest": "static/bolsa/index.html\n",
                     "web/static/bolsa/index.html": "<html>synthetic</html>"}
            for name in ("runtime.py", "aprovisionar.py", "arrancar.sh", "comprobar.sh", "vec-publico.service"):
                files["deploy/publico/" + name] = "synthetic launcher\n"
            for name, content in files.items():
                path = repository / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
            env = {"PATH": os.defpath, "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null"}
            for args in (["init", "-q"], ["add", "web", "deploy"],
                         ["-c", "user.name=aavidad", "-c", "user.email=avidad@dipgra.es", "commit", "-q", "-m", "fixture"]):
                subprocess.run(["git", "-C", str(repository), *args], env=env, check=True)
            binary = root / "binary"
            binary.write_bytes(b"synthetic approved binary")
            with patch.object(empaquetar, "ROOT", repository):
                for name in ("web/static/bolsa/index.html", "deploy/publico/runtime.py"):
                    path = repository / name
                    path.write_text("uncommitted modification")
                    target = root / "artifact"
                    with self.assertRaisesRegex(ValueError, "source_worktree_changed"):
                        empaquetar.package(binary, target)
                    self.assertFalse(target.exists())
                    path.write_text(files[name])
                empaquetar.package(binary, root / "artifact")
                self.assertEqual((root / "artifact/web/static/bolsa/index.html").read_text(), files["web/static/bolsa/index.html"])


@unittest.skipUnless(os.environ.get("VEC_PUBLICO_TEST_CONFIG"), "PostgreSQL dedicado no solicitado")
class DedicatedPostgresTest(unittest.TestCase):
    def setUp(self):
        self.cfg = install.configuration(os.environ["VEC_PUBLICO_TEST_CONFIG"])
        self.psql = os.environ.get("VEC_PUBLICO_TEST_PSQL", "psql")

    def execute(self, sql):
        return install.execute(self.cfg, sql, self.psql)

    def test_01_failed_second_migration_rolls_back_all_roles_and_schemas(self):
        self.execute("CREATE FUNCTION public.preflight_test() RETURNS integer LANGUAGE sql AS 'SELECT 1';")
        try:
            with self.assertRaisesRegex(ValueError, "P0001"):
                self.execute(install.install_sql(self.cfg, True))
        finally:
            self.execute("DROP FUNCTION public.preflight_test();")
        scripts = install.migrations()
        scripts[2] = "SELECT 1/0;\n" + scripts[2]
        baseline = self.execute(install.FINGERPRINT)
        with patch.object(install, "migrations", return_value=scripts):
            with self.assertRaisesRegex(ValueError, "22012"):
                self.execute(install.install_sql(self.cfg, True))
        self.execute("DO $check$ BEGIN IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname LIKE 'vec_%') OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname LIKE 'vec_%') THEN RAISE EXCEPTION 'partial_installation'; END IF; END $check$;")
        self.assertEqual(baseline, self.execute(install.FINGERPRINT))
        self.execute("DO $check$ BEGIN IF (SELECT shobj_description(oid,'pg_database') FROM pg_database WHERE datname=current_database()) IS NOT NULL THEN RAISE EXCEPTION 'comment_not_rolled_back'; END IF; END $check$;")

    def test_02_install_replay_and_incompatible_acl_rejection(self):
        install.provision(self.cfg, True, self.psql)
        install.provision(self.cfg, True, self.psql)
        self.execute("DO $check$ BEGIN IF (SELECT count(*) FROM pg_authid WHERE rolname IN ('vec_publico_login','vec_bolsa_publica_publicador_login') AND rolpassword LIKE 'SCRAM-SHA-256$%')<>2 THEN RAISE EXCEPTION 'password_not_scram'; END IF; END $check$;")
        with tempfile.NamedTemporaryFile(dir=Path(self.cfg["pass_file"]).parent) as changed:
            changed.write(b"synthetic-password-that-must-not-match-123456")
            changed.flush()
            incompatible = dict(self.cfg, reader_password_file=changed.name)
            with self.assertRaises(ValueError):
                install.provision(incompatible, True, self.psql)
        self.execute("ALTER ROLE vec_publico_login NOINHERIT;")
        try:
            with self.assertRaises(ValueError):
                self.execute(install.install_sql(self.cfg, True))
        finally:
            self.execute("ALTER ROLE vec_publico_login INHERIT;")
        self.execute("ALTER ROLE vec_publico_login IN DATABASE vec_bolsa_publica_ensayo SET default_transaction_read_only=off;")
        try:
            with self.assertRaises(ValueError):
                self.execute(install.install_sql(self.cfg, True))
        finally:
            self.execute("ALTER ROLE vec_publico_login IN DATABASE vec_bolsa_publica_ensayo RESET ALL;")
        self.execute(install.install_sql(self.cfg, False))
        self.execute("DO $check$ BEGIN IF (SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_publica_lectura' AND c.relkind='v')<>12 OR EXISTS (SELECT 1 FROM pg_database WHERE datname<>current_database() AND datallowconn AND has_database_privilege('vec_publico_login',oid,'CONNECT')) OR has_database_privilege('vec_publico_login',current_database(),'CREATE,TEMPORARY') OR has_schema_privilege('vec_publico_login','vec_bolsa_publica_datos','USAGE,CREATE') OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' AND has_function_privilege('vec_publico_login',p.oid,'EXECUTE')) THEN RAISE EXCEPTION 'acl'; END IF; END $check$;")


if __name__ == "__main__":
    unittest.main()
