import importlib.util
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch


FILE = Path(__file__).with_name("precondicion.py")
spec = importlib.util.spec_from_file_location("precondicion", FILE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class PrecondicionTest(unittest.TestCase):
    def test_sql_canonico_se_incrusta_sin_terminar_transaccion(self):
        for path in (module.ROL, module.IDENTIDAD):
            body, digest = module.migracion_sin_transaccion(path)
            self.assertEqual(len(digest), 64)
            self.assertNotIn("\nCOMMIT;", body)
            self.assertNotIn("\nBEGIN;", body)
            self.assertTrue(body.strip())

    def test_rechaza_delimitador_alterado(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "alterada.up.sql"
            path.write_text("BEGIN;\nSELECT 1;\nCOMMIT;\nCOMMIT;\n")
            with self.assertRaises(ValueError):
                module.migracion_sin_transaccion(path)

    def test_inventario_contiene_grafico_y_acl_public(self):
        self.assertIn("pg_catalog.pg_auth_members", module.INVENTARIO)
        self.assertIn("pg_catalog.aclexplode", module.INVENTARIO)
        self.assertIn("FROM vec_pre_membresias EXCEPT ALL", module.FINALIZAR)
        self.assertIn("has_function_privilege", module.FINALIZAR)

    def test_actas_fuera_de_git_y_sin_enlaces(self):
        with self.assertRaises(ValueError):
            module.exigir_ruta_privada(FILE.with_name("informe.json"))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            link = root / "enlace"
            link.symlink_to(root, target_is_directory=True)
            with self.assertRaises(ValueError):
                module.exigir_ruta_privada(link / "informe.json")
            module.exigir_ruta_privada(root / "informe.json")

    def test_lectura_posterior_fallida_con_comit_acusado_es_indeterminada(self):
        inventory = {"membresias": [], "tipos_public": [],
                     "selector_existente": False, "fachada_existente": False,
                     "acl_public": [], "base": "sintetica"}
        digest = hashlib.sha256(json.dumps(
            inventory, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        _, huellas = module.migraciones_precondicion()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            report = root / "resultado.json"
            proof = root / "restauracion.json"
            rollback = root / "ensayo.json"
            proof.write_text(json.dumps({
                "verified": True, "source_database": "sintetica",
                "source_inventory_sha256": digest, "restore_exit_zero": True,
                "check_validated": True, "backup_sha256": "a" * 64,
                "restore_manifest_sha256": "b" * 64,
            }))
            rollback.write_text(json.dumps({
                "resultado": "ensayo revertido", "inventario_sha256": digest,
                "migraciones_sha256": huellas,
            }))
            argv = ["precondicion.py", "--mode", "commit", "--database", "sintetica",
                    "--report", str(report), "--expected-inventory-sha256", digest,
                    "--restoration-evidence", str(proof), "--rollback-report", str(rollback)]
            with patch.object(sys, "argv", argv), patch.object(
                module, "ejecutar", side_effect=[json.dumps(inventory),
                  "VERIFICACION_PRECIERRE_OK\nCOMMIT_CONFIRMADO",
                  RuntimeError("lectura posterior fallida")]
            ) as fake:
                with self.assertRaises(module.EstadoIndeterminado):
                    module.main()
            self.assertEqual(fake.call_count, 3)
            self.assertIn("VERIFICACION_PRECIERRE_OK", fake.call_args_list[1].args[1])
            self.assertLess(fake.call_args_list[1].args[1].index("VERIFICACION_PRECIERRE_OK"),
                            fake.call_args_list[1].args[1].index("COMMIT;"))
            self.assertEqual(json.loads(report.read_text())["resultado"], "indeterminado")


if __name__ == "__main__":
    unittest.main()
