\set ON_ERROR_STOP on
-- Usuarios 000008 (Fase 1, paso 3 del estudio de datos personales):
-- «Mis preferencias» (5.08a) y «Mi imagen» (5.08c) dejan de compartirse entre
-- el portal interno (RRHH, superficie interna_corporativa) y el Área personal
-- (superficie externa_personal). La superficie entra en la clave de estado,
-- historia y recibos: una fila por persona y superficie. Las políticas de fila
-- exigen además que la fila sea de la superficie del contexto V3 consumido, de
-- modo que una función que olvidara filtrar tampoco vería la otra superficie.
--
-- Filas existentes: la superficie de cada versión de la historia y de cada
-- recibo se toma de la decisión V3 firmada que la autorizó
-- (vec_autorizacion_atestada_v3.atestacion_decision_v3, vínculo de
-- autenticación). Es una lectura única del DBA en esta migración; en
-- ejecución Usuarios no lee tablas de V3. Si una decisión falta o es ambigua,
-- la migración se detiene (55000) sin cambiar nada.
-- El estado vigente de cada superficie es su última versión propia. Una
-- persona que usó los dos portales conserva en cada uno lo último que guardó
-- en él. Ninguna fila de historia ni recibo cambia salvo por la columna nueva
-- (se comprueba). Si el estado de imagen de una superficie apunta a una foto
-- que subió la otra o que ya se retiró, esa superficie vuelve a iniciales con
-- su misma paleta y la historia lo anota con una versión nueva marcada
-- «migracion:usuarios:000008».
--
-- Una sola transacción: cualquier fallo la revierte entera. Aplicar como DBA
-- después de Documentos 000008. Firmas, ACL y respuestas no cambian.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000008',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_usuarios.preferencias_recibo') IS NULL
    OR to_regclass('vec_usuarios.imagen_recibo') IS NULL
    OR to_regclass('vec_autorizacion_atestada_v3.atestacion_decision_v3') IS NULL
    OR to_regprocedure('vec_documentos.superficie_sesion_imagen_v1()') IS NULL
    OR to_regprocedure('vec_usuarios.contexto_autorizado(text,text[])') IS NULL
    OR to_regprocedure('vec_usuarios.contexto_autorizado_imagen(text,text[])') IS NULL
    OR to_regclass('vec_usuarios.preferencias_actual') IS NULL OR to_regclass('vec_usuarios.preferencias_historia') IS NULL
    OR to_regclass('vec_usuarios.imagen_actual') IS NULL OR to_regclass('vec_usuarios.imagen_historia') IS NULL
    OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid IN (to_regclass('vec_usuarios.preferencias_actual'),
      to_regclass('vec_usuarios.preferencias_historia'),to_regclass('vec_usuarios.preferencias_recibo'),
      to_regclass('vec_usuarios.imagen_actual'),to_regclass('vec_usuarios.imagen_historia'),to_regclass('vec_usuarios.imagen_recibo'))
      AND attname='superficie' AND NOT attisdropped)
    OR (SELECT count(*) FROM pg_constraint WHERE connamespace='vec_usuarios'::regnamespace AND conname IN (
      'preferencias_actual_pkey','preferencias_historia_pkey','preferencias_historia_persona_ref_fkey',
      'preferencias_recibo_pkey','preferencias_recibo_persona_ref_version_fkey',
      'imagen_actual_pkey','imagen_historia_pkey','imagen_historia_persona_ref_fkey',
      'imagen_recibo_pkey','imagen_recibo_persona_ref_version_fkey'))<>10
 THEN RAISE EXCEPTION 'Usuarios 000008: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_usuarios.preferencias_actual,vec_usuarios.preferencias_historia,vec_usuarios.preferencias_recibo,
 vec_usuarios.contexto_transaccion,vec_usuarios.imagen_actual,vec_usuarios.imagen_historia,vec_usuarios.imagen_recibo,
 vec_usuarios.imagen_contexto IN ACCESS EXCLUSIVE MODE;
DO $vivo$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_usuarios.contexto_transaccion) OR EXISTS (SELECT 1 FROM vec_usuarios.imagen_contexto)
 THEN RAISE EXCEPTION 'Usuarios 000008: hay contextos abiertos' USING ERRCODE='55000'; END IF;
END $vivo$;

