#!/usr/bin/env python3
"""Activa a Elena, técnica ficticia de Bolsa, solo en la copia local O."""

import argparse
import hashlib
import hmac
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tempfile


SUJETO = "desarrollo:elena-ruiz-martin-bolsa"
NOMBRE = "Elena Ruiz Martín"
ROL_ID = "tecnica_bolsa_inscripcion_desarrollo"
ROL_REF = "rol:" + ROL_ID + ":v1"
OPERADOR = "operacion:ensayo:tecnica-bolsa"
MARCA = "2026-10-14T00:00:00Z"
HASTA = "2027-10-01T00:00:00Z"
ORGANIZACION = "organizacion:desarrollo:dipgra"
UNIDAD = "unidad:desarrollo:rrhh"
AMBITO = "ambito:desarrollo:bolsa"
ESPACIO = "https://localhost/vec/desarrollo/identidad"
DOMINIO_HMAC = "vec.identidad.desarrollo.hmac.v1"
FUENTES = (
    ("bolsa_lecturas_rrhh_b103_v1.json", ("bolsa.inscripcion.rrhh.listar",
      "bolsa.inscripcion.rrhh.consultar", "bolsa.inscripcion.rrhh.convocatorias.listar")),
    ("../inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json",
     ("bolsa.inscripcion.rrhh.motivos", "bolsa.inscripcion.rrhh.decidir",
      "bolsa.inscripcion.rrhh.incorporar")),
    ("bolsa_alegaciones_propias_v1.json", ("bolsa.inscripcion.alegacion.rrhh.consultar",)),
    ("meritos_revision_rrhh_v1.json", ("meritos.hecho.revision_rrhh.consultar",)),
)


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(path, directorio=False):
    st = path.lstat()
    exigir(st.st_uid == os.getuid() and not path.is_symlink() and not st.st_mode & 0o077 and
           (stat.S_ISDIR(st.st_mode) if directorio else stat.S_ISREG(st.st_mode)), "material_no_privado")


def leer_json(path):
    privado(path)
    def objeto_unico(pares):
        dato = {}
        for clave, valor in pares:
            exigir(clave not in dato, "json_privado_duplicado")
            dato[clave] = valor
        return dato
    return json.loads(path.read_text(), object_pairs_hook=objeto_unico)


def sql_lit(value):
    exigir(isinstance(value, str) and "\x00" not in value, "literal_invalido")
    return "'" + value.replace("'", "''") + "'"


def sql_json(value):
    return sql_lit(json.dumps(value, ensure_ascii=False, separators=(",", ":"))) + "::jsonb"


def psql(pg, sql):
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
                       input=sql, text=True, capture_output=True, timeout=40, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        raise ValueError("sql_tecnica_bolsa_" + (codigo.group(0) if codigo else "rechazado"))
    return r.stdout.strip()


def comando(args, entrada=None):
    r = subprocess.run(args, input=entrada, capture_output=True, timeout=30, check=False)
    exigir(r.returncode == 0, "certificado_invalido")
    return r.stdout


def referencia(prefijo, material):
    digest = hashlib.sha256(b"vec.ct.alta.desarrollo.v1\0" + material.encode()).digest()
    return prefijo + digest[:16].hex()


def fuentes(repo):
    base = repo / "deploy/principal/fuentes"
    concesiones = []
    for nombre, acciones in FUENTES:
        fuente = json.loads((base / nombre).read_text())
        canon = fuente["entradas_canon"]
        esperadas = [{k: v for k, v in e.items() if k != "fuente_huella_sha256"}
                     for e in fuente["entradas"]]
        exigir(hashlib.sha256(canon.encode()).hexdigest() == fuente["huella_sha256"] and
               json.loads(canon) == esperadas and
               all(e["fuente_huella_sha256"] == fuente["huella_sha256"] for e in fuente["entradas"]),
               "fuente_divergente")
        por_accion = {e["concesion"]["accion"]: e for e in fuente["entradas"]}
        exigir(all(a in por_accion and por_accion[a]["clase_control"] == "ordinario" and
                   por_accion[a]["dimensiones_ambito"] == ["unidad_ref", "ambito_ref"] for a in acciones),
               "acciones_tecnica_no_publicadas")
        concesiones.extend(por_accion[a]["concesion"] for a in acciones)
    exigir(len(concesiones) == 8 and len({x["accion"] for x in concesiones}) == 8,
           "concesiones_tecnica_divergentes")
    return concesiones


