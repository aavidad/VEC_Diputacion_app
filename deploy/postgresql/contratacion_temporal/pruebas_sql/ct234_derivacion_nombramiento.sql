\set ON_ERROR_STOP on
-- Comprobación focal en copia local tras CT234. Sin firmas ni decisiones dobles.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
DO $test$
DECLARE f oid:=to_regprocedure(
 'vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 estado text; mensaje text; fuente_sql text; fila record;
BEGIN
 SELECT prosrc INTO STRICT fuente_sql FROM pg_proc WHERE oid=f;
 IF strpos(fuente_sql,'padre_canonico')>0
    OR strpos(fuente_sql,'json_each(p_material::json))<>12')=0
    OR strpos(fuente_sql,'preimagen_padre:=actual.agregado_json::text')=0
    OR strpos(fuente_sql,'preimagen_padre_json')=0
    OR strpos(fuente_sql,'octet_length(preimagen_padre)>262144')=0
    OR strpos(fuente_sql,'sha256(convert_to(preimagen_padre,''UTF8''))')=0
    OR strpos(fuente_sql,'preimagen_padre:=actual.agregado_json::text')<
       strpos(fuente_sql,'registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada')
 THEN RAISE EXCEPTION 'CT234: material o lectura previa a V3 divergente'; END IF;
 IF f IS NULL OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.privilege_type='EXECUTE'
         AND a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_ejecutor'::regrole))
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
       WHERE c.conrelid='vec_contratacion_temporal.publicacion_version_rrhh'::regclass
         AND c.convalidated AND strpos(pg_get_constraintdef(c.oid),'-P(100|[1-9][0-9]?)')>0)
    OR (SELECT count(*) FROM pg_trigger t
       WHERE t.tgrelid='vec_contratacion_temporal.expediente_integral_actual'::regclass
         AND t.tgname='impedir_cierre_padre_con_hijos_ct234'
         AND NOT t.tgisinternal AND t.tgenabled='O')<>1
 THEN RAISE EXCEPTION 'CT234: instalación o ACL incompleta'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p
    WHERE p.oid IN (
      to_regprocedure('vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'),
      to_regprocedure('vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()'),
      to_regprocedure('vec_contratacion_temporal.registrar_instantanea_fase_regla_v1()'),
      to_regprocedure('vec_contratacion_temporal.proteger_version_circuito_ct163()'),
      to_regprocedure('vec_contratacion_temporal.proteger_version_circuito_ct164()'))
      AND (NOT p.prosecdef OR NOT EXISTS(SELECT 1 FROM unnest(p.proconfig) cfg
        WHERE replace(cfg,' ','')='search_path=pg_catalog,pg_temp')))
 THEN RAISE EXCEPTION 'CT234: función de publicación sin frontera fija'; END IF;

 -- Si el recorrido ya produjo hijos, exige publicación, inicio de fase y
 -- captura de regla propios de v6, unidos al mismo recibo del hijo.
 IF EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.nombramiento_derivado_v1 d
  JOIN vec_contratacion_temporal.expediente_alta a ON a.expediente_ref=d.hijo_ref
  LEFT JOIN vec_contratacion_temporal.expediente_version_integral v
    ON v.expediente_ref=d.hijo_ref AND v.version=6
  LEFT JOIN vec_contratacion_temporal.publicacion_version_rrhh p
    ON p.expediente_ref=d.hijo_ref AND p.version=6
  LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh fase
    ON fase.expediente_ref=d.hijo_ref AND fase.version=6
  LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 regla
    ON regla.expediente_ref=d.hijo_ref AND regla.version=6
  LEFT JOIN vec_contratacion_temporal.expediente_version_integral fuente
    ON fuente.expediente_ref=d.padre_ref AND fuente.version=d.version_padre
  WHERE a.derivacion_ref IS DISTINCT FROM d.derivacion_ref
    OR v.origen_version IS DISTINCT FROM 'derivacion_nombramiento_ct234'
    OR v.agregado_json IS DISTINCT FROM d.recibo_json->'expediente'
    OR v.agregado_json#>'{instruccion_compartida,expediente_padre}' IS DISTINCT FROM fuente.agregado_json
    OR v.agregado_json#>>'{instruccion_compartida,preimagen_padre_json}' IS DISTINCT FROM fuente.agregado_json::text
    OR v.agregado_json#>>'{instruccion_compartida,huella_padre_sha256}' IS DISTINCT FROM
       encode(sha256(convert_to(fuente.agregado_json::text,'UTF8')),'hex')
    OR fuente.agregado_json_huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(fuente.agregado_json::text,'UTF8')),'hex')
    OR p.numero_visible IS DISTINCT FROM a.numero_visible
    OR fase.fase_clave IS DISTINCT FROM v.fase_clave
    OR fase.fase_desde IS DISTINCT FROM p.actualizado_en
    OR regla.fase_desde IS DISTINCT FROM fase.fase_desde
 ) THEN RAISE EXCEPTION 'CT234: hijo v6 sin publicación completa'; END IF;
 FOR fila IN SELECT d.hijo_ref,v.agregado_json
  FROM vec_contratacion_temporal.nombramiento_derivado_v1 d
  JOIN vec_contratacion_temporal.expediente_version_integral v
    ON v.expediente_ref=d.hijo_ref AND v.version=6 LOOP
  IF vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
       fila.hijo_ref,fila.agregado_json) IS NOT TRUE
     OR vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
       fila.hijo_ref,jsonb_set(fila.agregado_json,
        '{instruccion_compartida,preimagen_padre_json}','"{}"'::jsonb)) IS NOT FALSE
  THEN RAISE EXCEPTION 'CT234: preimagen alterada aceptada'; END IF;
 END LOOP;
 IF EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.nombramiento_derivado_v1 d
  JOIN vec_contratacion_temporal.expediente_integral_actual hijo
    ON hijo.expediente_ref=d.hijo_ref
  JOIN vec_contratacion_temporal.expediente_version_integral hv
    ON hv.expediente_ref=hijo.expediente_ref AND hv.version=hijo.version
  JOIN vec_contratacion_temporal.expediente_integral_actual padre
    ON padre.expediente_ref=d.padre_ref
  JOIN vec_contratacion_temporal.expediente_version_integral pv
    ON pv.expediente_ref=padre.expediente_ref AND pv.version=padre.version
  WHERE (hv.estado='en_curso' OR hv.agregado_json->>'estado_actual'='en_curso')
    AND pv.estado IN ('completado','cancelado')
 ) THEN RAISE EXCEPTION 'CT234: padre terminal con hijo abierto'; END IF;

 BEGIN
  PERFORM vec_contratacion_temporal.derivar_nombramiento_ct234_v1(
   '{}'::text,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT234: llamada directa aceptada';
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  IF estado<>'42501' OR mensaje<>'CT234: sesión denegada' THEN
   RAISE EXCEPTION 'CT234: denegación inesperada'; END IF;
 END;
END $test$;
ROLLBACK;
