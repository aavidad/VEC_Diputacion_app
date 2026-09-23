\set ON_ERROR_STOP on
-- Ensayo desechable: ejecutar tras roles, Personal007, Dietas001..004 y
-- casos_funcionales.sql con solo_preparar=true. Nunca sobre datos conservados.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_dietas_propietario;
INSERT INTO vec_dietas.version_tarifa_provisional VALUES
 ('provisional:ensayo:20260921','PROVISIONAL · pendiente de confirmación por RRHH','BOE-A-2005-19988 / RD 462/2002','BOE-A-2023-16462 / RD 462/2002','2026-09-21',NULL,clock_timestamp());
INSERT INTO vec_dietas.importe_dieta_provisional VALUES
 ('provisional:ensayo:20260921','ES',1,60,50),
 ('provisional:ensayo:20260921','ES',2,60,50),
 ('provisional:ensayo:20260921','ES',3,60,50);
INSERT INTO vec_dietas.importe_km_provisional VALUES ('provisional:ensayo:20260921','automovil',0.2600);
RESET ROLE;
GRANT USAGE ON SCHEMA vec_dietas_prueba TO vec_dietas_propietario;
CREATE FUNCTION vec_dietas_prueba.preparar_calculada(op text,recurso text,nonce text,clave text)
RETURNS TABLE(material text,capacidad bytea,decision bytea,contexto bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $casos$
DECLARE m jsonb; d jsonb; cap jsonb; c jsonb; tramo jsonb; opciones jsonb:='[]'::jsonb; calc jsonb; i jsonb; amb text; atr text; h text; mt text; grupo int; campos jsonb;
BEGIN
 SELECT x.material,x.capacidad,x.decision,x.contexto INTO material,capacidad,decision,contexto
 FROM vec_dietas_prueba.preparar(op,recurso,nonce,clave) x;
 m:=material::jsonb; d:=convert_from(decision,'UTF8')::jsonb; cap:=convert_from(capacidad,'UTF8')::jsonb;
 IF op='crear' THEN
  tramo:=jsonb_build_object('fecha','2026-09-21','tipo','manutencion','porcentaje',50,'importe_centimos',2500,'version_tarifa_ref','provisional:ensayo:20260921','rotulo','PROVISIONAL · pendiente de confirmación por RRHH');
  FOR grupo IN 1..3 LOOP
   opciones:=opciones||jsonb_build_array(jsonb_build_object('grupo',grupo,'calculo',jsonb_build_object('tramos',jsonb_build_array(tramo),'manutencion_centimos',2500,'alojamiento_tope_centimos',0,'total_maximo_orientativo_centimos',2500,'version_tarifa_ref','provisional:ensayo:20260921','rotulo','PROVISIONAL · pendiente de confirmación por RRHH')));
  END LOOP;
  calc:=jsonb_build_object('procedencia','osrm_interno','motor','OSRM','version_grafo','grafo-sintetico-v1','version_tarifa','provisional:ensayo:20260921','rotulo','PROVISIONAL · pendiente de confirmación por RRHH','hora_inicio','08:00','hora_fin','18:00','kilometros','12.0000','eur_por_km','0.2600','importe_kilometraje_centimos',312,'tramos_ruta',jsonb_build_array(jsonb_build_object('origen_codigo','GR:001','destino_codigo','GR:002','kilometros','12.0000')),'opciones_dieta',opciones);
  m:=jsonb_set(m,'{comando,calculo}',calc,true);
  mt:=m::text; i:=m->'identidad';
  amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||',"relacion_ref":'||(i->'relacion_ref')::text||',"unidad_ref":'||(i->'unidad_ref')::text||'}';
  atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(mt,'UTF8')),'hex')||'","operacion":'||to_jsonb(op)::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||(m->'recurso_ref')::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":"sin_fin"}';
  h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
  d:=jsonb_set(d,'{contexto_recurso_huella_sha256}',to_jsonb(h));
  cap:=jsonb_set(cap,'{huella_efecto_sha256}',to_jsonb(h));
 END IF;
 campos:=CASE op WHEN 'lista' THEN '["items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.registrado_en","items.recibo.referencia","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb
 ELSE '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb END;
 d:=jsonb_set(d,'{campos_permitidos}',campos);
 cap:=jsonb_set(cap,'{huella_decision_sha256}',to_jsonb(encode(sha256(convert_to(d::text,'UTF8')),'hex')));
 material:=m::text; decision:=convert_to(d::text,'UTF8'); capacidad:=convert_to(cap::text,'UTF8');
 RETURN NEXT;
END $casos$;
ALTER FUNCTION vec_dietas_prueba.preparar_calculada(text,text,text,text) OWNER TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_dietas_prueba.preparar_calculada(text,text,text,text) TO vec_dietas_ejecutor;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION vec_prueba_dietas;
DO $pruebas$
DECLARE mt text; cap bytea; dec bytea; ctx bytea; salida jsonb; recibo text; ref text; n bigint;
BEGIN
 SELECT * INTO mt,cap,dec,ctx FROM vec_dietas_prueba.preparar_calculada('crear','dietas:borradores:propios','nonce_calculo_viejo','clave_calculo_00000009');
 SELECT x.decision,x.capacidad INTO dec,cap FROM vec_dietas_prueba.mutar_decision_capacidad(dec,cap,ARRAY['campos_permitidos'],'["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb) x;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_comision_calculada_v1(mt,cap,dec,'motivo'::bytea,ctx,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'proyección antigua aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 IF vec_dietas_prueba.contar_consumos_ad3()<>0 THEN RAISE EXCEPTION 'denegación consumió AD3'; END IF;
 SELECT * INTO mt,cap,dec,ctx FROM vec_dietas_prueba.preparar_calculada('crear','dietas:borradores:propios','nonce_calculo_uno','clave_calculo_00000001');
 salida:=vec_dietas.crear_o_recuperar_comision_calculada_v1(mt,cap,dec,'motivo'::bytea,ctx,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 recibo:=salida#>>'{recibo,referencia}'; ref:=salida#>>'{comision,referencia}';
 IF recibo IS NULL OR salida#>>'{comision,calculo,kilometros}'<>'12.0000' OR salida#>>'{comision,calculo,importe_kilometraje_centimos}'<>'312' THEN RAISE EXCEPTION 'comisión calculada no registrada'; END IF;
 SELECT * INTO mt,cap,dec,ctx FROM vec_dietas_prueba.preparar_calculada('crear','dietas:borradores:propios','nonce_calculo_dos','clave_calculo_00000001');
 salida:=vec_dietas.crear_o_recuperar_comision_calculada_v1(mt,cap,dec,'motivo'::bytea,ctx,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF salida#>>'{recibo,referencia}' IS DISTINCT FROM recibo OR salida#>>'{recibo,repeticion}'<>'true' THEN RAISE EXCEPTION 'replay no conservó recibo'; END IF;
 SELECT * INTO mt,cap,dec,ctx FROM vec_dietas_prueba.preparar_calculada('detalle',ref,'nonce_calculo_detalle','');
 salida:=vec_dietas.consultar_comisiones_calculadas_v1(mt,cap,dec,'motivo'::bytea,ctx,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF salida#>>'{recibo,referencia}' IS DISTINCT FROM recibo OR salida#>>'{comision,calculo,kilometros}'<>'12.0000' THEN RAISE EXCEPTION 'detalle no recuperó cálculo'; END IF;
 SELECT vec_dietas_prueba.contar_consumos_ad3() INTO n;
 IF n<>3 THEN RAISE EXCEPTION 'consumos AD3 inesperados: %',n; END IF;
END $pruebas$;
ROLLBACK;
