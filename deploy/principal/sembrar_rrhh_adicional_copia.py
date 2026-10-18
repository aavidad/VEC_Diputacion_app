#!/usr/bin/env python3
"""Prepara la pertenencia sintética del segundo RRHH solo en una copia propia."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import tempfile

from deploy.principal.preparar_rrhh_adicional_copia import huella, leer_json, privado, referencia

REF = re.compile(r"^(?:per_|cta_|prf_|vca_)[A-Za-z0-9_-]{22,128}$")
ORG_CA37 = re.compile(r"^org_[a-z0-9]{16,80}$")
SHA = re.compile(r"^[0-9a-f]{64}$")
OPERACION = re.compile(r"^piis_[A-Za-z0-9_-]{22,123}$")
ORGANIZACION = "organizacion:desarrollo:dipgra"
UNIDAD = "unidad:desarrollo:rrhh"
ROL_BASE = "rol:tecnico_rrhh_desarrollo:v1"
ACCION_BASE = "contratacion_temporal.solicitud.crear"
ACCION_C22 = "contratacion_temporal.incorporacion.no_incorporacion"


def literal(valor):
    if not isinstance(valor, str) or "\x00" in valor:
        raise ValueError("Valor SQL inválido")
    return "'" + valor.replace("'", "''") + "'"


def comprobar_acuse(material: Path, ruta: Path):
    identidad = leer_json(material / "identidad/tecnico-rrhh-adicional.json")
    hoja = leer_json(material / "identidad/tecnico-rrhh-adicional-identidad.json")
    certificado = material / "mtls/tecnico-rrhh-adicional.crt"
    privado(certificado)
    certificado_sha = huella(certificado)
    if (identidad.get("certificado") != "mtls/tecnico-rrhh-adicional.crt"
            or identidad.get("identidad") != "identidad/tecnico-rrhh-adicional-identidad.json"
            or hoja.get("subject") != identidad.get("sujeto")
            or hoja.get("roles") != ["tecnico_rrhh"]
            or hoja.get("certificate_sha256") != certificado_sha):
        raise ValueError("La segunda hoja no coincide con la identidad declarada")
    acuse = leer_json(ruta)
    recibo = acuse.get("recibo")
    if acuse.get("estado") != "permitido" or not isinstance(recibo, dict):
        raise ValueError("Falta un recibo permitido de CA37")
    is_recibo, ca_recibo = recibo.get("is"), recibo.get("ca")
    if not isinstance(is_recibo, dict) or not isinstance(ca_recibo, dict):
        raise ValueError("El recibo CA37 no contiene ambas autoridades")
    datos_is, datos_ca = is_recibo.get("datos"), ca_recibo.get("datos")
    if not isinstance(datos_is, dict) or not isinstance(datos_ca, dict):
        raise ValueError("El recibo CA37 carece de titularidad")
    persona, cuenta = datos_ca.get("persona_ref"), datos_ca.get("cuenta_ordinaria_ref")
    operacion, plan_sha = recibo.get("operacion_ref"), recibo.get("plan_sha256")
    if (not isinstance(persona, str) or not REF.fullmatch(persona)
            or not isinstance(cuenta, str) or not REF.fullmatch(cuenta)
            or not isinstance(operacion, str) or not OPERACION.fullmatch(operacion)
            or not isinstance(plan_sha, str) or not SHA.fullmatch(plan_sha)
            or persona != identidad.get("persona_ref")
            or datos_is.get("persona_ref") != persona
            or datos_is.get("cuenta_ordinaria_ref") != cuenta
            or not isinstance(datos_ca.get("organizacion_ref"), str)
            or not ORG_CA37.fullmatch(datos_ca["organizacion_ref"])
            or datos_ca.get("persona_version") != 1
            or datos_ca.get("proyeccion_cuenta_version") != 1
            or datos_is.get("version_titularidad") != 1
            or recibo.get("esquema") != "vec.aut.fuentes-identidad-interna-sintetica.v1"
            or is_recibo.get("esquema") != "vec.is.identidad-interna-sintetica.v1"
            or ca_recibo.get("esquema") != "vec.ca.identidad-interna-sintetica.v1"
            or is_recibo.get("operacion_ref") != operacion
            or ca_recibo.get("operacion_ref") != operacion
            or is_recibo.get("plan_sha256") != plan_sha
            or ca_recibo.get("plan_sha256") != plan_sha
            or not str(ca_recibo.get("recibo_ref", "")).startswith("recibo_ca_identidad:")
            or not str(is_recibo.get("recibo_ref", "")).startswith("recibo_is_identidad:")):
        raise ValueError("El recibo CA37 no corresponde a la segunda persona y cuenta")
    base = identidad["sujeto"] + "\0" + certificado_sha
    perfil = identidad.get("perfil_ref")
    vinculo = referencia("vca_", base + "\0vinculo")
    if not isinstance(perfil, str) or not REF.fullmatch(perfil) or perfil == vinculo:
        raise ValueError("El perfil base no corresponde a la segunda hoja")
    if perfil != referencia("prf_", base + "\0perfil"):
        raise ValueError("El perfil base no está ligado a la segunda hoja")
    procedencia = referencia("prc_", base + "\0semilla-copia-ca37")
    fuente_sha = hashlib.sha256(("vec.segundo-rrhh.copia.v1\0" + operacion + "\0" + plan_sha + "\0" + certificado_sha).encode()).hexdigest()
    asignacion_id = "semilla_rrhh_" + hashlib.sha256((persona + "\0" + perfil + "\0" + certificado_sha).encode()).hexdigest()[:24]
    return {"version": 1, "autoridad": "no_autoritativo", "sujeto": identidad["sujeto"],
            "certificado_sha256": certificado_sha, "persona_ref": persona, "cuenta_ref": cuenta,
            "perfil_base_ref": perfil, "vinculo_base_ref": vinculo,
            "procedencia_ref": procedencia, "procedencia_sha256": fuente_sha,
            "asignacion_base_id": asignacion_id, "rol_base_ref": ROL_BASE,
            "ca37_operacion_ref": operacion, "ca37_plan_sha256": plan_sha,
            "ca37_recibo_ref": ca_recibo["recibo_ref"],
            "ca37_organizacion_ref": datos_ca["organizacion_ref"], "unidad_ref": UNIDAD}


def comprobar_plan_ca37(ruta: Path, semilla: dict, ahora=None):
    documento = leer_json(ruta)
    plan = documento.get("plan")
    if not isinstance(plan, dict) or documento.get("huella_plan_sha256") != semilla["ca37_plan_sha256"]:
        raise ValueError("El plan CA37 no corresponde al recibo")
    persona, organizacion = plan.get("persona"), plan.get("organizacion")
    if (not isinstance(persona, dict) or not isinstance(organizacion, dict)
            or plan.get("operacion_ref") != semilla["ca37_operacion_ref"]
            or persona.get("persona_ref") != semilla["persona_ref"]
            or organizacion.get("organizacion_ref") != semilla["ca37_organizacion_ref"]):
        raise ValueError("El plan CA37 contiene otra persona u organización")
    try:
        hasta = datetime.fromisoformat(persona["vigente_hasta"].replace("Z", "+00:00"))
        hasta_org = datetime.fromisoformat(organizacion["vigente_hasta"].replace("Z", "+00:00"))
    except (KeyError, AttributeError, ValueError) as exc:
        raise ValueError("La vigencia CA37 es inválida") from exc
    ahora = ahora or datetime.now(timezone.utc)
    segundos = (hasta - ahora).total_seconds()
    if hasta.tzinfo != timezone.utc or hasta_org.tzinfo != timezone.utc or hasta > hasta_org or not 60*86400 <= segundos <= 90*86400:
        raise ValueError("La vigencia CA37 debe cubrir de 60 a 90 días")
    return hasta


def sentencia(semilla: dict, confirmar: bool) -> bytes:
    d = {clave: literal(semilla[clave]) for clave in (
        "persona_ref", "cuenta_ref", "perfil_base_ref", "vinculo_base_ref",
        "procedencia_ref", "procedencia_sha256", "asignacion_base_id",
        "ca37_operacion_ref", "ca37_plan_sha256", "ca37_recibo_ref", "certificado_sha256")}
    documento = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='25s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:desarrollo:segundo-rrhh:semilla:v1',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contexto$
DECLARE c record; p record; efecto record; ahora timestamptz; hasta timestamptz;
BEGIN
 SELECT * INTO STRICT efecto FROM vec_contexto_actor_v1.efectos_identidad_interna_sintetica_v1
  WHERE operacion_ref={d['ca37_operacion_ref']} AND plan_sha256={d['ca37_plan_sha256']}
   AND recibo#>>'{{datos,cuenta_ordinaria_ref}}'={d['cuenta_ref']}
   AND recibo#>>'{{datos,persona_ref}}'={d['persona_ref']}
   AND recibo->>'recibo_ref'={d['ca37_recibo_ref']};
 SELECT v.* INTO STRICT c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
  WHERE a.cuenta_ref={d['cuenta_ref']} FOR UPDATE OF a;
 SELECT v.* INTO STRICT p FROM vec_contexto_actor_v1.persona_actual a
  JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
  WHERE a.persona_ref={d['persona_ref']} FOR UPDATE OF a;
 IF c.version<>1 OR p.version<>1 OR c.procedencia_autoridad<>'no_autoritativa'
  OR p.procedencia_autoridad<>'no_autoritativa'
  OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t
   WHERE t.cuenta_ref=c.cuenta_ref AND t.persona_ref=p.persona_ref AND t.tipo_operacion='identidad_interna_sintetica_v1'
    AND t.operacion_identidad_ref={d['ca37_operacion_ref']} AND t.alcance_fuente='sintetico_declarado')
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref={d['procedencia_ref']})
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref={d['perfil_base_ref']})
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref={d['vinculo_base_ref']})
 THEN RAISE EXCEPTION 'semilla RRHH: preimagen CA37 incompatible' USING ERRCODE='55000'; END IF;
 ahora:=clock_timestamp(); hasta:=LEAST(c.vigente_hasta,p.vigente_hasta,ahora+interval '89 days');
 IF hasta<ahora+interval '60 days' THEN RAISE EXCEPTION 'semilla RRHH: vigencia inferior a 60 dias' USING ERRCODE='55000'; END IF;
 PERFORM pg_catalog.set_config('vec.segundo_rrhh.semilla_hasta',hasta::text,true);
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES({d['procedencia_ref']},1,{d['procedencia_sha256']},'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES(c.cuenta_ref,2,{d['procedencia_ref']},1,{d['procedencia_sha256']},'autoridad_maestra_acreditada','activo',ahora,hasta);
 INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES(p.persona_ref,2,{d['procedencia_ref']},1,{d['procedencia_sha256']},'autoridad_maestra_acreditada','activo',ahora,hasta);
 UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2 WHERE cuenta_ref=c.cuenta_ref AND version=1;
 IF NOT FOUND THEN RAISE EXCEPTION 'semilla RRHH: CAS cuenta perdido' USING ERRCODE='40001'; END IF;
 UPDATE vec_contexto_actor_v1.persona_actual SET version=2 WHERE persona_ref=p.persona_ref AND version=1;
 IF NOT FOUND THEN RAISE EXCEPTION 'semilla RRHH: CAS persona perdido' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES({d['perfil_base_ref']},1,p.persona_ref,{d['procedencia_ref']},1,{d['procedencia_sha256']},'autoridad_maestra_acreditada','activo',ahora,hasta);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES({d['perfil_base_ref']},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES({d['vinculo_base_ref']},1,c.cuenta_ref,{d['perfil_base_ref']},p.persona_ref,{d['procedencia_ref']},1,{d['procedencia_sha256']},'autoridad_maestra_acreditada','activo',ahora,hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES({d['vinculo_base_ref']},1);
END $contexto$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $asignacion$
DECLARE r record; control record; doc jsonb; ahora timestamptz; hasta timestamptz; id text:={d['asignacion_base_id']};
BEGIN
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref='{ROL_BASE}';
 SELECT v.* INTO STRICT control FROM vec_autorizacion.control_vigencia_version_rol_actual a
  JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision)
  WHERE a.version_rol_ref='{ROL_BASE}';
 IF r.documento->>'estado'<>'publicada' OR control.estado<>'habilitada'
  OR jsonb_array_length(r.documento->'concesiones')<>1
  OR r.documento#>>'{{concesiones,0,accion}}'<>'{ACCION_BASE}'
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x->>'accion'='{ACCION_C22}')
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={d['perfil_base_ref']})
 THEN RAISE EXCEPTION 'semilla RRHH: rol base o asignación incompatible' USING ERRCODE='55000'; END IF;
 ahora:=clock_timestamp(); hasta:=current_setting('vec.segundo_rrhh.semilla_hasta')::timestamptz;
 IF hasta<ahora+interval '60 days' OR hasta>ahora+interval '89 days' THEN
  RAISE EXCEPTION 'semilla RRHH: asignacion fuera de vigencia' USING ERRCODE='55000'; END IF;
 doc:=jsonb_build_object('asignacion_id',id,'version',1,'perfil_activo_ref',{d['perfil_base_ref']},
  'principal_id',{d['persona_ref']},'version_rol_ref','{ROL_BASE}','estado','activa',
  'ambitos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array('{ORGANIZACION}')),
   jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array('{UNIDAD}'))),
  'emitida_por','semilla:desarrollo:rrhh-adicional','emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',to_char(hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revocada_en','0001-01-01T00:00:00Z');
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||id||':v1',id,1,{d['perfil_base_ref']},{d['persona_ref']},'{ROL_BASE}',
  encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),ahora,doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES({d['perfil_base_ref']},'asignacion:'||id||':v1',ahora,'semilla:desarrollo:rrhh-adicional','semilla:desarrollo:rrhh-adicional');
 IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
  JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref)
  JOIN vec_autorizacion.version_rol v USING(version_rol_ref)
  CROSS JOIN LATERAL jsonb_array_elements(v.documento->'concesiones') permiso
  WHERE x.principal_id={d['persona_ref']} AND permiso->>'accion'='{ACCION_C22}')
 THEN RAISE EXCEPTION 'semilla RRHH: c22 ya concedida' USING ERRCODE='42501'; END IF;
END $asignacion$;
RESET ROLE;
"""
    return (documento + ("COMMIT;\n" if confirmar else "ROLLBACK;\n")).encode()


