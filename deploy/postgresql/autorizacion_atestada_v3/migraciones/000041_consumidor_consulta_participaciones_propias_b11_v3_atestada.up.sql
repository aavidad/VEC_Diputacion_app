\set ON_ERROR_STOP on
-- B11: consumidor y revalidación V3 de la lectura propia de Bolsa.
-- Preimagen: consumidores AD3 de consultas ya instalados; el grupo exterior
-- B11 se crea por roles_b11_up.sql. No concede CT ni reutiliza su audiencia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000041', 0));

DO $precondicion$
DECLARE
    v_propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    v_bolsa oid := 'vec_bolsa_llamamientos_propietario'::regrole;
BEGIN
    IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
       OR pg_catalog.to_regrole('vec_bolsa_llamamientos_consultor_participaciones_propias') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE oid=v_bolsa
                      AND NOT rolcanlogin AND NOT rolsuper AND NOT rolinherit
                      AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication
                      AND NOT rolbypassrls)
       OR NOT pg_catalog.has_schema_privilege(v_bolsa, 'vec_autorizacion_atestada_v3', 'USAGE')
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'AD3-000041: preimagen B11 incompatible' USING ERRCODE='55000';
    END IF;
END $precondicion$;

-- Se clonan los dos núcleos de consulta vigentes, sustituyendo solo el perfil,
-- audiencia y frontera nominal. Así se conservan íntegros la verificación COSE,
-- HMAC, revocación, auditoría, idempotencia y revalidación viva de AD3.
DO $clonar_nucleos$
DECLARE
    v_consumo text;
    v_revalidacion text;
    v_hash_consumo text;
    v_hash_revalidacion text;
    v_consumo_origen oid := 'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_revalidacion_origen oid := 'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(p.oid),pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
      INTO STRICT v_consumo,v_hash_consumo FROM pg_catalog.pg_proc p WHERE p.oid=v_consumo_origen
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
    SELECT pg_catalog.pg_get_functiondef(p.oid),pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
      INTO STRICT v_revalidacion,v_hash_revalidacion FROM pg_catalog.pg_proc p WHERE p.oid=v_revalidacion_origen
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s'];
    IF v_hash_consumo IS DISTINCT FROM '1cec6ba3faa9d25607273638e458d76dd5f7e1eca0373754d1a2e3f28c6fa137'
       OR v_hash_revalidacion IS DISTINCT FROM '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff' THEN
        RAISE EXCEPTION 'AD3-000041: núcleo posterior a AD3-000021 requerido' USING ERRCODE='55000';
    END IF;
    IF pg_catalog.strpos(v_consumo, 'p_perfil_consulta = ''cuadro''') = 0
       OR pg_catalog.strpos(v_revalidacion, 'p_perfil_consulta = ''cuadro''') = 0
       OR pg_catalog.strpos(v_consumo, 'vec_contratacion_temporal_consultor_rrhh') = 0
       OR pg_catalog.strpos(v_revalidacion, 'vec_contratacion_temporal_consultor_rrhh') = 0 THEN
        RAISE EXCEPTION 'AD3-000041: núcleo de consulta no reconocible' USING ERRCODE='55000';
    END IF;
    v_consumo := pg_catalog.replace(v_consumo,
        'consumir_consulta_rrhh_v3_interna',
        'consumir_b11_participaciones_v3_interna');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'revalidar_consumo_consulta_rrhh_v3_interna',
        'revalidar_b11_participaciones_v3_interna');
    v_consumo := pg_catalog.replace(v_consumo,
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_bolsa_llamamientos_consultor_participaciones_propias');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'vec_contratacion_temporal_consultor_rrhh',
        'vec_bolsa_llamamientos_consultor_participaciones_propias');
    v_consumo := pg_catalog.replace(v_consumo,
        'OR pg_catalog.pg_has_role(\n           session_user, ''vec_contratacion_temporal_propietario'', ''MEMBER'')',
        'OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_propietario'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_migrador'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_ejecutor'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(\n           session_user, ''vec_contratacion_temporal_propietario'', ''MEMBER'')');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'OR pg_catalog.pg_has_role(\n           session_user, ''vec_contratacion_temporal_propietario'', ''MEMBER'')',
        'OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_propietario'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_migrador'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(session_user, ''vec_bolsa_llamamientos_ejecutor'', ''MEMBER'')\n       OR pg_catalog.pg_has_role(\n           session_user, ''vec_contratacion_temporal_propietario'', ''MEMBER'')');
    v_consumo := pg_catalog.replace(v_consumo,
        'p_perfil_consulta = ''cuadro''',
        'p_perfil_consulta = ''b11_participaciones_propias''');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'p_perfil_consulta = ''cuadro''',
        'p_perfil_consulta = ''b11_participaciones_propias''');
    v_consumo := pg_catalog.replace(v_consumo,
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1', 'vec.bolsa.mi-bolsa.v1');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1', 'vec.bolsa.mi-bolsa.v1');
    v_consumo := pg_catalog.replace(v_consumo,
        'contratacion_temporal.cuadro.consultar', 'bolsa.candidato.participaciones.consultar');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'contratacion_temporal.cuadro.consultar', 'bolsa.candidato.participaciones.consultar');
    v_consumo := pg_catalog.replace(v_consumo, '''contratacion_temporal''', '''bolsa''');
    v_revalidacion := pg_catalog.replace(v_revalidacion, '''contratacion_temporal''', '''bolsa''');
    v_consumo := pg_catalog.replace(v_consumo,
        'cuadro_rrhh_contratacion_temporal', 'participaciones_propias_candidato');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'cuadro_rrhh_contratacion_temporal', 'participaciones_propias_candidato');
    v_consumo := pg_catalog.replace(v_consumo,
        'gestion_operativa_contratacion_temporal', 'consulta_posicion_propia_bolsa');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'gestion_operativa_contratacion_temporal', 'consulta_posicion_propia_bolsa');
    v_consumo := pg_catalog.replace(v_consumo,
        'p_perfil_consulta NOT IN (''cuadro'', ''detalle'')',
        'p_perfil_consulta IS DISTINCT FROM ''b11_participaciones_propias''');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'p_perfil_consulta NOT IN (''cuadro'', ''detalle'')',
        'p_perfil_consulta IS DISTINCT FROM ''b11_participaciones_propias''');
    v_revalidacion := pg_catalog.replace(v_revalidacion,
        'IF p_perfil_consulta = ''b11_participaciones_propias'' THEN',
        'IF p_perfil_consulta = ''b11_participaciones_propias'' THEN');
    IF pg_catalog.strpos(v_consumo, 'vec.bolsa.mi-bolsa.v1') = 0
       OR pg_catalog.strpos(v_revalidacion, 'vec.bolsa.mi-bolsa.v1') = 0
       OR pg_catalog.strpos(v_consumo, 'bolsa.candidato.participaciones.consultar') = 0
       OR pg_catalog.strpos(v_revalidacion, 'bolsa.candidato.participaciones.consultar') = 0 THEN
        RAISE EXCEPTION 'AD3-000041: sustitución B11 incompleta' USING ERRCODE='55000';
    END IF;
    EXECUTE v_consumo;
    EXECUTE v_revalidacion;
    IF pg_catalog.pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_consumo
       OR pg_catalog.pg_get_functiondef('vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_revalidacion THEN
        RAISE EXCEPTION 'AD3-000041: postimagen de núcleos B11 divergente' USING ERRCODE='55000';
    END IF;
END $clonar_nucleos$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,auditoria_huella_sha256 text,
    consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $funcion$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(
   'b11_participaciones_propias',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,consumo_huella_sha256 text,revalidada_en timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='1s'
AS $funcion$
 SELECT * FROM vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(
   'b11_participaciones_propias',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$funcion$;

DO $audiencia$
DECLARE v_def text; v_nueva text;
BEGIN
 LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT v_def
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
 IF pg_catalog.strpos(v_def,'''vec.bolsa.mi-bolsa.v1''')<>0 THEN
   RAISE EXCEPTION 'AD3-000041: gobierno de audiencia incompatible' USING ERRCODE='55000';
 END IF;
 IF v_def ~ '^CHECK \(audiencia_consumo IN \(.+\)\)$' THEN
   v_nueva := pg_catalog.left(v_def,pg_catalog.length(v_def)-2)||', ''vec.bolsa.mi-bolsa.v1''))';
 ELSIF v_def ~ '^CHECK \(audiencia_consumo = ANY \(ARRAY\[.+\]\)\)$' THEN
   v_nueva := pg_catalog.left(v_def,pg_catalog.length(v_def)-3)||', ''vec.bolsa.mi-bolsa.v1''::text]))';
 ELSE
   RAISE EXCEPTION 'AD3-000041: gobierno de audiencia incompatible' USING ERRCODE='55000';
 END IF;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
      || v_nueva;
END $audiencia$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $revocar_acl_ajenas$
DECLARE f oid; a record;
BEGIN
 FOR f IN SELECT unnest(ARRAY[
   'vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure]) LOOP
   FOR a IN SELECT DISTINCT x.grantee FROM pg_catalog.pg_proc p,LATERAL pg_catalog.aclexplode(p.proacl) x
             WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
     EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %I',f::regprocedure,pg_catalog.pg_get_userbyid(a.grantee));
   END LOOP;
 END LOOP;
END $revocar_acl_ajenas$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
COMMIT;
