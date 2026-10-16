\set ON_ERROR_STOP on
-- ÚNICAMENTE copia ficticia HZ39 de P. Dirección autorizó rehacer datos de prueba.
-- La app debe estar parada. Esta corrección no es un acto ADMIN ni un recibo.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $datos$
DECLARE
  persona CONSTANT text := 'per_8014599251fc9a7aff11113c502e3317';
  perfil CONSTANT text := 'prf_8d0fed2990714f97e78b0afc21b337e7';
  antigua CONSTANT text := 'can_Xx9GdqcSDoRdUVUPyGMDJlcSBvLDUXc0Or-G86S2pxc';
  nueva CONSTANT text := 'can_Vxd6XlZE9ctNzw38AUSmgQbCcRyoFKhaojDaFsEKC1M';
  referencia CONSTANT text := 'asignacion:asg_88b864b488f1335f8d34b024c0d7cd0c:v3';
  preimagen CONSTANT text := 'f5920ae751f66b48755303ddcc1d7b799820fad3eebe7d4c2f59220be0632586';
  objetivo_huella CONSTANT text := 'f3f9955e9ba0a90f220f018de5c746ec4bfb4e88c782ca3eb7d95bd507026bc9';
  actual record;
  documento_nuevo jsonb;
  huella_nueva text;
  filas integer;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: solo superusuario de la copia ficticia' USING ERRCODE = '42501';
  END IF;
  IF (SELECT count(*)
      FROM vec_contexto_actor_v1.vinculo_referencia_actual va
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref, version)
      WHERE v.persona_ref = persona AND v.tipo = 'candidato' AND v.estado = 'activo'
        AND v.vigente_desde <= statement_timestamp()
        AND statement_timestamp() < v.vigente_hasta) <> 1
     OR NOT EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual va
       JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref, version)
       WHERE v.persona_ref = persona AND v.tipo = 'candidato' AND v.estado = 'activo'
         AND v.referencia = nueva AND v.vigente_desde <= statement_timestamp()
         AND statement_timestamp() < v.vigente_hasta) THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: vínculo vigente de Antonio distinto' USING ERRCODE = '55000';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger
                 WHERE tgrelid = 'vec_autorizacion.asignacion_perfil'::regclass
                   AND tgname = 'asignacion_inmutable' AND tgenabled = 'O') THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: guarda de inmutabilidad ausente' USING ERRCODE = '55000';
  END IF;
  SELECT a.asignacion_ref, a.asignacion_id, a.version, a.perfil_activo_ref,
         a.principal_id, a.version_rol_ref, a.documento, a.huella_sha256,
         p.acto_ref, p.actualizada_por
    INTO STRICT actual
    FROM vec_autorizacion.asignacion_perfil_actual p
    JOIN vec_autorizacion.asignacion_perfil a
      USING (perfil_activo_ref, asignacion_ref)
   WHERE p.perfil_activo_ref = perfil
   FOR UPDATE OF p, a;
  IF actual.asignacion_ref IS DISTINCT FROM referencia
     OR actual.asignacion_id IS DISTINCT FROM 'asg_88b864b488f1335f8d34b024c0d7cd0c'
     OR actual.version IS DISTINCT FROM 3 OR actual.perfil_activo_ref IS DISTINCT FROM perfil
     OR actual.principal_id IS DISTINCT FROM persona
     OR actual.version_rol_ref IS DISTINCT FROM 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1'
     OR actual.acto_ref IS DISTINCT FROM 'acto:bolsa:mi-bolsa:asignacion:v1'
     OR actual.actualizada_por IS DISTINCT FROM 'identidad:desarrollo:no-autoritativa'
     OR actual.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(
       vec_autorizacion.canon_asignacion_perfil_admin_v1(actual.documento), 'UTF8'
     )), 'hex') THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: preimagen o procedencia distinta' USING ERRCODE = '55000';
  END IF;
  documento_nuevo := jsonb_set(actual.documento, '{ambitos,0,valores,0}',
                               to_jsonb(nueva), false);
  huella_nueva := encode(sha256(convert_to(
    vec_autorizacion.canon_asignacion_perfil_admin_v1(documento_nuevo), 'UTF8'
  )), 'hex');
  IF huella_nueva IS DISTINCT FROM objetivo_huella
     OR jsonb_array_length(actual.documento -> 'ambitos') IS DISTINCT FROM 1
     OR actual.documento #>> '{ambitos,0,clave}' IS DISTINCT FROM 'candidato_ref'
     OR jsonb_array_length(actual.documento #> '{ambitos,0,valores}') IS DISTINCT FROM 1 THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: documento objetivo inesperado' USING ERRCODE = '55000';
  END IF;
  IF actual.huella_sha256 = objetivo_huella
     AND actual.documento #>> '{ambitos,0,valores,0}' = nueva THEN
    RAISE NOTICE 'Mi Bolsa HZ39: dato ficticio ya corregido; sin escrituras';
    RETURN;
  END IF;
  IF actual.huella_sha256 IS DISTINCT FROM preimagen
     OR actual.documento #>> '{ambitos,0,valores,0}' IS DISTINCT FROM antigua THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: ámbito o huella anterior distinta' USING ERRCODE = '55000';
  END IF;

  -- La copia de prueba puede rehacer el dato histórico. La guarda se restaura
  -- antes del COMMIT; ningún disparador ni función del producto cambia en Git.
  EXECUTE 'ALTER TABLE vec_autorizacion.asignacion_perfil DISABLE TRIGGER asignacion_inmutable';
  UPDATE vec_autorizacion.asignacion_perfil
     SET documento = documento_nuevo, huella_sha256 = huella_nueva
   WHERE asignacion_ref = referencia AND huella_sha256 = preimagen
     AND documento = actual.documento;
  GET DIAGNOSTICS filas = ROW_COUNT;
  IF filas <> 1 THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: comparación de preimagen fallida' USING ERRCODE = '40001';
  END IF;
  EXECUTE 'ALTER TABLE vec_autorizacion.asignacion_perfil ENABLE TRIGGER asignacion_inmutable';
  IF NOT EXISTS (SELECT 1 FROM pg_trigger
                 WHERE tgrelid = 'vec_autorizacion.asignacion_perfil'::regclass
                   AND tgname = 'asignacion_inmutable' AND tgenabled = 'O')
     OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil a
                    WHERE a.asignacion_ref = referencia
                      AND a.documento = documento_nuevo
                      AND a.huella_sha256 = objetivo_huella) THEN
    RAISE EXCEPTION 'Mi Bolsa HZ39: posimagen incompleta' USING ERRCODE = '55000';
  END IF;
  RAISE NOTICE 'Mi Bolsa HZ39: dato ficticio corregido; sin acto ni recibo nuevos';
END $datos$;

SELECT a.asignacion_ref, a.version,
       a.documento #>> '{ambitos,0,valores,0}' AS candidato_ref,
       a.huella_sha256,
       a.huella_sha256 = encode(sha256(convert_to(
         vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento), 'UTF8'
       )), 'hex') AS huella_valida,
       t.tgenabled = 'O' AS guarda_inmutabilidad_activa
  FROM vec_autorizacion.asignacion_perfil_actual p
  JOIN vec_autorizacion.asignacion_perfil a
    USING (perfil_activo_ref, asignacion_ref)
  JOIN pg_trigger t ON t.tgrelid = 'vec_autorizacion.asignacion_perfil'::regclass
                   AND t.tgname = 'asignacion_inmutable'
 WHERE p.perfil_activo_ref = 'prf_8d0fed2990714f97e78b0afc21b337e7';
COMMIT;
