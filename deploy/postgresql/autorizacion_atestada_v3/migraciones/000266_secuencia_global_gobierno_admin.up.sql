\set ON_ERROR_STOP on
-- AD266: ADMIN prepara desde la secuencia global de confianza.
-- Literal de AUT235 instalado en la copia principal; misma raíz CT y preimagen.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)')),'UTF8')),'hex') INTO actual;
 IF actual IS DISTINCT FROM 'b041f59e474191196ef911c73f6c2baf0ca5d80e25bc1b1587f1d4189ad917d7' THEN
  RAISE EXCEPTION 'AD266: PARO clave=instantanea_gobierno_admin_lectura_v1(integer) actual=% esperado=b041f59e474191196ef911c73f6c2baf0ca5d80e25bc1b1587f1d4189ad917d7',coalesce(actual,'ausente') USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(p_conjunto integer)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET quote_all_identifiers TO 'off'
AS $function$
DECLARE envuelta regprocedure;huella text;pre text;doc jsonb;
BEGIN
 IF p_conjunto IS NULL OR p_conjunto NOT BETWEEN 0 AND 1000000
 THEN RAISE EXCEPTION 'AD235: PARO clave=conjunto actual=fuera_de_rango esperado=0_a_1000000' USING ERRCODE='22023'; END IF;
 IF p_conjunto=0 THEN
  envuelta:=to_regprocedure('vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()');
  huella:='117916cefb1a0f8891f9b6ec46f15b2f0231489dd34b036eb15fd7e28f97c222';
 ELSE
  envuelta:=to_regprocedure('vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)');
  huella:='924fb5a69b691404247420f885abaebd2de659711ab115b21960bfba58fe1366';
 END IF;
 IF envuelta IS NULL OR encode(sha256(convert_to(pg_get_functiondef(envuelta),'UTF8')),'hex') IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AD235: PARO clave=preimagen_envuelta actual=distinta esperado=AD188_AD198' USING ERRCODE='55000'; END IF;
 IF p_conjunto=0 THEN
  pre:=encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex');
 ELSE
  pre:=encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(p_conjunto)::text,'UTF8')),'hex');
 END IF;
 SELECT jsonb_build_object('revision',c.revision,'secuencia',(SELECT pg_catalog.max(secuencia) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),'spki',encode(r.clave_publica_spki,'base64'),'clave_id',r.clave_id,'version',r.version,'audiencia',r.audiencia_despliegue,'desde',r.valida_desde,'hasta',r.valida_hasta,'orden',(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),'max_version',(SELECT max(version) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'max_revision',(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'pre_sha',pre)
 INTO doc
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version ORDER BY p.orden DESC LIMIT 1;
 RETURN doc;
END $function$;
COMMIT;
