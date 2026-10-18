"""Prueba focal del material ficticio del preparador de Mi Bolsa."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("preparar_candidato_desarrollo.sh")
CANDIDATOS = ("can_" + "a" * 24, "can_" + "b" * 24)
BOLSA = "bolsa:prueba:administrativo"


class PrepararCandidatoDesarrolloTest(unittest.TestCase):
    def setUp(self):
        scratch = os.environ.get("VEC_TEST_SCRATCH") or tempfile.gettempdir()
        self.temporal = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.temporal.cleanup)
        self.raiz = Path(self.temporal.name)
        self.bin = self.raiz / "bin"
        self.bin.mkdir()
        motor = self.bin / "docker"
        motor.write_text(
            "#!/usr/bin/env python3\n"
            "import json, os, sys\n"
            "from pathlib import Path\n"
            "sql = sys.stdin.read()\n"
            "with Path(os.environ['VEC_TEST_MOTOR_LOG']).open('a') as f:\n"
            "    f.write(json.dumps({'args': sys.argv[1:], 'sql': sql}) + '\\n')\n"
            "if 'SELECT vc.candidato_ref, vc.participacion_ref, c.bolsa_ref' in sql:\n"
            "    seleccion = next((a.split('=', 1)[1] for a in sys.argv[1:] "
            "if a.startswith('seleccion=')), '')\n"
            "    candidatos = os.environ['VEC_TEST_CANDIDATOS'].split(',')\n"
            "    if seleccion in candidatos or not seleccion:\n"
            "        ref = seleccion or candidatos[0]\n"
            "        print(ref + '|participacion-' + ref + '|' + os.environ['VEC_TEST_BOLSA'])\n",
            encoding="utf-8",
        )
        motor.chmod(0o700)
        self.log = self.raiz / "motor.jsonl"
        self.env = {
            "PATH": f"{self.bin}:/usr/bin:/bin",
            "HOME": str(self.raiz),
            "VEC_TEST_MOTOR_LOG": str(self.log),
            "VEC_TEST_CANDIDATOS": ",".join(CANDIDATOS),
            "VEC_TEST_BOLSA": BOLSA,
        }

    def material(self, nombre):
        material = self.raiz / nombre
        ca = material / "ca"
        ca.mkdir(parents=True, mode=0o700)
        material.chmod(0o700)
        subprocess.run(
            ["openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt",
             "ec_paramgen_curve:P-256", "-nodes", "-days", "90", "-sha256",
             "-subj", "/CN=CA ficticia", "-addext", "basicConstraints=critical,CA:TRUE",
             "-addext", "keyUsage=critical,keyCertSign,cRLSign", "-keyout", str(ca / "ca.key"),
             "-out", str(ca / "ca.crt")],
            check=True, capture_output=True, env=self.env,
        )
        (ca / "ca.key").chmod(0o600)
        (ca / "ca.crt").chmod(0o600)
        return material

    def ejecutar(self, material, *args):
        env = dict(self.env, VEC_DEVELOPMENT_MATERIAL_DIR=str(material))
        return subprocess.run(
            ["bash", str(SCRIPT), *args], capture_output=True, text=True,
            env=env, timeout=30,
        )

    def llamadas(self):
        if not self.log.exists():
            return []
        return [json.loads(linea) for linea in self.log.read_text().splitlines()]

    def test_dos_identidades_seleccion_y_repeticion(self):
        primero = self.material("material-antonio")
        segundo = self.material("material-otra")
        for material, slug, nombre, ref in (
            (primero, "antonio", "Antonio Reyes Álvarez", CANDIDATOS[0]),
            (segundo, "otra", "Otra persona ficticia", CANDIDATOS[1]),
        ):
            resultado = self.ejecutar(material, "--motor", "docker", "--identidad", slug,
                                     "--nombre", nombre, "--candidato-ref", ref)
            self.assertEqual(resultado.returncode, 0, resultado.stderr)
            self.assertIn("docker restart", resultado.stdout)
            identidad = json.loads((material / "identidad/candidato.json").read_text())
            manifiesto = json.loads((material / "identidad/bolsa-candidato.json").read_text())
            self.assertEqual(identidad["display_name"], nombre)
            self.assertEqual(manifiesto["candidato_ref"], ref)
            self.assertEqual(manifiesto["certificado"], "mtls/candidato.crt")
            self.assertEqual(manifiesto["identidad"], "identidad/candidato.json")
            consulta = next(c for c in self.llamadas()[::-1] if "SELECT vc.candidato_ref" in c["sql"])
            self.assertIn("seleccion=" + ref, consulta["args"])
            self.assertIn("vc.candidato_ref = :'seleccion'", consulta["sql"])
            cert = material / "mtls/candidato.crt"
            texto = subprocess.check_output(
                ["openssl", "x509", "-in", str(cert), "-noout", "-subject", "-ext",
                 "subjectAltName"], text=True, env=self.env,
            )
            self.assertIn("candidato-bolsa-desarrollo-" + slug, texto)
            self.assertIn("urn:vec:desarrollo:candidato-bolsa:" + slug, texto)

        identidades = [json.loads((m / "identidad/candidato.json").read_text()) for m in (primero, segundo)]
        cuentas = [json.loads((m / "identidad/bolsa-candidato.json").read_text()) for m in (primero, segundo)]
        self.assertNotEqual(identidades[0]["subject"], identidades[1]["subject"])
        self.assertNotEqual(cuentas[0]["cuenta_ref"], cuentas[1]["cuenta_ref"])

        antes = len(self.llamadas())
        repeticion = self.ejecutar(primero, "--motor", "docker", "--identidad", "antonio",
                                  "--nombre", "Antonio Reyes Álvarez", "--candidato-ref", CANDIDATOS[0])
        self.assertEqual(repeticion.returncode, 0, repeticion.stderr)
        self.assertEqual(len(self.llamadas()), antes + 3)

        antes = len(self.llamadas())
        cambio = self.ejecutar(primero, "--motor", "docker", "--identidad", "antonio",
                               "--nombre", "Antonio Reyes Álvarez", "--candidato-ref", CANDIDATOS[1])
        self.assertNotEqual(cambio.returncode, 0)
        self.assertIn("otro candidato", cambio.stderr)
        self.assertEqual(len(self.llamadas()), antes)

        huella = json.loads((primero / "identidad/candidato.json").read_text())["certificate_sha256"]
        intencion = primero / "identidad/.bolsa-candidato-intencion.json"
        intencion.write_text(json.dumps({"version": 1, "certificate_sha256": huella,
                                         "candidato_ref": CANDIDATOS[0],
                                         "participacion_ref": "participacion-" + CANDIDATOS[0],
                                         "bolsa_ref": BOLSA}))
        intencion.chmod(0o600)
        intento = self.ejecutar(primero, "--motor", "docker", "--identidad", "antonio",
                                "--nombre", "Antonio Reyes Álvarez", "--candidato-ref", CANDIDATOS[1])
        self.assertNotEqual(intento.returncode, 0)
        self.assertIn("intención privada de otro candidato", intento.stderr)
        self.assertEqual(len(self.llamadas()), antes)
        intencion.unlink()

        cert = primero / "mtls/candidato.crt"
        clave = primero / "mtls/candidato.key"
        ca = primero / "ca"
        csr = self.raiz / "largo.csr"
        subprocess.run(["openssl", "req", "-new", "-key", str(clave), "-subj",
                        "/CN=candidato-bolsa-desarrollo-antonio/O=VEC Desarrollo/OU=NO AUTORITATIVO",
                        "-out", str(csr)], check=True, capture_output=True, env=self.env)
        extension = self.raiz / "largo.ext"
        extension.write_text("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\n"
                             "extendedKeyUsage=clientAuth\n"
                             "subjectAltName=URI:urn:vec:desarrollo:candidato-bolsa:antonio\n")
        for dias in (59, 60, 91, 397):
            with self.subTest(dias=dias):
                subprocess.run(["openssl", "x509", "-req", "-days", str(dias), "-in", str(csr),
                                "-CA", str(ca / "ca.crt"), "-CAkey", str(ca / "ca.key"),
                                "-set_serial", "0x9876", "-extfile", str(extension), "-out", str(cert)],
                               check=True, capture_output=True, env=self.env)
                subprocess.run(["openssl", "pkcs12", "-export", "-inkey", str(clave), "-in", str(cert),
                                "-certfile", str(ca / "ca.crt"), "-out", str(primero / "mtls/candidato.p12"),
                                "-passout", "file:" + str(primero / "mtls/candidato.p12.password")],
                               check=True, capture_output=True, env=self.env)
                for nombre in ("candidato.json", "bolsa-candidato.json"):
                    (primero / "identidad" / nombre).unlink(missing_ok=True)
                antes = len(self.llamadas())
                resultado = self.ejecutar(primero, "--motor", "docker", "--identidad", "antonio",
                                          "--nombre", "Antonio Reyes Álvarez", "--candidato-ref", CANDIDATOS[0])
                if dias == 60:
                    self.assertEqual(resultado.returncode, 0, resultado.stderr)
                    self.assertEqual(len(self.llamadas()), antes + 3)
                else:
                    self.assertNotEqual(resultado.returncode, 0)
                    self.assertIn("fuera de 60 a 90 días", resultado.stderr)
                    self.assertEqual(len(self.llamadas()), antes)

    def test_entradas_invalidas_antes_de_efectos(self):
        inexistente = self.raiz / "no-creado"
        for args, mensaje in (
            (("--motor", "otro"), "motor inválido"),
            (("--identidad", "../otra"), "identidad inválida"),
            (("--nombre", "Nombre\nroto"), "nombre visible inválido"),
            (("--candidato-ref", ""), "referencia candidata inválida"),
        ):
            with self.subTest(args=args):
                resultado = self.ejecutar(inexistente, *args)
                self.assertNotEqual(resultado.returncode, 0)
                self.assertIn(mensaje, resultado.stderr)
        self.assertFalse(inexistente.exists())
        self.assertEqual(self.llamadas(), [])

    def test_sin_opciones_conserva_identidad_predeterminada(self):
        (self.bin / "podman").symlink_to(self.bin / "docker")
        material = self.material("material-predeterminado")
        resultado = self.ejecutar(material)
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        self.assertIn("podman restart", resultado.stdout)
        identidad = json.loads((material / "identidad/candidato.json").read_text())
        self.assertEqual(identidad["display_name"], "Persona candidata")
        texto = subprocess.check_output(
            ["openssl", "x509", "-in", str(material / "mtls/candidato.crt"), "-noout",
             "-subject", "-ext", "subjectAltName"], text=True, env=self.env,
        )
        self.assertIn("CN=candidato-bolsa-desarrollo", texto)
        self.assertIn("URI:urn:vec:desarrollo:candidato-bolsa", texto)


if __name__ == "__main__":
    unittest.main()
