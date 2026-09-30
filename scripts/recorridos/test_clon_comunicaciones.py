"""Focused checks for clone ownership, isolation and TLS configuration."""
import copy
import importlib.util
from pathlib import Path
import smtplib
import ssl
import unittest
from unittest.mock import MagicMock, patch

spec = importlib.util.spec_from_file_location("clon_comunicaciones", Path(__file__).with_name("clon_comunicaciones.py"))
c = importlib.util.module_from_spec(spec)
spec.loader.exec_module(c)


def resource():
    return {"Image": "image:local", "Config": {"Labels": c._labels(c.STATE), "Cmd": list(c.SMTP_FLAGS)},
            "HostConfig": {"ReadonlyRootfs": True, "AutoRemove": True, "NetworkMode": c.NETWORK,
                           "PortBindings": {"1025/tcp": [{"HostIp": "127.0.0.1", "HostPort": "11025"}],
                                            "8025/tcp": [{"HostIp": "127.0.0.1", "HostPort": "18532"}]}},
            "Mounts": [{"Destination": "/tls", "Source": str(c.STATE / "material/comunicaciones"), "RW": False},
                       {"Destination": "/data", "Source": str(c.STATE / "comunicaciones/buzon"), "RW": True}],
            "State": {"Running": True}}


class CloneCommunicationTests(unittest.TestCase):
    def test_rejects_foreign_sink_global_binds_and_relay_flags(self):
        c._validate_sink(resource(), c.STATE, "image:local")
        changes = [lambda r: r["Config"]["Labels"].update({c.LABEL_OWNER: "other"}),
                   lambda r: r["HostConfig"]["PortBindings"]["1025/tcp"][0].update(HostIp="0.0.0.0"),
                   lambda r: r["Config"]["Cmd"].append("--smtp-relay-all"),
                   lambda r: r["Mounts"][0].update(RW=True),
                   lambda r: r["Mounts"][1].update(Source="/someone-else"),
                   lambda r: r["State"].update(Running=False),
                   lambda r: r["HostConfig"].update(NetworkMode="bridge"),
                   lambda r: r["HostConfig"].update(ReadonlyRootfs=False)]
        for mutate in changes:
            with self.subTest(mutate=mutate):
                r = copy.deepcopy(resource())
                mutate(r)
                with self.assertRaises(c.PreparationError):
                    c._validate_sink(r, c.STATE, "image:local")

    def test_image_preparation_survives_smtp_blocker(self):
        result = c._result(source="a" * 40, image=True, error="mailpit_loopback_port_busy")
        self.assertEqual(result["env"]["VEC_USUARIOS_IMAGEN_ENABLED"], "true")
        self.assertNotIn("VEC_USUARIOS_CORREOS_ENABLED", result["env"])
        self.assertNotIn("VEC_SMTP_HOST", result["env"])
        self.assertEqual(result["blockers"][0]["profile"], "usuarios_correos")
        self.assertFalse(result["profiles"]["usuarios_comunicaciones"]["corporate_delivery"])

    def test_env_follows_existing_tls_contract(self):
        env = c._result(image=True, smtp=True)["env"]
        self.assertEqual(env["VEC_SMTP_HOST"], "127.0.0.1")
        self.assertEqual(env["VEC_SMTP_MODO_TLS"], "starttls")
        self.assertEqual(env["VEC_SMTP_FROM"], "rrhh@example.test")
        self.assertEqual(env["VEC_SMTP_CA_FILE"], str(c.STATE / "material/ca/ca.crt"))
        self.assertFalse(any("INSECURE" in key for key in env))

    def test_scope_rejects_remote_or_other_clone_before_commands(self):
        args = (Path("/source"), "other-container", c.STATE, c.STATE / "material", 55531, "docker")
        with patch.object(c, "_canonical", side_effect=lambda p: Path(p)), patch.object(c, "_run") as run:
            with self.assertRaises(c.PreparationError):
                c._scope(*args)
            run.assert_not_called()

    def test_proxy_has_only_owned_fixed_destination_and_loopback_listener(self):
        self.assertEqual(c._proxy_command("192.168.224.2", 11025, 1025)[-1],
                         "TCP4:192.168.224.2:1025,connect-timeout=5")
        self.assertIn("bind=127.0.0.1", c._proxy_command("192.168.224.2", 18532, 8025)[-2])
        for address in ("example.com", "8.8.8.8", "127.0.0.1", "0.0.0.0", "224.0.0.1"):
            with self.subTest(address=address), self.assertRaises((ValueError, c.PreparationError)):
                c._proxy_command(address, 11025, 1025)
        with self.assertRaises(c.PreparationError):
            c._proxy_command("192.168.224.2", 25, 25)

    def test_source_reads_pinned_commit_instead_of_worktree(self):
        calls = []
        required = {
            "config/config.go": "VEC_SMTP_HOST VEC_SMTP_PORT VEC_SMTP_FROM VEC_SMTP_CA_FILE VEC_SMTP_MODO_TLS",
            "internal/app/bootstrap/usuarios_correos_material.go": "VEC_USUARIOS_CORREOS_ENABLED",
            "internal/app/bootstrap/usuarios_imagen_material.go": "VEC_USUARIOS_IMAGEN_ENABLED",
            "internal/app/bootstrap/correo_llamamiento_smtp_desarrollo.go": "smtp.STARTTLSObligatorio cfg.SMTPCAFile",
        }
        def run(argv):
            calls.append(argv)
            if "rev-parse" in argv:
                return b"0123456789012345678901234567890123456789\n"
            return required[argv[-1].split(":", 1)[1]].encode()
        with patch.object(c, "_run", side_effect=run):
            c._source_contract(Path("/source"))
        self.assertEqual(calls[0][-1], c.SOURCE_REF + "^{commit}")
        self.assertTrue(all("show" in argv for argv in calls[1:]))

    def test_preflight_checks_ports_without_writing_or_running_mailpit(self):
        parameters = dict(repo=Path("/source"), container=c.PG_CONTAINER, state=c.STATE,
                          material=c.STATE / "material", pg_port=55531, engine="docker")
        proof = b'{"roles_version":2,"assignments_version":2,"surfaces":2,"grants_per_role":10}'
        def inspect(engine, kind, name):
            return {"Id": "image:local"} if kind == "image" else None
        with patch.object(c, "_scope", return_value=(Path("/source"), c.STATE, c.STATE / "material")), \
                patch.object(c, "_source_contract", return_value="a" * 40), \
                patch.object(c, "_private", return_value=proof), \
                patch.object(c, "_inspect", side_effect=inspect), \
                patch.object(c, "_free_ports", side_effect=c.PreparationError("mailpit_loopback_port_busy")), \
                patch.object(c, "_certificate") as certificate:
            result = c.provision(**parameters)
        certificate.assert_not_called()
        self.assertEqual(result["blockers"][0]["detail"], "mailpit_loopback_port_busy")
        self.assertEqual(result["env"]["VEC_USUARIOS_IMAGEN_ENABLED"], "true")

    def test_noninternal_network_blocks_before_effects(self):
        proof = b'{"roles_version":2,"assignments_version":2,"surfaces":2,"grants_per_role":10}'
        def inspect(engine, kind, name):
            if kind == "image": return {"Id": "image:local"}
            if kind == "network": return {"Internal": False, "Labels": c._labels(c.STATE)}
            return None
        with patch.object(c, "_scope", return_value=(Path("/source"), c.STATE, c.STATE / "material")), \
                patch.object(c, "_source_contract", return_value="a" * 40), \
                patch.object(c, "_private", return_value=proof), patch.object(c, "_inspect", side_effect=inspect):
            result = c.preflight(None, None, None, None, None, None)
        self.assertEqual(result["blockers"][0]["detail"], "mailpit_network_not_isolated")

    def test_probe_verifies_tls_and_rejects_external_rcpt_without_data(self):
        connection = MagicMock()
        connection.has_extn.return_value = True
        connection.rcpt.return_value = (550, b"synthetic recipients only")
        context = ssl.create_default_context()
        with patch.object(c.ssl, "create_default_context", return_value=context), \
                patch.object(c.smtplib, "SMTP") as smtp:
            smtp.return_value.__enter__.return_value = connection
            result = c._smtp_probe(Path("/ca"))
        self.assertTrue(context.check_hostname)
        self.assertEqual(context.verify_mode, ssl.CERT_REQUIRED)
        connection.starttls.assert_called_once_with(context=context)
        connection.sendmail.assert_not_called()
        self.assertTrue(result["external_recipient_rejected"])
        connection.rcpt.return_value = (250, b"unexpected")
        with patch.object(c.ssl, "create_default_context", return_value=context), \
                patch.object(c.smtplib, "SMTP") as smtp:
            smtp.return_value.__enter__.return_value = connection
            with self.assertRaises(c.PreparationError): c._smtp_probe(Path("/ca"))


if __name__ == "__main__":
    unittest.main()
