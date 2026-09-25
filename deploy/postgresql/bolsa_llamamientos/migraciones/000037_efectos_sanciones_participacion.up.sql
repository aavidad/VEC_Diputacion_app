\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000037', 0));

-- Duda 62 de RRHH: efectos de las sanciones de 000026 que quedaron sin hacer.
-- Qué efecto tiene cada consecuencia lo decide el catálogo de reglas; aquí
-- solo se ejecuta y se conserva, con historia de solo adición:
--  1. «Pasar al final»: penalización del orden vigente ligada a la sanción y
--     a la versión exacta de la regla. El cálculo del orden (000018) la
--     aplica: la persona penalizada se ordena tras las no penalizadas de su
--     bolsa y conserva su orden relativo. Se sustituye solo el cuerpo exacto
--     instalado por 000018 de leer_orden_vigente_bolsa_v1, que siguen usando
--     la reserva B7, los avisos y las lecturas; su firma y ACL no cambian.
--  2. Suspensión con fin: la sanción deja la situación B2 existente
--     «disponible_desde» con la fecha de fin calculada, de modo que la
--     persona vuelve al turno sola al llegar la fecha. El cambio pasa por la
--     política de transiciones vigente de 000032 («disponible>disponible_desde»,
--     incluida en su versión inicial) y guarda su versión. La operación B8
--     queda registrada como «pausar» (salir del turno, justificada por la
--     resolución y validada por quien resuelve), igual que en 000026; en B8
--     «pausar» sin fecha deja «no_disponible» y con fecha de fin deja
--     «disponible_desde»: el histórico B8 muestra la situación resultante.
--  3. Recurso estimado: al anotar un estado del recurso que el catálogo
--     declara revocatorio, en la misma transacción y bajo la misma
--     autorización se revierte la sanción: una baja o una suspensión vigente
--     devuelven la situación anterior (readmisión) y la penalización del
--     orden deja de aplicarse. La readmisión la hace
--     readmitir_participacion_por_recurso_v1 de 000032, única excepción a
--     «nunca se sale de excluido»; para una suspensión exige además la
--     política vigente. Deja en B8 una operación «reactivar» justificada por
--     la resolución que estima el recurso y validada por quien la resuelve.
--     No existe como operación libre: exige un recurso registrado de esa
--     sanción, que sea su último estado, y una segunda persona que lo
--     resuelve.
-- Depende de 000032 (política de transiciones y readmisión).
-- Sin catálogo, o con consecuencias que no declaran estos efectos, la
-- conducta es la de 000026: las funciones _v1 siguen instaladas.
DO $precondicion$
DECLARE v_fuente text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.recurso_sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_sancion_participacion_v1(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(text,text,text,date,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.penalizacion_orden_sancion') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.reversion_sancion_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para los efectos de las sanciones' USING ERRCODE='55000';
 END IF;
 SELECT p.prosrc INTO v_fuente FROM pg_catalog.pg_proc p
  WHERE p.oid = to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)');
 IF v_fuente IS NULL OR md5(v_fuente) <> '99bc67526e88fef0febade892b3846d2' THEN
  RAISE EXCEPTION 'leer_orden_vigente_bolsa_v1 no es la de 000018' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.penalizacion_orden_sancion (
    sancion_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.sancion_participacion(sancion_ref),
    bolsa_ref text NOT NULL CHECK (octet_length(bolsa_ref) BETWEEN 1 AND 512),
    participacion_ref text NOT NULL CHECK (octet_length(participacion_ref) BETWEEN 1 AND 512),
    posicion text NOT NULL CHECK (posicion = 'final'),
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 300),
    regla_huella_sha256 text NOT NULL CHECK (regla_huella_sha256 ~ '^[a-f0-9]{64}$'),
    aplicada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL
);
COMMENT ON TABLE vec_bolsa_llamamientos.penalizacion_orden_sancion IS
    'Penalizaciones del orden vigente impuestas por una sanción, con la versión exacta de la regla. Solo adición.';
