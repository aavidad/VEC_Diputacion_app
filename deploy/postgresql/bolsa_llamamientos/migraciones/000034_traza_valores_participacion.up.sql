\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000034', 0));

-- Petición RRHH p.4: la traza de cada cambio debe decir el valor anterior y el
-- nuevo de los campos cambiados. Las historias B2 (situación) y B4 (datos de
-- contacto) ya son de solo adición: cada fila es la postimagen y la anterior es
-- la preimagen. Esta migración no toca esas filas; añade una traza derivada,
-- también de solo adición, escrita en la misma transacción que el cambio.
--
-- Minimización: la situación y la fecha de disponibilidad son códigos y fechas
-- de gestión, sin dato personal, y se guardan tal cual. Los datos de contacto
-- (correo y teléfonos) nunca se guardan en claro ni con huella (la base no ve
-- ninguna huella de ellos, véase 000016): la traza dice qué campo cambió y
-- enlaza la versión cifrada anterior y la nueva ('version:N').
--
-- La IP o el equipo desde el que se opera NO se registran aquí: quedan
-- pendientes de la política de seguridad.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NOT NULL THEN
  RAISE EXCEPTION 'estado incompatible para la traza de valores' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.traza_valor_participacion (
    participacion_ref text NOT NULL,
    recibo_ref text NOT NULL CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256),
    campo text NOT NULL CHECK (campo IN ('situacion','fecha_disponible','datos_contacto','correo','telefono_1','telefono_2')),
    valor_anterior text,
    valor_nuevo text,
    actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 256),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (participacion_ref, recibo_ref, campo),
    CHECK (valor_anterior IS DISTINCT FROM valor_nuevo),
    -- Cada campo solo admite su forma minimizada; un claro de contacto no cabe.
    CHECK (CASE campo
      WHEN 'situacion' THEN coalesce(valor_anterior,'disponible') IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde')
                        AND valor_nuevo IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde')
      WHEN 'fecha_disponible' THEN coalesce(valor_anterior,'2000-01-01T00:00:00.000000Z') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
                        AND coalesce(valor_nuevo,'2000-01-01T00:00:00.000000Z') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
      ELSE coalesce(valor_anterior,'version:1') ~ '^version:[1-9][0-9]{0,18}$' AND valor_nuevo ~ '^version:[1-9][0-9]{0,18}$'
    END)
);
CREATE INDEX traza_valor_participacion_fecha ON vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref, registrada_en DESC, recibo_ref);
ALTER TABLE vec_bolsa_llamamientos.traza_valor_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.traza_valor_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY traza_valor_participacion_solo_propietario ON vec_bolsa_llamamientos.traza_valor_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.traza_valor_participacion FROM PUBLIC;
CREATE TRIGGER traza_valor_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.traza_valor_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.fecha_traza_v1(p timestamptz)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT CASE WHEN p IS NULL THEN NULL ELSE to_char(p AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.fecha_traza_v1(timestamptz) FROM PUBLIC;

-- B2: cada situación nueva deja situación y, si cambia, fecha de disponibilidad
-- anterior y nueva. La primera situación (constitución) no tiene anterior.
CREATE FUNCTION vec_bolsa_llamamientos.trazar_situacion_participacion_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE anterior record;
BEGIN
 SELECT s.situacion, s.fecha_disponible INTO anterior
   FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref = NEW.participacion_ref AND s.desde < NEW.desde
  ORDER BY s.desde DESC LIMIT 1;
 IF anterior.situacion IS DISTINCT FROM NEW.situacion THEN
  INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
  VALUES (NEW.participacion_ref, NEW.recibo_ref, 'situacion', anterior.situacion, NEW.situacion, NEW.actor, NEW.registrada_en);
 END IF;
 IF anterior.fecha_disponible IS DISTINCT FROM NEW.fecha_disponible THEN
  INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
  VALUES (NEW.participacion_ref, NEW.recibo_ref, 'fecha_disponible', vec_bolsa_llamamientos.fecha_traza_v1(anterior.fecha_disponible), vec_bolsa_llamamientos.fecha_traza_v1(NEW.fecha_disponible), NEW.actor, NEW.registrada_en);
 END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.trazar_situacion_participacion_v1() FROM PUBLIC;
CREATE TRIGGER situacion_participacion_traza_valores
AFTER INSERT ON vec_bolsa_llamamientos.situacion_participacion
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.trazar_situacion_participacion_v1();

-- B4: la versión cifrada anterior y la nueva. Qué subcampos cambiaron solo lo
-- sabe la aplicación, que es quien descifra: lo declara en la misma
-- transacción con set_config('vec_bolsa.campos_contacto_cambiados', ..., true)
-- como lista separada por comas de correo, telefono_1 y telefono_2. Sin esa
-- declaración (versiones anteriores a esta migración) queda solo la traza
-- de versión. La declaración se consume al primer uso.
CREATE FUNCTION vec_bolsa_llamamientos.trazar_datos_contacto_participacion_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_declarados text; v_campo text; v_anterior text;
BEGIN
 v_anterior := CASE WHEN NEW.version > 1 THEN 'version:' || (NEW.version - 1)::text END;
 INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
 VALUES (NEW.participacion_ref, NEW.recibo_ref, 'datos_contacto', v_anterior, 'version:' || NEW.version::text, NEW.actor, NEW.registrada_en);
 v_declarados := nullif(current_setting('vec_bolsa.campos_contacto_cambiados', true), '');
 IF v_declarados IS NOT NULL THEN
  PERFORM set_config('vec_bolsa.campos_contacto_cambiados', '', true);
  IF v_declarados !~ '^(correo|telefono_1|telefono_2)(,(correo|telefono_1|telefono_2)){0,2}$' THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='campos de contacto cambiados invalidos';
  END IF;
  FOREACH v_campo IN ARRAY (SELECT array_agg(DISTINCT c ORDER BY c) FROM unnest(string_to_array(v_declarados, ',')) c) LOOP
   INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
   VALUES (NEW.participacion_ref, NEW.recibo_ref, v_campo, v_anterior, 'version:' || NEW.version::text, NEW.actor, NEW.registrada_en);
  END LOOP;
 END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.trazar_datos_contacto_participacion_v1() FROM PUBLIC;
CREATE TRIGGER datos_contacto_participacion_traza_valores
AFTER INSERT ON vec_bolsa_llamamientos.datos_contacto_participacion
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.trazar_datos_contacto_participacion_v1();

-- La historia ya registrada se traza igual, sin reescribirla.
INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
SELECT participacion_ref, recibo_ref, 'situacion', anterior, situacion, actor, registrada_en
  FROM (SELECT s.*, lag(s.situacion) OVER (PARTITION BY s.participacion_ref ORDER BY s.desde) AS anterior
          FROM vec_bolsa_llamamientos.situacion_participacion s) h
 WHERE anterior IS DISTINCT FROM situacion;
INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
SELECT participacion_ref, recibo_ref, 'fecha_disponible', vec_bolsa_llamamientos.fecha_traza_v1(anterior), vec_bolsa_llamamientos.fecha_traza_v1(fecha_disponible), actor, registrada_en
  FROM (SELECT s.*, lag(s.fecha_disponible) OVER (PARTITION BY s.participacion_ref ORDER BY s.desde) AS anterior
          FROM vec_bolsa_llamamientos.situacion_participacion s) h
 WHERE anterior IS DISTINCT FROM fecha_disponible;
INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref,recibo_ref,campo,valor_anterior,valor_nuevo,actor,registrada_en)
SELECT d.participacion_ref, d.recibo_ref, 'datos_contacto', CASE WHEN d.version > 1 THEN 'version:' || (d.version - 1)::text END, 'version:' || d.version::text, d.actor, d.registrada_en
  FROM vec_bolsa_llamamientos.datos_contacto_participacion d;

