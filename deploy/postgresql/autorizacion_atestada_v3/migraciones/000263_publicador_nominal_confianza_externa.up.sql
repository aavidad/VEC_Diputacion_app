\set ON_ERROR_STOP on
-- AD263: preparación administrativa nominal de la confianza externa (HZ38).
-- Dependencias: copia post-HZ37, AD140 y AD258; no altera los punteros internos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD263: PARO clave=instalador actual=incompatible esperado=PG18_superusuario' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)')),'UTF8')),'hex') INTO actual;
 IF actual IS DISTINCT FROM 'dcae955459b4fcd76122e660f8466402415fda3689c6a78b4f5f3565a0e3e5f0'
 THEN RAISE EXCEPTION 'AD263: PARO clave=preparacion actual=% esperado=dcae955459b4fcd76122e660f8466402415fda3689c6a78b4f5f3565a0e3e5f0',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF actual IS DISTINCT FROM 'df178e53fb5afa2e18efa30e8d535f5d2d1df1316afa9a51b814bd712190a0b3'
 THEN RAISE EXCEPTION 'AD263: PARO clave=familia actual=% esperado=df178e53fb5afa2e18efa30e8d535f5d2d1df1316afa9a51b814bd712190a0b3',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_publicador_externo')
 OR to_regprocedure('vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD263: PARO clave=instalacion actual=presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_autorizacion_atestada_v3_publicador_externo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v3_publicador_externo',current_database());
END $conexion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- Definición literal extraída de la copia principal post-HZ37. Solo añade las
-- seis audiencias externas con consumidor real, manteniendo todas sus guardas.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; audiencias pg_catalog.text[]:=ARRAY[
 'vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1',
 'vec_usuarios.correos.consultar.externa_personal.v1','vec_usuarios.correos.anadir.externa_personal.v1','vec_usuarios.correos.reenviar.externa_personal.v1','vec_usuarios.correos.verificar.externa_personal.v1','vec_usuarios.correos.activar.externa_personal.v1','vec_usuarios.correos.retirar.externa_personal.v1',
 'vec_usuarios.imagen.consultar.externa_personal.v1','vec_usuarios.imagen.actualizar.externa_personal.v1',
 'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1','vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1','vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1','vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1','vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1','vec_bolsa_llamamientos.inscripcion.presentar.v1','vec_bolsa_llamamientos.alegacion.presentar.v1','vec.bolsa.preferencias-bolsas.ordenar.v1','vec_meritos.cartera.retirar_propio.v1','vec_meritos.hecho.declarar.v1'];
