\set ON_ERROR_STOP on
-- F1-131: solo dato ficticio en una copia PostgreSQL 18 ya preparada.
-- Entradas psql: sid_copia (identificador de sistema), perfil_ref, organismo_ref.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.set_config('vec.f131.sid_copia', (:'sid_copia'::numeric)::text, true);
SELECT pg_catalog.set_config('vec.f131.perfil_ref', :'perfil_ref'::text, true);
SELECT pg_catalog.set_config('vec.f131.organismo_ref', :'organismo_ref'::text, true);
DO $pre$
BEGIN
 IF session_user <> 'postgres' OR current_user <> 'postgres'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system())
       IS DISTINCT FROM current_setting('vec.f131.sid_copia')
    OR vec_contexto_actor_v1.referencia_valida(current_setting('vec.f131.perfil_ref'),'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.organizacion_ref_valida(current_setting('vec.f131.organismo_ref')) IS NOT TRUE
 THEN RAISE EXCEPTION 'F1-131: PostgreSQL, copia o referencias no acreditadas' USING ERRCODE='55000'; END IF;
END $pre$;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:f1-131:corporativo:'||current_setting('vec.f131.sid_copia')||':'||current_setting('vec.f131.perfil_ref'),0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $semilla$
DECLARE
 s record;
 e vec_contexto_actor_v1.vinculo_corporativo_versiones%ROWTYPE;
 a vec_contexto_actor_v1.vinculo_corporativo_actual%ROWTYPE;
 previo vec_contexto_actor_v1.vinculo_corporativo_versiones%ROWTYPE;
 ahora timestamptz := pg_catalog.clock_timestamp();
 sid text := current_setting('vec.f131.sid_copia');
 perfil text := current_setting('vec.f131.perfil_ref');
 organismo text := current_setting('vec.f131.organismo_ref');
BEGIN
 SELECT v.vinculo_ref, v.version AS vinculo_version, v.cuenta_ref, v.perfil_ref, v.persona_ref,
        v.procedencia_ref, v.procedencia_version, v.procedencia_huella_sha256, v.procedencia_autoridad,
        v.vigente_desde AS v_desde, v.vigente_hasta AS v_hasta,
        ca.version AS cuenta_version, c.vigente_desde AS c_desde, c.vigente_hasta AS c_hasta,
        pa.version AS perfil_version, p.vigente_desde AS p_desde, p.vigente_hasta AS p_hasta,
        xa.version AS persona_version, x.vigente_desde AS x_desde, x.vigente_hasta AS x_hasta,
        oa.version AS organizacion_version, o.procedencia_ref AS org_procedencia_ref,
        o.procedencia_version AS org_procedencia_version,
        o.procedencia_huella_sha256 AS org_procedencia_huella,
        o.procedencia_autoridad AS org_procedencia_autoridad,
        o.vigente_desde AS o_desde, o.vigente_hasta AS o_hasta
 INTO STRICT s
 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
   ON v.vinculo_ref=va.vinculo_ref AND v.version=va.version
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c
   ON c.cuenta_ref=ca.cuenta_ref AND c.version=ca.version
 JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.perfil_versiones p
   ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
 JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.persona_versiones x
   ON x.persona_ref=xa.persona_ref AND x.version=xa.version
 JOIN vec_contexto_actor_v1.organizacion_actual oa ON oa.organizacion_ref=organismo
 JOIN vec_contexto_actor_v1.organizacion_versiones o
   ON o.organizacion_ref=oa.organizacion_ref AND o.version=oa.version
 WHERE v.perfil_ref=perfil AND p.persona_ref=v.persona_ref
   AND v.estado='activo' AND c.estado='activo' AND p.estado='activo'
   AND x.estado='activo' AND o.estado='activo'
   AND v.procedencia_autoridad='autoridad_maestra_acreditada'
   AND o.procedencia_autoridad='autoridad_maestra_acreditada'
   AND ahora >= v.vigente_desde AND ahora < v.vigente_hasta
   AND ahora >= c.vigente_desde AND ahora < c.vigente_hasta
   AND ahora >= p.vigente_desde AND ahora < p.vigente_hasta
   AND ahora >= x.vigente_desde AND ahora < x.vigente_hasta
   AND ahora >= o.vigente_desde AND ahora < o.vigente_hasta
 FOR SHARE OF va,v,ca,c,pa,p,xa,x,oa,o;

 e.vinculo_corporativo_ref := 'vcr_'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec:f1-131:corporativo:'||sid||':'||perfil,'UTF8')),'hex'),1,32);
 e.version:=1; e.cuenta_ref:=s.cuenta_ref; e.cuenta_version:=s.cuenta_version;
 e.persona_ref:=s.persona_ref; e.persona_version:=s.persona_version;
 e.perfil_ref:=s.perfil_ref; e.perfil_version:=s.perfil_version;
 e.vinculo_contexto_ref:=s.vinculo_ref; e.vinculo_contexto_version:=s.vinculo_version;
 e.organizacion_ref:=organismo; e.organizacion_version:=s.organizacion_version;
 e.organizacion_procedencia_ref:=s.org_procedencia_ref;
 e.organizacion_procedencia_version:=s.org_procedencia_version;
 e.organizacion_procedencia_huella_sha256:=s.org_procedencia_huella;
 e.organizacion_procedencia_autoridad:=s.org_procedencia_autoridad;
 e.superficie:='interna_corporativa'; e.uso:='consulta_rrhh';
 e.procedencia_ref:=s.procedencia_ref; e.procedencia_version:=s.procedencia_version;
 e.procedencia_huella_sha256:=s.procedencia_huella_sha256;
 e.procedencia_autoridad:=s.procedencia_autoridad; e.estado:='activo';
 SELECT * INTO a FROM vec_contexto_actor_v1.vinculo_corporativo_actual
 WHERE cuenta_ref=e.cuenta_ref AND perfil_ref=e.perfil_ref
   AND superficie=e.superficie AND uso=e.uso FOR UPDATE;
 IF FOUND THEN
   IF a.vinculo_corporativo_ref IS DISTINCT FROM e.vinculo_corporativo_ref OR a.version<>1
   THEN RAISE EXCEPTION 'F1-131: vinculo existente ajeno o retirado' USING ERRCODE='55000'; END IF;
   SELECT * INTO previo FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
    WHERE vinculo_corporativo_ref=e.vinculo_corporativo_ref AND version=1 FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'F1-131: historia original ausente' USING ERRCODE='55000'; END IF;
   e.vigente_desde:=previo.vigente_desde; e.vigente_hasta:=previo.vigente_hasta;
   IF (SELECT pg_catalog.count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
       WHERE vinculo_corporativo_ref=e.vinculo_corporativo_ref)<>1
      OR previo.estado<>'activo' OR ahora<e.vigente_desde OR ahora>=e.vigente_hasta
      OR e.vigente_hasta IS DISTINCT FROM e.vigente_desde + interval '88 days'
      OR e.vigente_desde < greatest(s.v_desde,s.c_desde,s.p_desde,s.x_desde,s.o_desde)
      OR e.vigente_hasta > least(s.v_hasta,s.c_hasta,s.p_hasta,s.x_hasta,s.o_hasta)
      OR pg_catalog.to_jsonb(previo) IS DISTINCT FROM pg_catalog.to_jsonb(e)
   THEN RAISE EXCEPTION 'F1-131: vinculo existente ajeno, retirado o divergente' USING ERRCODE='55000'; END IF;
   RETURN;
 END IF;
 IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
            WHERE vinculo_corporativo_ref=e.vinculo_corporativo_ref)
 THEN RAISE EXCEPTION 'F1-131: historia corporativa previa sin puntero exacto' USING ERRCODE='55000'; END IF;
 e.vigente_desde:=greatest(s.v_desde,s.c_desde,s.p_desde,s.x_desde,s.o_desde,
                           pg_catalog.date_trunc('second',ahora)-interval '1 minute');
 e.vigente_hasta:=e.vigente_desde + interval '88 days';
 IF e.vigente_hasta > least(s.v_hasta,s.c_hasta,s.p_hasta,s.x_hasta,s.o_hasta)
    OR ahora < e.vigente_desde OR ahora >= e.vigente_hasta
 THEN RAISE EXCEPTION 'F1-131: no cabe la vigencia de 88 dias en las fuentes' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 (vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,
  perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,
  organizacion_version,organizacion_procedencia_ref,organizacion_procedencia_version,
  organizacion_procedencia_huella_sha256,organizacion_procedencia_autoridad,superficie,uso,
  procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
  estado,vigente_desde,vigente_hasta)
 VALUES
 (e.vinculo_corporativo_ref,e.version,e.cuenta_ref,e.cuenta_version,e.persona_ref,e.persona_version,
  e.perfil_ref,e.perfil_version,e.vinculo_contexto_ref,e.vinculo_contexto_version,e.organizacion_ref,
  e.organizacion_version,e.organizacion_procedencia_ref,e.organizacion_procedencia_version,
  e.organizacion_procedencia_huella_sha256,e.organizacion_procedencia_autoridad,e.superficie,e.uso,
  e.procedencia_ref,e.procedencia_version,e.procedencia_huella_sha256,e.procedencia_autoridad,
  e.estado,e.vigente_desde,e.vigente_hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual
 (cuenta_ref,perfil_ref,superficie,uso,vinculo_corporativo_ref,version)
 VALUES (e.cuenta_ref,e.perfil_ref,e.superficie,e.uso,e.vinculo_corporativo_ref,e.version);
END $semilla$;
COMMIT;
