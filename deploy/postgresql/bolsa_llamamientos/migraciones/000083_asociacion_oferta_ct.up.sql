\set ON_ERROR_STOP on
-- B83. RRHH asocia expresamente una plaza de una oferta real ya publicada
-- con un expediente CT. La asociación no altera la oferta ni adjudica persona.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000083', 0));

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_asociacion_oferta_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR pg_catalog.has_schema_privilege('vec_contratacion_temporal_propietario',
        'vec_bolsa_llamamientos','USAGE')
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.asociacion_oferta_ct') IS NOT NULL THEN
  RAISE EXCEPTION 'B83: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

-- No se reserva una candidatura: la unicidad se refiere solo a la plaza de
-- una oferta y al ciclo del expediente que RRHH vincula explícitamente.
CREATE TABLE vec_bolsa_llamamientos.asociacion_oferta_ct (
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 numero_plaza integer NOT NULL CHECK (numero_plaza BETWEEN 1 AND 100),
 bolsa_ref text NOT NULL,
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 version_esperada bigint NOT NULL CHECK (version_esperada > 0),
 ciclo_ref text NOT NULL,
 necesidad_ref text NOT NULL,
 categoria_ref text NOT NULL,
 unidad_ref text NOT NULL,
 ambito_ref text NOT NULL,
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 correlacion_ref text NOT NULL,
 intencion_sha256 text NOT NULL CHECK (intencion_sha256 ~ '^[0-9a-f]{64}$'),
 publicacion_sha256 text NOT NULL CHECK (publicacion_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_publicacion_ref text NOT NULL,
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 asociada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (oferta_ref, numero_plaza),
 UNIQUE (organizacion_ref, expediente_ref, ciclo_ref),
 UNIQUE (bolsa_ref, clave_idempotencia),
 CHECK (bolsa_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND ciclo_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND necesidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND categoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'
    AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND clave_idempotencia ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    AND recibo_publicacion_ref ~ '^recibo:oferta:[0-9a-f]{64}$'
    AND recibo_ref ~ '^recibo:asociacion-oferta-ct:[0-9a-f]{64}$'
    AND auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$')
);
CREATE TABLE vec_bolsa_llamamientos.asociacion_oferta_ct_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.asociacion_oferta_ct(recibo_ref),
 oferta_ref text NOT NULL,
 numero_plaza integer NOT NULL,
 expediente_ref text NOT NULL,
 material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
 creada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.asociacion_oferta_ct ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.asociacion_oferta_ct FORCE ROW LEVEL SECURITY;
CREATE POLICY asociacion_oferta_ct_propietario ON vec_bolsa_llamamientos.asociacion_oferta_ct
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
ALTER TABLE vec_bolsa_llamamientos.asociacion_oferta_ct_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.asociacion_oferta_ct_outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY asociacion_oferta_ct_outbox_propietario ON vec_bolsa_llamamientos.asociacion_oferta_ct_outbox
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.asociacion_oferta_ct,
 vec_bolsa_llamamientos.asociacion_oferta_ct_outbox FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.asociacion_oferta_ct,
 vec_bolsa_llamamientos.asociacion_oferta_ct_outbox FROM PUBLIC;
CREATE TRIGGER negar_mutacion_asociacion_oferta_ct
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.asociacion_oferta_ct
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER negar_mutacion_asociacion_oferta_ct_outbox
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.asociacion_oferta_ct_outbox
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.asociar_oferta_ct_v1(
 p_oferta text, p_plaza integer, p_bolsa text, p_organizacion text,
 p_expediente text, p_version_esperada bigint, p_ciclo text,
 p_necesidad text, p_categoria text, p_unidad text, p_ambito text,
 p_actor text, p_perfil text, p_clave text, p_correlacion text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea,
 p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(oferta_ref text, numero_plaza integer, bolsa_ref text,
 categoria_ref text, recibo_publicacion_ref text, publicacion_sha256 text,
 publicada_en timestamptz, recibo_asociacion_ref text, auditoria_ref text,
 asociada_en timestamptz, reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
SET statement_timeout='15s' AS $f$
DECLARE o record; b record; previa record; consumo record; c jsonb; d jsonb;
 v_ahora timestamptz; v_material text; v_huella text; v_permiso_huella text; v_contexto text;
 v_contexto_sha text; v_publicacion_sha text; v_recibo text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_plaza IS NULL OR p_plaza NOT BETWEEN 1 AND 100
    OR p_bolsa IS NULL OR p_bolsa !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_organizacion IS NULL OR p_organizacion !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_perfil IS NULL OR p_perfil !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_version_esperada IS NULL OR p_version_esperada <= 0 OR p_version_esperada > 9007199254740990
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    OR p_correlacion IS NULL OR p_correlacion !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_expediente IS NULL OR p_expediente !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_ciclo IS NULL OR p_ciclo !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_necesidad IS NULL OR p_necesidad !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_categoria IS NULL OR p_categoria !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' THEN
  RAISE EXCEPTION 'B83: solicitud inválida' USING ERRCODE='22023';
 END IF;
 -- La correlación identifica cada intento autorizado, no el efecto
 -- semántico: un replay puede traer otra correlación y conserva el recibo.
 v_material:=pg_catalog.array_to_string(ARRAY[p_oferta,p_plaza::text,p_bolsa,
  p_organizacion,p_expediente,p_version_esperada::text,p_ciclo,p_necesidad,
  p_categoria,p_unidad,p_ambito,p_actor,p_perfil,p_clave],pg_catalog.chr(31));
 v_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_material,'UTF8')),'hex');
 v_permiso_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.array_to_string(ARRAY[p_oferta,p_plaza::text,p_bolsa,p_organizacion,
   p_expediente,p_version_esperada::text,p_unidad,p_ambito,p_actor,p_perfil,
   p_clave,p_correlacion],pg_catalog.chr(31)),'UTF8')),'hex');
 v_contexto:='{"ambitos":{"ambito_ref":'||pg_catalog.to_json(p_ambito)::text||
  ',"organizacion_ref":'||pg_catalog.to_json(p_organizacion)::text||
  ',"unidad_ref":'||pg_catalog.to_json(p_unidad)::text||
  '},"atributos":{"asociacion_bolsa_sha256":'||pg_catalog.to_json(v_permiso_huella)::text||'}}';
 v_contexto_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_contexto,'UTF8')),'hex');
 BEGIN c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
       d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B83: decisión inválida' USING ERRCODE='42501'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.oferta_ct.asociar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.oferta.ct.asociar'
    OR c->>'efecto_ref' IS DISTINCT FROM p_oferta
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_contexto_sha
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
    OR d->>'accion' IS DISTINCT FROM 'bolsa.oferta.ct.asociar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'oferta_publicada'
    OR d->>'finalidad' IS DISTINCT FROM 'asociar_oferta_con_expediente_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM p_oferta
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_sha THEN
  RAISE EXCEPTION 'B83: asociación no autorizada' USING ERRCODE='42501';
 END IF;
 -- B58 toma este mismo cerrojo antes de registrar un acto sobre la plaza.
 -- Así un acto confirmado primero impide una asociación retrospectiva.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_asociacion_oferta_ct_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM p_oferta OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_sha
    OR consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'B83: consumo no ligado' USING ERRCODE='42501';
 END IF;
 v_ahora:=pg_catalog.clock_timestamp();
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('bolsa:asociacion-oferta:'||p_oferta||':'||p_plaza::text,0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('bolsa:asociacion-expediente:'||p_organizacion||':'||p_expediente||':'||p_ciclo,0));
 SELECT op.oferta_ref,op.recibo_ref,op.bolsa_ref,op.huella_comando_sha256,
        op.publicada_en,op.vence_antes_de,op.datos,op.plazo,
        po.numero_plazas,po.politica_version INTO o
 FROM vec_bolsa_llamamientos.oferta_publicada op
 JOIN vec_bolsa_llamamientos.plazas_oferta po USING(oferta_ref)
 WHERE op.oferta_ref=p_oferta AND op.bolsa_ref=p_bolsa;
 IF NOT FOUND OR p_plaza>o.numero_plazas OR o.publicada_en>v_ahora
    OR pg_catalog.jsonb_typeof(o.plazo->'notificacion') IS DISTINCT FROM 'object'
    OR o.plazo#>>'{notificacion,fuente}' IS DISTINCT FROM 'correo_externo_declarado_rrhh'
    OR pg_catalog.coalesce(o.plazo#>>'{notificacion,referencia_correo}','') !~ '^[A-Za-z0-9][A-Za-z0-9:._-]{0,191}$'
    OR pg_catalog.coalesce(o.plazo#>>'{notificacion,huella_correo_sha256}','') !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'B83: oferta no disponible' USING ERRCODE='23503';
 END IF;
 v_publicacion_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.jsonb_build_object('oferta_ref',o.oferta_ref,'recibo_ref',o.recibo_ref,
   'bolsa_ref',o.bolsa_ref,'datos',o.datos,'plazo',o.plazo,
   'publicada_en',o.publicada_en,'vence_antes_de',o.vence_antes_de,
   'numero_plazas',o.numero_plazas,'politica_version',o.politica_version)::text,
  'UTF8')),'hex');
 SELECT * INTO previa FROM vec_bolsa_llamamientos.asociacion_oferta_ct a
 WHERE (a.oferta_ref=p_oferta AND a.numero_plaza=p_plaza)
    OR (a.organizacion_ref=p_organizacion AND a.expediente_ref=p_expediente AND a.ciclo_ref=p_ciclo)
    OR (a.bolsa_ref=p_bolsa AND a.clave_idempotencia=p_clave)
 FOR SHARE;
 IF FOUND THEN
  IF previa.intencion_sha256 IS DISTINCT FROM v_huella
     OR previa.oferta_ref IS DISTINCT FROM p_oferta
     OR previa.numero_plaza IS DISTINCT FROM p_plaza
     OR previa.organizacion_ref IS DISTINCT FROM p_organizacion
     OR previa.expediente_ref IS DISTINCT FROM p_expediente
     OR previa.version_esperada IS DISTINCT FROM p_version_esperada
     OR previa.ciclo_ref IS DISTINCT FROM p_ciclo
     OR previa.publicacion_sha256 IS DISTINCT FROM v_publicacion_sha THEN
   RAISE EXCEPTION 'B83: asociación divergente' USING ERRCODE='23505';
  END IF;
  RETURN QUERY SELECT previa.oferta_ref,previa.numero_plaza,previa.bolsa_ref,
   previa.categoria_ref,previa.recibo_publicacion_ref,previa.publicacion_sha256,
  o.publicada_en,previa.recibo_ref,previa.auditoria_ref,previa.asociada_en,true;
  RETURN;
 END IF;
 SELECT c.categoria_ref INTO b FROM vec_bolsa_llamamientos.listar_constituciones_v1() c
 WHERE c.bolsa_ref=p_bolsa AND c.estado='vigente' AND c.vigente_desde<=v_ahora
   AND (c.vigente_hasta IS NULL OR v_ahora<c.vigente_hasta);
 IF NOT FOUND OR b.categoria_ref IS DISTINCT FROM p_categoria
    OR v_ahora>=o.vence_antes_de
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta a
               WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=p_plaza) THEN
  RAISE EXCEPTION 'B83: oferta o bolsa no disponible para asociación' USING ERRCODE='23503';
 END IF;
 -- Otro proceso pudo esperar la cadena de auditoría común: el plazo se
 -- reevalúa inmediatamente antes del efecto nuevo.
 v_ahora:=pg_catalog.clock_timestamp();
 IF v_ahora>=o.vence_antes_de THEN
  RAISE EXCEPTION 'B83: plazo de oferta vencido' USING ERRCODE='23503';
 END IF;
 v_recibo:='recibo:asociacion-oferta-ct:'||v_huella;
 INSERT INTO vec_bolsa_llamamientos.asociacion_oferta_ct VALUES(
  p_oferta,p_plaza,p_bolsa,p_organizacion,p_expediente,p_version_esperada,
  p_ciclo,p_necesidad,p_categoria,p_unidad,p_ambito,p_actor,p_perfil,p_clave,p_correlacion,
  v_huella,v_publicacion_sha,o.recibo_ref,
  v_recibo,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.asociacion_oferta_ct_outbox VALUES(
  v_recibo,p_oferta,p_plaza,p_expediente,v_huella,v_ahora);
 RETURN QUERY SELECT p_oferta,p_plaza,p_bolsa,p_categoria,o.recibo_ref,
  v_publicacion_sha,o.publicada_en,v_recibo,consumo.auditoria_ref,v_ahora,false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.asociar_oferta_ct_v1(
 text,integer,text,text,text,bigint,text,text,text,text,text,text,text,text,text,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.asociar_oferta_ct_v1(
 text,integer,text,text,text,bigint,text,text,text,text,text,text,text,text,text,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.asociar_oferta_ct_v1(
 text,integer,text,text,text,bigint,text,text,text,text,text,text,text,text,text,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_contratacion_temporal_propietario;
COMMIT;
