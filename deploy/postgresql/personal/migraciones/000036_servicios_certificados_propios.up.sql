\set ON_ERROR_STOP on
-- Personal 000036: Personal sirve a Certificados los servicios de la propia
-- persona (LectorServiciosParaCertificadosV1/V2), sólo en autoservicio.
-- El empleado procede de la proyección canónica de la persona que consulta
-- (Personal16) y se revalida aquí; nunca de la petición. La consulta de RRHH
-- sobre otra persona exige otra competencia y no existe en esta migración.
-- Lectura al corte bitemporal de Personal17 (última revisión conocida de cada
-- servicio, vigente en la fecha del corte), consumo V3 propio (AD195) y
-- auditoría común en la misma transacción SERIALIZABLE. El recibo es la PK de
-- esa auditoría común: no hay tabla nueva de recibos ni de auditoría.
-- Dependencias: Personal16/17/20 y AD195 instaladas. Una sola vez; sin DOWN.
-- El plazo de la sentencia lo fija la transacción del adaptador (SET LOCAL
-- statement_timeout); dentro de la función no limitaría a quien la llama.
-- Una capacidad se consume una vez: si se pierde la respuesta de un COMMIT,
-- el reintento con la misma capacidad se deniega y hay que pedir otra decisión.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000036:servicios-certificados',0));
SET LOCAL ROLE vec_personal_propietario;
DO $pre$
DECLARE consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_personal_propietario' OR consumidor IS NULL
 OR to_regprocedure('vec_personal.consultar_servicios_certificados_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)') IS NULL
 OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NULL
 OR (SELECT count(*) FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace
     WHERE n.nspname='vec_personal' AND t.relname IN ('servicio_reconocido_historia','relacion_servicio_historia')
       AND t.relkind='r' AND t.relowner='vec_personal_propietario'::regrole AND t.relrowsecurity AND t.relforcerowsecurity)<>2
 OR (SELECT count(*) FROM pg_attribute a WHERE a.attrelid=to_regclass('vec_personal.servicio_reconocido_historia')
     AND NOT a.attisdropped AND a.attnum>0 AND a.attname IN ('servicio_ref','revision','relacion_ref','relacion_revision',
       'empleado_ref','organismo_ref','clase_ref','dias_reconocidos','periodo_desde','periodo_hasta','estado',
       'vigente_desde','vigente_hasta','conocido_desde','acto_ref','fuente_ref','fuente_version',
       'firma_oficial','eficacia_administrativa','catalogo_snapshot'))<>20
 OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid IN (to_regclass('vec_personal.servicio_reconocido_historia'),
       to_regclass('vec_personal.relacion_servicio_historia')) AND p.polname='propietario_interno' AND p.polpermissive AND p.polcmd='*'
       AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid] AND pg_get_expr(p.polqual,p.polrelid)='true')<>2
 OR NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=to_regclass('vec_personal.servicio_reconocido_historia')
     AND a.attname='catalogo_snapshot' AND NOT a.attisdropped AND a.attnotnull AND a.atttypid='jsonb'::regtype)
 OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname='vec_personal_ejecutor' AND NOT r.rolcanlogin
     AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'PARO clave=Personal36.preimagen esperado=Personal16_17_20_AD195_sin_36 actual=incompatible' USING ERRCODE='55000'; END IF;
 IF NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE')
 OR has_function_privilege('vec_personal_ejecutor',consumidor,'EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=consumidor
     AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proretset
     AND p.prorettype='record'::regtype AND p.pronargs=10
     AND p.proargnames[11:17]=ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=consumidor AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole)
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'PARO clave=Personal36.consumidor esperado=AD195_privado actual=incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_personal.consultar_servicios_certificados_propios_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC'
 SET datestyle='ISO, YMD' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; x jsonb; vinculo jsonb; proyeccion record; consumo record;
 fecha date; conocido timestamptz(6); ahora timestamptz(6);
 cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;
 actor_desde timestamptz; actor_hasta timestamptz; actor_resuelto timestamptz; v_desde timestamptz; v_hasta timestamptz;
 empleado text; organismo text; persona text;
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 servicios jsonb; cardinalidad integer; version_fuente bigint;
 campos constant jsonb:='["cobertura","corte","evidencia","servicios"]';
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
     AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members r WHERE r.member=session_user::regrole AND r.roleid='vec_personal_ejecutor'::regrole
     AND r.inherit_option AND NOT r.set_option AND NOT r.admin_option)
 OR (SELECT count(*) FROM pg_auth_members r WHERE r.member=session_user::regrole)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members r WHERE r.member='vec_personal_ejecutor'::regrole)
 OR has_table_privilege(session_user,'vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 OR has_table_privilege(session_user,'vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal36: lectura de servicios denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal36: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(x) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'Personal36: material no es objeto' USING ERRCODE='22023'; END IF;
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
   'empleado_ref','esquema','operacion','organismo_ref','perfil_ref','perfil_version',
   'persona_ref','persona_version','vigente_en']
 OR EXISTS(SELECT 1 FROM jsonb_each(m) e WHERE
   (e.key IN ('contexto_version','cuenta_version','perfil_version','persona_version') AND jsonb_typeof(e.value)<>'number')
   OR (e.key NOT IN ('contexto_version','cuenta_version','perfil_version','persona_version') AND jsonb_typeof(e.value)<>'string'))
 OR m->>'esquema' IS DISTINCT FROM 'vec.personal.servicios-certificados.consulta.v1'
 OR m->>'operacion' IS DISTINCT FROM 'servicios_certificados_propios'
 OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
 OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'contexto_actor_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'vigente_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR left(m->>'vigente_en',4)='0000'
 OR m->>'conocido_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR EXISTS(SELECT 1 FROM jsonb_each_text(m) e WHERE e.key IN ('contexto_version','cuenta_version','perfil_version','persona_version')
   AND e.value !~ '^[1-9][0-9]{0,18}$') THEN
  RAISE EXCEPTION 'Personal36: material incompatible' USING ERRCODE='22023'; END IF;
 BEGIN
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
  cap_desde:=(c->>'emitida_en')::timestamptz; cap_hasta:=(c->>'expira_en')::timestamptz;
  dec_hasta:=(d->>'valida_hasta')::timestamptz;
  actor_desde:=(x->>'vigente_desde')::timestamptz; actor_hasta:=(x->>'vigente_hasta')::timestamptz;
  actor_resuelto:=(x->>'resuelto_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal36: fechas inválidas' USING ERRCODE='22023'; END;
 ahora:=clock_timestamp();
 empleado:=m->>'empleado_ref'; organismo:=m->>'organismo_ref'; persona:=m->>'persona_ref';
 IF fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido) OR conocido>ahora
 OR fecha::text IS DISTINCT FROM m->>'vigente_en'
 OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
 OR (m->>'persona_version')::numeric>9223372036854775807 OR (m->>'perfil_version')::numeric>9223372036854775807
 OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
 OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 OR m->>'actor_ref' IS DISTINCT FROM persona
 OR m->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
 OR m->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
 OR m->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
 OR m->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
 OR m->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
 OR m->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
 OR m->>'perfil_version' IS DISTINCT FROM x->>'perfil_version'
 OR persona IS DISTINCT FROM x->>'persona_ref'
 OR m->>'persona_version' IS DISTINCT FROM x->>'persona_version'
 OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR x->>'estado' IS DISTINCT FROM 'activo'
 OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 OR cap_desde IS NULL OR NOT isfinite(cap_desde) OR cap_hasta IS NULL OR NOT isfinite(cap_hasta)
 OR dec_hasta IS NULL OR NOT isfinite(dec_hasta)
 OR actor_desde IS NULL OR NOT isfinite(actor_desde) OR actor_hasta IS NULL OR NOT isfinite(actor_hasta)
 OR actor_resuelto IS NULL OR NOT isfinite(actor_resuelto)
 OR ahora<cap_desde OR ahora>=cap_hasta OR ahora>=dec_hasta
 OR ahora<actor_desde OR ahora>=actor_hasta OR ahora<actor_resuelto
 OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'
 OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
 OR d->>'modulo_id' IS DISTINCT FROM 'personal'
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa' THEN
  RAISE EXCEPTION 'Personal36: actor o vigencia divergente' USING ERRCODE='42501'; END IF;
 -- Canon exacto json.Marshal de los 15 campos en el orden del contrato Go.
 material_canon:='{"esquema":"vec.personal.servicios-certificados.consulta.v1","operacion":"servicios_certificados_propios"'||
  ',"empleado_ref":'||to_jsonb(empleado)::text||',"organismo_ref":'||to_jsonb(organismo)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(persona)::text||
  ',"persona_version":'||(m->>'persona_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'Personal36: material no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 -- RecursoAutorizable.HuellaContextoAutorizacionSHA256: mapas en orden Go.
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||',"organismo_ref":'||to_jsonb(organismo)::text||
  '},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||',"material_sha256":"'||material_sha||
  '","operacion":"servicios_certificados_propios","vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.servicios_certificados.consultar'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'servicios_certificados'
 OR d->>'finalidad' IS DISTINCT FROM 'consultar_servicios_para_certificados'
 OR d->'campos_permitidos' IS DISTINCT FROM campos
 OR d->>'recurso_ref' IS DISTINCT FROM empleado
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
 OR c->>'operacion' IS DISTINCT FROM d->>'accion'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.servicios_certificados.v1'
 OR c->>'efecto_ref' IS DISTINCT FROM empleado
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'Personal36: concesión divergente' USING ERRCODE='42501'; END IF;
 -- Autoservicio: el empleado pedido es el único vínculo de empleado del actor
 -- y, ahora, el empleado canónico de su persona según la proyección de
 -- Personal16. La barrera de lectores convierte una publicación concurrente en
 -- 40001. Ausencia, ambigüedad, revocación o empleado ajeno se deniegan igual.
 IF (SELECT count(*) FROM jsonb_array_elements(x->'vinculos') v(e) WHERE v.e->>'tipo'='empleado')<>1 THEN
  RAISE EXCEPTION 'Personal36: vínculo de empleado ambiguo' USING ERRCODE='42501'; END IF;
 SELECT v.e INTO STRICT vinculo FROM jsonb_array_elements(x->'vinculos') v(e) WHERE v.e->>'tipo'='empleado';
 IF vinculo->>'referencia' IS DISTINCT FROM empleado OR vinculo->>'estado' IS DISTINCT FROM 'activo'
 OR coalesce(vinculo->>'vinculo_ref','') !~ '^pep_[A-Za-z0-9_-]{22,128}$'
 OR jsonb_typeof(vinculo->'version') IS DISTINCT FROM 'number' OR coalesce(vinculo->>'version','') !~ '^[1-9][0-9]{0,18}$' THEN
  RAISE EXCEPTION 'Personal36: vínculo de empleado divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,ahora);
 IF proyeccion.resultado IS DISTINCT FROM 'empleado' OR proyeccion.persona_ref IS DISTINCT FROM persona
 OR proyeccion.empleado_ref IS DISTINCT FROM empleado OR proyeccion.proyeccion_ref IS DISTINCT FROM vinculo->>'vinculo_ref'
 OR proyeccion.version::text IS DISTINCT FROM vinculo->>'version' THEN
  RAISE EXCEPTION 'Personal36: empleado propio no vigente' USING ERRCODE='42501'; END IF;
 -- AD195 revalida firma, gobierno, revocación, origen técnico y vigencia y
 -- escribe consumo y auditoría común; un fallo posterior los revierte.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM empleado OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR coalesce(consumo.consumo_huella_sha256,'') !~ '^[0-9a-f]{64}$'
 OR consumo.auditoria_ref IS DISTINCT FROM 'aud_v3_'||left(consumo.consumo_huella_sha256,32) THEN
  RAISE EXCEPTION 'Personal36: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Una revisión de servicio del empleado ligada a una relación de otra persona
 -- es una incoherencia de la fuente: no se devuelve nada.
 IF EXISTS(SELECT 1 FROM vec_personal.servicio_reconocido_historia s
   WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido
     AND NOT EXISTS(SELECT 1 FROM vec_personal.relacion_servicio_historia r
       WHERE r.relacion_ref=s.relacion_ref AND r.revision=s.relacion_revision AND r.empleado_ref=s.empleado_ref
         AND r.organismo_ref=s.organismo_ref AND r.persona_ref=persona)) THEN
  RAISE EXCEPTION 'Personal36: fuente incoherente' USING ERRCODE='55000'; END IF;
 -- Corte bitemporal: la última revisión conocida de cada servicio manda; se
 -- incluye si su registro está vigente en la fecha del corte y el servicio ya
 -- había comenzado. Periodo [desde,hasta) y días tal como constan, sin recalcular.
 SELECT count(*) INTO cardinalidad FROM (
  SELECT 1 FROM (SELECT DISTINCT ON (s.servicio_ref) s.* FROM vec_personal.servicio_reconocido_historia s
    WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido
    ORDER BY s.servicio_ref,s.conocido_desde DESC,s.revision DESC) u
  WHERE u.vigente_desde<=fecha AND (u.vigente_hasta IS NULL OR fecha<u.vigente_hasta) AND u.periodo_desde<=fecha
  LIMIT 201) limite;
 IF cardinalidad>200 THEN RAISE EXCEPTION 'Personal36: excede límite' USING ERRCODE='54000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_personal.servicio_reconocido_historia s
   WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido
     AND (s.firma_oficial IS DISTINCT FROM false OR s.eficacia_administrativa IS DISTINCT FROM false)) THEN
  RAISE EXCEPTION 'Personal36: fuente incompatible' USING ERRCODE='55000'; END IF;
 WITH ultima AS (SELECT DISTINCT ON (s.servicio_ref) s.* FROM vec_personal.servicio_reconocido_historia s
    WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido
    ORDER BY s.servicio_ref,s.conocido_desde DESC,s.revision DESC)
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'servicio_ref',u.servicio_ref,'relacion_ref',u.relacion_ref,'version',u.revision,
   'periodo',jsonb_build_object('desde',u.periodo_desde::text,'hasta',u.periodo_hasta::text),
   'dias_reconocidos',u.dias_reconocidos,'estado',u.estado,
   'clase_ref',u.clase_ref,
   'clase_version',CASE WHEN jsonb_typeof(u.catalogo_snapshot#>'{clase_servicio,version}')='number'
     AND (u.catalogo_snapshot#>>'{clase_servicio,ref}') IS NOT DISTINCT FROM u.clase_ref
     THEN (u.catalogo_snapshot#>>'{clase_servicio,version}')::bigint ELSE 0 END,
   'procedencia',jsonb_build_object('acto_ref',u.acto_ref,'fuente_ref',u.fuente_ref,
     'fuente_version',u.fuente_version::text,'certeza','no_acreditado'))
   ORDER BY u.periodo_desde,u.servicio_ref COLLATE "C"),'[]'::jsonb)
 INTO servicios FROM ultima u
 WHERE u.vigente_desde<=fecha AND (u.vigente_hasta IS NULL OR fecha<u.vigente_hasta) AND u.periodo_desde<=fecha;
 IF jsonb_array_length(servicios) IS DISTINCT FROM cardinalidad THEN
  RAISE EXCEPTION 'Personal36: cardinalidad divergente' USING ERRCODE='55000'; END IF;
 -- Versión de la fuente al corte: revisiones de servicios conocidas del
 -- empleado en el organismo. Sólo crece por adición; no es un contador global.
 SELECT count(*) INTO version_fuente FROM vec_personal.servicio_reconocido_historia s
  WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF ahora<cap_desde OR ahora>=cap_hasta OR ahora>=dec_hasta OR ahora<actor_desde OR ahora>=actor_hasta
 OR ahora<actor_resuelto OR ahora<conocido
 OR ahora<proyeccion.vigente_desde OR (proyeccion.vigente_hasta IS NOT NULL AND ahora>=proyeccion.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal36: autorización caducada' USING ERRCODE='42501'; END IF;
 FOR vinculo IN SELECT value FROM jsonb_array_elements(x->'vinculos') LOOP
  BEGIN
   v_desde:=(vinculo->>'vigente_desde')::timestamptz; v_hasta:=(vinculo->>'vigente_hasta')::timestamptz;
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal36: vínculo inválido' USING ERRCODE='42501'; END;
  IF vinculo->>'estado' IS DISTINCT FROM 'activo' OR v_desde IS NULL OR v_hasta IS NULL
  OR NOT isfinite(v_desde) OR NOT isfinite(v_hasta) OR ahora<v_desde OR ahora>=v_hasta THEN
   RAISE EXCEPTION 'Personal36: vínculo caducado' USING ERRCODE='42501'; END IF;
 END LOOP;
 -- La cobertura de este registro no está acreditada y sus hechos carecen de
 -- eficacia administrativa (Personal17): certeza no_acreditado en todos.
 RETURN jsonb_build_object(
  'servicios',jsonb_build_object('empleado_ref',empleado,'organismo_ref',organismo,'version',version_fuente,
    'corte',jsonb_build_object('vigente_en',m->>'vigente_en','conocido_en',m->>'conocido_en'),
    'cobertura','no_acreditada','servicios',servicios),
  'evidencia',jsonb_build_object('recibo_ref',consumo.auditoria_ref,'decision_ref',consumo.decision_ref,
    'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
    'auditoria_ref',consumo.auditoria_ref,'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_servicios_certificados_propios_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_servicios_certificados_propios_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
DO $post$
DECLARE f oid:=to_regprocedure('vec_personal.consultar_servicios_certificados_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole AND p.prosecdef)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR has_table_privilege('vec_personal_ejecutor','vec_personal.servicio_reconocido_historia','SELECT')
 THEN RAISE EXCEPTION 'PARO clave=Personal36.ACL esperado=propietario_y_ejecutor actual=ampliada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
