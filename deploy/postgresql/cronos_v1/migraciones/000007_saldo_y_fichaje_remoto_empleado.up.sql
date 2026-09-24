\set ON_ERROR_STOP on
-- Cronos para la persona empleada, primer corte: saldo propio, disponibilidad
-- del fichaje remoto, fichaje remoto y recuperación de su recibo. Cada función
-- consume en su misma transacción una decisión V3 nueva de su audiencia
-- (AD3-53), revalida el contexto con un único empleado vigente, fija la RLS
-- de esa persona, escribe su propia evidencia de acceso y sólo entonces lee
-- o registra. Requiere 000001–000006 y AD3-53. No crea cuentas LOGIN.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000007',0));
DO $pre$
DECLARE fachada text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_policy WHERE polname='leer_resultado_para_recibo_nominal'
                AND polrelid='vec_cronos_v1.resultado_ejecucion_marcaje'::regclass)
    OR to_regprocedure('vec_cronos_v1.registrar_marcaje_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)') IS NULL
    OR to_regclass('vec_cronos_v1.teletrabajo_autorizacion') IS NULL
    OR to_regclass('vec_cronos_v1.marcaje_remoto_autorizado') IS NOT NULL
    OR to_regprocedure('vec_cronos_v1.registrar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
   RAISE EXCEPTION 'Cronos 000007: preimagen incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH fachada IN ARRAY ARRAY['registrar_y_consumir_cronos_marcaje_propio_v3_atestada','consumir_cronos_disponibilidad_remota_v3_atestada',
   'consumir_cronos_recibo_remoto_v3_atestada','consumir_cronos_saldo_propio_v3_atestada'] LOOP
   IF to_regprocedure('vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
      OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
     RAISE EXCEPTION 'Cronos 000007: falta consumidor AD3-53 %',fachada USING ERRCODE='55000';
   END IF;
 END LOOP;
END $pre$;

-- Evidencia propia de cada acceso autorizado. Solo adición y RLS por persona.
CREATE TABLE vec_cronos_v1.saldo_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  desde date NOT NULL,
  hasta date NOT NULL,
  zona_horaria text NOT NULL CHECK (zona_horaria IN ('Europe/Madrid','Atlantic/Canary')),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL,
  CHECK (desde<=hasta)
);
CREATE TABLE vec_cronos_v1.remoto_consulta_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  clave_operacion text CHECK (clave_operacion IS NULL OR clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  autorizacion_ref text REFERENCES vec_cronos_v1.teletrabajo_autorizacion,
  continuidad_confirmada boolean NOT NULL,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL,
  CHECK (autorizacion_ref IS NOT NULL OR NOT continuidad_confirmada)
);
-- Un marcaje remoto sólo existe junto a la autorización de teletrabajo que lo
-- amparó. La fila se escribe antes del hecho y el disparador de 000005 la exige.
CREATE TABLE vec_cronos_v1.marcaje_remoto_autorizado (
  marcaje_ref text PRIMARY KEY REFERENCES vec_cronos_v1.marcaje_original DEFERRABLE INITIALLY DEFERRED,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  clave_operacion text NOT NULL UNIQUE,
  autorizacion_ref text NOT NULL REFERENCES vec_cronos_v1.teletrabajo_autorizacion,
  decision_ref text NOT NULL UNIQUE,
  registrada_en timestamptz(6) NOT NULL,
  CHECK (marcaje_ref='marcaje:cronos:'||clave_operacion)
);
CREATE TABLE vec_cronos_v1.marcaje_remoto_recuperacion (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  clave_operacion text NOT NULL CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  resultado text NOT NULL CHECK (resultado IN ('encontrado','ausente')),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['saldo_acceso','remoto_consulta_acceso','marcaje_remoto_autorizado','marcaje_remoto_recuperacion'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY lectura_propia ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
  EXECUTE format('CREATE POLICY adicion_propia ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
END $seguridad$;

-- Denegaciones de la frontera HTTP (sin identidad, sin empleado, acceso
-- denegado o dependencia caída). Solo referencias opacas; escribe el auditor.
CREATE TABLE vec_cronos_v1.denegacion_frontera (
  denegacion_ref text PRIMARY KEY DEFAULT 'denegacion:cronos:'||gen_random_uuid()::text,
  correlacion_ref text NOT NULL CHECK (correlacion_ref ~ '^corr_([0-9a-f]{32}|no_disponible)$'),
  motivo text NOT NULL CHECK (motivo IN ('autenticacion_requerida','acceso_denegado','sin_empleado','empleado_ambiguo','dependencia')),
  ruta text NOT NULL CHECK (ruta IN ('/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
    '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo','otra')),
  metodo text NOT NULL CHECK (metodo IN ('GET','POST','otro')),
  actor_ref text CHECK (actor_ref IS NULL OR actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE vec_cronos_v1.denegacion_frontera ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.denegacion_frontera FORCE ROW LEVEL SECURITY;
CREATE POLICY registrar_denegacion_nominal ON vec_cronos_v1.denegacion_frontera
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.denegacion_frontera
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON vec_cronos_v1.denegacion_frontera FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

-- Auxiliares internas, sin EXECUTE para ningún LOGIN.
CREATE FUNCTION vec_cronos_v1.acreditar_empleado_contexto_v1(x jsonb,p_actor text,p_perfil text,p_empleado text,p_ahora timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE vinculo jsonb;
BEGIN
 IF x->>'principal_ref' IS DISTINCT FROM p_actor OR x->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
   RAISE EXCEPTION 'contexto Cronos divergente' USING ERRCODE='PC003';
 END IF;
 -- V3 verifica la procedencia de este contexto completo. Se exige además un
 -- único vínculo empleado activo en el instante vivo, nunca del cliente.
 BEGIN
   SELECT e INTO STRICT vinculo FROM jsonb_array_elements(x->'vinculos') e
    WHERE e->>'tipo'='empleado' AND e->>'estado'='activo'
      AND (e->>'vigente_desde')::timestamptz<=p_ahora AND p_ahora<(e->>'vigente_hasta')::timestamptz;
 EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN
   RAISE EXCEPTION 'vínculo Cronos ambiguo o caducado' USING ERRCODE='PC003';
 END;
 IF vinculo->>'referencia' IS DISTINCT FROM p_empleado THEN
   RAISE EXCEPTION 'empleado Cronos divergente' USING ERRCODE='PC003';
 END IF;
 RETURN vinculo;
END $f$;

CREATE FUNCTION vec_cronos_v1.vence_autorizacion_v1(c jsonb,d jsonb,x jsonb,vinculo jsonb)
RETURNS timestamptz LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE plazos timestamptz[];
BEGIN
 plazos:=ARRAY[(c->>'expira_en')::timestamptz,(c->>'decision_valida_hasta')::timestamptz,
   (c->>'configuracion_expira_en')::timestamptz,(c->>'raiz_valida_hasta')::timestamptz,
   (d->>'valida_hasta')::timestamptz,(x->>'vigente_hasta')::timestamptz,(vinculo->>'vigente_hasta')::timestamptz];
 IF array_position(plazos,NULL) IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia Cronos incompleta' USING ERRCODE='PC003';
 END IF;
 RETURN (SELECT min(p) FROM unnest(plazos) p);
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'vigencia Cronos inválida' USING ERRCODE='PC003';
END $f$;

-- Reproduce RecursoAutorizable.HuellaContextoAutorizacionSHA256 de Go para
-- ámbito empleado_ref y atributo material_sha256.
CREATE FUNCTION vec_cronos_v1.huella_contexto_empleado_v1(p_empleado text,p_material_sha256 text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{"empleado_ref":'||to_jsonb(p_empleado)::text||'},"atributos":{"material_sha256":"'||p_material_sha256||'"}}','UTF8')),'hex')
$f$;

CREATE FUNCTION vec_cronos_v1.comprobar_decision_cronos_v1(d jsonb,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_actor text,p_perfil text,p_huella text)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 IF d->>'accion' IS DISTINCT FROM p_accion OR d->>'modulo_id' IS DISTINCT FROM 'cronos'
    OR d->>'tipo_recurso' IS DISTINCT FROM p_tipo OR d->>'finalidad' IS DISTINCT FROM p_finalidad
    OR d->>'recurso_ref' IS DISTINCT FROM p_recurso OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM p_huella THEN
   RAISE EXCEPTION 'autorización Cronos divergente' USING ERRCODE='PC003';
 END IF;
END $f$;

CREATE FUNCTION vec_cronos_v1.material_canal_remoto_v1(m jsonb)
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_typeof(m->'canal')='object'
   AND (SELECT count(*) FROM jsonb_object_keys(m->'canal'))=4
   AND (m->'canal')-ARRAY['politica_version_ref','canal_ref','origen_ref','calidad_ref']='{}'::jsonb
   AND NOT EXISTS (SELECT 1 FROM jsonb_each(m->'canal') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string'
                    OR e.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,127}$')
   AND m#>>'{canal,origen_ref}'='remoto'
   AND EXISTS (SELECT 1 FROM vec_cronos_v1.clasificacion_canal c
     WHERE c.tipo_origen='remoto' AND c.politica_version_ref=m#>>'{canal,politica_version_ref}'
       AND c.canal_ref=m#>>'{canal,canal_ref}' AND c.origen_ref=m#>>'{canal,origen_ref}'
       AND c.calidad_ref=m#>>'{canal,calidad_ref}')
$f$;

-- Autorización de teletrabajo que cubre el instante, recortada por su
-- revocación. El disparador de 000005 impide que haya más de una.
CREATE FUNCTION vec_cronos_v1.teletrabajo_en_v1(p_empleado text,p_instante timestamptz,
  OUT autorizacion_ref text,OUT desde timestamptz,OUT hasta timestamptz)
LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
BEGIN
 SELECT a.autorizacion_ref,lower(a.periodo),least(upper(a.periodo),coalesce(r.efectiva_en,upper(a.periodo)))
   INTO autorizacion_ref,desde,hasta
   FROM vec_cronos_v1.teletrabajo_autorizacion a
   LEFT JOIN vec_cronos_v1.teletrabajo_revocacion r ON r.autorizacion_ref=a.autorizacion_ref
  WHERE a.empleado_ref=p_empleado AND a.periodo@>p_instante
    AND (r.efectiva_en IS NULL OR r.efectiva_en>p_instante);
END $f$;

-- Secuencia del último hecho de la persona. Una secuencia abierta antes del
-- periodo autorizado no se continúa desde la web: se corrige por solicitud.
CREATE FUNCTION vec_cronos_v1.secuencia_remota_v1(p_empleado text,p_desde timestamptz,
  OUT continuidad boolean,OUT permitidos text[],OUT ultimo_instante timestamptz)
LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE ultimo record; hay boolean;
BEGIN
 SELECT m.movimiento,m.instante_utc INTO ultimo FROM vec_cronos_v1.marcaje_original m
  WHERE m.empleado_ref=p_empleado ORDER BY m.instante_utc DESC,m.marcaje_ref DESC LIMIT 1;
 hay:=FOUND;
 ultimo_instante:=CASE WHEN hay THEN ultimo.instante_utc END;
 IF NOT hay OR ultimo.movimiento='salida' THEN
   continuidad:=true; permitidos:=ARRAY['entrada'];
 ELSIF ultimo.instante_utc<p_desde THEN
   continuidad:=false; permitidos:=ARRAY[]::text[];
 ELSIF ultimo.movimiento IN ('entrada','fin_pausa') THEN
   continuidad:=true; permitidos:=ARRAY['salida','inicio_pausa'];
 ELSE
   continuidad:=true; permitidos:=ARRAY['fin_pausa'];
 END IF;
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_saldo_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; recurso text;
 ahora timestamptz(6); vence timestamptz; consumo record; desde date; hasta date; resultado jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
   IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
      OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','desde','hasta','zona_horaria']<>'{}'::jsonb
      OR EXISTS (SELECT 1 FROM jsonb_each(m) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
      OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'zona_horaria','') NOT IN ('Europe/Madrid','Atlantic/Canary') THEN
     RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
   END IF;
   desde:=(m->>'desde')::date; hasta:=(m->>'hasta')::date;
   IF to_char(desde,'YYYY-MM-DD') IS DISTINCT FROM m->>'desde' OR to_char(hasta,'YYYY-MM-DD') IS DISTINCT FROM m->>'hasta'
      OR hasta<desde OR hasta-desde>366 THEN
     RAISE EXCEPTION 'periodo Cronos inválido' USING ERRCODE='PC001';
   END IF;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 recurso:='saldo:cronos:'||(m->>'empleado_ref');
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,'cronos.saldo.propio.consultar','saldo_propio','consultar_saldo_propio',
   recurso,m->>'actor_ref',m->>'perfil_ref',vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_cronos_saldo_propio_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM recurso
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella) THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 -- El consumidor pudo esperar bloqueos: revalidar plazos y vínculo con reloj vivo.
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 INSERT INTO vec_cronos_v1.saldo_acceso(decision_ref,empleado_ref,desde,hasta,zona_horaria,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(consumo.decision_ref,m->>'empleado_ref',desde,hasta,m->>'zona_horaria',huella,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 resultado:=vec_cronos_v1.consultar_libro_saldo_interno_v1(m->>'empleado_ref',desde,hasta,m->>'zona_horaria');
 IF clock_timestamp()>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 RETURN resultado;
END $f$;

-- Disponibilidad del fichaje remoto. Con clave, concilia una operación ya
-- registrada (replay) bajo el mismo bloqueo clave+empleado que el registro.
CREATE FUNCTION vec_cronos_v1.consultar_estado_remoto_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; recurso text; clave text;
 ahora timestamptz(6); instante timestamptz(6); vence timestamptz; consumo record; tele record; sec record;
 previa vec_cronos_v1.marcaje_original%ROWTYPE; continuidad boolean:=false; permitidos text[]:=ARRAY[]::text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
   IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
      OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','instante_utc','canal']<>'{}'::jsonb
      OR EXISTS (SELECT 1 FROM jsonb_each(m-'canal') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
      OR NOT vec_cronos_v1.material_canal_remoto_v1(m)
      OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
      OR (m->>'clave_operacion'<>'' AND m->>'clave_operacion' !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
      OR coalesce(m->>'instante_utc','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
     RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
   END IF;
   instante:=(m->>'instante_utc')::timestamptz;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF instante>ahora OR ahora-instante>interval '2 minutes' THEN
   RAISE EXCEPTION 'instante Cronos inválido' USING ERRCODE='PC001';
 END IF;
 clave:=nullif(m->>'clave_operacion','');
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 recurso:='teletrabajo:cronos:'||(m->>'empleado_ref');
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,'cronos.marcaje.remoto.disponibilidad.consultar','marcaje_remoto_disponibilidad',
   'consultar_disponibilidad_marcaje_remoto',recurso,m->>'actor_ref',m->>'perfil_ref',vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella));
 IF clave IS NOT NULL THEN
   PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:clave:'||clave,0));
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:empleado:'||(m->>'empleado_ref'),0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_cronos_disponibilidad_remota_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM recurso
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella) THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 SELECT * INTO tele FROM vec_cronos_v1.teletrabajo_en_v1(m->>'empleado_ref',instante);
 IF tele.autorizacion_ref IS NOT NULL THEN
   IF clave IS NOT NULL THEN
     SELECT o.* INTO previa FROM vec_cronos_v1.marcaje_original o
       JOIN vec_cronos_v1.marcaje_remoto_autorizado r ON r.marcaje_ref=o.marcaje_ref
      WHERE o.clave_operacion=clave;
   END IF;
   IF previa.marcaje_ref IS NOT NULL THEN
     continuidad:=true; permitidos:=ARRAY[previa.movimiento];
   ELSE
     SELECT * INTO sec FROM vec_cronos_v1.secuencia_remota_v1(m->>'empleado_ref',tele.desde);
     continuidad:=sec.continuidad AND (sec.ultimo_instante IS NULL OR sec.ultimo_instante<instante);
     permitidos:=CASE WHEN continuidad THEN sec.permitidos ELSE ARRAY[]::text[] END;
   END IF;
 END IF;
 INSERT INTO vec_cronos_v1.remoto_consulta_acceso(decision_ref,empleado_ref,clave_operacion,autorizacion_ref,continuidad_confirmada,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(consumo.decision_ref,m->>'empleado_ref',clave,tele.autorizacion_ref,continuidad,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 IF clock_timestamp()>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 IF tele.autorizacion_ref IS NULL THEN
   RETURN jsonb_build_object('autorizado',false,'continuidad_confirmada',false,'movimientos_permitidos','[]'::jsonb);
 END IF;
 RETURN jsonb_build_object('autorizado',true,'desde',tele.desde,'hasta',tele.hasta,
   'continuidad_confirmada',continuidad,'movimientos_permitidos',to_jsonb(permitidos));
END $f$;

-- Fichaje remoto: mismas garantías que 000002 más teletrabajo vigente en el
-- instante del servidor y secuencia, bajo bloqueos clave, empleado y
-- teletrabajo, en la transacción que consume V3 y escribe hecho, recibo,
-- historia y outbox.
CREATE FUNCTION vec_cronos_v1.registrar_marcaje_remoto_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; ref text;
 instante timestamptz(6); ahora timestamptz(6); vence timestamptz; consumo record; tele record; sec record;
 previa vec_cronos_v1.marcaje_original%ROWTYPE; recibo jsonb; recibo_ref text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
   IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>7
      OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','movimiento','instante_utc','canal']<>'{}'::jsonb
      OR EXISTS (SELECT 1 FROM jsonb_each(m-'canal') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
      OR NOT vec_cronos_v1.material_canal_remoto_v1(m)
      OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
      OR coalesce(m->>'movimiento','') NOT IN ('entrada','salida','inicio_pausa','fin_pausa')
      OR coalesce(m->>'instante_utc','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,6})?Z$' THEN
     RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
   END IF;
   instante:=(m->>'instante_utc')::timestamptz;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF instante>ahora OR ahora-instante>interval '2 minutes' THEN
   RAISE EXCEPTION 'instante Cronos inválido' USING ERRCODE='PC001';
 END IF;
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 ref:='marcaje:cronos:'||(m->>'clave_operacion');
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,'cronos.marcaje.propio.registrar','marcaje_propio','registrar_marcaje_propio',
   ref,m->>'actor_ref',m->>'perfil_ref',vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella));
 -- Esperar al escritor de esta clave y de esta persona ANTES de consumir V3.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:clave:'||(m->>'clave_operacion'),0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:empleado:'||(m->>'empleado_ref'),0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM ref
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella) THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 SELECT * INTO previa FROM vec_cronos_v1.marcaje_original WHERE clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF previa.actor_ref IS DISTINCT FROM m->>'actor_ref' OR previa.perfil_ref IS DISTINCT FROM m->>'perfil_ref'
      OR previa.empleado_ref IS DISTINCT FROM m->>'empleado_ref'
      OR NOT EXISTS (SELECT 1 FROM vec_cronos_v1.marcaje_remoto_autorizado r WHERE r.marcaje_ref=previa.marcaje_ref) THEN
     RAISE EXCEPTION 'operación Cronos ajena' USING ERRCODE='PC003';
   END IF;
   IF (previa.material::jsonb-'instante_utc') IS DISTINCT FROM (m-'instante_utc') THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.marcaje_acceso(decision_ref,marcaje_ref,empleado_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
   VALUES(consumo.decision_ref,ref,m->>'empleado_ref',consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
   IF clock_timestamp()>=vence THEN
     RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
   END IF;
   RETURN previa.recibo_json||jsonb_build_object('replay',true);
 END IF;
 -- Serializa con altas y revocaciones de teletrabajo de la misma persona.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:teletrabajo:'||(m->>'empleado_ref'),0));
 SELECT * INTO tele FROM vec_cronos_v1.teletrabajo_en_v1(m->>'empleado_ref',instante);
 IF tele.autorizacion_ref IS NULL THEN
   RAISE EXCEPTION 'teletrabajo Cronos no autorizado' USING ERRCODE='PC004';
 END IF;
 SELECT * INTO sec FROM vec_cronos_v1.secuencia_remota_v1(m->>'empleado_ref',tele.desde);
 IF NOT sec.continuidad OR (sec.ultimo_instante IS NOT NULL AND sec.ultimo_instante>=instante) THEN
   RAISE EXCEPTION 'continuidad Cronos no confirmada' USING ERRCODE='PC006';
 END IF;
 IF NOT (m->>'movimiento'=ANY(sec.permitidos)) THEN
   RAISE EXCEPTION 'secuencia Cronos no permitida' USING ERRCODE='PC005';
 END IF;
 recibo_ref:='recibo:cronos:'||gen_random_uuid()::text;
 recibo:=jsonb_build_object('referencia',recibo_ref,'instante_utc',instante,'marcaje_original_ref',ref,'replay',false);
 INSERT INTO vec_cronos_v1.marcaje_remoto_autorizado(marcaje_ref,empleado_ref,clave_operacion,autorizacion_ref,decision_ref,registrada_en)
 VALUES(ref,m->>'empleado_ref',m->>'clave_operacion',tele.autorizacion_ref,consumo.decision_ref,ahora);
 INSERT INTO vec_cronos_v1.marcaje_original(
   marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,
   movimiento,instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
 VALUES(ref,m->>'empleado_ref',m->>'clave_operacion',m->>'actor_ref',m->>'perfil_ref',p_material,huella,
   m->>'movimiento',instante,recibo_ref,recibo,consumo.auditoria_ref,consumo.decision_ref,consumo.consumo_huella_sha256,ahora);
 INSERT INTO vec_cronos_v1.marcaje_historia(marcaje_ref,empleado_ref,version,material_sha256,registrada_en)
 VALUES(ref,m->>'empleado_ref',1,huella,ahora);
 INSERT INTO vec_cronos_v1.marcaje_outbox(evento_ref,marcaje_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,ref,m->>'empleado_ref','cronos.marcaje.propio.registrado',
   jsonb_build_object('marcaje_original_ref',ref,'recibo_ref',recibo_ref,'version',1,'origen','remoto'),ahora);
 IF clock_timestamp()>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 RETURN recibo;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

-- Lectura auditada del recibo de un fichaje remoto propio. No comprueba el
-- teletrabajo actual ni crea hechos. La ausencia sólo se afirma tras esperar
-- al escritor de la misma clave y queda registrada con su decisión.
CREATE FUNCTION vec_cronos_v1.recuperar_marcaje_remoto_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; ref text;
 ahora timestamptz(6); vence timestamptz; consumo record; previa vec_cronos_v1.marcaje_original%ROWTYPE;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
   IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
      OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','movimiento','canal']<>'{}'::jsonb
      OR EXISTS (SELECT 1 FROM jsonb_each(m-'canal') e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
      OR NOT vec_cronos_v1.material_canal_remoto_v1(m)
      OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
      OR coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
      OR coalesce(m->>'movimiento','') NOT IN ('entrada','salida','inicio_pausa','fin_pausa') THEN
     RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
   END IF;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 ref:='marcaje:cronos:'||(m->>'clave_operacion');
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,'cronos.marcaje.remoto.recibo.consultar','marcaje_remoto_recibo',
   'recuperar_recibo_marcaje_remoto',ref,m->>'actor_ref',m->>'perfil_ref',vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:clave:'||(m->>'clave_operacion'),0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_cronos_recibo_remoto_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM ref
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',huella) THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 -- La RLS limita la búsqueda a la persona: una clave ajena se ve ausente.
 SELECT o.* INTO previa FROM vec_cronos_v1.marcaje_original o
   JOIN vec_cronos_v1.marcaje_remoto_autorizado r ON r.marcaje_ref=o.marcaje_ref
  WHERE o.clave_operacion=m->>'clave_operacion';
 IF previa.marcaje_ref IS NULL THEN
   INSERT INTO vec_cronos_v1.marcaje_remoto_recuperacion(decision_ref,empleado_ref,clave_operacion,resultado,auditoria_ref,consumo_huella_sha256,consultada_en)
   VALUES(consumo.decision_ref,m->>'empleado_ref',m->>'clave_operacion','ausente',consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
   IF clock_timestamp()>=vence THEN
     RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
   END IF;
   RETURN jsonb_build_object('ausente',true);
 END IF;
 IF previa.actor_ref IS DISTINCT FROM m->>'actor_ref' THEN
   RAISE EXCEPTION 'operación Cronos ajena' USING ERRCODE='PC003';
 END IF;
 IF previa.movimiento IS DISTINCT FROM m->>'movimiento' THEN
   RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
 END IF;
 INSERT INTO vec_cronos_v1.marcaje_remoto_recuperacion(decision_ref,empleado_ref,clave_operacion,resultado,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(consumo.decision_ref,m->>'empleado_ref',m->>'clave_operacion','encontrado',consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 IF clock_timestamp()>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 RETURN previa.recibo_json||jsonb_build_object('replay',true);
END $f$;

CREATE FUNCTION vec_cronos_v1.registrar_denegacion_frontera_v1(
  p_correlacion_ref text,p_motivo text,p_ruta text,p_metodo text,p_actor_ref text
) RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE ref text;
BEGIN
 IF session_user=current_user OR NOT pg_has_role(session_user,'vec_cronos_v1_auditor','MEMBER')
    OR pg_has_role(session_user,'vec_cronos_v1_ejecutor','MEMBER') THEN
   RAISE EXCEPTION 'auditor Cronos inválido' USING ERRCODE='42501';
 END IF;
 -- Sin RETURNING: la tabla no concede lectura ni siquiera al propietario.
 ref:='denegacion:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.denegacion_frontera(denegacion_ref,correlacion_ref,motivo,ruta,metodo,actor_ref)
 VALUES(ref,p_correlacion_ref,p_motivo,p_ruta,p_metodo,nullif(p_actor_ref,''));
 RETURN ref;
END $f$;

-- El marcaje remoto sólo entra con su fila de autorización de teletrabajo,
-- que únicamente escribe registrar_marcaje_remoto_v1. El registro de terminal
-- de 000002 sigue limitado a canales clasificados como terminal.
CREATE OR REPLACE FUNCTION vec_cronos_v1.bloquear_marcaje_remoto_sin_consumidor_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF NEW.tipo_origen IN ('terminal','remoto') AND EXISTS (SELECT 1 FROM vec_cronos_v1.clasificacion_canal c
      WHERE c.tipo_origen=NEW.tipo_origen
        AND c.politica_version_ref=NEW.material::jsonb #>> '{canal,politica_version_ref}'
        AND c.canal_ref=NEW.material::jsonb #>> '{canal,canal_ref}'
        AND c.origen_ref=NEW.material::jsonb #>> '{canal,origen_ref}'
        AND c.calidad_ref=NEW.material::jsonb #>> '{canal,calidad_ref}')
    AND (NEW.tipo_origen='terminal' OR EXISTS (SELECT 1 FROM vec_cronos_v1.marcaje_remoto_autorizado r
      WHERE r.marcaje_ref=NEW.marcaje_ref AND r.empleado_ref=NEW.empleado_ref AND r.clave_operacion=NEW.clave_operacion
        AND r.decision_ref=NEW.decision_ref)) THEN
   RETURN NEW;
 END IF;
 RAISE EXCEPTION 'canal Cronos no disponible' USING ERRCODE='PC003';
END $f$;

DO $acl$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.acreditar_empleado_contexto_v1(jsonb,text,text,text,timestamptz)',
   'vec_cronos_v1.vence_autorizacion_v1(jsonb,jsonb,jsonb,jsonb)',
   'vec_cronos_v1.huella_contexto_empleado_v1(text,text)',
   'vec_cronos_v1.comprobar_decision_cronos_v1(jsonb,text,text,text,text,text,text,text)',
   'vec_cronos_v1.material_canal_remoto_v1(jsonb)',
   'vec_cronos_v1.teletrabajo_en_v1(text,timestamptz)',
   'vec_cronos_v1.secuencia_remota_v1(text,timestamptz)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_estado_remoto_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.registrar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.recuperar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_cronos_v1.registrar_denegacion_frontera_v1(text,text,text,text,text) FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
 GRANT EXECUTE ON FUNCTION vec_cronos_v1.registrar_denegacion_frontera_v1(text,text,text,text,text) TO vec_cronos_v1_auditor;
END $acl$;
COMMIT;
