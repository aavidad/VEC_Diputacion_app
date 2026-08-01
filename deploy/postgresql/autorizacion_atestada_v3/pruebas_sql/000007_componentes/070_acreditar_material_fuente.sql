-- Prueba focal F0-C1: acreditacion privada, cruces y locks de gobierno.
CREATE FUNCTION vec_autorizacion_atestada_v3.perfil_c1_prueba(p_indice pg_catalog.int4)
RETURNS TABLE (audiencia pg_catalog.text, accion pg_catalog.text, tipo pg_catalog.text)
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog
AS $funcion$
    SELECT p.audiencia, p.accion, p.tipo
      FROM (VALUES
        (1,'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
           'contexto_actor.organizacion_corporativa.publicar','organizacion_corporativa.alta'),
        (2,'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
           'contexto_actor.organizacion_corporativa.revocar','organizacion_corporativa.revocacion'),
        (3,'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
           'contexto_actor.vinculo_corporativo.publicar','vinculo_corporativo.alta'),
        (4,'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
           'contexto_actor.vinculo_corporativo.revocar','vinculo_corporativo.revocacion')
      ) AS p(indice,audiencia,accion,tipo)
     WHERE p.indice = p_indice
$funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.manifiesto_c1_prueba(
    p_indice pg_catalog.int4, p_emitida_en pg_catalog.timestamptz)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_perfil record;
BEGIN
    SELECT * INTO STRICT v_perfil
      FROM vec_autorizacion_atestada_v3.perfil_c1_prueba(p_indice);
    RETURN pg_catalog.convert_to(
        '{"esquema":' || vec_autorizacion_atestada_v3.texto_json_go(
            'vec.contexto-actor.fuente-corporativa.manifiesto.v1') ||
        ',"version":1,"fuente_ref":"fuente:f0-c1-sintetica"' ||
        ',"fuente_version":700001,"evento_fuente_ref":"evento:f0-c1-' ||
            p_indice::pg_catalog.text || '"' ||
        ',"huella_evento_fuente_sha256":"' || pg_catalog.repeat('1',64) ||
        '","evento_fuente_emitido_en":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                vec_autorizacion_atestada_v3
                    .representacion_instante_utc_fuente(
                        p_emitida_en-pg_catalog.make_interval(secs=>0.1))) ||
        ',"audiencia_consumo":' || vec_autorizacion_atestada_v3
            .texto_json_go(v_perfil.audiencia) || ',"accion":' ||
            vec_autorizacion_atestada_v3.texto_json_go(v_perfil.accion) ||
        ',"tipo_efecto":' || vec_autorizacion_atestada_v3.texto_json_go(v_perfil.tipo) ||
        ',"operacion_ref":"oca_f0_c1_' ||
            pg_catalog.lpad(p_indice::pg_catalog.text,20,'0') || '"' ||
        ',"efecto_ref":"efecto:f0-c1-' || p_indice::pg_catalog.text || '"' ||
        ',"huella_efecto_sha256":"' || pg_catalog.repeat('4',64) || '"}',
        'UTF8');
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.capacidad_c1_prueba(
    p_indice pg_catalog.int4, p_manifiesto pg_catalog.bytea,
    p_sobre pg_catalog.bytea, p_evidencia pg_catalog.bytea,
    p_spki pg_catalog.bytea, p_secreto pg_catalog.bytea,
    p_emitida_en pg_catalog.timestamptz)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog
AS $funcion$
DECLARE
    m pg_catalog.json:=pg_catalog.convert_from(p_manifiesto,'UTF8')::pg_catalog.json;
    v_capacidad pg_catalog.bytea;
    v_mac pg_catalog.text;
