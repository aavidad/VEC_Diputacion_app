\if :{?solo_precondicion}
\else
\set ON_ERROR_STOP on
\endif
-- F1-63: dato ficticio de copia. Sin decisión ni recibo de Administración.
-- Variables psql: organizacion_canonica_ref, persona_*/perfil_* (cuatro
-- firmantes), plan_aut53_sha256, vigente_desde, vigente_hasta, emitida_en.
-- El modo prueba usa -v semilla_sin_tx=1 dentro de una transacción exterior.
\if :{?semilla_sin_tx}
\else
BEGIN ISOLATION LEVEL SERIALIZABLE;
\endif
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('semilla:copia:f1-63:firma',0));
SELECT pg_catalog.set_config('vec.f163.firma',pg_catalog.jsonb_build_object(
 'organizacion_canonica_ref',:'organizacion_canonica_ref',
 'organizacion_operativa_ref','organizacion:desarrollo:dipgra',
 'unidad_ref','unidad:desarrollo:rrhh',
 'plan_aut53_sha256',:'plan_aut53_sha256',
 'vigente_desde',:'vigente_desde','vigente_hasta',:'vigente_hasta',
 'emitida_en',:'emitida_en',
 'firmantes',pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('papel','tecnico','persona',:'persona_tecnico','perfil',:'perfil_tecnico'),
  pg_catalog.jsonb_build_object('papel','jefatura','persona',:'persona_jefatura','perfil',:'perfil_jefatura'),
  pg_catalog.jsonb_build_object('papel','diputacion','persona',:'persona_diputacion','perfil',:'perfil_diputacion'),
  pg_catalog.jsonb_build_object('papel','secretaria','persona',:'persona_secretaria','perfil',:'perfil_secretaria'))
 )::text,true) IS NOT NULL AS parametros_cargados;

