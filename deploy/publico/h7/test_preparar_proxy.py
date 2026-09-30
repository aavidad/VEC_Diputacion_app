import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import preparar_proxy as proxy


class TLSFixture:
    def __init__(self):
        self.temp = tempfile.TemporaryDirectory(prefix="vec-h7-synthetic-")
        self.root = Path(self.temp.name)
        self.ca = self.root / "ca.pem"
        self.server = self.root / "server.pem"
        self.server_key = self.root / "server.key"
        self.client = self.root / "client.pem"
        self.client_key = self.root / "client.key"
        self.run("req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
                 "-subj", "/CN=H7 synthetic CA", "-keyout", str(self.root / "ca.key"),
                 "-out", str(self.ca))
        for name, cert, key, purpose in (("server", self.server, self.server_key, "serverAuth"),
                                         ("client", self.client, self.client_key, "clientAuth")):
            csr = self.root / (name + ".csr")
            extensions = self.root / (name + ".ext")
            extensions.write_text("extendedKeyUsage=" + purpose + "\n" +
                                  ("subjectAltName=DNS:public.test,DNS:internal.test,DNS:edge.test\n"
                                   if name == "server" else ""))
            self.run("req", "-newkey", "rsa:2048", "-nodes", "-subj", "/CN=H7 synthetic " + name,
                     "-keyout", str(key), "-out", str(csr))
            self.run("x509", "-req", "-in", str(csr), "-CA", str(self.ca), "-CAkey",
                     str(self.root / "ca.key"), "-CAcreateserial", "-days", "1", "-extfile",
                     str(extensions), "-out", str(cert))
            key.chmod(0o600)
        self.ca.chmod(0o644)
        self.server.chmod(0o644)
        self.client.chmod(0o644)

    def run(self, *args):
        subprocess.run(["/usr/bin/openssl", *args], check=True, stdin=subprocess.DEVNULL,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)

    def config(self):
        return {"dedicated_public_caddy": True, "public_host": "edge.test", "public_port": 18443,
                "upstream_address": "127.0.0.1", "upstream_port": 19443,
                "upstream_server_name": "public.test", "upstream_ca_file": str(self.ca),
                "edge_certificate_file": str(self.server), "edge_key_file": str(self.server_key)}

    def private(self, name, value):
        file = self.root / name
        file.write_text(json.dumps(value))
        file.chmod(0o600)
        return str(file)


class PrepareProxyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = TLSFixture()

    @classmethod
    def tearDownClass(cls):
        cls.fixture.temp.cleanup()

    def test_full_host_https_trust_and_no_credentials_removed(self):
        result = proxy.render(self.fixture.config())
        self.assertIn("https://edge.test:18443 {", result)
        self.assertIn("reverse_proxy https://127.0.0.1:19443", result)
        self.assertIn("tls_trust_pool file", result)
        self.assertIn("tls_server_name public.test", result)
        self.assertIn("protocols h1", result)
        self.assertIn("@transfer header Transfer-Encoding *", result)
        for forbidden in ("tls_insecure_skip_verify", "handle_path", "trusted_proxies",
                          "header_up -Cookie", "header_up -Authorization", "header_up -X-Vec"):
            self.assertNotIn(forbidden, result)

    def test_ipv6_literal_is_bracketed(self):
        cfg = self.fixture.config()
        cfg["upstream_address"] = "::1"
        self.assertIn("https://[::1]:19443", proxy.render(cfg))

    def test_nonloopback_and_nonliteral_destinations_rejected(self):
        for address in ("localhost", "public.test", "0.0.0.0", "192.0.2.1", "::", "::ffff:127.0.0.1"):
            with self.subTest(address=address):
                cfg = self.fixture.config()
                cfg["upstream_address"] = address
                with self.assertRaises(ValueError):
                    proxy.render(cfg)

    def test_injection_and_unknown_insecure_options_rejected(self):
        for key, value in (("dedicated_public_caddy", False),
                           ("public_host", "*.example.test"), ("public_host", "edge.test\nrespond 200"),
                           ("public_host", "edge.test/path"), ("public_host", "localhost"),
                           ("public_port", True), ("upstream_port", 65536),
                           ("upstream_server_name", "{env.HOST}"),
                           ("upstream_ca_file", str(self.fixture.root / "{env.CA}")),
                           ("tls_insecure_skip_verify", True)):
            with self.subTest(key=key):
                cfg = self.fixture.config()
                cfg[key] = value
                with self.assertRaises((ValueError, OSError)):
                    proxy.render(cfg)

    def test_duplicate_keys_and_private_config_permissions(self):
        filename = self.fixture.root / "config.json"
        filename.write_text('{"public_host":"edge.test","public_host":"other.test"}')
        filename.chmod(0o600)
        with self.assertRaises(ValueError):
            proxy.private_json(str(filename))
        filename.write_text("{}")
        filename.chmod(0o644)
        with self.assertRaises(ValueError):
            proxy.private_json(str(filename))

    def test_symlink_hardlink_and_writable_files_rejected(self):
        target = self.fixture.root / "untrusted"
        target.write_text("{}")
        target.chmod(0o600)
        symlink = self.fixture.root / "symlink"
        symlink.symlink_to(target)
        with self.assertRaises(ValueError):
            proxy.read_file(str(symlink), private=True)
        link = self.fixture.root / "hardlink"
        os.link(target, link)
        with self.assertRaises(ValueError):
            proxy.read_file(str(target), private=True)
        link.unlink()
        target.chmod(0o622)
        with self.assertRaises(ValueError):
            proxy.read_file(str(target))

    def test_output_is_private_exclusive_and_errors_do_not_leak_paths(self):
        cfg = self.fixture.private("valid.json", self.fixture.config())
        output = str(self.fixture.root / "Caddyfile")
        result = subprocess.run(["python3", "-B", str(Path(proxy.__file__)), "--config", cfg,
                                 "--output", output], capture_output=True, text=True, timeout=15)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(Path(output).stat().st_mode & 0o777, 0o600)
        self.assertEqual(json.loads(result.stdout)["sha256"],
                         hashlib.sha256(Path(output).read_bytes()).hexdigest())
        again = subprocess.run(["python3", "-B", str(Path(proxy.__file__)), "--config", cfg,
                                "--output", output], capture_output=True, text=True, timeout=15)
        self.assertEqual(again.returncode, 1)
        self.assertEqual(again.stderr.strip(), "proxy_preparation_rejected")
        self.assertNotIn(self.fixture.temp.name, again.stdout + again.stderr)


if __name__ == "__main__":
    unittest.main()
