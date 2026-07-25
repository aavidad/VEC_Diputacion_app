"""Regresión del aislamiento entre el portal base y Dietas/cartografía."""

from pathlib import Path
import re
import unittest


RAIZ = Path(__file__).resolve().parents[2]


class PresentacionModularTest(unittest.TestCase):
    def test_proxy_no_depende_de_servicios_de_dietas(self):
        compose = (RAIZ / "docker-compose.yml").read_text(encoding="utf-8")
        bloque = compose.split("  proxy-presentacion:", 1)[1].split(
            "\n  entrada-remota-presentacion:", 1
        )[0]
        dependencias = bloque.split("    depends_on:", 1)[1].split(
            "\n    ports:", 1
        )[0]
        self.assertIn("vec-presentacion:", dependencias)
        self.assertNotIn("vec-cartografia-presentacion", dependencias)
        self.assertNotIn("tiles-osm-presentacion", dependencias)
        self.assertNotIn("osrm-presentacion", dependencias)

    def test_cartografia_es_un_perfil_opcional(self):
        compose = (RAIZ / "docker-compose.yml").read_text(encoding="utf-8")
        for inicio, fin in (
            ("  vec-cartografia-presentacion:", "\n  osrm-presentacion:"),
            ("  osrm-presentacion:", "\n  tiles-osm-presentacion:"),
            ("  tiles-osm-presentacion:", "\n  smoke-cartografia-presentacion:"),
        ):
            bloque = compose.split(inicio, 1)[1].split(fin, 1)[0]
            self.assertRegex(
                bloque,
                r"profiles:\s*\n\s*-\s*presentacion-cartografia\b",
            )
            self.assertNotRegex(
                bloque,
                r"profiles:\s*\n\s*-\s*presentacion\s*(?:\n|$)",
            )

    def test_proxy_resuelve_modulos_opcionales_en_cada_peticion(self):
        nginx = (
            RAIZ / "deploy/proxy-local/nginx-presentacion.conf"
        ).read_text(encoding="utf-8")
        self.assertIn("resolver 127.0.0.11", nginx)
        self.assertNotRegex(
            nginx,
            r"upstream\s+(?:mediador_cartografico|teselas_internas)",
        )
        self.assertRegex(
            nginx,
            re.compile(
                r"set \$mediador_cartografico\s+"
                r"http://vec-cartografia-presentacion:8080;",
                re.MULTILINE,
            ),
        )
        self.assertIn(
            "set $teselas_internas http://tiles-osm-presentacion:8080;",
            nginx,
        )

    def test_arranque_base_solo_activa_cartografia_bajo_peticion(self):
        script = (
            RAIZ / "scripts/arrancar_presentacion_rrhh.sh"
        ).read_text(encoding="utf-8")
        self.assertIn('VEC_PRESENTACION_CON_CARTOGRAFIA:-false', script)
        self.assertIn("--profile presentacion-cartografia", script)
        self.assertIn('COMPOSE_PROJECT_NAME:-vec-ct-local', script)
        self.assertRegex(
            script,
            r'if \[ "\$\{VEC_PRESENTACION_CON_CARTOGRAFIA:-false\}" '
            r'= "true" \]; then',
        )


if __name__ == "__main__":
    unittest.main()
