\set ON_ERROR_STOP on
-- AUT46: fechas prospectivas, sin mutación de historia ni publicación al instalar.
BEGIN;
SET LOCAL search_path=pg_catalog;SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN RAISE EXCEPTION 'AUT46: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';END IF;
 IF to_regprocedure('vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(jsonb)') IS NOT NULL THEN RAISE EXCEPTION 'AUT46: PARO clave=instalacion actual=utility_existente esperado=sin_AUT46' USING ERRCODE='55000';END IF;
END $pre$;
DO $guarda$ DECLARE actual text;BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.documento_asignacion_destino_mantenimiento_v1(jsonb,jsonb,text)') AND proowner='vec_autorizacion_propietario'::regrole;
 IF actual IS DISTINCT FROM '07418350915baf31efe860bffdc2bba5246b3b4de38ea02cc0bf6723470a84ff' THEN RAISE EXCEPTION 'AUT46: PARO clave=vec_autorizacion.documento_asignacion_destino_mantenimiento_v1.prosrc actual=% esperado=07418350915baf31efe860bffdc2bba5246b3b4de38ea02cc0bf6723470a84ff',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
END $guarda$;
DO $guarda$ DECLARE actual text;BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.aplicar_mantenimiento_perfil_fijo_admin_v1(text,text)') AND proowner='vec_autorizacion_propietario'::regrole;
 IF actual IS DISTINCT FROM '2d7d1d1eea131c0e3cb972a587577fd455c7115ca2260dae9a33575add03904f' THEN RAISE EXCEPTION 'AUT46: PARO clave=vec_autorizacion.aplicar_mantenimiento_perfil_fijo_admin_v1.prosrc actual=% esperado=2d7d1d1eea131c0e3cb972a587577fd455c7115ca2260dae9a33575add03904f',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
END $guarda$;
DO $guarda$ DECLARE actual text;BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_admin_v1(text,text)') AND proowner='vec_autorizacion_propietario'::regrole;
 IF actual IS DISTINCT FROM '5f65bc17ae17ea4844bd23f9a6ba8122b0315181b03987a4813901e004ee7eeb' THEN RAISE EXCEPTION 'AUT46: PARO clave=vec_autorizacion.mantener_version_perfil_fijo_admin_v1.prosrc actual=% esperado=5f65bc17ae17ea4844bd23f9a6ba8122b0315181b03987a4813901e004ee7eeb',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
