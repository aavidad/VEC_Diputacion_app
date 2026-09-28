\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.ct138_material_sintetico(
    p_llamamiento text,p_comunicacion text,p_clave text,p_huella text)
RETURNS text LANGUAGE sql IMMUTABLE AS $funcion$
    SELECT jsonb_build_object(
        'ClaveIdempotencia',p_clave,'OrganizacionRef','org:ct138',
        'ExpedienteRef','exp:ct138','LlamamientoRef',p_llamamiento,
        'ComunicacionRef',p_comunicacion,'VersionComunicacionEsperada',2,
        'Respuesta','aceptacion','CorreoRef','correo:sintetico',
        'CorreoSHA256',p_huella,'RecibidaEn','2026-09-05T10:00:00Z')::text
$funcion$;
CREATE FUNCTION vec_contratacion_temporal.ct138_registrar_sintetico(
    p_material text,p_version integer,p_denegar boolean DEFAULT false)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $funcion$
DECLARE
    s jsonb; h text; d jsonb;
BEGIN
    s:=p_material::jsonb;
    h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    h:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
    d:=jsonb_build_object(
        'accion',CASE WHEN p_denegar THEN 'accion:ajena' ELSE
            'contratacion_temporal.llamamiento.respuesta.registrar' END,
        'modulo_id','contratacion_temporal',
        'tipo_recurso','respuesta_recibida_llamamiento_contratacion_temporal',
        'finalidad','gestionar_contratacion_temporal',
        'recurso_ref','exp:ct138','contexto_recurso_huella_sha256',h,
        'principal_id','actor:sintetico','perfil_activo_ref','perfil:sintetico');
    IF p_version=1 THEN
        RETURN vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(
            p_material,decode('01','hex'),convert_to(d::text,'UTF8'),decode('02','hex'),
            decode('03','hex'),1,1,decode('04','hex'),decode('05','hex'),
            decode('06','hex'),decode('07','hex'));
    END IF;
    RETURN vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(
        p_material,decode('01','hex'),convert_to(d::text,'UTF8'),decode('02','hex'),
        decode('03','hex'),1,1,decode('04','hex'),decode('05','hex'),
        decode('06','hex'),decode('07','hex'));
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.ct138_material_sintetico(text,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.ct138_registrar_sintetico(text,integer,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.ct138_material_sintetico(text,text,text,text) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.ct138_registrar_sintetico(text,integer,boolean) TO vec_contratacion_temporal_ejecutor;
RESET ROLE;
