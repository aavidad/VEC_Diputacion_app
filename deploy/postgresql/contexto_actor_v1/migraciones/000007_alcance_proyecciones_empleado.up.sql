-- ContextoActor V2: alcance cerrado de proyecciones pedidas por la composición.
--
-- Parte de la postimagen exacta de 000006 (vínculos efectivos temporales) y
-- conecta, solo cuando se pide, la proyección gobernada persona→empleado de
-- Personal 000016 (vec_personal.resolver_empleado_canonico_persona_v1), por
-- contrato y sin leer tablas de Personal.
--
-- * Alcance vacío ('{}'): el contexto, el canon y el manifiesto son idénticos
--   byte a byte a los de 000006 (representación heredada, p. ej. CT).
-- * Alcance {empleado}: el empleado procede exclusivamente de Personal. Los
--   punteros de empleado del núcleo no entran en el contexto; sin_empleado y
--   ambiguo deniegan con motivo propio (SQLSTATE PCA01 y PCA02).
-- * El alcance queda ligado al registro durable sin claves nuevas: la entrada
--   de Personal aparece en "vinculos" con su propia identidad pep_ (versión y
--   vigencia de Personal) y en el manifiesto con su procedencia prc_/versión/
--   huella. Los consumidores SQL existentes (Dietas, Personal 000008) exigen
--   exactamente las claves V2 actuales, por eso no se añade ninguna. El alcance
--   de un registro se deriva de su canon almacenado; idempotencia,
--   reconciliación y acreditar_uso lo reconstruyen exactamente.
-- * acreditar_uso vuelve a consultar Personal en su instante autoritativo:
--   revocación, baja, caducidad o cambio de versión anulan el uso posterior.
-- * Instantánea obsoleta: resolutor y acreditación son SERIALIZABLE y la
--   historia de Personal es de solo adición, así que esperar al publicador en
--   el consultivo no basta (se leería el estado anterior a una revocación ya
--   confirmada). Ambos llaman antes del reloj a la barrera de Personal
--   vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1, que toma
--   con FOR SHARE la generación de la persona: una publicación confirmada
--   después de la instantánea devuelve 40001 y el llamante reintenta.
--
-- Subdecisión (consenso 25/09/2026): se cierran las altas nuevas de punteros
-- vinculo_referencia de tipo empleado en el núcleo. Los existentes se conservan
-- (no se borran ni se revocan aquí) y solo admiten versiones nuevas que los
-- revoquen o recorten su vigencia; un candidato no pasa a empleado y un
-- empleado revocado no se reactiva. Los de candidato no cambian. Inventario de altas en el repositorio a esta fecha:
--   - deploy/principal/preparar_dietas_desarrollo.py (employee_link: la vía del
--     incidente del 23/09; debe publicar en Personal en su lugar);
--   - fixtures sintéticos de prueba que crean emp_ antes de esta migración:
--     contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql,
--     contexto_actor_v1/probar_contexto_actor_v1_base_pg18_4.sh,
--     autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql y
--     contratacion_temporal/pruebas_sql/fixture_contexto_actor_b_o3.sql.
-- Los punteros existentes en cada base se enumeran con NOTICE al aplicar.
--
-- Orden obligatorio de instalación: 000001, 000002, 000003, 000004, 000005,
-- 000006 y, con Personal 000016 ya instalada, 000007. 000004 (vínculo
-- corporativo RRHH) no puede aplicarse después de 000007, así que esta
-- preimagen exige sus objetos (historia, puntero actual con sus
-- disparadores de generación y la unicidad de actor que añade) y rechaza la
-- instalación si falta.
--
-- Deuda para una 000008: el lector histórico leer_contexto_original_v2
-- (000005, línea ~206) exige vinculo_ref 'vin_' en cada vínculo del canon y
-- no reconstruye la entrada 'pep_' de Personal. Los registros resueltos con
-- alcance {empleado} fallan cerrado en esa lectura (22023) en lugar de
-- reconstruirse; los de alcance vacío no cambian.
--
-- DOWN prohibido: las funciones sustituidas firman recibos con historia.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:migracion:alcance_proyecciones:v1',0));
DO $preimagen$
DECLARE
  r oid; c oid; a oid; e oid; pe oid; pb oid;
  propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
  runtime oid := 'vec_contexto_actor_v1_runtime'::regrole;
  personal oid := pg_catalog.to_regrole('vec_personal_propietario');
  consumidor oid := pg_catalog.to_regrole('vec_autorizacion_propietario');
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
     OR current_setting('transaction_isolation') <> 'read committed' THEN
    RAISE EXCEPTION 'migracion ContextoActor 000007 requiere superusuario y transaccion ordinaria' USING ERRCODE='42501';
  END IF;
  r := pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)');
  c := pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)');
  a := pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)');
  e := pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()');
  pe := pg_catalog.to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)');
  pb := pg_catalog.to_regprocedure('vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)');
  IF r IS NULL OR c IS NULL OR a IS NULL OR e IS NULL OR consumidor IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.alcance_registro_contexto_v2(bytea)') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz)') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.rechazar_alta_puntero_empleado_v2()') IS NOT NULL THEN
    RAISE EXCEPTION 'falta postimagen ContextoActor 000006 o 000007 ya aplicada' USING ERRCODE='55000';
  END IF;
  -- 000004 instalada: no puede aplicarse después de esta migración.
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                  WHERE oid = pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_corporativo_versiones')
                    AND relkind = 'r' AND relowner = propietario AND relforcerowsecurity)
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                  WHERE oid = pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual')
                    AND relkind = 'r' AND relowner = propietario AND relforcerowsecurity)
     OR (SELECT count(*) FROM pg_catalog.pg_trigger t
          WHERE t.tgrelid = pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual')
            AND NOT t.tgisinternal
            AND t.tgname IN ('serializar_mutacion_punteros_actuales_v2',
                             'avanzar_generacion_punteros_actuales_v2',
                             'puntero_actual_no_truncable_v2')) <> 3
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
                  WHERE conrelid = 'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass
                    AND conname = 'vinculo_contexto_versiones_actor_uq' AND contype = 'u') THEN
    RAISE EXCEPTION 'ContextoActor 000007 exige 000004 instalada antes' USING ERRCODE='55000';
  END IF;
  -- Postimagen exacta de 000006 (y de las funciones heredadas que se sustituyen).
  IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=r) <> '8f26afb738716bd2f6d4354611de99a1a79287da355ceb6112aef16bc623478d'
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=a) <> 'bf9e116484deefd91072e10fbdf5dd42b016c547a805642dd3ed63462a16e0cd'
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=c) <> '8967ace0d19751692dcdaf2a8a59111afe43ae7f1c5b9d6db087cf4fa6b56b52'
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=e) <> '3ccccca12b113c42167d152e8c68da3e81f4806d52135ca7ba4822ca77fbd465' THEN
    RAISE EXCEPTION 'preimagen de funciones ContextoActor 000006 divergente' USING ERRCODE='55000';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid IN (r,c,a,e)
      AND (p.proowner <> propietario OR NOT p.prosecdef
           OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
           OR EXISTS (
             SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
               pg_catalog.acldefault('f',p.proowner))) acl
             WHERE acl.privilege_type <> 'EXECUTE' OR acl.is_grantable
                OR acl.grantor <> propietario
                OR NOT (acl.grantee=propietario
                  OR (p.oid IN (r,c) AND acl.grantee=runtime)
                  OR (p.oid=a AND acl.grantee=consumidor))
           ))
  ) OR NOT pg_catalog.has_function_privilege(runtime,r,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege(runtime,c,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege(consumidor,a,'EXECUTE')
    OR pg_catalog.has_function_privilege(runtime,a,'EXECUTE') THEN
    RAISE EXCEPTION 'preimagen owner/configuracion/ACL ContextoActor divergente' USING ERRCODE='55000';
  END IF;
  -- Contrato de Personal 000016: función exacta, propia de Personal, sin
  -- consumidores concedidos todavía.
  IF personal IS NULL OR pe IS NULL OR pb IS NULL
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
           FROM pg_catalog.pg_proc WHERE oid=pe) <> 'ee33f00aee1cfdacfad434125265f8c407c9679cd6e11da9a8dd31355db2d52a'
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
           FROM pg_catalog.pg_proc WHERE oid=pb) <> '1b2438dceb621d1095db73871fe64e9aa1ced15f8bb1cefe0cb58fdbdd41c739'
     OR EXISTS (
       SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid IN (pe,pb)
         AND (p.proowner <> personal OR NOT p.prosecdef
              OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on']::text[]
              OR EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
                  pg_catalog.acldefault('f',p.proowner))) acl
                WHERE acl.grantee <> personal)))
     OR pg_catalog.has_function_privilege(propietario,pe,'EXECUTE')
     OR pg_catalog.has_function_privilege(propietario,pb,'EXECUTE') THEN
    RAISE EXCEPTION 'contrato Personal 000016 ausente o divergente' USING ERRCODE='55000';
  END IF;
