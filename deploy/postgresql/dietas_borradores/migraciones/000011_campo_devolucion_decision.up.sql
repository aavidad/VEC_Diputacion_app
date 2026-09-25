\set ON_ERROR_STOP on
-- D6: la devolución solo se devuelve cuando la decisión V3 la concede.
-- 000010 proyecta comision.devolucion a la persona titular y a quien revisa
-- un reenvío, pero la decisión V3 limita los campos de cada lectura y sus
-- listas no incluían la devolución. Esta migración, simétrica de AD3-75:
-- 1) los cotejos admiten EXACTAMENTE la lista base o la base más la
--    devolución (comision.devolucion y, en la consulta con listado, también
--    items.comision.devolucion); cualquier otra lista se sigue rechazando;
-- 2) la titular solo recibe comision.devolucion (detalle, listado y
--    respuesta de mutación) si su decisión la concede: autorizar_documento_v2
--    fija en la transacción vec.dietas.campo_devolucion ('concedido' o
--    'denegado') junto a vec.dietas.persona_ref, y las dos proyecciones, que
--    solo ejecuta el propietario tras esa autorización, lo exigen. Cualquier
--    otro valor, o su ausencia, omite la devolución;
-- 3) quien revisa solo recibe la devolución anterior si su decisión la
--    concede, comprobado en consultar_documento_circuito_v1;
-- 4) corrige la clave de auditoría del detalle en
--    consultar_comisiones_propias_v2: «a->>'x'||m->>'y'» se agrupaba como
--    «(a->>'x'||m)->>'y'» y fallaba con 42883 en toda consulta de detalle
--    concedida; ahora cada extracción va entre paréntesis. Y el listado
--    fallaba con 42702 porque el alias «b» de la consulta coincidía con la
--    variable de fila «b»; la consulta usa ahora el alias «bc».
-- Reescritura por anclas: preimagen exacta (md5 del cuerpo, dueño, SECURITY
-- DEFINER, ACL y entorno) y postimagen exacta (md5 nuevo, metadatos y
-- dependencias idénticos y reversión textual al original). Requiere 000010.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000011:campo-devolucion:v1',0));