-- 1. Superficie de cada decisión según su vínculo V3 firmado.
CREATE TEMP TABLE u8_decision ON COMMIT DROP AS
SELECT a.decision_ref,(convert_from(a.decision_canonica,'UTF8')::jsonb)#>>'{vinculo_autenticacion_actor,superficie}' AS superficie
FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
WHERE a.decision_ref IN (SELECT decision_ref FROM vec_usuarios.preferencias_historia
 UNION SELECT decision_ref FROM vec_usuarios.preferencias_recibo
 UNION SELECT decision_ref FROM vec_usuarios.imagen_historia
 UNION SELECT decision_ref FROM vec_usuarios.imagen_recibo);
DO $mapa$ BEGIN
 IF EXISTS (SELECT 1 FROM u8_decision WHERE superficie IS NULL OR superficie NOT IN ('interna_corporativa','externa_personal'))
    OR EXISTS (SELECT 1 FROM (SELECT decision_ref FROM vec_usuarios.preferencias_historia
      UNION SELECT decision_ref FROM vec_usuarios.preferencias_recibo
      UNION SELECT decision_ref FROM vec_usuarios.imagen_historia
      UNION SELECT decision_ref FROM vec_usuarios.imagen_recibo) q
     WHERE NOT EXISTS (SELECT 1 FROM u8_decision d WHERE d.decision_ref=q.decision_ref))
 THEN RAISE EXCEPTION 'Usuarios 000008: decisión sin superficie deducible; revisar antes de migrar' USING ERRCODE='55000'; END IF;
END $mapa$;

-- 2. Copias de control de historia, recibos y estado.
CREATE TEMP TABLE u8_antes ON COMMIT DROP AS
SELECT 'preferencias_historia'::text AS tabla,persona_ref||'|'||version AS clave,to_jsonb(t) AS fila FROM vec_usuarios.preferencias_historia t
UNION ALL SELECT 'preferencias_recibo',persona_ref||'|'||clave_operacion,to_jsonb(t) FROM vec_usuarios.preferencias_recibo t
UNION ALL SELECT 'preferencias_actual',persona_ref,to_jsonb(t) FROM vec_usuarios.preferencias_actual t
UNION ALL SELECT 'imagen_historia',persona_ref||'|'||version,to_jsonb(t) FROM vec_usuarios.imagen_historia t
UNION ALL SELECT 'imagen_recibo',persona_ref||'|'||clave_operacion,to_jsonb(t) FROM vec_usuarios.imagen_recibo t
UNION ALL SELECT 'imagen_actual',persona_ref,to_jsonb(t) FROM vec_usuarios.imagen_actual t;

-- 3. Claves: se retiran las antiguas antes de reconstruirlas con superficie.
ALTER TABLE vec_usuarios.preferencias_recibo DROP CONSTRAINT preferencias_recibo_persona_ref_version_fkey;
ALTER TABLE vec_usuarios.preferencias_historia DROP CONSTRAINT preferencias_historia_persona_ref_fkey;
ALTER TABLE vec_usuarios.preferencias_recibo DROP CONSTRAINT preferencias_recibo_pkey;
ALTER TABLE vec_usuarios.preferencias_historia DROP CONSTRAINT preferencias_historia_pkey;
ALTER TABLE vec_usuarios.preferencias_actual DROP CONSTRAINT preferencias_actual_pkey;
ALTER TABLE vec_usuarios.imagen_recibo DROP CONSTRAINT imagen_recibo_persona_ref_version_fkey;
ALTER TABLE vec_usuarios.imagen_historia DROP CONSTRAINT imagen_historia_persona_ref_fkey;
ALTER TABLE vec_usuarios.imagen_recibo DROP CONSTRAINT imagen_recibo_pkey;
ALTER TABLE vec_usuarios.imagen_historia DROP CONSTRAINT imagen_historia_pkey;
ALTER TABLE vec_usuarios.imagen_actual DROP CONSTRAINT imagen_actual_pkey;
ALTER TABLE vec_usuarios.preferencias_actual ADD COLUMN superficie text;
ALTER TABLE vec_usuarios.preferencias_historia ADD COLUMN superficie text;
ALTER TABLE vec_usuarios.preferencias_recibo ADD COLUMN superficie text;
ALTER TABLE vec_usuarios.imagen_actual ADD COLUMN superficie text;
ALTER TABLE vec_usuarios.imagen_historia ADD COLUMN superficie text;
ALTER TABLE vec_usuarios.imagen_recibo ADD COLUMN superficie text;

