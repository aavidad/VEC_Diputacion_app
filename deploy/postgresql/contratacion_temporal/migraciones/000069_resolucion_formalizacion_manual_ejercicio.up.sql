\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:resolucion-formalizacion:dependencia:000025-000069',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000069',0));
DO $dependencias$
DECLARE v_origen text;
BEGIN
 IF to_regclass('vec_contratacion_temporal.resolucion_formalizacion') IS NOT NULL
 OR to_regclass('vec_contratacion_temporal.propuesta_formalizacion') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(
 'vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_formalizacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
 AND has_function_privilege(current_user,p.oid,'EXECUTE')) THEN
 RAISE EXCEPTION 'dependencias de resolución incompatibles' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
 WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
 AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
 IF v_origen IS DISTINCT FROM $origen$CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text])))$origen$ THEN
 RAISE EXCEPTION 'origen integral incompatible con CT61' USING ERRCODE='55000';
 END IF;
END $dependencias$;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check
 CHECK (origen_version IN ('alta_o2','analisis_o3','cobertura_o4','asignacion_o5','informe_juridico_o5','fiscalizacion_o5','propuesta_formalizacion_o6','resolucion_formalizacion_o6'));

-- Registro de una validación manual del ejercicio. No sustituye ni duplica la
-- propuesta; conserva vínculos y compromisos del PDF generado en servidor.
CREATE TABLE vec_contratacion_temporal.resolucion_formalizacion (
 resolucion_ref text PRIMARY KEY,
 propuesta_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.propuesta_formalizacion,
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 clave_idempotencia uuid NOT NULL,
 version_resultante numeric(20,0) NOT NULL CHECK (version_resultante=8),
 material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 16384),
 material_json jsonb NOT NULL CHECK (material_json=material::jsonb),
 material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE,
 recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object' AND recibo_json->>'Estado'='registrada'
 AND recibo_json->'Solicitud'=material_json->'Solicitud' AND recibo_json->>'ResolucionRef'=resolucion_ref
 AND recibo_json->>'ReciboRef'=recibo_ref AND recibo_json->'VersionResultante'='8'::jsonb),
 evento_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.outbox_expediente_integral,
 registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en) AND registrada_en<>'0001-01-01T00:00:00Z'::timestamptz),
 tipo_validacion text NOT NULL DEFAULT 'manual_de_ejercicio' CHECK (tipo_validacion='manual_de_ejercicio'),
 firma_oficial boolean NOT NULL DEFAULT false CHECK (firma_oficial=false),
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK (eficacia_administrativa=false),
 FOREIGN KEY (expediente_ref,version_resultante) REFERENCES vec_contratacion_temporal.expediente_version_integral,
 UNIQUE (organizacion_ref,clave_idempotencia),
 UNIQUE (organizacion_ref,expediente_ref)
);
ALTER TABLE vec_contratacion_temporal.resolucion_formalizacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.resolucion_formalizacion FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.resolucion_formalizacion
 TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.resolucion_formalizacion
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.resolucion_formalizacion
 FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador;

