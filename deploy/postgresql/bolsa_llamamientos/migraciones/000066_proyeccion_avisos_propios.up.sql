\set ON_ERROR_STOP on
-- B66, posterior a B62 y B65. Sólo cambia dos fragmentos de proyección Own.
-- Migración hacia delante: no ejecutar DOWN ni reaplicar sobre historia instalada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000066',0));
DO $proyeccion$
DECLARE cambio record; antes pg_catalog.pg_proc%ROWTYPE; despues pg_catalog.pg_proc%ROWTYPE;
 definicion text; cuerpo text; acl_actual jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.aviso_externo_outbox') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.aviso_externo_resultado') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B66: dependencias B62/B65 incompatibles' USING ERRCODE='55000'; END IF;
 FOR cambio IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','ded995d98d206749aae198b3e15d13c38624c6732b426ac24f8fdb07d717d12b',
   $antes$   SELECT l.llamamiento_ref,l.emitido_en,contacto.resultado
     FROM vec_bolsa_llamamientos.llamamiento_emitido l
     JOIN vec_bolsa_llamamientos.contacto_participacion contacto
       ON contacto.llamamiento_ref=l.llamamiento_ref
      AND contacto.bolsa_ref=l.bolsa_ref
      AND contacto.participacion_ref=participacion.participacion_ref
      AND contacto.canal='correo'
      AND contacto.resultado IN('enviado','no_enviado')
      AND contacto.instante<=p_consultada_en
    WHERE l.bolsa_ref=participacion.bolsa_ref
      AND l.participaciones ? participacion.participacion_ref
      AND l.emitido_en<=p_consultada_en
    ORDER BY l.emitido_en DESC,l.llamamiento_ref DESC,contacto.instante DESC,contacto.contacto_ref DESC
    LIMIT 1$antes$,
   $nuevo$   SELECT aviso.llamamiento_ref,aviso.emitido_en,aviso.resultado
   FROM (
     -- B62: una fila por emisión y participación, incluso sin resultado.
     SELECT l.llamamiento_ref,l.emitido_en,
       CASE despacho.estado WHEN 'aceptado' THEN 'enviado'
         WHEN 'no_aceptado' THEN 'no_enviado' WHEN 'sin_destino' THEN 'no_enviado'
         ELSE 'aviso_pendiente' END AS resultado,
       l.emitido_en AS orden_instante,o.evento_ref AS orden_ref
     FROM vec_bolsa_llamamientos.llamamiento_emitido l
     JOIN vec_bolsa_llamamientos.aviso_externo_outbox o
       ON o.llamamiento_ref=l.llamamiento_ref
      AND o.participacion_ref=participacion.participacion_ref
     LEFT JOIN LATERAL (
       SELECT r.estado FROM vec_bolsa_llamamientos.aviso_externo_resultado r
       WHERE r.productor_ref=o.productor_ref AND r.evento_ref=o.evento_ref
         AND r.registrada_en<=p_consultada_en
       ORDER BY r.version DESC LIMIT 1
     ) despacho ON true
     WHERE l.bolsa_ref=participacion.bolsa_ref
       AND l.participaciones ? participacion.participacion_ref
       AND l.emitido_en<=p_consultada_en AND o.registrada_en<=p_consultada_en
     UNION ALL
     -- Legado sin pareja B62. No reutilizar el contacto terminal del outbox:
     -- su instante es la emisión, aunque su resultado se conociera después.
     SELECT l.llamamiento_ref,l.emitido_en,contacto.resultado,
       contacto.instante,contacto.contacto_ref
     FROM vec_bolsa_llamamientos.llamamiento_emitido l
     JOIN vec_bolsa_llamamientos.contacto_participacion contacto
       ON contacto.llamamiento_ref=l.llamamiento_ref
      AND contacto.bolsa_ref=l.bolsa_ref
      AND contacto.participacion_ref=participacion.participacion_ref
      AND contacto.canal='correo'
      AND contacto.resultado IN('enviado','no_enviado')
      AND contacto.instante<=p_consultada_en
     WHERE l.bolsa_ref=participacion.bolsa_ref
       AND l.participaciones ? participacion.participacion_ref
       AND l.emitido_en<=p_consultada_en
       AND NOT EXISTS (
         SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox o
         WHERE o.llamamiento_ref=l.llamamiento_ref
           AND o.participacion_ref=participacion.participacion_ref
       )
   ) aviso
   ORDER BY aviso.emitido_en DESC,aviso.llamamiento_ref DESC,
     aviso.orden_instante DESC,aviso.orden_ref DESC
   LIMIT 1$nuevo$),
  ('vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','86e484771e7ebe1e9beb8643ac96dd73e0b62c6ee4782dc65072a4f1cbf999c9',
   $antes$  SELECT l.emitido_en, 'llamamiento:'||l.llamamiento_ref||':'||ct.contacto_ref,
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
   WHERE l.emitido_en<=p_consultada_en AND ct.instante<=p_consultada_en$antes$,
   $nuevo$  SELECT l.emitido_en, 'llamamiento:'||l.llamamiento_ref||':'||o.evento_ref,
   jsonb_build_object('clase','llamamiento','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(l.emitido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'canal','correo','resultado',CASE despacho.estado
      WHEN 'aceptado' THEN 'enviado' WHEN 'no_aceptado' THEN 'no_enviado'
      WHEN 'sin_destino' THEN 'no_enviado' ELSE 'aviso_pendiente' END)
   FROM propias p
   JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.bolsa_ref=p.bolsa_ref
    AND l.participaciones ? p.participacion_ref
   JOIN vec_bolsa_llamamientos.aviso_externo_outbox o
    ON o.llamamiento_ref=l.llamamiento_ref AND o.participacion_ref=p.participacion_ref
   LEFT JOIN LATERAL (
    SELECT r.estado FROM vec_bolsa_llamamientos.aviso_externo_resultado r
    WHERE r.productor_ref=o.productor_ref AND r.evento_ref=o.evento_ref
     AND r.registrada_en<=p_consultada_en
    ORDER BY r.version DESC LIMIT 1
   ) despacho ON true
   WHERE l.emitido_en<=p_consultada_en AND o.registrada_en<=p_consultada_en
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
    AND NOT EXISTS (
     SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox o
     WHERE o.llamamiento_ref=l.llamamiento_ref AND o.participacion_ref=p.participacion_ref
    )$nuevo$)
 ) AS cambios(firma,huella,anterior,nuevo) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=to_regprocedure(cambio.firma);
  IF NOT FOUND THEN RAISE EXCEPTION 'B66: función ausente %',cambio.firma USING ERRCODE='55000'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object(
    'grantee',r.rolname,'grantor',g.rolname,'grantable',a.is_grantable,'privilege',a.privilege_type)
    ORDER BY r.rolname,g.rolname,a.privilege_type,a.is_grantable),'[]'::jsonb)
  INTO acl_actual FROM aclexplode(coalesce(antes.proacl,acldefault('f',antes.proowner))) a
   LEFT JOIN pg_roles r ON r.oid=a.grantee LEFT JOIN pg_roles g ON g.oid=a.grantor;
  IF antes.proowner IS DISTINCT FROM to_regrole('vec_bolsa_llamamientos_propietario')
     OR antes.prosecdef IS NOT TRUE
     OR antes.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s','statement_timeout=15s']
     OR encode(sha256(convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM cambio.huella
     OR acl_actual IS DISTINCT FROM '[{"grantee":"vec_bolsa_llamamientos_ejecutor","grantor":"vec_bolsa_llamamientos_propietario","grantable":false,"privilege":"EXECUTE"},{"grantee":"vec_bolsa_llamamientos_portal_externo","grantor":"vec_bolsa_llamamientos_propietario","grantable":false,"privilege":"EXECUTE"},{"grantee":"vec_bolsa_llamamientos_propietario","grantor":"vec_bolsa_llamamientos_propietario","grantable":false,"privilege":"EXECUTE"}]'::jsonb
     OR (length(antes.prosrc)-length(replace(antes.prosrc,cambio.anterior,'')))<>length(cambio.anterior)
  THEN RAISE EXCEPTION 'B66: preimagen incompatible %',cambio.firma USING ERRCODE='55000'; END IF;
  definicion:=pg_get_functiondef(antes.oid);
  IF (length(definicion)-length(replace(definicion,cambio.anterior,'')))<>length(cambio.anterior)
  THEN RAISE EXCEPTION 'B66: marca ambigua %',cambio.firma USING ERRCODE='55000'; END IF;
  cuerpo:=replace(antes.prosrc,cambio.anterior,cambio.nuevo);
  EXECUTE replace(definicion,cambio.anterior,cambio.nuevo);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  -- Guardas, consumo V3, auditoría y resto de prosrc quedan idénticos;
  -- firma, OID, propietario, ACL y toda la configuración también.
  IF despues.prosrc IS DISTINCT FROM cuerpo
     OR (to_jsonb(despues)-'prosrc') IS DISTINCT FROM (to_jsonb(antes)-'prosrc')
  THEN RAISE EXCEPTION 'B66: postimagen incompatible %',cambio.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $proyeccion$;
COMMIT;