BEGIN
    v_capacidad := pg_catalog.convert_to(
        '{"esquema":' || vec_autorizacion_atestada_v3.texto_json_go(
            'vec.contexto-actor.fuente-corporativa.capacidad.v1') ||
        ',"version":1,"fuente_ref":' ||
            vec_autorizacion_atestada_v3.texto_json_go(m->>'fuente_ref') ||
        ',"fuente_version":' || (m->'fuente_version')::pg_catalog.text ||
        ',"evento_fuente_ref":' || vec_autorizacion_atestada_v3
            .texto_json_go(m->>'evento_fuente_ref') ||
        ',"huella_evento_fuente_sha256":' || vec_autorizacion_atestada_v3
            .texto_json_go(m->>'huella_evento_fuente_sha256') ||
        ',"evento_fuente_emitido_en":' || vec_autorizacion_atestada_v3
            .texto_json_go(m->>'evento_fuente_emitido_en') ||
        ',"huella_manifiesto_fuente_sha256":"' ||
            pg_catalog.encode(pg_catalog.sha256(p_manifiesto),'hex') || '"' ||
        ',"huella_sobre_cose_sign1_sha256":"' ||
            pg_catalog.encode(pg_catalog.sha256(p_sobre),'hex') || '"' ||
        ',"huella_prueba_confianza_sha256":"' ||
            pg_catalog.encode(pg_catalog.sha256(p_evidencia),'hex') || '"' ||
        ',"audiencia_consumo":' || vec_autorizacion_atestada_v3
            .texto_json_go(m->>'audiencia_consumo') ||
        ',"accion":' || vec_autorizacion_atestada_v3.texto_json_go(
            m->>'accion') || ',"tipo_efecto":' ||
            vec_autorizacion_atestada_v3.texto_json_go(m->>'tipo_efecto') ||
        ',"operacion_ref":' || vec_autorizacion_atestada_v3.texto_json_go(
            m->>'operacion_ref') || ',"efecto_ref":' ||
            vec_autorizacion_atestada_v3.texto_json_go(m->>'efecto_ref') ||
        ',"huella_efecto_sha256":' || vec_autorizacion_atestada_v3
            .texto_json_go(m->>'huella_efecto_sha256') ||
        ',"clave_id":"clave:f0-c1-' || p_indice::pg_catalog.text || '"' ||
        ',"clave_version":' || (700000+p_indice)::pg_catalog.text ||
        ',"revision_gobierno":' || (700000+p_indice)::pg_catalog.text ||
        ',"huella_gobierno_sha256":"' || pg_catalog.repeat('5',64) || '"' ||
        ',"emisor_id":"emisor:f0-c1-' || p_indice::pg_catalog.text || '"' ||
        ',"configuracion_revision":"configuracion:f0-c1"' ||
        ',"configuracion_secuencia":700001' ||
        ',"huella_configuracion_sha256":"' ||
            pg_catalog.repeat('6',64) || '"' ||
        ',"raiz_clave_id":"raiz:f0-c1","raiz_version":700001' ||
        ',"huella_raiz_spki_sha256":"' ||
            pg_catalog.encode(pg_catalog.sha256(p_spki),'hex') || '"' ||
        ',"audiencia_despliegue":"vec-diputacion/pruebas/f0/c1"' ||
        ',"suite":"VEC-AD-3-COSE-EDDSA-1","nonce":"' ||
            pg_catalog.repeat('8',64) || '","emitida_en":' ||
            vec_autorizacion_atestada_v3.texto_json_go(
                vec_autorizacion_atestada_v3
                    .representacion_instante_utc_fuente(p_emitida_en)) ||
        ',"expira_en":' || vec_autorizacion_atestada_v3.texto_json_go(
            vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
                p_emitida_en+pg_catalog.make_interval(secs=>5))) ||
        ',"mac_sha256":"' || pg_catalog.repeat('9',64) || '"}', 'UTF8');
    v_mac := pg_catalog.encode(public.hmac(
        vec_autorizacion_atestada_v3
            .preimagen_mac_fuente_corporativa_v1(v_capacidad),
        p_secreto,'sha256'),'hex');
    RETURN pg_catalog.convert_to(pg_catalog.replace(
        pg_catalog.convert_from(v_capacidad,'UTF8'),
        '"mac_sha256":"'||pg_catalog.repeat('9',64)||'"',
        '"mac_sha256":"'||v_mac||'"'),'UTF8');
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.material_c1_prueba(
    p_indice pg_catalog.int4, p_emitida pg_catalog.timestamptz)
RETURNS TABLE (secreto pg_catalog.bytea, spki pg_catalog.bytea,
    sobre pg_catalog.bytea, evidencia pg_catalog.bytea,
    manifiesto pg_catalog.bytea, capacidad pg_catalog.bytea)
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog
AS $funcion$
BEGIN
    secreto:=pg_catalog.sha256(pg_catalog.convert_to(
        'secreto-hmac-f0-c1-'||p_indice,'UTF8'));
    spki:=pg_catalog.decode(
        '302a300506032b6570032100'||pg_catalog.repeat('a1',32),'hex');
    sobre:=pg_catalog.decode(pg_catalog.repeat('b1',128),'hex');
    evidencia:=pg_catalog.decode(pg_catalog.repeat('c1',32),'hex');
    manifiesto:=vec_autorizacion_atestada_v3.manifiesto_c1_prueba(
        p_indice,p_emitida);
    capacidad:=vec_autorizacion_atestada_v3.capacidad_c1_prueba(
        p_indice,manifiesto,sobre,evidencia,spki,secreto,p_emitida);
    RETURN NEXT;
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
    p_capacidad pg_catalog.bytea, p_campo pg_catalog.text,
    p_valor_json pg_catalog.text, p_secreto pg_catalog.bytea)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_texto pg_catalog.text:=pg_catalog.convert_from(p_capacidad,'UTF8');
    v_json pg_catalog.json:=v_texto::pg_catalog.json;
    v_mac_anterior pg_catalog.text:=v_json->>'mac_sha256';
    v_alterada pg_catalog.bytea;
    v_mac pg_catalog.text;
BEGIN
    v_texto := pg_catalog.replace(v_texto,
        '"'||p_campo||'":'||(v_json->p_campo)::pg_catalog.text,
        '"'||p_campo||'":'||p_valor_json);
    v_alterada := pg_catalog.convert_to(v_texto,'UTF8');
    v_mac := pg_catalog.encode(public.hmac(
        vec_autorizacion_atestada_v3
            .preimagen_mac_fuente_corporativa_v1(v_alterada),
        p_secreto,'sha256'),'hex');
    RETURN pg_catalog.convert_to(pg_catalog.replace(v_texto,
        '"mac_sha256":"'||v_mac_anterior||'"',
        '"mac_sha256":"'||v_mac||'"'),'UTF8');
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.invocar_c1_prueba(
    p_indice pg_catalog.int4, p_capacidad pg_catalog.bytea,
    p_manifiesto pg_catalog.bytea, p_sobre pg_catalog.bytea,
    p_evidencia pg_catalog.bytea, p_spki pg_catalog.bytea)
RETURNS pg_catalog.jsonb
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_perfil record;
    m pg_catalog.json:=pg_catalog.convert_from(p_manifiesto,'UTF8')::pg_catalog.json;
    v_salida pg_catalog.jsonb;
