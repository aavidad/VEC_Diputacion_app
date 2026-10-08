\set ON_ERROR_STOP on
-- Ejecutar solo en PostgreSQL 18 desechable, despues de AD218 y B93.
BEGIN READ ONLY;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(text,text,text,text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF f IS NULL OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_desarrollo',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))
 THEN RAISE EXCEPTION 'B93: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
ROLLBACK;

SELECT count(*) AS consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3 \gset
SELECT count(*) AS auditorias_antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 \gset
SELECT count(*) AS lotes_antes FROM vec_bolsa_importacion_convoca.lote \gset
SELECT count(*) AS filas_antes FROM vec_bolsa_importacion_convoca.fila_staging \gset
SELECT count(*) AS originales_antes FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada \gset
SELECT count(*) AS constituciones_antes FROM vec_bolsa_llamamientos.constitucion \gset
SELECT count(*) AS vinculos_antes FROM vec_bolsa_llamamientos.vinculo_candidato \gset
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $negativos$
DECLARE fichero text:=pg_catalog.repeat('c',64); categoria text:='categoria:rpt:x'; actor text:='per_x';
 acta text:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(pg_catalog.repeat('c',64)||pg_catalog.chr(31)||'categoria:rpt:x','UTF8')),'hex');
 canon text:='{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{"desplazamiento":"0","esquema":"vec.bolsa.rrhh.carga_convoca.vista_previa.v1","fase":"vista_previa","filtro":"todas","limite":"50"}}';
 contexto bytea; huella text; c bytea; d bytea; fallo text;
BEGIN
 contexto:=pg_catalog.convert_to(canon,'UTF8');
 huella:=pg_catalog.encode(pg_catalog.sha256(contexto),'hex');
 c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,'huella_efecto_sha256',huella,
  'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','dec_ficticia','principal_id',actor,
  'recurso_ref',acta,'contexto_recurso_huella_sha256',huella,'accion','bolsa.carga_convoca.confirmar',
  'finalidad','carga_bolsa_convoca','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8');
 -- Una pagina canonica pero sin evidencia V3 autentica no registra exito.
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,actor,categoria,fichero,contexto,c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: material V3 ficticio aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'AD218: consumo B1 denegado%' THEN RAISE; END IF;
 END;
 -- Una preimagen JSON con la misma semantica pero claves duplicadas se rechaza
 -- sobre sus bytes, antes de llegar a AD218.
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,actor,categoria,fichero,pg_catalog.convert_to(pg_catalog.replace(canon,
   '"filtro":"todas"','"filtro":"todas","filtro":"todas"'),'UTF8'),
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: claves duplicadas aceptadas' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B93: contexto no canonico%' THEN RAISE; END IF;
 END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,actor,categoria,fichero,pg_catalog.convert_to(pg_catalog.replace(canon,
   '"limite":"50"','"limite":"101"'),'UTF8'),
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: pagina fuera de limite aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B93: pagina fuera de limite%' THEN RAISE; END IF;
 END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,'per_ajeno',categoria,fichero,contexto,c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: actor ajeno aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B93: decision divergente%' THEN RAISE; END IF;
 END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,actor,'categoria:rpt:otra',fichero,contexto,c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: categoria ajena aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B93: vista previa no autorizada%' THEN RAISE; END IF;
 END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
   acta,actor,categoria,fichero,pg_catalog.convert_to(pg_catalog.replace(canon,
   '"filtro":"todas"','"filtro":"aceptadas"'),'UTF8'),
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B93: pagina distinta de la atestada aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B93: decision divergente%' THEN RAISE; END IF;
 END;
END $negativos$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SELECT CASE WHEN
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)=:'consumos_antes'::bigint
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)=:'auditorias_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote)=:'lotes_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.fila_staging)=:'filas_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada)=:'originales_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion)=:'constituciones_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato)=:'vinculos_antes'::bigint
 THEN 'true' ELSE 'false' END AS historia_intacta \gset
\if :historia_intacta
\else
SELECT 1/0 AS b93_historia_divergente;
\endif
SELECT 'B93 negativos OK' AS resultado;
