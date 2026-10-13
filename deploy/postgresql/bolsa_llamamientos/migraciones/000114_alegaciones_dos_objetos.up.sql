\set ON_ERROR_STOP on
-- B114: recepción en VEC sobre acto/version exactos de Bolsa. No es registro oficial.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000114',0));
DO $pre$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_version') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.inscripcion_incorporacion') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(text[],text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_alegacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.alegacion_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'B114: preimagen o autoridad ausente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.alegacion_v1 (
 alegacion_ref text COLLATE "C" PRIMARY KEY CHECK(alegacion_ref~'^alegacion_[0-9a-f]{64}$'),
 persona_ref text COLLATE "C" NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 candidato_ref text COLLATE "C" NOT NULL CHECK(candidato_ref~'^can_[A-Za-z0-9_-]{22,128}$'),
 objeto_tipo text NOT NULL CHECK(objeto_tipo IN('decision_inscripcion','acto_participacion')),
 solicitud_ref text COLLATE "C" REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 participacion_ref text COLLATE "C" REFERENCES vec_bolsa_llamamientos.vinculo_candidato(participacion_ref),
 acto_ref text COLLATE "C" NOT NULL CHECK(octet_length(acto_ref) BETWEEN 3 AND 512),
 version_acto bigint NOT NULL CHECK(version_acto BETWEEN 1 AND 9007199254740991),
 convocatoria_ref text COLLATE "C",
 bolsa_ref text COLLATE "C",
 unidad_ref text COLLATE "C" NOT NULL,
 ambito_ref text COLLATE "C" NOT NULL,
 fundamento text NOT NULL CHECK(char_length(fundamento) BETWEEN 1 AND 4000 AND octet_length(fundamento)<=16000 AND btrim(fundamento)<>''),
 clave_sha256 text NOT NULL CHECK(clave_sha256~'^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 recibo_ref text COLLATE "C" NOT NULL UNIQUE CHECK(recibo_ref~'^recibo_alegacion_[0-9a-f]{64}$'),
 decision_ref text COLLATE "C" NOT NULL UNIQUE,
 auditoria_ref text COLLATE "C" NOT NULL UNIQUE,
 presentada_en timestamptz(6) NOT NULL CHECK(isfinite(presentada_en)),
 UNIQUE(persona_ref,clave_sha256),
 CHECK((objeto_tipo='decision_inscripcion' AND solicitud_ref IS NOT NULL AND participacion_ref IS NULL
       AND convocatoria_ref IS NOT NULL AND bolsa_ref IS NULL AND version_acto>=2)
    OR (objeto_tipo='acto_participacion' AND participacion_ref IS NOT NULL AND solicitud_ref IS NULL
       AND bolsa_ref IS NOT NULL AND convocatoria_ref IS NULL))
);
CREATE INDEX alegacion_v1_persona_pagina ON vec_bolsa_llamamientos.alegacion_v1(persona_ref,alegacion_ref);
CREATE INDEX alegacion_v1_rrhh_convocatoria ON vec_bolsa_llamamientos.alegacion_v1(unidad_ref,ambito_ref,convocatoria_ref,alegacion_ref);
CREATE INDEX alegacion_v1_rrhh_bolsa ON vec_bolsa_llamamientos.alegacion_v1(unidad_ref,ambito_ref,bolsa_ref,alegacion_ref);
CREATE TABLE vec_bolsa_llamamientos.alegacion_historia_v1 (
 historia_ref text COLLATE "C" PRIMARY KEY,
 alegacion_ref text COLLATE "C" NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.alegacion_v1(alegacion_ref),
 version bigint NOT NULL CHECK(version=1),
 estado text NOT NULL CHECK(estado='presentada'),
 actor_ref text COLLATE "C" NOT NULL,
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_bolsa_llamamientos.alegacion_outbox_v1 (
 evento_ref text COLLATE "C" PRIMARY KEY,
 historia_ref text COLLATE "C" NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.alegacion_historia_v1(historia_ref),
 alegacion_ref text COLLATE "C" NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.alegacion_v1(alegacion_ref),
 esquema text NOT NULL CHECK(esquema='vec.bolsa.alegacion.presentada.v1'),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_bolsa_llamamientos.alegacion_acceso_v1 (
 acceso_ref text COLLATE "C" PRIMARY KEY,
 alegacion_ref text COLLATE "C" REFERENCES vec_bolsa_llamamientos.alegacion_v1(alegacion_ref),
 persona_ref text COLLATE "C" NOT NULL,
 accion text NOT NULL CHECK(accion IN('presentar','consultar_propia','consultar_rrhh')),
 resultado text NOT NULL CHECK(resultado IN('confirmada','repetida','obtenida','no_encontrada')),
 auditoria_ref text COLLATE "C" NOT NULL UNIQUE,
 accedida_en timestamptz(6) NOT NULL
);

ALTER TABLE vec_bolsa_llamamientos.alegacion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alegacion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY alegacion_solo_propietario ON vec_bolsa_llamamientos.alegacion_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alegacion_v1 FROM PUBLIC;
CREATE TRIGGER alegacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_llamamientos.alegacion_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
ALTER TABLE vec_bolsa_llamamientos.alegacion_historia_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alegacion_historia_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY historia_alegacion_solo_propietario ON vec_bolsa_llamamientos.alegacion_historia_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alegacion_historia_v1 FROM PUBLIC;
CREATE TRIGGER historia_alegacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_llamamientos.alegacion_historia_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
ALTER TABLE vec_bolsa_llamamientos.alegacion_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alegacion_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY outbox_alegacion_solo_propietario ON vec_bolsa_llamamientos.alegacion_outbox_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alegacion_outbox_v1 FROM PUBLIC;
CREATE TRIGGER outbox_alegacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_llamamientos.alegacion_outbox_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
ALTER TABLE vec_bolsa_llamamientos.alegacion_acceso_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alegacion_acceso_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_alegacion_solo_propietario ON vec_bolsa_llamamientos.alegacion_acceso_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alegacion_acceso_v1 FROM PUBLIC;
CREATE TRIGGER acceso_alegacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_llamamientos.alegacion_acceso_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.presentar_alegacion_v1(
 p_material text,p_captura_actor jsonb,p_recurso_canonico bytea,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb; o jsonb; d jsonb; c jsonb; contexto jsonb; recurso jsonb; consumo record;
 v_persona text;v_candidato text;v_ref text;v_tipo text;v_acto text;v_sujeto text;v_version bigint;
 v_clave_sha text;v_material_sha text;v_recurso_sha text;v_recibo text;v_historia text;v_evento text;
 v_convocatoria text;v_bolsa text;v_unidad text;v_ambito text;v_conjunto jsonb;
 v_incorporacion text;v_existente vec_bolsa_llamamientos.alegacion_v1%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user<>'vec_externo_bolsa_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 32768
 OR p_recurso_canonico IS NULL OR octet_length(p_recurso_canonico) NOT BETWEEN 2 AND 8192
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B114: presentación no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;contexto:=convert_from(p_contexto,'UTF8')::jsonb;
  recurso:=convert_from(p_recurso_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'B114: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['clave_idempotencia','esquema','fundamento','objeto']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.alegacion.presentar.v1'
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9_-]{16,128}$'
 OR jsonb_typeof(m->'fundamento') IS DISTINCT FROM 'string'
 OR char_length(m->>'fundamento') NOT BETWEEN 1 AND 4000
 OR octet_length(m->>'fundamento')>16000 OR btrim(m->>'fundamento')=''
 OR jsonb_typeof(m->'objeto') IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B114: formulario inválido' USING ERRCODE='22023'; END IF;
 o:=m->'objeto';v_tipo:=o->>'tipo';v_acto:=o->>'acto_ref';
 IF jsonb_typeof(o->'version_acto') IS DISTINCT FROM 'number'
 OR coalesce(o->>'version_acto','') !~ '^[1-9][0-9]{0,15}$'
 OR (o->>'version_acto')::numeric>9007199254740991
 OR coalesce(v_acto,'') !~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,511}$'
 THEN RAISE EXCEPTION 'B114: acto inválido' USING ERRCODE='22023'; END IF;
 v_version:=(o->>'version_acto')::bigint;
 IF v_tipo='decision_inscripcion' THEN
  IF ARRAY(SELECT jsonb_object_keys(o) ORDER BY 1) IS DISTINCT FROM ARRAY['acto_ref','solicitud_ref','tipo','version_acto']
  OR coalesce(o->>'solicitud_ref','') !~ '^solicitud_inscripcion_[0-9a-f]{64}$' OR v_version<2
  THEN RAISE EXCEPTION 'B114: decisión inválida' USING ERRCODE='22023'; END IF;
  v_sujeto:=o->>'solicitud_ref';
 ELSIF v_tipo='acto_participacion' THEN
  IF ARRAY(SELECT jsonb_object_keys(o) ORDER BY 1) IS DISTINCT FROM ARRAY['acto_ref','participacion_ref','tipo','version_acto']
  OR coalesce(o->>'participacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,511}$'
  THEN RAISE EXCEPTION 'B114: participación inválida' USING ERRCODE='22023'; END IF;
  v_sujeto:=o->>'participacion_ref';
 ELSE RAISE EXCEPTION 'B114: tipo de objeto inválido' USING ERRCODE='22023'; END IF;
 v_persona:=d->>'principal_id';v_candidato:=recurso#>>'{ambitos,candidato_ref}';
 v_ref:='alegacion_'||encode(sha256(convert_to('vec.bolsa.alegacion.v1'||chr(31)||v_persona||chr(31)||m->>'clave_idempotencia','UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_recurso_sha:=encode(sha256(p_recurso_canonico),'hex');
 IF coalesce(v_persona,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(v_candidato,'') !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_persona
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR contexto->>'principal_ref' IS DISTINCT FROM v_persona
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.alegacion.presentar'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'alegacion_bolsa'
 OR d->>'finalidad' IS DISTINCT FROM 'presentacion_alegacion_propia'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.alegacion.presentar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.alegacion.presentar.v1'
 OR d->>'recurso_ref' IS DISTINCT FROM v_ref OR c->>'efecto_ref' IS DISTINCT FROM v_ref
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
 OR ARRAY(SELECT jsonb_object_keys(recurso) ORDER BY 1) IS DISTINCT FROM ARRAY['ambitos','atributos']
 OR jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'ambitos') ORDER BY 1) IS DISTINCT FROM ARRAY['candidato_ref']
 OR ARRAY(SELECT jsonb_object_keys(recurso->'atributos') ORDER BY 1) IS DISTINCT FROM
   (CASE WHEN v_tipo='decision_inscripcion' THEN
     ARRAY['acto_ref','material_sha256','solicitud_ref','tipo_objeto','version_acto']
   ELSE ARRAY['acto_ref','material_sha256','participacion_ref','tipo_objeto','version_acto'] END)
 OR recurso#>>'{atributos,tipo_objeto}' IS DISTINCT FROM v_tipo
 OR recurso#>>'{atributos,acto_ref}' IS DISTINCT FROM v_acto
 OR recurso#>>'{atributos,version_acto}' IS DISTINCT FROM v_version::text
 OR recurso#>>'{atributos,material_sha256}' IS DISTINCT FROM v_material_sha
 OR recurso#>>'{atributos,solicitud_ref}' IS DISTINCT FROM (CASE WHEN v_tipo='decision_inscripcion' THEN v_sujeto ELSE NULL END)
 OR recurso#>>'{atributos,participacion_ref}' IS DISTINCT FROM (CASE WHEN v_tipo='acto_participacion' THEN v_sujeto ELSE NULL END)
 OR jsonb_typeof(contexto->'vinculos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') x(v)
     WHERE x.v->>'tipo'='candidato' AND x.v->>'referencia'=v_candidato)<>1
 THEN RAISE EXCEPTION 'B114: actor, permiso o recurso divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:alegacion:'||v_persona||':'||v_clave_sha,0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_presentacion_alegacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_ref OR
    consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B114: consumo divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_existente FROM vec_bolsa_llamamientos.alegacion_v1
 WHERE persona_ref=v_persona AND clave_sha256=v_clave_sha;
 IF FOUND THEN
  IF v_existente.material_sha256 IS DISTINCT FROM v_material_sha OR v_existente.objeto_tipo IS DISTINCT FROM v_tipo
    OR v_existente.acto_ref IS DISTINCT FROM v_acto OR v_existente.version_acto IS DISTINCT FROM v_version
    OR coalesce(v_existente.solicitud_ref,v_existente.participacion_ref) IS DISTINCT FROM v_sujeto
  THEN RAISE EXCEPTION 'B114: clave reutilizada con otro material' USING ERRCODE='23505'; END IF;
  INSERT INTO vec_bolsa_llamamientos.alegacion_acceso_v1(acceso_ref,alegacion_ref,persona_ref,accion,resultado,auditoria_ref,accedida_en)
  VALUES('acceso_alegacion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),v_ref,v_persona,'presentar','repetida',consumo.auditoria_ref,consumo.consumida_en);
  RETURN jsonb_build_object('alegacion_ref',v_ref,'objeto',o,'fundamento',v_existente.fundamento,
   'estado','presentada','presentada_en',v_existente.presentada_en,'recibo_ref',v_existente.recibo_ref,
   'auditoria_ref',v_existente.auditoria_ref,'repetida',true);
 END IF;
 IF v_tipo='decision_inscripcion' THEN
  SELECT s.convocatoria_ref,s.unidad_ref,s.ambito_ref INTO v_convocatoria,v_unidad,v_ambito
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_version sv ON sv.solicitud_ref=s.solicitud_ref
  WHERE s.solicitud_ref=v_sujeto AND s.persona_ref=v_persona
    AND sv.version=v_version AND sv.decision_ref=v_acto
    AND sv.estado IN('admitida_a_convocatoria','rechazada')
  FOR SHARE OF s,sv;
 ELSE
  SELECT c.bolsa_ref,i.solicitud_ref,s.unidad_ref,s.ambito_ref
   INTO v_bolsa,v_incorporacion,v_unidad,v_ambito
  FROM vec_bolsa_llamamientos.vinculo_candidato vc
  JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=vc.acta_ref
  LEFT JOIN vec_bolsa_llamamientos.inscripcion_incorporacion i ON i.participacion_ref=vc.participacion_ref
  LEFT JOIN vec_bolsa_llamamientos.solicitud_inscripcion s ON s.solicitud_ref=i.solicitud_ref
  WHERE vc.participacion_ref=v_sujeto AND vc.candidato_ref=v_candidato
    AND vc.acta_ref=v_acto AND vc.version_instantanea=v_version
    AND (i.participacion_ref IS NULL OR (s.persona_ref=v_persona AND i.persona_ref=v_persona))
  FOR SHARE OF vc,c;
  IF FOUND AND v_incorporacion IS NULL THEN
   v_conjunto:=vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
   v_unidad:=v_conjunto->>'unidad_ref';v_ambito:=v_conjunto->>'ambito_ref';
  END IF;
 END IF;
 IF NOT FOUND OR v_unidad IS NULL OR v_ambito IS NULL
 THEN RAISE EXCEPTION 'B114: acto ajeno o ámbito ausente' USING ERRCODE='42501'; END IF;
 v_historia:='historia_alegacion_'||encode(sha256(convert_to(v_ref||':1','UTF8')),'hex');
 v_evento:='evento_alegacion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
 v_recibo:='recibo_alegacion_'||encode(sha256(convert_to(v_ref||chr(31)||v_material_sha,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.alegacion_v1(
  alegacion_ref,persona_ref,candidato_ref,objeto_tipo,solicitud_ref,participacion_ref,acto_ref,version_acto,
  convocatoria_ref,bolsa_ref,unidad_ref,ambito_ref,fundamento,clave_sha256,material_sha256,
  recibo_ref,decision_ref,auditoria_ref,presentada_en)
 VALUES(v_ref,v_persona,v_candidato,v_tipo,
  CASE WHEN v_tipo='decision_inscripcion' THEN v_sujeto ELSE NULL END,
  CASE WHEN v_tipo='acto_participacion' THEN v_sujeto ELSE NULL END,
  v_acto,v_version,v_convocatoria,v_bolsa,v_unidad,v_ambito,m->>'fundamento',v_clave_sha,v_material_sha,
  v_recibo,consumo.decision_ref,consumo.auditoria_ref,consumo.consumida_en);
 INSERT INTO vec_bolsa_llamamientos.alegacion_historia_v1(historia_ref,alegacion_ref,version,estado,actor_ref,registrada_en)
 VALUES(v_historia,v_ref,1,'presentada',v_persona,consumo.consumida_en);
 INSERT INTO vec_bolsa_llamamientos.alegacion_outbox_v1(evento_ref,historia_ref,alegacion_ref,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_ref,'vec.bolsa.alegacion.presentada.v1',
  jsonb_build_object('alegacion_ref',v_ref,'objeto',o,'estado','presentada','recibo_ref',v_recibo),consumo.consumida_en);
 INSERT INTO vec_bolsa_llamamientos.alegacion_acceso_v1(acceso_ref,alegacion_ref,persona_ref,accion,resultado,auditoria_ref,accedida_en)
 VALUES('acceso_alegacion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),v_ref,v_persona,'presentar','confirmada',consumo.auditoria_ref,consumo.consumida_en);
 RETURN jsonb_build_object('alegacion_ref',v_ref,'objeto',o,'fundamento',m->>'fundamento',
  'estado','presentada','presentada_en',consumo.consumida_en,'recibo_ref',v_recibo,
  'auditoria_ref',consumo.auditoria_ref,'repetida',false);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.presentar_alegacion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.presentar_alegacion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_portal_externo;

CREATE FUNCTION vec_bolsa_llamamientos.proyectar_alegacion_v1(a vec_bolsa_llamamientos.alegacion_v1)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT jsonb_build_object('alegacion_ref',a.alegacion_ref,
  'objeto',jsonb_strip_nulls(jsonb_build_object('tipo',a.objeto_tipo,
    'solicitud_ref',a.solicitud_ref,'participacion_ref',a.participacion_ref,
    'acto_ref',a.acto_ref,'version_acto',a.version_acto)),
  'fundamento',a.fundamento,'estado','presentada','presentada_en',a.presentada_en,
  'recibo_ref',a.recibo_ref,'auditoria_ref',a.auditoria_ref)
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_alegacion_v1(vec_bolsa_llamamientos.alegacion_v1) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_alegaciones_v1(
 p_accion text,p_selector jsonb,p_contexto_canonico bytea,p_vinculo_canonico bytea,p_captura jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE v_contexto jsonb;v_vinculo jsonb;v_proyeccion jsonb;v_orden jsonb;v_resultado text:='obtenida';
 v_persona text;v_candidato text;v_tipo text;v_ref text;v_cursor text;v_canal text;v_unidad text;v_ambito text;
 v_limite integer;v_total bigint;v_siguiente text;v_auditoria record;v_ambito_actual jsonb;
 v_refs text[];v_metadatos jsonb;v_sujetos_b7 jsonb:='{}'::jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user NOT IN('vec_bolsa_inscripciones_lector','vec_bolsa_inscripciones_rrhh_lector')
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_selector) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['cursor','limite','ref','tipo']
 OR jsonb_typeof(p_captura) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B114: lectura no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'B114: identidad lectora inválida' USING ERRCODE='42501'; END;
 v_persona:=v_contexto->>'principal_ref';v_canal:=p_captura->>'canal';
 v_tipo:=p_selector->>'tipo';v_ref:=p_selector->>'ref';v_cursor:=p_selector->>'cursor';
 IF jsonb_typeof(p_selector->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_selector->>'limite','') !~ '^(?:[1-9]|1[0-9]|20)$'
 OR coalesce(v_cursor,'') !~ '^[A-Za-z0-9:._-]{0,255}$'
 OR p_captura->>'persona_ref' IS DISTINCT FROM v_persona
 OR p_captura->>'perfil_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR p_captura->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR p_captura->>'sesion_ref' IS DISTINCT FROM v_vinculo->>'sesion_ref'
 OR p_captura->>'autenticacion_ref' IS DISTINCT FROM v_vinculo->>'autenticacion_ref'
 OR p_captura->>'accion' IS DISTINCT FROM p_accion
 OR coalesce(p_captura->>'idioma','') NOT IN('es','en')
 OR p_captura->>'finalidad' IS DISTINCT FROM (CASE WHEN v_canal='externa_personal'
   THEN 'consulta_alegaciones_propias' ELSE 'consulta_alegaciones_rrhh' END)
 OR p_captura#>>'{filtro,limite}' IS DISTINCT FROM p_selector->>'limite'
 OR p_captura#>>'{filtro,cursor}' IS DISTINCT FROM v_cursor
 OR p_captura#>>'{filtro,estado}' IS DISTINCT FROM ''
 OR p_captura#>>'{filtro,convocatoria_ref}' IS DISTINCT FROM ''
 OR coalesce(p_captura->>'recurso_ref','') !~ '^alegaciones_lectura_[0-9a-f]{64}$'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_persona
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'superficie' IS DISTINCT FROM v_canal
 OR v_vinculo->>'metodo_observado' NOT IN('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 THEN RAISE EXCEPTION 'B114: captura de lectura divergente' USING ERRCODE='42501'; END IF;
 v_limite:=(p_selector->>'limite')::integer;
 IF v_canal='externa_personal' THEN
  IF session_user<>'vec_bolsa_inscripciones_lector'
  OR p_accion IS DISTINCT FROM 'bolsa.alegacion.propias.consultar'
  OR v_ref NOT IN('objetos','propias')
  OR p_captura ? 'conjunto_gestion' OR p_captura ? 'ambito_solicitud'
  OR jsonb_typeof(v_contexto->'vinculos') IS DISTINCT FROM 'array'
  THEN RAISE EXCEPTION 'B114: lector propio divergente' USING ERRCODE='42501'; END IF;
  SELECT x.v->>'referencia' INTO v_candidato
  FROM jsonb_array_elements(v_contexto->'vinculos') x(v)
  WHERE x.v->>'tipo'='candidato';
  IF v_candidato IS NULL OR v_candidato !~ '^can_[A-Za-z0-9_-]{22,128}$'
  OR (SELECT count(*) FROM jsonb_array_elements(v_contexto->'vinculos') x(v)
      WHERE x.v->>'tipo'='candidato')<>1
  THEN RAISE EXCEPTION 'B114: candidato no acreditado' USING ERRCODE='42501'; END IF;
 ELSIF v_canal='interna_corporativa' THEN
  IF session_user<>'vec_bolsa_inscripciones_rrhh_lector'
  OR p_accion IS DISTINCT FROM 'bolsa.inscripcion.alegacion.rrhh.consultar'
  OR jsonb_typeof(p_captura->'conjunto_gestion') IS DISTINCT FROM 'object'
  OR p_captura ? 'ambito_solicitud'
  THEN RAISE EXCEPTION 'B114: lector RRHH divergente' USING ERRCODE='42501'; END IF;
  v_ambito_actual:=vec_bolsa_llamamientos.resolver_conjunto_rrhh_inscripcion_previa_v1();
  v_unidad:=v_ambito_actual->>'unidad_ref';v_ambito:=v_ambito_actual->>'ambito_ref';
  IF v_unidad IS NULL OR v_ambito IS NULL
  OR p_captura#>>'{conjunto_gestion,unidad_ref}' IS DISTINCT FROM v_unidad
  OR p_captura#>>'{conjunto_gestion,ambito_ref}' IS DISTINCT FROM v_ambito
  OR p_captura#>>'{conjunto_gestion,fuente_ref}' IS DISTINCT FROM v_ambito_actual->>'fuente_ref'
  OR p_captura#>>'{conjunto_gestion,fuente_version}' IS DISTINCT FROM v_ambito_actual->>'fuente_version'
  OR p_captura#>>'{conjunto_gestion,fuente_sha256}' IS DISTINCT FROM v_ambito_actual->>'fuente_sha256'
  THEN RAISE EXCEPTION 'B114: ámbito RRHH divergente' USING ERRCODE='42501'; END IF;
 ELSE RAISE EXCEPTION 'B114: canal desconocido' USING ERRCODE='42501'; END IF;
 IF v_tipo='objetos' AND v_canal='externa_personal' AND v_ref='objetos' THEN
  WITH disponibles AS (
   SELECT 'objeto_decision_'||encode(sha256(convert_to(sv.solicitud_ref||':'||sv.decision_ref||':'||sv.version,'UTF8')),'hex') clave,
    jsonb_build_object('tipo','decision_inscripcion','solicitud_ref',sv.solicitud_ref,
      'acto_ref',sv.decision_ref,'version_acto',sv.version,'titulo','',
      'estado',sv.estado,'ocurrido_en',sv.aplicada_en,'convocatoria_ref',s.convocatoria_ref) dato
   FROM vec_bolsa_llamamientos.solicitud_inscripcion s
   JOIN vec_bolsa_llamamientos.solicitud_inscripcion_version sv ON sv.solicitud_ref=s.solicitud_ref
   WHERE s.persona_ref=v_persona AND sv.estado IN('admitida_a_convocatoria','rechazada')
   UNION ALL
   SELECT 'objeto_participacion_'||encode(sha256(convert_to(vc.participacion_ref||':'||vc.acta_ref||':'||vc.version_instantanea,'UTF8')),'hex') clave,
    jsonb_build_object('tipo','acto_participacion','participacion_ref',vc.participacion_ref,
      'acto_ref',vc.acta_ref,'version_acto',vc.version_instantanea,
      'titulo',to_char(c.confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'estado','constituida','ocurrido_en',vc.registrada_en,'bolsa_ref',c.bolsa_ref) dato
   FROM vec_bolsa_llamamientos.vinculo_candidato vc
   JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=vc.acta_ref
   LEFT JOIN vec_bolsa_llamamientos.inscripcion_incorporacion i ON i.participacion_ref=vc.participacion_ref
   LEFT JOIN vec_bolsa_llamamientos.solicitud_inscripcion s ON s.solicitud_ref=i.solicitud_ref
   WHERE vc.candidato_ref=v_candidato
     AND (i.participacion_ref IS NULL OR (s.persona_ref=v_persona AND i.persona_ref=v_persona))
  ), pagina AS (SELECT clave,dato,row_number() OVER(ORDER BY clave) n
    FROM disponibles WHERE v_cursor='' OR clave>v_cursor ORDER BY clave LIMIT v_limite+1)
  SELECT (SELECT count(*) FROM disponibles),
   coalesce((SELECT jsonb_agg(dato ORDER BY clave) FROM pagina WHERE n<=v_limite),'[]'::jsonb),
   CASE WHEN EXISTS(SELECT 1 FROM pagina WHERE n>v_limite)
     THEN (SELECT max(clave) FROM pagina WHERE n<=v_limite) ELSE NULL END
  INTO v_total,v_proyeccion,v_siguiente;
  SELECT coalesce(array_agg(q.ref ORDER BY q.ref),ARRAY[]::text[]) INTO v_refs
  FROM (SELECT DISTINCT x.valor->>'convocatoria_ref' ref
    FROM jsonb_array_elements(v_proyeccion) x(valor)
    WHERE x.valor->>'tipo'='decision_inscripcion') q;
  IF cardinality(v_refs)>0 THEN
   v_metadatos:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
    v_refs,p_captura->>'idioma');
   IF jsonb_typeof(v_metadatos) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_metadatos)<>cardinality(v_refs)
   OR EXISTS(SELECT 1 FROM jsonb_array_elements(v_metadatos) WITH ORDINALITY x(valor,orden)
      WHERE x.valor->>'convocatoria_ref' IS DISTINCT FROM v_refs[x.orden]
       OR coalesce(x.valor->>'titulo','')='' OR char_length(x.valor->>'titulo')>180)
   THEN RAISE EXCEPTION 'B114: título publicado ausente' USING ERRCODE='55000'; END IF;
  ELSE v_metadatos:='[]'::jsonb; END IF;
  SELECT coalesce(jsonb_agg(CASE WHEN x.valor->>'tipo'='decision_inscripcion'
    THEN jsonb_set(x.valor,'{titulo}',to_jsonb((SELECT m.valor->>'titulo'
      FROM jsonb_array_elements(v_metadatos) m(valor)
      WHERE m.valor->>'convocatoria_ref'=x.valor->>'convocatoria_ref')))
    ELSE x.valor END ORDER BY x.orden),'[]'::jsonb)
  INTO v_proyeccion FROM jsonb_array_elements(v_proyeccion) WITH ORDINALITY x(valor,orden);
  v_proyeccion:=jsonb_build_object('objetos',v_proyeccion,'total',v_total,'cursor_siguiente',v_siguiente);
 ELSIF v_tipo='propias' AND v_canal='externa_personal' AND v_ref='propias' THEN
  SELECT count(*) INTO v_total FROM vec_bolsa_llamamientos.alegacion_v1 WHERE persona_ref=v_persona;
  SELECT coalesce(jsonb_agg(a.dato ORDER BY a.alegacion_ref)
    FILTER(WHERE a.n<=v_limite),'[]'::jsonb),
   CASE WHEN count(*)>v_limite THEN max(a.alegacion_ref) FILTER(WHERE a.n<=v_limite) ELSE NULL END
  INTO v_proyeccion,v_siguiente FROM (
   SELECT a.alegacion_ref,vec_bolsa_llamamientos.proyectar_alegacion_v1(a) dato,
    row_number() OVER(ORDER BY a.alegacion_ref) n
   FROM vec_bolsa_llamamientos.alegacion_v1 a WHERE a.persona_ref=v_persona
     AND (v_cursor='' OR a.alegacion_ref>v_cursor)
   ORDER BY a.alegacion_ref LIMIT v_limite+1) a;
  v_proyeccion:=jsonb_build_object('alegaciones',v_proyeccion,'total',v_total,'cursor_siguiente',v_siguiente);
 ELSIF v_tipo IN('convocatoria','bolsa') AND v_canal='interna_corporativa'
   AND starts_with(v_ref,v_tipo||':') AND length(v_ref)>length(v_tipo)+1 THEN
  -- La referencia procede del filtro exacto ya autorizado; toda fila conserva
  -- el ámbito de la fuente histórica que originó el acto.
  IF v_tipo='convocatoria' THEN
   SELECT count(*) INTO v_total FROM vec_bolsa_llamamientos.alegacion_v1 a
    WHERE a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito AND a.convocatoria_ref=substr(v_ref,14);
   SELECT coalesce(jsonb_agg(a.dato ORDER BY a.alegacion_ref)
     FILTER(WHERE a.n<=v_limite),'[]'::jsonb),
     CASE WHEN count(*)>v_limite THEN max(a.alegacion_ref) FILTER(WHERE a.n<=v_limite) ELSE NULL END
    INTO v_proyeccion,v_siguiente FROM (SELECT a.alegacion_ref,
     vec_bolsa_llamamientos.proyectar_alegacion_v1(a) dato,
     row_number() OVER(ORDER BY a.alegacion_ref) n
     FROM vec_bolsa_llamamientos.alegacion_v1 a
    WHERE a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito AND a.convocatoria_ref=substr(v_ref,14)
      AND (v_cursor='' OR a.alegacion_ref>v_cursor)
    ORDER BY a.alegacion_ref LIMIT v_limite+1) a;
  ELSE
   SELECT count(*) INTO v_total FROM vec_bolsa_llamamientos.alegacion_v1 a
    WHERE a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito AND a.bolsa_ref=substr(v_ref,7);
   SELECT coalesce(jsonb_agg(a.dato ORDER BY a.alegacion_ref)
     FILTER(WHERE a.n<=v_limite),'[]'::jsonb),
     CASE WHEN count(*)>v_limite THEN max(a.alegacion_ref) FILTER(WHERE a.n<=v_limite) ELSE NULL END
    INTO v_proyeccion,v_siguiente FROM (SELECT a.alegacion_ref,
     vec_bolsa_llamamientos.proyectar_alegacion_v1(a) dato,
     row_number() OVER(ORDER BY a.alegacion_ref) n
     FROM vec_bolsa_llamamientos.alegacion_v1 a
    WHERE a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito AND a.bolsa_ref=substr(v_ref,7)
      AND (v_cursor='' OR a.alegacion_ref>v_cursor)
    ORDER BY a.alegacion_ref LIMIT v_limite+1) a;
  END IF;
  v_proyeccion:=jsonb_build_object('alegaciones',v_proyeccion,'total',v_total,'cursor_siguiente',v_siguiente);
 ELSIF v_tipo='ambitos' AND v_canal='interna_corporativa' AND v_ref='ambitos' THEN
  WITH x AS (
   SELECT 'convocatoria' tipo,convocatoria_ref referencia,NULL::timestamptz confirmada_en
    FROM vec_bolsa_llamamientos.alegacion_v1
    WHERE unidad_ref=v_unidad AND ambito_ref=v_ambito AND convocatoria_ref IS NOT NULL
   UNION
   SELECT 'bolsa',a.bolsa_ref,max(c.confirmada_en)
    FROM vec_bolsa_llamamientos.alegacion_v1 a
    JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=a.acto_ref
    WHERE a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito AND a.bolsa_ref IS NOT NULL
    GROUP BY a.bolsa_ref
  ), datos AS (SELECT tipo,referencia,confirmada_en,tipo||':'||referencia clave FROM x),
  pagina AS (SELECT *,row_number() OVER(ORDER BY clave) n FROM datos
   WHERE v_cursor='' OR clave>v_cursor ORDER BY clave LIMIT v_limite+1)
  SELECT (SELECT count(*) FROM datos),
   coalesce((SELECT jsonb_agg(jsonb_build_object('tipo',tipo,'referencia',referencia,
    'titulo',CASE WHEN tipo='bolsa' THEN to_char(confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') ELSE '' END) ORDER BY clave)
    FROM pagina WHERE n<=v_limite),'[]'::jsonb),
   CASE WHEN EXISTS(SELECT 1 FROM pagina WHERE n>v_limite)
     THEN (SELECT max(clave) FROM pagina WHERE n<=v_limite) ELSE NULL END
  INTO v_total,v_proyeccion,v_siguiente;
  SELECT coalesce(array_agg(q.ref ORDER BY q.ref),ARRAY[]::text[]) INTO v_refs
  FROM (SELECT DISTINCT x.valor->>'referencia' ref
    FROM jsonb_array_elements(v_proyeccion) x(valor)
    WHERE x.valor->>'tipo'='convocatoria') q;
  IF cardinality(v_refs)>0 THEN
   v_metadatos:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
    v_refs,p_captura->>'idioma');
   IF jsonb_typeof(v_metadatos) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_metadatos)<>cardinality(v_refs)
   OR EXISTS(SELECT 1 FROM jsonb_array_elements(v_metadatos) WITH ORDINALITY x(valor,orden)
      WHERE x.valor->>'convocatoria_ref' IS DISTINCT FROM v_refs[x.orden]
       OR coalesce(x.valor->>'titulo','')='' OR char_length(x.valor->>'titulo')>180)
   THEN RAISE EXCEPTION 'B114: título RRHH publicado ausente' USING ERRCODE='55000'; END IF;
  ELSE v_metadatos:='[]'::jsonb; END IF;
  SELECT coalesce(jsonb_agg(CASE WHEN x.valor->>'tipo'='convocatoria'
    THEN jsonb_set(x.valor,'{titulo}',to_jsonb((SELECT m.valor->>'titulo'
      FROM jsonb_array_elements(v_metadatos) m(valor)
      WHERE m.valor->>'convocatoria_ref'=x.valor->>'referencia')))
    ELSE x.valor END ORDER BY x.orden),'[]'::jsonb)
  INTO v_proyeccion FROM jsonb_array_elements(v_proyeccion) WITH ORDINALITY x(valor,orden);
  v_proyeccion:=jsonb_build_object('ambitos',v_proyeccion,'total',v_total,'cursor_siguiente',v_siguiente);
 ELSIF v_tipo='detalle' AND v_canal='interna_corporativa' AND v_ref~'^alegacion_[0-9a-f]{64}$' THEN
  SELECT vec_bolsa_llamamientos.proyectar_alegacion_v1(a) INTO v_proyeccion
  FROM vec_bolsa_llamamientos.alegacion_v1 a
  WHERE a.alegacion_ref=v_ref AND a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito;
  IF NOT FOUND THEN v_resultado:='no_encontrada';END IF;
 ELSE RAISE EXCEPTION 'B114: selector no admitido' USING ERRCODE='42501';END IF;
 -- Sobre privado de la página RRHH. Sólo el acto B7 todavía vigente y su
 -- fila exacta aportan procedencia para descifrar el nombre fuera de SQL.
 -- No forma parte de proyeccion ni de los campos enviados por HTTP.
 IF v_canal='interna_corporativa' AND v_resultado='obtenida' AND v_tipo IN('bolsa','detalle') THEN
  IF v_tipo='detalle' THEN
   v_refs:=ARRAY[v_ref];
  ELSE
   SELECT coalesce(array_agg(x.valor->>'alegacion_ref'),ARRAY[]::text[]) INTO v_refs
    FROM jsonb_array_elements(v_proyeccion->'alegaciones') x(valor);
  END IF;
  SELECT coalesce(jsonb_object_agg(a.alegacion_ref,jsonb_build_object(
    'acta_ref',c.acta_ref,'participacion_ref',vc.participacion_ref,
    'huella_listado_sha256',(convert_from(i.instantanea_canonica,'UTF8')::jsonb)->>'huella_listado_sha256',
    'categoria_ref',c.categoria_ref,'fila_numero',e.fila_numero)),'{}'::jsonb)
   INTO v_sujetos_b7
  FROM vec_bolsa_llamamientos.alegacion_v1 a
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=a.participacion_ref AND vc.candidato_ref=a.candidato_ref
    AND vc.acta_ref=a.acto_ref AND vc.version_instantanea=a.version_acto
  JOIN vec_bolsa_llamamientos.constitucion c
   ON c.acta_ref=vc.acta_ref AND c.bolsa_ref=a.bolsa_ref
    AND c.version_instantanea=vc.version_instantanea
  JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
   ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
    AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=i.instantanea_ref AND e.version_instantanea=i.version
    AND e.participacion_ref=vc.participacion_ref
  WHERE a.alegacion_ref=ANY(v_refs) AND a.objeto_tipo='acto_participacion'
   AND a.unidad_ref=v_unidad AND a.ambito_ref=v_ambito
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion posterior
    WHERE posterior.categoria_ref=c.categoria_ref
     AND (posterior.confirmada_en,posterior.registrada_en)>(c.confirmada_en,c.registrada_en))
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion empate
    WHERE empate.categoria_ref=c.categoria_ref AND empate.acta_ref<>c.acta_ref
     AND empate.confirmada_en=c.confirmada_en AND empate.registrada_en=c.registrada_en);
 END IF;
 v_orden:=jsonb_build_object(
  'intento_ref',p_captura->>'intento_ref','registro_contexto_ref',v_vinculo->>'registro_contexto_ref',
  'contexto_sha256',v_vinculo->>'contexto_actor_huella_sha256',
  'procedencia_sha256',v_vinculo->>'manifiesto_procedencia_huella_sha256',
  'autenticacion_ref',v_vinculo->>'autenticacion_ref','sesion_ref',v_vinculo->>'sesion_ref',
  'autenticacion_sha256',v_vinculo->>'autenticacion_huella_sha256',
  'lectura_revision_permisos',p_captura->>'revision_permisos',
  'lectura_instantanea_sha256',p_captura->>'huella_instantanea_sha256',
  'accion',p_accion,'modulo_id','bolsa','recurso_ref',p_captura->>'recurso_ref',
  'finalidad_ref',p_captura->>'finalidad','resultado',v_resultado,
  'motivo_ref',CASE WHEN v_resultado='obtenida' THEN 'inscripcion_lectura_correcta' ELSE 'inscripcion_no_encontrada' END,
  'proceso','vec-server','canal',v_canal,'correlacion_ref',p_captura->>'correlacion_ref');
 SELECT * INTO STRICT v_auditoria FROM vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
  p_contexto_canonico,p_vinculo_canonico,v_orden);
 IF v_auditoria.auditoria_ref IS NULL OR v_auditoria.correlacion_ref IS DISTINCT FROM p_captura->>'correlacion_ref'
 THEN RAISE EXCEPTION 'B114: asiento de lectura incompatible' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.alegacion_acceso_v1(acceso_ref,alegacion_ref,persona_ref,accion,resultado,auditoria_ref,accedida_en)
 VALUES('acceso_alegacion_'||encode(sha256(convert_to(v_auditoria.auditoria_ref,'UTF8')),'hex'),
  CASE WHEN v_tipo='detalle' AND v_resultado='obtenida' THEN v_ref ELSE NULL END,
  v_persona,CASE WHEN v_canal='externa_personal' THEN 'consultar_propia' ELSE 'consultar_rrhh' END,
  v_resultado,v_auditoria.auditoria_ref,v_auditoria.registrada_en);
 RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'sujetos_b7',v_sujetos_b7,
  'auditoria_ref',v_auditoria.auditoria_ref,'correlacion_ref',v_auditoria.correlacion_ref,
  'consultada_en',v_auditoria.registrada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_alegaciones_v1(text,jsonb,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_alegaciones_v1(text,jsonb,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_lector_inscripciones,vec_bolsa_llamamientos_lector_inscripciones_rrhh;

COMMIT;
