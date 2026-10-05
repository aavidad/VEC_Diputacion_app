\set ON_ERROR_STOP on
-- AUT50: preparación del cambio de perfiles para la pantalla de Administración.
-- Para una persona del conjunto del administrador (organización y unidad),
-- devuelve lo que la pantalla necesita para construir una orden del lote v3:
-- cuenta ordinaria, versiones y procedencia de la persona; perfiles que puede
-- recibir (registrados para el lote) con referencias nuevas y la huella de la
-- preimagen de cada alta; y sus perfiles actuales en esa unidad con la huella
-- de cada baja. Consume una decisión nominal por AD190 (acción del lote,
-- recurso de preparación) y no escribe ningún efecto: sólo el consumo y un
-- registro de la preparación. Lo ejecuta el LOGIN del lote.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT50: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regprocedure('vec_autorizacion.aplicar_lote_ordinario_admin_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion.preimagen_cambio_lote_admin_v1(jsonb,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.cuentas_titular_persona_admin_lote_v1(text)') IS NULL
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.cuentas_titular_persona_admin_lote_v1(text)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.enlace_perfil_admin_interno_v1(text,text)','EXECUTE') IS NOT TRUE
 OR to_regclass('vec_autorizacion.registro_preparacion_lote_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT50: PARO clave=dependencias actual=divergente esperado=AUT44_sin_AUT50' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Rastro propio de cada preparación: distingue en la auditoría común un
-- consumo de preparación de uno de aplicación (mismo auditoria_ref).
CREATE TABLE vec_autorizacion.registro_preparacion_lote_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^prep_admin:[0-9a-f]{32}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 actor_persona_ref text NOT NULL,
 persona_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
ALTER TABLE vec_autorizacion.registro_preparacion_lote_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.registro_preparacion_lote_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.registro_preparacion_lote_admin_v1 FOR ALL TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.registro_preparacion_lote_admin_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.registro_preparacion_lote_admin_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.registro_preparacion_lote_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.registro_preparacion_lote_admin_v1 FROM PUBLIC;

-- Material de la preparación (canon Go): 8 claves de texto, sin null.
CREATE FUNCTION vec_autorizacion.validar_material_preparacion_lote_admin_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN RAISE EXCEPTION 'AUT50: material invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (SELECT count(*) FROM jsonb_object_keys(m))<>8
 OR NOT m ?& ARRAY['Esquema','OperacionRef','ActorPersonaRef','PerfilActivoRef','AsignacionRef','OrganizacionRef','UnidadRef','PersonaRef']
 OR EXISTS(SELECT 1 FROM jsonb_each(m) e WHERE jsonb_typeof(e.value)<>'string')
 OR m->>'Esquema' IS DISTINCT FROM 'administracion_perfiles_lote_preparacion:v1'
 OR m->>'OperacionRef' !~ '^prep_admin:[0-9a-f]{32}$'
 OR m->>'ActorPersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR m->>'PersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'PerfilActivoRef' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'AsignacionRef' !~ '^asignacion:[A-Za-z0-9_:-]{1,200}:v[1-9][0-9]{0,9}$'
 OR m->>'OrganizacionRef' !~ '^[a-z][a-z0-9_:-]{2,127}$' OR m->>'UnidadRef' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR m->>'PersonaRef'=m->>'ActorPersonaRef'
 THEN RAISE EXCEPTION 'AUT50: material de preparacion invalido' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_material_preparacion_lote_admin_v1(text) FROM PUBLIC;

-- Mismo canon que domain.RecursoAutorizable.HuellaContextoAutorizacionSHA256,
-- con atributo propio para no confundirse con la huella de una solicitud.
CREATE FUNCTION vec_autorizacion.recurso_preparacion_lote_admin_v1(p_material text,m jsonb)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 WITH x AS (SELECT '{"ambitos":{"organizacion_ref":'||to_jsonb(m->>'OrganizacionRef')::text||',"unidad_ref":'||to_jsonb(m->>'UnidadRef')::text||'},"atributos":{"preparacion_sha256":'||to_jsonb(encode(sha256(convert_to(p_material,'UTF8')),'hex'))::text||'}}' AS canon)
 SELECT jsonb_build_object('recurso_ref',m->>'PersonaRef','contexto_sha256',encode(sha256(convert_to(canon,'UTF8')),'hex')) FROM x
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recurso_preparacion_lote_admin_v1(text,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.preparar_lote_ordinario_admin_v1(p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;rec jsonb;x record;actor record;org text;unidad text;persona text;ahora timestamptz;
 cuenta text;cuentas jsonb;base jsonb;enlace jsonb;cambio jsonb;pre jsonb;r record;altas jsonb:='[]'::jsonb;bajas jsonb:='[]'::jsonb;
 perfil text;vinculo text;asig record;truncado boolean:=false;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR vec_autorizacion.acreditar_login_lote_ordinario_admin_v1() IS NOT TRUE
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 THEN RAISE EXCEPTION 'AUT50: preparacion denegada' USING ERRCODE='42501'; END IF;
 m:=vec_autorizacion.validar_material_preparacion_lote_admin_v1(p_material);
 d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 rec:=vec_autorizacion.recurso_preparacion_lote_admin_v1(p_material,m);
 IF d->>'recurso_ref' IS DISTINCT FROM rec->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM rec->>'contexto_sha256'
 OR c->>'efecto_ref' IS DISTINCT FROM rec->>'recurso_ref' OR c->>'huella_efecto_sha256' IS DISTINCT FROM rec->>'contexto_sha256'
 OR d->>'principal_id' IS DISTINCT FROM m->>'ActorPersonaRef' OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'PerfilActivoRef'
 OR d->>'asignacion_ref' IS DISTINCT FROM m->>'AsignacionRef'
 THEN RAISE EXCEPTION 'AUT50: decision ajena a la preparacion' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.efecto_ref IS DISTINCT FROM rec->>'recurso_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM rec->>'contexto_sha256'
 THEN RAISE EXCEPTION 'AUT50: consumo ajeno a la preparacion' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 org:=m->>'OrganizacionRef';unidad:=m->>'UnidadRef';persona:=m->>'PersonaRef';ahora:=clock_timestamp();
 -- Mismas reglas de ámbito que el efecto (AUT44).
 SELECT q.* INTO actor FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
 WHERE q.perfil_activo_ref=m->>'PerfilActivoRef' AND a.asignacion_ref=m->>'AsignacionRef' AND a.principal_id=m->>'ActorPersonaRef'
 AND a.documento->>'estado'='activa'
 AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)))
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='unidad_ref' AND e->'valores' ? unidad);
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT50: ambito del actor insuficiente' USING ERRCODE='42501'; END IF;
 IF vec_contexto_actor_v1.metadatos_persona_administrable_v1(persona) IS NULL
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  WHERE a.principal_id=persona AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad))))
 THEN RAISE EXCEPTION 'AUT50: persona fuera del conjunto' USING ERRCODE='42501'; END IF;
 -- Cuenta ordinaria única y activa de la persona (por la fachada de CA35).
 cuentas:=vec_contexto_actor_v1.cuentas_titular_persona_admin_lote_v1(persona);
 IF (SELECT count(*) FROM jsonb_array_elements_text(cuentas) cta WHERE vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(cta) IS FALSE)<>1
 THEN RAISE EXCEPTION 'AUT50: cuenta ordinaria ausente o ambigua' USING ERRCODE='42501'; END IF;
 SELECT cta INTO STRICT cuenta FROM jsonb_array_elements_text(cuentas) cta WHERE vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(cta) IS FALSE;
 -- Versiones y procedencia de la persona y de la cuenta, de la misma preimagen
 -- que usa el efecto (referencias de perfil y vínculo inexistentes).
 base:=vec_contexto_actor_v1.preimagen_admin_interna_v1(cuenta,persona,'prf_'||repeat('0',32),'vca_'||repeat('0',32));
 -- Altas posibles: perfiles registrados para el lote y vigentes que la persona
 -- no tiene ya activos en esa unidad. Referencias nuevas para cada uno.
 FOR r IN SELECT ra.version_rol_ref,ra.unidad_requerida,ra.vigente_desde,ra.vigente_hasta,ra.duracion_propuesta,vr.documento->>'nombre' AS nombre
  FROM vec_autorizacion.rol_administrable_exacto_v1 ra JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol_actual cq USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol cc ON cc.version_rol_ref=cq.version_rol_ref AND cc.revision=cq.revision
  WHERE ra.clase='ordinario' AND ra.huella_sha256=vr.huella_sha256 AND vr.documento->>'estado'='publicada' AND cc.estado='habilitada'
  AND ahora>=ra.vigente_desde AND ahora<ra.vigente_hasta
  AND ra.audiencia_administrativa='vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
  AND ra.ambitos_fijos=jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)))
  AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
   WHERE a.principal_id=persona AND a.version_rol_ref=ra.version_rol_ref AND a.documento->>'estado'='activa' AND ahora<(a.documento->>'vigente_hasta')::timestamptz
   AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad))))
  ORDER BY ra.version_rol_ref COLLATE "C" LIMIT 65 LOOP
  IF jsonb_array_length(altas)=64 THEN truncado:=true; EXIT; END IF;
  perfil:='prf_'||replace(gen_random_uuid()::text,'-','');vinculo:='vca_'||replace(gen_random_uuid()::text,'-','');
  cambio:=jsonb_build_object('Operacion','otorgar','RolVersionRef',r.version_rol_ref,'Objetivo',jsonb_build_object('UnidadRef',unidad,'CuentaRef',cuenta,'PersonaRef',persona,'PerfilRef',perfil,'VinculoRef',vinculo));
  pre:=vec_autorizacion.preimagen_cambio_lote_admin_v1(cambio,org);
  altas:=altas||jsonb_build_array(jsonb_build_object('rol_version_ref',r.version_rol_ref,'nombre',coalesce(r.nombre,''),'unidad_requerida',r.unidad_requerida,
   'vigente_hasta_maxima',vec_autorizacion.instante_lote_admin_v1(r.vigente_hasta),'duracion_propuesta_segundos',extract(epoch FROM r.duracion_propuesta)::bigint,'perfil_ref',perfil,'vinculo_ref',vinculo,
   'huella_sha256',encode(sha256(convert_to(pre::text,'UTF8')),'hex')));
 END LOOP;
 -- Bajas posibles: asignaciones activas de la persona en esa organización y unidad
 -- de perfiles registrados como ordinarios, con su vínculo y versiones.
 -- El nombre del perfil sale del documento de su versión (como en las altas),
 -- para que la pantalla no muestre referencias.
 FOR asig IN SELECT x2.*,coalesce(vr2.documento->>'nombre','') AS nombre_rol FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x2 USING(asignacion_ref)
  JOIN vec_autorizacion.rol_administrable_exacto_v1 ra ON ra.version_rol_ref=x2.version_rol_ref AND ra.clase='ordinario'
  JOIN vec_autorizacion.version_rol vr2 ON vr2.version_rol_ref=x2.version_rol_ref
  WHERE x2.principal_id=persona AND x2.documento->>'estado'='activa'
  AND x2.documento->'ambitos'=jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(org)),jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(unidad)))
  ORDER BY x2.perfil_activo_ref COLLATE "C" LIMIT 65 LOOP
  IF jsonb_array_length(bajas)=64 THEN truncado:=true; EXIT; END IF;
  -- Un perfil sin vínculo único no se puede retirar por el lote; se omite
  -- sin impedir el resto de la preparación.
  BEGIN
   enlace:=vec_contexto_actor_v1.enlace_perfil_admin_interno_v1(persona,asig.perfil_activo_ref);
  EXCEPTION WHEN insufficient_privilege THEN CONTINUE;
  END;
  CONTINUE WHEN enlace->>'cuenta_ref' IS DISTINCT FROM cuenta;
  cambio:=jsonb_build_object('Operacion','revocar','RolVersionRef',asig.version_rol_ref,'Objetivo',jsonb_build_object('UnidadRef',unidad,'CuentaRef',cuenta,'PersonaRef',persona,'PerfilRef',asig.perfil_activo_ref,'VinculoRef',enlace->>'vinculo_ref'));
  pre:=vec_autorizacion.preimagen_cambio_lote_admin_v1(cambio,org);
  bajas:=bajas||jsonb_build_array(jsonb_build_object('rol_version_ref',asig.version_rol_ref,'nombre',asig.nombre_rol,'perfil_ref',asig.perfil_activo_ref,'vinculo_ref',enlace->>'vinculo_ref',
   'perfil_version',pre#>'{contexto,perfil,version}','vinculo_version',pre#>'{contexto,vinculo,version}',
   'vigente_desde',asig.documento->>'vigente_desde','vigente_hasta',asig.documento->>'vigente_hasta',
   'huella_sha256',encode(sha256(convert_to(pre::text,'UTF8')),'hex')));
 END LOOP;
 INSERT INTO vec_autorizacion.registro_preparacion_lote_admin_v1 VALUES(m->>'OperacionRef',encode(sha256(convert_to(p_material,'UTF8')),'hex'),
  m->>'ActorPersonaRef',persona,x.decision_ref,x.auditoria_ref,ahora);
 RETURN jsonb_build_object('operacion_ref',m->>'OperacionRef','auditoria_ref',x.auditoria_ref,'preparada_en',vec_autorizacion.instante_lote_admin_v1(ahora),
  'persona_ref',persona,'persona_version',base#>'{persona,version}','cuenta_ref',cuenta,'cuenta_version',base#>'{cuenta,version}',
  'procedencia_ref',base#>>'{persona,procedencia_ref}','procedencia_version',base#>'{persona,procedencia_version}','procedencia_huella_sha256',base#>>'{persona,procedencia_huella_sha256}',
  'organizacion_ref',org,'unidad_ref',unidad,'altas',altas,'bajas',bajas,'truncado',truncado);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preparar_lote_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.preparar_lote_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_lote_ejecutor;
RESET ROLE;
DO $acl$
DECLARE g oid:=to_regrole('vec_admin_perfiles_lote_ejecutor');f text;
BEGIN
 FOREACH f IN ARRAY ARRAY['vec_autorizacion.validar_material_preparacion_lote_admin_v1(text)','vec_autorizacion.recurso_preparacion_lote_admin_v1(text,jsonb)'] LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f::regprocedure AND a.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'AUT50: PARO clave=ACL_privada actual=ampliada %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_autorizacion.preparar_lote_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND a.grantee NOT IN(p.proowner,g))
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid='vec_autorizacion.registro_preparacion_lote_admin_v1'::regclass AND a.grantee<>c.relowner)
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.preparar_lote_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
 THEN RAISE EXCEPTION 'AUT50: PARO clave=ACL actual=ampliada esperado=propietario_y_grupo' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
