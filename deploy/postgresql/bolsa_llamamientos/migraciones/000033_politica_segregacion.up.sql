\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000033', 0));

-- Duda 6 de RRHH: qué operaciones de B8 valida una segunda persona. Hasta
-- ahora la lista era un literal ('excluir'). Esta migración la traslada a una
-- política versionada de solo adición que publica el catálogo configurable
-- vec.bolsa.roles_segregacion. La comprobación sigue en la base de datos:
--  * el CHECK de 000019 (exclusión con otra persona) se conserva como mínimo
--    fijo y ninguna política puede quitar 'excluir';
--  * un disparador aplica la política vigente a cada nueva operación y deja
--    constancia de la versión aplicada en la propia fila.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_segregacion') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute
                WHERE attrelid = 'vec_bolsa_llamamientos.operacion_situacion_participacion'::regclass
                  AND attname = 'politica_segregacion_version' AND NOT attisdropped) THEN
  RAISE EXCEPTION 'estado incompatible para la politica de segregacion B8' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.politica_segregacion (
    version bigint PRIMARY KEY CHECK (version >= 1),
    catalogo_ref text NOT NULL CHECK (octet_length(catalogo_ref) BETWEEN 1 AND 512 AND catalogo_ref = btrim(catalogo_ref)),
    catalogo_sha256 text NOT NULL CHECK (catalogo_sha256 ~ '^[a-f0-9]{64}$'),
    -- Lista canónica: sin nulos ni repetidos, en el orden de B8, y siempre
    -- con la exclusión, que B2 no permite deshacer.
    operaciones text[] NOT NULL CHECK (
        array_ndims(operaciones) = 1
        AND operaciones IN (ARRAY['excluir'], ARRAY['pausar','excluir'], ARRAY['reactivar','excluir'], ARRAY['pausar','reactivar','excluir'])),
    publicada_en timestamptz(6) NOT NULL
);
COMMENT ON TABLE vec_bolsa_llamamientos.politica_segregacion IS
    'Versiones de solo adición de las operaciones B8 que exigen validador distinto del actor. La vigente es la de mayor versión.';
ALTER TABLE vec_bolsa_llamamientos.politica_segregacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_segregacion FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_segregacion_solo_propietario ON vec_bolsa_llamamientos.politica_segregacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.politica_segregacion FROM PUBLIC;
CREATE TRIGGER politica_segregacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.politica_segregacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Versión 1: exactamente la conducta anterior.
INSERT INTO vec_bolsa_llamamientos.politica_segregacion(version, catalogo_ref, catalogo_sha256, operaciones, publicada_en)
VALUES (1, 'migracion:bolsa_llamamientos:000033:defecto', encode(sha256(convert_to('excluir', 'UTF8')), 'hex'), ARRAY['excluir'], clock_timestamp());

-- Las filas anteriores quedan sin versión (rigió el literal de 000019); toda
-- fila nueva la lleva. NOT VALID limita la exigencia a las nuevas.
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion
    ADD COLUMN politica_segregacion_version bigint REFERENCES vec_bolsa_llamamientos.politica_segregacion(version);
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion
    ADD CONSTRAINT operacion_situacion_politica_segregacion_obligatoria CHECK (politica_segregacion_version IS NOT NULL) NOT VALID;

CREATE FUNCTION vec_bolsa_llamamientos.aplicar_politica_segregacion()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
DECLARE v_politica record;
BEGIN
 -- Lectura compartida frente a la publicación, que toma el cerrojo exclusivo.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_segregacion', 0));
 SELECT version, operaciones INTO STRICT v_politica
   FROM vec_bolsa_llamamientos.politica_segregacion ORDER BY version DESC LIMIT 1;
 IF NEW.operacion = ANY (v_politica.operaciones) AND NEW.validador = NEW.actor THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='operacion sin segunda persona';
 END IF;
 NEW.politica_segregacion_version := v_politica.version;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.aplicar_politica_segregacion() FROM PUBLIC;
CREATE TRIGGER operacion_situacion_politica_segregacion BEFORE INSERT ON vec_bolsa_llamamientos.operacion_situacion_participacion
    FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.aplicar_politica_segregacion();

-- Publica una entrada exacta del catálogo. Solo crea versión si difiere de la
-- vigente; nunca admite una lista sin la exclusión.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_segregacion_v1(p_catalogo_ref text, p_catalogo_sha256 text, p_operaciones text[])
RETURNS TABLE(version bigint, reutilizada boolean, catalogo_ref text, operaciones text[])
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' SET statement_timeout = '5s' AS $f$
DECLARE v_canonica text[]; v_vigente record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 IF p_operaciones IS NULL OR array_ndims(p_operaciones) IS DISTINCT FROM 1 OR array_position(p_operaciones, NULL) IS NOT NULL
    OR p_catalogo_ref IS NULL OR p_catalogo_sha256 IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de segregacion invalida';
 END IF;
 v_canonica := ARRAY(SELECT o FROM unnest(ARRAY['pausar','reactivar','excluir']) WITH ORDINALITY AS c(o, n)
                      WHERE o = ANY (p_operaciones) ORDER BY n);
 IF cardinality(v_canonica) <> cardinality(p_operaciones) OR NOT ('excluir' = ANY (v_canonica)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='politica de segregacion invalida';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica_segregacion', 0));
 SELECT p.version, p.catalogo_ref, p.catalogo_sha256, p.operaciones INTO STRICT v_vigente
   FROM vec_bolsa_llamamientos.politica_segregacion p ORDER BY p.version DESC LIMIT 1;
 IF v_vigente.catalogo_ref = p_catalogo_ref AND v_vigente.catalogo_sha256 = p_catalogo_sha256 AND v_vigente.operaciones = v_canonica THEN
  RETURN QUERY SELECT v_vigente.version, true, v_vigente.catalogo_ref, v_vigente.operaciones;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_segregacion(version, catalogo_ref, catalogo_sha256, operaciones, publicada_en)
 VALUES (v_vigente.version + 1, p_catalogo_ref, p_catalogo_sha256, v_canonica, clock_timestamp());
 RETURN QUERY SELECT v_vigente.version + 1, false, p_catalogo_ref, v_canonica;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_politica_segregacion_v1()
RETURNS TABLE(version bigint, catalogo_ref text, operaciones text[])
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 RETURN QUERY SELECT p.version, p.catalogo_ref, p.operaciones
   FROM vec_bolsa_llamamientos.politica_segregacion p ORDER BY p.version DESC LIMIT 1;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_politica_segregacion_v1(text,text,text[]) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_politica_segregacion_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_politica_segregacion_v1(text,text,text[]) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_politica_segregacion_v1() TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
