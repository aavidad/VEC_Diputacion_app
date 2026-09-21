-- Ejecutar solo sobre PostgreSQL 18 efímero con 000001 y 000002 instaladas.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $contrato$
DECLARE a text[]; b text[]; v2 boolean; v3 boolean;
BEGIN
 SELECT array_agg(column_name ORDER BY ordinal_position) INTO a FROM information_schema.columns WHERE table_schema='vec_bolsa_publica_lectura' AND table_name='bolsas_v1';
 SELECT array_agg(column_name ORDER BY ordinal_position) INTO b FROM information_schema.columns WHERE table_schema='vec_bolsa_publica_lectura' AND table_name='posiciones_bolsa_v1';
 SELECT has_function_privilege('vec_bolsa_publica_publicador_login','vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)','EXECUTE'),has_function_privilege('vec_bolsa_publica_publicador_login','vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text)','EXECUTE') INTO v2,v3;
 IF a <> ARRAY['bolsa_ref','categoria','categoria_clave','grupos','tipo_lista','vigente_desde','vigente_hasta','total'] OR b <> ARRAY['bolsa_ref','orden','documento_enmascarado','estado_clave'] OR v2 OR NOT v3 OR NOT has_table_privilege('vec_bolsa_publica_consulta','vec_bolsa_publica_lectura.bolsas_v1','SELECT') OR has_table_privilege('vec_bolsa_publica_consulta','vec_bolsa_publica_datos.bolsa_publica','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'contrato o ACL B10 inesperado'; END IF;
END $contrato$;
COMMIT;
SELECT set_config('vec.prueba_v2',$j${"fuente":{"revision":"b10-prueba","actualizada_en":"2026-09-20T10:00:00Z"},"catalogos":[{"referencia":"tipos","version":1,"entradas":[{"clave":"bolsa","etiqueta":"Bolsa","descripcion":"","semantica":"informacion","orden":1}]}],"categorias":{"actual":{"catalogo_id":"categorias-profesionales","catalogo_version":1,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","catalogo_huella_proyeccion_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"snapshots":[{"catalogo_id":"categorias-profesionales","version":1,"huella_gobernada_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","huella_proyeccion_publica_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","categorias":[{"clave":"administrativo","etiqueta":"Administrativo","descripcion":"","semantica":"informacion","orden":1,"area":"administracion","area_etiqueta":"Administración","suscribible":true,"vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":null}]}]},"convocatorias":[]}$j$,false);
SELECT set_config('vec.prueba_b10',$j${"generado_en":"2026-09-20T10:00:00Z","bolsas":[{"bolsa_ref":"bolsa:administrativo:2026","categoria":"Administrativo","categoria_clave":"administrativo","grupos":["C1"],"tipo_lista":"definitiva","vigente_desde":"2026-09-20T10:00:00Z","vigente_hasta":null,"total":2,"posiciones":[{"orden":1,"documento_enmascarado":"***1234**","estado_clave":"disponible"},{"orden":2,"documento_enmascarado":"***1234**","estado_clave":"ocupado"}]}]}$j$,false);
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v3(current_setting('vec.prueba_v2')::jsonb,current_setting('vec.prueba_b10')::jsonb,repeat('a',64));
DO $redaccion$
DECLARE mensaje text;
BEGIN
 BEGIN
  PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
   current_setting('vec.prueba_v2')::jsonb,
   jsonb_set(current_setting('vec.prueba_b10')::jsonb,'{bolsas,0,vigente_desde}',to_jsonb('correo-sintetico-b10@example.invalid'::text)),
   repeat('b',64));
  RAISE EXCEPTION 'V3 acepto una fecha B10 invalida';
 EXCEPTION WHEN SQLSTATE '22023' THEN
  GET STACKED DIAGNOSTICS mensaje = MESSAGE_TEXT;
  IF mensaje <> 'publicacion rechazada: contenido invalido' OR position('correo-sintetico-b10@example.invalid' IN mensaje) > 0 THEN RAISE EXCEPTION 'V3 filtro el valor invalido: %',mensaje; END IF;
 END;
END $redaccion$;
RESET SESSION AUTHORIZATION;
BEGIN;
SET LOCAL ROLE vec_bolsa_publica_propietario;
DO $publicacion$
BEGIN
 IF (SELECT total FROM vec_bolsa_publica_lectura.bolsas_v1 WHERE bolsa_ref='bolsa:administrativo:2026')<>2 OR (SELECT count(*) FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1 WHERE bolsa_ref='bolsa:administrativo:2026')<>2 OR (SELECT manifiesto_sha256 FROM vec_bolsa_publica_lectura.fuente_publica_v2 WHERE control_id)<>repeat('a',64) THEN RAISE EXCEPTION 'V3 no publico B10 junto al ancla: total %, posiciones %, ancla %',(SELECT total FROM vec_bolsa_publica_lectura.bolsas_v1 WHERE bolsa_ref='bolsa:administrativo:2026'),(SELECT count(*) FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1 WHERE bolsa_ref='bolsa:administrativo:2026'),(SELECT manifiesto_sha256 FROM vec_bolsa_publica_lectura.fuente_publica_v2 WHERE control_id); END IF;
END $publicacion$;
RESET ROLE;
SET LOCAL ROLE vec_bolsa_publica_consulta;
DO $lectura$
BEGIN
 IF (SELECT count(*) FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1 WHERE bolsa_ref='bolsa:administrativo:2026')<>2 THEN RAISE EXCEPTION 'lector B10 no ve la proyeccion por la vista'; END IF;
END $lectura$;
RESET ROLE;
SET LOCAL ROLE vec_bolsa_publica_propietario;
UPDATE vec_bolsa_publica_datos.bolsa_publica SET categoria='Administrativo actualizado' WHERE bolsa_ref='bolsa:administrativo:2026';
RESET ROLE;
DO $invalida$
BEGIN
 IF (SELECT manifiesto_sha256 FROM vec_bolsa_publica_datos.fuente WHERE control_id)<>repeat('0',64) THEN RAISE EXCEPTION 'DML lateral B10 no invalido el ancla'; END IF;
END $invalida$;
ROLLBACK;
