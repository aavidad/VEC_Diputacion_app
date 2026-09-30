\set ON_ERROR_STOP on
-- AD3-126: tres usos nominales del catalogo comun RPT. Requiere
-- H6 completo (incluida AD3-125), catalogos_configurables 000003 y AD3-117.
-- AD3-132 debe preceder RPT en la instalacion. No publica concesiones ni roles.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000126',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE rol text;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
          'vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)','EXECUTE') THEN
        RAISE EXCEPTION 'AD3-126: preimagen de autoridad incompatible' USING ERRCODE='55000';
    END IF;
    FOREACH rol IN ARRAY ARRAY['vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                       WHERE rolname=rol AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
           OR pg_catalog.has_function_privilege(rol,
                'vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)','EXECUTE') THEN
            RAISE EXCEPTION 'AD3-126: rol tecnico incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

-- Preimagen completa postH6/postAD117: cuerpo/definicion y metadata de PG18, cotejados
-- con la cadena causal. La reconstruccion conserva OID, ACL, propietario y
-- dependencias; cambia solo el cuerpo nominal y search_path a pg_catalog, pg_temp.
DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    original text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
    fuente text; fuente_sha256 text; definicion_sha256 text;
    propietario oid; config text[]; definidora boolean;
    exclusor text := E'               AND p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion_v2'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_categorias''';
    cierre_roles text := E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',\n            MESSAGE = ''consumo VEC-AD-3 rechazado'';';
    rol_nuevo text := $x$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'usos_categorias'
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
 p_perfil_mutacion IS NOT DISTINCT FROM 'usos_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.usos_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.reservar_uso',
                                   'vec.catalogos.categorias.confirmar_uso',
                                   'vec.catalogos.categorias.cancelar_uso'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'uso_categoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'vincular_categoria_a_operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo","uso"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p),p.proacl,p.proowner,p.proconfig,p.prosecdef,p.prosrc
      INTO STRICT original,meta,acl,propietario,config,definidora,fuente FROM pg_catalog.pg_proc AS p WHERE p.oid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO deps FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      INTO deps_compartidas FROM pg_catalog.pg_shdepend AS d
     WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())
       AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    fuente_sha256 := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex');
    definicion_sha256 := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex');
    IF fuente_sha256 IS DISTINCT FROM '8cda19bc0ab03f811386f7ec6b54a4662b68988590538d2857f48691c8c3c842'
       OR definicion_sha256 IS DISTINCT FROM '334d3a9d8397de1a37ca559ba12e0955510649da624b0ca3bdbab20a5729839f'
       OR propietario <> 'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
       OR deps IS DISTINCT FROM (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
            'classid','pg_catalog.pg_proc'::regclass::oid,'objid',f,'objsubid',0,
            'refclassid',v.refclassid,'refobjid',v.refobjid,'refobjsubid',0,'deptype','n')
            ORDER BY v.refclassid)
          FROM (VALUES ('pg_catalog.pg_language'::regclass::oid,(SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')),
                       ('pg_catalog.pg_namespace'::regclass::oid,'vec_autorizacion_atestada_v3'::regnamespace::oid)) AS v(refclassid,refobjid))
       OR deps_compartidas IS DISTINCT FROM pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object(
            'dbid',(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database()),
            'classid','pg_catalog.pg_proc'::regclass::oid,'objid',f,'objsubid',0,
            'refclassid','pg_catalog.pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
       OR (meta-'prosrc'-'oid'-'prolang'-'pronamespace'-'proowner')
            IS DISTINCT FROM $meta$
{
    "proacl": [
        "vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario"
    ],
    "proallargtypes": [
        "25",
        "17",
        "17",
        "17",
        "17",
        "1700",
        "1700",
        "17",
        "17",
        "17",
        "17",
        "25",
        "25",
        "25",
        "25",
        "25",
        "1184",
        "16"
    ],
    "proargdefaults": null,
    "proargmodes": [
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "i",
        "t",
        "t",
        "t",
        "t",
        "t",
        "t",
        "t"
    ],
    "proargnames": [
        "p_perfil_mutacion",
        "p_capacidad_canonica",
        "p_decision_canonica",
        "p_motivo_canonico",
        "p_contexto_actor_canonico",
        "p_persona_version",
        "p_perfil_version",
        "p_payload_vec_ad_3",
        "p_sobre_cose_sign1",
        "p_evidencia_verificacion",
        "p_raiz_publica_spki",
        "decision_ref",
        "efecto_ref",
        "huella_efecto_sha256",
        "consumo_huella_sha256",
        "auditoria_ref",
        "consumida_en",
        "consumo_nuevo"
    ],
    "proargtypes": [
        "25",
        "17",
        "17",
        "17",
        "17",
        "1700",
        "1700",
        "17",
        "17",
        "17",
        "17"
    ],
    "probin": null,
    "proconfig": [
        "search_path=pg_catalog",
        "lock_timeout=2s"
    ],
    "procost": 100,
    "proisstrict": false,
    "prokind": "f",
    "proleakproof": false,
    "proname": "consumir_decision_mutacion_v3_interna",
    "pronargdefaults": 0,
    "pronargs": 11,
    "proparallel": "u",
    "proretset": true,
    "prorettype": "2249",
    "prorows": 1000,
    "prosecdef": true,
    "prosqlbody": null,
    "prosupport": "-",
    "protrftypes": null,
    "provariadic": "0",
    "provolatile": "v"
}
$meta$::jsonb
       OR (SELECT prolang FROM pg_catalog.pg_proc WHERE oid=f)
            IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
       OR (SELECT pronamespace FROM pg_catalog.pg_proc WHERE oid=f)
            IS DISTINCT FROM 'vec_autorizacion_atestada_v3'::regnamespace::oid
       OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,exclusor,''))<>pg_catalog.length(exclusor)
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,cierre_roles,''))<>pg_catalog.length(cierre_roles)
       OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca_capacidad,''))<>pg_catalog.length(marca_capacidad)
       OR pg_catalog.strpos(original,'lectura_categorias')=0
       OR pg_catalog.strpos(original,'usos_categorias')<>0
       OR pg_catalog.strpos(original,'ajustes_reglas_ct')=0 THEN
        RAISE EXCEPTION 'AD3-126: núcleo V3 incompatible' USING ERRCODE='55000';
    END IF;
    nuevo := pg_catalog.replace(original,exclusor,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''usos_categorias''');
    nuevo := pg_catalog.replace(nuevo,cierre_roles,rol_nuevo||cierre_roles);
    nuevo := pg_catalog.replace(nuevo,marca_capacidad,capacidad_nueva||marca_capacidad);
    nuevo := pg_catalog.replace(nuevo,E'SET search_path TO ''pg_catalog''',E'SET search_path TO ''pg_catalog'', ''pg_temp''');
    EXECUTE nuevo;
    SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
    IF actual IS DISTINCT FROM nuevo
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual,'UTF8')),'hex')
            IS DISTINCT FROM '3a06d11882d7b1ed5d3256f0547debf672975dce3c27060118bf169ce6bed692'
       OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
             FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '848799985debd0b736a3c281b0a3a3635e6182bd78789fb364121fe7f78d8a06'
       OR (SELECT pg_catalog.to_jsonb(p)-'prosrc'-'proconfig' FROM pg_catalog.pg_proc AS p WHERE p.oid=f)
            IS DISTINCT FROM (meta-'prosrc'-'proconfig')
       OR 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid IS DISTINCT FROM f
       OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
       OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_shdepend AS d
            WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())
              AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas THEN
        RAISE EXCEPTION 'AD3-126: núcleo V3 alterado fuera de contrato' USING ERRCODE='55000';
    END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE
    c_oid oid; original text; definicion text; esperada text; actual text;
    meta jsonb; deps jsonb; deps_compartidas jsonb;
    nueva text := 'vec_catalogos_configurables.usos_categorias.v1';
