\set ON_ERROR_STOP on
-- Cronos C3: programación diaria versionada y libro de movimientos DERIVADO.
-- La publicación de una programación requiere una autoridad de gobierno aún
-- no compuesta; esta migración no concede DML ni lectura al runtime.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000004',0));
CREATE TABLE vec_cronos_v1.programacion_jornada (
    programacion_ref text PRIMARY KEY CHECK (programacion_ref ~ '^programacion:cronos:[-A-Za-z0-9_]{1,128}$'),
    empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
    fecha date NOT NULL,
    version integer NOT NULL CHECK (version BETWEEN 1 AND 2147483647),
    turno_ref text NOT NULL CHECK (turno_ref ~ '^[-A-Za-z0-9_.:]{1,128}$'),
    politica_version_ref text NOT NULL CHECK (politica_version_ref ~ '^[-A-Za-z0-9_.:]{1,128}$'),
    fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
    zona_horaria text NOT NULL CHECK (zona_horaria IN ('Europe/Madrid','Atlantic/Canary')),
    minutos_previstos integer NOT NULL CHECK (minutos_previstos BETWEEN 0 AND 1440),
    publicada_en timestamptz(6) NOT NULL,
    UNIQUE (empleado_ref,fecha,version)
);
CREATE INDEX programacion_jornada_persona_fecha_idx ON vec_cronos_v1.programacion_jornada(empleado_ref,fecha,version DESC);
ALTER TABLE vec_cronos_v1.programacion_jornada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.programacion_jornada FORCE ROW LEVEL SECURITY;
CREATE POLICY programacion_lectura_propia ON vec_cronos_v1.programacion_jornada
 FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY programacion_adicion_propia ON vec_cronos_v1.programacion_jornada
 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.programacion_jornada
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON vec_cronos_v1.programacion_jornada FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;

