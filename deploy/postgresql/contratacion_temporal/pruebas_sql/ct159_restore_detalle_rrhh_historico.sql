\set ON_ERROR_STOP on
-- Solo sobre clon sintético PG18 tras CT159, con un operador que pueda SET ROLE.
-- La tabla temporal reproduce la fase de datos de pg_restore: CHECK activos,
-- trigger de inserción instalado después. No duplica recibos en la tabla original.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
CREATE TEMP TABLE ct159_prueba (
    LIKE vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
    INCLUDING CONSTRAINTS INCLUDING GENERATED
) ON COMMIT DROP;
ALTER TABLE pg_temp.ct159_prueba
    OWNER TO vec_contratacion_temporal_propietario;
SET LOCAL ROLE vec_contratacion_temporal_propietario;

SET LOCAL statement_timeout = '20min';
DO $prueba$
DECLARE
    v_columnas text;
    v_v3 jsonb;
    v_cuadro jsonb;
    v_malo jsonb;
    v_error boolean;
    v_restriccion text;
    v_v1 integer;
    v_v2 integer;
    v_legacy integer;
    v_cantidad bigint;
    v_original bigint;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger t
        JOIN pg_catalog.pg_proc f ON f.oid = t.tgfoid
        WHERE t.tgrelid =
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'::regclass
          AND t.tgname = 'prueba_resultado_recibo_rrhh_v2_detalle_nuevo_ct159'
          AND NOT t.tgisinternal AND t.tgenabled = 'O' AND t.tgtype = 7
          AND f.oid =
            'vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159()'
            ::regprocedure
          AND NOT f.prosecdef
          AND f.proowner = 'vec_contratacion_temporal_propietario'::regrole
          AND f.proconfig @> ARRAY['search_path=pg_catalog', 'TimeZone=UTC']
          AND NOT EXISTS (
              SELECT 1 FROM pg_catalog.aclexplode(
                  COALESCE(f.proacl, pg_catalog.acldefault('f', f.proowner))
              ) acl
              WHERE acl.grantee = 0 AND acl.privilege_type = 'EXECUTE'
          )
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
        WHERE oid =
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'::regclass
          AND relrowsecurity AND relforcerowsecurity
    ) OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
        WHERE conrelid =
            'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2'::regclass
          AND contype = 'f') <> 5 THEN
        RAISE EXCEPTION 'CT159: trigger, RLS o relaciones incompatibles';
    END IF;

    IF pg_catalog.has_table_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2', 'SELECT'
    ) OR pg_catalog.has_table_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2', 'INSERT'
    ) THEN
        RAISE EXCEPTION 'CT159: el consultor tiene acceso directo';
    END IF;

    SELECT pg_catalog.string_agg(pg_catalog.format('%I', attname), ', '
                                ORDER BY attnum)
      INTO STRICT v_columnas FROM pg_catalog.pg_attribute
     WHERE attrelid = 'pg_temp.ct159_prueba'::regclass
       AND attnum > 0 AND NOT attisdropped AND attgenerated = '';
    SELECT pg_catalog.count(*) INTO v_original
      FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2;
    -- Todas las filas pasan los CHECK conservados sin recanonizar detalle.
    EXECUTE pg_catalog.format(
        'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
        || 'vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2',
        v_columnas, v_columnas
    );
    SELECT pg_catalog.count(*) INTO v_cantidad FROM pg_temp.ct159_prueba;
    IF v_cantidad <> v_original THEN
        RAISE EXCEPTION 'CT159: la carga histórica perdió filas';
    END IF;
    SELECT pg_catalog.count(*) FILTER (WHERE pg_catalog.substr(contenido_canonico,
               1, pg_catalog.octet_length(pg_catalog.convert_to(
                   'VEC-CT-CONTENIDO-DETALLE-RRHH-V1' || pg_catalog.chr(10), 'UTF8'
               ))) = pg_catalog.convert_to(
                   'VEC-CT-CONTENIDO-DETALLE-RRHH-V1' || pg_catalog.chr(10), 'UTF8')),
           pg_catalog.count(*) FILTER (WHERE pg_catalog.substr(contenido_canonico,
               1, pg_catalog.octet_length(pg_catalog.convert_to(
                   'VEC-CT-CONTENIDO-DETALLE-RRHH-V2' || pg_catalog.chr(10), 'UTF8'
               ))) = pg_catalog.convert_to(
                   'VEC-CT-CONTENIDO-DETALLE-RRHH-V2' || pg_catalog.chr(10), 'UTF8')),
           pg_catalog.count(*) FILTER (WHERE
               (detalle).fiscalizacion_presente IS NULL)
      INTO v_v1, v_v2, v_legacy FROM pg_temp.ct159_prueba
     WHERE tipo_consulta = 'detalle';
    IF v_v1 = 0 OR v_v2 = 0 OR v_legacy = 0 THEN
        RAISE EXCEPTION 'CT159: faltan testigos históricos V1/V2 con campos NULL';
    END IF;
    RAISE NOTICE 'CT159: V1=%, V2=%, legacy NULL=%, total=%',
        v_v1, v_v2, v_legacy, v_cantidad;

    -- El mismo trigger que la tabla real, instalado después de los datos.
    CREATE TRIGGER ct159_detalle_nuevo
    BEFORE INSERT ON pg_temp.ct159_prueba
    FOR EACH ROW EXECUTE FUNCTION
        vec_contratacion_temporal.validar_detalle_nuevo_recibo_rrhh_ct159();
    SELECT pg_catalog.to_jsonb(p) INTO STRICT v_v3
      FROM pg_temp.ct159_prueba p
     WHERE tipo_consulta = 'detalle'
       AND (detalle).fiscalizacion_presente IS NOT NULL
     LIMIT 1;
    EXECUTE pg_catalog.format(
        'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
        || 'pg_catalog.jsonb_populate_record(NULL::pg_temp.ct159_prueba, $1)',
        v_columnas, v_columnas
    ) USING v_v3;

    -- Canon alterado: el trigger rechaza antes de comprobar hashes/recibo.
    v_malo := pg_catalog.jsonb_set(v_v3, '{contenido_canonico}',
        pg_catalog.to_jsonb(pg_catalog.convert_to('alterado', 'UTF8')));
    v_error := false;
    BEGIN
        EXECUTE pg_catalog.format(
            'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
            || 'pg_catalog.jsonb_populate_record(NULL::pg_temp.ct159_prueba, $1)',
            v_columnas, v_columnas
        ) USING v_malo;
    EXCEPTION WHEN check_violation THEN
        GET STACKED DIAGNOSTICS v_restriccion = CONSTRAINT_NAME;
        v_error := v_restriccion =
            'prueba_resultado_recibo_rrhh_v2_detalle_nuevo_ct159';
    END;
    IF NOT v_error THEN RAISE EXCEPTION 'CT159: aceptó un canon nuevo alterado'; END IF;

    -- La forma histórica no sirve como inserción nueva V3.
    v_malo := pg_catalog.jsonb_set(v_v3, '{detalle,fiscalizacion_presente}', 'null');
    v_error := false;
    BEGIN
        EXECUTE pg_catalog.format(
            'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
            || 'pg_catalog.jsonb_populate_record(NULL::pg_temp.ct159_prueba, $1)',
            v_columnas, v_columnas
        ) USING v_malo;
    EXCEPTION WHEN invalid_parameter_value THEN v_error := true;
    END;
    IF NOT v_error THEN RAISE EXCEPTION 'CT159: aceptó un detalle nuevo legacy'; END IF;

    SELECT pg_catalog.to_jsonb(p) INTO STRICT v_cuadro
      FROM pg_temp.ct159_prueba p WHERE tipo_consulta = 'cuadro' LIMIT 1;
    EXECUTE pg_catalog.format(
        'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
        || 'pg_catalog.jsonb_populate_record(NULL::pg_temp.ct159_prueba, $1)',
        v_columnas, v_columnas
    ) USING v_cuadro;
    v_malo := pg_catalog.jsonb_set(v_cuadro, '{hay_mas}',
        pg_catalog.to_jsonb(NOT (v_cuadro ->> 'hay_mas')::boolean));
    v_error := false;
    BEGIN
        EXECUTE pg_catalog.format(
            'INSERT INTO pg_temp.ct159_prueba (%s) SELECT %s FROM '
            || 'pg_catalog.jsonb_populate_record(NULL::pg_temp.ct159_prueba, $1)',
            v_columnas, v_columnas
        ) USING v_malo;
    EXCEPTION WHEN check_violation THEN
        GET STACKED DIAGNOSTICS v_restriccion = CONSTRAINT_NAME;
        v_error := v_restriccion = 'prueba_resultado_recibo_rrhh_v2_check1';
    WHEN invalid_parameter_value THEN
        -- El canon rechaza hay_mas/cursor incoherentes antes del CHECK.
        v_error := true;
    END;
    IF NOT v_error THEN RAISE EXCEPTION 'CT159: aceptó un cuadro nuevo alterado'; END IF;
    SELECT pg_catalog.count(*) INTO v_cantidad FROM pg_temp.ct159_prueba;
    IF v_cantidad <> v_original + 2 THEN
        RAISE EXCEPTION 'CT159: las inserciones rechazadas dejaron filas';
    END IF;
END
$prueba$;
SET LOCAL statement_timeout = '30s';

SET LOCAL ROLE vec_contratacion_temporal_consultor_rrhh;
DO $denegacion$
DECLARE
    v_error boolean := false;
BEGIN
    BEGIN
        PERFORM acceso_ref FROM
            vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 LIMIT 1;
    EXCEPTION WHEN insufficient_privilege THEN v_error := true;
    END;
    IF NOT v_error THEN RAISE EXCEPTION 'CT159: SELECT directo no fue denegado'; END IF;
    v_error := false;
    BEGIN
        INSERT INTO vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(
            acceso_ref
        ) SELECT 'acceso:ct159:denegado' WHERE false;
    EXCEPTION WHEN insufficient_privilege THEN v_error := true;
    END;
    IF NOT v_error THEN RAISE EXCEPTION 'CT159: INSERT directo no fue denegado'; END IF;
END
$denegacion$;
ROLLBACK;
