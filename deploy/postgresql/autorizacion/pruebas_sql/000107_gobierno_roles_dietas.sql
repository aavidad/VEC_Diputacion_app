\set ON_ERROR_STOP on
-- Prueba focal sobre base desechable. Solo sustituye la lectura del catálogo;
-- el canon y el validador AUT107 son reales. No simula decisiones ni recibos.
-- El ROLLBACK retira datos y dobles de lectura. No usar contra una aplicación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE aut107_catalogo (documento jsonb NOT NULL);
GRANT SELECT ON pg_temp.aut107_catalogo TO vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)
RETURNS bytea LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT convert_to(documento::text,'UTF8') FROM pg_temp.aut107_catalogo
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_catalogo_acciones_administracion_v2(text,integer,text)
RETURNS TABLE(catalogo_canon bytea,paquete_canon bytea)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT convert_to(documento::text,'UTF8'),NULL::bytea FROM pg_temp.aut107_catalogo
$f$;
INSERT INTO vec_autorizacion.cabeza_catalogo_acciones_admin_v1 VALUES
 ('catalogo:aut107:contrato',1,repeat('a',64),'caa_'||repeat('a',24),clock_timestamp());
DO $test$
DECLARE r record;g jsonb;c jsonb;p jsonb;cbase jsonb;pbase jsonb;e jsonb;selecciones jsonb;
 entrada jsonb;resultado jsonb;i integer;nombre text;rechazada boolean;desde text;hasta text;
 vencida text;
