\set ON_ERROR_STOP on
-- B90. Un cese B13 aún sin restricción B45 mantiene al candidato pendiente.
-- La condición nace del primer evento durable, aunque B81 todavía no se haya
-- proyectado o B8 vincule al candidato antes del siguiente ciclo del relevo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000090',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.cese_ajeno_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.cese_sin_candidato_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.candidatos_cese_pendiente_b90') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(text,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'B90: clave=preimagen_pendiente esperado=B13_B45_B50_B81_instaladas actual=incompatible_o_ya_instalada'
   USING ERRCODE='55000';
 END IF;
END $pre$;

-- Vista privada de hechos: los lectores aplican su propio corte temporal.
-- La proyección B81 no resuelve el cese; una entrega B13 en cuarentena sigue
-- pendiente. El instante recibido_en fija desde cuándo Bolsa conoce el hecho.
CREATE VIEW vec_bolsa_llamamientos.candidatos_cese_pendiente_b90 AS
 SELECT vc.candidato_ref,cp.participacion_ref,cp.evento_ref,cp.origen_ref,
        cp.huella_sha256,cp.origen_posicion,cp.ocurrido_en,cp.origen_creada_en,
        cp.recibido_en,r.recibida_en AS restriccion_recibida_en,
        a.recibido_en AS ajeno_recibido_en
 FROM vec_bolsa_llamamientos.contrato_participacion cp
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=cp.participacion_ref
 LEFT JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa r
   ON r.evento_ref=cp.evento_ref AND r.origen_ref=cp.origen_ref
 LEFT JOIN vec_bolsa_llamamientos.cese_ajeno_bolsa a
   ON a.origen_ref=cp.origen_ref AND a.llamamiento_ref=cp.llamamiento_ref
 WHERE cp.tipo='cese' AND cp.participacion_ref IS NOT NULL
 ;
REVOKE ALL ON vec_bolsa_llamamientos.candidatos_cese_pendiente_b90 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.candidatos_cese_pendiente_b90 FROM PUBLIC;

-- B50 conserva la fachada v1 para sus consumidores históricos. Esta v2
-- añade el estado pendiente sin inventar fecha de disponibilidad. La frontera
-- Go autorizada lee una fila incluso cuando B45 aún no tiene restricción.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(
 p_participacion_ref text,p_corte timestamptz)
RETURNS TABLE(fecha_efecto date,disponible_desde date,en_restriccion boolean,
 trabajo_cesado boolean,cese_pendiente boolean,pendiente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET statement_timeout='5s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_participacion_ref IS NULL OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 512
    OR p_corte IS NULL OR NOT isfinite(p_corte) THEN
  RAISE EXCEPTION 'B90: consulta de estado de cese denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 WITH titular AS (
  SELECT vc.candidato_ref FROM vec_bolsa_llamamientos.vinculo_candidato vc
   WHERE vc.participacion_ref=p_participacion_ref
 ), pendiente AS (
  SELECT min(cp.recibido_en) AS desde
    FROM vec_bolsa_llamamientos.candidatos_cese_pendiente_b90 cp
    JOIN titular t ON t.candidato_ref=cp.candidato_ref
   WHERE cp.ocurrido_en<=p_corte AND cp.origen_creada_en<=p_corte
     AND cp.recibido_en<=p_corte
     AND (cp.restriccion_recibida_en IS NULL OR cp.restriccion_recibida_en>p_corte)
     AND (cp.ajeno_recibido_en IS NULL OR cp.ajeno_recibido_en>p_corte)
 )
 SELECT ec.fecha_efecto,ec.disponible_desde,coalesce(ec.en_restriccion,false),
        coalesce(ec.trabajo_cesado,false),p.desde IS NOT NULL,p.desde
 FROM pendiente p
 LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(p_participacion_ref,p_corte) ec ON true
 WHERE p.desde IS NOT NULL OR ec.fecha_efecto IS NOT NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(text,timestamptz)
 TO vec_bolsa_llamamientos_ejecutor;

DO $post$
DECLARE f regprocedure:='vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2(text,timestamptz)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT 'search_path=pg_catalog, pg_temp'=ANY(proconfig) FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text) FROM pg_proc p
        CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)
       IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos_propietario']
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',
       'vec_bolsa_llamamientos.candidatos_cese_pendiente_b90','SELECT')
    OR has_table_privilege('vec_bolsa_llamamientos_relevo_cese',
       'vec_bolsa_llamamientos.candidatos_cese_pendiente_b90','SELECT') THEN
  RAISE EXCEPTION 'B90: clave=postimagen_pendiente esperado=solo_owner_y_fachada_ejecutor actual=ACL_o_definicion_divergente'
   USING ERRCODE='55000';
 END IF;