END $guarda$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(d jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE inicio timestamptz;emision timestamptz;fin timestamptz;
BEGIN
 IF jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(d->'vigente_desde') IS DISTINCT FROM 'string' OR jsonb_typeof(d->'vigente_hasta') IS DISTINCT FROM 'string' OR jsonb_typeof(d->'emitida_en') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AUT46: fechas_asignacion_invalidas' USING ERRCODE='22023';END IF;
 inicio:=(d->>'vigente_desde')::timestamptz;emision:=(d->>'emitida_en')::timestamptz;fin:=(d->>'vigente_hasta')::timestamptz;
 IF NOT isfinite(inicio) OR NOT isfinite(emision) OR NOT isfinite(fin) THEN RAISE EXCEPTION 'AUT46: fechas_asignacion_no_finitas' USING ERRCODE='22023';END IF;
 IF inicio<emision THEN d:=jsonb_set(d,'{vigente_desde}',d->'emitida_en');inicio:=emision;END IF;
 IF inicio>=fin THEN RAISE EXCEPTION 'AUT46: ventana_asignacion_vacia' USING ERRCODE='22023';END IF;
 RETURN d;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(jsonb) FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_autorizacion.aplicar_mantenimiento_perfil_fijo_admin_v1(plan_canonico text, sha_aprobado text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE cfg vec_autorizacion.config_mantenimiento_perfil_fijo_admin_v1;p jsonb;sha text;pre jsonb;pre_sha text;r4 record;r5 record;target jsonb;target_sha text;control jsonb;control_sha text;c jsonb;meta record;gob record;t jsonb;old_a record;new_a record;doc jsonb;ref text;cat jsonb;events jsonb:='[]';aud record;e jsonb;corr text;recibo jsonb;replay boolean:=false;instante timestamptz;origenes jsonb:='[]';destinos jsonb:='[]';i int:=0;sello record;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_mantenimiento_fijo_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT42: PARO clave=plan actual=divergente esperado=plan_privado_aprobado' USING ERRCODE='42501'; END IF;
 sha:=encode(pg_catalog.sha256(convert_to(plan_canonico,'UTF8')),'hex');p:=plan_canonico::jsonb;
 IF sha IS DISTINCT FROM cfg.plan_sha256 OR p->>'catalogo_sha256' IS DISTINCT FROM cfg.catalogo_sha256 OR p#>>'{rol_destino_doc,publicada_por}' IS DISTINCT FROM 'mantenimiento_operador:'||session_user::text
 OR (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AUT42: PARO clave=aprobacion actual=divergente esperado=operador_plan_catalogo_vigentes' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO r5 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v5' FOR SHARE;replay:=FOUND;
 SELECT * INTO STRICT r4 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v4' FOR SHARE;
 target:=p->'rol_destino_doc';target_sha:=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(target),'UTF8')),'hex');
 IF NOT replay THEN
  pre:=vec_autorizacion.preimagen_mantenimiento_perfil_fijo_admin_v1(p);pre_sha:=encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex');
  IF pre_sha IS DISTINCT FROM cfg.preimagen_sha256 THEN RAISE EXCEPTION 'AUT42: PARO clave=preimagen actual=% esperado=%',pre_sha,cfg.preimagen_sha256 USING ERRCODE='40001'; END IF;
 ELSE
  pre_sha:=cfg.preimagen_sha256;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual q JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision) WHERE q.version_rol_ref='rol:administracion_perfiles:v5' AND x.estado='habilitada') THEN RAISE EXCEPTION 'AUT42: PARO clave=replay_control actual=no_vigente esperado=rol5_habilitado' USING ERRCODE='40001'; END IF;
  IF r5.huella_sha256 IS DISTINCT FROM target_sha OR r5.documento IS DISTINCT FROM target OR r5.documento->>'estado'<>'publicada' OR r4.huella_sha256 IS DISTINCT FROM p->>'rol_origen_sha256' THEN RAISE EXCEPTION 'AUT42: PARO clave=replay_rol actual=divergente esperado=publicacion_original_viva' USING ERRCODE='40001'; END IF;
 END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') ORDER BY value->>'perfil_ref' LOOP
  SELECT * INTO STRICT old_a FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=t->>'asignacion_origen_ref';
  IF old_a.huella_sha256 IS DISTINCT FROM t->>'asignacion_origen_sha256' OR old_a.version_rol_ref<>'rol:administracion_perfiles:v4' OR old_a.version<>1 OR old_a.principal_id IS DISTINCT FROM t->>'persona_ref' OR old_a.perfil_activo_ref IS DISTINCT FROM t->>'perfil_ref' THEN RAISE EXCEPTION 'AUT42: PARO clave=origen actual=divergente esperado=asignacion_original4' USING ERRCODE='40001'; END IF;
  doc:=vec_autorizacion.documento_asignacion_destino_mantenimiento_v1(old_a.documento,p,session_user::text);
  IF replay THEN
   SELECT x.* INTO STRICT new_a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=t->>'perfil_ref' FOR SHARE OF q;
   -- Primero coteja los bytes de la fórmula histórica; sólo una revisión
   -- prospectiva persistida permite elegir el candidato con fechas ajustadas.
   IF new_a.documento IS DISTINCT FROM doc THEN doc:=vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(doc);END IF;
  ELSE doc:=vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(doc);END IF;ref:='asignacion:'||old_a.asignacion_id||':v2';
  origenes:=origenes||jsonb_build_array(jsonb_build_object('ref',old_a.asignacion_ref,'sha',old_a.huella_sha256));destinos:=destinos||jsonb_build_array(jsonb_build_object('ref',ref,'sha',encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),'documento',doc));
  IF replay THEN
   SELECT x.* INTO STRICT new_a FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE q.perfil_activo_ref=t->>'perfil_ref' FOR SHARE OF q;
   SELECT * INTO STRICT sello FROM vec_autorizacion.sello_efecto_admin_tx_v1 WHERE asignacion_ref=ref;
   IF new_a.asignacion_ref IS DISTINCT FROM ref OR new_a.documento IS DISTINCT FROM doc OR new_a.huella_sha256 IS DISTINCT FROM destinos#>>'{-1,sha}' OR new_a.documento->>'estado'<>'activa' OR clock_timestamp()>=(new_a.documento->>'vigente_hasta')::timestamptz OR sello.operacion_ref IS DISTINCT FROM p->>'operacion_ref' THEN RAISE EXCEPTION 'AUT42: PARO clave=replay_asignacion actual=revocada_o_divergente esperado=destino_original_sin_rescate' USING ERRCODE='40001'; END IF;
   IF vec_contexto_actor_v1.bloquear_contexto_admin_v1(t->>'cuenta_ref',t->>'persona_ref',t->>'perfil_ref',t->>'vinculo_ref',(t->>'cuenta_version')::numeric,(t->>'persona_version')::numeric,(t->>'perfil_version')::numeric,(t->>'vinculo_version')::numeric) IS NOT TRUE
   OR vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(t->'ambitos_fuente',(new_a.documento->>'vigente_hasta')::timestamptz)->'ambitos' IS DISTINCT FROM new_a.documento->'ambitos' THEN
    RAISE EXCEPTION 'AUT42: PARO clave=replay_CA_ambitos actual=divergente esperado=CA_y_ambitos_originales_vivos' USING ERRCODE='42501';
   END IF;
  END IF;
 END LOOP;
 corr:='correlacion_'||substr(sha,33,32);
 e:=jsonb_build_object('tipo_registro','mantenimiento_perfil_fijo_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',pre_sha,'catalogo_sha256',cfg.catalogo_sha256,
  'rol_origen_ref','rol:administracion_perfiles:v4','rol_origen_sha256',r4.huella_sha256,'rol_destino_ref','rol:administracion_perfiles:v5','rol_destino_sha256',target_sha,
  'asignacion_1_origen_ref',origenes#>>'{0,ref}','asignacion_1_origen_sha256',origenes#>>'{0,sha}','asignacion_1_destino_ref',destinos#>>'{0,ref}','asignacion_1_destino_sha256',destinos#>>'{0,sha}',
  'asignacion_2_origen_ref',origenes#>>'{1,ref}','asignacion_2_origen_sha256',origenes#>>'{1,sha}','asignacion_2_destino_ref',destinos#>>'{1,ref}','asignacion_2_destino_sha256',destinos#>>'{1,sha}',
  'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr);
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(e);
 IF replay THEN
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(destinos) x JOIN vec_autorizacion.sello_efecto_admin_tx_v1 z ON z.asignacion_ref=x.value->>'ref' WHERE z.auditoria_ref IS DISTINCT FROM aud.auditoria_ref OR z.operacion_ref IS DISTINCT FROM p->>'operacion_ref') THEN RAISE EXCEPTION 'AUT42: PARO clave=acuse actual=divergente esperado=confirmacion_comun_original' USING ERRCODE='40001'; END IF;
 ELSE
  instante:=(target->>'publicada_en')::timestamptz;
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES('rol:administracion_perfiles:v5','administracion_perfiles',5,target_sha,instante,target);
  control:=jsonb_build_object('version_rol_ref','rol:administracion_perfiles:v5','revision',1,'estado','habilitada','actualizado_por','mantenimiento_operador:'||session_user::text,'actualizado_en',target->>'publicada_en');control_sha:=encode(pg_catalog.sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento,creada_en) VALUES('rol:administracion_perfiles:v5',1,'habilitada',control_sha,instante,control,clock_timestamp());
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES('rol:administracion_perfiles:v5',1,instante,'mantenimiento_operador:'||session_user::text,p->>'operacion_ref');
  INSERT INTO vec_autorizacion.rol_sensible_exacto VALUES('rol:administracion_perfiles:v5','administrador',target_sha);
  INSERT INTO vec_autorizacion.perfil_fijo_categoria_nominal_v1 VALUES('rol:administracion_perfiles:v5',target_sha,'aplicacion','fijo_sistema','rol:administracion_perfiles:v5',5,target_sha,instante);
  SELECT * INTO STRICT gob FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref='rol:administracion_perfiles:v4';
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1 SELECT 'rol:administracion_perfiles:v5',gob.clase,target_sha,gob.vigente_desde,gob.vigente_hasta,gob.unidad_requerida,gob.audiencia_administrativa,gob.ambitos_fijos,gob.duracion_propuesta;
  FOR c IN SELECT to_jsonb(x) FROM vec_autorizacion.catalogo_accion_nominal_v1 x WHERE x.version_rol_ref='rol:administracion_perfiles:v4' ORDER BY x.accion_ref LOOP
   INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1 VALUES(c->>'accion_ref',2,'rol:administracion_perfiles:v5',5,target_sha,'rol:administracion_perfiles:v5',c->'concesion',c->'dimensiones_ambito','administrador_aplicacion',instante,(c->>'vigente_hasta')::timestamptz);
  END LOOP;
  FOR c IN SELECT value FROM jsonb_array_elements(vec_autorizacion.concesiones_mantenimiento_fijo_admin_v1()) LOOP
   INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1 VALUES('accion:'||(c->>'accion'),1,'rol:administracion_perfiles:v5',5,target_sha,'rol:administracion_perfiles:v5',c,'["organizacion_ref","unidad_ref"]','administrador_aplicacion',instante,NULL);
  END LOOP;
  FOR t IN SELECT value FROM jsonb_array_elements(destinos) LOOP
   doc:=t->'documento';
   INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento) VALUES(t->>'ref',doc->>'asignacion_id',2,doc->>'perfil_activo_ref',doc->>'principal_id','rol:administracion_perfiles:v5',t->>'sha',(doc->>'emitida_en')::timestamptz,doc);
   INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(t->>'ref',txid_current(),p->>'operacion_ref',aud.auditoria_ref);
   UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=t->>'ref',actualizada_en=clock_timestamp(),actualizada_por='mantenimiento_operador:'||session_user::text,acto_ref=p->>'operacion_ref' WHERE perfil_activo_ref=doc->>'perfil_activo_ref' AND asignacion_ref='asignacion:'||(doc->>'asignacion_id')||':v1';
   IF NOT FOUND THEN RAISE EXCEPTION 'AUT42: PARO clave=CAS_final actual=divergente esperado=puntero_v1_original' USING ERRCODE='40001'; END IF;
  END LOOP;
 END IF;
 PERFORM vec_autorizacion.exigir_operador_mantenimiento_fijo_admin_v1();IF clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'AUT42: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501';END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(destinos) x WHERE clock_timestamp()<(x.value#>>'{documento,vigente_desde}')::timestamptz OR clock_timestamp()>=(x.value#>>'{documento,vigente_hasta}')::timestamptz) THEN RAISE EXCEPTION 'AUT46: PARO clave=ventana_final actual=no_viva esperado=ventana_original_viva' USING ERRCODE='42501';END IF;
 recibo:=jsonb_build_object('esquema','vec.admin.mantenimiento-fijo.v1','operacion_ref',p->>'operacion_ref','plan_sha256',sha,'rol_origen_ref','rol:administracion_perfiles:v4','rol_destino_ref','rol:administracion_perfiles:v5','rol_destino_sha256',target_sha,'asignaciones',(SELECT jsonb_agg(x.value-'documento') FROM jsonb_array_elements(destinos) x),'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 RETURN jsonb_build_object('recibo',recibo,'replay',replay);
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.mantener_version_perfil_fijo_admin_v1(plan_canonico text, sha_aprobado text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;sol text;evento text;corr text;solsha text;
BEGIN
 sol:='solicitud_mantenimiento:'||replace(gen_random_uuid()::text,'-','');evento:='evento_'||replace(gen_random_uuid()::text,'-','');corr:='correlacion_'||replace(gen_random_uuid()::text,'-','');solsha:=encode(pg_catalog.sha256(convert_to(jsonb_build_object('plan',plan_canonico,'sha_aprobado',sha_aprobado)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_mantenimiento_perfil_fijo_admin_v1(plan_canonico,sha_aprobado);motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'mantenimiento_replay' ELSE 'mantenimiento_registrado' END;
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb_build_object('tipo_registro','intento_mantenimiento_perfil_fijo_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','mantener_version_perfil_fijo_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr));
  PERFORM vec_autorizacion.exigir_operador_mantenimiento_fijo_admin_v1();
  IF clock_timestamp()>=(plan_canonico::jsonb->>'caduca_en')::timestamptz OR EXISTS(SELECT 1 FROM jsonb_array_elements(respuesta#>'{recibo,asignaciones}') x JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x.value->>'ref' WHERE clock_timestamp()<(a.documento->>'vigente_desde')::timestamptz OR clock_timestamp()>=(a.documento->>'vigente_hasta')::timestamptz) THEN RAISE EXCEPTION 'AUT46: PARO clave=vigencia_tras_append actual=no_viva esperado=plan_y_ventanas_originales_vivas' USING ERRCODE='42501';END IF;
 EXCEPTION WHEN OTHERS THEN respuesta:=NULL;codigo:=SQLSTATE;estado:=CASE WHEN codigo IN('42501','22023','40001','23505','25000') THEN 'denegado' ELSE 'error' END;motivo:=CASE WHEN estado='denegado' THEN 'mantenimiento_denegado' ELSE 'mantenimiento_error' END;codigo:=CASE WHEN estado='denegado' THEN 'mantenimiento_rechazado' ELSE 'mantenimiento_no_disponible' END;
 END;
 IF estado<>'permitido' THEN
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb_build_object('tipo_registro','intento_mantenimiento_perfil_fijo_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','mantener_version_perfil_fijo_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','mantenimiento_perfil_fijo_admin','correlacion_ref',corr));
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo','replay',COALESCE((respuesta->>'replay')::boolean,false),'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $function$;

COMMIT;
