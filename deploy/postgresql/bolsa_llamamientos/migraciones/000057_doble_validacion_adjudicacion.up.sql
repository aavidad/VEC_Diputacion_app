\set ON_ERROR_STOP on
-- B57: una oferta puede cubrir varias plazas. La preparación y la
-- confirmación son dos actos durables por (oferta, número de plaza). El
-- catálogo vigente al publicar queda congelado en plazo.politica_version;
-- versiones anteriores sin la nueva clave exigen dos personas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000057',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.capacidad_oferta') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.preparacion_adjudicacion_oferta') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_confirmacion_adjudicacion_oferta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc p
       WHERE p.oid=to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
         AND position('B56: motivo unido' in pg_get_functiondef(p.oid))>0)
 THEN RAISE EXCEPTION 'B57: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.capacidad_oferta (
 oferta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 numero_plazas integer NOT NULL CHECK(numero_plazas BETWEEN 1 AND 100),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[A-Za-z0-9:_-]+$' AND octet_length(unidad_ref) BETWEEN 1 AND 256),
 ambito_ref text NOT NULL CHECK(ambito_ref ~ '^[A-Za-z0-9:_-]+$' AND octet_length(ambito_ref) BETWEEN 1 AND 256),
 recibo_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_bolsa_llamamientos.preparacion_adjudicacion_oferta (
 preparacion_ref text PRIMARY KEY CHECK(preparacion_ref ~ '^recibo:preparacion-oferta:[0-9a-f]{64}$'),
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 numero_de_plaza integer NOT NULL CHECK(numero_de_plaza BETWEEN 1 AND 100),
 participacion_ref text NOT NULL CHECK(octet_length(participacion_ref) BETWEEN 1 AND 256),
 orden_vigente bigint NOT NULL CHECK(orden_vigente>0),
 actor_ref text NOT NULL CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[A-Za-z0-9:_-]+$' AND octet_length(unidad_ref) BETWEEN 1 AND 256),
 ambito_ref text NOT NULL CHECK(ambito_ref ~ '^[A-Za-z0-9:_-]+$' AND octet_length(ambito_ref) BETWEEN 1 AND 256),
 clave_idempotencia text NOT NULL CHECK(octet_length(clave_idempotencia) BETWEEN 8 AND 256),
 recibo_resolucion_ref text NOT NULL CHECK(recibo_resolucion_ref ~ '^recibo:resolucion-oferta:[0-9a-f]{64}$'),
 politica_version bigint NOT NULL CHECK(politica_version>0),
 requiere_segunda_validacion boolean NOT NULL CHECK(requiere_segunda_validacion),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 preparada_en timestamptz(6) NOT NULL,
 UNIQUE(oferta_ref,numero_de_plaza,actor_ref,clave_idempotencia)
);
CREATE INDEX preparacion_adjudicacion_ultima_idx ON vec_bolsa_llamamientos.preparacion_adjudicacion_oferta
 (oferta_ref,numero_de_plaza,preparada_en DESC,preparacion_ref DESC);
CREATE TABLE vec_bolsa_llamamientos.adjudicacion_oferta (
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 numero_de_plaza integer NOT NULL CHECK(numero_de_plaza BETWEEN 1 AND 100),
 participacion_ref text NOT NULL CHECK(octet_length(participacion_ref) BETWEEN 1 AND 256),
 orden_vigente bigint NOT NULL CHECK(orden_vigente>0),
 preparacion_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.preparacion_adjudicacion_oferta(preparacion_ref),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:adjudicacion-oferta:[0-9a-f]{64}$'),
 actor_preparador_ref text NOT NULL CHECK(actor_preparador_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 actor_confirmador_ref text NOT NULL CHECK(actor_confirmador_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_confirmacion text NOT NULL,
 decision_confirmacion_ref text NOT NULL UNIQUE,
 auditoria_confirmacion_ref text NOT NULL UNIQUE,
 confirmada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(oferta_ref,numero_de_plaza),
 UNIQUE(oferta_ref,participacion_ref),
 CHECK(actor_confirmador_ref<>actor_preparador_ref)
);
CREATE TABLE vec_bolsa_llamamientos.evento_adjudicacion_oferta (
 recibo_ref text PRIMARY KEY,
 oferta_ref text NOT NULL,
 numero_de_plaza integer NOT NULL,
 tipo text NOT NULL CHECK(tipo IN ('preparada','confirmada')),
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 CHECK(numero_de_plaza BETWEEN 1 AND 100)
);

DO $proteccion$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['capacidad_oferta','preparacion_adjudicacion_oferta','adjudicacion_oferta','evento_adjudicacion_oferta'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_interno ON vec_bolsa_llamamientos.%I FOR ALL TO vec_bolsa_llamamientos_propietario USING (current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user=''vec_bolsa_llamamientos_propietario'')',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_llamamientos_ejecutor',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
 END LOOP;
END $proteccion$;

-- B54 admite exactamente dos campos de adjudicación. La única ampliación
-- permitida es el booleano gobernado; ninguna versión ya publicada se edita.
DO $politica$
DECLARE f regprocedure:='vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 antigua text; nueva text;
 marca text:=$m$OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))<>2$m$;
 reemplazo text:=$r$OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion')) NOT IN (2,3)
    OR (p_politica->'adjudicacion' ? 'requiere_segunda_validacion' AND
        p_politica#>>'{adjudicacion,requiere_segunda_validacion}' IS DISTINCT FROM 'true')
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))=3
       AND NOT (p_politica->'adjudicacion' ? 'requiere_segunda_validacion')$r$;
