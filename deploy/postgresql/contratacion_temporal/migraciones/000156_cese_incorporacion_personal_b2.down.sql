\set ON_ERROR_STOP on
-- CT156: incorporación/cese B2 hacia el histórico B13 y retorno B45.
-- CT no decide disponibilidad ni lee tablas Bolsa/Personal. Bolsa13 debe
-- consumir la incorporación y el cese antes de Bolsa45. No acredita baja
-- Personal, firma, eficacia administrativa ni incorporación real.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000156',0));
-- Solo base desechable sin historia de los dos protocolos. No ejecutar
-- sobre principal ni sobre datos conservados: corregir hacia delante.
LOCK TABLE vec_contratacion_temporal.incorporacion_registro_v2,
 vec_contratacion_temporal.plan_incorporacion_personal_b2_v1,
 vec_contratacion_temporal.origen_incorporacion_personal_b2_v1,
 vec_contratacion_temporal.cese_nombramiento_v1 IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1)
 THEN RAISE EXCEPTION 'CT156: DOWN prohibido con historia' USING ERRCODE='55000'; END IF;
END $pre$;
DO $parches$
DECLARE x record; p record; def text;
BEGIN
 FOR x IN SELECT * FROM (VALUES
('registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)',$old$ IF NOT FOUND THEN RAISE EXCEPTION 'CT75: expediente no disponible' USING ERRCODE='55000'; END IF;
 -- CT156: exclusión recíproca. Ambas fachadas exigen SERIALIZABLE,
 -- bloquean expediente y leen el origen contrario antes de escribir;
 -- SSI rechaza también la carrera con instantánea anterior (40001).
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 b
   WHERE b.organizacion_ref=preparacion->>'OrganizacionRef' AND b.expediente_ref=s->>'expediente_ref')
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 b
   WHERE b.organizacion_ref=preparacion->>'OrganizacionRef' AND b.expediente_ref=s->>'expediente_ref')
 THEN RAISE EXCEPTION 'CT156: expediente con origen B2' USING ERRCODE='55000'; END IF;$old$,$new$ IF NOT FOUND THEN RAISE EXCEPTION 'CT75: expediente no disponible' USING ERRCODE='55000'; END IF;$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$'incorporacion_ref',coalesce(x.incorporacion_ref,x.incorporacion_b2_ref),$old$,$new$'incorporacion_ref',x.incorporacion_ref,$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$       AND ((x.incorporacion_protocolo='ejercicio_v2' AND x.relacion_ref ~ '^ref:[0-9a-f]{64}$')
            OR (x.incorporacion_protocolo='personal_b2_v1' AND x.relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'))$old$,$new$       AND x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}' ~ '^ref:[0-9a-f]{64}$'$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$x.recibo_ref,coalesce(x.incorporacion_ref,x.incorporacion_b2_ref),x.llamamiento_ref,$old$,$new$x.recibo_ref,x.incorporacion_ref,x.llamamiento_ref,$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$           x.relacion_ref,$old$,$new$           x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}',$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$vec_contratacion_temporal.instante_contrato_bolsa_v1(x.inicio_instante)$old$,$new$vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (x.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz)$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) i$old$,$new$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$SELECT c.*, i.inicio_instante, i.relacion_ref,$old$,$new$SELECT c.*, i.material_json,$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$vec_contratacion_temporal.instante_contrato_bolsa_v1(c.inicio_instante)$old$,$new$vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (c.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz)$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) i$old$,$new$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$SELECT c.*, i.inicio_instante, i.relacion_ref,$old$,$new$SELECT c.*, i.material_json,$new$),
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$    ), incorporaciones_b2 AS (
        SELECT i.outbox_ref AS origen,
               vec_contratacion_temporal.posicion_contrato_bolsa_v1(i.transaccion_publicacion) AS posicion,
               i.registrada_en AS creada,
               'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('incorporacion'||chr(31)||i.outbox_ref,'UTF8')),'hex') AS ref,
               jsonb_build_object(
                   'esquema','vec.contratacion-temporal.contrato-bolsa.v1','tipo','incorporacion',
                   'origen_ref',i.outbox_ref,'organizacion_ref',i.organizacion_ref,'expediente_ref',i.expediente_ref,
                   'llamamiento_ref',p.material_json#>>'{material,bolsa,llamamiento_ref}',
                   'inicio',vec_contratacion_temporal.instante_contrato_bolsa_v1((p.material_json#>>'{material,desde}')::date::timestamp AT TIME ZONE 'UTC'),
                   'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(nullif(p.material_json#>>'{material,hasta}','')::date::timestamp AT TIME ZONE 'UTC'),
                   'modalidad_clave',e.agregado_json#>>'{analisis,modalidad_clave}',
                   'categoria_ref',e.agregado_json#>>'{analisis,categoria_ref}',
                   'causa_clave',e.agregado_json#>>'{analisis,causa_clave}',
                   'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(i.registrada_en)) AS cuerpo
        FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
        JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p
          ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
        JOIN vec_contratacion_temporal.expediente_version_integral e
          ON e.expediente_ref=p.expediente_ref AND e.version=p.version_expediente
        JOIN vec_contratacion_temporal.outbox_expediente_integral o
          ON o.evento_ref=i.outbox_ref AND o.expediente_ref=i.expediente_ref AND o.version_expediente=p.version_expediente
         AND o.tipo_evento='ct.incorporacion-personal.v1'
        WHERE p.material_json#>>'{material,bolsa,llamamiento_ref}' IS NOT NULL
          AND (coalesce(i.transaccion_publicacion,'0'::xid8)<pg_snapshot_xmin(pg_current_snapshot())
               OR i.transaccion_publicacion=pg_current_xact_id_if_assigned())
          AND (p_desde_posicion IS NULL
               OR (vec_contratacion_temporal.posicion_contrato_bolsa_v1(i.transaccion_publicacion),i.outbox_ref)>(p_desde_posicion,p_desde_ref))
        ORDER BY 2,1 LIMIT p_limite
    ), base AS (
        SELECT * FROM incorporaciones UNION ALL SELECT * FROM ceses UNION ALL SELECT * FROM incorporaciones_b2$old$,$new$    ), base AS (
        SELECT * FROM incorporaciones UNION ALL SELECT * FROM ceses$new$),
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$                   'llamamiento_ref', c.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(r.inicio_instante),$old$,$new$                   'llamamiento_ref', c.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),$new$),
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) r$old$,$new$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r ON r.recibo_ref = c.incorporacion_ref$new$)
 ) v(firma,anterior,nuevo) LOOP
 SELECT pg_get_functiondef(q.oid) AS def,to_jsonb(q)-'prosrc' AS meta INTO STRICT p
 FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma) AND q.proowner=current_user::regrole;
 IF length(p.def)-length(replace(p.def,x.anterior,''))<>length(x.anterior)
 THEN RAISE EXCEPTION 'CT156: preimagen incompatible: %',x.firma USING ERRCODE='55000'; END IF;
 def:=replace(p.def,x.anterior,x.nuevo); EXECUTE def;
 IF (SELECT pg_get_functiondef(q.oid) FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM def
 OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM p.meta
 THEN RAISE EXCEPTION 'CT156: firma/OID/ACL/metadatos alterados: %',x.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $parches$;
DROP FUNCTION vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text) RESTRICT;
DROP INDEX vec_contratacion_temporal.origen_personal_b2_publicacion_ct156;
ALTER TABLE vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 DROP COLUMN transaccion_publicacion;
COMMIT;
