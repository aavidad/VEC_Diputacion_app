\set ON_ERROR_STOP on
-- AD3-117: tres lecturas nominales del catalogo comun RPT. Requiere
-- catalogos_configurables 000002 y AD3-114. No publica concesiones ni roles.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000117',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE
    rol text;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_ct_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.listar_habilitadas(text,text,integer)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.consultar_uso(text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
          'vec_catalogos_configurables.listar_habilitadas(text,text,integer)','EXECUTE') THEN
        RAISE EXCEPTION 'AD3-117: preimagen de autoridad incompatible' USING ERRCODE='55000';
    END IF;
    FOREACH rol IN ARRAY ARRAY['vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                       WHERE rolname=rol AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
           OR pg_catalog.has_function_privilege(rol,'vec_catalogos_configurables.listar_habilitadas(text,text,integer)','EXECUTE') THEN
            RAISE EXCEPTION 'AD3-117: rol tecnico incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

-- La funcion comun conserva OID, ACL, propietario, configuracion, dependencias
-- y todos los caminos anteriores. Solo se insertan un perfil tecnico nuevo y
-- su lista positiva de acciones/campos/audiencia sobre marcas exactas.
DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
    propietario oid; config text[]; definidora boolean;
    exclusor text := E'               AND p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion_v2''';
    cierre_roles text := E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',\n            MESSAGE = ''consumo VEC-AD-3 rechazado'';';
    rol_nuevo text := $x$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_categorias'
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
                    WHERE m.member=session_user::pg_catalog.regrole
                      AND g.rolname IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                      AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
                      AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))=1
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                    WHERE m.member IN ('vec_contratacion_temporal_ejecutor'::pg_catalog.regrole,
                                       'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole,
                                       'vec_personal_ejecutor'::pg_catalog.regrole))
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE pg_catalog.left(r.rolname,4)='vec_'
                    AND r.rolname<>session_user
                    AND r.rolname NOT IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                    AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
