\set ON_ERROR_STOP on
-- B77: declaración documental propia y resolución RRHH de solo adición.
-- La referencia y SHA del documento no acreditan que VEC custodie sus bytes.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000077',0));

DO $precondicion$
DECLARE actual jsonb; esperado jsonb:=pg_catalog.jsonb_build_object(
 'rol',true,'ad155',true,'b76',true,'b61',true,'tabla_libre',true,'avisos',true);
BEGIN
 actual:=pg_catalog.jsonb_build_object(
  'rol',current_user='vec_bolsa_llamamientos_propietario',
  'ad155',pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
  'b76',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(text,text,text,timestamp with time zone,timestamp with time zone,text,text,text,text,timestamp with time zone,text,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
  'b61',pg_catalog.to_regclass('vec_bolsa_llamamientos.denegacion_frontera_portal_externo') IS NOT NULL
    AND pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)') IS NOT NULL,
  'tabla_libre',pg_catalog.to_regclass('vec_bolsa_llamamientos.solicitud_documental_rrhh') IS NULL,
  'avisos',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(timestamp with time zone)') IS NOT NULL);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'PARO clave=B77.preimagen, actual=%, esperado=%',actual,esperado USING ERRCODE='55000';
 END IF;
END $precondicion$;

-- La ruta documental debe auditar una denegación prePDP con el registrador
-- exterior B61. Se amplían exactamente su CHECK y su guarda, sin tocar RLS,
-- ACL ni otras rutas ya instaladas.
DO $frontera_b61$
DECLARE
 ruta_nueva text:='/api/vec/bolsa/mi-bolsa/solicitudes-documentales';
 tabla pg_catalog.regclass:='vec_bolsa_llamamientos.denegacion_frontera_portal_externo'::pg_catalog.regclass;
 nombre text; anterior_expr text; actual_expr text; v_total integer;
 firma pg_catalog.regprocedure:='vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
 anterior text; nuevo text; actual text; marca text:='''/api/vec/bolsa/mi-bolsa/contacto''';
 acl pg_catalog.aclitem[]; propietario oid; config text[]; definidora boolean;
BEGIN
 SELECT pg_catalog.count(*),pg_catalog.min(c.conname),pg_catalog.min(pg_catalog.pg_get_expr(c.conbin,c.conrelid))
 INTO v_total,nombre,anterior_expr FROM pg_catalog.pg_constraint c
 WHERE c.conrelid=tabla AND c.contype='c' AND c.convalidated
   AND pg_catalog.pg_get_constraintdef(c.oid) LIKE '%/api/vec/bolsa/mi-bolsa/contacto%';
 IF v_total<>1 OR anterior_expr IS NULL OR pg_catalog.strpos(anterior_expr,ruta_nueva)<>0 THEN
  RAISE EXCEPTION 'PARO clave=B61.ruta_CHECK, actual=%/%, esperado=1/false',
   v_total,coalesce(pg_catalog.strpos(anterior_expr,ruta_nueva)<>0,false) USING ERRCODE='55000'; END IF;
 EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo DROP CONSTRAINT %I',nombre);
 EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo ADD CONSTRAINT %I CHECK ((%s) OR ruta=%L)',
  nombre,anterior_expr,ruta_nueva);
 SELECT pg_catalog.pg_get_expr(c.conbin,c.conrelid) INTO STRICT actual_expr FROM pg_catalog.pg_constraint c
  WHERE c.conrelid=tabla AND c.conname=nombre AND c.convalidated;
 IF pg_catalog.strpos(actual_expr,ruta_nueva)=0 OR
    pg_catalog.strpos(actual_expr,'/api/vec/bolsa/mi-bolsa/contacto')=0 OR
    NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity) THEN
  RAISE EXCEPTION 'PARO clave=B61.ruta_y_RLS, actual=%/%, esperado=true/true',
   pg_catalog.strpos(actual_expr,ruta_nueva)<>0,
   EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity)
   USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.pg_get_functiondef(firma),p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT anterior,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=firma;
 IF pg_catalog.length(anterior)-pg_catalog.length(pg_catalog.replace(anterior,marca,''))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(anterior,ruta_nueva)<>0 OR NOT definidora THEN
  RAISE EXCEPTION 'PARO clave=B61.guarda_ruta, actual=%/%, esperado=1/false',
   (pg_catalog.length(anterior)-pg_catalog.length(pg_catalog.replace(anterior,marca,'')))/pg_catalog.length(marca),
   pg_catalog.strpos(anterior,ruta_nueva)<>0 USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(anterior,marca,marca||','||pg_catalog.quote_literal(ruta_nueva));
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(firma) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR pg_catalog.replace(actual,marca||','||pg_catalog.quote_literal(ruta_nueva),marca)
    IS DISTINCT FROM anterior OR
    (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM acl OR
    (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM propietario OR
    (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM config OR
    (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM definidora THEN
  RAISE EXCEPTION 'PARO clave=B61.definicion_y_ACL, actual=%, esperado=%',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual||coalesce((SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=firma),''),'UTF8')),'hex'),
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nuevo||coalesce(acl::text,''),'UTF8')),'hex')
   USING ERRCODE='55000'; END IF;
END $frontera_b61$;

