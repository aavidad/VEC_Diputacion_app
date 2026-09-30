\set ON_ERROR_STOP on
-- PG18 real desechable con CT145/AD3-125/CT152. SOLO PRUEBA.
-- Dos dobles transaccionales sustituyen el consumo central criptográfico y
-- el consumidor de escritura de firma. AD3-125 nominal y CT152 son reales.
-- No acredita criptografía V3 ni firma legal. Todo termina en ROLLBACK.
-- Guardas de sesión sin dobles; cada bloque revierte su rol de prueba.
BEGIN ISOLATION LEVEL READ COMMITTED;
CREATE ROLE vec_ct152_guarda_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct152_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_ct152_guarda_ejecutor;
DO $guarda$
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO aislamiento';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ COMMITTED denegado'; END;
END $guarda$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
CREATE ROLE vec_ct152_guarda_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct152_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_ct152_guarda_ejecutor;
SET LOCAL transaction_read_only='on';
DO $guarda$
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO READ ONLY';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ ONLY denegado'; END;
END $guarda$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='3s';
SELECT set_config('prueba_ct152.org',v.agregado_json->>'organizacion_ref',true),
       set_config('prueba_ct152.exp',a.expediente_ref,true),
       set_config('prueba_ct152.version',a.version::text,true)
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
 WHERE NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f WHERE f.expediente_ref=a.expediente_ref)
 ORDER BY a.expediente_ref LIMIT 1;
CREATE ROLE vec_ct152_prueba_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct152_prueba_ejecutor;
CREATE TEMP TABLE prueba_ct152_consumos (capacidad bytea PRIMARY KEY,auditoria_ref text NOT NULL);
GRANT SELECT,INSERT ON pg_temp.prueba_ct152_consumos TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $stub$
DECLARE c jsonb:=convert_from(p_capacidad_canonica,'UTF8')::jsonb; nueva boolean; a text:='auditoria-prueba:'||md5(p_capacidad_canonica::text);
BEGIN
 IF p_perfil_mutacion IS DISTINCT FROM 'consulta_firmas_documento_ct' THEN RAISE EXCEPTION 'perfil de prueba incorrecto'; END IF;
 nueva:=NOT EXISTS (SELECT 1 FROM pg_temp.prueba_ct152_consumos WHERE capacidad=p_capacidad_canonica);
 IF nueva THEN INSERT INTO pg_temp.prueba_ct152_consumos VALUES(p_capacidad_canonica,a); END IF;
 RETURN QUERY SELECT 'decision-prueba:'||md5(p_capacidad_canonica::text),c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('b',64),a,clock_timestamp(),nueva;
END $stub$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $stub$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN RETURN QUERY SELECT 'decision-prueba:'||md5(random()::text),d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',encode(sha256(convert_to(random()::text,'UTF8')),'hex'),'auditoria-prueba:'||md5(random()::text),clock_timestamp(),true; END $stub$;
RESET ROLE;
CREATE FUNCTION pg_temp.solicitud(p_org text, p_exp text, p_version numeric, p_paso int, p_sec int, p_resultado text,
    p_motivo text, p_original text, p_firmado text, p_clave text, p_doc text DEFAULT NULL, p_doc_version numeric DEFAULT NULL) RETURNS text LANGUAGE sql AS $s$
  SELECT jsonb_build_object('OrganizacionRef',p_org,'ExpedienteRef',p_exp,'VersionExpediente',p_version,
    'Documento','informe_definitivo','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
    'CatalogoHuella',repeat('c',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p'||p_paso,
    'PasoOrden',p_paso,'Secuencia',p_sec,'Resultado',p_resultado,'MotivoDevolucion',p_motivo,
    'OriginalHuella',p_original,'FirmadoHuella',p_firmado,
    'CertificadoHuella',CASE WHEN p_resultado='firmado' THEN repeat('e',64) END,
    'FirmanteRef',CASE WHEN p_resultado='firmado' THEN 'ref:'||repeat('f',64) END,
    'PoliticaVerificacion',CASE WHEN p_resultado='firmado' THEN 'politica:vec:firma:verificacion-autonoma:v1' END,
    'RevocacionEstado',CASE WHEN p_resultado='firmado' THEN 'vigente' END,
    'SelloTiempoEstado',CASE WHEN p_resultado='firmado' THEN 'no_presente' END,
    'ClaveIdempotencia',p_clave,'DocumentoCustodiaRef',p_doc,'DocumentoCustodiaVersion',p_doc_version)::text
$s$;
CREATE FUNCTION pg_temp.decision(p_solicitud text) RETURNS bytea LANGUAGE sql AS $d$
  SELECT convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firmar','modulo_id','contratacion_temporal',
    'tipo_recurso','firma_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
    'recurso_ref','operacion-firma-ct:'||(p_solicitud::jsonb->>'ClaveIdempotencia'),
    'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(p_solicitud::jsonb->>'OrganizacionRef')||
      '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
    'principal_id','principal:prueba:rrhh','perfil_activo_ref','perfil:prueba:rrhh')::text,'UTF8')