-- El tipo visual del origen procede de una clasificación exacta y gobernada
-- del canal acreditado. Un código opaco desconocido se presenta como NULL.
-- La política cambia de referencia al cambiar su clasificación; las filas
-- históricas conservan la referencia original y no se reetiquetan.
CREATE TABLE vec_cronos_v1.clasificacion_canal (
  politica_version_ref text NOT NULL CHECK (politica_version_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
  canal_ref text NOT NULL CHECK (canal_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
  origen_ref text NOT NULL CHECK (origen_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
  calidad_ref text NOT NULL CHECK (calidad_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
  tipo_origen text NOT NULL CHECK (tipo_origen IN ('terminal','remoto')),
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL,
  PRIMARY KEY (politica_version_ref,canal_ref,origen_ref,calidad_ref)
);
CREATE TRIGGER clasificacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.clasificacion_canal
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON vec_cronos_v1.clasificacion_canal FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
-- Se inmoviliza el tipo en el hecho al INSERT. Una clasificación publicada
-- después no puede cambiar el significado histórico de ese marcaje.
ALTER TABLE vec_cronos_v1.marcaje_original ADD COLUMN tipo_origen text
 CHECK (tipo_origen IN ('terminal','remoto'));
CREATE FUNCTION vec_cronos_v1.asentar_tipo_origen_marcaje_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 SELECT c.tipo_origen INTO NEW.tipo_origen FROM vec_cronos_v1.clasificacion_canal c
 WHERE c.politica_version_ref=NEW.material::jsonb #>> '{canal,politica_version_ref}'
   AND c.canal_ref=NEW.material::jsonb #>> '{canal,canal_ref}'
   AND c.origen_ref=NEW.material::jsonb #>> '{canal,origen_ref}'
   AND c.calidad_ref=NEW.material::jsonb #>> '{canal,calidad_ref}';
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.asentar_tipo_origen_marcaje_v1() FROM PUBLIC;
CREATE TRIGGER asentar_tipo_origen BEFORE INSERT ON vec_cronos_v1.marcaje_original
 FOR EACH ROW EXECUTE FUNCTION vec_cronos_v1.asentar_tipo_origen_marcaje_v1();

-- No persiste un número mutable. Cada fila virtual cita el hecho y versión
-- que explica el movimiento; el trabajo se conserva en microsegundos.
CREATE FUNCTION vec_cronos_v1.consultar_libro_saldo_interno_v1(
    p_empleado_ref text,p_desde date,p_hasta date,p_zona_horaria text
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE jornadas jsonb; marcajes jsonb; movimientos jsonb; pendientes integer;
BEGIN
    IF p_empleado_ref IS NULL OR p_empleado_ref !~ '^emp_[-A-Za-z0-9_]{22,128}$'
       OR p_desde IS NULL OR p_hasta IS NULL OR p_hasta<p_desde OR p_hasta-p_desde>366
       OR p_zona_horaria IS NULL OR p_zona_horaria NOT IN ('Europe/Madrid','Atlantic/Canary')
       OR p_empleado_ref IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'') THEN
        RAISE EXCEPTION 'consulta Cronos inválida' USING ERRCODE='PC003';
    END IF;
    -- Una segunda versión no sustituye silenciosamente el asiento histórico:
    -- esa fecha queda no disponible hasta un recálculo gobernado y versionado.
    WITH vigente AS (
      SELECT p.programacion_ref,p.fecha,p.turno_ref,p.politica_version_ref,
             p.fuente_ref,p.zona_horaria,p.minutos_previstos,p.version
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('programacion_ref',programacion_ref,'fecha',fecha,
      'turno_ref',turno_ref,'politica_version_ref',politica_version_ref,'fuente_ref',fuente_ref,
      'zona_horaria',zona_horaria,'minutos_previstos',minutos_previstos,'version',version)
      ORDER BY fecha),'[]'::jsonb) INTO jornadas FROM vigente;
    -- Los hechos limítrofes permiten parejas nocturnas. También se incluye
    -- el último hecho anterior si deja una secuencia abierta, aunque sea
    -- más antiguo que el margen. La zona local nunca depende del pool.
    SELECT coalesce(jsonb_agg(jsonb_build_object('marcaje_ref',m.marcaje_ref,'movimiento',m.movimiento,
      'instante_utc',m.instante_utc,'canal',m.material::jsonb->'canal',
      'tipo_origen',m.tipo_origen) ORDER BY m.instante_utc,m.marcaje_ref),'[]'::jsonb)
      INTO marcajes FROM vec_cronos_v1.marcaje_original m
      WHERE m.empleado_ref=p_empleado_ref
        AND ((m.instante_utc>=((p_desde-2)::timestamp AT TIME ZONE p_zona_horaria)
          AND m.instante_utc<((p_hasta+3)::timestamp AT TIME ZONE p_zona_horaria))
          OR (m.movimiento IN ('entrada','inicio_pausa','fin_pausa')
            AND m.marcaje_ref=(SELECT anterior.marcaje_ref
              FROM vec_cronos_v1.marcaje_original anterior
              WHERE anterior.empleado_ref=p_empleado_ref
                AND anterior.instante_utc<(p_desde::timestamp AT TIME ZONE p_zona_horaria)
              ORDER BY anterior.instante_utc DESC,anterior.marcaje_ref DESC LIMIT 1)));
    WITH ordenados AS (
      SELECT marcaje_ref,movimiento,instante_utc,
        lead(marcaje_ref) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_ref,
        lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_movimiento,
        lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc
      FROM vec_cronos_v1.marcaje_original WHERE empleado_ref=p_empleado_ref
    ), tramos AS (
      SELECT marcaje_ref,siguiente_ref,instante_utc,siguiente_utc FROM ordenados
      WHERE (movimiento='entrada' AND siguiente_movimiento IN ('salida','inicio_pausa'))
         OR (movimiento='fin_pausa' AND siguiente_movimiento IN ('salida','inicio_pausa'))
    ), partidos AS (
      SELECT t.marcaje_ref,t.siguiente_ref,g.fecha::date AS fecha,
        (extract(epoch FROM (least(t.siguiente_utc,(((g.fecha::date+1)::timestamp) AT TIME ZONE p_zona_horaria))
                  - greatest(t.instante_utc,((g.fecha::date)::timestamp AT TIME ZONE p_zona_horaria))))*1000000)::bigint AS microsegundos
      FROM tramos t CROSS JOIN LATERAL generate_series(
        greatest(p_desde,(t.instante_utc AT TIME ZONE p_zona_horaria)::date),
        least(p_hasta,((t.siguiente_utc - interval '1 microsecond') AT TIME ZONE p_zona_horaria)::date),
        interval '1 day') g(fecha)
      WHERE t.siguiente_utc>t.instante_utc AND t.siguiente_utc-t.instante_utc<=interval '24 hours'
    ), trabajo AS (
      SELECT fecha,sum(microsegundos)::bigint AS microsegundos,
        array_agg(marcaje_ref||'/'||siguiente_ref ORDER BY marcaje_ref,siguiente_ref) AS fuentes
      FROM partidos WHERE microsegundos>0 GROUP BY fecha
    ), previsto AS (
      SELECT p.fecha,p.programacion_ref,p.version,p.minutos_previstos
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    ), libro AS (
      SELECT fecha,'trabajado'::text AS tipo,microsegundos AS delta_microsegundos,
        to_jsonb(fuentes) AS fuentes FROM trabajo
      UNION ALL
      SELECT fecha,'previsto',-(minutos_previstos::bigint*60000000),jsonb_build_array(programacion_ref||'/v'||version)
      FROM previsto
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('fecha',fecha,'tipo',tipo,
       'delta_microsegundos',delta_microsegundos,'fuentes',fuentes) ORDER BY fecha,tipo),'[]'::jsonb)
      INTO movimientos FROM libro;
    -- No dar saldo definitivo cuando hay una secuencia inválida o falta
    -- programación. El lector verá hechos y un estado explícito.
    WITH o AS (
      SELECT movimiento,lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente,
       lag(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS anterior,
       lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc,
       instante_utc FROM vec_cronos_v1.marcaje_original WHERE empleado_ref=p_empleado_ref
    )
    -- Sin recorte inferior: una entrada antigua abierta sigue solapando B.
    SELECT count(*) INTO pendientes FROM o
    WHERE instante_utc<((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
      AND coalesce(siguiente_utc,'infinity'::timestamptz)>(p_desde::timestamp AT TIME ZONE p_zona_horaria)
      AND ((movimiento IN ('entrada','fin_pausa') AND
          (siguiente IS DISTINCT FROM 'salida' AND siguiente IS DISTINCT FROM 'inicio_pausa'
           OR siguiente_utc<=instante_utc OR siguiente_utc-instante_utc>interval '24 hours'))
        OR (movimiento='inicio_pausa' AND siguiente IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='salida' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='inicio_pausa' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='fin_pausa' AND anterior IS DISTINCT FROM 'inicio_pausa'));
    RETURN jsonb_build_object('empleado_ref',p_empleado_ref,'desde',p_desde,'hasta',p_hasta,
      'zona_horaria',p_zona_horaria,'jornadas',jornadas,'marcajes',marcajes,
      'movimientos_saldo',movimientos,'completo',
      pendientes=0 AND jsonb_array_length(jornadas)=p_hasta-p_desde+1
      AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(jornadas) j
        WHERE j->>'zona_horaria' IS DISTINCT FROM p_zona_horaria));
END $f$;
CREATE INDEX marcaje_original_persona_instante_idx ON vec_cronos_v1.marcaje_original(empleado_ref,instante_utc,marcaje_ref);
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
COMMIT;
