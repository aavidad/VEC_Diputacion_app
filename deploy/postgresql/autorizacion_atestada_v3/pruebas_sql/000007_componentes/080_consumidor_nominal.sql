-- Prueba focal F0-C2: ABI, R0, consumo nominal, replay y colisiones.

CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_forma_c2_prueba()
RETURNS pg_catalog.bool
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    WITH propietario AS (
        SELECT r.oid
          FROM pg_catalog.pg_roles AS r
         WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin AND NOT r.rolsuper
           AND NOT r.rolcreatedb AND NOT r.rolcreaterole
           AND NOT r.rolreplication AND NOT r.rolbypassrls
    ), candidata AS (
        SELECT p.*
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
          JOIN propietario AS o ON o.oid = p.proowner
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.proname =
               'consumir_fuente_corporativa_contexto_actor_v1_atestada'
           AND pg_catalog.oidvectortypes(p.proargtypes) =
               'text, text, text, text, text, text, bytea, bytea, bytea, bytea, bytea'
           AND p.prokind = 'f' AND p.prorettype = 'record'::pg_catalog.regtype
           AND p.pronargs = 11 AND p.pronargdefaults = 0
           AND p.proargdefaults IS NULL AND p.provariadic = 0
           AND p.proretset AND p.provolatile = 'v' AND NOT p.proisstrict
           AND p.prosecdef AND NOT p.proleakproof AND p.proparallel = 'u'
           AND p.procost = 100 AND p.prorows = 1000
           AND p.prosupport = 0 AND p.protrftypes IS NULL
           AND p.probin IS NULL AND p.prosqlbody IS NULL
           AND p.proconfig = ARRAY[
               'search_path=pg_catalog', 'lock_timeout=2s'
           ]::pg_catalog.text[]
           AND p.prolang = (SELECT l.oid FROM pg_catalog.pg_language AS l
                             WHERE l.lanname = 'plpgsql')
           AND p.proargnames = ARRAY[
             'p_audiencia_consumo_esperada','p_accion_esperada',
             'p_tipo_efecto_esperado','p_operacion_ref_esperada',
             'p_efecto_ref_esperada','p_huella_efecto_sha256_esperada',
             'p_capacidad_canonica','p_manifiesto_fuente_canonico',
             'p_sobre_cose_sign1','p_evidencia_verificacion',
             'p_raiz_publica_spki','capacidad_ref','fuente_ref',
             'fuente_version','evento_fuente_ref',
             'huella_evento_fuente_sha256',
             'huella_manifiesto_fuente_sha256','operacion_ref','efecto_ref',
             'huella_efecto_sha256','consumo_huella_sha256','consumida_en',
             'consumo_nuevo'
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
           AND (SELECT pg_catalog.count(*) FROM pg_catalog.aclexplode(
               COALESCE(p.proacl, pg_catalog.acldefault('f',p.proowner)))) = 1
           AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(
               COALESCE(p.proacl, pg_catalog.acldefault('f',p.proowner))) AS a
                WHERE a.grantor=o.oid AND a.grantee=o.oid
                  AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    )
    SELECT (SELECT pg_catalog.count(*) FROM propietario)=1
       AND (SELECT pg_catalog.count(*) FROM candidata)=1
       AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid=p.pronamespace
            WHERE n.nspname='vec_autorizacion_atestada_v3'
              AND p.proname=
                'consumir_fuente_corporativa_contexto_actor_v1_atestada')=1
       AND NOT pg_catalog.has_function_privilege(
          'vec_contexto_actor_v1_propietario',
          'vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure,
          'EXECUTE')
       AND NOT pg_catalog.has_schema_privilege(
          'vec_contexto_actor_v1_propietario',
          'vec_autorizacion_atestada_v3','USAGE')
       AND (SELECT pg_catalog.strpos(pg_catalog.lower(p.prosrc),
                    'on conflict')=0
                  AND pg_catalog.strpos(pg_catalog.lower(p.prosrc),'limit ')=0
                  AND pg_catalog.strpos(p.prosrc,E'\nEXCEPTION\n')=0
                  AND pg_catalog.strpos(pg_catalog.lower(p.prosrc),' update ')=0
                  AND pg_catalog.strpos(pg_catalog.lower(p.prosrc),' delete ')=0
                  AND pg_catalog.strpos(pg_catalog.lower(p.prosrc),'truncate')=0
                  AND pg_catalog.strpos(p.prosrc,'SESSION_USER')>0
                  AND pg_catalog.strpos(p.prosrc,'pg_auth_members')>0
                  AND pg_catalog.strpos(p.prosrc,'pg_db_role_setting')>0
                  AND pg_catalog.strpos(p.prosrc,'pg_has_role')>0
                  AND pg_catalog.strpos(p.prosrc,'SELECT DISTINCT bloqueos.clave')>0
                  AND pg_catalog.strpos(p.prosrc,'ORDER BY bloqueos.clave')>0
                  AND pg_catalog.strpos(p.prosrc,'pg_advisory_xact_lock')>0
                  AND pg_catalog.strpos(p.prosrc,
                    'canon_y_huella_consumo_fuente_corporativa_v1')>0
                  AND pg_catalog.strpos(p.prosrc,
                    'acreditar_material_fuente_corporativa_contexto_actor_v1')>0
               FROM candidata AS p)
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_c2_prueba() FROM PUBLIC;

