\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000032', 0));

-- Reglamento de bolsas, arts. 10 y 11 (duda 62 de RRHH): hasta ahora las
-- transiciones de situación de B2 eran un literal de
-- registrar_situacion_participacion_v1 (000012). Esta migración las traslada a
-- una política versionada de solo adición que publica el catálogo
-- configurable (b28.transiciones.<origen>), igual que 000033 hace con la
-- segunda persona de B8. La comprobación sigue en la base de datos:
--  * la versión 1 es el literal anterior más «disponible>disponible_desde»
--    (decisión de dirección): una suspensión con fecha de fin deja a la
--    persona fuera del turno hasta ese día (000037) y ese cambio también
--    pasa por la política;
--  * invariantes fijas: nunca se sale de «excluido», no hay transiciones a
--    la misma situación y desde toda situación se puede dar de baja
--    definitiva («excluido», art. 11);
--  * la ÚNICA excepción a «nunca se sale de excluido» es la readmisión por
--    un recurso de reposición estimado y registrado contra la sanción que
--    excluyó (readmitir_participacion_por_recurso_v1, más abajo). No la
--    concede ninguna versión de la política ni puede publicarse desde el
--    catálogo: solo la invoca la función de recurso de 000037;
--  * cada nuevo cambio deja en su fila la versión de política aplicada.
-- Las situaciones ya registradas (incluidas las renuncias vigentes) no se
-- tocan: la historia es de solo adición.
DO $precondicion$
DECLARE v_fuente text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.recurso_sancion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz)') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.transiciones_situacion_canonicas(text[])') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute
                WHERE attrelid = 'vec_bolsa_llamamientos.situacion_participacion'::regclass
                  AND attname = 'politica_transiciones_version' AND NOT attisdropped) THEN
  RAISE EXCEPTION 'estado incompatible para la politica de transiciones B2' USING ERRCODE='55000';
 END IF;
 -- Solo se sustituye el cuerpo exacto instalado por 000012.
 SELECT p.prosrc INTO v_fuente FROM pg_catalog.pg_proc p
  WHERE p.oid = to_regprocedure('vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF v_fuente IS NULL OR md5(v_fuente) <> 'b310b46f4d9979503ecf37e64b5174f8' THEN
  RAISE EXCEPTION 'registrar_situacion_participacion_v1 no es la de 000012' USING ERRCODE='55000';
 END IF;
END $precondicion$;

-- Forma canónica de una política: pares «origen>destino» sin nulos ni
-- repetidos, en el orden de las situaciones de B2. Devuelve NULL si la lista
-- incumple una invariante fija; el CHECK de la tabla la exige canónica.
CREATE FUNCTION vec_bolsa_llamamientos.transiciones_situacion_canonicas(p_transiciones text[])
RETURNS text[] LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $f$
DECLARE
 v_orden constant text[] := ARRAY['disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde'];
 v_canonica text[];
 v_par text;
 v_origen text;
BEGIN
 IF array_ndims(p_transiciones) IS DISTINCT FROM 1 OR array_position(p_transiciones, NULL) IS NOT NULL
    OR cardinality(p_transiciones) > 42 THEN
  RETURN NULL;
 END IF;
 FOREACH v_par IN ARRAY p_transiciones LOOP
  IF split_part(v_par, '>', 1) <> ALL (v_orden) OR split_part(v_par, '>', 2) <> ALL (v_orden)
     OR v_par <> split_part(v_par, '>', 1) || '>' || split_part(v_par, '>', 2)
     OR split_part(v_par, '>', 1) = split_part(v_par, '>', 2)
     OR split_part(v_par, '>', 1) = 'excluido' THEN
   RETURN NULL;
  END IF;
 END LOOP;
 v_canonica := ARRAY(
  SELECT o.s || '>' || d.s
    FROM unnest(v_orden) WITH ORDINALITY AS o(s, n)
    CROSS JOIN unnest(v_orden) WITH ORDINALITY AS d(s, m)
   WHERE (o.s || '>' || d.s) = ANY (p_transiciones)
   ORDER BY o.n, d.m);
 IF cardinality(v_canonica) <> cardinality(p_transiciones) THEN
  RETURN NULL;
 END IF;
 FOREACH v_origen IN ARRAY v_orden LOOP
  IF v_origen <> 'excluido' AND NOT ((v_origen || '>excluido') = ANY (v_canonica)) THEN
   RETURN NULL;
  END IF;
 END LOOP;
 RETURN v_canonica;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.transiciones_situacion_canonicas(text[]) FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.politica_transiciones_situacion (
    version bigint PRIMARY KEY CHECK (version >= 1),
    catalogo_ref text NOT NULL CHECK (octet_length(catalogo_ref) BETWEEN 1 AND 512 AND catalogo_ref = btrim(catalogo_ref)),
    catalogo_sha256 text NOT NULL CHECK (catalogo_sha256 ~ '^[a-f0-9]{64}$'),
    transiciones text[] NOT NULL CHECK (transiciones IS NOT DISTINCT FROM vec_bolsa_llamamientos.transiciones_situacion_canonicas(transiciones)),
    publicada_en timestamptz(6) NOT NULL
);
COMMENT ON TABLE vec_bolsa_llamamientos.politica_transiciones_situacion IS
    'Versiones de solo adición de las transiciones de situación B2 admitidas (pares origen>destino). La vigente es la de mayor versión.';
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_situacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_transiciones_situacion FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_transiciones_situacion_solo_propietario ON vec_bolsa_llamamientos.politica_transiciones_situacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.politica_transiciones_situacion FROM PUBLIC;
CREATE TRIGGER politica_transiciones_situacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.politica_transiciones_situacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Versión 1: el literal de 000012 más «disponible>disponible_desde».
INSERT INTO vec_bolsa_llamamientos.politica_transiciones_situacion(version, catalogo_ref, catalogo_sha256, transiciones, publicada_en)
SELECT 1, 'migracion:bolsa_llamamientos:000032:defecto', encode(sha256(convert_to(array_to_string(t, ','), 'UTF8')), 'hex'), t, clock_timestamp()
  FROM (SELECT vec_bolsa_llamamientos.transiciones_situacion_canonicas(ARRAY[
   'disponible>no_disponible','disponible>pendiente_incorporacion','disponible>renuncia','disponible>excluido','disponible>disponible_desde',
   'no_disponible>disponible','no_disponible>excluido',
   'pendiente_incorporacion>trabajando','pendiente_incorporacion>disponible','pendiente_incorporacion>renuncia','pendiente_incorporacion>excluido',
   'trabajando>disponible','trabajando>disponible_desde','trabajando>excluido',
   'disponible_desde>disponible','disponible_desde>excluido',
   'renuncia>disponible','renuncia>excluido']) AS t) v;
DO $v1$
BEGIN
 IF (SELECT cardinality(transiciones) FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version = 1) IS DISTINCT FROM 18 THEN
  RAISE EXCEPTION 'la version 1 no es la esperada' USING ERRCODE='55000';
 END IF;
END $v1$;

-- Las filas anteriores quedan sin versión (rigió el literal de 000012) y las
-- de constitución no son transiciones; todo cambio registrado desde aquí la
-- lleva.
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion
    ADD COLUMN politica_transiciones_version bigint REFERENCES vec_bolsa_llamamientos.politica_transiciones_situacion(version);

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(p_bolsa_ref text,p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE anterior record; consumo record; decision jsonb; v_politica record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_situacion NOT IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde') OR p_desde IS NULL OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_clave_idempotencia<>btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256 OR p_recibo_ref IS NULL OR p_recibo_ref<>btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 OR p_registrada_en IS NULL OR (p_situacion='disponible_desde' AND p_fecha_disponible IS NULL) OR (p_situacion<>'disponible_desde' AND p_fecha_disponible IS NOT NULL) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='situacion invalida'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='participacion ajena a la bolsa'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p_participacion_ref ORDER BY desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar' OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia AND (s.situacion<>p_situacion OR s.motivo<>p_motivo OR s.fecha_disponible IS DISTINCT FROM p_fecha_disponible)) THEN RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro comando'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia) THEN RETURN QUERY SELECT true, s.recibo_ref, s.situacion, s.desde, s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia; RETURN; END IF;
 IF p_desde < anterior.desde THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='desde anterior a la situacion vigente'; END IF;
 IF p_situacion='disponible_desde' AND p_fecha_disponible<=p_registrada_en THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='fecha disponible no futura'; END IF;
 -- 000032: la tabla de transiciones es la política vigente, no un literal.
 -- Cerrojo compartido frente a la publicación, que toma el exclusivo.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
 SELECT p.version, p.transiciones INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
 IF NOT ((anterior.situacion || '>' || p_situacion) = ANY (v_politica.transiciones)) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version) VALUES(p_participacion_ref,p_situacion,p_desde,NULL,p_fecha_disponible,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref,v_politica.version);
 RETURN QUERY SELECT false,p_recibo_ref,p_situacion,p_desde,p_fecha_disponible;
