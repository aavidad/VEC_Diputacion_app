\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
-- Mismo orden que AD3-28: nunca adquirir Personal después de CT.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000072:registro:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

-- Fundación de almacenamiento, NO API de registro. No hay GRANT runtime ni
-- función que acepte un posterior calculado por el llamador. SHA(bytes) prueba
-- integridad, no reproducción del dominio ni procedencia de publicación.
-- Pendientes: codec binario/replay de dominio SQL y publicación gobernada.
-- CT70/71 NO cubren raíz/definición/historia de seguimiento.
DO $precondiciones$
DECLARE nombre text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user
     AND NOT rolcanlogin AND NOT rolsuper AND NOT rolinherit
     AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal'
     AND nspowner=current_user::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_contratacion_temporal' AND p.proname='rechazar_mutacion_historia_v1'
     AND p.pronargs=0 AND p.prorettype='trigger'::regtype AND p.proowner=current_user::regrole)
 THEN RAISE EXCEPTION 'CT72: precondiciones incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['expediente_alta','expediente_version_integral','expediente_integral_actual'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_contratacion_temporal' AND c.relname=nombre AND c.relkind='r'
      AND c.relowner=current_user::regrole
      -- CT3 protege expediente_alta por ACL exclusiva, sin instalar RLS.
      -- CT6 sí fuerza RLS sobre las dos tablas integrales. No alterar ninguna.
      AND ((nombre='expediente_alta' AND NOT EXISTS (
        SELECT 1 FROM aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
         WHERE a.grantee<>c.relowner OR a.grantor<>c.relowner))
       OR (nombre<>'expediente_alta' AND c.relrowsecurity AND c.relforcerowsecurity))) THEN
   RAISE EXCEPTION 'CT72: expediente durable requerido' USING ERRCODE='55000';
  END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2'] LOOP
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_contratacion_temporal' AND c.relname=nombre)
  OR EXISTS (SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname='vec_contratacion_temporal' AND t.typname=nombre) THEN
   RAISE EXCEPTION 'CT72: objeto preexistente; no se reemplaza' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $precondiciones$;

-- Bytes de publicación gobernada V1. No se deduce canon de jsonb::text.
-- Sin API de publicación: un migrador no debe importar JSON del navegador.
CREATE TABLE vec_contratacion_temporal.seguimiento_definicion_v2 (
 definicion_ref text NOT NULL CHECK (definicion_ref ~ '^ref:[0-9a-f]{64}$'
     AND definicion_ref<>'ref:'||repeat('0',64)),
 definicion_version numeric(20,0) NOT NULL CHECK (definicion_version BETWEEN 1 AND 9007199254740991),
 definicion_sha256 text NOT NULL CHECK (definicion_sha256 ~ '^[0-9a-f]{64}$'
     AND definicion_sha256<>repeat('0',64)),
 publicacion_json jsonb NOT NULL CHECK (jsonb_typeof(publicacion_json)='object'
     AND octet_length(publicacion_json::text) BETWEEN 2 AND 8388608),
 definicion_canonica bytea NOT NULL CHECK (octet_length(definicion_canonica) BETWEEN 1 AND 8388608),
 PRIMARY KEY (definicion_ref,definicion_version),
 UNIQUE (definicion_ref,definicion_version,definicion_sha256),
 CHECK (definicion_sha256=encode(sha256(definicion_canonica),'hex')),
 CHECK ((publicacion_json->'referencia'=to_jsonb(definicion_ref)
     AND publicacion_json->'version'=to_jsonb(definicion_version)
     AND publicacion_json->'huella_sha256'=to_jsonb(definicion_sha256)
     AND publicacion_json->'canon'='{"dominio":"vec.dipgra.contratacion-temporal.seguimiento.definicion","version_esquema":1,"algoritmo":"sha-256"}'::jsonb) IS TRUE)
);

-- Raíz explícita, no creada al confirmar. No imponemos una falsa unicidad
-- organización/expediente/relación: la referencia de raíz es parte del CAS.
CREATE TABLE vec_contratacion_temporal.seguimiento_raiz_v2 (
 seguimiento_ref text PRIMARY KEY CHECK (seguimiento_ref ~ '^ref:[0-9a-f]{64}$'
     AND seguimiento_ref<>'ref:'||repeat('0',64)),
 organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^ref:[0-9a-f]{64}$'
     AND organizacion_ref<>'ref:'||repeat('0',64)),
 expediente_ref text NOT NULL CHECK (expediente_ref ~ '^ref:[0-9a-f]{64}$'
     AND expediente_ref<>'ref:'||repeat('0',64)),
 relacion_ref text NOT NULL CHECK (relacion_ref ~ '^ref:[0-9a-f]{64}$'
     AND relacion_ref<>'ref:'||repeat('0',64)),
 version_expediente_observada numeric(20,0) NOT NULL
     CHECK (version_expediente_observada BETWEEN 1 AND 9007199254740991),
 definicion_ref text NOT NULL,
 definicion_version numeric(20,0) NOT NULL,
 definicion_sha256 text NOT NULL,
 raiz_canonica bytea NOT NULL CHECK (octet_length(raiz_canonica) BETWEEN 1 AND 8388608),
 raiz_sha256 text NOT NULL CHECK (raiz_sha256 ~ '^[0-9a-f]{64}$' AND raiz_sha256<>repeat('0',64)),
 version_inicial numeric(20,0) NOT NULL DEFAULT 0 CHECK (version_inicial=0),
 FOREIGN KEY (expediente_ref,version_expediente_observada)
     REFERENCES vec_contratacion_temporal.expediente_version_integral,
 FOREIGN KEY (definicion_ref,definicion_version,definicion_sha256)
     REFERENCES vec_contratacion_temporal.seguimiento_definicion_v2(definicion_ref,definicion_version,definicion_sha256),
 CHECK (raiz_sha256=encode(sha256(raiz_canonica),'hex')),
 UNIQUE (seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,
     definicion_ref,definicion_version,definicion_sha256,raiz_sha256)
);

