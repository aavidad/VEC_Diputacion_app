\set ON_ERROR_STOP on
-- AD3-75: la devolución del circuito de Dietas (D6, Dietas 000010) entra en
-- las listas cerradas de campos que admiten las fachadas del documento.
-- Dietas 000010 proyecta comision.devolucion a la persona titular (lectura y
-- respuesta de mutación: proyectar_comision_v2 y proyectar_revision_exacta_v2)
-- y a quien revisa un reenvío (consultar_documento_circuito_v1), pero las
-- listas de AD3-59 y AD3-80 no lo incluían. Cada fachada admite ahora
-- EXACTAMENTE una de dos listas: la base o la base más la devolución
-- (comision.devolucion y, en la consulta con listado, también
-- items.comision.devolucion). Cualquier otra lista, más amplia o parcial,
-- se sigue denegando con 42501. Dietas 000011 hace el cotejo simétrico y
-- solo devuelve la devolución cuando la decisión la concede.
--
-- Alineación con Dietas 000006: las listas de la persona titular de AD3-59
-- no coincidían con las que coteja Dietas (cotejar_recurso_documento_v2), que
-- son las que realmente proyecta: comision.centro_ref, comision.unidad_ref,
-- recibo.regla_ref y recibo.regla_huella_sha256 (y sus items.*). Ninguna
-- decisión podía superar a la vez Dietas y AD3-59, así que la ruta de la
-- titular estaba cerrada de hecho. La base de las dos fachadas de la
-- titular pasa a ser exactamente la de Dietas 000006; la lista antigua deja
-- de admitirse porque Dietas la rechaza siempre. La del revisor (AD3-80) ya
-- coincidía con Dietas 000008 y solo gana la devolución.
--
-- Reescritura por anclas con preimagen exacta (md5 del cuerpo, dueño,
-- SECURITY DEFINER, entorno y ACL cerrada) y postimagen exacta (md5 del
-- cuerpo nuevo, metadatos y dependencias idénticos, y reversión textual al
-- original). No toca el núcleo ni la restricción de audiencias; toma aun así
-- el consultivo común del núcleo para instalarse en serie con AD3-53/59/61/80.
-- Requiere AD3-59 y AD3-80. Se instala una sola vez; sin DOWN, como AD3-54/56.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000075',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $fachadas$
DECLARE
 x record; f oid; original text; nuevo text; actual text; meta jsonb; deps jsonb;
 acl_cerrada text:='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_dietas_propietario=X/vec_autorizacion_atestada_v3_propietario}';
 comparacion text:=E'    OR d->''campos_permitidos'' IS DISTINCT FROM campos\n';
 comparacion_nueva text:=E'    OR NOT coalesce(d->''campos_permitidos''=campos OR d->''campos_permitidos''=campos_devolucion,false)\n';
 hechas int:=0;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-75: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('registrar_y_consumir_dietas_documento_v3_atestada','0ae0a5f64c1fd8a9ba6cd57ff86f2857',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;$a$,
   '25761c5d3043a77f683aa5eab3f35d17'),
  ('registrar_y_consumir_dietas_documento_consulta_v3_atestada','d0052e3fcaddbdc33e9b7f7d017a125e',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.devolucion","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
   '500cb2bbcf38de7143ff81ac323e7659'),
  ('registrar_y_consumir_dietas_revisor_documento_v3_atestada','b0b8039b47e499237078fdd3477b295b',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
   '4ab4c204d9e14b6bf9749550cab4914a')
 ) v(nombre,huella,campos,campos_nuevos,huella_nueva) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD3-75: fachada ausente %',x.nombre USING ERRCODE='55000'; END IF;
  SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
  -- Preimagen exacta: el cuerpo instalado por AD3-59/AD3-80, su dueño, su
  -- entorno y su ACL cerrada; cada ancla aparece exactamente una vez.
  IF (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella
     OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
     OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl_cerrada
     OR length(original)-length(replace(original,x.campos,''))<>length(x.campos)
     OR length(original)-length(replace(original,comparacion,''))<>length(comparacion)
     OR strpos(original,'campos_devolucion')<>0
  THEN RAISE EXCEPTION 'AD3-75: preimagen de % incompatible',x.nombre USING ERRCODE='55000'; END IF;
  nuevo:=replace(replace(original,x.campos,x.campos_nuevos),comparacion,comparacion_nueva);
  EXECUTE nuevo;
  SELECT pg_get_functiondef(f) INTO STRICT actual;
  IF actual IS DISTINCT FROM nuevo
     OR (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella_nueva
     OR replace(replace(actual,comparacion_nueva,comparacion),x.campos_nuevos,x.campos) IS DISTINCT FROM original
     OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
     OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
         FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'AD3-75: % alterada fuera del contrato',x.nombre USING ERRCODE='55000'; END IF;
  hechas:=hechas+1;
 END LOOP;
 IF hechas<>3 THEN RAISE EXCEPTION 'AD3-75: postimagen incompleta' USING ERRCODE='55000'; END IF;
END $fachadas$;
COMMIT;
