-- CT205: conserva CT155 instalada y corrige solamente la comparación de
-- identificadores de instrumento (UUID) con versiones tipadas (plantilla:/rpt:).
-- Fuente: pg_get_functiondef de la copia PG18 HZ21 posterior a CT204.
-- UP único; no ejecutar DOWN ni reaplicar en una base con historia.
BEGIN;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '15s';

DO $ct205_pre$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_contratacion_temporal.validar_plan_personal_ct155(jsonb,jsonb)');
        p record;
BEGIN
  IF f IS NULL THEN
    RAISE EXCEPTION 'CT205: PARO clave=funcion actual=ausente esperado=CT155' USING ERRCODE='55000';
  END IF;
  SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex') AS huella,
         x.proowner::regrole::text AS propietario, x.prosecdef AS definidor, x.provolatile AS volatilidad,
         x.proconfig AS configuracion, x.proacl::text AS acl
    INTO p FROM pg_catalog.pg_proc x WHERE x.oid=f;
  IF p.huella IS DISTINCT FROM '8b3dbde72cdb63bdb35bf0fddc75ac9fedcc23f8348c2f93838e16c26657a7e0' THEN
    RAISE EXCEPTION 'CT205: PARO clave=preimagen actual=% esperado=8b3dbde72cdb63bdb35bf0fddc75ac9fedcc23f8348c2f93838e16c26657a7e0',p.huella USING ERRCODE='55000';
  END IF;
  IF p.propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario' OR p.definidor IS DISTINCT FROM false
     OR p.volatilidad IS DISTINCT FROM 'i' OR p.configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}' THEN
    RAISE EXCEPTION 'CT205: PARO clave=metadatos actual=%,%,%,%,% esperado=propietario,invoker,immutable,pg_catalog,acl_unica',
      p.propietario,p.definidor,p.volatilidad,p.configuracion,p.acl USING ERRCODE='55000';
  END IF;
