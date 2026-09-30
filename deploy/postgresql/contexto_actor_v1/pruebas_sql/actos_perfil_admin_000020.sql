\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

DO $prueba$
DECLARE c constant text := 'cta_ca20_prueba_aaaaaaaaaaaaaaaaaaaaaaaa';
        pe constant text := 'per_ca20_prueba_bbbbbbbbbbbbbbbbbbbbbbbb';
        p0 constant text := 'prf_ca20_prueba_000000000000000000000000';
        v0 constant text := 'vca_ca20_prueba_000000000000000000000000';
        p1 constant text := 'prf_ca20_prueba_cccccccccccccccccccccccc';
        v1 constant text := 'vca_ca20_prueba_dddddddddddddddddddddddd';
        p2 constant text := 'prf_ca20_prueba_eeeeeeeeeeeeeeeeeeeeeeee';
        v2 constant text := 'vca_ca20_prueba_ffffffffffffffffffffffff';
        pe_ajena constant text := 'per_ca20_prueba_zzzzzzzzzzzzzzzzzzzzzzzz';
        p_ajeno constant text := 'prf_ca20_prueba_zzzzzzzzzzzzzzzzzzzzzzzz';
        v_ajeno constant text := 'vca_ca20_prueba_zzzzzzzzzzzzzzzzzzzzzzzz';
        pr1 constant text := 'prc_ca20_prueba_111111111111111111111111';
        pr2 constant text := 'prc_ca20_prueba_222222222222222222222222';
        pr3 constant text := 'prc_ca20_prueba_333333333333333333333333';
        fallo text;
BEGIN
  IF pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',
    'vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz)',
    'EXECUTE') THEN
    RAISE EXCEPTION 'CA20: runtime puede mutar';
  END IF;
  INSERT INTO vec_contexto_actor_v1.procedencias VALUES
    (pr1,1,repeat('1',64),'autoridad_maestra_acreditada'),
    (pr2,1,repeat('2',64),'autoridad_maestra_acreditada'),
    (pr3,1,repeat('3',64),'autoridad_maestra_acreditada');
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
    (cuenta_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (c,1,pr1,1,repeat('1',64),'autoridad_maestra_acreditada','activo',
          pg_catalog.clock_timestamp()-interval '1 hour',pg_catalog.clock_timestamp()+interval '1 hour');
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES (c,1);
  INSERT INTO vec_contexto_actor_v1.persona_versiones
    (persona_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (pe,1,pr1,1,repeat('1',64),'autoridad_maestra_acreditada','activo',
          pg_catalog.clock_timestamp()-interval '1 hour',pg_catalog.clock_timestamp()+interval '1 hour');
  INSERT INTO vec_contexto_actor_v1.persona_actual VALUES (pe,1);
  -- Vínculo previo acreditado por la fuente: el alta administrativa no crea
  -- por sí sola la relación entre una cuenta y una persona.
  INSERT INTO vec_contexto_actor_v1.perfil_versiones
    (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (p0,1,pe,pr1,1,repeat('1',64),'autoridad_maestra_acreditada','activo',
          pg_catalog.clock_timestamp()-interval '1 hour',pg_catalog.clock_timestamp()+interval '1 hour');
  INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES (p0,1);
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
    (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
     procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (v0,1,c,p0,pe,pr1,1,repeat('1',64),'autoridad_maestra_acreditada','activo',
          pg_catalog.clock_timestamp()-interval '1 hour',pg_catalog.clock_timestamp()+interval '1 hour');
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES (v0,1);
  INSERT INTO vec_contexto_actor_v1.persona_versiones
    (persona_ref,version,procedencia_ref,procedencia_version,procedencia_huella_sha256,
     procedencia_autoridad,estado,vigente_desde,vigente_hasta)
  VALUES (pe_ajena,1,pr1,1,repeat('1',64),'autoridad_maestra_acreditada','activo',
          pg_catalog.clock_timestamp()-interval '1 hour',pg_catalog.clock_timestamp()+interval '1 hour');
  INSERT INTO vec_contexto_actor_v1.persona_actual VALUES (pe_ajena,1);
  BEGIN
    PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
      c,pe_ajena,1,1,p_ajeno,v_ajeno,pr2,1,repeat('2',64),pg_catalog.clock_timestamp()+interval '1 hour');
    RAISE EXCEPTION 'CA20: cuenta vinculada a persona ajena';
  EXCEPTION WHEN sqlstate '55000' THEN NULL;
  END;

  PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
    c,pe,1,1,p1,v1,pr2,1,repeat('2',64),pg_catalog.clock_timestamp()+interval '1 hour');
  PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
    c,pe,1,1,p2,v2,pr2,1,repeat('2',64),pg_catalog.clock_timestamp()+interval '1 hour');
  IF (SELECT count(*) FROM vec_contexto_actor_v1.perfil_actual pa
      JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
      WHERE pv.persona_ref=pe AND pv.estado='activo'
        AND pa.perfil_ref IN (p1,p2)) <> 2 THEN
    RAISE EXCEPTION 'CA20: multiples perfiles de persona no conservados';
  END IF;

  PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(
    c,pe,p1,v1,1,1,1,1,pr3,1,repeat('3',64));
  IF (SELECT version FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=p1) <> 2
     OR (SELECT version FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=v1) <> 2
     OR (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p1) <> 2
     OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=v1) <> 2
     OR (SELECT version FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=p2) <> 1 THEN
    RAISE EXCEPTION 'CA20: CAS o historia incorrectos';
  END IF;
  BEGIN
    PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(
      c,pe,p1,v1,1,1,1,1,pr3,1,repeat('3',64));
    RAISE EXCEPTION 'CA20: segundo CAS aceptado';
  EXCEPTION WHEN sqlstate '40001' THEN NULL;
  END;
  BEGIN
    PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(
      c,pe,1,1,p1,v1,pr2,1,repeat('2',64),pg_catalog.clock_timestamp()+interval '1 hour');
    RAISE EXCEPTION 'CA20: referencia revocada reutilizada';
  EXCEPTION WHEN sqlstate '55000' THEN NULL;
  END;
  IF (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p1) <> 2 THEN
    RAISE EXCEPTION 'CA20: prueba negativa altero historia';
  END IF;
END $prueba$;
ROLLBACK;
