\set ON_ERROR_STOP on
-- Tras CT241 (F1-19), en una copia desechable con datos (necesita una
-- selección O6 de un expediente ordinario). Todo termina en ROLLBACK.
-- El hijo es una fila de alta con derivacion_ref, la marca que CT234 deja en
-- todo nombramiento derivado; su clave foránea es diferida y no llega a
-- comprobarse porque no hay COMMIT. No se finge ninguna decisión V3: en la
-- oferta y el vínculo, el expediente ordinario sigue hasta el consumo de la
-- autorización y falla allí, después de la guarda.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';

DO $estructura$
DECLARE x record; c text;
BEGIN
 FOR x IN SELECT f.firma,f.huella,f.consumo,p.prosecdef,p.proacl::text AS acl,p.proconfig,
   pg_get_functiondef(p.oid) AS def
  FROM (VALUES
   ('vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(uuid,text,text)',
    'af2d9ff4aa9a26c12c2445833473b3427b9f6be4731c2c7f12121c5eaa249529',
    'INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'),
   ('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    '6a8e5f1feff9051e960a8db3866669569c5c551eca009d7f6889c87f88e383e8',
    'consumir_etapa_seleccion_oferta_ct220('),
   ('vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    'fd7421e9d5dba11dda82368873a8ea41a30fc5002f921e0f226b649ba13ac344',
    'leer_ambitos_vinculo_emision_bolsa_ct_v1(')
  ) AS f(firma,huella,consumo)
  JOIN pg_proc p ON p.oid=f.firma::regprocedure LOOP
  c:=x.def;
  IF x.prosecdef IS NOT TRUE
     OR encode(sha256(convert_to(c,'UTF8')),'hex') IS DISTINCT FROM x.huella
     OR x.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
     OR NOT ('search_path=pg_catalog, pg_temp'=ANY(x.proconfig))
  THEN RAISE EXCEPTION 'CT241: cuerpo, ACL o configuración divergente: %',x.firma; END IF;
  -- La guarda va antes del efecto (O6) o antes de leer ámbitos y consumir la
  -- autorización (oferta y vínculo).
  IF NOT (position('CT241: expediente derivado sin selección nueva' IN c)>0
     AND position('CT241: expediente derivado sin selección nueva' IN c)<position(x.consumo IN c))
  THEN RAISE EXCEPTION 'CT241: guarda ausente o tardía: %',x.firma; END IF;
 END LOOP;
END $estructura$;

-- Expediente ordinario con una selección O6 y su hijo de prueba.
CREATE TEMP TABLE ct241_caso ON COMMIT DROP AS
 SELECT e.solicitud_json AS s0, e.solicitud_json->>'expediente_ref' AS padre,
  'expediente:ct:'||encode(sha256(convert_to('ct241-prueba-hijo','UTF8')),'hex') AS hijo
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 JOIN vec_contratacion_temporal.expediente_alta a
   ON a.expediente_ref=e.solicitud_json->>'expediente_ref' AND a.derivacion_ref IS NULL
 WHERE e.tipo_seleccion='llamamiento'
 ORDER BY e.creada_en LIMIT 1;
DO $caso$ BEGIN
 IF (SELECT count(*) FROM pg_temp.ct241_caso)<>1 THEN
  RAISE EXCEPTION 'CT241: la prueba necesita una copia con una selección O6 ordinaria'; END IF;
END $caso$;
INSERT INTO vec_contratacion_temporal.expediente_alta(expediente_ref,reserva_ref,numero_visible,
 organizacion_ref,actor_ref,perfil_ref,decision_ref,efecto_ref,huella_efecto_sha256,creada_en,
 confirmacion_ref,derivacion_ref)
SELECT k.hijo,NULL,a.numero_visible||'-P1',a.organizacion_ref,a.actor_ref,a.perfil_ref,
 'decision:ct241-prueba','efecto:ct241-prueba',a.huella_efecto_sha256,a.creada_en,NULL,
 'derivacion:ct241-prueba'
FROM pg_temp.ct241_caso k
JOIN vec_contratacion_temporal.expediente_alta a ON a.expediente_ref=k.padre;

