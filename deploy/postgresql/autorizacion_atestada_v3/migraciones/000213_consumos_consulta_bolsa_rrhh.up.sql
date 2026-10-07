\set ON_ERROR_STOP on
-- AD213. Tres lecturas RRHH de Bolsa con acciones y audiencias propias.
-- No crea concesiones humanas; sólo habilita el consumo exacto para B84.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000213',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD213: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- AD214, dueño del núcleo y del CHECK de audiencias, debe preceder a AD213.
-- Esta migración sólo añade la fachada nominal y su ACL; no modifica el
-- material V3, la historia ni la autoridad común.
DO $post214$
DECLARE
 nucleo text;
 fuente text;
 audiencias text;
 f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 propietario oid;
 definidora boolean;
 configuracion text[];
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,p.proowner,p.prosecdef,p.proconfig
 INTO STRICT nucleo,fuente,propietario,definidora,configuracion
 FROM pg_catalog.pg_proc p WHERE p.oid=f;
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT audiencias
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check'
   AND c.contype='c' AND c.convalidated;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR NOT definidora
    OR configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nucleo,'UTF8')),'hex')
       IS DISTINCT FROM '63afb3d4e6f33d4ee8efce1e54d7e8f92ac9f8cec58506c2af146e88dca4552e'
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex')
       IS DISTINCT FROM '78137d8750422597c0da56797fd3ddff797cd54f18d98b0c1c8f4b7f742079dd'
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(audiencias,'UTF8')),'hex')
       IS DISTINCT FROM '8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e'
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=f AND a.grantee=0)
    OR pg_catalog.pg_get_function_result(f) IS DISTINCT FROM
       'TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)'
    OR pg_catalog.strpos(nucleo,'consulta_rrhh_bolsa')=0
    OR pg_catalog.strpos(nucleo,'bolsa.rrhh.bolsas.consultar')=0
    OR pg_catalog.strpos(nucleo,'bolsa.rrhh.estadisticas.consultar')=0
    OR pg_catalog.strpos(nucleo,'bolsa.rrhh.candidatos.consultar')=0
    OR pg_catalog.strpos(audiencias,'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1')=0
    OR pg_catalog.strpos(audiencias,'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1')=0
    OR pg_catalog.strpos(audiencias,'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD213: exige AD214 con tres acciones y audiencias Bolsa' USING ERRCODE='55000'; END IF;
END $post214$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(
 p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD213: material invalido' USING ERRCODE='22023'; END;
 IF p_accion NOT IN ('bolsa.rrhh.bolsas.consultar','bolsa.rrhh.estadisticas.consultar','bolsa.rrhh.candidatos.consultar')
    OR c->>'operacion' IS DISTINCT FROM p_accion
    OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'AD213: consulta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'consulta_rrhh_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD213: consulta requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=f AND a.grantee=0)
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD213: ACL incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