BEGIN
    SELECT * INTO STRICT v_perfil
      FROM vec_autorizacion_atestada_v3.perfil_c1_prueba(p_indice);
    SELECT pg_catalog.to_jsonb(r) INTO v_salida
      FROM vec_autorizacion_atestada_v3
        .acreditar_material_fuente_corporativa_contexto_actor_v1(
            v_perfil.audiencia,v_perfil.accion,v_perfil.tipo,
            m->>'operacion_ref',m->>'efecto_ref',m->>'huella_efecto_sha256',
            p_capacidad,p_manifiesto,p_sobre,p_evidencia,p_spki) AS r;
    RETURN v_salida;
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.locks_c1_prueba(
    p_checkpoint_esperado pg_catalog.bool)
RETURNS pg_catalog.bool
LANGUAGE sql VOLATILE SET search_path = pg_catalog
AS $funcion$
    WITH relaciones(nombre,checkpoint) AS (VALUES
        ('checkpoint_gobierno',true),
        ('fuente_corporativa_contexto_actor_v1',false),
        ('clave_capacidad_version',false),('puntero_clave_emision',false),
        ('configuracion_confianza_version',false),
        ('puntero_configuracion_actual',false),
        ('raiz_confianza_version',false),('configuracion_raiz',false)
    )
    SELECT pg_catalog.bool_and(EXISTS (
        SELECT 1 FROM pg_catalog.pg_locks AS l
         WHERE l.pid=pg_catalog.pg_backend_pid() AND l.granted
           AND l.relation=pg_catalog.to_regclass(
               'vec_autorizacion_atestada_v3.'||r.nombre)
           AND l.mode='RowShareLock'
    ) = CASE WHEN r.checkpoint THEN p_checkpoint_esperado ELSE true END)
      FROM relaciones AS r
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.perfil_c1_prueba(pg_catalog.int4),
    vec_autorizacion_atestada_v3.manifiesto_c1_prueba(
        pg_catalog.int4,pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.capacidad_c1_prueba(
        pg_catalog.int4,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
        pg_catalog.bytea,pg_catalog.bytea,pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.material_c1_prueba(
        pg_catalog.int4,pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
        pg_catalog.bytea,pg_catalog.text,pg_catalog.text,pg_catalog.bytea),
    vec_autorizacion_atestada_v3.invocar_c1_prueba(
        pg_catalog.int4,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
        pg_catalog.bytea,pg_catalog.bytea),
    vec_autorizacion_atestada_v3.locks_c1_prueba(pg_catalog.bool)
FROM PUBLIC;
DO $fixture_c1$
DECLARE
    v_ahora pg_catalog.timestamptz := pg_catalog.clock_timestamp();
    v_secreto pg_catalog.bytea:=pg_catalog.sha256(
        pg_catalog.convert_to('secreto-hmac-f0-c1-1','UTF8'));
    v_spki pg_catalog.bytea:=pg_catalog.decode('302a300506032b6570032100'||
        pg_catalog.repeat('a1',32),'hex');
    v_perfil record;
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
    VALUES ('configuracion:f0-c1',700001,pg_catalog.repeat('6',64),
        v_ahora-pg_catalog.make_interval(hours=>1),v_ahora+pg_catalog.make_interval(hours=>1),
        'acto:f0-c1-config',v_ahora);
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
    VALUES ('raiz:f0-c1',700001,v_spki,
        pg_catalog.encode(pg_catalog.sha256(v_spki),'hex'),
        v_ahora-pg_catalog.make_interval(hours=>1),v_ahora+pg_catalog.make_interval(hours=>1),
        'VEC-AD-3-COSE-EDDSA-1','vec-diputacion/pruebas/f0/c1',
        'acto:f0-c1-raiz',v_ahora);
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
    VALUES ('configuracion:f0-c1','raiz:f0-c1',700001);
    INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
    VALUES (700001,'configuracion:f0-c1',
        v_ahora-pg_catalog.make_interval(hours=>1),
        'acto:f0-c1-puntero-config',v_ahora);
    FOR v_perfil IN
        SELECT i.indice,p.* FROM pg_catalog.generate_series(1,4) AS i(indice)
        CROSS JOIN LATERAL vec_autorizacion_atestada_v3.perfil_c1_prueba(i.indice) AS p
    LOOP
        v_secreto:=pg_catalog.sha256(pg_catalog.convert_to(
            'secreto-hmac-f0-c1-'||v_perfil.indice,'UTF8'));
        INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
        VALUES ('clave:f0-c1-'||v_perfil.indice,700000+v_perfil.indice,
            700000+v_perfil.indice,pg_catalog.repeat('5',64),v_secreto,
            pg_catalog.encode(pg_catalog.sha256(v_secreto),'hex'),
            'emisor:f0-c1-'||v_perfil.indice,v_perfil.audiencia,
            v_ahora-pg_catalog.make_interval(hours=>1),v_ahora+pg_catalog.make_interval(hours=>1),
            'acto:f0-c1-clave-'||v_perfil.indice,v_ahora);
        INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
        VALUES (700000+v_perfil.indice,
            'clave:f0-c1-'||v_perfil.indice,700000+v_perfil.indice,
            v_ahora-pg_catalog.make_interval(hours=>1),
            'acto:f0-c1-puntero-clave-'||v_perfil.indice,v_ahora);
    END LOOP;
    ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
        DISABLE TRIGGER f0_checkpoint_antes;
    FOR v_perfil IN
        SELECT i.indice,p.* FROM pg_catalog.generate_series(1,4) AS i(indice)
        CROSS JOIN LATERAL vec_autorizacion_atestada_v3.perfil_c1_prueba(i.indice) AS p
    LOOP
        INSERT INTO vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1 VALUES (
            'fuente:f0-c1-sintetica',700001,v_perfil.audiencia,
            v_perfil.accion,v_perfil.tipo,'clave:f0-c1-'||v_perfil.indice,
            700000+v_perfil.indice,700000+v_perfil.indice,
            pg_catalog.repeat('5',64),'emisor:f0-c1-'||v_perfil.indice,
            'configuracion:f0-c1',700001,pg_catalog.repeat('6',64),
            'raiz:f0-c1',700001,
            pg_catalog.encode(pg_catalog.sha256(v_spki),'hex'),
            'vec-diputacion/pruebas/f0/c1','VEC-AD-3-COSE-EDDSA-1',
            v_ahora-pg_catalog.make_interval(hours=>1),v_ahora+pg_catalog.make_interval(hours=>1),
            'acto:f0-c1-fuente-'||v_perfil.indice,v_ahora);
    END LOOP;
    ALTER TABLE vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
        ENABLE TRIGGER f0_checkpoint_antes;
    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision=revision+4,
           configuracion_secuencia_minima=GREATEST(configuracion_secuencia_minima,700001),
           raiz_version_minima=GREATEST(raiz_version_minima,700001),
           actualizada_en=v_ahora WHERE control_id;
END $fixture_c1$;
DO $nominales_y_locks_c1$
DECLARE
    i pg_catalog.int4;
    v_emitida pg_catalog.timestamptz;
    v_salida pg_catalog.jsonb;
    m record;
BEGIN
    FOR i IN 1..4 LOOP
        v_emitida:=pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.05);
        SELECT * INTO STRICT m
          FROM vec_autorizacion_atestada_v3.material_c1_prueba(i,v_emitida);
        BEGIN
            v_salida:=vec_autorizacion_atestada_v3.invocar_c1_prueba(
                i,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
            IF v_salida->>'capacidad_ref' <> 'cfc_'||
                   pg_catalog.encode(pg_catalog.sha256(m.capacidad),'hex')
               OR v_salida->>'fuente_ref' <> 'fuente:f0-c1-sintetica'
               OR v_salida->>'evento_fuente_ref' <> 'evento:f0-c1-'||i
               OR v_salida ? 'emitida_en' OR v_salida ? 'expira_en'
               OR (SELECT pg_catalog.count(*)
                     FROM pg_catalog.jsonb_object_keys(v_salida)) <> 12
               OR vec_autorizacion_atestada_v3.locks_c1_prueba(true)
                  IS NOT TRUE THEN
                RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: nominal, salida o locks incorrectos';
            END IF;
            RAISE SQLSTATE 'ZC101';
        EXCEPTION WHEN SQLSTATE 'ZC101' THEN NULL;
        END;
        IF EXISTS (SELECT 1 FROM pg_catalog.pg_locks AS l
             WHERE l.pid=pg_catalog.pg_backend_pid() AND l.granted
               AND l.relation='vec_autorizacion_atestada_v3.checkpoint_gobierno'::pg_catalog.regclass
               AND l.mode='RowShareLock') THEN
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: subtransaccion no libero lock de checkpoint';
        END IF;
    END LOOP;
END $nominales_y_locks_c1$;
DO $punteros_checkpoint_y_rotacion_c1$
DECLARE
    v_emitida pg_catalog.timestamptz;
    v_antes pg_catalog.numeric;
    v_despues pg_catalog.numeric;
    v_nuevo_secreto pg_catalog.bytea:=pg_catalog.sha256(pg_catalog.convert_to('secreto-hmac-f0-c1-rotada','UTF8'));
    c record;
    m record;
BEGIN
    FOR c IN SELECT * FROM (VALUES (600090,1,700001),(700090,2,700090)) AS x(secuencia,incremento,minimo) LOOP
        BEGIN
            SELECT revision INTO STRICT v_antes FROM vec_autorizacion_atestada_v3.checkpoint_gobierno;
            INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
            VALUES ('configuracion:f0-c1-causal',c.secuencia,pg_catalog.repeat('d',64),
                clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour',
                'acto:f0-c1-config-causal',clock_timestamp());
            INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
            VALUES (700090,'configuracion:f0-c1-causal',clock_timestamp(),
                'acto:f0-c1-puntero-config-causal',clock_timestamp());
            SELECT revision INTO STRICT v_despues FROM vec_autorizacion_atestada_v3.checkpoint_gobierno;
            IF v_despues<>v_antes+c.incremento OR (SELECT configuracion_secuencia_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno)<>c.minimo THEN
                RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: avance causal de configuracion incorrecto';
            END IF;
            RAISE SQLSTATE 'ZC101';
        EXCEPTION WHEN SQLSTATE 'ZC101' THEN NULL;
        END;
    END LOOP;
    FOR c IN SELECT * FROM (VALUES (-0.01::pg_catalog.float8,false),(0::pg_catalog.float8,false),
        (0.01::pg_catalog.float8,true)) AS x(desfase,acepta) LOOP
        v_emitida:=pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.05);
        SELECT * INTO STRICT m
          FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
        BEGIN
            SELECT revision INTO STRICT v_antes FROM vec_autorizacion_atestada_v3.checkpoint_gobierno;
            INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
            VALUES ('clave:f0-c1-rotada',700090,700090,pg_catalog.repeat('5',64),
                v_nuevo_secreto,pg_catalog.encode(pg_catalog.sha256(v_nuevo_secreto),'hex'),
                'emisor:f0-c1-rotada',
                'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
                clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour',
                'acto:f0-c1-clave-rotada',clock_timestamp());
            INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
            VALUES (700090,'clave:f0-c1-rotada',700090,
                v_emitida+pg_catalog.make_interval(secs=>c.desfase),
                'acto:f0-c1-puntero-clave-rotada',clock_timestamp());
            SELECT revision INTO STRICT v_despues FROM vec_autorizacion_atestada_v3.checkpoint_gobierno;
            IF v_despues<>v_antes+1 THEN
                RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: puntero de clave no avanzo una revision';
            END IF;
            IF c.acepta THEN
                PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                    1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
                RAISE SQLSTATE 'ZC101';
            END IF;
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: clave anterior aceptada tras rotacion efectiva';
        EXCEPTION
            WHEN SQLSTATE 'ZC101' THEN NULL;
            WHEN insufficient_privilege THEN
                IF c.acepta THEN RAISE; END IF;
        END;
    END LOOP;
    IF NOT EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision AS p
        JOIN vec_autorizacion_atestada_v3.clave_capacidad_version AS k
          ON (k.clave_id,k.version)=(p.clave_id,p.version)
        WHERE k.audiencia_consumo<>'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
          AND p.orden>700001 AND p.establecida_en<=v_emitida
    ) THEN
        RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: no se construyo rotacion de otra audiencia';
    END IF;
