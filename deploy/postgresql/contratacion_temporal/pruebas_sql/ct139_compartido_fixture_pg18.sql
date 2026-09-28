\set ON_ERROR_STOP on
-- Solo fixture de consulta CT139: V3 se sustituye por auditoría sintética.
SET ROLE vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh (
    resolucion_ref text PRIMARY KEY, organizacion_ref text, expediente_ref text,
    llamamiento_ref text, comunicacion_ref text, seleccion_clave uuid, estado text,
    solicitud_json jsonb, continuacion_clave uuid, continuacion_recibo jsonb);
ALTER TABLE vec_contratacion_temporal.comunicacion_llamamiento_local ADD COLUMN recibo_json jsonb;
UPDATE vec_contratacion_temporal.comunicacion_llamamiento_local c
   SET recibo_json=jsonb_build_object('ComunicacionRef',c.comunicacion_ref,
       'Solicitud',jsonb_build_object('OrganizacionRef',c.organizacion_ref,
           'ExpedienteRef',c.expediente_ref,'LlamamientoRef',c.llamamiento_ref,
           'PruebaEntregaRef',c.material_json->'solicitud'->>'PruebaEntregaRef'));
UPDATE vec_contratacion_temporal.comunicacion_llamamiento_local
   SET material_json=jsonb_set(material_json,'{solicitud}',recibo_json->'Solicitud');
INSERT INTO vec_contratacion_temporal.resolucion_manual_respuesta_rrhh VALUES (
    'resolucion:previa','org:ct138','exp:ct138','llamamiento:nuevo',
    'comunicacion:nueva','22222222-2222-4222-8222-222222222222','confirmado',
    '{"Respuesta":"renuncia"}','44444444-4444-4444-8444-444444444444',
    jsonb_build_object('Estado','confirmado','ReciboRef','recibo:continuacion',
      'Solicitud',jsonb_build_object('OrganizacionRef','org:ct138',
        'ExpedienteRef','exp:ct138','ResolucionRef','resolucion:previa',
        'ClaveIdempotencia','44444444-4444-4444-8444-444444444444'),
      'LlamamientoAnteriorRef','llamamiento:nuevo',
      'ReciboBolsa',jsonb_build_object('LlamamientoRef','llamamiento:sucesor')));
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
    (comunicacion_ref,organizacion_ref,expediente_ref,llamamiento_ref,estado,
     version_resultante,material_json,seleccion_clave,recibo_json)
SELECT 'comunicacion:sucesora','org:ct138','exp:ct138','llamamiento:sucesor',
    'registrada_localmente',2,
    jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','org:ct138',
      'ExpedienteRef','exp:ct138','LlamamientoRef','llamamiento:sucesor',
      'PruebaEntregaRef','recibo:continuacion','TipoAntecedente','continuacion_confirmada')),
    '22222222-2222-4222-8222-222222222222',
    jsonb_build_object('ComunicacionRef','comunicacion:sucesora',
      'Solicitud',jsonb_build_object('OrganizacionRef','org:ct138',
        'ExpedienteRef','exp:ct138','LlamamientoRef','llamamiento:sucesor',
        'PruebaEntregaRef','recibo:continuacion','TipoAntecedente','continuacion_confirmada'));
DO $respuesta$
DECLARE b vec_contratacion_temporal.respuesta_recibida_rrhh%ROWTYPE;
        m jsonb; h text;
