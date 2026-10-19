import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ESPEC = importlib.util.spec_from_file_location("preparar_rrhh_adicional_copia", Path(__file__).with_name("preparar_rrhh_adicional_copia.py"))
MODULO = importlib.util.module_from_spec(ESPEC)
ESPEC.loader.exec_module(MODULO)


def openssl(*args):
    subprocess.run(["openssl", *map(str, args)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


class SegundoRRHHTest(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.material = Path(self.temporal.name) / "material"
        self.material.mkdir(mode=0o700)
        for nombre in ("ca", "mtls", "identidad"):
            (self.material / nombre).mkdir(parents=True, mode=0o700)
        ca = self.material / "ca"
        openssl("req", "-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256",
                "-nodes", "-days", "90", "-subj", "/CN=VEC CA Desarrollo", "-keyout", ca / "ca.key", "-out", ca / "ca.crt")
        self._certificado("cliente", "desarrollo:original", "identidad")
        huella = MODULO.huella(self.material / "mtls/cliente.crt")
        (self.material / "identidad/identidad.json").write_text(json.dumps({
            "version": 1, "autoridad": "no_autoritativo", "subject": "desarrollo:original",
            "display_name": "Técnica original", "roles": ["tecnico_rrhh"], "certificate_sha256": huella}))
        (self.material / "manifiesto.json").write_text(json.dumps({
            "perfil": "desarrollo", "autoridad": "no_autoritativo", "migrable_a_produccion": False,
            "huella_ca_sha256": MODULO.huella(ca / "ca.crt"), "huella_cliente_sha256": huella}))
        for ruta in self.material.rglob("*"):
            if ruta.is_file():
                ruta.chmod(0o600)

    def _certificado(self, nombre, sujeto, identidad):
        mtls = self.material / "mtls"
        openssl("req", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes",
                "-subj", f"/CN={sujeto}", "-keyout", mtls / f"{nombre}.key", "-out", mtls / f"{nombre}.csr")
        openssl("x509", "-req", "-days", "90", "-in", mtls / f"{nombre}.csr",
                "-CA", self.material / "ca/ca.crt", "-CAkey", self.material / "ca/ca.key",
                "-set_serial", "0x123456789", "-out", mtls / f"{nombre}.crt")
        (mtls / f"{nombre}.csr").unlink()

    def test_genera_segunda_hoja_misma_ca_y_referencias_distintas(self):
        antes = {p.name: p.read_bytes() for p in (self.material / "ca").iterdir()}
        original = (self.material / "identidad/identidad.json").read_bytes()
        dato = MODULO.preparar(self.material, "desarrollo:segunda-rrhh", "Elena Torres Ruiz")
        self.assertEqual(dato["autoridad"], "no_autoritativo")
        self.assertNotEqual(dato["cuenta_ref"], MODULO.referencia("cta_", "desarrollo:original\0" + MODULO.huella(self.material / "mtls/cliente.crt") + "\0cuenta"))
        self.assertNotEqual(dato["persona_ref"], MODULO.referencia("per_", "desarrollo:original\0" + MODULO.huella(self.material / "mtls/cliente.crt") + "\0persona"))
        self.assertEqual({p.name: p.read_bytes() for p in (self.material / "ca").iterdir()}, antes)
        self.assertEqual((self.material / "identidad/identidad.json").read_bytes(), original)
        openssl("verify", "-purpose", "sslclient", "-CAfile", self.material / "ca/ca.crt", self.material / dato["certificado"])
        for nombre in MODULO.ARCHIVOS:
            ruta = self.material / nombre
            self.assertTrue(ruta.is_file())
            self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
        clave_segunda = self.material / "mtls/tecnico-rrhh-adicional.key"
        MODULO.cotejar_clave(self.material / dato["certificado"], clave_segunda)
        clave_segunda.write_bytes((self.material / "mtls/cliente.key").read_bytes())
        with self.assertRaisesRegex(ValueError, "no corresponde"):
            MODULO.cotejar_clave(self.material / dato["certificado"], clave_segunda)
        with self.assertRaisesRegex(ValueError, "ya existe|no se sobrescribe"):
            MODULO.preparar(self.material, "desarrollo:segunda-rrhh", "Elena Torres Ruiz")

    def test_rechaza_fifo_y_enlace_multiple_antes_de_leer(self):
        fifo = self.material / "entrada"
        os.mkfifo(fifo, 0o600)
        with self.assertRaisesRegex(ValueError, "propia y privada"):
            MODULO.privado(fifo)
        original = self.material / "ca/ca.crt"
        os.link(original, self.material / "copia-ca.crt")
        with self.assertRaisesRegex(ValueError, "propia y privada"):
            MODULO.privado(original)

    def test_rechaza_sujeto_original_y_ca_incoherente(self):
        with self.assertRaises(ValueError):
            MODULO.preparar(self.material, "desarrollo:original", "Elena Torres Ruiz")
        dato = json.loads((self.material / "manifiesto.json").read_text())
        dato["huella_ca_sha256"] = "0" * 64
        (self.material / "manifiesto.json").write_text(json.dumps(dato))
        with self.assertRaisesRegex(ValueError, "no coinciden"):
            MODULO.preparar(self.material, "desarrollo:segunda-rrhh", "Elena Torres Ruiz")
        self.assertFalse(any((self.material / nombre).exists() for nombre in MODULO.ARCHIVOS))


if __name__ == "__main__":
    unittest.main()
