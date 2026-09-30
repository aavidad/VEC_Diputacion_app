\set ON_ERROR_STOP on
-- Sólo clon desechable PG18 con la cadena AD3 previa instalada. Todos los
-- valores de esta prueba son sintéticos y los efectos terminan en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='30s';
-- Captura independiente de OID y metadatos antes de cualquier efecto AD135.
CREATE TEMP TABLE ad135_catalogo_pre AS SELECT pg_catalog.to_jsonb(p) AS fila
 FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace;
-- Nombres de tipos del usuario más cercano: no deben dominar el catálogo.
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(false);
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK(false);
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(false);
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(false);
-- La sonda compila la normalización completa; rechaza alias PL/pgSQL ambiguos.
-- AD135-CORRECTIVA-AQUI
RESET ROLE;
DO $conservacion$
DECLARE fila_pre pg_catalog.jsonb; funcion_actual pg_catalog.pg_proc%ROWTYPE; original pg_catalog.text;
 lector pg_catalog.bool; modificados pg_catalog.int4:=0; nuevos pg_catalog.int4;
BEGIN
 FOR fila_pre IN SELECT fila FROM ad135_catalogo_pre LOOP
  SELECT * INTO STRICT funcion_actual FROM pg_catalog.pg_proc WHERE oid=(fila_pre->>'oid')::pg_catalog.oid;
  lector:=funcion_actual.proname=ANY(ARRAY['comprobar_material_emision_externa_v1','consumir_decision_mutacion_v3_externa_interna','consumir_decision_mutacion_v3_usuarios_externa_interna','leer_configuracion_externa_v1']);
  IF pg_catalog.to_jsonb(funcion_actual)-'prosrc' IS DISTINCT FROM fila_pre-'prosrc'
  THEN RAISE EXCEPTION 'metadatos/OID históricos alterados'; END IF;
  IF lector THEN
   original:=pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(funcion_actual.prosrc,
    'vec_autorizacion_atestada_v3.puntero_configuracion_externa','vec_autorizacion_atestada_v3.puntero_configuracion_actual'),
    'vec_autorizacion_atestada_v3.checkpoint_gobierno_externo','vec_autorizacion_atestada_v3.checkpoint_gobierno'),
    'vec_autorizacion_atestada_v3.puntero_clave_emision_externa','vec_autorizacion_atestada_v3.puntero_clave_emision');
   IF original IS DISTINCT FROM fila_pre->>'prosrc' OR funcion_actual.prosrc IS NOT DISTINCT FROM fila_pre->>'prosrc'
   THEN RAISE EXCEPTION 'linaje del lector perdido'; END IF;
   modificados:=modificados+1;
  ELSIF funcion_actual.prosrc IS DISTINCT FROM fila_pre->>'prosrc' THEN
   RAISE EXCEPTION 'núcleo interno alterado';
  END IF;
 END LOOP;
 SELECT pg_catalog.count(*) INTO nuevos FROM pg_catalog.pg_proc catalogo_funcion
 WHERE catalogo_funcion.pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace
 AND NOT EXISTS(SELECT 1 FROM ad135_catalogo_pre catalogo_pre WHERE (catalogo_pre.fila->>'oid')::pg_catalog.oid=catalogo_funcion.oid);
 IF modificados<>4 OR nuevos<>5 THEN RAISE EXCEPTION 'efectos AD135 incompletos'; END IF;
 -- AD134/136 siguen presentes en el núcleo completo; cuatro lectores no lo acreditan.
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc nucleo
 WHERE nucleo.oid=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
 AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nucleo.prosrc,'UTF8')),'hex')='91f93a2695862b677c282ce6d0c4972148fe170e38540c8cfe3751f0ad5e4fb5')
 THEN RAISE EXCEPTION 'núcleo RPT y custodia incompleto'; END IF;
