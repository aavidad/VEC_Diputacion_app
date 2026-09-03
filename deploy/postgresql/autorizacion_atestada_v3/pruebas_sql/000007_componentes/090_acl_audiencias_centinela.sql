-- Prueba focal F0-C3: publicacion minima, clausura catalogal y cuatro cruces.

CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba()
RETURNS pg_catalog.bool
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    contexto pg_catalog.oid :=
        'vec_contexto_actor_v1_propietario'::pg_catalog.regrole;
    contratacion pg_catalog.oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
    esquema pg_catalog.oid :=
        'vec_autorizacion_atestada_v3'::pg_catalog.regnamespace;
    funcion pg_catalog.oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'
    )::pg_catalog.oid;
    propietario pg_catalog.oid :=
        'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole;
    serializar pg_catalog.oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()'
    )::pg_catalog.oid;
    centinela pg_catalog.oid;
    total pg_catalog.int8;
    validos pg_catalog.int8;
BEGIN
    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = propietario
                 AND NOT a.is_grantable
                 AND a.privilege_type = 'EXECUTE'
                 AND a.grantee IN (propietario, contexto)
           )
      INTO total, validos
      FROM pg_catalog.pg_proc AS p
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))
      ) AS a
     WHERE p.oid = funcion
       AND p.proowner = propietario
       AND p.prokind = 'f'
       AND p.prorettype = 'record'::pg_catalog.regtype
       AND p.proretset
       AND p.pronargs = 11
       AND p.pronargdefaults = 0
       AND p.proargdefaults IS NULL
       AND p.provariadic = 0
       AND p.provolatile = 'v'
       AND NOT p.proisstrict
       AND p.prosecdef
       AND NOT p.proleakproof
       AND p.proparallel = 'u'
       AND p.procost = 100
       AND p.prorows = 1000
       AND p.prosupport = 0
       AND p.protrftypes IS NULL
       AND p.probin IS NULL
       AND p.prosqlbody IS NULL
       AND p.prolang = (
           SELECT l.oid FROM pg_catalog.pg_language AS l
            WHERE l.lanname = 'plpgsql'
       )
       AND p.proconfig = ARRAY[
           'search_path=pg_catalog', 'lock_timeout=2s'
       ]::pg_catalog.text[]
       AND p.proargnames = ARRAY[
           'p_audiencia_consumo_esperada','p_accion_esperada',
           'p_tipo_efecto_esperado','p_operacion_ref_esperada',
           'p_efecto_ref_esperada','p_huella_efecto_sha256_esperada',
           'p_capacidad_canonica','p_manifiesto_fuente_canonico',
           'p_sobre_cose_sign1','p_evidencia_verificacion',
           'p_raiz_publica_spki','capacidad_ref','fuente_ref',
           'fuente_version','evento_fuente_ref',
           'huella_evento_fuente_sha256','huella_manifiesto_fuente_sha256',
           'operacion_ref','efecto_ref','huella_efecto_sha256',
           'consumo_huella_sha256','consumida_en','consumo_nuevo'
       ]::pg_catalog.text[]
       AND p.proargmodes = ARRAY[
           'i','i','i','i','i','i','i','i','i','i','i',
           't','t','t','t','t','t','t','t','t','t','t','t'
       ]::pg_catalog."char"[]
       AND p.proallargtypes = ARRAY[
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'bytea'::pg_catalog.regtype,'bytea'::pg_catalog.regtype,
           'bytea'::pg_catalog.regtype,'bytea'::pg_catalog.regtype,
           'bytea'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'numeric'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'text'::pg_catalog.regtype,
           'text'::pg_catalog.regtype,'timestamptz'::pg_catalog.regtype,
           'boolean'::pg_catalog.regtype
       ]::pg_catalog.oid[]
       AND (SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_proc AS otro
             WHERE otro.pronamespace = esquema
               AND otro.proname =
                   'consumir_fuente_corporativa_contexto_actor_v1_atestada') = 1;
    IF (total, validos) <>
       (2::pg_catalog.int8, 2::pg_catalog.int8) THEN
        RETURN false;
    END IF;

    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = propietario
                 AND NOT a.is_grantable
                 AND (
                    (a.grantee = propietario AND
                     a.privilege_type IN ('CREATE', 'USAGE'))
                    OR
                    (a.grantee IN (contratacion, contexto) AND
                     a.privilege_type = 'USAGE')
                 )
           )
      INTO total, validos
      FROM pg_catalog.pg_namespace AS n
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
      ) AS a
     WHERE n.oid = esquema AND n.nspowner = propietario;
    IF (total, validos) <>
       (4::pg_catalog.int8, 4::pg_catalog.int8) THEN
        RETURN false;
    END IF;

    SELECT t.oid INTO centinela
      FROM pg_catalog.pg_trigger AS t
     WHERE t.tgrelid =
           'vec_autorizacion_atestada_v3.revocacion_raiz'
               ::pg_catalog.regclass
       AND t.tgname = 'dependencia_f0_fuente_corporativa_v1'
       AND NOT t.tgisinternal
       AND t.tgfoid = serializar
       AND t.tgtype = 7
       AND t.tgenabled = 'D'
       AND t.tgnargs = 0
       AND t.tgparentid = 0
       AND t.tgconstraint = 0
       AND t.tgconstrrelid = 0
       AND NOT t.tgdeferrable
       AND NOT t.tginitdeferred
       AND t.tgqual IS NULL;
    RETURN centinela IS NOT NULL
       AND (SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_trigger AS t
             WHERE t.tgfoid = serializar AND NOT t.tgisinternal) = 4
       AND (SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_depend AS d
             WHERE d.classid = 'pg_catalog.pg_trigger'::pg_catalog.regclass
               AND d.objid = centinela
               AND d.objsubid = 0
               AND d.refclassid = 'pg_catalog.pg_proc'::pg_catalog.regclass
               AND d.refobjid = serializar
               AND d.refobjsubid = 0
               AND d.deptype = 'n') = 1
       AND (SELECT pg_catalog.pg_get_constraintdef(c.oid, false)
              FROM pg_catalog.pg_constraint AS c
             WHERE c.conrelid =
                   'vec_autorizacion_atestada_v3.clave_capacidad_version'
                       ::pg_catalog.regclass
               AND c.conname =
                   'clave_capacidad_version_audiencia_consumo_check'
               AND c.contype = 'c') =
          'CHECK (audiencia_consumo = ANY (ARRAY[''vec_contratacion_temporal.confirmar_alta_atestada.v1''::text, ''vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1''::text, ''vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1''::text, ''vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1''::text, ''vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1''::text, ''vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1''::text, ''vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1''::text]))'
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class AS c
             JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
            WHERE n.oid = esquema AND a.grantee = contexto
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type AS t
             JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
            WHERE n.oid = esquema AND a.grantee = contexto
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
            WHERE p.pronamespace = esquema
              AND p.oid <> funcion
              AND a.grantee = contexto
       )
       AND NOT pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_publicador_corporativo',
           funcion, 'EXECUTE')
       AND NOT pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_revocador_corporativo',
           funcion, 'EXECUTE')
       AND NOT pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_despachador_corporativo',
           funcion, 'EXECUTE');
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() FROM PUBLIC;

