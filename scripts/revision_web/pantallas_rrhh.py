"""Matriz contractual de las 17 pantallas solicitadas por RRHH.

La matriz no contiene HTML ni construye estados de negocio. Describe cómo
alcanzar, asentar y revisar cada pantalla sobre la superficie definitiva,
usando únicamente controles públicos de la interfaz y un contexto de
presentación aislado.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Sequence
from urllib.parse import parse_qs, urlparse

from .modelo import Flujo, PasoInteraccion, TamanoVista


RUTA_CONTRATACION_RRHH = (
    "/portal-empleado/?presentacion=rrhh&perfil=administrador"
    "#contratacion-temporal"
)
PERFIL_RRHH = "administrador"
EXPEDIENTE_RRHH = "exp-demo-contratacion-005487"

TAMANOS_PANTALLAS_RRHH: tuple[TamanoVista, ...] = (
    TamanoVista("rrhh-1536", "RRHH referencia 1536", 1536, 1024),
    TamanoVista("rrhh-1440", "RRHH escritorio 1440", 1440, 1000),
    TamanoVista("rrhh-1280", "RRHH escritorio 1280", 1280, 900),
)

CRITERIOS_COMUNES = (
    "Conserva el shell, la navegación lateral y los tokens del tema común.",
    "Muestra fase, estado y acción mediante texto; el color nunca es la única señal.",
    "No recorta ni solapa títulos, formularios, tablas, estados o acciones.",
    "Mantiene visible la marca inequívoca de presentación con datos sintéticos.",
)


@dataclass(frozen=True, slots=True)
class PantallaRRHH:
    """Contrato reproducible de una pantalla numerada del documento de RRHH."""

    numero: int
    clave: str
    nombre: str
    ruta: str
    perfil: str
    expediente_ref: str
    tarea_ref: str
    pestana: str
    pasos: tuple[PasoInteraccion, ...]
    selector_asentamiento: str
    nombre_captura: str
    criterios_visuales: tuple[str, ...]
    paridad: str = "pendiente_revision_visual"
    brecha: str = ""
    bloqueo: str = ""

    def como_flujo(self) -> Flujo:
        """Adapta la definición al motor general de Playwright."""
        return Flujo(
            clave=self.clave,
            nombre=f"{self.numero:02d}. {self.nombre}",
            superficie="gestion-rrhh",
            ruta=self.ruta,
            selector_titulo="#titulo-vista",
            titulo_esperado="Gestión de expedientes de contratación temporal",
            selectores_listos=(
                ".aviso-presentacion:not([hidden])",
                '[data-modulo="contratacion-temporal"]',
            ),
            pasos=(
                *self.pasos,
                PasoInteraccion(
                    "asentar-arriba",
                    '[data-modulo="contratacion-temporal"]',
                ),
            ),
            selector_menu_actual='[data-vista="contratacion-temporal"]',
            requiere_demo=True,
            tipo="pantalla-rrhh",
        )


def _pasos_cuadro() -> tuple[PasoInteraccion, ...]:
    return (
        PasoInteraccion("esperar", '[data-modulo="contratacion-temporal"]'),
        PasoInteraccion("esperar", '[data-ct-exp-vista="cuadro"][aria-current="page"]'),
        PasoInteraccion("esperar", "[data-ct-exp-filtros]"),
        PasoInteraccion("esperar", ".ct-exp-listado .tabla-datos", "2026/CT-05487"),
        PasoInteraccion("enfocar", ".ct-exp-listado"),
    )


def _pasos_alta() -> tuple[PasoInteraccion, ...]:
    return (
        PasoInteraccion("esperar", '[data-modulo="contratacion-temporal"]'),
        PasoInteraccion("clic", '[data-ct-exp-vista="alta"]'),
        PasoInteraccion("esperar", '[data-ct-exp-vista="alta"][aria-current="page"]'),
        PasoInteraccion("esperar", "[data-ct-form]"),
        PasoInteraccion("esperar", "#ct-alta-titulo", "Nueva solicitud"),
        PasoInteraccion("enfocar", "[data-ct-form]"),
    )


def _pasos_tarea(
    tarea_ref: str,
    titulo: str,
    *pasos_finales: PasoInteraccion,
) -> tuple[PasoInteraccion, ...]:
    return (
        PasoInteraccion("esperar", '[data-modulo="contratacion-temporal"]'),
        PasoInteraccion(
            "clic",
            f'[data-ct-exp-abrir="{EXPEDIENTE_RRHH}"]',
        ),
        PasoInteraccion("esperar", "#ct-exp-tarea-titulo"),
        PasoInteraccion("clic", f'[data-ct-exp-tarea="{tarea_ref}"]'),
        PasoInteraccion(
            "esperar",
            f'[data-ct-exp-tarea="{tarea_ref}"][aria-current="step"]',
        ),
        PasoInteraccion("esperar", "#ct-exp-tarea-titulo", titulo),
        *pasos_finales,
    )


def _criterios(*propios: str) -> tuple[str, ...]:
    return (*CRITERIOS_COMUNES, *propios)


MATRIZ_PANTALLAS_RRHH: tuple[PantallaRRHH, ...] = (
    PantallaRRHH(
        1, "rrhh-pantalla-01-cuadro-mando", "Inicio y cuadro de mando",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, "", "", "cuadro",
        _pasos_cuadro(), ".ct-exp-listado",
        "01-inicio-cuadro-mando.png",
        _criterios(
            "Presenta indicadores compactos, filtros y expedientes accionables en el primer recorrido.",
            "Expone número, centro, categoría, modalidad, estado, fase, plazo y acción de apertura.",
        ),
    ),
    PantallaRRHH(
        2, "rrhh-pantalla-02-nueva-peticion", "Nueva petición de personal",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, "", "", "alta",
        _pasos_alta(), "[data-ct-form]",
        "02-nueva-peticion-personal.png",
        _criterios(
            "Separa datos del centro, necesidad, periodo, RC y documentación con etiquetas visibles.",
            "Distingue edición, revisión y recibo; no presenta el envío como acto ya registrado.",
        ),
    ),
    PantallaRRHH(
        3, "rrhh-pantalla-03-analisis", "Análisis de RRHH",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-analisis", "expediente",
        _pasos_tarea(
            "tarea-analisis", "Análisis de RRHH",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-analisis"][aria-current="step"]',
        "03-analisis-rrhh.png",
        _criterios(
            "Agrupa modalidad, causa, duración, categoría, jornada, coste y RC con su fuente.",
            "La validación de RRHH conserva unidad, responsable, versión y referencias probatorias.",
        ),
    ),
    PantallaRRHH(
        4, "rrhh-pantalla-04-cobertura", "Gestión de bolsa y vía de cobertura",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-cobertura", "expediente",
        _pasos_tarea(
            "tarea-cobertura", "Comprobaciones y vía de cobertura",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-cobertura"][aria-current="step"]',
        "04-gestion-bolsa-cobertura.png",
        _criterios(
            "Hace visibles las comprobaciones automáticas, su fuente y la vía resultante.",
            "No confunde una propuesta automática con la decisión humana confirmada.",
        ),
    ),
    PantallaRRHH(
        5, "rrhh-pantalla-05-bandeja-unidad", "Bandeja y asignación de la unidad",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-asignacion", "expediente",
        _pasos_tarea(
            "tarea-asignacion", "Asignación a unidad",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-asignacion"][aria-current="step"]',
        "05-bandeja-asignacion-unidad.png",
        _criterios(
            "Identifica unidad, responsable, entrada en bandeja y notificación de asignación.",
            "Mantiene contexto del expediente y acceso a todas sus tareas.",
        ),
        brecha=(
            "La asignación es navegable y trazable, pero aún no existe una bandeja "
            "independiente de la unidad con su listado propio como en la referencia."
        ),
    ),
    PantallaRRHH(
        6, "rrhh-pantalla-06-informe-juridico", "Informe jurídico de la unidad",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-informe-juridico", "expediente",
        _pasos_tarea(
            "tarea-informe-juridico", "Informe jurídico",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-informe-juridico"][aria-current="step"]',
        "06-informe-juridico-unidad.png",
        _criterios(
            "Muestra datos minimizados, plantilla/versión y evidencia de generación.",
            "Separa guardar, generar, revisar y enviar a firma.",
        ),
    ),
    PantallaRRHH(
        7, "rrhh-pantalla-07-firma-envio", "Firma de jefatura y envío a Intervención",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-envio-intervencion", "expediente",
        _pasos_tarea(
            "tarea-envio-intervencion", "Firma y envío a Intervención",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-envio-intervencion"][aria-current="step"]',
        "07-firma-jefatura-envio-intervencion.png",
        _criterios(
            "Distingue documento generado, firma pendiente/completada y traslado.",
            "Expone índice/documentos que se remiten sin afirmar una firma productiva.",
        ),
    ),
    PantallaRRHH(
        8, "rrhh-pantalla-08-fiscalizacion", "Fiscalización por Intervención",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-fiscalizacion", "expediente",
        _pasos_tarea(
            "tarea-fiscalizacion", "Fiscalización",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-fiscalizacion"][aria-current="step"]',
        "08-fiscalizacion-intervencion.png",
        _criterios(
            "Mantiene segregado el resultado de Intervención de la unidad proponente.",
            "Muestra resultado, observaciones/reparo, recibo e histórico sin sobrescritura.",
        ),
    ),
    PantallaRRHH(
        9, "rrhh-pantalla-09-subsanacion", "Subsanación de reparos",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-subsanacion", "expediente",
        _pasos_tarea(
            "tarea-subsanacion", "Subsanación de reparos",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-subsanacion"][aria-current="step"]',
        "09-subsanacion-reparos.png",
        _criterios(
            "Conserva el reparo original, la respuesta, la documentación y el nuevo envío.",
            "Cuando no procede subsanar, lo declara expresamente y no inventa un reparo.",
        ),
    ),
    PantallaRRHH(
        10, "rrhh-pantalla-10-inicio-llamamiento", "Inicio del llamamiento",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-iniciar-llamamiento", "expediente",
        _pasos_tarea(
            "tarea-iniciar-llamamiento", "Inicio del llamamiento",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-iniciar-llamamiento"][aria-current="step"]',
        "10-inicio-llamamiento.png",
        _criterios(
            "Identifica bolsa, regla/version, orden y universo de candidaturas.",
            "No permite saltarse el orden mediante un dato controlado por el navegador.",
        ),
    ),
    PantallaRRHH(
        11, "rrhh-pantalla-11-seleccion", "Selección de candidatura",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-seleccion-candidato", "expediente",
        _pasos_tarea(
            "tarea-seleccion-candidato", "Selección de candidatura",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-seleccion-candidato"][aria-current="step"]',
        "11-seleccion-candidatura.png",
        _criterios(
            "Presenta orden, estado, situación, puntuación y acción sin ocultar el fundamento.",
            "Minimiza la identidad y deja visibles exclusiones o renuncias con señal textual.",
        ),
    ),
    PantallaRRHH(
        12, "rrhh-pantalla-12-resultado", "Resultado del llamamiento",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-resultado-llamamiento", "expediente",
        _pasos_tarea(
            "tarea-resultado-llamamiento", "Resultado del llamamiento",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-resultado-llamamiento"][aria-current="step"]',
        "12-resultado-llamamiento.png",
        _criterios(
            "Distingue aceptación, renuncia, ausencia, rechazo y falta de requisitos.",
            "Liga resultado, observaciones, evidencia y acta al llamamiento exacto.",
        ),
    ),
    PantallaRRHH(
        13, "rrhh-pantalla-13-traslado", "Preparación y traslado a Intervención",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-traslado-intervencion", "expediente",
        _pasos_tarea(
            "tarea-traslado-intervencion", "Traslado de candidatura",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-traslado-intervencion"][aria-current="step"]',
        "13-documentacion-traslado-intervencion.png",
        _criterios(
            "Resume expediente, candidatura, aceptación e índice documental remitido.",
            "Distingue generado, firmado y pendiente; no convierte descarga en autorización.",
        ),
    ),
    PantallaRRHH(
        14, "rrhh-pantalla-14-informe-definitivo", "Informe definitivo para formalización",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-informe-definitivo", "expediente",
        _pasos_tarea(
            "tarea-informe-definitivo", "Informe definitivo",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-informe-definitivo"][aria-current="step"]',
        "14-informe-definitivo-formalizacion.png",
        _criterios(
            "Presenta datos, formato, plantilla, versión, vista previa y circuito posterior.",
            "Separa informe, resolución y toma de posesión como piezas versionadas.",
        ),
    ),
    PantallaRRHH(
        15, "rrhh-pantalla-15-ginpix-previo", "Preparación de datos para GINPIX",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-ginpix", "expediente",
        _pasos_tarea(
            "tarea-ginpix", "Integración con GINPIX",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-ginpix"][aria-current="step"]',
        "15-preparacion-datos-ginpix.png",
        _criterios(
            "Muestra una proyección canónica validada, común a adaptadores API y fichero.",
            "Marca el estado de cada grupo de datos y no muestra datos personales reales.",
        ),
    ),
    PantallaRRHH(
        16, "rrhh-pantalla-16-ginpix-recibo", "Resumen y recibo de envío GINPIX",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-ginpix", "expediente",
        _pasos_tarea(
            "tarea-ginpix", "Integración con GINPIX",
            PasoInteraccion(
                "clic-confirmando",
                '[data-ct-exp-efecto="enviar_ginpix"]',
            ),
            PasoInteraccion("esperar", "[data-ct-exp-recibo]", "Enviar a GINPIX"),
            PasoInteraccion("enfocar", "[data-ct-exp-recibo]"),
        ),
        "[data-ct-exp-recibo]",
        "16-resumen-recibo-ginpix.png",
        _criterios(
            "Conserva resumen, documentos/validaciones y recibo correlacionado de la operación.",
            "Declara que el recibo es sintético y que no hubo transmisión real a GINPIX.",
        ),
        brecha=(
            "El estado posterior se alcanza mediante el adaptador DEMO autorizado en un "
            "contexto limpio; aún no existe una pantalla final independiente ni acredita "
            "idempotencia, transmisión, acuse o conciliación productivos."
        ),
    ),
    PantallaRRHH(
        17, "rrhh-pantalla-17-formalizacion", "Generación documental para formalización",
        RUTA_CONTRATACION_RRHH, PERFIL_RRHH, EXPEDIENTE_RRHH,
        "tarea-formalizacion", "expediente",
        _pasos_tarea(
            "tarea-formalizacion", "Formalización y firmas",
            PasoInteraccion("enfocar", ".ct-exp-tarea-actual"),
        ),
        '[data-ct-exp-tarea="tarea-formalizacion"][aria-current="step"]',
        "17-documentos-formalizacion.png",
        _criterios(
            "Lista piezas, orden, versión, estado, firma y acción documental por separado.",
            "Hace visibles pendientes, firmantes y siguiente paso sin aparentar una firma real.",
        ),
    ),
)

MANIFIESTO_PANTALLAS_RRHH: tuple[Flujo, ...] = tuple(
    pantalla.como_flujo() for pantalla in MATRIZ_PANTALLAS_RRHH
)
PANTALLAS_RRHH_POR_CLAVE = {
    pantalla.clave: pantalla for pantalla in MATRIZ_PANTALLAS_RRHH
}


def tamanos_para_pantalla(clave_escenario: str) -> tuple[TamanoVista, ...] | None:
    """Devuelve los dos tamaños contractuales o ``None`` para otro escenario."""
    if clave_escenario in PANTALLAS_RRHH_POR_CLAVE:
        return TAMANOS_PANTALLAS_RRHH
    return None


def nombre_captura_pantalla(clave_escenario: str) -> str | None:
    pantalla = PANTALLAS_RRHH_POR_CLAVE.get(clave_escenario)
    return pantalla.nombre_captura if pantalla else None


def metadatos_pantalla(clave_escenario: str) -> PantallaRRHH | None:
    return PANTALLAS_RRHH_POR_CLAVE.get(clave_escenario)


def validar_matriz_pantallas_rrhh(
    pantallas: Sequence[PantallaRRHH] = MATRIZ_PANTALLAS_RRHH,
) -> list[str]:
    """Comprueba numeración, navegación, capturas y paridad sin abrir navegador."""
    errores: list[str] = []
    numeros = [pantalla.numero for pantalla in pantallas]
    if numeros != list(range(1, 18)):
        errores.append("la matriz RRHH debe contener las pantallas ordenadas del 1 al 17")

    claves: set[str] = set()
    capturas: set[str] = set()
    for pantalla in pantallas:
        if pantalla.clave in claves:
            errores.append(f"clave de pantalla RRHH duplicada: {pantalla.clave}")
        claves.add(pantalla.clave)
        if pantalla.nombre_captura in capturas:
            errores.append(f"captura RRHH duplicada: {pantalla.nombre_captura}")
        capturas.add(pantalla.nombre_captura)

        if not re.fullmatch(rf"{pantalla.numero:02d}-[a-z0-9-]+\.png", pantalla.nombre_captura):
            errores.append(f"nombre de captura RRHH no determinista: {pantalla.nombre_captura}")
        if pantalla.ruta != RUTA_CONTRATACION_RRHH:
            errores.append(f"ruta RRHH no canónica en pantalla {pantalla.numero}")
        consulta = parse_qs(urlparse(pantalla.ruta).query)
        if consulta.get("perfil") != [pantalla.perfil] or pantalla.perfil != PERFIL_RRHH:
            errores.append(f"perfil RRHH incoherente en pantalla {pantalla.numero}")
        if pantalla.pestana not in {"cuadro", "alta", "expediente"}:
            errores.append(f"pestaña desconocida en pantalla {pantalla.numero}")
        if pantalla.pestana == "expediente":
            if pantalla.expediente_ref != EXPEDIENTE_RRHH or not pantalla.tarea_ref:
                errores.append(f"contexto de expediente incompleto en pantalla {pantalla.numero}")
            selector_tarea = f'[data-ct-exp-tarea="{pantalla.tarea_ref}"]'
            if not any(selector_tarea in paso.selector for paso in pantalla.pasos):
                errores.append(f"la pantalla {pantalla.numero} no selecciona su tarea")
        elif pantalla.expediente_ref or pantalla.tarea_ref:
            errores.append(f"contexto de expediente improcedente en pantalla {pantalla.numero}")

        if not pantalla.pasos or not pantalla.selector_asentamiento:
            errores.append(f"navegación no comprobable en pantalla {pantalla.numero}")
        if not any(
            pantalla.selector_asentamiento in paso.selector
            and paso.accion in {"esperar", "enfocar"}
            for paso in pantalla.pasos
        ):
            errores.append(f"selector de asentamiento no esperado en pantalla {pantalla.numero}")
        if len(pantalla.criterios_visuales) < len(CRITERIOS_COMUNES) + 1:
            errores.append(f"criterios visuales incompletos en pantalla {pantalla.numero}")
        if pantalla.paridad not in {
            "pendiente_revision_visual",
            "validada",
            "bloqueada",
        }:
            errores.append(f"paridad desconocida en pantalla {pantalla.numero}")
        if pantalla.paridad == "validada" and (pantalla.brecha or pantalla.bloqueo):
            errores.append(f"pantalla validada con brecha/bloqueo en {pantalla.numero}")
        if pantalla.paridad == "bloqueada" and not pantalla.bloqueo:
            errores.append(f"pantalla bloqueada sin causa en {pantalla.numero}")
    return errores