DO $forma_c2$
BEGIN
    IF vec_autorizacion_atestada_v3.acreditar_forma_c2_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='XX000',
            MESSAGE='C2: ABI, propietario, ACL o cuerpo invalidos';
    END IF;
END
$forma_c2$;

CREATE TEMP VIEW dependencia_c2_prueba AS
SELECT * FROM vec_autorizacion_atestada_v3
  .consumir_fuente_corporativa_contexto_actor_v1_atestada(
    NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,
    NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,
    NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,
    NULL::pg_catalog.bytea,NULL::pg_catalog.bytea);
DO $reentrada_y_down_seguro$
BEGIN
 BEGIN
  EXECUTE 'CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea) RETURNS integer LANGUAGE sql AS ''SELECT 1''';
  RAISE SQLSTATE 'ZC201';
 EXCEPTION
  WHEN duplicate_function THEN NULL;
  WHEN SQLSTATE 'ZC201' THEN
   RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: reentrada sustituyo evidencia';
 END;
 BEGIN
  EXECUTE 'DROP FUNCTION vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)';
  RAISE SQLSTATE 'ZC202';
 EXCEPTION
  WHEN dependent_objects_still_exist THEN NULL;
  WHEN SQLSTATE 'ZC202' THEN
   RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: down retiro evidencia viva';
 END;
 IF vec_autorizacion_atestada_v3.acreditar_forma_c2_prueba() IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: evidencia no conservada';
 END IF;
END
$reentrada_y_down_seguro$;
DROP VIEW pg_temp.dependencia_c2_prueba;

CREATE FUNCTION vec_autorizacion_atestada_v3.manifiesto_c2_prueba(
    p_emitida pg_catalog.timestamptz)
RETURNS pg_catalog.bytea
LANGUAGE sql VOLATILE SET search_path=pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
      '{"esquema":"vec.contexto-actor.fuente-corporativa.manifiesto.v1"' ||
      ',"version":1,"fuente_ref":"fuente:f0-c2-sintetica"' ||
      ',"fuente_version":800001,"evento_fuente_ref":"evento:f0-c2-0001"' ||
      ',"huella_evento_fuente_sha256":"'||pg_catalog.repeat('1',64)||'"' ||
      ',"evento_fuente_emitido_en":'||vec_autorizacion_atestada_v3.texto_json_go(
        vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
          p_emitida-pg_catalog.make_interval(secs=>0.1))) ||
      ',"audiencia_consumo":' || vec_autorizacion_atestada_v3.texto_json_go(
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1') ||
      ',"accion":"contexto_actor.organizacion_corporativa.publicar"' ||
      ',"tipo_efecto":"organizacion_corporativa.alta"' ||
      ',"operacion_ref":"oca_f0_c2_00000000000000000001"' ||
      ',"efecto_ref":"efecto:f0-c2-0001"' ||
      ',"huella_efecto_sha256":"'||pg_catalog.repeat('4',64)||'"}','UTF8')
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.capacidad_c2_prueba(
    p_manifiesto pg_catalog.bytea,p_sobre pg_catalog.bytea,
    p_evidencia pg_catalog.bytea,p_spki pg_catalog.bytea,
    p_secreto pg_catalog.bytea,p_emitida pg_catalog.timestamptz)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog
