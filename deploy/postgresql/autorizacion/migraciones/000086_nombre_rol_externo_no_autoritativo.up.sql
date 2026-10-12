\set ON_ERROR_STOP on
-- AUT86: retirar sólo la comparación con una clave de presentación textual.
-- La identidad del rol, la base y las concesiones siguen cotejadas por AUT84.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT86: migrador PostgreSQL18 requerido' USING ERRCODE='55000';END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.validar_plan_version_rol_bolsa_externa_v1(jsonb,boolean)');
 h:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF h IS DISTINCT FROM '547155fdec8425fec4b4a9c74cf84c8bf8131a58de71fb7503e22270ee665ac4'
 OR p.proowner<>to_regrole('vec_autorizacion_propietario')
 OR p.prosecdef IS NOT TRUE
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','row_security=on']::text[]
 OR p.proacl IS DISTINCT FROM ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario'::aclitem]
 THEN RAISE EXCEPTION 'AUT86: preimagen divergente actual=% esperado=547155fdec8425fec4b4a9c74cf84c8bf8131a58de71fb7503e22270ee665ac4',h USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
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
COMMIT;
