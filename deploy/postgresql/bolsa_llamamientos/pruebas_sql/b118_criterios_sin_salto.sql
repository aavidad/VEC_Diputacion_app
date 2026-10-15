\set ON_ERROR_STOP on
-- Ejecutar en copia desechable después de CT221 y B118.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$ DECLARE c jsonb; variante jsonb;
BEGIN
 c:='{"fuente_normativa":"BOP Granada 9/2026, art. 9.4", "mejora": {
  "habilitada":true,"modalidades_destino":["interinidad"],
  "causas_destino":["vacante"],"maximo_vacante_meses":36,
  "reduccion_por_modalidades_previas":["acumulacion_tareas","sustitucion"],
  "condiciones_actuales":[{"modalidad":"acumulacion_tareas"},
    {"modalidad":"sustitucion","requiere_baja_comprobada":true,
     "meses_transcurridos":{"meses":9,"incluye_limite":true}},
    {"modalidad":"programa","meses_duracion_inicial":
      {"meses":9,"incluye_limite":false}}]}}'::jsonb;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(c) IS NOT TRUE THEN
  RAISE EXCEPTION 'B118: regla publicada válida rechazada'; END IF;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
   c||'{"modalidades_aceptables":["interinidad"]}'::jsonb) IS NOT FALSE THEN
  RAISE EXCEPTION 'B118: una preferencia altera la selección'; END IF;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
   jsonb_set(c,'{mejora,maximo_vacante_meses}','1201'::jsonb)) IS NOT FALSE THEN
  RAISE EXCEPTION 'B118: supera el límite técnico del catálogo'; END IF;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
   jsonb_set(c,'{mejora}',(c->'mejora')-'causas_destino')) IS NOT FALSE THEN
  RAISE EXCEPTION 'B118: vacante sin causa gobernada'; END IF;
 IF vec_bolsa_llamamientos.motivo_relacion_oferta_v1(
   'participacion:sin-vinculo','trabajando',false,'interinidad','vacante',
   clock_timestamp(),c) IS DISTINCT FROM 'pendiente_comprobacion_rrhh' THEN
  RAISE EXCEPTION 'B118: hechos ausentes no dejan posición pendiente'; END IF;
 variante:=jsonb_set(c,'{mejora,supuestos_reduccion_vacante}','["programa"]'::jsonb);
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(variante) IS NOT TRUE
 OR variante#>'{mejora,reduccion_por_modalidades_previas}' IS DISTINCT FROM
   '["acumulacion_tareas","sustitucion"]'::jsonb THEN
  RAISE EXCEPTION 'B118: confunde el supuesto con los periodos descontables'; END IF;
 FOREACH variante IN ARRAY ARRAY['null'::jsonb,'[]'::jsonb,'["programa","programa"]'::jsonb,
   '["ajena"]'::jsonb,'{}'::jsonb,'[7]'::jsonb] LOOP
  IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
    jsonb_set(c,'{mejora,supuestos_reduccion_vacante}',variante)) IS NOT FALSE THEN
   RAISE EXCEPTION 'B118: supuesto inválido admitido'; END IF;
 END LOOP;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
   jsonb_set(c,'{mejora,campo_ajeno}','true'::jsonb)) IS NOT FALSE THEN
  RAISE EXCEPTION 'B118: séptimo campo ajeno admitido'; END IF;
END $test$;
-- El propietario efectivo de AD276 debe poder ejecutar el validador cerrado.
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $lector$ BEGIN
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1('{}'::jsonb) IS NOT TRUE
 OR has_function_privilege('public',
   'vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'B118: permiso nominal del validador incorrecto';
 END IF;
END $lector$;
ROLLBACK;