BEGIN
    SELECT c.oid,pg_catalog.pg_get_constraintdef(c.oid,true),pg_catalog.to_jsonb(c)
      INTO STRICT c_oid,original,meta FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check';
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)-'objid'
        ORDER BY d.classid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO deps FROM pg_catalog.pg_depend AS d
     WHERE d.classid='pg_catalog.pg_constraint'::regclass AND d.objid=c_oid;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)-'objid'
        ORDER BY d.dbid,d.classid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      INTO deps_compartidas FROM pg_catalog.pg_shdepend AS d
     WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())
       AND d.classid='pg_catalog.pg_constraint'::regclass AND d.objid=c_oid;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex')
            IS DISTINCT FROM 'cd92724aeb8e8baf8819be986ea23a215104619945944fc709352a5a0e5ec982'
       OR (meta-'oid'-'conbin'-'conrelid'-'connamespace') IS DISTINCT FROM $meta$
{
    "condeferrable": false,
    "condeferred": false,
    "conenforced": true,
    "conexclop": null,
    "confdelsetcols": null,
    "confdeltype": " ",
    "conffeqop": null,
    "confkey": null,
    "confmatchtype": " ",
    "confrelid": "0",
    "confupdtype": " ",
    "conindid": "0",
    "coninhcount": 0,
    "conislocal": true,
    "conkey": [
        8
    ],
    "conname": "clave_capacidad_version_audiencia_consumo_check",
    "connoinherit": false,
    "conparentid": "0",
    "conperiod": false,
    "conpfeqop": null,
    "conppeqop": null,
    "contype": "c",
    "contypid": "0",
    "convalidated": true
}
$meta$::jsonb
       OR (meta->>'connamespace')::oid IS DISTINCT FROM 'vec_autorizacion_atestada_v3'::regnamespace::oid
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
             WHERE oid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
               AND relowner='vec_autorizacion_atestada_v3_propietario'::regrole AND relkind='r') THEN
        RAISE EXCEPTION 'AD3-126: CHECK de audiencias incompatible' USING ERRCODE='55000';
    END IF;
    definicion := pg_catalog.regexp_replace(original,'\s+',' ','g');
    esperada := pg_catalog.left(definicion,pg_catalog.length(definicion)-3)||', '||pg_catalog.quote_literal(nueva)||'::text]))';
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||esperada;
    SELECT c.oid,pg_catalog.pg_get_constraintdef(c.oid,true)
      INTO STRICT c_oid,actual FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check';
    -- DROP/ADD cambia el OID propio y conbin; todo el resto debe conservarse.
    IF pg_catalog.regexp_replace(actual,'\s+',' ','g') IS DISTINCT FROM esperada
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual,'UTF8')),'hex')
            IS DISTINCT FROM '4fef385ffcee91a94046b1dda4368d3b85630df12d251384fd9d723352c8fdb8'
       OR (SELECT pg_catalog.to_jsonb(c)-'oid'-'conbin' FROM pg_catalog.pg_constraint AS c WHERE c.oid=c_oid)
            IS DISTINCT FROM (meta-'oid'-'conbin')
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)-'objid'
             ORDER BY d.classid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_depend AS d
            WHERE d.classid='pg_catalog.pg_constraint'::regclass AND d.objid=c_oid) IS DISTINCT FROM deps
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)-'objid'
             ORDER BY d.dbid,d.classid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_shdepend AS d
            WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())
              AND d.classid='pg_catalog.pg_constraint'::regclass AND d.objid=c_oid) IS DISTINCT FROM deps_compartidas THEN
        RAISE EXCEPTION 'AD3-126: CHECK alterado fuera de contrato' USING ERRCODE='55000';
    END IF;
