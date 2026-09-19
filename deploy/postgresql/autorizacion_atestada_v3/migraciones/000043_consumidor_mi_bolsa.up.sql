\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000043', 0));

DO $precondicion$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN
        RAISE EXCEPTION 'estado incompatible para consumidor Mi bolsa V3' USING ERRCODE='55000';
    END IF;
END $precondicion$;

-- Base mínima AD3-38: esta migración no presupone las reservas 39–42. Toda migración
-- posterior rescatada debe preservar esta audiencia y sus dos guardas runtime de Mi bolsa.
-- Solo se añade el perfil nominal al núcleo vigente. El núcleo conserva sus
-- bloqueos, revalidación de sesión/contexto/vínculo/versiones/vigencia,
-- revocación, HMAC, COSE y auditoría dentro de la transacción.
DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    definicion text; metadata jsonb; dependencias jsonb;
    guarda_ct text := $ct$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'$ct$;
    guarda_bolsa text := $bolsa$               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')$bolsa$;
    guarda_ct_nueva text := $ctnuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'$ctnuevo$;
    guarda_bolsa_nueva text := $bolsanuevo$               (p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')$bolsanuevo$;
    marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.mi-bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'consulta_participaciones_propias'
           )
$perfil$;
BEGIN
    SELECT pg_get_functiondef(p.oid), to_jsonb(p)-'prosrc' INTO STRICT definicion,metadata FROM pg_proc p WHERE p.oid=f;
    SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO dependencias FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
    IF length(definicion)-length(replace(definicion,marca,''))<>length(marca)
       OR strpos(definicion,'consulta_participaciones_propias_bolsa')<>0
       OR strpos(definicion,'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva')=0 THEN
       RAISE EXCEPTION 'núcleo incompatible para Mi bolsa V3' USING ERRCODE='55000';
    END IF;
    definicion := replace(definicion,marca,extension||marca);
    IF length(definicion)-length(replace(definicion,guarda_ct,''))<>length(guarda_ct)
       OR length(definicion)-length(replace(definicion,guarda_bolsa,''))<>length(guarda_bolsa) THEN
       RAISE EXCEPTION 'guardas runtime incompatibles para Mi bolsa V3' USING ERRCODE='55000';
    END IF;
    definicion := replace(definicion,guarda_ct,guarda_ct_nueva);
    definicion := replace(definicion,guarda_bolsa,guarda_bolsa_nueva);
    EXECUTE definicion;
    IF length(definicion)-length(replace(definicion,guarda_ct_nueva,''))<>length(guarda_ct_nueva)
       OR length(definicion)-length(replace(definicion,guarda_bolsa_nueva,''))<>length(guarda_bolsa_nueva) THEN
       RAISE EXCEPTION 'Mi bolsa no conserva las guardas runtime segregadas' USING ERRCODE='55000';
    END IF;
    IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM dependencias THEN
       RAISE EXCEPTION 'Mi bolsa alteró metadata o dependencias del núcleo' USING ERRCODE='55000';
    END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE definicion text; nueva text;
BEGIN
    SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
    IF strpos(definicion,'vec.bolsa.mi-bolsa.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN
       RAISE EXCEPTION 'audiencias incompatibles para Mi bolsa V3' USING ERRCODE='55000';
    END IF;
    nueva := left(definicion,length(definicion)-3)||', ''vec.bolsa.mi-bolsa.v1''::text]))';
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
    BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN RAISE EXCEPTION 'material Mi bolsa inválido' USING ERRCODE='22023'; END;
    IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.mi-bolsa.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.participaciones_propias.consultar'
       OR d->>'accion' IS DISTINCT FROM 'bolsa.participaciones_propias.consultar' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato' OR d->>'finalidad' IS DISTINCT FROM 'consulta_participaciones_propias'
       OR jsonb_array_length(coalesce(d->'campos_permitidos','[]'::jsonb))<>1 OR d->'campos_permitidos'->>0 IS DISTINCT FROM 'participaciones_candidato_minimizadas'
       OR jsonb_array_length(coalesce(d->'obligaciones','[]'::jsonb))<>0 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
        RAISE EXCEPTION 'material Mi bolsa rechazado' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('consulta_participaciones_propias_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'Mi bolsa requiere consumo nuevo' USING ERRCODE='42501'; END IF;
    RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
COMMIT;
