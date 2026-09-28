\set ON_ERROR_STOP on
-- CT140: lista paginada de comunicaciones locales propias de CT54.
-- AD3-105 debe estar instalada antes. La lista no reconstruye recibos del POST.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000140',0));

DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.comunicacion_llamamiento_local') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_alta') IS NULL
    OR to_regclass('vec_contratacion_temporal.respuesta_recibida_rrhh') IS NULL
    OR to_regclass('vec_contratacion_temporal.historia_respuesta_recibida_rrhh') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_lista_comunicaciones_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT140: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Misma clave semántica que CT138: organización + llamamiento + selección
-- seudonimizada en el recibo CT46. El estado no revela actor ni respuesta.
CREATE FUNCTION vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(
 p_comunicacion_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY INVOKER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $estado$
DECLARE
 v_comunicacion record; v_seleccion_ref text; v_total bigint; v_validas boolean;
BEGIN
 SELECT c.organizacion_ref,c.expediente_ref,c.llamamiento_ref,c.seleccion_clave,
  e.situacion,e.solicitud_json,e.recibo_json
 INTO v_comunicacion
 FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
 JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   ON e.clave_idempotencia=c.seleccion_clave
 WHERE c.comunicacion_ref=p_comunicacion_ref;
 IF NOT FOUND OR v_comunicacion.situacion IS DISTINCT FROM 'confirmada'
    OR v_comunicacion.solicitud_json->>'organizacion_ref' IS DISTINCT FROM v_comunicacion.organizacion_ref
    OR v_comunicacion.solicitud_json->>'expediente_ref' IS DISTINCT FROM v_comunicacion.expediente_ref
    OR v_comunicacion.recibo_json->>'organizacion_ref' IS DISTINCT FROM v_comunicacion.organizacion_ref
    OR v_comunicacion.recibo_json->>'expediente_ref' IS DISTINCT FROM v_comunicacion.expediente_ref THEN
  RAISE EXCEPTION 'CT140: selección inconsistente' USING ERRCODE='P1405';
 END IF;
 v_seleccion_ref:=v_comunicacion.recibo_json->>'seleccion_ref';
 IF v_seleccion_ref IS NULL
    OR v_seleccion_ref !~ '^hmac-sha256:vec[.]contratacion-temporal[.]seleccion/v[1-9][0-9]*:[0-9a-f]{64}$'
    OR right(v_seleccion_ref,64)=repeat('0',64) THEN
  RAISE EXCEPTION 'CT140: selección sin seudónimo' USING ERRCODE='P1405';
 END IF;
 -- Si existe una respuesta del mismo llamamiento sin pseudónimo válido,
 -- no podemos concluir que esta selección carezca de respuesta.
 IF EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
  LEFT JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
    ON e.clave_idempotencia=r.seleccion_clave
  WHERE r.organizacion_ref=v_comunicacion.organizacion_ref
    AND r.llamamiento_ref=v_comunicacion.llamamiento_ref
    AND (e.clave_idempotencia IS NULL
      OR coalesce(e.recibo_json->>'seleccion_ref','') !~
       '^hmac-sha256:vec[.]contratacion-temporal[.]seleccion/v[1-9][0-9]*:[0-9a-f]{64}$'
      OR right(coalesce(e.recibo_json->>'seleccion_ref',''),64)=repeat('0',64))
 ) THEN RAISE EXCEPTION 'CT140: respuesta sin selección acreditada' USING ERRCODE='P1405'; END IF;

 -- La fila CT56 apunta a una comunicación concreta, pero CT138 deduplica
 -- también si la misma selección respondió en otra comunicación.
 SELECT count(*),coalesce(bool_and(coalesce((
   r.estado='registrada_por_rrhh'
   AND r.version_comunicacion=2
   AND e.situacion='confirmada'
   AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
   AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
   AND e.recibo_json->>'organizacion_ref'=r.organizacion_ref
   AND e.recibo_json->>'expediente_ref'=r.expediente_ref
   AND (e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
     OR c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada')
   AND c.comunicacion_ref IS NOT NULL
   AND c.organizacion_ref=r.organizacion_ref
   AND c.expediente_ref=r.expediente_ref
   AND c.llamamiento_ref=r.llamamiento_ref
   AND c.seleccion_clave=r.seleccion_clave
   AND c.estado='registrada_localmente'
   AND c.version_resultante=2
   AND c.recibo_json->>'ComunicacionRef'=c.comunicacion_ref
   AND c.recibo_json->>'Estado'=c.estado
   AND c.recibo_json->'Solicitud'=c.material_json->'solicitud'
   AND h.justificante_ref=r.justificante_ref
   AND h.actor_ref=r.actor_ref AND h.perfil_ref=r.perfil_ref
   AND r.recibo_json->>'JustificanteRef'=r.justificante_ref
   AND r.recibo_json->>'ReciboRef'=r.recibo_ref
   AND r.recibo_json->>'AuditoriaRef'=h.auditoria_ref
   AND r.recibo_json->>'Estado'=r.estado
   AND r.recibo_json->>'RegistradaEn'=to_char(r.registrada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
   AND r.recibo_json->'Solicitud'=r.material_json
   AND r.material_json->>'OrganizacionRef'=r.organizacion_ref
   AND r.material_json->>'ExpedienteRef'=r.expediente_ref
   AND r.material_json->>'LlamamientoRef'=r.llamamiento_ref
   AND r.material_json->>'ComunicacionRef'=r.comunicacion_ref
   AND r.material_json->>'Respuesta'=r.respuesta
  ),false)),true)
 INTO v_total,v_validas
 FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
 JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   ON e.clave_idempotencia=r.seleccion_clave
 LEFT JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
   ON c.comunicacion_ref=r.comunicacion_ref
 LEFT JOIN vec_contratacion_temporal.historia_respuesta_recibida_rrhh h
   ON h.justificante_ref=r.justificante_ref
 WHERE r.organizacion_ref=v_comunicacion.organizacion_ref
   AND r.llamamiento_ref=v_comunicacion.llamamiento_ref
   AND e.recibo_json->>'seleccion_ref'=v_seleccion_ref;
 IF v_total>1 OR v_validas IS NOT TRUE THEN
  RAISE EXCEPTION 'CT140: respuestas incompatibles' USING ERRCODE='P1405';
 END IF;
 -- El índice CT56 impide una segunda respuesta sobre la misma comunicación.
 -- Si su selección no coincide con la clave semántica, no responder ausente.
 IF EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
  JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
    ON e.clave_idempotencia=r.seleccion_clave
  WHERE r.organizacion_ref=v_comunicacion.organizacion_ref
    AND r.comunicacion_ref=p_comunicacion_ref
    AND (r.llamamiento_ref IS DISTINCT FROM v_comunicacion.llamamiento_ref
      OR e.recibo_json->>'seleccion_ref' IS DISTINCT FROM v_seleccion_ref)
 ) THEN RAISE EXCEPTION 'CT140: respuesta directa incompatible' USING ERRCODE='P1405'; END IF;
 RETURN CASE WHEN v_total=1 THEN 'registrada' ELSE 'sin_respuesta' END;
END $estado$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(text)
 FROM PUBLIC,vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 s jsonb; d jsonb; c jsonb; v_hash text; v_material_hash text; v_consumo record;
 v_cursor_fecha timestamptz(6); v_cursor_ref text; v_items jsonb; v_cantidad integer;
 v_total bigint:=0; v_ancla text; v_estados text:=''; v_fila record;
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
      OR NOT (
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
             AND r.estado='confirmado' AND r.solicitud_json->>'Respuesta'='renuncia'
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
      )
    )
 ) THEN RAISE EXCEPTION 'CT140: comunicación inconsistente' USING ERRCODE='P1405'; END IF;
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
 -- CT54 y CT56 son inmutables. El ancla detecta altas o cambios del estado
 -- semántico entre páginas y obliga al cliente a reiniciar el recorrido.
 v_ancla:=encode(sha256(convert_to(
  (s->>'organizacion_ref')||chr(10)||(s->>'expediente_ref')||chr(10)||
  v_total::text||chr(10)||v_estados,'UTF8')),'hex');
 IF s->>'cursor'<>'' THEN
  IF right(s->>'cursor',64) IS DISTINCT FROM v_ancla
  THEN RAISE EXCEPTION 'CT140: página caducada' USING ERRCODE='P1405'; END IF;
  v_cursor_ref:=left(s->>'cursor',length(s->>'cursor')-65);
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
 WITH orden AS (
  SELECT x.organizacion_ref,x.expediente_ref,x.llamamiento_ref,x.comunicacion_ref,
    x.version_resultante::bigint AS version,x.estado,x.registrada_en,
    vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(x.comunicacion_ref) AS estado_respuesta,
    x.recibo_json->>'ReciboRef' AS recibo_comunicacion_ref,
    CASE WHEN x.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
      THEN 'continuacion_confirmada' ELSE 'seleccion_confirmada' END AS antecedente_tipo,
    x.material_json->'solicitud'->>'PruebaEntregaRef' AS recibo_antecedente_ref
  FROM vec_contratacion_temporal.comunicacion_llamamiento_local x
  WHERE x.organizacion_ref=s->>'organizacion_ref'
    AND x.expediente_ref=s->>'expediente_ref'
    AND (v_cursor_ref IS NULL OR (x.registrada_en,x.comunicacion_ref)>(v_cursor_fecha,v_cursor_ref))
  ORDER BY x.registrada_en,x.comunicacion_ref
  LIMIT v_limite+1
 ), pagina AS (
  SELECT * FROM orden ORDER BY registrada_en,comunicacion_ref LIMIT v_limite
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'organizacion_ref',p.organizacion_ref,'expediente_ref',p.expediente_ref,
   'llamamiento_ref',p.llamamiento_ref,
   'comunicacion_ref',p.comunicacion_ref,'version',p.version,'estado',p.estado,
   'estado_respuesta',p.estado_respuesta,
   'recibo_comunicacion_ref',p.recibo_comunicacion_ref,
   'antecedente_tipo',p.antecedente_tipo,
   'recibo_antecedente_ref',p.recibo_antecedente_ref,
   'registrada_en',to_char(p.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
   ORDER BY p.registrada_en,p.comunicacion_ref),'[]'::jsonb),
   (SELECT count(*) FROM orden)
 INTO v_items,v_cantidad FROM pagina p;
 IF v_cantidad>v_limite THEN
  v_siguiente:=v_items->(v_limite-1)->>'comunicacion_ref';
  IF length(v_siguiente)>95
  THEN RAISE EXCEPTION 'CT140: cursor no representable' USING ERRCODE='P1405'; END IF;
  v_siguiente:=v_siguiente||'#'||v_ancla;
 END IF;
 RETURN jsonb_build_object('encontrado',true,'expediente_ref',s->>'expediente_ref',
  'comunicaciones',v_items,'siguiente_cursor',v_siguiente);
EXCEPTION
 WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
  RAISE EXCEPTION 'CT140: consulta transitoria' USING ERRCODE='P1405';
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(
  text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
  TO vec_contratacion_temporal_ejecutor;
 IF has_function_privilege('public',f,'EXECUTE')
    OR has_function_privilege('public','vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(text)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.estado_respuesta_comunicacion_ct140(text)','EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT140: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
