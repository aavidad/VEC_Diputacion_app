\set ON_ERROR_STOP on
-- B88: ajustes de reglas de Bolsa (plazos configurables).
-- Estudio: docs/estudio_requisitos/plazos_configurables_2026-09-30.md.
--
-- El catálogo base de reglas no está en la base: lo resuelve Go. Aquí vive
-- solo la capa de ajustes que RRHH fija desde la aplicación, versionada y de
-- solo adición: cada versión guarda todos los ajustes vigentes, quién la
-- publicó, cuándo, por qué y el antes/después de cada campo cambiado. La
-- validación contra la regla base (campos editables, opciones y límites) es
-- del dueño del catálogo y del resolutor. La base comprueba forma, límites,
-- encadenamiento de versiones, delta frente a la cabeza y huella.
--
-- AD223 debe precederla. Sin DOWN tras historia.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000088',0));
DO $pre$
DECLARE f oid;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_bolsa_llamamientos.regla_ajuste_version_v1') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor')
 THEN RAISE EXCEPTION 'B88: preimagen incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_bolsa_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'B88: AD223 requerida' USING ERRCODE='55000'; END IF;
END $pre$;

-- Forma de un conjunto de ajustes: objeto de como mucho 64 reglas; cada una,
-- de 1 a 4 campos ajustables con valor [a-z0-9_]{1,64} (enteros canónicos o
-- claves de unidad y cómputo). Con ese alfabeto la forma canónica de Go y la
-- de la base no pueden divergir por escapes.
CREATE FUNCTION vec_bolsa_llamamientos.ajustes_reglas_forma_valida_v1(p jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT jsonb_typeof(p)='object'
   AND (SELECT count(*) FROM jsonb_object_keys(p))<=64
   AND NOT EXISTS (
    SELECT 1 FROM jsonb_each(p) r
     WHERE r.key !~ '^[a-z0-9][a-z0-9._-]{1,79}$'
        OR jsonb_typeof(r.value)<>'object'
        OR (SELECT count(*) FROM jsonb_object_keys(r.value)) NOT BETWEEN 1 AND 4
        OR EXISTS (SELECT 1 FROM jsonb_each(r.value) c
                    WHERE c.key<>ALL(ARRAY['cantidad','cantidad_urgente','unidad','computo'])
                       OR jsonb_typeof(c.value)<>'string'
                       OR c.value#>>'{}' !~ '^[a-z0-9_]{1,64}$'))
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ajustes_reglas_forma_valida_v1(jsonb) FROM PUBLIC;

CREATE TABLE vec_bolsa_llamamientos.regla_ajuste_version_v1 (
 catalogo_id text NOT NULL CHECK(catalogo_id='vec.bolsa.reglas.ajustes'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 9999999),
 version_esperada bigint NOT NULL,
 organizacion_ref text NOT NULL CHECK(organizacion_ref='organizacion:desarrollo:dipgra'),
 ajustes jsonb NOT NULL CHECK(vec_bolsa_llamamientos.ajustes_reglas_forma_valida_v1(ajustes)),
 ajustes_canonico text NOT NULL CHECK(octet_length(ajustes_canonico)<=16384),
 huella_sha256 text NOT NULL CHECK(huella_sha256~'^[0-9a-f]{64}$'),
 base_version bigint NOT NULL CHECK(base_version BETWEEN 1 AND 9999999),
 base_huella_sha256 text NOT NULL CHECK(base_huella_sha256~'^[0-9a-f]{64}$'),
 material_original jsonb NOT NULL CHECK(jsonb_typeof(material_original)='object' AND octet_length(material_original::text)<=65536),
 actor_ref text NOT NULL CHECK(actor_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 motivo_clave text NOT NULL CHECK(motivo_clave~'^[a-z][a-z0-9_]{2,63}$'),
 referencia text CHECK(referencia IS NULL OR (char_length(referencia) BETWEEN 1 AND 120
   AND referencia=btrim(referencia) AND referencia !~ '[[:cntrl:]]')),
 nota text CHECK(nota IS NULL OR (char_length(nota) BETWEEN 1 AND 500
   AND nota=btrim(nota) AND nota !~ '[[:cntrl:]]')),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en) AND publicada_en=date_trunc('microseconds',publicada_en)),
 vigente_desde timestamptz(6) NOT NULL CHECK(isfinite(vigente_desde) AND vigente_desde=date_trunc('microseconds',vigente_desde) AND vigente_desde>=publicada_en),
 clave_idempotencia uuid NOT NULL UNIQUE,
 solicitud_huella_sha256 text NOT NULL CHECK(solicitud_huella_sha256~'^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^recibo:[0-9a-f-]{36}$'),
 decision_ref text NOT NULL CHECK(decision_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL CHECK(auditoria_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 UNIQUE(decision_ref),
 UNIQUE(auditoria_ref),
 PRIMARY KEY(catalogo_id,version),
 CHECK(version=version_esperada+1),
 CHECK(material_original->>'catalogo_id'=catalogo_id),
 CHECK(material_original->>'clave_idempotencia'=clave_idempotencia::text),
 CHECK(huella_sha256=encode(sha256(convert_to(ajustes_canonico,'UTF8')),'hex')),
 CHECK(ajustes_canonico::jsonb=ajustes)
);
CREATE INDEX regla_ajuste_version_vigencia_v1
 ON vec_bolsa_llamamientos.regla_ajuste_version_v1(catalogo_id,vigente_desde DESC,version DESC);

CREATE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_ajuste_reglas_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RAISE EXCEPTION 'B88: historia inmutable' USING ERRCODE='55000';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_ajuste_reglas_v1() FROM PUBLIC;

-- Antes y después de cada campo cambiado en cada versión.
CREATE TABLE vec_bolsa_llamamientos.regla_ajuste_cambio_v1 (
 catalogo_id text NOT NULL,
 version bigint NOT NULL,
 regla_clave text NOT NULL CHECK(regla_clave~'^[a-z0-9][a-z0-9._-]{1,79}$'),
 campo text NOT NULL CHECK(campo IN ('cantidad','cantidad_urgente','unidad','computo')),
 anterior text NOT NULL CHECK(anterior~'^[a-z0-9_]{1,64}$'),
 nuevo text NOT NULL CHECK(nuevo~'^[a-z0-9_]{1,64}$'),
 PRIMARY KEY(catalogo_id,version,regla_clave,campo),
 CHECK(anterior<>nuevo),
 FOREIGN KEY(catalogo_id,version) REFERENCES vec_bolsa_llamamientos.regla_ajuste_version_v1(catalogo_id,version)
);

-- Un evento por versión: catálogo, versión y huella; nunca actor ni motivo.
CREATE TABLE vec_bolsa_llamamientos.regla_ajuste_outbox_v1 (
 evento_ref text PRIMARY KEY CHECK(evento_ref~'^evento:[0-9a-f-]{36}$'),
 catalogo_id text NOT NULL,
 version bigint NOT NULL,
 huella_sha256 text NOT NULL CHECK(huella_sha256~'^[0-9a-f]{64}$'),
 tipo text NOT NULL CHECK(tipo='bolsa.reglas.ajustadas'),
 estado text NOT NULL DEFAULT 'pendiente' CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(catalogo_id,version),
 FOREIGN KEY(catalogo_id,version) REFERENCES vec_bolsa_llamamientos.regla_ajuste_version_v1(catalogo_id,version)
);

DO $proteccion$
DECLARE tabla text; r record;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['regla_ajuste_version_v1','regla_ajuste_cambio_v1','regla_ajuste_outbox_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_ajuste_reglas_v1()',tabla);
  EXECUTE format('CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_mutacion_ajuste_reglas_v1()',tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_total ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC',tabla);
  FOR r IN SELECT DISTINCT a.grantee FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid=to_regclass('vec_bolsa_llamamientos.'||tabla) AND a.grantee<>0 AND a.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM %I',tabla,pg_get_userbyid(r.grantee));
  END LOOP;
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_llamamientos.%I FROM PUBLIC',tabla);
 END LOOP;
END $proteccion$;

-- Lectura del resolutor de reglas: la versión de ajustes vigente en un
-- instante (la de mayor número con vigente_desde no posterior). No devuelve
-- actor, motivo ni nota: los ajustes no llevan datos personales y se leen en
-- cada cálculo de plazo, sin decisión V3. Sin filas: aún no hay ajustes.
CREATE FUNCTION vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(p_catalogo text,p_instante timestamptz)
RETURNS TABLE(version bigint,huella_sha256 text,ajustes_canonico text,vigente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $leer$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 THEN RAISE EXCEPTION 'B88: lectura denegada' USING ERRCODE='42501'; END IF;
 IF p_catalogo IS DISTINCT FROM 'vec.bolsa.reglas.ajustes'
    OR p_instante IS NULL OR NOT isfinite(p_instante)
 THEN RAISE EXCEPTION 'B88: lectura inválida' USING ERRCODE='22023'; END IF;
 RETURN QUERY
  SELECT v.version,v.huella_sha256,v.ajustes_canonico,v.vigente_desde
    FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
   WHERE v.catalogo_id=p_catalogo AND v.vigente_desde<=p_instante
   ORDER BY v.vigente_desde DESC,v.version DESC LIMIT 1;
END $leer$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;

-- Preparación del CAS: la cabeza publicada puede tener efecto futuro. Esta
-- proyección omite actor, motivo, referencia y nota como la lectura efectiva.
CREATE FUNCTION vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1(p_catalogo text)
RETURNS TABLE(version bigint,huella_sha256 text,ajustes_canonico text,
 publicada_en timestamptz,vigente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $cabeza$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 THEN RAISE EXCEPTION 'B88: lectura denegada' USING ERRCODE='42501'; END IF;
 IF p_catalogo IS DISTINCT FROM 'vec.bolsa.reglas.ajustes'
 THEN RAISE EXCEPTION 'B88: lectura inválida' USING ERRCODE='22023'; END IF;
 RETURN QUERY SELECT v.version,v.huella_sha256,v.ajustes_canonico,
   v.publicada_en,v.vigente_desde
  FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_catalogo ORDER BY v.version DESC LIMIT 1;
END $cabeza$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1(text) TO vec_bolsa_llamamientos_ejecutor;

-- Herramienta interna del propietario. Contiene actor y motivo y, por tanto,
-- no recibe EXECUTE del runtime. La futura ruta GET debe pasar por la
-- autoridad común de lectura W/V y auditar en la misma transacción.
CREATE FUNCTION vec_bolsa_llamamientos.leer_historial_ajustes_reglas_v1(
 p_catalogo text,p_limite integer,p_antes_de bigint,p_instante timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY INVOKER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $historial$
DECLARE cabeza jsonb; vigente jsonb; filas jsonb; hay_mas boolean;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_bolsa_llamamientos_propietario'
 THEN RAISE EXCEPTION 'B88: historial denegado' USING ERRCODE='42501'; END IF;
 IF p_catalogo IS DISTINCT FROM 'vec.bolsa.reglas.ajustes'
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 50
    OR p_antes_de IS NOT NULL AND p_antes_de NOT BETWEEN 2 AND 10000000
    OR p_instante IS NULL OR NOT isfinite(p_instante)
 THEN RAISE EXCEPTION 'B88: historial inválido' USING ERRCODE='22023'; END IF;
 SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,
   'ajustes',v.ajustes,'publicada_en',v.publicada_en,'vigente_desde',v.vigente_desde,
   'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
  INTO cabeza FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_catalogo ORDER BY v.version DESC LIMIT 1;
 SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,
   'ajustes',v.ajustes,'publicada_en',v.publicada_en,'vigente_desde',v.vigente_desde,
   'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
  INTO vigente FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_catalogo AND v.vigente_desde<=p_instante
  ORDER BY v.vigente_desde DESC,v.version DESC LIMIT 1;
 SELECT coalesce(jsonb_agg(h.fila ORDER BY h.version DESC),'[]'::jsonb)
  INTO filas FROM (
   SELECT v.version,jsonb_build_object('version',v.version,
     'publicada_en',v.publicada_en,'vigente_desde',v.vigente_desde,
     'actor_ref',v.actor_ref,'motivo_clave',v.motivo_clave,
     'referencia',v.referencia,'nota',v.nota,'base_version',v.base_version,
     'recibo_ref',v.recibo_ref,'cambios',(
       SELECT coalesce(jsonb_agg(jsonb_build_object('regla_clave',c.regla_clave,
        'campo',c.campo,'anterior',c.anterior,'nuevo',c.nuevo)
        ORDER BY c.regla_clave,c.campo),'[]'::jsonb)
       FROM vec_bolsa_llamamientos.regla_ajuste_cambio_v1 c
       WHERE c.catalogo_id=v.catalogo_id AND c.version=v.version)) AS fila
   FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
   WHERE v.catalogo_id=p_catalogo AND (p_antes_de IS NULL OR v.version<p_antes_de)
   ORDER BY v.version DESC LIMIT p_limite) h;
 SELECT count(*)>p_limite INTO hay_mas FROM (
  SELECT 1 FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_catalogo AND (p_antes_de IS NULL OR v.version<p_antes_de)
  LIMIT p_limite+1) x;
 RETURN jsonb_build_object('cabeza',cabeza,'vigente',vigente,
   'historial',filas,'hay_mas',hay_mas);
END $historial$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_historial_ajustes_reglas_v1(text,integer,bigint,timestamptz) FROM PUBLIC;

-- El material original puede contener nota/referencia libres. Sólo lo puede
-- usar una futura fachada de lectura nominal con W/V y auditoría misma TX.
CREATE FUNCTION vec_bolsa_llamamientos.leer_material_original_ajuste_reglas_v1(
 p_catalogo text,p_clave uuid)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY INVOKER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $original$
DECLARE material jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user<>'vec_bolsa_llamamientos_propietario'
 THEN RAISE EXCEPTION 'B88: material original denegado' USING ERRCODE='42501'; END IF;
 IF p_catalogo IS DISTINCT FROM 'vec.bolsa.reglas.ajustes' OR p_clave IS NULL
 THEN RAISE EXCEPTION 'B88: consulta original inválida' USING ERRCODE='22023'; END IF;
 SELECT v.material_original INTO material
 FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
 WHERE v.catalogo_id=p_catalogo AND v.clave_idempotencia=p_clave;
 RETURN material;
END $original$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_material_original_ajuste_reglas_v1(text,uuid) FROM PUBLIC;

-- Solo escritura gobernada. El historial personal se leerá mediante la
-- frontera común de lectura nominal y auditoría, nunca directamente por el
-- ejecutor. La decisión V3 cubre el material completo, incluida fecha futura.
CREATE FUNCTION vec_bolsa_llamamientos.operar_ajustes_reglas_v1(
 p_material jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 d jsonb; consumo record; previa record; anterior record;
 actor text; clave uuid; esperada bigint; nueva bigint;
 ajustes jsonb; canonico text; huella text; previos jsonb; cambio jsonb; r record; c record; solicitud_h text;
 publicada timestamptz(6); efecto timestamptz(6); recibo text; n integer;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B88: ejecución denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object' OR octet_length(p_material::text)>65536
 THEN RAISE EXCEPTION 'B88: material inválido' USING ERRCODE='22023'; END IF;
 IF p_material->>'operacion' IS DISTINCT FROM 'ajustar'
    OR p_material->>'catalogo_id' IS DISTINCT FROM 'vec.bolsa.reglas.ajustes'
    OR p_material->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
    OR NOT p_material ?& ARRAY['operacion','organizacion_ref','catalogo_id','clave_idempotencia',
         'version_esperada','base_version','base_huella_sha256','ajustes_canonico',
         'ajustes_huella_sha256','cambios','motivo_clave']
    OR (p_material-ARRAY['operacion','organizacion_ref','catalogo_id','clave_idempotencia',
         'version_esperada','base_version','base_huella_sha256','ajustes_canonico','ajustes_huella_sha256',
         'cambios','motivo_clave','referencia','nota','vigente_desde'])<>'{}'::jsonb
 THEN RAISE EXCEPTION 'B88: operación inválida' USING ERRCODE='22023'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B88: decisión inválida' USING ERRCODE='22023'; END;
 actor:=d->>'principal_id';
 IF actor IS NULL OR actor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.reglas.ajustar'
 THEN RAISE EXCEPTION 'B88: decisión divergente' USING ERRCODE='42501'; END IF;
 -- AD223 calcula la huella del contexto desde este mismo material.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_bolsa_v3_atestada(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM 'vec.bolsa.reglas'
 THEN RAISE EXCEPTION 'B88: consumo divergente' USING ERRCODE='42501'; END IF;

 -- Ajustar.
 BEGIN
  clave:=(p_material->>'clave_idempotencia')::uuid;
  esperada:=(p_material->>'version_esperada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B88: versión o clave inválida' USING ERRCODE='22023'; END;
 IF clave IS NULL OR p_material->>'clave_idempotencia'<>clave::text
    OR jsonb_typeof(p_material->'version_esperada')<>'number' OR esperada IS NULL OR esperada NOT BETWEEN 0 AND 9999998
    OR jsonb_typeof(p_material->'base_version')<>'number'
    OR coalesce(p_material->>'base_version','') !~ '^[1-9][0-9]{0,6}$'
    OR coalesce(p_material->>'base_huella_sha256','') !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_material->'ajustes_canonico')<>'string'
    OR coalesce(p_material->>'ajustes_huella_sha256','') !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_material->'cambios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_material->'cambios') NOT BETWEEN 1 AND 256
    OR coalesce(p_material->>'motivo_clave','') !~ '^[a-z][a-z0-9_]{2,63}$'
    OR (p_material ? 'referencia' AND jsonb_typeof(p_material->'referencia')<>'string')
    OR (p_material ? 'nota' AND jsonb_typeof(p_material->'nota')<>'string')
    OR (p_material ? 'referencia' AND (char_length(p_material->>'referencia') NOT BETWEEN 1 AND 120
         OR p_material->>'referencia' IS DISTINCT FROM btrim(p_material->>'referencia')
         OR p_material->>'referencia' ~ '[[:cntrl:]]'))
    OR (p_material ? 'nota' AND (char_length(p_material->>'nota') NOT BETWEEN 1 AND 500
         OR p_material->>'nota' IS DISTINCT FROM btrim(p_material->>'nota')
         OR p_material->>'nota' ~ '[[:cntrl:]]'))
    OR (p_material ? 'vigente_desde' AND
         (jsonb_typeof(p_material->'vigente_desde')<>'string'
          OR coalesce(p_material->>'vigente_desde','') !~
           '^[0-9]{4}-(0[1-9]|1[0-2])-([0-2][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{1,6})?Z$'))
 THEN RAISE EXCEPTION 'B88: ajuste inválido' USING ERRCODE='22023'; END IF;
 IF p_material ? 'vigente_desde' THEN
  BEGIN efecto:=(p_material->>'vigente_desde')::timestamptz(6);
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'B88: fecha inválida' USING ERRCODE='22023'; END;
  IF NOT isfinite(efecto) THEN
   RAISE EXCEPTION 'B88: fecha inválida' USING ERRCODE='22023'; END IF;
 END IF;
 canonico:=p_material->>'ajustes_canonico';
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 IF octet_length(canonico)>16384 OR huella<>p_material->>'ajustes_huella_sha256'
 THEN RAISE EXCEPTION 'B88: huella de ajustes divergente' USING ERRCODE='22023'; END IF;
 BEGIN ajustes:=canonico::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B88: ajustes ilegibles' USING ERRCODE='22023'; END;
 IF NOT vec_bolsa_llamamientos.ajustes_reglas_forma_valida_v1(ajustes)
 THEN RAISE EXCEPTION 'B88: forma de ajustes inválida' USING ERRCODE='22023'; END IF;

 IF EXISTS (SELECT 1 FROM jsonb_array_elements(p_material->'cambios') e
             WHERE jsonb_typeof(e)<>'object' OR (e-ARRAY['regla_clave','campo','anterior','nuevo'])<>'{}'::jsonb
                OR jsonb_typeof(e->'regla_clave')<>'string' OR jsonb_typeof(e->'campo')<>'string'
                OR jsonb_typeof(e->'anterior')<>'string' OR jsonb_typeof(e->'nuevo')<>'string')
 THEN RAISE EXCEPTION 'B88: cambio mal formado' USING ERRCODE='22023'; END IF;
 -- La repetición se reconoce por lo que pidió el cliente (clave, versión
 -- esperada, valores nuevos y motivo), no por el material entero: el valor
 -- anterior y la instantánea se recalculan sobre la cabeza, que puede haber
 -- avanzado precisamente por la primera ejecución de esta misma petición.
 solicitud_h:=encode(sha256(convert_to(jsonb_build_object(
   'organizacion_ref',p_material->>'organizacion_ref','clave_idempotencia',clave::text,'version_esperada',esperada,
   'cambios',(SELECT jsonb_agg(jsonb_build_object('regla_clave',e->>'regla_clave','campo',e->>'campo','nuevo',e->>'nuevo')
               ORDER BY e->>'regla_clave',e->>'campo') FROM jsonb_array_elements(p_material->'cambios') e),
   'motivo_clave',p_material->>'motivo_clave','referencia',p_material->>'referencia','nota',p_material->>'nota',
   'vigente_desde',p_material->>'vigente_desde')::text,'UTF8')),'hex');

 -- Serializa escritores cooperantes. La lectura de la cabeza no bloquea la
 -- instantánea de otro escritor serializable: la PK o UNIQUE detecta el
 -- conflicto concurrente con 40001 en el cliente que debe reintentar.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:ajustes_reglas',0));
 SELECT * INTO previa FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v WHERE v.clave_idempotencia=clave;
 IF FOUND THEN
  IF previa.solicitud_huella_sha256 IS DISTINCT FROM solicitud_h OR previa.actor_ref IS DISTINCT FROM actor
  THEN RAISE EXCEPTION 'B88: clave reutilizada con otra solicitud' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('ajustes',previa.ajustes,'replay',true,'recibo',jsonb_build_object(
   'recibo_ref',previa.recibo_ref,'clave_idempotencia',clave,'version',previa.version,
   'huella_sha256',previa.huella_sha256,'publicada_en',previa.publicada_en,'vigente_desde',previa.vigente_desde,
   'decision_ref',previa.decision_ref,'auditoria_ref',previa.auditoria_ref,
   'consumo_huella_sha256',previa.consumo_huella_sha256));
 END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_material->>'catalogo_id' ORDER BY v.version DESC LIMIT 1;
 IF coalesce(anterior.version,0)<>esperada
 THEN RAISE EXCEPTION 'B88: conflicto de versión' USING ERRCODE='40001'; END IF;
 previos:=coalesce(anterior.ajustes,'{}'::jsonb);
 nueva:=esperada+1;

 -- Cada cambio: campo único, valor nuevo en la versión nueva y, si ya estaba
 -- ajustado, el anterior coincide con la versión previa.
 n:=0;
 FOR cambio IN SELECT e FROM jsonb_array_elements(p_material->'cambios') e LOOP
  IF cambio->>'anterior'=cambio->>'nuevo'
     OR ajustes #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS DISTINCT FROM cambio->>'nuevo'
     OR (previos #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS NOT NULL
         AND previos #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS DISTINCT FROM cambio->>'anterior')
  THEN RAISE EXCEPTION 'B88: cambio incoherente con los ajustes' USING ERRCODE='22023'; END IF;
  n:=n+1;
 END LOOP;
 IF (SELECT count(DISTINCT (e->>'regla_clave',e->>'campo')) FROM jsonb_array_elements(p_material->'cambios') e)<>n
 THEN RAISE EXCEPTION 'B88: cambio repetido' USING ERRCODE='22023'; END IF;
 -- Nada cambia sin su fila de cambio y ningún ajuste previo desaparece.
 FOR r IN SELECT x.key AS regla,y.key AS campo,y.value#>>'{}' AS valor
           FROM jsonb_each(ajustes) x,LATERAL jsonb_each(x.value) y LOOP
  IF previos #>> ARRAY[r.regla,r.campo] IS DISTINCT FROM r.valor AND NOT EXISTS (
     SELECT 1 FROM jsonb_array_elements(p_material->'cambios') e
      WHERE e->>'regla_clave'=r.regla AND e->>'campo'=r.campo)
  THEN RAISE EXCEPTION 'B88: ajuste sin cambio declarado' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR c IN SELECT x.key AS regla,y.key AS campo FROM jsonb_each(previos) x,LATERAL jsonb_each(x.value) y LOOP
  IF ajustes #>> ARRAY[c.regla,c.campo] IS NULL
  THEN RAISE EXCEPTION 'B88: un ajuste previo no puede desaparecer' USING ERRCODE='22023'; END IF;
 END LOOP;

 -- La publicación y el efecto son instantes distintos. No se permite
 -- intercalar una fecha efectiva anterior a la cabeza publicada.
 publicada:=date_trunc('microseconds',clock_timestamp());
 IF efecto IS NULL THEN efecto:=publicada; END IF;
 IF efecto<publicada OR (anterior.version IS NOT NULL AND efecto<anterior.vigente_desde)
 THEN RAISE EXCEPTION 'B88: fecha de efecto anterior a la cabeza' USING ERRCODE='40001'; END IF;
 recibo:='recibo:'||gen_random_uuid()::text;
 INSERT INTO vec_bolsa_llamamientos.regla_ajuste_version_v1(
  catalogo_id,version,version_esperada,organizacion_ref,ajustes,ajustes_canonico,huella_sha256,
  base_version,base_huella_sha256,material_original,actor_ref,motivo_clave,referencia,nota,publicada_en,vigente_desde,
  clave_idempotencia,solicitud_huella_sha256,recibo_ref,decision_ref,consumo_huella_sha256,auditoria_ref)
 VALUES(p_material->>'catalogo_id',nueva,esperada,p_material->>'organizacion_ref',ajustes,canonico,huella,
  (p_material->>'base_version')::bigint,p_material->>'base_huella_sha256',p_material,actor,p_material->>'motivo_clave',
  p_material->>'referencia',p_material->>'nota',publicada,efecto,
  clave,solicitud_h,recibo,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref);
 INSERT INTO vec_bolsa_llamamientos.regla_ajuste_cambio_v1(catalogo_id,version,regla_clave,campo,anterior,nuevo)
 SELECT p_material->>'catalogo_id',nueva,e->>'regla_clave',e->>'campo',e->>'anterior',e->>'nuevo'
   FROM jsonb_array_elements(p_material->'cambios') e;
 INSERT INTO vec_bolsa_llamamientos.regla_ajuste_outbox_v1(evento_ref,catalogo_id,version,huella_sha256,tipo,estado,creada_en)
 VALUES('evento:'||gen_random_uuid()::text,p_material->>'catalogo_id',nueva,huella,
  'bolsa.reglas.ajustadas','pendiente',publicada);
 RETURN jsonb_build_object('ajustes',ajustes,'replay',false,'recibo',jsonb_build_object(
  'recibo_ref',recibo,'clave_idempotencia',clave,'version',nueva,'huella_sha256',huella,
  'publicada_en',publicada,'vigente_desde',efecto,'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumo_huella_sha256',consumo.consumo_huella_sha256));
END $funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

DO $acl$
DECLARE f regprocedure; n text; a record; permitido regrole;
BEGIN
 FOREACH n IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(text,timestamptz)',
  'vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1(text)',
  'vec_bolsa_llamamientos.leer_historial_ajustes_reglas_v1(text,integer,bigint,timestamptz)',
  'vec_bolsa_llamamientos.leer_material_original_ajuste_reglas_v1(text,uuid)',
  'vec_bolsa_llamamientos.rechazar_mutacion_ajuste_reglas_v1()',
  'vec_bolsa_llamamientos.ajustes_reglas_forma_valida_v1(jsonb)'] LOOP
  f:=n::regprocedure;
  permitido:=CASE WHEN n LIKE '%operar_ajustes_reglas_v1%' OR n LIKE '%leer_ajustes_reglas_en_v1%'
                   OR n LIKE '%leer_cabeza_ajustes_reglas_v1%'
                  THEN 'vec_bolsa_llamamientos_ejecutor'::regrole ELSE NULL END;
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f AND x.grantee<>p.proowner AND x.grantee IS DISTINCT FROM permitido LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
     CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
     OR (permitido IS NOT NULL AND (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE)
     OR ((n LIKE '%historial%' OR n LIKE '%material_original%')
         AND (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT FALSE)
     OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                WHERE p.oid=f AND (x.grantee<>p.proowner AND x.grantee IS DISTINCT FROM permitido
                                   OR x.privilege_type<>'EXECUTE'))
  THEN RAISE EXCEPTION 'B88: ACL de fachada incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
