"""Modelo, manifiesto y helpers puros de la revisión web."""

from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass
from ipaddress import ip_address
from pathlib import Path
from typing import Any, Mapping, Sequence
from urllib.parse import parse_qs, urljoin, urlparse


RAIZ_REPOSITORIO = Path(__file__).resolve().parents[2]
SALIDA_PREDETERMINADA = RAIZ_REPOSITORIO / "var" / "revision-web"
CABECERA_MODO_PRESENTACION = "X-VEC-Modo-Presentacion"
VALOR_MODO_PRESENTACION = "aislada-sintetica-v1"


@dataclass(frozen=True, slots=True)
class TamanoVista:
    clave: str
    nombre: str
    ancho: int
    alto: int


@dataclass(frozen=True, slots=True)
class Superficie:
    clave: str
    nombre: str
    selector_contenedor_menu: str | None
    selectores_menu: tuple[str, ...]
    selector_abrir_menu: str | None = None
    selector_cerrar_menu: str | None = None
    selector_banner_demo: str | None = None
    privada: bool = False


@dataclass(frozen=True, slots=True)
class Vista:
    clave: str
    nombre: str
    superficie: str
    ruta: str
    selector_titulo: str
    titulo_esperado: str
    selectores_listos: tuple[str, ...]
    selector_menu_actual: str | None = None
    selectores_menu: tuple[str, ...] = ()
    tipo: str = "vista"


@dataclass(frozen=True, slots=True)
class PasoInteraccion:
    accion: str
    selector: str
    texto_esperado: str = ""


@dataclass(frozen=True, slots=True)
class Flujo:
    clave: str
    nombre: str
    superficie: str
    ruta: str
    selector_titulo: str
    titulo_esperado: str
    selectores_listos: tuple[str, ...]
    pasos: tuple[PasoInteraccion, ...]
    selector_menu_actual: str | None = None
    selectores_menu: tuple[str, ...] = ()
    requiere_demo: bool = False
    tipo: str = "flujo"


Escenario = Vista | Flujo

TAMANOS_VISTA: tuple[TamanoVista, ...] = (
    TamanoVista("escritorio", "Escritorio", 1440, 1000),
    TamanoVista("portatil", "Portátil", 1024, 900),
    TamanoVista("movil", "Móvil", 390, 844),
)

RUTAS_MENU_ASPIRANTE = (
    "inicio", "convocatorias", "perfil", "meritos", "solicitud",
    "autobaremacion", "seguimiento", "llamamientos", "subsanaciones",
    "alegaciones", "mensajes", "certificados", "ayuda",
)
RUTAS_MENU_RRHH = (
    "portal", "resumen", "elaboracion", "convocatorias", "solicitudes",
    "meritos", "baremacion", "alegaciones", "importacion", "llamamientos",
    "contratos", "documentos", "comunicaciones", "estadisticas",
    "auditoria", "configuracion",
)

SUPERFICIES: dict[str, Superficie] = {
    "lanzador": Superficie(
        clave="lanzador",
        nombre="Lanzador",
        selector_contenedor_menu=None,
        selectores_menu=(
            'a[href="/bolsa/"]',
            'a[href^="/area-personal/"]',
            'a[href^="/portal-empleado/"]',
        ),
    ),
    "portal-publico": Superficie(
        clave="portal-publico",
        nombre="Portal público",
        selector_contenedor_menu=".navegacion-publica",
        selectores_menu=(
            '.navegacion-publica a[href="#contenido-principal"]',
            '.navegacion-publica a[href="#filtros-convocatorias"]',
            '.navegacion-publica a[href="#directorio-categorias"]',
            '.navegacion-publica a[href="#ayuda-publica"]',
        ),
    ),
    "area-aspirante": Superficie(
        clave="area-aspirante",
        nombre="Área aspirante",
        selector_contenedor_menu="#navegacion-lateral",
        selectores_menu=tuple(f'[data-ruta="{ruta}"]' for ruta in RUTAS_MENU_ASPIRANTE),
        selector_abrir_menu='[data-accion="alternar-menu"]',
        selector_cerrar_menu='.velo-menu[data-accion="cerrar-menu"]',
        selector_banner_demo="#aviso-presentacion",
        privada=True,
    ),
    "gestion-rrhh": Superficie(
        clave="gestion-rrhh",
        nombre="Gestión interna RRHH",
        selector_contenedor_menu="#navegacion-lateral",
        selectores_menu=tuple(f'[data-vista="{ruta}"]' for ruta in RUTAS_MENU_RRHH),
        selector_abrir_menu="#boton-menu",
        selector_cerrar_menu="#velo-menu",
        selector_banner_demo=".aviso-presentacion",
        privada=True,
    ),
}


