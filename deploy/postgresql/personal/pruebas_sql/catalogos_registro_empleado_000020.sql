\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $casos$
DECLARE
 m jsonb; m_error jsonb; filtro jsonb; c jsonb; c_error jsonb; d jsonb; d_error jsonb; x jsonb; r jsonb; r2 jsonb; q jsonb;
 huella text; mh text; rh text; error_mh text; error_rh text; efecto text; recurso text;
 cap_desde timestamptz:=clock_timestamp()-interval '1 minute';
 cap_hasta timestamptz:=clock_timestamp()+interval '5 minutes';
BEGIN
 huella:=encode(sha256(convert_to(concat_ws(E'\n',
  'vec.personal.catalogo-registro-empleado.entrada.v1','org:sintetico','regimen','reg:sintetico',
  '1','1','Régimen sintético','2020-01-01','','acto:sintetico:publicar'),'UTF8')),'hex');
 m:=jsonb_build_object('esquema','vec.personal.catalogo-registro-empleado.v1',
  'operacion','publicar','organismo_ref','org:sintetico','tipo','regimen','ref','reg:sintetico',
  'version',1,'revision',1,'denominacion','Régimen sintético','huella_sha256',huella,
  'vigente_desde','2020-01-01','vigente_hasta',NULL,'acto_ref','acto:sintetico:publicar',
  'actor_ref','actor:sintetico:rrhh','idempotencia_ref','11111111-1111-4111-8111-111111111111');
 x:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref','actor:sintetico:rrhh',
  'perfil_activo_ref','perfil:sintetico:rrhh','persona_version',1,'perfil_version',1);
 efecto:='org:sintetico:regimen:reg:sintetico:1';
 mh:=encode(sha256(convert_to(m::text,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":"'||efecto||'","organismo_ref":"org:sintetico"},"atributos":{"material_sha256":"'||mh||'","operacion":"publicar"}}';
 rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c:=jsonb_build_object('operacion','personal.registro_empleado.catalogo.publicar',
  'audiencia_consumo','vec_personal.registro_empleado.catalogo.publicar.v1',
  'efecto_ref',efecto,'huella_efecto_sha256',rh,'emitida_en',cap_desde,'expira_en',cap_hasta,
  'decision_valida_hasta',cap_hasta);
 d:=jsonb_build_object('principal_id','actor:sintetico:rrhh','perfil_activo_ref','perfil:sintetico:rrhh',
  'concedida',true,'modulo_id','personal','obligaciones','[]'::jsonb,
  'tipo_recurso','entrada_catalogo_empleado_rrhh','finalidad','gobernar_catalogo_empleado',
  'campos_permitidos','["entrada","recibo"]'::jsonb,'accion','personal.registro_empleado.catalogo.publicar',
  'recurso_ref',efecto,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:sintetica:publicar',
  'valida_hasta',cap_hasta);
 m_error:=m||jsonb_build_object('huella_sha256',repeat('0',64));
 error_mh:=encode(sha256(convert_to(m_error::text,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":"'||efecto||'","organismo_ref":"org:sintetico"},"atributos":{"material_sha256":"'||error_mh||'","operacion":"publicar"}}';
 error_rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c_error:=c||jsonb_build_object('huella_efecto_sha256',error_rh);
 d_error:=d||jsonb_build_object('contexto_recurso_huella_sha256',error_rh);
 BEGIN
  PERFORM vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(
   m_error::text,convert_to(c_error::text,'UTF8'),convert_to(d_error::text,'UTF8'),'x'::bytea,
   convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'huella falsa aceptada';
 EXCEPTION WHEN SQLSTATE '23514' THEN NULL; END;
 r:=vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(m::text,
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'x'::bytea,
  convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
 IF r->'entrada'->>'estado'<>'publicada' OR r#>>'{acceso_actual,estado_replay}'<>'registrado'
    OR r#>>'{recibo,registrado_en}' IS DISTINCT FROM r#>>'{acceso_actual,registrado_en}' THEN
  RAISE EXCEPTION 'publicación/recibo inválidos %',r; END IF;
 r2:=vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(m::text,
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'x'::bytea,
  convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
 IF r2->'recibo' IS DISTINCT FROM r->'recibo'
    OR r2#>>'{acceso_actual,estado_replay}'<>'replay'
    OR r2#>>'{acceso_actual,consumo_huella_sha256}'=r#>>'{acceso_actual,consumo_huella_sha256}' THEN
  RAISE EXCEPTION 'replay incorrecto original=% replay=%',r,r2; END IF;
 filtro:=jsonb_build_object('esquema','vec.personal.catalogo-registro-empleado.consulta.v1',
  'organismo_ref','org:sintetico','tipo','regimen','estado','publicada',
  'cursor_ref',NULL,'cursor_version',NULL,'limite',10);
 efecto:='org:sintetico:regimen';
 mh:=encode(sha256(convert_to(filtro::text,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":"'||efecto||'","organismo_ref":"org:sintetico"},"atributos":{"material_sha256":"'||mh||'","operacion":"consultar"}}';
 rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c:=c||jsonb_build_object('operacion','personal.registro_empleado.catalogo.consultar',
  'audiencia_consumo','vec_personal.registro_empleado.catalogo.consultar.v1','efecto_ref',efecto,
  'huella_efecto_sha256',rh);
 d:=d||jsonb_build_object('tipo_recurso','catalogo_empleado_rrhh','finalidad','consultar_catalogo_empleado',
  'campos_permitidos','["cursor_siguiente","entradas","evidencia","organismo_ref"]'::jsonb,
  'accion','personal.registro_empleado.catalogo.consultar','recurso_ref',efecto,
  'contexto_recurso_huella_sha256',rh,'decision_ref','decision:sintetica:consultar');
 q:=vec_personal.consultar_catalogo_empleado_rrhh_v1(filtro::text,
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'x'::bytea,
  convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
 IF q->>'organismo_ref'<>'org:sintetico' OR jsonb_array_length(q->'entradas')<>1
    OR q#>>'{entradas,0,huella_sha256}'<>huella THEN RAISE EXCEPTION 'consulta inválida %',q; END IF;
 m:=m||jsonb_build_object('operacion','retirar','revision',2,
   'acto_ref','acto:sintetico:retirar','idempotencia_ref','22222222-2222-4222-8222-222222222222');
 efecto:='org:sintetico:regimen:reg:sintetico:1';
 mh:=encode(sha256(convert_to(m::text,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":"'||efecto||'","organismo_ref":"org:sintetico"},"atributos":{"material_sha256":"'||mh||'","operacion":"retirar"}}';
 rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c:=c||jsonb_build_object('operacion','personal.registro_empleado.catalogo.retirar',
  'audiencia_consumo','vec_personal.registro_empleado.catalogo.retirar.v1',
  'efecto_ref',efecto,'huella_efecto_sha256',rh);
 d:=d||jsonb_build_object('tipo_recurso','entrada_catalogo_empleado_rrhh',
  'finalidad','gobernar_catalogo_empleado','campos_permitidos','["entrada","recibo"]'::jsonb,
  'accion','personal.registro_empleado.catalogo.retirar','recurso_ref',efecto,
  'contexto_recurso_huella_sha256',rh,'decision_ref','decision:sintetica:retirar');
 r:=vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(m::text,
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'x'::bytea,
  convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
 IF r#>>'{entrada,estado}'<>'retirada' THEN RAISE EXCEPTION 'retirada falló'; END IF;
END $casos$;
COMMIT;