-- La referencia del documento se conserva como identificador opaco. Misma
-- exclusión de etiquetas/DNI/NIE que JustificanteOperacionSituacion.Validar:
-- solo las referencias propias `espacio:sha256` eximen coincidencias numéricas
-- fortuitas dentro de una huella, nunca una etiqueta de identidad.
CREATE FUNCTION vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(p_ref text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT p_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'
  AND p_ref !~* '(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)'
  AND (p_ref ~ '^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$'
       OR p_ref !~* '(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]')
$f$;

CREATE TABLE vec_bolsa_llamamientos.solicitud_documental_rrhh (
 solicitud_ref text PRIMARY KEY CHECK (solicitud_ref ~ '^solicitud-documental:[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:solicitud-documental:[0-9a-f]{64}$'),
 contenido_sha256 text NOT NULL CHECK (contenido_sha256 ~ '^[0-9a-f]{64}$'),
 bolsa_ref text NOT NULL CHECK (pg_catalog.octet_length(bolsa_ref) BETWEEN 3 AND 512),
 participacion_ref text NOT NULL CHECK (pg_catalog.octet_length(participacion_ref) BETWEEN 3 AND 512),
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 tipo text NOT NULL CHECK (tipo='documental_rrhh'),
 documento_ref text NOT NULL CHECK (vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(documento_ref)),
 documento_sha256 text NOT NULL CHECK (documento_sha256 ~ '^[0-9a-f]{64}$'),
 fecha_fin_causa date CHECK (fecha_fin_causa IS NULL OR pg_catalog.isfinite(fecha_fin_causa)),
 version integer NOT NULL CHECK (version=1),
 estado_origen text NOT NULL CHECK (estado_origen='pendiente_rrhh'),
 clave_idempotencia text NOT NULL CHECK (pg_catalog.octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia=pg_catalog.btrim(clave_idempotencia)),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(registrada_en)),
 UNIQUE(participacion_ref,clave_idempotencia)
);
CREATE INDEX solicitud_documental_rrhh_pendiente_idx
 ON vec_bolsa_llamamientos.solicitud_documental_rrhh(participacion_ref,registrada_en DESC,solicitud_ref);

CREATE TABLE vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh (
 solicitud_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.solicitud_documental_rrhh(solicitud_ref),
 version_esperada integer NOT NULL CHECK (version_esperada=1),
 contenido_sha256 text NOT NULL CHECK (contenido_sha256 ~ '^[0-9a-f]{64}$'),
 resultado text NOT NULL CHECK (resultado IN ('validada','rechazada')),
 motivo text NOT NULL CHECK (pg_catalog.octet_length(motivo) BETWEEN 1 AND 1000 AND motivo=pg_catalog.btrim(motivo)),
 actor_ref text NOT NULL CHECK (pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 256),
 clave_idempotencia text NOT NULL CHECK (pg_catalog.octet_length(clave_idempotencia) BETWEEN 8 AND 256),
 recibo_ref text NOT NULL UNIQUE CHECK (pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 256),
 recibo_b8_ref text UNIQUE,
 situacion_desde timestamptz(6),
 decision_ref text UNIQUE,
 auditoria_ref text,
 resuelta_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(resuelta_en)),
 CHECK ((resultado='validada')=(recibo_b8_ref IS NOT NULL AND situacion_desde IS NOT NULL)),
 CHECK ((resultado='rechazada')=(decision_ref IS NOT NULL AND auditoria_ref IS NOT NULL))
);

DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['solicitud_documental_rrhh','resolucion_solicitud_documental_rrhh'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')',t||'_propietario',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t||'_inmutable',t);
 END LOOP;
END $tablas$;

-- Canon de las tres referencias; un separador después de cada pieza iguala
-- la huella del servidor Go y evita ambigüedad de concatenación.
CREATE FUNCTION vec_bolsa_llamamientos.b77_huella_partes_v1(VARIADIC p_partes text[])
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
DECLARE parte text; material bytea:=''::bytea;
BEGIN
 IF pg_catalog.cardinality(p_partes)=0 OR pg_catalog.cardinality(p_partes)>16 THEN RETURN NULL; END IF;
 FOREACH parte IN ARRAY p_partes LOOP
  IF parte IS NULL OR pg_catalog.octet_length(parte)>512
     OR pg_catalog.strpos(parte,pg_catalog.chr(31))<>0 THEN RETURN NULL; END IF;
  material:=material||pg_catalog.convert_to(parte,'UTF8')||pg_catalog.decode('1f','hex');
 END LOOP;
 RETURN pg_catalog.encode(pg_catalog.sha256(material),'hex');
END $f$;

-- B76 pública sigue sirviendo actos manuales y replays anteriores. Si hay
-- solicitud documental pendiente, la regularización debe usar esta fachada
-- B77 con CAS y cierre; citar el nuevo prefijo por v2 queda denegado.
DO $guardia_b76$
DECLARE firma pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(text,text,text,timestamp with time zone,timestamp with time zone,text,text,text,text,timestamp with time zone,text,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 original text; nuevo text; actual text; acl pg_catalog.aclitem[]; propietario oid; config text[];
 marca text := ' SELECT * INTO STRICT v_cambio FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76(';
 insercion text := $g$ IF p_justificante_ref LIKE 'solicitud-documental:%'
    OR (p_operacion='regularizar' AND v_previa.participacion_ref IS NULL
      AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.solicitud_documental_rrhh sd
       WHERE sd.participacion_ref=p_participacion_ref AND sd.bolsa_ref=p_bolsa_ref
         AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh rd
          WHERE rd.solicitud_ref=sd.solicitud_ref))) THEN
  RAISE EXCEPTION 'regularización documental exige CAS' USING ERRCODE='42501'; END IF;
