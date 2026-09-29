-- Comprobaciones de CT-000142 sobre un clon desechable de la principal (datos
-- sintéticos) ya migrado. Se ejecuta como superusuario; las consultas se
-- llaman con el rol propietario, igual que desde sus fachadas. Todo ocurre en
-- una transacción que termina en ROLLBACK. Cada comprobación imprime «ok»; un
-- fallo imprime «FALLO ...» y detiene el ensayo.
\set ON_ERROR_STOP on
\pset tuples_only on
\pset format unaligned
BEGIN;

CREATE FUNCTION pg_temp.comprobar(p_condicion boolean, p_nombre text)
RETURNS text LANGUAGE plpgsql AS $$
BEGIN
    IF p_condicion IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'FALLO %', p_nombre;
    END IF;
    RETURN 'ok';
END $$;

CREATE TEMP TABLE ct142_tecnicos AS
SELECT alta.expediente_ref, alta.numero_visible AS numero_anterior, alta.creada_en
  FROM vec_contratacion_temporal.expediente_alta alta
 WHERE alta.numero_visible ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$';
GRANT SELECT ON ct142_tecnicos TO PUBLIC;

-- 1. Todo expediente técnico tiene número anual; la historia del alta no cambia.
SELECT pg_temp.comprobar(
    (SELECT pg_catalog.count(*) FROM ct142_tecnicos) > 0
    AND NOT EXISTS (
        SELECT 1 FROM ct142_tecnicos t
         WHERE NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.numeracion_anual_asignada n
                            WHERE n.expediente_ref = t.expediente_ref
                              AND n.numero_anterior = t.numero_anterior)),
    'todos los técnicos numerados sin tocar expediente_alta');

-- 2. Correlativos, sin huecos, en orden de alta y hasta el contador vigente.
SELECT pg_temp.comprobar(
    NOT EXISTS (
        SELECT 1 FROM (
            SELECT pg_catalog.substring(n.numero_visible, '([0-9]+)$')::integer AS cifra,
                   pg_catalog.row_number() OVER (ORDER BY t.creada_en, t.expediente_ref COLLATE "C") AS orden,
                   pg_catalog.max(pg_catalog.substring(n.numero_visible, '([0-9]+)$')::integer) OVER () AS maximo,
                   pg_catalog.count(*) OVER () AS total
              FROM ct142_tecnicos t
              JOIN vec_contratacion_temporal.numeracion_anual_asignada n USING (expediente_ref)
        ) x
         WHERE x.cifra <> x.maximo - x.total + x.orden)
    AND (SELECT pg_catalog.max(pg_catalog.substring(n.numero_visible, '([0-9]+)$')::integer)
           FROM vec_contratacion_temporal.numeracion_anual_asignada n WHERE n.anio = 2026)
        = (SELECT e.ultimo FROM vec_contratacion_temporal.numeracion_expedientes e WHERE e.anio = 2026),
    'correlativos por fecha de alta hasta el contador');

-- 3. Repetir no renumera ni consume números.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE TEMP TABLE ct142_contador AS
SELECT ultimo FROM vec_contratacion_temporal.numeracion_expedientes WHERE anio = 2026;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1() = 0, 'repetición idempotente');
SELECT pg_temp.comprobar(
    (SELECT ultimo FROM vec_contratacion_temporal.numeracion_expedientes WHERE anio = 2026)
    = (SELECT ultimo FROM ct142_contador), 'contador intacto al repetir');
RESET ROLE;

-- 4. Solo adición.
SAVEPOINT mutacion;
DO $$ BEGIN
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    UPDATE vec_contratacion_temporal.numeracion_anual_asignada SET numero_visible = '2026/CT-999999';
    RAISE EXCEPTION 'FALLO actualización admitida';
EXCEPTION WHEN OTHERS THEN
    IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
END $$;
ROLLBACK TO SAVEPOINT mutacion;
SELECT pg_temp.comprobar(
    NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.numeracion_anual_asignada
                 WHERE numero_visible = '2026/CT-999999'), 'historia inmutable');

-- 5. Un evento por asignación, encadenado y con la cabeza del control al día.
SELECT pg_temp.comprobar(
    (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.outbox_expediente_integral o
       JOIN vec_contratacion_temporal.numeracion_anual_asignada n USING (evento_ref)
      WHERE o.tipo_evento = 'contratacion_temporal.numero_expediente_asignado'
        AND o.expediente_ref = n.expediente_ref
        AND o.version_expediente = n.version_expediente
        AND pg_catalog.convert_from(o.payload_canonico, 'UTF8')::jsonb ->> 'numero_visible' = n.numero_visible)
    = (SELECT pg_catalog.count(*) FROM ct142_tecnicos)
    AND NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral o
          JOIN vec_contratacion_temporal.outbox_expediente_integral previo
            ON previo.secuencia = o.secuencia - 1
         WHERE o.tipo_evento = 'contratacion_temporal.numero_expediente_asignado'
           AND o.anterior_sha256 <> previo.huella_sha256)
    AND (SELECT c.cabeza_outbox_sha256 = o.huella_sha256 AND c.secuencia_outbox = o.secuencia
           FROM vec_contratacion_temporal.control_cadenas_expediente_integral c,
                LATERAL (SELECT * FROM vec_contratacion_temporal.outbox_expediente_integral
                          ORDER BY secuencia DESC LIMIT 1) o),
    'outbox encadenado');

