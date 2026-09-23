\set ON_ERROR_STOP on
-- Sólo se admite sobre postimagen sin concesiones, consumos ni registros F2.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000053',0));
DO $guardia$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='vec_identidad_sesiones_v1' AND c.relname='registro_propio_v1')
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec.registro_propio.v1')
 THEN RAISE EXCEPTION 'AD3-53: historia o dependencia impide DOWN' USING ERRCODE='55000'; END IF;
END $guardia$;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_identidad_sesiones_v1_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 actual text; anterior text;
 guarda text:=$g$/* AD3-51 GUARDA INICIO */ (CASE WHEN p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$g$;
 guarda_nueva text:=$g$/* AD3-51 GUARDA INICIO */ (CASE WHEN p_perfil_mutacion='registro_propio' THEN vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1() IS NOT TRUE WHEN p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$g$;
 excl text:=$e$               AND p_perfil_mutacion IS DISTINCT FROM 'acceso_rutas_dietas'$e$;
 excl_nueva text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''registro_propio''';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_propio'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.registro_propio.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.registro_propio.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'vec.registro_propio.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'registro_propio'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'alta_vec_propia'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF length(actual)-length(replace(actual,guarda_nueva,''))<>length(guarda_nueva)
 OR length(actual)-length(replace(actual,excl_nueva,''))<>length(excl_nueva)
 OR length(actual)-length(replace(actual,extension||marca,''))<>length(extension||marca)
 THEN RAISE EXCEPTION 'AD3-53: postimagen divergente' USING ERRCODE='55000'; END IF;
 anterior:=replace(replace(replace(actual,extension||marca,marca),excl_nueva,excl),guarda_nueva,guarda);
 EXECUTE anterior;
 IF pg_get_functiondef(f) IS DISTINCT FROM anterior
 THEN RAISE EXCEPTION 'AD3-53: reversión divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;
DROP FUNCTION vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1();
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE def text; valores text[]; canon text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
 SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.-]+)''::text','g') WITH ORDINALITY x(m,n);
 canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
 SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
 IF def IS DISTINCT FROM canon OR valores[cardinality(valores)] IS DISTINCT FROM 'vec.registro_propio.v1'
 THEN RAISE EXCEPTION 'AD3-53: audiencia postimagen divergente' USING ERRCODE='55000'; END IF;
 valores:=valores[1:cardinality(valores)-1];
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
 array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;
COMMIT;
