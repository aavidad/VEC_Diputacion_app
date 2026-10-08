-- B95: ejecutar sólo sobre clon PostgreSQL 18 desechable tras BIC7/B79/B80/B95.
-- Material V3 ficticio: los casos comprueban denegación, nunca éxito nominal.
\set ON_ERROR_STOP on
BEGIN READ ONLY;
DO $acl$
DECLARE antigua oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        nueva oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF antigua IS NULL OR nueva IS NULL
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_desarrollo',antigua,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_desarrollo',nueva,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',nueva,'EXECUTE')
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=nueva
   AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC',
    'lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s'])
 THEN RAISE EXCEPTION 'B95: ACL o GUC incompatibles' USING ERRCODE='55000'; END IF;
END $acl$;
ROLLBACK;

SELECT count(*) AS consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3 \gset
SELECT count(*) AS auditorias_antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 \gset
SELECT count(*) AS lotes_antes FROM vec_bolsa_importacion_convoca.lote \gset
SELECT count(*) AS originales_antes FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada \gset
SELECT count(*) AS constituciones_antes FROM vec_bolsa_llamamientos.constitucion \gset
SELECT count(*) AS recibos_antes FROM vec_bolsa_llamamientos.recibo_carga_convoca \gset
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $negativos$
DECLARE h text:=pg_catalog.repeat('c',64); cat text:='categoria:rpt:x'; actor text:='per_x';
 acta text:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(pg_catalog.repeat('c',64)||pg_catalog.chr(31)||'categoria:rpt:x','UTF8')),'hex');
 a jsonb; b bytea; filas jsonb:='[{"numero":1}]'; entradas jsonb:='[{"participacion_ref":"p","fila_numero":1}]';
 confirmacion bytea:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{}}','UTF8');
 previa bytea:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{"desplazamiento":"0","esquema":"vec.bolsa.rrhh.carga_convoca.vista_previa.v1","fase":"vista_previa","filtro":"todas","limite":"50"}}','UTF8');
 c bytea; d bytea; huella text; huella_previa text;
BEGIN
 a:=pg_catalog.jsonb_build_object('acta_ref',acta,'categoria_ref',cat,'bolsa_ref','bolsa:x',
  'huella_fichero_sha256',h,'actor_ref','actor:rrhh:'||pg_catalog.substr(pg_catalog.encode(
   pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||actor,'UTF8')),'hex'),1,32));
 b:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('bolsa_ref','bolsa:x',
  'categoria_ref',cat,'huella_listado_sha256',h)::text,'UTF8');
 huella:=pg_catalog.encode(pg_catalog.sha256(confirmacion),'hex');
 huella_previa:=pg_catalog.encode(pg_catalog.sha256(previa),'hex');
 c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,
  'huella_efecto_sha256',huella,'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','decision:ficticia',
  'principal_id',actor,'recurso_ref',acta,'contexto_recurso_huella_sha256',huella)::text,'UTF8');
 -- Una autorización de vista previa con la misma acta y la misma audiencia
 -- nunca puede pasar el guard de confirmación, incluso si su hash coincide.
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v2(a,filas,'{}',acta,actor,cat,'bolsa:x',
   1,b,now(),'ins',1,'\x7b7d',now(),now(),entradas,now(),'[]',previa,
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B95: contexto preview aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B95: contexto de confirmación inválido%' THEN RAISE; END IF;
 END;
 -- Contexto de confirmación canónico pero capacidad/decisión selladas para
 -- otra página: se deniega antes de AD218 y no hay consumo.
 c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,
  'huella_efecto_sha256',huella_previa,'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','decision:ficticia',
  'principal_id',actor,'recurso_ref',acta,'contexto_recurso_huella_sha256',huella_previa)::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v2(a,filas,'{}',acta,actor,cat,'bolsa:x',
   1,b,now(),'ins',1,'\x7b7d',now(),now(),entradas,now(),'[]',confirmacion,
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B95: decisión de otra página aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B95: contexto y decisión divergentes%' THEN RAISE; END IF;
 END;
 -- Con la huella correcta, material ficticio todavía debe caer en AD218.
 c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,
  'huella_efecto_sha256',huella,'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','decision:ficticia',
  'principal_id',actor,'recurso_ref',acta,'contexto_recurso_huella_sha256',huella)::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v2(a,filas,'{}',acta,actor,cat,'bolsa:x',
   1,b,now(),'ins',1,'\x7b7d',now(),now(),entradas,now(),'[]',confirmacion,
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B95: material V3 ficticio aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'AD218:%' THEN RAISE; END IF;
 END;
END $negativos$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SELECT CASE WHEN
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)=:'consumos_antes'::bigint
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)=:'auditorias_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote)=:'lotes_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada)=:'originales_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion)=:'constituciones_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_llamamientos.recibo_carga_convoca)=:'recibos_antes'::bigint
 THEN 'true' ELSE 'false' END AS historia_intacta \gset
\if :historia_intacta
\else
SELECT 1/0 AS b95_historia_divergente;
\endif
SELECT 'B95 negativos OK' AS resultado;
