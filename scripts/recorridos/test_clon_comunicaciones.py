"""Focused checks for clone ownership, isolation and TLS configuration."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import smtplib
import ssl
import tempfile
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


def record(state, port=11025, pid=700123):
    return {"owner": c.OWNER, "state": str(state), "pid": pid,
            "command": c._proxy_command("192.168.224.2", port, 1025 if port == 11025 else 8025),
            "start_ticks": "12345"}


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

    def test_guard_confirms_missing_pid_without_signalling(self):
        data = record(c.STATE)
        with patch.object(c.os, "pidfd_open", side_effect=ProcessLookupError), \
                patch.object(c.signal, "pidfd_send_signal") as send:
            self.assertEqual(c._proxy_guard(data, data["command"], c.STATE), ("dead", None))
        send.assert_not_called()

    def test_empty_or_foreign_record_is_never_treated_as_missing(self):
        for data in ({}, None, {**record(c.STATE), "owner": "other"}, {**record(c.STATE), "pid": True}):
            with self.subTest(data=data), patch.object(c.os, "pidfd_open") as opened:
                with self.assertRaises(c.PreparationError): c._proxy_guard(data, record(c.STATE)["command"], c.STATE)
                opened.assert_not_called()

    def test_guard_rejects_reused_pid_and_never_signals(self):
        data = record(c.STATE)
        for snapshot in ({"uid": os.getuid(), "start_ticks": "99999", "command": [x.encode() for x in data["command"]]},
                         {"uid": os.getuid(), "start_ticks": "12345", "command": [b"other-service"]},
                         {"uid": os.getuid() + 1, "start_ticks": "12345", "command": [x.encode() for x in data["command"]]}):
            with self.subTest(snapshot=snapshot), \
                    patch.object(c.os, "pidfd_open", return_value=99), patch.object(c.os, "close") as close, \
                    patch.object(c, "_pidfd_dead", return_value=False), \
                    patch.object(c, "_process_snapshot", return_value=snapshot), \
                    patch.object(c.signal, "pidfd_send_signal") as send:
                with self.assertRaises(c.PreparationError): c._proxy_guard(data, data["command"], c.STATE)
                close.assert_called_once_with(99)
                send.assert_not_called()

    def test_guard_handles_exit_while_reading_proc_only_when_pidfd_confirms(self):
        data = record(c.STATE)
        with patch.object(c.os, "pidfd_open", return_value=99), patch.object(c.os, "close"), \
                patch.object(c, "_pidfd_dead", side_effect=[False, True]), \
                patch.object(c, "_process_snapshot", side_effect=FileNotFoundError):
            self.assertEqual(c._proxy_guard(data, data["command"], c.STATE), ("dead", None))
        with patch.object(c.os, "pidfd_open", return_value=99), patch.object(c.os, "close"), \
                patch.object(c, "_pidfd_dead", return_value=False), \
                patch.object(c, "_process_snapshot", side_effect=FileNotFoundError):
            with self.assertRaises(c.PreparationError): c._proxy_guard(data, data["command"], c.STATE)

    def test_existing_private_log_appends_and_rejects_symlink_mode_and_hardlink(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            root = Path(scratch)
            log = root / "proxy.log"
            c._write_new(log, b"previous\n")
            with c._private_append(log) as stream: stream.write(b"new\n")
            self.assertEqual(log.read_bytes(), b"previous\nnew\n")
            info = list(log.stat())
            info[4] = os.getuid() + 1
            with patch.object(c.os, "fstat", return_value=os.stat_result(info)):
                with self.assertRaises(c.PreparationError): c._private_append(log)
            linked = root / "linked.log"
            linked.symlink_to(log)
            with self.assertRaises(OSError): c._private_append(linked)
            os.chmod(log, 0o644)
            with self.assertRaises(c.PreparationError): c._private_append(log)
            os.chmod(log, 0o600)
            os.link(log, root / "hardlink.log")
            with self.assertRaises(c.PreparationError): c._private_append(log)

    def test_dead_record_and_existing_log_recover_without_erasing_old_log(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            data = record(state)
            path = state / "comunicaciones/proxy-11025.json"
            c._write_new(path, json.dumps(data).encode())
            log = state / "comunicaciones/proxy-11025.log"
            c._write_new(log, b"old diagnostic\n")
            sink = {"NetworkSettings": {"Networks": {c.NETWORK: {"IPAddress": "192.168.224.2"}}}}
            process = MagicMock(pid=700124)
            process.poll.return_value = None
            with patch.object(c, "_proxy_guard", return_value=("dead", None)), \
                    patch.object(c.subprocess, "Popen", return_value=process) as popen, \
                    patch.object(c.socket, "socket"), patch.object(c.time, "sleep"), \
                    patch.object(c.os, "pidfd_open", return_value=99), patch.object(c.os, "close"), \
                    patch.object(c, "_process_snapshot", return_value={"start_ticks": "12346"}):
                result = c._ensure_proxies(sink, state)
            self.assertEqual(popen.call_count, 2)
            self.assertEqual(json.loads(c._private(path))["pid"], 700124)
            self.assertEqual(json.loads(c._private(path))["start_ticks"], "12346")
            self.assertEqual(log.read_bytes(), b"old diagnostic\n")
            self.assertEqual(len(result), 2)

    def test_missing_record_with_retained_log_can_recover(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            log = state / "comunicaciones/proxy-11025.log"
            c._write_new(log, b"retained\n")
            sink = {"NetworkSettings": {"Networks": {c.NETWORK: {"IPAddress": "192.168.224.2"}}}}
            process = MagicMock(pid=700124)
            process.poll.return_value = None
            with patch.object(c.subprocess, "Popen", return_value=process), patch.object(c.socket, "socket"), \
                    patch.object(c.time, "sleep"), patch.object(c.os, "pidfd_open", return_value=99), \
                    patch.object(c.os, "close"), patch.object(c, "_process_snapshot", return_value={"start_ticks": "12346"}):
                c._ensure_proxies(sink, state)
            self.assertEqual(log.read_bytes(), b"retained\n")

    def test_reused_pid_blocks_before_recovery_or_log_changes(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            path = state / "comunicaciones/proxy-11025.json"
            data = json.dumps(record(state)).encode()
            c._write_new(path, data)
            sink = {"NetworkSettings": {"Networks": {c.NETWORK: {"IPAddress": "192.168.224.2"}}}}
            with patch.object(c, "_proxy_guard", side_effect=c.PreparationError("mailpit_proxy_process_mismatch")), \
                    patch.object(c.subprocess, "Popen") as popen, patch.object(c.signal, "pidfd_send_signal") as send:
                with self.assertRaises(c.PreparationError): c._ensure_proxies(sink, state)
            popen.assert_not_called()
            send.assert_not_called()
            self.assertEqual(c._private(path), data)

    def test_record_publication_failure_cleans_only_new_pinned_child(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            sink = {"NetworkSettings": {"Networks": {c.NETWORK: {"IPAddress": "192.168.224.2"}}}}
            process = MagicMock(pid=700124)
            process.poll.return_value = None
            with patch.object(c.subprocess, "Popen", return_value=process), patch.object(c.socket, "socket"), \
                    patch.object(c.time, "sleep"), patch.object(c.os, "pidfd_open", return_value=99), \
                    patch.object(c.os, "close"), patch.object(c, "_process_snapshot", return_value={"start_ticks": "12346"}), \
                    patch.object(c, "_publish_record", side_effect=c.PreparationError("mailpit_proxy_record_changed")), \
                    patch.object(c, "_terminate_child") as terminate:
                with self.assertRaises(c.PreparationError): c._ensure_proxies(sink, state)
            terminate.assert_called_once_with(99)

    def test_record_replacement_denies_changed_preimage(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            path = Path(scratch) / "record.json"
            c._write_new(path, b"changed")
            with self.assertRaises(c.PreparationError): c._publish_record(path, {}, b"original")
            self.assertEqual(c._private(path), b"changed")

    def test_stop_rejects_foreign_container_before_signal_or_remove(self):
        foreign = resource()
        foreign["Config"]["Labels"][c.LABEL_OWNER] = "other"
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            with patch.object(c, "_scope"), patch.object(c, "_inspect", side_effect=[foreign, None, {"Id": "image:local"}]), \
                    patch.object(c, "_run") as run, patch.object(c.signal, "pidfd_send_signal") as send:
                result = c.stop(None, None, state, None, None, "docker")
            self.assertTrue(result["blockers"])
            run.assert_not_called()
            send.assert_not_called()

    def test_status_and_stop_preserve_foreign_attachment_and_reused_pid(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            network = {"Id": "network:owned", "Internal": True, "Labels": c._labels(state), "Containers": {"other": {}}}
            with patch.object(c, "_scope"), patch.object(c, "_inspect", side_effect=[None, network]), patch.object(c, "_run") as run:
                result = c.stop(None, None, state, None, None, "docker")
            self.assertEqual(result["blockers"][0]["detail"], "mailpit_network_foreign_attachment")
            run.assert_not_called()
            path = state / "comunicaciones/proxy-11025.json"
            c._write_new(path, json.dumps(record(state)).encode())
            for operation in (c.status, c.stop):
                with patch.object(c, "_scope"), patch.object(c, "_inspect", return_value=None), \
                        patch.object(c, "_proxy_guard", side_effect=c.PreparationError("mailpit_proxy_process_mismatch")), \
                        patch.object(c.signal, "pidfd_send_signal") as send:
                    result = operation(None, None, state, None, None, "docker")
                self.assertTrue(result["blockers"])
                send.assert_not_called()
                self.assertTrue(path.exists())

    def test_status_does_not_signal_and_stop_uses_pinned_descriptors(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            path = state / "comunicaciones/proxy-11025.json"
            c._write_new(path, json.dumps(record(state)).encode())
            parameters = (None, None, state, None, None, "docker")
            with patch.object(c, "_scope"), patch.object(c, "_inspect", return_value=None), \
                    patch.object(c, "_proxy_guard", return_value=("alive", 99)), patch.object(c.os, "close"), \
                    patch.object(c, "_terminate_child") as terminate:
                result = c.status(*parameters)
                terminate.assert_not_called()
                self.assertEqual(result["lifecycle"]["proxies"][0]["status"], "alive")
                self.assertTrue(path.exists())
                result = c.stop(*parameters)
            terminate.assert_called_once_with(99)
            self.assertFalse(path.exists())
            self.assertFalse(result["blockers"])

    def test_termination_signals_only_pinned_descriptor(self):
        with patch.object(c, "_pidfd_dead", side_effect=[False, True]), \
                patch.object(c.signal, "pidfd_send_signal") as send, patch.object(c.os, "kill") as kill:
            c._terminate_child(99)
        send.assert_called_once_with(99, c.signal.SIGTERM)
        kill.assert_not_called()

    def test_stop_removes_only_validated_resource_ids_with_loopback_bindings(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            sink = resource()
            sink["Id"] = "container:owned"
            sink["Config"]["Labels"] = c._labels(state)
            sink["Mounts"][0]["Source"] = str(state / "material/comunicaciones")
            sink["Mounts"][1]["Source"] = str(state / "comunicaciones/buzon")
            network = {"Id": "network:owned", "Internal": True, "Labels": c._labels(state),
                       "Containers": {"container:owned": {}}}
            with patch.object(c, "_scope"), patch.object(c, "_inspect", side_effect=[sink, network, {"Id": "image:local"}]), \
                    patch.object(c, "_run") as run:
                result = c.stop(None, None, state, None, None, "docker")
            self.assertFalse(result["blockers"])
            self.assertEqual(run.call_args_list[0].args[0], ["docker", "stop", "--time", "5", "container:owned"])
            self.assertEqual(run.call_args_list[1].args[0], ["docker", "network", "rm", "network:owned"])


if __name__ == "__main__":
    unittest.main()
