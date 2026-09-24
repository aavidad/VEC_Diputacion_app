\set ON_ERROR_STOP on
-- Focal de Personal 000013 con AD3 TEST-ONLY. No acredita concesión real.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.asignacion_dietas VALUES
 ('ads_aaaaaaaaaaaaaaaaaaaaaa','rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv',
  'unidad-sintetica',1,'centro-sintetico','per_bbbbbbbbbbbbbbbbbbbbbb',
  'per_dddddddddddddddddddddd',1,current_date,'alta sintética','acto-sintetico',
  'actor-sintetico',clock_timestamp()),
 ('ads_bbbbbbbbbbbbbbbbbbbbbb','rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv',
  'unidad-sintetica',2,'centro-sintetico','per_cccccccccccccccccccccc',
  'per_dddddddddddddddddddddd',1,current_date,'corrección sintética','acto-sintetico-2',
  'actor-sintetico',clock_timestamp());
COMMIT;

CREATE FUNCTION vec_prueba_d7.competencias_contexto(p_persona text)
RETURNS bytea LANGUAGE sql STABLE AS $$
 SELECT convert_to(jsonb_build_object(
  'esquema','vec.contexto-actor.vinculado.v2',
  'principal_ref','actor-'||substr(p_persona,5,8),
  'contexto_actor_ref','ctx-a','contexto_version','1',
  'cuenta_ref','cuenta-a','cuenta_version','1',
  'perfil_activo_ref','perfil-a','persona_ref',p_persona,
  'persona_version','1','perfil_version','1',
  'vinculos',jsonb_build_array(jsonb_build_object(
    'tipo','empleado','estado','activo','referencia','emp_'||substr(p_persona,5)))
 )::text,'UTF8') $$;
CREATE FUNCTION vec_prueba_d7.competencias_material(p_persona text)
RETURNS text LANGUAGE sql STABLE AS $$
 SELECT jsonb_build_object(
  'esquema','vec.personal.asignacion-dietas.competencias.v1',
  'fecha_referencia',current_date::text,
  'identidad',jsonb_build_object(
   'actor_ref','actor-'||substr(p_persona,5,8),
   'contexto_actor_ref','ctx-a','contexto_version','1',
   'cuenta_ref','cuenta-a','cuenta_version','1',
   'empleado_ref','emp_'||substr(p_persona,5),
   'perfil_ref','perfil-a','perfil_version','1',
   'persona_ref',p_persona,'persona_version','1'))::text $$;
CREATE FUNCTION vec_prueba_d7.competencias_capacidad(p_nonce text)
RETURNS bytea LANGUAGE sql IMMUTABLE AS $$
 SELECT convert_to(jsonb_build_object(
  'nonce',p_nonce,
  'operacion','personal.asignacion_dietas.competencias_consultar',
  'audiencia_consumo','vec_personal.asignacion_dietas.competencias.v1')::text,'UTF8') $$;
CREATE FUNCTION vec_prueba_d7.competencias_decision(p_persona text,p_nonce text)
RETURNS bytea LANGUAGE plpgsql STABLE AS $$
DECLARE h text; recurso text;
BEGIN
 h:=encode(sha256(convert_to(vec_prueba_d7.competencias_material(p_persona),'UTF8')),'hex');
 recurso:='{"ambitos":{"persona_ref":'||to_jsonb(p_persona)::text||
  '},"atributos":{"fecha_referencia":'||to_jsonb(current_date::text)::text||
  ',"material_sha256":'||to_jsonb(h)::text||',"operacion":"lista"}}';
 RETURN convert_to(jsonb_build_object(
  'concedida',true,'accion','personal.asignacion_dietas.competencias_consultar',
  'modulo_id','personal','tipo_recurso','asignacion_dietas_competencias',
  'recurso_ref',p_persona,'finalidad','tramitar_dietas_asignadas',
  'principal_id','actor-'||substr(p_persona,5,8),
  'perfil_activo_ref','perfil-a','decision_ref','dec_'||p_nonce,
  'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),
  'campos_permitidos','["asignacion_ref","auditoria_ad3_ref","cardinalidad","competencias","consultada_en","consumo_huella_sha256","decision_ref","efecto_ref","recibo_ref","relacion_ref","rol","unidad_ref","version","vigente_desde"]'::jsonb
 )::text,'UTF8');
END $$;
CREATE ROLE vec_prueba_d7b_personal NOLOGIN NOINHERIT NOSUPERUSER
 NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba_d7b_personal
 WITH INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA vec_prueba_d7 TO vec_personal_d7_ejecutor;
GRANT EXECUTE ON FUNCTION vec_prueba_d7.competencias_contexto(text),
 vec_prueba_d7.competencias_material(text),vec_prueba_d7.competencias_capacidad(text),
 vec_prueba_d7.competencias_decision(text,text) TO vec_personal_d7_ejecutor;