END $f$;


-- Publica la política derivada del catálogo. Solo crea versión si difiere de
-- la vigente; nunca admite una lista que incumpla las invariantes fijas.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(p_catalogo_ref text, p_catalogo_sha256 text, p_transiciones text[])
RETURNS TABLE(version bigint, reutilizada boolean, catalogo_ref text, transiciones text[])
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' SET statement_timeout = '5s' AS $f$
DECLARE v_canonica text[]; v_vigente record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 IF p_catalogo_ref IS NULL OR p_catalogo_ref <> btrim(p_catalogo_ref) OR octet_length(p_catalogo_ref) NOT BETWEEN 1 AND 512
    OR p_catalogo_sha256 IS NULL OR p_catalogo_sha256 !~ '^[a-f0-9]{64}$' OR p_transiciones IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de transiciones invalida';
 END IF;
 v_canonica := vec_bolsa_llamamientos.transiciones_situacion_canonicas(p_transiciones);
 IF v_canonica IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de transiciones invalida';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
 SELECT p.version, p.catalogo_ref, p.catalogo_sha256, p.transiciones INTO STRICT v_vigente
   FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
 IF v_vigente.catalogo_ref = p_catalogo_ref AND v_vigente.catalogo_sha256 = p_catalogo_sha256 AND v_vigente.transiciones = v_canonica THEN
  RETURN QUERY SELECT v_vigente.version, true, v_vigente.catalogo_ref, v_vigente.transiciones;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_transiciones_situacion(version, catalogo_ref, catalogo_sha256, transiciones, publicada_en)
 VALUES (v_vigente.version + 1, p_catalogo_ref, p_catalogo_sha256, v_canonica, clock_timestamp());
 RETURN QUERY SELECT v_vigente.version + 1, false, p_catalogo_ref, v_canonica;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1()
