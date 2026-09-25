\set ON_ERROR_STOP on
-- Sólo para el arnés desechable de Dietas 000009. Completa, después de
-- preparar_entorno_stub_ad3.sql, las fachadas AD3 y Personal que exigen las
-- precondiciones de 000006-000008. No son migraciones ni consumidores
-- instalables: cualquier invocación falla. El ensayo de 000009 solo usa
-- validar_documento_v2, que no consume autorización.
DO $stubs$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['documento','documento_consulta','circuito','bandeja','prelectura','revisor_documento'] LOOP
  EXECUTE format($f$CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_%s_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $c$ BEGIN RAISE EXCEPTION 'stub AD3 no invocable' USING ERRCODE='55000'; END $c$$f$, nombre);
  EXECUTE format('GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_%s_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_propietario', nombre);
 END LOOP;
END $stubs$;
CREATE FUNCTION vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog
AS $$ BEGIN RAISE EXCEPTION 'stub Personal no invocable' USING ERRCODE='55000'; END $$;
GRANT EXECUTE ON FUNCTION vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date) TO vec_dietas_propietario;
