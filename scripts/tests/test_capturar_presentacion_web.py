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
                "gestion-rrhh": 18,
            },
        )
        self.assertEqual(
            {(tamano.ancho, tamano.alto) for tamano in capturador.TAMANOS_VISTA},
            {(1440, 1000), (1024, 900), (390, 844)},
        )
        self.assertEqual(capturador.validar_manifiesto(), [])
        lanzador = next(
            vista for vista in capturador.MANIFIESTO_VISTAS
            if vista.clave == "lanzador-recorrido"
        )
        self.assertEqual(lanzador.titulo_esperado, "Demostración funcional del portal")

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
            and vista.clave not in {"rrhh-cronos", "rrhh-dietas"}
        }
        self.assertEqual(rutas, set(capturador.RUTAS_MENU_RRHH))
        self.assertNotIn("reglas", rutas)
        self.assertNotIn("consulta", rutas)

    def test_cronos_y_dietas_se_auditan_como_modulos_internos_en_tres_tamanos(self) -> None:
        modulos = {
            vista.clave: vista
            for vista in capturador.MANIFIESTO_VISTAS
            if vista.clave in {"rrhh-cronos", "rrhh-dietas"}
        }
        self.assertEqual(set(modulos), {"rrhh-cronos", "rrhh-dietas"})
        self.assertEqual(modulos["rrhh-cronos"].ruta.rsplit("#", 1)[-1], "cronos")
        self.assertEqual(modulos["rrhh-dietas"].ruta.rsplit("#", 1)[-1], "dietas")
        self.assertEqual(len(capturador.TAMANOS_VISTA), 3)
        for clave, vista in modulos.items():
            with self.subTest(modulo=clave):
                self.assertEqual(vista.superficie, "gestion-rrhh")
                self.assertIn("perfil=administrador", vista.ruta)
                self.assertEqual(len(vista.selectores_menu), 2)

    def test_toda_ruta_privada_usa_presentacion_rrhh(self) -> None:
        for escenario in capturador.MANIFIESTO:
            superficie = capturador.SUPERFICIES[escenario.superficie]
            if not superficie.privada:
                continue
            consulta = parse_qs(urlparse(escenario.ruta).query)
            self.assertEqual(consulta.get("presentacion"), ["rrhh"], escenario.clave)

    def test_flujos_se_distinguen_y_cubren_interacciones_demo(self) -> None:
        self.assertEqual(len(capturador.MANIFIESTO_FLUJOS), 21)
        claves = {flujo.clave for flujo in capturador.MANIFIESTO_FLUJOS}
        self.assertTrue({
            "publico-ficha-convocatoria",
            "aspirante-convocatoria-abierta",
            "aspirante-confirmacion-demo",
            "aspirante-recibo-demo",
            "rrhh-borrador-abierto",
            "rrhh-recibo-demo",
            "rrhh-perfil-tecnico-restringido",
            "aspirante-menu-movil-abierto",
            "rrhh-menu-movil-abierto",
            "rrhh-menu-bolsa-movil-abierto",
        }.issubset(claves))
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
        self.assertIn("DEMO-PERFIL-TECNICO-RRHH-01", tecnico.pasos[0].selector)
        self.assertEqual(
            [paso.accion for paso in tecnico.pasos],
            ["esperar", "esperar-habilitado", "esperar-deshabilitado"],
        )

    def test_operaciones_rrhh_representativas_exigen_perfil_y_recibo_demo(self) -> None:
        self.assertEqual(len(capturador.FLUJOS_RRHH_CON_RECIBO), 11)
        vistas = {flujo.ruta.rsplit("/", 1)[-1] for flujo in capturador.FLUJOS_RRHH_CON_RECIBO}
        self.assertEqual(vistas, {
            "convocatorias", "solicitudes", "meritos", "baremacion", "importacion",
            "llamamientos", "contratos", "comunicaciones", "estadisticas",
            "configuracion", "alegaciones",
        })
        for flujo in capturador.FLUJOS_RRHH_CON_RECIBO:
            with self.subTest(flujo=flujo.clave):
                self.assertTrue(flujo.requiere_demo)
                self.assertIn("perfil=administrador", flujo.ruta)
                if flujo.clave == "rrhh-llamamiento-recibo-demo":
                    self.assertEqual(
                        [paso.accion for paso in flujo.pasos],
                        ["clic", "esperar", "clic", "esperar", "clic", "clic-confirmando", "esperar", "esperar"],
                    )
                    self.assertEqual(flujo.pasos[0].selector, '[data-accion="solicitar-propuesta"]')
                    self.assertEqual(flujo.pasos[5].selector, '[data-accion="preparar-llamamiento-demo"]')
                    self.assertEqual(flujo.pasos[6].texto_esperado, "DEMO-REC")
                    self.assertEqual(flujo.pasos[7].texto_esperado, "DEMO-LLA-045")
                else:
                    self.assertEqual(
                        [paso.accion for paso in flujo.pasos],
                        ["clic-confirmando", "esperar", "esperar"],
                    )
                    self.assertIn('[data-accion="operacion-presentacion"]', flujo.pasos[0].selector)
                    self.assertEqual(flujo.pasos[1].texto_esperado, "DEMO-REC")
                    self.assertRegex(flujo.pasos[2].texto_esperado, r"^DEMO-")

    def test_flujos_de_menu_dejan_capturable_el_estado_abierto(self) -> None:
        por_clave = {flujo.clave: flujo for flujo in capturador.MANIFIESTO_FLUJOS}
        aspirante = por_clave["aspirante-menu-movil-abierto"]
        rrhh = por_clave["rrhh-menu-movil-abierto"]
        bolsa = por_clave["rrhh-menu-bolsa-movil-abierto"]
        self.assertEqual(aspirante.pasos[0].accion, "abrir-menu")
        self.assertEqual(rrhh.pasos[0].accion, "abrir-menu")
        self.assertEqual(bolsa.pasos[0].accion, "abrir-menu")
        self.assertEqual(bolsa.pasos[1].accion, "abrir-menu")
        self.assertEqual(bolsa.pasos[1].selector, '[data-grupo-bolsa="auditoria"]')
        self.assertEqual(bolsa.pasos[2].selector, "#submenu-auditoria")
        self.assertIn("perfil=administrador", rrhh.ruta)
        self.assertIn("perfil=administrador", bolsa.ruta)
        self.assertIn("#bolsa/resumen", bolsa.ruta)

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
        self.assertEqual(capturador.normalizar_url_base("http://[::1]:8081/"), "http://[::1]:8081")
        for invalida in (
            "", "127.0.0.1:8081", "ftp://127.0.0.1", "http://u:p@127.0.0.1",
            "http://127.0.0.1/?x=1", "http://localhost:8081", "http://0.0.0.0:8081",
            "http://192.168.1.10:8081", "http://8.8.8.8", "http://127.0.0.1:invalido",
        ):
            with self.subTest(invalida=invalida), self.assertRaises(ValueError):
                capturador.normalizar_url_base(invalida)

    def test_cabecera_presentacion_debe_coincidir_exactamente(self) -> None:
        self.assertTrue(capturador.cabecera_presentacion_valida({
            "x-vec-modo-presentacion": capturador.VALOR_MODO_PRESENTACION,
        }))
        for cabeceras in ({},
                          {capturador.CABECERA_MODO_PRESENTACION: "demo"},
                          {"X-VEC-Otra": capturador.VALOR_MODO_PRESENTACION}):
            self.assertFalse(capturador.cabecera_presentacion_valida(cabeceras))

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
