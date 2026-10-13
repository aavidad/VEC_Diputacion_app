\set ON_ERROR_STOP on
-- Ejecutar tras publicar el segundo vínculo F1 en una cuenta sintética.
-- El ROLLBACK conserva historia, auditoría y recibos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE cuenta text; r record; primero text; segundo text; j jsonb; n integer:=0;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint
  WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
   AND conname='vinculo_corporativo_actual_pk'
   AND pg_get_constraintdef(oid,false)='PRIMARY KEY (cuenta_ref, perfil_ref, superficie, uso)')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint
  WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
   AND conname='vinculo_corporativo_actual_version_fk'
   AND pg_get_constraintdef(oid,false)='FOREIGN KEY (cuenta_ref, perfil_ref, superficie, uso, vinculo_corporativo_ref, version) REFERENCES vec_contexto_actor_v1.vinculo_corporativo_versiones(cuenta_ref, perfil_ref, superficie, uso, vinculo_corporativo_ref, version) MATCH FULL')
 THEN RAISE EXCEPTION 'CA43 prueba: PK/FK por perfil ausentes'; END IF;
 SELECT a.cuenta_ref INTO cuenta FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 WHERE a.superficie='interna_corporativa' AND a.uso='consulta_rrhh'
 GROUP BY a.cuenta_ref HAVING count(DISTINCT a.perfil_ref)>=2
 ORDER BY a.cuenta_ref LIMIT 1;
 IF cuenta IS NULL THEN RAISE EXCEPTION 'CA43 prueba: faltan dos perfiles sintéticos en una cuenta'; END IF;
 FOR r IN
  SELECT a.cuenta_ref,a.perfil_ref,cv.persona_ref,cv.vinculo_contexto_ref,
   cv.cuenta_version,cv.persona_version,cv.perfil_version,
   cv.vinculo_contexto_version,cv.vigente_hasta,cv.estado,
   prf.plan_sha256,prf.auditoria_ref
  FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
  JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
   ON (cv.cuenta_ref,cv.perfil_ref,cv.superficie,cv.uso,cv.vinculo_corporativo_ref,cv.version)=
      (a.cuenta_ref,a.perfil_ref,a.superficie,a.uso,a.vinculo_corporativo_ref,a.version)
  JOIN vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 prf
   ON prf.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND prf.version=a.version
  WHERE a.cuenta_ref=cuenta AND a.superficie='interna_corporativa' AND a.uso='consulta_rrhh'
  ORDER BY a.perfil_ref LIMIT 2
 LOOP
  n:=n+1;
  IF r.estado<>'activo' OR r.plan_sha256 !~ '^[0-9a-f]{64}$' OR r.auditoria_ref IS NULL
  THEN RAISE EXCEPTION 'CA43 prueba: vínculo o recibo inválido'; END IF;
  IF n=1 THEN primero:=r.perfil_ref; ELSE segundo:=r.perfil_ref; END IF;
  j:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','estado','activo',
   'cuenta_ref',r.cuenta_ref,'persona_ref',r.persona_ref,'perfil_activo_ref',r.perfil_ref,
   'contexto_actor_ref',r.vinculo_contexto_ref,'cuenta_version',r.cuenta_version::text,
   'persona_version',r.persona_version::text,'perfil_version',r.perfil_version::text,
   'contexto_version',r.vinculo_contexto_version::text,'vigente_hasta',r.vigente_hasta::text);
  IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to(j::text,'UTF8')) IS NOT TRUE
  THEN RAISE EXCEPTION 'CA43 prueba: cotejo positivo del perfil % falló',n; END IF;
  IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to(
   jsonb_set(j,'{vigente_hasta}',to_jsonb((clock_timestamp()-interval '1 second')::text))::text,'UTF8')) IS NOT FALSE
  THEN RAISE EXCEPTION 'CA43 prueba: contexto caducado admitido'; END IF;
 END LOOP;
 IF n<>2 OR primero=segundo THEN RAISE EXCEPTION 'CA43 prueba: selección de perfiles no exacta'; END IF;
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to(
  jsonb_set(j,'{perfil_activo_ref}',to_jsonb(primero))::text,'UTF8')) IS NOT FALSE
 THEN RAISE EXCEPTION 'CA43 prueba: perfil cruzado admitido'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
  JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones v
   ON v.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND v.version=a.version
  WHERE a.cuenta_ref=cuenta AND a.perfil_ref IS DISTINCT FROM v.perfil_ref)
 THEN RAISE EXCEPTION 'CA43 prueba: cruce de historia y puntero'; END IF;
END $prueba$;
ROLLBACK;
