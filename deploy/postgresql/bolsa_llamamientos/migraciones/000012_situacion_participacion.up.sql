\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000012', 0));

-- B2, Petición RRHH p.2: histórico append-only de la situación de cada participación.
CREATE TABLE vec_bolsa_llamamientos.situacion_participacion (
    participacion_ref text NOT NULL,
    situacion text NOT NULL CHECK (situacion IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde')),
    desde timestamptz(6) NOT NULL,
    hasta timestamptz(6),
    fecha_disponible timestamptz(6),
    motivo text NOT NULL CHECK (octet_length(motivo) BETWEEN 1 AND 1000 AND motivo = btrim(motivo)),
    actor text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    recibo_ref text NOT NULL CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256 AND recibo_ref = btrim(recibo_ref)),
    PRIMARY KEY (participacion_ref, desde),
    UNIQUE (participacion_ref, clave_idempotencia),
    UNIQUE (recibo_ref),
    CHECK (hasta IS NULL OR hasta = desde),
    CHECK ((situacion = 'disponible_desde' AND fecha_disponible IS NOT NULL AND fecha_disponible > desde) OR (situacion <> 'disponible_desde' AND fecha_disponible IS NULL))
);
CREATE INDEX situacion_participacion_vigente ON vec_bolsa_llamamientos.situacion_participacion(participacion_ref, desde DESC);
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY situacion_participacion_solo_propietario ON vec_bolsa_llamamientos.situacion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.situacion_participacion FROM PUBLIC;
CREATE TRIGGER situacion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.situacion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Las participaciones ya constituidas, y las que constituya el comando
-- existente después de esta migración, nacen disponibles sin otro dato.
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT e.participacion_ref,'disponible',c.confirmada_en,NULL,NULL,'Constitución de bolsa','sistema:constitucion',c.confirmada_en,'constitucion:' || e.participacion_ref,'recibo:situacion:constitucion:' || e.participacion_ref
  FROM vec_bolsa_llamamientos.constitucion_entrada e
  JOIN vec_bolsa_llamamientos.constitucion c ON c.instantanea_ref=e.instantanea_ref AND c.version_instantanea=e.version_instantanea;

CREATE FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_desde timestamptz;
BEGIN
 SELECT c.confirmada_en INTO v_desde FROM vec_bolsa_llamamientos.constitucion c WHERE c.instantanea_ref=NEW.instantanea_ref AND c.version_instantanea=NEW.version_instantanea;
 IF v_desde IS NULL THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='constitucion inexistente'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(NEW.participacion_ref,'disponible',v_desde,NULL,NULL,'Constitución de bolsa','sistema:constitucion',v_desde,'constitucion:' || NEW.participacion_ref,'recibo:situacion:constitucion:' || NEW.participacion_ref);
 RETURN NEW;
END $f$;
CREATE TRIGGER constitucion_entrada_inicia_situacion_participacion
AFTER INSERT ON vec_bolsa_llamamientos.constitucion_entrada
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1();

CREATE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE anterior record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_situacion NOT IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde') OR p_desde IS NULL OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_clave_idempotencia<>btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256 OR p_recibo_ref IS NULL OR p_recibo_ref<>btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 OR p_registrada_en IS NULL OR (p_situacion='disponible_desde' AND (p_fecha_disponible IS NULL OR p_fecha_disponible<=p_registrada_en)) OR (p_situacion<>'disponible_desde' AND p_fecha_disponible IS NOT NULL) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='situacion invalida'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p_participacion_ref ORDER BY desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 IF p_desde < anterior.desde THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='desde anterior a la situacion vigente'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia AND (s.situacion<>p_situacion OR s.motivo<>p_motivo OR s.fecha_disponible IS DISTINCT FROM p_fecha_disponible)) THEN RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='clave idempotente reutilizada con otro comando'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia) THEN RETURN QUERY SELECT true, s.recibo_ref, s.situacion, s.desde, s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia; RETURN; END IF;
 IF NOT ((anterior.situacion='disponible' AND p_situacion IN ('no_disponible','pendiente_incorporacion','renuncia','excluido')) OR (anterior.situacion='no_disponible' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='pendiente_incorporacion' AND p_situacion IN ('trabajando','disponible','renuncia','excluido')) OR (anterior.situacion='trabajando' AND p_situacion IN ('disponible','disponible_desde','excluido')) OR (anterior.situacion='disponible_desde' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='renuncia' AND p_situacion IN ('disponible','excluido'))) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref) VALUES(p_participacion_ref,p_situacion,p_desde,NULL,p_fecha_disponible,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref);
 RETURN QUERY SELECT false,p_recibo_ref,p_situacion,p_desde,p_fecha_disponible;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(p_participacion_ref text)
RETURNS TABLE(situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT s.situacion,s.desde,s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref ORDER BY s.desde DESC LIMIT 1
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,timestamptz,timestamptz,text,text,text,text,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,timestamptz,timestamptz,text,text,text,text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(text) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