END $conservacion$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TEMP TABLE ad135_fixture(nombre pg_catalog.text PRIMARY KEY,material pg_catalog.jsonb);
GRANT SELECT ON ad135_fixture TO vec_externo_preflight_v3_desarrollo;
CREATE FUNCTION pg_temp.propuesta_externa(g pg_catalog.int4) RETURNS pg_catalog.jsonb LANGUAGE plpgsql AS $f$
DECLARE e pg_catalog.jsonb;c pg_catalog.jsonb;r pg_catalog.jsonb;ks pg_catalog.jsonb;pk pg_catalog.text;h pg_catalog.text;n pg_catalog.numeric;
BEGIN
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();n:=(e->>'clave_version_siguiente')::pg_catalog.numeric;
 pk:='302a300506032b6570032100'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('fixture-ad135-root-'||g,'UTF8')),'hex');
 h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(pk,'hex')),'hex');
 r:=pg_catalog.jsonb_build_object('clave_id','clave:atestacion:externo:'||h,'version',1,'clave_publica_spki_hex',pk,'huella_spki_sha256',h,'valida_desde','2026-09-01T00:00:00Z','valida_hasta','2030-01-01T00:00:00Z','suite','VEC-AD-3-COSE-EDDSA-1','audiencia_despliegue','vec:desarrollo:contratacion-temporal:atestacion:v3');
 c:=pg_catalog.jsonb_build_object('revision','confianza:atestacion:externo:fixture-'||g,'secuencia',e->'configuracion_secuencia_siguiente','huella_configuracion_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('fixture-ad135-config-'||g,'UTF8')),'hex'),'publicada_en','2026-09-01T00:00:00Z','expira_en','2029-01-01T00:00:00Z');
 SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object('clave_id','clave:capacidad:externo:fixture:'||g||':'||i,'version',n+i-1,'revision_gobierno',n+i-1,'orden',n+i-1,'huella_gobierno_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('fixture-ad135-governance-'||g||':'||i,'UTF8')),'hex'),'secreto_hmac_hex',sh,'huella_secreto_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(sh,'hex')),'hex'),'emisor_id','emisor:externo:fixture-'||g,'audiencia_consumo',a,'valida_desde','2026-09-01T00:00:00Z','valida_hasta','2030-01-01T00:00:00Z') ORDER BY i) INTO ks FROM (
 SELECT a,i,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('fixture-ad135-dummy-'||g||':'||i,'UTF8')),'hex') sh FROM pg_catalog.unnest(ARRAY[
 'vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1',
 'vec_usuarios.correos.consultar.externa_personal.v1','vec_usuarios.correos.anadir.externa_personal.v1','vec_usuarios.correos.reenviar.externa_personal.v1','vec_usuarios.correos.verificar.externa_personal.v1','vec_usuarios.correos.activar.externa_personal.v1','vec_usuarios.correos.retirar.externa_personal.v1',
 'vec_usuarios.imagen.consultar.externa_personal.v1','vec_usuarios.imagen.actualizar.externa_personal.v1',
 'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1','vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1','vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1','vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1']) WITH ORDINALITY u(a,i)) q;
 RETURN pg_catalog.jsonb_build_object('configuracion',c,'raiz',r,'claves',ks);
END $f$;
CREATE FUNCTION pg_temp.preflight_externo(m pg_catalog.jsonb) RETURNS pg_catalog.jsonb LANGUAGE sql AS $f$
 SELECT pg_catalog.jsonb_build_object('configuracion',(m->'configuracion')-'publicada_en'-'expira_en','raiz',(m->'raiz')-'clave_publica_spki_hex'-'valida_desde'-'valida_hasta','claves',(SELECT pg_catalog.jsonb_agg(k-'secreto_hmac_hex'-'valida_desde'-'valida_hasta'-'orden' ORDER BY i) FROM pg_catalog.jsonb_array_elements(m->'claves') WITH ORDINALITY u(k,i) WHERE k->>'audiencia_consumo' IN ('vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1')))
