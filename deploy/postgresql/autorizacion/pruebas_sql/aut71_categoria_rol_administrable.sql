\set ON_ERROR_STOP on
-- AUT71: el resolver real del catálogo administrable devuelve categoria_admin
-- con el contrato del adaptador Go (rolJSON). Antes de AUT71 esta prueba falla.
-- Ejecutar como superusuario en un clon PG18 después de AUT71. Todo termina en
-- ROLLBACK: los dos roles de prueba del catálogo no quedan en la base.
-- Llama con los grupos que usa el Go en ejecución, no con un doble.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';

-- ADMIN vigente: la versión más alta de administracion_perfiles que AUT24 resuelve.
CREATE TEMP TABLE aut71_admin ON COMMIT DROP AS
SELECT a.version_rol_ref,v.version FROM vec_autorizacion.rol_administrable_exacto_v1 a
JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=a.version_rol_ref AND v.huella_sha256=a.huella_sha256
JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=a.version_rol_ref
JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
WHERE v.rol_id='administracion_perfiles' AND a.clase='administrador' AND v.documento->>'estado'='publicada'
AND c.estado='habilitada' AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta
ORDER BY v.version DESC LIMIT 1;
GRANT SELECT ON aut71_admin TO vec_autorizacion_propietario,vec_admin_perfiles_lote_ejecutor,vec_admin_gobierno_roles_ejecutor,vec_admin_version_rol_bolsa_ejecutor;

-- RolID RRHH ordinario: última versión publicada y habilitada. Si el catálogo
-- aún no la tiene, se registra aquí como fila ordinaria (fixture, ROLLBACK).
-- La versión anterior se registra como «administrador» sin categoría nominal
-- para comprobar que el resolver falla cerrado.
CREATE TEMP TABLE aut71_rrhh ON COMMIT DROP AS
SELECT v.version_rol_ref,v.version,v.huella_sha256,
 lag(v.version_rol_ref) OVER (ORDER BY v.version) AS anterior_ref
FROM vec_autorizacion.version_rol v
JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=v.version_rol_ref
JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
WHERE v.rol_id='tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo' AND v.documento->>'estado'='publicada' AND c.estado='habilitada'
ORDER BY v.version DESC LIMIT 1;
GRANT SELECT ON aut71_rrhh TO vec_autorizacion_propietario,vec_admin_perfiles_lote_ejecutor,vec_admin_gobierno_roles_ejecutor,vec_admin_version_rol_bolsa_ejecutor;

DO $previo$
BEGIN
 IF (SELECT count(*) FROM aut71_admin)<>1 THEN RAISE EXCEPTION 'AUT71 prueba: sin ADMIN vigente'; END IF;
 IF (SELECT count(*) FROM aut71_rrhh WHERE anterior_ref IS NOT NULL)<>1 THEN RAISE EXCEPTION 'AUT71 prueba: sin dos versiones del RolID RRHH'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a JOIN aut71_rrhh r ON a.version_rol_ref=r.anterior_ref)
 THEN RAISE EXCEPTION 'AUT71 prueba: la versión anterior RRHH ya está en el catálogo'; END IF;
END $previo$;

SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.rol_administrable_exacto_v1
SELECT r.version_rol_ref,'ordinario',r.huella_sha256,clock_timestamp()-interval '1 hour',clock_timestamp()+interval '400 days',false,
 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1','[]'::jsonb,interval '1 day'
FROM aut71_rrhh r WHERE NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a WHERE a.version_rol_ref=r.version_rol_ref);
INSERT INTO vec_autorizacion.rol_administrable_exacto_v1
SELECT v.version_rol_ref,'administrador',v.huella_sha256,clock_timestamp()-interval '1 hour',clock_timestamp()+interval '400 days',false,
 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1','[]'::jsonb,interval '1 day'
FROM aut71_rrhh r JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=r.anterior_ref;
RESET ROLE;