END $post$;
-- Lecturas de orden, resumen y lote del mismo pendiente B13, en esta transacción.
DO $pre$
DECLARE actual jsonb; esperado jsonb:=jsonb_build_object(
 'rol',true,'pendiente_b90',true,'vinculo',true,'restriccion',true,
 'contrato',true,'situacion',true,'orden_b6',true,'resumen_b82',true,'lote_libre',true,
 'def_orden_sha256','701b9c64abe2c08f53de224343b174d056d2f7b61a414ac87a936e435a6d962a',
 'def_resumen_sha256','4fb1cd92068acde37fc22c08c5ae4c56e9f6a13d6c3aa3fa90bc4fc60b741c13',
 'owner_orden','vec_bolsa_llamamientos_propietario','owner_resumen','vec_bolsa_llamamientos_propietario',
 'acl_orden','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}',
 'acl_resumen','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}');
BEGIN
 SELECT jsonb_build_object(
 'rol',current_user='vec_bolsa_llamamientos_propietario',
 'pendiente_b90',to_regclass('vec_bolsa_llamamientos.candidatos_cese_pendiente_b90') IS NOT NULL,
 'vinculo',to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NOT NULL,
 'restriccion',to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NOT NULL,
 'contrato',to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NOT NULL,
 'situacion',to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NOT NULL,
 'orden_b6',to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NOT NULL,
 'resumen_b82',to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)') IS NOT NULL,
 'lote_libre',to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)') IS NULL
    AND to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)') IS NULL,
 'def_orden_sha256',encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)')),'UTF8')),'hex'),
 'def_resumen_sha256',encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)')),'UTF8')),'hex'),
 'owner_orden',(SELECT proowner::regrole::text FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)')),
 'owner_resumen',(SELECT proowner::regrole::text FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)')),
 'acl_orden',(SELECT proacl::text FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)')),
 'acl_resumen',(SELECT proacl::text FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)')))
 INTO actual;
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'B90: clave=preimagen_lectores esperado=% actual=%',esperado,actual USING ERRCODE='55000';
 END IF;
END $pre$;

-- Núcleo privado de conjunto. B6/B82 y la fachada nominal ejecutan este
-- cálculo como propietario; ningún LOGIN obtiene EXECUTE directo.
CREATE FUNCTION vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(
 p_participaciones text[],p_corte timestamptz)
