"""Offline contract and filesystem adversarial tests, using synthetic fixtures."""
import copy
from datetime import datetime, timezone
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("accredited", Path(__file__).with_name("clon_fuente_acreditada.py"))
module = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(module)
NOW = datetime(2026, 10, 1, 1, tzinfo=timezone.utc)


class AccreditationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.source_root = self.base / "source"
        self.source_root.mkdir(mode=0o700)
        refs = {key: module.reference(prefix, key) for key, prefix in
                (("cuenta", "cta_"), ("persona", "per_"), ("perfil", "prf_"), ("contexto", "vca_"),
                 ("vinculo", "vin_"), ("candidato", "can_"), ("procedencia", "prc_"))}
        self.source = {
            "version": 1, "esquema": "vec.fuente.sintetica.externa.propuesta.v1", "estado": "propuesta",
            "datos_sinteticos": True, "acreditacion": None, "autoridad_maestra": module.AUTHORITY,
            "responsable": module.RESPONSIBLE, "procedencia_ref": refs["procedencia"], "procedencia_version": 1,
            "personas": [{"persona_ref": refs["persona"], "nombre_visible": "Persona sintética de prueba"}],
            "cuentas": [{"cuenta_ref": refs["cuenta"], "persona_ref": refs["persona"],
                         "sujeto": "desarrollo:clon-recorridos:candidato", "poblaciones": ["candidato", "usuarios"]}],
            "candidatoBolsa": {key + "_ref": refs[key] for key in ("cuenta", "persona", "perfil", "candidato")},
            "vigente_desde_propuesto": "2026-10-01T00:00:00.000000Z",
            "vigente_hasta_propuesto": "2026-10-15T00:00:00.000000Z"}
        self.source_bytes = module.encoded(self.source)
        self.ack = {"version": 1, "kind": "acuse_fuente_sintetica_v1", "fuente_sha256": module.sha(self.source_bytes),
                    "autoridad_maestra": module.AUTHORITY, "responsable": module.RESPONSIBLE,
                    "acreditado_en": module.ACK_TIME, "mensaje": "Acuse sintético del fixture.\n",
                    "mensaje_sha256": module.sha(b"Acuse sint\xc3\xa9tico del fixture.\n")}
        for name, value in (("SOURCE_SHA256", module.sha(self.source_bytes)),
                            ("ACK_MESSAGE_SHA256", self.ack["mensaje_sha256"])):
            mocked = patch.object(module, name, value)
            mocked.start()
            self.addCleanup(mocked.stop)
        self.ack_path = self.base / "ack.json"
        self.put(self.ack_path, self.ack)
        self.put(self.source_root / "fuente-sintetica-v1.json", self.source)
        self.metadata = module.validate_accreditation(self.source_bytes, self.ack, NOW)
        self.snapshots = {}
        for population in module.POPULATIONS:
            component = lambda key: {"referencia": refs[key], "version": 1, "procedencia_ref": refs["procedencia"],
                    "procedencia_version": 1, "procedencia_huella_sha256": module.SOURCE_SHA256,
                    "procedencia_autoridad": "propuesta", "estado": "activo",
                    "vigente_desde": self.metadata["vigente_desde"], "vigente_hasta": self.metadata["vigente_hasta"]}
            snapshot = {"provision_ref": module.reference("pce_" if population == "candidato" else "pue_", population),
                        "poblacion": population, "estado": "activo",
                        **{name: component(name) for name in ("cuenta", "persona", "perfil", "contexto")},
                        "vinculo_candidato": None}
            if population == "candidato":
                snapshot["vinculo_candidato"] = component("vinculo") | {"candidato_ref": refs["candidato"]}
            self.snapshots[population] = snapshot
            self.put(self.source_root / (population + "-fuente.propuesta-v1.json"),
                     {"version": 1, "snapshot": snapshot, "preimagen": None})
        self.context = {"version": 1, "pgid": {"system_identifier": "fixture-pgid", "database_name": "postgres",
                         "database_oid": 5, "pg_container_id": "a" * 64, "pg_image": "fixture@sha256:" + "b" * 64,
                         "pg_image_id": "sha256:" + "c" * 64, "pg_volume": "fixture-owned"},
                        "journal_sha256": "d" * 64, "restore_receipt_sha256": "e" * 64}
        self.act = {"version": 1, "kind": "preimagenes_nominales_ro_v1", "source_sha256": module.SOURCE_SHA256,
                    "observado_en": "2026-10-01T00:30:00Z", **{k: v for k, v in self.context.items() if k != "version"},
                    "observaciones": {}}
        for population, snapshot in self.snapshots.items():
            self.act["observaciones"][population] = {
                "cuenta_ref": refs["cuenta"], "perfil_ref": refs["perfil"], "provision_ref": snapshot["provision_ref"],
                "preimagen": {name: "" if name.startswith("huella_") else 0 for name in module.CAS_FIELDS},
                "mediciones": {name: {"estado": "ausente", "consulta_sha256": "f" * 64,
                                       "valor": [] if name == "catalogos" else None} for name in module.MEASUREMENTS}}

    @staticmethod
    def put(path, value):
        data = value if isinstance(value, bytes) else module.encoded(value)
        descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(data)

    def test_missing_material_and_unmeasured_null_emit_no_snapshots(self):
        result, outputs = module.assemble(self.source_root, self.ack_path, now=NOW)
        self.assertEqual(outputs, {})
        self.assertEqual(result["estado"], "bloqueado")
        self.assertEqual(result["snapshots"], 0)
        self.assertIn("preimagen_cas_nominal_pendiente", result["blockers"])
        self.assertIn("material_externo_validado_pendiente", result["blockers"])
        self.assertFalse((self.base / "output").exists())

    def test_exact_ack_required_even_when_its_hash_is_self_consistent(self):
        ack = copy.deepcopy(self.ack)
        ack["mensaje"] = "Aprobación distinta.\n"
        ack["mensaje_sha256"] = module.sha(ack["mensaje"].encode())
        with self.assertRaisesRegex(module.AccreditationError, "ack_mismatch"):
            module.validate_accreditation(self.source_bytes, ack, NOW)
        for name, value in (("fuente_sha256", "0" * 64), ("acreditado_en", "2026-10-01T00:53:00Z")):
            ack = self.ack | {name: value}
            with self.assertRaisesRegex(module.AccreditationError, "ack_mismatch"):
                module.validate_accreditation(self.source_bytes, ack, NOW)

    def test_changed_source_and_duplicate_account_fail(self):
        with self.assertRaisesRegex(module.AccreditationError, "source_sha_mismatch"):
            module.validate_accreditation(self.source_bytes + b" ", self.ack, NOW)
        source = copy.deepcopy(self.source)
        source["cuentas"].append(source["cuentas"][0])
        data = module.encoded(source)
        with patch.object(module, "SOURCE_SHA256", module.sha(data)):
            with self.assertRaisesRegex(module.AccreditationError, "source_population_ambiguous"):
                module.validate_accreditation(data, self.ack, NOW)

    def test_validity_and_snapshot_reference_or_version_change_fail(self):
        with self.assertRaisesRegex(module.AccreditationError, "source_not_current"):
            module.validate_accreditation(self.source_bytes, self.ack, datetime(2026, 10, 15, tzinfo=timezone.utc))
        for key, value in (("referencia", module.reference("per_", "different")), ("version", 2)):
            proposal = {"version": 1, "snapshot": copy.deepcopy(self.snapshots["candidato"]), "preimagen": None}
            proposal["snapshot"]["persona"][key] = value
            with self.assertRaisesRegex(module.AccreditationError, "snapshot_reference_changed"):
                module.validate_proposal(proposal, "candidato", self.metadata)

    def test_null_or_missing_cas_and_nonmeasured_status_fail(self):
        for change in (lambda row: row.update(preimagen=None),
                       lambda row: row["mediciones"].pop("control_rol"),
                       lambda row: row["mediciones"]["contexto"].update(estado="no_medido")):
            act = copy.deepcopy(self.act)
            change(act["observaciones"]["candidato"])
            with self.assertRaises(module.AccreditationError):
                module.validate_observations(act, self.context, self.snapshots, NOW)

    def test_measurement_and_clone_bindings_fail_closed(self):
        mutations = (
            lambda a: a.update(journal_sha256="0" * 64),
            lambda a: a["pgid"].update(system_identifier="another-instance"),
            lambda a: a["observaciones"]["candidato"].update(cuenta_ref=module.reference("cta_", "other")),
            lambda a: a["observaciones"]["candidato"]["preimagen"].update(version_contexto=1, huella_contexto="1" * 64),
            lambda a: a["observaciones"]["usuarios"]["mediciones"]["cuenta"].update(estado="presente", valor={"cuenta_ref":"other"}),
        )
        for mutate in mutations:
            act = copy.deepcopy(self.act)
            mutate(act)
            with self.assertRaises(module.AccreditationError):
                module.validate_observations(act, self.context, self.snapshots, NOW)
        preimages = module.validate_observations(self.act, self.context, self.snapshots, NOW)
        self.assertEqual(preimages["candidato"]["version_contexto"], 0)

    def test_present_cas_must_match_measurement_and_role(self):
        for population in module.POPULATIONS:
            row = self.act["observaciones"][population]
            row["preimagen"].update(revision_control_rol=2, huella_control_rol="1" * 64)
            row["mediciones"]["control_rol"].update(estado="presente", valor={"revision":2,"huella_control":"1"*64})
            row["mediciones"]["rol"].update(estado="presente", valor={"version_rol_ref":module.ROLE_REF,
                                                                     "version":1,"huella_sha256":"2"*64})
        module.validate_observations(self.act, self.context, self.snapshots, NOW)
        self.act["observaciones"]["candidato"]["mediciones"]["control_rol"]["valor"]["revision"] = 3
        with self.assertRaisesRegex(module.AccreditationError, "cas_present_incoherent"):
            module.validate_observations(self.act, self.context, self.snapshots, NOW)

    def complete_material(self):
        root = self.base / "material"
        root.mkdir(mode=0o700)
        (root / "runtime-externo").mkdir(mode=0o700)
        names = ("ca/ca.crt", "tls/servidor.crt", "tls/servidor.key", "mtls/candidato.crt",
                 "identidad/candidato.json", "identidad/bolsa-candidato.json", "idempotencia/configuracion.json",
                 "manifiesto.json", "portal-proceso.json")
        hashes = {}
        for name in names:
            path = root / "runtime-externo" / name
            path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
            data = b"synthetic fixture material\n"
            if name == "identidad/candidato.json":
                data = module.encoded({"subject":self.metadata["sujeto"],"certificate_sha256":"1"*64})
            if name == "identidad/bolsa-candidato.json":
                data = module.encoded({name+"_ref":self.metadata["refs"][name] for name in
                                      ("cuenta","persona","perfil","candidato")} | {"sujeto":self.metadata["sujeto"]})
            self.put(path, data)
            hashes["runtime-externo/"+name] = {"sha256":module.sha(data),"bytes":len(data)}
        material = {key:self.metadata[key] for key in ("autoridad_maestra","responsable","acreditado_en","refs","sujeto",
                                                      "vigente_desde","vigente_hasta")}
        material.update(version=1,kind="material_externo_offline_v1",estado="preparado_offline",
                        fuente_sha256=module.SOURCE_SHA256,acuse_sha256=module.sha(module.encoded(self.ack)),
                        huella_ca_sha256="2"*64,huella_servidor_sha256="3"*64,certificate_sha256="1"*64,
                        archivos=hashes,exportador_ejecutado=False,provision_ejecutada=False,runtime_activado=False)
        material_path = self.base / "material-act.json"
        self.put(material_path, material)
        alias = {"version":1,"cuentas":[{"cuenta_ref":self.metadata["refs"]["cuenta"],"esquema":"vec.identidad.hmac-sha256.v1",
                    "dominio_ref":"idh_"+module.sha(b"vec.ct.alta.desarrollo.v1\0https://localhost/vec/desarrollo/identidad")[:32],
                    "clave_id":"vec.identidad.desarrollo.externo.g1","clave_version":1,
                    "cuenta_id_hmac":"4"*64,"sujeto_id_hmac":"5"*64}]}
        alias_path = self.base / "alias.json"
        self.put(alias_path, alias)
        receipt = {"version":1,"kind":"exportacion_seudonimos_externos_v1","fuente_sha256":module.SOURCE_SHA256,
                   "material_acta_sha256":module.sha(module.encoded(material)),
                   "exportador":"cmd/vec-server exportar-seudonimos-portal-externo","salida_sha256":module.sha(module.encoded(alias))}
        alias_act_path = self.base / "alias-act.json"
        self.put(alias_act_path, receipt)
        obs_path, context_path = self.base / "observations.json", self.base / "clone-context.json"
        self.put(obs_path, self.act)
        self.put(context_path, self.context)
        return {"material_root":root,"material_act_path":material_path,"alias_path":alias_path,
                "alias_act_path":alias_act_path,"observations_path":obs_path,"clone_context_path":context_path}

    def test_complete_assembly_and_exact_replay_preserve_source(self):
        inputs = self.complete_material()
        original_source = {p.name:p.read_bytes() for p in self.source_root.iterdir()}
        result, outputs = module.assemble(self.source_root, self.ack_path, now=NOW, **inputs)
        self.assertEqual(result["estado"], "ensamblado_offline")
        self.assertEqual(result["snapshots"], 2)
        for population in module.POPULATIONS:
            value = json.loads(outputs[population+"-fuente.acreditada-v1.json"])
            self.assertEqual(value["snapshot"]["cuenta"]["procedencia_autoridad"], "autoridad_maestra_acreditada")
            self.assertIsInstance(value["preimagen"], dict)
        destination = self.base / "output"
        module.write_bundle(destination, outputs)
        inodes = {p.name:p.stat().st_ino for p in (destination/"derivados-v1").iterdir()}
        module.write_bundle(destination, outputs)
        self.assertEqual(inodes, {p.name:p.stat().st_ino for p in (destination/"derivados-v1").iterdir()})
        self.assertEqual(original_source, {p.name:p.read_bytes() for p in self.source_root.iterdir()})
        changed = outputs | {"candidato-fuente.acreditada-v1.json": b"changed"}
        with self.assertRaisesRegex(module.AccreditationError, "replay_bundle_changed"):
            module.write_bundle(destination, changed)

    def test_material_change_missing_alias_and_duplicate_alias_fail(self):
        inputs = self.complete_material()
        result, outputs = module.assemble(self.source_root,self.ack_path,now=NOW,**(inputs|{"alias_path":None}))
        self.assertEqual(outputs,{})
        self.assertIn("alias_externo_nominal_pendiente",result["blockers"])
        alias = json.loads(inputs["alias_path"].read_bytes())
        alias["cuentas"].append(alias["cuentas"][0])
        receipt = json.loads(inputs["alias_act_path"].read_bytes())
        receipt["salida_sha256"] = module.sha(module.encoded(alias))
        with self.assertRaisesRegex(module.AccreditationError,"alias_account_ambiguous"):
            module.validate_alias(module.encoded(alias),receipt,inputs["material_act_path"].read_bytes(),self.metadata)
        self.put(inputs["material_root"]/"runtime-externo/ca/ca.crt",b"changed")
        with self.assertRaisesRegex(module.AccreditationError,"material_preimage_changed"):
            module.assemble(self.source_root,self.ack_path,now=NOW,**inputs)

    def test_symlink_git_hardlink_and_permissions_fail(self):
        real = self.base/"real"
        real.mkdir(mode=0o700)
        link = self.base/"link"
        link.symlink_to(real,target_is_directory=True)
        with self.assertRaises(OSError):
            module.PrivateDirectory(link)
        (real/".git").write_text("gitdir: fixture")
        with self.assertRaisesRegex(module.AccreditationError,"inside_git"):
            module.PrivateDirectory(real)
        target = self.base/"other.json"
        os.link(self.ack_path,target)
        with self.assertRaisesRegex(module.AccreditationError,"private_file_invalid"):
            module.read_private(target)
        target.unlink()
        self.ack_path.chmod(0o644)
        with self.assertRaisesRegex(module.AccreditationError,"private_file_invalid"):
            module.read_private(self.ack_path)

    def test_ancestor_swap_and_leaf_toctou_cannot_redirect(self):
        ancestor = self.base/"ancestor"
        ancestor.mkdir(mode=0o700)
        root = ancestor/"private"
        root.mkdir(mode=0o700)
        other = self.base/"other"
        other.mkdir(mode=0o700)
        (other/".git").write_text("fixture")
        with module.PrivateDirectory(root) as retained:
            ancestor.rename(self.base/"held")
            ancestor.symlink_to(other,target_is_directory=True)
            with self.assertRaisesRegex(module.AccreditationError,"private_directory_changed"):
                retained.check()
        old_read = module.os.read
        changed = False
        def swap(fd,count):
            nonlocal changed
            data = old_read(fd,count)
            if not changed:
                changed = True
                self.ack_path.rename(self.base/"held-ack.json")
                self.put(self.ack_path,self.ack)
            return data
        with patch.object(module.os,"read",side_effect=swap):
            with self.assertRaisesRegex(module.AccreditationError,"private_file_changed"):
                module.read_private(self.ack_path)

    def test_output_toctou_and_exclusive_commit_do_not_replace(self):
        _,outputs = module.assemble(self.source_root,self.ack_path,now=NOW,**self.complete_material())
        ancestor=self.base/"ancestor"
        ancestor.mkdir(mode=0o700)
        root=ancestor/"output"
        redirected=self.base/"redirected"
        redirected.mkdir(mode=0o700)
        (redirected/".git").write_text("fixture")
        original=module.rename_exclusive
        def swapped(fd,source,destination):
            ancestor.rename(self.base/"held")
            ancestor.symlink_to(redirected,target_is_directory=True)
            return original(fd,source,destination)
        with patch.object(module,"rename_exclusive",side_effect=swapped):
            with self.assertRaisesRegex(module.AccreditationError,"private_directory_changed"):
                module.write_bundle(root,outputs)
        self.assertEqual(set(p.name for p in redirected.iterdir()),{".git"})
        with module.PrivateDirectory(self.base) as parent:
            (self.base/"from").mkdir(mode=0o700)
            (self.base/"to").mkdir(mode=0o700)
            with self.assertRaisesRegex(module.AccreditationError,"exclusive_output_commit_failed"):
                module.rename_exclusive(parent.fd,"from","to")

    def test_duplicate_json_keys_fail(self):
        with self.assertRaisesRegex(module.AccreditationError,"duplicate_json_key"):
            module.decoded(b'{"version":1,"version":1}')


if __name__ == "__main__":
    unittest.main()
