\set ON_ERROR_STOP on
-- B110. Fuente canónica de Bolsa constituida para el llamamiento nativo CT.
-- Lectura por una sola categoría vigente; ninguna regla ni posición se deduce
-- en Contratación. La reserva CT se añade más abajo a la operación B7.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000110',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text)') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(text,text,text,text)') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sustitucion_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.instantanea_orden_bolsa') IS NULL
    OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_orden_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR strpos(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
      'leer_orden_vigente_bolsa_v1')=0
    OR strpos(pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
      '^bolsa-llamamiento-v[1-9][0-9]{0,3}$')=0
 THEN RAISE EXCEPTION 'B110: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Sólo el propietario del wrapper AD261 accede a las primitivas de lectura.
-- El LOGIN técnico nunca recibe ejecución directa ni acceso a tablas de Bolsa.
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos
 TO vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(p_categoria text)
RETURNS TABLE(acta_ref text,categoria_ref text,bolsa_ref text,version_bolsa bigint,
 huella_bolsa_sha256 text,bolsa_canonica bytea,instantanea_ref text,
 version_instantanea bigint,huella_instantanea_sha256 text,instantanea_canonica bytea,
 politica_ref text,version_politica bigint,politica_canonica bytea,
 politica_provisional boolean,corte timestamptz,
 bolsa_vigente_hasta timestamptz,politica_vigente_hasta timestamptz,
 entradas jsonb)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET statement_timeout='15s' AS $f$
DECLARE
 v_corte timestamptz(6):=date_trunc('microseconds',statement_timestamp());
 v_const record;
 v_politica vec_bolsa_llamamientos.politica_orden_bolsa%ROWTYPE;
 v_entradas jsonb;
 v_total bigint;
 v_orden bigint;
 v_sin_vinculo bigint;
BEGIN
 IF p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
    OR p_categoria<>btrim(p_categoria) THEN
  RAISE EXCEPTION 'B110: categoría inválida' USING ERRCODE='22023'; END IF;
 SELECT count(*) INTO v_total
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_total<>1 THEN
  RAISE EXCEPTION 'B110: constitución vigente ausente o ambigua' USING ERRCODE='P1101'; END IF;
 SELECT c.acta_ref,c.categoria_ref,c.bolsa_ref,c.version_bolsa,c.huella_bolsa_sha256,
        b.bolsa_canonica,c.instantanea_ref,c.version_instantanea,
        c.huella_instantanea_sha256,i.instantanea_canonica,i.total_participaciones,
        b.vigente_hasta AS bolsa_vigente_hasta
 INTO STRICT v_const
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
   ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
  AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF encode(sha256(v_const.bolsa_canonica),'hex')<>v_const.huella_bolsa_sha256
    OR encode(sha256(v_const.instantanea_canonica),'hex')<>v_const.huella_instantanea_sha256
 THEN RAISE EXCEPTION 'B110: canónico divergente' USING ERRCODE='P1101'; END IF;
 SELECT * INTO v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa p
 WHERE p.bolsa_ref=v_const.bolsa_ref AND p.vigente_desde<=v_corte
   AND (p.vigente_hasta IS NULL OR v_corte<p.vigente_hasta)
 ORDER BY p.version DESC LIMIT 1;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'B110: política de orden publicada ausente' USING ERRCODE='P1101'; END IF;
 WITH posiciones AS MATERIALIZED (
   SELECT o.* FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(v_const.bolsa_ref,v_corte) o
 )
 SELECT count(*),count(p.participacion_ref),
        count(*) FILTER(WHERE p.orden_vigente IS NOT NULL AND v.candidato_ref IS NULL),
        jsonb_agg(jsonb_build_object('orden_acta',e.orden,'orden_vigente',p.orden_vigente,
          'participacion_ref',e.participacion_ref,'candidato_ref',v.candidato_ref,
          'situacion',p.situacion,'razon',p.razon) ORDER BY e.orden)
 INTO v_total,v_orden,v_sin_vinculo,v_entradas
 FROM vec_bolsa_llamamientos.constitucion_entrada e
 LEFT JOIN posiciones p ON p.participacion_ref=e.participacion_ref AND p.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato v
   ON v.participacion_ref=e.participacion_ref AND v.acta_ref=v_const.acta_ref
 WHERE e.instantanea_ref=v_const.instantanea_ref
   AND e.version_instantanea=v_const.version_instantanea;
 IF v_total<>v_const.total_participaciones OR v_orden<>v_total OR v_sin_vinculo<>0
    OR v_entradas IS NULL
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(v_const.bolsa_ref,v_corte))<>v_total
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(v_const.bolsa_ref,v_corte) o
      WHERE o.politica_ref<>v_politica.politica_ref OR o.version_politica<>v_politica.version
        OR o.provisional IS DISTINCT FROM v_politica.provisional)
 THEN RAISE EXCEPTION 'B110: orden o vínculo incompleto' USING ERRCODE='P1101'; END IF;
 RETURN QUERY SELECT v_const.acta_ref,v_const.categoria_ref,v_const.bolsa_ref,v_const.version_bolsa,
   v_const.huella_bolsa_sha256,v_const.bolsa_canonica,v_const.instantanea_ref,
   v_const.version_instantanea,v_const.huella_instantanea_sha256,v_const.instantanea_canonica,
   v_politica.politica_ref,v_politica.version,convert_to(to_jsonb(v_politica)::text,'UTF8'),
   v_politica.provisional,v_corte,v_const.bolsa_vigente_hasta,v_politica.vigente_hasta,v_entradas;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text)
 TO vec_bolsa_llamamientos_ejecutor,vec_autorizacion_atestada_v3_propietario;

