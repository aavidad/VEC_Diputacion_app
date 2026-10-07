\set ON_ERROR_STOP on
-- B85: un recuento por conjunto para el mismo cuadro RRHH que lee B82.
-- Conserva el predicado de B17 y sólo devuelve bolsa_ref y un agregado.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000085',0));

DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(text)');
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()') IS NOT NULL
    OR f IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
 THEN RAISE EXCEPTION 'B85: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
     FROM pg_catalog.pg_proc p WHERE p.oid=f)
       IS DISTINCT FROM '514c8ab9a9081a172e48e0921876db27ecf6195571ea2c5c11747ae92b7838cb'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc p WHERE p.oid=f)
       IS DISTINCT FROM 'b8e414f53f0ab92b4771f3d18618970dae33d7f8d14ccac56229068e2b371d29'
 THEN RAISE EXCEPTION 'B85: predicado B17 divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()
RETURNS TABLE(bolsa_ref text,llamamientos_en_curso bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'B85: lectura de recuentos denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 WITH bolsas AS MATERIALIZED (
   SELECT DISTINCT k.bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() k
 ), completos AS MATERIALIZED (
   SELECT l.bolsa_ref
   FROM vec_bolsa_llamamientos.llamamiento_emitido l
   JOIN bolsas b ON b.bolsa_ref=l.bolsa_ref
   WHERE (SELECT pg_catalog.count(*)
      FROM pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
      JOIN vec_bolsa_llamamientos.contacto_participacion c
        ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
       AND c.canal='correo'
       AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
       AND c.recibo_ref='recibo:contacto:'||pg_catalog.encode(pg_catalog.sha256(
         pg_catalog.convert_to(l.bolsa_ref||pg_catalog.chr(31)||l.clave_idempotencia||
           pg_catalog.chr(31)||x.ref,'UTF8')),'hex'))
       =pg_catalog.jsonb_array_length(l.participaciones)
 )
 SELECT b.bolsa_ref,pg_catalog.count(c.bolsa_ref)::bigint
 FROM bolsas b LEFT JOIN completos c ON c.bolsa_ref=b.bolsa_ref
 GROUP BY b.bolsa_ref ORDER BY b.bolsa_ref;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() TO vec_bolsa_llamamientos_ejecutor;
DO $acl$
DECLARE f oid:='vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()'::pg_catalog.regprocedure;
BEGIN
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
          OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
       AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
       AND p.prosecdef AND p.provolatile='s'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 THEN RAISE EXCEPTION 'B85: ACL o definicion incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
