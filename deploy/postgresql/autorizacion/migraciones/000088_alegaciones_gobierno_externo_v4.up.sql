\set ON_ERROR_STOP on
-- AUT88: extiende el fabricante ADMIN existente sólo con dos acciones
-- prospectivas de Alegaciones sobre la cabeza externa v3 de #1079.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'c6e72de45b788b35ecdc16b0efd0d8e392d066c60c6d148935c1a70efc7ccbd3'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_plan_version_rol_bolsa_v2(jsonb,boolean)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '845cb4b1821672af6ea51dc8f92f587c4826808923b2c4efb3332b27a008e4da'
 THEN RAISE EXCEPTION 'AUT88: preimagen externa v3 divergente' USING ERRCODE='55000';END IF;
END $pre$;
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
 origen_contacto boolean:=false; origen_inscripcion boolean:=false; origen_alegacion boolean:=false;
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
 origen_alegacion:=jsonb_array_length(p->'selecciones')=1 AND p#>>'{selecciones,0,entrada_ref}'='accion:bolsa.inscripcion.alegacion.rrhh.consultar';
 origen_contacto:=jsonb_array_length(p->'selecciones')=1 AND NOT origen_alegacion;
 IF jsonb_array_length(p->'selecciones')=3 THEN
  origen_inscripcion:=true;
  acciones:=ARRAY['bolsa.inscripcion.rrhh.motivos','bolsa.inscripcion.rrhh.decidir','bolsa.inscripcion.rrhh.incorporar'];
  tipos:=ARRAY['motivos_inscripcion','solicitud_inscripcion','solicitud_inscripcion'];
  finalidades:=ARRAY['consulta_motivos_inscripcion_rrhh','revisar_inscripcion','incorporar_inscripcion'];
 END IF;
 IF origen_contacto OR origen_alegacion THEN
  IF (SELECT count(*) FROM jsonb_array_elements(base#>'{rol,concesiones}') c
      WHERE c->>'accion'=ANY(acciones))<>4 THEN
   RAISE EXCEPTION 'AUT79: falta base B103 de cuatro concesiones' USING ERRCODE='42501';
  END IF;
  acciones:=ARRAY['bolsa.datos_contacto_participacion.origen.consultar'];tipos:=ARRAY['participacion_bolsa'];
  finalidades:=ARRAY['consulta_origen_datos_contacto_participacion'];
  IF origen_alegacion THEN
   acciones:=ARRAY['bolsa.inscripcion.alegacion.rrhh.consultar'];tipos:=ARRAY['alegaciones_bolsa'];
   finalidades:=ARRAY['consulta_alegaciones_rrhh'];
  END IF;
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
  OR (origen_alegacion AND entrada->>'fuente_ref' IS DISTINCT FROM 'fuente:bolsa:alegaciones_propias_v1')
  OR (origen_alegacion AND entrada->>'fuente_version' IS DISTINCT FROM '1')
  OR (origen_alegacion AND entrada->>'fuente_huella_sha256' IS DISTINCT FROM '5b9f0fc2d99f4c2cf7b08829bb56bcddd5262c5cb44cc797b48a3f1f072a1060')
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
  ELSIF origen_alegacion THEN hojas:=ARRAY['alegacion_ref','alegaciones[].alegacion_ref','alegaciones[].auditoria_ref','alegaciones[].estado','alegaciones[].fundamento','alegaciones[].objeto.acto_ref','alegaciones[].objeto.participacion_ref','alegaciones[].objeto.solicitud_ref','alegaciones[].objeto.tipo','alegaciones[].objeto.version_acto','alegaciones[].persona_resumen','alegaciones[].presentada_en','alegaciones[].recibo_ref','ambitos[].referencia','ambitos[].tipo','ambitos[].titulo','auditoria_ref','cursor_siguiente','estado','fundamento','objeto.acto_ref','objeto.participacion_ref','objeto.solicitud_ref','objeto.tipo','objeto.version_acto','persona_resumen','presentada_en','recibo_ref','total']::text[];
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

DO $post$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'fe6f52d34cfb010dcbd8875230cb4b4e6f6d7ed3613fad88b724f267ddedf0e1'
 THEN RAISE EXCEPTION 'AUT88: postimagen externa v4 divergente' USING ERRCODE='55000';END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.validar_plan_version_rol_bolsa_v2(jsonb,boolean)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '1c0cf66d59e48ce6f97b5eee492591402d72d2e63ce12c13570c765982f3a65f'
 THEN RAISE EXCEPTION 'AUT88: postimagen RRHH divergente' USING ERRCODE='55000';END IF;
END $post$;

-- Sólo se admiten las nuevas familias con su acción exacta; los tres
-- catálogos Mi Bolsa y sus roles históricos conservan la misma semántica.
DO $fachadas_pre$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '9c769c3d153ce749fa20013878b4e6f9b4a4d47747ad3c11f7842e584233dda7'
 THEN RAISE EXCEPTION 'AUT88: fachadas_pre registrar_decision_candidato_externo_v3 divergente' USING ERRCODE='55000';END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'fecb798afe9ddb48f55a2f0a160dadc6d7d2d9f4eae002436c70f6e818f83e43'
 THEN RAISE EXCEPTION 'AUT88: fachadas_pre registrar_y_revalidar_decision_contexto_actor_externa_v3 divergente' USING ERRCODE='55000';END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamp with time zone)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'bcae4e6146e2906bbdf31ddb1c5286e139b1d5f220ca031cd562c8d5cc534383'
 THEN RAISE EXCEPTION 'AUT88: fachadas_pre resolver_motivo_candidato_externo_v1 divergente' USING ERRCODE='55000';END IF;
