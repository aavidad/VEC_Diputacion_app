\set ON_ERROR_STOP on
-- AD268. Sucesora prospectiva de AD75, que no llegó a instalarse: la copia
-- vigente tiene las tres fachadas de AD59/AD80 con search_path endurecido a
-- pg_catalog, pg_temp. AD75 espera otra proconfig y se detiene antes de escribir.
-- Solo se ajustan las listas cerradas de campos y su comparación como en AD75:
-- la titular recibe las listas que coteja Dietas 000006, opcionalmente con
-- devolución de Dietas 000010/000011; el revisor conserva su lista base y
-- admite la devolución. Toda lista parcial o adicional sigue denegada.
-- Requiere AD59 y AD80. No crea acción, permiso, tabla ni efecto externo.
-- Guarda de definición SHA256 y metadatos del clon; postimagen SHA256 y
-- conservación exacta de metadatos y dependencias. Una sola instalación.
-- Sin DOWN sobre historia. Dirección ensaya e instala con Dietas D9-D11.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000268',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $fachadas$
DECLARE
 x record; f oid; original text; nuevo text; actual text; meta jsonb; deps jsonb;
 acl_cerrada text:='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_dietas_propietario=X/vec_autorizacion_atestada_v3_propietario}';
 comparacion text:=E'    OR d->''campos_permitidos'' IS DISTINCT FROM campos\n';
 comparacion_nueva text:=E'    OR NOT coalesce(d->''campos_permitidos''=campos OR d->''campos_permitidos''=campos_devolucion,false)\n';
 hechas int:=0;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD268: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('registrar_y_consumir_dietas_documento_v3_atestada','0ae0a5f64c1fd8a9ba6cd57ff86f2857','be3605e10950bfd386f44716873f2640fda6c2305e268c4438879f3857754602',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;$a$,
   '25761c5d3043a77f683aa5eab3f35d17','7cae47bdaabccb07db5391971e5317e760650755d62dcdd9e18285dcb60496eb'),
  ('registrar_y_consumir_dietas_documento_consulta_v3_atestada','d0052e3fcaddbdc33e9b7f7d017a125e','7080ead4d594e041013d8e27d5ad60b5f8bbef0f6683db0acedf6198986355bf',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.devolucion","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;$a$,
   '500cb2bbcf38de7143ff81ac323e7659','1af5b1035daf3a59ed819232ac02e806efb31e5c3fe675efc1ca499424b8c2c1'),
  ('registrar_y_consumir_dietas_revisor_documento_v3_atestada','b0b8039b47e499237078fdd3477b295b','f0edfa8d3e70ed77993da388b234ab9a4f61a99f79e0b5b4339ab6e07ef31020',
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
   $a$ campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;$a$,
   '4ab4c204d9e14b6bf9749550cab4914a','b9d8f2bdb400bb178e0f4c00c0405ca5bae5096b6d197ad923ce8f592dec1a15')
 ) v(nombre,huella,pre_sha,campos,campos_nuevos,huella_nueva,post_sha) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD268: fachada ausente %',x.nombre USING ERRCODE='55000'; END IF;
  SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
  -- Preimagen PG18 del clon: definición SHA256, cuerpo, dueño, entorno y ACL.
  -- Cada ancla aparece exactamente una vez; ninguna otra cláusula se modifica.
  IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM x.pre_sha
     OR (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella
     OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
     OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl_cerrada
     OR length(original)-length(replace(original,x.campos,''))<>length(x.campos)
     OR length(original)-length(replace(original,comparacion,''))<>length(comparacion)
     OR strpos(original,'campos_devolucion')<>0
  THEN RAISE EXCEPTION 'AD268: preimagen de % incompatible',x.nombre USING ERRCODE='55000'; END IF;
  nuevo:=replace(replace(original,x.campos,x.campos_nuevos),comparacion,comparacion_nueva);
  EXECUTE nuevo;
  SELECT pg_get_functiondef(f) INTO STRICT actual;
  IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM x.post_sha
     OR actual IS DISTINCT FROM nuevo
     OR (SELECT md5(prosrc) FROM pg_proc WHERE oid=f) IS DISTINCT FROM x.huella_nueva
     OR replace(replace(actual,comparacion_nueva,comparacion),x.campos_nuevos,x.campos) IS DISTINCT FROM original
     OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
     OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
         FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'AD268: % alterada fuera del contrato',x.nombre USING ERRCODE='55000'; END IF;
  hechas:=hechas+1;
 END LOOP;
 IF hechas<>3 THEN RAISE EXCEPTION 'AD268: postimagen incompleta' USING ERRCODE='55000'; END IF;
END $fachadas$;
COMMIT;
