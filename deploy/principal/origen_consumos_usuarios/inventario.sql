\set ON_ERROR_STOP on
-- Inventario de solo lectura: filas de origen AD172 y estado de los LOGIN de
-- Usuarios. No contiene secretos ni datos personales.
BEGIN READ ONLY;
SET LOCAL search_path = pg_catalog;
SELECT jsonb_build_object(
 'filas', (SELECT coalesce(jsonb_agg(jsonb_build_array(c.login_nombre, c.audiencia_consumo, c.operacion,
             c.proceso, c.canal_permitido) ORDER BY c.login_nombre, c.audiencia_consumo, c.operacion), '[]'::jsonb)
           FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c),
 'logins', (SELECT coalesce(jsonb_object_agg(r.rolname, jsonb_build_object(
             'login', r.rolcanlogin, 'inherit', r.rolinherit,
             'membresias', (SELECT coalesce(jsonb_agg(g.rolname ORDER BY g.rolname), '[]'::jsonb)
                              FROM pg_auth_members m JOIN pg_roles g ON g.oid = m.roleid WHERE m.member = r.oid))), '{}'::jsonb)
           FROM pg_roles r WHERE r.rolname IN ('vec_pref508a_i_ue', 'vec_pref508a_e_ue'))
)::text;
COMMIT;
