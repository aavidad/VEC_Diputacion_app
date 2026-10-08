\set ON_ERROR_STOP on
-- BORRADOR Personal34: no ensayado ni instalado. Sólo lectura nominal propia.
-- Dependencias reales: Personal16/17/20/22 y AD180 FINAL aprobado e instalado.
-- AD180 está cerrado por su preimagen pendiente; su ausencia impide CREATE.
-- No cambia una migración instalada, no añade historia/recibos/outbox y no
-- acredita cobertura institucional, derechos ni documentos firmados.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000034:historia-servicios-propios',0));
DO $pre$
DECLARE consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_personal_propietario' OR consumidor IS NULL
    OR to_regprocedure('vec_personal.consultar_historia_servicios_propios_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)') IS NULL
    OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR (SELECT count(*) FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace
        WHERE n.nspname='vec_personal' AND t.relname IN ('servicio_reconocido_historia','relacion_servicio_historia')
          AND t.relkind='r' AND t.relowner='vec_personal_propietario'::regrole AND t.relrowsecurity AND t.relforcerowsecurity)<>2
    OR NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=to_regclass('vec_personal.servicio_reconocido_historia')
          AND a.attname='catalogo_snapshot' AND NOT a.attisdropped AND a.attnotnull AND a.atttypid='jsonb'::regtype)
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname='vec_personal_ejecutor' AND NOT r.rolcanlogin
          AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
 THEN RAISE EXCEPTION 'Personal34: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',consumidor,'EXECUTE')
    OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=consumidor
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proretset
          AND p.prorettype='record'::regtype AND p.pronargs=10
          AND p.proargnames[11:17]=ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
          WHERE p.oid=consumidor AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole)
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'Personal34: consumidor AD180 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_personal.consultar_historia_servicios_propios_empleado_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
 SET datestyle='ISO, YMD' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; ctx jsonb; vinculo jsonb; proyeccion record; consumo record;
 empleado text; persona text; desde date; hasta date; conocido timestamptz(6); ahora timestamptz(6);
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 revisiones jsonb; cardinalidad integer; k text;
 campos constant jsonb:='["cobertura","corte","evidencia","revisiones"]';
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
          AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members r WHERE r.member=session_user::regrole AND r.roleid='vec_personal_ejecutor'::regrole
          AND r.inherit_option AND NOT r.set_option AND NOT r.admin_option)
    OR (SELECT count(*) FROM pg_auth_members r WHERE r.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_auth_members r WHERE r.member='vec_personal_ejecutor'::regrole)
    OR EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname<>session_user AND r.rolname<>'vec_personal_ejecutor' AND pg_has_role(session_user,r.oid,'MEMBER'))
    OR has_table_privilege(session_user,'vec_personal.servicio_reconocido_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_table_privilege(session_user,'vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 131072
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 65536
    OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR p_raiz IS NULL OR octet_length(p_raiz)<>44 THEN
  RAISE EXCEPTION 'historia de servicios propios denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
  desde:=(m->>'efectos_desde')::date; hasta:=(m->>'efectos_hasta')::date; conocido:=(m->>'conocido_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de historia inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(ctx) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'material de historia inválido' USING ERRCODE='22023'; END IF;
 empleado:=m->>'empleado_ref'; persona:=ctx->>'persona_ref';
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['conocido_en','contexto_actor','efectos_desde','efectos_hasta','empleado_ref','esquema']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.historia-servicios-propia.consulta.v1'
    OR empleado IS NULL OR empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR desde IS NULL OR hasta IS NULL OR NOT isfinite(desde) OR NOT isfinite(hasta) OR desde>=hasta
    OR conocido IS NULL OR NOT isfinite(conocido) OR conocido>clock_timestamp()
    OR coalesce(m->>'efectos_desde','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR left(m->>'efectos_desde',4)='0000'
    OR coalesce(m->>'efectos_hasta','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR left(m->>'efectos_hasta',4)='0000'
    OR desde::text IS DISTINCT FROM m->>'efectos_desde' OR hasta::text IS DISTINCT FROM m->>'efectos_hasta'
    OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR m->'contexto_actor' IS DISTINCT FROM ctx
    OR ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR ctx->>'principal_ref' IS DISTINCT FROM persona OR ctx->>'estado' IS DISTINCT FROM 'activo'
    OR ctx->>'cuenta_ref' IS NULL OR ctx->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR ctx->>'perfil_activo_ref' IS NULL OR ctx->>'perfil_activo_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR ctx->>'contexto_actor_ref' IS NULL OR ctx->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR jsonb_typeof(ctx->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'material de historia incompatible' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','persona_version','perfil_version'] LOOP
  IF jsonb_typeof(ctx->k) IS DISTINCT FROM 'number' OR coalesce(ctx->>k,'') !~ '^[1-9][0-9]{0,18}$' THEN
   RAISE EXCEPTION 'versión de contexto inválida' USING ERRCODE='22023'; END IF;
  IF (ctx->>k)::numeric>9223372036854775807 THEN
   RAISE EXCEPTION 'versión de contexto inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF ctx->>'persona_version' IS DISTINCT FROM p_persona_version::text OR ctx->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR persona IS DISTINCT FROM d->>'principal_id' OR ctx->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref' THEN
  RAISE EXCEPTION 'actor de historia divergente' USING ERRCODE='42501'; END IF;
 -- RawMessage Go conserva el contexto canónico completo, no una selección de campos.
 material_canon:='{"esquema":"vec.personal.historia-servicios-propia.consulta.v1","empleado_ref":'||to_jsonb(empleado)::text||
  ',"efectos_desde":'||to_jsonb(m->>'efectos_desde')::text||',"efectos_hasta":'||to_jsonb(m->>'efectos_hasta')::text||
  ',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||',"contexto_actor":'||convert_from(p_contexto,'UTF8')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de historia no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 -- Recurso.HuellaContextoAutorizacionSHA256: mapas en orden lexicográfico Go.
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||'},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"efectos_desde":'||to_jsonb(m->>'efectos_desde')::text||',"efectos_hasta":'||to_jsonb(m->>'efectos_hasta')::text||
  ',"material_sha256":"'||material_sha||'","operacion":"historia_servicios_propia"}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.registro_empleado.servicios.historia_propia.consultar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'historia_servicios_propia'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_historia_servicios_propios' OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->>'recurso_ref' IS DISTINCT FROM empleado OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM d->>'accion' OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.servicios.historia_propia.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM empleado OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión de historia divergente' USING ERRCODE='42501'; END IF;
 IF (SELECT count(*) FROM jsonb_array_elements(ctx->'vinculos') v(e) WHERE v.e->>'tipo'='empleado')<>1 THEN
  RAISE EXCEPTION 'vínculo de empleado ambiguo' USING ERRCODE='42501'; END IF;
 SELECT v.e INTO STRICT vinculo FROM jsonb_array_elements(ctx->'vinculos') v(e) WHERE v.e->>'tipo'='empleado';
 IF vinculo->>'referencia' IS DISTINCT FROM empleado OR vinculo->>'estado' IS DISTINCT FROM 'activo'
    OR coalesce(vinculo->>'vinculo_ref','') !~ '^pep_[A-Za-z0-9_-]{22,128}$'
    OR jsonb_typeof(vinculo->'version') IS DISTINCT FROM 'number' OR coalesce(vinculo->>'version','') !~ '^[1-9][0-9]{0,18}$' THEN
  RAISE EXCEPTION 'vínculo de empleado divergente' USING ERRCODE='42501'; END IF;
 IF (vinculo->>'version')::numeric>9223372036854775807 THEN
  RAISE EXCEPTION 'versión de vínculo inválida' USING ERRCODE='22023'; END IF;
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,clock_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado' OR proyeccion.persona_ref IS DISTINCT FROM persona
    OR proyeccion.empleado_ref IS DISTINCT FROM empleado OR proyeccion.proyeccion_ref IS DISTINCT FROM vinculo->>'vinculo_ref'
    OR proyeccion.version::text IS DISTINCT FROM vinculo->>'version' THEN
  RAISE EXCEPTION 'empleado propio no vigente' USING ERRCODE='42501'; END IF;
 -- AD180 revalida contexto/perfil/persona, firma, gobierno y vigencia actuales.
 -- Consumo, lectura y auditoría común pertenecen a la misma TX del adaptador.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM empleado OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
    OR coalesce(consumo.consumo_huella_sha256,'') !~ '^[0-9a-f]{64}$'
    OR consumo.auditoria_ref IS DISTINCT FROM 'aud_v3_'||left(consumo.consumo_huella_sha256,32) THEN
  RAISE EXCEPTION 'consumo de historia divergente' USING ERRCODE='42501'; END IF;
 -- La detección de exceso cuenta hasta201; nunca devuelve un prefijo de filas.
 SELECT count(*) INTO cardinalidad FROM (SELECT 1 FROM vec_personal.servicio_reconocido_historia s
  WHERE s.empleado_ref=empleado AND s.conocido_desde<=conocido AND s.vigente_desde<hasta
    AND (s.vigente_hasta IS NULL OR desde<s.vigente_hasta) LIMIT 201) limite;
 IF cardinalidad>200 THEN RAISE EXCEPTION 'historia excede límite' USING ERRCODE='54000'; END IF;
 -- Incoherencia del vínculo de cualquier revisión: no excluirla silenciosamente.
 IF EXISTS(SELECT 1 FROM vec_personal.servicio_reconocido_historia s
  WHERE s.empleado_ref=empleado AND s.conocido_desde<=conocido AND s.vigente_desde<hasta
    AND (s.vigente_hasta IS NULL OR desde<s.vigente_hasta)
    AND NOT EXISTS(SELECT 1 FROM vec_personal.relacion_servicio_historia r WHERE r.relacion_ref=s.relacion_ref
      AND r.revision=s.relacion_revision AND r.empleado_ref=empleado AND r.organismo_ref=s.organismo_ref
      AND r.persona_ref=persona AND r.conocido_desde<=conocido)) THEN
  RAISE EXCEPTION 'historia propia incoherente' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'servicio_ref',s.servicio_ref,'relacion_ref',s.relacion_ref,'periodo_desde',s.periodo_desde::text,
  'periodo_hasta',s.periodo_hasta::text,'dias_reconocidos',s.dias_reconocidos,'estado',s.estado,
  'clase',coalesce(s.catalogo_snapshot#>>'{clase_servicio,denominacion}',''),
  'traza',jsonb_strip_nulls(jsonb_build_object('desde',s.vigente_desde::text,'hasta',s.vigente_hasta::text,
    'registrada_en',to_char(s.conocido_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'version',s.revision,'acto_ref',s.acto_ref,'fuente_ref',s.fuente_ref,'fuente_version',s.fuente_version)))
  ORDER BY s.vigente_desde DESC,s.conocido_desde DESC,s.servicio_ref COLLATE "C" ASC,s.revision DESC),'[]'::jsonb)
 INTO revisiones FROM vec_personal.servicio_reconocido_historia s
 JOIN vec_personal.relacion_servicio_historia r ON r.relacion_ref=s.relacion_ref AND r.revision=s.relacion_revision
   AND r.empleado_ref=s.empleado_ref AND r.organismo_ref=s.organismo_ref
 WHERE s.empleado_ref=empleado AND r.persona_ref=persona AND r.conocido_desde<=conocido AND s.conocido_desde<=conocido
   AND s.vigente_desde<hasta AND (s.vigente_hasta IS NULL OR desde<s.vigente_hasta);
 IF jsonb_array_length(revisiones) IS DISTINCT FROM cardinalidad THEN
  RAISE EXCEPTION 'cardinalidad de historia divergente' USING ERRCODE='55000'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz
    OR ahora<proyeccion.vigente_desde OR ahora>=proyeccion.vigente_hasta THEN
  RAISE EXCEPTION 'historia propia caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('historia',jsonb_build_object('empleado_ref',empleado,
  'corte',jsonb_build_object('efectos_desde',desde::text,'efectos_hasta',hasta::text,'conocido_en',m->>'conocido_en'),
  'cobertura','no_acreditada','revisiones',revisiones),
  'evidencia',jsonb_build_object('recibo_ref',consumo.auditoria_ref,'decision_ref',consumo.decision_ref,
    'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,'auditoria_ref',consumo.auditoria_ref,
    'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_historia_servicios_propios_empleado_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_historia_servicios_propios_empleado_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
