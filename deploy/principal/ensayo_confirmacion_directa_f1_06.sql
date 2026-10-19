\set ON_ERROR_STOP on
-- Tras la confirmación autorizada en la copia propia, pasar -v respuesta_ref=...
SELECT set_config('vec_ensayo.respuesta_ref', :'respuesta_ref', false);
DO $comprobar$
DECLARE ref text:=current_setting('vec_ensayo.respuesta_ref'); x record;
BEGIN
 IF ref !~ '^respuesta-portal:[a-f0-9]{64}$' THEN
  RAISE EXCEPTION 'F1-06: referencia inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO x FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa
  WHERE respuesta_ref=ref;
 IF NOT FOUND OR x.evento_sha256 IS DISTINCT FROM encode(sha256(x.evento_canonico),'hex')
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref=x.participacion_ref AND s.desde=x.situacion_desde_nueva
   AND s.situacion='pendiente_incorporacion' AND s.recibo_ref=x.recibo_ref)
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(
  x.bolsa_ref,clock_timestamp()) p WHERE p.participacion_ref=x.participacion_ref) THEN
  RAISE EXCEPTION 'F1-06: recibo, historia o marca incoherentes' USING ERRCODE='55000';
 END IF;
END $comprobar$;
SELECT 'F1-06: confirmación, recibo y salida pendiente coherentes' AS resultado;
