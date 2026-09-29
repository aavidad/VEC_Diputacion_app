import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("clon_usuarios_h4", Path(__file__).with_name("clon_usuarios_h4.py"))
h4 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(h4)


class H4Tests(unittest.TestCase):
    def contract_source(self):
        return "\n".join(name + " = " + repr(("purpose", "resource", tuple((name + str(i), ["version"]) for i in range(count))))
                         for name, count in [("PREFERENCIAS", 2), ("CORREOS", 6), ("IMAGEN", 2)])

    def test_private_kit_is_read_as_data_and_never_executed(self):
        source = self.contract_source() + '\nraise RuntimeError("must not run")\n'
        contract = h4.material_contract(source)
        self.assertEqual(len(contract), 10)
        self.assertEqual(contract[2]["accion"], "CORREOS0")

    def test_contract_rejects_duplicate_missing_and_executable_data(self):
        for source in [self.contract_source() + "\nIMAGEN = ('purpose','resource',())",
                       self.contract_source().replace("IMAGEN =", "OTHER ="),
                       self.contract_source().replace("CORREOS =", "CORREOS = execute() #")]:
            with self.subTest(source=source), self.assertRaises((h4.H4Error, ValueError)):
                h4.material_contract(source)

    def test_identity_plan_ignores_ephemeral_installation_counts(self):
        result = dict(accounts=["a", "b"], persons=["p", "q"], profiles=["x", "y"],
                      install_at="historic", references_preserved=True, technical_logins_created=16)
        before = h4.identity_preimage(result)
        result["technical_logins_created"] = 0
        self.assertEqual(h4.identity_preimage(result), before)
        result["persons"] = ["changed", "q"]
        self.assertNotEqual(h4.identity_preimage(result), before)

    def test_wrong_destinations_fail_before_loading_private_or_database_inputs(self):
        for container, port, engine in [("principal", 55531, "docker"), ("vec-codexm-recorridos-20260930", 55441, "docker"),
                                        ("vec-codexm-recorridos-20260930", 55531, "podman")]:
            with self.subTest(container=container, port=port), self.assertRaises(h4.H4Error):
                h4.preflight("/missing", container, "/missing", "/missing/material", port, engine)


if __name__ == "__main__":
    unittest.main()
