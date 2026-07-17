-- Entrada real del read model. El objeto es cerrado y solo admite datos
-- agregados o referencias opacas; cualquier clave adicional se rechaza.
BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_panel.publicar_proyeccion_panel_v1(p_documento jsonb)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    selector jsonb;
    clase text;
    organizacion text;
    unidad text := '';
    revision_nueva text;
    actualizada timestamptz;
    registrada timestamptz;
    huella text;
    existente record;
    actual record;
    elemento record;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'publicacion rechazada: requiere SERIALIZABLE';
    END IF;
    IF p_documento IS NULL OR jsonb_typeof(p_documento) <> 'object'
       OR pg_column_size(p_documento) > 8388608
       OR (SELECT count(*) FROM jsonb_object_keys(p_documento)) <> 7
       OR NOT (p_documento ?& ARRAY[
           'esquema', 'selector', 'revision', 'actualizada_en',
           'indicadores', 'convocatorias', 'actuaciones_pendientes'
       ])
       OR p_documento ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.panel.proyeccion.v1'
       OR vec_bolsa_panel.selector_valido(
              p_documento -> 'selector'
          ) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'revision') <> 'string'
       OR vec_bolsa_panel.referencia_opaca_valida(
              p_documento ->> 'revision', 'rev_'
          ) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'actualizada_en') <> 'string'
       OR vec_bolsa_panel.instante_utc_microsegundo_valido(
              p_documento ->> 'actualizada_en'
          ) IS NOT TRUE
       OR vec_bolsa_panel.indicadores_validos(
              p_documento -> 'indicadores'
          ) IS NOT TRUE
       OR jsonb_typeof(p_documento -> 'convocatorias') <> 'array'
       OR jsonb_array_length(p_documento -> 'convocatorias') > 40
       OR jsonb_typeof(p_documento -> 'actuaciones_pendientes') <> 'array'
       OR jsonb_array_length(
              p_documento -> 'actuaciones_pendientes'
          ) > 80
       OR EXISTS (
           SELECT 1 FROM jsonb_array_elements(
               p_documento -> 'convocatorias'
           ) AS fila(valor)
            WHERE vec_bolsa_panel.convocatoria_resumen_valida(valor)
                  IS NOT TRUE
       )
       OR EXISTS (
           SELECT 1 FROM jsonb_array_elements(
               p_documento -> 'actuaciones_pendientes'
           ) AS fila(valor)
            WHERE vec_bolsa_panel.actuacion_pendiente_valida(valor)
                  IS NOT TRUE
       )
       OR (SELECT count(*) FROM (
               SELECT DISTINCT valor ->> 'convocatoria_ref'
                 FROM jsonb_array_elements(
                     p_documento -> 'convocatorias'
                 ) AS fila(valor)
           ) AS unicas) <>
          jsonb_array_length(p_documento -> 'convocatorias')
       OR (SELECT count(*) FROM (
               SELECT DISTINCT valor ->> 'actuacion_ref'
                 FROM jsonb_array_elements(
                     p_documento -> 'actuaciones_pendientes'
                 ) AS fila(valor)
           ) AS unicas) <>
          jsonb_array_length(p_documento -> 'actuaciones_pendientes') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyeccion del panel invalida';
    END IF;

    selector := p_documento -> 'selector';
    clase := selector ->> 'clase';
    organizacion := selector ->> 'organizacion_ref';
    IF clase = 'unidad_gestion' THEN
        unidad := selector ->> 'unidad_gestion_ref';
    END IF;
    revision_nueva := p_documento ->> 'revision';
    actualizada := (p_documento ->> 'actualizada_en')::timestamptz;
    registrada := clock_timestamp();
    IF actualizada > registrada THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'la proyeccion no puede proceder del futuro';
    END IF;
    huella := encode(sha256(convert_to(p_documento::text, 'UTF8')), 'hex');

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_panel:proyeccion:' || clase || ':' || organizacion || ':' || unidad,
        0
    ));
    SELECT p.documento_huella_sha256 INTO existente
      FROM vec_bolsa_panel.proyeccion_panel AS p
     WHERE p.clase_ambito = clase
       AND p.organizacion_ref = organizacion
       AND p.unidad_gestion_ref = unidad
       AND p.revision = revision_nueva;
    IF FOUND THEN
        IF existente.documento_huella_sha256 <> huella THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'revision reutilizada con otro contenido';
        END IF;
        RETURN true;
    END IF;
    SELECT * INTO actual
      FROM vec_bolsa_panel.proyeccion_actual AS puntero
     WHERE puntero.clase_ambito = clase
       AND puntero.organizacion_ref = organizacion
       AND puntero.unidad_gestion_ref = unidad
     FOR UPDATE;
    IF FOUND AND actualizada <= actual.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'la proyeccion no avanza de forma monotona';
    END IF;

    INSERT INTO vec_bolsa_panel.proyeccion_panel(
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
        actualizada_en, indicadores, documento_huella_sha256, registrada_en
    ) VALUES (
        clase, organizacion, unidad, revision_nueva, actualizada,
        p_documento -> 'indicadores', huella, registrada
    );
    FOR elemento IN
        SELECT valor, ordinalidad
          FROM jsonb_array_elements(p_documento -> 'convocatorias')
               WITH ORDINALITY AS fila(valor, ordinalidad)
    LOOP
        INSERT INTO vec_bolsa_panel.convocatoria_resumen(
            clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
            convocatoria_ref, ordinal, documento
        ) VALUES (
            clase, organizacion, unidad, revision_nueva,
            elemento.valor ->> 'convocatoria_ref',
            elemento.ordinalidad::smallint, elemento.valor
        );
    END LOOP;
    FOR elemento IN
        SELECT valor, ordinalidad
          FROM jsonb_array_elements(
              p_documento -> 'actuaciones_pendientes'
          ) WITH ORDINALITY AS fila(valor, ordinalidad)
    LOOP
        INSERT INTO vec_bolsa_panel.actuacion_pendiente(
            clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
            actuacion_ref, ordinal, documento
        ) VALUES (
            clase, organizacion, unidad, revision_nueva,
            elemento.valor ->> 'actuacion_ref',
            elemento.ordinalidad::smallint, elemento.valor
        );
    END LOOP;
    INSERT INTO vec_bolsa_panel.proyeccion_actual(
        clase_ambito, organizacion_ref, unidad_gestion_ref, revision,
        actualizada_en
    ) VALUES (clase, organizacion, unidad, revision_nueva, actualizada)
    ON CONFLICT (clase_ambito, organizacion_ref, unidad_gestion_ref)
    DO UPDATE SET revision = EXCLUDED.revision,
                  actualizada_en = EXCLUDED.actualizada_en;
    RETURN true;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_panel TO vec_bolsa_panel_proyector;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb)
    TO vec_bolsa_panel_proyector;
COMMIT;