END $punteros_checkpoint_y_rotacion_c1$;
DO $cruces_y_artefactos_c1$
DECLARE
    v_emitida pg_catalog.timestamptz :=
        pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.05);
    v_alterada pg_catalog.bytea;
    v_manifiesto_alterado pg_catalog.bytea;
    c record;
    m record;
BEGIN
    SELECT * INTO STRICT m
      FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
    FOR c IN SELECT * FROM (VALUES
        ('clave_id','"clave:f0-c1-ausente"'),('clave_version','700099'),
        ('revision_gobierno','700099'),
        ('huella_gobierno_sha256','"'||pg_catalog.repeat('e',64)||'"'),
        ('emisor_id','"emisor:f0-c1-hostil"'),
        ('configuracion_revision','"configuracion:f0-c1-hostil"'),
        ('configuracion_secuencia','700099'),
        ('huella_configuracion_sha256','"'||pg_catalog.repeat('d',64)||'"'),
        ('raiz_clave_id','"raiz:f0-c1-hostil"'),('raiz_version','700099'),
        ('huella_raiz_spki_sha256','"'||pg_catalog.repeat('e',64)||'"'),
        ('audiencia_despliegue','"vec-diputacion/pruebas/f0/hostil"')
    ) AS x(campo,valor_json) LOOP
        v_alterada:=vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
            m.capacidad,c.campo,c.valor_json,m.secreto);
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,v_alterada,m.manifiesto,m.sobre,m.evidencia,m.spki);
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: binding de gobierno hostil aceptado';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
    END LOOP;
    v_manifiesto_alterado:=pg_catalog.convert_to(pg_catalog.replace(
        pg_catalog.convert_from(m.manifiesto,'UTF8'),
        '"fuente_version":700001','"fuente_version":700099'),'UTF8');
    v_alterada:=vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
        m.capacidad,'fuente_version','700099',m.secreto);
    v_alterada:=vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
        v_alterada,'huella_manifiesto_fuente_sha256','"'||
        pg_catalog.encode(pg_catalog.sha256(v_manifiesto_alterado),'hex')||'"',
        m.secreto);
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
            1,v_alterada,v_manifiesto_alterado,m.sobre,m.evidencia,m.spki);
        RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: version de fuente ausente aceptada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    FOR c IN SELECT * FROM (VALUES
        ('manifiesto',pg_catalog.convert_to(pg_catalog.replace(
            pg_catalog.convert_from(m.manifiesto,'UTF8'),'evento:f0-c1-1',
            'evento:f0-c1-x'),'UTF8'),m.sobre,m.evidencia,m.spki),
        ('COSE',m.manifiesto,pg_catalog.set_byte(m.sobre,127,0),m.evidencia,m.spki),
        ('evidencia',m.manifiesto,m.sobre,
            pg_catalog.set_byte(m.evidencia,31,0),m.spki),
        ('SPKI',m.manifiesto,m.sobre,m.evidencia,pg_catalog.set_byte(m.spki,43,0))
    ) AS x(nombre,manifiesto,sobre,evidencia,spki) LOOP
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,m.capacidad,c.manifiesto,c.sobre,c.evidencia,c.spki);
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: artefacto cruzado aceptado';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
    END LOOP;
    v_alterada:=vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
        m.capacidad,'suite','"VEC-AD-3-COSE-EDDSA-2"',m.secreto);
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
            1,v_alterada,m.manifiesto,m.sobre,m.evidencia,m.spki);
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C1: suite hostil aceptada';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
END $cruces_y_artefactos_c1$;
DO $vigencias_checkpoint_y_limites_c1$
DECLARE
    v_emitida pg_catalog.timestamptz :=
        pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.05);
    c record;
    m record;
