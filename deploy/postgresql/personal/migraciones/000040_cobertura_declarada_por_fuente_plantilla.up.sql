\set ON_ERROR_STOP on
-- Personal40. La consulta de vacantes B2 (Personal18) solo lista plazas de una
-- plantilla publicada cuya cobertura de ocupaciones conste «completa». Hasta
-- ahora el disparador de Personal17 rechazaba siempre ese estado, de modo que
-- ninguna plantilla podía ofrecer vacantes.
--
-- Regla nueva, cerrada: una revisión «completa» solo se admite si la declara
-- la misma fuente que publicó la plantilla (misma revisión, organismo,
-- fuente_ref, huella y acto) y ninguna plaza de esa plantilla tiene todavía
-- ocupación registrada. Las ocupaciones posteriores entran por el alta B2 y la
-- consulta de vacantes las descuenta. Los demás estados y la continuidad de
-- revisiones no cambian. Solo el propietario de Personal escribe en la tabla
-- (RLS y sin permisos para la aplicación); no se concede nada nuevo.
--
-- Sustitución en sitio con preimagen exacta de Personal17. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000040',0));
DO $pre$
DECLARE f regprocedure:=pg_catalog.to_regprocedure('vec_personal.validar_revision_cobertura_ocupaciones_v1()');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal40: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal40: PARO clave=PG actual=% esperado=180000..189999',pg_catalog.current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF f IS NULL OR pg_catalog.to_regclass('vec_personal.cobertura_ocupaciones_historia') IS NULL
 OR pg_catalog.to_regclass('vec_personal.version_plantilla_historia') IS NULL
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 THEN RAISE EXCEPTION 'Personal40: PARO clave=preimagen actual=incompatible esperado=Personal17' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
   IS DISTINCT FROM 'f77a382997b709e09d88496b4ef862bc6a646b96b3aed38107f3b905716611cf'
 THEN RAISE EXCEPTION 'Personal40: PARO clave=validar_cobertura actual=distinta esperado=Personal17' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_personal_propietario;
CREATE OR REPLACE FUNCTION vec_personal.validar_revision_cobertura_ocupaciones_v1()
 RETURNS trigger
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE previa record;
BEGIN
 -- Personal40: «completa» solo la declara la fuente que publicó la plantilla.
 IF NEW.estado='completa' AND NOT EXISTS (
  SELECT 1 FROM vec_personal.version_plantilla_historia p
   WHERE p.version_ref=NEW.plantilla_version_ref AND p.revision=NEW.plantilla_revision
     AND p.organismo_ref=NEW.organismo_ref AND p.estado='publicada' AND NOT p.retirado
     AND p.fuente_ref=NEW.fuente_ref AND p.huella_fuente_sha256=NEW.fuente_huella_sha256
     AND p.acto_ref=NEW.acto_ref
 ) OR NEW.estado='completa' AND EXISTS (
  -- Ninguna ocupación anterior sobre esa plantilla: lo que se declara
  -- completo es una plantilla sin ocupar; las altas posteriores las descuenta
  -- la consulta de vacantes.
  SELECT 1 FROM vec_personal.ocupacion_empleado_historia o
   JOIN vec_personal.plaza_plantilla_historia pl ON pl.plaza_ref=o.plaza_ref
   WHERE pl.plantilla_version_ref=NEW.plantilla_version_ref
 ) THEN
  RAISE EXCEPTION 'cobertura de ocupaciones no acreditada' USING ERRCODE='P7401';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cobertura-ocupaciones:'||NEW.cobertura_ref::text,0));
 SELECT revision,organismo_ref,plantilla_version_ref,conocido_desde,estado INTO previa
   FROM vec_personal.cobertura_ocupaciones_historia
   WHERE cobertura_ref=NEW.cobertura_ref ORDER BY revision DESC LIMIT 1;
 NEW.conocido_desde:=clock_timestamp();
 IF NOT FOUND THEN
  IF NEW.revision<>1 THEN RAISE EXCEPTION 'cobertura inicial inválida' USING ERRCODE='23505'; END IF;
 ELSE
  IF NEW.revision<>previa.revision+1 OR NEW.organismo_ref<>previa.organismo_ref
     OR NEW.plantilla_version_ref<>previa.plantilla_version_ref
     OR previa.estado='revocada' THEN
   RAISE EXCEPTION 'continuidad de cobertura inválida' USING ERRCODE='23505';
  END IF;
  NEW.conocido_desde:=greatest(NEW.conocido_desde,previa.conocido_desde + interval '1 microsecond');
 END IF;
 IF NOT EXISTS (
  SELECT 1 FROM vec_personal.version_plantilla_historia p
   WHERE p.version_ref=NEW.plantilla_version_ref AND p.revision=NEW.plantilla_revision
     AND p.organismo_ref=NEW.organismo_ref AND p.estado='publicada' AND NOT p.retirado
     AND p.vigente_desde<=NEW.vigente_desde
     AND (p.vigente_hasta IS NULL OR NEW.vigente_hasta<=p.vigente_hasta)
 ) THEN RAISE EXCEPTION 'plantilla de cobertura no publicada' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $function$;
RESET ROLE;

DO $post$
DECLARE f regprocedure:='vec_personal.validar_revision_cobertura_ocupaciones_v1()'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=f AND (x.grantee<>p.proowner OR x.is_grantable))
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(f),'p.fuente_ref=NEW.fuente_ref AND p.huella_fuente_sha256=NEW.fuente_huella_sha256')=0
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger WHERE tgrelid='vec_personal.cobertura_ocupaciones_historia'::regclass
   AND tgname='revision_continua' AND tgfoid=f AND tgenabled='O')
 THEN RAISE EXCEPTION 'Personal40: PARO clave=postimagen actual=divergente esperado=cobertura_por_fuente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
