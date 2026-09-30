\set ON_ERROR_STOP on
-- AD3-134: gobierno nominal del catalogo comun RPT. Requiere
-- catalogos_configurables 000004 y AD3-126. El DBA crea el rol sin miembros.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
DO $dba$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regrole('vec_catalogos_configurables_gobierno_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-134: aprovisionamiento DBA incompatible' USING ERRCODE='55000'; END IF;
END $dba$;
CREATE ROLE vec_catalogos_configurables_gobierno_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000134',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text)','EXECUTE')
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_catalogos_configurables_gobierno_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_catalogos_configurables_gobierno_ejecutor'::regrole OR roleid='vec_catalogos_configurables_gobierno_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'AD3-134: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La funcion comun conserva OID, ACL, propietario, configuracion, dependencias
-- y todos los caminos anteriores. Solo se insertan un perfil tecnico nuevo y
-- su lista positiva de acciones/campos/audiencia sobre marcas exactas.
DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
    propietario oid; config text[]; definidora boolean;
    exclusor text := E'               AND p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion_v2'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_categorias'' AND p_perfil_mutacion IS DISTINCT FROM ''usos_categorias''';
    cierre_roles text := E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',\n            MESSAGE = ''consumo VEC-AD-3 rechazado'';';
    rol_nuevo text := $x$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_categorias'
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole
                    AND m.roleid='vec_catalogos_configurables_gobierno_ejecutor'::pg_catalog.regrole
                    AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
                    AND pg_catalog.pg_has_role(session_user,m.roleid,'MEMBER'))=1
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_catalogos_configurables_gobierno_ejecutor'::pg_catalog.regrole)
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE pg_catalog.left(r.rolname,4)='vec_'
                    AND r.rolname<>session_user AND r.rolname<>'vec_catalogos_configurables_gobierno_ejecutor'
                    AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
$x$;
    marca_capacidad text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    capacidad_nueva text := $x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.gobierno_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.gobierno.proponer',
                                   'vec.catalogos.categorias.gobierno.aprobar',
                                   'vec.catalogos.categorias.gobierno.confirmar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_categoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_categorias_rpt'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["gobierno","recibo"]'::jsonb
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
       OR pg_catalog.strpos(original,'lectura_categorias')=0
       OR pg_catalog.strpos(original,'gobierno_categorias')<>0
       OR pg_catalog.strpos(original,'usos_categorias')=0 THEN
        RAISE EXCEPTION 'AD3-134: núcleo V3 incompatible' USING ERRCODE='55000';
    END IF;
    nuevo := pg_catalog.replace(original,exclusor,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''gobierno_categorias''');
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
        RAISE EXCEPTION 'AD3-134: núcleo V3 alterado fuera de contrato' USING ERRCODE='55000';
    END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE
    definicion text;
    nueva text := 'vec_catalogos_configurables.gobierno_categorias.v1';
