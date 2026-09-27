\set ON_ERROR_STOP on
DO $acl$
BEGIN
 IF has_table_privilege('vec_b52_publicador_test','vec_bolsa_llamamientos.captura_cese_b10','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_b52_publicador_test','vec_bolsa_llamamientos.material_cese_b10','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.capturar_cese_b10_v1(bigint,text,text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_b52_publicador_test',
      'vec_bolsa_llamamientos.capturar_cese_b10_v1(bigint,text,text,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'Bolsa52: ACL abierta';
 END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_b52_publicador_test;
BEGIN ISOLATION LEVEL REPEATABLE READ;
DO $captura$
DECLARE v record; evento text;
BEGIN
 SELECT r.evento_ref INTO STRICT evento FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1() r;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.capturar_cese_b10_v1(
   2,'origen:cese:b45:2',evento,'bolsa:rev','cese');
  RAISE EXCEPTION 'Bolsa52: cursor adelantado aceptado';
 EXCEPTION WHEN foreign_key_violation OR object_not_in_prerequisite_state THEN NULL; END;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.capturar_cese_b10_v1(
  1,'origen:cese:b45:1',evento,'bolsa:rev','cese');
 IF jsonb_array_length(v.filas)<>5
    OR (SELECT count(*) FROM jsonb_array_elements(v.filas) f WHERE f->>'es_origen'='true')<>1
    OR (SELECT count(*) FROM jsonb_array_elements(v.filas) f WHERE f->>'estado_efectivo'='disponible_desde')<>2
    OR (SELECT count(DISTINCT f->>'bolsa_ref') FROM jsonb_array_elements(v.filas) f)<>2
    OR v.proyeccion_v2 IS NOT NULL OR v.manifiesto_v2 IS NOT NULL
    OR v.bolsas_v1 IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa52: captura incompleta';
 END IF;
 IF EXISTS (SELECT 1 FROM jsonb_array_elements(v.filas) f,
     jsonb_object_keys(f) k WHERE k NOT IN
      ('bolsa_ref','categoria_ref','acta_ref','vigente_desde','vigente_hasta','total',
       'tipo_lista','participacion_ref','fila_numero','orden','estado_efectivo','es_origen','fecha_disponible')) THEN
  RAISE EXCEPTION 'Bolsa52: datos no minimizados';
 END IF;
END $captura$;
COMMIT;
-- Un cambio posterior en Bolsa no puede reescribir el corte ya capturado.
RESET SESSION AUTHORIZATION;
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
 participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:rev:3','no_disponible',clock_timestamp(),
 'Cambio posterior al corte','sistema:prueba',clock_timestamp(),
 'b52:despues:3','recibo:b52:despues:3');
COMMIT;
SET SESSION AUTHORIZATION vec_b52_publicador_test;
BEGIN ISOLATION LEVEL REPEATABLE READ;
DO $replay$
DECLARE v record; evento text;
BEGIN
 SELECT r.evento_ref INTO STRICT evento FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1() r;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.capturar_cese_b10_v1(
  1,'origen:cese:b45:1',evento,'bolsa:rev','cese');
 IF jsonb_array_length(v.filas)<>5 OR (SELECT count(*) FROM jsonb_array_elements(v.filas) f WHERE f->>'es_origen'='true')<>1
    OR (SELECT f->>'estado_efectivo' FROM jsonb_array_elements(v.filas) f WHERE f->>'participacion_ref'='participacion:rev:3')<>'disponible' THEN
  RAISE EXCEPTION 'Bolsa52: replay divergente';
 END IF;
END $replay$;
DO $material$
DECLARE v record; evento text; bolsas jsonb; proyeccion bytea; manifiesto bytea;
 corte_texto text; nueva boolean;
BEGIN
 SELECT r.evento_ref INTO STRICT evento FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1() r;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.capturar_cese_b10_v1(
  1,'origen:cese:b45:1',evento,'bolsa:rev','cese');
 corte_texto:=to_char(v.corte AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 SELECT jsonb_build_object('generado_en',corte_texto,'bolsas',jsonb_agg(
  jsonb_build_object('bolsa_ref',b.bolsa_ref,'categoria','Sintética',
   'categoria_clave','categoria','grupos',jsonb_build_array('C1'),
   'tipo_lista',b.tipo_lista,'vigente_desde',b.vigente_desde,
   'vigente_hasta',b.vigente_hasta,'total',b.total,'posiciones',b.posiciones)
  ORDER BY b.bolsa_ref)) INTO bolsas FROM (
   SELECT f->>'bolsa_ref' bolsa_ref,f->>'tipo_lista' tipo_lista,
    (f->>'total')::bigint total,min(f->>'vigente_desde') vigente_desde,
    min(f->>'vigente_hasta') vigente_hasta,
    jsonb_agg(jsonb_build_object('orden',(f->>'orden')::integer,
      'documento_enmascarado','***1234**','estado_clave',CASE f->>'estado_efectivo'
       WHEN 'disponible' THEN 'disponible' WHEN 'trabajando' THEN 'ocupado'
       WHEN 'pendiente_incorporacion' THEN 'ocupado' WHEN 'disponible_desde' THEN 'no_disponible'
       WHEN 'no_disponible' THEN 'no_disponible' WHEN 'excluido' THEN 'excluido'
       WHEN 'renuncia' THEN 'renuncia_pendiente' END)
      ORDER BY (f->>'orden')::integer) posiciones
   FROM jsonb_array_elements(v.filas) f GROUP BY f->>'bolsa_ref',f->>'tipo_lista',f->>'total'
  ) b;
 proyeccion:=convert_to(jsonb_build_object('fuente',jsonb_build_object('actualizada_en',corte_texto))::text,'UTF8');
 manifiesto:='{}'::bytea;
 -- El doble V2 solo ensaya el slot SQL: la aplicación exige además el
 -- manifiesto canónico V2 del proveedor gobernado antes de invocarlo.
 BEGIN
  PERFORM vec_bolsa_llamamientos.guardar_material_cese_b10_v1(evento,'cese',v.corte,NULL,
   proyeccion,manifiesto,convert_to(
    jsonb_set(bolsas,'{bolsas,0,posiciones,0,nombre}','"NO PUBLICAR"'::jsonb)::text,'UTF8'));
  RAISE EXCEPTION 'Bolsa52: dato personal adicional aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 SELECT vec_bolsa_llamamientos.guardar_material_cese_b10_v1(evento,'cese',v.corte,NULL,
  proyeccion,manifiesto,convert_to(bolsas::text,'UTF8')) INTO nueva;
 IF nueva THEN RAISE EXCEPTION 'Bolsa52: primer material marcado replay'; END IF;
 SELECT vec_bolsa_llamamientos.guardar_material_cese_b10_v1(evento,'cese',v.corte,NULL,
  proyeccion,manifiesto,convert_to(bolsas::text,'UTF8')) INTO nueva;
 IF NOT nueva THEN RAISE EXCEPTION 'Bolsa52: replay material no reconocido'; END IF;
END $material$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.captura_cese_b10)<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.material_cese_b10)<>1 THEN
  RAISE EXCEPTION 'Bolsa52: historia inesperada';
 END IF;
END $historia$;
