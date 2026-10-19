\set ON_ERROR_STOP on
-- Tras CT241 (F1-19), en una copia desechable con datos (necesita una
-- selección O6 de un expediente ordinario). Todo termina en ROLLBACK.
-- Cada vía rechaza al hijo y deja pasar al expediente ordinario hasta el paso
-- siguiente, que es donde falla ahora por falta de autorización real.
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
    '52a4058e93be16793cdd9cadc0778b452d48385848dbe167c58a2f84855b3730',
    'INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'),
   ('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    '6a8e5f1feff9051e960a8db3866669569c5c551eca009d7f6889c87f88e383e8',
    'consumir_etapa_seleccion_oferta_ct220('),
   ('vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    'fd7421e9d5dba11dda82368873a8ea41a30fc5002f921e0f226b649ba13ac344',
    'leer_ambitos_vinculo_emision_bolsa_ct_v1('),
   ('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    '82941407107240e6c27850e86c85b6d8706fe9fea8486f3e23ae207ebb4947f9',
    'registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada('),
   ('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)',
    'e596c2e5a65f4a29ce306021de21cfe416767c71f58006247d1ee76534284a92',
    'RETURN QUERY'),
   ('vec_contratacion_temporal.consultar_circuito_rrhh_v1(text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    '0081675ff4ab7e4961dc1ee1f01b96922ccf78291f3d3eb9af4ce95e45fae732',
    'registrar_y_consumir_consulta_circuito_ct_v3_atestada('),
   ('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)',
    '0f2879ac5905b513bd313efb28711d517e2db84b21ae2845bba40ee883b4d09b',
    'JOIN vec_contratacion_temporal.expediente_integral_actual a'),
   ('vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)',
    '59fff3ea60ad4cfdb021766e491dbd2ad1f86ff352be74a54d13eca8deb105ab',
    'JOIN vec_contratacion_temporal.expediente_integral_actual i')
  ) AS f(firma,huella,consumo)
  JOIN pg_proc p ON p.oid=f.firma::regprocedure LOOP
  c:=x.def;
  IF x.prosecdef IS NOT TRUE
     OR encode(sha256(convert_to(c,'UTF8')),'hex') IS DISTINCT FROM x.huella
     OR x.acl IS DISTINCT FROM CASE
        WHEN x.firma LIKE '%leer_modalidad_origen_bolsa_ct_v1%' THEN '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_contratacion_temporal_propietario,vec_bolsa_llamamientos_propietario=X/vec_contratacion_temporal_propietario}'
        WHEN x.firma LIKE '%cotejar_comprobacion_bolsa_v1%' THEN '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_bolsa_llamamientos_propietario=X/vec_contratacion_temporal_propietario}'
        ELSE '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}' END
     OR NOT ('search_path=pg_catalog, pg_temp'=ANY(x.proconfig))
  THEN RAISE EXCEPTION 'CT241: cuerpo, ACL o configuración divergente: %',x.firma; END IF;
  -- La guarda va antes del efecto (O6), antes de leer ámbitos y consumir la
  -- autorización (oferta y vínculo) o antes de leer el agregado (CT221, CT233).
  IF NOT (position('CT241: ' IN c)>0 AND position('CT241: ' IN c)<position(x.consumo IN c))
  THEN RAISE EXCEPTION 'CT241: guarda ausente o tardía: %',x.firma; END IF;
 END LOOP;
 -- El disparador central de escritura va primero en las tres tablas.
 IF (SELECT count(*) FROM pg_trigger t
     WHERE t.tgname='expediente_derivado_ct241' AND t.tgenabled='O' AND t.tgtype=7
       AND t.tgfoid='vec_contratacion_temporal.rechazar_nombramiento_derivado_ct241()'::regprocedure
       AND NOT EXISTS(SELECT 1 FROM pg_trigger o WHERE o.tgrelid=t.tgrelid AND NOT o.tgisinternal
         AND (o.tgtype & 6)=6 AND o.tgname<t.tgname))<>3
    OR encode(sha256(convert_to(pg_get_functiondef(
       'vec_contratacion_temporal.rechazar_nombramiento_derivado_ct241()'::regprocedure),'UTF8')),'hex')
       IS DISTINCT FROM '75d464994382f006ec6956673661ba894c4cc8fa65536e5beb6fc9c0947d9ae9'
 THEN RAISE EXCEPTION 'CT241: disparador central ausente o no primero'; END IF;