BEGIN
    SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
    IF pg_catalog.strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1
       OR pg_catalog.right(definicion,3)<>']))'
       OR pg_catalog.strpos(definicion,'vec_catalogos_configurables.usos_categorias.v1')=0
       OR pg_catalog.strpos(definicion,pg_catalog.quote_literal(nueva))<>0 THEN
        RAISE EXCEPTION 'AD3-134: audiencias previas incompatibles' USING ERRCODE='55000';
    END IF;
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
        ||pg_catalog.left(definicion,pg_catalog.length(definicion)-3)||', '||pg_catalog.quote_literal(nueva)||'::text]))';
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(
    p_material jsonb,p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
    p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,
    consumida_en timestamptz,actor_ref text,motivo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; m jsonb; ca jsonb; x record; contenido jsonb; alcance jsonb;
    material_h text; contexto_h text; ambitos text; motivo_canonico text;
BEGIN
    IF p_material IS NULL OR pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR pg_catalog.octet_length(p_material::text)>18874368
       OR p_accion NOT IN ('vec.catalogos.categorias.gobierno.proponer',
           'vec.catalogos.categorias.gobierno.aprobar','vec.catalogos.categorias.gobierno.confirmar')
       OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_material))<>
           (CASE WHEN p_accion='vec.catalogos.categorias.gobierno.proponer' THEN 4 ELSE 6 END)
       OR pg_catalog.jsonb_typeof(p_material->'propuesta_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'propuesta_ref') NOT BETWEEN 3 AND 160
       OR ((p_material->>'propuesta_ref') COLLATE "C") !~ '^[!-~]{3,160}$'
       OR pg_catalog.strpos(p_material->>'propuesta_ref','*')>0
       OR pg_catalog.jsonb_typeof(p_material->'recibo_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.octet_length(p_material->>'recibo_ref') NOT BETWEEN 3 AND 160
       OR ((p_material->>'recibo_ref') COLLATE "C") !~ '^[!-~]{3,160}$'
       OR pg_catalog.jsonb_typeof(p_material->'huella_sha256') IS DISTINCT FROM 'string'
       OR p_material->>'huella_sha256' !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'AD3-134: material invalido' USING ERRCODE='22023';
    END IF;
    IF p_accion='vec.catalogos.categorias.gobierno.proponer' THEN
        contenido:=p_material->'contenido';
        IF pg_catalog.jsonb_typeof(contenido) IS DISTINCT FROM 'object'
           OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(contenido))<>12
           OR contenido->>'accion' NOT IN ('publicar','deshabilitar')
           OR pg_catalog.jsonb_typeof(contenido->'catalogo_id') IS DISTINCT FROM 'string'
           OR contenido->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR pg_catalog.jsonb_typeof(contenido->'modulo_id') IS DISTINCT FROM 'string'
           OR contenido->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR pg_catalog.jsonb_typeof(contenido->'version') IS DISTINCT FROM 'number'
           OR contenido->>'version' !~ '^[1-9][0-9]{0,9}$'
           OR (contenido->>'version')::numeric>2147483647
           OR pg_catalog.jsonb_typeof(contenido->'preimagenes_control') IS DISTINCT FROM 'object'
           OR pg_catalog.jsonb_typeof(contenido->'preimagenes_huella_sha256') IS DISTINCT FROM 'string'
           OR contenido->>'preimagenes_huella_sha256' !~ '^[0-9a-f]{64}$'
           OR pg_catalog.jsonb_typeof(contenido->'motivo_ref') IS DISTINCT FROM 'string'
           OR pg_catalog.jsonb_typeof(contenido->'fuente_ref') IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(contenido->>'fuente_ref') NOT BETWEEN 3 AND 320
           OR p_material->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(
              pg_catalog.sha256(pg_catalog.convert_to(contenido::text,'UTF8')),'hex') THEN
            RAISE EXCEPTION 'AD3-134: contenido invalido' USING ERRCODE='22023';
        END IF;
        IF pg_catalog.jsonb_typeof(contenido->'documento_canonico') IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(contenido->>'documento_canonico') NOT BETWEEN 2 AND 16777216
           OR pg_catalog.jsonb_typeof(contenido->'documento_huella_sha256') IS DISTINCT FROM 'string'
           OR contenido->>'documento_huella_sha256' IS DISTINCT FROM pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(contenido->>'documento_canonico','UTF8')),'hex') THEN
            RAISE EXCEPTION 'AD3-134: documento completo invalido' USING ERRCODE='22023';
        END IF;
        IF contenido->>'accion'='publicar' THEN
            IF contenido->'categoria_id' IS DISTINCT FROM 'null'::jsonb OR contenido->'revision_esperada' IS DISTINCT FROM 'null'::jsonb THEN
                RAISE EXCEPTION 'AD3-134: publicacion invalida' USING ERRCODE='22023';
            END IF;
        ELSE
            IF pg_catalog.jsonb_typeof(contenido->'categoria_id') IS DISTINCT FROM 'string'
               OR contenido->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
               OR pg_catalog.jsonb_typeof(contenido->'revision_esperada') IS DISTINCT FROM 'number'
               OR contenido->>'revision_esperada' !~ '^[1-9][0-9]{0,17}$'
               OR (contenido->>'version')::integer<2
               OR NOT (contenido->'preimagenes_control' ? (contenido->>'categoria_id')) THEN
                RAISE EXCEPTION 'AD3-134: deshabilitacion invalida' USING ERRCODE='22023';
            END IF;
        END IF;
        IF contenido->>'preimagenes_huella_sha256' IS DISTINCT FROM pg_catalog.encode(
           pg_catalog.sha256(pg_catalog.convert_to((contenido->'preimagenes_control')::text,'UTF8')),'hex') THEN
            RAISE EXCEPTION 'AD3-134: preimagenes invalidas' USING ERRCODE='22023';
        END IF;
    ELSE
        IF pg_catalog.jsonb_typeof(p_material->'catalogo_id') IS DISTINCT FROM 'string'
           OR p_material->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR pg_catalog.jsonb_typeof(p_material->'modulo_id') IS DISTINCT FROM 'string'
           OR p_material->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
           OR pg_catalog.jsonb_typeof(p_material->'revision_esperada') IS DISTINCT FROM 'number'
           OR p_material->>'revision_esperada' !~ '^[1-9][0-9]{0,18}$'
           OR (p_material->>'revision_esperada')::numeric>9223372036854775807::numeric THEN
            RAISE EXCEPTION 'AD3-134: transicion invalida' USING ERRCODE='22023';
        END IF;
    END IF;
    BEGIN
        c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
        d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
        m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
        ca:=pg_catalog.convert_from(p_contexto,'UTF8')::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'AD3-134: atestacion invalida' USING ERRCODE='22023';
    END;
    alcance:=CASE WHEN contenido IS NULL THEN p_material ELSE contenido END;
    material_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material::text,'UTF8')),'hex');
    ambitos:='{"ambitos":{"catalogo_id":"'||(alcance->>'catalogo_id')||
        '","modulo_id":"'||(alcance->>'modulo_id')||
        '"},"atributos":{"material_sha256":"'||material_h||'"}}';
    contexto_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ambitos,'UTF8')),'hex');
    IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.gobierno_categorias.v1'
       OR c->>'operacion' IS DISTINCT FROM p_accion
       OR d->>'accion' IS DISTINCT FROM p_accion
       OR d->>'modulo_id' IS DISTINCT FROM alcance->>'modulo_id'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'propuesta_categoria'
       OR d->>'finalidad' IS DISTINCT FROM 'gobernar_categorias_rpt'
       OR d->>'recurso_ref' IS DISTINCT FROM p_material->>'propuesta_ref'
       OR c->>'efecto_ref' IS DISTINCT FROM p_material->>'propuesta_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
       OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
       OR d->'campos_permitidos' IS DISTINCT FROM '["gobierno","recibo"]'::jsonb
       OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION 'AD3-134: gobierno no autorizado' USING ERRCODE='42501';
    END IF;
    IF pg_catalog.jsonb_typeof(ca) IS DISTINCT FROM 'object'
       OR ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
       OR pg_catalog.jsonb_typeof(ca->'persona_ref') IS DISTINCT FROM 'string'
       OR ca->>'persona_ref' IS DISTINCT FROM d->>'principal_id'
       OR ca->>'principal_ref' IS DISTINCT FROM ca->>'persona_ref'
       OR ca->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' THEN
        RAISE EXCEPTION 'AD3-134: persona canonica incompatible' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'gobierno_categorias',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF x.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'AD3-134: decision historica denegada' USING ERRCODE='42501';
    END IF;
    IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(m->'referencia') IS DISTINCT FROM 'object'
       OR m->>'esquema' IS DISTINCT FROM 'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
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
       OR d->>'motivo_huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(p_motivo),'hex')
       OR pg_catalog.octet_length(d->>'principal_id') NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'AD3-134: motivo o actor invalidos' USING ERRCODE='42501';
    END IF;
    motivo_canonico:=(m#>>'{referencia,catalogo_id}')||':'||
        (m#>>'{referencia,catalogo_version}')||':'||(m#>>'{referencia,entrada_clave}');
    IF pg_catalog.octet_length(motivo_canonico) NOT BETWEEN 3 AND 320
       OR (contenido IS NOT NULL AND contenido->>'motivo_ref' IS DISTINCT FROM motivo_canonico) THEN
        RAISE EXCEPTION 'AD3-134: motivo ajeno' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
        x.consumo_huella_sha256,x.auditoria_ref,
        x.consumida_en,d->>'principal_id',motivo_canonico;
END $f$;

-- Una aprobación consumida es evidencia histórica. La decisión actual de
-- confirmación se consume aparte con vigencia positiva; no se recicla la anterior.
CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(
    p_aprobacion jsonb,p_propuesta_ref text,p_huella text,p_catalogo_id text,p_modulo_id text
) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE t record; c jsonb; d jsonb; material jsonb; material_h text; ambitos text; contexto_h text;
BEGIN
    IF pg_catalog.jsonb_typeof(p_aprobacion) IS DISTINCT FROM 'object'
       OR p_aprobacion->>'propuesta_ref' IS DISTINCT FROM p_propuesta_ref
       OR p_aprobacion->>'huella_sha256' IS DISTINCT FROM p_huella
       OR pg_catalog.jsonb_typeof(p_aprobacion->'decision_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_aprobacion->'actor_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_aprobacion->'recibo_ref') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(p_aprobacion->'ordinal') IS DISTINCT FROM 'number'
       OR p_aprobacion->>'ordinal' IS DISTINCT FROM '1'
       OR p_aprobacion->>'consumo_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR pg_catalog.jsonb_typeof(p_aprobacion->'auditoria_ref') IS DISTINCT FROM 'string' THEN
        RAISE EXCEPTION 'AD3-134: aprobacion historica incompatible' USING ERRCODE='42501';
    END IF;
    SELECT a.capacidad_canonica,a.decision_canonica,u.consumo_huella_sha256,v.auditoria_ref
      INTO STRICT t FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
      JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 u USING(decision_ref)
      JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 v USING(decision_ref)
     WHERE a.decision_ref=p_aprobacion->>'decision_ref' FOR SHARE OF a,u,v;
    BEGIN
        c:=pg_catalog.convert_from(t.capacidad_canonica,'UTF8')::jsonb;
        d:=pg_catalog.convert_from(t.decision_canonica,'UTF8')::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'AD3-134: aprobacion historica ilegible' USING ERRCODE='42501';
    END;
    material:=pg_catalog.jsonb_build_object('propuesta_ref',p_propuesta_ref,
        'huella_sha256',p_huella,'recibo_ref',p_aprobacion->>'recibo_ref',
        'revision_esperada',(p_aprobacion->>'ordinal')::bigint,
        'catalogo_id',p_catalogo_id,'modulo_id',p_modulo_id);
    material_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(material::text,'UTF8')),'hex');
    ambitos:='{"ambitos":{"catalogo_id":"'||p_catalogo_id||'","modulo_id":"'||p_modulo_id||
        '"},"atributos":{"material_sha256":"'||material_h||'"}}';
    contexto_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ambitos,'UTF8')),'hex');
    IF t.consumo_huella_sha256 IS DISTINCT FROM p_aprobacion->>'consumo_huella_sha256'
       OR t.auditoria_ref IS DISTINCT FROM p_aprobacion->>'auditoria_ref'
       OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.gobierno_categorias.v1'
       OR c->>'operacion' IS DISTINCT FROM 'vec.catalogos.categorias.gobierno.aprobar'
       OR c->>'efecto_ref' IS DISTINCT FROM p_propuesta_ref
       OR d->>'accion' IS DISTINCT FROM c->>'operacion'
       OR d->>'recurso_ref' IS DISTINCT FROM p_propuesta_ref
       OR d->>'principal_id' IS DISTINCT FROM p_aprobacion->>'actor_ref'
       OR d->>'modulo_id' IS DISTINCT FROM p_modulo_id
       OR d->>'tipo_recurso' IS DISTINCT FROM 'propuesta_categoria'
       OR d->>'finalidad' IS DISTINCT FROM 'gobernar_categorias_rpt'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
       OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
       OR d->'campos_permitidos' IS DISTINCT FROM '["gobierno","recibo"]'::jsonb
       OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION 'AD3-134: aprobacion historica sin evidencia exacta' USING ERRCODE='42501';
    END IF;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(
    p_accion text,p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,
    p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE a record; r jsonb; consulta jsonb; contenido jsonb; aprobaciones jsonb; aprobacion jsonb;
    actor_a text; actor_b text; esperado text; revision bigint;
BEGIN
    SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(
        p_material,p_accion,p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    esperado:=p_material->>'recibo_ref';
    IF p_accion='vec.catalogos.categorias.gobierno.proponer' THEN
        r:=vec_catalogos_configurables.registrar_propuesta_gobierno(
            p_material->>'propuesta_ref',p_material->'contenido',p_material->>'huella_sha256',
            a.actor_ref,a.decision_ref,esperado,a.motivo_ref);
    ELSE
        -- La consulta del propietario Cat serializa las transiciones de esta
        -- propuesta. Nunca se confía en catalogo/modulo enviados por el cliente.
        consulta:=vec_catalogos_configurables.consultar_aprobaciones_gobierno(
            p_material->>'propuesta_ref',p_material->>'huella_sha256');
        contenido:=consulta#>'{propuesta,contenido}';
        IF pg_catalog.jsonb_typeof(contenido) IS DISTINCT FROM 'object'
           OR contenido->>'catalogo_id' IS DISTINCT FROM p_material->>'catalogo_id'
           OR contenido->>'modulo_id' IS DISTINCT FROM p_material->>'modulo_id'
           OR contenido->>'motivo_ref' IS DISTINCT FROM a.motivo_ref
           OR p_material->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(contenido::text,'UTF8')),'hex')
           OR pg_catalog.jsonb_typeof(consulta#>'{propuesta,revision}') IS DISTINCT FROM 'number'
           OR consulta#>>'{propuesta,revision}' !~ '^[1-3]$'
           OR pg_catalog.jsonb_typeof(consulta->'aprobaciones') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'AD3-134: propuesta consultada incompatible' USING ERRCODE='42501';
        END IF;
        revision:=(p_material->>'revision_esperada')::bigint;
        IF p_accion='vec.catalogos.categorias.gobierno.aprobar' THEN
            r:=vec_catalogos_configurables.aprobar_propuesta_gobierno(
                p_material->>'propuesta_ref',p_material->>'huella_sha256',revision,
                a.actor_ref,a.decision_ref,a.auditoria_ref,a.consumo_huella_sha256,
                esperado,a.motivo_ref);
        ELSE
            aprobaciones:=consulta->'aprobaciones';
            IF revision<>2 OR consulta#>>'{propuesta,revision}' NOT IN ('2','3')
               OR pg_catalog.jsonb_array_length(aprobaciones)<>1 THEN
                RAISE EXCEPTION 'AD3-134: aprobacion exigida' USING ERRCODE='42501';
            END IF;
            aprobacion:=aprobaciones->0;
            actor_a:=consulta#>>'{propuesta,editor_ref}';
            actor_b:=aprobacion->>'actor_ref';
            IF aprobacion->>'ordinal' IS DISTINCT FROM '1'
               OR actor_a IS NOT DISTINCT FROM actor_b
               OR actor_b IS DISTINCT FROM a.actor_ref THEN
                RAISE EXCEPTION 'AD3-134: separacion o confirmador incompatible' USING ERRCODE='42501';
            END IF;
            PERFORM vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(
                aprobacion,p_material->>'propuesta_ref',p_material->>'huella_sha256',
                p_material->>'catalogo_id',p_material->>'modulo_id');
            r:=vec_catalogos_configurables.confirmar_propuesta_gobierno(
                p_material->>'propuesta_ref',p_material->>'huella_sha256',revision,
                a.actor_ref,a.decision_ref,a.auditoria_ref,esperado,a.motivo_ref);
        END IF;
    END IF;
    IF pg_catalog.jsonb_typeof(r) IS DISTINCT FROM 'object'
       OR r->>'propuesta_ref' IS DISTINCT FROM p_material->>'propuesta_ref'
       OR r->>'huella_sha256' IS DISTINCT FROM p_material->>'huella_sha256'
       OR r->>'recibo_ref' IS DISTINCT FROM esperado
       OR pg_catalog.jsonb_typeof(r->'revision') IS DISTINCT FROM 'number'
       OR pg_catalog.octet_length(r::text)>50331648 THEN
        RAISE EXCEPTION 'AD3-134: resultado Cat incoherente' USING ERRCODE='55000';
    END IF;
    -- Revalidación antes de devolver el efecto, incluido replay; revocación falla cerrado.
    IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
        p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL THEN
        RAISE EXCEPTION 'AD3-134: autorizacion revocada antes de efecto' USING ERRCODE='42501';
    END IF;
    RETURN pg_catalog.jsonb_build_object('decision_ref',a.decision_ref,
        'efecto_ref',a.efecto_ref,'huella_efecto_sha256',a.huella_efecto_sha256,
        'consumo_huella_sha256',a.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,
        'consumida_en',pg_catalog.to_char(a.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'consumo_nuevo',true,'gobierno',r,'recibo_ref',esperado);
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.gobierno.proponer',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.gobierno.aprobar',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(
    p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
    p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
BEGIN
    RETURN vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(
        'vec.catalogos.categorias.gobierno.confirmar',p_material,p_capacidad,p_decision,
        p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(
    jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(jsonb,text,text,text,text),
    vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(
    text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_catalogos_configurables_gobierno_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_catalogos_configurables_gobierno_ejecutor;
DO $acl$
DECLARE f regprocedure;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(jsonb,text,text,text,text)'::regprocedure,
  'vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
 ] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
      AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef
      AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable
       OR (pg_catalog.strpos(f::text,'_interna(')>0 AND a.grantee<>p.proowner)
       OR (pg_catalog.strpos(f::text,'_atestada(')>0 AND a.grantee NOT IN
           (p.proowner,'vec_catalogos_configurables_gobierno_ejecutor'::regrole)))) THEN
   RAISE EXCEPTION 'AD3-134: ACL de fachada incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