END
$preimagen$;

-- Concesión nominal mínima de Personal: solo el propietario de ContextoActor,
-- que ejecuta el resolutor y la acreditación como SECURITY DEFINER, puede
-- invocar la lectura gobernada y su barrera. Ni runtime ni PUBLIC las reciben.
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
REVOKE ALL ON FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_personal TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)
  TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)
  TO vec_contexto_actor_v1_propietario;
RESET ROLE;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- Alcance de un registro ya firmado: se deriva de su canon almacenado. Una
-- entrada pep_ en "vinculos" solo la emite la proyección de Personal pedida.
CREATE FUNCTION vec_contexto_actor_v1.alcance_registro_contexto_v2(p_representacion bytea)
RETURNS text[] LANGUAGE sql IMMUTABLE STRICT SET search_path = pg_catalog AS $f$
  SELECT CASE WHEN EXISTS (
    SELECT 1 FROM pg_catalog.jsonb_array_elements(
      pg_catalog.convert_from(p_representacion,'UTF8')::jsonb -> 'vinculos') AS v(e)
     WHERE v.e ->> 'tipo' = 'empleado' AND pg_catalog.starts_with(v.e ->> 'vinculo_ref','pep_')
  ) THEN ARRAY['empleado']::text[] ELSE '{}'::text[] END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.alcance_registro_contexto_v2(bytea) FROM PUBLIC;