END $estructura$;

-- Expediente ordinario con una selección O6 y su hijo de prueba.
CREATE TEMP TABLE ct241_caso ON COMMIT DROP AS
 SELECT e.solicitud_json AS s0, e.solicitud_json->>'expediente_ref' AS padre,
  'expediente:ct:'||encode(sha256(convert_to('ct241-prueba-hijo','UTF8')),'hex') AS hijo,
  a.organizacion_ref AS organizacion, v.agregado_json#>>'{analisis,categoria_ref}' AS categoria,
  v.agregado_json#>>'{asignacion,unidad_ref}' AS unidad
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 JOIN vec_contratacion_temporal.expediente_alta a
   ON a.expediente_ref=e.solicitud_json->>'expediente_ref' AND a.derivacion_ref IS NULL
 JOIN vec_contratacion_temporal.expediente_integral_actual i ON i.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=i.expediente_ref AND v.version=i.version
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
  repeat('c',64)||'","clave_idempotencia":"'||f.clave::text||'"}' AS material_vinculo,
 jsonb_build_object('esquema','vec.ct.seleccion-oferta.consulta-expediente.v1',
  'organizacion_ref',f.s->>'organizacion_ref','expediente_ref',f.ref,'version_observada',6,
  'categoria_ref','categoria:ct241','centro_ref','centro:ct241',
  'actor_ref','per_ct241pruebaactor000000000','perfil_ref','perfil:ct241:prueba')::text AS material_estado
FROM firmada f;
DO $material$ DECLARE m record; BEGIN
 FOR m IN SELECT * FROM pg_temp.ct241_material LOOP
  IF vec_contratacion_temporal.solicitud_desde_texto_seleccion_llamamiento_o6_v1(m.texto_o6) IS NULL
     OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(m.comando,m.intencion) IS NOT TRUE
     OR octet_length(m.material_vinculo) NOT BETWEEN 128 AND 4096
     OR octet_length(m.material_estado) NOT BETWEEN 128 AND 65536
  THEN RAISE EXCEPTION 'CT241: material de prueba mal formado (%)',m.quien; END IF;
 END LOOP;
END $material$;

-- Sesión del ejecutor CT, como la aplicación: miembro de ejecutor y no del
-- propietario. El rol existe solo dentro de esta transacción.
CREATE ROLE ct241_prueba_ejecutor NOLOGIN IN ROLE vec_contratacion_temporal_ejecutor;
GRANT SELECT ON pg_temp.ct241_material TO ct241_prueba_ejecutor;
GRANT SELECT ON pg_temp.ct241_caso TO vec_contratacion_temporal_propietario,
 vec_bolsa_llamamientos_propietario;
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

  -- 4. Lecturas con la unidad o el circuito copiados del padre.
  FOR r IN SELECT * FROM (VALUES ('contexto_oferta'),('estado_oferta'),('circuito')) AS v(via) LOOP
   BEGIN
    IF r.via='contexto_oferta' THEN
     PERFORM * FROM vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(
      m.intencion->>'organizacion_ref',m.ref);
    ELSIF r.via='estado_oferta' THEN
     PERFORM * FROM vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
      m.material_estado,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
    ELSE
     PERFORM * FROM vec_contratacion_temporal.consultar_circuito_rrhh_v1(
      m.intencion->>'organizacion_ref',m.ref,6,NULL,'\x7b7d'::bytea,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
    END IF;
    estado:='00000'; mensaje:='';
   EXCEPTION WHEN OTHERS THEN
    GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
   END;
   IF m.quien='hijo' AND (estado<>'42501' OR mensaje<>'CT241: nombramiento derivado sin autoridad propia')
   THEN RAISE EXCEPTION 'CT241: % admitió al hijo (% %)',r.via,estado,mensaje; END IF;
   IF m.quien='padre' AND (mensaje LIKE 'CT241%'
      OR (r.via='estado_oferta' AND mensaje NOT IN ('CT225: contexto CT cambió','CT225: decisión no ligada'))
      OR (r.via='circuito' AND mensaje<>'autorización de consulta de circuito divergente')
      OR (r.via='contexto_oferta' AND estado<>'00000'))
   THEN RAISE EXCEPTION 'CT241: % del ordinario alterado (% %)',r.via,estado,mensaje; END IF;
  END LOOP;
 END LOOP;
END $vias$;

RESET SESSION AUTHORIZATION;

-- 4b. Cotejos con los que Bolsa crea llamamientos y ofertas de origen CT
-- (B110, B117, B137, B140), con el rol de Bolsa que los llama.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $bolsa$
DECLARE k record; c record; estado text; mensaje text; r record; ok boolean;
BEGIN
 SELECT * INTO STRICT k FROM pg_temp.ct241_caso;
 FOR c IN SELECT * FROM (VALUES ('padre',k.padre),('hijo',k.hijo)) AS v(quien,ref) LOOP
  FOR r IN SELECT * FROM (VALUES ('modalidad'),('cotejo_nuevo'),('cotejo_recuperacion')) AS v(via) LOOP
   BEGIN
    IF r.via='modalidad' THEN
     PERFORM * FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
      c.ref,k.categoria,k.organizacion,NULL);
    ELSE
     ok:=vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb_build_object(
      'expediente_ref',c.ref,'organizacion_ref',k.organizacion,'unidad_ref',k.unidad,
      'version_expediente',6),r.via='cotejo_nuevo');
    END IF;
    estado:='00000'; mensaje:='';
   EXCEPTION WHEN OTHERS THEN
    GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
   END;
   IF c.quien='hijo' AND (estado<>'23505' OR mensaje<>'CT241: expediente derivado sin selección nueva')
   THEN RAISE EXCEPTION 'CT241: % admitió al hijo (% %)',r.via,estado,mensaje; END IF;
   IF c.quien='padre' AND (estado<>'00000' OR (r.via='cotejo_recuperacion' AND ok IS NOT TRUE))
   THEN RAISE EXCEPTION 'CT241: % del ordinario alterado (% %)',r.via,estado,mensaje; END IF;
  END LOOP;
 END LOOP;
