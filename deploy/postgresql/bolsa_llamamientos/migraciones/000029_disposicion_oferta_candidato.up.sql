\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000029', 0));

-- B-OF (continuación de 000028): la persona manifiesta su disposición a una
-- oferta publicada de su bolsa desde «Mi bolsa». Consume la acción propia
-- AD3-84 'bolsa.participaciones_propias.manifestar_disposicion' sobre el
-- recurso 'oferta:<huella>'; como el recurso no es la propia bolsa, aquí se
-- coteja al candidato con el único vínculo candidato activo del contexto
-- atestado y se resuelve su participación en la bolsa de la oferta.
--
-- La tabla disposicion_oferta (000028) no se altera: la decisión consumida y
-- el candidato quedan en una tabla nueva, de solo adición, ligada a cada
-- disposición. Solo se admite con la oferta abierta (publicada, sin resolver
-- y antes del vencimiento) y una sola vez por participación; la misma clave
-- repite el recibo, siempre con una decisión nueva consumida antes.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.disposicion_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.resolucion_oferta') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'dependencias de la disposición a ofertas ausentes (000028 y AD3-84)' USING ERRCODE='55000';
 END IF;
 IF to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'consulta Mi bolsa ausente (000010 y 000023)' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.disposicion_oferta_candidato') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.secreto_marca_consumo') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion 000029 ya aplicada' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato(
 oferta_ref text NOT NULL,
 participacion_ref text NOT NULL,
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 decision_ref text NOT NULL UNIQUE CHECK (octet_length(decision_ref) BETWEEN 1 AND 256),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (oferta_ref, participacion_ref),
 FOREIGN KEY (oferta_ref, participacion_ref) REFERENCES vec_bolsa_llamamientos.disposicion_oferta(oferta_ref, participacion_ref)
);
CREATE INDEX disposicion_oferta_candidato_candidato ON vec_bolsa_llamamientos.disposicion_oferta_candidato(candidato_ref, oferta_ref);
ALTER TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato FORCE ROW LEVEL SECURITY;
CREATE POLICY disposicion_oferta_candidato_solo_propietario ON vec_bolsa_llamamientos.disposicion_oferta_candidato
 TO vec_bolsa_llamamientos_propietario
 USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.disposicion_oferta_candidato FROM PUBLIC;
CREATE TRIGGER disposicion_oferta_candidato_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.disposicion_oferta_candidato
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Participación vigente del candidato en cada bolsa: si una misma bolsa le
-- dio varias participaciones (una constitución nueva de la misma bolsa), la
-- vigente es la de confirmación más reciente. Es el ÚNICO criterio: lo usan
-- tanto las listas del candidato como sus acciones (ofertas, portal y
-- contacto), para que nunca se muestre una participación y se actúe sobre
-- otra.
CREATE FUNCTION vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref text)
RETURNS TABLE(bolsa_ref text, participacion_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT DISTINCT ON (x.bolsa_ref) x.bolsa_ref, x.participacion_ref
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) x
  ORDER BY x.bolsa_ref, x.confirmada_en DESC, x.participacion_ref DESC
$f$;

