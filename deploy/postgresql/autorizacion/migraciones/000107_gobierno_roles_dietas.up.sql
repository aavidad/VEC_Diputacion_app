\set ON_ERROR_STOP on
-- AUT107 — F4-DIETAS-COPIA: gobierno nominal de comisiones propias e itinerarios.
-- Preimagen: cuerpos instalados en HZ37; main no los reconstruye después de AUT80.
-- Consumidor: propuesta y cierre ADMIN existentes; no publica ni asigna permisos.
-- No cambia el núcleo AD3, ACL, roles técnicos ni historia; solo UP.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
CREATE TEMP TABLE aut107_preimagen ON COMMIT DROP AS
 SELECT p.oid,to_jsonb(p)-'prosrc' AS metadatos
 FROM pg_proc p WHERE p.oid IN (
  to_regprocedure('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)'),
  to_regprocedure('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)'));
DO $pre$
DECLARE r record;f pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT107: requiere PostgreSQL18 y operador de migración' USING ERRCODE='55000';END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)','c0b057e136fb4084f33cc9aafc645a731759e703ab5c311b932e427b643113aa'),
  ('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)','3bac2ff899318cd0c9c76a221842345ecf9a22426e704a6e63faf18e3d21bfb0')
 ) x(firma,huella) LOOP
  SELECT * INTO f FROM pg_proc WHERE oid=to_regprocedure(r.firma);
  IF f.oid IS NULL OR f.proowner IS DISTINCT FROM to_regrole('vec_autorizacion_propietario')
   OR encode(sha256(convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM r.huella
   OR f.proconfig IS DISTINCT FROM (CASE WHEN r.firma LIKE '%canon_gobierno%'
       THEN ARRAY['search_path=pg_catalog, pg_temp']
       ELSE ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC'] END)
   OR f.prosecdef IS DISTINCT FROM (r.firma LIKE '%validar_plan%')
  THEN RAISE EXCEPTION 'AUT107: preimagen de gobierno divergente' USING ERRCODE='55000';END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_temp.aut107_preimagen)<>2
 THEN RAISE EXCEPTION 'AUT107: faltan funciones de gobierno' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion.canon_gobierno_rol_nuevo_v1(p jsonb, t text) RETURNS text
    LANGUAGE plpgsql IMMUTABLE
    SET search_path TO 'pg_catalog', 'pg_temp'
    AS $_$
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
    CASE CASE t WHEN 'plan' THEN p#>>'{definicion_nueva,rol_id}' ELSE p->>'rol_id' END
     WHEN 'persona_meritos_cartera' THEN jsonb_array_length(v) IN (2,3)
     WHEN 'persona_dietas_comisiones' THEN jsonb_array_length(v)=3
     WHEN 'persona_dietas_rutas' THEN jsonb_array_length(v)=2
     ELSE jsonb_array_length(v)=1 END)
   THEN RAISE EXCEPTION 'AUT107: cardinalidad de concesiones incompatible' USING ERRCODE='22023';END IF;
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
END $_$;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_rol_nuevo_v1(p jsonb, p_exigir_cabeza boolean) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET "TimeZone" TO 'UTC'
    AS $_$
DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;rolid text;catalogo_bytes bytea;
 idx integer;esperado jsonb;primero jsonb;hasta timestamptz;hasta_texto text;
 esperadas jsonb;dimensiones jsonb;
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
 IF rolid IN ('persona_dietas_comisiones','persona_dietas_rutas') THEN
  CASE rolid
  WHEN 'persona_dietas_comisiones' THEN esperadas:='[{"accion":"personal.relacion.propia.consultar_dietas","modulo_id":"personal","tipo_recurso":"relacion_empleado_dietas","finalidades":["preparar_borrador_dietas"],"garantia_minima":"alto","campos_permitidos":["desde","empleado_ref","estado","fuente_ref","fuente_version","hasta","persona_ref","procedencia_acto_ref","relacion_ref","unidad_ref","version"]},{"accion":"dietas.borrador.propio.crear","modulo_id":"dietas","tipo_recurso":"comision_borrador","finalidades":["crear_borrador_propio"],"garantia_minima":"alto","campos_permitidos":["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]},{"accion":"dietas.borrador.propio.consultar","modulo_id":"dietas","tipo_recurso":"comision_borrador","finalidades":["consultar_borrador_propio"],"garantia_minima":"alto","campos_permitidos":["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]}]'::jsonb;
   dimensiones:='["empleado_ref","persona_ref"]'::jsonb;
  WHEN 'persona_dietas_rutas' THEN esperadas:='[{"accion":"dietas.ruta.catalogo.consultar","modulo_id":"dietas","tipo_recurso":"catalogo_rutas_dietas","finalidades":["consultar_itinerario_dietas"],"garantia_minima":"alto"},{"accion":"dietas.ruta.calculo.solicitar","modulo_id":"dietas","tipo_recurso":"calculo_rutas_dietas","finalidades":["consultar_itinerario_dietas"],"garantia_minima":"alto"}]'::jsonb;
   dimensiones:='["ambito_ref"]'::jsonb;
  END CASE;
  IF jsonb_array_length(p->'selecciones') IS DISTINCT FROM jsonb_array_length(esperadas)
   OR jsonb_array_length(p#>'{definicion_nueva,concesiones}') IS DISTINCT FROM jsonb_array_length(esperadas)
   OR (SELECT count(DISTINCT x->>'entrada_ref') FROM jsonb_array_elements(p->'selecciones') x)
       IS DISTINCT FROM jsonb_array_length(esperadas)::bigint
  THEN RAISE EXCEPTION 'AUT107: selecciones Dietas incompatibles' USING ERRCODE='22023';END IF;
  ahora:=clock_timestamp();
  FOR idx IN 0..jsonb_array_length(esperadas)-1 LOOP
   s:=p->'selecciones'->idx;esperado:=esperadas->idx;
   SELECT value INTO STRICT e FROM jsonb_array_elements(c->'entradas')
    WHERE value->>'referencia'=s->>'entrada_ref';
   IF p_exigir_cabeza THEN
    PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(e,rolid,p#>>'{definicion_nueva,nombre}');
   END IF;
   IF e->>'version' IS DISTINCT FROM s->>'entrada_version'
    OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex')
       IS DISTINCT FROM s->>'entrada_huella_sha256'
    OR vec_autorizacion.objeto_canonico_go_admin_v1(e->'concesion','concesion')
       IS DISTINCT FROM vec_autorizacion.objeto_canonico_go_admin_v1(esperado,'concesion')
    OR vec_autorizacion.objeto_canonico_go_admin_v1((p#>'{definicion_nueva,concesiones}')->idx,'concesion')
       IS DISTINCT FROM vec_autorizacion.objeto_canonico_go_admin_v1(esperado,'concesion')
    OR e->>'clase_control' IS DISTINCT FROM 'ordinario'
    OR e->'dimensiones_ambito' IS DISTINCT FROM dimensiones
    OR (p_exigir_cabeza AND ahora<(c->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND c->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
        AND ahora>=(c->>'vigente_hasta')::timestamptz)
    OR (p_exigir_cabeza AND ahora<(e->>'vigente_desde')::timestamptz)
    OR (p_exigir_cabeza AND e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
        AND ahora>=(e->>'vigente_hasta')::timestamptz)
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q WHERE q#>>'{rol,rol_id}'=rolid)
   THEN RAISE EXCEPTION 'AUT107: descriptor Dietas divergente' USING ERRCODE='42501';END IF;
   IF idx=0 THEN primero:=e;END IF;
   IF e->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND (hasta IS NULL OR (e->>'vigente_hasta')::timestamptz<hasta) THEN
    hasta:=(e->>'vigente_hasta')::timestamptz;hasta_texto:=e->>'vigente_hasta';
   END IF;
  END LOOP;
  IF hasta IS NOT NULL THEN primero:=jsonb_set(primero,'{vigente_hasta}',to_jsonb(hasta_texto));END IF;
  RETURN primero;
 ELSIF rolid='persona_meritos_cartera' THEN
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
END $_$;

RESET ROLE;
DO $post$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_temp.aut107_preimagen x LEFT JOIN pg_proc p USING(oid)
   WHERE p.oid IS NULL OR to_jsonb(p)-'prosrc' IS DISTINCT FROM x.metadatos)
 THEN RAISE EXCEPTION 'AUT107: metadatos o ACL cambiados' USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
