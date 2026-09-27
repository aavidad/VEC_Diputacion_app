\set ON_ERROR_STOP on
-- Orígenes sintéticos, dos candidatos en una bolsa y el primero también en
-- otra bolsa. El doble CT solo devuelve estos orígenes exactos.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:rev:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rev',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES('instantanea:rev:2',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:rev:2',1,encode(sha256('{}'::bytea),'hex'),1,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
VALUES('acta:rev:2','bolsa:rev:2',1,encode(sha256('{}'::bytea),'hex'),'instantanea:rev:2',1,encode(sha256('{}'::bytea),'hex'),'categoria:rev','sistema:prueba',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES('instantanea:rev:2',1,1,'participacion:rev:segunda',1);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:rev:segunda','disponible',now()-interval '10 days','Constitución de bolsa','sistema:constitucion',now()-interval '10 days','constitucion:participacion:rev:segunda','recibo:situacion:constitucion:participacion:rev:segunda');
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES('politica:rev:2','bolsa:rev:2',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Prueba segunda bolsa','sistema:prueba',now()-interval '10 days',NULL,now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
VALUES ('participacion:rev:1','can_aaaaaaaaaaaaaaaaaaaaaa','acta:rev','instantanea:rev',1,now()),
       ('participacion:rev:2','can_bbbbbbbbbbbbbbbbbbbbbb','acta:rev','instantanea:rev',1,now()),
       ('participacion:rev:segunda','can_aaaaaaaaaaaaaaaaaaaaaa','acta:rev:2','instantanea:rev:2',1,now());
INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,
 registro_canonico,registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en)
SELECT 'operacion:propuesta:b45:'||n,'propuesta','necesidad:b45:'||n,1,'operacion:orden:b45:'||n,
 convert_to(jsonb_build_object('propuesta',jsonb_build_object('participacion_seleccionada_ref','participacion:rev:'||n))::text,'UTF8'),
 encode(sha256(convert_to(jsonb_build_object('propuesta',jsonb_build_object('participacion_seleccionada_ref','participacion:rev:'||n))::text,'UTF8')),'hex'),
 repeat('c',64),'decision:b45:'||n,'recibo:b45:'||n,now()
FROM generate_series(1,2) n;
INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo(llamamiento_ref,operacion_ref,propuesta_ref,bolsa_ref,
 necesidad_ref,version,estado,abierto_en,datos_canonicos)
SELECT 'llamamiento:b45:'||n,'operacion:propuesta:b45:'||n,'propuesta:b45:'||n,'bolsa:rev',
 'necesidad:b45:'||n,1,'abierto',now(),'{}'::jsonb FROM generate_series(1,2) n;
INSERT INTO vec_contratacion_temporal.cese_prueba(origen_ref,huella,posicion,modalidad,causa,fecha,llamamiento,relacion)
VALUES ('origen:cese:b45:1',repeat('a',64),1,'interinidad',NULL,current_date-1,'llamamiento:b45:1','relacion:b45:1'),
       ('origen:cese:b45:2',repeat('b',64),2,'interinidad','acumulacion_tareas',current_date-1,'llamamiento:b45:2','relacion:b45:2'),
       ('origen:cese:b45:3',repeat('c',64),3,'interinidad',NULL,current_date-1,'llamamiento:b45:2','relacion:b45:2');
COMMIT;

-- La ventana usa la fecha del cese; PostgreSQL ajusta el día final si el
-- mes de destino no lo contiene.
DO $calendario$ BEGIN
 IF ('2026-01-31'::date+make_interval(months=>5))::date<>'2026-06-30'::date
    OR ('2024-05-31'::date+make_interval(months=>9))::date<>'2025-02-28'::date THEN
  RAISE EXCEPTION 'B45: ajuste fin de mes incorrecto';
 END IF;
END $calendario$;

SET SESSION AUTHORIZATION vec_b45_relevo_test;
DO $prueba$
DECLARE a record; b record; repetida record;
BEGIN
 SELECT * INTO STRICT a FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:1',repeat('a',64),1);
 SELECT * INTO STRICT b FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:2',repeat('b',64),2);
 SELECT * INTO STRICT repetida FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:1',repeat('a',64),1);
 IF a.reutilizada OR b.reutilizada OR NOT repetida.reutilizada OR a.recibo_ref<>repetida.recibo_ref
    OR a.disponible_desde<>(current_date-1+make_interval(months=>5))::date
    OR b.disponible_desde<>(current_date-1+make_interval(months=>9))::date
    OR a.politica_version<>1 OR b.politica_version<>1 THEN
  RAISE EXCEPTION 'B45: +5/+9 o replay incorrecto';
 END IF;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:1',repeat('f',64),1);
  RAISE EXCEPTION 'B45: origen falso aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $prueba$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b45_ejecutor_test;
DO $politica$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(
  'catalogo:bolsa:cese:ejemplo-sintetico:v2','{"relevo":"general"}'::jsonb,4,8,'ejemplo_sintetico');
 IF v.version<>2 OR v.reutilizada THEN RAISE EXCEPTION 'B45: publicación v2 incorrecta'; END IF;
END $politica$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b45_relevo_test;
DO $mapeo$
DECLARE v record;
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:3',repeat('c',64),3);
  RAISE EXCEPTION 'B45: modalidad no mapeada aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $mapeo$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b45_ejecutor_test;
DO $politica$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(
  'catalogo:bolsa:cese:ejemplo-sintetico:v3','{"interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas"}'::jsonb,4,8,'ejemplo_sintetico');
 IF v.version<>3 OR v.reutilizada THEN RAISE EXCEPTION 'B45: publicación v3 incorrecta'; END IF;
END $politica$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_b45_relevo_test;
DO $historia$
DECLARE nueva record; previa record;
BEGIN
 SELECT * INTO STRICT nueva FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:3',repeat('c',64),3);
 SELECT * INTO STRICT previa FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b45:1',repeat('a',64),1);
 IF nueva.reutilizada OR nueva.politica_version<>3 OR nueva.disponible_desde<>(current_date-1+make_interval(months=>4))::date
    OR NOT previa.reutilizada OR previa.politica_version<>1 OR previa.disponible_desde<>(current_date-1+make_interval(months=>5))::date THEN
  RAISE EXCEPTION 'B45: nueva política reescribió recibo anterior';
 END IF;
END $historia$;
RESET SESSION AUTHORIZATION;
DO $orden$
DECLARE p1 record; p2 record; n integer;
BEGIN
 SELECT * INTO STRICT p1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:rev',now())
  WHERE participacion_ref='participacion:rev:1';
 SELECT * INTO STRICT p2 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:rev',now())
  WHERE participacion_ref='participacion:rev:2';
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:rev:2',now())
             WHERE participacion_ref='participacion:rev:segunda' AND (orden_vigente IS NOT NULL OR razon<>'restriccion_cese')) THEN
  RAISE EXCEPTION 'B45: la restricción no alcanza la segunda bolsa';
 END IF;
 IF p1.orden_vigente IS NOT NULL OR p2.orden_vigente IS NOT NULL
    OR p1.razon<>'restriccion_cese' OR p2.razon<>'restriccion_cese'
    OR p1.situacion<>'disponible_desde' OR p2.situacion<>'disponible_desde' THEN
  RAISE EXCEPTION 'B45: la disponibilidad no cambió en el orden';
 END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1('participacion:rev:1',now()))<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1('participacion:rev:segunda',now()))<>1
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1('participacion:rev:1',now()+interval '1 year')) THEN
  RAISE EXCEPTION 'B45: lector de fecha global incompatible';
 END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.restriccion_cese_bolsa;
 IF n<>3 OR (SELECT count(*) FROM vec_bolsa_llamamientos.auditoria_cese_bolsa)<>3
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref IN
  ('participacion:rev:1','participacion:rev:2'))<>2 THEN
  RAISE EXCEPTION 'B45: cese duplicado o situación por bolsa copiada';
 END IF;