CREATE INDEX penalizacion_orden_sancion_lectura
    ON vec_bolsa_llamamientos.penalizacion_orden_sancion(bolsa_ref, participacion_ref, aplicada_en);
ALTER TABLE vec_bolsa_llamamientos.penalizacion_orden_sancion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.penalizacion_orden_sancion FORCE ROW LEVEL SECURITY;
CREATE POLICY penalizacion_orden_sancion_solo_propietario ON vec_bolsa_llamamientos.penalizacion_orden_sancion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.penalizacion_orden_sancion FROM PUBLIC;
CREATE TRIGGER penalizacion_orden_sancion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.penalizacion_orden_sancion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE TABLE vec_bolsa_llamamientos.reversion_sancion_participacion (
    sancion_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.sancion_participacion(sancion_ref),
    participacion_ref text NOT NULL CHECK (octet_length(participacion_ref) BETWEEN 1 AND 512),
    recurso_clave_idempotencia text NOT NULL,
    estado_recurso text NOT NULL CHECK (estado_recurso ~ '^[a-z][a-z0-9_]{0,63}$'),
    regla_ref text NOT NULL CHECK (octet_length(regla_ref) BETWEEN 1 AND 300),
    regla_huella_sha256 text NOT NULL CHECK (regla_huella_sha256 ~ '^[a-f0-9]{64}$'),
    efecto_revertido text NOT NULL CHECK (efecto_revertido IN ('ninguna','pausar','excluir')),
    situacion_restaurada text CHECK (situacion_restaurada IS NULL OR situacion_restaurada IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','disponible_desde')),
    situacion_desde timestamptz(6),
    resuelta_por text NOT NULL CHECK (octet_length(resuelta_por) BETWEEN 1 AND 256 AND resuelta_por = btrim(resuelta_por)),
    actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 256 AND actor = btrim(actor)),
    recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:readmision:[a-f0-9]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (sancion_ref, recurso_clave_idempotencia) REFERENCES vec_bolsa_llamamientos.recurso_sancion_participacion(sancion_ref, clave_idempotencia),
    FOREIGN KEY (participacion_ref, situacion_desde) REFERENCES vec_bolsa_llamamientos.situacion_participacion(participacion_ref, desde),
    CHECK ((situacion_restaurada IS NULL) = (situacion_desde IS NULL)),
    CHECK (efecto_revertido <> 'excluir' OR situacion_restaurada IS NOT NULL),
    CHECK (efecto_revertido <> 'ninguna' OR situacion_restaurada IS NULL),
    -- Como la baja (duda 6), la readmisión la resuelve otra persona.
    CHECK (resuelta_por <> actor)
);
COMMENT ON TABLE vec_bolsa_llamamientos.reversion_sancion_participacion IS
    'Reversión de una sanción por un recurso de reposición en estado revocatorio del catálogo: readmisión o fin de la suspensión y de la penalización. Solo adición.';
