\set ON_ERROR_STOP on
-- AD3-51: reversión solo si no existe historia V3. No usar sobre historia
-- conservada: los locks bloquean consumos concurrentes hasta COMMIT/ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000051', 0));

-- El consumidor V3 toma checkpoint_gobierno antes de escribir atestación,
-- consumo y auditoría. El primer lock espera a cualquier efecto en curso;
-- los siguientes cierran también la vía de escrituras directas.
LOCK TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno,
           vec_autorizacion_atestada_v3.atestacion_decision_v3,
           vec_autorizacion_atestada_v3.consumo_decision_v3,
           vec_autorizacion_atestada_v3.auditoria_consumo_v3,
           vec_autorizacion_atestada_v3.control_cadena_auditoria
    IN ACCESS EXCLUSIVE MODE;

DO $historia$
DECLARE
    v_cadena record;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario' THEN
        RAISE EXCEPTION 'AD3-51: propietario incompatible' USING ERRCODE='55000';
    END IF;
    SELECT control_id, secuencia, cabeza_sha256 INTO v_cadena
      FROM vec_autorizacion_atestada_v3.control_cadena_auditoria;
    IF NOT FOUND OR v_cadena.control_id IS NOT TRUE
       OR v_cadena.secuencia IS DISTINCT FROM 0::numeric
       OR v_cadena.cabeza_sha256 IS DISTINCT FROM pg_catalog.repeat('0',64)
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.control_cadena_auditoria) <> 1
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
    THEN
        RAISE EXCEPTION 'AD3-51: DOWN rechazado por historia V3 o control incompleto'
            USING ERRCODE='55000';
    END IF;
END
$historia$;

DO $migracion$
DECLARE
    v_objetivo record;
    v_antes record;
    v_despues record;
    v_deps_antes jsonb;
    v_deps_despues jsonb;
    v_nueva text;
    v_membresia_antigua text := $fragmento$r.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'$fragmento$;
    v_membresia_nueva text := $fragmento$r.rolname IN (
                  'vec_contratacion_temporal_consultor_rrhh',
                  'vec_contratacion_temporal_consultor_rrhh_ambito'
              )$fragmento$;
    v_grupo_antiguo text := $fragmento$g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'$fragmento$;
    v_grupo_nuevo text := $fragmento$g.rolname IN (
                  'vec_contratacion_temporal_consultor_rrhh',
                  'vec_contratacion_temporal_consultor_rrhh_ambito'
              )$fragmento$;
    v_marca text := $fragmento$       OR pg_catalog.pg_has_role(
           session_user,
           'vec_autorizacion_atestada_v3_propietario', 'MEMBER')$fragmento$;
    v_guardia_nueva text := $fragmento$       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles g
            WHERE g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh_ambito'
              AND (
                  g.rolcanlogin OR g.rolsuper OR g.rolcreatedb
                  OR g.rolcreaterole OR g.rolreplication
                  OR g.rolbypassrls OR NOT g.rolinherit
              )
       )
$fragmento$;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh_ambito'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member = 'vec_contratacion_temporal_consultor_rrhh_ambito'::regrole)
    THEN
        RAISE EXCEPTION 'AD3-51: rol técnico incompatible' USING ERRCODE = '55000';
    END IF;

    FOR v_objetivo IN SELECT * FROM (VALUES
        ('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
         '6baef6127627ce9d6e6146c9d5425d7463f1a89b10411aac70fa70ed1944fd98',
         '1cec6ba3faa9d25607273638e458d76dd5f7e1eca0373754d1a2e3f28c6fa137', 2, false),
        ('vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
         '3530828669500274e9a46838c0d890aa0a974c2779e3fa38b5b2054c3919706d',
         '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff', 1, true)
    ) AS objetivos(firma, sha_previa, sha_nueva, apariciones_grupo, agrega_guardia)
    LOOP
        SELECT p.oid, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.prosrc AS cuerpo, p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor,
               pg_catalog.to_jsonb(p) - 'prosrc' AS metadatos
          INTO v_antes FROM pg_catalog.pg_proc p
         WHERE p.oid = pg_catalog.to_regprocedure(v_objetivo.firma);
        IF NOT FOUND
           OR v_antes.propietario <> 'vec_autorizacion_atestada_v3_propietario'::regrole
           OR v_antes.definidor IS NOT TRUE
           OR v_antes.configuracion IS DISTINCT FROM
              (CASE WHEN v_objetivo.agrega_guardia THEN
                  ARRAY['search_path=pg_catalog', 'lock_timeout=1s']
              ELSE ARRAY['search_path=pg_catalog', 'lock_timeout=2s'] END)
           OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_antes.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_objetivo.sha_previa
           OR (pg_catalog.length(v_antes.definicion) - pg_catalog.length(pg_catalog.replace(v_antes.definicion, v_membresia_nueva, '')))
              <> pg_catalog.length(v_membresia_nueva)
           OR (pg_catalog.length(v_antes.definicion) - pg_catalog.length(pg_catalog.replace(v_antes.definicion, v_grupo_nuevo, '')))
              <> v_objetivo.apariciones_grupo * pg_catalog.length(v_grupo_nuevo)
           OR pg_catalog.strpos(v_antes.definicion, v_membresia_antigua) <> 0
           OR pg_catalog.strpos(v_antes.definicion, v_grupo_antiguo) <> 0
           OR (v_objetivo.agrega_guardia AND
               (pg_catalog.length(v_antes.definicion) - pg_catalog.length(pg_catalog.replace(v_antes.definicion, v_guardia_nueva || v_marca, '')))
                <> pg_catalog.length(v_guardia_nueva || v_marca))
        THEN
            RAISE EXCEPTION 'AD3-51: preimagen V3 incompatible: %', v_objetivo.firma
                USING ERRCODE = '55000';
        END IF;
        SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
                   ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype), '[]'::jsonb)
          INTO v_deps_antes FROM pg_catalog.pg_depend d
         WHERE d.classid = 'pg_catalog.pg_proc'::regclass AND d.objid = v_antes.oid;

        v_nueva := pg_catalog.replace(v_antes.definicion, v_membresia_nueva, v_membresia_antigua);
        v_nueva := pg_catalog.replace(v_nueva, v_grupo_nuevo, v_grupo_antiguo);
        IF v_objetivo.agrega_guardia THEN
            v_nueva := pg_catalog.replace(v_nueva, v_guardia_nueva || v_marca, v_marca);
        END IF;
        EXECUTE v_nueva;
        SELECT p.oid, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.prosrc AS cuerpo, p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor,
               pg_catalog.to_jsonb(p) - 'prosrc' AS metadatos
          INTO STRICT v_despues FROM pg_catalog.pg_proc p WHERE p.oid = v_antes.oid;
        SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
                   ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype), '[]'::jsonb)
          INTO v_deps_despues FROM pg_catalog.pg_depend d
         WHERE d.classid = 'pg_catalog.pg_proc'::regclass AND d.objid = v_antes.oid;
        IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_despues.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_objetivo.sha_nueva
           OR v_despues.definicion IS DISTINCT FROM v_nueva
           OR v_despues.metadatos IS DISTINCT FROM v_antes.metadatos
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS DISTINCT FROM v_antes.definidor
           OR v_deps_despues IS DISTINCT FROM v_deps_antes
        THEN
            RAISE EXCEPTION 'AD3-51: postimagen o ACL V3 inesperada: %', v_objetivo.firma
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$migracion$;
COMMIT;
