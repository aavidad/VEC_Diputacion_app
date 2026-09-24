"""Contratos focales del alias nominal; sin token ni PostgreSQL reales."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import aprovisionar as app


SUBJECT = "per_" + "a" * 24
EXTERNAL = "cta_" + "b" * 24
INTERNAL = "cta_" + "c" * 24
ROOT = Path("/material-de-prueba")


def metadata() -> dict:
    return {"version": 1, "modulo": "/usr/lib/pkcs11.so", "token_label": "Token",
            "token_serial": "123456", "objeto_id_hex": "12" * 16,
            "clave_id": "clave-identidad-1", "clave_version": 3,
            "dominio_ref": "idh_" + "d" * 24,
            "espacio_identidad": "https://identidad.example.test/desarrollo",
            "pin_fichero": str(ROOT / "identidad" / "hmac.pin")}


def arguments(**changes: str) -> argparse.Namespace:
    data = {"subject_id": SUBJECT, "external_account_id": EXTERNAL,
            "internal_account_ref": INTERNAL, "database": "vec_test",
            "admin_user": "postgres", "pg_container": "vec-pg",
            "container_engine": "docker"}
    data.update(changes)
    return argparse.Namespace(**data)


def private_files() -> dict[Path, bytes]:
    identity = ROOT / "identidad"
    return {
        identity / "certificados.json": json.dumps({"version": 1, "certificados": [
            {"activo": True, "sujeto_id": SUBJECT, "cuenta_id": EXTERNAL}]}).encode(),
        identity / "contextos.json": json.dumps({"version": 1, "contextos": [
            {"cuenta_ref": INTERNAL}]}).encode(),
        identity / "hmac.json": json.dumps(metadata()).encode(),
        identity / "hmac.pin": b"pin-de-prueba-123\n",
    }


class AliasHMACTest(unittest.TestCase):
    def test_v3_roles_exige_preflight_y_lectura_de_gobierno(self) -> None:
        sql = app.role_sql({key: "clave-sintetica" for key in app.V3_ROLES}, "vec_test",
                           "ROLLBACK", app.V3_ROLES, app.V3_FUNCTIONS,
                           "vec_interno_v3_", {"gobierno_v3": "vec_interno_preflight_v3_desarrollo"})
        self.assertIn("comprobar_material_emision_interna_v1(jsonb)", sql)
        self.assertIn("leer_configuracion_interna_v1(jsonb)", sql)
        self.assertIn("has_function_privilege", sql)

    def test_mensaje_canonico_separa_cuenta_y_sujeto(self) -> None:
        meta = metadata()
        expected = b"".join(len(part.encode()).to_bytes(4, "big") + part.encode()
                            for part in ("vec.identidad.hmac-sha256.v1", meta["dominio_ref"],
                                         meta["espacio_identidad"], "3", "cuenta", EXTERNAL))
        self.assertEqual(app.mensaje_hmac_identidad(meta, "cuenta", EXTERNAL), expected)
        self.assertNotEqual(app.mensaje_hmac_identidad(meta, "sujeto", EXTERNAL), expected)

    def test_replay_usa_misma_operacion_y_funcion_existente(self) -> None:
        files = private_files()
        statements: list[str] = []
        account_digest = hashlib.sha256(b"cuenta").digest()
        subject_digest = hashlib.sha256(b"sujeto").digest()

        def read(path: Path) -> bytes:
            return files[path]

        with patch.object(app, "read_private", side_effect=read), \
             patch.object(app, "firmar_alias_hmac", return_value=(account_digest, subject_digest)) as sign, \
             patch.object(app, "psql", side_effect=lambda sql, *_: statements.append(sql)):
            app.registrar_alias_hmac(ROOT, arguments())
            app.registrar_alias_hmac(ROOT, arguments())
        self.assertEqual(sign.call_count, 2)
        self.assertEqual(len(statements), 4)
        self.assertEqual(statements[0].replace("ROLLBACK", "COMMIT"), statements[1])
        self.assertEqual(statements[:2], statements[2:])
        self.assertIn("registrar_alias_hmac_cuenta_v1", statements[1])
        self.assertIn("LOCK TABLE vec_identidad_sesiones_v1.alias_hmac_cuenta", statements[1])
        self.assertIn(account_digest.hex(), statements[1])
        self.assertIn(subject_digest.hex(), statements[1])
        self.assertNotIn(EXTERNAL, statements[1])
        self.assertNotIn(SUBJECT, statements[1])
        self.assertNotIn("pin-de-prueba", statements[1])

    def test_cruce_local_impide_firma_y_sql(self) -> None:
        files = private_files()
        files[ROOT / "identidad" / "certificados.json"] = json.dumps(
            {"version": 1, "certificados": [{"activo": True,
             "sujeto_id": SUBJECT, "cuenta_id": "cta_" + "x" * 24}]}).encode()
        with patch.object(app, "read_private", side_effect=lambda path: files[path]), \
             patch.object(app, "firmar_alias_hmac") as sign, patch.object(app, "psql") as sql:
            with self.assertRaisesRegex(RuntimeError, "no vinculado"):
                app.registrar_alias_hmac(ROOT, arguments())
        sign.assert_not_called()
        sql.assert_not_called()

    def test_certificado_revocado_impide_alias(self) -> None:
        files = private_files()
        files[ROOT / "identidad" / "certificados.json"] = json.dumps(
            {"version": 1, "certificados": [{"activo": False,
             "sujeto_id": SUBJECT, "cuenta_id": EXTERNAL}]}).encode()
        with patch.object(app, "read_private", side_effect=lambda path: files[path]), \
             patch.object(app, "firmar_alias_hmac") as sign, patch.object(app, "psql") as sql:
            with self.assertRaisesRegex(RuntimeError, "no vinculado"):
                app.registrar_alias_hmac(ROOT, arguments())
        sign.assert_not_called()
        sql.assert_not_called()

    def test_cruce_sql_y_resultado_nulo_rechazan(self) -> None:
        query = app.sql_alias_hmac("opr_" + "e" * 64, INTERNAL, metadata(),
                                   bytes.fromhex("11" * 32), bytes.fromhex("22" * 32),
                                   "COMMIT")
        self.assertIn("a.sujeto_id_hmac=decode", query)
        self.assertIn("a.cuenta_ref=", query)
        self.assertIn("IS DISTINCT FROM", query)
        self.assertIn("RAISE EXCEPTION 'alias HMAC cruzado'", query)


if __name__ == "__main__":
    unittest.main()