ALTER TABLE vec_bolsa_llamamientos.reversion_sancion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.reversion_sancion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY reversion_sancion_participacion_solo_propietario ON vec_bolsa_llamamientos.reversion_sancion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.reversion_sancion_participacion FROM PUBLIC;
CREATE TRIGGER reversion_sancion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.reversion_sancion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Suspensión con fin automático. Mismas validaciones, autorización,
-- idempotencia y operación B8 que 000026; la situación resultante es
-- «disponible_desde» con el primer día tras la suspensión (hora peninsular).
-- Como «pausar» en B2, solo se aplica a una participación disponible.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_suspension_con_fin_v1(
 p_bolsa_ref text, p_participacion_ref text, p_sancion_ref text, p_consecuencia text, p_consecuencia_etiqueta text,
 p_causa text, p_fecha_notificacion date, p_resolucion_ref text, p_resolucion_sha256 text,
 p_resuelta_por text, p_regla_ref text, p_regla_huella text, p_suspension_hasta date, p_recurso_vence date,
 p_recurso_regla_ref text, p_recurso_regla_huella text, p_desde timestamptz, p_actor text, p_clave_idempotencia text,
 p_recibo_ref text, p_registrada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, sancion_ref text, recibo_ref text, situacion text, desde timestamptz)
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v_previa record; v_anterior record; v_situacion record; v_fin timestamptz; v_politica record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='sancion no autorizada'; END IF;
 IF p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_clave_idempotencia IS NULL OR p_actor IS NULL OR p_registrada_en IS NULL
    OR p_sancion_ref IS DISTINCT FROM 'sancion:' || encode(sha256(convert_to(p_participacion_ref || chr(31) || p_clave_idempotencia, 'UTF8')), 'hex')
    OR p_fecha_notificacion IS NULL OR p_fecha_notificacion > (p_registrada_en AT TIME ZONE 'Europe/Madrid')::date
    OR p_recurso_vence IS NULL OR p_recurso_vence <= p_fecha_notificacion
    OR p_suspension_hasta IS NULL OR p_suspension_hasta <= p_fecha_notificacion
    OR p_desde IS NULL OR p_causa IS NULL OR p_causa <> btrim(p_causa) OR octet_length(p_causa) NOT BETWEEN 1 AND 1000
    OR p_resolucion_ref IS NULL OR p_resolucion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$' OR octet_length(p_resolucion_ref) NOT BETWEEN 1 AND 256
    OR p_resolucion_sha256 IS NULL OR p_resolucion_sha256 !~ '^[a-f0-9]{64}$'
    OR p_resuelta_por IS NULL OR p_resuelta_por <> btrim(p_resuelta_por) OR octet_length(p_resuelta_por) NOT BETWEEN 1 AND 256
    OR p_recibo_ref IS DISTINCT FROM 'recibo:situacion:' || substr(p_sancion_ref, 9) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='sancion invalida';
 END IF;
 v_fin := ((p_suspension_hasta + 1)::timestamp AT TIME ZONE 'Europe/Madrid');
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
                  JOIN vec_bolsa_llamamientos.constitucion c USING (instantanea_ref, version_instantanea)
                 WHERE e.participacion_ref = p_participacion_ref AND c.bolsa_ref = p_bolsa_ref) THEN
  RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion ajena a la bolsa';
 END IF;
 -- Como B2, se consume la autorización antes de resolver el replay.
 PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(p_participacion_ref, p_actor, p_capacidad, p_decision,
   p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 SELECT s.* INTO v_previa FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.participacion_ref = p_participacion_ref AND s.clave_idempotencia = p_clave_idempotencia;
 SELECT sp.* INTO v_situacion FROM vec_bolsa_llamamientos.situacion_participacion sp
  WHERE sp.participacion_ref = p_participacion_ref AND sp.clave_idempotencia = p_clave_idempotencia;
 IF v_previa.sancion_ref IS NOT NULL THEN
  IF v_previa.bolsa_ref <> p_bolsa_ref OR v_previa.consecuencia <> p_consecuencia OR v_previa.efecto <> 'pausar'
     OR v_previa.causa <> p_causa OR v_previa.fecha_notificacion <> p_fecha_notificacion
     OR v_previa.resolucion_ref <> p_resolucion_ref OR v_previa.resolucion_sha256 <> p_resolucion_sha256
     OR v_previa.resuelta_por <> p_resuelta_por OR v_previa.actor <> p_actor
     OR v_previa.suspension_hasta IS DISTINCT FROM p_suspension_hasta
     OR v_situacion.situacion IS DISTINCT FROM 'disponible_desde' OR v_situacion.fecha_disponible IS DISTINCT FROM v_fin THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra sancion';
  END IF;
  RETURN QUERY SELECT true, v_previa.sancion_ref, v_previa.recibo_ref, v_situacion.situacion, v_situacion.desde;
  RETURN;
 END IF;
 IF v_situacion.participacion_ref IS NOT NULL THEN
  RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra operacion';
 END IF;
 IF v_fin <= p_registrada_en OR v_fin <= p_desde THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='suspension ya cumplida';
 END IF;
 SELECT sp.* INTO v_anterior FROM vec_bolsa_llamamientos.situacion_participacion sp
  WHERE sp.participacion_ref = p_participacion_ref ORDER BY sp.desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 IF v_anterior.situacion <> 'disponible' OR p_desde < v_anterior.desde THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida';
 END IF;
 -- Como B2 (000032): la política vigente debe admitir el cambio y la fila
 -- guarda su versión. Cerrojo compartido frente a la publicación.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
 SELECT p.version, p.transiciones INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
 IF NOT ((v_anterior.situacion || '>disponible_desde') = ANY (v_politica.transiciones)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref, situacion, desde, hasta, fecha_disponible, motivo, actor, registrada_en, clave_idempotencia, recibo_ref, politica_transiciones_version)
 VALUES (p_participacion_ref, 'disponible_desde', p_desde, NULL, v_fin, p_causa, p_actor, p_registrada_en, p_clave_idempotencia, p_recibo_ref, v_politica.version);
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256, actor, validador, validada_en, registrada_en, clave_idempotencia)
 VALUES (p_participacion_ref, p_desde, 'pausar', 'resolucion', p_resolucion_ref, p_resolucion_sha256, p_actor, p_resuelta_por, p_registrada_en, p_registrada_en, p_clave_idempotencia);
 INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
   sancion_ref, participacion_ref, bolsa_ref, consecuencia, consecuencia_etiqueta, efecto, causa, fecha_notificacion,
   resolucion_ref, resolucion_sha256, resuelta_por, regla_ref, regla_huella_sha256, suspension_hasta, recurso_vence,
   recurso_regla_ref, recurso_regla_huella_sha256, situacion_desde, recibo_ref, actor, registrada_en, clave_idempotencia)
 VALUES (p_sancion_ref, p_participacion_ref, p_bolsa_ref, p_consecuencia, p_consecuencia_etiqueta, 'pausar', p_causa,
   p_fecha_notificacion, p_resolucion_ref, p_resolucion_sha256, p_resuelta_por, p_regla_ref, p_regla_huella,
   p_suspension_hasta, p_recurso_vence, p_recurso_regla_ref, p_recurso_regla_huella, p_desde, p_recibo_ref,
   p_actor, p_registrada_en, p_clave_idempotencia);
 RETURN QUERY SELECT false, p_sancion_ref, p_recibo_ref, 'disponible_desde'::text, p_desde;
