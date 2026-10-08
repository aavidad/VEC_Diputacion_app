\set ON_ERROR_STOP on
-- AUT61: corrección prospectiva de la admisión del catálogo y del gobierno
-- de un RolID ordinario. Esta versión no reescribe asientos ni funciones
-- históricas; la sustitución de cuerpo preserva las firmas/OID/ACL.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE p record;actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT61: PARO clave=postgres18_superuser actual=no esperado=si' USING ERRCODE='55000';END IF;
 IF pg_catalog.to_regclass('vec_autorizacion.catalogo_accion_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.catalogo_fijo_sistemas_admin_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.registro_catalogo_acciones_admin_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_version_rol_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1()') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.exigir_rol_ordinario_gobierno_v1(jsonb,text,text)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)') IS NOT NULL
 OR pg_catalog.to_regrole('vec_admin_catalogo_acciones_ejecutor') IS NULL
 OR pg_catalog.to_regrole('vec_admin_catalogo_acciones_lector') IS NULL THEN
  RAISE EXCEPTION 'AUT61: PARO clave=dependencias actual=divergente esperado=POST_AUT60' USING ERRCODE='55000';END IF;
 FOR p IN SELECT * FROM (VALUES
  ('vec_autorizacion.aplicar_catalogo_acciones_admin_v1(text,text)',
   '81fa34a48edd0f0f8024489512b5d66f7c0cf205624ab7aeba80efda466edf67'),
  ('vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text)',
   'ba4f9399d4778ba15ad7c82f0369b409c652951d5230af7cbd12bc651f494f8d'),
  ('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)',
   '23eae9e453d9fcf089d0990c118d2a09748aefcda86c5af7f4a14307705fda8d'),
  ('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)',
   '2d0bcb0d38a933643b4305e525d15b19aa8e2b7ce55252925af321c935f96dd7'),
  ('vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '064a2e60877f85955621bf87b621da593294a3562e24c26f3dbbf7d677841230'),
  ('vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(text,text,text)',
   '5c808a4e76623fe4d386f17e84fb4aaf2377e0b7a54c6807110eb5595b70aed7')
 ) AS esperado(firma,huella) LOOP
  SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex')
   INTO actual FROM pg_catalog.pg_proc f WHERE f.oid=pg_catalog.to_regprocedure(p.firma)
   AND f.proowner=pg_catalog.to_regrole('vec_autorizacion_propietario')
   AND f.prosecdef AND pg_catalog.array_position(f.proconfig,'search_path=pg_catalog, pg_temp') IS NOT NULL;
  IF actual IS DISTINCT FROM p.huella THEN
   RAISE EXCEPTION 'AUT61: PARO clave=% actual=% esperado=%',p.firma,pg_catalog.coalesce(actual,'ausente'),p.huella USING ERRCODE='55000';END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- La preimagen de la fuente omite exclusivamente fuente_huella_sha256 para
-- evitar un punto fijo. Este orden y escape corresponden al struct Go de la
-- entrada, sin usar jsonb::text como canon de bytes.
CREATE FUNCTION vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(p_entradas jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE e jsonb;fragmento text;salida text:='';
BEGIN
 IF pg_catalog.jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(p_entradas) NOT BETWEEN 1 AND 512 THEN
  RAISE EXCEPTION 'AUT61: entradas de fuente invalidas' USING ERRCODE='22023';END IF;
 FOR e IN SELECT value FROM pg_catalog.jsonb_array_elements(p_entradas) WITH ORDINALITY AS x(value,n) ORDER BY n LOOP
  IF pg_catalog.jsonb_typeof(e) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(e))<>10
  OR (e ?& ARRAY['referencia','version','fuente_ref','fuente_version','fuente_huella_sha256',
   'concesion','dimensiones_ambito','clase_control','vigente_desde','vigente_hasta']) IS NOT TRUE THEN
   RAISE EXCEPTION 'AUT61: entrada de fuente incompleta' USING ERRCODE='22023';END IF;
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada');
  fragmento:='{"referencia":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(e->>'referencia')
   ||',"version":'||(e->>'version')
   ||',"fuente_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(e->>'fuente_ref')
   ||',"fuente_version":'||(e->>'fuente_version')
   ||',"concesion":'||vec_autorizacion.objeto_canonico_go_admin_v1(e->'concesion','concesion')
   ||',"dimensiones_ambito":'||vec_autorizacion.array_cadenas_canonico_go_admin_v1(e->'dimensiones_ambito')
   ||',"clase_control":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(e->>'clase_control')
   ||',"vigente_desde":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(
      vec_autorizacion.fecha_canonica_go_admin_v1(e->>'vigente_desde',false))
   ||',"vigente_hasta":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(
      vec_autorizacion.fecha_canonica_go_admin_v1(e->>'vigente_hasta',true))||'}';
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||fragmento;
  IF pg_catalog.octet_length(salida)>16777216 THEN
   RAISE EXCEPTION 'AUT61: fuente excede limite' USING ERRCODE='22023';END IF;
 END LOOP;
 RETURN '['||salida||']';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(jsonb) FROM PUBLIC;