-- Traducción cerrada del contrato de Personal. Devuelve siempre una fila con
-- la clase y, solo para 'empleado', los campos ya validados. No elige nunca.
CREATE FUNCTION vec_contexto_actor_v1.proyeccion_empleado_personal_v2(
    p_persona_ref text, p_instante timestamptz
) RETURNS TABLE (
    resultado text, proyeccion_ref text, version numeric, empleado_ref text,
    vigente_desde timestamptz, vigente_hasta timestamptz, procedencia_ref text,
    procedencia_version numeric, procedencia_huella_sha256 text
) LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog AS $f$
DECLARE f record; n integer := 0;
BEGIN
    FOR f IN SELECT * FROM vec_personal.resolver_empleado_canonico_persona_v1(p_persona_ref,p_instante) LOOP
        n := n + 1;
        IF n > 1 THEN
            RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'contrato Personal persona-empleado divergente';
        END IF;
        IF f.resultado = 'empleado' THEN
            IF f.persona_ref IS DISTINCT FROM p_persona_ref OR f.efectivas <> 1
               OR f.estado IS DISTINCT FROM 'activa'
               OR f.proyeccion_ref !~ '^pep_[A-Za-z0-9_-]{22,128}$'
               OR f.empleado_ref !~ '^emp_[A-Za-z0-9_-]{22,128}$'
               OR f.procedencia_ref !~ '^prc_[A-Za-z0-9_-]{22,128}$'
               OR f.procedencia_huella_sha256 !~ '^[0-9a-f]{64}$'
               OR f.version IS NULL OR f.version < 1 OR f.procedencia_version IS NULL
               OR f.procedencia_version < 1 OR f.vigente_desde IS NULL
               OR f.vigente_hasta IS NULL OR f.vigente_hasta <= f.vigente_desde
               OR p_instante < f.vigente_desde OR p_instante >= f.vigente_hasta THEN
                RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'contrato Personal persona-empleado divergente';
            END IF;
            RETURN QUERY SELECT 'empleado'::text, f.proyeccion_ref, f.version::numeric, f.empleado_ref,
              f.vigente_desde, f.vigente_hasta, f.procedencia_ref, f.procedencia_version::numeric,
              f.procedencia_huella_sha256;
        ELSIF f.resultado IN ('sin_empleado','ambiguo') AND f.empleado_ref IS NULL THEN
            RETURN QUERY SELECT f.resultado, NULL::text, NULL::numeric, NULL::text, NULL::timestamptz,
              NULL::timestamptz, NULL::text, NULL::numeric, NULL::text;
        ELSE
            RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'contrato Personal persona-empleado divergente';
        END IF;
    END LOOP;
    IF n <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'contrato Personal persona-empleado divergente';
    END IF;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz) FROM PUBLIC;

