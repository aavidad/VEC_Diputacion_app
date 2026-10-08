import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import runtime


def sha256(data):
    return hashlib.sha256(data).hexdigest()


class RuntimeConfigTest(unittest.TestCase):
    def test_lector_publico_acepta_solo_dos_opciones_tls_unicas_y_no_vacias(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / "vec-publico"
            binary.write_bytes(b"binario sintetico")
            manifest = root / "web/produccion.manifest"
            manifest.parent.mkdir()
            manifest.write_bytes(b"static/bolsa/index.html\n")
            page = root / "web/static/bolsa/index.html"
            page.parent.mkdir(parents=True)
            page.write_bytes(b"pagina publica sintetica")
            key = root / "clave-sintetica"
            key.write_bytes(b"sin material TLS")
            key.chmod(0o600)
            cfg_path = root / "runtime.json"
            base_dsn = ("postgres://vec_publico_login:clave-sintetica@localhost:55441/"
                        "vec_bolsa_publica?sslmode=verify-full&sslrootcert=/privado/ca.crt")
            cfg = {
                "binary": str(binary), "binary_sha256": sha256(binary.read_bytes()),
                "web_sha256": {"produccion.manifest": sha256(manifest.read_bytes()),
                               "static/bolsa/index.html": sha256(page.read_bytes())},
                "environment": {
                    "VEC_HTTP_ADDR": "127.0.0.1:18443",
                    "VEC_TLS_CERT_FILE": str(root / "certificado-sintetico"),
                    "VEC_TLS_KEY_FILE": str(key),
                    "VEC_BOLSA_PUBLICA_DATABASE_URL": base_dsn,
                    "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256": "a" * 64,
                    "VEC_BOLSA_CATEGORIES_CATALOG_ID": "categorias-publicas",
                    "VEC_BOLSA_CATEGORIES_CATALOG_VERSION": "1",
                    "VEC_BOLSA_CATEGORIES_CATALOG_SHA256": "b" * 64,
                    "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256": "c" * 64,
                },
                "check": {"origin": "https://localhost:18443", "ca_file": "/privado/ca.crt",
                          "convocatoria": "convocatoria:sintetica", "bolsa": "bolsa:sintetica"},
            }

            def cargar(dsn):
                cfg["environment"]["VEC_BOLSA_PUBLICA_DATABASE_URL"] = dsn
                cfg_path.write_text(json.dumps(cfg))
                cfg_path.chmod(0o600)
                with patch.object(runtime.ssl, "SSLContext"):
                    return runtime.load(str(cfg_path))

            _, entorno = cargar(base_dsn)
            self.assertEqual(entorno["VEC_AUTH_MODE"], "disabled")
            for adicional in ("&sslmode=", "&sslmode=verify-full", "&sslrootcert=",
                              "&sslrootcert=/privado/otra.crt", "&options="):
                with self.subTest(adicional=adicional):
                    with self.assertRaisesRegex(ValueError, "database TLS"):
                        cargar(base_dsn + adicional)


if __name__ == "__main__":
    unittest.main()
