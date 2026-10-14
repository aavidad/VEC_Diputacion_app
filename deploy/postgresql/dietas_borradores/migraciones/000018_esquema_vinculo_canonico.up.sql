\set ON_ERROR_STOP on
-- Dietas18: los cotejos instalados conservaban un identificador de esquema
-- ajeno al contrato V2 vigente. Se corrige el literal en ambos consumidores.
-- Definiciones completas obtenidas de la copia de la principal; no se altera
-- bloque_version, la huella del recurso, la identidad ni sus restricciones.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000018',0));

DO $pre_0$
DECLARE f oid:=to_regprocedure('vec_dietas.cotejar_recurso_dietas_borrador_v1(text,bytea,bytea,bytea)'); p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 OR current_user<>'vec_dietas_propietario' OR f IS NULL THEN
  RAISE EXCEPTION 'Dietas18: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
  OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario}'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'9cb8506b476c6baae1bc888b5cceba94e05c9ce07146af2e2d223a3ce1b8d535'
  OR (SELECT count(*) FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f)<>2
 THEN RAISE EXCEPTION 'Dietas18: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre_0$;

CREATE OR REPLACE FUNCTION vec_dietas.cotejar_recurso_dietas_borrador_v1(p_material text, p_capacidad bytea, p_decision bytea, p_contexto bytea)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; v jsonb; amb text; atr text; h text; hasta text; accion text; finalidad text; audiencia text; hc text; campos_esperados jsonb; campos_calculo jsonb;
  claves_decision text[]:=ARRAY['accion','asignacion_huella_sha256','asignacion_ref','bloque_version','campos_permitidos','catalogo_politicas_huella_sha256','codigo','concedida','contexto_recurso_huella_sha256','control_vigencia_version_rol_huella_sha256','control_vigencia_version_rol_ref','control_vigencia_version_rol_revision','correlacion_ref','decision_ref','emitida_en','esquema','esquema_huella_motivo','esquema_huella_solicitud','finalidad','garantia_minima','modulo_id','motivo_huella_sha256','obligaciones','perfil_activo_ref','politicas_aplicables','politicas_evaluadas','principal_id','recurso_ref','revision_catalogo_politicas','solicitud_huella_sha256','tipo_recurso','valida_hasta','version_rol_huella_sha256','version_rol_ref','vinculo_autenticacion_actor'];
  claves_vinculo text[]:=ARRAY['asercion_ref','autenticacion_huella_sha256','autenticacion_ref','autenticacion_verificada_en','autoridad_efectiva','bloque_version','contexto_actor_cuenta_version','contexto_actor_esquema','contexto_actor_huella_sha256','contexto_actor_ref','contexto_actor_version','control_sesion_huella_sha256','control_sesion_ref','control_sesion_revision','cuenta_ordinaria_ref','cuenta_privilegiada','cuenta_ref','esquema','garantia_observada','manifiesto_procedencia_huella_sha256','metodo_observado','perfil_activo_ref','politica_garantia_huella_sha256','politica_garantia_ref','principal_id','registro_contexto_ref','sesion_emitida_en','sesion_ref','sesion_revalidada_en','sesion_valida_hasta','superficie'];
  claves_capacidad text[]:=ARRAY['audiencia_consumo','audiencia_despliegue','clave_id','clave_version','configuracion_expira_en','configuracion_publicada_en','configuracion_secuencia','contexto_ref','decision_ref','decision_valida_hasta','efecto_ref','emisor_id','emitida_en','esquema','expira_en','huella_configuracion_sha256','huella_contexto_sha256','huella_decision_sha256','huella_efecto_sha256','huella_gobierno_sha256','huella_motivo_sha256','huella_payload_vec_ad_3_sha256','huella_prueba_confianza_sha256','huella_raiz_spki_sha256','huella_sobre_cose_sign1_sha256','mac_sha256','nonce','operacion','raiz_clave_id','raiz_valida_desde','raiz_valida_hasta','raiz_version','revision_confianza','revision_gobierno','suite','verificada_en','version'];
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RETURN false; END IF;
 IF p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL THEN RETURN false; END IF;
 BEGIN m:=p_material::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RETURN false; END; i:=m->'identidad'; hasta:=coalesce(i->>'vigente_hasta',''); IF hasta='' THEN hasta:='sin_fin'; END IF;
 IF i IS NULL OR m->>'esquema'<>'vec.dietas.borrador-operacion.v1' OR m->>'recurso_ref' IS NULL OR vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) IS NOT TRUE THEN RETURN false; END IF;
 -- Dietas aún no tiene un preflight gobernado que traduzca restricciones de
 -- campo u obligaciones. Hasta que exista, la proyección exacta del contrato
 -- de salida es la única aceptada y toda obligación se deniega antes de AD3.
 campos_esperados:=CASE m->>'operacion'
   WHEN 'crear' THEN '["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'detalle' THEN '["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'lista' THEN '["items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb
   ELSE NULL END;
 campos_calculo:=CASE m->>'operacion'
   WHEN 'crear' THEN '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'detalle' THEN '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb
   WHEN 'lista' THEN '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb END;
 IF campos_esperados IS NULL OR (d->'campos_permitidos' IS DISTINCT FROM campos_esperados AND d->'campos_permitidos' IS DISTINCT FROM campos_calculo) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RETURN false; END IF;
 v:=d->'vinculo_autenticacion_actor'; hc:=encode(sha256(p_contexto),'hex');
 IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM claves_decision
    OR jsonb_typeof(v)<>'object' OR ARRAY(SELECT jsonb_object_keys(v) ORDER BY 1) IS DISTINCT FROM claves_vinculo
    OR ARRAY(SELECT jsonb_object_keys(c) ORDER BY 1) IS DISTINCT FROM claves_capacidad
    OR d->>'esquema'<>'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2' OR d->>'bloque_version'<>'3'
    OR d->>'concedida'<>'true' OR v->>'esquema'<>'vec.autenticacion-actor.vinculo.v2.contexto-registrado' OR v->>'bloque_version'<>'2'
    OR v->>'autoridad_efectiva'<>'autoridad_maestra_acreditada'
    OR v->>'principal_id' IS DISTINCT FROM i->>'actor_ref' OR v->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR v->>'cuenta_ref' IS DISTINCT FROM i->>'cuenta_ref' OR v->>'contexto_actor_ref' IS DISTINCT FROM i->>'contexto_actor_ref'
    OR v->>'contexto_actor_version' IS DISTINCT FROM i->>'contexto_version' OR v->>'contexto_actor_cuenta_version' IS DISTINCT FROM i->>'cuenta_version'
    OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM hc OR c->>'contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref'
    OR c->>'huella_contexto_sha256' IS DISTINCT FROM hc OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') OR c->>'esquema'<>'vec.autorizacion.capacidad-registro-consumo-atestado.v3' OR c->>'version'<>'3'
 THEN RETURN false; END IF;
 accion:=CASE m->>'operacion' WHEN 'crear' THEN 'dietas.borrador.propio.crear' WHEN 'detalle' THEN 'dietas.borrador.propio.consultar' WHEN 'lista' THEN 'dietas.borrador.propio.consultar' ELSE NULL END;
 finalidad:=CASE m->>'operacion' WHEN 'crear' THEN 'crear_borrador_propio' WHEN 'detalle' THEN 'consultar_borrador_propio' WHEN 'lista' THEN 'consultar_borrador_propio' ELSE NULL END;
 audiencia:=CASE m->>'operacion' WHEN 'crear' THEN 'vec_dietas.borrador_propio.crear.v1' WHEN 'detalle' THEN 'vec_dietas.borrador_propio.consultar.v1' WHEN 'lista' THEN 'vec_dietas.borrador_propio.consultar.v1' ELSE NULL END;
 IF accion IS NULL OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref' OR d->>'correlacion_ref' IS NULL OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia THEN RETURN false; END IF;
 -- Mismos ámbitos que el RecursoAutorizable de Go (persona y empleado): relación y
 -- unidad quedan ligadas por material_sha256 y por la revalidación de Personal.
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'","operacion":'||to_jsonb(m->>'operacion')::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||(m->'recurso_ref')::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":'||to_jsonb(hasta)::text||'}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 RETURN d->>'accion'=accion AND d->>'finalidad'=finalidad AND d->>'modulo_id'='dietas' AND d->>'tipo_recurso'='comision_borrador' AND d->>'recurso_ref'=m->>'recurso_ref' AND d->>'contexto_recurso_huella_sha256'=h AND c->>'operacion'=accion AND c->>'efecto_ref'=m->>'recurso_ref' AND c->>'huella_efecto_sha256'=h;
END $function$


DO $pre_1$
DECLARE f oid:=to_regprocedure('vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea)'); p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 OR current_user<>'vec_dietas_propietario' OR f IS NULL THEN
  RAISE EXCEPTION 'Dietas18: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
  OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario}'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'1d39d125588b857d1f1c7ee2cdebec2e5f53cae0c0ef210068b9448bf94a3407'
  OR (SELECT count(*) FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f)<>2
 THEN RAISE EXCEPTION 'Dietas18: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre_1$;

