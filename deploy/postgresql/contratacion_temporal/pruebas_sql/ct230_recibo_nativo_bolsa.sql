\set ON_ERROR_STOP on
BEGIN;
SET TRANSACTION READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

-- Focal de canon JSON: referencias y sellos sintéticos; no se confirma O6.
DO $ct230_canon$
DECLARE
    v_ref jsonb := pg_catalog.jsonb_build_object(
        'referencia','ref:prueba','version',1,'huella_sha256',pg_catalog.repeat('a',64));
    v_evidencia jsonb := pg_catalog.jsonb_build_object(
        'evidencia_ref','ref:prueba','clave_verificacion_ref','ref:prueba',
        'sello_hmac','ref:prueba','emitida_en','2026-10-16T00:37:15Z',
        'valida_hasta','2026-10-16T00:42:15Z',
        'retener_hasta','2026-10-17T00:42:15Z');
    v_recibo jsonb;
    v_legado text;
    v_nativo text;
    v_politica text;
BEGIN
    v_recibo := pg_catalog.jsonb_build_object(
        'operacion_ref','ref:prueba','organizacion_ref','ref:prueba',
        'expediente_ref','ref:prueba','version_expediente',1,
        'correlacion_ref','ref:prueba','necesidad',v_ref,'bolsa',v_ref,
        'orden',v_ref,'politica',v_ref,'resultado',v_ref,
        'propuesta_generada',true,'propuesta',v_ref,'accion_evento',v_ref,
        'llamamiento_ref','ref:prueba','seleccion_ref','ref:prueba',
        'retencion_seleccion',v_ref,'orden_seleccionado',5,
        'recibo_ref','ref:prueba','auditoria_ref','ref:prueba',
        'evento_ref','ref:prueba','confirmada_en','2026-10-16T00:37:15Z',
        'procedencia',pg_catalog.jsonb_build_object(
            'autoridad_ref','ref:prueba','respuesta_ref','ref:prueba',
            'contrato_version',1,'fuente',v_ref,'evidencia',v_evidencia));
    v_legado := vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(v_recibo);
    IF v_legado IS NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(v_legado)
       IS DISTINCT FROM v_recibo THEN
        RAISE EXCEPTION 'CT230: canon legado alterado';
    END IF;

    v_nativo := pg_catalog.left(v_legado,pg_catalog.length(v_legado)-2) ||
        ',"sin_retencion_declarada":true}}';
    v_recibo := pg_catalog.jsonb_set(v_recibo,
        '{procedencia,sin_retencion_declarada}','true'::jsonb,true);
    IF vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(v_recibo)
       IS DISTINCT FROM v_nativo OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(v_nativo)
       IS DISTINCT FROM v_recibo THEN
        RAISE EXCEPTION 'CT230: canon nativo Go incompatible';
    END IF;
    v_politica := pg_catalog.left(v_legado,pg_catalog.length(v_legado)-2) ||
        ',"sin_retencion_declarada":true,"politica_provisional":true}}';
    v_recibo := pg_catalog.jsonb_set(v_recibo,
        '{procedencia,politica_provisional}','true'::jsonb,true);
    IF vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(v_recibo)
       IS DISTINCT FROM v_politica OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(v_politica)
       IS DISTINCT FROM v_recibo THEN
        RAISE EXCEPTION 'CT230: orden de política provisional incompatible';
    END IF;

    IF vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.replace(v_nativo,'"sin_retencion_declarada":true',
             '"sin_retencion_declarada":false')) IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.replace(v_nativo,'"sin_retencion_declarada":true',
             '"sin_retencion_declarada":"true"')) IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.replace(v_nativo,'"sin_retencion_declarada":true',
             '"sin_retencion_declarada":null')) IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.replace(v_nativo,'"sin_retencion_declarada":true',
             '"sin_retencion_declarada":{}')) IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.left(v_nativo,pg_catalog.length(v_nativo)-1) || ',"extra":1}')
         IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.replace(v_politica,'"politica_provisional":true',
             '"politica_provisional":false')) IS NOT NULL OR
       vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
         pg_catalog.left(v_legado,pg_catalog.length(v_legado)-2) ||
             ',"politica_provisional":true}}') IS NOT NULL THEN
        RAISE EXCEPTION 'CT230: marca malformada, extra o fuera de contrato admitida';
    END IF;
