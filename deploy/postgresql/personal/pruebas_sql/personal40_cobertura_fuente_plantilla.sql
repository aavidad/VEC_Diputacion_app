\set ON_ERROR_STOP on
-- Personal40 y la carga organizacion_rpt_2026_v1. Se ejecuta como superusuario
-- en una copia desechable después de ambas; todo termina en ROLLBACK.
--   psql -X -v organismo=<organismo_ref> -f personal40_cobertura_fuente_plantilla.sql
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_catalog.set_config('vec.organismo', :'organismo', true);
DO $prueba$
DECLARE
 org text:=pg_catalog.current_setting('vec.organismo');
 ns text:='vec:personal:organizacion-rpt-2026:v1:'||org||':';
 v_pla uuid:=pg_catalog.md5(ns||'plantilla')::uuid;
 p record; s jsonb; n integer; fallo text;
BEGIN
 SELECT * INTO STRICT p FROM vec_personal.version_plantilla_historia WHERE version_ref=v_pla AND revision=1;
 SET LOCAL ROLE vec_personal_propietario;
 -- 1. Selección de plaza y puesto del plan B2 (Personal23) para cada plaza.
 FOR s IN SELECT jsonb_build_object('plaza_ref','plaza:'||pl.plaza_ref,'puesto_ref','puesto:'||v.puesto_ref,'desde','2026-11-02')
   FROM vec_personal.plaza_plantilla_historia pl JOIN vec_personal.vinculo_plaza_puesto_historia v ON v.plaza_ref=pl.plaza_ref
   WHERE pl.plantilla_version_ref=v_pla LOOP
  s:=vec_personal.seleccion_plan_ct_interna(org,s);
  IF s->>'version_plantilla_ref'<>'plantilla:'||v_pla OR (s->>'revision_rpt')::int<>1 OR s->>'rpt_fuente_ref' IS NULL THEN
   RAISE EXCEPTION 'P40: selección incompleta %',s; END IF;
 END LOOP;
 -- 2. Mismas condiciones que la consulta de vacantes de Personal18.
 SELECT count(*) INTO n FROM vec_personal.cobertura_ocupaciones_historia
  WHERE organismo_ref=org AND estado='completa' AND plantilla_version_ref=v_pla AND vigente_desde<='2026-11-02' AND '2026-11-02'<vigente_hasta;
 IF n<>1 THEN RAISE EXCEPTION 'P40: cobertura no unívoca (%)',n; END IF;
 SELECT count(*) INTO n FROM vec_personal.plaza_plantilla_historia pl
  JOIN vec_personal.vinculo_plaza_puesto_historia v ON v.plaza_ref=pl.plaza_ref AND v.estado='confirmado'
  JOIN vec_personal.puesto_rpt_historia pu ON pu.puesto_ref=v.puesto_ref
  JOIN vec_personal.puesto_tipo_historia pt ON pt.tipo_ref=pu.tipo_ref AND pt.revision=pu.tipo_revision
  WHERE pl.plantilla_version_ref=v_pla AND pl.dotacion_presupuestaria='acreditada' AND pl.estado_estructural='vigente'
    AND pt.denominacion<>'' AND pl.codigo_plaza_fuente<>'';
 IF n<>32 THEN RAISE EXCEPTION 'P40: vacantes %',n; END IF;
 -- 3. «Completa» declarada por otra fuente, otra huella u otro acto: rechazada.
 FOREACH fallo IN ARRAY ARRAY['fuente','huella','acto'] LOOP
  BEGIN
   INSERT INTO vec_personal.cobertura_ocupaciones_historia(cobertura_ref,revision,organismo_ref,plantilla_version_ref,
     plantilla_revision,estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256)
   VALUES(gen_random_uuid(),1,org,v_pla,1,'completa','2026-01-01','2027-01-01',now(),
     CASE WHEN fallo='acto' THEN 'acto:otro' ELSE p.acto_ref END,
     CASE WHEN fallo='fuente' THEN 'fuente:otra' ELSE p.fuente_ref END,1,
     CASE WHEN fallo='huella' THEN pg_catalog.repeat('0',64) ELSE p.huella_fuente_sha256 END);
   RAISE EXCEPTION 'P40: completa con % distinto admitida',fallo;
  EXCEPTION WHEN SQLSTATE 'P7401' THEN NULL;
  END;
 END LOOP;
 -- 4. «Parcial» sigue admitiéndose con cualquier fuente, como en Personal17.
 INSERT INTO vec_personal.cobertura_ocupaciones_historia(cobertura_ref,revision,organismo_ref,plantilla_version_ref,
   plantilla_revision,estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256)
 VALUES(gen_random_uuid(),1,org,v_pla,1,'parcial','2026-01-01','2027-01-01',now(),'acto:otro','fuente:otra',1,pg_catalog.repeat('0',64));
 RESET ROLE;
 -- 5. La aplicación no escribe ni lee estas tablas directamente.
 IF pg_catalog.has_table_privilege('vec_personal_ejecutor','vec_personal.cobertura_ocupaciones_historia','INSERT')
  OR pg_catalog.has_table_privilege('vec_personal_ejecutor','vec_personal.plaza_plantilla_historia','SELECT') THEN
  RAISE EXCEPTION 'P40: privilegios de la aplicación'; END IF;
 RAISE NOTICE 'P40: OK';
END $prueba$;
ROLLBACK;
