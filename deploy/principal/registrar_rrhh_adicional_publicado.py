#!/usr/bin/env python3
"""Vincula la segunda hoja RRHH al perfil publicado por el lote ADMIN real."""

import argparse
import json
import os
from pathlib import Path
import re
import tempfile
from datetime import datetime, timezone

from deploy.principal.preparar_rrhh_adicional_copia import huella, leer_json, privado

REF = re.compile(r"^(?:prf_|vca_|per_|cta_)[A-Za-z0-9_-]{22,128}$")
SHA = re.compile(r"^[0-9a-f]{64}$")
ROL = re.compile(r"^rol:[a-z][a-z0-9_]{2,127}:v[1-9][0-9]*$")


def instante(valor):
    if not isinstance(valor, str) or not valor.endswith("Z"):
        raise ValueError("El recibo no contiene un instante UTC")
    try:
        fecha = datetime.fromisoformat(valor.replace("Z", "+00:00"))
    except ValueError as exc:
        raise ValueError("El recibo contiene un instante inválido") from exc
    if fecha.tzinfo != timezone.utc:
        raise ValueError("El recibo no está en UTC")
    return fecha


def extraer(material: Path, preparacion: dict, respuesta: dict, rol: str):
    identidad = leer_json(material / "identidad/tecnico-rrhh-adicional.json")
    semilla = leer_json(material / "identidad/tecnico-rrhh-adicional-semilla.json")
    if (identidad.get("certificado") != "mtls/tecnico-rrhh-adicional.crt"
            or identidad.get("identidad") != "identidad/tecnico-rrhh-adicional-identidad.json"):
        raise ValueError("La identidad privada del segundo RRHH no corresponde al productor")
    hoja = leer_json(material / identidad["identidad"])
    privado(material / identidad["certificado"])
    huella_certificado = huella(material / identidad["certificado"])
    if (hoja.get("certificate_sha256") != huella_certificado
            or hoja.get("subject") != identidad.get("sujeto") or hoja.get("roles") != ["tecnico_rrhh"]):
        raise ValueError("La hoja y la identidad no coinciden")
    if (semilla.get("certificado_sha256") != huella_certificado
            or semilla.get("sujeto") != identidad.get("sujeto")
            or semilla.get("rol_base_ref") != "rol:tecnico_rrhh_desarrollo:v1"
            or not isinstance(semilla.get("ca37_organizacion_ref"), str)
            or not re.fullmatch(r"org_[a-z0-9]{16,80}", semilla["ca37_organizacion_ref"])
            or not str(semilla.get("ca37_recibo_ref", "")).startswith("recibo_ca_identidad:")):
        raise ValueError("La semilla de copia no corresponde a la segunda hoja")
    if not ROL.fullmatch(rol):
        raise ValueError("La versión de rol elegida es inválida")
    p, r = preparacion.get("preparacion"), respuesta.get("recibo")
    if not isinstance(p, dict) or not isinstance(r, dict) or len(r.get("cambios", [])) != 1:
        raise ValueError("Faltan la preparación o el recibo del lote")
    altas = [alta for alta in p.get("altas", []) if alta.get("rol_version_ref") == rol]
    if len(altas) != 1:
        raise ValueError("El rol no está ofrecido una sola vez en la preparación")
    alta, cambio = altas[0], r["cambios"][0]
    persona, cuenta = semilla.get("persona_ref"), semilla.get("cuenta_ref")
    perfil, vinculo = alta.get("perfil_ref"), alta.get("vinculo_ref")
    if (any(not isinstance(ref, str) or not REF.fullmatch(ref) for ref in (persona, cuenta, perfil, vinculo))
            or perfil == semilla.get("perfil_base_ref") or p.get("persona_ref") != persona
            or p.get("cuenta_ref") != cuenta or p.get("unidad_ref") != cambio.get("unidad_ref")
            or cambio.get("objetivo_persona_ref") != persona
            or cambio.get("perfil_ref") != perfil or cambio.get("vinculo_ref") != vinculo
            or cambio.get("rol_version_ref") != rol or cambio.get("estado_posterior") != "activo"
            or cambio.get("version_posterior") != 1 or cambio.get("operacion_ref") != r.get("operacion_ref")
            or cambio.get("acto_ref") != r.get("acto_ref") or cambio.get("recibo_ref") != r.get("recibo_ref")
            or cambio.get("huella_antes_sha256") != alta.get("huella_sha256")
            or not SHA.fullmatch(str(alta.get("huella_sha256", "")))
            or not SHA.fullmatch(str(cambio.get("huella_despues_sha256", "")))
            or not SHA.fullmatch(str(r.get("huella_solicitud_sha256", "")))
            or not SHA.fullmatch(str(r.get("fuentes_sha256", "")))
            or not isinstance(r.get("acto_ref"), str) or not r["acto_ref"].startswith("acto_admin:")
            or not isinstance(r.get("recibo_ref"), str) or not r["recibo_ref"].startswith("recibo_admin:")):
        raise ValueError("El lote no publicó el perfil de la segunda persona")
    confirmado, desde, hasta = map(instante, (r.get("confirmado_en"), cambio.get("vigente_desde"), cambio.get("vigente_hasta")))
    if desde > confirmado or hasta <= confirmado:
        raise ValueError("El perfil no está vigente al confirmarse el lote")
    return {"version": 1, "autoridad": "no_autoritativo", "sujeto": identidad["sujeto"],
            "certificado_sha256": huella_certificado,
            "persona_ref": persona, "cuenta_ref": cuenta, "perfil_ref": perfil, "vinculo_ref": vinculo,
            "rol_version_ref": rol, "unidad_ref": p["unidad_ref"],
            "ca37_organizacion_ref": semilla["ca37_organizacion_ref"], "operacion_ref": r["operacion_ref"],
            "acto_ref": r["acto_ref"], "recibo_ref": r["recibo_ref"], "confirmado_en": r["confirmado_en"],
            "vigente_desde": cambio["vigente_desde"], "vigente_hasta": cambio["vigente_hasta"],
            "huella_antes_sha256": cambio["huella_antes_sha256"],
            "huella_despues_sha256": cambio["huella_despues_sha256"]}


def registrar(material: Path, preparacion: Path, recibo: Path, rol: str):
    material = material.absolute()
    if material.resolve(strict=True) != material:
        raise ValueError("La ruta del material no debe contener enlaces")
    privado(material, True)
    privado(material / "identidad", True)
    dato = extraer(material, leer_json(preparacion), leer_json(recibo), rol)
    destino = material / "identidad/tecnico-rrhh-adicional-publicado.json"
    contenido = (json.dumps(dato, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
    if destino.exists() or destino.is_symlink():
        privado(destino)
        if destino.read_bytes() != contenido:
            raise ValueError("El perfil publicado difiere del material retenido")
        return dato
    descriptor, temporal = tempfile.mkstemp(prefix=".rrhh-publicado-", dir=destino.parent)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as salida:
            salida.write(contenido)
            salida.flush()
            os.fsync(salida.fileno())
        os.link(temporal, destino)
    finally:
        os.unlink(temporal)
    return dato


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--material-dir", type=Path, required=True)
    parser.add_argument("--preparacion", type=Path, required=True)
    parser.add_argument("--recibo", type=Path, required=True)
    parser.add_argument("--rol-version-ref", required=True)
    args = parser.parse_args()
    registrar(args.material_dir, args.preparacion, args.recibo, args.rol_version_ref)
    print("Perfil del segundo RRHH vinculado al recibo ADMIN; falta comprobar el acceso en la copia.")


if __name__ == "__main__":
    main()
