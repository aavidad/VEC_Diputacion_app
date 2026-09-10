\set ON_ERROR_STOP on
\set VERBOSITY terse
-- PRUEBA DE CATÁLOGOS PREPARADA, NO EJECUTADA. No acredita criptografía,
-- instalación efectiva, legitimidad de fuente ni firma legal.
-- Sólo una base DESECHABLE vec_ct87_prueba_<12 hex>, identificada además por
-- pg_temp.ct87_entorno(base name PRIMARY KEY, solo_sinteticos boolean), una fila
-- current_database()/true. Nunca la base conservada; este archivo no instala SQL.
-- El runner autorizado conserva en LA MISMA SESIÓN antes de instalar AD3-31:
--   CREATE TEMP TABLE ad3_31_preimagen AS
--     SELECT p.oid::regprocedure::text AS identidad, to_jsonb(p) AS metadatos,
--            pg_get_functiondef(p.oid) AS definicion
--     FROM pg_proc p
--     WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace;
-- Capturar con search_path=pg_catalog, mantener la temporal entre transacciones,
-- instalar una sola vez en el clon autorizado y luego incluir esta prueba.
-- Si falta esa preimagen, se aborta: el estado final no demuestra por sí solo
-- que las ACL/propietarios/configuración del núcleo o del alta se conservaron.
-- No modificar una ACL para hacer pasar esta prueba. No contiene UP ni DOWN.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $catalogos$
DECLARE
 nuevo text:='vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_administrativo_sin_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 nucleo text:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 alta text:='vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 f oid; propietario oid; ct oid; p record; b record; identidad text;
BEGIN
 IF current_database() !~ '^vec_ct87_prueba_[a-f0-9]{12}$'
  OR to_regclass('pg_temp.ct87_entorno') IS NULL
  OR to_regclass('pg_temp.ad3_31_preimagen') IS NULL THEN
  RAISE EXCEPTION 'AD3-31: falta entorno desechable/preimagen anterior'; END IF;
 IF (SELECT count(*) FROM pg_temp.ct87_entorno)<>1 OR NOT EXISTS(
  SELECT 1 FROM pg_temp.ct87_entorno WHERE base=current_database() AND solo_sinteticos IS TRUE) THEN
  RAISE EXCEPTION 'AD3-31: entorno no identificado como sintético'; END IF;
 f:=to_regprocedure(nuevo);
 propietario:='vec_autorizacion_atestada_v3_propietario'::regrole;
 ct:='vec_contratacion_temporal_propietario'::regrole;
 IF f IS NULL OR to_regprocedure(nucleo) IS NULL OR to_regprocedure(alta) IS NULL THEN
  RAISE EXCEPTION 'AD3-31: consumidores requeridos ausentes'; END IF;
 IF EXISTS(SELECT 1 FROM pg_temp.ad3_31_preimagen WHERE identidad=nuevo)
  OR NOT EXISTS(SELECT 1 FROM pg_temp.ad3_31_preimagen WHERE identidad=nucleo)
  OR NOT EXISTS(SELECT 1 FROM pg_temp.ad3_31_preimagen WHERE identidad=alta)
  OR EXISTS(SELECT identidad FROM pg_temp.ad3_31_preimagen GROUP BY identidad HAVING count(*)<>1)
  OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace)
     <> 1+(SELECT count(*) FROM pg_temp.ad3_31_preimagen) THEN
  RAISE EXCEPTION 'AD3-31: preimagen/inventario incompatible'; END IF;

 -- Todas las funciones previas conservan catálogo completo y definición.
 -- Sólo prosrc del núcleo puede cambiar; su OID, ACL, owner, argumentos,
 -- resultado, volatility, SECURITY DEFINER y GUC deben permanecer idénticos.
 FOR b IN SELECT * FROM pg_temp.ad3_31_preimagen LOOP
  IF b.identidad IS NULL OR b.metadatos IS NULL OR b.definicion IS NULL
   OR to_regprocedure(b.identidad) IS NULL THEN
   RAISE EXCEPTION 'AD3-31: preimagen incompleta o función previa retirada'; END IF;
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(b.identidad);
  IF b.identidad=nucleo THEN
   IF (to_jsonb(p)-'prosrc') IS DISTINCT FROM (b.metadatos-'prosrc')
    OR p.prosrc IS NOT DISTINCT FROM b.metadatos->>'prosrc' THEN
    RAISE EXCEPTION 'AD3-31: metadatos núcleo alterados o extensión ausente'; END IF;
  ELSIF to_jsonb(p) IS DISTINCT FROM b.metadatos OR pg_get_functiondef(p.oid) IS DISTINCT FROM b.definicion THEN
   RAISE EXCEPTION 'AD3-31: definición/ACL/autoridad previa alterada';
  END IF;
 END LOOP;

 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>propietario OR NOT p.prosecdef OR p.provolatile<>'v'
  OR NOT p.proretset OR p.pronargs<>10 OR p.prorettype<>'record'::regtype
  OR p.proacl IS NULL OR ('search_path=pg_catalog'=ANY(p.proconfig)) IS DISTINCT FROM true
  OR ('lock_timeout=2s'=ANY(p.proconfig)) IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'AD3-31: contrato catalogal del consumidor nuevo incorrecto'; END IF;
 -- ACL explícita exacta: ejecución del propietario AD3 implícita/normal y
 -- concesión únicamente a CT propietario. Sin PUBLIC, grant option ni terceros.
 IF EXISTS(SELECT 1 FROM aclexplode(p.proacl) a
     WHERE a.grantee NOT IN (propietario,ct) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  OR (SELECT count(*) FROM aclexplode(p.proacl) a WHERE a.grantee=ct AND a.privilege_type='EXECUTE')<>1
  OR NOT has_function_privilege(ct,f,'EXECUTE') THEN
  RAISE EXCEPTION 'AD3-31: consumidor nuevo accesible fuera del propietario CT'; END IF;
 FOREACH identidad IN ARRAY ARRAY['vec_autorizacion_atestada_v3_consumidor','vec_autorizacion_atestada_v3_emisor',
  'vec_contratacion_temporal_ejecutor','vec_personal_propietario','vec_personal_ejecutor',
  'vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_ejecutor'] LOOP
  IF has_function_privilege(identidad,f,'EXECUTE') THEN
   RAISE EXCEPTION 'AD3-31: ejecución heredada no prevista para %',identidad; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE oid IN (propietario,ct)
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)) THEN
  RAISE EXCEPTION 'AD3-31: rol propietario privilegiado o con login'; END IF;
 RAISE NOTICE 'AD3-31: catálogo y preimagen comprobados; no acredita consumo V3 dinámico';
END $catalogos$;
ROLLBACK;
