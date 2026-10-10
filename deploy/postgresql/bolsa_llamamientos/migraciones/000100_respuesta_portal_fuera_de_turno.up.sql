\set ON_ERROR_STOP on
-- B100. Una respuesta de Mi Bolsa (acepta o renuncia) que RRHH aún no ha
-- reflejado en la situación saca a la persona del turno B6. Hasta ahora solo
-- se guardaba en respuesta_portal_llamamiento: quien había aceptado en firme
-- seguía «disponible», salía como primera del orden y podía recibir otro
-- llamamiento. El orden vigente es la única autoridad del turno: lo usan el
-- «primero disponible», el asistente del llamamiento, la emisión B7 (exige
-- orden_vigente), las ofertas y la adjudicación telemática. La situación no
-- cambia sola: la refleja RRHH con «Cambiar situación» (B2/B8), y en cuanto
-- registra una situación posterior a la respuesta la persona vuelve a contar.
-- La marca «en revisión» (B41) pasa a cubrir también la aceptación, con el
-- mismo criterio de «sin reflejar» que el orden.
-- Reconstruye literalmente las definiciones instaladas (B90 y B41) y solo
-- añade el cálculo de respuestas pendientes; firma, OID y ACL se conservan.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000100',0));

DO $pre$
DECLARE n text; h text; v_actual text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'B100: clave=preimagen esperado=B30_B41_B90_instaladas actual=incompatible_o_ya_instalada'
   USING ERRCODE='55000';
 END IF;
 FOR n,h IN SELECT * FROM (VALUES
  ('leer_orden_vigente_bolsa_v1','6ac6c4f9319651e7557f8fff2a6f9724'),
  ('consultar_marcas_participaciones_v1','7d67f8279257a86b557698eaafe3712e')) AS t(n,h) LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
     WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname=n AND p.prosecdef
       AND pg_get_userbyid(p.proowner)=current_user AND md5(p.prosrc)=h) THEN
   SELECT string_agg(md5(p.prosrc),',' ORDER BY p.oid) INTO v_actual
     FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
    WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname=n;
   RAISE EXCEPTION 'B100: clave=%, actual=%, esperado=%', n, coalesce(v_actual,'ausente'), h USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;

-- Respuestas del portal que ninguna situación registrada después ha
-- reflejado, la más reciente por participación. Se compara con registrada_en
-- (instante del servidor al guardar), no con «desde», que puede fecharse
-- hacia atrás. Invocadora y sin concesiones: solo la llaman las lecturas
-- definidoras del propietario.
CREATE FUNCTION vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(p_bolsa_ref text, p_en timestamptz)
RETURNS TABLE(participacion_ref text, respuesta text, respondida_en timestamptz)
LANGUAGE sql STABLE SET search_path = pg_catalog, pg_temp SET timezone = 'UTC' AS $f$
 SELECT DISTINCT ON (r.participacion_ref) r.participacion_ref, r.respuesta, r.respondida_en
   FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  WHERE r.bolsa_ref = p_bolsa_ref AND r.respondida_en <= p_en
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                     WHERE s.participacion_ref = r.participacion_ref
                       AND s.registrada_en >= r.respondida_en AND s.registrada_en <= p_en)
  ORDER BY r.participacion_ref, r.respondida_en DESC, r.respuesta_ref DESC
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz) FROM PUBLIC;

