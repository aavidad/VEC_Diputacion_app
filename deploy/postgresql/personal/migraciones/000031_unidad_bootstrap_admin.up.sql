\set ON_ERROR_STOP on
-- Personal31: cotejo privado de unidad y protección de publicación hasta COMMIT.
-- No publica unidades, perfiles, permisos ni otra auditoría.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000031:unidad-bootstrap-admin',0));
DO $pre$
DECLARE t oid:=to_regclass('vec_personal.org_nodo_historia');prop oid:=to_regrole('vec_personal_propietario');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR t IS NULL OR prop IS NULL OR to_regrole('vec_autorizacion_propietario') IS NULL
 OR NOT has_schema_privilege('vec_autorizacion_propietario','vec_personal','USAGE')
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=t AND relowner=prop AND relrowsecurity AND relforcerowsecurity)
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=t AND a.grantee<>prop)
 OR to_regclass('vec_personal.control_unidad_bootstrap_admin_v1') IS NOT NULL
 OR to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal31: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;

-- Barrera técnica: no guarda unidades ni acredita una autoridad de publicación.
-- Cada INSERT de la fuente, incluida retirada por revisión nueva, cambia esta
-- fila en la misma transacción del publicador y no puede sortear la instantánea.
CREATE TABLE vec_personal.control_unidad_bootstrap_admin_v1 (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 generacion bigint NOT NULL CHECK(generacion>=1)
);
ALTER TABLE vec_personal.control_unidad_bootstrap_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.control_unidad_bootstrap_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_personal.control_unidad_bootstrap_admin_v1 FOR ALL TO vec_personal_propietario
 USING(current_user='vec_personal_propietario') WITH CHECK(current_user='vec_personal_propietario');
REVOKE ALL ON TABLE vec_personal.control_unidad_bootstrap_admin_v1 FROM PUBLIC,vec_personal_ejecutor;
REVOKE ALL ON TYPE vec_personal.control_unidad_bootstrap_admin_v1 FROM PUBLIC,vec_personal_ejecutor;
INSERT INTO vec_personal.control_unidad_bootstrap_admin_v1 VALUES(true,1);

CREATE FUNCTION vec_personal.avanzar_barrera_unidad_bootstrap_admin_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF TG_TABLE_SCHEMA<>'vec_personal' OR TG_TABLE_NAME<>'org_nodo_historia' OR TG_OP<>'INSERT' OR TG_LEVEL<>'STATEMENT'
 THEN RAISE EXCEPTION 'Personal31: uso de barrera incompatible' USING ERRCODE='55000'; END IF;
 UPDATE vec_personal.control_unidad_bootstrap_admin_v1 SET generacion=generacion+1 WHERE singleton;
 IF NOT FOUND THEN RAISE EXCEPTION 'Personal31: barrera ausente' USING ERRCODE='55000'; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.avanzar_barrera_unidad_bootstrap_admin_v1() FROM PUBLIC,vec_personal_ejecutor;
CREATE TRIGGER barrera_unidad_bootstrap_admin_v1 BEFORE INSERT ON vec_personal.org_nodo_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.avanzar_barrera_unidad_bootstrap_admin_v1();

