import copy
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import threading
import time
import unittest
from unittest.mock import patch

import comprobar_coexistencia as coexist
import preparar_proxy as proxy
from test_preparar_proxy import TLSFixture

BODY = b"h7-synthetic\n"
BODY_HASH = hashlib.sha256(BODY).hexdigest()


class SyntheticServer:
    def __init__(self, fixture, *, internal=False):
        self.requests = []
        self.failure = None
        self.internal = internal
        owner = self

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                owner.requests.append((self.path, dict(self.headers)))
                dangerous = any(key.lower() in ("cookie", "authorization", "proxy-authorization",
                                                "forwarded", "via", "x-remote-user", "remote-user")
                                or key.lower().startswith(("x-vec-", "x-auth-", "x-forwarded-"))
                                for key in self.headers)
                status = 400 if dangerous else 404 if not internal and self.path in coexist.PRIVATE_ROUTES else 200
                if internal and self.connection.selected_alpn_protocol() != "http/1.1":
                    status = 400
                body = BODY
                if owner.failure == "private" and self.path in coexist.PRIVATE_ROUTES:
                    status = 200
                if owner.failure == "body" and self.path == "/readyz":
                    body = b"drift\n"
                if owner.failure == "redirect":
                    status = 302
                self.send_response(status)
                if owner.failure == "cookie":
                    self.send_header("Set-Cookie", "synthetic=1")
                if owner.failure == "location":
                    self.send_header("Location", "https://external.test/")
                if owner.failure == "redirect":
                    self.send_header("Location", "https://external.test/")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *args):
                pass

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(fixture.server, fixture.server_key)
        if internal:
            context.verify_mode = ssl.CERT_REQUIRED
            context.load_verify_locations(fixture.ca)
            context.set_alpn_protocols(["http/1.1"])
        self.server.socket = context.wrap_socket(self.server.socket, server_side=True)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.fixture = fixture

    def endpoint(self):
        routes = coexist.INTERNAL_REQUIRED if self.internal else coexist.PUBLIC_REQUIRED | {
            "/api/publico/bolsa/convocatorias/synthetic", "/api/publico/bolsa/bolsas/synthetic/lista"}
        endpoint = {"address": "127.0.0.1", "port": self.server.server_port,
                    "server_name": "internal.test" if self.internal else "public.test",
                    "ca_file": str(self.fixture.ca),
                    "checks": [{"path": path, "sha256": BODY_HASH} for path in sorted(routes)]}
        if self.internal:
            endpoint.update(client_certificate_file=str(self.fixture.client),
                            client_key_file=str(self.fixture.client_key))
        return endpoint

    def close(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=5)


class CoexistenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = TLSFixture()
        cls.public = SyntheticServer(cls.fixture)
        cls.internal = SyntheticServer(cls.fixture, internal=True)

    @classmethod
    def tearDownClass(cls):
        cls.public.close()
        cls.internal.close()
        cls.fixture.temp.cleanup()

    def setUp(self):
        self.public.failure = None
        self.internal.failure = None

    def config(self):
        binary = self.fixture.root / "synthetic-binary"
        binary.write_bytes(b"synthetic provenance only")
        binary.chmod(0o644)
        evidence = self.fixture.root / "h6-evidence"
        evidence.write_bytes(b"synthetic evidence, no READY declaration")
        evidence.chmod(0o600)
        return {"fuente_commit": "1" * 40, "binario_file": str(binary),
                "binario_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
                "evidencia_h6_file": str(evidence),
                "evidencia_h6_sha256": hashlib.sha256(evidence.read_bytes()).hexdigest(),
                "publico": self.public.endpoint(), "interno": self.internal.endpoint(), "rondas": 2}

    def test_real_https_mtls_concurrent_rounds_and_continuity(self):
        filename = self.fixture.private("coexist.json", self.config())
        cfg = coexist.load_config(filename)
        result = coexist.check(cfg)
        self.assertEqual(result["result"], "checks_passed")
        self.assertFalse(result["h6_ready_accredited"])
        self.assertFalse(result["writes"])
        self.assertGreater(len(self.public.requests), 20)

    def test_wrong_ca_and_wrong_name_fail(self):
        endpoint = self.public.endpoint()
        endpoint["server_name"] = "untrusted.test"
        with self.assertRaises(ssl.SSLError):
            coexist.fetch(endpoint, "/")
        other = TLSFixture()
        try:
            endpoint = self.public.endpoint()
            endpoint["ca_file"] = str(other.ca)
            with self.assertRaises(ssl.SSLError):
                coexist.fetch(endpoint, "/")
        finally:
            other.temp.cleanup()

    def test_internal_requires_client_certificate(self):
        coexist.anonymous_internal_denied(self.internal.endpoint())
        with self.assertRaises((ssl.SSLError, ConnectionResetError)):
            coexist.fetch(self.internal.endpoint(), "/portal-empleado/", client=False)
        self.assertEqual(coexist.fetch(self.internal.endpoint(), "/portal-empleado/")[0], 200)

    def test_internal_requires_negotiated_http11_alpn(self):
        endpoint = self.internal.endpoint()
        context = proxy.tls_context(endpoint["ca_file"], endpoint["client_certificate_file"],
                                    endpoint["client_key_file"])
        context.set_alpn_protocols([])
        with patch.object(coexist, "tls_context", return_value=context):
            self.assertEqual(coexist.fetch(endpoint, "/portal-empleado/")[0], 400)
        self.assertEqual(coexist.fetch(endpoint, "/portal-empleado/")[0], 200)

    def test_cookies_redirects_location_and_private_leak_fail(self):
        for failure in ("cookie", "redirect", "location", "private", "body"):
            with self.subTest(failure=failure):
                self.public.failure = failure
                with self.assertRaises(ValueError):
                    coexist.check(self.config())

    def test_provenance_schema_routes_and_listener_guards(self):
        cases = []
        cfg = self.config()
        for key, value in (("evidencia_h6_sha256", "2" * 64), ("binario_sha256", "3" * 64),
                           ("fuente_commit", "wrong"), ("rondas", True), ("unknown", True)):
            candidate = copy.deepcopy(cfg)
            candidate[key] = value
            cases.append(candidate)
        candidate = copy.deepcopy(cfg)
        candidate["publico"]["address"] = "192.0.2.1"
        cases.append(candidate)
        candidate = copy.deepcopy(cfg)
        candidate["publico"]["port"] = candidate["interno"]["port"]
        cases.append(candidate)
        candidate = copy.deepcopy(cfg)
        candidate["interno"]["checks"][0]["path"] = "/mutar"
        cases.append(candidate)
        for index, candidate in enumerate(cases):
            with self.subTest(index=index):
                filename = self.fixture.private("bad-check.json", candidate)
                with self.assertRaises(ValueError):
                    coexist.load_config(filename)

    def test_barrier_starts_both_probes_and_round_failure_is_not_hidden(self):
        seen = []

        def synthetic_probe(endpoint, *, internal, barrier):
            barrier.wait(timeout=2)
            seen.append(internal)
            return {"/portal-empleado/": BODY_HASH} if internal else {"/": BODY_HASH}

        with patch.object(coexist, "probe", synthetic_probe), \
                patch.object(coexist, "anonymous_internal_denied"), \
                patch.object(coexist, "fetch", return_value=(200, BODY_HASH)):
            coexist.check(self.config())
        self.assertEqual(sorted(seen), [False, False, True, True])


