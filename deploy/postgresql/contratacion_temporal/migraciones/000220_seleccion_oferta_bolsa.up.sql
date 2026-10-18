\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000220',0));
DO $pre$
DECLARE v_huellas text[];
BEGIN
 SELECT array_agg(encode(sha256(convert_to(pg_get_constraintdef(oid,false),'UTF8')),'hex') ORDER BY conname)
 INTO v_huellas FROM pg_constraint
 WHERE conrelid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
 AND conname IN ('ejecucion_seleccion_llamamiento_o6_check',
  'ejecucion_seleccion_llamamiento_o6_check2','ejecucion_seleccion_llamamiento_o6_check3',
  'ejecucion_seleccion_llamamiento_o6_efecto_check',
  'ejecucion_seleccion_llamamiento_o6_reserva_ref_check',
  'ejecucion_seleccion_llamamiento_o6_situacion_check');
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR v_huellas IS DISTINCT FROM ARRAY[
      'c46cde9929f105aeb80e9c7f0be2ab8f8687bc69f26654c041e87bb3387c9a09',
      '6ad527bf6af8162c07708b028c9f54e88a15fe3274e4bfb31b3ff94dc68c9953',
      '68329881d9dd62d7c29e86a3ea64171784434134c4180e68bef178dd34ceca60',
      '8fc44873d1b7d085dd11437727158cd6153ef983839b65432820acf7c4f5e257',
      '28328fb44b1d35293ee64582a2ee9ef2c531cad1970e06d3ace2f2ca3e682016',
      'e25366f3919f19ab8338e8fa68a43e36a6dc858f6ed097fc5408a63f01d9cdfb']
    OR NOT EXISTS(SELECT 1 FROM pg_class c
      WHERE c.oid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
        AND c.relowner='vec_contratacion_temporal_propietario'::regrole
        AND c.relrowsecurity AND c.relforcerowsecurity AND c.relacl IS NULL)
    OR EXISTS(SELECT 1 FROM pg_trigger t
      WHERE t.tgrelid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
        AND NOT t.tgisinternal)
    OR to_regclass('vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(jsonb,jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_descarte_seleccion_oferta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'CT220: preimagen instalada incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Se conserva la cabecera O6 y sus filas singulares. La variante tipada usa
-- sus reservas, arrendamiento, fencing y terminal; el comando propio se fija
-- antes de abrir la ventana de publicación.
CREATE FUNCTION vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(i jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT encode(sha256(convert_to(string_agg(octet_length(convert_to(x,'UTF8'))::text||':'||x,'' ORDER BY n),'UTF8')),'hex')
 FROM unnest(ARRAY[
  'vec.ct.seleccion-oferta.intencion.v1', i->>'organizacion_ref',
  i->>'expediente_ref', i->>'version_expediente', i->>'clave_idempotencia',
  i->>'actor_ref', i->>'perfil_ref']) WITH ORDINALITY AS z(x,n)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(i jsonb,p_clave uuid,p_huella text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(
  jsonb_typeof(i)='object'
  AND octet_length(i::text)<=65536
  AND i ?& ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'clave_idempotencia','actor_ref','perfil_ref','categoria_ref','centro_ref','huella_intencion']
  AND i-ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'clave_idempotencia','actor_ref','perfil_ref','categoria_ref','centro_ref','huella_intencion']='{}'::jsonb
  AND i->>'organizacion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND i->>'expediente_ref' ~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
  AND jsonb_typeof(i->'version_expediente')='number'
  AND i->>'version_expediente' ~ '^[1-9][0-9]{0,15}$'
  AND (i->>'version_expediente')::numeric<=9007199254740991
  AND i->>'clave_idempotencia'=p_clave::text
  AND i->>'actor_ref' ~ '^per_[A-Za-z0-9_-]{22,128}$'
  AND i->>'perfil_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND i->>'categoria_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND i->>'centro_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND p_huella ~ '^[0-9a-f]{64}$'
  AND i->>'huella_intencion'=p_huella
  AND vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(i)=p_huella,false)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(jsonb,uuid,text) FROM PUBLIC;

ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 ADD COLUMN tipo_seleccion text NOT NULL DEFAULT 'llamamiento',
 ADD COLUMN comando_oferta_json jsonb;
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 ADD CONSTRAINT ejecucion_seleccion_o6_tipo_ct220 CHECK(tipo_seleccion IN ('llamamiento','oferta'));
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_check,
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_check2,
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_check3,
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_efecto_check,
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_reserva_ref_check,
 DROP CONSTRAINT ejecucion_seleccion_llamamiento_o6_situacion_check;
ALTER TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 ADD CONSTRAINT ejecucion_seleccion_o6_situacion_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND situacion IN ('propietaria','indeterminada','confirmada'))
  OR (tipo_seleccion='oferta' AND situacion IN ('propietaria','indeterminada','confirmada','descartada'))),
 ADD CONSTRAINT ejecucion_seleccion_o6_solicitud_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND comando_oferta_json IS NULL AND
   jsonb_typeof(solicitud_json)='object' AND octet_length(solicitud_json::text)<=1048576 AND
   vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(solicitud_json) IS NOT DISTINCT FROM huella_semantica AND
   solicitud_json->>'clave_idempotencia' IS NOT DISTINCT FROM clave_idempotencia::text AND
   solicitud_json->>'huella_semantica' IS NOT DISTINCT FROM huella_semantica)
  OR (tipo_seleccion='oferta' AND
   vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(solicitud_json,clave_idempotencia,huella_semantica)
   AND jsonb_typeof(comando_oferta_json)='object'
   AND octet_length(comando_oferta_json::text)<=1048576)),
 ADD CONSTRAINT ejecucion_seleccion_o6_reserva_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND reserva_ref ~ '^reserva:seleccion-llamamiento:v2:[0-9a-f]{64}$'
   AND reserva_ref <> 'reserva:seleccion-llamamiento:v2:'||repeat('0',64))
  OR (tipo_seleccion='oferta' AND reserva_ref ~ '^reserva:seleccion-oferta:v1:[0-9a-f]{64}$')),
 ADD CONSTRAINT ejecucion_seleccion_o6_efecto_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND efecto IN ('preparar_orden','solicitar_llamamiento'))
  OR (tipo_seleccion='oferta' AND efecto='publicar_oferta') OR efecto IS NULL),
 ADD CONSTRAINT ejecucion_seleccion_o6_recibo_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND
   ((recibo_json IS NULL AND artefacto_canonico IS NULL) OR
    (jsonb_typeof(recibo_json)='object' AND jsonb_typeof(artefacto_canonico::jsonb)='object'
     AND octet_length(recibo_json::text)+octet_length(artefacto_canonico)<=1048576)))
  OR (tipo_seleccion='oferta' AND artefacto_canonico IS NULL
    AND (recibo_json IS NULL OR (jsonb_typeof(recibo_json)='object' AND octet_length(recibo_json::text)<=65536)))),
 ADD CONSTRAINT ejecucion_seleccion_o6_estado_ct220 CHECK(
  (tipo_seleccion='llamamiento' AND
   ((situacion='propietaria' AND recibo_json IS NULL AND artefacto_canonico IS NULL AND
      ((NOT ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto IS NULL)
       OR (ventana_orden_abierta AND NOT ventana_llamamiento_abierta AND efecto='preparar_orden')
       OR (ventana_orden_abierta AND ventana_llamamiento_abierta AND efecto='solicitar_llamamiento')))
    OR (situacion='indeterminada' AND ventana_orden_abierta AND efecto IN ('preparar_orden','solicitar_llamamiento')
       AND recibo_json IS NULL AND artefacto_canonico IS NULL)
    OR (situacion='confirmada' AND ventana_orden_abierta AND ventana_llamamiento_abierta
       AND efecto IS NULL AND recibo_json IS NOT NULL AND artefacto_canonico IS NOT NULL)))
  OR (tipo_seleccion='oferta' AND NOT ventana_llamamiento_abierta AND artefacto_canonico IS NULL AND
   ((situacion='propietaria' AND recibo_json IS NULL AND
      ((NOT ventana_orden_abierta AND efecto IS NULL) OR (ventana_orden_abierta AND efecto='publicar_oferta')))
    OR (situacion='indeterminada' AND ventana_orden_abierta AND efecto='publicar_oferta' AND recibo_json IS NULL)
    OR (situacion='confirmada' AND ventana_orden_abierta AND efecto IS NULL AND recibo_json IS NOT NULL)
    OR (situacion='descartada' AND efecto IS NULL AND recibo_json IS NOT NULL))));