-- 4. Historia y recibos: solo se rellena la columna nueva. Los disparadores
-- de inmutabilidad se suspenden únicamente para esta asignación.
ALTER TABLE vec_usuarios.preferencias_historia DISABLE TRIGGER historia_inmutable;
ALTER TABLE vec_usuarios.preferencias_recibo DISABLE TRIGGER recibo_inmutable;
ALTER TABLE vec_usuarios.imagen_historia DISABLE TRIGGER imagen_historia_inmutable;
ALTER TABLE vec_usuarios.imagen_recibo DISABLE TRIGGER imagen_recibo_inmutable;
UPDATE vec_usuarios.preferencias_historia t SET superficie=d.superficie FROM u8_decision d WHERE d.decision_ref=t.decision_ref;
UPDATE vec_usuarios.preferencias_recibo t SET superficie=d.superficie FROM u8_decision d WHERE d.decision_ref=t.decision_ref;
UPDATE vec_usuarios.imagen_historia t SET superficie=d.superficie FROM u8_decision d WHERE d.decision_ref=t.decision_ref;
UPDATE vec_usuarios.imagen_recibo t SET superficie=d.superficie FROM u8_decision d WHERE d.decision_ref=t.decision_ref;
ALTER TABLE vec_usuarios.preferencias_historia ENABLE TRIGGER historia_inmutable;
ALTER TABLE vec_usuarios.preferencias_recibo ENABLE TRIGGER recibo_inmutable;
ALTER TABLE vec_usuarios.imagen_historia ENABLE TRIGGER imagen_historia_inmutable;
ALTER TABLE vec_usuarios.imagen_recibo ENABLE TRIGGER imagen_recibo_inmutable;
DO $coherencia$ BEGIN
 -- Cada recibo nace en la misma transacción que su versión: misma decisión.
 IF EXISTS (SELECT 1 FROM vec_usuarios.preferencias_recibo r JOIN vec_usuarios.preferencias_historia h
     ON h.persona_ref=r.persona_ref AND h.version=r.version WHERE h.superficie IS DISTINCT FROM r.superficie)
    OR EXISTS (SELECT 1 FROM vec_usuarios.imagen_recibo r JOIN vec_usuarios.imagen_historia h
     ON h.persona_ref=r.persona_ref AND h.version=r.version WHERE h.superficie IS DISTINCT FROM r.superficie)
 THEN RAISE EXCEPTION 'Usuarios 000008: recibo y versión con superficies distintas' USING ERRCODE='55000'; END IF;
END $coherencia$;

-- 5. Estado vigente: la última versión propia de cada superficie. No hay
-- disparador de inmutabilidad en el estado; se reconstruye desde la historia.
DELETE FROM vec_usuarios.preferencias_actual;
INSERT INTO vec_usuarios.preferencias_actual(persona_ref,superficie,version,catalogo_version_ref,valores,recibo_ref,actualizado_en)
SELECT DISTINCT ON (persona_ref,superficie) persona_ref,superficie,version,catalogo_version_ref,valores,recibo_ref,registrada_en
FROM vec_usuarios.preferencias_historia ORDER BY persona_ref,superficie,version DESC;
DELETE FROM vec_usuarios.imagen_actual;
INSERT INTO vec_usuarios.imagen_actual(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,actualizado_en)
SELECT DISTINCT ON (persona_ref,superficie) persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,registrada_en
FROM vec_usuarios.imagen_historia ORDER BY persona_ref,superficie,version DESC;

-- 6. Foto ajena o ya retirada: la superficie vuelve a iniciales y se anota.
-- La foto es de la superficie que la subió (primera versión que la cita).
CREATE TEMP TABLE u8_foto_duena ON COMMIT DROP AS
SELECT DISTINCT ON (foto_ref) foto_ref,superficie FROM vec_usuarios.imagen_historia
WHERE foto_ref IS NOT NULL ORDER BY foto_ref,version;
CREATE TEMP TABLE u8_foto_rehecha ON COMMIT DROP AS
SELECT a.persona_ref,a.superficie,a.version+1 AS version,a.catalogo_version_ref,
 jsonb_build_object('modo','iniciales','paleta',a.eleccion->>'paleta','icono','') AS eleccion,
 'img_'||replace(gen_random_uuid()::text,'-','') AS recibo_ref
FROM vec_usuarios.imagen_actual a JOIN u8_foto_duena f ON f.foto_ref=a.foto_ref
WHERE f.superficie<>a.superficie
   OR EXISTS (SELECT 1 FROM vec_usuarios.imagen_historia h WHERE h.persona_ref=a.persona_ref AND h.foto_retirada_ref=a.foto_ref);