END $ct205_pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.validar_plan_personal_ct155(p jsonb, s jsonb)
 RETURNS void
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE k text; nodo jsonb; desde date; hasta date;
BEGIN
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','bolsa','persona_ref','persona_version','persona_fuente','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','fuente_organizacion','fuente_plantilla','fuente_rpt','version_plaza_ref','version_puesto_ref','revision_plaza','revision_puesto','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','catalogo_rpt_version','catalogo_rpt_sha256','categoria_ref','vinculo_revision','vinculo_recibo_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256','ejercicio_sintetico']) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT155: plan incompleto' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','analisis_recibo_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','persona_ref','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','version_plaza_ref','version_puesto_ref','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','categoria_ref','vinculo_recibo_ref','documento_ref'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'CT155: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['version_expediente','analisis_version','persona_version','revision_plaza','revision_puesto','catalogo_rpt_version','vinculo_revision'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'number' OR p->>k!~'^[1-9][0-9]{0,15}$' OR (p->>k)::numeric>9007199254740991 THEN RAISE EXCEPTION 'CT155: versión inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['analisis_sha256','catalogo_rpt_sha256','documento_sha256'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[0-9a-f]{64}$' OR p->>k=repeat('0',64) THEN RAISE EXCEPTION 'CT155: huella inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['persona_fuente'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version','sha256']) IS NOT TRUE
 OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,15}$'
 OR (nodo->>'version')::numeric>9007199254740991 OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64)
 THEN RAISE EXCEPTION 'CT155: fuente inválida' USING ERRCODE='22023'; END IF;
 END LOOP;

 nodo:=p->'fuente_organizacion';
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: fuente organización inválida' USING ERRCODE='22023';END IF;
 FOREACH k IN ARRAY ARRAY['fuente_plantilla','fuente_rpt'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','revision','fuente_ref','fuente_sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' OR nodo->>'revision'!~'^[1-9][0-9]{0,15}$' OR (nodo->>'revision')::numeric>9007199254740991 OR nodo->>'fuente_ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'fuente_sha256'!~'^[0-9a-f]{64}$' OR nodo->>'fuente_sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: instrumento inválido' USING ERRCODE='22023';END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['regimen','modalidad'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version']) IS NOT TRUE OR nodo->>'ref'!~'^[a-z][a-z0-9_:-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,9}$' OR (nodo->>'version')::numeric>2147483647
 THEN RAISE EXCEPTION 'CT155: catálogo inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'bolsa',ARRAY['unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','apertura_registro_sha256','llamamiento_ref','propuesta_ref']) IS NOT TRUE
 OR EXISTS(SELECT 1 FROM jsonb_each_text(p->'bolsa') e WHERE (CASE WHEN e.key LIKE '%sha256' THEN e.value!~'^[0-9a-f]{64}$' OR e.value=repeat('0',64) ELSE e.value!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' END))
 OR p->>'categoria_ref' IS DISTINCT FROM p#>>'{bolsa,categoria_ref}'
 OR jsonb_typeof(p->'ejercicio_sintetico') IS DISTINCT FROM 'boolean'
 OR p->>'motivo_clave'!~'^[a-z][a-z0-9_.-]{1,79}$'
 OR jsonb_typeof(p->'clase_ocupacion') IS DISTINCT FROM 'string'
 OR p->>'clase_ocupacion' !~ '^[a-z][a-z0-9_]{0,63}$'
 OR p->>'desde'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR (p->>'hasta'<>'' AND p->>'hasta'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
 THEN RAISE EXCEPTION 'CT155: intención inválida' USING ERRCODE='22023'; END IF;
 BEGIN desde:=(p->>'desde')::date;hasta:=nullif(p->>'hasta','')::date;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT155: fecha inválida' USING ERRCODE='22023'; END;
 IF NOT isfinite(desde) OR to_char(desde,'YYYY-MM-DD')<>p->>'desde' OR (hasta IS NOT NULL AND (NOT isfinite(hasta) OR hasta<=desde OR to_char(hasta,'YYYY-MM-DD')<>p->>'hasta'))
 OR (p->>'analisis_version')::numeric>(p->>'version_expediente')::numeric
 OR (SELECT jsonb_object_agg(x.clave,p->x.clave) FROM unnest(ARRAY['organizacion_ref','expediente_ref','version_expediente','puesto_ref','plaza_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256']) AS x(clave))
 IS DISTINCT FROM s-ARRAY['clave_idempotencia','version_plantilla_ref','version_rpt_ref']
 OR p->>'version_plaza_ref' IS DISTINCT FROM s->>'version_plantilla_ref' OR p->>'version_puesto_ref' IS DISTINCT FROM s->>'version_rpt_ref'
 OR p->>'version_plaza_ref' IS DISTINCT FROM ('plantilla:' || (p#>>'{fuente_plantilla,ref}'))
 OR p->>'version_puesto_ref' IS DISTINCT FROM ('rpt:' || (p#>>'{fuente_rpt,ref}'))
 THEN RAISE EXCEPTION 'CT155: selección divergente' USING ERRCODE='22023'; END IF;
END $function$;

DO $ct205_post$
DECLARE f regprocedure := pg_catalog.to_regprocedure('vec_contratacion_temporal.validar_plan_personal_ct155(jsonb,jsonb)');
        p record;
BEGIN
  SELECT pg_catalog.pg_get_functiondef(f) AS definicion,x.proowner::regrole::text AS propietario,
         x.prosecdef AS definidor,x.provolatile AS volatilidad,x.proconfig AS configuracion,x.proacl::text AS acl
    INTO p FROM pg_catalog.pg_proc x WHERE x.oid=f;
  IF pg_catalog.strpos(p.definicion,'version_plaza_ref'' IS DISTINCT FROM (''plantilla:'' ||')=0
     OR pg_catalog.strpos(p.definicion,'version_puesto_ref'' IS DISTINCT FROM (''rpt:'' ||')=0
     OR pg_catalog.strpos(p.definicion,'p#>>''{fuente_plantilla,ref}'' IS DISTINCT FROM s->>')<>0 THEN
    RAISE EXCEPTION 'CT205: PARO clave=postimagen actual=sin_comparacion_tipada esperado=dos_comparaciones_tipadas' USING ERRCODE='55000';
  END IF;
  IF p.propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario' OR p.definidor IS DISTINCT FROM false
     OR p.volatilidad IS DISTINCT FROM 'i' OR p.configuracion IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
     OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}' THEN
    RAISE EXCEPTION 'CT205: PARO clave=metadatos_post actual=distintos esperado=metadatos_CT155' USING ERRCODE='55000';
  END IF;
END $ct205_post$;
COMMIT;
