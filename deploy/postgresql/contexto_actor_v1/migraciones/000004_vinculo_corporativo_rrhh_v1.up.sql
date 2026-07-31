-- Historia y puntero opacos del vinculo corporativo RRHH C2.2-B.
-- Documento autonomo: no publica, selecciona ni concede acceso de negocio.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $precondiciones_minimas$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vinculo corporativo RRHH V1 requiere migracion superusuario';
    ELSIF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR pg_catalog.current_setting('server_encoding') <> 'UTF8' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'vinculo corporativo RRHH V1 requiere PostgreSQL 18 y UTF8';
    END IF;
END
$precondiciones_minimas$;

-- Barreras globales A -> B -> C; ninguna decision de catalogo se toma antes.
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0));
SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1', 0));

LOCK TABLE vec_contexto_actor_v1.procedencias,
    vec_contexto_actor_v1.proyeccion_cuenta_versiones,
    vec_contexto_actor_v1.persona_versiones IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.perfil_versiones,
    vec_contexto_actor_v1.vinculo_contexto_versiones IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones,
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 IN SHARE MODE;
LOCK TABLE pg_catalog.pg_authid, pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting, pg_catalog.pg_database,
    pg_catalog.pg_class, pg_catalog.pg_attribute, pg_catalog.pg_attrdef,
    pg_catalog.pg_index, pg_catalog.pg_namespace, pg_catalog.pg_language,
    pg_catalog.pg_collation, pg_catalog.pg_proc, pg_catalog.pg_type,
    pg_catalog.pg_default_acl, pg_catalog.pg_description,
    pg_catalog.pg_seclabel, pg_catalog.pg_init_privs,
    pg_catalog.pg_depend, pg_catalog.pg_shdepend,
    pg_catalog.pg_constraint, pg_catalog.pg_trigger, pg_catalog.pg_policy,
    pg_catalog.pg_inherits, pg_catalog.pg_rewrite, pg_catalog.pg_am,
    pg_catalog.pg_tablespace, pg_catalog.pg_publication,
    pg_catalog.pg_publication_namespace, pg_catalog.pg_publication_rel,
    pg_catalog.pg_subscription_rel, pg_catalog.pg_statistic_ext IN SHARE MODE;

