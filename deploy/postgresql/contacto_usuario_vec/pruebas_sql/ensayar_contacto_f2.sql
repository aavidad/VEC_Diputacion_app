\set ON_ERROR_STOP on
-- Evidencia posterior a COMMIT en una base F2 desechable. No añade historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $test$
DECLARE f oid; fuente text;
BEGIN
    f:=to_regprocedure('vec_autorizacion_atestada_v3.contacto_version_validar_material_v1(text,bytea,bytea,bytea,bytea,bytea)');
    IF f IS NULL THEN RAISE EXCEPTION 'F2: validador AD3 ausente'; END IF;
    SELECT prosrc INTO fuente FROM pg_proc WHERE oid=f;
    IF strpos(fuente,'vec.contacto_usuario.version_propia.v1')=0
       OR strpos(fuente,'vec.contacto_usuario.version_llamamiento.v1')=0
       OR strpos(fuente,'gestion_contacto_propio')=0
       OR strpos(fuente,'envio_llamamiento')=0
       OR strpos(fuente,'persona_ref')=0 THEN
        RAISE EXCEPTION 'F2: contratos de selector cruzados';
    END IF;
    IF NOT has_function_privilege('vec_contacto_usuario_writer',
        'vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR NOT has_function_privilege('vec_contacto_usuario_reader',
        'vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_reader',
        'vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_writer',
        'vec_contacto_usuario_v1.consultar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_table_privilege('vec_contacto_usuario_reader','vec_contacto_usuario_v1.actual','SELECT')
       OR has_table_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.versiones','SELECT') THEN
        RAISE EXCEPTION 'F2: ACL cruzada';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.versiones)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.actual)
       OR EXISTS (SELECT 1 FROM vec_contacto_usuario_v1.outbox) THEN
        RAISE EXCEPTION 'F2: ensayo mezclado con historia';
    END IF;
END $test$;
ROLLBACK;
