\set ON_ERROR_STOP on
-- AUT35: constructor de bytes V1 desde autoridades nominales. Sin escritura de evidencia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000035',0));

DO $preimagen$
BEGIN
 IF current_user <> 'vec_autorizacion_propietario'
  OR to_regprocedure('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)') IS NOT NULL
  OR to_regprocedure('vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.texto_json_go_v3(text)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)') IS NULL
  OR to_regprocedure('vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)') IS NULL
  OR to_regprocedure('vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)') IS NULL
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)','EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'aut35_preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

CREATE FUNCTION vec_autorizacion.canon_texto_json_go_ct_v1(v text)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SECURITY INVOKER
SET search_path=pg_catalog AS $f$
 SELECT replace(replace(vec_autorizacion.texto_json_go_v3(v),chr(8232),'\u2028'),chr(8233),'\u2029')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_texto_json_go_ct_v1(text) FROM PUBLIC;

-- Orden cerrado de encoding/json sobre los structs de CanonCompetenciaFirmanteHistoricaV1.
-- jsonb sólo aporta campos tipados; jamás se convierte su representación textual en canon.
CREATE FUNCTION vec_autorizacion.canon_json_competencia_firmante_ct_v1(v jsonb, tipo text)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SECURITY INVOKER
SET search_path=pg_catalog AS $f$
DECLARE campos text[]; tipos text[]; i integer; salida text := '{'; valor text; marca timestamptz;
BEGIN
 CASE tipo
 WHEN 'raiz' THEN
  campos:=ARRAY['esquema','identidad','competencia','personal','recurso','relacion_ct','accion','finalidad','motivo','circuito','paso_ref','paso_orden','fecha_historica'];
  tipos:=ARRAY['s','identidad','competencia','personal','recurso','relacion','s','s','motivo','ref','s','n','t'];
 WHEN 'identidad' THEN
  campos:=ARRAY['certificado_der_sha256','persona_ref','persona','cuenta','vinculo_cuenta_persona','cuenta_persona_cuenta_ref','cuenta_persona_persona_ref','vinculo_certificado','vinculo_cuenta_ref','vinculo_persona_ref','vinculo_der_sha256'];
  tipos:=ARRAY['s','s','ref','ref','ref','s','s','ref','s','s','s'];
 WHEN 'competencia' THEN
  campos:=ARRAY['asignacion','rol','rol_id','control_rol','persona_ref','perfil_esperado_ref','perfil_activo_ref','modulo_id','tipo_recurso','recurso_ref','ambito_organizacion_ref','ambito_unidad_ref','asignacion_rol_ref','control_rol_ref','vigente_desde','vigente_hasta'];
  tipos:=ARRAY['ref','ref','s','ref','s','s','s','s','s','s','s','s','s','s','t','t'];
 WHEN 'personal' THEN
  campos:=ARRAY['cargo','enlace_ocupante','ocupante_persona_ref','cargo_ref_enlace','cargo_vigente_desde','cargo_vigente_hasta','enlace_vigente_desde','enlace_vigente_hasta','delegacion'];
  tipos:=ARRAY['ref','ref','s','s','t','t','t','t','delegacion_opcional'];
 WHEN 'delegacion' THEN
  campos:=ARRAY['acto','delegante_persona_ref','delegado_persona_ref','cargo_ref','vigente_desde','vigente_hasta'];
  tipos:=ARRAY['ref','s','s','s','t','t'];
 WHEN 'recurso' THEN
  campos:=ARRAY['organizacion_ref','unidad_ref','expediente_ref','documento_ref','recurso_autorizable_ref','modulo_id','tipo_recurso','recurso_contexto_sha256','original','pdf_raiz_sha256','firmado','pdf_firmado_sha256','numero_firmas','entrada_revision'];
  tipos:=ARRAY['s','s','s','s','s','s','s','s','ref','s','ref','s','n','revision_opcional'];
 WHEN 'relacion' THEN
  campos:=ARRAY['expediente_ref','unidad_ref','origen_ref','origen_version','prueba_snapshot_sha256','evento_ref','evento_huella_sha256','confirmada_en'];
  tipos:=ARRAY['s','s','s','n','s','s','s','t'];
 WHEN 'motivo' THEN
  IF ((v->>'catalogo_id') COLLATE "C" ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
   OR ((v->>'entrada_clave') COLLATE "C" ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
   OR (v->>'catalogo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  campos:=ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'];
  tipos:=ARRAY['s','n','s','s'];
 WHEN 'ref' THEN
  IF (v->>'huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  campos:=ARRAY['referencia','version','huella_sha256']; tipos:=ARRAY['s','n','s'];
 WHEN 'delegacion_opcional' THEN
  IF jsonb_typeof(v)='null' THEN RETURN 'null'; END IF;
  RETURN vec_autorizacion.canon_json_competencia_firmante_ct_v1(v,'delegacion');
 WHEN 'revision_opcional' THEN
  IF jsonb_typeof(v)='null' THEN RETURN 'null'; END IF;
  RETURN vec_autorizacion.canon_json_competencia_firmante_ct_v1(v,'ref');
 WHEN 's' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'string'
   OR octet_length(v#>>'{}') NOT BETWEEN 1 AND 512
   OR (v#>>'{}') COLLATE "C" !~ '^[!-~]+$'
   OR strpos(v#>>'{}','*') <> 0
  THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  -- Go encoding/json escapa HTML y los separadores de línea JS.
  RETURN vec_autorizacion.canon_texto_json_go_ct_v1(v#>>'{}');
 WHEN 'n' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR (v#>>'{}') !~ '^[1-9][0-9]{0,15}$'
   OR (v#>>'{}')::numeric > 9007199254740991
  THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  RETURN v#>>'{}';
 WHEN 't' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'string' THEN
   RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  marca:=(v#>>'{}')::timestamptz;
  IF NOT isfinite(marca) OR extract(year FROM marca AT TIME ZONE 'UTC') NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
  valor:=to_char(marca AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS');
  IF extract(microseconds FROM marca)::integer % 1000000 <> 0 THEN
   valor:=valor || '.' || rtrim(to_char(marca AT TIME ZONE 'UTC','US'),'0');
  END IF;
  RETURN vec_autorizacion.texto_json_go_v3(valor || 'Z');
 ELSE RAISE EXCEPTION 'aut35_tipo_invalido' USING ERRCODE='42501';
 END CASE;
 IF jsonb_typeof(v) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(v)) <> cardinality(campos)
  OR (v ?& campos) IS NOT TRUE
 THEN RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
 FOR i IN 1..cardinality(campos) LOOP
  IF i>1 THEN salida:=salida||','; END IF;
  salida:=salida||to_json(campos[i])::text||':'||
   vec_autorizacion.canon_json_competencia_firmante_ct_v1(v->campos[i],tipos[i]);
 END LOOP;
 RETURN salida||'}';
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
 RAISE EXCEPTION 'aut35_fecha_invalida' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_json_competencia_firmante_ct_v1(jsonb,text) FROM PUBLIC;

-- El descriptor CT sólo selecciona material gobernado; no transporta identidad,
-- cargo, rol ni asignación como hechos acreditados. Esquema y claves cerrados.
CREATE FUNCTION vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(
 descriptor_ct jsonb, relacion_ct jsonb, consumo_v3 jsonb
) RETURNS bytea
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET TimeZone='UTC'
AS $f$
DECLARE sel jsonb; rec jsonb; ca jsonb; per jsonb; c jsonb; identidad jsonb;
 competencia jsonb; personal jsonb; relacion jsonb; canon text; bytes bytea;
 asignacion record; rol record; fecha timestamptz(6); now_utc timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
  OR current_setting('transaction_read_only') <> 'off'
  OR pg_is_in_recovery() OR current_setting('TimeZone') <> 'UTC'
 THEN RAISE EXCEPTION 'aut35_transaccion_no_admitida' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(descriptor_ct) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(descriptor_ct)) <> 11
  OR (descriptor_ct ?& ARRAY['esquema','certificado_der_sha256','seleccion',
     'recurso','accion','finalidad','motivo','circuito','paso_ref',
     'paso_orden','fecha_historica']) IS NOT TRUE
  OR descriptor_ct->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.constructor-ct.v1'
  OR (descriptor_ct->>'certificado_der_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR jsonb_typeof(descriptor_ct->'seleccion') IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(descriptor_ct->'seleccion')) <> 5
  OR (descriptor_ct->'seleccion' ?& ARRAY['perfil_esperado_ref','perfil_activo_ref',
     'rol_id','cargo_ref','enlace_ejercicio_ref']) IS NOT TRUE
  OR jsonb_typeof(descriptor_ct->'recurso') IS DISTINCT FROM 'object'
  OR jsonb_typeof(relacion_ct) IS DISTINCT FROM 'object'
  OR jsonb_typeof(consumo_v3) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'aut35_descriptor_no_admitido' USING ERRCODE='42501'; END IF;
 sel:=descriptor_ct->'seleccion'; rec:=descriptor_ct->'recurso';
 -- La misma gramática tipada que la salida evita coerciones, campos opacos y
 -- cualquier serialización JSONB textual antes de leer las fuentes.
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(rec,'recurso');
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(descriptor_ct->'motivo','motivo');
 PERFORM vec_autorizacion.canon_json_competencia_firmante_ct_v1(descriptor_ct->'circuito','ref');
 IF vec_autorizacion.texto_positivo_valido(sel->>'perfil_esperado_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'perfil_activo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'rol_id',128) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'cargo_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(sel->>'enlace_ejercicio_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'accion',256) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'finalidad',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(descriptor_ct->>'paso_ref',512) IS NOT TRUE
  OR vec_autorizacion.instante_utc_microsegundo_valido(descriptor_ct->>'fecha_historica') IS NOT TRUE
  OR rec->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'documento_ref'
  OR rec->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR rec->>'organizacion_ref' IS DISTINCT FROM relacion_ct->>'organizacion_ref'
  OR rec->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_ref'
  OR rec->>'expediente_ref' IS DISTINCT FROM relacion_ct->>'expediente_ref'
 THEN RAISE EXCEPTION 'aut35_descriptor_no_admitido' USING ERRCODE='42501'; END IF;
 fecha:=(descriptor_ct->>'fecha_historica')::timestamptz;
 now_utc:=clock_timestamp();
 IF fecha > now_utc OR relacion_ct->>'esquema' IS DISTINCT FROM
    'vec.contratacion-temporal.relacion-unidad-expediente.v1'
  OR relacion_ct->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_ref_esperada'
  OR relacion_ct->>'unidad_ref' IS DISTINCT FROM relacion_ct->>'unidad_snapshot_solicitado_ref'
  OR relacion_ct->>'tipo_evento_origen' IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR (relacion_ct->>'asignacion_confirmada_en')::timestamptz > fecha
 THEN RAISE EXCEPTION 'aut35_relacion_no_admitida' USING ERRCODE='42501'; END IF;
 -- AD167 acredita antes de cualquier lectura nominal que consumo y auditoría
 -- fueron insertados en esta misma transacción para el efecto CT.
 PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(consumo_v3);
 -- CA25 devuelve fuentes revalidadas y bloqueadas, incluyendo la organización
 -- acreditada. Ningún campo de identidad del descriptor reemplaza a CA25.
 ca:=vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(
   descriptor_ct->>'certificado_der_sha256');
 IF ca->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-ct.v2'
  OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM descriptor_ct->>'certificado_der_sha256'
  OR ca#>>'{organizacion_destino,organizacion_ref}' IS DISTINCT FROM rec->>'organizacion_ref'
 THEN RAISE EXCEPTION 'aut35_identidad_no_admitida' USING ERRCODE='42501'; END IF;
 identidad:=jsonb_build_object(
  'certificado_der_sha256',ca->>'certificado_der_sha256',
  'persona_ref',ca->>'persona_ref',
  'persona',ca->'persona',
  'cuenta',ca->'cuenta',
  'vinculo_cuenta_persona',jsonb_build_object(
   'referencia',ca#>>'{vinculo_cuenta_persona,referencia}',
   'version',ca#>'{vinculo_cuenta_persona,version}',
   'huella_sha256',ca#>>'{vinculo_cuenta_persona,huella_sha256}'),
  'cuenta_persona_cuenta_ref',ca#>>'{vinculo_cuenta_persona,cuenta_ref}',
  'cuenta_persona_persona_ref',ca#>>'{vinculo_cuenta_persona,persona_ref}',
  'vinculo_certificado',jsonb_build_object(
   'referencia',ca#>>'{vinculo_certificado,referencia}',
   'version',ca#>'{vinculo_certificado,version}',
   'huella_sha256',ca#>>'{vinculo_certificado,huella_sha256}'),
  'vinculo_cuenta_ref',ca#>>'{vinculo_certificado,cuenta_ref}',
  'vinculo_persona_ref',ca#>>'{vinculo_certificado,persona_ref}',
  'vinculo_der_sha256',ca#>>'{vinculo_certificado,certificado_der_sha256}');
 -- Personal29 resuelve sólo el selector explícito y contrasta el ejercicio
 -- nominal; su fachada bloquea la fuente propietaria hasta COMMIT.
 per:=vec_personal.resolver_fuente_cargo_ocupante_ct_v1(
   sel->>'cargo_ref',sel->>'enlace_ejercicio_ref',ca->>'persona_ref',
   rec->>'organizacion_ref',rec->>'unidad_ref');
 IF per->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR per->>'persona_ejerciente_ref' IS DISTINCT FROM ca->>'persona_ref'
  OR per->>'organizacion_ref' IS DISTINCT FROM rec->>'organizacion_ref'
  OR per->>'unidad_ref' IS DISTINCT FROM rec->>'unidad_ref'
  OR per#>>'{cargo,referencia}' IS DISTINCT FROM sel->>'cargo_ref'
  OR per#>>'{enlace_ejerciente,referencia}' IS DISTINCT FROM sel->>'enlace_ejercicio_ref'
  OR per->>'accion_ref' IS DISTINCT FROM descriptor_ct->>'accion'
  OR per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'recurso_autorizable_ref'
  OR per->>'finalidad_ref' IS DISTINCT FROM descriptor_ct->>'finalidad'
 THEN RAISE EXCEPTION 'aut35_personal_no_admitido' USING ERRCODE='42501'; END IF;
 personal:=jsonb_build_object('cargo',per->'cargo','enlace_ocupante',per->'enlace_ocupante',
  'ocupante_persona_ref',per->>'ocupante_persona_ref',
  'cargo_ref_enlace',per->>'cargo_ref_enlace',
  'cargo_vigente_desde',per->'cargo_vigente_desde',
  'cargo_vigente_hasta',per->'cargo_vigente_hasta',
  'enlace_vigente_desde',per->'enlace_vigente_desde',
  'enlace_vigente_hasta',per->'enlace_vigente_hasta',
  'delegacion',per->'delegacion');
 -- La asignación y el rol se eligen por perfil del plan y se leen de la
 -- autoridad AUT. No se deriva un rol de cargo, título o certificado.
 SELECT a.asignacion_ref,a.version,a.huella_sha256,a.principal_id,a.perfil_activo_ref,
   a.version_rol_ref,a.documento
 INTO STRICT asignacion FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 WHERE p.perfil_activo_ref=sel->>'perfil_activo_ref' FOR SHARE OF p,a;
 SELECT r.version_rol_ref,r.version,r.rol_id,r.huella_sha256,r.documento,
   v.revision,v.estado,v.huella_sha256 AS control_huella
 INTO STRICT rol FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.control_vigencia_version_rol_actual x ON x.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=x.version_rol_ref AND v.revision=x.revision
 WHERE r.version_rol_ref=asignacion.version_rol_ref FOR SHARE OF r,x,v;
 IF asignacion.principal_id IS DISTINCT FROM ca->>'persona_ref'
  OR asignacion.perfil_activo_ref IS DISTINCT FROM sel->>'perfil_activo_ref'
  OR rol.rol_id IS DISTINCT FROM sel->>'rol_id'
  OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
  OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
  OR rol.estado IS DISTINCT FROM 'habilitada'
 THEN RAISE EXCEPTION 'aut35_competencia_no_admitida' USING ERRCODE='42501'; END IF;
 competencia:=jsonb_build_object(
  'asignacion',jsonb_build_object('referencia',asignacion.asignacion_ref,
   'version',asignacion.version,'huella_sha256',asignacion.huella_sha256),
  'rol',jsonb_build_object('referencia',rol.version_rol_ref,'version',rol.version,
   'huella_sha256',rol.huella_sha256),
  'rol_id',rol.rol_id,
  'control_rol',jsonb_build_object('referencia',rol.version_rol_ref,
   'version',rol.revision,'huella_sha256',rol.control_huella),
  'persona_ref',asignacion.principal_id,
  'perfil_esperado_ref',sel->>'perfil_esperado_ref',
  'perfil_activo_ref',asignacion.perfil_activo_ref,
  'modulo_id',rec->>'modulo_id','tipo_recurso',rec->>'tipo_recurso',
  'recurso_ref',rec->>'recurso_autorizable_ref',
  'ambito_organizacion_ref',rec->>'organizacion_ref',
  'ambito_unidad_ref',rec->>'unidad_ref',
  'asignacion_rol_ref',asignacion.version_rol_ref,
  'control_rol_ref',rol.version_rol_ref,
  'vigente_desde',asignacion.documento->'vigente_desde',
  'vigente_hasta',asignacion.documento->'vigente_hasta');
 relacion:=jsonb_build_object('expediente_ref',relacion_ct->>'expediente_ref',
  'unidad_ref',relacion_ct->>'unidad_ref',
  'origen_ref',relacion_ct->>'operacion_origen_ref',
  'origen_version',relacion_ct->'version_origen_vinculo',
  'prueba_snapshot_sha256',relacion_ct->>'prueba_snapshot_origen_huella_sha256',
  'evento_ref',relacion_ct->>'evento_asignacion_ref',
  'evento_huella_sha256',relacion_ct->>'evento_payload_huella_sha256',
  'confirmada_en',relacion_ct->'asignacion_confirmada_en');
 c:=jsonb_build_object('esquema','vec.competencia-firmante.historica.v1',
  'identidad',identidad,'competencia',competencia,'personal',personal,
  'recurso',rec,'relacion_ct',relacion,'accion',descriptor_ct->'accion',
  'finalidad',descriptor_ct->'finalidad','motivo',descriptor_ct->'motivo',
  'circuito',descriptor_ct->'circuito','paso_ref',descriptor_ct->'paso_ref',
  'paso_orden',descriptor_ct->'paso_orden',
  'fecha_historica',descriptor_ct->'fecha_historica');
 canon:=vec_autorizacion.canon_json_competencia_firmante_ct_v1(c,'raiz');
 bytes:=convert_to(canon,'UTF8');
 IF octet_length(bytes) NOT BETWEEN 512 AND 32768 THEN
  RAISE EXCEPTION 'aut35_canon_invalido' USING ERRCODE='42501'; END IF;
 -- AUT32 repite las lecturas y comprueba consumo V3, ACL, política y vigencia
 -- en esta transacción. Construir no inserta evidencia ni sustituye acreditar.
 PERFORM vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   bytes,relacion_ct,consumo_v3);
 RETURN bytes;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'aut35_fuente_no_disponible' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)
 TO vec_contratacion_temporal_propietario;

DO $acl$
DECLARE f regprocedure; x record; permitido oid;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.canon_texto_json_go_ct_v1(text)'::regprocedure,
  'vec_autorizacion.canon_json_competencia_firmante_ct_v1(jsonb,text)'::regprocedure,
  'vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure
 ] LOOP
  permitido:=CASE WHEN f='vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure
   THEN 'vec_contratacion_temporal_propietario'::regrole::oid
   ELSE 'vec_autorizacion_propietario'::regrole::oid END;
  FOR x IN SELECT DISTINCT acl.grantee FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
   WHERE p.oid=f AND acl.grantee NOT IN ('vec_autorizacion_propietario'::regrole,permitido)
  LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
   WHERE p.oid=f AND acl.grantee NOT IN ('vec_autorizacion_propietario'::regrole,permitido))
  THEN RAISE EXCEPTION 'aut35_acl_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;

COMMIT;
