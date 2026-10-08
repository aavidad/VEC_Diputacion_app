\set ON_ERROR_STOP on
BEGIN;
DO $prueba$
DECLARE
 raw bytea := pg_catalog.convert_to(
   '{"esquema":"vec.ct.necesidades_alta.v1","referencia":"catalogo:ct:ejemplo","version":1,"jornada_referencia_minutos":2250,"causas":[{"clave":"sustitucion","regla_ref":"regla:ct:sustitucion:v1","fecha_fin":"opcional","causa_fin":"reincorporacion_titular","maximo_meses":36,"campos_permitidos":["puesto_codigo"],"campos_obligatorios":["puesto_codigo"]}]}',
   'UTF8');
 n jsonb;
 s jsonb;
 a jsonb;
 canon_v2 text;
 canon_v3 text;
 efecto_v3 text;
 p_abierto jsonb;
 s_abierta jsonb;
 canon_abierto text;
 raw_nuevo bytea;
 n_nuevo jsonb;
 s_nueva jsonb;
BEGIN
 n:=pg_catalog.jsonb_build_object(
   'esquema','vec.ct.necesidad_alta.v1',
   'catalogo_ref','catalogo:ct:ejemplo',
   'catalogo_version',1,
   'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(raw),'hex'),
   'causa_clave','sustitucion',
   'periodo',pg_catalog.jsonb_build_object('inicio','2026-10-01','fin','2026-10-31'),
   'jornada_minutos',2250,
   'campos',pg_catalog.jsonb_build_object('puesto_codigo','auxiliar'),
   'catalogo_instantanea',pg_catalog.replace(pg_catalog.encode(raw,'base64'),E'\n',''));
 s:=pg_catalog.jsonb_build_object(
   'centro_ref','centro:ejemplo', 'contacto_ref','contacto:ejemplo',
   'categoria_ref','categoria:ejemplo','grupo_subgrupo','C2',
   'motivo_clave','sustitucion','detalle','Prueba',
   'periodo',n->'periodo',
   'rc',pg_catalog.jsonb_build_object('existe',false,'numero','','fecha','',
       'importe',pg_catalog.jsonb_build_object('centimos',0,'moneda','EUR'),
       'documento_ref',''),
   'documentos_adjuntos','[]'::jsonb,'observaciones','',
   'necesidad',n);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s,'{necesidad,catalogo_huella_sha256}',
           pg_catalog.to_jsonb(pg_catalog.repeat('0',64)))) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s,'{necesidad,jornada_minutos}','10081'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s,'{necesidad,periodo,fin}',
           '"2026-11-01"'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s,'{necesidad,catalogo_instantanea}',
           '"abc="'::jsonb)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT193: validación de necesidad divergente';
 END IF;
 -- La publicación anterior conserva su admisión sin número de personas.
 -- La nueva publicación lo exige y liga su texto canónico al efecto.
 raw_nuevo:=pg_catalog.convert_to(
   '{"esquema":"vec.ct.necesidades_alta.v1","referencia":"catalogo:ct:ejemplo","version":2,"jornada_referencia_minutos":2250,"causas":[{"clave":"sustitucion","regla_ref":"regla:ct:sustitucion:v1","fecha_fin":"opcional","causa_fin":"reincorporacion_titular","maximo_meses":36,"campos_permitidos":["numero_personas","puesto_codigo"],"campos_obligatorios":["numero_personas","puesto_codigo"]}]}',
   'UTF8');
 n_nuevo:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(pg_catalog.jsonb_set(
   pg_catalog.jsonb_set(n,'{catalogo_version}','2'::jsonb),
   '{catalogo_huella_sha256}',pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(raw_nuevo),'hex'))),
   '{catalogo_instantanea}',pg_catalog.to_jsonb(pg_catalog.replace(pg_catalog.encode(raw_nuevo,'base64'),E'\n',''))),
   '{campos}',pg_catalog.jsonb_build_object('numero_personas','2','puesto_codigo','auxiliar'));
 s_nueva:=pg_catalog.jsonb_set(s,'{necesidad}',n_nuevo);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s_nueva) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s_nueva,'{necesidad,campos}',
           pg_catalog.jsonb_build_object('puesto_codigo','auxiliar'))) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s_nueva,'{necesidad,campos,numero_personas}',
           '"0"'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s_nueva,'{necesidad,campos,numero_personas}',
           '"01"'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
         pg_catalog.jsonb_set(s_nueva,'{necesidad,campos,numero_personas}',
           '"4294967296"'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s_nueva)
       NOT LIKE '%"campos":{"numero_personas":"2","puesto_codigo":"auxiliar"},"catalogo_instantanea":%' THEN
   RAISE EXCEPTION 'CT193: número de personas no ligado al catálogo y canon';
 END IF;
 canon_v2:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s-'necesidad');
 canon_v3:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s);
 IF vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(s-'necesidad')
       IS DISTINCT FROM canon_v2
    OR canon_v3 NOT LIKE pg_catalog.left(canon_v2,-1) || ',"necesidad":%'
    OR canon_v3 NOT LIKE '%"jornada_minutos":2250,"campos":{"puesto_codigo":"auxiliar"},"catalogo_instantanea":%'
    OR canon_v3 NOT LIKE '%"periodo":{"inicio":"2026-10-01","fin":"2026-10-31"}%'
    THEN RAISE EXCEPTION 'CT193: canon de solicitud divergente'; END IF;
 a:=pg_catalog.jsonb_build_object(
   'esquema','vec.contratacion-temporal.efecto-alta.v3',
   'reserva_ref','reserva:ejemplo','expediente_ref','expediente:ejemplo',
   'numero_visible','2026/001','recibo_ref','recibo:ejemplo',
   'organizacion_ref','org:ejemplo','actor_ref','actor:ejemplo',
   'perfil_ref','perfil:ejemplo','version',1,
   'flujo',pg_catalog.jsonb_build_object('definicion_ref','flujo:ejemplo',
      'version',1,'huella_sha256',pg_catalog.repeat('a',64)),
   'fase_actual','solicitud','estado_actual','en_curso','solicitud',s,
   'creado_en','2026-10-08T12:00:00.000000Z',
   'actualizado_en','2026-10-08T12:00:00.000000Z',
   'actuacion',pg_catalog.jsonb_build_object('secuencia',1,
      'version_expediente',1,'accion_clave','alta',
      'actor_ref','actor:ejemplo','unidad_ref','unidad:ejemplo',
      'recibo_ref','recibo:ejemplo','realizada_en','2026-10-08T12:00:00.000000Z',
      'fase_origen','','fase_destino','solicitud','estado_origen','pendiente',
      'estado_destino','en_curso','observaciones','',
      'documentos_ref','[]'::jsonb));
 efecto_v3:=pg_catalog.convert_from(
   vec_contratacion_temporal.reconstruir_efecto_alta_v3(a),'UTF8');
 IF efecto_v3 NOT LIKE '%"solicitud":' || canon_v3 || ',"creado_en":%'
    OR pg_catalog.convert_from(
       vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a),'UTF8')
       IS DISTINCT FROM efecto_v3
    OR vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(
       pg_catalog.jsonb_set(a-'solicitud', '{esquema}',
         '"vec.contratacion-temporal.efecto-alta.v2"'::jsonb)
       || pg_catalog.jsonb_build_object('solicitud',s-'necesidad'))
       IS DISTINCT FROM vec_contratacion_temporal.reconstruir_efecto_alta_v2(
       pg_catalog.jsonb_set(a-'solicitud', '{esquema}',
         '"vec.contratacion-temporal.efecto-alta.v2"'::jsonb)
       || pg_catalog.jsonb_build_object('solicitud',s-'necesidad')) THEN
   RAISE EXCEPTION 'CT193: canon de efecto divergente';
 END IF;
 p_abierto:=pg_catalog.jsonb_build_object(
   'inicio','2026-10-01','causa_fin','reincorporacion_titular',
   'politica_fin',pg_catalog.jsonb_build_object(
      'regla_ref','regla:ct:sustitucion:v1','catalogo_version',1,
      'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(raw),'hex'),
      'fecha_fin','opcional','causa_fin','reincorporacion_titular'));
 s_abierta:=pg_catalog.jsonb_set(
   pg_catalog.jsonb_set(s,'{periodo}',p_abierto),'{necesidad,periodo}',p_abierto);
 canon_abierto:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s_abierta);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s_abierta) IS NOT TRUE
    OR canon_abierto NOT LIKE '%"periodo":{"inicio":"2026-10-01","causa_fin":"reincorporacion_titular","politica_fin":{"regla_ref":"regla:ct:sustitucion:v1","catalogo_version":1,"catalogo_huella_sha256":"%'
    OR pg_catalog.convert_from(vec_contratacion_temporal.reconstruir_efecto_alta_v3(
       pg_catalog.jsonb_set(a,'{solicitud}',s_abierta)),'UTF8')
       NOT LIKE '%"solicitud":' || canon_abierto || ',"creado_en":%' THEN
   RAISE EXCEPTION 'CT193: canon abierto de sustitución divergente';
 END IF;
 IF pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)','EXECUTE') THEN
   RAISE EXCEPTION 'CT193: ACL divergente';
 END IF;
 IF NOT EXISTS (
   SELECT 1 FROM pg_catalog.pg_proc p
    WHERE p.oid=pg_catalog.to_regprocedure(
      'vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)')
      AND p.prosecdef
      AND EXISTS (SELECT 1 FROM pg_catalog.unnest(p.proconfig) cfg
                   WHERE pg_catalog.regexp_replace(cfg,'[[:space:]]','','g')
                         ='search_path=pg_catalog,pg_temp')
      AND p.proconfig @> ARRAY['row_security=on']) THEN
   RAISE EXCEPTION 'CT193: guarda SECURITY DEFINER divergente';
 END IF;
END $prueba$;
ROLLBACK;
