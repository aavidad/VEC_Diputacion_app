\set ON_ERROR_STOP on
BEGIN;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $ct89$
DECLARE
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_estado vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_material vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1;
    v_bruto bytea;
    v_evidencia text;
    v_localizador text;
    v_vector_bruto bytea := decode(repeat('ff', 32), 'hex');
    v_vector_token text;
BEGIN
    v_vector_token := rtrim(translate(encode(v_vector_bruto, 'base64'), '+/', '-_'), E'=\n');
    IF encode(sha256(v_vector_bruto), 'hex') <>
           'af9613760f72635fbdb44a5a0a63c39f12af30f950a6ee5c971be188e89c4051'
       OR encode(sha256(v_vector_bruto), 'hex') IS NOT DISTINCT FROM
          encode(sha256(convert_to(v_vector_token, 'UTF8')), 'hex') THEN
        RAISE EXCEPTION 'CT89: vector de evidencia binaria incompatible';
    END IF;

    SELECT p.expediente_ref, p.organizacion_ref,
        p.numero_visible, p.version, p.flujo_ref,
        p.flujo_version, p.flujo_huella_sha256,
        p.fase_clave, p.estado_clave,
        p.centro_ref, p.categoria_ref,
        COALESCE(p.modalidad_clave, ''),
        COALESCE(p.unidad_ref, ''),
        p.creado_en, p.actualizado_en
      INTO STRICT v_resumen
      FROM vec_contratacion_temporal.publicacion_version_rrhh p
     ORDER BY p.corte_global
     LIMIT 1;
    v_estado := ROW(false, NULL, 1, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
        ::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
    v_material := ROW(ARRAY[v_resumen], true, v_resumen.actualizado_en,
        v_resumen.expediente_ref)::vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
    v_salida := vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(
        v_estado, v_material
    );
    v_bruto := decode(rpad(translate(v_salida.cursor_siguiente, '-_', '+/'), 44, '='), 'base64');
    v_evidencia := encode(sha256(v_bruto), 'hex');
    v_localizador := encode(sha256(convert_to(v_salida.cursor_siguiente, 'UTF8')), 'hex');

    IF NOT v_salida.hay_mas
       OR length(v_bruto) <> 32
       OR rtrim(translate(encode(v_bruto, 'base64'), '+/', '-_'), E'=\n')
            IS DISTINCT FROM v_salida.cursor_siguiente
       OR v_evidencia IS DISTINCT FROM encode(v_salida.cursor_huella, 'hex')
       OR v_localizador IS DISTINCT FROM v_salida.token_nuevo_huella_sha256
       OR v_evidencia IS NOT DISTINCT FROM v_localizador THEN
        RAISE EXCEPTION 'CT89: preparar_salida no separa evidencia binaria y localizador ASCII';
    END IF;

    -- El negativo de aplicar_efectos exige una línea base durable legítima y
    -- una única mutación de cursor_huella; CT44B no sirve porque su cierre
    -- terminal ya se rechaza por otra causa. El recorrido 50+2, replay y esa
    -- mutación se verifican en un ensayo nativo aislado antes del GO funcional.
END
$ct89$;
ROLLBACK;
