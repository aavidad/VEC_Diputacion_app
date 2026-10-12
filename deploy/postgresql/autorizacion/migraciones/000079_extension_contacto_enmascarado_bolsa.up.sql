\set ON_ERROR_STOP on
-- AUT79: literal obtenido del clon HZ21 post AD248; instalación única.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE h text;
BEGIN
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.canon_version_rol_bolsa_v2(jsonb,text)');
 IF h IS DISTINCT FROM 'd4251e52a9d7be749ffa6631ec116c6b30ef34e6c5c4c14b9443a775dc8cbdd8' THEN RAISE EXCEPTION 'PARO clave=AUT79.canon_version_rol_bolsa_v2 actual=% esperado=d4251e52a9d7be749ffa6631ec116c6b30ef34e6c5c4c14b9443a775dc8cbdd8',h USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.validar_plan_version_rol_bolsa_v2(jsonb,boolean)');
 IF h IS DISTINCT FROM '6cce8e05cf80a107089d4ae5bc3a95125c311b724fecef65a5bc12d9349c6845' THEN RAISE EXCEPTION 'PARO clave=AUT79.validar_plan_version_rol_bolsa_v2 actual=% esperado=6cce8e05cf80a107089d4ae5bc3a95125c311b724fecef65a5bc12d9349c6845',h USING ERRCODE='55000'; END IF;
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
   OR jsonb_array_length(p->'selecciones') NOT IN (1,4)
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
 origen_contacto boolean:=false;
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
  IF origen_contacto THEN hojas:=ARRAY['enmascarados','origen','participacion_ref','registrada_en','sin_contacto','version'];
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
  SELECT jsonb_agg(x.valor ORDER BY x.valor COLLATE "C") INTO campos_esperados
  FROM unnest(hojas) AS x(valor);
  IF jsonb_typeof(entrada#>'{concesion,campos_permitidos}') IS DISTINCT FROM 'array'
  THEN RAISE EXCEPTION 'AUT75: campos ausentes' USING ERRCODE='42501';END IF;
  SELECT jsonb_agg(x.valor ORDER BY x.valor COLLATE "C") INTO campos_actuales
  FROM jsonb_array_elements_text(entrada#>'{concesion,campos_permitidos}') AS x(valor);
  IF campos_actuales IS DISTINCT FROM campos_esperados
  OR jsonb_array_length(entrada#>'{concesion,campos_permitidos}')<>cardinality(hojas)
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
   WHERE x->>'accion'=ANY(acciones)
    OR x->>'accion' IN('bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar'))
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

COMMIT;
