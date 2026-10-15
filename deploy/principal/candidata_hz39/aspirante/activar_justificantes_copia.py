#!/usr/bin/env python3
"""Prepara justificantes de renuncia en una copia ficticia de HZ37.

Fundamento funcional: Reglamento de bolsas de la Diputación, art. 10,
BOP de Granada 9/2026, 16 de enero. Exige acreditar documentalmente la
renuncia justificada; no fija garantía alta para la lectura de RRHH.
https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf

Orden de la copia: instalar una vez AD263, AD265, AD277, AUT95, D20, D22,
B119 y CT222;
crear los dos LOGIN documentales sin grupo antes de D22, fijar contraseñas
SCRAM fuera de Git y conceder los grupos exactos tras D22, antes de B119;
restaurar las ACL de base y tipos de HZ37; publicar la identidad y el contexto
externos del candidato por el CLI nominal. Si falta su asignación, ejecutar
`asignar_candidato_copia.py` con la misma copia y su CA, y repetir este guion:
la nueva asignación queda ligada por CAS al rol v2. El director publica las
28 claves de confianza, comprueba el arranque con y sin estos dos JSON y
recorre subida, admisión y descarga desde Chrome. Nada de eso lo hace este
guion ni lo acredita un recibo local del almacén.

Caso PDF: la bolsa sintética `bolsa:administrativo:2026-09-17` tiene dos
llamamientos antiguos de Antonio, ya respondidos y vencidos el 13/10/2026.
En la copia ficticia, ejecutar después el guion privado
`scratch/fixture_antonio_justificante.py`: añade un llamamiento y un contacto
actuales marcados como datos de ensayo, sin borrar la historia anterior. Esa
semilla no acredita una emisión de RRHH. La subida, respuesta y descarga del
recorrido deben producir sus propios recibos reales. El plazo de respuesta
conserva la regla publicada: un día hábil desde el contacto efectivo.
El alias HMAC del candidato se coteja con
`incorporacion/actos/aliases-canonicos-20261014.json`: si la cuenta del clon
conserva el alias anterior, el guion cambia solo ese dato ficticio. El alias
de Usuarios queda fuera de esta operación.
Los tres llamantes técnicos de Bolsa se admiten en Identidad con referencias
de ensayo guardadas fuera de Git. Eso no sustituye las aprobaciones formales
que harían falta para publicar esos llamantes fuera de la copia.
"""

import argparse
import copy
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import subprocess


PG = "vec-codexo-aspirante-pg-20261014"
OPERADOR = "operacion:ensayo:bolsa-justificantes"
MOTIVO = "motivos_autorizacion_justificante_renuncia_bolsa"
CANDIDATO_ROL = "candidato_bolsa_portal_historial_propio_desarrollo"
RRHH_ROL = "renuncia_bolsa_rrhh_desarrollo"
SUBIR = "bolsa.justificante_renuncia.subir_propio"
DESCARGAR = "bolsa.justificante_renuncia.descargar_propio"
RRHH = "bolsa.justificante_renuncia.descargar_rrhh"
LLAMANTES = (
    ("vec_externo_v3_registro_autorizacion_desarrollo", "vec_autorizacion_registro_externo"),
    ("vec_externo_bolsa_desarrollo", "vec_bolsa_llamamientos_portal_externo"),
    ("vec_externo_documentos_bolsa_datos", "vec_documentos_bolsa_subida_datos"),
)
ELENA = "desarrollo:elena-ruiz-martin-bolsa"
REF_ROL = lambda rol, version: f"rol:{rol}:v{version}"


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(ruta, directorio=False):
    st = ruta.lstat()
    exigir(st.st_uid == os.getuid() and not ruta.is_symlink() and
           stat.S_IMODE(st.st_mode) == (0o700 if directorio else 0o600) and
           (stat.S_ISDIR(st.st_mode) if directorio else stat.S_ISREG(st.st_mode)),
           "material_no_privado")


def lit(valor):
    exigir(isinstance(valor, str) and "\x00" not in valor, "literal_invalido")
    return "'" + valor.replace("'", "''") + "'"


def jlit(valor):
    return lit(json.dumps(valor, ensure_ascii=False, separators=(",", ":"))) + "::jsonb"


def psql(sql):
    r = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "postgres", "-d", "postgres",
                        "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=verbose"],
                       input=sql, text=True, capture_output=True, timeout=40, check=False)
    if r.returncode:
        codigo = re.search(r"\b[0-9]{2}[A-Z0-9]{3}\b", r.stderr)
        restriccion = re.search(r'constraint "([a-z0-9_]+)"|CONSTRAINT NAME: ([a-z0-9_]+)', r.stderr)
        raise ValueError("sql_justificantes_" + (codigo.group(0) if codigo else "rechazado") +
                         ("_" + (restriccion.group(1) or restriccion.group(2)) if restriccion else ""))
    return r.stdout.strip()


