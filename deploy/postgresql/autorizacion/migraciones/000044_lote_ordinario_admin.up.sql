\set ON_ERROR_STOP on
-- AUT44: efecto del lote ordinario de perfiles de Administración (contrato v3).
-- Una sola transacción SERIALIZABLE por orden: consume la decisión nominal por
-- AD190 (auditoría común incluida), coteja actor, organización, unidad, fuentes
-- y la preimagen exacta de cada cambio, registra la procedencia del acto,
-- crea o revoca vínculos (CA35) y asignaciones con su historia, y escribe
-- registro, outbox y recibo. El replay de la misma operación devuelve el recibo
-- original tras consumir otra decisión; otro material con la misma referencia
-- se rechaza. Sólo perfiles de clase ordinario (AUT49); nadie se asigna a sí
-- mismo. Lo ejecuta únicamente un LOGIN exclusivo de vec_admin_perfiles_lote_ejecutor.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT44: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR to_regrole('vec_admin_perfiles_lote_ejecutor') IS NULL
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz,timestamptz,timestamptz)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text,timestamptz)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.registrar_procedencia_acto_admin_lote_v1(text,text)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.preimagen_admin_interna_v1(text,text,text,text)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.metadatos_persona_administrable_v1(text)','EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(jsonb,timestamptz)') IS NULL
 OR has_function_privilege('vec_autorizacion_propietario','vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(text)','EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NULL
 OR to_regprocedure('vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb)') IS NULL
 OR to_regclass('vec_autorizacion.sello_efecto_admin_tx_v1') IS NULL
 OR to_regclass('vec_autorizacion.registro_perfiles_asignables_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion.registro_lote_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT44: PARO clave=dependencias actual=divergente esperado=AD190_CA35_AUT49_sin_AUT44' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Historia del efecto: material exacto, decisión y auditoría del primer consumo,
-- fuentes cotejadas y recibo. El outbox anuncia el lote aplicado.
CREATE TABLE vec_autorizacion.registro_lote_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 65536),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(material),'hex')),
 actor_persona_ref text NOT NULL,
 actor_perfil_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 fuentes jsonb NOT NULL CHECK(jsonb_typeof(fuentes)='object'),
 recibo jsonb NOT NULL CHECK(jsonb_typeof(recibo)='object'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_autorizacion.outbox_lote_admin_v1(
 operacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.registro_lote_admin_v1(operacion_ref),
 evento text NOT NULL CHECK(evento='perfiles_lote_aplicado'),
 auditoria_ref text NOT NULL,
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['registro_lote_admin_v1','outbox_lote_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;

-- La composición de vec-admin comprueba su LOGIN antes de aceptar órdenes.
CREATE FUNCTION vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.oid=to_regrole('vec_admin_perfiles_lote_ejecutor') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.instante_lote_admin_v1(t timestamptz)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.instante_lote_admin_v1(timestamptz) FROM PUBLIC;

-- Valida la forma exacta del canon Go de SolicitudLoteAdministracionPerfiles v3.
CREATE FUNCTION vec_autorizacion.validar_material_lote_admin_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb;c jsonb;o jsonb;i int:=0;primera jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT44: material invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (SELECT count(*) FROM jsonb_object_keys(m))<>9
 OR NOT m ?& ARRAY['Esquema','OperacionRef','ActorPersonaRef','PerfilActivoRef','AsignacionRef','OrganizacionRef','Cambios','Motivo','ReferenciaActo']
 OR m->>'Esquema' IS DISTINCT FROM 'administracion_perfiles_lote:v3'
 OR EXISTS(SELECT 1 FROM unnest(ARRAY['OperacionRef','ActorPersonaRef','PerfilActivoRef','AsignacionRef','OrganizacionRef']) k WHERE jsonb_typeof(m->k)<>'string')
 OR m->>'OperacionRef' !~ '^acto_admin:[0-9a-f]{32}$'
 OR m->>'ActorPersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'PerfilActivoRef' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'AsignacionRef' !~ '^asignacion:[A-Za-z0-9_:-]{1,200}:v[1-9][0-9]{0,9}$'
 OR m->>'OrganizacionRef' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR jsonb_typeof(m->'Motivo') IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m->'Motivo'))<>4
 OR NOT (m->'Motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']
 OR jsonb_typeof(m->'ReferenciaActo') IS DISTINCT FROM 'string' OR octet_length(m->>'ReferenciaActo')>512
 OR jsonb_typeof(m->'Cambios') IS DISTINCT FROM 'array' OR jsonb_array_length(m->'Cambios') NOT BETWEEN 1 AND 32
 THEN RAISE EXCEPTION 'AUT44: material de lote invalido' USING ERRCODE='22023'; END IF;
 primera:=m#>'{Cambios,0,Objetivo}';
 FOR c IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  o:=c->'Objetivo';
  IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>4
  OR NOT c ?& ARRAY['Operacion','InicioVigencia','RolVersionRef','Objetivo']
  OR c->>'Operacion' NOT IN('otorgar','revocar')
  OR (c->>'Operacion'='otorgar' AND c->>'InicioVigencia' NOT IN('inmediato','programado'))
  OR (c->>'Operacion'='revocar' AND c->>'InicioVigencia'<>'')
  OR c->>'RolVersionRef' !~ '^rol:[a-z0-9][a-z0-9_:.-]{0,190}:v[1-9][0-9]{0,8}$'
  OR jsonb_typeof(o) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(o))<>17
  OR NOT o ?& ARRAY['UnidadRef','CentroRef','CuentaRef','CuentaVersion','PersonaRef','PersonaVersion','PerfilRef','PerfilVersion','VinculoRef','VinculoVersion','HuellaSHA256','RevisionContinuidad','ProcedenciaRef','ProcedenciaVersion','ProcedenciaHuellaSHA256','VigenteDesde','VigenteHasta']
  OR o->>'CentroRef'<>'' OR o->>'UnidadRef' IS DISTINCT FROM primera->>'UnidadRef' OR o->>'UnidadRef' !~ '^[A-Za-z0-9][A-Za-z0-9_:.-]{0,255}$'
  OR o->>'CuentaRef' !~ '^cta_[A-Za-z0-9_-]{22,128}$' OR o->>'PersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR o->>'PerfilRef' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR o->>'VinculoRef' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
  OR o->>'HuellaSHA256' !~ '^[0-9a-f]{64}$' OR o->>'ProcedenciaHuellaSHA256' !~ '^[0-9a-f]{64}$'
  OR o->>'ProcedenciaRef' !~ '^prc_[A-Za-z0-9_-]{22,128}$'
  OR jsonb_typeof(o->'CuentaVersion')<>'number' OR jsonb_typeof(o->'PersonaVersion')<>'number' OR jsonb_typeof(o->'PerfilVersion')<>'number'
  OR jsonb_typeof(o->'VinculoVersion')<>'number' OR jsonb_typeof(o->'ProcedenciaVersion')<>'number' OR (o->>'RevisionContinuidad')::numeric<>0
  OR (o->>'CuentaVersion')::numeric<1 OR (o->>'PersonaVersion')::numeric<1 OR (o->>'ProcedenciaVersion')::numeric<1
  OR o->>'PersonaRef' IS DISTINCT FROM primera->>'PersonaRef' OR o->>'CuentaRef' IS DISTINCT FROM primera->>'CuentaRef'
  OR o->'PersonaVersion' IS DISTINCT FROM primera->'PersonaVersion' OR o->'CuentaVersion' IS DISTINCT FROM primera->'CuentaVersion'
  OR o->>'ProcedenciaRef' IS DISTINCT FROM primera->>'ProcedenciaRef' OR o->'ProcedenciaVersion' IS DISTINCT FROM primera->'ProcedenciaVersion'
  OR o->>'ProcedenciaHuellaSHA256' IS DISTINCT FROM primera->>'ProcedenciaHuellaSHA256'
  OR o->>'PersonaRef' IS NOT DISTINCT FROM m->>'ActorPersonaRef'
  OR (c->>'Operacion'='otorgar' AND ((o->>'PerfilVersion')::numeric<>0 OR (o->>'VinculoVersion')::numeric<>0
   OR o->>'VigenteHasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+Z$' OR o->>'VigenteHasta' LIKE '0001-01-01%'
   OR (c->>'InicioVigencia'='inmediato' AND o->>'VigenteDesde'<>'0001-01-01T00:00:00Z')
   OR (c->>'InicioVigencia'='programado' AND (o->>'VigenteDesde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+Z$' OR o->>'VigenteDesde' LIKE '0001-01-01%' OR (o->>'VigenteDesde')::timestamptz>=(o->>'VigenteHasta')::timestamptz))))
  OR (c->>'Operacion'='revocar' AND ((o->>'PerfilVersion')::numeric<1 OR (o->>'VinculoVersion')::numeric<1 OR o->'PerfilVersion' IS DISTINCT FROM o->'VinculoVersion'
   OR o->>'VigenteDesde'<>'0001-01-01T00:00:00Z' OR o->>'VigenteHasta'<>'0001-01-01T00:00:00Z'))
  THEN RAISE EXCEPTION 'AUT44: cambio de lote invalido' USING ERRCODE='22023'; END IF;
  i:=i+1;
 END LOOP;
 IF (SELECT count(DISTINCT x->'Objetivo'->>'PerfilRef') FROM jsonb_array_elements(m->'Cambios') x)<>i
 OR (SELECT count(DISTINCT x->'Objetivo'->>'VinculoRef') FROM jsonb_array_elements(m->'Cambios') x)<>i
 OR (SELECT count(DISTINCT x->>'RolVersionRef') FROM jsonb_array_elements(m->'Cambios') x)<>i
 THEN RAISE EXCEPTION 'AUT44: cambios repetidos' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_material_lote_admin_v1(text) FROM PUBLIC;

