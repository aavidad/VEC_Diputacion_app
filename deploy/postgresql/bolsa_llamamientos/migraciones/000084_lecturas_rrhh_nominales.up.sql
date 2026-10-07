\set ON_ERROR_STOP on
-- B84: consumo nominal único y lectura de conjunto en la misma transacción.
-- La página de candidatos se añade a esta migración antes de su ensayo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000084',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(text,timestamptz)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B84: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El primer barrido de texto necesita dos lecturas SQL dentro de una TX:
-- metadatos/filas cifradas y, tras aplicar Contains en Go, contactos de su
-- página. Esta marca sólo puede vivir dentro de esa TX. El disparador
-- diferido rechaza cualquier commit que olvidara consumirla.
CREATE TABLE vec_bolsa_llamamientos.barrido_rrhh_tx (
 consumo_huella_sha256 text PRIMARY KEY CHECK (consumo_huella_sha256 ~ '^[a-f0-9]{64}$'),
 bolsa_ref text NOT NULL,
 actor_ref text NOT NULL,
 origen_login text NOT NULL,
 snapshot_sha256 text NOT NULL CHECK (snapshot_sha256 ~ '^[a-f0-9]{64}$'),
 corte timestamptz NOT NULL,
 transaccion xid8 NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.barrido_rrhh_tx ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.barrido_rrhh_tx FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.barrido_rrhh_tx
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.barrido_rrhh_tx FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.exigir_barrido_rrhh_tx_finalizado_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.barrido_rrhh_tx b
             WHERE b.consumo_huella_sha256=NEW.consumo_huella_sha256) THEN
   RAISE EXCEPTION 'B84: barrido sin finalizar' USING ERRCODE='55000';
 END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_barrido_rrhh_tx_finalizado_v1() FROM PUBLIC;
CREATE CONSTRAINT TRIGGER barrido_rrhh_finalizado
 AFTER INSERT ON vec_bolsa_llamamientos.barrido_rrhh_tx
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.exigir_barrido_rrhh_tx_finalizado_v1();

CREATE FUNCTION vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(
 p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE
 v_consumo record;
 v_corte timestamptz;
 v_situaciones jsonb;
 v_politicas jsonb;
 v_emisiones jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR p_accion NOT IN ('bolsa.rrhh.bolsas.consultar','bolsa.rrhh.estadisticas.consultar')
 THEN RAISE EXCEPTION 'B84: lectura denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(
   p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR
    (p_accion='bolsa.rrhh.bolsas.consultar' AND v_consumo.efecto_ref<>'coleccion:bolsa:rrhh:bolsas') OR
    (p_accion='bolsa.rrhh.estadisticas.consultar' AND v_consumo.efecto_ref<>'coleccion:bolsa:rrhh:estadisticas')
 THEN RAISE EXCEPTION 'B84: consumo incompatible' USING ERRCODE='42501'; END IF;
 v_corte:=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
 WITH s AS MATERIALIZED (
   SELECT * FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(v_corte)
 ), b AS MATERIALIZED (SELECT DISTINCT s.bolsa_ref FROM s)
 SELECT (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY x.categoria_ref,x.orden),'[]'::jsonb) FROM s x),
        (SELECT coalesce(pg_catalog.jsonb_object_agg(x.bolsa_ref,
          vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(x.bolsa_ref)),'{}'::jsonb) FROM b x)
 INTO v_situaciones,v_emisiones;
 IF pg_catalog.jsonb_path_exists(v_situaciones,'$[*] ? (@.participacion_ref == null)') THEN
   RAISE EXCEPTION 'B84: constitucion sin participaciones' USING ERRCODE='55000';
 END IF;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(p) ORDER BY p.bolsa_ref),'[]'::jsonb)
 INTO v_politicas FROM vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(v_corte) p;
 RETURN pg_catalog.jsonb_build_object(
   'generado_en',v_corte,'situaciones',v_situaciones,'politicas',v_politicas,
   'llamamientos_en_curso',v_emisiones,'historico_llamamientos_disponible',false);
END $f$;

