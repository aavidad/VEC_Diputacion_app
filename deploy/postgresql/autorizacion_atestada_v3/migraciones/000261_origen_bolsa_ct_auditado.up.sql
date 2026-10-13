\set ON_ERROR_STOP on
-- AD261. Frontera de lectura prospectiva CT: el LOGIN técnico sólo ve la
-- referencia opaca para crear una captura actual. El contenido sale después
-- del cotejo y del asiento positivo dentro de la misma transacción.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000261',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(bytea,bytea,text,text,text,text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(text,text,text,text)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'AD261: preimagen o dependencias ausentes' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p
 WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)');
 IF h IS DISTINCT FROM 'ae6152998c778cff5e6385e2cee65184480cc97ee5e2efa50bce01fd93d098fd'
 THEN RAISE EXCEPTION 'AD261: AD259 no instalada o divergente' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO h FROM pg_proc p
 WHERE p.oid=to_regprocedure('vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(jsonb,text,text)');
 IF h IS DISTINCT FROM 'cd28e4aaae2dcf55c098941165028cb8cd41be98b439d836437a4ced390dc091'
 THEN RAISE EXCEPTION 'AD261: AUT89 no instalada o divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(
 p_contexto bytea,p_vinculo bytea,p_expediente text,p_clave text,p_categoria text,p_unidad text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;v jsonb;s record;bolsa text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR vec_autorizacion_atestada_v3.acreditar_registrador_intentos_v1('vec-rrhh','interna_corporativa') IS NOT TRUE
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536
 OR p_vinculo IS NULL OR octet_length(p_vinculo) NOT BETWEEN 1 AND 16384
 OR p_expediente IS NULL OR p_clave IS NULL OR p_categoria IS NULL OR p_unidad IS NULL
 OR p_categoria<>btrim(p_categoria) OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
 OR p_unidad !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
 THEN RAISE EXCEPTION 'AD261: lectura denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_contexto,'UTF8')::jsonb;
       v:=convert_from(p_vinculo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD261: lectura denegada' USING ERRCODE='42501'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(v) IS DISTINCT FROM 'object'
 OR c->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v->>'principal_id' IS DISTINCT FROM c->>'principal_ref'
 OR v->>'perfil_activo_ref' IS DISTINCT FROM c->>'perfil_activo_ref'
 OR v->>'cuenta_ref' IS DISTINCT FROM c->>'cuenta_ref'
 OR v->>'contexto_actor_ref' IS DISTINCT FROM c->>'contexto_actor_ref'
 OR v->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(p_contexto) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(p_vinculo) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD261: lectura denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
   v->>'autenticacion_ref',v->>'sesion_ref');
 IF NOT FOUND OR s.autenticacion_huella_sha256 IS DISTINCT FROM v->>'autenticacion_huella_sha256'
 OR s.cuenta_ref IS DISTINCT FROM c->>'cuenta_ref'
 OR s.cuenta_ordinaria_ref IS DISTINCT FROM v->>'cuenta_ordinaria_ref'
 OR s.control_sesion_ref IS DISTINCT FROM v->>'control_sesion_ref'
 OR s.control_sesion_revision IS DISTINCT FROM v->>'control_sesion_revision'
 OR s.control_sesion_huella_sha256 IS DISTINCT FROM v->>'control_sesion_huella_sha256'
 OR s.superficie IS DISTINCT FROM 'interna_corporativa'
 OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.cuenta_privilegiada IS NOT FALSE
 OR s.sesion_valida_hasta IS DISTINCT FROM (v->>'sesion_valida_hasta')::timestamptz
 OR s.sesion_revalidada_en IS DISTINCT FROM (v->>'sesion_revalidada_en')::timestamptz
 THEN RAISE EXCEPTION 'AD261: lectura denegada' USING ERRCODE='42501'; END IF;
 SELECT vec_bolsa_llamamientos.resolver_bolsa_origen_ct_v1(
  p_expediente,p_clave,p_categoria,p_unidad) INTO bolsa;
 IF bolsa IS NULL OR bolsa='' THEN
  RAISE EXCEPTION 'AD261: origen ausente' USING ERRCODE='42501'; END IF;
 RETURN bolsa;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception OR invalid_text_representation THEN
 RAISE EXCEPTION 'AD261: origen no disponible' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(bytea,bytea,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(bytea,bytea,text,text,text,text)
 TO vec_autorizacion_atestada_v3_registrador_intentos;
-- El permiso se coteja antes de obtener candidatos y canónicos. La reserva
-- histórica se relee bajo la misma autoridad actual; nunca da acceso por sí.
CREATE FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb,p_auditoria jsonb,
 p_expediente text,p_clave text,p_categoria text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;v jsonb;bolsa text;reserva jsonb;origen jsonb;lectura_raw jsonb;
        lectura record;asiento record;
BEGIN
 IF jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.origen_ct.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_origen_ct_correcta'
 OR p_orden->>'unidad_ref' IS NULL OR p_orden->>'ambito_ref' IS NULL
 OR p_orden->>'recurso_ref' IS DISTINCT FROM p_orden->>'bolsa_ref'
 OR p_orden->>'contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex')
 THEN RAISE EXCEPTION 'AD261: captura de origen inválida' USING ERRCODE='42501'; END IF;
 bolsa:=vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(
   p_contexto,p_vinculo,p_expediente,p_clave,p_categoria,p_orden->>'unidad_ref');
 c:=convert_from(p_contexto,'UTF8')::jsonb;
 v:=convert_from(p_vinculo,'UTF8')::jsonb;
 IF p_orden->>'recurso_ref' IS DISTINCT FROM bolsa
 OR p_orden->>'bolsa_ref' IS DISTINCT FROM bolsa
 OR p_orden->>'registro_contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref'
 OR p_orden->>'procedencia_sha256' IS DISTINCT FROM v->>'manifiesto_procedencia_huella_sha256'
 OR p_orden->>'autenticacion_ref' IS DISTINCT FROM v->>'autenticacion_ref'
 OR p_orden->>'sesion_ref' IS DISTINCT FROM v->>'sesion_ref'
 OR p_orden->>'autenticacion_sha256' IS DISTINCT FROM v->>'autenticacion_huella_sha256'
 OR vec_autorizacion.cotejar_lectura_bolsa_actual_interna_v1(
    p_orden,c->>'principal_ref',c->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD261: autoridad actual denegada' USING ERRCODE='42501'; END IF;
 reserva:=vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(p_expediente,p_clave);
 -- La lectura histórica se proyecta una sola vez. Los campos de reserva
 -- se extraen después, sin devolver origen_ct ni contactos de B7.
 IF reserva IS NOT NULL THEN
  lectura_raw:=reserva->'origen_ct'->'lectura';
  IF reserva->'origen_ct'->>'expediente_ref' IS DISTINCT FROM p_expediente
  OR reserva->'origen_ct'->>'clave_ct' IS DISTINCT FROM p_clave
  OR reserva->'origen_ct'->>'unidad_ref' IS DISTINCT FROM p_orden->>'unidad_ref'
  OR reserva->'origen_ct'->>'bolsa_ref' IS DISTINCT FROM bolsa
  OR reserva->>'bolsa_ref' IS DISTINCT FROM bolsa
  OR reserva->'origen_ct'->>'llamamiento_ref' IS DISTINCT FROM reserva->>'llamamiento_ref'
  OR reserva->'origen_ct'->>'recibo_ref' IS DISTINCT FROM reserva->>'recibo_ref'
  OR lectura_raw->>'categoria_ref' IS DISTINCT FROM p_categoria
  OR lectura_raw->>'bolsa_ref' IS DISTINCT FROM bolsa
  THEN RAISE EXCEPTION 'AD261: reserva de origen incompatible' USING ERRCODE='42501'; END IF;
 ELSE
  SELECT * INTO STRICT lectura FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(p_categoria);
  IF lectura.bolsa_ref IS DISTINCT FROM bolsa
  THEN RAISE EXCEPTION 'AD261: fuente de origen cambió' USING ERRCODE='40001'; END IF;
  lectura_raw:=to_jsonb(lectura)||jsonb_build_object(
    'huella_politica_sha256',encode(sha256(lectura.politica_canonica),'hex'));
 END IF;
 -- Lista blanca estructural: una columna nueva de B110 o una propiedad nueva
 -- en la reserva no se publica de forma implícita.
 IF jsonb_typeof(lectura_raw) IS DISTINCT FROM 'object'
 OR lectura_raw ?& ARRAY['acta_ref','categoria_ref','bolsa_ref','version_bolsa',
   'huella_bolsa_sha256','bolsa_canonica','instantanea_ref','version_instantanea',
   'huella_instantanea_sha256','instantanea_canonica','politica_ref','version_politica',
   'huella_politica_sha256','politica_provisional','corte','entradas'] IS NOT TRUE
 OR jsonb_typeof(lectura_raw->'entradas') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AD261: origen incompleto' USING ERRCODE='42501'; END IF;
 origen:=jsonb_build_object(
  'acta_ref',lectura_raw->'acta_ref',
  'categoria_ref',lectura_raw->'categoria_ref',
  'bolsa_ref',lectura_raw->'bolsa_ref',
  'version_bolsa',lectura_raw->'version_bolsa',
  'huella_bolsa_sha256',lectura_raw->'huella_bolsa_sha256',
  'bolsa_canonica',lectura_raw->'bolsa_canonica',
  'instantanea_ref',lectura_raw->'instantanea_ref',
  'version_instantanea',lectura_raw->'version_instantanea',
  'huella_instantanea_sha256',lectura_raw->'huella_instantanea_sha256',
  'instantanea_canonica',lectura_raw->'instantanea_canonica',
  'politica_ref',lectura_raw->'politica_ref',
  'version_politica',lectura_raw->'version_politica',
  'huella_politica_sha256',lectura_raw->'huella_politica_sha256',
  'politica_provisional',lectura_raw->'politica_provisional',
  'corte',lectura_raw->'corte',
  'entradas',(SELECT jsonb_agg(jsonb_build_object(
   'orden_acta',x.valor->'orden_acta',
   'orden_vigente',x.valor->'orden_vigente',
   'participacion_ref',x.valor->'participacion_ref',
   'candidato_ref',x.valor->'candidato_ref',
   'situacion',x.valor->'situacion',
   'razon',x.valor->'razon',
   'elegible',x.valor->'orden_vigente'<>'null'::jsonb) ORDER BY x.n)
   FROM jsonb_array_elements(lectura_raw->'entradas') WITH ORDINALITY x(valor,n)));
 -- AD259 admite únicamente este éxito nominal. La proyección a la cadena
 -- común se hace aquí, tras el cotejo, y se revierte junto con la lectura.
 IF jsonb_typeof(p_auditoria) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_auditoria))<>16
 OR p_auditoria->>'intento_ref' !~ '^intento_[0-9a-f]{32}$'
 OR p_auditoria->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR p_auditoria->>'contexto_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR p_auditoria->>'procedencia_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR p_auditoria->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR p_auditoria->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR p_auditoria->>'autenticacion_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR p_auditoria->>'accion' IS DISTINCT FROM p_orden->>'accion'
 OR p_auditoria->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR p_auditoria->>'recurso_ref' IS DISTINCT FROM bolsa
 OR p_auditoria->>'finalidad_ref' IS DISTINCT FROM p_orden->>'finalidad_ref'
 OR p_auditoria->>'resultado' IS DISTINCT FROM 'exito'
 OR p_auditoria->>'proceso' IS DISTINCT FROM 'vec-rrhh'
 OR p_auditoria->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_auditoria->>'motivo_ref' IS NULL
 OR p_auditoria->>'correlacion_ref' IS NULL
 THEN RAISE EXCEPTION 'AD261: asiento de origen incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT asiento FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
  p_contexto,p_vinculo,p_auditoria);
 IF reserva IS NOT NULL THEN
  IF jsonb_typeof(reserva->'configuracion') IS DISTINCT FROM 'object'
   OR (SELECT count(*) FROM jsonb_object_keys(reserva->'configuracion'))<>10
   OR (reserva->'configuracion') ?& ARRAY['referencia','descripcion','categoria',
       'centro','modalidad','fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE
   OR EXISTS(SELECT 1 FROM jsonb_each(reserva->'configuracion') x
       WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
       OR octet_length(x.value#>>'{}') NOT BETWEEN 2 AND 4000)
  THEN RAISE EXCEPTION 'AD261: configuración de replay inválida' USING ERRCODE='42501'; END IF;
  reserva:=jsonb_build_object(
   'expediente_ref',reserva->'origen_ct'->>'expediente_ref',
   'clave_ct',reserva->'origen_ct'->>'clave_ct',
   'unidad_ref',reserva->'origen_ct'->>'unidad_ref',
   'participacion_ref',reserva->'origen_ct'->>'participacion_ref',
   'candidato_ref',reserva->'origen_ct'->>'candidato_ref',
   'orden_seleccionado',reserva->'origen_ct'->'orden_seleccionado',
   'configuracion',reserva->'configuracion');
 END IF;
 RETURN jsonb_build_object('origen',origen,'reserva',reserva,
  'acuse',jsonb_build_object('auditoria_ref',asiento.auditoria_ref,
   'secuencia',asiento.secuencia,'huella_sha256',asiento.huella_sha256,
   'correlacion_ref',asiento.correlacion_ref,'registrada_en',asiento.registrada_en));
EXCEPTION WHEN data_exception OR invalid_text_representation THEN
 RAISE EXCEPTION 'AD261: lectura no disponible' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)
 TO vec_autorizacion_atestada_v3_registrador_intentos;
COMMIT;