-- El recibo de origen no sustituye el llamamiento_emitido B7. Lo enlaza con
-- una sola necesidad CT y conserva el recibo y auditoría emitidos en el mismo
-- consumo V3. El evento se guarda en outbox append-only de Bolsa.
CREATE TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
 organizacion_ref text NOT NULL,unidad_ref text NOT NULL
  CHECK(unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'),
 expediente_ref text NOT NULL,
 version_expediente bigint NOT NULL CHECK(version_expediente>0),
 clave_ct text NOT NULL,necesidad_ref text NOT NULL,
 acta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
 categoria_ref text NOT NULL,
 bolsa_ref text NOT NULL,version_bolsa bigint NOT NULL,
 huella_bolsa_sha256 text NOT NULL CHECK(huella_bolsa_sha256~'^[0-9a-f]{64}$'),
 instantanea_ref text NOT NULL,version_instantanea bigint NOT NULL,
 huella_instantanea_sha256 text NOT NULL CHECK(huella_instantanea_sha256~'^[0-9a-f]{64}$'),
 politica_ref text NOT NULL,version_politica bigint NOT NULL,
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256~'^[0-9a-f]{64}$'),
 participacion_ref text NOT NULL,candidato_ref text NOT NULL,
 orden_seleccionado bigint NOT NULL CHECK(orden_seleccionado>0),
 version_token text NOT NULL CHECK(version_token~'^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 llamamiento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
 recibo_ref text NOT NULL UNIQUE,decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,evento_ref text NOT NULL UNIQUE,
 emitido_en timestamptz(6) NOT NULL,origen_canonico jsonb NOT NULL,
 origen_huella_sha256 text NOT NULL CHECK(origen_huella_sha256~'^[0-9a-f]{64}$'
  AND origen_huella_sha256=encode(sha256(convert_to(origen_canonico::text,'UTF8')),'hex')),
 CHECK(jsonb_typeof(origen_canonico)='object'
  AND origen_canonico->>'organizacion_ref' IS NOT DISTINCT FROM organizacion_ref
  AND origen_canonico->>'expediente_ref' IS NOT DISTINCT FROM expediente_ref
  AND origen_canonico->>'unidad_ref' IS NOT DISTINCT FROM unidad_ref
  AND origen_canonico->>'clave_ct' IS NOT DISTINCT FROM clave_ct
  AND origen_canonico->>'bolsa_ref' IS NOT DISTINCT FROM bolsa_ref
  AND origen_canonico->>'participacion_ref' IS NOT DISTINCT FROM participacion_ref
  AND origen_canonico->>'candidato_ref' IS NOT DISTINCT FROM candidato_ref
  AND origen_canonico->>'version_token' IS NOT DISTINCT FROM version_token),
 PRIMARY KEY(expediente_ref,clave_ct),
 FOREIGN KEY(bolsa_ref,version_bolsa,huella_bolsa_sha256)
  REFERENCES vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref,version,huella_bolsa_sha256),
 FOREIGN KEY(instantanea_ref,version_instantanea,huella_instantanea_sha256)
  REFERENCES vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256),
 FOREIGN KEY(instantanea_ref,version_instantanea,participacion_ref)
  REFERENCES vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,participacion_ref),
 FOREIGN KEY(politica_ref,version_politica)
  REFERENCES vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,version),
 CHECK(candidato_ref~'^can_[A-Za-z0-9_-]{22,128}$')
);
CREATE TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1(
 evento_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.origen_llamamiento_ct_v1(evento_ref),
 llamamiento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.origen_llamamiento_ct_v1(llamamiento_ref),
 tipo text NOT NULL CHECK(tipo='bolsa.llamamiento.ct.origen_confirmado'),
 estado text NOT NULL CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.origen_llamamiento_ct_v1 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.origen_llamamiento_ct_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.origen_llamamiento_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
ALTER TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
 p_expediente text,p_clave_ct text) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT o.origen_canonico || jsonb_build_object(
  'llamamiento_ref',o.llamamiento_ref,'recibo_ref',o.recibo_ref,
  'decision_ref',o.decision_ref,'auditoria_ref',o.auditoria_ref,
  'evento_ref',o.evento_ref,'emitido_en',o.emitido_en,'reutilizada',true)
 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
 JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
   ON ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l
   ON l.llamamiento_ref=o.llamamiento_ref AND l.recibo_ref=o.recibo_ref
  AND l.decision_ref=o.decision_ref AND l.bolsa_ref=o.bolsa_ref
 WHERE o.expediente_ref=p_expediente AND o.clave_ct=p_clave_ct
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(text,text)
 TO vec_bolsa_llamamientos_ejecutor,vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_lectura_origen_ct_v1(
 p_expediente text,p_clave_ct text) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT o.origen_canonico->'lectura'
 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
 JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
   ON ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l
   ON l.llamamiento_ref=o.llamamiento_ref AND l.recibo_ref=o.recibo_ref
  AND l.decision_ref=o.decision_ref AND l.bolsa_ref=o.bolsa_ref
 WHERE o.expediente_ref=p_expediente AND o.clave_ct=p_clave_ct
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_lectura_origen_ct_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_lectura_origen_ct_v1(text,text)
 TO vec_bolsa_llamamientos_ejecutor,vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(
 p_expediente text,p_clave_ct text) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(o.bolsa_ref,l.clave_idempotencia)
  || jsonb_build_object('origen_ct',vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
       o.expediente_ref,o.clave_ct))
 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
 JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
   ON ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l
   ON l.llamamiento_ref=o.llamamiento_ref AND l.recibo_ref=o.recibo_ref
  AND l.decision_ref=o.decision_ref AND l.bolsa_ref=o.bolsa_ref
 WHERE o.expediente_ref=p_expediente AND o.clave_ct=p_clave_ct
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(text,text)
 TO vec_bolsa_llamamientos_ejecutor,vec_autorizacion_atestada_v3_propietario;

