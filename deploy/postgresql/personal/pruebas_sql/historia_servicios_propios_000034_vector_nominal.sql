\set ON_ERROR_STOP on
-- BORRADOR. Ensayo futuro por Dirección, exclusivamente en clon sintético.
-- Requiere conexión del LOGIN nominal segregado y un bundle nuevo generado
-- por el emisor Go real sobre AD180 y sus autoridades sintéticas del ensayo.
-- Este archivo no crea permisos, identidades, claves, firmas ni dobles SQL.
-- Variables privadas: historia_material y diez piezas del bundle, hex para
-- bytes. No imprime el material. El ensayo entero termina en ROLLBACK.
\if :{?historia_material}
\else
\echo 'Falta el vector nominal privado; ensayo no ejecutable.'
\quit 3
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT set_config('vec_test.historia.material', :'historia_material',true) AS material,
 set_config('vec_test.historia.capacidad', :'historia_capacidad_hex',true) AS capacidad,
 set_config('vec_test.historia.decision', :'historia_decision_hex',true) AS decision,
 set_config('vec_test.historia.motivo', :'historia_motivo_hex',true) AS motivo,
 set_config('vec_test.historia.contexto', :'historia_contexto_hex',true) AS contexto,
 set_config('vec_test.historia.persona_version', :'historia_persona_version',true) AS persona_version,
 set_config('vec_test.historia.perfil_version', :'historia_perfil_version',true) AS perfil_version,
 set_config('vec_test.historia.payload', :'historia_payload_hex',true) AS payload,
 set_config('vec_test.historia.sobre', :'historia_sobre_hex',true) AS sobre,
 set_config('vec_test.historia.evidencia', :'historia_evidencia_hex',true) AS evidencia,
 set_config('vec_test.historia.raiz', :'historia_raiz_hex',true) AS raiz
\gset historia_preparado_
DO $vector$
DECLARE
 m text:=current_setting('vec_test.historia.material');
 capacidad bytea:=decode(current_setting('vec_test.historia.capacidad'),'hex');
 decision bytea:=decode(current_setting('vec_test.historia.decision'),'hex');
 motivo bytea:=decode(current_setting('vec_test.historia.motivo'),'hex');
 contexto bytea:=decode(current_setting('vec_test.historia.contexto'),'hex');
 persona_version numeric:=current_setting('vec_test.historia.persona_version')::numeric;
 perfil_version numeric:=current_setting('vec_test.historia.perfil_version')::numeric;
 payload bytea:=decode(current_setting('vec_test.historia.payload'),'hex');
 sobre bytea:=decode(current_setting('vec_test.historia.sobre'),'hex');
 evidencia bytea:=decode(current_setting('vec_test.historia.evidencia'),'hex');
 raiz bytea:=decode(current_setting('vec_test.historia.raiz'),'hex');
 respuesta jsonb; alterado text; denegada boolean; fila jsonb; anterior jsonb; identidades text[]:=ARRAY[]::text[]; identidad text;
BEGIN
 -- Un cambio de corte con las mismas firmas nunca amplía la lectura.
 alterado:=jsonb_set(m::jsonb,'{efectos_hasta}',to_jsonb('2099-12-31'::text))::text;
 denegada:=false;
 BEGIN
  PERFORM vec_personal.consultar_historia_servicios_propios_empleado_v1(alterado,capacidad,decision,motivo,contexto,persona_version,perfil_version,payload,sobre,evidencia,raiz);
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'Personal34: corte alterado admitido'; END IF;
 -- El bundle de la misma persona con otra revisión no es intercambiable.
 denegada:=false;
 BEGIN
  PERFORM vec_personal.consultar_historia_servicios_propios_empleado_v1(m,capacidad,decision,motivo,contexto,persona_version+1,perfil_version,payload,sobre,evidencia,raiz);
 EXCEPTION WHEN SQLSTATE '42501' THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'Personal34: versión de persona alterada admitida'; END IF;
 respuesta:=vec_personal.consultar_historia_servicios_propios_empleado_v1(m,capacidad,decision,motivo,contexto,persona_version,perfil_version,payload,sobre,evidencia,raiz);
 IF ARRAY(SELECT jsonb_object_keys(respuesta) ORDER BY 1) IS DISTINCT FROM ARRAY['evidencia','historia']
    OR ARRAY(SELECT jsonb_object_keys(respuesta->'historia') ORDER BY 1) IS DISTINCT FROM ARRAY['cobertura','corte','empleado_ref','revisiones']
    OR ARRAY(SELECT jsonb_object_keys(respuesta->'historia'->'corte') ORDER BY 1) IS DISTINCT FROM ARRAY['conocido_en','efectos_desde','efectos_hasta']
    OR respuesta#>>'{historia,empleado_ref}' IS DISTINCT FROM m::jsonb->>'empleado_ref'
    OR respuesta#>>'{historia,cobertura}' IS DISTINCT FROM 'no_acreditada'
    OR respuesta#>>'{historia,corte,efectos_desde}' IS DISTINCT FROM m::jsonb->>'efectos_desde'
    OR respuesta#>>'{historia,corte,efectos_hasta}' IS DISTINCT FROM m::jsonb->>'efectos_hasta'
    OR respuesta#>>'{historia,corte,conocido_en}' IS DISTINCT FROM m::jsonb->>'conocido_en'
    OR jsonb_typeof(respuesta#>'{historia,revisiones}') IS DISTINCT FROM 'array'
    OR jsonb_array_length(respuesta#>'{historia,revisiones}')>200
    OR coalesce(respuesta#>>'{evidencia,consumo_huella_sha256}','') !~ '^[0-9a-f]{64}$'
    OR respuesta#>>'{evidencia,recibo_ref}' IS DISTINCT FROM respuesta#>>'{evidencia,auditoria_ref}'
    OR respuesta#>>'{evidencia,recibo_ref}' IS DISTINCT FROM 'aud_v3_'||left(respuesta#>>'{evidencia,consumo_huella_sha256}',32) THEN
  RAISE EXCEPTION 'Personal34: respuesta divergente'; END IF;
 FOR fila IN SELECT e FROM jsonb_array_elements(respuesta#>'{historia,revisiones}') x(e) LOOP
  IF ARRAY(SELECT jsonb_object_keys(fila) ORDER BY 1) IS DISTINCT FROM ARRAY['clase','dias_reconocidos','estado','periodo_desde','periodo_hasta','relacion_ref','servicio_ref','traza'] THEN
   RAISE EXCEPTION 'Personal34: columnas extra en revisión'; END IF;
  identidad:=(fila->>'servicio_ref')||':'||(fila#>>'{traza,version}');
  IF identidad=ANY(identidades) THEN RAISE EXCEPTION 'Personal34: revisión duplicada'; END IF;
  identidades:=array_append(identidades,identidad);
  IF anterior IS NOT NULL AND ROW(anterior#>>'{traza,desde}',anterior#>>'{traza,registrada_en}')<ROW(fila#>>'{traza,desde}',fila#>>'{traza,registrada_en}') THEN
   RAISE EXCEPTION 'Personal34: revisiones fuera de orden temporal'; END IF;
  anterior:=fila;
 END LOOP;
 -- No reutilizar la misma capacidad: no añade otra lectura confirmada.
 denegada:=false;
 BEGIN
  PERFORM vec_personal.consultar_historia_servicios_propios_empleado_v1(m,capacidad,decision,motivo,contexto,persona_version,perfil_version,payload,sobre,evidencia,raiz);
 EXCEPTION WHEN SQLSTATE '42501' THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'Personal34: capacidad ya consumida admitida'; END IF;
END $vector$;
ROLLBACK;
