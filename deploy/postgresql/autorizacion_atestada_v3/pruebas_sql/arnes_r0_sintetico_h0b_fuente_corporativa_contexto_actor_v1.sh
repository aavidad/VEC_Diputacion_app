#!/usr/bin/env bash

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    printf 'el auxiliar R0/H0b solo puede cargarse desde su runner acreditado\n' >&2
    exit 64
fi
if [[ "${VEC_F0_CARGA_PRIVADA:-}" != '1' ]]; then
    printf 'carga no acreditada del auxiliar R0/H0b\n' >&2
    return 64
fi
unset VEC_F0_CARGA_PRIVADA

plantilla_migracion_c2_virtual_h0b_f0() {
    command cat <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(
 p_audiencia_consumo_esperada text,p_accion_esperada text,p_tipo_efecto_esperado text,
 p_operacion_ref_esperada text,p_efecto_ref_esperada text,p_huella_efecto_sha256_esperada text,
 p_capacidad_canonica bytea,p_manifiesto_fuente_canonico bytea,p_sobre_cose_sign1 bytea,
 p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE (capacidad_ref text,fuente_ref text,fuente_version numeric,
 evento_fuente_ref text,huella_evento_fuente_sha256 text,huella_manifiesto_fuente_sha256 text,
 operacion_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE CALLED ON NULL INPUT SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET lock_timeout='2s' AS $c2$
BEGIN
 IF pg_catalog.to_regrole('vec_contexto_actor_v1_publicador_corporativo') IS NULL
    OR pg_catalog.to_regrole('vec_contexto_actor_v1_revocador_corporativo') IS NULL
    OR pg_catalog.to_regrole('vec_contexto_actor_v1_despachador_corporativo') IS NULL
    OR session_user <> 'vec_f0_h0_publicador'
    OR pg_catalog.current_setting('role') <> 'none'
    OR NOT pg_catalog.pg_has_role(session_user,
       'vec_contexto_actor_v1_publicador_corporativo','USAGE') THEN
  RAISE EXCEPTION USING ERRCODE='42501';
 END IF;
 RETURN;
END
$c2$;
SQL
}

plantilla_prueba_c2_virtual_h0b_f0() {
    local modo="${1:-}"
    [[ "${modo}" == 'nominal' || "${modo}" == 'error' ]] || return 64
    command cat <<'SQL'
CREATE FUNCTION vec_autorizacion_atestada_v3.invocar_c2_virtual_h0b()
RETURNS bigint LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT count(*) FROM vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.invocar_c2_virtual_h0b() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_f0_h0_publicador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.invocar_c2_virtual_h0b() TO vec_f0_h0_publicador;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
SELECT vec_autorizacion_atestada_v3.invocar_c2_virtual_h0b()=0;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SQL
    [[ "${modo}" == 'nominal' ]] || printf 'SELECT 1/0;\n'
}

foto_roles() {
    valor "WITH estado AS (
      SELECT pg_catalog.concat_ws('|','r',rolname,rolsuper,rolinherit,
        rolcreaterole,rolcreatedb,rolcanlogin,rolreplication,rolconnlimit,
        rolpassword,rolvaliduntil,rolbypassrls,
        (SELECT r.rolconfig::text FROM pg_catalog.pg_roles r
          WHERE r.oid=pg_authid.oid)) AS objeto
      FROM pg_catalog.pg_authid UNION ALL
      SELECT pg_catalog.concat_ws('|','m',r.rolname,u.rolname,g.rolname,
        m.admin_option,m.inherit_option,m.set_option)
      FROM pg_catalog.pg_auth_members m
      JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
      JOIN pg_catalog.pg_roles u ON u.oid=m.member
      JOIN pg_catalog.pg_roles g ON g.oid=m.grantor UNION ALL
      SELECT pg_catalog.concat_ws('|','s',coalesce(r.rolname,'0'),
        coalesce(d.datname,'0'),s.setconfig::text)
      FROM pg_catalog.pg_db_role_setting s
      LEFT JOIN pg_catalog.pg_roles r ON r.oid=s.setrole
      LEFT JOIN pg_catalog.pg_database d ON d.oid=s.setdatabase UNION ALL
      SELECT pg_catalog.concat_ws('|','d',r.rolname,d.description)
      FROM pg_catalog.pg_shdescription d JOIN pg_catalog.pg_roles r ON r.oid=d.objoid
      WHERE d.classoid='pg_catalog.pg_authid'::pg_catalog.regclass UNION ALL
      SELECT pg_catalog.concat_ws('|','l',r.rolname,l.provider,l.label)
      FROM pg_catalog.pg_shseclabel l JOIN pg_catalog.pg_roles r ON r.oid=l.objoid
      WHERE l.classoid='pg_catalog.pg_authid'::pg_catalog.regclass)
    SELECT pg_catalog.encode(public.digest(pg_catalog.convert_to(
      pg_catalog.string_agg(objeto,E'\\n' ORDER BY objeto),'UTF8'),
      'sha256'),'hex') FROM estado"
}

definicion_audiencia() {
    valor "SELECT pg_catalog.regexp_replace(
      pg_catalog.pg_get_constraintdef(c.oid,true),'\\s+',' ','g')
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'"
}

