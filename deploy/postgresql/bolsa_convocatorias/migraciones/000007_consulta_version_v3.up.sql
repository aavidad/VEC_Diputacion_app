\set ON_ERROR_STOP on
-- S1: lectura exacta de Bolsa por una concesión V3 nueva, sin abrir V1.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000007',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.lectura_version_v3') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_convocatorias_ejecutor_consulta' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta','vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Bolsa007: dependencia o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_convocatorias.lectura_version_v3 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref~'^lectura_convocatoria_[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 convocatoria_id text NOT NULL,
 secuencia bigint NOT NULL CHECK(secuencia BETWEEN 1 AND 9007199254740991),
 resultado text NOT NULL CHECK(resultado IN ('obtenida','no_encontrada')),
 huella_version_sha256 text CHECK(huella_version_sha256~'^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL,
 correlacion_ref text NOT NULL CHECK(correlacion_ref~'^correlacion_[0-9a-f]{32}$'),
 consultada_en timestamptz(6) NOT NULL,
 CHECK((resultado='obtenida')=(huella_version_sha256 IS NOT NULL))
);
REVOKE ALL ON TABLE vec_bolsa_convocatorias.lectura_version_v3 FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_consulta;
REVOKE ALL ON TYPE vec_bolsa_convocatorias.lectura_version_v3 FROM PUBLIC;
ALTER TABLE vec_bolsa_convocatorias.lectura_version_v3 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_convocatorias.lectura_version_v3 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_lectura_v3 ON vec_bolsa_convocatorias.lectura_version_v3
 FOR ALL TO vec_bolsa_convocatorias_propietario
 USING(current_user='vec_bolsa_convocatorias_propietario')
 WITH CHECK(current_user='vec_bolsa_convocatorias_propietario');
CREATE TRIGGER lectura_version_v3_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_convocatorias.lectura_version_v3 FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
CREATE TRIGGER lectura_version_v3_no_truncar BEFORE TRUNCATE
 ON vec_bolsa_convocatorias.lectura_version_v3 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable();

CREATE FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v3(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(resultado text,version_canonica bytea,huella_version_sha256 text,
 decision_ref text,consumo_huella_sha256 text,auditoria_ref text,recibo_ref text,
 correlacion_ref text,consultada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $fn$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; consumo record;
 id text; sec bigint; referencia text; material_canon text; material_sha text; recurso_canon text; recurso_sha text;
 fuente vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 encontrada boolean; estado_lectura text; recibo text; instante timestamptz(6);
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication)
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid='vec_bolsa_convocatorias_ejecutor_consulta'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_bolsa_convocatorias_ejecutor_consulta'::regrole)
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 8192
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
 OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
 OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
 OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
 OR p_raiz IS NULL OR octet_length(p_raiz)<>44
 THEN RAISE EXCEPTION 'consulta V3 de convocatoria rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;contexto:=convert_from(p_contexto,'UTF8')::jsonb;
  id:=m->>'convocatoria_id';sec:=(m->>'secuencia')::bigint;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'material V3 de convocatoria inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
 'actor_ref','contexto_actor_ref','contexto_version','convocatoria_id','correlacion_ref','esquema','perfil_ref','perfil_version','persona_version','secuencia']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.convocatoria.consulta-version.v3'
 OR id IS NULL OR length(id) NOT BETWEEN 1 AND 480 OR id !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]*$'
 OR sec IS NULL OR sec NOT BETWEEN 1 AND 9007199254740991 OR sec::text IS DISTINCT FROM m->>'secuencia'
 OR (m->>'actor_ref') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR (m->>'perfil_ref') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR (m->>'contexto_actor_ref') !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR (m->>'correlacion_ref') !~ '^correlacion_[0-9a-f]{32}$'
 OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
 OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
 OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 OR m->>'contexto_actor_ref' IS DISTINCT FROM contexto->>'contexto_actor_ref'
 OR m->>'contexto_version' IS DISTINCT FROM contexto->>'contexto_version'
 OR m->>'actor_ref' IS DISTINCT FROM contexto->>'principal_ref'
 OR m->>'persona_version' IS DISTINCT FROM contexto->>'persona_version'
 OR m->>'perfil_version' IS DISTINCT FROM contexto->>'perfil_version'
 OR m->>'perfil_ref' IS DISTINCT FROM contexto->>'perfil_activo_ref'
 OR m->>'correlacion_ref' IS DISTINCT FROM d->>'correlacion_ref'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.convocatoria.version.consultar'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'version_convocatoria_gobernada'
 OR d->>'finalidad' IS DISTINCT FROM 'consulta_interna_convocatorias'
 OR d->'campos_permitidos' IS DISTINCT FROM '["version_convocatoria"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_convocatorias.version.consultar.v1'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.convocatoria.version.consultar'
 THEN RAISE EXCEPTION 'material y operación V3 divergentes' USING ERRCODE='42501'; END IF;
 referencia:=id||'#'||sec::text;
 material_canon:='{"esquema":"vec.bolsa.convocatoria.consulta-version.v3","convocatoria_id":'||to_jsonb(id)::text||',"secuencia":'||sec::text||
 ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
 ',"contexto_version":'||(m->>'contexto_version')||',"persona_version":'||(m->>'persona_version')||
 ',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||',"perfil_version":'||(m->>'perfil_version')||
 ',"correlacion_ref":'||to_jsonb(m->>'correlacion_ref')::text||'}';
 IF material_canon IS DISTINCT FROM p_material THEN
  RAISE EXCEPTION 'material V3 no canónico' USING ERRCODE='22023';
 END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 recurso_canon:='{"ambitos":{"convocatoria_id":'||to_jsonb(id)::text||',"secuencia":'||to_jsonb(sec::text)::text||'},"atributos":{"material_sha256":"'||material_sha||'"}}';
 recurso_sha:=encode(sha256(convert_to(recurso_canon,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM referencia
 OR c->>'efecto_ref' IS DISTINCT FROM referencia
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
 THEN RAISE EXCEPTION 'selector exacto V3 divergente' USING ERRCODE='42501'; END IF;

 -- AD141 valida firma, gobierno, sesión y versiones vigentes. Su consumo y
 -- auditoría permanecen en esta transacción, antes de leer la fuente de Bolsa.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM referencia
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
 THEN RAISE EXCEPTION 'consumo V3 de convocatoria incompatible' USING ERRCODE='42501'; END IF;

 SELECT v.* INTO fuente FROM vec_bolsa_convocatorias.version_convocatoria v
 WHERE v.convocatoria_id=id AND v.secuencia=sec;
 encontrada:=FOUND;
 IF encontrada AND (octet_length(fuente.version_canonica)>33554432 OR
 encode(sha256(fuente.version_canonica),'hex') IS DISTINCT FROM fuente.huella_version_sha256) THEN
  RAISE EXCEPTION 'fuente exacta de convocatoria no disponible' USING ERRCODE='55000';
 END IF;
 estado_lectura:=CASE WHEN encontrada THEN 'obtenida' ELSE 'no_encontrada' END;
 instante:=consumo.consumida_en;
 recibo:='lectura_convocatoria_'||encode(sha256(convert_to(consumo.decision_ref||':'||consumo.consumo_huella_sha256,'UTF8')),'hex');
 INSERT INTO vec_bolsa_convocatorias.lectura_version_v3(
 recibo_ref,decision_ref,convocatoria_id,secuencia,resultado,huella_version_sha256,material_sha256,
 consumo_huella_sha256,auditoria_ref,correlacion_ref,consultada_en)
 VALUES(recibo,consumo.decision_ref,id,sec,estado_lectura,CASE WHEN encontrada THEN fuente.huella_version_sha256 END,
 material_sha,consumo.consumo_huella_sha256,consumo.auditoria_ref,m->>'correlacion_ref',instante);
 RETURN QUERY SELECT estado_lectura,CASE WHEN encontrada THEN fuente.version_canonica END,
 CASE WHEN encontrada THEN fuente.huella_version_sha256 END,consumo.decision_ref::text,
 consumo.consumo_huella_sha256::text,consumo.auditoria_ref::text,recibo,m->>'correlacion_ref',instante;
END $fn$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_convocatorias TO vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_convocatorias_ejecutor_consulta;
COMMENT ON FUNCTION vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 IS 'Lectura exacta nueva con consumo nominal AD141 y recibo durable; no abre la consulta V1 ni acredita fases u OEP ausentes.';
COMMIT;
