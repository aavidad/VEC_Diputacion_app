\set ON_ERROR_STOP on

DROP FUNCTION public.invocar_salida_nula_ct45(text, text, text);
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DROP FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
);
DROP FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
);
ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_cuadro_rrhh_real_ct45(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RENAME TO motor_consultar_cuadro_rrhh_v1;
ALTER FUNCTION
vec_contratacion_temporal.motor_consultar_detalle_rrhh_real_ct45(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3
) RENAME TO motor_consultar_detalle_rrhh_v1;
COMMIT;
