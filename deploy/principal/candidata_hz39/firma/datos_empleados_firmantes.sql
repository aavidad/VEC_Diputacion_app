\set ON_ERROR_STOP on
-- Datos ficticios HZ39: cuatro personas nominales ya existentes obtienen una
-- proyección de empleado de Personal y una relación de servicio en RRHH.
-- Operación técnica de datos, sin decisión V3, recibo ni firma oficial.
-- Los valores privados entran mediante un fichero psql 0600 fuera de Git.
\if :{?persona_tecnico}
\else
\quit 2
\endif
\if :{?persona_jefatura}
\else
\quit 2
\endif
\if :{?persona_diputacion}
\else
\quit 2
\endif
\if :{?persona_secretaria}
\else
\quit 2
\endif

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:datos:recorrido:hz39:firmantes',0));

-- La sustitución de psql ocurre aquí, fuera de un literal dollar-quoted.
-- El SELECT solo imprime un booleano; no muestra personas ni conexiones.
SELECT pg_catalog.set_config('vec.hz39.firmantes',pg_catalog.jsonb_build_object(
 'organismo',:'organismo_ref','unidad',:'unidad_ref',
 'regimen',:'regimen_ref','modalidad',:'modalidad_ref',
 'desde',:'vigente_desde','hasta',:'vigente_hasta',
 'firmantes',pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('persona',:'persona_tecnico','perfil',:'perfil_tecnico'),
  pg_catalog.jsonb_build_object('persona',:'persona_jefatura','perfil',:'perfil_jefatura'),
  pg_catalog.jsonb_build_object('persona',:'persona_diputacion','perfil',:'perfil_diputacion'),
  pg_catalog.jsonb_build_object('persona',:'persona_secretaria','perfil',:'perfil_secretaria'))
 )::text,true) IS NOT NULL AS parametros_cargados;

-- Todas las personas, perfiles, unidad y vigencias se cotejan antes de tocar
-- Personal. La asignación de perfil no se toma del fichero como autoridad.
DO $pre$
DECLARE j jsonb:=pg_catalog.current_setting('vec.hz39.firmantes')::jsonb;
 f jsonb; inicio timestamptz; fin timestamptz; org text; unidad text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR pg_catalog.jsonb_typeof(j->'firmantes') IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_array_length(j->'firmantes')<>4
    OR (SELECT count(DISTINCT x->>'persona') FROM pg_catalog.jsonb_array_elements(j->'firmantes') x)<>4
    OR (SELECT count(DISTINCT x->>'perfil') FROM pg_catalog.jsonb_array_elements(j->'firmantes') x)<>4
 THEN RAISE EXCEPTION 'HZ39: preimagen de firmantes incompatible' USING ERRCODE='55000'; END IF;
 BEGIN inicio:=(j->>'desde')::timestamptz; fin:=(j->>'hasta')::timestamptz;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'HZ39: vigencia inválida' USING ERRCODE='22023'; END;
 org:=j->>'organismo'; unidad:=j->>'unidad';
 IF org IS NULL OR unidad IS NULL OR j->>'regimen' IS NULL OR j->>'modalidad' IS NULL
    OR org !~ '^[a-z][a-z0-9_:-]{2,127}$' OR unidad !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR j->>'regimen' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR j->>'modalidad' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR inicio IS NULL OR fin IS NULL OR inicio<>date_trunc('day',inicio)
    OR fin<>date_trunc('day',fin)
    OR inicio>pg_catalog.clock_timestamp() OR fin-inicio<interval '400 days'
    OR fin<pg_catalog.clock_timestamp()+interval '60 days'
    OR NOT pg_catalog.isfinite(inicio) OR NOT pg_catalog.isfinite(fin)
    OR (SELECT count(*) FROM vec_personal.org_nodo_historia n
       WHERE n.organismo_ref=org AND n.unidad_ref=unidad AND NOT n.retirado
         AND n.vigente_desde<=inicio::date
         AND (n.vigente_hasta IS NULL OR n.vigente_hasta>=fin::date)
         AND NOT EXISTS (SELECT 1 FROM vec_personal.org_nodo_historia nueva
           WHERE nueva.nodo_ref=n.nodo_ref AND nueva.revision>n.revision))<>1
 THEN RAISE EXCEPTION 'HZ39: organización, unidad o vigencia no acreditadas' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT value FROM pg_catalog.jsonb_array_elements(j->'firmantes') ORDER BY value->>'persona' LOOP
  IF f->>'persona' IS NULL OR f->>'perfil' IS NULL
     OR f->>'persona' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR f->>'perfil' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
     OR NOT EXISTS (SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a
       JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
       WHERE a.perfil_ref=f->>'perfil' AND v.persona_ref=f->>'persona' AND v.estado='activo'
         AND v.vigente_desde<=pg_catalog.clock_timestamp() AND v.vigente_hasta>=fin)
     OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
       JOIN vec_autorizacion.asignacion_perfil v ON v.asignacion_ref=a.asignacion_ref
       WHERE a.perfil_activo_ref=f->>'perfil' AND v.principal_id=f->>'persona'
         AND v.documento->>'estado'='activa'
         AND (v.documento->>'vigente_desde')::timestamptz<=pg_catalog.clock_timestamp()
         AND (v.documento->>'vigente_hasta')::timestamptz>=fin)
  THEN RAISE EXCEPTION 'HZ39: persona o perfil nominal no acreditado' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

