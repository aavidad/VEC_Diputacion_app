\set ON_ERROR_STOP on
-- CT250 / F1-125A. Vincular comprueba la referencia opaca de una O6 propia;
-- la lectura ofrece su emisión B98 y conserva el origen del participante.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000250',0));
DO $pre_ct250$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL
    OR to_regclass('vec_contratacion_temporal.seleccion_confirmada_unica_expediente_ct238') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL THEN
  RAISE EXCEPTION 'CT250: dependencias incompatibles' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      IS DISTINCT FROM '44c8d0e2c3597533300cdd066279a7c4a17526dd2c9eb1f158722a0c36249fcd' THEN
  RAISE EXCEPTION 'CT250: lector instalado divergente o ya sustituido' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      IS DISTINCT FROM 'fd7421e9d5dba11dda82368873a8ea41a30fc5002f921e0f226b649ba13ac344' THEN
  RAISE EXCEPTION 'CT250: vínculo instalado divergente' USING ERRCODE='55000'; END IF;
 -- SHA-256 públicos de pg_get_functiondef instalado; no son credenciales.
 IF EXISTS (SELECT 1 FROM (VALUES
  ('vec_bolsa_llamamientos.verificar_emision_ct_v1(text,text,text,text,text)',
   '918300ef8986c910f11e718fc25dc6534cd9b87cee3560b4d77c88345c83de22','statement_timeout=3s'),
  ('vec_bolsa_llamamientos.listar_emisiones_ct_v1(text,text,timestamp with time zone,text,integer,text[])',
   '3d406f5c21847b35e80611345323197776c8e86d8cdd3731a9afc686e3e85ae5','statement_timeout=3s')) esperado(firma,huella,plazo)
  LEFT JOIN pg_proc q ON q.oid=to_regprocedure(esperado.firma)
  WHERE q.oid IS NULL OR q.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR q.prosecdef IS DISTINCT FROM true OR q.provolatile IS DISTINCT FROM 's'
    OR q.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_contratacion_temporal_propietario=X/vec_bolsa_llamamientos_propietario}'
    OR q.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',esperado.plazo]
    OR encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') IS DISTINCT FROM esperado.huella) THEN
  RAISE EXCEPTION 'CT250: lectura o verificación B98 divergente' USING ERRCODE='55000'; END IF;
