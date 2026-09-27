\set ON_ERROR_STOP on
-- CT134: lectura nominal y durable del antecedente CT115/CT75 para CT130.
-- AD3-98 debe estar instalado. El llamante usa SERIALIZABLE READ WRITE.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000134',0));

DO $pre$
DECLARE f regprocedure;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_contratacion_temporal_propietario' OR f IS NULL
    OR NOT has_function_privilege(current_user,f,'EXECUTE')
    OR to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.lectura_reincorporacion_titular_v1') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.validar_material_reincorporacion_ct130(jsonb)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.huella_contexto_go_ct115(jsonb,jsonb)') IS NULL
    OR to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.incorporacion_registro_v2') IS NULL
    OR to_regclass('vec_contratacion_temporal.seguimiento_raiz_v2') IS NULL THEN
  RAISE EXCEPTION 'CT134: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 (
 lectura_ref text PRIMARY KEY CHECK (lectura_ref ~ '^lectura:[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
 peticion_sha256 text NOT NULL CHECK (peticion_sha256 ~ '^[0-9a-f]{64}$'),
 contexto_sha256 text NOT NULL CHECK (contexto_sha256 ~ '^[0-9a-f]{64}$'),
 resultado text NOT NULL CHECK (resultado IN ('coincide','no_coincide')),
 cese_evento_ref text NOT NULL,
 cese_recibo_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
 registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
 CHECK ((resultado='coincide' AND cese_evento_ref<>'' AND cese_recibo_ref<>'')
     OR (resultado='no_coincide' AND cese_evento_ref='' AND cese_recibo_ref='')),
 CHECK (vec_contratacion_temporal.referencia_valida_ct115(organizacion_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(expediente_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(actor_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(perfil_ref))
);
CREATE INDEX lectura_reincorporacion_titular_v1_expediente
 ON vec_contratacion_temporal.lectura_reincorporacion_titular_v1(organizacion_ref,expediente_ref,registrada_en);
ALTER TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY insercion_ct134 ON vec_contratacion_temporal.lectura_reincorporacion_titular_v1
 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
 organizacion_ref=current_setting('vec.ct134.organizacion_ref',true)
 AND expediente_ref=current_setting('vec.ct134.expediente_ref',true)
 AND actor_ref=current_setting('vec.ct134.actor_ref',true)
 AND perfil_ref=current_setting('vec.ct134.perfil_ref',true)
 AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
-- Solo la migración puede inspeccionar si hay historia antes de DOWN.
CREATE POLICY lectura_migracion_ct134 ON vec_contratacion_temporal.lectura_reincorporacion_titular_v1
 FOR SELECT TO vec_contratacion_temporal_propietario USING (
 pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER'));
CREATE TRIGGER lectura_reincorporacion_titular_v1_inmutable
 BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.lectura_reincorporacion_titular_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(
 p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 v_fecha date; v_version numeric; v_ambitos jsonb; v_atributos jsonb; v_hash text; v_peticion_sha text;
 v_capacidad jsonb; v_decision jsonb; v_contexto jsonb; v_consumo record;
 v_evento text:=''; v_recibo text:=''; v_resultado text; v_lectura text; v_ahora timestamptz(6);
BEGIN
 PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
 v_fecha:=vec_contratacion_temporal.validar_material_reincorporacion_ct130(p_material);
 v_version:=(p_material->>'version_esperada')::numeric;
 v_ambitos:=jsonb_build_object('organizacion_ref',p_material->>'organizacion_ref','expediente_ref',p_material->>'expediente_ref');
 v_atributos:=jsonb_build_object('version_expediente',v_version::text,'relacion_ref',p_material->>'relacion_ref',
  'fecha_efectiva',p_material->>'fecha_efectiva','documento_ref',p_material->>'documento_ref',
  'documento_sha256',p_material->>'documento_sha256');
 v_hash:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
 v_peticion_sha:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
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
    OR p_persona_version IS NULL OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
    OR p_perfil_version IS NULL OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version<>trunc(p_persona_version) OR p_perfil_version<>trunc(p_perfil_version) THEN
  RAISE EXCEPTION 'CT134: lectura denegada' USING ERRCODE='42501'; END IF;
 v_capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
 v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 v_contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.lectura_reincorporacion_titular.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'contratacion_temporal.seguimiento.consultar_reincorporacion_titular'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM p_material->>'expediente_ref'
    OR v_capacidad->>'huella_efecto_sha256' IS DISTINCT FROM v_hash
    OR v_capacidad->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.seguimiento.consultar_reincorporacion_titular'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'lectura_reincorporacion_titular_contratacion_temporal'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'verificar_antecedente_reincorporacion_titular'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_material->>'expediente_ref'
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_hash
    OR v_decision->>'principal_id' IS DISTINCT FROM p_material->>'actor_ref'
    OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM p_material->>'perfil_ref'
    OR v_decision->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
    OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
    OR v_contexto->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR v_contexto->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR v_decision->'campos_permitidos' IS DISTINCT FROM
      '["cese_evento_ref","cese_recibo_ref","documento_ref","documento_sha256","existe_cese","fecha_efectiva","relacion_ref"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'CT134: lectura denegada' USING ERRCODE='42501'; END IF;

 -- AD3 registra auditoría y consume la decisión antes de observar CT115/CT75.
 SELECT * INTO STRICT v_consumo FROM
  vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.decision_ref IS DISTINCT FROM v_decision->>'decision_ref'
    OR v_consumo.efecto_ref IS DISTINCT FROM p_material->>'expediente_ref'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_hash
    OR coalesce(v_consumo.consumo_huella_sha256,'') !~ '^[0-9a-f]{64}$'
    OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'CT134: consumo divergente' USING ERRCODE='42501'; END IF;

 PERFORM set_config('vec.ct115.organizacion_ref',p_material->>'organizacion_ref',true);
 PERFORM set_config('vec.ct115.expediente_ref',p_material->>'expediente_ref',true);
 -- La coincidencia exige el cese confirmado, su evento y la relación raíz CT75.
 SELECT c.evento_ref,c.recibo_ref INTO v_evento,v_recibo
 FROM vec_contratacion_temporal.cese_nombramiento_v1 c
 JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref
 WHERE c.organizacion_ref=p_material->>'organizacion_ref'
   AND c.expediente_ref=p_material->>'expediente_ref'
   AND c.estado='confirmada' AND c.causa_clave='fin_sustitucion'
   AND c.justificante_tipo='comunicacion_reincorporacion'
   AND c.fecha_efecto=v_fecha AND c.justificante_ref=p_material->>'documento_ref'
   AND c.justificante_sha256=p_material->>'documento_sha256'
   AND i.organizacion_ref=c.organizacion_ref AND i.expediente_ref=c.expediente_ref
   AND r.organizacion_ref=c.organizacion_ref AND r.expediente_ref=c.expediente_ref
   AND i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'=r.relacion_ref
   AND r.relacion_ref=p_material->>'relacion_ref'
   AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral o
    WHERE o.evento_ref=c.evento_ref AND o.expediente_ref=c.expediente_ref
      AND o.operacion_ref=c.reserva_ref AND o.version_expediente=c.version_esperada+1
      AND o.tipo_evento='ct.cese.v1');
 v_resultado:=CASE WHEN FOUND THEN 'coincide' ELSE 'no_coincide' END;
 IF v_resultado='no_coincide' THEN v_evento:=''; v_recibo:=''; END IF;
 v_lectura:='lectura:'||encode(sha256(convert_to(
  'VEC-CT134-LECTURA'||chr(10)||v_consumo.consumo_huella_sha256||chr(10)||v_peticion_sha,'UTF8')),'hex');
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 PERFORM set_config('vec.ct134.organizacion_ref',p_material->>'organizacion_ref',true);
 PERFORM set_config('vec.ct134.expediente_ref',p_material->>'expediente_ref',true);
 PERFORM set_config('vec.ct134.actor_ref',p_material->>'actor_ref',true);
 PERFORM set_config('vec.ct134.perfil_ref',p_material->>'perfil_ref',true);
 INSERT INTO vec_contratacion_temporal.lectura_reincorporacion_titular_v1(
  lectura_ref,organizacion_ref,expediente_ref,actor_ref,perfil_ref,version_esperada,
  peticion_sha256,contexto_sha256,resultado,cese_evento_ref,cese_recibo_ref,decision_ref,
  consumo_huella_sha256,auditoria_ref,registrada_en)
 VALUES(v_lectura,p_material->>'organizacion_ref',p_material->>'expediente_ref',
  p_material->>'actor_ref',p_material->>'perfil_ref',v_version,v_peticion_sha,v_hash,
  v_resultado,v_evento,v_recibo,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,
  v_consumo.auditoria_ref,v_ahora);
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.lectura-reincorporacion-titular.v1',
  'resultado',v_resultado,'expediente_ref',p_material->>'expediente_ref',
  'cese_evento_ref',v_evento,'cese_recibo_ref',v_recibo,'lectura_ref',v_lectura,
  'auditoria_ref',v_consumo.auditoria_ref,'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
  'registrada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;

ALTER FUNCTION vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 OWNER TO vec_contratacion_temporal_propietario;
DO $acl_cerrar$
DECLARE x record; f regprocedure:=
 'vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_class c
  CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid='vec_contratacion_temporal.lectura_reincorporacion_titular_v1'::regclass
    AND a.grantee<>c.relowner LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 FROM %s',
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
END $acl_cerrar$;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
DO $acl$
BEGIN
 IF has_table_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.lectura_reincorporacion_titular_v1',
   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_function_privilege('public',
   'vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'EXECUTE') THEN RAISE EXCEPTION 'CT134: ACL abierta' USING ERRCODE='42501'; END IF;
END $acl$;
COMMENT ON TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 IS
 'CT134: recibos inmutables de lecturas autorizadas del antecedente de reincorporación del titular.';
COMMIT;
