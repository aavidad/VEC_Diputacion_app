\set ON_ERROR_STOP on
-- Sólo para el arnés desechable de Dietas 000010, después de
-- devolucion_000010_preparar.sql. TEST-ONLY: no es migración ni consumidor
-- instalable. Sustituye en este contenedor las fachadas AD3 «documento» y
-- «revisor_documento» y la revalidación Personal de la asignación por dobles
-- que consumen una sola vez, registra la relación Personal real de la
-- titular sintética y publica dos selladores que construyen material,
-- decisión, capacidad y contexto con las mismas huellas que calcula Go. Así
-- se invocan las funciones nominales reales mutar_comision_propia_v2 y
-- consultar_documento_circuito_v1, con su RLS y sus comprobaciones.
BEGIN;
DO $fachadas$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['documento','revisor_documento'] LOOP
  EXECUTE format($f$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_%s_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $c$
   DECLARE c jsonb; d jsonb; insertada boolean;
   BEGIN
    c:=convert_from($1,'UTF8')::jsonb; d:=convert_from($2,'UTF8')::jsonb;
    IF c->>'nonce' IS NULL OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref' THEN
     RAISE EXCEPTION 'doble AD3: capacidad y decisión incompatibles' USING ERRCODE='PD003'; END IF;
    INSERT INTO vec_autorizacion_atestada_v3.stub_consumo VALUES(c->>'nonce') ON CONFLICT DO NOTHING RETURNING true INTO insertada;
    RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
     encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'aad_stub_'||substr(c->>'nonce',1,16),clock_timestamp(),coalesce(insertada,false);
   END $c$$f$, nombre);
 END LOOP;
END $fachadas$;
CREATE OR REPLACE FUNCTION vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)
RETURNS boolean LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT true $$;
INSERT INTO vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES ('rel_RRRRRRRRRRRRRRRRRRRRRR','per_TTTTTTTTTTTTTTTTTTTTTT','emp_EEEEEEEEEEEEEEEEEEEEEE','U01','activa',DATE '2026-01-01',NULL,1,'acto','fuente',1);

CREATE SCHEMA prueba_000010;
-- Borrado propio: material v2 con la huella semántica de Go, decisión y
-- capacidad ligadas a la huella de efecto de cotejar_recurso_documento_v2.
CREATE FUNCTION prueba_000010.sellar_borrado(p_ref text,p_version bigint,p_clave text,
 OUT material text,OUT capacidad bytea,OUT decision bytea,OUT contexto bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE i jsonb; comando text; h text; hc text; amb text; atr text; d jsonb; campos jsonb;
BEGIN
 i:=jsonb_build_object('persona_ref','per_TTTTTTTTTTTTTTTTTTTTTT','empleado_ref','emp_EEEEEEEEEEEEEEEEEEEEEE',
  'relacion_ref','rel_RRRRRRRRRRRRRRRRRRRRRR','unidad_ref','U01','relacion_version',1,
  'actor_ref','per_TTTTTTTTTTTTTTTTTTTTTT','perfil_ref','perfil_empleado','cuenta_ref','cta_titular',
  'contexto_actor_ref','ctx_titular','contexto_version',1,'cuenta_version',1,'persona_version',1,'perfil_version',1,
  'fecha_referencia','2026-09-24','fuente_ref','fuente','fuente_version',1,'procedencia_acto_ref','acto',
  'vigente_desde','2026-01-01','vigente_hasta','');
 contexto:=convert_to(jsonb_build_object('contexto_actor_ref','ctx_titular','contexto_version',1,'cuenta_ref','cta_titular',
  'cuenta_version',1,'esquema','vec.contexto-actor.vinculado.v2','estado','activo','garantia','alta','metodo','certificado',
  'perfil_activo_ref','perfil_empleado','perfil_version',1,'persona_ref','per_TTTTTTTTTTTTTTTTTTTTTT','persona_version',1,
  'principal_ref','per_TTTTTTTTTTTTTTTTTTTTTT','resuelto_en','2026-09-24T08:00:00Z','vigente_desde','2026-09-24T08:00:00Z',
  'vigente_hasta','2026-09-24T09:00:00Z','vinculos',jsonb_build_array(jsonb_build_object('estado','activo',
   'referencia','emp_EEEEEEEEEEEEEEEEEEEEEE','tipo','empleado','version',1,'vigente_desde','2026-01-01',
   'vigente_hasta','2027-01-01','vinculo_ref','vin_titular')))::text,'UTF8');
 hc:=encode(sha256(contexto),'hex');
 comando:='{"clave_idempotencia":"'||p_clave||'","version_esperada":'||p_version||',"relacion_ref":"rel_RRRRRRRRRRRRRRRRRRRRRR"}';
 material:='{"esquema":"vec.dietas.borrador-operacion.v2","operacion":"borrar","recurso_ref":"'||p_ref||
  '","identidad":'||i::text||',"huella_semantica":"#","comando":'||comando||',"referencia":"'||p_ref||'"}';
 material:=replace(material,'"huella_semantica":"#"','"huella_semantica":"'||vec_dietas.huella_semantica_mutacion_v2(material)||'"');
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(material,'UTF8')),'hex')||'","operacion":"borrar","perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||to_jsonb(p_ref)::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":"sin_fin"}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 d:=jsonb_build_object('esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2','concedida',true,'bloque_version',3,
  'accion','dietas.borrador.propio.borrar','finalidad','borrar_borrador_propio','modulo_id','dietas',
  'tipo_recurso','comision_borrador','recurso_ref',p_ref,'campos_permitidos',campos,'obligaciones','[]'::jsonb,
  'decision_ref','dec_'||p_clave,'principal_id',i->>'actor_ref','perfil_activo_ref',i->>'perfil_ref',
  'contexto_recurso_huella_sha256',h,
  'vinculo_autenticacion_actor',jsonb_build_object('esquema','vec.autorizacion.vinculo-autenticacion-actor.v2',
   'autoridad_efectiva','autoridad_maestra_acreditada','principal_id',i->>'actor_ref','perfil_activo_ref',i->>'perfil_ref',
   'cuenta_ref',i->>'cuenta_ref','contexto_actor_ref',i->>'contexto_actor_ref','contexto_actor_version',i->>'contexto_version',
   'contexto_actor_cuenta_version',i->>'cuenta_version','contexto_actor_huella_sha256',hc,'registro_contexto_ref','rctx_titular'));
 decision:=convert_to(d::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('esquema','vec.autorizacion.capacidad-registro-consumo-atestado.v3',
  'operacion','dietas.borrador.propio.borrar','audiencia_consumo','vec_dietas.borrador_propio.borrar.v1',
  'efecto_ref',p_ref,'decision_ref','dec_'||p_clave,'huella_decision_sha256',encode(sha256(decision),'hex'),
  'contexto_ref','rctx_titular','huella_contexto_sha256',hc,'huella_efecto_sha256',h,'nonce','n_'||p_clave)::text,'UTF8');