BEGIN
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();
 IF pg_catalog.octet_length(m::pg_catalog.text)>262144 OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(m))<>3
 OR NOT(m ?& ARRAY['configuracion','raiz','claves']) THEN RAISE EXCEPTION 'AD3-140: material rechazado' USING ERRCODE='42501'; END IF;
 c:=m->'configuracion';r:=m->'raiz';
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
 OR NOT(c ?& ARRAY['revision','secuencia','huella_configuracion_sha256','publicada_en','expira_en'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(c) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='secuencia' THEN 'number' ELSE 'string' END)
 OR pg_catalog.jsonb_typeof(r) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(r))<>8
 OR NOT(r ?& ARRAY['clave_id','version','clave_publica_spki_hex','huella_spki_sha256','valida_desde','valida_hasta','suite','audiencia_despliegue'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(r) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='version' THEN 'number' ELSE 'string' END)
 OR NOT pg_catalog.starts_with(c->>'revision','confianza:atestacion:externo:')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(c->>'huella_configuracion_sha256')
 OR (c->>'secuencia')!~'^[1-9][0-9]{0,15}$'
 OR (r->>'version')!~'^[1-9][0-9]{0,15}$'
 OR r->>'clave_id' IS DISTINCT FROM 'clave:atestacion:externo:'||(r->>'huella_spki_sha256')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(r->>'huella_spki_sha256')
 OR (r->>'clave_publica_spki_hex')!~'^302a300506032b6570032100[0-9a-f]{64}$'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(r->>'clave_publica_spki_hex','hex')),'hex') IS DISTINCT FROM r->>'huella_spki_sha256'
 OR r->>'suite' IS DISTINCT FROM 'VEC-AD-3-COSE-EDDSA-1'
 OR r->>'audiencia_despliegue' IS DISTINCT FROM 'vec:desarrollo:contratacion-temporal:atestacion:v3'
 OR NOT pg_catalog.isfinite((c->>'publicada_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((c->>'expira_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_desde')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_hasta')::pg_catalog.timestamptz)
 OR (c->>'publicada_en')::pg_catalog.timestamptz>pg_catalog.clock_timestamp()
 OR (c->>'expira_en')::pg_catalog.timestamptz<=pg_catalog.clock_timestamp()
 OR (r->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
 OR (r->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.puntero_configuracion_actual p ON p.configuracion_revision=cr.configuracion_revision WHERE (cr.raiz_clave_id,cr.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version x WHERE x.huella_spki_sha256=r->>'huella_spki_sha256' AND (x.clave_id,x.version) IS DISTINCT FROM (r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE (x.raiz_clave_id,x.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=c->>'revision')
 OR pg_catalog.jsonb_typeof(m->'claves') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(m->'claves')<>23
 THEN RAISE EXCEPTION 'AD3-140: material rechazado' USING ERRCODE='42501'; END IF;
 IF (SELECT pg_catalog.count(DISTINCT u.item->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>23
 THEN RAISE EXCEPTION 'AD3-140: audiencias rechazadas' USING ERRCODE='42501'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_array_elements(m->'claves') x LOOP
  IF pg_catalog.jsonb_typeof(k) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(k))<>11
  OR NOT(k ?& ARRAY['clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac_hex','huella_secreto_sha256','emisor_id','audiencia_consumo','valida_desde','valida_hasta','orden'])
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(k) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key=ANY(ARRAY['version','revision_gobierno','orden']) THEN 'number' ELSE 'string' END)
  OR k->>'audiencia_consumo'<>ALL(audiencias)
  OR NOT vec_autorizacion_atestada_v3.texto_tecnico_valido(k->>'clave_id',512)
  OR NOT pg_catalog.starts_with(k->>'emisor_id','emisor:externo:')
  OR (k->>'version')!~'^[1-9][0-9]{0,15}$' OR (k->>'revision_gobierno')!~'^[1-9][0-9]{0,15}$' OR (k->>'orden')!~'^[1-9][0-9]{0,15}$'
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_gobierno_sha256')
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_secreto_sha256')
  OR (k->>'secreto_hmac_hex')!~'^[0-9a-f]+$' OR pg_catalog.length(k->>'secreto_hmac_hex') NOT BETWEEN 64 AND 8192 OR pg_catalog.length(k->>'secreto_hmac_hex')%2<>0
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(k->>'secreto_hmac_hex','hex')),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR (k->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
  OR (k->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
  OR NOT pg_catalog.isfinite((k->>'valida_desde')::pg_catalog.timestamptz) OR NOT pg_catalog.isfinite((k->>'valida_hasta')::pg_catalog.timestamptz)
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad x WHERE (x.clave_id,x.version)=(k->>'clave_id',(k->>'version')::pg_catalog.numeric))
  THEN RAISE EXCEPTION 'AD3-140: clave externa rechazada' USING ERRCODE='42501'; END IF;
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('preimagen_sha256',e->>'preimagen_sha256',
 'huella_aprobacion_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion_atestada_v3.material_publico_externo_v1(m)::pg_catalog.text,'UTF8')),'hex'));
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'AD3-140: propuesta externa rechazada' USING ERRCODE='42501';
END $function$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.auditoria_confianza_externa_valida_v1(j jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(
 j->>'tipo_registro' IN ('consulta_confianza_externa_admin','publicacion_confianza_externa_admin')
 AND (jsonb_strip_nulls(j)-ARRAY[
 'auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en',
 'tipo_registro','evento_ref','evento_material_sha256','operador_login','actor_ref',
 'perfil_activo_ref','plan_sha256','aprobacion_ref','accion','modulo_id','recurso_ref',
 'finalidad_ref','resultado','motivo_ref','proceso','canal','correlacion_ref'])='{}'::jsonb
 AND j->>'evento_ref' ~ '^evento_[0-9a-f]{32}$'
 AND j->>'evento_material_sha256' ~ '^[0-9a-f]{64}$'
 AND j->>'plan_sha256' ~ '^[0-9a-f]{64}$'
 AND j->>'operador_login' ~ '^vec_confianza_externa_publicador_[a-z0-9_]{1,31}$'
 AND j->>'actor_ref'=j->>'operador_login'
 AND j->>'perfil_activo_ref'='vec_autorizacion_atestada_v3_publicador_externo'
 AND j->>'aprobacion_ref'='publicacion_externa_cli'
 AND ((j->>'tipo_registro'='consulta_confianza_externa_admin' AND j->>'accion'='leer_estado_publicacion_externa_v1' AND j->>'motivo_ref'='preimagen_comprobada')
 OR (j->>'tipo_registro'='publicacion_confianza_externa_admin' AND j->>'accion'='publicar_confianza_externa_v1' AND j->>'motivo_ref' IN ('confianza_publicada','confianza_replay')))
 AND j->>'modulo_id'='autorizacion'
 AND j->>'recurso_ref' ~ '^confianza_externa:[0-9a-f]{32}$'
 AND j->>'finalidad_ref'='preparar_portal_externo'
 AND j->>'resultado'='permitido'
 AND j->>'proceso'='postgresql' AND j->>'canal'='operacion_tecnica_privada'
 AND j->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$',false)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.auditoria_confianza_externa_valida_v1(jsonb) FROM PUBLIC;
DO $familia$
DECLARE anterior text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||
 left(anterior,length(anterior)-1)||' OR vec_autorizacion_atestada_v3.auditoria_confianza_externa_valida_v1(to_jsonb(auditoria_consumo_v3)))';
END $familia$;

-- No concede al LOGIN ni SET ROLE ni permisos sobre tablas o funciones AD140.
-- La identidad del operador procede de session_user, nunca del JSON del cliente.
CREATE FUNCTION vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE r record; e jsonb; salida jsonb; replay boolean; tipo text; accion text; motivo text;
 evento text; material text; msha text; seq numeric; anterior text; instante timestamptz(6); ref text; huella text; plan text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR session_user::text !~ '^vec_confianza_externa_publicador_[a-z0-9_]{1,31}$'
 THEN RAISE EXCEPTION 'AD263: PARO clave=operador actual=incompatible esperado=publicador_nominal_serializable_UTC' USING ERRCODE='42501'; END IF;
 SELECT x.* INTO STRICT r FROM pg_roles x WHERE x.rolname=session_user;
 IF NOT r.rolcanlogin OR NOT r.rolinherit OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls
 OR (SELECT count(*) FROM pg_auth_members a WHERE a.member=r.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members a JOIN pg_roles g ON g.oid=a.roleid WHERE a.member=r.oid
 AND g.rolname='vec_autorizacion_atestada_v3_publicador_externo' AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option
 AND NOT g.rolcanlogin AND NOT g.rolsuper AND NOT g.rolcreatedb AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members superior WHERE superior.member=g.oid))
 OR has_database_privilege(session_user,current_database(),'TEMP')
 THEN RAISE EXCEPTION 'AD263: PARO clave=membresia actual=incompatible esperado=grupo_unico_sin_SET_ADMIN_TEMP' USING ERRCODE='42501'; END IF;
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();
 IF m IS NULL THEN
  salida:=e;plan:=e->>'preimagen_sha256';tipo:='consulta_confianza_externa_admin';accion:='leer_estado_publicacion_externa_v1';motivo:='preimagen_comprobada';
 ELSE
  -- Las claves externas tampoco pueden repetir un secreto activo interno.
  IF octet_length(m::text)>262144 OR jsonb_typeof(m->'claves') IS DISTINCT FROM 'array'
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'claves') k
    WHERE k->>'clave_id' NOT LIKE 'clave:capacidad:externo:%'
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
     JOIN vec_autorizacion_atestada_v3.clave_capacidad_version v USING(clave_id,version)
     WHERE v.huella_secreto_sha256=k->>'huella_secreto_sha256'))
  THEN RAISE EXCEPTION 'AD263: PARO clave=separacion actual=incompatible esperado=claves_externas_propias' USING ERRCODE='42501'; END IF;
  replay:=(e->'publicacion_externa_actual'->'configuracion'->>'revision') IS NOT DISTINCT FROM (m->'configuracion'->>'revision');
  e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
  salida:=vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,e->>'huella_aprobacion_sha256',e->>'preimagen_sha256');
  plan:=e->>'huella_aprobacion_sha256';tipo:='publicacion_confianza_externa_admin';accion:='publicar_confianza_externa_v1';
  motivo:=CASE WHEN replay THEN 'confianza_replay' ELSE 'confianza_publicada' END;
 END IF;
 -- Un asiento común por acción, incluso recuperación. No contiene secretos.
 evento:='evento_'||replace(gen_random_uuid()::text,'-','');
 material:=jsonb_build_object('operador_login',session_user,'tipo',tipo,'plan_sha256',plan,'motivo',motivo,'evento_ref',evento)::text;
 msha:=encode(sha256(convert_to(material,'UTF8')),'hex');
 SELECT x.secuencia_previa,x.anterior_sha256 INTO STRICT seq,anterior FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() x;
 IF seq>=9007199254740991::numeric THEN RAISE EXCEPTION 'AD263: secuencia agotada' USING ERRCODE='22003'; END IF;
 seq:=seq+1;instante:=clock_timestamp();ref:='aud_v3_ext_'||substr(evento,8,32);
 huella:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.confianza-externa.v1')||
 vec_autorizacion_atestada_v3.encuadrar_mac(seq::text)||vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||
 vec_autorizacion_atestada_v3.encuadrar_mac(ref)||vec_autorizacion_atestada_v3.encuadrar_mac(msha)||
 vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
 evento_ref,evento_material_sha256,operador_login,actor_ref,perfil_activo_ref,plan_sha256,aprobacion_ref,accion,modulo_id,recurso_ref,
 finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(ref,seq,anterior,huella,instante,tipo,evento,msha,session_user,session_user,'vec_autorizacion_atestada_v3_publicador_externo',plan,
 'publicacion_externa_cli',accion,'autorizacion','confianza_externa:'||substr(plan,1,32),'preparar_portal_externo','permitido',motivo,
 'postgresql','operacion_tecnica_privada','correlacion_'||substr(evento,8,32));
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_atestada_v3_publicador_externo;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb) TO vec_autorizacion_atestada_v3_publicador_externo;
COMMIT;
