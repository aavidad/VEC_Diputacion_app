from __future__ import annotations

import ast
import unittest
from collections import Counter
from dataclasses import replace
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from scripts import capturar_presentacion_web as capturador


class ManifiestoRevisionWebTests(unittest.TestCase):
    def test_cubre_todas_las_vistas_y_tamanos_acordados(self) -> None:
        por_superficie = Counter(vista.superficie for vista in capturador.MANIFIESTO_VISTAS)
        self.assertEqual(
            por_superficie,
            {
                "lanzador": 1,
                "portal-publico": 1,
                "area-aspirante": 14,
                "gestion-rrhh": 16,
            },
        )
        self.assertEqual(
            {(tamano.ancho, tamano.alto) for tamano in capturador.TAMANOS_VISTA},
            {(1440, 1000), (1024, 900), (390, 844)},
        )
        self.assertEqual(capturador.validar_manifiesto(), [])

    def test_vistas_aspirante_son_exhaustivas_y_el_detalle_es_determinista(self) -> None:
        vistas = {
            vista.clave.removeprefix("aspirante-"): vista
            for vista in capturador.MANIFIESTO_VISTAS
            if vista.superficie == "area-aspirante"
        }
        self.assertEqual(set(vistas), {
            "inicio", "convocatorias", "convocatoria", "perfil", "meritos",
            "solicitud", "autobaremacion", "seguimiento", "llamamientos",
            "subsanaciones", "alegaciones", "mensajes", "certificados", "ayuda",
        })
        self.assertIn("id=DEMO-CONV-001", vistas["convocatoria"].ruta)

    def test_vistas_rrhh_coinciden_con_el_menu_final(self) -> None:
        rutas = {
            vista.clave.removeprefix("rrhh-")
            for vista in capturador.MANIFIESTO_VISTAS
            if vista.superficie == "gestion-rrhh"
        }
        self.assertEqual(rutas, set(capturador.RUTAS_MENU_RRHH))
        self.assertNotIn("reglas", rutas)
        self.assertNotIn("consulta", rutas)

    def test_toda_ruta_privada_usa_presentacion_rrhh(self) -> None:
        for escenario in capturador.MANIFIESTO:
            superficie = capturador.SUPERFICIES[escenario.superficie]
            if not superficie.privada:
                continue
            consulta = parse_qs(urlparse(escenario.ruta).query)
            self.assertEqual(consulta.get("presentacion"), ["rrhh"], escenario.clave)

    def test_flujos_se_distinguen_y_cubren_interacciones_demo(self) -> None:
        self.assertEqual(len(capturador.MANIFIESTO_FLUJOS), 7)
        claves = {flujo.clave for flujo in capturador.MANIFIESTO_FLUJOS}
        self.assertEqual(claves, {
            "publico-ficha-convocatoria",
            "aspirante-convocatoria-abierta",
            "aspirante-confirmacion-demo",
            "aspirante-recibo-demo",
            "rrhh-borrador-abierto",
            "rrhh-recibo-demo",
            "rrhh-perfil-tecnico-restringido",
        })
        for flujo in capturador.MANIFIESTO_FLUJOS:
            self.assertEqual(flujo.tipo, "flujo")
            self.assertTrue(flujo.pasos)
        privados = [flujo for flujo in capturador.MANIFIESTO_FLUJOS if flujo.requiere_demo]
        self.assertTrue(privados)
        self.assertTrue(all(capturador.SUPERFICIES[flujo.superficie].privada for flujo in privados))

        tecnico = next(
            flujo for flujo in capturador.MANIFIESTO_FLUJOS
            if flujo.clave == "rrhh-perfil-tecnico-restringido"
        )
        self.assertIn("perfil=tecnico", tecnico.ruta)
        self.assertEqual(
            [paso.accion for paso in tecnico.pasos],
            ["esperar", "esperar-habilitado", "esperar-deshabilitado"],
        )

    def test_detecta_duplicados_del_manifiesto_sin_navegador(self) -> None:
        original = capturador.MANIFIESTO_VISTAS[0]
        duplicada = replace(original)
        errores = capturador.validar_manifiesto((original, duplicada))
        self.assertTrue(any("clave de escenario duplicada" in error for error in errores))
        self.assertTrue(any("ruta de vista duplicada" in error for error in errores))


