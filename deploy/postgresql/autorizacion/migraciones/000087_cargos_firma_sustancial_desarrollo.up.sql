\set ON_ERROR_STOP on
-- AUT87: plan v2 nuevo, sólo entorno de desarrollo aprobado. No publica
-- roles al instalar ni modifica versiones, perfiles o recibos emitidos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ DECLARE actual text; BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
 RAISE EXCEPTION 'AUT87: PARO clave=migrador actual=no_acreditado esperado=PG18_superusuario' USING ERRCODE='42501'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') INTO actual FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)') AND p.proowner='vec_autorizacion_propietario'::regrole AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}';
 IF actual IS DISTINCT FROM 'df40909cd75c2974d56af9296d203d5918bfa0715bc4ff09c0a3e28f2ea09cdc' THEN RAISE EXCEPTION 'AUT87: PARO clave=documento_rol_cargo_firma_v1.definicion_owner_acl actual=% esperado=df40909cd75c2974d56af9296d203d5918bfa0715bc4ff09c0a3e28f2ea09cdc',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') INTO actual FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.aplicar_cargos_firma_admin_v1(text,text)') AND p.proowner='vec_autorizacion_propietario'::regrole AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario}';
 IF actual IS DISTINCT FROM 'd9e1c18818b72a24b36ac9f9d7e8423f70ae20d6c9c77de4d99aee0ae8436580' THEN RAISE EXCEPTION 'AUT87: PARO clave=aplicar_cargos_firma_admin_v1.definicion_owner_acl actual=% esperado=d9e1c18818b72a24b36ac9f9d7e8423f70ae20d6c9c77de4d99aee0ae8436580',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(text)') IS NOT NULL THEN RAISE EXCEPTION 'AUT87: PARO clave=fabrica_v2 actual=existente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(p_operacion text)
RETURNS jsonb LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $function$
 SELECT CASE p_operacion
 WHEN 'firmar_vec' THEN jsonb_set(vec_autorizacion.concesion_v2_cargo_firma_v1(p_operacion),'{garantia_minima}','"sustancial"'::jsonb)
 WHEN 'consultar_r5' THEN jsonb_set(vec_autorizacion.concesion_v2_cargo_firma_v1(p_operacion),'{garantia_minima}','"sustancial"'::jsonb)
 WHEN 'consultar_detalle' THEN '{"accion":"contratacion_temporal.expediente.consultar","modulo_id":"contratacion_temporal","tipo_recurso":"expediente_contratacion_temporal","finalidades":["tramitacion_expediente_contratacion_temporal"],"garantia_minima":"sustancial","campos_permitidos":[],"obligaciones":[]}'::jsonb
 WHEN 'descargar_original' THEN '{"accion":"documentos.original.descargar","modulo_id":"documentos","tipo_recurso":"documento_original","finalidades":["descargar_documento_original"],"garantia_minima":"sustancial","campos_permitidos":["contenido","documento"],"obligaciones":[]}'::jsonb
 WHEN 'custodiar_firmado' THEN '{"accion":"documentos.firmado.custodiar","modulo_id":"documentos","tipo_recurso":"documento_firmado","finalidades":["custodiar_documento_firmado"],"garantia_minima":"sustancial","campos_permitidos":["documento_firmado.custodia","evidencia_custodia"],"obligaciones":[]}'::jsonb
 WHEN 'reservar_original' THEN '{"accion":"documentos.original_firmable.reservar","modulo_id":"documentos","tipo_recurso":"documento_original_firmable","finalidades":["custodiar_original_firmable"],"garantia_minima":"sustancial","campos_permitidos":["intento","reserva"],"obligaciones":[]}'::jsonb
 WHEN 'confirmar_original' THEN '{"accion":"documentos.original_firmable.confirmar","modulo_id":"documentos","tipo_recurso":"documento_original_firmable","finalidades":["custodiar_original_firmable"],"garantia_minima":"sustancial","campos_permitidos":["documento","recibo"],"obligaciones":[]}'::jsonb
 WHEN 'escribir_original' THEN '{"accion":"documentos.original_firmable.almacen.escribir","modulo_id":"documentos","tipo_recurso":"documento_original_firmable","finalidades":["custodiar_original_firmable"],"garantia_minima":"sustancial","campos_permitidos":["evidencia_almacen","original_firmable.contenido"],"obligaciones":[]}'::jsonb
 END
