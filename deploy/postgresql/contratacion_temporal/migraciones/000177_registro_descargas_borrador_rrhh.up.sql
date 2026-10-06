\set ON_ERROR_STOP on
-- CT177: registro de solo adición de las descargas de borradores RRHH (PDF y
-- DOCX) servidas por la consulta de detalle del expediente. Cada descarga
-- consume su propia decisión V3 (AD199) en la misma transacción que escribe la
-- fila: qué expediente y versión, qué tipo de borrador, en qué formato y el
-- SHA256 y el tamaño del archivo entregado. Tipo, formato y huella van dentro
-- de la huella del contexto del recurso que firma la decisión; la función la
-- recalcula antes de consumir, así que no se puede registrar otro documento
-- con una decisión ajena. La consulta del detalle conserva su propio consumo.
-- Una sola vez; sin DOWN. Requiere AD199.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000177',0));

DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT pg_catalog.has_function_privilege(current_user,
         'vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure,'EXECUTE')
    OR pg_catalog.to_regclass('vec_contratacion_temporal.descarga_borrador_rrhh_v1') IS NOT NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR pg_catalog.to_regtype('vec_contratacion_temporal.alcance_consulta_rrhh_v1') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_contratacion_temporal_consultor_rrhh'
                   AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT177: PARO clave=preimagen esperado=AD199_sin_CT177 actual=incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.descarga_borrador_rrhh_v1 (
 descarga_ref text PRIMARY KEY CHECK (descarga_ref ~ '^descarga_borrador:[0-9a-f]{64}$'),
 expediente_ref text NOT NULL CHECK (expediente_ref ~ '^expediente:[A-Za-z0-9._:/#-]{1,149}$'),
 version_expediente numeric(20,0) NOT NULL CHECK (version_expediente BETWEEN 1 AND 9007199254740991),
 tipo_borrador text NOT NULL CHECK (tipo_borrador ~ '^[a-z][a-z0-9_]{1,63}$'),
 formato text NOT NULL CHECK (formato IN ('pdf','docx')),
 documento_sha256 text NOT NULL CHECK (documento_sha256 ~ '^[0-9a-f]{64}$'),
 tamano_bytes integer NOT NULL CHECK (tamano_bytes BETWEEN 1 AND 16777216),
 consulta_huella_sha256 text NOT NULL CHECK (consulta_huella_sha256 ~ '^[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
 contexto_huella_sha256 text NOT NULL CHECK (contexto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(registrada_en)),
 FOREIGN KEY (expediente_ref,version_expediente)
  REFERENCES vec_contratacion_temporal.expediente_version_integral(expediente_ref,version)
);
CREATE INDEX descarga_borrador_rrhh_v1_expediente_idx
 ON vec_contratacion_temporal.descarga_borrador_rrhh_v1(expediente_ref,registrada_en);
ALTER TABLE vec_contratacion_temporal.descarga_borrador_rrhh_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.descarga_borrador_rrhh_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY descarga_borrador_rrhh_v1_propietario ON vec_contratacion_temporal.descarga_borrador_rrhh_v1
 TO vec_contratacion_temporal_propietario
 USING (current_user='vec_contratacion_temporal_propietario')
 WITH CHECK (current_user='vec_contratacion_temporal_propietario');
REVOKE ALL ON vec_contratacion_temporal.descarga_borrador_rrhh_v1 FROM PUBLIC;
CREATE TRIGGER descarga_borrador_rrhh_v1_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_contratacion_temporal.descarga_borrador_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Canon de la descarga. Los valores tienen forma cerrada y no contienen saltos
-- de línea; Go calcula la misma cadena (ports.CanonDescargaBorradorRRHH).
CREATE FUNCTION vec_contratacion_temporal.canon_descarga_borrador_rrhh_v1(
 p_expediente_ref text,p_version numeric,p_tipo text,p_formato text,
 p_documento_sha256 text,p_tamano integer,p_consulta_huella text)
RETURNS bytea LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.convert_to(
  'vec.contratacion_temporal.descarga_borrador_rrhh.v1' || E'\n' || p_expediente_ref || E'\n' ||
  p_version::text || E'\n' || p_tipo || E'\n' || p_formato || E'\n' || p_documento_sha256 || E'\n' ||
  p_tamano::text || E'\n' || p_consulta_huella,'UTF8')
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_descarga_borrador_rrhh_v1(text,numeric,text,text,text,integer,text) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.registrar_descarga_borrador_rrhh_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_expediente_ref text,p_version numeric,p_tipo text,p_formato text,
 p_documento_sha256 text,p_tamano integer,p_consulta_huella text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(descarga_ref text,auditoria_ref text,decision_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $f$
DECLARE v_login pg_catalog.pg_roles%ROWTYPE; d jsonb; c jsonb; a jsonb;
 v_contexto text; v_huella text; v_consumo record; v_ref text;
BEGIN
 SELECT * INTO v_login FROM pg_catalog.pg_roles WHERE rolname=SESSION_USER;
 -- Misma sesión que la consulta de detalle: un LOGIN con una sola membresía,
 -- la del consultor RRHH de CT, sin atributos privilegiados.
 IF CURRENT_USER<>'vec_contratacion_temporal_propietario' OR SESSION_USER=CURRENT_USER
    OR v_login.oid IS NULL OR NOT v_login.rolcanlogin OR NOT v_login.rolinherit
    OR v_login.rolsuper OR v_login.rolcreatedb OR v_login.rolcreaterole
    OR v_login.rolreplication OR v_login.rolbypassrls
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=v_login.oid)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
                   WHERE m.member=v_login.oid AND r.rolname='vec_contratacion_temporal_consultor_rrhh'
                     AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option) THEN
  RAISE EXCEPTION 'CT177: descarga de borrador denegada' USING ERRCODE='42501';
 END IF;
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT177: transacción de descarga no admitida' USING ERRCODE='55000';
 END IF;
 IF p_alcance IS NULL
    OR p_alcance.organizacion_ref IS NULL OR p_alcance.organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_alcance.clase_ambito IS NULL OR p_alcance.clase_ambito !~ '^[a-z][a-z_]{1,63}$'
    OR p_alcance.ambito_ref IS NULL OR p_alcance.ambito_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^expediente:[A-Za-z0-9._:/#-]{1,149}$'
    OR p_version IS NULL OR p_version<>pg_catalog.trunc(p_version) OR p_version NOT BETWEEN 1 AND 9007199254740991
    OR p_tipo IS NULL OR p_tipo !~ '^[a-z][a-z0-9_]{1,63}$'
    OR p_formato IS NULL OR p_formato NOT IN ('pdf','docx')
    OR p_documento_sha256 IS NULL OR p_documento_sha256 !~ '^[0-9a-f]{64}$'
    OR p_tamano IS NULL OR p_tamano NOT BETWEEN 1 AND 16777216
    OR p_consulta_huella IS NULL OR p_consulta_huella !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CT177: descarga inválida' USING ERRCODE='22023';
 END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
  a:=pg_catalog.convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT177: descarga de borrador denegada' USING ERRCODE='42501'; END;
 -- Mismo contexto canónico que RecursoAutorizable.HuellaContextoAutorizacionSHA256:
 -- claves ordenadas, sin espacios. Se concatena sin escapar porque las
 -- expresiones regulares de arriba excluyen comillas y barras; si se relajan,
 -- hay que construirlo con jsonb y el mismo orden de claves.
 v_contexto:='{"ambitos":{"ambito_ref":"'||p_alcance.ambito_ref||'","clase_ambito":"'||p_alcance.clase_ambito
  ||'","organizacion_ref":"'||p_alcance.organizacion_ref||'"},"atributos":{"consulta_dominio":"'
  ||'vec.contratacion_temporal.descarga_borrador_rrhh.v1'||'","consulta_huella_sha256":"'
  ||pg_catalog.encode(pg_catalog.sha256(vec_contratacion_temporal.canon_descarga_borrador_rrhh_v1(
     p_expediente_ref,p_version,p_tipo,p_formato,p_documento_sha256,p_tamano,p_consulta_huella)),'hex')||'"}}';
 v_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_contexto,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.borrador_rrhh.descargar'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM p_expediente_ref
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_huella
    OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR d->>'principal_id' IS NULL OR d->>'principal_id' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR d->>'principal_id' IS DISTINCT FROM a->>'principal_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_activo_ref'
    OR a->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR a->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR c->>'efecto_ref' IS DISTINCT FROM p_expediente_ref
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_huella THEN
  RAISE EXCEPTION 'CT177: descarga de borrador denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM p_expediente_ref
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_huella
    OR v_consumo.decision_ref IS NULL OR v_consumo.auditoria_ref IS NULL THEN
  RAISE EXCEPTION 'CT177: consumo de descarga divergente' USING ERRCODE='42501';
 END IF;
 v_ref:='descarga_borrador:'||v_consumo.consumo_huella_sha256;
 -- La versión debe existir: el FK lo comprueba dentro de esta transacción.
 INSERT INTO vec_contratacion_temporal.descarga_borrador_rrhh_v1(
  descarga_ref,expediente_ref,version_expediente,tipo_borrador,formato,documento_sha256,tamano_bytes,
  consulta_huella_sha256,organizacion_ref,actor_ref,decision_ref,auditoria_ref,contexto_huella_sha256,registrada_en)
 VALUES (v_ref,p_expediente_ref,p_version,p_tipo,p_formato,p_documento_sha256,p_tamano,
  p_consulta_huella,p_alcance.organizacion_ref,d->>'principal_id',v_consumo.decision_ref,v_consumo.auditoria_ref,
  v_huella,pg_catalog.date_trunc('microseconds',v_consumo.consumida_en));
 RETURN QUERY SELECT v_ref,v_consumo.auditoria_ref,v_consumo.decision_ref,
  pg_catalog.date_trunc('microseconds',v_consumo.consumida_en);
EXCEPTION
 WHEN foreign_key_violation THEN
  RAISE EXCEPTION 'CT177: versión de expediente inexistente' USING ERRCODE='P0002';
END $f$;

DO $acl$
DECLARE f pg_catalog.regprocedure:='vec_contratacion_temporal.registrar_descarga_borrador_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,text,numeric,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc q
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
  WHERE q.oid=f AND a.grantee<>q.proowner LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_consultor_rrhh',f::text);
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::pg_catalog.regrole
    OR NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f)
    OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_consultor_rrhh',f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc q
         CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
         WHERE q.oid=f AND a.grantee NOT IN (q.proowner,'vec_contratacion_temporal_consultor_rrhh'::pg_catalog.regrole))<>0
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc q
         CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
         WHERE q.oid='vec_contratacion_temporal.canon_descarga_borrador_rrhh_v1(text,numeric,text,text,text,integer,text)'::pg_catalog.regprocedure
           AND a.grantee<>q.proowner)
    OR pg_catalog.has_table_privilege('vec_contratacion_temporal_consultor_rrhh','vec_contratacion_temporal.descarga_borrador_rrhh_v1','SELECT')
 THEN RAISE EXCEPTION 'CT177: PARO clave=ACL esperado=propietario_y_consultor_rrhh actual=ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