-- Mismo canon que domain.RecursoAutorizable.HuellaContextoAutorizacionSHA256
-- (mapas ordenados por clave): ámbitos de organización y unidad y la huella
-- de la solicitud como único atributo.
CREATE FUNCTION vec_autorizacion.recurso_lote_admin_v1(p_material text,m jsonb)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 WITH x AS (SELECT encode(sha256(convert_to(p_material,'UTF8')),'hex') AS sol,
  '{"ambitos":{"organizacion_ref":'||to_jsonb(m->>'OrganizacionRef')::text||',"unidad_ref":'||to_jsonb(m#>>'{Cambios,0,Objetivo,UnidadRef}')::text||'},"atributos":{"solicitud_sha256":'||to_jsonb(encode(sha256(convert_to(p_material,'UTF8')),'hex'))::text||'}}' AS canon)
 SELECT jsonb_build_object('recurso_ref',m#>>'{Cambios,0,Objetivo,PersonaRef}','solicitud_sha256',sol,'contexto_canonico',canon,'contexto_sha256',encode(sha256(convert_to(canon,'UTF8')),'hex')) FROM x
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recurso_lote_admin_v1(text,jsonb) FROM PUBLIC;

-- Preimagen de un cambio: cuenta, persona, perfil y vínculo (con bloqueo), la
-- asignación actual del perfil, la versión y control del rol y todas las
-- asignaciones actuales de la persona. Su huella es la que compara la orden.
CREATE FUNCTION vec_autorizacion.preimagen_cambio_lote_admin_v1(p_cambio jsonb,p_org text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE o jsonb:=p_cambio->'Objetivo';ca jsonb;r record;cv record;a record;actual jsonb:=null;persona jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT44: preimagen requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 ca:=vec_contexto_actor_v1.preimagen_admin_interna_v1(o->>'CuentaRef',o->>'PersonaRef',o->>'PerfilRef',o->>'VinculoRef');
 SELECT v.* INTO STRICT r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=p_cambio->>'RolVersionRef' FOR SHARE;
 SELECT x.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref) WHERE q.perfil_activo_ref=o->>'PerfilRef' FOR UPDATE OF q;
 IF FOUND THEN actual:=jsonb_build_object('asignacion_ref',a.asignacion_ref,'version',a.version,'huella_sha256',a.huella_sha256,'estado',a.documento->>'estado','version_rol_ref',a.version_rol_ref,'principal_id',a.principal_id,'ambitos',a.documento->'ambitos'); END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('asignacion_ref',x.asignacion_ref,'huella_sha256',x.huella_sha256) ORDER BY x.asignacion_ref COLLATE "C"),'[]'::jsonb) INTO persona
 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref) WHERE x.principal_id=o->>'PersonaRef';
 RETURN jsonb_build_object('esquema','vec.admin.perfiles.lote.preimagen.v1','operacion',p_cambio->>'Operacion',
  'rol_version_ref',r.version_rol_ref,'rol_huella_sha256',r.huella_sha256,'control_revision',cv.revision,'control_huella_sha256',cv.huella_sha256,
  'organizacion_ref',p_org,'unidad_ref',o->>'UnidadRef','contexto',ca,'asignacion',actual,'asignaciones_persona',persona);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_cambio_lote_admin_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_lote_ordinario_admin_v1(p_material text,p_fuentes jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;rec jsonb;adm record;x record;previo record;actor record;org text;unidad text;
 ahora timestamptz;acto text;recibo_ref text;procedencia text;sol text;fuentes_sha text;
 cambio jsonb;o jsonb;pre jsonb;pres jsonb:='[]'::jsonb;post jsonb;rol jsonb;desc_amb jsonb;amb jsonb;i int:=0;desde timestamptz;hasta timestamptz;
 asig_id text;asig_ref text;doc jsonb;ant record;ver bigint;cambios jsonb:='[]'::jsonb;inicios jsonb:='[]'::jsonb;recibo jsonb;persona text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() IS NOT TRUE
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR (p_fuentes IS NOT NULL AND octet_length(p_fuentes::text)>65536)
 THEN RAISE EXCEPTION 'AUT44: lote denegado' USING ERRCODE='42501'; END IF;
 m:=vec_autorizacion.validar_material_lote_admin_v1(p_material);
 d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 rec:=vec_autorizacion.recurso_lote_admin_v1(p_material,m);sol:=rec->>'solicitud_sha256';
 -- La decisión debe ser para ESTE material: recurso, contexto y actor exactos.
 IF d->>'recurso_ref' IS DISTINCT FROM rec->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM rec->>'contexto_sha256'
 OR c->>'efecto_ref' IS DISTINCT FROM rec->>'recurso_ref' OR c->>'huella_efecto_sha256' IS DISTINCT FROM rec->>'contexto_sha256'
 OR d->>'principal_id' IS DISTINCT FROM m->>'ActorPersonaRef' OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'PerfilActivoRef'
 OR d->>'asignacion_ref' IS DISTINCT FROM m->>'AsignacionRef'
 OR d->>'correlacion_ref' IS NULL OR d->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$'
 THEN RAISE EXCEPTION 'AUT44: decision ajena al lote' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.efecto_ref IS DISTINCT FROM rec->>'recurso_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM rec->>'contexto_sha256'
 THEN RAISE EXCEPTION 'AUT44: consumo ajeno al lote' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.registro_lote_admin_v1 WHERE operacion_ref=m->>'OperacionRef';
 IF FOUND THEN
  -- Replay: el acceso nuevo ya quedó consumido y auditado; mismo material.
  IF previo.material IS DISTINCT FROM convert_to(p_material,'UTF8') THEN RAISE EXCEPTION 'AUT44: idempotencia divergente' USING ERRCODE='23505'; END IF;
  RETURN previo.recibo;
 END IF;
 org:=m->>'OrganizacionRef';unidad:=m#>>'{Cambios,0,Objetivo,UnidadRef}';persona:=m#>>'{Cambios,0,Objetivo,PersonaRef}';
 -- Competencia del actor: su asignación vigente cubre organización y unidad.
 SELECT a.* INTO actor FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
 WHERE q.perfil_activo_ref=m->>'PerfilActivoRef' AND a.asignacion_ref=m->>'AsignacionRef' AND a.principal_id=m->>'ActorPersonaRef' FOR SHARE OF q;
 IF NOT FOUND OR actor.documento->>'estado'<>'activa'
 OR NOT actor.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)))
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(actor.documento->'ambitos') e WHERE e->>'clave'='unidad_ref' AND e->'valores' ? unidad)
 THEN RAISE EXCEPTION 'AUT44: ambito del actor insuficiente' USING ERRCODE='42501'; END IF;
 -- Persona destinataria dentro del conjunto administrable de esa unidad.
 IF vec_contexto_actor_v1.metadatos_persona_administrable_v1(persona) IS NULL
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  WHERE a.principal_id=persona AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad))))
 THEN RAISE EXCEPTION 'AUT44: persona fuera del conjunto' USING ERRCODE='42501'; END IF;
 -- Fuentes privadas de organización y unidad, una por cambio.
 IF p_fuentes IS NULL OR jsonb_typeof(p_fuentes)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p_fuentes))<>6
 OR p_fuentes->>'esquema' IS DISTINCT FROM 'vec.admin.perfiles.lote.fuentes.v1' OR p_fuentes->'version' IS DISTINCT FROM '1'::jsonb
 OR p_fuentes->>'solicitud_sha256' IS DISTINCT FROM sol OR p_fuentes->>'organizacion_ref' IS DISTINCT FROM org OR p_fuentes->>'unidad_ref' IS DISTINCT FROM unidad
 OR jsonb_typeof(p_fuentes->'ambitos_por_cambio')<>'array' OR jsonb_array_length(p_fuentes->'ambitos_por_cambio')<>jsonb_array_length(m->'Cambios')
 THEN RAISE EXCEPTION 'AUT44: fuentes no disponibles' USING ERRCODE='42501'; END IF;
 fuentes_sha:=encode(sha256(convert_to(p_fuentes::text,'UTF8')),'hex');
 ahora:=clock_timestamp();
 acto:='acto_admin:'||substr(encode(sha256(convert_to(m->>'OperacionRef','UTF8')),'hex'),1,32);
 recibo_ref:='recibo_admin:'||substr(encode(sha256(convert_to((m->>'OperacionRef')||':recibo','UTF8')),'hex'),1,32);
 procedencia:='prc_'||substr(encode(sha256(convert_to((m->>'OperacionRef')||':procedencia','UTF8')),'hex'),1,32);
 -- Primero se comparan TODAS las preimágenes con el estado anterior al lote;
 -- después se aplican los cambios. Así el CAS es sobre lo que vio quien firmó.
 FOR cambio IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  o:=cambio->'Objetivo';
  -- Altas: perfil asignable vigente y habilitado, configurado para el lote
  -- (organización fija = la del lote; la unidad la añade cada asignación).
  -- Bajas: basta que la versión esté registrada como ordinaria; una retirada
  -- no exige que el perfil siga ofreciéndose.
  SELECT * INTO adm FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=cambio->>'RolVersionRef';
  IF NOT FOUND OR adm.clase<>'ordinario' THEN RAISE EXCEPTION 'AUT44: rol no ordinario' USING ERRCODE='42501'; END IF;
  IF cambio->>'Operacion'='otorgar' THEN
   rol:=vec_autorizacion.resolver_rol_administrable_v1(cambio->>'RolVersionRef');
   IF rol->>'clase' IS DISTINCT FROM 'ordinario' OR adm.audiencia_administrativa<>'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
   OR adm.ambitos_fijos IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)))
   THEN RAISE EXCEPTION 'AUT44: perfil no configurado para el lote' USING ERRCODE='42501'; END IF;
  END IF;
  pre:=vec_autorizacion.preimagen_cambio_lote_admin_v1(cambio,org);
  IF encode(sha256(convert_to(pre::text,'UTF8')),'hex') IS DISTINCT FROM o->>'HuellaSHA256'
  OR (pre#>>'{contexto,cuenta,version}')::numeric IS DISTINCT FROM (o->>'CuentaVersion')::numeric
  OR (pre#>>'{contexto,persona,version}')::numeric IS DISTINCT FROM (o->>'PersonaVersion')::numeric
  OR pre#>>'{contexto,persona,procedencia_ref}' IS DISTINCT FROM o->>'ProcedenciaRef'
  OR (pre#>>'{contexto,persona,procedencia_version}')::numeric IS DISTINCT FROM (o->>'ProcedenciaVersion')::numeric
  OR pre#>>'{contexto,persona,procedencia_huella_sha256}' IS DISTINCT FROM o->>'ProcedenciaHuellaSHA256'
  THEN RAISE EXCEPTION 'AUT44: preimagen divergente' USING ERRCODE='40001'; END IF;
  pres:=pres||jsonb_build_array(pre);
 END LOOP;
 PERFORM vec_contexto_actor_v1.registrar_procedencia_acto_admin_lote_v1(procedencia,sol);
 FOR cambio IN SELECT value FROM jsonb_array_elements(m->'Cambios') LOOP
  o:=cambio->'Objetivo';
  pre:=pres->i;
  desc_amb:=p_fuentes->'ambitos_por_cambio'->i;
  IF cambio->>'Operacion'='otorgar' THEN
   rol:=vec_autorizacion.resolver_rol_administrable_v1(cambio->>'RolVersionRef');
   hasta:=(o->>'VigenteHasta')::timestamptz;
   desde:=CASE WHEN cambio->>'InicioVigencia'='inmediato' THEN ahora ELSE (o->>'VigenteDesde')::timestamptz END;
   IF desde<ahora OR hasta<=desde OR hasta<=ahora OR desde<(rol->>'vigente_desde')::timestamptz OR hasta>(rol->>'vigente_hasta')::timestamptz
   OR (cambio->>'InicioVigencia'='programado' AND desde<=ahora)
   THEN RAISE EXCEPTION 'AUT44: vigencia fuera del perfil' USING ERRCODE='22023'; END IF;
   IF pre#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb OR pre#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb OR pre->'asignacion' IS DISTINCT FROM 'null'::jsonb
   THEN RAISE EXCEPTION 'AUT44: alta requiere referencias nuevas' USING ERRCODE='40001'; END IF;
   IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
    WHERE a.principal_id=persona AND a.version_rol_ref=cambio->>'RolVersionRef' AND a.documento->>'estado'='activa'
    AND ahora<(a.documento->>'vigente_hasta')::timestamptz
    AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad))))
   THEN RAISE EXCEPTION 'AUT44: perfil ya asignado' USING ERRCODE='23505'; END IF;
   amb:=vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(desc_amb,hasta)->'ambitos';
   IF amb IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
   THEN RAISE EXCEPTION 'AUT44: fuentes de ambito divergentes' USING ERRCODE='42501'; END IF;
   IF vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(o->>'CuentaRef') IS DISTINCT FROM false
   THEN RAISE EXCEPTION 'AUT44: cuenta no ordinaria' USING ERRCODE='42501'; END IF;
   PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_lote_v1(o->>'CuentaRef',persona,(o->>'CuentaVersion')::numeric,(o->>'PersonaVersion')::numeric,
    o->>'PerfilRef',o->>'VinculoRef',procedencia,1,sol,desde,hasta,ahora);
   asig_id:='admin_'||substr(encode(sha256(convert_to((m->>'OperacionRef')||':'||i,'UTF8')),'hex'),1,32);ver:=1;
   doc:=jsonb_build_object('asignacion_id',asig_id,'version',ver,'perfil_activo_ref',o->>'PerfilRef','principal_id',persona,'version_rol_ref',cambio->>'RolVersionRef',
    'estado','activa','ambitos',amb,'emitida_por',m->>'ActorPersonaRef','emitida_en',vec_autorizacion.instante_lote_admin_v1(ahora),
    'vigente_desde',vec_autorizacion.instante_lote_admin_v1(desde),'vigente_hasta',vec_autorizacion.instante_lote_admin_v1(hasta),'revocada_en','0001-01-01T00:00:00Z');
   asig_ref:='asignacion:'||asig_id||':v'||ver;
   INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
   VALUES(asig_ref,asig_id,ver,o->>'PerfilRef',persona,cambio->>'RolVersionRef',encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),(doc->>'emitida_en')::timestamptz,doc);
   INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(asig_ref,txid_current(),m->>'OperacionRef',x.auditoria_ref);
   INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES(o->>'PerfilRef',asig_ref,ahora,m->>'ActorPersonaRef',acto);
   inicios:=inicios||jsonb_build_array(jsonb_build_object('modo',cambio->>'InicioVigencia','vigente_desde',vec_autorizacion.instante_lote_admin_v1(desde)));
  ELSE
   IF pre#>>'{contexto,perfil,persona_ref}' IS DISTINCT FROM persona OR pre#>>'{contexto,vinculo,persona_ref}' IS DISTINCT FROM persona
   OR pre#>>'{contexto,vinculo,cuenta_ref}' IS DISTINCT FROM o->>'CuentaRef' OR pre#>>'{contexto,vinculo,perfil_ref}' IS DISTINCT FROM o->>'PerfilRef'
   OR (pre#>>'{contexto,perfil,version}')::numeric IS DISTINCT FROM (o->>'PerfilVersion')::numeric
   OR (pre#>>'{contexto,vinculo,version}')::numeric IS DISTINCT FROM (o->>'VinculoVersion')::numeric
   OR pre#>>'{asignacion,version_rol_ref}' IS DISTINCT FROM cambio->>'RolVersionRef' OR pre#>>'{asignacion,estado}' IS DISTINCT FROM 'activa'
   OR pre#>>'{asignacion,principal_id}' IS DISTINCT FROM persona
   OR pre#>'{asignacion,ambitos}' IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
   THEN RAISE EXCEPTION 'AUT44: baja sin vinculo y asignacion exactos' USING ERRCODE='40001'; END IF;
   SELECT * INTO STRICT ant FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=pre#>>'{asignacion,asignacion_ref}';
   PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_lote_v1(o->>'CuentaRef',persona,o->>'PerfilRef',o->>'VinculoRef',
    (o->>'CuentaVersion')::numeric,(o->>'PersonaVersion')::numeric,(o->>'PerfilVersion')::numeric,(o->>'VinculoVersion')::numeric,procedencia,1,sol,ahora);
   asig_id:=ant.asignacion_id;ver:=ant.version+1;desde:=(ant.documento->>'vigente_desde')::timestamptz;hasta:=(ant.documento->>'vigente_hasta')::timestamptz;
   doc:=jsonb_set(jsonb_set(ant.documento,'{version}',to_jsonb(ver)),'{estado}','"revocada"'::jsonb)
    ||jsonb_build_object('revocada_por',m->>'ActorPersonaRef','revocada_en',vec_autorizacion.instante_lote_admin_v1(ahora),'revocacion_ref',acto);
   asig_ref:='asignacion:'||asig_id||':v'||ver;
   INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
   VALUES(asig_ref,asig_id,ver,o->>'PerfilRef',persona,cambio->>'RolVersionRef',encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),(doc->>'emitida_en')::timestamptz,doc);
   INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(asig_ref,txid_current(),m->>'OperacionRef',x.auditoria_ref);
   UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=asig_ref,actualizada_en=ahora,actualizada_por=m->>'ActorPersonaRef',acto_ref=acto
   WHERE perfil_activo_ref=o->>'PerfilRef' AND asignacion_ref=ant.asignacion_ref;
   IF NOT FOUND THEN RAISE EXCEPTION 'AUT44: CAS de asignacion perdido' USING ERRCODE='40001'; END IF;
   inicios:=inicios||jsonb_build_array(jsonb_build_object('modo','','vigente_desde','0001-01-01T00:00:00Z'));
  END IF;
  post:=vec_autorizacion.preimagen_cambio_lote_admin_v1(cambio,org);
  cambios:=cambios||jsonb_build_array(jsonb_build_object('centro_ref','','actor_persona_ref',m->>'ActorPersonaRef','perfil_activo_ref',m->>'PerfilActivoRef',
   'asignacion_perfil_ref',m->>'AsignacionRef','correlacion_ref',d->>'correlacion_ref','rol_version_ref',cambio->>'RolVersionRef',
   'vigente_desde',vec_autorizacion.instante_lote_admin_v1(desde),'vigente_hasta',vec_autorizacion.instante_lote_admin_v1(hasta),'motivo',m->'Motivo',
   'operacion_ref',m->>'OperacionRef','acto_ref',acto,'recibo_ref',recibo_ref,'propuesta_ref','','auditoria_ref',x.auditoria_ref,
   'objetivo_persona_ref',persona,'perfil_ref',o->>'PerfilRef','vinculo_ref',o->>'VinculoRef',
   'estado_posterior',CASE WHEN cambio->>'Operacion'='otorgar' THEN 'activo' ELSE 'revocado' END,
   'version_posterior',CASE WHEN cambio->>'Operacion'='otorgar' THEN 1 ELSE (o->>'VinculoVersion')::numeric+1 END,
   'huella_antes_sha256',o->>'HuellaSHA256','huella_despues_sha256',encode(sha256(convert_to(post::text,'UTF8')),'hex'),
   'confirmado_en',vec_autorizacion.instante_lote_admin_v1(ahora),'unidad_ref',unidad,'referencia_acto',m->>'ReferenciaActo'));
  i:=i+1;
 END LOOP;
 recibo:=jsonb_build_object('operacion_ref',m->>'OperacionRef','acto_ref',acto,'recibo_ref',recibo_ref,'auditoria_ref',x.auditoria_ref,
  'huella_solicitud_sha256',sol,'fuentes_sha256',fuentes_sha,'confirmado_en',vec_autorizacion.instante_lote_admin_v1(ahora),'cambios',cambios,'inicios',inicios);
 INSERT INTO vec_autorizacion.registro_lote_admin_v1 VALUES(m->>'OperacionRef',convert_to(p_material,'UTF8'),sol,m->>'ActorPersonaRef',m->>'PerfilActivoRef',
  x.decision_ref,x.auditoria_ref,p_fuentes,recibo,clock_timestamp());
 INSERT INTO vec_autorizacion.outbox_lote_admin_v1 VALUES(m->>'OperacionRef','perfiles_lote_aplicado',x.auditoria_ref,clock_timestamp());
 IF vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() IS NOT TRUE THEN RAISE EXCEPTION 'AUT44: login revocado' USING ERRCODE='42501'; END IF;
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_lote_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() TO vec_admin_perfiles_lote_ejecutor;
-- El adaptador del lote coteja cada alta con el perfil registrado (AUT24
-- resolver_rol_administrable_v1, sólo lectura del catálogo) antes de gastar una
-- decisión. Sin este permiso toda alta acabaría en 42501 en la composición real.
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_rol_administrable_v1(text) TO vec_admin_perfiles_lote_ejecutor;
RESET ROLE;
DO $acl$
DECLARE g oid:=to_regrole('vec_admin_perfiles_lote_ejecutor');f text;
BEGIN
 FOREACH f IN ARRAY ARRAY['vec_autorizacion.instante_lote_admin_v1(timestamptz)','vec_autorizacion.validar_material_lote_admin_v1(text)',
  'vec_autorizacion.recurso_lote_admin_v1(text,jsonb)','vec_autorizacion.preimagen_cambio_lote_admin_v1(jsonb,text)'] LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f::regprocedure AND a.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'AUT44: PARO clave=ACL_privada actual=ampliada esperado=solo_propietario %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY['vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()'] LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f::regprocedure AND a.grantee NOT IN(p.proowner,g))
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f::regprocedure AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
  THEN RAISE EXCEPTION 'AUT44: PARO clave=ACL_fachada actual=ampliada esperado=propietario_y_grupo %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_autorizacion.resolver_rol_administrable_v1(text)'::regprocedure AND (a.grantee NOT IN(p.proowner,g) OR a.is_grantable))
 OR NOT has_function_privilege(g,'vec_autorizacion.resolver_rol_administrable_v1(text)','EXECUTE')
 THEN RAISE EXCEPTION 'AUT44: PARO clave=ACL_catalogo actual=divergente esperado=propietario_y_grupo' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid IN('vec_autorizacion.registro_lote_admin_v1'::regclass,'vec_autorizacion.outbox_lote_admin_v1'::regclass) AND a.grantee<>c.relowner)
 THEN RAISE EXCEPTION 'AUT44: PARO clave=ACL_tablas actual=ampliada esperado=solo_propietario' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
