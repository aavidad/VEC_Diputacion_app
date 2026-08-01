"""Pruebas focales y mutantes del catálogo operativo canónico."""

import copy
import importlib.util
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

DIRECTORIO = Path(__file__).resolve().parent
CATALOGO = DIRECTORIO / "catalogo_operativo.json"
HERRAMIENTA = DIRECTORIO / "verificar_catalogo_operativo.py"
RAIZ = DIRECTORIO.parents[1]
ESPECIFICACION = importlib.util.spec_from_file_location(
    "catalogo_operativo", HERRAMIENTA
)
MODULO = importlib.util.module_from_spec(ESPECIFICACION)
ESPECIFICACION.loader.exec_module(MODULO)

PLANTILLA_PUBLICA = [
    "VEC_EXECUTION_PROFILE=produccion",
    "VEC_AUTH_MODE=disabled",
    "VEC_BOLSA_PUBLICA_DATABASE_URL=",
    "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256=",
    "VEC_BOLSA_CATEGORIES_CATALOG_ID=categorias-profesionales",
    "VEC_BOLSA_CATEGORIES_CATALOG_VERSION=",
    "VEC_BOLSA_CATEGORIES_CATALOG_SHA256=",
    "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256=",
]


class CatalogoOperativoTest(unittest.TestCase):
    def catalogo(self) -> dict:
        return json.loads(CATALOGO.read_text(encoding="utf-8"))

    def validar(self, catalogo: dict, raiz: Path = RAIZ) -> str:
        return "\n".join(MODULO.validar_catalogo(catalogo, raiz))

    def uso(self, catalogo: dict, superficie: str, nombre: str) -> dict:
        return next(
            uso
            for uso in catalogo["uso_por_superficie"][superficie]
            if uso["nombre"] == nombre
        )

    def ejecutar_cli(self, *argumentos: str) -> subprocess.CompletedProcess:
        return subprocess.run(
            [sys.executable, "-B", str(HERRAMIENTA), *argumentos],
            text=True,
            capture_output=True,
            check=False,
        )

    def copiar_go_productivo(self, destino: Path) -> None:
        for directorio in ("cmd", "config", "internal"):
            for origen in (RAIZ / directorio).rglob("*.go"):
                relativa = origen.relative_to(RAIZ)
                if origen.name.endswith("_test.go") or "testdata" in relativa.parts:
                    continue
                copia = destino / relativa
                copia.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(origen, copia)

    def test_catalogo_canonico_supera_validacion(self) -> None:
        self.assertEqual(self.validar(self.catalogo()), "")

    def test_variable_faltante_no_supera_paridad(self) -> None:
        catalogo = self.catalogo()
        catalogo["variables"].pop()
        self.assertIn("CAT005", self.validar(catalogo))

    def test_variable_duplicada_no_supera_unicidad(self) -> None:
        catalogo = self.catalogo()
        catalogo["variables"].append(copy.deepcopy(catalogo["variables"][0]))
        self.assertIn("CAT004", self.validar(catalogo))

    def test_secreto_no_admite_valor_ni_desclasificacion(self) -> None:
        catalogo = self.catalogo()
        secreto = next(
            variable for variable in catalogo["variables"] if variable["secreto"]
        )
        secreto["valor_predeterminado"] = "no-versionar"
        self.assertIn("CAT021", self.validar(catalogo))
        catalogo = self.catalogo()
        secreto = next(
            variable
            for variable in catalogo["variables"]
            if variable["nombre"] == "VEC_BOLSA_PUBLICA_DATABASE_URL"
        )
        secreto["secreto"] = False
        self.assertIn("CAT020", self.validar(catalogo))

    def test_estado_sentinela_publico_no_se_puede_relajar(self) -> None:
        catalogo = self.catalogo()
        uso = self.uso(catalogo, "vec-publico", "VEC_BOLSA_PUBLIC_SOURCE_PATH")
        uso["estado"] = "configurable"
        self.assertIn("CAT041", self.validar(catalogo))

    def test_lector_declarado_debe_ser_exacto(self) -> None:
        catalogo = self.catalogo()
        variable = next(
            variable
            for variable in catalogo["variables"]
            if variable["nombre"] == "VEC_HTTP_ADDR"
        )
        variable["lectores"].pop()
        self.assertIn("CAT008", self.validar(catalogo))

    def test_plantilla_publica_es_el_conjunto_exacto_del_runbook(self) -> None:
        resultado = self.ejecutar_cli("--plantilla", "vec-publico")
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        asignaciones = [
            linea for linea in resultado.stdout.splitlines() if "=" in linea
        ]
        self.assertEqual(asignaciones, PLANTILLA_PUBLICA)
        for prohibida in MODULO.SENTINELAS_PUBLICOS:
            self.assertNotIn(prohibida + "=", resultado.stdout)

    def test_omision_publica_requerida_es_rechazada(self) -> None:
        catalogo = self.catalogo()
        uso = self.uso(
            catalogo,
            "vec-publico",
            "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256",
        )
        uso["plantilla"] = "omitir"
        self.assertIn("CAT045", self.validar(catalogo))

    def test_ancla_grupo_tls_e_infraestructura_no_se_relajan(self) -> None:
        catalogo = self.catalogo()
        ancla = next(
            variable
            for variable in catalogo["variables"]
            if variable["nombre"] == "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256"
        )
        ancla["ancla"] = False
        self.assertIn("CAT023", self.validar(catalogo))
        catalogo = self.catalogo()
        catalogo["grupos_configuracion"][0]["miembros"][0] = "VEC_HTTP_ADDR"
        self.assertIn("CAT059", self.validar(catalogo))
        catalogo = self.catalogo()
        catalogo["infraestructura"]["ceph"] = []
        self.assertIn("CAT056", self.validar(catalogo))

    def test_catalogo_arbitrario_no_es_opcion_cli(self) -> None:
        resultado = self.ejecutar_cli("--catalogo", "/dev/stdin", "--validar")
        self.assertEqual(resultado.returncode, 2)

    def test_literal_nuevo_en_cmd_vec_publico_rompe_paridad(self) -> None:
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            self.copiar_go_productivo(raiz)
            mutante = raiz / "cmd/vec-publico/config_cat_mutante.go"
            mutante.parent.mkdir(parents=True, exist_ok=True)
            mutante.write_text(
                'package main\n\nconst configCatMutante = "VEC_CONFIG_CAT_MUTANTE"\n',
                encoding="utf-8",
            )
            self.assertIn("CAT005", self.validar(self.catalogo(), raiz))

    def test_lectura_adicional_en_fichero_permitido_rompe_inventario(self) -> None:
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            self.copiar_go_productivo(raiz)
            mutante = raiz / "config/config.go"
            with mutante.open("a", encoding="utf-8") as archivo:
                archivo.write(
                    '\nfunc configCatLecturaMutante() { _ = os.Getenv("VEC_HTTP_ADDR") }\n'
                )
            self.assertIn("CAT013", self.validar(self.catalogo(), raiz))


if __name__ == "__main__":
    unittest.main()