def certificado(material, aplicar):
    mtls = material / "mtls"
    privado(mtls, True)
    paquete = mtls / "elena-bolsa"
    cert, key = paquete / "cliente.crt", paquete / "cliente.key"
    ca, ca_key = material / "ca/ca.crt", material / "ca/ca.key"
    privado(ca)
    privado(ca_key)
    if paquete.exists():
        privado(paquete, True)
        exigir(cert.exists() and key.exists(), "certificado_tecnica_parcial")
    else:
        exigir(aplicar, "certificado_tecnica_pendiente")
        with tempfile.TemporaryDirectory(prefix=".elena-bolsa-", dir=mtls) as temporal:
            tmp = Path(temporal)
            (tmp / "ext").write_text("basicConstraints=critical,CA:FALSE\n"
                                      "keyUsage=critical,digitalSignature\n"
                                      "extendedKeyUsage=clientAuth\n"
                                      "subjectAltName=URI:urn:vec:desarrollo:elena-ruiz-martin-bolsa\n")
            comando(["openssl", "req", "-new", "-newkey", "rsa:3072", "-nodes", "-sha256",
                     "-subj", "/CN=elena-ruiz-martin-bolsa/O=VEC desarrollo",
                     "-keyout", str(tmp / "key"), "-out", str(tmp / "csr")])
            comando(["openssl", "x509", "-req", "-sha256", "-days", "365", "-in", str(tmp / "csr"),
                     "-CA", str(ca), "-CAkey", str(ca_key), "-set_serial",
                     "0x" + os.urandom(16).hex(), "-extfile", str(tmp / "ext"), "-out", str(tmp / "cert")])
            os.chmod(tmp / "key", 0o600)
            os.chmod(tmp / "cert", 0o600)
            os.rename(tmp / "key", tmp / "cliente.key")
            os.rename(tmp / "cert", tmp / "cliente.crt")
            (tmp / "csr").unlink()
            (tmp / "ext").unlink()
            os.rename(tmp, paquete)
    privado(cert)
    privado(key)
    comando(["openssl", "verify", "-purpose", "sslclient", "-CAfile", str(ca), str(cert)])
    comando(["openssl", "x509", "-in", str(cert), "-checkend", "0", "-noout"])
    der = comando(["openssl", "x509", "-in", str(cert), "-outform", "DER"])
    pub_cert = comando(["openssl", "x509", "-in", str(cert), "-pubkey", "-noout"])
    pub_key = comando(["openssl", "pkey", "-in", str(key), "-pubout"])
    exigir(pub_cert == pub_key and b"elena-ruiz-martin-bolsa" in
           comando(["openssl", "x509", "-in", str(cert), "-noout", "-subject"]),
           "certificado_tecnica_no_corresponde")
    return hashlib.sha256(der).hexdigest()


def datos(huella, concesiones):
    base = SUJETO + "\0" + huella
    persona = referencia("per_", base + "\0persona")
    perfil = referencia("prf_", base + "\0perfil")
    vinculo = referencia("vca_", base + "\0vinculo")
    asignacion_id = referencia("asg_", base + "\0asignacion-bolsa")
    procedencia = referencia("prc_", base + "\0procedencia")
    procedencia_huella = hashlib.sha256(("ensayo:elena-bolsa\0" + base).encode()).hexdigest()
    rol = dict(rol_id=ROL_ID, version=1, nombre="tecnicaBolsa.inscripcion.rol",
               estado="publicada", concesiones=concesiones, publicada_por=OPERADOR,
               publicada_en=MARCA, retirada_en="0001-01-01T00:00:00Z")
    control = dict(version_rol_ref=ROL_REF, revision=1, estado="habilitada",
                   actualizado_por=OPERADOR, actualizado_en=MARCA)
    asig = dict(asignacion_id=asignacion_id, version=1, perfil_activo_ref=perfil,
                principal_id=persona, version_rol_ref=ROL_REF, estado="activa",
                ambitos=[{"clave": "organizacion_ref", "valores": [ORGANIZACION]},
                         {"clave": "unidad_ref", "valores": [UNIDAD]},
                         {"clave": "ambito_ref", "valores": [AMBITO]}],
                emitida_por=OPERADOR, emitida_en=MARCA, vigente_desde=MARCA,
                vigente_hasta=HASTA, revocada_en="0001-01-01T00:00:00Z")
    return dict(huella=huella, persona=persona, perfil=perfil, vinculo=vinculo,
                procedencia=procedencia, procedencia_huella=procedencia_huella,
                asig_ref="asignacion:" + asignacion_id + ":v1", rol=rol, control=control, asig=asig)


