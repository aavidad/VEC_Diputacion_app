\set ON_ERROR_STOP on
-- F1-19. La instrucción sigue perteneciendo al padre; cada hijo tiene
-- identidad, aceptación, actuación, autorización y recibo propios.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000234',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.nombramiento_derivado_v1') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
       WHERE c.conrelid='vec_contratacion_temporal.expediente_alta'::regclass
         AND c.conname='expediente_alta_numero_visible_check' AND c.convalidated
         AND strpos(pg_get_constraintdef(c.oid),'^[0-9]{4}/[A-Za-z0-9._-]{1,40}$')>0)
    OR (SELECT count(*) FROM pg_attribute a
        WHERE a.attrelid='vec_contratacion_temporal.expediente_alta'::regclass
          AND a.attname IN ('reserva_ref','confirmacion_ref') AND a.attnotnull)<>2
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
        WHERE c.conrelid='vec_contratacion_temporal.expediente_alta'::regclass
          AND c.conname='expediente_marcador_fk' AND c.contype='f')
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(text,text,text,integer,integer,text,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'CT234: dependencias o preimagen incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.nombramiento_derivado_v1(
 derivacion_ref text PRIMARY KEY,
 organizacion_ref text NOT NULL,
 padre_ref text NOT NULL REFERENCES vec_contratacion_temporal.expediente_alta,
 version_padre numeric NOT NULL CHECK(version_padre BETWEEN 6 AND 9007199254740991),
 hijo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.expediente_alta DEFERRABLE INITIALLY DEFERRED,
 oferta_ref text NOT NULL CHECK(oferta_ref ~ '^oferta:[0-9a-f]{64}$'),
 numero_plaza integer NOT NULL CHECK(numero_plaza BETWEEN 1 AND 100),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 aceptacion_ref text NOT NULL UNIQUE,
 evidencia_bolsa jsonb NOT NULL CHECK(jsonb_typeof(evidencia_bolsa)='object'),
 clave_idempotencia uuid NOT NULL,
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 material text NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 262144),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 recibo_json jsonb NOT NULL CHECK(jsonb_typeof(recibo_json)='object'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(organizacion_ref,padre_ref,numero_plaza),
 UNIQUE(organizacion_ref,clave_idempotencia),
 CHECK(padre_ref<>hijo_ref),
 FOREIGN KEY(padre_ref,version_padre) REFERENCES vec_contratacion_temporal.expediente_version_integral
);
ALTER TABLE vec_contratacion_temporal.nombramiento_derivado_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.nombramiento_derivado_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.nombramiento_derivado_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON vec_contratacion_temporal.nombramiento_derivado_v1 FROM PUBLIC,vec_contratacion_temporal_ejecutor;
REVOKE ALL ON TYPE vec_contratacion_temporal.nombramiento_derivado_v1 FROM PUBLIC;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.nombramiento_derivado_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.nombramiento_derivado_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Dos clases positivas de alta: reserva ordinaria o derivación acreditada.
-- Nunca se rellena una reserva ficticia ni un HMAC aparente.
ALTER TABLE vec_contratacion_temporal.expediente_alta
 ADD COLUMN derivacion_ref text UNIQUE REFERENCES vec_contratacion_temporal.nombramiento_derivado_v1 DEFERRABLE INITIALLY DEFERRED,
 ALTER COLUMN reserva_ref DROP NOT NULL,
 ALTER COLUMN confirmacion_ref DROP NOT NULL,
 DROP CONSTRAINT expediente_alta_numero_visible_check;