def fuente(repo):
    d = json.loads((repo / "deploy/principal/fuentes/bolsa_justificantes_renuncia_v1.json").read_text())
    entradas = d["entradas"]
    canon = d["entradas_canon"]
    esperado = [{k: copy.deepcopy(v) for k, v in e.items() if k != "fuente_huella_sha256"} for e in entradas]
    for entrada in esperado:
        if entrada["concesion"].get("obligaciones") == []:
            del entrada["concesion"]["obligaciones"]
    exigir(d["modulo_id"] == "bolsa" and d["version"] == 1 and
           d["referencia"] == "fuente:bolsa:justificantes_renuncia_v1" and
           hashlib.sha256(canon.encode()).hexdigest() == d["huella_sha256"] and
           json.loads(canon) == esperado and len(entradas) == 3 and
           all(e["fuente_huella_sha256"] == d["huella_sha256"] for e in entradas),
           "fuente_documental_divergente")
    concesiones = {e["concesion"]["accion"]: e for e in entradas}
    exigir(list(concesiones) == [SUBIR, DESCARGAR, RRHH] and
           all(e["concesion"]["tipo_recurso"] == "justificante_renuncia" and
               e["concesion"]["finalidades"] == ["custodiar_justificante_renuncia_bolsa"] and
               e["concesion"]["obligaciones"] == [] for e in entradas) and
           [concesiones[a]["dimensiones_ambito"] for a in (SUBIR, DESCARGAR, RRHH)] ==
           [["persona_ref"], ["persona_ref"], ["bolsa_ref", "participacion_ref"]] and
           [concesiones[a]["concesion"]["garantia_minima"] for a in (SUBIR, DESCARGAR, RRHH)] ==
           ["alto", "alto", "sustancial"], "concesiones_documentales_divergentes")
    return {accion: entrada["concesion"] for accion, entrada in concesiones.items()}


def modulo_elena(repo):
    ruta=repo/"deploy/principal/candidata_hz39/aspirante/activar_tecnica_bolsa_copia.py"
    especificacion=importlib.util.spec_from_file_location("activar_tecnica_bolsa_reutilizada",ruta)
    exigir(especificacion is not None and especificacion.loader is not None,"guion_elena_ausente")
    modulo=importlib.util.module_from_spec(especificacion)
    especificacion.loader.exec_module(modulo)
    return modulo


def leer_rol(rol, version):
    q = ("SELECT jsonb_build_object('rol',r.documento,'rol_huella',r.huella_sha256,"
         "'control',c.documento,'control_huella',c.huella_sha256)::text "
         "FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.control_vigencia_version_rol_actual ca "
         "USING(version_rol_ref) JOIN vec_autorizacion.control_vigencia_version_rol c "
         "ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision "
         "WHERE r.version_rol_ref=" + lit(REF_ROL(rol, version)))
    dato = psql(q)
    exigir(bool(dato), "rol_base_ausente")
    d = json.loads(dato)
    exigir(d["rol"]["rol_id"] == rol and d["rol"]["version"] == version and
           d["rol"]["estado"] == "publicada" and d["control"]["estado"] == "habilitada" and
           psql("SELECT encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(" +
                jlit(d["rol"]) + "),'UTF8')),'hex')") == d["rol_huella"] and
           psql("SELECT encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(" +
                jlit(d["control"]) + "),'UTF8')),'hex')") == d["control_huella"],
           "rol_base_divergente")
    return d["rol"]


def nueva_version(base, nuevas, marca):
    existentes = {c["accion"] for c in base["concesiones"]}
    exigir(len(existentes) == len(base["concesiones"]) and
           not any(c["accion"] in existentes for c in nuevas), "concesion_duplicada")
    d = dict(base)
    d.update(version=base["version"] + 1, concesiones=base["concesiones"] + nuevas,
             publicada_en=marca, publicada_por=OPERADOR, estado="publicada")
    c = {"version_rol_ref": REF_ROL(d["rol_id"], d["version"]), "revision": 1,
         "estado": "habilitada", "actualizado_por": OPERADOR, "actualizado_en": marca}
    return d, c


def publicar_rol(rol, control):
    ref = control["version_rol_ref"]
    r, c = jlit(rol), jlit(control)
    sql = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:bolsa-justificantes:rol',0));
DO $pre$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={lit(ref)})
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE
  version_rol_ref={lit(REF_ROL(rol['rol_id'],rol['version']-1))})
 THEN RAISE EXCEPTION 'rol_documental_preimagen' USING ERRCODE='55000'; END IF;
END $pre$;
INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 SELECT {lit(ref)},{lit(rol['rol_id'])},{rol['version']},
  encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1({r}),'UTF8')),'hex'),
  ({r}->>'publicada_en')::timestamptz,{r};
INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 SELECT {lit(ref)},1,'habilitada',
  encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1({c}),'UTF8')),'hex'),
  ({c}->>'actualizado_en')::timestamptz,{c};
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
 VALUES({lit(ref)},1,({c}->>'actualizado_en')::timestamptz,{lit(OPERADOR)},
  {lit('acto:ensayo:bolsa-justificantes:rol:v'+str(rol['version']))});