BEGIN
    SELECT * INTO STRICT m
      FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
    BEGIN
        ALTER TABLE vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
            DISABLE TRIGGER f0_historia_inmutable;
        DELETE FROM vec_autorizacion_atestada_v3
            .fuente_corporativa_contexto_actor_v1
         WHERE fuente_ref='fuente:f0-c1-sintetica';
        PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
            1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C1: fuente ausente aceptada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    FOR c IN SELECT * FROM (VALUES
        ('fuente_corporativa_contexto_actor_v1','f0_historia_inmutable',
         'valida_hasta','fuente_ref=''fuente:f0-c1-sintetica'''),
        ('clave_capacidad_version','inmutable','valida_hasta',
         'clave_id=''clave:f0-c1-1'''),
        ('configuracion_confianza_version','inmutable','expira_en',
         'revision=''configuracion:f0-c1'''),
        ('raiz_confianza_version','inmutable','valida_hasta',
         'clave_id=''raiz:f0-c1''')
    ) AS x(tabla,trigger_nombre,columna,predicado) LOOP
        BEGIN
            EXECUTE pg_catalog.format(
                'ALTER TABLE vec_autorizacion_atestada_v3.%I DISABLE TRIGGER %I',
                c.tabla,c.trigger_nombre);
            EXECUTE pg_catalog.format(
                'UPDATE vec_autorizacion_atestada_v3.%I SET %I=$1 WHERE %s',
                c.tabla,c.columna,c.predicado)
                USING pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>1);
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: ventana agotada aceptada';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
    END LOOP;
    v_emitida:=pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>6);
    SELECT * INTO STRICT m
      FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
            1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C1: capacidad caducada aceptada';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    v_emitida:=pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.05);
    SELECT * INTO STRICT m
      FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
    FOR c IN SELECT * FROM (VALUES
        ('configuracion_secuencia_minima'),('raiz_version_minima')
    ) AS x(columna) LOOP
        BEGIN
            EXECUTE pg_catalog.format(
                'UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET %I=700002 WHERE control_id',
                c.columna);
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C1: minimo de checkpoint hostil aceptado';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
    END LOOP;
    BEGIN
        DELETE FROM vec_autorizacion_atestada_v3.checkpoint_gobierno;
        PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
            1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C1: checkpoint ausente aceptado';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    FOR c IN SELECT * FROM (VALUES
        ('TimeZone','Europe/Madrid'),('statement_timeout','10001ms'),
        ('transaction_timeout','15001ms'),
        ('idle_in_transaction_session_timeout','15001ms')
    ) AS x(ajuste,valor) LOOP
        BEGIN
            PERFORM pg_catalog.set_config(c.ajuste,c.valor,true);
            PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C1: limite de sesion hostil aceptado';
        EXCEPTION WHEN insufficient_privilege THEN NULL;
        END;
    END LOOP;