END $audiencia$;

-- El preparador obtiene la huella con SELECT sha256(p_material::text) antes
-- de emitir la decision. El LOGIN debe traer un solo rol tecnico heredado.
CREATE FUNCTION vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(
    p_material jsonb,p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
    p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,
    consumo_nuevo boolean,actor_ref text,motivo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE
    c jsonb; d jsonb; m jsonb; x record; material_h text; contexto_h text;
    ambitos text; consumidor_tecnico text; motivo_canonico text;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR pg_catalog.octet_length(p_material::text)>4096
       OR p_accion IS NULL OR p_accion NOT IN (
           'vec.catalogos.categorias.reservar_uso',
           'vec.catalogos.categorias.confirmar_uso',
           'vec.catalogos.categorias.cancelar_uso') THEN
        RAISE EXCEPTION 'AD3-126: material de uso invalido' USING ERRCODE='22023';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <>
           (CASE WHEN p_accion='vec.catalogos.categorias.reservar_uso' THEN 8 ELSE 11 END)
       OR pg_catalog.jsonb_typeof(p_material->'catalogo_id') IS DISTINCT FROM 'string'
       OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR pg_catalog.jsonb_typeof(p_material->'modulo_id') IS DISTINCT FROM 'string'
       OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR pg_catalog.jsonb_typeof(p_material->'consumidor') IS DISTINCT FROM 'string'
       OR p_material->>'consumidor' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR pg_catalog.jsonb_typeof(p_material->'uso_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'uso_ref') NOT BETWEEN 3 AND 160
       OR ((p_material->>'uso_ref') COLLATE "C") !~ '^[!-~]{3,160}$'
       OR pg_catalog.strpos(p_material->>'uso_ref','*')>0
       OR pg_catalog.jsonb_typeof(p_material->'categoria_id') IS DISTINCT FROM 'string'
       OR p_material->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR pg_catalog.jsonb_typeof(p_material->'version') IS DISTINCT FROM 'number'
       OR p_material->>'version' !~ '^[1-9][0-9]{0,9}$'
       OR pg_catalog.jsonb_typeof(p_material->'huella_sha256') IS DISTINCT FROM 'string'
       OR p_material->>'huella_sha256' !~ '^[0-9a-f]{64}$'
       OR pg_catalog.jsonb_typeof(p_material->'reserva_recibo_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'reserva_recibo_ref') NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'AD3-126: material de uso invalido' USING ERRCODE='22023';
    END IF;
    IF (p_material->>'version')::numeric>2147483647 THEN
        RAISE EXCEPTION 'AD3-126: version fuera de rango' USING ERRCODE='22023';
    END IF;
    IF p_accion<>'vec.catalogos.categorias.reservar_uso' AND (
       pg_catalog.jsonb_typeof(p_material->'terminal_recibo_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'terminal_recibo_ref') NOT BETWEEN 3 AND 160
       OR pg_catalog.jsonb_typeof(p_material->'evidencia_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'evidencia_ref') NOT BETWEEN 3 AND 160
       OR pg_catalog.jsonb_typeof(p_material->'evidencia_sha256') IS DISTINCT FROM 'string'
       OR p_material->>'evidencia_sha256' !~ '^[0-9a-f]{64}$') THEN
        RAISE EXCEPTION 'AD3-126: material terminal invalido' USING ERRCODE='22023';
    END IF;
    consumidor_tecnico := CASE
        WHEN pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') THEN 'contratacion_temporal'
        WHEN pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') THEN 'bolsa'
        WHEN pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER') THEN 'personal'
        ELSE NULL END;
    IF consumidor_tecnico IS DISTINCT FROM p_material->>'consumidor' THEN
        RAISE EXCEPTION 'AD3-126: consumidor ajeno' USING ERRCODE='42501';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
        m := pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'AD3-126: atestacion invalida' USING ERRCODE='22023';
    END;
    material_h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material::text,'UTF8')),'hex');
    ambitos := '{"ambitos":{"catalogo_id":"'||(p_material->>'catalogo_id')||
        '","consumidor":"'||consumidor_tecnico||'","modulo_id":"'||(p_material->>'modulo_id')||
        '"},"atributos":{"material_sha256":"'||material_h||'"}}';
    contexto_h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ambitos,'UTF8')),'hex');
    IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.usos_categorias.v1'
       OR c->>'operacion' IS DISTINCT FROM p_accion
       OR d->>'accion' IS DISTINCT FROM p_accion
       OR d->>'modulo_id' IS DISTINCT FROM p_material->>'modulo_id'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'uso_categoria'
       OR d->>'finalidad' IS DISTINCT FROM 'vincular_categoria_a_operacion'
       OR d->>'recurso_ref' IS DISTINCT FROM p_material->>'uso_ref'
       OR c->>'efecto_ref' IS DISTINCT FROM p_material->>'uso_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
       OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
       OR d->'campos_permitidos' IS DISTINCT FROM '["recibo","uso"]'::jsonb
       OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION 'AD3-126: uso no autorizado' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'usos_categorias',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF x.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-126: uso requiere consumo nuevo' USING ERRCODE='42501';
    END IF;
    IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(m->'referencia') IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'AD3-126: motivo atestado invalido' USING ERRCODE='42501';
    END IF;
    IF m->>'esquema' IS DISTINCT FROM 'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(m))<>2
       OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(m->'referencia'))<>4
       OR pg_catalog.jsonb_typeof(m#>'{referencia,catalogo_id}') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(m#>'{referencia,catalogo_version}') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(m#>'{referencia,entrada_clave}') IS DISTINCT FROM 'string'
       OR m#>>'{referencia,catalogo_id}' !~ '^[a-z][a-z0-9._-]{0,127}$'
       OR m#>>'{referencia,catalogo_version}' !~ '^[1-9][0-9]{0,9}$'
       OR m#>>'{referencia,entrada_clave}' !~ '^[a-z][a-z0-9._-]{0,127}$'
       OR pg_catalog.jsonb_typeof(m#>'{referencia,catalogo_huella_sha256}') IS DISTINCT FROM 'string'
       OR m#>>'{referencia,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$'
       OR d->>'motivo_huella_sha256' IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_motivo),'hex') THEN
        RAISE EXCEPTION 'AD3-126: motivo atestado invalido' USING ERRCODE='42501';
    END IF;
    motivo_canonico := (m#>>'{referencia,catalogo_id}')||':'||
        (m#>>'{referencia,catalogo_version}')||':'||(m#>>'{referencia,entrada_clave}');
    IF pg_catalog.octet_length(motivo_canonico) NOT BETWEEN 3 AND 320
       OR pg_catalog.octet_length(d->>'principal_id') NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'AD3-126: referencias atestadas invalidas' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
        x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true,
        d->>'principal_id',motivo_canonico;
END $f$;

-- Los datos de modulo/publicacion se cotejan con el canon historico en SQL;
-- no se devuelven porque la concesion de escritura solo permite recibo y uso.
CREATE FUNCTION vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
    p_accion text,p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,
    p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE
    a record; r jsonb; previo jsonb; u jsonb; pub jsonb; doc jsonb;
    recibo text; esperado text; estado_esperado text; respuesta jsonb;
BEGIN
    SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(
        p_material,p_accion,p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF p_accion='vec.catalogos.categorias.reservar_uso' THEN
        esperado := p_material->>'reserva_recibo_ref';
        recibo := vec_catalogos_configurables.reservar(
            p_material->>'consumidor',p_material->>'uso_ref',p_material->>'categoria_id',
            p_material->>'catalogo_id',(p_material->>'version')::integer,
            p_material->>'huella_sha256',a.actor_ref,a.decision_ref,esperado,a.motivo_ref);
    ELSE
        previo := vec_catalogos_configurables.obtener_uso_publicacion(
            p_material->>'consumidor',p_material->>'uso_ref',p_material->>'reserva_recibo_ref');
        IF previo->>'encontrado' IS DISTINCT FROM 'true'
           OR previo#>>'{datos,uso,categoria_id}' IS DISTINCT FROM p_material->>'categoria_id'
           OR previo#>>'{datos,uso,catalogo_id}' IS DISTINCT FROM p_material->>'catalogo_id'
           OR previo#>>'{datos,uso,version}' IS DISTINCT FROM p_material->>'version'
           OR previo#>>'{datos,uso,huella_sha256}' IS DISTINCT FROM p_material->>'huella_sha256' THEN
            RAISE EXCEPTION 'AD3-126: reserva exacta ausente' USING ERRCODE='55000';
        END IF;
        estado_esperado := CASE WHEN p_accion='vec.catalogos.categorias.confirmar_uso'
            THEN 'confirmado' ELSE 'cancelado' END;
        esperado := p_material->>'terminal_recibo_ref';
        recibo := vec_catalogos_configurables.terminar_uso_con_evidencia(
            p_material->>'consumidor',p_material->>'uso_ref',p_material->>'reserva_recibo_ref',
            estado_esperado,a.actor_ref,a.decision_ref,esperado,a.motivo_ref,
            p_material->>'evidencia_ref',p_material->>'evidencia_sha256');
    END IF;
    r := vec_catalogos_configurables.obtener_uso_publicacion(
        p_material->>'consumidor',p_material->>'uso_ref',p_material->>'reserva_recibo_ref');
    u := r#>'{datos,uso}'; pub := r#>'{datos,publicacion}';
    IF r->>'encontrado' IS DISTINCT FROM 'true'
       OR pg_catalog.jsonb_typeof(u) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(pub) IS DISTINCT FROM 'object'
       OR recibo IS DISTINCT FROM esperado
       OR u->>'consumidor' IS DISTINCT FROM p_material->>'consumidor'
       OR u->>'uso_ref' IS DISTINCT FROM p_material->>'uso_ref'
       OR u->>'reserva_recibo_ref' IS DISTINCT FROM p_material->>'reserva_recibo_ref'
       OR u->>'categoria_id' IS DISTINCT FROM p_material->>'categoria_id'
       OR u->>'catalogo_id' IS DISTINCT FROM p_material->>'catalogo_id'
       OR u->>'version' IS DISTINCT FROM p_material->>'version'
       OR u->>'huella_sha256' IS DISTINCT FROM p_material->>'huella_sha256'
       OR pub->>'catalogo_id' IS DISTINCT FROM p_material->>'catalogo_id'
       OR pub->>'version' IS DISTINCT FROM p_material->>'version'
       OR pub->>'huella_sha256' IS DISTINCT FROM p_material->>'huella_sha256'
       OR pg_catalog.jsonb_typeof(pub->'documento_canonico') IS DISTINCT FROM 'string'
       OR pub->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(pub->>'documento_canonico','UTF8')),'hex') THEN
        RAISE EXCEPTION 'AD3-126: uso o publicacion incoherente' USING ERRCODE='55000';
    END IF;
    IF p_accion='vec.catalogos.categorias.reservar_uso' THEN
        IF u->>'estado' NOT IN ('reservado','confirmado','cancelado') THEN
            RAISE EXCEPTION 'AD3-126: reserva sin estado' USING ERRCODE='55000';
        END IF;
    ELSIF u->>'estado' IS DISTINCT FROM estado_esperado
       OR u->>'terminal_recibo_ref' IS DISTINCT FROM esperado THEN
        RAISE EXCEPTION 'AD3-126: terminal incoherente' USING ERRCODE='55000';
    END IF;
    BEGIN doc := (pub->>'documento_canonico')::jsonb;
    EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-126: canon ilegible' USING ERRCODE='55000'; END;
    IF pg_catalog.jsonb_typeof(doc->'id') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(doc->'modulo_id') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(doc->'version') IS DISTINCT FROM 'number'
       OR doc->>'id' IS DISTINCT FROM p_material->>'catalogo_id'
       OR doc->>'modulo_id' IS DISTINCT FROM p_material->>'modulo_id'
       OR doc->>'version' IS DISTINCT FROM p_material->>'version' THEN
        RAISE EXCEPTION 'AD3-126: modulo de publicacion ajeno' USING ERRCODE='42501';
    END IF;
    respuesta := pg_catalog.jsonb_build_object(
        'decision_ref',a.decision_ref,'efecto_ref',a.efecto_ref,
        'huella_efecto_sha256',a.huella_efecto_sha256,
        'consumo_huella_sha256',a.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,
        'consumida_en',pg_catalog.to_char(a.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'consumo_nuevo',a.consumo_nuevo,'recibo_ref',recibo,'uso',u);
    IF pg_catalog.octet_length(respuesta::text)>50331648 THEN
        RAISE EXCEPTION 'AD3-126: respuesta demasiado grande' USING ERRCODE='54000';
    END IF;
    RETURN respuesta;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.reservar_uso',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.confirmar_uso',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.cancelar_uso',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(
    jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
    text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor,vec_bolsa_llamamientos_ejecutor,vec_personal_ejecutor;
DO $acl$
DECLARE f regprocedure;
BEGIN
    FOREACH f IN ARRAY ARRAY[
      'vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
           AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef
           AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
            WHERE p.oid=f AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable
              OR (pg_catalog.strpos(f::text,'_interna(')>0 AND a.grantee<>p.proowner)
              OR (pg_catalog.strpos(f::text,'_atestada(')>0 AND a.grantee NOT IN (p.proowner,
                'vec_contratacion_temporal_ejecutor'::regrole,
                'vec_bolsa_llamamientos_ejecutor'::regrole,
                'vec_personal_ejecutor'::regrole)))) THEN
            RAISE EXCEPTION 'AD3-126: ACL de fachada incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
COMMIT;