END $pre_ct250$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE s jsonb; d jsonb; c record; actual_ct record; amb record; anterior record;
 clave uuid; version_esperada numeric; material_h text; contexto_h text;
 numero text; recibo text; evento text; instante timestamptz(6); salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT201: sesión de vínculo denegada' USING ERRCODE='42501';
 END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 4096 THEN
  RAISE EXCEPTION 'CT201: material inválido' USING ERRCODE='22023';
 END IF;
 BEGIN
  s:=p_material::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  clave:=(s->>'clave_idempotencia')::uuid;
  version_esperada:=(s->>'version_esperada')::numeric;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT201: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM json_each(p_material::json))<>8
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>8
    OR (SELECT array_agg(key ORDER BY ord) FROM json_each(p_material::json) WITH ORDINALITY AS j(key,val,ord))
      IS DISTINCT FROM ARRAY['esquema','organizacion_ref','expediente_ref','version_esperada',
        'bolsa_ref','llamamiento_ref','recibo_emision_ref','clave_idempotencia']
    OR s->>'esquema'<>'vec.ct.vinculo-emision-bolsa.v1'
    OR version_esperada NOT BETWEEN 1 AND 9007199254740991::numeric
    OR version_esperada<>trunc(version_esperada)
    OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR s->>'bolsa_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'llamamiento_ref' !~ '^llamamiento:[0-9a-f]{64}$'
    OR s->>'recibo_emision_ref' !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR s->>'clave_idempotencia' IS DISTINCT FROM clave::text THEN
  RAISE EXCEPTION 'CT201: contrato de vínculo inválido' USING ERRCODE='22023';
 END IF;
 -- CT241 / F1-19: un nombramiento derivado no se vincula a otro llamamiento.
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
   WHERE alta.expediente_ref=s->>'expediente_ref' AND alta.derivacion_ref IS NOT NULL) THEN
  RAISE EXCEPTION 'CT241: expediente derivado sin selección nueva' USING ERRCODE='23505';
 END IF;
	material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
	SELECT * INTO amb FROM vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
	 s->>'organizacion_ref',s->>'expediente_ref',version_esperada,s->>'bolsa_ref',
	 s->>'llamamiento_ref',s->>'recibo_emision_ref',s->>'clave_idempotencia');
	IF NOT FOUND THEN
	 RAISE EXCEPTION 'CT201: ámbitos del expediente divergentes' USING ERRCODE='23505';
	END IF;
	contexto_h:=encode(sha256(convert_to(
	   '{"ambitos":{"categoria_ref":'||to_json(amb.categoria_ref)::text||
	   ',"centro_ref":'||to_json(amb.centro_ref)::text||
	   ',"organizacion_ref":'||(s->'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'CT201: decisión no ligada al material' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM s->>'expediente_ref' OR c.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: consumo de efecto divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:vinculo_bolsa:'||(s->>'expediente_ref'),0));
 SELECT * INTO anterior FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.organizacion_ref=s->>'organizacion_ref' AND x.clave_idempotencia=clave;
 IF FOUND THEN
  IF anterior.material_sha256 IS DISTINCT FROM material_h
     OR anterior.actor_ref IS DISTINCT FROM d->>'principal_id'
     OR anterior.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
   RAISE EXCEPTION 'CT201: clave usada con otro contenido' USING ERRCODE='23505';
  END IF;
  RETURN anterior.recibo_json||jsonb_build_object('reutilizado',true);
 END IF;
 SELECT e.organizacion_ref,e.numero_visible,actual.version,v.agregado_json
  INTO actual_ct FROM vec_contratacion_temporal.expediente_alta e
  JOIN vec_contratacion_temporal.expediente_integral_actual actual ON actual.expediente_ref=e.expediente_ref
  JOIN vec_contratacion_temporal.expediente_version_integral v
    ON v.expediente_ref=actual.expediente_ref AND v.version=actual.version
  WHERE e.expediente_ref=s->>'expediente_ref' AND e.organizacion_ref=s->>'organizacion_ref'
  FOR SHARE OF actual;
 IF NOT FOUND OR actual_ct.version IS DISTINCT FROM version_esperada
    OR actual_ct.agregado_json#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>'{via_cobertura,bolsa_ref}' IS NOT NULL
    OR actual_ct.agregado_json#>>'{solicitud,centro_ref}' IS DISTINCT FROM amb.centro_ref
    OR actual_ct.agregado_json#>>'{analisis,categoria_ref}' IS DISTINCT FROM amb.categoria_ref THEN
  RAISE EXCEPTION 'CT201: versión o bolsa de cobertura divergente' USING ERRCODE='23505';
 END IF;
 -- La O6 original puede haber emitido con la referencia opaca del expediente.
 -- El recibo B98 decide cuál de las dos referencias es realmente la suya.
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   JOIN vec_contratacion_temporal.expediente_version_integral h
     ON h.expediente_ref=s->>'expediente_ref' AND h.version=CASE
      WHEN e.recibo_json->>'version_expediente' ~ '^[1-9][0-9]{0,15}$'
      THEN (e.recibo_json->>'version_expediente')::numeric END
   WHERE e.tipo_seleccion='llamamiento' AND e.situacion='confirmada'
     AND e.artefacto_canonico IS NOT NULL
     AND jsonb_typeof(e.recibo_json->'version_expediente')='number'
     AND e.solicitud_json->>'organizacion_ref'=s->>'organizacion_ref'
     AND e.solicitud_json->>'expediente_ref'=s->>'expediente_ref'
     AND e.recibo_json->>'organizacion_ref'=s->>'organizacion_ref'
     AND e.recibo_json->>'expediente_ref'=s->>'expediente_ref'
     AND e.recibo_json#>>'{bolsa,referencia}'=s->>'bolsa_ref'
     AND e.recibo_json->>'llamamiento_ref'=s->>'llamamiento_ref'
     AND e.recibo_json->>'recibo_ref'=s->>'recibo_emision_ref'
     AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
     AND e.recibo_json->'propuesta_generada'='true'::jsonb
     AND h.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
     AND h.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
     AND h.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL
     AND h.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL) THEN
  numero:=s->>'expediente_ref';
 ELSE
  numero:=vec_contratacion_temporal.numero_visible_vigente_v1(s->>'expediente_ref',actual_ct.numero_visible);
 END IF;
 IF vec_bolsa_llamamientos.verificar_emision_ct_v1(
    s->>'bolsa_ref',s->>'llamamiento_ref',s->>'recibo_emision_ref',numero,
    actual_ct.agregado_json#>>'{analisis,categoria_ref}') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: emisión real ajena al expediente' USING ERRCODE='42501';
 END IF;
 instante:=date_trunc('microseconds',clock_timestamp());
 recibo:='recibo:ct:bolsa:'||gen_random_uuid()::text;
 evento:='evento:ct:bolsa:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('expediente_ref',s->>'expediente_ref',
  'bolsa_ref',s->>'bolsa_ref','llamamiento_ref',s->>'llamamiento_ref',
  'recibo_emision_ref',s->>'recibo_emision_ref','recibo_vinculo_ref',recibo,
  'vinculado_en',instante,'auditoria_ref',c.auditoria_ref,'evento_ref',evento,'reutilizado',false);
 INSERT INTO vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(
  organizacion_ref,expediente_ref,bolsa_ref,llamamiento_ref,recibo_emision_ref,
  version_expediente_esperada,clave_idempotencia,material_sha256,actor_ref,perfil_ref,
  decision_ref,auditoria_ref,consumo_sha256,recibo_ref,vinculado_en,recibo_json)
 VALUES(s->>'organizacion_ref',s->>'expediente_ref',s->>'bolsa_ref',s->>'llamamiento_ref',
  s->>'recibo_emision_ref',version_esperada,clave,material_h,d->>'principal_id',d->>'perfil_activo_ref',
  c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,recibo,instante,salida);
 INSERT INTO vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1(
  evento_ref,expediente_ref,llamamiento_ref,recibo_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(evento,s->>'expediente_ref',s->>'llamamiento_ref',recibo,
  'contratacion_temporal.bolsa.emision_vinculada.v1',salida,
  encode(sha256(convert_to(salida::text,'UTF8')),'hex'),instante);
 RETURN salida;
END $function$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_cursor text)
 RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, resultado_bolsa jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE d record; actual record; vinculos jsonb; sugeridas jsonb; origenes jsonb; opacas jsonb; nativa record; n integer;
 entradas jsonb; todos jsonb; lote jsonb; pagina_n integer; antes_en timestamptz; antes_ref text;
 ultimo_en timestamptz; ultimo_ref text; mas boolean:=false; siguiente text; bolsa_vigente text;
 personas_text text; personas numeric; aceptaciones integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_consultor_rrhh','MEMBER') IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR (p_cursor IS NOT NULL AND p_cursor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z#llamamiento:[0-9a-f]{64}$') THEN
  RAISE EXCEPTION 'CT201: lectura de resultado denegada' USING ERRCODE='42501';
 END IF;
 IF p_cursor IS NOT NULL THEN
  antes_en:=split_part(p_cursor,'#',1)::timestamptz;
  antes_ref:=split_part(p_cursor,'#',2);
 END IF;
 SELECT * INTO STRICT d FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
  'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)),
  '[]'::jsonb) INTO todos FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.expediente_ref=d.expediente_ref;
 n:=jsonb_array_length(todos);
 aceptaciones:=vec_bolsa_llamamientos.contar_aceptaciones_firmes_ct_v1(todos);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
   'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO entradas
 FROM (SELECT v.bolsa_ref,v.llamamiento_ref,v.recibo_emision_ref,v.vinculado_en
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC LIMIT 20) x;
 pagina_n:=jsonb_array_length(entradas);
 IF pagina_n=20 THEN
  SELECT v.vinculado_en,v.llamamiento_ref INTO ultimo_en,ultimo_ref
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC OFFSET 19 LIMIT 1;
  SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
    WHERE v.expediente_ref=d.expediente_ref
      AND (v.vinculado_en,v.llamamiento_ref)<(ultimo_en,ultimo_ref)) INTO mas;
  IF mas THEN
   siguiente:=to_char(ultimo_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')||'#'||ultimo_ref;
  END IF;
 END IF;
 lote:=vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(entradas);
 -- O6 se recorre una vez para esta página; el mapa solo contiene emisiones
 -- con una selección exacta y snapshot fiscalizado original.
 WITH pagina AS MATERIALIZED (
  SELECT x.organizacion_ref,x.expediente_ref,x.bolsa_ref,x.llamamiento_ref,
   x.recibo_emision_ref
  FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  JOIN LATERAL jsonb_array_elements(entradas) item(valor)
   ON item.valor->>'llamamiento_ref'=x.llamamiento_ref
  WHERE x.expediente_ref=d.expediente_ref
 ), o6 AS MATERIALIZED (
  SELECT e.solicitud_json->>'organizacion_ref' AS organizacion_ref,
   e.solicitud_json->>'expediente_ref' AS expediente_ref,
   e.recibo_json#>>'{bolsa,referencia}' AS bolsa_ref,
   e.recibo_json->>'llamamiento_ref' AS llamamiento_ref,
   e.recibo_json->>'recibo_ref' AS recibo_emision_ref,
   e.solicitud_json->'version_expediente' AS version_solicitud,
   e.recibo_json->'version_expediente' AS version_recibo,
   e.recibo_json->>'organizacion_ref' AS organizacion_recibo,
   e.recibo_json->>'expediente_ref' AS expediente_recibo
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  WHERE e.situacion='confirmada'
   AND e.solicitud_json->>'expediente_ref'=d.expediente_ref
   AND e.recibo_json->'propuesta_generada'='true'::jsonb
 ), exactas AS (
  SELECT x.llamamiento_ref,count(*) AS cantidad FROM pagina x
  JOIN o6 e ON e.organizacion_ref=x.organizacion_ref
   AND e.expediente_ref=x.expediente_ref
   AND e.organizacion_recibo=x.organizacion_ref
   AND e.expediente_recibo=x.expediente_ref
   AND e.bolsa_ref=x.bolsa_ref AND e.llamamiento_ref=x.llamamiento_ref
   AND e.recibo_emision_ref=x.recibo_emision_ref
   AND e.version_recibo=e.version_solicitud
   AND jsonb_typeof(e.version_recibo)='number'
  JOIN vec_contratacion_temporal.expediente_version_integral h
   ON h.expediente_ref=x.expediente_ref AND h.version=CASE
      WHEN e.version_recibo#>>'{}' ~ '^[1-9][0-9]{0,15}$'
      THEN (e.version_recibo#>>'{}')::numeric END
   AND h.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
   AND h.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
   AND h.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL
   AND h.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
  GROUP BY x.llamamiento_ref
 ) SELECT coalesce(jsonb_object_agg(llamamiento_ref,cantidad=1),'{}'::jsonb)
 INTO origenes FROM exactas;
 -- CT201 fija la emisión original; B98 entrega sus participantes. La O6
 -- exacta y única acredita al participante solo cuando B98 devuelve uno.
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa_ref',x.bolsa_ref,'llamamiento_ref',x.llamamiento_ref,
   'recibo_emision_ref',x.recibo_emision_ref,
   'recibo_vinculo_ref',x.recibo_ref,'vinculado_en',x.vinculado_en,
   'emitido_en',b.valor->'emitido_en',
   'participaciones',(
    SELECT coalesce(jsonb_agg(part.valor || jsonb_build_object(
     'origen_ct_confirmado',coalesce(
       jsonb_array_length(b.valor->'participaciones')=1
       AND part.valor->>'respuesta'='acepta' AND part.valor->>'modo'='firme'
       AND part.valor->>'recibo_respuesta_ref' IS NOT NULL
       AND part.valor->>'respondida_en' IS NOT NULL
       AND origenes->>x.llamamiento_ref='true',false))
     ORDER BY part.ordinal),'[]'::jsonb)
    FROM jsonb_array_elements(b.valor->'participaciones') WITH ORDINALITY AS part(valor,ordinal)
   ))
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO vinculos
 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
 JOIN LATERAL jsonb_array_elements(lote) b(valor)
   ON b.valor->>'llamamiento_ref'=x.llamamiento_ref
  AND b.valor->>'bolsa_ref'=x.bolsa_ref
  AND b.valor->>'recibo_emision_ref'=x.recibo_emision_ref
 WHERE x.expediente_ref=d.expediente_ref;
 IF jsonb_array_length(vinculos)<>pagina_n THEN
  RAISE EXCEPTION 'CT201: resultado Bolsa no recuperable' USING ERRCODE='55000';
 END IF;
 SELECT e.numero_visible,a.version,v.agregado_json INTO actual
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.expediente_ref=d.expediente_ref;
 IF NOT FOUND OR actual.version IS DISTINCT FROM d.version_expediente THEN
  RAISE EXCEPTION 'CT201: agregado de lectura divergente' USING ERRCODE='55000';
 END IF;
 personas_text:=actual.agregado_json#>>'{solicitud,numero_personas}';
 IF personas_text IS NULL THEN
  personas_text:=actual.agregado_json#>>'{solicitud,necesidad,campos,numero_personas}';
 END IF;
 IF personas_text IS NOT NULL THEN
  IF personas_text !~ '^[1-9][0-9]{0,9}$' THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  IF personas_text::numeric>4294967295 THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  personas:=personas_text::numeric;
 END IF;
 sugeridas:='[]'::jsonb;
 IF actual.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND actual.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND actual.agregado_json#>'{via_cobertura,bolsa_ref}' IS NULL
    AND actual.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL THEN
  bolsa_vigente:=vec_bolsa_llamamientos.resolver_bolsa_vigente_ct_v1(
   actual.agregado_json#>>'{analisis,categoria_ref}');
 END IF;
 IF bolsa_vigente IS NOT NULL THEN
  SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY b.emitido_en DESC,b.llamamiento_ref DESC),'[]'::jsonb)
   INTO sugeridas FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,
    vec_contratacion_temporal.numero_visible_vigente_v1(d.expediente_ref,actual.numero_visible),
    NULL,NULL,20,ARRAY(SELECT x.llamamiento_ref
     FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
     WHERE x.expediente_ref=d.expediente_ref)) b;
  -- CT238 permite una O6 confirmada por expediente; B98 devuelve su
  -- referencia de configuración real, aunque sea la opaca del expediente.
  SELECT e.recibo_json->>'llamamiento_ref' AS llamamiento_ref,
   e.recibo_json->>'recibo_ref' AS recibo_emision_ref INTO nativa
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  JOIN vec_contratacion_temporal.expediente_version_integral h
   ON h.expediente_ref=d.expediente_ref AND h.version=CASE
    WHEN e.recibo_json->>'version_expediente' ~ '^[1-9][0-9]{0,15}$'
    THEN (e.recibo_json->>'version_expediente')::numeric END
  WHERE e.tipo_seleccion='llamamiento' AND e.situacion='confirmada'
    AND e.artefacto_canonico IS NOT NULL
    AND jsonb_typeof(e.recibo_json->'version_expediente')='number'
    AND e.solicitud_json->>'organizacion_ref'=p_alcance.organizacion_ref
    AND e.solicitud_json->>'expediente_ref'=d.expediente_ref
    AND e.recibo_json->>'organizacion_ref'=p_alcance.organizacion_ref
    AND e.recibo_json->>'expediente_ref'=d.expediente_ref
    AND e.recibo_json#>>'{bolsa,referencia}'=bolsa_vigente
    AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
    AND e.recibo_json->'propuesta_generada'='true'::jsonb
    AND e.recibo_json->>'llamamiento_ref' ~ '^llamamiento:[0-9a-f]{64}$'
    AND e.recibo_json->>'recibo_ref'='recibo:'||(e.recibo_json->>'llamamiento_ref')
    AND h.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND h.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND h.agregado_json#>>'{analisis,categoria_ref}'=actual.agregado_json#>>'{analisis,categoria_ref}'
    AND h.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
      WHERE x.llamamiento_ref=e.recibo_json->>'llamamiento_ref');
  IF FOUND THEN
   SELECT coalesce(jsonb_agg(to_jsonb(b)),'[]'::jsonb) INTO opacas
   FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,d.expediente_ref,NULL,NULL,20,
    ARRAY(SELECT x.llamamiento_ref FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
      WHERE x.expediente_ref=d.expediente_ref)) b
   WHERE b.llamamiento_ref=nativa.llamamiento_ref
     AND b.recibo_emision_ref=nativa.recibo_emision_ref;
   IF jsonb_array_length(opacas)=1 THEN
    SELECT coalesce(jsonb_agg(z.valor ORDER BY z.orden),'[]'::jsonb) INTO sugeridas
    FROM (SELECT opacas->0 AS valor,0::bigint AS orden
      UNION ALL
      SELECT m.valor,m.pos FROM jsonb_array_elements(sugeridas) WITH ORDINALITY m(valor,pos)
      WHERE m.valor->>'llamamiento_ref' IS DISTINCT FROM nativa.llamamiento_ref
      ORDER BY orden LIMIT 20) z;
   END IF;
  END IF;
 END IF;
 RETURN QUERY SELECT d.contenido_canonico,d.esquema,d.acceso_ref,d.secuencia,
  d.anterior_sha256,d.huella_sha256,d.vinculo_identidad_huella_sha256,
  d.alcance_huella_sha256,d.registrada_en,d.auditoria_vec_ref,
  d.auditoria_vec_huella_sha256,d.consumo_vec_huella_sha256,
  d.contenido_huella_sha256,d.resultado_huella_sha256,d.cursor_huella_sha256,
  d.generada_en,d.expediente_ref,d.version_expediente,d.total,d.recibo_sello_sha256,
  jsonb_build_object('vinculos',vinculos,'total_vinculos',n,
   'personas_solicitadas',personas,'aceptaciones_firmes',aceptaciones,
   'emisiones_vinculables',sugeridas,'siguiente_cursor',siguiente);
END $function$;
DO $post_ct250$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      IS DISTINCT FROM '82a116f0069fb7220fa81e6411b01ea6fde4a474f66e3f5863f8bb2d944485b1' THEN
  RAISE EXCEPTION 'CT250: lector final divergente' USING ERRCODE='55000'; END IF;
 IF (SELECT encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex')
     FROM pg_proc q WHERE q.oid=to_regprocedure(
      'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
    IS DISTINCT FROM '7b8874a22af362380a46f85f62ad25ab9ab2ac6cc453ff666bf34a25f982287d' THEN
  RAISE EXCEPTION 'CT250: sobrecarga de lectura divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      IS DISTINCT FROM '9f8ccd5cf39f77be1f33607ab63041115eaacb76fa62fad1f242b9815e356ae3' THEN
  RAISE EXCEPTION 'CT250: vínculo final divergente' USING ERRCODE='55000'; END IF;
END $post_ct250$;
COMMIT;