-- Coteja bytes Go, SHA y proyeccion estructural; el metadato de fuente queda
-- ligado a cada descriptor y al paquete exacto aprobado por el DBA.
CREATE FUNCTION vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(p_paquete jsonb)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE fuente jsonb;e jsonb;proyectadas jsonb;canon text;identidad text;vistas text[]:=ARRAY[]::text[];
BEGIN
 IF pg_catalog.jsonb_typeof(p_paquete) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_paquete))<>5
 OR (p_paquete ?& ARRAY['esquema','referencia','version','fuentes','perfiles']) IS NOT TRUE
 OR p_paquete->>'esquema' IS DISTINCT FROM 'vec.admin.catalogo-acciones.paquete.v2'
 OR pg_catalog.jsonb_typeof(p_paquete->'fuentes') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(p_paquete->'fuentes') NOT BETWEEN 1 AND 512 THEN
  RAISE EXCEPTION 'AUT61: paquete de fuentes invalido' USING ERRCODE='22023';END IF;
 FOR fuente IN SELECT value FROM pg_catalog.jsonb_array_elements(p_paquete->'fuentes') WITH ORDINALITY AS f(value,n) ORDER BY n LOOP
  IF pg_catalog.jsonb_typeof(fuente) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(fuente))<>6
  OR (fuente ?& ARRAY['modulo_id','referencia','version','huella_sha256','entradas_canon','entradas']) IS NOT TRUE
  OR pg_catalog.jsonb_typeof(fuente->'modulo_id') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'referencia') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'version') IS DISTINCT FROM 'number'
  OR pg_catalog.jsonb_typeof(fuente->'huella_sha256') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'entradas_canon') IS DISTINCT FROM 'string'
  OR fuente->>'modulo_id' !~ '^[a-z][a-z0-9._:-]{0,127}$'
  OR fuente->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
  OR fuente->>'version' !~ '^[1-9][0-9]{0,8}$'
  OR fuente->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR pg_catalog.octet_length(fuente->>'entradas_canon') NOT BETWEEN 2 AND 16777216
  OR pg_catalog.jsonb_typeof(fuente->'entradas') IS DISTINCT FROM 'array'
  OR pg_catalog.jsonb_array_length(fuente->'entradas') NOT BETWEEN 1 AND 512 THEN
   RAISE EXCEPTION 'AUT61: fuente incompleta' USING ERRCODE='22023';END IF;
  identidad:=(fuente->>'modulo_id')||pg_catalog.chr(31)||(fuente->>'referencia')||pg_catalog.chr(31)||(fuente->>'version');
  IF identidad=ANY(vistas) THEN
   RAISE EXCEPTION 'AUT61: fuente duplicada' USING ERRCODE='42501';END IF;
  vistas:=pg_catalog.array_append(vistas,identidad);
  canon:=vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(fuente->'entradas');
  SELECT pg_catalog.jsonb_agg(e.value-'fuente_huella_sha256' ORDER BY e.n)
   INTO proyectadas FROM pg_catalog.jsonb_array_elements(fuente->'entradas') WITH ORDINALITY AS e(value,n);
  IF fuente->>'entradas_canon' IS DISTINCT FROM canon
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(canon,'UTF8')),'hex') IS DISTINCT FROM fuente->>'huella_sha256'
  OR (fuente->>'entradas_canon')::jsonb IS DISTINCT FROM proyectadas THEN
   RAISE EXCEPTION 'AUT61: huella o canon de fuente divergente' USING ERRCODE='42501';END IF;
  FOR e IN SELECT value FROM pg_catalog.jsonb_array_elements(fuente->'entradas') WITH ORDINALITY AS x(value,n) ORDER BY n LOOP
   IF e->>'fuente_ref' IS DISTINCT FROM fuente->>'referencia'
   OR e->>'fuente_version' IS DISTINCT FROM fuente->>'version'
   OR e->>'fuente_huella_sha256' IS DISTINCT FROM fuente->>'huella_sha256'
   OR e#>>'{concesion,modulo_id}' IS DISTINCT FROM fuente->>'modulo_id' THEN
    RAISE EXCEPTION 'AUT61: descriptor fuera de su fuente' USING ERRCODE='42501';END IF;
  END LOOP;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(jsonb) FROM PUBLIC;

-- Preimagen cerrada del catálogo central: Aplicación tiene como fuente la
-- versión de rol; Sistemas usa su catálogo inmutable AUT36. Una clase nueva
-- no obtiene por omisión autorización de alta ordinaria.
CREATE FUNCTION vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1()
RETURNS void LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 IF EXISTS(
  SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  LEFT JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=n.version_rol_ref
  LEFT JOIN vec_autorizacion.catalogo_fijo_sistemas_admin_v1 sis
   ON sis.catalogo_ref=n.fuente_ref AND sis.version=n.fuente_version
   AND sis.huella_sha256=n.fuente_huella_sha256
  WHERE CASE n.clase_control
   WHEN 'administrador_aplicacion' THEN
    v.version_rol_ref IS NULL OR n.fuente_ref IS DISTINCT FROM v.version_rol_ref
    OR n.fuente_version IS DISTINCT FROM v.version
    OR n.fuente_huella_sha256 IS DISTINCT FROM v.huella_sha256
    OR n.accion_ref IS DISTINCT FROM 'accion:'||(n.concesion->>'accion')
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(v.documento),'UTF8')),'hex') IS DISTINCT FROM v.huella_sha256
    OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(v.documento->'concesiones') q WHERE q=n.concesion)
   WHEN 'administrador_sistemas' THEN
    v.version_rol_ref IS NULL OR sis.catalogo_ref IS NULL
    OR pg_catalog.encode(pg_catalog.sha256(sis.material),'hex') IS DISTINCT FROM sis.huella_sha256
    OR sis.documento IS DISTINCT FROM pg_catalog.convert_from(sis.material,'UTF8')::jsonb
    OR v.rol_id IS DISTINCT FROM sis.documento#>>'{perfil_fijo,rol_id}'
    OR n.accion_ref IS DISTINCT FROM 'accion:'||(n.concesion->>'accion')
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(v.documento),'UTF8')),'hex') IS DISTINCT FROM v.huella_sha256
    OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(v.documento->'concesiones') q WHERE q=n.concesion)
    OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(sis.documento->'acciones') x
       WHERE x->>'accion_ref'=n.accion_ref
       AND x->>'clase_control'=n.clase_control
       AND x->'dimensiones_ambito'=n.dimensiones_ambito
       AND (x-ARRAY['accion_ref','dimensiones_ambito','clase_control'])=n.concesion)
   ELSE true END) THEN
  RAISE EXCEPTION 'AUT61: catalogo central divergente' USING ERRCODE='55000';END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1() FROM PUBLIC;
SELECT vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1();

