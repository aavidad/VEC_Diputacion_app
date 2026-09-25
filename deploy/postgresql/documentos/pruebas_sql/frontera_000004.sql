\set ON_ERROR_STOP on
-- Documentos-4, como superusuario de la base desechable: vector de huella del
-- efecto (mismo valor que ports.HuellaEfectoV3 en Go), ACL del registro de
-- denegaciones y separación auditor/ejecutor.
CREATE ROLE vec_documentos_auditor_ensayo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_documentos_auditor TO vec_documentos_auditor_ensayo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
DO $checks$
BEGIN
 IF (SELECT encode(sha256(convert_to('{"ambitos":{},"atributos":{"preimagen_sha256":"'||encode(sha256('abc'::bytea),'hex')||'"}}','UTF8')),'hex'))
    <> 'c2535d333933e853c1cd0ecb0ed11250173705932d435f6f4310bf3cf4f1d614'
 THEN RAISE EXCEPTION 'vector de huella de efecto distinto del de Go'; END IF;
 SET LOCAL ROLE vec_documentos_propietario;
 IF vec_documentos.huella_efecto_v1('abc'::bytea) <> 'c2535d333933e853c1cd0ecb0ed11250173705932d435f6f4310bf3cf4f1d614'
 THEN RAISE EXCEPTION 'huella_efecto_v1 no coincide con Go'; END IF;
 -- Mismos vectores que TestHuellaEfectoV3VectoresCompartidosConSQL
 -- (internal/vec/documentos/ports/efecto_v3_test.go).
 IF vec_documentos.huella_efecto_v1(convert_to('{"accion":"documentos.expediente.listar","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","cursor":"","limite":1}','UTF8'))
    <> 'c0fd576c1fb94f16eceb0da654d6b5bf53013383f6bfd5ad3730730fd05b440a'
    OR vec_documentos.huella_efecto_v1(convert_to('{"motivo":"año"}','UTF8'))
    <> 'eed395327dbdfbda3586ec4ceaeae44086c90f7dceed7bd78c90eb7aa5898178'
 THEN RAISE EXCEPTION 'huella_efecto_v1 no coincide con los vectores de Go'; END IF;
 RESET ROLE;
 IF has_function_privilege('vec_documentos_ensayo','vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_documentos_auditor_ensayo','vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')
    OR has_function_privilege('vec_documentos_auditor_ensayo','vec_documentos.huella_efecto_v1(bytea)','EXECUTE')
    OR has_function_privilege('vec_documentos_ensayo','vec_documentos.huella_efecto_v1(bytea)','EXECUTE')
    OR has_function_privilege('vec_documentos_auditor_ensayo','vec_documentos.listar_expediente_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_documentos_auditor_ensayo','vec_documentos.denegacion_frontera','SELECT')
    OR has_table_privilege('vec_documentos_auditor_ensayo','vec_documentos.denegacion_frontera','INSERT')
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid='vec_documentos.denegacion_frontera'::regclass AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'ACL de la frontera documental incompatible'; END IF;
END $checks$;
