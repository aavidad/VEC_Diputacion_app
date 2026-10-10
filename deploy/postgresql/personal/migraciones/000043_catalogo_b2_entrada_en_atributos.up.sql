\set ON_ERROR_STOP on
-- Personal43. Publicar y retirar entradas del catálogo de registro de empleado
-- (Personal20): la entrada (organismo:tipo:ref:versión) pasa de los ámbitos a
-- los atributos del recurso V3. El perfil nominal de gobierno del catálogo
-- tiene competencia sobre el organismo y no puede enumerar entradas; la
-- entrada sigue en la huella firmada y recurso_ref/efecto_ref no cambian.
-- La consulta del catálogo (consultar_catalogo_empleado_rrhh_v1) conserva su
-- objetivo cerrado en los ámbitos y no se toca.
-- Consenso Claude-Fable-Astra del 10/10/2026, opción B.
--
-- Sustitución en sitio con preimagen exacta de Personal20 (pg_get_functiondef
-- instalado) y CREATE OR REPLACE literal: solo cambia lo marcado «Personal43».
-- Propietario, ACL y SECURITY DEFINER se conservan. Una sola vez; sin DOWN.
-- Va con el vec-server que construye el recurso nuevo: Go y SQL se despliegan
-- juntos (Personal41 a Personal44).
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000043',0));
DO $pre$
DECLARE f regprocedure:=pg_catalog.to_regprocedure('vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal43: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal43: PARO clave=PG actual=% esperado=180000..189999',pg_catalog.current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF f IS NULL OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'Personal43: PARO clave=preimagen actual=incompatible esperado=Personal20' USING ERRCODE='55000'; END IF;
 actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex');
 IF actual='67c1721c5e122d97d086d46447026612c6dffe821dd580a329fa10f7a55466f0'
 THEN RAISE EXCEPTION 'Personal43: PARO clave=ya-aplicada actual=postimagen esperado=Personal20' USING ERRCODE='55000'; END IF;
 IF actual IS DISTINCT FROM 'f97187e919d3c148685e2638915208af05d72adfb00cf9ac1ce05902c19fba89'
 THEN RAISE EXCEPTION 'Personal43: PARO clave=registrar_entrada_catalogo_empleado_rrhh_v1 actual=distinta esperado=Personal20' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_personal_propietario;
CREATE OR REPLACE FUNCTION vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; v record; previo record; anterior record;
 accion text; audiencia text; efecto text; recurso text; recurso_sha text; material_sha text;
 clave uuid; fecha_desde date; fecha_hasta date; ver integer; rev integer; estado text; huella_esperada text; acto_interno text;
 cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' OR current_user<>'vec_personal_propietario'
    OR session_user=current_user OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'acto de catálogo denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
  clave:=(m->>'idempotencia_ref')::uuid;
  fecha_desde:=(m->>'vigente_desde')::date;
  fecha_hasta:=NULLIF(m->>'vigente_hasta','')::date;
  ver:=(m->>'version')::integer; rev:=(m->>'revision')::integer;
  cap_desde:=(c->>'emitida_en')::timestamptz;
  cap_hasta:=(c->>'expira_en')::timestamptz;
  dec_hasta:=(d->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de catálogo inválido' USING ERRCODE='22023'; END;
 IF m ? 'acto_ref' THEN
  RAISE EXCEPTION 'procedencia de catálogo externa denegada' USING ERRCODE='23514'; END IF;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','denominacion','esquema','huella_sha256','idempotencia_ref','operacion','organismo_ref','ref','revision',
   'tipo','version','vigente_desde','vigente_hasta']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.catalogo-registro-empleado.v1'
    OR m->>'operacion' NOT IN ('publicar','retirar')
    OR m->>'tipo' NOT IN ('regimen','modalidad','situacion','clase_servicio')
    OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR m->>'ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'denominacion' IS NULL OR octet_length(m->>'denominacion') NOT BETWEEN 1 AND 256
    OR m->>'denominacion'<>btrim(m->>'denominacion') OR m->>'denominacion' ~ '[[:cntrl:]]'
    OR ver<1 OR rev<1 OR fecha_desde IS NULL OR NOT isfinite(fecha_desde)
    OR (fecha_hasta IS NOT NULL AND (NOT isfinite(fecha_hasta) OR fecha_hasta<=fecha_desde))
    OR m->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR x->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
    OR cap_desde IS NULL OR cap_hasta IS NULL OR dec_hasta IS NULL
    OR NOT isfinite(cap_desde) OR NOT isfinite(cap_hasta) OR NOT isfinite(dec_hasta)
    OR cap_hasta<=cap_desde OR dec_hasta<=cap_desde
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'
    OR clock_timestamp()<cap_desde OR clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta
    OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'catálogo o identidad divergente' USING ERRCODE='42501'; END IF;
 estado:=CASE m->>'operacion' WHEN 'publicar' THEN 'publicada' ELSE 'retirada' END;
 accion:='personal.registro_empleado.catalogo.'||(m->>'operacion');
 audiencia:='vec_personal.registro_empleado.catalogo.'||(m->>'operacion')||'.v1';
 efecto:=(m->>'organismo_ref')||':'||(m->>'tipo')||':'||(m->>'ref')||':'||ver;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 -- Personal43: ámbito organismo; la entrada va en los atributos firmados.
 recurso:='{"ambitos":{"organismo_ref":'||to_jsonb(m->>'organismo_ref')::text||'},"atributos":{"material_sha256":"'||material_sha||'","objetivo_ref":'||to_jsonb(efecto)::text||',"operacion":'||to_jsonb(m->>'operacion')::text||'}}';
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM efecto OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'accion' IS DISTINCT FROM accion OR d->>'recurso_ref' IS DISTINCT FROM efecto
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'tipo_recurso' IS DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
    OR d->>'finalidad' IS DISTINCT FROM 'gobernar_catalogo_empleado'
    OR d->'campos_permitidos' IS DISTINCT FROM '["entrada","recibo"]'::jsonb THEN
  RAISE EXCEPTION 'acto de catálogo sin permiso nominal' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS NULL OR v.decision_ref=''
    OR v.efecto_ref IS DISTINCT FROM efecto
    OR v.huella_efecto_sha256 IS DISTINCT FROM recurso_sha THEN
  RAISE EXCEPTION 'consumo de catálogo divergente' USING ERRCODE='42501'; END IF;
 -- Este acto_ref identifica únicamente una configuración interna consumida por
 -- V3; no acredita documento, firma B5, fuente normativa ni eficacia jurídica.
 -- decision_ref y auditoria_ref originales permanecen en la historia.
 acto_interno:='personal:catalogo:v3:'||encode(sha256(convert_to(v.decision_ref,'UTF8')),'hex');
 IF v.consumida_en<cap_desde OR v.consumida_en>=cap_hasta OR v.consumida_en>=dec_hasta THEN
  RAISE EXCEPTION 'consumo de catálogo fuera de vigencia' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:idempotencia:'||clave::text,0));
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 SELECT * INTO previo FROM vec_personal.entrada_catalogo_registro_empleado_historia WHERE idempotencia_ref=clave;
 IF FOUND THEN
  IF previo.organismo_ref IS DISTINCT FROM m->>'organismo_ref' OR previo.tipo IS DISTINCT FROM m->>'tipo'
     OR previo.ref IS DISTINCT FROM m->>'ref' OR previo.version IS DISTINCT FROM ver
     OR previo.revision IS DISTINCT FROM rev OR previo.estado IS DISTINCT FROM estado
     OR previo.actor_ref IS DISTINCT FROM m->>'actor_ref'
     OR previo.huella_sha256 IS DISTINCT FROM m->>'huella_sha256'
     OR previo.denominacion IS DISTINCT FROM m->>'denominacion'
     OR previo.vigente_desde IS DISTINCT FROM fecha_desde
     OR previo.vigente_hasta IS DISTINCT FROM fecha_hasta THEN
   RAISE EXCEPTION 'idempotencia de catálogo divergente' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('entrada',jsonb_build_object('organismo_ref',previo.organismo_ref,'tipo',previo.tipo,
   'ref',previo.ref,'version',previo.version,'revision',previo.revision,'estado',previo.estado,
   'denominacion',previo.denominacion,'huella_sha256',previo.huella_sha256,
   'vigente_desde',previo.vigente_desde,'vigente_hasta',previo.vigente_hasta),
   'recibo',jsonb_build_object('decision_ref',previo.decision_ref,'auditoria_ref',previo.auditoria_ref,
    'consumo_huella_sha256',previo.consumo_huella_sha256,'registrado_en',previo.registrado_en),
   'acceso_actual',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
    'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en,
   'estado_replay','replay'));
 END IF;
 -- El trigger de historia usa esta misma clave. Adquirirla antes de la última
 -- comprobación temporal impide que la espera del INSERT cruce la caducidad.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:'||
  (m->>'organismo_ref')||':'||(m->>'tipo')||':'||(m->>'ref'),0));
 IF estado='publicada' THEN
  huella_esperada:=encode(sha256(convert_to(concat_ws(E'\n',
   'vec.personal.catalogo-registro-empleado.entrada.v1',m->>'organismo_ref',m->>'tipo',m->>'ref',
   ver::text,rev::text,m->>'denominacion',fecha_desde::text,coalesce(fecha_hasta::text,'')),
   'UTF8')),'hex');
  IF m->>'huella_sha256' IS DISTINCT FROM huella_esperada THEN
   RAISE EXCEPTION 'huella de entrada divergente' USING ERRCODE='23514'; END IF;
 ELSE
  SELECT * INTO anterior FROM vec_personal.entrada_catalogo_registro_empleado_historia
   WHERE organismo_ref=m->>'organismo_ref' AND tipo=m->>'tipo' AND ref=m->>'ref'
     AND version=ver ORDER BY revision DESC LIMIT 1;
  IF NOT FOUND OR anterior.estado<>'publicada' OR rev<>anterior.revision+1
     OR m->>'huella_sha256' IS DISTINCT FROM anterior.huella_sha256
     OR m->>'denominacion' IS DISTINCT FROM anterior.denominacion
     OR fecha_desde IS DISTINCT FROM anterior.vigente_desde
     OR fecha_hasta IS DISTINCT FROM anterior.vigente_hasta THEN
   RAISE EXCEPTION 'retirada sin revisión vigente' USING ERRCODE='23514'; END IF;
 END IF;
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_personal.entrada_catalogo_registro_empleado_historia(
  organismo_ref,tipo,ref,version,revision,estado,vigente_desde,vigente_hasta,huella_sha256,denominacion,
  acto_ref,actor_ref,idempotencia_ref,decision_ref,consumo_huella_sha256,auditoria_ref,registrado_en)
 VALUES(m->>'organismo_ref',m->>'tipo',m->>'ref',ver,rev,estado,fecha_desde,fecha_hasta,m->>'huella_sha256',m->>'denominacion',
  acto_interno,m->>'actor_ref',clave,v.decision_ref,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en);
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('entrada',jsonb_build_object('organismo_ref',m->>'organismo_ref','tipo',m->>'tipo','ref',m->>'ref',
  'version',ver,'revision',rev,'estado',estado,'huella_sha256',m->>'huella_sha256',
  'denominacion',m->>'denominacion',
  'vigente_desde',fecha_desde,'vigente_hasta',fecha_hasta),
  'recibo',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en),
  'acceso_actual',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en,
  'estado_replay','registrado'));
END $function$;
RESET ROLE;

DO $post$
DECLARE f regprocedure:='vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM '{vec_personal_propietario=X/vec_personal_propietario,vec_personal_ejecutor=X/vec_personal_propietario}'
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM true
 OR (SELECT proconfig::text FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM
    '{"search_path=pg_catalog, pg_temp",row_security=on,lock_timeout=2s}'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
   IS DISTINCT FROM '67c1721c5e122d97d086d46447026612c6dffe821dd580a329fa10f7a55466f0'
 THEN RAISE EXCEPTION 'Personal43: PARO clave=postimagen actual=divergente esperado=registrar_entrada_catalogo_empleado_rrhh_v1' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
