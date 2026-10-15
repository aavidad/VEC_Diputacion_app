\set ON_ERROR_STOP on
-- AD281: la custodia externa de Bolsa usa solo la pareja nominal AUT99.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000281',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE; f text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 THEN RAISE EXCEPTION 'AD281: preimagen estructural incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT unnest(ARRAY[
  'vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric)',
  'vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric)']) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f);
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
  OR p.prosecdef IS NOT TRUE
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR p.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
  THEN RAISE EXCEPTION 'AD281: pareja nominal incompatible %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
 '7d13731f452b0f8d7020ea0ab602c465897617fce61a84a7887491309e1bbb48'
 THEN RAISE EXCEPTION 'AD281: cuerpo central divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $patch$
DECLARE f text;old1 text;new1 text;old2 text;new2 text;h text;
BEGIN
 f:=pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 old1:=$m$    ELSE
        SELECT * INTO v_registro
          FROM vec_autorizacion.
               registrar_y_revalidar_decision_contexto_actor_v3(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;$m$;
 new1:=$m$    ELSIF p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_externa' THEN
        SELECT * INTO v_registro
          FROM vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    ELSE
        SELECT * INTO v_registro
          FROM vec_autorizacion.
               registrar_y_revalidar_decision_contexto_actor_v3(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;$m$;
 old2:=$m$    ELSE
        v_revalidada_en :=
          vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;$m$;
 new2:=$m$    ELSIF p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_bolsa_externa' THEN
        v_revalidada_en :=
          vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    ELSE
        v_revalidada_en :=
          vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;$m$;
 IF (length(f)-length(replace(f,old1,'')))/length(old1)<>1
 OR (length(f)-length(replace(f,old2,'')))/length(old2)<>1
 THEN RAISE EXCEPTION 'AD281: marcas centrales divergentes' USING ERRCODE='55000'; END IF;
 f:=replace(replace(f,old1,new1),old2,new2);
 h:=encode(sha256(convert_to(f,'UTF8')),'hex');
 EXECUTE f;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'AD281: posthuella divergente' USING ERRCODE='55000'; END IF;
END $patch$;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD281: propietario, flags o ACL alterados' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
