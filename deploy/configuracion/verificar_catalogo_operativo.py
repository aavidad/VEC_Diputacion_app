"""Valida y proyecta el catálogo operativo canónico sin leer secretos."""

import argparse
import json
import re
import sys
from collections import Counter, defaultdict
from pathlib import Path
from typing import Dict, List, Set, Tuple

CATALOGO = Path(__file__).with_name("catalogo_operativo.json").resolve()
RAIZ = Path(__file__).resolve().parents[2]
SUPERFICIES = {"vec-publico", "vec-interno", "vec-emisor-capacidad-v4"}
ESTADOS = {
    "configurable",
    "guardia",
    "observado-prohibido",
    "alias",
    "heredado",
    "prohibido",
}
PLANTILLAS = {"fijo", "vacio", "omitir"}
PATRON_NOMBRE = re.compile(r"^(?:VEC|BOLSA)_[A-Z0-9_]+$")
PATRON_IDENTIFICADOR = re.compile(r"\b[A-Za-z_][A-Za-z0-9_]*\b")
PATRON_ACCESO = re.compile(r"\bos\s*\.\s*(Getenv|LookupEnv)\s*\(")
PATRON_ASIGNACION = re.compile(r"([A-Za-z_][A-Za-z0-9_]*)\s*=\s*$")

CAMPOS_RAIZ = {
    "version",
    "origen",
    "variables",
    "uso_por_superficie",
    "literales_entorno",
    "lectores_entorno",
    "grupos_configuracion",
    "infraestructura",
}
CAMPOS_VARIABLE = {
    "nombre",
    "definiciones",
    "lectores",
    "valor_predeterminado",
    "tipo",
    "secreto",
    "ancla",
    "descripcion",
}
CAMPOS_USO = {"nombre", "estado", "requerido", "plantilla"}
CAMPOS_INVENTARIO = {"ruta", "cantidad"}
CAMPOS_LECTOR = {"ruta", "funcion", "argumento", "cantidad"}
CAMPOS_GRUPO = {
    "nombre",
    "superficie",
    "cardinalidad",
    "obligatorio",
    "miembros",
    "descripcion",
}

SECRETOS_CONOCIDOS = {
    "VEC_FAKE_CREDENTIALS_FILE",
    "VEC_TLS_KEY_FILE",
    "VEC_DEVELOPMENT_MATERIAL_DIR",
    "VEC_BOLSA_BORRADORES_EJECUTOR_CONSULTA_DATABASE_URL",
    "VEC_BOLSA_BORRADORES_PROYECTOR_GOBIERNO_DATABASE_URL",
    "VEC_BOLSA_BORRADORES_VERIFICADOR_RECIBO_DATABASE_URL",
    "VEC_BOLSA_PUBLICA_DATABASE_URL",
    "VEC_INTERNO_TLS_KEY_FILE",
    "VEC_V4_EMISOR_DATABASE_URL",
    "VEC_V4_EJECUTOR_DATABASE_URL",
}
ANCLAS_CONOCIDAS = {
    "VEC_BOLSA_CATEGORIES_CATALOG_SHA256",
    "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256",
    "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256",
    "VEC_INTERNO_PROXY_TLS_SHA256",
}
PUBLICAS_OBLIGATORIAS = {
    "VEC_EXECUTION_PROFILE",
    "VEC_AUTH_MODE",
    "VEC_BOLSA_PUBLICA_DATABASE_URL",
    "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256",
    "VEC_BOLSA_CATEGORIES_CATALOG_ID",
    "VEC_BOLSA_CATEGORIES_CATALOG_VERSION",
    "VEC_BOLSA_CATEGORIES_CATALOG_SHA256",
    "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256",
}
PUBLICAS_FIJAS = {
    "VEC_EXECUTION_PROFILE": "produccion",
    "VEC_AUTH_MODE": "disabled",
    "VEC_BOLSA_CATEGORIES_CATALOG_ID": "categorias-profesionales",
}
SENTINELAS_PUBLICOS = {
    "VEC_DEVELOPMENT_GUARD",
    "VEC_DEVELOPMENT_MATERIAL_DIR",
    "VEC_BOLSA_PUBLIC_SOURCE_PATH",
    "VEC_BOLSA_CATEGORIES_SOURCE_PATH",
    "VEC_RRHH_PRESENTATION_ENABLED",
    "VEC_RRHH_PRESENTATION_GUARD_ONE",
    "VEC_RRHH_PRESENTATION_GUARD_TWO",
}
SENTINELAS_INTERNOS_PROHIBIDOS = {
    "VEC_AUTH_MODE",
    "BOLSA_AUTH_MODE",
    "VEC_BOLSA_STORAGE_MODE",
    "BOLSA_STORAGE_MODE",
    "VEC_DEVELOPMENT_GUARD",
    "VEC_DEVELOPMENT_MATERIAL_DIR",
    "VEC_RRHH_PRESENTATION_ENABLED",
    "VEC_RRHH_PRESENTATION_GUARD_ONE",
    "VEC_RRHH_PRESENTATION_GUARD_TWO",
}