-- Resolución con alcance explícito. Con '{}' produce exactamente los bytes de
-- 000006; con {empleado} sustituye los punteros de empleado del núcleo por la
-- proyección de Personal o deniega.
CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz,
    p_proyecciones text[]
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
    pide_empleado boolean; persona_perfil text; empleado record;
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
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE
       OR p_proyecciones IS NULL
       OR (p_proyecciones IS DISTINCT FROM '{}'::text[]
           AND p_proyecciones IS DISTINCT FROM ARRAY['empleado']::text[]) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'solicitud de contexto actor V2 invalida';
    END IF;
    pide_empleado := p_proyecciones = ARRAY['empleado']::text[];

    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));

    -- Idempotencia por operación, solicitud y alcance. El alcance del registro
    -- se deriva de su canon firmado; otro alcance es colisión.
    IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r WHERE r.operacion_ref = p_operacion_ref) THEN
        RETURN QUERY
        SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,
               r.huella_sha256,r.manifiesto_procedencia_canonico,
               r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
          FROM vec_contexto_actor_v1.registros_contexto r
         WHERE r.operacion_ref=p_operacion_ref AND r.cuenta_ref=p_cuenta_ref
           AND r.perfil_ref=p_perfil_ref AND r.metodo=p_metodo
           AND r.garantia=p_garantia AND r.solicitado_en=p_solicitado_en
           AND vec_contexto_actor_v1.alcance_registro_contexto_v2(r.representacion_canonica) = p_proyecciones;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'colision de operacion de contexto actor V2';
        END IF;
        RETURN;
    END IF;

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
    -- Barrera de Personal antes del reloj y antes de leer su historia:
    -- consultivo compartido por persona y generación FOR SHARE. Una
    -- publicación confirmada tras la instantánea SERIALIZABLE provoca 40001.
    IF pide_empleado THEN
        SELECT pv.persona_ref INTO persona_perfil
          FROM vec_contexto_actor_v1.perfil_actual pa
          JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
         WHERE pa.perfil_ref=p_perfil_ref;
        IF persona_perfil IS NOT NULL THEN
            PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona_perfil);
        END IF;
    END IF;

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

    -- Filtros temporales de 000006 intactos. Con alcance {empleado} los
    -- punteros de empleado del núcleo quedan fuera: Personal es la autoridad.
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
       AND ahora < vr.vigente_hasta
       AND (NOT pide_empleado OR vr.tipo <> 'empleado');
    IF numero_enlaces > 128 OR tipos <> numero_enlaces OR referencias <> numero_enlaces
       OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
           JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
           WHERE vr.persona_ref=perfil.persona_ref
             AND vr.estado='activo' AND ahora >= vr.vigente_desde
             AND ahora < vr.vigente_hasta
             AND (NOT pide_empleado OR vr.tipo <> 'empleado')
             AND vr.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
    END IF;

    IF pide_empleado THEN
        SELECT * INTO STRICT empleado
          FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(perfil.persona_ref,ahora);
        IF empleado.resultado = 'sin_empleado' THEN
            RAISE EXCEPTION USING ERRCODE = 'PCA01', MESSAGE = 'proyeccion empleado pedida sin empleado canonico';
        ELSIF empleado.resultado <> 'empleado' THEN
            RAISE EXCEPTION USING ERRCODE = 'PCA02', MESSAGE = 'proyeccion empleado pedida ambigua';
        END IF;
        IF numero_enlaces >= 128 THEN
            RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
        END IF;
        -- 'empleado' es el mayor tipo admitido y el núcleo ya no aporta
        -- ninguno: añadirlo al final conserva el orden canónico.
        enlaces_texto := concat_ws(',', enlaces_texto, format(
             '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"estado":"activo","vigente_desde":%s,"vigente_hasta":%s}',
             to_json(empleado.proyeccion_ref)::text, empleado.version::text,
             to_json(empleado.empleado_ref)::text,
             to_json(to_char(empleado.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
             to_json(to_char(empleado.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text));
        enlaces_procedencia_texto := concat_ws(',', enlaces_procedencia_texto, format(
             '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":"autoridad_maestra_acreditada"}',
             to_json(empleado.proyeccion_ref)::text, empleado.version::text,
             to_json(empleado.empleado_ref)::text, to_json(empleado.procedencia_ref)::text,
             empleado.procedencia_version::text, to_json(empleado.procedencia_huella_sha256)::text));
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
    IF vec_contexto_actor_v1.alcance_registro_contexto_v2(canonica) IS DISTINCT FROM p_proyecciones THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'alcance de contexto actor V2 no reconstruible';
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

-- Reconciliación tras COMMIT ambiguo: además de la solicitud exacta exige el
-- mismo alcance, derivado del canon almacenado.
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz,
    p_proyecciones text[]
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF current_setting('transaction_isolation') <> 'read committed'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'reconciliacion de contexto actor V2 requiere READ COMMITTED de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE
       OR p_proyecciones IS NULL
       OR (p_proyecciones IS DISTINCT FROM '{}'::text[]
           AND p_proyecciones IS DISTINCT FROM ARRAY['empleado']::text[]) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'reconciliacion de contexto actor V2 invalida';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));
    RETURN QUERY
    SELECT r.operacion_ref, r.registro_contexto_ref, r.representacion_canonica,
           r.huella_sha256,r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.operacion_ref = p_operacion_ref
       AND r.registro_contexto_ref = p_registro_contexto_ref
       AND r.cuenta_ref = p_cuenta_ref AND r.perfil_ref = p_perfil_ref
       AND r.metodo = p_metodo AND r.garantia = p_garantia
       AND r.solicitado_en = p_solicitado_en
       AND vec_contexto_actor_v1.alcance_registro_contexto_v2(r.representacion_canonica) = p_proyecciones;
END
$f$;

-- Las firmas heredadas de siete argumentos equivalen al alcance vacío.
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
    RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
      p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,
      p_garantia,p_solicitado_en,'{}'::text[]);
END
$f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
    RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
      p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,
      p_garantia,p_solicitado_en,'{}'::text[]);
