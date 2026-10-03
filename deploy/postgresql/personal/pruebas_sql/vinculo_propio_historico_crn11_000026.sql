\set ON_ERROR_STOP on
-- Fixture nominal sobre la preimagen de la copia fría principal.
-- Infraestructura de ensayo exclusiva del clon privado. Autoridades y datos
-- sintéticos propios: no acreditan IdP, fuente institucional ni política RRHH.
-- El material positivo atraviesa COSE EdDSA y HMAC Go, registro durable V3 y
-- consumidor nominal SQL. Las funciones de este archivo nunca sustituyen al núcleo.
SET search_path=pg_catalog;
SET timezone='UTC';
CREATE TABLE public.crn11_ensayo_vector (
 caso text PRIMARY KEY, material text, entrada jsonb, bundle jsonb, confirmacion jsonb
);
REVOKE ALL ON TABLE public.crn11_ensayo_vector FROM PUBLIC;
CREATE FUNCTION public.crn11_ensayo_actor(p text) RETURNS void LANGUAGE plpgsql
SET search_path=pg_catalog AS $f$
DECLARE s text:=substr(encode(sha256(convert_to(p,'UTF8')),'hex'),1,32);
 t timestamptz(6):=clock_timestamp()-interval '2 minutes'; h timestamptz(6):=clock_timestamp()+interval '2 hours';
