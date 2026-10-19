\set ON_ERROR_STOP on
-- CT242 / F1-47: la lectura exacta del seguimiento entrega la versión actual
-- y los datos mínimos de identificación para la segunda revisión de RRHH.
-- Conserva firma, propietario, RLS y ACL de CT115; no crea otro lector.
-- La consulta de incorporación acreditada (CT124) añade, dentro de la
-- propuesta pendiente, quién la hizo, para que el servidor no ofrezca
-- confirmarla a esa misma persona; el navegador solo recibe un booleano.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000242',0));

DO $pre$
DECLARE f oid := to_regprocedure('vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)'); p record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR f IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.numero_visible_vigente_v1(text,text)') IS NULL THEN
  RAISE EXCEPTION 'CT242: preimagen ausente' USING ERRCODE='55000'; END IF;
 -- La numeración se lee como propietario: debe seguir siendo suya, sin
 -- SECURITY DEFINER propio y sin ejecución para otros roles.
 IF NOT EXISTS (SELECT 1 FROM pg_proc
                 WHERE oid=to_regprocedure('vec_contratacion_temporal.numero_visible_vigente_v1(text,text)')
                   AND proowner='vec_contratacion_temporal_propietario'::regrole AND NOT prosecdef AND provolatile='s'
                   AND proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}') THEN
  RAISE EXCEPTION 'CT242: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig,
        encode(sha256(convert_to(pg_get_functiondef(oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']
    OR p.huella IS DISTINCT FROM 'd40b42b767e919c4d26d8e34bd9560cb21f733cee6e0cc10b0689044f324b5a0' THEN
  RAISE EXCEPTION 'CT242: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig,
        encode(sha256(convert_to(pg_get_functiondef(oid),'UTF8')),'hex') AS huella
 INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']
    OR p.huella IS DISTINCT FROM '73895d45c2c9188c26d3cfca2845ba8bc0e7ae42a7eed47977cc71ca989d2c62' THEN
  RAISE EXCEPTION 'CT242: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(p_organizacion text, p_expediente text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c record; k record; i record; e record;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT115: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT115: consulta inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion) THEN
        RAISE EXCEPTION 'CT115: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.ct115.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct115.expediente_ref',p_expediente,true);
    -- Fase y estado salen de las columnas de la versión vigente; el número es
    -- el mismo que muestra el cuadro de RRHH (numeración anual de CT142).
    SELECT a.version AS version_expediente, v.fase_clave, v.estado AS estado_clave,
           vec_contratacion_temporal.numero_visible_vigente_v1(p_expediente, v.agregado_json->>'numero_visible') AS numero_expediente
      INTO e FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref=a.expediente_ref AND v.version=a.version
     WHERE a.expediente_ref=p_expediente
       AND v.agregado_json->>'organizacion_ref'=p_organizacion;
    IF NOT FOUND OR e.version_expediente NOT BETWEEN 1 AND 9007199254740991
       OR e.fase_clave IS NULL OR e.estado_clave IS NULL
       OR e.numero_expediente IS NULL THEN
        RAISE EXCEPTION 'CT242: expediente no disponible' USING ERRCODE='23503';
    END IF;
    SELECT * INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO k FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_expediente_ct115(p_organizacion,p_expediente);
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.cese-cierre-expediente.v1',
        'expediente_ref',p_expediente,
        'version_expediente',e.version_expediente,'fase_clave',e.fase_clave,
        'estado_clave',e.estado_clave,'numero_expediente',e.numero_expediente,
        'incorporacion',CASE WHEN i.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo_ref',i.recibo_ref,'inicio',to_char(i.inicio,'YYYY-MM-DD')) END,
        'cese',CASE WHEN c.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('causa_clave',c.causa_clave,
            'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),'justificante_tipo',c.justificante_tipo,'justificante_ref',c.justificante_ref,
            'justificante_sha256',c.justificante_sha256,
            'recibo',jsonb_build_object('recibo_ref',c.recibo_json->'recibo_ref','registrada_en',c.recibo_json->'registrada_en')) END,
        'cierre',CASE WHEN k.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('condiciones',to_jsonb(k.condiciones),
            'ginpix_numero',k.ginpix_numero,'ginpix_confirmada_en',coalesce(to_char(k.ginpix_confirmada_en,'YYYY-MM-DD'),''),
            'recibo',jsonb_build_object('recibo_ref',k.recibo_json->'recibo_ref','registrada_en',k.recibo_json->'registrada_en')) END);
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(p_organizacion text, p_expediente text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE g record; c record; n record; p record;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT124: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT124: consulta inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion) THEN
        RAISE EXCEPTION 'CT124: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.ct115.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct115.expediente_ref',p_expediente,true);
    SELECT * INTO g FROM vec_contratacion_temporal.ginpix_confirmado_ct124(p_organizacion,p_expediente);
    SELECT * INTO c FROM vec_contratacion_temporal.incorporacion_centro_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO n FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente
     ORDER BY registrada_en DESC, recibo_ref COLLATE "C" DESC LIMIT 1;
    -- Propuesta pendiente de la segunda persona (cuatro ojos), si la hay.
    SELECT pp.propuesta_ref, pp.motivo_clave, pp.consecuencia_clave, pp.resolucion_ref, pp.resolucion_sha256, pp.fecha_notificacion, pp.registrada_en,
           pp.actor_ref AS propuesta_actor_ref
      INTO p
      FROM vec_contratacion_temporal.aceptacion_vigente_ct124(p_organizacion,p_expediente) av
     CROSS JOIN LATERAL vec_contratacion_temporal.propuesta_pendiente_ct124(av.resolucion_ref) pp;
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.incorporacion-acreditada.v1','expediente_ref',p_expediente,
        'ginpix',CASE WHEN g.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('ginpix_numero',g.ginpix_numero,
            'ginpix_confirmada_en',to_char(g.ginpix_confirmada_en,'YYYY-MM-DD'),
            'recibo',jsonb_build_object('recibo_ref',g.recibo_ref,'registrada_en',to_char(g.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END,
        'centro',CASE WHEN c.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('fecha_incorporacion',to_char(c.fecha_incorporacion,'YYYY-MM-DD'),
            'documento_tipo',c.documento_tipo,'documento_ref',c.documento_ref,'documento_sha256',c.documento_sha256,
            'recibo',jsonb_build_object('recibo_ref',c.recibo_ref,'registrada_en',to_char(c.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END,
        'no_incorporacion',CASE WHEN n.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('motivo_clave',n.motivo_clave,
            'consecuencia_clave',n.consecuencia_clave,'resolucion_ref',n.resolucion_ref,'resolucion_sha256',n.resolucion_sha256,
            'resuelta_por',n.resuelta_por,'fecha_notificacion',to_char(n.fecha_notificacion,'YYYY-MM-DD'),
            'recibo',jsonb_build_object('recibo_ref',n.recibo_ref,'registrada_en',to_char(n.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END,
        'no_incorporacion_propuesta',CASE WHEN p.propuesta_ref IS NULL THEN NULL ELSE jsonb_build_object('propuesta_ref',p.propuesta_ref,
            'motivo_clave',p.motivo_clave,'consecuencia_clave',p.consecuencia_clave,'resolucion_ref',p.resolucion_ref,
            'resolucion_sha256',p.resolucion_sha256,'fecha_notificacion',to_char(p.fecha_notificacion,'YYYY-MM-DD'),
            'registrada_en',to_char(p.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            -- Solo para el servidor: decide si quien consulta es quien propuso;
            -- nunca se entrega al navegador.
            'propuesta_actor_ref',p.propuesta_actor_ref) END);
END
$function$;

DO $post$
DECLARE p record;
BEGIN
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig
 INTO STRICT p FROM pg_proc
 WHERE oid='vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC']
    OR has_function_privilege('public', 'vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)', 'EXECUTE') THEN
  RAISE EXCEPTION 'CT242: ACL o postimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proacl::text AS acl,proconfig
 INTO STRICT p FROM pg_proc
 WHERE oid='vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 's'
    OR p.acl IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC']
    OR has_function_privilege('public', 'vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text)', 'EXECUTE') THEN
  RAISE EXCEPTION 'CT242: ACL o postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
