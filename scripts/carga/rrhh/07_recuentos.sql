\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT jsonb_pretty(jsonb_build_object(
  'ct', jsonb_build_object(
    'expedientes_originales', (SELECT count(DISTINCT expediente_ref)
      FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref NOT LIKE 'expediente:perf:%'),
    'publicaciones_originales', (SELECT count(*)
      FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref NOT LIKE 'expediente:perf:%'),
    'expedientes_proyeccion', (SELECT count(DISTINCT expediente_ref)
      FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref LIKE 'expediente:perf:%'),
    'publicaciones_proyeccion', (SELECT count(*)
      FROM vec_contratacion_temporal.publicacion_version_rrhh
      WHERE expediente_ref LIKE 'expediente:perf:%'),
    'peticiones_proyeccion', (SELECT count(*)
      FROM vec_contratacion_temporal.peticion_centro_revision
      WHERE peticion_ref LIKE 'peticion:perf:%'),
    'fases', (SELECT jsonb_object_agg(fase_clave,total)
      FROM (SELECT fase_clave,count(*) total
        FROM (SELECT DISTINCT ON (expediente_ref) expediente_ref,fase_clave
              FROM vec_contratacion_temporal.publicacion_version_rrhh
              WHERE expediente_ref LIKE 'expediente:perf:%'
              ORDER BY expediente_ref,version DESC) ult
        GROUP BY fase_clave) x),
    'incidencias', (SELECT count(*)
      FROM (SELECT DISTINCT ON (expediente_ref) expediente_ref,estado_clave
            FROM vec_contratacion_temporal.publicacion_version_rrhh
            WHERE expediente_ref LIKE 'expediente:perf:%'
            ORDER BY expediente_ref,version DESC) ult
      WHERE estado_clave='incidencia'),
    'marcas_urgencia_proyeccion', (SELECT count(*)
      FROM vec_contratacion_temporal.urgencia_expediente_analisis
      WHERE expediente_ref LIKE 'expediente:perf:%'),
    'cronologia_incoherente', (SELECT count(*)
      FROM vec_contratacion_temporal.publicacion_version_rrhh p
      JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh e USING(expediente_ref,version)
      WHERE p.expediente_ref LIKE 'expediente:perf:%'
        AND (e.fase_desde < p.creado_en OR e.fase_desde > p.actualizado_en)),
    'fase_mas_de_30_dias', (SELECT count(*)
      FROM (SELECT DISTINCT ON (expediente_ref) expediente_ref,version
            FROM vec_contratacion_temporal.publicacion_version_rrhh
            WHERE expediente_ref LIKE 'expediente:perf:%'
            ORDER BY expediente_ref,version DESC) ult
      JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh e USING(expediente_ref,version)
      WHERE e.fase_desde < now()-interval '30 days'),
    'vencidos_calculados', 'pendiente_de_calculadora_de_aplicacion'),
  'bolsa', jsonb_build_object(
    'bolsas_originales', (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion
      WHERE bolsa_ref <> 'bolsa:rendimiento:20261012'),
    'participaciones_originales', (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion_entrada
      WHERE instantanea_ref <> 'instantanea:rendimiento:20261012'),
    'participaciones_proyeccion', (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion_entrada
      WHERE instantanea_ref = 'instantanea:rendimiento:20261012'),
    'vinculos_candidato_proyeccion', (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato
      WHERE acta_ref='acta:rendimiento:20261012'),
    'participaciones_canonicas', (SELECT jsonb_array_length(convert_from(instantanea_canonica,'UTF8')::jsonb->'entradas')
      FROM vec_bolsa_llamamientos.instantanea_orden_bolsa
      WHERE bolsa_ref='bolsa:rendimiento:20261012'),
    'situaciones_proyeccion', (SELECT jsonb_object_agg(situacion,total)
      FROM (SELECT situacion,count(*) total FROM vec_bolsa_llamamientos.situacion_participacion
        WHERE participacion_ref LIKE 'participacion:rendimiento:%' GROUP BY situacion) x)
))) AS resumen;
COMMIT;