END $$;

-- Lectura del documento por quien revisa en su etapa: material v2 del
-- circuito con la huella de cotejar_efecto_circuito_v2.
CREATE FUNCTION prueba_000010.sellar_consulta(p_ref text,p_etapa text,p_unidad text,p_persona text,p_actor text,p_nonce text,
 OUT material text,OUT capacidad bytea,OUT decision bytea,OUT contexto bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE i jsonb; h text; amb text; atr text; d jsonb;
BEGIN
 i:=jsonb_build_object('persona_ref',p_persona,'actor_ref',p_actor,'perfil_ref','perfil_revisor','cuenta_ref','cta_revisor',
  'contexto_actor_ref','ctx_revisor','contexto_version','1','cuenta_version','1','persona_version','1','perfil_version','1');
 contexto:=convert_to(jsonb_build_object('contexto_actor_ref','ctx_revisor','contexto_version','1','cuenta_ref','cta_revisor',
  'cuenta_version','1','estado','activo','persona_ref',p_persona,'principal_ref',p_persona,'perfil_activo_ref','perfil_revisor',
  'persona_version','1','perfil_version','1')::text,'UTF8');
 material:=jsonb_build_object('esquema','vec.dietas.circuito-operacion.v2','operacion','consultar_documento','etapa',p_etapa,
  'unidad_ref',p_unidad,'recurso_ref',p_ref,'identidad',i)::text;
 amb:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(p_persona)||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(p_unidad)||'}';
 atr:='"contexto_actor_ref":'||vec_dietas.cadena_json_go_v1('ctx_revisor')||',"contexto_version":"1","cuenta_ref":'||
  vec_dietas.cadena_json_go_v1('cta_revisor')||',"cuenta_version":"1","etapa":'||vec_dietas.cadena_json_go_v1(p_etapa)||
  ',"material_sha256":"'||encode(sha256(convert_to(material,'UTF8')),'hex')||'","operacion":"consultar_documento"'||
  ',"perfil_version":"1","persona_version":"1","recurso_ref":'||vec_dietas.cadena_json_go_v1(p_ref);
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":{'||atr||'}}','UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','dec_'||p_nonce,'principal_id',p_actor,'perfil_activo_ref','perfil_revisor',
  'concedida',true,'modulo_id','dietas','tipo_recurso','documento_dietas','obligaciones','[]'::jsonb,'recurso_ref',p_ref,
  'accion','dietas.circuito.documento.consultar','finalidad','revisar_documento_circuito_dietas',
  'campos_permitidos','["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb,
  'contexto_recurso_huella_sha256',h);
 decision:=convert_to(d::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('operacion','dietas.circuito.documento.consultar',
  'audiencia_consumo','vec_dietas.circuito.documento.consultar.v1','efecto_ref',p_ref,'decision_ref','dec_'||p_nonce,
  'huella_decision_sha256',encode(sha256(decision),'hex'),'huella_contexto_sha256',encode(sha256(contexto),'hex'),
  'huella_efecto_sha256',h,'nonce','n_'||p_nonce)::text,'UTF8');
END $$;
GRANT USAGE ON SCHEMA prueba_000010 TO vec_prueba_dietas;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_000010 TO vec_prueba_dietas;
COMMIT;
