\set ON_ERROR_STOP on
-- AUT94: reutiliza el rol portal v1 inmutable publicado antes de la provisión externa.
-- Literales de principal13b + las 19 SQL HZ37. AUT88 ya conserva el nombre
-- entre versiones ADMIN de preferencias/inscripción/alegaciones; no se pierde ese cuerpo.
-- Sólo el documento v1 con huella11044c69… admite el nombre heredado.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $verificacion$
DECLARE f record; esperado record; actual text;
BEGIN
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','1aefc467337e3c68bd305058670be6314df12e40523e464394137068e840d1ba','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_publicador_candidato_externo=X/vec_autorizacion_propietario}'),
  ('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)','bd3e88b25146ae031c4ffb73137a8ba8727d9b291231bcfd33f3a9da52be3bee','{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'),
  ('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)','fe6f52d34cfb010dcbd8875230cb4b4e6f6d7ed3613fad88b724f267ddedf0e1','{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 ) AS v(firma,huella,acl) LOOP
  SELECT p.*, pg_get_userbyid(p.proowner) AS propietario INTO f FROM pg_proc p
  WHERE p.oid=to_regprocedure(esperado.firma);
  actual:=encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(esperado.firma)),'UTF8')),'hex');
  IF actual IS DISTINCT FROM esperado.huella
  THEN RAISE EXCEPTION 'AUT94: definición % actual=% esperada=%', esperado.firma,actual,esperado.huella USING ERRCODE='55000'; END IF;
  IF f.propietario IS DISTINCT FROM 'vec_autorizacion_propietario'
  THEN RAISE EXCEPTION 'AUT94: propietario % actual=% esperado=vec_autorizacion_propietario',esperado.firma,f.propietario USING ERRCODE='55000'; END IF;
  IF f.proacl::text IS DISTINCT FROM esperado.acl
  THEN RAISE EXCEPTION 'AUT94: ACL % actual=% esperada=%',esperado.firma,f.proacl::text,esperado.acl USING ERRCODE='55000'; END IF;
  IF NOT (f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
  THEN RAISE EXCEPTION 'AUT94: search_path % actual=% esperado=pg_catalog, pg_temp',esperado.firma,f.proconfig USING ERRCODE='55000'; END IF;
 END LOOP;
END $verificacion$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion.publicar_rol_candidato_externo_v1(p_rol_canonico bytea, p_rol_huella text, p_control_canonico bytea, p_control_huella text, p_revision_esperada numeric, p_control_huella_esperada text, p_actualizada_por text, p_acto_ref text)
 RETURNS TABLE(version_rol_ref text, version bigint, revision numeric, huella_rol text, huella_control text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; c jsonb; ref text; id text; v bigint; rev numeric;
        existente record; actual record; x jsonb; acciones text[]; esperadas text[];
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_rol_canonico IS NULL OR pg_catalog.octet_length(p_rol_canonico) NOT BETWEEN 1 AND 65536
    OR p_control_canonico IS NULL OR pg_catalog.octet_length(p_control_canonico) NOT BETWEEN 1 AND 16384
    OR p_rol_huella !~ '^[0-9a-f]{64}$' OR p_control_huella !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_rol_canonico),'hex') IS DISTINCT FROM p_rol_huella
    OR pg_catalog.encode(pg_catalog.sha256(p_control_canonico),'hex') IS DISTINCT FROM p_control_huella
    OR p_revision_esperada IS NULL OR p_revision_esperada<0 OR p_revision_esperada<>trunc(p_revision_esperada)
    OR (p_revision_esperada=0 AND p_control_huella_esperada IS NOT NULL)
    OR (p_revision_esperada>0 AND p_control_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación de rol externo rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_rol_canonico,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_control_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento de rol externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'rol_id'; v:=(d->>'version')::bigint;
 ref:='rol:'||id||':v'||v::text; rev:=(c->>'revision')::numeric;
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' AND v IS DISTINCT FROM 1
 THEN RAISE EXCEPTION 'AUT77: versión portal reservada al acto ADMIN' USING ERRCODE='42501';END IF;
 IF id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR vec_autorizacion.rol_candidato_externo_acotado_v1(d) IS NOT TRUE
    OR v IS NULL OR v<1 OR v=9223372036854775807
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR (d->>'nombre' IS DISTINCT FROM (CASE WHEN id='candidato_bolsa_portal_historial_propio_desarrollo'
       THEN 'areaPersonal.miBolsa.rolPortal' ELSE 'Consulta propia de bolsa en desarrollo' END)
      AND (id='candidato_bolsa_portal_historial_propio_desarrollo' AND d->>'version'='1'
       AND d->>'nombre'='Consulta y acciones propias de bolsa en desarrollo'
       AND encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex')
        ='11044c697a11418870dc21794a9cb570505e017090ca8d767ad316851ec34b15') IS NOT TRUE)
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM 'null'::jsonb AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR d ?| ARRAY['retirada_por','retirada_ref','motivo_retirada_codigo']
    OR c->>'version_rol_ref' IS DISTINCT FROM ref
    OR c->>'estado' IS DISTINCT FROM 'habilitada'
    OR rev IS DISTINCT FROM p_revision_esperada+1
    OR c->>'actualizado_por' IS DISTINCT FROM p_actualizada_por
    OR c ?| ARRAY['acto_ref','motivo_codigo']
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'rol externo inválido' USING ERRCODE='42501'; END IF;
 esperadas:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar'];
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' THEN
  esperadas:=esperadas||ARRAY[
   'bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto'];
 END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RAISE EXCEPTION 'concesiones de rol externo incompatibles' USING ERRCODE='42501'; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x) k WHERE k NOT IN
      ('accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','obligaciones','garantia_minima'))<>0
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'bolsa' OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR x->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN x->>'accion'='bolsa.participaciones_propias.manifestar_disposicion' THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END)
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[CASE
      WHEN x->>'accion'='bolsa.historial_propio.consultar' THEN 'consulta_historial_propio'
      WHEN x->>'accion'='bolsa.participaciones_propias.consultar' THEN 'consulta_participaciones_propias'
      ELSE 'gestion_participaciones_propias' END])
    OR (x->>'accion'='bolsa.participaciones_propias.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["participaciones_candidato_minimizadas"]'::jsonb)
    OR (x->>'accion'='bolsa.historial_propio.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb)
    OR (x->>'accion' NOT IN ('bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar')
      AND COALESCE(x->'campos_permitidos','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb)
    OR COALESCE(x->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
  THEN RAISE EXCEPTION 'concesión externa no acotada' USING ERRCODE='42501'; END IF;
 END LOOP;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:rol_candidato_externo:'||id,0));
 SELECT r.version_rol_ref,r.rol_id,r.version,r.huella_sha256,r.documento INTO existente
 FROM vec_autorizacion.version_rol r WHERE r.version_rol_ref=ref FOR SHARE;
 IF FOUND THEN
  IF existente.rol_id IS DISTINCT FROM id OR existente.version IS DISTINCT FROM v
     OR existente.huella_sha256 IS DISTINCT FROM p_rol_huella OR existente.documento IS DISTINCT FROM d
  THEN RAISE EXCEPTION 'rol externo publicado divergente' USING ERRCODE='40001'; END IF;
 ELSE
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES(ref,id,v,p_rol_huella,(d->>'publicada_en')::timestamptz,d);
 END IF;
 SELECT a.revision,h.huella_sha256 INTO actual
 FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_revision_esperada<>0 OR p_control_huella_esperada IS NOT NULL))
    OR (FOUND AND (actual.revision IS DISTINCT FROM p_revision_esperada
      OR actual.huella_sha256 IS DISTINCT FROM p_control_huella_esperada))
 THEN RAISE EXCEPTION 'control de rol externo: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(ref,rev,'habilitada',p_control_huella,(c->>'actualizado_en')::timestamptz,c);
 IF p_revision_esperada=0 THEN
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
  VALUES(ref,rev,(c->>'actualizado_en')::timestamptz,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.control_vigencia_version_rol_actual AS puntero SET revision=rev,
   actualizada_en=(c->>'actualizado_en')::timestamptz,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.version_rol_ref=ref AND puntero.revision=p_revision_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'control de rol externo cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('rol_control',p_acto_ref,ref,rev,p_control_huella_esperada,p_control_huella,p_actualizada_por);
 RETURN QUERY SELECT ref,v,rev,p_rol_huella,p_control_huella;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.rol_candidato_externo_acotado_v1(d jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE id text; x jsonb; acciones text[]; esperadas text[];
BEGIN
 id:=d->>'rol_id';
 IF pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM 'null'::jsonb AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR NOT d ?& ARRAY['rol_id','version','nombre','estado','concesiones','publicada_por','publicada_en']
    OR (d->>'nombre' IS DISTINCT FROM (CASE WHEN id='candidato_bolsa_portal_historial_propio_desarrollo'
       THEN 'areaPersonal.miBolsa.rolPortal' ELSE 'Consulta propia de bolsa en desarrollo' END)
      AND (id='candidato_bolsa_portal_historial_propio_desarrollo' AND d->>'version'='1'
       AND d->>'nombre'='Consulta y acciones propias de bolsa en desarrollo'
       AND encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex')
        ='11044c697a11418870dc21794a9cb570505e017090ca8d767ad316851ec34b15') IS NOT TRUE)
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 esperadas:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar'];
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' THEN
  esperadas:=esperadas||ARRAY['bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto'];
 END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RETURN false; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x) k WHERE k NOT IN
      ('accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','obligaciones','garantia_minima'))<>0
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'bolsa' OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR x->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN x->>'accion'='bolsa.participaciones_propias.manifestar_disposicion' THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END)
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[CASE
      WHEN x->>'accion'='bolsa.historial_propio.consultar' THEN 'consulta_historial_propio'
      WHEN x->>'accion'='bolsa.participaciones_propias.consultar' THEN 'consulta_participaciones_propias'
      ELSE 'gestion_participaciones_propias' END])
    OR (x->>'accion'='bolsa.participaciones_propias.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["participaciones_candidato_minimizadas"]'::jsonb)
    OR (x->>'accion'='bolsa.historial_propio.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb)
    OR (x->>'accion' NOT IN ('bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar')
      AND COALESCE(x->'campos_permitidos','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb)
    OR COALESCE(x->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(p jsonb, p_primera boolean)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET row_security TO 'on'
AS $function$
DECLARE catalogo jsonb;entrada jsonb;seleccion jsonb;base jsonb;nueva jsonb;
 cabeza record;rol record;control record;puntero_control record;asignacion record;
 t jsonb;original jsonb;vistas text[]:=ARRAY[]::text[];
 acciones text[]:=ARRAY['bolsa.preferencias_bolsas.consultar_propias',
  'bolsa.preferencias_bolsas.ordenar_propias'];
 tipos text[]:=ARRAY['participaciones_candidato','participaciones_candidato'];
 finalidades text[]:=ARRAY['gestion_preferencias_bolsas_propias',
  'gestion_preferencias_bolsas_propias'];
 hojas text[];normalizadas jsonb:='[]'::jsonb;concesion jsonb;ahora timestamptz;
 i integer;campos_actuales jsonb;campos_esperados jsonb;ambitos jsonb;
 limite_entradas timestamptz:='infinity'::timestamptz;
 origen_inscripcion boolean:=false;origen_alegacion boolean:=false;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_externa_v1(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 OR (jsonb_array_length(p->'selecciones')=2 AND p#>>'{base,rol,version}' NOT IN ('1','2','3'))
 OR (jsonb_array_length(p->'selecciones')=5 AND p#>>'{base,rol,version}' NOT IN ('1','2'))
 THEN RAISE EXCEPTION 'AUT77: version o base invalida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';ahora:=clock_timestamp();
 origen_inscripcion:=jsonb_array_length(p->'selecciones')=5;
 origen_alegacion:=jsonb_array_length(p->'selecciones')=2 AND base#>>'{rol,version}'='3';
 IF origen_inscripcion THEN
  acciones:=ARRAY['bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
   'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar','bolsa.inscripcion.presentar'];
  tipos:=ARRAY['convocatoria_inscripcion','convocatoria_inscripcion','solicitud_inscripcion',
   'solicitud_inscripcion','inscripcion_convocatoria'];
  finalidades:=ARRAY['consulta_convocatoria_abierta','consulta_convocatoria_abierta',
   'consulta_inscripcion_propia','consulta_inscripcion_propia','presentar_inscripcion'];
 ELSIF origen_alegacion THEN
  acciones:=ARRAY['bolsa.alegacion.propias.consultar','bolsa.inscripcion.alegacion.presentar'];
  tipos:=ARRAY['alegaciones_propias','alegacion_bolsa'];
  finalidades:=ARRAY['consulta_alegaciones_propias','presentacion_alegacion_propia'];
 END IF;
 IF p_primera THEN
  SELECT * INTO cabeza FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR cabeza.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR cabeza.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT77: cabeza de catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 -- El catálogo gobernado acredita las dos entradas exactas.
 SELECT convert_from(x.catalogo_canon,'UTF8')::jsonb INTO STRICT catalogo
 FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256') x;
 IF p_primera AND (ahora<(catalogo->>'vigente_desde')::timestamptz
 OR (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
  AND ahora>=(catalogo->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT77: catalogo no vigente' USING ERRCODE='42501';END IF;
 FOR i IN 0..cardinality(acciones)-1 LOOP
  seleccion:=p->'selecciones'->i;
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(seleccion,'seleccion');
  SELECT value INTO entrada FROM jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT77: descriptor ausente' USING ERRCODE='42501';END IF;
  concesion:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_build_object('concesiones',jsonb_build_array(entrada->'concesion')))#>'{concesiones,0}';
  IF entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
  OR (origen_inscripcion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:inscripcion_recorrido_20261013')
  OR (origen_inscripcion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_inscripcion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '07bf028039044c91a23f09a2d5d5fe0d2af254e2fe85aa7fbbacea7c473487fe')
  OR (origen_alegacion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:alegaciones_propias_v1')
  OR (origen_alegacion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_alegacion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '5b9f0fc2d99f4c2cf7b08829bb56bcddd5262c5cb44cc797b48a3f1f072a1060')
  OR entrada#>>'{concesion,accion}' IS DISTINCT FROM acciones[i+1]
  OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
  OR entrada#>>'{concesion,tipo_recurso}' IS DISTINCT FROM tipos[i+1]
  OR entrada#>'{concesion,finalidades}' IS DISTINCT FROM jsonb_build_array(finalidades[i+1])
  OR entrada#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
  OR NOT (coalesce(entrada#>'{concesion,obligaciones}','[]'::jsonb)='[]'::jsonb
    OR (NOT origen_inscripcion AND i=0 AND entrada#>'{concesion,obligaciones}'='["auditar"]'::jsonb))
  OR entrada->'dimensiones_ambito' IS DISTINCT FROM '["candidato_ref"]'::jsonb
  OR entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
  OR concesion IS DISTINCT FROM nueva->'concesiones'->
    (jsonb_array_length(base#>'{rol,concesiones}')+i)
  THEN RAISE EXCEPTION 'AUT77: descriptor o concesion divergente %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF origen_inscripcion THEN
   hojas:=CASE WHEN i=0 THEN ARRAY['convocatorias[].catalogo_version','convocatorias[].categorias[].categoria','convocatorias[].categorias[].categoria_ref','convocatorias[].convocatoria_ref','convocatorias[].estado_solicitud_propia','convocatorias[].impedimento_etiqueta','convocatorias[].numero_categorias','convocatorias[].plazo_fin','convocatorias[].plazo_inicio','convocatorias[].puede_iniciar','convocatorias[].requisitos[].codigo','convocatorias[].requisitos[].descripcion','convocatorias[].requisitos[].estado','convocatorias[].requisitos[].hito_cumplimiento','convocatorias[].requisitos[].hito_etiqueta','convocatorias[].requisitos[].hito_fecha','convocatorias[].requisitos[].motivo_codigo','convocatorias[].requisitos[].motivo_etiqueta','convocatorias[].requisitos[].obligatorio','convocatorias[].requisitos[].procedencia_ref','convocatorias[].requisitos_resumen','convocatorias[].solicitud_ref','convocatorias[].titulo','cursor_siguiente','total']::text[] WHEN i=1 THEN ARRAY['catalogo_version','categorias[].categoria','categorias[].categoria_ref','convocatoria_ref','estado_solicitud_propia','impedimento_etiqueta','numero_categorias','plazo_fin','plazo_inicio','puede_iniciar','requisitos[].codigo','requisitos[].descripcion','requisitos[].estado','requisitos[].hito_cumplimiento','requisitos[].hito_etiqueta','requisitos[].hito_fecha','requisitos[].motivo_codigo','requisitos[].motivo_etiqueta','requisitos[].obligatorio','requisitos[].procedencia_ref','requisitos_resumen','solicitud_ref','titulo']::text[] WHEN i=2 THEN ARRAY['cursor_siguiente','solicitudes[].bases_ref','solicitudes[].bolsa_ref','solicitudes[].catalogo_version','solicitudes[].categoria','solicitudes[].categoria_ref','solicitudes[].convocatoria_ref','solicitudes[].convocatoria_titulo','solicitudes[].decidida_en','solicitudes[].decision_ref','solicitudes[].declaracion_ref','solicitudes[].estado','solicitudes[].motivo_codigo','solicitudes[].motivo_etiqueta','solicitudes[].participacion_ref','solicitudes[].plazo_fin','solicitudes[].plazo_inicio','solicitudes[].recibo_ref','solicitudes[].registrada_en','solicitudes[].requisitos[].codigo','solicitudes[].requisitos[].descripcion','solicitudes[].requisitos[].estado','solicitudes[].requisitos[].hito_cumplimiento','solicitudes[].requisitos[].hito_etiqueta','solicitudes[].requisitos[].hito_fecha','solicitudes[].requisitos[].motivo_codigo','solicitudes[].requisitos[].motivo_etiqueta','solicitudes[].requisitos[].obligatorio','solicitudes[].requisitos[].procedencia_ref','solicitudes[].solicitud_ref','solicitudes[].version','total']::text[] WHEN i=3 THEN ARRAY['bases_ref','bolsa_ref','catalogo_version','categoria','categoria_ref','convocatoria_ref','convocatoria_titulo','decidida_en','decision_ref','declaracion_ref','estado','motivo_codigo','motivo_etiqueta','participacion_ref','plazo_fin','plazo_inicio','recibo_ref','registrada_en','requisitos[].codigo','requisitos[].descripcion','requisitos[].estado','requisitos[].hito_cumplimiento','requisitos[].hito_etiqueta','requisitos[].hito_fecha','requisitos[].motivo_codigo','requisitos[].motivo_etiqueta','requisitos[].obligatorio','requisitos[].procedencia_ref','solicitud_ref','version']::text[] WHEN i=4 THEN ARRAY[]::text[] END;
  ELSIF origen_alegacion THEN
  hojas:=CASE WHEN i=0 THEN ARRAY['alegaciones[].alegacion_ref','alegaciones[].auditoria_ref','alegaciones[].estado','alegaciones[].fundamento','alegaciones[].objeto.acto_ref','alegaciones[].objeto.participacion_ref','alegaciones[].objeto.solicitud_ref','alegaciones[].objeto.tipo','alegaciones[].objeto.version_acto','alegaciones[].presentada_en','alegaciones[].recibo_ref','cursor_siguiente','objetos[].acto_ref','objetos[].bolsa_ref','objetos[].convocatoria_ref','objetos[].estado','objetos[].ocurrido_en','objetos[].participacion_ref','objetos[].solicitud_ref','objetos[].tipo','objetos[].titulo','objetos[].version_acto','total']::text[] ELSE ARRAY[]::text[] END;
  ELSE
  hojas:=CASE WHEN i=0 THEN ARRAY['version','orden_guardado','bolsas_actuales',
   'bolsas_nuevas','bolsas_retiradas','revision_requerida','recibo_ref','registrado_en']::text[]
   ELSE ARRAY[]::text[] END;
  END IF;
  IF coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb) IS DISTINCT FROM to_jsonb(hojas)
  THEN RAISE EXCEPTION 'AUT77: campos fuera de contrato %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF p_primera AND (ahora<(entrada->>'vigente_desde')::timestamptz
   OR (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND ahora>=(entrada->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT77: descriptor caducado %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
   limite_entradas:=LEAST(limite_entradas,(entrada->>'vigente_hasta')::timestamptz);
  END IF;
  normalizadas:=normalizadas||jsonb_build_array(concesion);
 END LOOP;
 IF catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
  limite_entradas:=LEAST(limite_entradas,(catalogo->>'vigente_hasta')::timestamptz);
 END IF;
 SELECT * INTO rol FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:candidato_bolsa_portal_historial_propio_desarrollo:v'||(base#>>'{rol,version}')
  FOR SHARE;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 -- AUT94: la base v1 heredada sólo se admite con su documento exacto.
 OR (rol.version=1 AND rol.documento->>'nombre' IS DISTINCT FROM 'areaPersonal.miBolsa.rolPortal'
  AND (rol.documento->>'nombre'='Consulta y acciones propias de bolsa en desarrollo'
   AND rol.huella_sha256='11044c697a11418870dc21794a9cb570505e017090ca8d767ad316851ec34b15') IS NOT TRUE)
 OR (rol.version=1 AND rol.documento->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa')
 OR rol.version::text IS DISTINCT FROM (base#>>'{rol,version}')
 OR rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(rol.documento) IS DISTINCT FROM base->'rol'
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.rol_id=rol.rol_id AND z.version>rol.version))
 THEN RAISE EXCEPTION 'AUT77: preimagen del rol divergente' USING ERRCODE='40001';END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=rol.version_rol_ref
 AND x.revision=(base#>>'{control_vigencia,revision}')::numeric FOR SHARE;
 IF NOT FOUND OR control.estado IS DISTINCT FROM 'habilitada'
 OR control.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control.documento),'UTF8')),'hex')
 OR control.documento IS DISTINCT FROM base->'control_vigencia'
 THEN RAISE EXCEPTION 'AUT77: control o perfil catalogado divergente' USING ERRCODE='40001';END IF;
 -- El cierre nuevo retiene el puntero base hasta COMMIT. Una retirada o
 -- avance concurrente no puede intercalarse entre cotejo y efecto.
 IF p_primera THEN
  SELECT q.revision INTO puntero_control
  FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=control.version_rol_ref FOR SHARE;
  IF NOT FOUND OR puntero_control.revision IS DISTINCT FROM control.revision
  THEN RAISE EXCEPTION 'AUT77: control base ya no actual' USING ERRCODE='40001';END IF;
 END IF;
 IF nueva->>'nombre' IS DISTINCT FROM base#>>'{rol,nombre}'
 OR nueva->'concesiones' IS DISTINCT FROM (base#>'{rol,concesiones}')||normalizadas
 OR vec_autorizacion.concesiones_positivas_validas(nueva) IS NOT TRUE
 OR jsonb_array_length(base#>'{rol,concesiones}')<>
   (CASE WHEN rol.version=1 THEN 7 WHEN rol.version=3 THEN 14 WHEN origen_inscripcion THEN 9 ELSE 12 END)
 OR (rol.version=2 AND origen_inscripcion AND NOT (base#>'{rol,concesiones}' @>
    '[{"accion":"bolsa.preferencias_bolsas.consultar_propias"},
      {"accion":"bolsa.preferencias_bolsas.ordenar_propias"}]'::jsonb))
 OR (rol.version=2 AND NOT origen_inscripcion AND NOT (base#>'{rol,concesiones}' @>
    '[{"accion":"bolsa.inscripcion.convocatorias.listar"},
      {"accion":"bolsa.inscripcion.convocatoria.consultar"},
      {"accion":"bolsa.inscripcion.propias.listar"},
      {"accion":"bolsa.inscripcion.propia.consultar"},
      {"accion":"bolsa.inscripcion.presentar"}]'::jsonb))
 OR (rol.version=3 AND NOT ((base#>'{rol,concesiones}') @>
    '[{"accion":"bolsa.preferencias_bolsas.consultar_propias"},
      {"accion":"bolsa.preferencias_bolsas.ordenar_propias"},
      {"accion":"bolsa.inscripcion.convocatorias.listar"},
      {"accion":"bolsa.inscripcion.convocatoria.consultar"},
      {"accion":"bolsa.inscripcion.propias.listar"},
      {"accion":"bolsa.inscripcion.propia.consultar"},
      {"accion":"bolsa.inscripcion.presentar"}]'::jsonb))
 OR NOT ((base#>'{rol,concesiones}') @> '[{"accion":"bolsa.historial_propio.consultar"},
  {"accion":"bolsa.participaciones_propias.consultar"},
  {"accion":"bolsa.participaciones_propias.solicitar_pausa"},
  {"accion":"bolsa.participaciones_propias.solicitar_reactivacion"},
  {"accion":"bolsa.participaciones_propias.responder_llamamiento"},
  {"accion":"bolsa.participaciones_propias.manifestar_disposicion"},
  {"accion":"bolsa.participaciones_propias.confirmar_contacto"}]'::jsonb)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(base#>'{rol,concesiones}') x
   WHERE x->>'accion'=ANY(acciones))
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=p->>'version_rol_objetivo_ref'))
 THEN RAISE EXCEPTION 'AUT77: delta de dos altas divergente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
  IF t->>'asignacion_ref'=ANY(vistas)
  THEN RAISE EXCEPTION 'AUT77: asignacion duplicada' USING ERRCODE='22023';END IF;
  vistas:=array_append(vistas,t->>'asignacion_ref');original:=t->'documento';
  ambitos:=original->'ambitos';
  IF t->>'asignacion_ref' IS DISTINCT FROM 'asignacion:'||(original->>'asignacion_id')||':v'||(original->>'version')
  OR t->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(original),'UTF8')),'hex')
  OR original->>'version_rol_ref' IS DISTINCT FROM ('rol:'||(base#>>'{rol,rol_id}')||':v'||(base#>>'{rol,version}'))
  OR original->>'estado' IS DISTINCT FROM 'activa'
  OR vec_autorizacion.ambitos_positivos_validos(original) IS NOT TRUE
  OR jsonb_typeof(ambitos) IS DISTINCT FROM 'array' OR jsonb_array_length(ambitos)<>1
  OR ambitos->0->>'clave' IS DISTINCT FROM 'candidato_ref'
  OR jsonb_typeof(ambitos#>'{0,valores}') IS DISTINCT FROM 'array'
  OR jsonb_array_length(ambitos#>'{0,valores}')<>1
  OR (ambitos#>>'{0,valores,0}' ~ '^can_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (p_primera AND vec_contexto_actor_v1.acreditar_candidato_externo_v1(
   original->>'principal_id',original->>'perfil_activo_ref',ambitos#>>'{0,valores,0}') IS NOT TRUE)
  THEN RAISE EXCEPTION 'AUT77: asignacion o ambitos invalidos' USING ERRCODE='22023';END IF;
  SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_externa x
   WHERE x.asignacion_ref=t->>'asignacion_ref' FOR SHARE;
  IF NOT FOUND OR asignacion.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
  OR asignacion.documento IS DISTINCT FROM original
  OR asignacion.principal_id IS DISTINCT FROM original->>'principal_id'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM original->>'perfil_activo_ref'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=asignacion.version_rol_ref AND z.rol_id=rol.rol_id AND z.version<=rol.version)
  THEN RAISE EXCEPTION 'AUT77: asignacion historica divergente' USING ERRCODE='40001';END IF;
  IF p_primera AND (NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa q
    WHERE q.perfil_activo_ref=asignacion.perfil_activo_ref AND q.asignacion_ref=asignacion.asignacion_ref)
   OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
   OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
   OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa rev
     WHERE rev.asignacion_id=asignacion.asignacion_id AND rev.documento->>'estado'='revocada'))
  THEN RAISE EXCEPTION 'AUT77: asignacion no actual o vigente' USING ERRCODE='40001';END IF;
 END LOOP;
 RETURN jsonb_build_object('vigente_hasta',CASE WHEN isfinite(limite_entradas)
  THEN to_char(limite_entradas AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  ELSE '0001-01-01T00:00:00Z' END);
END $function$;

RESET ROLE;
DO $verificacion$
DECLARE f record; esperado record; actual text;
BEGIN
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','bfb72a89426c710574f9549989b8cfb590d302fc09c8e3778b05e3d4707bcd82','{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_publicador_candidato_externo=X/vec_autorizacion_propietario}'),
  ('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)','ce2c6fb18cbdc22c7540377540ea2171df663eb3cf23f075f982c8cd7bf31260','{vec_autorizacion_propietario=X/vec_autorizacion_propietario}'),
  ('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)','a7baee7b6e0f935e3fccc79b91ba865a6db8d04ef3f9b6928f5d3457d6d0c564','{vec_autorizacion_propietario=X/vec_autorizacion_propietario}')
 ) AS v(firma,huella,acl) LOOP
  SELECT p.*, pg_get_userbyid(p.proowner) AS propietario INTO f FROM pg_proc p
  WHERE p.oid=to_regprocedure(esperado.firma);
  actual:=encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(esperado.firma)),'UTF8')),'hex');
  IF actual IS DISTINCT FROM esperado.huella
  THEN RAISE EXCEPTION 'AUT94: definición % actual=% esperada=%', esperado.firma,actual,esperado.huella USING ERRCODE='55000'; END IF;
  IF f.propietario IS DISTINCT FROM 'vec_autorizacion_propietario'
  THEN RAISE EXCEPTION 'AUT94: propietario % actual=% esperado=vec_autorizacion_propietario',esperado.firma,f.propietario USING ERRCODE='55000'; END IF;
  IF f.proacl::text IS DISTINCT FROM esperado.acl
  THEN RAISE EXCEPTION 'AUT94: ACL % actual=% esperada=%',esperado.firma,f.proacl::text,esperado.acl USING ERRCODE='55000'; END IF;
  IF NOT (f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
  THEN RAISE EXCEPTION 'AUT94: search_path % actual=% esperado=pg_catalog, pg_temp',esperado.firma,f.proconfig USING ERRCODE='55000'; END IF;
 END LOOP;
END $verificacion$;
COMMIT;