CREATE OR REPLACE FUNCTION vec_dietas.cotejar_recurso_documento_v2(p_material text, p_capacidad bytea, p_decision bytea, p_contexto bytea)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; v jsonb; operacion text; accion text; finalidad text; audiencia text;
        amb text; atr text; h text; hasta text; hc text; campos jsonb; campos_devolucion jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RETURN false; END IF;
 IF p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL THEN RETURN false; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
       d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RETURN false; END;
 i:=m->'identidad'; operacion:=m->>'operacion';
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(i)<>'object'
    OR m->>'esquema'<>'vec.dietas.borrador-operacion.v2'
    OR vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) IS NOT TRUE THEN RETURN false; END IF;
 IF operacion IN ('editar','borrar','enviar') THEN
  IF m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
     OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref' THEN RETURN false; END IF;
  accion:='dietas.borrador.propio.'||operacion;
  finalidad:=operacion||'_borrador_propio';
  audiencia:='vec_dietas.borrador_propio.'||operacion||'.v1';
  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
  campos_devolucion:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 ELSIF operacion IN ('detalle','lista') THEN
  IF (operacion='detalle' AND (m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$' OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref'))
     OR (operacion='lista' AND m->>'recurso_ref'<>'dietas:borradores:propios') THEN RETURN false; END IF;
  accion:='dietas.documento.propio.consultar'; finalidad:='consultar_documento_propio_dietas';
  audiencia:='vec_dietas.documento_propio.consultar.v1';
  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
  campos_devolucion:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.devolucion","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
 ELSE RETURN false; END IF;
 IF d->>'esquema'<>'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2'
    OR d->>'concedida'<>'true' OR d->>'bloque_version'<>'3'
    OR d->>'accion' IS DISTINCT FROM accion OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'modulo_id'<>'dietas' OR d->>'tipo_recurso'<>'comision_borrador'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'esquema'<>'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
    OR c->>'operacion' IS DISTINCT FROM accion
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN RETURN false; END IF;
 v:=d->'vinculo_autenticacion_actor'; hc:=encode(sha256(p_contexto),'hex');
 IF v->>'esquema'<>'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
    OR v->>'autoridad_efectiva'<>'autoridad_maestra_acreditada'
    OR v->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR v->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR v->>'cuenta_ref' IS DISTINCT FROM i->>'cuenta_ref'
    OR v->>'contexto_actor_ref' IS DISTINCT FROM i->>'contexto_actor_ref'
    OR v->>'contexto_actor_version' IS DISTINCT FROM i->>'contexto_version'
    OR v->>'contexto_actor_cuenta_version' IS DISTINCT FROM i->>'cuenta_version'
    OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM hc
    OR c->>'contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref'
    OR c->>'huella_contexto_sha256' IS DISTINCT FROM hc
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref' THEN RETURN false; END IF;
 hasta:=coalesce(i->>'vigente_hasta',''); IF hasta='' THEN hasta:='sin_fin'; END IF;
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'","operacion":'||to_jsonb(operacion)::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||(m->'recurso_ref')::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":'||to_jsonb(hasta)::text||'}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 RETURN d->>'contexto_recurso_huella_sha256'=h AND c->>'huella_efecto_sha256'=h;
END $function$


DO $post_0$
DECLARE f oid:='vec_dietas.cotejar_recurso_dietas_borrador_v1(text,bytea,bytea,bytea)'::regprocedure; p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
  OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario}'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'790081a9e65805de73a9686a8fb7a190ecaacde8a0ea37d70b8ec2bbb22375d5'
  OR (SELECT count(*) FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f)<>2
 THEN RAISE EXCEPTION 'Dietas18: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post_0$;

DO $post_1$
DECLARE f oid:='vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea)'::regprocedure; p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
  OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario}'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'f39143b2928f329e363eee6f797d681c560afc03bf4584a128e32031d8bdad62'
  OR (SELECT count(*) FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=f)<>2
 THEN RAISE EXCEPTION 'Dietas18: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post_1$;

COMMIT;