$g$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(firma),p.proacl,p.proowner,p.proconfig
 INTO STRICT original,acl,propietario,config FROM pg_catalog.pg_proc p WHERE p.oid=firma;
 IF pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(original,'regularización documental exige CAS')<>0 THEN
  RAISE EXCEPTION 'PARO clave=B76.marca, actual=%/%, esperado=1/false',
   (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,'')))/pg_catalog.length(marca),
   pg_catalog.strpos(original,'regularización documental exige CAS')<>0 USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,marca,insercion||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(firma) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR pg_catalog.replace(actual,insercion||marca,marca) IS DISTINCT FROM original
    OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM config THEN
  RAISE EXCEPTION 'PARO clave=B76.definicion_y_ACL, actual=%, esperado=%',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual||coalesce((SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=firma),'')||
    (SELECT proowner::text FROM pg_catalog.pg_proc WHERE oid=firma)||coalesce((SELECT proconfig::text FROM pg_catalog.pg_proc WHERE oid=firma),''),'UTF8')),'hex'),
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nuevo||coalesce(acl::text,'')||propietario::text||coalesce(config::text,''),'UTF8')),'hex')
   USING ERRCODE='55000'; END IF;
END $guardia_b76$;

-- RRHH17 retira nuevas pausas y reactivaciones del integrante. La función
-- antigua conserva su replay tras autorización vigente; el efecto nuevo se
-- detiene justo después de esa rama, sin tocar historia B30.
DO $guardia_b30$
DECLARE firma pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 original text; nuevo text; actual text; acl pg_catalog.aclitem[]; propietario oid; config text[];
 marca text := ' RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(';
 insercion text := $g$ RAISE EXCEPTION 'solicitud histórica sin efecto nuevo' USING ERRCODE='22023';
$g$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(firma),p.proacl,p.proowner,p.proconfig
 INTO STRICT original,acl,propietario,config FROM pg_catalog.pg_proc p WHERE p.oid=firma;
 IF pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(original,'solicitud histórica sin efecto nuevo')<>0 THEN
  RAISE EXCEPTION 'PARO clave=B30.marca, actual=%/%, esperado=1/false',
   (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,'')))/pg_catalog.length(marca),
   pg_catalog.strpos(original,'solicitud histórica sin efecto nuevo')<>0 USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,marca,insercion||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(firma) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR pg_catalog.replace(actual,insercion||marca,marca) IS DISTINCT FROM original
    OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=firma) IS DISTINCT FROM config THEN
  RAISE EXCEPTION 'PARO clave=B30.definicion_y_ACL, actual=%, esperado=%',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual||coalesce((SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=firma),'')||
    (SELECT proowner::text FROM pg_catalog.pg_proc WHERE oid=firma)||coalesce((SELECT proconfig::text FROM pg_catalog.pg_proc WHERE oid=firma),''),'UTF8')),'hex'),
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nuevo||coalesce(acl::text,'')||propietario::text||coalesce(config::text,''),'UTF8')),'hex')
   USING ERRCODE='55000'; END IF;
END $guardia_b30$;

-- El acto B76 consume la decisión V3 y registra la situación. Esta fachada
-- añade el CAS de solicitud, coteja el documento y cierra en la misma TX.
CREATE FUNCTION vec_bolsa_llamamientos.regularizar_solicitud_documental_rrhh_v1(
 p_solicitud_ref text,p_version_esperada integer,p_contenido_sha256 text,p_bolsa_ref text,p_participacion_ref text,
 p_desde timestamptz,p_situacion_esperada_desde timestamptz,p_motivo text,p_actor text,p_clave_idempotencia text,
 p_recibo_ref text,p_registrada_en timestamptz,p_validador text,p_validada_en timestamptz,
 p_documento_ref text,p_documento_sha256 text,p_fecha_fin_causa date,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_contexto_recurso bytea)
