#!/usr/bin/env python3
"""Parametriza datos ficticios de firmantes P; sin SQL ni HTTP ejecutados."""
import argparse
import datetime as dt
import hashlib
import hmac
import json
import os
import re
import subprocess
from pathlib import Path

ROLES = ("tecnico", "jefatura", "diputacion", "secretaria")
HEX64 = re.compile(r"^[0-9a-f]{64}$")
REF = re.compile(r"^[A-Za-z0-9][A-Za-z0-9:._-]{2,127}$")
ESPACIO = "https://localhost/vec/desarrollo/identidad"
DOMINIO = "vec.identidad.desarrollo.hmac.v1"


def fuente(value):
    p = Path(value)
    if not p.is_absolute() or p.is_symlink() or not p.is_file() or p.stat().st_mode & 0o077 or p.stat().st_uid != os.getuid():
        raise ValueError("fuente privada inválida")
    return p


def plantilla(name):
    """Lee solo una plantilla versionada junto a este generador."""
    p = Path(__file__).resolve().parent / name
    if p.is_symlink() or not p.is_file() or p.stat().st_uid != os.getuid() or p.stat().st_mode & 0o022:
        raise ValueError("plantilla pública inválida")
    return p


def exigir(ok, message):
    if not ok:
        raise ValueError(message)


