\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $carrera$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; huella text; mh text; rh text;
 recurso text; inicio timestamptz:=clock_timestamp(); vence timestamptz:=clock_timestamp()+interval '800 milliseconds';
BEGIN
 huella:=encode(sha256(convert_to(concat_ws(E'\n',
  'vec.personal.catalogo-registro-empleado.entrada.v1','org:sintetico','regimen','reg:carrera',
  '1','1','Régimen carrera','2020-01-01','','acto:sintetico:carrera'),'UTF8')),'hex');
 m:=jsonb_build_object('esquema','vec.personal.catalogo-registro-empleado.v1',
  'operacion','publicar','organismo_ref','org:sintetico','tipo','regimen','ref','reg:carrera',
  'version',1,'revision',1,'denominacion','Régimen carrera','huella_sha256',huella,
  'vigente_desde','2020-01-01','vigente_hasta',NULL,'acto_ref','acto:sintetico:carrera',
  'actor_ref','actor:sintetico:rrhh','idempotencia_ref','33333333-3333-4333-8333-333333333333');
 x:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref','actor:sintetico:rrhh',
  'perfil_activo_ref','perfil:sintetico:rrhh','persona_version',1,'perfil_version',1);
 mh:=encode(sha256(convert_to(m::text,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":"org:sintetico:regimen:reg:carrera:1","organismo_ref":"org:sintetico"},"atributos":{"material_sha256":"'||mh||'","operacion":"publicar"}}';
 rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c:=jsonb_build_object('operacion','personal.registro_empleado.catalogo.publicar',
  'audiencia_consumo','vec_personal.registro_empleado.catalogo.publicar.v1',
  'efecto_ref','org:sintetico:regimen:reg:carrera:1','huella_efecto_sha256',rh,
  'emitida_en',inicio-interval '1 second','expira_en',vence,'decision_valida_hasta',vence);
 d:=jsonb_build_object('principal_id','actor:sintetico:rrhh','perfil_activo_ref','perfil:sintetico:rrhh',
  'concedida',true,'modulo_id','personal','obligaciones','[]'::jsonb,
  'tipo_recurso','entrada_catalogo_empleado_rrhh','finalidad','gobernar_catalogo_empleado',
  'campos_permitidos','["entrada","recibo"]'::jsonb,'accion','personal.registro_empleado.catalogo.publicar',
  'recurso_ref','org:sintetico:regimen:reg:carrera:1','contexto_recurso_huella_sha256',rh,
  'decision_ref','decision:sintetica:carrera','valida_hasta',vence);
 BEGIN
  PERFORM vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(m::text,
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'x'::bytea,
   convert_to(x::text,'UTF8'),1,1,'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'capacidad caducada produjo efecto';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF clock_timestamp()-inicio<interval '900 milliseconds' THEN
   RAISE EXCEPTION 'caducidad ocurrió antes de atravesar el lock'; END IF;
 END;
END $carrera$;
ROLLBACK;
