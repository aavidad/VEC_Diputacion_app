\set ON_ERROR_STOP on
-- Utilidades de las pruebas de CT124 (funciones temporales de la sesión):
-- construyen las entradas igual que los adaptadores Go e invocan las fachadas
-- con dobles explícitos de la autorización. Se cargan antes de cada prueba.
CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;
CREATE FUNCTION pg_temp.instante() RETURNS text LANGUAGE sql AS $$
 SELECT to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') $$;
CREATE FUNCTION pg_temp.hasta() RETURNS text LANGUAGE sql AS $$
 SELECT to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') $$;

-- Entrada común de una operación de seguimiento igual que el adaptador Go.
CREATE FUNCTION pg_temp.entrada(p_op text, p_dominio text, p_accion text, p_tipo text, p_finalidad text, p_exp text, p_version numeric,
  m_extra jsonb, atr_extra jsonb, act_extra jsonb, p_estado_destino text, p_sufijo text, p_accion_decision text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; inst text:=pg_temp.instante(); m jsonb; act jsonb; amb jsonb; atr jsonb; h text; dec jsonb; amb_hmac text; hue_hmac text; org text;
BEGIN
 SELECT v.agregado_json INTO STRICT ag FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=p_exp AND v.version=p_version;
 org:=ag->>'organizacion_ref';
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.'||p_dominio||'.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.'||p_dominio||'.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo||m_extra::text,'UTF8')),'hex');
 m:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'version_esperada',p_version,'actor_ref','per_ct124_actor','perfil_ref','prf_ct124')||m_extra;
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,'accion_clave',p_accion,
   'actor_ref','per_ct124_actor','unidad_ref',ag#>>'{asignacion,unidad_ref}','recibo_ref','recibo:ct124:'||p_sufijo,'realizada_en',inst,
   'fase_origen','nombramiento','fase_destino','nombramiento','estado_origen','en_curso','estado_destino',p_estado_destino)||act_extra;
 amb:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'fase_previa','nombramiento','estado_previo','en_curso');
 atr:=jsonb_build_object('version_expediente',p_version::text)||atr_extra||jsonb_build_object(
   'politica_ref','vec.contratacion_temporal.reglas','politica_version','1','politica_huella_sha256',repeat('d',64),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac);
 h:=vec_contratacion_temporal.huella_contexto_go_ct115(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct124:'||p_sufijo,'accion',coalesce(p_accion_decision,p_accion),'modulo_id','contratacion_temporal',
   'tipo_recurso',p_tipo,'finalidad',p_finalidad,'recurso_ref',p_exp,'principal_id','per_ct124_actor','perfil_activo_ref','prf_ct124','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object('operacion',p_op,'material',m,
   'referencias',jsonb_build_object('reserva_ref','reserva:ct124:'||p_sufijo,'recibo_ref','recibo:ct124:'||p_sufijo,'evento_ref','evento:ct124:'||p_sufijo),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,'expediente_anterior',ag,
   'expediente_siguiente',ag||jsonb_build_object('version',p_version+1,'actualizado_en',inst,'actuaciones',(ag->'actuaciones')||jsonb_build_array(act))
     ||CASE WHEN p_estado_destino<>'en_curso' THEN jsonb_build_object('estado_actual',p_estado_destino) ELSE '{}'::jsonb END,
   'actuacion',act,'politica',jsonb_build_object('definicion_ref','vec.contratacion_temporal.reglas','definicion_version',1,'definicion_huella_sha256',repeat('d',64),
     'accion',p_accion,'finalidad',p_finalidad,'evaluada_en',inst,'valida_hasta',pg_temp.hasta()),
   'autorizacion',jsonb_build_object('accion',p_accion,'finalidad',p_finalidad,'recurso_ref',p_exp,'principal_id','per_ct124_actor',
     'perfil_activo_ref','prf_ct124','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),
     'persona_version',1,'perfil_version',1,'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),'decision_ref','decision:ct124:'||p_sufijo,
     'contexto_recurso_huella_sha256',h),
   'instante_efecto',inst,'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),'_decision',dec);
END $$;

CREATE FUNCTION pg_temp.entrada_ginpix(p_exp text, p_version numeric, p_numero text, p_fecha text, p_sufijo text, p_inc text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT pg_temp.entrada('confirmar_ginpix','confirmacion-ginpix','contratacion_temporal.ginpix.confirmar','confirmacion_ginpix_contratacion_temporal',
   'confirmar_ginpix_contratacion_temporal',p_exp,p_version,
   jsonb_build_object('ginpix_numero',p_numero,'ginpix_confirmada_en',p_fecha,'observaciones',''),
   jsonb_build_object('ginpix_numero',p_numero,'ginpix_confirmada_en',p_fecha,'observaciones_huella_sha256',encode(sha256(convert_to('','UTF8')),'hex'),
     'incorporacion_ref',p_inc),
   jsonb_build_object('documentos_ref',jsonb_build_array('ginpix:'||p_numero)),'en_curso',p_sufijo)
   ||jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-confirmacion-ginpix.v1') $$;
CREATE FUNCTION pg_temp.entrada_cese(p_exp text, p_version numeric, p_sufijo text, p_inc text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT pg_temp.entrada('registrar_cese','cese','contratacion_temporal.seguimiento.cesar','cese_contratacion_temporal',
   'registrar_cese_contratacion_temporal',p_exp,p_version,
   jsonb_build_object('causa_clave','fin_sustitucion','fecha_efecto','2027-02-15','justificante_tipo','comunicacion_reincorporacion',
     'justificante_ref','documento:ct124:justificante','justificante_sha256',repeat('e',64),'observaciones',''),
   jsonb_build_object('causa_clave','fin_sustitucion','fecha_efecto','2027-02-15','justificante_tipo','comunicacion_reincorporacion',
     'justificante_ref','documento:ct124:justificante','justificante_sha256',repeat('e',64),
     'observaciones_huella_sha256',encode(sha256(convert_to('','UTF8')),'hex'),'incorporacion_ref',p_inc),
   jsonb_build_object('documentos_ref',jsonb_build_array('documento:ct124:justificante')),'en_curso',p_sufijo)
   ||jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-cese.v1') $$;
CREATE FUNCTION pg_temp.entrada_cierre(p_exp text, p_version numeric, p_numero text, p_fecha text, p_sufijo text, p_cese text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT pg_temp.entrada('cerrar_expediente','cierre-expediente','contratacion_temporal.expediente.cerrar','cierre_expediente_contratacion_temporal',
   'cerrar_expediente_tras_cese',p_exp,p_version,
   jsonb_build_object('condiciones',jsonb_build_array('cese_registrado','ginpix_confirmado'),'ginpix_numero',p_numero,'ginpix_confirmada_en',p_fecha,'observaciones',''),
   jsonb_build_object('cese_recibo_ref',p_cese,'condiciones','cese_registrado,ginpix_confirmado','ginpix_numero',p_numero,'ginpix_confirmada_en',p_fecha,
     'observaciones_huella_sha256',encode(sha256(convert_to('','UTF8')),'hex')),
   jsonb_build_object('documentos_ref',jsonb_build_array('ginpix:'||p_numero)),'completado',p_sufijo)
   ||jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-cierre-expediente.v1') $$;

CREATE FUNCTION pg_temp.confirmar(p_funcion text, e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
 EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)',p_funcion) INTO r
   USING e-'_decision','\x6361706163696461640a'::bytea,convert_to((e->'_decision')::text,'UTF8'),'\x6d6f7469766f'::bytea,'\x01'::bytea,1::numeric,1::numeric,
     '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea;
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.preparar(p_funcion text, p_esquema text, e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
 EXECUTE format('SELECT vec_contratacion_temporal.%I($1)',p_funcion) INTO r USING jsonb_build_object('esquema',p_esquema,'operacion',e->>'operacion',
  'material',e->'material','sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',e->>'ambito_idempotencia_hmac',
  'generacion',1,'huella_peticion_hmac',e->>'huella_peticion_hmac'),'retenidos','[]'::jsonb),'referencias_candidatas',e->'referencias');
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;

CREATE FUNCTION pg_temp.codigo_consulta(o text, e text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN PERFORM vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(o, e); RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE; END $f$;
-- Se evalúa como el creador de la sesión: el ejecutor no invoca la huella de CT115.
CREATE FUNCTION pg_temp.decision_centro(p_accion text, p_recurso text, a jsonb, p_org text, p_material text, p_sufijo text) RETURNS bytea LANGUAGE sql SECURITY DEFINER AS $$
 SELECT convert_to(jsonb_build_object('decision_ref','decision:ct124c:'||p_sufijo,'accion',p_accion,'modulo_id','contratacion_temporal',
   'tipo_recurso','incorporacion_centro_contratacion_temporal','finalidad','gestionar_peticion_centro','recurso_ref',p_recurso,
   'principal_id',a->>'actor_ref','perfil_activo_ref',a->>'perfil_ref',
   'contexto_recurso_huella_sha256',vec_contratacion_temporal.huella_contexto_go_ct115(
     jsonb_build_object('centro_ref',a->>'centro_ref','organizacion_ref',p_org),
     jsonb_build_object('material_sha256',encode(sha256(convert_to(p_material,'UTF8')),'hex'))))::text,'UTF8') $$;
CREATE FUNCTION pg_temp.material_centro(a jsonb, p_org text, p_pet text, p_exp text, p_clave text, p_fecha text, p_modalidad text) RETURNS text LANGUAGE sql AS $$
 SELECT jsonb_build_object('operacion','confirmar','clave_idempotencia',p_clave,'organizacion_ref',p_org,'actor',a,'peticion_ref',p_pet,
   'expediente_ref',p_exp,'fecha_incorporacion',p_fecha,
   'documento',jsonb_build_object('tipo','toma_posesion','referencia','registro:centro-520:2027/15','sha256',repeat('f',64)),
   'regla',jsonb_build_object('referencia','vec.contratacion_temporal.reglas:1:c21.acreditacion_incorporacion','huella_sha256',repeat('a',64),
     'modalidad_clave',p_modalidad))::text $$;
CREATE FUNCTION pg_temp.centro(p_funcion text, p_material text, p_decision bytea) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
 EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)',p_funcion) INTO r
   USING p_material,'\x63'::bytea,p_decision,'\x6d'::bytea,'\x01'::bytea,1::numeric,1::numeric,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea;
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime',pg_my_temp_schema()::regnamespace); END $$;