-- Historia inmutable. El futuro writer bloqueará la fila raíz y cotejará MAX
-- versión + estado/binario antes de insertar N+1; la FK impide saltos/huecos.
-- No se interpreta existencia de un hash como autorización o validación dominio.
CREATE TABLE vec_contratacion_temporal.seguimiento_estado_v2 (
 seguimiento_ref text NOT NULL,
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 relacion_ref text NOT NULL,
 definicion_ref text NOT NULL,
 definicion_version numeric(20,0) NOT NULL,
 definicion_sha256 text NOT NULL,
 raiz_sha256 text NOT NULL,
 version_seguimiento numeric(20,0) NOT NULL CHECK (version_seguimiento BETWEEN 0 AND 9007199254740991),
 version_anterior numeric(20,0),
 estado_anterior_sha256 text,
 estado_json jsonb NOT NULL CHECK (jsonb_typeof(estado_json)='object'
     AND octet_length(estado_json::text) BETWEEN 2 AND 8388608),
 estado_canonico bytea NOT NULL CHECK (octet_length(estado_canonico) BETWEEN 1 AND 8388608),
 estado_sha256 text NOT NULL CHECK (estado_sha256 ~ '^[0-9a-f]{64}$' AND estado_sha256<>repeat('0',64)),
 PRIMARY KEY (seguimiento_ref,version_seguimiento),
 UNIQUE (seguimiento_ref,version_seguimiento,estado_sha256),
 FOREIGN KEY (seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,
     definicion_ref,definicion_version,definicion_sha256,raiz_sha256)
     REFERENCES vec_contratacion_temporal.seguimiento_raiz_v2(seguimiento_ref,organizacion_ref,
         expediente_ref,relacion_ref,definicion_ref,definicion_version,definicion_sha256,raiz_sha256),
 FOREIGN KEY (seguimiento_ref,version_anterior,estado_anterior_sha256)
     REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento,estado_sha256),
 CHECK ((version_seguimiento=0 AND version_anterior IS NULL AND estado_anterior_sha256 IS NULL)
     OR (version_seguimiento>0 AND version_anterior IS NOT NULL AND estado_anterior_sha256 IS NOT NULL
         AND version_anterior=version_seguimiento-1 AND estado_anterior_sha256 ~ '^[0-9a-f]{64}$'
         AND estado_anterior_sha256<>repeat('0',64))),
 CHECK (estado_sha256=encode(sha256(estado_canonico),'hex')),
 CHECK ((estado_json->'referencia'=to_jsonb(seguimiento_ref)
     AND estado_json->'organizacion_ref'=to_jsonb(organizacion_ref)
     AND estado_json->'expediente_ref'=to_jsonb(expediente_ref)
     AND estado_json->'relacion_ref'=to_jsonb(relacion_ref)
     AND estado_json->'version'=to_jsonb(version_seguimiento)
     AND estado_json->'huella_raiz_sha256'=to_jsonb(raiz_sha256)
     AND estado_json->'definicion'=jsonb_build_object('referencia',definicion_ref,
         'version',definicion_version,'huella_sha256',definicion_sha256)) IS TRUE)
);
-- No raíz huérfana confirmable: publicación y estado inicial en la misma TX.
ALTER TABLE vec_contratacion_temporal.seguimiento_raiz_v2
 ADD CONSTRAINT seguimiento_raiz_v2_inicial_fk FOREIGN KEY (seguimiento_ref,version_inicial)
 REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento)
 DEFERRABLE INITIALLY DEFERRED;

DO $seguridad$
DECLARE tabla text; rol record;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2'] LOOP
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
  EXECUTE format('CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',tabla);
  -- Elimina solo grants heredados por defaults SOBRE LAS TABLAS NUEVAS.
  -- No modifica default ACL global ni permisos/roles de otras capacidades.
  FOR rol IN SELECT DISTINCT r.rolname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      CROSS JOIN LATERAL aclexplode(c.relacl) a JOIN pg_roles r ON r.oid=a.grantee
      WHERE n.nspname='vec_contratacion_temporal' AND c.relname=tabla
        AND a.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',tabla,rol.rolname);
  END LOOP;
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a
      WHERE n.nspname='vec_contratacion_temporal' AND c.relname=tabla
        AND (a.grantee<>c.relowner OR a.grantor<>c.relowner)) THEN
   RAISE EXCEPTION 'CT72: ACL final no exclusiva' USING ERRCODE='55000';
  END IF;
  EXECUTE format('COMMENT ON TABLE vec_contratacion_temporal.%I IS %L',tabla,
      'CT72:fundacion-seguimiento-v2;sin-api-registro;codec-dominio-pendiente');
 END LOOP;
END $seguridad$;
-- No GRANT EXECUTE, no SELECT Personal, no cambio de expediente actual,
-- no consumo ficticio, no recibo ni función de negocio incompleta expuesta.
COMMIT;
