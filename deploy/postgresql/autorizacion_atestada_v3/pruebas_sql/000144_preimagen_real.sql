\set ON_ERROR_STOP on
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
-- Sólo lectura en el clon causal post142 ya inventariado. No modifica roles.
WITH nucleo AS (
 SELECT p.* FROM pg_proc p WHERE p.oid=
 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
), audiencia AS (
 SELECT pg_get_constraintdef(c.oid,true) d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check'
 AND c.contype='c' AND c.convalidated
)
SELECT jsonb_build_object(
 'server_version',current_setting('server_version'),
 'nucleo_def_sha256',encode(sha256(convert_to(pg_get_functiondef(n.oid),'UTF8')),'hex'),
 'nucleo_prosrc_sha256',encode(sha256(convert_to(n.prosrc,'UTF8')),'hex'),
 'audiencia_sha256',(SELECT encode(sha256(convert_to(d,'UTF8')),'hex') FROM audiencia),
 'proowner',pg_get_userbyid(n.proowner),'proconfig',n.proconfig,
 'prosecdef',n.prosecdef,'proacl',n.proacl::text,
 'perfil_meritos_presente',strpos(n.prosrc,'meritos_hecho_propio_interno')>0,
 'perfil_baremo_ausente',strpos(n.prosrc,'gobierno_borrador_reglas_baremo')=0,
 'roles_baremo',(SELECT jsonb_agg(jsonb_build_object(
   'rol',r.rolname,'login',r.rolcanlogin,'super',r.rolsuper,'createdb',r.rolcreatedb,
   'createrole',r.rolcreaterole,'replication',r.rolreplication,'bypassrls',r.rolbypassrls)
   ORDER BY r.rolname) FROM pg_roles r
   WHERE r.rolname IN('vec_bolsa_reglas_baremo_propietario','vec_bolsa_reglas_baremo_ejecutor_gobierno')),
 'namespace_baremo',to_regnamespace('vec_bolsa_reglas_baremo') IS NOT NULL
) FROM nucleo n;
COMMIT;
