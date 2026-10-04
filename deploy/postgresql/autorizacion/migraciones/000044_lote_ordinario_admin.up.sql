\set ON_ERROR_STOP on
-- AUT44: un lote ordinario, un consumo V3, un recibo y un outbox.
-- DEPENDENCIA: CA35 -> AUT45 (Rol6) -> AD190 -> AUT44.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE c oid;definicion text;huella text;
BEGIN
 -- Borrador preservado: no abrir fachada sin dos revisiones del hash final y
 -- ensayo PostgreSQL del escritor único sobre las fuentes y fechas reales.
 IF true THEN RAISE EXCEPTION 'AUT44: borrador pendiente de revision y ensayo' USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_admin_perfiles_lote_ejecutor') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT44: dependencias CA35/AUT45/AD190 ausentes o preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT oid,pg_get_constraintdef(oid,false) INTO c,definicion
 FROM pg_constraint WHERE conrelid='vec_autorizacion.outbox_acto_admin_v1'::regclass
 AND conname='outbox_acto_admin_v1_evento_check' AND contype='c' AND convalidated;
 huella:=encode(sha256(convert_to(definicion,'UTF8')),'hex');
 -- Esta guarda permanece cerrada hasta medir pg_get_constraintdef real post-AUT24.
 IF c IS NULL OR huella IS DISTINCT FROM '1c64a8bec76ef1b4b19474477d5681d23e22ad951a1b66c6140a296cb964b6e8' THEN
  RAISE EXCEPTION 'AUT44: CHECK outbox no medido o divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
LOCK TABLE vec_autorizacion.outbox_acto_admin_v1 IN ACCESS EXCLUSIVE MODE;
DO $outbox$
DECLARE anterior text;nuevo text;
BEGIN
 SELECT pg_get_constraintdef(oid,false) INTO STRICT anterior FROM pg_constraint
 WHERE conrelid='vec_autorizacion.outbox_acto_admin_v1'::regclass
 AND conname='outbox_acto_admin_v1_evento_check' AND contype='c' AND convalidated;
 nuevo:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR (evento IS NOT DISTINCT FROM ''lote_perfiles_aplicado''))';
 ALTER TABLE vec_autorizacion.outbox_acto_admin_v1 DROP CONSTRAINT outbox_acto_admin_v1_evento_check;
 EXECUTE 'ALTER TABLE vec_autorizacion.outbox_acto_admin_v1 ADD CONSTRAINT outbox_acto_admin_v1_evento_check '||nuevo;
END $outbox$;

