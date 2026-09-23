\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000001',0));
-- Bootstrap explícito del módulo. No crea cuentas LOGIN ni abre conexiones.
CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_cronos_v1_migrador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_cronos_v1_propietario TO vec_cronos_v1_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
CREATE SCHEMA vec_cronos_v1 AUTHORIZATION vec_cronos_v1_propietario;
REVOKE ALL ON SCHEMA vec_cronos_v1 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_cronos_v1_ejecutor;
SET LOCAL ROLE vec_cronos_v1_propietario;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_cronos_v1_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_cronos_v1_propietario IN SCHEMA vec_cronos_v1 REVOKE ALL ON TABLES FROM PUBLIC;

CREATE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
    RAISE EXCEPTION 'historia Cronos inmutable' USING ERRCODE='55000';
END
$f$;

CREATE TABLE vec_cronos_v1.marcaje_original (
    marcaje_ref text PRIMARY KEY,
    empleado_ref text NOT NULL,
    clave_operacion text NOT NULL UNIQUE,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 4096),
    material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    movimiento text NOT NULL CHECK (movimiento IN ('entrada','salida','inicio_pausa','fin_pausa')),
    instante_utc timestamptz(6) NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    auditoria_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.marcaje_historia (
    marcaje_ref text PRIMARY KEY REFERENCES vec_cronos_v1.marcaje_original,
    empleado_ref text NOT NULL,
    version smallint NOT NULL CHECK (version=1),
    material_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.marcaje_outbox (
    evento_ref text PRIMARY KEY,
    marcaje_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.marcaje_original,
    empleado_ref text NOT NULL,
    tipo text NOT NULL CHECK (tipo='cronos.marcaje.propio.registrado'),
    carga_json jsonb NOT NULL CHECK (jsonb_typeof(carga_json)='object'),
    creada_en timestamptz(6) NOT NULL
);
-- Cada replay obtiene autorización nueva y conserva la evidencia de acceso.
CREATE TABLE vec_cronos_v1.marcaje_acceso (
    decision_ref text PRIMARY KEY,
    marcaje_ref text NOT NULL REFERENCES vec_cronos_v1.marcaje_original,
    empleado_ref text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    consultada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['marcaje_original','marcaje_historia','marcaje_outbox','marcaje_acceso'] LOOP
        EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
        EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
        EXECUTE format('CREATE POLICY lectura_propia ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (empleado_ref = nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
        EXECUTE format('CREATE POLICY adicion_propia ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (empleado_ref = nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
        EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador',tabla);
    END LOOP;
END
$seguridad$;
COMMIT;