def hmac_cuenta(material, cuenta, huella):
    config = leer_json(material / "idempotencia/configuracion.json")
    exigir(config["version"] == 1 and config["autoridad"] == "no_autoritativo" and
           config["generaciones"][0]["generacion"] == 2, "hmac_identidad_divergente")
    key = material / "idempotencia/g2-huella-solicitud.bin"
    privado(key)
    secreto = key.read_bytes()
    exigir(len(secreto) == 32, "hmac_identidad_invalido")
    def digerir(tipo, valor):
        preimagen = (DOMINIO_HMAC + "\0" + ESPACIO + "\0" + tipo + "\0" + valor).encode()
        return hmac.new(secreto, preimagen, hashlib.sha256).hexdigest()
    return digerir("cuenta", "desarrollo:" + cuenta), digerir("sujeto", SUJETO)


def publicar(pg, material, d):
    # El LOGIN y el registro de cuenta son los ya publicados; todo queda en una TX.
    idh = referencia("idh_", ESPACIO)
    operacion = referencia("opr_", SUJETO + "\0" + d["huella"] + "\0provision")
    cuenta_esperada = referencia("cta_", SUJETO + "\0" + d["huella"] + "\0cuenta")
    ch, sh = hmac_cuenta(material, cuenta_esperada, d["huella"])
    # La cuenta de la composición interna es determinista; la función nominal
    # genera una aleatoria. Aquí se publica la fila sintética exacta de la copia.
    q = lambda x: sql_lit(x)
    rol, control, asig = (sql_json(d[x]) for x in ("rol", "control", "asig"))
    publicaciones_bolsa = "\n".join(
        " IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r "
        "CROSS JOIN LATERAL jsonb_array_elements(r.documento->'concesiones') x "
        "WHERE r.rol_id IN ('tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo',"
        "'tecnico_rrhh_desarrollo') AND x=" + sql_json(grant) + ") "
        "THEN RAISE EXCEPTION 'concesion_bolsa_no_publicada' USING ERRCODE='55000'; END IF;"
        for grant in d["rol"]["concesiones"][:-1]
    )
    sql = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:elena-bolsa',0));
