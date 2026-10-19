\set ON_ERROR_STOP on
-- BC14 / F1-61: publicación oficial de las bases de una convocatoria.
-- Las bases se publican fuera de VEC (BOP, tablón de edictos, BOE o BOJA) y
-- el plazo se cuenta desde esa publicación (LPAC art. 45; TREBEP arts. 55 y
-- 61). VEC no publica en esos medios: guarda la declaración del hecho con su
-- referencia verificable, como dato de solo adición con recibo y evento.
-- * publicacion_oficial_bases: una fila por declaración; una corrección es una
--   fila nueva que sustituye a la anterior de la misma versión. Nada se borra.
-- * declarar_publicacion_oficial_bases_v1: solo del propietario, sin
--   concesiones. El acto de RRHH desde la web espera a su consumidor en el
--   núcleo de autorización; mientras tanto ninguna cuenta de aplicación puede
--   declarar.
-- * comprobar_publicacion_inscripcion_v1 y proyectar_abierta_inscripcion_v1
--   (postimagen BC12) exigen una declaración vigente de la versión con fecha
--   anterior al primer día del plazo. Sin ella la versión no se abre.
-- Sin DOWN ni reaplicación. No rellena filas.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000014',0));
DO $pre$
DECLARE r record; f regprocedure; p record;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_setting('server_encoding')<>'UTF8'
 OR to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases') IS NOT NULL
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases_evento') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.rechazar_mutacion_inmutable()') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(text,bigint,text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_timezone_names WHERE name='Europe/Madrid')
 THEN RAISE EXCEPTION 'BC14: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
   ('comprobar','vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)','510d5fb7fa0daa905cf8565f2e53de8d00b2ed5623bf0d5a1a028082cbc71547','{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario=X/vec_bolsa_convocatorias_propietario}','v','u'),
   ('proyectar','vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamp with time zone)','d47623ea04fef1e6eed09aebb017771ddea8bd685613fb3e63a939455ee40ef8','{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario}','s','r')
  ) AS esperado(clave,firma,huella,acl,volatilidad,paralelismo) LOOP
  f:=to_regprocedure(r.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'BC14: funcion causal ausente' USING ERRCODE='55000'; END IF;
  SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>r.volatilidad OR p.proparallel::text<>r.paralelismo
  OR p.proacl::text IS DISTINCT FROM r.acl OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>r.huella
  THEN RAISE EXCEPTION 'BC14: definicion o metadatos BC12 divergentes' USING ERRCODE='55000'; END IF;
  PERFORM set_config('vec.bc14.oid_'||r.clave,f::oid::text,true);
 END LOOP;
END $pre$;

-- Medios fijados por la norma para publicar convocatorias de selección. El
-- boletín lleva número; el tablón de edictos no. La fecha es civil (no un
-- instante) y nunca futura; la dirección es opcional y solo https.
CREATE TABLE vec_bolsa_convocatorias.publicacion_oficial_bases (
 publicacion_ref text PRIMARY KEY,
 orden bigint NOT NULL UNIQUE,
 convocatoria_id text NOT NULL,
 secuencia bigint NOT NULL,
 huella_version_sha256 text NOT NULL,
 medio text NOT NULL,
 boletin_numero integer,
 anuncio_ref text NOT NULL,
 fecha_publicacion date NOT NULL,
 direccion text,
 sustituye_ref text,
 declarada_por text NOT NULL,
 motivo_ref text NOT NULL,
 clave_idempotencia text NOT NULL UNIQUE,
 solicitud_canonica text NOT NULL,
 huella_solicitud_sha256 text NOT NULL,
 huella_anterior_sha256 text NOT NULL,
 huella_registro_sha256 text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE (publicacion_ref,convocatoria_id,secuencia),
 FOREIGN KEY (convocatoria_id,secuencia)
  REFERENCES vec_bolsa_convocatorias.version_convocatoria(convocatoria_id,secuencia),
 -- Una corrección sustituye siempre a una declaración de la misma versión.
 FOREIGN KEY (sustituye_ref,convocatoria_id,secuencia)
  REFERENCES vec_bolsa_convocatorias.publicacion_oficial_bases(publicacion_ref,convocatoria_id,secuencia),
 CONSTRAINT publicacion_oficial_referencias CHECK (
  publicacion_ref ~ '^publicacion_bases:[0-9a-f]{64}$'
  AND recibo_ref='recibo:'||publicacion_ref
  AND (sustituye_ref IS NULL OR (sustituye_ref ~ '^publicacion_bases:[0-9a-f]{64}$'
   AND sustituye_ref<>publicacion_ref))
  AND orden>0 AND secuencia>0
  AND vec_bolsa_convocatorias.texto_opaco_valido(convocatoria_id,480)),
 CONSTRAINT publicacion_oficial_medio CHECK (
  medio IN ('bop_granada','tablon_edictos_electronico','boe','boja')
  AND (CASE WHEN medio='tablon_edictos_electronico' THEN boletin_numero IS NULL
       ELSE boletin_numero BETWEEN 1 AND 99999 END)
  AND anuncio_ref ~ '^[A-Za-z0-9][A-Za-z0-9./_-]{0,79}$'
  AND fecha_publicacion>=DATE '2000-01-01'
  AND (direccion IS NULL OR (octet_length(direccion) BETWEEN 12 AND 512
   AND direccion ~ '^https://[A-Za-z0-9.-]{3,253}(/[!-~]*)?$'))),
 CONSTRAINT publicacion_oficial_autoria CHECK (
  declarada_por ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{2,159}$'
  AND motivo_ref ~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$'
  AND clave_idempotencia ~ '^[A-Za-z0-9:_-]{8,128}$'),
 CONSTRAINT publicacion_oficial_huellas CHECK (
  huella_version_sha256 ~ '^[0-9a-f]{64}$'
  AND huella_anterior_sha256 ~ '^[0-9a-f]{64}$'
  AND octet_length(solicitud_canonica) BETWEEN 2 AND 8192
  AND huella_solicitud_sha256=encode(sha256(convert_to(solicitud_canonica,'UTF8')),'hex')
  AND huella_registro_sha256=encode(sha256(convert_to(
   huella_anterior_sha256||'|'||huella_solicitud_sha256||'|'||orden::text,'UTF8')),'hex'))
);
-- Una sola declaración inicial por versión y una sola sucesora por declaración:
-- la vigente es siempre el extremo único de la cadena.
CREATE UNIQUE INDEX publicacion_oficial_inicial_uq
 ON vec_bolsa_convocatorias.publicacion_oficial_bases(convocatoria_id,secuencia)
 WHERE sustituye_ref IS NULL;
CREATE UNIQUE INDEX publicacion_oficial_sucesora_uq
 ON vec_bolsa_convocatorias.publicacion_oficial_bases(sustituye_ref)
 WHERE sustituye_ref IS NOT NULL;

CREATE TABLE vec_bolsa_convocatorias.publicacion_oficial_bases_evento (
 evento_ref text PRIMARY KEY,
 publicacion_ref text NOT NULL UNIQUE
  REFERENCES vec_bolsa_convocatorias.publicacion_oficial_bases(publicacion_ref),
 tipo text NOT NULL,
 registrado_en timestamptz(6) NOT NULL,
 CHECK (evento_ref='evento:'||publicacion_ref
  AND tipo IN ('bolsa.convocatoria.publicacion_oficial.declarada',
   'bolsa.convocatoria.publicacion_oficial.corregida'))
);

DO $protecciones$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['publicacion_oficial_bases','publicacion_oficial_bases_evento'] LOOP
  EXECUTE format('CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_convocatorias.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',tabla,tabla);
  EXECUTE format('CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',tabla,tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_convocatorias.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_convocatorias.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY acceso_propietario_exacto ON vec_bolsa_convocatorias.%I FOR ALL TO vec_bolsa_convocatorias_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
   tabla,'vec_bolsa_convocatorias_propietario','vec_bolsa_convocatorias_propietario');
  EXECUTE format('REVOKE ALL ON vec_bolsa_convocatorias.%I FROM PUBLIC',tabla);
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_convocatorias.%I FROM PUBLIC',tabla);
 END LOOP;
END $protecciones$;

-- Fecha de la publicación oficial vigente de una versión exacta, o NULL. Solo
-- la usan los lectores del propietario; sin concesiones.
CREATE FUNCTION vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(
 p_id text,p_secuencia bigint,p_huella_sha256 text)
RETURNS date LANGUAGE plpgsql STABLE PARALLEL RESTRICTED
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_fecha date;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario' THEN
  RAISE EXCEPTION 'BC14: lectura no autorizada' USING ERRCODE='42501'; END IF;
 SELECT p.fecha_publicacion INTO v_fecha
 FROM vec_bolsa_convocatorias.publicacion_oficial_bases p
 WHERE p.convocatoria_id=p_id AND p.secuencia=p_secuencia
  AND p.huella_version_sha256=p_huella_sha256
  AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases s
   WHERE s.sustituye_ref=p.publicacion_ref);
 RETURN v_fecha;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(text,bigint,text) FROM PUBLIC;

