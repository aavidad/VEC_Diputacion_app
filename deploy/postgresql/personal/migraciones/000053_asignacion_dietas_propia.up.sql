\set ON_ERROR_STOP on
-- Personal 53: lectura propia de la asignación vigente para preparar Dietas.
-- La lectura competente de Personal 12 y sus recibos históricos siguen intactos.
-- Depende del consumidor nominal nuevo de autorización atestada V3.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000053:asignacion-propia',0));

DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_personal_propietario'
    OR to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_personal.ejecutar_asignacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_personal.consultar_asignacion_dietas_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_propia_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_personal.relacion_empleado_dietas') IS NULL
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL
    OR to_regclass('vec_personal.recibo_asignacion_dietas') IS NULL
    OR to_regclass('vec_personal.evidencia_asignacion_dietas') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario',
      to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_propia_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
    OR NOT has_schema_privilege('vec_personal_d7_ejecutor','vec_personal','USAGE')
 THEN RAISE EXCEPTION 'Personal53: dependencia o entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=
   to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
     '323b88bab32274f14fe0c008371cdc335d130e16d1d1402ee8202823790e34ff'
    OR p.proowner IS DISTINCT FROM 'vec_personal_propietario'::regrole OR p.prosecdef IS NOT TRUE
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM
      '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_d7_ejecutor=X/vec_personal_propietario}'
 THEN RAISE EXCEPTION 'Personal53: preimagen de lectura antigua incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=
   to_regprocedure('vec_personal.ejecutar_asignacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM
     '5aec425f8bbe9ae2dd5ce30911de2b047933bc8f2d7d669e0096bd2482b983b0'
    OR p.proowner IS DISTINCT FROM 'vec_personal_propietario'::regrole OR p.prosecdef IS NOT TRUE
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s']::text[]
    OR p.proacl::text IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario}'
 THEN RAISE EXCEPTION 'Personal53: preimagen del núcleo antiguo incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc f WHERE f.oid=
      to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_propia_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND f.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND f.prosecdef
       AND f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[])
    OR (SELECT count(*) FROM pg_class c WHERE c.oid IN
      (to_regclass('vec_personal.relacion_empleado_dietas'),
       to_regclass('vec_personal.asignacion_dietas'),
       to_regclass('vec_personal.recibo_asignacion_dietas'),
       to_regclass('vec_personal.evidencia_asignacion_dietas'))
       AND c.relowner='vec_personal_propietario'::regrole
       AND c.relkind='r' AND c.relrowsecurity AND c.relforcerowsecurity)<>4
    OR NOT EXISTS (SELECT 1 FROM pg_policy q
      WHERE q.polrelid=to_regclass('vec_personal.asignacion_dietas')
        AND q.polname='persona_contextual' AND q.polroles=ARRAY['vec_personal_propietario'::regrole::oid]
        AND encode(sha256(convert_to(coalesce(pg_get_expr(q.polqual,q.polrelid),'')||'|'||
          coalesce(pg_get_expr(q.polwithcheck,q.polrelid),''),'UTF8')),'hex')=
          '4cf19f84271002d23542013dd457124ac2d005af22bf533a9d0296f1841ea2e9')
 THEN RAISE EXCEPTION 'Personal53: RLS o consumidor V3 incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_personal.consultar_asignacion_dietas_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $lectura$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; cap jsonb; d jsonb; ctx jsonb;
 r vec_personal.relacion_empleado_dietas%ROWTYPE;
 a vec_personal.asignacion_dietas%ROWTYPE;
 z record; actor_empleado text; sujeto_persona text; sujeto_empleado text;
 relacion text; unidad text; fecha date; material_sha text; recurso_sha text;
 amb text; atr text; v_recibo_ref text; ahora timestamptz(6);
 campos jsonb:='["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_d7_ejecutor','MEMBER')
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members x
       WHERE x.member=session_user::regrole AND x.roleid='vec_personal_d7_ejecutor'::regrole
         AND NOT x.admin_option AND x.inherit_option AND NOT x.set_option)
    OR (SELECT count(*) FROM pg_auth_members x WHERE x.member=session_user::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_roles v WHERE left(v.rolname,4)='vec_'
       AND v.rolname<>session_user AND v.rolname<>'vec_personal_d7_ejecutor'
       AND pg_has_role(session_user,v.oid,'MEMBER'))
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_setting('idle_in_transaction_session_timeout')='0'
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
    OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'lectura propia Personal incompatible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de lectura propia Personal inválido' USING ERRCODE='22023';
 END;
 i:=m->'identidad'; sujeto_persona:=m->>'persona_ref'; sujeto_empleado:=m->>'empleado_ref';
 relacion:=m->>'relacion_ref'; unidad:=m->>'unidad_ref';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
       ARRAY['administrativo_persona_ref','centro_ref','clave_idempotencia',
        'empleado_ref','esquema','fecha_referencia','grupo_dieta','identidad',
        'motivo_revision','operacion','persona_ref','procedencia_acto_ref',
        'relacion_ref','responsable_persona_ref','unidad_ref','version_esperada','vigente_desde']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.asignacion-dietas-propia.v1'
    OR m->>'operacion' IS DISTINCT FROM 'consultar_propia'
    OR relacion !~ '^rel_[A-Za-z0-9_-]{22,128}$'
    OR sujeto_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR sujeto_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR unidad IS NULL OR length(unidad) NOT BETWEEN 1 AND 256
    OR unidad IS DISTINCT FROM btrim(unidad) OR unidad ~ '[[:cntrl:]]'
    OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref',
        'cuenta_version','empleado_ref','perfil_ref','perfil_version',
        'persona_ref','persona_version']
    OR m->'clave_idempotencia' IS DISTINCT FROM '""'::jsonb
    OR m->'version_esperada' IS DISTINCT FROM '0'::jsonb
    OR m->'centro_ref' IS DISTINCT FROM '""'::jsonb
    OR m->'administrativo_persona_ref' IS DISTINCT FROM '""'::jsonb
    OR m->'responsable_persona_ref' IS DISTINCT FROM '""'::jsonb
    OR m->'grupo_dieta' IS DISTINCT FROM '0'::jsonb
    OR m->'vigente_desde' IS DISTINCT FROM '""'::jsonb
    OR m->'motivo_revision' IS DISTINCT FROM '""'::jsonb
    OR m->'procedencia_acto_ref' IS DISTINCT FROM '""'::jsonb
 THEN RAISE EXCEPTION 'solicitud propia Personal no canónica' USING ERRCODE='22023'; END IF;
 fecha:=(m->>'fecha_referencia')::date;
 IF fecha IS DISTINCT FROM (statement_timestamp() AT TIME ZONE 'UTC')::date
 THEN RAISE EXCEPTION 'consulta propia Personal no vigente' USING ERRCODE='P7201'; END IF;
 SELECT e.valor->>'referencia' INTO actor_empleado
 FROM jsonb_array_elements(coalesce(ctx->'vinculos','[]'::jsonb)) e(valor)
 WHERE jsonb_typeof(e.valor)='object'
   AND e.valor->>'tipo'='empleado' AND e.valor->>'estado'='activo';
 IF jsonb_typeof(ctx) IS DISTINCT FROM 'object'
    OR ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR ctx->>'estado' IS DISTINCT FROM 'activo'
    OR ctx->>'principal_ref' IS DISTINCT FROM ctx->>'persona_ref'
    OR (SELECT count(*) FROM jsonb_array_elements(coalesce(ctx->'vinculos','[]'::jsonb)) e(valor)
       WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
         AND e.valor->>'estado'='activo')<>1
    OR sujeto_persona IS DISTINCT FROM ctx->>'persona_ref'
    OR sujeto_empleado IS DISTINCT FROM actor_empleado
    OR i->>'actor_ref' IS DISTINCT FROM ctx->>'principal_ref'
    OR i->>'persona_ref' IS DISTINCT FROM sujeto_persona
    OR i->>'empleado_ref' IS DISTINCT FROM sujeto_empleado
    OR i->>'perfil_ref' IS DISTINCT FROM ctx->>'perfil_activo_ref'
    OR i->>'contexto_actor_ref' IS DISTINCT FROM ctx->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM ctx->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM ctx->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM ctx->>'cuenta_version'
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR ctx->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR ctx->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 THEN RAISE EXCEPTION 'identidad propia Personal incompatible' USING ERRCODE='42501'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 amb:='{"empleado_ref":'||to_jsonb(sujeto_empleado)::text||
      ',"persona_ref":'||to_jsonb(sujeto_persona)::text||'}';
 atr:='{"fecha_referencia":'||to_jsonb(fecha::text)::text||
      ',"material_sha256":'||to_jsonb(material_sha)::text||
      ',"operacion":"consultar_propia"'||
      ',"relacion_ref":'||to_jsonb(relacion)::text||
      ',"unidad_ref":'||to_jsonb(unidad)::text||'}';
 recurso_sha:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 IF cap->>'operacion' IS DISTINCT FROM 'personal.asignacion_dietas.propia.consultar'
    OR cap->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.asignacion_dietas.propia.consultar.v1'
    OR cap->>'efecto_ref' IS DISTINCT FROM relacion
    OR cap->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR cap->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex')
    OR cap->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR cap->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'accion' IS DISTINCT FROM 'personal.asignacion_dietas.propia.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'asignacion_dietas_propia'
    OR d->>'finalidad' IS DISTINCT FROM 'preparar_borrador_dietas'
    OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR d->>'recurso_ref' IS DISTINCT FROM relacion
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'autorización propia Personal incompatible' USING ERRCODE='42501'; END IF;
 -- La fachada común consume la decisión V3 y deja su asiento antes de leer
 -- relación y asignación. Un fallo posterior revierte también el consumo.
 SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_propia_asignacion_dietas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF z.consumo_nuevo IS NOT TRUE OR z.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR z.efecto_ref IS DISTINCT FROM relacion OR z.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
    OR z.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' OR z.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'consumo propio Personal incompatible' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',sujeto_persona,true);
 SELECT * INTO r FROM vec_personal.relacion_empleado_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.empleado_ref=sujeto_empleado AND v.unidad_ref=unidad
   AND v.estado='activa' AND v.desde<=fecha AND (v.hasta IS NULL OR fecha<v.hasta)
 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'relación propia Personal no disponible' USING ERRCODE='P7201'; END IF;
 SELECT * INTO a FROM vec_personal.asignacion_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.unidad_ref=unidad AND v.vigente_desde<=fecha
 ORDER BY v.version DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'asignación propia Personal no disponible' USING ERRCODE='P7201'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 v_recibo_ref:='rad_'||substr(encode(sha256(convert_to(
   z.consumo_huella_sha256||'|'||material_sha||'|'||ahora::text,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.recibo_asignacion_dietas
 (referencia,operacion,clave_idempotencia,huella_semantica_sha256,asignacion_ref,
  relacion_ref,persona_ref,unidad_ref,version,decision_ref,efecto_ref,
  consumo_huella_sha256,auditoria_ad3_ref,registrada_en)
 VALUES(v_recibo_ref,'consulta',NULL,NULL,a.asignacion_ref,relacion,sujeto_persona,
  unidad,a.version,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,z.auditoria_ref,ahora);
 INSERT INTO vec_personal.evidencia_asignacion_dietas(evidencia_ref,recibo_ref,tipo,registrada_en)
 VALUES('ead_'||substr(encode(sha256(convert_to(v_recibo_ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32),
  v_recibo_ref,'consulta_asignacion_dietas_autorizada',ahora);
 RETURN QUERY SELECT v_recibo_ref,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,
  z.auditoria_ref,ahora,'consultada'::text,a.asignacion_ref,a.relacion_ref,
  a.persona_ref,a.unidad_ref,a.centro_ref,a.administrativo_persona_ref,
  a.responsable_persona_ref,a.grupo_dieta,a.vigente_desde,a.version;
END $lectura$;
ALTER FUNCTION vec_personal.consultar_asignacion_dietas_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 OWNER TO vec_personal_propietario;
REVOKE ALL ON FUNCTION vec_personal.consultar_asignacion_dietas_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_asignacion_dietas_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;

DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=
  'vec_personal.consultar_asignacion_dietas_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_personal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
    OR NOT EXISTS (SELECT 1 FROM pg_language l WHERE l.oid=p.prolang AND l.lanname='plpgsql')
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
       'be2a05800acbd869c28fa74b0cc12a357b05e4dcd55fd74b9165a5dadb90e98d'
    OR pg_get_function_result(p.oid) IS DISTINCT FROM
       pg_get_function_result(to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
      'TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
    OR p.proacl::text IS DISTINCT FROM
      '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_d7_ejecutor=X/vec_personal_propietario}'
    OR has_function_privilege('vec_personal_ejecutor',p.oid,'EXECUTE')
    OR has_function_privilege('vec_dietas_ejecutor',p.oid,'EXECUTE')
    OR has_function_privilege('vec_dietas_propietario',p.oid,'EXECUTE')
 THEN RAISE EXCEPTION 'Personal53: postimagen o ACL incompatibles' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