-- B90 más «respuesta_pendiente»: no ocupa turno y su razón es
-- respuesta_portal_pendiente. La columna situacion no cambia.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa_ref text, p_en timestamp with time zone)
 RETURNS TABLE(politica_ref text, version_politica bigint, criterio text, tipo_lista text, reposicion text, provisional boolean, rotulo text, actor text, vigente_desde timestamp with time zone, participacion_ref text, orden_acta bigint, orden_vigente bigint, situacion text, razon text)
 LANGUAGE sql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 WITH politica AS (
  SELECT p.* FROM vec_bolsa_llamamientos.politica_orden_bolsa p
   WHERE p.bolsa_ref=p_bolsa_ref AND p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
   ORDER BY p.version DESC LIMIT 1
 ), ceses AS MATERIALIZED (
  SELECT ec.* FROM vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(
   coalesce((SELECT array_agg(e.participacion_ref) FROM vec_bolsa_llamamientos.constitucion c
     JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    WHERE c.bolsa_ref=p_bolsa_ref),ARRAY[]::text[]),p_en) ec
 ), pendientes AS MATERIALIZED (
  SELECT rp.participacion_ref FROM vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(p_bolsa_ref,p_en) rp
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         coalesce(rc.en_restriccion,false) AS cese_restringido,
         coalesce(rc.cese_pendiente,false) AS cese_pendiente,
         coalesce(rc.trabajo_cesado,false) AS trabajo_cesado,
         (pp.participacion_ref IS NOT NULL) AS respuesta_pendiente,
         ((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false)) AND NOT coalesce(rc.en_restriccion,false)
          AND NOT coalesce(rc.cese_pendiente,false)) AS turno_sin_respuesta,
         ((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false)) AND NOT coalesce(rc.en_restriccion,false)
          AND NOT coalesce(rc.cese_pendiente,false) AND pp.participacion_ref IS NULL) AS ocupa_turno,
         r.aplicada_en AS repuesta_en,
         EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion pe
                  WHERE pe.bolsa_ref=c.bolsa_ref AND pe.participacion_ref=e.participacion_ref AND pe.aplicada_en<=p_en
                    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                     WHERE rv.sancion_ref=pe.sancion_ref AND rv.registrada_en<=p_en)) AS penalizada
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN ceses rc ON rc.participacion_ref=e.participacion_ref
    LEFT JOIN pendientes pp ON pp.participacion_ref=e.participacion_ref
    LEFT JOIN LATERAL (SELECT ro.aplicada_en FROM vec_bolsa_llamamientos.reposicion_orden_bolsa ro WHERE ro.bolsa_ref=c.bolsa_ref AND ro.participacion_ref=e.participacion_ref AND ro.aplicada_en<=p_en ORDER BY ro.aplicada_en DESC LIMIT 1) r ON true
   WHERE c.bolsa_ref=p_bolsa_ref
 ), elegibles AS (
  SELECT b.participacion_ref,row_number() OVER(ORDER BY
    CASE WHEN b.penalizada THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN NULL ELSE b.orden_acta END,
    b.repuesta_en,b.orden_acta,b.participacion_ref)::bigint AS orden_vigente
   FROM base b CROSS JOIN politica p WHERE b.ocupa_turno
 )
 SELECT p.politica_ref,p.version,p.criterio,p.tipo_lista,p.reposicion,p.provisional,p.rotulo,p.actor,p.vigente_desde,
        b.participacion_ref,b.orden_acta,e.orden_vigente,
        CASE WHEN b.cese_pendiente AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'no_disponible'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'disponible_desde' WHEN b.trabajo_cesado THEN 'disponible'
             ELSE b.situacion END AS situacion,
        CASE WHEN b.cese_pendiente AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'cese_pendiente'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde') THEN 'restriccion_cese'
             WHEN b.respuesta_pendiente AND b.turno_sin_respuesta THEN 'respuesta_portal_pendiente'
             WHEN b.trabajo_cesado AND b.ocupa_turno THEN 'retorno_tras_cese'
             WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'
             WHEN NOT b.ocupa_turno AND b.situacion='trabajando' THEN 'trabajando'
             WHEN NOT b.ocupa_turno THEN 'sin_turno'
             WHEN b.penalizada THEN 'sancion_al_final'
             WHEN b.repuesta_en IS NOT NULL AND e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'reposicion_tras_contrato'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta
                  AND EXISTS (SELECT 1 FROM base o WHERE o.penalizada AND o.ocupa_turno AND o.orden_acta < b.orden_acta) THEN 'adelanta_por_sancion'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'pausa'
             ELSE 'orden_acta' END
   FROM base b CROSS JOIN politica p LEFT JOIN elegibles e USING(participacion_ref)
  ORDER BY e.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref
 $function$;

