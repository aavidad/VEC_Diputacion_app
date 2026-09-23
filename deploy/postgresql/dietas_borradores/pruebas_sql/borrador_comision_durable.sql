\set ON_ERROR_STOP on
-- Arnés focal: requiere snapshot desechable con AD3-39 publicado y una relación
-- Personal sintética. No se ejecuta sobre bases vivas.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
DO $$ BEGIN
 IF to_regprocedure('vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_schema_privilege('vec_dietas_ejecutor','vec_dietas','USAGE')
    OR has_table_privilege('vec_dietas_ejecutor','vec_dietas.borrador_comision','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_dietas_propietario','vec_personal.relacion_empleado_dietas','SELECT')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)','EXECUTE') THEN
   RAISE EXCEPTION 'ACL/fachada Dietas incompatible' USING ERRCODE='55000';
 END IF;
END $$;
-- Vector compartido con encoding/json de Go: dos rutas y caracteres que
-- jsonb::text no escapa como Go. Comprueba bytes y SHA, no sólo formato.
DO $$
DECLARE canon text := '{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","empleado_ref":"emp_bbbbbbbbbbbbbbbbbbbbbb","relacion_ref":"rel_cccccccccccccccccccccc","unidad_ref":"unidad:prueba","relacion_version":3,"vigente_desde":"2026-09-01","vigente_hasta":"","procedencia_acto_ref":"acto:prueba","fuente_ref":"fuente:prueba","fuente_version":4,"clave_idempotencia":"clave_idempotente_0001","fecha_inicio":"2026-09-21","fecha_fin":"2026-09-21","motivo":"Motivo á\u003c\u003e\u0026\u2028\u2029","codigos_ruta":["GR:001","GR:002"]}';
BEGIN
 IF vec_dietas.cadena_json_go_v1('Motivo á<>&' || chr(8232) || chr(8233)) IS DISTINCT FROM '"Motivo á\u003c\u003e\u0026\u2028\u2029"'
    OR octet_length(canon) <> octet_length(convert_to(canon,'UTF8'))
    OR encode(sha256(convert_to(canon,'UTF8')),'hex') IS DISTINCT FROM '39b42af3a7a70c8f2b6a3fc613767ce5fd42b077757c7e6cf2132fd425895eef' THEN
   RAISE EXCEPTION 'vector canónico Go/SQL Dietas divergente' USING ERRCODE='55000';
 END IF;
END $$;
-- Invariantes que no dependen del material criptográfico del arnés: una
-- auditoría de consulta puede conservar recurso inexistente/página sin fingir
-- comisión, pero un recibo/historia sólo referencia comisión durable.
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_dietas.auditoria_borrador_comision'::regclass AND attname='comision_ref' AND NOT attnotnull)
    OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_dietas.auditoria_borrador_comision'::regclass AND attname='recurso_ref' AND attnotnull)
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_dietas.recibo_borrador_comision'::regclass AND tgname='inmutable')
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_dietas.historia_borrador_comision'::regclass AND tgname='inmutable') THEN
   RAISE EXCEPTION 'historia/auditoría Dietas incompatible' USING ERRCODE='55000';
 END IF;
END $$;
ROLLBACK;
