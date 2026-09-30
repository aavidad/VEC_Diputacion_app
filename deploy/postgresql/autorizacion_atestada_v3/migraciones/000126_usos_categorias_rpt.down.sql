\set ON_ERROR_STOP on
-- Sólo para clon desechable sin claves ni decisiones de esta audiencia.
-- No ejecutar DOWN en instalaciones con historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000126',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
                   WHERE audiencia_consumo='vec_catalogos_configurables.usos_categorias.v1')
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'AD3-126: DOWN con historia o preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(
    text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(
    jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

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
    IF fuente_sha256 IS DISTINCT FROM '848799985debd0b736a3c281b0a3a3635e6182bd78789fb364121fe7f78d8a06'
       OR definicion_sha256 IS DISTINCT FROM '3a06d11882d7b1ed5d3256f0547debf672975dce3c27060118bf169ce6bed692'
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
        "search_path=pg_catalog, pg_temp",
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
       OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
       OR pg_catalog.strpos(original,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''usos_categorias''')=0
       OR pg_catalog.strpos(original,rol_nuevo||cierre_roles)=0
       OR pg_catalog.strpos(original,capacidad_nueva||marca_capacidad)=0 THEN
        RAISE EXCEPTION 'AD3-126: nucleo cambiado, DOWN denegado' USING ERRCODE='55000';
    END IF;
    nuevo := pg_catalog.replace(original,capacidad_nueva||marca_capacidad,marca_capacidad);
    nuevo := pg_catalog.replace(nuevo,rol_nuevo||cierre_roles,cierre_roles);
    nuevo := pg_catalog.replace(nuevo,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''usos_categorias''',exclusor);
    nuevo := pg_catalog.replace(nuevo,E'SET search_path TO ''pg_catalog'', ''pg_temp''',E'SET search_path TO ''pg_catalog''');
    EXECUTE nuevo;
    SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
    IF actual IS DISTINCT FROM nuevo
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual,'UTF8')),'hex')
            IS DISTINCT FROM '334d3a9d8397de1a37ca559ba12e0955510649da624b0ca3bdbab20a5729839f'
       OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
             FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '8cda19bc0ab03f811386f7ec6b54a4662b68988590538d2857f48691c8c3c842'
       OR (SELECT pg_catalog.to_jsonb(p)-'prosrc'-'proconfig' FROM pg_catalog.pg_proc AS p WHERE p.oid=f)
            IS DISTINCT FROM (meta-'prosrc'-'proconfig')
       OR 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid IS DISTINCT FROM f
       OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
       OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_shdepend AS d
            WHERE d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database())
              AND d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas THEN
        RAISE EXCEPTION 'AD3-126: nucleo alterado en DOWN' USING ERRCODE='55000';
    END IF;
END $nucleo$;

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
            IS DISTINCT FROM '4fef385ffcee91a94046b1dda4368d3b85630df12d251384fd9d723352c8fdb8'
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
    esperada := pg_catalog.replace(definicion,', '||pg_catalog.quote_literal(nueva)||'::text]))',$cierre$]))$cierre$);
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
            IS DISTINCT FROM 'cd92724aeb8e8baf8819be986ea23a215104619945944fc709352a5a0e5ec982'
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
COMMIT;