$f$;
DO $publicar$
DECLARE m pg_catalog.jsonb;e pg_catalog.jsonb;r pg_catalog.jsonb;cp pg_catalog.jsonb;interno pg_catalog.jsonb;mutado pg_catalog.jsonb;n pg_catalog.int8;
BEGIN
 SELECT pg_catalog.to_jsonb(x) INTO cp FROM vec_autorizacion_atestada_v3.checkpoint_gobierno x;
 SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY orden) INTO interno FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual x;
 INSERT INTO ad135_fixture VALUES('cp_interno',cp),('puntero_interno',interno);
 m:=pg_temp.propuesta_externa(1);e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,pg_catalog.repeat('f',64),e->>'preimagen_sha256');RAISE EXCEPTION 'aceptó aprobación ajena';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,e->>'huella_aprobacion_sha256',pg_catalog.repeat('f',64));RAISE EXCEPTION 'aceptó CAS ajeno';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 mutado:=pg_catalog.jsonb_set(m,'{raiz,clave_id}','null'::pg_catalog.jsonb);
 BEGIN PERFORM vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(mutado);RAISE EXCEPTION 'aceptó raíz null';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 r:=vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,e->>'huella_aprobacion_sha256',e->>'preimagen_sha256');
 IF r::pg_catalog.text LIKE '%secreto_hmac%' THEN RAISE EXCEPTION 'recibo contiene secreto';END IF;
 IF vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,e->>'huella_aprobacion_sha256',e->>'preimagen_sha256') IS DISTINCT FROM r THEN RAISE EXCEPTION 'replay cambió recibo';END IF;
 SELECT pg_catalog.count(*) INTO n FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa;IF n<>1 THEN RAISE EXCEPTION 'replay duplicado';END IF;
 IF (SELECT pg_catalog.to_jsonb(x) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno x) IS DISTINCT FROM cp OR (SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(x) ORDER BY orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual x) IS DISTINCT FROM interno THEN RAISE EXCEPTION 'publicación externa alteró interno';END IF;
 mutado:=pg_catalog.jsonb_set(m,'{configuracion}',(pg_temp.propuesta_externa(99)->'configuracion'));
 mutado:=pg_catalog.jsonb_set(mutado,'{claves,0,orden}',m#>'{claves,1,orden}');
 mutado:=pg_catalog.jsonb_set(mutado,'{claves,1,orden}',m#>'{claves,0,orden}');
 e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(mutado);
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(mutado,e->>'huella_aprobacion_sha256',e->>'preimagen_sha256');RAISE EXCEPTION 'aceptó órdenes cruzados';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 INSERT INTO ad135_fixture VALUES('m1',m),('sonda1',pg_temp.preflight_externo(m));
END $publicar$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda1'));
DO $publicador_cruzado$ BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();RAISE EXCEPTION 'externo accedió publicador';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $publicador_cruzado$;
SELECT vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda1')) IS NOT NULL AS lectura_externa_ok;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $rotacion$
DECLARE m pg_catalog.jsonb;e pg_catalog.jsonb;cp pg_catalog.jsonb;interno pg_catalog.jsonb;revision_previa pg_catalog.text;s pg_catalog.numeric;raiz record;
BEGIN
 SELECT material INTO cp FROM ad135_fixture WHERE nombre='cp_interno';SELECT material INTO interno FROM ad135_fixture WHERE nombre='puntero_interno';
 -- Renovación del puntero INTERNO primero: misma raíz y claves, otra config.
 SELECT p.configuracion_revision INTO revision_previa FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p ORDER BY p.orden DESC LIMIT 1;
 SELECT * INTO STRICT raiz FROM vec_autorizacion_atestada_v3.configuracion_raiz WHERE configuracion_revision=revision_previa;
 SELECT pg_catalog.max(secuencia)+1 INTO s FROM vec_autorizacion_atestada_v3.configuracion_confianza_version;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 SELECT 'confianza:fixture:interna:ad135:'||s,s,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('internal-renewal-ad135-'||s,'UTF8')),'hex'),publicada_en,expira_en,'acto:fixture:interno:ad135:'||s FROM vec_autorizacion_atestada_v3.configuracion_confianza_version WHERE revision=revision_previa;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES('confianza:fixture:interna:ad135:'||s,raiz.raiz_clave_id,raiz.raiz_version);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref) VALUES(s,'confianza:fixture:interna:ad135:'||s,'2026-09-01T00:00:00Z','acto:fixture:puntero-interno:ad135:'||s);
 SELECT pg_catalog.to_jsonb(x) INTO cp FROM vec_autorizacion_atestada_v3.checkpoint_gobierno x;
 m:=pg_temp.propuesta_externa(2);e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
 PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,e->>'huella_aprobacion_sha256',e->>'preimagen_sha256');
 IF (SELECT pg_catalog.to_jsonb(x) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno x) IS DISTINCT FROM cp THEN RAISE EXCEPTION 'rotación externa alteró checkpoint interno';END IF;
 IF (SELECT configuracion_secuencia_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo)<>(m#>>'{configuracion,secuencia}')::pg_catalog.numeric THEN RAISE EXCEPTION 'checkpoint externo no avanzó';END IF;
 INSERT INTO ad135_fixture VALUES('m2',m),('sonda2',pg_temp.preflight_externo(m));
END $rotacion$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
DO $vieja$
BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda1'));RAISE EXCEPTION 'aceptó raíz anterior';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $vieja$;
RESET SESSION AUTHORIZATION;
DO $acl$
DECLARE p record;t record;
BEGIN
 FOR p IN SELECT * FROM pg_catalog.pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace AND proname IN ('leer_estado_publicacion_externa_v1','preparar_publicacion_externa_v1','publicar_confianza_externa_v1','material_publico_externo_v1','avanzar_checkpoint_externo') LOOP
  IF EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE a.grantee<>p.proowner) THEN RAISE EXCEPTION 'publicador expuesto';END IF;
 END LOOP;
 FOR t IN SELECT * FROM pg_catalog.pg_class WHERE relnamespace='vec_autorizacion_atestada_v3'::pg_catalog.regnamespace AND relname IN ('puntero_clave_emision_externa','puntero_configuracion_externa','checkpoint_gobierno_externo') LOOP
  IF NOT t.relrowsecurity OR NOT t.relforcerowsecurity OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(t.relacl,pg_catalog.acldefault('r',t.relowner))) a WHERE a.grantee<>t.relowner) THEN RAISE EXCEPTION 'tabla externa expuesta';END IF;
 END LOOP;
