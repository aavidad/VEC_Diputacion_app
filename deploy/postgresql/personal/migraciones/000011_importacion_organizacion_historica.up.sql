\set ON_ERROR_STOP on
-- B3. Preparación y conciliación gobernadas. La publicación carece todavía de
-- una autoridad durable de fuente/acto/custodia verificable por Personal y
-- permanece cerrada. No se concede INSERT/UPDATE/DELETE a la cuenta de app.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000011:importacion-organizacion:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.org_nodo_historia') IS NULL
    OR to_regclass('vec_personal.importacion_organizacion_revision') IS NOT NULL
    OR to_regprocedure('vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
  RAISE EXCEPTION 'Personal 000011: dependencias o preimagen incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_personal.importacion_organizacion_revision (
 lote_ref text NOT NULL CHECK(lote_ref~'^lote:[0-9a-f-]{36}$'),
 revision bigint NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 fase text NOT NULL CHECK(fase IN ('preparar','conciliar')),
 estado text NOT NULL CHECK(estado IN ('preparacion_no_autoritativa','conciliacion_pendiente','conciliada')),
 manifiesto jsonb NOT NULL CHECK(jsonb_typeof(manifiesto)='object'),
 hechos jsonb NOT NULL CHECK(jsonb_typeof(hechos)='array' AND jsonb_array_length(hechos) BETWEEN 1 AND 3000),
 hechos_huella_sha256 text NOT NULL CHECK(hechos_huella_sha256~'^[0-9a-f]{64}$'),
 decisiones jsonb NOT NULL CHECK(jsonb_typeof(decisiones)='array' AND jsonb_array_length(decisiones)<=3000),
 fuente_ref text NOT NULL CHECK(length(fuente_ref) BETWEEN 3 AND 256),
 fuente_version text NOT NULL CHECK(length(fuente_version) BETWEEN 1 AND 160),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256~'^[0-9a-f]{64}$'),
 version_fuente_ref uuid NOT NULL,
 version_fuente_revision integer NOT NULL CHECK(version_fuente_revision>0),
 catalogo_unidades jsonb NOT NULL CHECK(jsonb_typeof(catalogo_unidades)='object'),
 catalogo_clasificaciones jsonb NOT NULL CHECK(jsonb_typeof(catalogo_clasificaciones)='object'),
 actor_ref text NOT NULL CHECK(length(actor_ref) BETWEEN 3 AND 256),
 perfil_ref text NOT NULL CHECK(length(perfil_ref) BETWEEN 3 AND 256),
 recibo_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(lote_ref,revision),
 CHECK((fase='preparar')=(revision=1))
);
CREATE INDEX importacion_organizacion_ambito_idx ON vec_personal.importacion_organizacion_revision(organismo_ref,lote_ref,revision DESC);

CREATE TABLE vec_personal.importacion_organizacion_recibo (
 clave_idempotencia uuid PRIMARY KEY,
 lote_ref text NOT NULL,
 revision bigint NOT NULL,
 fase text NOT NULL CHECK(fase IN ('preparar','conciliar')),
 material text NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 4194304),
 material_huella_sha256 text NOT NULL CHECK(material_huella_sha256~'^[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK(length(actor_ref) BETWEEN 3 AND 256),
 decision_ref text NOT NULL CHECK(length(decision_ref) BETWEEN 3 AND 256),
 auditoria_ref text NOT NULL CHECK(length(auditoria_ref) BETWEEN 3 AND 256),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^recibo:[0-9a-f-]{36}$'),
 recibo_json jsonb NOT NULL CHECK(jsonb_typeof(recibo_json)='object'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(lote_ref,revision) REFERENCES vec_personal.importacion_organizacion_revision(lote_ref,revision),
 UNIQUE(lote_ref,revision)
);
DO $cerrar$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['importacion_organizacion_revision','importacion_organizacion_recibo'] LOOP
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_interno ON vec_personal.%I FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true)',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',t);
 END LOOP;
END $cerrar$;