ALTER TABLE vec_contratacion_temporal.expediente_alta ADD CONSTRAINT alta_origen_ct234 CHECK(
 (derivacion_ref IS NULL AND reserva_ref IS NOT NULL AND confirmacion_ref IS NOT NULL
  AND numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$')
 OR (derivacion_ref IS NOT NULL AND reserva_ref IS NULL AND confirmacion_ref IS NULL
  AND numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}-P(100|[1-9][0-9]?)$'));
DO $origen$
DECLARE expresion text;
BEGIN
 SELECT pg_get_expr(conbin,conrelid) INTO STRICT expresion FROM pg_constraint
 WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
   AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
 IF expresion IS NULL OR position('derivacion_nombramiento_ct234' IN expresion)>0
    OR position('reincorporacion_titular_ct130' IN expresion)=0 THEN
  RAISE EXCEPTION 'CT234: origen ya instalado' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_contratacion_temporal.expediente_version_integral DROP CONSTRAINT expediente_version_integral_origen_version_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check CHECK(('
  ||expresion||') OR origen_version=''derivacion_nombramiento_ct234'')';
END $origen$;

-- Las tres funciones instaladas conservan OID, propietario, ACL y cuerpo
-- salvo la sustitución única indicada. SHA de prosrc calculados desde las
-- últimas definiciones de origin/main: CT037, CT110 y CT190, respectivamente.
DO $publicacion$
DECLARE f regprocedure; p pg_proc%ROWTYPE; definicion text; marca text; nueva text;
 con record; expresion text;
BEGIN
 SELECT c.conname,pg_get_constraintdef(c.oid) AS definicion INTO STRICT con
 FROM pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.publicacion_version_rrhh'::regclass
   AND c.contype='c' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid),
     'numero_visible ~ ''^[0-9]{4}/[A-Za-z0-9._-]{1,40}$''')>0;
 IF (length(con.definicion)-length(replace(con.definicion,
      '''^[0-9]{4}/[A-Za-z0-9._-]{1,40}$''','')))
      <>length('''^[0-9]{4}/[A-Za-z0-9._-]{1,40}$''') THEN
  RAISE EXCEPTION 'CT234: preimagen de número publicado divergente' USING ERRCODE='55000'; END IF;
 expresion:=replace(con.definicion,
  '''^[0-9]{4}/[A-Za-z0-9._-]{1,40}$''',
  '''^([0-9]{4}/[A-Za-z0-9._-]{1,40}|[0-9]{4}/[A-Za-z0-9._-]{1,40}-P(100|[1-9][0-9]?))$''');
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh DROP CONSTRAINT %I',con.conname);
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh ADD CONSTRAINT %I %s',con.conname,expresion);

 f:='vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'::regprocedure;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')<>
      'daba512040d294cd5f31d0627990509c661957bfbf42ba53f101c09a84d5361f'
    OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.provolatile<>'s'
    OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'CT234: preimagen del extractor divergente' USING ERRCODE='55000'; END IF;
 marca:=$m$OR COALESCE(numero_visible, '') !~
            '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'$m$;
 nueva:=$n$OR (COALESCE(numero_visible, '') !~
            '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
          AND NOT (COALESCE(numero_visible, '') ~
            '^[0-9]{4}/[A-Za-z0-9._-]{1,40}-P(100|[1-9][0-9]?)$'
            AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta_derivada
              JOIN vec_contratacion_temporal.nombramiento_derivado_v1 derivada
                ON derivada.derivacion_ref=alta_derivada.derivacion_ref
              WHERE alta_derivada.expediente_ref=p_expediente_ref
                AND alta_derivada.numero_visible=p_numero_esperado
                AND derivada.hijo_ref=p_expediente_ref)))$n$;
 definicion:=pg_get_functiondef(f);
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'CT234: marca del extractor no única' USING ERRCODE='55000'; END IF;
 EXECUTE replace(definicion,marca,nueva);
 ALTER FUNCTION vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
  text,numeric,jsonb,text,text,numeric,text,text,text,timestamptz,text,text)
  SET search_path TO pg_catalog,pg_temp;
END $publicacion$;

DO $inicio_hijo$
DECLARE f regprocedure; p pg_proc%ROWTYPE; definicion text; marca text; nueva text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()'::regprocedure,
  'vec_contratacion_temporal.registrar_instantanea_fase_regla_v1()'::regprocedure] LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR NOT p.prosecdef OR p.provolatile<>'v'
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
        (CASE WHEN f::text LIKE '%registrar_fase_entrada%'
         THEN 'd42ae05e1fb4db85551379ceec97848eda218cdc5a99bb264b05225396104fdc'
         ELSE '6a519c1c2ca4cd3b8af954bfd3fe3a5530c5a5e4b4d3d416ce18228d625767ac' END)
     OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE a.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'CT234: preimagen de fase divergente: %',f USING ERRCODE='55000'; END IF;
  marca:=CASE WHEN f::text LIKE '%registrar_fase_entrada%'
      THEN 'IF NEW.version > 1 THEN' ELSE 'IF NEW.version>1 THEN' END;
  nueva:=$n$IF NEW.version>1 AND NOT (NEW.version=6 AND EXISTS(
      SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta_derivada
      JOIN vec_contratacion_temporal.nombramiento_derivado_v1 derivada
        ON derivada.derivacion_ref=alta_derivada.derivacion_ref
      JOIN vec_contratacion_temporal.expediente_version_integral inicial
        ON inicial.expediente_ref=derivada.hijo_ref AND inicial.version=6
      WHERE alta_derivada.expediente_ref=NEW.expediente_ref
        AND derivada.hijo_ref=NEW.expediente_ref
        AND inicial.origen_version='derivacion_nombramiento_ct234')) THEN$n$;
  definicion:=pg_get_functiondef(f);
  IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'CT234: marca de fase no única: %',f USING ERRCODE='55000'; END IF;
  EXECUTE replace(definicion,marca,nueva);
  EXECUTE format('ALTER FUNCTION %s SET search_path TO pg_catalog,pg_temp',f);
 END LOOP;