RETURNS TABLE(participacion_ref text,fecha_efecto date,disponible_desde date,
 en_restriccion boolean,trabajo_cesado boolean,cese_pendiente boolean,
 pendiente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY INVOKER ROWS 20000
SET search_path=pg_catalog,pg_temp
SET statement_timeout='5s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_corte IS NULL OR NOT isfinite(p_corte)
    OR p_participaciones IS NULL OR cardinality(p_participaciones) NOT BETWEEN 0 AND 20000
    OR EXISTS (SELECT 1 FROM unnest(p_participaciones) x
                WHERE x IS NULL OR octet_length(x) NOT BETWEEN 1 AND 512) THEN
  RAISE EXCEPTION 'B90: consulta de ceses en lote denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 WITH refs AS MATERIALIZED (
  SELECT DISTINCT x.ref FROM unnest(p_participaciones) AS x(ref)
 ), titulares AS MATERIALIZED (
  SELECT i.ref,vc.candidato_ref FROM refs i
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=i.ref
 ), candidatos AS MATERIALIZED (
  SELECT DISTINCT candidato_ref FROM titulares
 ), pendientes AS MATERIALIZED (
  SELECT coalesce(jsonb_object_agg(z.candidato_ref,to_jsonb(z.pendiente_desde)),'{}'::jsonb) AS mapa
  FROM (
   SELECT cp.candidato_ref,min(cp.recibido_en) AS pendiente_desde
    FROM vec_bolsa_llamamientos.candidatos_cese_pendiente_b90 cp
    JOIN candidatos c ON c.candidato_ref=cp.candidato_ref
   WHERE cp.ocurrido_en<=p_corte AND cp.origen_creada_en<=p_corte
     AND cp.recibido_en<=p_corte
     AND (cp.restriccion_recibida_en IS NULL OR cp.restriccion_recibida_en>p_corte)
     AND (cp.ajeno_recibido_en IS NULL OR cp.ajeno_recibido_en>p_corte)
   GROUP BY cp.candidato_ref
  ) z
 ), restricciones AS MATERIALIZED (
  SELECT r.candidato_ref,r.fecha_efecto,r.disponible_desde,r.llamamiento_ref,r.evento_ref
   FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
   JOIN candidatos c ON c.candidato_ref=r.candidato_ref
  WHERE r.fecha_efecto<=(p_corte AT TIME ZONE 'Europe/Madrid')::date
 ), ultimo AS MATERIALIZED (
  SELECT DISTINCT ON (r.candidato_ref) r.candidato_ref,r.fecha_efecto,r.llamamiento_ref
   FROM restricciones r ORDER BY r.candidato_ref,r.fecha_efecto DESC,r.evento_ref DESC
 ), maximo AS MATERIALIZED (
  SELECT r.candidato_ref,max(r.disponible_desde) AS disponible_desde
   FROM restricciones r GROUP BY r.candidato_ref
 ), ultimo_estado AS MATERIALIZED (
  SELECT DISTINCT ON (sp.participacion_ref) sp.participacion_ref,sp.situacion,sp.desde
   FROM vec_bolsa_llamamientos.situacion_participacion sp
   JOIN refs i ON i.ref=sp.participacion_ref
  WHERE sp.desde<=p_corte ORDER BY sp.participacion_ref,sp.desde DESC
 ), incorporaciones AS MATERIALIZED (
  SELECT vc.candidato_ref,cp.llamamiento_ref,coalesce(cp.inicio,cp.ocurrido_en) AS inicio
   FROM candidatos c
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.candidato_ref=c.candidato_ref
   JOIN vec_bolsa_llamamientos.contrato_participacion cp ON cp.participacion_ref=vc.participacion_ref
  WHERE cp.tipo='incorporacion' AND coalesce(cp.inicio,cp.ocurrido_en)<=p_corte
 ), cierre_relacion AS MATERIALIZED (
  SELECT r.candidato_ref,r.llamamiento_ref,max(r.fecha_efecto) AS fecha_efecto
   FROM restricciones r GROUP BY r.candidato_ref,r.llamamiento_ref
 ), prueba AS MATERIALIZED (
  SELECT u.candidato_ref,
   coalesce(bool_or(i.llamamiento_ref=u.llamamiento_ref AND
    i.inicio<((u.fecha_efecto+1)::timestamp AT TIME ZONE 'Europe/Madrid')),false) AS misma_relacion,
   coalesce(bool_or(i.inicio IS NOT NULL AND (cr.fecha_efecto IS NULL OR
    i.inicio>=((cr.fecha_efecto+1)::timestamp AT TIME ZONE 'Europe/Madrid'))),false) AS otra_abierta
  FROM ultimo u LEFT JOIN incorporaciones i ON i.candidato_ref=u.candidato_ref
   LEFT JOIN cierre_relacion cr ON cr.candidato_ref=i.candidato_ref
    AND cr.llamamiento_ref=i.llamamiento_ref
  GROUP BY u.candidato_ref
 )
 SELECT t.ref,u.fecha_efecto,m.disponible_desde,
  coalesce(m.disponible_desde>(p_corte AT TIME ZONE 'Europe/Madrid')::date AND
   (s.situacion IS DISTINCT FROM 'trabajando' OR
    (s.situacion='trabajando' AND pr.misma_relacion AND NOT pr.otra_abierta
     AND s.desde<((u.fecha_efecto+1)::timestamp AT TIME ZONE 'Europe/Madrid'))),false),
  coalesce(s.situacion='trabajando' AND pr.misma_relacion AND NOT pr.otra_abierta
   AND s.desde<((u.fecha_efecto+1)::timestamp AT TIME ZONE 'Europe/Madrid'),false),
  (p.mapa ? t.candidato_ref),(p.mapa->>t.candidato_ref)::timestamptz
 FROM titulares t CROSS JOIN pendientes p
  LEFT JOIN ultimo u ON u.candidato_ref=t.candidato_ref
  LEFT JOIN maximo m ON m.candidato_ref=t.candidato_ref
  LEFT JOIN ultimo_estado s ON s.participacion_ref=t.ref
  LEFT JOIN prueba pr ON pr.candidato_ref=t.candidato_ref
 WHERE u.candidato_ref IS NOT NULL OR p.mapa ? t.candidato_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz) FROM PUBLIC;