DO $pre$
DECLARE j jsonb:=pg_catalog.current_setting('vec.f163.firma')::jsonb;
 f jsonb; r record; cargo_publicado record; p record; inicio timestamptz; fin timestamptz;
 emision timestamptz; ahora timestamptz:=pg_catalog.clock_timestamp();
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolsuper)
    OR current_database()<>'postgres'
 THEN RAISE EXCEPTION 'F1-63: solo DBA de la copia PostgreSQL 18' USING ERRCODE='42501'; END IF;
 IF j->>'organizacion_canonica_ref' IS NULL
    OR j->>'organizacion_canonica_ref' !~ '^org_[A-Za-z0-9_-]{22,128}$'
    OR j->>'plan_aut53_sha256' IS NULL
    OR j->>'plan_aut53_sha256' !~ '^[0-9a-f]{64}$'
    OR pg_catalog.jsonb_typeof(j->'firmantes') IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_array_length(j->'firmantes') IS DISTINCT FROM 4
    OR (SELECT count(DISTINCT x->>'persona') FROM pg_catalog.jsonb_array_elements(j->'firmantes') x)<>4
    OR (SELECT count(DISTINCT x->>'perfil') FROM pg_catalog.jsonb_array_elements(j->'firmantes') x)<>4
    OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(j->'firmantes') x
       WHERE x->>'persona' IS NULL OR x->>'perfil' IS NULL
          OR x->>'persona' !~ '^per_[A-Za-z0-9_-]{22,128}$'
          OR x->>'perfil' !~ '^prf_[A-Za-z0-9_-]{22,128}$')
 THEN RAISE EXCEPTION 'F1-63: referencias de copia inválidas' USING ERRCODE='22023'; END IF;
 IF j->>'vigente_desde' IS NULL OR j->>'vigente_hasta' IS NULL OR j->>'emitida_en' IS NULL
    OR j->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
    OR j->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
    OR j->>'emitida_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
 THEN RAISE EXCEPTION 'F1-63: fechas UTC canónicas requeridas' USING ERRCODE='22023'; END IF;
 inicio:=(j->>'vigente_desde')::timestamptz;
 fin:=(j->>'vigente_hasta')::timestamptz;
 emision:=(j->>'emitida_en')::timestamptz;
 IF inicio IS NULL OR fin IS NULL OR emision IS NULL
    OR NOT pg_catalog.isfinite(inicio) OR NOT pg_catalog.isfinite(fin) OR NOT pg_catalog.isfinite(emision)
    OR inicio>ahora OR emision>ahora OR emision>inicio
    OR fin<ahora+interval '60 days' OR fin>ahora+interval '90 days'
 THEN RAISE EXCEPTION 'F1-63: vigencia fuera de 60 a 90 días' USING ERRCODE='22023'; END IF;

 SELECT * INTO r FROM vec_autorizacion.registro_cargos_firma_admin_v1
  WHERE plan_sha256=j->>'plan_aut53_sha256';
 IF NOT FOUND THEN
  RAISE EXCEPTION 'F1-63: publicación AUT53 propia de 16 cargos ausente' USING ERRCODE='55000';
 END IF;
 IF r.recibo->>'plan_sha256' IS DISTINCT FROM j->>'plan_aut53_sha256'
    OR r.recibo->>'auditoria_ref' IS NULL
    OR pg_catalog.jsonb_array_length(r.recibo->'cargos')<>16
    OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.operacion_cargos_firma_admin_v1 o
       WHERE o.auditoria_ref=r.auditoria_ref AND o.tipo='confirmacion'
         AND o.resultado='permitido' AND o.operacion_ref=r.operacion_ref)
 THEN RAISE EXCEPTION 'F1-63: publicación AUT53 propia de 16 cargos ausente' USING ERRCODE='55000'; END IF;

 -- Ninguna asignación canónica vigente de estas personas se interpreta como
 -- autorización para el ámbito legado. Detener también una segunda pasada.
 IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(j->'firmantes') firmante_dato
   JOIN vec_autorizacion.asignacion_perfil a ON a.principal_id=firmante_dato->>'persona'
   JOIN vec_autorizacion.asignacion_perfil_actual ac ON ac.asignacion_ref=a.asignacion_ref
   CROSS JOIN LATERAL pg_catalog.jsonb_array_elements(a.documento->'ambitos') b
   WHERE a.documento->>'estado'='activa' AND b->>'clave'='organizacion_ref'
     AND b->'valores' @> pg_catalog.jsonb_build_array(j->>'organizacion_canonica_ref'))
 THEN RAISE EXCEPTION 'F1-63: firmante con asignación canónica; semilla denegada' USING ERRCODE='42501'; END IF;

 FOR f IN SELECT value FROM pg_catalog.jsonb_array_elements(j->'firmantes') LOOP
  SELECT a.*,v.persona_ref,v.estado,v.vigente_desde,v.vigente_hasta INTO p
    FROM vec_contexto_actor_v1.perfil_actual a
    JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
   WHERE a.perfil_ref=f->>'perfil';
  IF NOT FOUND THEN RAISE EXCEPTION 'F1-63: firmante sin perfil vigente' USING ERRCODE='55000'; END IF;
  IF p.persona_ref IS DISTINCT FROM f->>'persona'
     OR p.estado IS DISTINCT FROM 'activo' OR p.vigente_desde>ahora OR p.vigente_hasta<fin
  THEN RAISE EXCEPTION 'F1-63: firmante sin perfil vigente' USING ERRCODE='55000'; END IF;
  SELECT vr.*,ar.vigente_desde AS rol_desde,ar.vigente_hasta AS rol_hasta,
         ar.ambitos_fijos AS rol_ambitos,c.estado AS control_estado,
         c.huella_sha256 AS control_sha,ca.revision AS control_revision INTO cargo_publicado
    FROM vec_autorizacion.version_rol vr
    JOIN vec_autorizacion.rol_administrable_exacto_v1 ar
      ON ar.version_rol_ref=vr.version_rol_ref AND ar.huella_sha256=vr.huella_sha256
    JOIN vec_autorizacion.control_vigencia_version_rol_actual ca
      ON ca.version_rol_ref=vr.version_rol_ref
    JOIN vec_autorizacion.control_vigencia_version_rol c
      ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
   WHERE vr.version_rol_ref='rol:ct_cargo_'||(f->>'papel')||'_firma_dev:v1';
  IF NOT FOUND THEN RAISE EXCEPTION 'F1-63: cargo de firma no publicado' USING ERRCODE='55000'; END IF;
  IF cargo_publicado.control_estado<>'habilitada' OR cargo_publicado.rol_desde>inicio OR cargo_publicado.rol_hasta<fin
     OR cargo_publicado.rol_ambitos IS DISTINCT FROM pg_catalog.jsonb_build_array(
       pg_catalog.jsonb_build_object('clave','organizacion_ref',
         'valores',pg_catalog.jsonb_build_array(j->>'organizacion_operativa_ref')))
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(r.recibo->'cargos') c
       WHERE c->>'version_rol_ref'=cargo_publicado.version_rol_ref
         AND c->>'version_rol_sha256'=cargo_publicado.huella_sha256
         AND c->>'control_sha256'=cargo_publicado.control_sha
         AND (c->>'control_revision')::numeric=cargo_publicado.control_revision)
  THEN RAISE EXCEPTION 'F1-63: cargo de firma no publicado o sin vigencia' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