-- 6. Cuadro, totales y detalle muestran el número anual.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE TEMP TABLE ct142_cuadro AS
SELECT r.expediente_ref, r.numero_visible
  FROM pg_catalog.unnest((vec_contratacion_temporal.materializar_cuadro_rrhh_v1(
        ROW('organizacion:desarrollo:dipgra', 'organizacion', 'organizacion:desarrollo:dipgra')
            ::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
        ROW('', '', '', 100, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        ROW(false, NULL, (SELECT ultimo_corte FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control),
            0, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
            ::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1
     )).resumenes) r;
SELECT pg_temp.comprobar(
    (SELECT pg_catalog.count(*) FROM ct142_cuadro) >= (SELECT pg_catalog.count(*) FROM ct142_tecnicos)
    AND NOT EXISTS (SELECT 1 FROM ct142_cuadro WHERE numero_visible ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$')
    AND NOT EXISTS (SELECT 1 FROM ct142_cuadro c JOIN vec_contratacion_temporal.numeracion_anual_asignada n
                     USING (expediente_ref) WHERE c.numero_visible <> n.numero_visible),
    'cuadro con número anual');
SELECT pg_temp.comprobar(
    t.total_filtrado = (SELECT pg_catalog.count(*) FROM ct142_cuadro WHERE numero_visible LIKE '2026/CT-0%')
    AND t.total_filtrado > (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.expediente_alta
                             WHERE numero_visible LIKE '2026/CT-0%'),
    'totales filtran por número anual')
  FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(
        ROW('organizacion:desarrollo:dipgra', 'organizacion', 'organizacion:desarrollo:dipgra')
            ::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
        ROW('2026/CT-0', '', '', 100, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        '') t;
SELECT pg_temp.comprobar(
    ((vec_contratacion_temporal.materializar_detalle_rrhh_v1(
        ROW('organizacion:desarrollo:dipgra', 'organizacion', 'organizacion:desarrollo:dipgra')
            ::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
        ROW(n.expediente_ref, actual.version)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
        (SELECT ultimo_corte FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control)
     )).detalle).resumen.numero_visible = n.numero_visible,
    'detalle con número anual')
  FROM vec_contratacion_temporal.numeracion_anual_asignada n
  JOIN vec_contratacion_temporal.expediente_integral_actual actual USING (expediente_ref)
 ORDER BY n.asignado_en LIMIT 3;

-- 7. La lista del centro también.
CREATE TEMP TABLE ct142_centro AS
SELECT DISTINCT e.expediente_ref, e.numero_visible, n.numero_visible AS asignado
  FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion c
  JOIN vec_contratacion_temporal.numeracion_anual_asignada n USING (expediente_ref)
  JOIN vec_contratacion_temporal.peticion_centro_revision r ON r.peticion_ref = c.peticion_ref
  CROSS JOIN LATERAL vec_contratacion_temporal.expedientes_centro_ct124(
      r.peticion -> 'configuracion' -> 'solicitante', 'organizacion:desarrollo:dipgra') e
 WHERE e.expediente_ref = n.expediente_ref;
SELECT pg_temp.comprobar(
    EXISTS (SELECT 1 FROM ct142_centro)
    AND NOT EXISTS (SELECT 1 FROM ct142_centro WHERE numero_visible <> asignado),
    'lista del centro con número anual');
RESET ROLE;

-- 8. Las altas nuevas siguen el mismo contador.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SELECT pg_temp.comprobar(
    vec_contratacion_temporal.siguiente_numero_visible_v1(2026)
    = '2026/CT-' || pg_catalog.lpad(((SELECT ultimo FROM ct142_contador) + 1)::text, 6, '0'),
    'alta nueva continúa la serie');
RESET ROLE;

-- 9. Privilegios mínimos.
SELECT pg_temp.comprobar(
    NOT has_function_privilege('public', 'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)', 'EXECUTE')
    AND NOT has_function_privilege('public', 'vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()', 'EXECUTE')
    AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor', 'vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()', 'EXECUTE')
    AND NOT has_table_privilege('public', 'vec_contratacion_temporal.numeracion_anual_asignada', 'SELECT')
    AND NOT has_table_privilege('vec_contratacion_temporal_ejecutor', 'vec_contratacion_temporal.numeracion_anual_asignada', 'SELECT'),
    'privilegios mínimos');

ROLLBACK;