-- Material propietario adicional, no otra corriente de auditoría. El canon
-- de solicitud se conserva en registro_acto_admin_v1 y el outbox existente.
CREATE TABLE vec_autorizacion.material_fuentes_lote_admin_v1(
 operacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.registro_acto_admin_v1(operacion_ref),
 version integer NOT NULL CHECK(version=1),
 solicitud_sha256 text NOT NULL CHECK(solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 material_fuentes jsonb NOT NULL CHECK(jsonb_typeof(material_fuentes)='object'),
 material_fuentes_sha256 text NOT NULL
  CHECK(material_fuentes_sha256=encode(sha256(convert_to(material_fuentes::text,'UTF8')),'hex')),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
ALTER TABLE vec_autorizacion.material_fuentes_lote_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.material_fuentes_lote_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.material_fuentes_lote_admin_v1
 FOR ALL TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.material_fuentes_lote_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion.material_fuentes_lote_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.material_fuentes_lote_admin_v1 FROM PUBLIC,vec_admin_perfiles_lote_ejecutor;
REVOKE ALL ON TYPE vec_autorizacion.material_fuentes_lote_admin_v1 FROM PUBLIC,vec_admin_perfiles_lote_ejecutor;

CREATE FUNCTION vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid JOIN pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND l.rolconfig IS NULL AND g.oid=to_regrole('vec_admin_perfiles_lote_ejecutor')
 AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid))
 AND NOT EXISTS(SELECT 1 FROM pg_shdepend x WHERE x.refclassid='pg_catalog.pg_authid'::regclass AND x.refobjid=l.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_shdepend x
  WHERE x.refclassid='pg_catalog.pg_authid'::regclass AND x.refobjid=g.oid
  AND NOT((x.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
    AND x.classid='pg_catalog.pg_namespace'::regclass
    AND x.objid=to_regnamespace('vec_autorizacion') AND x.deptype='a')
   OR(x.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
    AND x.classid='pg_catalog.pg_proc'::regclass AND x.deptype='a'
    AND x.objid IN(to_regprocedure('vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()'),
      to_regprocedure('vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
      to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)')))
   OR(x.dbid=0 AND x.classid='pg_catalog.pg_database'::regclass
    AND x.objid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND x.deptype='a')))
 AND (SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
  FROM pg_database db CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
  WHERE db.datname=current_database() AND a.grantee=g.oid)
 AND (SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
  FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
  WHERE n.nspname='vec_autorizacion' AND a.grantee=g.oid)
 AND (SELECT count(*) FROM pg_proc p WHERE p.oid IN(
  to_regprocedure('vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()'),
  to_regprocedure('vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
  to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'))
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
  AND EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee NOT IN(p.proowner,g.oid) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))=3
 AND NOT has_database_privilege(l.oid,current_database(),'CREATE,TEMP')
 AND NOT has_schema_privilege(l.oid,'vec_autorizacion','CREATE'))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() FROM PUBLIC;

-- Reglas de forma; la fuente de rol/ámbito y el actor se cotejan de nuevo
-- dentro de aplicar_lote_ordinario_admin_v1 después de obtener el bloqueo.
CREATE FUNCTION vec_autorizacion.validar_material_lote_ordinario_admin_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m jsonb;c jsonb;o jsonb;primero jsonb;clave text;perfiles text[]:=ARRAY[]::text[];vinculos text[]:=ARRAY[]::text[];roles_ambito text[]:=ARRAY[]::text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536 THEN
  RAISE EXCEPTION 'AUT44: material fuera de limite' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>9
 OR NOT m ?& ARRAY['Esquema','OperacionRef','ActorPersonaRef','PerfilActivoRef','AsignacionRef','OrganizacionRef','Cambios','Motivo','ReferenciaActo']
 OR jsonb_path_exists(m,'$.** ? (@ == null)')
 OR m->>'Esquema' IS DISTINCT FROM 'administracion_perfiles_lote:v3'
 OR m->>'OperacionRef' !~ '^acto_admin:[0-9a-f]{32}$'
 OR m->>'ActorPersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'PerfilActivoRef' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'OrganizacionRef' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR jsonb_typeof(m->'Cambios') IS DISTINCT FROM 'array' OR jsonb_array_length(m->'Cambios') NOT BETWEEN 1 AND 32
 OR jsonb_typeof(m->'Motivo') IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m->'Motivo'))<>4
 OR NOT(m->'Motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']
 OR m->>'ReferenciaActo' IS NULL OR length(m->>'ReferenciaActo')>256
 THEN RAISE EXCEPTION 'AUT44: canon de lote invalido' USING ERRCODE='22023'; END IF;
 primero:=m#>'{Cambios,0,Objetivo}';
 IF m->>'ActorPersonaRef' IS NOT DISTINCT FROM primero->>'PersonaRef' THEN
  RAISE EXCEPTION 'AUT44: autoasignacion denegada' USING ERRCODE='42501'; END IF;
 FOR c IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  o:=c->'Objetivo';clave:=jsonb_build_array(c->>'RolVersionRef',o->>'UnidadRef',o->>'CentroRef')::text;
  IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>4
  OR NOT c ?& ARRAY['Operacion','InicioVigencia','RolVersionRef','Objetivo']
  OR c->>'Operacion' NOT IN('otorgar','revocar')
  OR c->>'RolVersionRef' !~ '^rol:[a-z0-9_]+:v[1-9][0-9]*$'
  OR jsonb_typeof(o) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(o))<>17
  OR NOT o ?& ARRAY['UnidadRef','CentroRef','CuentaRef','CuentaVersion','PersonaRef','PersonaVersion','PerfilRef','PerfilVersion','VinculoRef','VinculoVersion','HuellaSHA256','RevisionContinuidad','ProcedenciaRef','ProcedenciaVersion','ProcedenciaHuellaSHA256','VigenteDesde','VigenteHasta']
  OR o->>'PersonaRef' IS DISTINCT FROM primero->>'PersonaRef'
  OR o->>'UnidadRef' IS NULL OR o->>'UnidadRef' IS DISTINCT FROM primero->>'UnidadRef'
  OR o->>'CentroRef' IS DISTINCT FROM ''
  OR o->>'CuentaRef' IS DISTINCT FROM primero->>'CuentaRef'
  OR o->>'PersonaVersion' IS DISTINCT FROM primero->>'PersonaVersion'
  OR o->>'CuentaVersion' IS DISTINCT FROM primero->>'CuentaVersion'
  OR o->>'ProcedenciaRef' IS DISTINCT FROM primero->>'ProcedenciaRef'
  OR o->>'ProcedenciaVersion' IS DISTINCT FROM primero->>'ProcedenciaVersion'
  OR o->>'ProcedenciaHuellaSHA256' IS DISTINCT FROM primero->>'ProcedenciaHuellaSHA256'
  OR o->>'PerfilRef' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
  OR o->>'VinculoRef' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
  OR o->>'HuellaSHA256' !~ '^[0-9a-f]{64}$'
  OR o->>'ProcedenciaHuellaSHA256' !~ '^[0-9a-f]{64}$'
  OR (c->>'Operacion'='otorgar' AND (o->>'PerfilVersion'<>'0' OR o->>'VinculoVersion'<>'0'
      OR c->>'InicioVigencia' NOT IN('inmediato','programado')
      OR (c->>'InicioVigencia'='inmediato' AND o->>'VigenteDesde'<>'0001-01-01T00:00:00Z')
      OR (c->>'InicioVigencia'='programado' AND o->>'VigenteDesde'='0001-01-01T00:00:00Z')))
  OR (c->>'Operacion'='revocar' AND ((o->>'PerfilVersion')::numeric<1 OR (o->>'VinculoVersion')::numeric<1
      OR o->>'PerfilVersion' IS DISTINCT FROM o->>'VinculoVersion'
      OR c->>'InicioVigencia'<>'' OR o->>'VigenteDesde'<>'0001-01-01T00:00:00Z'
      OR o->>'VigenteHasta'<>'0001-01-01T00:00:00Z'))
  OR o->>'PerfilRef'=ANY(perfiles) OR o->>'VinculoRef'=ANY(vinculos) OR clave=ANY(roles_ambito)
  THEN RAISE EXCEPTION 'AUT44: cambio incompatible' USING ERRCODE='22023'; END IF;
  perfiles:=array_append(perfiles,o->>'PerfilRef');vinculos:=array_append(vinculos,o->>'VinculoRef');roles_ambito:=array_append(roles_ambito,clave);
 END LOOP;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_material_lote_ordinario_admin_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(p_solicitud text,p_fuentes jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET timezone='UTC' SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb;item jsonb;descriptor jsonb;ambitos jsonb;esperados jsonb;asignacion record;
 d jsonb;org text;unidad text;hasta timestamptz;indice integer:=0;sha text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_fuentes IS NULL OR jsonb_typeof(p_fuentes) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_fuentes))<>6
 OR NOT p_fuentes ?& ARRAY['esquema','version','solicitud_sha256','organizacion_ref','unidad_ref','ambitos_por_cambio']
 OR jsonb_path_exists(p_fuentes,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT44: fuentes de lote ausentes' USING ERRCODE='42501'; END IF;
 m:=vec_autorizacion.validar_material_lote_ordinario_admin_v1(p_solicitud);
 org:=m->>'OrganizacionRef';unidad:=m#>>'{Cambios,0,Objetivo,UnidadRef}';
 sha:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 IF p_fuentes->>'esquema' IS DISTINCT FROM 'vec.admin.perfiles.lote.fuentes.v1'
 OR p_fuentes->>'version' IS DISTINCT FROM '1'
 OR p_fuentes->>'solicitud_sha256' IS DISTINCT FROM sha
 OR p_fuentes->>'organizacion_ref' IS DISTINCT FROM org
 OR p_fuentes->>'unidad_ref' IS DISTINCT FROM unidad
 OR jsonb_typeof(p_fuentes->'ambitos_por_cambio') IS DISTINCT FROM 'array'
 OR jsonb_array_length(p_fuentes->'ambitos_por_cambio')<>jsonb_array_length(m->'Cambios')
 THEN RAISE EXCEPTION 'AUT44: fuentes no ligadas a solicitud' USING ERRCODE='42501'; END IF;
 esperados:=jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),
  jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)));
 SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=m->>'PerfilActivoRef' AND a.asignacion_ref=m->>'AsignacionRef' FOR SHARE OF a,x;
 IF NOT FOUND OR asignacion.principal_id IS DISTINCT FROM m->>'ActorPersonaRef'
 OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
 OR (asignacion.documento->'ambitos' @> esperados) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT44: ambito del administrador no acreditado' USING ERRCODE='42501'; END IF;
 FOR item IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  indice:=indice+1;descriptor:=p_fuentes->'ambitos_por_cambio'->(indice-1);
  IF jsonb_typeof(descriptor) IS DISTINCT FROM 'array' OR jsonb_array_length(descriptor)<>2
   OR descriptor#>>'{0,dimension}' IS DISTINCT FROM 'organizacion_ref'
   OR descriptor#>>'{0,valores,0}' IS DISTINCT FROM org
   OR descriptor#>>'{1,dimension}' IS DISTINCT FROM 'unidad_ref'
   OR descriptor#>>'{1,valores,0}' IS DISTINCT FROM unidad
   OR item#>>'{Objetivo,UnidadRef}' IS DISTINCT FROM unidad
  THEN RAISE EXCEPTION 'AUT44: fuente de item divergente' USING ERRCODE='42501'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(descriptor) AS x(value)
   WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(x.value))<>3
    OR NOT (x.value ?& ARRAY['dimension','valores','fuente'])
    OR jsonb_typeof(x.value->'fuente') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(x.value->'fuente'))<>3
    OR NOT ((x.value->'fuente') ?& ARRAY['referencia','version','huella_sha256']))
  THEN RAISE EXCEPTION 'AUT44: descriptor abierto' USING ERRCODE='22023'; END IF;
  IF item->>'Operacion'='otorgar' THEN
   hasta:=(item#>>'{Objetivo,VigenteHasta}')::timestamptz;
  ELSE
   hasta:=clock_timestamp()+interval '1 microsecond';
  END IF;
  d:=vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(descriptor,hasta);
  IF d->'ambitos' IS DISTINCT FROM esperados OR jsonb_typeof(d->'unidades') IS DISTINCT FROM 'array'
   OR jsonb_array_length(d->'unidades')<>1
   OR d#>>'{unidades,0,organizacion_ref}' IS DISTINCT FROM org
   OR d#>>'{unidades,0,unidad_ref}' IS DISTINCT FROM unidad
  THEN RAISE EXCEPTION 'AUT44: unidad propietaria no acreditada' USING ERRCODE='42501'; END IF;
 END LOOP;
 RETURN jsonb_build_object('fuentes_sha256',encode(sha256(convert_to(p_fuentes::text,'UTF8')),'hex'),
  'ambitos',esperados);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(text,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_lote_ordinario_admin_v1(
 p_material text,p_fuentes jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET timezone='UTC' SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb;c jsonb;d jsonb;recurso jsonb;item jsonb;o jsonb;objeto jsonb;antes jsonb;despues jsonb;
 rol jsonb;consumo record;previo record;fuentes_originales record;asignacion record;validaciones jsonb[]:=ARRAY[]::jsonb[];
 ahora timestamptz;acto text;recibo text;asig_id text;asig_ref text;doc jsonb;ambitos jsonb;
 version_ca numeric;version_asig numeric;estado text;vig_desde timestamptz;vig_hasta timestamptz;
 huella text;hantes text;hdespues text;ref text;resultado jsonb;fuentes_validas jsonb;fuentes_sha text;
 recibos jsonb:='[]'::jsonb;inicios jsonb:='[]'::jsonb;
 indice integer:=0;modo text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT44: runtime o transaccion no acreditados' USING ERRCODE='42501'; END IF;
 m:=vec_autorizacion.validar_material_lote_ordinario_admin_v1(p_material);
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 recurso:=vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(p_material);
 ref:=m#>>'{Cambios,0,Objetivo,PersonaRef}';
 huella:=recurso->>'solicitud_sha256';
 IF c->>'operacion' IS DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
 OR c->>'efecto_ref' IS DISTINCT FROM ref OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'persona' OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d->>'recurso_ref' IS DISTINCT FROM ref OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso->>'contexto_sha256'
 OR d->>'principal_id' IS DISTINCT FROM m->>'ActorPersonaRef'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'PerfilActivoRef'
 OR d->>'asignacion_ref' IS DISTINCT FROM m->>'AsignacionRef'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 THEN RAISE EXCEPTION 'AUT44: decision no ligada al lote' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(
 p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM recurso->>'contexto_sha256'
 OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR consumo.auditoria_ref IS NULL OR consumo.consumida_en IS NULL
 THEN RAISE EXCEPTION 'AUT44: acuse de consumo incompatible' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x
  WHERE x->>'persona_ref'=m->>'ActorPersonaRef' AND x->>'perfil_ref'=m->>'PerfilActivoRef'
  AND x->>'asignacion_ref'=m->>'AsignacionRef')
 THEN RAISE EXCEPTION 'AUT44: administrador no efectivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO previo FROM vec_autorizacion.registro_acto_admin_v1
 WHERE operacion_ref=m->>'OperacionRef' FOR SHARE;
 IF FOUND THEN
  IF previo.material IS DISTINCT FROM convert_to(p_material,'UTF8') OR previo.huella_sha256 IS DISTINCT FROM huella
  THEN RAISE EXCEPTION 'AUT44: replay distinto' USING ERRCODE='23505'; END IF;
  SELECT * INTO STRICT fuentes_originales FROM vec_autorizacion.material_fuentes_lote_admin_v1
   WHERE operacion_ref=m->>'OperacionRef' FOR SHARE;
  fuentes_validas:=vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(
   p_material,fuentes_originales.material_fuentes);
  IF fuentes_originales.solicitud_sha256 IS DISTINCT FROM huella
   OR fuentes_originales.material_fuentes_sha256 IS DISTINCT FROM fuentes_validas->>'fuentes_sha256'
   OR previo.resultado->>'fuentes_sha256' IS DISTINCT FROM fuentes_validas->>'fuentes_sha256'
  THEN RAISE EXCEPTION 'AUT44: replay de fuente distinto' USING ERRCODE='42501'; END IF;
  -- La lectura de fuentes pudo esperar. No devuelve un recibo histórico bajo
  -- una autorización o fuente que ya venció tras esa última espera.
  IF (vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(
    p_material,fuentes_originales.material_fuentes))->>'fuentes_sha256'
      IS DISTINCT FROM fuentes_originales.material_fuentes_sha256
  THEN RAISE EXCEPTION 'AUT44: replay de fuente caducada' USING ERRCODE='42501'; END IF;
  IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
    p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
  THEN RAISE EXCEPTION 'AUT44: replay de decision caducada' USING ERRCODE='42501'; END IF;
  IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
    d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
    'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
    '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT44: replay sin autoridad actual' USING ERRCODE='42501'; END IF;
  IF clock_timestamp()>=(d->>'valida_hasta')::timestamptz
  THEN RAISE EXCEPTION 'AUT44: replay vencido' USING ERRCODE='42501'; END IF;
  RETURN previo.resultado;
 END IF;
 fuentes_validas:=vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(p_material,p_fuentes);
 fuentes_sha:=fuentes_validas->>'fuentes_sha256';

 -- Primero se bloquean y cotejan TODAS las preimágenes, sin cambiar ninguna.
 FOR item IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  o:=item->'Objetivo';
  modo:=item->>'InicioVigencia';
  rol:=vec_autorizacion.resolver_rol_administrable_v1(item->>'RolVersionRef');
  IF rol->>'clase' IS DISTINCT FROM 'ordinario'
   OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto WHERE version_rol_ref=item->>'RolVersionRef')
   OR (rol->>'unidad_requerida')::boolean IS DISTINCT FROM (o->>'UnidadRef'<>'')
   OR (item->>'Operacion'='otorgar' AND (item->>'InicioVigencia'='programado' AND (o->>'VigenteDesde')::timestamptz<(rol->>'vigente_desde')::timestamptz
       OR (o->>'VigenteHasta')::timestamptz>(rol->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT44: rol o ambito no ordinario' USING ERRCODE='42501'; END IF;
  objeto:=jsonb_build_object('cuenta_ref',o->>'CuentaRef','cuenta_version',o->'CuentaVersion',
   'persona_ref',o->>'PersonaRef','persona_version',o->'PersonaVersion',
   'perfil_ref',o->>'PerfilRef','perfil_version',o->'PerfilVersion',
   'vinculo_ref',o->>'VinculoRef','vinculo_version',o->'VinculoVersion',
   'huella_sha256','','revision_continuidad',o->'RevisionContinuidad',
   'procedencia_ref',o->>'ProcedenciaRef','procedencia_version',o->'ProcedenciaVersion',
   'procedencia_huella_sha256',o->>'ProcedenciaHuellaSHA256','vigente_hasta',o->>'VigenteHasta');
  item:=jsonb_build_object('operacion',item->>'Operacion','rol_version_ref',item->>'RolVersionRef',
   'objetivo',objeto,'unidad_ref',NULLIF(o->>'UnidadRef',''));
  antes:=vec_autorizacion.preimagen_cambio_admin_interna_v1(item);
  hantes:=encode(sha256(convert_to(antes::text,'UTF8')),'hex');
  IF hantes IS DISTINCT FROM o->>'HuellaSHA256'
   OR (antes#>>'{contexto,cuenta,version}')::numeric IS DISTINCT FROM (o->>'CuentaVersion')::numeric
   OR (antes#>>'{contexto,persona,version}')::numeric IS DISTINCT FROM (o->>'PersonaVersion')::numeric
   OR antes#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'consumido'
  THEN RAISE EXCEPTION 'AUT44: CAS de preimagen divergente' USING ERRCODE='40001'; END IF;
  IF item->>'operacion'='otorgar' THEN
   IF antes#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb
   OR antes#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb
   OR antes->'asignacion' IS DISTINCT FROM 'null'::jsonb
   OR (item->>'InicioVigencia'='programado' AND (o->>'VigenteDesde')::timestamptz<=clock_timestamp())
   OR (o->>'VigenteHasta')::timestamptz<=clock_timestamp()
   THEN RAISE EXCEPTION 'AUT44: alta sin referencias nuevas' USING ERRCODE='40001'; END IF;
  ELSE
   IF antes#>>'{contexto,perfil,estado}' IS DISTINCT FROM 'activo'
   OR antes#>>'{contexto,vinculo,estado}' IS DISTINCT FROM 'activo'
   OR antes#>>'{asignacion,estado}' IS DISTINCT FROM 'activa'
   OR antes#>>'{asignacion,version_rol_ref}' IS DISTINCT FROM item->>'rol_version_ref'
   OR ((rol->>'unidad_requerida')::boolean AND antes#>'{asignacion,ambitos}' IS DISTINCT FROM
      jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(o->>'UnidadRef')))
      )
   OR (antes#>>'{contexto,perfil,version}')::numeric IS DISTINCT FROM (o->>'PerfilVersion')::numeric
   OR (antes#>>'{contexto,vinculo,version}')::numeric IS DISTINCT FROM (o->>'VinculoVersion')::numeric
   OR o->>'VigenteDesde' IS DISTINCT FROM '0001-01-01T00:00:00Z'
   OR o->>'VigenteHasta' IS DISTINCT FROM '0001-01-01T00:00:00Z'
   THEN RAISE EXCEPTION 'AUT44: baja sin perfil exacto vivo' USING ERRCODE='40001'; END IF;
  END IF;
  validaciones:=array_append(validaciones,jsonb_build_object('item',item,'objetivo',o,'rol',rol,'antes',antes,'modo',modo));
 END LOOP;

 -- Sólo después del último CAS se elige la fecha privada compartida.
 IF (vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(p_material,p_fuentes))->>'fuentes_sha256' IS DISTINCT FROM fuentes_sha
 OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
  '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
  p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR clock_timestamp()>=(d->>'valida_hasta')::timestamptz
 THEN RAISE EXCEPTION 'AUT44: fuente o permiso vencido tras bloqueo' USING ERRCODE='42501'; END IF;
 ahora:=clock_timestamp();acto:='acto_admin:'||substr(encode(sha256(convert_to(m->>'OperacionRef','UTF8')),'hex'),1,32);
 recibo:='recibo_admin:'||substr(encode(sha256(convert_to((m->>'OperacionRef')||':recibo','UTF8')),'hex'),1,32);
 FOREACH item IN ARRAY validaciones LOOP
  indice:=indice+1;objeto:=item->'item';o:=item->'objetivo';rol:=item->'rol';antes:=item->'antes';modo:=item->>'modo';
  IF clock_timestamp()>=(d->>'valida_hasta')::timestamptz
   OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
    d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
    'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
    '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT44: permiso vencido durante lote' USING ERRCODE='42501'; END IF;
  IF objeto->>'operacion'='otorgar' THEN
   vig_desde:=CASE WHEN modo='inmediato' THEN ahora ELSE (o->>'VigenteDesde')::timestamptz END;
   IF vig_desde<ahora OR (modo='programado' AND vig_desde<=clock_timestamp())
   THEN RAISE EXCEPTION 'AUT44: inicio de lote ya vencido' USING ERRCODE='40001'; END IF;
   PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(
    o->>'CuentaRef',o->>'PersonaRef',(o->>'CuentaVersion')::numeric,(o->>'PersonaVersion')::numeric,
    o->>'PerfilRef',o->>'VinculoRef',o->>'ProcedenciaRef',(o->>'ProcedenciaVersion')::numeric,
    o->>'ProcedenciaHuellaSHA256',vig_desde,(o->>'VigenteHasta')::timestamptz,ahora);
   asig_id:='admin_'||substr(encode(sha256(convert_to(
    'vec.admin.lote.item.v1'||chr(10)||(m->>'OperacionRef')||chr(10)||indice::text||chr(10)||(o->>'PerfilRef'),
    'UTF8')),'hex'),1,32);
   version_asig:=1;version_ca:=1;estado:='activo';
   IF (rol->>'unidad_requerida')::boolean THEN
    ambitos:=jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(o->>'UnidadRef')));
   ELSE
    SELECT ambitos_fijos INTO STRICT ambitos FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=objeto->>'rol_version_ref';
    IF vec_autorizacion.ambitos_positivos_validos(jsonb_build_object('ambitos',ambitos)) IS NOT TRUE
    THEN RAISE EXCEPTION 'AUT44: ambito fijo no publicado' USING ERRCODE='42501'; END IF;
   END IF;
   vig_hasta:=(o->>'VigenteHasta')::timestamptz;
   doc:=jsonb_build_object('asignacion_id',asig_id,'version',version_asig,
    'perfil_activo_ref',o->>'PerfilRef','principal_id',o->>'PersonaRef',
    'version_rol_ref',objeto->>'rol_version_ref','estado','activa','ambitos',ambitos,
    'emitida_por',m->>'ActorPersonaRef','emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'vigente_desde',to_char(vig_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'vigente_hasta',to_char(vig_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'revocada_en','0001-01-01T00:00:00Z');
  ELSE
   PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(
    o->>'CuentaRef',o->>'PersonaRef',o->>'PerfilRef',o->>'VinculoRef',
    (o->>'CuentaVersion')::numeric,(o->>'PersonaVersion')::numeric,
    (o->>'PerfilVersion')::numeric,(o->>'VinculoVersion')::numeric,
    o->>'ProcedenciaRef',(o->>'ProcedenciaVersion')::numeric,o->>'ProcedenciaHuellaSHA256',ahora);
   SELECT a.* INTO STRICT asignacion FROM vec_autorizacion.asignacion_perfil a
    WHERE a.asignacion_ref=antes#>>'{asignacion,asignacion_ref}' FOR SHARE;
   asig_id:=asignacion.asignacion_id;version_asig:=asignacion.version+1;
   version_ca:=(o->>'PerfilVersion')::numeric+1;estado:='revocado';
   vig_desde:=(antes#>>'{contexto,perfil,vigente_desde}')::timestamptz;
   vig_hasta:=(antes#>>'{contexto,perfil,vigente_hasta}')::timestamptz;
   doc:=jsonb_set(jsonb_set(asignacion.documento,'{version}',to_jsonb(version_asig)),'{estado}','"revocada"'::jsonb)
    ||jsonb_build_object('revocada_por',m->>'ActorPersonaRef',
      'revocada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'revocacion_ref',acto);
  END IF;
  inicios:=inicios||jsonb_build_array(jsonb_build_object(
   'modo',CASE WHEN objeto->>'operacion'='otorgar' THEN modo ELSE '' END,
   'vigente_desde',CASE WHEN objeto->>'operacion'='otorgar'
    THEN to_char(vig_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    ELSE '0001-01-01T00:00:00Z' END));
  asig_ref:='asignacion:'||asig_id||':v'||version_asig::text;
  INSERT INTO vec_autorizacion.asignacion_perfil(
   asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
  VALUES(asig_ref,asig_id,version_asig,o->>'PerfilRef',o->>'PersonaRef',objeto->>'rol_version_ref',
   encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),
   (doc->>'emitida_en')::timestamptz,doc);
  INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(asig_ref,txid_current(),m->>'OperacionRef',consumo.auditoria_ref);
  IF objeto->>'operacion'='otorgar' THEN
   INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES(o->>'PerfilRef',asig_ref,ahora,m->>'ActorPersonaRef',acto);
  ELSE
   UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=asig_ref,actualizada_en=ahora,
    actualizada_por=m->>'ActorPersonaRef',acto_ref=acto
   WHERE perfil_activo_ref=o->>'PerfilRef' AND asignacion_ref=antes#>>'{asignacion,asignacion_ref}';
   IF NOT FOUND THEN RAISE EXCEPTION 'AUT44: CAS de asignacion perdido' USING ERRCODE='40001'; END IF;
  END IF;
  despues:=vec_autorizacion.preimagen_cambio_admin_interna_v1(objeto);
  hdespues:=encode(sha256(convert_to(despues::text,'UTF8')),'hex');
  recibos:=recibos||jsonb_build_array(jsonb_build_object(
   'operacion_ref',m->>'OperacionRef','acto_ref',acto,'recibo_ref',recibo,'propuesta_ref','',
   'auditoria_ref',consumo.auditoria_ref,'actor_persona_ref',m->>'ActorPersonaRef',
   'perfil_activo_ref',m->>'PerfilActivoRef','asignacion_perfil_ref',m->>'AsignacionRef',
   'correlacion_ref',d->>'correlacion_ref','objetivo_persona_ref',o->>'PersonaRef',
   'perfil_ref',o->>'PerfilRef','vinculo_ref',o->>'VinculoRef','estado_posterior',estado,
   'version_posterior',version_ca,'huella_antes_sha256',o->>'HuellaSHA256',
   'huella_despues_sha256',hdespues,'confirmado_en',ahora,'unidad_ref',o->>'UnidadRef',
   'centro_ref',o->>'CentroRef','rol_version_ref',objeto->>'rol_version_ref',
   'vigente_desde',vig_desde,'vigente_hasta',vig_hasta,'motivo',m->'Motivo',
   'referencia_acto',m->>'ReferenciaActo'));
 END LOOP;
 resultado:=jsonb_build_object('operacion_ref',m->>'OperacionRef','acto_ref',acto,
  'recibo_ref',recibo,'auditoria_ref',consumo.auditoria_ref,'huella_solicitud_sha256',huella,
  'fuentes_sha256',fuentes_sha,
  'confirmado_en',ahora,'cambios',recibos,'inicios',inicios);
 INSERT INTO vec_autorizacion.registro_acto_admin_v1 VALUES(m->>'OperacionRef',convert_to(p_material,'UTF8'),huella,
  m->>'ActorPersonaRef',m->>'PerfilActivoRef',consumo.decision_ref,consumo.auditoria_ref,resultado,ahora);
 INSERT INTO vec_autorizacion.material_fuentes_lote_admin_v1 VALUES(m->>'OperacionRef',1,huella,p_fuentes,fuentes_sha,ahora);
 INSERT INTO vec_autorizacion.outbox_acto_admin_v1 VALUES(m->>'OperacionRef','lote_perfiles_aplicado',consumo.auditoria_ref,ahora);
 -- Comprueba otra vez tras todos los INSERT y cualquier espera de índices o
 -- triggers. Si venció algo, la excepción revierte el lote entero y AD190.
 FOR item IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  rol:=vec_autorizacion.resolver_rol_administrable_v1(item->>'RolVersionRef');
  IF rol->>'clase' IS DISTINCT FROM 'ordinario'
  THEN RAISE EXCEPTION 'AUT44: rol retirado durante lote' USING ERRCODE='42501'; END IF;
 END LOOP;
 IF (vec_autorizacion.cotejar_fuentes_lote_ordinario_admin_v1(p_material,p_fuentes))->>'fuentes_sha256'
      IS DISTINCT FROM fuentes_sha
 THEN RAISE EXCEPTION 'AUT44: fuente vencida al devolver recibo' USING ERRCODE='42501'; END IF;
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
  p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 THEN RAISE EXCEPTION 'AUT44: decision vencida al devolver recibo' USING ERRCODE='42501'; END IF;
 IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
  d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
  'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
  '[]'::jsonb,d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT44: permiso vencido al devolver recibo' USING ERRCODE='42501'; END IF;
 IF clock_timestamp()>=(d->>'valida_hasta')::timestamptz
 THEN RAISE EXCEPTION 'AUT44: lote vencido al devolver recibo' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_lote_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_login_lote_ordinario_admin_v1(),
 vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.resolver_rol_administrable_v1(text) TO vec_admin_perfiles_lote_ejecutor;
COMMIT;
