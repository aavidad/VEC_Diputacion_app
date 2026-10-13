\set ON_ERROR_STOP on
-- CA45: correspondencia de la autoridad de Organización. No concede permisos.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000045',0));

CREATE TABLE vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 (
 correspondencia_ref text NOT NULL CHECK (correspondencia_ref ~ '^correspondencia-org:[0-9a-f-]{36}$'),
 organizacion_operativa_ref text NOT NULL CHECK (length(organizacion_operativa_ref) BETWEEN 3 AND 512),
 version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 9007199254740991),
 organizacion_canonica_ref text NOT NULL,
 organizacion_version numeric(20,0) NOT NULL,
 documento_canonico bytea NOT NULL CHECK (octet_length(documento_canonico) BETWEEN 2 AND 8192),
 huella_sha256 text NOT NULL CHECK (huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
 publicada_por name NOT NULL,
 publicada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (organizacion_operativa_ref,version),
 UNIQUE (correspondencia_ref,version),
 FOREIGN KEY (organizacion_canonica_ref,organizacion_version)
  REFERENCES vec_contexto_actor_v1.organizacion_versiones(organizacion_ref,version),
 CHECK (organizacion_operativa_ref<>organizacion_canonica_ref)
);
CREATE TABLE vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 (
 organizacion_operativa_ref text PRIMARY KEY,
 version numeric(20,0) NOT NULL,
 FOREIGN KEY (organizacion_operativa_ref,version)
  REFERENCES vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1(organizacion_operativa_ref,version)
);
ALTER TABLE vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1
 TO vec_contexto_actor_v1_propietario USING(true) WITH CHECK(true);
CREATE POLICY propietario ON vec_contexto_actor_v1.correspondencia_organizacion_actual_v1
 TO vec_contexto_actor_v1_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1,
 vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1,
 vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 FROM PUBLIC;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
 ON vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 FOR EACH ROW
 EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE
 ON vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();

