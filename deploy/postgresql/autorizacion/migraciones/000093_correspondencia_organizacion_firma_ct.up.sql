-- AUT93. La organización operativa del expediente conserva sus bytes y ámbitos.
-- CA45 resuelve una correspondencia vigente y bloqueada hacia la organización
-- canónica de CA25/Personal. La prueba se guarda fuera del canon V1.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000093',0));
DO $preflight$
BEGIN
 IF to_regprocedure('vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)') IS NULL
  OR to_regclass('vec_autorizacion.evidencia_competencia_firmante_ct_v1') IS NULL
  OR to_regprocedure('vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)') IS NULL
  OR to_regprocedure('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
       AND conname='evidencia_competencia_firmante_ct_v1_check1')
  OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
       AND attname='prueba_correspondencia_organizacion' AND NOT attisdropped)
  OR (SELECT count(*) FROM (VALUES
   ('acreditar_competencia_nominal_firmante_ct_v1','7b2d78a8b2b5bab99a13882b4915dd0653c01e641b5e3ccdd1e9337aea75058b'),
   ('construir_contexto_nominal_firmante_ct_v1','db34e0a9bbc7f7b69734cc477f346075bb719da745c643032e50eaa40605eec2'),
   ('recuperar_evidencia_competencia_firmante_ct_v1','74ffe6b838fd3bf067a8c91c4c97cb39c565dcd7bf62802348d0bcb8ee07b70a'),
   ('seleccionar_firmante_plan_ct_v1','04b145f7ee05c869f9d556db701ca1c5cbac6baadae7218fba8fa1b5ee2653b5'),
   ('validar_competencia_nominal_firmante_ct_v1','6529ba5edf6690ff7b70875caeba7c3354f48677ed57dc04398b913d8fc25bf9')
  ) AS esperado(nombre,huella)
  JOIN pg_proc p ON p.proname=esperado.nombre
  JOIN pg_namespace n ON n.oid=p.pronamespace AND n.nspname='vec_autorizacion'
  WHERE encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=esperado.huella)<>5
 THEN RAISE EXCEPTION 'AUT93: preimagen CA45/AUT32/AUT35/AUT56 ausente' USING ERRCODE='55000'; END IF;
END $preflight$;

-- Predicado sin lecturas para el CHECK durable y la recuperación histórica.
-- La autoridad de la prueba es CA45; este predicado sólo fija su forma y su
-- vínculo exacto con el recibo CA25 y el instante de creación.
CREATE FUNCTION vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(
 p_organizacion_operativa text, p_organizacion_destino jsonb, p_prueba jsonb, p_creada_en timestamptz)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT
SET search_path = pg_catalog, pg_temp AS $f$
BEGIN
 RETURN jsonb_typeof(p_organizacion_destino)='object'
  AND jsonb_typeof(p_prueba)='object'
  AND (SELECT count(*) FROM jsonb_object_keys(p_prueba))=14
  AND p_prueba ?& ARRAY['esquema','correspondencia_ref','version',
   'organizacion_operativa_ref','organizacion_canonica_ref','organizacion_version',
   'procedencia_ref','procedencia_version','procedencia_huella_sha256',
   'procedencia_autoridad','estado','vigente_desde','vigente_hasta','huella_sha256']
  AND p_prueba->>'esquema'='vec.organizacion.correspondencia.v1'
  AND p_prueba->>'organizacion_operativa_ref'=p_organizacion_operativa
  AND p_prueba->>'organizacion_canonica_ref'=p_organizacion_destino->>'organizacion_ref'
  AND p_organizacion_operativa<>p_organizacion_destino->>'organizacion_ref'
  AND p_prueba->>'organizacion_version'=p_organizacion_destino->>'organizacion_version'
  AND p_prueba->>'procedencia_ref'=p_organizacion_destino->>'organizacion_procedencia_ref'
  AND p_prueba->>'procedencia_version'=p_organizacion_destino->>'organizacion_procedencia_version'
  AND p_prueba->>'procedencia_huella_sha256'=p_organizacion_destino->>'organizacion_procedencia_sha256'
  AND p_prueba->>'procedencia_autoridad' ~ '^[a-z][a-z0-9_]{2,127}$'
  AND p_prueba->>'estado'='vigente'
  AND p_prueba->>'correspondencia_ref' ~ '^correspondencia-org:[0-9a-f-]{36}$'
  AND p_prueba->>'version' ~ '^[1-9][0-9]{0,15}$'
  AND p_prueba->>'organizacion_version' ~ '^[1-9][0-9]{0,15}$'
  AND p_prueba->>'procedencia_version' ~ '^[1-9][0-9]{0,15}$'
  AND p_prueba->>'procedencia_huella_sha256' ~ '^[0-9a-f]{64}$'
  AND p_prueba->>'huella_sha256' ~ '^[0-9a-f]{64}$'
  AND p_prueba->>'huella_sha256'=encode(sha256(
   convert_to((p_prueba - 'huella_sha256')::text,'UTF8')),'hex')
  AND p_creada_en IS NOT NULL AND isfinite(p_creada_en)
  AND p_creada_en >= (p_prueba->>'vigente_desde')::timestamptz
  AND p_creada_en < (p_prueba->>'vigente_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN RETURN false;
END $f$;
ALTER FUNCTION vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(text,jsonb,jsonb,timestamptz)
 OWNER TO vec_autorizacion_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(text,jsonb,jsonb,timestamptz) FROM PUBLIC;

-- Sólo AUT puede pedir la correspondencia. Un nulo es válido exclusivamente
-- para la identidad directa; una respuesta inválida siempre lanza 42501.
CREATE FUNCTION vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
 p_organizacion_operativa text, p_organizacion_destino jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp SET row_security = on SET lock_timeout = '2s'
SET "TimeZone" = 'UTC' AS $f$
DECLARE r jsonb; prueba jsonb; canonica text; ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR pg_is_in_recovery() OR current_setting('TimeZone')<>'UTC'
  OR vec_autorizacion.texto_positivo_valido(p_organizacion_operativa,512) IS NOT TRUE
  OR jsonb_typeof(p_organizacion_destino) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'aut93_correspondencia_no_admitida' USING ERRCODE='42501'; END IF;
 r:=vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(p_organizacion_operativa);
 IF jsonb_typeof(r) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(r))<>3
  OR r->>'organizacion_operativa_ref' IS DISTINCT FROM p_organizacion_operativa
  OR r->>'organizacion_canonica_ref' IS DISTINCT FROM p_organizacion_destino->>'organizacion_ref'
 THEN RAISE EXCEPTION 'aut93_correspondencia_no_admitida' USING ERRCODE='42501'; END IF;
 canonica:=r->>'organizacion_canonica_ref'; prueba:=r->'prueba';
 IF canonica=p_organizacion_operativa THEN
  IF prueba IS DISTINCT FROM 'null'::jsonb THEN
   RAISE EXCEPTION 'aut93_correspondencia_no_admitida' USING ERRCODE='42501'; END IF;
  RETURN NULL;
 END IF;
 ahora:=clock_timestamp();
 IF vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(
    p_organizacion_operativa,p_organizacion_destino,prueba,ahora) IS NOT TRUE THEN
  RAISE EXCEPTION 'aut93_correspondencia_no_admitida' USING ERRCODE='42501'; END IF;
 RETURN prueba;
END $f$;
ALTER FUNCTION vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(text,jsonb)
 OWNER TO vec_autorizacion_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(text,jsonb) FROM PUBLIC;

ALTER TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1
 ADD COLUMN prueba_correspondencia_organizacion jsonb;
