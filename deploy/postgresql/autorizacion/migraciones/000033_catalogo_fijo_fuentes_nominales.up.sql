\set ON_ERROR_STOP on
-- AUT33: versión prospectiva del perfil fijo de Administración de Aplicación.
-- No crea perfiles activos, asignaciones, cuentas ni sesión alguna.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:catalogo_nominal:000033',0));
DO $pre$
DECLARE v record;c record;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
  OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
  OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb)') IS NULL
  OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb)','EXECUTE')
  OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.catalogo_accion_nominal_v1') IS NOT NULL
  OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles' AND version>=4)
  OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT33: preimagen estructural incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO v FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v3';
 SELECT x.* INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
 WHERE a.version_rol_ref='rol:administracion_perfiles:v3';
 IF v.version_rol_ref IS NULL OR v.rol_id<>'administracion_perfiles' OR v.version<>3
  OR v.documento->>'estado'<>'publicada'
  OR v.huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(v.documento),'UTF8')),'hex')
  OR c.version_rol_ref IS NULL OR c.estado<>'habilitada'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto x
   WHERE x.version_rol_ref=v.version_rol_ref AND x.clase='administrador' AND x.huella_sha256=v.huella_sha256)
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil x WHERE x.version_rol_ref='rol:administracion_perfiles:v4')
 THEN RAISE EXCEPTION 'AUT33: perfil base incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- La categoría es dato de la misma autoridad de roles. Una fila sólo vale