$x$;
    marca_capacidad text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    capacidad_nueva text := $x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.lectura_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.listar_habilitadas',
                                   'vec.catalogos.categorias.consultar_historica',
                                   'vec.catalogos.categorias.consultar_uso'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='vec.catalogos.categorias.consultar_uso'
       THEN 'uso_categoria' ELSE 'catalogo_configurable' END
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_categorias_rpt'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE c->>'operacion'
       WHEN 'vec.catalogos.categorias.listar_habilitadas' THEN '["categorias","paginacion","publicaciones"]'::jsonb
       WHEN 'vec.catalogos.categorias.consultar_historica' THEN '["publicacion","entrada","control_actual"]'::jsonb
       ELSE '["uso"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
      INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc AS p WHERE p.oid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO deps FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    IF propietario <> 'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
       OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,exclusor,''))<>pg_catalog.length(exclusor)
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,cierre_roles,''))<>pg_catalog.length(cierre_roles)
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca_capacidad,''))<>pg_catalog.length(marca_capacidad)
       OR pg_catalog.strpos(original,'lectura_categorias')<>0
       OR pg_catalog.strpos(original,'ajustes_reglas_ct')=0 THEN
        RAISE EXCEPTION 'AD3-117: núcleo V3 incompatible' USING ERRCODE='55000';
    END IF;
    nuevo := pg_catalog.replace(original,exclusor,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_categorias''');
    nuevo := pg_catalog.replace(nuevo,cierre_roles,rol_nuevo||cierre_roles);
    nuevo := pg_catalog.replace(nuevo,marca_capacidad,capacidad_nueva||marca_capacidad);
    EXECUTE nuevo;
    SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
    IF actual IS DISTINCT FROM nuevo
       OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc AS p WHERE p.oid=f) IS DISTINCT FROM meta
       OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
       OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM config
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-117: núcleo V3 alterado fuera de contrato' USING ERRCODE='55000';
    END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE
    definicion text;
    nueva text := 'vec_catalogos_configurables.lectura_categorias.v1';
BEGIN
    SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
    IF pg_catalog.strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1
       OR pg_catalog.right(definicion,3)<>']))'
       OR pg_catalog.strpos(definicion,'vec_contratacion_temporal.ajustes_reglas.v1')=0
       OR pg_catalog.strpos(definicion,pg_catalog.quote_literal(nueva))<>0 THEN
        RAISE EXCEPTION 'AD3-117: audiencias previas incompatibles' USING ERRCODE='55000';
    END IF;
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
        ||pg_catalog.left(definicion,pg_catalog.length(definicion)-3)||', '||pg_catalog.quote_literal(nueva)||'::text]))';
END $audiencia$;

-- La huella del material es el texto canonico de jsonb de PostgreSQL. El
-- preparador V3 la obtiene por SELECT antes de solicitar la decision. Esta
-- funcion la recalcula, liga recurso y campos y consume una decision nueva.
CREATE FUNCTION vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(
    p_material jsonb,p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
    p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE
    c jsonb; d jsonb; x record;
    material_h text; contexto_h text; ambitos text; recurso text; tipo text; campos jsonb;
    consumidor_tecnico text;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material)<>'object'
       OR pg_catalog.octet_length(p_material::text)>2048
       OR p_accion IS NULL OR p_accion NOT IN (
            'vec.catalogos.categorias.listar_habilitadas',
            'vec.catalogos.categorias.consultar_historica',
            'vec.catalogos.categorias.consultar_uso') THEN
        RAISE EXCEPTION 'AD3-117: material de lectura invalido' USING ERRCODE='22023';
    END IF;
    IF p_material->>'catalogo_id' IS NULL OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'modulo_id' IS NULL OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$' THEN
        RAISE EXCEPTION 'AD3-117: descriptor de catalogo invalido' USING ERRCODE='22023';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'AD3-117: atestacion invalida' USING ERRCODE='22023';
    END;
    material_h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material::text,'UTF8')),'hex');
    IF p_accion='vec.catalogos.categorias.consultar_uso' THEN
        recurso := p_material->>'uso_ref';
        tipo := 'uso_categoria';
        campos := '["uso"]'::jsonb;
        IF p_material->>'consumidor' IS NULL OR p_material->>'consumidor' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR recurso IS NULL OR pg_catalog.octet_length(recurso) NOT BETWEEN 3 AND 160
           OR p_material->>'reserva_recibo_ref' IS NULL
           OR pg_catalog.octet_length(p_material->>'reserva_recibo_ref') NOT BETWEEN 3 AND 160 THEN
            RAISE EXCEPTION 'AD3-117: uso invalido' USING ERRCODE='22023';
        END IF;
        consumidor_tecnico := CASE
            WHEN pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') THEN 'contratacion_temporal'
            WHEN pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') THEN 'bolsa'
            WHEN pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER') THEN 'personal'
            ELSE NULL END;
        IF consumidor_tecnico IS DISTINCT FROM p_material->>'consumidor' THEN
            RAISE EXCEPTION 'AD3-117: consumidor ajeno' USING ERRCODE='42501';
        END IF;
        ambitos := '{"ambitos":{"catalogo_id":"'||(p_material->>'catalogo_id')||'","consumidor":"'||consumidor_tecnico||'","modulo_id":"'||(p_material->>'modulo_id')||'"},"atributos":{"material_sha256":"'||material_h||'"}}';
    ELSE
        recurso := p_material->>'catalogo_id';
        tipo := 'catalogo_configurable';
        campos := CASE WHEN p_accion='vec.catalogos.categorias.listar_habilitadas'
            THEN '["categorias","paginacion","publicaciones"]'::jsonb
            ELSE '["publicacion","entrada","control_actual"]'::jsonb END;
        ambitos := '{"ambitos":{"catalogo_id":"'||recurso||'","modulo_id":"'||(p_material->>'modulo_id')||'"},"atributos":{"material_sha256":"'||material_h||'"}}';
    END IF;
    contexto_h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ambitos,'UTF8')),'hex');
    IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.lectura_categorias.v1'
       OR c->>'operacion' IS DISTINCT FROM p_accion
       OR d->>'accion' IS DISTINCT FROM p_accion
       OR d->>'modulo_id' IS DISTINCT FROM p_material->>'modulo_id'
       OR d->>'tipo_recurso' IS DISTINCT FROM tipo
       OR d->>'finalidad' IS DISTINCT FROM 'consultar_categorias_rpt'
       OR d->>'recurso_ref' IS DISTINCT FROM recurso
       OR c->>'efecto_ref' IS DISTINCT FROM recurso
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
       OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
       OR d->'campos_permitidos' IS DISTINCT FROM campos
       OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION 'AD3-117: lectura no autorizada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'lectura_categorias',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF x.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-117: lectura requiere consumo nuevo' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
        x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

-- El documento canonico se devuelve como STRING original. La decision V3
-- se consume antes de leer; una denegacion posterior revierte su transaccion.
CREATE FUNCTION vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE
    a record; r jsonb; pub jsonb; doc jsonb; respuesta jsonb; catalogo text; modulo text; limite integer;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material)<>'object' THEN
        RAISE EXCEPTION 'AD3-117: lista invalida' USING ERRCODE='22023';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_material))<>4
       OR p_material->>'catalogo_id' IS NULL OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'modulo_id' IS NULL OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR NOT (p_material ? 'cursor_categoria_id')
       OR (pg_catalog.jsonb_typeof(p_material->'cursor_categoria_id') NOT IN ('null','string'))
       OR (pg_catalog.jsonb_typeof(p_material->'cursor_categoria_id')='string'
           AND p_material->>'cursor_categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$')
       OR p_material->>'limite' IS NULL OR p_material->>'limite' !~ '^(100|[1-9][0-9]?)$' THEN
        RAISE EXCEPTION 'AD3-117: lista invalida' USING ERRCODE='22023';
    END IF;
    catalogo := p_material->>'catalogo_id'; modulo := p_material->>'modulo_id';
    limite := (p_material->>'limite')::integer;
    SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(
        p_material,'vec.catalogos.categorias.listar_habilitadas',p_capacidad,p_decision,p_motivo,
        p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    r := vec_catalogos_configurables.listar_habilitadas(catalogo,p_material->>'cursor_categoria_id',limite);
    IF r->>'encontrado'='true' THEN
        IF pg_catalog.jsonb_typeof(r#>'{datos,anclaje_publicacion}') IS DISTINCT FROM 'object'
           OR pg_catalog.jsonb_typeof(r#>'{datos,publicaciones}') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'AD3-117: pagina sin anclaje' USING ERRCODE='55000';
        END IF;
        FOR pub IN SELECT value FROM pg_catalog.jsonb_array_elements(
            pg_catalog.jsonb_build_array(r#>'{datos,anclaje_publicacion}') || (r#>'{datos,publicaciones}')) LOOP
            IF pub->>'catalogo_id' IS DISTINCT FROM catalogo
               OR pub->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to(pub->>'documento_canonico','UTF8')),'hex') THEN
                RAISE EXCEPTION 'AD3-117: publicacion ajena' USING ERRCODE='55000';
            END IF;
            BEGIN doc := (pub->>'documento_canonico')::jsonb;
            EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-117: canon ilegible' USING ERRCODE='55000'; END;
            IF doc->>'id' IS DISTINCT FROM catalogo
               OR doc->>'version' IS DISTINCT FROM pub->>'version'
               OR doc->>'modulo_id' IS DISTINCT FROM modulo THEN
                RAISE EXCEPTION 'AD3-117: modulo de publicacion ajeno' USING ERRCODE='42501';
            END IF;
        END LOOP;
    END IF;
    respuesta := pg_catalog.jsonb_build_object(
        'decision_ref',a.decision_ref,'efecto_ref',a.efecto_ref,
        'huella_efecto_sha256',a.huella_efecto_sha256,
        'consumo_huella_sha256',a.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,
        'consumida_en',pg_catalog.to_char(a.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'consumo_nuevo',a.consumo_nuevo,'encontrado',(r->>'encontrado')::boolean,'datos',r->'datos');
    IF pg_catalog.octet_length(respuesta::text)>50331648 THEN
        RAISE EXCEPTION 'AD3-117: pagina excede presupuesto de respuesta' USING ERRCODE='54000';
    END IF;
    RETURN respuesta;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE
    a record; r jsonb; doc jsonb; catalogo text; modulo text;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material)<>'object' THEN
        RAISE EXCEPTION 'AD3-117: historia invalida' USING ERRCODE='22023';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_material))<>5
       OR p_material->>'catalogo_id' IS NULL OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'modulo_id' IS NULL OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'version' IS NULL OR p_material->>'version' !~ '^[1-9][0-9]{0,9}$'
       OR p_material->>'huella_sha256' IS NULL OR p_material->>'huella_sha256' !~ '^[0-9a-f]{64}$'
       OR p_material->>'categoria_id' IS NULL OR p_material->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$' THEN
        RAISE EXCEPTION 'AD3-117: historia invalida' USING ERRCODE='22023';
    END IF;
    IF (p_material->>'version')::numeric>2147483647 THEN
        RAISE EXCEPTION 'AD3-117: version historica fuera de rango' USING ERRCODE='22023';
    END IF;
    catalogo := p_material->>'catalogo_id'; modulo := p_material->>'modulo_id';
    SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(
        p_material,'vec.catalogos.categorias.consultar_historica',p_capacidad,p_decision,p_motivo,
        p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    r := vec_catalogos_configurables.leer_publicacion_categoria(catalogo,
        (p_material->>'version')::integer,p_material->>'huella_sha256',p_material->>'categoria_id');
    IF r->>'encontrado'='true' THEN
        BEGIN doc := (r#>>'{datos,publicacion,documento_canonico}')::jsonb;
        EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-117: canon ilegible' USING ERRCODE='55000'; END;
        IF doc->>'id' IS DISTINCT FROM catalogo
           OR doc->>'version' IS DISTINCT FROM p_material->>'version'
           OR doc->>'modulo_id' IS DISTINCT FROM modulo
           OR p_material->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
                pg_catalog.convert_to(r#>>'{datos,publicacion,documento_canonico}','UTF8')),'hex')
           OR r#>>'{datos,entrada,clave}' IS DISTINCT FROM p_material->>'categoria_id' THEN
            RAISE EXCEPTION 'AD3-117: publicacion historica ajena' USING ERRCODE='42501';
        END IF;
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'decision_ref',a.decision_ref,'efecto_ref',a.efecto_ref,
        'huella_efecto_sha256',a.huella_efecto_sha256,
        'consumo_huella_sha256',a.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,
        'consumida_en',pg_catalog.to_char(a.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'consumo_nuevo',a.consumo_nuevo,'encontrado',(r->>'encontrado')::boolean,'datos',r->'datos');
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE
    a record; r jsonb; historica jsonb; doc jsonb; catalogo text; modulo text;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material)<>'object' THEN
        RAISE EXCEPTION 'AD3-117: uso invalido' USING ERRCODE='22023';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_material))<>5
       OR p_material->>'catalogo_id' IS NULL OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'modulo_id' IS NULL OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'consumidor' IS NULL OR p_material->>'consumidor' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_material->>'uso_ref' IS NULL OR pg_catalog.octet_length(p_material->>'uso_ref') NOT BETWEEN 3 AND 160
       OR p_material->>'reserva_recibo_ref' IS NULL
       OR pg_catalog.octet_length(p_material->>'reserva_recibo_ref') NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'AD3-117: uso invalido' USING ERRCODE='22023';
    END IF;
    catalogo := p_material->>'catalogo_id'; modulo := p_material->>'modulo_id';
    SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(
        p_material,'vec.catalogos.categorias.consultar_uso',p_capacidad,p_decision,p_motivo,
        p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    r := vec_catalogos_configurables.consultar_uso(p_material->>'consumidor',
        p_material->>'uso_ref',p_material->>'reserva_recibo_ref');
    IF r->>'encontrado'='true' THEN
        IF r#>>'{datos,catalogo_id}' IS DISTINCT FROM catalogo
           OR r#>>'{datos,consumidor}' IS DISTINCT FROM p_material->>'consumidor'
           OR r#>>'{datos,uso_ref}' IS DISTINCT FROM p_material->>'uso_ref'
           OR r#>>'{datos,reserva_recibo_ref}' IS DISTINCT FROM p_material->>'reserva_recibo_ref' THEN
            RAISE EXCEPTION 'AD3-117: uso ajeno' USING ERRCODE='42501';
        END IF;
        historica := vec_catalogos_configurables.leer_publicacion_categoria(catalogo,
            (r#>>'{datos,version}')::integer,r#>>'{datos,huella_sha256}',r#>>'{datos,categoria_id}');
        IF historica->>'encontrado' IS DISTINCT FROM 'true' THEN
            RAISE EXCEPTION 'AD3-117: uso sin publicacion' USING ERRCODE='55000';
        END IF;
        BEGIN doc := (historica#>>'{datos,publicacion,documento_canonico}')::jsonb;
        EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-117: canon ilegible' USING ERRCODE='55000'; END;
        IF doc->>'id' IS DISTINCT FROM catalogo OR doc->>'modulo_id' IS DISTINCT FROM modulo
           OR doc->>'version' IS DISTINCT FROM r#>>'{datos,version}'
           OR r#>>'{datos,huella_sha256}' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
                pg_catalog.convert_to(historica#>>'{datos,publicacion,documento_canonico}','UTF8')),'hex') THEN
            RAISE EXCEPTION 'AD3-117: modulo de uso ajeno' USING ERRCODE='42501';
        END IF;
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'decision_ref',a.decision_ref,'efecto_ref',a.efecto_ref,
        'huella_efecto_sha256',a.huella_efecto_sha256,
        'consumo_huella_sha256',a.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,
        'consumida_en',pg_catalog.to_char(a.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'consumo_nuevo',a.consumo_nuevo,'encontrado',(r->>'encontrado')::boolean,'datos',r->'datos');
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor,vec_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor,vec_personal_ejecutor;
DO $acl$
DECLARE
    f regprocedure;
    x record;
    ayuda regprocedure := 'vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        ayuda,
        'vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ] LOOP
        FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc AS p
          CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
         WHERE p.oid=f AND a.grantee<>p.proowner LOOP
            EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
                CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
        END LOOP;
        IF f<>ayuda THEN
            EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor,vec_personal_ejecutor',f::text);
        END IF;
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p WHERE p.oid=f
                       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
                       AND p.prosecdef
                       AND p.proconfig=CASE WHEN f=ayuda
                           THEN ARRAY['search_path=pg_catalog','lock_timeout=2s']
                           ELSE ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=30s'] END)
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p
             CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
             WHERE p.oid=f AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable
               OR (f=ayuda AND a.grantee<>p.proowner)
               OR (f<>ayuda AND a.grantee NOT IN (p.proowner,
                   'vec_contratacion_temporal_ejecutor'::regrole,
                   'vec_bolsa_llamamientos_ejecutor'::regrole,
                   'vec_personal_ejecutor'::regrole)))) THEN
            RAISE EXCEPTION 'AD3-117: ACL de fachada incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
COMMIT;
