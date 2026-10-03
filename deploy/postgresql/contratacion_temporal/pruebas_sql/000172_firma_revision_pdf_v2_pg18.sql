\set ON_ERROR_STOP on
-- CT172: estructura y rechazo con funciones reales; no sustituye V3 ni K.
-- La cadena favorable requiere fuentes nominales reales de CA/Personal/AUT.
-- Un fixture externo autorizado debe entrar por el LOGIN CT, en otra puerta.
-- No ejecutar esta prueba sobre principal ni sobre historia de producción.
\if :{?ct172_fixture_sql}
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $runtime$
BEGIN
 IF current_setting('role')<>'none' OR current_user<>session_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'CT172 fixture requiere LOGIN runtime CT sin SET ROLE'
   USING ERRCODE='42501'; END IF;
END $runtime$;
-- El fixture no contiene BEGIN/COMMIT/ROLLBACK, SET ROLE ni funciones dobles.
-- Debe comprobar alta/CAS, raíz/entrada exacta, dos firmas, denegaciones sin
-- efecto y recuperación nominal vigente con recibo/fecha/historia idénticos.
-- Replay con decisión V3 nueva exige otra transacción; reinicio exige otro
-- recorrido sobre un clon desechable, nunca se atribuyen a este ROLLBACK.
\i :ct172_fixture_sql
ROLLBACK;
\else
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $estructura$
DECLARE f regprocedure; a record; propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 ejecutor oid:='vec_contratacion_temporal_ejecutor'::regrole;
BEGIN
 IF to_regclass('vec_contratacion_temporal.firma_documento_revision_pdf_v2') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'CT172 ausente o ABI incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
  AND c.relowner=propietario AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND p.polname='propietario' AND p.polroles=ARRAY[propietario])
  OR EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND p.polroles<>ARRAY[propietario])
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass AND a.grantee<>propietario)
  OR EXISTS(SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
   WHERE t.oid=(SELECT reltype FROM pg_class WHERE oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass)
    AND a.grantee<>propietario)
  OR has_table_privilege(ejecutor,'vec_contratacion_temporal.firma_documento_revision_pdf_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN
  RAISE EXCEPTION 'CT172 ACL/RLS abierta' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario
    AND p.prosecdef AND p.provolatile='v'
    AND p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'])
   OR NOT has_function_privilege(ejecutor,f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(propietario,ejecutor) OR a.is_grantable)) THEN
   RAISE EXCEPTION 'CT172 fachada abierta o configuración divergente: %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.impedir_degradacion_firma_pdf_v2()'::regprocedure,
  'vec_contratacion_temporal.comprobar_hija_firma_pdf_v2()'::regprocedure] LOOP
  IF has_function_privilege(ejecutor,f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee<>propietario) THEN
   RAISE EXCEPTION 'CT172 guarda ejecutable desde fuera' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
   AND t.tgname='hija_pdf_v2_ai' AND t.tgdeferrable AND t.tginitdeferred AND t.tgenabled='O')
  OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
   AND t.tgname='impedir_degradacion_pdf_bi' AND t.tgenabled='O')
  OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND t.tgname='revision_pdf_inmutable' AND t.tgenabled='O') THEN
  RAISE EXCEPTION 'CT172 guardas ausentes' USING ERRCODE='55000'; END IF;
END $estructura$;
-- La entrada se rechaza por la fachada real antes de consumir V3. Esta
-- comprobación no acredita una decisión favorable ni la competencia nominal.
CREATE ROLE vec_ct172_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct172_prueba;
CREATE TEMP TABLE ct172_preimagen AS
 SELECT 'firma'::text AS tipo,count(*) AS filas FROM vec_contratacion_temporal.firma_documento_v1
 UNION ALL SELECT 'revision',count(*) FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2
 UNION ALL SELECT 'custodia',count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1
 UNION ALL SELECT 'auditoria',count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1
 UNION ALL SELECT 'outbox',count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1;
SET SESSION AUTHORIZATION vec_ct172_prueba;
DO $negativas$
DECLARE campo text; material jsonb;
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_verificada_v2(
   '{}','2026-10-03T00:00:00Z','\x','\x','\x','\x',1,1,'\x','\x','\x','\x','\x');
  RAISE EXCEPTION 'CT172 aceptó material vacío' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN NULL;
 END;
 -- Casos mínimos para aislar la guarda previa al consumo, sin construir
 -- una firma favorable. Se exige su diagnóstico fijo: otra validación no
 -- puede hacer pasar esta regresión cuando falta la guarda de metadatos.
 FOREACH campo IN ARRAY ARRAY['PuestoFirmanteRef','AmbitoFirmanteRef','ActoCompetenciaRef'] LOOP
  material:=jsonb_build_object('Via','certificado_vec','PuestoFirmanteRef',NULL,
   'AmbitoFirmanteRef',NULL,'ActoCompetenciaRef',NULL)||jsonb_build_object(campo,'ref:ct172:no-acreditada');
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_firma_verificada_v2(
    material::text,'2026-10-03T00:00:00Z','\x',convert_to('{}','UTF8'),'\x','\x',1,1,'\x','\x','\x','\x','\x');
   RAISE EXCEPTION 'CT172 aceptó metadato sin fuente: %',campo USING ERRCODE='55000';
  EXCEPTION WHEN SQLSTATE '22023' THEN
   IF SQLERRM IS DISTINCT FROM 'metadatos nominales no acreditados' THEN
    RAISE EXCEPTION 'CT172 guarda de metadato no comprobada: %',campo USING ERRCODE='55000'; END IF;
  END;
 END LOOP;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2('{}','\x','\x','\x','\x',1,1,'\x','\x','\x','\x');
  RAISE EXCEPTION 'CT172 aceptó consulta vacía' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN NULL;
 END;
END $negativas$;
RESET SESSION AUTHORIZATION;
DO $sin_efectos$
BEGIN
 IF EXISTS(SELECT tipo,filas FROM pg_temp.ct172_preimagen EXCEPT
  (SELECT 'firma',count(*) FROM vec_contratacion_temporal.firma_documento_v1
   UNION ALL SELECT 'revision',count(*) FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2
   UNION ALL SELECT 'custodia',count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1
   UNION ALL SELECT 'auditoria',count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1
   UNION ALL SELECT 'outbox',count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1)) THEN
  RAISE EXCEPTION 'CT172 rechazo produjo efecto' USING ERRCODE='55000'; END IF;
END $sin_efectos$;
ROLLBACK;
\echo 'CT172: estructura y rechazo; favorable/CAS/replay/reinicio nominal NO EJECUTADOS.'
\endif
