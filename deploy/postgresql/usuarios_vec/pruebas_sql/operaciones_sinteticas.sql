\set ON_ERROR_STOP on
-- Wrapper de prueba: genera material V3 sintético coherente. No es emisor COSE.
CREATE FUNCTION public.probar_preferencias(p_operacion text,p_version bigint,p_clave text,p_valores jsonb,
 p_falsa boolean DEFAULT false,p_superficie text DEFAULT 'interna_corporativa',
 p_audiencia_override text DEFAULT NULL,p_vinculo_override text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE persona constant text:='per_ABCDEFGHIJKLMNOPQRSTUV';
 perfil constant text:='prf_ABCDEFGHIJKLMNOPQRSTUV';
 accion text; audiencia text; campos jsonb; material text; h text; c bytea; d bytea; m jsonb; carga bytea;
BEGIN
 IF p_operacion NOT IN ('get','put','rec') OR p_superficie NOT IN ('interna_corporativa','externa_personal')
 THEN RAISE EXCEPTION 'operación de prueba inválida'; END IF;
 accion:=CASE WHEN p_operacion='get' THEN 'vec.preferencias.consultar' ELSE 'vec.preferencias.actualizar' END;
 audiencia:=coalesce(p_audiencia_override,
  CASE WHEN p_operacion='get' THEN 'vec_usuarios.preferencias.consultar.'||p_superficie||'.v1'
   ELSE 'vec_usuarios.preferencias.actualizar.'||p_superficie||'.v1' END);
 campos:=CASE WHEN p_operacion='get' THEN '["catalogo","valores","version"]'::jsonb ELSE '["valores","version"]'::jsonb END;
 m:=jsonb_build_object('superficie',p_superficie,'persona_ref',persona,'perfil_ref',perfil,'accion',accion,
  'finalidad_ref','finalidad:usuarios:preferencias-propias:v1',
  'catalogo_version_ref','usuarios-preferencias-v1','version_esperada',p_version,
  'clave_operacion',coalesce(p_clave,''),'huella_peticion',
   CASE WHEN p_operacion='get' THEN '' ELSE vec_usuarios.huella_semantica_preferencias(persona,p_version,'usuarios-preferencias-v1',p_valores) END,
  'valores',p_valores);
 material:=m::text;
 h:=vec_usuarios.huella_contexto_preferencias(material);
 c:=convert_to(jsonb_build_object('suite','VEC-AD-3-COSE-EDDSA-1','audiencia_consumo',audiencia,
  'operacion',accion,'efecto_ref',persona,'huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('principal_id',persona,'perfil_activo_ref',perfil,
  'decision_ref','dec_'||replace(gen_random_uuid()::text,'-',''),
  'vinculo_autenticacion_actor',jsonb_build_object('superficie',coalesce(p_vinculo_override,p_superficie)),
  'accion',accion,'modulo_id','usuarios','tipo_recurso','preferencias_persona',
  'finalidad','finalidad:usuarios:preferencias-propias:v1','recurso_ref',persona,
  'contexto_recurso_huella_sha256',h,'concedida',true,'campos_permitidos',campos,'obligaciones','[]'::jsonb)::text,'UTF8');
 carga:=CASE WHEN p_falsa THEN convert_to('falsa','UTF8') ELSE convert_to(gen_random_uuid()::text,'UTF8') END;
 IF p_operacion='get' THEN
  RETURN vec_usuarios.consultar_preferencias_propias_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 ELSIF p_operacion='rec' THEN
  RETURN vec_usuarios.recuperar_preferencias_operacion_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 ELSE
  RETURN vec_usuarios.guardar_preferencias_propias_v1(material,p_valores,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 END IF;
END $f$;
REVOKE ALL ON FUNCTION public.probar_preferencias(text,bigint,text,jsonb,boolean,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_preferencias(text,bigint,text,jsonb,boolean,text,text,text)
 TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
