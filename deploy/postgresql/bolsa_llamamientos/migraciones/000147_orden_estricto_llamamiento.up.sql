\set ON_ERROR_STOP on
-- B147 (F1-110): la emisión general de llamamientos respeta el orden estricto
-- entre las personas disponibles (regla b01, art. 8.2.a del Reglamento).
-- * saltos_orden_llamamiento_v1 calcula, con el mismo orden vigente B6 que
--   ve el asistente, quién va antes de la última persona elegida y no está en
--   el llamamiento. No cuentan quienes ya tienen un llamamiento emitido en la
--   bolsa sin situación registrada después ni quienes la política de avisos
--   excluye por prestar servicios (B41).
-- * reservar_llamamiento_v2 es la reserva B7 instalada (B17+B18+B25) con esa
--   comprobación antes de consumir la autorización: sin motivo, un salto se
--   rechaza con SQLSTATE VBL02 (llamamiento_fuera_de_orden) y la primera
--   persona saltada en DETAIL. Un salto solo se admite con un motivo de la
--   entrada b01.salto.* del catálogo de reglas, que la aplicación resuelve y
--   esta función guarda junto al llamamiento, ligado a su decisión y
--   auditoría V3. Hoy el catálogo no tiene ninguna: el orden es estricto.
-- * reservar_llamamiento_v1 conserva firma, OID y ACL y delega en v2 sin
--   motivo. La reserva nativa de Contratación (B110) ya exige el turno 1.
-- Sin DOWN ni reaplicación.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000147',0));

DO $pre$
DECLARE p record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.salto_orden_llamamiento') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.saltos_orden_llamamiento_v1(text,jsonb,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.presta_servicios_en_v1(timestamptz)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'B147: preimagen incompatible' USING ERRCODE='55000'; END IF;
 -- La reserva instalada debe ser exactamente la de B17 ajustada por B18 y B25.
 SELECT pg_get_userbyid(f.proowner) AS dueno,f.prosecdef,f.provolatile,f.proacl::text AS acl,f.proconfig,
        encode(sha256(convert_to(pg_get_functiondef(f.oid),'UTF8')),'hex') AS huella
   INTO STRICT p FROM pg_proc f
  WHERE f.oid='vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.dueno<>'vec_bolsa_llamamientos_propietario' OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
 OR p.acl IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']::text[]
 OR p.huella<>'e125985f7257cee63fc515173ef92fe91c87465aae05bf074d6d2b1280de640f'
 THEN RAISE EXCEPTION 'B147: reserva B7 instalada distinta de la esperada: %',p.huella USING ERRCODE='55000'; END IF;
END $pre$;

