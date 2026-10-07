\set ON_ERROR_STOP on
-- BR4: preparación nominal V3 del borrador, consulta exacta y recuperación.
-- Acceso runtime sólo por el ejecutor nominal. AD144 aún debe ensayarse.
-- No instala ni reabre V1/V2; no contiene publicación ni activación formal.
BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_reglas_baremo:migracion:000004',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_reglas_baremo_propietario'
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND NOT rolcanlogin
   AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb)
 OR to_regclass('vec_bolsa_reglas_baremo.version_reglas_baremo') IS NULL
 OR to_regclass('vec_bolsa_reglas_baremo.estado_actual') IS NULL
 OR to_regclass('vec_bolsa_reglas_baremo.acceso_borrador_v3') IS NOT NULL
 THEN RAISE EXCEPTION 'BR4: preimagen propia incompatible' USING ERRCODE='55000'; END IF;
 -- Abrir una fachada anterior no puede quedar disimulado por el corte nuevo.
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.pronamespace='vec_bolsa_reglas_baremo'::regnamespace
   AND a.privilege_type='EXECUTE' AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'BR4: las puertas anteriores deben estar cerradas' USING ERRCODE='55000'; END IF;
END $pre$;

-- La auditoría de acceso incluye el consumo central de ESTA llamada. Un replay
-- añade acceso y conserva el recibo y consumo ORIGINAL del alta.
CREATE TABLE vec_bolsa_reglas_baremo.acceso_borrador_v3 (
 acceso_ref text PRIMARY KEY CHECK(acceso_ref ~ '^acceso:reglas-baremo:v3:[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 decision_huella_sha256 text NOT NULL CHECK(decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
 efecto_ref text NOT NULL,
 efecto_huella_sha256 text NOT NULL CHECK(efecto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 persona_ref text NOT NULL,
 perfil_ref text NOT NULL,
 convocatoria_ref text NOT NULL,
 expediente_ref text NOT NULL,
 operacion text NOT NULL CHECK(operacion IN ('alta_borrador','consultar_exacta','recuperar_recibo')),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 resultado text NOT NULL CHECK(resultado IN ('creado','replay','obtenida','no_encontrada','recuperado')),
 consumida_en timestamptz(6) NOT NULL CHECK(isfinite(consumida_en))
);

CREATE TABLE vec_bolsa_reglas_baremo.outbox_borrador_v3 (
 outbox_ref text PRIMARY KEY CHECK(outbox_ref ~ '^outbox:reglas-baremo:v3:[0-9a-f]{64}$'),
 acceso_original_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_reglas_baremo.acceso_borrador_v3(acceso_ref),
 tenant_id text NOT NULL CHECK(tenant_id='diputacion_granada'),
 contenido_ref text NOT NULL,
 contenido_version numeric(20,0) NOT NULL,
 revision numeric(20,0) NOT NULL CHECK(revision=1),
 huella_estado_sha256 text NOT NULL,
 esquema_evento text NOT NULL CHECK(esquema_evento='vec.bolsa.reglas-baremo.borrador-creado.v3'),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en)),
 FOREIGN KEY(tenant_id,contenido_ref,contenido_version,revision,huella_estado_sha256)
 REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(tenant_id,contenido_ref,contenido_version,revision,huella_estado_sha256)
);