END $orden$;
DO $mi_bolsa$
DECLARE j jsonb; futuro jsonb; candidato text:='can_aaaaaaaaaaaaaaaaaaaaaa'; capacidad bytea; decision bytea; contexto bytea;
BEGIN
 capacidad:=convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||candidato,'huella_efecto_sha256',repeat('d',64))::text,'UTF8');
 decision:=convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||candidato,'contexto_recurso_huella_sha256',repeat('d',64))::text,'UTF8');
 contexto:=convert_to(jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',candidato)))::text,'UTF8');
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,now(),capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT j;
 IF jsonb_array_length(j->'participaciones')<>2 OR EXISTS
   (SELECT 1 FROM jsonb_array_elements(j->'participaciones') p
     WHERE p #>> '{situacion_actual,estado}'<>'disponible_desde'
        OR p #>> '{situacion_actual,fecha_disponible}' IS NULL) THEN
  RAISE EXCEPTION 'B45: Mi Bolsa no proyecta las dos participaciones restringidas';
 END IF;
 SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,now()+interval '1 year',capacidad,decision,'{}'::bytea,contexto,
  1,1,'{}'::bytea,'{}'::bytea,'{}'::bytea,'{}'::bytea) INTO STRICT futuro;
 IF EXISTS (SELECT 1 FROM jsonb_array_elements(futuro->'participaciones') p
            WHERE p #>> '{situacion_actual,estado}'<>'disponible'
               OR p #>> '{situacion_actual,fecha_disponible}' IS NOT NULL) THEN
  RAISE EXCEPTION 'B45: Mi Bolsa no recupera disponibilidad al vencer';
 END IF;
END $mi_bolsa$;
DO $acl$
BEGIN
 IF has_table_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.restriccion_cese_bolsa','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint)','EXECUTE') THEN
  RAISE EXCEPTION 'B45: ACL abierta o cerrada incorrectamente';
 END IF;
END $acl$;
