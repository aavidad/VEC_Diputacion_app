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

SUPERFICIES: dict[str, Superficie] = {
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
}

MANIFIESTO_VISTAS: tuple[Vista, ...] = (
    Vista(
        clave="publico-convocatorias", nombre="Consulta pública", superficie="portal-publico",
        ruta="/bolsa/", selector_titulo="#titulo-portal",
        titulo_esperado="Bolsas y procesos selectivos",
        selectores_listos=('#panel-listado[aria-busy="false"]', '#directorio-categorias[aria-busy="false"]'),
        selector_menu_actual='.navegacion-publica a[href="#contenido-principal"]',
    ),
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
)

MANIFIESTO: tuple[Escenario, ...] = (*MANIFIESTO_VISTAS, *MANIFIESTO_FLUJOS)


def normalizar_url_base(valor: str, permitir_red_privada: bool = False) -> str:
    """Valida loopback o, con concesión explícita, una IP Docker privada."""
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
    if not ip.is_loopback and not (permitir_red_privada and ip.is_private):
        raise ValueError("la URL base solo puede usar loopback o la red Docker privada autorizada")
    return valor.rstrip("/")


def cabecera_presentacion_valida(cabeceras: Mapping[str, str]) -> bool:
    """Comprueba la marca exacta emitida únicamente por el servidor DEMO."""
    return any(
        nombre.casefold() == CABECERA_MODO_PRESENTACION.casefold() and valor == VALOR_MODO_PRESENTACION
        for nombre, valor in cabeceras.items()
    )


def construir_url(url_base: str, ruta: str, permitir_red_privada: bool = False) -> str:
    """Une la URL base y una ruta del manifiesto de forma determinista."""
    return urljoin(
        f"{normalizar_url_base(url_base, permitir_red_privada=permitir_red_privada)}/",
        ruta.lstrip("/"),
    )


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