ALTER TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1
 DROP CONSTRAINT evidencia_competencia_firmante_ct_v1_check1;
ALTER TABLE vec_autorizacion.evidencia_competencia_firmante_ct_v1
 ADD CONSTRAINT evidencia_competencia_firmante_ct_v1_check1 CHECK (
  ((organizacion_destino->>'organizacion_ref'=organizacion_ref) IS TRUE
   AND prueba_correspondencia_organizacion IS NULL)
  OR
  ((organizacion_destino->>'organizacion_ref'<>organizacion_ref) IS TRUE
   AND vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(
    organizacion_ref,organizacion_destino,prueba_correspondencia_organizacion,creada_en) IS TRUE)
 );

CREATE OR REPLACE FUNCTION vec_autorizacion.seleccionar_firmante_plan_ct_v1(p_certificado_der_sha256 text, p_cargo_ref text, p_rol_id text, p_accion text, p_tipo_recurso text, p_finalidad text, p_organizacion_ref text, p_unidad_ref text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE ca jsonb;en jsonb;a record;ahora timestamptz(6);prueba_org jsonb;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR pg_catalog.current_setting('role')<>'none'
  OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_catalog.pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
  OR (p_certificado_der_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_rol_id ~ '^[a-z][a-z0-9_]{2,63}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'seleccion_firmante_denegada' USING ERRCODE='42501'; END IF;
 -- Persona del certificado: CA25 revalida vínculo, cuenta, persona y
 -- organización de destino; un certificado retirado o ajeno no selecciona.
 ca:=vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(p_certificado_der_sha256);
 IF ca IS NULL OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM p_certificado_der_sha256
 THEN RAISE EXCEPTION 'seleccion_firmante_certificado_no_admitido' USING ERRCODE='42501'; END IF;
 prueba_org:=vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
  p_organizacion_ref,ca->'organizacion_destino');
 -- Enlace vigente y único del cargo por tipo (Personal38 valida el resto).
 en:=vec_personal.localizar_enlace_cargo_ct_v1(p_cargo_ref,ca->>'persona_ref',p_accion,p_tipo_recurso,p_finalidad,
  ca#>>'{organizacion_destino,organizacion_ref}',p_unidad_ref);
 -- Única asignación activa y vigente de esa persona con el rol del paso, rol
 -- publicado y control habilitado. Más de una es ambigua: se deniega.
 ahora:=pg_catalog.clock_timestamp();
 SELECT x.asignacion_ref,x.version,x.huella_sha256,x.perfil_activo_ref,x.documento,
  r.version_rol_ref,r.huella_sha256 AS rol_huella,v.revision AS control_revision,v.huella_sha256 AS control_huella
  INTO STRICT a
  FROM vec_autorizacion.asignacion_perfil x
  JOIN vec_autorizacion.asignacion_perfil_actual p ON p.perfil_activo_ref=x.perfil_activo_ref AND p.asignacion_ref=x.asignacion_ref
  JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=x.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol_actual c ON c.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=c.version_rol_ref AND v.revision=c.revision
  WHERE x.principal_id=ca->>'persona_ref' AND r.rol_id=p_rol_id
   AND x.documento->>'estado'='activa' AND r.documento->>'estado'='publicada' AND v.estado='habilitada'
   AND ahora>=(x.documento->>'vigente_desde')::timestamptz AND ahora<(x.documento->>'vigente_hasta')::timestamptz
   -- Ámbitos exactos, como AUT32 y AD206: organización y unidad, un valor
   -- cada una e iguales a las del paso; ninguna otra dimensión.
   AND pg_catalog.jsonb_typeof(x.documento->'ambitos')='array' AND pg_catalog.jsonb_array_length(x.documento->'ambitos')=2
   AND x.documento->'ambitos' @> pg_catalog.jsonb_build_array(
    pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(p_organizacion_ref)),
    pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(p_unidad_ref)))
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(x.documento->'ambitos') e
    WHERE pg_catalog.jsonb_typeof(e->'valores') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(e->'valores')<>1)
  FOR SHARE OF x,p,r,c,v;
 RETURN pg_catalog.jsonb_build_object('esquema','vec.autorizacion.seleccion-firmante-plan.ct.v1',
  'persona_ref',ca->>'persona_ref','perfil_activo_ref',a.perfil_activo_ref,'rol_id',p_rol_id,
  'cuenta_ref',ca#>>'{vinculo_certificado,cuenta_ref}',
  'vinculo_certificado',pg_catalog.jsonb_build_object('referencia',ca#>>'{vinculo_certificado,referencia}',
   'version',ca#>'{vinculo_certificado,version}','huella_sha256',ca#>>'{vinculo_certificado,huella_sha256}'),
  'cargo_ref',p_cargo_ref,'enlace_ejercicio_ref',en#>>'{enlace,referencia}',
  'asignacion',pg_catalog.jsonb_build_object('referencia',a.asignacion_ref,'version',a.version,'huella_sha256',a.huella_sha256,
   'vigente_desde',a.documento->'vigente_desde','vigente_hasta',a.documento->'vigente_hasta'),
  'rol',pg_catalog.jsonb_build_object('referencia',a.version_rol_ref,'huella_sha256',a.rol_huella),
  'control_rol',pg_catalog.jsonb_build_object('referencia',a.version_rol_ref,'revision',a.control_revision,'huella_sha256',a.control_huella));
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'seleccion_firmante_ausente_o_ambigua' USING ERRCODE='42501';
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(descriptor_ct jsonb, relacion_ct jsonb, consumo_v3 jsonb)
 RETURNS bytea
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE sel jsonb; rec jsonb; ca jsonb; per jsonb; c jsonb; identidad jsonb;
 competencia jsonb; personal jsonb; relacion jsonb; canon text; bytes bytea;
 asignacion record; rol record; fecha timestamptz(6); now_utc timestamptz(6); prueba_org jsonb;
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
  OR current_setting('transaction_read_only') <> 'off'
  OR pg_is_in_recovery() OR current_setting('TimeZone') <> 'UTC'
 THEN RAISE EXCEPTION 'aut35_transaccion_no_admitida' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(descriptor_ct) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(descriptor_ct)) <> 11
  OR (descriptor_ct ?& ARRAY['esquema','certificado_der_sha256','seleccion',
     'recurso','accion','finalidad','motivo','circuito','paso_ref',
     'paso_orden','fecha_historica']) IS NOT TRUE
  OR descriptor_ct->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.constructor-ct.v1'
  OR (descriptor_ct->>'certificado_der_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR jsonb_typeof(descriptor_ct->'seleccion') IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(descriptor_ct->'seleccion')) <> 5
  OR (descriptor_ct->'seleccion' ?& ARRAY['perfil_esperado_ref','perfil_activo_ref',
     'rol_id','cargo_ref','enlace_ejercicio_ref']) IS NOT TRUE
  OR jsonb_typeof(descriptor_ct->'recurso') IS DISTINCT FROM 'object'
  OR jsonb_typeof(relacion_ct) IS DISTINCT FROM 'object'
  OR jsonb_typeof(consumo_v3) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'aut35_descriptor_no_admitido' USING ERRCODE='42501'; END IF;
 sel:=descriptor_ct->'seleccion'; rec:=descriptor_ct->'recurso';
 -- La misma gramática tipada que la salida evita coerciones, campos opacos y
 -- cualquier serialización JSONB textual antes de leer las fuentes.
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(rec,'recurso');
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(descriptor_ct->'motivo','motivo');
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(descriptor_ct->'circuito','ref');
 IF vec_autorizacion.texto_positivo_valido(sel->>'perfil_esperado_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'perfil_activo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'rol_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'cargo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'enlace_ejercicio_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'accion',256) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'finalidad',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'paso_ref',512) IS NOT TRUE
  OR vec_autorizacion.instante_utc_microsegundo_valido(descriptor_ct->>'fecha_historica') IS NOT TRUE
  OR rec->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'documento_ref'
  OR rec->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR rec->>'organizacion_ref' IS DISTINCT FROM relacion_ct->>'organizacion_ref'
  OR rec->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_ref'
  OR rec->>'expediente_ref' IS DISTINCT FROM relacion_ct->>'expediente_ref'
 THEN RAISE EXCEPTION 'aut35_descriptor_no_admitido' USING ERRCODE='42501'; END IF;
 fecha:=(descriptor_ct->>'fecha_historica')::timestamptz;
 now_utc:=clock_timestamp();
 IF fecha > now_utc OR relacion_ct->>'esquema' IS DISTINCT FROM
    'vec.contratacion-temporal.relacion-unidad-expediente.v1'
  OR relacion_ct->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_ref_esperada'
  OR relacion_ct->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_snapshot_solicitado_ref'
  OR relacion_ct->>'tipo_evento_origen' IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR (relacion_ct->>'asignacion_confirmada_en')::timestamptz > fecha
 THEN RAISE EXCEPTION 'aut35_relacion_no_admitida' USING ERRCODE='42501'; END IF;
 -- AD167 acredita antes de cualquier lectura nominal que consumo y auditoría
 -- fueron insertados en esta misma transacción para el efecto CT.
 PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(consumo_v3);
 -- CA25 devuelve fuentes revalidadas y bloqueadas, incluyendo la organización
 -- acreditada. Ningún campo de identidad del descriptor reemplaza a CA25.
 ca:=vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(
   descriptor_ct->>'certificado_der_sha256');
 IF ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-ct.v2'
  OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM descriptor_ct->>'certificado_der_sha256'
 THEN RAISE EXCEPTION 'aut35_identidad_no_admitida' USING ERRCODE='42501'; END IF;
 prueba_org:=vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
  rec->>'organizacion_ref',ca->'organizacion_destino');
 identidad:=jsonb_build_object(
  'certificado_der_sha256',ca->>'certificado_der_sha256',
  'persona_ref',ca->>'persona_ref',
  'persona',ca->'persona',
  'cuenta',ca->'cuenta',
  'vinculo_cuenta_persona',jsonb_build_object(
   'referencia',ca#>>'{vinculo_cuenta_persona,referencia}',
   'version',ca#>'{vinculo_cuenta_persona,version}',
   'huella_sha256',ca#>>'{vinculo_cuenta_persona,huella_sha256}'),
  'cuenta_persona_cuenta_ref',ca#>>'{vinculo_cuenta_persona,cuenta_ref}',
  'cuenta_persona_persona_ref',ca#>>'{vinculo_cuenta_persona,persona_ref}',
  'vinculo_certificado',jsonb_build_object(
   'referencia',ca#>>'{vinculo_certificado,referencia}',
   'version',ca#>'{vinculo_certificado,version}',
   'huella_sha256',ca#>>'{vinculo_certificado,huella_sha256}'),
  'vinculo_cuenta_ref',ca#>>'{vinculo_certificado,cuenta_ref}',
  'vinculo_persona_ref',ca#>>'{vinculo_certificado,persona_ref}',
  'vinculo_der_sha256',ca#>>'{vinculo_certificado,certificado_der_sha256}');
 -- Personal29 resuelve sólo el selector explícito y contrasta el ejercicio
 -- nominal; su fachada bloquea la fuente propietaria hasta COMMIT.
 per:=vec_personal.resolver_fuente_cargo_ocupante_ct_v1(
   sel->>'cargo_ref',sel->>'enlace_ejercicio_ref',ca->>'persona_ref',
   ca#>>'{organizacion_destino,organizacion_ref}',rec->>'unidad_ref');
 IF per->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR per->>'persona_ejerciente_ref' IS DISTINCT FROM ca->>'persona_ref'
  OR per->>'organizacion_ref' IS DISTINCT FROM ca#>>'{organizacion_destino,organizacion_ref}'
  OR per->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR per#>>'{cargo,referencia}' IS DISTINCT FROM sel->>'cargo_ref'
  OR per#>>'{enlace_ejerciente,referencia}' IS DISTINCT FROM sel->>'enlace_ejercicio_ref'
  OR per->>'accion_ref' IS DISTINCT FROM descriptor_ct->>'accion'
  OR per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'tipo_recurso'
  OR per->>'finalidad_ref' IS DISTINCT FROM descriptor_ct->>'finalidad'
 THEN RAISE EXCEPTION 'aut35_personal_no_admitido' USING ERRCODE='42501'; END IF;
 personal:=jsonb_build_object('cargo',per->'cargo','enlace_ocupante',per->'enlace_ocupante',
  'ocupante_persona_ref',per->>'ocupante_persona_ref',
  'cargo_ref_enlace',per->>'cargo_ref_enlace',
  'cargo_vigente_desde',per->'cargo_vigente_desde',
  'cargo_vigente_hasta',per->'cargo_vigente_hasta',
  'enlace_vigente_desde',per->'enlace_vigente_desde',
  'enlace_vigente_hasta',per->'enlace_vigente_hasta',
  'delegacion',per->'delegacion');
 -- La asignación y el rol se eligen por perfil del plan y se leen de la
 -- autoridad AUT. No se deriva un rol de cargo, título o certificado.
 SELECT a.asignacion_ref,a.version,a.huella_sha256,a.principal_id,a.perfil_activo_ref,
   a.version_rol_ref,a.documento
 INTO STRICT asignacion FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 WHERE p.perfil_activo_ref=sel->>'perfil_activo_ref' FOR SHARE OF p,a;
 SELECT r.version_rol_ref,r.version,r.rol_id,r.huella_sha256,r.documento,
   v.revision,v.estado,v.huella_sha256 AS control_huella
 INTO STRICT rol FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.control_vigencia_version_rol_actual x ON x.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=x.version_rol_ref AND v.revision=x.revision
 WHERE r.version_rol_ref=asignacion.version_rol_ref FOR SHARE OF r,x,v;
 IF asignacion.principal_id IS DISTINCT FROM ca->>'persona_ref'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM sel->>'perfil_activo_ref'
  OR rol.rol_id IS DISTINCT FROM sel->>'rol_id'
  OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
  OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
  OR rol.estado IS DISTINCT FROM 'habilitada'
 THEN RAISE EXCEPTION 'aut35_competencia_no_admitida' USING ERRCODE='42501'; END IF;
 competencia:=jsonb_build_object(
  'asignacion',jsonb_build_object('referencia',asignacion.asignacion_ref,
   'version',asignacion.version,'huella_sha256',asignacion.huella_sha256),
  'rol',jsonb_build_object('referencia',rol.version_rol_ref,'version',rol.version,
   'huella_sha256',rol.huella_sha256),
  'rol_id',rol.rol_id,
  'control_rol',jsonb_build_object('referencia',rol.version_rol_ref,
   'version',rol.revision,'huella_sha256',rol.control_huella),
  'persona_ref',asignacion.principal_id,
  'perfil_esperado_ref',sel->>'perfil_esperado_ref',
  'perfil_activo_ref',asignacion.perfil_activo_ref,
  'modulo_id',rec->>'modulo_id','tipo_recurso',rec->>'tipo_recurso',
  'recurso_ref',rec->>'recurso_autorizable_ref',
  'ambito_organizacion_ref',rec->>'organizacion_ref',
  'ambito_unidad_ref',rec->>'unidad_ref',
  'asignacion_rol_ref',asignacion.version_rol_ref,
  'control_rol_ref',rol.version_rol_ref,
  'vigente_desde',asignacion.documento->'vigente_desde',
  'vigente_hasta',asignacion.documento->'vigente_hasta');
 relacion:=jsonb_build_object('expediente_ref',relacion_ct->>'expediente_ref',
  'unidad_ref',relacion_ct->>'unidad_ref',
  'origen_ref',relacion_ct->>'operacion_origen_ref',
  'origen_version',relacion_ct->'version_origen_vinculo',
  'prueba_snapshot_sha256',relacion_ct->>'prueba_snapshot_origen_huella_sha256',
  'evento_ref',relacion_ct->>'evento_asignacion_ref',
  'evento_huella_sha256',relacion_ct->>'evento_payload_huella_sha256',
  'confirmada_en',relacion_ct->'asignacion_confirmada_en');
 c:=jsonb_build_object('esquema','vec.competencia-firmante.historica.v1',
  'identidad',identidad,'competencia',competencia,'personal',personal,
  'recurso',rec,'relacion_ct',relacion,'accion',descriptor_ct->'accion',
  'finalidad',descriptor_ct->'finalidad','motivo',descriptor_ct->'motivo',
  'circuito',descriptor_ct->'circuito','paso_ref',descriptor_ct->'paso_ref',
  'paso_orden',descriptor_ct->'paso_orden',
  'fecha_historica',descriptor_ct->'fecha_historica');
 canon:=vec_autorizacion.canon_json_competencia_firmante_ct_v1(c,'raiz');
 bytes:=convert_to(canon,'UTF8');
 IF octet_length(bytes) NOT BETWEEN 512 AND 32768 THEN
  RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
 -- AUT32 repite las lecturas y comprueba consumo V3, ACL, política y vigencia
 -- en esta transacción. Construir no inserta evidencia ni sustituye acreditar.
 PERFORM vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   bytes,relacion_ct,consumo_v3);
 RETURN bytes;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'aut35_fuente_no_disponible' USING ERRCODE='42501';
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(p_contexto_nominal bytea, p_relacion_ct jsonb, p_consumo_v3 jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c jsonb; ident jsonb; comp jsonb; per jsonb; rec jsonb; ct jsonb;
 ca jsonb; org_ca jsonb; prueba_org jsonb; fuente_personal jsonb; v3 jsonb; a record; rol record;
 catalogo record;
 concedida boolean; amb_org boolean; amb_unidad boolean;
 instante timestamptz(6); fecha_historica timestamptz(6); efecto text;
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
  OR current_setting('transaction_read_only') <> 'off'
  OR pg_is_in_recovery() OR current_setting('TimeZone') <> 'UTC' THEN
  RAISE EXCEPTION 'aut32_transaccion_no_admitida' USING ERRCODE = '42501';
 END IF;
 IF p_contexto_nominal IS NULL OR octet_length(p_contexto_nominal) NOT BETWEEN 512 AND 32768
  OR jsonb_typeof(p_relacion_ct) IS DISTINCT FROM 'object'
  OR jsonb_typeof(p_consumo_v3) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'aut32_contexto_no_admitido' USING ERRCODE = '42501';
 END IF;
 c := convert_from(p_contexto_nominal,'UTF8')::jsonb;
 ident := c->'identidad'; comp := c->'competencia'; per := c->'personal';
 rec := c->'recurso'; ct := c->'relacion_ct';
 IF c->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
  OR jsonb_typeof(ident) IS DISTINCT FROM 'object'
  OR jsonb_typeof(comp) IS DISTINCT FROM 'object'
  OR jsonb_typeof(per) IS DISTINCT FROM 'object'
  OR jsonb_typeof(rec) IS DISTINCT FROM 'object'
  OR jsonb_typeof(ct) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(c)) <> 13
  OR c->>'accion' IS NULL OR c->>'finalidad' IS NULL
  OR vec_autorizacion.texto_positivo_valido(c->>'accion',256) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(c->>'finalidad',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'rol_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'perfil_esperado_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'perfil_activo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'modulo_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'tipo_recurso',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(comp->>'recurso_ref',512) IS NOT TRUE
  OR vec_autorizacion.instante_utc_microsegundo_valido(c->>'fecha_historica') IS NOT TRUE
  OR rec->>'organizacion_ref' IS NULL OR rec->>'unidad_ref' IS NULL
  OR rec->>'expediente_ref' IS NULL OR rec->>'documento_ref' IS NULL
  OR rec->>'recurso_autorizable_ref' IS NULL
  OR (rec->>'recurso_contexto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (rec->>'pdf_raiz_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (rec->>'pdf_firmado_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR rec->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'documento_ref'
  OR rec#>>'{original,huella_sha256}' IS DISTINCT FROM rec->>'pdf_raiz_sha256'
  OR rec#>>'{firmado,huella_sha256}' IS DISTINCT FROM rec->>'pdf_firmado_sha256'
  OR ((rec->>'numero_firmas')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((rec->>'numero_firmas')::numeric=1 AND rec->'entrada_revision' IS DISTINCT FROM 'null'::jsonb)
  OR ((rec->>'numero_firmas')::numeric>1 AND jsonb_typeof(rec->'entrada_revision') IS DISTINCT FROM 'object')
  OR c#>>'{motivo,catalogo_id}' IS NULL
  OR ((c#>>'{motivo,catalogo_version}')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (c#>>'{motivo,catalogo_huella_sha256}' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR c#>>'{motivo,entrada_clave}' IS NULL
  OR ((c#>>'{circuito,version}')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (c#>>'{circuito,huella_sha256}' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR c->>'paso_ref' IS NULL
  OR ((c->>'paso_orden')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE THEN
  RAISE EXCEPTION 'aut32_contexto_no_admitido' USING ERRCODE = '42501';
 END IF;
 -- CT174 procede del propietario CT y conserva el puntero bajo FOR SHARE.
 IF p_relacion_ct->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.relacion-unidad-expediente.v1'
  OR ct->>'expediente_ref' IS DISTINCT FROM rec->>'expediente_ref'
  OR ct->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR p_relacion_ct->>'organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR p_relacion_ct->>'expediente_ref' IS DISTINCT FROM ct->>'expediente_ref'
  OR p_relacion_ct->>'unidad_ref' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'unidad_ref_esperada' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'unidad_snapshot_solicitado_ref' IS DISTINCT FROM ct->>'unidad_ref'
  OR p_relacion_ct->>'operacion_origen_ref' IS DISTINCT FROM ct->>'origen_ref'
  OR (p_relacion_ct->>'version_origen_vinculo')::numeric IS DISTINCT FROM (ct->>'origen_version')::numeric
  OR p_relacion_ct->>'prueba_snapshot_origen_huella_sha256' IS DISTINCT FROM ct->>'prueba_snapshot_sha256'
  OR p_relacion_ct->>'evento_asignacion_ref' IS DISTINCT FROM ct->>'evento_ref'
  OR p_relacion_ct->>'evento_payload_huella_sha256' IS DISTINCT FROM ct->>'evento_huella_sha256'
  OR (p_relacion_ct->>'asignacion_confirmada_en')::timestamptz
     IS DISTINCT FROM (ct->>'confirmada_en')::timestamptz THEN
  RAISE EXCEPTION 'aut32_relacion_ct_no_admitida' USING ERRCODE = '42501';
 END IF;
 fecha_historica := (c->>'fecha_historica')::timestamptz;
 instante := clock_timestamp();
 IF fecha_historica IS NULL OR fecha_historica > instante
  OR p_relacion_ct->>'tipo_evento_origen' IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR (ct->>'confirmada_en')::timestamptz > fecha_historica THEN
  RAISE EXCEPTION 'aut32_relacion_ct_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- La correlación del consumo identifica el efecto CT. Su auditoría pertenece
 -- al núcleo V3; AUT sólo conserva sus referencias, no crea otra cadena.
 efecto := p_consumo_v3->>'efecto_ref';
 IF vec_autorizacion.texto_positivo_valido(efecto,512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_consumo_v3->>'decision_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_consumo_v3->>'auditoria_ref',512) IS NOT TRUE
  OR (p_consumo_v3->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_consumo_v3->>'huella_efecto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR p_consumo_v3->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb THEN
  RAISE EXCEPTION 'aut32_consumo_v3_no_admitido' USING ERRCODE = '42501';
 END IF;
 v3 := vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo_v3);
 IF v3->>'decision_ref' IS DISTINCT FROM p_consumo_v3->>'decision_ref'
  OR v3->>'efecto_ref' IS DISTINCT FROM efecto
  OR v3->>'huella_efecto_sha256' IS DISTINCT FROM p_consumo_v3->>'huella_efecto_sha256'
  OR v3->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo_v3->>'consumo_huella_sha256'
  OR v3->>'auditoria_ref' IS DISTINCT FROM p_consumo_v3->>'auditoria_ref' THEN
  RAISE EXCEPTION 'aut32_consumo_v3_no_admitido' USING ERRCODE = '42501';
 END IF;
 ca := vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(ident->>'certificado_der_sha256');
 org_ca := ca->'organizacion_destino';
 IF ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-ct.v2'
  OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM ident->>'certificado_der_sha256'
  OR ca->>'persona_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR ca->'persona' IS DISTINCT FROM ident->'persona'
  OR ca->'cuenta' IS DISTINCT FROM ident->'cuenta'
  OR ca#>>'{vinculo_cuenta_persona,referencia}' IS DISTINCT FROM ident#>>'{vinculo_cuenta_persona,referencia}'
  OR (ca#>>'{vinculo_cuenta_persona,version}')::numeric IS DISTINCT FROM (ident#>>'{vinculo_cuenta_persona,version}')::numeric
  OR ca#>>'{vinculo_cuenta_persona,huella_sha256}' IS DISTINCT FROM ident#>>'{vinculo_cuenta_persona,huella_sha256}'
  OR ca#>>'{vinculo_cuenta_persona,cuenta_ref}' IS DISTINCT FROM ident->>'cuenta_persona_cuenta_ref'
  OR ca#>>'{vinculo_cuenta_persona,persona_ref}' IS DISTINCT FROM ident->>'cuenta_persona_persona_ref'
  OR ca#>>'{persona,referencia}' IS DISTINCT FROM ident->>'persona_ref'
  OR ca->'vinculo_certificado'->>'referencia' IS DISTINCT FROM ident#>>'{vinculo_certificado,referencia}'
  OR (ca#>>'{vinculo_certificado,version}')::numeric IS DISTINCT FROM (ident#>>'{vinculo_certificado,version}')::numeric
  OR ca#>>'{vinculo_certificado,huella_sha256}' IS DISTINCT FROM ident#>>'{vinculo_certificado,huella_sha256}'
  OR ca#>>'{vinculo_certificado,cuenta_ref}' IS DISTINCT FROM ident#>>'{cuenta,referencia}'
  OR ca#>>'{vinculo_certificado,cuenta_ref}' IS DISTINCT FROM ident->>'vinculo_cuenta_ref'
  OR ca#>>'{vinculo_certificado,persona_ref}' IS DISTINCT FROM ident->>'persona_ref'
  OR ca#>>'{vinculo_certificado,persona_ref}' IS DISTINCT FROM ident->>'vinculo_persona_ref'
  OR ca#>>'{vinculo_certificado,certificado_der_sha256}' IS DISTINCT FROM ident->>'certificado_der_sha256'
  OR ca#>>'{vinculo_certificado,certificado_der_sha256}' IS DISTINCT FROM ident->>'vinculo_der_sha256'
  OR ca#>>'{vinculo_cuenta_persona,cuenta_ref}' IS DISTINCT FROM ident#>>'{cuenta,referencia}'
  OR ca#>>'{vinculo_cuenta_persona,persona_ref}' IS DISTINCT FROM ident->>'persona_ref' THEN
  RAISE EXCEPTION 'aut32_identidad_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- La organización acreditada por CA4 se conserva fuera del canon V1: el
 -- documento CA25 original liga estos metadatos a la huella de su vínculo.
 IF jsonb_typeof(org_ca) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(org_ca)) <> 9
  OR vec_autorizacion.texto_positivo_valido(org_ca->>'organizacion_procedencia_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(org_ca->>'vinculo_corporativo_ref',512) IS NOT TRUE
  OR ((org_ca->>'organizacion_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((org_ca->>'organizacion_procedencia_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR ((org_ca->>'vinculo_corporativo_version')::numeric BETWEEN 1 AND 9007199254740991) IS NOT TRUE
  OR (org_ca->>'organizacion_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (org_ca->>'organizacion_procedencia_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (org_ca->>'vinculo_corporativo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'aut32_organizacion_no_admitida' USING ERRCODE = '42501';
 END IF;
 prueba_org:=vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
  rec->>'organizacion_ref',org_ca);
 -- Personal38 recibe únicamente un DTO de consulta. El canon firmado V1 y
 -- todos sus selectores CT siguen íntegros en p_contexto_nominal.
 fuente_personal := vec_personal.leer_revalidar_cargo_ocupante_ct_v1(convert_to(
  jsonb_build_object(
   'esquema',c->'esquema','fecha_historica',c->'fecha_historica',
   'personal',per,'identidad',ident,'accion',c->'accion','finalidad',c->'finalidad',
   'recurso',jsonb_build_object(
    'organizacion_ref',org_ca->>'organizacion_ref',
    'unidad_ref',rec->'unidad_ref','expediente_ref',rec->'expediente_ref',
    'documento_ref',rec->'documento_ref',
    'recurso_autorizable_ref',rec->'recurso_autorizable_ref',
    'modulo_id',rec->'modulo_id','tipo_recurso',rec->'tipo_recurso',
    'recurso_contexto_sha256',rec->'recurso_contexto_sha256',
    'original',rec->'original','pdf_raiz_sha256',rec->'pdf_raiz_sha256',
    'firmado',rec->'firmado','pdf_firmado_sha256',rec->'pdf_firmado_sha256',
    'numero_firmas',rec->'numero_firmas','entrada_revision',rec->'entrada_revision'))::text,'UTF8'));
 IF fuente_personal->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR fuente_personal->'cargo'->>'referencia' IS DISTINCT FROM per#>>'{cargo,referencia}'
  OR (fuente_personal#>>'{cargo,version}')::numeric IS DISTINCT FROM (per#>>'{cargo,version}')::numeric
  OR fuente_personal#>>'{cargo,huella_sha256}' IS DISTINCT FROM per#>>'{cargo,huella_sha256}'
  OR fuente_personal->'enlace_ocupante'->>'referencia' IS DISTINCT FROM per#>>'{enlace_ocupante,referencia}'
  OR (fuente_personal#>>'{enlace_ocupante,version}')::numeric IS DISTINCT FROM (per#>>'{enlace_ocupante,version}')::numeric
  OR fuente_personal#>>'{enlace_ocupante,huella_sha256}' IS DISTINCT FROM per#>>'{enlace_ocupante,huella_sha256}'
  OR fuente_personal->>'ocupante_persona_ref' IS DISTINCT FROM per->>'ocupante_persona_ref'
  OR fuente_personal->>'cargo_ref_enlace' IS DISTINCT FROM per->>'cargo_ref_enlace'
  OR fuente_personal->>'organizacion_ref' IS DISTINCT FROM org_ca->>'organizacion_ref'
  OR fuente_personal->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR fuente_personal->>'persona_ejerciente_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR vec_autorizacion.texto_positivo_valido(fuente_personal->>'procedencia_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(fuente_personal->>'recibo_ref',512) IS NOT TRUE
  OR (per->'delegacion' = 'null'::jsonb AND fuente_personal->>'tipo_ejercicio' IS DISTINCT FROM 'titular')
  OR (per->'delegacion' <> 'null'::jsonb AND
      (fuente_personal->>'tipo_ejercicio' IN
       ('delegacion_competencia','delegacion_firma','suplencia')) IS NOT TRUE)
  OR (fuente_personal->>'cargo_vigente_desde')::timestamptz IS DISTINCT FROM (per->>'cargo_vigente_desde')::timestamptz
  OR (fuente_personal->>'cargo_vigente_hasta')::timestamptz IS DISTINCT FROM (per->>'cargo_vigente_hasta')::timestamptz
  OR (fuente_personal->>'enlace_vigente_desde')::timestamptz IS DISTINCT FROM (per->>'enlace_vigente_desde')::timestamptz
  OR (fuente_personal->>'enlace_vigente_hasta')::timestamptz IS DISTINCT FROM (per->>'enlace_vigente_hasta')::timestamptz
  OR (fuente_personal->'delegacion' IS DISTINCT FROM 'null'::jsonb
      AND per->'delegacion' IS DISTINCT FROM 'null'::jsonb
      AND (fuente_personal#>'{delegacion,acto}' IS DISTINCT FROM per#>'{delegacion,acto}'
       OR fuente_personal#>>'{delegacion,delegante_persona_ref}' IS DISTINCT FROM per#>>'{delegacion,delegante_persona_ref}'
       OR fuente_personal#>>'{delegacion,delegado_persona_ref}' IS DISTINCT FROM per#>>'{delegacion,delegado_persona_ref}'
       OR fuente_personal#>>'{delegacion,cargo_ref}' IS DISTINCT FROM per#>>'{delegacion,cargo_ref}'
       OR (fuente_personal#>>'{delegacion,vigente_desde}')::timestamptz IS DISTINCT FROM (per#>>'{delegacion,vigente_desde}')::timestamptz
       OR (fuente_personal#>>'{delegacion,vigente_hasta}')::timestamptz IS DISTINCT FROM (per#>>'{delegacion,vigente_hasta}')::timestamptz))
  OR (fuente_personal->'delegacion' IS DISTINCT FROM 'null'::jsonb)
     IS DISTINCT FROM (per->'delegacion' IS DISTINCT FROM 'null'::jsonb) THEN
  RAISE EXCEPTION 'aut32_cargo_no_admitido' USING ERRCODE = '42501';
 END IF;
 IF per->'delegacion' = 'null'::jsonb AND per->>'ocupante_persona_ref' IS DISTINCT FROM ident->>'persona_ref'
  OR per->'delegacion' <> 'null'::jsonb AND (
    per#>>'{delegacion,delegante_persona_ref}' IS DISTINCT FROM per->>'ocupante_persona_ref'
    OR per#>>'{delegacion,delegado_persona_ref}' IS DISTINCT FROM ident->>'persona_ref') THEN
  RAISE EXCEPTION 'aut32_cargo_no_admitido' USING ERRCODE = '42501';
 END IF;
 -- Protección del puntero y de las versiones centrales. La asignación propia
 -- pertenece a la persona del certificado y a un único perfil activo.
 SELECT a.asignacion_ref,a.version,a.huella_sha256,a.principal_id,a.perfil_activo_ref,
   a.version_rol_ref,a.documento
 INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 WHERE p.perfil_activo_ref=comp->>'perfil_activo_ref' FOR SHARE OF p,a;
 SELECT r.version_rol_ref,r.version,r.rol_id,r.huella_sha256,r.documento,
   v.revision,v.estado,v.huella_sha256 AS control_huella,v.documento AS control_documento
 INTO STRICT rol FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.control_vigencia_version_rol_actual x ON x.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=x.version_rol_ref AND v.revision=x.revision
 WHERE r.version_rol_ref=a.version_rol_ref FOR SHARE OF r,x,v;
 -- Toda mutación del puntero de políticas exige actualizar esta fila en la
 -- misma transacción. FOR SHARE impide su COMMIT mientras se decide el efecto.
 SELECT revision,huella_sha256 INTO STRICT catalogo
  FROM vec_autorizacion.control_catalogo_politicas WHERE control_id FOR SHARE;
 instante := clock_timestamp();
 IF a.principal_id IS DISTINCT FROM ident->>'persona_ref'
  OR a.principal_id IS DISTINCT FROM comp->>'persona_ref'
  OR a.asignacion_ref IS DISTINCT FROM comp#>>'{asignacion,referencia}'
  OR a.version IS DISTINCT FROM (comp#>>'{asignacion,version}')::bigint
  OR a.huella_sha256 IS DISTINCT FROM comp#>>'{asignacion,huella_sha256}'
  OR a.perfil_activo_ref IS DISTINCT FROM comp->>'perfil_activo_ref'
  OR a.version_rol_ref IS DISTINCT FROM comp->>'asignacion_rol_ref'
  OR rol.version_rol_ref IS DISTINCT FROM comp#>>'{rol,referencia}'
  OR rol.version_rol_ref IS DISTINCT FROM comp->>'control_rol_ref'
  OR rol.version_rol_ref IS DISTINCT FROM comp#>>'{control_rol,referencia}'
  OR rol.version IS DISTINCT FROM (comp#>>'{rol,version}')::bigint
  OR rol.rol_id IS DISTINCT FROM comp->>'rol_id'
  OR rol.huella_sha256 IS DISTINCT FROM comp#>>'{rol,huella_sha256}'
  OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
  OR rol.documento->>'rol_id' IS DISTINCT FROM rol.rol_id
  OR rol.documento->>'version' IS DISTINCT FROM rol.version::text
  OR (rol.documento->>'publicada_en')::timestamptz > instante
  OR rol.estado IS DISTINCT FROM 'habilitada'
  OR (rol.control_documento->>'actualizado_en')::timestamptz > instante
  OR rol.revision IS DISTINCT FROM (comp#>>'{control_rol,version}')::numeric
  OR rol.control_huella IS DISTINCT FROM comp#>>'{control_rol,huella_sha256}'
  OR a.documento->>'estado' IS DISTINCT FROM 'activa'
  OR a.documento->>'vigente_desde' IS DISTINCT FROM comp->>'vigente_desde'
  OR a.documento->>'vigente_hasta' IS DISTINCT FROM comp->>'vigente_hasta'
  OR instante < (a.documento->>'vigente_desde')::timestamptz
  OR instante >= (a.documento->>'vigente_hasta')::timestamptz
  OR fecha_historica < (a.documento->>'vigente_desde')::timestamptz
  OR fecha_historica >= (a.documento->>'vigente_hasta')::timestamptz
  OR instante < (ca#>>'{vinculo_certificado,vigente_desde}')::timestamptz
  OR instante >= (ca#>>'{vinculo_certificado,vigente_hasta}')::timestamptz
  OR fecha_historica < (ca#>>'{vinculo_certificado,vigente_desde}')::timestamptz
  OR fecha_historica >= (ca#>>'{vinculo_certificado,vigente_hasta}')::timestamptz
  OR instante < (per->>'cargo_vigente_desde')::timestamptz
  OR instante >= (per->>'cargo_vigente_hasta')::timestamptz
  OR fecha_historica < (per->>'cargo_vigente_desde')::timestamptz
  OR fecha_historica >= (per->>'cargo_vigente_hasta')::timestamptz
  OR instante < (per->>'enlace_vigente_desde')::timestamptz
  OR instante >= (per->>'enlace_vigente_hasta')::timestamptz
  OR instante >= (v3->>'decision_valida_hasta')::timestamptz
  OR fecha_historica < (per->>'enlace_vigente_desde')::timestamptz
  OR fecha_historica >= (per->>'enlace_vigente_hasta')::timestamptz
  OR (per->'delegacion' IS DISTINCT FROM 'null'::jsonb AND
    (instante < (per#>>'{delegacion,vigente_desde}')::timestamptz
     OR instante >= (per#>>'{delegacion,vigente_hasta}')::timestamptz))
  OR comp->>'modulo_id' IS DISTINCT FROM rec->>'modulo_id'
  OR comp->>'tipo_recurso' IS DISTINCT FROM rec->>'tipo_recurso'
  OR comp->>'recurso_ref' IS DISTINCT FROM rec->>'recurso_autorizable_ref'
  OR comp->>'ambito_organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR comp->>'ambito_unidad_ref' IS DISTINCT FROM rec->>'unidad_ref' THEN
  RAISE EXCEPTION 'aut32_concesion_no_admitida' USING ERRCODE = '42501';
 END IF;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(rol.documento->'concesiones') q
   WHERE q->>'accion'=c->>'accion' AND q->>'modulo_id'=comp->>'modulo_id'
    AND q->>'tipo_recurso'=comp->>'tipo_recurso'
    AND q->'finalidades' ? (c->>'finalidad')
    AND coalesce(q->'obligaciones','[]'::jsonb)='[]'::jsonb
    AND coalesce(q->'campos_permitidos','[]'::jsonb)='[]'::jsonb) INTO concedida;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave'='organizacion_ref' AND q->'valores' ? (rec->>'organizacion_ref')) INTO amb_org;
 SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave'='unidad_ref' AND q->'valores' ? (rec->>'unidad_ref')) INTO amb_unidad;
 IF concedida IS NOT TRUE OR amb_org IS NOT TRUE OR amb_unidad IS NOT TRUE
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') q
   WHERE q->>'clave' NOT IN ('organizacion_ref','unidad_ref','expediente_ref','documento_ref')
    OR (q->>'clave'='expediente_ref' AND
       (q->'valores' ? (rec->>'expediente_ref')) IS NOT TRUE)
    OR (q->>'clave'='documento_ref' AND
       (q->'valores' ? (rec->>'documento_ref')) IS NOT TRUE)) THEN
  RAISE EXCEPTION 'aut32_concesion_no_admitida' USING ERRCODE = '42501';
 END IF;
 -- No existe un evaluador ABAC offline autorizado para el firmante. El
 -- catálogo protege el conjunto actual hasta COMMIT. Una política aplicable
 -- publicada, incluso de inicio futuro, deja esta vía cerrada.
 IF catalogo.huella_sha256 !~ '^[0-9a-f]{64}$'
  OR EXISTS(SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual x
   JOIN vec_autorizacion.politica_restrictiva p USING(politica_id,politica_ref)
   WHERE jsonb_typeof(p.documento->'acciones') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p.documento->'modulos') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p.documento->'tipos_recurso') IS DISTINCT FROM 'array'
    OR (p.documento->>'estado' IN ('publicada','retirada')) IS NOT TRUE
    OR (p.documento->>'efecto' IN ('restringir','denegar')) IS NOT TRUE
    OR vec_autorizacion.instante_utc_microsegundo_valido(p.documento->>'vigente_desde') IS NOT TRUE
    OR vec_autorizacion.instante_utc_microsegundo_valido(p.documento->>'vigente_hasta') IS NOT TRUE
    OR (p.documento->>'vigente_hasta')::timestamptz <=
       (p.documento->>'vigente_desde')::timestamptz)
  OR EXISTS(SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual x
   JOIN vec_autorizacion.politica_restrictiva p USING(politica_id,politica_ref)
   WHERE p.documento->>'estado'='publicada'
    AND instante < (p.documento->>'vigente_hasta')::timestamptz
    AND ((p.documento->'acciones' ? (c->>'accion')) OR (p.documento->'acciones' ? '*'))
    AND ((p.documento->'modulos' ? (comp->>'modulo_id')) OR (p.documento->'modulos' ? '*'))
    AND ((p.documento->'tipos_recurso' ? (comp->>'tipo_recurso')) OR (p.documento->'tipos_recurso' ? '*'))) THEN
  RAISE EXCEPTION 'aut32_politica_no_evaluable' USING ERRCODE='42501';
 END IF;
 RETURN c;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(p_contexto_nominal bytea, p_relacion_ct jsonb, p_consumo_v3 jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c jsonb; h text; ref text; previo vec_autorizacion.evidencia_competencia_firmante_ct_v1%ROWTYPE;
 v3 jsonb; org_ca jsonb; prueba_org jsonb; catalogo record;
BEGIN
 c := vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   p_contexto_nominal,p_relacion_ct,p_consumo_v3);
 org_ca := (vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(
   c#>>'{identidad,certificado_der_sha256}'))->'organizacion_destino';
 IF jsonb_typeof(org_ca) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'aut32_organizacion_no_admitida' USING ERRCODE='42501'; END IF;
 prueba_org:=vec_autorizacion.comprobar_correspondencia_organizacion_firma_ct_v1(
  c#>>'{recurso,organizacion_ref}',org_ca);
 SELECT revision,huella_sha256 INTO STRICT catalogo
  FROM vec_autorizacion.control_catalogo_politicas WHERE control_id FOR SHARE;
 h := encode(sha256(p_contexto_nominal),'hex');
 v3 := vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo_v3);
 ref := 'evidencia:competencia-firmante-ct:' ||
  encode(sha256(convert_to(p_consumo_v3->>'efecto_ref','UTF8') || p_contexto_nominal),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
   'vec_autorizacion:competencia_ct:' || (p_consumo_v3->>'efecto_ref'),0));
 SELECT * INTO previo FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1
  WHERE efecto_ref=p_consumo_v3->>'efecto_ref' FOR SHARE;
 IF FOUND THEN
  -- La vigencia actual ya se ha comprobado. El enlace histórico permanece.
  IF previo.organizacion_ref IS DISTINCT FROM c#>>'{recurso,organizacion_ref}'
   OR previo.unidad_ref IS DISTINCT FROM c#>>'{recurso,unidad_ref}'
   OR previo.expediente_ref IS DISTINCT FROM c#>>'{recurso,expediente_ref}'
   OR previo.documento_ref IS DISTINCT FROM c#>>'{recurso,documento_ref}'
   OR previo.persona_ref IS DISTINCT FROM c#>>'{identidad,persona_ref}'
   OR previo.certificado_der_sha256 IS DISTINCT FROM c#>>'{identidad,certificado_der_sha256}'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'recurso' IS DISTINCT FROM c->'recurso'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'accion' IS DISTINCT FROM c->>'accion'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'finalidad' IS DISTINCT FROM c->>'finalidad'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'motivo' IS DISTINCT FROM c->'motivo'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->'circuito' IS DISTINCT FROM c->'circuito'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'paso_ref' IS DISTINCT FROM c->>'paso_ref'
   OR (convert_from(previo.canonico,'UTF8')::jsonb)->>'paso_orden' IS DISTINCT FROM c->>'paso_orden' THEN
   RAISE EXCEPTION 'aut32_efecto_divergente' USING ERRCODE = '42501';
  END IF;
  RETURN jsonb_build_object('evidencia_ref',previo.evidencia_ref,'esquema',previo.esquema,
   'version',1,'huella_sha256',previo.huella_sha256,
   'organizacion_destino_huella_sha256',previo.organizacion_destino_huella_sha256,
   'catalogo_politicas_revision',previo.catalogo_politicas_revision,
   'catalogo_politicas_huella_sha256',previo.catalogo_politicas_huella_sha256,
   'recuperada',true);
 END IF;
 INSERT INTO vec_autorizacion.evidencia_competencia_firmante_ct_v1 (
  evidencia_ref,efecto_ref,esquema,canonico,huella_sha256,organizacion_ref,unidad_ref,
  expediente_ref,documento_ref,persona_ref,certificado_der_sha256,
  organizacion_destino,organizacion_destino_huella_sha256,prueba_correspondencia_organizacion,
  catalogo_politicas_revision,catalogo_politicas_huella_sha256,
  consumo_decision_ref,consumo_huella_sha256,auditoria_consumo_ref,
  auditoria_consumo_huella_sha256,registrador_principal_ref,registrador_perfil_ref)
 VALUES (ref,p_consumo_v3->>'efecto_ref','vec.competencia-firmante.historica.v1',
  p_contexto_nominal,h,c#>>'{recurso,organizacion_ref}',c#>>'{recurso,unidad_ref}',
  c#>>'{recurso,expediente_ref}',c#>>'{recurso,documento_ref}',c#>>'{identidad,persona_ref}',
  c#>>'{identidad,certificado_der_sha256}',org_ca,
  encode(sha256(convert_to(org_ca::text,'UTF8')),'hex'),prueba_org,
  catalogo.revision,catalogo.huella_sha256,p_consumo_v3->>'decision_ref',
  p_consumo_v3->>'consumo_huella_sha256',p_consumo_v3->>'auditoria_ref',
  v3->>'auditoria_huella_sha256',v3->>'registrador_principal_ref',
  v3->>'registrador_perfil_ref');
 RETURN jsonb_build_object('evidencia_ref',ref,'esquema','vec.competencia-firmante.historica.v1',
  'version',1,'huella_sha256',h,
  'organizacion_destino_huella_sha256',encode(sha256(convert_to(org_ca::text,'UTF8')),'hex'),
  'catalogo_politicas_revision',catalogo.revision,
  'catalogo_politicas_huella_sha256',catalogo.huella_sha256,
  'recuperada',false);
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(p_evidencia_ref text, p_huella_sha256 text, p_contexto_actual bytea, p_relacion_ct jsonb, p_consumo_v3 jsonb)
 RETURNS bytea
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE c jsonb; historico jsonb; fuente_ca jsonb;
 original vec_autorizacion.evidencia_competencia_firmante_ct_v1%ROWTYPE;
BEGIN
 c := vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   p_contexto_actual,p_relacion_ct,p_consumo_v3);
 SELECT * INTO STRICT original FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1
  WHERE evidencia_ref=p_evidencia_ref AND huella_sha256=p_huella_sha256 FOR SHARE;
 IF original.efecto_ref IS DISTINCT FROM p_consumo_v3->>'efecto_ref'
  OR original.organizacion_ref IS DISTINCT FROM c#>>'{recurso,organizacion_ref}'
  OR original.unidad_ref IS DISTINCT FROM c#>>'{recurso,unidad_ref}'
  OR original.expediente_ref IS DISTINCT FROM c#>>'{recurso,expediente_ref}'
  OR original.documento_ref IS DISTINCT FROM c#>>'{recurso,documento_ref}'
  OR original.persona_ref IS DISTINCT FROM c#>>'{identidad,persona_ref}'
  OR original.certificado_der_sha256 IS DISTINCT FROM c#>>'{identidad,certificado_der_sha256}'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'recurso' IS DISTINCT FROM c->'recurso'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'accion' IS DISTINCT FROM c->>'accion'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'finalidad' IS DISTINCT FROM c->>'finalidad'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'motivo' IS DISTINCT FROM c->'motivo'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->'circuito' IS DISTINCT FROM c->'circuito'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'paso_ref' IS DISTINCT FROM c->>'paso_ref'
  OR (convert_from(original.canonico,'UTF8')::jsonb)->>'paso_orden' IS DISTINCT FROM c->>'paso_orden'
  OR original.huella_sha256 IS DISTINCT FROM encode(sha256(original.canonico),'hex') THEN
  RAISE EXCEPTION 'aut32_evidencia_no_disponible' USING ERRCODE = '42501';
 END IF;
 historico := convert_from(original.canonico,'UTF8')::jsonb;
 fuente_ca := vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(
  historico#>>'{identidad,vinculo_certificado,referencia}',
  (historico#>>'{identidad,vinculo_certificado,version}')::numeric,
  historico#>>'{identidad,vinculo_certificado,huella_sha256}');
 IF fuente_ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-historia.v2'
  OR fuente_ca->>'cuenta_ref' IS DISTINCT FROM historico#>>'{identidad,cuenta,referencia}'
  OR fuente_ca->>'cuenta_version' IS DISTINCT FROM historico#>>'{identidad,cuenta,version}'
  OR fuente_ca->>'cuenta_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,cuenta,huella_sha256}'
  OR fuente_ca->>'persona_ref' IS DISTINCT FROM historico#>>'{identidad,persona,referencia}'
  OR fuente_ca->>'persona_version' IS DISTINCT FROM historico#>>'{identidad,persona,version}'
  OR fuente_ca->>'persona_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,persona,huella_sha256}'
  OR fuente_ca->>'vinculo_cuenta_persona_ref' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,referencia}'
  OR fuente_ca->>'vinculo_cuenta_persona_version' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,version}'
  OR fuente_ca->>'vinculo_cuenta_persona_huella_sha256' IS DISTINCT FROM historico#>>'{identidad,vinculo_cuenta_persona,huella_sha256}'
  OR fuente_ca->'organizacion_destino' IS DISTINCT FROM original.organizacion_destino
  OR (original.prueba_correspondencia_organizacion IS NULL AND
      original.organizacion_ref IS DISTINCT FROM original.organizacion_destino->>'organizacion_ref')
  OR (original.prueba_correspondencia_organizacion IS NOT NULL AND
      vec_autorizacion.prueba_correspondencia_organizacion_ct_valida_v1(
       original.organizacion_ref,original.organizacion_destino,
       original.prueba_correspondencia_organizacion,original.creada_en) IS NOT TRUE)
  OR original.organizacion_destino_huella_sha256 IS DISTINCT FROM
   encode(sha256(convert_to(original.organizacion_destino::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'aut32_evidencia_no_disponible' USING ERRCODE = '42501';
 END IF;
 RETURN original.canonico;
END $function$;

COMMIT;
