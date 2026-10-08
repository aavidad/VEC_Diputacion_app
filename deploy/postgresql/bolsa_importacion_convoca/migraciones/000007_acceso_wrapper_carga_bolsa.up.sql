\set ON_ERROR_STOP on
-- BIC7: el propietario de Bolsa invoca sólo las dos fachadas necesarias para
-- confirmar una carga B1 en su propia transacción. No recibe tablas ni roles.
BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_importacion_convoca:migracion:000007',0));
DO $pre$
DECLARE f1 oid:=pg_catalog.to_regprocedure('vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)');
        f2 oid:=pg_catalog.to_regprocedure('vec_bolsa_importacion_convoca.consultar_estado_v1(text,text)');
BEGIN
 IF current_user<>'vec_bolsa_importacion_convoca_propietario'
    OR f1 IS NULL OR f2 IS NULL
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid IN (f1,f2)
       AND (proowner<>'vec_bolsa_importacion_convoca_propietario'::pg_catalog.regrole OR NOT prosecdef))
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario'
       AND NOT rolcanlogin AND NOT rolbypassrls)
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f1,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f2,'EXECUTE')
 THEN RAISE EXCEPTION 'BIC7: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_bolsa_importacion_convoca TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb),
 vec_bolsa_importacion_convoca.consultar_estado_v1(text,text)
 TO vec_bolsa_llamamientos_propietario;

