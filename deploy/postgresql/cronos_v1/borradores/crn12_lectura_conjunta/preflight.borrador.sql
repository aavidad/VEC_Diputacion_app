-- CRN12: inventario de metadatos de solo lectura, nunca una puerta de instalación.
-- No ejecutado por el productor. Usar únicamente clon/local y canal autorizado.
-- No llama funciones Cronos/V3, no lee filas personales ni modifica sesión/ACL.
-- No enumera LOGIN, secretos o material de autorizaciones. Ausencia => NULL/false.
-- La postimagen aprobada y las fuentes C3 son dependencias pendientes, siempre.

SELECT 'bloqueado'::text AS estado_crn12,
       false AS autoriza_instalacion,
       false AS consumidor_v3_acordado,
       false AS fuente_c3_acordada,
       false AS postimagen_aprobada;

WITH esperados(nombre) AS (VALUES
  ('vec_cronos_v1_propietario'), ('vec_cronos_v1_ejecutor'),
  ('vec_cronos_v1_migrador'), ('vec_cronos_v1_auditor')
)
SELECT e.nombre, r.oid IS NOT NULL AS existe, r.rolcanlogin,
       r.rolsuper, r.rolbypassrls
FROM esperados e LEFT JOIN pg_catalog.pg_roles r ON r.rolname=e.nombre
ORDER BY e.nombre;

WITH esperadas(nombre) AS (VALUES
  ('programacion_jornada'), ('marcaje_original'), ('permiso_catalogo'),
  ('permiso_solicitud'), ('permiso_estado'), ('permiso_resolucion')
)
SELECT e.nombre, c.oid IS NOT NULL AS existe, c.relkind,
       pg_catalog.pg_get_userbyid(c.relowner) AS propietario,
       c.relrowsecurity, c.relforcerowsecurity,
       (SELECT count(*) FROM pg_catalog.pg_policy p
         WHERE p.polrelid=c.oid) AS numero_politicas,
       (SELECT count(*) FROM pg_catalog.pg_trigger t
         WHERE t.tgrelid=c.oid AND NOT t.tgisinternal
           AND t.tgenabled <> 'D') AS triggers_habilitados
FROM esperadas e
LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='vec_cronos_v1'
LEFT JOIN pg_catalog.pg_class c ON c.relnamespace=n.oid AND c.relname=e.nombre
ORDER BY e.nombre;

-- Inventario de funciones existentes para que Dirección elija postimagen.
-- Un hash MD5 facilita comparación textual; no acredita integridad criptográfica
-- ni sustituye huellas SHA256 aprobadas de definición completa y ACL.
SELECT p.proname,
       pg_catalog.pg_get_function_identity_arguments(p.oid) AS firma,
       pg_catalog.pg_get_function_result(p.oid) AS resultado,
       pg_catalog.pg_get_userbyid(p.proowner) AS propietario,
       p.prosecdef, p.provolatile, p.proconfig,
       pg_catalog.md5(pg_catalog.pg_get_functiondef(p.oid)) AS definicion_md5,
       EXISTS (SELECT 1 FROM pg_catalog.aclexplode(
         coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
         WHERE a.grantee=0 AND a.privilege_type='EXECUTE') AS execute_public
FROM pg_catalog.pg_proc p
JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
WHERE n.nspname='vec_cronos_v1' AND p.proname IN (
  'consultar_libro_saldo_interno_v1','acreditar_empleado_contexto_v1',
  'huella_contexto_empleado_v1','comprobar_decision_cronos_v1',
  'vence_autorizacion_v1','estado_permiso_actual_v1',
  'consumir_propio_v1','consultar_saldo_propio_v1','consultar_permisos_propio_v1')
ORDER BY p.proname, firma;
