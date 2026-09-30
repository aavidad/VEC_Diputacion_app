\set ON_ERROR_STOP on
-- AD3-123. Misma autoridad Gobierno: publicación explícita EXTERNO y raíz propia.
-- Requiere AD3-112/116/118/121/122. No reescribe la publicación ni el checkpoint interno.
-- El publicador entrega los 17 consumidores externos; los lectores consumen su subconjunto.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000123',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_configuracion_externa') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-123: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE TABLE vec_autorizacion_atestada_v3.puntero_configuracion_externa (
 orden pg_catalog.numeric(20,0) PRIMARY KEY CHECK(orden BETWEEN 1 AND 9007199254740991),
 configuracion_revision pg_catalog.text NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.configuracion_confianza_version,
 establecida_en pg_catalog.timestamptz(6) NOT NULL,
 acto_ref pg_catalog.text NOT NULL UNIQUE,
 registrada_en pg_catalog.timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 huella_aprobacion_sha256 pg_catalog.text NOT NULL CHECK(vec_autorizacion_atestada_v3.huella_sha256_valida(huella_aprobacion_sha256)),
 preimagen_sha256 pg_catalog.text NOT NULL CHECK(vec_autorizacion_atestada_v3.huella_sha256_valida(preimagen_sha256)),
 material_publico pg_catalog.jsonb NOT NULL
);
CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision_externa (
 orden pg_catalog.numeric(20,0) PRIMARY KEY CHECK(orden BETWEEN 1 AND 9007199254740991),
 clave_id pg_catalog.text NOT NULL,
 version pg_catalog.numeric(20,0) NOT NULL UNIQUE,
 establecida_en pg_catalog.timestamptz(6) NOT NULL,
 acto_ref pg_catalog.text NOT NULL UNIQUE,
 registrada_en pg_catalog.timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(clave_id,version) REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version
);
CREATE TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno_externo (
 control_id pg_catalog.bool PRIMARY KEY DEFAULT true CHECK(control_id),
 revision pg_catalog.numeric(20,0) NOT NULL CHECK(revision BETWEEN 0 AND 9007199254740991),
 configuracion_secuencia_minima pg_catalog.numeric(20,0) NOT NULL CHECK(configuracion_secuencia_minima BETWEEN 0 AND 9007199254740991),
 raiz_version_minima pg_catalog.numeric(20,0) NOT NULL CHECK(raiz_version_minima BETWEEN 0 AND 9007199254740991),
 actualizada_en pg_catalog.timestamptz(6) NOT NULL
);
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno_externo VALUES(true,0,0,0,pg_catalog.clock_timestamp());
DO $proteccion$
DECLARE t pg_catalog.text;
BEGIN
 FOREACH t IN ARRAY ARRAY['puntero_configuracion_externa','puntero_clave_emision_externa','checkpoint_gobierno_externo'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I FOR ALL TO vec_autorizacion_atestada_v3_propietario USING(current_user=''vec_autorizacion_atestada_v3_propietario'') WITH CHECK(current_user=''vec_autorizacion_atestada_v3_propietario'')',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON vec_autorizacion_atestada_v3.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.%I FROM PUBLIC',t);
 END LOOP;
END $proteccion$;
CREATE TRIGGER no_mutar BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.puntero_clave_emision_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_mutar BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.puntero_configuracion_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();

CREATE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r pg_catalog.numeric; c pg_catalog.numeric;
BEGIN
 SELECT cr.raiz_version,cfg.secuencia INTO STRICT r,c
 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
 JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cfg ON cfg.revision=cr.configuracion_revision
 WHERE cr.configuracion_revision=NEW.configuracion_revision;
 UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno_externo
 SET revision=revision+1,configuracion_secuencia_minima=c,raiz_version_minima=r,actualizada_en=pg_catalog.clock_timestamp()
 WHERE control_id AND configuracion_secuencia_minima<c AND raiz_version_minima<=r;
 IF NOT FOUND THEN RAISE EXCEPTION 'AD3-123: retroceso externo' USING ERRCODE='42501'; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo() FROM PUBLIC;
CREATE TRIGGER checkpoint_despues AFTER INSERT ON vec_autorizacion_atestada_v3.puntero_configuracion_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo();

