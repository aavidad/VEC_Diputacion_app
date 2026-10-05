\set ON_ERROR_STOP on
-- CT176 tras AD178, AD177 y CC7. Se ejecuta conectado con un LOGIN del runtime
-- CT (miembro de vec_contratacion_temporal_ejecutor, sin propietario/migrador).
-- Comprueba composición y rechazos previos a cualquier consumo; no hay firma positiva.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE f regprocedure:='vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 nombre text;
BEGIN
 IF NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') THEN
  RAISE EXCEPTION 'CT176 prueba: conéctate con un LOGIN del runtime CT'; END IF;
 -- El runtime sólo ve la fachada CT, no las piezas AD177 ni la lectura CC7.
 IF NOT has_function_privilege(session_user,f,'EXECUTE') THEN
  RAISE EXCEPTION 'CT176 prueba: runtime sin la fachada'; END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea)',
  'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'] LOOP
  -- Sin USAGE del esquema el nombre ni se resuelve: se busca el OID en el catálogo.
  IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname||'.'||p.proname||'('||pg_get_function_identity_arguments(p.oid)||')'
          =replace(nombre,',',', ')
      AND has_function_privilege(session_user,p.oid,'EXECUTE')) THEN
   RAISE EXCEPTION 'CT176 prueba: EXECUTE inesperado en %',nombre; END IF;
 END LOOP;
 IF has_table_privilege(session_user,'vec_contratacion_temporal.firma_documento_plan_v2','SELECT')
  OR has_table_privilege(session_user,'vec_contratacion_temporal.firma_documento_plan_v2','INSERT') THEN
  RAISE EXCEPTION 'CT176 prueba: la tabla hija no es privada'; END IF;
 -- 1. Contextos de actor distintos entre decisión interior y exterior: 42501.
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_con_plan_v2('{}',clock_timestamp(),
   '\x7b7d','\x7b7d','\x00','\x01',NULL,1,1,NULL,NULL,NULL,NULL,
   '\x7b7d','\x7b7d','\x00','\x02',1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT176 prueba: contextos distintos aceptados';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM<>'CT176 materiales ligados inválidos' THEN RAISE EXCEPTION 'CT176 prueba: causa inesperada 1: %',SQLERRM; END IF;
 END;
 -- 2. Envoltorio que no es JSON: 22023 antes de consumir.
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_con_plan_v2('{}',clock_timestamp(),
   convert_to('no-json','UTF8'),'\x7b7d','\x7b7d','\x00','\x01',1,1,NULL,NULL,NULL,NULL,
   '\x7b7d','\x7b7d','\x00','\x01',1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT176 prueba: envoltorio inválido aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN
  IF SQLERRM<>'CT176 material JSON inválido' THEN RAISE EXCEPTION 'CT176 prueba: causa inesperada 2: %',SQLERRM; END IF;
 END;
  -- 3. Envoltorio JSON sin la forma ct.plan-autorizado-firma.v2: 22023.
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_con_plan_v2('{}',clock_timestamp(),
   '\x7b7d','\x7b7d','\x7b7d','\x00','\x01',1,1,NULL,NULL,NULL,NULL,
   '\x7b7d','\x7b7d','\x00','\x01',1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT176 prueba: envoltorio sin forma aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN
  IF SQLERRM<>'CT176 envoltorio de plan inválido' THEN RAISE EXCEPTION 'CT176 prueba: causa inesperada 3: %',SQLERRM; END IF;
 END;
END $prueba$;
ROLLBACK;
SELECT 'CT176-COMPOSICION-Y-3-RECHAZOS-OK; sin firma positiva';
