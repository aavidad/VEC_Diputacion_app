\set ON_ERROR_STOP on
-- CT180: el detalle RRHH de un expediente cuyo análisis no tiene coste
-- previsto vuelve a abrirse.
-- Go omite «coste_previsto» cuando no hay coste (omitempty, AnalisisRRHH y
-- AnalisisOperativoRRHH). materializar_detalle_rrhh_v1 calculaba «hay coste»
-- con jsonb_typeof(...) = 'object', que con la clave ausente da NULL y no
-- false. canon_contenido_detalle_rrhh_v1 rechaza ese NULL y la consulta acaba
-- en «detalle RRHH no disponible» (404 en la API). El valor correcto es false:
-- sin coste, céntimos 0, moneda y fuente vacías, igual que valida el canon.
-- Sólo cambia esa asignación, y sólo para la forma que escribe Go (sin coste
-- y sin fuente). Un coste presente, o una fuente sin coste, se tratan igual
-- que antes; la propiedad, la configuración, los permisos y la historia no
-- cambian.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000180',0));
DO $cambio$
DECLARE
 f oid:=pg_catalog.to_regprocedure(
   'vec_contratacion_temporal.materializar_detalle_rrhh_v1('
   'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
   'vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)');
 original text; fuente text; nuevo text; actual text;
 meta jsonb; deps jsonb;
 marca text:=$marca$        v_coste_presente :=
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object';$marca$;
 cambio text:=$nuevo$        -- Sin coste previsto Go tampoco guarda la fuente del coste: sólo
        -- entonces «hay coste» es false. Con fuente y sin coste sigue NULL y
        -- el canon lo rechaza, como antes de CT180.
        v_coste_presente := COALESCE(
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object',
            CASE WHEN v_agregado #> '{analisis,fuente_coste_ref}' IS NULL
                 THEN false END
        );$nuevo$;
BEGIN
 IF f IS NULL OR current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'CT180: autoridad ausente' USING ERRCODE='55000';
 END IF;
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc'
   INTO STRICT original,fuente,meta
   FROM pg_catalog.pg_proc p WHERE p.oid=f;
 -- Preimagen exacta: el cuerpo vigente en la principal tras CT165.
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex')
      IS DISTINCT FROM '1c103bce808f121a86b972152e7c9ad6f83846e4006ff4c3dbd6f97fbed7df38'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex')
      IS DISTINCT FROM 'a356003a903dcfcba4ad28f55197423df9c25bedad2ceba7192ee46d9aa9c2a9'
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
    AND p.proowner='vec_contratacion_temporal_propietario'::pg_catalog.regrole
    AND p.prosecdef AND p.prokind='f' AND p.provolatile='s' AND NOT p.proisstrict
    AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC',
      'lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s'])
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))
      <>pg_catalog.length(marca) THEN
  RAISE EXCEPTION 'CT180: preimagen de función incompatible' USING ERRCODE='55000';
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
      IS DISTINCT FROM 'dbd41bd36d06c2b450b71d9a2c9a9d98e0cb26d40b82692f0e6552e6d3c1ef92' THEN
  RAISE EXCEPTION 'CT180: función alterada fuera del cambio previsto' USING ERRCODE='55000';
 END IF;
 IF (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f)
      IS DISTINCT FROM meta
 OR (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d)
          ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_catalog.pg_depend d
      WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=f)
      IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT180: metadatos o dependencias alterados' USING ERRCODE='55000';
 END IF;
END
$cambio$;
COMMIT;
