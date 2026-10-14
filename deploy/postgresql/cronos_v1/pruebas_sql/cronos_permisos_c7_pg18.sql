\set ON_ERROR_STOP on
-- Ejecutar en la copia HZ37 tras AD267 y CRN16. Consulta metadatos sin escribir datos.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $comprobar$
DECLARE f regprocedure; t regclass; firma text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
   RAISE EXCEPTION 'C7: PostgreSQL 18 requerido';
 END IF;
 FOREACH firma IN ARRAY ARRAY[
   'vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_bandeja_permisos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_avisos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.archivar_aviso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   f:=to_regprocedure(firma);
   IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prosecdef
       AND 'search_path=pg_catalog, pg_temp'=ANY(p.proconfig))
      OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid=f AND a.grantee=0) THEN
     RAISE EXCEPTION 'C7: función o privilegios incompatibles: %',firma;
   END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY[
   'vec_cronos_v1.permiso_resolutor'::regclass,
   'vec_cronos_v1.permiso_resolutor_retirada'::regclass,
   'vec_cronos_v1.permiso_circuito_directo'::regclass,
   'vec_cronos_v1.permiso_circuito_directo_retirada'::regclass,
   'vec_cronos_v1.permiso_resolucion'::regclass,
   'vec_cronos_v1.permiso_aviso'::regclass] LOOP
   IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=t AND c.relrowsecurity AND c.relforcerowsecurity)
      OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
          WHERE c.oid=t AND a.grantee=0) THEN
     RAISE EXCEPTION 'C7: tabla o RLS incompatibles: %',t;
   END IF;
 END LOOP;
 IF to_regclass('vec_cronos_v1.notificacion') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
   RAISE EXCEPTION 'C7: notificaciones C9 se mezclaron con el circuito';
 END IF;
END $comprobar$;
COMMIT;