END $bolsa$;
RESET ROLE;

-- 5. Escrituras del hijo: el disparador central va antes que el resto de
-- comprobaciones; al ordinario lo para después otra restricción.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $escrituras$
DECLARE k record; c record; estado text; mensaje text;
BEGIN
 SELECT * INTO STRICT k FROM pg_temp.ct241_caso;
 FOR c IN SELECT * FROM (VALUES
   ('hijo','version7',k.hijo),('hijo','version6_ct234',k.hijo),('padre','version7',k.padre),
   ('hijo','version7_ct234',k.hijo),('hijo','version6_otro',k.hijo),
   ('hijo','publicacion',k.hijo),('padre','publicacion',k.padre),
   ('hijo','firma',k.hijo),('padre','firma',k.padre)) AS v(quien,via,ref) LOOP
  BEGIN
   IF c.via='version7' THEN
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,origen_version)
    VALUES(c.ref,7,'ct241_prueba');
   ELSIF c.via='version6_ct234' THEN
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,origen_version)
    VALUES(c.ref,6,'derivacion_nombramiento_ct234');
   ELSIF c.via='version7_ct234' THEN
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,origen_version)
    VALUES(c.ref,7,'derivacion_nombramiento_ct234');
   ELSIF c.via='version6_otro' THEN
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,origen_version)
    VALUES(c.ref,6,'ct241_prueba');
   ELSIF c.via='publicacion' THEN
    INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh(expediente_ref,version) VALUES(c.ref,6);
   ELSE
    INSERT INTO vec_contratacion_temporal.firma_documento_v1(expediente_ref) VALUES(c.ref);
   END IF;
   estado:='00000'; mensaje:='';
  EXCEPTION WHEN OTHERS THEN
   GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
  END;
  IF estado='00000' THEN RAISE EXCEPTION 'CT241: escritura incompleta aceptada (% %)',c.quien,c.via; END IF;
  IF (c.quien='hijo' AND c.via<>'version6_ct234')
     <> (estado='42501' AND mensaje='CT241: nombramiento derivado sin autoridad propia')
  THEN RAISE EXCEPTION 'CT241: escritura % % inesperada (% %)',c.quien,c.via,estado,mensaje; END IF;
 END LOOP;
END $escrituras$;
RESET ROLE;
SELECT 'ct241_guardas_nombramiento_derivado: 0 fallos' AS resultado;
ROLLBACK;
