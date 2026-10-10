\set ON_ERROR_STOP on
-- AUT72: la respuesta de un intento B1 fallido lleva el SQLSTATE (sólo el
-- código) y el asiento de auditoría sigue siendo el de AD227. Antes de AUT72
-- esta prueba falla porque la respuesta no trae «sqlstate».
-- Ejecutar como superusuario en un clon PG18 SIN vec-server después de AUT72
-- (toca el latido del sellador dentro de la transacción). Todo termina en
-- ROLLBACK: el LOGIN de prueba, su membresía y los asientos no quedan.
-- Llama a las fachadas reales con un LOGIN del grupo B1, como el Go.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';

DO $solo_clon$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid()
  AND backend_type='client backend') THEN
  RAISE EXCEPTION 'AUT72 prueba: hay otras sesiones en la base; ejecutar solo en un clon sin vec-server'; END IF;
END $solo_clon$;

CREATE ROLE vec_adm_b1_prueba_aut72 LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_admin_version_rol_bolsa_ejecutor TO vec_adm_b1_prueba_aut72 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- Sólo en esta transacción: llamar directamente a la función con códigos
-- que las fachadas no producen (formato inválido).
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(text,text,text) TO vec_adm_b1_prueba_aut72;
CREATE TEMP TABLE aut72_resultado(caso text PRIMARY KEY,respuesta jsonb NOT NULL) ON COMMIT DROP;
GRANT INSERT,SELECT ON aut72_resultado TO vec_adm_b1_prueba_aut72;

-- Sin vec-server no late el sellador y AD207 rechaza todo asiento nuevo:
-- en esta transacción se da por recién latido (ROLLBACK al final).
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=now() WHERE control;

SET SESSION AUTHORIZATION vec_adm_b1_prueba_aut72;
-- 1) Contexto inválido (decisión vacía): aplicar falla con 42501 → denegado.
INSERT INTO aut72_resultado SELECT 'denegado_42501',vec_autorizacion.proponer_version_rol_bolsa_v1(
 '{}',convert_to('{}','UTF8'),''::bytea,convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,
 convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to(repeat('a',44),'UTF8'));
-- 2) Material que no es JSON: el cast falla con 22P02 → error.
INSERT INTO aut72_resultado SELECT 'error_22P02',vec_autorizacion.cerrar_version_rol_bolsa_v1(
 'x',convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,
 convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to(repeat('a',44),'UTF8'));
-- 3) Llamada directa: código válido de error, y dos formatos que no son SQLSTATE.
INSERT INTO aut72_resultado SELECT 'error_42703',vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
 '{}','administracion.perfiles.version_bolsa.proponer','42703');
INSERT INTO aut72_resultado SELECT 'minusculas',vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
 '{}','administracion.perfiles.version_bolsa.proponer','4270a');
INSERT INTO aut72_resultado SELECT 'largo',vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
 '{}','administracion.perfiles.version_bolsa.aprobar','42703 x');
INSERT INTO aut72_resultado SELECT 'nulo',vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
 '{}','administracion.perfiles.version_bolsa.aprobar',NULL);
RESET SESSION AUTHORIZATION;

DO $comprobar$
DECLARE r record;esperado jsonb:='{
 "denegado_42501":["denegado","42501"],"error_22P02":["error","22P02"],"error_42703":["error","42703"],
 "minusculas":["error",null],"largo":["error",null],"nulo":["error",null]}';
 asiento record;n integer:=0;
BEGIN
 FOR r IN SELECT * FROM aut72_resultado LOOP
  IF r.respuesta->>'estado' IS DISTINCT FROM esperado#>>ARRAY[r.caso,'0']
  OR r.respuesta->>'codigo' IS DISTINCT FROM 'version_rol_bolsa_'||(esperado#>>ARRAY[r.caso,'0'])
  OR NOT r.respuesta ? 'sqlstate'
  OR r.respuesta->'sqlstate' IS DISTINCT FROM esperado#>ARRAY[r.caso,'1']
  OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(r.respuesta) k)
     IS DISTINCT FROM ARRAY['auditoria_intento','codigo','estado','sqlstate']::text[] THEN
   RAISE EXCEPTION 'AUT72 prueba [%]: respuesta inesperada %',r.caso,r.respuesta; END IF;
  -- El asiento es el de AD227: sin código SQL, mismo formato de siempre.
  SELECT * INTO STRICT asiento FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
   WHERE auditoria_ref=r.respuesta#>>'{auditoria_intento,auditoria_ref}';
  IF asiento.tipo_registro<>'intento_version_rol_bolsa' OR asiento.resultado<>r.respuesta->>'estado'
  OR asiento.motivo_ref<>r.respuesta->>'codigo' OR asiento.operador_login<>'vec_adm_b1_prueba_aut72'
  OR EXISTS(SELECT 1 FROM jsonb_each_text(to_jsonb(asiento)) e WHERE e.value=esperado#>>ARRAY[r.caso,'1']) THEN
   RAISE EXCEPTION 'AUT72 prueba [%]: asiento inesperado',r.caso; END IF;
  n:=n+1;
 END LOOP;
 IF n<>6 THEN RAISE EXCEPTION 'AUT72 prueba: % casos, esperados 6',n; END IF;
 RAISE NOTICE 'AUT72 prueba: OK % casos',n;
END $comprobar$;
ROLLBACK;
