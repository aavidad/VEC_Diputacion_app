\set ON_ERROR_STOP on
-- AD3-143: consumo nominal de órdenes y doble control de restauración ADMIN.
-- BORRADOR: falta contrato fijo de perfiles ADMIN y ratificación de preimagen.
-- Preimagen real post-AD145; conserva AD141/142/144/145 y el payload V3.
-- Las fachadas ADMIN quedan cerradas mientras falten sus fuentes nominales.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000143',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $roles$
DECLARE nombre text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD3-143: instalación privilegiada requerida' USING ERRCODE='42501'; END IF;
 FOREACH nombre IN ARRAY ARRAY['vec_administracion_copias_propietario','vec_administracion_copias_ejecutor','vec_administracion_copias_migrador'] LOOP
  IF to_regrole(nombre) IS NOT NULL THEN RAISE EXCEPTION 'AD3-143: rol técnico ya existe' USING ERRCODE='55000'; END IF;
  EXECUTE format('CREATE ROLE %I NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS',nombre);
 END LOOP;
 EXECUTE format('GRANT CONNECT,CREATE ON DATABASE %I TO vec_administracion_copias_propietario',current_database());
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_administracion_copias_ejecutor,vec_administracion_copias_migrador',current_database());
END $roles$;
GRANT vec_administracion_copias_propietario TO vec_administracion_copias_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_administracion_copias_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_administracion_copias_propietario REVOKE ALL ON TYPES FROM PUBLIC;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pre$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_orden_copias_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-143: orden causal incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Preimagen medida en PG18.4 tras AD145, sobre el núcleo real AD142/144.
 esperada_def_sha256 text:=$esperada_def_sha256$060fa1c5e51a8a2fa4a2dd8fd7ebfe705f79e96ab71666e2754a52c378a0d89e$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$993e82bef26142ba928de996f92632d8a06a5b459715f0090d64ca756bb25358$esperada_fuente_sha256$;
 marca text:=$marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
 excl text:=$excl$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$excl$;
 excl_nuevo text:=$excl_nuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'admin_copias_orden'
               AND p_perfil_mutacion IS DISTINCT FROM 'admin_copias_propuesta'
               AND p_perfil_mutacion IS DISTINCT FROM 'admin_copias_revision'
$excl_nuevo$;
 runtime text:=$runtime$           OR (
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
$runtime$;
 runtime_nuevo text:=$runtime_nuevo$           OR (
               p_perfil_mutacion IN ('admin_copias_orden','admin_copias_propuesta','admin_copias_revision')
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_administracion_copias_ejecutor'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_administracion_copias_ejecutor'::regrole)
           )
           OR (
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
$runtime_nuevo$;
 extension text:=$extension$           OR (
 p_perfil_mutacion IN ('admin_copias_orden','admin_copias_propuesta','admin_copias_revision')
 AND ((p_perfil_mutacion='admin_copias_orden'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_administracion_copias.comprometer_orden.v1'
       AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.copias.orden.emitir'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'orden_copia'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'emitir_orden_copia'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["orden","recibo"]'::jsonb)
   OR (p_perfil_mutacion='admin_copias_propuesta'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_administracion_copias.proponer_restauracion.v1'
       AND c->>'operacion' IS NOT DISTINCT FROM 'copias_restauracion_proponer'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_restauracion_copia'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'proponer_restauracion_copia'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["propuesta","recibo"]'::jsonb)
   OR (p_perfil_mutacion='admin_copias_revision'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_administracion_copias.revisar_restauracion.v1'
       AND c->>'operacion' IS NOT DISTINCT FROM 'copias_restauracion_revisar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_restauracion_copia'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_restauracion_copia'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["propuesta","recibo","revision"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS NOT DISTINCT FROM 'alto'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-143: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-143: metadatos de núcleo ausentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 -- Perfil nuevo en las dos listas del núcleo: exclusión del bloque general y
 -- selección de la guarda de sesión miembro exclusivo del ejecutor Copias.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_database db
         CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
         WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolcanlogin
         AND has_database_privilege(r.oid,current_database(),'TEMPORARY'))
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee=propietario AND a.grantor=propietario
           AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee<>propietario OR a.grantor<>propietario
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_language'::regclass::oid,
           'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
           'refobjsubid',0,'deptype','n'),
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_namespace'::regclass::oid,
           'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
           'refobjsubid',0,'deptype','n'))
    OR deps_compartidas IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
           'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR strpos(original,'admin_copias_orden')<>0
    OR strpos(original,'admin_copias_propuesta')<>0
    OR strpos(original,'admin_copias_revision')<>0
    OR strpos(original,'administracion.copias.orden.emitir')<>0
 THEN RAISE EXCEPTION 'AD3-143: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,excl,excl_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),excl_nuevo,excl),runtime_nuevo,runtime) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD3-143: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM '8f3d3c4b34e0e4904c2e132982347800e2cf34a9e5d5842c79988f83a11161f7'
 OR strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
 OR strpos(d,'vec_administracion_copias.')<>0
 THEN RAISE EXCEPTION 'AD3-143: preimagen de audiencias incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||left(d,length(d)-3)||', ''vec_administracion_copias.comprometer_orden.v1''::text, ''vec_administracion_copias.proponer_restauracion.v1''::text, ''vec_administracion_copias.revisar_restauracion.v1''::text]))';
