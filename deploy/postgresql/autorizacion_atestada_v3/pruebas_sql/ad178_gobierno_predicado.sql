\set ON_ERROR_STOP on
-- Sólo tras AD178 instalada en el clon. Evalúa el predicado real; no consume
-- una autorización ni acredita identidad, grupo, bytes de material o CAS.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $gobierno$
DECLARE c jsonb;d jsonb;v_c jsonb;v_d jsonb;aceptable boolean;n integer;src text;condicion text;
 predicado text:=$predicado$ p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
 AND (c->>'operacion' IN ('vec.catalogos.crear','vec.catalogos.actualizar','vec.catalogos.publicar','vec.catalogos.retirar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_configurable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND (d->>'recurso_ref' ~ '^[a-z][a-z0-9._-]{2,127}:[1-9][0-9]{0,9}$') IS TRUE
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb$predicado$;
BEGIN
 SELECT prosrc INTO STRICT src FROM pg_proc
 WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF length(src)-length(replace(src,E'           OR (\n'||predicado||E')\n',''))<>length(E'           OR (\n'||predicado||E')\n') THEN
  RAISE EXCEPTION 'AD178 prueba: PARO clave=predicado_GOB actual=divergente esperado=una_coincidencia_exacta'; END IF;
 condicion:=replace(predicado,'p_perfil_mutacion',quote_literal('gobierno_plan_nominal_firma_ct'));
 c:=jsonb_build_object('operacion','vec.catalogos.publicar','audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1',
  'efecto_ref','plan_ct_prueba:1','huella_efecto_sha256',repeat('a',64));
 d:=jsonb_build_object('accion',c->>'operacion','modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable',
  'finalidad','gestionar_contratacion_temporal','recurso_ref',c->>'efecto_ref','contexto_recurso_huella_sha256',repeat('a',64),
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','administracion_privilegiada','cuenta_privilegiada',true),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb);
 EXECUTE 'SELECT ('||condicion||') FROM (SELECT $1::jsonb c,$2::jsonb d) datos' INTO aceptable USING c,d;
 IF aceptable IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD178 prueba: PARO clave=forma_GOB actual=rechazada esperado=true_sin_acreditar_consumo'; END IF;
 FOR n IN 1..5 LOOP
  v_c:=c;v_d:=d;
  CASE n
   WHEN 1 THEN v_d:=jsonb_set(v_d,'{vinculo_autenticacion_actor,superficie}','"interna_corporativa"'::jsonb);
   WHEN 2 THEN v_c:=v_c||jsonb_build_object('operacion','vec.catalogos.consultar');v_d:=v_d||jsonb_build_object('accion','vec.catalogos.consultar');
   WHEN 3 THEN v_c:=v_c||jsonb_build_object('audiencia_consumo','vec_contratacion_temporal.firmas_r5.recuperar.v2');
   WHEN 4 THEN v_d:=v_d||jsonb_build_object('recurso_ref','plan*:1');v_c:=v_c||jsonb_build_object('efecto_ref','plan*:1');
   WHEN 5 THEN v_d:=v_d||jsonb_build_object('campos_permitidos','["CanonNominal"]'::jsonb);
  END CASE;
  EXECUTE 'SELECT ('||condicion||') FROM (SELECT $1::jsonb c,$2::jsonb d) datos' INTO aceptable USING v_c,v_d;
  IF aceptable IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD178 prueba: PARO clave=negativo_GOB_% actual=aceptable esperado=false',n; END IF;
 END LOOP;
END $gobierno$;
ROLLBACK;
SELECT 'AD178-GOB-PREDICADO-5-RECHAZOS-OK; sin consumo ni actor acreditado';
