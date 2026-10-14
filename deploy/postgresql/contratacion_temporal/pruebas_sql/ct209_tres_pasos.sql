\set ON_ERROR_STOP on
BEGIN;
-- Copia sólo las restricciones que se han ampliado, tomadas del motor.
-- No crea firmas, recibos ni actúa sobre un expediente de la aplicación.
CREATE TEMP TABLE revision_ct209 (
  orden_pdf integer,
  firma_ref text,
  firma_anterior_ref text,
  recibo_anterior_ref text
) ON COMMIT DROP;
DO $comprobar$
DECLARE r record; aceptadas integer;
BEGIN
  FOR r IN SELECT conname, pg_catalog.pg_get_constraintdef(oid) AS definicion
    FROM pg_catalog.pg_constraint
    WHERE conrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
      AND conname IN ('firma_documento_revision_pdf_v2_orden_pdf_check', 'firma_documento_revision_pdf_v2_check6')
  LOOP
    EXECUTE pg_catalog.format('ALTER TABLE revision_ct209 ADD CONSTRAINT %I %s', r.conname, r.definicion);
  END LOOP;
  SELECT count(*) INTO aceptadas FROM pg_catalog.pg_constraint WHERE conrelid='revision_ct209'::regclass;
  IF aceptadas<>2 THEN RAISE EXCEPTION 'CT209 clave=restricciones esperado=2 obtenido=%',aceptadas; END IF;
  INSERT INTO revision_ct209 VALUES (1,'firma:primera',NULL,NULL),
    (2,'firma:segunda','firma:primera','recibo:primera'),
    (3,'firma:tercera','firma:segunda','recibo:segunda');
  BEGIN
    INSERT INTO revision_ct209 VALUES (4,'firma:cuarta','firma:tercera','recibo:tercera');
    RAISE EXCEPTION 'CT209 cuarto_paso_admitido';
  EXCEPTION WHEN check_violation THEN NULL;
  END;
  BEGIN
    INSERT INTO revision_ct209 VALUES (3,'firma:sin_antecedente',NULL,NULL);
    RAISE EXCEPTION 'CT209 antecedente_omitido';
  EXCEPTION WHEN check_violation THEN NULL;
  END;
  FOR r IN SELECT oid,proname,proconfig,proacl,proowner,prosrc FROM pg_catalog.pg_proc
    WHERE oid IN (
      'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure)
  LOOP
    IF NOT r.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']
      OR pg_catalog.pg_get_userbyid(r.proowner)<>'vec_contratacion_temporal_propietario'
      OR r.proacl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'::aclitem[] THEN
      RAISE EXCEPTION 'CT209 clave=contrato_% esperado=propietario_acl_search_path obtenido=%',r.proname,r.proconfig;
    END IF;
    IF position('revision_anterior.evidencia_firmas_canonica IS DISTINCT FROM' IN r.prosrc)=0
      OR position('anterior.paso_orden<>(s->>''OrdenFirmaPDF'')::integer-1' IN r.prosrc)=0 THEN
      RAISE EXCEPTION 'CT209 clave=cadena_% esperado=antecedente_inmediato_prefijo_completo obtenido=ausente',r.proname;
    END IF;
  END LOOP;
END;
$comprobar$;
ROLLBACK;