COMMIT;"""
    psql(sql)
    exigir(leer_rol(rol["rol_id"], rol["version"])["concesiones"] == rol["concesiones"],
           "rol_documental_no_cotejado")


def motivo_canon(ahora):
    entrada = {"clave": "motivo_" + hashlib.sha256(("vec:ensayo:bolsa:" + MOTIVO).encode()).hexdigest()[:32],
               "vigente_desde": ahora, "vigente_hasta": None}
    canon = {"catalogo_id": MOTIVO, "catalogo_version": 1, "entradas": [entrada]}
    huella = hashlib.sha256(json.dumps(canon, ensure_ascii=False, sort_keys=True,
                                     separators=(",", ":")).encode()).hexdigest()
    return {**canon, "catalogo_huella_publicada_sha256": huella}


def guardar_exclusivo(ruta, contenido, aplicar):
    if ruta.exists() or ruta.is_symlink():
        privado(ruta)
        exigir(ruta.read_bytes() == contenido, "material_existente_divergente")
        return
    exigir(aplicar, "material_pendiente")
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as out:
        out.write(contenido)
        out.flush()
        os.fsync(out.fileno())


def publicar_motivo(ensayo, aplicar):
    ruta = ensayo / "incorporacion/actos" / ("catalogo-" + MOTIVO + ".json")
    privado(ruta.parent,True)
    ahora = (datetime.now(timezone.utc) - timedelta(minutes=1)).replace(microsecond=0)
    propuesta = motivo_canon(ahora.strftime("%Y-%m-%dT%H:%M:%S.%fZ"))
    if ruta.exists():
        privado(ruta)
        d = json.loads(ruta.read_text())
    else:
        d = propuesta
        guardar_exclusivo(ruta, (json.dumps(d,ensure_ascii=False,separators=(",",":"))+"\n").encode(), aplicar)
    canon = {k: d[k] for k in ("catalogo_id","catalogo_version","entradas")}
    huella = hashlib.sha256(json.dumps(canon,ensure_ascii=False,sort_keys=True,separators=(",",":")).encode()).hexdigest()
    exigir(d["catalogo_id"] == MOTIVO and d["catalogo_version"] == 1 and
           d["catalogo_huella_publicada_sha256"] == huella and len(d["entradas"]) == 1 and
           d["entradas"][0]["vigente_hasta"] is None, "motivo_divergente")
    q = ("SELECT c.catalogo_huella_publicada_sha256,e.entrada_clave "
         "FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
         "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
         "WHERE c.catalogo_id="+lit(MOTIVO)+" AND c.catalogo_version=1")
    actual = psql(q)
    if not actual:
        exigir(aplicar, "motivo_no_publicado")
        sec = int(psql("SELECT ultima_secuencia+1 FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true"))
        evento_huella = hashlib.sha256((huella+":"+str(sec)).encode()).hexdigest()
        sql = ("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
               "SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s'; "
               "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:bolsa:motivo',0)); "
               "SELECT vec_autorizacion.publicar_motivos_autorizacion_v2("+
               lit("evento_"+evento_huella[:32])+","+str(sec)+","+lit(evento_huella)+","+
               lit(MOTIVO)+",1,"+lit(huella)+","+lit(d["entradas"][0]["vigente_desde"])+"::timestamptz,"+
               jlit(d["entradas"])+"); COMMIT;")
        psql(sql)
        actual = psql(q)
    exigir(actual == huella+"|"+d["entradas"][0]["clave"], "motivo_no_cotejado")
    return {"catalogo_id": MOTIVO,"catalogo_version":1,"catalogo_huella_sha256":huella,
            "entrada_clave":d["entradas"][0]["clave"]}


def preparar_config(ensayo, motivo, aplicar):
    externo = ensayo / "externo/externo/documentos"
    interno = ensayo / "material/identidad"
    privado(externo, True); privado(interno, True)
    plantilla = json.loads((externo / "subidas.json").read_text())
    exigir(plantilla["version"] == 1 and plantilla["directorio"] == "/vec-documentos-custodia" and
           plantilla["retencion_minima_dias"] >= 0, "almacen_documental_divergente")
    claves = ("bolsa-correlacion.bin", "bolsa-seudonimo.bin", "bolsa-solicitud.bin")
    for nombre in claves:
        ruta = externo / nombre
        if not ruta.exists() and not ruta.is_symlink():
            exigir(aplicar, "clave_pendiente")
            guardar_exclusivo(ruta,os.urandom(32),True)
        privado(ruta)
        exigir(len(ruta.read_bytes()) == 32 and ruta.read_bytes() != bytes(32),"clave_invalida")
    exigir(len({(externo/n).read_bytes() for n in claves}) == 3,"claves_repetidas")
    dsn = "postgresql://vec_externo_documentos_bolsa_datos@127.0.0.1:5432/postgres?sslmode=verify-full&sslrootcert=/vec-pg-ca.crt"
    datos = dict(plantilla, dsn_datos=dsn, dsn_frontera="",
                 clave_correlacion_archivo=claves[0],clave_seudonimo_archivo=claves[1],
                 clave_solicitud_archivo=claves[2])
    conf = {"version":1,"datos":datos,"motivo":motivo}
    guardar_exclusivo(externo/"justificantes-bolsa.json",
                      (json.dumps(conf,ensure_ascii=False,separators=(",",":"))+"\n").encode(),aplicar)
    origen = interno / "elena-bolsa.json"
    privado(origen)
    identidad = json.loads(origen.read_text())
    huella = identidad["certificate_sha256"]
    exigir(identidad["subject"] == ELENA and re.fullmatch(r"[0-9a-f]{64}",huella),
           "identidad_elena_divergente")
    def referencia(prefijo, valor):
        digerido = hashlib.sha256(b"vec.ct.alta.desarrollo.v1\0"+valor.encode()).digest()
        return prefijo+digerido[:16].hex()
    base=ELENA+"\0"+huella
    interna={"version":1,"superficie":"interna_corporativa",
             "dsn_documentos":"postgresql://vec_interno_documentos_bolsa_rrhh@127.0.0.1:5432/postgres?sslmode=verify-full&sslrootcert=/vec-pg-ca.crt",
             "certificado_sha256":huella,"sujeto":ELENA,
             "cuenta_ref":referencia("cta_",ELENA+"\0"+huella+"\0cuenta"),
             "perfil_ref":referencia("prf_",base+"\0perfil"),
             "persona_ref":referencia("per_",base+"\0persona")}
    guardar_exclusivo(interno/"bolsa-justificantes-rrhh.json",
                      (json.dumps(interna,ensure_ascii=False,separators=(",",":"))+"\n").encode(),aplicar)


def conectar_documentos():
    parejas=(("vec_externo_documentos_bolsa_datos","vec_documentos_bolsa_subida_datos"),
            ("vec_interno_documentos_bolsa_rrhh","vec_documentos_bolsa_rrhh"))
    consultas=[]
    for login,grupo in parejas:
        q=("SELECT l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb "
           "OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) "
           "AND l.rolconfig IS NULL AND a.rolpassword LIKE 'SCRAM-SHA-256$%' "
           "AND NOT g.rolcanlogin AND g.rolinherit AND NOT(g.rolsuper OR g.rolcreatedb "
           "OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) "
           "AND g.rolconfig IS NULL "
           "AND (SELECT count(*)=1 FROM pg_auth_members WHERE member=l.oid) "
           "AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid "
           "AND m.roleid=g.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option) "
           "AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid) "
           "AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN (l.oid,g.oid)) "
           "AND NOT has_database_privilege(l.rolname,'postgres','TEMPORARY') "
           "FROM pg_roles l JOIN pg_authid a ON a.oid=l.oid "
           "JOIN pg_roles g ON g.rolname="+lit(grupo)+
           " WHERE l.rolname="+lit(login))
        exigir(psql(q)=="t","login_documental_no_nominal")
        estado=psql("SELECT has_database_privilege("+lit(grupo)+",'postgres','CONNECT'),"
                    "has_database_privilege("+lit(grupo)+",'postgres','TEMPORARY')")
        exigir(estado in ("t|f","f|f"),"acl_base_documental_divergente")
        consultas.append(q)
    pre=" OR ".join("("+q+") IS DISTINCT FROM TRUE" for q in consultas)
    pre+=" OR "+" OR ".join("has_database_privilege("+lit(grupo)+
                          ",'postgres','TEMPORARY')" for _,grupo in parejas)
    post=pre+" OR "+" OR ".join("NOT has_database_privilege("+lit(rol)+
                                ",'postgres','CONNECT')" for pareja in parejas for rol in pareja)
    sql=("BEGIN; SET LOCAL search_path=pg_catalog; SET LOCAL lock_timeout='3s'; "
         "DO $pre$ BEGIN IF "+pre+" THEN RAISE EXCEPTION "
         "'login_documental_preimagen' USING ERRCODE='55000'; END IF; END $pre$; "
         "GRANT CONNECT ON DATABASE postgres TO "+",".join(grupo for _,grupo in parejas)+"; "
         "DO $post$ BEGIN IF "+post+" THEN RAISE EXCEPTION "
         "'login_documental_postimagen' USING ERRCODE='55000'; END IF; END $post$; COMMIT;")
    psql(sql)


def admitir_llamantes_bolsa(ensayo, aplicar):
    ruta=ensayo/"incorporacion/actos/aprobaciones-llamantes-justificante-bolsa.json"
    privado(ruta.parent,True)
    if ruta.exists() or ruta.is_symlink():
        privado(ruta)
        referencias=json.loads(ruta.read_text())
    else:
        exigir(aplicar,"aprobaciones_llamantes_pendientes")
        referencias={login:"apr_"+hashlib.sha256(os.urandom(32)).hexdigest()
                     for login,_ in LLAMANTES}
        guardar_exclusivo(ruta,(json.dumps(referencias,separators=(",",":"))+"\n").encode(),True)
    exigir(isinstance(referencias,dict) and
           set(referencias)=={login for login,_ in LLAMANTES} and
           all(isinstance(ref,str) and re.fullmatch(r"apr_[0-9a-f]{64}",ref)
               for ref in referencias.values()) and
           len(set(referencias.values()))==3,"aprobaciones_llamantes_invalidas")
    nombres=",".join(lit(login) for login,_ in LLAMANTES)
    lectura=psql("SELECT login::text,grupo_esperado::text,aprobacion_ref "
                 "FROM vec_identidad_externa_v1.llamante_autorizacion "
                 "WHERE login IN ("+nombres+") ORDER BY login")
    actuales={campos[0]:(campos[1],campos[2]) for campos in
              (linea.split("|") for linea in lectura.splitlines() if linea)}
    exigir(len(actuales)==len(lectura.splitlines()) and
           all(actuales[login]==(grupo,referencias[login]) for login,grupo in LLAMANTES
               if login in actuales),"llamantes_preexistentes_divergentes")
    comprobaciones=[]
    inserciones=[]
    for login,grupo in LLAMANTES:
        l,g=lit(login),lit(grupo)
        comprobaciones.append("NOT EXISTS(SELECT 1 FROM pg_roles l "
            "JOIN pg_auth_members m ON m.member=l.oid "
            "JOIN pg_roles g ON g.oid=m.roleid "
            "WHERE l.rolname="+l+" AND g.rolname="+g+
            " AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL "
            "AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR "
            "l.rolreplication OR l.rolbypassrls) "
            "AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR "
            "g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) "
            "AND g.rolconfig IS NULL AND m.inherit_option "
            "AND NOT m.set_option AND NOT m.admin_option "
            "AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid) "
            "AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1)")
        inserciones.append("INSERT INTO vec_identidad_externa_v1.llamante_autorizacion "
            "(login,grupo_esperado,aprobacion_ref) SELECT "+l+","+g+","+
            lit(referencias[login])+" WHERE NOT EXISTS(SELECT 1 FROM "
            "vec_identidad_externa_v1.llamante_autorizacion WHERE login="+l+");")
    pre=" OR ".join(comprobaciones)
    for login,grupo in LLAMANTES:
        pre+=" OR EXISTS(SELECT 1 FROM vec_identidad_externa_v1.llamante_autorizacion "
        pre+=("WHERE login="+lit(login)+" AND (grupo_esperado<>"+lit(grupo)+
              " OR aprobacion_ref<>"+lit(referencias[login])+"))")
    post=" OR ".join("NOT EXISTS(SELECT 1 FROM "
        "vec_identidad_externa_v1.llamante_autorizacion WHERE login="+lit(login)+
        " AND grupo_esperado="+lit(grupo)+" AND aprobacion_ref="+
        lit(referencias[login])+")" for login,grupo in LLAMANTES)
    sql=("BEGIN ISOLATION LEVEL SERIALIZABLE; "
         "SET LOCAL ROLE vec_identidad_sesiones_v1_propietario; "
         "SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC'; "
         "SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='15s'; "
         "SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:o:identidad:bolsa-justificantes',0)); "
         "DO $pre$ BEGIN IF "+pre+" THEN RAISE EXCEPTION "
         "'llamantes_bolsa_preimagen' USING ERRCODE='55000'; END IF; END $pre$; "
         +" ".join(inserciones)+
         "DO $post$ BEGIN IF "+post+" THEN RAISE EXCEPTION "
         "'llamantes_bolsa_postimagen' USING ERRCODE='55000'; END IF; END $post$; COMMIT;")
    psql(sql)
    final=psql("SELECT login::text,grupo_esperado::text,aprobacion_ref "
               "FROM vec_identidad_externa_v1.llamante_autorizacion "
               "WHERE login IN ("+nombres+") ORDER BY login")
    exigir({campos[0]:(campos[1],campos[2]) for campos in
            (linea.split("|") for linea in final.splitlines() if linea)}==
           {login:(grupo,referencias[login]) for login,grupo in LLAMANTES},
           "llamantes_bolsa_no_cotejados")


def alias_canonico_candidato(ensayo):
    actos=ensayo/"incorporacion/actos"
    anterior_ruta=actos/"alias-candidato-externo.json"
    canon_ruta=actos/"aliases-canonicos-20261014.json"
    identidad_ruta=ensayo/"externo/identidad/bolsa-candidato.json"
    for ruta in (anterior_ruta,canon_ruta,identidad_ruta):
        privado(ruta)
    anterior=json.loads(anterior_ruta.read_text())
    canon=json.loads(canon_ruta.read_text())
    identidad=json.loads(identidad_ruta.read_text())
    exigir(anterior["version"]==canon["version"]==1 and len(anterior["cuentas"])==1,
           "alias_candidato_material_divergente")
    viejo=anterior["cuentas"][0]
    candidatos=[x for x in canon["cuentas"] if x.get("cuenta_ref")==identidad["cuenta_ref"]]
    exigir(len(candidatos)==1 and viejo["cuenta_ref"]==identidad["cuenta_ref"],
           "alias_candidato_ambiguo")
    nuevo=candidatos[0]
    exigir(set(viejo)==set(nuevo) and all(viejo[k]==nuevo[k] for k in viejo
           if k!="cuenta_id_hmac") and
           re.fullmatch(r"[0-9a-f]{64}",viejo["cuenta_id_hmac"]) is not None and
           re.fullmatch(r"[0-9a-f]{64}",nuevo["cuenta_id_hmac"]) is not None and
           viejo["cuenta_id_hmac"]!=nuevo["cuenta_id_hmac"],
           "alias_candidato_preimagen_divergente")
    comun=("cuenta_ref="+lit(nuevo["cuenta_ref"])+
           " AND esquema_hmac="+lit(nuevo["esquema"])+
           " AND dominio_hmac_ref="+lit(nuevo["dominio_ref"])+
           " AND clave_hmac_id="+lit(nuevo["clave_id"])+
           " AND clave_hmac_version="+str(nuevo["clave_version"])+
           " AND sujeto_id_hmac=decode("+lit(nuevo["sujeto_id_hmac"])+",'hex')")
    tabla="vec_identidad_externa_v1.alias_cuenta"
    estado=psql("SELECT count(*),count(*) FILTER(WHERE cuenta_id_hmac=decode("+
                lit(viejo["cuenta_id_hmac"])+",'hex')),"+
                "count(*) FILTER(WHERE cuenta_id_hmac=decode("+
                lit(nuevo["cuenta_id_hmac"])+",'hex')) FROM "+tabla+
                " WHERE cuenta_ref="+lit(nuevo["cuenta_ref"]))
    exigir(estado in ("1|1|0","1|0|1"),"alias_candidato_db_divergente")
    if estado=="1|1|0":
        sql=("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL search_path=pg_catalog; "
             "SET LOCAL lock_timeout='3s'; SET LOCAL session_replication_role=replica; "
             "DO $fix$ DECLARE n bigint; BEGIN UPDATE "+tabla+
             " SET cuenta_id_hmac=decode("+lit(nuevo["cuenta_id_hmac"])+",'hex') WHERE "+
             comun+" AND cuenta_id_hmac=decode("+lit(viejo["cuenta_id_hmac"])+",'hex'); "
             "GET DIAGNOSTICS n=ROW_COUNT; IF n<>1 OR (SELECT count(*) FROM "+tabla+
             " WHERE cuenta_ref="+lit(nuevo["cuenta_ref"])+")<>1 THEN "
             "RAISE EXCEPTION 'alias_canonico_preimagen' USING ERRCODE='55000'; "
             "END IF; END $fix$; COMMIT;")
        psql(sql)
    exigir(psql("SELECT count(*) FROM "+tabla+" WHERE "+comun+
                " AND cuenta_id_hmac=decode("+lit(nuevo["cuenta_id_hmac"])+",'hex')")=="1",
           "alias_candidato_no_cotejado")


def publicar_elena(ensayo, modulo, rol, aplicar):
    material=ensayo/"material"
    privado(material,True)
    huella=modulo.certificado(material,False)
    d=modulo.datos(huella,[])
    referencia_rol=REF_ROL(RRHH_ROL,rol["version"])
    d["asig"]["version_rol_ref"]=referencia_rol
    cuenta=modulo.referencia("cta_",ELENA+"\0"+huella+"\0cuenta")
    existente=psql("SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref="+lit(cuenta))
    exigir(existente in ("0","1"),"cuenta_elena_duplicada")
    if existente=="0":
        exigir(aplicar,"elena_pendiente")
        ch,sh=modulo.hmac_cuenta(material,cuenta,huella)
        marca=modulo.MARCA
        alias=modulo.referencia("ali_",ELENA+"\0"+huella+"\0alias")
        dominio=modulo.referencia("idh_",modulo.ESPACIO)
        operacion=modulo.referencia("opr_",ELENA+"\0"+huella+"\0provision")
        q=lit
        asig=jlit(d["asig"])
        sql=f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:bolsa-justificantes:elena',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={q(referencia_rol)})
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref={q(cuenta)})
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref={q(d['persona'])})
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref={q(d['perfil'])})
 THEN RAISE EXCEPTION 'elena_documental_preimagen' USING ERRCODE='55000'; END IF;
END $pre$;
INSERT INTO vec_identidad_sesiones_v1.cuenta VALUES({q(cuenta)},false,NULL,{q(marca)}::timestamptz,{q(operacion)});
INSERT INTO vec_identidad_sesiones_v1.alias_hmac_cuenta VALUES(
 {q(alias)},{q(cuenta)},
 'vec.identidad.hmac-sha256.v1',{q(dominio)},
 'vec.identidad.desarrollo.g2',2,decode({q(ch)},'hex'),decode({q(sh)},'hex'),
 {q(marca)}::timestamptz,{q(operacion)});
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta VALUES({q(cuenta)},1,'activa',{q(marca)}::timestamptz,{q(operacion)});
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual VALUES({q(cuenta)},1,{q(marca)}::timestamptz,{q(operacion)});
INSERT INTO vec_contexto_actor_v1.procedencias VALUES({q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES({q(cuenta)},1,{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(marca)}::timestamptz,{q(modulo.HASTA)}::timestamptz);
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES({q(cuenta)},1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES({q(d['persona'])},1,{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(marca)}::timestamptz,{q(modulo.HASTA)}::timestamptz);
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES({q(d['persona'])},1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES({q(d['perfil'])},1,{q(d['persona'])},{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(marca)}::timestamptz,{q(modulo.HASTA)}::timestamptz);
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES({q(d['perfil'])},1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES({q(d['vinculo'])},1,{q(cuenta)},{q(d['perfil'])},{q(d['persona'])},{q(d['procedencia'])},1,{q(d['procedencia_huella'])},'autoridad_maestra_acreditada','activo',{q(marca)}::timestamptz,{q(modulo.HASTA)}::timestamptz);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES({q(d['vinculo'])},1);
INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT {q(d['asig_ref'])},{q(d['asig']['asignacion_id'])},1,{q(d['perfil'])},{q(d['persona'])},{q(referencia_rol)},
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({asig}),'UTF8')),'hex'),{q(marca)}::timestamptz,{asig};
INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES({q(d['perfil'])},{q(d['asig_ref'])},{q(marca)}::timestamptz,{q(OPERADOR)},'acto:ensayo:elena-bolsa:documental');
COMMIT;"""
        psql(sql)
    actualizar_asignacion_elena(d,aplicar)
    exigir(psql("SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual a "
                "JOIN vec_autorizacion.asignacion_perfil p USING(asignacion_ref) WHERE "
                "a.perfil_activo_ref="+lit(d["perfil"])+" AND p.version_rol_ref="+lit(referencia_rol)+
                " AND (p.documento->>'vigente_hasta')::timestamptz > clock_timestamp()+interval '90 days'")=="1",
           "elena_documental_no_cotejada")
    modulo.manifiestos(material,d,cuenta,aplicar)
    return d