END
$f$;

-- Acreditación de uso: mismo contrato; el alcance sale del registro firmado.
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
    pide_empleado boolean;
    empleado record;
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
    -- El alcance procede del canon firmado; no lo aporta el consumidor.
    pide_empleado := vec_contexto_actor_v1.alcance_registro_contexto_v2(
        registro.representacion_canonica) = ARRAY['empleado']::text[];

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

    -- Barrera de Personal antes del reloj: el consultivo solo ordena frente
    -- al publicador; la generación FOR SHARE impide acreditar con la
    -- instantánea anterior a una publicación ya confirmada (40001).
    IF pide_empleado THEN
        PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(p_persona_ref);
    END IF;

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
       AND ahora < v.vigente_hasta
       AND (NOT pide_empleado OR v.tipo <> 'empleado');

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
              AND (NOT pide_empleado OR v.tipo <> 'empleado')
              AND (v.procedencia_autoridad <> p_autoridad_efectiva
                   OR p_emitida_en < v.vigente_desde
                   OR p_valida_hasta > v.vigente_hasta)
       ) THEN
        RETURN NULL;
    END IF;

    -- Revalidación en el instante autoritativo: la proyección de Personal debe
    -- seguir siendo la única efectiva y cubrir toda la vigencia del uso. Una
    -- revocación, baja, caducidad o nueva versión cambia los bytes y anula.
    IF pide_empleado THEN
        SELECT * INTO STRICT empleado
          FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(p_persona_ref, ahora);
        IF empleado.resultado IS DISTINCT FROM 'empleado' OR numero_vinculos >= 128
           OR p_emitida_en < empleado.vigente_desde
           OR p_valida_hasta > empleado.vigente_hasta THEN
            RETURN NULL;
        END IF;
        vinculos_texto := concat_ws(',', vinculos_texto, pg_catalog.format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"estado":"activo","vigente_desde":%s,"vigente_hasta":%s}',
          pg_catalog.to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          pg_catalog.to_json(empleado.empleado_ref)::text,
          pg_catalog.to_json(pg_catalog.to_char(
              empleado.vigente_desde AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
          pg_catalog.to_json(pg_catalog.to_char(
              empleado.vigente_hasta AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text));
        vinculos_procedencia_texto := concat_ws(',', vinculos_procedencia_texto, pg_catalog.format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":"autoridad_maestra_acreditada"}',
          pg_catalog.to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          pg_catalog.to_json(empleado.empleado_ref)::text,
          pg_catalog.to_json(empleado.procedencia_ref)::text,
          empleado.procedencia_version::text,
          pg_catalog.to_json(empleado.procedencia_huella_sha256)::text));
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
        OR datetime_field_overflow OR no_data_found OR too_many_rows
        OR object_not_in_prerequisite_state THEN
        RETURN NULL;
END
$funcion$;


-- Estructura del runtime: cinco funciones (dos firmas nuevas con alcance).
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;
BEGIN
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contexto_actor_v1_runtime';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
    ];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 5 OR array_position(funciones,NULL) IS NOT NULL
       OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper OR NOT login.rolinherit OR login.rolcreaterole OR login.rolcreatedb
       OR login.rolreplication OR login.rolbypassrls OR login.rolconfig IS NOT NULL
       OR grupo.rolcanlogin OR grupo.rolsuper OR grupo.rolinherit
       OR grupo.rolcreaterole OR grupo.rolcreatedb OR grupo.rolreplication
       OR grupo.rolbypassrls OR grupo.rolconfig IS NOT NULL
       OR current_setting('role') <> 'none' OR membresias <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,r.oid,'MEMBER')
              AND r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting s
            WHERE s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR a.grantee IN (login_oid,runtime_oid)
               OR a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy p
            WHERE login_oid=ANY(p.polroles) OR runtime_oid=ANY(p.polroles)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=login_oid
       )
       OR NOT COALESCE((
           SELECT count(*)=7 AND bool_and(
             d.deptype='a' AND d.objsubid=0 AND (
               (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base_oid) OR
               (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema_oid) OR
               (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))
             ))
             FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) a
            WHERE b.oid=base_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) a
            WHERE n.oid=esquema_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=5 AND count(DISTINCT p.oid)=5
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
            login_oid,base_oid,esquema_oid,funciones) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN runtime de contexto actor V1 no acreditado';
    END IF;
    RETURN session_user;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1() FROM PUBLIC;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[]) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[]) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])
    TO vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])
    TO vec_contexto_actor_v1_runtime;