INSERT INTO vec_usuarios.imagen_historia(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,foto_retirada_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
SELECT persona_ref,superficie,version,catalogo_version_ref,eleccion,NULL,NULL,NULL,recibo_ref,
 'migracion:usuarios:000008','migracion:usuarios:000008',date_trunc('microseconds',clock_timestamp())
FROM u8_foto_rehecha;
UPDATE vec_usuarios.imagen_actual a SET version=r.version,eleccion=r.eleccion,foto_ref=NULL,foto_sha256=NULL,
 recibo_ref=r.recibo_ref,actualizado_en=h.registrada_en
FROM u8_foto_rehecha r JOIN vec_usuarios.imagen_historia h ON h.recibo_ref=r.recibo_ref
WHERE a.persona_ref=r.persona_ref AND a.superficie=r.superficie;

-- 7. Claves nuevas: persona y superficie.
ALTER TABLE vec_usuarios.preferencias_actual ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.preferencias_historia ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.preferencias_recibo ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.imagen_actual ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.imagen_historia ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.imagen_recibo ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_usuarios.preferencias_actual ADD CONSTRAINT preferencias_actual_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.preferencias_historia ADD CONSTRAINT preferencias_historia_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.preferencias_recibo ADD CONSTRAINT preferencias_recibo_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.imagen_actual ADD CONSTRAINT imagen_actual_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.imagen_historia ADD CONSTRAINT imagen_historia_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.imagen_recibo ADD CONSTRAINT imagen_recibo_superficie_check CHECK(superficie IN ('interna_corporativa','externa_personal'));
ALTER TABLE vec_usuarios.preferencias_actual ADD CONSTRAINT preferencias_actual_pkey PRIMARY KEY(persona_ref,superficie);
ALTER TABLE vec_usuarios.preferencias_historia ADD CONSTRAINT preferencias_historia_pkey PRIMARY KEY(persona_ref,superficie,version);
ALTER TABLE vec_usuarios.preferencias_recibo ADD CONSTRAINT preferencias_recibo_pkey PRIMARY KEY(persona_ref,superficie,clave_operacion);
ALTER TABLE vec_usuarios.preferencias_historia ADD CONSTRAINT preferencias_historia_persona_ref_fkey
 FOREIGN KEY(persona_ref,superficie) REFERENCES vec_usuarios.preferencias_actual(persona_ref,superficie) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_usuarios.preferencias_recibo ADD CONSTRAINT preferencias_recibo_persona_ref_version_fkey
 FOREIGN KEY(persona_ref,superficie,version) REFERENCES vec_usuarios.preferencias_historia(persona_ref,superficie,version) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_usuarios.imagen_actual ADD CONSTRAINT imagen_actual_pkey PRIMARY KEY(persona_ref,superficie);
ALTER TABLE vec_usuarios.imagen_historia ADD CONSTRAINT imagen_historia_pkey PRIMARY KEY(persona_ref,superficie,version);
ALTER TABLE vec_usuarios.imagen_recibo ADD CONSTRAINT imagen_recibo_pkey PRIMARY KEY(persona_ref,superficie,clave_operacion);
ALTER TABLE vec_usuarios.imagen_historia ADD CONSTRAINT imagen_historia_persona_ref_fkey
 FOREIGN KEY(persona_ref,superficie) REFERENCES vec_usuarios.imagen_actual(persona_ref,superficie) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_usuarios.imagen_recibo ADD CONSTRAINT imagen_recibo_persona_ref_version_fkey
 FOREIGN KEY(persona_ref,superficie,version) REFERENCES vec_usuarios.imagen_historia(persona_ref,superficie,version) DEFERRABLE INITIALLY DEFERRED;

-- 8. Comprobación: nada de la historia ni de los recibos cambió salvo la
-- superficie; el estado de cada superficie es su última versión; y el
-- estado anterior de cada persona sigue vigente en la superficie que lo guardó.
DO $control$ BEGIN
 IF EXISTS (SELECT 1 FROM (SELECT persona_ref||'|'||version AS clave,to_jsonb(t)-'superficie' AS fila FROM vec_usuarios.preferencias_historia t) n
     FULL JOIN (SELECT clave,fila FROM u8_antes WHERE tabla='preferencias_historia') a USING(clave) WHERE n.fila IS DISTINCT FROM a.fila)
    OR EXISTS (SELECT 1 FROM (SELECT persona_ref||'|'||clave_operacion AS clave,to_jsonb(t)-'superficie' AS fila FROM vec_usuarios.preferencias_recibo t) n
     FULL JOIN (SELECT clave,fila FROM u8_antes WHERE tabla='preferencias_recibo') a USING(clave) WHERE n.fila IS DISTINCT FROM a.fila)
    OR EXISTS (SELECT 1 FROM (SELECT persona_ref||'|'||version AS clave,to_jsonb(t)-'superficie' AS fila FROM vec_usuarios.imagen_historia t
       WHERE decision_ref<>'migracion:usuarios:000008') n
     FULL JOIN (SELECT clave,fila FROM u8_antes WHERE tabla='imagen_historia') a USING(clave) WHERE n.fila IS DISTINCT FROM a.fila)
    OR EXISTS (SELECT 1 FROM (SELECT persona_ref||'|'||clave_operacion AS clave,to_jsonb(t)-'superficie' AS fila FROM vec_usuarios.imagen_recibo t) n
     FULL JOIN (SELECT clave,fila FROM u8_antes WHERE tabla='imagen_recibo') a USING(clave) WHERE n.fila IS DISTINCT FROM a.fila)
    OR EXISTS (SELECT 1 FROM u8_antes a WHERE a.tabla='preferencias_actual' AND NOT EXISTS (
      SELECT 1 FROM vec_usuarios.preferencias_actual n WHERE n.persona_ref=a.clave AND (to_jsonb(n)-'superficie')=a.fila))
    OR EXISTS (SELECT 1 FROM u8_antes a WHERE a.tabla='imagen_actual' AND NOT EXISTS (
      SELECT 1 FROM vec_usuarios.imagen_actual n WHERE n.persona_ref=a.clave
       AND ((to_jsonb(n)-'superficie')=a.fila OR n.recibo_ref IN (SELECT recibo_ref FROM u8_foto_rehecha))))
    OR (SELECT count(DISTINCT persona_ref) FROM vec_usuarios.preferencias_actual)<>(SELECT count(*) FROM u8_antes WHERE tabla='preferencias_actual')
    OR (SELECT count(DISTINCT persona_ref) FROM vec_usuarios.imagen_actual)<>(SELECT count(*) FROM u8_antes WHERE tabla='imagen_actual')
    OR EXISTS (SELECT 1 FROM vec_usuarios.preferencias_actual a WHERE a.version<>(SELECT max(version) FROM vec_usuarios.preferencias_historia h
      WHERE h.persona_ref=a.persona_ref AND h.superficie=a.superficie))
    OR EXISTS (SELECT 1 FROM vec_usuarios.imagen_actual a WHERE a.version<>(SELECT max(version) FROM vec_usuarios.imagen_historia h
      WHERE h.persona_ref=a.persona_ref AND h.superficie=a.superficie))
    -- Cada foto vigente está viva en Documentos, es de la misma persona y
    -- Documentos 000008 la asignó al mismo portal (por su LOGIN de custodia).
    OR EXISTS (SELECT 1 FROM vec_usuarios.imagen_actual a WHERE a.foto_ref IS NOT NULL AND NOT EXISTS (
      SELECT 1 FROM vec_documentos.imagen_personal d WHERE d.imagen_ref=a.foto_ref AND d.persona_ref=a.persona_ref
       AND d.superficie=a.superficie AND d.estado='activa' AND d.huella_sha256=a.foto_sha256))
 THEN RAISE EXCEPTION 'Usuarios 000008: la migración alteraría la historia o el estado' USING ERRCODE='55000'; END IF;
END $control$;

SET LOCAL ROLE vec_usuarios_propietario;
-- 9. Políticas de fila: persona, superficie y modo del contexto V3.
CREATE FUNCTION vec_usuarios.contexto_autorizado_superficie(p_persona text,p_superficie text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_usuarios.contexto_transaccion c
  WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid()
    AND c.sesion=session_user AND c.persona_ref=p_persona AND c.superficie=p_superficie AND c.modo=ANY(p_modos)
    AND vec_usuarios.sesion_superficie_valida(c.superficie))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado_superficie(text,text,text[]) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios.contexto_autorizado_imagen_superficie(p_persona text,p_superficie text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto c
 WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid()
 AND c.sesion=session_user AND c.superficie=vec_usuarios.superficie_sesion_correos()
 AND c.persona_ref=p_persona AND c.superficie=p_superficie AND c.modo=ANY(p_modos))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado_imagen_superficie(text,text,text[]) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;

DROP POLICY estado_lectura ON vec_usuarios.preferencias_actual;
DROP POLICY estado_alta ON vec_usuarios.preferencias_actual;
DROP POLICY estado_cambio ON vec_usuarios.preferencias_actual;
DROP POLICY historia_alta ON vec_usuarios.preferencias_historia;
DROP POLICY recibo_lectura ON vec_usuarios.preferencias_recibo;
DROP POLICY recibo_alta ON vec_usuarios.preferencias_recibo;
CREATE POLICY estado_lectura ON vec_usuarios.preferencias_actual FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['consultar','actualizar']));
CREATE POLICY estado_alta ON vec_usuarios.preferencias_actual FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY estado_cambio ON vec_usuarios.preferencias_actual FOR UPDATE TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['actualizar']))
 WITH CHECK (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY historia_alta ON vec_usuarios.preferencias_historia FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY recibo_lectura ON vec_usuarios.preferencias_recibo FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['recuperar','actualizar']));
