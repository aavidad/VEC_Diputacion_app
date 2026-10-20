\set ON_ERROR_STOP on
-- Dato de copia R: grupos nominales conservados por AD278. Nunca aplicar fuera de esta copia.
BEGIN;
DO $dato$ DECLARE x record;n text;vistos text[]:=ARRAY[]::text[];
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.vinculo_emision_bolsa.v1','contratacion_temporal.bolsa.vincular','vec_contratacion_temporal_ejecutor'),
  ('vec_bolsa_llamamientos.llamamiento.emitir.v1','llamamiento.emitir.v1','vec_bolsa_llamamientos_ejecutor')
 ) v(audiencia,operacion,grupo) LOOP
  SELECT login_nombre INTO STRICT n FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE audiencia_consumo=x.audiencia AND operacion=x.operacion AND canal_permitido='interna_corporativa';
  IF n !~ '^vec_[a-z0-9_]+$' THEN RAISE EXCEPTION 'dato de copia: nombre ajeno'; END IF;
  IF to_regrole(n) IS NULL THEN EXECUTE format('CREATE ROLE %I NOLOGIN',n); END IF;
  IF n=ANY(vistos) OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=to_regrole(n) AND roleid<>to_regrole(x.grupo))
  THEN RAISE EXCEPTION 'dato de copia: identidades técnicas incompatibles'; END IF;
  EXECUTE format('ALTER ROLE %I LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS VALID UNTIL %L',n,clock_timestamp()+interval '90 days');
  EXECUTE format('GRANT %I TO %I WITH ADMIN FALSE,INHERIT TRUE,SET FALSE',x.grupo,n);
  vistos:=array_append(vistos,n);
 END LOOP;
END $dato$;
COMMIT;
