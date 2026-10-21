\set ON_ERROR_STOP on
BEGIN READ ONLY;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE fila record;antes jsonb;despues jsonb;esperado jsonb;cuenta integer:=0;
 base jsonb:='{"estado":"publicada","concesiones":[{"accion":"prueba","campos_permitidos":[],"obligaciones":[]}]}'::jsonb;
 n jsonb;cargo jsonb;plan jsonb;rol jsonb;
BEGIN
 -- Reproduce el cuerpo instalado anterior para cotejar todas las versiones.
 FOR fila IN SELECT documento FROM vec_autorizacion.version_rol LOOP
  cuenta:=cuenta+1;
  antes:=jsonb_set(fila.documento,'{concesiones}',
   (SELECT jsonb_agg(CASE WHEN gc ? 'obligaciones'
     AND gc->'obligaciones' IN ('[]'::jsonb,'null'::jsonb)
      THEN gc-'obligaciones' ELSE gc END ORDER BY ordinal)
    FROM jsonb_array_elements(fila.documento->'concesiones') WITH ORDINALITY AS x(g,ordinal)
    CROSS JOIN LATERAL (SELECT CASE WHEN g ? 'campos_permitidos'
     AND g->'campos_permitidos' IN ('[]'::jsonb,'null'::jsonb)
      THEN g-'campos_permitidos' ELSE g END AS gc) normalizada));
  esperado:=CASE WHEN fila.documento->>'estado'='publicada'
    AND NOT (fila.documento ? 'retirada_en')
    THEN jsonb_set(antes,'{retirada_en}','"0001-01-01T00:00:00Z"'::jsonb)
    ELSE antes END;
  despues:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(fila.documento);
  IF despues IS DISTINCT FROM esperado
   OR despues-'retirada_en' IS DISTINCT FROM antes-'retirada_en'
  THEN RAISE EXCEPTION 'AUT106: una versión almacenada cambió fuera del instante cero' USING ERRCODE='55000'; END IF;
 END LOOP;
 RAISE NOTICE 'AUT106: versiones históricas cotejadas=%',cuenta;

 -- Reproduce el rol que publica el circuito de cargos; no requiere semilla.
 cargo:=jsonb_build_object('rol_id','ct_cargo_jefatura_firma_dev','version',2,
  'nombre','Jefatura de RRHH','version_anterior_sha256',repeat('a',64),
  'operaciones_v2','["consultar_r5","firmar_vec","recuperar_r5"]'::jsonb,
  'competencias','[{"accion":"contratacion_temporal.documento.firma_vec.registrar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}]'::jsonb,
  'version_rol_sha256',repeat('b',64),'regla_asignacion','lote_ordinario',
  'organizacion_ref','org:rrhh','vigente_desde','2026-10-16T00:00:00Z',
  'vigente_hasta','2027-01-13T00:00:00Z','duracion_propuesta_segundos',3600);
 plan:=jsonb_build_object('esquema','vec.admin.cargos-firma.plan.v2',
  'operacion_ref','rpa_cf_00000000000000000000000000000106','preparado_en','2026-10-16T00:00:00Z');
 rol:=vec_autorizacion.documento_rol_cargo_firma_v1(cargo,plan);
 IF rol->>'estado' IS DISTINCT FROM 'publicada' OR rol ? 'retirada_en'
 THEN RAISE EXCEPTION 'AUT106: el constructor no reproduce la omisión' USING ERRCODE='55000'; END IF;
 antes:=jsonb_set(rol,'{concesiones}',
  (SELECT jsonb_agg(CASE WHEN gc ? 'obligaciones'
    AND gc->'obligaciones' IN ('[]'::jsonb,'null'::jsonb)
     THEN gc-'obligaciones' ELSE gc END ORDER BY ordinal)
   FROM jsonb_array_elements(rol->'concesiones') WITH ORDINALITY AS x(g,ordinal)
   CROSS JOIN LATERAL (SELECT CASE WHEN g ? 'campos_permitidos'
    AND g->'campos_permitidos' IN ('[]'::jsonb,'null'::jsonb)
     THEN g-'campos_permitidos' ELSE g END AS gc) normalizada));
 esperado:=jsonb_set(antes,'{retirada_en}','"0001-01-01T00:00:00Z"'::jsonb);
 IF vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(rol) IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'AUT106: cargo publicado no coincide con el canon Go' USING ERRCODE='55000'; END IF;

 n:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(base);
 IF n IS DISTINCT FROM '{"estado":"publicada","retirada_en":"0001-01-01T00:00:00Z","concesiones":[{"accion":"prueba"}]}'::jsonb
 THEN RAISE EXCEPTION 'AUT106: publicado sin retirada no coincide con Go' USING ERRCODE='55000'; END IF;
 IF vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(base-'estado')
  IS DISTINCT FROM n-'estado'-'retirada_en'
 THEN RAISE EXCEPTION 'AUT106: documento sin estado recibió instante' USING ERRCODE='55000'; END IF;
 IF vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_set(base,'{estado}','"retirada"'::jsonb)) ? 'retirada_en'
 THEN RAISE EXCEPTION 'AUT106: rol retirado recibió instante de publicado' USING ERRCODE='55000'; END IF;
 IF vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   base||'{"retirada_en":null}'::jsonb)->'retirada_en' IS DISTINCT FROM 'null'::jsonb
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   base||'{"retirada_en":"2026-10-20T00:00:00Z"}'::jsonb)->>'retirada_en'
   IS DISTINCT FROM '2026-10-20T00:00:00Z'
 THEN RAISE EXCEPTION 'AUT106: instante explícito reescrito' USING ERRCODE='55000'; END IF;
 IF vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(base||'{"otro":"conservar"}'::jsonb)->>'otro'
    IS DISTINCT FROM 'conservar'
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
   jsonb_set(base,'{concesiones,0,accion}','"distinta"'::jsonb))#>>'{concesiones,0,accion}'
    IS DISTINCT FROM 'distinta'
 THEN RAISE EXCEPTION 'AUT106: campo o concesión alterada en normalización' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
