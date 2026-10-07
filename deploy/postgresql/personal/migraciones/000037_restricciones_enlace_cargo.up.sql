\set ON_ERROR_STOP on
-- Personal37: dos CHECK de enlace_cargo_competencial_historia (Personal28)
-- usan la repetición {2,511}, que PostgreSQL no admite (máximo 255): la regla
-- se compila al evaluarla y todo INSERT falla con «invalid regular expression:
-- invalid repetition count(s)», así que ningún enlace de cargo se podía
-- publicar. Se sustituyen por la misma intención —primer carácter, alfabeto y
-- longitud total de 3 a 512— con la longitud medida aparte. No toca la
-- fachada ni otras tablas. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000037',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal37: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal37: PARO clave=PG actual=% esperado=180000..189999',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_personal.enlace_cargo_competencial_historia') IS NULL
 OR (SELECT relowner FROM pg_class WHERE oid='vec_personal.enlace_cargo_competencial_historia'::regclass)<>'vec_personal_propietario'::regrole
 THEN RAISE EXCEPTION 'Personal37: PARO clave=Personal28 actual=ausente esperado=tabla_de_enlaces' USING ERRCODE='55000'; END IF;
 -- Preimagen exacta: las dos reglas defectuosas, tal como las dejó Personal28.
 IF (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='vec_personal.enlace_cargo_competencial_historia'::regclass
     AND conname='enlace_cargo_competencial_historia_recurso_ref_check') IS DISTINCT FROM
     'CHECK ((recurso_ref ~ ''^[a-z][a-z0-9_.:/#-]{2,511}$''::text))'
 OR (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='vec_personal.enlace_cargo_competencial_historia'::regclass
     AND conname='enlace_cargo_competencial_historia_finalidad_ref_check') IS DISTINCT FROM
     'CHECK ((finalidad_ref ~ ''^[a-z][a-z0-9_.:-]{2,511}$''::text))'
 THEN RAISE EXCEPTION 'Personal37: PARO clave=preimagen actual=distinta esperado=CHECK_Personal28_con_2_511' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
-- La tabla no puede tener filas (todo INSERT fallaba); la validación de las
-- reglas nuevas lo confirma igualmente al añadirlas.
ALTER TABLE vec_personal.enlace_cargo_competencial_historia
 DROP CONSTRAINT enlace_cargo_competencial_historia_recurso_ref_check,
 DROP CONSTRAINT enlace_cargo_competencial_historia_finalidad_ref_check,
 ADD CONSTRAINT enlace_cargo_competencial_historia_recurso_ref_check
  CHECK (recurso_ref ~ '^[a-z][a-z0-9_.:/#-]+$' AND char_length(recurso_ref) BETWEEN 3 AND 512),
 ADD CONSTRAINT enlace_cargo_competencial_historia_finalidad_ref_check
  CHECK (finalidad_ref ~ '^[a-z][a-z0-9_.:-]+$' AND char_length(finalidad_ref) BETWEEN 3 AND 512);
RESET ROLE;
DO $post$
DECLARE n int;
BEGIN
 SELECT count(*) INTO n FROM pg_constraint WHERE conrelid='vec_personal.enlace_cargo_competencial_historia'::regclass
  AND contype='c' AND convalidated AND conname IN('enlace_cargo_competencial_historia_recurso_ref_check','enlace_cargo_competencial_historia_finalidad_ref_check')
  AND pg_get_constraintdef(oid) !~ '\{[0-9]+,[0-9]+\}';
 IF n<>2 THEN RAISE EXCEPTION 'Personal37: PARO clave=postimagen actual=% esperado=2_reglas_validas',n USING ERRCODE='55000'; END IF;
 -- Las reglas nuevas se evalúan sin error: un valor válido pasa y uno con
 -- mayúsculas no (se comprueba la expresión, no se inserta nada).
 IF NOT ('documento_contratacion_temporal' ~ '^[a-z][a-z0-9_.:/#-]+$' AND char_length('documento_contratacion_temporal') BETWEEN 3 AND 512)
 OR ('Documento' ~ '^[a-z][a-z0-9_.:/#-]+$')
 THEN RAISE EXCEPTION 'Personal37: PARO clave=evaluacion actual=distinta esperado=reglas_evaluables' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