-- Motivo del salto, una fila por llamamiento que lo necesitó. Inmutable.
CREATE TABLE vec_bolsa_llamamientos.salto_orden_llamamiento(
 llamamiento_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
 bolsa_ref text NOT NULL,
 motivo_clave text NOT NULL CHECK(motivo_clave ~ '^b01\.salto\.[a-z0-9_]{1,64}$'),
 regla_ref text NOT NULL CHECK(octet_length(regla_ref) BETWEEN 1 AND 512 AND right(regla_ref,length(motivo_clave)+1)=':'||motivo_clave),
 huella_catalogo_sha256 text NOT NULL CHECK(huella_catalogo_sha256 ~ '^[0-9a-f]{64}$'),
 saltadas jsonb NOT NULL CHECK(jsonb_typeof(saltadas)='array' AND jsonb_array_length(saltadas) BETWEEN 1 AND 20000),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrado_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.salto_orden_llamamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.salto_orden_llamamiento FORCE ROW LEVEL SECURITY;
CREATE POLICY salto_orden_llamamiento_solo_propietario ON vec_bolsa_llamamientos.salto_orden_llamamiento
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.salto_orden_llamamiento FROM PUBLIC;
CREATE TRIGGER salto_orden_llamamiento_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.salto_orden_llamamiento FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Personas en turno antes de la última elegida que no están en el llamamiento,
-- por orden vigente. Invocadora y sin concesiones: solo la llama la reserva.
CREATE FUNCTION vec_bolsa_llamamientos.saltos_orden_llamamiento_v1(p_bolsa text,p_participaciones jsonb,p_emitido timestamptz)
RETURNS jsonb LANGUAGE sql STABLE
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
 WITH orden AS MATERIALIZED (
  SELECT o.participacion_ref,o.orden_vigente
    FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o
   WHERE o.orden_vigente IS NOT NULL
 ), ultima AS (
  SELECT max(o.orden_vigente) AS orden FROM orden o WHERE p_participaciones ? o.participacion_ref
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object('participacion_ref',o.participacion_ref,'orden_vigente',o.orden_vigente)
          ORDER BY o.orden_vigente),'[]'::jsonb)
   FROM orden o CROSS JOIN ultima u
  WHERE o.orden_vigente<u.orden
    AND NOT (p_participaciones ? o.participacion_ref)
    -- La política de avisos (B41) en «excluir» impide llamarla: no bloquea.
    AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.presta_servicios_en_v1(p_emitido) x
                    WHERE x.modo='excluir' AND x.participacion_ref=o.participacion_ref)
    -- Ya llamada en esta bolsa y sin situación registrada después: su turno
    -- está en curso y no se le vuelve a exigir.
    AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido l
                    WHERE l.bolsa_ref=p_bolsa AND l.emitido_en<=p_emitido
                      AND l.participaciones ? o.participacion_ref
                      AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                                      WHERE s.participacion_ref=o.participacion_ref
                                        AND s.registrada_en>l.emitido_en AND s.registrada_en<=p_emitido
                                        AND s.desde<=p_emitido))
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.saltos_orden_llamamiento_v1(text,jsonb,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v2(p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,p_huella_finalizacion bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_salto jsonb)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE previo record; consumo record; d jsonb; v_huella text; v_total int; v_validas int; v_ordenadas int; v_contactos jsonb; v_saltadas jsonb; v_salto_previo jsonb; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$' OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$' OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones) NOT BETWEEN 1 AND 100 OR NOT(SELECT bool_and(jsonb_typeof(value)='string') FROM jsonb_array_elements(p_participaciones)) OR jsonb_typeof(p_configuracion)<>'object' OR p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad','fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE OR NOT(SELECT count(*)=10 AND bool_and(jsonb_typeof(value)='string' AND octet_length(value#>>'{}') BETWEEN 2 AND 4000 AND value#>>'{}'=btrim(value#>>'{}')) FROM jsonb_each(p_configuracion)) OR p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$' OR octet_length(p_huella_finalizacion) IS DISTINCT FROM 32 OR p_emitido IS NULL THEN RAISE EXCEPTION 'emisión B7 inválida' USING ERRCODE='22023'; END IF;
 -- El motivo de salto es la entrada del catálogo de reglas que la aplicación
 -- ha resuelto: clave, referencia versionada y huella del catálogo.
 IF p_salto IS NOT NULL AND (jsonb_typeof(p_salto)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p_salto))<>3
    OR (p_salto ?& ARRAY['clave','regla_ref','huella_catalogo']) IS NOT TRUE
    OR jsonb_typeof(p_salto->'clave')<>'string' OR jsonb_typeof(p_salto->'regla_ref')<>'string' OR jsonb_typeof(p_salto->'huella_catalogo')<>'string'
    OR p_salto->>'clave' !~ '^b01\.salto\.[a-z0-9_]{1,64}$'
    OR octet_length(p_salto->>'regla_ref') NOT BETWEEN 1 AND 512
    OR right(p_salto->>'regla_ref',length(p_salto->>'clave')+1)<>':'||(p_salto->>'clave')
    OR p_salto->>'huella_catalogo' !~ '^[0-9a-f]{64}$') THEN RAISE EXCEPTION 'motivo de salto inválido' USING ERRCODE='22023'; END IF;
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  SELECT jsonb_build_object('clave',s.motivo_clave,'regla_ref',s.regla_ref,'huella_catalogo',s.huella_catalogo_sha256) INTO v_salto_previo
    FROM vec_bolsa_llamamientos.salto_orden_llamamiento s WHERE s.llamamiento_ref=previo.llamamiento_ref;
  IF previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella OR v_salto_previo IS DISTINCT FROM p_salto THEN RAISE EXCEPTION 'clave B7 divergente' USING ERRCODE='VBE01'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('participacion_ref',c.participacion_ref,'resultado',c.resultado,'recibo_ref',c.recibo_ref) ORDER BY x.ordinality),'[]'::jsonb) INTO v_contactos FROM jsonb_array_elements_text(previo.participaciones) WITH ORDINALITY x(ref,ordinality) JOIN vec_bolsa_llamamientos.contacto_participacion c ON c.participacion_ref=x.ref AND c.llamamiento_ref=previo.llamamiento_ref AND c.canal='correo' AND c.clave_idempotencia=previo.clave_idempotencia||':correo:'||x.ordinality AND c.recibo_ref='recibo:contacto:'||encode(sha256(convert_to(previo.bolsa_ref||chr(31)||previo.clave_idempotencia||chr(31)||x.ref,'UTF8')),'hex');
  RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',previo.llamamiento_ref,'recibo_ref',previo.recibo_ref,'bolsa_ref',previo.bolsa_ref,'estado',CASE WHEN jsonb_array_length(v_contactos)=jsonb_array_length(previo.participaciones) THEN 'emitido_pendiente_respuesta' ELSE 'emision_reservada_resultado_pendiente' END,'participaciones',previo.participaciones,'configuracion',previo.configuracion,'emitido_en',previo.emitido_en,'contactos',v_contactos),true; RETURN;
 END IF;
 -- Regla b01: orden estricto entre los disponibles, antes de consumir nada.
 v_saltadas:=vec_bolsa_llamamientos.saltos_orden_llamamiento_v1(p_bolsa,p_participaciones,p_emitido);
 IF jsonb_array_length(v_saltadas)>0 AND p_salto IS NULL THEN
  RAISE EXCEPTION 'llamamiento fuera de orden' USING ERRCODE='VBL02',DETAIL=(v_saltadas->0)::text;
 END IF;
 IF jsonb_array_length(v_saltadas)=0 AND p_salto IS NOT NULL THEN RAISE EXCEPTION 'motivo de salto sin salto' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'decisión B7 inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN RAISE EXCEPTION 'emisión B7 no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa WHERE c.bolsa_ref=p_bolsa AND b.estado='vigente' AND b.vigente_desde<=p_emitido AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)) THEN RAISE EXCEPTION 'bolsa no vigente' USING ERRCODE='23503'; END IF;
 v_total:=jsonb_array_length(p_participaciones);
 SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL;
 SELECT count(*) INTO v_ordenadas FROM (SELECT o.orden_vigente AS orden,lag(o.orden_vigente) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL) q WHERE anterior IS NULL OR orden>anterior;
 IF v_validas<>v_total OR v_ordenadas<>v_total OR (SELECT count(DISTINCT value) FROM jsonb_array_elements_text(p_participaciones))<>v_total THEN RAISE EXCEPTION 'participaciones ajenas, repetidas o desordenadas' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref) VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,v_huella,p_huella_finalizacion,'emision_reservada',p_emitido,consumo.decision_ref);
 IF p_salto IS NOT NULL THEN
  INSERT INTO vec_bolsa_llamamientos.salto_orden_llamamiento(llamamiento_ref,bolsa_ref,motivo_clave,regla_ref,huella_catalogo_sha256,saltadas,decision_ref,auditoria_ref,registrado_en)
  VALUES(p_llamamiento,p_bolsa,p_salto->>'clave',p_salto->>'regla_ref',p_salto->>'huella_catalogo',v_saltadas,consumo.decision_ref,consumo.auditoria_ref,clock_timestamp());
 END IF;
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,'bolsa_ref',p_bolsa,'estado','emision_reservada_resultado_pendiente','participaciones',p_participaciones,'configuracion',p_configuracion,'emitido_en',p_emitido,'contactos','[]'::jsonb),false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;

