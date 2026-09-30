\set ON_ERROR_STOP on
SET search_path = pg_catalog;
SET statement_timeout = '30s';
WITH target(key, signature) AS (
  VALUES
    ('ad125_core', 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    ('ad125_consumer', 'vec_autorizacion_atestada_v3.consumir_consulta_firmas_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    ('ct145_register', 'vec_contratacion_temporal.registrar_firma_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    ('ct145_read', 'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)'),
    ('ct152_attested_read', 'vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    ('ct153_reincorporation', 'vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()')
), funcs AS (
  SELECT t.key,
         CASE WHEN p.oid IS NULL THEN NULL ELSE
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
             pg_catalog.jsonb_build_object(
               'definition', pg_catalog.pg_get_functiondef(p.oid),
               'owner', r.rolname,
               'acl', p.proacl::text,
               'config', p.proconfig,
               'security_definer', p.prosecdef,
               'volatility', p.provolatile,
               'kind', p.prokind,
               'args', pg_catalog.pg_get_function_identity_arguments(p.oid),
               'returns', p.prorettype::regtype::text
             )::text, 'UTF8')), 'hex') END AS sha256
    FROM target t
    LEFT JOIN pg_catalog.pg_proc p ON p.oid = pg_catalog.to_regprocedure(t.signature)
    LEFT JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
), ct145_table AS (
  SELECT CASE WHEN c.oid IS NULL OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_policy p
      CROSS JOIN LATERAL pg_catalog.unnest(p.polroles) u(role_oid)
      LEFT JOIN pg_catalog.pg_roles r_policy ON r_policy.oid=u.role_oid
      WHERE p.polrelid=c.oid AND u.role_oid<>0 AND r_policy.oid IS NULL
    ) THEN NULL ELSE
    pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.jsonb_build_object(
        'owner', r.rolname,
        'acl', c.relacl::text,
        'rls', c.relrowsecurity,
        'force_rls', c.relforcerowsecurity,
        'kind', c.relkind,
        'columns', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(a.attnum, a.attname,
            pg_catalog.format_type(a.atttypid,a.atttypmod), a.attnotnull,
            a.attidentity, a.attgenerated,
            pg_catalog.pg_get_expr(d.adbin,d.adrelid)) ORDER BY a.attnum), '[]'::jsonb)
          FROM pg_catalog.pg_attribute a LEFT JOIN pg_catalog.pg_attrdef d
            ON d.adrelid=a.attrelid AND d.adnum=a.attnum
          WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped),
        'constraints', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(x.conname, x.contype,
            pg_catalog.pg_get_constraintdef(x.oid,true)) ORDER BY x.conname), '[]'::jsonb)
          FROM pg_catalog.pg_constraint x WHERE x.conrelid=c.oid),
        'indexes', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.pg_get_indexdef(i.indexrelid) ORDER BY i.indexrelid::regclass::text), '[]'::jsonb)
          FROM pg_catalog.pg_index i WHERE i.indrelid=c.oid),
        'policies', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(p.polname,p.polcmd,p.polpermissive,
            (SELECT coalesce(pg_catalog.jsonb_agg(
                CASE WHEN u.role_oid=0 THEN pg_catalog.jsonb_build_array('PUBLIC')
                     ELSE pg_catalog.jsonb_build_array('ROLE', r_policy.rolname) END
                ORDER BY CASE WHEN u.role_oid=0 THEN 'PUBLIC' ELSE r_policy.rolname END,
                         u.role_oid=0),
                '[]'::jsonb)
             FROM pg_catalog.unnest(p.polroles) u(role_oid)
             LEFT JOIN pg_catalog.pg_roles r_policy ON r_policy.oid=u.role_oid),
            pg_catalog.pg_get_expr(p.polqual,p.polrelid),
            pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)) ORDER BY p.polname), '[]'::jsonb)
          FROM pg_catalog.pg_policy p WHERE p.polrelid=c.oid),
        'triggers', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(t.tgname,t.tgenabled,
            pg_catalog.pg_get_triggerdef(t.oid,true)) ORDER BY t.tgname), '[]'::jsonb)
          FROM pg_catalog.pg_trigger t WHERE t.tgrelid=c.oid AND NOT t.tgisinternal)
      )::text, 'UTF8')), 'hex') END AS sha256
    FROM (SELECT 'vec_contratacion_temporal.firma_documento_custodia_v1'::regclass AS oid) x
    LEFT JOIN pg_catalog.pg_class c ON c.oid=x.oid
    LEFT JOIN pg_catalog.pg_roles r ON r.oid=c.relowner
)
SELECT pg_catalog.jsonb_build_object(
  'functions', (SELECT pg_catalog.jsonb_object_agg(key,sha256 ORDER BY key) FROM funcs),
  'ct145_table', (SELECT sha256 FROM ct145_table));