DO $acl$
DECLARE f regprocedure:='vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_personal_d7_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_dietas_ejecutor',f,'EXECUTE')
    OR has_table_privilege('vec_dietas_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR has_table_privilege('vec_personal_d7_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid=f AND x.grantee=0)
 THEN RAISE EXCEPTION 'ACL D7b abierta' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_policies
   WHERE schemaname='vec_personal'
     AND tablename IN ('recibo_competencias_asignacion_dietas',
                       'evidencia_competencias_asignacion_dietas')
     AND cmd IN ('ALL','UPDATE','DELETE'))
    OR (SELECT array_agg(cmd ORDER BY cmd) FROM pg_policies
        WHERE schemaname='vec_personal'
          AND tablename='recibo_competencias_asignacion_dietas')
       IS DISTINCT FROM ARRAY['INSERT','SELECT']
    OR (SELECT array_agg(cmd ORDER BY cmd) FROM pg_policies
        WHERE schemaname='vec_personal'
          AND tablename='evidencia_competencias_asignacion_dietas')
       IS DISTINCT FROM ARRAY['INSERT']
 THEN RAISE EXCEPTION 'RLS D7b no está separada por operación'; END IF;
END $acl$;

SET SESSION AUTHORIZATION vec_prueba_d7b_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE r record; p text; n text;
BEGIN
 p:='per_bbbbbbbbbbbbbbbbbbbbbb'; n:='competencias-b-0001';
 SELECT * INTO r FROM vec_personal.consultar_competencias_asignacion_dietas_v1(
  vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
  vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
  1,1,'p','s','e','r');
 IF r.cardinalidad<>0 OR r.competencias::jsonb<>'[]'::jsonb
 THEN RAISE EXCEPTION 'administrativo sustituido conserva competencia'; END IF;
 p:='per_cccccccccccccccccccccc'; n:='competencias-c-0001';
 SELECT * INTO r FROM vec_personal.consultar_competencias_asignacion_dietas_v1(
  vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
  vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
  1,1,'p','s','e','r');
 IF r.cardinalidad<>1 OR r.competencias::jsonb->0->>'rol'<>'administrativo'
    OR r.competencias::jsonb->0->>'version'<>'2'
 THEN RAISE EXCEPTION 'administrativo actual no consultable'; END IF;
 p:='per_dddddddddddddddddddddd'; n:='competencias-d-0001';
 SELECT * INTO r FROM vec_personal.consultar_competencias_asignacion_dietas_v1(
  vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
  vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
  1,1,'p','s','e','r');
 IF r.cardinalidad<>1 OR r.competencias::jsonb->0->>'rol'<>'responsable'
 THEN RAISE EXCEPTION 'responsable actual no consultable'; END IF;
 BEGIN
  PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(
   vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
   vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
   1,1,'p','s','e','r');
  RAISE EXCEPTION 'replay de consumo aceptado';
 EXCEPTION WHEN SQLSTATE 'P0573' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(
   vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad('competencias-ajena-1'),
   vec_prueba_d7.competencias_decision('per_cccccccccccccccccccccc','competencias-ajena-1'),
   'm',vec_prueba_d7.competencias_contexto(p),1,1,'p','s','e','r');
  RAISE EXCEPTION 'decisión de otro actor aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $prueba$;
COMMIT;
RESET SESSION AUTHORIZATION;

DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_personal.recibo_competencias_asignacion_dietas)<>3
    OR (SELECT count(*) FROM vec_personal.evidencia_competencias_asignacion_dietas)<>3
 THEN RAISE EXCEPTION 'recibos/evidencias D7b incompletos'; END IF;
END $historia$;

-- Un login con rol Personal antiguo añadido pierde la fachada D7b.
GRANT vec_personal_ejecutor TO vec_prueba_d7b_personal
 WITH INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_prueba_d7b_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $rol_mixto$
DECLARE p text:='per_cccccccccccccccccccccc'; n text:='competencias-rol-mixto';
BEGIN
 BEGIN
  PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(
   vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
   vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
   1,1,'p','s','e','r');
  RAISE EXCEPTION 'rol Personal mixto aceptado en D7b';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $rol_mixto$;
COMMIT;
RESET SESSION AUTHORIZATION;
REVOKE vec_personal_ejecutor FROM vec_prueba_d7b_personal;

-- 101 relaciones actuales fallan cerradas; no hay página parcial ni recibo.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.relacion_empleado_dietas
 (relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,
  version,procedencia_acto_ref,fuente_ref,fuente_version)
SELECT 'rel_'||lpad(g::text,22,'0'),'per_abcdefghijklmnopqrstuv',
 'emp_abcdefghijklmnopqrstuv','unidad-'||g,'activa',current_date-1,NULL,
 1,'acto-limite-'||g,'fuente-limite',1
