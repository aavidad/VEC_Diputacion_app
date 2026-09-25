"""Pruebas focales del aprovisionamiento gobernado de empleados sintéticos."""

from __future__ import annotations

import copy
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import personal_altas_sinteticas as altas  # noqa: E402

PERSONA = "per_" + "a" * 24
PLAN = {
    "version": 1,
    "organismo_ref": "org:sintetico",
    "procedencia": {"acto_ref": "acto:alta-sintetica", "fuente_ref": "fuente:plan-sintetico",
                    "fuente_version": 1, "fuente_huella_sha256": "b" * 64},
    "catalogo": [{"tipo": "regimen", "ref": "reg:funcionario", "version": 1,
                  "denominacion": "Funcionario de carrera", "vigente_desde": "2026-01-01", "vigente_hasta": ""}],
    "altas": [{"persona_ref": PERSONA, "unidad_ref": "uni:sintetica",
               "regimen": {"ref": "reg:funcionario", "version": 1}, "modalidad": {"ref": "mod:carrera", "version": 1},
               "vigente_desde": "2026-09-25", "vigente_hasta": ""}],
}


class PlanTest(unittest.TestCase):
    def test_plan_valido_y_rechazos(self) -> None:
        plan = altas.validar_plan(copy.deepcopy(PLAN))
        self.assertEqual(plan["altas"][0]["persona_ref"], PERSONA)
        casos = []
        for campo, valor in (("version", 2), ("organismo_ref", "Org"), ("altas", [])):
            malo = copy.deepcopy(PLAN)
            malo[campo] = valor
            casos.append(malo)
        malo = copy.deepcopy(PLAN)
        malo["altas"][0]["nombre"] = "Dato civil"
        casos.append(malo)
        malo = copy.deepcopy(PLAN)
        malo["altas"].append(copy.deepcopy(malo["altas"][0]))
        casos.append(malo)
        malo = copy.deepcopy(PLAN)
        malo["altas"][0]["vigente_hasta"] = "2026-09-01"
        casos.append(malo)
        malo = copy.deepcopy(PLAN)
        malo["catalogo"][0]["denominacion"] = "Con\tcontrol"
        casos.append(malo)
        for caso in casos:
            with self.assertRaises(altas.PlanInvalido):
                altas.validar_plan(caso)

    def test_clave_determinista_uuid_v4(self) -> None:
        a = altas.clave_idempotencia("vec.personal.alta-sintetica.v1", "org:sintetico", PERSONA)
        b = altas.clave_idempotencia("vec.personal.alta-sintetica.v1", "org:sintetico", PERSONA)
        c = altas.clave_idempotencia("vec.personal.alta-sintetica.v1", "org:sintetico", "per_" + "b" * 24)
        self.assertEqual(a, b)
        self.assertNotEqual(a, c)
        self.assertRegex(a, r"^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

    def test_huella_de_catalogo_igual_que_el_portal(self) -> None:
        # Valor de calcularHuellaPublicacionCatalogoB2 (registro-b2-catalogos-cliente.js).
        entrada = altas.validar_plan(copy.deepcopy(PLAN))["catalogo"][0]
        self.assertEqual(altas.huella_catalogo("org:sintetico", entrada),
                         "ae8011a4b1a250f90395a74c4fcffca245e06fcf32697b1a2af45bfeedd29237")

    def test_plan_privado_fuera_del_repositorio(self) -> None:
        repo = Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory() as directorio:
            ruta = Path(directorio) / "plan.json"
            ruta.write_text(json.dumps(PLAN), encoding="utf-8")
            os.chmod(ruta, 0o644)
            with self.assertRaises(altas.PlanInvalido):
                altas._leer_privado(str(ruta), repo)
            os.chmod(ruta, 0o600)
            self.assertEqual(json.loads(altas._leer_privado(str(ruta), repo)), PLAN)
        with self.assertRaises(altas.PlanInvalido):
            altas._leer_privado(str(Path(__file__).resolve()), repo)
        with self.assertRaises(altas.PlanInvalido):
            json.loads('{"version":1,"version":1}', object_pairs_hook=altas._sin_duplicados)


class EjecucionTest(unittest.TestCase):
    def ejecutar(self, respuestas: dict[str, list[tuple[int, object]]]) -> tuple[int, list[tuple[str, dict, str]], list[str]]:
        enviados: list[tuple[str, dict, str]] = []
        mensajes: list[str] = []

        def enviar(ruta: str, cuerpo: dict, clave: str) -> tuple[int, object]:
            enviados.append((ruta, cuerpo, clave))
            return respuestas[ruta].pop(0)

        plan = altas.validar_plan(copy.deepcopy(PLAN))
        return altas.ejecutar(plan, enviar, mensajes.append), enviados, mensajes

    def test_alta_y_repeticion_con_la_misma_clave(self) -> None:
        empleado = "emp_" + "c" * 24
        recibo = {"data": {"recibo": {"empleado_ref": empleado}}}
        codigo, enviados, mensajes = self.ejecutar({altas.RUTA_CATALOGO: [(201, {})], altas.RUTA_ALTA: [(201, recibo)]})
        self.assertEqual(codigo, 0)
        self.assertEqual([r for r, _, _ in enviados], [altas.RUTA_CATALOGO, altas.RUTA_ALTA])
        self.assertEqual(enviados[0][1]["huella_sha256"], "ae8011a4b1a250f90395a74c4fcffca245e06fcf32697b1a2af45bfeedd29237")
        self.assertEqual(enviados[1][1]["organismo_ref"], "org:sintetico")
        self.assertNotIn("nombre", json.dumps(enviados[1][1]))
        self.assertIn(empleado, mensajes[-1])
        codigo2, enviados2, _ = self.ejecutar({altas.RUTA_CATALOGO: [(200, {})], altas.RUTA_ALTA: [(200, recibo)]})
        self.assertEqual(codigo2, 0)
        self.assertEqual([c for _, _, c in enviados], [c for _, _, c in enviados2])

    def test_conflicto_no_crea_otra_alta_y_caida_no_es_exito(self) -> None:
        codigo, _, mensajes = self.ejecutar({altas.RUTA_CATALOGO: [(409, {})], altas.RUTA_ALTA: [(409, {})]})
        self.assertEqual(codigo, 0)
        self.assertTrue(any("409" in m for m in mensajes))
        for estado in (0, 401, 403, 503):
            codigo, _, _ = self.ejecutar({altas.RUTA_CATALOGO: [(201, {})], altas.RUTA_ALTA: [(estado, None)]})
            self.assertEqual(codigo, 1, estado)
        codigo, enviados, _ = self.ejecutar({altas.RUTA_CATALOGO: [(503, None)], altas.RUTA_ALTA: []})
        self.assertEqual((codigo, len(enviados)), (1, 1))
        codigo, _, _ = self.ejecutar({altas.RUTA_CATALOGO: [(201, {})], altas.RUTA_ALTA: [(201, {"data": {}})]})
        self.assertEqual(codigo, 1)

    def test_solo_https(self) -> None:
        with self.assertRaises(altas.PlanInvalido):
            altas.crear_enviar("http://localhost:8443", "c", "k", "ca")
        self.assertIsNotNone(re.compile(altas.RUTA_ALTA))


if __name__ == "__main__":
    unittest.main()
