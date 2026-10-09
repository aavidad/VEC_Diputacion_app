\set ON_ERROR_STOP on
-- AD228: presentación externa y revisión RRHH de inscripción; lectura positiva
-- audit-only en la corriente común. Orden causal: AD220, AD227, AD230, AD228.
-- El LOGIN privado vec_bolsa_inscripciones_lector se aprovisiona fuera de Git.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000228',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_ratificacion_catalogo_admin_v1(jsonb)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND a.attname='ratificacion_catalogo_detalle' AND a.atttypid='jsonb'::regtype
   AND NOT a.attisdropped)
 OR EXISTS(SELECT 1 FROM pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND a.attname IN('lectura_revision_permisos','lectura_instantanea_sha256')
   AND NOT a.attisdropped)
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones_empleado') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones_rrhh') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD228: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_bolsa_llamamientos_lector_inscripciones NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_lector_inscripciones_empleado NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_llamamientos_lector_inscripciones_rrhh NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_lector_inscripciones,
  vec_bolsa_llamamientos_lector_inscripciones_empleado,vec_bolsa_llamamientos_lector_inscripciones_rrhh',current_database());
END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Tres LOGIN técnicos separados: la familia de acción y la superficie fijan
-- el único rol NOLOGIN admitido; el rol no concede lectura de tablas AD3.
CREATE FUNCTION vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(p_accion text,p_canal text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM (VALUES
  ('vec_bolsa_inscripciones_lector','vec_bolsa_llamamientos_lector_inscripciones','externa_personal','propia'),
  ('vec_bolsa_inscripciones_empleado_lector','vec_bolsa_llamamientos_lector_inscripciones_empleado','interna_corporativa','propia'),
  ('vec_bolsa_inscripciones_rrhh_lector','vec_bolsa_llamamientos_lector_inscripciones_rrhh','interna_corporativa','rrhh')
 ) AS permitido(login_nombre,rol_nombre,canal,familia)
 JOIN pg_roles l ON l.rolname=permitido.login_nombre
 JOIN pg_auth_members m ON m.member=l.oid
 JOIN pg_roles g ON g.oid=m.roleid AND g.rolname=permitido.rol_nombre
 WHERE permitido.login_nombre=session_user AND permitido.canal=p_canal
 AND ((permitido.familia='propia' AND p_accion IN (
  'bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
  'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar'))
  OR (permitido.familia='rrhh' AND p_accion IN (
  'bolsa.inscripcion.rrhh.listar','bolsa.inscripcion.rrhh.consultar',
  'bolsa.inscripcion.rrhh.motivos')))
 AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(text,text) FROM PUBLIC;

-- Parche externo sobre POST-AD218 exacto: guarda de perfil y ligadura nominal.
DO $externo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 guarda text:=$m$p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')$m$;
 nueva_guarda text:=$m$p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa','presentacion_inscripcion')$m$;
 marca text:=$m$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$m$;
 rama text:=$r$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'presentacion_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.presentar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.presentar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'inscripcion_convocatoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$r$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD228: núcleo externo ausente' USING ERRCODE='55000';END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '1714236dd60e5e9534432f9df50d71817f1c480585bf2678d514a05ff971de49'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '1b65bd3dc792259e8a6ac2ca5eb800e0d9d519dbf2f1115fca1686ff757719c8'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR length(original)-length(replace(original,guarda,''))<>length(guarda)
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'presentacion_inscripcion')<>0
 THEN RAISE EXCEPTION 'AD228: preimagen núcleo externo divergente' USING ERRCODE='55000';END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
 AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(replace(original,guarda,nueva_guarda),marca,rama||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(actual,rama||marca,marca),nueva_guarda,guarda) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD228: postimagen núcleo externo divergente' USING ERRCODE='55000';END IF;
END $externo$;

