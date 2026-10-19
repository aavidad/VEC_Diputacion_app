\set ON_ERROR_STOP on
-- AD300 en copia desechable, después de B138, B143 y AD300. Todo revierte.
-- Requiere sesión de superusuario para adoptar el LOGIN externo de la copia.
-- La lectura de preferencias solo admite el motivo que corresponde a la
-- proyección pedida; la restricción de formato admite los dos motivos.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $metadatos$
DECLARE def text;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
   'vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM 'c1d5c3f79c6ed910f57a2a9c6ef2685eb999537e695bcb265a625bc04473ca2f'
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '4de97da400231c21a6704846808782a2dcf3d0acaf7a462f34d6e4b83772d223'
 THEN RAISE EXCEPTION 'AD300: postimagen distinta de la migración' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT def FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(def,'UTF8')),'hex')
   IS DISTINCT FROM '9364e6e24509ac95340610b822310a82d1d556f5ad39f372907bc269b6de260e'
 OR strpos(def,'interna_corporativa')=0
 OR strpos(def,'''lectura_preferencias_bolsas_correcta''::text, ''lectura_preferencias_modalidades_correcta''::text')=0
 THEN RAISE EXCEPTION 'AD300: restricción de formato divergente' USING ERRCODE='55000'; END IF;
END $metadatos$;

-- La lectura de Bolsa y su fachada auditada comprueban el LOGIN de sesión.
SET LOCAL SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $casos$
DECLARE cand text:='can_'||repeat('a',43);
 c8 jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]';
 c5 jsonb:='["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 base jsonb;caso text;campos jsonb;motivo text;e text;m text;r jsonb:='{}'::jsonb;
BEGIN
 base:=jsonb_build_object(
  'intento_ref','lectura_'||repeat('0',32),'registro_contexto_ref','registro:ad300',
  'contexto_sha256',repeat('1',64),'procedencia_sha256',repeat('2',64),
  'autenticacion_ref','autenticacion:ad300','sesion_ref','sesion:ad300',
  'autenticacion_sha256',repeat('3',64),'accion','bolsa.preferencias_bolsas.consultar_propias',
  'modulo_id','bolsa','recurso_ref','mi-bolsa:'||cand,'candidato_ref',cand,
  'tipo_recurso','participaciones_candidato','finalidad_ref','gestion_preferencias_bolsas_propias',
  'resultado','obtenida','proceso','vec-server','canal','externa_personal',
  'correlacion_ref','correlacion-ad300','lectura_revision_permisos','1',
  'lectura_instantanea_sha256',repeat('4',64),'asignacion_ref','asignacion:ad300',
  'asignacion_version',1,'asignacion_huella_sha256',repeat('5',64),
  'version_rol_ref','rol:ad300:v1','version_rol_version',1,'version_rol_huella_sha256',repeat('6',64),
  'control_version_rol_ref','control:ad300','revision_control_version_rol',1,
  'control_version_rol_huella_sha256',repeat('7',64),'revision_catalogo_politicas',1,
  'catalogo_politicas_huella_sha256',repeat('8',64),'politicas_evaluadas','[]'::jsonb,
  'emitida_en','2026-10-19T00:00:00Z','valida_hasta','2026-10-19T00:00:30Z');
 FOREACH caso IN ARRAY ARRAY['cinco_motivo_bolsas','ocho_motivo_modalidades','cinco_motivo_ajeno',
  'cinco_motivo_modalidades','ocho_motivo_bolsas'] LOOP
  campos:=CASE WHEN caso LIKE 'cinco%' THEN c5 ELSE c8 END;
  motivo:=CASE WHEN caso LIKE '%motivo_bolsas' THEN 'lectura_preferencias_bolsas_correcta'
   WHEN caso LIKE '%motivo_modalidades' THEN 'lectura_preferencias_modalidades_correcta'
   ELSE 'lectura_preferencias_otra_correcta' END;
  BEGIN
   PERFORM vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(cand,
    convert_to('{}','UTF8'),convert_to('{}','UTF8'),
    base||jsonb_build_object('campos_permitidos',campos,'motivo_ref',motivo));
   e:='00000';m:='sin error';
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS e=RETURNED_SQLSTATE,m=MESSAGE_TEXT;
  END;
  r:=r||jsonb_build_object(caso,jsonb_build_object('estado',e,'mensaje',m));
 END LOOP;
 RAISE NOTICE 'AD300 casos: %',r;
 -- Motivo cruzado o ajeno: contrato inválido antes de mirar la evidencia.
 IF EXISTS(SELECT 1 FROM unnest(ARRAY['cinco_motivo_bolsas','ocho_motivo_modalidades','cinco_motivo_ajeno']) k
   WHERE r#>>ARRAY[k,'estado'] IS DISTINCT FROM '22023'
    OR r#>>ARRAY[k,'mensaje'] IS DISTINCT FROM 'AD246: contrato de lectura inválido')
 -- Motivo que corresponde: supera el contrato y se detiene en la evidencia falsa.
 OR EXISTS(SELECT 1 FROM unnest(ARRAY['cinco_motivo_modalidades','ocho_motivo_bolsas']) k
   WHERE r#>>ARRAY[k,'estado']='00000'
    OR r#>>ARRAY[k,'mensaje'] IN ('AD246: contrato de lectura inválido','AD246: lectura sin emisor acreditado'))
 THEN RAISE EXCEPTION 'AD300: casos divergentes %',r USING ERRCODE='55000'; END IF;
END $casos$;
SELECT 'AD300 OK' AS resultado;
ROLLBACK;
