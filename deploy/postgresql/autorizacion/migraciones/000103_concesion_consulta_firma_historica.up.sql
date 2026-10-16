\set ON_ERROR_STOP on
-- AUT103: el cargo consulta conserva cuadro/descarga y sustituye únicamente
-- su concesión de detalle por la misma acción con el campo agregado cerrado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000103',0));
DO $pre$
DECLARE h oid:=to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(text)');
 r oid:=to_regprocedure('vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)');
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR h IS NULL OR r IS NULL
  OR to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(text)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=h AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='97c9649ddcf3ca97611e4bf04cb8ab952da06adcadea3768936de97f3d02be0e')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=r AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='797531cbe53ccf499f9c83a141060dd4397e4f5606516965bebc08ce28012154')
 THEN RAISE EXCEPTION 'AUT103: preimagen o ACL divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(p_operacion text)
 RETURNS jsonb
 LANGUAGE sql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 SELECT CASE p_operacion
 WHEN 'consultar_detalle_selectores' THEN
  jsonb_set(vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2('consultar_detalle'),
   '{campos_permitidos}','["selectores_recuperacion_firmas"]'::jsonb)
 WHEN 'recuperar_r5' THEN
  '{"accion":"contratacion_temporal.documento.firmas_r5_v2.recuperar","modulo_id":"contratacion_temporal","tipo_recurso":"expediente_contratacion_temporal","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"sustancial","campos_permitidos":["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"],"obligaciones":[]}'::jsonb
 ELSE vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(p_operacion)
 END
