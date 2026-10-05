\set ON_ERROR_STOP on
-- AD184. Fachadas nominales propias PUBLICAR/LEER; no provisiona permisos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000184',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion.validar_administrador_denominacion_persona_v1(jsonb,jsonb)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'AD184: PARO clave=operador_PG18 actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000'; END IF;
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=to_regrole('vec_autorizacion_propietario') AND p.prosecdef AND p.prorettype='boolean'::regtype)
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD184: PARO clave=gate_AUT42 actual=ausente_o_no_acreditado esperado=gate_real_propietario_AUT_boolean_EXECUTE_AD' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR to_regrole('vec_contexto_actor_v1_propietario') IS NULL THEN
  RAISE EXCEPTION 'AD184: PARO clave=preimagen actual=incompatible esperado=AD173_AUT42_sin_AD184' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.referencia_denominacion_v1(v text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$ SELECT v IS NOT NULL AND octet_length(v) BETWEEN 3 AND 128 AND v ~ '^[A-Za-z0-9_:-]+$' $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.base64_denominacion_v1(v bytea)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$ SELECT replace(encode(v,'base64'),chr(10),'') $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.canon_sobre_denominacion_persona_v1(s jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE i jsonb:=s->'Indice';t jsonb;anterior bytea;actual bytea;tokens text:='';k text;v text;
BEGIN
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(i) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(s))<>7 OR (SELECT count(*) FROM jsonb_object_keys(i))<>5
 OR NOT s ?& ARRAY['Esquema','PersonaRef','ClaveRef','Version','Nonce','Cifrado','Indice']
 OR NOT i ?& ARRAY['AmbitoRef','NormaRef','NormaSHA256','ClaveRef','Tokens']
 OR s->>'Esquema' IS DISTINCT FROM 'vec.persona.denominacion.aead.v1'
 OR s->>'PersonaRef' !~ '^per_[A-Za-z0-9_-]{22,124}$' OR length(s->>'PersonaRef')>128
 OR jsonb_typeof(s->'Version') IS DISTINCT FROM 'number' OR s->>'Version' !~ '^[1-9][0-9]{0,15}$'
 OR (s->>'Version')::numeric>9007199254740991
 OR jsonb_typeof(i->'NormaSHA256') IS DISTINCT FROM 'string' OR i->>'NormaSHA256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(i->'Tokens') IS DISTINCT FROM 'array' OR jsonb_array_length(i->'Tokens') NOT BETWEEN 1 AND 32
 THEN RAISE EXCEPTION 'AD184: sobre inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['ClaveRef'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR vec_autorizacion_atestada_v3.referencia_denominacion_v1(s->>k) IS NOT TRUE THEN RAISE EXCEPTION 'AD184: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['AmbitoRef','NormaRef','ClaveRef'] LOOP
  IF jsonb_typeof(i->k) IS DISTINCT FROM 'string' OR vec_autorizacion_atestada_v3.referencia_denominacion_v1(i->>k) IS NOT TRUE THEN RAISE EXCEPTION 'AD184: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['Nonce','Cifrado'] LOOP
  v:=s->>k;
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR v IS NULL OR v!~'^[A-Za-z0-9+/]+={0,2}$'
  OR vec_autorizacion_atestada_v3.base64_denominacion_v1(decode(v,'base64')) IS DISTINCT FROM v
  OR (k='Nonce' AND octet_length(decode(v,'base64'))<>12)
  OR (k='Cifrado' AND octet_length(decode(v,'base64')) NOT BETWEEN 17 AND 4112)
  THEN RAISE EXCEPTION 'AD184: cifrado inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR t IN SELECT value FROM jsonb_array_elements(i->'Tokens') LOOP
  v:=t#>>'{}';actual:=decode(v,'base64');
  IF jsonb_typeof(t) IS DISTINCT FROM 'string' OR octet_length(actual)<>32
  OR vec_autorizacion_atestada_v3.base64_denominacion_v1(actual) IS DISTINCT FROM v
  OR (anterior IS NOT NULL AND anterior>=actual) THEN RAISE EXCEPTION 'AD184: índice inválido' USING ERRCODE='22023'; END IF;
  tokens:=tokens||CASE WHEN tokens='' THEN '' ELSE ',' END||to_jsonb(v)::text;anterior:=actual;
 END LOOP;
 RETURN '{"Esquema":"vec.persona.denominacion.aead.v1","PersonaRef":'||to_jsonb(s->>'PersonaRef')::text||',"ClaveRef":'||to_jsonb(s->>'ClaveRef')::text||',"Version":'||(s->>'Version')||',"Nonce":'||to_jsonb(s->>'Nonce')::text||',"Cifrado":'||to_jsonb(s->>'Cifrado')::text||',"Indice":{"AmbitoRef":'||to_jsonb(i->>'AmbitoRef')::text||',"NormaRef":'||to_jsonb(i->>'NormaRef')::text||',"NormaSHA256":'||to_jsonb(i->>'NormaSHA256')::text||',"ClaveRef":'||to_jsonb(i->>'ClaveRef')::text||',"Tokens":['||tokens||']}}';
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(p_documento jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m jsonb:=p_documento;a jsonb;can text;sobre text;p_publicar boolean;
BEGIN
 IF p_documento IS NULL OR octet_length(p_documento::text)>65536 THEN RAISE EXCEPTION 'AD184: material_invalido' USING ERRCODE='22023';END IF;
 p_publicar:=m->>'esquema'='vec.persona.denominacion.publicar.v1';a:=m->'ambitos';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
 OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (SELECT count(*) FROM jsonb_object_keys(a))<>2
 OR NOT a ?& ARRAY['organizacion_ref','unidad_ref']
 OR vec_autorizacion_atestada_v3.referencia_denominacion_v1(a->>'organizacion_ref') IS NOT TRUE
 OR vec_autorizacion_atestada_v3.referencia_denominacion_v1(a->>'unidad_ref') IS NOT TRUE
 OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,124}$' OR length(m->>'persona_ref')>128
 THEN RAISE EXCEPTION 'AD184: material inválido' USING ERRCODE='22023'; END IF;
 can:='{"esquema":'||to_jsonb(m->>'esquema')::text||',"persona_ref":'||to_jsonb(m->>'persona_ref')::text;
 IF p_publicar THEN
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>10 OR NOT m ?& ARRAY['esquema','persona_ref','version_esperada','procedencia_ref','procedencia_version','procedencia_sha256','procedencia_autoridad','sobre_sha256','sobre','ambitos']
  OR m->>'esquema' IS DISTINCT FROM 'vec.persona.denominacion.publicar.v1'
  OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number' OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,15})$' OR (m->>'version_esperada')::numeric>=9007199254740991
  OR jsonb_typeof(m->'procedencia_version') IS DISTINCT FROM 'number' OR m->>'procedencia_version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'procedencia_version')::numeric>9007199254740991
  OR m->>'procedencia_ref' !~ '^prc_[A-Za-z0-9_-]{22,124}$' OR length(m->>'procedencia_ref')>128
  OR m->>'procedencia_sha256' !~ '^[0-9a-f]{64}$' OR m->>'procedencia_autoridad' NOT IN('no_autoritativa','autoridad_maestra_acreditada')
  THEN RAISE EXCEPTION 'AD184: publicación inválida' USING ERRCODE='22023'; END IF;
  sobre:=vec_autorizacion_atestada_v3.canon_sobre_denominacion_persona_v1(m->'sobre');
  IF m->>'persona_ref' IS DISTINCT FROM m#>>'{sobre,PersonaRef}' OR (m->>'version_esperada')::numeric+1 IS DISTINCT FROM (m#>>'{sobre,Version}')::numeric
  OR m->>'sobre_sha256' IS DISTINCT FROM encode(sha256(convert_to(sobre,'UTF8')),'hex') THEN RAISE EXCEPTION 'AD184: sobre divergente' USING ERRCODE='22023'; END IF;
  can:=can||',"version_esperada":'||(m->>'version_esperada')||',"procedencia_ref":'||to_jsonb(m->>'procedencia_ref')::text||',"procedencia_version":'||(m->>'procedencia_version')||',"procedencia_sha256":'||to_jsonb(m->>'procedencia_sha256')::text||',"procedencia_autoridad":'||to_jsonb(m->>'procedencia_autoridad')::text||',"sobre_sha256":'||to_jsonb(m->>'sobre_sha256')::text||',"sobre":'||sobre;
 ELSE
  IF (SELECT count(*) FROM jsonb_object_keys(m))<>4 OR NOT m ?& ARRAY['esquema','persona_ref','version','ambitos']
  OR m->>'esquema' IS DISTINCT FROM 'vec.persona.denominacion.leer.v1'
  OR jsonb_typeof(m->'version') IS DISTINCT FROM 'number' OR m->>'version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version')::numeric>9007199254740991 THEN RAISE EXCEPTION 'AD184: lectura inválida' USING ERRCODE='22023'; END IF;
  can:=can||',"version":'||(m->>'version');
 END IF;
 can:=can||',"ambitos":{"organizacion_ref":'||to_jsonb(a->>'organizacion_ref')::text||',"unidad_ref":'||to_jsonb(a->>'unidad_ref')::text||'}}';
 RETURN can;