-- La clasificación positiva procede del descriptor publicado y aprobado.
-- Las exclusiones AUT49 y la tripleta central impiden reclasificar una acción
-- administrativa cambiando sólo finalidades o el nombre del rol.
CREATE FUNCTION vec_autorizacion.exigir_rol_ordinario_gobierno_v1(p_entrada jsonb,p_rolid text,p_nombre text)
RETURNS void LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 PERFORM vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1();
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(p_entrada,'entrada');
 IF p_rolid IS NULL OR p_rolid !~ '^[a-z][a-z0-9_]{2,63}$'
 OR p_nombre IS NULL OR p_nombre=''
 OR p_rolid IN('administracion_perfiles','operador_plataforma')
 OR p_rolid ~ '(^candidato_|extern)' OR p_rolid ~ '^intervencion'
 OR p_nombre ~* '(fiscaliz|intervenc)'
 OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol v WHERE v.rol_id=p_rolid)
 OR p_entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
 OR p_entrada#>>'{concesion,modulo_id}' IS NULL
 OR p_entrada#>>'{concesion,modulo_id}' IN('administracion','intervencion','aspirantes')
 OR p_entrada#>>'{concesion,accion}' LIKE 'administracion.%'
 OR p_entrada#>>'{concesion,accion}' LIKE '%fiscalizacion%'
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements_text(p_entrada#>'{concesion,finalidades}') fi
   WHERE fi LIKE '%fiscaliz%') THEN
  RAISE EXCEPTION 'AUT61: descriptor reservado' USING ERRCODE='42501';END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  WHERE n.concesion->>'accion'=p_entrada#>>'{concesion,accion}'
   AND n.concesion->>'modulo_id'=p_entrada#>>'{concesion,modulo_id}'
   AND n.concesion->>'tipo_recurso'=p_entrada#>>'{concesion,tipo_recurso}') THEN
  RAISE EXCEPTION 'AUT61: accion central reservada' USING ERRCODE='42501';END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_rol_ordinario_gobierno_v1(jsonb,text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_autorizacion.aplicar_catalogo_acciones_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE cfg vec_autorizacion.config_catalogo_acciones_admin_v1;
 p jsonb; paquete jsonb;catalogo jsonb;fuente jsonb;entrada jsonb;
 sha text;paquete_sha text;catalogo_sha text;censo_sha text;flatten jsonb;
 canon bytea;paquete_bytes bytea;ahora timestamptz(6);prev record;aud record;recibo jsonb;
 esperado_version integer;esperado_sha text;actual record;numero_entradas integer;numero_perfiles integer;campo text;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1();
 IF plan_canonico IS NULL OR pg_catalog.octet_length(plan_canonico) NOT BETWEEN 1 AND 33554432
 OR sha_aprobado !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'AUT58: plan o huella inválidos' USING ERRCODE='22023'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM sha_aprobado OR sha IS DISTINCT FROM cfg.plan_sha256 THEN
  RAISE EXCEPTION 'AUT58: plan no aprobado' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'AUT58: plan no es un objeto' USING ERRCODE='22023'; END IF;
 FOREACH campo IN ARRAY ARRAY['esquema','operacion_ref','aprobacion_ref','aprobacion_sha256',
  'paquete_canon','paquete_ref','paquete_version','paquete_sha256','catalogo_canon',
  'catalogo_ref','catalogo_version','catalogo_sha256','esperado_version','esperado_sha256',
  'preparado_en','caduca_en','entorno'] LOOP
  IF pg_catalog.jsonb_typeof(p->campo) IS DISTINCT FROM 'string' THEN
   RAISE EXCEPTION 'AUT58: campo del plan ausente o no textual' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p))<>17
 OR (p ?& ARRAY['esquema','operacion_ref','aprobacion_ref','aprobacion_sha256','paquete_canon',
  'paquete_ref','paquete_version','paquete_sha256','catalogo_canon','catalogo_ref',
  'catalogo_version','catalogo_sha256','esperado_version','esperado_sha256',
  'preparado_en','caduca_en','entorno']) IS NOT TRUE
 OR p->>'esquema' NOT IN ('vec.admin.catalogo-acciones.plan.v1','vec.admin.catalogo-acciones.plan.v2')
 OR p->>'operacion_ref' !~ '^caa_[A-Za-z0-9_-]{22,123}$'
 OR p->>'aprobacion_ref' IS DISTINCT FROM cfg.aprobacion_ref
 OR p->>'aprobacion_sha256' IS DISTINCT FROM cfg.aprobacion_sha256
 OR p->>'paquete_ref' IS DISTINCT FROM cfg.paquete_ref
 OR p->>'paquete_version' IS DISTINCT FROM cfg.paquete_version::text
 OR p->>'paquete_sha256' IS DISTINCT FROM cfg.paquete_sha256
 OR p->>'catalogo_ref' IS DISTINCT FROM cfg.destino_ref
 OR p->>'catalogo_version' !~ '^[1-9][0-9]{0,8}$'
 OR p->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'esperado_version' !~ '^(0|[1-9][0-9]{0,8})$'
 OR p->>'esperado_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'entorno' IS DISTINCT FROM cfg.entorno
 OR pg_catalog.jsonb_typeof(p->'paquete_canon') IS DISTINCT FROM 'string'
 OR pg_catalog.jsonb_typeof(p->'catalogo_canon') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'AUT58: plan divergente de la aprobación' USING ERRCODE='42501'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF p->>'preparado_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR p->>'caduca_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR (p->>'preparado_en')::timestamptz(6)>ahora
 OR ahora>=(p->>'caduca_en')::timestamptz(6)
 OR (p->>'caduca_en')::timestamptz(6)>(p->>'preparado_en')::timestamptz(6)+interval '1 day'
 OR (p->>'caduca_en')::timestamptz(6)>cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT58: plan caducado o fuera de ventana' USING ERRCODE='42501'; END IF;
 IF pg_catalog.octet_length(p->>'paquete_canon') NOT BETWEEN 1 AND 16777216
 OR pg_catalog.octet_length(p->>'catalogo_canon') NOT BETWEEN 1 AND 16777216 THEN
  RAISE EXCEPTION 'AUT58: paquete o catálogo excede límite' USING ERRCODE='22023'; END IF;
 paquete_bytes:=pg_catalog.convert_to(p->>'paquete_canon','UTF8');
 canon:=pg_catalog.convert_to(p->>'catalogo_canon','UTF8');
 paquete_sha:=pg_catalog.encode(pg_catalog.sha256(paquete_bytes),'hex');
 catalogo_sha:=pg_catalog.encode(pg_catalog.sha256(canon),'hex');
 IF paquete_sha IS DISTINCT FROM cfg.paquete_sha256 OR catalogo_sha IS DISTINCT FROM p->>'catalogo_sha256' THEN
  RAISE EXCEPTION 'AUT58: paquete o catálogo alterado' USING ERRCODE='42501'; END IF;
 -- El plan v1 sólo puede devolver el recibo histórico exacto. Ninguna
 -- nueva instantánea v1 puede entrar tras AUT61, aunque el DBA mantenga cfg.
 IF p->>'esquema'='vec.admin.catalogo-acciones.plan.v1' THEN
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:catalogo-acciones:'||cfg.destino_ref,0));
  SELECT * INTO prev FROM vec_autorizacion.operacion_catalogo_acciones_admin_v1
   WHERE operacion_ref=p->>'operacion_ref';
  IF NOT FOUND THEN
   RAISE EXCEPTION 'AUT61: plan v1 cerrado para nuevos efectos' USING ERRCODE='42501';END IF;
  IF prev.plan_sha256 IS DISTINCT FROM sha OR prev.catalogo_ref IS DISTINCT FROM cfg.destino_ref
  OR prev.version IS DISTINCT FROM (p->>'catalogo_version')::integer
  OR prev.huella_sha256 IS DISTINCT FROM catalogo_sha THEN
   RAISE EXCEPTION 'AUT61: replay v1 divergente' USING ERRCODE='23505';END IF;
  PERFORM vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
   prev.catalogo_ref,prev.version,prev.huella_sha256);
  RETURN pg_catalog.jsonb_build_object('recibo',prev.recibo,'replay',true);
 END IF;
 paquete:=(p->>'paquete_canon')::jsonb;
 catalogo:=(p->>'catalogo_canon')::jsonb;
 IF pg_catalog.jsonb_typeof(paquete) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(paquete))<>5
 OR (paquete ?& ARRAY['esquema','referencia','version','fuentes','perfiles']) IS NOT TRUE
 OR paquete->>'esquema' IS DISTINCT FROM 'vec.admin.catalogo-acciones.paquete.v2'
 OR paquete->>'referencia' IS DISTINCT FROM cfg.paquete_ref
 OR paquete->>'version' IS DISTINCT FROM cfg.paquete_version::text
 OR pg_catalog.jsonb_typeof(paquete->'fuentes') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(paquete->'fuentes') NOT BETWEEN 1 AND 512
 OR pg_catalog.jsonb_typeof(paquete->'perfiles') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(paquete->'perfiles') NOT BETWEEN 1 AND 512
 OR pg_catalog.jsonb_typeof(catalogo) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(catalogo))<>9
 OR (catalogo ?& ARRAY['referencia','version','fuente_ref','fuente_version','fuente_huella_sha256',
  'vigente_desde','vigente_hasta','entradas','perfiles']) IS NOT TRUE
 OR catalogo->>'referencia' IS DISTINCT FROM cfg.destino_ref
 OR catalogo->>'version' IS DISTINCT FROM p->>'catalogo_version'
 OR catalogo->>'fuente_ref' IS DISTINCT FROM cfg.paquete_ref
 OR catalogo->>'fuente_version' IS DISTINCT FROM cfg.paquete_version::text
 OR catalogo->>'fuente_huella_sha256' IS DISTINCT FROM cfg.paquete_sha256
 OR catalogo->'perfiles' IS DISTINCT FROM paquete->'perfiles'
 OR pg_catalog.jsonb_typeof(catalogo->'entradas') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(catalogo->'entradas') NOT BETWEEN 1 AND 512
 THEN RAISE EXCEPTION 'AUT58: paquete o instantánea incompletos' USING ERRCODE='22023'; END IF;
 numero_entradas:=pg_catalog.jsonb_array_length(catalogo->'entradas');
 numero_perfiles:=pg_catalog.jsonb_array_length(catalogo->'perfiles');
 IF pg_catalog.jsonb_typeof(catalogo->'vigente_desde') IS DISTINCT FROM 'string'
 OR catalogo->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR pg_catalog.jsonb_typeof(catalogo->'vigente_hasta') IS DISTINCT FROM 'string'
 OR catalogo->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
 OR (catalogo->>'vigente_desde')::timestamptz(6)>ahora
 OR (catalogo->>'vigente_hasta')::timestamptz(6)<=ahora
 THEN RAISE EXCEPTION 'AUT58: instantánea sin vigencia actual' USING ERRCODE='42501'; END IF;
 PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(paquete);
 -- Todas las entradas son aportadas por una fuente declarada en el paquete
 -- comprometido; ninguna fuente vacía ni entrada extra puede pasar.
 FOR fuente IN SELECT value FROM pg_catalog.jsonb_array_elements(paquete->'fuentes') LOOP
  IF pg_catalog.jsonb_typeof(fuente) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(fuente))<>6
  OR (fuente ?& ARRAY['modulo_id','referencia','version','huella_sha256','entradas_canon','entradas']) IS NOT TRUE
  OR pg_catalog.jsonb_typeof(fuente->'modulo_id') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'referencia') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(fuente->'version') IS DISTINCT FROM 'number'
  OR pg_catalog.jsonb_typeof(fuente->'huella_sha256') IS DISTINCT FROM 'string'
  OR fuente->>'modulo_id' !~ '^[a-z][a-z0-9._:-]{0,127}$'
  OR fuente->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
  OR fuente->>'version' !~ '^[1-9][0-9]{0,8}$'
  OR fuente->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR pg_catalog.jsonb_typeof(fuente->'entradas') IS DISTINCT FROM 'array'
  OR pg_catalog.jsonb_array_length(fuente->'entradas') NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AUT58: fuente propietaria inválida' USING ERRCODE='22023'; END IF;
  FOR entrada IN SELECT value FROM pg_catalog.jsonb_array_elements(fuente->'entradas') LOOP
   IF pg_catalog.jsonb_typeof(entrada) IS DISTINCT FROM 'object'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_ref') IS DISTINCT FROM 'string'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_version') IS DISTINCT FROM 'number'
   OR pg_catalog.jsonb_typeof(entrada->'fuente_huella_sha256') IS DISTINCT FROM 'string'
   OR pg_catalog.jsonb_typeof(entrada#>'{concesion,modulo_id}') IS DISTINCT FROM 'string'
   OR entrada->>'fuente_ref' IS DISTINCT FROM fuente->>'referencia'
   OR entrada->>'fuente_version' IS DISTINCT FROM fuente->>'version'
   OR entrada->>'fuente_huella_sha256' IS DISTINCT FROM fuente->>'huella_sha256'
   OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM fuente->>'modulo_id'
   THEN RAISE EXCEPTION 'AUT58: descriptor fuera de su fuente' USING ERRCODE='42501'; END IF;
  END LOOP;
 END LOOP;
 SELECT pg_catalog.jsonb_agg(e.value ORDER BY f.n,e.n) INTO flatten
 FROM pg_catalog.jsonb_array_elements(paquete->'fuentes') WITH ORDINALITY AS f(value,n)
 CROSS JOIN LATERAL pg_catalog.jsonb_array_elements(f.value->'entradas') WITH ORDINALITY AS e(value,n);
 IF flatten IS DISTINCT FROM catalogo->'entradas' THEN
  RAISE EXCEPTION 'AUT58: entradas omitidas, añadidas o reordenadas' USING ERRCODE='42501'; END IF;
 censo_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((catalogo->'perfiles')::text,'UTF8')),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:catalogo-acciones:'||cfg.destino_ref,0));
 SELECT * INTO prev FROM vec_autorizacion.operacion_catalogo_acciones_admin_v1
  WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF prev.plan_sha256 IS DISTINCT FROM sha OR prev.catalogo_ref IS DISTINCT FROM cfg.destino_ref
  OR prev.version IS DISTINCT FROM (p->>'catalogo_version')::integer
  OR prev.huella_sha256 IS DISTINCT FROM catalogo_sha THEN
   RAISE EXCEPTION 'AUT58: operación repetida con material diferente' USING ERRCODE='23505'; END IF;
  RETURN pg_catalog.jsonb_build_object('recibo',prev.recibo,'replay',true);
 END IF;
 -- Censo actual completo: una única versión máxima por RolID, con el control
 -- vigente exacto. El paquete aprobado aporta el tipo; sólo las fuentes
 -- centrales positivas pueden demostrar una condición fija ya existente.
 IF numero_perfiles IS DISTINCT FROM (
  SELECT pg_catalog.count(*) FROM (SELECT DISTINCT ON (v.rol_id) v.rol_id
   FROM vec_autorizacion.version_rol v ORDER BY v.rol_id,v.version DESC) x)
 OR EXISTS(
  SELECT 1 FROM (SELECT DISTINCT ON (v.rol_id) v.rol_id,v.version_rol_ref,v.documento
   FROM vec_autorizacion.version_rol v ORDER BY v.rol_id,v.version DESC) v
  LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
  LEFT JOIN vec_autorizacion.control_vigencia_version_rol c
   ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision
  WHERE c.documento IS NULL OR NOT EXISTS(
   SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'perfiles') perfil
   WHERE perfil#>>'{rol,rol_id}'=v.rol_id
   AND perfil->'rol'=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(v.documento)
   AND perfil->'control_vigencia'=c.documento
   AND perfil->>'tipo_perfil' IN ('fijo_sistema','administrable')
   AND (perfil->>'tipo_perfil'='fijo_sistema' OR NOT EXISTS(
    SELECT 1 FROM vec_autorizacion.version_rol anterior
    WHERE anterior.rol_id=v.rol_id AND (
     EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 f
      WHERE f.version_rol_ref=anterior.version_rol_ref)
     OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s
      WHERE s.version_rol_ref=anterior.version_rol_ref))))))
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'perfiles') perfil
  WHERE pg_catalog.jsonb_typeof(perfil) IS DISTINCT FROM 'object'
  OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(perfil))<>3
  OR NOT (perfil ?& ARRAY['rol','control_vigencia','tipo_perfil']))
 THEN RAISE EXCEPTION 'AUT58: censo de roles incompleto o fijo reclasificado' USING ERRCODE='42501'; END IF;
 esperado_version:=(p->>'esperado_version')::integer;
 esperado_sha:=p->>'esperado_sha256';
 IF (esperado_version=0 AND esperado_sha<>pg_catalog.repeat('0',64))
 OR (esperado_version>0 AND esperado_sha=pg_catalog.repeat('0',64))
 OR (p->>'catalogo_version')::integer<>esperado_version+1 THEN
  RAISE EXCEPTION 'AUT58: preimagen CAS incoherente' USING ERRCODE='22023'; END IF;
 SELECT * INTO actual FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
  WHERE catalogo_ref=cfg.destino_ref FOR UPDATE;
 IF esperado_version=0 THEN
  IF FOUND THEN RAISE EXCEPTION 'AUT58: cabeza ya publicada' USING ERRCODE='40001'; END IF;
 ELSE
  IF NOT FOUND OR actual.version<>esperado_version OR actual.huella_sha256<>esperado_sha THEN
   RAISE EXCEPTION 'AUT58: cabeza CAS divergente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(
  pg_catalog.jsonb_build_object('tipo_registro','catalogo_acciones_admin','evento_ref','evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'operacion_ref',p->>'operacion_ref','plan_sha256',sha,
   'catalogo_ref',cfg.destino_ref,'catalogo_version',p->>'catalogo_version','catalogo_sha256',catalogo_sha,
   'paquete_ref',cfg.paquete_ref,'paquete_version',cfg.paquete_version::text,'paquete_sha256',paquete_sha,
   'censo_sha256',censo_sha,'entradas_numero',numero_entradas::text,'perfiles_numero',numero_perfiles::text,
   'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,
   'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','catalogo_acciones_admin',
   'correlacion_ref','correlacion_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','')));
 recibo:=pg_catalog.jsonb_build_object('esquema','vec.admin.catalogo-acciones.recibo.v1',
  'operacion_ref',p->>'operacion_ref','plan_sha256',sha,'catalogo_ref',cfg.destino_ref,
  'catalogo_version',p->>'catalogo_version','catalogo_sha256',catalogo_sha,
  'paquete_ref',cfg.paquete_ref,'paquete_version',cfg.paquete_version,
  'paquete_sha256',paquete_sha,'censo_sha256',censo_sha,'aprobacion_ref',cfg.aprobacion_ref,
  'aprobacion_sha256',cfg.aprobacion_sha256,'aprobador_ref',cfg.aprobador_ref,
  'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,
  'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_catalogo_acciones_admin_v1(
  catalogo_ref,version,huella_sha256,canon,paquete_ref,paquete_version,paquete_sha256,paquete_canon,
  censo_sha256,operacion_ref,plan_sha256,aprobacion_ref,aprobacion_sha256,aprobador_ref,login_nombre,
  auditoria_ref,recibo,registrada_en)
 VALUES(cfg.destino_ref,(p->>'catalogo_version')::integer,catalogo_sha,canon,cfg.paquete_ref,cfg.paquete_version,
  paquete_sha,paquete_bytes,censo_sha,p->>'operacion_ref',sha,cfg.aprobacion_ref,cfg.aprobacion_sha256,
  cfg.aprobador_ref,session_user,aud.auditoria_ref,recibo,aud.registrada_en);
 IF esperado_version=0 THEN
  INSERT INTO vec_autorizacion.cabeza_catalogo_acciones_admin_v1 VALUES(
   cfg.destino_ref,(p->>'catalogo_version')::integer,catalogo_sha,p->>'operacion_ref',aud.registrada_en);
 ELSE
  UPDATE vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   SET version=(p->>'catalogo_version')::integer,huella_sha256=catalogo_sha,
   operacion_ref=p->>'operacion_ref',actualizada_en=aud.registrada_en
   WHERE catalogo_ref=cfg.destino_ref AND version=esperado_version AND huella_sha256=esperado_sha;
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT58: CAS concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.operacion_catalogo_acciones_admin_v1
 VALUES(p->>'operacion_ref',sha,cfg.destino_ref,(p->>'catalogo_version')::integer,
  catalogo_sha,aud.auditoria_ref,recibo,aud.registrada_en);
 PERFORM vec_autorizacion.exigir_operador_catalogo_acciones_admin_v1();
 IF pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz(6) THEN
  RAISE EXCEPTION 'AUT58: plan vencido antes del efecto' USING ERRCODE='42501'; END IF;
 RETURN pg_catalog.jsonb_build_object('recibo',recibo,'replay',false);