def _vistas_aspirante() -> tuple[Vista, ...]:
    definiciones = (
        ("inicio", "Inicio y plazos", "inicio", "Inicio y plazos"),
        ("convocatorias", "Convocatorias", "convocatorias", "Convocatorias"),
        ("convocatoria", "Detalle de convocatoria", "convocatoria", "Detalle de convocatoria"),
        ("perfil", "Perfil y contacto", "perfil", "Perfil y contacto"),
        ("meritos", "Méritos y documentos", "meritos", "Méritos y documentos"),
        ("solicitud", "Nueva solicitud", "solicitud", "Nueva solicitud"),
        ("autobaremacion", "Autobaremación", "autobaremacion", "Autobaremación"),
        ("seguimiento", "Mis expedientes", "seguimiento", "Mis expedientes"),
        ("llamamientos", "Disponibilidad y llamamientos", "llamamientos", "Disponibilidad y llamamientos"),
        ("subsanaciones", "Subsanaciones", "subsanaciones", "Subsanaciones"),
        ("alegaciones", "Alegaciones", "alegaciones", "Alegaciones"),
        ("mensajes", "Mensajes y noticias", "mensajes", "Mensajes y noticias"),
        ("certificados", "Certificados y descargas", "certificados", "Certificados y descargas"),
        ("ayuda", "Ayuda y accesibilidad", "ayuda", "Ayuda y accesibilidad"),
    )
    vistas: list[Vista] = []
    for clave, nombre, ruta, titulo in definiciones:
        id_convocatoria = "&id=DEMO-CONV-001" if clave == "convocatoria" else ""
        menu_actual = "convocatorias" if clave == "convocatoria" else ruta
        vistas.append(Vista(
            clave=f"aspirante-{clave}", nombre=nombre, superficie="area-aspirante",
            ruta=f"/area-personal/?presentacion=rrhh&vista={ruta}{id_convocatoria}",
            selector_titulo="#titulo-vista", titulo_esperado=titulo,
            selectores_listos=("#aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
            selector_menu_actual=f'[data-ruta="{menu_actual}"]',
        ))
    return tuple(vistas)


def _vistas_rrhh() -> tuple[Vista, ...]:
    definiciones = (
        ("portal", "Inicio del portal", "Portal del Empleado"),
        ("resumen", "Cuadro de mando", "Cuadro de mando"),
        ("elaboracion", "Borradores de convocatorias", "Borradores de convocatorias"),
        ("convocatorias", "Convocatorias, bases y calendario", "Convocatorias, bases y calendario"),
        ("solicitudes", "Solicitudes y admisión", "Solicitudes y admisión"),
        ("meritos", "Revisión de méritos", "Revisión de méritos"),
        ("baremacion", "Baremación y ranking", "Baremación y ranking"),
        ("alegaciones", "Alegaciones", "Alegaciones"),
        ("importacion", "Importación Convoca", "Importación Convoca"),
        ("llamamientos", "Llamamientos", "Nuevo llamamiento"),
        ("contratos", "Contratos, ceses y reincorporaciones", "Contratos, ceses y reincorporaciones"),
        ("documentos", "Documentos y firma", "Generación y firma de documentos"),
        ("comunicaciones", "Comunicaciones", "Correo y mensajería"),
        ("estadisticas", "Estadísticas y exportación", "Estadísticas y explotación de datos"),
        ("auditoria", "Auditoría y trazabilidad", "Auditoría y trazabilidad"),
        ("configuracion", "Configuración y roles", "Configuración y roles"),
    )
    vistas: list[Vista] = []
    for ruta, nombre, titulo in definiciones:
        hash_vista = "#portal" if ruta == "portal" else f"#bolsa/{ruta}"
        menu_reducido = ('[data-vista="portal"]', '[data-vista="resumen"]') if ruta == "portal" else ()
        vistas.append(Vista(
            clave=f"rrhh-{ruta}", nombre=nombre, superficie="gestion-rrhh",
            ruta=f"/portal-empleado/?presentacion=rrhh&perfil=administrador{hash_vista}",
            selector_titulo="#titulo-vista", titulo_esperado=titulo,
            selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
            selector_menu_actual=f'[data-vista="{ruta}"]', selectores_menu=menu_reducido,
        ))
    return tuple(vistas)


MANIFIESTO_VISTAS: tuple[Vista, ...] = (
    Vista(
        clave="lanzador-recorrido", nombre="Recorrido de presentación", superficie="lanzador",
        ruta="/presentacion/", selector_titulo="h1",
        titulo_esperado="Recorrido de presentación del portal", selectores_listos=("#contenido-principal",),
    ),
    Vista(
        clave="publico-convocatorias", nombre="Consulta pública", superficie="portal-publico",
        ruta="/bolsa/", selector_titulo="#titulo-portal",
        titulo_esperado="Bolsas y procesos selectivos",
        selectores_listos=('#panel-listado[aria-busy="false"]', '#directorio-categorias[aria-busy="false"]'),
        selector_menu_actual='.navegacion-publica a[href="#contenido-principal"]',
    ),
    *_vistas_aspirante(),
    *_vistas_rrhh(),
)


MANIFIESTO_FLUJOS: tuple[Flujo, ...] = (
    Flujo(
        clave="publico-ficha-convocatoria", nombre="Ficha pública tras consultar",
        superficie="portal-publico", ruta="/bolsa/", selector_titulo="#titulo-portal",
        titulo_esperado="Bolsas y procesos selectivos",
        selectores_listos=('#panel-listado[aria-busy="false"]', '#directorio-categorias[aria-busy="false"]'),
        pasos=(PasoInteraccion("clic", ".enlace-detalle"),
               PasoInteraccion("esperar", "#contenido-detalle:not([hidden])", "Plazos"),
               PasoInteraccion("enfocar", "#panel-detalle")),
        selector_menu_actual='.navegacion-publica a[href="#contenido-principal"]',
    ),
    Flujo(
        clave="aspirante-convocatoria-abierta", nombre="Convocatoria abierta desde el listado",
        superficie="area-aspirante", ruta="/area-personal/?presentacion=rrhh&vista=convocatorias",
        selector_titulo="#titulo-vista", titulo_esperado="Convocatorias",
        selectores_listos=("#aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("clic", '[data-accion="abrir-convocatoria"][data-id="DEMO-CONV-001"]'),
               PasoInteraccion("esperar", "#espacio-trabajo", "Resumen de la convocatoria")),
        selector_menu_actual='[data-ruta="convocatorias"]', requiere_demo=True,
    ),
    Flujo(
        clave="aspirante-confirmacion-demo", nombre="Confirmación de operación DEMO",
        superficie="area-aspirante",
        ruta="/area-personal/?presentacion=rrhh&vista=seguimiento",
        selector_titulo="#titulo-vista", titulo_esperado="Mis expedientes",
        selectores_listos=("#aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("clic", '[data-accion="preparar-operacion"][data-operacion="solicitar_descarga"]'),
               PasoInteraccion("esperar", "#dialogo-confirmacion[open]", "Recibo DEMO")),
        selector_menu_actual='[data-ruta="seguimiento"]', requiere_demo=True,
    ),
    Flujo(
        clave="aspirante-recibo-demo", nombre="Recibo de operación DEMO", superficie="area-aspirante",
        ruta="/area-personal/?presentacion=rrhh&vista=seguimiento",
        selector_titulo="#titulo-vista", titulo_esperado="Mis expedientes",
        selectores_listos=("#aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("clic", '[data-accion="preparar-operacion"][data-operacion="solicitar_descarga"]'),
               PasoInteraccion("esperar", "#dialogo-confirmacion[open]", "Recibo DEMO"),
               PasoInteraccion("clic", '#formulario-confirmacion button[value="confirmar"]'),
               PasoInteraccion("esperar", "#dialogo-recibo[open] .recibo-demo", "DEMO-REC")),
        selector_menu_actual='[data-ruta="seguimiento"]', requiere_demo=True,
    ),
    Flujo(
        clave="rrhh-borrador-abierto", nombre="Borrador RRHH abierto", superficie="gestion-rrhh",
        ruta="/portal-empleado/?presentacion=rrhh&perfil=administrador#bolsa/elaboracion",
        selector_titulo="#titulo-vista", titulo_esperado="Borradores de convocatorias",
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("esperar", '[data-borrador-accion="borradores-abrir"]'),
               PasoInteraccion("clic", '[data-borrador-accion="borradores-abrir"]'),
               PasoInteraccion("esperar", 'form[data-borrador-form="editor"]', "Motivo gobernado"),
               PasoInteraccion("enfocar", 'form[data-borrador-form="editor"]')),
        selector_menu_actual='[data-vista="elaboracion"]', requiere_demo=True,
    ),
    Flujo(
        clave="rrhh-recibo-demo", nombre="Operación RRHH con recibo DEMO", superficie="gestion-rrhh",
        ruta="/portal-empleado/?presentacion=rrhh&perfil=administrador#bolsa/documentos",
        selector_titulo="#titulo-vista", titulo_esperado="Generación y firma de documentos",
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("clic-confirmando", '[data-accion="operacion-presentacion"][data-operacion="generar-documento"]'),
               PasoInteraccion("esperar", "#dialogo-detalle[open] .recibo-presentacion", "DEMO-REC")),
        selector_menu_actual='[data-vista="documentos"]', requiere_demo=True,
    ),
    Flujo(
        clave="rrhh-perfil-tecnico-restringido", nombre="Perfil técnico con permisos restringidos",
        superficie="gestion-rrhh",
        ruta="/portal-empleado/?presentacion=rrhh&perfil=tecnico#portal",
        selector_titulo="#titulo-vista", titulo_esperado="Portal del Empleado",
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("esperar", '#sesion-visible[data-actor-ref="DEMO-PERFIL-TECNICO-RRHH-01"]'),
               PasoInteraccion("esperar-habilitado", '[data-vista="resumen"]'),
               PasoInteraccion("esperar-deshabilitado", '[data-vista="configuracion"]')),
        selector_menu_actual='[data-vista="portal"]',
        selectores_menu=('[data-vista="portal"]', '[data-vista="resumen"]'),
        requiere_demo=True,
    ),
    Flujo(
        clave="aspirante-menu-movil-abierto", nombre="Menú aspirante abierto",
        superficie="area-aspirante",
        ruta="/area-personal/?presentacion=rrhh&vista=inicio",
        selector_titulo="#titulo-vista", titulo_esperado="Inicio y plazos",
        selectores_listos=("#aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("abrir-menu", '[data-accion="alternar-menu"]'),
               PasoInteraccion("esperar", "#navegacion-lateral", "Convocatorias")),
        selector_menu_actual='[data-ruta="inicio"]', requiere_demo=True,
    ),
    Flujo(
        clave="rrhh-menu-movil-abierto", nombre="Menú RRHH abierto",
        superficie="gestion-rrhh",
        ruta="/portal-empleado/?presentacion=rrhh&perfil=administrador#portal",
        selector_titulo="#titulo-vista", titulo_esperado="Portal del Empleado",
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("abrir-menu", "#boton-menu"),
               PasoInteraccion("esperar", "#navegacion-lateral", "Bolsas de trabajo")),
        selector_menu_actual='[data-vista="portal"]',
        selectores_menu=('[data-vista="portal"]', '[data-vista="resumen"]'),
        requiere_demo=True,
    ),
    Flujo(
        clave="rrhh-menu-bolsa-movil-abierto", nombre="Submenú de Bolsa RRHH abierto",
        superficie="gestion-rrhh",
        ruta="/portal-empleado/?presentacion=rrhh&perfil=administrador#bolsa/resumen",
        selector_titulo="#titulo-vista", titulo_esperado="Cuadro de mando",
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(PasoInteraccion("abrir-menu", "#boton-menu"),
               PasoInteraccion("esperar", "#navegacion-lateral", "Configuración y roles")),
        selector_menu_actual='[data-vista="resumen"]', requiere_demo=True,
    ),
)


