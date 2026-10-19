\set ON_ERROR_STOP on
-- Tras CT240 (F1-10) en copia desechable. Cuerpo, permisos y orden de los
-- pasos de la reserva de oferta; no finge V3. El cotejo con datos lo cubre
-- deploy/postgresql/bolsa_llamamientos/pruebas_sql/b140_recotejo_reserva.sql.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $guardas$ DECLARE f regprocedure;x record;c text;consumo int;cerrojo int;cotejo int;bloqueo int;
 singular int;par int;
BEGIN
 f:='vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT p.prosecdef,p.proacl::text AS acl,pg_get_functiondef(p.oid) AS def INTO STRICT x
 FROM pg_proc p WHERE p.oid=f;
 c:=x.def;
 IF x.prosecdef IS NOT TRUE
 -- CT241 (F1-19) añade después la guarda del nombramiento derivado.
 OR encode(sha256(convert_to(c,'UTF8')),'hex') NOT IN (
   '45f84d4328a28c909966560149e08fd51ad13c73e737657a889e07dc3d3dd735',
   '6a8e5f1feff9051e960a8db3866669569c5c551eca009d7f6889c87f88e383e8')
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
 -- CT235 (F1-11) se conserva: la reserva nueva exige el par modalidad y causa
 -- detrás de las guardas de CT220.
 singular:=position('expediente ya tiene selección singular' IN c);
 par:=position('CT235: la nueva reserva exige modalidad y causa' IN c);
 IF NOT (bloqueo<singular AND singular<par)
 OR position('CT235: modalidad o causa distinta del análisis vigente' IN c)<par
 THEN RAISE EXCEPTION 'CT240: guarda de CT235 perdida o adelantada % %',singular,par; END IF;
END $guardas$;
SELECT 'ct240_recotejo_reserva_oferta: 0 fallos' AS resultado;
ROLLBACK;