\if :{?solo_precondicion}
\set f163_pre_estado :SQLSTATE
\else

-- CA45 publica la correspondencia desde la organización maestra existente.
-- Si ya está publicada, se comprueba y no se genera otra versión ni recibo.
DO $correspondencia$
DECLARE j jsonb:=pg_catalog.current_setting('vec.f163.firma')::jsonb;
 actual record; org record; d jsonb; fin timestamptz:=(j->>'vigente_hasta')::timestamptz;
BEGIN
 SELECT h.* INTO STRICT org FROM vec_contexto_actor_v1.organizacion_actual a
  JOIN vec_contexto_actor_v1.organizacion_versiones h USING(organizacion_ref,version)
  WHERE a.organizacion_ref=j->>'organizacion_canonica_ref';
 IF org.estado<>'activo' OR org.vigente_desde>pg_catalog.clock_timestamp()
    OR org.vigente_hasta<fin
 THEN RAISE EXCEPTION 'F1-63: organización canónica sin vigencia' USING ERRCODE='55000'; END IF;
 SELECT h.* INTO actual FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
 JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h
   USING(organizacion_operativa_ref,version)
 WHERE a.organizacion_operativa_ref=j->>'organizacion_operativa_ref';
 IF FOUND THEN
  d:=pg_catalog.convert_from(actual.documento_canonico,'UTF8')::jsonb;
  IF actual.organizacion_canonica_ref IS DISTINCT FROM j->>'organizacion_canonica_ref'
    OR actual.organizacion_version IS DISTINCT FROM org.version
    OR d->>'procedencia_ref' IS DISTINCT FROM org.procedencia_ref
    OR (d->>'procedencia_version')::numeric IS DISTINCT FROM org.procedencia_version
    OR d->>'procedencia_huella_sha256' IS DISTINCT FROM org.procedencia_huella_sha256
    OR d->>'procedencia_autoridad' IS DISTINCT FROM org.procedencia_autoridad
    OR d->>'estado'<>'vigente'
    OR (d->>'vigente_desde')::timestamptz>pg_catalog.clock_timestamp()
    OR (d->>'vigente_hasta')::timestamptz<fin
  THEN RAISE EXCEPTION 'F1-63: correspondencia CA45 distinta o vencida' USING ERRCODE='55000'; END IF;
 ELSE
  PERFORM vec_contexto_actor_v1.publicar_correspondencia_organizacion_v1(
   pg_catalog.jsonb_build_object('esquema','vec.organizacion.correspondencia.v1',
    'correspondencia_ref','correspondencia-org:'||pg_catalog.gen_random_uuid()::text,
    'version',1,'organizacion_operativa_ref',j->>'organizacion_operativa_ref',
    'organizacion_canonica_ref',org.organizacion_ref,'organizacion_version',org.version,
    'procedencia_ref',org.procedencia_ref,'procedencia_version',org.procedencia_version,
    'procedencia_huella_sha256',org.procedencia_huella_sha256,
    'procedencia_autoridad',org.procedencia_autoridad,'estado','vigente',
    'vigente_desde',pg_catalog.to_char(org.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    'vigente_hasta',pg_catalog.to_char(org.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'))::text,0,'');
 END IF;
END $correspondencia$;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $asignaciones$
DECLARE j jsonb:=pg_catalog.current_setting('vec.f163.firma')::jsonb;
 f jsonb; doc jsonb; anterior record; id text; ref text; rol text; sha text;
BEGIN
 FOR f IN SELECT value FROM pg_catalog.jsonb_array_elements(j->'firmantes') ORDER BY value->>'papel' LOOP
  rol:='rol:ct_cargo_'||(f->>'papel')||'_firma_dev:v1';
  id:='asg_'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
       pg_catalog.convert_to('semilla:f1-63:'||(f->>'perfil'),'UTF8')),'hex'),1,32);
  ref:='asignacion:'||id||':v1';
  doc:=pg_catalog.jsonb_build_object('asignacion_id',id,'version',1,
   'perfil_activo_ref',f->>'perfil','principal_id',f->>'persona',
   'version_rol_ref',rol,'estado','activa','ambitos',pg_catalog.jsonb_build_array(
     pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(j->>'organizacion_operativa_ref')),
     pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(j->>'unidad_ref'))),
   'emitida_por','semilla:copia:f1-63','emitida_en',j->>'emitida_en',
   'vigente_desde',j->>'vigente_desde','vigente_hasta',j->>'vigente_hasta',
   'revocada_en','0001-01-01T00:00:00Z');
  sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
     vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex');
  SELECT a.* INTO anterior FROM vec_autorizacion.asignacion_perfil_actual x
    JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
   WHERE x.perfil_activo_ref=f->>'perfil';
  IF FOUND THEN
   -- El kit anterior puede haber creado esta asignación. Se conserva intacta
   -- si ya acredita exactamente persona, cargo, organización y unidad.
   IF anterior.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
       pg_catalog.convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(anterior.documento),'UTF8')),'hex')
      OR anterior.principal_id IS DISTINCT FROM f->>'persona'
      OR anterior.version_rol_ref IS DISTINCT FROM rol
      OR anterior.documento->>'estado'<>'activa'
      OR anterior.documento->'ambitos' IS DISTINCT FROM doc->'ambitos'
      OR (anterior.documento->>'vigente_desde')::timestamptz>(j->>'vigente_desde')::timestamptz
      OR (anterior.documento->>'vigente_hasta')::timestamptz<(j->>'vigente_hasta')::timestamptz
   THEN RAISE EXCEPTION 'F1-63: asignación previa distinta' USING ERRCODE='55000'; END IF;
  ELSE
   IF EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil
     WHERE perfil_activo_ref=f->>'perfil' OR asignacion_id=id)
   THEN RAISE EXCEPTION 'F1-63: historia previa sin puntero; no revivir' USING ERRCODE='55000'; END IF;
   INSERT INTO vec_autorizacion.asignacion_perfil
    (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
     version_rol_ref,huella_sha256,emitida_en,documento)
   VALUES(ref,id,1,f->>'perfil',f->>'persona',rol,sha,
     (j->>'emitida_en')::timestamptz,doc);
   INSERT INTO vec_autorizacion.asignacion_perfil_actual
    (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
   VALUES(f->>'perfil',ref,pg_catalog.clock_timestamp(),
     'semilla:copia:f1-63','semilla:copia:f1-63:asignacion');
  END IF;
 END LOOP;
END $asignaciones$;
RESET ROLE;

-- Salida sin referencias de personas, certificados ni claves.
SELECT pg_catalog.jsonb_build_object(
 'tipo','dato_de_copia_f1_63',
 'correspondencia_sha256',(SELECT h.huella_sha256 FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
    JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h USING(organizacion_operativa_ref,version)
    WHERE a.organizacion_operativa_ref='organizacion:desarrollo:dipgra'),
 'asignaciones_sha256',(SELECT pg_catalog.jsonb_agg(a.huella_sha256 ORDER BY a.version_rol_ref)
    FROM vec_autorizacion.asignacion_perfil_actual x
    JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
    WHERE a.perfil_activo_ref IN (SELECT f->>'perfil' FROM pg_catalog.jsonb_array_elements(
       pg_catalog.current_setting('vec.f163.firma')::jsonb->'firmantes') f))) AS resumen;
\endif
\if :{?semilla_sin_tx}
\else
COMMIT;
\endif