-- v1 conserva firma, OID y ACL: quien la llame recibe la misma comprobación,
-- sin motivo de salto.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_v1(p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,p_huella_finalizacion bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
BEGIN
 RETURN QUERY SELECT r.emision,r.reutilizada FROM vec_bolsa_llamamientos.reservar_llamamiento_v2(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,p_emitido,p_huella_finalizacion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,NULL::jsonb) r;
END $f$;

DO $post$
DECLARE r record; f regprocedure; p record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',true,'{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','833e278079c944f61aad240d3da936804c0746c59ed4df1f85f1d8f20be40645'),
  ('vec_bolsa_llamamientos.reservar_llamamiento_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)',true,'{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}','9961668a8bb3406dd600f6cfbaaf02a4a12a23f37a453bc1e30a1157236756fa'),
  ('vec_bolsa_llamamientos.saltos_orden_llamamiento_v1(text,jsonb,timestamptz)',false,'{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}','83b6db86d21e321f5f2ea0af205d4856b93d95b75457d3978203c8ac05bdd686')
 ) AS esperado(firma,definidora,acl,huella) LOOP
  f:=to_regprocedure(r.firma);
  SELECT pg_get_userbyid(x.proowner) AS dueno,x.prosecdef,x.proacl::text AS acl,
         encode(sha256(convert_to(pg_get_functiondef(x.oid),'UTF8')),'hex') AS huella
    INTO STRICT p FROM pg_proc x WHERE x.oid=f;
  IF p.dueno<>'vec_bolsa_llamamientos_propietario' OR p.prosecdef IS DISTINCT FROM r.definidora
  OR p.acl IS DISTINCT FROM r.acl OR p.huella<>r.huella
  THEN RAISE EXCEPTION 'B147: posimagen divergente % %',r.firma,p.huella USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.salto_orden_llamamiento'::regclass)
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.salto_orden_llamamiento','SELECT')
 THEN RAISE EXCEPTION 'B147: tabla de saltos abierta' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
