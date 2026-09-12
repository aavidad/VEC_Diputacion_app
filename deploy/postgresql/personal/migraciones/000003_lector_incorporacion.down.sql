\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000003:lector:v1',0));
-- El lock precede a la comprobación de historia: ninguna lectura puede colarse entre ambas.
LOCK TABLE vec_personal.auditoria_lectura_incorporacion IN ACCESS EXCLUSIVE MODE;

DO $proteger$
DECLARE
    f oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v1(text,text,text,bigint,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
    t oid:='vec_personal.auditoria_lectura_incorporacion'::regclass;
    marca text:=obj_description(t,'pg_class'); ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
    IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
          AND prosecdef AND NOT proretset AND prorettype='jsonb'::regtype AND provolatile='v'
          AND proconfig=ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s'])
       OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
          WHERE n.nspname='vec_personal' AND p.proname='acreditar_alta_ejercicio_v1' AND p.oid<>f)
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=current_user::regrole)
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t AND relkind='r' AND relowner=current_user::regrole
          AND relrowsecurity AND relforcerowsecurity)
       OR marca IS NULL OR marca NOT IN ('Personal000003:v1:ct_usage=ausente','Personal000003:v1:ct_usage=presente') THEN
        RAISE EXCEPTION 'retirada Personal 000003: objetos incompatibles' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_personal.auditoria_lectura_incorporacion) THEN
        RAISE EXCEPTION 'retirada Personal 000003: existe historia lectora' USING ERRCODE='55000';
    END IF;
    IF (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='propietario' AND polcmd='*'
           AND polpermissive AND polroles=ARRAY[current_user::regrole::oid]
           AND pg_get_expr(polqual,polrelid)='true' AND pg_get_expr(polwithcheck,polrelid)='true')
       OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal
           AND tgname='inmutable' AND tgtype=58 AND tgenabled='O'
           AND tgfoid='vec_personal.rechazar_mutacion_alta_ejercicio_v1()'::regprocedure)
       OR EXISTS (SELECT 1 FROM pg_class c,LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a
           WHERE c.oid=t AND a.grantee<>c.relowner)
       OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
           WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,ct,'vec_personal_ejecutor'::regrole)
               OR (a.grantee<>p.proowner AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner))))
       OR NOT EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(p.proacl) a
           WHERE p.oid=f AND a.grantee=ct AND a.privilege_type='EXECUTE')
       OR NOT EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(p.proacl) a
           WHERE p.oid=f AND a.grantee='vec_personal_ejecutor'::regrole AND a.privilege_type='EXECUTE')
       OR has_schema_privilege(ct,'vec_personal','CREATE')
       OR NOT EXISTS (SELECT 1 FROM pg_namespace n,LATERAL aclexplode(n.nspacl) a
           WHERE n.nspname='vec_personal' AND a.grantee=ct AND a.privilege_type='USAGE'
             AND NOT a.is_grantable AND a.grantor=n.nspowner) THEN
        RAISE EXCEPTION 'retirada Personal 000003: ACL/RLS/trigger incompatibles' USING ERRCODE='55000';
    END IF;
    -- SQL con dependencias normales, sin confundir OID de roles/tablas con funciones.
    -- PL/pgSQL no registra necesariamente llamadas: se comprueban también cuerpo y tipos por catálogo.
    IF EXISTS (SELECT 1 FROM pg_depend WHERE refclassid='pg_proc'::regclass AND refobjid=f AND deptype<>'i')
       OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid<>f AND
           (p.prosrc LIKE '%acreditar_alta_ejercicio_v1%'
            OR p.prosrc LIKE '%auditoria_lectura_incorporacion%'
            OR p.prorettype=(SELECT reltype FROM pg_class WHERE oid=t)
            OR (SELECT reltype FROM pg_class WHERE oid=t)=ANY(p.proargtypes::oid[])))
       OR EXISTS (SELECT 1 FROM pg_depend d JOIN pg_constraint c ON d.classid='pg_constraint'::regclass AND d.objid=c.oid
           WHERE d.refclassid='pg_class'::regclass AND d.refobjid=t AND c.conrelid<>t)
       OR EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_class'::regclass AND d.refobjid=t
           AND d.classid='pg_rewrite'::regclass) THEN
        RAISE EXCEPTION 'retirada Personal 000003: consumidor, tipo o dependencia vigente' USING ERRCODE='55000';
    END IF;
    -- Si introdujimos USAGE, no lo retiramos bajo otros consumidores nuevos.
    IF marca='Personal000003:v1:ct_usage=ausente' AND (
        EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
            LATERAL aclexplode(p.proacl) a WHERE n.nspname='vec_personal' AND p.oid<>f AND a.grantee=ct)
        OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.proowner=ct AND p.prosrc LIKE '%vec_personal%')) THEN
        RAISE EXCEPTION 'retirada Personal 000003: USAGE requerido por otro consumidor CT' USING ERRCODE='55000';
    END IF;
    EXECUTE format('DROP FUNCTION %s RESTRICT',f::regprocedure);
    DROP TABLE vec_personal.auditoria_lectura_incorporacion RESTRICT;
    IF marca='Personal000003:v1:ct_usage=ausente' THEN
        REVOKE USAGE ON SCHEMA vec_personal FROM vec_contratacion_temporal_propietario RESTRICT;
    END IF;
END
$proteger$;
-- Se conservan codecs, cinco tablas originales, triggers compartidos, AD3-28 y roles.
COMMIT;
