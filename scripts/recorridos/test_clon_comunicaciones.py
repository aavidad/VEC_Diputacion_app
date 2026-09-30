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


def clone_fixture(state, container="vec-test-generic", pg_port=55532):
    clone = {"propietario": c.OWNER, "contenedor": container, "puerto_pg": pg_port,
             "puerto_web": 18534, "commit": "b" * 40}
    c._write_new(state / "clon.json", json.dumps(clone).encode())
    return clone


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
                return b"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"
            return required[argv[-1].split(":", 1)[1]].encode()
        with patch.object(c, "_run", side_effect=run):
            c._source_contract(Path("/source"), "b" * 40)
        self.assertEqual(calls[0][-1], "b" * 40 + "^{commit}")
        self.assertTrue(all("show" in argv for argv in calls[1:]))

    def test_preflight_checks_ports_without_writing_or_running_mailpit(self):
        parameters = dict(repo=Path("/source"), container=c.PG_CONTAINER, state=c.STATE,
                          material=c.STATE / "material", pg_port=55531, engine="docker")
        proof = b'{"roles_version":2,"assignments_version":2,"surfaces":2,"grants_per_role":10,"commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
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
        proof = b'{"roles_version":2,"assignments_version":2,"surfaces":2,"grants_per_role":10,"commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
        def inspect(engine, kind, name):
            if kind == "image": return {"Id": "image:local"}
            if kind == "network": return {"Internal": False, "Labels": c._labels(c.STATE)}
            return None
        with patch.object(c, "_scope", return_value=(Path("/source"), c.STATE, c.STATE / "material")), \
                patch.object(c, "_source_contract", return_value="a" * 40), \
                patch.object(c, "_private", return_value=proof), patch.object(c, "_inspect", side_effect=inspect):
            result = c.preflight(None, c.PG_CONTAINER, None, None, 55531, "docker")
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
                result = c.stop(None, "vec-test-clone", state, state / "material", 55531, "docker")
            self.assertTrue(result["blockers"])
            run.assert_not_called()
            send.assert_not_called()

    def test_status_and_stop_preserve_foreign_attachment_and_reused_pid(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            network = {"Id": "network:owned", "Internal": True, "Labels": c._labels(state), "Containers": {"other": {}}}
            with patch.object(c, "_scope"), patch.object(c, "_inspect", side_effect=[None, network]), patch.object(c, "_run") as run:
                result = c.stop(None, "vec-test-clone", state, state / "material", 55531, "docker")
            self.assertEqual(result["blockers"][0]["detail"], "mailpit_network_foreign_attachment")
            run.assert_not_called()
            path = state / "comunicaciones/proxy-11025.json"
            c._write_new(path, json.dumps(record(state)).encode())
            for operation in (c.status, c.stop):
                with patch.object(c, "_scope"), patch.object(c, "_inspect", return_value=None), \
                        patch.object(c, "_proxy_guard", side_effect=c.PreparationError("mailpit_proxy_process_mismatch")), \
                        patch.object(c.signal, "pidfd_send_signal") as send:
                    result = operation(None, "vec-test-clone", state, state / "material", 55531, "docker")
                self.assertTrue(result["blockers"])
                send.assert_not_called()
                self.assertTrue(path.exists())

    def test_status_does_not_signal_and_stop_uses_pinned_descriptors(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            c._directory(state / "comunicaciones")
            path = state / "comunicaciones/proxy-11025.json"
            c._write_new(path, json.dumps(record(state)).encode())
            parameters = (None, "vec-test-clone", state, state / "material", 55531, "docker")
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
            sink["HostConfig"]["NetworkMode"] = c._build_target(state, "vec-test-clone", 55531, 11025, 18532)["network"]
            sink["Mounts"][0]["Source"] = str(state / "material/comunicaciones")
            sink["Mounts"][1]["Source"] = str(state / "comunicaciones/buzon")
            network = {"Id": "network:owned", "Internal": True, "Labels": c._labels(state),
                       "Containers": {"container:owned": {}}}
            with patch.object(c, "_scope"), patch.object(c, "_inspect", side_effect=[sink, network, {"Id": "image:local"}]), \
                    patch.object(c, "_run") as run:
                result = c.stop(None, "vec-test-clone", state, state / "material", 55531, "docker")
            self.assertFalse(result["blockers"])
            self.assertEqual(run.call_args_list[0].args[0], ["docker", "stop", "--time", "5", "container:owned"])
            self.assertEqual(run.call_args_list[1].args[0], ["docker", "network", "rm", "network:owned"])

    def test_generic_configure_before_h4_persists_explicit_ports_for_six_argument_api(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            postgres = {"Config": {"Labels": c._labels(state)}}
            def inspect(engine, kind, name):
                return postgres if kind == "container" and name == "vec-test-generic" else None
            arguments = (state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            with patch.object(c, "_inspect", side_effect=inspect), patch.object(c, "_free_ports") as free, \
                    patch.object(c, "_certificate") as certificate, patch.object(c, "_run") as run:
                result = c.configure(*arguments, smtp_port=12025, http_port=12026)
                before = c._private(state / "comunicaciones/target.json")
                replay = c.configure(*arguments)
            free.assert_called_with((12025, 12026))
            certificate.assert_not_called()
            run.assert_not_called()
            self.assertEqual(before, c._private(state / "comunicaciones/target.json"))
            self.assertEqual(result["env"], {})
            self.assertEqual(result["profiles"], replay["profiles"])
            target = c._target(state, "vec-test-generic", 55532)
            self.assertEqual((target["smtp_port"], target["http_port"]), (12025, 12026))
            env = c._result(smtp=True, target=target)["env"]
            self.assertEqual(env["VEC_SMTP_PORT"], "12025")
            self.assertEqual(env["VEC_SMTP_CA_FILE"], str(state / "material/ca/ca.crt"))

    def test_target_names_depend_on_state_and_pg_identity_and_preserve_legacy(self):
        legacy = c._build_target(c.STATE, c.PG_CONTAINER, 55531, 11025, 18532)
        self.assertEqual(legacy["container"], c.MAIL_CONTAINER)
        self.assertEqual(legacy["network"], c.NETWORK)
        first = c._build_target(Path.home() / "one", "vec-one", 55532, 12025, 12026)
        second = c._build_target(Path.home() / "two", "vec-one", 55532, 12027, 12028)
        third = c._build_target(Path.home() / "one", "vec-two", 55533, 12029, 12030)
        self.assertEqual(len({first["container"], second["container"], third["container"]}), 3)
        self.assertEqual(len({first["network"], second["network"], third["network"]}), 3)
        self.assertEqual(first, c._build_target(Path.home() / "one", "vec-one", 55532, 12025, 12026))

    def test_busy_ports_do_not_publish_target_and_explicit_changes_do_not_replace(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            arguments = (state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            with patch.object(c, "_scope", return_value=(state / "repo", state, state / "material")), \
                    patch.object(c, "_inspect", return_value=None), \
                    patch.object(c, "_free_ports", side_effect=c.PreparationError("mailpit_loopback_port_busy")):
                with self.assertRaises(c.PreparationError): c.configure(*arguments, smtp_port=12025, http_port=12026)
            path = state / "comunicaciones/target.json"
            self.assertFalse(path.exists())
            target = c._build_target(state, "vec-test-generic", 55532, 12025, 12026)
            c._write_new(path, json.dumps(target).encode())
            before = c._private(path)
            with self.assertRaises(c.PreparationError): c._target(state, "vec-test-generic", 55532, smtp_port=12027)
            self.assertEqual(before, c._private(path))
            with self.assertRaises(c.PreparationError): c._target(state, "vec-other", 55532)
            with self.assertRaises(c.PreparationError): c._build_target(state, "vec-test-generic", 55532, 55532, 12026)
            with self.assertRaises(c.PreparationError): c._target(state, "vec-test-generic", 55532, http_port=18534)

    def test_generic_pg_scope_denies_foreign_labels_before_configuration(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            foreign = {"Config": {"Labels": {c.LABEL_OWNER: "other", c.LABEL_STATE: str(state)}}}
            with patch.object(c, "_inspect", return_value=foreign), patch.object(c, "_free_ports") as free:
                with self.assertRaises(c.PreparationError):
                    c.configure(state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            free.assert_not_called()
            self.assertFalse((state / "comunicaciones/target.json").exists())

    def test_stop_and_status_are_idempotent_when_owned_postgres_is_already_removed(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            arguments = (state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            with patch.object(c, "_inspect", return_value=None), patch.object(c, "_run") as run, \
                    patch.object(c.signal, "pidfd_send_signal") as send:
                for operation in (c.status, c.stop, c.stop):
                    result = operation(*arguments)
                    self.assertFalse(result["blockers"])
                with self.assertRaises(c.PreparationError): c.configure(*arguments)
                with self.assertRaises(c.PreparationError): c._scope(*arguments)
            run.assert_not_called()
            send.assert_not_called()

    def test_lifecycle_with_removed_postgres_still_denies_foreign_sink(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            target = c._build_target(state, "vec-test-generic", 55532, 11025, 18532)
            foreign = {"Image": "image:local", "Config": {"Labels": {c.LABEL_OWNER: "other"}}}
            def inspect(engine, kind, name):
                if kind == "container" and name == target["container"]: return foreign
                if kind == "image": return {"Id": "image:local"}
                return None
            with patch.object(c, "_inspect", side_effect=inspect), patch.object(c, "_run") as run, \
                    patch.object(c.signal, "pidfd_send_signal") as send:
                result = c.stop(state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            self.assertTrue(result["blockers"])
            self.assertEqual(result["blockers"][0]["detail"], "foreign_container")
            run.assert_not_called()
            send.assert_not_called()

    def test_generic_sink_only_accepts_selected_ports_and_own_network(self):
        state = Path.home() / "synthetic-generic"
        target = c._build_target(state, "vec-generic", 55532, 12025, 12026)
        sink = resource()
        sink["Config"]["Labels"] = c._labels(state)
        sink["HostConfig"]["NetworkMode"] = target["network"]
        sink["HostConfig"]["PortBindings"] = {"1025/tcp": [{"HostIp": "127.0.0.1", "HostPort": "12025"}],
                                             "8025/tcp": [{"HostIp": "127.0.0.1", "HostPort": "12026"}]}
        sink["Mounts"][0]["Source"] = str(state / "material/comunicaciones")
        sink["Mounts"][1]["Source"] = str(state / "comunicaciones/buzon")
        c._validate_sink(sink, state, "image:local", target=target)
        command = c._proxy_command("192.168.224.2", 12025, 1025)
        self.assertIn("bind=127.0.0.1", command[-2])
        self.assertEqual(c._record_command({"command": command}, 12025, target=target), command)
        sink["HostConfig"]["PortBindings"]["1025/tcp"][0]["HostPort"] = "11025"
        with self.assertRaises(c.PreparationError): c._validate_sink(sink, state, "image:local", target=target)

    def test_preflight_uses_actual_clone_commit_and_configured_ports_without_worktree_reads(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone = clone_fixture(state)
            c._directory(state / "comunicaciones")
            target = c._build_target(state, "vec-test-generic", 55532, 12025, 12026)
            c._write_new(state / "comunicaciones/target.json", json.dumps(target).encode())
            c._write_new(state / "usuarios-h4-result.json", b'{"roles_version":2,"assignments_version":2,"surfaces":2,"grants_per_role":10}')
            c._directory(state / "material")
            c._directory(state / "material/ca")
            c._write_new(state / "material/ca/ca.crt", b"synthetic CA")
            c._write_new(state / "material/ca/ca.key", b"synthetic key")
            def inspect(engine, kind, name):
                if kind == "image": return {"Id": "image:local"}
                if kind == "container" and name == "vec-test-generic": return {"Config": {"Labels": c._labels(state)}}
                return None
            with patch.object(c, "_inspect", side_effect=inspect), patch.object(c, "_free_ports") as free, \
                    patch.object(c, "_source_contract", return_value=clone["commit"]) as source:
                result = c.preflight(state / "repo", "vec-test-generic", state, state / "material", 55532, "docker")
            source.assert_called_once_with(state / "repo", clone["commit"])
            free.assert_called_once_with((12025, 12026))
            self.assertFalse(result["blockers"])
            self.assertEqual(result["profiles"]["usuarios_comunicaciones"]["source_commit"], clone["commit"])

    def test_relative_api_state_uses_canonical_scope_for_provision_and_lifecycle(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as scratch:
            state = Path(scratch)
            clone_fixture(state)
            target = c._build_target(state, "vec-test-generic", 55532, 12025, 12026)
            result = c._result(image=True, target=target)
            with patch.object(c, "preflight", return_value=result), patch.object(c, "_proxy_lock") as lock, \
                    patch.object(c, "_provision_prepared", return_value=result) as prepare:
                c.provision(state / "repo", "vec-test-generic", Path(state.name), Path(state.name) / "material", 55532, "docker")
            lock.assert_called_once_with(state)
            prepare.assert_called_once_with(result, state, state / "material", "docker")
            with patch.object(c.os, "getcwd", return_value=str(state.parent)), patch.object(c, "_inspect", return_value=None):
                result = c.status(state / "repo", "vec-test-generic", Path(state.name), Path(state.name) / "material", 55532, "docker")
            self.assertFalse(result["blockers"])
            self.assertEqual(result["lifecycle"]["target"]["state"], str(state))


if __name__ == "__main__":
    unittest.main()