END $audiencias$;
-- La aprobación depende de una fachada del propietario ADMIN que se instala
-- después de AD143. Propuesta/revisión no la invocan: así no hay ciclo.
-- El núcleo común permanece privado y conserva sus siete campos de salida.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_copias_v3_interna(
 p_perfil text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean,persona_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;ctx jsonb;v jsonb;consumo record;guard oid;doble oid;proveedor oid;
 audiencia text;accion text;tipo text;finalidad text;campos jsonb;rol_esperado text;
 -- Candidatas comunicadas por E el 02/10/2026: Sistemas operador_plataforma:v1;
 -- Aplicación administracion_perfiles:v3. No son publicación ni concesión.
 -- La fuente definitiva debe resolver plantilla fija y versión publicada viva;
 -- no usar estas versiones candidatas ni un SHA de archivo como autoridad.
 -- Estos NULL mantienen cerrado el borrador mientras falta el contrato central.
 rol_sistemas_version_ref CONSTANT text:=NULL;
 rol_aplicacion_version_ref CONSTANT text:=NULL;
BEGIN
 IF p_perfil='admin_copias_orden' THEN
  audiencia:='vec_administracion_copias.comprometer_orden.v1';accion:='administracion.copias.orden.emitir';
  tipo:='orden_copia';finalidad:='emitir_orden_copia';campos:='["orden","recibo"]'::jsonb;
  rol_esperado:=rol_sistemas_version_ref;
 ELSIF p_perfil='admin_copias_propuesta' THEN
  audiencia:='vec_administracion_copias.proponer_restauracion.v1';accion:='copias_restauracion_proponer';
  tipo:='propuesta_restauracion_copia';finalidad:='proponer_restauracion_copia';campos:='["propuesta","recibo"]'::jsonb;
  rol_esperado:=rol_sistemas_version_ref;
 ELSIF p_perfil='admin_copias_revision' THEN
  audiencia:='vec_administracion_copias.revisar_restauracion.v1';accion:='copias_restauracion_revisar';
  tipo:='propuesta_restauracion_copia';finalidad:='revisar_restauracion_copia';campos:='["propuesta","recibo","revision"]'::jsonb;
  rol_esperado:=rol_aplicacion_version_ref;
 ELSE RAISE EXCEPTION 'AD3-143: perfil de consumo denegado' USING ERRCODE='42501'; END IF;
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
 THEN RAISE EXCEPTION 'AD3-143: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
       ctx:=convert_from(p_contexto,'UTF8')::jsonb;v:=d->'vinculo_autenticacion_actor';
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'AD3-143: material inválido' USING ERRCODE='22023'; END;
 IF p_perfil='admin_copias_revision'
 AND d->>'version_rol_ref' IS NOT DISTINCT FROM 'rol:operador_plataforma:v1'
 THEN RAISE EXCEPTION 'AD3-143: sistemas no aprueba restauración' USING ERRCODE='42501'; END IF;
 IF rol_esperado IS NULL OR rol_sistemas_version_ref IS NOT DISTINCT FROM rol_aplicacion_version_ref
 THEN RAISE EXCEPTION 'AD3-143: contrato de perfiles ADMIN pendiente' USING ERRCODE='55000'; END IF;
 IF c->>'audiencia_consumo' IS DISTINCT FROM audiencia OR c->>'operacion' IS DISTINCT FROM accion
 OR d->>'accion' IS DISTINCT FROM accion OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM tipo OR d->>'finalidad' IS DISTINCT FROM finalidad
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d->>'version_rol_ref' IS DISTINCT FROM rol_esperado
 OR v->>'superficie' IS DISTINCT FROM 'administracion_privilegiada'
 OR v->'cuenta_privilegiada' IS DISTINCT FROM 'true'::jsonb
 OR v->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR ctx->>'persona_ref' IS NULL OR ctx->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR ctx->>'cuenta_ref' IS DISTINCT FROM v->>'cuenta_ref'
 OR encode(sha256(p_contexto),'hex') IS DISTINCT FROM v->>'contexto_actor_huella_sha256'
 THEN RAISE EXCEPTION 'AD3-143: orden o control no ligados' USING ERRCODE='42501'; END IF;
 guard:=to_regprocedure('vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text)');
 proveedor:=to_regrole('vec_identidad_sesiones_v1_propietario');
 -- Esta fachada no modifica identidad ni se presta al runtime de otros portales.
 IF guard IS NULL OR proveedor IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=guard
   AND p.proowner=proveedor AND p.prosecdef AND p.prokind='f' AND p.provolatile='v'
   AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.prorettype='boolean'::regtype AND NOT p.proretset)
 OR NOT has_function_privilege(current_user,guard,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=guard AND (a.grantee NOT IN(proveedor,'vec_autorizacion_atestada_v3_propietario'::regrole)
      OR a.grantor<>proveedor OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD3-143: fuente ADMIN incompatible' USING ERRCODE='55000'; END IF;
 -- Revalidación obligatoria incluso si el núcleo devuelve un replay histórico.
 IF vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(
  v->>'autenticacion_ref',v->>'sesion_ref',v->>'cuenta_ref',v->>'cuenta_ordinaria_ref',
  ctx->>'persona_ref',d->>'perfil_activo_ref',v->>'politica_garantia_ref',v->>'politica_garantia_huella_sha256') IS NOT TRUE
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
  p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 THEN RAISE EXCEPTION 'AD3-143: autoridad ADMIN no vigente' USING ERRCODE='42501'; END IF;
 IF p_perfil='admin_copias_orden' THEN
  doble:=to_regprocedure('vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz)');
  proveedor:='vec_administracion_copias_propietario'::regrole;
  IF doble IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=doble
    AND p.proowner=proveedor AND p.prosecdef AND p.prokind='f' AND p.provolatile='v'
    AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
    AND p.prorettype='boolean'::regtype AND NOT p.proretset)
  OR NOT has_function_privilege(current_user,doble,'EXECUTE')
  OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=doble AND (a.grantee NOT IN(proveedor,'vec_autorizacion_atestada_v3_propietario'::regrole)
      OR a.grantor<>proveedor OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD3-143: doble control incompatible' USING ERRCODE='55000'; END IF;
  IF vec_administracion_copias.acreditar_doble_control_orden_v1(
   c->>'efecto_ref',c->>'huella_efecto_sha256',ctx->>'persona_ref',
   (d->>'emitida_en')::timestamptz,(d->>'valida_hasta')::timestamptz) IS NOT TRUE
  THEN RAISE EXCEPTION 'AD3-143: doble control no vigente' USING ERRCODE='42501'; END IF;
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  p_perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- Las autoridades quedan bloqueadas hasta el COMMIT del repositorio ADMIN.
 RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,consumo.huella_efecto_sha256,
  consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en,consumo.consumo_nuevo,ctx->>'persona_ref';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_copias_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_orden_copias_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean,persona_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s'
AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.consumir_copias_v3_interna(
 'admin_copias_orden',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_orden_copias_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_propuesta_restauracion_copias_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean,persona_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s'
AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.consumir_copias_v3_interna(
 'admin_copias_propuesta',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_propuesta_restauracion_copias_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_revision_restauracion_copias_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean,persona_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s'
AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.consumir_copias_v3_interna(
 'admin_copias_revision',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_revision_restauracion_copias_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_administracion_copias_propietario;
DO $acl$
DECLARE nombre text;f oid;a record;perfil text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consumir_copias_v3_interna','consumir_orden_copias_v3_atestada',
  'consumir_propuesta_restauracion_copias_v3_atestada','consumir_revision_restauracion_copias_v3_atestada'] LOOP
  perfil:=CASE WHEN nombre='consumir_copias_v3_interna' THEN 'text,' ELSE '' END;
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre||'('||perfil||'bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::regprocedure::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
  IF nombre<>'consumir_copias_v3_interna' THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_administracion_copias_propietario',f::regprocedure::text);
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f AND (x.grantor<>p.proowner OR x.privilege_type<>'EXECUTE' OR x.is_grantable
    OR (x.grantee<>p.proowner AND (nombre='consumir_copias_v3_interna'
     OR x.grantee<>'vec_administracion_copias_propietario'::regrole))))
  THEN RAISE EXCEPTION 'AD3-143: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