END $acl$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $audiencias$
DECLARE m pg_catalog.jsonb;
BEGIN
 SELECT material INTO m FROM ad135_fixture WHERE nombre='m2';
 m:=pg_catalog.jsonb_set(m,'{claves,1}',m#>'{claves,0}');
 BEGIN PERFORM vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
  RAISE EXCEPTION 'aceptó audiencias duplicadas'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT material INTO m FROM ad135_fixture WHERE nombre='m2';
 m:=pg_catalog.jsonb_set(m,'{claves}',(m->'claves')-16);
 BEGIN PERFORM vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
  RAISE EXCEPTION 'aceptó 16 audiencias'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $audiencias$;
RESET ROLE;

SAVEPOINT revocacion_configuracion;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion(configuracion_revision,revocada_en,motivo_catalogado_ref,acto_ref)
 SELECT material#>>'{configuracion,revision}',pg_catalog.clock_timestamp(),'motivo:fixture:ad135:revocacion','acto:fixture:ad135:configuracion' FROM ad135_fixture WHERE nombre='m2';
DO $replay_revocado$
DECLARE m pg_catalog.jsonb; a pg_catalog.text; p pg_catalog.text;
BEGIN
 SELECT material INTO m FROM ad135_fixture WHERE nombre='m2';
 SELECT huella_aprobacion_sha256,preimagen_sha256 INTO STRICT a,p
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa
 WHERE configuracion_revision=m#>>'{configuracion,revision}';
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,a,p);
  RAISE EXCEPTION 'aceptó replay revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $replay_revocado$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
DO $lectura_revocada$
BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'aceptó material revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'leyó configuración revocada'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $lectura_revocada$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT revocacion_configuracion;
RELEASE SAVEPOINT revocacion_configuracion;

SAVEPOINT revocacion_raiz;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz(raiz_clave_id,raiz_version,revocada_en,motivo_catalogado_ref,acto_ref)
 SELECT material#>>'{raiz,clave_id}',(material#>>'{raiz,version}')::pg_catalog.numeric,pg_catalog.clock_timestamp(),'motivo:fixture:ad135:revocacion','acto:fixture:ad135:raiz' FROM ad135_fixture WHERE nombre='m2';
DO $replay_revocado$
DECLARE m pg_catalog.jsonb; a pg_catalog.text; p pg_catalog.text;
BEGIN
 SELECT material INTO m FROM ad135_fixture WHERE nombre='m2';
 SELECT huella_aprobacion_sha256,preimagen_sha256 INTO STRICT a,p
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa
 WHERE configuracion_revision=m#>>'{configuracion,revision}';
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,a,p);
  RAISE EXCEPTION 'aceptó replay revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $replay_revocado$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
DO $lectura_revocada$
BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'aceptó material revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'leyó configuración revocada'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $lectura_revocada$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT revocacion_raiz;
RELEASE SAVEPOINT revocacion_raiz;

SAVEPOINT revocacion_clave;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad(clave_id,version,revocada_en,motivo_catalogado_ref,acto_ref)
 SELECT material#>>'{claves,0,clave_id}',(material#>>'{claves,0,version}')::pg_catalog.numeric,pg_catalog.clock_timestamp(),'motivo:fixture:ad135:revocacion','acto:fixture:ad135:clave' FROM ad135_fixture WHERE nombre='m2';
DO $replay_revocado$
DECLARE m pg_catalog.jsonb; a pg_catalog.text; p pg_catalog.text;
BEGIN
 SELECT material INTO m FROM ad135_fixture WHERE nombre='m2';
 SELECT huella_aprobacion_sha256,preimagen_sha256 INTO STRICT a,p
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa
 WHERE configuracion_revision=m#>>'{configuracion,revision}';
 BEGIN PERFORM vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m,a,p);
  RAISE EXCEPTION 'aceptó replay revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $replay_revocado$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
DO $lectura_revocada$
BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'aceptó material revocado'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('usuarios_preferencias',(SELECT material FROM ad135_fixture WHERE nombre='sonda2'));
  RAISE EXCEPTION 'leyó configuración revocada'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $lectura_revocada$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT revocacion_clave;
RELEASE SAVEPOINT revocacion_clave;
ROLLBACK;
\echo AD135-RAIZ-LINAJE-CONSERVACION-OK
