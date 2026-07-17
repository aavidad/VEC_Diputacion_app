-- Prueba estructural del orden de la seccion critica. La prueba positiva se
-- mantiene cerrada hasta COSE, pero ninguna refactorizacion puede mover el
-- reloj definitivo antes de un bloqueo ni quitar la segunda revalidacion.
DO $orden_temporal$
DECLARE
    definicion text := pg_get_functiondef(
      'vec_bolsa_llamamientos.guardar_propuesta_v1(jsonb,jsonb,bytea,bytea)'::regprocedure
    );
    posicion_bloqueo_auditoria integer;
    posicion_reloj_fresco integer;
    posicion_revalidacion_final integer;
    posicion_primer_efecto integer;
    numero_revalidaciones integer;
    llamada text :=
      'vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(';
BEGIN
    posicion_bloqueo_auditoria := strpos(
        definicion,
        'FROM vec_bolsa_llamamientos.auditoria_actual'
    );
    posicion_reloj_fresco := strpos(
        definicion,
        'v_ahora := clock_timestamp();'
    );
    posicion_revalidacion_final := posicion_reloj_fresco + strpos(
        substr(definicion, posicion_reloj_fresco), llamada
    ) - 1;
    posicion_primer_efecto := strpos(
        definicion,
        'INSERT INTO vec_bolsa_llamamientos.propuesta'
    );
    numero_revalidaciones := (
        length(definicion) - length(replace(definicion, llamada, ''))
    ) / length(llamada);
    IF posicion_bloqueo_auditoria <= 0 OR posicion_reloj_fresco <= 0 OR
       posicion_revalidacion_final <= 0 OR posicion_primer_efecto <= 0 OR
       numero_revalidaciones <> 2 OR
       NOT (
          posicion_bloqueo_auditoria < posicion_reloj_fresco AND
          posicion_reloj_fresco < posicion_revalidacion_final AND
          posicion_revalidacion_final < posicion_primer_efecto
       ) THEN
        RAISE EXCEPTION 'orden temporal inseguro: auditoria=%, reloj=%, revalidacion=%, efecto=%, llamadas=%',
          posicion_bloqueo_auditoria, posicion_reloj_fresco,
          posicion_revalidacion_final, posicion_primer_efecto,
          numero_revalidaciones;
    END IF;
END
$orden_temporal$;
