\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor desechable del ensayo de AD3-75 y
-- Dietas 000011. Nunca se instala en una base conservada. Se carga DESPUÉS de
-- comprobar las fachadas AD3-75 reales: las sustituye en ESTE contenedor por
-- dobles de un solo uso (nonce) para recorrer la lógica de Dietas sin claves
-- V3, igual que devolucion_000010_fachadas.sql, pero incluida la consulta de
-- la titular. Publica selladores con la lista de campos elegida por el
-- ensayo y las mismas huellas que calcula Go, para invocar las funciones
-- nominales reales consultar_comisiones_propias_v2 y
-- consultar_documento_circuito_v1 con su RLS y sus cotejos.
BEGIN;
CREATE ROLE vec_prueba_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE SCHEMA prueba_000011 AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA prueba_000011 FROM PUBLIC;
CREATE TABLE prueba_000011.consumo(nonce text PRIMARY KEY);
REVOKE ALL ON prueba_000011.consumo FROM PUBLIC;
GRANT USAGE ON SCHEMA prueba_000011 TO vec_autorizacion_atestada_v3_propietario;
GRANT SELECT,INSERT ON prueba_000011.consumo TO vec_autorizacion_atestada_v3_propietario;

DO $fachadas$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['documento','documento_consulta','revisor_documento'] LOOP
  EXECUTE format($f$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_%s_v3_atestada(
   p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $c$
   DECLARE c jsonb; d jsonb; insertada boolean;
   BEGIN
    c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
    IF c->>'nonce' IS NULL OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref' THEN
     RAISE EXCEPTION 'doble AD3: capacidad y decisión incompatibles' USING ERRCODE='PD003'; END IF;
    INSERT INTO prueba_000011.consumo VALUES(c->>'nonce') ON CONFLICT DO NOTHING RETURNING true INTO insertada;
    RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
     encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'aad_doble_'||substr(c->>'nonce',1,16),clock_timestamp(),coalesce(insertada,false);
   END $c$$f$, nombre);
 END LOOP;
END $fachadas$;
-- La asignación sintética del reenvío no existe en Personal: la revalidación
-- real (Personal 000012) se sustituye aquí, con sus mismos parámetros.
DO $revalidar$
BEGIN
 EXECUTE format('CREATE OR REPLACE FUNCTION vec_personal.revalidar_asignacion_dietas_v1(%s) RETURNS boolean LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $c$ SELECT true $c$',
  pg_get_function_arguments('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)'::regprocedure));
END $revalidar$;
INSERT INTO vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES ('rel_RRRRRRRRRRRRRRRRRRRRRR','per_TTTTTTTTTTTTTTTTTTTTTT','emp_EEEEEEEEEEEEEEEEEEEEEE','U01','activa',DATE '2026-01-01',NULL,1,'acto','fuente',1);

-- Titular: detalle, listado o borrado propio con la lista de campos dada.
-- Material v2, contexto y huella de efecto de cotejar_recurso_documento_v2.
CREATE FUNCTION prueba_000011.sellar_titular(p_operacion text,p_ref text,p_campos jsonb,p_nonce text,
 OUT material text,OUT capacidad bytea,OUT decision bytea,OUT contexto bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE i jsonb; recurso text; accion text; finalidad text; audiencia text; h text; hc text; amb text; atr text; d jsonb;
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
 IF p_operacion='lista' THEN
  recurso:='dietas:borradores:propios';
  material:='{"esquema":"vec.dietas.borrador-operacion.v2","operacion":"lista","recurso_ref":"'||recurso||
   '","identidad":'||i::text||',"consulta":{"limite":10,"cursor":""}}';
 ELSIF p_operacion='detalle' THEN
  recurso:=p_ref;
  material:='{"esquema":"vec.dietas.borrador-operacion.v2","operacion":"detalle","recurso_ref":"'||recurso||
   '","identidad":'||i::text||',"referencia":"'||recurso||'"}';
 ELSE
  recurso:=p_ref;
  material:='{"esquema":"vec.dietas.borrador-operacion.v2","operacion":"borrar","recurso_ref":"'||recurso||
   '","identidad":'||i::text||',"huella_semantica":"#","comando":{"clave_idempotencia":"clave'||p_nonce||
   'Borrado","version_esperada":1,"relacion_ref":"rel_RRRRRRRRRRRRRRRRRRRRRR"},"referencia":"'||recurso||'"}';
  material:=replace(material,'"huella_semantica":"#"','"huella_semantica":"'||vec_dietas.huella_semantica_mutacion_v2(material)||'"');
 END IF;
 IF p_operacion IN ('lista','detalle') THEN
  accion:='dietas.documento.propio.consultar'; finalidad:='consultar_documento_propio_dietas';
  audiencia:='vec_dietas.documento_propio.consultar.v1';
 ELSE
  accion:='dietas.borrador.propio.borrar'; finalidad:='borrar_borrador_propio';
  audiencia:='vec_dietas.borrador_propio.borrar.v1';
 END IF;
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(material,'UTF8')),'hex')||'","operacion":'||to_jsonb(p_operacion)::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||to_jsonb(recurso)::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":"sin_fin"}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 d:=jsonb_build_object('esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2','concedida',true,'bloque_version',3,
  'accion',accion,'finalidad',finalidad,'modulo_id','dietas',
  'tipo_recurso','comision_borrador','recurso_ref',recurso,'campos_permitidos',p_campos,'obligaciones','[]'::jsonb,
  'decision_ref','dec_'||p_nonce,'principal_id',i->>'actor_ref','perfil_activo_ref',i->>'perfil_ref',
  'correlacion_ref','cor_'||p_nonce,'contexto_recurso_huella_sha256',h,
  'vinculo_autenticacion_actor',jsonb_build_object('esquema','vec.autorizacion.vinculo-autenticacion-actor.v2',
   'autoridad_efectiva','autoridad_maestra_acreditada','principal_id',i->>'actor_ref','perfil_activo_ref',i->>'perfil_ref',
   'cuenta_ref',i->>'cuenta_ref','contexto_actor_ref',i->>'contexto_actor_ref','contexto_actor_version',i->>'contexto_version',
   'contexto_actor_cuenta_version',i->>'cuenta_version','contexto_actor_huella_sha256',hc,'registro_contexto_ref','rctx_titular'));
 decision:=convert_to(d::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('esquema','vec.autorizacion.capacidad-registro-consumo-atestado.v3',
  'operacion',accion,'audiencia_consumo',audiencia,
  'efecto_ref',recurso,'decision_ref','dec_'||p_nonce,'huella_decision_sha256',encode(sha256(decision),'hex'),
  'contexto_ref','rctx_titular','huella_contexto_sha256',hc,'huella_efecto_sha256',h,'nonce','n_'||p_nonce)::text,'UTF8');
