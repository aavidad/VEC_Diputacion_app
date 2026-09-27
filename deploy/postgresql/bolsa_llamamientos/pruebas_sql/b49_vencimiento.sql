\set ON_ERROR_STOP on
-- Fixture de fase diferida: el cese se publicó ayer y hoy es el primer día
-- local Europe/Madrid de disponibilidad. Inserción directa solo en base
-- desechable para representar el paso de medianoche sin esperar meses.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa(
 evento_ref,origen_ref,origen_huella_sha256,origen_posicion,candidato_ref,llamamiento_ref,
 relacion_ref,expediente_ref,organizacion_ref,modalidad_clave,causa_contrato_clave,
 clase_bolsa,fecha_efecto,disponible_desde,politica_version,fuente_tipo,fuente_ref,
 fuente_sha256,recibo_ct_ref,recibo_ref,recibida_en)
SELECT 'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||'origen:cese:b49:vence','UTF8')),'hex'),
 'origen:cese:b49:vence',r.origen_huella_sha256,4,r.candidato_ref,r.llamamiento_ref,
 r.relacion_ref,r.expediente_ref,r.organizacion_ref,r.modalidad_clave,r.causa_contrato_clave,
 r.clase_bolsa,r.fecha_efecto,(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date,r.politica_version,
 r.fuente_tipo,r.fuente_ref,r.fuente_sha256,r.recibo_ct_ref,
 'recibo:bolsa:cese:'||encode(sha256(convert_to('origen:cese:b49:vence','UTF8')),'hex'),
 clock_timestamp()-interval '1 day'
FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r WHERE r.origen_ref='origen:cese:b45:1';
INSERT INTO vec_bolsa_llamamientos.publicacion_cese_b10(
 evento_ref,fase,origen_ref,origen_posicion,ancla_manifiesto_sha256,confirmada_en,confirmada_por)
SELECT r.evento_ref,'cese',r.origen_ref,r.origen_posicion,repeat('f',64),
 clock_timestamp()-interval '1 day','vec_b49_publicador_test'
FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r WHERE r.origen_ref='origen:cese:b49:vence';
COMMIT;

SET SESSION AUTHORIZATION vec_b49_publicador_test;
DO $vencimiento$
DECLARE v record; a boolean;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>4 OR v.origen_ref<>'origen:cese:b49:vence'
    OR v.fase<>'vencimiento' OR v.bolsa_ref<>'bolsa:rev' THEN
  RAISE EXCEPTION 'B49: vencimiento local no apareció en feed';
 END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(
  4,'origen:cese:b49:vence','vencimiento',repeat('1',64));
 IF a IS DISTINCT FROM false OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()) THEN
  RAISE EXCEPTION 'B49: vencimiento no confirmó o feed no agotado';
 END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(
  4,'origen:cese:b49:vence','vencimiento',repeat('1',64));
 IF a IS DISTINCT FROM true THEN RAISE EXCEPTION 'B49: replay del vencimiento incorrecto'; END IF;
END $vencimiento$;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.publicacion_cese_b10)<>7
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)<>5 THEN
  RAISE EXCEPTION 'B49: historia de fases duplicada';
 END IF;
END $historia$;