-- B41 con la aceptación: aceptacion_pendiente junto a renuncia_pendiente, y
-- ambas con el mismo criterio de «sin reflejar» que el orden.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(p_bolsa_ref text, p_corte timestamp with time zone)
 RETURNS TABLE(participacion_ref text, presta_servicios text, en_revision text, encadenamiento_dias bigint, encadenamiento_umbral_meses integer, encadenamiento_ventana_meses integer)
 LANGUAGE sql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET "TimeZone" TO 'UTC'
AS $function$
 WITH propias AS (
  SELECT s.participacion_ref, s.clave_persona FROM vec_bolsa_llamamientos.situaciones_en_v1(p_corte) s WHERE s.bolsa_ref = p_bolsa_ref
 ), presta AS (SELECT * FROM vec_bolsa_llamamientos.presta_servicios_en_v1(p_corte)),
 encadenadas AS (
  SELECT e.* FROM vec_bolsa_llamamientos.encadenamiento_en_v1(p_corte) e
   WHERE e.segundos > extract(epoch FROM p_corte - (p_corte - make_interval(months => e.umbral_meses)))
 ), revision AS (
  SELECT r.participacion_ref,
         CASE WHEN r.respuesta = 'acepta' THEN 'aceptacion_pendiente' ELSE 'renuncia_pendiente' END::text AS motivo
    FROM vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(p_bolsa_ref, p_corte) r
  UNION ALL
  SELECT s.participacion_ref, 'solicitud_pendiente'::text
    FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
   WHERE s.bolsa_ref = p_bolsa_ref AND s.registrada_en <= p_corte AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
 )
 SELECT o.participacion_ref, ps.modo,
        (SELECT min(r.motivo) FROM revision r WHERE r.participacion_ref = o.participacion_ref),
        floor(e.segundos / 86400)::bigint, e.umbral_meses, e.ventana_meses
   FROM propias o
   LEFT JOIN presta ps ON ps.participacion_ref = o.participacion_ref
   LEFT JOIN encadenadas e ON e.clave_persona = o.clave_persona
  WHERE ps.modo IS NOT NULL OR e.clave_persona IS NOT NULL
     OR EXISTS (SELECT 1 FROM revision r WHERE r.participacion_ref = o.participacion_ref)
  ORDER BY o.participacion_ref
  LIMIT 20000
$function$;

DO $post$
DECLARE esperado jsonb := jsonb_build_object(
 'owner_orden','vec_bolsa_llamamientos_propietario','owner_marcas','vec_bolsa_llamamientos_propietario',
 'owner_ayuda','vec_bolsa_llamamientos_propietario','definidoras',true,'ayuda_invocadora',true,
 'orden_ejecutor',true,'marcas_ejecutor',true,'ayuda_ejecutor',false,'ayuda_publica',false,
 'orden_publico',false,'marcas_publico',false);
 actual jsonb;
BEGIN
 SELECT jsonb_build_object(
 'owner_orden',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure),
 'owner_marcas',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)'::regprocedure),
 'owner_ayuda',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)'::regprocedure),
 'definidoras',(SELECT bool_and(prosecdef) FROM pg_proc WHERE oid IN (
   'vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure,
   'vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)'::regprocedure)),
 'ayuda_invocadora',(SELECT NOT prosecdef FROM pg_proc WHERE oid='vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)'::regprocedure),
 'orden_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)','EXECUTE'),
 'marcas_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)','EXECUTE'),
 'ayuda_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)','EXECUTE'),
 'ayuda_publica',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid='vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)'::regprocedure AND a.grantee=0),
 'orden_publico',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure AND a.grantee=0),
 'marcas_publico',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid='vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)'::regprocedure AND a.grantee=0))
 INTO actual;
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'B100: clave=postimagen esperado=% actual=%',esperado,actual USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
