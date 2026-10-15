\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000122',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.transiciones_situacion_canonicas(text[])') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[])') IS NULL
    OR to_regprocedure('vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(integer,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_politica_transiciones_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_gobierno') IS NOT NULL
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[])','EXECUTE')
 THEN RAISE EXCEPTION 'B122: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.transiciones_situacion_canonicas(text[])'::regprocedure),'UTF8')),'hex')
      IS DISTINCT FROM '625f4a248ca9c096445d738562cc984eeac68182e5bb36f860672926f59d6d33'
    OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[])'::regprocedure),'UTF8')),'hex')
      IS DISTINCT FROM 'd44823dc40643571f66710b21e8992f002fa4a94f91f92b8d5d52d1cca1ddf8b'
 THEN RAISE EXCEPTION 'B122: funciones B32 distintas' USING ERRCODE='55000'; END IF;
END $pre$;

-- B32 conserva íntegra su historia. Esta tabla añade a cada nueva versión el
-- acto nominal V3 y la referencia normativa activa en ese instante.
CREATE TABLE vec_bolsa_llamamientos.politica_transiciones_gobierno (
 version bigint PRIMARY KEY REFERENCES vec_bolsa_llamamientos.politica_transiciones_situacion(version),
 version_esperada bigint NOT NULL CHECK(version_esperada>=1 AND version=version_esperada+1),
 catalogo_version integer NOT NULL CHECK(catalogo_version>=1),
 catalogo_sha256 text NOT NULL CHECK(catalogo_sha256 ~ '^[0-9a-f]{64}$'),
 limites_snapshot jsonb NOT NULL CHECK(jsonb_typeof(limites_snapshot)='object'),
 contexto_recurso_sha256 text NOT NULL CHECK(contexto_recurso_sha256 ~ '^[0-9a-f]{64}$'),
 ambito_sha256 text NOT NULL CHECK(ambito_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 perfil_ref text NOT NULL CHECK(perfil_ref ~ '^[A-Za-z0-9:_-]{3,256}$'),
 motivo text NOT NULL CHECK(motivo=btrim(motivo) AND octet_length(motivo) BETWEEN 1 AND 1000),
 clave_idempotencia text NOT NULL UNIQUE CHECK(clave_idempotencia ~ '^[A-Za-z0-9:_-]{8,256}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:politica-transiciones:[0-9a-f]{64}$'),
 contenido_sha256 text NOT NULL CHECK(contenido_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 publicada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_gobierno ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_gobierno FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_transiciones_gobierno_propietario ON vec_bolsa_llamamientos.politica_transiciones_gobierno
 TO vec_bolsa_llamamientos_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON vec_bolsa_llamamientos.politica_transiciones_gobierno FROM PUBLIC;
CREATE TRIGGER politica_transiciones_gobierno_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.politica_transiciones_gobierno FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.politica_transiciones_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.politica_transiciones_gobierno(recibo_ref),
 version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_transiciones_gobierno(version),
 contenido_sha256 text NOT NULL CHECK(contenido_sha256 ~ '^[0-9a-f]{64}$'),
 esquema text NOT NULL CHECK(esquema='vec.bolsa.politica-transiciones.publicada.v1'),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 evento_sha256 text NOT NULL CHECK(evento_sha256=encode(sha256(convert_to(evento::text,'UTF8')),'hex')),
 creada_en timestamptz(6) NOT NULL,
 entregada_en timestamptz(6)
);
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_transiciones_outbox_propietario ON vec_bolsa_llamamientos.politica_transiciones_outbox
 TO vec_bolsa_llamamientos_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON vec_bolsa_llamamientos.politica_transiciones_outbox FROM PUBLIC;

-- La selección privada del perfil sirve para preparar la primera activación.
-- La política vigente permanece en B32 hasta un POST autorizado V3.
CREATE FUNCTION vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(
 p_catalogo_version integer,p_catalogo_sha256 text) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE v_limites jsonb; v_politica record; v_gobierno record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'B122: lectura denegada' USING ERRCODE='42501'; END IF;
 v_limites:=vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(p_catalogo_version,p_catalogo_sha256);
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 SELECT * INTO v_gobierno FROM vec_bolsa_llamamientos.politica_transiciones_gobierno WHERE version=v_politica.version;
 RETURN jsonb_build_object('version',v_politica.version,
  'transiciones_guardadas',to_jsonb(v_politica.transiciones),
  'limites',v_limites,
  'catalogo_activado',v_gobierno.version IS NOT NULL AND
   v_gobierno.catalogo_version=p_catalogo_version AND v_gobierno.catalogo_sha256=p_catalogo_sha256,
  'recibo_ref',v_gobierno.recibo_ref);
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(
 p_version_esperada bigint,p_transiciones text[],p_motivo text,p_clave text,p_recibo text,p_actor text,
 p_catalogo_version integer,p_catalogo_sha256 text,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(politica jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE v_consumo record; v_decision jsonb; v_limites jsonb; v_permitidas text[];
 v_canonica text[]; v_vigente record; v_previa record; v_anterior record; v_huella text; v_ahora timestamptz:=clock_timestamp();
 v_contexto text; v_perfil_ref text; v_ambito_sha256 text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_version_esperada IS NULL OR p_version_esperada<1
    OR current_setting('TimeZone')<>'UTC'
    OR (SELECT setting::bigint FROM pg_settings WHERE name='statement_timeout') NOT BETWEEN 1 AND 15000
    OR (SELECT setting::bigint FROM pg_settings WHERE name='idle_in_transaction_session_timeout') NOT BETWEEN 1 AND 20000
    OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]{8,256}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-transiciones:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'B122: propuesta inválida' USING ERRCODE='22023'; END IF;
 v_canonica:=vec_bolsa_llamamientos.transiciones_situacion_canonicas(p_transiciones);
 IF v_canonica IS NULL OR v_canonica IS DISTINCT FROM p_transiciones THEN
  RAISE EXCEPTION 'B122: transiciones inválidas' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_politica_transiciones_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B122: autorización inválida' USING ERRCODE='42501'; END;
 v_perfil_ref:=v_decision#>>'{vinculo_autenticacion_actor,perfil_activo_ref}';
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'recurso_ref' IS DISTINCT FROM 'bolsa:politica:transiciones:global'
    OR v_perfil_ref IS NULL OR v_perfil_ref !~ '^[A-Za-z0-9:_-]{3,256}$'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'B122: ámbito global denegado' USING ERRCODE='42501'; END IF;
 -- También en replay se valida V3 fresca contra la configuración normativa
 -- actual. La norma original queda congelada en el acto guardado.
 v_limites:=vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(p_catalogo_version,p_catalogo_sha256);
 v_contexto:=format('{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{'
  ||'"actor_ref":"%s","catalogo_sha256":"%s","catalogo_version":"%s",'
  ||'"clave_sha256":"%s","motivo_sha256":"%s","perfil_activo_ref":"%s",'
  ||'"transiciones_sha256":"%s","version_esperada":"%s"}}',
   v_limites->>'ambito_ref',v_limites->>'unidad_ref',p_actor,p_catalogo_sha256,p_catalogo_version,
   encode(sha256(convert_to(p_clave,'UTF8')),'hex'),encode(sha256(convert_to(p_motivo,'UTF8')),'hex'),
   v_perfil_ref,encode(sha256(convert_to(array_to_string(v_canonica,','),'UTF8')),'hex'),p_version_esperada);
 IF v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(v_contexto,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B122: intención no ligada a V3' USING ERRCODE='42501'; END IF;
 v_ambito_sha256:=encode(sha256(convert_to(format('{"ambito_ref":"%s","unidad_ref":"%s"}',
  v_limites->>'ambito_ref',v_limites->>'unidad_ref'),'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion',0));
 v_huella:=encode(sha256(convert_to(p_version_esperada::text||chr(31)||array_to_string(v_canonica,',')
  ||chr(31)||p_motivo||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.politica_transiciones_gobierno WHERE clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previa.actor_ref IS DISTINCT FROM p_actor OR v_previa.version_esperada IS DISTINCT FROM p_version_esperada
     OR v_previa.contenido_sha256 IS DISTINCT FROM v_huella OR v_previa.recibo_ref IS DISTINCT FROM p_recibo
  THEN RAISE EXCEPTION 'B122: clave reutilizada' USING ERRCODE='VBT01'; END IF;
  IF v_previa.perfil_ref IS DISTINCT FROM v_perfil_ref OR v_previa.ambito_sha256 IS DISTINCT FROM v_ambito_sha256 THEN
   RAISE EXCEPTION 'B122: ámbito global denegado' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT v_anterior FROM vec_bolsa_llamamientos.politica_transiciones_situacion
   WHERE version=v_previa.version;
  IF v_anterior.transiciones IS DISTINCT FROM v_canonica
     OR v_anterior.catalogo_sha256 IS DISTINCT FROM v_previa.catalogo_sha256
     OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_transiciones_outbox o
                    WHERE o.recibo_ref=v_previa.recibo_ref AND o.version=v_previa.version
                      AND o.contenido_sha256=v_previa.contenido_sha256
                      AND o.evento->>'recibo_ref'=v_previa.recibo_ref
                      AND o.evento->>'version'=v_previa.version::text) THEN
   RAISE EXCEPTION 'B122: recibo incompleto' USING ERRCODE='55000'; END IF;
  RETURN QUERY SELECT jsonb_build_object('version',v_previa.version,
   'transiciones_guardadas',to_jsonb(v_anterior.transiciones),
   'limites',v_previa.limites_snapshot,'catalogo_activado',true,'recibo_ref',v_previa.recibo_ref),true;
  RETURN;
 END IF;
 SELECT array_agg(valor ORDER BY ordinal) INTO v_permitidas FROM jsonb_array_elements_text(v_limites->'transiciones') WITH ORDINALITY AS t(valor,ordinal);
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(v_permitidas) IS DISTINCT FROM v_permitidas
    OR NOT (v_canonica <@ v_permitidas) THEN
  RAISE EXCEPTION 'B122: propuesta fuera de límites normativos' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT v_vigente FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 IF v_vigente.version<>p_version_esperada THEN
  RAISE EXCEPTION 'B122: versión obsoleta' USING ERRCODE='VBT01'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_situacion(version,catalogo_ref,catalogo_sha256,transiciones,publicada_en)
 VALUES(v_vigente.version+1,v_limites->>'catalogo_ref',p_catalogo_sha256,v_canonica,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_gobierno
  (version,version_esperada,catalogo_version,catalogo_sha256,limites_snapshot,contexto_recurso_sha256,ambito_sha256,
   actor_ref,perfil_ref,motivo,clave_idempotencia,recibo_ref,
   contenido_sha256,decision_ref,auditoria_ref,publicada_en)
 VALUES(v_vigente.version+1,p_version_esperada,p_catalogo_version,p_catalogo_sha256,v_limites,
   v_decision->>'contexto_recurso_huella_sha256',v_ambito_sha256,p_actor,v_perfil_ref,p_motivo,p_clave,p_recibo,
   v_huella,v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_outbox
  (recibo_ref,version,contenido_sha256,esquema,evento,evento_sha256,creada_en)
 SELECT p_recibo,v_vigente.version+1,v_huella,'vec.bolsa.politica-transiciones.publicada.v1',e,
        encode(sha256(convert_to(e::text,'UTF8')),'hex'),v_ahora
 FROM (SELECT jsonb_build_object('esquema','vec.bolsa.politica-transiciones.publicada.v1',
   'recibo_ref',p_recibo,'version',v_vigente.version+1,'contenido_sha256',v_huella,
   'catalogo_ref',v_limites->>'catalogo_ref','catalogo_sha256',p_catalogo_sha256) e) v;
 RETURN QUERY SELECT vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(p_catalogo_version,p_catalogo_sha256),false;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(integer,text),
 vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(integer,text),
 vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
-- B32 sigue instalada para su historia, pero su publicador sin V3 deja de
-- estar disponible para el ejecutor de aplicación.
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[])
 FROM vec_bolsa_llamamientos_ejecutor;
DO $acl$
BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[])','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR (SELECT proowner FROM pg_proc WHERE oid=
      'vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=
      'vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=
      'vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s']
 THEN RAISE EXCEPTION 'B122: privilegios incompatibles' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
