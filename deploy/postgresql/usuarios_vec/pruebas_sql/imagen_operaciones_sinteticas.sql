\set ON_ERROR_STOP on
-- Doble V3 de forma para Usuarios 000006; no acredita COSE ni KMS.
-- p_op: 'consultar', 'recuperar' o 'guardar'. p_eleccion es {modo,paleta,icono}.
-- p_huella fuerza otra huella de petición (para probar el conflicto).
CREATE FUNCTION public.probar_imagen(p_op text,p_version bigint,p_clave text,p_eleccion jsonb,
 p_foto bytea DEFAULT NULL,p_falsa boolean DEFAULT false,p_superficie text DEFAULT 'interna_corporativa',
 p_huella text DEFAULT NULL,p_sha text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE persona constant text:='per_ABCDEFGHIJKLMNOPQRSTUV';
 perfil constant text:='prf_ABCDEFGHIJKLMNOPQRSTUV';
 accion text; segmento text; m jsonb; material text; h text; c bytea; d bytea; carga bytea; t record;
BEGIN
 accion:=CASE WHEN p_op='consultar' THEN 'vec.imagen.consultar' ELSE 'vec.imagen.actualizar' END;
 segmento:=split_part(accion,'.',3);
 IF p_op='consultar' THEN
  m:=jsonb_build_object('superficie',p_superficie,'persona_ref',persona,'perfil_ref',perfil,'accion',accion,
   'finalidad_ref','finalidad:usuarios:imagen-propia:v1','catalogo_version_ref','usuarios-imagen-v1','version_esperada',0,
   'clave_operacion','','huella_peticion','','eleccion','{"modo":"","paleta":"","icono":""}'::jsonb,'foto_sha256','');
 ELSE
  m:=jsonb_build_object('superficie',p_superficie,'persona_ref',persona,'perfil_ref',perfil,'accion',accion,
   'finalidad_ref','finalidad:usuarios:imagen-propia:v1','catalogo_version_ref','usuarios-imagen-v1','version_esperada',p_version,
   'clave_operacion',p_clave,'huella_peticion','','eleccion',p_eleccion,
   'foto_sha256',coalesce(p_sha,CASE WHEN p_foto IS NULL THEN '' ELSE encode(sha256(p_foto),'hex') END));
  m:=jsonb_set(m,'{huella_peticion}',to_jsonb(coalesce(p_huella,vec_usuarios.huella_peticion_imagen(m))));
 END IF;
 material:=m::text;
 h:=vec_usuarios.huella_contexto_imagen(material);
 c:=convert_to(jsonb_build_object('suite','VEC-AD-3-COSE-EDDSA-1','audiencia_consumo','vec_usuarios.imagen.'||segmento||'.'||p_superficie||'.v1',
  'operacion',accion,'efecto_ref',persona,'huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('principal_id',persona,'perfil_activo_ref',perfil,
  'decision_ref','dec_'||replace(gen_random_uuid()::text,'-',''),
  'accion',accion,'modulo_id','usuarios','tipo_recurso','imagen_persona',
  'finalidad','finalidad:usuarios:imagen-propia:v1','recurso_ref',persona,'contexto_recurso_huella_sha256',h,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie',p_superficie),
  'concedida',true,'campos_permitidos','["foto","icono","modo","paleta","version"]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8');
 carga:=CASE WHEN p_falsa THEN convert_to('falsa','UTF8') ELSE convert_to(gen_random_uuid()::text,'UTF8') END;
 IF p_op='consultar' THEN
  SELECT * INTO t FROM vec_usuarios.consultar_imagen_propia_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
  RETURN t.datos||jsonb_build_object('foto_hex',encode(t.foto,'hex'));
 ELSIF p_op='recuperar' THEN
  RETURN vec_usuarios.recuperar_imagen_operacion_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 END IF;
 RETURN vec_usuarios.guardar_imagen_propia_v1(material,p_foto,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
END $f$;
REVOKE ALL ON FUNCTION public.probar_imagen(text,bigint,text,jsonb,bytea,boolean,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_imagen(text,bigint,text,jsonb,bytea,boolean,text,text,text) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