def _flujo_rrhh_con_recibo(
    clave: str,
    nombre: str,
    vista: str,
    titulo: str,
    operacion: str,
    objetivo: str,
    selector: str | None = None,
) -> Flujo:
    """Declara una operación RRHH representativa y exige su recibo sintético."""
    selector_operacion = selector or (
        f'[data-accion="operacion-presentacion"][data-operacion="{operacion}"]'
        f'[data-objetivo="{objetivo}"]'
    )
    return Flujo(
        clave=clave,
        nombre=nombre,
        superficie="gestion-rrhh",
        ruta=f"/portal-empleado/?presentacion=rrhh&perfil=administrador#bolsa/{vista}",
        selector_titulo="#titulo-vista",
        titulo_esperado=titulo,
        selectores_listos=(".aviso-presentacion:not([hidden])", "#espacio-trabajo > :first-child"),
        pasos=(
            PasoInteraccion("clic-confirmando", selector_operacion),
            PasoInteraccion("esperar", "#dialogo-detalle[open] .recibo-presentacion", "DEMO-REC"),
            PasoInteraccion("esperar", "#dialogo-detalle[open] .recibo-presentacion", objetivo),
        ),
        selector_menu_actual=f'[data-vista="{vista}"]',
        requiere_demo=True,
    )


