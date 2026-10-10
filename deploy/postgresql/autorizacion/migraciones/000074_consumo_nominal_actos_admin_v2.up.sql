\set ON_ERROR_STOP on
-- AUT74: adapta los contratos emitidos por Go v2 y liga AUT24 al consumidor AD236.
-- Definiciones literales de copia local HZ20; conserva ACL y alcance de las fachadas.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT74: falta dependencia AD236' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_material_acto_admin_v1(text,boolean)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'ad61d78e763fa42c53832bdafcc66fa6ff3475268f246d1984fe905d10286392'
 THEN RAISE EXCEPTION 'AUT74: preimagen validador actual=% esperado=ad61d78e763fa42c53832bdafcc66fa6ff3475268f246d1984fe905d10286392',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'e4948ef6d45c445b2b0f472f9d937305ca9726d0dc463e12d3da791b67a67f4d'
 THEN RAISE EXCEPTION 'AUT74: preimagen consumidor actual=% esperado=e4948ef6d45c445b2b0f472f9d937305ca9726d0dc463e12d3da791b67a67f4d',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '687bed669c9343fce839a4aa177806d7589e5a5b2eb302e568f9a085acd51c96'
 THEN RAISE EXCEPTION 'AUT74: preimagen cierre actual=% esperado=687bed669c9343fce839a4aa177806d7589e5a5b2eb302e568f9a085acd51c96',h USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.validar_material_acto_admin_v1(p_material text, p_propuesta boolean)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE m jsonb; o jsonb; r jsonb; claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
  RAISE EXCEPTION 'AUT24: material invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb; o:=m->'objetivo';
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_acto_v2'
 OR NOT m ?& ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','asignacion_ref']
 OR jsonb_path_exists(m,'$.** ? (@ == null)')
 OR claves <@ ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','asignacion_ref','unidad_ref','referencia_acto'] IS NOT TRUE
 OR jsonb_typeof(o) IS DISTINCT FROM 'object'
 OR jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object'
 OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']
 OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE
 OR (m#>>'{motivo,catalogo_version}')::numeric<1
 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$'
 OR m->>'operacion' NOT IN ('otorgar','revocar')
 OR (CASE WHEN p_propuesta THEN m->>'operacion_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' ELSE m->>'operacion_ref' !~ '^acto_admin:[0-9a-f]{32}$' END)
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_persona_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_perfil_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'asignacion_ref',512) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT24: DTO de acto invalido' USING ERRCODE='22023'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 IF r->>'huella_sha256' IS DISTINCT FROM m->>'rol_huella_sha256'
 OR (p_propuesta AND r->>'clase' NOT IN ('administrador','intervencion'))
 OR (NOT p_propuesta AND r->>'clase' IS DISTINCT FROM 'ordinario')
 OR ((r->>'unidad_requerida')::boolean AND vec_autorizacion.texto_positivo_valido(m->>'unidad_ref',512) IS NOT TRUE)
 OR (NOT (r->>'unidad_requerida')::boolean AND m ? 'unidad_ref')
 OR (m ? 'referencia_acto' AND vec_autorizacion.texto_positivo_valido(m->>'referencia_acto',512) IS NOT TRUE)
 OR (m->>'actor_persona_ref' IS NOT DISTINCT FROM o->>'persona_ref'
  AND NOT(p_propuesta AND r->>'clase'='administrador' AND m->>'operacion'='revocar')) THEN
  RAISE EXCEPTION 'AUT24: rol, unidad o independencia invalidos' USING ERRCODE='42501'; END IF;
 IF NOT o ?& ARRAY['cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','perfil_version','vinculo_ref','vinculo_version','huella_sha256','revision_continuidad','procedencia_ref','procedencia_version','procedencia_huella_sha256','vigente_hasta','vigente_desde']
 OR (SELECT count(*) FROM jsonb_object_keys(o))<>(CASE WHEN o ? 'centro_ref' THEN 16 ELSE 15 END)
 OR (o ? 'centro_ref' AND vec_autorizacion.texto_positivo_valido(o->>'centro_ref',512) IS NOT TRUE)
 OR o->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR o->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR o->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR o->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR o->>'huella_sha256' !~ '^[0-9a-f]{64}$'
 OR o->>'procedencia_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR (o->>'cuenta_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'persona_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'procedencia_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR NOT isfinite((o->>'vigente_desde')::timestamptz)
 OR (m->>'operacion'='otorgar' AND ((o->>'perfil_version')::numeric<>0 OR (o->>'vinculo_version')::numeric<>0 OR NOT isfinite((o->>'vigente_hasta')::timestamptz)))
 OR (m->>'operacion'='revocar' AND ((o->>'perfil_version')::numeric<1 OR (o->>'vinculo_version')::numeric<1 OR o->>'vigente_hasta'<>'0001-01-01T00:00:00Z')) THEN
  RAISE EXCEPTION 'AUT24: preimagen estructural invalida' USING ERRCODE='22023'; END IF;
 RETURN m;
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE m jsonb:=p_material::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb; c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; x record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'AUT24: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF c->>'efecto_ref' IS DISTINCT FROM m->>'operacion_ref'
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM encode(sha256(convert_to(p_material,'UTF8')),'hex')
 OR c->>'operacion' IS DISTINCT FROM d->>'accion'
 OR d->>'recurso_ref' IS DISTINCT FROM m->>'operacion_ref'
 OR d->>'asignacion_ref' IS DISTINCT FROM m->>'asignacion_ref'
 OR d->>'principal_id' IS DISTINCT FROM m->>'actor_persona_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'actor_perfil_ref'
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true' THEN
  RAISE EXCEPTION 'AUT24: sello de negocio divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM m->>'operacion_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.auditoria_ref IS NULL OR x.consumida_en IS NULL
 THEN RAISE EXCEPTION 'AUT74: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 -- AD150 exige IS12 nominal dentro de la misma transacción, también replay.
 RETURN jsonb_build_object('decision_ref',x.decision_ref,'auditoria_ref',x.auditoria_ref,'consumida_en',x.consumida_en,'consumo_nuevo',x.consumo_nuevo);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE m jsonb:=p_material::jsonb;consumo jsonb;resultado jsonb;propuesta record;material_propuesta jsonb;rol jsonb;revision bigint;efectivos jsonb;huella text;ahora timestamptz;recibo jsonb:=null;acto jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: cierre invalido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[] OR jsonb_path_exists(m,'$.** ? (@ == null)') OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_cierre_v2' OR vec_autorizacion.texto_positivo_valido(m->>'asignacion_ref',512) IS NOT TRUE OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$' OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'decision' NOT IN ('aprobada','rechazada') THEN RAISE EXCEPTION 'AUT24: DTO cierre invalido' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object' OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'] OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE OR (m#>>'{motivo,catalogo_version}')::numeric<1 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'AUT24: motivo de cierre invalido' USING ERRCODE='22023'; END IF;
 IF d->>'accion' IS DISTINCT FROM (CASE WHEN m->>'decision'='aprobada' THEN 'administracion.perfiles.aprobar' ELSE 'administracion.perfiles.rechazar' END) THEN RAISE EXCEPTION 'AUT24: accion cierre divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_aplicacion_efectivos_internos_v3();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: aprobador no efectivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=m->>'propuesta_ref' FOR SHARE;
 IF propuesta.huella_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256' OR m->>'actor_persona_ref' IN (propuesta.proponente_persona_ref,propuesta.objetivo_persona_ref) THEN RAISE EXCEPTION 'AUT24: cierre no independiente' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 SELECT cc.revision INTO STRICT revision FROM vec_autorizacion.control_continuidad_admin cc WHERE cc.control_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible WHERE propuesta_ref=propuesta.propuesta_ref) THEN RAISE EXCEPTION 'AUT24: propuesta ya cerrada' USING ERRCODE='23505'; END IF;
 material_propuesta:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb;
 IF m->>'decision'='aprobada' THEN
  IF clock_timestamp()>=propuesta.caduca_en OR revision IS DISTINCT FROM propuesta.revision_continuidad_esperada OR (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=propuesta.proponente_persona_ref AND x->>'perfil_ref'=propuesta.proponente_perfil_ref) THEN RAISE EXCEPTION 'AUT24: propuesta obsoleta o proponente revocado' USING ERRCODE='40001'; END IF;
 END IF;
 ahora:=clock_timestamp();huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 INSERT INTO vec_autorizacion.cierre_propuesta_perfil_sensible(propuesta_ref,cierre_ref,resultado,aprobador_persona_ref,aprobador_cuenta_ref,aprobador_perfil_ref,motivo_codigo,revision_continuidad_observada,documento_canonico,huella_sha256,cerrada_en)
 VALUES(propuesta.propuesta_ref,m->>'operacion_ref',m->>'decision',m->>'actor_persona_ref',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',m->>'actor_perfil_ref',m#>>'{motivo,entrada_clave}',revision,convert_to(p_material,'UTF8'),huella,ahora);
 IF m->>'decision'='aprobada' THEN
  recibo:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(material_propuesta,m->>'operacion_ref',propuesta.propuesta_ref,consumo->>'auditoria_ref',m->>'actor_persona_ref',false);
  ahora:=(recibo->>'confirmado_en')::timestamptz;
  acto:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'preimagen_huella_sha256',recibo->>'huella_antes_sha256','postimagen_huella_sha256',recibo->>'huella_despues_sha256','auditoria_ref',consumo->>'auditoria_ref');
  INSERT INTO vec_autorizacion.acto_perfil_sensible VALUES(recibo->>'acto_ref',propuesta.propuesta_ref,m->>'operacion_ref',revision,revision+1,recibo->>'huella_antes_sha256',recibo->>'huella_despues_sha256',convert_to(acto::text,'UTF8'),encode(sha256(convert_to(acto::text,'UTF8')),'hex'),ahora);
  INSERT INTO vec_autorizacion.recibo_perfil_sensible VALUES(recibo->>'recibo_ref',recibo->>'acto_ref',convert_to(recibo::text,'UTF8'),encode(sha256(convert_to(recibo::text,'UTF8')),'hex'),ahora);
 END IF;
 resultado:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'decision',m->>'decision','huella_cierre_sha256',huella,'confirmado_en',ahora,'recibo',recibo);
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,CASE WHEN m->>'decision'='aprobada' THEN 'propuesta_aprobada' ELSE 'propuesta_rechazada' END);
END $function$;

DO $post$
DECLARE h text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.validar_material_acto_admin_v1(text,boolean)'::regprocedure
   AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'])
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
   AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'])
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
   AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC'])
 THEN RAISE EXCEPTION 'AUT74: postimagen ACL o propietario divergente' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_material_acto_admin_v1(text,boolean)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '2ce7b31acb11c859d437774b6c5235e0639d1cbfc8825113889f2440b55eb6ed'
 THEN RAISE EXCEPTION 'AUT74: validador postimagen actual=%',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '91bc582ee90cc8006336df048febdff3041e6b207985c4eccf28b1649bc4f6fe'
 THEN RAISE EXCEPTION 'AUT74: consumidor postimagen actual=%',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'bca6a57bda24e9fa7179f67fa1d2a8a7523b057f19cdbad7d32d94dc11ea92e7'
 THEN RAISE EXCEPTION 'AUT74: cierre postimagen actual=%',h USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
