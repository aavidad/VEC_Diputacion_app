"""Contadores RPT: familias históricas y AD172, sin ejecutar el runner completo."""

from __future__ import annotations

import os
import re
import subprocess
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
RUNNER = RAIZ / "deploy/postgresql/personal/pruebas_sql/relacion_para_rpt_000027_pg18.sh"
FUENTE = RUNNER.read_text(encoding="utf-8")
CONSULTA = re.search(r"<<'SQLCONTADORES'\n(.*?)\nSQLCONTADORES", FUENTE, re.S)
FUNCION = re.search(r"\n(contadores\(\) \{.*?\n\})\narchivo\(\)", FUENTE, re.S)
assert CONSULTA is not None and FUNCION is not None
SQL = CONSULTA.group(1)
CONTENEDOR = os.environ.get("VEC_RPT_MIXTA_TEST_CONTAINER", "")


class ContadoresRPTShellTests(unittest.TestCase):
    def ejecutar(self, valor: str) -> subprocess.CompletedProcess[str]:
        # Sólo se ejecuta la función contadora extraída; valor no conecta a PG.
        codigo = (
            'set -euo pipefail\nfixture=$1\n'
            'valor() { printf "%s\\n" "$fixture"; }\n'
            'fallo() { printf "%s\\n" "$1" >&2; exit 1; }\n'
            + FUNCION.group(1)
            + "\ncontadores\n"
        )
        return subprocess.run(
            ["bash", "-c", codigo, "rpt-mixta-test", valor],
            capture_output=True, text=True, check=False, timeout=5,
        )

    def test_conserva_formato_de_cuatro_contadores(self) -> None:
        proceso = self.ejecutar("3|6243|6243|0")
        self.assertEqual(proceso.returncode, 0, proceso.stderr)
        self.assertEqual(proceso.stdout, "3|6243|6243|0\n")

    def test_incompatible_no_se_acepta_como_contador_igual_antes_y_despues(self) -> None:
        for valor in ("auditoria_incompatible", "", "1|2|3", "1|2|3|4|5", "1|2|3|-1", "1|2|3|4\n1"):
            with self.subTest(valor=valor):
                proceso = self.ejecutar(valor)
                self.assertNotEqual(proceso.returncode, 0)
                self.assertEqual(proceso.stdout, "")
                self.assertIn("familia o versión de auditoría incompatible", proceso.stderr)

    def test_reanudacion_y_resultados_usan_el_mismo_contador(self) -> None:
        self.assertIn("estado_go=$(contadores)", FUENTE)
        self.assertIn("$(contadores) == '3|6243|6243|0'", FUENTE)
        self.assertEqual(FUENTE.count("FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a"), 1)


