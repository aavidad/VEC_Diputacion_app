\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor efímero del ensayo AD3-59. Nunca se
-- instala en una base conservada. No crea claves, configuración ni decisiones:
-- toma de la propia preimagen el último alta CT ya consumida (replay real) y,
-- a partir de ella, material coherente para cada fachada de Dietas y D7.
BEGIN;
SET LOCAL search_path=pg_catalog;
CREATE SCHEMA prueba59 AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA prueba59 FROM PUBLIC;
CREATE TABLE prueba59.material(
 caso text PRIMARY KEY, capacidad bytea NOT NULL, decision bytea NOT NULL,
 motivo bytea NOT NULL, contexto bytea NOT NULL, persona_version numeric NOT NULL,
 perfil_version numeric NOT NULL, payload bytea NOT NULL, sobre bytea NOT NULL,
 evidencia bytea NOT NULL, raiz bytea NOT NULL);
REVOKE ALL ON prueba59.material FROM PUBLIC;

-- Alta CT consumida por el núcleo: su repetición idéntica recorre guarda,
-- prevalidación y ligadura del núcleo vigente y devuelve el consumo original.
INSERT INTO prueba59.material
SELECT 'ct_alta', t.capacidad_canonica, t.decision_canonica, t.motivo_canonico,
       t.contexto_actor_canonico,
       (convert_from(t.contexto_actor_canonico,'UTF8')::jsonb->>'persona_version')::numeric,
       (convert_from(t.contexto_actor_canonico,'UTF8')::jsonb->>'perfil_version')::numeric,
       t.payload_vec_ad_3, t.sobre_cose_sign1, t.evidencia_verificacion, t.raiz_publica_spki
  FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 t
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 a USING (decision_ref)
  JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u USING (decision_ref)
 WHERE convert_from(t.capacidad_canonica,'UTF8')::jsonb->>'operacion'='contratacion_temporal.solicitud.crear'
 ORDER BY t.registrada_en DESC, t.decision_ref LIMIT 1;
DO $x$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM prueba59.material WHERE caso='ct_alta') THEN
  RAISE EXCEPTION 'preimagen sin alta CT consumida';
 END IF;
END $x$;

-- Material nuevo para una fachada: operación, audiencia y decisión exactas de
-- su contrato y campos leídos de la propia fachada instalada. Sin clave para
-- esa audiencia, el núcleo solo puede detenerse después de la ligadura.
CREATE FUNCTION prueba59.preparar(p_caso text,p_fachada text,p_operacion text,
 p_audiencia text,p_modulo text,p_tipo text,p_finalidad text) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b prueba59.material; c jsonb; d jsonb; campos jsonb; decision bytea; efecto text; huella text;
BEGIN
 SELECT * INTO STRICT b FROM prueba59.material WHERE caso='ct_alta';
 SELECT substring(p.prosrc FROM 'campos jsonb:=''(\[[^'']*\])''')::jsonb INTO STRICT campos
   FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.'||p_fachada
     ||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF campos IS NULL THEN
  SELECT substring(p.prosrc FROM 'campos jsonb:=''(\[[^'']*\])''')::jsonb INTO STRICT campos
    FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 END IF;
 efecto:='prueba59:'||p_caso;
 huella:=encode(sha256(convert_to(efecto,'UTF8')),'hex');
 d:=convert_from(b.decision,'UTF8')::jsonb
   || jsonb_build_object('decision_ref','decision:prueba59:'||p_caso,'accion',p_operacion,
      'modulo_id',p_modulo,'tipo_recurso',p_tipo,'finalidad',p_finalidad,'recurso_ref',efecto,
      'contexto_recurso_huella_sha256',huella,'campos_permitidos',campos,'obligaciones','[]'::jsonb);
 decision:=convert_to(d::text,'UTF8');
 c:=convert_from(b.capacidad,'UTF8')::jsonb
   || jsonb_build_object('decision_ref',d->>'decision_ref','operacion',p_operacion,
      'audiencia_consumo',p_audiencia,'efecto_ref',efecto,'huella_efecto_sha256',huella,
      'huella_decision_sha256',encode(sha256(decision),'hex'),
      'nonce',encode(sha256(convert_to('nonce:prueba59:'||p_caso,'UTF8')),'hex'));
 INSERT INTO prueba59.material VALUES (p_caso,
  vec_autorizacion_atestada_v3.capacidad_canonica(c),decision,b.motivo,b.contexto,
  b.persona_version,b.perfil_version,b.payload,b.sobre,b.evidencia,b.raiz);
END $f$;

-- Un llamador por propietario nominal: el login de sesión es el que decide.
DO $llamadores$
DECLARE nombre text; propietario text;
BEGIN
 FOR nombre,propietario IN VALUES ('ct','vec_contratacion_temporal_propietario'),
   ('dietas','vec_dietas_propietario'),('personal','vec_personal_propietario') LOOP
  EXECUTE format($q$CREATE FUNCTION prueba59.consumir_%s(p_fachada text,p_caso text)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $c$
   DECLARE m prueba59.material;
   BEGIN
    SELECT * INTO STRICT m FROM prueba59.material WHERE caso=p_caso;
    RETURN QUERY EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',p_fachada)
     USING m.capacidad,m.decision,m.motivo,m.contexto,m.persona_version,m.perfil_version,
           m.payload,m.sobre,m.evidencia,m.raiz;
   END $c$$q$,nombre);
  EXECUTE format('REVOKE ALL ON FUNCTION prueba59.consumir_%s(text,text) FROM PUBLIC',nombre);
  EXECUTE format('ALTER FUNCTION prueba59.consumir_%s(text,text) OWNER TO %I',nombre,propietario);
  EXECUTE format('GRANT USAGE ON SCHEMA prueba59 TO %I',propietario);
  EXECUTE format('GRANT SELECT ON prueba59.material TO %I',propietario);
 END LOOP;
END $llamadores$;
COMMIT;
