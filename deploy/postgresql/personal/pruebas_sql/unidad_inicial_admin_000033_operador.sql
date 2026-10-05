\set ON_ERROR_STOP on
-- Conexión REAL del LOGIN técnico aprobado; no SET ROLE ni config/GRANT aquí.
-- Dirección carga GUC privados por parámetros enlazados. El plan y la fuente
-- proceden del preparador y de su aprobación externa. Sólo clon; ROLLBACK final.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pruebas$
DECLARE plan text:=current_setting('vec.ensayo.plan_unidad_inicial');fuente text:=current_setting('vec.ensayo.fuente_unidad_inicial');
 sha text:=encode(sha256(convert_to(plan,'UTF8')),'hex');r jsonb;repetido jsonb;negativo jsonb;caso text;anterior numeric;
BEGIN
 r:=vec_personal.inicializar_unidad_sintetica_admin_v1(plan,sha,fuente);
 IF r->>'estado' IS DISTINCT FROM 'permitido' OR r->>'replay' IS DISTINCT FROM 'false'
 OR r#>>'{recibo,esquema}' IS DISTINCT FROM 'vec.personal.unidad-inicial.v1'
 OR r#>>'{recibo,alcance_fuente}' IS DISTINCT FROM 'sintetico_declarado'
 OR r#>>'{recibo,unidad,revision}' IS DISTINCT FROM '1'
 OR r#>>'{recibo,unidad,nodo_ref}' IS DISTINCT FROM plan::jsonb#>>'{unidad,nodo_ref}'
 OR r#>>'{recibo,auditoria_ref}' IS NULL OR r#>>'{auditoria_intento,auditoria_ref}' IS NULL
 THEN RAISE EXCEPTION 'Personal33 operador: positivo no conserva unidad/recibo/auditoría reales'; END IF;
 repetido:=vec_personal.inicializar_unidad_sintetica_admin_v1(plan,sha,fuente);
 IF repetido->>'estado' IS DISTINCT FROM 'permitido' OR repetido->>'replay' IS DISTINCT FROM 'true'
 OR repetido->'recibo' IS DISTINCT FROM r->'recibo'
 OR repetido#>>'{auditoria_intento,auditoria_ref}'=r#>>'{auditoria_intento,auditoria_ref}'
 OR (repetido#>>'{auditoria_intento,secuencia}')::numeric<>(r#>>'{auditoria_intento,secuencia}')::numeric+1
 THEN RAISE EXCEPTION 'Personal33 operador: replay duplica efecto o reutiliza intento'; END IF;
 anterior:=(repetido#>>'{auditoria_intento,secuencia}')::numeric;
 FOREACH caso IN ARRAY ARRAY['aprobacion','fuente','produccion','campo_ajeno','nombre_con_blanco'] LOOP
  CASE caso
  WHEN 'aprobacion' THEN negativo:=vec_personal.inicializar_unidad_sintetica_admin_v1(plan,repeat('0',64),fuente);
  WHEN 'fuente' THEN negativo:=vec_personal.inicializar_unidad_sintetica_admin_v1(plan,sha,fuente||' ');
  WHEN 'produccion' THEN negativo:=vec_personal.inicializar_unidad_sintetica_admin_v1(jsonb_set(plan::jsonb,'{entorno}','"produccion"')::text,sha,fuente);
  WHEN 'campo_ajeno' THEN negativo:=vec_personal.inicializar_unidad_sintetica_admin_v1((plan::jsonb||'{"perfil":"administrador"}'::jsonb)::text,sha,fuente);
  WHEN 'nombre_con_blanco' THEN negativo:=vec_personal.inicializar_unidad_sintetica_admin_v1(jsonb_set(plan::jsonb,'{unidad,denominacion}',to_jsonb(' '||(plan::jsonb#>>'{unidad,denominacion}')))::text,sha,fuente);
  END CASE;
  IF negativo->>'estado' IS DISTINCT FROM 'denegado' OR negativo->'recibo' IS DISTINCT FROM 'null'::jsonb
  OR negativo->>'replay' IS DISTINCT FROM 'false' OR negativo#>>'{auditoria_intento,auditoria_ref}' IS NULL
  OR (negativo#>>'{auditoria_intento,secuencia}')::numeric<>anterior+1
  THEN RAISE EXCEPTION 'Personal33 operador: rechazo sin auditoría nominal o con falsa confirmación'; END IF;
  anterior:=(negativo#>>'{auditoria_intento,secuencia}')::numeric;
 END LOOP;
 IF vec_personal.inicializar_unidad_sintetica_admin_v1(plan,sha,fuente)->'recibo' IS DISTINCT FROM r->'recibo'
 THEN RAISE EXCEPTION 'Personal33 operador: negativos modificaron recibo original'; END IF;
END $pruebas$;
ROLLBACK;
