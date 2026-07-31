-- Integridad relacional focal C2.2-B1. Todo se revierte al finalizar.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada'),
 ('prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_corporativa_rrhh_000000000001',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_corporativa_rrhh_000000000001',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_corporativo_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001','prf_corporativo_rrhh_000000000001',
  'per_corporativa_rrhh_000000000001',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_corporativa_rrhh_000000000002',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_corporativa_rrhh_000000000002',1,
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_corporativo_rrhh_000000000002',1,
  'per_corporativa_rrhh_000000000002',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_corporativo_rrhh_000000000002',1,
  'cta_corporativa_rrhh_000000000002','prf_corporativo_rrhh_000000000002',
  'per_corporativa_rrhh_000000000002',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_diputaciondemo0001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',
  '2026-01-01','2027-01-01');

INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01'),
 ('vcr_corporativo_rrhh_000000000001',2,
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','revocado','2026-01-01','2027-01-01'),
 ('vcr_corporativo_rrhh_000000000002',1,
  'cta_corporativa_rrhh_000000000002',1,
  'per_corporativa_rrhh_000000000002',1,
  'prf_corporativo_rrhh_000000000002',1,
  'vca_corporativo_rrhh_000000000002',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_vinculo_corporativo_rrhh_000001',1,repeat('b',64),
  'autoridad_maestra_acreditada','activo','2026-01-01','2027-01-01');

DO $integridad$
DECLARE
    campo text;
    valor jsonb;
    sentencia text;
    indice integer := 0;
    rechazadas integer := 0;
BEGIN
  INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
  SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
    to_jsonb(v)||jsonb_build_object('vinculo_corporativo_ref','vcr_'||repeat('a',22)))).*
    FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
   WHERE version=1 AND cuenta_ref='cta_corporativa_rrhh_000000000001' LIMIT 1;
  INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
  SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
    to_jsonb(v)||jsonb_build_object('vinculo_corporativo_ref','vcr_'||repeat('b',128)))).*
    FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
   WHERE version=1 AND cuenta_ref='cta_corporativa_rrhh_000000000001' LIMIT 1;
  INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
  SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
    to_jsonb(v)||jsonb_build_object('vinculo_corporativo_ref',
      'vcr_precision_microsegundos_0001','vigente_desde','2026-01-01 00:00:00.1234567+00'))).*
    FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
   WHERE version=1 AND cuenta_ref='cta_corporativa_rrhh_000000000001' LIMIT 1;
  IF (SELECT vigente_desde FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
       WHERE vinculo_corporativo_ref='vcr_precision_microsegundos_0001')
       <> '2026-01-01 00:00:00.123457+00'::timestamptz THEN
    RAISE EXCEPTION 'precision timestamptz(6) no normalizada';
  END IF;
  EXECUTE $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
    SELECT (jsonb_populate_record(
      NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
      to_jsonb(v)||jsonb_build_object('vinculo_corporativo_ref',$1)||
      jsonb_build_object($2,$3))).*
    FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
    WHERE version=1 AND cuenta_ref='cta_corporativa_rrhh_000000000001' LIMIT 1$$
    USING 'vcr_control_dinamico_valido_0001','estado',to_jsonb('activo'::text);

  FOR campo,valor IN SELECT * FROM (VALUES
    ('vinculo_corporativo_ref','null'::jsonb),
    ('vinculo_corporativo_ref',to_jsonb(''::text)),
    ('vinculo_corporativo_ref',to_jsonb('vcr_'||repeat('a',21))),
    ('vinculo_corporativo_ref',to_jsonb('vcr_'||repeat('a',129))),
    ('vinculo_corporativo_ref',to_jsonb('otra_'||repeat('a',22))),
    ('vinculo_corporativo_ref',to_jsonb('vcr_valor'||chr(10)||repeat('a',22))),
    ('version',to_jsonb(0::numeric)),
    ('version',to_jsonb(18446744073709551616::numeric)),
    ('cuenta_version',to_jsonb(0::numeric)),
    ('cuenta_version',to_jsonb(18446744073709551616::numeric)),
    ('persona_version',to_jsonb(0::numeric)),
    ('persona_version',to_jsonb(18446744073709551616::numeric)),
    ('perfil_version',to_jsonb(0::numeric)),
    ('perfil_version',to_jsonb(18446744073709551616::numeric)),
    ('vinculo_contexto_version',to_jsonb(0::numeric)),
    ('vinculo_contexto_version',to_jsonb(18446744073709551616::numeric)),
    ('organizacion_version',to_jsonb(0::numeric)),
    ('organizacion_version',to_jsonb(18446744073709551616::numeric)),
    ('organizacion_procedencia_version',to_jsonb(0::numeric)),
    ('organizacion_procedencia_version',to_jsonb(18446744073709551616::numeric)),
    ('procedencia_version',to_jsonb(0::numeric)),
    ('procedencia_version',to_jsonb(18446744073709551616::numeric)),
    ('cuenta_ref',to_jsonb('cta_corporativa_rrhh_000000000002'::text)),
    ('persona_ref',to_jsonb('per_corporativa_rrhh_000000000002'::text)),
    ('perfil_ref',to_jsonb('prf_corporativo_rrhh_000000000002'::text)),
    ('vinculo_contexto_ref',to_jsonb('vca_corporativo_rrhh_000000000002'::text)),
    ('organizacion_ref',to_jsonb('org_inexistentedemo0001'::text)),
    ('organizacion_procedencia_huella_sha256',to_jsonb(repeat('b',64))),
    ('organizacion_procedencia_huella_sha256',to_jsonb(repeat('z',64))),
    ('organizacion_procedencia_autoridad',to_jsonb('no_autoritativa'::text)),
    ('procedencia_huella_sha256',to_jsonb(repeat('a',64))),
    ('procedencia_huella_sha256',to_jsonb(repeat('z',64))),
    ('procedencia_autoridad',to_jsonb('no_autoritativa'::text)),
    ('superficie',to_jsonb('externa'::text)),
    ('uso',to_jsonb('administrar'::text)),
    ('estado',to_jsonb('suspendido'::text)),
    ('vigente_desde',to_jsonb('infinity'::text)),
    ('vigente_desde',to_jsonb('-infinity'::text)),
    ('vigente_hasta',to_jsonb('infinity'::text)),
    ('vigente_hasta',to_jsonb('-infinity'::text)),
    ('vigente_hasta',to_jsonb('2026-01-01'::text)),
    ('vigente_hasta',to_jsonb('2025-12-31'::text))
  ) m(campo,valor) LOOP
    indice := indice+1;
    BEGIN
      EXECUTE $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
        SELECT (jsonb_populate_record(
          NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
          to_jsonb(v)||jsonb_build_object('vinculo_corporativo_ref',$1)||
          jsonb_build_object($2,$3))).*
        FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
        WHERE version=1 AND cuenta_ref='cta_corporativa_rrhh_000000000001' LIMIT 1$$
        USING 'vcr_adversarial_'||lpad(indice::text,22,'0'),campo,valor;
    EXCEPTION WHEN OTHERS THEN
      rechazadas := rechazadas+1;
    END;
  END LOOP;

  FOREACH sentencia IN ARRAY ARRAY[
    $$UPDATE vec_contexto_actor_v1.vinculo_corporativo_versiones SET estado='activo'$$,
    $$DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.vinculo_corporativo_versiones$$,
    $$TRUNCATE vec_contexto_actor_v1.vinculo_corporativo_actual$$
  ] LOOP
    BEGIN
      EXECUTE sentencia;
    EXCEPTION WHEN OTHERS THEN
      rechazadas := rechazadas + 1;
    END;
  END LOOP;
  IF rechazadas <> 46
     OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_versiones) <> 7 THEN
    RAISE EXCEPTION 'integridad adversarial incompleta: % rechazos', rechazadas;
  END IF;
