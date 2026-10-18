\set ON_ERROR_STOP on
-- Tras AUT104 en copia desechable. Prueba el predicado que usan las dos
-- lecturas nominales; publicar el rol v3 exige el acto ADMIN gobernado aparte.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE bolsas jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]';
 modalidades jsonb:='["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 union_diez jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 funcion regprocedure:='vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(jsonb,jsonb)'::regprocedure;
BEGIN
 IF vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(bolsas,bolsas) IS NOT TRUE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(bolsas,union_diez) IS NOT TRUE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(modalidades,union_diez) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT104: proyección propia legítima denegada'; END IF;
 IF vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(modalidades,bolsas) IS NOT FALSE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  modalidades||'["diagnosticos_ajenos"]'::jsonb,union_diez) IS NOT FALSE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  bolsas-'version',union_diez) IS NOT FALSE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  '[]'::jsonb,union_diez) IS NOT FALSE
 OR vec_autorizacion.cotejar_campos_preferencias_modalidades_aut104(
  modalidades,'["modalidades_guardadas"]'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT104: proyección ajena o incompleta admitida'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
  WHERE p.oid=funcion AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=funcion
   AND p.proowner='vec_autorizacion_propietario'::regrole AND NOT p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[])
 THEN RAISE EXCEPTION 'AUT104: helper con ACL o configuración incompatible'; END IF;
END $test$;

DO $estados$
DECLARE base text[]:=ARRAY[
 'bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar',
 'bolsa.participaciones_propias.solicitar_pausa','bolsa.participaciones_propias.solicitar_reactivacion',
 'bolsa.participaciones_propias.responder_llamamiento','bolsa.participaciones_propias.manifestar_disposicion',
 'bolsa.participaciones_propias.confirmar_contacto'];
 pref text[]:=ARRAY['bolsa.preferencias_bolsas.consultar_propias','bolsa.preferencias_bolsas.ordenar_propias'];
 inscr text[]:=ARRAY['bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
  'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar','bolsa.inscripcion.presentar'];
 aleg text[]:=ARRAY['bolsa.alegacion.propias.consultar','bolsa.inscripcion.alegacion.presentar'];
 campos8 jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]';
 campos10 jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 caso record;concesiones jsonb;rol jsonb;legado jsonb;ampliado jsonb;desordenado jsonb;estados jsonb:='{}'::jsonb;par text[];
BEGIN
 FOR caso IN SELECT * FROM (VALUES
  (1,base,'base',campos8),
  (2,base||pref,'p8',campos8),(2,base||pref,'p10',campos10),
  (2,base||inscr,'i',campos8),
  (3,base||pref,'p10',campos10),
  (3,base||pref||inscr,'pi8',campos8),(3,base||pref||inscr,'pi10',campos10),
  (3,base||inscr||pref,'ip8',campos8),(3,base||inscr||pref,'ip10',campos10),
  (4,base||pref||inscr,'pi10',campos10),(4,base||inscr||pref,'ip10',campos10),
  (4,base||pref||inscr||aleg,'pia8',campos8),(4,base||inscr||pref||aleg,'ipa8',campos8),
  (4,base||pref||inscr||aleg,'pia10',campos10),(4,base||inscr||pref||aleg,'ipa10',campos10),
  (5,base||pref||inscr||aleg,'pia10',campos10),(5,base||inscr||pref||aleg,'ipa10',campos10)
 ) t(version,acciones,esperado,campos) LOOP
  SELECT jsonb_agg(CASE WHEN x.accion='bolsa.preferencias_bolsas.consultar_propias'
   THEN jsonb_build_object('accion',x.accion,'campos_permitidos',caso.campos)
   ELSE jsonb_build_object('accion',x.accion) END ORDER BY x.n)
  INTO concesiones FROM unnest(caso.acciones) WITH ORDINALITY x(accion,n);
  rol:=jsonb_build_object('rol_id','candidato_bolsa_portal_historial_propio_desarrollo',
   'version',caso.version,'concesiones',concesiones);
  IF vec_autorizacion.estado_rol_candidato_preferencias_aut104(rol) IS DISTINCT FROM caso.esperado THEN
   RAISE EXCEPTION 'AUT104: estado finito divergente %, versión %',caso.esperado,caso.version; END IF;
  estados:=jsonb_set(estados,ARRAY[caso.esperado||':'||caso.version::text],rol,true);
  IF caso.version=2 AND caso.esperado='p8' THEN legado:=rol; END IF;
  IF caso.version=3 AND caso.esperado='p10' THEN ampliado:=rol; END IF;
 END LOOP;
 IF vec_autorizacion.delta_modalidades_candidato_aut104(legado,ampliado) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT104: cambio exacto de consulta rechazado'; END IF;
 FOREACH par SLICE 1 IN ARRAY ARRAY[
  ARRAY['pi8:3','pi10:4'],ARRAY['ip8:3','ip10:4'],
  ARRAY['pia8:4','pia10:5'],ARRAY['ipa8:4','ipa10:5']] LOOP
  IF vec_autorizacion.delta_modalidades_candidato_aut104(estados->par[1],estados->par[2]) IS NOT TRUE THEN
   RAISE EXCEPTION 'AUT104: cambio tras inscripción o alegación rechazado %',par[1]; END IF;
 END LOOP;
 desordenado:=jsonb_set(jsonb_set(ampliado,'{concesiones,0}',ampliado#>'{concesiones,1}',false),
  '{concesiones,1}',ampliado#>'{concesiones,0}',false);
 IF vec_autorizacion.delta_modalidades_candidato_aut104(
  jsonb_set(legado,'{rol_id}','"rol_ajeno"'::jsonb),ampliado) IS NOT FALSE
 OR vec_autorizacion.delta_modalidades_candidato_aut104(legado,
  jsonb_set(ampliado,'{concesiones,0,tipo_recurso}','"ajeno"'::jsonb,true)) IS NOT FALSE
 OR vec_autorizacion.estado_rol_candidato_preferencias_aut104(desordenado) IS NOT NULL
 OR vec_autorizacion.estado_rol_candidato_preferencias_aut104(
  jsonb_set(ampliado,'{concesiones,0,accion}','"bolsa.accion_ajena"'::jsonb)) IS NOT NULL
 OR vec_autorizacion.estado_rol_candidato_preferencias_aut104(
  jsonb_set(ampliado,'{version}','4'::jsonb)) IS NOT NULL
 THEN RAISE EXCEPTION 'AUT104: grant ajeno, acción o versión mutados admitidos'; END IF;
END $estados$;
ROLLBACK;
