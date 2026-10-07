\set ON_ERROR_STOP on
-- Fixture RPT27 exclusiva del clon autorizado: no instalar en principal.
-- Activación únicamente por Dirección tras fuente congelada, dos GO y preimagen.
-- Fixture nominal sobre la captura causal POST149 del clon principal.
-- Conserva las extensiones presentes, sin exigir Méritos ni Baremo ausentes.
-- Infraestructura de ensayo exclusiva del clon privado. Autoridades y datos
-- sintéticos propios: no acreditan IdP, fuente institucional ni política RRHH.
-- El material positivo atraviesa COSE EdDSA y HMAC Go, registro durable V3 y
-- consumidor nominal SQL. Las funciones de este archivo nunca sustituyen al núcleo.
SET search_path=pg_catalog;
SET timezone='UTC';
CREATE TABLE public.rpt27_ensayo_vector (
 caso text PRIMARY KEY, material text, entrada jsonb, bundle jsonb, confirmacion jsonb,
 empleado_ref text, relacion_ref text, organismo_ref text, version_esperada bigint,
 vigente_en date, conocido_en timestamptz(6)
);
REVOKE ALL ON TABLE public.rpt27_ensayo_vector FROM PUBLIC;
CREATE FUNCTION public.rpt27_ensayo_actor(p text) RETURNS void LANGUAGE plpgsql
SET search_path=pg_catalog AS $f$
DECLARE s text:=substr(encode(sha256(convert_to(p,'UTF8')),'hex'),1,32);
 t timestamptz(6):=date_trunc('microseconds',clock_timestamp()-interval '5 seconds');
 h timestamptz(6):=t+interval '5 minutes'; control_sha text; autenticacion_sha text;