BEGIN
 antigua:=pg_get_functiondef(f);
 IF (length(antigua)-length(replace(antigua,marca,'')))<>length(marca)
    OR strpos(antigua,'requiere_segunda_validacion')<>0
 THEN RAISE EXCEPTION 'B57: publicador B54 incompatible' USING ERRCODE='55000'; END IF;
 nueva:=replace(antigua,marca,reemplazo);
 EXECUTE nueva;
 IF pg_get_functiondef(f) IS DISTINCT FROM nueva THEN
  RAISE EXCEPTION 'B57: publicador alterado' USING ERRCODE='55000'; END IF;
END $politica$;

-- El publicador nuevo reproduce la guarda B54 de plazo, añadiendo el número
-- de plazas al material autorizado. Después delega la autorización/alta B28
-- con datos históricos sin la clave nueva; el ejecutor no puede invocar esas
-- funciones internas directamente.
CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(
 p_oferta text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_plazo jsonb,
 p_publicada timestamptz,p_vence timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE v_numero integer; x record; capacidad_guardada record; d jsonb;
 v_calendarios text; v_material text; v_contexto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR jsonb_typeof(p_datos) IS DISTINCT FROM 'object'
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_datos) k WHERE k NOT IN
       ('categoria','centro','fecha_inicio','fecha_fin','descripcion','numero_plazas'))
    OR (p_datos ? 'numero_plazas' AND (jsonb_typeof(p_datos->'numero_plazas') IS DISTINCT FROM 'number'
       OR (p_datos->>'numero_plazas') !~ '^[1-9][0-9]{0,2}$'))
 THEN RAISE EXCEPTION 'B57: número de plazas inválido' USING ERRCODE='22023'; END IF;
 v_numero:=coalesce((p_datos->>'numero_plazas')::integer,1);
 IF v_numero NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'B57: número de plazas fuera de límite' USING ERRCODE='22023'; END IF;
 IF p_publicada IS NULL OR p_vence IS NULL OR p_bolsa IS NULL OR p_plazo IS NULL
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_unidad) NOT BETWEEN 1 AND 256
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_ambito) NOT BETWEEN 1 AND 256
    OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>12)
    OR (p_plazo->>'unidad'<>'horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>10)
    OR NOT (p_plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                              'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR jsonb_typeof(p_plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$')
    OR p_plazo->>'regla_ref' IS NULL OR p_plazo->>'huella_catalogo' IS NULL
    OR p_plazo->>'unidad' IS NULL OR p_plazo->>'cantidad' IS NULL
    OR p_plazo->>'computo' IS NULL OR p_plazo->>'municipio_sede' IS NULL
    OR p_plazo->>'ultimo_dia' IS NULL OR p_plazo->>'politica_version' IS NULL
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (p_plazo->>'apertura_en' IS NULL OR p_plazo->>'vence_en' IS NULL))
 THEN RAISE EXCEPTION 'B57: material de plazo incompleto' USING ERRCODE='22023'; END IF;
 SELECT string_agg(c#>>'{}',chr(30) ORDER BY n) INTO v_calendarios
 FROM jsonb_array_elements(p_plazo->'calendarios') WITH ORDINALITY a(c,n);
 v_material:=encode(sha256(convert_to(array_to_string(ARRAY[
  p_bolsa,
  to_char(p_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(p_vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  p_plazo->>'regla_ref',p_plazo->>'huella_catalogo',p_plazo->>'unidad',
  p_plazo->>'cantidad',p_plazo->>'computo',p_plazo->>'municipio_sede',
  p_plazo->>'ultimo_dia',p_plazo->>'politica_version',v_calendarios
 ] || CASE WHEN p_plazo->>'unidad'='horas_naturales'
           THEN ARRAY[p_plazo->>'apertura_en',p_plazo->>'vence_en']
           ELSE ARRAY[]::text[] END || ARRAY[v_numero::text],chr(31)),'UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||v_material||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: decisión de oferta inválida' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B57: número de plazas distinto del autorizado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_bolsa_llamamientos.publicar_oferta_v1(
  p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos-'numero_plazas',p_plazo,p_publicada,p_vence,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO capacidad_guardada FROM vec_bolsa_llamamientos.capacidad_oferta WHERE oferta_ref=p_oferta;
 IF FOUND THEN
  IF capacidad_guardada.numero_plazas<>v_numero OR capacidad_guardada.recibo_ref<>p_recibo
     OR capacidad_guardada.unidad_ref<>p_unidad OR capacidad_guardada.ambito_ref<>p_ambito THEN
   RAISE EXCEPTION 'B57: clave con otra capacidad' USING ERRCODE='VBO01'; END IF;
 ELSE
  IF x.reutilizada THEN
   IF v_numero<>1 THEN RAISE EXCEPTION 'B57: oferta anterior con otra capacidad' USING ERRCODE='VBO01'; END IF;
  ELSE
   INSERT INTO vec_bolsa_llamamientos.capacidad_oferta(oferta_ref,numero_plazas,unidad_ref,ambito_ref,recibo_ref,registrada_en)
   VALUES(p_oferta,v_numero,p_unidad,p_ambito,p_recibo,clock_timestamp());
  END IF;
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp()),x.reutilizada;
END $f$;

-- Proyección B57: siempre muestra la capacidad y las adjudicaciones por plaza;
-- no publica actor, decisión ni auditoría. Una preparación pendiente no es
-- una adjudicación. La siguiente propuesta excluye a quienes ya recibieron
-- una plaza de esta oferta.
CREATE FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta text,p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE base jsonb; o record; cap integer; adjudicaciones jsonb; n integer; pendiente record; propuesto record; estado text; propuesta jsonb;
BEGIN
 base:=vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta,p_corte);
 IF base IS NULL THEN RETURN NULL; END IF;
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 SELECT coalesce((SELECT numero_plazas FROM vec_bolsa_llamamientos.capacidad_oferta WHERE oferta_ref=p_oferta),1) INTO cap;
 SELECT count(*),coalesce(jsonb_agg(jsonb_build_object('numero_de_plaza',a.numero_de_plaza,
    'participacion_ref',a.participacion_ref,'orden_vigente',a.orden_vigente,'recibo_ref',a.recibo_ref,
    'confirmada_en',to_char(a.confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
    ORDER BY a.numero_de_plaza),'[]'::jsonb)
 INTO n,adjudicaciones FROM vec_bolsa_llamamientos.adjudicacion_oferta a
 WHERE a.oferta_ref=p_oferta AND a.confirmada_en<=p_corte;
 IF base->'resolucion' IS NOT NULL AND base->'resolucion'<>'null'::jsonb THEN
  RETURN base||jsonb_build_object('datos',(base->'datos')||jsonb_build_object('numero_plazas',cap),
    'adjudicaciones',adjudicaciones,'preparacion',NULL);
 END IF;
 SELECT * INTO pendiente FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta p
 WHERE p.oferta_ref=p_oferta AND p.numero_de_plaza=n+1 AND p.preparada_en<=p_corte
 ORDER BY p.preparada_en DESC,p.preparacion_ref DESC LIMIT 1;
 propuesta:=NULL;
 IF p_corte>=o.vence_antes_de AND n<cap THEN
  SELECT d.participacion_ref,ov.orden_vigente INTO propuesto
  FROM vec_bolsa_llamamientos.disposicion_oferta d
  JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref,p_corte) ov
    ON ov.participacion_ref=d.participacion_ref
  WHERE d.oferta_ref=p_oferta AND d.manifestada_en<o.vence_antes_de
    AND ov.orden_vigente IS NOT NULL
    AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.adjudicacion_oferta a
                   WHERE a.oferta_ref=p_oferta AND a.participacion_ref=d.participacion_ref)
  ORDER BY ov.orden_vigente,d.participacion_ref LIMIT 1;
  IF propuesto.participacion_ref IS NOT NULL THEN
   propuesta:=jsonb_build_object('tipo','adjudicar','numero_de_plaza',n+1,
     'participacion_ref',propuesto.participacion_ref,'orden_vigente',propuesto.orden_vigente);
  ELSIF n=0 THEN propuesta:=jsonb_build_object('tipo','llamamiento_directo'); END IF;
 END IF;
 estado:=CASE WHEN n=cap THEN 'adjudicada'
   WHEN pendiente.preparacion_ref IS NOT NULL THEN 'pendiente_segunda_validacion'
   WHEN p_corte<o.vence_antes_de THEN 'abierta' ELSE 'pendiente_resolucion' END;
 RETURN base||jsonb_build_object('datos',(base->'datos')||jsonb_build_object('numero_plazas',cap),
   'estado',estado,'propuesta',propuesta,'adjudicaciones',adjudicaciones,
   'preparacion',CASE WHEN pendiente.preparacion_ref IS NULL THEN NULL ELSE jsonb_build_object(
     'numero_de_plaza',pendiente.numero_de_plaza,'participacion_ref',pendiente.participacion_ref,
     'orden_vigente',pendiente.orden_vigente,'recibo_ref',pendiente.preparacion_ref,
     'preparada_en',to_char(pendiente.preparada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END);
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v2(p_bolsa text,p_corte timestamptz,p_limite integer)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_oferta_v2(x.oferta_ref,p_corte)
   ORDER BY x.publicada_en DESC,x.oferta_ref),'[]'::jsonb)
 FROM (SELECT oferta_ref,publicada_en FROM vec_bolsa_llamamientos.oferta_publicada
   WHERE bolsa_ref=p_bolsa AND publicada_en<=p_corte AND p_limite BETWEEN 1 AND 100
   ORDER BY publicada_en DESC,oferta_ref LIMIT greatest(least(p_limite,100),1)) x
$f$;

-- El ejecutor pierde B28: ninguna llamada SQL directa puede adjudicar con
-- un único actor. V2 toma el mismo cerrojo de B28. El primer permiso sigue
-- siendo la emisión B7 sobre la bolsa; el segundo tiene acción AD3-102 propia.
CREATE FUNCTION vec_bolsa_llamamientos.resolver_oferta_v2(
 p_oferta text,p_recibo text,p_bolsa text,p_participacion text,p_actor text,p_clave text,p_numero_de_plaza integer,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE o record; vieja record; prep record; capacidad_guardada record; consumo record; d jsonb; v_cap integer; v_ocupadas integer;
 v_ahora timestamptz(6):=clock_timestamp(); v_propuesto record; v_politica record;
 v_requiere boolean; v_preparacion text; v_contexto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:resolucion-oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_numero_de_plaza IS NULL OR p_numero_de_plaza NOT BETWEEN 1 AND 100
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_unidad) NOT BETWEEN 1 AND 256
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_ambito) NOT BETWEEN 1 AND 256
    OR (p_participacion IS NOT NULL AND (p_participacion<>btrim(p_participacion) OR octet_length(p_participacion) NOT BETWEEN 1 AND 256))
 THEN RAISE EXCEPTION 'B57: preparación inválida' USING ERRCODE='22023'; END IF;
 IF p_recibo IS DISTINCT FROM 'recibo:resolucion-oferta:'||encode(sha256(convert_to(
    p_oferta||chr(31)||p_numero_de_plaza::text||chr(31)||p_clave,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B57: recibo de plaza incompatible' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 IF NOT FOUND OR o.bolsa_ref<>p_bolsa THEN
  RAISE EXCEPTION 'B57: oferta no disponible' USING ERRCODE='23503'; END IF;
 SELECT * INTO vieja FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref=p_oferta;
 IF FOUND THEN
  IF p_numero_de_plaza<>1 THEN RAISE EXCEPTION 'B57: oferta ya resuelta' USING ERRCODE='VBO02'; END IF;
  RETURN QUERY SELECT x.oferta,x.reutilizada FROM vec_bolsa_llamamientos.resolver_oferta_v1(
   p_oferta,p_recibo,p_bolsa,p_participacion,p_actor,p_clave,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) x;
  RETURN;
 END IF;
 SELECT * INTO capacidad_guardada FROM vec_bolsa_llamamientos.capacidad_oferta WHERE oferta_ref=p_oferta;
 IF NOT FOUND AND p_participacion IS NULL AND p_numero_de_plaza=1 THEN
  -- Oferta B28 anterior sin ámbito estructurado: conserva exclusivamente
  -- el paso a llamamiento directo ya admitido por B28. Nunca adjudica.
  RETURN QUERY SELECT x.oferta,x.reutilizada FROM vec_bolsa_llamamientos.resolver_oferta_v1(
   p_oferta,p_recibo,p_bolsa,NULL,p_actor,p_clave,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) x;
  RETURN;
 END IF;
 IF NOT FOUND OR capacidad_guardada.unidad_ref<>p_unidad OR capacidad_guardada.ambito_ref<>p_ambito THEN
  RAISE EXCEPTION 'B57: ámbito de oferta no acreditado' USING ERRCODE='42501'; END IF;
 v_cap:=capacidad_guardada.numero_plazas;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: decisión inválida' USING ERRCODE='42501'; END;
 v_contexto:=encode(sha256(convert_to(
   '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||'"},"atributos":{}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B57: ámbito distinto del autorizado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_ocupadas FROM vec_bolsa_llamamientos.adjudicacion_oferta WHERE oferta_ref=p_oferta;
 IF p_participacion IS NULL THEN
  IF p_numero_de_plaza<>1 OR v_ocupadas<>0 OR EXISTS(
    SELECT 1 FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta WHERE oferta_ref=p_oferta)
  THEN RAISE EXCEPTION 'B57: cobertura parcial no es llamamiento directo' USING ERRCODE='VBO04'; END IF;
  RETURN QUERY SELECT x.oferta,x.reutilizada FROM vec_bolsa_llamamientos.resolver_oferta_v1(
   p_oferta,p_recibo,p_bolsa,NULL,p_actor,p_clave,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) x;
  RETURN;
 END IF;
 -- Consumir autorización viva incluso en replay; un recibo anterior no
 -- concede acceso. Si falla cualquier comprobación, la transacción revierte.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
 THEN RAISE EXCEPTION 'B57: preparación denegada' USING ERRCODE='42501'; END IF;
 v_preparacion:='recibo:preparacion-oferta:'||substring(p_recibo FROM 26);
 SELECT * INTO prep FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta
 WHERE preparacion_ref=v_preparacion;
 IF FOUND THEN
  IF prep.oferta_ref<>p_oferta OR prep.numero_de_plaza<>p_numero_de_plaza
     OR prep.participacion_ref<>p_participacion OR prep.actor_ref<>p_actor
     OR prep.clave_idempotencia<>p_clave OR prep.recibo_resolucion_ref<>p_recibo
  THEN RAISE EXCEPTION 'B57: clave reutilizada' USING ERRCODE='VBO01'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora)||
    jsonb_build_object('recibo_preparacion_replay',prep.preparacion_ref),true;
  RETURN;
 END IF;
 IF v_ahora<o.vence_antes_de THEN
  RAISE EXCEPTION 'B57: plazo abierto' USING ERRCODE='VBO03'; END IF;
 IF p_numero_de_plaza<>v_ocupadas+1 OR p_numero_de_plaza>v_cap THEN
  RAISE EXCEPTION 'B57: plaza ya cubierta o fuera de capacidad' USING ERRCODE='VBO02'; END IF;
 SELECT p.version,p.politica INTO v_politica FROM vec_bolsa_llamamientos.politica_ofertas_version p
 WHERE p.bolsa_ref=p_bolsa AND p.version=(o.plazo->>'politica_version')::bigint;
 IF NOT FOUND THEN RAISE EXCEPTION 'B57: política de oferta no disponible' USING ERRCODE='55000'; END IF;
 IF v_politica.politica#>>'{adjudicacion,requiere_segunda_validacion}' NOT IN ('true','false')
    AND v_politica.politica#>'{adjudicacion,requiere_segunda_validacion}' IS NOT NULL
 THEN RAISE EXCEPTION 'B57: política de confirmación inválida' USING ERRCODE='55000'; END IF;
 -- B57 no transforma una versión histórica con false en permiso para un
 -- solo actor: toda adjudicación exige dos actos, incluso en ese caso.
 v_requiere:=true;
 SELECT d.participacion_ref,ov.orden_vigente INTO v_propuesto
 FROM vec_bolsa_llamamientos.disposicion_oferta d
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref,v_ahora) ov
   ON ov.participacion_ref=d.participacion_ref
 WHERE d.oferta_ref=p_oferta AND d.manifestada_en<o.vence_antes_de
   AND ov.orden_vigente IS NOT NULL
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.adjudicacion_oferta a
                  WHERE a.oferta_ref=p_oferta AND a.participacion_ref=d.participacion_ref)
 ORDER BY ov.orden_vigente,d.participacion_ref LIMIT 1;
 IF v_propuesto.participacion_ref IS DISTINCT FROM p_participacion THEN
  RAISE EXCEPTION 'B57: propuesta de orden cambiada' USING ERRCODE='VBO04'; END IF;
 INSERT INTO vec_bolsa_llamamientos.preparacion_adjudicacion_oferta(
  preparacion_ref,oferta_ref,numero_de_plaza,participacion_ref,orden_vigente,actor_ref,unidad_ref,ambito_ref,
  clave_idempotencia,recibo_resolucion_ref,politica_version,requiere_segunda_validacion,
  decision_ref,auditoria_ref,preparada_en)
 VALUES(v_preparacion,p_oferta,p_numero_de_plaza,p_participacion,v_propuesto.orden_vigente,p_actor,p_unidad,p_ambito,
  p_clave,p_recibo,v_politica.version,v_requiere,consumo.decision_ref,consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.evento_adjudicacion_oferta
  (recibo_ref,oferta_ref,numero_de_plaza,tipo,auditoria_ref,registrada_en)
 VALUES(v_preparacion,p_oferta,p_numero_de_plaza,'preparada',consumo.auditoria_ref,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),false;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(
 p_oferta text,p_bolsa text,p_numero_de_plaza integer,p_preparacion text,p_actor text,p_clave text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE o record; prep record; ultima record; adjudicada record; consumo record; d jsonb;
 v_cap integer; v_ocupadas integer; v_ahora timestamptz(6):=clock_timestamp(); v_propuesto record; v_recibo text; v_contexto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_numero_de_plaza IS NULL OR p_numero_de_plaza NOT BETWEEN 1 AND 100
    OR p_preparacion IS NULL OR p_preparacion !~ '^recibo:preparacion-oferta:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
 THEN RAISE EXCEPTION 'B57: confirmación inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 -- AD3-102 autoriza la operación y la preparación exacta. Una decisión
 -- antigua no recupera recibo: cada intento consume autorización viva.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_confirmacion_adjudicacion_oferta_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B57: confirmación denegada' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_preparacion OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.oferta.adjudicacion.confirmar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'confirmar_adjudicacion_oferta'
    OR d->>'recurso_ref' IS DISTINCT FROM p_preparacion
    OR d->>'tipo_recurso' IS DISTINCT FROM 'preparacion_adjudicacion_oferta'
 THEN RAISE EXCEPTION 'B57: confirmación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 SELECT * INTO prep FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta WHERE preparacion_ref=p_preparacion;
 IF o.oferta_ref IS NULL OR o.bolsa_ref<>p_bolsa OR prep.preparacion_ref IS NULL
    OR prep.oferta_ref<>p_oferta OR prep.numero_de_plaza<>p_numero_de_plaza
    OR NOT prep.requiere_segunda_validacion OR prep.actor_ref=p_actor
 THEN RAISE EXCEPTION 'B57: segunda persona o preparación no válida' USING ERRCODE='42501'; END IF;
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||prep.ambito_ref||'","unidad_ref":"'||prep.unidad_ref||
  '"},"atributos":{"bolsa_ref":"'||p_bolsa||'","numero_de_plaza":"'||p_numero_de_plaza::text||
  '","oferta_ref":"'||p_oferta||'"}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B57: ámbito de confirmación distinto del preparado' USING ERRCODE='42501'; END IF;
 SELECT * INTO adjudicada FROM vec_bolsa_llamamientos.adjudicacion_oferta
 WHERE oferta_ref=p_oferta AND numero_de_plaza=p_numero_de_plaza;
 IF FOUND THEN
  IF adjudicada.preparacion_ref<>p_preparacion OR adjudicada.actor_confirmador_ref<>p_actor
     OR adjudicada.clave_confirmacion<>p_clave
  THEN RAISE EXCEPTION 'B57: adjudicación ya confirmada' USING ERRCODE='VBO02'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),true;
  RETURN;
 END IF;
 SELECT * INTO ultima FROM vec_bolsa_llamamientos.preparacion_adjudicacion_oferta
 WHERE oferta_ref=p_oferta AND numero_de_plaza=p_numero_de_plaza
 ORDER BY preparada_en DESC,preparacion_ref DESC LIMIT 1;
 IF ultima.preparacion_ref IS DISTINCT FROM p_preparacion THEN
  RAISE EXCEPTION 'B57: preparación sustituida' USING ERRCODE='VBO04'; END IF;
 SELECT coalesce((SELECT numero_plazas FROM vec_bolsa_llamamientos.capacidad_oferta WHERE oferta_ref=p_oferta),1) INTO v_cap;
 SELECT count(*) INTO v_ocupadas FROM vec_bolsa_llamamientos.adjudicacion_oferta WHERE oferta_ref=p_oferta;
 IF p_numero_de_plaza<>v_ocupadas+1 OR p_numero_de_plaza>v_cap
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.resolucion_oferta WHERE oferta_ref=p_oferta)
 THEN RAISE EXCEPTION 'B57: plaza no confirmable' USING ERRCODE='VBO02'; END IF;
 SELECT d.participacion_ref,ov.orden_vigente INTO v_propuesto
 FROM vec_bolsa_llamamientos.disposicion_oferta d
 JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref,v_ahora) ov
   ON ov.participacion_ref=d.participacion_ref
 WHERE d.oferta_ref=p_oferta AND d.manifestada_en<o.vence_antes_de
   AND ov.orden_vigente IS NOT NULL
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.adjudicacion_oferta a
                  WHERE a.oferta_ref=p_oferta AND a.participacion_ref=d.participacion_ref)
 ORDER BY ov.orden_vigente,d.participacion_ref LIMIT 1;
 IF v_propuesto.participacion_ref IS DISTINCT FROM prep.participacion_ref
    OR v_propuesto.orden_vigente IS DISTINCT FROM prep.orden_vigente
 THEN RAISE EXCEPTION 'B57: propuesta cambiada' USING ERRCODE='VBO04'; END IF;
 v_recibo:='recibo:adjudicacion-oferta:'||encode(sha256(convert_to(p_preparacion||chr(31)||p_clave,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.adjudicacion_oferta
  (oferta_ref,numero_de_plaza,participacion_ref,orden_vigente,preparacion_ref,recibo_ref,
   actor_preparador_ref,actor_confirmador_ref,clave_confirmacion,decision_confirmacion_ref,
   auditoria_confirmacion_ref,confirmada_en)
 VALUES(p_oferta,p_numero_de_plaza,prep.participacion_ref,prep.orden_vigente,p_preparacion,v_recibo,
   prep.actor_ref,p_actor,p_clave,consumo.decision_ref,consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.evento_adjudicacion_oferta
  (recibo_ref,oferta_ref,numero_de_plaza,tipo,auditoria_ref,registrada_en)
 VALUES(v_recibo,p_oferta,p_numero_de_plaza,'confirmada',consumo.auditoria_ref,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),false;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz),
 vec_bolsa_llamamientos.listar_ofertas_bolsa_v2(text,timestamptz,integer),
 vec_bolsa_llamamientos.resolver_oferta_v2(text,text,text,text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)
 FROM vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.listar_ofertas_bolsa_v2(text,timestamptz,integer),
 vec_bolsa_llamamientos.resolver_oferta_v2(text,text,text,text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text),
 vec_bolsa_llamamientos.confirmar_adjudicacion_oferta_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
