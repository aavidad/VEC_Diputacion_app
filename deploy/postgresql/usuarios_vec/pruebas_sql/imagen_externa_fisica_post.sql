\set ON_ERROR_STOP on
-- Ejecutar como DBA después de Documentos 000010 y Usuarios 000011.
-- Aserciones de estructura, ausencia de mezcla y privilegios efectivos.
DO $prueba$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_documentos.imagen_personal WHERE superficie='externa_personal')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_actual WHERE superficie='externa_personal')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_historia WHERE superficie='externa_personal')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo WHERE superficie='externa_personal')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto)
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto_externa)
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_actual_externa a
      WHERE a.foto_ref IS NOT NULL AND NOT EXISTS(
       SELECT 1 FROM vec_documentos.imagen_personal_externa d
       WHERE d.imagen_ref=a.foto_ref AND d.persona_ref=a.persona_ref AND d.estado='activa'))
    OR EXISTS(SELECT 1 FROM vec_documentos.imagen_personal_externa d
      WHERE d.estado='activa' AND encode(sha256(d.contenido),'hex')<>d.huella_sha256)
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.oid IN (
       'vec_documentos.imagen_personal_externa'::regclass,
       'vec_documentos.imagen_personal_historia_externa'::regclass,
       'vec_usuarios.imagen_actual_externa'::regclass,
       'vec_usuarios.imagen_historia_externa'::regclass,
       'vec_usuarios.imagen_recibo_externa'::regclass,
       'vec_usuarios.imagen_contexto_externa'::regclass)
       AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
    OR has_table_privilege('vec_usuarios_ejecutor_externo','vec_usuarios.imagen_actual_externa','SELECT')
    OR has_table_privilege('vec_usuarios_ejecutor_externo','vec_documentos.imagen_personal_externa','SELECT')
    OR has_table_privilege('vec_usuarios_propietario','vec_documentos.imagen_personal_externa','SELECT')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',
      'vec_usuarios.guardar_imagen_propia_v1_externa(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno',
      'vec_documentos.custodiar_imagen_personal_v1_externa(text,bytea,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'imagen externa: separación, custodia o ACL incompatibles' USING ERRCODE='55000'; END IF;
END $prueba$;
