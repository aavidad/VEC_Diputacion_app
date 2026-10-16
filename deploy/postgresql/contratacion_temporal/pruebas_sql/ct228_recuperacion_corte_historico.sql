\set ON_ERROR_STOP on
-- Comprobación posterior a CT228; el ensayo de producto aporta decisiones V3 reales.
BEGIN READ ONLY;
DO $prueba$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 cuerpo text;
BEGIN
 SELECT p.prosrc INTO STRICT cuerpo FROM pg_proc p WHERE p.oid=f
  AND p.proowner=to_regrole('vec_contratacion_temporal_propietario')
  AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
   'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[];
 IF encode(sha256(convert_to(cuerpo,'UTF8')),'hex')<>'76a5bb483bca50b9f1c414da2a5510825b80f7f7568496684bc9433fee0c8c57'
  OR strpos(cuerpo,$m$f.clave_idempotencia=s->>'ClaveIdempotencia'$m$)=0
  OR strpos(cuerpo,$m$exacta.unidad_firmante_ref IS DISTINCT FROM s->>'UnidadRef'$m$)=0
  OR strpos(cuerpo,'paso_plan_firma_con_unidad_v1(')>0
  OR strpos(cuerpo,'enlace.firma_anterior_ref')=0
  OR strpos(cuerpo,'f.firma_ref=ANY(cadena_refs)')=0
  OR strpos(cuerpo,'total IS DISTINCT FROM exacta.paso_orden')=0
  OR strpos(cuerpo,'f.secuencia<=exacta.secuencia')>0
  OR strpos(cuerpo,'fila.firma_ref=exacta.firma_ref')=0
  OR strpos(cuerpo,$m$'CanonNominal',convert_from(canon,'UTF8')$m$)=0
  OR strpos(cuerpo,$m$IF s->>'Via'='certificado_vec' AND (fila.unidad_firmante_ref$m$)>0
 THEN RAISE EXCEPTION 'CT228: corte o unidad del acto exacto divergentes' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