class HelpersRevisionWebTests(unittest.TestCase):
    def test_url_base_y_construccion_de_ruta(self) -> None:
        self.assertEqual(
            capturador.normalizar_url_base(" http://127.0.0.1:8081/ "),
            "http://127.0.0.1:8081",
        )
        self.assertEqual(
            capturador.construir_url("http://127.0.0.1:8081/", "/bolsa/?vista=demo#ficha"),
            "http://127.0.0.1:8081/bolsa/?vista=demo#ficha",
        )
        for invalida in ("", "127.0.0.1:8081", "ftp://localhost", "http://u:p@localhost", "http://localhost/?x=1"):
            with self.subTest(invalida=invalida), self.assertRaises(ValueError):
                capturador.normalizar_url_base(invalida)

    def test_slug_castellano_es_estable(self) -> None:
        self.assertEqual(capturador.slug_castellano("Méritos y Baremación"), "meritos-y-baremacion")
        self.assertEqual(capturador.slug_castellano("  ---  "), "sin-nombre")

    def test_solo_descarta_aborto_de_media_con_respuesta_http_valida(self) -> None:
        fallidos = [
            {"url": "http://local/guia.mp3", "tipo": "media", "error": "net::ERR_ABORTED"},
            {"url": "http://local/rota.mp3", "tipo": "media", "error": "net::ERR_ABORTED"},
            {"url": "http://local/app.js", "tipo": "script", "error": "net::ERR_ABORTED"},
        ]
        correctos = [{"url": "http://local/guia.mp3", "tipo": "media", "estado": 206}]
        filtrados = capturador.filtrar_abortos_media_exitosos(fallidos, correctos)
        self.assertEqual([recurso["url"] for recurso in filtrados], [
            "http://local/rota.mp3",
            "http://local/app.js",
        ])

    def test_resumen_codigo_salida_e_informe_distinguen_vista_y_flujo(self) -> None:
        base = {
            "clave": "uno",
            "nombre": "Escenario",
            "superficie": "lanzador",
            "nombre_superficie": "Lanzador",
            "ruta": "/",
            "url": "http://localhost/",
            "tamano": {"clave": "movil", "nombre": "Móvil", "ancho": 390, "alto": 844},
            "captura": "capturas/movil/vista/lanzador/uno.png",
            "alcance_captura": "pagina-completa",
            "duracion_ms": 5,
            "metricas": {},
        }
        resultados = [
            {**base, "tipo": "vista", "correcto": True, "hallazgos": []},
            {
                **base,
                "clave": "dos",
                "tipo": "flujo",
                "correcto": False,
                "hallazgos": [{"severidad": "error", "codigo": "prueba", "mensaje": "Hallazgo"}],
            },
        ]
        resumen = capturador.resumir_resultados(resultados)
        self.assertEqual((resumen["vistas"], resumen["flujos"]), (1, 1))
        self.assertEqual(capturador.codigo_salida(resultados, tolerante=False), 1)
        self.assertEqual(capturador.codigo_salida(resultados, tolerante=True), 0)
        informe = {
            "correcto": False,
            "tolerante": False,
            "url_base": "http://localhost",
            "generado_en": "2026-07-18T00:00:00+00:00",
            "resumen": resumen,
            "resultados": resultados,
        }
        markdown = capturador.crear_informe_markdown(informe)
        self.assertIn("(1 vistas, 1 flujos)", markdown)
        self.assertIn("| flujo |", markdown)
        self.assertIn("`prueba`", markdown)

    def test_playwright_no_se_importa_en_el_nivel_superior(self) -> None:
        ruta = Path(capturador.__file__)
        arbol = ast.parse(ruta.read_text(encoding="utf-8"))
        importaciones_superiores = [
            nodo
            for nodo in arbol.body
            if isinstance(nodo, (ast.Import, ast.ImportFrom))
        ]
        modulos = {
            alias.name.split(".")[0]
            for nodo in importaciones_superiores
            for alias in nodo.names
        }
        self.assertNotIn("playwright", modulos)


if __name__ == "__main__":
    unittest.main()
