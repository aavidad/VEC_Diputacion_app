-- Pruebas de la migración 000025 (correo personalizado de B7). Se ejecutan como
-- superusuario sobre una base con 000025 aplicada y no dejan filas: todo ocurre
-- dentro de una transacción que termina en ROLLBACK.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
VALUES('llamamiento:'||repeat('a',64),'recibo:llamamiento:'||repeat('a',64),'bolsa:prueba-b25','per_'||repeat('A',22),'clave-b25-0001','["part:1","part:2"]','{"plantilla_version":"bolsa-llamamiento-v2"}',repeat('b',64),sha256('\x0101010101010101010101010101010101010101010101010101010101010101'::bytea),'emision_reservada',now(),'decision:b25');
RESET ROLE;
DO $t$
DECLARE
 tok bytea := '\x0101010101010101010101010101010101010101010101010101010101010101';
 bueno jsonb := jsonb_build_array(
   jsonb_build_object('participacion_ref','part:1','huella_asunto_sha256',repeat('1',64),'huella_cuerpo_sha256',repeat('2',64),'caracteres',120),
   jsonb_build_object('participacion_ref','part:2','huella_asunto_sha256',repeat('3',64),'huella_cuerpo_sha256',repeat('4',64),'caracteres',121));
 actor text := 'per_'||repeat('A',22);
 caso record;
 estado text;
BEGIN
 EXECUTE 'SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor';
 PERFORM vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1('bolsa:prueba-b25','clave-b25-0001',actor,tok,bueno);
 PERFORM vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1('bolsa:prueba-b25','clave-b25-0001',actor,tok,bueno);
 FOR caso IN SELECT * FROM (VALUES
   ('divergente','VBE01',jsonb_set(bueno,'{1,huella_cuerpo_sha256}',to_jsonb(repeat('5',64))),tok,actor),
   ('token ajeno','42501',bueno,'\x02'::bytea||substr(tok,2),actor),
   ('actor ajeno','42501',bueno,tok,'per_'||repeat('B',22)),
   ('orden cambiado','42501',jsonb_build_array(bueno->1,bueno->0),tok,actor),
   ('cardinalidad','42501',jsonb_build_array(bueno->0),tok,actor),
   ('clave extra','42501',jsonb_set(bueno,'{0,extra}','1'),tok,actor),
   ('huella corta','42501',jsonb_set(bueno,'{0,huella_asunto_sha256}','"abc"'),tok,actor),
   ('caracteres texto','42501',jsonb_set(bueno,'{0,caracteres}','"120"'),tok,actor),
   ('caracteres cero','42501',jsonb_set(bueno,'{0,caracteres}','0'),tok,actor),
   ('caracteres excesivos','42501',jsonb_set(bueno,'{0,caracteres}','20001'),tok,actor),
   ('no array','22023','{}'::jsonb,tok,actor),
   ('vacío','22023','[]'::jsonb,tok,actor)
 ) v(nombre,esperado,cuerpos,token,act) LOOP
   BEGIN
     PERFORM vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1('bolsa:prueba-b25','clave-b25-0001',caso.act,caso.token,caso.cuerpos);
     RAISE EXCEPTION 'FALLO %: aceptado', caso.nombre;
   EXCEPTION WHEN OTHERS THEN
     GET STACKED DIAGNOSTICS estado = RETURNED_SQLSTATE;
     IF estado<>caso.esperado THEN RAISE EXCEPTION 'FALLO %: sqlstate % (esperado %) %', caso.nombre, estado, caso.esperado, SQLERRM; END IF;
     RAISE NOTICE 'OK negativo %', caso.nombre;
   END;
 END LOOP;
 BEGIN
   PERFORM vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1('bolsa:prueba-b25','clave-inexistente',actor,tok,bueno);
   RAISE EXCEPTION 'FALLO reserva inexistente aceptada';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK negativo reserva inexistente'; END;
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto;
   RAISE EXCEPTION 'FALLO el ejecutor lee la tabla';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK ejecutor sin acceso directo'; END;
 EXECUTE 'RESET ROLE';
END $t$;
DO $t$
DECLARE n int; BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto WHERE llamamiento_ref='llamamiento:'||repeat('a',64);
 IF n<>2 THEN RAISE EXCEPTION 'FALLO filas registradas: %', n; END IF;
 RAISE NOTICE 'OK dos huellas, la repetición idéntica no duplica';
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.cuerpo_llamamiento_contacto'::regclass) THEN RAISE EXCEPTION 'FALLO RLS'; END IF;
 IF has_function_privilege('public','vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1(text,text,text,bytea,jsonb)','EXECUTE') THEN RAISE EXCEPTION 'FALLO EXECUTE público'; END IF;
 RAISE NOTICE 'OK RLS forzada y sin EXECUTE público';
 EXECUTE 'SET LOCAL ROLE vec_bolsa_llamamientos_propietario';
 BEGIN
   UPDATE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto SET caracteres=1;
   RAISE EXCEPTION 'FALLO huella mutable';
 EXCEPTION WHEN OTHERS THEN IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF; RAISE NOTICE 'OK huellas inmutables'; END;
 EXECUTE 'RESET ROLE';
END $t$;
-- La reserva admite versiones bolsa-llamamiento-vN y rechaza cualquier otra
-- antes de consumir la autorización.
DO $t$
DECLARE cfg jsonb := '{"referencia":"RF","descripcion":"DS","categoria":"CT","centro":"CE","modalidad":"MO","fecha_inicio":"2026-10-01","plazo":"24 horas","plantilla_version":"X","asunto":"AS","cuerpo":"CU"}';
 version text; mensaje text;
BEGIN
 EXECUTE 'SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor';
 FOREACH version IN ARRAY ARRAY['bolsa-llamamiento-x','bolsa-llamamiento-v0','otra-v2','bolsa-llamamiento-v12345','bolsa-llamamiento-v1','bolsa-llamamiento-v2'] LOOP
   BEGIN
     PERFORM vec_bolsa_llamamientos.reservar_llamamiento_v1('llamamiento:'||repeat('c',64),'recibo:llamamiento:'||repeat('c',64),'bolsa:prueba-b25','per_'||repeat('A',22),'clave-b25-0002','["part:1"]',jsonb_set(cfg,'{plantilla_version}',to_jsonb(version)),now(),'\x0101010101010101010101010101010101010101010101010101010101010101','\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
     RAISE EXCEPTION 'FALLO reserva aceptada con autorización vacía';
   EXCEPTION WHEN OTHERS THEN
     mensaje := SQLERRM;
     IF mensaje LIKE 'FALLO%' THEN RAISE; END IF;
     IF version IN ('bolsa-llamamiento-v1','bolsa-llamamiento-v2') AND mensaje='emisión B7 inválida' THEN RAISE EXCEPTION 'FALLO % rechazada por formato', version; END IF;
     IF version NOT IN ('bolsa-llamamiento-v1','bolsa-llamamiento-v2') AND mensaje<>'emisión B7 inválida' THEN RAISE EXCEPTION 'FALLO % no rechazada por formato: %', version, mensaje; END IF;
     RAISE NOTICE 'OK versión % -> %', version, mensaje;
   END;
 END LOOP;
 EXECUTE 'RESET ROLE';
END $t$;
ROLLBACK;