END $fachadas_pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_motivo_candidato_externo_v1(p_catalogo_id text, p_catalogo_version integer, p_huella text, p_entrada text, p_instante timestamp with time zone)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_motivos_autorizacion_desarrollo','vec_autorizacion_motivos_externos') IS NOT TRUE
    OR p_catalogo_id NOT IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo','motivos_inscripcion_bolsa_desarrollo','motivos_alegaciones_bolsa')
 THEN RAISE EXCEPTION 'motivo externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_historico(p_catalogo_id,p_catalogo_version,p_huella,p_entrada,p_instante);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_candidato_externo_v3(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; v jsonb; m jsonb;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d := pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m := pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END;
 v := d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR NOT (m->'referencia'->>'catalogo_id' IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
      OR (m->'referencia'->>'catalogo_id'='motivos_inscripcion_bolsa_desarrollo' AND d->>'accion'='bolsa.inscripcion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_alegaciones_bolsa' AND d->>'accion'='bolsa.inscripcion.alegacion.presentar'))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      WHERE r.version_rol_ref=d->>'version_rol_ref'
        AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro externo rechazado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(p_decision,p_motivo,p_persona_version,p_perfil_version);
END $function$;
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone, revalidada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; m jsonb; v jsonb; registro record; viva timestamptz;
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo') IS NOT TRUE
    OR p_decision IS NULL OR p_motivo IS NULL
    OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  m:=pg_catalog.convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END;
 v:=d->'vinculo_autenticacion_actor';
 IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
    OR vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
    OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
    OR NOT (m->'referencia'->>'catalogo_id' IN ('motivos_mi_bolsa_desarrollo','motivos_historial_mi_bolsa_desarrollo','motivos_portal_mi_bolsa_desarrollo')
      OR (m->'referencia'->>'catalogo_id'='motivos_inscripcion_bolsa_desarrollo' AND d->>'accion'='bolsa.inscripcion.presentar')
      OR (m->'referencia'->>'catalogo_id'='motivos_alegaciones_bolsa' AND d->>'accion'='bolsa.inscripcion.alegacion.presentar'))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
       WHERE r.version_rol_ref=d->>'version_rol_ref'
         AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo'))
 THEN RAISE EXCEPTION 'registro AD3 externo rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND OR registro.concedida IS NOT TRUE THEN RETURN; END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN; END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,registro.registrada_en,viva;
END $function$;

DO $fachadas_post$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '903570e631b0aea0197ec722827bc234d0754c3ad5e383bbdd786835925eb2b8'
 THEN RAISE EXCEPTION 'AUT88: fachadas_post registrar_decision_candidato_externo_v3 divergente' USING ERRCODE='55000';END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '90abe20e791c6c0ba0f8883d6014cd4403c54400bfea944cc4ec6aaf66c03100'
 THEN RAISE EXCEPTION 'AUT88: fachadas_post registrar_y_revalidar_decision_contexto_actor_externa_v3 divergente' USING ERRCODE='55000';END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamp with time zone)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '3dbdb5273e89873cee995c12ff84a31fb174caf5fc1e0ef834a662b1eab7c177'
 THEN RAISE EXCEPTION 'AUT88: fachadas_post resolver_motivo_candidato_externo_v1 divergente' USING ERRCODE='55000';END IF;
END $fachadas_post$;

RESET ROLE;
COMMIT;