$d$;
CREATE FUNCTION pg_temp.registrar(p_solicitud text) RETURNS jsonb LANGUAGE sql AS $r$
  SELECT vec_contratacion_temporal.registrar_firma_documento_v2(p_solicitud,'\x',pg_temp.decision(p_solicitud),'\x','\x',1,1,'\x','\x','\x','\x')
$r$;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text, p_codigo text, p_etiqueta text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN others THEN
        IF SQLSTATE <> p_codigo THEN
            RAISE EXCEPTION 'FALLO %: código % (esperado %): %', p_etiqueta, SQLSTATE, p_codigo, SQLERRM;
        END IF;
        RAISE NOTICE 'OK %', p_etiqueta;
        RETURN;
    END;
    RAISE EXCEPTION 'FALLO %: no se rechazó', p_etiqueta;
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
CREATE FUNCTION pg_temp.material(p_org text,p_exp text) RETURNS text LANGUAGE sql AS $m$
 SELECT '{"OrganizacionRef":'||to_json(p_org)::text||',"ExpedienteRef":'||to_json(p_exp)::text||'}'
$m$;
CREATE FUNCTION pg_temp.consultar(p_material text,p_nonce text,p_alteracion jsonb DEFAULT '{}'::jsonb) RETURNS jsonb LANGUAGE plpgsql AS $c$
DECLARE s jsonb:=p_material::jsonb; d jsonb; c jsonb; h text;
BEGIN
 h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||'"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('accion','contratacion_temporal.documento.firmas.consultar','modulo_id','contratacion_temporal','tipo_recurso','expediente_contratacion_temporal','finalidad','gestionar_contratacion_temporal','recurso_ref',s->>'ExpedienteRef','contexto_recurso_huella_sha256',h,
  'campos_permitidos','["CatalogoHuella","CatalogoRef","ClaveIdempotencia","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FirmaRef","FirmadoHuella","OriginalHuella","PasoOrden","PasoRef","ReciboRef","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado"]'::jsonb,'obligaciones','[]'::jsonb)||p_alteracion;
 c:=jsonb_build_object('audiencia_consumo','vec_contratacion_temporal.firmas_documento.consultar.v1','operacion','contratacion_temporal.documento.firmas.consultar','efecto_ref',s->>'ExpedienteRef','huella_efecto_sha256',h,'huella_decision_sha256',encode(sha256(convert_to(d::text,'UTF8')),'hex'),'nonce',p_nonce,'relleno',repeat('x',512));
 RETURN vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(p_material,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'\x01','\x01',1,1,'\x01','\x01','\x01',decode(repeat('a',88),'hex'));