CREATE TABLE vec_bolsa_reglas_baremo.recibo_borrador_v3 (
 -- La clave es global dentro del módulo: cambiar actor/ámbito con ella da conflicto.
 clave_operacion text PRIMARY KEY CHECK(clave_operacion ~ '^[0-9a-f]{32}$'),
 huella_solicitud_sha256 text NOT NULL CHECK(huella_solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 persona_ref text NOT NULL,
 convocatoria_ref text NOT NULL,
 expediente_ref text NOT NULL,
 motivo_canonico bytea NOT NULL CHECK(octet_length(motivo_canonico) BETWEEN 2 AND 65536),
 tenant_id text NOT NULL CHECK(tenant_id='diputacion_granada'),
 contenido_ref text NOT NULL,
 contenido_version numeric(20,0) NOT NULL,
 huella_contenido_sha256 text NOT NULL,
 revision numeric(20,0) NOT NULL CHECK(revision=1),
 huella_estado_sha256 text NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:reglas-baremo:v3:[0-9a-f]{64}$'),
 transaccion_ref text NOT NULL UNIQUE CHECK(transaccion_ref ~ '^transaccion:reglas-baremo:v3:[0-9a-f]{64}$'),
 acceso_original_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_reglas_baremo.acceso_borrador_v3(acceso_ref),
 outbox_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_reglas_baremo.outbox_borrador_v3(outbox_ref),
 recibo jsonb NOT NULL CHECK(jsonb_typeof(recibo)='object'),
 confirmada_en timestamptz(6) NOT NULL CHECK(isfinite(confirmada_en)),
 UNIQUE(tenant_id,contenido_ref,contenido_version),
 FOREIGN KEY(tenant_id,contenido_ref,contenido_version,revision,huella_estado_sha256)
 REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(tenant_id,contenido_ref,contenido_version,revision,huella_estado_sha256),
 FOREIGN KEY(tenant_id,contenido_ref,contenido_version,huella_contenido_sha256)
 REFERENCES vec_bolsa_reglas_baremo.contenido_reglas_baremo(tenant_id,contenido_ref,contenido_version,huella_contenido_sha256)
);

DO $cerrar$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['acceso_borrador_v3','outbox_borrador_v3','recibo_borrador_v3'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_reglas_baremo.%I FROM PUBLIC,vec_bolsa_reglas_baremo_ejecutor_gobierno,vec_bolsa_reglas_baremo_ejecutor_consulta,vec_bolsa_reglas_baremo_publicador_outbox',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_reglas_baremo.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE vec_bolsa_reglas_baremo.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_reglas_baremo.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_bolsa_reglas_baremo.%I FOR ALL TO vec_bolsa_reglas_baremo_propietario USING(current_user=''vec_bolsa_reglas_baremo_propietario'') WITH CHECK(current_user=''vec_bolsa_reglas_baremo_propietario'')',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_reglas_baremo.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_reglas_baremo.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable()',t);
 END LOOP;
END $cerrar$;

-- Solo coteja material; no acredita identidad, concesión ni canon semántico.
-- El adaptador debe restaurar la versión Go antes de invocar la fachada.
CREATE FUNCTION vec_bolsa_reglas_baremo.validar_material_borrador_v3(p_material bytea,p_motivo bytea)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE
SET search_path=pg_catalog SET timezone='UTC'
AS $fn$
DECLARE m jsonb; v jsonb; motivo jsonb; raw_version json; canon bytea; conjunto bytea;
 intencion text; estable text; huella text; instante timestamptz;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8388608
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'BR4: material fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN
  m:=convert_from(p_material,'UTF8')::jsonb;
  motivo:=convert_from(p_motivo,'UTF8')::jsonb;
  instante:=(m->>'solicitada_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'BR4: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR m-'esquema'-'operacion'-'accion'-'modulo_id'-'tipo_recurso'-'finalidad'-'persona_ref'-'perfil_ref'-'convocatoria_ref'-'expediente_ref'-'estado'-'estado_esperado'-'version_canonica'-'clave_operacion'-'huella_solicitud_sha256'-'motivo_canonico'-'solicitada_en' <> '{}'::jsonb
 OR (SELECT count(*) FROM jsonb_object_keys(m))<>17
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.gobierno-borrador.material.v3'
 OR m->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR m->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN m->>'operacion'='alta_borrador'
   THEN 'intencion_gobierno_reglas_baremo' ELSE 'version_reglas_baremo_gobernada' END)
 OR coalesce(m->>'operacion','') NOT IN ('alta_borrador','consultar_exacta','recuperar_recibo')
 OR coalesce(m->>'persona_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR NOT vec_bolsa_reglas_baremo.referencia_valida(m->>'perfil_ref')
 OR NOT vec_bolsa_reglas_baremo.referencia_valida(m->>'convocatoria_ref')
 OR NOT vec_bolsa_reglas_baremo.referencia_valida(m->>'expediente_ref')
 OR m->'estado_esperado' IS DISTINCT FROM 'null'::jsonb
 OR NOT isfinite(instante)
 OR jsonb_typeof(m->'estado') IS DISTINCT FROM 'object'
 OR (m->'estado')-'referencia'-'version'-'huella_contenido_sha256'-'revision'-'huella_estado_sha256' <> '{}'::jsonb
 OR (SELECT count(*) FROM jsonb_object_keys(m->'estado'))<>5
 OR NOT vec_bolsa_reglas_baremo.referencia_valida(m#>>'{estado,referencia}')
 OR coalesce(m#>>'{estado,version}','') !~ '^[1-9][0-9]{0,9}$'
 OR m#>>'{estado,revision}' IS DISTINCT FROM '1'
 OR NOT vec_bolsa_reglas_baremo.huella_sha256_valida(m#>>'{estado,huella_contenido_sha256}')
 OR NOT vec_bolsa_reglas_baremo.huella_sha256_valida(m#>>'{estado,huella_estado_sha256}')
 THEN RAISE EXCEPTION 'BR4: proyecciones incompatibles' USING ERRCODE='22023'; END IF;
 -- El formato ya comprobado permite convertir sin admitir exponentes ni
 -- decimales. El mismo límite autoritativo del almacén incluye 1000000000.
 IF NOT vec_bolsa_reglas_baremo.version_valida((m#>>'{estado,version}')::numeric) THEN
  RAISE EXCEPTION 'BR4: versión fuera de límites' USING ERRCODE='22023';
 END IF;
 BEGIN
  IF decode(m->>'motivo_canonico','base64') IS DISTINCT FROM p_motivo THEN
   RAISE EXCEPTION 'motivo divergente';
  END IF;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'BR4: motivo divergente' USING ERRCODE='22023'; END;
 IF m->>'operacion'='alta_borrador' THEN
  IF m->>'accion' IS DISTINCT FROM 'bolsa.reglas_baremo.borrador.crear'
  OR m->>'finalidad' IS DISTINCT FROM 'gobierno_reglas_baremo'
  OR coalesce(m->>'clave_operacion','') !~ '^[0-9a-f]{32}$'
  OR NOT vec_bolsa_reglas_baremo.huella_sha256_valida(m->>'huella_solicitud_sha256')
  THEN RAISE EXCEPTION 'BR4: intención inválida' USING ERRCODE='22023'; END IF;
  BEGIN
   canon:=decode(m->>'version_canonica','base64');
   raw_version:=convert_from(canon,'UTF8')::json;
   v:=raw_version::jsonb;
   -- json (no jsonb) conserva los bytes del contenido emitidos por Go.
   conjunto:=convert_to((raw_version->'contenido')::text,'UTF8');
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'BR4: propuesta inválida' USING ERRCODE='22023'; END;
  huella:=encode(sha256(canon),'hex');
  IF octet_length(canon) NOT BETWEEN 2 AND 5242880
  OR huella IS DISTINCT FROM m#>>'{estado,huella_estado_sha256}'
  OR v->>'esquema' IS DISTINCT FROM 'vec.bolsa.gobierno-reglas-baremo.v1'
  OR v->>'estado' IS DISTINCT FROM 'borrador' OR v->>'revision' IS DISTINCT FROM '1'
  OR v-'esquema'-'contenido'-'referencia_contenido'-'revision'-'estado'-'creada_por'-'creada_en'-'motivo_creacion' <> '{}'::jsonb
  OR v->>'creada_por' IS DISTINCT FROM m->>'persona_ref'
  OR (v->>'creada_en')::timestamptz IS DISTINCT FROM instante
  OR v#>>'{referencia_contenido,referencia}' IS DISTINCT FROM m#>>'{estado,referencia}'
  OR v#>>'{referencia_contenido,version}' IS DISTINCT FROM m#>>'{estado,version}'
  OR v#>>'{referencia_contenido,huella_sha256}' IS DISTINCT FROM m#>>'{estado,huella_contenido_sha256}'
  OR encode(sha256(conjunto),'hex') IS DISTINCT FROM m#>>'{estado,huella_contenido_sha256}'
  OR v#>>'{contenido,identidad,referencia}' IS DISTINCT FROM m#>>'{estado,referencia}'
  OR v#>>'{contenido,identidad,version}' IS DISTINCT FROM m#>>'{estado,version}'
  OR v#>>'{contenido,identidad,convocatoria_ref}' IS DISTINCT FROM m->>'convocatoria_ref'
  OR v#>>'{contenido,identidad,expediente_ref}' IS DISTINCT FROM m->>'expediente_ref'
  OR v#>>'{motivo_creacion,catalogo,referencia}' IS DISTINCT FROM motivo#>>'{referencia,catalogo_id}'
  OR v#>>'{motivo_creacion,catalogo,version}' IS DISTINCT FROM motivo#>>'{referencia,catalogo_version}'
  OR v#>>'{motivo_creacion,catalogo,huella_sha256}' IS DISTINCT FROM motivo#>>'{referencia,catalogo_huella_sha256}'
  OR v#>>'{motivo_creacion,clave}' IS DISTINCT FROM motivo#>>'{referencia,entrada_clave}'
  THEN RAISE EXCEPTION 'BR4: canon y proyecciones divergentes' USING ERRCODE='22023'; END IF;
  -- Orden de campos del DTO negocioAltaGobiernoV3; base64 PostgreSQL se
  -- compacta para coincidir con encoding/json Go. No se hashéa jsonb::text.
  intencion:='{"esquema":"vec.bolsa.gobierno-borrador.intencion.v3","operacion":"alta_borrador","persona_ref":'||to_json(m->>'persona_ref')::text||
   ',"convocatoria_ref":'||to_json(m->>'convocatoria_ref')::text||',"expediente_ref":'||to_json(m->>'expediente_ref')::text||
   ',"conjunto_canonico":"'||replace(encode(conjunto,'base64'),E'\n','')||'","motivo_canonico":"'||replace(encode(p_motivo,'base64'),E'\n','')||
   '","clave_operacion":'||to_json(m->>'clave_operacion')::text||'}';
  estable:=encode(sha256(convert_to(intencion,'UTF8')),'hex');
  IF estable IS DISTINCT FROM m->>'huella_solicitud_sha256' THEN
   RAISE EXCEPTION 'BR4: huella estable divergente' USING ERRCODE='22023';
  END IF;
 ELSE
  IF m->>'accion' IS DISTINCT FROM (CASE WHEN m->>'operacion'='recuperar_recibo'
    THEN 'bolsa.reglas_baremo.recibo.consultar' ELSE 'bolsa.reglas_baremo.version.consultar' END)
  OR m->>'finalidad' IS DISTINCT FROM 'consulta_gobierno_reglas_baremo'
  OR m->'version_canonica' IS DISTINCT FROM 'null'::jsonb
  OR (m->>'operacion'='consultar_exacta' AND
     (m->>'clave_operacion' IS DISTINCT FROM '' OR m->>'huella_solicitud_sha256' IS DISTINCT FROM ''))
  OR (m->>'operacion'='recuperar_recibo' AND
     (coalesce(m->>'clave_operacion','') !~ '^[0-9a-f]{32}$' OR NOT vec_bolsa_reglas_baremo.huella_sha256_valida(m->>'huella_solicitud_sha256')))
  THEN RAISE EXCEPTION 'BR4: consulta incompatible' USING ERRCODE='22023'; END IF;
 END IF;
 RETURN m;
END $fn$;
REVOKE ALL ON FUNCTION vec_bolsa_reglas_baremo.validar_material_borrador_v3(bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_reglas_baremo.operar_borrador_v3(
 p_material bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(resultado text,version_canonica bytea,recibo jsonb,acceso jsonb,replay boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $fn$
DECLARE m jsonb; d jsonb; c jsonb; consumo record; acceso_actual jsonb; codigo text;
 material_sha text; recurso_sha text; recurso_canon text; ref_recurso text; campos jsonb;
 ar text; rr text; tr text; ob text; instante timestamptz; canon bytea;
 historico vec_bolsa_reglas_baremo.recibo_borrador_v3%ROWTYPE;
 fuente vec_bolsa_reglas_baremo.version_reglas_baremo%ROWTYPE;
 existe boolean; version_existe boolean; replay_actual boolean:=false; recibo_actual jsonb;
 consumer regprocedure;
BEGIN
 IF current_user<>'vec_bolsa_reglas_baremo_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'BR4: requiere transacción SERIALIZABLE READ WRITE' USING ERRCODE='25000'; END IF;
 -- No hay sustituto V2 ni consumidor central genérico si AD144 está ausente.
 consumer:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF consumer IS NULL THEN RAISE EXCEPTION 'BR4: consumidor nominal no disponible' USING ERRCODE='55000'; END IF;
 IF NOT has_function_privilege(current_user,consumer,'EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=consumer AND prosecdef
   AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole)
 THEN RAISE EXCEPTION 'BR4: consumidor nominal incompatible' USING ERRCODE='55000'; END IF;
 m:=vec_bolsa_reglas_baremo.validar_material_borrador_v3(p_material,p_motivo);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'BR4: material V3 inválido' USING ERRCODE='22023'; END;
 material_sha:=encode(sha256(p_material),'hex');
 ref_recurso:=CASE WHEN m->>'operacion'='alta_borrador'
  THEN 'intencion-reglas-baremo:'||(m->>'huella_solicitud_sha256')
  ELSE 'reglas-baremo:'||(m#>>'{estado,huella_estado_sha256}') END;
 recurso_canon:='{"ambitos":{"convocatoria_ref":'||to_json(m->>'convocatoria_ref')::text||
  ',"expediente_ref":'||to_json(m->>'expediente_ref')::text||'},"atributos":{"material_sha256":"'||material_sha||'"}}';
 recurso_sha:=encode(sha256(convert_to(recurso_canon,'UTF8')),'hex');
 campos:=CASE m->>'operacion' WHEN 'alta_borrador' THEN '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb
   WHEN 'consultar_exacta' THEN '["estado_reglas_baremo"]'::jsonb ELSE '["estado_reglas_baremo","recibo"]'::jsonb END;
 IF d->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
 OR d->>'accion' IS DISTINCT FROM m->>'accion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM m->>'tipo_recurso' OR d->>'finalidad' IS DISTINCT FROM m->>'finalidad'
 OR d->>'recurso_ref' IS DISTINCT FROM ref_recurso OR c->>'efecto_ref' IS DISTINCT FROM ref_recurso
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
 OR c->>'operacion' IS DISTINCT FROM m->>'accion'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_reglas_baremo.gobierno_borrador.v3'
 OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'false'
 OR d->>'motivo_huella_sha256' IS DISTINCT FROM encode(sha256(p_motivo),'hex')
 THEN RAISE EXCEPTION 'BR4: material y concesión divergentes' USING ERRCODE='42501'; END IF;
 instante:=clock_timestamp();
 IF (m->>'solicitada_en')::timestamptz>instante OR instante-(m->>'solicitada_en')::timestamptz>interval '30 seconds'
 THEN RAISE EXCEPTION 'BR4: solicitud caducada' USING ERRCODE='42501'; END IF;
 -- Bloqueo estable antes de consumir autoridad; timeout limita la espera.
 PERFORM pg_advisory_xact_lock(hashtextextended('BR4:contenido:'||(m#>>'{estado,referencia}')||':'||(m#>>'{estado,version}'),0));
 IF m->>'operacion' IN ('alta_borrador','recuperar_recibo') THEN
  PERFORM pg_advisory_xact_lock(hashtextextended('BR4:intencion:'||(m->>'clave_operacion'),0));
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM ref_recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
 OR consumo.consumida_en IS NULL OR NOT isfinite(consumo.consumida_en)
 THEN RAISE EXCEPTION 'BR4: requiere consumo nominal nuevo' USING ERRCODE='42501'; END IF;
 instante:=clock_timestamp();
 IF instante>=(d->>'valida_hasta')::timestamptz
 OR instante-(m->>'solicitada_en')::timestamptz>interval '30 seconds'
 THEN RAISE EXCEPTION 'BR4: autorización caducada tras consumo' USING ERRCODE='42501'; END IF;
 ar:='acceso:reglas-baremo:v3:'||consumo.consumo_huella_sha256;
 acceso_actual:=jsonb_build_object('decision_ref',consumo.decision_ref,'decision_huella_sha256',encode(sha256(p_decision),'hex'),
  'efecto_ref',consumo.efecto_ref,'efecto_huella_sha256',consumo.huella_efecto_sha256,
  'consumo_huella_sha256',consumo.consumo_huella_sha256,'auditoria_ref',consumo.auditoria_ref,'consumida_en',consumo.consumida_en);
 IF m->>'operacion' IN ('alta_borrador','recuperar_recibo') THEN
  SELECT * INTO historico FROM vec_bolsa_reglas_baremo.recibo_borrador_v3 r WHERE r.clave_operacion=m->>'clave_operacion';
  existe:=FOUND;
  IF existe AND (historico.huella_solicitud_sha256 IS DISTINCT FROM m->>'huella_solicitud_sha256'
    OR historico.persona_ref IS DISTINCT FROM m->>'persona_ref'
    OR historico.convocatoria_ref IS DISTINCT FROM m->>'convocatoria_ref'
    OR historico.expediente_ref IS DISTINCT FROM m->>'expediente_ref'
    OR historico.contenido_ref IS DISTINCT FROM m#>>'{estado,referencia}'
    OR historico.contenido_version IS DISTINCT FROM (m#>>'{estado,version}')::numeric
    OR historico.huella_contenido_sha256 IS DISTINCT FROM m#>>'{estado,huella_contenido_sha256}'
    OR (m->>'operacion'='alta_borrador' AND historico.motivo_canonico IS DISTINCT FROM p_motivo)
    OR (m->>'operacion'='recuperar_recibo' AND historico.huella_estado_sha256 IS DISTINCT FROM m#>>'{estado,huella_estado_sha256}'))
  THEN RAISE EXCEPTION 'BR4: clave de intención con otro material' USING ERRCODE='23505'; END IF;
 END IF;
 IF m->>'operacion'='alta_borrador' AND NOT existe THEN
  -- CAS de ausencia: no se adopta un alta histórica ajena ni otro borrador.
  IF EXISTS(SELECT 1 FROM vec_bolsa_reglas_baremo.contenido_reglas_baremo v
    WHERE v.tenant_id='diputacion_granada' AND v.contenido_ref=m#>>'{estado,referencia}' AND v.contenido_version=(m#>>'{estado,version}')::numeric)
  THEN RAISE EXCEPTION 'BR4: contenido existente sin esta intención' USING ERRCODE='40001'; END IF;
  canon:=decode(m->>'version_canonica','base64'); codigo:='creado';
 ELSE
  SELECT v.* INTO fuente FROM vec_bolsa_reglas_baremo.version_reglas_baremo v
   WHERE v.tenant_id='diputacion_granada' AND v.contenido_ref=m#>>'{estado,referencia}'
   AND v.contenido_version=(m#>>'{estado,version}')::numeric AND v.revision=1
   AND v.huella_contenido_sha256=m#>>'{estado,huella_contenido_sha256}'
   AND v.huella_estado_sha256=CASE WHEN m->>'operacion'='alta_borrador' THEN historico.huella_estado_sha256 ELSE m#>>'{estado,huella_estado_sha256}' END;
  version_existe:=FOUND;
  IF version_existe THEN
   canon:=fuente.version_canonica;
   IF encode(sha256(canon),'hex') IS DISTINCT FROM fuente.huella_estado_sha256
   OR convert_from(canon,'UTF8')::jsonb#>>'{contenido,identidad,convocatoria_ref}' IS DISTINCT FROM m->>'convocatoria_ref'
   OR convert_from(canon,'UTF8')::jsonb#>>'{contenido,identidad,expediente_ref}' IS DISTINCT FROM m->>'expediente_ref'
   THEN RAISE EXCEPTION 'BR4: fuente y ámbito divergentes' USING ERRCODE='55000'; END IF;
  END IF;
  IF m->>'operacion'='alta_borrador' THEN
   IF NOT version_existe THEN RAISE EXCEPTION 'BR4: recibo sin estado original' USING ERRCODE='55000'; END IF;
   codigo:='replay'; replay_actual:=true; recibo_actual:=historico.recibo;
  ELSIF m->>'operacion'='recuperar_recibo' THEN
   codigo:=CASE WHEN existe AND version_existe THEN 'recuperado' ELSE 'no_encontrada' END;
   IF codigo='recuperado' THEN recibo_actual:=historico.recibo; ELSE canon:=NULL; END IF;
  ELSE codigo:=CASE WHEN version_existe THEN 'obtenida' ELSE 'no_encontrada' END;
  END IF;
 END IF;
 INSERT INTO vec_bolsa_reglas_baremo.acceso_borrador_v3 VALUES(ar,consumo.decision_ref,encode(sha256(p_decision),'hex'),
  consumo.efecto_ref,consumo.huella_efecto_sha256,consumo.consumo_huella_sha256,consumo.auditoria_ref,
  m->>'persona_ref',m->>'perfil_ref',m->>'convocatoria_ref',m->>'expediente_ref',m->>'operacion',material_sha,codigo,consumo.consumida_en);
 IF codigo='creado' THEN
  INSERT INTO vec_bolsa_reglas_baremo.contenido_reglas_baremo VALUES('diputacion_granada',m#>>'{estado,referencia}',
   (m#>>'{estado,version}')::numeric,m#>>'{estado,huella_contenido_sha256}',consumo.consumida_en);
  INSERT INTO vec_bolsa_reglas_baremo.version_reglas_baremo VALUES('diputacion_granada',m#>>'{estado,referencia}',
   (m#>>'{estado,version}')::numeric,m#>>'{estado,huella_contenido_sha256}',1,'borrador',canon,m#>>'{estado,huella_estado_sha256}',
   'alta_borrador','intencion:reglas-baremo:v3:'||(m->>'huella_solicitud_sha256'),3,m->>'huella_solicitud_sha256',consumo.consumida_en);
  INSERT INTO vec_bolsa_reglas_baremo.estado_actual VALUES('diputacion_granada',m#>>'{estado,referencia}',
   (m#>>'{estado,version}')::numeric,m#>>'{estado,huella_contenido_sha256}',1,m#>>'{estado,huella_estado_sha256}',consumo.consumida_en);
  rr:='recibo:reglas-baremo:v3:'||(m->>'huella_solicitud_sha256');
  tr:='transaccion:reglas-baremo:v3:'||consumo.consumo_huella_sha256;
  ob:='outbox:reglas-baremo:v3:'||(m->>'huella_solicitud_sha256');
  recibo_actual:=jsonb_build_object('recibo_ref',rr,'clave_operacion',m->>'clave_operacion','huella_solicitud_sha256',m->>'huella_solicitud_sha256',
   'estado',m->'estado','transaccion_ref',tr,'auditoria_ref',consumo.auditoria_ref,'outbox_ref',ob,
   'consumo_original',acceso_actual,'confirmada_en',consumo.consumida_en);
  INSERT INTO vec_bolsa_reglas_baremo.outbox_borrador_v3 VALUES(ob,ar,'diputacion_granada',m#>>'{estado,referencia}',
   (m#>>'{estado,version}')::numeric,1,m#>>'{estado,huella_estado_sha256}','vec.bolsa.reglas-baremo.borrador-creado.v3',
   jsonb_build_object('esquema','vec.bolsa.reglas-baremo.borrador-creado.v3','recibo_ref',rr,'estado',m->'estado'),consumo.consumida_en);
  INSERT INTO vec_bolsa_reglas_baremo.recibo_borrador_v3 VALUES(m->>'clave_operacion',m->>'huella_solicitud_sha256',
   m->>'persona_ref',m->>'convocatoria_ref',m->>'expediente_ref',p_motivo,'diputacion_granada',m#>>'{estado,referencia}',
   (m#>>'{estado,version}')::numeric,m#>>'{estado,huella_contenido_sha256}',1,m#>>'{estado,huella_estado_sha256}',rr,tr,ar,ob,recibo_actual,consumo.consumida_en);
 END IF;
 RETURN QUERY SELECT codigo,canon,recibo_actual,acceso_actual,replay_actual;
END $fn$;
REVOKE ALL ON FUNCTION vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
-- El LOGIN hereda únicamente este ejecutor; la guarda central AD144 vuelve
-- a comprobar esa membresía, la sesión y los materiales atestados. No accede
-- al núcleo, a las tablas ni a las puertas V1/V2.
GRANT USAGE ON SCHEMA vec_bolsa_reglas_baremo TO vec_bolsa_reglas_baremo_ejecutor_gobierno;
GRANT EXECUTE ON FUNCTION vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_reglas_baremo_ejecutor_gobierno;
DO $acl_runtime$
DECLARE f oid:='vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        ejecutor oid:='vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole;
BEGIN
 IF NOT has_schema_privilege(ejecutor,'vec_bolsa_reglas_baremo','USAGE')
 OR NOT has_function_privilege(ejecutor,f,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,ejecutor) OR a.grantor<>p.proowner
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.pronamespace='vec_bolsa_reglas_baremo'::regnamespace AND p.oid<>f
     AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'BR4: ACL nominal runtime incompatible' USING ERRCODE='55000'; END IF;
END $acl_runtime$;
COMMIT;
