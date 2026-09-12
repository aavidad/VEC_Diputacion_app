\set ON_ERROR_STOP on
-- Personal 000004: sólo identidad de NUEVAS relaciones sintéticas.
-- No modifica datos, recibos, material, canon, ocupación ni autorización V3.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
-- Mismo lock común que 000002/000003 y los runtimes; siempre primero.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000004:referencia:v1',0));
LOCK TABLE vec_personal.relacion_alta_ejercicio IN ACCESS EXCLUSIVE MODE;

-- Comparación estructural de CHECK sin adivinar el formato de pg_get_expr.
-- Columna 1 text, como la tabla propietaria. Sin privilegio TEMP de base.
-- El auxiliar se crea y retira en ESTA transacción; nunca se confirma.
DO $auxiliar_previo$
BEGIN
    IF current_user<>'vec_personal_propietario'
       OR (SELECT nspowner FROM pg_namespace WHERE oid='vec_personal'::regnamespace)
          IS DISTINCT FROM 'vec_personal_propietario'::regrole::oid
       OR to_regclass('vec_personal.personal004_check_esperado') IS NOT NULL
       OR to_regtype('vec_personal.personal004_check_esperado') IS NOT NULL THEN
        RAISE EXCEPTION 'Personal 000004: auxiliar previo o propietario incompatible' USING ERRCODE='55000';
    END IF;
END
$auxiliar_previo$;
CREATE TABLE vec_personal.personal004_check_esperado (
    relacion_ref text,
    CONSTRAINT anterior CHECK (relacion_ref ~ '^relacion:personal:ejercicio:[0-9a-f-]{36}$'),
    CONSTRAINT posterior CHECK ((relacion_ref ~ '^relacion:personal:ejercicio:[0-9a-f-]{36}$') OR (relacion_ref ~ '^ref:[0-9a-f]{64}$' AND relacion_ref <> 'ref:'||repeat('0',64)))
);
REVOKE ALL ON TABLE vec_personal.personal004_check_esperado
    FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

