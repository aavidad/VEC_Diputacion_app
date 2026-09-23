\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000018', 0));

-- B6: política explicable y versionada. Los valores iniciales son una
-- configuración técnica provisional; no se presentan como Reglamento de Granada.
CREATE TABLE vec_bolsa_llamamientos.politica_orden_bolsa (
    politica_ref text NOT NULL,
    bolsa_ref text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    criterio text NOT NULL CHECK (criterio = 'puntuacion_desc_acta'),
    tipo_lista text NOT NULL CHECK (tipo_lista IN ('cerrada','rotatoria')),
    reposicion text NOT NULL CHECK (reposicion IN ('misma_posicion','fin_lista','no_disponible_hasta_fecha')),
    provisional boolean NOT NULL,
    rotulo text NOT NULL CHECK (rotulo = btrim(rotulo) AND octet_length(rotulo) BETWEEN 1 AND 200),
    actor text NOT NULL CHECK (actor = btrim(actor) AND octet_length(actor) BETWEEN 1 AND 256),
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (politica_ref, version),
    UNIQUE (bolsa_ref, version),
    CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde),
    CHECK (NOT provisional OR rotulo = 'Provisional, pendiente de RRHH (dudas 13–14)')
);
ALTER TABLE vec_bolsa_llamamientos.politica_orden_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.politica_orden_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY politica_orden_bolsa_solo_propietario ON vec_bolsa_llamamientos.politica_orden_bolsa
    TO vec_bolsa_llamamientos_propietario
    USING (current_user = 'vec_bolsa_llamamientos_propietario')
    WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.politica_orden_bolsa FROM PUBLIC;
CREATE TRIGGER politica_orden_bolsa_inmutable BEFORE UPDATE OR DELETE
    ON vec_bolsa_llamamientos.politica_orden_bolsa FOR EACH ROW
    EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa (
    reposicion_ref text PRIMARY KEY,
    bolsa_ref text NOT NULL,
    participacion_ref text NOT NULL,
    situacion_recibo_ref text NOT NULL UNIQUE,
    politica_ref text NOT NULL,
    version_politica bigint NOT NULL,
    reposicion text NOT NULL CHECK (reposicion IN ('misma_posicion','fin_lista','no_disponible_hasta_fecha')),
    posicion_resultante bigint NOT NULL CHECK (posicion_resultante > 0),
    aplicada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (politica_ref, version_politica)
        REFERENCES vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref, version)
);
CREATE INDEX reposicion_orden_bolsa_lectura
    ON vec_bolsa_llamamientos.reposicion_orden_bolsa(bolsa_ref, aplicada_en, participacion_ref);
ALTER TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY reposicion_orden_bolsa_solo_propietario ON vec_bolsa_llamamientos.reposicion_orden_bolsa
    TO vec_bolsa_llamamientos_propietario
    USING (current_user = 'vec_bolsa_llamamientos_propietario')
    WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.reposicion_orden_bolsa FROM PUBLIC;
CREATE TRIGGER reposicion_orden_bolsa_inmutable BEFORE UPDATE OR DELETE
    ON vec_bolsa_llamamientos.reposicion_orden_bolsa FOR EACH ROW
    EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.crear_politica_orden_provisional_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_ref text;
