import importlib.util
from pathlib import Path
import tempfile
import unittest


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


if __name__ == "__main__":
    unittest.main()
