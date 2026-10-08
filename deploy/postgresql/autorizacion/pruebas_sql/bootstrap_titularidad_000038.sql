\set ON_ERROR_STOP on
-- Clon sintético con AUT37/CA33/IS15. Dirección carga por parámetros enlazados
-- vec.ensayo.plan_bootstrap: nuevo plan V3 preparado con cuentas IS reales.
-- Este vector comprueba la fachada propietaria CA, no acredita arranque 2+1.
-- No sustituye la comprobación de unidades Personal31 de la composición AUT.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $vectores$
DECLARE plan jsonb:=current_setting('vec.ensayo.plan_bootstrap')::jsonb;
 persona jsonb;pre jsonb;replay jsonb;malo jsonb;cuenta_actual record;
 perfiles_antes bigint;vinculos_antes bigint;titularidades_antes bigint;rechazo boolean;
BEGIN
 IF plan IS NULL OR jsonb_array_length(plan->'personas')<>2 THEN
  RAISE EXCEPTION 'AUT38 pruebas: falta plan sintético privado'; END IF;
 SELECT count(*) INTO perfiles_antes FROM vec_contexto_actor_v1.perfil_versiones;
 SELECT count(*) INTO vinculos_antes FROM vec_contexto_actor_v1.vinculo_contexto_versiones;
 SELECT count(*) INTO titularidades_antes FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1;
 FOR persona IN SELECT e.value FROM jsonb_array_elements(plan->'personas') e LOOP
  IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
   JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
   WHERE v.cuenta_ref=persona->>'cuenta_ref' AND v.persona_ref=persona->>'persona_ref')
  THEN RAISE EXCEPTION 'AUT38 pruebas: fixture no representa titularidad sin perfil'; END IF;
  pre:=vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(persona);
  replay:=vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(persona);
  IF pre IS DISTINCT FROM replay OR pre->'titularidad' IS NULL
  OR pre#>>'{titularidad,cuenta_ref}' IS DISTINCT FROM persona->>'cuenta_ref'
  OR pre#>>'{titularidad,persona_ref}' IS DISTINCT FROM persona->>'persona_ref'
  OR pre#>>'{titularidad,alcance_fuente}' IS DISTINCT FROM 'sintetico_declarado'
  OR pre#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb
  OR pre#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb
  THEN RAISE EXCEPTION 'AUT38 pruebas: preimagen sin titularidad real/replay estable'; END IF;
  -- Cambio de versión solicitada no puede reusar la titularidad v1.
  rechazo:=false;
  BEGIN
   PERFORM vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb_set(persona,'{cuenta_version}','2'));
  EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'AUT38 pruebas: CAS divergente aceptado'; END IF;
  -- Cruce de dos Personas existentes: identidad individual no acredita titularidad.
  malo:=jsonb_set(persona,'{persona_ref}',to_jsonb(CASE WHEN persona->>'persona_ref'=plan#>>'{personas,0,persona_ref}' THEN plan#>>'{personas,1,persona_ref}' ELSE plan#>>'{personas,0,persona_ref}' END));
  rechazo:=false;
  BEGIN
   PERFORM vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(malo);
  EXCEPTION WHEN serialization_failure THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'AUT38 pruebas: cruce cuenta-Persona aceptado'; END IF;
  -- Simular retirada en la proyección sintética propietaria: publicación v2
  -- sólo dentro de este subbloque, que siempre revierte. No es una revocación
  -- administrativa atestada ni una fuente para el positivo del bootstrap.
  SELECT v.* INTO STRICT cuenta_actual FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
   JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
   WHERE a.cuenta_ref=persona->>'cuenta_ref' FOR UPDATE OF a;
  rechazo:=false;
  BEGIN
   INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
   VALUES(cuenta_actual.cuenta_ref,cuenta_actual.version+1,cuenta_actual.procedencia_ref,cuenta_actual.procedencia_version,cuenta_actual.procedencia_huella_sha256,cuenta_actual.procedencia_autoridad,'revocado',clock_timestamp(),cuenta_actual.vigente_hasta);
   UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=cuenta_actual.version+1 WHERE cuenta_ref=cuenta_actual.cuenta_ref;
   PERFORM vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(persona);
   RAISE EXCEPTION 'AUT38 pruebas: retirada aceptada';
  EXCEPTION WHEN insufficient_privilege THEN rechazo:=true; END;
  IF NOT rechazo THEN RAISE EXCEPTION 'AUT38 pruebas: retirada no denegada'; END IF;
  IF vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(persona) IS DISTINCT FROM pre
  THEN RAISE EXCEPTION 'AUT38 pruebas: subbloque negativo modificó fuente original'; END IF;
 END LOOP;
 IF (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones)<>perfiles_antes
 OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones)<>vinculos_antes
 OR (SELECT count(*) FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1)<>titularidades_antes
 THEN RAISE EXCEPTION 'AUT38 pruebas: consulta creó perfiles/vínculos/titularidad'; END IF;
END $vectores$;
ROLLBACK;
