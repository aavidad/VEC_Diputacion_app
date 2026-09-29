\set ON_ERROR_STOP on
-- Doble V3 de forma para Aspirantes 000001: construye material, capacidad y
-- decisión como lo haría Go y llama a las fachadas con los privilegios del
-- LOGIN que ejecuta la prueba (SECURITY INVOKER). No acredita COSE ni KMS.
CREATE FUNCTION public.probar_ficha(p_accion text,p_version bigint,p_clave text,p_indice text,
 p_extra jsonb DEFAULT NULL,p_falsa boolean DEFAULT false,p_semantica text DEFAULT '',
 p_persona text DEFAULT 'per_ABCDEFGHIJKLMNOPQRSTUV',p_audiencia text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE perfil constant text:='prf_ABCDEFGHIJKLMNOPQRSTUV';
 segmento text; campos jsonb; m jsonb; material text; h text; c bytea; d bytea; carga bytea; r jsonb;
BEGIN
 SELECT x.segmento,x.campos INTO segmento,campos FROM (VALUES
 ('vec.aspirantes.ficha.consultar','consultar','["codigo_postal","documento","domicilio","movil","nombre","primer_apellido","segundo_apellido","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.alta','alta','["codigo_postal","documento","domicilio","movil","nombre","primer_apellido","segundo_apellido","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.rectificar','rectificar','["codigo_postal","domicilio","movil","telefono","version"]'::jsonb)
 ) x(accion,segmento,campos) WHERE x.accion=p_accion;
 m:=jsonb_build_object('superficie','externa_personal','persona_ref',p_persona,'perfil_ref',perfil,'accion',p_accion,
  'finalidad_ref','finalidad:aspirantes:ficha-propia:v1','version_esperada',p_version,
  'clave_operacion',CASE WHEN segmento='consultar' THEN '' ELSE p_clave END,
  'huellas_peticion',CASE WHEN segmento='consultar' THEN '{}'::jsonb ELSE
   jsonb_build_object('activa',jsonb_build_object('clave_ref','hmac-aspirantes-v1',
    'valor',encode(sha256(convert_to(p_accion||':'||p_clave||':'||p_version||':'||coalesce(p_extra::text,'')||':'||p_semantica,'UTF8')),'hex')),
    'retenidas','[]'::jsonb) END,
  'indice_documento',jsonb_build_object('clave_ref','indice-v1','valor',p_indice));
 material:=m::text;
 -- Misma huella canónica que Go; el ejecutor no puede llamar a la auxiliar.
 h:=encode(sha256(convert_to('{"ambitos":{"persona_ref":'||to_jsonb(p_persona)::text||'},"atributos":{"material_sha256":"'||
   encode(sha256(convert_to(material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 c:=convert_to(jsonb_build_object('suite','VEC-AD-3-COSE-EDDSA-1',
  'audiencia_consumo',coalesce(p_audiencia,'vec_aspirantes.ficha.'||segmento||'.externa_personal.v1'),
  'operacion',p_accion,'efecto_ref',p_persona,'huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('principal_id',p_persona,'perfil_activo_ref',perfil,
  'decision_ref','dec_'||replace(gen_random_uuid()::text,'-',''),
  'accion',p_accion,'modulo_id','aspirantes','tipo_recurso','ficha_aspirante_propia',
  'finalidad','finalidad:aspirantes:ficha-propia:v1','recurso_ref',p_persona,'contexto_recurso_huella_sha256',h,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'),
  'concedida',true,'campos_permitidos',campos,'obligaciones','[]'::jsonb)::text,'UTF8');
 carga:=CASE WHEN p_falsa THEN convert_to('falsa','UTF8') ELSE convert_to(gen_random_uuid()::text,'UTF8') END;
 IF segmento='consultar' THEN
  RETURN vec_aspirantes.consultar_ficha_propia_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 ELSIF segmento='alta' THEN
  RETURN vec_aspirantes.alta_ficha_propia_v1(material,p_extra,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 END IF;
 r:=vec_aspirantes.preparar_rectificacion_ficha_v1(material,c,d,'x'::bytea,'x'::bytea,1,1,carga,'x'::bytea,'x'::bytea,'x'::bytea);
 IF r ? 'replay' OR p_extra IS NULL THEN RETURN r; END IF;
 RETURN vec_aspirantes.aplicar_rectificacion_ficha_v1(p_extra);
END $f$;
REVOKE ALL ON FUNCTION public.probar_ficha(text,bigint,text,text,jsonb,boolean,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_ficha(text,bigint,text,text,jsonb,boolean,text,text,text) TO vec_aspirantes_ejecutor_externo;

-- Ficha sintética de alta: Lucía Fernández Moreno, con teléfono. Los sobres
-- son bytes de forma válida; el contenido cifrado no se interpreta en SQL.
CREATE FUNCTION public.ficha_prueba(p_asp text,p_doc text,p_indice text)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('aspirante_ref',p_asp,'catalogo_ref','vec.aspirantes.datos_personales:1',
  'documento',jsonb_build_object('documento_ref',p_doc,'tipo','dni','pais','ES','clave_ref','cifrado-v1',
   'nonce_hex',repeat('aa',12),'cifrado_hex',repeat('bb',25),'indice',jsonb_build_object('clave_ref','indice-v1','valor',p_indice)),
  'valores',jsonb_build_array(
   jsonb_build_object('campo','nombre','version',1,'origen','certificado','clave_ref','cifrado-v1','nonce_hex',repeat('01',12),'cifrado_hex',repeat('11',22)),
   jsonb_build_object('campo','primer_apellido','version',1,'origen','certificado','clave_ref','cifrado-v1','nonce_hex',repeat('02',12),'cifrado_hex',repeat('12',26)),
   jsonb_build_object('campo','segundo_apellido','version',1,'origen','certificado','clave_ref','cifrado-v1','nonce_hex',repeat('03',12),'cifrado_hex',repeat('13',22)),
   jsonb_build_object('campo','telefono','version',1,'origen','titular','clave_ref','cifrado-v1','nonce_hex',repeat('04',12),'cifrado_hex',repeat('14',25))))
$f$;
GRANT EXECUTE ON FUNCTION public.ficha_prueba(text,text,text) TO vec_aspirantes_ejecutor_externo;

CREATE FUNCTION public.cambio_prueba(p_version bigint,p_motivo text,p_campo text,p_retirar boolean DEFAULT false)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('catalogo_ref','vec.aspirantes.datos_personales:1','motivo',p_motivo,'valores',jsonb_build_array(
  CASE WHEN p_retirar THEN jsonb_build_object('campo',p_campo,'version',p_version,'origen','titular','estado','retirado',
    'clave_ref',NULL,'nonce_hex',NULL,'cifrado_hex',NULL)
  ELSE jsonb_build_object('campo',p_campo,'version',p_version,'origen','titular','estado','presente',
    'clave_ref','cifrado-v1','nonce_hex',repeat('05',12),'cifrado_hex',repeat('15',25)) END))
$f$;
GRANT EXECUTE ON FUNCTION public.cambio_prueba(bigint,text,text,boolean) TO vec_aspirantes_ejecutor_externo;
