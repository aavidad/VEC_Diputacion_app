\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:ct231:seleccion-pendiente-aviso',0));
DO $pre_ct231$
DECLARE p pg_catalog.pg_proc%ROWTYPE; actual text;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT231: PostgreSQL 18 requerido' USING ERRCODE='55000';
 END IF;
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'] THEN
  RAISE EXCEPTION 'CT231: metadatos instalados divergentes' USING ERRCODE='55000';
 END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF actual IS DISTINCT FROM '11de58f7276bc11ead3af080f55d7b6d5936d482d4b3c8271744d99765f098cc' THEN
  RAISE EXCEPTION 'CT231: preimagen divergente' USING ERRCODE='55000';
 END IF;
END $pre_ct231$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
 s jsonb; d jsonb; c jsonb; v_hash text; v_material_hash text; v_consumo record;
 v_cursor_fecha timestamptz(6); v_cursor_ref text; v_items jsonb; v_cantidad integer;
 v_total bigint:=0; v_ancla text; v_estados text:=''; v_fila record;
 v_pendientes bigint:=0; v_pendiente_ref text; v_pendiente_fecha timestamptz(6);
 v_pendiente_recibo jsonb; v_pendiente_artefacto text; v_pendiente_fila jsonb;
 v_fecha_texto text; v_claves text[];
 v_siguiente text:=''; v_limite integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT140: sesión denegada' USING ERRCODE='P1403'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 2048
 THEN RAISE EXCEPTION 'CT140: material inválido' USING ERRCODE='P1400'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT140: JSON inválido' USING ERRCODE='P1400'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'CT140: objeto requerido' USING ERRCODE='P1400'; END IF;
 IF (SELECT count(*) FROM json_each(p_material::json))<>4
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>4
    OR jsonb_typeof(s->'organizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'expediente_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'limite') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'cursor') IS DISTINCT FROM 'string'
    OR (s->>'organizacion_ref')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'expediente_ref')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR ((s->>'cursor')<>'' AND (s->>'cursor')!~'^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}#[0-9a-f]{64}$')
    OR (s->>'limite')!~'^[0-9]{1,2}$'
 THEN RAISE EXCEPTION 'CT140: campos inválidos' USING ERRCODE='P1400'; END IF;
 v_limite:=(s->>'limite')::integer;
 IF v_limite NOT BETWEEN 1 AND 20
 THEN RAISE EXCEPTION 'CT140: límite inválido' USING ERRCODE='P1400'; END IF;
 v_material_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_hash:=encode(sha256(convert_to(
  '{"ambitos":{"organizacion_ref":"'||(s->>'organizacion_ref')||
  '"},"atributos":{"material_sha256":"'||v_material_hash||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
    OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR octet_length(p_raiz)<>44
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
    OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version<>trunc(p_persona_version)
    OR p_perfil_version<>trunc(p_perfil_version)
 THEN RAISE EXCEPTION 'CT140: autorización denegada' USING ERRCODE='P1403'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT140: autorización denegada' USING ERRCODE='P1403'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.comunicaciones_expediente.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.comunicaciones.consultar'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_hash
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_hash
    OR d->'campos_permitidos' IS DISTINCT FROM
       '["antecedente_tipo","comunicacion_ref","estado","estado_respuesta","expediente_ref","llamamiento_ref","organizacion_ref","recibo_antecedente_ref","recibo_comunicacion_ref","registrada_en","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT140: autorización divergente' USING ERRCODE='P1403'; END IF;
 SELECT * INTO STRICT v_consumo FROM
  vec_autorizacion_atestada_v3.consumir_lista_comunicaciones_ct_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.efecto_ref IS DISTINCT FROM s->>'expediente_ref'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_hash
 THEN RAISE EXCEPTION 'CT140: consumo divergente' USING ERRCODE='P1403'; END IF;

 -- La existencia se comprueba después del consumo; una página vacía es
 -- normal, pero una referencia ajena o inexistente no revela diferencias.
 IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_alta e
  WHERE e.expediente_ref=s->>'expediente_ref'
    AND e.organizacion_ref=s->>'organizacion_ref') THEN
  RETURN jsonb_build_object('encontrado',false,'expediente_ref',s->>'expediente_ref',
   'comunicaciones','[]'::jsonb,'siguiente_cursor','');
 END IF;
 -- Todo elemento visible debe conservar el recibo CT54 y uno de los dos
 -- antecedentes propios que CT54/CT62 comprobaron al crearlo.
 IF EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
  WHERE x.organizacion_ref=s->>'organizacion_ref'
    AND x.expediente_ref=s->>'expediente_ref'
    AND (
      x.estado<>'registrada_localmente' OR x.version_resultante<>2
      OR x.recibo_json->>'ComunicacionRef' IS DISTINCT FROM x.comunicacion_ref
      OR coalesce(x.recibo_json->>'ReciboRef','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
      OR x.recibo_json->'Solicitud' IS DISTINCT FROM x.material_json->'solicitud'
      OR x.recibo_json->>'Estado' IS DISTINCT FROM x.estado
      OR x.recibo_json->>'RegistradaEn' IS NULL
      OR (x.recibo_json->>'RegistradaEn')::timestamptz IS DISTINCT FROM x.registrada_en
      OR (
       (NOT (x.material_json->'solicitud' ? 'TipoAntecedente') AND EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        WHERE e.clave_idempotencia=x.seleccion_clave AND e.situacion='confirmada'
          AND e.solicitud_json->>'organizacion_ref'=x.organizacion_ref
          AND e.solicitud_json->>'expediente_ref'=x.expediente_ref
          AND e.recibo_json->>'llamamiento_ref'=x.llamamiento_ref
          AND e.recibo_json->>'recibo_ref'=x.material_json->'solicitud'->>'PruebaEntregaRef'
          AND e.recibo_json->'propuesta_generada'='true'::jsonb
       ))
       OR (x.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
          AND EXISTS (
           SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
           JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
             ON e.clave_idempotencia=r.seleccion_clave AND e.clave_idempotencia=x.seleccion_clave
           JOIN vec_contratacion_temporal.comunicacion_llamamiento_local previa
             ON previa.comunicacion_ref=r.comunicacion_ref
             AND previa.seleccion_clave=x.seleccion_clave
             AND previa.organizacion_ref=r.organizacion_ref
             AND previa.expediente_ref=r.expediente_ref
             AND previa.llamamiento_ref=r.llamamiento_ref
           WHERE r.organizacion_ref=x.organizacion_ref
             AND r.expediente_ref=x.expediente_ref
             AND r.estado='confirmado'
             AND r.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')
             AND r.continuacion_clave IS NOT NULL
             AND r.continuacion_recibo->>'Estado'='confirmado'
             AND r.continuacion_recibo->>'ReciboRef'=x.material_json->'solicitud'->>'PruebaEntregaRef'
             AND r.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
             AND r.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
             AND r.continuacion_recibo->'Solicitud'->>'ResolucionRef'=r.resolucion_ref
             AND r.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=r.continuacion_clave::text
             AND r.continuacion_recibo->>'LlamamientoAnteriorRef'=r.llamamiento_ref
             AND r.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=x.llamamiento_ref
             AND r.llamamiento_ref<>x.llamamiento_ref
             AND e.situacion='confirmada'
             AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
             AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
             AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
             AND e.recibo_json->'propuesta_generada'='true'::jsonb
          ))
      ) IS NOT TRUE
    )
 ) THEN RAISE EXCEPTION 'CT140: comunicación inconsistente' USING ERRCODE='P1405'; END IF;
 -- La fila O6 nace de la selección ya confirmada. Una comunicación CT54
 -- existente la sustituye; dos selecciones sin aviso cierran por ambigüedad.
 SELECT count(*) INTO v_pendientes
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 WHERE e.situacion='confirmada'
   AND e.solicitud_json->>'organizacion_ref'=s->>'organizacion_ref'
   AND e.solicitud_json->>'expediente_ref'=s->>'expediente_ref'
   AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
                   WHERE x.seleccion_clave=e.clave_idempotencia);
 IF v_pendientes>1 THEN
  RAISE EXCEPTION 'CT231: selecciones pendientes ambiguas' USING ERRCODE='P1405';
 END IF;
 IF v_pendientes=1 THEN
  SELECT e.recibo_json,e.artefacto_canonico INTO STRICT v_pendiente_recibo,v_pendiente_artefacto
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  WHERE e.situacion='confirmada'
    AND e.solicitud_json->>'organizacion_ref'=s->>'organizacion_ref'
    AND e.solicitud_json->>'expediente_ref'=s->>'expediente_ref'
    AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
                    WHERE x.seleccion_clave=e.clave_idempotencia);
  v_pendiente_ref:=v_pendiente_recibo->>'llamamiento_ref';
  IF v_pendiente_artefacto IS NULL OR octet_length(v_pendiente_artefacto)>1048576
     OR v_pendiente_recibo->>'organizacion_ref' IS DISTINCT FROM s->>'organizacion_ref'
     OR v_pendiente_recibo->>'expediente_ref' IS DISTINCT FROM s->>'expediente_ref'
     OR v_pendiente_recibo->'propuesta_generada' IS DISTINCT FROM 'true'::jsonb
     OR v_pendiente_ref IS NULL OR v_pendiente_ref !~
        '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}$'
     OR coalesce(v_pendiente_recibo->>'recibo_ref','') !~
        '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR vec_contratacion_temporal.instante_utc_json_canonico_v2(
          v_pendiente_recibo->'confirmada_en',false) IS NOT TRUE
     OR vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
          v_pendiente_artefacto::jsonb,v_pendiente_artefacto,v_pendiente_recibo,
          vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
              v_pendiente_recibo)) IS NOT TRUE
     OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
                WHERE x.organizacion_ref=s->>'organizacion_ref'
                  AND x.expediente_ref=s->>'expediente_ref'
                  AND x.comunicacion_ref=v_pendiente_ref)
  THEN RAISE EXCEPTION 'CT231: selección pendiente inconsistente' USING ERRCODE='P1405'; END IF;
  v_pendiente_fecha:=(v_pendiente_recibo->>'confirmada_en')::timestamptz(6);
  v_fecha_texto:=to_char(v_pendiente_fecha AT TIME ZONE 'UTC',
                        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
  v_pendiente_fila:=jsonb_build_object(
   'organizacion_ref',s->>'organizacion_ref','expediente_ref',s->>'expediente_ref',
   'llamamiento_ref',v_pendiente_ref,'estado','seleccion_confirmada_pendiente_aviso',
   'registrada_en',v_fecha_texto,'version',1,
   'antecedente_tipo','seleccion_confirmada',
   'recibo_antecedente_ref',v_pendiente_recibo->>'recibo_ref');
 END IF;
 FOR v_fila IN
  SELECT x.comunicacion_ref FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
  WHERE x.organizacion_ref=s->>'organizacion_ref'
    AND x.expediente_ref=s->>'expediente_ref'
  ORDER BY x.registrada_en,x.comunicacion_ref
 LOOP
  v_total:=v_total+1;
  v_estados:=v_estados||v_fila.comunicacion_ref||'='||
   vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(v_fila.comunicacion_ref)||chr(10);
 END LOOP;
 IF v_pendiente_fila IS NOT NULL THEN
  v_total:=v_total+1;
  v_estados:=v_estados||v_pendiente_ref||'=seleccion_confirmada_pendiente_aviso'||chr(10);
 END IF;
 -- CT54 y CT56 son inmutables. El ancla detecta altas o cambios del estado
 -- semántico entre páginas y obliga al cliente a reiniciar el recorrido.
 v_ancla:=encode(sha256(convert_to(
  (s->>'organizacion_ref')||chr(10)||(s->>'expediente_ref')||chr(10)||
  v_total::text||chr(10)||v_estados,'UTF8')),'hex');
 IF s->>'cursor'<>'' THEN
  IF right(s->>'cursor',64) IS DISTINCT FROM v_ancla
  THEN RAISE EXCEPTION 'CT140: página caducada' USING ERRCODE='P1405'; END IF;
  v_cursor_ref:=left(s->>'cursor',length(s->>'cursor')-65);
  IF v_pendiente_ref IS NOT NULL AND v_cursor_ref=v_pendiente_ref THEN
   v_cursor_fecha:=v_pendiente_fecha;
  ELSE
   SELECT x.registrada_en,x.comunicacion_ref INTO v_cursor_fecha,v_cursor_ref
   FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
   WHERE x.comunicacion_ref=v_cursor_ref
     AND x.organizacion_ref=s->>'organizacion_ref'
     AND x.expediente_ref=s->>'expediente_ref';
   IF NOT FOUND THEN
    RETURN jsonb_build_object('encontrado',false,'expediente_ref',s->>'expediente_ref',
     'comunicaciones','[]'::jsonb,'siguiente_cursor','');
   END IF;
  END IF;
 END IF;
 WITH filas AS (
  SELECT x.registrada_en,x.comunicacion_ref AS clave,
   jsonb_build_object(
    'organizacion_ref',x.organizacion_ref,'expediente_ref',x.expediente_ref,
    'llamamiento_ref',x.llamamiento_ref,
    'comunicacion_ref',x.comunicacion_ref,'version',x.version_resultante::bigint,
    'estado',x.estado,
    'estado_respuesta',vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(x.comunicacion_ref),
    'recibo_comunicacion_ref',x.recibo_json->>'ReciboRef',
    'antecedente_tipo',CASE WHEN x.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
      THEN 'continuacion_confirmada' ELSE 'seleccion_confirmada' END,
    'recibo_antecedente_ref',x.material_json->'solicitud'->>'PruebaEntregaRef',
    'registrada_en',to_char(x.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) AS contenido
  FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
  WHERE x.organizacion_ref=s->>'organizacion_ref'
    AND x.expediente_ref=s->>'expediente_ref'
  UNION ALL
  SELECT v_pendiente_fecha,v_pendiente_ref,v_pendiente_fila
  WHERE v_pendiente_fila IS NOT NULL
 ), orden AS (
  SELECT * FROM filas
  WHERE v_cursor_ref IS NULL OR (registrada_en,clave)>(v_cursor_fecha,v_cursor_ref)
  ORDER BY registrada_en,clave LIMIT v_limite+1
 ), pagina AS (
  SELECT * FROM orden ORDER BY registrada_en,clave LIMIT v_limite
 )
 SELECT coalesce(jsonb_agg(p.contenido ORDER BY p.registrada_en,p.clave),'[]'::jsonb),
   (SELECT count(*) FROM orden),
   coalesce(array_agg(p.clave ORDER BY p.registrada_en,p.clave),ARRAY[]::text[])
 INTO v_items,v_cantidad,v_claves FROM pagina p;
 IF v_cantidad>v_limite THEN
  v_siguiente:=v_claves[v_limite];
  IF length(v_siguiente)>95
  THEN RAISE EXCEPTION 'CT140: cursor no representable' USING ERRCODE='P1405'; END IF;
  v_siguiente:=v_siguiente||'#'||v_ancla;
 END IF;
 RETURN jsonb_build_object('encontrado',true,'expediente_ref',s->>'expediente_ref',
  'comunicaciones',v_items,'siguiente_cursor',v_siguiente);
EXCEPTION
 WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
  RAISE EXCEPTION 'CT140: consulta transitoria' USING ERRCODE='P1405';
END $function$;
DO $post_ct231$
DECLARE p pg_catalog.pg_proc%ROWTYPE; actual text;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'] THEN
  RAISE EXCEPTION 'CT231: metadatos de salida divergentes' USING ERRCODE='55000';
 END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF actual IS DISTINCT FROM '2037b0497f2c5bec4db4940f4079d1c44976fb774cff67dad4ca3063defa80f9' OR
    pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
    IS DISTINCT FROM '55ff31d6f33afa8f9ca386def26b99ebbaa00287fc55bcd05564d81d63f05071' THEN
  RAISE EXCEPTION 'CT231: postimagen divergente' USING ERRCODE='55000';
 END IF;
END $post_ct231$;
COMMIT;
