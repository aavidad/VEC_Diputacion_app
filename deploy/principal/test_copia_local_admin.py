"""Focales del límite de copia y de los acuses ADMIN; no usan Docker."""

import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import subprocess

import copia_local_admin as admin


class CopiaFalsa:
    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / "material"
        self.nombre = "vec-copia-l-prueba"
        self.sql_llamadas = []
        self.estado = {"nombre": self.nombre, "raiz": str(raiz), "revision": "abc",
                       "preparada": False, "puerto_pg": 55441, "puerto_app": 18441,
                       "origen": str(raiz / "origen")}

    def leer(self):
        return self.estado

    def sql_bytes(self, contenido, log):
        self.sql_llamadas.append((contenido, log))


class CopiaLocalAdminTest(unittest.TestCase):
    def test_kit_antiguo_para_antes_de_sql_y_sin_carpeta_admin(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            base = Path(temporal)
            base.chmod(0o700)
            copia = CopiaFalsa(base / "copia")
            copia.raiz.mkdir(mode=0o700)
            copia.material.mkdir(mode=0o700)
            ca = copia.material / "ca"
            ca.mkdir(mode=0o700)
            (ca / "ca.crt").write_text("ca ficticia")
            (ca / "ca.crt").chmod(0o600)
            kit = base / "kit"
            kit.mkdir(mode=0o700)
            with self.assertRaisesRegex(FileNotFoundError, "contrato-copia-local-admin"):
                admin.preparar(copia, kit)
            self.assertEqual(copia.sql_llamadas, [])
            self.assertFalse((copia.raiz / "admin").exists())

    def test_admin_ya_preparada_no_inicia_kit(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            copia = CopiaFalsa(Path(temporal))
            copia.estado["preparada"] = True
            copia.estado["admin_preparada"] = True
            with self.assertRaisesRegex(ValueError, "ya preparada para ADMIN"):
                admin.preparar(copia, Path(temporal) / "kit")
            self.assertEqual(copia.sql_llamadas, [])

    def test_instancia_ct_preparada_puede_iniciar_admin(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            copia = CopiaFalsa(Path(temporal))
            copia.estado["preparada"] = True
            with patch.object(admin, "_privado"), patch.object(admin, "_comprobar_kit", side_effect=ValueError("kit pendiente")):
                with self.assertRaisesRegex(ValueError, "kit pendiente"):
                    admin.preparar(copia, Path(temporal) / "kit")
            self.assertEqual(copia.sql_llamadas, [])

    def test_h10_exige_acuse_auditado_y_no_finge_exito(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            base = Path(temporal)
            run = base / "run"
            (run / "dba").mkdir(parents=True)
            (run / "fuentes").mkdir()
            (run / "dba/fuentes-password.sql").write_text("SELECT 1;")
            (run / "dba/fuentes-password.sql").chmod(0o600)
            (run / "fuentes/acuse-primero.json").write_text(json.dumps(
                {"estado": "permitido", "recibo": {"recibo_ref": "sin auditoría"}}))
            (run / "fuentes/acuse-primero.json").chmod(0o600)
            copia = CopiaFalsa(base)
            with patch.object(admin, "_ejecutar") as ejecutar:
                with self.assertRaisesRegex(ValueError, "acuse ADMIN permitido y auditado"):
                    admin._fase_h10(copia, base / "kit", run, "fuentes", "F0-28-local")
            self.assertEqual(ejecutar.call_count, 4)
            self.assertEqual(len(copia.sql_llamadas), 1)

    def test_commit_incierto_no_reprepara_plan(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            base = Path(temporal)
            run = base / "run"
            (run / "dba").mkdir(parents=True)
            (run / "dba/fuentes-password.sql").write_text("SELECT 1;")
            (run / "dba/fuentes-password.sql").chmod(0o600)
            copia = CopiaFalsa(base)
            with patch.object(admin, "_ejecutar", side_effect=[None, None, None, ValueError("fallo")]) as ejecutar:
                with self.assertRaisesRegex(ValueError, "recuperar_fuentes"):
                    admin._fase_h10(copia, base / "kit", run, "fuentes", "F0-28-local")
            self.assertEqual(ejecutar.call_count, 4)

    def test_fallo_del_kit_guarda_salida_privada_sin_revelarla(self):
        with tempfile.TemporaryDirectory(prefix="vec-admin-") as temporal:
            kit = Path(temporal)
            kit.chmod(0o700)
            resultado = subprocess.CompletedProcess([], 2, "", "detalle privado de prueba")
            with patch.object(admin.subprocess, "run", return_value=resultado):
                with self.assertRaisesRegex(ValueError, "consulte la carpeta privada") as error:
                    admin._ejecutar(kit, "preparar_copia.py")
            self.assertNotIn("detalle privado", str(error.exception))
            log = kit / "errores-admin.log"
            self.assertIn("detalle privado de prueba", log.read_text())
            self.assertEqual(log.stat().st_mode & 0o777, 0o600)


if __name__ == "__main__":
    unittest.main()
