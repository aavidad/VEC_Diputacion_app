\set ON_ERROR_STOP on
-- Fixture del ensayo de registros simultáneos de CT118 sobre la estructura
-- real restaurada (volcado sintético) con AD3-85 y CT118 instaladas. Base
-- desechable: se confirma. SOLO PRUEBA: la fachada AD3-85 se sustituye por un
-- doble que devuelve un consumo nuevo ligado a la decisión (prueba la
-- transacción CT, no la criptografía V3). Deja en prueba_ct118s las
-- funciones para construir y registrar una solicitud con la cuenta de
-- ejecución, y el expediente elegido.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
    h text := encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
BEGIN
 RETURN QUERY SELECT 'decision:'||substr(h,1,32), d->>'recurso_ref', d->>'contexto_recurso_huella_sha256',
  h, 'auditoria:'||substr(h,33,32), clock_timestamp(), true;
END $f$;

CREATE ROLE vec_ct118s_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_ct118s_runtime;
CREATE SCHEMA prueba_ct118s;
GRANT USAGE ON SCHEMA prueba_ct118s TO vec_ct118s_runtime;
CREATE TABLE prueba_ct118s.expediente AS
SELECT a.expediente_ref, a.version, v.agregado_json->>'organizacion_ref' AS organizacion_ref
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref, version)
 ORDER BY a.expediente_ref LIMIT 1;
GRANT SELECT ON prueba_ct118s.expediente TO vec_ct118s_runtime;
-- Solicitud de firma de un paso del expediente elegido: todos los pasos
-- firman el mismo borrador («a…»), como exige la cadena de CT118.
CREATE FUNCTION prueba_ct118s.solicitud(p_clave text, p_paso int, p_secuencia int) RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS $s$
  SELECT jsonb_build_object('OrganizacionRef',e.organizacion_ref,'ExpedienteRef',e.expediente_ref,'VersionExpediente',e.version,
    'Documento','informe_definitivo','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
    'CatalogoHuella',repeat('c',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p'||p_paso,
    'PasoOrden',p_paso,'Secuencia',p_secuencia,'Resultado','firmado','MotivoDevolucion',NULL,
    'OriginalHuella',repeat('a',64),'FirmadoHuella',repeat('b',64),'CertificadoHuella',repeat('e',64),
    'FirmanteRef','ref:'||repeat('f',64),'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v1',
    'RevocacionEstado','vigente','SelloTiempoEstado','no_presente','ClaveIdempotencia',p_clave)::text
    FROM prueba_ct118s.expediente e
$s$;
CREATE FUNCTION prueba_ct118s.registrar(p_clave text, p_paso int, p_secuencia int) RETURNS jsonb LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $r$
  WITH s AS (SELECT prueba_ct118s.solicitud(p_clave,p_paso,p_secuencia) AS t)
  SELECT vec_contratacion_temporal.registrar_firma_documento_v1(s.t,'\x',
    convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firmar','modulo_id','contratacion_temporal',
      'tipo_recurso','firma_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
      'recurso_ref','operacion-firma-ct:'||p_clave,
      'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s.t::jsonb->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(s.t,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
      'principal_id','principal:prueba:rrhh','perfil_activo_ref','perfil:prueba:rrhh')::text,'UTF8'),
    '\x','\x',1,1,'\x','\x','\x','\x')
    FROM s
$r$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_ct118s TO vec_ct118s_runtime;
SELECT 'fixture CT118 simultaneo OK' AS resultado;