$function$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(text) FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_autorizacion.documento_rol_cargo_firma_v1(c jsonb, p jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE concesiones jsonb:='[]';x jsonb;op text;anterior text:='';desarrollo boolean:=p->>'esquema'='vec.admin.cargos-firma.plan.v2';clase text;
BEGIN
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>12
 OR NOT c ?& ARRAY['rol_id','version','nombre','version_anterior_sha256','operaciones_v2','competencias','version_rol_sha256','regla_asignacion','organizacion_ref','vigente_desde','vigente_hasta','duracion_propuesta_segundos']
 OR jsonb_typeof(c->'rol_id') IS DISTINCT FROM 'string' OR c->>'rol_id' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'version') IS DISTINCT FROM 'number' OR c->>'version' !~ '^[1-9][0-9]{0,8}$'
 OR jsonb_typeof(c->'nombre') IS DISTINCT FROM 'string' OR char_length(c->>'nombre') NOT BETWEEN 3 AND 200 OR c->>'nombre'<>btrim(c->>'nombre') OR c->>'nombre' ~ '[[:cntrl:]]' OR c->>'nombre' ~ '^[[:space:]]|[[:space:]]$'
 OR jsonb_typeof(c->'version_anterior_sha256') IS DISTINCT FROM 'string'
 OR ((c->>'version')='1' AND c->>'version_anterior_sha256'<>'') OR ((c->>'version')<>'1' AND c->>'version_anterior_sha256' !~ '^[0-9a-f]{64}$')
 OR jsonb_typeof(c->'operaciones_v2') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'operaciones_v2')>(CASE WHEN desarrollo THEN 5 ELSE 2 END)
 OR jsonb_typeof(c->'competencias') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'competencias') NOT BETWEEN (CASE WHEN desarrollo THEN 0 ELSE 1 END) AND 8
 OR jsonb_typeof(c->'version_rol_sha256') IS DISTINCT FROM 'string' OR c->>'version_rol_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(c->'regla_asignacion') IS DISTINCT FROM 'string' OR c->>'regla_asignacion' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'organizacion_ref') IS DISTINCT FROM 'string' OR c->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'AUT53: PARO clave=cargo actual=invalido esperado=campos_exactos' USING ERRCODE='22023'; END IF;

 IF desarrollo THEN
  IF c->>'rol_id' !~ '^ct_cargo_(tecnico|jefatura|diputacion|secretaria)_(firma|consulta|documentos)_dev$'
  OR c->>'version'<>'1' OR c->>'version_anterior_sha256'<>'' OR jsonb_array_length(c->'operaciones_v2')=0 THEN
  RAISE EXCEPTION 'AUT87: PARO clave=rol_desarrollo actual=invalido esperado=rol_nuevo_ct_cargo_dev' USING ERRCODE='22023'; END IF;
  clase:=split_part(c->>'rol_id','_',4);
  IF (clase='firma' AND jsonb_array_length(c->'competencias')=0) OR (clase<>'firma' AND jsonb_array_length(c->'competencias')<>0)
  OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c->'operaciones_v2') o WHERE NOT(
   (clase='firma' AND o IN('consultar_r5','firmar_vec')) OR (clase='consulta' AND o='consultar_detalle')
   OR (clase='documentos' AND (o IN('descargar_original','custodiar_firmado') OR (split_part(c->>'rol_id','_',3)='tecnico' AND o IN('reservar_original','confirmar_original','escribir_original')))))) THEN
  RAISE EXCEPTION 'AUT87: PARO clave=concesiones_desarrollo actual=invalido esperado=permisos_minimos_del_cargo' USING ERRCODE='22023'; END IF;
 END IF;
 -- Operaciones V2: subconjunto ordenado y sin repetir de las dos admitidas.
 FOR op IN SELECT value FROM jsonb_array_elements_text(c->'operaciones_v2') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF (NOT desarrollo AND op NOT IN('consultar_r5','firmar_vec')) OR (desarrollo AND vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(op) IS NULL) OR op COLLATE "C"<=anterior COLLATE "C" THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=subconjunto_ordenado' USING ERRCODE='22023'; END IF;
  anterior:=op;
  concesiones:=concesiones||jsonb_build_array(CASE WHEN desarrollo THEN vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(op) ELSE vec_autorizacion.concesion_v2_cargo_firma_v1(op) END);
 END LOOP;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(c->'operaciones_v2') e WHERE jsonb_typeof(e)<>'string') THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=cadenas' USING ERRCODE='22023'; END IF;
 FOR x IN SELECT value FROM jsonb_array_elements(c->'competencias') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF jsonb_typeof(x) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(x))<>3 OR NOT x ?& ARRAY['accion','tipo_recurso','finalidad']
  OR jsonb_typeof(x->'accion') IS DISTINCT FROM 'string' OR x->>'accion' !~ '^contratacion_temporal\.documento(\.[a-z][a-z0-9_]{1,63}){1,4}$'
  OR jsonb_typeof(x->'tipo_recurso') IS DISTINCT FROM 'string' OR x->>'tipo_recurso' !~ '^documento_([a-z0-9_]{1,80}_)?contratacion_temporal$'
  OR jsonb_typeof(x->'finalidad') IS DISTINCT FROM 'string' OR x->>'finalidad' !~ '^[a-z][a-z0-9_]{2,127}$' OR x->>'finalidad' ~ '(fiscaliz|intervenc)'
  THEN RAISE EXCEPTION 'AUT53: PARO clave=competencia actual=invalida esperado=documento_contratacion_temporal' USING ERRCODE='22023'; END IF;
  concesiones:=concesiones||jsonb_build_array(jsonb_build_object('accion',x->>'accion','modulo_id','contratacion_temporal','tipo_recurso',x->>'tipo_recurso',
   'finalidades',jsonb_build_array(x->>'finalidad'),'garantia_minima',CASE WHEN desarrollo THEN 'sustancial' ELSE 'alto' END,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb));
 END LOOP;
 RETURN jsonb_build_object('rol_id',c->>'rol_id','version',(c->>'version')::bigint,'nombre',c->>'nombre','estado','publicada','concesiones',concesiones,
  'publicada_por','operacion:cargos_firma:'||(p->>'operacion_ref'),'publicada_en',p->>'preparado_en');