END $vigencias_checkpoint_y_limites_c1$;
DO $mutantes_temporales_y_locks_c1$
DECLARE
    v_funcion pg_catalog.regprocedure:=
        'vec_autorizacion_atestada_v3.acreditar_material_fuente_corporativa_contexto_actor_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'::regprocedure;
    v_original pg_catalog.text := pg_catalog.pg_get_functiondef(v_funcion);
    v_pausada pg_catalog.text;
    v_mutante pg_catalog.text;
    v_marca pg_catalog.text := E'    -- El segundo reloj y las cuatro revocaciones cierran la ventana hasta C2.\n    v_ahora := pg_catalog.clock_timestamp();';
    v_pos pg_catalog.int4;
    v_prefijo pg_catalog.text;
    v_sufijo pg_catalog.text;
    v_emitida pg_catalog.timestamptz;
    v_futura pg_catalog.timestamptz;
    v_correcta pg_catalog.bool;
    c record;
    m record;
BEGIN
    IF pg_catalog.length(v_original)-pg_catalog.length(
           pg_catalog.replace(v_original,v_marca,''))<>pg_catalog.length(v_marca) THEN
        RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: marcador temporal no unico';
    END IF;
    v_pos:=pg_catalog.strpos(v_original,v_marca);
    v_prefijo:=pg_catalog.left(v_original,v_pos-1);
    v_sufijo:=pg_catalog.substr(v_original,v_pos+pg_catalog.length(v_marca));
    v_pausada:=pg_catalog.replace(v_original,v_marca,
        v_marca||E'\n    PERFORM pg_catalog.pg_sleep(0.15);\n    v_ahora := pg_catalog.clock_timestamp();');
    FOR c IN SELECT * FROM (VALUES
        ('puntero','WHERE p.orden > v_puntero_configuracion.orden'),('fuente','WHERE r.fuente_ref = v_fuente.fuente_ref'),
        ('clave','WHERE r.clave_id = v_clave.clave_id'),('configuracion','WHERE r.configuracion_revision = v_configuracion.revision'),
        ('raiz','WHERE r.raiz_clave_id = v_raiz.clave_id')
    ) AS x(tipo,aguja) LOOP
        IF pg_catalog.length(v_prefijo)-pg_catalog.length(
               pg_catalog.replace(v_prefijo,c.aguja,''))<>pg_catalog.length(c.aguja)
           OR pg_catalog.length(v_sufijo)-pg_catalog.length(
               pg_catalog.replace(v_sufijo,c.aguja,''))<>pg_catalog.length(c.aguja) THEN
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: revalidacion temporal no aparece dos veces';
        END IF;
        v_pos:=pg_catalog.strpos(v_pausada,v_marca);
        v_mutante:=pg_catalog.left(v_pausada,v_pos-1)||pg_catalog.replace(
            pg_catalog.substr(v_pausada,v_pos),c.aguja,
            pg_catalog.replace(c.aguja,'WHERE ','WHERE false AND '));
        IF v_mutante=v_pausada THEN
            RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: mutante temporal no construido';
        END IF;
        FOR v_correcta IN SELECT false UNION ALL SELECT true LOOP
            IF v_correcta THEN EXECUTE v_pausada; ELSE EXECUTE v_mutante; END IF;
            v_emitida:=pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.02);
            v_futura:=pg_catalog.clock_timestamp()+pg_catalog.make_interval(secs=>0.08);
            SELECT * INTO STRICT m
              FROM vec_autorizacion_atestada_v3.material_c1_prueba(1,v_emitida);
            BEGIN
                IF c.tipo='puntero' THEN
                    INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
                    VALUES ('configuracion:f0-c1-futura',600001,pg_catalog.repeat('d',64),
                        clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour',
                        'acto:f0-c1-config-futura',clock_timestamp());
                    INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
                    VALUES (700099,'configuracion:f0-c1-futura',v_futura,'acto:f0-c1-puntero-futuro',clock_timestamp());
                ELSIF c.tipo='fuente' THEN
                    INSERT INTO vec_autorizacion_atestada_v3
                        .revocacion_fuente_corporativa_contexto_actor_v1 VALUES (
                        'fuente:f0-c1-sintetica',700001,
                        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
                        v_futura,'motivo:f0-c1','acto:f0-c1-rev-fuente',clock_timestamp());
                ELSIF c.tipo='clave' THEN
                    INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
                    VALUES ('clave:f0-c1-1',700001,v_futura,'motivo:f0-c1','acto:f0-c1-rev-clave',clock_timestamp());
                ELSIF c.tipo='configuracion' THEN
                    INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion
                    VALUES ('configuracion:f0-c1',v_futura,'motivo:f0-c1','acto:f0-c1-rev-config',clock_timestamp());
                ELSE
                    INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz
                    VALUES ('raiz:f0-c1',700001,v_futura,'motivo:f0-c1','acto:f0-c1-rev-raiz',clock_timestamp());
                END IF;
                PERFORM vec_autorizacion_atestada_v3.invocar_c1_prueba(
                    1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki);
                IF v_correcta THEN
                    RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: revalidacion temporal no rechazo';
                END IF;
                RAISE SQLSTATE 'ZC101';
            EXCEPTION
                WHEN SQLSTATE 'ZC101' THEN NULL;
                WHEN insufficient_privilege THEN
                    IF NOT v_correcta THEN RAISE; END IF;
            END;
        END LOOP;
    END LOOP;
    EXECUTE v_original;
    v_mutante:=pg_catalog.replace(v_original,'     FOR UPDATE;','     ;');
    IF v_mutante=v_original THEN
        RAISE EXCEPTION USING ERRCODE='XX000', MESSAGE='C1: mutante de checkpoint no construido';
    END IF;
    EXECUTE v_mutante;
    BEGIN
        IF vec_autorizacion_atestada_v3.invocar_c1_prueba(
               1,m.capacidad,m.manifiesto,m.sobre,m.evidencia,m.spki) IS NULL
           OR vec_autorizacion_atestada_v3.locks_c1_prueba(false)
              IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE='XX000',
                MESSAGE='C1: mutante de checkpoint no quedo expuesto';
        END IF;
        RAISE SQLSTATE 'ZC101';
    EXCEPTION WHEN SQLSTATE 'ZC101' THEN NULL;
    END;
    EXECUTE v_original;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_locks AS l
         WHERE l.pid=pg_catalog.pg_backend_pid() AND l.granted
           AND l.relation='vec_autorizacion_atestada_v3.checkpoint_gobierno'::pg_catalog.regclass
           AND l.mode='RowShareLock') THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C1: locks mutantes no fueron liberados';
    END IF;