def actualizar_asignacion_elena(d,aplicar):
    q=("SELECT jsonb_build_object('ref',a.asignacion_ref,'documento',a.documento,"
       "'huella',a.huella_sha256)::text FROM vec_autorizacion.asignacion_perfil_actual actual "
       "JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref) "
       "WHERE actual.perfil_activo_ref="+lit(d["perfil"]))
    lectura=psql(q)
    exigir(bool(lectura),"asignacion_elena_ausente")
    anterior=json.loads(lectura)
    doc=anterior["documento"]
    exigir(doc["perfil_activo_ref"]==d["perfil"] and doc["principal_id"]==d["persona"] and
           doc["asignacion_id"]==d["asig"]["asignacion_id"] and doc["estado"]=="activa" and
           doc["ambitos"]==d["asig"]["ambitos"] and
           doc["revocada_en"]=="0001-01-01T00:00:00Z" and
           psql("SELECT encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1("+
                jlit(doc)+"),'UTF8')),'hex')")==anterior["huella"],
           "asignacion_elena_divergente")
    fin=datetime.fromisoformat(doc["vigente_hasta"].replace("Z","+00:00"))
    exigir(fin>datetime.now(timezone.utc)+timedelta(days=90),"vigencia_elena_corta")
    destino=REF_ROL(RRHH_ROL,2)
    if doc["version_rol_ref"]==destino:
        exigir((doc["version"]==1 and doc==d["asig"] and anterior["ref"]==d["asig_ref"])
               or (doc["version"]==2 and anterior["ref"]==
                   "asignacion:"+doc["asignacion_id"]+":v2"),
               "asignacion_elena_version_divergente")
        return
    esperado=dict(d["asig"],version_rol_ref=REF_ROL(RRHH_ROL,1))
    exigir(doc==esperado and anterior["ref"]==d["asig_ref"] and doc["version"]==1,
           "asignacion_elena_v1_divergente")
    exigir(aplicar,"asignacion_elena_v2_pendiente")
    ahora=datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    nuevo=dict(doc,version=2,version_rol_ref=destino,emitida_por=OPERADOR,
               emitida_en=ahora,vigente_desde=ahora)
    nueva_ref="asignacion:"+doc["asignacion_id"]+":v2"
    nd=jlit(nuevo)
    sql=f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:bolsa-justificantes:elena',0));
