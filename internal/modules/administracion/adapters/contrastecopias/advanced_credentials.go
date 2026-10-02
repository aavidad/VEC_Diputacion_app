package contrastecopias

// El verificador nunca transporta los verificadores de contraseña. PostgreSQL
// sella juntos todos los pares rol/verificador ordenados; sólo este digest de
// conjunto se incorpora al agregado privado de roles, nunca a un diagnóstico.
// No añade claves ni una autoridad de autenticación. La custodia y el cifrado
// del Snapshot corresponden al conjunto protegido de la copia.
const rolesSelladosSQL = `SELECT jsonb_build_array('credential_inventory',encode(pg_catalog.sha256(pg_catalog.convert_to((SELECT coalesce(jsonb_agg(jsonb_build_array(a.rolname,a.rolpassword) ORDER BY a.rolname),'[]'::jsonb)::text FROM pg_catalog.pg_authid a),'UTF8')),'hex'))::text WHERE EXISTS(SELECT 1 FROM pg_catalog.pg_authid WHERE rolpassword IS NOT NULL)`