BEGIN
  SELECT * INTO STRICT b FROM vec_contratacion_temporal.respuesta_recibida_rrhh
   WHERE comunicacion_ref='comunicacion:nueva';
  m:=b.material_json || jsonb_build_object('LlamamientoRef','llamamiento:sucesor',
      'ComunicacionRef','comunicacion:sucesora',
      'ClaveIdempotencia','55555555-5555-4555-8555-555555555555');
  h:=encode(sha256(convert_to(m::text,'UTF8')),'hex');
  INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh
    (justificante_ref,organizacion_ref,expediente_ref,llamamiento_ref,comunicacion_ref,
     seleccion_clave,clave_idempotencia,actor_ref,perfil_ref,version_comunicacion,
     respuesta,correo_ref,correo_sha256,recibida_en,material,material_json,
     material_huella_sha256,recibo_ref,recibo_json,estado,registrada_en)
  VALUES ('justificante:sucesora',b.organizacion_ref,b.expediente_ref,
     'llamamiento:sucesor','comunicacion:sucesora',b.seleccion_clave,
     '55555555-5555-4555-8555-555555555555',b.actor_ref,b.perfil_ref,
     2,b.respuesta,b.correo_ref,b.correo_sha256,b.recibida_en,m::text,m,h,
     'recibo:sucesora',jsonb_build_object('Solicitud',m,
       'JustificanteRef','justificante:sucesora','ReciboRef','recibo:sucesora',
       'AuditoriaRef','auditoria:sucesora',
       'RegistradaEn',to_char(b.registrada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
       'Estado','registrada_por_rrhh'),'registrada_por_rrhh',b.registrada_en);
  INSERT INTO vec_contratacion_temporal.historia_respuesta_recibida_rrhh
    (auditoria_ref,justificante_ref,actor_ref,perfil_ref,decision_ref,
     consumo_huella_sha256,evidencia_huella_sha256,material_huella_sha256,registrada_en)
  VALUES ('auditoria:sucesora','justificante:sucesora',b.actor_ref,b.perfil_ref,
    'decision:sucesora',repeat('f',64),repeat('e',64),h,b.registrada_en);
END $respuesta$;
RESET ROLE;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.ct139_lecturas_sinteticas (
    auditoria_ref text PRIMARY KEY, lector text NOT NULL, efecto_ref text NOT NULL);
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; a text;
BEGIN
    d:=convert_from(p_decision,'UTF8')::jsonb;
    a:='auditoria:'||gen_random_uuid()::text;
    INSERT INTO vec_autorizacion_atestada_v3.ct139_lecturas_sinteticas
        VALUES (a,d->>'principal_id',d->>'recurso_ref');
    RETURN QUERY SELECT d->>'decision_ref',d->>'recurso_ref',
        d->>'contexto_recurso_huella_sha256',repeat('a',64),a,clock_timestamp(),true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;
RESET ROLE;

SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.ct139_consultar_sintetico(
    p_organizacion text,p_expediente text,p_comunicacion text,p_actor text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $funcion$
DECLARE s text; h text; d jsonb; c jsonb;
BEGIN
    s:='{"OrganizacionRef":"'||p_organizacion||'","ExpedienteRef":"'||
       p_expediente||'","ComunicacionRef":"'||p_comunicacion||'"}';
    h:=encode(sha256(convert_to(s,'UTF8')),'hex');
    h:=encode(sha256(convert_to('{"ambitos":{"expediente_ref":"'||p_expediente||
       '","organizacion_ref":"'||p_organizacion||'"},"atributos":{"material_sha256":"'||
       h||'"}}','UTF8')),'hex');
    d:=jsonb_build_object('decision_ref','decision:lectura','accion',
       'contratacion_temporal.llamamiento.respuesta.consultar_recibo',
       'modulo_id','contratacion_temporal','tipo_recurso','respuesta_recibida_comunicacion_ct',
       'finalidad','gestionar_contratacion_temporal','recurso_ref',p_comunicacion,
       'contexto_recurso_huella_sha256',h,'principal_id',p_actor,
       'perfil_activo_ref','perfil:rrhh','campos_permitidos',
       '["auditoria_ref","comunicacion_ref","estado","expediente_ref","justificante_ref","organizacion_ref","recibo_ref","registrada_en","respuesta"]'::jsonb,
       'obligaciones','[]'::jsonb);
    c:=jsonb_build_object('principal_ref',p_actor,'perfil_activo_ref','perfil:rrhh',
       'persona_version','1','perfil_version','1');
    RETURN vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(s,
       decode('01','hex'),convert_to(d::text,'UTF8'),decode('02','hex'),
       convert_to(c::text,'UTF8'),1,1,decode('03','hex'),decode('04','hex'),
       decode('05','hex'),decode('06','hex'));
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.ct139_consultar_sintetico(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.ct139_consultar_sintetico(text,text,text,text)
    TO vec_contratacion_temporal_ejecutor;
RESET ROLE;