def guardar_semilla(material: Path, dato: dict):
    destino = material / "identidad/tecnico-rrhh-adicional-semilla.json"
    contenido = (json.dumps(dato, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
    if destino.exists() or destino.is_symlink():
        privado(destino)
        if destino.read_bytes() != contenido:
            raise ValueError("La semilla retenida difiere de CA37")
        return
    descriptor, temporal = tempfile.mkstemp(prefix=".rrhh-semilla-", dir=destino.parent)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as salida:
            salida.write(contenido)
            salida.flush()
            os.fsync(salida.fileno())
        os.link(temporal, destino)
    finally:
        os.unlink(temporal)


def sembrar(copia, acuse: Path, plan: Path):
    copia.leer()
    if not copia.existe():
        raise ValueError("La copia propia no tiene PostgreSQL en marcha")
    privado(copia.material, True)
    dato = comprobar_acuse(copia.material, acuse)
    dato["ca37_vigente_hasta"] = comprobar_plan_ca37(plan, dato).isoformat().replace("+00:00", "Z")
    copia.sql_bytes(sentencia(dato, False), "segundo-rrhh-semilla-ensayo.log")
    copia.sql_bytes(sentencia(dato, True), "segundo-rrhh-semilla-aplicacion.log")
    guardar_semilla(copia.material, dato)
    return dato


def main():
    from deploy.principal.copia_local import Copia

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--equipo", required=True)
    parser.add_argument("--instancia", required=True)
    parser.add_argument("--acuse-ca37", required=True, type=Path)
    parser.add_argument("--plan-ca37", required=True, type=Path)
    args = parser.parse_args()
    if not re.fullmatch(r"[a-z][a-z0-9]{0,12}", args.equipo) or not re.fullmatch(r"[a-z0-9-]{1,30}", args.instancia):
        raise ValueError("La copia indicada no es válida")
    sembrar(Copia(args.equipo, args.instancia), args.acuse_ca37, args.plan_ca37)
    print("Pertenencia inicial sintética preparada; el perfil de no incorporación sigue pendiente de Administración.")


if __name__ == "__main__":
    main()
