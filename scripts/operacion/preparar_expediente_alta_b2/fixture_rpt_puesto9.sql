\set ON_ERROR_STOP on
-- SOLO clon sintético B2: añade puesto/plaza 9 y unidad si faltan.
-- Ficción técnica identificable; no publica ni rectifica la RPT institucional.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT set_config('vec.fixture.organismo', :'organismo', true);
SET LOCAL ROLE vec_personal_propietario;
DO $fixture$
DECLARE
  org constant text:=current_setting('vec.fixture.organismo');
  unidad constant text:='unidad:centro:520';
  fuente constant text:='fuente:ensayo:rpt520:b2';
  acto constant text:='acto:ensayo:rpt520:b2';
  huella text:=encode(sha256(convert_to('fixture-rpt-puesto9-b2-20261012','UTF8')),'hex');
  ahora timestamptz(6):=clock_timestamp();
  tipo_id uuid:=md5('fixture-b2-tipo9')::uuid;
  dota_id uuid:=md5('fixture-b2-dotacion9')::uuid;
  puesto_id uuid:=md5('fixture-b2-puesto9')::uuid;
  plaza_id uuid:=md5('fixture-b2-plaza9')::uuid;
  vinculo_id uuid:=md5('fixture-b2-vinculo9')::uuid;
  nodo_id uuid:=md5('fixture-b2-centro520')::uuid;
  st vec_personal.puesto_tipo_historia%ROWTYPE;
  sd vec_personal.dotacion_rpt_historia%ROWTYPE;
  sp vec_personal.puesto_rpt_historia%ROWTYPE;
  sl vec_personal.plaza_plantilla_historia%ROWTYPE;
  sv vec_personal.vinculo_plaza_puesto_historia%ROWTYPE;
  n_tipo integer;
  n_puesto integer;
  n_plaza integer;
  n_vinculo integer;
  n_ids integer;
  n_fixture integer;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtext('fixture-rpt-b2-puesto9'),hashtext(org));
 SELECT count(*) INTO n_tipo FROM vec_personal.puesto_tipo_historia
  WHERE organismo_ref=org AND unidad_ref=unidad AND codigo_fila_fuente='9';
 SELECT count(*) INTO n_puesto FROM vec_personal.puesto_rpt_historia
  WHERE organismo_ref=org AND unidad_ref=unidad AND codigo_puesto_fuente='9/01';
 SELECT count(*) INTO n_plaza FROM vec_personal.plaza_plantilla_historia
  WHERE organismo_ref=org AND unidad_ref=unidad AND codigo_plaza_fuente='2026/0033';
 SELECT count(*) INTO n_vinculo FROM vec_personal.vinculo_plaza_puesto_historia v
  JOIN vec_personal.puesto_rpt_historia p ON p.puesto_ref=v.puesto_ref AND p.organismo_ref=v.organismo_ref
  JOIN vec_personal.plaza_plantilla_historia l ON l.plaza_ref=v.plaza_ref AND l.organismo_ref=v.organismo_ref
  WHERE v.organismo_ref=org AND v.unidad_ref=unidad AND p.unidad_ref=unidad AND l.unidad_ref=unidad
    AND p.codigo_puesto_fuente='9/01' AND l.codigo_plaza_fuente='2026/0033'
    AND NOT p.retirado AND NOT l.retirado AND NOT v.retirado
    AND p.vigente_desde<='2026-12-01' AND l.vigente_desde<='2026-12-01' AND v.vigente_desde<='2026-12-01'
    AND (p.vigente_hasta IS NULL OR p.vigente_hasta>='2027-01-31')
    AND (l.vigente_hasta IS NULL OR l.vigente_hasta>='2027-01-31')
    AND (v.vigente_hasta IS NULL OR v.vigente_hasta>='2027-01-31');
 SELECT (SELECT count(*) FROM vec_personal.puesto_tipo_historia WHERE tipo_ref=tipo_id)
      + (SELECT count(*) FROM vec_personal.dotacion_rpt_historia WHERE dotacion_ref=dota_id)
      + (SELECT count(*) FROM vec_personal.puesto_rpt_historia WHERE puesto_ref=puesto_id)
      + (SELECT count(*) FROM vec_personal.plaza_plantilla_historia WHERE plaza_ref=plaza_id)
      + (SELECT count(*) FROM vec_personal.vinculo_plaza_puesto_historia WHERE vinculo_ref=vinculo_id)
   INTO n_ids;
 SELECT count(*) INTO n_fixture FROM vec_personal.puesto_tipo_historia t
  JOIN vec_personal.dotacion_rpt_historia d ON d.tipo_ref=t.tipo_ref
  JOIN vec_personal.puesto_rpt_historia p ON p.tipo_ref=t.tipo_ref
  JOIN vec_personal.plaza_plantilla_historia l ON l.organismo_ref=p.organismo_ref AND l.unidad_ref=p.unidad_ref
  JOIN vec_personal.vinculo_plaza_puesto_historia v ON v.puesto_ref=p.puesto_ref AND v.plaza_ref=l.plaza_ref
  WHERE t.tipo_ref=tipo_id AND d.dotacion_ref=dota_id AND p.puesto_ref=puesto_id
    AND l.plaza_ref=plaza_id AND v.vinculo_ref=vinculo_id
    AND t.organismo_ref=org AND d.organismo_ref=org AND p.organismo_ref=org AND l.organismo_ref=org AND v.organismo_ref=org
    AND t.unidad_ref=unidad AND d.unidad_ref=unidad AND p.unidad_ref=unidad AND l.unidad_ref=unidad AND v.unidad_ref=unidad
    AND t.codigo_fila_fuente='9' AND p.codigo_puesto_fuente='9/01' AND l.codigo_plaza_fuente='2026/0033'
    AND NOT t.retirado AND NOT d.retirado AND NOT p.retirado AND NOT l.retirado AND NOT v.retirado
    AND t.fuente_ref=fuente AND d.fuente_ref=fuente AND p.fuente_ref=fuente AND l.fuente_ref=fuente AND v.fuente_ref=fuente;
 IF EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia WHERE nodo_ref=nodo_id
   AND (organismo_ref<>org OR unidad_ref<>unidad)) THEN
   RAISE EXCEPTION 'fixture_rpt_puesto9: nodo determinista ajeno';
 END IF;
 IF EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia WHERE organismo_ref=org AND unidad_ref=unidad)
    AND NOT EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia WHERE organismo_ref=org AND unidad_ref=unidad
      AND NOT retirado AND vigente_desde<='2026-12-01'
      AND (vigente_hasta IS NULL OR vigente_hasta>='2027-01-31')) THEN
   RAISE EXCEPTION 'fixture_rpt_puesto9: unidad histórica sin vigencia para el periodo';
 END IF;
 IF EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia WHERE nodo_ref=nodo_id) AND n_ids<>5 THEN
   RAISE EXCEPTION 'fixture_rpt_puesto9: nodo determinista parcial';
 END IF;
 IF n_tipo=1 AND n_puesto=1 AND n_plaza=1 AND n_vinculo=1
    AND (n_ids=0 OR (n_ids=5 AND n_fixture=1)) THEN
   RETURN;
 END IF;
 IF n_tipo<>0 OR n_puesto<>0 OR n_plaza<>0 OR n_vinculo<>0 OR n_ids<>0 THEN
   RAISE EXCEPTION 'fixture_rpt_puesto9: preimagen parcial o ajena (tipo %, puesto %, plaza %, vinculo %, ids %)',
     n_tipo,n_puesto,n_plaza,n_vinculo,n_ids;
 END IF;
 SELECT * INTO STRICT st FROM vec_personal.puesto_tipo_historia WHERE organismo_ref=org AND codigo_fila_fuente='10' AND revision=1;
 SELECT * INTO STRICT sd FROM vec_personal.dotacion_rpt_historia WHERE organismo_ref=org AND tipo_ref=st.tipo_ref AND revision=1;
 SELECT * INTO STRICT sp FROM vec_personal.puesto_rpt_historia WHERE organismo_ref=org AND tipo_ref=st.tipo_ref AND revision=1;
 SELECT * INTO STRICT sl FROM vec_personal.plaza_plantilla_historia WHERE organismo_ref=org AND codigo_plaza_fuente='2026/0001' AND revision=1;
 SELECT * INTO STRICT sv FROM vec_personal.vinculo_plaza_puesto_historia WHERE organismo_ref=org AND plaza_ref=sl.plaza_ref AND puesto_ref=sp.puesto_ref AND revision=1;
 IF NOT EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia WHERE organismo_ref=org AND unidad_ref=unidad) THEN
   INSERT INTO vec_personal.org_nodo_historia(nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,catalogo_entrada_clave,denominacion,centro_padre_ref,retirado,vigente_desde,vigente_hasta,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
   VALUES(nodo_id,1,org,unidad,'centro','estructura-organizativa-dipgra',1,1,'centro_520','Transformación Digital',NULL,false,'2026-01-01',NULL,ahora,fuente,acto,huella);
 END IF;
 INSERT INTO vec_personal.puesto_tipo_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.puesto_tipo_historia,to_jsonb(st)||jsonb_build_object(
  'tipo_ref',tipo_id::text,'unidad_ref',unidad,'codigo_fila_fuente','9','denominacion','ADMINISTRATIVO',
  'conocido_desde',ahora,'fuente_ref',fuente,'acto_ref',acto,'huella_fuente_sha256',huella))).*;
 INSERT INTO vec_personal.dotacion_rpt_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.dotacion_rpt_historia,to_jsonb(sd)||jsonb_build_object(
  'dotacion_ref',dota_id::text,'unidad_ref',unidad,'tipo_ref',tipo_id::text,'cantidad',1,
  'conocido_desde',ahora,'fuente_ref',fuente,'acto_ref',acto,'huella_fuente_sha256',huella))).*;
 INSERT INTO vec_personal.puesto_rpt_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.puesto_rpt_historia,to_jsonb(sp)||jsonb_build_object(
  'puesto_ref',puesto_id::text,'unidad_ref',unidad,'tipo_ref',tipo_id::text,'codigo_puesto_fuente','9/01',
  'conocido_desde',ahora,'fuente_ref',fuente,'acto_ref',acto,'huella_fuente_sha256',huella))).*;
 INSERT INTO vec_personal.plaza_plantilla_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.plaza_plantilla_historia,to_jsonb(sl)||jsonb_build_object(
  'plaza_ref',plaza_id::text,'unidad_ref',unidad,'codigo_plaza_fuente','2026/0033',
  'conocido_desde',ahora,'fuente_ref',fuente,'acto_ref',acto,'huella_fuente_sha256',huella))).*;
 INSERT INTO vec_personal.vinculo_plaza_puesto_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.vinculo_plaza_puesto_historia,to_jsonb(sv)||jsonb_build_object(
  'vinculo_ref',vinculo_id::text,'unidad_ref',unidad,'plaza_ref',plaza_id::text,'puesto_ref',puesto_id::text,
  'conocido_desde',ahora,'fuente_ref',fuente,'acto_ref',acto,'huella_fuente_sha256',huella))).*;
 SELECT count(*) INTO n_fixture FROM vec_personal.puesto_tipo_historia t
  JOIN vec_personal.dotacion_rpt_historia d ON d.tipo_ref=t.tipo_ref
  JOIN vec_personal.puesto_rpt_historia p ON p.tipo_ref=t.tipo_ref
  JOIN vec_personal.plaza_plantilla_historia l ON l.organismo_ref=p.organismo_ref AND l.unidad_ref=p.unidad_ref
  JOIN vec_personal.vinculo_plaza_puesto_historia v ON v.puesto_ref=p.puesto_ref AND v.plaza_ref=l.plaza_ref
  WHERE t.tipo_ref=tipo_id AND d.dotacion_ref=dota_id AND p.puesto_ref=puesto_id
    AND l.plaza_ref=plaza_id AND v.vinculo_ref=vinculo_id
    AND t.organismo_ref=org AND d.organismo_ref=org AND p.organismo_ref=org AND l.organismo_ref=org AND v.organismo_ref=org
    AND t.unidad_ref=unidad AND d.unidad_ref=unidad AND p.unidad_ref=unidad AND l.unidad_ref=unidad AND v.unidad_ref=unidad
    AND t.codigo_fila_fuente='9' AND p.codigo_puesto_fuente='9/01' AND l.codigo_plaza_fuente='2026/0033'
    AND NOT t.retirado AND NOT d.retirado AND NOT p.retirado AND NOT l.retirado AND NOT v.retirado
    AND t.fuente_ref=fuente AND d.fuente_ref=fuente AND p.fuente_ref=fuente AND l.fuente_ref=fuente AND v.fuente_ref=fuente;
 IF n_fixture<>1 THEN RAISE EXCEPTION 'fixture_rpt_puesto9: resultado incompleto antes de COMMIT'; END IF;
 RAISE NOTICE 'fixture RPT puesto9: nodo, tipo, dotación, puesto, plaza y vínculo insertados solo en clon';
END $fixture$;
COMMIT;