AS $funcion$
DECLARE
    m pg_catalog.json:=pg_catalog.convert_from(p_manifiesto,'UTF8')::pg_catalog.json;
    c pg_catalog.bytea;
    h pg_catalog.text;
BEGIN
    c:=pg_catalog.convert_to(
      '{"esquema":"vec.contexto-actor.fuente-corporativa.capacidad.v1"' ||
      ',"version":1,"fuente_ref":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'fuente_ref') ||
      ',"fuente_version":800001,"evento_fuente_ref":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'evento_fuente_ref') ||
      ',"huella_evento_fuente_sha256":"'||pg_catalog.repeat('1',64)||'"' ||
      ',"evento_fuente_emitido_en":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'evento_fuente_emitido_en') ||
      ',"huella_manifiesto_fuente_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_manifiesto),'hex')||'"' ||
      ',"huella_sobre_cose_sign1_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_sobre),'hex')||'"' ||
      ',"huella_prueba_confianza_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_evidencia),'hex')||'"' ||
      ',"audiencia_consumo":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'audiencia_consumo') ||
      ',"accion":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'accion') ||
      ',"tipo_efecto":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'tipo_efecto') ||
      ',"operacion_ref":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'operacion_ref') ||
      ',"efecto_ref":'||vec_autorizacion_atestada_v3.texto_json_go(m->>'efecto_ref') ||
      ',"huella_efecto_sha256":"'||pg_catalog.repeat('4',64)||'"' ||
      ',"clave_id":"clave:f0-c2","clave_version":800001' ||
      ',"revision_gobierno":800001,"huella_gobierno_sha256":"'||pg_catalog.repeat('5',64)||'"' ||
      ',"emisor_id":"emisor:f0-c2","configuracion_revision":"configuracion:f0-c2"' ||
      ',"configuracion_secuencia":800001,"huella_configuracion_sha256":"'||pg_catalog.repeat('6',64)||'"' ||
      ',"raiz_clave_id":"raiz:f0-c2","raiz_version":800001' ||
      ',"huella_raiz_spki_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_spki),'hex')||'"' ||
      ',"audiencia_despliegue":"vec-diputacion/pruebas/f0/c2"' ||
      ',"suite":"VEC-AD-3-COSE-EDDSA-1","nonce":"'||pg_catalog.repeat('8',64)||'"' ||
      ',"emitida_en":'||vec_autorizacion_atestada_v3.texto_json_go(
        vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(p_emitida)) ||
      ',"expira_en":'||vec_autorizacion_atestada_v3.texto_json_go(
        vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
          p_emitida+pg_catalog.make_interval(secs=>5))) ||
      ',"mac_sha256":"'||pg_catalog.repeat('9',64)||'"}','UTF8');
    h:=pg_catalog.encode(public.hmac(
      vec_autorizacion_atestada_v3.preimagen_mac_fuente_corporativa_v1(c),
      p_secreto,'sha256'),'hex');
    RETURN pg_catalog.convert_to(pg_catalog.replace(
      pg_catalog.convert_from(c,'UTF8'),pg_catalog.repeat('9',64),h),'UTF8');
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.material_c2_prueba(
    p_emitida pg_catalog.timestamptz)
RETURNS TABLE(secreto pg_catalog.bytea,spki pg_catalog.bytea,
    sobre pg_catalog.bytea,evidencia pg_catalog.bytea,
    manifiesto pg_catalog.bytea,capacidad pg_catalog.bytea)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog
AS $funcion$
BEGIN
    secreto:=pg_catalog.sha256(pg_catalog.convert_to('secreto-hmac-f0-c2','UTF8'));
    spki:=pg_catalog.decode('302a300506032b6570032100'||pg_catalog.repeat('a2',32),'hex');
    sobre:=pg_catalog.decode(pg_catalog.repeat('b2',128),'hex');
    evidencia:=pg_catalog.decode(pg_catalog.repeat('c2',32),'hex');
    manifiesto:=vec_autorizacion_atestada_v3.manifiesto_c2_prueba(p_emitida);
    capacidad:=vec_autorizacion_atestada_v3.capacidad_c2_prueba(
      manifiesto,sobre,evidencia,spki,secreto,p_emitida);
    RETURN NEXT;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.manifiesto_c2_prueba(pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.capacidad_c2_prueba(
      pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
      pg_catalog.bytea,pg_catalog.timestamptz),
    vec_autorizacion_atestada_v3.material_c2_prueba(pg_catalog.timestamptz)
FROM PUBLIC;

DO $fixture_c2$
DECLARE
    ahora pg_catalog.timestamptz:=pg_catalog.clock_timestamp();
    secreto pg_catalog.bytea:=pg_catalog.sha256(
      pg_catalog.convert_to('secreto-hmac-f0-c2','UTF8'));
    spki pg_catalog.bytea:=pg_catalog.decode(
      '302a300506032b6570032100'||pg_catalog.repeat('a2',32),'hex');
BEGIN
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
    VALUES ('configuracion:f0-c2',800001,pg_catalog.repeat('6',64),
      ahora-pg_catalog.make_interval(hours=>1),ahora+pg_catalog.make_interval(hours=>1),
      'acto:f0-c2-config',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
    VALUES ('raiz:f0-c2',800001,spki,pg_catalog.encode(pg_catalog.sha256(spki),'hex'),
      ahora-pg_catalog.make_interval(hours=>1),ahora+pg_catalog.make_interval(hours=>1),
      'VEC-AD-3-COSE-EDDSA-1','vec-diputacion/pruebas/f0/c2','acto:f0-c2-raiz',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
    VALUES ('configuracion:f0-c2','raiz:f0-c2',800001);
    INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
    VALUES (800001,'configuracion:f0-c2',ahora-pg_catalog.make_interval(hours=>1),
      'acto:f0-c2-puntero-config',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
    VALUES ('clave:f0-c2',800001,800001,pg_catalog.repeat('5',64),secreto,
      pg_catalog.encode(pg_catalog.sha256(secreto),'hex'),'emisor:f0-c2',
      'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
      ahora-pg_catalog.make_interval(hours=>1),ahora+pg_catalog.make_interval(hours=>1),
      'acto:f0-c2-clave',ahora);
    INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
    VALUES (800001,'clave:f0-c2',800001,ahora-pg_catalog.make_interval(hours=>1),
      'acto:f0-c2-puntero-clave',ahora);
    ALTER TABLE vec_autorizacion_atestada_v3
      .fuente_corporativa_contexto_actor_v1 DISABLE TRIGGER f0_checkpoint_antes;
    INSERT INTO vec_autorizacion_atestada_v3
      .fuente_corporativa_contexto_actor_v1 VALUES (
      'fuente:f0-c2-sintetica',800001,
      'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
      'contexto_actor.organizacion_corporativa.publicar',
      'organizacion_corporativa.alta','clave:f0-c2',800001,800001,
      pg_catalog.repeat('5',64),'emisor:f0-c2','configuracion:f0-c2',800001,
      pg_catalog.repeat('6',64),'raiz:f0-c2',800001,
      pg_catalog.encode(pg_catalog.sha256(spki),'hex'),
      'vec-diputacion/pruebas/f0/c2','VEC-AD-3-COSE-EDDSA-1',
      ahora-pg_catalog.make_interval(hours=>1),ahora+pg_catalog.make_interval(hours=>1),
      'acto:f0-c2-fuente',ahora);
    ALTER TABLE vec_autorizacion_atestada_v3
      .fuente_corporativa_contexto_actor_v1 ENABLE TRIGGER f0_checkpoint_antes;
    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision=revision+1,
           configuracion_secuencia_minima=pg_catalog.greatest(
             configuracion_secuencia_minima,800001),
           raiz_version_minima=pg_catalog.greatest(raiz_version_minima,800001),
           actualizada_en=ahora WHERE control_id;
END
$fixture_c2$;

CREATE TEMP TABLE material_c2_prueba_tabla(
    secreto pg_catalog.bytea,spki pg_catalog.bytea,sobre pg_catalog.bytea,
    evidencia pg_catalog.bytea,manifiesto pg_catalog.bytea,
    capacidad pg_catalog.bytea) ON COMMIT DROP;
INSERT INTO material_c2_prueba_tabla
SELECT * FROM vec_autorizacion_atestada_v3.material_c2_prueba(
  pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.02));

CREATE FUNCTION vec_autorizacion_atestada_v3.invocar_c2_prueba(
    p_caso pg_catalog.text)
RETURNS pg_catalog.jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $funcion$
DECLARE
    m record;
    j pg_catalog.json;
    sobre_usado pg_catalog.bytea;
    operacion pg_catalog.text;
    efecto pg_catalog.text;
    cantidad pg_catalog.int8;
    salidas pg_catalog.jsonb[];
BEGIN
    SELECT * INTO STRICT m FROM pg_temp.material_c2_prueba_tabla;
    j:=pg_catalog.convert_from(m.manifiesto,'UTF8')::pg_catalog.json;
    sobre_usado:=CASE WHEN p_caso='material' THEN
      pg_catalog.set_byte(m.sobre,0,pg_catalog.get_byte(m.sobre,0)#1)
      ELSE m.sobre END;
    operacion:=CASE WHEN p_caso='operacion' THEN
      'oca_f0_c2_cruce0000000000000001' ELSE j->>'operacion_ref' END;
    efecto:=CASE WHEN p_caso='efecto' THEN
      'efecto:f0-c2-cruzado' ELSE j->>'efecto_ref' END;
    SELECT pg_catalog.count(*),pg_catalog.array_agg(pg_catalog.to_jsonb(r))
      INTO cantidad,salidas
      FROM vec_autorizacion_atestada_v3
        .consumir_fuente_corporativa_contexto_actor_v1_atestada(
          j->>'audiencia_consumo',j->>'accion',j->>'tipo_efecto',operacion,
          efecto,j->>'huella_efecto_sha256',m.capacidad,m.manifiesto,
          sobre_usado,m.evidencia,m.spki) AS r;
    IF cantidad<>1 THEN
      RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: retorno no singular';
    END IF;
    RETURN salidas[1];
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.capturar_estado_c2_prueba(
    p_caso pg_catalog.text)
RETURNS pg_catalog.text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $funcion$
BEGIN
    PERFORM vec_autorizacion_atestada_v3.invocar_c2_prueba(p_caso);
    RETURN 'ok';
EXCEPTION WHEN OTHERS THEN
    RETURN SQLSTATE;
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.colision_c2_prueba(
    p_coordenada pg_catalog.text)
RETURNS pg_catalog.bool
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $funcion$
DECLARE
    m record;
    c pg_catalog.json;
    cap_real pg_catalog.text;
    cap_falsa pg_catalog.text;
    canon pg_catalog.bytea;
    huella pg_catalog.text;
    instante pg_catalog.timestamptz:=pg_catalog.clock_timestamp();
BEGIN
    SELECT * INTO STRICT m FROM pg_temp.material_c2_prueba_tabla;
    c:=pg_catalog.convert_from(m.capacidad,'UTF8')::pg_catalog.json;
    cap_real:='cfc_'||pg_catalog.encode(pg_catalog.sha256(m.capacidad),'hex');
    cap_falsa:='cfc_'||pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to('colision-f0-c2-'||p_coordenada,'UTF8')),'hex');
    SELECT x.consumo_canonico,x.consumo_huella_sha256 INTO STRICT canon,huella
      FROM vec_autorizacion_atestada_v3
        .canon_y_huella_consumo_fuente_corporativa_v1(m.capacidad,instante) AS x;
    BEGIN
      INSERT INTO vec_autorizacion_atestada_v3
        .atestacion_fuente_corporativa_contexto_actor_v1 VALUES (
        CASE WHEN p_coordenada='capacidad' THEN cap_real ELSE cap_falsa END,
        CASE WHEN p_coordenada='evento' THEN c->>'fuente_ref'
             ELSE 'fuente:f0-c2-colision' END,800001,
        CASE WHEN p_coordenada='evento' THEN c->>'evento_fuente_ref'
             ELSE 'evento:f0-c2-colision' END);
      INSERT INTO vec_autorizacion_atestada_v3
        .consumo_fuente_corporativa_contexto_actor_v1 VALUES (
        CASE WHEN p_coordenada='capacidad' THEN cap_real ELSE cap_falsa END,
        CASE WHEN p_coordenada='nonce' THEN c->>'nonce'
             ELSE pg_catalog.repeat('a',64) END,
        CASE WHEN p_coordenada='operacion' THEN c->>'operacion_ref'
             ELSE 'oca_f0_c2_colision00000000000001' END,
        canon,huella,instante);
      PERFORM vec_autorizacion_atestada_v3.invocar_c2_prueba('normal');
      RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: colision aceptada';
    EXCEPTION WHEN unique_violation THEN
      RETURN true;
    END;
END
$funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.fallar_estado_c2_prueba()
RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $funcion$
DECLARE codigo pg_catalog.text:=pg_catalog.current_setting('vec.f0_c2_estado',true);
BEGIN
    IF codigo IN ('23505','40001') THEN
      RAISE EXCEPTION USING ERRCODE=codigo,MESSAGE='estado inyectado C2';
    END IF;
    RETURN NEW;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.invocar_c2_prueba(pg_catalog.text),
    vec_autorizacion_atestada_v3.capturar_estado_c2_prueba(pg_catalog.text),
    vec_autorizacion_atestada_v3.colision_c2_prueba(pg_catalog.text),
    vec_autorizacion_atestada_v3.fallar_estado_c2_prueba()
FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO
    vec_f0_h0_publicador,vec_f0_h0_revocador,vec_f0_h0_despachador,
    vec_f0_h0_cruzado,vec_f0_h0_extra,vec_f0_h0_sin_rol;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3.invocar_c2_prueba(pg_catalog.text),
    vec_autorizacion_atestada_v3.capturar_estado_c2_prueba(pg_catalog.text),
    vec_autorizacion_atestada_v3.colision_c2_prueba(pg_catalog.text)
TO vec_f0_h0_publicador,vec_f0_h0_revocador,vec_f0_h0_despachador,
    vec_f0_h0_cruzado,vec_f0_h0_extra,vec_f0_h0_sin_rol;

CREATE TRIGGER f0_c2_estado_prueba BEFORE INSERT ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.fallar_estado_c2_prueba();

RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
DO $publicador_adversarial$
DECLARE caso pg_catalog.text;
BEGIN
    FOREACH caso IN ARRAY ARRAY['material','operacion','efecto'] LOOP
      IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba(caso)<>'42501' THEN
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: ligadura hostil aceptada';
      END IF;
    END LOOP;
    FOREACH caso IN ARRAY ARRAY['capacidad','evento','nonce','operacion'] LOOP
      IF vec_autorizacion_atestada_v3.colision_c2_prueba(caso) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: colision no cerrada';
      END IF;
    END LOOP;
    IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .atestacion_fuente_corporativa_contexto_actor_v1)<>0
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1)<>0 THEN
      RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: colision dejo historia';
    END IF;