END $ct230_canon$;

-- Las confirmaciones previas deben conservar sus bytes y su prueba canónica.
DO $ct230_historia$
DECLARE
    v_total integer;
    v_correctas integer;
    v_artefacto jsonb;
    v_mutado jsonb;
BEGIN
    SELECT pg_catalog.count(*), pg_catalog.count(*) FILTER (WHERE
       recibo_json->'procedencia' ? 'sin_retencion_declarada' IS FALSE AND
       artefacto_canonico::jsonb->'evidencia' ? 'sin_retencion_declarada' IS FALSE AND
       vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
           artefacto_canonico::jsonb,artefacto_canonico,recibo_json,
           vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(recibo_json)))
      INTO v_total,v_correctas
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
     WHERE situacion='confirmada';
    IF v_total = 0 OR v_correctas IS DISTINCT FROM v_total THEN
        RAISE EXCEPTION 'CT230: confirmaciones históricas alteradas';
    END IF;

    SELECT artefacto_canonico::jsonb INTO v_artefacto
      FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
     WHERE situacion='confirmada' ORDER BY creada_en LIMIT 1;
    -- Solo se ejercita el serializador: este material no se confirma ni se firma.
    v_mutado := pg_catalog.jsonb_set(pg_catalog.jsonb_set(v_artefacto,
        '{recibo,procedencia,sin_retencion_declarada}','true'::jsonb,true),
        '{evidencia,sin_retencion_declarada}','true'::jsonb,true);
    IF vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
           v_mutado,false) IS NULL OR
       pg_catalog.length(vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
           v_mutado,false)) -
       pg_catalog.length(pg_catalog.replace(
           vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(v_mutado,false),
           '"sin_retencion_declarada":true','')) IS DISTINCT FROM
           2*pg_catalog.length('"sin_retencion_declarada":true') OR
       vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
           pg_catalog.jsonb_set(v_mutado,'{evidencia,sin_retencion_declarada}',
               'false'::jsonb),false) IS NOT NULL OR
       vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(
           pg_catalog.jsonb_set(v_mutado,'{recibo,procedencia,politica_provisional}',
               'null'::jsonb),false) IS NOT NULL OR
       vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
           pg_catalog.jsonb_set(v_mutado,'{evidencia,sin_retencion_declarada}',
               'false'::jsonb),
           vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(v_mutado,false),
           v_mutado->'recibo',
           vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
               v_mutado->'recibo')) THEN
        RAISE EXCEPTION 'CT230: evidencias divergentes o marca extra admitida';
    END IF;
END $ct230_historia$;

