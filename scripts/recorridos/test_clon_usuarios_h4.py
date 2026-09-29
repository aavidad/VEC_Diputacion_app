import importlib.util
import base64
import hashlib
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest

spec = importlib.util.spec_from_file_location("clon_usuarios_h4", Path(__file__).with_name("clon_usuarios_h4.py"))
h4 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(h4)


class H4Tests(unittest.TestCase):
    def identity_fixture(self):
        der = b"synthetic-certificate-der"
        fingerprint = hashlib.sha256(der).hexdigest()
        certificate = b"-----BEGIN CERTIFICATE-----\n" + base64.b64encode(der) + b"\n-----END CERTIFICATE-----\n"
        configuration = {"cuentas": [{"sujeto": "synthetic:subject", "certificado_sha256": fingerprint}]}
        identity = dict(version=1, autoridad="synthetic", certificate_sha256=fingerprint,
                        subject="synthetic:subject", display_name="Synthetic", roles=["tecnico_rrhh"])
        return configuration, identity, certificate

    def test_certificate_and_subject_pin_reject_mismatches(self):
        configuration, identity, certificate = self.identity_fixture()
        h4.validate_identity_input(configuration, identity, certificate, "tecnico_rrhh")
        for field, value in [("certificate_sha256", "0" * 64), ("subject", "other:subject"),
                             ("roles", ["intervencion"])]:
            changed = dict(identity, **{field: value})
            with self.subTest(field=field), self.assertRaises(h4.H4Error):
                h4.validate_identity_input(configuration, changed, certificate, "tecnico_rrhh")
        configuration["cuentas"][0]["certificado_sha256"] = "0" * 64
        with self.assertRaises(h4.H4Error):
            h4.validate_identity_input(configuration, identity, certificate, "tecnico_rrhh")
        with self.assertRaises(h4.H4Error):
            h4.validate_identity_input(configuration, identity, certificate + certificate, "tecnico_rrhh")

    def test_closed_identity_json_rejects_repeated_fields(self):
        with self.assertRaises(h4.H4Error):
            h4.closed_json('{"subject":"first","subject":"other"}')

    def test_every_users_dsn_uses_the_runtime_destination_guard(self):
        runtime_path = os.environ.get("H4_RUNTIME_TEST_MODULE", str(Path(__file__).with_name("clon_runtime.py")))
        if not Path(runtime_path).is_file():
            self.skipTest("The independently owned runtime dependency is required for these negative checks.")
        runtime = h4.load(runtime_path, "h4_negative_runtime")
        fields = ["dsn_contexto", "dsn_fuente_autorizacion", "dsn_motivos", "dsn_registro_autorizacion",
                  "dsn_registro_identidad", "dsn_revalidacion_identidad", "dsn_usuarios", "dsn_usuarios_frontera"]
        with tempfile.TemporaryDirectory() as folder:
            state = Path(folder);material = state / "material"
            (material / "identidad").mkdir(parents=True);(material / "mtls").mkdir()
            configuration, identity, certificate = self.identity_fixture()
            valid = "postgres://fixture@127.0.0.1:55531/postgres?sslmode=verify-full"
            configs = {}
            for surface, cert, file, role in [("interna", "cliente", "identidad", "tecnico_rrhh"),
                                               ("externa", "intervencion", "intervencion", "intervencion")]:
                configs[surface] = dict(configuration, **dict.fromkeys(fields, valid))
                (material / "identidad" / (file + ".json")).write_text(json.dumps(dict(identity, roles=[role])))
                (material / "mtls" / (cert + ".crt")).write_bytes(certificate)
            def write():
                for surface, config in configs.items():
                    (material / "identidad" / ("usuarios-preferencias-" + surface + ".json")).write_text(json.dumps(config))
            users = SimpleNamespace(private=lambda path: path.read_bytes())
            write();h4.validate_users_inputs(material, state, 55531, users, runtime)
            for surface in configs:
                for field in fields:
                    for bad in [valid.replace("127.0.0.1", "remote.invalid"), valid.replace("55531", "55441"), valid.replace("verify-full", "disable"), valid + "&sslrootcert=/outside-clone/ca.crt"]:
                        configs[surface][field] = bad;write()
                        with self.subTest(surface=surface, field=field, bad=bad), self.assertRaises(runtime.RuntimeErrorLocal):
                            h4.validate_users_inputs(material, state, 55531, users, runtime)
                    configs[surface][field] = valid
            write();h4.validate_users_inputs(material, state, 55531, users, runtime)

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
