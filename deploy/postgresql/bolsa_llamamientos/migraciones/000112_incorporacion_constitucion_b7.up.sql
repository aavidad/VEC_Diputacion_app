\set ON_ERROR_STOP on
-- B112: incorpora desde B7 original o desde B79 con recibo real, conservando B97.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000112',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'B112: migrador PostgreSQL18 requerido' USING ERRCODE='55000';END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.incorporar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 h:=encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex');
 IF h IS DISTINCT FROM 'e1805d4450e2b6dbb725571d8fc394cc90b8354d4b9119b57a71c48f8f164c5a'
 OR p.proowner<>to_regrole('vec_bolsa_llamamientos_propietario') OR p.prosecdef IS NOT TRUE
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
  'TimeZone=UTC','lock_timeout=2s','statement_timeout=20s']::text[]
 OR p.proacl IS DISTINCT FROM ARRAY[
  'vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario'::aclitem,
  'vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario'::aclitem]
 THEN RAISE EXCEPTION 'B112: preimagen divergente actual=% esperado=e1805d4450e2b6dbb725571d8fc394cc90b8354d4b9119b57a71c48f8f164c5a',h USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.incorporar_inscripcion_v1(p_material text, p_captura_actor jsonb, p_recurso_canonico bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '20s'
AS $function$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; recurso jsonb; v_conjunto jsonb; consumo record;
 s vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
 actual vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 aprobada vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 asociacion vec_bolsa_llamamientos.inscripcion_acta_asociada%ROWTYPE;
 vinculo vec_bolsa_llamamientos.inscripcion_incorporacion%ROWTYPE;
 previo vec_bolsa_llamamientos.solicitud_inscripcion_recibo%ROWTYPE;
 acta vec_bolsa_llamamientos.constitucion%ROWTYPE;
 carga vec_bolsa_llamamientos.recibo_carga_convoca%ROWTYPE;
 instantanea vec_bolsa_llamamientos.instantanea_orden_bolsa%ROWTYPE;
 bolsa vec_bolsa_llamamientos.bolsa_constituida%ROWTYPE;
 entrada record;
 publicacion jsonb; politica jsonb; ca jsonb; inversa jsonb; etiquetas jsonb;
 bolsa_canon jsonb; instantanea_canon jsonb;
 v_ref text; v_acta text; v_actor text; v_material_sha text; v_clave_sha text;
 v_recurso_sha text; v_candidato text; v_categoria text;
 v_historia text; v_evento text; v_recibo_ref text; v_instante timestamptz(6);
 v_esperada bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_bolsa_llamamientos_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 OR p_recurso_canonico IS NULL OR octet_length(p_recurso_canonico) NOT BETWEEN 2 AND 4096
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
 THEN RAISE EXCEPTION 'B97: incorporación no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  contexto:=convert_from(p_contexto,'UTF8')::jsonb;
  recurso:=convert_from(p_recurso_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B97: material inválido' USING ERRCODE='B9605';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
  'clave_idempotencia','esquema','evidencia_ref','solicitud_ref','version_esperada']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.incorporar.v1'
 OR coalesce(m->>'solicitud_ref','') !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR vec_bolsa_llamamientos.constitucion_texto_valido(m->>'evidencia_ref',512) IS NOT TRUE
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9_-][A-Za-z0-9._:-]{15,127}$'
 OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
 OR m->>'version_esperada' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'B97: incorporación inválida' USING ERRCODE='B9605'; END IF;
 v_ref:=m->>'solicitud_ref';
 v_acta:=m->>'evidencia_ref';
 v_esperada:=2;
 v_actor:=d->>'principal_id';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 -- Mismo recurso que application/inscripcion.RecursoIncorporacion: ámbito de
 -- gestión RRHH y atributos en cadena. Se coteja con la fuente vigente tras
 -- consumir V3, igual que la revisión B96.
 v_recurso_sha:=encode(sha256(p_recurso_canonico),'hex');
 IF jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso) ORDER BY 1) IS DISTINCT FROM ARRAY['ambitos','atributos']
 OR jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'ambitos') ORDER BY 1) IS DISTINCT FROM
   ARRAY['ambito_ref','unidad_ref']
 OR coalesce(recurso#>>'{ambitos,unidad_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR coalesce(recurso#>>'{ambitos,ambito_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'atributos') ORDER BY 1) IS DISTINCT FROM ARRAY[
  'conjunto_fuente_ref','conjunto_fuente_sha256','conjunto_fuente_version','conjunto_ref',
  'evidencia_ref','material_sha256','solicitud_fuente_ref','solicitud_fuente_sha256',
  'solicitud_fuente_version','solicitud_ref','version_esperada']
 OR EXISTS(SELECT 1 FROM jsonb_each(recurso->'atributos') AS a(clave,valor)
   WHERE jsonb_typeof(a.valor) IS DISTINCT FROM 'string')
 OR recurso#>>'{atributos,solicitud_ref}' IS DISTINCT FROM v_ref
 OR recurso#>>'{atributos,evidencia_ref}' IS DISTINCT FROM v_acta
 OR recurso#>>'{atributos,version_esperada}' IS DISTINCT FROM m->>'version_esperada'
 OR recurso#>>'{atributos,material_sha256}' IS DISTINCT FROM v_material_sha
 OR coalesce(recurso#>>'{atributos,conjunto_ref}','')=''
 OR coalesce(recurso#>>'{atributos,conjunto_fuente_ref}','')=''
 OR recurso#>>'{atributos,conjunto_fuente_version}' !~ '^[1-9][0-9]{0,9}$'
 OR recurso#>>'{atributos,conjunto_fuente_sha256}' !~ '^[0-9a-f]{64}$'
 OR coalesce(recurso#>>'{atributos,solicitud_fuente_ref}','')=''
 OR recurso#>>'{atributos,solicitud_fuente_version}' !~ '^[1-9][0-9]{0,9}$'
 OR recurso#>>'{atributos,solicitud_fuente_sha256}' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B97: recurso de incorporación incompatible' USING ERRCODE='42501'; END IF;
 IF coalesce(v_actor,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_actor
 OR coalesce(p_captura_actor->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(p_captura_actor->>'cuenta_ref','') !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_captura_actor->>'idioma' IS NULL
 OR p_captura_actor->>'idioma' NOT IN('es','en')
 OR contexto->>'principal_ref' IS DISTINCT FROM v_actor
 OR contexto->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitud_inscripcion'
 OR d->>'finalidad' IS DISTINCT FROM 'incorporar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.incorporar.v1'
 OR d->>'recurso_ref' IS DISTINCT FROM v_ref
 OR c->>'efecto_ref' IS DISTINCT FROM v_ref
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B97: decisión no exacta' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:inscripcion:'||v_ref,0));
 -- Cada reintento consume una decisión vigente distinta. Si cualquier fuente
 -- falla después, toda la transacción y su consumo vuelven atrás.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B97: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO s FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_ref FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'B97: solicitud ausente' USING ERRCODE='B9703'; END IF;
 IF v_actor IS NOT DISTINCT FROM s.persona_ref THEN
  RAISE EXCEPTION 'B97: el solicitante no puede incorporar' USING ERRCODE='42501';
 END IF;
 v_conjunto:=vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
 IF recurso#>>'{ambitos,unidad_ref}' IS DISTINCT FROM s.unidad_ref
 OR recurso#>>'{ambitos,ambito_ref}' IS DISTINCT FROM s.ambito_ref
 OR v_conjunto->>'unidad_ref' IS DISTINCT FROM s.unidad_ref
 OR v_conjunto->>'ambito_ref' IS DISTINCT FROM s.ambito_ref
 OR recurso#>>'{atributos,conjunto_ref}' IS DISTINCT FROM v_conjunto->>'conjunto_ref'
 OR recurso#>>'{atributos,conjunto_fuente_ref}' IS DISTINCT FROM v_conjunto->>'fuente_ref'
 OR recurso#>>'{atributos,conjunto_fuente_version}' IS DISTINCT FROM v_conjunto->>'fuente_version'
 OR recurso#>>'{atributos,conjunto_fuente_sha256}' IS DISTINCT FROM v_conjunto->>'fuente_sha256'
 OR recurso#>>'{atributos,solicitud_fuente_ref}' IS DISTINCT FROM s.ambito_fuente_ref
 OR recurso#>>'{atributos,solicitud_fuente_version}' IS DISTINCT FROM s.ambito_fuente_version::text
 OR recurso#>>'{atributos,solicitud_fuente_sha256}' IS DISTINCT FROM s.ambito_fuente_sha256
 THEN RAISE EXCEPTION 'B97: gestión RRHH no vigente para solicitud' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:inscripcion:convocatoria:'||s.convocatoria_ref||':'||s.categoria_ref,0));
 SELECT * INTO STRICT actual FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
  WHERE solicitud_ref=v_ref ORDER BY version DESC LIMIT 1 FOR SHARE;
 SELECT * INTO aprobada FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
  WHERE solicitud_ref=v_ref AND version=2 FOR SHARE;
 IF NOT FOUND OR aprobada.estado IS DISTINCT FROM 'admitida_a_convocatoria'
 OR aprobada.actor_ref IS NOT DISTINCT FROM s.persona_ref
 OR actual.version NOT IN (2,3)
 OR actual.estado NOT IN ('admitida_a_convocatoria','incorporada')
 THEN RAISE EXCEPTION 'B97: solicitud no admitida' USING ERRCODE='B9703'; END IF;

 -- La versión publicada original puede tener cerrado ya el plazo. BC9
 -- comprueba bytes, huella y categoría históricas sin reabrir inscripciones.
 BEGIN
  publicacion:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   s.convocatoria_ref,s.categoria_ref,s.version_sha256);
 EXCEPTION WHEN SQLSTATE 'B9601' OR SQLSTATE 'B9605' THEN
  RAISE EXCEPTION 'B97: publicación histórica no acreditada' USING ERRCODE='B9703';
 END;
 IF publicacion->>'convocatoria_ref' IS DISTINCT FROM s.convocatoria_ref
 OR publicacion->>'convocatoria_id' IS DISTINCT FROM s.convocatoria_id
 OR publicacion->>'secuencia' IS DISTINCT FROM s.secuencia::text
 OR publicacion->>'version_sha256' IS DISTINCT FROM s.version_sha256
 OR publicacion->>'categoria_ref' IS DISTINCT FROM s.categoria_ref
 OR publicacion->>'identificador_publico' IS DISTINCT FROM s.identificador_publico
 OR publicacion->>'bases_ref' IS DISTINCT FROM s.bases_ref
 OR publicacion->>'catalogo_ref' IS DISTINCT FROM s.catalogo_ref
 OR publicacion->>'catalogo_version' IS DISTINCT FROM s.catalogo_version::text
 OR publicacion->>'catalogo_sha256' IS DISTINCT FROM s.catalogo_sha256
 OR publicacion->>'politica_catalogo_ref' IS DISTINCT FROM s.politica_catalogo_ref
 OR publicacion->>'politica_catalogo_version' IS DISTINCT FROM s.politica_catalogo_version::text
 OR publicacion->>'politica_catalogo_sha256' IS DISTINCT FROM s.politica_catalogo_sha256
 OR publicacion->>'formulario_ref' IS DISTINCT FROM s.formulario_ref
 OR publicacion->>'formulario_version' IS DISTINCT FROM s.formulario_version::text
 OR publicacion->>'formulario_sha256' IS DISTINCT FROM s.formulario_sha256
 OR publicacion->>'requisitos_sha256' IS DISTINCT FROM s.requisitos_sha256
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(publicacion->'plazos_inscripcion') AS p(valor)
  WHERE p.valor->>'plazo_ref'=s.plazo_ref
   AND (p.valor->>'plazo_abre_en')::timestamptz=s.plazo_abre_en
   AND (p.valor->>'plazo_cierra_en')::timestamptz=s.plazo_cierra_en)
 THEN RAISE EXCEPTION 'B97: publicación histórica divergente' USING ERRCODE='B9703'; END IF;
 BEGIN
  politica:=vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256);
 EXCEPTION WHEN SQLSTATE 'B9601' THEN
  RAISE EXCEPTION 'B97: política de asociación no acreditada' USING ERRCODE='B9703';
 END;
 IF politica->>'permitida' IS DISTINCT FROM 'true'
 OR politica->>'politica_ref' IS DISTINCT FROM 'asociacion.manual.acta'
 OR politica->>'version' IS DISTINCT FROM s.politica_catalogo_version::text
 OR politica->>'sha256' IS DISTINCT FROM s.politica_catalogo_sha256
 OR politica->>'alcance_ref' IS DISTINCT FROM s.convocatoria_ref
 OR coalesce(politica->>'motivo_ref','')=''
 THEN RAISE EXCEPTION 'B97: política de asociación ausente' USING ERRCODE='B9703'; END IF;

 -- Acta seleccionada por RRHH: la categoría y vigencia se cotejan en B7.
 -- B79 confirma lote/original/B7/vínculos/recibo en una transacción y su
 -- recibo es inmutable: su ausencia con B7 íntegra acredita vía B7 original.
 SELECT * INTO acta FROM vec_bolsa_llamamientos.constitucion
  WHERE acta_ref=v_acta FOR SHARE;
 SELECT * INTO carga FROM vec_bolsa_llamamientos.recibo_carga_convoca
  WHERE acta_ref=v_acta FOR SHARE;
 IF acta.acta_ref IS NULL OR acta.categoria_ref IS DISTINCT FROM s.categoria_ref
 THEN RAISE EXCEPTION 'B97: acta no acreditada' USING ERRCODE='B9702'; END IF;
 -- El recibo B79 sólo existe para el circuito CONVOCA confirmado. B7 original
 -- conserva su acta sin recibo B79 y se acredita con canónicos y vínculos propios.
 IF carga.acta_ref IS NOT NULL AND (
  carga.categoria_ref IS DISTINCT FROM s.categoria_ref
  OR carga.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
  OR carga.recibo_constitucion->>'acta_ref' IS DISTINCT FROM acta.acta_ref
  OR carga.recibo_constitucion->>'bolsa_ref' IS DISTINCT FROM acta.bolsa_ref
  OR carga.recibo_constitucion->>'instantanea_ref' IS DISTINCT FROM acta.instantanea_ref
  OR carga.recibo_constitucion->>'version_bolsa' IS DISTINCT FROM acta.version_bolsa::text
  OR carga.recibo_constitucion->>'version_instantanea' IS DISTINCT FROM acta.version_instantanea::text)
 THEN RAISE EXCEPTION 'B97: recibo B79 divergente' USING ERRCODE='B9702'; END IF;
 SELECT * INTO bolsa FROM vec_bolsa_llamamientos.bolsa_constituida
  WHERE bolsa_ref=acta.bolsa_ref AND version=acta.version_bolsa
   AND huella_bolsa_sha256=acta.huella_bolsa_sha256 FOR SHARE;
 IF bolsa.bolsa_ref IS NULL OR bolsa.categoria_ref IS DISTINCT FROM s.categoria_ref
 OR bolsa.estado IS DISTINCT FROM 'vigente'
 OR consumo.consumida_en<bolsa.vigente_desde
 OR (bolsa.vigente_hasta IS NOT NULL AND consumo.consumida_en>=bolsa.vigente_hasta)
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa reemplazo
  WHERE reemplazo.bolsa_ref_sustituida=acta.bolsa_ref
   AND reemplazo.version_sustituida=acta.version_bolsa
   AND reemplazo.huella_sustituida_sha256=acta.huella_bolsa_sha256
   AND reemplazo.sustituida_en<=consumo.consumida_en)
 THEN RAISE EXCEPTION 'B97: bolsa sin vigencia' USING ERRCODE='B9702'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.instantanea_orden_bolsa i
  WHERE i.instantanea_ref=acta.instantanea_ref AND i.version=acta.version_instantanea
   AND i.huella_instantanea_sha256=acta.huella_instantanea_sha256
   AND i.bolsa_ref=acta.bolsa_ref AND i.version_bolsa=acta.version_bolsa
   AND i.huella_bolsa_sha256=acta.huella_bolsa_sha256)
 THEN RAISE EXCEPTION 'B97: instantánea ajena' USING ERRCODE='B9702'; END IF;

 ca:=vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(s.persona_ref);
 IF ca->>'estado' IS DISTINCT FROM 'acreditado'
 OR ca->>'persona_ref' IS DISTINCT FROM s.persona_ref
 OR coalesce(ca->>'candidato_ref','') !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(ca->>'persona_version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'procedencia_version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'procedencia_sha256','') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B97: vínculo de identidad pendiente' USING ERRCODE='B9701'; END IF;
 v_candidato:=ca->>'candidato_ref';
 inversa:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(v_candidato);
 IF inversa->>'estado' IS DISTINCT FROM 'acreditado'
 OR inversa#>>'{persona,ref}' IS DISTINCT FROM s.persona_ref
 OR inversa#>>'{persona,version}' IS DISTINCT FROM ca->>'persona_version'
 OR inversa#>>'{vinculo,ref}' IS DISTINCT FROM ca->>'vinculo_ref'
 OR inversa#>>'{vinculo,version}' IS DISTINCT FROM ca->>'version'
 OR inversa#>>'{vinculo,procedencia_ref}' IS DISTINCT FROM ca->>'procedencia_ref'
 OR inversa#>>'{vinculo,procedencia_version}' IS DISTINCT FROM ca->>'procedencia_version'
 OR inversa#>>'{vinculo,procedencia_sha256}' IS DISTINCT FROM ca->>'procedencia_sha256'
 OR inversa#>>'{vinculo,poblacion}' IS DISTINCT FROM ca->>'poblacion'
 THEN RAISE EXCEPTION 'B97: identidad y candidato divergentes' USING ERRCODE='B9701'; END IF;
 SELECT vc.participacion_ref,vc.acta_ref,vc.instantanea_ref,vc.version_instantanea,
  e.orden, c2.bolsa_ref
 INTO entrada
 FROM vec_bolsa_llamamientos.vinculo_candidato vc
 JOIN vec_bolsa_llamamientos.constitucion c2 ON c2.acta_ref=vc.acta_ref
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
  ON e.instantanea_ref=vc.instantanea_ref AND e.version_instantanea=vc.version_instantanea
  AND e.participacion_ref=vc.participacion_ref
 WHERE vc.candidato_ref=v_candidato AND vc.acta_ref=v_acta
  AND vc.instantanea_ref=acta.instantanea_ref
  AND vc.version_instantanea=acta.version_instantanea
  AND c2.bolsa_ref=acta.bolsa_ref;
 IF NOT FOUND OR entrada.participacion_ref IS NULL OR entrada.orden<1
 THEN RAISE EXCEPTION 'B97: participación ajena o ausente' USING ERRCODE='B9702'; END IF;

 IF carga.acta_ref IS NULL THEN
  -- La ruta B7 original nunca fabrica un recibo CONVOCA. La constitución,
  -- la bolsa y la instantánea son filas inmutables con FKs y SHA256 del canónico.
  SELECT * INTO instantanea FROM vec_bolsa_llamamientos.instantanea_orden_bolsa i
   WHERE i.instantanea_ref=acta.instantanea_ref AND i.version=acta.version_instantanea
    AND i.huella_instantanea_sha256=acta.huella_instantanea_sha256
    AND i.bolsa_ref=acta.bolsa_ref AND i.version_bolsa=acta.version_bolsa
    AND i.huella_bolsa_sha256=acta.huella_bolsa_sha256 FOR SHARE;
  IF instantanea.instantanea_ref IS NULL OR acta.confirmada_en>consumo.consumida_en
  THEN RAISE EXCEPTION 'B97: origen B7 ausente' USING ERRCODE='B9702';END IF;
  BEGIN
   bolsa_canon:=convert_from(bolsa.bolsa_canonica,'UTF8')::jsonb;
   instantanea_canon:=convert_from(instantanea.instantanea_canonica,'UTF8')::jsonb;
   IF jsonb_typeof(bolsa_canon) IS DISTINCT FROM 'object'
   OR jsonb_typeof(instantanea_canon) IS DISTINCT FROM 'object'
   OR jsonb_typeof(instantanea_canon->'entradas') IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'B97: canónico B7 inválido' USING ERRCODE='B9702';END IF;
   IF bolsa_canon->>'bolsa_ref' IS DISTINCT FROM bolsa.bolsa_ref
   OR bolsa_canon->>'version' IS DISTINCT FROM bolsa.version::text
   OR bolsa_canon->>'categoria_ref' IS DISTINCT FROM acta.categoria_ref
   OR coalesce(bolsa_canon->>'proceso_ref','')=''
   OR coalesce(bolsa_canon->>'listado_definitivo_ref','')=''
   OR coalesce(bolsa_canon->>'resolucion_constitucion_ref','')=''
   OR coalesce(bolsa_canon->>'version_listado','') !~ '^[1-9][0-9]*$'
   OR coalesce(bolsa_canon->>'huella_listado_sha256','') !~ '^[0-9a-f]{64}$'
   OR coalesce(bolsa_canon->>'huella_resolucion_sha256','') !~ '^[0-9a-f]{64}$'
   OR (bolsa_canon->>'vigente_desde')::timestamptz IS DISTINCT FROM bolsa.vigente_desde
   OR (bolsa_canon->>'vigente_hasta')::timestamptz IS DISTINCT FROM bolsa.vigente_hasta
   OR instantanea_canon->>'instantanea_ref' IS DISTINCT FROM instantanea.instantanea_ref
   OR instantanea_canon->>'version' IS DISTINCT FROM instantanea.version::text
   OR instantanea_canon->>'bolsa_ref' IS DISTINCT FROM bolsa.bolsa_ref
   OR instantanea_canon->>'version_bolsa' IS DISTINCT FROM bolsa.version::text
   OR instantanea_canon->>'huella_bolsa_sha256' IS DISTINCT FROM bolsa.huella_bolsa_sha256
   OR instantanea_canon->>'listado_definitivo_ref' IS DISTINCT FROM bolsa_canon->>'listado_definitivo_ref'
   OR instantanea_canon->>'version_listado' IS DISTINCT FROM bolsa_canon->>'version_listado'
   OR instantanea_canon->>'huella_listado_sha256' IS DISTINCT FROM bolsa_canon->>'huella_listado_sha256'
   OR coalesce(instantanea_canon->>'huella_contenido_sha256','') !~ '^[0-9a-f]{64}$'
   OR (instantanea_canon->>'referida_en')::timestamptz IS DISTINCT FROM instantanea.referida_en
   OR (instantanea_canon->>'generada_en')::timestamptz IS DISTINCT FROM instantanea.generada_en
   OR jsonb_array_length(instantanea_canon->'entradas')<>instantanea.total_participaciones
   OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(instantanea_canon->'entradas') x(valor)
      WHERE x.valor->>'orden'=entrada.orden::text
       AND x.valor#>>'{participacion,participacion_ref}'=entrada.participacion_ref
       AND x.valor#>>'{participacion,bolsa_ref}'=bolsa.bolsa_ref)
   THEN RAISE EXCEPTION 'B97: canónico B7 divergente' USING ERRCODE='B9702';END IF;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'B97: canónico B7 ilegible' USING ERRCODE='B9702';
  END;
 END IF;

 SELECT * INTO asociacion FROM vec_bolsa_llamamientos.inscripcion_acta_asociada
  WHERE convocatoria_ref=s.convocatoria_ref AND categoria_ref=s.categoria_ref FOR SHARE;
 IF FOUND THEN
  IF asociacion.acta_ref IS DISTINCT FROM v_acta
   OR asociacion.convocatoria_id IS DISTINCT FROM s.convocatoria_id
   OR asociacion.secuencia IS DISTINCT FROM s.secuencia
   OR asociacion.version_sha256 IS DISTINCT FROM s.version_sha256
   OR asociacion.categoria_ref IS DISTINCT FROM s.categoria_ref
   OR asociacion.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
   OR asociacion.instantanea_ref IS DISTINCT FROM acta.instantanea_ref
   OR asociacion.version_instantanea IS DISTINCT FROM acta.version_instantanea
   OR asociacion.politica_catalogo_ref IS DISTINCT FROM s.politica_catalogo_ref
   OR asociacion.politica_catalogo_version IS DISTINCT FROM s.politica_catalogo_version
   OR asociacion.politica_catalogo_sha256 IS DISTINCT FROM s.politica_catalogo_sha256
   OR asociacion.alcance_ref IS DISTINCT FROM s.convocatoria_ref
  THEN RAISE EXCEPTION 'B97: convocatoria asociada a otra acta' USING ERRCODE='B9702'; END IF;
 ELSIF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.inscripcion_acta_asociada x WHERE x.acta_ref=v_acta) THEN
  RAISE EXCEPTION 'B97: acta ya asociada a otra convocatoria' USING ERRCODE='B9702';
 END IF;

 etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
  ARRAY[s.categoria_ref],p_captura_actor->>'idioma');
 v_categoria:=etiquetas#>>'{0,categoria}';
 IF v_categoria IS NULL OR octet_length(v_categoria) NOT BETWEEN 1 AND 200
 THEN RAISE EXCEPTION 'B97: categoría no publicable' USING ERRCODE='B9703'; END IF;

 IF actual.version=3 THEN
  IF actual.actor_ref IS DISTINCT FROM v_actor
   OR actual.perfil_ref IS DISTINCT FROM p_captura_actor->>'perfil_ref'
   OR actual.cuenta_ref IS DISTINCT FROM p_captura_actor->>'cuenta_ref'
  THEN RAISE EXCEPTION 'B97: reintento de otra identidad' USING ERRCODE='42501'; END IF;
  SELECT * INTO previo FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo
   WHERE solicitud_ref=v_ref AND version=3;
  SELECT * INTO vinculo FROM vec_bolsa_llamamientos.inscripcion_incorporacion
   WHERE solicitud_ref=v_ref;
  IF previo.recibo_ref IS NULL OR vinculo.solicitud_ref IS NULL
  THEN RAISE EXCEPTION 'B97: recibo histórico incompleto' USING ERRCODE='55000'; END IF;
  IF previo.clave_sha256 IS DISTINCT FROM v_clave_sha
   OR previo.material_sha256 IS DISTINCT FROM v_material_sha
   OR previo.canal IS DISTINCT FROM 'interna_corporativa'
  THEN RAISE EXCEPTION 'B97: clave reutilizada con otro material' USING ERRCODE='B9603'; END IF;
  IF vinculo.acta_ref IS DISTINCT FROM v_acta
   OR vinculo.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
   OR vinculo.participacion_ref IS DISTINCT FROM entrada.participacion_ref
   OR vinculo.candidato_ref IS DISTINCT FROM v_candidato
   OR vinculo.orden_acta IS DISTINCT FROM entrada.orden
   OR vinculo.recibo_ref IS DISTINCT FROM previo.recibo_ref
  THEN RAISE EXCEPTION 'B97: reintento incompatible' USING ERRCODE='B9702'; END IF;
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
   acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
  VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
   v_ref,v_actor,'incorporar',consumo.decision_ref,consumo.auditoria_ref,'repetida',consumo.consumida_en);
  RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',previo.recibo_ref,
   'convocatoria_ref',s.convocatoria_ref,'categoria_ref',s.categoria_ref,'categoria',v_categoria,
   'bases_ref',s.bases_ref,'declaracion_ref',s.declaracion_ref,
   'estado','incorporada','version',3,'participacion_ref',vinculo.participacion_ref,
   'bolsa_ref',vinculo.bolsa_ref,'registrada_en',s.presentada_en,
   'decidida_en',vinculo.incorporada_en,'repetida',true,'auditoria_ref',consumo.auditoria_ref);
 END IF;
 IF actual.version IS DISTINCT FROM v_esperada OR actual.estado<>'admitida_a_convocatoria'
 THEN RAISE EXCEPTION 'B97: versión superada' USING ERRCODE='B9703'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.inscripcion_incorporacion i
    WHERE i.participacion_ref=entrada.participacion_ref)
 THEN RAISE EXCEPTION 'B97: participación ya incorporada' USING ERRCODE='B9702'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
    WHERE r.solicitud_ref=v_ref AND r.canal='interna_corporativa' AND r.clave_sha256=v_clave_sha)
 THEN RAISE EXCEPTION 'B97: clave ya utilizada' USING ERRCODE='B9603'; END IF;

 v_instante:=consumo.consumida_en;
 v_historia:='historia_inscripcion_'||encode(sha256(convert_to(v_ref||':3','UTF8')),'hex');
 v_evento:='evento_inscripcion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
 v_recibo_ref:='recibo_inscripcion_'||encode(sha256(convert_to(
  v_ref||chr(31)||'3'||chr(31)||v_material_sha,'UTF8')),'hex');
 -- Incorporaciones simultáneas: la segunda choca con una clave única tras la
 -- foto SERIALIZABLE; 40001 para reintentar y responder por la rama de repetición.
 BEGIN
 IF asociacion.convocatoria_ref IS NULL THEN
  INSERT INTO vec_bolsa_llamamientos.inscripcion_acta_asociada(
   convocatoria_ref,convocatoria_id,secuencia,version_sha256,categoria_ref,
   acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
   instantanea_ref,version_instantanea,huella_instantanea_sha256,
   solicitud_origen_ref,politica_ref,politica_catalogo_ref,politica_catalogo_version,
   politica_catalogo_sha256,motivo_ref,alcance_ref,actor_ref,decision_ref,auditoria_ref,asociada_en)
  VALUES(s.convocatoria_ref,s.convocatoria_id,s.secuencia,s.version_sha256,s.categoria_ref,
   v_acta,acta.bolsa_ref,acta.version_bolsa,acta.huella_bolsa_sha256,
   acta.instantanea_ref,acta.version_instantanea,acta.huella_instantanea_sha256,
   v_ref,'asociacion.manual.acta',s.politica_catalogo_ref,s.politica_catalogo_version,
   s.politica_catalogo_sha256,politica->>'motivo_ref',politica->>'alcance_ref',
   v_actor,consumo.decision_ref,consumo.auditoria_ref,v_instante);
 END IF;
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(v_ref,3,'incorporada',v_actor,p_captura_actor->>'perfil_ref',
  p_captura_actor->>'cuenta_ref',aprobada.evaluacion,consumo.decision_ref,
  consumo.consumo_huella_sha256,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
  historia_ref,solicitud_ref,version,accion,actor_ref,estado,registrada_en)
 VALUES(v_historia,v_ref,3,'incorporar',v_actor,'incorporada',v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
  evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_ref,3,'vec.bolsa.inscripcion.incorporada.v1',
  jsonb_build_object('solicitud_ref',v_ref,'convocatoria_ref',s.convocatoria_ref,
   'acta_ref',v_acta,'bolsa_ref',acta.bolsa_ref,
   'participacion_ref',entrada.participacion_ref,'version',3,'estado','incorporada'),v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
  recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
  historia_ref,evento_ref,auditoria_ref,emitida_en)
 VALUES(v_recibo_ref,v_ref,3,s.persona_ref,'interna_corporativa',v_clave_sha,v_material_sha,
  v_historia,v_evento,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.inscripcion_incorporacion(
  solicitud_ref,version,convocatoria_ref,categoria_ref,acta_ref,bolsa_ref,instantanea_ref,version_instantanea,
  participacion_ref,candidato_ref,orden_acta,persona_ref,persona_version,
  vinculo_ref,vinculo_version,procedencia_ref,procedencia_version,procedencia_sha256,
  decision_ref,auditoria_ref,recibo_ref,incorporada_en)
 VALUES(v_ref,3,s.convocatoria_ref,s.categoria_ref,v_acta,acta.bolsa_ref,acta.instantanea_ref,acta.version_instantanea,
  entrada.participacion_ref,v_candidato,entrada.orden,s.persona_ref,
  (ca->>'persona_version')::bigint,ca->>'vinculo_ref',(ca->>'version')::bigint,
  ca->>'procedencia_ref',(ca->>'procedencia_version')::bigint,ca->>'procedencia_sha256',
  consumo.decision_ref,consumo.auditoria_ref,v_recibo_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
  acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
 VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
  v_ref,v_actor,'incorporar',consumo.decision_ref,consumo.auditoria_ref,'confirmada',v_instante);
 EXCEPTION WHEN unique_violation THEN
  RAISE EXCEPTION 'B97: incorporación concurrente, reintentar' USING ERRCODE='40001';
 END;
 RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',v_recibo_ref,
  'convocatoria_ref',s.convocatoria_ref,'categoria_ref',s.categoria_ref,'categoria',v_categoria,
  'bases_ref',s.bases_ref,'declaracion_ref',s.declaracion_ref,
  'estado','incorporada','version',3,'participacion_ref',entrada.participacion_ref,
  'bolsa_ref',acta.bolsa_ref,'registrada_en',s.presentada_en,
  'decidida_en',v_instante,'repetida',false,'auditoria_ref',consumo.auditoria_ref);
END $function$;
COMMIT;