END $f$;

-- La lectura para efectos nuevos entrega bytes crudos de las dos piezas
-- persistidas. La v1 sigue disponible únicamente para historia; ninguna
-- fuente v1 se interpreta como aprobada por esta interfaz prospectiva.
CREATE FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
 p_ref text,p_version integer,p_sha text)
RETURNS TABLE(catalogo_canon bytea,paquete_canon bytea)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE r record;p jsonb;c jsonb;entradas jsonb;
BEGIN
 catalogo_canon:=vec_autorizacion.resolver_catalogo_acciones_administracion_v1(p_ref,p_version,p_sha);
 SELECT * INTO STRICT r FROM vec_autorizacion.registro_catalogo_acciones_admin_v1 reg
  WHERE reg.catalogo_ref=p_ref AND reg.version=p_version AND reg.huella_sha256=p_sha;
 paquete_canon:=r.paquete_canon;
 p:=pg_catalog.convert_from(paquete_canon,'UTF8')::jsonb;
 c:=pg_catalog.convert_from(catalogo_canon,'UTF8')::jsonb;
 PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(p);
 SELECT pg_catalog.jsonb_agg(e.value ORDER BY f.n,e.n) INTO entradas
 FROM pg_catalog.jsonb_array_elements(p->'fuentes') WITH ORDINALITY AS f(value,n)
 CROSS JOIN LATERAL pg_catalog.jsonb_array_elements(f.value->'entradas') WITH ORDINALITY AS e(value,n);
 IF pg_catalog.encode(pg_catalog.sha256(paquete_canon),'hex') IS DISTINCT FROM r.paquete_sha256
 OR p->>'referencia' IS DISTINCT FROM r.paquete_ref
 OR p->>'version' IS DISTINCT FROM r.paquete_version::text
 OR p->'perfiles' IS DISTINCT FROM c->'perfiles'
 OR entradas IS DISTINCT FROM c->'entradas'
 OR r.recibo->>'operacion_ref' IS DISTINCT FROM r.operacion_ref
 OR r.recibo->>'plan_sha256' IS DISTINCT FROM r.plan_sha256
 OR r.recibo->>'catalogo_ref' IS DISTINCT FROM r.catalogo_ref
 OR r.recibo->>'catalogo_version' IS DISTINCT FROM r.version::text
 OR r.recibo->>'catalogo_sha256' IS DISTINCT FROM r.huella_sha256
 OR r.recibo->>'paquete_sha256' IS DISTINCT FROM r.paquete_sha256
 OR r.recibo->>'aprobacion_ref' IS DISTINCT FROM r.aprobacion_ref
 OR r.recibo->>'aprobacion_sha256' IS DISTINCT FROM r.aprobacion_sha256
 OR r.recibo->>'aprobador_ref' IS DISTINCT FROM r.aprobador_ref
 OR r.recibo->>'auditoria_ref' IS DISTINCT FROM r.auditoria_ref THEN
  RAISE EXCEPTION 'AUT61: registro aprobado divergente' USING ERRCODE='42501';END IF;
 RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)
 TO vec_admin_catalogo_acciones_ejecutor,vec_admin_catalogo_acciones_lector;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_rol_nuevo_v1(p jsonb,p_exigir_cabeza boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;rolid text;catalogo_bytes bytea;
BEGIN
 canon:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(p,'plan');
 rolid:=p#>>'{definicion_nueva,rol_id}';
 IF p_exigir_cabeza IS NULL OR p->>'operacion' IS DISTINCT FROM 'crear' OR p#>>'{definicion_nueva,version}' IS DISTINCT FROM '1'
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM 'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v1'
 OR p#>>'{definicion_nueva,rol_id}' !~ '^[a-z][a-z0-9_]{2,63}$'
 THEN RAISE EXCEPTION 'AUT60: solo RolID nuevo v1' USING ERRCODE='22023';END IF;
 IF p_exigir_cabeza THEN
  SELECT * INTO h FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1 WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR h.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR h.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT60: cabeza catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 IF p_exigir_cabeza THEN
  SELECT x.catalogo_canon INTO STRICT catalogo_bytes
   FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
    p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256') x;
 ELSE
  catalogo_bytes:=vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
   p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256');
 END IF;
 c:=convert_from(catalogo_bytes,'UTF8')::jsonb;
 s:=p#>'{selecciones,0}';
 SELECT value INTO STRICT e FROM jsonb_array_elements(c->'entradas') WHERE value->>'referencia'=s->>'entrada_ref';
 ahora:=clock_timestamp();
 IF p_exigir_cabeza THEN
  PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(e,rolid,p#>>'{definicion_nueva,nombre}');
 END IF;
 IF e->>'version' IS DISTINCT FROM s->>'entrada_version'
 OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex') IS DISTINCT FROM s->>'entrada_huella_sha256'
 OR vec_autorizacion.objeto_canonico_go_admin_v1(e->'concesion','concesion') IS DISTINCT FROM vec_autorizacion.objeto_canonico_go_admin_v1(p#>'{definicion_nueva,concesiones,0}','concesion')
 OR (p_exigir_cabeza AND ahora<(c->>'vigente_desde')::timestamptz)
 OR (p_exigir_cabeza AND c->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(c->>'vigente_hasta')::timestamptz)
 OR (p_exigir_cabeza AND ahora<(e->>'vigente_desde')::timestamptz)
 OR (p_exigir_cabeza AND e->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(e->>'vigente_hasta')::timestamptz)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q WHERE q#>>'{rol,rol_id}'=p#>>'{definicion_nueva,rol_id}')
 THEN RAISE EXCEPTION 'AUT60: descriptor no vigente o divergente' USING ERRCODE='42501';END IF;
 RETURN e;
END $f$;

CREATE OR REPLACE FUNCTION vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(
 p_cierre boolean,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC'
SET lock_timeout='2s' AS $f$
DECLARE m jsonb;mat jsonb;plan jsonb;d jsonb;c jsonb;motivo jsonb;entrada jsonb;efectivos jsonb;
 prop vec_autorizacion.propuesta_gobierno_rol_nuevo_v1;prev vec_autorizacion.cierre_gobierno_rol_nuevo_v1;
 x record;a record;amb jsonb;ambcanon text;ctx text;huella text;material_sha text;
 persona text;perfil text;asignacion text;recurso text;op text;accion text;audiencia text;
 ahora timestamptz(6);caduca timestamptz(6);fecha text;rol jsonb;control jsonb;rol_sha text;control_sha text;
 resultado jsonb;recibo jsonb;acto text;recibo_ref text;claves text[];replay boolean:=false;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR p_cierre IS NULL OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AUT60: contexto invalido' USING ERRCODE='42501';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT60: envelope invalido' USING ERRCODE='22023';END IF;
 SELECT array_agg(k ORDER BY k COLLATE "C") INTO claves FROM jsonb_object_keys(m) k;
 IF NOT p_cierre THEN
  IF claves IS DISTINCT FROM ARRAY['correlacion_ref','esquema','material_canon','material_sha256','plan_sha256']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_gobierno_rol_nuevo_propuesta_v1'
  OR jsonb_typeof(m->'material_canon') IS DISTINCT FROM 'string' OR octet_length(m->>'material_canon') NOT BETWEEN 1 AND 60000
  THEN RAISE EXCEPTION 'AUT60: propuesta invalida' USING ERRCODE='22023';END IF;
  mat:=(m->>'material_canon')::jsonb;plan:=mat->'Plan';
  IF vec_autorizacion.canon_gobierno_rol_nuevo_v1(mat,'material') IS DISTINCT FROM m->>'material_canon'
  OR encode(sha256(convert_to(m->>'material_canon','UTF8')),'hex') IS DISTINCT FROM m->>'material_sha256'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(plan,'plan'),'UTF8')),'hex') IS DISTINCT FROM m->>'plan_sha256'
  OR mat->>'OperacionRef' !~ '^propuesta_admin:[0-9a-f]{32}$'
  THEN RAISE EXCEPTION 'AUT60: canon o huella divergente' USING ERRCODE='22023';END IF;
  persona:=mat->>'ProponentePersonaRef';perfil:=mat->>'PerfilActivoRef';asignacion:=mat->>'AsignacionPerfilRef';
  recurso:=plan->>'version_rol_objetivo_ref';op:=mat->>'OperacionRef';motivo:=plan->'motivo';
  accion:='administracion.perfiles.definicion.proponer';audiencia:='vec_autorizacion.gobierno_rol_nuevo.propuesta.v1';
 ELSE
  IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_gobierno_rol_nuevo_cierre_v1'
  OR m->>'decision' IS DISTINCT FROM 'aprobada' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AUT60: cierre invalido' USING ERRCODE='22023';END IF;
  persona:=m->>'actor_persona_ref';perfil:=m->>'actor_perfil_ref';asignacion:=m->>'asignacion_ref';
  recurso:=m->>'propuesta_ref';op:=m->>'operacion_ref';motivo:=m->'motivo';
  accion:='administracion.perfiles.definicion.aprobar';audiencia:='vec_autorizacion.gobierno_rol_nuevo.cierre.v1';
 END IF;
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(motivo,'motivo');
 IF (persona ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE OR (perfil ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR m->>'correlacion_ref' IS NULL OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR (convert_from(p_motivo,'UTF8')::jsonb)->'referencia' IS DISTINCT FROM motivo
 OR d->>'principal_id' IS DISTINCT FROM persona OR d->>'perfil_activo_ref' IS DISTINCT FROM perfil
 OR d->>'asignacion_ref' IS DISTINCT FROM asignacion OR d->>'accion' IS DISTINCT FROM accion
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60: actor o decision ajenos' USING ERRCODE='42501';END IF;
 SELECT q.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil q WHERE q.asignacion_ref=asignacion;
 IF jsonb_typeof(a.documento->'ambitos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(a.documento->'ambitos'))<>2
 OR (SELECT count(DISTINCT b->>'clave') FROM jsonb_array_elements(a.documento->'ambitos') b)<>2
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' NOT IN('organizacion_ref','unidad_ref')
  OR (CASE WHEN jsonb_typeof(b->'valores')='array' THEN jsonb_array_length(b->'valores')=1 ELSE false END) IS NOT TRUE)
 THEN RAISE EXCEPTION 'AUT60: ambitos no unitarios' USING ERRCODE='42501';END IF;
 SELECT jsonb_object_agg(b->>'clave',b#>>'{valores,0}') INTO amb FROM jsonb_array_elements(a.documento->'ambitos') b;
 IF (SELECT count(*) FROM jsonb_object_keys(amb))<>2 OR NOT amb ?& ARRAY['organizacion_ref','unidad_ref']
 THEN RAISE EXCEPTION 'AUT60: ambitos divergentes' USING ERRCODE='42501';END IF;
 ambcanon:='{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'organizacion_ref')
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'unidad_ref')||'}';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:='{"ambitos":'||ambcanon||',"atributos":{"material_sha256":"'||material_sha||'"}}';
 huella:=encode(sha256(convert_to(ctx,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM recurso OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR c->>'efecto_ref' IS DISTINCT FROM recurso OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT60: recurso no ligado al material' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso OR x.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT60: consumo divergente' USING ERRCODE='42501';END IF;
 efectivos:=vec_autorizacion.administradores_gobierno_rol_nuevo_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 THEN RAISE EXCEPTION 'AUT60: dos ADMIN actuales necesarios' USING ERRCODE='42501';END IF;
 IF p_cierre THEN
  SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=recurso FOR SHARE;
  mat:=convert_from(prop.material,'UTF8')::jsonb;plan:=mat->'Plan';
  IF prop.material_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256'
  OR motivo IS DISTINCT FROM plan->'motivo'
  OR prop.proponente_persona_ref=persona OR prop.proponente_perfil_ref=perfil
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref
   AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref)
  THEN RAISE EXCEPTION 'AUT60: cierre no independiente o proponente revocado' USING ERRCODE='42501';END IF;
 ELSE
  SELECT * INTO prop FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=op FOR SHARE;
  IF FOUND THEN
   IF vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(prop.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT60: replay de propuesta divergente' USING ERRCODE='23505';END IF;
   replay:=true;
  END IF;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:rol-nuevo:'||(plan#>>'{definicion_nueva,rol_id}'),0));
 IF p_cierre THEN
  SELECT * INTO prev FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=prop.propuesta_ref;
  IF FOUND THEN
   IF prev.operacion_ref IS DISTINCT FROM op
   OR vec_autorizacion.solicitud_replay_gobierno_rol_nuevo_v1(prev.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT60: propuesta cerrada con otro material' USING ERRCODE='23505';END IF;
   PERFORM vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(prop.propuesta_ref);
   replay:=true;
  END IF;
 END IF;
 entrada:=vec_autorizacion.validar_plan_rol_nuevo_v1(plan,NOT replay);
 IF p_cierre THEN
  IF replay THEN
   resultado:=prev.resultado||jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=clock_timestamp();
   IF ahora>=prop.caduca_en OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}')
   THEN RAISE EXCEPTION 'AUT60: propuesta caducada o RolID ocupado' USING ERRCODE='40001';END IF;
   fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
   acto:='acto_admin:'||substr(op,14,32);recibo_ref:='recibo_admin:'||substr(op,14,32);
   rol:=plan->'definicion_nueva'||jsonb_build_object('estado','publicada','publicada_por',persona,'publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
   rol_sha:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex');
   control:=jsonb_build_object('version_rol_ref',prop.version_rol_ref,'revision',1,'estado','habilitada','actualizado_por',persona,'actualizado_en',fecha);
   control_sha:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
   INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
    VALUES(prop.version_rol_ref,rol->>'rol_id',1,rol_sha,ahora,rol);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
    VALUES(prop.version_rol_ref,1,'habilitada',control_sha,ahora,control);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
    VALUES(prop.version_rol_ref,1,ahora,persona,acto);
   recibo:=jsonb_build_object('acto_ref',acto,'recibo_ref',recibo_ref,'actor_persona_ref',persona,'perfil_activo_ref',perfil,
    'asignacion_perfil_ref',asignacion,'correlacion_ref',m->>'correlacion_ref','motivo',motivo,'auditoria_ref',x.auditoria_ref,'version_rol',rol,'control_posterior',control);
   resultado:=jsonb_build_object('operacion_ref',op,'material_canon',convert_from(prop.material,'UTF8'),'propuesta_huella_sha256',prop.material_sha256,
    'decision','aprobada','confirmado_en',ahora,'auditoria_acceso_ref',x.auditoria_ref,'recibo',recibo);
   INSERT INTO vec_autorizacion.cierre_gobierno_rol_nuevo_v1 VALUES(prop.propuesta_ref,op,convert_to(p_material,'UTF8'),prop.version_rol_ref,rol_sha,control_sha,persona,perfil,asignacion,x.auditoria_ref,resultado,ahora);
   INSERT INTO vec_autorizacion.outbox_gobierno_rol_nuevo_v1 VALUES(op,'definicion_publicada',x.auditoria_ref,ahora);
  END IF;
 ELSE
  IF replay THEN
   IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=op) THEN
    PERFORM vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(op);
   ELSIF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}') THEN
    RAISE EXCEPTION 'AUT60: RolID ocupado' USING ERRCODE='40001';END IF;
   resultado:=vec_autorizacion.resultado_propuesta_acceso_gobierno_rol_nuevo_v1(prop.resultado,x.auditoria_ref);
  ELSE
   IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}')
   THEN RAISE EXCEPTION 'AUT60: RolID debe ser nuevo' USING ERRCODE='40001';END IF;
   ahora:=clock_timestamp();
   caduca:=LEAST(ahora+interval '1 day',(a.documento->>'vigente_hasta')::timestamptz,
    NULLIF(entrada->>'vigente_hasta','0001-01-01T00:00:00Z')::timestamptz);
   resultado:=jsonb_build_object('material_canon',m->>'material_canon','huella_sha256',m->>'material_sha256',
    'caduca_en',caduca,'auditoria_acceso_ref',x.auditoria_ref);
   INSERT INTO vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 VALUES(op,convert_to(m->>'material_canon','UTF8'),m->>'material_sha256',convert_to(p_material,'UTF8'),
    persona,perfil,asignacion,plan->>'catalogo_ref',(plan->>'catalogo_version')::integer,plan->>'catalogo_huella_sha256',recurso,ahora,caduca,x.auditoria_ref,resultado);
   INSERT INTO vec_autorizacion.outbox_gobierno_rol_nuevo_v1 VALUES(op,'definicion_propuesta',x.auditoria_ref,ahora);
  END IF;
 END IF;
 -- La autorización queda revalidada bajo los bloqueos y la barrera de auditoría.
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60: autorizacion final no vigente' USING ERRCODE='42501';END IF;
 PERFORM vec_autorizacion.validar_plan_rol_nuevo_v1(plan,NOT replay);
 efectivos:=vec_autorizacion.administradores_gobierno_rol_nuevo_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 OR (p_cierre AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref))
 OR (p_cierre AND NOT replay AND clock_timestamp()>=prop.caduca_en)
 THEN RAISE EXCEPTION 'AUT60: doble control final no vigente' USING ERRCODE='42501';END IF;
 RETURN resultado||jsonb_build_object('estado','permitido','replay',replay);
END $f$;

CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(
 p_material text,p_accion text,p_sqlstate text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;h text;corr text;estado text;codigo text;a record;material_no_json boolean:=false;
BEGIN
 h:=encode(sha256(convert_to(coalesce(p_material,''),'UTF8')),'hex');
 corr:='correlacion_'||substr(h,1,32);
 BEGIN
  m:=p_material::jsonb;
  IF jsonb_typeof(m)='object' AND m->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  THEN corr:=m->>'correlacion_ref';END IF;
 EXCEPTION WHEN invalid_text_representation THEN material_no_json:=true;
 WHEN OTHERS THEN NULL;
 END;
 estado:=CASE WHEN material_no_json OR p_sqlstate IN('42501','22023','23505','P0002') THEN 'denegado' ELSE 'error' END;
 codigo:='gobierno_rol_nuevo_'||estado;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(
  jsonb_build_object('tipo_registro','intento_gobierno_rol_nuevo',
   'evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_gobierno_rol_nuevo:'||substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'auditoria_intento',
  jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;

RESET ROLE;
DO $post$
DECLARE item record;f record;numero integer;propietario oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(jsonb)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(jsonb)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1()',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.exigir_rol_ordinario_gobierno_v1(jsonb,text,text)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.aplicar_catalogo_acciones_admin_v1(text,text)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(text,text,text)',ARRAY['vec_autorizacion_propietario']::text[]),
  ('vec_autorizacion.registrar_catalogo_acciones_admin_v1(text,text)',ARRAY['vec_autorizacion_propietario','vec_admin_catalogo_acciones_ejecutor']::text[]),
  ('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)',ARRAY['vec_autorizacion_propietario','vec_admin_catalogo_acciones_ejecutor','vec_admin_catalogo_acciones_lector']::text[]),
  ('vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)',ARRAY['vec_autorizacion_propietario','vec_admin_catalogo_acciones_ejecutor','vec_admin_catalogo_acciones_lector']::text[])
 ) AS permisos(firma,roles) LOOP
  SELECT * INTO f FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure(item.firma);
  IF NOT FOUND OR f.proowner IS DISTINCT FROM propietario
  OR pg_catalog.array_position(f.proconfig,'search_path=pg_catalog, pg_temp') IS NULL THEN
   RAISE EXCEPTION 'AUT61: PARO clave=funcion actual=% esperado=owner_search_path',item.firma USING ERRCODE='55000';END IF;
  SELECT pg_catalog.count(*) INTO numero FROM pg_catalog.aclexplode(
   pg_catalog.coalesce(f.proacl,pg_catalog.acldefault('f',f.proowner)));
  IF numero IS DISTINCT FROM pg_catalog.cardinality(item.roles)
  OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(
    pg_catalog.coalesce(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
    LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee
    WHERE pg_catalog.coalesce(r.rolname,'PUBLIC')<>ALL(item.roles)
    OR a.grantor IS DISTINCT FROM propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  OR EXISTS(SELECT 1 FROM pg_catalog.unnest(item.roles) rol
    WHERE NOT pg_catalog.has_function_privilege(rol,f.oid,'EXECUTE')) THEN
   RAISE EXCEPTION 'AUT61: PARO clave=ACL_% actual=% esperado=%',item.firma,numero,pg_catalog.cardinality(item.roles) USING ERRCODE='55000';END IF;
 END LOOP;
END $post$;
COMMIT;