-- Original técnico cifrado de la importación. No es una copia en claro ni una
-- ficha documental. Su acta, metadatos AEAD y bytes cifrados son inmutables.
CREATE TABLE vec_bolsa_importacion_convoca.original_exportacion_cifrada (
 acta_ref text PRIMARY KEY REFERENCES vec_bolsa_importacion_convoca.lote(acta_ref),
 huella_fichero_sha256 text NOT NULL CHECK (huella_fichero_sha256 ~ '^[0-9a-f]{64}$'),
 fichero_custodiado_ref text NOT NULL UNIQUE,
 formato text NOT NULL CHECK (formato IN ('xls','xlsx')),
 bytes_originales integer NOT NULL CHECK (bytes_originales BETWEEN 1 AND 1048576),
 esquema_proteccion text NOT NULL CHECK (pg_catalog.octet_length(esquema_proteccion) BETWEEN 3 AND 64),
 clave_ref text NOT NULL CHECK (pg_catalog.octet_length(clave_ref) BETWEEN 3 AND 256),
 clave_version integer NOT NULL CHECK (clave_version BETWEEN 1 AND 2147483647),
 nonce bytea NOT NULL CHECK (pg_catalog.octet_length(nonce)=12),
 contenido_cifrado bytea NOT NULL CHECK (pg_catalog.octet_length(contenido_cifrado)=bytes_originales+16),
 huella_contenido_cifrado_sha256 text NOT NULL CHECK (
  huella_contenido_cifrado_sha256=pg_catalog.encode(pg_catalog.sha256(contenido_cifrado),'hex')),
 aad_huella_sha256 text NOT NULL CHECK (aad_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
REVOKE ALL ON vec_bolsa_importacion_convoca.original_exportacion_cifrada FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_importacion_convoca.original_exportacion_cifrada FROM PUBLIC;
ALTER TABLE vec_bolsa_importacion_convoca.original_exportacion_cifrada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.original_exportacion_cifrada FORCE ROW LEVEL SECURITY;
CREATE POLICY original_solo_propietario ON vec_bolsa_importacion_convoca.original_exportacion_cifrada
 FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);

CREATE FUNCTION vec_bolsa_importacion_convoca.negar_mutacion_original_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RAISE EXCEPTION 'BIC7: original cifrado inmutable' USING ERRCODE='55000';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_importacion_convoca.negar_mutacion_original_v1() FROM PUBLIC;
CREATE TRIGGER negar_cambio_original BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_bolsa_importacion_convoca.original_exportacion_cifrada
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_importacion_convoca.negar_mutacion_original_v1();

CREATE FUNCTION vec_bolsa_importacion_convoca.guardar_original_v1(p_acta jsonb,p_original jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE previo vec_bolsa_importacion_convoca.original_exportacion_cifrada%ROWTYPE;
        formato text; n integer; cifrado bytea; nonce_b bytea; huella text;
        acta text; archivo text; ref text; aad text;
BEGIN
 IF current_user<>'vec_bolsa_importacion_convoca_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.jsonb_typeof(p_acta) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_original) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.array_agg(k ORDER BY k) FROM pg_catalog.jsonb_object_keys(p_original) k)
  IS DISTINCT FROM ARRAY['bytes_originales','clave_ref','clave_version','contenido_cifrado_hex',
   'esquema_proteccion','formato','huella_contenido_cifrado_sha256','nonce_hex']::text[]
 OR pg_catalog.jsonb_typeof(p_original->'bytes_originales') IS DISTINCT FROM 'number'
 OR pg_catalog.jsonb_typeof(p_original->'clave_version') IS DISTINCT FROM 'number'
 THEN RAISE EXCEPTION 'BIC7: original cifrado inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  n:=(p_original->>'bytes_originales')::integer;
  cifrado:=pg_catalog.decode(p_original->>'contenido_cifrado_hex','hex');
  nonce_b:=pg_catalog.decode(p_original->>'nonce_hex','hex');
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'BIC7: original cifrado inválido' USING ERRCODE='22023'; END;
 acta:=p_acta->>'acta_ref'; archivo:=p_acta->>'huella_fichero_sha256';
 ref:=p_acta->>'fichero_custodiado_ref'; formato:=p_original->>'formato';
 huella:=p_original->>'huella_contenido_cifrado_sha256';
 IF acta IS NULL OR acta !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 OR archivo IS NULL OR archivo !~ '^[0-9a-f]{64}$'
 OR ref IS DISTINCT FROM 'original:convoca:'||pg_catalog.right(acta,64)
 OR formato NOT IN ('xls','xlsx') OR formato IS NULL
 OR pg_catalog.lower(p_acta->>'nombre_fichero') NOT LIKE '%.'||formato
 OR n NOT BETWEEN 1 AND 1048576
 OR pg_catalog.octet_length(cifrado)<>n+16 OR pg_catalog.octet_length(nonce_b)<>12
 OR huella IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(cifrado),'hex')
 OR pg_catalog.octet_length(p_original->>'clave_ref') NOT BETWEEN 3 AND 256
 OR pg_catalog.octet_length(p_original->>'esquema_proteccion') NOT BETWEEN 3 AND 64
 OR (p_original->>'clave_version')::integer NOT BETWEEN 1 AND 2147483647
 OR NOT EXISTS (SELECT 1 FROM vec_bolsa_importacion_convoca.lote l
  WHERE l.acta_ref=acta AND l.huella_fichero_sha256=archivo
   AND l.acta_canonica->>'fichero_custodiado_ref'=ref)
 THEN RAISE EXCEPTION 'BIC7: original cifrado incompatible' USING ERRCODE='22023'; END IF;
 aad:='bolsa.importacion.original.v1'||pg_catalog.chr(31)||acta||pg_catalog.chr(31)||archivo||
  pg_catalog.chr(31)||formato||pg_catalog.chr(31)||n::text;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_importacion_convoca:original:'||acta,0));
 SELECT * INTO previo FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada
  WHERE acta_ref=acta FOR SHARE;
 IF FOUND THEN
  IF previo.huella_fichero_sha256<>archivo OR previo.fichero_custodiado_ref<>ref
  OR previo.formato<>formato OR previo.bytes_originales<>n
  OR previo.aad_huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(aad,'UTF8')),'hex')
  THEN RAISE EXCEPTION 'BIC7: original cifrado en conflicto' USING ERRCODE='23505'; END IF;
  RETURN true;
 END IF;
 INSERT INTO vec_bolsa_importacion_convoca.original_exportacion_cifrada(
  acta_ref,huella_fichero_sha256,fichero_custodiado_ref,formato,bytes_originales,
  esquema_proteccion,clave_ref,clave_version,nonce,contenido_cifrado,
  huella_contenido_cifrado_sha256,aad_huella_sha256)
 VALUES(acta,archivo,ref,formato,n,p_original->>'esquema_proteccion',p_original->>'clave_ref',
  (p_original->>'clave_version')::integer,nonce_b,cifrado,huella,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(aad,'UTF8')),'hex'));
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_importacion_convoca.guardar_original_v1(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_importacion_convoca.guardar_original_v1(jsonb,jsonb)
 TO vec_bolsa_llamamientos_propietario;
DO $post$
BEGIN
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca.consultar_estado_v1(text,text)','EXECUTE')
 OR pg_catalog.has_table_privilege('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca.lote','SELECT')
 OR pg_catalog.has_table_privilege('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca.original_exportacion_cifrada','SELECT')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca.guardar_original_v1(jsonb,jsonb)','EXECUTE')
 OR pg_catalog.pg_has_role('vec_bolsa_llamamientos_propietario',
    'vec_bolsa_importacion_convoca_propietario','MEMBER')
 THEN RAISE EXCEPTION 'BIC7: ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
