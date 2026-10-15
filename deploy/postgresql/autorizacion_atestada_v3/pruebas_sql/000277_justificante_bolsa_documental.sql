\set ON_ERROR_STOP on
-- Prueba focal tras AD277 en copia HZ37. No publica claves ni decisiones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid; c oid; def text; nombre text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AD277 prueba: PostgreSQL incompatible'; END IF;
 FOREACH nombre IN ARRAY ARRAY['vec_documentos_subida_datos','vec_documentos_frontera_externa'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=nombre AND NOT r.rolcanlogin
   AND NOT r.rolsuper AND NOT r.rolbypassrls AND NOT r.rolcreaterole AND NOT r.rolcreatedb)
   OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=nombre::regrole)
  THEN RAISE EXCEPTION 'AD277 prueba: grupo D20 incompatible: %',nombre; END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '2f50e8702cc5bc5bef150cd71bc33e0602b266ce009e1e5839cd67dce8bb90c0' OR
  (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
   IS DISTINCT FROM '0206aa1a00f6097ee2b8d3fac7bf1505871b9075c22f7db9092d1c0040a95e03' OR
  NOT has_function_privilege('vec_documentos_propietario',f,'EXECUTE') OR
  has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE') OR
  has_function_privilege('vec_documentos_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD277 prueba: consumidor Méritos alterado o abierto'; END IF;
 f:='vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
   p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
   AND p.proacl::text=
    '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_documentos_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE') OR
  has_function_privilege('vec_documentos_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD277 prueba: consumidor Bolsa abierto'; END IF;
 def:=pg_get_functiondef(f);
 FOREACH nombre IN ARRAY ARRAY[
  'bolsa.justificante_renuncia.subir_propio',
  'bolsa.justificante_renuncia.descargar_propio',
  'bolsa.justificante_renuncia.descargar_rrhh',
  'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
  'vec_documentos.bolsa.justificante_renuncia.subida.v1',
  'vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1',
  'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1',
  'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1'] LOOP
  IF position(nombre IN def)=0 THEN RAISE EXCEPTION 'AD277 prueba: acto/audiencia ausente: %',nombre; END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
   '7d13731f452b0f8d7020ea0ab602c465897617fce61a84a7887491309e1bbb48' OR
  NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
  p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND
  p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR position('revalidar_sesion_documental_bolsa_v1' IN pg_get_functiondef(f))=0 OR
  position('revalidar_sesion_documental_bolsa_rrhh_v1' IN pg_get_functiondef(f))=0 OR
  position('documentos_subida_local_externa' IN pg_get_functiondef(f))=0
 THEN RAISE EXCEPTION 'AD277 prueba: núcleo no conserva autoridades'; END IF;
 f:='vec_autorizacion.revalidar_sesion_documental_bolsa_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
  p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND
  p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
 OR has_function_privilege('vec_documentos_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD277 prueba: revalidación externa abierta'; END IF;
 f:='vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
  p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND
  p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}')
 OR has_function_privilege('vec_documentos_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD277 prueba: revalidación RRHH abierta'; END IF;
 f:='vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND
  p.proowner='vec_identidad_sesiones_v1_propietario'::regrole AND p.prosecdef AND
  p.proacl::text='{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario,vec_autorizacion_propietario=X/vec_identidad_sesiones_v1_propietario}')
 THEN RAISE EXCEPTION 'AD277 prueba: revalidador de identidad alterado'; END IF;
 c:=(SELECT oid FROM pg_constraint WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND conname='clave_capacidad_version_audiencia_consumo_check');
 IF c IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'AD277 prueba: claves sin frontera'; END IF;
 FOREACH nombre IN ARRAY ARRAY['vec_documentos.subida_local.acto_modulo.v1',
  'vec_documentos.subida_local.descarga_propia.v1',
  'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1',
  'vec_documentos.bolsa.justificante_renuncia.subida.v1',
  'vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1',
  'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1',
  'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1'] LOOP
  IF position(nombre IN pg_get_constraintdef(c))=0 THEN
   RAISE EXCEPTION 'AD277 prueba: clave sin audiencia: %',nombre; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['leer_configuracion_externa_v1(text,jsonb)',
  'comprobar_material_emision_externa_v1(text,jsonb)'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre);
  IF f IS NULL OR position('bolsa_justificantes' IN pg_get_functiondef(f))=0 OR
   position('meritos_cartera' IN pg_get_functiondef(f))=0
  THEN RAISE EXCEPTION 'AD277 prueba: inventario externo incompleto: %',nombre; END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
   'e7d87c2a63a927d9dddea33a5f4bf5bfbd6f1d767651da2ca5e11b0bf8678726' OR
  position('<>28' IN pg_get_functiondef(f))=0 THEN
  RAISE EXCEPTION 'AD277 prueba: publicación externa sin 28 claves'; END IF;
END $test$;
ROLLBACK;

-- SET ROLE no convierte al superusuario en un LOGIN documental nominal.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL ROLE vec_documentos_propietario;
DO $negativo$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD277 prueba: SET ROLE suplantó LOGIN documental';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $negativo$;
ROLLBACK;
\echo AD277_PRUEBA_ESTRUCTURAL_OK
