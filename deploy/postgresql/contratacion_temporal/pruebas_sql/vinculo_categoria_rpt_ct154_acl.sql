\set ON_ERROR_STOP on
-- Ejecutar solo sobre clon desechable tras AD3-127 y CT-154.
-- Comprueba firma, canon de entrada y denegación previa al acceso a datos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $pre$
BEGIN
 IF pg_catalog.to_regrole('vec_ct154_ensayo_login') IS NOT NULL
 THEN RAISE EXCEPTION 'CT154: identidad de ensayo ya existente' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_ct154_ensayo_login LOGIN INHERIT NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct154_ensayo_login WITH INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_ct154_ensayo_login;
DO $prueba$
DECLARE
 consulta text:='{"esquema":"vec.ct.vinculo-categoria-rpt.consulta.v1","organizacion_ref":"organizacion:desarrollo:dipgra","expediente_ref":"expediente:ensayo"}';
 mal_orden text:='{"expediente_ref":"expediente:ensayo","organizacion_ref":"organizacion:desarrollo:dipgra","esquema":"vec.ct.vinculo-categoria-rpt.consulta.v1"}';
 registro text:='{"esquema":"vec.ct.vinculo-categoria-rpt.registro.v1","organizacion_ref":"organizacion:desarrollo:dipgra","expediente_ref":"expediente:ensayo","version_expediente_esperada":2,"analisis_version":2,"analisis_recibo_ref":"recibo:ensayo","analisis_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","categoria_ref":"rpt:ensayo","catalogo_id":"rpt.categorias","modulo_id":"organizacion_rpt","catalogo_version":1,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","categoria_id":"rpt:ensayo","fuente_ref":"fuente:ensayo","motivo_ref":"motivo:ensayo","aprobacion_ref":"aprobacion:ensayo","revision_esperada":0,"anterior_recibo_ref":null,"clave_idempotencia":"00000000-0000-4000-8000-000000000154"}';
 b bytea:=convert_to('{}','UTF8');
 d bytea:=convert_to('{"principal_id":"actor:ensayo","perfil_activo_ref":"perfil:ensayo"}','UTF8');
 drpt bytea:=convert_to('{"principal_id":"actor:ensayo","perfil_activo_ref":"perfil:rpt:ensayo"}','UTF8');
	 mensaje text; fachada regprocedure; entorno text[];
BEGIN
	 FOREACH fachada IN ARRAY ARRAY[
	  'vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text)'::regprocedure,
	  'vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
	  'vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
	  SELECT proconfig INTO STRICT entorno FROM pg_proc WHERE oid=fachada;
	  IF entorno IS DISTINCT FROM (CASE
	    WHEN fachada='vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text)'::regprocedure
	    THEN ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
	    ELSE ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'] END)
	  THEN RAISE EXCEPTION 'CT154: search_path inseguro: %',fachada; END IF;
	 END LOOP;
	 IF has_table_privilege(current_user,'vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1','SELECT')
    OR has_table_privilege(current_user,'vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1','INSERT')
    OR NOT has_function_privilege(current_user,
      'vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege(current_user,
      'vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'CT154: ACL inesperada'; END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(
   mal_orden,b,d,b,b,1,1,b,b,b,b);
  RAISE EXCEPTION 'CT154: aceptó orden incorrecto';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(
   consulta,b,d,b,b,1,1,b,b,b,b);
  RAISE EXCEPTION 'CT154: consulta sin V3 aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
   registro,b,d,b,b,1,1,b,b,b,b,
   b,d,b,b,1,1,b,b,b,b);
  RAISE EXCEPTION 'CT154: aceptó el mismo perfil CT y RPT';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  IF mensaje<>'CT-154: identidades divergentes'
  THEN RAISE EXCEPTION 'CT154: no denegó el perfil compartido antes de V3'; END IF;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
   registro,b,d,b,b,1,1,b,b,b,b,
   b,drpt,b,b,1,1,b,b,b,b);
  RAISE EXCEPTION 'CT154: registro sin V3 aceptado';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  IF mensaje='CT-154: identidades divergentes'
  THEN RAISE EXCEPTION 'CT154: rechazó perfiles nominales distintos'; END IF;
 END;
END $prueba$;
ROLLBACK;
