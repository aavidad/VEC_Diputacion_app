\set ON_ERROR_STOP on
-- Ejecutar en PostgreSQL 18 desechable tras Bolsa 000045 y 000081, con el
-- verificador CT cerrado de probar_cese_pg18.sh. La prueba revierte sus datos.
-- No ejecutar sobre la principal ni reinstalar 000081 para repetirla.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '15s';
-- El doble B13/CT se siembra como propietario de la base; los triggers de
-- historia no deben actuar sobre filas preparadas a mano.
SET LOCAL session_replication_role = replica;
WITH datos(n,bolsa_ref,participacion_ref) AS (VALUES
 (1,'bolsa-sintetica:bbbbbbbbbbbbbbbb','participacion-sintetica-1:aaaaaaaaaaaaaaaa'),
 (2,'bolsa-sintetica:cccccccccccccccc','participacion-sintetica-2:aaaaaaaaaaaaaaaa')
), propuesta AS (
 SELECT n,bolsa_ref,participacion_ref,
  convert_to(jsonb_build_object('propuesta',jsonb_build_object('participacion_seleccionada_ref',participacion_ref))::text,'UTF8') AS canonica
 FROM datos
)
INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,
 registro_canonico,registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en)
SELECT 'operacion:propuesta:b81:'||n,'propuesta','necesidad:b81:'||n,1,'operacion:orden:b81:'||n,
 canonica,encode(sha256(canonica),'hex'),repeat('c',64),'decision:b81:'||n,'recibo:b81:'||n,now() FROM propuesta;
INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo(llamamiento_ref,operacion_ref,propuesta_ref,bolsa_ref,
 necesidad_ref,version,estado,abierto_en,datos_canonicos)
VALUES ('llamamiento:b81:1','operacion:propuesta:b81:1','propuesta:b81:1','bolsa-sintetica:bbbbbbbbbbbbbbbb',
 'necesidad:b81:1',1,'abierto',now(),'{}'::jsonb),
 ('llamamiento:b81:2','operacion:propuesta:b81:2','propuesta:b81:2','bolsa-sintetica:cccccccccccccccc',
 'necesidad:b81:2',1,'abierto',now(),'{}'::jsonb);
WITH datos(n,bolsa_ref,participacion_ref) AS (VALUES
 (1,'bolsa-sintetica:bbbbbbbbbbbbbbbb','participacion-sintetica-1:aaaaaaaaaaaaaaaa'),
 (2,'bolsa-sintetica:cccccccccccccccc','participacion-sintetica-2:aaaaaaaaaaaaaaaa')
), cuerpo AS (
 SELECT n,bolsa_ref,participacion_ref,'origen:cese:b81:'||n AS origen_ref,
 'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||'origen:cese:b81:'||n,'UTF8')),'hex') AS evento_ref
 FROM datos
), evento AS (
 SELECT *,jsonb_build_object('evento_ref',evento_ref,'tipo','cese','origen_ref',origen_ref,
  'organizacion_ref','organizacion:desarrollo:dipgra','expediente_ref','expediente:ct:1',
  'llamamiento_ref','llamamiento:b81:'||n) AS registro FROM cuerpo
)
INSERT INTO vec_bolsa_llamamientos.contrato_participacion(evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,
 origen_posicion,tipo,organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,ocurrido_en,recibido_en)
SELECT evento_ref,encode(sha256(convert_to(registro::text,'UTF8')),'hex'),registro,origen_ref,now(),80+n,'cese',
 'organizacion:desarrollo:dipgra','expediente:ct:1','llamamiento:b81:'||n,participacion_ref,bolsa_ref,now(),now() FROM evento;
INSERT INTO vec_contratacion_temporal.cese_prueba(origen_ref,huella,posicion,modalidad,causa,fecha,llamamiento,relacion)
SELECT origen_ref,huella_sha256,origen_posicion,'interinidad',NULL,current_date-1,llamamiento_ref,
 'relacion:b81:'||origen_posicion FROM vec_bolsa_llamamientos.contrato_participacion
WHERE origen_ref IN ('origen:cese:b81:1','origen:cese:b81:2');
-- La segunda bolsa sí está constituida. Aunque no tenga vínculo de candidato,
-- B81 debe conservar su cese pendiente.
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa-sintetica:cccccccccccccccc',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,
 'categoria:b81',now()-interval '1 day',NULL,'vigente',now()-interval '1 day');
SET LOCAL session_replication_role = origin;
SET SESSION AUTHORIZATION vec_b45_relevo_test;
DO $b81$
DECLARE h1 text; h2 text; repetida boolean; cursor_b81 record;
BEGIN
 h1:=public.huella_cese_b45('origen:cese:b81:1');
 h2:=public.huella_cese_b45('origen:cese:b81:2');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1('origen:cese:b81:1',h1,81);
  RAISE EXCEPTION 'B81: restricción aceptó una participación sin candidato';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_ajeno_bolsa_v1('origen:cese:b81:1',h1,81);
  RAISE EXCEPTION 'B81: el llamamiento Bolsa se marcó como ajeno';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 SELECT vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1('origen:cese:b81:1',h1,81) INTO repetida;
 IF repetida THEN RAISE EXCEPTION 'B81: primer registro devuelto como replay'; END IF;
 SELECT vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1('origen:cese:b81:1',h1,81) INTO repetida;
 IF NOT repetida THEN RAISE EXCEPTION 'B81: replay duplicado'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1('origen:cese:b81:1',repeat('f',64),81);
  RAISE EXCEPTION 'B81: huella divergente aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1('origen:cese:b81:2',h2,82);
  RAISE EXCEPTION 'B81: bolsa constituida sin vínculo tratada como sin candidato';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 SELECT * INTO STRICT cursor_b81 FROM vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1();
 IF cursor_b81.origen_posicion<>81 OR cursor_b81.origen_ref<>'origen:cese:b81:1' THEN
  RAISE EXCEPTION 'B81: cursor no avanzó exactamente hasta el cese acreditado';
 END IF;
END $b81$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $invariantes$
DECLARE registro_b81 record;
BEGIN
 SELECT * INTO STRICT registro_b81 FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa
 WHERE origen_ref='origen:cese:b81:1';
 IF registro_b81.motivo<>'participacion_no_constituida'
    OR registro_b81.registro_sha256<>encode(sha256(convert_to(registro_b81.registro::text,'UTF8')),'hex')
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa WHERE origen_ref LIKE 'origen:cese:b81:%')<>1
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa WHERE origen_ref LIKE 'origen:cese:b81:%')
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa WHERE origen_ref LIKE 'origen:cese:b81:%') THEN
  RAISE EXCEPTION 'B81: historia, huella o ausencia de efecto divergente';
 END IF;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
   ('bolsa-sintetica:bbbbbbbbbbbbbbbb',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,
    'categoria:b81',now(),NULL,'vigente',now());
  RAISE EXCEPTION 'B81: la bolsa sintética se constituyó tras el cese';
 EXCEPTION WHEN foreign_key_violation THEN
  IF SQLERRM<>'el puente sintético no constituye bolsas' THEN RAISE; END IF;
 END;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
   (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
  VALUES ('instantanea:b81:ausente',1,1,'participacion-sintetica-1:aaaaaaaaaaaaaaaa',1);
  RAISE EXCEPTION 'B81: la participación sintética se constituyó tras el cese';
 EXCEPTION WHEN foreign_key_violation THEN
  IF SQLERRM<>'el puente sintético no constituye participaciones' THEN RAISE; END IF;
 END;
END $invariantes$;
ROLLBACK;