BEGIN
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES ('prc_crn11_'||s,1,repeat('a',64),'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_crn11_'||s,1,'prc_crn11_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('cta_crn11_'||s,1);
 INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_crn11_'||s,1,'prc_crn11_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('per_crn11_'||s,1);
 INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_crn11_'||s,1,'per_crn11_'||s,'prc_crn11_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ('prf_crn11_'||s,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_crn11_'||s,1,'cta_crn11_'||s,'prf_crn11_'||s,'per_crn11_'||s,'prc_crn11_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('vca_crn11_'||s,1);
 PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_crn11_'||s,1,'per_crn11_'||s,'emp_crn11_'||s,'activa',t,h,NULL,'prc_crn11_'||s,1,repeat('a',64));
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1(
 sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,
 superficie,metodo_observado,garantia_observada,politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en)
 VALUES ('ses_crn11_'||s,'aut_crn11_'||s,repeat('b',64),'ase_crn11_'||s,'cta_crn11_'||s,'cta_crn11_'||s,false,
 'interna_corporativa','certificado','alto','pga_crn11_'||s,repeat('c',64),t,t);
 INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,sesion_revalidada_en,sesion_valida_hasta)
 VALUES ('cse_crn11_'||s,1,'ses_crn11_'||s,'activa',repeat('d',64),t,h);
 INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES ('ses_crn11_'||s,'cse_crn11_'||s,1,t,'acto:crn11:sesion:'||s);
 INSERT INTO public.crn11_ensayo_vector(caso) VALUES(p);
END $f$;
REVOKE ALL ON FUNCTION public.crn11_ensayo_actor(text) FROM PUBLIC;

CREATE FUNCTION public.crn11_ensayo_entrada(p text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE s text:=substr(encode(sha256(convert_to(p,'UTF8')),'hex'),1,32); r record; d jsonb; v jsonb; m text;
 k record; cat record; politicas jsonb; seq numeric; raiz numeric; ar jsonb;
BEGIN
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_crn11_'||s;
 d:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 -- Trece claves y orden idénticos al canon Go; sin texto reconstruido por jsonb.
 m:='{"esquema":"vec.personal.vinculo-propio-crn11.consulta.v1","empleado_ref":"emp_crn11_'||s||'","actor_ref":"per_crn11_'||s||'","contexto_actor_ref":'||to_jsonb(d->>'contexto_actor_ref')::text||',"contexto_version":'||(d->>'contexto_version')||',"cuenta_ref":"cta_crn11_'||s||'","cuenta_version":'||(d->>'cuenta_version')||',"perfil_ref":"prf_crn11_'||s||'","perfil_version":'||(d->>'perfil_version')||',"persona_ref":"per_crn11_'||s||'","persona_version":'||(d->>'persona_version')||',"vinculo_ref":"pep_crn11_'||s||'","vinculo_version":1}';
 SELECT * INTO STRICT k FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:crn11:ensayo';
 SELECT * INTO STRICT cat FROM vec_autorizacion.control_catalogo_politicas WHERE control_id;
 SELECT coalesce(jsonb_agg(p.documento ORDER BY p.politica_ref COLLATE "C"),'[]'::jsonb) INTO politicas
 FROM vec_autorizacion.politica_restrictiva_actual a JOIN vec_autorizacion.politica_restrictiva p ON p.politica_id=a.politica_id AND p.politica_ref=a.politica_ref;
 SELECT greatest(configuracion_secuencia_minima,coalesce((SELECT max(secuencia) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),0))+1,
 greatest(raiz_version_minima,coalesce((SELECT max(version) FROM vec_autorizacion_atestada_v3.raiz_confianza_version),0))+1 INTO seq,raiz
 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id;
 v:=jsonb_build_object('esquema','vec.autenticacion-actor.vinculo.v2.contexto-registrado','bloque_version',2,
 'autenticacion_ref','aut_crn11_'||s,'autenticacion_huella_sha256',repeat('b',64),'asercion_ref','ase_crn11_'||s,
 'sesion_ref','ses_crn11_'||s,'control_sesion_ref','cse_crn11_'||s,'control_sesion_revision',1,'control_sesion_huella_sha256',repeat('d',64),
 'cuenta_ref','cta_crn11_'||s,'cuenta_ordinaria_ref','cta_crn11_'||s,'principal_id','per_crn11_'||s,'perfil_activo_ref','prf_crn11_'||s,
 'cuenta_privilegiada',false,'superficie','interna_corporativa','metodo_observado','certificado','garantia_observada','alto',
 'politica_garantia_ref','pga_crn11_'||s,'politica_garantia_huella_sha256',repeat('c',64),
 'registro_contexto_ref',r.registro_contexto_ref,'contexto_actor_esquema',d->>'esquema','contexto_actor_ref',d->>'contexto_actor_ref',
 'contexto_actor_version',(d->>'contexto_version')::numeric,'contexto_actor_cuenta_version',(d->>'cuenta_version')::numeric,
 'contexto_actor_huella_sha256',r.huella_sha256,'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
 'autoridad_efectiva',r.autoridad_efectiva);
 SELECT v||jsonb_build_object('autenticacion_verificada_en',a.autenticacion_verificada_en,'sesion_emitida_en',a.sesion_emitida_en,
 'sesion_valida_hasta',c.sesion_valida_hasta,'sesion_revalidada_en',c.sesion_revalidada_en) INTO v
 FROM vec_autorizacion.sesion_autenticacion_v1 a JOIN vec_autorizacion.control_sesion_v1 c ON c.sesion_ref=a.sesion_ref WHERE a.sesion_ref='ses_crn11_'||s;
 ar:=jsonb_build_object('caso',p,'ahora',clock_timestamp(),'decision_plantilla_b64',encode(convert_to(jsonb_build_object(
 'decision_ref','decision:crn11:'||s,'accion','personal.vinculo_propio.crn11.consultar','recurso_ref','emp_crn11_'||s,
 'modulo_id','personal','tipo_recurso','vinculo_historico_propio_crn11','finalidad','acreditar_vinculo_historico_propio_crn11',
 'correlacion_ref','correlacion_'||s,'vinculo_autenticacion_actor',v)::text,'UTF8'),'base64'),
 'motivo_b64',encode(vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb_build_object('esquema','vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
 'referencia',jsonb_build_object('catalogo_id','motivos_crn11_ensayo','catalogo_version',1,'catalogo_huella_sha256',repeat('e',64),'entrada_clave','motivo_11111111111111111111111111111111'))),'base64'),
 'contexto_b64',encode(r.representacion_canonica,'base64'),'manifiesto_b64',encode(r.manifiesto_procedencia_canonico,'base64'),
 'manifiesto_huella_sha256',r.manifiesto_procedencia_huella_sha256,'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en,
 'alta_b64',encode(convert_to('{}','UTF8'),'base64'),'sellos_b64',encode(convert_to('{}','UTF8'),'base64'),'efecto_huella_sha256',encode(sha256(convert_to(m,'UTF8')),'hex'),
 'clave_id',k.clave_id,'clave_version',k.version,'revision_gobierno',k.revision_gobierno,'huella_gobierno_sha256',k.huella_gobierno_sha256,
 'emisor_id',k.emisor_id,'audiencia_consumo',k.audiencia_consumo,'clave_hmac_b64','','clave_valida_desde',k.valida_desde,'clave_valida_hasta',k.valida_hasta,
 'revision_confianza','configuracion:crn11:'||s,'secuencia_confianza',seq,'raiz_clave_id','raiz:crn11:'||s,'raiz_version',raiz,
 'audiencia_despliegue','vec-diputacion/pruebas/crn11/consumidor','politicas',politicas,'revision_catalogo',cat.revision,'huella_catalogo_sha256',cat.huella_sha256,
 'asignacion_id','crn11_'||s,'asignacion_version',1,'persona_version',(d->>'persona_version')::numeric,'perfil_version',(d->>'perfil_version')::numeric,
 'empleado_ref','emp_crn11_'||s,'material_canonico_b64',encode(convert_to(m,'UTF8'),'base64'));
 UPDATE public.crn11_ensayo_vector SET material=m,entrada=ar WHERE caso=p;
 RETURN ar;
END $f$;
REVOKE ALL ON FUNCTION public.crn11_ensayo_entrada(text) FROM PUBLIC;

CREATE FUNCTION public.crn11_ensayo_preparado(p text,b jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE db bytea:=decode(b->>'decision_b64','base64'); d jsonb:=convert_from(db,'UTF8')::jsonb;
 rol jsonb:=b->'version_rol_documento'; c jsonb:=b->'control_rol_documento'; a jsonb:=b->'asignacion_documento'; r record; conf jsonb;
BEGIN
 IF d->>'accion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
 OR d->'campos_permitidos' IS DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'ensayo: preparación no nominal'; END IF;
 INSERT INTO vec_autorizacion.version_rol VALUES (d->>'version_rol_ref',rol->>'rol_id',(rol->>'version')::numeric,d->>'version_rol_huella_sha256',(rol->>'publicada_en')::timestamptz,rol);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol VALUES (c->>'version_rol_ref',(c->>'revision')::numeric,c->>'estado',d->>'control_vigencia_version_rol_huella_sha256',(c->>'actualizado_en')::timestamptz,c);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES (c->>'version_rol_ref',(c->>'revision')::numeric,clock_timestamp(),'autoridad-crn11-sintetica','acto:crn11:control:'||p);
 INSERT INTO vec_autorizacion.asignacion_perfil VALUES (d->>'asignacion_ref',a->>'asignacion_id',(a->>'version')::numeric,a->>'perfil_activo_ref',a->>'principal_id',a->>'version_rol_ref',d->>'asignacion_huella_sha256',(a->>'emitida_en')::timestamptz,a);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES (a->>'perfil_activo_ref',d->>'asignacion_ref',clock_timestamp(),'autoridad-crn11-sintetica','acto:crn11:asignacion:'||p);
 SELECT * INTO STRICT r FROM vec_autorizacion.registrar_decision_contexto_actor_v3(db,decode(b->>'motivo_b64','base64'),(b->>'persona_version')::numeric,(b->>'perfil_version')::numeric);
 IF r.concedida IS NOT TRUE THEN RAISE EXCEPTION 'ensayo: registro durable V3 no concedido'; END IF;
 conf:=jsonb_build_object('decision_ref',d->>'decision_ref','decision_huella_sha256',r.decision_huella_sha256,'emitida_en',d->>'emitida_en','valida_hasta',d->>'valida_hasta','registrada_en',r.registrada_en);
 UPDATE public.crn11_ensayo_vector SET confirmacion=conf WHERE caso=p;
 RETURN conf;
END $f$;
REVOKE ALL ON FUNCTION public.crn11_ensayo_preparado(text,jsonb) FROM PUBLIC;

CREATE FUNCTION public.crn11_ensayo_firmado(p text,b jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE cap bytea:=decode(b->>'capacidad_b64','base64'); c jsonb:=convert_from(cap,'UTF8')::jsonb; spki bytea:=decode(b->>'spki_b64','base64'); ord numeric;
BEGIN
 IF vec_autorizacion_atestada_v3.capacidad_canonica(c) IS DISTINCT FROM cap
 OR c->>'operacion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
 OR octet_length(decode(b->>'cose_b64','base64'))<64 THEN RAISE EXCEPTION 'ensayo: bundle Go no nominal'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(c->>'revision_confianza',(c->>'configuracion_secuencia')::numeric,c->>'huella_configuracion_sha256',(c->>'configuracion_publicada_en')::timestamptz,(c->>'configuracion_expira_en')::timestamptz,'acto:crn11:configuracion:'||p);
 INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
 VALUES(c->>'raiz_clave_id',(c->>'raiz_version')::numeric,spki,encode(sha256(spki),'hex'),(c->>'raiz_valida_desde')::timestamptz,(c->>'raiz_valida_hasta')::timestamptz,c->>'suite',c->>'audiencia_despliegue','acto:crn11:raiz:'||p);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES(c->>'revision_confianza',c->>'raiz_clave_id',(c->>'raiz_version')::numeric);
 SELECT coalesce(max(orden),0)+1 INTO ord FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual;
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref) VALUES(ord,c->>'revision_confianza',clock_timestamp(),'acto:crn11:puntero:'||p);
 UPDATE public.crn11_ensayo_vector SET bundle=b WHERE caso=p;
END $f$;
REVOKE ALL ON FUNCTION public.crn11_ensayo_firmado(text,jsonb) FROM PUBLIC;

-- Invoker: session_user sigue siendo el LOGIN nominal del ensayo, jamás el
-- propietario. Esta envoltura transporta bytes al consumidor real sin decidir.
CREATE FUNCTION public.crn11_ensayo_consultar(p text, variante text DEFAULT 'valida') RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b jsonb; m text; d bytea; j jsonb;
BEGIN
 SELECT material,bundle INTO STRICT m,b FROM public.crn11_ensayo_vector WHERE caso=p;
 d:=decode(b->>'decision_b64','base64');
 IF variante='actor' THEN m:=replace(m,'"actor_ref":"per_crn11_','"actor_ref":"per_ajena_');
 ELSIF variante='ambito' THEN m:=replace(m,'"empleado_ref":"emp_crn11_','"empleado_ref":"emp_ajeno_');
 ELSIF variante='campos' THEN
 j:=convert_from(d,'UTF8')::jsonb; j:=jsonb_set(j,'{campos_permitidos}','["persona_ref"]'::jsonb);
 d:=convert_to(j::text,'UTF8');
 ELSIF variante='cose' THEN b:=jsonb_set(b,'{cose_b64}',to_jsonb(encode(set_byte(decode(b->>'cose_b64','base64'),10,255),'base64')));
 ELSIF variante<>'valida' THEN RAISE EXCEPTION 'ensayo: variante desconocida'; END IF;
 RETURN vec_personal.consultar_vinculo_propio_historico_crn11_v1(m,
 decode(b->>'capacidad_b64','base64'),d,decode(b->>'motivo_b64','base64'),decode(b->>'contexto_b64','base64'),
 (b->>'persona_version')::numeric,(b->>'perfil_version')::numeric,decode(b->>'payload_b64','base64'),
 decode(b->>'cose_b64','base64'),decode(b->>'evidencia_b64','base64'),decode(b->>'spki_b64','base64'));
END $f$;
REVOKE ALL ON FUNCTION public.crn11_ensayo_consultar(text,text) FROM PUBLIC;