-- Puerto privado de AUT. La revisión de la fuente es revision de Personal10,
-- no una versión inferida del nombre de catálogo o de un rol administrativo.
CREATE FUNCTION vec_personal.cotejar_unidad_bootstrap_admin_v1(p_organizacion text,p_ambito_unidad jsonb,p_hasta timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE unidad text;fuente jsonb;r record;generacion bigint;ahora timestamptz;fecha date;fin timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Personal31: requiere SERIALIZABLE READ WRITE' USING ERRCODE='25000'; END IF;
 IF p_organizacion IS NULL OR p_organizacion !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR p_hasta IS NULL OR NOT isfinite(p_hasta) OR p_hasta<=clock_timestamp()
 OR jsonb_typeof(p_ambito_unidad) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(p_ambito_unidad) ORDER BY 1) IS DISTINCT FROM ARRAY['dimension','fuente','valores']
 OR p_ambito_unidad->>'dimension' IS DISTINCT FROM 'unidad_ref'
 OR jsonb_typeof(p_ambito_unidad->'valores') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'Personal31: selector de unidad inválido' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_ambito_unidad->'valores')<>1
 OR jsonb_typeof(p_ambito_unidad#>'{valores,0}') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'Personal31: unidad exacta requerida' USING ERRCODE='22023'; END IF;
 unidad:=p_ambito_unidad#>>'{valores,0}';fuente:=p_ambito_unidad->'fuente';
 IF unidad IS NULL OR unidad !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR jsonb_typeof(fuente) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(fuente) ORDER BY 1) IS DISTINCT FROM ARRAY['huella_sha256','referencia','version']
 OR jsonb_typeof(fuente->'referencia') IS DISTINCT FROM 'string'
 OR (fuente->>'referencia') !~ '^[a-z][a-z0-9_:-]{2,159}$'
 OR jsonb_typeof(fuente->'version') IS DISTINCT FROM 'number' OR (fuente->>'version') !~ '^[1-9][0-9]{0,9}$'
 OR (fuente->>'version')::numeric>2147483647
 OR jsonb_typeof(fuente->'huella_sha256') IS DISTINCT FROM 'string'
 OR (fuente->>'huella_sha256') !~ '^[0-9a-f]{64}$' OR fuente->>'huella_sha256'=repeat('0',64)
 THEN RAISE EXCEPTION 'Personal31: procedencia de unidad inválida' USING ERRCODE='22023'; END IF;

 -- Un publicador actualiza esta misma fila antes de insertar. Si comprometió
 -- tras la instantánea, FOR SHARE produce 40001. Si viene después, espera al
 -- COMMIT de AUT. No se captura ni se reintenta 40001 dentro del helper.
 SELECT c.generacion INTO STRICT generacion FROM vec_personal.control_unidad_bootstrap_admin_v1 c WHERE c.singleton FOR SHARE;
 ahora:=clock_timestamp();fecha:=(ahora AT TIME ZONE 'UTC')::date;
 -- Primero se elige la última revisión conocida de cada nodo candidato.
 -- Después se comprueba la vigencia, retirada, organismo y unidad: no rescata
 -- una revisión anterior porque la nueva esté retirada o haya cambiado ámbito.
 SELECT actual.* INTO STRICT r FROM (
  SELECT DISTINCT ON(h.nodo_ref) h.* FROM vec_personal.org_nodo_historia h
  WHERE h.nodo_ref IN(SELECT candidato.nodo_ref FROM vec_personal.org_nodo_historia candidato WHERE candidato.organismo_ref=p_organizacion AND candidato.unidad_ref=unidad)
  ORDER BY h.nodo_ref,h.conocido_desde DESC,h.revision DESC
 ) actual WHERE actual.organismo_ref=p_organizacion AND actual.unidad_ref=unidad;
 IF r.retirado IS DISTINCT FROM false
 OR NOT isfinite(r.vigente_desde) OR (r.vigente_hasta IS NOT NULL AND NOT isfinite(r.vigente_hasta))
 OR NOT isfinite(r.conocido_desde) OR r.conocido_desde>ahora
 OR fecha<r.vigente_desde OR (r.vigente_hasta IS NOT NULL AND fecha>=r.vigente_hasta)
 OR r.fuente_ref IS DISTINCT FROM fuente->>'referencia'
 OR r.revision::numeric IS DISTINCT FROM (fuente->>'version')::numeric
 OR r.huella_fuente_sha256 IS DISTINCT FROM fuente->>'huella_sha256'
 THEN RAISE EXCEPTION 'Personal31: unidad actual no acreditada' USING ERRCODE='42501'; END IF;
 fin:=CASE WHEN r.vigente_hasta IS NULL THEN p_hasta ELSE r.vigente_hasta::timestamp AT TIME ZONE 'UTC' END;
 IF p_hasta>fin OR clock_timestamp()>=LEAST(fin,p_hasta)
 THEN RAISE EXCEPTION 'Personal31: vigencia de unidad insuficiente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('esquema','vec.personal.unidad-bootstrap-admin.v1',
  'organizacion_ref',r.organismo_ref,'unidad_ref',r.unidad_ref,'nodo_ref',r.nodo_ref::text,'revision',r.revision,
  'catalogo',jsonb_build_object('referencia',r.catalogo_ref,'version',r.catalogo_version,'revision',r.catalogo_revision,'entrada_clave',r.catalogo_entrada_clave),
  'fuente',jsonb_build_object('referencia',r.fuente_ref,'version',r.revision,'huella_sha256',r.huella_fuente_sha256),
  'acto_ref',r.acto_ref,'vigente_desde',r.vigente_desde,'vigente_hasta',r.vigente_hasta,
  'conocido_desde',to_char(r.conocido_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'valida_hasta',to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'generacion',generacion);
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'Personal31: barrera o unidad ausente o ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz) FROM PUBLIC,vec_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN('vec_personal_propietario'::regrole,'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'Personal31: ACL de helper ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