RETURNS TABLE(reutilizada boolean,recibo_ref text,situacion text,desde timestamptz,
 fecha_disponible timestamptz,recibo_resolucion_ref text,resuelta_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE solicitud vec_bolsa_llamamientos.solicitud_documental_rrhh%ROWTYPE;
 resolucion vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh%ROWTYPE;
 cambio record; operacion record; recibo_resolucion text; decision jsonb; recurso jsonb;
 huella_comando text; cas_micro text; causa_en timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('transaction_isolation')<>'serializable'
    OR p_version_esperada IS DISTINCT FROM 1 OR p_contenido_sha256 !~ '^[0-9a-f]{64}$'
    OR p_solicitud_ref !~ '^solicitud-documental:[0-9a-f]{64}$'
    OR p_documento_ref IS NULL OR p_documento_sha256 IS NULL OR p_documento_sha256 !~ '^[0-9a-f]{64}$'
    OR p_fecha_fin_causa IS NULL OR NOT pg_catalog.isfinite(p_fecha_fin_causa)
    OR p_desde IS NULL OR NOT pg_catalog.isfinite(p_desde)
    OR p_validada_en IS NULL OR NOT pg_catalog.isfinite(p_validada_en)
    OR p_registrada_en IS NULL OR NOT pg_catalog.isfinite(p_registrada_en)
    OR p_desde>p_registrada_en OR p_validada_en>p_registrada_en
    OR p_validador IS NULL OR pg_catalog.octet_length(p_validador) NOT BETWEEN 1 AND 256
    OR p_validador<>pg_catalog.btrim(p_validador)
    OR p_fecha_fin_causa>(p_validada_en AT TIME ZONE 'Europe/Madrid')::date
    OR p_fecha_fin_causa>(p_desde AT TIME ZONE 'Europe/Madrid')::date THEN
  RAISE EXCEPTION 'regularización documental inválida' USING ERRCODE='22023';
 END IF;
 causa_en:=p_fecha_fin_causa::timestamp AT TIME ZONE 'Europe/Madrid';
 IF causa_en>p_validada_en OR causa_en>p_desde THEN
  RAISE EXCEPTION 'fin de causa posterior al acto' USING ERRCODE='22023'; END IF;
 -- Se presenta la preimagen exacta del recurso firmado por V3. La decisión
 -- publica su huella, no los atributos; estos no se recuperan del cliente.
 BEGIN
  decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  recurso:=pg_catalog.convert_from(p_contexto_recurso,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'contexto documental no autorizado' USING ERRCODE='42501'; END;
 IF p_contexto_recurso IS NULL OR pg_catalog.octet_length(p_contexto_recurso)>4096
    OR pg_catalog.jsonb_typeof(recurso) IS DISTINCT FROM 'object'
    OR (recurso-ARRAY['ambitos','atributos'])<>'{}'::jsonb
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(recurso))<>2
    OR pg_catalog.jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
    OR pg_catalog.jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
    OR ((recurso->'ambitos')-ARRAY['ambito_ref','unidad_ref'])<>'{}'::jsonb
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(recurso->'ambitos'))<>2
    OR pg_catalog.jsonb_typeof(recurso#>'{ambitos,ambito_ref}') IS DISTINCT FROM 'string'
    OR pg_catalog.jsonb_typeof(recurso#>'{ambitos,unidad_ref}') IS DISTINCT FROM 'string'
    OR ((recurso->'atributos')-'regularizacion_documental_sha256')<>'{}'::jsonb
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(recurso->'atributos'))<>1
    OR pg_catalog.jsonb_typeof(recurso#>'{atributos,regularizacion_documental_sha256}') IS DISTINCT FROM 'string'
    OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM
       pg_catalog.encode(pg_catalog.sha256(p_contexto_recurso),'hex') THEN
  RAISE EXCEPTION 'contexto documental no autorizado' USING ERRCODE='42501'; END IF;
 IF p_situacion_esperada_desde IS NULL OR NOT pg_catalog.isfinite(p_situacion_esperada_desde) THEN
  RAISE EXCEPTION 'CAS documental inválido' USING ERRCODE='22023'; END IF;
 cas_micro:=((EXTRACT(EPOCH FROM p_situacion_esperada_desde)*1000000)::bigint)::text;
 huella_comando:=vec_bolsa_llamamientos.b77_huella_partes_v1(
  'regularizacion-documental-v1',p_solicitud_ref,p_version_esperada::text,p_contenido_sha256,
  p_bolsa_ref,p_participacion_ref,p_documento_ref,p_documento_sha256,
  pg_catalog.to_char(p_fecha_fin_causa,'YYYY-MM-DD'),cas_micro,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_motivo,'UTF8')),'hex'),
  p_actor,p_validador,p_clave_idempotencia,p_recibo_ref);
 IF huella_comando IS NULL THEN RAISE EXCEPTION 'canon documental inválido' USING ERRCODE='22023'; END IF;
 IF recurso#>>'{atributos,regularizacion_documental_sha256}' IS DISTINCT FROM huella_comando THEN
  RAISE EXCEPTION 'comando documental distinto del recurso autorizado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:documental:'||p_participacion_ref,0));
 SELECT * INTO STRICT cambio FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76(
  p_bolsa_ref,p_participacion_ref,'disponible',p_desde,NULL,p_motivo,p_actor,p_clave_idempotencia,
  p_recibo_ref,p_registrada_en,p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,
  p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,true,p_situacion_esperada_desde,
  'regularizar',causa_en);
 SELECT * INTO solicitud FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
  WHERE s.solicitud_ref=p_solicitud_ref FOR UPDATE;
 IF solicitud.solicitud_ref IS NULL OR solicitud.version IS DISTINCT FROM p_version_esperada
    OR solicitud.contenido_sha256 IS DISTINCT FROM p_contenido_sha256
    OR solicitud.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR solicitud.participacion_ref IS DISTINCT FROM p_participacion_ref
    OR solicitud.documento_ref IS DISTINCT FROM p_documento_ref OR solicitud.documento_sha256 IS DISTINCT FROM p_documento_sha256
    OR (solicitud.fecha_fin_causa IS NOT NULL
        AND solicitud.fecha_fin_causa IS DISTINCT FROM p_fecha_fin_causa) THEN
  RAISE EXCEPTION 'solicitud documental ajena o versión distinta' USING ERRCODE='VBS02';
 END IF;
 SELECT * INTO resolucion FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r
  WHERE r.solicitud_ref=p_solicitud_ref;
 IF cambio.reutilizada THEN
  SELECT * INTO operacion FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   WHERE o.participacion_ref=p_participacion_ref AND o.clave_idempotencia=p_clave_idempotencia;
  IF resolucion.solicitud_ref IS NULL OR resolucion.resultado<>'validada'
     OR resolucion.contenido_sha256 IS DISTINCT FROM p_contenido_sha256
     OR resolucion.actor_ref IS DISTINCT FROM p_actor OR resolucion.clave_idempotencia IS DISTINCT FROM p_clave_idempotencia
     OR resolucion.recibo_b8_ref IS DISTINCT FROM cambio.recibo_ref
     OR operacion.justificante_ref IS DISTINCT FROM p_solicitud_ref
     OR operacion.justificante_sha256 IS DISTINCT FROM p_documento_sha256
     OR operacion.validador IS DISTINCT FROM p_validador
     OR operacion.situacion_esperada_desde IS DISTINCT FROM p_situacion_esperada_desde
     OR operacion.causa_finalizada_en IS DISTINCT FROM causa_en THEN
   RAISE EXCEPTION 'replay documental distinto' USING ERRCODE='VBS01';
  END IF;
  RETURN QUERY SELECT true,cambio.recibo_ref,cambio.situacion,cambio.desde,cambio.fecha_disponible,
    resolucion.recibo_ref,resolucion.resuelta_en;
  RETURN;
 END IF;
 IF resolucion.solicitud_ref IS NOT NULL THEN
  RAISE EXCEPTION 'solicitud documental ya resuelta' USING ERRCODE='VBS02'; END IF;
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
  participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,
  actor,validador,validada_en,registrada_en,clave_idempotencia,situacion_esperada_desde,causa_finalizada_en)
 VALUES(p_participacion_ref,cambio.desde,'regularizar','solicitud_candidato',p_solicitud_ref,p_documento_sha256,
  p_actor,p_validador,p_validada_en,p_registrada_en,p_clave_idempotencia,p_situacion_esperada_desde,
  causa_en);
 recibo_resolucion:='recibo:solicitud-documental-resolucion:'||
  vec_bolsa_llamamientos.b77_huella_partes_v1('validada',p_solicitud_ref,cambio.recibo_ref);
 INSERT INTO vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh(
  solicitud_ref,version_esperada,contenido_sha256,resultado,motivo,actor_ref,clave_idempotencia,
  recibo_ref,recibo_b8_ref,situacion_desde,resuelta_en)
 VALUES(p_solicitud_ref,p_version_esperada,p_contenido_sha256,'validada',p_motivo,p_actor,p_clave_idempotencia,
  recibo_resolucion,cambio.recibo_ref,cambio.desde,p_registrada_en);
 RETURN QUERY SELECT false,cambio.recibo_ref,cambio.situacion,cambio.desde,cambio.fecha_disponible,
   recibo_resolucion,p_registrada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.solicitar_documental_portal_v1(
 p_solicitud_ref text,p_recibo_ref text,p_contenido_sha256 text,p_candidato_ref text,p_bolsa_ref text,
 p_documento_ref text,p_documento_sha256 text,p_fecha_fin_causa date,p_clave text,p_registrada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,solicitud_ref text,recibo_ref text,contenido_sha256 text,
 registrada_en timestamptz,version integer,estado text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE participacion text; consumo record; previa vec_bolsa_llamamientos.solicitud_documental_rrhh%ROWTYPE;
 referencia_huella text; decision jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('transaction_isolation')<>'serializable'
    OR p_solicitud_ref IS NULL OR p_recibo_ref IS NULL OR p_contenido_sha256 IS NULL
    OR vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(p_documento_ref) IS NOT TRUE
    OR p_documento_sha256 IS NULL OR p_documento_sha256 !~ '^[0-9a-f]{64}$'
    OR p_clave IS NULL OR pg_catalog.octet_length(p_clave) NOT BETWEEN 8 AND 256 OR p_clave<>pg_catalog.btrim(p_clave)
    OR (p_fecha_fin_causa IS NOT NULL AND NOT pg_catalog.isfinite(p_fecha_fin_causa))
    OR p_registrada_en IS NULL OR NOT pg_catalog.isfinite(p_registrada_en)
    OR (p_fecha_fin_causa IS NOT NULL
        AND p_fecha_fin_causa>(p_registrada_en AT TIME ZONE 'Europe/Madrid')::date) THEN
  RAISE EXCEPTION 'solicitud documental inválida' USING ERRCODE='22023';
 END IF;
 referencia_huella:=vec_bolsa_llamamientos.b77_huella_partes_v1('solicitud-documental',p_candidato_ref,p_bolsa_ref,p_clave);
 IF p_solicitud_ref IS DISTINCT FROM 'solicitud-documental:'||referencia_huella
    OR p_recibo_ref IS DISTINCT FROM 'recibo:solicitud-documental:'||vec_bolsa_llamamientos.b77_huella_partes_v1('recibo',referencia_huella)
    OR p_contenido_sha256 IS DISTINCT FROM vec_bolsa_llamamientos.b77_huella_partes_v1(
      'contenido-solicitud-documental',p_candidato_ref,p_bolsa_ref,p_documento_ref,p_documento_sha256,
      coalesce(pg_catalog.to_char(p_fecha_fin_causa,'YYYY-MM-DD'),'')) THEN
  RAISE EXCEPTION 'huella de solicitud documental distinta' USING ERRCODE='22023';
 END IF;
 participacion:=vec_bolsa_llamamientos.exigir_portal_candidato_v1(
  p_candidato_ref,p_bolsa_ref,'bolsa.participaciones_propias.presentar_solicitud_documental',
  p_capacidad,p_decision,p_contexto);
 BEGIN decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'decisión documental inválida' USING ERRCODE='42501'; END;
 IF decision->>'principal_id' IS NULL OR decision->>'accion' IS DISTINCT FROM 'bolsa.participaciones_propias.presentar_solicitud_documental'
    OR decision->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'solicitud documental no autorizada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(
  'bolsa.participaciones_propias.presentar_solicitud_documental',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref OR consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'solicitud documental no autorizada' USING ERRCODE='42501'; END IF;
 IF consumo.consumida_en IS NULL OR NOT pg_catalog.isfinite(consumo.consumida_en)
    OR (p_fecha_fin_causa IS NOT NULL
        AND p_fecha_fin_causa>(consumo.consumida_en AT TIME ZONE 'Europe/Madrid')::date) THEN
  RAISE EXCEPTION 'fecha civil documental futura' USING ERRCODE='22023'; END IF;
 -- El reloj de la base, ligado al consumo V3, es la fecha del hecho nuevo.
 p_registrada_en:=consumo.consumida_en;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:documental:'||participacion,0));
 SELECT * INTO previa FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
  WHERE s.participacion_ref=participacion AND s.clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previa.solicitud_ref IS DISTINCT FROM p_solicitud_ref OR previa.recibo_ref IS DISTINCT FROM p_recibo_ref
     OR previa.contenido_sha256 IS DISTINCT FROM p_contenido_sha256 OR previa.documento_ref IS DISTINCT FROM p_documento_ref
     OR previa.documento_sha256 IS DISTINCT FROM p_documento_sha256 OR previa.fecha_fin_causa IS DISTINCT FROM p_fecha_fin_causa
     OR previa.candidato_ref IS DISTINCT FROM p_candidato_ref OR previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref THEN
   RAISE EXCEPTION 'clave documental reutilizada con otro contenido' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true,previa.solicitud_ref,previa.recibo_ref,previa.contenido_sha256,previa.registrada_en,
    previa.version,CASE WHEN r.resultado='validada' THEN 'validada' WHEN r.resultado='rechazada' THEN 'rechazada' ELSE 'pendiente_rrhh' END
   FROM (SELECT 1) x LEFT JOIN vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r ON r.solicitud_ref=previa.solicitud_ref;
  RETURN;
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
            WHERE s.participacion_ref=participacion AND NOT EXISTS (
              SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r WHERE r.solicitud_ref=s.solicitud_ref)) THEN
  RAISE EXCEPTION 'ya hay una solicitud documental pendiente' USING ERRCODE='VBP02';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.solicitud_documental_rrhh(
  solicitud_ref,recibo_ref,contenido_sha256,bolsa_ref,participacion_ref,candidato_ref,tipo,documento_ref,
  documento_sha256,fecha_fin_causa,version,estado_origen,clave_idempotencia,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_solicitud_ref,p_recibo_ref,p_contenido_sha256,p_bolsa_ref,participacion,p_candidato_ref,'documental_rrhh',
  p_documento_ref,p_documento_sha256,p_fecha_fin_causa,1,'pendiente_rrhh',p_clave,consumo.decision_ref,
  consumo.auditoria_ref,p_registrada_en);
 RETURN QUERY SELECT false,p_solicitud_ref,p_recibo_ref,p_contenido_sha256,p_registrada_en,1,'pendiente_rrhh'::text;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.rechazar_solicitud_documental_rrhh_v1(
 p_solicitud_ref text,p_version_esperada integer,p_contenido_sha256 text,p_motivo text,p_actor text,
 p_clave_idempotencia text,p_recibo_ref text,p_resuelta_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,solicitud_ref text,recibo_ref text,estado text,resuelta_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE solicitud vec_bolsa_llamamientos.solicitud_documental_rrhh%ROWTYPE;
 previa vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh%ROWTYPE;
 consumo record; decision jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR current_setting('transaction_isolation')<>'serializable'
    OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^solicitud-documental:[0-9a-f]{64}$'
    OR p_version_esperada IS DISTINCT FROM 1 OR p_contenido_sha256 IS NULL OR p_contenido_sha256 !~ '^[0-9a-f]{64}$'
    OR p_motivo IS NULL OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_motivo<>pg_catalog.btrim(p_motivo)
    OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 256
    OR p_clave_idempotencia IS NULL OR pg_catalog.octet_length(p_clave_idempotencia) NOT BETWEEN 8 AND 256
    OR p_recibo_ref IS NULL OR pg_catalog.octet_length(p_recibo_ref) NOT BETWEEN 3 AND 256
    OR p_resuelta_en IS NULL OR NOT pg_catalog.isfinite(p_resuelta_en) THEN
  RAISE EXCEPTION 'resolución documental inválida' USING ERRCODE='22023';
 END IF;
 BEGIN decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'resolución documental no autorizada' USING ERRCODE='42501'; END;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(
  'bolsa.solicitudes_documentales.resolver',p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM p_solicitud_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR decision->>'principal_id' IS DISTINCT FROM p_actor THEN
  RAISE EXCEPTION 'resolución documental no autorizada' USING ERRCODE='42501';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:documental:'||p_solicitud_ref,0));
 SELECT * INTO solicitud FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
  WHERE s.solicitud_ref=p_solicitud_ref FOR UPDATE;
 IF solicitud.solicitud_ref IS NULL OR solicitud.version IS DISTINCT FROM p_version_esperada
    OR solicitud.contenido_sha256 IS DISTINCT FROM p_contenido_sha256 THEN
  RAISE EXCEPTION 'solicitud documental ajena o versión distinta' USING ERRCODE='VBS02'; END IF;
 SELECT * INTO previa FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r
  WHERE r.solicitud_ref=p_solicitud_ref;
 IF previa.solicitud_ref IS NOT NULL THEN
  IF previa.resultado IS DISTINCT FROM 'rechazada' OR previa.contenido_sha256 IS DISTINCT FROM p_contenido_sha256
     OR previa.actor_ref IS DISTINCT FROM p_actor OR previa.clave_idempotencia IS DISTINCT FROM p_clave_idempotencia
     OR previa.motivo IS DISTINCT FROM p_motivo OR previa.recibo_ref IS DISTINCT FROM p_recibo_ref THEN
   RAISE EXCEPTION 'replay documental distinto' USING ERRCODE='VBS01'; END IF;
  RETURN QUERY SELECT true,previa.solicitud_ref,previa.recibo_ref,'rechazada'::text,previa.resuelta_en;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh(
  solicitud_ref,version_esperada,contenido_sha256,resultado,motivo,actor_ref,clave_idempotencia,
  recibo_ref,decision_ref,auditoria_ref,resuelta_en)
 VALUES(p_solicitud_ref,p_version_esperada,p_contenido_sha256,'rechazada',p_motivo,p_actor,p_clave_idempotencia,
  p_recibo_ref,consumo.decision_ref,consumo.auditoria_ref,p_resuelta_en);
 RETURN QUERY SELECT false,p_solicitud_ref,p_recibo_ref,'rechazada'::text,p_resuelta_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_solicitudes_documentales_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_actor_ref text,p_corte timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(solicitud_ref text,version integer,contenido_sha256 text,documento_ref text,
 documento_sha256 text,fecha_fin_causa date,estado text,recibo_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; decision jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_actor_ref IS NULL
    OR p_corte IS NULL OR NOT pg_catalog.isfinite(p_corte) THEN
  RAISE EXCEPTION 'consulta documental no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'consulta documental no autorizada' USING ERRCODE='42501'; END;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(
  'bolsa.solicitudes_documentales.consultar_rrhh',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_participacion_ref
    OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref THEN
  RAISE EXCEPTION 'consulta documental no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
   WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'consulta documental no autorizada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT s.solicitud_ref,s.version,s.contenido_sha256,s.documento_ref,s.documento_sha256,
   s.fecha_fin_causa,'pendiente_rrhh'::text,s.recibo_ref,s.registrada_en
  FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
  WHERE s.bolsa_ref=p_bolsa_ref AND s.participacion_ref=p_participacion_ref AND s.registrada_en<=p_corte
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r
      WHERE r.solicitud_ref=s.solicitud_ref)
  ORDER BY s.registrada_en DESC,s.solicitud_ref DESC;
END $f$;

-- La bandeja ya estaba protegida por su lectura RRHH. Se añade la nueva
-- clase de aviso sin retirar ni reinterpretar los dos conjuntos B30.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(p_corte timestamptz)
RETURNS TABLE(tipo text,bolsa_ref text,referencia text,detalle jsonb,fecha timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT 'solicitud_portal'::text,s.bolsa_ref,s.solicitud_ref,
  pg_catalog.jsonb_build_object('participacion_ref',s.participacion_ref,'solicitud',s.tipo,
   'pausa_hasta',s.pausa_hasta,'situacion_previa',s.situacion_previa,
   'recibo_ref',s.recibo_ref,'regla_ref',s.regla_ref),s.registrada_en
 FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
 WHERE s.registrada_en<=p_corte AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
 UNION ALL
 SELECT 'respuesta_portal'::text,r.bolsa_ref,r.respuesta_ref,
  pg_catalog.jsonb_build_object('participacion_ref',r.participacion_ref,'llamamiento_ref',r.llamamiento_ref,
   'respuesta',r.respuesta,'modo',r.modo,'causa',r.causa,'justificante_ref',r.justificante_ref,
   'justificante_sha256',r.justificante_sha256,'contacto_en',r.contacto_en,
   'vence_antes_de',r.vence_antes_de,'recibo_ref',r.recibo_ref,'regla_ref',r.regla_ref),r.respondida_en
 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r WHERE r.respondida_en<=p_corte
 UNION ALL
 SELECT 'solicitud_portal'::text,s.bolsa_ref,s.solicitud_ref,
  pg_catalog.jsonb_build_object('solicitud','documental_rrhh','participacion_ref',s.participacion_ref,
   'estado','pendiente_rrhh'),s.registrada_en
 FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
 WHERE s.registrada_en<=p_corte AND NOT EXISTS
  (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r WHERE r.solicitud_ref=s.solicitud_ref)
$f$;

-- El lector propio ya exige una marca V3 de consulta/respuesta del mismo
-- candidato. Se amplía su proyección sin abrir acceso a otro integrante.
DO $preimagen_lector$
DECLARE f pg_catalog.regprocedure :=
 'vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamp with time zone,text[])'::pg_catalog.regprocedure;
 v_def text;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT v_def;
 IF pg_catalog.strpos(v_def,'exigir_consumo_candidato_v1')=0
    OR pg_catalog.strpos(v_def,'ultima_respuesta')=0
    OR pg_catalog.strpos(v_def,'ultima_solicitud_documental')<>0 THEN
  RAISE EXCEPTION 'PARO clave=lector_propio_marcas, actual=%/%/%, esperado=true/true/false',
   pg_catalog.strpos(v_def,'exigir_consumo_candidato_v1')<>0,
   pg_catalog.strpos(v_def,'ultima_respuesta')<>0,
   pg_catalog.strpos(v_def,'ultima_solicitud_documental')<>0 USING ERRCODE='55000'; END IF;
END $preimagen_lector$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_v1(
 p_candidato_ref text,p_corte timestamptz,p_resultados_efectivos text[])
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta','responder'],p_candidato_ref);
 RETURN (
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
  'bolsa',p.bolsa_ref,
  'llamamiento_abierto',CASE WHEN a.contacto_en IS NULL THEN NULL ELSE pg_catalog.jsonb_build_object(
    'contacto_en',pg_catalog.to_char(a.contacto_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END,
  'solicitud_pendiente',(SELECT pg_catalog.jsonb_build_object('tipo',s.tipo,'recibo',s.recibo_ref,
    'registrada_en',pg_catalog.to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'pausa_hasta',CASE WHEN s.pausa_hasta IS NULL THEN NULL ELSE pg_catalog.to_char(s.pausa_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
   FROM vec_bolsa_llamamientos.solicitud_portal_candidato s
   WHERE s.participacion_ref=p.participacion_ref AND s.registrada_en<=p_corte
     AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
   ORDER BY s.registrada_en DESC LIMIT 1),
  'ultima_solicitud_documental',(SELECT pg_catalog.jsonb_build_object(
    'tipo','documental_rrhh','recibo',s.recibo_ref,
    'registrada_en',pg_catalog.to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'estado',coalesce(r.resultado,'pendiente_rrhh'),'recibo_resolucion_ref',r.recibo_ref,
    'recibo_b8_ref',r.recibo_b8_ref)
   FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
   LEFT JOIN vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r ON r.solicitud_ref=s.solicitud_ref
   WHERE s.participacion_ref=p.participacion_ref AND s.candidato_ref=p_candidato_ref
     AND s.registrada_en<=p_corte ORDER BY s.registrada_en DESC,s.solicitud_ref DESC LIMIT 1),
  'solicitud_documental_pendiente',EXISTS(
   SELECT 1 FROM vec_bolsa_llamamientos.solicitud_documental_rrhh s
   WHERE s.participacion_ref=p.participacion_ref AND s.candidato_ref=p_candidato_ref
     AND s.registrada_en<=p_corte AND NOT EXISTS
      (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_solicitud_documental_rrhh r WHERE r.solicitud_ref=s.solicitud_ref)),
  'ultima_respuesta',(SELECT pg_catalog.jsonb_build_object('respuesta',r.respuesta,'modo',r.modo,
    'recibo',r.recibo_ref,'respondida_en',pg_catalog.to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
   FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
   WHERE r.participacion_ref=p.participacion_ref AND r.respondida_en<=p_corte
   ORDER BY r.respondida_en DESC LIMIT 1)
 ) ORDER BY p.bolsa_ref),'[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 LEFT JOIN LATERAL vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(
  p.participacion_ref,p_corte,p_resultados_efectivos) a ON true);
END $f$;

DO $acl$
DECLARE f pg_catalog.regprocedure; p record; x record;
BEGIN
 FOR p IN SELECT * FROM (VALUES
  ('b77_huella_partes_v1',1,false),
  ('b77_documento_ref_opaco_v1',1,false),
  ('solicitar_documental_portal_v1',20,true),
  ('regularizar_solicitud_documental_rrhh_v1',28,true),
  ('rechazar_solicitud_documental_rrhh_v1',18,true),
  ('consultar_solicitudes_documentales_rrhh_v1',14,true)
 ) AS z(nombre,argumentos,publica) LOOP
  SELECT q.oid::pg_catalog.regprocedure INTO STRICT f FROM pg_catalog.pg_proc q
   JOIN pg_catalog.pg_namespace n ON n.oid=q.pronamespace
   WHERE n.nspname='vec_bolsa_llamamientos' AND q.proname=p.nombre AND q.pronargs=p.argumentos;
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc q
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
   WHERE q.oid=f AND a.grantee<>q.proowner LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF p.publica THEN
   EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f::text);
  END IF;
  IF p.nombre='solicitar_documental_portal_v1' THEN
   IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_portal_externo'
      AND NOT rolcanlogin AND NOT rolbypassrls) THEN
    RAISE EXCEPTION 'PARO clave=rol_portal_externo, actual=false, esperado=true' USING ERRCODE='55000'; END IF;
   EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_portal_externo',f::text);
  END IF;
  IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
     OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM p.publica THEN
   RAISE EXCEPTION 'PARO clave=funcion_%, actual=%/%, esperado=%/%',p.nombre,
    (SELECT proowner::text FROM pg_catalog.pg_proc WHERE oid=f),
    (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f),
    'vec_bolsa_llamamientos_propietario',p.publica USING ERRCODE='55000';
  END IF;
 END LOOP;
 f:='vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(timestamp with time zone)'::pg_catalog.regprocedure;
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE') THEN
  RAISE EXCEPTION 'PARO clave=bandeja_RRHH_EXECUTE, actual=false, esperado=true' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