DO $forma_c3$
BEGIN
    IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() IS NOT TRUE
    THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'C3: forma, ACL, audiencias o centinela invalidos';
    END IF;
END
$forma_c3$;

-- Cada bloque hostil revierte su propia mutacion y debe dejar C3 exacto.
DO $derivas_c3$
BEGIN
    BEGIN
        EXECUTE 'GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea) TO vec_contexto_actor_v1_publicador_corporativo';
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: EXECUTE R0 adicional aceptado';
        END IF;
        RAISE SQLSTATE 'ZC301';
    EXCEPTION WHEN SQLSTATE 'ZC301' THEN NULL;
    END;
    BEGIN
        EXECUTE 'GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO PUBLIC';
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: USAGE PUBLIC aceptado';
        END IF;
        RAISE SQLSTATE 'ZC302';
    EXCEPTION WHEN SQLSTATE 'ZC302' THEN NULL;
    END;
    BEGIN
        EXECUTE 'ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea) STABLE';
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: deriva de funcion aceptada';
        END IF;
        RAISE SQLSTATE 'ZC303';
    EXCEPTION WHEN SQLSTATE 'ZC303' THEN NULL;
    END;
    BEGIN
        ALTER TABLE vec_autorizacion_atestada_v3.revocacion_raiz
            ENABLE TRIGGER dependencia_f0_fuente_corporativa_v1;
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: centinela habilitado aceptado';
        END IF;
        RAISE SQLSTATE 'ZC304';
    EXCEPTION WHEN SQLSTATE 'ZC304' THEN NULL;
    END;
    BEGIN
        EXECUTE 'GRANT SELECT ON vec_autorizacion_atestada_v3.checkpoint_gobierno TO vec_contexto_actor_v1_propietario';
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: lectura de tabla aceptada';
        END IF;
        RAISE SQLSTATE 'ZC305';
    EXCEPTION WHEN SQLSTATE 'ZC305' THEN NULL;
    END;
    BEGIN
        EXECUTE 'GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea) TO vec_contexto_actor_v1_propietario WITH GRANT OPTION';
        IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'C3: grant option aceptado';
        END IF;
        RAISE SQLSTATE 'ZC306';
    EXCEPTION WHEN SQLSTATE 'ZC306' THEN NULL;
    END;
    IF vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba() IS NOT TRUE
    THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'C3: una deriva no revirtio';
    END IF;
