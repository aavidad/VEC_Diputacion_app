\set ON_ERROR_STOP on
-- AUT40: intento nominal de cada llamada al bootstrap, con efecto AD171 intacto.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE p record;actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'AUT40: PARO clave=migrador actual=no_superusuario esperado=superusuario' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'AUT40: PARO clave=PG actual=% esperado=18',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(text,text)') IS NOT NULL THEN
  RAISE EXCEPTION 'AUT40: PARO clave=dependencias actual=divergente esperado=AD179_sin_AUT40' USING ERRCODE='55000'; END IF;
 SELECT x.*,encode(pg_catalog.sha256(convert_to(x.prosrc,'UTF8')),'hex') sha INTO p FROM pg_proc x
 WHERE x.oid=to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)');
 actual:=COALESCE(p.sha,'ausente');
 IF actual IS DISTINCT FROM 'bbbb9a849f3423b2eb010e792c01d383009688d9f7d89139664817dc184a1f94' OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
 OR p.prorettype IS DISTINCT FROM 'jsonb'::regtype OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 'v'
 OR cardinality(p.proconfig)<>5 OR NOT 'search_path=pg_catalog'=ANY(p.proconfig)
 OR NOT 'TimeZone=UTC'=ANY(p.proconfig) OR NOT 'row_security=on'=ANY(p.proconfig)
 OR NOT 'lock_timeout=2s'=ANY(p.proconfig) OR NOT 'statement_timeout=30s'=ANY(p.proconfig)
 THEN RAISE EXCEPTION 'AUT40: PARO clave=AUT37.prosrc_metadata actual=% esperado=bbbb9a849f3423b2eb010e792c01d383009688d9f7d89139664817dc184a1f94',actual USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
 WHERE a.grantee NOT IN('vec_autorizacion_propietario'::regrole,'vec_admin_bootstrap_central_v3_ejecutor'::regrole)
 OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
  RAISE EXCEPTION 'AUT40: PARO clave=ACL actual=ampliada esperado=owner_grupo_exclusivo' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