FLUJOS_RRHH_CON_RECIBO: tuple[Flujo, ...] = (
    _flujo_rrhh_con_recibo(
        "rrhh-bases-recibo-demo", "Bases guardadas con recibo DEMO", "convocatorias",
        "Convocatorias, bases y calendario", "guardar-bases", "DEMO-BOL-014",
        'form[data-comando="guardar-bases"] [data-accion="operacion-presentacion"][data-operacion="guardar-bases"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-admision-recibo-demo", "Admisión con recibo DEMO", "solicitudes",
        "Solicitudes y admisión", "admitir-solicitud", "DEMO-SOL-001",
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-merito-recibo-demo", "Mérito aceptado con recibo DEMO", "meritos",
        "Revisión de méritos", "aceptar-merito", "DEMO-MER-001",
        'form[aria-label*="DEMO-MER-001"] [data-accion="operacion-presentacion"][data-operacion="aceptar-merito"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-baremo-recibo-demo", "Reglas de baremo con recibo DEMO", "baremacion",
        "Baremación y ranking", "guardar-reglas-baremo", "DEMO-CRI-001",
        'form[data-comando="guardar-reglas-baremo"] [data-accion="operacion-presentacion"][data-operacion="guardar-reglas-baremo"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-importacion-recibo-demo", "Importación validada con recibo DEMO", "importacion",
        "Importación Convoca", "validar-importacion", "DEMO-IMP-NUEVA",
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-llamamiento-recibo-demo", "Llamamiento preparado con recibo DEMO", "llamamientos",
        "Nuevo llamamiento", "emitir-llamamiento", "DEMO-LLA-NUEVO",
        'form[data-comando="emitir-llamamiento"] [data-accion="operacion-presentacion"][data-operacion="emitir-llamamiento"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-contrato-recibo-demo", "Contrato registrado con recibo DEMO", "contratos",
        "Contratos, ceses y reincorporaciones", "registrar-contrato", "DEMO-CON-NUEVO",
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-comunicacion-recibo-demo", "Comunicación preparada con recibo DEMO", "comunicaciones",
        "Correo y mensajería", "preparar-comunicacion", "DEMO-COM-NUEVA",
        'form[data-comando="preparar-comunicacion"] [data-accion="operacion-presentacion"][data-operacion="preparar-comunicacion"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-exportacion-recibo-demo", "Exportación preparada con recibo DEMO", "estadisticas",
        "Estadísticas y explotación de datos", "exportar-informe", "DEMO-INF-ESTADISTICAS",
        'form[data-comando="exportar-informe"] [data-accion="operacion-presentacion"][data-operacion="exportar-informe"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-rol-recibo-demo", "Rol propuesto con recibo DEMO", "configuracion",
        "Configuración y roles", "crear-rol", "DEMO-ROL-NUEVO",
        'form[data-comando="crear-rol"] [data-accion="operacion-presentacion"][data-operacion="crear-rol"]',
    ),
    _flujo_rrhh_con_recibo(
        "rrhh-alegacion-recibo-demo", "Alegación estimada con recibo DEMO", "alegaciones",
        "Alegaciones", "resolver-alegacion", "DEMO-ALE-001",
    ),
)