END $function$
;
CREATE OR REPLACE FUNCTION vec_autorizacion.aplicar_cargos_firma_admin_v1(plan_canonico text, sha_aprobado text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE cfg vec_autorizacion.config_cargos_firma_admin_v1;p jsonb;sha text;c jsonb;d jsonb;ctl jsonb;ref text;h text;hc text;maxv bigint;prev text;
 reg record;lista jsonb:='[]';cargos jsonb:='[]';lista_sha text;e jsonb;aud record;recibo jsonb;previo record;publicador text;instante timestamptz;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_cargos_firma_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=divergente esperado=plan_privado_aprobado' USING ERRCODE='42501'; END IF;
 sha:=encode(sha256(convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan_sha actual=divergente esperado=aprobado' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;
 -- El texto aprobado debe ser la forma canónica de jsonb, como en AUT49.
 IF plan_canonico IS DISTINCT FROM p::text THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=no_canonico esperado=jsonb_text' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>5 OR NOT p ?& ARRAY['esquema','operacion_ref','preparado_en','caduca_en','cargos']
 OR COALESCE(p->>'esquema','') NOT IN ('vec.admin.cargos-firma.plan.v1','vec.admin.cargos-firma.plan.v2') OR jsonb_typeof(p->'operacion_ref') IS DISTINCT FROM 'string' OR p->>'operacion_ref' !~ '^rpa_cf_[A-Za-z0-9_-]{19,121}$'
 OR jsonb_typeof(p->'cargos') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'cargos') NOT BETWEEN 1 AND 16
 OR (SELECT count(DISTINCT x->>'rol_id') FROM jsonb_array_elements(p->'cargos') x)<>jsonb_array_length(p->'cargos')
 OR jsonb_typeof(p->'preparado_en') IS DISTINCT FROM 'string' OR jsonb_typeof(p->'caduca_en') IS DISTINCT FROM 'string'
 OR p->>'preparado_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' OR p->>'caduca_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR (p->>'caduca_en')::timestamptz>(p->>'preparado_en')::timestamptz+interval '1 day'
 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=invalido esperado=plan_cerrado_1_16' USING ERRCODE='22023'; END IF;
 IF p->>'esquema'='vec.admin.cargos-firma.plan.v2' AND cfg.entorno IS DISTINCT FROM 'desarrollo' THEN RAISE EXCEPTION 'AUT87: PARO clave=entorno actual=% esperado=desarrollo',cfg.entorno USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.registro_cargos_firma_admin_v1 WHERE operacion_ref=p->>'operacion_ref' FOR SHARE;
 IF FOUND THEN
  -- Replay: mismo plan exacto; el recibo original sólo se devuelve si cada
  -- versión publicada sigue con su huella, habilitada y registrada como ordinaria.
  IF previo.plan IS DISTINCT FROM convert_to(plan_canonico,'UTF8') OR previo.plan_sha256 IS DISTINCT FROM sha
  THEN RAISE EXCEPTION 'AUT53: PARO clave=replay actual=material_distinto esperado=plan_original' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(previo.recibo->'cargos') x LEFT JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=x->>'version_rol_ref'
   LEFT JOIN vec_autorizacion.rol_administrable_exacto_v1 a ON a.version_rol_ref=r.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=ca.version_rol_ref AND v.revision=ca.revision
   WHERE r.version_rol_ref IS NULL OR r.huella_sha256 IS DISTINCT FROM x->>'version_rol_sha256' OR a.version_rol_ref IS NULL OR a.clase<>'ordinario' OR a.huella_sha256 IS DISTINCT FROM r.huella_sha256
   OR v.estado IS DISTINCT FROM 'habilitada')
  THEN RAISE EXCEPTION 'AUT53: PARO clave=replay_cargos actual=divergente esperado=registro_original' USING ERRCODE='P0V01'; END IF;
  RETURN jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AUT53: PARO clave=plan_caducado actual=caducado esperado=vigente' USING ERRCODE='42501'; END IF;
 publicador:='operacion:cargos_firma:'||(p->>'operacion_ref');
 instante:=(p->>'preparado_en')::timestamptz;
 FOR c IN SELECT value FROM jsonb_array_elements(p->'cargos') ORDER BY value->>'rol_id' LOOP
  d:=vec_autorizacion.documento_rol_cargo_firma_v1(c,p);
  -- CAS de versión: la 1 exige que el rol no exista; la N exige que la última
  -- publicada sea N-1 y tenga la huella que el plan declara.
  SELECT max(version) INTO maxv FROM vec_autorizacion.version_rol WHERE rol_id=c->>'rol_id';
  SELECT huella_sha256 INTO prev FROM vec_autorizacion.version_rol WHERE rol_id=c->>'rol_id' AND version=(c->>'version')::bigint-1 FOR SHARE;
  IF ((c->>'version')::bigint=1 AND maxv IS NOT NULL) OR ((c->>'version')::bigint>1 AND (maxv IS DISTINCT FROM (c->>'version')::bigint-1 OR prev IS DISTINCT FROM c->>'version_anterior_sha256'))
  THEN RAISE EXCEPTION 'AUT53: PARO clave=version actual=divergente esperado=CAS_version_anterior' USING ERRCODE='P0V01'; END IF;
  -- Sólo se sucede a un cargo que publicó AUT53: la versión N-1 debe figurar,
  -- con su huella, en el recibo de una operación anterior de este circuito.
  -- Así no se saca la versión siguiente de un rol ajeno (Dietas, Bolsa o un
  -- rol de RRHH de Contratación temporal que no es un cargo).
  IF (c->>'version')::bigint>1 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.registro_cargos_firma_admin_v1 g
   CROSS JOIN LATERAL jsonb_array_elements(g.recibo->'cargos') x
   WHERE x->>'version_rol_ref'='rol:'||(c->>'rol_id')||':v'||((c->>'version')::bigint-1) AND x->>'version_rol_sha256' IS NOT DISTINCT FROM prev)
  THEN RAISE EXCEPTION 'AUT53: PARO clave=version_anterior actual=no_es_cargo esperado=publicada_por_aut53' USING ERRCODE='P0V01'; END IF;
  SELECT * INTO reg FROM vec_autorizacion.regla_asignacion_cargo_firma_v1 r WHERE r.regla=c->>'regla_asignacion';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT53: PARO clave=regla_asignacion actual=desconocida esperado=configurada' USING ERRCODE='22023'; END IF;
  IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE THEN RAISE EXCEPTION 'AUT53: PARO clave=concesiones actual=invalidas esperado=positivas_sin_repetir' USING ERRCODE='22023'; END IF;
  h:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex');
  IF h IS DISTINCT FROM c->>'version_rol_sha256' THEN RAISE EXCEPTION 'AUT53: PARO clave=version_rol_sha actual=divergente esperado=huella_del_plan' USING ERRCODE='P0V01'; END IF;
  ref:='rol:'||(c->>'rol_id')||':v'||(c->>'version');
  ctl:=jsonb_build_object('version_rol_ref',ref,'revision',1,'estado','habilitada','actualizado_por',publicador,'actualizado_en',p->>'preparado_en');
  hc:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(ctl),'UTF8')),'hex');
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES(ref,c->>'rol_id',(c->>'version')::bigint,h,instante,d);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES(ref,1,'habilitada',hc,instante,ctl);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) VALUES(ref,1,instante,publicador,publicador);
  -- Defensa de clases de AUT49 sobre la versión recién publicada. La unidad
  -- la aporta cada asignación: AUT32 exige organización y unidad.
  lista:=lista||jsonb_build_array(vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb_build_object('version_rol_ref',ref,'version_rol_sha256',h,
   'control_revision',1,'control_sha256',hc,'unidad_requerida',true,
   'ambitos_fijos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(c->>'organizacion_ref'))),
   'vigente_desde',c->'vigente_desde','vigente_hasta',c->'vigente_hasta','duracion_propuesta_segundos',c->'duracion_propuesta_segundos'),true));
  IF EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=ref) THEN RAISE EXCEPTION 'AUT53: PARO clave=ya_registrado actual=presente esperado=ausente' USING ERRCODE='23505'; END IF;
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1(version_rol_ref,clase,huella_sha256,vigente_desde,vigente_hasta,unidad_requerida,audiencia_administrativa,ambitos_fijos,duracion_propuesta)
  VALUES(ref,'ordinario',h,(c->>'vigente_desde')::timestamptz,(c->>'vigente_hasta')::timestamptz,true,reg.audiencia_administrativa,
   jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(c->>'organizacion_ref'))),
   make_interval(secs=>(c->>'duracion_propuesta_segundos')::double precision));
  cargos:=cargos||jsonb_build_array(jsonb_build_object('rol_id',c->>'rol_id','version_rol_ref',ref,'version_rol_sha256',h,'control_revision',1,'control_sha256',hc,'regla_asignacion',c->>'regla_asignacion'));
 END LOOP;
 lista_sha:=encode(sha256(convert_to(lista::text,'UTF8')),'hex');
 e:=jsonb_build_object('tipo_registro','perfiles_asignables_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,
  'operacion_ref',p->>'operacion_ref','perfiles_sha256',lista_sha,'perfiles_numero',jsonb_array_length(lista)::text,'aprobacion_sha256',cfg.aprobacion_sha256,
  'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref','correlacion_'||substr(sha,33,32));
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(e);
 INSERT INTO vec_autorizacion.operacion_cargos_firma_admin_v1 VALUES(aud.auditoria_ref,'confirmacion',session_user,sha,'permitido',p->>'operacion_ref',aud.registrada_en);
 recibo:=jsonb_build_object('esquema','vec.admin.cargos-firma.recibo.v1','operacion_ref',p->>'operacion_ref','plan_sha256',sha,'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,
  'cargos',cargos,'perfiles_sha256',lista_sha,'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_cargos_firma_admin_v1 VALUES(p->>'operacion_ref',sha,convert_to(plan_canonico,'UTF8'),session_user,aud.auditoria_ref,recibo,aud.registrada_en);
 -- Revalidación final bajo la misma transacción: configuración aún vigente.
 PERFORM vec_autorizacion.exigir_operador_cargos_firma_admin_v1();
 IF clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'AUT53: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recibo',recibo,'replay',false);
END $function$
;
REVOKE ALL ON FUNCTION vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_cargos_firma_admin_v1(text,text) FROM PUBLIC;
RESET ROLE;
COMMIT;