END $inicio_hijo$;

-- En los BEFORE INSERT de CT163/CT164 todavía no existe la fila v6. La
-- vinculación durable y el recibo del productor sustituyen solo su v5.
CREATE FUNCTION vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
 p_hijo text,p_agregado jsonb)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1
  FROM vec_contratacion_temporal.nombramiento_derivado_v1 derivada
  JOIN vec_contratacion_temporal.expediente_alta alta
    ON alta.expediente_ref=derivada.hijo_ref AND alta.derivacion_ref=derivada.derivacion_ref
  JOIN vec_contratacion_temporal.expediente_version_integral padre
    ON padre.expediente_ref=derivada.padre_ref AND padre.version=derivada.version_padre
  WHERE derivada.hijo_ref=p_hijo
    AND derivada.recibo_json->'expediente'=p_agregado
    AND p_agregado->>'referencia'=p_hijo AND p_agregado->>'version'='6'
    AND p_agregado#>>'{instruccion_compartida,recibo_derivacion_ref}'=derivada.recibo_ref
    AND p_agregado#>'{instruccion_compartida,expediente_padre}'=padre.agregado_json
    AND p_agregado->'circuito' IS NOT DISTINCT FROM padre.agregado_json->'circuito')
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(text,jsonb) FROM PUBLIC;

