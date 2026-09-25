\set ON_ERROR_STOP on
-- AD3-84 DOWN. Solo sin historia: se niega si Bolsa 000030 sigue instalada, si
-- existe una clave de capacidad de sus audiencias o una atestación de sus
-- operaciones. Deshace exactamente las tres inserciones del núcleo, retira la
-- fachada y devuelve la restricción de audiencias a su preimagen.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000084',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;

DO $proteger$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-84: no instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='solicitud_portal_candidato')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
                WHERE audiencia_consumo LIKE 'vec_bolsa_llamamientos.participaciones_propias.%')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
                WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion' LIKE 'bolsa.participaciones_propias.%'
                   AND convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion'<>'bolsa.participaciones_propias.consultar')
 THEN RAISE EXCEPTION 'AD3-84: DOWN denegado: Bolsa 000030 instalada o historia del portal del candidato' USING ERRCODE='55000'; END IF;
END $proteger$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 actual text; previa text; tras text; meta jsonb; acl aclitem[]; deps jsonb;
 -- Las tres piezas que insertó el UP se localizan por separado, cada una
 -- exactamente una vez: otra extensión instalada después pudo insertarse
 -- junto a las mismas anclas.
 linea_excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''portal_candidato_bolsa''\n';
 linea_runtime text:=E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''portal_candidato_bolsa''\n';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND ((((c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_pausa'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_reactivacion'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.responder_llamamiento'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'))
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.manifestar_disposicion'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'oferta_bolsa'))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl INTO STRICT actual,meta,acl FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(actual)-length(replace(actual,extension,''))<>length(extension)
    OR length(actual)-length(replace(actual,linea_excl,''))<>length(linea_excl)
    OR length(actual)-length(replace(actual,linea_runtime,''))<>length(linea_runtime)
 THEN RAISE EXCEPTION 'AD3-84: núcleo sin la extensión exacta' USING ERRCODE='55000'; END IF;
 previa:=replace(replace(replace(actual,extension,''),linea_runtime,''),linea_excl,'');
 IF strpos(previa,'portal_candidato_bolsa')<>0 THEN RAISE EXCEPTION 'AD3-84: restos de la extensión' USING ERRCODE='55000'; END IF;
 EXECUTE previa;
 SELECT pg_get_functiondef(f) INTO STRICT tras;
 IF tras IS DISTINCT FROM previa
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-84: reversión del núcleo fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencias$
DECLARE d text; a text;
 nuevas text[]:=ARRAY['vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',
                      'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 FOREACH a IN ARRAY nuevas LOOP
  IF length(d)-length(replace(d,', '||quote_literal(a)||'::text',''))<>length(', '||quote_literal(a)||'::text')
  THEN RAISE EXCEPTION 'AD3-84: audiencia ausente o repetida' USING ERRCODE='55000'; END IF;
  d:=replace(d,', '||quote_literal(a)||'::text','');
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||d;
END $audiencias$;
COMMIT;