DO $acreditar_base_y_ausencia$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    observado text;
BEGIN
    IF pg_catalog.current_setting('timezone') <> 'UTC'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_authid
                       WHERE rolname=current_user AND rolsuper)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
                       WHERE oid=esquema AND nspowner=propietario) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='entorno de vinculo corporativo RRHH V1 no reacreditado';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
         WHERE oid IN (
           'vec_contexto_actor_v1.procedencias'::regclass,
           'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
           'vec_contexto_actor_v1.persona_versiones'::regclass,
           'vec_contexto_actor_v1.perfil_versiones'::regclass,
           'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_versiones'::regclass,
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
         ) AND relkind='r' AND relpersistence='p' AND relowner=propietario) <> 7
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
            WHERE oid IN (
              'vec_contexto_actor_v1.referencia_valida(text,text)'::regprocedure,
              'vec_contexto_actor_v1.procedencia_valida(text,numeric,text,text)'::regprocedure,
              'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure,
              'vec_contexto_actor_v1.instante_valido(timestamptz)'::regprocedure,
              'vec_contexto_actor_v1.rechazar_mutacion_historia()'::regprocedure,
              'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure,
              'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
              'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure
            ) AND proowner=propietario) <> 8 THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='dependencias de vinculo corporativo RRHH V1 no acreditadas';
    END IF;

    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute a
         WHERE a.attrelid IN (
           'vec_contexto_actor_v1.procedencias'::regclass,
           'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
           'vec_contexto_actor_v1.persona_versiones'::regclass,
           'vec_contexto_actor_v1.perfil_versiones'::regclass,
           'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
           'vec_contexto_actor_v1.organizacion_versiones'::regclass,
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
           AND a.attnum>0 AND NOT a.attisdropped) <> 56
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
            LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) a
            WHERE c.oid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
              AND a.grantee<>propietario)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a,
            LATERAL pg_catalog.aclexplode(a.attacl) x
            WHERE a.attrelid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass,
              'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
              AND a.attnum>0 AND NOT a.attisdropped AND x.grantee<>propietario)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger t
            WHERE t.tgrelid IN (
              'vec_contexto_actor_v1.procedencias'::regclass,
              'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
              'vec_contexto_actor_v1.persona_versiones'::regclass,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
              'vec_contexto_actor_v1.organizacion_versiones'::regclass)
              AND NOT t.tgisinternal) <> 13
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint c
            WHERE c.conrelid='vec_contexto_actor_v1.organizacion_versiones'::regclass
              AND c.conname='organizacion_versiones_procedencia_uq'
              AND c.contype='u' AND c.conkey=ARRAY[1,2,3,4,5,6]::smallint[]
              AND c.convalidated AND NOT c.condeferrable AND NOT c.condeferred)
       OR (SELECT pg_catalog.count(*) FROM
            (VALUES
              ('referencia_valida(text,text)','sql','i',false),
              ('procedencia_valida(text,numeric,text,text)','sql','i',false),
              ('organizacion_ref_valida(text)','sql','i',false),
              ('instante_valido(timestamp with time zone)','sql','i',false),
              ('rechazar_mutacion_historia()','plpgsql','v',false),
              ('rechazar_truncado()','plpgsql','v',false),
              ('serializar_mutacion_punteros_actuales_v2()','plpgsql','v',true),
              ('avanzar_generacion_punteros_actuales_v2()','plpgsql','v',true)
            ) e(firma,lenguaje,volatilidad,definidor)
            JOIN pg_catalog.pg_proc p ON p.oid=
              pg_catalog.to_regprocedure('vec_contexto_actor_v1.'||e.firma)
            JOIN pg_catalog.pg_language l ON l.oid=p.prolang
           WHERE p.proowner=propietario AND l.lanname=e.lenguaje
             AND p.provolatile=e.volatilidad::"char" AND p.prosecdef=e.definidor
             AND p.proconfig=ARRAY['search_path=pg_catalog']::text[]) <> 8
       OR (SELECT pg_catalog.count(*) FROM
             vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
            WHERE control_id AND generacion>=0 AND pg_catalog.scale(generacion)=0
              AND vec_contexto_actor_v1.instante_valido(actualizada_en)) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE='55000',
          MESSAGE='forma estructural del predecesor 000001..000003 no acreditada';
    END IF;

    WITH objetos AS (
      SELECT pg_catalog.format('col|%s|%s|%s|%s|%s|%s|%s',c.relname,a.attnum,
               a.attname,pg_catalog.format_type(a.atttypid,a.atttypmod),
               a.attnotnull,a.attstorage,a.attcompression) e
        FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
       WHERE c.oid IN ('vec_contexto_actor_v1.procedencias'::regclass,
         'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
         'vec_contexto_actor_v1.persona_versiones'::regclass,
         'vec_contexto_actor_v1.perfil_versiones'::regclass,
         'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
         'vec_contexto_actor_v1.organizacion_versiones'::regclass,
         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
         AND a.attnum>0 AND NOT a.attisdropped
      UNION ALL
      SELECT pg_catalog.format('con|%s|%s|%s|%s',c.conrelid::regclass::text,
               c.conname,c.contype,pg_catalog.pg_get_constraintdef(c.oid,false))
        FROM pg_catalog.pg_constraint c WHERE c.conrelid IN (
         'vec_contexto_actor_v1.procedencias'::regclass,
         'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
         'vec_contexto_actor_v1.persona_versiones'::regclass,
         'vec_contexto_actor_v1.perfil_versiones'::regclass,
         'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
         'vec_contexto_actor_v1.organizacion_versiones'::regclass,
         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass)
      UNION ALL
      SELECT pg_catalog.format('trg|%s|%s|%s|%s',t.tgrelid::regclass::text,
               t.tgname,t.tgtype,t.tgfoid::regprocedure::text)
        FROM pg_catalog.pg_trigger t WHERE t.tgrelid IN (
         'vec_contexto_actor_v1.procedencias'::regclass,
         'vec_contexto_actor_v1.proyeccion_cuenta_versiones'::regclass,
         'vec_contexto_actor_v1.persona_versiones'::regclass,
         'vec_contexto_actor_v1.perfil_versiones'::regclass,
         'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass,
         'vec_contexto_actor_v1.organizacion_versiones'::regclass)
         AND NOT t.tgisinternal
      UNION ALL
      SELECT pg_catalog.format('fun|%s|%s',p.oid::regprocedure::text,
               pg_catalog.pg_get_functiondef(p.oid)) FROM pg_catalog.pg_proc p
       WHERE p.oid IN ('vec_contexto_actor_v1.referencia_valida(text,text)'::regprocedure,
         'vec_contexto_actor_v1.procedencia_valida(text,numeric,text,text)'::regprocedure,
         'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure,
         'vec_contexto_actor_v1.instante_valido(timestamptz)'::regprocedure,
         'vec_contexto_actor_v1.rechazar_mutacion_historia()'::regprocedure,
         'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure,
         'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
         'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure)
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.string_agg(e,E'\n' ORDER BY e),'UTF8')),'hex') INTO observado
      FROM objetos;
    IF observado IS DISTINCT FROM '198579de9a73824c68c7b57a4dbf623c57f758beb8151b01ade2d73560973bd8' THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='manifiesto simbolico del predecesor no acreditado',
        DETAIL='huella observada '||coalesce(observado,'<ausente>');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid='vec_contexto_actor_v1.organizacion_versiones'::regclass
           AND conname='organizacion_versiones_procedencia_uq'
           AND contype='u' AND convalidated AND NOT condeferrable
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE relnamespace=esquema AND relname IN (
           'vinculo_corporativo_versiones','vinculo_corporativo_actual',
           'perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'
         )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE connamespace=esquema AND conname IN (
           'perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq'
         )
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_type
         WHERE typnamespace=esquema AND typname IN (
           'vinculo_corporativo_versiones','_vinculo_corporativo_versiones',
           'vinculo_corporativo_actual','_vinculo_corporativo_actual'
         )
    ) OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication WHERE puballtables)
      OR EXISTS (SELECT 1 FROM pg_catalog.pg_publication_namespace
                  WHERE pnnspid=esquema) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='base o ausencia nominal de vinculo corporativo RRHH V1 no acreditada';
    END IF;