-- Vector sintético obtenido de materialReciboLlamamientoBolsa (Go) con
-- la clave de prueba del paquete: solo material, sin firma ni recibo de producto.
DO $ct230_vector_go$
DECLARE
    v_vector jsonb := $go_vector_ct230${"comando":{"contexto":{"datos":{"operacion_ref":"operacion:solicitar:llamamiento","organizacion_ref":"organizacion:diputacion","expediente_ref":"expediente:temporal","version_expediente":5,"correlacion_ref":"correlacion:temporal","contrato_version":1,"autoridad_solicitante":"autoridad:contratacion-temporal","autorizacion":{"referencia":"autorizacion:contratacion","version":3,"huella_sha256":"1111111111111111111111111111111111111111111111111111111111111111"},"accion":{"referencia":"accion:solicitar:llamamiento","version":3,"huella_sha256":"4444444444444444444444444444444444444444444444444444444444444444"},"recurso":{"referencia":"orden:instantanea","version":3,"huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},"finalidad":{"referencia":"finalidad:cobertura","version":3,"huella_sha256":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},"solicitada_en":"2026-07-23T08:00:00Z","valida_hasta":"2026-07-23T08:10:00Z"},"clave_verificacion_ref":"vec.contratacion-temporal.integracion-bolsa-peticion/v1","sello_hmac":"hmac-sha256:vec.contratacion-temporal.integracion-bolsa-peticion/v1:185d14c3342dda6ebf2405b5f61ee6a118ee4ac3f3801d95159a8eaf084a54e7"},"necesidad":{"referencia":"necesidad:temporal","version":3,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"bolsa":{"referencia":"bolsa:vigente","version":3,"huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"orden":{"referencia":"orden:instantanea","version":3,"huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},"politica":{"referencia":"politica:llamamiento","version":3,"huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},"total_posiciones_orden":82,"maxima_posicion_evaluable":50,"huella_recibo_orden":"f0a39f903beb3c2dedc8c239f637ac833f902fafe3041143d5751b765fdc467a"},"recibo":{"operacion_ref":"operacion:solicitar:llamamiento","organizacion_ref":"organizacion:diputacion","expediente_ref":"expediente:temporal","version_expediente":5,"correlacion_ref":"correlacion:temporal","necesidad":{"referencia":"necesidad:temporal","version":3,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"bolsa":{"referencia":"bolsa:vigente","version":3,"huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"orden":{"referencia":"orden:instantanea","version":3,"huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},"politica":{"referencia":"politica:llamamiento","version":3,"huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},"resultado":{"referencia":"resultado:propuesta","version":3,"huella_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},"propuesta_generada":true,"propuesta":{"referencia":"propuesta:primera","version":3,"huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},"accion_evento":{"referencia":"accion:registrar:evento","version":3,"huella_sha256":"5555555555555555555555555555555555555555555555555555555555555555"},"llamamiento_ref":"llamamiento:primero","seleccion_ref":"hmac-sha256:vec.contratacion-temporal.seleccion/v1:9999999999999999999999999999999999999999999999999999999999999999","retencion_seleccion":{"referencia":"","version":0,"huella_sha256":""},"orden_seleccionado":4,"recibo_ref":"recibo:llamamiento","auditoria_ref":"auditoria:llamamiento","evento_ref":"evento:llamamiento","confirmada_en":"2026-07-23T08:01:00Z","procedencia":{"autoridad_ref":"autoridad:bolsa","respuesta_ref":"respuesta:bolsa","contrato_version":1,"fuente":{"referencia":"fuente:bolsa","version":3,"huella_sha256":"8888888888888888888888888888888888888888888888888888888888888888"},"evidencia":{"evidencia_ref":"evidencia:bolsa","clave_verificacion_ref":"vec.contratacion-temporal.integracion-bolsa-respuesta/v1","sello_hmac":"","emitida_en":"2026-07-23T08:02:00Z","valida_hasta":"2026-07-23T08:08:00Z","retener_hasta":"0001-01-01T00:00:00Z"},"sin_retencion_declarada":true}}}$go_vector_ct230$::jsonb;
    v_politica jsonb;
BEGIN
    IF (vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
            v_vector))[2] IS DISTINCT FROM '5dca2b66f8aa33a07a8cd93620fd14e4f3f29541de88f1a06222390fcd1a9a14' THEN
        RAISE EXCEPTION 'CT230: huella Go nativa con política false incompatible';
    END IF;
    v_politica := pg_catalog.jsonb_set(v_vector,
        '{recibo,procedencia,politica_provisional}','true'::jsonb,true);
    IF (vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
            v_politica))[2] IS DISTINCT FROM 'f137e6df5fd1ee67d4ce86ff459741e55d562c4e20915b84fdf0bdf35debc759' THEN
        RAISE EXCEPTION 'CT230: huella Go nativa con política true incompatible';
    END IF;
END $ct230_vector_go$;
ROLLBACK;