-- Vía solo del propietario (despliegue o copia local con datos ficticios).
-- Idempotente por clave: la misma solicitud devuelve su recibo; otra con la
-- misma clave da 23505 sin escribir.
CREATE FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(p_declaracion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE
SET search_path=pg_catalog,pg_temp SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE d jsonb:=p_declaracion; v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 fila vec_bolsa_convocatorias.publicacion_oficial_bases%ROWTYPE;
 v_secuencia bigint; v_fecha date; v_hoy date; v_boletin integer; v_canon text; v_huella text;
 v_anterior text; v_orden bigint; v_ahora timestamptz(6); reutilizada boolean:=true;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario' THEN
  RAISE EXCEPTION 'BC14: declaración no autorizada' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(d) k) IS DISTINCT FROM
    ARRAY['anuncio_ref','boletin_numero','clave_idempotencia','convocatoria_id','declarada_por',
     'direccion','fecha_publicacion','huella_version_sha256','medio','motivo_ref','secuencia','sustituye_ref']
 OR jsonb_typeof(d->'convocatoria_id') IS DISTINCT FROM 'string'
 OR jsonb_typeof(d->'secuencia') IS DISTINCT FROM 'number'
 OR coalesce(d->>'secuencia','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(d->>'huella_version_sha256','') !~ '^[0-9a-f]{64}$'
 OR coalesce(d->>'medio','') NOT IN ('bop_granada','tablon_edictos_electronico','boe','boja')
 OR coalesce(jsonb_typeof(d->'boletin_numero'),'') NOT IN ('number','null')
 OR (jsonb_typeof(d->'boletin_numero')='number' AND d->>'boletin_numero' !~ '^[1-9][0-9]{0,4}$')
 OR jsonb_typeof(d->'anuncio_ref') IS DISTINCT FROM 'string'
 OR coalesce(d->>'fecha_publicacion','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR coalesce(jsonb_typeof(d->'direccion'),'') NOT IN ('string','null')
 OR coalesce(jsonb_typeof(d->'sustituye_ref'),'') NOT IN ('string','null')
 OR coalesce(d->>'anuncio_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9./_-]{0,79}$'
 OR (jsonb_typeof(d->'direccion')='string' AND (octet_length(d->>'direccion') NOT BETWEEN 12 AND 512
  OR d->>'direccion' !~ '^https://[A-Za-z0-9.-]{3,253}(/[!-~]*)?$'))
 OR (jsonb_typeof(d->'sustituye_ref')='string' AND d->>'sustituye_ref' !~ '^publicacion_bases:[0-9a-f]{64}$')
 OR jsonb_typeof(d->'declarada_por') IS DISTINCT FROM 'string'
 OR d->>'declarada_por' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{2,159}$'
 OR jsonb_typeof(d->'motivo_ref') IS DISTINCT FROM 'string'
 OR d->>'motivo_ref' !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$'
 OR jsonb_typeof(d->'clave_idempotencia') IS DISTINCT FROM 'string'
 OR d->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]{8,128}$'
 OR NOT vec_bolsa_convocatorias.texto_opaco_valido(d->>'convocatoria_id',480)
 THEN RAISE EXCEPTION 'BC14: declaración inválida' USING ERRCODE='22023'; END IF;
 v_secuencia:=(d->>'secuencia')::bigint;
 v_boletin:=CASE WHEN jsonb_typeof(d->'boletin_numero')='number' THEN (d->>'boletin_numero')::integer END;
 BEGIN v_fecha:=(d->>'fecha_publicacion')::date;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC14: fecha de publicación inválida' USING ERRCODE='22023'; END;
 v_hoy:=(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date;
 IF to_char(v_fecha,'YYYY-MM-DD')<>d->>'fecha_publicacion' OR v_fecha>v_hoy
 OR v_fecha<DATE '2000-01-01'
 OR (d->>'medio'='tablon_edictos_electronico')<>(v_boletin IS NULL)
 THEN RAISE EXCEPTION 'BC14: fecha o boletín incoherentes' USING ERRCODE='22023'; END IF;
 v_canon:=jsonb_build_object('convocatoria_id',d->>'convocatoria_id','secuencia',v_secuencia,
  'huella_version_sha256',d->>'huella_version_sha256','medio',d->>'medio',
  'boletin_numero',v_boletin,'anuncio_ref',d->>'anuncio_ref',
  'fecha_publicacion',to_char(v_fecha,'YYYY-MM-DD'),'direccion',d->>'direccion',
  'sustituye_ref',d->>'sustituye_ref','declarada_por',d->>'declarada_por',
  'motivo_ref',d->>'motivo_ref','clave_idempotencia',d->>'clave_idempotencia')::text;
 v_huella:=encode(sha256(convert_to(v_canon,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:publicacion_oficial',0));
 SELECT * INTO fila FROM vec_bolsa_convocatorias.publicacion_oficial_bases
  WHERE clave_idempotencia=d->>'clave_idempotencia';
 IF FOUND THEN
  IF fila.huella_solicitud_sha256<>v_huella THEN
   RAISE EXCEPTION 'BC14: clave reutilizada con otra declaración' USING ERRCODE='23505'; END IF;
 ELSE
  reutilizada:=false;
  BEGIN
   SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
    WHERE convocatoria_id=d->>'convocatoria_id' AND secuencia=v_secuencia FOR SHARE;
  EXCEPTION WHEN no_data_found THEN
   RAISE EXCEPTION 'BC14: versión no vigente' USING ERRCODE='B9601'; END;
  IF v.estado<>'publicada' OR v.huella_version_sha256<>d->>'huella_version_sha256'
  OR encode(sha256(v.version_canonica),'hex')<>v.huella_version_sha256
  OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
    WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
     AND posterior.estado IN('publicada','sustituida','retirada'))
  THEN RAISE EXCEPTION 'BC14: versión no vigente' USING ERRCODE='B9601'; END IF;
  IF d->>'sustituye_ref' IS NULL THEN
   IF EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases p
     WHERE p.convocatoria_id=v.convocatoria_id AND p.secuencia=v.secuencia) THEN
    RAISE EXCEPTION 'BC14: la versión ya tiene publicación declarada' USING ERRCODE='23505'; END IF;
  ELSIF NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases p
     WHERE p.publicacion_ref=d->>'sustituye_ref' AND p.convocatoria_id=v.convocatoria_id
      AND p.secuencia=v.secuencia
      AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases s
       WHERE s.sustituye_ref=p.publicacion_ref)) THEN
   RAISE EXCEPTION 'BC14: solo se corrige la declaración vigente' USING ERRCODE='23505';
  END IF;
  SELECT p.orden,p.huella_registro_sha256 INTO v_orden,v_anterior
  FROM vec_bolsa_convocatorias.publicacion_oficial_bases p ORDER BY p.orden DESC LIMIT 1;
  v_orden:=coalesce(v_orden,0)+1;
  v_anterior:=coalesce(v_anterior,repeat('0',64));
  v_ahora:=date_trunc('microseconds',clock_timestamp());
  INSERT INTO vec_bolsa_convocatorias.publicacion_oficial_bases(
   publicacion_ref,orden,convocatoria_id,secuencia,huella_version_sha256,medio,
   boletin_numero,anuncio_ref,fecha_publicacion,direccion,sustituye_ref,declarada_por,
   motivo_ref,clave_idempotencia,solicitud_canonica,huella_solicitud_sha256,
   huella_anterior_sha256,huella_registro_sha256,recibo_ref,registrada_en)
  VALUES('publicacion_bases:'||v_huella,v_orden,v.convocatoria_id,v.secuencia,
   v.huella_version_sha256,d->>'medio',v_boletin,d->>'anuncio_ref',v_fecha,d->>'direccion',
   d->>'sustituye_ref',d->>'declarada_por',d->>'motivo_ref',d->>'clave_idempotencia',
   v_canon,v_huella,v_anterior,
   encode(sha256(convert_to(v_anterior||'|'||v_huella||'|'||v_orden::text,'UTF8')),'hex'),
   'recibo:publicacion_bases:'||v_huella,v_ahora)
  RETURNING * INTO fila;
  INSERT INTO vec_bolsa_convocatorias.publicacion_oficial_bases_evento(
   evento_ref,publicacion_ref,tipo,registrado_en)
  VALUES('evento:'||fila.publicacion_ref,fila.publicacion_ref,
   CASE WHEN fila.sustituye_ref IS NULL THEN 'bolsa.convocatoria.publicacion_oficial.declarada'
   ELSE 'bolsa.convocatoria.publicacion_oficial.corregida' END,v_ahora);
 END IF;
 RETURN jsonb_build_object('publicacion_ref',fila.publicacion_ref,'recibo_ref',fila.recibo_ref,
  'evento_ref','evento:'||fila.publicacion_ref,'convocatoria_id',fila.convocatoria_id,
  'secuencia',fila.secuencia,'medio',fila.medio,'boletin_numero',fila.boletin_numero,
  'anuncio_ref',fila.anuncio_ref,'fecha_publicacion',to_char(fila.fecha_publicacion,'YYYY-MM-DD'),
  'direccion',fila.direccion,'sustituye_ref',fila.sustituye_ref,
  'huella_registro_sha256',fila.huella_registro_sha256,
  'registrada_en',fila.registrada_en,'reutilizada',reutilizada);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb) FROM PUBLIC;

-- Definición final explícita de los dos lectores BC12 con la exigencia BC14.

CREATE OR REPLACE FUNCTION vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(p_convocatoria_ref text, p_categoria_ref text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE
 v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE;
 c jsonb; plazo jsonb; candidato jsonb; catalogo jsonb; politica jsonb;
 formulario jsonb; bases jsonb; v_ambito jsonb;
 requisitos jsonb; pendientes jsonb; categorias jsonb; instante timestamptz(6);
 abre timestamptz; cierra timestamptz; v_secuencia bigint;
 partes text[]; v_id_sha256 text; publicada timestamptz; activos integer:=0;
 v_publicacion_oficial date;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_convocatoria_ref IS NULL OR octet_length(p_convocatoria_ref) NOT BETWEEN 9 AND 200
 OR p_categoria_ref IS NULL OR p_categoria_ref !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 THEN RAISE EXCEPTION 'BC9: selector de publicación inválido' USING ERRCODE='22023'; END IF;
 partes:=regexp_match(p_convocatoria_ref,'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;
 v_id_sha256:=partes[1];
 v_secuencia:=partes[2]::bigint;
 IF v_secuencia NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'BC9: referencia no canónica' USING ERRCODE='22023'; END IF;

 -- Bloqueo de fila y predicado SERIALIZABLE: la versión debe seguir siendo la
 -- publicada más reciente mientras se consume la autorización y se inserta.
 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(convocatoria_id) COLLATE "C"=v_id_sha256 COLLATE "C"
    AND secuencia=v_secuencia FOR SHARE;
 EXCEPTION WHEN no_data_found THEN
  RAISE EXCEPTION 'BC9: publicación no vigente' USING ERRCODE='B9601';
 WHEN too_many_rows THEN
  RAISE EXCEPTION 'BC9: referencia de publicación ambigua' USING ERRCODE='55000'; END;
 IF v.estado<>'publicada'
 OR v.referencia IS DISTINCT FROM v.convocatoria_id||'#'||v_secuencia::text
 OR encode(sha256(v.version_canonica),'hex') IS DISTINCT FROM v.huella_version_sha256
 OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
   WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v_secuencia
    AND posterior.estado IN('publicada','sustituida','retirada'))
 THEN RAISE EXCEPTION 'BC9: publicación no vigente' USING ERRCODE='B9601'; END IF;
 BEGIN c:=convert_from(v.version_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: versión gobernada ilegible' USING ERRCODE='55000';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR c->>'id' IS DISTINCT FROM v.convocatoria_id
 OR c->>'secuencia' IS DISTINCT FROM v_secuencia::text
 OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(c#>'{contenido,plazos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,requisitos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,catalogo_categorias}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,catalogos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,flujo_solicitud}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
 OR c#>>'{contenido,identificador_publico}' IS NULL
 OR c#>>'{contenido,titulo}' IS NULL OR c#>>'{contenido,resumen}' IS NULL
 OR c#>>'{contenido,tipo}' IS NULL
 OR jsonb_typeof(c->'aprobacion_publicacion') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c->'comprobacion_dependencias') IS DISTINCT FROM 'object'
 OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM v.referencia
 OR c->>'publicada_en' IS NULL
 THEN RAISE EXCEPTION 'BC9: material publicado incompleto' USING ERRCODE='55000'; END IF;
 -- BC12: B96 solo presenta una categoria. Una version conjunta exige F1-41.
 -- La ausencia historica de la clave conserva la modalidad separada.
 IF c#>'{contenido,modalidad_inscripcion}' IS NOT NULL THEN
  IF jsonb_typeof(c#>'{contenido,modalidad_inscripcion}') IS DISTINCT FROM 'string'
   OR c#>>'{contenido,modalidad_inscripcion}' NOT IN('separada','conjunta')
  THEN RAISE EXCEPTION 'BC12: modalidad de inscripcion invalida' USING ERRCODE='55000'; END IF;
  IF c#>>'{contenido,modalidad_inscripcion}'='conjunta'
  THEN RAISE EXCEPTION 'BC12: inscripcion conjunta pendiente' USING ERRCODE='B9608'; END IF;
 END IF;
 IF jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 THEN RAISE EXCEPTION 'BC9: categorías publicadas incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT ((c#>'{contenido,categorias}') @> jsonb_build_array(p_categoria_ref))
 THEN RAISE EXCEPTION 'BC9: categoría ajena a la publicación' USING ERRCODE='B9605'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END IF;

 instante:=date_trunc('microseconds',clock_timestamp());
 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
   WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción inválido' USING ERRCODE='55000';
  END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
     OR candidato->>'referencia' IS NULL THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción incompleto' USING ERRCODE='55000';
  END IF;
  IF abre<=instante AND instante<cierra THEN
   activos:=activos+1; plazo:=candidato;
  END IF;
 END LOOP;
 IF activos<>1 THEN RAISE EXCEPTION 'BC9: plazo de inscripción cerrado o ambiguo' USING ERRCODE='B9602'; END IF;
 abre:=(plazo->>'abre_en')::timestamptz;
 cierra:=(plazo->>'cierra_en')::timestamptz;
 -- BC14: solo se abre con la publicación oficial de las bases declarada para
 -- esta versión exacta y con fecha anterior al primer día del plazo.
 v_publicacion_oficial:=vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(
  v.convocatoria_id,v_secuencia,v.huella_version_sha256);
 IF v_publicacion_oficial IS NULL
 OR v_publicacion_oficial>=(abre AT TIME ZONE 'Europe/Madrid')::date
 THEN RAISE EXCEPTION 'BC14: bases sin publicación oficial declarada' USING ERRCODE='B9601'; END IF;

 catalogo:=c#>'{contenido,catalogo_categorias}';
 politica:=c#>'{configuracion,catalogos}';
 formulario:=c#>'{configuracion,flujo_solicitud}';
 requisitos:=c#>'{contenido,requisitos}';
 SELECT valor INTO bases FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
  WHERE valor->>'rol'='bases';
 IF catalogo->>'catalogo_id' IS NULL OR coalesce(catalogo->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(catalogo->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 OR politica->>'id' IS NULL OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(politica->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR formulario->>'id' IS NULL OR coalesce(formulario->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(formulario->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
    WHERE valor->>'rol'='bases')<>1
 OR bases->>'publicacion_ref' IS NULL
 OR jsonb_array_length(requisitos)>256
 THEN RAISE EXCEPTION 'BC9: referencias de bases incompletas' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
    'referencia',r.valor->>'referencia','codigo',r.valor->>'referencia',
    'descripcion',r.valor->>'descripcion',
    'obligatorio',(r.valor->>'obligatorio')::boolean,
    'estado','pendiente','motivo_codigo','requisito.pendiente')
    ORDER BY (r.valor->>'orden')::integer,r.valor->>'referencia'),'[]'::jsonb)
 INTO pendientes FROM jsonb_array_elements(requisitos) AS r(valor);
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(requisitos) AS r(valor)
   WHERE coalesce(r.valor->>'referencia','')=''
      OR jsonb_typeof(r.valor->'obligatorio') IS DISTINCT FROM 'boolean'
      OR coalesce(r.valor->>'orden','') !~ '^[1-9][0-9]{0,5}$')
 THEN RAISE EXCEPTION 'BC9: requisitos incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',categoria.valor)
   ORDER BY categoria.valor),'[]'::jsonb)
 INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS categoria(valor);
 v_ambito:=vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
  politica->>'id',(politica->>'version')::integer,
  politica->>'huella_contenido_sha256',p_convocatoria_ref);

 RETURN jsonb_build_object(
  'convocatoria_ref',p_convocatoria_ref,
  'convocatoria_id',v.convocatoria_id,'secuencia',v_secuencia,
  'categoria_ref',p_categoria_ref,'categorias',categorias,
  'numero_categorias',jsonb_array_length(categorias),
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'version_sha256',v.huella_version_sha256,
  'bases_ref',bases->>'publicacion_ref',
  'catalogo_ref',catalogo->>'catalogo_id',
  'catalogo_version',(catalogo->>'catalogo_version')::integer,
  'catalogo_sha256',catalogo->>'catalogo_huella_sha256',
  'politica_catalogo_ref',politica->>'id',
  'politica_catalogo_version',(politica->>'version')::integer,
  'politica_catalogo_sha256',politica->>'huella_contenido_sha256',
  'formulario_ref',formulario->>'id','formulario_version',(formulario->>'version')::integer,
  'formulario_sha256',formulario->>'huella_contenido_sha256',
  'plazo_ref',plazo->>'referencia','plazo_abre_en',abre,'plazo_cierra_en',cierra,
  'requisitos',pendientes,
  'requisitos_sha256',encode(sha256(convert_to(requisitos::text,'UTF8')),'hex'),
  'unidad_ref',v_ambito->>'unidad_ref','ambito_ref',v_ambito->>'ambito_ref',
  'fuente_ref',v_ambito->>'fuente_ref',
  'fuente_version',v_ambito->'fuente_version',
  'fuente_sha256',v_ambito->>'fuente_sha256',
  'comprobada_en',instante);
