\set ON_ERROR_STOP on
-- BIC6: entrega sólo las filas cifradas seleccionadas de un acta vigente.
-- El propietario de Bolsa la invoca desde su fachada nominal B84, después
-- del consumo V3. Ningún LOGIN recibe EXECUTE directo ni acceso a la tabla.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_importacion_convoca:migracion:000006',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_importacion_convoca_propietario'
    OR pg_catalog.to_regprocedure('vec_bolsa_importacion_convoca.consultar_estado_v1(text,text)') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_importacion_convoca.fila_staging') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(text,text,integer[])') IS NOT NULL
    OR pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 THEN RAISE EXCEPTION 'BIC6: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(
 p_huella text,p_categoria_ref text,p_numeros integer[])
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE
 estado jsonb;
 importacion text;
 filas jsonb;
 n integer;
 anterior integer:=1;
 total integer;
 bytes_total bigint;
BEGIN
 IF vec_bolsa_importacion_convoca.huella_valida(p_huella) IS NOT TRUE
    OR vec_bolsa_importacion_convoca.texto_opaco_valido(p_categoria_ref,512) IS NOT TRUE
    OR p_numeros IS NULL OR pg_catalog.array_ndims(p_numeros)<>1
    OR pg_catalog.cardinality(p_numeros) NOT BETWEEN 1 AND 5000
 THEN RAISE EXCEPTION 'BIC6: solicitud invalida' USING ERRCODE='22023'; END IF;
 FOREACH n IN ARRAY p_numeros LOOP
   IF n IS NULL OR n<=anterior THEN
     RAISE EXCEPTION 'BIC6: numeros no crecientes' USING ERRCODE='22023';
   END IF;
   anterior:=n;
 END LOOP;
 estado:=vec_bolsa_importacion_convoca.consultar_estado_v1(p_huella,p_categoria_ref);
 IF estado IS NULL OR estado->>'estado_staging'='expurgado' THEN
   RAISE EXCEPTION 'BIC6: acta no disponible' USING ERRCODE='42501';
 END IF;
 importacion:=estado#>>'{acta,importacion_ref}';
 IF importacion IS NULL THEN RAISE EXCEPTION 'BIC6: acta incoherente' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.count(*)::integer,
        coalesce(pg_catalog.sum(pg_catalog.octet_length(f.contenido_cifrado)
          +pg_catalog.octet_length(f.nonce)+pg_catalog.octet_length(f.derivacion_documento_hmac_sha256)
          +pg_catalog.octet_length(f.atestacion_fila_hmac_sha256)),0)
   INTO total,bytes_total
   FROM vec_bolsa_importacion_convoca.fila_staging f
  WHERE f.importacion_ref=importacion AND f.numero=ANY(p_numeros);
 IF total<>pg_catalog.cardinality(p_numeros) OR bytes_total>4194000 THEN
   RAISE EXCEPTION 'BIC6: filas ausentes o excesivas' USING ERRCODE='55000';
 END IF;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
          'numero',f.numero,
          'esquema_proteccion',f.esquema_proteccion,
          'clave_ref',f.clave_ref,
          'clave_derivacion_ref',f.clave_derivacion_ref,
          'clave_atestacion_ref',f.clave_atestacion_ref,
          'nonce_hex',pg_catalog.encode(f.nonce,'hex'),
          'contenido_cifrado_hex',pg_catalog.encode(f.contenido_cifrado,'hex'),
          'huella_contenido_cifrado_sha256',f.huella_contenido_cifrado_sha256,
          'derivacion_documento_hmac_sha256',pg_catalog.encode(f.derivacion_documento_hmac_sha256,'hex'),
          'atestacion_fila_hmac_sha256',pg_catalog.encode(f.atestacion_fila_hmac_sha256,'hex')) ORDER BY f.numero),'[]'::jsonb)
   INTO filas
   FROM vec_bolsa_importacion_convoca.fila_staging f
  WHERE f.importacion_ref=importacion AND f.numero=ANY(p_numeros);
 RETURN pg_catalog.jsonb_build_object('estado',estado,'filas',filas);
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(text,text,integer[]) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_importacion_convoca TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(text,text,integer[]) TO vec_bolsa_llamamientos_propietario;

DO $acl$
DECLARE f regprocedure:='vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(text,text,integer[])'::regprocedure;
BEGIN
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
        CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=f AND a.grantee=0)
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_bolsa_importacion_convoca_propietario'::regrole
 THEN RAISE EXCEPTION 'BIC6: ACL incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