CREATE FUNCTION vec_personal.ejecutar_importacion_organizacion_v1(
 p_material text,p_acreditacion jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $fn$
DECLARE
 m jsonb; mf jsonb; d jsonb; c jsonb; hechos jsonb; nuevas jsonb; prev vec_personal.importacion_organizacion_revision%ROWTYPE;
 v_previo vec_personal.importacion_organizacion_recibo%ROWTYPE;
 v_consumo record; v_fase text; v_estado text; v_org text; v_fuente text; v_ref text;
 v_clave uuid; v_lote text; v_revision_esperada bigint; v_revision_nueva bigint;
 v_sha text; v_hechos_sha text; v_contexto text; v_contexto_sha text;
 v_decisiones jsonb; v_ahora timestamptz(6); v_recibo_ref text; v_recibo jsonb;
 v_contar integer;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4194304
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'importación de organización denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; mf:=m->'manifiesto'; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  v_fase:=m->>'fase'; v_org:=mf->>'organismo_ref'; v_fuente:=mf->>'fuente_ref';
  v_clave:=(m->>'clave_idempotencia')::uuid;
  v_revision_esperada:=(m->>'revision_esperada')::bigint;
  v_lote:=m->>'lote_ref'; hechos:=m->'hechos'; nuevas:=m->'decisiones';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de importación inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(mf) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
     'actor_ref','clave_idempotencia','contexto_ref','correlacion_ref','decisiones','esquema',
     'fase','hechos','lote_ref','manifiesto','perfil_ref','perfil_version','persona_version',
     'revision_esperada','revisor_actor_ref']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.importacion-organizacion.v1'
    OR v_fase NOT IN ('preparar','conciliar','publicar')
    OR v_org IS NULL OR v_org !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR v_fuente IS NULL OR length(v_fuente) NOT BETWEEN 3 AND 256
    OR v_clave IS NULL OR v_clave::text !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    OR v_revision_esperada IS NULL OR v_revision_esperada<0
    OR jsonb_typeof(hechos) IS DISTINCT FROM 'array' OR jsonb_typeof(nuevas) IS DISTINCT FROM 'array'
    OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'importacion_organizacion_historica'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.organizacion_historica.importar.v1'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR d->>'accion' IS DISTINCT FROM 'personal.organizacion_historica.'||v_fase
    OR d->>'finalidad' IS DISTINCT FROM v_fase||'_organizacion_historica' THEN
  RAISE EXCEPTION 'material o autorización de importación divergente' USING ERRCODE='42501';
 END IF;
 -- La publicación nunca acepta una acreditación enviada como JSON del caller.
 -- Queda para una migración que pueda verificar la autoridad fuente en la TX.
 IF v_fase='publicar' THEN
  RAISE EXCEPTION 'fuente y acto sin acreditación durable compuesta' USING ERRCODE='42501';
 END IF;
 IF p_acreditacion IS NOT NULL
    OR (v_fase='preparar' AND (v_lote IS NULL OR v_lote<>'' OR v_revision_esperada<>0 OR jsonb_array_length(hechos) NOT BETWEEN 1 AND 3000 OR jsonb_array_length(nuevas)<>0))
    OR (v_fase='conciliar' AND (v_lote IS NULL OR v_lote !~ '^lote:[0-9a-f-]{36}$' OR v_revision_esperada<1 OR jsonb_array_length(hechos)<>0 OR jsonb_array_length(nuevas) NOT BETWEEN 1 AND 1000)) THEN
  RAISE EXCEPTION 'fase de importación inválida' USING ERRCODE='22023';
 END IF;
 IF mf->>'tipo' NOT IN ('rpt','plantilla') OR mf->>'version_ref' !~ '^[0-9a-f-]{36}$'
    OR (mf->>'version_revision')::integer<1 OR mf->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR mf->>'fuente_version' IS NULL OR length(mf->>'fuente_version') NOT BETWEEN 1 AND 160
    OR jsonb_typeof(mf->'catalogo_unidades') IS DISTINCT FROM 'object'
    OR jsonb_typeof(mf->'catalogo_clasificaciones') IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'manifiesto de importación inválido' USING ERRCODE='22023';
 END IF;
 v_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_ref:=CASE WHEN v_fase='preparar' THEN v_org ELSE v_lote END;
 v_contexto:='{"ambitos":{"organismo_ref":'||to_jsonb(v_org)::text||'},"atributos":{"fase":'||to_jsonb(v_fase)::text||
  ',"fuente_ref":'||to_jsonb(v_fuente)::text||',"lote_ref":'||to_jsonb(v_ref)::text||
  ',"material_sha256":"'||v_sha||'"}}';
 v_contexto_sha:=encode(sha256(convert_to(v_contexto,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM v_ref OR c->>'efecto_ref' IS DISTINCT FROM v_ref
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_sha
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_contexto_sha THEN
  RAISE EXCEPTION 'contexto de importación divergente' USING ERRCODE='42501';
 END IF;
 -- La función de AD3-52 revalida rol/perfil/capacidad y registra auditoría.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_ref
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_sha THEN
  RAISE EXCEPTION 'consumo de importación divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:personal:importacion:'||v_clave::text,0));
 SELECT * INTO v_previo FROM vec_personal.importacion_organizacion_recibo WHERE clave_idempotencia=v_clave;
 IF FOUND THEN
  IF v_previo.material IS DISTINCT FROM p_material OR v_previo.material_huella_sha256 IS DISTINCT FROM v_sha
     OR v_previo.actor_ref IS DISTINCT FROM m->>'actor_ref' OR v_previo.fase IS DISTINCT FROM v_fase THEN
   RAISE EXCEPTION 'clave idempotente de importación divergente' USING ERRCODE='P0112';
  END IF;
  RETURN v_previo.recibo_json||jsonb_build_object('replay',true);
 END IF;
 IF v_fase='preparar' THEN
  -- Lote y hechos quedan sellados; no se insertan en las tablas publicadas.
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(hechos) h WHERE jsonb_typeof(h)<>'object'
      OR h->>'organismo_ref' IS DISTINCT FROM v_org
      OR h->>'hecho_ref' !~ '^[0-9a-f-]{36}$'
      OR h->>'fila_fuente_ref' IS NULL
      OR (h->>'revision')::integer<1)
     OR (SELECT count(*) FROM jsonb_array_elements(hechos))<>
        (SELECT count(*) FROM (SELECT DISTINCT h->>'clase',h->>'hecho_ref' FROM jsonb_array_elements(hechos) h) u) THEN
   RAISE EXCEPTION 'hechos preparatorios inválidos' USING ERRCODE='22023';
  END IF;
  v_lote:='lote:'||gen_random_uuid()::text;
  v_hechos_sha:=encode(sha256(convert_to(hechos::text,'UTF8')),'hex');
  v_revision_nueva:=1; v_estado:='preparacion_no_autoritativa';
  v_decisiones:='[]'::jsonb;
 ELSE
  PERFORM pg_advisory_xact_lock(hashtextextended('vec:personal:importacion:lote:'||v_lote,0));
  SELECT * INTO prev FROM vec_personal.importacion_organizacion_revision
   WHERE lote_ref=v_lote ORDER BY revision DESC LIMIT 1 FOR UPDATE;
  IF NOT FOUND OR prev.revision<>v_revision_esperada OR prev.organismo_ref IS DISTINCT FROM v_org
     OR prev.fuente_ref IS DISTINCT FROM v_fuente
     OR prev.fuente_version IS DISTINCT FROM mf->>'fuente_version'
     OR prev.fuente_huella_sha256 IS DISTINCT FROM mf->>'fuente_huella_sha256'
     OR prev.version_fuente_ref IS DISTINCT FROM (mf->>'version_ref')::uuid
     OR prev.version_fuente_revision IS DISTINCT FROM (mf->>'version_revision')::integer
     OR prev.catalogo_unidades IS DISTINCT FROM mf->'catalogo_unidades'
     OR prev.catalogo_clasificaciones IS DISTINCT FROM mf->'catalogo_clasificaciones'
     OR prev.manifiesto IS DISTINCT FROM mf THEN
   RAISE EXCEPTION 'lote de importación ha cambiado' USING ERRCODE='P0112';
  END IF;
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(nuevas) x WHERE jsonb_typeof(x)<>'object'
      OR x->>'resultado' NOT IN ('vinculada','pendiente','descartada')
      OR x->>'clase' NOT IN ('unidad','clasificacion','puesto_tipo','dotacion','plaza','puesto_individual','vinculo')
      OR length(coalesce(x->>'fila_fuente_ref','')) NOT BETWEEN 3 AND 256
      OR length(coalesce(x->>'evidencia_ref','')) NOT BETWEEN 3 AND 256
      OR length(coalesce(x->>'motivo','')) NOT BETWEEN 1 AND 2048
      OR (x->>'resultado'='vinculada' AND length(coalesce(x->>'destino_ref','')) NOT BETWEEN 3 AND 256)
      OR (x->>'resultado'<>'vinculada' AND coalesce(x->>'destino_ref','')<>''))
     OR (SELECT count(*) FROM jsonb_array_elements(nuevas))<>
        (SELECT count(*) FROM (SELECT DISTINCT x->>'clase',x->>'fila_fuente_ref' FROM jsonb_array_elements(nuevas) x) u) THEN
   RAISE EXCEPTION 'decisiones de conciliación inválidas' USING ERRCODE='22023';
  END IF;
  -- Cada decisión sobre una fila debe referir un hecho sellado. La clase
  -- "unidad" corresponde al hecho "nodo"; una decisión de clasificación
  -- adicional exige también una fila existente en el lote.
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(nuevas) x
     WHERE NOT EXISTS
      (SELECT 1 FROM jsonb_array_elements(prev.hechos) h
       WHERE h->>'fila_fuente_ref'=x->>'fila_fuente_ref'
         AND (h->>'clase'=CASE WHEN x->>'clase'='unidad' THEN 'nodo' ELSE x->>'clase' END
              OR x->>'clase'='clasificacion'))) THEN
   RAISE EXCEPTION 'conciliación ajena al lote' USING ERRCODE='P0112';
  END IF;
  -- La revisión nueva conserva las decisiones previas no sustituidas y añade
  -- las nuevas, con motivo y evidencia explícitos en ambas revisiones.
  SELECT coalesce(jsonb_agg(x ORDER BY x->>'clase',x->>'fila_fuente_ref'),'[]'::jsonb)
   INTO v_decisiones FROM (
    SELECT e AS x FROM jsonb_array_elements(prev.decisiones) e
     WHERE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(nuevas) n
      WHERE n->>'clase'=e->>'clase' AND n->>'fila_fuente_ref'=e->>'fila_fuente_ref')
    UNION ALL SELECT e FROM jsonb_array_elements(nuevas) e
   ) merged;
  v_hechos_sha:=prev.hechos_huella_sha256;
  hechos:=prev.hechos;
  v_revision_nueva:=v_revision_esperada+1;
  -- "conciliada" sólo significa que cada hecho tiene decisión vinculada;
  -- nunca acredita publicación, ocupación ni disponibilidad.
  IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(hechos) h WHERE NOT EXISTS
      (SELECT 1 FROM jsonb_array_elements(v_decisiones) x
       WHERE x->>'fila_fuente_ref'=h->>'fila_fuente_ref'
         AND x->>'clase'=CASE WHEN h->>'clase'='nodo' THEN 'unidad' ELSE h->>'clase' END
         AND x->>'resultado'='vinculada'))
     AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(v_decisiones) x WHERE x->>'resultado'<>'vinculada') THEN
   v_estado:='conciliada';
  ELSE v_estado:='conciliacion_pendiente'; END IF;
 END IF;
 v_ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR v_ahora >= (d->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'importación caducada' USING ERRCODE='42501';
 END IF;
 v_recibo_ref:='recibo:'||gen_random_uuid()::text;
 v_recibo:=jsonb_build_object('recibo_ref',v_recibo_ref,'lote_ref',v_lote,'fase',v_fase,
  'estado',v_estado,'revision_anterior',v_revision_esperada,'revision_nueva',v_revision_nueva,
  'clave_idempotencia',v_clave::text,'material_huella_sha256',v_sha,
  'fuente_huella_sha256',mf->>'fuente_huella_sha256','actor_ref',m->>'actor_ref',
  'decision_ref',v_consumo.decision_ref,'auditoria_ref',v_consumo.auditoria_ref,
  'registrado_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'replay',false);
 INSERT INTO vec_personal.importacion_organizacion_revision
  (lote_ref,revision,organismo_ref,fase,estado,manifiesto,hechos,hechos_huella_sha256,
   decisiones,fuente_ref,fuente_version,fuente_huella_sha256,version_fuente_ref,
   version_fuente_revision,catalogo_unidades,catalogo_clasificaciones,
   actor_ref,perfil_ref,recibo_ref,registrada_en)
 VALUES(v_lote,v_revision_nueva,v_org,v_fase,v_estado,mf,hechos,v_hechos_sha,
  v_decisiones,v_fuente,mf->>'fuente_version',mf->>'fuente_huella_sha256',
  (mf->>'version_ref')::uuid,(mf->>'version_revision')::integer,
  mf->'catalogo_unidades',mf->'catalogo_clasificaciones',m->>'actor_ref',m->>'perfil_ref',
  v_recibo_ref,v_ahora);
 INSERT INTO vec_personal.importacion_organizacion_recibo
  (clave_idempotencia,lote_ref,revision,fase,material,material_huella_sha256,actor_ref,
   decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,recibo_json,registrada_en)
 VALUES(v_clave,v_lote,v_revision_nueva,v_fase,p_material,v_sha,m->>'actor_ref',
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,
  v_recibo_ref,v_recibo,v_ahora);
 RETURN v_recibo;
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMENT ON FUNCTION vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 IS 'B3: preparación/conciliación aditivas con AD3-52; publicación denegada hasta autoridad durable de fuente y acto.';
COMMIT;