MANIFIESTO_FLUJOS = (*MANIFIESTO_FLUJOS, *FLUJOS_RRHH_CON_RECIBO)
MANIFIESTO: tuple[Escenario, ...] = (*MANIFIESTO_VISTAS, *MANIFIESTO_FLUJOS)


def normalizar_url_base(valor: str) -> str:
    """Valida una URL HTTP(S) cuyo host sea una IP de loopback literal."""
    valor = valor.strip()
    if not valor:
        raise ValueError("la URL base no puede estar vacía")
    analizada = urlparse(valor)
    if analizada.scheme not in {"http", "https"} or not analizada.netloc:
        raise ValueError("la URL base debe usar http:// o https:// y contener un host")
    if analizada.username or analizada.password:
        raise ValueError("la URL base no puede incluir credenciales")
    if analizada.query or analizada.fragment:
        raise ValueError("la URL base no puede incluir consulta ni fragmento")
    host = analizada.hostname or ""
    if "%" in host:
        raise ValueError("la URL base exige una IP de loopback literal sin zona")
    try:
        ip = ip_address(host)
        _ = analizada.port
    except ValueError as error:
        raise ValueError("la URL base exige una IP de loopback literal y un puerto válido") from error
    if not ip.is_loopback:
        raise ValueError("la URL base solo puede usar una IP de loopback literal")
    return valor.rstrip("/")