-- Única fachada de consumo del lote: la sesión de RRHH ya comprobada por Go
-- conserva el rol técnico exacto. Las funciones internas B6/B82 usan el
-- núcleo privado y mantienen su propia ACL histórica.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(
 p_participaciones text[],p_corte timestamptz)
RETURNS TABLE(participacion_ref text,fecha_efecto date,disponible_desde date,
 en_restriccion boolean,trabajo_cesado boolean,cese_pendiente boolean,
 pendiente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER ROWS 20000
SET search_path=pg_catalog,pg_temp SET statement_timeout='5s' AS $fachada$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'B90: consulta nominal de ceses en lote denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.participacion_ref,x.fecha_efecto,x.disponible_desde,
  x.en_restriccion,x.trabajo_cesado,x.cese_pendiente,x.pendiente_desde
 FROM vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(p_participaciones,p_corte) x;
END $fachada$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)
 TO vec_bolsa_llamamientos_ejecutor;

-- Reconstrucciones literales de las definiciones instaladas. B90 solo cambia
-- el cálculo de cese; firma, OID y concesiones se conservan.
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
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         coalesce(rc.en_restriccion,false) AS cese_restringido,
         coalesce(rc.cese_pendiente,false) AS cese_pendiente,
         coalesce(rc.trabajo_cesado,false) AS trabajo_cesado,
         ((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false)) AND NOT coalesce(rc.en_restriccion,false)
          AND NOT coalesce(rc.cese_pendiente,false)) AS ocupa_turno,
         r.aplicada_en AS repuesta_en,
         EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion pe
                  WHERE pe.bolsa_ref=c.bolsa_ref AND pe.participacion_ref=e.participacion_ref AND pe.aplicada_en<=p_en
                    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                     WHERE rv.sancion_ref=pe.sancion_ref AND rv.registrada_en<=p_en)) AS penalizada
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN ceses rc ON rc.participacion_ref=e.participacion_ref
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
        CASE WHEN b.cese_pendiente THEN 'no_disponible'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'disponible_desde' WHEN b.trabajo_cesado THEN 'disponible'
             ELSE b.situacion END AS situacion,
        CASE WHEN b.cese_pendiente THEN 'cese_pendiente'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde') THEN 'restriccion_cese'
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

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(p_corte timestamp with time zone)
 RETURNS TABLE(bolsa_ref text, categoria_ref text, confirmada_en timestamp with time zone, instantanea_ref text, version_instantanea bigint, orden bigint, participacion_ref text, situacion text, desde timestamp with time zone, fecha_disponible timestamp with time zone, cese_fecha_efecto date, cese_disponible_desde date, cese_en_restriccion boolean, cese_trabajo_cesado boolean)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER ROWS 10000
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_corte IS NULL OR NOT isfinite(p_corte) THEN
  RAISE EXCEPTION 'lectura del resumen de bolsas denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 WITH k AS MATERIALIZED (
   SELECT DISTINCT c.bolsa_ref,c.categoria_ref,c.confirmada_en,c.instantanea_ref,c.version_instantanea
     FROM vec_bolsa_llamamientos.listar_constituciones_v1() c
 ), entradas AS MATERIALIZED (
   SELECT e.* FROM k
   JOIN vec_bolsa_llamamientos.constitucion_entrada e
     ON e.instantanea_ref=k.instantanea_ref AND e.version_instantanea=k.version_instantanea
 ), ceses AS MATERIALIZED (
   SELECT ec.* FROM vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(
    coalesce((SELECT array_agg(e.participacion_ref) FROM entradas e),ARRAY[]::text[]),p_corte) ec
 )
 SELECT k.bolsa_ref,k.categoria_ref,k.confirmada_en,
        k.instantanea_ref,k.version_instantanea,e.orden,e.participacion_ref,
        s.situacion,CASE WHEN x.cese_pendiente THEN x.pendiente_desde ELSE s.desde END,s.fecha_disponible,
        CASE WHEN x.cese_pendiente THEN NULL ELSE x.fecha_efecto END,
        CASE WHEN x.cese_pendiente THEN NULL ELSE x.disponible_desde END,
        CASE WHEN x.cese_pendiente THEN true ELSE x.en_restriccion END,
        CASE WHEN x.cese_pendiente THEN false ELSE x.trabajo_cesado END
   FROM k
   LEFT JOIN entradas e
     ON e.instantanea_ref=k.instantanea_ref AND e.version_instantanea=k.version_instantanea
   LEFT JOIN LATERAL (
     SELECT sp.situacion,sp.desde,sp.fecha_disponible
       FROM vec_bolsa_llamamientos.situacion_participacion sp
      WHERE sp.participacion_ref=e.participacion_ref
      ORDER BY sp.desde DESC LIMIT 1) s ON true
   LEFT JOIN ceses x ON x.participacion_ref=e.participacion_ref
  ORDER BY k.categoria_ref,e.orden;
