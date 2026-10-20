\set ON_ERROR_STOP on
-- Datos ficticios del recorrido: reutiliza la participación y el orden del acta.
-- No constituye otra bolsa, no modifica la importación ni reescribe historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:datos:hz39:antonio:disponible',0));
DO $permiso$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=rol actual=% esperado=superusuario',current_user USING ERRCODE='42501';
 END IF;
END $permiso$;
-- La identidad de Antonio (cuenta, persona, perfil y vínculo con esta
-- candidatura) la prepara preparar_candidato_desarrollo.sh con
-- --candidato-ref (F1-99). Este guion ya no mueve el vínculo de una persona
-- anterior: su permiso de Mi Bolsa seguiría ligado a la candidatura antigua.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $datos$
DECLARE
 bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 candidato constant text:='can_Vxd6XlZE9ctNzw38AUSmgQbCcRyoFKhaojDaFsEKC1M';
 participacion constant text:='participacion:chgljjfcdiflpedmjpdgmdhifofdioip';
 clave constant text:='datos:hz39:antonio:disponible:v1';
 recibo constant text:='recibo:datos:hz39:antonio:disponible:v1';
 ahora timestamptz(6):=clock_timestamp();
 actual bigint;
BEGIN
 SELECT count(*) INTO actual
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
 JOIN vec_bolsa_llamamientos.vinculo_candidato v
   ON v.acta_ref=c.acta_ref AND v.instantanea_ref=e.instantanea_ref
  AND v.version_instantanea=e.version_instantanea AND v.participacion_ref=e.participacion_ref
 WHERE c.bolsa_ref=bolsa AND c.version_bolsa=1 AND c.categoria_ref='categoria:rpt:administrativo'
   AND e.orden=1 AND e.participacion_ref=participacion AND v.candidato_ref=candidato;
 IF actual<>1 THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=participacion_acta_orden_1 actual=% esperado=1',actual USING ERRCODE='55000';
 END IF;
 -- Repetir la preparación no revierte lo que el recorrido haya registrado después.
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
           WHERE s.participacion_ref=participacion AND s.clave_idempotencia=clave) THEN
  IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                WHERE s.participacion_ref=participacion AND s.clave_idempotencia=clave
                  AND s.situacion='disponible' AND s.fecha_disponible IS NULL AND s.recibo_ref=recibo
                  AND s.actor='carga:hz39' AND s.motivo='Disponibilidad para la vacante de Administrativo') THEN
   RAISE EXCEPTION 'datos Antonio: PARO clave=carga_previa actual=distinta esperado=disponible_hz39_v1' USING ERRCODE='55000';
  END IF;
  RAISE NOTICE 'datos Antonio: carga ya existente; sin cambios';
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion
  (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(participacion,'disponible',ahora,NULL,NULL,'Disponibilidad para la vacante de Administrativo','carga:hz39',ahora,clave,recibo);
 PERFORM set_config('vec.datos_antonio_nuevos','true',true);
 RAISE NOTICE 'datos Antonio: situación disponible añadida; comprobación del turno pendiente';
END $datos$;
-- Otra sentencia obtiene un corte posterior a la situación recién añadida.
DO $turno$
DECLARE actual bigint;
BEGIN
 IF current_setting('vec.datos_antonio_nuevos',true) IS DISTINCT FROM 'true' THEN RETURN; END IF;
 -- La misma fuente de Bolsa que consume el llamamiento CT confirma el turno.
 SELECT count(*) INTO actual
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1('categoria:rpt:administrativo') o,
      LATERAL jsonb_array_elements(o.entradas) e
 WHERE o.bolsa_ref='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk' AND o.version_bolsa=1 AND e->>'candidato_ref'='can_Vxd6XlZE9ctNzw38AUSmgQbCcRyoFKhaojDaFsEKC1M'
   AND e->>'participacion_ref'='participacion:chgljjfcdiflpedmjpdgmdhifofdioip' AND e->>'situacion'='disponible'
   AND (e->>'orden_vigente')::bigint=1;
 IF actual<>1 THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=primer_turno_llamamiento actual=% esperado=1',actual USING ERRCODE='55000';
 END IF;
 RAISE NOTICE 'datos Antonio: primer turno de Administrativo v1 confirmado';
END $turno$;
COMMIT;