RETURNS TABLE(version bigint, catalogo_ref text, transiciones text[])
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 RETURN QUERY SELECT p.version, p.catalogo_ref, p.transiciones
   FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
END $f$;

-- Readmisión por recurso de reposición estimado: la ÚNICA salida de
-- «excluido» y el único cambio de situación que no exige que la política
-- vigente admita la transición cuando el origen es «excluido». Para cualquier
-- otro origen (una suspensión revocada) la política sí se exige. Sin EXECUTE
-- para nadie y con SECURITY INVOKER: solo la alcanzan funciones del
-- propietario, y además comprueba que quien la llama directamente es la
-- función de recurso de 000037 (registrar_recurso_sancion_participacion_v2),
-- que ya ha consumido la autorización, anotado el recurso y comprobado que el
-- catálogo declara revocatorio ese estado. Aquí se exige que ese recurso sea
-- el último estado registrado de la sanción, en este mismo instante, y que la
-- situación vigente sea la que dejó esa sanción. La situación restaurada es
-- la anterior a la sanción (si era «disponible_desde» y la fecha ya pasó,
-- «disponible»). La fila lleva la versión de política vigente.
CREATE FUNCTION vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(
 p_participacion_ref text, p_sancion_ref text, p_recurso_clave_idempotencia text, p_motivo text, p_actor text,
 p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz)
