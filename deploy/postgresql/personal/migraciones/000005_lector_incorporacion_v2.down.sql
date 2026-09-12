\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000005:lector:v2',0));
-- Retirar sólo fachada V2, nunca datos/auditorías ni acceso V1.
DO $proteger$
DECLARE
    f oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
    propietario oid:='vec_personal_propietario'::regrole;
    ct oid:='vec_contratacion_temporal_propietario'::regrole;
    ejecutor oid:='vec_personal_ejecutor'::regrole;
    v1 oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v1(text,text,text,bigint,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
    anterior jsonb; schema_anterior jsonb;
BEGIN
    IF current_user<>'vec_personal_propietario' OR f IS NULL OR v1 IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=propietario)
       OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_personal'::regnamespace
           AND proname='acreditar_alta_ejercicio_v2')<>1 THEN
        RAISE EXCEPTION 'retirada Personal 000005: fachada incompatible' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang WHERE p.oid=f
        AND p.proowner=propietario AND p.prosecdef AND p.prokind='f' AND l.lanname='plpgsql'
        AND NOT p.proretset AND p.prorettype='jsonb'::regtype AND p.provolatile='v' AND p.proparallel='u'
        AND NOT p.proisstrict AND NOT p.proleakproof AND p.pronargs=20 AND p.pronargdefaults=0
        AND p.provariadic=0 AND p.proallargtypes IS NULL AND p.proargmodes IS NULL
        AND p.proargdefaults IS NULL AND p.probin IS NULL AND p.prosqlbody IS NULL AND p.protrftypes IS NULL
        AND p.prosupport=0 AND p.procost=100 AND p.prorows=0
        AND p.proargnames=ARRAY['p_organizacion_ref','p_solicitud_ref','p_expediente_ref','p_version_expediente',
            'p_resultado_ref','p_recibo_ref','p_relacion_ref','p_ocupacion_ref','p_material_sha256','p_unidad_ref',
            'p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version',
            'p_payload','p_sobre','p_evidencia','p_raiz']
        AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s']
        AND obj_description(p.oid,'pg_proc')='Personal000005:lector_incorporacion:v2:ambitos_org_unidad'
        AND octet_length(p.prosrc)=18350
        AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d7973e804e74528b752ad73c2720198e783dddddc0da82615ecaa2a05eca6c27') THEN
        RAISE EXCEPTION 'retirada Personal 000005: definición alterada conservada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (
        WITH esperado(grantor,grantee,privilegio,delegable) AS (
            VALUES (propietario,propietario,'EXECUTE'::text,false),
                   (propietario,ct,'EXECUTE'::text,false),(propietario,ejecutor,'EXECUTE'::text,false)
        ), actual AS (
            SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable
            FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
        ), diferencia AS (
            (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
            UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
        ) SELECT 1 FROM diferencia
    ) THEN
        RAISE EXCEPTION 'retirada Personal 000005: ACL alterada conservada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_depend WHERE refclassid='pg_proc'::regclass AND refobjid=f)
       OR EXISTS (SELECT 1 FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f AND deptype='e')
       OR EXISTS (SELECT 1 FROM pg_proc WHERE oid<>f AND strpos(prosrc,'acreditar_alta_ejercicio_v2')>0) THEN
        RAISE EXCEPTION 'retirada Personal 000005: consumidor o dependencia vigente' USING ERRCODE='55000';
    END IF;
    SELECT to_jsonb(p) INTO STRICT anterior FROM pg_proc p WHERE oid=v1;
    SELECT to_jsonb(n) INTO STRICT schema_anterior FROM pg_namespace n WHERE nspname='vec_personal';
    EXECUTE format('DROP FUNCTION %s RESTRICT',f::regprocedure);
    IF (SELECT to_jsonb(p) FROM pg_proc p WHERE oid=v1) IS DISTINCT FROM anterior
       OR (SELECT to_jsonb(n) FROM pg_namespace n WHERE nspname='vec_personal') IS DISTINCT FROM schema_anterior THEN
        RAISE EXCEPTION 'retirada Personal 000005: V1 o esquema alterados' USING ERRCODE='55000';
    END IF;
END
$proteger$;
-- auditoria_lectura_incorporacion es historia compartida: se conserva íntegra
-- incluso con lecturas V2. 000003 mantiene su protección de tabla/dependencias.
COMMIT;