END
$publicador_adversarial$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

TRUNCATE pg_temp.material_c2_prueba_tabla;
INSERT INTO pg_temp.material_c2_prueba_tabla
SELECT * FROM vec_autorizacion_atestada_v3.material_c2_prueba(
  pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.02));

DO $preparar_23505$ BEGIN
 PERFORM pg_catalog.set_config('vec.f0_c2_estado','23505',true);
END $preparar_23505$;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
DO $preservar_23505$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'23505' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: 23505 enmascarado'; END IF;
END $preservar_23505$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $preparar_40001$ BEGIN
 PERFORM pg_catalog.set_config('vec.f0_c2_estado','40001',true);
END $preparar_40001$;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
DO $preservar_40001$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'40001' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: 40001 enmascarado'; END IF;
END $preservar_40001$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $limpiar_estado$ BEGIN
 PERFORM pg_catalog.set_config('vec.f0_c2_estado','',true);
END $limpiar_estado$;

DROP TRIGGER f0_c2_estado_prueba ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1;
DROP FUNCTION vec_autorizacion_atestada_v3.fallar_estado_c2_prueba();

RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_revocador;
DO $r0_revocador$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: cruce R0 aceptado'; END IF;
END $r0_revocador$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_despachador;
DO $r0_despachador$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: despachador aceptado'; END IF;
END $r0_despachador$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_cruzado;
DO $r0_cruzado$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: membresia cruzada aceptada'; END IF;
END $r0_cruzado$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_extra;
DO $r0_extra$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: membresia adicional aceptada'; END IF;
END $r0_extra$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_sin_rol;
DO $r0_ausente$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: LOGIN sin R0 aceptado'; END IF;
END $r0_ausente$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
SET LOCAL ROLE vec_f0_h0_publicador;
DO $r0_set_role$ BEGIN
 IF vec_autorizacion_atestada_v3.capturar_estado_c2_prueba('normal')<>'42501' THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: SET ROLE aceptado'; END IF;