@unittest.skipUnless(CONTENEDOR, "requiere contenedor PostgreSQL 18 sintético propio")
class ContadoresRPTPostgreSQLTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        if CONTENEDOR != "codexb-rpt-auditoria-mixta-20261003-pg":
            raise AssertionError("contenedor nominal sintético requerido")
        cls.psql("SELECT current_setting('server_version_num')", "180004")

    @staticmethod
    def psql(sql: str, esperado: str) -> None:
        proceso = subprocess.run(
            ["docker", "exec", "-i", CONTENEDOR, "env", "-i", "PATH=/usr/local/bin:/usr/bin:/bin",
             "psql", "-h", "/tmp", "-U", "postgres", "-d", "postgres", "-X", "-qAt",
             "-v", "ON_ERROR_STOP=1"],
            input=sql, text=True, capture_output=True, check=False, timeout=10,
        )
        if proceso.returncode != 0 or proceso.stdout.strip() != esperado:
            raise AssertionError(f"consulta sintética: {proceso.returncode}; {proceso.stdout!r}; {proceso.stderr}")

    def comprobar(self, filas: list[tuple[str, int | None, str | None, str | None]],
                  esperado: str, con_version: bool = True) -> None:
        # Cada vector usa tablas mínimas privadas dentro de ROLLBACK. No se
        # instalan funciones, ACL ni migraciones ni se restaura una captura.
        sentencias = [
            "BEGIN; CREATE SCHEMA vec_personal; CREATE SCHEMA vec_autorizacion_atestada_v3;",
            "CREATE TABLE vec_personal.recibo_relacion_para_rpt(id integer);",
            "CREATE TABLE vec_autorizacion_atestada_v3.consumo_decision_v3(id integer);",
            "INSERT INTO vec_personal.recibo_relacion_para_rpt VALUES(1);",
            "INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3 VALUES(1),(2);",
            "CREATE TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 "
            "(tipo_registro text,proceso text,canal text"
            + (",version_consumo smallint" if con_version else "") + ");",
        ]
        def literal(valor: str | int | None) -> str:
            if valor is None:
                return "NULL"
            if isinstance(valor, int):
                return str(valor)
            return "'" + valor.replace("'", "''") + "'"
        for familia, version, proceso, canal in filas:
            valores = [familia, proceso, canal] + ([version] if con_version else [])
            sentencias.append("INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 VALUES("
                               + ",".join(literal(v) for v in valores) + ");")
        sentencias.extend([SQL, "ROLLBACK;"])
        self.psql("\n".join(sentencias), esperado)

    def test_historicos_antes_y_despues_de_agregar_columna(self) -> None:
        for columna in (False, True):
            with self.subTest(version_consumo=columna):
                self.comprobar([("consumo_confirmado", None, None, None)], "1|2|1|0", columna)

    def test_mezcla_historico_v2_intento_y_eventos_admin(self) -> None:
        self.comprobar([
            ("consumo_confirmado", None, None, None),
            ("consumo_confirmado_v2", 2, "rpt27-ensayo", "interna_corporativa"),
            ("intento_nominal", None, "rpt27-ensayo", "interna_corporativa"),
            ("preperfil_autenticado", None, "vec-admin", "administracion_privilegiada"),
            ("bootstrap_operador", None, "vec-admin", "operacion_tecnica_privada"),
        ], "1|2|2|1")

    def test_v2_admite_solo_canales_publicados(self) -> None:
        for canal in ("interna_corporativa", "administracion_privilegiada", "externa_personal"):
            with self.subTest(canal=canal):
                self.comprobar([("consumo_confirmado_v2", 2, "rpt27-ensayo", canal)], "1|2|1|0")

    def test_familia_version_y_origen_incompatibles_detienen_el_contador(self) -> None:
        filas = [
            ("desconocido", None, None, None),
            ("consumo_confirmado_v3", 3, "rpt27-ensayo", "interna_corporativa"),
            ("consumo_confirmado", 1, None, None),
            ("consumo_confirmado", 2, None, None),
            ("consumo_confirmado", None, "rpt27-ensayo", "interna_corporativa"),
            ("consumo_confirmado_v2", None, "rpt27-ensayo", "interna_corporativa"),
            ("consumo_confirmado_v2", 1, "rpt27-ensayo", "interna_corporativa"),
            ("consumo_confirmado_v2", 3, "rpt27-ensayo", "interna_corporativa"),
            ("consumo_confirmado_v2", 2, None, "interna_corporativa"),
            ("consumo_confirmado_v2", 2, "", "interna_corporativa"),
            ("consumo_confirmado_v2", 2, "rpt27-ensayo", "operacion_tecnica_privada"),
            ("intento_nominal", 2, "rpt27-ensayo", "interna_corporativa"),
            ("preperfil_autenticado", 2, "vec-admin", "administracion_privilegiada"),
            ("bootstrap_operador", 2, "vec-admin", "operacion_tecnica_privada"),
        ]
        for fila in filas:
            with self.subTest(fila=fila):
                self.comprobar([("consumo_confirmado", None, None, None), fila], "auditoria_incompatible")


if __name__ == "__main__":
    unittest.main()