SET row_security=on SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE cfg vec_autorizacion.config_bootstrap_central_admin_v3;p jsonb;sha text;previo record;
 preimagen jsonb;preimagen_bytes bytea;preimagen_sha text;persona jsonb;sistema jsonb;gob jsonb;config_rol record;
 perfiles text[]:=ARRAY[]::text[];vinculos text[]:=ARRAY[]::text[];sys_count integer:=0;hash_campo record;
 acto text;recibo text;cert_acto text;aud record;resultados jsonb:='[]'::jsonb;resultado jsonb;
 ambitos jsonb;deadline timestamptz;ahora timestamptz(6);i integer:=0;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
 IF p_plan_canonico IS NULL OR pg_catalog.octet_length(p_plan_canonico) NOT BETWEEN 1 AND 65536
 OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=plan actual=invalido esperado=canon_V3_acotado_aprobado' USING ERRCODE='22023'; END IF;
 p:=p_plan_canonico::jsonb;
 IF vec_autorizacion.canon_bootstrap_central_admin_v3(p,'plan') IS DISTINCT FROM p_plan_canonico
 OR p->>'version' IS DISTINCT FROM '3' OR p->>'bootstrap_estado_esperado' IS DISTINCT FROM 'pendiente'
 OR pg_catalog.jsonb_array_length(p->'personas')<>2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=plan_canon actual=divergente esperado=ABI_V3_dos_personas' USING ERRCODE='22023'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_huella_aprobada OR sha IS DISTINCT FROM cfg.plan_sha256
 OR p->'fuente_reparto_aprobado' IS DISTINCT FROM cfg.fuente_reparto
 OR p->'fuente_identidad' IS DISTINCT FROM cfg.fuente_identidad
 OR p->'fuente_ca_admin' IS DISTINCT FROM cfg.fuente_ca_admin
 THEN RAISE EXCEPTION 'AUT37: PARO clave=aprobacion actual=no_coincidente esperado=plan_fuentes_aprobadas_del_LOGIN' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.bootstrap_central_admin_v3 WHERE singleton FOR SHARE;
 IF FOUND THEN
  IF previo.huella_plan_sha256 IS DISTINCT FROM sha OR previo.plan_canonico IS DISTINCT FROM pg_catalog.convert_to(p_plan_canonico,'UTF8')
  OR previo.operador_login IS DISTINCT FROM session_user::name OR previo.aprobacion_ref IS DISTINCT FROM cfg.aprobacion_ref
  OR previo.aprobacion_sha256 IS DISTINCT FROM cfg.aprobacion_sha256
  OR vec_autorizacion_atestada_v3.cotejar_acuse_bootstrap_central_admin_v3(previo.auditoria_ref,previo.auditoria_secuencia,previo.auditoria_huella_sha256,previo.auditoria_registrada_en,sha,previo.operador_login,previo.aprobacion_ref,previo.preimagen_sha256) IS NOT TRUE
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin
    WHERE control_id AND bootstrap_estado='consumido' AND bootstrap_acto_ref=previo.acto_ref)
  THEN RAISE EXCEPTION 'AUT37: PARO clave=replay actual=incompatible esperado=operador_aprobacion_plan_originales' USING ERRCODE='23505'; END IF;
  PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
  IF pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN
   RAISE EXCEPTION 'AUT40: PARO clave=replay_vigencia actual=caducada esperado=plan_vigente' USING ERRCODE='42501';
  END IF;
  RETURN previo.resultado;
 END IF;
 deadline:=LEAST(cfg.vigente_hasta,(p->>'caduca_en')::timestamptz);
 IF (p->>'preparado_en')::timestamptz>pg_catalog.clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz OR pg_catalog.clock_timestamp()>=deadline
 OR (p#>>'{personas,0,persona_ref}') COLLATE "C">=(p#>>'{personas,1,persona_ref}') COLLATE "C"
 OR p#>>'{personas,0,cuenta_ref}'=p#>>'{personas,1,cuenta_ref}'
 OR p#>>'{personas,0,certificado_admin,huella_sha256}'=p#>>'{personas,1,certificado_admin,huella_sha256}'
 THEN RAISE EXCEPTION 'AUT37: PARO clave=reparto actual=divergente esperado=dos_personas_independientes_y_vigencia' USING ERRCODE='22023'; END IF;
 FOR hash_campo IN SELECT e.key,e.value FROM pg_catalog.jsonb_path_query(p,'$.** ? (@.type() == "object")') o(objeto),
   LATERAL pg_catalog.jsonb_each(o.objeto) e
  WHERE e.key IN('huella_sha256','ca_huella_sha256','preimagen_huella_sha256','control_huella_sha256') LOOP
  IF pg_catalog.jsonb_typeof(hash_campo.value) IS DISTINCT FROM 'string'
  OR hash_campo.value#>>'{}' !~ '^[0-9a-f]{64}$' OR hash_campo.value#>>'{}'=pg_catalog.repeat('0',64)
  THEN RAISE EXCEPTION 'AUT37: PARO clave=huella_fuente actual=invalida esperado=SHA256_positivo' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  IF persona->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR persona->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
  OR persona->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR persona->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
  OR persona->>'perfil_ref'=ANY(perfiles) OR persona->>'vinculo_ref'=ANY(vinculos)
  OR persona#>>'{certificado_admin,persona_ref}' IS DISTINCT FROM persona->>'persona_ref'
  OR persona#>>'{certificado_admin,cuenta_ref}' IS DISTINCT FROM persona->>'cuenta_ref'
  OR persona#>>'{certificado_admin,ca_huella_sha256}' IS DISTINCT FROM p#>>'{fuente_ca_admin,huella_sha256}'
  OR (persona->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
  THEN RAISE EXCEPTION 'AUT37: PARO clave=persona actual=invalida esperado=identidad_refs_certificado_coherentes' USING ERRCODE='22023'; END IF;
  perfiles:=pg_catalog.array_append(perfiles,persona->>'perfil_ref');vinculos:=pg_catalog.array_append(vinculos,persona->>'vinculo_ref');
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(persona->'sistemas') LOOP
   sys_count:=sys_count+1;
   IF sistema->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR sistema->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
   OR sistema->>'perfil_ref'=ANY(perfiles) OR sistema->>'vinculo_ref'=ANY(vinculos)
   OR sistema#>>'{rol,version_ref}' IS NOT DISTINCT FROM p#>>'{rol,version_ref}'
   OR (sistema->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
   OR (sistema->>'vigente_hasta')::timestamptz>(persona->>'vigente_hasta')::timestamptz
   THEN RAISE EXCEPTION 'AUT37: PARO clave=sistemas actual=invalido esperado=perfil_separado_una_misma_persona' USING ERRCODE='22023'; END IF;
   perfiles:=pg_catalog.array_append(perfiles,sistema->>'perfil_ref');vinculos:=pg_catalog.array_append(vinculos,sistema->>'vinculo_ref');
  END LOOP;
 END LOOP;
 IF sys_count<>1
 THEN RAISE EXCEPTION 'AUT37: PARO clave=kit actual=cardinalidad_divergente esperado=dos_Aplicacion_un_Sistemas' USING ERRCODE='22023'; END IF;
 -- Todos los roles, fuentes y tres objetivos se acreditan antes del primer INSERT.
 preimagen:=vec_autorizacion.preimagen_bootstrap_central_admin_v3(p);
 preimagen_bytes:=pg_catalog.convert_to(preimagen::text,'UTF8');
 preimagen_sha:=pg_catalog.encode(pg_catalog.sha256(preimagen_bytes),'hex');
 IF preimagen_sha IS DISTINCT FROM cfg.preimagen_sha256
 THEN RAISE EXCEPTION 'AUT37: PARO clave=preimagen_aprobada actual=divergente esperado=estado_completo_aprobado' USING ERRCODE='40001'; END IF;
 -- Gobernanza existente se coteja; nunca se reemplaza ni alarga su historia.
 FOR gob IN SELECT value FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') LOOP
  SELECT * INTO config_rol FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=gob->>'version_ref' FOR SHARE;
  IF FOUND AND (config_rol.clase IS DISTINCT FROM gob->>'clase' OR config_rol.huella_sha256 IS DISTINCT FROM gob->>'huella_sha256'
   OR config_rol.vigente_desde IS DISTINCT FROM (gob->>'vigente_desde')::timestamptz
   OR config_rol.vigente_hasta IS DISTINCT FROM (gob->>'vigente_hasta')::timestamptz
   OR config_rol.unidad_requerida IS DISTINCT FROM (gob->>'unidad_requerida')::boolean
   OR config_rol.audiencia_administrativa IS DISTINCT FROM p#>>'{gobierno,audiencia_administrativa}'
   OR config_rol.ambitos_fijos IS DISTINCT FROM gob->'ambitos_fijos'
   OR config_rol.duracion_propuesta IS DISTINCT FROM pg_catalog.make_interval(secs=>(gob->>'duracion_propuesta_segundos')::double precision))
  THEN RAISE EXCEPTION 'AUT37: PARO clave=gobierno_existente actual=divergente esperado=preimagen_exacta_sin_sustitucion' USING ERRCODE='40001'; END IF;
 END LOOP;
 acto:='acto_admin:'||pg_catalog.substr(sha,1,32);recibo:='recibo_bootstrap:'||pg_catalog.substr(sha,33,32);
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(pg_catalog.jsonb_build_object(
  'tipo_registro','bootstrap_operador','evento_ref','evento_'||pg_catalog.substr(sha,1,32),'operador_login',session_user::text,
  'plan_sha256',sha,'aprobacion_ref',cfg.aprobacion_ref,'accion','ejecutar_plan_bootstrap_admin',
  'recurso_ref',acto,'resultado','permitido','motivo_ref','admin.bootstrap.plan_aprobado','proceso',cfg.proceso,
  'canal','operacion_tecnica_privada','finalidad_ref','bootstrap_admin','correlacion_ref','correlacion_'||pg_catalog.substr(sha,33,32),
  'fuente_ref','bootstrap_preimagen:'||pg_catalog.substr(sha,1,32),'fuente_sha256',preimagen_sha));
 FOR gob IN SELECT value FROM pg_catalog.jsonb_array_elements(p#>'{gobierno,roles}') LOOP
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=gob->>'version_ref') THEN
   INSERT INTO vec_autorizacion.rol_administrable_exacto_v1(version_rol_ref,clase,huella_sha256,vigente_desde,vigente_hasta,unidad_requerida,audiencia_administrativa,ambitos_fijos,duracion_propuesta)
   VALUES(gob->>'version_ref',gob->>'clase',gob->>'huella_sha256',(gob->>'vigente_desde')::timestamptz,(gob->>'vigente_hasta')::timestamptz,
    (gob->>'unidad_requerida')::boolean,p#>>'{gobierno,audiencia_administrativa}',gob->'ambitos_fijos',pg_catalog.make_interval(secs=>(gob->>'duracion_propuesta_segundos')::double precision));
  END IF;
 END LOOP;
 FOR persona IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'personas') LOOP
  i:=i+1;
  cert_acto:='acto_admin:'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(sha||':cert:'||i::text,'UTF8')),'hex'),1,32);
  PERFORM vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(persona,p->'gobierno',cert_acto);
  ambitos:=(vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(persona->'ambitos',(persona->>'vigente_hasta')::timestamptz))->'ambitos';
  resultados:=resultados||pg_catalog.jsonb_build_array(vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(persona,persona->>'perfil_ref',persona->>'vinculo_ref',p#>>'{rol,version_ref}',ambitos,(persona->>'vigente_hasta')::timestamptz,sha,aud.auditoria_ref,acto));
  FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(persona->'sistemas') LOOP
   ambitos:=(vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(sistema->'ambitos',(sistema->>'vigente_hasta')::timestamptz))->'ambitos';
   resultados:=resultados||pg_catalog.jsonb_build_array(vec_autorizacion.crear_asignacion_bootstrap_central_admin_v3(persona,sistema->>'perfil_ref',sistema->>'vinculo_ref',sistema#>>'{rol,version_ref}',ambitos,(sistema->>'vigente_hasta')::timestamptz,sha,aud.auditoria_ref,acto));
  END LOOP;
 END LOOP;
 ahora:=pg_catalog.clock_timestamp();
 UPDATE vec_autorizacion.control_continuidad_admin SET revision=revision+1,bootstrap_estado='consumido',bootstrap_acto_ref=acto,actualizado_en=ahora
 WHERE control_id AND revision=(p->>'control_continuidad_revision_esperada')::bigint AND bootstrap_estado='pendiente';
 IF NOT FOUND OR (SELECT pg_catalog.count(DISTINCT e.value->>'persona_ref') FROM pg_catalog.jsonb_array_elements(vec_autorizacion.administradores_aplicacion_efectivos_internos_v3()) e)<>2
 THEN RAISE EXCEPTION 'AUT37: PARO clave=CAS_final actual=divergente esperado=dos_personas_y_un_consumo' USING ERRCODE='40001'; END IF;
 PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
 IF pg_catalog.clock_timestamp()>=deadline
 THEN RAISE EXCEPTION 'AUT37: PARO clave=vigencia_final actual=caducada esperado=plan_configuracion_vigentes' USING ERRCODE='42501'; END IF;
 resultado:=pg_catalog.jsonb_build_object('acto_ref',acto,'recibo_ref',recibo,'huella_plan_sha256',sha,
  'primera_persona_ref',p#>>'{personas,0,persona_ref}','segunda_persona_ref',p#>>'{personas,1,persona_ref}',
  'auditoria_ref',aud.auditoria_ref,'perfiles',resultados,'confirmado_en',ahora);
 INSERT INTO vec_autorizacion.bootstrap_central_admin_v3(singleton,acto_ref,recibo_ref,plan_canonico,huella_plan_sha256,preimagen_canonica,preimagen_sha256,operador_login,aprobacion_ref,aprobacion_sha256,auditoria_ref,auditoria_secuencia,auditoria_huella_sha256,auditoria_registrada_en,resultado,confirmado_en)
 VALUES(true,acto,recibo,pg_catalog.convert_to(p_plan_canonico,'UTF8'),sha,preimagen_bytes,preimagen_sha,session_user::name,cfg.aprobacion_ref,cfg.aprobacion_sha256,aud.auditoria_ref,aud.secuencia,aud.huella_sha256,aud.registrada_en,resultado,ahora);
 INSERT INTO vec_autorizacion.outbox_bootstrap_central_admin_v3 VALUES(acto,'bootstrap_central_confirmado',aud.auditoria_ref,ahora);
 RETURN resultado;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'AUT37: PARO clave=fuente actual=ausente_o_ambigua esperado=fuentes_propietarias_exactas' USING ERRCODE='42501';
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(text,text) FROM PUBLIC,vec_admin_bootstrap_central_v3_ejecutor;
-- Se conserva la metadata pública y el contrato LOGIN de AUT37. El cliente
-- no elige resultado ni aporta referencias de efectos que aún no existen.
DO $envoltura$
DECLARE anterior jsonb;actual jsonb;f oid;
BEGIN
 f:=to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)');
 SELECT to_jsonb(p)-'prosrc' INTO STRICT anterior FROM pg_proc p WHERE p.oid=f;
 EXECUTE $def$
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_bootstrap_central_admin_v3(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC'
SET row_security=on SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;replay boolean:=false;
 solicitud text:='solicitud_bootstrap:'||replace(gen_random_uuid()::text,'-','');
 evento text:='evento_'||replace(gen_random_uuid()::text,'-','');
 correlacion text:='correlacion_'||replace(gen_random_uuid()::text,'-','');solicitud_sha text;
BEGIN
 IF p_plan_canonico IS NOT NULL AND octet_length(p_plan_canonico)>65536
 OR p_huella_aprobada IS NOT NULL AND octet_length(p_huella_aprobada)>64 THEN
  -- El cuerpo privado rechazará esta entrada; el intento compromete longitud
  -- y SHA, sin incluir material extenso en el propio registro de auditoría.
  solicitud_sha:=encode(pg_catalog.sha256(convert_to(jsonb_build_object(
   'plan_sha256',encode(pg_catalog.sha256(convert_to(COALESCE(p_plan_canonico,''),'UTF8')),'hex'),
   'plan_bytes',octet_length(p_plan_canonico),'aprobacion_sha256',encode(pg_catalog.sha256(convert_to(COALESCE(p_huella_aprobada,''),'UTF8')),'hex'))::text,'UTF8')),'hex');
 ELSE
  solicitud_sha:=encode(pg_catalog.sha256(convert_to(jsonb_build_object('plan',p_plan_canonico,'huella_aprobada',p_huella_aprobada)::text,'UTF8')),'hex');
 END IF;
 BEGIN
  -- Orden SERIALIZABLE y barrera únicos: evita atribuir replay a una primera
  -- confirmación concurrente. La misma transacción retiene el cerrojo.
  PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
  SELECT EXISTS(SELECT 1 FROM vec_autorizacion.bootstrap_central_admin_v3 WHERE singleton) INTO replay;
  respuesta:=vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(p_plan_canonico,p_huella_aprobada);
  PERFORM vec_autorizacion.exigir_operador_bootstrap_central_admin_v3();
  IF clock_timestamp()>=(p_plan_canonico::jsonb->>'caduca_en')::timestamptz THEN
   RAISE EXCEPTION 'AUT40: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
  motivo:=CASE WHEN replay THEN 'bootstrap_replay' ELSE 'bootstrap_registrado' END;
 EXCEPTION WHEN OTHERS THEN
  respuesta:=NULL;replay:=false;codigo:=SQLSTATE;
  estado:=CASE WHEN codigo IN('42501','22023','40001','23505','25000') THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'bootstrap_denegado' ELSE 'bootstrap_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'bootstrap_rechazado' ELSE 'bootstrap_no_disponible' END;
 END;
 SELECT * INTO aud FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(jsonb_build_object(
  'tipo_registro','intento_bootstrap_central_admin','evento_ref',evento,'operador_login',session_user::text,
  'solicitud_sha256',solicitud_sha,'accion','registrar_bootstrap_central_admin_v3','recurso_ref',solicitud,
  'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada',
  'finalidad_ref','bootstrap_admin','correlacion_ref',correlacion));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta,'replay',replay,
  'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,
   'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
$def$;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT actual FROM pg_proc p WHERE p.oid=f;
 IF actual IS DISTINCT FROM anterior THEN
  RAISE EXCEPTION 'AUT40: PARO clave=metadata_publica actual=alterada esperado=OID_firma_propietario_ACL_config_originales' USING ERRCODE='55000'; END IF;
END $envoltura$;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(text,text)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND (a.grantee<>'vec_autorizacion_propietario'::regrole OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'AUT40: PARO clave=ACL_efecto actual=ampliada esperado=owner_unico' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
