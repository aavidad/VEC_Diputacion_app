\set ON_ERROR_STOP on
-- B59: constancia de a qué correo se envió cada aviso de llamamiento.
-- Si la persona candidata tiene un correo activo verificado en «Mis correos»
-- (Usuarios), el aviso va a ese correo; si no, al del alta en la bolsa, como
-- hasta ahora. Aquí sólo queda la fuente usada, el motivo y, cuando la fuente
-- es «Mis correos», la referencia opaca del correo en Usuarios. Nunca la
-- dirección ni una huella de ella.
--
-- Bolsa no lee tablas de Usuarios: pide el correo por un puerto de Usuarios.
-- La referencia de candidato de cada participación se obtiene aquí con una
-- función propia, para que Usuarios resuelva a la persona con ContextoActor.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000059',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(text,text,text,bytea,jsonb)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contacto_fuente_correo') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'B59: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.contacto_fuente_correo(
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.contacto_participacion(recibo_ref),
 fuente text NOT NULL CHECK(fuente IN('mis_correos','alta_bolsa')),
 motivo text NOT NULL CHECK(motivo IN('correo_activo','sin_correo_activo','sin_persona_vinculada','mis_correos_no_disponible')),
 correo_ref text CHECK(correo_ref ~ '^correo:[0-9a-f]{32}$'),
 registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 CHECK((fuente='mis_correos' AND motivo='correo_activo' AND correo_ref IS NOT NULL)
    OR (fuente='alta_bolsa' AND motivo<>'correo_activo' AND correo_ref IS NULL))
);
ALTER TABLE vec_bolsa_llamamientos.contacto_fuente_correo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.contacto_fuente_correo FORCE ROW LEVEL SECURITY;
CREATE POLICY contacto_fuente_correo_solo_propietario ON vec_bolsa_llamamientos.contacto_fuente_correo TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.contacto_fuente_correo FROM PUBLIC;
CREATE TRIGGER contacto_fuente_correo_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.contacto_fuente_correo FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Referencia de candidato de una participación, o NULL si no está vinculada.
-- Sólo responde para una participación de un llamamiento de esa bolsa que ya
-- existe (reservado o con contactos registrados): Go sólo pregunta por los
-- avisos de una emisión real. Usuarios no puede comprobarlo por sí mismo; la
-- referencia queda fijada en la huella auditada de su lectura.
CREATE FUNCTION vec_bolsa_llamamientos.candidato_participacion_avisos_v1(p_bolsa text,p_llamamiento text,p_participacion text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT v.candidato_ref FROM vec_bolsa_llamamientos.vinculo_candidato v
  WHERE v.participacion_ref=p_participacion
    AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido l
                 WHERE l.llamamiento_ref=p_llamamiento AND l.bolsa_ref=p_bolsa AND l.participaciones ? p_participacion)
$f$;

-- Fuentes ya registradas para los avisos de una emisión, por recibo de
-- contacto. Sirve igual a la respuesta nueva que a la recuperación.
CREATE FUNCTION vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(p_bolsa text,p_clave text)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT coalesce(jsonb_object_agg(f.recibo_ref,jsonb_strip_nulls(jsonb_build_object('fuente',f.fuente,'motivo',f.motivo,'correo_ref',f.correo_ref))),'{}'::jsonb)
   FROM vec_bolsa_llamamientos.llamamiento_emitido l
   CROSS JOIN LATERAL jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
   JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
    AND c.canal='correo' AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
   JOIN vec_bolsa_llamamientos.contacto_fuente_correo f ON f.recibo_ref=c.recibo_ref
  WHERE l.bolsa_ref=p_bolsa AND l.clave_idempotencia=p_clave
$f$;

-- Igual que la v1 más la fuente de cada contacto, en la misma transacción.
-- Si los contactos ya estaban registrados (repetición), no escribe fuentes y
-- devuelve las que hubiera. Dos llamadas simultáneas con el mismo token no
-- duplican ni fallan: la v1 las serializa y la fuente se inserta una vez.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(p_bolsa text,p_clave text,p_actor text,p_token_finalizacion bytea,p_contactos jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE simples jsonb; existentes integer; resultado jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_contactos IS NULL OR jsonb_typeof(p_contactos)<>'array'
    OR jsonb_array_length(p_contactos) NOT BETWEEN 1 AND 100
    OR NOT coalesce((SELECT bool_and(
          jsonb_typeof(c.value)='object'
          AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c.value) k)=ARRAY['fuente_correo','participacion_ref','recibo_ref','resultado']
          AND jsonb_typeof(c.value->'fuente_correo')='object'
          AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c.value->'fuente_correo') k) IN (ARRAY['fuente','motivo'],ARRAY['correo_ref','fuente','motivo'])
          AND NOT EXISTS (SELECT 1 FROM jsonb_each(c.value->'fuente_correo') z WHERE jsonb_typeof(z.value)<>'string')
          AND ((c.value#>>'{fuente_correo,fuente}'='mis_correos' AND c.value#>>'{fuente_correo,motivo}'='correo_activo'
                AND c.value#>>'{fuente_correo,correo_ref}' ~ '^correo:[0-9a-f]{32}$')
            OR (c.value#>>'{fuente_correo,fuente}'='alta_bolsa' AND c.value->'fuente_correo'->'correo_ref' IS NULL
                AND c.value#>>'{fuente_correo,motivo}' IN ('sin_correo_activo','sin_persona_vinculada','mis_correos_no_disponible'))))
       FROM jsonb_array_elements(p_contactos) c),false)
 THEN RAISE EXCEPTION 'resultado B59 inválido' USING ERRCODE='22023'; END IF;
 SELECT jsonb_agg(c.value-'fuente_correo' ORDER BY c.ordinality) INTO simples
   FROM jsonb_array_elements(p_contactos) WITH ORDINALITY c(value,ordinality);
 SELECT count(*) INTO existentes FROM vec_bolsa_llamamientos.contacto_participacion p
  WHERE p.recibo_ref IN (SELECT c.value->>'recibo_ref' FROM jsonb_array_elements(p_contactos) c);
 resultado:=vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(p_bolsa,p_clave,p_actor,p_token_finalizacion,simples);
 IF existentes=0 THEN
  INSERT INTO vec_bolsa_llamamientos.contacto_fuente_correo(recibo_ref,fuente,motivo,correo_ref)
  SELECT c.value->>'recibo_ref',c.value#>>'{fuente_correo,fuente}',c.value#>>'{fuente_correo,motivo}',c.value#>>'{fuente_correo,correo_ref}'
    FROM jsonb_array_elements(p_contactos) c
  ON CONFLICT (recibo_ref) DO NOTHING;
 END IF;
 RETURN resultado||jsonb_build_object('fuentes_correo',vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(p_bolsa,p_clave));
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.candidato_participacion_avisos_v1(text,text,text),
 vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(text,text),
 vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.candidato_participacion_avisos_v1(text,text,text),
 vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(text,text),
 vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;

DO $post$
DECLARE f regprocedure;
BEGIN
 FOREACH f IN ARRAY ARRAY['vec_bolsa_llamamientos.candidato_participacion_avisos_v1(text,text,text)'::regprocedure,
   'vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(text,text)'::regprocedure,
   'vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb)'::regprocedure] LOOP
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::regrole)))
  THEN RAISE EXCEPTION 'B59: ACL incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
