-- ContextoActor V2: los vínculos de referencia se proyectan por vigencia actual.
-- Conserva el contrato de bytes y el bloqueo de todos los punteros antes del reloj.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:migracion:vinculos_efectivos_temporales:v1',0));
DO $preimagen$
DECLARE
  r oid; a oid; propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
  runtime oid := 'vec_contexto_actor_v1_runtime'::regrole;
  consumidor oid := pg_catalog.to_regrole('vec_autorizacion_propietario');
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
     OR current_setting('transaction_isolation') <> 'read committed' THEN
    RAISE EXCEPTION 'migracion ContextoActor 000006 requiere superusuario y transaccion ordinaria' USING ERRCODE='42501';
  END IF;
  r := pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)');
  a := pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)');
  IF r IS NULL OR a IS NULL OR consumidor IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)') IS NULL THEN
    RAISE EXCEPTION 'falta postimagen V2 historica de ContextoActor' USING ERRCODE='55000';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid IN (r,a)
      AND (p.proowner <> propietario OR NOT p.prosecdef
           OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
           OR EXISTS (
             SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
               pg_catalog.acldefault('f',p.proowner))) acl
             WHERE acl.privilege_type <> 'EXECUTE' OR acl.is_grantable
                OR acl.grantor <> propietario
                OR NOT (acl.grantee=propietario
                  OR (p.oid=r AND acl.grantee=runtime)
                  OR (p.oid=a AND acl.grantee=consumidor))
           ))
  ) OR (SELECT count(*) FROM pg_catalog.pg_proc p,
            LATERAL pg_catalog.aclexplode(p.proacl) acl
        WHERE p.oid=r AND acl.grantee=runtime
          AND acl.privilege_type='EXECUTE' AND acl.grantor=propietario
          AND NOT acl.is_grantable) <> 1
    OR (SELECT count(*) FROM pg_catalog.pg_proc p,
            LATERAL pg_catalog.aclexplode(p.proacl) acl
        WHERE p.oid=a AND acl.grantee=consumidor
          AND acl.privilege_type='EXECUTE' AND acl.grantor=propietario
          AND NOT acl.is_grantable) <> 1
    OR NOT pg_catalog.has_function_privilege(runtime,r,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege(consumidor,a,'EXECUTE')
    OR pg_catalog.has_function_privilege(runtime,a,'EXECUTE') THEN
    RAISE EXCEPTION 'preimagen owner/configuracion/ACL ContextoActor divergente' USING ERRCODE='55000';
  END IF;
  IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=r) <> '1581964f84ff00df1e2ebafe6084878cd770d9e94c10edf85cace9ec4552bc65'
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=a) <> '5fdb144d62dc301558caca8a243418308ac8eedc94f3c2c9d1b2d8230443fe46' THEN
    RAISE EXCEPTION 'preimagen de funciones ContextoActor divergente' USING ERRCODE='55000';
  END IF;
