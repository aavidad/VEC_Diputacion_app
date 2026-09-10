\set ON_ERROR_STOP on
-- CT86: anotación administrativa sobre una incorporación ya registrada.
-- Avance únicamente: no reescribe CT70--85 ni cambia ACL/roles existentes.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000086:anotacion-administrativa',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $pre$
DECLARE n text; f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8' THEN
  RAISE EXCEPTION 'CT86: propietario incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['expediente_version_integral','expediente_integral_actual','actuacion_expediente_integral',
   'outbox_expediente_integral','control_cadenas_expediente_integral','incorporacion_registro_v2','seguimiento_raiz_v2','seguimiento_estado_v2'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=('vec_contratacion_temporal.'||n)::regclass
    AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CT86: fuente durable incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_anotacion_administrativa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
  RAISE EXCEPTION 'CT86: consumidor V3 nominal AD3-30 no disponible' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_contratacion_temporal.registrar_anotacion_administrativa_incorporacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR to_regclass('vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1') IS NOT NULL THEN
  RAISE EXCEPTION 'CT86: objeto preexistente' USING ERRCODE='55000'; END IF;
END $pre$;

DO $origen$
DECLARE actual text;
BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT actual FROM pg_constraint
  WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
   AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
 IF actual IS DISTINCT FROM $preimagen$CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text, 'resolucion_formalizacion_o6'::text])))$preimagen$ THEN
  RAISE EXCEPTION 'CT86: preimagen origen_version incompatible' USING ERRCODE='55000'; END IF;
END $origen$;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check
 CHECK (origen_version IN ('alta_o2','analisis_o3','cobertura_o4','asignacion_o5','informe_juridico_o5','fiscalizacion_o5','propuesta_formalizacion_o6','resolucion_formalizacion_o6','anotacion_administrativa_ct86'));

-- La tabla es el recibo original inmutable; un replay consume una nueva
-- concesión AD3-30 y devuelve este recibo sin otra actuación/evento.
CREATE TABLE vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 (
 recibo_ref text PRIMARY KEY, organizacion_ref text NOT NULL, expediente_ref text NOT NULL UNIQUE,
 ambito_hmac text NOT NULL, huella_peticion_hmac text NOT NULL,
 material_json jsonb NOT NULL, version_anterior numeric(20,0) NOT NULL,
 version_resultante numeric(20,0) NOT NULL,
 recibo_incorporacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref),
 seguimiento_original jsonb NOT NULL, estado_seguimiento_sha256 text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE, evento_ref text NOT NULL UNIQUE,
 recibo_json jsonb NOT NULL, registrada_en timestamptz(6) NOT NULL,
 UNIQUE(organizacion_ref,ambito_hmac),
 FOREIGN KEY(expediente_ref,version_anterior) REFERENCES vec_contratacion_temporal.expediente_version_integral,
 FOREIGN KEY(expediente_ref,version_resultante) REFERENCES vec_contratacion_temporal.expediente_version_integral,
 CHECK(version_anterior BETWEEN 1 AND 9007199254740990 AND version_resultante=version_anterior+1),
 CHECK(jsonb_typeof(material_json)='object' AND jsonb_typeof(recibo_json)='object'),
 CHECK(isfinite(registrada_en))
);
ALTER TABLE vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.anotacion86_material(m jsonb) RETURNS void
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; s text;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_forma(m,ARRAY['organizacion_ref','expediente_ref','solicitud_personal_ref','version_esperada','actor_ref','perfil_ref','observaciones']);
 FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','solicitud_personal_ref','actor_ref','perfil_ref'] LOOP
  s:=m->>k;
  IF jsonb_typeof(m->k) IS DISTINCT FROM 'string' OR s !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
   RAISE EXCEPTION 'material inválido' USING ERRCODE='P0860'; END IF;
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_escalar(m->'version_esperada','u1');
 IF (m->>'version_esperada')::numeric NOT BETWEEN 1 AND 9007199254740990
  OR jsonb_typeof(m->'observaciones') IS DISTINCT FROM 'string' THEN
  RAISE EXCEPTION 'material inválido' USING ERRCODE='P0860'; END IF;
 s:=m->>'observaciones';
 IF char_length(s) NOT BETWEEN 1 AND 2000 OR s IS DISTINCT FROM normalize(s,NFC)
  OR s IS DISTINCT FROM btrim(s,E' \t\r\n') OR s ~ '[\x00-\x08\x0b-\x1f\x7f]' THEN
  RAISE EXCEPTION 'observaciones inválidas' USING ERRCODE='P0860'; END IF;
