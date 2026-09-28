\set ON_ERROR_STOP on
-- Solo lectura del catálogo. Ejecutar después de AD3-104 y CT139 en PG18.
DO $verificar$
DECLARE f oid; g oid; def text; v_owner oid; v_sec boolean; v_config text[];
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 g:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR g IS NULL THEN RAISE EXCEPTION 'CT139/AD3-104: función ausente'; END IF;
 SELECT pg_get_functiondef(f),p.proowner,p.prosecdef,p.proconfig INTO STRICT def,v_owner,v_sec,v_config
   FROM pg_proc p WHERE p.oid=f;
 IF v_owner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR v_sec IS NOT TRUE
    OR v_config IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR strpos(def,'r.actor_ref=d->>''principal_id''')=0
    OR strpos(def,'r.perfil_ref=d->>''perfil_activo_ref''')=0
    OR strpos(def,'registrar_y_consumir_recibo_respuesta_ct_v3_atestada')=0
    OR strpos(def,'vec_bolsa')<>0 OR strpos(def,'vec_persona')<>0 THEN
    RAISE EXCEPTION 'CT139: contrato de autoridad incompatible'; END IF;
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',g,'EXECUTE')
    OR has_table_privilege('vec_contratacion_temporal_ejecutor',
       'vec_contratacion_temporal.respuesta_recibida_rrhh','SELECT') THEN
    RAISE EXCEPTION 'CT139: ACL demasiado amplia o función inaccesible'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid='vec_contratacion_temporal.respuesta_recibida_rrhh'::regclass
    AND relrowsecurity AND relforcerowsecurity) THEN
    RAISE EXCEPTION 'CT139: RLS no forzada'; END IF;
END $verificar$;
