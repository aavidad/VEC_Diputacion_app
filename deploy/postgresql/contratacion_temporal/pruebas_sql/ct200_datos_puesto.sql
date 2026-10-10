\set ON_ERROR_STOP on
DO $ct200_prueba$
DECLARE s jsonb; original text; nuevo text; distintos bigint; total bigint;
BEGIN
 IF vec_contratacion_temporal.datos_puesto_peticion_validos_ct200('{}'::jsonb) IS DISTINCT FROM true
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200('{"jornada_minutos":2250,"numero_personas":2,"puesto_solicitado":"Administrativo C2"}'::jsonb) IS DISTINCT FROM true
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200('{"jornada_minutos":2250,"numero_personas":2.5,"puesto_solicitado":"Administrativo C2"}'::jsonb) IS DISTINCT FROM false
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200('{"jornada_minutos":2250,"numero_personas":0,"puesto_solicitado":"Administrativo C2"}'::jsonb) IS DISTINCT FROM false
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200('{"jornada_minutos":2250,"numero_personas":2}'::jsonb) IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'CT200: PARO clave=validacion_datos_puesto esperado=legacy+y_valido,_resto_no actual=divergente';
 END IF;
 SELECT pg_catalog.convert_from(alta_canonica,'UTF8')::jsonb->'solicitud' INTO s
 FROM vec_contratacion_temporal.expediente_alta_version ORDER BY expediente_ref LIMIT 1;
 IF s IS NULL THEN RAISE EXCEPTION 'CT200: PARO clave=fixture_alta esperado=presente actual=ausente'; END IF;
 original:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s);
 nuevo:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s||pg_catalog.jsonb_build_object('jornada_minutos',2250,'numero_personas',2,'puesto_solicitado','Administrativo C2'));
 IF nuevo IS DISTINCT FROM pg_catalog.left(original,-1)||',"jornada_minutos":2250,"numero_personas":2,"puesto_solicitado":"Administrativo C2"}' THEN
  RAISE EXCEPTION 'CT200: PARO clave=canon_nuevo esperado=campos_en_orden actual=divergente';
 END IF;
 SELECT pg_catalog.count(*), pg_catalog.count(*) FILTER (WHERE vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(pg_catalog.convert_from(alta_canonica,'UTF8')::jsonb) IS DISTINCT FROM alta_canonica)
 INTO total,distintos FROM vec_contratacion_temporal.expediente_alta_version;
 IF total=0 OR distintos<>0 THEN RAISE EXCEPTION 'CT200: PARO clave=canon_historico esperado=0_divergencias actual=%/%',distintos,total; END IF;
END $ct200_prueba$;