DO $funciones$
DECLARE x record; f oid; original text; nuevo text; actual text; meta jsonb; deps jsonb; n int; hechas int:=0;
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor')
    OR to_regprocedure('vec_dietas.devolucion_vigente_comision_v1(text,bigint)') IS NULL
    OR to_regprocedure('vec_dietas.devolucion_anterior_comision_v1(text,bigint)') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000011: falta 000010' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea)','e1d5ccfb0f95b067570881ba4b8610f0',
   '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog}',
   ARRAY[$a$        amb text; atr text; h text; hasta text; hc text; campos jsonb;$a$,
         $a$  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;$a$,
         $a$  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
         $a$    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb$a$],
   ARRAY[$a$        amb text; atr text; h text; hasta text; hc text; campos jsonb; campos_devolucion jsonb;$a$,
         $a$  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
  campos_devolucion:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;$a$,
         $a$  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
  campos_devolucion:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.devolucion","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
         $a$    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb$a$],
   'bd7bec9285ac10a6df2ee12f5c93fb50'),
  ('vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea)','a52f6b5cb8933db3a9161e45db799747',
   '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog}',
   ARRAY[$a$ campos jsonb; huella text; amb text; atr text;$a$,
         $a$  campos:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
         $a$    OR d->'campos_permitidos' IS DISTINCT FROM campos
$a$],
   ARRAY[$a$ campos jsonb; campos_devolucion jsonb; huella text; amb text; atr text;$a$,
         $a$  campos:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
  campos_devolucion:='["comision.calculo","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
         $a$    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false)
$a$],
   'e089171183d51c2b3b73b9fb47e12fe1'),
  ('vec_dietas.autorizar_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','2c1a009d48a0759a9e0680f2005b69b5',
   '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}',
   ARRAY[$a$ PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);$a$],
   ARRAY[$a$ PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);
 PERFORM set_config('vec.dietas.campo_devolucion',
  CASE WHEN d->'campos_permitidos' ? 'comision.devolucion' THEN 'concedido' ELSE 'denegado' END,true);$a$],
   '6ec54260f50834416008875a0d61c568'),
  ('vec_dietas.proyectar_comision_v2(text,boolean)','3cf1b715a1711dfffd34d63cbbbc1397',
   '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}',
   ARRAY[$a$  IF x.estado IN ('devuelta','borrador') THEN$a$],
   ARRAY[$a$  IF x.estado IN ('devuelta','borrador')
     AND current_setting('vec.dietas.campo_devolucion',true) IS NOT DISTINCT FROM 'concedido' THEN$a$],
   'b173ff2788190e519b43d0d439aaeb76'),
  ('vec_dietas.proyectar_revision_exacta_v2(text,bigint)','4dcae3043307e3bed4d44267c06a8692',
   '{vec_dietas_propietario=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC}',
   ARRAY[$a$ IF x.estado IN ('devuelta','borrador') THEN$a$],
   ARRAY[$a$ IF x.estado IN ('devuelta','borrador')
    AND current_setting('vec.dietas.campo_devolucion',true) IS NOT DISTINCT FROM 'concedido' THEN$a$],
   'd5cedfc3c5a792f5abbd693931710bb5'),
  ('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1e3ee14aed0ae31a52fdd979f2c688f8',
   '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}',
   ARRAY[$a$ IF devolucion IS NOT NULL THEN$a$],
   ARRAY[$a$ IF devolucion IS NOT NULL AND d->'campos_permitidos' ? 'comision.devolucion' THEN$a$],
   'd7e927640fa9329aefc5a04e5baaac0d'),
  ('vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','7f4934beacaf93db3806828b882c3199',
   '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}','{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}',
   ARRAY[$a$   'adi_'||md5(a->>'auditoria_ad3_ref'||m->>'referencia'||ahora::text),$a$,
         $a$ FOR b IN SELECT b.* FROM vec_dietas.borrador_comision b
  LEFT JOIN LATERAL (SELECT r.estado FROM vec_dietas.comision_revision r
   WHERE r.comision_ref=b.referencia ORDER BY r.version DESC LIMIT 1) actual ON true
  WHERE b.persona_ref=i->>'persona_ref' AND b.empleado_ref=i->>'empleado_ref'
    AND b.relacion_ref=i->>'relacion_ref' AND b.unidad_ref=i->>'unidad_ref'
    AND coalesce(actual.estado,'borrador')<>'eliminado'
    AND (coalesce(q->>'cursor','')='' OR b.referencia>q->>'cursor')
  ORDER BY b.referencia LIMIT (q->>'limite')::int+1 LOOP$a$],
   ARRAY[$a$   'adi_'||md5((a->>'auditoria_ad3_ref')||(m->>'referencia')||ahora::text),$a$,
         $a$ FOR b IN SELECT bc.* FROM vec_dietas.borrador_comision bc
  LEFT JOIN LATERAL (SELECT r.estado FROM vec_dietas.comision_revision r
   WHERE r.comision_ref=bc.referencia ORDER BY r.version DESC LIMIT 1) actual ON true
  WHERE bc.persona_ref=i->>'persona_ref' AND bc.empleado_ref=i->>'empleado_ref'
    AND bc.relacion_ref=i->>'relacion_ref' AND bc.unidad_ref=i->>'unidad_ref'
    AND coalesce(actual.estado,'borrador')<>'eliminado'
    AND (coalesce(q->>'cursor','')='' OR bc.referencia>q->>'cursor')
  ORDER BY bc.referencia LIMIT (q->>'limite')::int+1 LOOP$a$],
   '9e8b22789d382c3f895e0eeb90775b4c')
 ) v(firma,huella,acl,entorno,antes,despues,huella_nueva) LOOP
  f:=to_regprocedure(x.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'Dietas 000011: función ausente %',x.firma USING ERRCODE='55000'; END IF;
  SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
  IF (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella
     OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_dietas_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.acl
     OR (SELECT proconfig::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.entorno
     OR strpos(original,'campo_devolucion')<>0 OR strpos(original,'campos_devolucion')<>0
  THEN RAISE EXCEPTION 'Dietas 000011: preimagen de % incompatible',x.firma USING ERRCODE='55000'; END IF;
  nuevo:=original;
  FOR n IN 1..cardinality(x.antes) LOOP
   IF length(original)-length(replace(original,x.antes[n],''))<>length(x.antes[n])
      OR strpos(original,x.despues[n])<>0
   THEN RAISE EXCEPTION 'Dietas 000011: ancla % de % incompatible',n,x.firma USING ERRCODE='55000'; END IF;
   nuevo:=replace(nuevo,x.antes[n],x.despues[n]);
  END LOOP;
  EXECUTE nuevo;
  SELECT pg_get_functiondef(f) INTO STRICT actual;
  IF actual IS DISTINCT FROM nuevo
     OR (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella_nueva
     OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
     OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
         FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'Dietas 000011: % alterada fuera del contrato',x.firma USING ERRCODE='55000'; END IF;
  FOR n IN REVERSE cardinality(x.despues)..1 LOOP
   actual:=replace(actual,x.despues[n],x.antes[n]);
  END LOOP;
  IF actual IS DISTINCT FROM original THEN
   RAISE EXCEPTION 'Dietas 000011: % no revierte al original',x.firma USING ERRCODE='55000'; END IF;
  hechas:=hechas+1;
 END LOOP;
 IF hechas<>7 THEN RAISE EXCEPTION 'Dietas 000011: postimagen incompleta' USING ERRCODE='55000'; END IF;
END $funciones$;
COMMIT;