-- Preparación y publicación usan la autoridad existente, después de SET LOCAL ROLE.
-- ACL solo propietario; ningún emisor, consumidor o preflight puede ejecutar estas funciones.
CREATE FUNCTION vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1()
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE estado pg_catalog.jsonb; t pg_catalog.text; filas pg_catalog.jsonb; actual pg_catalog.jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD3-123: gobierno rechazado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Los máximos globales conservan las restricciones UNIQUE históricas. La
 -- preimagen minimizada también liga asociaciones, revocaciones y checkpoints.
 estado:='{}'::pg_catalog.jsonb;
 FOREACH t IN ARRAY ARRAY['clave_capacidad_version','puntero_clave_emision','puntero_clave_emision_externa','configuracion_confianza_version','raiz_confianza_version','configuracion_raiz','puntero_configuracion_actual','puntero_configuracion_externa','revocacion_clave_capacidad','revocacion_configuracion','revocacion_raiz','checkpoint_gobierno','checkpoint_gobierno_externo'] LOOP
  EXECUTE pg_catalog.format('SELECT coalesce(pg_catalog.jsonb_agg(f ORDER BY f::pg_catalog.text),''[]''::pg_catalog.jsonb) FROM (SELECT pg_catalog.to_jsonb(x)-''secreto_hmac'' AS f FROM vec_autorizacion_atestada_v3.%I x) q',t) INTO filas;
  estado:=estado||pg_catalog.jsonb_build_object(t,filas);
 END LOOP;
 SELECT material_publico INTO actual FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa ORDER BY orden DESC LIMIT 1;
 RETURN pg_catalog.jsonb_build_object(
 'preimagen_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(estado::pg_catalog.text,'UTF8')),'hex'),
 'configuracion_secuencia_siguiente',(SELECT coalesce(pg_catalog.max(secuencia),0)+1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),
 'clave_version_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'revision_gobierno_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'clave_orden_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'publicacion_externa_actual',actual);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.material_publico_externo_v1(m pg_catalog.jsonb)
RETURNS pg_catalog.jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.jsonb_build_object('configuracion',m->'configuracion','raiz',m->'raiz','claves',
 (SELECT pg_catalog.jsonb_agg(k-'secreto_hmac_hex' ORDER BY k->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') k))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.material_publico_externo_v1(pg_catalog.jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m pg_catalog.jsonb)
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; audiencias pg_catalog.text[]:=ARRAY[
 'vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1',
 'vec_usuarios.correos.consultar.externa_personal.v1','vec_usuarios.correos.anadir.externa_personal.v1','vec_usuarios.correos.reenviar.externa_personal.v1','vec_usuarios.correos.verificar.externa_personal.v1','vec_usuarios.correos.activar.externa_personal.v1','vec_usuarios.correos.retirar.externa_personal.v1',
 'vec_usuarios.imagen.consultar.externa_personal.v1','vec_usuarios.imagen.actualizar.externa_personal.v1',
 'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1','vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1','vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1','vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'];
BEGIN
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();
 IF pg_catalog.octet_length(m::pg_catalog.text)>262144 OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(m))<>3
 OR NOT(m ?& ARRAY['configuracion','raiz','claves']) THEN RAISE EXCEPTION 'AD3-123: material rechazado' USING ERRCODE='42501'; END IF;
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
 OR pg_catalog.jsonb_typeof(m->'claves') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(m->'claves')<>17
 THEN RAISE EXCEPTION 'AD3-123: material rechazado' USING ERRCODE='42501'; END IF;
 IF (SELECT pg_catalog.count(DISTINCT u.item->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>17
 THEN RAISE EXCEPTION 'AD3-123: audiencias rechazadas' USING ERRCODE='42501'; END IF;
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
  THEN RAISE EXCEPTION 'AD3-123: clave externa rechazada' USING ERRCODE='42501'; END IF;
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('preimagen_sha256',e->>'preimagen_sha256',
 'huella_aprobacion_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion_atestada_v3.material_publico_externo_v1(m)::pg_catalog.text,'UTF8')),'hex'));
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'AD3-123: propuesta externa rechazada' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(pg_catalog.jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m pg_catalog.jsonb,aprobacion pg_catalog.text,preimagen pg_catalog.text)
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; anterior pg_catalog.jsonb; acto pg_catalog.text; previa_aprobacion pg_catalog.text; previa_preimagen pg_catalog.text;
BEGIN
 e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
 IF aprobacion IS NULL OR preimagen IS NULL OR aprobacion IS DISTINCT FROM e->>'huella_aprobacion_sha256'
 THEN RAISE EXCEPTION 'AD3-123: aprobación o CAS rechazado' USING ERRCODE='42501'; END IF;
 c:=m->'configuracion';r:=m->'raiz';acto:='acto:externo:confianza:r'||(c->>'secuencia');
 SELECT material_publico,huella_aprobacion_sha256,preimagen_sha256 INTO anterior,previa_aprobacion,previa_preimagen FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa WHERE configuracion_revision=c->>'revision';
 IF FOUND THEN
  IF aprobacion IS DISTINCT FROM previa_aprobacion OR (preimagen IS DISTINCT FROM previa_preimagen AND preimagen IS DISTINCT FROM e->>'preimagen_sha256')
  OR anterior IS DISTINCT FROM vec_autorizacion_atestada_v3.material_publico_externo_v1(m)
  OR (SELECT configuracion_revision FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa ORDER BY orden DESC LIMIT 1) IS DISTINCT FROM c->>'revision'
  THEN RAISE EXCEPTION 'AD3-123: replay externo rechazado' USING ERRCODE='42501'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(anterior->'claves') x WHERE
   (SELECT (p.clave_id,p.version,p.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k ON (k.clave_id,k.version)=(p.clave_id,p.version) WHERE k.audiencia_consumo=x->>'audiencia_consumo' ORDER BY p.orden DESC LIMIT 1)
   IS DISTINCT FROM (x->>'clave_id',(x->>'version')::pg_catalog.numeric,(x->>'orden')::pg_catalog.numeric))
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo cp WHERE cp.control_id AND cp.configuracion_secuencia_minima=(c->>'secuencia')::pg_catalog.numeric AND cp.raiz_version_minima<=(r->>'version')::pg_catalog.numeric)
  THEN RAISE EXCEPTION 'AD3-123: replay de puntero rechazado' USING ERRCODE='42501'; END IF;
  RETURN anterior;
 END IF;
 IF preimagen IS DISTINCT FROM e->>'preimagen_sha256' THEN RAISE EXCEPTION 'AD3-123: CAS rechazado' USING ERRCODE='42501'; END IF;
 IF (c->>'secuencia')::pg_catalog.numeric<(SELECT configuracion_secuencia_minima+1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo WHERE control_id)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version WHERE revision=c->>'revision')
 OR (c->>'secuencia')::pg_catalog.numeric<>(vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1()->>'configuracion_secuencia_siguiente')::pg_catalog.numeric
 THEN RAISE EXCEPTION 'AD3-123: secuencia externa rechazada' USING ERRCODE='42501'; END IF;
 -- ON CONFLICT sólo permite el mismo material. Nunca sustituye una fila histórica.
 INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
 VALUES(r->>'clave_id',(r->>'version')::pg_catalog.numeric,pg_catalog.decode(r->>'clave_publica_spki_hex','hex'),r->>'huella_spki_sha256',(r->>'valida_desde')::pg_catalog.timestamptz,(r->>'valida_hasta')::pg_catalog.timestamptz,r->>'suite',r->>'audiencia_despliegue','acto:externo:raiz:'||(r->>'huella_spki_sha256')) ON CONFLICT(clave_id,version) DO NOTHING;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version x WHERE x.clave_id=r->>'clave_id' AND x.version=(r->>'version')::pg_catalog.numeric AND x.clave_publica_spki=pg_catalog.decode(r->>'clave_publica_spki_hex','hex') AND x.huella_spki_sha256=r->>'huella_spki_sha256' AND x.valida_desde=(r->>'valida_desde')::pg_catalog.timestamptz AND x.valida_hasta=(r->>'valida_hasta')::pg_catalog.timestamptz AND x.suite=r->>'suite' AND x.audiencia_despliegue=r->>'audiencia_despliegue') THEN RAISE EXCEPTION 'AD3-123: raíz histórica distinta' USING ERRCODE='42501'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_array_elements(m->'claves') x LOOP
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::pg_catalog.numeric,(k->>'revision_gobierno')::pg_catalog.numeric,k->>'huella_gobierno_sha256',pg_catalog.decode(k->>'secreto_hmac_hex','hex'),k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia_consumo',(k->>'valida_desde')::pg_catalog.timestamptz,(k->>'valida_hasta')::pg_catalog.timestamptz,'acto:externo:clave:r'||(k->>'revision_gobierno')) ON CONFLICT(clave_id,version) DO NOTHING;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' AND x.version=(k->>'version')::pg_catalog.numeric AND x.revision_gobierno=(k->>'revision_gobierno')::pg_catalog.numeric AND x.huella_gobierno_sha256=k->>'huella_gobierno_sha256' AND x.huella_secreto_sha256=k->>'huella_secreto_sha256' AND x.secreto_hmac=pg_catalog.decode(k->>'secreto_hmac_hex','hex') AND x.emisor_id=k->>'emisor_id' AND x.audiencia_consumo=k->>'audiencia_consumo' AND x.valida_desde=(k->>'valida_desde')::pg_catalog.timestamptz AND x.valida_hasta=(k->>'valida_hasta')::pg_catalog.timestamptz) THEN RAISE EXCEPTION 'AD3-123: clave histórica distinta' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision_externa(orden,clave_id,version,establecida_en,acto_ref)
  VALUES((k->>'orden')::pg_catalog.numeric,k->>'clave_id',(k->>'version')::pg_catalog.numeric,(c->>'publicada_en')::pg_catalog.timestamptz,'acto:externo:puntero-clave:r'||(k->>'orden')) ON CONFLICT(orden) DO NOTHING;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p WHERE p.orden=(k->>'orden')::pg_catalog.numeric AND p.clave_id=k->>'clave_id' AND p.version=(k->>'version')::pg_catalog.numeric)
  OR (SELECT (p.clave_id,p.version,p.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p JOIN vec_autorizacion_atestada_v3.clave_capacidad_version x ON (x.clave_id,x.version)=(p.clave_id,p.version) WHERE x.audiencia_consumo=k->>'audiencia_consumo' ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM (k->>'clave_id',(k->>'version')::pg_catalog.numeric,(k->>'orden')::pg_catalog.numeric) THEN RAISE EXCEPTION 'AD3-123: puntero de clave distinto' USING ERRCODE='42501'; END IF;
 END LOOP;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(c->>'revision',(c->>'secuencia')::pg_catalog.numeric,c->>'huella_configuracion_sha256',(c->>'publicada_en')::pg_catalog.timestamptz,(c->>'expira_en')::pg_catalog.timestamptz,acto);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES(c->>'revision',r->>'clave_id',(r->>'version')::pg_catalog.numeric);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_externa(orden,configuracion_revision,establecida_en,acto_ref,huella_aprobacion_sha256,preimagen_sha256,material_publico)
 VALUES((c->>'secuencia')::pg_catalog.numeric,c->>'revision',(c->>'publicada_en')::pg_catalog.timestamptz,acto||':puntero',aprobacion,preimagen,vec_autorizacion_atestada_v3.material_publico_externo_v1(m));
 RETURN vec_autorizacion_atestada_v3.material_publico_externo_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(pg_catalog.jsonb,pg_catalog.text,pg_catalog.text) FROM PUBLIC;

-- Cuerpos exactos posteriores a las correcciones de tipos AD3-121/122.
-- Se conservan OID, argumentos, ACL, propietario y toda comprobación de actor,
-- perfil, audiencias, revocaciones y recibos. Sólo cambia el ámbito de Gobierno.
DO $lectores$
DECLARE i record; antes pg_catalog.pg_proc%ROWTYPE; despues pg_catalog.pg_proc%ROWTYPE; fuente pg_catalog.text; definicion pg_catalog.text; a pg_catalog.text; b pg_catalog.text;
BEGIN
 FOR i IN SELECT * FROM pg_catalog.jsonb_to_recordset($inventario$[{"firma":"vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)","antes":"0ba180fec6ecd5328ab084e2b2e1cf3ec8f8cefec1fd603cf176e9741c5832e7","despues":"51837bd859b3025e34e903c5189bee45b954eaa2d62560a947e2122aa3b8d81b","punteros":1,"checkpoints":2,"claves":1,"config":["search_path=pg_catalog, pg_temp","lock_timeout=2s"],"acl":"{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_preflight_externo=X/vec_autorizacion_atestada_v3_propietario}","definidor":true},{"firma":"vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)","antes":"3e28e1daf77a874398e653fe46cdf30b1859c6209913914a16e68f484f43674c","despues":"9d29758f87172d72b6281a54b73d35e25223c8693ca2cc741b15ac052933025b","punteros":1,"checkpoints":2,"claves":1,"config":["search_path=pg_catalog, pg_temp","lock_timeout=2s"],"acl":"{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}","definidor":true},{"firma":"vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)","antes":"a823ac4f0406f8f718f3c37164217d95ba85af1c11b0f82c7e05df8ee872dae6","despues":"32a1f27e2cfa4c4181bb501408185fda6380f44721a3793b2fd97727080d66fc","punteros":1,"checkpoints":2,"claves":1,"config":["search_path=pg_catalog, pg_temp","lock_timeout=2s"],"acl":"{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}","definidor":true},{"firma":"vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)","antes":"f64428afaf5871119ff7a6c8e04bdda98616ca4a151c6e2bd23dd308d37bb3ee","despues":"a9f56c71a9e7f1e833a2279a56f9b59ca85133845d83fb10bc0f5c43f8256547","punteros":3,"checkpoints":2,"claves":1,"config":["search_path=pg_catalog, pg_temp","lock_timeout=2s"],"acl":"{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_atestada_v3_preflight_externo=X/vec_autorizacion_atestada_v3_propietario}","definidor":true}]$inventario$::pg_catalog.jsonb)
 AS x(firma pg_catalog.text,antes pg_catalog.text,despues pg_catalog.text,punteros pg_catalog.int4,checkpoints pg_catalog.int4,claves pg_catalog.int4,config pg_catalog.jsonb,acl pg_catalog.text,definidor pg_catalog.bool) LOOP
  SELECT p.* INTO STRICT antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(i.firma);
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM i.antes
  OR antes.proowner<>pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') OR antes.prosecdef IS DISTINCT FROM i.definidor
  OR pg_catalog.to_jsonb(antes.proconfig) IS DISTINCT FROM i.config OR antes.proacl::pg_catalog.text IS DISTINCT FROM i.acl
  THEN RAISE EXCEPTION 'AD3-123: lector incompatible %',i.firma USING ERRCODE='55000'; END IF;
  fuente:=antes.prosrc;
  a:='vec_autorizacion_atestada_v3.puntero_configuracion_actual';b:='vec_autorizacion_atestada_v3.puntero_configuracion_externa';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*i.punteros THEN RAISE EXCEPTION 'AD3-123: punteros incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  a:='vec_autorizacion_atestada_v3.checkpoint_gobierno';b:='vec_autorizacion_atestada_v3.checkpoint_gobierno_externo';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*i.checkpoints THEN RAISE EXCEPTION 'AD3-123: checkpoints incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  a:='vec_autorizacion_atestada_v3.puntero_clave_emision';b:='vec_autorizacion_atestada_v3.puntero_clave_emision_externa';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*i.claves THEN RAISE EXCEPTION 'AD3-123: claves incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM i.despues THEN RAISE EXCEPTION 'AD3-123: postimagen incompatible' USING ERRCODE='55000'; END IF;
  definicion:=pg_catalog.pg_get_functiondef(antes.oid);
  IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,antes.prosrc,'')))<>pg_catalog.length(antes.prosrc) THEN RAISE EXCEPTION 'AD3-123: definición incompatible' USING ERRCODE='55000'; END IF;
  EXECUTE pg_catalog.replace(definicion,antes.prosrc,fuente);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF despues.prosrc IS DISTINCT FROM fuente OR pg_catalog.to_jsonb(despues)-'prosrc' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'prosrc' THEN RAISE EXCEPTION 'AD3-123: metadatos alterados' USING ERRCODE='55000'; END IF;
 END LOOP;
END $lectores$;
COMMIT;