END $function$
;

CREATE OR REPLACE FUNCTION vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(p_id text, p_secuencia bigint, p_canonica bytea, p_huella_sha256 text, p_instante timestamp with time zone)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE PARALLEL RESTRICTED SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE c jsonb; catalogo jsonb; politica jsonb; formulario jsonb;
 bases jsonb; requisitos jsonb; pendientes jsonb; categorias jsonb;
 plazo jsonb; candidato jsonb; abre timestamptz; cierra timestamptz;
 activos integer:=0; referencia text; publicada timestamptz; v_publicacion_oficial date;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR p_id IS NULL OR p_id !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]*$'
 OR octet_length(p_id) NOT BETWEEN 1 AND 480
 OR p_secuencia NOT BETWEEN 1 AND 9007199254740991
 OR p_canonica IS NULL OR p_huella_sha256 IS NULL
 OR encode(sha256(p_canonica),'hex') IS DISTINCT FROM p_huella_sha256
 OR p_instante IS NULL
 THEN RAISE EXCEPTION 'BC9: versión de lectura inválida' USING ERRCODE='55000'; END IF;
 referencia:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(p_id)||'_v'||p_secuencia::text;
 BEGIN c:=convert_from(p_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: versión gobernada ilegible' USING ERRCODE='55000'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
 OR c->>'id' IS DISTINCT FROM p_id OR c->>'secuencia' IS DISTINCT FROM p_secuencia::text
 OR c->>'estado_gobierno' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(c#>'{contenido,categorias}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,categorias}') NOT BETWEEN 1 AND 128
 OR jsonb_typeof(c#>'{contenido,plazos}') IS DISTINCT FROM 'array'
 OR jsonb_typeof(c#>'{contenido,requisitos}') IS DISTINCT FROM 'array'
 OR jsonb_array_length(c#>'{contenido,requisitos}')>256
 OR jsonb_typeof(c#>'{contenido,catalogo_categorias}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,catalogos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,flujo_solicitud}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c#>'{configuracion,documentos}') IS DISTINCT FROM 'array'
 OR c#>>'{contenido,identificador_publico}' IS NULL
 OR c#>>'{contenido,titulo}' IS NULL OR c#>>'{contenido,resumen}' IS NULL
 OR c#>>'{contenido,tipo}' IS NULL
 OR c->>'publicada_en' IS NULL
 OR jsonb_typeof(c->'aprobacion_publicacion') IS DISTINCT FROM 'object'
 OR jsonb_typeof(c->'comprobacion_dependencias') IS DISTINCT FROM 'object'
 OR c#>>'{aprobacion_publicacion,convocatoria_ref}' IS DISTINCT FROM p_id||'#'||p_secuencia::text
 OR c#>>'{comprobacion_dependencias,convocatoria_ref}' IS DISTINCT FROM p_id||'#'||p_secuencia::text
 THEN RAISE EXCEPTION 'BC9: material publicado incompleto' USING ERRCODE='55000'; END IF;
 BEGIN publicada:=(c->>'publicada_en')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END;
 IF publicada IS NULL OR NOT isfinite(publicada)
 THEN RAISE EXCEPTION 'BC9: fecha de publicación inválida' USING ERRCODE='55000'; END IF;
 -- BC12: una modalidad invalida deja fuera solo esa version; la lista sigue.
 IF c#>'{contenido,modalidad_inscripcion}' IS NOT NULL
  AND (jsonb_typeof(c#>'{contenido,modalidad_inscripcion}') IS DISTINCT FROM 'string'
   OR c#>>'{contenido,modalidad_inscripcion}' NOT IN('separada','conjunta'))
 THEN RETURN NULL; END IF;

 FOR candidato IN SELECT valor FROM jsonb_array_elements(c#>'{contenido,plazos}') AS p(valor)
  WHERE valor->>'tipo'='inscripcion' LOOP
  BEGIN
   abre:=(candidato->>'abre_en')::timestamptz;
   cierra:=(candidato->>'cierra_en')::timestamptz;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'BC9: plazo de inscripción inválido' USING ERRCODE='55000'; END;
  IF abre IS NULL OR cierra IS NULL OR NOT isfinite(abre) OR NOT isfinite(cierra)
   OR abre>=cierra OR candidato->>'referencia' IS NULL
  THEN RAISE EXCEPTION 'BC9: plazo de inscripción incompleto' USING ERRCODE='55000'; END IF;
  IF abre<=p_instante AND p_instante<cierra THEN
   activos:=activos+1; plazo:=candidato;
  END IF;
 END LOOP;
 IF activos=0 THEN RETURN NULL; END IF;
 IF activos<>1 THEN RAISE EXCEPTION 'BC9: plazos abiertos ambiguos' USING ERRCODE='55000'; END IF;
 -- BC14: sin publicación oficial declarada antes del plazo no se proyecta.
 v_publicacion_oficial:=vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(
  p_id,p_secuencia,p_huella_sha256);
 IF v_publicacion_oficial IS NULL
 OR v_publicacion_oficial>=((plazo->>'abre_en')::timestamptz AT TIME ZONE 'Europe/Madrid')::date
 THEN RETURN NULL; END IF;

 catalogo:=c#>'{contenido,catalogo_categorias}';
 politica:=c#>'{configuracion,catalogos}';
 formulario:=c#>'{configuracion,flujo_solicitud}';
 requisitos:=c#>'{contenido,requisitos}';
 SELECT valor INTO bases FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
  WHERE valor->>'rol'='bases';
 IF coalesce(catalogo->>'catalogo_id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(catalogo->>'catalogo_version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(catalogo->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 OR coalesce(politica->>'id','') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(politica->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR formulario->>'id' IS NULL
 OR coalesce(formulario->>'version','') !~ '^[1-9][0-9]{0,8}$'
 OR coalesce(formulario->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
 OR NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(c#>'{configuracion,documentos}') AS d(valor)
    WHERE valor->>'rol'='bases')<>1
 OR bases->>'publicacion_ref' IS NULL
 OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor)
   WHERE x.valor !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR (SELECT count(DISTINCT x.valor) FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor))
    <>jsonb_array_length(c#>'{contenido,categorias}')
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(requisitos) AS r(valor)
   WHERE coalesce(r.valor->>'referencia','')=''
    OR jsonb_typeof(r.valor->'obligatorio') IS DISTINCT FROM 'boolean'
    OR coalesce(r.valor->>'orden','') !~ '^[1-9][0-9]{0,5}$')
 THEN RAISE EXCEPTION 'BC9: referencias publicadas incompletas' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',x.valor)
  ORDER BY x.valor COLLATE "C"),'[]'::jsonb)
 INTO categorias FROM jsonb_array_elements_text(c#>'{contenido,categorias}') AS x(valor);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'referencia',r.valor->>'referencia','codigo',r.valor->>'referencia',
  'descripcion',r.valor->>'descripcion',
  'obligatorio',(r.valor->>'obligatorio')::boolean,
  'estado','pendiente','motivo_codigo','requisito.pendiente')
  ORDER BY (r.valor->>'orden')::integer,r.valor->>'referencia'),'[]'::jsonb)
 INTO pendientes FROM jsonb_array_elements(requisitos) AS r(valor);
 abre:=(plazo->>'abre_en')::timestamptz;
 cierra:=(plazo->>'cierra_en')::timestamptz;
 RETURN jsonb_build_object(
  'convocatoria_ref',referencia,'convocatoria_id',p_id,'secuencia',p_secuencia,
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'titulo',c#>>'{contenido,titulo}','resumen',c#>>'{contenido,resumen}',
  'tipo',c#>>'{contenido,tipo}',
  'modalidad_inscripcion',coalesce(c#>>'{contenido,modalidad_inscripcion}','separada'),
  'numero_categorias',jsonb_array_length(c#>'{contenido,categorias}'),
  'categoria_ref_comprobacion',c#>>'{contenido,categorias,0}',
  'categorias_refs_comprobacion',c#>'{contenido,categorias}',
  'categorias',categorias,'numero_requisitos',jsonb_array_length(requisitos),
  'version_sha256',p_huella_sha256,'bases_ref',bases->>'publicacion_ref',
  'catalogo_ref',catalogo->>'catalogo_id',
  'catalogo_version',(catalogo->>'catalogo_version')::integer,
  'catalogo_sha256',catalogo->>'catalogo_huella_sha256',
  'politica_catalogo_ref',politica->>'id',
  'politica_catalogo_version',(politica->>'version')::integer,
  'politica_catalogo_sha256',politica->>'huella_contenido_sha256',
  'formulario_ref',formulario->>'id',
  'formulario_version',(formulario->>'version')::integer,
  'formulario_sha256',formulario->>'huella_contenido_sha256',
  'plazo_ref',plazo->>'referencia','plazo_abre_en',abre,'plazo_cierra_en',cierra,
  'requisitos',pendientes,
  'requisitos_sha256',encode(sha256(convert_to(requisitos::text,'UTF8')),'hex'),
  'comprobada_en',p_instante);
END $function$
;

DO $post$
DECLARE r record; f regprocedure; p record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('comprobar','vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)','80df8013324a3915386faddaa1073a5f32ec86362214ac80ed2e96670718cf23','{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario=X/vec_bolsa_convocatorias_propietario}','v','u'),
   ('proyectar','vec_bolsa_convocatorias.proyectar_abierta_inscripcion_v1(text,bigint,bytea,text,timestamp with time zone)','b15e0b0ba804d54c1e258702a64919b68874aa94df60f70cda08e27972b0e560','{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario}','s','r')
  ) AS esperado(clave,firma,huella,acl,volatilidad,paralelismo) LOOP
  f:=to_regprocedure(r.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'BC14: funcion causal ausente' USING ERRCODE='55000'; END IF;
  SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>r.volatilidad OR p.proparallel::text<>r.paralelismo
  OR p.proacl::text IS DISTINCT FROM r.acl OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>r.huella
  THEN RAISE EXCEPTION 'BC14: definicion o metadatos divergentes' USING ERRCODE='55000'; END IF;
  IF f::oid::text IS DISTINCT FROM current_setting('vec.bc14.oid_'||r.clave,true) THEN RAISE EXCEPTION 'BC14: OID divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc x CROSS JOIN LATERAL aclexplode(coalesce(x.proacl,acldefault('f',x.proowner))) a
   WHERE x.oid IN(to_regprocedure('vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(text,bigint,text)'),
    to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)'))
    AND a.grantee<>x.proowner)
 OR EXISTS(SELECT 1 FROM pg_class t CROSS JOIN LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
   WHERE t.oid IN('vec_bolsa_convocatorias.publicacion_oficial_bases'::regclass,
    'vec_bolsa_convocatorias.publicacion_oficial_bases_evento'::regclass) AND a.grantee<>t.relowner)
 OR EXISTS(SELECT 1 FROM pg_class t WHERE t.oid IN('vec_bolsa_convocatorias.publicacion_oficial_bases'::regclass,
    'vec_bolsa_convocatorias.publicacion_oficial_bases_evento'::regclass)
    AND (NOT t.relrowsecurity OR NOT t.relforcerowsecurity
     OR pg_get_userbyid(t.relowner)<>'vec_bolsa_convocatorias_propietario'))
 THEN RAISE EXCEPTION 'BC14: concesiones o protecciones inesperadas' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