END $f$;

-- Mismo codec Go de texto JSON ya publicado en CT70; mapas ordenados como
-- encoding/json. Solo se usa sobre las cadenas nominales del recurso V3.
CREATE FUNCTION vec_contratacion_temporal.anotacion86_mapa(j jsonb) RETURNS text
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; v jsonb; r text:='';
BEGIN
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' THEN RAISE EXCEPTION 'mapa inválido' USING ERRCODE='P0860'; END IF;
 FOR k,v IN SELECT key,value FROM jsonb_each(j) ORDER BY key COLLATE "C" LOOP
  IF jsonb_typeof(v) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'mapa inválido' USING ERRCODE='P0860'; END IF;
  r:=r||CASE WHEN r='' THEN '' ELSE ',' END||vec_contratacion_temporal.texto_json_incorporacion_go_v2(k)||':'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(v#>>'{}');
 END LOOP;
 RETURN '{'||r||'}';
END $f$;

-- Fuente interna: coteja el resultado exacto de CT75 y su estado resultante.
-- No consulta el puntero de estado actual ni usa la raíz como permiso.
CREATE FUNCTION vec_contratacion_temporal.anotacion86_origen(m jsonb) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE i vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 e vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 d vec_contratacion_temporal.seguimiento_definicion_v2%ROWTYPE; canon bytea;
BEGIN
 SELECT * INTO STRICT i FROM vec_contratacion_temporal.incorporacion_registro_v2
  WHERE organizacion_ref=m->>'organizacion_ref' AND expediente_ref=m->>'expediente_ref'
    AND solicitud_ref=m->>'solicitud_personal_ref';
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=i.seguimiento_ref;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.seguimiento_estado_v2
  WHERE seguimiento_ref=i.seguimiento_ref AND version_seguimiento=i.version_resultante AND estado_sha256=i.estado_resultante_sha256;
 SELECT * INTO STRICT d FROM vec_contratacion_temporal.seguimiento_definicion_v2
  WHERE definicion_ref=r.definicion_ref AND definicion_version=r.definicion_version AND definicion_sha256=r.definicion_sha256;
 canon:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(d.publicacion_json,e.estado_json);
 IF i.version_anterior IS DISTINCT FROM 0::numeric OR i.version_resultante IS DISTINCT FROM 1::numeric
  OR i.version_expediente>(m->>'version_esperada')::numeric
  OR r.organizacion_ref IS DISTINCT FROM i.organizacion_ref OR r.expediente_ref IS DISTINCT FROM i.expediente_ref
  OR e.organizacion_ref IS DISTINCT FROM i.organizacion_ref OR e.expediente_ref IS DISTINCT FROM i.expediente_ref
  OR e.relacion_ref IS DISTINCT FROM r.relacion_ref OR e.definicion_ref IS DISTINCT FROM r.definicion_ref OR e.definicion_version IS DISTINCT FROM r.definicion_version OR e.definicion_sha256 IS DISTINCT FROM r.definicion_sha256
  OR e.raiz_sha256 IS DISTINCT FROM r.raiz_sha256
  OR r.raiz_sha256 IS DISTINCT FROM encode(sha256(r.raiz_canonica),'hex')
  OR e.estado_canonico IS DISTINCT FROM canon OR e.estado_sha256 IS DISTINCT FROM encode(sha256(canon),'hex')
  OR i.material_canonico IS DISTINCT FROM vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(i.material_json)
  OR i.material_sha256 IS DISTINCT FROM encode(sha256(i.material_canonico),'hex')
  OR i.recibo_json#>>'{Transicion,recibo_ref}' IS DISTINCT FROM i.recibo_ref
  OR (i.recibo_json->>'VersionSeguimientoResultante')::numeric IS DISTINCT FROM i.version_resultante
  OR i.recibo_json->>'HuellaEstadoResultante' IS DISTINCT FROM i.estado_resultante_sha256 THEN
  RAISE EXCEPTION 'incorporación original divergente' USING ERRCODE='P0862'; END IF;
 RETURN jsonb_build_object('seguimiento_original',jsonb_build_object('seguimiento_ref',i.seguimiento_ref,'version_seguimiento',i.version_resultante,'huella_raiz_seguimiento_sha256',r.raiz_sha256),
  'estado_seguimiento_sha256',i.estado_resultante_sha256,'recibo_incorporacion_ref',i.recibo_ref);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'incorporación original no disponible' USING ERRCODE='P0862';
END $f$;

CREATE FUNCTION vec_contratacion_temporal.preparar_anotacion_administrativa_incorporacion_v1(p jsonb) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE m jsonb; pares jsonb; par jsonb; elegido jsonb; r vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1%ROWTYPE;
 v vec_contratacion_temporal.expediente_version_integral%ROWTYPE; origen jsonb; encontrado boolean; recibo text; evento text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' THEN RAISE EXCEPTION 'preparación denegada' USING ERRCODE='P0863'; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(p,ARRAY['material','sellos_hmac']);
 m:=p->'material'; PERFORM vec_contratacion_temporal.anotacion86_material(m);
 PERFORM vec_contratacion_temporal.seguimiento73_forma(p->'sellos_hmac',ARRAY['activo','retenidos']);
 IF jsonb_typeof(p#>'{sellos_hmac,retenidos}') IS DISTINCT FROM 'array' OR jsonb_array_length(p#>'{sellos_hmac,retenidos}')>16 THEN RAISE EXCEPTION 'sellos inválidos' USING ERRCODE='P0860'; END IF;
 pares:=jsonb_build_array(p#>'{sellos_hmac,activo}')||(p#>'{sellos_hmac,retenidos}');
 FOR par IN SELECT value FROM jsonb_array_elements(pares) LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_forma(par,ARRAY['generacion','ambito_hmac','huella_peticion_hmac']);
  PERFORM vec_contratacion_temporal.seguimiento73_escalar(par->'generacion','u1');
  IF coalesce(par->>'ambito_hmac','') !~ '^hmac-sha256:[a-z][a-z0-9._/-]{1,95}:[a-f0-9]{64}$'
   OR coalesce(par->>'huella_peticion_hmac','') !~ '^hmac-sha256:[a-z][a-z0-9._/-]{1,95}:[a-f0-9]{64}$' THEN RAISE EXCEPTION 'sellos inválidos' USING ERRCODE='P0860'; END IF;
 END LOOP;
 SELECT * INTO r FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 a
  WHERE a.organizacion_ref=m->>'organizacion_ref' AND EXISTS(SELECT 1 FROM jsonb_array_elements(pares) x WHERE x->>'ambito_hmac'=a.ambito_hmac);
 encontrado:=FOUND;
 IF encontrado THEN
  SELECT x INTO STRICT elegido FROM jsonb_array_elements(pares) x WHERE x->>'ambito_hmac'=r.ambito_hmac;
  IF r.material_json IS DISTINCT FROM m OR elegido->>'huella_peticion_hmac' IS DISTINCT FROM r.huella_peticion_hmac THEN RAISE EXCEPTION 'clave reutilizada' USING ERRCODE='P0861'; END IF;
  SELECT * INTO STRICT v FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=r.expediente_ref AND version=r.version_anterior;
  recibo:=r.recibo_ref; evento:=r.evento_ref;
 ELSE
  elegido:=p#>'{sellos_hmac,activo}';
  SELECT z.* INTO STRICT v FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral z USING(expediente_ref,version) WHERE a.expediente_ref=m->>'expediente_ref';
  IF v.version IS DISTINCT FROM (m->>'version_esperada')::numeric
   OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 WHERE expediente_ref=m->>'expediente_ref') THEN RAISE EXCEPTION 'versión incompatible' USING ERRCODE='P0865'; END IF;
  recibo:='recibo:'||gen_random_uuid()::text; evento:='evento:'||gen_random_uuid()::text;
 END IF;
 IF v.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
  OR v.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(v.agregado_json::text,'UTF8')),'hex') THEN RAISE EXCEPTION 'expediente divergente' USING ERRCODE='P0862'; END IF;
 origen:=vec_contratacion_temporal.anotacion86_origen(m);
 IF encontrado AND (r.seguimiento_original IS DISTINCT FROM origen->'seguimiento_original' OR r.estado_seguimiento_sha256 IS DISTINCT FROM origen->>'estado_seguimiento_sha256' OR r.recibo_incorporacion_ref IS DISTINCT FROM origen->>'recibo_incorporacion_ref') THEN RAISE EXCEPTION 'antecedente divergente' USING ERRCODE='P0862'; END IF;
 RETURN origen||jsonb_build_object('expediente',v.agregado_json,'material',m,'referencias',jsonb_build_object('recibo_ref',recibo,'evento_ref',evento),
  'ambito_idempotencia_hmac',elegido->'ambito_hmac','huella_peticion_hmac',elegido->'huella_peticion_hmac',
  'estado',CASE WHEN encontrado THEN 'confirmada' ELSE 'preparada' END,'recibo_confirmado',CASE WHEN encontrado THEN r.recibo_json ELSE NULL END);
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'preparación no disponible' USING ERRCODE='P0862';
END $f$;

