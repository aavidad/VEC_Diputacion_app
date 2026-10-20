"""Certificados e identidades propios de los dos actores del centro."""

from argparse import Namespace
import importlib.util
import json
import os
from pathlib import Path

from copia_local_necesidades import entorno_necesidades


def _productor(repo: Path):
    """Carga el productor Git de material sin ejecutar su CLI."""
    ruta = repo / "scripts/recorridos/clon_material.py"
    spec = importlib.util.spec_from_file_location("vec_clon_material_centros", ruta)
    if spec is None or spec.loader is None:
        raise ValueError("Falta el productor versionado de certificados")
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


def _publicar(ruta: Path, valor: dict) -> None:
    with open(ruta, "x", encoding="utf-8", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
        json.dump(valor, salida, ensure_ascii=False, indent=2)
        salida.write("\n")


def preparar(copia) -> dict:
    """Crea dos identidades de centro con la CA y el catálogo de esta copia.

    El primer arranque publica los perfiles con la autoridad nativa de CT.
    Esta etapa no escribe asignaciones ni concede roles PostgreSQL.
    """
    estado = copia.leer()
    if estado.get("material_modo") != "nuevo" or not copia.existe():
        raise ValueError("Se requiere una copia nueva con material propio y PostgreSQL activo")
    repo = Path(estado["fuente"])
    material = copia.material
    identidades = material / "identidad"
    destino = identidades / "centros.json"
    if destino.exists() or destino.is_symlink():
        raise ValueError("Los centros ya se prepararon")
    productor = _productor(repo)
    roles = (productor.ROLES["centro_solicitante"], productor.ROLES["ratificador"])
    for nombre, archivo, _ in roles:
        for ruta in (material / "mtls" / (nombre + extension) for extension in
                     (".key", ".crt", ".p12", ".p12.password")):
            if ruta.exists() or ruta.is_symlink():
                raise ValueError("Ya existe material de un centro")
        if (identidades / (archivo + ".json")).exists():
            raise ValueError("Ya existe la identidad de un centro")
    if not (material / "ca/ca.crt").is_file() or not (material / "ca/ca.key").is_file():
        raise ValueError("Falta la CA propia de la copia")
    seleccion = json.loads((repo / "deploy/principal/candidata_hz39/datos/seleccion.json").read_text())
    catalogo = productor.read_catalog(Namespace(engine="docker", container=copia.nombre))
    centro_4_0 = seleccion["centro_ref"]
    # La selección es dato versionado; los puestos de responsabilidad son los
    # publicados actualmente en la base de la copia.
    del_centro = {**catalogo, "centers": [centro_4_0],
                  "positions": [p for p in catalogo["positions"] if p.get("center") == centro_4_0]}
    vinculo = productor.center_bindings(del_centro)
    if vinculo is None or vinculo[0] != centro_4_0:
        raise ValueError("El centro del caso 4.0 carece de dos puestos publicados")
    nombres, _ = productor.fresh_profile_names()
    sujetos_existentes = set()
    huellas_existentes = set()
    for ruta in identidades.glob("*.json"):
        contenido = json.loads(ruta.read_text())
        if isinstance(contenido, dict):
            if isinstance(contenido.get("subject"), str):
                sujetos_existentes.add(contenido["subject"])
            if isinstance(contenido.get("certificate_sha256"), str):
                huellas_existentes.add(contenido["certificate_sha256"])
    centro, puesto_solicitante, puesto_ratificador = vinculo
    perfiles = []
    entradas = []
    for clave, puesto in (("centro_solicitante", puesto_solicitante),
                          ("ratificador", puesto_ratificador)):
        nombre_archivo, identidad_archivo, rol = productor.ROLES[clave]
        sujeto = "desarrollo:" + copia.nombre + ":" + clave
        if sujeto in sujetos_existentes:
            raise ValueError("Un sujeto del centro coincide con otro actor")
        sujetos_existentes.add(sujeto)
        productor.certificate(material, nombre_archivo, sujeto, days=90)
        huella = productor.cert_hash(material / "mtls" / (nombre_archivo + ".crt"))
        if huella in huellas_existentes:
            raise ValueError("El certificado del centro coincide con otro actor")
        huellas_existentes.add(huella)
        identidad = {"version": 1, "autoridad": "no_autoritativo",
                     "certificate_sha256": huella, "subject": sujeto,
                     "display_name": nombres[clave], "roles": [rol]}
        _publicar(identidades / (identidad_archivo + ".json"), identidad)
        entrada = {"certificate": "mtls/" + nombre_archivo + ".crt",
                   "identity": "identidad/" + identidad_archivo + ".json",
                   "role": rol, "centro_ref": centro, "puesto_ref": puesto}
        entradas.append(entrada)
        perfiles.append({"subject": sujeto, "role": rol, "certificate_sha256": huella})
    entradas[0]["ratificador_subject"] = perfiles[1]["subject"]
    _publicar(destino, {"version": 1, "autoridad": "no_autoritativo", "entradas": entradas})
    return {"centro_ref": centro, "puestos_ref": [puesto_solicitante, puesto_ratificador],
            "catalogo_version": catalogo["version"], "catalogo_revision": catalogo["revision"],
            "perfiles": perfiles, "perfiles_estado": "pendientes_del_siguiente_arranque",
            "entorno": {**entorno_necesidades(repo),
                        "VEC_PERSONAL_ORGANIZACION_POSTGRESQL": "true",
                        "VEC_PERSONAL_ORGANIZACION_VERSION": str(catalogo["version"])}}
