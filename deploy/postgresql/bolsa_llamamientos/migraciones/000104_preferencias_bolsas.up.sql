\set ON_ERROR_STOP on
-- B104. Base de historia propia para preferencias ENTRE bolsas. No modifica
-- puntuaciones, orden de bolsa, disponibilidad, ofertas ni llamamientos.
-- Se instala después de AD241; las fachadas V3 se completan en este mismo
-- corte antes de instalar la migración en cualquier base conservada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000104',0));

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.preferencia_bolsas_version') IS NOT NULL
 THEN RAISE EXCEPTION 'B104: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Solo adición. El último número de versión es la preferencia vigente; la
-- clave y huella conservan la semántica de reintento tras respuesta incierta.
CREATE TABLE vec_bolsa_llamamientos.preferencia_bolsas_version (
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK (version BETWEEN 1 AND 2147483647),
 orden text[] NOT NULL CHECK (array_ndims(orden)=1 AND cardinality(orden) BETWEEN 1 AND 256 AND array_position(orden,NULL) IS NULL),
 conjunto_huella_sha256 text NOT NULL CHECK (conjunto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9._:-]{8,128}$'),
 solicitud_huella_sha256 text NOT NULL CHECK (solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:preferencias-bolsas:[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE CHECK (octet_length(decision_ref) BETWEEN 1 AND 256),
 auditoria_ref text NOT NULL CHECK (octet_length(auditoria_ref) BETWEEN 1 AND 256),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (candidato_ref,version),
 UNIQUE (candidato_ref,clave_idempotencia)
);
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version FORCE ROW LEVEL SECURITY;
CREATE POLICY preferencia_bolsas_version_propietario ON vec_bolsa_llamamientos.preferencia_bolsas_version
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.preferencia_bolsas_version FROM PUBLIC;
CREATE TRIGGER preferencia_bolsas_version_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.preferencia_bolsas_version
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Un candidato puede conservar participaciones históricas de una bolsa.
-- Solo se ordena la constitución más reciente aún vigente o suspendida, en
-- su intervalo temporal; la bolsa extinguida pasa a 'retirada' en la lectura.
-- Esta función es privada al propietario, nunca una fachada de lectura.
CREATE FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(
 p_candidato_ref text,p_corte timestamptz)
RETURNS text[] LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_bolsas text[];
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_corte IS NULL OR NOT isfinite(p_corte)
 THEN RAISE EXCEPTION 'B104: consulta interna inválida' USING ERRCODE='22023'; END IF;
 WITH ultima AS (
  SELECT DISTINCT ON (p.bolsa_ref) p.bolsa_ref,p.estado,p.vigente_desde,p.vigente_hasta
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) p
   ORDER BY p.bolsa_ref,p.confirmada_en DESC,p.participacion_ref DESC
 )
 SELECT coalesce(array_agg(bolsa_ref ORDER BY bolsa_ref),ARRAY[]::text[])
 INTO v_bolsas FROM ultima
 WHERE estado IN ('vigente','suspendida') AND vigente_desde<=p_corte
   AND (vigente_hasta IS NULL OR vigente_hasta>p_corte);
 RETURN v_bolsas;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(text,timestamptz) FROM PUBLIC;

-- Proyección privada de la versión escrita y de las diferencias actuales.
-- Nunca añade una bolsa al orden guardado por inferencia.
CREATE FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(
 p_candidato_ref text,p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_actuales text[]; v_orden text[]; v_nuevas text[]; v_retiradas text[];
 v_version bigint:=0; v_recibo text; v_fecha timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'B104: lectura interna denegada' USING ERRCODE='42501';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,p_corte);
 SELECT p.version,p.orden,p.recibo_ref,p.registrada_en
 INTO v_version,v_orden,v_recibo,v_fecha
 FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref ORDER BY p.version DESC LIMIT 1;
 v_version:=coalesce(v_version,0);
 v_orden:=coalesce(v_orden,ARRAY[]::text[]);
 SELECT coalesce(array_agg(u.ref ORDER BY u.ref),ARRAY[]::text[]) INTO v_nuevas
 FROM unnest(v_actuales) AS u(ref) WHERE NOT u.ref=ANY(v_orden);
 SELECT coalesce(array_agg(u.ref ORDER BY u.ref),ARRAY[]::text[]) INTO v_retiradas
 FROM unnest(v_orden) AS u(ref) WHERE NOT u.ref=ANY(v_actuales);
 RETURN jsonb_build_object('version',v_version,'orden_guardado',to_jsonb(v_orden),
  'bolsas_actuales',to_jsonb(v_actuales),'bolsas_nuevas',to_jsonb(v_nuevas),
  'bolsas_retiradas',to_jsonb(v_retiradas),
  'revision_requerida',v_version=0 OR cardinality(v_nuevas)<>0 OR cardinality(v_retiradas)<>0,
  'recibo_ref',v_recibo,'registrado_en',v_fecha);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(text,timestamptz) FROM PUBLIC;

-- Fachada de efecto. AD241 verifica la concesión vigente, el contexto y el
-- material atestado y registra su auditoría dentro de esta transacción. Una
-- autorización nueva se exige también para recuperar un recibo por clave.
CREATE FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 p_candidato_ref text,p_orden text[],p_version_esperada bigint,p_clave text,p_registrada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,orden text[],registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_actuales text[]; v_ordenado text[];
 v_previa vec_bolsa_llamamientos.preferencia_bolsas_version%ROWTYPE;
 v_consumo record; v_version bigint; v_huella text; v_conjunto text; v_recibo text;
 v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_orden IS NULL OR array_ndims(p_orden) IS DISTINCT FROM 1 OR cardinality(p_orden) NOT BETWEEN 1 AND 256
    OR array_position(p_orden,NULL) IS NOT NULL
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
    OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_registrada_en < v_ahora-interval '5 minutes' OR p_registrada_en > v_ahora+interval '1 minute'
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B104: orden inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B104: material inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B104: orden denegado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'B104: candidato denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B104: consumo denegado' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias_bolsas:'||p_candidato_ref,0));
 IF EXISTS (SELECT 1 FROM unnest(p_orden) AS u(ref)
  WHERE u.ref IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(u.ref,512)) THEN
  RAISE EXCEPTION 'B104: referencia inválida' USING ERRCODE='22023'; END IF;
 -- La huella usa separador de control prohibido en las referencias canónicas.
 v_huella:=encode(sha256(convert_to(p_candidato_ref||chr(31)||p_version_esperada::text||chr(31)||
  array_to_string(p_orden,chr(31)),'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref AND p.clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previa.solicitud_huella_sha256<>v_huella OR v_previa.version<>p_version_esperada+1
  THEN RAISE EXCEPTION 'B104: clave reutilizada con otro orden' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.orden,v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B104: versión de preferencias desactualizada' USING ERRCODE='VBP01';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,p_registrada_en);
 IF cardinality(v_actuales)<>cardinality(p_orden) THEN
  RAISE EXCEPTION 'B104: conjunto de bolsas cambiado' USING ERRCODE='VBP03'; END IF;
 SELECT array_agg(u.ref ORDER BY u.ref) INTO v_ordenado FROM unnest(p_orden) AS u(ref);
 IF v_ordenado IS DISTINCT FROM v_actuales THEN
  RAISE EXCEPTION 'B104: orden incompleto, duplicado o ajeno' USING ERRCODE='VBP03'; END IF;
 v_conjunto:=encode(sha256(convert_to(array_to_string(v_actuales,chr(31)),'UTF8')),'hex');
 v_recibo:='recibo:preferencias-bolsas:'||encode(sha256(convert_to(
  p_candidato_ref||chr(31)||p_clave||chr(31)||v_huella,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_candidato_ref,v_version+1,p_orden,v_conjunto,p_clave,v_huella,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,p_registrada_en);
 RETURN QUERY SELECT false,v_recibo,v_version+1,p_orden,p_registrada_en;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
