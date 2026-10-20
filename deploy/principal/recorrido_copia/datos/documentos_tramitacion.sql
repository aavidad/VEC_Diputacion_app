\set ON_ERROR_STOP on
-- F1-131: semilla ficticia de copia, sin migración ni publicación de cargo.
-- AUT53 ya publicó el cargo; su CLI no asigna perfiles (README de la CLI, línea 4).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.set_config('vec.f131.docs.sid', __SID__, true);
SELECT pg_catalog.set_config('vec.f131.docs.original', __ORIGINAL__, true);
SELECT pg_catalog.set_config('vec.f131.docs.persona', __PERSONA__, true);
SELECT pg_catalog.set_config('vec.f131.docs.cuenta', __CUENTA__, true);
SELECT pg_catalog.set_config('vec.f131.docs.vinculo_original', __VINCULO_ORIGINAL__, true);
SELECT pg_catalog.set_config('vec.f131.docs.perfil', __PERFIL__, true);
SELECT pg_catalog.set_config('vec.f131.docs.vinculo', __VINCULO__, true);
SELECT pg_catalog.set_config('vec.f131.docs.asignacion', __ASIGNACION__, true);
SELECT pg_catalog.set_config('vec.f131.docs.organizacion', __ORGANIZACION__, true);
SELECT pg_catalog.set_config('vec.f131.docs.rol_sha', __ROL_SHA__, true);
SELECT pg_catalog.set_config('vec.f131.docs.control_sha', __CONTROL_SHA__, true);
SELECT pg_catalog.set_config('vec.f131.docs.retirada', __RETIRADA__, true);
DO $pre$
BEGIN
 IF session_user <> 'postgres' OR current_user <> 'postgres'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system())
       IS DISTINCT FROM current_setting('vec.f131.docs.sid')
 THEN RAISE EXCEPTION 'F1-131 Documentos: copia o ejecutor ajeno' USING ERRCODE='55000'; END IF;
END $pre$;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
 'vec:f131:documentos:'||current_setting('vec.f131.docs.sid')||':'||current_setting('vec.f131.docs.cuenta'),0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contexto$
DECLARE
 origen record;
 ahora timestamptz := pg_catalog.clock_timestamp();
 hasta timestamptz;
 perfil text := current_setting('vec.f131.docs.perfil');
 vinculo text := current_setting('vec.f131.docs.vinculo');