-- Comprobaciones con cada grupo de ejecución que usa el adaptador Go.
CREATE FUNCTION pg_temp.aut71_comprobar(grupo text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE admin jsonb;rrhh jsonb;rechazo text:=null;
BEGIN
 admin:=vec_autorizacion.resolver_rol_administrable_v1((SELECT version_rol_ref FROM aut71_admin));
 IF admin->>'clase' IS DISTINCT FROM 'administrador' OR NOT admin ? 'categoria_admin'
 OR admin->>'categoria_admin' IS DISTINCT FROM 'aplicacion' THEN
  RAISE EXCEPTION 'AUT71 prueba [%]: ADMIN % sin categoria_admin=aplicacion: %',grupo,(SELECT version_rol_ref FROM aut71_admin),admin; END IF;
 rrhh:=vec_autorizacion.resolver_rol_administrable_v1((SELECT version_rol_ref FROM aut71_rrhh));
 IF rrhh->>'clase' IS DISTINCT FROM 'ordinario' OR NOT rrhh ? 'categoria_admin'
 OR rrhh->'categoria_admin' IS DISTINCT FROM 'null'::jsonb THEN
  RAISE EXCEPTION 'AUT71 prueba [%]: RolID RRHH sin categoria_admin nula: %',grupo,rrhh; END IF;
 BEGIN
  PERFORM vec_autorizacion.resolver_rol_administrable_v1((SELECT anterior_ref FROM aut71_rrhh));
 EXCEPTION WHEN insufficient_privilege THEN rechazo:=SQLERRM;
 END;
 IF rechazo IS DISTINCT FROM 'AUT71: administrador sin categoria nominal' THEN
  RAISE EXCEPTION 'AUT71 prueba [%]: administrador sin categoria nominal no falla cerrado (%)',grupo,coalesce(rechazo,'resuelto'); END IF;
 RAISE NOTICE 'AUT71 prueba [%]: OK ADMIN=% RRHH=%',grupo,admin->>'categoria_admin',rrhh->'categoria_admin';
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.aut71_comprobar(text) TO vec_admin_perfiles_lote_ejecutor,vec_admin_gobierno_roles_ejecutor,vec_admin_version_rol_bolsa_ejecutor;

SET LOCAL ROLE vec_admin_perfiles_lote_ejecutor;
SELECT pg_temp.aut71_comprobar('vec_admin_perfiles_lote_ejecutor');
RESET ROLE;
SET LOCAL ROLE vec_admin_gobierno_roles_ejecutor;
SELECT pg_temp.aut71_comprobar('vec_admin_gobierno_roles_ejecutor');
RESET ROLE;
SET LOCAL ROLE vec_admin_version_rol_bolsa_ejecutor;
SELECT pg_temp.aut71_comprobar('vec_admin_version_rol_bolsa_ejecutor');
RESET ROLE;

-- Todo el catálogo resoluble respeta el contrato (administrador ⇒ categoría
-- nominal de su versión y huella; resto ⇒ null), incluido el de Sistemas.
DO $catalogo$
DECLARE fila record;resultado jsonb;n integer:=0;
BEGIN
 FOR fila IN SELECT a.version_rol_ref,a.clase,m.categoria_administrativa FROM vec_autorizacion.rol_administrable_exacto_v1 a
  JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=a.version_rol_ref AND v.huella_sha256=a.huella_sha256
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=a.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
  LEFT JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m ON m.version_rol_ref=a.version_rol_ref
  WHERE v.documento->>'estado'='publicada' AND c.estado='habilitada'
  AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta
  AND a.version_rol_ref<>(SELECT anterior_ref FROM aut71_rrhh) LOOP
  resultado:=vec_autorizacion.resolver_rol_administrable_v1(fila.version_rol_ref);
  IF NOT resultado ? 'categoria_admin'
  OR (fila.clase='administrador' AND resultado->>'categoria_admin' IS DISTINCT FROM fila.categoria_administrativa)
  OR (fila.clase<>'administrador' AND resultado->'categoria_admin' IS DISTINCT FROM 'null'::jsonb) THEN
   RAISE EXCEPTION 'AUT71 prueba: % (%) incumple el contrato: %',fila.version_rol_ref,fila.clase,resultado; END IF;
  n:=n+1;
 END LOOP;
 RAISE NOTICE 'AUT71 prueba: % roles del catálogo con categoria_admin conforme',n;
END $catalogo$;

-- Una fila nominal en un rol que no es administrador también falla cerrado.
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.perfil_fijo_categoria_nominal_v1
SELECT r.version_rol_ref,r.huella_sha256,'aplicacion','fijo_sistema',r.version_rol_ref,r.version,r.huella_sha256,clock_timestamp()
FROM aut71_rrhh r;
RESET ROLE;
DO $nominal_ordinario$
DECLARE rechazo text:=null;
BEGIN
 BEGIN
  PERFORM vec_autorizacion.resolver_rol_administrable_v1((SELECT version_rol_ref FROM aut71_rrhh));
 EXCEPTION WHEN insufficient_privilege THEN rechazo:=SQLERRM;
 END;
 IF rechazo IS DISTINCT FROM 'AUT71: categoria nominal en rol no administrador' THEN
  RAISE EXCEPTION 'AUT71 prueba: fila nominal en rol ordinario no falla cerrado (%)',coalesce(rechazo,'resuelto'); END IF;
 RAISE NOTICE 'AUT71 prueba: fila nominal en rol ordinario rechazada';
END $nominal_ordinario$;
ROLLBACK;