def cabecera_presentacion_valida(cabeceras: Mapping[str, str]) -> bool:
    """Comprueba la marca exacta emitida únicamente por el servidor DEMO."""
    return any(
        nombre.casefold() == CABECERA_MODO_PRESENTACION.casefold() and valor == VALOR_MODO_PRESENTACION
        for nombre, valor in cabeceras.items()
    )


def construir_url(url_base: str, ruta: str) -> str:
    """Une la URL base y una ruta del manifiesto de forma determinista."""
    return urljoin(f"{normalizar_url_base(url_base)}/", ruta.lstrip("/"))


def slug_castellano(valor: str) -> str:
    """Produce un nombre de fichero ASCII estable y legible."""
    normalizado = unicodedata.normalize("NFKD", valor)
    sin_tildes = "".join(caracter for caracter in normalizado if not unicodedata.combining(caracter))
    slug = re.sub(r"[^a-z0-9]+", "-", sin_tildes.casefold()).strip("-")
    return slug or "sin-nombre"


def validar_manifiesto(
    escenarios: Sequence[Escenario] = MANIFIESTO,
    superficies: dict[str, Superficie] = SUPERFICIES,
    tamanos: Sequence[TamanoVista] = TAMANOS_VISTA,
) -> list[str]:
    """Devuelve inconsistencias del manifiesto sin importar Playwright."""
    errores: list[str] = []
    claves: set[str] = set()
    rutas_tipo: set[tuple[str, str]] = set()
    for escenario in escenarios:
        if escenario.clave in claves:
            errores.append(f"clave de escenario duplicada: {escenario.clave}")
        claves.add(escenario.clave)
        identidad_ruta = (escenario.tipo, escenario.ruta)
        if identidad_ruta in rutas_tipo and escenario.tipo == "vista":
            errores.append(f"ruta de vista duplicada: {escenario.ruta}")
        rutas_tipo.add(identidad_ruta)
        if escenario.superficie not in superficies:
            errores.append(f"superficie desconocida en {escenario.clave}: {escenario.superficie}")
            continue
        if not escenario.ruta.startswith("/"):
            errores.append(f"ruta no absoluta en {escenario.clave}: {escenario.ruta}")
        if not escenario.selector_titulo or not escenario.titulo_esperado:
            errores.append(f"título no verificable en {escenario.clave}")
        if not escenario.selectores_listos:
            errores.append(f"sin condición de carga en {escenario.clave}")
        superficie = superficies[escenario.superficie]
        if superficie.privada:
            consulta = parse_qs(urlparse(escenario.ruta).query)
            if consulta.get("presentacion") != ["rrhh"]:
                errores.append(f"ruta privada fuera de presentación RRHH: {escenario.clave}")
        if isinstance(escenario, Flujo):
            acciones_validas = {
                "clic", "clic-confirmando", "esperar", "enfocar",
                "esperar-habilitado", "esperar-deshabilitado", "abrir-menu",
            }
            if not escenario.pasos:
                errores.append(f"flujo sin pasos: {escenario.clave}")
            for paso in escenario.pasos:
                if paso.accion not in acciones_validas or not paso.selector:
                    errores.append(f"paso inválido en {escenario.clave}: {paso.accion}")
            if escenario.requiere_demo and not superficie.privada:
                errores.append(f"flujo DEMO fuera de superficie privada: {escenario.clave}")

    claves_tamano: set[str] = set()
    for tamano in tamanos:
        if tamano.clave in claves_tamano:
            errores.append(f"clave de tamaño duplicada: {tamano.clave}")
        claves_tamano.add(tamano.clave)
        if tamano.ancho <= 0 or tamano.alto <= 0:
            errores.append(f"tamaño no positivo: {tamano.clave}")
    return errores


def hallazgo(codigo: str, mensaje: str, detalles: Any = None) -> dict[str, Any]:
    resultado: dict[str, Any] = {"severidad": "error", "codigo": codigo, "mensaje": mensaje}
    if detalles not in (None, [], {}, ""):
        resultado["detalles"] = detalles
    return resultado
