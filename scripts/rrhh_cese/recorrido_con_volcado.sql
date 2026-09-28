\set ON_ERROR_STOP on
-- Exige un volcado sintético con cese CT115 confirmado, relación CT75 opaca,
-- llamamiento Bolsa y candidatura B13 vinculada. No fabrica ninguno de ellos.
CREATE FUNCTION pg_temp.exigir_cese(p_ok boolean,p_mensaje text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
  IF p_ok IS DISTINCT FROM true THEN RAISE EXCEPTION 'Recorrido cese: %',p_mensaje; END IF;
END $$;
CREATE FUNCTION pg_temp.rechazar_cese_forjado(p_origen text,p_huella text,p_posicion bigint,p_estado text)
RETURNS boolean LANGUAGE plpgsql AS $$
DECLARE v_estado text;
BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(p_origen,p_huella,p_posicion);
  RETURN false;
EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS v_estado=RETURNED_SQLSTATE;
  RETURN v_estado=p_estado;
END $$;
DO $$ BEGIN
  EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_rrhh_cese_feed_pg18,vec_rrhh_cese_relevo_pg18',pg_my_temp_schema()::regnamespace);
END $$;
SET SESSION AUTHORIZATION vec_rrhh_cese_feed_pg18;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT count(*)>0 AS hay_cese FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100) \gset
SELECT pg_temp.exigir_cese(:'hay_cese'::boolean,'falta cese CT115 publicable en el volcado sintético');
SELECT bool_and(evento->>'tipo'='cese') AS solo_ceses
  FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100) \gset
SELECT pg_temp.exigir_cese(:'solo_ceses'::boolean,'CT129 publicó un evento distinto de cese');
SELECT evento::text AS evento,huella_sha256 AS huella,origen_ref AS origen,
       origen_posicion AS posicion,origen_creada_en::text AS creada
  FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100)
 ORDER BY origen_posicion,origen_ref LIMIT 1 \gset
SELECT count(*)=0 AS cursor_exclusivo
  FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(:'posicion'::bigint,:'origen',100)
 WHERE origen_ref=:'origen' \gset
SELECT pg_temp.exigir_cese(:'cursor_exclusivo'::boolean,'CT129 repitió la fila al avanzar el cursor');
COMMIT;
RESET SESSION AUTHORIZATION;

-- B45 no puede adelantar a B13 aunque el hecho CT129 sea verificable.
SET SESSION AUTHORIZATION vec_rrhh_cese_relevo_pg18;
SELECT pg_temp.rechazar_cese_forjado(:'origen',:'huella',:'posicion'::bigint,'23503') AS sin_b13 \gset
SELECT pg_temp.exigir_cese(:'sin_b13'::boolean,'B45 adelantó a B13');
RESET SESSION AUTHORIZATION;

SET ROLE vec_bolsa_llamamientos_propietario;
SELECT reutilizado AS b13_reutilizado,coalesce(participacion_ref,'') AS participacion,
       en_cuarentena AS cuarentena
  FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
    :'evento'::jsonb,:'huella',:'creada'::timestamptz,:'posicion'::bigint) \gset
RESET ROLE;
SELECT pg_temp.exigir_cese(NOT :'cuarentena'::boolean,'B13 puso el cese en cuarentena');
SELECT pg_temp.exigir_cese(:'participacion'<>'','B13 no resolvió la participación del llamamiento');

SET SESSION AUTHORIZATION vec_rrhh_cese_relevo_pg18;
SELECT reutilizada AS primera_reutilizada,recibo_ref AS recibo,
       candidato_ref AS candidato,disponible_desde::text AS disponible,
       politica_version AS politica
  FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(
    :'origen',:'huella',:'posicion'::bigint) \gset
SELECT pg_temp.exigir_cese(NOT :'primera_reutilizada'::boolean,'el cese elegido ya estaba aplicado en el volcado');
SELECT reutilizada AS replay,recibo_ref=:'recibo' AS mismo_recibo,
       disponible_desde::text=:'disponible' AS misma_fecha,
       politica_version::text=:'politica' AS misma_politica
  FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(
    :'origen',:'huella',:'posicion'::bigint) \gset
SELECT pg_temp.exigir_cese(:'replay'::boolean AND :'mismo_recibo'::boolean
  AND :'misma_fecha'::boolean AND :'misma_politica'::boolean,'replay divergente');
-- Un origen inventado, una huella adulterada y una posición distinta no
-- tienen hecho CT129 válido. Se captura SQLSTATE en la propia sesión nominal.
SELECT pg_temp.exigir_cese(
  pg_temp.rechazar_cese_forjado('evento:ct:forjado',:'huella',:'posicion'::bigint,'42501')
  AND pg_temp.rechazar_cese_forjado(:'origen',repeat('0',64),:'posicion'::bigint,'42501')
  AND pg_temp.rechazar_cese_forjado(:'origen',:'huella',:'posicion'::bigint+1,'42501'),
  'CT129/B45 aceptaron un cese forjado');
RESET SESSION AUTHORIZATION;

SELECT count(*)=1 AS unica_restriccion
  FROM vec_bolsa_llamamientos.restriccion_cese_bolsa WHERE origen_ref=:'origen' \gset
SELECT pg_temp.exigir_cese(:'unica_restriccion'::boolean,'B45 duplicó la restricción');
SELECT 'OK CT129/B13/B45 con recibo y replay' AS resultado;