-- Subdecisión: ninguna alta nueva de puntero de empleado en el núcleo. Un
-- puntero de empleado existente solo admite versiones que lo cierran: la
-- última versión de su vinculo_ref ya debe ser de empleado, con la misma
-- persona y referencia, y la nueva solo puede revocarlo o recortar su
-- vigencia (ventana contenida en la anterior). Un candidato no pasa a
-- empleado y un empleado revocado no se reactiva ni se reabre. Las filas de
-- tipo candidato no se examinan. Se serializa por vinculo_ref; la comprobación
-- lee lo confirmado tras el bloqueo (los mutadores del núcleo escriben en
-- READ COMMITTED).
CREATE FUNCTION vec_contexto_actor_v1.rechazar_alta_puntero_empleado_v2()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog AS $f$
DECLARE previa record;
BEGIN
    IF NEW.tipo <> 'empleado' THEN
        RETURN NEW;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:puntero_empleado:' || NEW.vinculo_ref,0));
    SELECT v.version, v.persona_ref, v.tipo, v.referencia, v.estado,
           v.vigente_desde, v.vigente_hasta
      INTO previa
      FROM vec_contexto_actor_v1.vinculo_referencia_versiones v
     WHERE v.vinculo_ref = NEW.vinculo_ref
     ORDER BY v.version DESC LIMIT 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'alta de puntero empleado cerrada: el empleado procede de Personal';
    END IF;
    IF NEW.version <= previa.version
       OR previa.tipo <> 'empleado'
       OR previa.persona_ref <> NEW.persona_ref
       OR previa.referencia <> NEW.referencia
       OR previa.estado <> 'activo'
       OR NEW.vigente_desde < previa.vigente_desde
       OR NEW.vigente_hasta > previa.vigente_hasta
       OR NOT (NEW.estado = 'revocado'
               OR NEW.vigente_desde > previa.vigente_desde
               OR NEW.vigente_hasta < previa.vigente_hasta) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'version de puntero empleado cerrada: solo revocacion o recorte de vigencia';
    END IF;
    RETURN NEW;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.rechazar_alta_puntero_empleado_v2() FROM PUBLIC;