def argumentos() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Valida el catálogo canónico o genera una plantilla mínima."
    )
    grupo = parser.add_mutually_exclusive_group(required=True)
    grupo.add_argument(
        "--validar", action="store_true", help="Valida catálogo y Go productivo."
    )
    grupo.add_argument(
        "--plantilla", choices=sorted(SUPERFICIES), help="Proyecta una superficie."
    )
    return parser.parse_args()


def cargar_catalogo() -> dict:
    try:
        with CATALOGO.open(encoding="utf-8") as archivo:
            return json.load(archivo)
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(
            "CAT001 catálogo canónico no legible: {}".format(error)
        ) from error


def fuentes_go(raiz: Path) -> List[Path]:
    resultado = []
    for directorio in ("cmd", "config", "internal"):
        for ruta in (raiz / directorio).rglob("*.go"):
            relativa = ruta.relative_to(raiz)
            if ruta.name.endswith("_test.go") or "testdata" in relativa.parts:
                continue
            resultado.append(ruta)
    return sorted(resultado)


def separar_go(texto: str) -> Tuple[str, List[Tuple[str, int, int]]]:
    """Oculta comentarios/literales y conserva posiciones para analizar llamadas."""
    codigo = list(texto)
    literales = []
    indice = 0
    limite = len(texto)
    while indice < limite:
        if texto.startswith("//", indice):
            fin = texto.find("\n", indice)
            fin = limite if fin < 0 else fin
            codigo[indice:fin] = " " * (fin - indice)
            indice = fin
        elif texto.startswith("/*", indice):
            fin = texto.find("*/", indice + 2)
            fin = limite if fin < 0 else fin + 2
            for posicion in range(indice, fin):
                if codigo[posicion] != "\n":
                    codigo[posicion] = " "
            indice = fin
        elif texto[indice] in ('"', "`", "'"):
            inicio = indice
            comilla = texto[indice]
            indice += 1
            contenido = []
            while indice < limite:
                if comilla != "`" and texto[indice] == "\\":
                    contenido.append(texto[indice : indice + 2])
                    indice += 2
                    continue
                if texto[indice] == comilla:
                    indice += 1
                    break
                contenido.append(texto[indice])
                indice += 1
            fin = indice
            if comilla in ('"', "`"):
                literales.append(("".join(contenido), inicio, fin))
            for posicion in range(inicio, fin):
                codigo[posicion] = " "
        else:
            indice += 1
    return "".join(codigo), literales


def analizar_fuentes(raiz: Path) -> dict:
    archivos = {}
    identificadores = defaultdict(set)
    definiciones = defaultdict(set)
    inventario_literales = Counter()
    accesos = Counter()
    for ruta in fuentes_go(raiz):
        relativa = ruta.relative_to(raiz).as_posix()
        texto = ruta.read_text(encoding="utf-8")
        codigo, literales = separar_go(texto)
        archivos[relativa] = (texto, codigo, literales)
        for valor, inicio, _ in literales:
            if not PATRON_NOMBRE.fullmatch(valor):
                continue
            definiciones[valor].add(relativa)
            inventario_literales[relativa] += 1
            asignacion = PATRON_ASIGNACION.search(codigo[:inicio])
            if asignacion:
                identificadores[asignacion.group(1)].add(valor)
        inicio_literales = {inicio: valor for valor, inicio, _ in literales}
        for acceso in PATRON_ACCESO.finditer(codigo):
            posicion = acceso.end()
            while posicion < len(texto) and texto[posicion].isspace():
                posicion += 1
            argumento = "literal" if posicion in inicio_literales else "dinamico"
            accesos[(relativa, acceso.group(1), argumento)] += 1
    lectores = defaultdict(set)
    for relativa, (_, codigo, literales) in archivos.items():
        if not any(clave[0] == relativa for clave in accesos):
            continue
        tokens = set(PATRON_IDENTIFICADOR.findall(codigo))
        for identificador in tokens:
            for nombre in identificadores.get(identificador, set()):
                lectores[nombre].add(relativa)
        inicio_literales = {inicio: valor for valor, inicio, _ in literales}
        for acceso in PATRON_ACCESO.finditer(codigo):
            posicion = acceso.end()
            while (
                posicion < len(archivos[relativa][0])
                and archivos[relativa][0][posicion].isspace()
            ):
                posicion += 1
            nombre = inicio_literales.get(posicion)
            if nombre and PATRON_NOMBRE.fullmatch(nombre):
                lectores[nombre].add(relativa)
    return {
        "definiciones": definiciones,
        "lectores": lectores,
        "literales": inventario_literales,
        "accesos": accesos,
    }