DO $pre$ BEGIN IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
 WHERE perfil_activo_ref={lit(d['perfil'])} AND asignacion_ref={lit(anterior['ref'])})
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref={lit(nueva_ref)})
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={lit(destino)})
 THEN RAISE EXCEPTION 'elena_documental_cas_preimagen' USING ERRCODE='55000'; END IF; END $pre$;
INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,
 perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT {lit(nueva_ref)},{lit(doc['asignacion_id'])},2,{lit(d['perfil'])},{lit(d['persona'])},
 {lit(destino)},encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({nd}),
 'UTF8')),'hex'),({nd}->>'emitida_en')::timestamptz,{nd};
UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref={lit(nueva_ref)},
 actualizada_en=clock_timestamp(),actualizada_por={lit(OPERADOR)},
 acto_ref='acto:ensayo:bolsa-justificantes:elena:v2'
 WHERE perfil_activo_ref={lit(d['perfil'])} AND asignacion_ref={lit(anterior['ref'])};
DO $post$ BEGIN IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
 WHERE perfil_activo_ref={lit(d['perfil'])} AND asignacion_ref={lit(nueva_ref)})
 THEN RAISE EXCEPTION 'elena_documental_cas' USING ERRCODE='40001'; END IF; END $post$;
COMMIT;"""
    psql(sql)
    exigir(json.loads(psql(q))["documento"]==nuevo,"asignacion_elena_v2_no_cotejada")


def asignar_candidato(ensayo, aplicar):
    ruta=ensayo/"externo/identidad/bolsa-candidato.json"
    privado(ruta)
    identidad=json.loads(ruta.read_text())
    perfil=identidad["perfil_ref"]
    exigir(re.fullmatch(r"prf_[A-Za-z0-9_-]{22,128}",perfil) is not None,
           "perfil_externo_invalido")
    q=("SELECT jsonb_build_object('ref',a.asignacion_ref,'documento',a.documento)::text "
       "FROM vec_autorizacion.asignacion_perfil_actual_externa actual "
       "JOIN vec_autorizacion.asignacion_perfil_externa a USING(asignacion_ref) "
       "WHERE actual.perfil_activo_ref="+lit(perfil))
    lectura=psql(q)
    if not lectura:
        return False
    anterior=json.loads(lectura)
    doc=anterior["documento"]
    exigir(doc["principal_id"]==identidad["persona_ref"] and
           doc["perfil_activo_ref"]==perfil and doc["estado"]=="activa" and
           doc["ambitos"]==[{"clave":"candidato_ref","valores":[identidad["candidato_ref"]]}],
           "asignacion_externa_divergente")
    ahora=datetime.now(timezone.utc).replace(microsecond=0)
    fin=datetime.fromisoformat(doc["vigente_hasta"].replace("Z","+00:00"))
    exigir(fin>=ahora+timedelta(days=90),"vigencia_candidato_corta")
    destino=REF_ROL(CANDIDATO_ROL,2)
    if doc["version_rol_ref"]==destino:
        return True
    exigir(doc["version_rol_ref"]==REF_ROL(CANDIDATO_ROL,1),"rol_candidato_no_previsto")
    exigir(aplicar,"asignacion_candidato_pendiente")
    version=doc["version"]+1
    nuevo=dict(doc,version=version,version_rol_ref=destino,emitida_por=OPERADOR,
               emitida_en=ahora.strftime("%Y-%m-%dT%H:%M:%SZ"),
               vigente_desde=ahora.strftime("%Y-%m-%dT%H:%M:%SZ"))
    nueva_ref=f"asignacion:{doc['asignacion_id']}:v{version}"
    nd=jlit(nuevo)
    sql=f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='3s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:ensayo:bolsa-justificantes:candidato',0));
DO $pre$ BEGIN IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa
 WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(anterior['ref'])})
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref={lit(nueva_ref)})
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={lit(destino)})
 THEN RAISE EXCEPTION 'candidato_documental_preimagen' USING ERRCODE='55000'; END IF; END $pre$;
INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,
 perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT {lit(nueva_ref)},{lit(doc['asignacion_id'])},{version},{lit(perfil)},
 {lit(doc['principal_id'])},{lit(destino)},
 encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1({nd}),'UTF8')),'hex'),
 ({nd}->>'emitida_en')::timestamptz,{nd};
UPDATE vec_autorizacion.asignacion_perfil_actual_externa SET asignacion_ref={lit(nueva_ref)},
 actualizada_en=clock_timestamp(),actualizada_por={lit(OPERADOR)},
 acto_ref='acto:ensayo:bolsa-justificantes:candidato'
 WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(anterior['ref'])};
DO $post$ BEGIN IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa
 WHERE perfil_activo_ref={lit(perfil)} AND asignacion_ref={lit(nueva_ref)})
 THEN RAISE EXCEPTION 'candidato_documental_cas' USING ERRCODE='40001'; END IF; END $post$;
COMMIT;"""
    psql(sql)
    exigir(psql(q)!="" and json.loads(psql(q))["documento"]["version_rol_ref"]==destino,
           "candidato_documental_no_cotejado")
    return True


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--ensayo",type=Path,required=True)
    p.add_argument("--pg",required=True)
    p.add_argument("--aplicar",action="store_true")
    a=p.parse_args()
    repo=Path(__file__).resolve().parents[4]
    exigir(a.pg==PG and a.ensayo.is_absolute() and a.ensayo.name=="vec-ensayo-codexo-aspirante-20261014" and
           repo not in a.ensayo.parents,"solo_clon_privado")
    privado(a.ensayo,True)
    concesiones=fuente(repo)
    if not a.aplicar:
        print("justificantes=plan_preparado base_sin_cambios")
        return
    conectar_documentos()
    admitir_llamantes_bolsa(a.ensayo,True)
    motivo=publicar_motivo(a.ensayo,True)
    para_candidato=leer_rol(CANDIDATO_ROL,1)
    para_rrhh=leer_rol(RRHH_ROL,1)
    marca=datetime.now(timezone.utc).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")
    elena=modulo_elena(repo)
    concesiones_elena=elena.fuentes(repo)
    exigir(len(concesiones_elena)==8 and len({c["accion"] for c in concesiones_elena})==8,
           "fuente_elena_divergente")
    for base,nuevas in ((para_candidato,[concesiones[SUBIR],concesiones[DESCARGAR]]),
                        (para_rrhh,concesiones_elena+[concesiones[RRHH]])):
        rol,control=nueva_version(base,nuevas,marca)
        if not psql("SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref="+lit(control["version_rol_ref"])):
            publicar_rol(rol,control)
        else:
            exigir(leer_rol(rol["rol_id"],rol["version"])["concesiones"]==rol["concesiones"],
                   "rol_documental_repetido_divergente")
    publicar_elena(a.ensayo,elena,leer_rol(RRHH_ROL,2),True)
    candidato=asignar_candidato(a.ensayo,True)
    preparar_config(a.ensayo,motivo,True)
    alias_canonico_candidato(a.ensayo)
    print("justificantes=preparados candidato="+("asignado" if candidato else "bootstrap_pendiente"))


if __name__=="__main__":
    try:
        main()
    except (OSError,ValueError,KeyError,TypeError,json.JSONDecodeError,subprocess.TimeoutExpired) as exc:
        raise SystemExit("justificantes=error clave="+str(exc).split(":",1)[0]) from None
