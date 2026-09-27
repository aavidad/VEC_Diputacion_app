\set ON_ERROR_STOP on
-- Ejecutar tras la fixture B45 y Bolsa 000049 UP en una base PostgreSQL 18
-- desechable. Este fichero confirma el primer cese y COMMIT; el siguiente
-- verifica recuperación en otra conexión.
CREATE ROLE vec_b49_publicador_test LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_b49_mixto_test LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_llamamientos_publicador_cese TO vec_b49_publicador_test,vec_b49_mixto_test;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b49_mixto_test;

DO $acl$
BEGIN
 IF has_table_privilege('vec_b49_publicador_test','vec_bolsa_llamamientos.restriccion_cese_bolsa','SELECT')
    OR has_table_privilege('vec_b49_publicador_test','vec_bolsa_llamamientos.publicacion_cese_b10','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_relevo_cese','vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(bigint,text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_b49_publicador_test','vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()','EXECUTE') THEN
  RAISE EXCEPTION 'B49: ACL abierta o cerrada incorrectamente';
 END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_b49_mixto_test;
DO $mixto$
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
  RAISE EXCEPTION 'B49: rol mixto leyó el feed';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $mixto$;
SET ROLE vec_bolsa_llamamientos_publicador_cese;
DO $mixto_set_role$
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
  RAISE EXCEPTION 'B49: SET ROLE ocultó membresía mixta';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $mixto_set_role$;
RESET ROLE;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_b49_publicador_test;
DO $primero$
DECLARE v record; a boolean;
BEGIN
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>1 OR v.origen_ref<>'origen:cese:b45:1' OR v.fase<>'cese'
    OR v.bolsa_ref<>'bolsa:rev' OR v.evento_ref !~ '^evento:ct:contrato-bolsa:[a-f0-9]{64}$' THEN
  RAISE EXCEPTION 'B49: primer cese o pista de bolsa incorrecto';
 END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(2,'origen:cese:b45:2','cese',repeat('b',64));
  RAISE EXCEPTION 'B49: salto del primer pendiente';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(1,'origen:cese:b45:1','cese',repeat('0',64));
  RAISE EXCEPTION 'B49: ancla nula aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(1,'origen:cese:b45:1','cese',repeat('a',64));
 IF a IS DISTINCT FROM false THEN RAISE EXCEPTION 'B49: alta incorrecta'; END IF;
 a:=vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(1,'origen:cese:b45:1','cese',repeat('a',64));
 IF a IS DISTINCT FROM true THEN RAISE EXCEPTION 'B49: replay incorrecto'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(1,'origen:cese:b45:1','cese',repeat('b',64));
  RAISE EXCEPTION 'B49: ancla divergente aceptada';
 EXCEPTION WHEN SQLSTATE 'VBP01' THEN NULL; END;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF v.origen_posicion<>2 OR v.fase<>'cese' THEN RAISE EXCEPTION 'B49: siguiente pendiente incorrecto'; END IF;
END $primero$;
RESET SESSION AUTHORIZATION;

-- Simula que un cese confirmado por B45 llegó después con posición menor.
-- El feed por fila debe recuperarlo aunque el primer cese ya tenga ACK.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa(
 evento_ref,origen_ref,origen_huella_sha256,origen_posicion,candidato_ref,llamamiento_ref,
 relacion_ref,expediente_ref,organizacion_ref,modalidad_clave,causa_contrato_clave,
 clase_bolsa,fecha_efecto,disponible_desde,politica_version,fuente_tipo,fuente_ref,
 fuente_sha256,recibo_ct_ref,recibo_ref,recibida_en)
SELECT 'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||'origen:cese:b49:tardio','UTF8')),'hex'),
 'origen:cese:b49:tardio',r.origen_huella_sha256,0,r.candidato_ref,r.llamamiento_ref,
 r.relacion_ref,r.expediente_ref,r.organizacion_ref,r.modalidad_clave,r.causa_contrato_clave,
 r.clase_bolsa,r.fecha_efecto,(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date,r.politica_version,r.fuente_tipo,r.fuente_ref,
 r.fuente_sha256,r.recibo_ct_ref,'recibo:bolsa:cese:'||encode(sha256(convert_to('origen:cese:b49:tardio','UTF8')),'hex'),clock_timestamp()
FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r WHERE r.origen_ref='origen:cese:b45:1';
COMMIT;