def inventario_declarado(
    elementos: object, campos: Set[str]
) -> Tuple[Counter, List[str]]:
    errores = []
    inventario = Counter()
    if not isinstance(elementos, list):
        return inventario, ["CAT010 inventario no es una lista."]
    for elemento in elementos:
        if not isinstance(elemento, dict) or set(elemento) != campos:
            errores.append("CAT011 entrada de inventario inválida.")
            continue
        orden = (
            ("ruta", "funcion", "argumento") if campos == CAMPOS_LECTOR else ("ruta",)
        )
        clave = tuple(elemento[campo] for campo in orden)
        cantidad = elemento["cantidad"]
        if (
            not all(isinstance(valor, str) and valor for valor in clave)
            or not isinstance(cantidad, int)
            or cantidad < 1
        ):
            errores.append("CAT012 entrada de inventario vacía o inválida.")
            continue
        inventario[clave] += cantidad
    return inventario, errores


def usos_primarios(catalogo: dict, superficie: str) -> List[dict]:
    usos = catalogo.get("uso_por_superficie", {}).get(superficie)
    return usos if isinstance(usos, list) else []


def validar_catalogo(catalogo: dict, raiz: Path = RAIZ) -> List[str]:
    errores = []
    if (
        not isinstance(catalogo, dict)
        or set(catalogo) != CAMPOS_RAIZ
        or catalogo.get("version") != 2
    ):
        return ["CAT002 esquema raíz cerrado inválido."]
    variables = catalogo["variables"]
    if not isinstance(variables, list):
        return ["CAT003 variables debe ser una lista."]
    por_nombre = {}
    for indice, variable in enumerate(variables, start=1):
        prefijo = "CAT{:03d}".format(100 + indice)
        if not isinstance(variable, dict) or set(variable) != CAMPOS_VARIABLE:
            errores.append("{} campos de variable inválidos.".format(prefijo))
            continue
        nombre = variable["nombre"]
        if not isinstance(nombre, str) or not PATRON_NOMBRE.fullmatch(nombre):
            errores.append("{} nombre inválido.".format(prefijo))
            continue
        if nombre in por_nombre:
            errores.append("CAT004 nombre duplicado: {}.".format(nombre))
        por_nombre[nombre] = variable
        for campo in ("definiciones", "lectores"):
            valores = variable[campo]
            if (
                not isinstance(valores, list)
                or not valores
                or len(valores) != len(set(valores))
            ):
                errores.append("{} {} vacío o duplicado.".format(prefijo, campo))
        if not isinstance(variable["secreto"], bool) or not isinstance(
            variable["ancla"], bool
        ):
            errores.append("{} secreto/ancla deben ser booleanos.".format(prefijo))
        if nombre in SECRETOS_CONOCIDOS and not variable["secreto"]:
            errores.append("CAT020 secreto conocido desclasificado: {}.".format(nombre))
        if variable["secreto"] and variable["valor_predeterminado"] is not None:
            errores.append(
                "CAT021 secreto con valor predeterminado: {}.".format(nombre)
            )
        if variable["ancla"] and (
            variable["secreto"] or variable["tipo"] not in {"sha256", "lista_sha256"}
        ):
            errores.append("CAT022 ancla incoherente: {}.".format(nombre))
        if nombre in ANCLAS_CONOCIDAS and not variable["ancla"]:
            errores.append("CAT023 ancla conocida desclasificada: {}.".format(nombre))
    analisis = analizar_fuentes(raiz)
    nombres_reales = set(analisis["definiciones"])
    if faltan := sorted(nombres_reales - set(por_nombre)):
        errores.append(
            "CAT005 faltan literales productivos: {}.".format(", ".join(faltan))
        )
    if sobran := sorted(set(por_nombre) - nombres_reales):
        errores.append("CAT006 sobran variables: {}.".format(", ".join(sobran)))
    for nombre in sorted(set(por_nombre) & nombres_reales):
        variable = por_nombre[nombre]
        if set(variable["definiciones"]) != analisis["definiciones"][nombre]:
            errores.append("CAT007 definiciones inexactas: {}.".format(nombre))
        if set(variable["lectores"]) != analisis["lectores"][nombre]:
            errores.append("CAT008 lectores inexactos: {}.".format(nombre))
    literales, fallos = inventario_declarado(
        catalogo["literales_entorno"], CAMPOS_INVENTARIO
    )
    errores.extend(fallos)
    reales_literales = Counter(
        {(ruta,): cantidad for ruta, cantidad in analisis["literales"].items()}
    )
    if literales != reales_literales:
        errores.append("CAT009 inventario de literales por ruta/cantidad inexacto.")
    lectores, fallos = inventario_declarado(catalogo["lectores_entorno"], CAMPOS_LECTOR)
    errores.extend(fallos)
    reales_lectores = Counter(
        {clave: cantidad for clave, cantidad in analisis["accesos"].items()}
    )
    if lectores != reales_lectores:
        errores.append(
            "CAT013 inventario de lectores por ruta/función/cantidad inexacto."
        )
    validar_usos(catalogo, por_nombre, errores)
    validar_grupos_e_infraestructura(catalogo, por_nombre, errores)
    return errores