END $f$;

-- No usa funciones ni tablas CA: CA32 valida su canon antes de esta llamada.
-- Aquí se comprueba el material semántico y se liga TODO su texto al recurso.
CREATE FUNCTION vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m jsonb;a jsonb;atributos text;canon text;accion text;k text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536 THEN RAISE EXCEPTION 'AD184: material_invalido' USING ERRCODE='22023';END IF;
 m:=p_material::jsonb;a:=m->'ambitos';
 IF vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(m) IS DISTINCT FROM p_material THEN RAISE EXCEPTION 'AD184: canon_divergente' USING ERRCODE='22023';END IF;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
 OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (SELECT count(*)FROM jsonb_object_keys(a))<>2
 OR NOT a ?& ARRAY['organizacion_ref','unidad_ref']
 OR jsonb_typeof(m->'persona_ref') IS DISTINCT FROM 'string' OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,124}$'
 THEN RAISE EXCEPTION 'AD184: material_invalido' USING ERRCODE='22023';END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ref']LOOP
  IF jsonb_typeof(a->k) IS DISTINCT FROM 'string' OR a->>k !~ '^[A-Za-z0-9_:-]{3,128}$' THEN RAISE EXCEPTION 'AD184: ambito_invalido' USING ERRCODE='22023';END IF;
 END LOOP;
 atributos:='{"material_sha256":'||to_jsonb(encode(sha256(convert_to(p_material,'UTF8')),'hex'))::text;
 IF m->>'esquema'='vec.persona.denominacion.publicar.v1' THEN
  accion:='vec.persona.denominacion.publicar';
  IF (SELECT count(*)FROM jsonb_object_keys(m))<>10 OR NOT m ?& ARRAY['esquema','persona_ref','version_esperada','procedencia_ref','procedencia_version','procedencia_sha256','procedencia_autoridad','sobre_sha256','sobre','ambitos']
  OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number' OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,15})$' OR (m->>'version_esperada')::numeric>=9007199254740991
  OR jsonb_typeof(m->'procedencia_ref') IS DISTINCT FROM 'string' OR m->>'procedencia_ref' !~ '^prc_[A-Za-z0-9_-]{22,124}$'
  OR jsonb_typeof(m->'procedencia_version') IS DISTINCT FROM 'number' OR m->>'procedencia_version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'procedencia_version')::numeric>9007199254740991
  OR jsonb_typeof(m->'procedencia_sha256') IS DISTINCT FROM 'string' OR m->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(m->'procedencia_autoridad') IS DISTINCT FROM 'string' OR m->>'procedencia_autoridad' NOT IN('no_autoritativa','autoridad_maestra_acreditada')
  OR jsonb_typeof(m->'sobre_sha256') IS DISTINCT FROM 'string' OR m->>'sobre_sha256' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(m->'sobre') IS DISTINCT FROM 'object' OR m->>'persona_ref' IS DISTINCT FROM m#>>'{sobre,PersonaRef}'
  OR jsonb_typeof(m#>'{sobre,Version}') IS DISTINCT FROM 'number' OR m#>>'{sobre,Version}' !~ '^[1-9][0-9]{0,15}$'
  OR (m#>>'{sobre,Version}')::numeric IS DISTINCT FROM (m->>'version_esperada')::numeric+1
  THEN RAISE EXCEPTION 'AD184: publicacion_invalida' USING ERRCODE='22023';END IF;
  -- JSON de maps Go: orden lexicográfico de claves; versión como string.
  atributos:=atributos||',"procedencia_autoridad":'||to_jsonb(m->>'procedencia_autoridad')::text||',"procedencia_sha256":'||to_jsonb(m->>'procedencia_sha256')::text||',"procedencia_version":'||to_jsonb(m->>'procedencia_version')::text;
 ELSIF m->>'esquema'='vec.persona.denominacion.leer.v1' THEN
  accion:='vec.persona.denominacion.leer';
  IF (SELECT count(*)FROM jsonb_object_keys(m))<>4 OR NOT m ?& ARRAY['esquema','persona_ref','version','ambitos']
  OR jsonb_typeof(m->'version') IS DISTINCT FROM 'number' OR m->>'version' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version')::numeric>9007199254740991 THEN RAISE EXCEPTION 'AD184: lectura_invalida' USING ERRCODE='22023';END IF;
 ELSE RAISE EXCEPTION 'AD184: esquema_invalido' USING ERRCODE='22023';END IF;
 canon:='{"ambitos":{"organizacion_ref":'||to_jsonb(a->>'organizacion_ref')::text||',"unidad_ref":'||to_jsonb(a->>'unidad_ref')::text||'},"atributos":'||atributos||'}}';
 RETURN jsonb_build_object('material',m,'accion',accion,'audiencia',accion||'.v1','recurso_ref',m->>'persona_ref','contexto_sha256',encode(sha256(convert_to(canon,'UTF8')),'hex'),'contexto_canonico',canon);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(text)FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.referencia_denominacion_v1(text),vec_autorizacion_atestada_v3.base64_denominacion_v1(bytea),vec_autorizacion_atestada_v3.canon_sobre_denominacion_persona_v1(jsonb),vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(jsonb)FROM PUBLIC;


CREATE FUNCTION vec_autorizacion_atestada_v3.validar_contrato_denominacion_persona_v1(p_material text,p_capacidad bytea,p_decision bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE r jsonb;c jsonb;d jsonb;campos jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR current_setting('role')<>'none' OR session_user=current_user
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN RAISE EXCEPTION 'AD184: contrato_rechazado' USING ERRCODE='42501';END IF;
 r:=vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(p_material);c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 campos:=CASE WHEN r->>'accion'='vec.persona.denominacion.publicar' THEN '["denominacion"]'::jsonb ELSE '["nombre_mostrar"]'::jsonb END;
 IF c->>'operacion' IS DISTINCT FROM r->>'accion' OR c->>'audiencia_consumo' IS DISTINCT FROM r->>'audiencia'
 OR c->>'efecto_ref' IS DISTINCT FROM r->>'recurso_ref' OR c->>'huella_efecto_sha256' IS DISTINCT FROM r->>'contexto_sha256'
 OR d->>'accion' IS DISTINCT FROM r->>'accion' OR d->>'modulo_id' IS DISTINCT FROM 'vec' OR d->>'tipo_recurso' IS DISTINCT FROM 'persona_denominacion'
 OR d->>'recurso_ref' IS DISTINCT FROM r->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM r->>'contexto_sha256'
 OR d->>'finalidad' IS DISTINCT FROM 'presentacion_persona' OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref' OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
 OR d->>'correlacion_ref' IS NULL OR d->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_autorizacion.validar_administrador_denominacion_persona_v1(d,r->'material') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD184: decision_nominal_rechazada' USING ERRCODE='42501';END IF;
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.validar_contrato_denominacion_persona_v1(text,bytea,bytea)FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid JOIN pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.oid=to_regrole('vec_persona_denominacion_ejecutor') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*)FROM pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1()FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''';
 runtime_nueva text:=E'       OR NOT (\n           (p_perfil_mutacion IN (''persona_denominacion_publicar'',''persona_denominacion_leer'')\n            AND vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS TRUE)\n           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin'$x$;
 excl_nueva text;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_publicar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_leer'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["nombre_mostrar"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
$x$;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM '6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9' THEN
  RAISE EXCEPTION 'AD184: PARO clave=nucleo_def_SHA actual=% esperado=6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9',h USING ERRCODE='55000';END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c' THEN
  RAISE EXCEPTION 'AD184: PARO clave=nucleo_src_SHA actual=% esperado=bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c',h USING ERRCODE='55000';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*)FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD184: PARO clave=nucleo_metadatos actual=incompatible esperado=propietario_AD_SECURITY_DEFINER_ACL_privada_config_original' USING ERRCODE='55000';END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d)ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d)ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 excl_nueva:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''persona_denominacion_publicar'' AND p_perfil_mutacion IS DISTINCT FROM ''persona_denominacion_leer''';
 IF length(original)-length(replace(original,marca,''))<>length(marca)
 OR length(original)-length(replace(original,runtime_marca,''))<>length(runtime_marca)
 OR length(original)-length(replace(original,excl_marca,''))<>length(excl_marca)
 THEN RAISE EXCEPTION 'AD184: PARO clave=marcas_nucleo actual=no_unicas esperado=tres_marcas_unicas' USING ERRCODE='55000';END IF;
 nueva:=replace(replace(replace(original,runtime_marca,runtime_nueva),excl_marca,excl_nueva),marca,extension||marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f)INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva OR replace(replace(replace(actual,extension||marca,marca),excl_nueva,excl_marca),runtime_nueva,runtime_marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc'FROM pg_proc p WHERE p.oid=f)IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d)ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d)ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()))IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD184: PARO clave=postimagen_nucleo actual=divergente esperado=extension_minima_y_metadata_intacta' USING ERRCODE='55000';END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE original text;nueva text;actual text;h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false)INTO STRICT original FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM '8935639700c8923c815400626135ce23add08a610a592eaaccc9ccd5d331b8c9' THEN RAISE EXCEPTION 'AD184: PARO clave=CHECK_audiencias_SHA actual=% esperado=8935639700c8923c815400626135ce23add08a610a592eaaccc9ccd5d331b8c9',h USING ERRCODE='55000';END IF;
 IF left(original,7)<>'CHECK (' OR right(original,1)<>')' OR strpos(original,'vec.persona.denominacion.')<>0 THEN RAISE EXCEPTION 'AD184: PARO clave=CHECK_audiencias_forma actual=incompatible esperado=CHECK_original_sin_denominacion' USING ERRCODE='55000';END IF;
 nueva:='CHECK (('||substr(original,8,length(original)-8)||') OR audiencia_consumo IN (''vec.persona.denominacion.publicar.v1'',''vec.persona.denominacion.leer.v1''))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE r jsonb;d jsonb;x record;perfil text;
BEGIN
 r:=vec_autorizacion_atestada_v3.validar_contrato_denominacion_persona_v1(p_material,p_capacidad,p_decision);
 IF vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS NOT TRUE THEN RAISE EXCEPTION 'AD184: login_rechazado' USING ERRCODE='42501';END IF;
 perfil:=CASE WHEN r->>'accion'='vec.persona.denominacion.publicar' THEN 'persona_denominacion_publicar' ELSE 'persona_denominacion_leer' END;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 -- La recuperación de un efecto necesita una decisión NUEVA. El cotejo
 -- separado sólo comprueba el acuse original, sin otra consulta de datos CA.
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM r->>'recurso_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM r->>'contexto_sha256'
 OR x.auditoria_ref IS NULL OR x.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c USING(decision_ref)JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t USING(decision_ref)
  WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='consumo_confirmado_v3' AND a.version_consumo=3
  AND a.efecto_ref=x.efecto_ref AND a.huella_efecto_sha256=x.huella_efecto_sha256
  AND a.actor_ref=d->>'principal_id' AND a.perfil_activo_ref=d->>'perfil_activo_ref' AND a.finalidad_ref='presentacion_persona'
  AND a.registrada_en=x.consumida_en AND c.consumida_en=x.consumida_en AND c.consumo_huella_sha256=x.consumo_huella_sha256
  AND t.capacidad_canonica=p_capacidad AND t.decision_canonica=p_decision AND t.motivo_canonico=p_motivo AND t.contexto_actor_canonico=p_contexto
  AND t.payload_vec_ad_3=p_payload AND t.sobre_cose_sign1=p_sobre AND t.evidencia_verificacion=p_evidencia AND t.raiz_publica_spki=p_raiz)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.validar_administrador_denominacion_persona_v1(d,r->'material') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD184: acuse_o_autoridad_rechazados' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_consumo_original jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE r jsonb;c jsonb;d jsonb;real jsonb;k record;cfg record;raiz record;ahora timestamptz;
BEGIN
 r:=vec_autorizacion_atestada_v3.validar_contrato_denominacion_persona_v1(p_material,p_capacidad,p_decision);
 IF vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS NOT TRUE OR jsonb_typeof(p_consumo_original) IS DISTINCT FROM 'object'
 OR (SELECT count(*)FROM jsonb_object_keys(p_consumo_original))<>7 OR NOT p_consumo_original ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']
 THEN RETURN false;END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 -- Todas las piezas deben ser las originales ya conservadas en AD. No basta
 -- un constructor Go de acuse ni la firma de un DTO aportado por el interesado.
 SELECT jsonb_build_object('decision_ref',s.decision_ref,'efecto_ref',s.efecto_ref,'huella_efecto_sha256',s.huella_efecto_sha256,'consumo_huella_sha256',s.consumo_huella_sha256,'auditoria_ref',a.auditoria_ref,'consumida_en',s.consumida_en,'consumo_nuevo',true)
 INTO real FROM vec_autorizacion_atestada_v3.consumo_decision_v3 s JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t USING(decision_ref)JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(decision_ref)
 WHERE s.decision_ref=c->>'decision_ref' AND s.efecto_ref=r->>'recurso_ref' AND s.huella_efecto_sha256=r->>'contexto_sha256'
 AND a.tipo_registro='consumo_confirmado_v3' AND a.version_consumo=3 AND a.actor_ref=d->>'principal_id' AND a.perfil_activo_ref=d->>'perfil_activo_ref' AND a.finalidad_ref='presentacion_persona'
 AND a.efecto_ref=s.efecto_ref AND a.huella_efecto_sha256=s.huella_efecto_sha256 AND a.registrada_en=s.consumida_en
 AND t.capacidad_canonica=p_capacidad AND t.decision_canonica=p_decision AND t.motivo_canonico=p_motivo AND t.contexto_actor_canonico=p_contexto
 AND t.payload_vec_ad_3=p_payload AND t.sobre_cose_sign1=p_sobre AND t.evidencia_verificacion=p_evidencia AND t.raiz_publica_spki=p_raiz;
 IF real IS NULL OR real IS DISTINCT FROM p_consumo_original THEN RETURN false;END IF;
 -- Gobierno actual: misma clave de emisión, configuración y raíz, sin
 -- recuperar una concesión histórica revocada como autoridad viva.
 ahora:=clock_timestamp();
 SELECT x.* INTO k FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=c->>'clave_id' AND x.version=(c->>'clave_version')::numeric FOR SHARE;
 IF NOT FOUND THEN RETURN false;END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision p WHERE p.clave_id=k.clave_id AND p.version=k.version AND p.establecida_en<=ahora
  AND p.orden=(SELECT max(x.orden)FROM vec_autorizacion_atestada_v3.puntero_clave_emision x WHERE x.clave_id=p.clave_id AND x.version=p.version AND x.establecida_en<=ahora))
 OR k.audiencia_consumo IS DISTINCT FROM r->>'audiencia' OR k.emisor_id IS DISTINCT FROM c->>'emisor_id'
 OR k.revision_gobierno IS DISTINCT FROM (c->>'revision_gobierno')::numeric OR k.huella_gobierno_sha256 IS DISTINCT FROM c->>'huella_gobierno_sha256'
 OR ahora<k.valida_desde OR ahora>=k.valida_hasta
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad x WHERE x.clave_id=k.clave_id AND x.version=k.version AND x.revocada_en<=ahora) THEN RETURN false;END IF;
 SELECT x.* INTO cfg FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=p.configuracion_revision WHERE p.establecida_en<=ahora ORDER BY p.orden DESC LIMIT 1 FOR SHARE OF p,x;
 IF NOT FOUND OR cfg.revision IS DISTINCT FROM c->>'revision_confianza' OR cfg.secuencia IS DISTINCT FROM (c->>'configuracion_secuencia')::numeric
 OR cfg.huella_configuracion_sha256 IS DISTINCT FROM c->>'huella_configuracion_sha256' OR ahora>=cfg.expira_en
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno cp WHERE cp.control_id AND cfg.secuencia<cp.configuracion_secuencia_minima)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=cfg.revision AND x.revocada_en<=ahora) THEN RETURN false;END IF;
 SELECT x.* INTO raiz FROM vec_autorizacion_atestada_v3.configuracion_raiz p JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=p.raiz_clave_id AND x.version=p.raiz_version WHERE p.configuracion_revision=cfg.revision AND x.clave_id=c->>'raiz_clave_id' AND x.version=(c->>'raiz_version')::numeric FOR SHARE OF x;
 IF NOT FOUND OR raiz.clave_publica_spki IS DISTINCT FROM p_raiz OR raiz.huella_spki_sha256 IS DISTINCT FROM c->>'huella_raiz_spki_sha256'
 OR ahora<raiz.valida_desde OR ahora>=raiz.valida_hasta
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno cp WHERE cp.control_id AND raiz.version<cp.raiz_version_minima)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE x.raiz_clave_id=raiz.clave_id AND x.raiz_version=raiz.version AND x.revocada_en<=ahora)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.validar_administrador_denominacion_persona_v1(d,r->'material') IS NOT TRUE THEN RETURN false;END IF;
 -- Es una revalidación inmediata antes del callback, no una consulta nueva
 -- mediante una capacidad pasada. Una consulta nueva exige nuevo consumo.
 RETURN clock_timestamp()<(c->>'expira_en')::timestamptz AND clock_timestamp()<(c->>'decision_valida_hasta')::timestamptz
  AND clock_timestamp()<cfg.expira_en AND clock_timestamp()<raiz.valida_hasta AND clock_timestamp()<k.valida_hasta;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)FROM PUBLIC;
-- Puerto puro de formato para AUT42; no lectura de datos ni capacidad.
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(jsonb),vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(text)TO vec_autorizacion_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)TO vec_contexto_actor_v1_propietario;
DO $acl$
DECLARE firma text;f oid;
BEGIN
 FOREACH firma IN ARRAY ARRAY['vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)']LOOP
  f:=to_regprocedure(firma);
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
  OR NOT has_function_privilege('vec_contexto_actor_v1_propietario',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=f AND(a.grantee NOT IN(p.proowner,'vec_contexto_actor_v1_propietario'::regrole)OR a.privilege_type<>'EXECUTE'OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD184: PARO clave=ACL_nominal actual=incompatible esperado=AD_CA_owners_EXECUTE_sin_grant_option' USING ERRCODE='55000';END IF;
 END LOOP;
END $acl$;
COMMIT;
