\set ON_ERROR_STOP on
-- Inventario de solo lectura: filas de origen AD172 y estado de los LOGIN que
-- nombra la lista. Variable psql `ternas`, la misma que recibe operacion.sql.
-- No contiene secretos ni datos personales.
BEGIN READ ONLY;
SET LOCAL search_path = pg_catalog;
WITH logins AS (
  SELECT DISTINCT (string_to_array(l.linea, E'\t'))[2] AS nombre
    FROM regexp_split_to_table(:'ternas', E'\n') AS l(linea)
   WHERE l.linea <> '' AND left(l.linea, 1) <> '#')
SELECT jsonb_build_object(
 'filas', (SELECT coalesce(jsonb_agg(jsonb_build_array(c.login_nombre, c.audiencia_consumo, c.operacion,
             c.proceso, c.canal_permitido) ORDER BY c.login_nombre, c.audiencia_consumo, c.operacion), '[]'::jsonb)
           FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c),
 'logins', (SELECT coalesce(jsonb_object_agg(n.nombre, (SELECT jsonb_build_object(
             'login', r.rolcanlogin, 'inherit', r.rolinherit,
             'membresias', (SELECT coalesce(jsonb_agg(g.rolname ORDER BY g.rolname), '[]'::jsonb)
                              FROM pg_auth_members m JOIN pg_roles g ON g.oid = m.roleid WHERE m.member = r.oid))
             FROM pg_roles r WHERE r.rolname = n.nombre)), '{}'::jsonb)
           FROM logins n)
)::text;
COMMIT;
