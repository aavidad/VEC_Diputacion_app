\set ON_ERROR_STOP on
-- Regresión CT85: sólo catálogos y bloque efímero, sin funciones persistentes.
-- Reproduce la colisión anterior y recorre el guard corregido con p aún presente.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='5s';
DO $regresion_ct85$
#variable_conflict error
DECLARE
 p vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 cantidad bigint; detectada boolean:=false; fuente text;
BEGIN
 BEGIN
  SELECT count(*) INTO cantidad FROM pg_policy p
   WHERE p.polrelid='vec_contratacion_temporal.incorporacion_registro_v2'::regclass;
 EXCEPTION WHEN undefined_column THEN
  IF SQLERRM IS DISTINCT FROM 'record "p" has no field "polrelid"' THEN RAISE; END IF;
  detectada:=true;
 END;
 IF NOT detectada THEN
  RAISE EXCEPTION 'CT85 regresión: colisión anterior no reproducida'; END IF;
 FOREACH fuente IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2',
  'incorporacion_outbox_v2','seguimiento_raiz_v2','seguimiento_definicion_v2','seguimiento_estado_v2'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace
  AND c.relname=fuente AND c.relowner='vec_contratacion_temporal_propietario'::regrole AND c.relkind='r' AND NOT c.relispartition
  AND c.relrowsecurity AND c.relforcerowsecurity
  AND (SELECT count(*) FROM pg_policy localizador_politica WHERE localizador_politica.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy localizador_politica WHERE localizador_politica.polrelid=c.oid AND localizador_politica.polname='propietario'
   AND localizador_politica.polcmd='*' AND localizador_politica.polpermissive AND localizador_politica.polroles=ARRAY['vec_contratacion_temporal_propietario'::regrole::oid]
   AND pg_get_expr(localizador_politica.polqual,localizador_politica.polrelid)='true' AND pg_get_expr(localizador_politica.polwithcheck,localizador_politica.polrelid)='true'))
  THEN RAISE EXCEPTION 'CT85 regresión: guard corregido no admite catálogo original'; END IF;
 END LOOP;
END $regresion_ct85$;
ROLLBACK;
