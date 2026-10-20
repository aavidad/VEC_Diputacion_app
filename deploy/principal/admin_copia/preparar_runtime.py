#!/usr/bin/env python3
"""Conecta los recibos H10 confirmados con la preparación privada de vec-admin."""
import argparse
import datetime as dt
import hashlib
import json
import os
import re
from pathlib import Path

RUNTIME_TEXTOS = ("preparar_gobierno.py", "preparar_lote_plan.py", "preparar_relevo_logins.py",
                  "preparar_runtime.py", "preparar_vec_admin.py", "preparar_overlay_externo.py",
                  "plantilla-runtime.json")


def leer(path):
    return json.loads(Path(path).read_text())


def escribir(path, data):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(data, f, ensure_ascii=False, indent=1)
        f.write("\n")
        f.flush()
        os.fsync(f.fileno())


def unico(items, ref):
    found = [x for x in items if x["persona_ref"] == ref]
    if len(found) != 1:
        raise ValueError("persona del recibo ausente o repetida")
    return found[0]


def huella(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def validar_copia(run, kit, sid):
    """Coteja la entrada H10 y el kit compilado antes de derivar runtime."""
    run, kit = Path(run), Path(kit)
    if not run.is_absolute() or not kit.is_absolute() or not re.fullmatch(r"[0-9]{15,20}", str(sid)):
        raise ValueError("ruta o SID inválido")
    contrato = leer(kit / "contrato-copia-local-admin-v1.json")
    manifiesto = leer(kit / "manifiesto-kit.json")
    build = leer(kit / "build-config.json")
    arranque = leer(run / "contrato-arranque.json")
    if (contrato.get("version") != 2 or contrato.get("conjunto_capacidades") != 7
            or manifiesto.get("estado") != "compilado_sin_actos"
            or arranque.get("conjunto_capacidades") != 7
            or len({contrato.get("sid_copia"), manifiesto.get("sid_copia"),
                    build.get("sid_copia"), arranque.get("sid_copia"), sid}) != 1
            or len({contrato.get("revision_candidata"), manifiesto.get("revision"),
                    build.get("repo_revision"), arranque.get("revision")}) != 1
            or arranque.get("kit_manifest_sha256") != huella(kit / "manifiesto-kit.json")
            or arranque.get("kit_contract_sha256") != huella(kit / "contrato-copia-local-admin-v1.json")
            or arranque.get("config_sha256") != huella(run / "config.json")
            or manifiesto.get("contrato_sha256") != huella(kit / "contrato-copia-local-admin-v1.json")
            or manifiesto.get("fuente_sha256") != huella(kit / "source-manifest.json")
            or len({arranque.get("puerto_admin"), contrato.get("puerto_admin"), manifiesto.get("puerto_admin")}) != 1):
        raise ValueError("kit o arranque ajeno a la copia")
    if any(manifiesto.get("textos_sha256", {}).get(name) != huella(kit / name)
           for name in RUNTIME_TEXTOS):
        raise ValueError("preparador de runtime ajeno al manifiesto")
    return build, arranque


def validar_entrada(e, run, sid):
    if (e.get("runner") != str(run) or e.get("sid_copia") != sid
            or e.get("entorno") != "desarrollo"
            or not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", e.get("pg_base", ""))
            or not re.fullmatch(r"[a-z0-9.-]{1,120}", e.get("pg_host", ""))
            or type(e.get("pg_port")) is not int or not 1024 <= e["pg_port"] <= 65535
            or not re.fullmatch(r"[a-z0-9_]{3,18}", e.get("sufijo_login", ""))
            or not Path(e.get("pg_ca", "")).is_absolute()
            or not re.fullmatch(r"[a-z0-9._/-]+", e.get("pg_ca", ""))):
        raise ValueError("entrada de runtime ajena o inválida")


def validar_gobierno(run, dia, aplicado=False):
    """Coteja las tres huellas del plan y, tras aplicarlo, su recibo."""
    gov = Path(run) / "gobierno" / dia
    candidato = leer(gov / "aprobacion-candidata.json")
    plan = leer(gov / "plan.json")
    material = leer(gov / "material.json")
    cm = leer(gov / "configuracion-material.json")
    if (plan.get("version") != 2 or plan.get("conjunto_version") != 7
            or len(plan.get("clave_ordenes", [])) != 12
            or len(material.get("claves", [])) != 12
            or len({c["audiencia"] for c in material["claves"]}) != 12
            or candidato.get("plan_sha256") != huella(gov / "plan.json")
            or candidato.get("material_sha256") != huella(gov / "material.json")
            or candidato.get("preimagen_sha256") != plan.get("preimagen_sha256")
            or candidato.get("caduca_en") != plan.get("caduca_en")
            or plan.get("configuracion", {}).get("revision") != cm.get("Gobierno", {}).get("Revision")):
        raise ValueError("plan o material de gobierno divergente")
    if aplicado:
        acuse = leer(gov / "acuse-aplicar.json")
        recibo = acuse.get("recibo", {})
        if (acuse.get("estado") != "permitido" or not acuse.get("auditoria_intento")
                or not recibo.get("auditoria_ref")
                or recibo.get("plan_sha256") != candidato["plan_sha256"]
                or recibo.get("preimagen_sha256") != candidato["preimagen_sha256"]
                or recibo.get("material_sha256") != candidato["material_sha256"]
                or recibo.get("configuracion_ref") != plan["configuracion"]["revision"]):
            raise ValueError("recibo de gobierno ajeno al plan")
    return candidato, material, cm


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--run", required=True, type=Path, help="salida de configuracion.py confirmada por arranque.py")
    ap.add_argument("--kit", required=True, type=Path)
    ap.add_argument("--instancia", required=True)
    ap.add_argument("--dia-gobierno", required=True)
    ap.add_argument("--semilla-raiz", required=True, type=Path)
    ap.add_argument("--ca-pg", required=True, type=Path)
    ap.add_argument("--pg-host", required=True)
    ap.add_argument("--pg-base", required=True)
    ap.add_argument("--puerto-pg", required=True, type=int)
    ap.add_argument("--sid-copia", required=True)
    ap.add_argument("--vigente-hasta-config", required=True)
    ap.add_argument("--retirada-en", required=True)
    ap.add_argument("--redes", required=True)
    ap.add_argument("--motivo-publicado", required=True, type=Path)
    a = ap.parse_args()
    if not a.run.is_absolute() or not re.fullmatch(r"[a-z0-9_]{3,18}", a.instancia) or not re.fullmatch(r"[0-9]{8}[a-z0-9]{1,12}", a.dia_gobierno):
        ap.error("rutas o identificadores inválidos")
    if (not 1024 <= a.puerto_pg <= 65535 or not re.fullmatch(r"[a-z0-9.-]{1,120}", a.pg_host)
            or not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", a.pg_base)):
        ap.error("puerto inválido")
    try:
        build, arranque = validar_copia(a.run, a.kit, a.sid_copia)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        ap.error(str(exc))
    for p in (a.semilla_raiz, a.ca_pg, a.motivo_publicado):
        if not p.is_absolute() or not p.is_file() or p.is_symlink():
            ap.error("material privado ausente")
    if not re.fullmatch(r"20[0-9]{2}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z", a.vigente_hasta_config) or not re.fullmatch(r"20[0-9]{2}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z", a.retirada_en):
        ap.error("vigencias inválidas")
    try:
        vigente = dt.datetime.fromisoformat(a.vigente_hasta_config.replace("Z", "+00:00"))
        retirada = dt.datetime.fromisoformat(a.retirada_en.replace("Z", "+00:00"))
    except ValueError:
        ap.error("vigencias inválidas")
    if not dt.datetime.now(dt.timezone.utc) < vigente < retirada:
        ap.error("orden de vigencias inválido")
    if not re.fullmatch(r"[a-z0-9:./,_-]+", a.redes):
        ap.error("redes inválidas")
    motivo = leer(a.motivo_publicado)
    if (set(motivo) != {"catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"}
            or not re.fullmatch(r"[0-9a-f]{64}", motivo.get("catalogo_huella_sha256", ""))
            or type(motivo.get("catalogo_version")) is not int or motivo["catalogo_version"] < 1):
        ap.error("motivo publicado inválido")
    for phase in ("fuentes", "unidad", "bootstrap"):
        receipt = leer(a.run / phase / "acuse-primero.json")
        if (receipt.get("estado") != "permitido" or not receipt.get("auditoria_intento")
                or not isinstance(receipt.get("recibo"), dict) or not receipt["recibo"].get("auditoria_ref")):
            ap.error("fase H10 sin recibo confirmado")

    config = leer(a.run / "config.json")
    originals = leer(a.run / "fuentes/originales.json")["personas"]
    plan = leer(a.run / "bootstrap/plan-v3.json")["plan"]["personas"]
    certs = leer(a.run / "fuentes/certificados.json")["certificados"]
    material = leer(a.run / "fuentes/material-hmac.raw")
    source = leer(a.run / "fuentes/fuente.json")
    accounts = leer(a.run / "fuentes/acuse-primero.json")["recibo"]["is"]["datos"]["personas"]
    if (source["fuente_hmac"]["huella_sha256"] != huella(a.run / "fuentes/material-hmac.raw")
            or {x["persona_ref"] for x in source["personas"]} != {x["persona_ref"] for x in originals}
            or source["politica_admin"]["ca_sha256"] != huella(a.run / "fuentes/ca.der")):
        ap.error("material de fuentes divergente")
    if (len(originals) != 2 or len(plan) != 2 or len(certs) != 2 or len(accounts) != 2
            or len({x["persona_ref"] for x in originals}) != 2
            or len({x["sujeto_original"] for x in originals}) != 2
            or len({x["cert_der_sha256"] for x in certs}) != 2
            or len({x["cuenta_ref"] for x in plan}) != 2
            or len({x["cuenta_ordinaria_ref"] for x in accounts}) != 2):
        ap.error("se esperaban dos personas ADMIN distintas")
    rows = []
    for i, o in enumerate(originals, 1):
        ref = o["persona_ref"]
        p, cert, account = unico(plan, ref), unico(certs, ref), unico(accounts, ref)
        if (p["certificado_admin"]["huella_sha256"] != cert["cert_der_sha256"]
                or p["certificado_admin"]["ca_huella_sha256"] != cert["ca_der_sha256"]
                or p["cuenta_ref"] != account["cuenta_privilegiada_ref"]
                or cert["cert_der_sha256"] != huella(a.run / "fuentes" / f"cliente-{i}.der")):
            ap.error("certificado no ligado al plan")
        rows.append({"persona_ref": ref, "cuenta_ref": p["cuenta_ref"], "cuenta_ordinaria_ref": account["cuenta_ordinaria_ref"],
                     "certificado_sha256": cert["cert_der_sha256"], "ca_sha256": cert["ca_der_sha256"],
                     "espacio_identidad": config["proveedor"]["espacio_identidad"], "dominio_hmac_ref": material["dominio_hmac_ref"],
                     "clave_hmac_id": material["clave_hmac_id"], "clave_hmac_version": material["clave_hmac_version"],
                     "fuente_ref": material["fuente_ref"], "fuente_sha256": source["fuente_hmac"]["huella_sha256"],
                     "sujeto_id": o["sujeto_original"], "cuenta_id": o["cuenta_privilegiada_original"],
                     "cuenta_ordinaria_id": o["cuenta_ordinaria_original"]})

    v = a.run / "vec-admin"
    if v.exists():
        ap.error("vec-admin ya existe: no se sobrescribe")
    (v / "identificadores").mkdir(parents=True, mode=0o700)
    v.chmod(0o700)
    escribir(v / "identificadores/identificadores.json", {"version": 1, "entradas": rows})
    e = leer(a.kit / "plantilla-runtime.json")
    e["vigente_hasta_config"] = a.vigente_hasta_config
    e["retirada_en"] = a.retirada_en
    e["host"] = build["host_admin"]
    e.update({"directorio_vec_admin": str(v), "runner": str(a.run), "kit_dir": str(a.kit), "checkout": str(a.kit / "source"),
              "pg_host": a.pg_host, "pg_port": a.puerto_pg, "pg_base": a.pg_base, "pg_ca": str(a.ca_pg),
              "sufijo_login": a.instancia, "gobierno_dir": str(a.run / "gobierno"), "semilla_raiz": str(a.semilla_raiz),
              "identificadores": str(v / "identificadores/identificadores.json"), "escucha": f"127.0.0.1:{arranque['puerto_admin']}"})
    e["sid_copia"] = a.sid_copia
    e["dia_gobierno"] = a.dia_gobierno
    e["redes"] = a.redes
    e["motivo"] = motivo
    e["audiencia"] = config["bootstrap"]["audiencia_administrativa"]
    e["espacio_identidad"] = config["proveedor"]["espacio_identidad"]
    e["emisor_identidad"] = config["proveedor"]["espacio_identidad"]
    e["identidad"].update({k: config["proveedor"][k] for k in ("espacio_identidad", "dominio_ref", "espacio_clave", "dominio_hmac")})
    e["identidad"]["directorio_material"] = config["proveedor"]["directorio_material"]
    e["identidad"]["ruta_configuracion_hmac"] = config["proveedor"]["ruta_configuracion_hmac"]
    escribir(a.run / "entrada-vec-admin.json", e)
    print("Entrada de vec-admin preparada; ejecutar las fases del kit y el SQL de sus pools")


if __name__ == "__main__":
    main()
