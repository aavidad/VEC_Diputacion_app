from __future__ import annotations

import ast
import unittest
from dataclasses import replace
from pathlib import Path

from scripts import capturar_presentacion_web as capturador
from scripts.revision_web import auditoria as auditoria_revision


class _PaginaAuditoria:
    def __init__(self, resultado: dict) -> None:
        self.resultado = resultado

    def evaluate(self, _codigo: str) -> dict:
        return self.resultado


class _ContextoAuditoria:
    @staticmethod
    def cookies() -> list[dict]:
        return []


class AuditoriaColumnasOperativasTests(unittest.TestCase):
    @staticmethod
    def resultado(correcta: bool) -> dict:
        tabla = {
            "selector": "div.tabla-contenedor--prioritaria",
            "prioridad": "estado-acciones",
            "faltantes": [],
            "recortadas": [] if correcta else [{"columna": "acciones", "derecha": 1448}],
            "sin_fijar": [],
            "controles_recortados": [],
            "contenidos_estado_recortados": [],
            "filas_sin_columnas": [],
            "solapadas": [],
            "correcta": correcta,
        }
        return {
            "ids_duplicados": [],
            "controles_sin_nombre": [],
            "desbordamiento_horizontal": {
                "existe": False, "ancho_cliente": 1440, "ancho_documento": 1440, "elementos": [],
            },
            "columnas_operativas": [tabla],
            "almacenamiento": {
                "local": [], "sesion": [], "indexeddb": [], "cache": [], "cookie_documento": "",
            },
        }

    def test_falla_si_estado_o_acciones_quedan_recortados(self) -> None:
        _auditoria, hallazgos = auditoria_revision.auditar_dom_y_estado(
            _PaginaAuditoria(self.resultado(False)), _ContextoAuditoria(),
        )
        self.assertIn("columnas_operativas_recortadas", {item["codigo"] for item in hallazgos})

    def test_acepta_columnas_fijas_completamente_operables(self) -> None:
        _auditoria, hallazgos = auditoria_revision.auditar_dom_y_estado(
            _PaginaAuditoria(self.resultado(True)), _ContextoAuditoria(),
        )
        self.assertNotIn("columnas_operativas_recortadas", {item["codigo"] for item in hallazgos})

    def test_falla_si_un_chip_de_estado_invade_otra_columna(self) -> None:
        resultado = self.resultado(True)
        resultado["columnas_operativas"][0]["contenidos_estado_recortados"] = [
            {"nombre": "Pendiente de subsanación", "izquierda": 1018, "derecha": 1218},
        ]
        resultado["columnas_operativas"][0]["correcta"] = False
        _auditoria, hallazgos = auditoria_revision.auditar_dom_y_estado(
            _PaginaAuditoria(resultado), _ContextoAuditoria(),
        )
        self.assertIn("columnas_operativas_recortadas", {item["codigo"] for item in hallazgos})

    def test_falla_si_una_fila_no_tiene_estado_o_acciones(self) -> None:
        resultado = self.resultado(True)
        resultado["columnas_operativas"][0]["filas_sin_columnas"] = [
            {"fila": 2, "columnas_faltantes": ["acciones"]},
        ]
        resultado["columnas_operativas"][0]["correcta"] = False
        _auditoria, hallazgos = auditoria_revision.auditar_dom_y_estado(
            _PaginaAuditoria(resultado), _ContextoAuditoria(),
        )
        self.assertIn("columnas_operativas_recortadas", {item["codigo"] for item in hallazgos})

    def test_la_medicion_dom_comprueba_geometria_controles_y_posicion_fija(self) -> None:
        codigo = auditoria_revision.AUDITORIA_DOM_JS
        for marcador in (
            "data-tabla-prioritaria", "getBoundingClientRect", "controlesRecortados",
            'position !== "sticky"', "contenidosEstadoRecortados", "filasSinColumnas",
            "scrollWidth", "solapadas", "anchoCliente >= 1024",
        ):
            self.assertIn(marcador, codigo)


class ManifiestoRevisionWebTests(unittest.TestCase):
    def test_campana_solo_cubre_consulta_publica_servida(self) -> None:
        self.assertEqual(set(capturador.SUPERFICIES), {"portal-publico"})
        self.assertEqual(
            [(escenario.clave, escenario.ruta) for escenario in capturador.MANIFIESTO],
            [
                ("publico-convocatorias", "/bolsa/"),
                ("publico-ficha-convocatoria", "/bolsa/"),
            ],
        )
        self.assertEqual(len(capturador.MANIFIESTO_VISTAS), 1)
        self.assertEqual(len(capturador.MANIFIESTO_FLUJOS), 1)
        self.assertFalse(any(escenario.requiere_demo for escenario in capturador.MANIFIESTO_FLUJOS))
        self.assertEqual(
            {(tamano.ancho, tamano.alto) for tamano in capturador.TAMANOS_VISTA},
            {(1440, 1000), (1024, 900), (390, 844)},
        )
        self.assertEqual(capturador.validar_manifiesto(), [])

    def test_ficha_publica_requiere_detalle_visible(self) -> None:
        flujo = capturador.MANIFIESTO_FLUJOS[0]
        self.assertEqual(flujo.tipo, "flujo")
        self.assertEqual([paso.accion for paso in flujo.pasos], ["clic", "esperar", "enfocar"])
        self.assertIn("#contenido-detalle:not([hidden])", flujo.pasos[1].selector)

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
        self.assertEqual(
            capturador.normalizar_url_base(
                "http://192.168.255.194:8080", permitir_red_privada=True,
            ),
            "http://192.168.255.194:8080",
        )
        with self.assertRaises(ValueError):
            capturador.normalizar_url_base("http://192.168.255.194:8080")
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
            "superficie": "portal-publico",
            "nombre_superficie": "Portal público",
            "ruta": "/",
            "url": "http://localhost/",
            "tamano": {"clave": "movil", "nombre": "Móvil", "ancho": 390, "alto": 844},
            "captura": "capturas/movil/vista/portal-publico/uno.png",
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