DO $publicar$ DECLARE existente integer; BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref='cta_ab6d8d5ecba47d2ecaf2acd350584cd9')
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
               JOIN vec_autorizacion.asignacion_perfil p USING(asignacion_ref)
               WHERE a.perfil_activo_ref='prf_5d140bb82db5f22e5ea49d54c9eb75ba'
               AND a.asignacion_ref='asignacion:asg_80e5ca1a22981c65cbc94a616b246bb1:v3'
               AND p.version_rol_ref='rol:tecnico_rrhh_desarrollo:v2')
 THEN RAISE EXCEPTION 'preimagen_carmen_ausente' USING ERRCODE='55000'; END IF;
 {publicaciones_bolsa}
 SELECT count(*) INTO existente FROM vec_autorizacion.version_rol WHERE version_rol_ref={q(ROL_REF)};
 IF existente=1 THEN
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol
      WHERE version_rol_ref={q(ROL_REF)} AND documento={rol}
      AND huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({rol}),'UTF8')),'hex'))
   OR NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta c
      JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ca USING(cuenta_ref)
      JOIN vec_identidad_sesiones_v1.estado_cuenta e ON e.cuenta_ref=ca.cuenta_ref AND e.revision=ca.revision
      JOIN vec_identidad_sesiones_v1.alias_hmac_cuenta al ON al.cuenta_ref=c.cuenta_ref
      WHERE c.cuenta_ref={q(cuenta_esperada)} AND c.acto_ref={q(operacion)}
      AND NOT c.cuenta_privilegiada AND e.estado='activa'
      AND al.dominio_hmac_ref={q(idh)} AND al.clave_hmac_id=('vec.identidad.desarrollo.g2')
      AND al.clave_hmac_version=2 AND al.cuenta_id_hmac=decode({q(ch)},'hex')
      AND al.sujeto_id_hmac=decode({q(sh)},'hex'))
   OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual ca
      JOIN vec_autorizacion.control_vigencia_version_rol c
      ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
      WHERE ca.version_rol_ref={q(ROL_REF)} AND c.documento={control}
      AND c.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({control}),'UTF8')),'hex'))
   OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
      JOIN vec_autorizacion.asignacion_perfil p USING(asignacion_ref)
      WHERE a.perfil_activo_ref={q(d['perfil'])} AND a.asignacion_ref={q(d['asig_ref'])}
        AND p.documento={asig}
        AND p.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({asig}),'UTF8')),'hex'))
   OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
      WHERE a.vinculo_ref={q(d['vinculo'])} AND v.perfil_ref={q(d['perfil'])}
        AND v.persona_ref={q(d['persona'])} AND v.cuenta_ref={q(cuenta_esperada)})
  THEN RAISE EXCEPTION 'elena_publicada_divergente' USING ERRCODE='55000'; END IF;
  RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref={q(cuenta_esperada)})
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref={q(d['persona'])})
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref={q(d['perfil'])})
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={q(d['perfil'])})
 THEN RAISE EXCEPTION 'elena_preimagen_parcial' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_identidad_sesiones_v1.cuenta VALUES({q(cuenta_esperada)},false,NULL,{q(MARCA)}::timestamptz,{q(operacion)});
 INSERT INTO vec_identidad_sesiones_v1.alias_hmac_cuenta
  VALUES({q(referencia('ali_',SUJETO+'\0'+d['huella']+'\0alias'))},{q(cuenta_esperada)},
  'vec.identidad.hmac-sha256.v1',{q(idh)},'vec.identidad.desarrollo.g2',2,
  decode({q(ch)},'hex'),decode({q(sh)},'hex'),{q(MARCA)}::timestamptz,{q(operacion)});
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta VALUES({q(cuenta_esperada)},1,'activa',{q(MARCA)}::timestamptz,{q(operacion)});
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual VALUES({q(cuenta_esperada)},1,{q(MARCA)}::timestamptz,{q(operacion)});
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES({q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES({q(cuenta_esperada)},1,{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(MARCA)}::timestamptz,{q(HASTA)}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES({q(cuenta_esperada)},1);
 INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES({q(d['persona'])},1,{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(MARCA)}::timestamptz,{q(HASTA)}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES({q(d['persona'])},1);
 INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES({q(d['perfil'])},1,{q(d['persona'])},{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(MARCA)}::timestamptz,{q(HASTA)}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES({q(d['perfil'])},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES({q(d['vinculo'])},1,{q(cuenta_esperada)},{q(d['perfil'])},{q(d['persona'])},{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(MARCA)}::timestamptz,{q(HASTA)}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES({q(d['vinculo'])},1);
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 SELECT {q(ROL_REF)},{q(ROL_ID)},1,encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({rol}),'UTF8')),'hex'),{q(MARCA)}::timestamptz,{rol};
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 SELECT {q(ROL_REF)},1,'habilitada',encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({control}),'UTF8')),'hex'),{q(MARCA)}::timestamptz,{control};
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES({q(ROL_REF)},1,{q(MARCA)}::timestamptz,{q(OPERADOR)},'acto:ensayo:elena-bolsa:control:v1');
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT {q(d['asig_ref'])},{q(d['asig']['asignacion_id'])},1,{q(d['perfil'])},{q(d['persona'])},{q(ROL_REF)},
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({asig}),'UTF8')),'hex'),{q(MARCA)}::timestamptz,{asig};
 INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES({q(d['perfil'])},{q(d['asig_ref'])},{q(MARCA)}::timestamptz,{q(OPERADOR)},'acto:ensayo:elena-bolsa:asignacion:v1');
