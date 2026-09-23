\set ON_ERROR_STOP on
-- Instalar después de roles Dietas y Personal 000007, y sólo cuando AD3-49
-- publique el consumidor/audiencia nominal de Dietas. Esta migración no amplía AD3.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000001:borrador:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.borrador_comision') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid='vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)'::regprocedure)
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_dietas_borrador_v3_atestada' AND p.pronargs=10)
    OR NOT has_schema_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3','USAGE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
   RAISE EXCEPTION 'Dietas 000001: falta Personal 000007 o consumidor AD3-49 nominal' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_dietas.borrador_comision (
 referencia text PRIMARY KEY CHECK(referencia~'^dco_[A-Za-z0-9_-]{22,128}$'),
 persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref~'^emp_[A-Za-z0-9_-]{22,128}$'),
 relacion_ref text NOT NULL CHECK(relacion_ref~'^rel_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL, relacion_version bigint NOT NULL CHECK(relacion_version>0),
 procedencia_acto_ref text NOT NULL, fuente_ref text NOT NULL, fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fecha_inicio date NOT NULL, fecha_fin date NOT NULL, motivo text NOT NULL CHECK(length(motivo) BETWEEN 3 AND 600),
 codigos_ruta jsonb NOT NULL CHECK(jsonb_typeof(codigos_ruta)='array'),
 clave_idempotencia text NOT NULL CHECK(clave_idempotencia~'^[A-Za-z0-9_-]{16,128}$'),
 huella_semantica_sha256 text NOT NULL CHECK(huella_semantica_sha256~'^[0-9a-f]{64}$'),
 version bigint NOT NULL DEFAULT 1 CHECK(version=1), creada_en timestamptz(6) NOT NULL,
 UNIQUE(persona_ref,clave_idempotencia), CHECK(fecha_inicio<=fecha_fin)
);
CREATE TABLE vec_dietas.recibo_borrador_comision (
 referencia text PRIMARY KEY CHECK(referencia~'^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
 comision_ref text NOT NULL REFERENCES vec_dietas.borrador_comision(referencia), version bigint NOT NULL CHECK(version=1),
 decision_ref text NOT NULL, efecto_ref text NOT NULL, consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL, registrada_en timestamptz(6) NOT NULL, UNIQUE(comision_ref,version)
);
CREATE TABLE vec_dietas.historia_borrador_comision (
 evento_ref text PRIMARY KEY, comision_ref text NOT NULL REFERENCES vec_dietas.borrador_comision(referencia),
 tipo text NOT NULL CHECK(tipo='borrador_creado'), recibo_ref text NOT NULL REFERENCES vec_dietas.recibo_borrador_comision(referencia),
 huella_semantica_sha256 text NOT NULL CHECK(huella_semantica_sha256~'^[0-9a-f]{64}$'), registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_dietas.auditoria_borrador_comision (
 auditoria_ref text PRIMARY KEY, comision_ref text REFERENCES vec_dietas.borrador_comision(referencia),
 recurso_ref text NOT NULL, accion text NOT NULL CHECK(accion IN ('crear','consultar')), actor_ref text NOT NULL, persona_ref text NOT NULL,
 resultado text NOT NULL CHECK(resultado IN ('concedido','no_encontrado')), correlacion_ref text NOT NULL, registrada_en timestamptz(6) NOT NULL
);
CREATE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$ BEGIN RAISE EXCEPTION 'historia Dietas inmutable' USING ERRCODE='55000'; END $$;
ALTER FUNCTION vec_dietas.rechazar_mutacion_borrador_v1() OWNER TO vec_dietas_propietario;
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['borrador_comision','recibo_borrador_comision','historia_borrador_comision','auditoria_borrador_comision'] LOOP
  EXECUTE format('ALTER TABLE vec_dietas.%I ENABLE ROW LEVEL SECURITY',t); EXECUTE format('ALTER TABLE vec_dietas.%I FORCE ROW LEVEL SECURITY',t);
  IF t='borrador_comision' THEN
    EXECUTE 'CREATE POLICY persona_contextual ON vec_dietas.borrador_comision FOR ALL TO vec_dietas_propietario USING (persona_ref=current_setting(''vec.dietas.persona_ref'',true) AND current_setting(''vec.dietas.persona_ref'',true) IS NOT NULL) WITH CHECK (persona_ref=current_setting(''vec.dietas.persona_ref'',true) AND current_setting(''vec.dietas.persona_ref'',true) IS NOT NULL)';
  ELSIF t IN ('recibo_borrador_comision','historia_borrador_comision') THEN
    EXECUTE format('CREATE POLICY persona_contextual ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=%I AND b.persona_ref=current_setting(''vec.dietas.persona_ref'',true))) WITH CHECK (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=%I AND b.persona_ref=current_setting(''vec.dietas.persona_ref'',true)))',t,'comision_ref','comision_ref');
  ELSE
    EXECUTE 'CREATE POLICY persona_contextual ON vec_dietas.auditoria_borrador_comision FOR ALL TO vec_dietas_propietario USING (persona_ref=current_setting(''vec.dietas.persona_ref'',true) AND current_setting(''vec.dietas.persona_ref'',true) IS NOT NULL) WITH CHECK (persona_ref=current_setting(''vec.dietas.persona_ref'',true) AND current_setting(''vec.dietas.persona_ref'',true) IS NOT NULL)';
  END IF;
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.%I FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',t);
 END LOOP;
END $$;

CREATE FUNCTION vec_dietas.consumir_ad3_borrador_v1(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $$
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN
   RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501';
 END IF;
 RETURN QUERY EXECUTE 'SELECT * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)'
 USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
END $$;

-- Mismo canon que RecursoAutorizable.HuellaContextoAutorizacionSHA256: JSON
-- sin espacios, claves ordenadas y valores string. Material llega del borde
-- confiable; su SHA se liga como atributo, nunca se toma del cliente.
CREATE FUNCTION vec_dietas.cotejar_contexto_dietas_borrador_v1(p_identidad jsonb,p_contexto bytea) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE x jsonb; n integer; claves text[]:=ARRAY['contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','esquema','estado','garantia','metodo','perfil_activo_ref','perfil_version','persona_ref','persona_version','principal_ref','resuelto_en','vigente_desde','vigente_hasta','vinculos'];
BEGIN
 IF p_identidad IS NULL OR p_contexto IS NULL THEN RETURN false; END IF;
 BEGIN x:=convert_from(p_contexto,'UTF8')::jsonb; EXCEPTION WHEN others THEN RETURN false; END;
 IF jsonb_typeof(x)<>'object' OR ARRAY(SELECT jsonb_object_keys(x) ORDER BY 1) IS DISTINCT FROM claves
    OR x->>'esquema'<>'vec.contexto-actor.vinculado.v2' OR x->>'estado'<>'activo' OR jsonb_typeof(x->'vinculos')<>'array'
    OR x->>'principal_ref' IS DISTINCT FROM x->>'persona_ref'
    OR x->>'principal_ref' IS DISTINCT FROM p_identidad->>'actor_ref' OR x->>'perfil_activo_ref' IS DISTINCT FROM p_identidad->>'perfil_ref'
    OR x->>'persona_ref' IS DISTINCT FROM p_identidad->>'persona_ref' OR x->>'contexto_actor_ref' IS DISTINCT FROM p_identidad->>'contexto_actor_ref'
    OR x->>'contexto_version' IS DISTINCT FROM p_identidad->>'contexto_version' OR x->>'cuenta_ref' IS DISTINCT FROM p_identidad->>'cuenta_ref'
    OR x->>'cuenta_version' IS DISTINCT FROM p_identidad->>'cuenta_version' OR x->>'persona_version' IS DISTINCT FROM p_identidad->>'persona_version'
    OR x->>'perfil_version' IS DISTINCT FROM p_identidad->>'perfil_version' OR coalesce((x->>'contexto_version')::numeric,0)<=0
    OR coalesce((x->>'cuenta_version')::numeric,0)<=0 OR coalesce((x->>'persona_version')::numeric,0)<=0 OR coalesce((x->>'perfil_version')::numeric,0)<=0
    OR x->>'vigente_desde' IS NULL OR x->>'vigente_hasta' IS NULL OR x->>'resuelto_en' IS NULL THEN RETURN false; END IF;
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') AS elemento(valor)
  WHERE jsonb_typeof(elemento.valor)='object' AND ARRAY(SELECT jsonb_object_keys(elemento.valor) ORDER BY 1)=ARRAY['estado','referencia','tipo','version','vigente_desde','vigente_hasta','vinculo_ref']
    AND elemento.valor->>'tipo'='empleado' AND elemento.valor->>'referencia'=p_identidad->>'empleado_ref' AND elemento.valor->>'estado'='activo'
    AND coalesce((elemento.valor->>'version')::numeric,0)>0 AND elemento.valor->>'vigente_desde' IS NOT NULL AND elemento.valor->>'vigente_hasta' IS NOT NULL;
 RETURN n=1;
END $$;

CREATE FUNCTION vec_dietas.cotejar_recurso_dietas_borrador_v1(p_material text,p_capacidad bytea,p_decision bytea,p_contexto bytea) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$
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
   WHEN 'crear' THEN '["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'detalle' THEN '["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'lista' THEN '["items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.registrado_en","items.recibo.referencia","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb
   ELSE NULL END;
 campos_calculo:=CASE m->>'operacion'
   WHEN 'crear' THEN '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'detalle' THEN '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb
   WHEN 'lista' THEN '["items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.registrado_en","items.recibo.referencia","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb END;
 IF campos_esperados IS NULL OR (d->'campos_permitidos' IS DISTINCT FROM campos_esperados AND d->'campos_permitidos' IS DISTINCT FROM campos_calculo) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RETURN false; END IF;
 v:=d->'vinculo_autenticacion_actor'; hc:=encode(sha256(p_contexto),'hex');
 IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM claves_decision
    OR jsonb_typeof(v)<>'object' OR ARRAY(SELECT jsonb_object_keys(v) ORDER BY 1) IS DISTINCT FROM claves_vinculo
    OR ARRAY(SELECT jsonb_object_keys(c) ORDER BY 1) IS DISTINCT FROM claves_capacidad
    OR d->>'esquema'<>'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2' OR d->>'bloque_version'<>'3'
    OR d->>'concedida'<>'true' OR v->>'esquema'<>'vec.autorizacion.vinculo-autenticacion-actor.v2' OR v->>'bloque_version'<>'2'
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
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||',"relacion_ref":'||(i->'relacion_ref')::text||',"unidad_ref":'||(i->'unidad_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'","operacion":'||to_jsonb(m->>'operacion')::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||(m->'recurso_ref')::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":'||to_jsonb(hasta)::text||'}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 RETURN d->>'accion'=accion AND d->>'finalidad'=finalidad AND d->>'modulo_id'='dietas' AND d->>'tipo_recurso'='comision_borrador' AND d->>'recurso_ref'=m->>'recurso_ref' AND d->>'contexto_recurso_huella_sha256'=h AND c->>'operacion'=accion AND c->>'efecto_ref'=m->>'recurso_ref' AND c->>'huella_efecto_sha256'=h;
END $$;

-- encoding/json escapa HTML y U+2028/U+2029. jsonb::text no lo hace,
-- por eso no puede usarse para el material de replay compartido con Go.
CREATE FUNCTION vec_dietas.cadena_json_go_v1(p_valor text) RETURNS text
LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v text;
BEGIN
 IF p_valor IS NULL THEN RAISE EXCEPTION 'cadena canónica ausente' USING ERRCODE='22023'; END IF;
 v:=to_json(p_valor)::text;
 v:=replace(v,'&',chr(92)||'u0026');
 v:=replace(v,'<',chr(92)||'u003c');
 v:=replace(v,'>',chr(92)||'u003e');
 v:=replace(v,chr(8232),chr(92)||'u2028');
 v:=replace(v,chr(8233),chr(92)||'u2029');
 RETURN v;
END $$;

-- Canon de replaySemanticoCrearBorradorV1. El material declara una huella,
-- pero nunca se confía en ella: esta función la reconstruye desde todos los
-- campos que determinan el efecto antes de usar una clave idempotente.
CREATE FUNCTION vec_dietas.huella_semantica_crear_borrador_v1(p_material text) RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE m jsonb; i jsonb; c jsonb; canon text; rutas text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER') THEN
   RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501';
 END IF;
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando'; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 IF i IS NULL OR c IS NULL OR jsonb_typeof(c->'codigos_ruta')<>'array' OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'codigos_ruta') e WHERE jsonb_typeof(e)<>'string') THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END IF;
 SELECT coalesce(string_agg(vec_dietas.cadena_json_go_v1(valor),',' ORDER BY ordinalidad),'') INTO rutas FROM jsonb_array_elements_text(c->'codigos_ruta') WITH ORDINALITY r(valor,ordinalidad);
 canon:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')||',"empleado_ref":'||vec_dietas.cadena_json_go_v1(i->>'empleado_ref')||',"relacion_ref":'||vec_dietas.cadena_json_go_v1(i->>'relacion_ref')||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(i->>'unidad_ref')||',"relacion_version":'||(i->'relacion_version')::text||',"vigente_desde":'||vec_dietas.cadena_json_go_v1(i->>'vigente_desde')||',"vigente_hasta":'||vec_dietas.cadena_json_go_v1(i->>'vigente_hasta')||',"procedencia_acto_ref":'||vec_dietas.cadena_json_go_v1(i->>'procedencia_acto_ref')||',"fuente_ref":'||vec_dietas.cadena_json_go_v1(i->>'fuente_ref')||',"fuente_version":'||(i->'fuente_version')::text||',"clave_idempotencia":'||vec_dietas.cadena_json_go_v1(c->>'clave_idempotencia')||',"fecha_inicio":'||vec_dietas.cadena_json_go_v1(c->>'fecha_inicio')||',"fecha_fin":'||vec_dietas.cadena_json_go_v1(c->>'fecha_fin')||',"motivo":'||vec_dietas.cadena_json_go_v1(c->>'motivo')||',"codigos_ruta":['||rutas||']}';
 IF c ? 'hora_inicio' THEN canon:=left(canon,length(canon)-1)||',"hora_inicio":'||vec_dietas.cadena_json_go_v1(c->>'hora_inicio')||'}'; END IF;
 IF c ? 'hora_fin' THEN canon:=left(canon,length(canon)-1)||',"hora_fin":'||vec_dietas.cadena_json_go_v1(c->>'hora_fin')||'}'; END IF;
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
END $$;

CREATE FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; c jsonb; cap jsonb; ctx jsonb; d jsonb; v record; b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.recibo_borrador_comision%ROWTYPE; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); nuevo boolean:=false; ref text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' OR p_material IS NULL THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; c:=m->'comando';
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1' OR m->>'operacion'<>'crear' OR m->>'recurso_ref'<>'dietas:borradores:propios' OR m->>'huella_semantica' !~ '^[0-9a-f]{64}$' OR i IS NULL OR c IS NULL OR i->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR i->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$' OR i->>'relacion_ref' !~ '^rel_[A-Za-z0-9_-]{22,128}$' OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$' OR c->>'fecha_inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR c->>'fecha_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR jsonb_typeof(c->'codigos_ruta')<>'array' THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END IF;
 IF vec_dietas.cotejar_recurso_dietas_borrador_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE OR m->>'huella_semantica' IS DISTINCT FROM vec_dietas.huella_semantica_crear_borrador_v1(p_material) THEN RAISE EXCEPTION 'preimagen RecursoAutorizable Dietas no acreditada por AD3' USING ERRCODE='PD003'; END IF;
 IF p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric THEN RAISE EXCEPTION 'versiones ContextoActor Dietas incoherentes' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_dietas.consumir_ad3_borrador_v1(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref' OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref' OR cap->>'huella_efecto_sha256' IS DISTINCT FROM v.huella_efecto_sha256 OR NOT vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) THEN RAISE EXCEPTION 'AD3 no ligado a contexto/decision/efecto/material Dietas' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);
 -- AD3 se liga antes de mirar Personal; si el sello no cuadra, el rollback
 -- deshace también el consumo y no transforma Personal en oráculo.
 PERFORM vec_personal.revalidar_relacion_dietas_v1(i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),(i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',(i->>'fuente_version')::bigint,(i->>'fecha_referencia')::date);
 SELECT * INTO b FROM vec_dietas.borrador_comision WHERE persona_ref=i->>'persona_ref' AND clave_idempotencia=c->>'clave_idempotencia' FOR UPDATE;
 IF FOUND THEN
   IF b.huella_semantica_sha256<>m->>'huella_semantica' THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
   SELECT * INTO STRICT r FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=b.referencia; nuevo:=false;
 ELSE
   IF v.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'replay AD3 sin recibo Dietas reconciliable' USING ERRCODE='PD003'; END IF;
   ref:='dco_'||md5(clock_timestamp()::text||random()::text||(i->>'persona_ref'));
   INSERT INTO vec_dietas.borrador_comision VALUES(ref,i->>'persona_ref',i->>'empleado_ref',i->>'relacion_ref',i->>'unidad_ref',(i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',(i->>'fuente_version')::bigint,(c->>'fecha_inicio')::date,(c->>'fecha_fin')::date,c->>'motivo',c->'codigos_ruta',c->>'clave_idempotencia',m->>'huella_semantica',1,ahora) RETURNING * INTO b;
   INSERT INTO vec_dietas.recibo_borrador_comision VALUES('rcd_'||substr(md5(ref),1,8)||'-'||substr(md5(ref),9,4)||'-'||substr(md5(ref),13,4)||'-'||substr(md5(ref),17,4)||'-'||substr(md5(ref),21,12),ref,1,v.decision_ref,v.efecto_ref,v.consumo_huella_sha256,v.auditoria_ref,ahora) RETURNING * INTO r;
   INSERT INTO vec_dietas.historia_borrador_comision VALUES('hdi_'||md5(ref||'historia'),ref,'borrador_creado',r.referencia,b.huella_semantica_sha256,ahora);
   nuevo:=true;
 END IF;
 INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||b.referencia||ahora::text),b.referencia,'dietas:borradores:propios','crear',d->>'principal_id',i->>'persona_ref','concedido',d->>'correlacion_ref',ahora);
 RETURN jsonb_build_object('resultado','concedido','comision',jsonb_build_object('referencia',b.referencia,'estado','borrador','fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref),'recibo',jsonb_build_object('referencia',r.referencia,'version',1,'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'repeticion',NOT nuevo));
END $$;

CREATE FUNCTION vec_dietas.consultar_borradores_propios_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; q jsonb; cap jsonb; ctx jsonb; d jsonb; v record; b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.recibo_borrador_comision%ROWTYPE; items jsonb:='[]'::jsonb; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); siguiente text:=''; vistos int:=0;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END; i:=m->'identidad'; q:=m->'consulta';
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1' OR m->>'operacion' NOT IN ('detalle','lista') OR i IS NULL OR i->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR (m->>'operacion'='detalle' AND m->>'referencia' !~ '^dco_[A-Za-z0-9_-]{22,128}$') OR (m->>'operacion'='lista' AND (q IS NULL OR coalesce((q->>'limite')::int,0) NOT BETWEEN 1 AND 50 OR (coalesce(q->>'cursor','')<>'' AND q->>'cursor' !~ '^dco_[A-Za-z0-9_-]{22,128}$'))) THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END IF;
 IF vec_dietas.cotejar_recurso_dietas_borrador_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE THEN RAISE EXCEPTION 'preimagen RecursoAutorizable Dietas no acreditada por AD3' USING ERRCODE='PD003'; END IF;
 IF p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric THEN RAISE EXCEPTION 'versiones ContextoActor Dietas incoherentes' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_dietas.consumir_ad3_borrador_v1(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref' OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref' OR cap->>'huella_efecto_sha256' IS DISTINCT FROM v.huella_efecto_sha256 OR NOT vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) THEN RAISE EXCEPTION 'AD3 no ligado a contexto/decision/efecto/material Dietas' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);
 PERFORM vec_personal.revalidar_relacion_dietas_v1(i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),(i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',(i->>'fuente_version')::bigint,(i->>'fecha_referencia')::date);
 IF m->>'operacion'='detalle' THEN SELECT * INTO b FROM vec_dietas.borrador_comision WHERE referencia=m->>'referencia' AND persona_ref=i->>'persona_ref' AND empleado_ref=i->>'empleado_ref' AND relacion_ref=i->>'relacion_ref' AND unidad_ref=i->>'unidad_ref'; IF NOT FOUND THEN INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||(m->>'referencia')||ahora::text),NULL,m->>'referencia','consultar',d->>'principal_id',i->>'persona_ref','no_encontrado',d->>'correlacion_ref',ahora); RETURN jsonb_build_object('resultado','no_encontrado'); END IF; SELECT * INTO STRICT r FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=b.referencia; INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||b.referencia||ahora::text),b.referencia,b.referencia,'consultar',d->>'principal_id',i->>'persona_ref','concedido',d->>'correlacion_ref',ahora); RETURN jsonb_build_object('resultado','concedido','comision',jsonb_build_object('referencia',b.referencia,'estado','borrador','fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref),'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'repeticion',true)); END IF;
 FOR b IN SELECT * FROM vec_dietas.borrador_comision WHERE persona_ref=i->>'persona_ref' AND empleado_ref=i->>'empleado_ref' AND relacion_ref=i->>'relacion_ref' AND unidad_ref=i->>'unidad_ref' AND (coalesce(q->>'cursor','')='' OR referencia>q->>'cursor') ORDER BY referencia LIMIT ((q->>'limite')::int+1) LOOP vistos:=vistos+1; IF vistos>(q->>'limite')::int THEN EXIT; END IF; siguiente:=b.referencia; SELECT * INTO STRICT r FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=b.referencia; items:=items||jsonb_build_array(jsonb_build_object('resultado','concedido','comision',jsonb_build_object('referencia',b.referencia,'estado','borrador','fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref),'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'repeticion',true))); END LOOP;
 IF vistos <= (q->>'limite')::int THEN siguiente:=''; END IF;
 INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||'dietas:borradores:propios'||ahora::text),NULL,'dietas:borradores:propios','consultar',d->>'principal_id',i->>'persona_ref','concedido',d->>'correlacion_ref',ahora);
 RETURN jsonb_build_object('items',items,'siguiente_cursor',siguiente);
END $$;
ALTER FUNCTION vec_dietas.consumir_ad3_borrador_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.cadena_json_go_v1(text) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.huella_semantica_crear_borrador_v1(text) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.cotejar_contexto_dietas_borrador_v1(jsonb,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.cotejar_recurso_dietas_borrador_v1(text,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_dietas FROM PUBLIC,vec_dietas_ejecutor;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_dietas FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_ejecutor;
COMMIT;
