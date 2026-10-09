\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000201',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_emision_ct_v1(text,text,text,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.resolver_bolsa_vigente_ct_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(jsonb)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.contar_aceptaciones_firmes_ct_v1(jsonb)') IS NULL
    OR to_regrole('vec_contratacion_temporal_consultor_rrhh') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'CT201: PARO clave=preimagen esperado=B98_AD233_CT_actual actual=incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL REFERENCES vec_contratacion_temporal.expediente_alta(expediente_ref),
 bolsa_ref text NOT NULL,
 llamamiento_ref text NOT NULL UNIQUE,
 recibo_emision_ref text NOT NULL UNIQUE,
 version_expediente_esperada numeric(20,0) NOT NULL,
 clave_idempotencia uuid NOT NULL,
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_sha256 text NOT NULL CHECK(consumo_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE,
 vinculado_en timestamptz(6) NOT NULL,
 recibo_json jsonb NOT NULL,
 PRIMARY KEY(expediente_ref,llamamiento_ref),
 UNIQUE(organizacion_ref,clave_idempotencia),
 CHECK(version_expediente_esperada BETWEEN 1 AND 9007199254740991::numeric),
 CHECK(vinculado_en=date_trunc('microseconds',vinculado_en)),
 CHECK(jsonb_typeof(recibo_json)='object')
);
CREATE INDEX vinculo_emision_bolsa_ct_expediente_fecha
 ON vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(expediente_ref,vinculado_en,llamamiento_ref);
ALTER TABLE vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER vinculo_emision_bolsa_ct_inmutable BEFORE UPDATE OR DELETE
 ON vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 FOR EACH ROW
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER vinculo_emision_bolsa_ct_no_truncar BEFORE TRUNCATE
 ON vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 FROM PUBLIC;

-- Outbox local de solo adición: conserva exactamente el hecho confirmado.
CREATE TABLE vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1(
 evento_ref text PRIMARY KEY,
 expediente_ref text NOT NULL,
 llamamiento_ref text NOT NULL,
 recibo_ref text NOT NULL UNIQUE,
 tipo text NOT NULL CHECK(tipo='contratacion_temporal.bolsa.emision_vinculada.v1'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(expediente_ref,llamamiento_ref)
  REFERENCES vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(expediente_ref,llamamiento_ref)
);
ALTER TABLE vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER outbox_vinculo_emision_bolsa_ct_inmutable BEFORE UPDATE OR DELETE
 ON vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 FOR EACH ROW
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER outbox_vinculo_emision_bolsa_ct_no_truncar BEFORE TRUNCATE
 ON vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
REVOKE ALL ON TABLE vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1 FROM PUBLIC;

-- Prelectura interna de ámbitos del perfil RRHH. No entrega datos personales
-- ni crea una concesión; el acto posterior vuelve a cotejar el agregado.
CREATE FUNCTION vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
 p_organizacion text,p_expediente text,p_version numeric,p_bolsa text,
 p_llamamiento text,p_recibo text,p_clave text)