foto_checkpoint() {
    valor "SELECT pg_catalog.encode(pg_catalog.convert_to(pg_catalog.row_to_json(c)::text,'UTF8'),'hex') FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c"
}

contar_objetos_f0() {
    valor "SELECT (
      (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
        JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
       WHERE n.nspname='vec_autorizacion_atestada_v3'
         AND c.relname LIKE '%fuente_corporativa%')+
      (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_autorizacion_atestada_v3'
         AND p.proname LIKE '%fuente_corporativa%')+
      (SELECT pg_catalog.count(*) FROM pg_catalog.pg_roles
       WHERE rolname LIKE 'vec_contexto_actor_v1_%corporativo')
    )::text"
}

acreditar_limpieza() {
    local audiencia="$1" checkpoint="$2" catalogo="$3" roles="$4"
    exigir_salida_f0 "${audiencia}" 'la audiencia base cambió durante H0' definicion_audiencia
    exigir_salida_f0 "${checkpoint}" 'checkpoint_gobierno cambió durante H0' foto_checkpoint
    exigir_salida_f0 "${catalogo}" 'la estructura completa cambió durante H0' foto_catalogo
    exigir_salida_f0 "${roles}" 'los roles o sus membresías cambiaron durante H0' foto_roles
    exigir_salida_f0 0 'H0 dejó objetos F0' contar_objetos_f0
    exigir_salida_f0 0 'H0 dejó transacciones preparadas' valor \
        'SELECT count(*)::text FROM pg_catalog.pg_prepared_xacts'
    exigir_salida_f0 0 'H0 dejó objetos temporales' valor "SELECT count(*)::text FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE c.relpersistence='t' AND n.nspname LIKE 'pg_temp_%'"
    exigir_salida_f0 0 'H0 dejó sesiones cliente activas' valor "SELECT count(*)::text FROM pg_catalog.pg_stat_activity WHERE backend_type='client backend' AND pid<>pg_catalog.pg_backend_pid()"
}

nombres_r0_sintetico_f0() {
    printf '%s\n' \
        vec_contexto_actor_v1_publicador_corporativo \
        vec_contexto_actor_v1_revocador_corporativo \
        vec_contexto_actor_v1_despachador_corporativo \
        vec_f0_h0_adicional vec_f0_h0_publicador vec_f0_h0_revocador \
        vec_f0_h0_despachador vec_f0_h0_cruzado vec_f0_h0_extra \
        vec_f0_h0_sin_rol
}

acreditar_r0_ausente_f0() {
    local nombres
    nombres="$(nombres_r0_sintetico_f0 | awk '{printf "%s%s", separador, "\047" $0 "\047"; separador=","}')" || return 65
    [[ "$(valor "SELECT count(*)::text FROM pg_catalog.pg_roles WHERE rolname IN (${nombres})")" == '0' ]]
}

crear_r0_sintetico_f0() {
    sql postgres "$(command cat <<'SQL'
BEGIN;
DO $r0$
BEGIN
  IF NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_database d
      JOIN pg_catalog.pg_roles r ON r.oid=d.datdba
      WHERE d.datname=pg_catalog.current_database()
        AND r.rolname='postgres' AND r.rolsuper
  ) THEN RAISE EXCEPTION 'el propietario de la base no es postgres superusuario';
  END IF;
END
$r0$;
CREATE ROLE vec_contexto_actor_v1_publicador_corporativo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_contexto_actor_v1_revocador_corporativo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_contexto_actor_v1_despachador_corporativo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_adicional NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_revocador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_despachador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_cruzado LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_extra LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
CREATE ROLE vec_f0_h0_sin_rol LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
GRANT vec_contexto_actor_v1_publicador_corporativo TO vec_f0_h0_publicador,vec_f0_h0_cruzado,vec_f0_h0_extra WITH ADMIN FALSE, INHERIT TRUE, SET FALSE GRANTED BY postgres;
GRANT vec_contexto_actor_v1_revocador_corporativo TO vec_f0_h0_revocador,vec_f0_h0_cruzado WITH ADMIN FALSE, INHERIT TRUE, SET FALSE GRANTED BY postgres;
GRANT vec_contexto_actor_v1_despachador_corporativo TO vec_f0_h0_despachador WITH ADMIN FALSE, INHERIT TRUE, SET FALSE GRANTED BY postgres;
GRANT vec_f0_h0_adicional TO vec_f0_h0_extra WITH ADMIN FALSE, INHERIT TRUE, SET FALSE GRANTED BY postgres;
COMMIT;
SQL
)" >/dev/null
}

