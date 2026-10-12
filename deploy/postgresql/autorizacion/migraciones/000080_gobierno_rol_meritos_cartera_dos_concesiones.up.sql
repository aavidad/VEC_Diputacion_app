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
 EXECUTE nuevo;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR replace(pg_get_functiondef(f),sustituto,marca) IS DISTINCT FROM original
 THEN RAISE EXCEPTION 'AUT80: canon alterado fuera del contrato' USING ERRCODE='55000';END IF;
END $canon$;

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
 EXECUTE nuevo;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR replace(replace(pg_get_functiondef(f),sustituto,marca),declaracion_nueva,declaracion)
  IS DISTINCT FROM original
 THEN RAISE EXCEPTION 'AUT80: validador alterado fuera del contrato' USING ERRCODE='55000';END IF;
END $plan$;
RESET ROLE;
COMMIT;
