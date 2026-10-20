"""Ensayo aislado de la CRL inicial de una CA sintética."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from recorrido_copia import certificados_base


class Copia:
    nombre = "vec-copia-f131-aislada"

    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / "material"
        self.sid = "1234567890123456789"

    def leer(self):
        return {"recorrido_completo": True, "material_modo": "nuevo", "sid_copia": self.sid}


class Contexto:
    def __init__(self, raiz, dias_ca=400):
        self.raiz = raiz
        self.copia = Copia(raiz)
        self.material = self.copia.material
        self.carpeta = raiz / "recorrido"
        for carpeta in (raiz, self.material, self.material / "ca", self.material / "identidad", self.carpeta):
            carpeta.mkdir(mode=0o700, exist_ok=True)
            carpeta.chmod(0o700)
        ca = self.material / "ca"
        subprocess.run(("/usr/bin/openssl", "genpkey", "-algorithm", "EC", "-pkeyopt",
                        "ec_paramgen_curve:P-256", "-out", str(ca / "ca.key")),
                       check=True, capture_output=True)
        subprocess.run(("/usr/bin/openssl", "req", "-new", "-x509", "-sha256",
                        "-days", str(dias_ca), "-key", str(ca / "ca.key"),
                        "-subj", "/CN=VEC F131 isolated CA", "-addext",
                        "basicConstraints=critical,CA:TRUE", "-addext",
                        "keyUsage=critical,keyCertSign,cRLSign", "-out", str(ca / "ca.crt")),
                       check=True, capture_output=True)
        (ca / "ca.key").chmod(0o600)
        (ca / "ca.crt").chmod(0o600)

    def consultar(self, sql):
        if sql != "SELECT system_identifier::text FROM pg_catalog.pg_control_system();":
            raise AssertionError("consulta fuera del SID propio")
        return self.copia.sid

    def ejecutar(self, argv, log, cwd=None, env=None):
        if argv[0] != "/usr/bin/openssl":
            raise AssertionError("solo OpenSSL local")
        return subprocess.run(argv, cwd=cwd, env=env, capture_output=True, check=False)

    @staticmethod
    def ahora():
        from datetime import datetime, timezone
        return datetime.now(timezone.utc)

    @staticmethod
    def guardar(ruta, valor):
        raw = valor if isinstance(valor, bytes) else json.dumps(valor).encode() + b"\n"
        fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, "wb") as salida:
            salida.write(raw)

    @staticmethod
    def leer(ruta):
        return json.loads(Path(ruta).read_bytes())


class CertificadosBaseTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = Contexto(Path(self.tmp.name) / "copia")

    def test_crl_inicial_firmada_y_repeticion_sin_reset(self):
        primera = certificados_base.preparar(self.c)
        indice = self.c.material / "ca/index.txt"
        crl = self.c.material / "identidad/clientes.crl"
        self.assertEqual(indice.read_bytes(), b"")
        self.assertEqual(primera["index_sha256"], certificados_base._sha(indice))
        self.assertEqual(primera["crl_sha256"], certificados_base._sha(crl))
        self.assertEqual(certificados_base.preparar(self.c), primera)
        self.assertEqual(indice.read_bytes(), b"")
        indice.write_bytes(b"V\t271231235959Z\t\t01\t/CN=fixture\n")
        with self.assertRaisesRegex(ValueError, "manifiesto de revocación divergente"):
            certificados_base.preparar(self.c)
        self.assertIn(b"fixture", indice.read_bytes())

    def test_productor_nativo_acepta_base_sin_crear_firmantes(self):
        repo = Path(__file__).resolve().parents[2]
        if not (repo / "scripts/operacion/firma_ct_tres_pasos/certificados.py").is_file():
            repo = Path(os.environ.get("VEC_F131_REPO", "/no-existe"))
        productor = repo / "scripts/operacion/firma_ct_tres_pasos/certificados.py"
        if not productor.is_file():
            self.skipTest("fuente del productor nativo no disponible")
        certificados_base.preparar(self.c)
        salida = self.c.raiz / "certificados"
        salida.mkdir(mode=0o700)
        resultado = subprocess.run(("/usr/bin/python3", str(productor), "--comprobar",
                                    "--material", str(self.c.material), "--salida", str(salida),
                                    "--cert-dias", "397", "--crl-dias", "90"),
                                   capture_output=True, text=True, check=False)
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        self.assertEqual(json.loads(resultado.stdout)["estado"], "comprobado")
        self.assertEqual(list(salida.iterdir()), [])

    def test_productor_emite_cuarto_tecnico_y_conserva_material_en_replay(self):
        repo = Path(__file__).resolve().parents[2]
        productor = repo / "scripts/operacion/firma_ct_tres_pasos/certificados.py"
        certificados_base.preparar(self.c)
        salida = self.c.raiz / "cuatro-certificados"
        salida.mkdir(mode=0o700)
        orden = ("/usr/bin/python3", str(productor), "--aplicar", "PREPARA-CERTIFICADOS-CT",
                 "--material", str(self.c.material), "--salida", str(salida),
                 "--cert-dias", "397", "--crl-dias", "90", "--incluir-tecnico")
        resultado = subprocess.run(orden, capture_output=True, text=True, check=False)
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        self.assertEqual(json.loads(resultado.stdout)["personas"], 4)
        manifiesto = json.loads((salida / "certificados.json").read_text())
        self.assertEqual(len({p["certificado_sha256"] for p in manifiesto["personas"]}), 4)
        self.assertEqual({p["usuario_web"] for p in manifiesto["personas"]},
                         {"rrhh_tecnico", "rrhh_jefatura", "rrhh_diputacion", "rrhh_secretaria"})
        anteriores = {str(p.relative_to(salida)): p.read_bytes() for p in salida.rglob("*") if p.is_file()}
        resultado = subprocess.run(orden, capture_output=True, text=True, check=False)
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        self.assertEqual(json.loads(resultado.stdout)["estado"], "replay")
        self.assertEqual(anteriores,
                         {str(p.relative_to(salida)): p.read_bytes() for p in salida.rglob("*") if p.is_file()})

    def test_estado_parcial_y_sid_ajeno_rechazados(self):
        self.c.consultar = lambda sql: "9999999999999999999"
        with self.assertRaisesRegex(ValueError, "SID propio"):
            certificados_base.preparar(self.c)
        self.c.consultar = lambda sql: self.c.copia.sid
        indice = self.c.material / "ca/index.txt"
        indice.write_bytes(b"")
        indice.chmod(0o600)
        with self.assertRaisesRegex(ValueError, "revocación parcial"):
            certificados_base.preparar(self.c)

    def test_ca_corta_no_crea_indice(self):
        with tempfile.TemporaryDirectory() as tmp:
            corta = Contexto(Path(tmp) / "copia", dias_ca=90)
            with self.assertRaisesRegex(ValueError, "OpenSSL|CA no cubre"):
                certificados_base.preparar(corta)
            self.assertFalse((corta.material / "ca/index.txt").exists())


if __name__ == "__main__":
    unittest.main()
