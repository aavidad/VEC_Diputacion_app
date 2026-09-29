\set ON_ERROR_STOP on
-- Doble V3 de forma para los esquemas por población de Usuarios 000009:
-- mismo contrato que probar_correos, pero cada superficie llama a las fachadas
-- de su esquema. p_esquema fuerza otro esquema para probar el cruce.
-- p_correo es la referencia del correo; p_huella fija la huella de igualdad
-- (por defecto se deriva de la referencia) y p_semantica la huella semántica.
CREATE FUNCTION public.probar_correos_poblacion(p_accion text,p_version bigint,p_clave text,p_correo text,
 p_modo text DEFAULT 'aplicar',p_valido boolean DEFAULT NULL,p_falsa boolean DEFAULT false,
 p_superficie text DEFAULT 'interna_corporativa',p_igualdad_ref text DEFAULT 'igualdad-v1',
 p_huella text DEFAULT '',p_semantica text DEFAULT '',p_esquema text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE persona constant text:='per_ABCDEFGHIJKLMNOPQRSTUV';
 perfil constant text:='prf_ABCDEFGHIJKLMNOPQRSTUV';
 audiencia text; segmento text; campos jsonb; m jsonb; material text; h text;
 c bytea; d bytea; carga bytea; sobre jsonb; reserva jsonb; r jsonb; e text;
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
 e:=coalesce(p_esquema,CASE p_superficie WHEN 'interna_corporativa' THEN 'vec_usuarios_correos_interno' ELSE 'vec_usuarios_correos_externo' END);
 m:=jsonb_build_object('superficie',p_superficie,'persona_ref',persona,'perfil_ref',perfil,'accion',p_accion,
  'finalidad_ref','finalidad:usuarios:correos-propios:v1','version_esperada',p_version,
  'clave_operacion',CASE WHEN p_accion='vec.correos.consultar' THEN '' ELSE p_clave END,
  'huellas_peticion',CASE WHEN p_accion='vec.correos.consultar' THEN '{}'::jsonb ELSE
   jsonb_build_object('activa',jsonb_build_object('clave_ref','hmac-correos-v1',
    'valor',encode(sha256(convert_to(p_accion||':'||p_clave||':'||p_version||':'||coalesce(p_correo,'')||':'||p_semantica,'UTF8')),'hex')),
    'retenidas','[]'::jsonb) END,
  'correo_ref',CASE WHEN p_accion IN ('vec.correos.anadir','vec.correos.consultar') THEN '' ELSE coalesce(p_correo,'') END);
 material:=m::text;
 EXECUTE format('SELECT %I.huella_contexto_correos($1)',e) INTO h USING material;
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
  EXECUTE format('SELECT %I.recuperar_correos_operacion_v1($1,$2,$3,$4,$4,1,1,$5,$4,$4,$4)',e) INTO r USING material,c,d,'x'::bytea,carga; RETURN r;
 ELSIF p_accion='vec.correos.consultar' THEN
  EXECUTE format('SELECT %I.consultar_correos_propios_v1($1,$2,$3,$4,$4,1,1,$5,$4,$4,$4)',e) INTO r USING material,c,d,'x'::bytea,carga; RETURN r;
 ELSIF p_accion='vec.correos.verificar' THEN
  EXECUTE format('SELECT %I.preparar_verificacion_correo_v1($1,$2,$3,$4,$4,1,1,$5,$4,$4,$4)',e) INTO r USING material,c,d,'x'::bytea,carga;
  IF r ? 'replay' THEN RETURN r; END IF;
  EXECUTE format('SELECT %I.cerrar_verificacion_correo_v1($1,$2,$3)',e) INTO r USING persona,p_clave,p_valido; RETURN r;
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
  EXECUTE format('SELECT %I.aplicar_correos_propios_v1($1,$2,$3,$4,$5,$6,$6,1,1,$7,$6,$6,$6)',e) INTO r USING material,sobre,reserva,c,d,'x'::bytea,carga; RETURN r;
 END IF;
END $f$;
REVOKE ALL ON FUNCTION public.probar_correos_poblacion(text,bigint,text,text,text,boolean,boolean,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_correos_poblacion(text,bigint,text,text,text,boolean,boolean,text,text,text,text,text) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
