\set ON_ERROR_STOP on
-- High exclusivo del ejercicio RRHH RPT con certificado personal mTLS.
-- La fila la provisiona Identidad mediante el canal administrativo privado.
-- No acredita Kerberos, identidad corporativa, ADMIN ni firma documental.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_identidad_sesiones_v1:migracion:base:v1',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_identidad_sesiones_v1:migracion:high-rpt-sintetica:000010',0));
DO $pre$
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer/10000<>18
       OR pg_catalog.current_setting('server_encoding')<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                      WHERE rolname=current_user AND rolsuper)
       OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1') IS NULL
       OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)') IS NULL THEN
        RAISE EXCEPTION 'IS10: precondiciones incompatibles' USING ERRCODE='55000';
    END IF;
END $pre$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

CREATE TABLE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 (
    singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
    politica_ref text NOT NULL UNIQUE CHECK(politica_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'),
    huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$' AND huella_sha256<>repeat('0',64)),
    alcance text NOT NULL CHECK(alcance='solo-sintetico;sin-kerberos;no-corporativa'),
    uso text NOT NULL CHECK(uso='gobierno_categorias_rpt'),
    metodo text NOT NULL CHECK(metodo='certificado'),
    garantia text NOT NULL CHECK(garantia='alto'),
    vigente_desde timestamptz(6) NOT NULL,
    retirar_en timestamptz(6) NOT NULL,
    aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    registrada_por name NOT NULL DEFAULT session_user,
    activa boolean NOT NULL DEFAULT true,
    retirada_en timestamptz(6),
    retirada_por name,
    retiro_ref text CHECK(retiro_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    CHECK(isfinite(vigente_desde) AND isfinite(retirar_en) AND isfinite(registrada_en)
          AND vigente_desde>=registrada_en AND retirar_en>vigente_desde),
    CHECK((activa AND retirada_en IS NULL AND retirada_por IS NULL AND retiro_ref IS NULL)
          OR (NOT activa AND retirada_en IS NOT NULL AND isfinite(retirada_en)
              AND retirada_en>=registrada_en AND retirada_por IS NOT NULL AND retiro_ref IS NOT NULL))
);
ALTER TABLE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
    TO vec_identidad_sesiones_v1_propietario
    USING(current_user='vec_identidad_sesiones_v1_propietario')
    WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.proteger_politica_high_rrhh_rpt_sintetica_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
    IF TG_OP='INSERT' THEN
        IF NEW.activa IS DISTINCT FROM true
           OR NEW.retirada_en IS NOT NULL OR NEW.retirada_por IS NOT NULL OR NEW.retiro_ref IS NOT NULL
           OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1 p
                     WHERE p.politica_ref=NEW.politica_ref)
           OR EXISTS(SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1 s
                     WHERE s.politica_garantia_ref=NEW.politica_ref) THEN
            RAISE EXCEPTION 'IS10: politica reutilizada o invalida' USING ERRCODE='55000';
        END IF;
        NEW.registrada_en:=clock_timestamp();
        NEW.registrada_por:=session_user;
        RETURN NEW;
    END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.activa AND NEW.activa IS FALSE
           AND NEW.retiro_ref IS NOT NULL AND NEW.retiro_ref IS DISTINCT FROM OLD.aprobacion_ref
           AND (to_jsonb(NEW)-ARRAY['activa','retirada_en','retirada_por','retiro_ref'])
               IS NOT DISTINCT FROM
               (to_jsonb(OLD)-ARRAY['activa','retirada_en','retirada_por','retiro_ref']) THEN
            NEW.retirada_en:=clock_timestamp();
            NEW.retirada_por:=session_user;
            RETURN NEW;
        END IF;
    END IF;
    RAISE EXCEPTION 'IS10: politica inmutable o ya retirada' USING ERRCODE='55000';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.proteger_politica_high_rrhh_rpt_sintetica_v1() FROM PUBLIC;
CREATE TRIGGER politica_rpt_control BEFORE INSERT OR UPDATE OR DELETE
    ON vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
    FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.proteger_politica_high_rrhh_rpt_sintetica_v1();
CREATE TRIGGER politica_rpt_no_truncar BEFORE TRUNCATE
    ON vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

CREATE FUNCTION vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(
    p_ref text,p_huella text,p_instante timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE p record; ahora timestamptz(6);
BEGIN
    IF p_ref IS NULL OR p_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$' OR p_huella=repeat('0',64)
       OR p_instante IS NULL OR NOT isfinite(p_instante)
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
    -- FOR SHARE mantiene la retirada bloqueada hasta acabar el efecto.
    -- Una instantánea anterior a la retirada aborta en SERIALIZABLE.
    SELECT * INTO p FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
     WHERE singleton FOR SHARE;
    ahora:=clock_timestamp();
    RETURN FOUND AND p.activa AND p.politica_ref=p_ref AND p.huella_sha256=p_huella
       AND p.alcance='solo-sintetico;sin-kerberos;no-corporativa'
       AND p.uso='gobierno_categorias_rpt' AND p.metodo='certificado' AND p.garantia='alto'
       AND p_instante>=p.vigente_desde AND p_instante<=ahora
       AND p_instante<p.retirar_en AND ahora>=p.vigente_desde AND ahora<p.retirar_en
       AND NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1 anterior
                      WHERE anterior.politica_ref=p_ref);
EXCEPTION WHEN data_exception OR read_only_sql_transaction THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(text,text,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(
    p_superficie text,p_metodo text,p_garantia text,p_privilegiada boolean,
    p_ref text,p_huella text,p_instante timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
    IF p_superficie='interna_corporativa' AND p_metodo='certificado' AND p_garantia='alto' THEN
        RETURN p_privilegiada IS FALSE AND
            vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(p_ref,p_huella,p_instante);
    END IF;
    -- La misma política sintética jamás acredita ADMIN, otras superficies,
    -- otro método o una elevación de Substantial.
    RETURN NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 p
                      WHERE p.politica_ref=p_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(text,text,text,boolean,text,text,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(
    p_ref text,p_huella text,p_vigente_desde timestamptz,p_retirar_en timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE p record;
BEGIN
    IF NOT COALESCE(vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(
        p_ref,p_huella,clock_timestamp()),false) THEN RETURN false; END IF;
    SELECT * INTO p FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 WHERE singleton;
    RETURN p.vigente_desde IS NOT DISTINCT FROM p_vigente_desde
       AND p.retirar_en IS NOT DISTINCT FROM p_retirar_en;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(text,text,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(text,text,timestamptz,timestamptz)
    TO vec_identidad_sesiones_v1_revalidador;

-- Las cinco preimágenes proceden de pg_get_functiondef en PostgreSQL 18.4
-- desechable restaurado del snapshot sintético, con IS6 ya instalada.
-- Se cotejan definición completa, propietario, ACL, configuración y todas
-- las dependencias, antes de reconstruir la definición REAL de cada función.
DO $ampliar$
DECLARE e record; p record; d text; nuevo text; dependencias text;
BEGIN
 FOR e IN SELECT * FROM (VALUES
  ('vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone,text,text)',
   '0fcbffaeba9c3d8a38c4d6a36b778f3a073805e5efb7ede95ad24fb0174ed4c8',
   'e38b3554987becad60a929e753031cc7b2ba95ce1f535c499de8ee38a4229a14',
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_registrador=X/vec_identidad_sesiones_v1_propietario}',
   '[["BEGIN\n","BEGIN\n    -- IS10: revalidar High sint\u00e9tico con la pol\u00edtica durable exacta.\n    IF vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(\n        p_superficie, p_metodo_observado, p_garantia_observada,\n        p_cuenta_privilegiada, p_politica_garantia_ref,\n        p_politica_garantia_huella_sha256, p_autenticacion_verificada_en\n    ) IS NOT TRUE THEN RETURN; END IF;\n\n"],["    autenticacion_nueva_ref :=\n","    -- IS10: revalidar High sint\u00e9tico con la pol\u00edtica durable exacta.\n    IF vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(\n        p_superficie, p_metodo_observado, p_garantia_observada,\n        p_cuenta_privilegiada, p_politica_garantia_ref,\n        p_politica_garantia_huella_sha256, p_autenticacion_verificada_en\n    ) IS NOT TRUE THEN RETURN; END IF;\n\n    autenticacion_nueva_ref :=\n"]]'::jsonb,
   'aaa8030900d4bf8046a6f1a4d7b05fe7315298648924502341ba4c100fd64c1f'),
  ('vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone,text,text)',
   '81645ff776ed8e5f95e46a129cafec542987c69cbf99e10ac91ed96a38810386',
   'ee3513929bfc44858c3b593435eb15eba0583c2246bcb578b12199486a40799a',
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_registrador=X/vec_identidad_sesiones_v1_propietario}',
   '[[" STABLE SECURITY DEFINER"," SECURITY DEFINER"],["           p_politica_garantia_huella_sha256\n","           p_politica_garantia_huella_sha256\n       AND vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(\n           sesion.superficie,sesion.metodo_observado,sesion.garantia_observada,\n           sesion.cuenta_privilegiada,sesion.politica_garantia_ref,\n           sesion.politica_garantia_huella_sha256,sesion.autenticacion_verificada_en\n       ) IS TRUE\n"]]'::jsonb,
   '59897cfe388aadeac4b7f59b93123a6bb4dd4b01d013fa0f0d95facbd1ee498a'),
  ('vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text,text,text,text,timestamp with time zone,timestamp with time zone)',
   '374711632248d198f2e9ac54525f82e382dba2c42e80bfea0d942bacd293274b',
   'e38b3554987becad60a929e753031cc7b2ba95ce1f535c499de8ee38a4229a14',
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario}',
   '[["    ahora := clock_timestamp();\n","    -- IS10: revalidar High sint\u00e9tico con la pol\u00edtica durable exacta.\n    IF vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(\n        sesion.superficie, sesion.metodo_observado, sesion.garantia_observada,\n        sesion.cuenta_privilegiada, sesion.politica_garantia_ref,\n        sesion.politica_garantia_huella_sha256, sesion.autenticacion_verificada_en\n    ) IS NOT TRUE THEN RETURN false; END IF;\n\n    ahora := clock_timestamp();\n"]]'::jsonb,
   '9a71e197b8300e546a59112cba38ceb6a118a1ee1ea85c688e145d5bd5229be9'),
  ('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
   'a6359500e578352c8c103ecd4771e4eda11f9a191a952601f07161f5abdc4bd1',
   'e38b3554987becad60a929e753031cc7b2ba95ce1f535c499de8ee38a4229a14',
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_identidad_sesiones_v1_revalidador=X/vec_identidad_sesiones_v1_propietario}',
   '[["    ahora := clock_timestamp();\n","    -- IS10: revalidar High sint\u00e9tico con la pol\u00edtica durable exacta.\n    IF vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(\n        sesion.superficie, sesion.metodo_observado, sesion.garantia_observada,\n        sesion.cuenta_privilegiada, sesion.politica_garantia_ref,\n        sesion.politica_garantia_huella_sha256, sesion.autenticacion_verificada_en\n    ) IS NOT TRUE THEN RETURN; END IF;\n\n    ahora := clock_timestamp();\n"]]'::jsonb,
   '39ac988307920ca8b76314214b5fb0b3456a94b412b53bf60677d679dada9a91'),
  ('vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)',
   '536e35476c36552a255ab9f76959a0302b8a9a856ca44f4ae439975f0f431efb',
   'e424b0b13145c318a12243290c356d6d5dbb4f48d45d96f45acc727d33c3c3b6',
   '{vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario,vec_contexto_actor_v1_propietario=X/vec_identidad_sesiones_v1_propietario}',
   '[["e473c7dc0456e97a5865171d1a60cac24ed91fe70d3112c197567b48d46521e3","bab2ba80aa19545e2117a462d9e96a86cadd8336c0705f01f88daa31d70bebda"],["octet_length(base.prosrc) = 8226","octet_length(base.prosrc) = 8644"]]'::jsonb,
   'f62e1e789ec3f96f9afc7e2f6798fc6bd496e77d17aebeb39c18d7417085baa9')
 ) AS manifiesto(firma,definicion_sha256,dependencias_sha256,acl_esperada,parches,nueva_definicion_sha256) LOOP
    SELECT * INTO STRICT p FROM pg_catalog.pg_proc
     WHERE oid=pg_catalog.to_regprocedure(e.firma);
    d:=pg_catalog.pg_get_functiondef(p.oid);
    SELECT encode(sha256(convert_to(jsonb_agg(jsonb_build_array(
       x.objsubid,x.refclassid::regclass::text,
       pg_describe_object(x.refclassid,x.refobjid,x.refobjsubid),x.refobjsubid,x.deptype)
       ORDER BY x.refclassid::regclass::text,
       pg_describe_object(x.refclassid,x.refobjid,x.refobjsubid),
       x.objsubid,x.refobjsubid,x.deptype)::text,'UTF8')),'hex')
      INTO dependencias FROM pg_catalog.pg_depend x
     WHERE x.classid='pg_catalog.pg_proc'::regclass AND x.objid=p.oid;
    IF p.proowner IS DISTINCT FROM 'vec_identidad_sesiones_v1_propietario'::regrole
       OR p.proacl::text IS DISTINCT FROM e.acl_esperada
       OR encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM e.definicion_sha256
       OR dependencias IS DISTINCT FROM e.dependencias_sha256
       OR NOT p.prosecdef OR p.proleakproof OR p.proisstrict
       OR p.prokind<>'f' OR p.proparallel<>'u' THEN
        RAISE EXCEPTION 'IS10: preimagen divergente' USING ERRCODE='55000';
    END IF;
    nuevo:=d;
    DECLARE cambio jsonb; marca text;
    BEGIN
     FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(e.parches) LOOP
      marca:=cambio->>0;
      IF (length(nuevo)-length(replace(nuevo,marca,'')))<>length(marca) THEN
        RAISE EXCEPTION 'IS10: marca no unica' USING ERRCODE='55000';
      END IF;
      nuevo:=replace(nuevo,marca,cambio->>1);
     END LOOP;
    END;
    IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM e.nueva_definicion_sha256 THEN
        RAISE EXCEPTION 'IS10: postimagen no aprobada' USING ERRCODE='55000';
    END IF;
    EXECUTE nuevo;
    IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=p.oid) IS DISTINCT FROM p.proowner
       OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=p.oid) IS DISTINCT FROM p.proacl
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=p.oid) IS DISTINCT FROM p.proconfig
       OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM e.nueva_definicion_sha256
       OR (SELECT encode(sha256(convert_to(jsonb_agg(jsonb_build_array(
           x.objsubid,x.refclassid::regclass::text,
           pg_describe_object(x.refclassid,x.refobjid,x.refobjsubid),x.refobjsubid,x.deptype)
           ORDER BY x.refclassid::regclass::text,
           pg_describe_object(x.refclassid,x.refobjid,x.refobjsubid),
           x.objsubid,x.refobjsubid,x.deptype)::text,'UTF8')),'hex')
          FROM pg_catalog.pg_depend x
          WHERE x.classid='pg_catalog.pg_proc'::regclass AND x.objid=p.oid)
          IS DISTINCT FROM e.dependencias_sha256 THEN
        RAISE EXCEPTION 'IS10: autoridad o dependencias alteradas' USING ERRCODE='55000';
    END IF;
 END LOOP;
END $ampliar$;
-- Sin fila activa al instalar; sin DOWN con historia.
COMMIT;