RETURNS TABLE(centro_ref text,categoria_ref text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='3s' AS $f$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR p_organizacion IS NULL OR p_expediente IS NULL OR p_bolsa IS NULL
    OR p_llamamiento IS NULL OR p_recibo IS NULL OR p_clave IS NULL
    OR p_version IS NULL OR p_version<1 OR p_version>9007199254740991::numeric
    OR p_version<>trunc(p_version) THEN
  RAISE EXCEPTION 'CT201: ámbitos no disponibles' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT v.agregado_json#>>'{solicitud,centro_ref}',
   v.agregado_json#>>'{analisis,categoria_ref}'
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.organizacion_ref=p_organizacion AND e.expediente_ref=p_expediente
   AND (a.version=p_version OR EXISTS(
      SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 previo
      WHERE previo.organizacion_ref=p_organizacion AND previo.expediente_ref=p_expediente
        AND previo.version_expediente_esperada=p_version
        AND previo.bolsa_ref=p_bolsa AND previo.llamamiento_ref=p_llamamiento
        AND previo.recibo_emision_ref=p_recibo AND previo.clave_idempotencia::text=p_clave))
   AND v.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
   AND v.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
   AND v.agregado_json#>'{via_cobertura,bolsa_ref}' IS NULL
   AND v.agregado_json#>>'{solicitud,centro_ref}' IS NOT NULL
   AND v.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
 text,text,numeric,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
 text,text,numeric,text,text,text,text) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; d jsonb; c record; actual_ct record; amb record; anterior record;
 clave uuid; version_esperada numeric; material_h text; contexto_h text;
 numero text; recibo text; evento text; instante timestamptz(6); salida jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT201: sesión de vínculo denegada' USING ERRCODE='42501';
 END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 4096 THEN
  RAISE EXCEPTION 'CT201: material inválido' USING ERRCODE='22023';
 END IF;
 BEGIN
  s:=p_material::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  clave:=(s->>'clave_idempotencia')::uuid;
  version_esperada:=(s->>'version_esperada')::numeric;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT201: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM json_each(p_material::json))<>8
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>8
    OR (SELECT array_agg(key ORDER BY ord) FROM json_each(p_material::json) WITH ORDINALITY AS j(key,val,ord))
      IS DISTINCT FROM ARRAY['esquema','organizacion_ref','expediente_ref','version_esperada',
        'bolsa_ref','llamamiento_ref','recibo_emision_ref','clave_idempotencia']
    OR s->>'esquema'<>'vec.ct.vinculo-emision-bolsa.v1'
    OR version_esperada NOT BETWEEN 1 AND 9007199254740991::numeric
    OR version_esperada<>trunc(version_esperada)
    OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR s->>'bolsa_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'llamamiento_ref' !~ '^llamamiento:[0-9a-f]{64}$'
    OR s->>'recibo_emision_ref' !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR s->>'clave_idempotencia' IS DISTINCT FROM clave::text THEN
  RAISE EXCEPTION 'CT201: contrato de vínculo inválido' USING ERRCODE='22023';
 END IF;
	material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
	SELECT * INTO amb FROM vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(
	 s->>'organizacion_ref',s->>'expediente_ref',version_esperada,s->>'bolsa_ref',
	 s->>'llamamiento_ref',s->>'recibo_emision_ref',s->>'clave_idempotencia');
	IF NOT FOUND THEN
	 RAISE EXCEPTION 'CT201: ámbitos del expediente divergentes' USING ERRCODE='23505';
	END IF;
	contexto_h:=encode(sha256(convert_to(
	   '{"ambitos":{"categoria_ref":'||to_json(amb.categoria_ref)::text||
	   ',"centro_ref":'||to_json(amb.centro_ref)::text||
	   ',"organizacion_ref":'||(s->'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'CT201: decisión no ligada al material' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT c FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF c.efecto_ref IS DISTINCT FROM s->>'expediente_ref' OR c.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR c.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: consumo de efecto divergente' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:vinculo_bolsa:'||(s->>'expediente_ref'),0));
 SELECT * INTO anterior FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.organizacion_ref=s->>'organizacion_ref' AND x.clave_idempotencia=clave;
 IF FOUND THEN
  IF anterior.material_sha256 IS DISTINCT FROM material_h
     OR anterior.actor_ref IS DISTINCT FROM d->>'principal_id'
     OR anterior.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
   RAISE EXCEPTION 'CT201: clave usada con otro contenido' USING ERRCODE='23505';
  END IF;
  RETURN anterior.recibo_json||jsonb_build_object('reutilizado',true);
 END IF;
 SELECT e.organizacion_ref,e.numero_visible,actual.version,v.agregado_json
  INTO actual_ct FROM vec_contratacion_temporal.expediente_alta e
  JOIN vec_contratacion_temporal.expediente_integral_actual actual ON actual.expediente_ref=e.expediente_ref
  JOIN vec_contratacion_temporal.expediente_version_integral v
    ON v.expediente_ref=actual.expediente_ref AND v.version=actual.version
  WHERE e.expediente_ref=s->>'expediente_ref' AND e.organizacion_ref=s->>'organizacion_ref'
  FOR SHARE OF actual;
 IF NOT FOUND OR actual_ct.version IS DISTINCT FROM version_esperada
    OR actual_ct.agregado_json#>>'{via_cobertura,via_clave}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}' IS DISTINCT FROM 'bolsa_vigente'
    OR actual_ct.agregado_json#>'{via_cobertura,bolsa_ref}' IS NOT NULL
    OR actual_ct.agregado_json#>>'{solicitud,centro_ref}' IS DISTINCT FROM amb.centro_ref
    OR actual_ct.agregado_json#>>'{analisis,categoria_ref}' IS DISTINCT FROM amb.categoria_ref THEN
  RAISE EXCEPTION 'CT201: versión o bolsa de cobertura divergente' USING ERRCODE='23505';
 END IF;
 numero:=vec_contratacion_temporal.numero_visible_vigente_v1(s->>'expediente_ref',actual_ct.numero_visible);
 IF vec_bolsa_llamamientos.verificar_emision_ct_v1(
    s->>'bolsa_ref',s->>'llamamiento_ref',s->>'recibo_emision_ref',numero,
    actual_ct.agregado_json#>>'{analisis,categoria_ref}') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT201: emisión real ajena al expediente' USING ERRCODE='42501';
 END IF;
 instante:=date_trunc('microseconds',clock_timestamp());
 recibo:='recibo:ct:bolsa:'||gen_random_uuid()::text;
 evento:='evento:ct:bolsa:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('expediente_ref',s->>'expediente_ref',
  'bolsa_ref',s->>'bolsa_ref','llamamiento_ref',s->>'llamamiento_ref',
  'recibo_emision_ref',s->>'recibo_emision_ref','recibo_vinculo_ref',recibo,
  'vinculado_en',instante,'auditoria_ref',c.auditoria_ref,'evento_ref',evento,'reutilizado',false);
 INSERT INTO vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1(
  organizacion_ref,expediente_ref,bolsa_ref,llamamiento_ref,recibo_emision_ref,
  version_expediente_esperada,clave_idempotencia,material_sha256,actor_ref,perfil_ref,
  decision_ref,auditoria_ref,consumo_sha256,recibo_ref,vinculado_en,recibo_json)
 VALUES(s->>'organizacion_ref',s->>'expediente_ref',s->>'bolsa_ref',s->>'llamamiento_ref',
  s->>'recibo_emision_ref',version_esperada,clave,material_h,d->>'principal_id',d->>'perfil_activo_ref',
  c.decision_ref,c.auditoria_ref,c.consumo_huella_sha256,recibo,instante,salida);
 INSERT INTO vec_contratacion_temporal.outbox_vinculo_emision_bolsa_ct_v1(
  evento_ref,expediente_ref,llamamiento_ref,recibo_ref,tipo,payload,payload_sha256,registrada_en)
 VALUES(evento,s->>'expediente_ref',s->>'llamamiento_ref',recibo,
  'contratacion_temporal.bolsa.emision_vinculada.v1',salida,
  encode(sha256(convert_to(salida::text,'UTF8')),'hex'),instante);
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

-- Reutiliza la lectura atestada de detalle: el consumo V3 y su auditoría se
-- ejecutan primero, y el resultado Bolsa se lee en la misma transacción.
-- El recibo CT conserva su canon original; el complemento sólo se entrega
-- con el detalle validado y la sesión nominal activa.
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_cursor text)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,resultado_bolsa jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
SET idle_in_transaction_session_timeout='6s' AS $f$
DECLARE d record; actual record; vinculos jsonb; sugeridas jsonb; n integer;
 entradas jsonb; todos jsonb; lote jsonb; pagina_n integer; antes_en timestamptz; antes_ref text;
 ultimo_en timestamptz; ultimo_ref text; mas boolean:=false; siguiente text; bolsa_vigente text;
 personas_text text; personas numeric; aceptaciones integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_consultor_rrhh','MEMBER') IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR (p_cursor IS NOT NULL AND p_cursor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z#llamamiento:[0-9a-f]{64}$') THEN
  RAISE EXCEPTION 'CT201: lectura de resultado denegada' USING ERRCODE='42501';
 END IF;
 IF p_cursor IS NOT NULL THEN
  antes_en:=split_part(p_cursor,'#',1)::timestamptz;
  antes_ref:=split_part(p_cursor,'#',2);
 END IF;
 SELECT * INTO STRICT d FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
  'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)),
  '[]'::jsonb) INTO todos FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.expediente_ref=d.expediente_ref;
 n:=jsonb_array_length(todos);
 aceptaciones:=vec_bolsa_llamamientos.contar_aceptaciones_firmes_ct_v1(todos);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
   'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO entradas
 FROM (SELECT v.bolsa_ref,v.llamamiento_ref,v.recibo_emision_ref,v.vinculado_en
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC LIMIT 20) x;
 pagina_n:=jsonb_array_length(entradas);
 IF pagina_n=20 THEN
  SELECT v.vinculado_en,v.llamamiento_ref INTO ultimo_en,ultimo_ref
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC OFFSET 19 LIMIT 1;
  SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
    WHERE v.expediente_ref=d.expediente_ref
      AND (v.vinculado_en,v.llamamiento_ref)<(ultimo_en,ultimo_ref)) INTO mas;
  IF mas THEN
   siguiente:=to_char(ultimo_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')||'#'||ultimo_ref;
  END IF;
 END IF;
 lote:=vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(entradas);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa_ref',x.bolsa_ref,'llamamiento_ref',x.llamamiento_ref,
   'recibo_emision_ref',x.recibo_emision_ref,
   'recibo_vinculo_ref',x.recibo_ref,'vinculado_en',x.vinculado_en,
   'emitido_en',b.valor->'emitido_en',
   'participaciones',b.valor->'participaciones')
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO vinculos
 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
 JOIN LATERAL jsonb_array_elements(lote) b(valor)
   ON b.valor->>'llamamiento_ref'=x.llamamiento_ref
  AND b.valor->>'bolsa_ref'=x.bolsa_ref
  AND b.valor->>'recibo_emision_ref'=x.recibo_emision_ref
 WHERE x.expediente_ref=d.expediente_ref;
 IF jsonb_array_length(vinculos)<>pagina_n THEN
  RAISE EXCEPTION 'CT201: resultado Bolsa no recuperable' USING ERRCODE='55000';
 END IF;
 SELECT e.numero_visible,a.version,v.agregado_json INTO actual
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.expediente_ref=d.expediente_ref;
 IF NOT FOUND OR actual.version IS DISTINCT FROM d.version_expediente THEN
  RAISE EXCEPTION 'CT201: agregado de lectura divergente' USING ERRCODE='55000';
 END IF;
 personas_text:=actual.agregado_json#>>'{solicitud,numero_personas}';
 IF personas_text IS NULL THEN
  personas_text:=actual.agregado_json#>>'{solicitud,necesidad,campos,numero_personas}';
 END IF;
 IF personas_text IS NOT NULL THEN
  IF personas_text !~ '^[1-9][0-9]{0,9}$' THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  IF personas_text::numeric>4294967295 THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  personas:=personas_text::numeric;
 END IF;
 sugeridas:='[]'::jsonb;
 IF actual.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND actual.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND actual.agregado_json#>'{via_cobertura,bolsa_ref}' IS NULL
    AND actual.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL THEN
  bolsa_vigente:=vec_bolsa_llamamientos.resolver_bolsa_vigente_ct_v1(
   actual.agregado_json#>>'{analisis,categoria_ref}');
 END IF;
 IF bolsa_vigente IS NOT NULL THEN
  SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY b.emitido_en DESC,b.llamamiento_ref DESC),'[]'::jsonb)
   INTO sugeridas FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,
    vec_contratacion_temporal.numero_visible_vigente_v1(d.expediente_ref,actual.numero_visible),
    NULL,NULL,20,ARRAY(SELECT x.llamamiento_ref
     FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
     WHERE x.expediente_ref=d.expediente_ref)) b;
 END IF;
 RETURN QUERY SELECT d.contenido_canonico,d.esquema,d.acceso_ref,d.secuencia,
  d.anterior_sha256,d.huella_sha256,d.vinculo_identidad_huella_sha256,
  d.alcance_huella_sha256,d.registrada_en,d.auditoria_vec_ref,
  d.auditoria_vec_huella_sha256,d.consumo_vec_huella_sha256,
  d.contenido_huella_sha256,d.resultado_huella_sha256,d.cursor_huella_sha256,
  d.generada_en,d.expediente_ref,d.version_expediente,d.total,d.recibo_sello_sha256,
  jsonb_build_object('vinculos',vinculos,'total_vinculos',n,
   'personas_solicitadas',personas,'aceptaciones_firmes',aceptaciones,
   'emisiones_vinculables',sugeridas,'siguiente_cursor',siguiente);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) TO vec_contratacion_temporal_consultor_rrhh;

-- La primera página conserva la firma que consume la ficha actual.
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(contenido_canonico bytea,esquema text,acceso_ref text,secuencia numeric,
 anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,resultado_huella_sha256 text,cursor_huella_sha256 text,
 generada_en timestamptz,expediente_ref text,version_expediente numeric,total smallint,
 recibo_sello_sha256 text,resultado_bolsa jsonb)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
SET idle_in_transaction_session_timeout='6s' AS $f$
 SELECT * FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,NULL::text)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