END
$preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    ahora timestamptz; coincidencias integer; cuenta record; perfil record;
    persona record; enlace record; enlaces_texto text; documento text;
    enlaces_procedencia_texto text; manifiesto_procedencia text;
    manifiesto_procedencia_bytes bytea; manifiesto_procedencia_huella text;
    canonica bytea; huella text; numero_enlaces integer; tipos integer; referencias integer;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'registro de contexto actor V2 requiere SERIALIZABLE de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'solicitud de contexto actor V2 invalida';
    END IF;

    -- El advisory estable serializa la identidad de operacion sin bloquear
    -- actores independientes. El adaptador repite SERIALIZABLE con el mismo
    -- oca_/rca_ ante un snapshot concurrente que termine en 23505/40001.
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));

    -- La entrada normal es idempotente por operacion y solicitud. El rca_ ya
    -- persistido se conserva; p_registro_contexto_ref solo se usa al crear.
    -- La funcion separada de reconciliacion si exige ese rca_ exacto porque
    -- coteja la respuesta observada antes de un COMMIT ambiguo.
    IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r WHERE r.operacion_ref = p_operacion_ref) THEN
        RETURN QUERY
        SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,
               r.huella_sha256,r.manifiesto_procedencia_canonico,
               r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
          FROM vec_contexto_actor_v1.registros_contexto r
         WHERE r.operacion_ref=p_operacion_ref AND r.cuenta_ref=p_cuenta_ref
           AND r.perfil_ref=p_perfil_ref AND r.metodo=p_metodo
           AND r.garantia=p_garantia AND r.solicitado_en=p_solicitado_en;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'colision de operacion de contexto actor V2';
        END IF;
        RETURN;
    END IF;

    -- Locks relevantes y deterministas: cuenta exacta, perfil exacto,
    -- candidatos vca_, personas derivadas y referencias de esas personas.
    -- Ninguna referencia se deriva de DNI, certificado, rol o claim libre.
    PERFORM 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
     WHERE ca.cuenta_ref=p_cuenta_ref ORDER BY ca.cuenta_ref FOR UPDATE OF ca;
    PERFORM 1 FROM vec_contexto_actor_v1.perfil_actual pa
     WHERE pa.perfil_ref=p_perfil_ref ORDER BY pa.perfil_ref FOR UPDATE OF pa;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ORDER BY va.vinculo_ref FOR UPDATE OF va;
    PERFORM 1 FROM vec_contexto_actor_v1.persona_actual pe
     WHERE pe.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY pe.persona_ref FOR UPDATE OF pe;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones rv USING (vinculo_ref,version)
     WHERE rv.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY ra.vinculo_ref FOR UPDATE OF ra;

    -- Este es el primer y unico reloj autoritativo. Toda lectura de negocio que
    -- sigue relee los punteros ya bloqueados y usa ventanas [desde,hasta).
    ahora := pg_catalog.clock_timestamp();
    IF ahora < p_solicitado_en OR ahora > p_solicitado_en + interval '5 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '57014', MESSAGE = 'ventana fresca de contexto actor V2 agotada';
    END IF;

    SELECT count(*) INTO coincidencias
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
        ON v.vinculo_ref=a.vinculo_ref AND v.version=a.version
     WHERE v.cuenta_ref=p_cuenta_ref AND v.perfil_ref=p_perfil_ref;
    IF coincidencias <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no resuelto';
    END IF;

    SELECT cv.version, cv.procedencia_ref,cv.procedencia_version,
           cv.procedencia_huella_sha256,cv.procedencia_autoridad,
           cv.estado, cv.vigente_desde, cv.vigente_hasta
      INTO STRICT cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING (cuenta_ref,version)
     WHERE ca.cuenta_ref=p_cuenta_ref;
    SELECT pv.version, pv.persona_ref,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT perfil
      FROM vec_contexto_actor_v1.perfil_actual pa
      JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
     WHERE pa.perfil_ref=p_perfil_ref;
    SELECT vv.vinculo_ref, vv.version, vv.persona_ref,
           vv.procedencia_ref,vv.procedencia_version,vv.procedencia_huella_sha256,
           vv.procedencia_autoridad,vv.estado, vv.vigente_desde, vv.vigente_hasta
      INTO STRICT enlace
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref;
    SELECT pv.version,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT persona
      FROM vec_contexto_actor_v1.persona_actual pa
      JOIN vec_contexto_actor_v1.persona_versiones pv USING (persona_ref,version)
     WHERE pa.persona_ref=perfil.persona_ref;

    IF perfil.persona_ref <> enlace.persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR enlace.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR perfil.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR persona.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR enlace.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR ahora < enlace.vigente_desde OR ahora >= enlace.vigente_hasta THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no vigente';
    END IF;

    SELECT count(*), count(DISTINCT vr.tipo), count(DISTINCT (vr.tipo,vr.referencia)),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             to_json(vr.vinculo_ref)::text, vr.version::text, to_json(vr.tipo)::text,
             to_json(vr.referencia)::text, to_json(vr.estado)::text,
             to_json(to_char(vr.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
             to_json(to_char(vr.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             to_json(vr.vinculo_ref)::text,vr.version::text,to_json(vr.tipo)::text,
             to_json(vr.referencia)::text,to_json(vr.procedencia_ref)::text,
             vr.procedencia_version::text,to_json(vr.procedencia_huella_sha256)::text,
             to_json(vr.procedencia_autoridad)::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref)
      INTO numero_enlaces, tipos, referencias, enlaces_texto,enlaces_procedencia_texto
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
     WHERE vr.persona_ref=perfil.persona_ref
       AND vr.estado='activo' AND ahora >= vr.vigente_desde
       AND ahora < vr.vigente_hasta;
    IF numero_enlaces > 128 OR tipos <> numero_enlaces OR referencias <> numero_enlaces
       OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
           JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
           WHERE vr.persona_ref=perfil.persona_ref
             AND vr.estado='activo' AND ahora >= vr.vigente_desde
             AND ahora < vr.vigente_hasta
             AND vr.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
    END IF;

    manifiesto_procedencia := format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      to_json(p_cuenta_ref)::text,cuenta.version::text,to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,to_json(cuenta.procedencia_huella_sha256)::text,to_json(cuenta.procedencia_autoridad)::text,
      to_json(perfil.persona_ref)::text,persona.version::text,to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,to_json(persona.procedencia_huella_sha256)::text,to_json(persona.procedencia_autoridad)::text,
      to_json(p_perfil_ref)::text,perfil.version::text,to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,to_json(perfil.procedencia_huella_sha256)::text,to_json(perfil.procedencia_autoridad)::text,
      to_json(enlace.vinculo_ref)::text,enlace.version::text,to_json(enlace.procedencia_ref)::text,
      enlace.procedencia_version::text,to_json(enlace.procedencia_huella_sha256)::text,to_json(enlace.procedencia_autoridad)::text,
      coalesce(enlaces_procedencia_texto,''));
    manifiesto_procedencia_bytes := convert_to(manifiesto_procedencia,'UTF8');
    manifiesto_procedencia_huella := encode(pg_catalog.sha256(manifiesto_procedencia_bytes),'hex');

    documento := format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      to_json(perfil.persona_ref)::text, to_json(p_metodo)::text, to_json(p_garantia)::text,
      to_json(p_perfil_ref)::text, to_json(perfil.persona_ref)::text,
      to_json(enlace.vinculo_ref)::text, enlace.version::text, to_json(p_cuenta_ref)::text, cuenta.version::text,
      persona.version::text, perfil.version::text, to_json(enlace.estado)::text,
      to_json(to_char(enlace.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(enlace.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      coalesce(enlaces_texto,''));
    canonica := convert_to(documento,'UTF8');
    IF octet_length(canonica) > 65536 THEN
        RAISE EXCEPTION USING ERRCODE = '22001', MESSAGE = 'snapshot de contexto actor V2 excede cota';
    END IF;
    huella := encode(pg_catalog.sha256(canonica),'hex');
    INSERT INTO vec_contexto_actor_v1.registros_contexto(
        operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
        solicitado_en,resuelto_en,representacion_canonica,huella_sha256,
        manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva
    ) VALUES (
        p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,
        p_solicitado_en,ahora,canonica,huella,
        manifiesto_procedencia_bytes,manifiesto_procedencia_huella,'autoridad_maestra_acreditada'
    );
    RETURN QUERY SELECT p_operacion_ref,p_registro_contexto_ref,canonica,huella,
      manifiesto_procedencia_bytes,manifiesto_procedencia_huella,
      'autoridad_maestra_acreditada'::text,ahora;
END
$f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    p_registro_contexto_ref text,
    p_contexto_actor_esquema text,
    p_contexto_actor_huella_sha256 text,
    p_manifiesto_procedencia_huella_sha256 text,
    p_autoridad_efectiva text,
    p_cuenta_ref text,
    p_cuenta_version numeric,
    p_persona_ref text,
    p_persona_version numeric,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_contexto_actor_ref text,
    p_contexto_actor_version numeric,
    p_metodo text,
    p_garantia text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz
) RETURNS timestamptz
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    registro record;
    cuenta record;
    perfil record;
    persona record;
    contexto record;
    generacion_observada numeric;
    ahora timestamptz;
    coincidencias integer;
    numero_vinculos integer;
    tipos integer;
    referencias integer;
    vinculos_texto text;
    vinculos_procedencia_texto text;
    representacion_texto text;
    representacion_reconstruida bytea;
    manifiesto_texto text;
    manifiesto_reconstruido bytea;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'acreditacion de uso ContextoActor V2 requiere SERIALIZABLE de escritura';
    END IF;

    IF vec_contexto_actor_v1.referencia_operacion_valida(
           p_registro_contexto_ref, 'rca_'
       ) IS NOT TRUE
       OR p_contexto_actor_esquema IS DISTINCT FROM
          'vec.contexto-actor.vinculado.v2'
       OR p_contexto_actor_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_manifiesto_procedencia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autoridad_efectiva IS DISTINCT FROM
          'autoridad_maestra_acreditada'
       OR vec_contexto_actor_v1.referencia_valida(
           p_cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_persona_ref, 'per_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_perfil_ref, 'prf_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_contexto_actor_ref, 'vca_'
       ) IS NOT TRUE
       OR p_cuenta_version IS NULL OR pg_catalog.scale(p_cuenta_version) <> 0
       OR p_cuenta_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_persona_version IS NULL OR pg_catalog.scale(p_persona_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_perfil_version IS NULL OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_contexto_actor_version IS NULL
       OR pg_catalog.scale(p_contexto_actor_version) <> 0
       OR p_contexto_actor_version NOT BETWEEN
          1 AND 18446744073709551615::numeric
       OR p_metodo NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad', 'demo'
       )
       OR p_garantia NOT IN ('bajo', 'sustancial', 'alto')
       OR vec_contexto_actor_v1.instante_valido(p_emitida_en) IS NOT TRUE
       OR vec_contexto_actor_v1.instante_valido(p_valida_hasta) IS NOT TRUE
       OR p_valida_hasta <= p_emitida_en THEN
        RETURN NULL;
    END IF;

    SELECT r.operacion_ref, r.registro_contexto_ref, r.cuenta_ref,
           r.perfil_ref, r.metodo, r.garantia, r.solicitado_en,
           r.resuelto_en, r.representacion_canonica, r.huella_sha256,
           r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,
           r.autoridad_efectiva
      INTO registro
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.registro_contexto_ref = p_registro_contexto_ref
     FOR SHARE OF r;
    IF NOT FOUND
       OR registro.cuenta_ref IS DISTINCT FROM p_cuenta_ref
       OR registro.perfil_ref IS DISTINCT FROM p_perfil_ref
       OR registro.metodo IS DISTINCT FROM p_metodo
       OR registro.garantia IS DISTINCT FROM p_garantia
       OR registro.huella_sha256 IS DISTINCT FROM
          p_contexto_actor_huella_sha256
       OR registro.manifiesto_procedencia_huella_sha256 IS DISTINCT FROM
          p_manifiesto_procedencia_huella_sha256
       OR registro.autoridad_efectiva IS DISTINCT FROM p_autoridad_efectiva
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.representacion_canonica), 'hex'
       ) IS DISTINCT FROM registro.huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.manifiesto_procedencia_canonico), 'hex'
       ) IS DISTINCT FROM registro.manifiesto_procedencia_huella_sha256
       OR registro.resuelto_en > p_emitida_en THEN
        RETURN NULL;
    END IF;

    -- El advisory global hace que todo mutador entre por su BEFORE STATEMENT
    -- antes de tocar filas. La seguridad frente a snapshots obsoletos no
    -- depende de este lock: la acredita la fila MVCC leida mas abajo.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0
        )
    );

    -- Orden identico al resolutor: cuenta, perfil, candidatos de contexto,
    -- personas y referencias de modulo. Se toma el reloj solo al final.
    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual AS a
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones AS v
        USING (cuenta_ref, version)
     WHERE a.cuenta_ref = p_cuenta_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    SELECT v.version, v.persona_ref, v.procedencia_ref,
           v.procedencia_version, v.procedencia_huella_sha256,
           v.procedencia_autoridad, v.estado, v.vigente_desde,
           v.vigente_hasta
      INTO perfil
      FROM vec_contexto_actor_v1.perfil_actual AS a
      JOIN vec_contexto_actor_v1.perfil_versiones AS v
        USING (perfil_ref, version)
     WHERE a.perfil_ref = p_perfil_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;
    GET DIAGNOSTICS coincidencias = ROW_COUNT;
    IF coincidencias <> 1 THEN RETURN NULL; END IF;

    SELECT v.vinculo_ref, v.version, v.cuenta_ref, v.perfil_ref,
           v.persona_ref, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO contexto
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref;

    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO persona
      FROM vec_contexto_actor_v1.persona_actual AS a
      JOIN vec_contexto_actor_v1.persona_versiones AS v
        USING (persona_ref, version)
     WHERE a.persona_ref = p_persona_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;

    -- Debe ocurrir despues de bloquear y releer todos los punteros. Si una
    -- mutacion comprometio despues del snapshot SERIALIZABLE, FOR SHARE no
    -- puede bloquear la version nueva invisible y PostgreSQL fuerza 40001.
    -- Si la acreditacion obtuvo primero el advisory, el mutador espera y queda
    -- serializado despues de su COMMIT.
    SELECT generacion
      INTO STRICT generacion_observada
      FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     WHERE control_id = true
     FOR SHARE;

    ahora := pg_catalog.clock_timestamp();

    IF cuenta.version IS DISTINCT FROM p_cuenta_version
       OR perfil.version IS DISTINCT FROM p_perfil_version
       OR perfil.persona_ref IS DISTINCT FROM p_persona_ref
       OR persona.version IS DISTINCT FROM p_persona_version
       OR contexto.vinculo_ref IS DISTINCT FROM p_contexto_actor_ref
       OR contexto.version IS DISTINCT FROM p_contexto_actor_version
       OR contexto.persona_ref IS DISTINCT FROM p_persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR contexto.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> p_autoridad_efectiva
       OR perfil.procedencia_autoridad <> p_autoridad_efectiva
       OR persona.procedencia_autoridad <> p_autoridad_efectiva
       OR contexto.procedencia_autoridad <> p_autoridad_efectiva
       OR p_emitida_en < cuenta.vigente_desde
       OR p_valida_hasta > cuenta.vigente_hasta
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR p_emitida_en < perfil.vigente_desde
       OR p_valida_hasta > perfil.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR p_emitida_en < persona.vigente_desde
       OR p_valida_hasta > persona.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR p_emitida_en < contexto.vigente_desde
       OR p_valida_hasta > contexto.vigente_hasta
       OR ahora < contexto.vigente_desde OR ahora >= contexto.vigente_hasta
       OR ahora < p_emitida_en OR ahora >= p_valida_hasta THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.count(*), pg_catalog.count(DISTINCT v.tipo),
           pg_catalog.count(DISTINCT (v.tipo, v.referencia)),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.estado)::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_desde AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_hasta AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.procedencia_ref)::text,
             v.procedencia_version::text,
             pg_catalog.to_json(v.procedencia_huella_sha256)::text,
             pg_catalog.to_json(v.procedencia_autoridad)::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref)
      INTO numero_vinculos, tipos, referencias, vinculos_texto,
           vinculos_procedencia_texto
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref
       AND v.estado = 'activo' AND ahora >= v.vigente_desde
       AND ahora < v.vigente_hasta;

    IF numero_vinculos > 128 OR tipos <> numero_vinculos
       OR referencias <> numero_vinculos
       OR EXISTS (
           SELECT 1
             FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
             JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
               USING (vinculo_ref, version)
            WHERE v.persona_ref = p_persona_ref
              AND v.estado = 'activo' AND ahora >= v.vigente_desde
              AND ahora < v.vigente_hasta
              AND (v.procedencia_autoridad <> p_autoridad_efectiva
                   OR p_emitida_en < v.vigente_desde
                   OR p_valida_hasta > v.vigente_hasta)
       ) THEN
        RETURN NULL;
    END IF;

    manifiesto_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      pg_catalog.to_json(p_cuenta_ref)::text, cuenta.version::text,
      pg_catalog.to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,
      pg_catalog.to_json(cuenta.procedencia_huella_sha256)::text,
      pg_catalog.to_json(cuenta.procedencia_autoridad)::text,
      pg_catalog.to_json(p_persona_ref)::text, persona.version::text,
      pg_catalog.to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,
      pg_catalog.to_json(persona.procedencia_huella_sha256)::text,
      pg_catalog.to_json(persona.procedencia_autoridad)::text,
      pg_catalog.to_json(p_perfil_ref)::text, perfil.version::text,
      pg_catalog.to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,
      pg_catalog.to_json(perfil.procedencia_huella_sha256)::text,
      pg_catalog.to_json(perfil.procedencia_autoridad)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(contexto.procedencia_ref)::text,
      contexto.procedencia_version::text,
      pg_catalog.to_json(contexto.procedencia_huella_sha256)::text,
      pg_catalog.to_json(contexto.procedencia_autoridad)::text,
      coalesce(vinculos_procedencia_texto, '')
    );
    manifiesto_reconstruido := pg_catalog.convert_to(
        manifiesto_texto, 'UTF8'
    );

    representacion_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(p_metodo)::text,
      pg_catalog.to_json(p_garantia)::text,
      pg_catalog.to_json(p_perfil_ref)::text,
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(p_cuenta_ref)::text,
      cuenta.version::text, persona.version::text, perfil.version::text,
      pg_catalog.to_json(contexto.estado)::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_desde AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_hasta AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          registro.resuelto_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      coalesce(vinculos_texto, '')
    );
    representacion_reconstruida := pg_catalog.convert_to(
        representacion_texto, 'UTF8'
    );

    IF representacion_reconstruida IS DISTINCT FROM
          registro.representacion_canonica
       OR manifiesto_reconstruido IS DISTINCT FROM
          registro.manifiesto_procedencia_canonico
       OR pg_catalog.encode(
           pg_catalog.sha256(representacion_reconstruida), 'hex'
       ) IS DISTINCT FROM p_contexto_actor_huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(manifiesto_reconstruido), 'hex'
       ) IS DISTINCT FROM p_manifiesto_procedencia_huella_sha256 THEN
        RETURN NULL;
    END IF;

    RETURN ahora;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN NULL;
END
$funcion$;

-- CREATE OR REPLACE conserva las concesiones nominales de consumidores existentes.
-- Ninguna sesión antigua obtiene autoridad de uso: acreditación reconstruye bytes.
COMMIT;
