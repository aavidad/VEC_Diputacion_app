\set ON_ERROR_STOP on
-- Comprueba instalación y orden de guardas; no simula un consumo V3.
BEGIN READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f oid; cuerpo text; primero integer; despues integer;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_selector_documental_aceptacion_nativa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT262: función, dueño o ACL divergente' USING ERRCODE='55000'; END IF;
 cuerpo:=pg_get_functiondef(f);
 primero:=position('consultar_detalle_rrhh_atestado_v1' IN cuerpo);
 despues:=position('SELECT * INTO STRICT propuesta' IN cuerpo);
 IF primero=0 OR despues<=primero
  OR position('actual.agregado_json#>''{actuaciones,6}''' IN cuerpo)=0
  OR position('propuesta.recibo_aceptacion_ref' IN cuerpo)=0
  OR position('propuesta.aceptacion_bolsa_json' IN cuerpo)=0
  OR position('RETURN QUERY SELECT lectura.contenido_canonico' IN cuerpo)=0
 THEN RAISE EXCEPTION 'CT262: proyección sin ligadura documental' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