CREATE FUNCTION vec_contratacion_temporal.registrar_anotacion_administrativa_incorporacion_v1(
 p jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE m jsonb; origen jsonb; d jsonb; ambitos jsonb; atributos jsonb; contexto_canon bytea; hcontexto text; consumo record;
 v vec_contratacion_temporal.expediente_version_integral%ROWTYPE; r vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1%ROWTYPE;
 actual numeric; encontrado boolean; previo jsonb; siguiente jsonb; a jsonb; esperada jsonb; referencias jsonb; politica jsonb;
 instante timestamptz; ahora timestamptz; recibo text; evento text; prueba bytea; payload bytea; huella text; cabeza text; secuencia numeric; resultado jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'anotación denegada' USING ERRCODE='P0863'; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(p,ARRAY['material','expediente_anterior','expediente_siguiente','seguimiento_original','estado_seguimiento_sha256','recibo_incorporacion_ref','ambito_idempotencia_hmac','huella_peticion_hmac','referencias','politica','instante_efecto']);
 IF pg_column_size(p)>4194304 THEN RAISE EXCEPTION 'anotación excesiva' USING ERRCODE='P0860'; END IF;
 m:=p->'material'; PERFORM vec_contratacion_temporal.anotacion86_material(m);
 FOREACH huella IN ARRAY ARRAY[p->>'ambito_idempotencia_hmac',p->>'huella_peticion_hmac'] LOOP
  IF huella IS NULL OR huella !~ '^hmac-sha256:[a-z][a-z0-9._/-]{1,95}:[a-f0-9]{64}$' THEN RAISE EXCEPTION 'sellos inválidos' USING ERRCODE='P0860'; END IF;
 END LOOP;
 -- Misma serialización de todas las escrituras por expediente; el CAS final sigue obligatorio.
 PERFORM pg_advisory_xact_lock(hashtextextended('CT86:'||(m->>'expediente_ref'),0));
 SELECT * INTO r FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1
  WHERE organizacion_ref=m->>'organizacion_ref' AND ambito_hmac=p->>'ambito_idempotencia_hmac' FOR SHARE;
 encontrado:=FOUND;
 IF encontrado AND (r.material_json IS DISTINCT FROM m OR r.huella_peticion_hmac IS DISTINCT FROM p->>'huella_peticion_hmac') THEN RAISE EXCEPTION 'clave reutilizada' USING ERRCODE='P0861'; END IF;
 SELECT version INTO STRICT actual FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=m->>'expediente_ref' FOR UPDATE;
 SELECT * INTO STRICT v FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=m->>'expediente_ref' AND version=(m->>'version_esperada')::numeric;
 IF v.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref' OR v.agregado_json IS DISTINCT FROM p->'expediente_anterior'
  OR v.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(v.agregado_json::text,'UTF8')),'hex')
  OR v.estado IN ('completado','cancelado') THEN RAISE EXCEPTION 'preimagen divergente' USING ERRCODE='P0862'; END IF;
 IF NOT encontrado AND (actual IS DISTINCT FROM v.version OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 WHERE expediente_ref=m->>'expediente_ref')) THEN RAISE EXCEPTION 'versión incompatible' USING ERRCODE='P0865'; END IF;
 origen:=vec_contratacion_temporal.anotacion86_origen(m);
 IF origen->'seguimiento_original' IS DISTINCT FROM p->'seguimiento_original' OR origen->'estado_seguimiento_sha256' IS DISTINCT FROM p->'estado_seguimiento_sha256' OR origen->'recibo_incorporacion_ref' IS DISTINCT FROM p->'recibo_incorporacion_ref' THEN RAISE EXCEPTION 'antecedente divergente' USING ERRCODE='P0862'; END IF;
 politica:=p->'politica';
 PERFORM vec_contratacion_temporal.seguimiento73_forma(politica,ARRAY['definicion_ref','definicion_version','definicion_huella_sha256','evaluada_en','valida_hasta']);
 IF coalesce(politica->>'definicion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'política inválida' USING ERRCODE='P0860'; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_escalar(politica->'definicion_version','u1');
 PERFORM vec_contratacion_temporal.seguimiento73_escalar(politica->'definicion_huella_sha256','hash');
 PERFORM vec_contratacion_temporal.seguimiento73_micro(politica->'evaluada_en'); PERFORM vec_contratacion_temporal.seguimiento73_micro(politica->'valida_hasta');
 PERFORM vec_contratacion_temporal.seguimiento73_micro(p->'instante_efecto');
 instante:=(p->>'instante_efecto')::timestamptz; ahora:=clock_timestamp();
 IF instante<v.registrada_en OR ahora<instante OR ahora<(politica->>'evaluada_en')::timestamptz OR ahora>=(politica->>'valida_hasta')::timestamptz THEN RAISE EXCEPTION 'vigencia agotada' USING ERRCODE='P0863'; END IF;
 ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref','fase_previa',v.fase_clave,'estado_previo',v.estado);
 atributos:=jsonb_build_object('operacion','registrar_anotacion_administrativa','version_expediente',m->>'version_esperada','solicitud_personal_ref',m->>'solicitud_personal_ref',
  'seguimiento_ref',p#>>'{seguimiento_original,seguimiento_ref}','version_seguimiento',p#>>'{seguimiento_original,version_seguimiento}','huella_raiz_seguimiento_sha256',p#>>'{seguimiento_original,huella_raiz_seguimiento_sha256}',
  'estado_seguimiento_sha256',p->>'estado_seguimiento_sha256','recibo_incorporacion_ref',p->>'recibo_incorporacion_ref','observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
  'ambito_idempotencia_hmac',p->>'ambito_idempotencia_hmac','huella_peticion_hmac',p->>'huella_peticion_hmac','politica_ref',politica->>'definicion_ref','politica_version',politica->>'definicion_version','politica_huella_sha256',politica->>'definicion_huella_sha256');
 contexto_canon:=convert_to('{"ambitos":'||vec_contratacion_temporal.anotacion86_mapa(ambitos)||',"atributos":'||vec_contratacion_temporal.anotacion86_mapa(atributos)||'}','UTF8');
 hcontexto:=encode(sha256(contexto_canon),'hex');
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.anotacion_administrativa.registrar' OR d->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
  OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'anotacion_administrativa_incorporacion_v1'
  OR d->>'finalidad' IS DISTINCT FROM 'registrar_anotacion_administrativa_incorporacion'
  OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM hcontexto THEN RAISE EXCEPTION 'material no autorizado' USING ERRCODE='P0863'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_anotacion_administrativa_ct_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
  OR consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref' OR consumo.huella_efecto_sha256 IS DISTINCT FROM hcontexto
  OR coalesce(consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'consumo divergente' USING ERRCODE='P0863'; END IF;
 -- La recuperación exige consumo nuevo antes de revelar el recibo histórico.
 IF encontrado THEN
  IF r.seguimiento_original IS DISTINCT FROM p->'seguimiento_original' OR r.estado_seguimiento_sha256 IS DISTINCT FROM p->>'estado_seguimiento_sha256' OR r.recibo_incorporacion_ref IS DISTINCT FROM p->>'recibo_incorporacion_ref' THEN RAISE EXCEPTION 'recibo divergente' USING ERRCODE='P0862'; END IF;
  RETURN r.recibo_json;
 END IF;
 referencias:=p->'referencias'; PERFORM vec_contratacion_temporal.seguimiento73_forma(referencias,ARRAY['recibo_ref','evento_ref']);
 recibo:=referencias->>'recibo_ref'; evento:=referencias->>'evento_ref';
 IF coalesce(recibo,'') !~ '^recibo:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' OR coalesce(evento,'') !~ '^evento:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN RAISE EXCEPTION 'referencias inválidas' USING ERRCODE='P0860'; END IF;
 previo:=v.agregado_json; siguiente:=p->'expediente_siguiente';
 a:=siguiente->'actuaciones'->-1;
 esperada:=jsonb_build_object('secuencia',v.version+1,'version_expediente',v.version+1,'accion_clave','contratacion_temporal.anotacion_administrativa.registrar',
  'actor_ref',m->>'actor_ref','unidad_ref',previo#>>'{asignacion,unidad_ref}','recibo_ref',recibo,'realizada_en',p->'instante_efecto',
  'fase_origen',v.fase_clave,'fase_destino',v.fase_clave,'estado_origen',v.estado,'estado_destino',v.estado,'observaciones',m->>'observaciones','seguimiento_original',p->'seguimiento_original');
 IF coalesce(previo#>>'{asignacion,unidad_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(previo->'actuaciones') IS DISTINCT FROM 'array'
  OR jsonb_array_length(previo->'actuaciones') IS DISTINCT FROM v.version
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(previo->'actuaciones') x WHERE x->>'accion_clave'='contratacion_temporal.anotacion_administrativa.registrar')
  OR a IS DISTINCT FROM esperada OR siguiente IS DISTINCT FROM (previo||jsonb_build_object('version',v.version+1,'actualizado_en',p->'instante_efecto','actuaciones',(previo->'actuaciones')||jsonb_build_array(esperada))) THEN RAISE EXCEPTION 'postimagen divergente' USING ERRCODE='P0862'; END IF;
 huella:=encode(sha256(convert_to(siguiente::text,'UTF8')),'hex');
 -- Codec de prueba por operación, siguiendo CT61; conserva bytes/historia anteriores.
 prueba:=convert_to('VEC-CT-EXPEDIENTE-ANOTACION-V1'||chr(10)||(m->>'expediente_ref')||chr(10)||(v.version+1)::text||chr(10)||huella||chr(10)||recibo||chr(10)||consumo.decision_ref||chr(10)||instante::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral(expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
 VALUES(m->>'expediente_ref',v.version+1,siguiente,huella,prueba,encode(sha256(prueba),'hex'),v.flujo_ref,v.flujo_version,v.flujo_huella_sha256,v.fase_clave,v.estado,'anotacion_administrativa_ct86',recibo,instante);
 UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v.version+1,actualizada_en=instante,operacion_ref=recibo WHERE expediente_ref=m->>'expediente_ref' AND version=v.version;
 IF NOT FOUND THEN RAISE EXCEPTION 'versión perdida' USING ERRCODE='P0865'; END IF;
 -- La prueba acotada liga la huella del JSON íntegro; 2000 caracteres UTF8
 -- pueden ocupar más de 4096 bytes. El texto permanece completo en actuacion_json.
 prueba:=convert_to('VEC-CT-ACTUACION-ANOTACION-V1'||chr(10)||encode(sha256(convert_to(a::text,'UTF8')),'hex')||chr(10)||recibo||chr(10)||instante::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
 VALUES(m->>'expediente_ref',v.version+1,v.version+1,recibo,recibo,a,encode(sha256(convert_to(a::text,'UTF8')),'hex'),prueba,encode(sha256(prueba),'hex'),instante);
 SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT secuencia,cabeza FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
 IF secuencia>=9007199254740991 THEN RAISE EXCEPTION 'outbox agotado' USING ERRCODE='P0864'; END IF;
 payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.anotacion-administrativa.v1','expediente_ref',m->>'expediente_ref','version_resultante',v.version+1,'recibo_ref',recibo,'seguimiento_original',p->'seguimiento_original')::text,'UTF8');
 huella:=encode(sha256(cabeza::bytea||payload),'hex');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
 VALUES(evento,secuencia+1,recibo,m->>'expediente_ref',v.version+1,'contratacion_temporal.anotacion_administrativa_registrada',payload,encode(sha256(payload),'hex'),cabeza,huella,instante);
 UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral SET secuencia_outbox=secuencia+1,cabeza_outbox_sha256=huella,actualizada_en=instante WHERE control_id;
 resultado:=jsonb_build_object('operacion','registrar_anotacion_administrativa','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref','version_anterior',v.version,'version_resultante',v.version+1,'fase_resultante',v.fase_clave,'estado_resultante',v.estado,'seguimiento_original',p->'seguimiento_original','recibo_ref',recibo,'auditoria_ref',consumo.auditoria_ref,'evento_ref',evento,'actor_ref',m->>'actor_ref','registrada_en',p->'instante_efecto');
 INSERT INTO vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 VALUES(recibo,m->>'organizacion_ref',m->>'expediente_ref',p->>'ambito_idempotencia_hmac',p->>'huella_peticion_hmac',m,v.version,v.version+1,p->>'recibo_incorporacion_ref',p->'seguimiento_original',p->>'estado_seguimiento_sha256',consumo.auditoria_ref,evento,resultado,instante);
 RETURN resultado;
EXCEPTION WHEN unique_violation THEN RAISE EXCEPTION 'clave reutilizada' USING ERRCODE='P0861';
 WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'anotación no disponible' USING ERRCODE='P0862';
END $f$;

-- Lector privado estrecho: el bootstrap exige antes ConsultaDetalleRRHH V3.
-- No se monta como endpoint ni implementa el puerto de lectura autorizada.
CREATE FUNCTION vec_contratacion_temporal.recuperar_material_anotacion_administrativa_v1(p jsonb) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1%ROWTYPE;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'on' THEN RAISE EXCEPTION 'lectura privada denegada' USING ERRCODE='P0863'; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(p,ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref','ambitos_hmac']);
 IF jsonb_typeof(p->'ambitos_hmac') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'ambitos_hmac') NOT BETWEEN 1 AND 17 THEN RAISE EXCEPTION 'selector inválido' USING ERRCODE='P0860'; END IF;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1
  WHERE organizacion_ref=p->>'organizacion_ref' AND expediente_ref=p->>'expediente_ref'
   AND material_json->>'actor_ref'=p->>'actor_ref' AND material_json->>'perfil_ref'=p->>'perfil_ref'
   AND ambito_hmac IN(SELECT jsonb_array_elements_text(p->'ambitos_hmac'));
 RETURN r.material_json;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'anotación original no disponible' USING ERRCODE='P0862';
END $f$;

DO $acl$
DECLARE f regprocedure;
BEGIN
 FOR f IN SELECT oid::regprocedure FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND (proname LIKE 'anotacion86_%' OR proname IN ('recuperar_material_anotacion_administrativa_v1','preparar_anotacion_administrativa_incorporacion_v1','registrar_anotacion_administrativa_incorporacion_v1')) LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador',f);
 END LOOP;
END $acl$;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.recuperar_material_anotacion_administrativa_v1(jsonb) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_anotacion_administrativa_incorporacion_v1(jsonb) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_anotacion_administrativa_incorporacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
COMMIT;
