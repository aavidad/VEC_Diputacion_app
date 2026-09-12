\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000002:registro:v1',0));

-- No se borran datos ni se interpreta una tabla vacía como ausencia de toda historia.
-- El lock común excluye efectos y migraciones coordinadas; se bloquean las cinco tablas.
LOCK TABLE vec_personal.relacion_alta_ejercicio,vec_personal.ocupacion_alta_ejercicio,
    vec_personal.registro_alta_ejercicio,vec_personal.auditoria_alta_ejercicio,
    vec_personal.outbox_alta_ejercicio IN ACCESS EXCLUSIVE MODE;

DO $proteger$
DECLARE t text; hay_historia boolean;
BEGIN
    FOREACH t IN ARRAY ARRAY['relacion_alta_ejercicio','ocupacion_alta_ejercicio','registro_alta_ejercicio',
        'auditoria_alta_ejercicio','outbox_alta_ejercicio'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
            WHERE n.nspname='vec_personal' AND c.relname=t AND c.relkind='r'
              AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
           OR NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=format('vec_personal.%I',t)::regclass
               AND p.polname='propietario' AND p.polcmd='*' AND p.polpermissive
               AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid]
               AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true')
           OR EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=format('vec_personal.%I',t)::regclass
               AND p.polname<>'propietario') THEN
            RAISE EXCEPTION 'retirada Personal denegada: forma o RLS incompatible' USING ERRCODE='55000';
        END IF;
        -- No permitir que una política alterada oculte filas al propietario FORCE RLS.
        EXECUTE format('SELECT EXISTS (SELECT 1 FROM vec_personal.%I)',t) INTO hay_historia;
        IF hay_historia THEN
            RAISE EXCEPTION 'retirada Personal denegada: historia conservada' USING ERRCODE='55000';
        END IF;
    END LOOP;
    -- Dependencias SQL registradas se protegen además por DROP RESTRICT (nunca CASCADE).
    -- Para llamadas PL/pgSQL textuales no registradas en pg_depend, rechazo conservador.
    IF EXISTS (SELECT 1 FROM pg_proc p
        WHERE p.oid<>'vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)'::regprocedure
          AND strpos(p.prosrc,'registrar_alta_ejercicio_v1')>0) THEN
        RAISE EXCEPTION 'retirada Personal denegada: función consumidora dependiente' USING ERRCODE='55000';
    END IF;
END
$proteger$;

DROP FUNCTION vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea) RESTRICT;
-- Solo dependencias internas conocidas del ciclo recibo/auditoría/outbox, tras ausencia de TODA historia.
ALTER TABLE vec_personal.registro_alta_ejercicio DROP CONSTRAINT registro_alta_auditoria_fk;
ALTER TABLE vec_personal.registro_alta_ejercicio DROP CONSTRAINT registro_alta_outbox_fk;
DROP TABLE vec_personal.outbox_alta_ejercicio RESTRICT;
DROP TABLE vec_personal.auditoria_alta_ejercicio RESTRICT;
DROP TABLE vec_personal.registro_alta_ejercicio RESTRICT;
DROP TABLE vec_personal.ocupacion_alta_ejercicio RESTRICT;
DROP TABLE vec_personal.relacion_alta_ejercicio RESTRICT;
DROP FUNCTION vec_personal.rechazar_mutacion_alta_ejercicio_v1() RESTRICT;
DROP FUNCTION vec_personal.material_registro_alta_valido_v1(jsonb) RESTRICT;
DROP FUNCTION vec_personal.fecha_registro_alta_pg_v1(text) RESTRICT;
DROP FUNCTION vec_personal.claves_registro_alta_v1(jsonb,text[]) RESTRICT;
-- Se conservan 000001, AD3-26, roles, audiencias, claves y todo el gobierno común.
COMMIT;
