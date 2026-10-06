\set ON_ERROR_STOP on
-- CT179: corrige el fallo de CT166 en o404e_construir_lote_c1_v1.
-- CT166 añadió un EXISTS sobre jsonb_array_elements(...) con alias «x». La
-- función ya declara una variable «x jsonb», así que PL/pgSQL rechaza cada
-- llamada con «column reference "x" is ambiguous» y la confirmación de la vía
-- de cobertura falla siempre. Aquí sólo se renombra ese alias. La condición,
-- la huella de las órdenes, el lote, la propiedad, la configuración y los
-- permisos de la función quedan como estaban; CT166 no se toca ni se reaplica.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:o4_04:migraciones',0));
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000179',0));
DO $cambio$
DECLARE
 f oid:=pg_catalog.to_regprocedure(
   'vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text)');
 original text; fuente text; nuevo text; actual text;
 meta jsonb; deps jsonb;
 marca text:=$marca$         SELECT 1 FROM pg_catalog.jsonb_array_elements(p_carga->'consumos_c1') x
          WHERE (x#>'{periodo,politica_fin}' IS NOT NULL
              OR p_carga#>'{concesion,propuesta,periodo,politica_fin}' IS NOT NULL)
            AND x->'periodo' IS DISTINCT FROM$marca$;
 cambio text:=$nuevo$         SELECT 1 FROM pg_catalog.jsonb_array_elements(p_carga->'consumos_c1')
                AS ct179_consumo(valor)
          WHERE (ct179_consumo.valor#>'{periodo,politica_fin}' IS NOT NULL
              OR p_carga#>'{concesion,propuesta,periodo,politica_fin}' IS NOT NULL)
            AND ct179_consumo.valor->'periodo' IS DISTINCT FROM$nuevo$;
BEGIN
 IF f IS NULL OR current_user<>'vec_contratacion_temporal_propietario'
    OR pg_catalog.to_regprocedure(
         'vec_contratacion_temporal.ct166_periodo_valido_v1(jsonb)') IS NULL THEN
  RAISE EXCEPTION 'CT179: autoridad o CT166 ausentes' USING ERRCODE='55000';
 END IF;
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc'
   INTO STRICT original,fuente,meta
   FROM pg_catalog.pg_proc p WHERE p.oid=f;
 -- Preimagen exacta: el cuerpo que dejó CT166 en la principal.
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex')
      IS DISTINCT FROM '4c11cb3c77095397b8dcc2cfda1395c92296d3de23850a621d9b8f8f5a41de65'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex')
      IS DISTINCT FROM '99da4b44fa75483d1378d756fd80517aa37607b18db8168b7b8ead2a8cc4970d'
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
    AND p.proowner='vec_contratacion_temporal_propietario'::pg_catalog.regrole
    AND NOT p.prosecdef AND p.prokind='f' AND p.provolatile='i' AND p.proisstrict
    AND p.proconfig=ARRAY['search_path=pg_catalog'])
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))
      <>pg_catalog.length(marca)
 OR pg_catalog.strpos(original,'ct179_consumo')<>0 THEN
  RAISE EXCEPTION 'CT179: preimagen de función incompatible' USING ERRCODE='55000';
 END IF;
 SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
          ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_catalog.pg_depend d
  WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=f;

 nuevo:=pg_catalog.replace(original,marca,cambio);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
 OR pg_catalog.replace(actual,cambio,marca) IS DISTINCT FROM original
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual,'UTF8')),'hex')
      IS DISTINCT FROM '6d01c3b098e69ee434d20b4f9864b43fca3ad212fed0d85f3404c1a15b9852b2' THEN
  RAISE EXCEPTION 'CT179: función alterada fuera del cambio previsto' USING ERRCODE='55000';
 END IF;
 IF (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f)
      IS DISTINCT FROM meta
 OR (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
          ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_catalog.pg_depend d
      WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=f)
      IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT179: metadatos o dependencias alterados' USING ERRCODE='55000';
 END IF;
END
$cambio$;
COMMIT;