END $f$;

-- Registro de una sanción con los efectos que el catálogo declara:
-- p_orden_final (penalización «al final») y p_fin_automatico (suspensión
-- que termina sola). Con ambos a falso equivale exactamente a la _v1.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v2(
 p_bolsa_ref text, p_participacion_ref text, p_sancion_ref text, p_consecuencia text, p_consecuencia_etiqueta text,
 p_efecto text, p_causa text, p_fecha_notificacion date, p_resolucion_ref text, p_resolucion_sha256 text,
 p_resuelta_por text, p_regla_ref text, p_regla_huella text, p_suspension_hasta date, p_recurso_vence date,
 p_recurso_regla_ref text, p_recurso_regla_huella text, p_desde timestamptz, p_actor text, p_clave_idempotencia text,
 p_recibo_ref text, p_registrada_en timestamptz, p_orden_final boolean, p_fin_automatico boolean,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, sancion_ref text, recibo_ref text, situacion text, desde timestamptz, orden_final boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_r record; v_pen record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='sancion no autorizada'; END IF;
 IF p_orden_final IS NULL OR p_fin_automatico IS NULL
    OR (p_orden_final AND p_efecto IS DISTINCT FROM 'ninguna' AND p_efecto IS DISTINCT FROM 'pausar')
    OR (p_fin_automatico AND (p_efecto IS DISTINCT FROM 'pausar' OR p_suspension_hasta IS NULL)) THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='sancion invalida';
 END IF;
 IF p_fin_automatico THEN
  SELECT * INTO STRICT v_r FROM vec_bolsa_llamamientos.registrar_suspension_con_fin_v1(
    p_bolsa_ref, p_participacion_ref, p_sancion_ref, p_consecuencia, p_consecuencia_etiqueta, p_causa, p_fecha_notificacion,
    p_resolucion_ref, p_resolucion_sha256, p_resuelta_por, p_regla_ref, p_regla_huella, p_suspension_hasta, p_recurso_vence,
    p_recurso_regla_ref, p_recurso_regla_huella, p_desde, p_actor, p_clave_idempotencia, p_recibo_ref, p_registrada_en,
    p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 ELSE
  SELECT * INTO STRICT v_r FROM vec_bolsa_llamamientos.registrar_sancion_participacion_v1(
    p_bolsa_ref, p_participacion_ref, p_sancion_ref, p_consecuencia, p_consecuencia_etiqueta, p_efecto, p_causa, p_fecha_notificacion,
    p_resolucion_ref, p_resolucion_sha256, p_resuelta_por, p_regla_ref, p_regla_huella, p_suspension_hasta, p_recurso_vence,
    p_recurso_regla_ref, p_recurso_regla_huella, p_desde, p_actor, p_clave_idempotencia, p_recibo_ref, p_registrada_en,
    p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 END IF;
 SELECT pe.* INTO v_pen FROM vec_bolsa_llamamientos.penalizacion_orden_sancion pe WHERE pe.sancion_ref = v_r.sancion_ref;
 IF v_r.reutilizada THEN
  -- El replay debe pedir el mismo efecto sobre el orden.
  IF (v_pen.sancion_ref IS NOT NULL) <> p_orden_final
     OR (p_orden_final AND (v_pen.regla_ref <> p_regla_ref OR v_pen.regla_huella_sha256 <> p_regla_huella)) THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra sancion';
  END IF;
 ELSIF p_orden_final THEN
  INSERT INTO vec_bolsa_llamamientos.penalizacion_orden_sancion(sancion_ref, bolsa_ref, participacion_ref, posicion, regla_ref, regla_huella_sha256, aplicada_en, registrada_en)
  VALUES (v_r.sancion_ref, p_bolsa_ref, p_participacion_ref, 'final', p_regla_ref, p_regla_huella, p_registrada_en, p_registrada_en);
 END IF;
 RETURN QUERY SELECT v_r.reutilizada, v_r.sancion_ref, v_r.recibo_ref, v_r.situacion, v_r.desde, p_orden_final;
END $f$;

-- Anota un estado del recurso (como la _v1) y, si el catálogo lo declara
-- revocatorio (p_revierte), revierte la sanción en la misma transacción.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(
 p_participacion_ref text, p_sancion_ref text, p_estado text, p_fecha date, p_documento_ref text, p_documento_sha256 text,
 p_actor text, p_clave_idempotencia text, p_registrada_en timestamptz,
 p_revierte boolean, p_resuelta_por text, p_regla_ref text, p_regla_huella text, p_motivo text, p_recibo_ref text,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, sancion_ref text, estado text, registrada_en timestamptz, recibo_ref text, situacion text, desde timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_r record; v_sancion record; v_rev record; v_ultimo record; v_actual record; v_readmision record;
 v_situacion text; v_desde timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='recurso no autorizado'; END IF;
 IF p_revierte IS NULL THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso invalido'; END IF;
 IF NOT p_revierte THEN
  IF p_resuelta_por IS NOT NULL OR p_regla_ref IS NOT NULL OR p_regla_huella IS NOT NULL OR p_motivo IS NOT NULL OR p_recibo_ref IS NOT NULL THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso invalido';
  END IF;
  SELECT * INTO STRICT v_r FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(
    p_participacion_ref, p_sancion_ref, p_estado, p_fecha, p_documento_ref, p_documento_sha256, p_actor, p_clave_idempotencia, p_registrada_en,
    p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
  IF v_r.reutilizada AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                  WHERE rv.sancion_ref = p_sancion_ref AND rv.recurso_clave_idempotencia = p_clave_idempotencia) THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro recurso';
  END IF;
  RETURN QUERY SELECT v_r.reutilizada, v_r.sancion_ref, v_r.estado, v_r.registrada_en, NULL::text, NULL::text, NULL::timestamptz;
  RETURN;
 END IF;
 IF p_documento_ref IS NULL OR p_documento_sha256 IS NULL OR p_resuelta_por IS NULL OR p_resuelta_por <> btrim(p_resuelta_por) OR octet_length(p_resuelta_por) NOT BETWEEN 1 AND 256
    OR p_resuelta_por = p_actor OR p_regla_ref IS NULL OR octet_length(p_regla_ref) NOT BETWEEN 1 AND 300
    OR p_regla_huella IS NULL OR p_regla_huella !~ '^[a-f0-9]{64}$'
    OR p_motivo IS NULL OR p_motivo <> btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000
    OR p_sancion_ref IS NULL OR p_clave_idempotencia IS NULL
    OR p_recibo_ref IS DISTINCT FROM 'recibo:readmision:' || encode(sha256(convert_to(p_sancion_ref || chr(31) || p_clave_idempotencia, 'UTF8')), 'hex') THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso invalido';
 END IF;
 -- La _v1 consume la autorización, valida la sanción y anota el estado.
 SELECT * INTO STRICT v_r FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(
   p_participacion_ref, p_sancion_ref, p_estado, p_fecha, p_documento_ref, p_documento_sha256, p_actor, p_clave_idempotencia, p_registrada_en,
   p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 SELECT s.* INTO STRICT v_sancion FROM vec_bolsa_llamamientos.sancion_participacion s
  WHERE s.sancion_ref = p_sancion_ref AND s.participacion_ref = p_participacion_ref FOR UPDATE;
 SELECT rv.* INTO v_rev FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv WHERE rv.sancion_ref = p_sancion_ref;
 IF v_rev.sancion_ref IS NOT NULL THEN
  IF v_rev.recurso_clave_idempotencia <> p_clave_idempotencia OR NOT v_r.reutilizada
     OR v_rev.resuelta_por <> p_resuelta_por OR v_rev.regla_ref <> p_regla_ref OR v_rev.estado_recurso <> p_estado THEN
   -- Una sanción se revierte una sola vez.
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='sancion ya revertida';
  END IF;
  RETURN QUERY SELECT true, v_rev.sancion_ref, v_rev.estado_recurso, v_r.registrada_en, v_rev.recibo_ref, v_rev.situacion_restaurada, v_rev.situacion_desde;
  RETURN;
 END IF;
 IF v_r.reutilizada THEN
  -- La clave ya anotó un estado sin reversión: no se adopta.
  RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro recurso';
 END IF;
 SELECT r.* INTO STRICT v_ultimo FROM vec_bolsa_llamamientos.recurso_sancion_participacion r
  WHERE r.sancion_ref = p_sancion_ref ORDER BY r.registrada_en DESC, r.clave_idempotencia DESC LIMIT 1;
 IF v_ultimo.clave_idempotencia <> p_clave_idempotencia THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='recurso no vigente';
 END IF;
 IF v_sancion.situacion_desde IS NOT NULL THEN
  SELECT sp.* INTO STRICT v_actual FROM vec_bolsa_llamamientos.situacion_participacion sp
   WHERE sp.participacion_ref = p_participacion_ref ORDER BY sp.desde DESC LIMIT 1 FOR UPDATE;
  IF v_actual.desde = v_sancion.situacion_desde THEN
   -- El efecto sigue vigente: se vuelve a la situación anterior a la sanción
   -- por la readmisión de 000032, y B8 lo anota como «reactivar».
   SELECT * INTO STRICT v_readmision FROM vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(
     p_participacion_ref, p_sancion_ref, p_clave_idempotencia, p_motivo, p_actor,
     'readmision:' || substr(p_recibo_ref, 19), p_recibo_ref, p_registrada_en);
   v_situacion := v_readmision.situacion; v_desde := v_readmision.desde;
   INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256, actor, validador, validada_en, registrada_en, clave_idempotencia)
   VALUES (p_participacion_ref, v_desde, 'reactivar', 'resolucion', p_documento_ref, p_documento_sha256, p_actor, p_resuelta_por, p_registrada_en, p_registrada_en, 'readmision:' || substr(p_recibo_ref, 19));
  ELSIF v_sancion.efecto = 'excluir' THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='readmision invalida';
  END IF;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.reversion_sancion_participacion(sancion_ref, participacion_ref, recurso_clave_idempotencia, estado_recurso,
   regla_ref, regla_huella_sha256, efecto_revertido, situacion_restaurada, situacion_desde, resuelta_por, actor, recibo_ref, registrada_en)
 VALUES (p_sancion_ref, p_participacion_ref, p_clave_idempotencia, p_estado, p_regla_ref, p_regla_huella, v_sancion.efecto,
   v_situacion, v_desde, p_resuelta_por, p_actor, p_recibo_ref, p_registrada_en);
 RETURN QUERY SELECT false, p_sancion_ref, p_estado, p_registrada_en, p_recibo_ref, v_situacion, v_desde;
END $f$;

-- Histórico con el efecto aplicado y la reversión, si la hay.
CREATE FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v2(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(sancion_ref text, consecuencia text, consecuencia_etiqueta text, efecto text, causa text, fecha_notificacion date,
 resolucion_ref text, resolucion_sha256 text, resuelta_por text, regla_ref text, regla_huella_sha256 text,
 suspension_hasta date, recurso_vence date, recurso_regla_ref text, recurso_regla_huella_sha256 text,
 situacion_desde timestamptz, actor text, registrada_en timestamptz, recursos jsonb,
 situacion_aplicada text, fecha_disponible timestamptz, orden_final boolean, reversion jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de sanciones no autorizada';
 END IF;
 PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1(p_participacion_ref, p_actor, p_capacidad, p_decision,
   p_motivo_autorizacion, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 RETURN QUERY SELECT s.sancion_ref, s.consecuencia, s.consecuencia_etiqueta, s.efecto, s.causa, s.fecha_notificacion,
   s.resolucion_ref, s.resolucion_sha256, s.resuelta_por, s.regla_ref, s.regla_huella_sha256, s.suspension_hasta,
   s.recurso_vence, s.recurso_regla_ref, s.recurso_regla_huella_sha256, s.situacion_desde, s.actor, s.registrada_en,
   coalesce((SELECT jsonb_agg(jsonb_build_object('estado', r.estado, 'fecha', r.fecha, 'documento_ref', r.documento_ref,
       'documento_sha256', r.documento_sha256, 'actor', r.actor, 'registrada_en', r.registrada_en) ORDER BY r.registrada_en, r.clave_idempotencia)
     FROM vec_bolsa_llamamientos.recurso_sancion_participacion r WHERE r.sancion_ref = s.sancion_ref), '[]'::jsonb),
   sp.situacion, sp.fecha_disponible, pe.sancion_ref IS NOT NULL,
   CASE WHEN rv.sancion_ref IS NULL THEN NULL ELSE jsonb_build_object('estado_recurso', rv.estado_recurso,
     'regla_ref', rv.regla_ref, 'efecto_revertido', rv.efecto_revertido, 'situacion_restaurada', rv.situacion_restaurada,
     'situacion_desde', rv.situacion_desde, 'resuelta_por', rv.resuelta_por, 'actor', rv.actor,
     'recibo_ref', rv.recibo_ref, 'registrada_en', rv.registrada_en) END
   FROM vec_bolsa_llamamientos.sancion_participacion s
   LEFT JOIN vec_bolsa_llamamientos.situacion_participacion sp ON sp.participacion_ref = s.participacion_ref AND sp.desde = s.situacion_desde
   LEFT JOIN vec_bolsa_llamamientos.penalizacion_orden_sancion pe ON pe.sancion_ref = s.sancion_ref
   LEFT JOIN vec_bolsa_llamamientos.reversion_sancion_participacion rv ON rv.sancion_ref = s.sancion_ref
  WHERE s.participacion_ref = p_participacion_ref
  ORDER BY s.registrada_en DESC, s.sancion_ref
  LIMIT 200;
END $f$;

-- Orden vigente de 000018 con la penalización «al final». Diferencias: la
-- columna «penalizada» (penalización aplicada en p_en y no revertida)
-- encabeza la ordenación y da la razón «sancion_al_final»; quien sube porque
-- alguien con mejor orden de acta está penalizado recibe la razón
-- «adelanta_por_sancion» en lugar de «pausa».
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa_ref text,p_en timestamptz)
RETURNS TABLE(politica_ref text,version_politica bigint,criterio text,tipo_lista text,reposicion text,provisional boolean,rotulo text,actor text,vigente_desde timestamptz,participacion_ref text,orden_acta bigint,orden_vigente bigint,situacion text,razon text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 WITH politica AS (
  SELECT p.* FROM vec_bolsa_llamamientos.politica_orden_bolsa p
   WHERE p.bolsa_ref=p_bolsa_ref AND p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
   ORDER BY p.version DESC LIMIT 1
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         (s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)) AS ocupa_turno,
         r.aplicada_en AS repuesta_en,
         EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion pe
                  WHERE pe.bolsa_ref=c.bolsa_ref AND pe.participacion_ref=e.participacion_ref AND pe.aplicada_en<=p_en
                    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                     WHERE rv.sancion_ref=pe.sancion_ref AND rv.registrada_en<=p_en)) AS penalizada
    FROM vec_bolsa_llamamientos.constitucion c
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en FROM vec_bolsa_llamamientos.reposicion_orden_bolsa ro WHERE ro.bolsa_ref=c.bolsa_ref AND ro.participacion_ref=e.participacion_ref AND ro.aplicada_en<=p_en ORDER BY ro.aplicada_en DESC LIMIT 1) r ON true
   WHERE c.bolsa_ref=p_bolsa_ref
 ), elegibles AS (
  SELECT b.participacion_ref,row_number() OVER(ORDER BY
    CASE WHEN b.penalizada THEN 1 ELSE 0 END,
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
             WHEN b.penalizada THEN 'sancion_al_final'
             WHEN b.repuesta_en IS NOT NULL AND e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'reposicion_tras_contrato'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta
                  AND EXISTS (SELECT 1 FROM base o WHERE o.penalizada AND o.ocupa_turno AND o.orden_acta < b.orden_acta) THEN 'adelanta_por_sancion'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'pausa'
             ELSE 'orden_acta' END
   FROM base b CROSS JOIN politica p LEFT JOIN elegibles e USING(participacion_ref)
  ORDER BY e.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref
 $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_suspension_con_fin_v1(text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v2(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,boolean,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(text,text,text,date,text,text,text,text,timestamptz,boolean,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_sancion_participacion_v2(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,boolean,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(text,text,text,date,text,text,text,text,timestamptz,boolean,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_sanciones_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