END $c$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
SET SESSION AUTHORIZATION vec_ct152_prueba_ejecutor;
DO $recorrido$
DECLARE org text:=current_setting('prueba_ct152.org'); exp text:=current_setting('prueba_ct152.exp'); ver numeric:=current_setting('prueba_ct152.version')::numeric; m text; r jsonb; f jsonb;
BEGIN
 m:=pg_temp.material(org,exp);
 r:=pg_temp.consultar(m,'vacio');
 IF r IS DISTINCT FROM jsonb_build_object('Encontrado',true,'ExpedienteRef',exp,'Firmas','[]'::jsonb) THEN RAISE EXCEPTION 'FALLO existente sin firmas'; END IF;
 PERFORM pg_temp.registrar(pg_temp.solicitud(org,exp,ver,1,1,'firmado',NULL,repeat('a',64),repeat('1',64),'clave-ct152-00000001','ref:'||repeat('d',64),1));
 PERFORM pg_temp.registrar(pg_temp.solicitud(org,exp,ver,1,2,'devuelto','motivo de prueba',NULL,NULL,'clave-ct152-00000002'));
 r:=pg_temp.consultar(m,'firmas');
 IF r->>'Encontrado'<>'true' OR jsonb_array_length(r->'Firmas')<>2 THEN RAISE EXCEPTION 'FALLO firmas'; END IF;
 f:=r->'Firmas'->0;
 IF f->>'ClaveIdempotencia'<>'clave-ct152-00000001' OR f->>'DocumentoCustodiaRef'<>'ref:'||repeat('d',64) OR f->'DocumentoCustodiaVersion'<>'1'::jsonb
    OR (SELECT count(*) FROM jsonb_object_keys(f))<>18
    OR f ?| ARRAY['ActorRef','PerfilRef','FirmanteRef','CertificadoHuella','MotivoDevolucion'] THEN RAISE EXCEPTION 'FALLO proyección minimizada'; END IF;
 IF r->'Firmas'->1->'ConMotivoDevolucion'<>'true'::jsonb OR r->'Firmas'->1->'DocumentoCustodiaRef'<>'null'::jsonb THEN RAISE EXCEPTION 'FALLO devolución histórica'; END IF;
 r:=pg_temp.consultar(pg_temp.material(org,'expediente:prueba:ausente'),'ausente');
 IF r->'Encontrado'<>'false'::jsonb OR r->'Firmas'<>'[]'::jsonb THEN RAISE EXCEPTION 'FALLO ausencia autorizada'; END IF;
 r:=pg_temp.consultar(pg_temp.material('organizacion:prueba:ajena',exp),'ajeno');
 IF r->'Encontrado'<>'false'::jsonb OR r->'Firmas'<>'[]'::jsonb THEN RAISE EXCEPTION 'FALLO cruce organización'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',m,'firmas'),'42501','replay de lectura rechazado');
 PERFORM pg_temp.debe_fallar(format('SELECT vec_contratacion_temporal.consultar_firmas_documento_v1(%L,%L)',org,exp),'42501','raw v1 cerrado');
 PERFORM pg_temp.debe_fallar(format('SELECT vec_contratacion_temporal.consultar_firmas_documento_v2(%L,%L)',org,exp),'42501','raw v2 cerrado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'campos','{"campos_permitidos":["ActorRef"]}'),'42501','campos cruzados');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'obligaciones','{"obligaciones":["otra"]}'),'42501','obligaciones cruzadas');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'recurso','{"recurso_ref":"expediente:prueba:otro"}'),'42501','expediente cruzado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'accion','{"accion":"contratacion_temporal.documento.firmar"}'),'42501','acción de escritura no habilita lectura');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'contexto','{"contexto_recurso_huella_sha256":"otro"}'),'42501','contexto cruzado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',replace(m,'{"','{ "'),'espacios'),'22023','material no canónico');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)','{"OrganizacionRef":"'||org||'","OrganizacionRef":"'||org||'","ExpedienteRef":"'||exp||'"}','duplicadas'),'22023','clave duplicada');
 RAISE NOTICE 'OK consulta vacía, firmas/custodia/devolución, ausencia y organización ajena';
END $recorrido$;
RESET SESSION AUTHORIZATION;
DO $auditoria$
BEGIN
 IF (SELECT count(*) FROM pg_temp.prueba_ct152_consumos)<>4 THEN RAISE EXCEPTION 'FALLO consumo/auditoría de lecturas, incluida ausencia'; END IF;
 RAISE NOTICE 'OK cuatro lecturas con consumo/auditoría; denegaciones sin efectos';
END $auditoria$;
ROLLBACK;
