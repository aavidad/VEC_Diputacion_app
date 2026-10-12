\set ON_ERROR_STOP on
-- AUT80: amplía únicamente el RolID M4 a tres entradas del catálogo ADMIN.
-- Conserva el plan histórico de una entrada y el cierre V3 de AUT60/AUT61.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ DECLARE c pg_proc%ROWTYPE;v pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO c FROM pg_proc WHERE oid=to_regprocedure(
  'vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)');
 SELECT * INTO v FROM pg_proc WHERE oid=to_regprocedure(
  'vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)');
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR c.oid IS NULL OR v.oid IS NULL
 OR c.proowner<>to_regrole('vec_autorizacion_propietario')
 OR v.proowner<>to_regrole('vec_autorizacion_propietario')
 OR c.prosecdef IS TRUE OR v.prosecdef IS NOT TRUE
 OR c.provolatile<>'i' OR v.provolatile<>'v'
 OR encode(sha256(convert_to(c.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  'c3305d5279d0c6db6da8433987369805a823fc8a0bce993fab9e359787107f48'
 OR encode(sha256(convert_to(v.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  '91a7d56ccfede8849f8c28ff0c5cf322e724135616d365175d2ca45c03c9075d'
 OR to_regprocedure('vec_autorizacion.exigir_rol_ordinario_gobierno_v1(jsonb,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AUT80: preimagen AUT60/61 divergente' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $canon$
DECLARE f oid:=to_regprocedure('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)');
 original text;nuevo text;meta jsonb;
 marca text:=$m$
   IF jsonb_typeof(v) IS DISTINCT FROM 'array' OR jsonb_array_length(v)<>1 THEN RAISE EXCEPTION 'AUT60: se requiere una concesion' USING ERRCODE='22023';END IF;
   IF tipos[i]='concesion[]' THEN s:='['||vec_autorizacion.objeto_canonico_go_admin_v1(v->0,'concesion')||']';
   ELSE s:='['||vec_autorizacion.canon_gobierno_rol_nuevo_v1(v->0,'seleccion')||']';END IF;
$m$;
 sustituto text:=$s$
   IF jsonb_typeof(v) IS DISTINCT FROM 'array' OR NOT (
    (jsonb_array_length(v) IN (2,3) AND
      ((t='plan' AND p#>>'{definicion_nueva,rol_id}'='persona_meritos_cartera')
       OR (t='definicion' AND p->>'rol_id'='persona_meritos_cartera')))
    OR (jsonb_array_length(v)=1 AND NOT
      ((t='plan' AND p#>>'{definicion_nueva,rol_id}'='persona_meritos_cartera')
       OR (t='definicion' AND p->>'rol_id'='persona_meritos_cartera'))))
   THEN RAISE EXCEPTION 'AUT80: cardinalidad de concesiones incompatible' USING ERRCODE='22023';END IF;
   IF tipos[i]='concesion[]' THEN
    SELECT '['||string_agg(vec_autorizacion.objeto_canonico_go_admin_v1(x.value,'concesion'),
      ',' ORDER BY x.ord)||']' INTO s
    FROM jsonb_array_elements(v) WITH ORDINALITY x(value,ord);
   ELSE
    SELECT '['||string_agg(vec_autorizacion.canon_gobierno_rol_nuevo_v1(x.value,'seleccion'),
      ',' ORDER BY x.ord)||']' INTO s
    FROM jsonb_array_elements(v) WITH ORDINALITY x(value,ord);
   END IF;
$s$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL OR
  (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT80: canon sin marca única' USING ERRCODE='55000';END IF;
 nuevo:=replace(original,marca,sustituto);
END $canon$;

CREATE OR REPLACE FUNCTION vec_autorizacion.canon_gobierno_rol_nuevo_v1(p jsonb, t text)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE campos text[];tipos text[];opc text[]:=ARRAY[]::text[];i integer;k text;v jsonb;s text;salida text:='';
BEGIN
 CASE t
 WHEN 'material' THEN campos:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];tipos:=ARRAY['string','string','string','string','plan'];
 WHEN 'plan' THEN campos:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256','version_rol_objetivo_ref','definicion_nueva','selecciones','motivo','referencia_acto'];tipos:=ARRAY['string','string','int','string','string','definicion','seleccion[]','motivo','string'];opc:=ARRAY['referencia_acto'];
 WHEN 'definicion' THEN campos:=ARRAY['rol_id','version','nombre','concesiones'];tipos:=ARRAY['string','int','nombre','concesion[]'];
 WHEN 'seleccion' THEN campos:=ARRAY['entrada_ref','entrada_version','entrada_huella_sha256'];tipos:=ARRAY['string','int64','string'];
 WHEN 'motivo' THEN campos:=ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'];tipos:=ARRAY['string','int','string','string'];
 WHEN 'entrada' THEN campos:=ARRAY['referencia','version','fuente_ref','fuente_version','fuente_huella_sha256','concesion','dimensiones_ambito','clase_control','vigente_desde','vigente_hasta'];tipos:=ARRAY['string','int64','string','int','string','concesion','strings','string','fecha','fecha'];
 ELSE RAISE EXCEPTION 'AUT60: canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>60000
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(campos))
 THEN RAISE EXCEPTION 'AUT60: objeto canon invalido' USING ERRCODE='22023'; END IF;
 FOR i IN 1..cardinality(campos) LOOP
  k:=campos[i];v:=p->k;s:=NULL;
  IF k=ANY(opc) AND (v IS NULL OR v='""'::jsonb) THEN CONTINUE; END IF;
  IF v IS NULL OR v='null'::jsonb THEN RAISE EXCEPTION 'AUT60: campo canon ausente' USING ERRCODE='22023'; END IF;
  IF tipos[i] IN('string','nombre','fecha') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR octet_length(v#>>'{}') NOT BETWEEN 1 AND 512
   OR (tipos[i]<>'nombre' AND (vec_autorizacion.texto_positivo_valido(v#>>'{}',512) IS NOT TRUE
    OR (v#>>'{}') COLLATE "C" ~ '[^!-~]'))
   OR (tipos[i]='nombre' AND ((v#>>'{}')<>btrim(v#>>'{}') OR (v#>>'{}') ~ '[[:cntrl:]]'))
   THEN RAISE EXCEPTION 'AUT60: cadena canon invalida' USING ERRCODE='22023'; END IF;
   IF tipos[i]='fecha' THEN s:=vec_autorizacion.fecha_canonica_go_admin_v1(v#>>'{}',k='vigente_hasta');ELSE s:=v#>>'{}';END IF;
   s:=vec_autorizacion.json_cadena_canonica_go_admin_v1(s);
  ELSIF tipos[i] IN('int','int64') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR (v#>>'{}') !~ '^[1-9][0-9]{0,18}$'
   OR (v#>>'{}')::numeric>(CASE WHEN tipos[i]='int' THEN 2147483647::numeric ELSE 9223372036854775807::numeric END)
   THEN RAISE EXCEPTION 'AUT60: entero canon invalido' USING ERRCODE='22023'; END IF;s:=v#>>'{}';
  ELSIF tipos[i] IN('seleccion[]','concesion[]') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'array' OR NOT (
    (jsonb_array_length(v) IN (2,3) AND
      ((t='plan' AND p#>>'{definicion_nueva,rol_id}'='persona_meritos_cartera')
       OR (t='definicion' AND p->>'rol_id'='persona_meritos_cartera')))
    OR (jsonb_array_length(v)=1 AND NOT
      ((t='plan' AND p#>>'{definicion_nueva,rol_id}'='persona_meritos_cartera')
       OR (t='definicion' AND p->>'rol_id'='persona_meritos_cartera'))))
   THEN RAISE EXCEPTION 'AUT80: cardinalidad de concesiones incompatible' USING ERRCODE='22023';END IF;
   IF tipos[i]='concesion[]' THEN
    SELECT '['||string_agg(vec_autorizacion.objeto_canonico_go_admin_v1(x.value,'concesion'),
      ',' ORDER BY x.ord)||']' INTO s
    FROM jsonb_array_elements(v) WITH ORDINALITY x(value,ord);
   ELSE
    SELECT '['||string_agg(vec_autorizacion.canon_gobierno_rol_nuevo_v1(x.value,'seleccion'),
      ',' ORDER BY x.ord)||']' INTO s
    FROM jsonb_array_elements(v) WITH ORDINALITY x(value,ord);
   END IF;
  ELSIF tipos[i]='concesion' THEN s:=vec_autorizacion.objeto_canonico_go_admin_v1(v,'concesion');
  ELSIF tipos[i]='strings' THEN s:=vec_autorizacion.array_cadenas_canonico_go_admin_v1(v);
  ELSE s:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(v,tipos[i]);END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(k)||':'||s;
 END LOOP;
 RETURN '{'||salida||'}';
END $function$;

DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '6e16d2fb60e60d12911067286b2cff728ed4fef7663f86040082b286037415c6'
 THEN RAISE EXCEPTION 'AUT80: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;


-- El validador conserva el caso histórico M4 de dos selecciones y acepta
-- la tercera concesión exacta de declaración. Mantiene vigente_hasta para AUT60.
DO $plan$
DECLARE f oid:=to_regprocedure('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)');
 original text;nuevo text;meta jsonb;
 marca text:=$m$
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
$m$;
 sustituto text:=$s$
 IF rolid='persona_meritos_cartera' THEN
  IF jsonb_array_length(p->'selecciones') NOT IN (2,3)
   OR jsonb_array_length(p#>'{definicion_nueva,concesiones}') IS DISTINCT FROM jsonb_array_length(p->'selecciones')
   OR p#>>'{definicion_nueva,concesiones,0,accion}' IS DISTINCT FROM 'meritos.cartera.consultar_propia'
   OR p#>>'{definicion_nueva,concesiones,1,accion}' IS DISTINCT FROM 'meritos.cartera.retirar_propio'
   OR (jsonb_array_length(p->'selecciones')=3
    AND p#>>'{definicion_nueva,concesiones,2,accion}' IS DISTINCT FROM 'meritos.hecho.declarar')
   OR p#>>'{selecciones,0,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,1,entrada_ref}'
   OR (jsonb_array_length(p->'selecciones')=3 AND (
    p#>>'{selecciones,0,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,2,entrada_ref}'
    OR p#>>'{selecciones,1,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,2,entrada_ref}'))
  THEN RAISE EXCEPTION 'AUT80: selecciones M4 incompatibles' USING ERRCODE='22023';END IF;
  ahora:=clock_timestamp();
  FOR idx IN 0..jsonb_array_length(p->'selecciones')-1 LOOP
   s:=p->'selecciones'->idx;
   SELECT value INTO STRICT e FROM jsonb_array_elements(c->'entradas')
    WHERE value->>'referencia'=s->>'entrada_ref';
   IF p_exigir_cabeza THEN
    PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(e,rolid,p#>>'{definicion_nueva,nombre}');
   END IF;
   esperado:=CASE idx WHEN 0 THEN jsonb_build_object(
    'accion','meritos.cartera.consultar_propia','modulo_id','meritos',
    'tipo_recurso','cartera_personal','finalidades',jsonb_build_array('gestion_cartera_propia'),
    'garantia_minima','alto','campos_permitidos',jsonb_build_array(
     'denominacion','disponibilidad','hecho_ref','motivos_retirada','retirada','tipo','total','version_hecho'),
    'obligaciones',jsonb_build_array('auditar'))
    WHEN 1 THEN jsonb_build_object(
     'accion','meritos.cartera.retirar_propio','modulo_id','meritos',
     'tipo_recurso','hecho','finalidades',jsonb_build_array('gestion_cartera_propia'),
     'garantia_minima','alto','campos_permitidos',jsonb_build_array('disponibilidad','recibo'),
     'obligaciones',jsonb_build_array('auditar'))
    ELSE jsonb_build_object(
     'accion','meritos.hecho.declarar','modulo_id','meritos',
     'tipo_recurso','hecho','finalidades',jsonb_build_array('declaracion_hecho_propio'),
     'garantia_minima','alto','campos_permitidos',jsonb_build_array(
      'declarante_ref','hecho','recibo','version'),
     'obligaciones',jsonb_build_array('auditar')) END;
   IF e->>'version' IS DISTINCT FROM s->>'entrada_version'
    OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex')
     IS DISTINCT FROM s->>'entrada_huella_sha256'
    OR e->'concesion' IS DISTINCT FROM esperado
    OR (p#>'{definicion_nueva,concesiones}')->idx IS DISTINCT FROM esperado
    OR e->>'clase_control' IS DISTINCT FROM 'ordinario'
    OR e->'dimensiones_ambito' IS DISTINCT FROM '["persona_ref"]'::jsonb
    OR (p_exigir_cabeza AND ahora<(c->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND c->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
     AND ahora>=(c->>'vigente_hasta')::timestamptz)
    OR (p_exigir_cabeza AND ahora<(e->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
     AND ahora>=(e->>'vigente_hasta')::timestamptz)
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q
     WHERE q#>>'{rol,rol_id}'=rolid)
   THEN RAISE EXCEPTION 'AUT80: descriptor M4 divergente' USING ERRCODE='42501';END IF;
   IF idx=0 THEN primero:=e;END IF;
   IF e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND (hasta IS NULL OR (e->>'vigente_hasta')::timestamptz<hasta) THEN
    hasta:=(e->>'vigente_hasta')::timestamptz;
    hasta_texto:=e->>'vigente_hasta';
   END IF;
  END LOOP;
  IF hasta IS NOT NULL THEN
   primero:=jsonb_set(primero,'{vigente_hasta}',to_jsonb(hasta_texto));
  END IF;
  RETURN primero;
 ELSE
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
 END IF;
$s$;
 declaracion text:=$d$DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;rolid text;catalogo_bytes bytea;$d$;
 declaracion_nueva text:=$d$DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;rolid text;catalogo_bytes bytea;
 idx integer;esperado jsonb;primero jsonb;hasta timestamptz;hasta_texto text;$d$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL
 OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 OR (length(original)-length(replace(original,declaracion,'')))/length(declaracion)<>1
 THEN RAISE EXCEPTION 'AUT80: validador sin marcas únicas' USING ERRCODE='55000';END IF;
 nuevo:=replace(replace(original,declaracion,declaracion_nueva),marca,sustituto);
END $plan$;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_rol_nuevo_v1(p jsonb, p_exigir_cabeza boolean)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;rolid text;catalogo_bytes bytea;
 idx integer;esperado jsonb;primero jsonb;hasta timestamptz;hasta_texto text;
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
 IF rolid='persona_meritos_cartera' THEN
  IF jsonb_array_length(p->'selecciones') NOT IN (2,3)
   OR jsonb_array_length(p#>'{definicion_nueva,concesiones}') IS DISTINCT FROM jsonb_array_length(p->'selecciones')
   OR p#>>'{definicion_nueva,concesiones,0,accion}' IS DISTINCT FROM 'meritos.cartera.consultar_propia'
   OR p#>>'{definicion_nueva,concesiones,1,accion}' IS DISTINCT FROM 'meritos.cartera.retirar_propio'
   OR (jsonb_array_length(p->'selecciones')=3
    AND p#>>'{definicion_nueva,concesiones,2,accion}' IS DISTINCT FROM 'meritos.hecho.declarar')
   OR p#>>'{selecciones,0,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,1,entrada_ref}'
   OR (jsonb_array_length(p->'selecciones')=3 AND (
    p#>>'{selecciones,0,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,2,entrada_ref}'
    OR p#>>'{selecciones,1,entrada_ref}' IS NOT DISTINCT FROM p#>>'{selecciones,2,entrada_ref}'))
  THEN RAISE EXCEPTION 'AUT80: selecciones M4 incompatibles' USING ERRCODE='22023';END IF;
  ahora:=clock_timestamp();
  FOR idx IN 0..jsonb_array_length(p->'selecciones')-1 LOOP
   s:=p->'selecciones'->idx;
   SELECT value INTO STRICT e FROM jsonb_array_elements(c->'entradas')
    WHERE value->>'referencia'=s->>'entrada_ref';
   IF p_exigir_cabeza THEN
    PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(e,rolid,p#>>'{definicion_nueva,nombre}');
   END IF;
   esperado:=CASE idx WHEN 0 THEN jsonb_build_object(
    'accion','meritos.cartera.consultar_propia','modulo_id','meritos',
    'tipo_recurso','cartera_personal','finalidades',jsonb_build_array('gestion_cartera_propia'),
    'garantia_minima','alto','campos_permitidos',jsonb_build_array(
     'denominacion','disponibilidad','hecho_ref','motivos_retirada','retirada','tipo','total','version_hecho'),
    'obligaciones',jsonb_build_array('auditar'))
    WHEN 1 THEN jsonb_build_object(
     'accion','meritos.cartera.retirar_propio','modulo_id','meritos',
     'tipo_recurso','hecho','finalidades',jsonb_build_array('gestion_cartera_propia'),
     'garantia_minima','alto','campos_permitidos',jsonb_build_array('disponibilidad','recibo'),
     'obligaciones',jsonb_build_array('auditar'))
    ELSE jsonb_build_object(
     'accion','meritos.hecho.declarar','modulo_id','meritos',
     'tipo_recurso','hecho','finalidades',jsonb_build_array('declaracion_hecho_propio'),
     'garantia_minima','alto','campos_permitidos',jsonb_build_array(
      'declarante_ref','hecho','recibo','version'),
     'obligaciones',jsonb_build_array('auditar')) END;
   IF e->>'version' IS DISTINCT FROM s->>'entrada_version'
    OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex')
     IS DISTINCT FROM s->>'entrada_huella_sha256'
    OR e->'concesion' IS DISTINCT FROM esperado
    OR (p#>'{definicion_nueva,concesiones}')->idx IS DISTINCT FROM esperado
    OR e->>'clase_control' IS DISTINCT FROM 'ordinario'
    OR e->'dimensiones_ambito' IS DISTINCT FROM '["persona_ref"]'::jsonb
    OR (p_exigir_cabeza AND ahora<(c->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND c->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
     AND ahora>=(c->>'vigente_hasta')::timestamptz)
    OR (p_exigir_cabeza AND ahora<(e->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
     AND ahora>=(e->>'vigente_hasta')::timestamptz)
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q
     WHERE q#>>'{rol,rol_id}'=rolid)
   THEN RAISE EXCEPTION 'AUT80: descriptor M4 divergente' USING ERRCODE='42501';END IF;
   IF idx=0 THEN primero:=e;END IF;
   IF e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND (hasta IS NULL OR (e->>'vigente_hasta')::timestamptz<hasta) THEN
    hasta:=(e->>'vigente_hasta')::timestamptz;
    hasta_texto:=e->>'vigente_hasta';
   END IF;
  END LOOP;
  IF hasta IS NOT NULL THEN
   primero:=jsonb_set(primero,'{vigente_hasta}',to_jsonb(hasta_texto));
  END IF;
  RETURN primero;
 ELSE
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
 END IF;
END $function$;

DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '912c94e1cf94bd4c39fcad7eecd6e706b15baf592a07126fe969ebec939c2d1d'
 THEN RAISE EXCEPTION 'AUT80: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;

RESET ROLE;
COMMIT;
