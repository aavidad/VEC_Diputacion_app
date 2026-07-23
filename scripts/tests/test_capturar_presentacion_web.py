from __future__ import annotations

import ast
import unittest
from collections import Counter
from dataclasses import replace
from pathlib import Path
from urllib.parse import parse_qs, urlparse

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
    def test_cubre_todas_las_vistas_y_tamanos_acordados(self) -> None:
        por_superficie = Counter(vista.superficie for vista in capturador.MANIFIESTO_VISTAS)
        self.assertEqual(
            por_superficie,
            {
                "lanzador": 1,
                "portal-publico": 1,
                "area-aspirante": 14,
                "gestion-rrhh": 20,
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
        self.assertIn("reglas", rutas)
        self.assertIn("consulta", rutas)

    def test_cronos_y_dietas_se_auditan_como_modulos_internos_en_tres_tamanos(self) -> None:
        modulos = {
            vista.clave: vista
            for vista in capturador.MANIFIESTO_VISTAS
            if vista.clave in {"rrhh-cronos", "rrhh-dietas"}
        }
        self.assertEqual(set(modulos), {"rrhh-cronos", "rrhh-dietas"})
        self.assertTrue(all("perfil=funcionario" in vista.ruta for vista in modulos.values()))
        self.assertEqual(modulos["rrhh-cronos"].ruta.rsplit("#", 1)[-1], "cronos")
        self.assertEqual(modulos["rrhh-dietas"].ruta.rsplit("#", 1)[-1], "dietas")
        self.assertEqual(len(capturador.TAMANOS_VISTA), 3)
        for clave, vista in modulos.items():
            with self.subTest(modulo=clave):
                self.assertEqual(vista.superficie, "gestion-rrhh")
                self.assertIn("perfil=funcionario", vista.ruta)
                self.assertEqual(len(vista.selectores_menu), 2)

    def test_toda_ruta_privada_usa_presentacion_rrhh(self) -> None:
        for escenario in capturador.MANIFIESTO:
            superficie = capturador.SUPERFICIES[escenario.superficie]
            if not superficie.privada:
                continue
            consulta = parse_qs(urlparse(escenario.ruta).query)
            self.assertEqual(consulta.get("presentacion"), ["rrhh"], escenario.clave)

    def test_flujos_se_distinguen_y_cubren_interacciones_demo(self) -> None:
        self.assertEqual(len(capturador.MANIFIESTO_FLUJOS), 25)
        claves = {flujo.clave for flujo in capturador.MANIFIESTO_FLUJOS}
        self.assertTrue({
            "publico-ficha-convocatoria",
            "aspirante-convocatoria-abierta",
            "aspirante-confirmacion-demo",
            "aspirante-recibo-demo",
            "rrhh-borrador-abierto",
            "rrhh-recibo-demo",
            "rrhh-dietas-ruta-real",
            "rrhh-perfil-tecnico-restringido",
            "funcionario-autoservicio-restringido",
            "administrador-selector-perfiles-abierto",
            "aspirante-selector-perfiles-abierto",
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
            [
                "esperar", "esperar-habilitado", "esperar-deshabilitado",
                "esperar-deshabilitado", "esperar-deshabilitado",
            ],
        )

    def test_selector_cubre_cuatro_rutas_desde_admin_y_aspirante_en_tres_tamanos(self) -> None:
        por_clave = {flujo.clave: flujo for flujo in capturador.MANIFIESTO_FLUJOS}
        selectores = {
            clave: por_clave[clave]
            for clave in (
                "administrador-selector-perfiles-abierto",
                "aspirante-selector-perfiles-abierto",
            )
        }
        perfiles_esperados = {"usuario_externo", "funcionario", "tecnico", "administrador"}
        rutas_esperadas = {
            "/area-personal/?presentacion=rrhh&vista=inicio",
            "/portal-empleado/?presentacion=rrhh&perfil=funcionario#portal",
            "/portal-empleado/?presentacion=rrhh&perfil=tecnico#portal",
            "/portal-empleado/?presentacion=rrhh&perfil=administrador#portal",
        }
        for clave, flujo in selectores.items():
            with self.subTest(flujo=clave):
                self.assertTrue(flujo.requiere_demo)
                self.assertEqual(len(flujo.pasos), 6)
                self.assertEqual(flujo.pasos[0].accion, "clic")
                self.assertIn("data-selector-perfil", flujo.pasos[0].selector)
                self.assertEqual(flujo.pasos[1].accion, "esperar")
                self.assertIn(":not([hidden])", flujo.pasos[1].selector)
                controles = flujo.pasos[2:]
                perfiles = {
                    selector.split('data-perfil-presentacion="', 1)[1].split('"', 1)[0]
                    for selector in (paso.selector for paso in controles)
                }
                rutas = {
                    selector.split('href="', 1)[1].rsplit('"]', 1)[0]
                    for selector in (paso.selector for paso in controles)
                }
                self.assertEqual(perfiles, perfiles_esperados)
                self.assertEqual(rutas, rutas_esperadas)
        self.assertEqual(
            len(selectores) * len(capturador.TAMANOS_VISTA),
            6,
            "los dos selectores se capturan en escritorio, portátil y móvil",
        )

    def test_funcionario_y_tecnico_aplican_minimo_privilegio_en_vista_directa(self) -> None:
        por_clave = {flujo.clave: flujo for flujo in capturador.MANIFIESTO_FLUJOS}
        funcionario = por_clave["funcionario-autoservicio-restringido"]
        tecnico = por_clave["rrhh-perfil-tecnico-restringido"]
        self.assertIn("perfil=funcionario", funcionario.ruta)
        self.assertIn("DEMO-PERFIL-FUNCIONARIO-01", funcionario.pasos[0].selector)
        self.assertEqual(
            [(paso.accion, paso.selector) for paso in funcionario.pasos[1:]],
            [
                ("esperar-deshabilitado", '[data-modulo-portal="bolsa"]'),
                ("esperar-habilitado", '[data-modulo-portal="cronos"][data-vista="cronos"]'),
                ("esperar-habilitado", '[data-modulo-portal="dietas"][data-vista="dietas"]'),
            ],
        )
        self.assertIn("perfil=tecnico", tecnico.ruta)
        self.assertIn(
            ("esperar-habilitado", '[data-modulo-portal="bolsa"][data-vista="resumen"]'),
            [(paso.accion, paso.selector) for paso in tecnico.pasos],
        )
        self.assertTrue(all(
            paso.accion == "esperar-deshabilitado"
            for paso in tecnico.pasos[2:]
        ))

    def test_operaciones_rrhh_representativas_exigen_perfil_y_recibo_demo(self) -> None:
        self.assertEqual(len(capturador.FLUJOS_RRHH_CON_RECIBO), 11)
        vistas = {flujo.ruta.rsplit("/", 1)[-1] for flujo in capturador.FLUJOS_RRHH_CON_RECIBO}
        self.assertEqual(vistas, {
            "convocatorias", "solicitudes", "meritos", "reglas", "importacion",
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


class MatrizPantallasRRHHTests(unittest.TestCase):
    def test_matriz_numera_y_nombra_exactamente_las_diecisiete_pantallas(self) -> None:
        pantallas = capturador.MATRIZ_PANTALLAS_RRHH
        self.assertEqual([pantalla.numero for pantalla in pantallas], list(range(1, 18)))
        self.assertEqual(len({pantalla.clave for pantalla in pantallas}), 17)
        self.assertEqual(len({pantalla.nombre_captura for pantalla in pantallas}), 17)
        self.assertEqual(capturador.validar_matriz_pantallas_rrhh(), [])
        for pantalla in pantallas:
            with self.subTest(pantalla=pantalla.numero):
                self.assertTrue(
                    pantalla.nombre_captura.startswith(f"{pantalla.numero:02d}-"),
                )
                self.assertTrue(pantalla.nombre_captura.endswith(".png"))

    def test_cada_pantalla_declara_contexto_navegacion_asentamiento_y_criterios(self) -> None:
        for pantalla in capturador.MATRIZ_PANTALLAS_RRHH:
            with self.subTest(pantalla=pantalla.numero):
                self.assertEqual(pantalla.ruta, capturador.RUTA_CONTRATACION_RRHH)
                self.assertEqual(pantalla.perfil, "administrador")
                self.assertTrue(pantalla.pasos)
                self.assertTrue(pantalla.selector_asentamiento)
                self.assertTrue(any(
                    pantalla.selector_asentamiento in paso.selector
                    and paso.accion in {"esperar", "enfocar"}
                    for paso in pantalla.pasos
                ))
                self.assertGreaterEqual(
                    len(pantalla.criterios_visuales),
                    len(capturador.CRITERIOS_COMUNES) + 1,
                )
                if pantalla.pestana == "expediente":
                    self.assertEqual(
                        pantalla.expediente_ref,
                        capturador.EXPEDIENTE_RRHH,
                    )
                    self.assertTrue(pantalla.tarea_ref)
                else:
                    self.assertEqual(pantalla.expediente_ref, "")
                    self.assertEqual(pantalla.tarea_ref, "")

    def test_mapeo_de_tareas_respeta_el_flujo_rrhh_y_declara_brechas(self) -> None:
        tareas = {
            pantalla.numero: pantalla.tarea_ref
            for pantalla in capturador.MATRIZ_PANTALLAS_RRHH
        }
        self.assertEqual(tareas, {
            1: "", 2: "",
            3: "tarea-analisis",
            4: "tarea-cobertura",
            5: "tarea-asignacion",
            6: "tarea-informe-juridico",
            7: "tarea-envio-intervencion",
            8: "tarea-fiscalizacion",
            9: "tarea-subsanacion",
            10: "tarea-iniciar-llamamiento",
            11: "tarea-seleccion-candidato",
            12: "tarea-resultado-llamamiento",
            13: "tarea-traslado-intervencion",
            14: "tarea-informe-definitivo",
            15: "tarea-ginpix",
            16: "tarea-ginpix",
            17: "tarea-formalizacion",
        })
        brechas = {
            pantalla.numero for pantalla in capturador.MATRIZ_PANTALLAS_RRHH
            if pantalla.brecha
        }
        bloqueadas = {
            pantalla.numero for pantalla in capturador.MATRIZ_PANTALLAS_RRHH
            if pantalla.paridad == "bloqueada"
        }
        self.assertEqual(brechas, {5, 16})
        self.assertEqual(bloqueadas, set())
        self.assertEqual(
            {pantalla.paridad for pantalla in capturador.MATRIZ_PANTALLAS_RRHH},
            {"pendiente_revision_visual"},
        )

    def test_pantallas_se_capturan_en_1536_1440_y_1280_con_nombre_contractual(self) -> None:
        self.assertEqual(
            {
                (tamano.ancho, tamano.alto)
                for tamano in capturador.TAMANOS_PANTALLAS_RRHH
            },
            {(1536, 1024), (1440, 1000), (1280, 900)},
        )
        self.assertEqual(len(capturador.MANIFIESTO_PANTALLAS_RRHH), 17)
        for pantalla, escenario in zip(
            capturador.MATRIZ_PANTALLAS_RRHH,
            capturador.MANIFIESTO_PANTALLAS_RRHH,
            strict=True,
        ):
            with self.subTest(pantalla=pantalla.numero):
                self.assertEqual(escenario.tipo, "pantalla-rrhh")
                self.assertEqual(escenario.clave, pantalla.clave)
                self.assertTrue(escenario.requiere_demo)
                self.assertEqual(escenario.pasos[-1].accion, "asentar-arriba")
                self.assertEqual(
                    escenario.pasos[-1].selector,
                    '[data-modulo="contratacion-temporal"]',
                )
                self.assertEqual(
                    capturador.tamanos_para_pantalla(escenario.clave),
                    capturador.TAMANOS_PANTALLAS_RRHH,
                )
                self.assertEqual(
                    capturador.nombre_captura_pantalla(escenario.clave),
                    pantalla.nombre_captura,
                )

    def test_estado_ginpix_se_prepara_por_operacion_demo_exacta(self) -> None:
        pantalla = capturador.MATRIZ_PANTALLAS_RRHH[15]
        confirmaciones = [
            paso for paso in pantalla.pasos
            if paso.accion == "clic-confirmando"
        ]
        self.assertEqual(len(confirmaciones), 1)
        self.assertEqual(
            confirmaciones[0].selector,
            '[data-ct-exp-efecto="enviar_ginpix"]',
        )
        self.assertEqual(pantalla.selector_asentamiento, "[data-ct-exp-recibo]")
        codigo = Path(auditoria_revision.__file__).read_text(encoding="utf-8")
        self.assertIn('flujo.clave == "rrhh-pantalla-16-ginpix-recibo"', codigo)
        self.assertIn('[data-ct-exp-efecto="enviar_ginpix"]', codigo)

    def test_validador_detecta_validacion_visual_con_brecha(self) -> None:
        original = capturador.MATRIZ_PANTALLAS_RRHH[4]
        invalida = replace(original, paridad="validada")
        errores = capturador.validar_matriz_pantallas_rrhh((
            *capturador.MATRIZ_PANTALLAS_RRHH[:4],
            invalida,
            *capturador.MATRIZ_PANTALLAS_RRHH[5:],
        ))
        self.assertIn("pantalla validada con brecha/bloqueo en 5", errores)


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