FROM generate_series(1,100) g;
INSERT INTO vec_personal.asignacion_dietas
 (asignacion_ref,relacion_ref,persona_ref,unidad_ref,version,centro_ref,
  administrativo_persona_ref,responsable_persona_ref,grupo_dieta,
  vigente_desde,motivo_revision,procedencia_acto_ref,registrada_por_ref,registrada_en)
SELECT 'ads_'||lpad(g::text,22,'0'),'rel_'||lpad(g::text,22,'0'),
 'per_abcdefghijklmnopqrstuv','unidad-'||g,1,'centro-sintetico',
 'per_cccccccccccccccccccccc','per_dddddddddddddddddddddd',1,current_date,
 'alta sintética de límite','acto-limite-'||g,'actor-sintetico',clock_timestamp()
FROM generate_series(1,100) g;
COMMIT;
SET SESSION AUTHORIZATION vec_prueba_d7b_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $limite$
DECLARE p text:='per_cccccccccccccccccccccc'; n text:='competencias-c-limite';
BEGIN
 BEGIN
  PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(
   vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
   vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
   1,1,'p','s','e','r');
  RAISE EXCEPTION '101 competencias devueltas parcialmente';
 EXCEPTION WHEN SQLSTATE 'P7202' THEN NULL; END;
END $limite$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $sin_recibo$
BEGIN
 IF (SELECT count(*) FROM vec_personal.recibo_competencias_asignacion_dietas)<>3
 THEN RAISE EXCEPTION 'consulta >100 conservó recibo parcial'; END IF;
END $sin_recibo$;

-- 1001 candidatas históricas del actor sustituido ejercitan el tope interno.
-- Ninguna es competencia actual de ese actor; se deniega por coste acotado.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.relacion_empleado_dietas
 (relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,
  version,procedencia_acto_ref,fuente_ref,fuente_version)
SELECT 'rel_'||lpad((g+100)::text,22,'0'),'per_abcdefghijklmnopqrstuv',
 'emp_abcdefghijklmnopqrstuv','unidad-historica-'||g,'activa',current_date-1,NULL,
 1,'acto-historico-'||g,'fuente-historica',1
FROM generate_series(1,1000) g;
INSERT INTO vec_personal.asignacion_dietas
 (asignacion_ref,relacion_ref,persona_ref,unidad_ref,version,centro_ref,
  administrativo_persona_ref,responsable_persona_ref,grupo_dieta,
  vigente_desde,motivo_revision,procedencia_acto_ref,registrada_por_ref,registrada_en)
SELECT 'ads_'||lpad((g+100)::text,22,'0'),'rel_'||lpad((g+100)::text,22,'0'),
 'per_abcdefghijklmnopqrstuv','unidad-historica-'||g,1,'centro-sintetico',
 'per_bbbbbbbbbbbbbbbbbbbbbb','per_dddddddddddddddddddddd',1,current_date,
 'alta sintética histórica','acto-historico-'||g,'actor-sintetico',clock_timestamp()
FROM generate_series(1,1000) g;
INSERT INTO vec_personal.asignacion_dietas
 (asignacion_ref,relacion_ref,persona_ref,unidad_ref,version,centro_ref,
  administrativo_persona_ref,responsable_persona_ref,grupo_dieta,
  vigente_desde,motivo_revision,procedencia_acto_ref,registrada_por_ref,registrada_en)
SELECT 'ads_'||lpad((g+2100)::text,22,'0'),'rel_'||lpad((g+100)::text,22,'0'),
 'per_abcdefghijklmnopqrstuv','unidad-historica-'||g,2,'centro-sintetico',
 'per_cccccccccccccccccccccc','per_dddddddddddddddddddddd',1,current_date,
 'corrección sintética histórica','acto-correccion-'||g,'actor-sintetico',clock_timestamp()
FROM generate_series(1,1000) g;
COMMIT;
SET SESSION AUTHORIZATION vec_prueba_d7b_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $candidatos$
DECLARE p text:='per_bbbbbbbbbbbbbbbbbbbbbb'; n text:='competencias-b-candidatas';
BEGIN
 BEGIN
  PERFORM vec_personal.consultar_competencias_asignacion_dietas_v1(
   vec_prueba_d7.competencias_material(p),vec_prueba_d7.competencias_capacidad(n),
   vec_prueba_d7.competencias_decision(p,n),'m',vec_prueba_d7.competencias_contexto(p),
   1,1,'p','s','e','r');
  RAISE EXCEPTION '1001 candidatas históricas no aplicaron límite';
 EXCEPTION WHEN SQLSTATE 'P7202' THEN NULL; END;
END $candidatos$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $sin_recibo_candidatos$
BEGIN
 IF (SELECT count(*) FROM vec_personal.recibo_competencias_asignacion_dietas)<>3
 THEN RAISE EXCEPTION 'consulta >1000 candidatas conservó recibo parcial'; END IF;
END $sin_recibo_candidatos$;
