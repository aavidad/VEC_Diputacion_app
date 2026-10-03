\set ON_ERROR_STOP on
-- AUT34 añade al catálogo nominal la concesión ya publicada por AUT33.
-- No cambia el perfil fijo v4 ni crea cuentas, vínculos o asignaciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regrole('vec_admin_perfiles_lector') IS NOT NULL
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.catalogo_accion_nominal_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT34: PARO clave=dependencias actual=incompatible esperado=AUT33_sin_AUT34' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_admin_perfiles_lector NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_lector;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $catalogo$
DECLARE r record;c jsonb;meta record;
BEGIN
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v4' FOR SHARE;
 SELECT * INTO STRICT meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref=r.version_rol_ref FOR SHARE;
 SELECT e.valor INTO STRICT c FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') e(valor)
 WHERE e.valor->>'accion'='administracion.perfiles.consultar';
 IF r.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR meta.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR meta.categoria_administrativa IS DISTINCT FROM 'aplicacion' OR meta.tipo_perfil IS DISTINCT FROM 'fijo_sistema'
 OR meta.fuente_ref IS DISTINCT FROM r.version_rol_ref OR meta.fuente_version IS DISTINCT FROM 4
 OR meta.fuente_huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR c->>'modulo_id' IS DISTINCT FROM 'administracion' OR c->>'tipo_recurso' IS DISTINCT FROM 'perfil'
 OR c->'finalidades' IS DISTINCT FROM '["gestion_perfiles"]'::jsonb
 OR c->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR c->'campos_permitidos' IS DISTINCT FROM '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb
 OR COALESCE(c->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AUT34: PARO clave=concesion actual=incompatible esperado=perfil_fijo_v4_consulta_exacta' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1
 VALUES('accion:administracion.perfiles.consultar',1,r.version_rol_ref,4,r.huella_sha256,r.version_rol_ref,c,
  '["organizacion_ref"]'::jsonb,'administrador_aplicacion',r.publicada_en,NULL);
END $catalogo$;
-- La fachada y su ACL se crean en AD168, después del consumidor causal V3.
-- Hasta entonces el rol no tiene EXECUTE, permisos de tabla ni mutaciones.
COMMIT;