DO $circuito$
DECLARE f regprocedure; p pg_proc%ROWTYPE; definicion text; marca text; nueva text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.proteger_version_circuito_ct163()'::regprocedure,
  'vec_contratacion_temporal.proteger_version_circuito_ct164()'::regprocedure] LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
        (CASE WHEN f::text LIKE '%ct163%'
         THEN '4947358f5cebf637b199b783037e918e689835ebee677b36bd76f97bb6adbd37'
         ELSE 'b0d331efa4763a8ad416fcedcf96bcb942a9e6d2dd91beb56bdee80dc9982aab' END)
  THEN RAISE EXCEPTION 'CT234: preimagen del circuito divergente: %',f USING ERRCODE='55000'; END IF;
  IF f::text LIKE '%ct163%' THEN
   marca:=$m$    SELECT v.agregado_json INTO v_anterior$m$;
   nueva:=$n$    IF NEW.version=6 AND NEW.origen_version='derivacion_nombramiento_ct234' THEN
        IF vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
            NEW.expediente_ref,NEW.agregado_json) IS NOT TRUE
           OR (v_tiene_circuito AND
               vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json) IS NOT TRUE)
        THEN RAISE EXCEPTION 'CT234: circuito inicial divergente' USING ERRCODE='55000'; END IF;
        RETURN NEW;
    END IF;
    SELECT v.agregado_json INTO v_anterior$n$;
  ELSE
   marca:=$m$ IF NEW.version>1 THEN
  SELECT agregado_json INTO v_anterior$m$;
   nueva:=$n$ IF NEW.version=6 AND NEW.origen_version='derivacion_nombramiento_ct234' THEN
  IF vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
       NEW.expediente_ref,NEW.agregado_json) IS NOT TRUE
     OR vec_contratacion_temporal.circuito_agregado_valido_ct164(NEW.agregado_json) IS NOT TRUE
  THEN RAISE EXCEPTION 'CT234: terna inicial divergente' USING ERRCODE='42501'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.version>1 THEN
  SELECT agregado_json INTO v_anterior$n$;
  END IF;
  definicion:=pg_get_functiondef(f);
  IF length(definicion)-length(replace(definicion,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'CT234: marca del circuito no única: %',f USING ERRCODE='55000'; END IF;
  EXECUTE replace(definicion,marca,nueva);
  EXECUTE format('ALTER FUNCTION %s SECURITY DEFINER',f);
  EXECUTE format('ALTER FUNCTION %s SET search_path TO pg_catalog,pg_temp',f);
  EXECUTE format('ALTER FUNCTION %s SET row_security TO on',f);
 END LOOP;
END $circuito$;

-- Todo cierre de la cabecera del padre pasa por su versión actual. La
-- derivación toca esa misma fila para que una instantánea SERIALIZABLE
-- anterior al hijo aborte en vez de cerrar sobre una lectura antigua.
CREATE FUNCTION vec_contratacion_temporal.impedir_cierre_padre_con_hijos_ct234()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE destino text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR TG_OP<>'UPDATE' OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal'
    OR TG_TABLE_NAME<>'expediente_integral_actual'
 THEN RAISE EXCEPTION 'CT234: cierre de padre no autorizado' USING ERRCODE='42501'; END IF;
 SELECT v.estado INTO STRICT destino FROM vec_contratacion_temporal.expediente_version_integral v
  WHERE v.expediente_ref=NEW.expediente_ref AND v.version=NEW.version;
 IF destino IN ('completado','cancelado') THEN
  PERFORM 1 FROM vec_contratacion_temporal.nombramiento_derivado_v1 derivada
   JOIN vec_contratacion_temporal.expediente_integral_actual hijo
     ON hijo.expediente_ref=derivada.hijo_ref
   JOIN vec_contratacion_temporal.expediente_version_integral version_hijo
     ON version_hijo.expediente_ref=hijo.expediente_ref AND version_hijo.version=hijo.version
   WHERE derivada.padre_ref=NEW.expediente_ref
     AND (version_hijo.estado='en_curso' OR version_hijo.agregado_json->>'estado_actual'='en_curso')
   FOR SHARE OF hijo;
  IF FOUND THEN RAISE EXCEPTION 'CT234: padre con nombramientos abiertos' USING ERRCODE='23505'; END IF;
 END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.impedir_cierre_padre_con_hijos_ct234() FROM PUBLIC;
CREATE TRIGGER impedir_cierre_padre_con_hijos_ct234
AFTER UPDATE OF version ON vec_contratacion_temporal.expediente_integral_actual
FOR EACH ROW WHEN (NEW.version IS DISTINCT FROM OLD.version)
EXECUTE FUNCTION vec_contratacion_temporal.impedir_cierre_padre_con_hijos_ct234();

CREATE FUNCTION vec_contratacion_temporal.derivar_nombramiento_ct234_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE
 m jsonb; d jsonb; padre jsonb; hijo jsonb; acto jsonb; bolsa jsonb; recibo jsonb;
 consumo record; original vec_contratacion_temporal.nombramiento_derivado_v1%ROWTYPE;
 actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 seleccion vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 h text; contexto_h text; campo text; referencia text; aceptacion text; recibo_ref text;
 ahora timestamptz(6); instante text; numero text; prueba bytea; evento text; payload bytea;
 secuencia numeric; anterior text; outbox_h text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT234: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 262144 THEN
  RAISE EXCEPTION 'CT234: material inválido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY[
   'esquema','organizacion_ref','padre_ref','version_padre','hijo_ref','oferta_ref','numero_plaza',
   'secuencia_plaza','participacion_ref','recibo_adjudicacion_ref','clave_idempotencia','unidad_ref','padre_canonico']) IS NOT TRUE
    OR (SELECT count(*) FROM json_each(p_material::json))<>13
    OR m->>'esquema' IS DISTINCT FROM 'vec.ct.nombramiento.derivar.v1'
    OR m->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    OR jsonb_typeof(m->'version_padre') IS DISTINCT FROM 'number'
    OR m->>'version_padre' !~ '^[1-9][0-9]{0,15}$'
    OR (m->>'version_padre')::numeric NOT BETWEEN 6 AND 9007199254740991
    OR jsonb_typeof(m->'numero_plaza') IS DISTINCT FROM 'number'
    OR (m->>'numero_plaza')::numeric NOT BETWEEN 1 AND 100
    OR (m->>'numero_plaza') !~ '^[1-9][0-9]{0,2}$'
    OR jsonb_typeof(m->'secuencia_plaza') IS DISTINCT FROM 'number'
    OR m->>'secuencia_plaza' !~ '^[1-9][0-9]{0,4}$'
    OR (m->>'secuencia_plaza')::numeric NOT BETWEEN 1 AND 10000
    OR jsonb_typeof(m->'padre_canonico') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'CT234: solicitud inválida' USING ERRCODE='22023'; END IF;
 FOREACH campo IN ARRAY ARRAY['organizacion_ref','padre_ref','hijo_ref','oferta_ref',
   'participacion_ref','recibo_adjudicacion_ref','unidad_ref'] LOOP
  IF jsonb_typeof(m->campo) IS DISTINCT FROM 'string'
     OR m->>campo !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
   RAISE EXCEPTION 'CT234: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF m->>'hijo_ref' IS DISTINCT FROM 'expediente:nombramiento:'||encode(sha256(convert_to(
  (m->>'organizacion_ref')||chr(10)||(m->>'padre_ref')||chr(10)||(m->>'numero_plaza'),'UTF8')),'hex') THEN
  RAISE EXCEPTION 'CT234: identidad de hijo divergente' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(m->>'organizacion_ref')||
  '","unidad_ref":"'||(m->>'unidad_ref')||'"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.nombramiento.derivar'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'nombramiento_derivado_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'hijo_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'CT234: permiso divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM m->>'hijo_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'CT234: consumo desligado' USING ERRCODE='42501'; END IF;
 -- Solo después del consumo se lee negocio, también para recuperar el original.
 SELECT * INTO original FROM vec_contratacion_temporal.nombramiento_derivado_v1
 WHERE organizacion_ref=m->>'organizacion_ref' AND clave_idempotencia=(m->>'clave_idempotencia')::uuid;
 IF FOUND THEN
  IF original.actor_ref IS DISTINCT FROM d->>'principal_id' OR original.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
     OR original.material IS DISTINCT FROM p_material THEN
   RAISE EXCEPTION 'CT234: recuperación divergente' USING ERRCODE='23505'; END IF;
  IF original.evidencia_bolsa->>'persona_ref' IS DISTINCT FROM original.persona_ref
     OR original.evidencia_bolsa->>'version_padre' IS DISTINCT FROM original.version_padre::text
     OR original.evidencia_bolsa->>'bolsa_ref' IS NULL
     OR original.evidencia_bolsa->>'recibo_oferta_ref' IS NULL
     OR original.evidencia_bolsa->>'numero_de_plaza' IS DISTINCT FROM original.numero_plaza::text
     OR original.evidencia_bolsa->>'secuencia' IS DISTINCT FROM m->>'secuencia_plaza'
     OR original.evidencia_bolsa->>'participacion_ref' IS DISTINCT FROM m->>'participacion_ref'
     OR original.evidencia_bolsa->>'recibo_adjudicacion_ref' IS DISTINCT FROM m->>'recibo_adjudicacion_ref'
     OR original.recibo_json->>'aceptacion_ref' IS DISTINCT FROM original.aceptacion_ref
     OR original.recibo_json->>'recibo_ref' IS DISTINCT FROM original.recibo_ref
     OR original.recibo_json->>'hijo_ref' IS DISTINCT FROM original.hijo_ref
     OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral padre_original
        WHERE padre_original.expediente_ref=original.padre_ref
          AND padre_original.version=original.version_padre
          AND padre_original.agregado_json_huella_sha256=
             encode(sha256(convert_to(padre_original.agregado_json::text,'UTF8')),'hex')
          AND padre_original.agregado_json IS NOT DISTINCT FROM
             original.recibo_json#>'{expediente,instruccion_compartida,expediente_padre}')
     OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta a
    JOIN vec_contratacion_temporal.expediente_version_integral v ON v.expediente_ref=a.expediente_ref AND v.version=6
    JOIN vec_contratacion_temporal.actuacion_expediente_integral ac ON ac.expediente_ref=v.expediente_ref AND ac.secuencia=6
    WHERE a.expediente_ref=original.hijo_ref AND a.derivacion_ref=original.derivacion_ref
      AND v.agregado_json=original.recibo_json->'expediente'
      AND ac.recibo_ref=original.recibo_ref
      AND ac.actuacion_json->>'accion_clave'='contratacion_temporal.nombramiento.derivar')
     OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral ev
        WHERE ev.operacion_ref=original.derivacion_ref AND ev.expediente_ref=original.hijo_ref
          AND ev.version_expediente=6
          AND ev.tipo_evento='contratacion_temporal.nombramiento_derivado'
          AND ev.payload_huella_sha256=encode(sha256(ev.payload_canonico),'hex')
          AND (convert_from(ev.payload_canonico,'UTF8')::jsonb)->>'recibo_ref'=original.recibo_ref)
  THEN
   RAISE EXCEPTION 'CT234: original incompleto' USING ERRCODE='55000'; END IF;
  RETURN original.recibo_json||jsonb_build_object('estado','replay_confirmado');
 END IF;
 IF consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'CT234: creación sin consumo nuevo' USING ERRCODE='42501'; END IF;
 SELECT v.* INTO actual FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
 WHERE a.expediente_ref=m->>'padre_ref' FOR UPDATE OF a;
 padre:=(m->>'padre_canonico')::jsonb;
 IF actual.expediente_ref IS NULL OR actual.version<>(m->>'version_padre')::numeric
    OR actual.fase_clave<>'fiscalizacion' OR actual.estado<>'en_curso'
    OR padre ? 'instruccion_compartida'
    OR padre->>'referencia' IS DISTINCT FROM actual.expediente_ref
    OR padre->>'version' IS DISTINCT FROM actual.version::text
    OR padre->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
    OR actual.agregado_json->>'referencia' IS DISTINCT FROM actual.expediente_ref
    OR actual.agregado_json->>'version' IS DISTINCT FROM actual.version::text
    OR actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
    OR actual.agregado_json->>'fase_actual' IS DISTINCT FROM actual.fase_clave
    OR actual.agregado_json->>'estado_actual' IS DISTINCT FROM actual.estado
    OR padre->>'numero_visible' IS DISTINCT FROM actual.agregado_json->>'numero_visible'
    OR padre->'flujo' IS DISTINCT FROM actual.agregado_json->'flujo'
    OR padre->>'fase_actual' IS DISTINCT FROM actual.fase_clave
    OR padre->>'estado_actual' IS DISTINCT FROM actual.estado
    OR padre#>>'{asignacion,unidad_ref}' IS DISTINCT FROM m->>'unidad_ref'
    OR padre#>>'{asignacion,unidad_ref}' IS DISTINCT FROM actual.agregado_json#>>'{asignacion,unidad_ref}'
    OR coalesce(padre#>>'{fiscalizacion,resultado}','') NOT IN ('favorable','favorable_con_observaciones')
    OR padre#>>'{fiscalizacion,resultado}' IS DISTINCT FROM actual.agregado_json#>>'{fiscalizacion,resultado}'
    OR padre#>>'{solicitud,categoria_ref}' IS DISTINCT FROM actual.agregado_json#>>'{solicitud,categoria_ref}'
    OR (padre#>>'{solicitud,numero_personas}')::integer NOT BETWEEN 2 AND 100
    OR padre#>>'{solicitud,numero_personas}' IS DISTINCT FROM actual.agregado_json#>>'{solicitud,numero_personas}'
    OR (padre#>>'{solicitud,periodo,inicio}')::timestamptz IS DISTINCT FROM
       (actual.agregado_json#>>'{solicitud,periodo,inicio}')::timestamptz
    OR (padre#>>'{solicitud,periodo,fin}')::timestamptz IS DISTINCT FROM
       (actual.agregado_json#>>'{solicitud,periodo,fin}')::timestamptz
    OR padre#>>'{solicitud,periodo,causa_fin}' IS DISTINCT FROM
       actual.agregado_json#>>'{solicitud,periodo,causa_fin}'
    OR padre#>'{solicitud,periodo,politica_fin}' IS DISTINCT FROM
       actual.agregado_json#>'{solicitud,periodo,politica_fin}'
    OR (padre->>'creado_en')::timestamptz IS DISTINCT FROM
       (actual.agregado_json->>'creado_en')::timestamptz
    OR (padre->>'actualizado_en')::timestamptz IS DISTINCT FROM
       (actual.agregado_json->>'actualizado_en')::timestamptz
    OR actual.agregado_json_huella_sha256 IS DISTINCT FROM
       encode(sha256(convert_to(actual.agregado_json::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'CT234: padre no derivable' USING ERRCODE='23505'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.nombramiento_derivado_v1
  WHERE organizacion_ref=m->>'organizacion_ref' AND padre_ref=m->>'padre_ref' AND numero_plaza=(m->>'numero_plaza')::integer) THEN
  RAISE EXCEPTION 'CT234: plaza ya derivada' USING ERRCODE='23505'; END IF;
 -- Bump MVCC sin cambiar el valor publicado: un cierre iniciado antes de
 -- esta derivación no puede confirmar una instantánea sin el nuevo hijo.
 UPDATE vec_contratacion_temporal.expediente_integral_actual
 SET actualizada_en=actualizada_en
 WHERE expediente_ref=m->>'padre_ref' AND version=actual.version;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT234: cabecera del padre cambió' USING ERRCODE='40001'; END IF;
 SELECT * INTO seleccion FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE solicitud_json->>'expediente_ref'=m->>'padre_ref' AND tipo_seleccion='oferta' AND situacion='confirmada';
 IF seleccion.clave_idempotencia IS NULL
    OR vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(seleccion) IS NOT TRUE
    OR seleccion.recibo_json->>'oferta_ref' IS DISTINCT FROM m->>'oferta_ref' THEN
  RAISE EXCEPTION 'CT234: oferta no vinculada' USING ERRCODE='23505'; END IF;
 bolsa:=vec_bolsa_llamamientos.verificar_plaza_adjudicada_ct_v1(m->>'organizacion_ref',m->>'padre_ref',m->>'oferta_ref',
  (m->>'numero_plaza')::integer,(m->>'secuencia_plaza')::integer,m->>'participacion_ref',m->>'recibo_adjudicacion_ref');
 IF (bolsa->>'version_padre')::numeric IS DISTINCT FROM actual.version THEN
  RAISE EXCEPTION 'CT234: adjudicación de otra instrucción' USING ERRCODE='23505'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF ahora<(actual.agregado_json->>'actualizado_en')::timestamptz OR ahora<(bolsa->>'adjudicada_en')::timestamptz THEN
  RAISE EXCEPTION 'CT234: reloj anterior al antecedente' USING ERRCODE='55000'; END IF;
 instante:=to_char(ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 referencia:='derivacion:'||gen_random_uuid()::text;
 aceptacion:='aceptacion-plaza:'||gen_random_uuid()::text;
 recibo_ref:='recibo:'||gen_random_uuid()::text;
 numero:=(actual.agregado_json->>'numero_visible')||'-P'||(m->>'numero_plaza');
 acto:=jsonb_build_object('secuencia',6,'version_expediente',6,'accion_clave','contratacion_temporal.nombramiento.derivar',
  'actor_ref',d->>'principal_id','unidad_ref',m->>'unidad_ref','recibo_ref',recibo_ref,'realizada_en',instante,
  'fase_origen','fiscalizacion','estado_origen','en_curso','fase_destino','fiscalizacion','estado_destino','en_curso');
 hijo:=actual.agregado_json||jsonb_build_object('referencia',m->>'hijo_ref','numero_visible',numero,'version',6,
  'creado_en',instante,'actualizado_en',instante,'actuaciones',jsonb_build_array(acto),
  'solicitud',(actual.agregado_json->'solicitud')||jsonb_build_object('numero_personas',1),
  'instruccion_compartida',jsonb_build_object('expediente_padre',actual.agregado_json,'oferta_ref',m->>'oferta_ref',
   'numero_plaza',(m->>'numero_plaza')::integer,'recibo_derivacion_ref',recibo_ref,
   'huella_padre_sha256',encode(sha256(convert_to(m->>'padre_canonico','UTF8')),'hex')));
 recibo:=jsonb_build_object('esquema','vec.ct.nombramiento.derivado.v1','estado','confirmado',
  'padre_ref',m->>'padre_ref','hijo_ref',m->>'hijo_ref','numero_visible',numero,'numero_plaza',(m->>'numero_plaza')::integer,
  'total_personas',(padre#>>'{solicitud,numero_personas}')::integer,'version',6,'aceptacion_ref',aceptacion,
  'recibo_ref',recibo_ref,'auditoria_ref',consumo.auditoria_ref,'confirmada_en',instante,'expediente',hijo);
 INSERT INTO vec_contratacion_temporal.nombramiento_derivado_v1 VALUES(
  referencia,m->>'organizacion_ref',m->>'padre_ref',actual.version,m->>'hijo_ref',m->>'oferta_ref',(m->>'numero_plaza')::integer,
  bolsa->>'persona_ref',aceptacion,bolsa,(m->>'clave_idempotencia')::uuid,d->>'principal_id',d->>'perfil_activo_ref',
  p_material,h,consumo.decision_ref,consumo.auditoria_ref,recibo_ref,recibo,ahora);
 INSERT INTO vec_contratacion_temporal.expediente_alta(expediente_ref,reserva_ref,numero_visible,organizacion_ref,actor_ref,perfil_ref,
  decision_ref,efecto_ref,huella_efecto_sha256,creada_en,derivacion_ref) VALUES(m->>'hijo_ref',NULL,numero,m->>'organizacion_ref',
  d->>'principal_id',d->>'perfil_activo_ref',consumo.decision_ref,m->>'hijo_ref',contexto_h,ahora,referencia);
 prueba:=convert_to('VEC-CT-DERIVACION-1'||chr(10)||(m->>'hijo_ref')||chr(10)||referencia||chr(10)||h||chr(10)||recibo_ref||chr(10)||instante,'UTF8');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,agregado_json,agregado_json_huella_sha256,
  prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
 VALUES(m->>'hijo_ref',6,hijo,
  encode(sha256(convert_to(hijo::text,'UTF8')),'hex'),prueba,encode(sha256(prueba),'hex'),
  actual.flujo_ref,actual.flujo_version,actual.flujo_huella_sha256,'fiscalizacion','en_curso','derivacion_nombramiento_ct234',referencia,ahora);
 INSERT INTO vec_contratacion_temporal.expediente_integral_actual VALUES(m->>'hijo_ref',6,ahora,referencia);
 INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral VALUES(m->>'hijo_ref',6,6,referencia,recibo_ref,acto,
  encode(sha256(convert_to(acto::text,'UTF8')),'hex'),prueba,encode(sha256(prueba),'hex'),ahora);
 SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT secuencia,anterior
 FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
 IF secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT234: límite de outbox' USING ERRCODE='55000'; END IF;
 evento:='evento:'||gen_random_uuid()::text;
 payload:=convert_to(jsonb_build_object('esquema','vec.ct.nombramiento.derivado.v1','padre_ref',m->>'padre_ref',
  'hijo_ref',m->>'hijo_ref','numero_plaza',(m->>'numero_plaza')::integer,'recibo_ref',recibo_ref)::text,'UTF8');
 outbox_h:=encode(sha256(anterior::bytea||payload),'hex');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(evento_ref,secuencia,operacion_ref,expediente_ref,
  version_expediente,tipo_evento,payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
 VALUES(evento,secuencia+1,referencia,m->>'hijo_ref',6,
  'contratacion_temporal.nombramiento_derivado',payload,encode(sha256(payload),'hex'),anterior,outbox_h,ahora);
 UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral SET secuencia_outbox=secuencia+1,
  cabeza_outbox_sha256=outbox_h,actualizada_en=ahora WHERE control_id;
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
DO $post$
DECLARE f oid:=to_regprocedure(
 'vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class c
      WHERE c.oid='vec_contratacion_temporal.nombramiento_derivado_v1'::regclass
        AND c.relowner='vec_contratacion_temporal_propietario'::regrole
        AND c.relrowsecurity AND c.relforcerowsecurity)
    OR (SELECT count(*) FROM pg_trigger t
        WHERE t.tgrelid='vec_contratacion_temporal.nombramiento_derivado_v1'::regclass
          AND NOT t.tgisinternal AND t.tgenabled='O')<>2
    OR NOT EXISTS(SELECT 1 FROM pg_proc p
        WHERE p.oid=f AND p.proowner='vec_contratacion_temporal_propietario'::regrole
          AND p.prosecdef AND p.provolatile='v'
          AND EXISTS(SELECT 1 FROM unnest(p.proconfig) cfg
             WHERE replace(cfg,' ','')='search_path=pg_catalog,pg_temp'))
    OR EXISTS(SELECT 1 FROM pg_proc p
        CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=f AND a.privilege_type='EXECUTE'
          AND a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_ejecutor'::regrole))
 THEN RAISE EXCEPTION 'CT234: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