END $$;

-- Quien revisa: lectura del documento en su etapa con la lista dada.
CREATE FUNCTION prueba_000011.sellar_revisor(p_ref text,p_etapa text,p_unidad text,p_persona text,p_actor text,
 p_campos jsonb,p_nonce text,OUT material text,OUT capacidad bytea,OUT decision bytea,OUT contexto bytea)
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
  'campos_permitidos',p_campos,'contexto_recurso_huella_sha256',h);
 decision:=convert_to(d::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('operacion','dietas.circuito.documento.consultar',
  'audiencia_consumo','vec_dietas.circuito.documento.consultar.v1','efecto_ref',p_ref,'decision_ref','dec_'||p_nonce,
  'huella_decision_sha256',encode(sha256(decision),'hex'),'huella_contexto_sha256',encode(sha256(contexto),'hex'),
  'huella_efecto_sha256',h,'nonce','n_'||p_nonce)::text,'UTF8');
END $$;
GRANT USAGE ON SCHEMA prueba_000011 TO vec_prueba_dietas;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_000011 TO vec_prueba_dietas;
-- Solo en este contenedor: el login de prueba coteja directamente el borrado
-- propio (la respuesta de mutación usa la misma autorización que el detalle).
GRANT EXECUTE ON FUNCTION vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea) TO vec_prueba_dietas;
COMMIT;