CREATE POLICY recibo_alta ON vec_usuarios.preferencias_recibo FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_superficie(persona_ref,superficie,ARRAY['actualizar']));
DROP POLICY imagen_actual_lectura ON vec_usuarios.imagen_actual;
DROP POLICY imagen_actual_alta ON vec_usuarios.imagen_actual;
DROP POLICY imagen_actual_cambio ON vec_usuarios.imagen_actual;
DROP POLICY imagen_historia_alta ON vec_usuarios.imagen_historia;
DROP POLICY imagen_recibo_lectura ON vec_usuarios.imagen_recibo;
DROP POLICY imagen_recibo_alta ON vec_usuarios.imagen_recibo;
CREATE POLICY imagen_actual_lectura ON vec_usuarios.imagen_actual FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['consultar','actualizar']));
CREATE POLICY imagen_actual_alta ON vec_usuarios.imagen_actual FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_actual_cambio ON vec_usuarios.imagen_actual FOR UPDATE TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']))
 WITH CHECK (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_historia_alta ON vec_usuarios.imagen_historia FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_recibo_lectura ON vec_usuarios.imagen_recibo FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['recuperar','actualizar']));
CREATE POLICY imagen_recibo_alta ON vec_usuarios.imagen_recibo FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
-- Los predicados sin superficie ya no tienen consumidores.
DROP FUNCTION vec_usuarios.contexto_autorizado(text,text[]);
DROP FUNCTION vec_usuarios.contexto_autorizado_imagen(text,text[]);