-- para la versión y huella publicadas, sin reclasificar la historia v1–v3.
CREATE TABLE vec_autorizacion.perfil_fijo_categoria_nominal_v1 (
 version_rol_ref text PRIMARY KEY REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 version_rol_huella_sha256 text NOT NULL CHECK(version_rol_huella_sha256 ~ '^[0-9a-f]{64}$'),
 categoria_administrativa text NOT NULL CHECK(categoria_administrativa='aplicacion'),
 tipo_perfil text NOT NULL CHECK(tipo_perfil='fijo_sistema'),
 fuente_ref text NOT NULL,fuente_version numeric(20,0) NOT NULL CHECK(fuente_version>=1),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 publicada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_autorizacion.perfil_fijo_categoria_nominal_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.perfil_fijo_categoria_nominal_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.perfil_fijo_categoria_nominal_v1 FOR ALL TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.perfil_fijo_categoria_nominal_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.perfil_fijo_categoria_nominal_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.perfil_fijo_categoria_nominal_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.perfil_fijo_categoria_nominal_v1 FROM PUBLIC;

-- Publicación versionada de cada acción. Su fuente es la versión fija del
-- rol; se coteja su huella y concesión antes de autorizar cualquier consumo.
CREATE TABLE vec_autorizacion.catalogo_accion_nominal_v1 (
 accion_ref text NOT NULL,version numeric(20,0) NOT NULL CHECK(version>=1),
 fuente_ref text NOT NULL,fuente_version numeric(20,0) NOT NULL CHECK(fuente_version>=1),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 version_rol_ref text NOT NULL REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 concesion jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(concesion)='object'),
 dimensiones_ambito jsonb NOT NULL CHECK(dimensiones_ambito IN ('["organizacion_ref"]'::jsonb,'["organizacion_ref","unidad_ref"]'::jsonb)),
 clase_control text NOT NULL CHECK(clase_control='administrador_aplicacion'),
 vigente_desde timestamptz(6) NOT NULL,vigente_hasta timestamptz(6),
 PRIMARY KEY(accion_ref,version),UNIQUE(version_rol_ref,accion_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion.catalogo_accion_nominal_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.catalogo_accion_nominal_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.catalogo_accion_nominal_v1 FOR ALL TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.catalogo_accion_nominal_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.catalogo_accion_nominal_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.catalogo_accion_nominal_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.catalogo_accion_nominal_v1 FROM PUBLIC;

DO $publicar$
DECLARE anterior record;d jsonb;control jsonb;acciones jsonb;ahora timestamptz(6);fecha text;sha text;accion jsonb;
BEGIN
 SELECT * INTO STRICT anterior FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v3' FOR SHARE;
 ahora:=pg_catalog.clock_timestamp();
 fecha:=pg_catalog.to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 acciones:=pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('accion','administracion.certificados.nominal.publicar','modulo_id','administracion','tipo_recurso','vinculo_certificado_nominal','finalidades',pg_catalog.jsonb_build_array('gestionar_certificados_firmantes'),'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb),
  pg_catalog.jsonb_build_object('accion','administracion.certificados.nominal.retirar','modulo_id','administracion','tipo_recurso','vinculo_certificado_nominal','finalidades',pg_catalog.jsonb_build_array('gestionar_certificados_firmantes'),'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb),
  pg_catalog.jsonb_build_object('accion','personal.cargo_competencial.publicar','modulo_id','personal','tipo_recurso','cargo_competencial','finalidades',pg_catalog.jsonb_build_array('administrar_cargos_competenciales'),'garantia_minima','alto','campos_permitidos','["cargo","enlace","huella_sha256","recibo","version"]'::jsonb,'obligaciones','[]'::jsonb));
 IF EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(anterior.documento->'concesiones') AS c(valor)
  JOIN pg_catalog.jsonb_array_elements(acciones) AS n(valor) ON c.valor->>'accion'=n.valor->>'accion')
 THEN RAISE EXCEPTION 'AUT33: acción ya publicada' USING ERRCODE='55000'; END IF;
 d:=pg_catalog.jsonb_set(anterior.documento,'{version}','4'::jsonb);
 d:=pg_catalog.jsonb_set(d,'{concesiones}',(anterior.documento->'concesiones')||acciones);
 d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(d,'{publicada_por}','"migracion:autorizacion:000033"'::jsonb),'{publicada_en}',pg_catalog.to_jsonb(fecha));
 IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT33: acciones incompatibles con RBAC' USING ERRCODE='23514'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex');
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 VALUES('rol:administracion_perfiles:v4','administracion_perfiles',4,sha,ahora,d);
 control:=pg_catalog.jsonb_build_object('version_rol_ref','rol:administracion_perfiles:v4','revision',1,'estado','habilitada',
  'actualizado_por','migracion:autorizacion:000033','actualizado_en',fecha);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento,creada_en)
 VALUES('rol:administracion_perfiles:v4',1,'habilitada',
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex'),ahora,control,ahora);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
 VALUES('rol:administracion_perfiles:v4',1,ahora,'migracion:autorizacion:000033','migracion:autorizacion:000033');
 INSERT INTO vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase,huella_sha256)
 VALUES('rol:administracion_perfiles:v4','administrador',sha);
 INSERT INTO vec_autorizacion.perfil_fijo_categoria_nominal_v1
 VALUES('rol:administracion_perfiles:v4',sha,'aplicacion','fijo_sistema',
  'rol:administracion_perfiles:v4',4,sha,ahora);
 FOR accion IN SELECT e.valor FROM pg_catalog.jsonb_array_elements(acciones) AS e(valor) LOOP
  INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1
  VALUES('accion:'||(accion->>'accion'),1,'rol:administracion_perfiles:v4',4,sha,
   'rol:administracion_perfiles:v4',accion,
   CASE WHEN accion->>'modulo_id'='administracion' THEN '["organizacion_ref"]'::jsonb
    ELSE '["organizacion_ref","unidad_ref"]'::jsonb END,
   'administrador_aplicacion',ahora,NULL);
 END LOOP;
END $publicar$;

