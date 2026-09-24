\set ON_ERROR_STOP on
-- Se ejecuta tras preparar_asignacion_dietas_000011.sql y 000011 en PG18 aislado.
-- El rol dedicado nace en 000011; los LOGIN sintéticos reciben solo ese grupo.
GRANT vec_personal_d7_ejecutor TO vec_prueba_d7_personal
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA vec_prueba_d7 TO vec_personal_d7_ejecutor;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vec_prueba_d7 TO vec_personal_d7_ejecutor;
CREATE FUNCTION vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1(
 p_relacion text,p_persona text,p_unidad text,p_asignacion text,p_version bigint,
 p_grupo smallint,p_centro text,p_administrativo text,p_responsable text,p_fecha date)
RETURNS boolean LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $$
 SELECT vec_personal.revalidar_asignacion_dietas_v1($1,$2,$3,coalesce($4,vec_prueba_d7.asignacion_v3_prueba()),$5,$6,$7,$8,$9,$10)
$$;
ALTER FUNCTION vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date) TO vec_dietas_ejecutor;
DO $acl$
DECLARE consulta regprocedure:='vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  corrige regprocedure:='vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  grupo regprocedure:='vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  alta regprocedure:='vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  sello regprocedure:='vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)'::regprocedure;
BEGIN
 IF has_table_privilege('vec_dietas_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR has_table_privilege('vec_personal_d7_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.asignacion_dietas','SELECT')
    OR has_function_privilege('vec_dietas_ejecutor',consulta,'EXECUTE')
    OR has_function_privilege('vec_dietas_ejecutor',corrige,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',consulta,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',consulta,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',corrige,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',corrige,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',grupo,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',grupo,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',alta,'EXECUTE')
    OR NOT has_function_privilege('vec_personal_d7_ejecutor',alta,'EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario',sello,'EXECUTE')
    OR pg_has_role('vec_prueba_d7_personal','vec_personal_ejecutor','MEMBER')
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m
      WHERE m.member='vec_prueba_d7_personal'::regrole
        AND m.roleid='vec_personal_d7_ejecutor'::regrole
        AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    OR (SELECT count(*) FROM pg_auth_members m
      WHERE m.member='vec_prueba_d7_personal'::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid IN (consulta,corrige,grupo,sello) AND x.grantee=0)
 THEN RAISE EXCEPTION 'ACL D7 no nominal ni cerrada' USING ERRCODE='55000'; END IF;
 IF (SELECT array_agg(cmd ORDER BY cmd) FROM pg_policies
     WHERE schemaname='vec_personal' AND tablename='recibo_asignacion_dietas')
      IS DISTINCT FROM ARRAY['INSERT','SELECT']::text[]
    OR (SELECT array_agg(cmd ORDER BY cmd) FROM pg_policies
     WHERE schemaname='vec_personal' AND tablename='evidencia_asignacion_dietas')
      IS DISTINCT FROM ARRAY['INSERT']::text[]
 THEN RAISE EXCEPTION 'políticas D7 deben ser por operación, sin UPDATE/DELETE/ALL'
      USING ERRCODE='55000'; END IF;
END $acl$;

SET SESSION AUTHORIZATION vec_prueba_d7_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SELECT * FROM vec_personal.registrar_asignacion_dietas_inicial_v1(
 vec_prueba_d7.material('registrar_inicial','018f1d1e-1234-4abc-8def-0123456789ab'),vec_prueba_d7.capacidad('registrar_inicial','alta-d7-000000000001'),vec_prueba_d7.decision('registrar_inicial','alta-d7-000000000001','018f1d1e-1234-4abc-8def-0123456789ab'),'m',vec_prueba_d7.contexto('registrar_inicial'),1,1,'p','s','e','r');
DO $consulta$
DECLARE r record;
BEGIN
 SELECT * INTO r FROM vec_personal.consultar_asignacion_dietas_v1(
   vec_prueba_d7.material('consultar'),vec_prueba_d7.capacidad('consultar','consulta-d7-00000001'),vec_prueba_d7.decision('consultar','consulta-d7-00000001'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 IF r.version<>1 OR r.grupo_dieta<>1 THEN RAISE EXCEPTION 'consulta D7 no devolvió la asignación inicial'; END IF;
END $consulta$;
COMMIT;
RESET ROLE; RESET SESSION AUTHORIZATION;

CREATE ROLE vec_prueba_d7_mixto NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor,vec_personal_ejecutor TO vec_prueba_d7_mixto
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_prueba_d7_mixto;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $mixto$
BEGIN
 BEGIN
  PERFORM vec_personal.consultar_asignacion_dietas_v1(
   vec_prueba_d7.material('consultar'),vec_prueba_d7.capacidad('consultar','mixto-d7-000000001'),
   vec_prueba_d7.decision('consultar','mixto-d7-000000001'),'m',
   vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'login mixto D7 aceptado';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $mixto$;
ROLLBACK;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_prueba_d7_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $corrige$
DECLARE r record;
BEGIN
 SELECT * INTO r FROM vec_personal.corregir_asignacion_dietas_v1(
   vec_prueba_d7.material('corregir','018f1d1e-1234-4abc-8def-0123456789ac'),vec_prueba_d7.capacidad('corregir','corr-d7-000000000001'),vec_prueba_d7.decision('corregir','corr-d7-000000000001','018f1d1e-1234-4abc-8def-0123456789ac'),'m',vec_prueba_d7.contexto('corregir'),1,1,'p','s','e','r');
 IF r.version<>2 OR r.estado_local<>'registrada' THEN RAISE EXCEPTION 'corrección nominal D7 no creó v2'; END IF;
 SELECT * INTO r FROM vec_personal.corregir_asignacion_dietas_v1(
   vec_prueba_d7.material('corregir','018f1d1e-1234-4abc-8def-0123456789ac'),vec_prueba_d7.capacidad('corregir','corr-d7-000000000002'),vec_prueba_d7.decision('corregir','corr-d7-000000000002','018f1d1e-1234-4abc-8def-0123456789ac'),'m',vec_prueba_d7.contexto('corregir'),1,1,'p','s','e','r');
 IF r.version<>2 OR r.estado_local<>'replay_confirmado' THEN RAISE EXCEPTION 'replay con consumo fresco no recuperó v2'; END IF;
 BEGIN
  PERFORM vec_personal.corregir_asignacion_dietas_v1(vec_prueba_d7.material('corregir','018f1d1e-1234-4abc-8def-0123456789ac','centro-conflicto'),vec_prueba_d7.capacidad('corregir','corr-d7-000000000003'),vec_prueba_d7.decision('corregir','corr-d7-000000000003','018f1d1e-1234-4abc-8def-0123456789ac','centro-conflicto'),'m',vec_prueba_d7.contexto('corregir'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'conflicto idempotente aceptado';
 EXCEPTION WHEN SQLSTATE 'P7204' THEN NULL; END;
 BEGIN
  PERFORM vec_personal.corregir_asignacion_dietas_v1(vec_prueba_d7.material('corregir','018f1d1e-1234-4abc-8def-0123456789ad','centro-corregido',2::smallint,2),vec_prueba_d7.capacidad('corregir','corr-d7-000000000004'),vec_prueba_d7.decision('corregir','corr-d7-000000000004','018f1d1e-1234-4abc-8def-0123456789ad','centro-corregido',2::smallint,2),'m',vec_prueba_d7.contexto('corregir'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'corrección ordinaria cambió grupo';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $corrige$;
COMMIT;
RESET ROLE; RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_prueba_d7_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $grupo$
DECLARE r record;
BEGIN
 BEGIN
  PERFORM vec_personal.corregir_grupo_dieta_v1(
   vec_prueba_d7.material_grupo_propio('018f1d1e-1234-4abc-8def-0123456789af'),vec_prueba_d7.capacidad('grupo_corregir','grupo-d7-000000000000'),vec_prueba_d7.decision_grupo_propio('grupo-d7-000000000000','018f1d1e-1234-4abc-8def-0123456789af'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'grupo propio aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO r FROM vec_personal.corregir_grupo_dieta_v1(
   vec_prueba_d7.material('grupo_corregir','018f1d1e-1234-4abc-8def-0123456789ae','centro-corregido',2::smallint,2),vec_prueba_d7.capacidad('grupo_corregir','grupo-d7-000000000001'),vec_prueba_d7.decision('grupo_corregir','grupo-d7-000000000001','018f1d1e-1234-4abc-8def-0123456789ae','centro-corregido',2::smallint,2),'m',vec_prueba_d7.contexto('grupo_corregir'),1,1,'p','s','e','r');
 IF r.version<>3 OR r.estado_local<>'registrada' THEN RAISE EXCEPTION 'sujeto/grupo separado no creó v3'; END IF;
END $grupo$;
COMMIT;
RESET ROLE; RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_prueba_d7_dietas;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $sello$
BEGIN
 BEGIN
  PERFORM vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','unidad-sintetica','ads_XXXXXXXXXXXXXXXXXXXXXX',2,1::smallint,'centro-corregido','per_bbbbbbbbbbbbbbbbbbbbbb','per_cccccccccccccccccccccc',current_date);
  RAISE EXCEPTION 'sello previo aceptado';
 EXCEPTION WHEN SQLSTATE 'P7201' THEN NULL; END;
 IF NOT vec_prueba_d7.revalidar_asignacion_terminal_dietas_v1('rel_abcdefghijklmnopqrstuv','per_abcdefghijklmnopqrstuv','unidad-sintetica',NULL,3,2::smallint,'centro-corregido','per_bbbbbbbbbbbbbbbbbbbbbb','per_cccccccccccccccccccccc',current_date) THEN RAISE EXCEPTION 'sello D7 vigente rechazado'; END IF;
END $sello$;
COMMIT;
RESET ROLE; RESET SESSION AUTHORIZATION;

BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
DO $historia$
BEGIN
 BEGIN UPDATE vec_personal.asignacion_dietas SET centro_ref='alterado' WHERE version=1; RAISE EXCEPTION 'historia D7 modificable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF (SELECT count(*) FROM vec_personal.asignacion_dietas)<>3
    OR (SELECT count(*) FROM vec_personal.recibo_asignacion_dietas)<>4
    OR (SELECT count(*) FROM vec_personal.evidencia_asignacion_dietas)<>0
    OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.d7_consumo_prueba)<>5
 THEN RAISE EXCEPTION 'replay/conflicto alteró historia o visibilidad RLS D7'
      USING ERRCODE='55000'; END IF;
END $historia$;
RESET ROLE;
DO $evidencia$
BEGIN
 IF (SELECT count(*) FROM vec_personal.evidencia_asignacion_dietas)<>4
 THEN RAISE EXCEPTION 'evidencia D7 durable incompleta' USING ERRCODE='55000'; END IF;
END $evidencia$;
COMMIT;
