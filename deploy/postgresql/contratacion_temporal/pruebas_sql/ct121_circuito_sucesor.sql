\set ON_ERROR_STOP on
-- Circuito real del sucesor con CT121 instalada, tras el antecedente de
-- ct121_circuito_antecedente.sql (expiración confirmada por RRHH y
-- continuación de CT119). Cuenta de ejecución y SERIALIZABLE:
--   CT62  aviso del sucesor con antecedente de continuación;
--   CT63  respuesta del sucesor (aceptación);
--   CT57  consulta del justificante con la continuación;
--   CT64  resolución manual de la aceptación del sucesor;
--   CT65  propuesta (consulta y confirmación) de la versión 6 a la 7.
-- Cada escritor, también los del antecedente, se repite con el mismo
-- material (replay sin efecto nuevo). Los recibos quedan en
-- prueba_ct121c.caso para compararlos tras el reinicio.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set llam_1 'llamamiento:nccfkjnioljdeikkpipkcgcpilbogjnociankdfbapmnaekanagbiioaahphbmgj'
\set llam_2 'llamamiento:eipmgkkfjncbalgebihinpeeajhllfmkoikjhiodlbcdlhaohmgpojefdfilcmoe'
\set org 'organizacion:desarrollo:dipgra'

SET SESSION AUTHORIZATION vec_ct121c_runtime;
SET timezone = 'UTC';

-- 4. CT62: aviso del sucesor con antecedente de continuación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('aviso','registrar_comunicacion_llamamiento_local_v1',prueba_ct121c.material_aviso(),
    'contratacion_temporal.llamamiento.comunicacion.registrar','comunicacion_llamamiento_contratacion_temporal') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('aviso')->>'Estado'='registrada_localmente','aviso del sucesor registrado');

-- 5. CT63: respuesta del sucesor (aceptación).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('respuesta','registrar_respuesta_recibida_rrhh_v1',
    format('{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000005","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionComunicacionEsperada":2,"Respuesta":"aceptacion","CorreoRef":"correo:sintetico:ct121-sucesor","CorreoSHA256":"%s","RecibidaEn":"%s"}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_2'),prueba_ct121c.r('aviso')->>'ComunicacionRef',
        encode(sha256('correo sintético ct121'::bytea),'hex'),to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')),
    'contratacion_temporal.llamamiento.respuesta.registrar','respuesta_recibida_llamamiento_contratacion_temporal') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('respuesta')->>'Estado'='registrada_por_rrhh','respuesta del sucesor registrada');

-- 6. CT57: consulta del justificante, que devuelve la continuación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('justificante','consultar_justificante_respuesta_recibida_rrhh_v1',
    format('{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000006","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionEsperada":2,"Respuesta":"aceptacion","PruebaRespuestaRef":"%s"}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_2'),prueba_ct121c.r('aviso')->>'ComunicacionRef',
        prueba_ct121c.r('respuesta')->>'JustificanteRef'),
    'contratacion_temporal.llamamiento.respuesta.consultar_justificante','justificante_respuesta_recibida_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('justificante')->'Continuacion'->>'ReciboRef'=prueba_ct121c.r('continuacion')->>'ReciboRef',
    'justificante con la continuación tras la expiración');

-- 7. CT64: resolución manual de la aceptación del sucesor.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('resolucion','registrar_resolucion_manual_respuesta_rrhh_v1',
    format('{"Solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000007","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionEsperada":2,"Respuesta":"aceptacion","PruebaRespuestaRef":"%s","RevisionRespuestaRRHH":true,"RevisionPlazoRRHH":true,"CriterioValidacionRef":"politica:ct:revision-manual-sintetica:20260906"},"Politica":{"Referencia":"politica:ct:revision-manual-sintetica:20260906","Version":1,"HuellaSHA256":"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"}}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_2'),prueba_ct121c.r('aviso')->>'ComunicacionRef',
        prueba_ct121c.r('respuesta')->>'JustificanteRef'),
    'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar','resolucion_manual_respuesta_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('resolucion')->>'Estado'='confirmado','aceptación del sucesor resuelta');