END $publicar$;
COMMIT;
SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={q(d['perfil'])} AND asignacion_ref={q(d['asig_ref'])};"""
    exigir(psql(pg, sql).splitlines()[-1] == "1", "tecnica_sin_recibo_publicacion")
    return cuenta_esperada


def comprobar_publicado(pg, material, d, cuenta):
    ch, sh = hmac_cuenta(material, cuenta, d["huella"])
    q = sql_lit
    rol, control, asig = (sql_json(d[x]) for x in ("rol", "control", "asig"))
    sql = f"""SELECT CASE WHEN
 EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta c
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ca USING(cuenta_ref)
  JOIN vec_identidad_sesiones_v1.estado_cuenta e ON e.cuenta_ref=ca.cuenta_ref AND e.revision=ca.revision
  JOIN vec_identidad_sesiones_v1.alias_hmac_cuenta al ON al.cuenta_ref=c.cuenta_ref
  WHERE c.cuenta_ref={q(cuenta)} AND NOT c.cuenta_privilegiada AND e.estado='activa'
  AND al.esquema_hmac='vec.identidad.hmac-sha256.v1'
  AND al.dominio_hmac_ref={q(referencia('idh_',ESPACIO))}
  AND al.clave_hmac_id=('vec.identidad.desarrollo.g2') AND al.clave_hmac_version=2
  AND al.cuenta_id_hmac=decode({q(ch)},'hex') AND al.sujeto_id_hmac=decode({q(sh)},'hex'))
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias p
  WHERE p.procedencia_ref={q(d['procedencia'])} AND p.procedencia_version=1
  AND p.procedencia_huella_sha256={q(d['procedencia_huella'])}
  AND p.procedencia_autoridad='autoridad_maestra_acreditada')
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
  WHERE a.cuenta_ref={q(cuenta)} AND a.version=1 AND v.estado='activo'
  AND v.procedencia_ref={q(d['procedencia'])}
  AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta)
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual a
  JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
  WHERE a.persona_ref={q(d['persona'])} AND a.version=1 AND v.estado='activo'
  AND v.procedencia_ref={q(d['procedencia'])}
  AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta)
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a
  JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
  WHERE a.perfil_ref={q(d['perfil'])} AND a.version=1 AND v.estado='activo'
  AND v.persona_ref={q(d['persona'])} AND v.procedencia_ref={q(d['procedencia'])}
  AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta)
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  WHERE a.vinculo_ref={q(d['vinculo'])} AND a.version=1 AND v.estado='activo'
  AND v.cuenta_ref={q(cuenta)} AND v.persona_ref={q(d['persona'])}
  AND v.perfil_ref={q(d['perfil'])} AND v.procedencia_ref={q(d['procedencia'])}
  AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta)
 AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol c
  ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
  WHERE r.version_rol_ref={q(ROL_REF)} AND r.documento={rol}
  AND r.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({rol}),'UTF8')),'hex')
  AND c.documento={control} AND c.estado='habilitada'
  AND c.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({control}),'UTF8')),'hex'))
 AND EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
  JOIN vec_autorizacion.asignacion_perfil p USING(asignacion_ref)
  WHERE a.perfil_activo_ref={q(d['perfil'])} AND a.asignacion_ref={q(d['asig_ref'])}
  AND p.documento={asig}
  AND p.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({asig}),'UTF8')),'hex')
  AND clock_timestamp()>=(p.documento->>'vigente_desde')::timestamptz
  AND clock_timestamp()<(p.documento->>'vigente_hasta')::timestamptz)
 THEN 1 ELSE 0 END;"""
    exigir(psql(pg, sql) == "1", "tecnica_bolsa_postimagen_divergente")


def guardar_json(path, dato):
    fd, temp = tempfile.mkstemp(prefix=".elena-bolsa-", dir=path.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w") as salida:
            json.dump(dato, salida, ensure_ascii=False, separators=(",", ":"))
            salida.write("\n")
            salida.flush()
            os.fsync(salida.fileno())
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def manifiestos(material, d, cuenta, aplicar):
    identidad = material / "identidad"
    privado(identidad, True)
    entrada = dict(certificate="mtls/elena-bolsa/cliente.crt", identity="identidad/elena-bolsa.json",
                   subject=SUJETO, perfil_ref=d["perfil"], organizacion_ref=ORGANIZACION,
                   clase_ambito="organizacion", ambito_ref=ORGANIZACION)
    entry_cuenta = dict(certificado_sha256=d["huella"], sujeto=SUJETO,
                        cuenta_ref=cuenta, perfil_ref=d["perfil"])
    ruta_id = identidad / "elena-bolsa.json"
    id_doc = dict(version=1, autoridad="no_autoritativo", certificate_sha256=d["huella"],
                  subject=SUJETO, display_name=NOMBRE, roles=["lector_rrhh"])
    if ruta_id.exists():
        exigir(leer_json(ruta_id) == id_doc, "identidad_elena_divergente")
    else:
        exigir(aplicar, "identidad_elena_pendiente")
        guardar_json(ruta_id, id_doc)
    ct = leer_json(identidad / "consultas-rrhh.json")
    exigir(set(ct) == {"version", "autoridad", "entradas"} and
           ct.get("version") == 1 and ct.get("autoridad") == "no_autoritativo" and
           len(ct.get("entradas", [])) == 1 and
           set(ct["entradas"][0]) == {"certificate", "identity", "subject", "perfil_ref",
                                      "organizacion_ref", "clase_ambito", "ambito_ref"} and
           ct["entradas"][0].get("subject") == "desarrollo:operador-rrhh",
           "lector_ct_no_separado")
    ruta = identidad / "inscripcion-bolsa-rrhh.json"
    doc = leer_json(ruta)
    claves = {"version", "autoridad", "cuentas", "lectores", "dsn_registro_identidad",
              "dsn_revalidacion_identidad", "dsn_contexto", "dsn_fuente_autorizacion",
              "dsn_registro_autorizacion", "dsn_motivos", "motivos"}
    exigir(set(doc) in (claves, claves - {"lectores"}) and doc["version"] == 1 and
           doc["autoridad"] == "no_autoritativo" and
           all(isinstance(doc[k], str) and doc[k] for k in claves if k.startswith("dsn_")) and
           isinstance(doc["motivos"], dict) and set(doc["motivos"]) == {"decidir", "incorporar"} and
           all(isinstance(v, dict) and set(v) == {"catalogo_id", "catalogo_version",
                                                   "catalogo_huella_sha256", "entrada_clave"} and
               isinstance(v["catalogo_id"], str) and v["catalogo_id"] and
               isinstance(v["catalogo_version"], int) and v["catalogo_version"] > 0 and
               isinstance(v["catalogo_huella_sha256"], str) and
               re.fullmatch(r"[0-9a-f]{64}", v["catalogo_huella_sha256"]) and
               isinstance(v["entrada_clave"], str) and v["entrada_clave"]
               for v in doc["motivos"].values()) and
           isinstance(doc["cuentas"], list) and len(doc["cuentas"]) in range(1, 65) and
           all(isinstance(c, dict) and set(c) == {"certificado_sha256", "sujeto", "cuenta_ref", "perfil_ref"}
               for c in doc["cuentas"]),
           "manifiesto_inscripcion_invalido")
    cuentas = doc["cuentas"]
    existentes = [x for x in cuentas if x.get("sujeto") == SUJETO]
    exigir(len(existentes) <= 1 and (not existentes or existentes[0] == entry_cuenta) and
           (bool(existentes) or len(cuentas) < 64) and
           len([x for x in cuentas if x.get("sujeto") == "desarrollo:operador-rrhh"]) == 1,
           "cuentas_inscripcion_divergentes")
    lectores = doc.get("lectores", [])
    exigir(lectores in ([], [entrada]), "lectores_inscripcion_divergentes")
    if not existentes or not lectores:
        exigir(aplicar, "manifiesto_inscripcion_pendiente")
        if not existentes:
            cuentas.append(entry_cuenta)
        doc["lectores"] = [entrada]
        guardar_json(ruta, doc)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo", type=Path, required=True)
    p.add_argument("--pg", required=True)
    p.add_argument("--aplicar", action="store_true")
    a = p.parse_args()
    repo = Path(__file__).resolve().parents[4]
    exigir(a.ensayo.is_absolute() and a.ensayo.name == "vec-ensayo-codexo-aspirante-20261014" and
           repo not in a.ensayo.parents and a.pg == "vec-codexo-aspirante-pg-20261014",
           "solo_copia_o")
    privado(a.ensayo, True)
    material = a.ensayo / "material"
    privado(material, True)
    concesiones = fuentes(repo)
    huella = certificado(material, a.aplicar)
    d = datos(huella, concesiones)
    cuenta = referencia("cta_", SUJETO + "\0" + huella + "\0cuenta")
    if a.aplicar:
        publicar(a.pg, material, d)
    comprobar_publicado(a.pg, material, d, cuenta)
    manifiestos(material, d, cuenta, a.aplicar)
    print("tecnica_bolsa=preparada reinicio_app=necesario")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("tecnica_bolsa=error clave=" + str(error).split(":", 1)[0]) from None
