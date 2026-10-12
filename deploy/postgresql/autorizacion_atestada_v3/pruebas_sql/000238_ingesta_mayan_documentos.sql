\set ON_ERROR_STOP on
-- Comprobación estructural exacta después de AD238 en clon PostgreSQL 18.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE core text; fachada text; a text;
BEGIN
 SELECT pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
  INTO core;
 SELECT pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
  INTO fachada;
 IF core IS NULL OR fachada IS NULL OR strpos(core,'documentos.original_firmable.reservar')=0
   OR strpos(fachada,'documentos.original_firmable.confirmar')=0
 THEN RAISE EXCEPTION 'AD238: se perdió la autoridad documental previa'; END IF;
 FOREACH a IN ARRAY ARRAY[
  'documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva',
  'documentos.ingesta_mayan.abandonar_recurso',
  'documentos.ingesta_mayan.preparar','documentos.ingesta_mayan.despachar',
  'documentos.ingesta_mayan.resolver','documentos.ingesta_mayan.conciliar',
  'documentos.ingesta_mayan.consultar'] LOOP
  IF strpos(core,a)=0 OR strpos(fachada,a)=0 THEN
   RAISE EXCEPTION 'AD238: acción ausente %',a; END IF;
 END LOOP;
 IF has_function_privilege('public',
  'vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'AD238: PUBLIC puede consumir'; END IF;
END $test$;
ROLLBACK;