CREATE FUNCTION vec_contratacion_temporal.registrar_resolucion_formalizacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 m jsonb; s jsonb; d jsonb; v_campo text;
 v_hash text; v_contexto_hash text; v_consumo record;
 v_propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
 v_previa vec_contratacion_temporal.resolucion_formalizacion%ROWTYPE;
 v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_original vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_resultado jsonb; v_ahora timestamptz(6); v_fecha date; v_fecha_propuesta timestamptz;
 v_resolucion_ref text; v_recibo text; v_evento text; v_unidad text;
 v_actuacion jsonb; v_agregado jsonb; v_agregado_hash text; v_prueba bytea;
 v_secuencia numeric(20,0); v_anterior text; v_payload bytea; v_outbox_hash text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
 OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
 RAISE EXCEPTION 'resolución denegada' USING ERRCODE='P0693';
 END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec:resolucion-formalizacion:dependencia:000025-000069',0));
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384 THEN
 RAISE EXCEPTION 'material inválido' USING ERRCODE='P0690'; END IF;
 BEGIN m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'JSON inválido' USING ERRCODE='P0690'; END;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY[
 'Solicitud','OrganizacionRef','DocumentoRef','PropuestaConfirmadaEn','DocumentoVersion','DocumentoSHA256']) IS NOT TRUE
 OR (SELECT count(*) FROM json_each(p_material::json))<>6
 OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',ARRAY[
 'ExpedienteRef','VersionEsperada','PropuestaRef','ClaveIdempotencia','NumeroResolucion','FechaResolucion','Motivo',
 'ConfirmaRevisionPropuesta','ConfirmaEjercicioManual']) IS NOT TRUE
 OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>9 THEN
 RAISE EXCEPTION 'contrato inválido' USING ERRCODE='P0690'; END IF;
 s:=m->'Solicitud';
 IF s->'VersionEsperada' IS DISTINCT FROM '7'::jsonb
 OR s->'ConfirmaRevisionPropuesta' IS DISTINCT FROM 'true'::jsonb
 OR s->'ConfirmaEjercicioManual' IS DISTINCT FROM 'true'::jsonb
 OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
 OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
 OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000' THEN
 RAISE EXCEPTION 'solicitud inválida' USING ERRCODE='P0690'; END IF;
 FOREACH v_campo IN ARRAY ARRAY['ExpedienteRef','PropuestaRef'] LOOP
 IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
 OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
 RAISE EXCEPTION 'referencia inválida' USING ERRCODE='P0690'; END IF; END LOOP;
 FOREACH v_campo IN ARRAY ARRAY['NumeroResolucion','Motivo','FechaResolucion'] LOOP
 IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string' OR btrim(s->>v_campo) IS DISTINCT FROM s->>v_campo
 OR (s->>v_campo) ~ '[[:cntrl:]]'
 OR octet_length(s->>v_campo) NOT BETWEEN 1 AND (CASE v_campo WHEN 'Motivo' THEN 2000 WHEN 'NumeroResolucion' THEN 80 ELSE 10 END) THEN
 RAISE EXCEPTION 'datos inválidos' USING ERRCODE='P0690'; END IF; END LOOP;
 IF (s->>'FechaResolucion')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR jsonb_typeof(m->'PropuestaConfirmadaEn') IS DISTINCT FROM 'string'
 OR (m->>'PropuestaConfirmadaEn')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
 RAISE EXCEPTION 'fecha inválida' USING ERRCODE='P0690'; END IF;
 BEGIN v_fecha:=(s->>'FechaResolucion')::date;
 v_fecha_propuesta:=(m->>'PropuestaConfirmadaEn')::timestamptz;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'fecha inválida' USING ERRCODE='P0690'; END;
 IF NOT isfinite(v_fecha) OR to_char(v_fecha,'YYYY-MM-DD') IS DISTINCT FROM s->>'FechaResolucion'
 OR jsonb_typeof(m->'OrganizacionRef') IS DISTINCT FROM 'string'
 OR (m->>'OrganizacionRef')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR jsonb_typeof(m->'DocumentoSHA256') IS DISTINCT FROM 'string'
 OR (m->>'DocumentoSHA256')!~'^[0-9a-f]{64}$' OR m->>'DocumentoSHA256'=repeat('0',64)
 OR m->'DocumentoVersion' IS DISTINCT FROM '7'::jsonb
 OR m->>'DocumentoRef' IS DISTINCT FROM 'documento-resolucion:'||encode(sha256(convert_to(s->>'PropuestaRef','UTF8')),'hex')
 OR NOT isfinite(v_fecha_propuesta) OR v_fecha_propuesta IS NULL THEN
 RAISE EXCEPTION 'fuente documental inválida' USING ERRCODE='P0690'; END IF;
 v_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_contexto_hash:=encode(sha256(convert_to(
 '{"ambitos":{"organizacion_ref":"'||(m->>'OrganizacionRef')||
 '"},"atributos":{"material_sha256":"'||v_hash||'"}}','UTF8')),'hex');
 IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
 RAISE EXCEPTION 'decisión inválida' USING ERRCODE='P0693'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'decisión inválida' USING ERRCODE='P0693'; END;
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.formalizacion.resolucion.manual_ejercicio.registrar'
 OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'resolucion_formalizacion_ct'
 OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_hash THEN
 RAISE EXCEPTION 'permiso divergente' USING ERRCODE='P0693'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_formalizacion_ct_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
 OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash THEN
 RAISE EXCEPTION 'consumo no ligado' USING ERRCODE='P0693'; END IF;

 -- Primera lectura de negocio tras consumir V3. PDF calculado por el servidor
 -- desde esta versión inmutable antes de atestar el material completo.
 SELECT * INTO v_propuesta FROM vec_contratacion_temporal.propuesta_formalizacion
 WHERE propuesta_ref=s->>'PropuestaRef' AND expediente_ref=s->>'ExpedienteRef'
 AND organizacion_ref=m->>'OrganizacionRef' AND version_resultante=7 FOR SHARE;
 IF NOT FOUND OR v_propuesta.confirmada_en IS DISTINCT FROM v_fecha_propuesta
 OR v_propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM v_propuesta.propuesta_ref
 OR v_propuesta.recibo_json->>'ReciboLocalRef' IS DISTINCT FROM v_propuesta.recibo_ref THEN
 RAISE EXCEPTION 'propuesta divergente' USING ERRCODE='P0692'; END IF;
 SELECT * INTO v_original FROM vec_contratacion_temporal.expediente_version_integral
 WHERE expediente_ref=v_propuesta.expediente_ref AND version=7 FOR SHARE;
 IF NOT FOUND OR v_original.operacion_ref IS DISTINCT FROM v_propuesta.propuesta_ref
 OR v_original.agregado_json#>>'{actuaciones,6,recibo_ref}' IS DISTINCT FROM v_propuesta.recibo_ref
 OR v_original.agregado_json#>>'{actuaciones,6,accion_clave}' IS DISTINCT FROM 'registrar_propuesta_formalizacion'
 OR v_original.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(v_original.agregado_json::text,'UTF8')),'hex') THEN
 RAISE EXCEPTION 'antecedente v7 no confiable' USING ERRCODE='P0694'; END IF;
 SELECT * INTO v_previa FROM vec_contratacion_temporal.resolucion_formalizacion
 WHERE organizacion_ref=m->>'OrganizacionRef' AND clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
 IF FOUND THEN
 IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id' OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
 RAISE EXCEPTION 'replay denegado' USING ERRCODE='P0693'; END IF;
 IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM m THEN
 RAISE EXCEPTION 'clave divergente' USING ERRCODE='P0691'; END IF;
 RETURN v_previa.recibo_json||jsonb_build_object('Estado','replay_registrada');
 END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_formalizacion
 WHERE propuesta_ref=v_propuesta.propuesta_ref OR (organizacion_ref=m->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef')) THEN
 RAISE EXCEPTION 'resolución ya registrada' USING ERRCODE='P0691'; END IF;
 SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
 WHERE a.expediente_ref=s->>'ExpedienteRef' FOR UPDATE OF a,v;
 IF NOT FOUND OR v_actual.version<>7 OR v_actual.fase_clave<>'nombramiento' OR v_actual.estado<>'en_curso'
 OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'OrganizacionRef'
 OR v_actual.agregado_json->'version' IS DISTINCT FROM '7'::jsonb
 OR v_actual.agregado_json IS DISTINCT FROM v_original.agregado_json
 OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_actual.agregado_json->'actuaciones')<>7 THEN
 RAISE EXCEPTION 'versión en conflicto' USING ERRCODE='P0692'; END IF;
    v_unidad:=v_actual.agregado_json#>>'{asignacion,unidad_ref}';
    IF coalesce(v_unidad,'')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'unidad de expediente no disponible' USING ERRCODE='P0694';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_propuesta.confirmada_en OR v_ahora<v_actual.registrada_en THEN
        RAISE EXCEPTION 'reloj anterior a los antecedentes' USING ERRCODE='P0694';
    END IF;
    v_resolucion_ref:='resolucion:'||gen_random_uuid()::text;
    v_recibo:='recibo:'||gen_random_uuid()::text;
    v_evento:='evento:'||gen_random_uuid()::text;
    v_actuacion:=jsonb_build_object(
        'secuencia',8,'version_expediente',8,'accion_clave','registrar_resolucion_formalizacion',
        'actor_ref',d->>'principal_id','unidad_ref',v_unidad,'recibo_ref',v_recibo,
        'realizada_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'fase_origen',v_actual.agregado_json->'fase_actual','fase_destino','nombramiento',
        'estado_origen',v_actual.agregado_json->'estado_actual','estado_destino','en_curso');
    v_agregado:=v_actual.agregado_json||jsonb_build_object(
        'version',8,'fase_actual','nombramiento','estado_actual','en_curso',
        'actualizado_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    v_agregado_hash:=encode(sha256(convert_to(v_agregado::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-RESOLUCION-V1'||chr(10)||
        (s->>'ExpedienteRef')||chr(10)||'8'||chr(10)||v_agregado_hash||chr(10)||
        v_resolucion_ref||chr(10)||v_recibo||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',8,v_agregado,v_agregado_hash,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
        'resolucion_formalizacion_o6',v_resolucion_ref,v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version=8,actualizada_en=v_ahora,operacion_ref=v_resolucion_ref
     WHERE expediente_ref=s->>'ExpedienteRef' AND version=7;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'versión de propuesta perdida' USING ERRCODE='P0692';
    END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-RESOLUCION-V1'||chr(10)||v_actuacion::text||chr(10)||v_recibo||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',8,8,v_resolucion_ref,v_recibo,v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991::numeric THEN
        RAISE EXCEPTION 'límite de outbox alcanzado' USING ERRCODE='P0694';
    END IF;
    v_secuencia:=v_secuencia+1;
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.resolucion-formalizacion.v1',
        'expediente_ref',s->>'ExpedienteRef','version_resultante',8,'resolucion_formalizacion_ref',v_resolucion_ref,'propuesta_ref',v_propuesta.propuesta_ref,
        'tipo_validacion','manual_de_ejercicio','firma_oficial',false,'eficacia_administrativa',false,'recibo_ref',v_recibo)::text,'UTF8');
    v_outbox_hash:=encode(sha256(v_anterior::bytea||v_payload),'hex');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,
        payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en
    ) VALUES (
        v_evento,v_secuencia,v_resolucion_ref,s->>'ExpedienteRef',8,'contratacion_temporal.resolucion_formalizacion_registrada',
        v_payload,encode(sha256(v_payload),'hex'),v_anterior,v_outbox_hash,v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=v_outbox_hash,actualizada_en=v_ahora WHERE control_id;
    v_resultado:=jsonb_build_object('Solicitud',s,'Estado','registrada',
 'ResolucionRef',v_resolucion_ref,'DocumentoRef',m->>'DocumentoRef',
 'DocumentoVersion',7,'DocumentoSHA256',m->>'DocumentoSHA256','ActuacionRef',v_resolucion_ref,
 'AuditoriaRef',v_consumo.auditoria_ref,'OutboxRef',v_evento,'ReciboRef',v_recibo,
 'VersionResultante',8,'RegistradaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_contratacion_temporal.resolucion_formalizacion(
 resolucion_ref,propuesta_ref,organizacion_ref,expediente_ref,clave_idempotencia,version_resultante,
 material,material_json,material_sha256,actor_ref,perfil_ref,auditoria_ref,decision_ref,
 consumo_huella_sha256,recibo_ref,recibo_json,evento_ref,registrada_en)
 VALUES (v_resolucion_ref,v_propuesta.propuesta_ref,m->>'OrganizacionRef',s->>'ExpedienteRef',
 (s->>'ClaveIdempotencia')::uuid,8,p_material,m,v_hash,d->>'principal_id',d->>'perfil_activo_ref',
 v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_recibo,v_resultado,v_evento,v_ahora);
 RETURN v_resultado;
EXCEPTION
 WHEN unique_violation THEN RAISE EXCEPTION 'conflicto de resolución' USING ERRCODE='P0691';
 WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'resolución no disponible' USING ERRCODE='P0694';
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_resolucion_formalizacion_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_contratacion_temporal_migrador,vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_resolucion_formalizacion_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
-- Mantiene el control de versión general. Solo permite releer el v7
-- original tras esta actuación v8 y sin cambiar organización, centro o unidad.
DO $detalle_historico$
DECLARE v_def text; v_acl aclitem[]; v_owner oid;
 v_marca text := $marca$           AND publicacion.corte_global <= p_corte_global$marca$;
 v_extension text := $extension$           AND (
               p_consulta.version_observada <> 7 OR publicacion.version = 7
               OR NOT EXISTS (
                   SELECT 1 FROM (
                       SELECT actual.* FROM vec_contratacion_temporal.publicacion_version_rrhh actual
                       WHERE actual.expediente_ref = publicacion.expediente_ref
                         AND actual.corte_global <= p_corte_global
                       ORDER BY actual.corte_global DESC LIMIT 1
                   ) vigente
                   JOIN vec_contratacion_temporal.resolucion_formalizacion resolucion
                     ON resolucion.expediente_ref = vigente.expediente_ref
                    AND resolucion.organizacion_ref = vigente.organizacion_ref
                   WHERE vigente.version = 8
                     AND vigente.organizacion_ref = publicacion.organizacion_ref
                     AND vigente.centro_ref IS NOT DISTINCT FROM publicacion.centro_ref
                     AND vigente.unidad_ref IS NOT DISTINCT FROM publicacion.unidad_ref
               )
           )
$extension$;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner INTO STRICT v_def,v_acl,v_owner
 FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca) OR strpos(v_def,'resolucion_formalizacion')<>0 THEN
 RAISE EXCEPTION 'materializador incompatible con recuperación v7' USING ERRCODE='55000'; END IF;
 EXECUTE replace(v_def,v_marca,v_marca||chr(10)||v_extension);
 IF (SELECT proacl FROM pg_proc WHERE oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure) IS DISTINCT FROM v_acl
 OR (SELECT proowner FROM pg_proc WHERE oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure) IS DISTINCT FROM v_owner THEN
 RAISE EXCEPTION 'recuperación alteró autoridad del materializador' USING ERRCODE='55000'; END IF;
