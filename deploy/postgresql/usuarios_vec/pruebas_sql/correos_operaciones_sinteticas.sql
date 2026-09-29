\set ON_ERROR_STOP on
-- Doble V3 de forma para Usuarios 000004; no acredita COSE, KMS ni SMTP.
-- p_correo es la referencia del correo; p_huella fija la huella de igualdad
-- (por defecto se deriva de la referencia) y p_semantica la huella semántica.
CREATE FUNCTION public.probar_correos(p_accion text,p_version bigint,p_clave text,p_correo text,
 p_modo text DEFAULT 'aplicar',p_valido boolean DEFAULT NULL,p_falsa boolean DEFAULT false,
 p_superficie text DEFAULT 'interna_corporativa',p_igualdad_ref text DEFAULT 'igualdad-v1',
 p_huella text DEFAULT '',p_semantica text DEFAULT '')
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE persona constant text:='per_ABCDEFGHIJKLMNOPQRSTUV';
 perfil constant text:='prf_ABCDEFGHIJKLMNOPQRSTUV';
 audiencia text; segmento text; campos jsonb; m jsonb; material text; h text;
 c bytea; d bytea; carga bytea; sobre jsonb; reserva jsonb; r jsonb;
BEGIN
 SELECT x.segmento,x.campos INTO segmento,campos FROM (VALUES
 ('vec.correos.consultar','consultar','["activo","correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.anadir','anadir','["correo_ref","direccion","estado","version"]'::jsonb),
 ('vec.correos.reenviar','reenviar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.verificar','verificar','["correo_ref","estado","version"]'::jsonb),
 ('vec.correos.activar','activar','["activo","correo_ref","version"]'::jsonb),
 ('vec.correos.retirar','retirar','["activo","correo_ref","estado","version"]'::jsonb)
 ) x(accion,segmento,campos) WHERE x.accion=p_accion;
 IF segmento IS NULL OR p_superficie NOT IN ('interna_corporativa','externa_personal')
 THEN RAISE EXCEPTION 'acción o superficie de prueba inválida'; END IF;
 audiencia:='vec_usuarios.correos.'||segmento||'.'||p_superficie||'.v1';
 m:=jsonb_build_object('superficie',p_superficie,'persona_ref',persona,'perfil_ref',perfil,'accion',p_accion,
  'finalidad_ref','finalidad:usuarios:correos-propios:v1','version_esperada',p_version,
  'clave_operacion',CASE WHEN p_accion='vec.correos.consultar' THEN '' ELSE p_clave END,
  'huellas_peticion',CASE WHEN p_accion='vec.correos.consultar' THEN '{}'::jsonb ELSE
   jsonb_build_object('activa',jsonb_build_object('clave_ref','hmac-correos-v1',
    'valor',encode(sha256(convert_to(p_accion||':'||p_clave||':'||p_version||':'||coalesce(p_correo,'')||':'||p_semantica,'UTF8')),'hex')),
    'retenidas','[]'::jsonb) END,
  'correo_ref',CASE WHEN p_accion IN ('vec.correos.anadir','vec.correos.consultar') THEN '' ELSE coalesce(p_correo,'') END);
 material:=m::text;
 h:=vec_usuarios.huella_contexto_correos(material);
 c:=convert_to(jsonb_build_object('suite','VEC-AD-3-COSE-EDDSA-1','audiencia_consumo',audiencia,
  'operacion',p_accion,'efecto_ref',persona,'huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('principal_id',persona,'perfil_activo_ref',perfil,
  'decision_ref','dec_'||replace(gen_random_uuid()::text,'-',''),
  'accion',p_accion,'modulo_id','usuarios','tipo_recurso','correos_persona',
  'finalidad','finalidad:usuarios:correos-propios:v1','recurso_ref',persona,'contexto_recurso_huella_sha256',h,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie',p_superficie),
  'concedida',true,'campos_permitidos',campos,'obligaciones','[]'::jsonb)::text,'UTF8');
 carga:=CASE WHEN p_falsa THEN convert_to('falsa','UTF8') ELSE convert_to(gen_random_uuid()::text,'UTF8') END;
 IF p_modo='recuperar' THEN
  RETURN vec_usuarios.recuperar_correos_operacion_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 ELSIF p_accion='vec.correos.consultar' THEN
  RETURN vec_usuarios.consultar_correos_propios_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 ELSIF p_accion='vec.correos.verificar' THEN
  r:=vec_usuarios.preparar_verificacion_correo_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
  IF r ? 'replay' THEN RETURN r; END IF;
  RETURN vec_usuarios.cerrar_verificacion_correo_v1(persona,p_clave,p_valido);
 ELSE
  IF p_accion='vec.correos.anadir' THEN
   sobre:=jsonb_build_object('correo_ref',p_correo,'version',p_version+1,'clave_ref','cifrado-v1',
    'clave_igualdad_ref',p_igualdad_ref,
    'nonce_hex',repeat('aa',12),'cifrado_hex',repeat('bb',32),
    'huella_igualdad_hex',encode(sha256(convert_to(coalesce(nullif(p_huella,''),p_correo),'UTF8')),'hex'));
  END IF;
  IF p_accion IN ('vec.correos.anadir','vec.correos.reenviar') THEN
   reserva:=jsonb_build_object('desafio_ref','desafio:'||replace(gen_random_uuid()::text,'-',''),
    'huella_codigo_hex',repeat('dd',32),'clave_ref','codigo-v1',
    'vence_utc',to_char((date_trunc('microseconds',clock_timestamp())+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
  END IF;
  RETURN vec_usuarios.aplicar_correos_propios_v1(material,sobre,reserva,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 END IF;
END $f$;
REVOKE ALL ON FUNCTION public.probar_correos(text,bigint,text,text,text,boolean,boolean,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_correos(text,bigint,text,text,text,boolean,boolean,text,text,text,text) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
