\set ON_ERROR_STOP on
-- Una persona en dos bolsas: la pausa B2 de la segunda termina después de
-- la restricción global B45. La primera conserva su propia situación.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:rev:b53',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rev',
 now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,
 instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES('instantanea:rev:b53',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:rev:b53',1,
 encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
 version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
VALUES('acta:rev:b53','bolsa:rev:b53',1,encode(sha256('{}'::bytea),'hex'),'instantanea:rev:b53',1,
 encode(sha256('{}'::bytea),'hex'),'categoria:rev','sistema:prueba',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES('instantanea:rev:b53',1,1,'participacion:rev:b53',1);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,
 registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:rev:b53','disponible',now()-interval '10 days','Constitución sintética','sistema:constitucion',
 now()-interval '10 days','constitucion:participacion:rev:b53','recibo:situacion:constitucion:participacion:rev:b53');
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(participacion_ref,candidato_ref,acta_ref,instantanea_ref,
 version_instantanea,registrada_en)
VALUES('participacion:rev:b53','can_bbbbbbbbbbbbbbbbbbbbbb','acta:rev:b53','instantanea:rev:b53',1,now());
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,fecha_disponible,motivo,
 actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:rev:b53','disponible_desde',now()-interval '1 hour',
 ((current_date-1+make_interval(months=>11))::date)::timestamp AT TIME ZONE 'Europe/Madrid',
 'Pausa B2 posterior al cese','sistema:prueba',now()-interval '1 hour','b53:pausa-larga','recibo:b53:pausa-larga');
COMMIT;

DO $recorrido$
DECLARE candidato text:='can_bbbbbbbbbbbbbbbbbbbbbb'; capacidad bytea; decision bytea; contexto bytea;
 j jsonb; fila jsonb; corte timestamptz; b2_hasta timestamptz; b2_desde timestamptz;
 cese_hasta timestamptz;
BEGIN
 capacidad:=convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||candidato,'huella_efecto_sha256',repeat('d',64))::text,'UTF8');
 decision:=convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||candidato,'contexto_recurso_huella_sha256',repeat('d',64))::text,'UTF8');
 contexto:=convert_to(jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',candidato)))::text,'UTF8');
 b2_hasta:=((current_date-1+make_interval(months=>11))::date)::timestamp AT TIME ZONE 'Europe/Madrid';
 cese_hasta:=((current_date-1+make_interval(months=>9))::date)::timestamp AT TIME ZONE 'Europe/Madrid';
 SELECT s.desde INTO STRICT b2_desde FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref='participacion:rev:b53' AND s.clave_idempotencia='b53:pausa-larga';

 corte:=(current_date-3)::timestamp AT TIME ZONE 'Europe/Madrid';
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,corte,capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 IF jsonb_array_length(j->'participaciones')<>2 OR EXISTS (
   SELECT 1 FROM jsonb_array_elements(j->'participaciones') p
   WHERE p #>> '{situacion_actual,estado}'<>'disponible'
      OR p #>> '{situacion_actual,fecha_disponible}' IS NOT NULL) THEN
  RAISE EXCEPTION 'B53: la lectura anterior al cese cambió';
 END IF;

 corte:=now();
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,corte,capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 SELECT p INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') p WHERE p->>'bolsa'='bolsa:rev:b53';
 IF fila #>> '{situacion_actual,estado}'<>'disponible_desde'
    OR (fila #>> '{situacion_actual,fecha_disponible}')::timestamptz<>b2_hasta
    OR (fila #>> '{situacion_actual,desde}')::timestamptz<>b2_desde THEN
  RAISE EXCEPTION 'B53: Mi Bolsa adelantó la pausa B2 posterior al cese';
 END IF;
 SELECT p INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') p WHERE p->>'bolsa'='bolsa:rev';
 IF fila #>> '{situacion_actual,estado}'<>'disponible_desde'
    OR (fila #>> '{situacion_actual,fecha_disponible}')::timestamptz<>cese_hasta THEN
  RAISE EXCEPTION 'B53: la otra bolsa perdió la fecha global de cese';
 END IF;

 corte:=cese_hasta+interval '1 day';
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,corte,capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 SELECT p INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') p WHERE p->>'bolsa'='bolsa:rev:b53';
 IF fila #>> '{situacion_actual,estado}'<>'disponible_desde'
    OR (fila #>> '{situacion_actual,fecha_disponible}')::timestamptz<>b2_hasta THEN
  RAISE EXCEPTION 'B53: Mi Bolsa saltó a disponible al vencer solo el cese';
 END IF;

 corte:=b2_hasta+interval '1 day';
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,corte,capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 SELECT p INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') p WHERE p->>'bolsa'='bolsa:rev:b53';
 IF fila #>> '{situacion_actual,estado}'<>'disponible'
    OR fila #>> '{situacion_actual,fecha_disponible}' IS NOT NULL
    OR (fila #>> '{situacion_actual,desde}')::timestamptz<b2_hasta THEN
  RAISE EXCEPTION 'B53: Mi Bolsa no recuperó disponibilidad tras vencer ambos plazos';
 END IF;
END $recorrido$;

-- Caso contrario: una pausa B2 posterior en historia pero de plazo más
-- corto no reduce la restricción global del cese. Se revierte la fila de
-- ensayo; la historia conservada de la prueba anterior permanece intacta.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,fecha_disponible,motivo,
 actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:rev:b53','disponible_desde',now()+interval '1 hour',
 ((current_date-1+make_interval(months=>3))::date)::timestamp AT TIME ZONE 'Europe/Madrid',
 'Pausa B2 corta sintética','sistema:prueba',now()+interval '1 hour','b53:pausa-corta','recibo:b53:pausa-corta');
DO $maximo$
DECLARE candidato text:='can_bbbbbbbbbbbbbbbbbbbbbb'; j jsonb; fila jsonb; corte timestamptz:=now()+interval '1 day';
 capacidad bytea; decision bytea; contexto bytea; cese_hasta timestamptz;
BEGIN
 capacidad:=convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||candidato,'huella_efecto_sha256',repeat('d',64))::text,'UTF8');
 decision:=convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||candidato,'contexto_recurso_huella_sha256',repeat('d',64))::text,'UTF8');
 contexto:=convert_to(jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',candidato)))::text,'UTF8');
 cese_hasta:=((current_date-1+make_interval(months=>9))::date)::timestamp AT TIME ZONE 'Europe/Madrid';
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,corte,capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 SELECT p INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') p WHERE p->>'bolsa'='bolsa:rev:b53';
 IF fila #>> '{situacion_actual,estado}'<>'disponible_desde'
    OR (fila #>> '{situacion_actual,fecha_disponible}')::timestamptz<>cese_hasta THEN
  RAISE EXCEPTION 'B53: pausa B2 corta redujo la restricción global';
 END IF;
END $maximo$;
ROLLBACK;