-- Sólo el propietario AD3 verifica la categoría del actor de una decisión.
-- Bloquea asignación, control y versión; no confunde clase administrador
-- genérica con categoría Aplicación.
CREATE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
 version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,
 accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record;ct record;asig record;meta record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v4'
  OR accion IS NULL OR modulo IS NULL OR tipo IS NULL OR finalidad IS NULL
  OR campos IS NULL OR pg_catalog.jsonb_typeof(campos)<>'array'
  OR pg_catalog.jsonb_typeof(autenticacion)<>'object'
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(autenticacion) IS NOT TRUE
  THEN RETURN false; END IF;
 SELECT v.* INTO r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT c.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=version_ref FOR SHARE OF a,c;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO asig FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=perfil_ref AND a.asignacion_ref=p_asignacion_ref FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.version=4 AND r.documento->>'estado'='publicada'
  AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  AND ct.estado='habilitada' AND ct.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
  AND meta.fuente_ref=version_ref AND meta.fuente_version=4 AND meta.fuente_huella_sha256=r.huella_sha256
  AND asig.version_rol_ref=version_ref AND asig.principal_id=principal_ref AND asig.perfil_activo_ref=perfil_ref
  AND asig.documento->>'estado'='activa'
  AND ahora>=(asig.documento->>'vigente_desde')::timestamptz
  AND ahora<(asig.documento->>'vigente_hasta')::timestamptz
  AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 catalogo
    WHERE catalogo.version_rol_ref=version_ref AND catalogo.accion_ref='accion:'||accion
     AND catalogo.version=1 AND catalogo.fuente_ref=version_ref AND catalogo.fuente_version=4
     AND catalogo.fuente_huella_sha256=r.huella_sha256
     AND catalogo.clase_control='administrador_aplicacion'
     AND catalogo.dimensiones_ambito=CASE WHEN modulo='administracion' THEN '["organizacion_ref"]'::jsonb
       ELSE '["organizacion_ref","unidad_ref"]'::jsonb END
     AND ahora>=catalogo.vigente_desde AND (catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
     AND catalogo.concesion->>'accion'=accion AND catalogo.concesion->>'modulo_id'=modulo
     AND catalogo.concesion->>'tipo_recurso'=tipo
     AND catalogo.concesion->'finalidades'=pg_catalog.jsonb_build_array(finalidad)
     AND catalogo.concesion->>'garantia_minima'='alto'
     AND COALESCE(catalogo.concesion->'campos_permitidos','[]'::jsonb)=campos
     AND COALESCE(catalogo.concesion->'obligaciones','[]'::jsonb)='[]'::jsonb)
  AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
   WHERE c->>'accion'=accion AND c->>'modulo_id'=modulo AND c->>'tipo_recurso'=tipo
    AND c->'finalidades'=pg_catalog.jsonb_build_array(finalidad) AND c->>'garantia_minima'='alto'
    AND COALESCE(c->'campos_permitidos','[]'::jsonb)=campos AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;

-- El ámbito del actor se coteja contra la organización acreditada por CA4,
-- no contra un valor de la petición. Personal conserva su unidad propia.
CREATE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v1(version_ref text,p_asignacion_ref text,org text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v4'
  OR org IS NULL OR org !~ '^org_[a-z0-9]{16,80}$' THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.documento->>'estado'<>'activa' THEN RETURN false; END IF;
 RETURN a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(org)));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text)
 TO vec_autorizacion_atestada_v3_propietario;

-- IS clasifica la cuenta bajo sus propios permisos y bloquea su estado.
-- Ausencia o estado inactivo nunca equivalen a cuenta ordinaria.
RESET ROLE;
DO $pre_is$
BEGIN
 IF pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
  OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.cuenta') IS NULL
  OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.estado_cuenta_actual') IS NULL
  OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.estado_cuenta') IS NULL
  OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT33: fuente IS incompatible' USING ERRCODE='55000'; END IF;
END $pre_is$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off' OR ref IS NULL THEN RETURN NULL; END IF;
 SELECT x.cuenta_privilegiada,x.cuenta_ref,s.estado INTO c
 FROM vec_identidad_sesiones_v1.cuenta x
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
 JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
 WHERE x.cuenta_ref=ref FOR SHARE OF x,a,s;
 IF NOT FOUND OR c.estado<>'activa' THEN RETURN NULL; END IF;
 RETURN c.cuenta_privilegiada;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(text)
 TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.destino_no_administrador_certificado_nominal_v1(cuenta text,persona text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE privilegiada boolean;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR cuenta IS NULL OR persona IS NULL THEN RETURN false; END IF;
 privilegiada:=vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(cuenta);
 IF privilegiada IS DISTINCT FROM false THEN RETURN false; END IF;
 -- Impide una asignación administrativa concurrente, incluida una sobre
 -- otro perfil activo de la misma persona, hasta el COMMIT del certificado.
 LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE MODE;
 LOCK TABLE vec_autorizacion.control_vigencia_version_rol_actual IN SHARE MODE;
 ahora:=pg_catalog.clock_timestamp();
 RETURN NOT EXISTS(
  SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual x
  JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
  JOIN vec_autorizacion.rol_sensible_exacto s ON s.version_rol_ref=a.version_rol_ref
  JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=s.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol cv USING(version_rol_ref,revision)
  WHERE a.principal_id=persona AND s.clase='administrador' AND s.huella_sha256=r.huella_sha256
   AND a.documento->>'estado'='activa' AND r.documento->>'estado'='publicada'
   AND cv.estado='habilitada'
   AND ahora>=(a.documento->>'vigente_desde')::timestamptz
   AND ahora<(a.documento->>'vigente_hasta')::timestamptz);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
