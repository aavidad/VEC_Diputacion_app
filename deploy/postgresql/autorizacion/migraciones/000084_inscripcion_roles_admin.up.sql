\set ON_ERROR_STOP on
-- AUT84: deltas exactos de inscripción RRHH y aspirante por sus circuitos ADMIN existentes.
-- Definiciones literales extraídas con pg_get_functiondef de la copia PG18 principal.sql.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT84: migrador PostgreSQL18 requerido' USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion.canon_version_rol_bolsa_v2(jsonb,text)')),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '9f305bf3ae95eb4807dfbf91453d8af940efa2f27793723ebe0780d66663dd58' THEN RAISE EXCEPTION 'AUT84: preimagen divergente canon_int actual=% esperado=9f305bf3ae95eb4807dfbf91453d8af940efa2f27793723ebe0780d66663dd58',h USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion.validar_plan_version_rol_bolsa_v2(jsonb,boolean)')),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '2ae9f50376afb231140b06336b94142ecd34f5e6ec8882df61249fbccfc70da2' THEN RAISE EXCEPTION 'AUT84: preimagen divergente valid_int actual=% esperado=2ae9f50376afb231140b06336b94142ecd34f5e6ec8882df61249fbccfc70da2',h USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion.canon_version_rol_bolsa_externa_v1(jsonb,text)')),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '80400be1443bcf13d8274017d1660876e277d8fead60c4d31499d3a2e8cc8f9c' THEN RAISE EXCEPTION 'AUT84: preimagen divergente canon_ext actual=% esperado=80400be1443bcf13d8274017d1660876e277d8fead60c4d31499d3a2e8cc8f9c',h USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)')),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'e0ad68f9a0912a950642ebb3b90e5d60a65c1d7d959ef11fee9353af70ab16cd' THEN RAISE EXCEPTION 'AUT84: preimagen divergente valid_ext actual=% esperado=e0ad68f9a0912a950642ebb3b90e5d60a65c1d7d959ef11fee9353af70ab16cd',h USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()')),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'ba78080f9b02054aa61603825a885988a9090af31571fbb55f5076d6c18e1374' THEN RAISE EXCEPTION 'AUT84: preimagen divergente base_ext actual=% esperado=ba78080f9b02054aa61603825a885988a9090af31571fbb55f5076d6c18e1374',h USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.canon_version_rol_bolsa_v2(p jsonb, t text)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE s text:='';selecciones text:='';salida text:='';e jsonb;partes text[]:=ARRAY[]::text[];claves text[];
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>131072
 THEN RAISE EXCEPTION 'AUT75: canon invalido' USING ERRCODE='22023';END IF;
 CASE t
 WHEN 'material' THEN
  claves:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'OperacionRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'ProponentePersonaRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'PerfilActivoRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'AsignacionPerfilRef'),
   vec_autorizacion.canon_version_rol_bolsa_v2(p->'Plan','plan')];
 WHEN 'plan' THEN
  claves:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256',
   'version_rol_objetivo_ref','base','definicion_nueva','selecciones','asignaciones','motivo','referencia_acto'];
  IF p->>'operacion' IS DISTINCT FROM 'versionar'
   OR jsonb_typeof(p->'selecciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'selecciones') NOT IN (1,3,4)
   OR jsonb_typeof(p->'asignaciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'asignaciones') NOT BETWEEN 1 AND 16
   OR (p->>'catalogo_version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT75: plan invalido' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'selecciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF selecciones<>'' THEN selecciones:=selecciones||',';END IF;
   selecciones:=selecciones||vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'seleccion');
  END LOOP;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.canon_version_rol_bolsa_v2(e,'asignacion');
  END LOOP;
  s:='['||s||']';
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'operacion'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_ref'),
   p->>'catalogo_version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_huella_sha256'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'version_rol_objetivo_ref'),
   vec_autorizacion.canon_version_rol_bolsa_v2(p->'base','base'),
   vec_autorizacion.canon_version_rol_bolsa_v2(p->'definicion_nueva','definicion'),
   '['||selecciones||']',
   s,
   vec_autorizacion.canon_gobierno_rol_nuevo_v1(p->'motivo','motivo'),
   CASE WHEN coalesce(p->>'referencia_acto','')='' THEN NULL
    ELSE vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'referencia_acto') END];
 WHEN 'base' THEN
  claves:=ARRAY['rol','control_vigencia','tipo_perfil'];
  partes:=ARRAY[
   vec_autorizacion.canon_version_rol_admin_v1(p->'rol'),
   vec_autorizacion.canon_control_rol_admin_v1(p->'control_vigencia'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'tipo_perfil')];
 WHEN 'definicion' THEN
  claves:=ARRAY['rol_id','version','nombre','concesiones'];
  IF jsonb_typeof(p->'concesiones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'concesiones') NOT BETWEEN 2 AND 512
   OR (p->>'version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT75: definicion invalida' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'concesiones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.objeto_canonico_go_admin_v1(e,'concesion');
  END LOOP;
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'rol_id'),
   p->>'version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'nombre'),
   '['||s||']'];
 WHEN 'asignacion' THEN
  claves:=ARRAY['asignacion_ref','huella_sha256','documento'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'asignacion_ref'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'huella_sha256'),
   vec_autorizacion.canon_asignacion_perfil_admin_v1(p->'documento')];
 ELSE RAISE EXCEPTION 'AUT75: tipo canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF t='plan' AND partes[11] IS NULL THEN
  IF p ? 'referencia_acto' THEN
   RAISE EXCEPTION 'AUT75: referencia de acto vacía' USING ERRCODE='22023';END IF;
  claves:=claves[1:10];partes:=partes[1:10];
 END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves)
 OR NOT (p ?& claves)
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(claves))
 OR array_position(partes,NULL) IS NOT NULL
 THEN RAISE EXCEPTION 'AUT75: campos canon divergentes' USING ERRCODE='22023';END IF;
 FOR i IN 1..cardinality(claves) LOOP
  IF partes[i] IS NULL THEN CONTINUE;END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(claves[i])||':'||partes[i];
 END LOOP;
 RETURN '{'||salida||'}';