END
$acreditar_base_y_ausencia$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

ALTER TABLE vec_contexto_actor_v1.perfil_versiones
    ADD CONSTRAINT perfil_versiones_persona_uq
    UNIQUE (perfil_ref, version, persona_ref);
ALTER TABLE vec_contexto_actor_v1.vinculo_contexto_versiones
    ADD CONSTRAINT vinculo_contexto_versiones_actor_uq
    UNIQUE (vinculo_ref, version, cuenta_ref, perfil_ref, persona_ref);

CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones (
    vinculo_corporativo_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    cuenta_ref text NOT NULL,
    cuenta_version numeric(20,0) NOT NULL,
    persona_ref text NOT NULL,
    persona_version numeric(20,0) NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20,0) NOT NULL,
    vinculo_contexto_ref text NOT NULL,
    vinculo_contexto_version numeric(20,0) NOT NULL,
    organizacion_ref text NOT NULL,
    organizacion_version numeric(20,0) NOT NULL,
    organizacion_procedencia_ref text NOT NULL,
    organizacion_procedencia_version numeric(20,0) NOT NULL,
    organizacion_procedencia_huella_sha256 text NOT NULL,
    organizacion_procedencia_autoridad text NOT NULL,
    superficie text NOT NULL,
    uso text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    CONSTRAINT vinculo_corporativo_versiones_pk
      PRIMARY KEY (vinculo_corporativo_ref, version),
    CONSTRAINT vinculo_corporativo_versiones_actual_uq UNIQUE
      (cuenta_ref, superficie, uso, vinculo_corporativo_ref, version),
    CONSTRAINT vinculo_corporativo_versiones_cuenta_fk FOREIGN KEY
      (cuenta_ref, cuenta_version) REFERENCES
      vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version)
      MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_persona_fk FOREIGN KEY
      (persona_ref, persona_version) REFERENCES
      vec_contexto_actor_v1.persona_versiones(persona_ref,version) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_perfil_persona_fk FOREIGN KEY
      (perfil_ref,perfil_version,persona_ref) REFERENCES
      vec_contexto_actor_v1.perfil_versiones(perfil_ref,version,persona_ref)
      MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_vinculo_contexto_fk FOREIGN KEY
      (vinculo_contexto_ref,vinculo_contexto_version,cuenta_ref,perfil_ref,persona_ref)
      REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones
      (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_organizacion_fk FOREIGN KEY
      (organizacion_ref,organizacion_version,organizacion_procedencia_ref,
       organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
       organizacion_procedencia_autoridad) REFERENCES
      vec_contexto_actor_v1.organizacion_versiones
      (organizacion_ref,version,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_procedencia_fk FOREIGN KEY
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) REFERENCES vec_contexto_actor_v1.procedencias
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_corporativo_versiones_ref_ck CHECK
      (vec_contexto_actor_v1.referencia_valida(vinculo_corporativo_ref,'vcr_')),
    CONSTRAINT vinculo_corporativo_versiones_version_ck CHECK
      (version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_cuenta_version_ck CHECK
      (cuenta_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_persona_version_ck CHECK
      (persona_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_perfil_version_ck CHECK
      (perfil_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_vinculo_contexto_version_ck CHECK
      (vinculo_contexto_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_version_ck CHECK
      (organizacion_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_versiones_superficie_ck CHECK
      (superficie='interna_corporativa'),
    CONSTRAINT vinculo_corporativo_versiones_uso_ck CHECK (uso='consulta_rrhh'),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(
       organizacion_procedencia_ref,organizacion_procedencia_version,
       organizacion_procedencia_huella_sha256,organizacion_procedencia_autoridad)),
    CONSTRAINT vinculo_corporativo_versiones_organizacion_autoridad_ck CHECK
      (organizacion_procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_corporativo_versiones_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(procedencia_ref,
       procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CONSTRAINT vinculo_corporativo_versiones_procedencia_autoridad_ck CHECK
      (procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_corporativo_versiones_estado_ck CHECK
      (estado IN ('activo','revocado')),
    CONSTRAINT vinculo_corporativo_versiones_vigente_desde_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CONSTRAINT vinculo_corporativo_versiones_vigente_hasta_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CONSTRAINT vinculo_corporativo_versiones_ventana_ck CHECK
      (vigente_hasta > vigente_desde)
);

CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_actual (
    cuenta_ref text NOT NULL,
    superficie text NOT NULL,
    uso text NOT NULL,
    vinculo_corporativo_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    CONSTRAINT vinculo_corporativo_actual_pk
      PRIMARY KEY (cuenta_ref,superficie,uso),
    CONSTRAINT vinculo_corporativo_actual_superficie_ck CHECK
      (superficie='interna_corporativa'),
    CONSTRAINT vinculo_corporativo_actual_uso_ck CHECK (uso='consulta_rrhh'),
    CONSTRAINT vinculo_corporativo_actual_version_ck CHECK
      (version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_corporativo_actual_version_fk FOREIGN KEY
      (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version) REFERENCES
      vec_contexto_actor_v1.vinculo_corporativo_versiones
      (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version) MATCH FULL
);

ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones
    AS PERMISSIVE FOR ALL TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER='vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER='vec_contexto_actor_v1_propietario');
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.vinculo_corporativo_actual
    AS PERMISSIVE FOR ALL TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER='vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER='vec_contexto_actor_v1_propietario');

CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones FOR EACH ROW
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE
    ON vec_contexto_actor_v1.vinculo_corporativo_versiones FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_actual_no_truncable_v2 BEFORE TRUNCATE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER serializar_mutacion_punteros_actuales_v2
    BEFORE INSERT OR UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion_punteros_actuales_v2
    AFTER INSERT OR UPDATE OR DELETE
    ON vec_contexto_actor_v1.vinculo_corporativo_actual FOR EACH STATEMENT
    EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();

REVOKE ALL ON TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones,
    vec_contexto_actor_v1.vinculo_corporativo_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON TYPE vec_contexto_actor_v1.vinculo_corporativo_versiones,
    vec_contexto_actor_v1.vinculo_corporativo_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;

DO $postcondiciones$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass;
BEGIN
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class
         WHERE oid IN (versiones,actual) AND relkind='r' AND relowner=propietario
           AND relrowsecurity AND relforcerowsecurity) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid IN (versiones,actual)
              AND polname='acceso_propietario_exacto' AND polcmd='*'
              AND polpermissive AND polroles=ARRAY[propietario]::oid[]
              AND polqual IS NOT NULL AND polwithcheck IS NOT NULL) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid IN (versiones,actual)) <> 60
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE connamespace=esquema AND conname IN
             ('perfil_versiones_persona_uq','vinculo_contexto_versiones_actor_uq')) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_constraint
            WHERE conrelid IN (versiones,actual) AND contype='f'
              AND confmatchtype='f' AND confupdtype='a' AND confdeltype='a'
              AND NOT condeferrable AND NOT condeferred AND convalidated) <> 7
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index
            WHERE indrelid IN (versiones,actual)) <> 3
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class x ON x.oid=i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid=x.relam
            WHERE i.indrelid IN (versiones,actual,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass)
              AND x.relname IN (
                'vinculo_corporativo_versiones_pk',
                'vinculo_corporativo_versiones_actual_uq',
                'vinculo_corporativo_actual_pk','perfil_versiones_persona_uq',
                'vinculo_contexto_versiones_actor_uq')
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.reloptions IS NULL AND am.amname='btree') <> 5
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid IN (versiones,actual) AND NOT tgisinternal) <> 5
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute
                   WHERE attrelid IN (versiones,actual) AND attnum>0
                     AND NOT attisdropped AND (attacl IS NOT NULL
                       OR atthasdef OR attstorage<>(SELECT typstorage
                         FROM pg_catalog.pg_type WHERE oid=atttypid)
                       OR attcompression<>''))
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_type t
            WHERE t.oid IN (
              (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
              (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual),
              (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=versiones),
              (SELECT typarray FROM pg_catalog.pg_type WHERE typrelid=actual))
              AND t.typowner=propietario AND (
                (t.typrelid IN (versiones,actual) AND NOT EXISTS (
                  SELECT 1 FROM pg_catalog.aclexplode(coalesce(t.typacl,
                    pg_catalog.acldefault('T',t.typowner))) a
                   WHERE a.grantee<>propietario))
                OR (t.typrelid=0 AND t.typacl IS NULL AND t.typelem IN (
                  (SELECT reltype FROM pg_catalog.pg_class WHERE oid=versiones),
                  (SELECT reltype FROM pg_catalog.pg_class WHERE oid=actual))))
              ) <> 4
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
            JOIN pg_catalog.pg_am am ON am.oid=c.relam
            WHERE c.oid IN (versiones,actual) AND c.relowner=propietario
              AND c.reltablespace=0 AND c.reloptions IS NULL
              AND am.amname='heap' AND c.reltoastrelid<>0) <> 2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class t
            JOIN pg_catalog.pg_class p ON p.reltoastrelid=t.oid
            JOIN pg_catalog.pg_am am ON am.oid=t.relam
            WHERE p.oid IN (versiones,actual) AND t.relkind='t'
              AND t.relowner=propietario AND t.reltablespace=0
              AND t.reloptions IS NULL AND am.amname='heap') <> 2
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
            LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
              pg_catalog.acldefault('r',c.relowner))) a
            WHERE c.oid IN (versiones,actual) AND a.grantee<>propietario)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a,
            LATERAL pg_catalog.aclexplode(a.attacl) x
            WHERE a.attrelid IN (versiones,actual)
              AND a.attnum>0 AND NOT a.attisdropped
              AND x.grantee<>propietario) THEN
        RAISE EXCEPTION USING ERRCODE='55000',
            MESSAGE='postcondiciones de vinculo corporativo RRHH V1 no acreditadas';
    END IF;
END
$postcondiciones$;

COMMIT;
