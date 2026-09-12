-- Evaluación histórica: documentos originales, nunca autoridad vigente.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:lectura_evaluacion_historica_v3:000012',0));
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';

DO $pre$
DECLARE t oid; f oid; firma text;
 r oid:='vec_autorizacion_evaluacion_historica_lector'::regrole;
 o oid:='vec_autorizacion_propietario'::regrole;
BEGIN
 IF getdatabaseencoding()<>'UTF8'
 OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_autorizacion'::regnamespace AND proname='leer_evaluacion_original_contexto_actor_v3')
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid='vec_autorizacion'::regnamespace AND nspowner=o)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=o AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
   AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_autorizacion'::regnamespace AND a.grantee=r)
 OR (SELECT count(*) FROM pg_class WHERE relnamespace='vec_autorizacion'::regnamespace AND relkind='r'
   AND relowner=o AND relname IN('asignacion_perfil','version_rol','control_vigencia_version_rol','politica_restrictiva'))<>4
 THEN RAISE EXCEPTION 'precondiciones evaluacion historica no satisfechas' USING ERRCODE='55000'; END IF;
 t:='vec_autorizacion.decision_concedida_contexto_actor_v3'::regclass;
 IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t AND relkind='r' AND relpersistence='p' AND NOT relispartition
  AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity
  AND obj_description(oid,'pg_class')='vec_autorizacion:registro-contexto-actor-v3:000005')
 OR EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid=t OR inhparent=t)
 OR EXISTS (SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid=t AND (a.grantee<>c.relowner OR a.grantor<>c.relowner))
 -- AD3/roles_up concede sólo REFERENCES(decision_ref) para su FK.
 -- El GRANT de superusuario se registra con el propietario de la tabla
 -- como grantor. No requiere AD3 instalado ni crea/repara sus privilegios.
 OR EXISTS (SELECT 1 FROM pg_attribute at WHERE at.attrelid=t AND at.attnum>0
  AND at.attacl IS NOT NULL AND NOT (
   at.attname='decision_ref' AND NOT at.attisdropped
   AND EXISTS (SELECT 1 FROM pg_roles receptor
    WHERE receptor.rolname='vec_autorizacion_atestada_v3_propietario'
     AND at.attacl=ARRAY[makeaclitem(receptor.oid,current_user::regrole::oid,'REFERENCES',false)])))
 OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='acceso_propietario_exacto' AND polcmd='*' AND polpermissive
  AND polroles=ARRAY[current_user::regrole::oid]
  AND pg_get_expr(polqual,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
  AND pg_get_expr(polwithcheck,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)') THEN
  RAISE EXCEPTION 'auth12: fuente propietaria incompatible' USING ERRCODE='55000'; END IF;

 FOREACH firma IN ARRAY ARRAY[
 'vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)',
 'vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)',
 'vec_autorizacion.manifiesto_politicas_v3_canonico(jsonb)',
 'vec_autorizacion.ambitos_positivos_validos(jsonb)',
 'vec_autorizacion.concesiones_positivas_validas(jsonb)',
 'vec_autorizacion.instante_utc_microsegundo_valido(text)'] LOOP
   f:=to_regprocedure(firma);
   IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=o
     AND prokind='f' AND provolatile='i' AND NOT prosecdef)
   THEN RAISE EXCEPTION 'helper propietario requerido' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(
 p_decision_ref text,p_decision_sha256 text,p_solicitud_sha256 text
) RETURNS TABLE(
 documento_asignacion jsonb,documento_rol jsonb,documento_control_rol jsonb,
 revision_catalogo text,huella_catalogo text,documentos_politicas jsonb
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET row_security=on
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE

    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;

    original vec_autorizacion.decision_concedida_contexto_actor_v3%ROWTYPE;
    d jsonb; v jsonb; asignacion record; rol record; control record; politica record;
    entrada jsonb; manifiesto jsonb:='[]'::jsonb; documentos jsonb:='[]'::jsonb;
    ids text[]:=ARRAY[]::text[]; total bigint:=0;
    inicio timestamptz:=clock_timestamp();
BEGIN
    IF current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'on'
       OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'evaluacion historica requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000';
    END IF;

    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_autorizacion_evaluacion_historica_lector';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_autorizacion';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)')];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 1 OR array_position(funciones,NULL) IS NOT NULL
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
           SELECT count(*)=3 AND bool_and(
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
           SELECT count(*)=1 AND count(DISTINCT p.oid)=1
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR (pg_catalog.has_database_privilege(login_oid,base_oid,'CONNECT')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'CREATE')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'TEMPORARY')
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_namespace n
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND ((n.oid <> esquema_oid
                  AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE'))
                 OR pg_catalog.has_schema_privilege(login_oid,n.oid,'CREATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind IN ('r','p','v','m','f') AND (
              pg_catalog.has_table_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'DELETE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRUNCATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'REFERENCES') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRIGGER') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'MAINTAIN') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'REFERENCES'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind='S' AND (
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'USAGE') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'UPDATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_proc p
         JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
            AND p.oid <> ALL(funciones)
            AND pg_catalog.has_function_privilege(login_oid,p.oid,'EXECUTE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_type t
         JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND t.typtype IN ('c','d','e','m','r')
            AND pg_catalog.has_type_privilege(login_oid,t.oid,'USAGE')
            -- PostgreSQL atribuye USAGE implícito al tipo fila de una tabla.
            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
              AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_class relacion
                 WHERE relacion.oid=t.typrelid
                   AND relacion.relkind IN ('r','p','v','m','f')
              )
            )
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_largeobject_metadata l
          WHERE pg_catalog.has_largeobject_privilege(login_oid,l.oid,'SELECT')
             OR pg_catalog.has_largeobject_privilege(login_oid,l.oid,'UPDATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(login_oid,f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server s
          WHERE pg_catalog.has_server_privilege(login_oid,s.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_language l
          WHERE l.oid >= 16384
            AND pg_catalog.has_language_privilege(login_oid,l.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_tablespace t
          WHERE t.oid >= 16384
            AND pg_catalog.has_tablespace_privilege(login_oid,t.oid,'CREATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_parameter_acl a
          WHERE pg_catalog.has_parameter_privilege(login_oid,a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(login_oid,a.parname,'ALTER SYSTEM')
       )) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;

    IF vec_autorizacion.texto_ascii_visible_v3_valido(p_decision_ref,512) IS NOT TRUE
       OR p_decision_sha256 IS NULL OR p_decision_sha256 !~ '^[0-9a-f]{64}$'
       OR p_solicitud_sha256 IS NULL OR p_solicitud_sha256 !~ '^[0-9a-f]{64}$'
       OR p_solicitud_sha256=repeat('0',64) THEN
        RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
    PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion:migracion:lectura_evaluacion_historica_v3:000012',0));
    SELECT r.* INTO STRICT original FROM vec_autorizacion.decision_concedida_contexto_actor_v3 r
      WHERE r.decision_ref=p_decision_ref AND r.huella_decision_sha256=p_decision_sha256
        AND r.documento->>'solicitud_huella_sha256'=p_solicitud_sha256;
    d:=original.documento;
    IF octet_length(original.decision_canonica) NOT BETWEEN 1 AND 524288
       OR vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
       OR d->>'concedida' IS DISTINCT FROM 'true'
       OR d->>'decision_ref' IS DISTINCT FROM p_decision_ref
       OR encode(sha256(original.decision_canonica),'hex') IS DISTINCT FROM p_decision_sha256
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM original.decision_canonica
       OR d->>'asignacion_ref' IS DISTINCT FROM original.asignacion_ref
       OR d->>'version_rol_ref' IS DISTINCT FROM original.version_rol_ref
       OR (d->>'control_vigencia_version_rol_revision')::numeric IS DISTINCT FROM original.control_vigencia_version_rol_revision
       OR (d->>'revision_catalogo_politicas')::numeric IS DISTINCT FROM original.revision_catalogo_politicas
       OR (d->>'emitida_en')::timestamptz IS DISTINCT FROM original.emitida_en
       OR (d->>'valida_hasta')::timestamptz IS DISTINCT FROM original.valida_hasta
       OR NOT isfinite(original.registrada_en)
       OR original.registrada_en < original.emitida_en OR original.registrada_en >= original.valida_hasta THEN
        RAISE EXCEPTION 'decision historica divergente' USING ERRCODE='22023';
    END IF;
    v:=d->'vinculo_autenticacion_actor';
    IF v->>'registro_contexto_ref' IS DISTINCT FROM original.registro_contexto_ref
       OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM original.contexto_actor_huella_sha256
       OR v->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM original.manifiesto_procedencia_huella_sha256 THEN
        RAISE EXCEPTION 'vinculo historico divergente' USING ERRCODE='22023';
    END IF;
    SELECT a.* INTO STRICT asignacion FROM vec_autorizacion.asignacion_perfil a
      WHERE a.asignacion_ref=original.asignacion_ref
        AND a.huella_sha256=d->>'asignacion_huella_sha256'
        AND a.principal_id=d->>'principal_id' AND a.perfil_activo_ref=d->>'perfil_activo_ref'
        AND a.version_rol_ref=original.version_rol_ref;
    SELECT r.* INTO STRICT rol FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=original.version_rol_ref AND r.huella_sha256=d->>'version_rol_huella_sha256';
    SELECT c.* INTO STRICT control FROM vec_autorizacion.control_vigencia_version_rol c
      WHERE c.version_rol_ref=original.version_rol_ref
        AND c.version_rol_ref=d->>'control_vigencia_version_rol_ref'
        AND c.revision=original.control_vigencia_version_rol_revision
        AND c.huella_sha256=d->>'control_vigencia_version_rol_huella_sha256';
    -- Helpers de estructura existentes; las huellas documentales son columnas
    -- propietarias ligadas a decisión. No inventar hash de jsonb::text.
    -- Cota del transporte completo: tres objetos y el array de políticas.
    -- Los dos bytes iniciales son []; cada separador JSONB posterior es ', '.
    total:=octet_length(asignacion.documento::text)::bigint
      +octet_length(rol.documento::text)::bigint
      +octet_length(control.documento::text)::bigint+2;
    IF total>33554432 THEN
        RAISE EXCEPTION 'documentos historicos exceden cota' USING ERRCODE='22023';
    END IF;
    IF vec_autorizacion.ambitos_positivos_validos(asignacion.documento) IS NOT TRUE
       OR vec_autorizacion.concesiones_positivas_validas(rol.documento) IS NOT TRUE
       OR asignacion.documento->>'asignacion_id' IS DISTINCT FROM asignacion.asignacion_id
       OR (asignacion.documento->>'version')::bigint IS DISTINCT FROM asignacion.version
       OR asignacion.asignacion_ref IS DISTINCT FROM 'asignacion:'||asignacion.asignacion_id||':v'||asignacion.version::text
       OR asignacion.documento->>'principal_id' IS DISTINCT FROM asignacion.principal_id
       OR asignacion.documento->>'perfil_activo_ref' IS DISTINCT FROM asignacion.perfil_activo_ref
       OR asignacion.documento->>'version_rol_ref' IS DISTINCT FROM rol.version_rol_ref
       OR rol.documento->>'rol_id' IS DISTINCT FROM rol.rol_id
       OR (rol.documento->>'version')::bigint IS DISTINCT FROM rol.version
       OR rol.version_rol_ref IS DISTINCT FROM 'rol:'||rol.rol_id||':v'||rol.version::text
       OR control.documento->>'version_rol_ref' IS DISTINCT FROM rol.version_rol_ref
       OR (control.documento->>'revision')::numeric IS DISTINCT FROM control.revision
       OR control.documento->>'estado' IS DISTINCT FROM control.estado THEN
        RAISE EXCEPTION 'documentos historicos divergentes' USING ERRCODE='22023';
    END IF;
    IF vec_autorizacion.instante_utc_microsegundo_valido(asignacion.documento->>'emitida_en') IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(asignacion.documento->>'vigente_desde') IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(asignacion.documento->>'vigente_hasta') IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(rol.documento->>'publicada_en') IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(control.documento->>'actualizado_en') IS NOT TRUE
       OR (asignacion.documento->>'emitida_en')::timestamptz IS DISTINCT FROM asignacion.emitida_en
       OR (rol.documento->>'publicada_en')::timestamptz IS DISTINCT FROM rol.publicada_en
       OR (control.documento->>'actualizado_en')::timestamptz IS DISTINCT FROM control.actualizado_en
       OR (asignacion.documento->>'vigente_desde')::timestamptz < asignacion.emitida_en
       OR (asignacion.documento->>'vigente_hasta')::timestamptz <= (asignacion.documento->>'vigente_desde')::timestamptz
       OR control.actualizado_en < rol.publicada_en
       OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
       OR rol.documento->>'estado' IS DISTINCT FROM 'publicada' OR control.estado<>'habilitada'
       OR original.registrada_en < (asignacion.documento->>'vigente_desde')::timestamptz
       OR original.registrada_en >= (asignacion.documento->>'vigente_hasta')::timestamptz
       OR rol.publicada_en > original.registrada_en THEN
        RAISE EXCEPTION 'cronologia historica divergente' USING ERRCODE='22023';
    END IF;
    -- TODAS las evaluadas, incluidas no aplicables o vencidas entonces.
    -- No filtrar por estado/ventana ACTUALES ni por politicas_aplicables.
    FOR entrada IN SELECT e.value FROM jsonb_array_elements(d->'politicas_evaluadas')
      WITH ORDINALITY AS e(value,numero) ORDER BY e.numero LOOP
        SELECT p.* INTO STRICT politica FROM vec_autorizacion.politica_restrictiva p
          WHERE p.politica_ref=entrada->>'referencia' AND p.huella_sha256=entrada->>'huella_sha256';
        IF politica.politica_id=ANY(ids)
           OR politica.documento->>'politica_id' IS DISTINCT FROM politica.politica_id
           OR (politica.documento->>'version')::bigint IS DISTINCT FROM politica.version
           OR politica.politica_ref IS DISTINCT FROM 'politica:'||politica.politica_id||':v'||politica.version::text
           OR vec_autorizacion.instante_utc_microsegundo_valido(politica.documento->>'publicada_en') IS NOT TRUE
           OR vec_autorizacion.instante_utc_microsegundo_valido(politica.documento->>'vigente_desde') IS NOT TRUE
           OR vec_autorizacion.instante_utc_microsegundo_valido(politica.documento->>'vigente_hasta') IS NOT TRUE
           OR (politica.documento->>'publicada_en')::timestamptz IS DISTINCT FROM politica.publicada_en
           OR (politica.documento->>'vigente_desde')::timestamptz < politica.publicada_en
           OR (politica.documento->>'vigente_hasta')::timestamptz <= (politica.documento->>'vigente_desde')::timestamptz THEN
            RAISE EXCEPTION 'politica historica divergente' USING ERRCODE='22023';
        END IF;
        total:=total+octet_length(politica.documento::text)::bigint
          +(CASE WHEN cardinality(ids)>0 THEN 2 ELSE 0 END);
        IF total>33554432 THEN
            RAISE EXCEPTION 'politicas historicas exceden cota' USING ERRCODE='22023';
        END IF;
        ids:=array_append(ids,politica.politica_id);
        manifiesto:=manifiesto||jsonb_build_array(jsonb_build_object(
          'referencia',politica.politica_ref,'huella_sha256',politica.huella_sha256));
        documentos:=documentos||jsonb_build_array(politica.documento);
    END LOOP;
    IF octet_length(asignacion.documento::text)::bigint
       +octet_length(rol.documento::text)::bigint
       +octet_length(control.documento::text)::bigint
       +octet_length(documentos::text)::bigint>33554432 THEN
        RAISE EXCEPTION 'documentos historicos exceden cota' USING ERRCODE='22023';
    END IF;
    IF jsonb_array_length(documentos)<>jsonb_array_length(d->'politicas_evaluadas')
       OR manifiesto IS DISTINCT FROM d->'politicas_evaluadas'
       OR encode(sha256(convert_to(vec_autorizacion.manifiesto_politicas_v3_canonico(manifiesto),'UTF8')),'hex')
          IS DISTINCT FROM d->>'catalogo_politicas_huella_sha256' THEN
        RAISE EXCEPTION 'catalogo historico divergente' USING ERRCODE='22023';
    END IF;
    IF clock_timestamp()>inicio+interval '5 seconds' THEN
        RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
    END IF;
    RETURN QUERY SELECT asignacion.documento,rol.documento,control.documento,
      original.revision_catalogo_politicas::text,d->>'catalogo_politicas_huella_sha256',documentos;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'evaluacion historica no disponible' USING ERRCODE='P0002';
    WHEN data_exception THEN
        RAISE EXCEPTION 'evaluacion historica invalida' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_evaluacion_historica_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)
 TO vec_autorizacion_evaluacion_historica_lector;
DO $acl$
DECLARE f oid:='vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)'::regprocedure;
 o oid:='vec_autorizacion_propietario'::regrole; r oid:='vec_autorizacion_evaluacion_historica_lector'::regrole;
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
 AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'ACL historica de creacion inesperada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