BEGIN
 v_ref := 'politica:orden:provisional:' || encode(sha256(convert_to(NEW.bolsa_ref,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(
   politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en
 ) VALUES (
   v_ref,NEW.bolsa_ref,1,'puntuacion_desc_acta','rotatoria','misma_posicion',true,
   'Provisional, pendiente de RRHH (dudas 13–14)','sistema:migracion:000018',NEW.vigente_desde,NULL,clock_timestamp()
 ) ON CONFLICT (bolsa_ref,version) DO NOTHING;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.crear_politica_orden_provisional_v1() FROM PUBLIC;
CREATE TRIGGER bolsa_constituida_crea_politica_orden
AFTER INSERT ON vec_bolsa_llamamientos.bolsa_constituida
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.crear_politica_orden_provisional_v1();

INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(
 politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en
)
SELECT DISTINCT ON (b.bolsa_ref) 'politica:orden:provisional:' || encode(sha256(convert_to(b.bolsa_ref,'UTF8')),'hex'),
       b.bolsa_ref,1,'puntuacion_desc_acta','rotatoria','misma_posicion',true,
       'Provisional, pendiente de RRHH (dudas 13–14)','sistema:migracion:000018',b.vigente_desde,NULL,clock_timestamp()
  FROM vec_bolsa_llamamientos.bolsa_constituida b ORDER BY b.bolsa_ref,b.vigente_desde;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_reposicion_orden_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_anterior text; v_bolsa text; v_politica record; v_posicion bigint;
BEGIN
 SELECT s.situacion INTO v_anterior
   FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref=NEW.participacion_ref AND (s.desde,s.clave_idempotencia)<>(NEW.desde,NEW.clave_idempotencia)
  ORDER BY s.desde DESC LIMIT 1;
 IF v_anterior IS DISTINCT FROM 'trabajando' OR NEW.situacion NOT IN ('disponible','disponible_desde') THEN RETURN NEW; END IF;
 SELECT c.bolsa_ref INTO STRICT v_bolsa
   FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
  WHERE e.participacion_ref=NEW.participacion_ref;
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa p
  WHERE p.bolsa_ref=v_bolsa AND p.vigente_desde<=NEW.desde AND (p.vigente_hasta IS NULL OR p.vigente_hasta>NEW.desde)
  ORDER BY p.version DESC LIMIT 1;
 IF v_politica.reposicion='fin_lista' THEN
   SELECT count(*)+1 INTO v_posicion
     FROM vec_bolsa_llamamientos.constitucion_entrada e
     JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
     JOIN LATERAL (SELECT s.situacion,s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=e.participacion_ref ORDER BY s.desde DESC LIMIT 1) s ON true
    WHERE c.bolsa_ref=v_bolsa AND e.participacion_ref<>NEW.participacion_ref
      AND (s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=NEW.desde));
 ELSE
   SELECT count(*) INTO v_posicion
     FROM vec_bolsa_llamamientos.constitucion_entrada e
     JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
     JOIN vec_bolsa_llamamientos.constitucion_entrada propia ON propia.instantanea_ref=e.instantanea_ref AND propia.version_instantanea=e.version_instantanea AND propia.participacion_ref=NEW.participacion_ref
     JOIN LATERAL (SELECT s.situacion,s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=e.participacion_ref ORDER BY s.desde DESC LIMIT 1) s ON true
    WHERE c.bolsa_ref=v_bolsa AND e.orden<=propia.orden
      AND (e.participacion_ref=NEW.participacion_ref OR s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=NEW.desde));
 END IF;
 INSERT INTO vec_bolsa_llamamientos.reposicion_orden_bolsa(reposicion_ref,bolsa_ref,participacion_ref,situacion_recibo_ref,politica_ref,version_politica,reposicion,posicion_resultante,aplicada_en,registrada_en)
 VALUES('reposicion:'||encode(sha256(convert_to(NEW.recibo_ref,'UTF8')),'hex'),v_bolsa,NEW.participacion_ref,NEW.recibo_ref,v_politica.politica_ref,v_politica.version,v_politica.reposicion,greatest(v_posicion,1),NEW.desde,clock_timestamp());
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_reposicion_orden_v1() FROM PUBLIC;
CREATE TRIGGER situacion_participacion_registra_reposicion
AFTER INSERT ON vec_bolsa_llamamientos.situacion_participacion
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.registrar_reposicion_orden_v1();

CREATE FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa_ref text,p_en timestamptz)
RETURNS TABLE(politica_ref text,version_politica bigint,criterio text,tipo_lista text,reposicion text,provisional boolean,rotulo text,actor text,vigente_desde timestamptz,participacion_ref text,orden_acta bigint,orden_vigente bigint,situacion text,razon text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 WITH politica AS (
  SELECT p.* FROM vec_bolsa_llamamientos.politica_orden_bolsa p
   WHERE p.bolsa_ref=p_bolsa_ref AND p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
   ORDER BY p.version DESC LIMIT 1
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         (s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)) AS ocupa_turno,
         r.aplicada_en AS repuesta_en
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en FROM vec_bolsa_llamamientos.reposicion_orden_bolsa ro WHERE ro.bolsa_ref=c.bolsa_ref AND ro.participacion_ref=e.participacion_ref AND ro.aplicada_en<=p_en ORDER BY ro.aplicada_en DESC LIMIT 1) r ON true
   WHERE c.bolsa_ref=p_bolsa_ref
 ), elegibles AS (
  SELECT b.participacion_ref,row_number() OVER(ORDER BY
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN NULL ELSE b.orden_acta END,
    b.repuesta_en,b.orden_acta,b.participacion_ref)::bigint AS orden_vigente
   FROM base b CROSS JOIN politica p WHERE b.ocupa_turno
 )
 SELECT p.politica_ref,p.version,p.criterio,p.tipo_lista,p.reposicion,p.provisional,p.rotulo,p.actor,p.vigente_desde,
        b.participacion_ref,b.orden_acta,e.orden_vigente,b.situacion,
        CASE WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'
             WHEN NOT b.ocupa_turno AND b.situacion='trabajando' THEN 'trabajando'
             WHEN NOT b.ocupa_turno THEN 'sin_turno'
             WHEN b.repuesta_en IS NOT NULL AND e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'reposicion_tras_contrato'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'pausa'
             ELSE 'orden_acta' END
   FROM base b CROSS JOIN politica p LEFT JOIN elegibles e USING(participacion_ref)
  ORDER BY e.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref
 $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;

-- B7 reserva contra la misma lectura B6 que muestra el asistente. Se sustituye
-- únicamente la validación heredada del acta y se conserva su consumo V3,
-- idempotencia, recibo y transacción originales.
DO $ajuste_b7$
DECLARE
 v_def text;
 v_validas_ant text := $sql$SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref;$sql$;
 v_orden_ant text := $sql$SELECT count(*) INTO v_ordenadas FROM (SELECT e.orden,lag(e.orden) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref) q WHERE anterior IS NULL OR orden>anterior;$sql$;
 v_validas_nueva text := $sql$SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL;$sql$;
 v_orden_nueva text := $sql$SELECT count(*) INTO v_ordenadas FROM (SELECT o.orden_vigente AS orden,lag(o.orden_vigente) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL) q WHERE anterior IS NULL OR orden>anterior;$sql$;
BEGIN
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO STRICT v_def;
 IF strpos(v_def,v_validas_ant)=0 OR strpos(v_def,v_orden_ant)=0 OR strpos(substr(v_def,strpos(v_def,v_validas_ant)+length(v_validas_ant)),v_validas_ant)<>0 OR strpos(substr(v_def,strpos(v_def,v_orden_ant)+length(v_orden_ant)),v_orden_ant)<>0 THEN
  RAISE EXCEPTION 'reserva B7 incompatible con ajuste B6' USING ERRCODE='55000';
 END IF;
 EXECUTE replace(replace(v_def,v_validas_ant,v_validas_nueva),v_orden_ant,v_orden_nueva);
END $ajuste_b7$;
COMMIT;
