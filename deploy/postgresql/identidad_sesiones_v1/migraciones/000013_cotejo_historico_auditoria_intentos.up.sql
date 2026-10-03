-- Cotejo del vínculo autenticación/sesión original para la auditoría común.
-- Sólo el propietario de AD3 puede invocarlo dentro de su transacción.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:cotejo_auditoria_intentos:v1',0));
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL timezone='UTC';

DO $pre$
BEGIN
 IF current_user <> 'vec_identidad_sesiones_v1_propietario'
    OR to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'precondiciones IS13 no satisfechas' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
 p_vinculo_canonico bytea
) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET row_security=on
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE
 doc jsonb; p_autenticacion_ref text; p_sesion_ref text; p_autenticacion_sha256 text;
 sesion record; estado record; instante timestamptz;
 cuentas integer:=0; esperadas integer;
 inicio timestamptz:=clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' THEN
   RAISE EXCEPTION 'cotejo historico requiere SERIALIZABLE READ WRITE UTC' USING ERRCODE='25000';
 END IF;
 IF p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384 THEN
   RAISE EXCEPTION 'vinculo historico invalido' USING ERRCODE='22023';
 END IF;
 doc:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 IF jsonb_typeof(doc) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(doc)) <> 31
    OR NOT (doc ?& ARRAY[
      'esquema','bloque_version','autenticacion_ref','autenticacion_huella_sha256',
      'asercion_ref','sesion_ref','control_sesion_ref','control_sesion_revision',
      'control_sesion_huella_sha256','cuenta_ref','cuenta_ordinaria_ref',
      'principal_id','perfil_activo_ref','cuenta_privilegiada','superficie',
      'metodo_observado','garantia_observada','politica_garantia_ref',
      'politica_garantia_huella_sha256','autenticacion_verificada_en',
      'sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en',
      'registro_contexto_ref','contexto_actor_esquema','contexto_actor_ref',
      'contexto_actor_version','contexto_actor_cuenta_version',
      'contexto_actor_huella_sha256','manifiesto_procedencia_huella_sha256',
      'autoridad_efectiva'])
    OR doc->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
    OR doc->'bloque_version' IS DISTINCT FROM '2'::jsonb THEN
   RAISE EXCEPTION 'esquema de vinculo historico invalido' USING ERRCODE='22023';
 END IF;
 p_autenticacion_ref:=doc->>'autenticacion_ref';
 p_sesion_ref:=doc->>'sesion_ref';
 p_autenticacion_sha256:=doc->>'autenticacion_huella_sha256';
    IF vec_identidad_sesiones_v1.referencia_valida(p_autenticacion_ref,'aut_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_sesion_ref,'ses_') IS NOT TRUE
       OR p_autenticacion_sha256 IS NULL OR p_autenticacion_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_sha256=repeat('0',64) THEN
        RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended(
      'vec_identidad_sesiones_v1:migracion:cotejo_auditoria_intentos:v1',0));
    -- Selección explícita: nunca leer ni devolver columnas HMAC de consumo.
    -- La revisión es la fijada por el consumo original; no punteros actuales.
    SELECT base.autenticacion_ref,
           base.autenticacion_huella_sha256,
           base.asercion_ref,
           base.sesion_ref,
           control.control_sesion_ref,
           control.revision AS control_sesion_revision,
           control.estado AS control_sesion_estado,
           control.huella_sha256 AS control_sesion_huella_sha256,
           base.cuenta_ref,
           base.cuenta_ordinaria_ref,
           base.cuenta_privilegiada,
           base.superficie,
           base.metodo_observado,
           base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en,
           base.sesion_emitida_en,
           control.sesion_valida_hasta,
           control.sesion_revalidada_en,
           consumo.cuenta_revision,
           consumo.cuenta_ordinaria_revision,
           consumo.operacion_ref, consumo.consumida_en
      INTO STRICT sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = base.sesion_ref
       AND consumo.autenticacion_ref = base.autenticacion_ref
       AND consumo.autenticacion_huella_sha256 =
           base.autenticacion_huella_sha256
       AND consumo.asercion_ref = base.asercion_ref
       AND consumo.cuenta_ref = base.cuenta_ref
       AND consumo.cuenta_ordinaria_ref = base.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = consumo.sesion_ref
       AND control.control_sesion_ref = consumo.control_sesion_ref
       AND control.revision = consumo.control_sesion_revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.sesion_ref = p_sesion_ref
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
       AND consumo.autenticacion_huella_sha256 = p_autenticacion_sha256;


    IF sesion.control_sesion_estado <> 'activa'
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.asercion_ref, 'ase_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.control_sesion_ref, 'cse_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ordinaria_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.politica_garantia_ref, 'pga_'
       ) IS NOT TRUE
       OR sesion.control_sesion_revision NOT BETWEEN
           1 AND 18446744073709551615
       OR sesion.autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.autenticacion_huella_sha256 = repeat('0', 64)
       OR sesion.control_sesion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.control_sesion_huella_sha256 = repeat('0', 64)
       OR sesion.politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.politica_garantia_huella_sha256 = repeat('0', 64)
       OR sesion.superficie NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR sesion.metodo_observado NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR sesion.garantia_observada NOT IN (
           'bajo', 'sustancial', 'alto'
       )
       OR (sesion.superficie = 'externa_personal'
           AND sesion.garantia_observada = 'bajo')
       OR (sesion.superficie IN (
               'interna_corporativa', 'administracion_privilegiada'
           ) AND sesion.garantia_observada <> 'alto')
       OR sesion.autenticacion_verificada_en > sesion.sesion_emitida_en
       OR sesion.sesion_revalidada_en < sesion.autenticacion_verificada_en
       OR sesion.sesion_revalidada_en < sesion.sesion_emitida_en
       OR sesion.sesion_valida_hasta <= sesion.sesion_revalidada_en
       OR (sesion.cuenta_privilegiada AND (
           sesion.superficie <> 'administracion_privilegiada'
           OR sesion.cuenta_ref = sesion.cuenta_ordinaria_ref
       ))
       OR (NOT sesion.cuenta_privilegiada AND (
           sesion.superficie = 'administracion_privilegiada'
           OR sesion.cuenta_ref <> sesion.cuenta_ordinaria_ref
       )) THEN
        RAISE EXCEPTION 'autenticacion historica incoherente' USING ERRCODE='22023';
    END IF;


    IF sesion.autenticacion_ref IS DISTINCT FROM p_autenticacion_ref
       OR sesion.sesion_ref IS DISTINCT FROM p_sesion_ref
       OR sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_autenticacion_sha256
       OR sesion.consumida_en IS DISTINCT FROM sesion.sesion_revalidada_en
       OR sesion.control_sesion_huella_sha256 IS DISTINCT FROM
         vec_identidad_sesiones_v1.huella_control_sesion_v1(
           sesion.control_sesion_ref,sesion.control_sesion_revision,sesion.sesion_ref,
           sesion.control_sesion_estado,sesion.sesion_revalidada_en,sesion.sesion_valida_hasta,
           sesion.operacion_ref)
       OR sesion.sesion_valida_hasta > sesion.sesion_emitida_en + interval '5 minutes'
       OR sesion.sesion_revalidada_en >= sesion.autenticacion_verificada_en + (CASE sesion.superficie
           WHEN 'externa_personal' THEN interval '12 hours'
           WHEN 'interna_corporativa' THEN interval '15 minutes'
           WHEN 'administracion_privilegiada' THEN interval '5 minutes'
           ELSE interval '0 seconds' END) THEN
        RAISE EXCEPTION 'origen historico divergente' USING ERRCODE='22023';
    END IF;
    FOREACH instante IN ARRAY ARRAY[sesion.autenticacion_verificada_en,sesion.sesion_emitida_en,
       sesion.sesion_valida_hasta,sesion.sesion_revalidada_en,sesion.consumida_en] LOOP
        IF instante IS NULL OR NOT isfinite(instante)
           OR instante < timestamptz '0001-01-01 00:00:00+00'
           OR instante >= timestamptz '10000-01-01 00:00:00+00' THEN
            RAISE EXCEPTION 'fecha historica invalida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    esperadas:=CASE WHEN sesion.cuenta_privilegiada THEN 2 ELSE 1 END;
    FOR estado IN
      SELECT e.cuenta_ref,e.revision,e.estado,e.registrada_en
      FROM vec_identidad_sesiones_v1.estado_cuenta e
      WHERE (e.cuenta_ref=sesion.cuenta_ref AND e.revision=sesion.cuenta_revision)
         OR (e.cuenta_ref=sesion.cuenta_ordinaria_ref AND e.revision=sesion.cuenta_ordinaria_revision)
    LOOP
        cuentas:=cuentas+1;
        IF estado.estado IS DISTINCT FROM 'activa'
           OR NOT isfinite(estado.registrada_en)
           OR estado.registrada_en > sesion.consumida_en
           OR (estado.cuenta_ref=sesion.cuenta_ref AND estado.revision<>sesion.cuenta_revision)
           OR (estado.cuenta_ref=sesion.cuenta_ordinaria_ref AND estado.revision<>sesion.cuenta_ordinaria_revision) THEN
            RAISE EXCEPTION 'cuenta historica divergente' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF cuentas<>esperadas THEN
        RAISE EXCEPTION 'cuenta historica ausente' USING ERRCODE='P0002';
    END IF;
    -- El reloj actual sólo limita la ejecución. No exige permiso actual.
    IF clock_timestamp()>inicio+interval '5 seconds' THEN
        RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
    END IF;
    IF jsonb_typeof(doc->'cuenta_privilegiada') IS DISTINCT FROM 'boolean'
       OR jsonb_typeof(doc->'control_sesion_revision') IS DISTINCT FROM 'number'
       OR doc->>'asercion_ref' IS DISTINCT FROM sesion.asercion_ref
       OR doc->>'control_sesion_ref' IS DISTINCT FROM sesion.control_sesion_ref
       OR (doc->>'control_sesion_revision')::numeric IS DISTINCT FROM sesion.control_sesion_revision
       OR doc->>'control_sesion_huella_sha256' IS DISTINCT FROM sesion.control_sesion_huella_sha256
       OR doc->>'cuenta_ref' IS DISTINCT FROM sesion.cuenta_ref
       OR doc->>'cuenta_ordinaria_ref' IS DISTINCT FROM sesion.cuenta_ordinaria_ref
       OR (doc->>'cuenta_privilegiada')::boolean IS DISTINCT FROM sesion.cuenta_privilegiada
       OR doc->>'superficie' IS DISTINCT FROM sesion.superficie
       OR doc->>'metodo_observado' IS DISTINCT FROM sesion.metodo_observado
       OR doc->>'garantia_observada' IS DISTINCT FROM sesion.garantia_observada
       OR doc->>'politica_garantia_ref' IS DISTINCT FROM sesion.politica_garantia_ref
       OR doc->>'politica_garantia_huella_sha256' IS DISTINCT FROM sesion.politica_garantia_huella_sha256
       OR (doc->>'autenticacion_verificada_en')::timestamptz IS DISTINCT FROM sesion.autenticacion_verificada_en
       OR (doc->>'sesion_emitida_en')::timestamptz IS DISTINCT FROM sesion.sesion_emitida_en
       OR (doc->>'sesion_valida_hasta')::timestamptz IS DISTINCT FROM sesion.sesion_valida_hasta
       OR (doc->>'sesion_revalidada_en')::timestamptz IS DISTINCT FROM sesion.sesion_revalidada_en THEN
       RAISE EXCEPTION 'vinculo historico divergente' USING ERRCODE='22023';
    END IF;
    RETURN true;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'autenticacion historica no disponible' USING ERRCODE='P0002';
    WHEN data_exception THEN
        RAISE EXCEPTION 'autenticacion historica invalida' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)
 TO vec_autorizacion_atestada_v3_propietario;
DO $acl$
DECLARE f oid:='vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)'::regprocedure;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
 r oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'ACL IS13 inesperada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