CREATE UNIQUE INDEX ejecucion_seleccion_oferta_unica_expediente_ct220
 ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 ((solicitud_json->>'organizacion_ref'),(solicitud_json->>'expediente_ref'))
 WHERE tipo_seleccion='oferta' AND situacion<>'descartada';

-- Las fachadas O6 antiguas siguen sirviendo al llamamiento singular. Esta
-- guarda física impide que su liberación DELETE alcance una oferta y que una
-- actualización cambie la identidad o el comando de la variante nueva.
CREATE FUNCTION vec_contratacion_temporal.guardar_variante_oferta_o6_ct220()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 IF TG_OP='DELETE' THEN
  IF OLD.tipo_seleccion='oferta' THEN
   RAISE EXCEPTION 'CT220: reserva de oferta inmutable' USING ERRCODE='42501'; END IF;
  RETURN OLD;
 END IF;
 IF OLD.tipo_seleccion<>'oferta' THEN
  IF NEW.tipo_seleccion='oferta' THEN
   RAISE EXCEPTION 'CT220: variante histórica inmutable' USING ERRCODE='42501'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.tipo_seleccion<>'oferta'
    OR NEW.clave_idempotencia IS DISTINCT FROM OLD.clave_idempotencia
    OR NEW.huella_semantica IS DISTINCT FROM OLD.huella_semantica
    OR NEW.solicitud_json IS DISTINCT FROM OLD.solicitud_json
    OR NEW.comando_oferta_json IS DISTINCT FROM OLD.comando_oferta_json
    OR NEW.creada_en IS DISTINCT FROM OLD.creada_en
    OR NEW.fencing_version<OLD.fencing_version
    OR (OLD.ventana_orden_abierta AND NEW.fencing_version IS DISTINCT FROM OLD.fencing_version)
    OR (NEW.reserva_ref IS DISTINCT FROM OLD.reserva_ref AND
      (OLD.ventana_orden_abierta OR NEW.fencing_version<>OLD.fencing_version+1))
    OR NEW.ventana_llamamiento_abierta IS DISTINCT FROM false
    OR NEW.artefacto_canonico IS NOT NULL
    OR (OLD.situacion IN ('confirmada','descartada'))
    OR (OLD.situacion='indeterminada' AND NEW.situacion NOT IN ('confirmada','descartada'))
    OR (OLD.ventana_orden_abierta AND NOT NEW.ventana_orden_abierta)
    OR (NOT OLD.ventana_orden_abierta AND NEW.situacion='indeterminada')
    OR (NEW.situacion='confirmada' AND (NOT NEW.ventana_orden_abierta OR NEW.recibo_json IS NULL))
    OR (NEW.situacion='descartada' AND NEW.recibo_json IS NULL)
    OR (NEW.situacion='propietaria' AND OLD.ventana_orden_abierta
      AND NEW.reserva_ref IS DISTINCT FROM OLD.reserva_ref)
 THEN RAISE EXCEPTION 'CT220: transición de oferta incompatible' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.guardar_variante_oferta_o6_ct220() FROM PUBLIC;
CREATE TRIGGER guardar_variante_oferta_ct220 BEFORE UPDATE OR DELETE
 ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.guardar_variante_oferta_o6_ct220();

CREATE FUNCTION vec_contratacion_temporal.rechazar_truncado_ejecucion_o6_ct220()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 RAISE EXCEPTION 'CT220: historia de selección inmutable' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.rechazar_truncado_ejecucion_o6_ct220() FROM PUBLIC;
CREATE TRIGGER no_truncar_ejecucion_ct220 BEFORE TRUNCATE
 ON vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_truncado_ejecucion_o6_ct220();