@unittest.skipUnless(os.environ.get("H7_CADDY_BINARY"), "local Caddy binary not provided")
class CaddyBoundaryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = TLSFixture()
        cls.upstream = SyntheticServer(cls.fixture)
        cls.binary = os.environ["H7_CADDY_BINARY"]
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            cls.edge_port = sock.getsockname()[1]
        cfg = cls.fixture.config()
        cfg.update(public_port=cls.edge_port, upstream_port=cls.upstream.server.server_port)
        cls.cfg = cfg
        cls.config_file = cls.fixture.root / "Caddyfile"
        cls.config_file.write_text("{\n admin off\n auto_https disable_redirects\n}\n" +
                                   proxy.render(cfg).replace("    tls ", "    bind 127.0.0.1\n    tls ", 1))
        cls.env = {"PATH": "/usr/bin:/bin", "XDG_DATA_HOME": str(cls.fixture.root / "data"),
                   "XDG_CONFIG_HOME": str(cls.fixture.root / "config")}
        for action in ("adapt", "validate"):
            result = subprocess.run([cls.binary, action, "--config", str(cls.config_file),
                                     "--adapter", "caddyfile"], env=cls.env, capture_output=True,
                                    text=True, timeout=15)
            if result.returncode:
                cls.upstream.close()
                cls.fixture.temp.cleanup()
                raise AssertionError(result.stderr)
        cls.process = subprocess.Popen([cls.binary, "run", "--config", str(cls.config_file),
                                        "--adapter", "caddyfile"], env=cls.env,
                                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        cls.edge = cls.upstream.endpoint()
        cls.edge.update(port=cls.edge_port, server_name="edge.test")
        for _ in range(50):
            try:
                if coexist.fetch(cls.edge, "/")[0] == 200:
                    return
            except OSError:
                pass
            time.sleep(0.1)
        cls.tearDownClass()
        raise AssertionError("synthetic Caddy did not become ready")

    @classmethod
    def tearDownClass(cls):
        cls.process.terminate()
        cls.process.wait(timeout=10)
        cls.upstream.close()
        cls.fixture.temp.cleanup()

    def test_clean_request_and_no_generated_identity_headers(self):
        self.assertEqual(coexist.fetch(self.edge, "/")[0], 200)
        forwarded = self.upstream.requests[-1][1]
        self.assertEqual(forwarded["Host"], "public.test")
        for header in ("X-Forwarded-For", "X-Forwarded-Proto", "X-Forwarded-Host", "Via"):
            self.assertNotIn(header, forwarded)

    def test_incoming_proxy_metadata_including_empty_is_rejected_before_upstream(self):
        for header in ("X-Forwarded-For", "X-Forwarded-Proto", "X-Forwarded-Host",
                       "Forwarded", "Via", "Proxy-Authorization"):
            for value in ("", "h7-synthetic"):
                with self.subTest(header=header, value=value):
                    before = len(self.upstream.requests)
                    self.assertEqual(coexist.fetch(self.edge, "/", headers={header: value})[0], 400)
                    self.assertEqual(len(self.upstream.requests), before)

    def test_credentials_preserved_for_backend_rejection(self):
        for header in ("Cookie", "Authorization", "X-Vec-Actor", "X-Auth-User", "Remote-User"):
            for value in ("", "h7-synthetic"):
                with self.subTest(header=header):
                    before = len(self.upstream.requests)
                    self.assertEqual(coexist.fetch(self.edge, "/", headers={header: value})[0], 400)
                    self.assertEqual(len(self.upstream.requests), before + 1)
                    self.assertIn(header, self.upstream.requests[-1][1])

    def test_sensitive_connection_tokens_and_repeated_headers_are_rejected(self):
        for token in ("Cookie", "Authorization", "Proxy-Authorization", "X-Vec-Actor",
                      "X-Auth-User", "X-Forwarded-Host", "Remote-User", "Forwarded", "Via", "Trailer"):
            with self.subTest(token=token):
                before = len(self.upstream.requests)
                connection = coexist.LoopbackHTTPS(self.edge, proxy.tls_context(self.edge["ca_file"]))
                try:
                    connection.putrequest("GET", "/")
                    connection.putheader("Connection", "keep-alive")
                    connection.putheader("Connection", "upgrade, " + token.lower())
                    connection.putheader(token, "h7-synthetic")
                    connection.endheaders()
                    response = connection.getresponse()
                    self.assertEqual(response.status, 400)
                    response.read()
                finally:
                    connection.close()
                self.assertEqual(len(self.upstream.requests), before)

    def test_caddy_rejects_wrong_upstream_name_and_ca(self):
        other = TLSFixture()
        try:
            for changed in ({"upstream_server_name": "wrong.test"},
                            {"upstream_ca_file": str(other.ca)}):
                with self.subTest(changed=list(changed)):
                    with socket.socket() as sock:
                        sock.bind(("127.0.0.1", 0))
                        edge_port = sock.getsockname()[1]
                    cfg = dict(self.cfg, public_port=edge_port, **changed)
                    config_file = self.fixture.root / "Caddyfile-tls-negative"
                    config_file.write_text("{\n admin off\n auto_https disable_redirects\n}\n" +
                                           proxy.render(cfg).replace("    tls ", "    bind 127.0.0.1\n    tls ", 1))
                    process = subprocess.Popen([self.binary, "run", "--config", str(config_file),
                                                "--adapter", "caddyfile"], env=self.env,
                                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                    endpoint = dict(self.edge, port=edge_port)
                    before = len(self.upstream.requests)
                    try:
                        for _ in range(50):
                            try:
                                self.assertEqual(coexist.fetch(endpoint, "/")[0], 502)
                                break
                            except ConnectionRefusedError:
                                time.sleep(0.1)
                        else:
                            self.fail("synthetic negative Caddy did not become ready")
                        self.assertEqual(len(self.upstream.requests), before)
                    finally:
                        process.terminate()
                        process.wait(timeout=10)
        finally:
            other.temp.cleanup()


if __name__ == "__main__":
    unittest.main()