def validar_usos(
    catalogo: dict, por_nombre: Dict[str, dict], errores: List[str]
) -> None:
    usos = catalogo["uso_por_superficie"]
    if not isinstance(usos, dict) or set(usos) != SUPERFICIES | {
        "configuracion-general"
    }:
        errores.append("CAT030 superficies incompletas o desconocidas.")
        return
    general = usos["configuracion-general"]
    if not isinstance(general, dict) or set(general) != {"heredado", "alias"}:
        errores.append("CAT031 configuración general inválida.")
        return
    nombres_usados = set(general["heredado"]) | set(general["alias"])
    if set(general["heredado"]) & set(general["alias"]):
        errores.append("CAT032 estado general duplicado.")
    for superficie in sorted(SUPERFICIES):
        vistos = set()
        for uso in usos_primarios(catalogo, superficie):
            if not isinstance(uso, dict) or set(uso) != CAMPOS_USO:
                errores.append("CAT033 uso por superficie inválido.")
                continue
            nombre = uso["nombre"]
            if nombre in vistos or nombre not in por_nombre:
                errores.append(
                    "CAT034 variable repetida o desconocida en {}.".format(superficie)
                )
            vistos.add(nombre)
            nombres_usados.add(nombre)
            if uso["estado"] not in ESTADOS or uso["plantilla"] not in PLANTILLAS:
                errores.append(
                    "CAT035 estado o plantilla inválidos: {}.".format(nombre)
                )
            proyectable = uso["estado"] == "configurable"
            if not proyectable and (uso["requerido"] or uso["plantilla"] != "omitir"):
                errores.append("CAT036 estado no proyectable: {}.".format(nombre))
            if uso["plantilla"] != "omitir" and not uso["requerido"]:
                errores.append("CAT037 plantilla no requerida: {}.".format(nombre))
            if uso["requerido"] and uso["plantilla"] == "omitir":
                errores.append(
                    "CAT045 variable requerida omitida de plantilla: {}.".format(nombre)
                )
    if nombres_usados != set(por_nombre):
        errores.append("CAT038 variables sin uso operativo o uso desconocido.")
    publicos = {uso["nombre"]: uso for uso in usos_primarios(catalogo, "vec-publico")}
    obligatorios = {nombre for nombre, uso in publicos.items() if uso["requerido"]}
    if obligatorios != PUBLICAS_OBLIGATORIAS:
        errores.append("CAT039 requisitos de vec-publico distintos del runbook.")
    for nombre, valor in PUBLICAS_FIJAS.items():
        if (
            publicos.get(nombre, {}).get("plantilla") != "fijo"
            or por_nombre.get(nombre, {}).get("valor_predeterminado") != valor
        ):
            errores.append("CAT040 valor fijo público incoherente: {}.".format(nombre))
    for nombre in SENTINELAS_PUBLICOS:
        if publicos.get(nombre, {}).get("estado") != "observado-prohibido":
            errores.append(
                "CAT041 sentinela público mal clasificado: {}.".format(nombre)
            )
    internos = {uso["nombre"]: uso for uso in usos_primarios(catalogo, "vec-interno")}
    if internos.get("VEC_EXECUTION_PROFILE", {}).get("estado") != "guardia":
        errores.append("CAT042 guardia interna de perfil ausente.")
    for nombre in SENTINELAS_INTERNOS_PROHIBIDOS:
        if internos.get(nombre, {}).get("estado") != "observado-prohibido":
            errores.append(
                "CAT043 sentinela interno mal clasificado: {}.".format(nombre)
            )
    emisores = {
        uso["nombre"]: uso
        for uso in usos_primarios(catalogo, "vec-emisor-capacidad-v4")
    }
    if emisores.get("VEC_V4_EJECUTOR_DATABASE_URL", {}).get("estado") != "prohibido":
        errores.append("CAT044 DSN ejecutor V4 no está prohibido.")


