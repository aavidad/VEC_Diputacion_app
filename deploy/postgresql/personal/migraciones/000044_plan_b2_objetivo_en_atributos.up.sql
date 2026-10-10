\set ON_ERROR_STOP on
-- Personal44. Pasos del plan de incorporación B2 (Personal23): preparar,
-- consultar, ejecutar, confirmar y seleccionar llevan el plan o la plaza en
-- los atributos del recurso V3 y no en los ámbitos. El perfil nominal fijo
-- de Personal tiene competencia sobre el organismo y no puede enumerar planes
-- ni plazas; el objetivo sigue en la huella firmada, recurso_ref y efecto_ref
-- no cambian, y la unidad, versión y estado del expediente los sigue
-- comprobando esta función. La lectura de clases de ocupación conserva su
-- objetivo cerrado en los ámbitos.
--
-- Además, plan_incorporacion_ct_v1 y registrar_acto_plan_incorporacion_ct_v1
-- fijaban statement_timeout=30s en su definición. El núcleo V3 exige que el
-- consumo corra con statement_timeout entre 1 ms y 15 s, y el SET de la
-- función manda sobre el de la transacción: el núcleo rechazaba siempre los
-- pasos del plan y el alta/hecho que pasan por él («límites VEC-AD-3
-- ausentes»). Ambas pasan a 15 s; idle_in_transaction_session_timeout lo
-- sigue fijando la transacción de la aplicación.
-- Consenso Claude-Fable-Astra del 10/10/2026, opción B.
--
-- Sustitución en sitio con preimagen exacta de Personal23 (pg_get_functiondef
-- instalado) y CREATE OR REPLACE literal: solo cambia lo marcado «Personal44».
-- Propietario, ACL y SECURITY DEFINER se conservan. Una sola vez; sin DOWN.
-- Va con el vec-server que construye el recurso nuevo: Go y SQL se despliegan
-- juntos (Personal41 a Personal44).
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000044',0));
DO $pre$
DECLARE f regprocedure:=pg_catalog.to_regprocedure('vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'); g regprocedure;
 actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal44: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal44: PARO clave=PG actual=% esperado=180000..189999',pg_catalog.current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF f IS NULL OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'Personal44: PARO clave=preimagen actual=incompatible esperado=Personal23' USING ERRCODE='55000'; END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex');
 IF actual='635c8a84369a66c2a1c2bc899061c088e51b1fd6a44e3f9d1201c285330df068'
 THEN RAISE EXCEPTION 'Personal44: PARO clave=ya-aplicada actual=postimagen esperado=Personal23' USING ERRCODE='55000'; END IF;
 IF actual IS DISTINCT FROM 'e8d51304f21e8bc8d6a3425177e4ceae1be74b255ab50ed53d2a87f0e0d6e649'
 THEN RAISE EXCEPTION 'Personal44: PARO clave=plan_incorporacion_ct_v1 actual=distinta esperado=Personal23' USING ERRCODE='55000'; END IF;
 g:=pg_catalog.to_regprocedure('vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF g IS NULL OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=g)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=g) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(g),'UTF8')),'hex')
   IS DISTINCT FROM '5c253ef3a740dc4b23cce1c74ae43892f1138f996163e7008ae013a157755a34'
 THEN RAISE EXCEPTION 'Personal44: PARO clave=registrar_acto_plan actual=distinta esperado=Personal23' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_personal_propietario;
CREATE OR REPLACE FUNCTION vec_personal.plan_incorporacion_ct_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; a jsonb; datos jsonb; seleccion jsonb; op text; selector text; org text;
 datos_raw text; actor_raw text; canon text; sha text; recurso text; recurso_sha text; clave uuid;
 p vec_personal.plan_incorporacion_ct%ROWTYPE; v_consumo record; catalogo vec_personal.clases_ocupacion_plan_ct_catalogo%ROWTYPE; est jsonb; evidencia jsonb;
 acceso text; k text; n bigint; emp text; modo text; planref text; reciboref text; ka uuid; ko uuid;
 uso text; reserva text; confirmacion text; ph text; dec_hasta timestamptz; cap_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 32768
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal23: plan denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
  a:=m->'actor'; datos:=m->'datos'; op:=m->>'operacion'; selector:=m->>'plan_ref'; org:=m->>'organismo_ref';
  dec_hasta:=(d->>'valida_hasta')::timestamptz; cap_hasta:=(c->>'expira_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal23: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM (CASE WHEN op='seleccionar' THEN ARRAY['actor','datos','esquema','negocio_sha256','operacion','organismo_ref','plan_ref','seleccion'] ELSE ARRAY['actor','datos','esquema','negocio_sha256','operacion','organismo_ref','plan_ref'] END)
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.plan-incorporacion-ct.v1'
    OR op IS NULL OR op NOT IN ('preparar','consultar','ejecutar','confirmar','seleccionar','clases_ocupacion')
    OR org IS NULL OR org !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR selector IS NULL OR jsonb_typeof(a) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(a) ORDER BY 1) IS DISTINCT FROM ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','perfil_ref','perfil_version','persona_ref','persona_version']
    OR a->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR a->>'persona_ref' IS DISTINCT FROM x->>'persona_ref'
    OR a->>'actor_ref' IS DISTINCT FROM a->>'persona_ref'
    OR a->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR a->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR a->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR a->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR a->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR a->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR a->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR dec_hasta IS NULL OR cap_hasta IS NULL OR NOT isfinite(dec_hasta) OR NOT isfinite(cap_hasta)
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta' THEN
  RAISE EXCEPTION 'Personal23: contexto divergente' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','perfil_version','persona_version'] LOOP
  IF jsonb_typeof(a->k) IS DISTINCT FROM 'number' OR a->>k !~ '^[1-9][0-9]{0,19}$' THEN
   RAISE EXCEPTION 'Personal23: versión de actor inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF a->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR a->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR a->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR a->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$' THEN
  RAISE EXCEPTION 'Personal23: actor inválido' USING ERRCODE='22023'; END IF;
 IF op='preparar' THEN
  IF selector !~ '^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$'
     OR m->>'negocio_sha256' IS NULL OR m->>'negocio_sha256' !~ '^[0-9a-f]{64}$'
     OR jsonb_typeof(datos) IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION 'Personal23: preparación inválida' USING ERRCODE='22023'; END IF;
  datos_raw:=substring(p_material FROM ',"datos":(.*),"negocio_sha256":"');
  PERFORM vec_personal.validar_datos_plan_ct_interno(datos,datos_raw);
  IF datos->>'idempotencia_ref' IS DISTINCT FROM selector OR datos->>'organismo_ref' IS DISTINCT FROM org
     OR encode(sha256(convert_to(datos_raw,'UTF8')),'hex') IS DISTINCT FROM m->>'negocio_sha256' THEN
   RAISE EXCEPTION 'Personal23: negocio divergente' USING ERRCODE='22023'; END IF;
 ELSIF op='clases_ocupacion' THEN
  IF selector IS DISTINCT FROM org OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'Personal23: selector de catálogo inválido' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 ELSIF op='seleccionar' THEN
  seleccion:=m->'seleccion';
  IF selector !~ '^plaza:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' OR jsonb_typeof(seleccion) IS DISTINCT FROM 'object'
     OR ARRAY(SELECT jsonb_object_keys(seleccion) ORDER BY 1) IS DISTINCT FROM ARRAY['desde','plaza_ref','puesto_ref']
     OR seleccion->>'plaza_ref' IS DISTINCT FROM selector
     OR seleccion->>'puesto_ref' IS NULL OR seleccion->>'puesto_ref' !~ '^puesto:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$'
     OR seleccion->>'desde' IS NULL OR seleccion->>'desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
   RAISE EXCEPTION 'Personal23: selección inválida' USING ERRCODE='22023'; END IF;
  IF NOT isfinite((seleccion->>'desde')::date) THEN RAISE EXCEPTION 'Personal23: fecha inválida' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 ELSE
  IF selector !~ '^perplan_[0-9a-f]{32}$' OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'Personal23: selector inválido' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 END IF;
 actor_raw:='{"actor_ref":'||to_jsonb(a->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(a->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(a->>'contexto_version')||',"cuenta_ref":'||to_jsonb(a->>'cuenta_ref')::text||
  ',"cuenta_version":'||(a->>'cuenta_version')||',"perfil_ref":'||to_jsonb(a->>'perfil_ref')::text||
  ',"perfil_version":'||(a->>'perfil_version')||',"persona_ref":'||to_jsonb(a->>'persona_ref')::text||
  ',"persona_version":'||(a->>'persona_version')||'}';
 canon:='{"esquema":"vec.personal.plan-incorporacion-ct.v1","operacion":'||to_jsonb(op)::text||
  ',"plan_ref":'||to_jsonb(selector)::text||',"organismo_ref":'||to_jsonb(org)::text||',"datos":'||datos_raw||
  ',"negocio_sha256":'||to_jsonb(m->>'negocio_sha256')::text||',"actor":'||actor_raw||'}';
 IF op='seleccionar' THEN
  canon:=left(canon,length(canon)-1)||',"seleccion":{"plaza_ref":'||to_jsonb(seleccion->>'plaza_ref')::text||',"puesto_ref":'||to_jsonb(seleccion->>'puesto_ref')::text||',"desde":'||to_jsonb(seleccion->>'desde')::text||'}}';
 END IF;
 IF p_material IS DISTINCT FROM canon THEN RAISE EXCEPTION 'Personal23: material no canónico' USING ERRCODE='22023'; END IF;
 sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 -- Personal44: ámbito organismo y plan o plaza en los atributos firmados;
 -- la lectura de clases conserva su objetivo cerrado en los ámbitos.
 recurso:=CASE WHEN op='clases_ocupacion' THEN
  '{"ambitos":{"objetivo_ref":'||to_jsonb(selector)::text||',"organismo_ref":'||to_jsonb(org)::text||
  '},"atributos":{"material_sha256":"'||sha||'","operacion":'||to_jsonb(op)::text||'}}'
 ELSE '{"ambitos":{"organismo_ref":'||to_jsonb(org)::text||
  '},"atributos":{"material_sha256":"'||sha||'","objetivo_ref":'||to_jsonb(selector)::text||
  ',"operacion":'||to_jsonb(op)::text||'}}' END;
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM 'personal.plan_incorporacion_ct.'||op
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.plan_incorporacion_ct.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM selector OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'plan_incorporacion_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_incorporacion_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM selector OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN op='seleccionar' THEN '["evidencia","seleccion"]'::jsonb WHEN op='clases_ocupacion' THEN '["catalogo","evidencia"]'::jsonb ELSE '["ejecucion_huella_sha256","ejecucion_recibo_ref","estado","evidencia","plan","recibo_alta_relacion","recibo_ocupacion"]'::jsonb END) THEN
  RAISE EXCEPTION 'Personal23: permiso nominal divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM selector
    OR v_consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
    OR v_consumo.consumo_huella_sha256 IS NULL OR v_consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_consumo.consumida_en>=cap_hasta OR v_consumo.consumida_en>=dec_hasta THEN
  RAISE EXCEPTION 'Personal23: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:plan-incorporacion-ct:'||selector,0));
 IF op='clases_ocupacion' THEN
  SELECT * INTO catalogo FROM vec_personal.clases_ocupacion_plan_ct_catalogo
  WHERE ref='personal:incorporacion_ct:clases_ocupacion' ORDER BY version DESC LIMIT 1;
  IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: catálogo no disponible' USING ERRCODE='55000'; END IF;
  est:=jsonb_build_object('catalogo',jsonb_build_object('ref',catalogo.ref,'version',catalogo.version,
    'huella_sha256',catalogo.huella_sha256,'opciones',(SELECT jsonb_agg(jsonb_build_object(
      'valor',o->>'valor','texto_clave',o->>'texto_clave') ORDER BY ord)
     FROM jsonb_array_elements(catalogo.datos->'opciones') WITH ORDINALITY x(o,ord))));
 ELSIF op='seleccionar' THEN
  est:=jsonb_build_object('seleccion',vec_personal.seleccion_plan_ct_interna(org,seleccion));
 ELSIF op='preparar' THEN
  clave:=selector::uuid;
  SELECT * INTO p FROM vec_personal.plan_incorporacion_ct WHERE idempotencia_ref=clave;
  IF FOUND THEN
   IF p.negocio_sha256 IS DISTINCT FROM m->>'negocio_sha256' OR p.datos_canon IS DISTINCT FROM datos_raw
      OR p.organismo_ref IS DISTINCT FROM org THEN
    RAISE EXCEPTION 'Personal23: idempotencia divergente' USING ERRCODE='23505'; END IF;
  ELSE
   seleccion:=vec_personal.seleccion_plan_ct_interna(org,jsonb_build_object(
    'plaza_ref',datos->>'plaza_ref','puesto_ref',datos->>'puesto_ref','desde',datos->>'desde'));
   FOREACH k IN ARRAY ARRAY['version_plantilla_ref','version_rpt_ref','revision_plaza','revision_puesto',
       'fuente_organizacion_ref','fuente_organizacion_huella_sha256','unidad_ref'] LOOP
    IF seleccion->>k IS DISTINCT FROM datos->>k THEN
     RAISE EXCEPTION 'Personal23: fuente estructural del plan divergente' USING ERRCODE='42501'; END IF;
   END LOOP;
   -- La clase debe figurar en la versión vigente antes de reservar claves o actos.
   SELECT * INTO catalogo FROM vec_personal.clases_ocupacion_plan_ct_catalogo
    WHERE ref='personal:incorporacion_ct:clases_ocupacion' ORDER BY version DESC LIMIT 1;
   IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: catálogo no disponible' USING ERRCODE='55000'; END IF;
   IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(catalogo.datos->'opciones') o
      WHERE o->>'valor'=datos->>'clase_ocupacion') THEN
    RAISE EXCEPTION 'Personal23: clase no publicada' USING ERRCODE='22023'; END IF;
   -- La relación histórica evita una segunda alta tras vencer una proyección.
   -- B2 conserva su control de vigencia al ejecutar la nueva relación.
   PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(datos->>'persona_ref');
   SELECT count(DISTINCT z.empleado_ref),min(z.empleado_ref) INTO n,emp FROM (
     SELECT empleado_ref FROM vec_personal.relacion_servicio_historia WHERE persona_ref=datos->>'persona_ref'
     UNION SELECT empleado_ref FROM vec_personal.proyeccion_empleado_persona_historia WHERE persona_ref=datos->>'persona_ref') z;
   IF n>1 THEN RAISE EXCEPTION 'Personal23: empleado histórico ambiguo' USING ERRCODE='55000'; END IF;
   modo:=CASE WHEN n=0 THEN 'alta_empleado' ELSE 'nueva_relacion' END; emp:=coalesce(emp,'');
   planref:='perplan_'||replace(gen_random_uuid()::text,'-','');
   reciboref:='perplanrec_'||replace(gen_random_uuid()::text,'-','');
   ka:=gen_random_uuid(); ko:=gen_random_uuid();
   uso:='uso:'||gen_random_uuid()::text; reserva:='reserva:'||gen_random_uuid()::text;
   confirmacion:='confirmacion:'||gen_random_uuid()::text;
   ph:=encode(sha256(convert_to((m->>'negocio_sha256')||'|'||planref||'|'||reciboref||'|'||modo||'|'||emp||'|'||ka::text||'|'||ko::text||'|'||uso||'|'||reserva||'|'||confirmacion||'|'||catalogo.ref||'|'||catalogo.version::text||'|'||catalogo.huella_sha256,'UTF8')),'hex');
   INSERT INTO vec_personal.plan_incorporacion_ct VALUES(clave,planref,reciboref,org,datos_raw,datos,
    m->>'negocio_sha256',modo,emp,ka,ko,uso,reserva,confirmacion,catalogo.ref,catalogo.version,catalogo.huella_sha256,ph,v_consumo.decision_ref,
    v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en) RETURNING * INTO p;
  END IF;
 ELSE
  SELECT * INTO p FROM vec_personal.plan_incorporacion_ct WHERE plan_ref=selector AND organismo_ref=org;
  IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: plan no encontrado' USING ERRCODE='P7404'; END IF;
 END IF;
 IF op NOT IN ('seleccionar','clases_ocupacion') THEN est:=vec_personal.estado_plan_ct_interno(p); END IF;
 IF op='ejecutar' AND est->>'estado' IN ('preparado','relacion_registrada') THEN
  BEGIN
   seleccion:=vec_personal.seleccion_plan_ct_interna(org,jsonb_build_object(
    'plaza_ref',p.datos->>'plaza_ref','puesto_ref',p.datos->>'puesto_ref','desde',p.datos->>'desde'));
  -- El helper no consume permisos: estos errores describen cambios de la
  -- estructura reservada. La autorización actual ya se consumió antes.
  EXCEPTION WHEN SQLSTATE 'P7404' OR SQLSTATE '42501' THEN
   RAISE EXCEPTION 'Personal23: estructura reservada cambió' USING ERRCODE='23505';
  END;
  FOREACH k IN ARRAY ARRAY['version_plantilla_ref','version_rpt_ref','revision_plaza','revision_puesto',
      'fuente_organizacion_ref','fuente_organizacion_huella_sha256','unidad_ref'] LOOP
   IF seleccion->>k IS DISTINCT FROM p.datos->>k THEN
    RAISE EXCEPTION 'Personal23: estructura cambió antes de ejecutar' USING ERRCODE='23505'; END IF;
  END LOOP;
 END IF;
 IF op='confirmar' AND est->>'estado'<>'ejecutado' THEN
  IF est->>'estado' IS DISTINCT FROM 'ocupacion_registrada' THEN
   RAISE EXCEPTION 'Personal23: efectos pendientes' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_personal.ejecucion_plan_incorporacion_ct VALUES(p.plan_ref,
    'perplaneje_'||replace(gen_random_uuid()::text,'-',''),
    encode(sha256(convert_to(p.huella_sha256||'|'||(est->'recibo_alta_relacion'->>'recibo_ref')||'|'||(est->'recibo_ocupacion'->>'recibo_ref'),'UTF8')),'hex'),
    est->'recibo_alta_relacion'->>'recibo_ref',est->'recibo_ocupacion'->>'recibo_ref',
    v_consumo.decision_ref,v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en);
  est:=vec_personal.estado_plan_ct_interno(p);
 END IF;
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'Personal23: autorización caducada' USING ERRCODE='42501'; END IF;
 acceso:='perplanacc_'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_personal.acceso_plan_incorporacion_ct VALUES(acceso,op,selector,a->>'actor_ref',sha,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en);
 evidencia:=jsonb_build_object('recibo_ref',acceso,'decision_ref',v_consumo.decision_ref,
  'efecto_ref',selector,'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
  'auditoria_ref',v_consumo.auditoria_ref,'consultada_en',v_consumo.consumida_en);
 RETURN est||jsonb_build_object('evidencia',evidencia);
END $function$;
ALTER FUNCTION vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET statement_timeout TO '15s';
RESET ROLE;

DO $post$
DECLARE f regprocedure:='vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; g regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM true
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
   IS DISTINCT FROM '635c8a84369a66c2a1c2bc899061c088e51b1fd6a44e3f9d1201c285330df068'
 THEN RAISE EXCEPTION 'Personal44: PARO clave=postimagen actual=divergente esperado=plan_incorporacion_ct_v1' USING ERRCODE='55000'; END IF;
 g:='vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=g)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=g) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=g)
 OR (SELECT proconfig::text FROM pg_catalog.pg_proc WHERE oid=g) IS DISTINCT FROM
    '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s,statement_timeout=15s}'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(g),'UTF8')),'hex')
   IS DISTINCT FROM '7ce49537c80a3b9821b5a9c4d8322d51aad722e7966250b8e81bca4d94949455'
 THEN RAISE EXCEPTION 'Personal44: PARO clave=postimagen-registrar_acto_plan actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
