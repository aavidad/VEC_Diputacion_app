\set ON_ERROR_STOP on
-- Datos ficticios del recorrido: vincula a Antonio con SU fila del acta vigente
-- (orden 5, «Antonio Reyes Álvarez») y deja no disponibles a las cuatro
-- personas anteriores, para que el primer turno sea el suyo sin tocar el acta.
-- No constituye otra bolsa, no modifica la importación ni reescribe historia.
-- Corrige la v1, que lo vinculaba a la fila 1 del acta (otra persona) y hacía
-- que la lista de la bolsa mostrara un nombre que no era el suyo.
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
-- Misma proyección de candidato que preparar_candidato_desarrollo.sh:
-- conserva persona, cuenta, perfil y certificado; añade una versión del vínculo.
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $vinculo$
DECLARE
 persona constant text:='per_8014599251fc9a7aff11113c502e3317';
 -- Candidato de la fila del acta que lleva el nombre de Antonio.
 candidato constant text:='can_dJo5qQLf4agrD4c3jkIlHUkga2uQ-BusRCLCYPzKLyY';
 -- Admitidos antes de esta versión: el del acta sustituida y el de la v1 (fila 1).
 anteriores constant text[]:=ARRAY['can_Xx9GdqcSDoRdUVUPyGMDJlcSBvLDUXc0Or-G86S2pxc',
                                   'can_Vxd6XlZE9ctNzw38AUSmgQbCcRyoFKhaojDaFsEKC1M'];
 origen constant text:='prc_2e4e98543d7939709072c4afc75c51ce';
 huella text:=encode(sha256(convert_to(persona||'|'||candidato||'|hz39:v2','UTF8')),'hex');
 previo vec_contexto_actor_v1.vinculo_referencia_versiones%ROWTYPE;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
           JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
           WHERE v.tipo='candidato' AND v.referencia=candidato AND v.estado='activo'
             AND v.persona_ref<>persona) THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=candidato_persona actual=otra_persona esperado=%',persona USING ERRCODE='55000';
 END IF;
 SELECT v.* INTO STRICT previo
 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
 JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
 WHERE v.persona_ref=persona AND v.tipo='candidato' AND v.estado='activo'
   AND v.vigente_desde<=statement_timestamp() AND statement_timestamp()<v.vigente_hasta
 FOR UPDATE OF a;
 IF previo.referencia=candidato THEN RETURN; END IF;
 IF NOT previo.referencia=ANY(anteriores) THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=candidato_anterior actual=% esperado=%',previo.referencia,anteriores USING ERRCODE='55000';
 END IF;
 -- Si el vínculo volvió atrás y se repite, la procedencia ya existe: se acepta
 -- solo si es exactamente la misma.
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias p WHERE p.procedencia_ref=origen) THEN
  IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias p
                WHERE p.procedencia_ref=origen AND p.procedencia_version=1
                  AND p.procedencia_huella_sha256=huella AND p.procedencia_autoridad='autoridad_maestra_acreditada') THEN
   RAISE EXCEPTION 'datos Antonio: PARO clave=procedencia actual=distinta esperado=%',origen USING ERRCODE='55000';
  END IF;
 ELSE
  INSERT INTO vec_contexto_actor_v1.procedencias
  VALUES(origen,1,huella,'autoridad_maestra_acreditada');
 END IF;
 INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
  (vinculo_ref,version,persona_ref,tipo,referencia,procedencia_ref,procedencia_version,
   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(previo.vinculo_ref,previo.version+1,persona,'candidato',candidato,origen,1,
        huella,'autoridad_maestra_acreditada','activo',statement_timestamp(),previo.vigente_hasta);
 UPDATE vec_contexto_actor_v1.vinculo_referencia_actual
 SET version=previo.version+1 WHERE vinculo_ref=previo.vinculo_ref AND version=previo.version;
 RAISE NOTICE 'datos Antonio: vínculo añadido a su participación de la bolsa vigente; versión anterior conservada';
END $vinculo$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $datos$
DECLARE
 bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 candidato constant text:='can_dJo5qQLf4agrD4c3jkIlHUkga2uQ-BusRCLCYPzKLyY';
 participacion constant text:='participacion:iedpeeeebddemcmgdefieokgpkcoepap';
 -- Las cuatro primeras filas del acta, por orden; ninguna es Antonio.
 previas constant text[]:=ARRAY['participacion:chgljjfcdiflpedmjpdgmdhifofdioip',
  'participacion:gnpboboekjmlgflkaknffefgcacnnjbj','participacion:bghdjojcccmjllmjbblgneoahhjddmon',
  'participacion:gfcpijnniglaboekdkkmmigpamoodjoi'];
 clave constant text:='datos:hz39:anteriores:no-disponible:v2';
 motivo_carga constant text:='Ha comunicado que no está disponible';
 ahora timestamptz(6):=clock_timestamp();
 actual bigint;
 ultima text;
 nuevos integer:=0;
 i integer;
BEGIN
 SELECT count(*) INTO actual
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
 JOIN vec_bolsa_llamamientos.vinculo_candidato v
   ON v.acta_ref=c.acta_ref AND v.instantanea_ref=e.instantanea_ref
  AND v.version_instantanea=e.version_instantanea AND v.participacion_ref=e.participacion_ref
 WHERE c.bolsa_ref=bolsa AND c.version_bolsa=1 AND c.categoria_ref='categoria:rpt:administrativo'
   AND e.orden=5 AND e.participacion_ref=participacion AND v.candidato_ref=candidato;
 IF actual<>1 THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=participacion_acta_orden_5 actual=% esperado=1',actual USING ERRCODE='55000';
 END IF;
 FOR i IN 1..4 LOOP
  SELECT count(*) INTO actual
  FROM vec_bolsa_llamamientos.constitucion c
  JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
  WHERE c.bolsa_ref=bolsa AND c.version_bolsa=1 AND e.orden=i AND e.participacion_ref=previas[i];
  IF actual<>1 THEN
   RAISE EXCEPTION 'datos Antonio: PARO clave=participacion_acta_orden_% actual=% esperado=1',i,actual USING ERRCODE='55000';
  END IF;
  -- Mismo bloqueo por participación que las operaciones de Bolsa.
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:situacion:'||previas[i],0));
  -- Repetir la preparación no revierte lo que el recorrido haya registrado después.
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
            WHERE s.participacion_ref=previas[i] AND s.clave_idempotencia=clave) THEN
   IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                 WHERE s.participacion_ref=previas[i] AND s.clave_idempotencia=clave
                   AND s.situacion='no_disponible' AND s.fecha_disponible IS NULL
                   AND s.recibo_ref='recibo:datos:hz39:no-disponible:v2:'||i
                   AND s.actor='carga:hz39' AND s.motivo=motivo_carga) THEN
    RAISE EXCEPTION 'datos Antonio: PARO clave=carga_previa_orden_% actual=distinta esperado=no_disponible_hz39_v2',i USING ERRCODE='55000';
   END IF;
   CONTINUE;
  END IF;
  -- Solo se marca a quien sigue disponible: si el recorrido ya la llamó, la
  -- contrató o cambió su situación, se para en vez de taparlo.
  SELECT s.situacion INTO ultima FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref=previas[i] ORDER BY s.desde DESC LIMIT 1 FOR UPDATE;
  IF FOUND AND ultima IS DISTINCT FROM 'disponible' THEN
   RAISE EXCEPTION 'datos Antonio: PARO clave=situacion_orden_% actual=% esperado=disponible',i,ultima USING ERRCODE='55000';
  END IF;
  INSERT INTO vec_bolsa_llamamientos.situacion_participacion
   (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
  VALUES(previas[i],'no_disponible',ahora,NULL,NULL,motivo_carga,'carga:hz39',ahora,clave,
         'recibo:datos:hz39:no-disponible:v2:'||i);
  nuevos:=nuevos+1;
 END LOOP;
 IF nuevos=0 THEN
  RAISE NOTICE 'datos Antonio: carga ya existente; sin cambios';
  RETURN;
 END IF;
 PERFORM set_config('vec.datos_antonio_nuevos','true',true);
 RAISE NOTICE 'datos Antonio: % posiciones anteriores no disponibles; comprobación del turno pendiente',nuevos;
END $datos$;
-- Otra sentencia obtiene un corte posterior a las situaciones recién añadidas.
DO $turno$
DECLARE actual bigint;
BEGIN
 IF current_setting('vec.datos_antonio_nuevos',true) IS DISTINCT FROM 'true' THEN RETURN; END IF;
 -- La misma fuente de Bolsa que consume el llamamiento CT confirma el turno.
 SELECT count(*) INTO actual
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1('categoria:rpt:administrativo') o,
      LATERAL jsonb_array_elements(o.entradas) e
 WHERE o.bolsa_ref='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk' AND o.version_bolsa=1
   AND e->>'candidato_ref'='can_dJo5qQLf4agrD4c3jkIlHUkga2uQ-BusRCLCYPzKLyY'
   AND e->>'participacion_ref'='participacion:iedpeeeebddemcmgdefieokgpkcoepap' AND e->>'situacion'='disponible'
   AND (e->>'orden_acta')::bigint=5 AND (e->>'orden_vigente')::bigint=1;
 IF actual<>1 THEN
  RAISE EXCEPTION 'datos Antonio: PARO clave=primer_turno_llamamiento actual=% esperado=1',actual USING ERRCODE='55000';
 END IF;
 RAISE NOTICE 'datos Antonio: primer turno de Administrativo v1 confirmado';
END $turno$;
COMMIT;