-- Una consulta de candidatos consume su propia acción AD213. El modo
-- barrido_texto devuelve únicamente filas de la bolsa/estado prefiltrados;
-- el predicado sobre nombre/documento cifrados se aplica en Go y sólo para
-- esa primera búsqueda. pagina_texto recibe las referencias de una selección
-- opaca, breve y ligada al actor que conserva el proceso servidor.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1(
 p_bolsa_ref text,p_estado text,p_texto text,p_cursor_token text,p_cursor_ref text,
 p_snapshot_sha256 text,p_limite integer,p_modo text,p_ids text[],p_cese_activo boolean,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE
 v_consumo record;
 v_decision jsonb;
 v_corte timestamptz;
 v_esperado text;
 v_bolsa record;
 v_politica record;
 v_snapshot text;
 v_cursor_valido boolean;
 v_total integer;
 v_estado_conteos jsonb;
 v_filtrado integer;
 v_pagina jsonb;
 v_refs text[];
 v_numeros integer[];
 v_tiene_mas boolean;
 v_siguiente text;
 v_turno_siguiente jsonb;
 v_turno_ultimo jsonb;
 v_contactos jsonb;
 v_marcas jsonb;
 v_protegidas jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR p_bolsa_ref IS NULL OR p_bolsa_ref !~ '^bolsa:[A-Za-z0-9:_-]+$'
    OR p_estado IS NULL OR (p_estado<>'' AND p_estado NOT IN
      ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision'))
    OR p_texto IS NULL OR pg_catalog.octet_length(p_texto)>300
    OR p_cursor_token IS NULL OR pg_catalog.octet_length(p_cursor_token)>256
    OR p_cursor_ref IS NULL OR pg_catalog.octet_length(p_cursor_ref)>512
    OR p_snapshot_sha256 IS NULL OR (p_snapshot_sha256<>'' AND p_snapshot_sha256 !~ '^[a-f0-9]{64}$')
    OR p_limite NOT BETWEEN 1 AND 100 OR p_modo NOT IN ('pagina','barrido_texto','pagina_texto')
    OR p_cese_activo IS NULL
    OR (p_modo='pagina' AND (p_texto<>'' OR p_ids IS NOT NULL))
    OR (p_modo IN ('barrido_texto','pagina_texto') AND p_texto='')
    OR (p_modo='barrido_texto' AND (p_cursor_token<>'' OR p_cursor_ref<>'' OR p_snapshot_sha256<>'' OR p_ids IS NOT NULL))
    OR (p_modo='pagina_texto' AND (p_ids IS NULL OR pg_catalog.cardinality(p_ids) NOT BETWEEN 1 AND 100 OR p_snapshot_sha256=''))
    OR (p_modo='pagina' AND (p_cursor_token='')<>(p_cursor_ref=''))
    OR (p_cursor_token<>'' AND p_snapshot_sha256='')
    OR (p_modo='pagina' AND p_cursor_token<>'' AND p_cursor_token IS DISTINCT FROM
        'sql.'||pg_catalog.encode(pg_catalog.convert_to(p_cursor_ref,'UTF8'),'hex')||'.'||p_snapshot_sha256)
    OR (p_modo='pagina_texto' AND p_cursor_token IS DISTINCT FROM
        'txt.'||pg_catalog.split_part(p_cursor_token,'.',2)||'.'||pg_catalog.split_part(p_cursor_token,'.',3)||'.'||p_snapshot_sha256||'.'||
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.array_to_string(p_ids,pg_catalog.chr(31)),'UTF8')),'hex'))
 THEN RAISE EXCEPTION 'B84: parámetros de candidatos inválidos' USING ERRCODE='22023'; END IF;
 v_esperado:=p_bolsa_ref||':filtro:'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   p_estado||pg_catalog.chr(31)||p_texto||pg_catalog.chr(31)||p_cursor_token||pg_catalog.chr(31)||p_limite::text,'UTF8')),'hex');
 BEGIN v_decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B84: decisión inválida' USING ERRCODE='22023'; END;
 IF v_decision->>'recurso_ref' IS DISTINCT FROM v_esperado
 THEN RAISE EXCEPTION 'B84: recurso de filtro ajeno' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(
  'bolsa.rrhh.candidatos.consultar',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_esperado
 THEN RAISE EXCEPTION 'B84: consumo de candidatos incompatible' USING ERRCODE='42501'; END IF;
 v_corte:=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
 SELECT c.categoria_ref,c.confirmada_en,c.instantanea_ref,c.version_instantanea,b.version AS version_bolsa,
        b.estado,b.vigente_desde,b.vigente_hasta,
        pg_catalog.convert_from(b.bolsa_canonica,'UTF8')::jsonb->>'huella_listado_sha256' AS huella_listado
 INTO v_bolsa
 FROM vec_bolsa_llamamientos.listar_constituciones_v1() l
 JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=l.acta_ref
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 WHERE l.bolsa_ref=p_bolsa_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'B84: bolsa no encontrada' USING ERRCODE='VBR04'; END IF;
 IF v_bolsa.huella_listado IS NULL OR v_bolsa.huella_listado !~ '^[a-f0-9]{64}$'
 THEN RAISE EXCEPTION 'B84: acta de bolsa incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(v_corte) p WHERE p.bolsa_ref=p_bolsa_ref;

 WITH s AS MATERIALIZED (
   SELECT * FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(v_corte) x WHERE x.bolsa_ref=p_bolsa_ref
 ), o AS MATERIALIZED (
   SELECT * FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa_ref,v_corte)
 ), base AS MATERIALIZED (
   SELECT s.participacion_ref,e.fila_numero,s.orden AS orden_acta,o.orden_vigente,o.razon,
          s.situacion AS base_estado,s.desde AS base_desde,s.fecha_disponible AS base_disponible,
          s.cese_fecha_efecto,s.cese_disponible_desde,s.cese_en_restriccion,s.cese_trabajo_cesado,
          CASE
           WHEN p_cese_activo AND s.cese_en_restriccion IS TRUE AND s.situacion IN ('disponible','trabajando','disponible_desde') THEN 'disponible_desde'
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='trabajando' AND s.cese_trabajo_cesado IS TRUE
                AND (s.fecha_disponible IS NULL OR s.fecha_disponible<=v_corte) THEN 'disponible'
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='trabajando' AND s.cese_trabajo_cesado IS TRUE THEN 'disponible_desde'
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='disponible_desde' AND s.fecha_disponible<=v_corte THEN 'disponible'
           ELSE s.situacion END AS estado,
          CASE
           WHEN p_cese_activo AND s.cese_en_restriccion IS TRUE AND s.situacion IN ('disponible','trabajando','disponible_desde')
                AND (s.fecha_disponible IS NULL OR s.fecha_disponible<=s.cese_disponible_desde::timestamp AT TIME ZONE 'UTC')
             THEN s.cese_fecha_efecto::timestamp AT TIME ZONE 'UTC'
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='trabajando' AND s.cese_trabajo_cesado IS TRUE
                AND (s.fecha_disponible IS NULL OR s.fecha_disponible<=v_corte)
             THEN s.cese_fecha_efecto::timestamp AT TIME ZONE 'UTC'
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='disponible_desde' AND s.fecha_disponible<=v_corte THEN s.fecha_disponible
           ELSE s.desde END AS estado_desde,
          CASE
           WHEN p_cese_activo AND s.cese_en_restriccion IS TRUE AND s.situacion IN ('disponible','trabajando','disponible_desde')
             THEN greatest(s.cese_disponible_desde::timestamp AT TIME ZONE 'UTC',s.fecha_disponible)
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='trabajando' AND s.cese_trabajo_cesado IS TRUE
                AND (s.fecha_disponible IS NULL OR s.fecha_disponible<=v_corte) THEN NULL
           WHEN p_cese_activo AND s.cese_fecha_efecto IS NOT NULL AND s.cese_en_restriccion IS FALSE
                AND s.situacion='disponible_desde' AND s.fecha_disponible<=v_corte THEN NULL
           ELSE s.fecha_disponible END AS disponible_desde
   FROM s JOIN vec_bolsa_llamamientos.constitucion_entrada e
     ON e.instantanea_ref=s.instantanea_ref AND e.version_instantanea=s.version_instantanea AND e.participacion_ref=s.participacion_ref
   JOIN o ON o.participacion_ref=s.participacion_ref
   WHERE s.instantanea_ref=v_bolsa.instantanea_ref AND s.version_instantanea=v_bolsa.version_instantanea
 ), ordenados AS MATERIALIZED (
   SELECT b.*,pg_catalog.row_number() OVER (ORDER BY b.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref) AS posicion
   FROM base b
 ), prefiltro AS MATERIALIZED (
   SELECT * FROM ordenados x WHERE p_estado='' OR x.estado=p_estado
 ), cursor AS (
   SELECT x.posicion FROM prefiltro x WHERE x.participacion_ref=p_cursor_ref
 ), elegidos AS MATERIALIZED (
   SELECT x.* FROM prefiltro x
   WHERE (p_modo='barrido_texto')
      OR (p_modo='pagina' AND (p_cursor_ref='' OR x.posicion>(SELECT c.posicion FROM cursor c)))
      OR (p_modo='pagina_texto' AND x.participacion_ref=ANY(p_ids))
   ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,x.participacion_ref) ELSE x.posicion::integer END
   LIMIT CASE WHEN p_modo='barrido_texto' THEN 5001 ELSE p_limite+1 END
 ), pagina AS MATERIALIZED (
   SELECT * FROM elegidos x ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,x.participacion_ref) ELSE x.posicion::integer END LIMIT p_limite
 ), siguiente AS (
   SELECT x.participacion_ref,x.fila_numero,x.orden_vigente FROM ordenados x WHERE x.orden_vigente IS NOT NULL
   ORDER BY x.posicion LIMIT 1
 ), ultimo AS (
   SELECT c.participacion_ref,o.fila_numero,o.orden_vigente,c.contacto_ref,c.llamamiento_ref,c.instante,c.canal,c.resultado
   FROM vec_bolsa_llamamientos.contacto_participacion c
   JOIN ordenados o ON o.participacion_ref=c.participacion_ref
   WHERE c.bolsa_ref=p_bolsa_ref AND c.llamamiento_ref IS NOT NULL AND c.resultado<>'no_enviado' AND c.registrada_en<=v_corte
   ORDER BY c.instante DESC,c.contacto_ref DESC LIMIT 1
 ), numeros AS (
   SELECT x.fila_numero FROM (
      SELECT fila_numero FROM pagina WHERE p_modo<>'barrido_texto'
      UNION SELECT fila_numero FROM elegidos WHERE p_modo='barrido_texto'
      UNION SELECT fila_numero FROM siguiente
      UNION SELECT fila_numero FROM ultimo) x
 )
 SELECT
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    v_bolsa.instantanea_ref||':'||v_bolsa.version_instantanea::text||':'||v_bolsa.version_bolsa::text||':'||v_politica.version_politica::text||'|'
      ||coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY x.posicion)::text FROM ordenados x),'[]')
      ||'|'||coalesce((SELECT pg_catalog.count(*)::text||':'||pg_catalog.max(c.registrada_en)::text FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.bolsa_ref=p_bolsa_ref AND c.registrada_en<=v_corte),'0')
      ||'|'||coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(m) ORDER BY m.participacion_ref)::text
          FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(p_bolsa_ref,v_corte) m),'[]'),
    'UTF8')),'hex'),
  (SELECT pg_catalog.count(*)::integer FROM ordenados),
  (SELECT coalesce(pg_catalog.jsonb_object_agg(z.estado,z.total),'{}'::jsonb) FROM
      (SELECT estado,pg_catalog.count(*)::integer AS total FROM ordenados GROUP BY estado) z),
  (SELECT pg_catalog.count(*)::integer FROM prefiltro),
  (p_cursor_ref='' OR EXISTS (SELECT 1 FROM cursor)),
  (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
      'participacion_ref',p.participacion_ref,'fila_numero',p.fila_numero,
      'orden',p.orden_vigente,'orden_acta',p.orden_acta,'razon_orden',p.razon,
      'estado_clave',p.estado,'estado_desde',p.estado_desde,'disponible_desde',p.disponible_desde)
      ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,p.participacion_ref) ELSE p.posicion::integer END),'[]'::jsonb)
     FROM (SELECT * FROM elegidos ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,participacion_ref) ELSE posicion::integer END
       LIMIT CASE WHEN p_modo='barrido_texto' THEN 5001 ELSE p_limite END) p),
  (SELECT pg_catalog.array_agg(p.participacion_ref ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,p.participacion_ref) ELSE p.posicion::integer END) FROM pagina p),
  (SELECT pg_catalog.array_agg(n.fila_numero ORDER BY n.fila_numero) FROM numeros n),
  (SELECT pg_catalog.count(*)>p_limite FROM elegidos),
  (SELECT p.participacion_ref FROM pagina p ORDER BY CASE WHEN p_modo='pagina_texto' THEN pg_catalog.array_position(p_ids,p.participacion_ref) ELSE p.posicion::integer END DESC LIMIT 1),
  (SELECT pg_catalog.to_jsonb(x) FROM siguiente x),
  (SELECT pg_catalog.to_jsonb(x) FROM ultimo x)
 INTO v_snapshot,v_total,v_estado_conteos,v_filtrado,v_cursor_valido,v_pagina,v_refs,v_numeros,v_tiene_mas,v_siguiente,v_turno_siguiente,v_turno_ultimo;

 IF v_total IS NULL OR v_filtrado IS NULL OR v_snapshot !~ '^[a-f0-9]{64}$'
    OR (p_modo='barrido_texto' AND v_filtrado>5000)
 THEN RAISE EXCEPTION 'B84: conjunto de candidatos incompatible' USING ERRCODE='55000'; END IF;
 IF (p_modo='pagina' AND NOT v_cursor_valido)
    OR (p_snapshot_sha256<>'' AND p_snapshot_sha256<>v_snapshot)
    OR (p_modo='pagina_texto' AND pg_catalog.cardinality(p_ids)<>pg_catalog.cardinality(v_refs))
 THEN RAISE EXCEPTION 'B84: cursor o instantánea caducada' USING ERRCODE='VBR09'; END IF;
 IF v_numeros IS NOT NULL THEN
   v_protegidas:=vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(v_bolsa.huella_listado,v_bolsa.categoria_ref,v_numeros);
 END IF;
 IF p_modo<>'barrido_texto' THEN
   IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.contacto_participacion c
        WHERE c.bolsa_ref=p_bolsa_ref AND c.participacion_ref=ANY(v_refs) AND c.registrada_en<=v_corte)>20000
   THEN RAISE EXCEPTION 'B84: contactos excesivos' USING ERRCODE='55000'; END IF;
   SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
      'contacto_ref',c.contacto_ref,'participacion_ref',c.participacion_ref,
      'llamamiento_ref',c.llamamiento_ref,'canal',c.canal,'instante',c.instante,
      'oferta_ref',c.oferta_ref,'evidencia_ref',c.evidencia_ref,'evidencia_huella_sha256',c.evidencia_huella,
      'actor_ref',c.actor,'resultado',c.resultado,'anotacion',c.anotacion)
      ORDER BY c.instante DESC,c.contacto_ref DESC),'[]'::jsonb)
   INTO v_contactos FROM vec_bolsa_llamamientos.contacto_participacion c
   WHERE c.bolsa_ref=p_bolsa_ref AND c.participacion_ref=ANY(v_refs) AND c.registrada_en<=v_corte;
   SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(m) ORDER BY m.participacion_ref),'[]'::jsonb)
   INTO v_marcas FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(p_bolsa_ref,v_corte) m
   WHERE m.participacion_ref=ANY(v_refs);
 ELSE
   v_contactos:='[]'::jsonb;
   v_marcas:='[]'::jsonb;
   INSERT INTO vec_bolsa_llamamientos.barrido_rrhh_tx(
      consumo_huella_sha256,bolsa_ref,actor_ref,origen_login,snapshot_sha256,corte,transaccion)
   VALUES(v_consumo.consumo_huella_sha256,p_bolsa_ref,v_decision->>'principal_id',session_user,
      v_snapshot,v_corte,pg_catalog.pg_current_xact_id());
 END IF;
 RETURN pg_catalog.jsonb_build_object(
   'generado_en',v_corte,'bolsa_ref',p_bolsa_ref,'categoria_ref',v_bolsa.categoria_ref,
   'huella_listado_sha256',v_bolsa.huella_listado,
   'confirmada_en',v_bolsa.confirmada_en,'tipo_lista',v_politica.tipo_lista,
   'politica',pg_catalog.to_jsonb(v_politica),'estado_bolsa',v_bolsa.estado,
   'vigente_desde',v_bolsa.vigente_desde,'vigente_hasta',v_bolsa.vigente_hasta,
   'llamamientos_en_curso',vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(p_bolsa_ref),
   'snapshot_sha256',v_snapshot,'total',v_total,'por_estado',v_estado_conteos,
   'total_prefiltrado',v_filtrado,'candidatos',v_pagina,'contactos',v_contactos,'marcas',v_marcas,
   'turno_siguiente',v_turno_siguiente,'turno_ultimo',v_turno_ultimo,
   'filas_protegidas',v_protegidas,'hay_mas',v_tiene_mas,'cursor_ref_siguiente',v_siguiente,
   'consumo_huella_sha256',v_consumo.consumo_huella_sha256);
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1(
 p_consumo_huella_sha256 text,p_bolsa_ref text,p_snapshot_sha256 text,p_ids text[])
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_barrido vec_bolsa_llamamientos.barrido_rrhh_tx%ROWTYPE;
 v_contactos jsonb; v_marcas jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR p_consumo_huella_sha256 !~ '^[a-f0-9]{64}$' OR p_bolsa_ref IS NULL
    OR p_snapshot_sha256 !~ '^[a-f0-9]{64}$' OR p_ids IS NULL
    OR pg_catalog.cardinality(p_ids)>100 OR pg_catalog.array_ndims(p_ids)>1
    OR EXISTS (SELECT 1 FROM pg_catalog.unnest(p_ids) x WHERE x IS NULL OR x='')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.unnest(p_ids) AS x(ref))<>
       (SELECT pg_catalog.count(DISTINCT x.ref) FROM pg_catalog.unnest(p_ids) AS x(ref))
 THEN RAISE EXCEPTION 'B84: cierre de barrido inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT v_barrido FROM vec_bolsa_llamamientos.barrido_rrhh_tx
  WHERE consumo_huella_sha256=p_consumo_huella_sha256 FOR UPDATE;
 IF v_barrido.bolsa_ref IS DISTINCT FROM p_bolsa_ref
    OR v_barrido.snapshot_sha256 IS DISTINCT FROM p_snapshot_sha256
    OR v_barrido.origen_login IS DISTINCT FROM session_user
    OR v_barrido.transaccion IS DISTINCT FROM pg_catalog.pg_current_xact_id()
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.unnest(p_ids) AS x(ref)) <>
       (SELECT pg_catalog.count(*) FROM pg_catalog.unnest(p_ids) AS x(ref)
          JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa_ref
          JOIN vec_bolsa_llamamientos.constitucion_entrada e
            ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea
             AND e.participacion_ref=x.ref)
 THEN RAISE EXCEPTION 'B84: página ajena al barrido' USING ERRCODE='42501'; END IF;
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.contacto_participacion c
      WHERE c.bolsa_ref=p_bolsa_ref AND c.participacion_ref=ANY(p_ids) AND c.registrada_en<=v_barrido.corte)>20000
 THEN RAISE EXCEPTION 'B84: contactos excesivos' USING ERRCODE='55000'; END IF;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
      'contacto_ref',c.contacto_ref,'participacion_ref',c.participacion_ref,
      'llamamiento_ref',c.llamamiento_ref,'canal',c.canal,'instante',c.instante,
      'oferta_ref',c.oferta_ref,'evidencia_ref',c.evidencia_ref,'evidencia_huella_sha256',c.evidencia_huella,
      'actor_ref',c.actor,'resultado',c.resultado,'anotacion',c.anotacion)
      ORDER BY c.instante DESC,c.contacto_ref DESC),'[]'::jsonb)
 INTO v_contactos FROM vec_bolsa_llamamientos.contacto_participacion c
 WHERE c.bolsa_ref=p_bolsa_ref AND c.participacion_ref=ANY(p_ids) AND c.registrada_en<=v_barrido.corte;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(m) ORDER BY m.participacion_ref),'[]'::jsonb)
 INTO v_marcas FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(p_bolsa_ref,v_barrido.corte) m
 WHERE m.participacion_ref=ANY(p_ids);
 DELETE FROM vec_bolsa_llamamientos.barrido_rrhh_tx WHERE consumo_huella_sha256=p_consumo_huella_sha256;
 RETURN pg_catalog.jsonb_build_object('contactos',v_contactos,'marcas',v_marcas);
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1(text,text,text,text,text,text,integer,text,text[],boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1(text,text,text,text,text,text,integer,text,text[],boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1(text,text,text,text[]) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1(text,text,text,text[]) TO vec_bolsa_llamamientos_ejecutor;
-- B82 era una optimización de lectura aún sin actor nominal. Sus funciones
-- quedan internas al propietario y sólo B84 las puede invocar desde la ruta.
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz),
 vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)
 FROM vec_bolsa_llamamientos_ejecutor;
COMMIT;
