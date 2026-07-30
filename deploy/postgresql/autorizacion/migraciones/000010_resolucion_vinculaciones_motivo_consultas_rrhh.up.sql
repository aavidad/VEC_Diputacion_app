-- Resolución nominal de las vinculaciones de motivo para consultas RRHH.
-- No acepta selectores: cuadro y detalle son dos capacidades distintas.
BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

-- M1.R precede a las migraciones. Los fundamentos se bloquean en el orden
-- contractual y 000010 conserva el bloqueo exclusivo durante la instalación.
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:rol-motivos-rrhh-resolutor:v1', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000010', 0));

SET LOCAL ROLE vec_autorizacion_propietario;

LOCK TABLE
    vec_autorizacion.motivo_v2_evento_origen,
    vec_autorizacion.motivo_v2_catalogo_publicado,
    vec_autorizacion.motivo_v2_entrada,
    vec_autorizacion.motivo_v2_retirada,
    vec_autorizacion.motivo_v2_checkpoint_origen,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    IN ACCESS SHARE MODE;

DO $prevalidacion$
DECLARE
    tablas regclass[] := ARRAY[
      'vec_autorizacion.motivo_v2_evento_origen'::regclass,
      'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,
      'vec_autorizacion.motivo_v2_entrada'::regclass,
      'vec_autorizacion.motivo_v2_retirada'::regclass,
      'vec_autorizacion.motivo_v2_checkpoint_origen'::regclass,
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass
    ];
    nombres_funciones name[] := ARRAY[
      'bloquear_mutacion_vinculacion_motivo_rrhh_v1',
      'validar_avance_vinculacion_motivo_rrhh_v1',
      'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1',
      'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
      'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1',
      'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
      'publicar_vinculacion_motivo_cuadro_rrhh_v1',
      'publicar_vinculacion_motivo_detalle_rrhh_v1',
      'retirar_vinculacion_motivo_cuadro_rrhh_v1',
      'retirar_vinculacion_motivo_detalle_rrhh_v1'
    ];
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer / 10000 <> 18
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_autorizacion'
              AND nspowner = 'vec_autorizacion_propietario'::regrole
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_motivos_rrhh_resolutor'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
              AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
              AND NOT rolbypassrls AND rolconnlimit = -1
              AND (rolpassword IS NULL OR rolpassword='********')
              AND rolvaliduntil IS NULL
              AND pg_catalog.shobj_description(oid, 'pg_authid') =
                  'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting
            WHERE setrole =
                  (SELECT oid FROM pg_catalog.pg_roles
                    WHERE rolname =
                          'vec_autorizacion_motivos_rrhh_resolutor')
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
            WHERE (m.member =
                     'vec_autorizacion_motivos_rrhh_resolutor'::regrole
                   OR m.grantor =
                     'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
               OR (m.roleid =
                     'vec_autorizacion_motivos_rrhh_resolutor'::regrole
                   AND (m.admin_option OR NOT m.inherit_option
                        OR m.set_option))
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;

    IF EXISTS (
           WITH rol AS (
             SELECT oid FROM pg_catalog.pg_roles
              WHERE rolname='vec_autorizacion_motivos_rrhh_resolutor'
           ), actual AS (
             SELECT d.datname,a.grantor,a.grantee,a.privilege_type,
                    a.is_grantable
               FROM pg_catalog.pg_database AS d
               CROSS JOIN LATERAL pg_catalog.aclexplode(d.datacl) AS a,rol
              WHERE a.grantee=rol.oid OR a.grantor=rol.oid
           ), esperado AS (
             SELECT d.datname,d.datdba,rol.oid,'CONNECT'::text,false
               FROM pg_catalog.pg_database AS d,rol
              WHERE d.datname=pg_catalog.current_database()
           )
           (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
           UNION ALL
           (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace AS n
           CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) AS a
           WHERE a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
           CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
           WHERE a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute AS atributo
           CROSS JOIN LATERAL pg_catalog.aclexplode(atributo.attacl) AS a
           WHERE a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
           CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
           WHERE a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_type AS t
           CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
           WHERE a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl AS d
           CROSS JOIN LATERAL pg_catalog.aclexplode(d.defaclacl) AS a
           WHERE d.defaclrole=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantee=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole
              OR a.grantor=
                 'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_database
            WHERE datdba=
                  'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspowner=
                  'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class
            WHERE relowner=
                  'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc
            WHERE proowner=
                  'vec_autorizacion_motivos_rrhh_resolutor'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_type
            WHERE typowner=
                  'vec_autorizacion_motivos_rrhh_resolutor'::regrole) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
         WHERE p.pronamespace = 'vec_autorizacion'::regnamespace
           AND p.proname = ANY (ARRAY[
             'resolver_motivo_cuadro_rrhh_v1',
             'resolver_motivo_detalle_rrhh_v1'
           ]::name[])
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class AS c
         WHERE c.oid = ANY (tablas)
           AND c.relkind = 'r' AND c.relpersistence = 'p'
           AND c.relam = (
               SELECT oid FROM pg_catalog.pg_am
                WHERE amname = 'heap' AND amtype = 't')
           AND NOT c.relispartition
           AND c.relowner = 'vec_autorizacion_propietario'::regrole
           AND c.relrowsecurity AND c.relforcerowsecurity) <> 8
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_inherits
            WHERE inhrelid = ANY (tablas) OR inhparent = ANY (tablas))
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_rewrite
            WHERE ev_class = ANY (tablas))
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid = ANY (tablas)
              AND polname = 'acceso_propietario_exacto'
              AND polpermissive AND polcmd = '*'
              AND polroles =
                  ARRAY['vec_autorizacion_propietario'::regrole::oid]
              AND pg_catalog.pg_get_expr(polqual, polrelid) =
                  '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
              AND pg_catalog.pg_get_expr(polwithcheck, polrelid) =
                  '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
          ) <> 8
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid = ANY (tablas)) <> 8
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
           CROSS JOIN LATERAL pg_catalog.aclexplode(
             COALESCE(c.relacl, pg_catalog.acldefault('r', c.relowner))) AS a
           WHERE c.oid = ANY (tablas) AND a.grantee <> c.relowner)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute AS a
           CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS permiso
           WHERE a.attrelid = ANY (tablas) AND a.attnum > 0
             AND permiso.grantee <>
                 'vec_autorizacion_propietario'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_type AS t
           CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS permiso
           WHERE t.typrelid = ANY (tablas)
             AND permiso.grantee <> t.typowner)
       OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(c.oid::regclass::text,a.attnum,a.attname,a.atttypid::regtype::text,a.atttypmod,a.attnotnull,a.attidentity,a.attgenerated,a.attcollation::regcollation::text,a.attstorage,a.attcompression,a.attisdropped,pg_catalog.pg_get_expr(d.adbin,d.adrelid,true)) ORDER BY c.oid::regclass::text,a.attnum)::text,'UTF8')),'hex')
             FROM pg_catalog.pg_class c JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 LEFT JOIN pg_catalog.pg_attrdef d ON (d.adrelid,d.adnum)=(a.attrelid,a.attnum)
            WHERE c.oid=ANY(tablas))<>'89c15ed702c1897c9e99b4fc5e2f030182cb1324e1a7a65f91d820102a5b83e8' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;

    IF EXISTS (
        WITH esperado(tabla,nombre,tipo,definicion,marca) AS (VALUES
          ('vec_autorizacion.motivo_v2_evento_origen'::regclass,'motivo_v2_evento_coordenadas_unicas','u','UNIQUE (secuencia_origen, evento_origen_ref)',NULL::text),
          ('vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,'motivo_v2_catalogo_evento_fk','f','FOREIGN KEY (secuencia_origen, evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)',NULL::text),
          ('vec_autorizacion.motivo_v2_entrada'::regclass,'motivo_v2_entrada_catalogo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version)',NULL::text),
          ('vec_autorizacion.motivo_v2_retirada'::regclass,'motivo_v2_retirada_catalogo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version)',NULL::text),
          ('vec_autorizacion.motivo_v2_retirada'::regclass,'motivo_v2_retirada_evento_fk','f','FOREIGN KEY (secuencia_origen, evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)',NULL::text),
          ('vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,'motivo_v2_catalogo_referencia_completa_unica','u','UNIQUE (catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)','vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008'),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_pk','p','PRIMARY KEY (clase_consulta, publicacion_version)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_publicacion_ref_unica','u','UNIQUE (publicacion_ref)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_publicacion_huella_unica','u','UNIQUE (publicacion_huella_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_clase_cerrada','c','CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_version_positiva','c','CHECK (publicacion_version > 0)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_ref_opaca','c','CHECK (publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_huellas_validas','c','CHECK (publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND publicacion_huella_sha256 <> repeat(''0''::text, 64) AND catalogo_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND catalogo_huella_sha256 <> repeat(''0''::text, 64))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_entrada_opaca','c','CHECK (entrada_clave ~ ''^motivo_[0-9a-f]{32}$''::text)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_instantes_validos','c','CHECK (isfinite(publicada_en) AND isfinite(registrada_en) AND publicada_en <= registrada_en AND EXTRACT(year FROM (publicada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (publicada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_publicacion_completa_unica','u','UNIQUE (clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_catalogo_completo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version, catalogo_huella_sha256) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_entrada_fk','f','FOREIGN KEY (catalogo_id, catalogo_version, entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id, catalogo_version, entrada_clave)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_pk','p','PRIMARY KEY (clase_consulta)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_clase_cerrada','c','CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_completo','c','CHECK (ultima_publicacion_version = 0 AND ultima_publicacion_ref IS NULL AND ultima_publicacion_huella_sha256 IS NULL OR ultima_publicacion_version > 0 AND ultima_publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text AND ultima_publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND ultima_publicacion_huella_sha256 <> repeat(''0''::text, 64))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_instante_valido','c','CHECK (isfinite(actualizado_en) AND EXTRACT(year FROM (actualizado_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (actualizado_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_historia_fk','f','FOREIGN KEY (clase_consulta, ultima_publicacion_version, ultima_publicacion_ref, ultima_publicacion_huella_sha256) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_pk','p','PRIMARY KEY (clase_consulta, operacion, publicacion_version)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_ref_unica','u','UNIQUE (evento_ref)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_huella_unica','u','UNIQUE (evento_huella_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_clase_cerrada','c','CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_operacion_cerrada','c','CHECK (operacion = ANY (ARRAY[''publicacion''::text, ''retirada''::text]))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_version_positiva','c','CHECK (publicacion_version > 0)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_refs_validas','c','CHECK (evento_ref ~ ''^evento_vinculacion_motivo_rrhh_[0-9a-f]{32}$''::text AND publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text AND prueba_vec_evento_origen_ref ~ ''^evento_[0-9a-f]{32}$''::text)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_huellas_validas','c','CHECK (evento_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND evento_huella_sha256 <> repeat(''0''::text, 64) AND publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND publicacion_huella_sha256 <> repeat(''0''::text, 64) AND prueba_vec_evento_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND prueba_vec_evento_huella_sha256 <> repeat(''0''::text, 64))',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_actor_tecnico','c','CHECK (char_length(actor_tecnico_ref) >= 1 AND char_length(actor_tecnico_ref) <= 63)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_prueba_secuencia','c','CHECK (prueba_vec_secuencia_origen > 0)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_instantes','c','CHECK (isfinite(ocurrida_en) AND isfinite(prueba_vec_validada_en) AND isfinite(registrada_en) AND ocurrida_en <= registrada_en AND prueba_vec_validada_en <= registrada_en AND EXTRACT(year FROM (ocurrida_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (ocurrida_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (prueba_vec_validada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (prueba_vec_validada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_txid','c','CHECK (registrada_txid > 0)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_publicacion_fk','f','FOREIGN KEY (clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)',NULL::text),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_prueba_vec_fk','f','FOREIGN KEY (prueba_vec_secuencia_origen, prueba_vec_evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)',NULL::text)
        ), esperado_exacto AS (
          SELECT e.*,true,false,false,(e.tipo<>'c') FROM esperado e
        ), actual AS (
          SELECT c.conrelid,c.conname::text,c.contype::text,
                 pg_catalog.pg_get_constraintdef(c.oid,true),
                 pg_catalog.obj_description(c.oid,'pg_constraint'),
                 c.convalidated,c.condeferrable,c.condeferred,c.connoinherit
            FROM pg_catalog.pg_constraint AS c
           WHERE c.contype<>'n'
             AND (c.conrelid=ANY(ARRAY[
               'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
               'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,
               'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass])
               OR EXISTS (SELECT 1 FROM esperado e
                           WHERE (e.tabla,e.nombre)=(c.conrelid,c.conname)))
        ), claves AS (
          SELECT c.oid,c.conrelid,c.confrelid,c.conindid
            FROM pg_catalog.pg_constraint c JOIN esperado e
              ON (e.tabla,e.nombre)=(c.conrelid,c.conname) WHERE e.tipo='f'
        ), ri_esperado AS (
          SELECT c.oid,x.tabla,x.otra,c.conindid,x.tipo,'O',true,x.funcion,true
            FROM claves c CROSS JOIN LATERAL (VALUES
              (c.conrelid,c.confrelid,5::smallint,'RI_FKey_check_ins'),
              (c.conrelid,c.confrelid,17::smallint,'RI_FKey_check_upd'),
              (c.confrelid,c.conrelid,9::smallint,'RI_FKey_noaction_del'),
              (c.confrelid,c.conrelid,17::smallint,'RI_FKey_noaction_upd'))
              x(tabla,otra,tipo,funcion)
        ), ri_actual AS (
          SELECT t.tgconstraint,t.tgrelid,t.tgconstrrelid,t.tgconstrindid,t.tgtype,
                 t.tgenabled::text,t.tgisinternal,p.proname::text,
                 p.pronamespace='pg_catalog'::regnamespace AND
                 ROW(t.tgparentid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
                     t.tgattr::text,pg_catalog.encode(t.tgargs,'hex'),
                     t.tgqual IS NULL,t.tgoldtable IS NULL,t.tgnewtable IS NULL)=
                 ROW(0::oid,false,false,0,'','',true,true,true)
            FROM pg_catalog.pg_trigger t JOIN pg_catalog.pg_proc p ON p.oid=t.tgfoid
           WHERE t.tgconstraint=ANY(ARRAY(SELECT oid FROM claves))
              OR (t.tgisinternal AND (t.tgrelid=ANY(tablas[6:8])
                                      OR t.tgconstrrelid=ANY(tablas[6:8])))
        ), propios_esperados(tabla,nombre,tipo,funcion,habilitado,interno,exacto) AS (VALUES
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_inmutable',27::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_no_truncar',34::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_avance',19::smallint,'vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_inmutable',15::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_no_truncar',34::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_validar',7::smallint,'vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_inmutable',27::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure,'O',false,true),
          ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass,'vinculacion_motivo_rrhh_evento_no_truncar',34::smallint,'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()'::regprocedure,'O',false,true)
        ), propios_actuales AS (
          SELECT t.tgrelid,t.tgname::text,t.tgtype,t.tgfoid,t.tgenabled::text,
                 t.tgisinternal,ROW(t.tgparentid,t.tgconstraint,t.tgconstrrelid,
                   t.tgconstrindid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
                   t.tgattr::text,pg_catalog.encode(t.tgargs,'hex'),t.tgqual IS NULL,
                   t.tgoldtable IS NULL,t.tgnewtable IS NULL)=
                   ROW(0::oid,0::oid,0::oid,0::oid,false,false,0,'','',true,true,true)
            FROM pg_catalog.pg_trigger t
           WHERE t.tgrelid=ANY(ARRAY[
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass])
             AND NOT t.tgisinternal
        ), diferencia AS (
          SELECT 1 FROM (SELECT * FROM esperado_exacto EXCEPT ALL SELECT * FROM actual) d
          UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado_exacto) d
          UNION ALL SELECT 1 FROM (SELECT * FROM ri_esperado EXCEPT ALL SELECT * FROM ri_actual) d
          UNION ALL SELECT 1 FROM (SELECT * FROM ri_actual EXCEPT ALL SELECT * FROM ri_esperado) d
          UNION ALL SELECT 1 FROM (SELECT * FROM propios_esperados EXCEPT ALL SELECT * FROM propios_actuales) d
          UNION ALL SELECT 1 FROM (SELECT * FROM propios_actuales EXCEPT ALL SELECT * FROM propios_esperados) d
        ) SELECT 1 FROM diferencia
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;

    IF EXISTS (
      WITH esperado(objeto,identidad,expuesta) AS (VALUES
        ('vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()','7d2c390e2c17c3d4eaca7da98d8c767f40707d81203fef9897be745b929143a7',false),
        ('vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()','73e9bbb667fa19234325b8b90acd6a8759b4cde136cae94e8b87dd3182317182',false),
        ('vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','822fd8c60a85e305cd9f87e9d2fd366db6261b0f1344498e93c9346bbfc1181c',true),
        ('vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','025ce8b53f6c2bd9f68f410921b98fb0edbaab9ba2af66426ad543c2abdc4de5',true),
        ('vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','eebea97900a4cecba6900f9908499b58c047ea6888478f0573c745964c08fa83',false),
        ('vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,timestamp with time zone)','7bbb751a6963dd4e503d7607e73d3a5a61be3583f1b36f695c78f610b2fa1c18',false),
        ('vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)','3a3c5460c4eeebf945613990a2589f26085f6c535968a14eebc00ed08373c778',true),
        ('vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)','32cc411780f0044ce8262d4367a390cef63c075a6e438dd4700e917606177626',true),
        ('vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()','3dffb1171f1d2e766ec927abeef5e8dc0de6801256845cc2c2fd930597c597a5',false),
        ('vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()','b1fb4729e54a7d27c584055376db7d9ff738769eaaac68295aa50299d96f71f8',false)
      ), actual AS (
        SELECT p.oid::regprocedure::text,pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(pg_catalog.jsonb_build_array(l.lanname,p.proowner::regrole::text,p.prokind::text,p.provolatile::text,p.proparallel::text,p.prosecdef,p.proleakproof,p.proisstrict,p.proretset,p.pronargs,p.pronargdefaults,p.prorettype::regtype::text,p.proargtypes::text,p.proallargtypes,p.proargmodes,p.proargnames,p.protrftypes,p.provariadic::regtype::text,p.proconfig,p.prosrc,p.probin,p.prosqlbody::text,p.procost,p.prorows,p.prosupport::regprocedure::text,pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex'),p.prosecdef
          FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
         WHERE p.pronamespace='vec_autorizacion'::regnamespace
           AND p.proname=ANY(nombres_funciones)
      ), acl_esperada AS (
        SELECT e.objeto,'vec_autorizacion_propietario'::regrole::oid,
               'vec_autorizacion_propietario'::regrole::oid,'EXECUTE'::text,false
          FROM esperado e UNION ALL
        SELECT e.objeto,'vec_autorizacion_propietario'::regrole::oid,
               'vec_autorizacion_motivos_proyector'::regrole::oid,'EXECUTE'::text,false
          FROM esperado e WHERE e.expuesta
      ), acl_actual AS (
        SELECT p.oid::regprocedure::text,a.grantor,a.grantee,
               a.privilege_type,a.is_grantable
          FROM pg_catalog.pg_proc p
          CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
         WHERE p.pronamespace='vec_autorizacion'::regnamespace
           AND p.proname=ANY(nombres_funciones)
      ), diferencia AS (
        SELECT 1 FROM (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual) d
        UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado) d
        UNION ALL SELECT 1 FROM (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual) d
        UNION ALL SELECT 1 FROM (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada) d
      ) SELECT 1 FROM diferencia
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'instalación de resolución RRHH rechazada';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(
    p_instante timestamptz)
RETURNS TABLE (
    catalogo_id text,
    catalogo_version integer,
    catalogo_huella_sha256 text,
    entrada_clave text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_actor text := session_user::text;
    v_entrada timestamptz := pg_catalog.clock_timestamp();
    v_actual timestamptz;
    v_checkpoint record;
BEGIN
    IF current_user IS DISTINCT FROM 'vec_autorizacion_propietario'
       OR p_instante IS NULL OR pg_catalog.isfinite(p_instante) IS NOT TRUE
       OR extract(year FROM (p_instante AT TIME ZONE 'UTC'))
            NOT BETWEEN 1 AND 9999
       OR pg_catalog.date_trunc('microseconds', p_instante) IS DISTINCT FROM
          p_instante
       OR p_instante > v_entrada THEN
        RETURN;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:rol-motivos-rrhh-resolutor:v1',0));
    IF (SELECT (pg_catalog.count(*)=1 AND pg_catalog.bool_and(
          r.rolcanlogin AND r.rolinherit
          AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
          AND NOT r.rolreplication AND NOT r.rolbypassrls
          AND grupo.rolname='vec_autorizacion_motivos_rrhh_resolutor'
          AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper
          AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
          AND NOT grupo.rolinherit AND NOT grupo.rolreplication
          AND NOT grupo.rolbypassrls AND grupo.rolconnlimit=-1
          AND (grupo.rolpassword IS NULL OR grupo.rolpassword='********')
          AND grupo.rolvaliduntil IS NULL AND grupo.rolconfig IS NULL
          AND pg_catalog.shobj_description(grupo.oid,'pg_authid')=
              'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
          AND NOT EXISTS (
              SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
               WHERE ajuste.setrole=grupo.oid)
          AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
          AND m.grantor=10)) IS TRUE
          FROM pg_catalog.pg_roles AS r
          JOIN pg_catalog.pg_auth_members AS m ON m.member=r.oid
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid=m.roleid
         WHERE r.rolname=v_actor) IS NOT TRUE
       OR NOT EXISTS (
          SELECT 1 FROM pg_catalog.pg_roles
           WHERE oid=10 AND rolsuper) THEN RETURN; END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000010',0));
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id FOR SHARE;
    IF NOT FOUND THEN RETURN; END IF;
    SELECT * INTO v_checkpoint
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
     WHERE clase_consulta='cuadro' FOR SHARE;
    IF NOT FOUND OR v_checkpoint.ultima_publicacion_version=0 THEN RETURN; END IF;
    v_actual := pg_catalog.clock_timestamp();
    RETURN QUERY
    SELECT h.catalogo_id,h.catalogo_version,h.catalogo_huella_sha256,
           h.entrada_clave
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 cp
      JOIN vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 h
        ON (h.clase_consulta,h.publicacion_version,h.publicacion_ref,
            h.publicacion_huella_sha256)=
           (cp.clase_consulta,cp.ultima_publicacion_version,
            cp.ultima_publicacion_ref,cp.ultima_publicacion_huella_sha256)
      JOIN vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 e
        ON (e.clase_consulta,e.publicacion_version,e.publicacion_ref,
            e.publicacion_huella_sha256)=
           (h.clase_consulta,h.publicacion_version,h.publicacion_ref,
            h.publicacion_huella_sha256)
      JOIN vec_autorizacion.motivo_v2_catalogo_publicado c
        ON (c.catalogo_id,c.catalogo_version,
            c.catalogo_huella_publicada_sha256)=
           (h.catalogo_id,h.catalogo_version,h.catalogo_huella_sha256)
      JOIN vec_autorizacion.motivo_v2_entrada entrada
        ON (entrada.catalogo_id,entrada.catalogo_version,
            entrada.entrada_clave)=
           (h.catalogo_id,h.catalogo_version,h.entrada_clave)
      JOIN vec_autorizacion.motivo_v2_evento_origen prueba
        ON (prueba.secuencia_origen,prueba.evento_origen_ref,
            prueba.huella_evento_sha256)=
           (e.prueba_vec_secuencia_origen,e.prueba_vec_evento_origen_ref,
            e.prueba_vec_evento_huella_sha256)
     WHERE cp.clase_consulta='cuadro'
       AND cp.ultima_publicacion_version=v_checkpoint.ultima_publicacion_version
       AND cp.ultima_publicacion_ref=v_checkpoint.ultima_publicacion_ref
       AND cp.ultima_publicacion_huella_sha256=
           v_checkpoint.ultima_publicacion_huella_sha256
       AND e.operacion='publicacion' AND e.ocurrida_en=h.publicada_en
       AND e.ocurrida_en<=p_instante
       AND e.prueba_vec_validada_en>=e.ocurrida_en
       AND prueba.tipo_evento='publicacion'
       AND (prueba.catalogo_id,prueba.catalogo_version)=
           (c.catalogo_id,c.catalogo_version)
       AND (c.secuencia_origen,c.evento_origen_ref)=
           (e.prueba_vec_secuencia_origen,e.prueba_vec_evento_origen_ref)
       AND c.publicado_en<=h.publicada_en
       AND entrada.vigente_desde<=h.publicada_en
       AND (entrada.vigente_hasta IS NULL
            OR h.publicada_en<entrada.vigente_hasta)
       AND c.publicado_en<=p_instante AND c.publicado_en<=v_actual
       AND entrada.vigente_desde<=p_instante
       AND (entrada.vigente_hasta IS NULL
            OR p_instante<entrada.vigente_hasta)
       AND entrada.vigente_desde<=v_actual
       AND (entrada.vigente_hasta IS NULL OR v_actual<entrada.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 r
            WHERE r.clase_consulta='cuadro' AND r.operacion='retirada'
              AND r.publicacion_version=h.publicacion_version
              AND r.publicacion_ref=h.publicacion_ref
              AND r.publicacion_huella_sha256=h.publicacion_huella_sha256)
       AND NOT EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
            WHERE (r.catalogo_id,r.catalogo_version)=
                  (c.catalogo_id,c.catalogo_version)
              AND r.retirado_en<=p_instante)
       AND NOT EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
            WHERE (r.catalogo_id,r.catalogo_version)=
                  (c.catalogo_id,c.catalogo_version)
              AND r.retirado_en<=v_actual);
END
$funcion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(
    p_instante timestamptz)
RETURNS TABLE (
    catalogo_id text,
    catalogo_version integer,
    catalogo_huella_sha256 text,
    entrada_clave text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_actor text := session_user::text;
    v_entrada timestamptz := pg_catalog.clock_timestamp();
    v_actual timestamptz;
    v_checkpoint record;
BEGIN
    IF current_user IS DISTINCT FROM 'vec_autorizacion_propietario'
       OR p_instante IS NULL OR pg_catalog.isfinite(p_instante) IS NOT TRUE
       OR extract(year FROM (p_instante AT TIME ZONE 'UTC'))
            NOT BETWEEN 1 AND 9999
       OR pg_catalog.date_trunc('microseconds', p_instante) IS DISTINCT FROM
          p_instante
       OR p_instante > v_entrada THEN
        RETURN;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:rol-motivos-rrhh-resolutor:v1',0));
    IF (SELECT (pg_catalog.count(*)=1 AND pg_catalog.bool_and(
          r.rolcanlogin AND r.rolinherit
          AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
          AND NOT r.rolreplication AND NOT r.rolbypassrls
          AND grupo.rolname='vec_autorizacion_motivos_rrhh_resolutor'
          AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper
          AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
          AND NOT grupo.rolinherit AND NOT grupo.rolreplication
          AND NOT grupo.rolbypassrls AND grupo.rolconnlimit=-1
          AND (grupo.rolpassword IS NULL OR grupo.rolpassword='********')
          AND grupo.rolvaliduntil IS NULL AND grupo.rolconfig IS NULL
          AND pg_catalog.shobj_description(grupo.oid,'pg_authid')=
              'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
          AND NOT EXISTS (
              SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
               WHERE ajuste.setrole=grupo.oid)
          AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
          AND m.grantor=10)) IS TRUE
          FROM pg_catalog.pg_roles AS r
          JOIN pg_catalog.pg_auth_members AS m ON m.member=r.oid
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid=m.roleid
         WHERE r.rolname=v_actor) IS NOT TRUE
       OR NOT EXISTS (
          SELECT 1 FROM pg_catalog.pg_roles
           WHERE oid=10 AND rolsuper) THEN RETURN; END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
      pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000010',0));
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id FOR SHARE;
    IF NOT FOUND THEN RETURN; END IF;
    SELECT * INTO v_checkpoint
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
     WHERE clase_consulta='detalle' FOR SHARE;
    IF NOT FOUND OR v_checkpoint.ultima_publicacion_version=0 THEN RETURN; END IF;
    v_actual := pg_catalog.clock_timestamp();
    RETURN QUERY
    SELECT h.catalogo_id,h.catalogo_version,h.catalogo_huella_sha256,
           h.entrada_clave
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 cp
      JOIN vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 h
        ON (h.clase_consulta,h.publicacion_version,h.publicacion_ref,
            h.publicacion_huella_sha256)=
           (cp.clase_consulta,cp.ultima_publicacion_version,
            cp.ultima_publicacion_ref,cp.ultima_publicacion_huella_sha256)
      JOIN vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 e
        ON (e.clase_consulta,e.publicacion_version,e.publicacion_ref,
            e.publicacion_huella_sha256)=
           (h.clase_consulta,h.publicacion_version,h.publicacion_ref,
            h.publicacion_huella_sha256)
      JOIN vec_autorizacion.motivo_v2_catalogo_publicado c
        ON (c.catalogo_id,c.catalogo_version,
            c.catalogo_huella_publicada_sha256)=
           (h.catalogo_id,h.catalogo_version,h.catalogo_huella_sha256)
      JOIN vec_autorizacion.motivo_v2_entrada entrada
        ON (entrada.catalogo_id,entrada.catalogo_version,
            entrada.entrada_clave)=
           (h.catalogo_id,h.catalogo_version,h.entrada_clave)
      JOIN vec_autorizacion.motivo_v2_evento_origen prueba
        ON (prueba.secuencia_origen,prueba.evento_origen_ref,
            prueba.huella_evento_sha256)=
           (e.prueba_vec_secuencia_origen,e.prueba_vec_evento_origen_ref,
            e.prueba_vec_evento_huella_sha256)
     WHERE cp.clase_consulta='detalle'
       AND cp.ultima_publicacion_version=v_checkpoint.ultima_publicacion_version
       AND cp.ultima_publicacion_ref=v_checkpoint.ultima_publicacion_ref
       AND cp.ultima_publicacion_huella_sha256=
           v_checkpoint.ultima_publicacion_huella_sha256
       AND e.operacion='publicacion' AND e.ocurrida_en=h.publicada_en
       AND e.ocurrida_en<=p_instante
       AND e.prueba_vec_validada_en>=e.ocurrida_en
       AND prueba.tipo_evento='publicacion'
       AND (prueba.catalogo_id,prueba.catalogo_version)=
           (c.catalogo_id,c.catalogo_version)
       AND (c.secuencia_origen,c.evento_origen_ref)=
           (e.prueba_vec_secuencia_origen,e.prueba_vec_evento_origen_ref)
       AND c.publicado_en<=h.publicada_en
       AND entrada.vigente_desde<=h.publicada_en
       AND (entrada.vigente_hasta IS NULL
            OR h.publicada_en<entrada.vigente_hasta)
       AND c.publicado_en<=p_instante AND c.publicado_en<=v_actual
       AND entrada.vigente_desde<=p_instante
       AND (entrada.vigente_hasta IS NULL
            OR p_instante<entrada.vigente_hasta)
       AND entrada.vigente_desde<=v_actual
       AND (entrada.vigente_hasta IS NULL OR v_actual<entrada.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 r
            WHERE r.clase_consulta='detalle' AND r.operacion='retirada'
              AND r.publicacion_version=h.publicacion_version
              AND r.publicacion_ref=h.publicacion_ref
              AND r.publicacion_huella_sha256=h.publicacion_huella_sha256)
       AND NOT EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
            WHERE (r.catalogo_id,r.catalogo_version)=
                  (c.catalogo_id,c.catalogo_version)
              AND r.retirado_en<=p_instante)
       AND NOT EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
            WHERE (r.catalogo_id,r.catalogo_version)=
                  (c.catalogo_id,c.catalogo_version)
              AND r.retirado_en<=v_actual);
END
$funcion$;

COMMENT ON FUNCTION vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(
    timestamptz) IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:resolver-cuadro-v1:000010';
COMMENT ON FUNCTION vec_autorizacion.resolver_motivo_detalle_rrhh_v1(
    timestamptz) IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:resolver-detalle-v1:000010';

REVOKE ALL ON FUNCTION
    vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz),
    vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
    FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador,
         vec_autorizacion_motivos_rrhh_resolutor;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_motivos_rrhh_resolutor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz),
    vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
    TO vec_autorizacion_motivos_rrhh_resolutor;

DO $postvalidacion$
DECLARE
  rol oid := 'vec_autorizacion_motivos_rrhh_resolutor'::regrole;
  propietario oid := 'vec_autorizacion_propietario'::regrole;
  funciones oid[] := ARRAY[
    'vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)'::regprocedure,
    'vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)'::regprocedure];
BEGIN
  IF EXISTS (
    WITH esperado(objeto,identidad,cuerpo,longitud) AS (VALUES
      ('vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamp with time zone)','d92704658d0af8acea83cd765e02976561c787a95906b9a10ee8a43ac0be16ef','a3d784e0a266885ca98b01e355a36cfd3acb0be1c3da6eea53a09376bc680264',6699),
      ('vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamp with time zone)','ec662cc7118eb25eb2ebe79107c1ad1f16e5f5197fab8ff5e2051b3ddbc9fc7a','a6c3617ccb33da4bf69e2deff447bb1f93966472409fca518723919e2ce80d60',6702)
    ), actual AS (
      SELECT p.oid::regprocedure::text,pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(pg_catalog.jsonb_build_array(l.lanname,p.proowner::regrole::text,p.prokind::text,p.provolatile::text,p.proparallel::text,p.prosecdef,p.proleakproof,p.proisstrict,p.proretset,p.pronargs,p.pronargdefaults,p.prorettype::regtype::text,p.proargtypes::text,p.proallargtypes,p.proargmodes,p.proargnames,p.protrftypes,p.provariadic::regtype::text,p.proconfig,p.prosrc,p.probin,p.prosqlbody::text,p.procost,p.prorows,p.prosupport::regprocedure::text,pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex'),
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
        pg_catalog.octet_length(p.prosrc)
        FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace='vec_autorizacion'::regnamespace AND
             p.proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1',
                                 'resolver_motivo_detalle_rrhh_v1']::name[])
    ), acl_esperada(clase,objeto,grantor,grantee,privilegio,delegable) AS (
      SELECT 'base',d.datname,d.datdba,rol,'CONNECT',false
        FROM pg_catalog.pg_database d WHERE d.datname=pg_catalog.current_database()
      UNION ALL SELECT 'esquema','vec_autorizacion',propietario,rol,'USAGE',false
      UNION ALL SELECT 'funcion',e.objeto,propietario,propietario,'EXECUTE',false FROM esperado e
      UNION ALL SELECT 'funcion',e.objeto,propietario,rol,'EXECUTE',false FROM esperado e
    ), acl_actual AS (
      SELECT 'base',d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(d.datacl) a
       WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL
      SELECT 'esquema',n.nspname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) a
       WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL
      SELECT 'funcion',p.oid::regprocedure::text,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=ANY(funciones)
    ), diferencia AS (
      SELECT 1 FROM (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada) d
    ) SELECT 1 FROM diferencia
  ) OR NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid=rol
      AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
      AND NOT r.rolcreaterole AND NOT r.rolinherit AND NOT r.rolreplication
      AND NOT r.rolbypassrls AND r.rolconnlimit=-1
      AND (r.rolpassword IS NULL OR r.rolpassword='********')
      AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
      AND pg_catalog.shobj_description(r.oid,'pg_authid')=
          'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
  ) OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole=rol)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE oid=10 AND rolsuper)
    OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_auth_members m LEFT JOIN pg_catalog.pg_roles a ON a.oid=m.member
       WHERE m.member=rol OR m.grantor=rol OR (m.roleid=rol AND
         (m.admin_option OR NOT m.inherit_option OR m.set_option OR m.grantor<>10
          OR NOT a.rolcanlogin OR NOT a.rolinherit OR a.rolsuper OR a.rolcreatedb
          OR a.rolcreaterole OR a.rolreplication OR a.rolbypassrls)))
    OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_attribute c CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_type c CROSS JOIN LATERAL pg_catalog.aclexplode(c.typacl) a WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_default_acl c CROSS JOIN LATERAL pg_catalog.aclexplode(c.defaclacl) a WHERE c.defaclrole=rol OR a.grantee=rol OR a.grantor=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid<>ALL(funciones) AND (a.grantee=rol OR a.grantor=rol))
    OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_database WHERE datdba=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_namespace WHERE nspowner=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_class WHERE relowner=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_proc WHERE proowner=rol
      UNION ALL SELECT 1 FROM pg_catalog.pg_type WHERE typowner=rol) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='instalación de resolución RRHH incompleta';
  END IF;
END $postvalidacion$;

COMMIT;
