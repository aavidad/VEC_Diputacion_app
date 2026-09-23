\set ON_ERROR_STOP on
-- Casos con AD3 stub: ejercitan la fachada Dietas, no certifican AD3 real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE SCHEMA IF NOT EXISTS vec_dietas_prueba;
GRANT USAGE ON SCHEMA vec_dietas_prueba TO vec_dietas_ejecutor;
CREATE FUNCTION vec_dietas_prueba.preparar(op text,recurso text,nonce text,clave text,p_motivo text DEFAULT 'Visita técnica',p_limite integer DEFAULT 1,p_cursor text DEFAULT '',p_relacion text DEFAULT 'rel_cccccccccccccccccccccc',p_unidad text DEFAULT 'unidad:prueba')
RETURNS TABLE(material text,capacidad bytea,decision bytea,contexto bytea)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $casos$
DECLARE
 p text:='per_aaaaaaaaaaaaaaaaaaaaaa'; e text:='emp_bbbbbbbbbbbbbbbbbbbbbb'; rel text:=p_relacion;
 ctx jsonb; m jsonb; d jsonb; cap jsonb; vin jsonb; h text; amb text; atr text; mt text; n text; salida jsonb; primera text; segunda text; campos jsonb;
BEGIN
  ctx:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','principal_ref',p,'metodo','certificado','garantia','alto','perfil_activo_ref','prf_dddddddddddddddddddddd','persona_ref',p,'contexto_actor_ref','vca_eeeeeeeeeeeeeeeeeeeeee','contexto_version',1,'cuenta_ref','cta_ffffffffffffffffffffff','cuenta_version',1,'persona_version',1,'perfil_version',1,'estado','activo','vigente_desde','2026-01-01T00:00:00.000000Z','vigente_hasta','2027-01-01T00:00:00.000000Z','resuelto_en','2026-09-21T00:00:00.000000Z','vinculos',jsonb_build_array(jsonb_build_object('vinculo_ref','vin_gggggggggggggggggggggg','version',1,'tipo','empleado','referencia',e,'estado','activo','vigente_desde','2026-01-01T00:00:00.000000Z','vigente_hasta','2027-01-01T00:00:00.000000Z')));
  m:=jsonb_build_object('esquema','vec.dietas.borrador-operacion.v1','operacion',op,'recurso_ref',recurso,'identidad',jsonb_build_object('actor_ref',p,'perfil_ref','prf_dddddddddddddddddddddd','persona_ref',p,'empleado_ref',e,'relacion_ref',rel,'unidad_ref',p_unidad,'relacion_version',3,'vigente_desde','2026-01-01','vigente_hasta','','fecha_referencia','2026-09-21','procedencia_acto_ref','acto:prueba','fuente_ref','fuente:prueba','fuente_version',4,'contexto_actor_ref','vca_eeeeeeeeeeeeeeeeeeeeee','contexto_version',1,'cuenta_ref','cta_ffffffffffffffffffffff','cuenta_version',1,'persona_version',1,'perfil_version',1));
 IF op='crear' THEN m:=m||jsonb_build_object('comando',jsonb_build_object('clave_idempotencia',clave,'fecha_inicio','2026-09-21','fecha_fin','2026-09-21','motivo',p_motivo,'codigos_ruta',jsonb_build_array('GR:001','GR:002'))); m:=m||jsonb_build_object('huella_semantica',vec_dietas.huella_semantica_crear_borrador_v1(m::text));
 ELSIF op='lista' THEN m:=m||jsonb_build_object('consulta',jsonb_build_object('limite',p_limite,'cursor',p_cursor));
  ELSE m:=m||jsonb_build_object('referencia',recurso); END IF;
  mt:=m::text; amb:='{"empleado_ref":'||(m#>'{identidad,empleado_ref}')::text||',"persona_ref":'||(m#>'{identidad,persona_ref}')::text||',"relacion_ref":'||(m#>'{identidad,relacion_ref}')::text||',"unidad_ref":'||(m#>'{identidad,unidad_ref}')::text||'}';
  atr:='{"contexto_actor_ref":'||(m#>'{identidad,contexto_actor_ref}')::text||',"contexto_version":'||to_jsonb(m#>>'{identidad,contexto_version}')::text||',"cuenta_ref":'||(m#>'{identidad,cuenta_ref}')::text||',"cuenta_version":'||to_jsonb(m#>>'{identidad,cuenta_version}')::text||',"fecha_referencia":'||(m#>'{identidad,fecha_referencia}')::text||',"fuente_ref":'||(m#>'{identidad,fuente_ref}')::text||',"fuente_version":'||to_jsonb(m#>>'{identidad,fuente_version}')::text||',"material_sha256":"'||encode(sha256(convert_to(mt,'UTF8')),'hex')||'","operacion":'||to_jsonb(op)::text||',"perfil_version":'||to_jsonb(m#>>'{identidad,perfil_version}')::text||',"persona_version":'||to_jsonb(m#>>'{identidad,persona_version}')::text||',"procedencia_acto_ref":'||(m#>'{identidad,procedencia_acto_ref}')::text||',"recurso_ref":'||to_jsonb(recurso)::text||',"relacion_version":'||to_jsonb(m#>>'{identidad,relacion_version}')::text||',"vigente_desde":'||(m#>'{identidad,vigente_desde}')::text||',"vigente_hasta":"sin_fin"}';
  h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
  vin:=jsonb_build_object('esquema','vec.autorizacion.vinculo-autenticacion-actor.v2','bloque_version',2,'autenticacion_ref','aut','autenticacion_huella_sha256',repeat('1',64),'asercion_ref','ase','sesion_ref','ses','control_sesion_ref','cs','control_sesion_revision',1,'control_sesion_huella_sha256',repeat('2',64),'cuenta_ref','cta_ffffffffffffffffffffff','cuenta_ordinaria_ref','cta_ffffffffffffffffffffff','principal_id',p,'perfil_activo_ref','prf_dddddddddddddddddddddd','cuenta_privilegiada',false,'superficie','portal','metodo_observado','certificado','garantia_observada','alto','politica_garantia_ref','pol','politica_garantia_huella_sha256',repeat('3',64),'autenticacion_verificada_en','2026-09-21T00:00:00.000000Z','sesion_emitida_en','2026-09-21T00:00:00.000000Z','sesion_valida_hasta','2026-09-21T01:00:00.000000Z','sesion_revalidada_en','2026-09-21T00:00:00.000000Z','registro_contexto_ref','rca_hhhhhhhhhhhhhhhhhhhhhh','contexto_actor_esquema','vec.contexto-actor.vinculado.v2','contexto_actor_ref','vca_eeeeeeeeeeeeeeeeeeeeee','contexto_actor_version',1,'contexto_actor_cuenta_version',1,'contexto_actor_huella_sha256',encode(sha256(convert_to(ctx::text,'UTF8')),'hex'),'manifiesto_procedencia_huella_sha256',repeat('4',64),'autoridad_efectiva','autoridad_maestra_acreditada');
  campos:=CASE op WHEN 'lista' THEN '["items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.registrado_en","items.recibo.referencia","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb ELSE '["comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb END;
  d:=jsonb_build_object('esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2','bloque_version',3,'decision_ref','dec_'||nonce,'concedida',true,'codigo','concedida','principal_id',p,'perfil_activo_ref','prf_dddddddddddddddddddddd','accion',CASE WHEN op='crear' THEN 'dietas.borrador.propio.crear' ELSE 'dietas.borrador.propio.consultar' END,'recurso_ref',recurso,'modulo_id','dietas','tipo_recurso','comision_borrador','contexto_recurso_huella_sha256',h,'finalidad',CASE WHEN op='crear' THEN 'crear_borrador_propio' ELSE 'consultar_borrador_propio' END,'correlacion_ref','correlacion_'||repeat('a',32),'esquema_huella_solicitud','x','solicitud_huella_sha256',repeat('5',64),'esquema_huella_motivo','x','motivo_huella_sha256',repeat('6',64),'vinculo_autenticacion_actor',vin,'asignacion_ref','asig','asignacion_huella_sha256',repeat('7',64),'version_rol_ref','rol','version_rol_huella_sha256',repeat('8',64),'control_vigencia_version_rol_ref','cv','control_vigencia_version_rol_revision',1,'control_vigencia_version_rol_huella_sha256',repeat('9',64),'revision_catalogo_politicas',1,'catalogo_politicas_huella_sha256',repeat('a',64),'politicas_evaluadas','[]'::jsonb,'politicas_aplicables','[]'::jsonb,'garantia_minima','alto','campos_permitidos',campos,'obligaciones','[]'::jsonb,'emitida_en','2026-09-21T00:00:00.000000Z','valida_hasta','2026-09-21T01:00:00.000000Z');
  cap:=jsonb_build_object('esquema','vec.autorizacion.capacidad-registro-consumo-atestado.v3','version',3,'clave_id','key','clave_version',1,'revision_gobierno',1,'huella_gobierno_sha256',repeat('b',64),'emisor_id','stub','audiencia_consumo',CASE WHEN op='crear' THEN 'vec_dietas.borrador_propio.crear.v1' ELSE 'vec_dietas.borrador_propio.consultar.v1' END,'nonce',nonce,'emitida_en','2026-09-21T00:00:00Z','expira_en','2026-09-21T01:00:00Z','decision_ref',d->>'decision_ref','huella_decision_sha256',encode(sha256(convert_to(d::text,'UTF8')),'hex'),'huella_motivo_sha256',repeat('c',64),'huella_payload_vec_ad_3_sha256',repeat('d',64),'huella_sobre_cose_sign1_sha256',repeat('e',64),'huella_prueba_confianza_sha256',repeat('f',64),'contexto_ref',vin->>'registro_contexto_ref','huella_contexto_sha256',vin->>'contexto_actor_huella_sha256','audiencia_despliegue','stub','operacion',d->>'accion','efecto_ref',recurso,'huella_efecto_sha256',h,'decision_valida_hasta','2026-09-21T01:00:00Z','verificada_en','2026-09-21T00:00:00Z','revision_confianza','1','configuracion_secuencia',1,'huella_configuracion_sha256',repeat('1',64),'configuracion_publicada_en','2026-09-20T00:00:00Z','configuracion_expira_en','2026-09-22T00:00:00Z','raiz_clave_id','root','raiz_version',1,'huella_raiz_spki_sha256',repeat('2',64),'raiz_valida_desde','2026-01-01T00:00:00Z','raiz_valida_hasta','2027-01-01T00:00:00Z','suite','VEC-AD-3-COSE-EDDSA-1','mac_sha256',repeat('3',64));
  material:=mt; capacidad:=convert_to(cap::text,'UTF8'); decision:=convert_to(d::text,'UTF8'); contexto:=convert_to(ctx::text,'UTF8');
 RETURN NEXT;
END $casos$;
ALTER FUNCTION vec_dietas_prueba.preparar(text,text,text,text,text,integer,text,text,text) OWNER TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_dietas_prueba.preparar(text,text,text,text,text,integer,text,text,text) TO vec_dietas_ejecutor;
CREATE FUNCTION vec_dietas_prueba.contar_consumos_ad3() RETURNS bigint
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT count(*) FROM vec_autorizacion_atestada_v3.stub_consumo
$$;
ALTER FUNCTION vec_dietas_prueba.contar_consumos_ad3() OWNER TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_dietas_prueba.contar_consumos_ad3() TO vec_dietas_ejecutor;
CREATE FUNCTION vec_dietas_prueba.mutar_decision_capacidad(p_decision bytea,p_capacidad bytea,p_ruta text[],p_valor jsonb)
RETURNS TABLE(decision bytea,capacidad bytea) LANGUAGE sql SET search_path=pg_catalog AS $$
 WITH mutada AS (SELECT jsonb_set(convert_from(p_decision,'UTF8')::jsonb,p_ruta,p_valor) AS valor)
 SELECT convert_to(valor::text,'UTF8'),convert_to(jsonb_set(convert_from(p_capacidad,'UTF8')::jsonb,'{huella_decision_sha256}',to_jsonb(encode(sha256(convert_to(valor::text,'UTF8')),'hex')))::text,'UTF8') FROM mutada
$$;
ALTER FUNCTION vec_dietas_prueba.mutar_decision_capacidad(bytea,bytea,text[],jsonb) OWNER TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_dietas_prueba.mutar_decision_capacidad(bytea,bytea,text[],jsonb) TO vec_dietas_ejecutor;
-- Personal 000007 real: dos relaciones/unidades sintéticas de la misma persona.
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,empleado_ref,unidad_ref,estado,desde,hasta,version,procedencia_acto_ref,fuente_ref,fuente_version)
VALUES ('rel_cccccccccccccccccccccc','per_aaaaaaaaaaaaaaaaaaaaaa','emp_bbbbbbbbbbbbbbbbbbbbbb','unidad:prueba','activa','2026-01-01',NULL,3,'acto:prueba','fuente:prueba',4),
       ('rel_iiiiiiiiiiiiiiiiiiiiii','per_aaaaaaaaaaaaaaaaaaaaaa','emp_bbbbbbbbbbbbbbbbbbbbbb','unidad:otra','activa','2026-01-01',NULL,3,'acto:prueba','fuente:prueba',4);
RESET ROLE;
\if :solo_preparar
COMMIT;
\else
SET LOCAL SESSION AUTHORIZATION vec_prueba_dietas;
DO $pruebas$
DECLARE mt text; capacidad bytea; decision bytea; contexto bytea; salida jsonb; primera text; segunda text; comision_uno text; tercera text; cursor_uno text; comision_otra text; antes_consumos bigint; op text; material_mutado text;
BEGIN
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_uno','clave_idempotente_0001');
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); primera:=salida#>>'{recibo,referencia}'; comision_uno:=salida#>>'{comision,referencia}';
 IF salida#>>'{recibo,repeticion}'<>'false' OR primera IS NULL THEN RAISE EXCEPTION 'crear no devolvió recibo nuevo'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_dos','clave_idempotente_0001');
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); segunda:=salida#>>'{recibo,referencia}';
 IF salida#>>'{recibo,repeticion}'<>'true' OR segunda IS DISTINCT FROM primera THEN RAISE EXCEPTION 'replay fresco no conserva recibo'; END IF;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'consumo viejo aceptado'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 -- Misma clave, preimagen distinta y autorización fresca: no puede reconciliar.
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_conflicto','clave_idempotente_0001','Otro motivo');
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'conflicto idempotente aceptado'; EXCEPTION WHEN SQLSTATE 'PD002' THEN NULL; END;
 -- Segunda alta para verificar paginación real, no sólo el catálogo de la función.
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_tres','clave_idempotente_0002','Segunda visita');
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); tercera:=salida#>>'{comision,referencia}';
 IF tercera IS NULL OR tercera=comision_uno THEN RAISE EXCEPTION 'segunda alta no durable'; END IF;
 -- Misma persona y empleado, pero otra relación/unidad: concesión A no ve B.
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_otra','clave_idempotente_otra','Otra relación',1,'','rel_iiiiiiiiiiiiiiiiiiiiii','unidad:otra');
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); comision_otra:=salida#>>'{comision,referencia}';
 IF comision_otra IS NULL THEN RAISE EXCEPTION 'alta otra relación no durable'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('detalle',comision_uno,'nonce_detalle','');
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF salida#>>'{resultado}'<>'concedido' OR salida#>>'{recibo,referencia}'<>primera OR salida#>>'{comision,referencia}'<>comision_uno THEN RAISE EXCEPTION 'detalle propio no recuperable'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('detalle','dco_zzzzzzzzzzzzzzzzzzzzzz','nonce_ausente','');
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF salida#>>'{resultado}'<>'no_encontrado' THEN RAISE EXCEPTION 'detalle ausente incorrecto'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('detalle',comision_otra,'nonce_detalle_ajeno','', 'Visita técnica',1,'');
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF salida#>>'{resultado}'<>'no_encontrado' THEN RAISE EXCEPTION 'detalle A devolvió relación/unidad B'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('lista','dietas:borradores:propios','nonce_lista_1','', 'Visita técnica',1,'');
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); cursor_uno:=salida->>'siguiente_cursor';
 IF jsonb_array_length(salida->'items')<>1 OR cursor_uno='' THEN RAISE EXCEPTION 'primera página inválida'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('lista','dietas:borradores:propios','nonce_lista_2','', 'Visita técnica',1,cursor_uno);
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF jsonb_array_length(salida->'items')<>1 OR salida->>'siguiente_cursor'<>'' THEN RAISE EXCEPTION 'segunda página inválida'; END IF;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('lista','dietas:borradores:propios','nonce_lista_otra','', 'Visita técnica',10,'','rel_iiiiiiiiiiiiiiiiiiiiii','unidad:otra');
 salida:=vec_dietas.consultar_borradores_propios_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF jsonb_array_length(salida->'items')<>1 OR salida#>>'{items,0,comision,referencia}'<>comision_otra THEN RAISE EXCEPTION 'lista B no quedó aislada por relación/unidad'; END IF;
 -- Las capacidades se rehacen desde la decisión mutada: este caso no puede
 -- quedar cubierto por una huella vieja que falle antes de la guarda Dietas.
 FOR op IN SELECT unnest(ARRAY['crear','detalle','lista']) LOOP
   IF op='crear' THEN SELECT * INTO material_mutado,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_campo_crear','clave_idempotente_0006');
   ELSIF op='detalle' THEN SELECT * INTO material_mutado,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('detalle',comision_uno,'nonce_campo_detalle','');
   ELSE SELECT * INTO material_mutado,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('lista','dietas:borradores:propios','nonce_campo_lista','', 'Visita técnica',1,''); END IF;
   antes_consumos:=vec_dietas_prueba.contar_consumos_ad3();
   SELECT * INTO decision,capacidad FROM vec_dietas_prueba.mutar_decision_capacidad(decision,capacidad,ARRAY['campos_permitidos'],'["campo.restringido"]'::jsonb);
   BEGIN IF op='crear' THEN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(material_mutado,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); ELSE PERFORM vec_dietas.consultar_borradores_propios_v1(material_mutado,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); END IF; RAISE EXCEPTION 'campo restringido aceptado %',op; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
   IF vec_dietas_prueba.contar_consumos_ad3()<>antes_consumos THEN RAISE EXCEPTION 'campo restringido produjo consumo %',op; END IF;
   SELECT * INTO material_mutado,capacidad,decision,contexto FROM vec_dietas_prueba.preparar(op,CASE WHEN op='detalle' THEN comision_uno ELSE 'dietas:borradores:propios' END,'nonce_obligacion_'||op,CASE WHEN op='crear' THEN 'clave_idempotente_0007' ELSE '' END, 'Visita técnica',1,'');
   SELECT * INTO decision,capacidad FROM vec_dietas_prueba.mutar_decision_capacidad(decision,capacidad,ARRAY['obligaciones'],'["auditar"]'::jsonb);
   BEGIN IF op='crear' THEN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(material_mutado,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); ELSE PERFORM vec_dietas.consultar_borradores_propios_v1(material_mutado,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); END IF; RAISE EXCEPTION 'obligación desconocida aceptada %',op; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
   IF vec_dietas_prueba.contar_consumos_ad3()<>antes_consumos THEN RAISE EXCEPTION 'obligación desconocida produjo consumo %',op; END IF;
 END LOOP;
 -- Cada fragmento ligado a la preimagen se rechaza si se altera sin rehacer la
 -- decisión/capacidad: identidad, versiones, fechas, rutas, cursor/límite y huellas.
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_mutacion','clave_idempotente_0003');
 FOREACH mt IN ARRAY ARRAY[
   jsonb_set(mt::jsonb,'{identidad,persona_ref}','"per_zzzzzzzzzzzzzzzzzzzzzz"')::text,
   jsonb_set(mt::jsonb,'{identidad,empleado_ref}','"emp_zzzzzzzzzzzzzzzzzzzzzz"')::text,
   jsonb_set(mt::jsonb,'{identidad,relacion_ref}','"rel_zzzzzzzzzzzzzzzzzzzzzz"')::text,
   jsonb_set(mt::jsonb,'{identidad,unidad_ref}','"unidad:ajena"')::text,
   jsonb_set(mt::jsonb,'{identidad,persona_version}','2')::text,
   jsonb_set(mt::jsonb,'{identidad,relacion_version}','4')::text,
   jsonb_set(mt::jsonb,'{comando,fecha_inicio}','"2026-09-22"')::text,
   jsonb_set(mt::jsonb,'{comando,codigos_ruta}','["GR:999"]')::text,
   jsonb_set(mt::jsonb,'{huella_semantica}','"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"')::text
 ] LOOP
   BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'mutación ligada aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 END LOOP;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('lista','dietas:borradores:propios','nonce_mutacion_lista','', 'Visita técnica',1,'');
 BEGIN PERFORM vec_dietas.consultar_borradores_propios_v1(jsonb_set(mt::jsonb,'{consulta,limite}','51')::text,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'límite alterado aceptado'; EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN PERFORM vec_dietas.consultar_borradores_propios_v1(jsonb_set(mt::jsonb,'{consulta,cursor}','"dco_invalido"')::text,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'cursor alterado aceptado'; EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_mutacion_sello','clave_idempotente_0004');
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,convert_to(jsonb_set(convert_from(decision,'UTF8')::jsonb,'{decision_ref}','"dec_alterada"')::text,'UTF8'),'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'decision_ref alterada aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,convert_to(jsonb_set(convert_from(capacidad,'UTF8')::jsonb,'{efecto_ref}','"efecto:alterado"')::text,'UTF8'),decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'efecto alterado aceptado'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,convert_to(jsonb_set(convert_from(capacidad,'UTF8')::jsonb,'{huella_efecto_sha256}','"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"')::text,'UTF8'),decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'huella alterada aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,convert_to(jsonb_set(convert_from(contexto,'UTF8')::jsonb,'{persona_version}','2')::text,'UTF8'),1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'contexto alterado aceptado'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,2,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'versión persona alterada aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,2,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'versión perfil alterada aceptada'; EXCEPTION WHEN SQLSTATE 'PD003' THEN NULL; END;
END $pruebas$;
RESET SESSION AUTHORIZATION;
DO $$ BEGIN
 IF (SELECT count(*) FROM vec_dietas.borrador_comision)<>3 OR (SELECT count(*) FROM vec_dietas.historia_borrador_comision)<>3 OR (SELECT count(*) FROM vec_dietas.auditoria_borrador_comision WHERE accion='crear')<>4 THEN RAISE EXCEPTION 'historia/auditoría crear incoherente'; END IF;
END $$;
SET LOCAL SESSION AUTHORIZATION vec_prueba_dietas;
DO $personal$
DECLARE mt text; capacidad bytea; decision bytea; contexto bytea;
BEGIN
 SELECT * INTO mt,capacidad,decision,contexto FROM vec_dietas_prueba.preparar('crear','dietas:borradores:propios','nonce_personal_revocado','clave_idempotente_0005','Visita técnica',1,'','rel_zzzzzzzzzzzzzzzzzzzzzz','unidad:revocada');
 BEGIN PERFORM vec_dietas.crear_o_recuperar_borrador_propio_v1(mt,capacidad,decision,'motivo'::bytea,contexto,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea); RAISE EXCEPTION 'Personal revocado aceptado'; EXCEPTION WHEN SQLSTATE 'P7201' THEN NULL; END;
END $personal$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_dietas_propietario;
SELECT set_config('vec.dietas.persona_ref','per_zzzzzzzzzzzzzzzzzzzzzz',true);
DO $$ BEGIN IF EXISTS (SELECT 1 FROM vec_dietas.borrador_comision) THEN RAISE EXCEPTION 'RLS expuso borradores a persona ajena'; END IF; END $$;
RESET ROLE;
\if :confirmar
COMMIT;
\else
ROLLBACK;
\endif
\endif
