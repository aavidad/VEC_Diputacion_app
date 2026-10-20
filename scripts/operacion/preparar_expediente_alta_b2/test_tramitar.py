"""La intención de una actuación se conserva antes de emitir su POST."""
import contextlib
import io
import json
import os
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from unittest.mock import patch

import tramitar as cliente


class TramitarTest(unittest.TestCase):
    def test_certificados_del_centro_generados_por_fabrica(self):
        with tempfile.TemporaryDirectory() as directory:
            material = Path(directory)
            (material / "mtls").mkdir()
            for actor, name in (("centro-solicitante", "solicitante"),
                                ("centro-ratificador", "ratificador")):
                cert = material / "mtls" / (name + ".crt")
                key = material / "mtls" / (name + ".key")
                cert.write_bytes(b"certificado-ficticio")
                key.write_bytes(b"clave-ficticia")
                self.assertEqual(cliente.mTLS(SimpleNamespace(material=material), actor),
                                 (cert, key))

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.material = self.root / "material"
        (self.material / "mtls").mkdir(parents=True)
        self.material.chmod(0o700)
        (self.material / "ca").mkdir()
        for path, content in ((self.material / "mtls/cliente.crt", b"certificado-ficticio"),
                              (self.material / "mtls/cliente.key", b"clave-ficticia"),
                              (self.material / "ca/ca.crt", b"ca-ficticia")):
            path.write_bytes(content)
            path.chmod(0o600)
        self.entrada = self.root / "entrada.json"
        self.estado = self.root / "estado.json"
        self.body = {"expediente_ref": "expediente:ficticio", "version_esperada": 1,
                     "clave_idempotencia": "intencion:ficticia"}
        self.write_body(self.body)
        self.receipt = {"recibo_ref": "recibo:ficticio", "expediente_ref": "expediente:ficticio",
                        "version_resultante": 2}

    def write_body(self, body):
        self.entrada.write_text(json.dumps(body))
        self.entrada.chmod(0o600)

    def run_client(self, *, origen="https://127.0.0.1:8443"):
        argv = ["--operacion", "analisis", "--origen", origen,
                "--material", str(self.material), "--estado", str(self.estado),
                "--entrada", str(self.entrada)]
        with contextlib.redirect_stdout(io.StringIO()) as output:
            result = cliente.main(argv)
        self.assertNotIn("recibo:ficticio", output.getvalue())
        self.assertNotIn("certificado-ficticio", output.getvalue())
        return result

    def test_rechaza_origen_y_destino_ajenos(self):
        with self.assertRaises(cliente.Corte):
            self.run_client(origen="https://servidor-ajeno:8443")
        self.assertFalse(self.estado.exists())
        with patch.object(cliente, "data_http", side_effect=cliente.Corte("respuesta incierta")):
            with self.assertRaises(cliente.Corte):
                self.run_client()
        with patch.object(cliente, "data_http") as post:
            with self.assertRaises(cliente.Corte):
                self.run_client(origen="https://127.0.0.1:8444")
            post.assert_not_called()
        (self.material / "mtls/cliente.crt").write_bytes(b"otro-certificado-ficticio")
        with patch.object(cliente, "data_http") as post:
            with self.assertRaises(cliente.Corte):
                self.run_client()
            post.assert_not_called()

    def test_rechaza_clave_abierta_sin_crear_intencion(self):
        (self.material / "mtls/cliente.key").chmod(0o644)
        with patch.object(cliente, "data_http") as post:
            with self.assertRaisesRegex(cliente.Corte, "clave mTLS privada insegura"):
                self.run_client()
            post.assert_not_called()
        self.assertFalse(self.estado.exists())

    def test_rechaza_directorio_de_estado_abierto_o_ajeno(self):
        self.root.chmod(0o777)
        try:
            with patch.object(cliente, "data_http") as post:
                with self.assertRaisesRegex(cliente.Corte, "directorio del estado privado inseguro"):
                    self.run_client()
                post.assert_not_called()
            self.assertFalse(self.estado.exists())
        finally:
            self.root.chmod(0o700)
        original_stat = Path.stat

        def uid_ajeno(path, *args, **kwargs):
            info = original_stat(path, *args, **kwargs)
            if path == self.root:
                fields = list(info)
                fields[4] = os.getuid() + 1
                return os.stat_result(fields)
            return info

        with patch.object(Path, "stat", uid_ajeno), patch.object(cliente, "data_http") as post:
            with self.assertRaisesRegex(cliente.Corte, "directorio del estado privado inseguro"):
                self.run_client()
            post.assert_not_called()
        self.assertFalse(self.estado.exists())

    def test_rechaza_cuerpo_alterado_y_respuesta_vacia(self):
        with patch.object(cliente, "data_http", return_value={}):
            with self.assertRaises(cliente.Corte):
                self.run_client()
        state = json.loads(self.estado.read_text())
        self.assertEqual(state["operaciones"]["analisis"]["estado"], "pendiente")
        self.assertEqual(self.estado.stat().st_mode & 0o777, 0o600)
        self.write_body({**self.body, "version_esperada": 2})
        with patch.object(cliente, "data_http") as post:
            with self.assertRaises(cliente.Corte):
                self.run_client()
            post.assert_not_called()

    def test_rechaza_cuerpo_guardado_alterado_antes_de_reintentar(self):
        with patch.object(cliente, "data_http", side_effect=cliente.Corte("respuesta incierta")):
            with self.assertRaises(cliente.Corte):
                self.run_client()
        state = json.loads(self.estado.read_text())
        state["operaciones"]["analisis"]["cuerpo"]["version_esperada"] = 9
        self.estado.write_text(json.dumps(state))
        self.estado.chmod(0o600)
        with patch.object(cliente, "data_http") as post:
            with self.assertRaises(cliente.Corte):
                self.run_client()
            post.assert_not_called()

    def test_reintenta_misma_intencion_y_no_reenvia_confirmada(self):
        with patch.object(cliente, "data_http", side_effect=cliente.Corte("respuesta incierta")) as post:
            with self.assertRaises(cliente.Corte):
                self.run_client()
            first = post.call_args.args[4]
        self.assertEqual(first["clave_idempotencia"], self.body["clave_idempotencia"])
        with patch.object(cliente, "data_http", return_value=self.receipt) as post:
            self.assertEqual(self.run_client(), 0)
            self.assertEqual(post.call_args.args[4], first)
        with patch.object(cliente, "data_http") as post:
            self.assertEqual(self.run_client(), 0)
            post.assert_not_called()
        self.assertEqual(json.loads(self.estado.read_text())["operaciones"]["analisis"]["respuesta"], self.receipt)
        state = json.loads(self.estado.read_text())
        state["operaciones"]["analisis"]["respuesta"] = {}
        self.estado.write_text(json.dumps(state))
        self.estado.chmod(0o600)
        with patch.object(cliente, "data_http") as post:
            with self.assertRaises(cliente.Corte):
                self.run_client()
            post.assert_not_called()


if __name__ == "__main__":
    unittest.main()
