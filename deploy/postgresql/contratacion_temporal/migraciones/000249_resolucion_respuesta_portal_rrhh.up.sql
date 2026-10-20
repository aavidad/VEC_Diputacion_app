\set ON_ERROR_STOP on
-- CT249 / F1-125. Resolución RRHH de aceptación firme de Mi Bolsa, sin
-- atribuir al portal una comunicación CT54 ni un justificante de correo CT56.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000249',0));
DO $pre_ct249$
DECLARE p pg_proc%ROWTYPE; t pg_class%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'CT249: PostgreSQL o propietario incompatible' USING ERRCODE='55000';
 END IF;
 SELECT * INTO t FROM pg_class WHERE oid=to_regclass('vec_contratacion_temporal.resolucion_manual_respuesta_rrhh');
 IF NOT FOUND OR t.relowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR t.relrowsecurity IS NOT TRUE OR t.relforcerowsecurity IS NOT TRUE
    OR to_regprocedure('vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid=t.oid AND a.attname='comunicacion_ref' AND a.attnotnull)
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid=t.oid AND a.attname='fuente_respuesta' AND NOT a.attisdropped) THEN
  RAISE EXCEPTION 'CT249: tabla o fuente incompatible' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
  GROUP BY r.organizacion_ref,r.expediente_ref,r.llamamiento_ref HAVING count(*)>1) THEN
  RAISE EXCEPTION 'CT249: resoluciones previas duplicadas' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint c
   WHERE c.conrelid=to_regclass('vec_contratacion_temporal.comunicacion_llamamiento_local')
     AND c.contype='u' AND pg_get_constraintdef(c.oid)=
       'UNIQUE (organizacion_ref, expediente_ref, llamamiento_ref)') THEN
  RAISE EXCEPTION 'CT249: unicidad CT54 ausente' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM (VALUES
   ('resolucion_manual_estado_plazo_check','3f3dd731ee89963572fd27edbfa154a08a7702b8564a3d967e38ecdf04a9c5c5'),
   ('resolucion_manual_plazo_coherente_check','1518eed558e60005d8f1474e2b28904a04ec79c400293db9ed289a9a6e6f2c50'),
   ('resolucion_manual_respuesta_rrhh_version_resultante_check','ce4489a677586dcbea2baaca5b6ff272bd63784367d06d03caa30afa8493595e'),
   ('resolucion_manual_respuesta_rrhh_comunicacion_ref_fkey',
    -- SHA-256 de pg_get_constraintdef: preimagen pública de la restricción instalada, sin credenciales.
    '32c0d6b1c2dd712c35ad53d475651c8777f623d4c74fcbd4958b0513092a9aa0'),
   ('resolucion_manual_respuesta_rrhh_justificante_ref_fkey',
    -- SHA-256 de pg_get_constraintdef: preimagen pública de la restricción instalada, sin credenciales.
    'fe76153bfaf8d112581c2931cfa99fd0e2a91fd238f0133abaaaed2253797030'),
   ('resolucion_manual_respuesta_rrhh_seleccion_clave_fkey',
    -- SHA-256 de pg_get_constraintdef: preimagen pública de la restricción instalada, sin credenciales.
    'e9142913f884211f1ffaae310f844fb7d65b746badd50a81d57c41bf9d4d0b98')) esperado(nombre,huella)
   LEFT JOIN pg_constraint c ON c.conrelid=t.oid AND c.conname=esperado.nombre
   WHERE c.oid IS NULL OR encode(sha256(convert_to(pg_get_constraintdef(c.oid),'UTF8')),'hex')<>esperado.huella) THEN
  RAISE EXCEPTION 'CT249: restricciones de preimagen divergentes' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')<>'69f02d5593fb259c32fa1aa102fab69d902f5752885fe1fae5b865a52007063b' THEN
  RAISE EXCEPTION 'CT249: CT58 instalada divergente' USING ERRCODE='55000'; END IF;
 IF has_function_privilege(current_user,to_regprocedure(
    'vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)'),'EXECUTE') IS NOT TRUE
    OR has_function_privilege(current_user,to_regprocedure(
    'vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT249: consumidores nominales ausentes' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc q WHERE q.oid=to_regprocedure(
    'vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)')
    AND q.proowner='vec_bolsa_llamamientos_propietario'::regrole
    AND q.prosecdef AND q.provolatile='v'
    AND q.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_contratacion_temporal_propietario=X/vec_bolsa_llamamientos_propietario}'
    AND q.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    AND encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex')='9ab544fd6903c5c57ebea6e8476bd2cb8f10d1238f52ba133e8363b16f4dc5cb')
    OR NOT EXISTS (SELECT 1 FROM pg_proc q WHERE q.oid=to_regprocedure(
    'vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
    AND q.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
    AND q.prosecdef AND q.provolatile='v'
    AND q.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    AND q.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    AND encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex')='7a79847bfd82079d77a513b8d62fa76acf7942123c923fb29620c4896500438b') THEN
  RAISE EXCEPTION 'CT249: B113 o AD17 divergente' USING ERRCODE='55000'; END IF;
END $pre_ct249$;

ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
 ADD COLUMN fuente_respuesta text NOT NULL DEFAULT 'correo_rrhh',
 ADD COLUMN respuesta_portal_ref text,
 ADD COLUMN recibo_respuesta_portal_ref text,
 ADD COLUMN participacion_ref text,
 ALTER COLUMN comunicacion_ref DROP NOT NULL,
 DROP CONSTRAINT resolucion_manual_estado_plazo_check,
 DROP CONSTRAINT resolucion_manual_plazo_coherente_check,
 DROP CONSTRAINT resolucion_manual_respuesta_rrhh_version_resultante_check,
 ADD CONSTRAINT resolucion_manual_estado_plazo_check
  CHECK (estado_plazo IN ('vigente','expirado','revision_manual_portal')),
 ADD CONSTRAINT resolucion_manual_plazo_coherente_check CHECK (
   (fuente_respuesta='portal' AND estado_plazo='revision_manual_portal'
    AND justificante_ref IS NULL AND contacto_ref IS NULL AND causa_justificada_ref IS NULL
    AND respuesta_fuera_de_plazo IS FALSE)
   OR (fuente_respuesta='correo_rrhh' AND estado_plazo IN ('vigente','expirado') AND (
    (solicitud_json->>'Respuesta'='expiracion_gobernada')=(estado_plazo='expirado')
    AND (solicitud_json->>'Respuesta'='expiracion_gobernada')=(justificante_ref IS NULL)
    AND (solicitud_json->>'Respuesta'<>'expiracion_gobernada' OR contacto_ref IS NOT NULL)
    AND (causa_justificada_ref IS NULL OR (respuesta_fuera_de_plazo AND contacto_ref IS NOT NULL))
    AND (NOT respuesta_fuera_de_plazo OR contacto_ref IS NOT NULL)) IS TRUE)),
 ADD CONSTRAINT resolucion_manual_respuesta_rrhh_version_resultante_check
  CHECK ((fuente_respuesta='correo_rrhh' AND version_resultante=3)
      OR (fuente_respuesta='portal' AND version_resultante=1)),
 ADD CONSTRAINT resolucion_manual_fuente_disjunta_check CHECK ((
  (fuente_respuesta='correo_rrhh' AND comunicacion_ref IS NOT NULL
   AND respuesta_portal_ref IS NULL AND recibo_respuesta_portal_ref IS NULL AND participacion_ref IS NULL)
  OR (fuente_respuesta='portal' AND comunicacion_ref IS NULL AND justificante_ref IS NULL
   AND solicitud_json->>'Respuesta'='aceptacion'
   AND respuesta_portal_ref ~ '^respuesta-portal:[0-9a-f]{64}$'
   AND recibo_respuesta_portal_ref ~ '^recibo:respuesta-portal:[0-9a-f]{64}$'
   AND participacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')) IS TRUE),
 ADD CONSTRAINT resolucion_manual_respuesta_portal_unica UNIQUE (respuesta_portal_ref),
 ADD CONSTRAINT resolucion_manual_recibo_portal_unico UNIQUE (recibo_respuesta_portal_ref),
 ADD CONSTRAINT resolucion_manual_llamamiento_unico UNIQUE (organizacion_ref,expediente_ref,llamamiento_ref);

CREATE FUNCTION vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; s jsonb; p jsonb; d jsonb; selector jsonb; anclaje jsonb;
 consumo record; previo record; e record; vinculo record; agregado jsonb; bolsa jsonb;
 huella text; contexto_h text; tipo_politica text; n bigint;
 ahora timestamptz(6); respondida timestamptz(6); resolucion text; recibo text; evaluacion text; salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT249: sesión denegada' USING ERRCODE='P2493'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
  RAISE EXCEPTION 'CT249: material inválido' USING ERRCODE='P2490'; END IF;
 BEGIN m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT249: JSON inválido' USING ERRCODE='P2490'; END;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['Solicitud','Politica']) IS NOT TRUE
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',ARRAY[
      'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','BolsaRef','LlamamientoRef',
      'ReciboEmisionRef','ParticipacionRef','ReciboRespuestaRef','Respuesta',
      'RevisionRespuestaRRHH','RevisionPlazoRRHH','CriterioValidacionRef']) IS NOT TRUE
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Politica',ARRAY[
      'Referencia','Version','HuellaSHA256']) IS NOT TRUE
    OR (SELECT count(*) FROM json_each(p_material::json))<>2
    OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>12
    OR (SELECT count(*) FROM json_each((p_material::json)->'Politica'))<>3 THEN
  RAISE EXCEPTION 'CT249: claves inválidas' USING ERRCODE='P2490'; END IF;
 s:=m->'Solicitud'; p:=m->'Politica';
 IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
    OR s->>'ClaveIdempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
    OR s->>'Respuesta' IS DISTINCT FROM 'aceptacion'
    OR s->'RevisionRespuestaRRHH' IS DISTINCT FROM 'true'::jsonb
    OR s->'RevisionPlazoRRHH' IS DISTINCT FROM 'true'::jsonb THEN
  RAISE EXCEPTION 'CT249: solicitud inválida' USING ERRCODE='P2490'; END IF;
 IF EXISTS (SELECT 1 FROM unnest(ARRAY['OrganizacionRef','ExpedienteRef','BolsaRef','LlamamientoRef',
     'ReciboEmisionRef','ParticipacionRef','ReciboRespuestaRef','CriterioValidacionRef']) campo
    WHERE jsonb_typeof(s->campo) IS DISTINCT FROM 'string'
       OR s->>campo !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
    OR p->>'Referencia' IS DISTINCT FROM s->>'CriterioValidacionRef'
    OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
    OR p->>'Version' !~ '^[1-9][0-9]{0,15}$'
    OR jsonb_typeof(p->'HuellaSHA256') IS DISTINCT FROM 'string'
    OR p->>'HuellaSHA256' !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CT249: referencias inválidas' USING ERRCODE='P2490'; END IF;
 IF s->>'ReciboRespuestaRef' !~ '^recibo:respuesta-portal:[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CT249: recibo portal inválido' USING ERRCODE='P2490'; END IF;
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to(
  '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||huella||'"}}','UTF8')),'hex');
 IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'CT249: autorización denegada' USING ERRCODE='P2493'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT249: autorización denegada' USING ERRCODE='P2493'; END;
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'resolucion_manual_respuesta_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'CT249: decisión divergente' USING ERRCODE='P2493'; END IF;
 BEGIN
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0583' THEN
  RAISE EXCEPTION 'CT249: consumo denegado' USING ERRCODE='P2493'; END;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'CT249: consumo divergente' USING ERRCODE='P2493'; END IF;
 tipo_politica:=vec_contratacion_temporal.politica_llamamiento_admitida_v1(
  p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256','aceptacion');
 IF tipo_politica IS DISTINCT FROM 'referencia_exacta' THEN
  RAISE EXCEPTION 'CT249: política de revisión no admitida' USING ERRCODE='P2490'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('ct249:'||(s->>'OrganizacionRef')||':'||(s->>'LlamamientoRef'),0));
 SELECT * INTO previo FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
  WHERE r.organizacion_ref=s->>'OrganizacionRef' AND r.clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
 IF FOUND THEN
  IF previo.fuente_respuesta IS DISTINCT FROM 'portal' OR previo.material IS DISTINCT FROM p_material
     OR previo.actor_ref IS DISTINCT FROM d->>'principal_id'
     OR previo.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
     OR previo.recibo_json->>'ResolucionRef' IS DISTINCT FROM previo.resolucion_ref
     OR previo.recibo_json->>'ReciboLocalRef' IS DISTINCT FROM previo.recibo_ref
     OR previo.recibo_json->>'AuditoriaRef' IS DISTINCT FROM previo.auditoria_ref
     OR previo.recibo_json->'Solicitud' IS DISTINCT FROM previo.solicitud_json
     OR previo.recibo_json#>>'{RespuestaPortal,RespuestaRef}' IS DISTINCT FROM previo.respuesta_portal_ref
     OR previo.recibo_json#>>'{RespuestaPortal,ReciboRespuestaRef}' IS DISTINCT FROM previo.recibo_respuesta_portal_ref
     OR previo.recibo_json#>>'{RespuestaPortal,ParticipacionRef}' IS DISTINCT FROM previo.participacion_ref
     OR previo.recibo_json->>'ResueltaEn' IS DISTINCT FROM
        to_char(previo.resuelta_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
     OR previo.recibo_json->>'Estado' IS DISTINCT FROM 'confirmado' THEN
   RAISE EXCEPTION 'CT249: clave usada con otro contenido' USING ERRCODE='P2491'; END IF;
  RETURN previo.recibo_json||jsonb_build_object('Estado','replay_confirmado');
 END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
     WHERE r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef') THEN
  RAISE EXCEPTION 'CT249: llamamiento ya resuelto' USING ERRCODE='P2491'; END IF;
 SELECT * INTO vinculo FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
  WHERE v.organizacion_ref=s->>'OrganizacionRef' AND v.expediente_ref=s->>'ExpedienteRef'
    AND v.bolsa_ref=s->>'BolsaRef' AND v.llamamiento_ref=s->>'LlamamientoRef'
    AND v.recibo_emision_ref=s->>'ReciboEmisionRef';
 IF NOT FOUND THEN RAISE EXCEPTION 'CT249: vínculo no acreditado' USING ERRCODE='P2492'; END IF;
 SELECT count(*) INTO n FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
  WHERE x.situacion='confirmada' AND x.solicitud_json->>'organizacion_ref'=s->>'OrganizacionRef'
    AND x.solicitud_json->>'expediente_ref'=s->>'ExpedienteRef'
    AND x.recibo_json->>'llamamiento_ref'=s->>'LlamamientoRef'
    AND x.recibo_json->>'recibo_ref'=s->>'ReciboEmisionRef';
 IF n<>1 THEN RAISE EXCEPTION 'CT249: selección ambigua o ausente' USING ERRCODE='P2492'; END IF;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
  WHERE x.situacion='confirmada' AND x.solicitud_json->>'organizacion_ref'=s->>'OrganizacionRef'
    AND x.solicitud_json->>'expediente_ref'=s->>'ExpedienteRef'
    AND x.recibo_json->>'llamamiento_ref'=s->>'LlamamientoRef'
    AND x.recibo_json->>'recibo_ref'=s->>'ReciboEmisionRef';
 SELECT v.agregado_json INTO agregado FROM vec_contratacion_temporal.expediente_version_integral v
  WHERE v.expediente_ref=s->>'ExpedienteRef' AND v.version=(e.solicitud_json->>'version_expediente')::numeric;
 IF NOT FOUND OR e.recibo_json->'propuesta_generada' IS DISTINCT FROM 'true'::jsonb
    OR e.recibo_json#>>'{procedencia,sin_retencion_declarada}' IS DISTINCT FROM 'true'
    OR e.recibo_json->'version_expediente' IS DISTINCT FROM e.solicitud_json->'version_expediente'
    OR e.recibo_json#>>'{bolsa,referencia}' IS DISTINCT FROM s->>'BolsaRef'
    OR agregado#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR agregado#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR agregado#>>'{analisis,categoria_ref}' IS NULL
    OR agregado#>>'{asignacion,unidad_ref}' IS NULL
    OR e.recibo_json#>>'{necesidad,referencia}' IS NULL THEN
  RAISE EXCEPTION 'CT249: selección nativa divergente' USING ERRCODE='P2492'; END IF;
 selector:=jsonb_build_object('organizacion_ref',s->>'OrganizacionRef','expediente_ref',s->>'ExpedienteRef',
  'clave_ct',e.clave_idempotencia::text,'unidad_ref',agregado#>>'{asignacion,unidad_ref}',
  'categoria_ref',agregado#>>'{analisis,categoria_ref}',
  'necesidad_ref',e.recibo_json#>>'{necesidad,referencia}',
  'bolsa_ref',s->>'BolsaRef','llamamiento_ref',s->>'LlamamientoRef',
  'recibo_emision_ref',s->>'ReciboEmisionRef','recibo_respuesta_ref',s->>'ReciboRespuestaRef');
 bolsa:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(selector);
 anclaje:=bolsa->'anclaje';
 IF bolsa->>'estado' NOT IN ('pendiente','acreditado') OR jsonb_typeof(anclaje) IS DISTINCT FROM 'object'
    OR anclaje->>'organizacion_ref' IS DISTINCT FROM s->>'OrganizacionRef'
    OR anclaje->>'expediente_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR anclaje->>'llamamiento_ref' IS DISTINCT FROM s->>'LlamamientoRef'
    OR anclaje->>'recibo_emision_ref' IS DISTINCT FROM s->>'ReciboEmisionRef'
    OR anclaje->>'recibo_respuesta_ref' IS DISTINCT FROM s->>'ReciboRespuestaRef'
    OR anclaje->>'participacion_ref' IS DISTINCT FROM s->>'ParticipacionRef'
    OR anclaje->>'clave_ct' IS DISTINCT FROM e.clave_idempotencia::text
    OR anclaje->>'version_expediente' IS DISTINCT FROM e.recibo_json->>'version_expediente'
    OR anclaje->>'respuesta_ref' !~ '^respuesta-portal:[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CT249: respuesta firme no acreditada' USING ERRCODE='P2492'; END IF;
 BEGIN respondida:=(anclaje->>'respondida_en')::timestamptz(6);
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT249: fecha de respuesta inválida' USING ERRCODE='P2492'; END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF respondida>ahora THEN RAISE EXCEPTION 'CT249: respuesta posterior al acto' USING ERRCODE='P2492'; END IF;
 resolucion:='resolucion:'||gen_random_uuid()::text;
 recibo:='recibo:'||gen_random_uuid()::text;
 evaluacion:='evaluacion:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('Solicitud',s,'Politica',p,'FuenteRespuesta','portal',
  'RespuestaPortal',jsonb_build_object('ParticipacionRef',s->>'ParticipacionRef',
    'RespuestaRef',anclaje->>'respuesta_ref','ReciboRespuestaRef',s->>'ReciboRespuestaRef',
    'RespondidaEn',anclaje->>'respondida_en'),
  'EvaluacionPlazoRef',evaluacion,'EstadoPlazo','revision_manual_portal',
  'ResolucionRef',resolucion,'ReciboLocalRef',recibo,'AuditoriaRef',consumo.auditoria_ref,
  'IntencionSiguiente','{}'::jsonb,'VersionResultante',1,
  'ResueltaEn',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'Estado','confirmado');
 INSERT INTO vec_contratacion_temporal.resolucion_manual_respuesta_rrhh(
  resolucion_ref,organizacion_ref,expediente_ref,llamamiento_ref,
  seleccion_clave,clave_idempotencia,actor_ref,perfil_ref,
  revision_respuesta_rrhh,revision_plazo_rrhh,politica_ref,politica_version,politica_sha256,
  evaluacion_plazo_ref,estado_plazo,version_resultante,material,material_json,material_huella_sha256,
  solicitud_json,auditoria_ref,decision_ref,consumo_huella_sha256,evidencia_huella_sha256,
  recibo_ref,recibo_json,estado,resuelta_en,fuente_respuesta,respuesta_portal_ref,
  recibo_respuesta_portal_ref,participacion_ref)
 VALUES(resolucion,s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'LlamamientoRef',
  e.clave_idempotencia,(s->>'ClaveIdempotencia')::uuid,d->>'principal_id',d->>'perfil_activo_ref',
  true,true,p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256',
  evaluacion,'revision_manual_portal',1,p_material,m,huella,s,consumo.auditoria_ref,
  consumo.decision_ref,consumo.consumo_huella_sha256,encode(sha256(p_evidencia),'hex'),
  recibo,salida,'confirmado',ahora,'portal',anclaje->>'respuesta_ref',
  s->>'ReciboRespuestaRef',s->>'ParticipacionRef');
 RETURN salida;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'CT249: resolución en conflicto' USING ERRCODE='P2491';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

DO $pre_lector_ct249$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
   OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
   OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'd5c812acbb329273947de7a4b7f2ed400a82ba7c94f9c3e749772247fbad1c1d' THEN
  RAISE EXCEPTION 'CT249: lector instalado divergente' USING ERRCODE='55000'; END IF;
END $pre_lector_ct249$;

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
DECLARE d record; actual record; vinculos jsonb; sugeridas jsonb; origenes jsonb; n integer;
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
   e.recibo_json->>'expediente_ref' AS expediente_recibo,
   e.recibo_json#>>'{procedencia,sin_retencion_declarada}'='true' AS nativa
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
  WHERE e.situacion='confirmada'
   AND e.solicitud_json->>'expediente_ref'=d.expediente_ref
   AND e.recibo_json->'propuesta_generada'='true'::jsonb
 ), exactas AS (
  SELECT x.llamamiento_ref,count(*) AS cantidad,
   count(*) FILTER (WHERE e.nativa) AS cantidad_nativa FROM pagina x
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
 ) SELECT coalesce(jsonb_object_agg(llamamiento_ref,jsonb_build_object(
    'confirmado',cantidad=1,'nativo',cantidad=1 AND cantidad_nativa=1)),'{}'::jsonb)
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
       AND origenes#>>ARRAY[x.llamamiento_ref,'confirmado']='true',false),
     'resolucion_portal_disponible',coalesce(
       jsonb_array_length(b.valor->'participaciones')=1
       AND part.valor->>'respuesta'='acepta' AND part.valor->>'modo'='firme'
       AND part.valor->>'recibo_respuesta_ref' IS NOT NULL
       AND part.valor->>'respondida_en' IS NOT NULL
       AND origenes#>>ARRAY[x.llamamiento_ref,'nativo']='true'
       AND r.resolucion_ref IS NULL,false)) ORDER BY part.ordinal),'[]'::jsonb)
    FROM jsonb_array_elements(b.valor->'participaciones') WITH ORDINALITY AS part(valor,ordinal)
    LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
      ON r.organizacion_ref=x.organizacion_ref AND r.expediente_ref=x.expediente_ref
     AND r.llamamiento_ref=x.llamamiento_ref
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
DO $post_ct249$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
   OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
   OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
       IS DISTINCT FROM 'be9cddb948958b0f7634355d0d73456334ccd4e8bc04beb8c884acb0d3f49fc8' THEN
  RAISE EXCEPTION 'CT249: lector final divergente' USING ERRCODE='55000'; END IF;
 IF (SELECT encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex')
     FROM pg_proc q WHERE q.oid=to_regprocedure(
      'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
    IS DISTINCT FROM '7b8874a22af362380a46f85f62ad25ab9ab2ac6cc453ff666bf34a25f982287d' THEN
  RAISE EXCEPTION 'CT249: sobrecarga de lectura divergente' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc q WHERE q.oid=to_regprocedure(
  'vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
  AND q.proowner='vec_contratacion_temporal_propietario'::regrole
  AND q.prosecdef IS TRUE AND q.provolatile='v'
  AND q.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
  AND q.proacl::text='{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}') THEN
  RAISE EXCEPTION 'CT249: función portal sin propietario o ACL previstos' USING ERRCODE='55000'; END IF;
END $post_ct249$;
COMMIT;