DO $referencia$
DECLARE
    f oid:=to_regprocedure('vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
    tabla oid:='vec_personal.relacion_alta_ejercicio'::regclass;
    propietario oid:='vec_personal_propietario'::regrole;
    ejecutor oid:='vec_personal_ejecutor'::regrole;
    p pg_proc%ROWTYPE;
    antes jsonb; despues jsonb; ddl text; nuevo text;
    origen text:=$antes$ref_relacion:='ref:'||encode(sha256(convert_to('vec.personal.relacion.ejercicio.v1:'||id,'UTF8')),'hex');$antes$;
    destino text:=$despues$ref_relacion:='relacion:personal:ejercicio:'||id;$despues$;
    esperada text; siguiente text; con pg_constraint%ROWTYPE;
    otras jsonb;
BEGIN
    IF current_user<>'vec_personal_propietario' OR f IS NULL
       OR (SELECT nspowner FROM pg_namespace WHERE oid='vec_personal'::regnamespace) IS DISTINCT FROM propietario THEN
        RAISE EXCEPTION 'Personal 000004: propietario o firma incompatibles' USING ERRCODE='55000';
    END IF;
    SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
    -- SHA exacto del prosrc de 000002 congelado; no sustituir un derivado.
    IF encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'dd8d3d102200cd7e65d31d94cf4a5a9512869eafab60cafbf4eb083f2d35ff9d'
       OR p.proowner<>propietario OR p.proname<>'registrar_alta_ejercicio_v1'
       OR p.prorettype<>'jsonb'::regtype OR p.proretset OR p.prokind<>'f'
       OR p.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
       OR NOT p.prosecdef OR p.provolatile<>'v' OR p.proparallel<>'u'
       OR p.proisstrict OR p.proleakproof OR p.prosupport<>0
       OR p.procost<>100 OR p.prorows<>0 OR p.provariadic<>0
       OR p.pronargs<>11 OR p.pronargdefaults<>0
       OR p.proargdefaults IS NOT NULL OR p.proallargtypes IS NOT NULL
       OR p.proargmodes IS NOT NULL OR p.probin IS NOT NULL OR p.prosqlbody IS NOT NULL
       OR p.proargnames IS DISTINCT FROM ARRAY['p_material','p_capacidad','p_decision','p_motivo',
          'p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz']::text[]
       OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s']::text[]
       OR (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_personal'::regnamespace
            AND proname='registrar_alta_ejercicio_v1')<>1 THEN
        RAISE EXCEPTION 'Personal 000004: definición original alterada' USING ERRCODE='55000';
    END IF;
    -- ACL completa, incluyendo grantor y opciones; ningún grant lateral.
    IF (SELECT array_agg(a::text ORDER BY a::text)
        FROM unnest(coalesce(p.proacl,acldefault('f',propietario))) a)
       IS DISTINCT FROM
       (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(ARRAY[
           makeaclitem(propietario,propietario,'EXECUTE',false),
           makeaclitem(ejecutor,propietario,'EXECUTE',false)]) a) THEN
        RAISE EXCEPTION 'Personal 000004: ACL de función alterada' USING ERRCODE='42501';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=tabla AND relkind='r'
        AND relowner=propietario AND relrowsecurity AND relforcerowsecurity)
       OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=tabla AND attnum=1
         AND attname='relacion_ref' AND atttypid='text'::regtype AND attnotnull AND NOT attisdropped)
       OR (SELECT count(*) FROM pg_policy WHERE polrelid=tabla)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=tabla AND polname='propietario'
         AND polcmd='*' AND polpermissive AND polroles=ARRAY[propietario]
         AND pg_get_expr(polqual,polrelid)='true' AND pg_get_expr(polwithcheck,polrelid)='true')
       OR EXISTS (SELECT 1 FROM pg_class t,
          LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
          WHERE t.oid=tabla AND (a.grantee<>propietario OR a.grantor<>propietario OR a.is_grantable))
       OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=tabla AND attacl IS NOT NULL)
       OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=tabla AND tgname='historia_inmutable'
          AND NOT tgisinternal AND tgenabled='O' AND tgtype=58 AND tgqual IS NULL
          AND tgnargs=0 AND tgfoid='vec_personal.rechazar_mutacion_alta_ejercicio_v1()'::regprocedure) THEN
        RAISE EXCEPTION 'Personal 000004: tabla, ACL, RLS o inmutabilidad alterada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT esperada FROM pg_constraint
        WHERE conrelid='vec_personal.personal004_check_esperado'::regclass AND conname='posterior';
    SELECT pg_get_constraintdef(oid) INTO STRICT siguiente FROM pg_constraint
        WHERE conrelid='vec_personal.personal004_check_esperado'::regclass AND conname='anterior';
    SELECT * INTO STRICT con FROM pg_constraint WHERE conrelid=tabla
        AND conname='relacion_alta_ejercicio_relacion_ref_check';
    IF con.contype<>'c' OR NOT con.convalidated OR NOT con.conenforced
       OR NOT con.conislocal OR con.coninhcount<>0 OR con.connoinherit
       OR con.condeferrable OR con.condeferred OR con.conkey IS DISTINCT FROM ARRAY[1]::smallint[]
       OR pg_get_constraintdef(con.oid) IS DISTINCT FROM esperada THEN
        RAISE EXCEPTION 'Personal 000004: CHECK previo alterado' USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) INTO otras FROM pg_constraint c
        WHERE c.conrelid=tabla AND c.oid<>con.oid;
    -- FORCE RLS no puede ocultar historia: política/owner comprobados arriba.
    -- Sin DELETE ni CASCADE: ni una relación canónica permite retirar 000004.
    IF EXISTS (SELECT 1 FROM vec_personal.relacion_alta_ejercicio
        WHERE relacion_ref !~ '^relacion:personal:ejercicio:[0-9a-f-]{36}$') THEN
        RAISE EXCEPTION 'Personal 000004: historia canónica conservada, retirada denegada' USING ERRCODE='55000';
    END IF;
    antes:=to_jsonb(p)-'prosrc';
    ddl:=pg_get_functiondef(f);
    IF (length(p.prosrc)-length(replace(p.prosrc,origen,'')))/length(origen)<>1
       OR (length(ddl)-length(replace(ddl,origen,'')))/length(origen)<>1 THEN
        RAISE EXCEPTION 'Personal 000004: sustitución no unívoca' USING ERRCODE='55000';
    END IF;
    nuevo:=replace(ddl,origen,destino);
    ALTER TABLE vec_personal.relacion_alta_ejercicio
        DROP CONSTRAINT relacion_alta_ejercicio_relacion_ref_check RESTRICT;
    EXECUTE 'ALTER TABLE vec_personal.relacion_alta_ejercicio ADD CONSTRAINT '
        ||'relacion_alta_ejercicio_relacion_ref_check '||siguiente;
    EXECUTE nuevo;
    SELECT to_jsonb(q)-'prosrc' INTO STRICT despues FROM pg_proc q WHERE oid=f;
    IF despues IS DISTINCT FROM antes
       OR pg_get_functiondef(f) IS DISTINCT FROM nuevo
       OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
          IS DISTINCT FROM 'cc96d9b84f7f47f22b85ad3876b8e4662c0fb0396338cc76e70960516e76a17d'
       OR (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid=tabla
          AND conname='relacion_alta_ejercicio_relacion_ref_check') IS DISTINCT FROM siguiente
       OR (SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_constraint c WHERE c.conrelid=tabla
          AND c.conname<>'relacion_alta_ejercicio_relacion_ref_check') IS DISTINCT FROM otras THEN
        RAISE EXCEPTION 'Personal 000004: postcondición no exacta' USING ERRCODE='55000';
    END IF;
END
$referencia$;
DROP TABLE vec_personal.personal004_check_esperado RESTRICT;
DO $auxiliar_retirado$
BEGIN
    IF to_regclass('vec_personal.personal004_check_esperado') IS NOT NULL
       OR to_regtype('vec_personal.personal004_check_esperado') IS NOT NULL THEN
        RAISE EXCEPTION 'Personal 000004: auxiliar residual' USING ERRCODE='55000';
    END IF;
END
$auxiliar_retirado$;
COMMIT;