END $r0_set_role$;
RESET ROLE;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

TRUNCATE pg_temp.material_c2_prueba_tabla;
INSERT INTO pg_temp.material_c2_prueba_tabla
SELECT * FROM vec_autorizacion_atestada_v3.material_c2_prueba(
  pg_catalog.clock_timestamp()-pg_catalog.make_interval(secs=>0.02));

CREATE FUNCTION vec_autorizacion_atestada_v3.nominal_c2_prueba()
RETURNS pg_catalog.bool
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $funcion$
DECLARE nuevo pg_catalog.jsonb; replay pg_catalog.jsonb; cantidad pg_catalog.int8;
BEGIN
    nuevo:=vec_autorizacion_atestada_v3.invocar_c2_prueba('normal');
    INSERT INTO vec_autorizacion_atestada_v3
      .revocacion_fuente_corporativa_contexto_actor_v1 VALUES (
      'fuente:f0-c2-sintetica',800001,
      'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
      pg_catalog.clock_timestamp(),'motivo:f0-c2-replay',
      'acto:f0-c2-replay',pg_catalog.clock_timestamp());
    replay:=vec_autorizacion_atestada_v3.invocar_c2_prueba('normal');
    SELECT pg_catalog.count(*) INTO cantidad
      FROM pg_catalog.pg_locks AS l
     WHERE l.pid=pg_catalog.pg_backend_pid() AND l.granted
       AND l.locktype='advisory' AND l.mode='ExclusiveLock';
    RETURN pg_catalog.jsonb_object_length(nuevo)=12
       AND pg_catalog.jsonb_object_length(replay)=12
       AND nuevo->>'consumo_nuevo'='true' AND replay->>'consumo_nuevo'='false'
       AND (nuevo-'consumo_nuevo')=(replay-'consumo_nuevo')
       AND nuevo->>'capacidad_ref'='cfc_'||pg_catalog.encode(pg_catalog.sha256(
          (SELECT m.capacidad FROM pg_temp.material_c2_prueba_tabla AS m)),'hex')
       AND nuevo->>'consumida_en'=replay->>'consumida_en'
       AND cantidad>=2
       AND (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .atestacion_fuente_corporativa_contexto_actor_v1)=1
       AND (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
          .consumo_fuente_corporativa_contexto_actor_v1)=1;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.nominal_c2_prueba()
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.nominal_c2_prueba()
TO vec_f0_h0_publicador;

RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_f0_h0_publicador;
DO $nominal_c2$
BEGIN
 IF vec_autorizacion_atestada_v3.nominal_c2_prueba() IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: alta o replay inexactos';
 END IF;
END
$nominal_c2$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

DO $append_only_c2$
BEGIN
 BEGIN
  UPDATE vec_autorizacion_atestada_v3
    .atestacion_fuente_corporativa_contexto_actor_v1 SET fuente_version=fuente_version;
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: UPDATE aceptado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  DELETE FROM vec_autorizacion_atestada_v3
    .consumo_fuente_corporativa_contexto_actor_v1;
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: DELETE aceptado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  TRUNCATE vec_autorizacion_atestada_v3
    .consumo_fuente_corporativa_contexto_actor_v1;
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: TRUNCATE aceptado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 IF (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3
      .consumo_fuente_corporativa_contexto_actor_v1)<>1 THEN
  RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='C2: historia perdida';
 END IF;
END
$append_only_c2$;

REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM
    vec_f0_h0_publicador,vec_f0_h0_revocador,vec_f0_h0_despachador,
    vec_f0_h0_cruzado,vec_f0_h0_extra,vec_f0_h0_sin_rol;
DROP FUNCTION vec_autorizacion_atestada_v3.nominal_c2_prueba();
DROP FUNCTION vec_autorizacion_atestada_v3.colision_c2_prueba(pg_catalog.text);
DROP FUNCTION vec_autorizacion_atestada_v3.capturar_estado_c2_prueba(pg_catalog.text);
DROP FUNCTION vec_autorizacion_atestada_v3.invocar_c2_prueba(pg_catalog.text);
DROP FUNCTION vec_autorizacion_atestada_v3.material_c2_prueba(pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_c2_prueba(
  pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,pg_catalog.bytea,
  pg_catalog.bytea,pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.manifiesto_c2_prueba(
  pg_catalog.timestamptz);
DROP FUNCTION vec_autorizacion_atestada_v3.acreditar_forma_c2_prueba();
DROP TABLE pg_temp.material_c2_prueba_tabla;
