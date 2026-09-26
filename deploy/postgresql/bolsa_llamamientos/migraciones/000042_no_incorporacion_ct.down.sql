\set ON_ERROR_STOP on
-- Bolsa 000042 DOWN: solo sin historia. Si la bandeja recibió alguna no
-- incorporación (también en cuarentena) o se publicó alguna política, la
-- reversión se niega: la historia es de solo adición. Retira el rol de grupo
-- del relevo; si alguna identidad LOGIN sigue siendo miembro, se niega.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000042',0));

DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = session_user AND rolsuper) THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: se ejecuta con una sesión DBA' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.no_incorporacion_bolsa') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_bolsa_llamamientos_relevo_no_incorporacion') THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: no está instalada' USING ERRCODE='55000';
 END IF;
 -- Sesión DBA: ve la historia aunque la tabla tenga RLS.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_no_incorporacion_bolsa) THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: reversión denegada con historia de no incorporaciones' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.roleid = 'vec_bolsa_llamamientos_relevo_no_incorporacion'::regrole) THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: el rol del relevo aún tiene miembros' USING ERRCODE='55000';
 END IF;
END $rol$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo IN SHARE ROW EXCLUSIVE MODE;

DO $continuacion$
DECLARE v_def text; v_acl aclitem[]; v_cambio record; v_oid oid:='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: rol de migración incompatible' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_no_incorporacion_bolsa) THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: reversión denegada con historia de no incorporaciones' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p WHERE p.oid=v_oid;
 FOR v_cambio IN SELECT * FROM (VALUES
   ($antes$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND tipo IN ('renuncia_rrhh','expiracion_rrhh') FOR SHARE;$antes$,
    $despues$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND (tipo IN ('renuncia_rrhh','expiracion_rrhh')
      OR (tipo='aceptacion_rrhh' AND vec_bolsa_llamamientos.no_incorporacion_registrada_b42(operacion_ref))) FOR SHARE;$despues$),
   ($antes$       (CASE v_terminal_anterior.tipo WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' ELSE 'renuncia' END) OR$antes$,
    $despues$       (CASE v_terminal_anterior.tipo WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' WHEN 'aceptacion_rrhh' THEN 'aceptacion' ELSE 'renuncia' END) OR$despues$)
 ) AS cambios(anterior,nuevo) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.nuevo,''))<>length(v_cambio.nuevo) THEN
   RAISE EXCEPTION 'Bolsa 000042 DOWN: guardado no localizado' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.nuevo,v_cambio.anterior);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000042 DOWN: guardado alterado' USING ERRCODE='55000';
 END IF;
END $continuacion$;

DROP FUNCTION vec_bolsa_llamamientos.consultar_avisos_rrhh_v3(timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.no_incorporacion_registrada_b42(text);
DROP FUNCTION vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1();
DROP FUNCTION vec_bolsa_llamamientos.reevaluar_no_incorporacion_bolsa_v1(text,jsonb);
DROP FUNCTION vec_bolsa_llamamientos.pendientes_no_incorporacion_bolsa_v1(integer);
DROP FUNCTION vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb);
DROP FUNCTION vec_bolsa_llamamientos.evaluar_no_incorporacion_b42(text,jsonb);
DROP FUNCTION vec_bolsa_llamamientos.consecuencia_no_incorporacion_b42(text,date,jsonb);
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_no_incorporacion_bolsa_v1(text,text,jsonb,text,text);
DROP FUNCTION vec_bolsa_llamamientos.consecuencia_politica_valida_b42(text,jsonb);
DROP TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena;
DROP TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa_evaluacion;
DROP TABLE vec_bolsa_llamamientos.no_incorporacion_bolsa;
DROP TABLE vec_bolsa_llamamientos.politica_no_incorporacion_bolsa;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_relevo_no_incorporacion;
RESET ROLE;
DO $retirar$
BEGIN
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_bolsa_llamamientos_relevo_no_incorporacion', current_database());
END $retirar$;
DROP ROLE vec_bolsa_llamamientos_relevo_no_incorporacion;
COMMIT;