BEGIN
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES ('prc_rpt27_'||s,1,repeat('a',64),'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_rpt27_'||s,1,'prc_rpt27_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('cta_rpt27_'||s,1);
 INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_rpt27_'||s,1,'prc_rpt27_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('per_rpt27_'||s,1);
 INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_rpt27_'||s,1,'per_rpt27_'||s,'prc_rpt27_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ('prf_rpt27_'||s,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_rpt27_'||s,1,'cta_rpt27_'||s,'prf_rpt27_'||s,'per_rpt27_'||s,'prc_rpt27_'||s,1,repeat('a',64),'autoridad_maestra_acreditada','activo',t,h);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('vca_rpt27_'||s,1);
 PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_rpt27_'||s,1,'per_rpt27_'||s,'emp_rpt27_'||s,'activa',t,h,NULL,'prc_rpt27_'||s,1,repeat('a',64));
 autenticacion_sha:=encode(sha256(convert_to('rpt27:autenticacion:'||s,'UTF8')),'hex');
 control_sha:=vec_identidad_sesiones_v1.huella_control_sesion_v1('cse_rpt27_'||s,1,'ses_rpt27_'||s,'activa',t,h,'opr_rpt27_'||s);
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1(
 sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,
 superficie,metodo_observado,garantia_observada,politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en)
 VALUES ('ses_rpt27_'||s,'aut_rpt27_'||s,autenticacion_sha,'ase_rpt27_'||s,'cta_rpt27_'||s,'cta_rpt27_'||s,false,
 'interna_corporativa','certificado','alto','pga_rpt27_'||s,repeat('c',64),t,t);
 INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,sesion_revalidada_en,sesion_valida_hasta)
 VALUES ('cse_rpt27_'||s,1,'ses_rpt27_'||s,'activa',control_sha,t,h);
 INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES ('ses_rpt27_'||s,'cse_rpt27_'||s,1,t,'acto:rpt27:sesion:'||s);
 -- Historia sintética nominal: IS13 coteja el consumo ORIGINAL, no una
 -- sesión autodeclarada ni su puntero actual. No acredita IdP real.
 INSERT INTO vec_identidad_sesiones_v1.cuenta
 VALUES('cta_rpt27_'||s,false,NULL,t,'opr_rpt27_cuenta_'||s);
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta
 VALUES('cta_rpt27_'||s,1,'activa',t,'opr_rpt27_estado_'||s);
 INSERT INTO vec_identidad_sesiones_v1.consumo_asercion
 (operacion_ref,esquema_hmac,dominio_hmac_ref,clave_hmac_id,clave_hmac_version,
 asercion_id_hmac,sesion_id_hmac,sujeto_id_hmac,cuenta_id_hmac,cuenta_ordinaria_id_hmac,
 autenticacion_ref,autenticacion_huella_sha256,asercion_ref,sesion_ref,
 control_sesion_ref,control_sesion_revision,cuenta_ref,cuenta_revision,
 cuenta_ordinaria_ref,cuenta_ordinaria_revision,consumida_en)
 VALUES('opr_rpt27_'||s,'vec.identidad.hmac-sha256.v1','idh_rpt27_'||s,'fixture:rpt27:identidad',1,
 sha256(convert_to('asercion:'||s,'UTF8')),sha256(convert_to('sesion:'||s,'UTF8')),
 sha256(convert_to('sujeto:'||s,'UTF8')),sha256(convert_to('cuenta:'||s,'UTF8')),NULL,
 'aut_rpt27_'||s,autenticacion_sha,'ase_rpt27_'||s,'ses_rpt27_'||s,
 'cse_rpt27_'||s,1,'cta_rpt27_'||s,1,'cta_rpt27_'||s,1,t);
 INSERT INTO public.rpt27_ensayo_vector(caso) VALUES(p);
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_actor(text) FROM PUBLIC;

-- El propietario publica datos sintéticos propios con banderas false.
-- Personal17 fija conocido_desde mediante su reloj y conserva la revisión.
-- El snapshot vacío representa una fuente sintética sin catálogo admitido;
-- no se inventan entradas publicadas. La revisión conserva ese mismo snapshot.
CREATE FUNCTION public.rpt27_ensayo_fuente(p text,familia text,estado_fuente text DEFAULT 'vigente') RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE s text:=substr(encode(sha256(convert_to(familia,'UTF8')),'hex'),1,32);
BEGIN
 IF estado_fuente NOT IN ('vigente','suspendida','finalizada') THEN RAISE EXCEPTION 'estado de fixture inválido'; END IF;
 SET LOCAL ROLE vec_personal_propietario;
 IF NOT EXISTS(SELECT 1 FROM vec_personal.relacion_servicio_historia WHERE relacion_ref='rel_rpt27_'||s) THEN
  PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_objetivo_rpt27_'||s,1,'per_objetivo_rpt27_'||s,'emp_objetivo_rpt27_'||s,
   'activa',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours',NULL,'prc_objetivo_rpt27_'||s,1,repeat('a',64));
  INSERT INTO vec_personal.relacion_servicio_historia(relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,modalidad_ref,
   estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256,firma_oficial,eficacia_administrativa,decision_ref,auditoria_ref,catalogo_snapshot)
  VALUES('rel_rpt27_'||s,1,'per_objetivo_rpt27_'||s,'emp_objetivo_rpt27_'||s,'organismo:rpt27:sintetico','unidad:rpt27:sintetica','regimen:rpt27:sintetico','modalidad:rpt27:sintetica',
   estado_fuente,'2026-01-01',CASE WHEN estado_fuente='finalizada' THEN '2026-12-31'::date ELSE NULL END,clock_timestamp(),
   'acto:rpt27:sintetico:'||s,'fuente:rpt27:sintetica:'||s,1,repeat('a',64),false,false,'fixture:rpt27:origen:'||s,'fixture:rpt27:auditoria-origen:'||s,'{}'::jsonb);
 END IF;
 RESET ROLE;
 UPDATE public.rpt27_ensayo_vector SET empleado_ref='emp_objetivo_rpt27_'||s,relacion_ref='rel_rpt27_'||s,
  organismo_ref='organismo:rpt27:sintetico',version_esperada=1,vigente_en='2026-03-01',conocido_en=date_trunc('microseconds',clock_timestamp()) WHERE caso=p;
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_fuente(text,text,text) FROM PUBLIC;

CREATE FUNCTION public.rpt27_ensayo_revision(rel text,estado_fuente text,desde date,hasta date) RETURNS timestamptz
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE anterior record; conocida timestamptz(6);
BEGIN
 SET LOCAL ROLE vec_personal_propietario;
 SELECT * INTO STRICT anterior FROM vec_personal.relacion_servicio_historia WHERE relacion_ref=rel ORDER BY revision DESC LIMIT 1;
 INSERT INTO vec_personal.relacion_servicio_historia(relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,modalidad_ref,
  estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256,firma_oficial,eficacia_administrativa,decision_ref,auditoria_ref,catalogo_snapshot)
 VALUES(rel,anterior.revision+1,anterior.persona_ref,anterior.empleado_ref,anterior.organismo_ref,anterior.unidad_ref,anterior.regimen_ref,anterior.modalidad_ref,
  estado_fuente,desde,hasta,clock_timestamp(),'acto:rpt27:correccion:'||substr(encode(sha256(convert_to(rel,'UTF8')),'hex'),1,32),anterior.fuente_ref,anterior.fuente_version+1,
  repeat('b',64),false,false,'fixture:rpt27:revision','fixture:rpt27:revision',anterior.catalogo_snapshot) RETURNING conocido_desde INTO conocida;
 RESET ROLE;
 RETURN conocida;
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_revision(text,text,date,date) FROM PUBLIC;

CREATE FUNCTION public.rpt27_ensayo_entrada(p text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE s text:=substr(encode(sha256(convert_to(p,'UTF8')),'hex'),1,32); r record; d jsonb; v jsonb; m text;
 k record; cat record; politicas jsonb; seq numeric; raiz numeric; ar jsonb; objetivo public.rpt27_ensayo_vector%ROWTYPE;
BEGIN
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_rpt27_'||s;
 d:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 SELECT * INTO STRICT objetivo FROM public.rpt27_ensayo_vector WHERE caso=p;
 -- Diecisiete claves en el orden congelado del canon1871cab4. El actor
 -- original se conserva; el empleado objetivo es ajeno y exige concesión RPT.
 m:='{"esquema":"vec.personal.relacion-rpt.consulta.v1","operacion":"relacion_para_rpt","empleado_ref":'||to_jsonb(objetivo.empleado_ref)::text||
 ',"relacion_ref":'||to_jsonb(objetivo.relacion_ref)::text||',"organismo_ref":'||to_jsonb(objetivo.organismo_ref)::text||
 ',"version_esperada":'||objetivo.version_esperada::text||',"vigente_en":'||to_jsonb(objetivo.vigente_en::text)::text||
 ',"conocido_en":'||to_jsonb(to_char(objetivo.conocido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text||
 ',"actor_ref":"per_rpt27_'||s||'","contexto_actor_ref":'||to_jsonb(d->>'contexto_actor_ref')::text||',"contexto_version":'||(d->>'contexto_version')||
 ',"cuenta_ref":"cta_rpt27_'||s||'","cuenta_version":'||(d->>'cuenta_version')||',"perfil_ref":"prf_rpt27_'||s||'","perfil_version":'||(d->>'perfil_version')||
 ',"persona_ref":"per_rpt27_'||s||'","persona_version":'||(d->>'persona_version')||'}';
 SELECT * INTO STRICT k FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta;
 SELECT * INTO STRICT cat FROM vec_autorizacion.control_catalogo_politicas WHERE control_id;
 SELECT coalesce(jsonb_agg(p.documento ORDER BY p.politica_ref COLLATE "C"),'[]'::jsonb) INTO politicas
 FROM vec_autorizacion.politica_restrictiva_actual a JOIN vec_autorizacion.politica_restrictiva p ON p.politica_id=a.politica_id AND p.politica_ref=a.politica_ref;
 SELECT greatest(configuracion_secuencia_minima,coalesce((SELECT max(secuencia) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),0))+1,
 greatest(raiz_version_minima,coalesce((SELECT max(version) FROM vec_autorizacion_atestada_v3.raiz_confianza_version),0))+1 INTO seq,raiz
 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id;
 v:=jsonb_build_object('esquema','vec.autenticacion-actor.vinculo.v2.contexto-registrado','bloque_version',2,
 'autenticacion_ref','aut_rpt27_'||s,'autenticacion_huella_sha256',encode(sha256(convert_to('rpt27:autenticacion:'||s,'UTF8')),'hex'),'asercion_ref','ase_rpt27_'||s,
 'sesion_ref','ses_rpt27_'||s,'control_sesion_ref','cse_rpt27_'||s,'control_sesion_revision',1,'control_sesion_huella_sha256',repeat('d',64),
 'cuenta_ref','cta_rpt27_'||s,'cuenta_ordinaria_ref','cta_rpt27_'||s,'principal_id','per_rpt27_'||s,'perfil_activo_ref','prf_rpt27_'||s,
 'cuenta_privilegiada',false,'superficie','interna_corporativa','metodo_observado','certificado','garantia_observada','alto',
 'politica_garantia_ref','pga_rpt27_'||s,'politica_garantia_huella_sha256',repeat('c',64),
 'registro_contexto_ref',r.registro_contexto_ref,'contexto_actor_esquema',d->>'esquema','contexto_actor_ref',d->>'contexto_actor_ref',
 'contexto_actor_version',(d->>'contexto_version')::numeric,'contexto_actor_cuenta_version',(d->>'cuenta_version')::numeric,
 'contexto_actor_huella_sha256',r.huella_sha256,'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
 'autoridad_efectiva',r.autoridad_efectiva);
 SELECT v||jsonb_build_object('autenticacion_verificada_en',a.autenticacion_verificada_en,'sesion_emitida_en',a.sesion_emitida_en,
 'sesion_valida_hasta',c.sesion_valida_hasta,'sesion_revalidada_en',c.sesion_revalidada_en,
 'control_sesion_huella_sha256',c.huella_sha256) INTO v
 FROM vec_autorizacion.sesion_autenticacion_v1 a JOIN vec_autorizacion.control_sesion_v1 c ON c.sesion_ref=a.sesion_ref WHERE a.sesion_ref='ses_rpt27_'||s;
 ar:=jsonb_build_object('caso',p,'ahora',clock_timestamp(),'decision_plantilla_b64',encode(convert_to(jsonb_build_object(
 'decision_ref','decision:rpt27:'||s,'accion','personal.relacion_rpt.consultar','recurso_ref',objetivo.relacion_ref,
 'modulo_id','personal','tipo_recurso','relacion_para_rpt','finalidad','conciliar_relacion_laboral_para_rpt',
 'correlacion_ref','correlacion_'||s,'vinculo_autenticacion_actor',v)::text,'UTF8'),'base64'),
 'motivo_b64',encode(vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb_build_object('esquema','vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
 'referencia',jsonb_build_object('catalogo_id','motivos_rpt27_ensayo','catalogo_version',1,'catalogo_huella_sha256',repeat('e',64),'entrada_clave','motivo_11111111111111111111111111111111'))),'base64'),
 'contexto_b64',encode(r.representacion_canonica,'base64'),'manifiesto_b64',encode(r.manifiesto_procedencia_canonico,'base64'),
 'manifiesto_huella_sha256',r.manifiesto_procedencia_huella_sha256,'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en,
 'alta_b64',encode(convert_to('{}','UTF8'),'base64'),'sellos_b64',encode(convert_to('{}','UTF8'),'base64'),'efecto_huella_sha256',encode(sha256(convert_to(m,'UTF8')),'hex'),
 'clave_id',k.clave_id,'clave_version',k.version,'revision_gobierno',k.revision_gobierno,'huella_gobierno_sha256',k.huella_gobierno_sha256,
 'emisor_id',k.emisor_id,'audiencia_consumo',k.audiencia_consumo,'clave_hmac_b64','','clave_valida_desde',k.valida_desde,'clave_valida_hasta',k.valida_hasta,
 'revision_confianza','configuracion:rpt27:'||s,'secuencia_confianza',seq,'raiz_clave_id','raiz:rpt27:'||s,'raiz_version',raiz,
 'audiencia_despliegue','vec-diputacion/pruebas/rpt27/consumidor','politicas',politicas,'revision_catalogo',cat.revision,'huella_catalogo_sha256',cat.huella_sha256,
 'asignacion_id','rpt27_'||s,'asignacion_version',1,'persona_version',(d->>'persona_version')::numeric,'perfil_version',(d->>'perfil_version')::numeric,
 'empleado_ref',objetivo.empleado_ref,'relacion_ref',objetivo.relacion_ref,'organismo_ref',objetivo.organismo_ref,
 'version_esperada',objetivo.version_esperada,'vigente_en',objetivo.vigente_en::text,'conocido_en',to_char(objetivo.conocido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'material_canonico_b64',encode(convert_to(m,'UTF8'),'base64'));
 UPDATE public.rpt27_ensayo_vector SET material=m,entrada=ar WHERE caso=p;
 RETURN ar;
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_entrada(text) FROM PUBLIC;

CREATE FUNCTION public.rpt27_ensayo_preparado(p text,b jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE db bytea:=decode(b->>'decision_b64','base64'); d jsonb:=convert_from(db,'UTF8')::jsonb;
 rol jsonb:=b->'version_rol_documento'; c jsonb:=b->'control_rol_documento'; a jsonb:=b->'asignacion_documento'; r record; conf jsonb;
BEGIN
 IF d->>'accion' IS DISTINCT FROM 'personal.relacion_rpt.consultar'
 OR d->'campos_permitidos' IS DISTINCT FROM '["cobertura","corte","estado","periodo","procedencia","version"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'ensayo: preparación no nominal'; END IF;
 INSERT INTO vec_autorizacion.version_rol VALUES (d->>'version_rol_ref',rol->>'rol_id',(rol->>'version')::numeric,d->>'version_rol_huella_sha256',(rol->>'publicada_en')::timestamptz,rol);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol VALUES (c->>'version_rol_ref',(c->>'revision')::numeric,c->>'estado',d->>'control_vigencia_version_rol_huella_sha256',(c->>'actualizado_en')::timestamptz,c);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES (c->>'version_rol_ref',(c->>'revision')::numeric,clock_timestamp(),'autoridad-rpt27-sintetica','acto:rpt27:control:'||p);
 INSERT INTO vec_autorizacion.asignacion_perfil VALUES (d->>'asignacion_ref',a->>'asignacion_id',(a->>'version')::numeric,a->>'perfil_activo_ref',a->>'principal_id',a->>'version_rol_ref',d->>'asignacion_huella_sha256',(a->>'emitida_en')::timestamptz,a);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES (a->>'perfil_activo_ref',d->>'asignacion_ref',clock_timestamp(),'autoridad-rpt27-sintetica','acto:rpt27:asignacion:'||p);
 SELECT * INTO STRICT r FROM vec_autorizacion.registrar_decision_contexto_actor_v3(db,decode(b->>'motivo_b64','base64'),(b->>'persona_version')::numeric,(b->>'perfil_version')::numeric);
 IF r.concedida IS NOT TRUE THEN RAISE EXCEPTION 'ensayo: registro durable V3 no concedido'; END IF;
 conf:=jsonb_build_object('decision_ref',d->>'decision_ref','decision_huella_sha256',r.decision_huella_sha256,'emitida_en',d->>'emitida_en','valida_hasta',d->>'valida_hasta','registrada_en',r.registrada_en);
 UPDATE public.rpt27_ensayo_vector SET confirmacion=conf WHERE caso=p;
 RETURN conf;
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_preparado(text,jsonb) FROM PUBLIC;

CREATE FUNCTION public.rpt27_ensayo_firmado(p text,b jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE cap bytea:=decode(b->>'capacidad_b64','base64'); c jsonb:=convert_from(cap,'UTF8')::jsonb; spki bytea:=decode(b->>'spki_b64','base64'); ord numeric;
BEGIN
 IF vec_autorizacion_atestada_v3.capacidad_canonica(c) IS DISTINCT FROM cap
 OR c->>'operacion' IS DISTINCT FROM 'personal.relacion_rpt.consultar'
 OR octet_length(decode(b->>'cose_b64','base64'))<64 THEN RAISE EXCEPTION 'ensayo: bundle Go no nominal'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(c->>'revision_confianza',(c->>'configuracion_secuencia')::numeric,c->>'huella_configuracion_sha256',(c->>'configuracion_publicada_en')::timestamptz,(c->>'configuracion_expira_en')::timestamptz,'acto:rpt27:configuracion:'||p);
 INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
 VALUES(c->>'raiz_clave_id',(c->>'raiz_version')::numeric,spki,encode(sha256(spki),'hex'),(c->>'raiz_valida_desde')::timestamptz,(c->>'raiz_valida_hasta')::timestamptz,c->>'suite',c->>'audiencia_despliegue','acto:rpt27:raiz:'||p);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES(c->>'revision_confianza',c->>'raiz_clave_id',(c->>'raiz_version')::numeric);
 SELECT coalesce(max(orden),0)+1 INTO ord FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual;
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref) VALUES(ord,c->>'revision_confianza',clock_timestamp(),'acto:rpt27:puntero:'||p);
 UPDATE public.rpt27_ensayo_vector SET bundle=b WHERE caso=p;
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_firmado(text,jsonb) FROM PUBLIC;

-- Invoker: session_user sigue siendo el LOGIN nominal del ensayo, jamás el
-- propietario. Esta envoltura transporta bytes al consumidor real sin decidir.
CREATE FUNCTION public.rpt27_ensayo_consultar(p text, variante text DEFAULT 'valida') RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b jsonb; m text; d bytea; j jsonb; original_m text; original_d bytea; original_b jsonb;
BEGIN
 SELECT material,bundle INTO STRICT m,b FROM public.rpt27_ensayo_vector WHERE caso=p;
 d:=decode(b->>'decision_b64','base64');
 original_m:=m; original_d:=d; original_b:=b;
 IF variante='actor' THEN m:=replace(m,'"actor_ref":"per_rpt27_','"actor_ref":"per_ajena_');
 ELSIF variante='ambito' THEN m:=replace(m,'"empleado_ref":"emp_objetivo_rpt27_','"empleado_ref":"emp_ajeno_');
 ELSIF variante='perfil' THEN m:=replace(m,'"perfil_ref":"prf_rpt27_','"perfil_ref":"prf_ajeno_');
 ELSIF variante='relacion' THEN m:=replace(m,'"relacion_ref":"rel_rpt27_','"relacion_ref":"rel_ajena_');
 ELSIF variante='organismo' THEN m:=replace(m,'organismo:rpt27:sintetico','organismo:rpt27:ajeno');
 ELSIF variante='campos' THEN
 j:=convert_from(d,'UTF8')::jsonb; j:=jsonb_set(j,'{campos_permitidos}','["persona_ref"]'::jsonb);
 d:=convert_to(j::text,'UTF8');
 ELSIF variante='cose' THEN b:=jsonb_set(b,'{cose_b64}',to_jsonb(encode(set_byte(decode(b->>'cose_b64','base64'),10,255),'base64')));
 ELSIF variante<>'valida' THEN RAISE EXCEPTION 'ensayo: variante desconocida'; END IF;
 IF variante<>'valida' AND ROW(m,d,b) IS NOT DISTINCT FROM ROW(original_m,original_d,original_b) THEN
  RAISE EXCEPTION 'fixture RPT27: variante no cambió material ni transporte' USING ERRCODE='55000';
 END IF;
 RETURN vec_personal.consultar_relacion_para_rpt_v1(m,
 decode(b->>'capacidad_b64','base64'),d,decode(b->>'motivo_b64','base64'),decode(b->>'contexto_b64','base64'),
 (b->>'persona_version')::numeric,(b->>'perfil_version')::numeric,decode(b->>'payload_b64','base64'),
 decode(b->>'cose_b64','base64'),decode(b->>'evidencia_b64','base64'),decode(b->>'spki_b64','base64'));
END $f$;
REVOKE ALL ON FUNCTION public.rpt27_ensayo_consultar(text,text) FROM PUBLIC;