-- Lectura para la ficha de RRHH: mismo permiso y mismas comprobaciones que el
-- historial de operaciones B8 (000019); una sola autorización consumida
-- devuelve las operaciones y la traza de valores. No abre otro consumidor.
CREATE FUNCTION vec_bolsa_llamamientos.listar_historial_participacion_v1(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(clase text, instante timestamptz, operacion text, situacion text, justificante_tipo text, justificante_ref text, justificante_sha256 text, actor text, validador text, validada_en timestamptz, motivo text, recibo_ref text, campo text, valor_anterior text, valor_nuevo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de historial no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de historial no autorizada'; END;
 IF v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de historial no autorizada';
 END IF;
 RETURN QUERY
  SELECT 'operacion'::text, o.desde, o.operacion, s.situacion, o.justificante_tipo, o.justificante_ref, o.justificante_sha256, o.actor, o.validador, o.validada_en, s.motivo, s.recibo_ref, NULL::text, NULL::text, NULL::text
    FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
    JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
   WHERE o.participacion_ref = p_participacion_ref
  UNION ALL
  SELECT 'cambio'::text, t.registrada_en, NULL::text, NULL::text, NULL::text, NULL::text, NULL::text, t.actor, NULL::text, NULL::timestamptz, NULL::text, t.recibo_ref, t.campo, t.valor_anterior, t.valor_nuevo
    FROM vec_bolsa_llamamientos.traza_valor_participacion t
   WHERE t.participacion_ref = p_participacion_ref
  ORDER BY 2 DESC, 1, 13;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_historial_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_historial_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