-- No reconstruye B17: esta entrada CT usa las mismas filas nativas y el mismo
-- consumidor V3 en la transacción de reserva, conservando auditoria_ref real.
CREATE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(
 p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,
 p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,
 p_huella_finalizacion bytea,p_capacidad bytea,p_decision bytea,
 p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_origen jsonb)
RETURNS TABLE(emision jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
SET statement_timeout='15s' AS $f$
DECLARE
 previo vec_bolsa_llamamientos.llamamiento_emitido%ROWTYPE;
 origen_previo vec_bolsa_llamamientos.origen_llamamiento_ct_v1%ROWTYPE;
 consumo record;
 d jsonb;
 v_huella text;
 v_origen_huella text;
 v_evento text;
 v_canon_politica bytea;
 v_confirmacion jsonb;
 v_cuantos bigint;
 v_lectura jsonb;
 v_entrada jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_emitido IS NULL OR octet_length(p_huella_finalizacion) IS DISTINCT FROM 32
    OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones)<>1
    OR jsonb_typeof(p_participaciones->0)<>'string'
    OR jsonb_typeof(p_configuracion)<>'object' OR
       p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad',
       'fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE
    OR (SELECT count(*)<>10 OR bool_or(jsonb_typeof(value)<>'string'
      OR octet_length(value#>>'{}') NOT BETWEEN 2 AND 4000
      OR value#>>'{}'<>btrim(value#>>'{}')) FROM jsonb_each(p_configuracion))
    OR p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$'
    OR jsonb_typeof(p_origen)<>'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_origen))<>22
    OR p_origen ?& ARRAY['organizacion_ref','unidad_ref','expediente_ref','version_expediente','clave_ct',
      'necesidad_ref','acta_ref','categoria_ref','bolsa_ref','version_bolsa','huella_bolsa_sha256',
      'instantanea_ref','version_instantanea','huella_instantanea_sha256','politica_ref',
      'version_politica','huella_politica_sha256','participacion_ref','candidato_ref','orden_seleccionado','lectura','version_token'] IS NOT TRUE
    OR p_origen->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR p_origen->>'participacion_ref' IS DISTINCT FROM p_participaciones->>0
    OR p_origen->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_origen->>'clave_ct' IS NULL OR p_origen->>'expediente_ref' IS NULL
    OR p_origen->>'organizacion_ref' IS NULL OR p_origen->>'necesidad_ref' IS NULL
    OR p_origen->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_origen->>'version_token' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'B110: reserva CT inválida' USING ERRCODE='22023'; END IF;
 v_lectura:=p_origen->'lectura';
 IF jsonb_typeof(v_lectura)<>'object'
    OR v_lectura->>'acta_ref' IS DISTINCT FROM p_origen->>'acta_ref'
    OR v_lectura->>'categoria_ref' IS DISTINCT FROM p_origen->>'categoria_ref'
    OR v_lectura->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR v_lectura->>'version_bolsa' IS DISTINCT FROM p_origen->>'version_bolsa'
    OR v_lectura->>'huella_bolsa_sha256' IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR v_lectura->>'instantanea_ref' IS DISTINCT FROM p_origen->>'instantanea_ref'
    OR v_lectura->>'version_instantanea' IS DISTINCT FROM p_origen->>'version_instantanea'
    OR v_lectura->>'huella_instantanea_sha256' IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
    OR v_lectura->>'politica_ref' IS DISTINCT FROM p_origen->>'politica_ref'
    OR v_lectura->>'version_politica' IS DISTINCT FROM p_origen->>'version_politica'
    OR v_lectura->>'huella_politica_sha256' IS DISTINCT FROM p_origen->>'huella_politica_sha256'
    OR jsonb_typeof(v_lectura->'politica_provisional') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(v_lectura->'entradas')<>'array'
    OR (v_lectura->>'corte')::timestamptz>p_emitido
    OR encode(sha256(decode(v_lectura->>'bolsa_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR encode(sha256(decode(v_lectura->>'instantanea_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
 THEN RAISE EXCEPTION 'B110: lectura CT incompatible' USING ERRCODE='22023'; END IF;
 SELECT x.value INTO v_entrada FROM jsonb_array_elements(v_lectura->'entradas') x(value)
 WHERE x.value->>'participacion_ref'=p_origen->>'participacion_ref'
   AND x.value->>'candidato_ref'=p_origen->>'candidato_ref'
   AND x.value->>'orden_acta'=p_origen->>'orden_seleccionado'
   AND x.value->>'elegible'='true' AND x.value->>'orden_vigente'='1';
 IF NOT FOUND THEN RAISE EXCEPTION 'B110: selección no pertenece a lectura' USING ERRCODE='22023'; END IF;
 -- La validación de la lista y del orden se hace con la política efectiva B6.
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 v_origen_huella:=encode(sha256(convert_to(p_origen::text,'UTF8')),'hex');
 v_evento:='evento:bolsa:ct:'||encode(sha256(convert_to(p_origen->>'expediente_ref'||chr(31)||p_origen->>'clave_ct','UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave FOR UPDATE;
 IF FOUND THEN
  SELECT * INTO origen_previo FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
   WHERE expediente_ref=p_origen->>'expediente_ref' AND clave_ct=p_origen->>'clave_ct';
  IF NOT FOUND OR previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella
     OR previo.llamamiento_ref<>p_llamamiento OR previo.recibo_ref<>p_recibo
     OR origen_previo.origen_huella_sha256<>v_origen_huella
     OR origen_previo.llamamiento_ref<>previo.llamamiento_ref THEN
   RAISE EXCEPTION 'B110: clave CT divergente' USING ERRCODE='VBE01'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(p_bolsa,p_clave)
    || jsonb_build_object('origen_ct',vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
      p_origen->>'expediente_ref',p_origen->>'clave_ct')),true;
  RETURN;
 END IF;
 SELECT count(*) INTO v_cuantos
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref
   AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i ON i.instantanea_ref=c.instantanea_ref
   AND i.version=c.version_instantanea AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref
   AND e.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=e.participacion_ref
   AND vc.acta_ref=c.acta_ref AND vc.instantanea_ref=c.instantanea_ref
   AND vc.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 JOIN vec_bolsa_llamamientos.politica_orden_bolsa pol
   ON pol.politica_ref=o.politica_ref AND pol.version=o.version_politica
 WHERE c.acta_ref=p_origen->>'acta_ref' AND c.categoria_ref=p_origen->>'categoria_ref'
   AND c.bolsa_ref=p_bolsa AND c.version_bolsa=(p_origen->>'version_bolsa')::bigint
   AND c.huella_bolsa_sha256=p_origen->>'huella_bolsa_sha256'
   AND c.instantanea_ref=p_origen->>'instantanea_ref'
   AND c.version_instantanea=(p_origen->>'version_instantanea')::bigint
   AND c.huella_instantanea_sha256=p_origen->>'huella_instantanea_sha256'
   AND e.participacion_ref=p_origen->>'participacion_ref'
   AND e.orden=(p_origen->>'orden_seleccionado')::bigint
   AND vc.candidato_ref=p_origen->>'candidato_ref'
   AND o.orden_vigente=1
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND pol.provisional=(v_lectura->>'politica_provisional')::boolean
   AND encode(sha256(convert_to(to_jsonb(pol)::text,'UTF8')),'hex')=p_origen->>'huella_politica_sha256'
   AND b.estado='vigente' AND b.vigente_desde<=p_emitido
   AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_cuantos<>1 THEN RAISE EXCEPTION 'B110: origen CT no vigente o divergente' USING ERRCODE='23503'; END IF;
 -- La decisión completa que se conservará para replay se coteja con el
 -- orden B6 al corte original; ninguna entrada puede sustituirse en tránsito.
 SELECT count(*) INTO v_cuantos
 FROM jsonb_array_elements(v_lectura->'entradas') WITH ORDINALITY x(valor,n)
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=p_origen->>'instantanea_ref'
  AND e.version_instantanea=(p_origen->>'version_instantanea')::bigint
  AND e.orden=x.n AND e.participacion_ref=x.valor->>'participacion_ref'
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(
   p_bolsa,(v_lectura->>'corte')::timestamptz) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=e.participacion_ref AND vc.acta_ref=p_origen->>'acta_ref'
 WHERE x.valor->>'orden_acta'=e.orden::text
   AND x.valor->>'orden_vigente' IS NOT DISTINCT FROM o.orden_vigente::text
   AND x.valor->>'situacion' IS NOT DISTINCT FROM o.situacion
   AND x.valor->>'razon' IS NOT DISTINCT FROM o.razon
   AND x.valor->>'candidato_ref' IS NOT DISTINCT FROM coalesce(vc.candidato_ref,'')
   AND x.valor->>'elegible'=(o.orden_vigente IS NOT NULL)::text
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND o.provisional=(v_lectura->>'politica_provisional')::boolean;
 IF v_cuantos<>jsonb_array_length(v_lectura->'entradas') THEN
  RAISE EXCEPTION 'B110: orden completo divergente' USING ERRCODE='23503'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B110: decisión inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B110: emisión no autorizada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
  llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,
  participaciones,configuracion,huella_comando_sha256,huella_finalizacion,
  estado,emitido_en,decision_ref)
 VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,
  p_configuracion,v_huella,p_huella_finalizacion,'emision_reservada',p_emitido,consumo.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
  organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
  acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
  version_instantanea,huella_instantanea_sha256,politica_ref,version_politica,
  huella_politica_sha256,participacion_ref,candidato_ref,orden_seleccionado,version_token,
  llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
  origen_canonico,origen_huella_sha256)
 VALUES(p_origen->>'organizacion_ref',p_origen->>'unidad_ref',p_origen->>'expediente_ref',
  (p_origen->>'version_expediente')::bigint,p_origen->>'clave_ct',p_origen->>'necesidad_ref',
  p_origen->>'acta_ref',p_origen->>'categoria_ref',p_bolsa,(p_origen->>'version_bolsa')::bigint,
  p_origen->>'huella_bolsa_sha256',p_origen->>'instantanea_ref',
  (p_origen->>'version_instantanea')::bigint,p_origen->>'huella_instantanea_sha256',
  p_origen->>'politica_ref',(p_origen->>'version_politica')::bigint,
  p_origen->>'huella_politica_sha256',p_origen->>'participacion_ref',
  p_origen->>'candidato_ref',(p_origen->>'orden_seleccionado')::bigint,p_origen->>'version_token',
  p_llamamiento,p_recibo,consumo.decision_ref,consumo.auditoria_ref,v_evento,
  p_emitido,p_origen,v_origen_huella);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1
   (evento_ref,llamamiento_ref,tipo,estado,creada_en)
 VALUES(v_evento,p_llamamiento,'bolsa.llamamiento.ct.origen_confirmado','pendiente',p_emitido);
 v_confirmacion:=vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
  p_origen->>'expediente_ref',p_origen->>'clave_ct') || jsonb_build_object('reutilizada',false);
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,
  'bolsa_ref',p_bolsa,'estado','emision_reservada_resultado_pendiente',
  'participaciones',p_participaciones,'configuracion',p_configuracion,
  'emitido_en',p_emitido,'contactos','[]'::jsonb,'origen_ct',v_confirmacion),false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(
 text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,
 numeric,numeric,bytea,bytea,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(
 text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,
 numeric,numeric,bytea,bytea,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;

-- Solo revela la referencia opaca previa a la captura de permiso. No lee
-- entradas, candidatos ni bytes canónicos. La selección sigue en el lector.
CREATE FUNCTION vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(
 p_expediente text,p_clave text,p_categoria text,p_unidad text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' AS $f$
DECLARE ref text;unidad text;categoria text;total bigint;ahora timestamptz(6):=statement_timestamp();
BEGIN
 IF p_expediente IS NULL OR p_clave IS NULL OR p_categoria IS NULL OR p_unidad IS NULL
 OR p_categoria<>btrim(p_categoria) OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
 OR p_unidad !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
 THEN RAISE EXCEPTION 'B110: origen CT inválido' USING ERRCODE='22023'; END IF;
 SELECT o.bolsa_ref,o.unidad_ref,o.categoria_ref INTO ref,unidad,categoria
 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
 WHERE o.expediente_ref=p_expediente AND o.clave_ct=p_clave;
 IF FOUND THEN
  IF unidad IS DISTINCT FROM p_unidad OR categoria IS DISTINCT FROM p_categoria
  THEN RAISE EXCEPTION 'B110: reserva CT incompatible' USING ERRCODE='42501'; END IF;
  RETURN ref;
 END IF;
 SELECT count(*),min(c.bolsa_ref) INTO total,ref
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
 ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
 AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
 AND b.vigente_desde<=ahora AND (b.vigente_hasta IS NULL OR ahora<b.vigente_hasta)
 AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
 WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
 AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF total<>1 THEN RAISE EXCEPTION 'B110: origen ausente o ambiguo' USING ERRCODE='P1101'; END IF;
 RETURN ref;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(text,text,text,text)
 TO vec_autorizacion_atestada_v3_propietario;

COMMIT;