def validar_grupos_e_infraestructura(
    catalogo: dict, por_nombre: Dict[str, dict], errores: List[str]
) -> None:
    grupos = catalogo["grupos_configuracion"]
    if not isinstance(grupos, list) or not grupos:
        errores.append("CAT050 grupos de configuración vacíos.")
    else:
        nombres_grupo = {
            grupo.get("nombre") for grupo in grupos if isinstance(grupo, dict)
        }
        if nombres_grupo != {"tls-publico"}:
            errores.append("CAT058 conjunto de grupos de configuración inesperado.")
        for grupo in grupos:
            if (
                not isinstance(grupo, dict)
                or set(grupo) != CAMPOS_GRUPO
                or not grupo["miembros"]
            ):
                errores.append("CAT051 grupo de configuración inválido.")
                continue
            if (
                grupo["superficie"] not in SUPERFICIES
                or grupo["cardinalidad"] != "0-o-2"
            ):
                errores.append("CAT052 cardinalidad o superficie de grupo inválida.")
            if grupo["obligatorio"] or len(grupo["miembros"]) != 2:
                errores.append("CAT053 grupo TLS público debe ser opcional 0-o-2.")
            if set(grupo["miembros"]) != {"VEC_TLS_CERT_FILE", "VEC_TLS_KEY_FILE"}:
                errores.append("CAT059 miembros del grupo TLS público inexactos.")
            if any(miembro not in por_nombre for miembro in grupo["miembros"]):
                errores.append("CAT054 miembro de grupo desconocido.")
    infraestructura = catalogo["infraestructura"]
    if not isinstance(infraestructura, dict) or set(infraestructura) != {
        "nota",
        "compose",
        "osm",
        "ceph",
    }:
        errores.append("CAT055 infraestructura inválida.")
        return
    for seccion in ("compose", "osm", "ceph"):
        valores = infraestructura[seccion]
        if (
            not isinstance(valores, list)
            or not valores
            or len(valores) != len(set(valores))
        ):
            errores.append(
                "CAT056 infraestructura vacía o duplicada: {}.".format(seccion)
            )
    if (
        "UID_GID" not in infraestructura["compose"]
        or "UID_GID" not in infraestructura["osm"]
    ):
        errores.append("CAT057 UID_GID debe pertenecer a Compose raíz y OSM.")


def generar_plantilla(catalogo: dict, superficie: str) -> str:
    por_nombre = {variable["nombre"]: variable for variable in catalogo["variables"]}
    lineas = [
        "# Plantilla mínima de {}; exportar solo en la sesión del proceso.".format(
            superficie
        )
    ]
    for uso in usos_primarios(catalogo, superficie):
        if uso["plantilla"] == "omitir":
            continue
        variable = por_nombre[uso["nombre"]]
        if uso["plantilla"] == "fijo":
            lineas.append(
                "{}={}".format(uso["nombre"], variable["valor_predeterminado"])
            )
        else:
            clase = "secreto" if variable["secreto"] else "ancla o dato gobernado"
            lineas.append("# OBLIGATORIA: inyectar {} autorizado.".format(clase))
            lineas.append("{}=".format(uso["nombre"]))
    return "\n".join(lineas) + "\n"


def main() -> int:
    opciones = argumentos()
    try:
        catalogo = cargar_catalogo()
        errores = validar_catalogo(catalogo)
    except ValueError as error:
        print(error, file=sys.stderr)
        return 2
    if errores:
        print("\n".join(errores), file=sys.stderr)
        return 1
    if opciones.validar:
        print("CAT000 catálogo canónico válido: 58 literales y lectores exactos.")
    else:
        sys.stdout.write(generar_plantilla(catalogo, opciones.plantilla))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