BEGIN
 SELECT v.*, p.vigente_hasta AS perfil_hasta, pc.vigente_hasta AS cuenta_hasta,
        pe.vigente_hasta AS persona_hasta INTO STRICT origen
 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
 JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.perfil_versiones p ON (p.perfil_ref,p.version)=(pa.perfil_ref,pa.version)
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones pc
   ON (pc.cuenta_ref,pc.version)=(ca.cuenta_ref,ca.version)
 JOIN vec_contexto_actor_v1.persona_actual pea ON pea.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.persona_versiones pe
   ON (pe.persona_ref,pe.version)=(pea.persona_ref,pea.version)
 WHERE v.vinculo_ref=current_setting('vec.f131.docs.vinculo_original')
   AND v.perfil_ref=current_setting('vec.f131.docs.original')
   AND v.persona_ref=current_setting('vec.f131.docs.persona')
   AND v.cuenta_ref=current_setting('vec.f131.docs.cuenta')
   AND p.persona_ref=v.persona_ref AND v.estado='activo' AND p.estado='activo'
   AND pc.estado='activo' AND pe.estado='activo'
   AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta
   AND ahora>=p.vigente_desde AND ahora<p.vigente_hasta
   AND ahora>=pc.vigente_desde AND ahora<pc.vigente_hasta
   AND ahora>=pe.vigente_desde AND ahora<pe.vigente_hasta;
 hasta := LEAST(origen.vigente_hasta,origen.perfil_hasta,origen.cuenta_hasta,
                origen.persona_hasta,ahora+interval '88 days',
                current_setting('vec.f131.docs.retirada')::timestamptz);
 IF hasta<ahora+interval '60 days'
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=perfil)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=perfil)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=vinculo)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=vinculo)
 THEN RAISE EXCEPTION 'F1-131 Documentos: vigencia o perfil previo incompatible' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
  (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
   procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(perfil,1,origen.persona_ref,origen.procedencia_ref,origen.procedencia_version,
   origen.procedencia_huella_sha256,origen.procedencia_autoridad,'activo',ahora-interval '1 minute',hasta);
 INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref,version) VALUES(perfil,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
  (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(vinculo,1,origen.cuenta_ref,perfil,origen.persona_ref,origen.procedencia_ref,
   origen.procedencia_version,origen.procedencia_huella_sha256,origen.procedencia_autoridad,
   'activo',ahora-interval '1 minute',hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref,version) VALUES(vinculo,1);
 -- El propietario de Autorización recibe solo la vigencia ya comprobada en esta TX.
 PERFORM pg_catalog.set_config('vec.f131.docs.ctx_hasta',hasta::text,true);
END $contexto$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $asignacion$
DECLARE
 ahora timestamptz := pg_catalog.date_trunc('second',pg_catalog.clock_timestamp());
 ctx_hasta timestamptz;
 hasta timestamptz;
 perfil text := current_setting('vec.f131.docs.perfil');
 asignacion text := current_setting('vec.f131.docs.asignacion');
 doc jsonb;
BEGIN
 ctx_hasta := NULLIF(current_setting('vec.f131.docs.ctx_hasta',true),'')::timestamptz;
 IF ctx_hasta IS NULL THEN
  RAISE EXCEPTION 'F1-131 Documentos: contexto no acreditado en la transacción' USING ERRCODE='55000';
 END IF;
 SELECT ra.vigente_hasta
 INTO STRICT hasta
 FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.rol_administrable_exacto_v1 ra
   ON ra.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca
   ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol cv
   ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
 WHERE r.version_rol_ref='rol:ct_cargo_tecnico_documentos_dev:v1'
   AND r.rol_id='ct_cargo_tecnico_documentos_dev' AND r.version=1
   AND r.huella_sha256=current_setting('vec.f131.docs.rol_sha')
   AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND cv.huella_sha256=current_setting('vec.f131.docs.control_sha')
   AND ra.huella_sha256=r.huella_sha256 AND ra.unidad_requerida IS FALSE
   AND ra.ambitos_fijos=jsonb_build_array(jsonb_build_object('clave','organizacion_ref',
     'valores',jsonb_build_array(current_setting('vec.f131.docs.organizacion'))))
   AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r2
                  WHERE r2.rol_id=r.rol_id AND r2.version>1);
 hasta:=LEAST(hasta,ctx_hasta);
 IF ctx_hasta<ahora+interval '60 days' OR ctx_hasta>ahora+interval '88 days'
    OR hasta<ahora+interval '60 days'
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil
              WHERE asignacion_id=asignacion OR perfil_activo_ref=perfil)
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=perfil)
 THEN RAISE EXCEPTION 'F1-131 Documentos: rol o asignación previa incompatible' USING ERRCODE='55000'; END IF;
 doc:=jsonb_build_object('asignacion_id',asignacion,'version',1,
   'perfil_activo_ref',perfil,'principal_id',current_setting('vec.f131.docs.persona'),
   'version_rol_ref','rol:ct_cargo_tecnico_documentos_dev:v1','estado','activa',
   'ambitos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref',
      'valores',jsonb_build_array(current_setting('vec.f131.docs.organizacion')))),
   'emitida_por','identidad:desarrollo:no-autoritativa',
   'emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vigente_hasta',to_char(hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'revocada_en','0001-01-01T00:00:00Z');
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,
   huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||asignacion||':v1',asignacion,1,perfil,
   current_setting('vec.f131.docs.persona'),'rol:ct_cargo_tecnico_documentos_dev:v1',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
     vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),ahora,doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
 VALUES(perfil,'asignacion:'||asignacion||':v1',ahora,
   'identidad:desarrollo:no-autoritativa','semilla:ensayo:f131:documentos-tramitacion');
END $asignacion$;
COMMIT;
