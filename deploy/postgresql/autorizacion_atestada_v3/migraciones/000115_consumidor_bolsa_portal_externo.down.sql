\set ON_ERROR_STOP on
-- Solo para un clon desechable sin login miembro del rol exterior. Nunca
-- retirar de una base con historia de uso del portal del candidato.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000115',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        original text; nuevo text; metadata jsonb; dependencias jsonb;
        anterior text := E'               AND pg_catalog.pg_has_role(\n                   session_user, ''vec_bolsa_llamamientos_ejecutor'', ''MEMBER'')';
        ampliado text := E'               AND ((pg_catalog.pg_has_role(\n                   session_user, ''vec_bolsa_llamamientos_ejecutor'', ''MEMBER'')\n                        AND NOT pg_catalog.pg_has_role(\n                           session_user, ''vec_bolsa_llamamientos_portal_externo'', ''MEMBER''))\n                   OR ((p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_participaciones_propias_bolsa''\n                        OR p_perfil_mutacion IS NOT DISTINCT FROM ''portal_candidato_bolsa'')\n                       AND pg_catalog.pg_has_role(\n                           session_user, ''vec_bolsa_llamamientos_portal_externo'', ''MEMBER'')\n                       AND NOT pg_catalog.pg_has_role(\n                           session_user, ''vec_bolsa_llamamientos_ejecutor'', ''MEMBER'')))';
        inicio text := E'       OR session_user = current_user\n       OR NOT (';
        inicioExterno text := E'       OR session_user = current_user\n       OR (pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_portal_externo'', ''MEMBER'')\n           AND (p_perfil_mutacion IS DISTINCT FROM ''consulta_participaciones_propias_bolsa''\n                AND p_perfil_mutacion IS DISTINCT FROM ''portal_candidato_bolsa''\n                OR EXISTS (WITH RECURSIVE roles(rol_id) AS (\n                     SELECT m.roleid FROM pg_catalog.pg_auth_members m\n                      WHERE m.member=session_user::pg_catalog.regrole\n                     UNION\n                     SELECT m.roleid FROM pg_catalog.pg_auth_members m\n                      JOIN roles r ON r.rol_id=m.member)\n                    SELECT 1 FROM roles\n                     WHERE rol_id<>''vec_bolsa_llamamientos_portal_externo''::pg_catalog.regrole)))\n       OR NOT (';
BEGIN
 IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_bolsa_llamamientos_portal_externo'::regrole)
 THEN RAISE EXCEPTION 'AD3-115: retirada denegada' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,metadata FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO dependencias FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,ampliado,''))<>length(ampliado)
    OR length(original)-length(replace(original,inicioExterno,''))<>length(inicioExterno)
 THEN RAISE EXCEPTION 'AD3-115: guarda incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(original,ampliado,anterior),inicioExterno,inicio);
 EXECUTE nuevo;
 IF (SELECT pg_get_functiondef(f) FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM nuevo
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM dependencias
 THEN RAISE EXCEPTION 'AD3-115: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