SET LOCAL ROLE vec_personal_propietario;
DO $datos$
DECLARE j jsonb:=pg_catalog.current_setting('vec.hz39.firmantes')::jsonb;
 f jsonb; persona text; empleado text; proyeccion text; relacion text; procedencia text;
 huella text; sufijo text; inicio timestamptz; fin timestamptz; previo record;
BEGIN
 inicio:=(j->>'desde')::timestamptz; fin:=(j->>'hasta')::timestamptz;
 FOR f IN SELECT value FROM pg_catalog.jsonb_array_elements(j->'firmantes') ORDER BY value->>'persona' LOOP
  persona:=f->>'persona';
  -- Identificadores opacos estables: distintos por persona y entre P/H.
  sufijo:=pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   'vec.hz39.empleado.firmante.v1|'||persona,'UTF8')),'hex'),1,32);
  empleado:='emp_'||sufijo; proyeccion:='pep_'||sufijo;
  relacion:='rel_'||sufijo; procedencia:='prc_'||sufijo;
  huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   'datos:recorrido:hz39|'||persona||'|'||empleado||'|'||(j->>'organismo')||'|'||(j->>'unidad')||'|'||
   (j->>'regimen')||'|'||(j->>'modalidad')||'|'||inicio::text||'|'||fin::text,'UTF8')),'hex');
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
   'vec_personal:proyeccion-empleado:persona:'||persona,0));

  SELECT * INTO previo FROM vec_personal.proyeccion_empleado_persona_historia
   WHERE proyeccion_ref=proyeccion AND version=1;
  IF FOUND THEN
   IF previo.persona_ref IS DISTINCT FROM persona OR previo.empleado_ref IS DISTINCT FROM empleado
      OR previo.estado IS DISTINCT FROM 'activa' OR previo.motivo IS NOT NULL
      OR previo.vigente_desde IS DISTINCT FROM inicio OR previo.vigente_hasta IS DISTINCT FROM fin
      OR previo.procedencia_acto_ref IS DISTINCT FROM 'datos:recorrido:hz39'
      OR previo.procedencia_ref IS DISTINCT FROM procedencia OR previo.procedencia_version IS DISTINCT FROM 1
      OR previo.procedencia_huella_sha256 IS DISTINCT FROM huella
   THEN RAISE EXCEPTION 'HZ39: proyección existente distinta' USING ERRCODE='55000'; END IF;
  ELSE
   IF EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia
       WHERE persona_ref=persona OR empleado_ref=empleado OR proyeccion_ref=proyeccion) THEN
    RAISE EXCEPTION 'HZ39: otra proyección ya ocupa persona o empleado' USING ERRCODE='55000'; END IF;
   INSERT INTO vec_personal.proyeccion_empleado_persona_historia
    (proyeccion_ref,version,persona_ref,empleado_ref,estado,motivo,vigente_desde,vigente_hasta,
     procedencia_acto_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,registrada_en)
   VALUES(proyeccion,1,persona,empleado,'activa',NULL,inicio,fin,
    'datos:recorrido:hz39',procedencia,1,huella,pg_catalog.clock_timestamp());
  END IF;

  SELECT * INTO previo FROM vec_personal.relacion_servicio_historia
   WHERE relacion_ref=relacion AND revision=1;
  IF FOUND THEN
   IF previo.persona_ref IS DISTINCT FROM persona OR previo.empleado_ref IS DISTINCT FROM empleado
      OR previo.organismo_ref IS DISTINCT FROM j->>'organismo'
      OR previo.unidad_ref IS DISTINCT FROM j->>'unidad'
      OR previo.regimen_ref IS DISTINCT FROM j->>'regimen'
      OR previo.modalidad_ref IS DISTINCT FROM j->>'modalidad'
      OR previo.estado IS DISTINCT FROM 'vigente' OR previo.vigente_desde IS DISTINCT FROM inicio::date
      OR previo.vigente_hasta IS DISTINCT FROM fin::date
      OR previo.acto_ref IS DISTINCT FROM 'datos:recorrido:hz39'
      OR previo.fuente_ref IS DISTINCT FROM 'datos:recorrido:hz39'
      OR previo.fuente_version IS DISTINCT FROM 1 OR previo.fuente_huella_sha256 IS DISTINCT FROM huella
      OR previo.firma_oficial IS DISTINCT FROM false OR previo.eficacia_administrativa IS DISTINCT FROM false
      OR previo.decision_ref IS DISTINCT FROM ('datos:recorrido:hz39:sin_decision_v3:'||sufijo)
      OR previo.auditoria_ref IS DISTINCT FROM ('datos:recorrido:hz39:sin_auditoria_v3:'||sufijo)
      OR previo.catalogo_snapshot IS DISTINCT FROM '{}'::jsonb
   THEN RAISE EXCEPTION 'HZ39: relación existente distinta' USING ERRCODE='55000'; END IF;
  ELSE
   IF EXISTS (SELECT 1 FROM vec_personal.relacion_servicio_historia
       WHERE persona_ref=persona OR empleado_ref=empleado OR relacion_ref=relacion) THEN
    RAISE EXCEPTION 'HZ39: otra relación ya ocupa persona o empleado' USING ERRCODE='55000'; END IF;
   INSERT INTO vec_personal.relacion_servicio_historia
    (relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,modalidad_ref,
     estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,
     fuente_huella_sha256,firma_oficial,eficacia_administrativa,decision_ref,auditoria_ref,catalogo_snapshot)
   VALUES(relacion,1,persona,empleado,j->>'organismo',j->>'unidad',j->>'regimen',j->>'modalidad',
    'vigente',inicio::date,fin::date,pg_catalog.clock_timestamp(),
    'datos:recorrido:hz39','datos:recorrido:hz39',1,huella,false,false,
    'datos:recorrido:hz39:sin_decision_v3:'||sufijo,
    'datos:recorrido:hz39:sin_auditoria_v3:'||sufijo,'{}'::jsonb);
  END IF;

  IF (SELECT count(*) FROM vec_personal.proyeccion_empleado_persona_historia WHERE persona_ref=persona)<>1
     OR (SELECT count(*) FROM vec_personal.relacion_servicio_historia WHERE persona_ref=persona)<>1
     OR (SELECT generacion FROM vec_personal.proyeccion_empleado_persona_control WHERE persona_ref=persona) IS DISTINCT FROM 1
  THEN RAISE EXCEPTION 'HZ39: historia o generación incompleta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $datos$;
COMMIT;
