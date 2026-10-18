\set ON_ERROR_STOP on
-- Ejecutar tras AD277 y D22 en una copia desechable; B119 habilita los actos.
-- Los positivos con material V3 y fichero real corresponden al ensayo conjunto.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid; nombre text; firma text:='bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea';
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_documentos.subida_local'::regclass
  AND relrowsecurity AND relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE
   conrelid='vec_documentos.subida_local'::regclass AND
   conname='subida_local_tipo_recurso_check' AND convalidated AND
   pg_get_constraintdef(oid,false) LIKE '%justificante_renuncia%' AND
   pg_get_constraintdef(oid,false) LIKE '%hecho%')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE
   conrelid='vec_documentos.subida_local_lectura'::regclass AND
   conname='subida_local_lectura_fase_check' AND convalidated AND
   pg_get_constraintdef(oid,false) LIKE '%abrir_rrhh%')
 OR has_table_privilege('vec_documentos_bolsa_subida_datos',
    'vec_documentos.subida_local','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('vec_documentos_bolsa_rrhh',
    'vec_documentos.subida_local','SELECT,INSERT,UPDATE,DELETE')
 OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_documentos','USAGE')
 OR to_regrole('vec_interno_documentos_dietas_datos') IS NOT NULL
 OR to_regrole('vec_interno_documentos_cronos_datos') IS NOT NULL
 THEN RAISE EXCEPTION 'D22 prueba: tabla, RLS o dependencia D21 divergente'; END IF;
 FOREACH nombre IN ARRAY ARRAY['vec_documentos_bolsa_subida_datos','vec_documentos_bolsa_rrhh'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=nombre AND NOT r.rolcanlogin
   AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
  OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=nombre::regrole)
  THEN RAISE EXCEPTION 'D22 prueba: grupo % no aislado',nombre; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['reservar_subida_local_v1','confirmar_recepcion_subida_local_v1',
  'iniciar_analisis_subida_local_v1','registrar_analisis_subida_local_v1',
  'cerrar_analisis_subida_local_v1','consultar_subida_local_propia_v1'] LOOP
  f:=to_regprocedure('vec_documentos.'||nombre||'('||firma||')');
  IF f IS NULL OR NOT has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE')
   OR NOT has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
   OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
   OR has_function_privilege('vec_documentos_frontera_externa',f,'EXECUTE')
   OR has_function_privilege('public',f,'EXECUTE')
  THEN RAISE EXCEPTION 'D22 prueba: ACL de fachada %',nombre; END IF;
 END LOOP;
 f:=to_regprocedure('vec_documentos.consultar_justificante_renuncia_rrhh_v1('||firma||')');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR position('validar_consulta_justificante_renuncia_rrhh_v1' IN pg_get_functiondef(f))=0
  OR position('consumir_subida_local_bolsa_v1' IN pg_get_functiondef(f))=0
 THEN RAISE EXCEPTION 'D22 prueba: consulta RRHH sin doble autoridad'; END IF;
 f:=to_regprocedure('vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text)');
 IF f IS NULL OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR position('verificar_integridad_subida_local_v1' IN pg_get_functiondef(f))=0
 THEN RAISE EXCEPTION 'D22 prueba: original sin verificación o ACL'; END IF;
 f:=to_regprocedure('vec_documentos.resolver_justificante_renuncia_bolsa_v1('
  ||'text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR position('proyectar_justificante_renuncia_v1' IN pg_get_functiondef(f))=0
  OR position('consumir_subida_local_bolsa_v1' IN pg_get_functiondef(f))=0
 THEN RAISE EXCEPTION 'D22 prueba: preparación propia sin autoridad'; END IF;
 f:=to_regprocedure('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1('
  ||'text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR position('proyectar_consulta_justificante_renuncia_rrhh_v1' IN pg_get_functiondef(f))=0
  OR position('consumir_subida_local_bolsa_v1' IN pg_get_functiondef(f))=0
 THEN RAISE EXCEPTION 'D22 prueba: preparación RRHH sin autoridad'; END IF;
 IF position('selector_subida_local_v1' IN pg_get_functiondef(
   to_regprocedure('vec_documentos.consumir_preimagen_subida_local_v1('||firma||')')))=0
 OR position('validar_propietario_subida_documental_v1' IN pg_get_functiondef(
   to_regprocedure('vec_documentos.operar_subida_local_interna_v1('||firma||')')))=0
 THEN RAISE EXCEPTION 'D22 prueba: selector o dueño no conectados'; END IF;
 f:=to_regprocedure('vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(jsonb,text,text,timestamptz)');
 IF f IS NULL OR has_function_privilege('public',f,'EXECUTE') OR
  has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE') OR
  position('candidato_propio_contexto_justificante_bolsa_v1' IN pg_get_functiondef(
   to_regprocedure('vec_documentos.resolver_justificante_renuncia_bolsa_v1('
    ||'text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')))=0
 THEN RAISE EXCEPTION 'D22 prueba: candidato firmado sin cotejo propio'; END IF;
END $test$;
ROLLBACK;

-- Focal puro: usa solo la forma canónica de ContextoActor y no emite una
-- capacidad falsa ni escribe una preparación B119.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $vinculo$
DECLARE ahora timestamptz(6):=clock_timestamp(); desde text; hasta text;
 persona constant text:='per_0123456789abcdefghijkl';
 candidato constant text:='can_0123456789abcdefghijkl';
 ajeno constant text:='can_zzzzzzzzzzzzzzzzzzzzzz';
 base jsonb; doble jsonb; vencido jsonb;
BEGIN
 desde:=to_char((ahora-interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 hasta:=to_char((ahora+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 base:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2',
  'persona_ref',persona,'estado','activo','vigente_desde',desde,'vigente_hasta',hasta,
  'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato',
   'estado','activo','referencia',candidato,'vigente_desde',desde,'vigente_hasta',hasta)));
 IF vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   base,persona,candidato,ahora) IS NOT TRUE OR
  vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   base,persona,ajeno,ahora) IS NOT FALSE
 THEN RAISE EXCEPTION 'D22 prueba: candidato propio o ajeno'; END IF;
 doble:=jsonb_set(base,'{vinculos}',base->'vinculos'||jsonb_build_array(
  jsonb_build_object('tipo','candidato','estado','activo','referencia',ajeno,
   'vigente_desde',desde,'vigente_hasta',hasta)));
 vencido:=jsonb_set(base,'{vinculos,0,vigente_hasta}',
  to_jsonb(to_char((ahora-interval '1 second') AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 IF vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   doble,persona,candidato,ahora) IS NOT FALSE OR
  vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   vencido,persona,candidato,ahora) IS NOT FALSE
 THEN RAISE EXCEPTION 'D22 prueba: varios candidatos o vínculo caducado'; END IF;
END $vinculo$;
ROLLBACK;

-- SET ROLE no cambia session_user ni sustituye a ninguno de los LOGIN.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_documentos_bolsa_subida_datos;
SET LOCAL TIME ZONE 'UTC';
DO $negativo$
BEGIN
 BEGIN
  PERFORM vec_documentos.reservar_subida_local_v1(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'D22 prueba: SET ROLE suplantó LOGIN externo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 BEGIN
  PERFORM vec_documentos.validar_justificante_renuncia_bolsa_v1(
   'ref:'||repeat('0',64),repeat('1',64),
   'justificante-renuncia:'||repeat('2',64),'persona','contexto');
  RAISE EXCEPTION 'D22 prueba: grupo externo validó originales';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $negativo$;
ROLLBACK;

BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_documentos_bolsa_rrhh;
SET LOCAL TIME ZONE 'UTC';
DO $negativo$
BEGIN
 BEGIN
  PERFORM vec_documentos.consultar_justificante_renuncia_rrhh_v1(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'D22 prueba: SET ROLE suplantó LOGIN RRHH';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $negativo$;
ROLLBACK;
\echo D22_PRUEBA_ESTRUCTURAL_OK