-- Participación vigente del candidato en la bolsa de la oferta. Sin vínculo:
-- 42501 (el recurso no le pertenece).
CREATE FUNCTION vec_bolsa_llamamientos.participacion_oferta_candidato_v1(p_candidato_ref text, p_bolsa_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v text;
BEGIN
 SELECT p.participacion_ref INTO v FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
  WHERE p.bolsa_ref = p_bolsa_ref;
 IF v IS NULL THEN RAISE EXCEPTION 'la oferta no es de una bolsa del candidato' USING ERRCODE='42501'; END IF;
 RETURN v;
END $f$;

-- Marca de consumo en la transacción. Las lecturas propias del candidato
-- (ofertas, portal y contacto) solo responden dentro de la transacción que
-- ya consumió una decisión propia de ESE candidato. La base lo comprueba así:
-- tras consumir, una función del propietario deja en una variable local de
-- la transacción (set_config(..., true)) el ámbito, el candidato, un dato y
-- una firma SHA-256 con un secreto que solo lee el propietario, ligada
-- al identificador de la transacción. Cualquier rol puede escribir la
-- variable, pero no firmarla: una marca forjada, de otra transacción o de
-- otro candidato se rechaza con 42501. La marca desaparece al terminar la
-- transacción.
CREATE TABLE vec_bolsa_llamamientos.secreto_marca_consumo(
 unico boolean PRIMARY KEY DEFAULT true CHECK (unico),
 secreto bytea NOT NULL CHECK (octet_length(secreto) = 32),
 creado_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.secreto_marca_consumo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.secreto_marca_consumo FORCE ROW LEVEL SECURITY;
CREATE POLICY secreto_marca_consumo_solo_propietario ON vec_bolsa_llamamientos.secreto_marca_consumo
 TO vec_bolsa_llamamientos_propietario
 USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.secreto_marca_consumo FROM PUBLIC;
CREATE TRIGGER secreto_marca_consumo_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.secreto_marca_consumo
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
-- gen_random_uuid usa el generador fuerte del servidor: 3 x 122 bits.
INSERT INTO vec_bolsa_llamamientos.secreto_marca_consumo(unico, secreto, creado_en)
VALUES (true, sha256(convert_to(gen_random_uuid()::text || gen_random_uuid()::text || gen_random_uuid()::text, 'UTF8')), clock_timestamp());

CREATE FUNCTION vec_bolsa_llamamientos.firma_marca_consumo_v1(p_xid xid8, p_ambito text, p_candidato_ref text, p_dato text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 -- Firma anidada con secreto: sha256(k || sha256(k || mensaje)). La
 -- envoltura exterior, de longitud fija, impide extender una firma vista.
 SELECT encode(sha256(s.secreto || sha256(s.secreto || convert_to(
          p_xid::text || chr(31) || p_ambito || chr(31) || p_candidato_ref || chr(31) || p_dato, 'UTF8'))), 'hex')
   FROM vec_bolsa_llamamientos.secreto_marca_consumo s
$f$;

-- Deja la marca tras un consumo propio. Sin EXECUTE para el ejecutor: solo
-- la llaman funciones del propietario inmediatamente después de consumir.
CREATE FUNCTION vec_bolsa_llamamientos.anotar_consumo_candidato_v1(p_ambito text, p_candidato_ref text, p_dato text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_xid xid8 := pg_current_xact_id();
BEGIN
 IF p_ambito IS NULL OR p_ambito !~ '^[a-z_]{1,32}$' OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_dato IS NULL OR octet_length(p_dato) NOT BETWEEN 1 AND 512 OR strpos(p_dato, chr(31)) <> 0 THEN
  RAISE EXCEPTION 'marca de consumo inválida' USING ERRCODE='22023';
 END IF;
 PERFORM set_config('vec_bolsa_llamamientos.marca_consumo',
   concat_ws(chr(31), p_ambito, p_candidato_ref, p_dato, vec_bolsa_llamamientos.firma_marca_consumo_v1(v_xid, p_ambito, p_candidato_ref, p_dato)), true);
END $f$;

-- Exige una marca válida de uno de los ámbitos para el candidato, en esta
-- misma transacción; devuelve su dato. Sin marca válida: 42501.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_consumo_candidato_v1(p_ambitos text[], p_candidato_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_marca text := current_setting('vec_bolsa_llamamientos.marca_consumo', true);
 v_xid xid8 := pg_current_xact_id_if_assigned(); v_partes text[];
BEGIN
 IF v_marca IS NULL OR v_marca = '' OR v_xid IS NULL OR p_candidato_ref IS NULL THEN
  RAISE EXCEPTION 'lectura del candidato sin consumo propio en la transacción' USING ERRCODE='42501';
 END IF;
 v_partes := string_to_array(v_marca, chr(31));
 IF cardinality(v_partes) <> 4 OR NOT (v_partes[1] = ANY (p_ambitos)) OR v_partes[2] IS DISTINCT FROM p_candidato_ref
    OR v_partes[4] IS DISTINCT FROM vec_bolsa_llamamientos.firma_marca_consumo_v1(v_xid, v_partes[1], v_partes[2], v_partes[3]) THEN
  RAISE EXCEPTION 'lectura del candidato sin consumo propio en la transacción' USING ERRCODE='42501';
 END IF;
 RETURN v_partes[3];
END $f$;

-- Consulta Mi bolsa (000010/000023) que además deja la marca de consumo, para
-- que en la misma transacción se lean las ofertas, el portal y el contacto
-- del mismo candidato. consultar_mi_bolsa_v1 consume la decisión propia o
-- falla; solo después se anota la marca.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(
 p_candidato_ref text, p_consultada_en timestamptz, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
 p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET lock_timeout = '2s' AS $f$
DECLARE v jsonb;
BEGIN
 v := vec_bolsa_llamamientos.consultar_mi_bolsa_v1(p_candidato_ref, p_consultada_en, p_capacidad, p_decision, p_motivo, p_contexto,
   p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_v1('consulta', p_candidato_ref, 'mi-bolsa');
 RETURN v;
END $f$;
-- Núcleo sin material: solo lo alcanza la función pública tras consumir la
-- decisión (y las pruebas como propietario).
CREATE FUNCTION vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(
 p_oferta_ref text, p_recibo_ref text, p_candidato_ref text, p_participacion_ref text, p_clave text,
 p_manifestada_en timestamptz, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, recibo_ref text, oferta_ref text, manifestada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_manifestada_en IS NULL OR p_decision_ref IS NULL
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:disposicion:[0-9a-f]{64}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'disposición a oferta inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:' || p_oferta_ref, 0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x WHERE x.oferta_ref = p_oferta_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'oferta inexistente' USING ERRCODE='23503'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta d WHERE d.oferta_ref = p_oferta_ref AND d.participacion_ref = p_participacion_ref) THEN
  RAISE EXCEPTION 'disposición ya manifestada' USING ERRCODE='VBO05';
 END IF;
 -- Abierta: publicada, sin resolución y antes del vencimiento, tanto en el
 -- instante declarado como en el reloj de la base al registrar.
 IF p_manifestada_en < o.publicada_en OR p_manifestada_en >= o.vence_antes_de OR clock_timestamp() >= o.vence_antes_de
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_oferta r WHERE r.oferta_ref = p_oferta_ref) THEN
  RAISE EXCEPTION 'la oferta no está abierta' USING ERRCODE='VBO06';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta(oferta_ref, participacion_ref, manifestada_en, clave_idempotencia, recibo_ref)
 VALUES (p_oferta_ref, p_participacion_ref, p_manifestada_en, p_clave, p_recibo_ref);
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta_candidato(oferta_ref, participacion_ref, candidato_ref, decision_ref, registrada_en)
 VALUES (p_oferta_ref, p_participacion_ref, p_candidato_ref, p_decision_ref, p_manifestada_en);
 RETURN QUERY SELECT false, p_recibo_ref, p_oferta_ref, p_manifestada_en;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(
 p_oferta_ref text, p_recibo_ref text, p_candidato_ref text, p_clave text, p_manifestada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, oferta_ref text, manifestada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' SET lock_timeout = '2s' AS $f$
DECLARE c jsonb; d jsonb; x jsonb; n integer; o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
 v_participacion text; v_previa vec_bolsa_llamamientos.disposicion_oferta%ROWTYPE; v_consumo record;
 v_accion constant text := 'bolsa.participaciones_propias.manifestar_disposicion';
 v_ahora timestamptz := clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_oferta_ref IS NULL OR p_oferta_ref !~ '^oferta:[0-9a-f]{64}$'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:disposicion:[0-9a-f]{64}$'
    OR p_manifestada_en IS NULL OR p_manifestada_en > v_ahora + interval '1 minute' OR p_manifestada_en < v_ahora - interval '5 minutes' THEN
  RAISE EXCEPTION 'disposición a oferta inválida' USING ERRCODE='22023';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de disposición inválido' USING ERRCODE='22023'; END;
 -- Acción y recurso exactos: la oferta es el recurso.
 IF c->>'efecto_ref' IS DISTINCT FROM p_oferta_ref OR d->>'recurso_ref' IS DISTINCT FROM p_oferta_ref
    OR c->>'operacion' IS DISTINCT FROM v_accion OR d->>'accion' IS DISTINCT FROM v_accion
    OR d->>'tipo_recurso' IS DISTINCT FROM 'oferta_bolsa'
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 -- El candidato es el único vínculo candidato activo del contexto atestado.
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
                           WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x2 WHERE x2.oferta_ref = p_oferta_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'oferta inexistente' USING ERRCODE='23503'; END IF;
 v_participacion := vec_bolsa_llamamientos.participacion_oferta_candidato_v1(p_candidato_ref, o.bolsa_ref);
 -- Como B2 y el portal (000030), la decisión viva se consume antes de
 -- resolver el replay: un reintento no devuelve el recibo sin una
 -- autorización nueva y verificada.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM p_oferta_ref THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:' || p_oferta_ref, 0));
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.disposicion_oferta dp
  WHERE dp.oferta_ref = p_oferta_ref AND dp.participacion_ref = v_participacion;
 IF FOUND THEN
  IF v_previa.clave_idempotencia <> p_clave THEN
   RAISE EXCEPTION 'disposición ya manifestada' USING ERRCODE='VBO05';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_ref, v_previa.oferta_ref, v_previa.manifestada_en;
  RETURN;
 END IF;
 IF clock_timestamp() >= o.vence_antes_de THEN
  RAISE EXCEPTION 'la oferta no está abierta' USING ERRCODE='VBO06';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(
  p_oferta_ref, p_recibo_ref, p_candidato_ref, v_participacion, p_clave, p_manifestada_en, v_consumo.decision_ref);
END $f$;

-- Ofertas de las bolsas del candidato que le interesan en un instante: las
-- abiertas y aquellas en que ya manifestó disposición. No expone
-- participaciones, disposiciones ajenas ni a quién se adjudicó. Solo
-- responde dentro de la transacción que ya consumió la consulta propia (Mi
-- bolsa) del mismo candidato: sin esa marca, 42501. La participación de cada
-- bolsa es la vigente, el mismo criterio que usa manifestar.
CREATE FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_v1(p_candidato_ref text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta'], p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'oferta_ref', o.oferta_ref, 'bolsa', o.bolsa_ref, 'datos', o.datos,
   'publicada_en', to_char(o.publicada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vence_antes_de', to_char(o.vence_antes_de,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado', CASE WHEN r.oferta_ref IS NOT NULL AND r.participacion_ref = p.participacion_ref THEN 'adjudicada_propia'
                  WHEN r.oferta_ref IS NOT NULL THEN 'resuelta'
                  WHEN p_corte < o.vence_antes_de THEN 'abierta'
                  ELSE 'pendiente_resolucion' END,
   'disposicion', CASE WHEN d.oferta_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo', d.recibo_ref,
      'manifestada_en', to_char(d.manifestada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY o.vence_antes_de, o.oferta_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN vec_bolsa_llamamientos.oferta_publicada o ON o.bolsa_ref = p.bolsa_ref AND o.publicada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.disposicion_oferta d
   ON d.oferta_ref = o.oferta_ref AND d.participacion_ref = p.participacion_ref AND d.manifestada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.resolucion_oferta r ON r.oferta_ref = o.oferta_ref AND r.resuelta_en <= p_corte
 WHERE p_corte IS NOT NULL
   AND ((r.oferta_ref IS NULL AND p_corte < o.vence_antes_de) OR d.oferta_ref IS NOT NULL));
END $f$;

DO $acl$
DECLARE f regprocedure; publicas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)'::regprocedure,
  'vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
 internas regprocedure[] := ARRAY[
  'vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text)'::regprocedure,
  'vec_bolsa_llamamientos.firma_marca_consumo_v1(xid8,text,text,text)'::regprocedure,
  'vec_bolsa_llamamientos.anotar_consumo_candidato_v1(text,text,text)'::regprocedure,
  'vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text)'::regprocedure,
  'vec_bolsa_llamamientos.participacion_oferta_candidato_v1(text,text)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(text,text,text,text,text,timestamptz,text)'::regprocedure];
BEGIN
 FOREACH f IN ARRAY publicas || internas LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor', f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas || internas LOOP
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
              WHERE p.oid = f AND a.grantee <> p.proowner
                AND (a.grantee <> 'vec_bolsa_llamamientos_ejecutor'::regrole OR a.is_grantable OR NOT f = ANY (publicas)))
     OR (SELECT proowner FROM pg_proc WHERE oid = f) <> 'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT (SELECT prosecdef FROM pg_proc WHERE oid = f) THEN
   RAISE EXCEPTION 'ACL de la disposición a ofertas abierta' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
