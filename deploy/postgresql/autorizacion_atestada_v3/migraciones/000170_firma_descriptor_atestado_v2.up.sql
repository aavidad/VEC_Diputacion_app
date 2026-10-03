\set ON_ERROR_STOP on
-- AD170 liga el descriptor nominal exterior y el material PDF a V3.
-- Requiere AD162 instalada. Conserva núcleo, audiencias, perfiles y ACL previos.
-- Reserva AD170 registrada antes de este archivo. Sólo UP; sin principal.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000170',0));
DO $pre$
DECLARE nombre text; f oid;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR getdatabaseencoding()<>'UTF8'
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD170 preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=to_regprocedure(nombre);
  IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
   RAISE EXCEPTION 'AD170 dependencia no disponible: %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(
 p_solicitud text,p_descriptor bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; descriptor jsonb; original json; nodo jsonb; original_nodo json;
 c jsonb; d jsonb; x record; bloque record;
 material_h text; descriptor_h text; contexto_h text; recurso text; accion text;
 audiencia text; tipo_recurso text; perfil text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off' OR pg_is_in_recovery() THEN
  RAISE EXCEPTION 'AD170 transacción no admitida' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
  OR p_descriptor IS NULL OR octet_length(p_descriptor) NOT BETWEEN 512 AND 32768
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD170 material inválido' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; original:=convert_from(p_descriptor,'UTF8')::json;
 descriptor:=original::jsonb;
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
  OR jsonb_typeof(d) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM json_each(p_solicitud::json))<>(SELECT count(*) FROM jsonb_each(s))
  OR s->>'Via' IS NULL OR s->>'Via' NOT IN('certificado_vec','portafirmas_registro_rrhh')
  OR s->>'PoliticaVerificacion' IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
  OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
  OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
  OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
  OR jsonb_typeof(s->'FirmantePrincipalRef') IS DISTINCT FROM 'string'
  OR s->>'FirmantePrincipalRef' !~ '^per_[A-Za-z0-9_-]{2,159}$' THEN
  RAISE EXCEPTION 'AD170 material de firma inválido' USING ERRCODE='22023'; END IF;
 -- Gramática exterior cerrada. Se conserva la representación recibida para
 -- la huella; el objeto JSONB sólo permite comprobar forma y duplicados.
 FOR bloque IN SELECT * FROM (VALUES
  (ARRAY[]::text[],ARRAY['esquema','certificado_der_sha256','seleccion','recurso','accion','finalidad','motivo','circuito','paso_ref','paso_orden','fecha_historica'],false),
  (ARRAY['seleccion'],ARRAY['perfil_esperado_ref','perfil_activo_ref','rol_id','cargo_ref','enlace_ejercicio_ref'],false),
  (ARRAY['recurso'],ARRAY['organizacion_ref','unidad_ref','expediente_ref','documento_ref','recurso_autorizable_ref','modulo_id','tipo_recurso','recurso_contexto_sha256','original','pdf_raiz_sha256','firmado','pdf_firmado_sha256','numero_firmas','entrada_revision'],false),
  (ARRAY['motivo'],ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'],false),
  (ARRAY['circuito'],ARRAY['referencia','version','huella_sha256'],false),
  (ARRAY['recurso','original'],ARRAY['referencia','version','huella_sha256'],false),
  (ARRAY['recurso','firmado'],ARRAY['referencia','version','huella_sha256'],false),
  (ARRAY['recurso','entrada_revision'],ARRAY['referencia','version','huella_sha256'],true)
 ) v(ruta,claves,opcional) LOOP
  nodo:=descriptor#>bloque.ruta; original_nodo:=original#>bloque.ruta;
  IF bloque.opcional AND nodo='null'::jsonb THEN CONTINUE; END IF;
  IF jsonb_typeof(nodo) IS DISTINCT FROM 'object'
   OR (SELECT count(*) FROM jsonb_object_keys(nodo))<>cardinality(bloque.claves)
   OR (nodo ?& bloque.claves) IS NOT TRUE
   OR (SELECT count(*) FROM json_each(original_nodo))<>(SELECT count(*) FROM jsonb_each(nodo)) THEN
   RAISE EXCEPTION 'AD170 descriptor inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 -- Los campos hoja son escalares tipados; no queda otro objeto donde
 -- esconder claves duplicadas o material ajeno al DTO exterior.
 FOR bloque IN SELECT * FROM (VALUES
  (ARRAY['esquema'],'string'),(ARRAY['certificado_der_sha256'],'string'),
  (ARRAY['accion'],'string'),(ARRAY['finalidad'],'string'),(ARRAY['paso_ref'],'string'),(ARRAY['paso_orden'],'number'),
  (ARRAY['seleccion','perfil_esperado_ref'],'string'),(ARRAY['seleccion','perfil_activo_ref'],'string'),
  (ARRAY['seleccion','rol_id'],'string'),(ARRAY['seleccion','cargo_ref'],'string'),(ARRAY['seleccion','enlace_ejercicio_ref'],'string'),
  (ARRAY['recurso','organizacion_ref'],'string'),(ARRAY['recurso','unidad_ref'],'string'),(ARRAY['recurso','expediente_ref'],'string'),
  (ARRAY['recurso','documento_ref'],'string'),(ARRAY['recurso','recurso_autorizable_ref'],'string'),
  (ARRAY['recurso','modulo_id'],'string'),(ARRAY['recurso','tipo_recurso'],'string'),(ARRAY['recurso','recurso_contexto_sha256'],'string'),
  (ARRAY['recurso','pdf_raiz_sha256'],'string'),(ARRAY['recurso','pdf_firmado_sha256'],'string'),(ARRAY['recurso','numero_firmas'],'number'),
  (ARRAY['motivo','catalogo_id'],'string'),(ARRAY['motivo','catalogo_version'],'number'),
  (ARRAY['motivo','catalogo_huella_sha256'],'string'),(ARRAY['motivo','entrada_clave'],'string'),
  (ARRAY['circuito','referencia'],'string'),(ARRAY['circuito','version'],'number'),(ARRAY['circuito','huella_sha256'],'string'),
  (ARRAY['recurso','original','referencia'],'string'),(ARRAY['recurso','original','version'],'number'),(ARRAY['recurso','original','huella_sha256'],'string'),
  (ARRAY['recurso','firmado','referencia'],'string'),(ARRAY['recurso','firmado','version'],'number'),(ARRAY['recurso','firmado','huella_sha256'],'string')
 ) v(ruta,tipo) LOOP
  nodo:=descriptor#>bloque.ruta;
  IF jsonb_typeof(nodo) IS DISTINCT FROM bloque.tipo
   OR (bloque.tipo='string' AND (octet_length(nodo#>>'{}') NOT BETWEEN 1 AND 512
    OR (nodo#>>'{}') COLLATE "C" !~ '^[!-~]+$' OR strpos(nodo#>>'{}','*')<>0))
   OR (bloque.tipo='number' AND ((nodo#>>'{}') !~ '^[1-9][0-9]{0,15}$'
    OR (nodo#>>'{}')::numeric>9007199254740991)) THEN
   RAISE EXCEPTION 'AD170 descriptor inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF descriptor#>'{recurso,entrada_revision}' IS DISTINCT FROM 'null'::jsonb THEN
  nodo:=descriptor#>'{recurso,entrada_revision}';
  IF jsonb_typeof(nodo->'referencia') IS DISTINCT FROM 'string'
   OR jsonb_typeof(nodo->'huella_sha256') IS DISTINCT FROM 'string'
   OR jsonb_typeof(nodo->'version') IS DISTINCT FROM 'number'
   OR octet_length(nodo->>'referencia') NOT BETWEEN 1 AND 512
   OR (nodo->>'referencia') COLLATE "C" !~ '^[!-~]+$' OR strpos(nodo->>'referencia','*')<>0
   OR (nodo->>'huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
   OR (nodo->>'version') !~ '^[1-9][0-9]{0,15}$'
   OR (nodo->>'version')::numeric>9007199254740991 THEN
   RAISE EXCEPTION 'AD170 descriptor inválido' USING ERRCODE='22023'; END IF;
 END IF;
 IF descriptor->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.constructor-ct.v1'
  OR descriptor->'fecha_historica' IS DISTINCT FROM 'null'::jsonb
  OR descriptor#>>'{recurso,organizacion_ref}' IS DISTINCT FROM s->>'OrganizacionRef'
  OR descriptor->>'certificado_der_sha256' IS DISTINCT FROM s->>'CertificadoHuella' THEN
  RAISE EXCEPTION 'AD170 descriptor inválido' USING ERRCODE='22023'; END IF;
 -- La competencia la revalidan AUT35/AUT32. Ningún selector JSON la concede.
 material_h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 descriptor_h:=encode(sha256(p_descriptor),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"descriptor_firma_sha256":"'||descriptor_h||'","material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF s->>'Via'='certificado_vec' THEN
  accion:='contratacion_temporal.documento.firma_vec.registrar'; audiencia:='vec_contratacion_temporal.firma_vec.v2';
  tipo_recurso:='firma_vec_documento_contratacion_temporal'; perfil:='firma_vec_documento_ct_v2';
  recurso:='operacion-firma-vec-ct:'||(s->>'ClaveIdempotencia');
 ELSE
  accion:='contratacion_temporal.documento.firma_externa.registrar'; audiencia:='vec_contratacion_temporal.firma_externa.v2';
  tipo_recurso:='firma_externa_documento_contratacion_temporal'; perfil:='firma_externa_documento_ct_v2';
  recurso:='operacion-firma-externa-ct:'||(s->>'ClaveIdempotencia');
 END IF;
 IF c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR d->>'accion' IS DISTINCT FROM accion OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR d->>'tipo_recurso' IS DISTINCT FROM tipo_recurso OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
  OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
  OR d->>'principal_id' IS NULL OR d->>'principal_id' !~ '^per_[A-Za-z0-9_-]{2,159}$'
  OR (s->>'Via'='certificado_vec' AND d->>'principal_id' IS DISTINCT FROM s->>'FirmantePrincipalRef')
  OR (s->>'Via'='portafirmas_registro_rrhh' AND (d->>'principal_id' IS NOT DISTINCT FROM s->>'FirmantePrincipalRef'
   OR d->>'version_rol_ref' IS DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1'))
  OR d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoOperadorRef'
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD170 firma descriptor divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD170 firma requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'AD170 material inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 x record; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>propietario LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
END $acl$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
COMMIT;