END
$integridad$;

DO $puntero$
DECLARE
  inicial numeric;
  observada numeric;
  sentencia text;
  rechazadas integer := 0;
BEGIN
  SELECT generacion INTO inicial
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  FOREACH sentencia IN ARRAY ARRAY[
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
       'vcr_corporativo_rrhh_000000000001',0)$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
       'vcr_corporativo_rrhh_000000000001',18446744073709551616)$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','externa','consulta_rrhh',
       'vcr_corporativo_rrhh_000000000001',2)$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','interna_corporativa','administrar',
       'vcr_corporativo_rrhh_000000000001',2)$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
       'vcr_corporativo_rrhh_000000000002',1)$$,
    $$INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
      ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
       'vcr_inexistente_rrhh_000000000001',1)$$
  ] LOOP
    BEGIN
      EXECUTE sentencia;
    EXCEPTION WHEN OTHERS THEN
      rechazadas := rechazadas+1;
    END;
  END LOOP;
  SELECT generacion INTO observada
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF rechazadas<>6 OR observada<>inicial
     OR EXISTS (SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual) THEN
    RAISE EXCEPTION 'punteros hostiles aceptados o alteraron generacion';
  END IF;
  INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
   ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
    'vcr_corporativo_rrhh_000000000001',2);
  SELECT generacion INTO observada
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF observada<>inicial+1 THEN RAISE EXCEPTION 'generacion tras INSERT no exacta'; END IF;
  UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=1;
  SELECT generacion INTO observada
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF observada<>inicial+2 THEN RAISE EXCEPTION 'generacion tras UPDATE no exacta'; END IF;
  DELETE FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
  SELECT generacion INTO observada
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF observada<>inicial+3 THEN RAISE EXCEPTION 'generacion tras DELETE no exacta'; END IF;
  INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
   ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
    'vcr_corporativo_rrhh_000000000001',2);
  SELECT generacion INTO observada
    FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;
  IF observada<>inicial+4
     OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_actual)<>1 THEN
    RAISE EXCEPTION 'puntero o generacion final no exactos';
  END IF;
END
$puntero$;
ROLLBACK;
