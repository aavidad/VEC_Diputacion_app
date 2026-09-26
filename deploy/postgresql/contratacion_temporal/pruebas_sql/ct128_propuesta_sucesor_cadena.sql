\set ON_ERROR_STOP on
-- Cadena de CT128 con la estructura real restaurada: tras la no incorporación
-- de A (CT124) y la continuación confirmada (siguiente llamamiento B), con la
-- cuenta de ejecución y en SERIALIZABLE:
--   CT62/CT63/CT57/CT64  aviso, aceptación, justificante y resolución de B;
--   CT96+CT128           propuesta de nombramiento de B (versión 8 → 9), que
--                        deja la de A sustituida por su no incorporación;
--   CT124/CT115          incorporación de B (sintética, como en CT124),
--                        publicación a Bolsa, confirmación de GINPIX, cese y
--                        cierre con la nueva persona.
-- Negativos: segunda propuesta sin no incorporación (función y escritura
-- directa), otra clave, versión anterior y aceptación ya usada. Requiere
-- antes ct128_propuesta_sucesor_fixture.sql y, en la misma sesión,
-- ct124_utilidades.sql. Datos sintéticos; base desechable.
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set llam_b 'llamamiento:prueba:b42'

SELECT agregado_json->>'organizacion_ref' AS org FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=7 \gset
SELECT propuesta_ref AS prop_a, llamamiento_ref AS llam_a FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=:'exp_a' \gset
SELECT recibo_ref AS ni_recibo FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE expediente_ref=:'exp_a' \gset
SELECT cont::text AS cont FROM public.prueba_ct124_ni \gset
INSERT INTO prueba_ct128.valor VALUES ('exp_a',:'exp_a'),('org',:'org'),('llam_b',:'llam_b'),('prop_a',:'prop_a'),('ni_recibo',:'ni_recibo'),
  ('cont_recibo',(:'cont'::jsonb)->>'ReciboRef'),('apertura_b',(:'cont'::jsonb)#>>'{ReciboBolsa,OperacionRef}');
SELECT prueba_ct128.exigir((SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_a')=8
  AND (SELECT fase_clave FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=8)='fiscalizacion',
  'A en fiscalización (versión 8) tras su no incorporación');

SET SESSION AUTHORIZATION vec_ct115_runtime;
SET timezone = 'UTC';

-- 1. CT62: aviso a B con la continuación de la no incorporación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('aviso','registrar_comunicacion_llamamiento_local_v1',
    format('{"solicitud":{"ClaveIdempotencia":"c1280000-0000-4000-8000-000000000001","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","VersionEsperada":1,"PruebaEntregaRef":"%s","TipoAntecedente":"continuacion_confirmada"},"canal":{"Referencia":"canal:ct:desarrollo:registro-local:v1","Version":1,"HuellaSHA256":"c2346af19aa8489c3b878b2c780fcf0251129e0a397ed6e3c32c4438c44a88dc"},"politica":{"Referencia":"politica:ct:desarrollo:registro-local:v2","Version":2,"HuellaSHA256":"24d08a85ca5e62b8b3832df49d208ccea098f2b202909639d900b9d42da0bacb"}}',
        prueba_ct128.v('org'),prueba_ct128.v('exp_a'),prueba_ct128.v('llam_b'),prueba_ct128.v('cont_recibo')),
    'contratacion_temporal.llamamiento.comunicacion.registrar','comunicacion_llamamiento_contratacion_temporal') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('aviso')->>'Estado'='registrada_localmente','aviso a B registrado');

-- 2. CT63: B acepta.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('respuesta','registrar_respuesta_recibida_rrhh_v1',
    format('{"ClaveIdempotencia":"c1280000-0000-4000-8000-000000000002","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionComunicacionEsperada":2,"Respuesta":"aceptacion","CorreoRef":"correo:sintetico:ct128-sucesor","CorreoSHA256":"%s","RecibidaEn":"%s"}',
        prueba_ct128.v('org'),prueba_ct128.v('exp_a'),prueba_ct128.v('llam_b'),prueba_ct128.r('aviso')->>'ComunicacionRef',
        encode(sha256('correo sintético de Lucía Moreno Castillo'::bytea),'hex'),to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')),
    'contratacion_temporal.llamamiento.respuesta.registrar','respuesta_recibida_llamamiento_contratacion_temporal') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('respuesta')->>'Estado'='registrada_por_rrhh','aceptación de B registrada');

-- 3. CT57: justificante con la continuación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('justificante','consultar_justificante_respuesta_recibida_rrhh_v1',
    format('{"ClaveIdempotencia":"c1280000-0000-4000-8000-000000000003","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionEsperada":2,"Respuesta":"aceptacion","PruebaRespuestaRef":"%s"}',
        prueba_ct128.v('org'),prueba_ct128.v('exp_a'),prueba_ct128.v('llam_b'),prueba_ct128.r('aviso')->>'ComunicacionRef',
        prueba_ct128.r('respuesta')->>'JustificanteRef'),
    'contratacion_temporal.llamamiento.respuesta.consultar_justificante','justificante_respuesta_recibida_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('justificante')->'Continuacion'->>'ReciboRef'=prueba_ct128.v('cont_recibo'),
    'justificante de B con la continuación de la no incorporación');

-- 4. CT64: resolución manual de la aceptación de B.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('resolucion','registrar_resolucion_manual_respuesta_rrhh_v1',
    format('{"Solicitud":{"ClaveIdempotencia":"c1280000-0000-4000-8000-000000000004","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionEsperada":2,"Respuesta":"aceptacion","PruebaRespuestaRef":"%s","RevisionRespuestaRRHH":true,"RevisionPlazoRRHH":true,"CriterioValidacionRef":"politica:ct:revision-manual-sintetica:20260906"},"Politica":{"Referencia":"politica:ct:revision-manual-sintetica:20260906","Version":1,"HuellaSHA256":"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"}}',
        prueba_ct128.v('org'),prueba_ct128.v('exp_a'),prueba_ct128.v('llam_b'),prueba_ct128.r('aviso')->>'ComunicacionRef',
        prueba_ct128.r('respuesta')->>'JustificanteRef'),
    'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar','resolucion_manual_respuesta_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('resolucion')->>'Estado'='confirmado','aceptación de B resuelta');

-- 5. Propuesta de B desde la versión 8 (consulta y confirmación).
CREATE FUNCTION pg_temp.solicitud_propuesta(p_clave text, p_version integer) RETURNS text LANGUAGE sql AS $f$
 SELECT format('"ClaveIdempotencia":"%s","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ResolucionLlamamientoAceptadaRef":"%s","ReciboResolucionAceptadaRef":"%s","VersionEsperada":%s,"TipoFormalizacion":%s,"Plantilla":%s,"Anexos":null,"PoliticaFirma":%s,"PlanFirma":%s',
   p_clave,prueba_ct128.v('org'),prueba_ct128.v('exp_a'),prueba_ct128.v('llam_b'),
   prueba_ct128.r('resolucion')->>'ResolucionRef',prueba_ct128.r('resolucion')->>'ReciboLocalRef',p_version,
   prueba_ct128.v('publicacion:TipoFormalizacion'),prueba_ct128.v('publicacion:Plantilla'),
   prueba_ct128.v('publicacion:PoliticaFirma'),prueba_ct128.v('publicacion:PlanFirma')) $f$;
CREATE FUNCTION pg_temp.material_propuesta(p_clave text, p_version integer) RETURNS text LANGUAGE sql AS $f$
 SELECT '{"Etapa":"confirmacion","Solicitud":{'||pg_temp.solicitud_propuesta(p_clave,p_version)||'},"AceptacionBolsa":'||format(
   '{"OperacionRef":"operacion-aceptacion-rrhh:ct128","AperturaOperacionRef":"%s","LlamamientoRef":"%s","JustificanteRef":"%s","EvaluacionPlazoRef":"%s","Politica":%s,"RegistroSHA256":"%s","ResueltaEn":"%s"}',
   prueba_ct128.v('apertura_b'),prueba_ct128.v('llam_b'),prueba_ct128.r('respuesta')->>'JustificanteRef',
   prueba_ct128.r('resolucion')->>'EvaluacionPlazoRef',prueba_ct128.r('resolucion')->'Politica',repeat('cd',32),
   prueba_ct128.instante(date_trunc('microseconds',clock_timestamp())))||'}' $f$;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('propuesta-consulta','registrar_propuesta_formalizacion_v2',
    '{"Etapa":"consulta","Solicitud":{'||pg_temp.solicitud_propuesta('c1280000-0000-4000-8000-000000000005',8)||'}}',
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('propuesta-consulta')->'Justificante'->'Continuacion'->>'ReciboRef'=prueba_ct128.v('cont_recibo')
    AND (prueba_ct128.r('propuesta-consulta')->'Justificante'->'Seleccion'->>'version_expediente')::int=6,
    'consulta de la propuesta de B: continuación de la no incorporación y selección original');
-- Negativo: desde una versión anterior a la no incorporación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.exigir(prueba_ct128.codigo('registrar_propuesta_formalizacion_v2',pg_temp.material_propuesta('c1280000-0000-4000-8000-000000000015',7),
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct')='P0615','propuesta de B desde la versión 7 rechazada');
ROLLBACK;
SELECT pg_temp.material_propuesta('c1280000-0000-4000-8000-000000000005',8) AS mat_prop \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.paso('propuesta','registrar_propuesta_formalizacion_v2',:'mat_prop',
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct128.exigir(prueba_ct128.r('propuesta')->>'Estado'='confirmado'
    AND (prueba_ct128.r('propuesta')->>'VersionResultante')::int=9,'propuesta de B confirmada: versión 9 '||prueba_ct128.r('propuesta')::text);
-- Repetición idéntica: mismo recibo, sin efecto nuevo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.repetir('propuesta')::text AS replay_prop \gset
COMMIT;
SELECT prueba_ct128.exigir((:'replay_prop'::jsonb)->>'Estado'='replay_confirmado'
    AND (:'replay_prop'::jsonb)-'Estado'=prueba_ct128.r('propuesta')-'Estado','repetición de la propuesta de B con el mismo recibo');
-- Negativos: otra clave con la misma aceptación y la aceptación de A.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.exigir(prueba_ct128.codigo('registrar_propuesta_formalizacion_v2',pg_temp.material_propuesta('c1280000-0000-4000-8000-000000000016',9),
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IN ('P0611','P0615'),'segunda propuesta de B con otra clave rechazada');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct128.exigir(prueba_ct128.codigo('registrar_propuesta_formalizacion_v2',
    replace(pg_temp.material_propuesta('c1280000-0000-4000-8000-000000000017',8),'c1280000-0000-4000-8000-000000000017','c1280000-0000-4000-8000-000000000018'),
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IN ('P0611','P0612'),'otra propuesta sobre la versión ya propuesta rechazada');
ROLLBACK;
-- Consulta del panel: la propuesta vigente y la historia.
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_propuestas_expediente_v1(prueba_ct128.v('org'),prueba_ct128.v('exp_a'))::text AS consulta \gset
COMMIT;
SELECT prueba_ct128.exigir(jsonb_array_length((:'consulta'::jsonb)->'propuestas')=2
    AND (:'consulta'::jsonb)#>>'{propuestas,0,vigente}'='false' AND (:'consulta'::jsonb)#>>'{propuestas,0,version_resultante}'='7'
    AND (:'consulta'::jsonb)#>>'{propuestas,0,sustitucion,motivo_clave}'='no_presentado'
    AND (:'consulta'::jsonb)#>>'{propuestas,0,sustitucion,no_incorporacion_recibo_ref}'=prueba_ct128.v('ni_recibo')
    AND (:'consulta'::jsonb)#>>'{propuestas,1,vigente}'='true' AND (:'consulta'::jsonb)#>>'{propuestas,1,version_resultante}'='9'
    AND (:'consulta'::jsonb)#>'{propuestas,1,sustitucion}'='null'::jsonb
    AND (:'consulta'::jsonb)#>>'{propuestas,1,recibo_ref}'=prueba_ct128.r('propuesta')->>'ReciboLocalRef','consulta: A sustituida y B vigente '||:'consulta');
SELECT prueba_ct128.exigir(NOT has_table_privilege('vec_contratacion_temporal.propuesta_sustitucion_v1','SELECT')
    AND NOT has_function_privilege('vec_contratacion_temporal.propuesta_sustituible_ct128(jsonb,text,text,numeric)','EXECUTE'),
    'tabla y auxiliar de la sustitución cerrados al ejecutor');
-- La no incorporación de B se puede preparar (la propuesta vigente es la de B).
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada('registrar_no_incorporacion','no-incorporacion','contratacion_temporal.incorporacion.no_incorporacion',
   'no_incorporacion_contratacion_temporal','registrar_no_incorporacion_contratacion_temporal',:'exp_a',9,
   jsonb_build_object('motivo_clave','no_presentado','consecuencia_clave','b24.sancion.baja_llamamiento_directo',
     'resolucion_ref','resolucion:rrhh:2026/0150','resolucion_sha256',repeat('b',64),'resuelta_por','per_ct124_segunda','segunda_persona',true,
     'fecha_notificacion','2026-09-21','observaciones',''),'{}'::jsonb,'{}'::jsonb,'en_curso','ni_b128')::text AS ni_b \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_no_incorporacion_v1','vec.contratacion-temporal.preparar-no-incorporacion.v1',:'ni_b'::jsonb)::text AS prep_ni_b \gset
COMMIT;
SELECT prueba_ct128.exigir((:'prep_ni_b'::jsonb)->>'resultado'='preparada'
    AND (:'prep_ni_b'::jsonb)#>>'{aceptacion,resolucion_ref}'=prueba_ct128.r('resolucion')->>'ResolucionRef',
    'la no incorporación de B tomaría su aceptación '||:'prep_ni_b');
RESET SESSION AUTHORIZATION;

-- Historia: A sustituida por su no incorporación; B vigente.
SELECT prueba_ct128.exigir(
    (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=:'exp_a')=2
    AND (SELECT count(*) FROM vec_contratacion_temporal.propuesta_sustitucion_v1 s
          WHERE s.expediente_ref=:'exp_a' AND s.propuesta_anterior_ref=:'prop_a' AND s.ronda=2
            AND s.no_incorporacion_recibo_ref=:'ni_recibo' AND s.propuesta_ref=prueba_ct128.r('propuesta')->>'PropuestaRef')=1
    AND (SELECT fase_clave||'/'||estado FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=9)='nombramiento/en_curso'
    AND (SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_a' AND version_expediente=9
          AND tipo_evento='contratacion_temporal.propuesta_formalizacion_registrada')=1,
    'historia: propuesta de A conservada y sustituida; propuesta de B en la versión 9');

-- Negativo estructural: una segunda propuesta vigente escrita directamente
-- (sin sustitución) se rechaza al confirmar la transacción.
SELECT propuesta_ref AS prop_b0 FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=:'exp_b' \gset
CREATE FUNCTION pg_temp.duplicar_propuesta(p_origen text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
    SET CONSTRAINTS ALL IMMEDIATE;
    INSERT INTO vec_contratacion_temporal.propuesta_formalizacion
    SELECT 'propuesta:ct128:duplicada',organizacion_ref,expediente_ref,llamamiento_ref,resolucion_ref||'x',recibo_aceptacion_ref,gen_random_uuid(),
           version_previa,version_resultante,material,material_json,material_sha256,solicitud_json,aceptacion_bolsa_json,actor_ref,perfil_ref,
           'aud:ct128:dup','decision:ct128:dup',repeat('9',64),evidencia_huella_sha256,'recibo:ct128:dup',recibo_json,evento_ref||'x',confirmada_en
      FROM vec_contratacion_temporal.propuesta_formalizacion WHERE propuesta_ref=p_origen;
    RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE||' '||SQLERRM;
END $f$;
BEGIN;
SET LOCAL session_replication_role=origin;
SELECT pg_temp.duplicar_propuesta(:'prop_b0') AS dup \gset
ROLLBACK;
SELECT prueba_ct128.exigir(:'dup' LIKE '23%','segunda propuesta sin no incorporación rechazada por la base: '||:'dup');

-- 6. Incorporación de B (sintética como en CT124: solo el periodo) en la
-- versión 9, publicación a Bolsa, GINPIX, cese y cierre con B.
SET session_replication_role=replica;
INSERT INTO vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref,seguimiento_ref,organizacion_ref,idempotencia_ref,solicitud_ref,expediente_ref,
  version_expediente,version_anterior,version_resultante,estado_anterior_sha256,estado_resultante_sha256,material_json,material_canonico,material_sha256,
  intencion_canonica,intencion_sha256,exportacion_ct,persona_version,perfil_version,recibo_json,auditoria_ref,outbox_ref,registrada_en,evidencia_orden_json)
SELECT 'ref:'||repeat('b',64),'seguimiento:ct128:b',:'org','idem:ct128:b','solicitud:ct128:b',:'exp_a',9,1,2,repeat('1',64),repeat('2',64),
  '{"Confirmacion":{"PeriodoIncorporacion":{"desde":"2027-01-01T00:00:00Z","hasta":"2027-03-31T00:00:00Z"}}}','\x01',encode(sha256('\x01'::bytea),'hex'),
  '\x02',encode(sha256('\x02'::bytea),'hex'),ARRAY['\x01','\x01','\x01','\x01','\x01','\x01','\x01','\x01']::bytea[],1,1,
  jsonb_build_object('MaterialOriginalSHA256',encode(sha256('\x01'::bytea),'hex'),'IntencionSHA256',encode(sha256('\x02'::bytea),'hex'),
    'SeguimientoRef','seguimiento:ct128:b','AuditoriaCTRef','auditoria:ct128:b','OutboxCTRef','ref:outbox:'||repeat('b',64),'Transicion',jsonb_build_object('recibo_ref','ref:'||repeat('b',64)),
    'EjercicioSintetico',true,'FirmaOficial',false,'EficaciaAdministrativa',false),'auditoria:ct128:b','ref:outbox:'||repeat('b',64),clock_timestamp(),'{}';
INSERT INTO vec_contratacion_temporal.incorporacion_outbox_v2(outbox_ref,recibo_ref,evento_json,estado_sha256,creada_en)
VALUES ('ref:outbox:'||repeat('b',64),'ref:'||repeat('b',64),jsonb_build_object('recibo_ref','ref:'||repeat('b',64)),repeat('a',64),clock_timestamp());
RESET session_replication_role;
SELECT 'ref:'||repeat('b',64) AS inc_b \gset

SELECT pg_temp.entrada_ginpix(:'exp_a',9,'GX-2027-0128','2027-02-10','g128',:'inc_b')::text AS ginpix \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix'::jsonb)::text AS prep_g \gset
COMMIT;
SELECT prueba_ct128.exigir((:'prep_g'::jsonb)->>'resultado'='preparada' AND (:'prep_g'::jsonb)#>>'{incorporacion,recibo_ref}'=:'inc_b','GINPIX de B preparada '||:'prep_g');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix'::jsonb)::text AS r_g \gset
COMMIT;
SELECT prueba_ct128.exigir((:'r_g'::jsonb)->>'resultado'='confirmada' AND (:'r_g'::jsonb)#>>'{recibo,version_resultante}'='10','GINPIX de B confirmada '||:'r_g');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada_cese(:'exp_a',10,'cese128',:'inc_b')::text AS cese \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_cese_nombramiento_v1',:'cese'::jsonb)::text AS r_cese \gset
COMMIT;
SELECT prueba_ct128.exigir((:'r_cese'::jsonb)->>'resultado'='confirmada' AND (:'r_cese'::jsonb)#>>'{recibo,version_resultante}'='11','cese de B '||:'r_cese');
RESET SESSION AUTHORIZATION;
SELECT prueba_ct128.exigir((SELECT llamamiento_ref FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE expediente_ref=:'exp_a')=:'llam_b'
    AND (SELECT incorporacion_ref FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE expediente_ref=:'exp_a')=:'inc_b',
    'el cese es el de B: su llamamiento y su incorporación');
SELECT pg_temp.entrada_cierre(:'exp_a',11,'GX-2027-0128','2027-02-10','cierre128',(:'r_cese'::jsonb)#>>'{recibo,recibo_ref}')::text AS cierre \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre'::jsonb)::text AS r_cierre \gset
COMMIT;
SELECT prueba_ct128.exigir((:'r_cierre'::jsonb)->>'resultado'='confirmada' AND (:'r_cierre'::jsonb)#>>'{recibo,estado_resultante}'='completado'
    AND (:'r_cierre'::jsonb)#>>'{recibo,version_resultante}'='12','cierre del expediente con B '||:'r_cierre');
RESET SESSION AUTHORIZATION;
-- Publicación a Bolsa: un solo contrato por la incorporación de B, con su llamamiento.
SELECT count(*) AS contratos_b, min(evento->>'llamamiento_ref') AS llam_contrato
  FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100)
 WHERE evento->>'expediente_ref'=:'exp_a' AND evento->>'tipo'='incorporacion' \gset
SELECT prueba_ct128.exigir(:'contratos_b'::int=1 AND :'llam_contrato'=:'llam_b',
    'publicación a Bolsa sin duplicar la incorporación y con el llamamiento de B');

INSERT INTO prueba_ct128.valor VALUES ('ginpix',:'ginpix'),('r_g',:'r_g'),('cese',:'cese'),('r_cese',:'r_cese'),('cierre',:'cierre'),
  ('r_cierre',:'r_cierre'),('consulta',:'consulta'),('conteo',prueba_ct128.conteo(:'exp_a'));
SELECT 'cadena CT128 OK';