END $function$;


CREATE OR REPLACE FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_v2(p jsonb, p_primera boolean)
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
 acciones text[]:=ARRAY['bolsa.inscripcion.rrhh.listar','bolsa.inscripcion.rrhh.consultar',
  'bolsa.inscripcion.rrhh.convocatorias.listar','bolsa.rrhh.respuestas_llamamiento.consultar'];
 tipos text[]:=ARRAY['solicitud_inscripcion','solicitud_inscripcion',
  'conjunto_gestion_inscripcion','llamamiento_bolsa'];
 finalidades text[]:=ARRAY['consulta_inscripcion_rrhh','consulta_inscripcion_rrhh',
  'consulta_convocatorias_gestion_rrhh','seguimiento_contacto_rrhh'];
 hojas_solicitud text[]:=ARRAY['solicitud_ref','recibo_ref','convocatoria_ref',
  'convocatoria_titulo','categoria_ref','bolsa_ref','categoria','declaracion_ref',
  'bases_ref','catalogo_version','plazo_inicio','plazo_fin','decision_ref',
  'estado','version','registrada_en','decidida_en','motivo_codigo',
  'motivo_etiqueta','participacion_ref','requisitos[].codigo',
  'requisitos[].descripcion','requisitos[].obligatorio','requisitos[].estado',
  'requisitos[].motivo_codigo','requisitos[].motivo_etiqueta',
  'requisitos[].procedencia_ref','requisitos[].hito_cumplimiento',
  'requisitos[].hito_etiqueta','requisitos[].hito_fecha','persona_resumen'];
 hojas text[];normalizadas jsonb:='[]'::jsonb;concesion jsonb;ahora timestamptz;
 i integer;campos_actuales jsonb;campos_esperados jsonb;ambitos jsonb;
 limite_entradas timestamptz:='infinity'::timestamptz;
 origen_contacto boolean:=false; origen_inscripcion boolean:=false;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_v2(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 THEN RAISE EXCEPTION 'AUT75: version o base invalida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';ahora:=clock_timestamp();
 origen_contacto:=jsonb_array_length(p->'selecciones')=1;
 IF jsonb_array_length(p->'selecciones')=3 THEN
  origen_inscripcion:=true;
  acciones:=ARRAY['bolsa.inscripcion.rrhh.motivos','bolsa.inscripcion.rrhh.decidir','bolsa.inscripcion.rrhh.incorporar'];
  tipos:=ARRAY['motivos_inscripcion','solicitud_inscripcion','solicitud_inscripcion'];
  finalidades:=ARRAY['consulta_motivos_inscripcion_rrhh','revisar_inscripcion','incorporar_inscripcion'];
 END IF;
 IF origen_contacto THEN
  IF (SELECT count(*) FROM jsonb_array_elements(base#>'{rol,concesiones}') c
      WHERE c->>'accion'=ANY(acciones))<>4 THEN
   RAISE EXCEPTION 'AUT79: falta base B103 de cuatro concesiones' USING ERRCODE='42501';
  END IF;
  acciones:=ARRAY['bolsa.datos_contacto_participacion.origen.consultar'];tipos:=ARRAY['participacion_bolsa'];
  finalidades:=ARRAY['consulta_origen_datos_contacto_participacion'];
 END IF;
 IF p_primera THEN
  SELECT * INTO cabeza FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR cabeza.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR cabeza.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT75: cabeza de catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 -- V2 coteja el paquete de fuentes aprobado; V1 solo sirve para historia B1.
 SELECT convert_from(x.catalogo_canon,'UTF8')::jsonb INTO STRICT catalogo
 FROM vec_autorizacion.resolver_catalogo_acciones_administracion_v2(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256') x;
 IF p_primera AND (ahora<(catalogo->>'vigente_desde')::timestamptz
 OR (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
  AND ahora>=(catalogo->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT75: catalogo no vigente' USING ERRCODE='42501';END IF;
 FOR i IN 0..cardinality(acciones)-1 LOOP
  seleccion:=p->'selecciones'->i;
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(seleccion,'seleccion');
  SELECT value INTO entrada FROM jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT75: descriptor ausente' USING ERRCODE='42501';END IF;
  concesion:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_build_object('concesiones',jsonb_build_array(entrada->'concesion')))#>'{concesiones,0}';
  IF entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
  OR (origen_inscripcion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:inscripcion_recorrido_20261013')
  OR (origen_inscripcion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_inscripcion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '07bf028039044c91a23f09a2d5d5fe0d2af254e2fe85aa7fbbacea7c473487fe')
  OR entrada#>>'{concesion,accion}' IS DISTINCT FROM acciones[i+1]
  OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
  OR entrada#>>'{concesion,tipo_recurso}' IS DISTINCT FROM tipos[i+1]
  OR entrada#>'{concesion,finalidades}' IS DISTINCT FROM jsonb_build_array(finalidades[i+1])
  OR entrada#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
  OR NOT (coalesce(entrada#>'{concesion,obligaciones}','[]'::jsonb)='[]'::jsonb
    OR ((i=3 OR origen_contacto) AND entrada#>'{concesion,obligaciones}'='["auditar"]'::jsonb))
  OR (entrada->'dimensiones_ambito' IS DISTINCT FROM '["unidad_ref","ambito_ref"]'::jsonb
    AND entrada->'dimensiones_ambito' IS DISTINCT FROM '["ambito_ref","unidad_ref"]'::jsonb)
  OR entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
  OR concesion IS DISTINCT FROM nueva->'concesiones'->
    (jsonb_array_length(base#>'{rol,concesiones}')+i)
  THEN RAISE EXCEPTION 'AUT75: descriptor o concesion divergente %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF origen_inscripcion THEN
   hojas:=CASE WHEN i=0 THEN ARRAY['catalogo_version','motivos[].codigo','motivos[].etiqueta','motivos[].obligatorio']::text[] ELSE ARRAY[]::text[] END;
  ELSIF origen_contacto THEN hojas:=ARRAY['enmascarados','origen','participacion_ref','registrada_en','sin_contacto','version'];
  ELSIF i=0 THEN
   SELECT array_agg('solicitudes[].'||x.valor) INTO hojas
   FROM unnest(hojas_solicitud) AS x(valor);
   hojas:=ARRAY['total','cursor_siguiente','convocatoria_titulo']||hojas;
  ELSIF i=1 THEN hojas:=hojas_solicitud;
  ELSIF i=2 THEN
   hojas:=ARRAY['convocatorias[].convocatoria_ref','convocatorias[].titulo',
    'convocatorias[].categorias_resumen','convocatorias[].plazo_fin',
    'convocatorias[].estado_publicacion','convocatorias[].pendientes',
    'total','cursor_siguiente'];
  ELSE hojas:=ARRAY['llamamiento_ref','respuestas[].participacion_ref','respuestas[].respuesta'];
  END IF;
  SELECT coalesce(jsonb_agg(x.valor ORDER BY x.valor COLLATE "C"),'[]'::jsonb) INTO campos_esperados
  FROM unnest(hojas) AS x(valor);
  IF jsonb_typeof(entrada#>'{concesion,campos_permitidos}') IS DISTINCT FROM 'array'
   AND NOT (origen_inscripcion AND i>0 AND NOT (entrada->'concesion' ? 'campos_permitidos'))
  THEN RAISE EXCEPTION 'AUT75: campos ausentes' USING ERRCODE='42501';END IF;
  SELECT coalesce(jsonb_agg(x.valor ORDER BY x.valor COLLATE "C"),'[]'::jsonb) INTO campos_actuales
  FROM jsonb_array_elements_text(coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb)) AS x(valor);
  IF campos_actuales IS DISTINCT FROM campos_esperados
  OR jsonb_array_length(coalesce(entrada#>'{concesion,campos_permitidos}','[]'::jsonb))<>cardinality(hojas)
  THEN RAISE EXCEPTION 'AUT75: hojas fuera de contrato %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF p_primera AND (ahora<(entrada->>'vigente_desde')::timestamptz
   OR (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND ahora>=(entrada->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT75: descriptor caducado %',acciones[i+1] USING ERRCODE='42501';END IF;
  IF entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
   limite_entradas:=LEAST(limite_entradas,(entrada->>'vigente_hasta')::timestamptz);
  END IF;
  normalizadas:=normalizadas||jsonb_build_array(concesion);
 END LOOP;
 IF catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z' THEN
  limite_entradas:=LEAST(limite_entradas,(catalogo->>'vigente_hasta')::timestamptz);
 END IF;
 SELECT * INTO rol FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v'||(base#>>'{rol,version}')
  FOR SHARE;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR rol.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol.documento),'UTF8')),'hex')
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(rol.documento) IS DISTINCT FROM base->'rol'
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.rol_id=rol.rol_id AND z.version>rol.version))
 THEN RAISE EXCEPTION 'AUT75: preimagen del rol divergente' USING ERRCODE='40001';END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=rol.version_rol_ref
 AND x.revision=(base#>>'{control_vigencia,revision}')::numeric FOR SHARE;
 IF NOT FOUND OR control.estado IS DISTINCT FROM 'habilitada'
 OR control.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control.documento),'UTF8')),'hex')
 OR control.documento IS DISTINCT FROM base->'control_vigencia'
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(catalogo->'perfiles') q
  WHERE q->'rol'=base->'rol' AND q->'control_vigencia'=base->'control_vigencia'
   AND q->>'tipo_perfil'='administrable')
 THEN RAISE EXCEPTION 'AUT75: control o perfil catalogado divergente' USING ERRCODE='40001';END IF;
 -- El cierre nuevo retiene el puntero base hasta COMMIT. Una retirada o
 -- avance concurrente no puede intercalarse entre cotejo y efecto.
 IF p_primera THEN
  SELECT q.revision INTO puntero_control
  FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=control.version_rol_ref FOR SHARE;
  IF NOT FOUND OR puntero_control.revision IS DISTINCT FROM control.revision
  THEN RAISE EXCEPTION 'AUT75: control base ya no actual' USING ERRCODE='40001';END IF;
 END IF;
 IF nueva->>'nombre' IS DISTINCT FROM base#>>'{rol,nombre}'
 OR nueva->'concesiones' IS DISTINCT FROM (base#>'{rol,concesiones}')||normalizadas
 OR vec_autorizacion.concesiones_positivas_validas(nueva) IS NOT TRUE
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(base#>'{rol,concesiones}') x
   WHERE (NOT origen_inscripcion AND x->>'accion'=ANY(acciones))
    OR x->>'accion' IN('bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar')
    OR (origen_inscripcion AND x->>'accion'=ANY(acciones)))
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=p->>'version_rol_objetivo_ref'))
 THEN RAISE EXCEPTION 'AUT75: delta de cuatro altas divergente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
  IF t->>'asignacion_ref'=ANY(vistas)
  THEN RAISE EXCEPTION 'AUT75: asignacion duplicada' USING ERRCODE='22023';END IF;
  vistas:=array_append(vistas,t->>'asignacion_ref');original:=t->'documento';
  ambitos:=original->'ambitos';
  IF t->>'asignacion_ref' IS DISTINCT FROM 'asignacion:'||(original->>'asignacion_id')||':v'||(original->>'version')
  OR t->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(original),'UTF8')),'hex')
  OR original->>'version_rol_ref' NOT LIKE 'rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v%'
  OR original->>'estado' IS DISTINCT FROM 'activa'
  OR vec_autorizacion.ambitos_positivos_validos(original) IS NOT TRUE
  OR jsonb_typeof(ambitos) IS DISTINCT FROM 'array' OR jsonb_array_length(ambitos)<>2
  OR (SELECT count(DISTINCT x->>'clave') FROM jsonb_array_elements(ambitos) x)<>2
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ambitos) x WHERE x->>'clave'='unidad_ref')
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ambitos) x WHERE x->>'clave'='ambito_ref')
  THEN RAISE EXCEPTION 'AUT75: asignacion o ambitos invalidos' USING ERRCODE='22023';END IF;
  SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil x
   WHERE x.asignacion_ref=t->>'asignacion_ref' FOR SHARE;
  IF NOT FOUND OR asignacion.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
  OR asignacion.documento IS DISTINCT FROM original
  OR asignacion.principal_id IS DISTINCT FROM original->>'principal_id'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM original->>'perfil_activo_ref'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=asignacion.version_rol_ref AND z.rol_id=rol.rol_id AND z.version<=rol.version)
  THEN RAISE EXCEPTION 'AUT75: asignacion historica divergente' USING ERRCODE='40001';END IF;
  IF p_primera AND (NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q
    WHERE q.perfil_activo_ref=asignacion.perfil_activo_ref AND q.asignacion_ref=asignacion.asignacion_ref)
   OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
   OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
   OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil rev
     WHERE rev.asignacion_id=asignacion.asignacion_id AND rev.documento->>'estado'='revocada'))
  THEN RAISE EXCEPTION 'AUT75: asignacion no actual o vigente' USING ERRCODE='40001';END IF;
 END LOOP;
 RETURN jsonb_build_object('vigente_hasta',CASE WHEN isfinite(limite_entradas)
  THEN to_char(limite_entradas AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  ELSE '0001-01-01T00:00:00Z' END);
END $function$;


CREATE OR REPLACE FUNCTION vec_autorizacion.canon_version_rol_bolsa_externa_v1(p jsonb, t text)
 RETURNS text
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE s text:='';selecciones text:='';salida text:='';e jsonb;partes text[]:=ARRAY[]::text[];claves text[];
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>131072
 THEN RAISE EXCEPTION 'AUT77: canon invalido' USING ERRCODE='22023';END IF;
 CASE t
 WHEN 'material' THEN
  claves:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'OperacionRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'ProponentePersonaRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'PerfilActivoRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'AsignacionPerfilRef'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'Plan','plan')];
 WHEN 'plan' THEN
  claves:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256',
   'version_rol_objetivo_ref','base','definicion_nueva','selecciones','asignaciones','motivo','referencia_acto'];
  IF p->>'operacion' IS DISTINCT FROM 'versionar'
   OR jsonb_typeof(p->'selecciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'selecciones') NOT IN (2,5)
   OR jsonb_typeof(p->'asignaciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'asignaciones') NOT BETWEEN 1 AND 16
   OR (p->>'catalogo_version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT77: plan invalido' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'selecciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF selecciones<>'' THEN selecciones:=selecciones||',';END IF;
   selecciones:=selecciones||vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'seleccion');
  END LOOP;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.canon_version_rol_bolsa_externa_v1(e,'asignacion');
  END LOOP;
  s:='['||s||']';
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'operacion'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_ref'),
   p->>'catalogo_version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_huella_sha256'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'version_rol_objetivo_ref'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'base','base'),
   vec_autorizacion.canon_version_rol_bolsa_externa_v1(p->'definicion_nueva','definicion'),
   '['||selecciones||']',
   s,
   vec_autorizacion.canon_gobierno_rol_nuevo_v1(p->'motivo','motivo'),
   CASE WHEN coalesce(p->>'referencia_acto','')='' THEN NULL
    ELSE vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'referencia_acto') END];
 WHEN 'base' THEN
  claves:=ARRAY['rol','control_vigencia','tipo_perfil'];
  partes:=ARRAY[
   vec_autorizacion.canon_version_rol_admin_v1(p->'rol'),
   vec_autorizacion.canon_control_rol_admin_v1(p->'control_vigencia'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'tipo_perfil')];
 WHEN 'definicion' THEN
  claves:=ARRAY['rol_id','version','nombre','concesiones'];
  IF jsonb_typeof(p->'concesiones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'concesiones') NOT BETWEEN 2 AND 512
   OR (p->>'version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT77: definicion invalida' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'concesiones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.objeto_canonico_go_admin_v1(e,'concesion');
  END LOOP;
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'rol_id'),
   p->>'version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'nombre'),
   '['||s||']'];
 WHEN 'asignacion' THEN
  claves:=ARRAY['asignacion_ref','huella_sha256','documento'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'asignacion_ref'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'huella_sha256'),
   vec_autorizacion.canon_asignacion_perfil_admin_v1(p->'documento')];
 ELSE RAISE EXCEPTION 'AUT77: tipo canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF t='plan' AND partes[11] IS NULL THEN
  IF p ? 'referencia_acto' THEN
   RAISE EXCEPTION 'AUT77: referencia de acto vacía' USING ERRCODE='22023';END IF;
  claves:=claves[1:10];partes:=partes[1:10];
 END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves)
 OR NOT (p ?& claves)
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(claves))
 OR array_position(partes,NULL) IS NOT NULL
 THEN RAISE EXCEPTION 'AUT77: campos canon divergentes' USING ERRCODE='22023';END IF;
 FOR i IN 1..cardinality(claves) LOOP
  IF partes[i] IS NULL THEN CONTINUE;END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(claves[i])||':'||partes[i];
 END LOOP;
 RETURN '{'||salida||'}';
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
 origen_inscripcion boolean:=false;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_externa_v1(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'candidato_bolsa_portal_historial_propio_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 OR (jsonb_array_length(p->'selecciones')=2 AND p#>>'{base,rol,version}' NOT IN ('1','2'))
 OR (jsonb_array_length(p->'selecciones')=5 AND p#>>'{base,rol,version}' NOT IN ('1','2'))
 THEN RAISE EXCEPTION 'AUT77: version o base invalida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';ahora:=clock_timestamp();
 origen_inscripcion:=jsonb_array_length(p->'selecciones')=5;
 IF origen_inscripcion THEN
  acciones:=ARRAY['bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
   'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar','bolsa.inscripcion.presentar'];
  tipos:=ARRAY['convocatoria_inscripcion','convocatoria_inscripcion','solicitud_inscripcion',
   'solicitud_inscripcion','inscripcion_convocatoria'];
  finalidades:=ARRAY['consulta_convocatoria_abierta','consulta_convocatoria_abierta',
   'consulta_inscripcion_propia','consulta_inscripcion_propia','presentar_inscripcion'];
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
 OR rol.documento->>'nombre' IS DISTINCT FROM 'areaPersonal.miBolsa.rolPortal'
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
   (CASE WHEN rol.version=1 THEN 7 WHEN origen_inscripcion THEN 9 ELSE 12 END)
 OR (rol.version=2 AND origen_inscripcion AND NOT (base#>'{rol,concesiones}' @>
    '[{"accion":"bolsa.preferencias_bolsas.consultar_propias"},
      {"accion":"bolsa.preferencias_bolsas.ordenar_propias"}]'::jsonb))
 OR (rol.version=2 AND NOT origen_inscripcion AND NOT (base#>'{rol,concesiones}' @>
    '[{"accion":"bolsa.inscripcion.convocatorias.listar"},
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


CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
AS $function$
DECLARE r record;c record;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('role')<>'none'
 OR vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT77: lectura base denegada' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE rol_id='candidato_bolsa_portal_historial_propio_desarrollo'
 ORDER BY version DESC LIMIT 1 FOR SHARE;
 SELECT h.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual p
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE p.version_rol_ref=r.version_rol_ref FOR SHARE OF p,h;
 IF r.rol_id<>'candidato_bolsa_portal_historial_propio_desarrollo'
 OR r.version NOT IN (1,2) OR r.documento->>'estado'<>'publicada' OR c.estado<>'habilitada'
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR c.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
  vec_autorizacion.canon_control_rol_admin_v1(c.documento),'UTF8')),'hex')
 OR (r.version=1 AND jsonb_array_length(r.documento->'concesiones')<>7)
 OR (r.version=2 AND jsonb_array_length(r.documento->'concesiones') NOT IN (9,12))
 OR (r.version=2 AND jsonb_array_length(r.documento->'concesiones')=9 AND NOT (r.documento->'concesiones' @>
    '[{"accion":"bolsa.preferencias_bolsas.consultar_propias"},
      {"accion":"bolsa.preferencias_bolsas.ordenar_propias"}]'::jsonb))
 OR (r.version=2 AND jsonb_array_length(r.documento->'concesiones')=12 AND NOT (r.documento->'concesiones' @>
    '[{"accion":"bolsa.inscripcion.convocatorias.listar"},
      {"accion":"bolsa.inscripcion.convocatoria.consultar"},
      {"accion":"bolsa.inscripcion.propias.listar"},
      {"accion":"bolsa.inscripcion.propia.consultar"},
      {"accion":"bolsa.inscripcion.presentar"}]'::jsonb))
 THEN RAISE EXCEPTION 'AUT77: base externa divergente' USING ERRCODE='40001';END IF;
 RETURN jsonb_build_object('rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento),
  'control_vigencia',c.documento,'tipo_perfil','administrable');
END $function$;


COMMIT;