$function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.documento_rol_cargo_firma_v1(c jsonb, p jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE concesiones jsonb:='[]';x jsonb;op text;anterior text:='';desarrollo boolean:=p->>'esquema'='vec.admin.cargos-firma.plan.v2';clase text;
BEGIN
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>12
 OR NOT c ?& ARRAY['rol_id','version','nombre','version_anterior_sha256','operaciones_v2','competencias','version_rol_sha256','regla_asignacion','organizacion_ref','vigente_desde','vigente_hasta','duracion_propuesta_segundos']
 OR jsonb_typeof(c->'rol_id') IS DISTINCT FROM 'string' OR c->>'rol_id' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'version') IS DISTINCT FROM 'number' OR c->>'version' !~ '^[1-9][0-9]{0,8}$'
 OR jsonb_typeof(c->'nombre') IS DISTINCT FROM 'string' OR char_length(c->>'nombre') NOT BETWEEN 3 AND 200 OR c->>'nombre'<>btrim(c->>'nombre') OR c->>'nombre' ~ '[[:cntrl:]]' OR c->>'nombre' ~ '^[[:space:]]|[[:space:]]$'
 OR jsonb_typeof(c->'version_anterior_sha256') IS DISTINCT FROM 'string'
 OR ((c->>'version')='1' AND c->>'version_anterior_sha256'<>'') OR ((c->>'version')<>'1' AND c->>'version_anterior_sha256' !~ '^[0-9a-f]{64}$')
 OR jsonb_typeof(c->'operaciones_v2') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'operaciones_v2')>(CASE WHEN desarrollo THEN 5 ELSE 2 END)
 OR jsonb_typeof(c->'competencias') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'competencias') NOT BETWEEN (CASE WHEN desarrollo THEN 0 ELSE 1 END) AND 8
 OR jsonb_typeof(c->'version_rol_sha256') IS DISTINCT FROM 'string' OR c->>'version_rol_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(c->'regla_asignacion') IS DISTINCT FROM 'string' OR c->>'regla_asignacion' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'organizacion_ref') IS DISTINCT FROM 'string' OR c->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'AUT53: PARO clave=cargo actual=invalido esperado=campos_exactos' USING ERRCODE='22023'; END IF;

 IF desarrollo THEN
  IF c->>'rol_id' !~ '^ct_cargo_(tecnico|jefatura|diputacion|secretaria)_(firma|consulta|documentos|circuito)_dev$'
  OR jsonb_array_length(c->'operaciones_v2')=0 THEN
  RAISE EXCEPTION 'AUT87: PARO clave=rol_desarrollo actual=invalido esperado=rol_nuevo_ct_cargo_dev' USING ERRCODE='22023'; END IF;
  clase:=split_part(c->>'rol_id','_',4);
  IF (clase='firma' AND jsonb_array_length(c->'competencias')=0) OR (clase<>'firma' AND jsonb_array_length(c->'competencias')<>0)
  OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c->'operaciones_v2') o WHERE NOT(
   (clase='firma' AND o IN('consultar_r5','firmar_vec','recuperar_r5')) OR (clase='consulta' AND o IN('consultar_cuadro','consultar_detalle','consultar_detalle_selectores','descargar_borrador'))
   OR (clase='circuito' AND o='consultar_circuito') OR (clase='documentos' AND (o IN('descargar_original','custodiar_firmado') OR (split_part(c->>'rol_id','_',3)='tecnico' AND o IN('reservar_original','confirmar_original','escribir_original')))))) THEN
  RAISE EXCEPTION 'AUT87: PARO clave=concesiones_desarrollo actual=invalido esperado=permisos_minimos_del_cargo' USING ERRCODE='22023'; END IF;
 END IF;
 -- Operaciones V2: subconjunto ordenado y sin repetir de las admitidas.
 FOR op IN SELECT value FROM jsonb_array_elements_text(c->'operaciones_v2') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF (NOT desarrollo AND op NOT IN('consultar_r5','firmar_vec')) OR (desarrollo AND vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(op) IS NULL) OR op COLLATE "C"<=anterior COLLATE "C" THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=subconjunto_ordenado' USING ERRCODE='22023'; END IF;
  anterior:=op;
  concesiones:=concesiones||jsonb_build_array(CASE WHEN desarrollo THEN vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(op) ELSE vec_autorizacion.concesion_v2_cargo_firma_v1(op) END);
 END LOOP;
 IF desarrollo AND clase='consulta' AND c->'operaciones_v2' ? 'consultar_detalle'
  AND c->'operaciones_v2' ? 'consultar_detalle_selectores' THEN
  RAISE EXCEPTION 'AUT103: dos concesiones de detalle en el mismo rol' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(c->'operaciones_v2') e WHERE jsonb_typeof(e)<>'string') THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=cadenas' USING ERRCODE='22023'; END IF;
 FOR x IN SELECT value FROM jsonb_array_elements(c->'competencias') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF jsonb_typeof(x) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(x))<>3 OR NOT x ?& ARRAY['accion','tipo_recurso','finalidad']
  OR jsonb_typeof(x->'accion') IS DISTINCT FROM 'string' OR x->>'accion' !~ '^contratacion_temporal\.documento(\.[a-z][a-z0-9_]{1,63}){1,4}$'
  OR jsonb_typeof(x->'tipo_recurso') IS DISTINCT FROM 'string' OR x->>'tipo_recurso' !~ '^documento_([a-z0-9_]{1,80}_)?contratacion_temporal$'
  OR jsonb_typeof(x->'finalidad') IS DISTINCT FROM 'string' OR x->>'finalidad' !~ '^[a-z][a-z0-9_]{2,127}$' OR x->>'finalidad' ~ '(fiscaliz|intervenc)'
  THEN RAISE EXCEPTION 'AUT53: PARO clave=competencia actual=invalida esperado=documento_contratacion_temporal' USING ERRCODE='22023'; END IF;
  concesiones:=concesiones||jsonb_build_array(jsonb_build_object('accion',x->>'accion','modulo_id','contratacion_temporal','tipo_recurso',x->>'tipo_recurso',
   'finalidades',jsonb_build_array(x->>'finalidad'),'garantia_minima',CASE WHEN desarrollo THEN 'sustancial' ELSE 'alto' END,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb));
 END LOOP;
 RETURN jsonb_build_object('rol_id',c->>'rol_id','version',(c->>'version')::bigint,'nombre',c->>'nombre','estado','publicada','concesiones',concesiones,
  'publicada_por','operacion:cargos_firma:'||(p->>'operacion_ref'),'publicada_en',p->>'preparado_en');
END $function$;

REVOKE ALL ON FUNCTION vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(text) FROM PUBLIC;
DO $post$
DECLARE h oid:=to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(text)');
 r oid:=to_regprocedure('vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)');
BEGIN
 IF h IS NULL OR r IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=h AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='77bb5555ee4d450a44b0ea6fa294b8e015216ca232d541160c66ee8457817e2d')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=r AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d3057878ac315982a222628a33d9606732d78f663178d171e4d1434dd0de2ad7')
 THEN RAISE EXCEPTION 'AUT103: postimagen o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