-- 10. Fachadas de preferencias: misma firma y respuesta; la superficie del
-- material (ligada al LOGIN y al vínculo V3) filtra cada lectura y escritura.
CREATE OR REPLACE FUNCTION vec_usuarios.consultar_preferencias_propias_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; x record; e record; c record; respuesta jsonb;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.consultar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref',m->>'superficie','consultar',x.decision_ref,x.consumo_huella_sha256);
 SELECT p.version_ref,p.definicion INTO STRICT c
 FROM vec_usuarios.catalogo_publicacion b JOIN vec_usuarios.catalogo_preferencias p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF m->>'catalogo_version_ref' IS DISTINCT FROM c.version_ref
 THEN RAISE EXCEPTION 'Usuarios: catálogo obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,valores INTO e FROM vec_usuarios.preferencias_actual
 WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie';
 IF FOUND THEN
  respuesta:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
    'catalogo_version_ref',e.catalogo_version_ref,'valores',e.valores);
 ELSE
  respuesta:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',c.version_ref,'valores',c.definicion->'predeterminados');
 END IF;
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref',m->>'superficie','consultar');
 RETURN respuesta;
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.recuperar_preferencias_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r record; x record; respuesta jsonb;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.actualizar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: recuperación denegada' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref',m->>'superficie','recuperar',x.decision_ref,x.consumo_huella_sha256);
 SELECT * INTO r FROM vec_usuarios.preferencias_recibo
  WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie' AND clave_operacion=m->>'clave_operacion';
 IF NOT FOUND THEN
  PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref',m->>'superficie','recuperar');
  RETURN NULL;
 END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 respuesta:=jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'version',r.version,
   'catalogo_version_ref',r.catalogo_version_ref,'valores',r.valores,'fecha_utc',r.registrada_en,'replay',true);
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref',m->>'superficie','recuperar');
 RETURN respuesta;
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.guardar_preferencias_propias_v1(
 p_material text,p_valores jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c record; previo record; x record; v bigint; ref text; ahora timestamptz(6); n integer; respuesta jsonb; s text;
BEGIN
 m:=vec_usuarios.validar_material_preferencias(p_material,'vec.preferencias.actualizar',p_capacidad,p_decision,p_persona_version,p_perfil_version);
 s:=m->>'superficie';
 IF p_valores IS DISTINCT FROM m->'valores' OR vec_usuarios.valores_validos(p_valores) IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios: valores divergentes' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_preferencias(p_material)
 THEN RAISE EXCEPTION 'Usuarios: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_usuarios.activar_contexto_preferencias(m->>'persona_ref',s,'actualizar',x.decision_ref,x.consumo_huella_sha256);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:preferencias:'||s||':'||(m->>'persona_ref'),0));
 IF EXISTS (SELECT 1 FROM vec_usuarios.preferencias_recibo WHERE persona_ref=m->>'persona_ref' AND superficie=s AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 SELECT p.version_ref,p.definicion,p.huella_sha256 INTO STRICT c
 FROM vec_usuarios.catalogo_publicacion b JOIN vec_usuarios.catalogo_preferencias p USING(version_ref)
 ORDER BY b.secuencia DESC LIMIT 1;
 IF m->>'catalogo_version_ref' IS DISTINCT FROM c.version_ref
 THEN RAISE EXCEPTION 'Usuarios: catálogo obsoleto' USING ERRCODE='P1409'; END IF;
 IF encode(sha256(convert_to(c.definicion::text,'UTF8')),'hex') IS DISTINCT FROM c.huella_sha256
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'idiomas') q WHERE q->>'codigo'=p_valores->>'idioma'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'tamanos_texto') q WHERE q->>'codigo'=p_valores->>'tamano_texto'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'temas') q WHERE q->>'codigo'=p_valores->>'tema'))
    OR NOT (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements(c.definicion->'inicios') q WHERE q->>'codigo'=p_valores->>'inicio'))
    OR NOT (c.definicion->'filas') @> jsonb_build_array((p_valores->>'filas')::integer)
 THEN RAISE EXCEPTION 'Usuarios: valores fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version INTO previo FROM vec_usuarios.preferencias_actual WHERE persona_ref=m->>'persona_ref' AND superficie=s FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='pref_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.preferencias_actual(persona_ref,superficie,version,catalogo_version_ref,valores,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',s,v,c.version_ref,p_valores,ref,ahora);
 ELSE
  UPDATE vec_usuarios.preferencias_actual SET version=v,catalogo_version_ref=c.version_ref,
   valores=p_valores,recibo_ref=ref,actualizado_en=ahora WHERE persona_ref=m->>'persona_ref' AND superficie=s AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión en conflicto' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_usuarios.preferencias_historia(persona_ref,superficie,version,catalogo_version_ref,valores,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',s,v,c.version_ref,p_valores,ref,x.decision_ref,x.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.preferencias_recibo(persona_ref,superficie,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,valores,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',s,m->>'clave_operacion',m->>'huella_peticion',ref,v,c.version_ref,p_valores,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256,ahora);
 respuesta:=jsonb_build_object('recibo_ref',ref,'persona_ref',m->>'persona_ref','version',v,
   'catalogo_version_ref',c.version_ref,'valores',p_valores,'fecha_utc',ahora,'replay',false);
 PERFORM vec_usuarios.retirar_contexto_preferencias(m->>'persona_ref',s,'actualizar');
 RETURN respuesta;
END $f$;

-- 11. Fachadas de imagen: igual. La foto que Documentos custodia también es
-- de la superficie de la sesión (Documentos 000008).
CREATE OR REPLACE FUNCTION vec_usuarios.consultar_imagen_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(datos jsonb,foto bytea) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; e record; cat jsonb; b bytea;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,eleccion,foto_ref INTO e FROM vec_usuarios.imagen_actual
 WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie';
 IF FOUND THEN
  IF e.foto_ref IS NOT NULL THEN b:=vec_documentos.abrir_imagen_personal_v1(m->>'persona_ref',e.foto_ref); END IF;
  datos:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
   'catalogo_version_ref',e.catalogo_version_ref,'eleccion',e.eleccion);
 ELSE
  datos:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',cat->>'version_ref','eleccion',cat->'predeterminada');
 END IF;
 foto:=b;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','consultar');
 RETURN NEXT;
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.recuperar_imagen_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r vec_usuarios.imagen_recibo;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'recuperar');
 SELECT * INTO r FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie'
  AND clave_operacion=m->>'clave_operacion';
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','recuperar');
 IF r.recibo_ref IS NULL THEN RETURN NULL; END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN vec_usuarios.recibo_imagen_json(r,true);
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.guardar_imagen_propia_v1(
 p_material text,p_foto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; cx record; cat jsonb; previo record; v bigint; ref text; ahora timestamptz(6); n integer;
 v_foto_ref text; v_foto_sha text; v_retirada text; r vec_usuarios.imagen_recibo; s text;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'actualizar');
 s:=m->>'superficie';
 IF (m->>'foto_sha256'='')<>(p_foto IS NULL)
    OR (p_foto IS NOT NULL AND encode(sha256(p_foto),'hex') IS DISTINCT FROM m->>'foto_sha256')
 THEN RAISE EXCEPTION 'Usuarios: foto divergente del material' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:imagen:'||s||':'||(m->>'persona_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND superficie=s AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'paletas') q WHERE q->>'codigo'=m#>>'{eleccion,paleta}')
    OR (m#>>'{eleccion,modo}'='icono' AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'iconos') q WHERE q->>'codigo'=m#>>'{eleccion,icono}'))
 THEN RAISE EXCEPTION 'Usuarios: imagen fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version,foto_ref,foto_sha256 INTO previo FROM vec_usuarios.imagen_actual WHERE persona_ref=m->>'persona_ref' AND superficie=s FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión de imagen en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='img_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 v_foto_ref:=previo.foto_ref; v_foto_sha:=previo.foto_sha256;
 IF m#>>'{eleccion,modo}'='foto' AND p_foto IS NULL AND previo.foto_ref IS NULL
 THEN RAISE EXCEPTION 'Usuarios: no hay foto que conservar' USING ERRCODE='P1409'; END IF;
 IF previo.foto_ref IS NOT NULL AND (p_foto IS NOT NULL OR m#>>'{eleccion,modo}'<>'foto') THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1(m->>'persona_ref',previo.foto_ref,ref);
  v_retirada:=previo.foto_ref; v_foto_ref:=NULL; v_foto_sha:=NULL;
 END IF;
 IF p_foto IS NOT NULL THEN
  v_foto_ref:=vec_documentos.custodiar_imagen_personal_v1(m->>'persona_ref',p_foto,m->>'foto_sha256',ref);
  v_foto_sha:=m->>'foto_sha256';
 END IF;
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.imagen_actual(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,ref,ahora);
 ELSE
  UPDATE vec_usuarios.imagen_actual SET version=v,catalogo_version_ref=cat->>'version_ref',eleccion=m->'eleccion',
   foto_ref=v_foto_ref,foto_sha256=v_foto_sha,recibo_ref=ref,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND superficie=s AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión de imagen concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT cx FROM vec_usuarios.imagen_contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo='actualizar' AND superficie=s;
 INSERT INTO vec_usuarios.imagen_historia(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,foto_retirada_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,v_retirada,ref,cx.decision_ref,cx.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.imagen_recibo(persona_ref,superficie,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,eleccion,foto_nueva,foto_retirada,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',s,m->>'clave_operacion',m->>'huella_peticion',ref,v,cat->>'version_ref',m->'eleccion',p_foto IS NOT NULL,v_retirada IS NOT NULL,cx.decision_ref,cx.auditoria_ref,cx.consumo_huella_sha256,ahora)
 RETURNING * INTO r;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','actualizar');
 RETURN vec_usuarios.recibo_imagen_json(r,false);
END $f$;
RESET ROLE;

-- 12. ACL: los ejecutores siguen con las mismas fachadas y sin tablas; las
-- funciones reemplazadas conservan propietario, SECURITY DEFINER y permisos.
DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_usuarios'::regnamespace
     AND p.proname IN ('consultar_preferencias_propias_v1','recuperar_preferencias_operacion_v1','guardar_preferencias_propias_v1',
       'consultar_imagen_propia_v1','recuperar_imagen_operacion_v1','guardar_imagen_propia_v1',
       'contexto_autorizado_superficie','contexto_autorizado_imagen_superficie')
     AND (p.proowner<>'vec_usuarios_propietario'::regrole OR NOT p.prosecdef))
    OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_usuarios'::regnamespace
     AND p.proname IN ('contexto_autorizado_superficie','contexto_autorizado_imagen_superficie')
     AND (has_function_privilege('vec_usuarios_ejecutor_interno',p.oid,'EXECUTE') OR has_function_privilege('vec_usuarios_ejecutor_externo',p.oid,'EXECUTE')))
    OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_usuarios'::regnamespace AND c.relkind='r'
     AND (c.relname LIKE 'preferencias_%' OR c.relname LIKE 'imagen_%')
     AND (has_table_privilege('vec_usuarios_ejecutor_interno',c.oid,'SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_usuarios_ejecutor_externo',c.oid,'SELECT,INSERT,UPDATE,DELETE')))
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios'
     AND (tablename LIKE 'preferencias_%' OR tablename LIKE 'imagen_%') AND roles<>ARRAY['vec_usuarios_propietario']::name[])
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios'
     AND tablename IN ('preferencias_actual','preferencias_historia','preferencias_recibo','imagen_actual','imagen_historia','imagen_recibo')
     AND position('superficie' IN coalesce(qual,'')||coalesce(with_check,''))=0)
    OR NOT has_function_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_ejecutor_externo','vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'Usuarios 000008: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