CREATE TABLE vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1(
 expediente_ref text NOT NULL REFERENCES vec_contratacion_temporal.expediente_alta(expediente_ref),
 organizacion_ref text NOT NULL,
 version_expediente bigint NOT NULL CHECK(version_expediente BETWEEN 1 AND 9007199254740991),
 clave_idempotencia uuid NOT NULL UNIQUE,
 bolsa_ref text NOT NULL,
 oferta_ref text NOT NULL UNIQUE CHECK(oferta_ref ~ '^oferta:[0-9a-f]{64}$'),
 recibo_oferta_ref text NOT NULL UNIQUE CHECK(recibo_oferta_ref ~ '^recibo:oferta:[0-9a-f]{64}$'),
 numero_plazas integer NOT NULL CHECK(numero_plazas BETWEEN 2 AND 100),
 origen_huella_sha256 text NOT NULL CHECK(origen_huella_sha256 ~ '^[0-9a-f]{64}$'),
 comando_json jsonb NOT NULL CHECK(jsonb_typeof(comando_json)='object'),
 comando_huella_sha256 text NOT NULL CHECK(comando_huella_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,perfil_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_vinculo_ref text NOT NULL UNIQUE,recibo_json jsonb NOT NULL,
 vinculada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(expediente_ref,oferta_ref),
 UNIQUE(organizacion_ref,expediente_ref),
 CHECK(comando_huella_sha256=encode(sha256(convert_to(comando_json::text,'UTF8')),'hex'))
);
ALTER TABLE vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1 FROM PUBLIC;

CREATE TABLE vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1(
 evento_ref text PRIMARY KEY,
 expediente_ref text NOT NULL,oferta_ref text NOT NULL UNIQUE,
 recibo_vinculo_ref text NOT NULL UNIQUE,
 tipo text NOT NULL CHECK(tipo='ct.oferta_bolsa_vinculada.v1'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(expediente_ref,oferta_ref)
  REFERENCES vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1(expediente_ref,oferta_ref),
 CHECK(payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex'))
);
ALTER TABLE vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1 FROM PUBLIC;

CREATE TABLE vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1(
 clave_idempotencia uuid PRIMARY KEY
  REFERENCES vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(clave_idempotencia),
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,
 version_expediente bigint NOT NULL CHECK(version_expediente BETWEEN 1 AND 9007199254740991),
 bolsa_ref text NOT NULL,clave_bolsa text NOT NULL,
 actor_ref text NOT NULL,perfil_descarte_ref text NOT NULL,
 comando_sha256 text NOT NULL CHECK(comando_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_cierre_ref text NOT NULL UNIQUE,cierre_json jsonb NOT NULL CHECK(jsonb_typeof(cierre_json)='object'),
 recibo_descarte_ref text NOT NULL UNIQUE,recibo_json jsonb NOT NULL CHECK(jsonb_typeof(recibo_json)='object'),
 decision_ref text NOT NULL UNIQUE,auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 evento_ref text NOT NULL UNIQUE,descartada_en timestamptz(6) NOT NULL,
 CHECK(comando_sha256 <> repeat('0',64))
);
ALTER TABLE vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1 FROM PUBLIC;

CREATE TABLE vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1(
 evento_ref text PRIMARY KEY,
 clave_idempotencia uuid NOT NULL UNIQUE
  REFERENCES vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1(clave_idempotencia),
 recibo_descarte_ref text NOT NULL UNIQUE,
 tipo text NOT NULL CHECK(tipo='ct.oferta_bolsa.preparacion_descartada.v1'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256 ~ '^[0-9a-f]{64}$'
  AND payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex')),
 registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1 FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c jsonb,i jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(
  jsonb_typeof(c)='object' AND octet_length(c::text)<=1048576
  AND c ?& ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
   'datos','origen_ct','clave_bolsa','inicio_plazo']
  AND c-ARRAY['organizacion_ref','expediente_ref','version_expediente',
   'actor_ref','bolsa_ref','necesidad_ref','numero_plazas','numero_elegibles',
   'datos','origen_ct','clave_bolsa','inicio_plazo']='{}'::jsonb
  AND c->>'organizacion_ref'=i->>'organizacion_ref'
  AND c->>'expediente_ref'=i->>'expediente_ref'
  AND c->'version_expediente'=i->'version_expediente'
  AND c->>'actor_ref'=i->>'actor_ref'
  AND jsonb_typeof(c->'version_expediente')='number'
  AND jsonb_typeof(c->'numero_plazas')='number'
  AND jsonb_typeof(c->'numero_elegibles')='number'
  AND c->>'numero_plazas' ~ '^([2-9]|[1-9][0-9]|100)$'
  AND c->>'numero_elegibles' ~ '^[1-9][0-9]{0,6}$'
  AND (c->>'numero_elegibles')::integer<=1000000
  AND (c->>'numero_elegibles')::integer >= (c->>'numero_plazas')::integer
  AND c->>'bolsa_ref' ~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
  AND c->>'necesidad_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND c->>'inicio_plazo'='publicacion'
  AND (c->>'clave_bolsa')=('ct-oferta:'||encode(sha256(convert_to(
   (i->>'organizacion_ref')||chr(31)||(i->>'expediente_ref')||chr(31)||
   (i->>'version_expediente')||chr(31)||(i->>'clave_idempotencia'),'UTF8')),'hex'))
  AND jsonb_typeof(c->'datos')='object'
  AND c->'datos' ?& ARRAY['categoria','centro','fecha_inicio','descripcion']
  AND (c->'datos')-ARRAY['categoria','centro','fecha_inicio','fecha_fin','descripcion']='{}'::jsonb
  AND NOT EXISTS(SELECT 1 FROM jsonb_each(c->'datos') d WHERE jsonb_typeof(d.value)<>'string')
  AND jsonb_typeof(c->'origen_ct')='object'
  AND c#>>'{origen_ct,organizacion_ref}'=i->>'organizacion_ref'
  AND c#>>'{origen_ct,expediente_ref}'=i->>'expediente_ref'
  AND c#>'{origen_ct,version_expediente}'=i->'version_expediente'
  AND c#>>'{origen_ct,clave_ct}'=i->>'clave_idempotencia'
  AND c#>>'{origen_ct,bolsa_ref}'=c->>'bolsa_ref'
  AND c#>>'{origen_ct,necesidad_ref}'=c->>'necesidad_ref'
  AND c#>>'{origen_ct,categoria_ref}'=i->>'categoria_ref'
  AND jsonb_typeof(c#>'{origen_ct,lectura}')='object'
  AND jsonb_typeof(c#>'{origen_ct,lectura,entradas}')='array'
  AND c#>>'{origen_ct,lectura,corte}' IS NOT NULL,false)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb) FROM PUBLIC;

-- Esta validación se ejecuta en la misma transacción que cada lectura/efecto.
-- AD278 acredita audiencia, acción y origen nominal; aquí se ligan los bytes
-- firmados a la etapa, expediente y material concreto de CT220.
CREATE FUNCTION vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
 p_etapa text,p_intencion jsonb,p_comando jsonb,p_oferta jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,auditoria_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb;d jsonb;consumo record;h text;ctx text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_etapa NOT IN ('consultar','reservar','abrir_ventana','confirmar')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 1048576
    OR vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(
      p_intencion,(p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion') IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: etapa no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT220: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m)<>'object' OR m->>'esquema'<>'vec.ct.seleccion-oferta.etapa.v1'
    OR m->>'etapa' IS DISTINCT FROM p_etapa
    OR m->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR m->'version_expediente' IS DISTINCT FROM p_intencion->'version_expediente'
    OR m->>'clave_idempotencia' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR m->>'huella_intencion' IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR m->>'categoria_ref' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR m->>'centro_ref' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR m->'comando' IS DISTINCT FROM p_comando
    OR m->'oferta' IS DISTINCT FROM p_oferta
    OR m-ARRAY['esquema','etapa','organizacion_ref','expediente_ref',
      'version_expediente','clave_idempotencia','huella_intencion',
      'categoria_ref','centro_ref','comando','oferta']<>'{}'::jsonb
    OR (p_etapa='consultar' AND (m ? 'comando' OR m ? 'oferta'))
    OR (p_etapa<>'consultar' AND (p_comando IS NULL OR NOT (m ? 'comando')))
    OR (p_etapa<>'confirmar' AND (p_oferta IS NOT NULL OR m ? 'oferta'))
    OR (p_etapa='confirmar' AND (p_oferta IS NULL OR NOT (m ? 'oferta')))
 THEN RAISE EXCEPTION 'CT220: etapa y material divergentes' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(p_intencion->>'categoria_ref')::text||',"centro_ref":'||
   to_json(p_intencion->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_intencion->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_oferta_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT220: decisión no ligada a etapa' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
 text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.estado_seleccion_oferta_ct220(e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)
RETURNS jsonb LANGUAGE sql STABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT jsonb_build_object(
  'situacion',CASE WHEN e.situacion='confirmada' THEN 'confirmada'
   WHEN e.situacion='descartada' THEN 'descartada'
   WHEN e.situacion='indeterminada' THEN 'indeterminada'
   ELSE 'reservada' END,
  'reserva_ref',CASE WHEN e.situacion IN ('confirmada','descartada') THEN '' ELSE e.reserva_ref END,
  'ventana_abierta',e.ventana_orden_abierta,
  'comando',e.comando_oferta_json,
  'recibo',CASE WHEN e.situacion='confirmada' THEN e.recibo_json ELSE NULL END,
  'recibo_descarte',CASE WHEN e.situacion='descartada' THEN e.recibo_json ELSE NULL END)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.estado_seleccion_oferta_ct220(vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(
 e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE v vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1%ROWTYPE;
 ev vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1%ROWTYPE;
 o jsonb; c jsonb; i jsonb;
BEGIN
 IF e.tipo_seleccion<>'oferta' OR e.situacion<>'confirmada'
    OR e.recibo_json IS NULL OR e.comando_oferta_json IS NULL THEN RETURN false; END IF;
 i:=e.solicitud_json;c:=e.comando_oferta_json;o:=e.recibo_json;
 SELECT * INTO v FROM vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1
 WHERE clave_idempotencia=e.clave_idempotencia;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO ev FROM vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1
 WHERE expediente_ref=v.expediente_ref AND oferta_ref=v.oferta_ref;
 IF NOT FOUND OR v.organizacion_ref IS DISTINCT FROM i->>'organizacion_ref'
    OR v.expediente_ref IS DISTINCT FROM i->>'expediente_ref'
    OR v.version_expediente IS DISTINCT FROM (i->>'version_expediente')::bigint
    OR v.bolsa_ref IS DISTINCT FROM c->>'bolsa_ref'
    OR v.numero_plazas IS DISTINCT FROM (c->>'numero_plazas')::integer
    OR v.actor_ref IS DISTINCT FROM i->>'actor_ref'
    OR v.perfil_ref IS DISTINCT FROM i->>'perfil_ref'
    OR v.comando_json IS DISTINCT FROM c
    OR v.comando_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(c::text,'UTF8')),'hex')
    OR v.origen_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to((c->'origen_ct')::text,'UTF8')),'hex')
    OR v.recibo_json IS DISTINCT FROM o
    OR ev.recibo_vinculo_ref IS DISTINCT FROM v.recibo_vinculo_ref
    OR ev.payload IS DISTINCT FROM o
    OR ev.payload_sha256 IS DISTINCT FROM encode(sha256(convert_to(o::text,'UTF8')),'hex')
    OR v.decision_ref !~ '^[A-Za-z0-9:_-]{8,160}$'
    OR v.auditoria_ref !~ '^[A-Za-z0-9:_-]{8,160}$'
    OR o->>'recibo_vinculo_ref' IS DISTINCT FROM v.recibo_vinculo_ref
    OR o->>'recibo_oferta_ref' IS DISTINCT FROM v.recibo_oferta_ref
    OR o->>'oferta_ref' IS DISTINCT FROM v.oferta_ref
    OR o->>'estado' IS DISTINCT FROM 'confirmado'
    OR o->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.recibo-seleccion-oferta.v1'
 THEN RETURN false; END IF;
 RETURN vec_bolsa_llamamientos.verificar_oferta_ct_v1(
  v.organizacion_ref,v.expediente_ref,v.version_expediente,
  e.clave_idempotencia::text,i->>'actor_ref',v.bolsa_ref,c->>'clave_bolsa',
  v.oferta_ref,v.recibo_oferta_ref,v.numero_plazas,c->'origen_ct',v.origen_huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(
 vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(
 e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE v vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1%ROWTYPE;
 ev vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1%ROWTYPE;
 i jsonb;c jsonb;r jsonb;
BEGIN
 IF e.tipo_seleccion<>'oferta' OR e.situacion<>'descartada'
    OR e.recibo_json IS NULL OR e.comando_oferta_json IS NULL THEN RETURN false; END IF;
 i:=e.solicitud_json;c:=e.comando_oferta_json;r:=e.recibo_json;
 SELECT * INTO v FROM vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
 WHERE clave_idempotencia=e.clave_idempotencia;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO ev FROM vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1
 WHERE clave_idempotencia=e.clave_idempotencia;
 IF NOT FOUND OR v.organizacion_ref IS DISTINCT FROM i->>'organizacion_ref'
    OR v.expediente_ref IS DISTINCT FROM i->>'expediente_ref'
    OR v.version_expediente IS DISTINCT FROM (i->>'version_expediente')::bigint
    OR v.actor_ref IS DISTINCT FROM i->>'actor_ref'
    OR v.perfil_descarte_ref IS NOT DISTINCT FROM i->>'perfil_ref'
    OR v.bolsa_ref IS DISTINCT FROM c->>'bolsa_ref'
    OR v.clave_bolsa IS DISTINCT FROM c->>'clave_bolsa'
    OR v.comando_sha256 IS DISTINCT FROM encode(sha256(convert_to(c::text,'UTF8')),'hex')
    OR v.recibo_cierre_ref IS DISTINCT FROM v.cierre_json->>'recibo_cierre_ref'
    OR v.recibo_json IS DISTINCT FROM r
    OR v.recibo_descarte_ref IS DISTINCT FROM r->>'recibo_descarte_ref'
    OR r->>'estado' IS DISTINCT FROM 'descartada'
    OR r->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.recibo-descarte-oferta.v1'
    OR ev.evento_ref IS DISTINCT FROM v.evento_ref
    OR ev.recibo_descarte_ref IS DISTINCT FROM v.recibo_descarte_ref
    OR ev.payload IS DISTINCT FROM r
    OR ev.payload_sha256 IS DISTINCT FROM encode(sha256(convert_to(r::text,'UTF8')),'hex')
 THEN RETURN false; END IF;
 RETURN vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(c,v.cierre_json);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(
 vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.consultar_seleccion_oferta_ct220(
 p_intencion jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 v_ahora timestamptz(6); v_auth record;
BEGIN
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'consultar',p_intencion,NULL,NULL,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF e.clave_idempotencia IS NULL THEN RETURN jsonb_build_object('situacion','ausente'); END IF;
 IF e.tipo_seleccion<>'oferta'
    OR e.huella_semantica IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR e.solicitud_json IS DISTINCT FROM p_intencion THEN
  RETURN jsonb_build_object('situacion','colision');
 END IF;
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(e.comando_oferta_json,p_intencion) IS NOT TRUE
    OR (e.situacion='confirmada' AND
      vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(e) IS NOT TRUE)
    OR (e.situacion='descartada' AND
      vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(e) IS NOT TRUE) THEN
  RAISE EXCEPTION 'CT220: ejecución histórica inconsistente' USING ERRCODE='55000';
 END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF e.situacion='propietaria' AND e.lease_hasta>v_ahora THEN
  RETURN jsonb_build_object('situacion','ocupada');
 END IF;
 IF e.situacion='propietaria' AND e.ventana_orden_abierta THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET situacion='indeterminada',actualizada_en=v_ahora
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 ELSIF e.situacion='propietaria' THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET reserva_ref='reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
      fencing_version=fencing_version+1,
      actualizada_en=v_ahora,lease_hasta=v_ahora+interval '30 seconds'
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 END IF;
 RETURN vec_contratacion_temporal.estado_seleccion_oferta_ct220(e);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_seleccion_oferta_ct220(
 jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_seleccion_oferta_ct220(
 jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.reservar_seleccion_oferta_ct220(
 p_intencion jsonb,p_comando jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 v_cabecera record;v_agregado jsonb;v_ahora timestamptz(6);v_auth record;v_insertadas integer;
BEGIN
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(p_comando,p_intencion) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT220: comando de oferta inválido' USING ERRCODE='22023'; END IF;
 -- Misma fila que bloquea CT122 antes de cancelar. El corte 3 añadirá la
 -- guarda de reservas pendientes después de adquirir exactamente este lock.
 SELECT a.version,v.agregado_json INTO v_cabecera
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'reservar',p_intencion,p_comando,NULL,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_cabecera.version IS NULL THEN
  RAISE EXCEPTION 'CT220: expediente inexistente' USING ERRCODE='23503'; END IF;
 v_agregado:=v_cabecera.agregado_json;
 IF v_cabecera.version IS DISTINCT FROM (p_intencion->>'version_expediente')::numeric
    OR v_agregado->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
    OR v_agregado->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR v_agregado#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR v_agregado#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR v_agregado#>>'{solicitud,numero_personas}' IS DISTINCT FROM p_comando->>'numero_plazas'
    OR v_agregado#>>'{solicitud,centro_ref}' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR v_agregado#>>'{analisis,categoria_ref}' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR v_agregado#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_comando#>>'{origen_ct,unidad_ref}'
 THEN RAISE EXCEPTION 'CT220: expediente ya no seleccionable' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
   WHERE x.expediente_ref=p_intencion->>'expediente_ref') THEN
  RAISE EXCEPTION 'CT220: expediente ya vinculado a llamamiento' USING ERRCODE='23505'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
   WHERE x.tipo_seleccion='llamamiento'
     AND x.solicitud_json->>'expediente_ref'=p_intencion->>'expediente_ref') THEN
  RAISE EXCEPTION 'CT220: expediente ya tiene selección singular' USING ERRCODE='23505'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
  clave_idempotencia,huella_semantica,solicitud_json,reserva_ref,tipo_seleccion,comando_oferta_json)
 VALUES((p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion',p_intencion,
  'reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'oferta',p_comando)
 ON CONFLICT (clave_idempotencia) DO NOTHING;
 GET DIAGNOSTICS v_insertadas=ROW_COUNT;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.huella_semantica IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR e.comando_oferta_json IS DISTINCT FROM p_comando THEN
  RETURN jsonb_build_object('situacion','colision'); END IF;
 IF v_insertadas=1 THEN RETURN vec_contratacion_temporal.estado_seleccion_oferta_ct220(e); END IF;
 IF e.situacion='propietaria' AND e.lease_hasta>v_ahora THEN
  RETURN jsonb_build_object('situacion','ocupada'); END IF;
 IF e.situacion='propietaria' AND e.ventana_orden_abierta THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET situacion='indeterminada',actualizada_en=v_ahora
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 ELSIF e.situacion='propietaria' THEN
  UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
  SET reserva_ref='reserva:seleccion-oferta:v1:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
      fencing_version=fencing_version+1,
      actualizada_en=v_ahora,lease_hasta=v_ahora+interval '30 seconds'
  WHERE clave_idempotencia=e.clave_idempotencia RETURNING * INTO e;
 END IF;
 RETURN vec_contratacion_temporal.estado_seleccion_oferta_ct220(e);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.reservar_seleccion_oferta_ct220(
 jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.reservar_seleccion_oferta_ct220(
 jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_oferta_ct220(
 p_intencion jsonb,p_reserva text,p_comando jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 v_auth record;v_ahora timestamptz(6);
BEGIN
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'abrir_ventana',p_intencion,p_comando,NULL,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF e.clave_idempotencia IS NULL OR e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.comando_oferta_json IS DISTINCT FROM p_comando
    OR e.reserva_ref IS DISTINCT FROM p_reserva
    OR e.situacion<>'propietaria' OR e.lease_hasta<=v_ahora
 THEN RAISE EXCEPTION 'CT220: reserva de oferta no vigente' USING ERRCODE='55000'; END IF;
 IF e.ventana_orden_abierta THEN RETURN true; END IF;
 UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET ventana_orden_abierta=true,efecto='publicar_oferta',actualizada_en=v_ahora,
     lease_hasta=v_ahora+interval '30 seconds'
 WHERE clave_idempotencia=e.clave_idempotencia;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_oferta_ct220(
 jsonb,text,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.abrir_ventana_seleccion_oferta_ct220(
 jsonb,text,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.confirmar_vinculo_seleccion_oferta_ct220(
 p_intencion jsonb,p_reserva text,p_comando jsonb,p_oferta jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 v_cabecera record;v_historico jsonb;v_historico_h text;
 v_auth record;v_ahora timestamptz(6);v_recibo text;
 v_evento text;v_resultado jsonb;v_origen_h text;
BEGIN
 IF jsonb_typeof(p_oferta)<>'object'
    OR p_oferta ?& ARRAY['clave_bolsa','bolsa_ref','oferta_ref','recibo_oferta_ref',
      'numero_plazas','datos','publicada_en','vence_antes_de'] IS NOT TRUE
    OR p_oferta-ARRAY['clave_bolsa','bolsa_ref','oferta_ref','recibo_oferta_ref',
      'numero_plazas','datos','publicada_en','vence_antes_de']<>'{}'::jsonb
    OR p_oferta->>'clave_bolsa' IS DISTINCT FROM p_comando->>'clave_bolsa'
    OR p_oferta->>'bolsa_ref' IS DISTINCT FROM p_comando->>'bolsa_ref'
    OR p_oferta->'numero_plazas' IS DISTINCT FROM p_comando->'numero_plazas'
    OR p_oferta->'datos' IS DISTINCT FROM p_comando->'datos'
    OR p_oferta->>'oferta_ref' !~ '^oferta:[0-9a-f]{64}$'
    OR p_oferta->>'recibo_oferta_ref' !~ '^recibo:oferta:[0-9a-f]{64}$'
    OR p_oferta->>'publicada_en' IS NULL
    OR p_oferta->>'vence_antes_de' IS NULL
    OR (p_oferta->>'vence_antes_de')::timestamptz <= (p_oferta->>'publicada_en')::timestamptz
 THEN RAISE EXCEPTION 'CT220: recibo de Bolsa inválido' USING ERRCODE='22023'; END IF;
 -- Orden igual que en reservar/cancelar: primero cabecera de CT, luego O6.
 SELECT a.version,v.agregado_json INTO v_cabecera
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
 SELECT * INTO STRICT v_auth FROM vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
  'confirmar',p_intencion,p_comando,p_oferta,p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF v_cabecera.version IS NULL OR
    v_cabecera.version<(p_intencion->>'version_expediente')::numeric THEN
  RAISE EXCEPTION 'CT220: expediente inexistente o historia divergente' USING ERRCODE='23503'; END IF;
 IF e.clave_idempotencia IS NULL OR e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.comando_oferta_json IS DISTINCT FROM p_comando
    OR e.reserva_ref IS DISTINCT FROM p_reserva
 THEN RAISE EXCEPTION 'CT220: reserva divergente' USING ERRCODE='23505'; END IF;
 IF e.situacion='confirmada' THEN
  IF vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(e) IS NOT TRUE
     OR e.recibo_json->>'oferta_ref' IS DISTINCT FROM p_oferta->>'oferta_ref'
     OR e.recibo_json->>'recibo_oferta_ref' IS DISTINCT FROM p_oferta->>'recibo_oferta_ref'
     OR e.recibo_json->'numero_plazas' IS DISTINCT FROM p_oferta->'numero_plazas'
  THEN RAISE EXCEPTION 'CT220: terminal divergente' USING ERRCODE='23505'; END IF;
  RETURN e.recibo_json;
 END IF;
 IF e.situacion NOT IN ('propietaria','indeterminada')
    OR NOT e.ventana_orden_abierta
 THEN RAISE EXCEPTION 'CT220: expediente o ventana ya no confirmable' USING ERRCODE='55000'; END IF;
 -- El efecto Bolsa puede haberse confirmado antes de que CT avanzara. La
 -- vinculación registra ese hecho contra la versión fiscalizada original;
 -- nunca vuelve la cabeza a un estado anterior ni crea otra publicación.
 SELECT v.agregado_json,v.agregado_json_huella_sha256 INTO v_historico,v_historico_h
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::numeric;
 IF NOT FOUND OR v_historico_h IS DISTINCT FROM encode(sha256(convert_to(v_historico::text,'UTF8')),'hex')
    OR v_historico->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
    OR v_historico->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR v_historico#>>'{solicitud,numero_personas}' IS DISTINCT FROM p_comando->>'numero_plazas'
    OR v_historico#>>'{solicitud,centro_ref}' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR v_historico#>>'{analisis,categoria_ref}' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR v_historico#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_comando#>>'{origen_ct,unidad_ref}'
 THEN RAISE EXCEPTION 'CT220: versión fiscalizada original divergente' USING ERRCODE='55000'; END IF;
 v_origen_h:=encode(sha256(convert_to((p_comando->'origen_ct')::text,'UTF8')),'hex');
 IF vec_bolsa_llamamientos.verificar_oferta_ct_v1(
   p_intencion->>'organizacion_ref',p_intencion->>'expediente_ref',
   (p_intencion->>'version_expediente')::bigint,p_intencion->>'clave_idempotencia',p_intencion->>'actor_ref',
   p_comando->>'bolsa_ref',p_comando->>'clave_bolsa',
   p_oferta->>'oferta_ref',p_oferta->>'recibo_oferta_ref',
   (p_comando->>'numero_plazas')::integer,p_comando->'origen_ct',v_origen_h) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: Bolsa no acredita oferta propia' USING ERRCODE='42501'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 v_recibo:='recibo:ct:oferta:'||gen_random_uuid()::text;
 v_evento:='evento:ct:oferta:'||gen_random_uuid()::text;
 v_resultado:=jsonb_build_object(
  'esquema','vec.contratacion-temporal.recibo-seleccion-oferta.v1',
  'estado','confirmado',
  'organizacion_ref',p_intencion->>'organizacion_ref',
  'expediente_ref',p_intencion->>'expediente_ref',
  'version_expediente',(p_intencion->>'version_expediente')::bigint,
  'bolsa_ref',p_comando->>'bolsa_ref',
  'oferta_ref',p_oferta->>'oferta_ref',
  'recibo_oferta_ref',p_oferta->>'recibo_oferta_ref',
  'recibo_vinculo_ref',v_recibo,
  'numero_plazas',(p_comando->>'numero_plazas')::integer,
  'publicada_en',p_oferta->'publicada_en',
  'vence_antes_de',p_oferta->'vence_antes_de');
 INSERT INTO vec_contratacion_temporal.vinculo_oferta_bolsa_ct_v1(
  expediente_ref,organizacion_ref,version_expediente,clave_idempotencia,
  bolsa_ref,oferta_ref,recibo_oferta_ref,numero_plazas,origen_huella_sha256,
  comando_json,comando_huella_sha256,actor_ref,perfil_ref,decision_ref,auditoria_ref,
  consumo_huella_sha256,recibo_vinculo_ref,recibo_json,vinculada_en)
 VALUES(p_intencion->>'expediente_ref',p_intencion->>'organizacion_ref',
  (p_intencion->>'version_expediente')::bigint,e.clave_idempotencia,
  p_comando->>'bolsa_ref',p_oferta->>'oferta_ref',p_oferta->>'recibo_oferta_ref',
  (p_comando->>'numero_plazas')::integer,v_origen_h,p_comando,
  encode(sha256(convert_to(p_comando::text,'UTF8')),'hex'),
  p_intencion->>'actor_ref',p_intencion->>'perfil_ref',v_auth.decision_ref,v_auth.auditoria_ref,
  v_auth.consumo_huella_sha256,v_recibo,v_resultado,v_ahora);
 INSERT INTO vec_contratacion_temporal.outbox_vinculo_oferta_bolsa_ct_v1(
  evento_ref,expediente_ref,oferta_ref,recibo_vinculo_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(v_evento,p_intencion->>'expediente_ref',p_oferta->>'oferta_ref',v_recibo,
  'ct.oferta_bolsa_vinculada.v1',v_resultado,
  encode(sha256(convert_to(v_resultado::text,'UTF8')),'hex'),v_ahora);
 UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET situacion='confirmada',efecto=NULL,recibo_json=v_resultado,actualizada_en=v_ahora
 WHERE clave_idempotencia=e.clave_idempotencia;
 RETURN v_resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.confirmar_vinculo_seleccion_oferta_ct220(
 jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.confirmar_vinculo_seleccion_oferta_ct220(
 jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.descartar_seleccion_oferta_ct220(
 p_intencion jsonb,p_reserva text,p_comando jsonb,p_cierre jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 cab record;m jsonb;d jsonb;auth record;h text;ctx text;
 historico jsonb;historico_h text;
 v_ahora timestamptz(6);v_recibo text;v_evento text;v_salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_reserva IS NULL OR p_material IS NULL
    OR octet_length(p_material) NOT BETWEEN 128 AND 1048576
    OR vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(
      p_intencion,(p_intencion->>'clave_idempotencia')::uuid,p_intencion->>'huella_intencion') IS NOT TRUE
    OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(p_comando,p_intencion) IS NOT TRUE
    OR jsonb_typeof(p_cierre)<>'object' OR octet_length(p_cierre::text)>65536
 THEN RAISE EXCEPTION 'CT220: descarte no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT220: material de descarte ilegible' USING ERRCODE='22023';
 END;
 IF m->>'esquema' IS DISTINCT FROM 'vec.ct.seleccion-oferta.descarte.v1'
    OR m->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR m->'version_expediente' IS DISTINCT FROM p_intencion->'version_expediente'
    OR m->>'clave_idempotencia' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR m->>'huella_intencion' IS DISTINCT FROM p_intencion->>'huella_intencion'
    OR m->>'actor_ref' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR m->>'categoria_ref' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR m->>'centro_ref' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR m->'comando' IS DISTINCT FROM p_comando
    OR m->'cierre' IS DISTINCT FROM p_cierre
    OR m-ARRAY['esquema','organizacion_ref','expediente_ref','version_expediente',
      'clave_idempotencia','huella_intencion','actor_ref','categoria_ref','centro_ref',
      'comando','cierre']<>'{}'::jsonb
    OR p_cierre->>'esquema' IS DISTINCT FROM 'vec.bolsa.oferta-ct.cierre.v1'
    OR p_cierre->>'estado' IS DISTINCT FROM 'cerrada_sin_publicar'
    OR p_cierre->>'organizacion_ref' IS DISTINCT FROM p_intencion->>'organizacion_ref'
    OR p_cierre->>'expediente_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR p_cierre->>'version_expediente' IS DISTINCT FROM p_intencion->>'version_expediente'
    OR p_cierre->>'clave_ct' IS DISTINCT FROM p_intencion->>'clave_idempotencia'
    OR p_cierre->>'actor_ref' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR p_cierre->>'bolsa_ref' IS DISTINCT FROM p_comando->>'bolsa_ref'
    OR p_cierre->>'clave_bolsa' IS DISTINCT FROM p_comando->>'clave_bolsa'
    OR p_cierre->'numero_plazas' IS DISTINCT FROM p_comando->'numero_plazas'
    OR p_cierre->>'comando_sha256' IS DISTINCT FROM encode(sha256(convert_to(p_comando::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT220: cierre o material divergente' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(p_intencion->>'categoria_ref')::text||',"centro_ref":'||
   to_json(p_intencion->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM p_intencion->>'actor_ref'
    OR d->>'perfil_activo_ref' IS NOT DISTINCT FROM p_intencion->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.descartar'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'seleccion_oferta_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT220: decisión de descarte divergente' USING ERRCODE='42501'; END IF;
 -- Igual orden que reservar y cancelar: cabecera CT → V3 → O6.
 SELECT a.version INTO cab FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta alta ON alta.expediente_ref=a.expediente_ref
 WHERE a.expediente_ref=p_intencion->>'expediente_ref'
   AND alta.organizacion_ref=p_intencion->>'organizacion_ref'
 FOR UPDATE OF a;
 SELECT * INTO STRICT auth FROM vec_autorizacion_atestada_v3.registrar_y_consumir_descarte_seleccion_oferta_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF auth.efecto_ref IS DISTINCT FROM p_intencion->>'expediente_ref'
    OR auth.huella_efecto_sha256 IS DISTINCT FROM ctx OR auth.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: consumo de descarte divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE clave_idempotencia=(p_intencion->>'clave_idempotencia')::uuid FOR UPDATE;
 IF cab.version IS NULL OR e.clave_idempotencia IS NULL
    OR e.tipo_seleccion<>'oferta' OR e.solicitud_json IS DISTINCT FROM p_intencion
    OR e.comando_oferta_json IS DISTINCT FROM p_comando
    OR e.reserva_ref IS DISTINCT FROM p_reserva
 THEN RAISE EXCEPTION 'CT220: reserva de descarte divergente' USING ERRCODE='23505'; END IF;
 IF e.situacion='descartada' THEN
  IF vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(e) IS NOT TRUE
     OR (SELECT cierre_json FROM vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1
       WHERE clave_idempotencia=e.clave_idempotencia) IS DISTINCT FROM p_cierre
  THEN RAISE EXCEPTION 'CT220: descarte previo divergente' USING ERRCODE='23505'; END IF;
  RETURN e.recibo_json;
 END IF;
 IF e.situacion NOT IN ('propietaria','indeterminada')
    OR vec_bolsa_llamamientos.verificar_cierre_intencion_oferta_ct_v1(p_comando,p_cierre) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT220: Bolsa no acredita cierre sin oferta' USING ERRCODE='42501'; END IF;
 SELECT v.agregado_json,v.agregado_json_huella_sha256 INTO historico,historico_h
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::numeric;
 IF NOT FOUND OR historico_h IS DISTINCT FROM encode(sha256(convert_to(historico::text,'UTF8')),'hex')
    OR historico->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
    OR historico->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR historico#>>'{solicitud,numero_personas}' IS DISTINCT FROM p_comando->>'numero_plazas'
    OR historico#>>'{solicitud,centro_ref}' IS DISTINCT FROM p_intencion->>'centro_ref'
    OR historico#>>'{analisis,categoria_ref}' IS DISTINCT FROM p_intencion->>'categoria_ref'
    OR historico#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_comando#>>'{origen_ct,unidad_ref}'
 THEN RAISE EXCEPTION 'CT220: versión original de descarte divergente' USING ERRCODE='55000'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF v_ahora<(p_cierre->>'cerrada_en')::timestamptz THEN
  RAISE EXCEPTION 'CT220: cierre futuro' USING ERRCODE='22023'; END IF;
 v_recibo:='recibo:ct:oferta-descarte:'||gen_random_uuid()::text;
 v_evento:='evento:ct:oferta-descarte:'||gen_random_uuid()::text;
 v_salida:=jsonb_build_object(
  'esquema','vec.contratacion-temporal.recibo-descarte-oferta.v1','estado','descartada',
  'organizacion_ref',p_intencion->>'organizacion_ref',
  'expediente_ref',p_intencion->>'expediente_ref',
  'version_expediente',(p_intencion->>'version_expediente')::bigint,
  'bolsa_ref',p_comando->>'bolsa_ref','clave_bolsa',p_comando->>'clave_bolsa',
  'recibo_cierre_ref',p_cierre->>'recibo_cierre_ref','recibo_descarte_ref',v_recibo,
  'motivo','origen_caducado',
  'descartada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'auditoria_ref',auth.auditoria_ref,'evento_ref',v_evento);
 INSERT INTO vec_contratacion_temporal.descarte_seleccion_oferta_ct_v1(
  clave_idempotencia,organizacion_ref,expediente_ref,version_expediente,
  bolsa_ref,clave_bolsa,actor_ref,perfil_descarte_ref,comando_sha256,
  recibo_cierre_ref,cierre_json,recibo_descarte_ref,recibo_json,
  decision_ref,auditoria_ref,consumo_huella_sha256,evento_ref,descartada_en)
 VALUES(e.clave_idempotencia,p_intencion->>'organizacion_ref',p_intencion->>'expediente_ref',
  (p_intencion->>'version_expediente')::bigint,p_comando->>'bolsa_ref',p_comando->>'clave_bolsa',
  p_intencion->>'actor_ref',d->>'perfil_activo_ref',encode(sha256(convert_to(p_comando::text,'UTF8')),'hex'),
  p_cierre->>'recibo_cierre_ref',p_cierre,v_recibo,v_salida,
  auth.decision_ref,auth.auditoria_ref,auth.consumo_huella_sha256,v_evento,v_ahora);
 INSERT INTO vec_contratacion_temporal.outbox_descarte_seleccion_oferta_ct_v1(
  evento_ref,clave_idempotencia,recibo_descarte_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(v_evento,e.clave_idempotencia,v_recibo,'ct.oferta_bolsa.preparacion_descartada.v1',v_salida,
  encode(sha256(convert_to(v_salida::text,'UTF8')),'hex'),v_ahora);
 UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET situacion='descartada',efecto=NULL,recibo_json=v_salida,actualizada_en=v_ahora
 WHERE clave_idempotencia=e.clave_idempotencia;
 RETURN v_salida;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.descartar_seleccion_oferta_ct220(
 jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.descartar_seleccion_oferta_ct220(
 jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
COMMIT;
