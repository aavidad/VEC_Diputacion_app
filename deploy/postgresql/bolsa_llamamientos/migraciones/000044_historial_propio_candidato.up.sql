\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000044',0));

-- Histórico informativo propio. La autorización y su auditoría de lectura se
-- consumen en la misma transacción que produce esta página. Las fuentes son
-- exclusivamente tablas de Bolsa; los eventos CT son copias recibidas aquí.
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR EXISTS (SELECT 1 FROM pg_class t
      WHERE t.oid=ANY (ARRAY[
       to_regclass('vec_bolsa_llamamientos.vinculo_candidato'),
       to_regclass('vec_bolsa_llamamientos.contrato_participacion'),
       to_regclass('vec_bolsa_llamamientos.llamamiento_emitido'),
       to_regclass('vec_bolsa_llamamientos.contacto_participacion'),
       to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento')])
       AND (NOT t.relrowsecurity OR NOT t.relforcerowsecurity))
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Bolsa 000044: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(
 p_candidato_ref text,p_consultada_en timestamptz,p_pagina integer,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_consumo record;
 v_items jsonb; v_hay_mas boolean;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_consultada_en IS NULL OR NOT isfinite(p_consultada_en)
    OR p_pagina IS NULL OR p_pagina NOT BETWEEN 1 AND 10000
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'consulta de historial inválida' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de historial inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.historial_propio.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.mi-bolsa.historial.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_historial_propio'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
   SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
    WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
  FROM vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;

 WITH propias AS MATERIALIZED (
  SELECT participacion_ref,bolsa_ref,categoria_ref
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref)
 ), entradas AS (
  SELECT co.ocurrido_en, 'contrato:'||co.evento_ref AS orden_interno,
   jsonb_build_object('clase','contrato_bolsa','procedencia','evento_ct_recibido','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(co.ocurrido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'tipo',co.tipo,'inicio',co.inicio,'fin_previsto',co.fin_previsto,
    'modalidad_clave',co.modalidad_clave) AS item
   FROM propias p JOIN vec_bolsa_llamamientos.contrato_participacion co
    ON co.participacion_ref=p.participacion_ref AND co.bolsa_ref=p.bolsa_ref
   WHERE co.ocurrido_en<=p_consultada_en
  UNION ALL
  SELECT l.emitido_en, 'llamamiento:'||l.llamamiento_ref||':'||ct.contacto_ref,
   jsonb_build_object('clase','llamamiento','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(l.emitido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'canal','correo','resultado',ct.resultado)
   FROM propias p
   JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.bolsa_ref=p.bolsa_ref
   JOIN LATERAL jsonb_array_elements_text(l.participaciones) WITH ORDINALITY lp(ref,ordinal) ON lp.ref=p.participacion_ref
   JOIN vec_bolsa_llamamientos.contacto_participacion ct
    ON ct.llamamiento_ref=l.llamamiento_ref AND ct.bolsa_ref=l.bolsa_ref
    AND ct.participacion_ref=p.participacion_ref AND ct.canal='correo'
    AND ct.resultado IN ('enviado','no_enviado')
    AND ct.instante=l.emitido_en
    AND ct.clave_idempotencia=l.clave_idempotencia||':correo:'||lp.ordinal
    AND ct.recibo_ref='recibo:contacto:'||encode(sha256(convert_to(
      l.bolsa_ref||chr(31)||l.clave_idempotencia||chr(31)||p.participacion_ref,'UTF8')),'hex')
   WHERE l.emitido_en<=p_consultada_en AND ct.instante<=p_consultada_en
  UNION ALL
  SELECT r.respondida_en, 'renuncia:'||r.respuesta_ref,
   jsonb_build_object('clase','renuncia','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'respuesta',r.respuesta,'modo',r.modo,
    'estado',CASE r.modo WHEN 'firme' THEN 'respuesta_registrada' ELSE 'propuesta_pendiente_rrhh' END)
   FROM propias p JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento r
    ON r.participacion_ref=p.participacion_ref AND r.bolsa_ref=p.bolsa_ref
    AND r.candidato_ref=p_candidato_ref AND r.respuesta IN ('renuncia','renuncia_justificada')
   WHERE r.respondida_en<=p_consultada_en
 ), pagina AS (
  SELECT e.ocurrido_en,e.orden_interno,e.item FROM entradas e
   ORDER BY e.ocurrido_en DESC,e.orden_interno DESC
   LIMIT 21 OFFSET (p_pagina-1)*20
 ), numerada AS (
  SELECT row_number() OVER (ORDER BY ocurrido_en DESC,orden_interno DESC) AS n,
   ocurrido_en,orden_interno,item FROM pagina
 )
 SELECT count(*)>20,
   coalesce(jsonb_agg(item ORDER BY ocurrido_en DESC,orden_interno DESC) FILTER (WHERE n<=20),'[]'::jsonb)
 INTO v_hay_mas,v_items FROM numerada;
 RETURN jsonb_build_object('consultada_en',to_char(p_consultada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'pagina',p_pagina,'tamano',20,'hay_mas',v_hay_mas,'items',v_items);
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s']
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND a.grantee<>p.proowner AND (a.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'Bolsa 000044: ACL de lectura abierta' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