-- Núcleo interno POST-AD227 exacto. El despacho externo y la rama RRHH se
-- insertan en marcas únicas; la postimagen inversa conserva todo B1 y AD220.
DO $interno$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 despacho text:=$m$p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')$m$;
 nuevo_despacho text:=$m$p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa','presentacion_inscripcion')$m$;
 exclusion text:=$m$AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo'$m$;
 nueva_exclusion text:=$m$AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo' AND p_perfil_mutacion IS DISTINCT FROM 'revision_inscripcion'$m$;
 runtime text:=$m$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
            AND vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS TRUE)
$m$;
 rama_runtime text:=$r$           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'
            AND session_user='vec_bolsa_llamamientos_desarrollo'
            AND current_setting('role')='none'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
              WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
              AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
              AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
              AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member='vec_bolsa_llamamientos_ejecutor'::regrole)
              AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,m.roleid))))
$r$;
 marca text:=$m$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
$m$;
 rama text:=$r$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.revisar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.rrhh.decidir'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_inscripcion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$r$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD228: núcleo interno ausente' USING ERRCODE='55000';END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'e33b3f06110ac5fb8711c4ac33ef80f140d1c9ac60e576a2e82f3c98170861e6'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '6ef6b37f7edca6306744d32895918f6b65f7f133ad6edd573eee1345a3d3399c'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR length(original)-length(replace(original,despacho,''))<>length(despacho)
 OR length(original)-length(replace(original,exclusion,''))<>length(exclusion)
 OR length(original)-length(replace(original,runtime,''))<>length(runtime)
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'revision_inscripcion')<>0 OR strpos(original,'presentacion_inscripcion')<>0
 THEN RAISE EXCEPTION 'AD228: preimagen núcleo interno divergente' USING ERRCODE='55000';END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(original,despacho,nuevo_despacho);
 nuevo:=replace(nuevo,exclusion,nueva_exclusion);
 nuevo:=replace(nuevo,runtime,rama_runtime||runtime);
 nuevo:=replace(nuevo,marca,rama||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR replace(replace(replace(replace(actual,rama||marca,marca),rama_runtime||runtime,runtime),nueva_exclusion,exclusion),nuevo_despacho,despacho) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD228: postimagen núcleo interno divergente' USING ERRCODE='55000';END IF;
END $interno$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM '3043f4742dc5e94eacf7038075fb5b7d4f5409828cc95c299ae4a5572558dea9'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_bolsa_llamamientos.inscripcion.presentar.v1')<>0
 OR strpos(anterior,'vec_bolsa_llamamientos.inscripcion.revisar.v1')<>0
 THEN RAISE EXCEPTION 'AD228: CHECK audiencias POSTAD227 incompatible' USING ERRCODE='55000';END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ANY (ARRAY[''vec_bolsa_llamamientos.inscripcion.presentar.v1'',''vec_bolsa_llamamientos.inscripcion.revisar.v1'']))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF session_user<>'vec_externo_bolsa_desarrollo' OR current_setting('role')<>'none'
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR p_capacidad IS NULL OR p_decision IS NULL
 THEN RAISE EXCEPTION 'AD228: presentación denegada' USING ERRCODE='42501';END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD228: material inválido' USING ERRCODE='22023';END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.presentar.v1'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'inscripcion_convocatoria'
 OR d->>'finalidad' IS DISTINCT FROM 'presentar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'recurso_ref' !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD228: presentación sin ligadura' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'presentacion_inscripcion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 THEN RAISE EXCEPTION 'AD228: presentación divergente' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;x record;
BEGIN
 IF session_user<>'vec_bolsa_llamamientos_desarrollo' OR current_setting('role')<>'none'
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR p_capacidad IS NULL OR p_decision IS NULL
 THEN RAISE EXCEPTION 'AD228: revisión denegada' USING ERRCODE='42501';END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD228: material inválido' USING ERRCODE='22023';END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.revisar.v1'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.decidir'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitud_inscripcion'
 OR d->>'finalidad' IS DISTINCT FROM 'revisar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'recurso_ref' !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD228: revisión sin ligadura' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'revision_inscripcion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 THEN RAISE EXCEPTION 'AD228: revisión divergente' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;

-- El asiento positivo no consume V3. Reutiliza la cadena interna y sus
-- columnas minimizadas; la rama nueva mantiene disjuntas las familias previas.
CREATE FUNCTION vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
 p_accion text,p_finalidad text,p_canal text,p_recurso text)
RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT octet_length(p_recurso) BETWEEN 1 AND 200 AND (
 (p_accion='bolsa.inscripcion.convocatorias.listar'
  AND p_finalidad='consulta_convocatoria_abierta'
  AND p_canal IN ('externa_personal','interna_corporativa')
  AND p_recurso ~ '^inscripciones_abiertas_[0-9a-f]{64}$')
 OR (p_accion='bolsa.inscripcion.convocatoria.consultar'
  AND p_finalidad='consulta_convocatoria_abierta'
  AND p_canal IN ('externa_personal','interna_corporativa')
  AND p_recurso ~ '^cv1_[A-Za-z0-9_-]+_v[1-9][0-9]{0,15}$')
 OR (p_accion='bolsa.inscripcion.propias.listar'
  AND p_finalidad='consulta_inscripcion_propia'
  AND p_canal IN ('externa_personal','interna_corporativa')
  AND p_recurso ~ '^inscripciones_propias_[0-9a-f]{64}$')
 OR (p_accion='bolsa.inscripcion.propia.consultar'
  AND p_finalidad='consulta_inscripcion_propia'
  AND p_canal IN ('externa_personal','interna_corporativa')
  AND p_recurso ~ '^solicitud_inscripcion_[0-9a-f]{64}$')
 OR (p_accion='bolsa.inscripcion.rrhh.listar'
  AND p_finalidad='consulta_inscripcion_rrhh' AND p_canal='interna_corporativa'
  AND p_recurso ~ '^inscripciones_rrhh_[0-9a-f]{64}$')
 OR (p_accion='bolsa.inscripcion.rrhh.consultar'
  AND p_finalidad='consulta_inscripcion_rrhh' AND p_canal='interna_corporativa'
  AND p_recurso ~ '^solicitud_inscripcion_[0-9a-f]{64}$')
 OR (p_accion='bolsa.inscripcion.rrhh.motivos'
  AND p_finalidad='consulta_motivos_inscripcion_rrhh' AND p_canal='interna_corporativa'
  AND p_recurso ~ '^motivos_inscripcion_[0-9a-f]{64}$'))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(text,text,text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN lectura_revision_permisos numeric(20,0),
 ADD COLUMN lectura_instantanea_sha256 text;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM '734f6fe84ba26fdc8ed02a54ffe7432c503a86d3437c97e32c7d5de4ea79a655'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD228: familia auditora POSTAD230 incompatible' USING ERRCODE='55000';END IF;
 nueva:='CHECK ((lectura_revision_permisos IS NULL AND lectura_instantanea_sha256 IS NULL AND ('||
  substr(anterior,8,length(anterior)-8)||')) OR ('||$rama$tipo_registro='lectura_inscripcion'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NOT NULL AND intento_material_sha256 IS NOT NULL
 AND actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL
 AND registro_contexto_ref IS NOT NULL AND contexto_sha256 IS NOT NULL
 AND procedencia_sha256 IS NOT NULL AND autenticacion_ref IS NOT NULL
 AND sesion_ref IS NOT NULL AND autenticacion_sha256 IS NOT NULL
 AND accion IS NOT NULL AND modulo_id IS NOT NULL AND recurso_ref IS NOT NULL
 AND finalidad_ref IS NOT NULL AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND proceso IS NOT NULL AND canal IS NOT NULL AND correlacion_ref IS NOT NULL
 AND vinculo_sha256 IS NOT NULL AND transaccion_origen IS NOT NULL
 AND lectura_revision_permisos IS NOT NULL AND lectura_instantanea_sha256 IS NOT NULL
 AND evento_ref IS NULL AND evento_material_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND operador_login IS NULL
 AND plan_sha256 IS NULL AND aprobacion_ref IS NULL AND version_consumo IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL
 AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL
 AND fuentes_solicitud_sha256 IS NULL AND unidad_plan_ref IS NULL
 AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL
 AND unidad_alcance IS NULL AND unidad_recibo_ref IS NULL
 AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL
 AND bootstrap_solicitud_sha256 IS NULL AND mantenimiento_detalle IS NULL
 AND mantenimiento_solicitud_sha256 IS NULL AND periodica_detalle IS NULL
 AND preservacion_detalle IS NULL AND gobierno_usuarios_detalle IS NULL
 AND gobierno_usuarios_solicitud_sha256 IS NULL AND perfiles_asignables_detalle IS NULL
 AND perfiles_asignables_solicitud_sha256 IS NULL AND identidad_operacion_ref IS NULL
 AND identidad_plan_ref IS NULL AND identidad_preimagen_sha256 IS NULL
 AND identidad_configuracion_sha256 IS NULL AND identidad_alcance IS NULL
 AND identidad_solicitud_sha256 IS NULL
 AND ratificacion_catalogo_detalle IS NULL$rama$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_inscripcion_formato_v1 CHECK (
 tipo_registro<>'lectura_inscripcion' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='bolsa' AND proceso='vec-server'
 AND vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
      accion,finalidad_ref,canal,recurso_ref)
 AND ((resultado='obtenida' AND motivo_ref='inscripcion_lectura_correcta')
   OR (resultado='no_encontrada' AND motivo_ref='inscripcion_no_encontrada'))
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 AND octet_length(actor_ref) BETWEEN 1 AND 128
 AND octet_length(perfil_activo_ref) BETWEEN 1 AND 128
 AND octet_length(registro_contexto_ref) BETWEEN 1 AND 128
 AND octet_length(autenticacion_ref) BETWEEN 1 AND 128
 AND octet_length(sesion_ref) BETWEEN 1 AND 128));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;
 v_claves constant text[]:=ARRAY['intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id','recurso_ref',
  'finalidad_ref','resultado','motivo_ref','proceso','canal','correlacion_ref',
  'lectura_revision_permisos','lectura_instantanea_sha256'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
     p_orden->>'accion',p_orden->>'canal') IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>18
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD228: lectura sin emisor acreditado' USING ERRCODE='42501';END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD228: orden de lectura inválida' USING ERRCODE='22023';END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'modulo_id'<>'bolsa' OR p_orden->>'proceso'<>'vec-server'
 OR vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
      p_orden->>'accion',p_orden->>'finalidad_ref',p_orden->>'canal',p_orden->>'recurso_ref') IS NOT TRUE
 OR NOT ((p_orden->>'resultado'='obtenida' AND p_orden->>'motivo_ref'='inscripcion_lectura_correcta')
   OR (p_orden->>'resultado'='no_encontrada' AND p_orden->>'motivo_ref'='inscripcion_no_encontrada'))
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD228: contrato de lectura inválido' USING ERRCODE='22023';END IF;
 IF (p_orden->>'lectura_revision_permisos')::numeric > 18446744073709551615::numeric
 THEN RAISE EXCEPTION 'AD228: revisión de permisos fuera de rango' USING ERRCODE='22023';END IF;
 BEGIN v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
       v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD228: material de lectura inválido' USING ERRCODE='22023';END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object' OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_esquema' IS DISTINCT FROM v_contexto->>'esquema'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM p_orden->>'canal'
 OR (((v_contexto->>'resuelto_en')::timestamptz >=
      (v_vinculo->>'sesion_revalidada_en')::timestamptz)
     AND ((v_contexto->>'resuelto_en')::timestamptz <
      (v_vinculo->>'sesion_valida_hasta')::timestamptz)) IS NOT TRUE
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
     p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
     p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD228: evidencia histórica de lectura inválida' USING ERRCODE='42501';END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_inscripcion.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN
  RAISE EXCEPTION 'AD228: lectura ya asentada' USING ERRCODE='23505';
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_li_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_inscripcion.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_inscripcion',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref',
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',p_orden->>'finalidad_ref',
  p_orden->>'resultado',p_orden->>'motivo_ref',p_orden->>'proceso',p_orden->>'canal',
  p_orden->>'correlacion_ref',encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb) TO vec_bolsa_llamamientos_propietario;
COMMIT;