END $mutantes_temporales_y_locks_c1$;
DO $forma_privada_c1$
DECLARE
    f pg_catalog.regprocedure:='vec_autorizacion_atestada_v3.acreditar_material_fuente_corporativa_contexto_actor_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'::regprocedure;
    g pg_catalog.regprocedure:='vec_autorizacion_atestada_v3.avanzar_checkpoint_puntero_fuente_corporativa_v1()'::pg_catalog.regprocedure;
    o pg_catalog.oid := (SELECT r.oid FROM pg_catalog.pg_roles AS r
        WHERE r.rolname='vec_autorizacion_atestada_v3_propietario'
          AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
          AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls);
BEGIN
    -- Aislamiento y read-only quedan ligados por esta huella; sus negativos
    -- conductuales requieren sesiones propias y pertenecen a Q1/T2.
    IF o IS NULL OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
         WHERE p.oid=f AND p.proowner=o AND p.prokind='f'
           AND p.prolang=(SELECT l.oid FROM pg_catalog.pg_language AS l WHERE l.lanname='plpgsql')
           AND p.prorettype='record'::pg_catalog.regtype AND p.proretset
           AND p.pronargs=11 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
           AND p.provariadic=0 AND p.provolatile='v' AND NOT p.proisstrict
           AND p.prosecdef AND NOT p.proleakproof AND p.proparallel='u'
           AND p.procost=100 AND p.prorows=1000 AND p.prosupport=0
           AND p.protrftypes IS NULL AND p.probin IS NULL AND p.prosqlbody IS NULL
           AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
           AND p.proargtypes=ARRAY[
               'text'::regtype,'text'::regtype,'text'::regtype,'text'::regtype,
               'text'::regtype,'text'::regtype,'bytea'::regtype,'bytea'::regtype,
               'bytea'::regtype,
               'bytea'::regtype,'bytea'::regtype]::pg_catalog.oidvector
           AND p.proallargtypes=ARRAY[
               'text'::regtype,'text'::regtype,'text'::regtype,'text'::regtype,'text'::regtype,
               'text'::regtype,'bytea'::regtype,'bytea'::regtype,'bytea'::regtype,
               'bytea'::regtype,'bytea'::regtype,'text'::regtype,'text'::regtype,
               'numeric'::regtype,'text'::regtype,'text'::regtype,
               'timestamptz'::regtype,'text'::regtype,'text'::regtype,
               'text'::regtype,'text'::regtype,'text'::regtype,
               'timestamptz'::regtype]::pg_catalog.oid[]
           AND p.proargmodes=ARRAY[
               'i','i','i','i','i','i','i','i','i','i','i',
               't','t','t','t','t','t','t','t','t','t','t','t']::"char"[]
           AND p.proargnames=ARRAY[
               'p_audiencia_consumo_esperada','p_accion_esperada','p_tipo_efecto_esperado',
               'p_operacion_ref_esperada',
               'p_efecto_ref_esperada','p_huella_efecto_sha256_esperada',
               'p_capacidad_canonica','p_manifiesto_fuente_canonico','p_sobre_cose_sign1',
               'p_evidencia_verificacion','p_raiz_publica_spki','capacidad_ref',
               'fuente_ref','fuente_version',
               'evento_fuente_ref','huella_evento_fuente_sha256','evento_fuente_emitido_en',
               'huella_manifiesto_fuente_sha256','operacion_ref','efecto_ref',
               'huella_efecto_sha256','nonce',
               'acreditada_en']::pg_catalog.text[]
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')=
               'f4da25b409d42f6ea50bb6c97358e224dc021b397462b22672dc810eb6c32f1f'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS p
         WHERE p.oid=g AND p.proowner=o AND p.prokind='f'
           AND p.prolang=(SELECT l.oid FROM pg_catalog.pg_language AS l WHERE l.lanname='plpgsql')
           AND p.prorettype='trigger'::pg_catalog.regtype AND NOT p.proretset
           AND p.pronargs=0 AND p.pronargdefaults=0 AND p.proargdefaults IS NULL
           AND p.provariadic=0 AND p.provolatile='v' AND p.proisstrict
           AND p.prosecdef AND NOT p.proleakproof AND p.proparallel='u'
           AND p.procost=100 AND p.prorows=0 AND p.prosupport=0
           AND p.protrftypes IS NULL AND p.probin IS NULL AND p.prosqlbody IS NULL
           AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
           AND p.proargtypes=''::pg_catalog.oidvector
           AND p.proallargtypes IS NULL AND p.proargmodes IS NULL
           AND p.proargnames IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')=
               '0d2a6ec8b7288b61e3a85a7da4d3ad490920d11c8a80d052ceb93aa0879b13ca'
    ) OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
          WHERE p.pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace
            AND p.proname IN ('acreditar_material_fuente_corporativa_contexto_actor_v1',
                'avanzar_checkpoint_puntero_fuente_corporativa_v1'))<>2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p CROSS JOIN LATERAL
            pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',o))) AS a
           WHERE a.grantor=o AND a.grantee=o AND a.privilege_type='EXECUTE'
             AND NOT a.is_grantable AND p.oid IN (f,g))<>2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p CROSS JOIN LATERAL
            pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',o))) AS a
           WHERE p.oid IN (f,g))<>2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger AS t
           WHERE t.tgname='f0_checkpoint_puntero_antes' AND t.tgfoid=g
             AND t.tgrelid IN (
                'vec_autorizacion_atestada_v3.puntero_clave_emision'::regclass,
                'vec_autorizacion_atestada_v3.puntero_configuracion_actual'::regclass)
             AND NOT t.tgisinternal AND t.tgenabled='O'
             AND t.tgtype=7 AND t.tgnargs=0 AND t.tgattr=''::int2vector
             AND t.tgargs='\x'::bytea AND t.tgqual IS NULL
             AND t.tgoldtable IS NULL AND t.tgnewtable IS NULL
             AND NOT t.tgdeferrable AND NOT t.tginitdeferred
             AND t.tgconstraint=0 AND t.tgconstrrelid=0 AND t.tgparentid=0)<>2
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_depend AS d
           JOIN pg_catalog.pg_trigger AS t
             ON d.classid='pg_trigger'::regclass AND d.objid=t.oid
          WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=g
            AND d.deptype='n' AND t.tgname='f0_checkpoint_puntero_antes')<>2 THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C1: forma pg_proc o ACL privada incorrecta';
    END IF;
END $forma_privada_c1$;
DROP FUNCTION vec_autorizacion_atestada_v3.locks_c1_prueba(pg_catalog.bool);
DROP FUNCTION vec_autorizacion_atestada_v3.invocar_c1_prueba(
    pg_catalog.int4,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
    pg_catalog.bytea,pg_catalog.bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.mutar_remac_c1_prueba(
    pg_catalog.bytea,pg_catalog.text,pg_catalog.text,pg_catalog.bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.material_c1_prueba(
    pg_catalog.int4,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_c1_prueba(
    pg_catalog.int4,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
    pg_catalog.bytea,pg_catalog.bytea,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.manifiesto_c1_prueba(
    pg_catalog.int4,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.perfil_c1_prueba(pg_catalog.int4);