RETURNS TABLE(situacion text, fecha_disponible timestamptz, desde timestamptz, politica_transiciones_version bigint)
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v_pila text; v_sancion record; v_ultimo record; v_actual record; v_previa record; v_politica record;
 v_situacion text; v_fecha timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='readmision no autorizada'; END IF;
 -- La pila empieza por esta función; la siguiente función de la pila (las
 -- líneas intermedias citan la sentencia) es quien la llama.
 GET DIAGNOSTICS v_pila = PG_CONTEXT;
 SELECT t.l INTO v_pila FROM unnest(string_to_array(v_pila, E'\n')) WITH ORDINALITY AS t(l, n)
  WHERE t.n > 1 AND (t.l LIKE 'PL/pgSQL function %' OR t.l LIKE 'SQL function %') ORDER BY t.n LIMIT 1;
 IF v_pila IS NULL OR v_pila NOT LIKE 'PL/pgSQL function vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(%' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='readmision no autorizada';
 END IF;
 IF p_participacion_ref IS NULL OR p_sancion_ref IS NULL OR p_recurso_clave_idempotencia IS NULL
    OR p_motivo IS NULL OR p_motivo <> btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000
    OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_recibo_ref IS NULL OR p_registrada_en IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida';
 END IF;
 SELECT s.* INTO v_sancion FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.sancion_ref = p_sancion_ref AND s.participacion_ref = p_participacion_ref;
 IF NOT FOUND OR v_sancion.situacion_desde IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida';
 END IF;
 SELECT r.* INTO v_ultimo FROM vec_bolsa_llamamientos.recurso_sancion_participacion r
  WHERE r.sancion_ref = p_sancion_ref ORDER BY r.registrada_en DESC, r.clave_idempotencia DESC LIMIT 1;
 IF NOT FOUND OR v_ultimo.clave_idempotencia <> p_recurso_clave_idempotencia OR v_ultimo.registrada_en <> p_registrada_en THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso no vigente';
 END IF;
 SELECT sp.* INTO STRICT v_actual FROM vec_bolsa_llamamientos.situacion_participacion sp
  WHERE sp.participacion_ref = p_participacion_ref ORDER BY sp.desde DESC LIMIT 1 FOR UPDATE;
 IF v_actual.desde <> v_sancion.situacion_desde OR p_registrada_en <= v_actual.desde
    OR (v_actual.situacion = 'excluido' AND v_sancion.efecto <> 'excluir') THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida';
 END IF;
 SELECT sp.* INTO v_previa FROM vec_bolsa_llamamientos.situacion_participacion sp
  WHERE sp.participacion_ref = p_participacion_ref AND sp.desde < v_sancion.situacion_desde ORDER BY sp.desde DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida'; END IF;
 v_situacion := v_previa.situacion; v_fecha := v_previa.fecha_disponible;
 IF v_situacion = 'disponible_desde' AND v_fecha <= p_registrada_en THEN
  v_situacion := 'disponible'; v_fecha := NULL;
 END IF;
 IF v_situacion = 'excluido' OR v_situacion = v_actual.situacion THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida';
 END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
 SELECT p.version, p.transiciones INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
 IF v_actual.situacion <> 'excluido' AND NOT ((v_actual.situacion || '>' || v_situacion) = ANY (v_politica.transiciones)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES (p_participacion_ref, v_situacion, p_registrada_en, NULL, v_fecha, p_motivo, p_actor, p_registrada_en, p_clave_idempotencia, p_recibo_ref, v_politica.version);
 RETURN QUERY SELECT v_situacion, v_fecha, p_registrada_en, v_politica.version;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz) FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[]) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(text,text,text[]) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1() TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