-- Es provisión de fuente corporativa, exclusiva del migrador existente.
-- La procedencia es la misma autoridad maestra que acredita la organización;
-- ni CT ni su ejecutor pueden publicar o retirar correspondencias.
CREATE FUNCTION vec_contexto_actor_v1.publicar_correspondencia_organizacion_v1(
 p_documento text,p_version_esperada numeric,p_huella_esperada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE d jsonb; actual record; org record; b bytea; h text; ahora timestamptz(6);
BEGIN
 IF NOT pg_catalog.pg_has_role(session_user,'vec_contexto_actor_v1_migrador','MEMBER')
  AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'correspondencia_organizacion_publicador_denegado' USING ERRCODE='42501';
 END IF;
 IF p_documento IS NULL OR p_version_esperada IS NULL OR p_huella_esperada IS NULL OR pg_catalog.octet_length(p_documento) NOT BETWEEN 2 AND 8192 THEN
  RAISE EXCEPTION 'correspondencia_organizacion_documento_invalido' USING ERRCODE='22023';
 END IF;
 d:=p_documento::jsonb;
 IF pg_catalog.jsonb_typeof(d)<>'object' OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>13
  OR NOT(d ?& ARRAY['esquema','correspondencia_ref','version','organizacion_operativa_ref',
   'organizacion_canonica_ref','organizacion_version','procedencia_ref','procedencia_version',
   'procedencia_huella_sha256','procedencia_autoridad','estado','vigente_desde','vigente_hasta'])
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(d) e WHERE e.value='null'::jsonb)
  OR d->>'esquema'<>'vec.organizacion.correspondencia.v1'
  OR (d->>'correspondencia_ref' ~ '^correspondencia-org:[0-9a-f-]{36}$') IS NOT TRUE
  OR (d->>'organizacion_operativa_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$') IS NOT TRUE
  OR vec_contexto_actor_v1.organizacion_ref_valida(d->>'organizacion_canonica_ref') IS NOT TRUE
  OR d->>'organizacion_operativa_ref'=d->>'organizacion_canonica_ref'
  OR d->>'estado' NOT IN('vigente','retirada')
  OR (d->>'version')::numeric NOT BETWEEN 1 AND 9007199254740991
  OR (d->>'vigente_hasta')::timestamptz<=(d->>'vigente_desde')::timestamptz
  OR NOT pg_catalog.isfinite((d->>'vigente_desde')::timestamptz)
  OR NOT pg_catalog.isfinite((d->>'vigente_hasta')::timestamptz) THEN
  RAISE EXCEPTION 'correspondencia_organizacion_documento_invalido' USING ERRCODE='22023';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('organizacion_correspondencia:'||(d->>'organizacion_operativa_ref'),0));
 SELECT v.* INTO actual FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
  JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 v USING(organizacion_operativa_ref,version)
  WHERE a.organizacion_operativa_ref=d->>'organizacion_operativa_ref' FOR UPDATE OF a;
 IF (actual.version IS NULL AND (p_version_esperada<>0 OR p_huella_esperada<>''))
  OR (actual.version IS NOT NULL AND (actual.version IS DISTINCT FROM p_version_esperada OR actual.huella_sha256 IS DISTINCT FROM p_huella_esperada))
  OR (d->>'version')::numeric<>coalesce(actual.version,0)+1
  OR (actual.version IS NOT NULL AND actual.correspondencia_ref IS DISTINCT FROM d->>'correspondencia_ref') THEN
  RAISE EXCEPTION 'correspondencia_organizacion_preimagen_divergente' USING ERRCODE='40001';
 END IF;
 SELECT v.* INTO STRICT org FROM vec_contexto_actor_v1.organizacion_actual a
  JOIN vec_contexto_actor_v1.organizacion_versiones v USING(organizacion_ref,version)
  WHERE a.organizacion_ref=d->>'organizacion_canonica_ref' FOR SHARE OF a,v;
 ahora:=pg_catalog.clock_timestamp();
 IF org.version IS DISTINCT FROM (d->>'organizacion_version')::numeric OR org.estado<>'activo'
  OR ahora<org.vigente_desde OR ahora>=org.vigente_hasta
  OR org.procedencia_ref IS DISTINCT FROM d->>'procedencia_ref'
  OR org.procedencia_version IS DISTINCT FROM (d->>'procedencia_version')::numeric
  OR org.procedencia_huella_sha256 IS DISTINCT FROM d->>'procedencia_huella_sha256'
  OR org.procedencia_autoridad IS DISTINCT FROM d->>'procedencia_autoridad'
  OR vec_contexto_actor_v1.procedencia_valida(org.procedencia_ref,org.procedencia_version,org.procedencia_huella_sha256,org.procedencia_autoridad) IS NOT TRUE THEN
  RAISE EXCEPTION 'correspondencia_organizacion_fuente_no_acreditada' USING ERRCODE='42501';
 END IF;
 b:=pg_catalog.convert_to(d::text,'UTF8'); h:=pg_catalog.encode(pg_catalog.sha256(b),'hex');
 INSERT INTO vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1
 VALUES(d->>'correspondencia_ref',d->>'organizacion_operativa_ref',(d->>'version')::numeric,
  org.organizacion_ref,org.version,b,h,session_user,ahora);
 INSERT INTO vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 VALUES(d->>'organizacion_operativa_ref',(d->>'version')::numeric)
 ON CONFLICT(organizacion_operativa_ref) DO UPDATE SET version=EXCLUDED.version;
 RETURN d||pg_catalog.jsonb_build_object('huella_sha256',h);
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(p_organizacion_operativa text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE v record; org record; prueba jsonb; canonica text; ahora timestamptz(6);
BEGIN
 IF p_organizacion_operativa IS NULL OR length(p_organizacion_operativa) NOT BETWEEN 3 AND 512 THEN
  RAISE EXCEPTION 'correspondencia_organizacion_referencia_invalida' USING ERRCODE='42501';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('organizacion_correspondencia:'||p_organizacion_operativa,0));
 IF vec_contexto_actor_v1.organizacion_ref_valida(p_organizacion_operativa) THEN
  canonica:=p_organizacion_operativa; prueba:=NULL;
 ELSE
  SELECT h.* INTO STRICT v FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
   JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h USING(organizacion_operativa_ref,version)
   WHERE a.organizacion_operativa_ref=p_organizacion_operativa FOR SHARE OF a,h;
  prueba:=pg_catalog.convert_from(v.documento_canonico,'UTF8')::jsonb||pg_catalog.jsonb_build_object('huella_sha256',v.huella_sha256);
  ahora:=pg_catalog.clock_timestamp();
  IF prueba->>'estado'<>'vigente' OR ahora<(prueba->>'vigente_desde')::timestamptz OR ahora>=(prueba->>'vigente_hasta')::timestamptz THEN
   RAISE EXCEPTION 'correspondencia_organizacion_no_vigente' USING ERRCODE='42501';
  END IF;
  canonica:=v.organizacion_canonica_ref;
 END IF;
 SELECT h.* INTO STRICT org FROM vec_contexto_actor_v1.organizacion_actual a
  JOIN vec_contexto_actor_v1.organizacion_versiones h USING(organizacion_ref,version)
  WHERE a.organizacion_ref=canonica FOR SHARE OF a,h;
 ahora:=pg_catalog.clock_timestamp();
 IF org.estado<>'activo' OR ahora<org.vigente_desde OR ahora>=org.vigente_hasta
  OR vec_contexto_actor_v1.procedencia_valida(org.procedencia_ref,org.procedencia_version,org.procedencia_huella_sha256,org.procedencia_autoridad) IS NOT TRUE
  OR (prueba IS NOT NULL AND (org.version IS DISTINCT FROM v.organizacion_version
   OR org.procedencia_ref IS DISTINCT FROM prueba->>'procedencia_ref'
   OR org.procedencia_version IS DISTINCT FROM (prueba->>'procedencia_version')::numeric
   OR org.procedencia_huella_sha256 IS DISTINCT FROM prueba->>'procedencia_huella_sha256'
   OR org.procedencia_autoridad IS DISTINCT FROM prueba->>'procedencia_autoridad')) THEN
  RAISE EXCEPTION 'correspondencia_organizacion_fuente_no_vigente' USING ERRCODE='42501';
 END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_operativa_ref',p_organizacion_operativa,'organizacion_canonica_ref',canonica,'prueba',prueba);
EXCEPTION WHEN NO_DATA_FOUND OR TOO_MANY_ROWS THEN
 RAISE EXCEPTION 'correspondencia_organizacion_no_unica' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_correspondencia_organizacion_v1(text,numeric,text),
 vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.publicar_correspondencia_organizacion_v1(text,numeric,text)
 TO vec_contexto_actor_v1_migrador;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)
 TO vec_autorizacion_propietario,vec_contratacion_temporal_propietario;
COMMIT;