def guardar(path, data):
    raw = data if isinstance(data, bytes) else json.dumps(data, ensure_ascii=False, separators=(",", ":"), allow_nan=False).encode() + b"\n"
    with path.open("xb") as f:
        f.write(raw)
        f.flush()
        os.fsync(f.fileno())
    path.chmod(0o600)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", required=True)
    ap.add_argument("--salida", required=True)
    ap.add_argument("--roles-recibo", help="recibo AUT53 real de P; habilita SQL de asignaciones")
    ap.add_argument("--roles-plan", help="plan canónico preparado por CLI AUT53 en P")
    ap.add_argument("--roles-fuente-publicada", help="fuente de 16 cargos con la que P preparó el plan")
    a = ap.parse_args()
    exigir(bool(a.roles_recibo) == bool(a.roles_plan) == bool(a.roles_fuente_publicada), "recibo, plan y fuente AUT53 se requieren juntos")
    os.umask(0o077)
    config = json.loads(fuente(a.config).read_bytes())
    expected = {"version", "copia_ref", "organizacion_canonica_ref", "organizacion_procedencia_ref", "organizacion_procedencia_sha256", "organizacion_operativa_ref", "unidad_ref", "nodo_ref", "hmac_g2_clave_archivo", "certificados", "acuse_unidad_h10"}
    exigir(set(config) == expected and config["version"] == 1 and re.fullmatch(r"[a-z][a-z0-9_-]{2,40}", config["copia_ref"]), "configuración de copia P inválida")
    for field in ("organizacion_canonica_ref", "organizacion_procedencia_ref", "organizacion_operativa_ref", "unidad_ref", "nodo_ref"):
        exigir(isinstance(config[field], str) and REF.fullmatch(config[field]), "referencia P inválida")
    exigir(config["organizacion_canonica_ref"].startswith("org_") and config["organizacion_operativa_ref"].startswith("organizacion:") and HEX64.fullmatch(config["organizacion_procedencia_sha256"]), "ámbitos o procedencia P inválidos")
    unidad_recibo = json.loads(fuente(config["acuse_unidad_h10"]).read_bytes())
    unidad_h10 = unidad_recibo.get("recibo", {}).get("unidad", {})
    exigir(unidad_recibo.get("estado") == "permitido" and unidad_recibo.get("recibo", {}).get("esquema") == "vec.personal.unidad-inicial.v1" and unidad_recibo.get("recibo", {}).get("auditoria_ref") and unidad_recibo.get("recibo", {}).get("recibo_ref") and unidad_h10.get("organismo_ref") == config["organizacion_canonica_ref"] and unidad_h10.get("unidad_ref") == config["unidad_ref"] and unidad_h10.get("nodo_ref") == config["nodo_ref"] and unidad_h10.get("revision") == 1, "organización, unidad o nodo ajenos al H10 P")
    clave = fuente(config["hmac_g2_clave_archivo"]).read_bytes()
    exigir(len(clave) == 32 and any(clave), "clave HMAC g2 P inválida")
    folder = Path(a.salida)
    exigir(folder.is_absolute() and not folder.exists() and not folder.is_symlink(), "salida nueva absoluta requerida")
    identity = plantilla("identidad.template.sql").read_text()
    assigns = plantilla("asignaciones.template.sql").read_text()
    datos = json.loads(plantilla("firmantes.template.json").read_bytes())
    cargos = json.loads(plantilla("cargos-roles.template.json").read_bytes())
    now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
    start_role_at = now - dt.timedelta(hours=1)
    start_role = start_role_at.isoformat().replace("+00:00", "Z")
    end_role = (start_role_at + dt.timedelta(days=90)).isoformat().replace("+00:00", "Z")
    for item in cargos["cargos"]:
        item["organizacion_ref"] = config["organizacion_operativa_ref"]
        item["vigente_desde"] = start_role
        item["vigente_hasta"] = end_role
        item["duracion_propuesta_segundos"] = 90 * 86400
    if a.roles_recibo:
        cargos = json.loads(fuente(a.roles_fuente_publicada).read_bytes())
        source_rows = cargos.get("cargos", [])
        exigir(cargos.get("esquema") == "vec.admin.cargos-firma.cargos.v2" and len(source_rows) == 16 and all(x.get("organizacion_ref") == config["organizacion_operativa_ref"] and x.get("duracion_propuesta_segundos") == 90 * 86400 for x in source_rows), "fuente AUT53 P no corresponde al ámbito operativo de 90 días")
        exigir(len({x["vigente_desde"] for x in source_rows}) == 1 and len({x["vigente_hasta"] for x in source_rows}) == 1, "vigencia AUT53 ambigua")
        start_role, end_role = source_rows[0]["vigente_desde"], source_rows[0]["vigente_hasta"]
    assigns = assigns.replace("2026-10-14T17:16:34Z", start_role).replace("2027-11-18T16:06:24Z", end_role)
    replacements = {"__ORG_CANONICA__": config["organizacion_canonica_ref"], "__ORG_PROCEDENCIA__": config["organizacion_procedencia_ref"],
                    "__ORG_PROCEDENCIA_SHA__": config["organizacion_procedencia_sha256"], "__ORG_OPERATIVA__": config["organizacion_operativa_ref"],
                    "__UNIDAD__": config["unidad_ref"], "__EMISOR_SEMILLA__": "semilla:ensayo:datos-ficticios-" + config["copia_ref"],
                    "__REFERENCIA_SEMILLA__": "semilla:ensayo:" + config["copia_ref"]}
    for token, value in replacements.items():
        identity, assigns = identity.replace(token, value), assigns.replace(token, value)
    for index, role in enumerate(ROLES, 1):
        ref = datos["referencias"][role]
        cuenta, persona = ref["cuenta_ref"], ref["persona_ref"]
        for label, value in (("CUENTA", "desarrollo:" + cuenta), ("SUJETO", persona)):
            preimage = (DOMINIO + "\0" + ESPACIO + "\0" + ("cuenta" if label == "CUENTA" else "sujeto") + "\0" + value).encode()
            token = "__" + label + "_HMAC_" + str(index) + "__"
            exigir(identity.count(token) == 1, "plantilla HMAC divergente")
            identity = identity.replace(token, hmac.new(clave, preimage, hashlib.sha256).hexdigest())
    exigir("__" not in identity and "__" not in assigns, "plantilla incompleta")
    certs = config["certificados"]
    exigir(isinstance(certs, dict) and set(certs) == set(ROLES), "cuatro rutas de certificado requeridas")
    complete_certs = all(isinstance(certs[role], str) and certs[role] for role in ROLES)
    if complete_certs:
        retirada = dt.datetime.fromisoformat(datos["retirada_en"].replace("Z", "+00:00"))
        seconds = max(1, int((retirada - dt.datetime.now(dt.timezone.utc)).total_seconds()))
        exigir(seconds > 0, "vigencia retirada")
        for role in ROLES:
            path = fuente(certs[role])
            der = subprocess.run(["/usr/bin/openssl", "x509", "-in", str(path), "-outform", "DER"], check=True, capture_output=True).stdout
            expiry = subprocess.run(["/usr/bin/openssl", "x509", "-in", str(path), "-checkend", str(seconds), "-noout"], capture_output=True)
            exigir(expiry.returncode == 0, "certificado no cubre vigencia")
            sha = hashlib.sha256(der).hexdigest()
            datos["referencias"][role]["certificado"] = str(path)
            datos["referencias"][role]["certificado_sha256"] = sha
            perfil = datos["referencias"][role]["perfil"]["firma"]
            matches = [f for f in datos["firmantes"] if f["perfil_firma_ref"] == perfil]
            exigir(len(matches) == 1, "perfil firmante ambiguo")
            matches[0]["certificado_sha256"] = sha
    datos["organizacion_ca_ref"] = config["organizacion_canonica_ref"]
    datos["unidad_ref"] = config["unidad_ref"]
    datos["nodo_ref"] = config["nodo_ref"]
    roles = None
    if a.roles_recibo:
        plan_raw = fuente(a.roles_plan).read_bytes()
        plan = json.loads(plan_raw)
        exigir(len(plan.get("cargos", [])) == 16 and all({k: v for k, v in row.items() if k != "version_rol_sha256"} == src for row, src in zip(plan["cargos"], cargos["cargos"])), "plan AUT53 distinto de su fuente")
        receipt = json.loads(fuente(a.roles_recibo).read_bytes())
        body = receipt.get("recibo", {})
        roles = body.get("cargos", [])
        refs = {"rol:ct_cargo_" + role + "_" + scope + "_dev:v1" for role in ROLES for scope in ("firma", "consulta", "documentos", "circuito")}
        exigir(receipt.get("estado") == "permitido" and body.get("auditoria_ref") and body.get("plan_sha256") == hashlib.sha256(plan_raw).hexdigest() and len(roles) == 16 and {r.get("version_rol_ref") for r in roles} == refs and all(HEX64.fullmatch(r.get("version_rol_sha256", "")) and HEX64.fullmatch(r.get("control_sha256", "")) for r in roles), "recibo AUT53 P no confirma 16 roles")
        plan_hashes = {"rol:" + row["rol_id"] + ":v" + str(row["version"]): row["version_rol_sha256"] for row in plan["cargos"]}
        exigir(len(plan_hashes) == 16 and all(plan_hashes.get(row["version_rol_ref"]) == row["version_rol_sha256"] for row in roles), "huellas de rol AUT53 divergentes")
    folder.mkdir(mode=0o700)
    guardar(folder / "datos-identidad-firmantes.sql", identity.encode())
    guardar(folder / "cargos-firma-fuente.json", cargos)
    if roles is not None:
        guardar(folder / "semilla-asignaciones-firma.sql", assigns.encode())
        guardar(folder / "roles-cotejados.json", {"recibo_sha256": hashlib.sha256(fuente(a.roles_recibo).read_bytes()).hexdigest(), "roles": [{"version_ref": r["version_rol_ref"], "huella_sha256": r["version_rol_sha256"]} for r in roles]})
    guardar(folder / ("firmantes-datos.json" if complete_certs else "firmantes-datos.pendiente.json"), datos)
    guardar(folder / "preparacion.json", {"copia_ref": config["copia_ref"], "identidades_preparadas": 4, "roles_aut53_cotejados": roles is not None, "certificados_cotejados": complete_certs, "asignaciones_sql_emitidas": roles is not None, "actos_ejecutados": 0, "sql_ejecutadas": 0, "roles_vigente_desde": start_role, "roles_vigente_hasta": end_role})
    print("Semilla P preparada offline; SQL sin ejecutar y cero actos")


if __name__ == "__main__":
    main()