CREATE TRIGGER alta_puntero_empleado_cerrada_v2
    BEFORE INSERT ON vec_contexto_actor_v1.vinculo_referencia_versiones
    FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_alta_puntero_empleado_v2();

-- Inventario de los punteros de empleado ya existentes en esta base. Solo
-- referencias opacas y estado actual; no se modifican.
DO $inventario$
DECLARE p record; total integer := 0;
BEGIN
    FOR p IN
        SELECT a.vinculo_ref, a.version, v.estado,
               v.vigente_hasta > pg_catalog.clock_timestamp() AS no_caducado
          FROM vec_contexto_actor_v1.vinculo_referencia_actual a
          JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref, version)
         WHERE v.tipo = 'empleado'
         ORDER BY a.vinculo_ref
    LOOP
        total := total + 1;
        RAISE NOTICE 'ContextoActor 000007 inventario puntero empleado: % v% % %', p.vinculo_ref, p.version,
            p.estado, CASE WHEN p.no_caducado THEN 'vigente_hasta_futuro' ELSE 'caducado' END;
    END LOOP;
    RAISE NOTICE 'ContextoActor 000007 inventario: % punteros de empleado conservados', total;
END
$inventario$;

-- Postimagen: runtime con exactamente cinco funciones y Personal solo para el
-- propietario de ContextoActor.
DO $postimagen$
DECLARE runtime oid := 'vec_contexto_actor_v1_runtime'::regrole;
        pe oid := 'vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)'::regprocedure;
        pb oid := 'vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)'::regprocedure;
BEGIN
    IF (SELECT count(*) FROM pg_catalog.pg_proc p
          WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
            AND pg_catalog.has_function_privilege(runtime, p.oid, 'EXECUTE')) <> 5
       OR pg_catalog.has_function_privilege(runtime, pe, 'EXECUTE')
       OR pg_catalog.has_function_privilege(runtime, pb, 'EXECUTE')
       OR NOT pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario', pe, 'EXECUTE')
       OR NOT pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario', pb, 'EXECUTE')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p, pg_catalog.aclexplode(p.proacl) acl
                   WHERE p.oid IN (pe, pb) AND acl.grantee = 0) THEN
        RAISE EXCEPTION 'postimagen ContextoActor 000007 divergente' USING ERRCODE='55000';
    END IF;
END
$postimagen$;
COMMIT;