acreditar_r0_sintetico_f0() {
    local obtenido
    obtenido="$(valor "$(command cat <<'SQL'
WITH esperados(nombre,login,hereda) AS (VALUES
 ('vec_contexto_actor_v1_publicador_corporativo',false,false),
 ('vec_contexto_actor_v1_revocador_corporativo',false,false),
 ('vec_contexto_actor_v1_despachador_corporativo',false,false),
 ('vec_f0_h0_adicional',false,false),('vec_f0_h0_publicador',true,true),
 ('vec_f0_h0_revocador',true,true),('vec_f0_h0_despachador',true,true),
 ('vec_f0_h0_cruzado',true,true),('vec_f0_h0_extra',true,true),
 ('vec_f0_h0_sin_rol',true,true)),
aristas(grupo,miembro) AS (VALUES
 ('vec_contexto_actor_v1_publicador_corporativo','vec_f0_h0_publicador'),
 ('vec_contexto_actor_v1_publicador_corporativo','vec_f0_h0_cruzado'),
 ('vec_contexto_actor_v1_publicador_corporativo','vec_f0_h0_extra'),
 ('vec_contexto_actor_v1_revocador_corporativo','vec_f0_h0_revocador'),
 ('vec_contexto_actor_v1_revocador_corporativo','vec_f0_h0_cruzado'),
 ('vec_contexto_actor_v1_despachador_corporativo','vec_f0_h0_despachador'),
 ('vec_f0_h0_adicional','vec_f0_h0_extra')),
roles AS (SELECT r.* FROM pg_catalog.pg_roles r JOIN esperados e ON e.nombre=r.rolname),
dba AS (SELECT d.datdba FROM pg_catalog.pg_database d WHERE d.datname=pg_catalog.current_database()),
miembros AS (
 SELECT gr.rolname grupo,mi.rolname miembro,m.grantor,m.admin_option,m.inherit_option,m.set_option
 FROM pg_catalog.pg_auth_members m JOIN roles gr ON gr.oid=m.roleid
 JOIN pg_catalog.pg_roles mi ON mi.oid=m.member
 UNION ALL
 SELECT gr.rolname,mi.rolname,m.grantor,m.admin_option,m.inherit_option,m.set_option
 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles gr ON gr.oid=m.roleid
 JOIN roles mi ON mi.oid=m.member WHERE NOT EXISTS (SELECT 1 FROM roles x WHERE x.oid=m.roleid))
SELECT
 (SELECT count(*)=10 AND bool_and(r.rolcanlogin=e.login AND r.rolinherit=e.hereda
    AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
    AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconnlimit=-1
    AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
    AND (SELECT a.rolpassword IS NULL FROM pg_catalog.pg_authid a WHERE a.oid=r.oid))
  FROM roles r JOIN esperados e ON e.nombre=r.rolname)
 AND (SELECT count(*)=7 AND bool_and(a.grupo IS NOT NULL AND m.grantor=d.datdba
    AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
  FROM miembros m LEFT JOIN aristas a USING(grupo,miembro) CROSS JOIN dba d)
 AND (SELECT bool_and(pg_catalog.pg_has_role(a.miembro,a.grupo,'MEMBER')
    AND pg_catalog.pg_has_role(a.miembro,a.grupo,'USAGE')
    AND NOT pg_catalog.pg_has_role(a.miembro,a.grupo,'SET')) FROM aristas a)
 AND (SELECT r.rolsuper FROM dba d JOIN pg_catalog.pg_roles r ON r.oid=d.datdba)
 AND NOT EXISTS (SELECT 1 FROM dba d JOIN roles r ON r.oid=d.datdba)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN roles r ON r.oid=m.grantor)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting s JOIN roles r ON r.oid=s.setrole)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_shdescription x JOIN roles r ON r.oid=x.objoid
                  WHERE x.classoid='pg_catalog.pg_authid'::pg_catalog.regclass)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_shseclabel x JOIN roles r ON r.oid=x.objoid
                  WHERE x.classoid='pg_catalog.pg_authid'::pg_catalog.regclass);
SQL
)" )" || return 65
    [[ "${obtenido}" == 't' ]] || {
        printf 'el R0 sintético no coincide con el catálogo canónico\n' >&2
        return 65
    }
}

retirar_r0_sintetico_f0() {
    sql postgres 'BEGIN; DROP ROLE vec_f0_h0_publicador,vec_f0_h0_revocador,vec_f0_h0_despachador,vec_f0_h0_cruzado,vec_f0_h0_extra,vec_f0_h0_sin_rol; DROP ROLE vec_f0_h0_adicional,vec_contexto_actor_v1_publicador_corporativo,vec_contexto_actor_v1_revocador_corporativo,vec_contexto_actor_v1_despachador_corporativo; COMMIT' >/dev/null
}