END
$derivas_c3$;

CREATE FUNCTION vec_autorizacion_atestada_v3.manifiesto_c3_prueba(
    p_indice pg_catalog.int4,
    p_emitida pg_catalog.timestamptz
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    audiencias pg_catalog.text[] := ARRAY[
      'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
      'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
      'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
      'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'];
    acciones pg_catalog.text[] := ARRAY[
      'contexto_actor.organizacion_corporativa.publicar',
      'contexto_actor.organizacion_corporativa.revocar',
      'contexto_actor.vinculo_corporativo.publicar',
      'contexto_actor.vinculo_corporativo.revocar'];
    tipos pg_catalog.text[] := ARRAY[
      'organizacion_corporativa.alta','organizacion_corporativa.revocacion',
      'vinculo_corporativo.alta','vinculo_corporativo.revocacion'];
    operacion pg_catalog.text :=
        'oca_f0_c3_' || pg_catalog.lpad(p_indice::pg_catalog.text, 24, '0');
BEGIN
    IF p_indice NOT BETWEEN 1 AND 4 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'C3: indice de prueba invalido';
    END IF;
    RETURN pg_catalog.convert_to(
      '{"esquema":"vec.contexto-actor.fuente-corporativa.manifiesto.v1"' ||
      ',"version":1,"fuente_ref":"fuente:f0-c3-' || p_indice || '"' ||
      ',"fuente_version":' || (900000+p_indice) ||
      ',"evento_fuente_ref":"evento:f0-c3-' || p_indice || '"' ||
      ',"huella_evento_fuente_sha256":"' ||
          pg_catalog.repeat(pg_catalog.substr('1234',p_indice,1),64) || '"' ||
      ',"evento_fuente_emitido_en":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
              p_emitida-pg_catalog.make_interval(secs=>0.1))) ||
      ',"audiencia_consumo":' ||
          vec_autorizacion_atestada_v3.texto_json_go(audiencias[p_indice]) ||
      ',"accion":' ||
          vec_autorizacion_atestada_v3.texto_json_go(acciones[p_indice]) ||
      ',"tipo_efecto":' ||
          vec_autorizacion_atestada_v3.texto_json_go(tipos[p_indice]) ||
      ',"operacion_ref":' ||
          vec_autorizacion_atestada_v3.texto_json_go(operacion) ||
      ',"efecto_ref":"efecto:f0-c3-' || p_indice || '"' ||
      ',"huella_efecto_sha256":"' ||
          pg_catalog.repeat(pg_catalog.substr('abcd',p_indice,1),64) ||
      '"}', 'UTF8');
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.capacidad_c3_prueba(
    p_indice pg_catalog.int4,
    p_manifiesto pg_catalog.bytea,
    p_sobre pg_catalog.bytea,
    p_evidencia pg_catalog.bytea,
    p_spki pg_catalog.bytea,
    p_secreto pg_catalog.bytea,
    p_emitida pg_catalog.timestamptz
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    m pg_catalog.json :=
        pg_catalog.convert_from(p_manifiesto,'UTF8')::pg_catalog.json;
    c pg_catalog.bytea;
    h pg_catalog.text;
BEGIN
    c := pg_catalog.convert_to(
      '{"esquema":"vec.contexto-actor.fuente-corporativa.capacidad.v1"' ||
      ',"version":1,"fuente_ref":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'fuente_ref') ||
      ',"fuente_version":' || (900000+p_indice) ||
      ',"evento_fuente_ref":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'evento_fuente_ref') ||
      ',"huella_evento_fuente_sha256":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            m->>'huella_evento_fuente_sha256') ||
      ',"evento_fuente_emitido_en":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            m->>'evento_fuente_emitido_en') ||
      ',"huella_manifiesto_fuente_sha256":"' ||
          pg_catalog.encode(pg_catalog.sha256(p_manifiesto),'hex') || '"' ||
      ',"huella_sobre_cose_sign1_sha256":"' ||
          pg_catalog.encode(pg_catalog.sha256(p_sobre),'hex') || '"' ||
      ',"huella_prueba_confianza_sha256":"' ||
          pg_catalog.encode(pg_catalog.sha256(p_evidencia),'hex') || '"' ||
      ',"audiencia_consumo":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'audiencia_consumo') ||
      ',"accion":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'accion') ||
      ',"tipo_efecto":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'tipo_efecto') ||
      ',"operacion_ref":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'operacion_ref') ||
      ',"efecto_ref":' ||
          vec_autorizacion_atestada_v3.texto_json_go(m->>'efecto_ref') ||
      ',"huella_efecto_sha256":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            m->>'huella_efecto_sha256') ||
      ',"clave_id":"clave:f0-c3-' || p_indice || '"' ||
      ',"clave_version":' || (900000+p_indice) ||
      ',"revision_gobierno":900001' ||
      ',"huella_gobierno_sha256":"' || pg_catalog.repeat('5',64) || '"' ||
      ',"emisor_id":"emisor:f0-c3"' ||
      ',"configuracion_revision":"configuracion:f0-c3"' ||
      ',"configuracion_secuencia":900001' ||
      ',"huella_configuracion_sha256":"' || pg_catalog.repeat('6',64) || '"' ||
      ',"raiz_clave_id":"raiz:f0-c3","raiz_version":900001' ||
      ',"huella_raiz_spki_sha256":"' ||
          pg_catalog.encode(pg_catalog.sha256(p_spki),'hex') || '"' ||
      ',"audiencia_despliegue":"vec-diputacion/pruebas/f0/c3"' ||
      ',"suite":"VEC-AD-3-COSE-EDDSA-1"' ||
      ',"nonce":"' ||
          pg_catalog.repeat(pg_catalog.substr('89ab',p_indice,1),64) || '"' ||
      ',"emitida_en":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
              p_emitida)) ||
      ',"expira_en":' ||
          vec_autorizacion_atestada_v3.texto_json_go(
            vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
              p_emitida+pg_catalog.make_interval(secs=>5))) ||
      ',"mac_sha256":"' || pg_catalog.repeat('f',64) || '"}', 'UTF8');
    h := pg_catalog.encode(public.hmac(
      vec_autorizacion_atestada_v3.preimagen_mac_fuente_corporativa_v1(c),
      p_secreto, 'sha256'), 'hex');
    RETURN pg_catalog.convert_to(pg_catalog.replace(
      pg_catalog.convert_from(c,'UTF8'), pg_catalog.repeat('f',64), h),'UTF8');
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.manifiesto_c3_prueba(
        pg_catalog.int4, pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.capacidad_c3_prueba(
        pg_catalog.int4, pg_catalog.bytea, pg_catalog.bytea,
        pg_catalog.bytea, pg_catalog.bytea, pg_catalog.bytea,
        pg_catalog.timestamptz)
FROM PUBLIC;

CREATE TEMP TABLE material_c3_prueba (
    indice pg_catalog.int4 PRIMARY KEY,
    audiencia pg_catalog.text,
    accion pg_catalog.text,
    tipo pg_catalog.text,
    operacion pg_catalog.text,
    efecto pg_catalog.text,
    huella_efecto pg_catalog.text,
    spki pg_catalog.bytea,
    sobre pg_catalog.bytea,
    evidencia pg_catalog.bytea,
    manifiesto pg_catalog.bytea,
    capacidad pg_catalog.bytea
) ON COMMIT DROP;

DO $fixture_c3$
DECLARE
    ahora pg_catalog.timestamptz := pg_catalog.clock_timestamp();
    audiencias pg_catalog.text[] := ARRAY[
      'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
      'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
      'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
      'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'];
    acciones pg_catalog.text[] := ARRAY[
      'contexto_actor.organizacion_corporativa.publicar',
      'contexto_actor.organizacion_corporativa.revocar',
      'contexto_actor.vinculo_corporativo.publicar',
      'contexto_actor.vinculo_corporativo.revocar'];
    tipos pg_catalog.text[] := ARRAY[
      'organizacion_corporativa.alta','organizacion_corporativa.revocacion',
      'vinculo_corporativo.alta','vinculo_corporativo.revocacion'];
    i pg_catalog.int4;
    manifiesto pg_catalog.bytea;
    capacidad pg_catalog.bytea;
    secreto pg_catalog.bytea;
    spki pg_catalog.bytea := pg_catalog.decode(
      '302a300506032b6570032100'||pg_catalog.repeat('a3',32),'hex');
    sobre pg_catalog.bytea := pg_catalog.decode(
      pg_catalog.repeat('b3',128),'hex');
    evidencia pg_catalog.bytea := pg_catalog.decode(
      pg_catalog.repeat('c3',32),'hex');
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
    VALUES ('configuracion:f0-c3',900001,pg_catalog.repeat('6',64),
      ahora-pg_catalog.make_interval(hours=>1),
      ahora+pg_catalog.make_interval(hours=>1),'acto:f0-c3-config',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
    VALUES ('raiz:f0-c3',900001,spki,
      pg_catalog.encode(pg_catalog.sha256(spki),'hex'),
      ahora-pg_catalog.make_interval(hours=>1),
      ahora+pg_catalog.make_interval(hours=>1),'VEC-AD-3-COSE-EDDSA-1',
      'vec-diputacion/pruebas/f0/c3','acto:f0-c3-raiz',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
    VALUES ('configuracion:f0-c3','raiz:f0-c3',900001);
    INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
    VALUES (900001,'configuracion:f0-c3',
      ahora-pg_catalog.make_interval(hours=>1),
      'acto:f0-c3-puntero-config',ahora);
    ALTER TABLE vec_autorizacion_atestada_v3
      .fuente_corporativa_contexto_actor_v1
      DISABLE TRIGGER f0_checkpoint_antes;
    FOR i IN 1..4 LOOP
        secreto := pg_catalog.sha256(pg_catalog.convert_to(
          'secreto-hmac-f0-c3-'||i,'UTF8'));
        INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
        VALUES ('clave:f0-c3-'||i,900000+i,900001,
          pg_catalog.repeat('5',64),secreto,
          pg_catalog.encode(pg_catalog.sha256(secreto),'hex'),'emisor:f0-c3',
          audiencias[i],ahora-pg_catalog.make_interval(hours=>1),
          ahora+pg_catalog.make_interval(hours=>1),
          'acto:f0-c3-clave-'||i,ahora);
        INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
        VALUES (900000+i,'clave:f0-c3-'||i,900000+i,
          ahora-pg_catalog.make_interval(hours=>1),
          'acto:f0-c3-puntero-clave-'||i,ahora);
        INSERT INTO vec_autorizacion_atestada_v3
          .fuente_corporativa_contexto_actor_v1 VALUES (
          'fuente:f0-c3-'||i,900000+i,audiencias[i],acciones[i],tipos[i],
          'clave:f0-c3-'||i,900000+i,900001,pg_catalog.repeat('5',64),
          'emisor:f0-c3','configuracion:f0-c3',900001,
          pg_catalog.repeat('6',64),'raiz:f0-c3',900001,
          pg_catalog.encode(pg_catalog.sha256(spki),'hex'),
          'vec-diputacion/pruebas/f0/c3','VEC-AD-3-COSE-EDDSA-1',
          ahora-pg_catalog.make_interval(hours=>1),
          ahora+pg_catalog.make_interval(hours=>1),
          'acto:f0-c3-fuente-'||i,ahora);
        manifiesto := vec_autorizacion_atestada_v3.manifiesto_c3_prueba(
          i, ahora-pg_catalog.make_interval(secs=>0.02));
        capacidad := vec_autorizacion_atestada_v3.capacidad_c3_prueba(
          i,manifiesto,sobre,evidencia,spki,secreto,
          ahora-pg_catalog.make_interval(secs=>0.02));
        INSERT INTO pg_temp.material_c3_prueba VALUES (
          i,audiencias[i],acciones[i],tipos[i],
          'oca_f0_c3_'||pg_catalog.lpad(i::pg_catalog.text,24,'0'),
          'efecto:f0-c3-'||i,
          pg_catalog.repeat(pg_catalog.substr('abcd',i,1),64),
          spki,sobre,evidencia,manifiesto,capacidad);
    END LOOP;
    ALTER TABLE vec_autorizacion_atestada_v3
      .fuente_corporativa_contexto_actor_v1
      ENABLE TRIGGER f0_checkpoint_antes;
    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision=revision+1,
           configuracion_secuencia_minima=pg_catalog.greatest(
             configuracion_secuencia_minima,900001),
           raiz_version_minima=pg_catalog.greatest(
             raiz_version_minima,900001),
           actualizada_en=ahora
     WHERE control_id;
END
$fixture_c3$;

GRANT SELECT ON pg_temp.material_c3_prueba TO
    vec_contexto_actor_v1_propietario,
    vec_contexto_actor_v1_publicador_corporativo,
    vec_contexto_actor_v1_revocador_corporativo,
    vec_contexto_actor_v1_despachador_corporativo,
    vec_f0_h0_cruzado, vec_f0_h0_extra, vec_f0_h0_sin_rol;

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.fachada_f0_c3_prueba(
    p_indice pg_catalog.int4
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    m record;
    cantidad pg_catalog.int8;
    nuevos pg_catalog.int8;
BEGIN
    SELECT * INTO STRICT m
      FROM pg_temp.material_c3_prueba AS x WHERE x.indice=p_indice;
    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (WHERE r.consumo_nuevo)
      INTO cantidad,nuevos
      FROM vec_autorizacion_atestada_v3
        .consumir_fuente_corporativa_contexto_actor_v1_atestada(
          m.audiencia,m.accion,m.tipo,m.operacion,m.efecto,m.huella_efecto,
          m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki) AS r;
    RETURN cantidad=1 AND nuevos=1;
END
$funcion$;

CREATE FUNCTION vec_contexto_actor_v1.directa_f0_c3_prueba(
    p_indice pg_catalog.int4
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    m record;
BEGIN
    SELECT * INTO STRICT m
      FROM pg_temp.material_c3_prueba AS x WHERE x.indice=p_indice;
    PERFORM * FROM vec_autorizacion_atestada_v3
      .consumir_fuente_corporativa_contexto_actor_v1_atestada(
        m.audiencia,m.accion,m.tipo,m.operacion,m.efecto,m.huella_efecto,
        m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
    RETURN true;
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_contexto_actor_v1.fachada_f0_c3_prueba(pg_catalog.int4),
    vec_contexto_actor_v1.directa_f0_c3_prueba(pg_catalog.int4)
FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO
    vec_contexto_actor_v1_publicador_corporativo,
    vec_contexto_actor_v1_revocador_corporativo,
    vec_contexto_actor_v1_despachador_corporativo,
    vec_f0_h0_cruzado, vec_f0_h0_extra, vec_f0_h0_sin_rol;
GRANT EXECUTE ON FUNCTION
    vec_contexto_actor_v1.fachada_f0_c3_prueba(pg_catalog.int4),
    vec_contexto_actor_v1.directa_f0_c3_prueba(pg_catalog.int4)
TO vec_contexto_actor_v1_publicador_corporativo,
   vec_contexto_actor_v1_revocador_corporativo,
   vec_contexto_actor_v1_despachador_corporativo,
   vec_f0_h0_cruzado, vec_f0_h0_extra, vec_f0_h0_sin_rol;

DO $public_c3$
BEGIN
    IF pg_catalog.has_schema_privilege(
           'vec_f0_h0_adicional','vec_contexto_actor_v1','USAGE')
       OR pg_catalog.has_function_privilege(
           'vec_f0_h0_adicional',
           'vec_contexto_actor_v1.fachada_f0_c3_prueba(integer)',
           'EXECUTE') THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: PUBLIC obtuvo la fachada sintetica';
    END IF;
END
$public_c3$;

RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
DO $publicador_c3$
DECLARE i pg_catalog.int4;
BEGIN
    FOREACH i IN ARRAY ARRAY[1,3] LOOP
        BEGIN
            PERFORM vec_contexto_actor_v1.directa_f0_c3_prueba(i);
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C3: llamada directa de publicador aceptada';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
        IF vec_contexto_actor_v1.fachada_f0_c3_prueba(i) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C3: publicacion anidada rechazada';
        END IF;
    END LOOP;
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(2);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: publicador acepto revocacion';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        EXECUTE 'UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET revision=revision';
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: DML R0 aceptado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$publicador_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_revocador;
DO $revocador_c3$
DECLARE i pg_catalog.int4;
BEGIN
    FOREACH i IN ARRAY ARRAY[2,4] LOOP
        BEGIN
            PERFORM vec_contexto_actor_v1.directa_f0_c3_prueba(i);
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C3: llamada directa de revocador aceptada';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
        IF vec_contexto_actor_v1.fachada_f0_c3_prueba(i) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C3: revocacion anidada rechazada';
        END IF;
    END LOOP;
END
$revocador_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_despachador;
DO $despachador_c3$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(1);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: despachador aceptado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$despachador_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_cruzado;
DO $cruzado_c3$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(1);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: membresia cruzada aceptada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$cruzado_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_extra;
DO $extra_c3$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(1);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: membresia adicional aceptada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$extra_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_sin_rol;
DO $sin_rol_c3$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(1);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: LOGIN sin R0 aceptado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$sin_rol_c3$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
SET LOCAL ROLE vec_f0_h0_publicador;
DO $set_role_c3$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.fachada_f0_c3_prueba(1);
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: SET ROLE aceptado';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$set_role_c3$;
RESET ROLE;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

DO $efectos_c3$
BEGIN
    IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .atestacion_fuente_corporativa_contexto_actor_v1) <> 4
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1) <> 4
       OR vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba()
          IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: efectos o clausura finales invalidos';
    END IF;
    BEGIN
        DROP FUNCTION vec_autorizacion_atestada_v3
            .serializar_revocacion_consultas_rrhh_v3();
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C3: centinela no bloqueo retirada';
    EXCEPTION WHEN dependent_objects_still_exist THEN NULL;
    END;
END
$efectos_c3$;

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
REVOKE EXECUTE ON FUNCTION
    vec_contexto_actor_v1.fachada_f0_c3_prueba(pg_catalog.int4),
    vec_contexto_actor_v1.directa_f0_c3_prueba(pg_catalog.int4)
FROM vec_contexto_actor_v1_publicador_corporativo,
     vec_contexto_actor_v1_revocador_corporativo,
     vec_contexto_actor_v1_despachador_corporativo,
     vec_f0_h0_cruzado, vec_f0_h0_extra, vec_f0_h0_sin_rol;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM
    vec_contexto_actor_v1_publicador_corporativo,
    vec_contexto_actor_v1_revocador_corporativo,
    vec_contexto_actor_v1_despachador_corporativo,
    vec_f0_h0_cruzado, vec_f0_h0_extra, vec_f0_h0_sin_rol;
DROP FUNCTION vec_contexto_actor_v1.directa_f0_c3_prueba(pg_catalog.int4);
DROP FUNCTION vec_contexto_actor_v1.fachada_f0_c3_prueba(pg_catalog.int4);

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_c3_prueba(
    pg_catalog.int4,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
    pg_catalog.bytea,pg_catalog.bytea,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.manifiesto_c3_prueba(
    pg_catalog.int4,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.acreditar_forma_c3_prueba();
DROP TABLE pg_temp.material_c3_prueba;