END $detalle_historico$;
-- Preparación sin efecto de negocio. La fachada existente comprueba el LOGIN,
-- acción/recurso/finalidad/ámbito y consume V3 con auditoría en esta transacción.
CREATE FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona numeric,p_perfil numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric,
 anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text, registrada_en timestamptz, auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text,
 generada_en timestamptz, expediente_ref text, version_expediente numeric,
 total smallint, recibo_sello_sha256 text, preparacion jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $preparacion$
DECLARE v_lectura record;
 v_propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
 v_original vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_resolucion vec_contratacion_temporal.resolucion_formalizacion%ROWTYPE;
 v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 v_preparacion jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec:resolucion-formalizacion:dependencia:000025-000069',0));
 IF p_consulta.version_observada IS DISTINCT FROM 0::numeric THEN
 RAISE EXCEPTION 'preparación inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_lectura FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
 p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona,p_perfil,p_payload,p_sobre,p_evidencia,p_raiz);
 -- No consulta tablas de negocio antes de validar y consumir el permiso lector.
 IF v_lectura.expediente_ref IS DISTINCT FROM p_consulta.expediente_ref
 OR v_lectura.version_expediente NOT IN (7,8) THEN
 RAISE EXCEPTION 'preparación fuera de fase' USING ERRCODE='P0692'; END IF;
 SELECT v.* INTO STRICT v_actual FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
 WHERE a.expediente_ref=v_lectura.expediente_ref;
 IF v_actual.version IS DISTINCT FROM v_lectura.version_expediente
 OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref THEN
 RAISE EXCEPTION 'publicación divergente' USING ERRCODE='P0694'; END IF;
 SELECT * INTO STRICT v_original FROM vec_contratacion_temporal.expediente_version_integral ev
 WHERE ev.expediente_ref=v_lectura.expediente_ref AND version=7;
 SELECT * INTO STRICT v_propuesta FROM vec_contratacion_temporal.propuesta_formalizacion pf
 WHERE propuesta_ref=v_original.operacion_ref AND pf.expediente_ref=v_lectura.expediente_ref
 AND organizacion_ref=p_alcance.organizacion_ref AND version_resultante=7;
 IF v_original.agregado_json#>>'{actuaciones,6,accion_clave}' IS DISTINCT FROM 'registrar_propuesta_formalizacion'
 OR v_original.agregado_json#>>'{actuaciones,6,recibo_ref}' IS DISTINCT FROM v_propuesta.recibo_ref
 OR v_propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM v_propuesta.propuesta_ref
 OR v_original.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(v_original.agregado_json::text,'UTF8')),'hex') THEN
 RAISE EXCEPTION 'propuesta no confiable' USING ERRCODE='P0694'; END IF;
 SELECT * INTO v_resolucion FROM vec_contratacion_temporal.resolucion_formalizacion rf
 WHERE rf.expediente_ref=v_lectura.expediente_ref AND organizacion_ref=p_alcance.organizacion_ref
 AND propuesta_ref=v_propuesta.propuesta_ref;
 IF (v_lectura.version_expediente=7 AND FOUND) OR (v_lectura.version_expediente=8 AND NOT FOUND) THEN
 RAISE EXCEPTION 'recibo no disponible' USING ERRCODE='P0694'; END IF;
 IF v_lectura.version_expediente=8 AND (
 v_actual.operacion_ref IS DISTINCT FROM v_resolucion.resolucion_ref
 OR v_actual.agregado_json#>>'{actuaciones,7,recibo_ref}' IS DISTINCT FROM v_resolucion.recibo_ref
 OR v_actual.agregado_json#>>'{actuaciones,7,accion_clave}' IS DISTINCT FROM 'registrar_resolucion_formalizacion'
 OR v_resolucion.recibo_json->>'AuditoriaRef' IS DISTINCT FROM v_resolucion.auditoria_ref
 OR v_resolucion.recibo_json->>'OutboxRef' IS DISTINCT FROM v_resolucion.evento_ref
 OR v_resolucion.recibo_json->>'DocumentoSHA256' IS DISTINCT FROM v_resolucion.material_json->>'DocumentoSHA256'
 OR v_resolucion.recibo_json->>'DocumentoRef' IS DISTINCT FROM v_resolucion.material_json->>'DocumentoRef'
 OR v_resolucion.tipo_validacion IS DISTINCT FROM 'manual_de_ejercicio'
 OR v_resolucion.firma_oficial IS DISTINCT FROM false OR v_resolucion.eficacia_administrativa IS DISTINCT FROM false) THEN
 RAISE EXCEPTION 'recibo no confiable' USING ERRCODE='P0694'; END IF;
 v_preparacion:=jsonb_build_object('ExpedienteRef',v_lectura.expediente_ref,
 'PropuestaRef',v_propuesta.propuesta_ref,'VersionEsperada',7,'VersionActual',v_lectura.version_expediente,
 'Recibo',CASE WHEN v_lectura.version_expediente=8 THEN v_resolucion.recibo_json ELSE NULL END);
 RETURN QUERY SELECT v_lectura.contenido_canonico,
 v_lectura.esquema,
 v_lectura.acceso_ref,
 v_lectura.secuencia,
 v_lectura.anterior_sha256,
 v_lectura.huella_sha256,
 v_lectura.vinculo_identidad_huella_sha256,
 v_lectura.alcance_huella_sha256,
 v_lectura.registrada_en,
 v_lectura.auditoria_vec_ref,
 v_lectura.auditoria_vec_huella_sha256,
 v_lectura.consumo_vec_huella_sha256,
 v_lectura.contenido_huella_sha256,
 v_lectura.resultado_huella_sha256,
 v_lectura.cursor_huella_sha256,
 v_lectura.generada_en,
 v_lectura.expediente_ref,
 v_lectura.version_expediente,
 v_lectura.total,
 v_lectura.recibo_sello_sha256, v_preparacion;
END $preparacion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador,vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
