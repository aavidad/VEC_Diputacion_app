\set ON_ERROR_STOP on
-- Dos terminales sobre el mismo clon desechable, nunca principal conservada.
-- Variables: lado=lector/publicador, caso=pre_guard/lector_primero/publicador_pendiente,
-- fin=commit/rollback; lector recibe organizacion, ambito jsonb y hasta del fixture
-- gobernado. Los prompts sincronizan; no hay semillas ni INSERT de unidad falsa.
SELECT :'lado'='lector' AS lector, :'caso'='pre_guard' AS pre_guard,
 :'caso'='lector_primero' AS lector_primero, :'fin'='commit' AS confirmar \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='20s';
SET LOCAL statement_timeout='30s';
\if :lector
 SET LOCAL application_name='personal31_lector';
 SET LOCAL ROLE vec_autorizacion_propietario;
 SELECT pg_current_snapshot() IS NOT NULL AS instantanea_fijada;
 \if :lector_primero
  SELECT vec_personal.cotejar_unidad_bootstrap_admin_v1(:'organizacion',:'ambito'::jsonb,:'hasta'::timestamptz)->>'esquema'='vec.personal.unidad-bootstrap-admin.v1' AS fuente_original_acreditada;
  \echo LECTOR_RETIENE_GUARD_HASTA_COMMIT
  \prompt 'Iniciar publicador y confirmar espera en guard; después pulsar Intro: ' continuar
  RESET ROLE;
  DO $espera_publicador$
  DECLARE bloqueado boolean:=false;
  BEGIN
   FOR i IN 1..40 LOOP
    PERFORM pg_stat_clear_snapshot();
    SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='personal31_publicador' AND wait_event_type='Lock' AND pg_backend_pid()=ANY(pg_blocking_pids(pid))) INTO bloqueado;
    EXIT WHEN bloqueado;
    PERFORM pg_sleep(0.025);
   END LOOP;
   IF NOT bloqueado THEN RAISE EXCEPTION 'publicador no espera el guard del lector'; END IF;
  END $espera_publicador$;
  COMMIT;
  \echo LECTOR_LIBERADO
 \else
  \if :pre_guard
   \prompt 'Publicador debe COMMIT antes de continuar; después pulsar Intro: ' continuar
  \else
   \prompt 'Publicador debe retener INSERT sin terminar; continuar para esperar guard: ' continuar
  \endif
  \set ON_ERROR_STOP off
  SELECT vec_personal.cotejar_unidad_bootstrap_admin_v1(:'organizacion',:'ambito'::jsonb,:'hasta'::timestamptz)->>'esquema'='vec.personal.unidad-bootstrap-admin.v1' AS fuente_original_acreditada;
  \set resultado :SQLSTATE
  \set ON_ERROR_STOP on
  ROLLBACK;
  SELECT CASE WHEN :'fin'='commit' THEN :'resultado'='40001' ELSE :'resultado'='00000' END AS resultado_esperado \gset
  \if :resultado_esperado
   \echo PERSONAL31_CONCURRENCIA_OK
  \else
   \echo PERSONAL31_CONCURRENCIA_FALLO
   \quit 1
  \endif
 \endif
\else
 SET LOCAL application_name='personal31_publicador';
 SET LOCAL ROLE vec_personal_propietario;
 -- INSERT de cero filas ejecuta el trigger STATEMENT real sin fabricar fuente
 -- positiva ni cambiar la historia. Prueba coordinación, no publicación eficaz.
 \echo PUBLICADOR_ENTRA_INSERT
 INSERT INTO vec_personal.org_nodo_historia SELECT * FROM vec_personal.org_nodo_historia WHERE false;
 \if :pre_guard
  COMMIT;
  \echo PUBLICADOR_GENERACION_CONFIRMADA
 \else
  \if :lector_primero
   -- Este punto sólo se alcanza después del COMMIT del lector.
   ROLLBACK;
   \echo PUBLICADOR_REANUDADO_TRAS_LECTOR
  \else
   \prompt 'Lector debe estar esperando guard; después pulsar Intro: ' continuar
   RESET ROLE;
   DO $espera_lector$
   DECLARE bloqueado boolean:=false;
   BEGIN
    FOR i IN 1..40 LOOP
     PERFORM pg_stat_clear_snapshot();
     SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='personal31_lector' AND wait_event_type='Lock' AND pg_backend_pid()=ANY(pg_blocking_pids(pid))) INTO bloqueado;
     EXIT WHEN bloqueado;
     PERFORM pg_sleep(0.025);
    END LOOP;
    IF NOT bloqueado THEN RAISE EXCEPTION 'lector no espera el guard del publicador'; END IF;
   END $espera_lector$;
   \if :confirmar
    COMMIT;
    \echo PUBLICADOR_CONFIRMADO
   \else
    ROLLBACK;
    \echo PUBLICADOR_REVERTIDO
   \endif
  \endif
 \endif
\endif
