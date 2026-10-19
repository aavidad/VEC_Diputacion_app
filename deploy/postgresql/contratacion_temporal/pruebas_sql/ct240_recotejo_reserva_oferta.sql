\set ON_ERROR_STOP on
-- Tras CT240 (F1-10) en copia desechable. Cuerpo, permisos y orden de los
-- pasos de la reserva de oferta; no finge V3. El cotejo con datos lo cubre
-- deploy/postgresql/bolsa_llamamientos/pruebas_sql/b140_recotejo_reserva.sql.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $guardas$ DECLARE f regprocedure;x record;c text;consumo int;cerrojo int;cotejo int;bloqueo int;
BEGIN
 f:='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT p.prosecdef,p.proacl::text AS acl,pg_get_functiondef(p.oid) AS def INTO STRICT x
 FROM pg_proc p WHERE p.oid=f;
 c:=x.def;
 IF x.prosecdef IS NOT TRUE
 OR encode(sha256(convert_to(c,'UTF8')),'hex') IS DISTINCT FROM
   '6c7777b7dc0232503cf834cb1373a576ab3fa995071ce4b86f09ab464e0e3a0a'
 OR x.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'CT240: ACL o cuerpo divergente'; END IF;
 -- Autoridad consumida, después el orden entre reservas, el cotejo en Bolsa y
 -- por último el bloqueo del expediente.
 consumo:=position('consumir_etapa_seleccion_oferta_ct220(' IN c);
 cerrojo:=position('reserva-oferta:' IN c);
 cotejo:=position('cotejar_comprobacion_reserva_ct_v1(' IN c);
 bloqueo:=position('FOR UPDATE OF a' IN c);
 IF NOT (consumo>0 AND consumo<cerrojo AND cerrojo<cotejo AND cotejo<bloqueo)
 THEN RAISE EXCEPTION 'CT240: orden de pasos divergente % % % %',consumo,cerrojo,cotejo,bloqueo; END IF;
END $guardas$;
SELECT 'ct240_recotejo_reserva_oferta: 0 fallos' AS resultado;
ROLLBACK;