-- Solicitudes O6 canónicas: misma solicitud del padre con otra clave, y la
-- del hijo. Material de oferta y de vínculo con la forma que exigen.
CREATE TEMP TABLE ct241_material ON COMMIT DROP AS
WITH base AS (
 SELECT k.*,x.quien,x.ref,gen_random_uuid() AS clave
 FROM pg_temp.ct241_caso k
 CROSS JOIN LATERAL (VALUES ('padre',k.padre),('hijo',k.hijo)) AS x(quien,ref)
), solicitud AS (
 SELECT b.*,b.s0||jsonb_build_object('expediente_ref',b.ref,'clave_idempotencia',b.clave::text) AS s
 FROM base b
), firmada AS (
 SELECT q.*,vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(q.s) AS huella
 FROM solicitud q
)
SELECT f.quien,f.ref,f.clave,f.huella,
 vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(
  f.s||jsonb_build_object('huella_semantica',f.huella)) AS texto_o6,
 jsonb_build_object('organizacion_ref',f.s->>'organizacion_ref','expediente_ref',f.ref,
  'version_expediente',6,'clave_idempotencia',f.clave::text,
  'actor_ref','per_ct241pruebaactor000000000','perfil_ref','perfil:ct241:prueba',
  'categoria_ref','categoria:ct241','centro_ref','centro:ct241',
  'huella_intencion',repeat('a',64)) AS intencion,
 jsonb_build_object('organizacion_ref',f.s->>'organizacion_ref','expediente_ref',f.ref,
  'version_expediente',6,'actor_ref','per_ct241pruebaactor000000000',
  'bolsa_ref','bolsa:ct241','necesidad_ref','necesidad:ct241','numero_plazas',2,
  'numero_elegibles',2,'inicio_plazo','publicacion',
  'clave_bolsa','ct-oferta:'||encode(sha256(convert_to((f.s->>'organizacion_ref')||chr(31)||
   f.ref||chr(31)||'6'||chr(31)||f.clave::text,'UTF8')),'hex'),
  'datos',jsonb_build_object('categoria','c','centro','c','fecha_inicio','2026-11-01',
   'descripcion','d','modalidad_clave','interinidad','causa_clave','vacante'),
  'origen_ct',jsonb_build_object('organizacion_ref',f.s->>'organizacion_ref',
   'expediente_ref',f.ref,'version_expediente',6,'clave_ct',f.clave::text,
   'bolsa_ref','bolsa:ct241','necesidad_ref','necesidad:ct241','categoria_ref','categoria:ct241',
   'lectura',jsonb_build_object('entradas','[]'::jsonb,'corte','ct241'))) AS comando,
 '{"esquema":"vec.ct.vinculo-emision-bolsa.v1","organizacion_ref":'||to_json(f.s->>'organizacion_ref')::text||
  ',"expediente_ref":'||to_json(f.ref)::text||',"version_esperada":6,"bolsa_ref":"bolsa:ct241",'||
  '"llamamiento_ref":"llamamiento:'||repeat('b',64)||'","recibo_emision_ref":"recibo:llamamiento:'||
  repeat('c',64)||'","clave_idempotencia":"'||f.clave::text||'"}' AS material_vinculo
FROM firmada f;
DO $material$ DECLARE m record; BEGIN
 FOR m IN SELECT * FROM pg_temp.ct241_material LOOP
  IF vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(m.texto_o6) IS NULL
     OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(m.comando,m.intencion) IS NOT TRUE
     OR octet_length(m.material_vinculo) NOT BETWEEN 128 AND 4096
  THEN RAISE EXCEPTION 'CT241: material de prueba mal formado (%)',m.quien; END IF;
 END LOOP;
END $material$;

-- Sesión del ejecutor CT, como la aplicación: miembro de ejecutor y no del
-- propietario. El rol existe solo dentro de esta transacción.
CREATE ROLE ct241_prueba_ejecutor NOLOGIN IN ROLE vec_contratacion_temporal_ejecutor;
GRANT SELECT ON pg_temp.ct241_material TO ct241_prueba_ejecutor;
SET SESSION AUTHORIZATION ct241_prueba_ejecutor;

DO $vias$
DECLARE m record; r record; estado text; mensaje text;
BEGIN
 FOR m IN SELECT * FROM pg_temp.ct241_material ORDER BY quien DESC LOOP
  -- 1. Reserva singular O6.
  BEGIN
   SELECT * INTO r FROM vec_contratacion_temporal.reservar_seleccion_llamamiento_o6_v1(
    m.clave,m.huella,m.texto_o6);
   estado:='00000'; mensaje:=r.situacion;
  EXCEPTION WHEN OTHERS THEN
   GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  END;
  IF m.quien='hijo' AND (estado<>'23505' OR mensaje<>'CT241: expediente derivado sin selección nueva')
  THEN RAISE EXCEPTION 'CT241: O6 admitió al hijo (% %)',estado,mensaje; END IF;
  IF m.quien='padre' AND (estado<>'00000' OR mensaje<>'propietaria')
  THEN RAISE EXCEPTION 'CT241: O6 cambió para el ordinario (% %)',estado,mensaje; END IF;

  -- 2. Reserva de oferta: el ordinario llega al consumo de la autorización.
  BEGIN
   PERFORM vec_contratacion_temporal.reservar_seleccion_oferta_ct220(m.intencion,m.comando,
    'material',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
   estado:='00000'; mensaje:='';
  EXCEPTION WHEN OTHERS THEN
   GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  END;
  IF m.quien='hijo' AND (estado<>'23505' OR mensaje<>'CT241: expediente derivado sin selección nueva')
  THEN RAISE EXCEPTION 'CT241: oferta admitió al hijo (% %)',estado,mensaje; END IF;
  IF m.quien='padre' AND (estado='00000' OR mensaje LIKE 'CT241%')
  THEN RAISE EXCEPTION 'CT241: oferta del ordinario sin consumo (% %)',estado,mensaje; END IF;

  -- 3. Vínculo de emisión: el ordinario llega a sus ámbitos o a la decisión.
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(m.material_vinculo,
    NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
   estado:='00000'; mensaje:='';
  EXCEPTION WHEN OTHERS THEN
   GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  END;
  IF m.quien='hijo' AND (estado<>'23505' OR mensaje<>'CT241: expediente derivado sin selección nueva')
  THEN RAISE EXCEPTION 'CT241: vínculo admitió al hijo (% %)',estado,mensaje; END IF;
  IF m.quien='padre' AND (estado='00000' OR mensaje LIKE 'CT241%'
     OR mensaje NOT IN ('CT201: ámbitos del expediente divergentes','CT201: decisión no ligada al material'))
  THEN RAISE EXCEPTION 'CT241: vínculo del ordinario alterado (% %)',estado,mensaje; END IF;
 END LOOP;
END $vias$;

RESET SESSION AUTHORIZATION;
SELECT 'ct241_derivado_sin_seleccion: 0 fallos' AS resultado;
ROLLBACK;