END $function$;

DO $post$
DECLARE actual jsonb; esperado jsonb:=jsonb_build_object(
 'owner_lote','vec_bolsa_llamamientos_propietario',
 'owner_nucleo','vec_bolsa_llamamientos_propietario',
 'nucleo_invocador',true,'nucleo_ejecutor',false,
 'owner_orden','vec_bolsa_llamamientos_propietario',
 'owner_resumen','vec_bolsa_llamamientos_propietario',
 'definidoras',true,'busqueda_cerrada',true,'lote_ejecutor',true,
 'lote_publico',false,'lote_relevo',false,'orden_ejecutor',true,
 'resumen_ejecutor',true,'orden_publico',false,'resumen_publico',false,
 'vista_ejecutor',false);
BEGIN
 SELECT jsonb_build_object(
 'owner_lote',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)'::regprocedure),
 'owner_nucleo',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)'::regprocedure),
 'nucleo_invocador',(SELECT NOT prosecdef AND 'search_path=pg_catalog, pg_temp'=ANY(proconfig)
    FROM pg_proc WHERE oid='vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)'::regprocedure),
 'nucleo_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor',
    'vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(text[],timestamptz)','EXECUTE'),
 'owner_orden',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure),
 'owner_resumen',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::regprocedure),
 'definidoras',(SELECT bool_and(prosecdef) FROM pg_proc WHERE oid IN (
  'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::regprocedure)),
 'busqueda_cerrada',(SELECT bool_and('search_path=pg_catalog, pg_temp'=ANY(proconfig)) FROM pg_proc WHERE oid IN (
  'vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::regprocedure)),
 'lote_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)','EXECUTE'),
 'lote_publico',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)'::regprocedure AND a.grantee=0),
 'lote_relevo',has_function_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)','EXECUTE'),
 'orden_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)','EXECUTE'),
 'resumen_ejecutor',has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)','EXECUTE'),
 'orden_publico',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure AND a.grantee=0),
 'resumen_publico',EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::regprocedure AND a.grantee=0),
 'vista_ejecutor',has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.candidatos_cese_pendiente_b90','SELECT'))
 INTO actual;
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'B90: clave=postimagen_lectores esperado=% actual=%',esperado,actual USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
