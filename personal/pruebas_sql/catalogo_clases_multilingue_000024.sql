\set ON_ERROR_STOP on
-- Ensayo en el clon propio tras Personal24. Todo cambio de datos se revierte.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $focal$
DECLARE original jsonb; actual jsonb; c vec_personal.clases_ocupacion_plan_ct_catalogo%ROWTYPE;
 nuevo jsonb; invalido jsonb; canon text; siguiente bigint; caso integer; cuentas bigint;
BEGIN
 SELECT jsonb_agg(to_jsonb(t) ORDER BY ref,version) INTO original FROM vec_personal.clases_ocupacion_plan_ct_catalogo t;
 SELECT * INTO STRICT c FROM vec_personal.clases_ocupacion_plan_ct_catalogo
 WHERE ref='personal:incorporacion_ct:clases_ocupacion' ORDER BY version DESC LIMIT 1;
 siguiente:=c.version+1;
 -- Se añade un idioma a los textos de una versión nueva, sin retocar ninguna
 -- etiqueta ni huella del catálogo original.
 SELECT jsonb_build_object('opciones',jsonb_agg(
   jsonb_set(o,'{etiquetas}',(o->'etiquetas')||jsonb_build_object('fr','Libellé')) ORDER BY ord))
 INTO nuevo FROM jsonb_array_elements(c.datos->'opciones') WITH ORDINALITY x(o,ord);
 canon:=nuevo::text;
 INSERT INTO vec_personal.clases_ocupacion_plan_ct_catalogo VALUES(c.ref,siguiente,canon,nuevo,
   encode(sha256(convert_to(canon,'UTF8')),'hex'),clock_timestamp());
 SELECT jsonb_agg(to_jsonb(t) ORDER BY ref,version) INTO actual
 FROM vec_personal.clases_ocupacion_plan_ct_catalogo t WHERE ref<>c.ref OR version<siguiente;
 IF actual IS DISTINCT FROM original THEN
  RAISE EXCEPTION 'Personal24: se modificó el catálogo anterior'; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_personal.clases_ocupacion_plan_ct_catalogo t
   WHERE ref=c.ref AND version=siguiente AND datos->'opciones'->0->'etiquetas'->>'fr'='Libellé') THEN
  RAISE EXCEPTION 'Personal24: no se conservó el tercer idioma'; END IF;
 SELECT count(*) INTO cuentas FROM vec_personal.clases_ocupacion_plan_ct_catalogo;
 FOR caso IN 1..6 LOOP
  invalido:=CASE caso
    WHEN 1 THEN jsonb_set(nuevo,'{opciones,0,etiquetas}',jsonb_build_object('idioma_invalido','Texto'))
    WHEN 2 THEN jsonb_set(nuevo,'{opciones,0,etiquetas}','{}'::jsonb)
    WHEN 3 THEN jsonb_set(nuevo,'{opciones,0,etiquetas}',jsonb_build_object('fr',7))
    WHEN 4 THEN jsonb_set(nuevo,'{opciones,0,etiquetas}',jsonb_build_object('fr',''))
    WHEN 5 THEN jsonb_set(nuevo,'{opciones,0,etiquetas}',jsonb_build_object('fr',repeat('a',81)))
    ELSE jsonb_build_object('opciones',jsonb_build_array(jsonb_build_object(
      'valor','reserva','texto_clave','rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.reserva',
      'etiquetas',jsonb_build_object('es','Reserva','en','Reserved','fr','Reserve')))) END;
  canon:=invalido::text;
  BEGIN
   INSERT INTO vec_personal.clases_ocupacion_plan_ct_catalogo VALUES(c.ref,siguiente+1,canon,invalido,
     encode(sha256(convert_to(canon,'UTF8')),'hex'),clock_timestamp());
   RAISE EXCEPTION 'Personal24: etiquetas inválidas admitidas, caso %',caso;
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 END LOOP;
 IF (SELECT count(*) FROM vec_personal.clases_ocupacion_plan_ct_catalogo)<>cuentas THEN
  RAISE EXCEPTION 'Personal24: rechazo dejó datos residuales'; END IF;
END $focal$;
ROLLBACK;