BEGIN
 desde:=to_char(clock_timestamp()-interval '1 day','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 hasta:=to_char(clock_timestamp()+interval '89 days','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 vencida:=to_char(clock_timestamp()-interval '1 hour','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 FOR r IN SELECT * FROM (VALUES
  ('persona_dietas_comisiones','[{"accion":"personal.relacion.propia.consultar_dietas","modulo_id":"personal","tipo_recurso":"relacion_empleado_dietas","finalidades":["preparar_borrador_dietas"],"garantia_minima":"alto","campos_permitidos":["desde","empleado_ref","estado","fuente_ref","fuente_version","hasta","persona_ref","procedencia_acto_ref","relacion_ref","unidad_ref","version"]},{"accion":"dietas.borrador.propio.crear","modulo_id":"dietas","tipo_recurso":"comision_borrador","finalidades":["crear_borrador_propio"],"garantia_minima":"alto","campos_permitidos":["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]},{"accion":"dietas.borrador.propio.consultar","modulo_id":"dietas","tipo_recurso":"comision_borrador","finalidades":["consultar_borrador_propio"],"garantia_minima":"alto","campos_permitidos":["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]}]'::jsonb,'["empleado_ref","persona_ref"]'::jsonb),
  ('persona_dietas_rutas','[{"accion":"dietas.ruta.catalogo.consultar","modulo_id":"dietas","tipo_recurso":"catalogo_rutas_dietas","finalidades":["consultar_itinerario_dietas"],"garantia_minima":"alto"},{"accion":"dietas.ruta.calculo.solicitar","modulo_id":"dietas","tipo_recurso":"calculo_rutas_dietas","finalidades":["consultar_itinerario_dietas"],"garantia_minima":"alto"}]'::jsonb,'["ambito_ref"]'::jsonb)
 ) x(rol,concesiones,dimensiones) LOOP
  c:=jsonb_build_object('referencia','catalogo:aut107:contrato','version',1,
   'vigente_desde',desde,'vigente_hasta',hasta,
   'entradas','[]'::jsonb,'perfiles','[]'::jsonb);
  selecciones:='[]'::jsonb;
  FOR i IN 0..jsonb_array_length(r.concesiones)-1 LOOP
   entrada:=jsonb_build_object('referencia','accion:aut107:'||i::text,'version',1,
    'fuente_ref','fuente:aut107:contrato','fuente_version',1,'fuente_huella_sha256',repeat('b',64),
    'concesion',r.concesiones->i,'dimensiones_ambito',r.dimensiones,'clase_control','ordinario',
    'vigente_desde',desde,'vigente_hasta',hasta);
   c:=jsonb_set(c,'{entradas}',(c->'entradas')||jsonb_build_array(entrada));
   selecciones:=selecciones||jsonb_build_array(jsonb_build_object(
    'entrada_ref',entrada->>'referencia','entrada_version',1,'entrada_huella_sha256',
    encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')));
  END LOOP;
  p:=jsonb_build_object('operacion','crear','catalogo_ref','catalogo:aut107:contrato',
   'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64),
   'version_rol_objetivo_ref','rol:'||r.rol||':v1',
   'definicion_nueva',jsonb_build_object('rol_id',r.rol,'version',1,'nombre','Comisiones de servicio',
    'concesiones',r.concesiones),'selecciones',selecciones,
   'motivo',jsonb_build_object('catalogo_id','motivos_aut107','catalogo_version',1,
    'catalogo_huella_sha256',repeat('c',64),'entrada_clave','motivo_'||repeat('d',32)));
  DELETE FROM pg_temp.aut107_catalogo;INSERT INTO pg_temp.aut107_catalogo VALUES(c);
  resultado:=vec_autorizacion.validar_plan_rol_nuevo_v1(p,true);
  IF resultado IS DISTINCT FROM c#>'{entradas,0}' THEN
   RAISE EXCEPTION 'AUT107: descriptor positivo divergente';END IF;
  PERFORM vec_autorizacion.validar_plan_rol_nuevo_v1(p,false);
  cbase:=c;pbase:=p;
  FOREACH nombre IN ARRAY ARRAY['campos','obligaciones','dimensiones','orden','rol','duplicada','vencida'] LOOP
   c:=cbase;p:=pbase;
   CASE nombre
    WHEN 'campos' THEN c:=jsonb_set(c,'{entradas,0,concesion,campos_permitidos}','["campo_ajeno"]');
    WHEN 'obligaciones' THEN c:=jsonb_set(c,'{entradas,0,concesion,obligaciones}','["obligacion_ajena"]');
    WHEN 'dimensiones' THEN c:=jsonb_set(c,'{entradas,0,dimensiones_ambito}','["unidad_ajena"]');
    WHEN 'orden' THEN c:=jsonb_set(c,'{entradas,0,concesion}',cbase#>'{entradas,1,concesion}');
    WHEN 'rol' THEN p:=jsonb_set(jsonb_set(p,'{definicion_nueva,rol_id}','"otro_rol"'),
     '{version_rol_objetivo_ref}','"rol:otro_rol:v1"');
    WHEN 'duplicada' THEN NULL;
    WHEN 'vencida' THEN c:=jsonb_set(c,'{entradas,0,vigente_hasta}',to_jsonb(vencida));
   END CASE;
   p:=jsonb_set(p,'{definicion_nueva,concesiones,0}',c#>'{entradas,0,concesion}');
   p:=jsonb_set(p,'{selecciones,0,entrada_huella_sha256}',to_jsonb(encode(sha256(convert_to(
    vec_autorizacion.canon_gobierno_rol_nuevo_v1(c#>'{entradas,0}','entrada'),'UTF8')),'hex')));
   IF nombre='duplicada' THEN p:=jsonb_set(p,'{selecciones,1}',p#>'{selecciones,0}');END IF;
   DELETE FROM pg_temp.aut107_catalogo;INSERT INTO pg_temp.aut107_catalogo VALUES(c);
   rechazada:=false;
   BEGIN
    PERFORM vec_autorizacion.validar_plan_rol_nuevo_v1(p,true);
   EXCEPTION WHEN invalid_parameter_value OR insufficient_privilege THEN rechazada:=true;END;
   IF rechazada IS NOT TRUE THEN RAISE EXCEPTION 'AUT107: aceptada variante % para %',nombre,r.rol;END IF;
  END LOOP;
 END LOOP;
END $test$;
ROLLBACK;
