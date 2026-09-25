\set ON_ERROR_STOP on
-- Prueba focal sin dobles. Ejecutar después de 000017, AD3-54 y 000018.
-- La ruta positiva V3 requiere un certificado de desarrollo y se recorre por HTTP.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $test$
DECLARE firma text:='text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea';
 f regprocedure; v record; n integer;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  ('vec_personal.consultar_registro_empleado_rrhh_v1('||firma||')')::regprocedure,
  ('vec_personal.consultar_vacantes_rrhh_v1('||firma||')')::regprocedure]
 LOOP
  SELECT p.proowner AS propietario,p.prosecdef AS definidora,p.provolatile AS volatilidad,
    p.proconfig AS configuracion INTO STRICT v FROM pg_proc p WHERE p.oid=f;
  IF v.propietario<>'vec_personal_propietario'::regrole OR NOT v.definidora
      OR v.volatilidad<>'v' OR NOT ('row_security=on'=ANY(v.configuracion))
      OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
      OR EXISTS (SELECT 1 FROM pg_proc px CROSS JOIN LATERAL
        aclexplode(coalesce(px.proacl,acldefault('f',px.proowner))) a
        WHERE px.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE') THEN
   RAISE EXCEPTION 'lectura B2: contrato de función/ACL incompatible %',f;
  END IF;
 END LOOP;
 IF has_function_privilege('vec_personal_ejecutor',
   'vec_personal.consultar_registro_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
 OR has_table_privilege('vec_personal_ejecutor','vec_personal.recibo_lectura_registro_empleado_b2','SELECT')
 OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
   WHERE oid='vec_personal.recibo_lectura_registro_empleado_b2'::regclass)
 THEN RAISE EXCEPTION 'lectura B2: frontera interna expuesta'; END IF;
 SELECT count(*) INTO n FROM pg_policies WHERE schemaname='vec_personal'
   AND tablename='recibo_lectura_registro_empleado_b2' AND roles=ARRAY['vec_personal_propietario']::name[];
 IF n<>1 THEN RAISE EXCEPTION 'lectura B2: política de recibo incompatible'; END IF;
 -- 000018 puede instalarse antes de 000020, pero no servir una ficha parcial.
 IF to_regprocedure('vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)') IS NOT NULL THEN
  SELECT count(*) INTO n FROM pg_attribute a
   WHERE a.attrelid IN ('vec_personal.relacion_servicio_historia'::regclass,
       'vec_personal.ocupacion_empleado_historia'::regclass,
       'vec_personal.situacion_empleado_historia'::regclass,
       'vec_personal.servicio_reconocido_historia'::regclass)
     AND a.attname='catalogo_snapshot' AND NOT a.attisdropped
     AND a.atttypid='jsonb'::regtype AND a.attnotnull;
  IF n<>4 THEN RAISE EXCEPTION 'lectura B2: catálogo incompleto'; END IF;
 END IF;
 FOR v IN SELECT unnest(ARRAY['relacion_servicio_historia','ocupacion_empleado_historia',
   'situacion_empleado_historia','servicio_reconocido_historia','cobertura_ocupaciones_historia',
   'plaza_plantilla_historia','puesto_rpt_historia']) AS nombre LOOP
  IF has_table_privilege('vec_personal_ejecutor','vec_personal.'||v.nombre,'SELECT')
      OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
        WHERE oid=('vec_personal.'||v.nombre)::regclass) THEN
   RAISE EXCEPTION 'lectura B2: tabla interna expuesta %',v.nombre;
  END IF;
 END LOOP;
END $test$;
-- Sin material V3 ninguna función puede entregar ficha ni cero vacantes.
DO $test$
BEGIN
 BEGIN
  PERFORM vec_personal.consultar_registro_empleado_rrhh_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'lectura B2: ficha sin autorización';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_vacantes_rrhh_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'lectura B2: vacantes sin autorización';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
-- Selector de otro organismo sin concesión ligada: respuesta opaca y cero recibos.
-- La prueba positiva con V3 firmado y dos relaciones de organismos distintos
-- pertenece al recorrido integrado; aquí no se sustituye AD3 por un doble.
DO $test$
DECLARE antes bigint; despues bigint;
 m text:='{"esquema":"vec.personal.registro-empleado-b2.consulta.v1","operacion":"ficha","empleado_ref":"emp_AAAAAAAAAAAAAAAAAAAAAA","organismo_ref":"org_ajeno","vigente_en":"2026-09-01","conocido_en":"2026-09-01T00:00:00.000000Z","limite":0,"cursor":"","actor_ref":"per_AAAAAAAAAAAAAAAAAAAAAA","contexto_actor_ref":"vca_AAAAAAAAAAAAAAAAAAAAAA","contexto_version":1,"cuenta_ref":"cta_AAAAAAAAAAAAAAAAAAAAAA","cuenta_version":1,"perfil_ref":"prf_AAAAAAAAAAAAAAAAAAAAAA","perfil_version":1,"persona_ref":"per_AAAAAAAAAAAAAAAAAAAAAA","persona_version":1}';
 vacio bytea:=convert_to('{}','UTF8');
BEGIN
 SELECT count(*) INTO antes FROM vec_personal.recibo_lectura_registro_empleado_b2;
 BEGIN
  PERFORM vec_personal.consultar_registro_empleado_rrhh_v1(m,vacio,vacio,vacio,vacio,
    1,1,vacio,vacio,vacio,vacio);
  RAISE EXCEPTION 'lectura B2: organismo ajeno obtuvo ficha';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 SELECT count(*) INTO despues FROM vec_personal.recibo_lectura_registro_empleado_b2;
 IF despues<>antes THEN RAISE EXCEPTION 'lectura B2: organismo ajeno dejó recibo'; END IF;
END $test$;
ROLLBACK;