-- 8. CT65: propuesta de la versión 6 a la 7 (consulta y confirmación).
SELECT format('"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000008","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ResolucionLlamamientoAceptadaRef":"%s","ReciboResolucionAceptadaRef":"%s","VersionEsperada":6,"TipoFormalizacion":%s,"Plantilla":%s,"Anexos":null,"PoliticaFirma":%s,"PlanFirma":%s',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_2'),
        prueba_ct121c.r('resolucion')->>'ResolucionRef',prueba_ct121c.r('resolucion')->>'ReciboLocalRef',
        prueba_ct121c.publicacion('TipoFormalizacion'),prueba_ct121c.publicacion('Plantilla'),
        prueba_ct121c.publicacion('PoliticaFirma'),prueba_ct121c.publicacion('PlanFirma')) AS solicitud_propuesta \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('propuesta-consulta','registrar_propuesta_formalizacion_v1',
    '{"Etapa":"consulta","Solicitud":{'||:'solicitud_propuesta'||'}}',
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('propuesta-consulta')->'Justificante'->'Continuacion'->>'ReciboRef'
    =prueba_ct121c.r('continuacion')->>'ReciboRef','consulta de propuesta con la continuación tras la expiración');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('propuesta','registrar_propuesta_formalizacion_v1',
    '{"Etapa":"confirmacion","Solicitud":{'||:'solicitud_propuesta'||'},"AceptacionBolsa":'||format(
        '{"OperacionRef":"operacion-aceptacion-rrhh:ct121","AperturaOperacionRef":"%s","LlamamientoRef":"%s","JustificanteRef":"%s","EvaluacionPlazoRef":"%s","Politica":%s,"RegistroSHA256":"%s","ResueltaEn":"%s"}',
        prueba_ct121c.r('continuacion')->'ReciboBolsa'->>'OperacionRef',prueba_ct121c.v('llam_2'),
        prueba_ct121c.r('respuesta')->>'JustificanteRef',prueba_ct121c.r('resolucion')->>'EvaluacionPlazoRef',
        prueba_ct121c.r('resolucion')->'Politica',repeat('cd',32),prueba_ct121c.instante(clock_timestamp()))||'}',
    'contratacion_temporal.formalizacion.propuesta.registrar','propuesta_formalizacion_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('propuesta')->>'Estado'='confirmado'
    AND (prueba_ct121c.r('propuesta')->>'VersionResultante')::int=7,'propuesta confirmada: versión 7');

-- 9. Replays inmediatos: mismo recibo, sin efecto nuevo.
CREATE TEMP TABLE replay AS SELECT caso, prueba_ct121c.repetir(caso) AS valor FROM prueba_ct121c.caso WHERE false;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO replay SELECT caso, prueba_ct121c.repetir(caso) FROM prueba_ct121c.caso
 WHERE caso IN ('contacto','expiracion','continuacion','aviso','respuesta','resolucion','propuesta') ORDER BY caso;
COMMIT;
SELECT prueba_ct121c.exigir(count(*)=7 AND bool_and(r.valor->>'Estado' LIKE 'replay%'
        AND r.valor-'Estado'=c.resultado-'Estado'),'siete replays devuelven el recibo original')
  FROM replay r JOIN prueba_ct121c.caso c USING (caso);
RESET SESSION AUTHORIZATION;

-- 10. Historia: una fila por hecho, sin duplicados; expediente en la versión 7.
SELECT prueba_ct121c.exigir(
    (SELECT count(*) FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh WHERE expediente_ref=:'exp_b')=1
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE expediente_ref=:'exp_b')=2
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
          WHERE expediente_ref=:'exp_b' AND solicitud_json->>'Respuesta'='expiracion_gobernada'
            AND justificante_ref IS NULL AND continuacion_clave IS NOT NULL)=1
    AND (SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local WHERE expediente_ref=:'exp_b')=2
    AND (SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=:'exp_b')=1
    AND (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=:'exp_b')=1
    AND (SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_b')=7,
    'historia única del circuito tras la expiración');
SELECT 'CT121 circuito OK' AS resultado;
