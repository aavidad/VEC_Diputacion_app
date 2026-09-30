\set ON_ERROR_STOP on
-- CT-154: acto prospectivo, de solo adicion, sobre una publicacion RPT exacta.
-- No rectifica el analisis, no altera la version del expediente y no reserva
-- ni confirma un uso del catalogo comun.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000154',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regclass('vec_contratacion_temporal.confirmacion_operacion_analisis') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal_propietario',
      'vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'CT-154: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1(
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 revision bigint NOT NULL,
 anterior_recibo_ref text,
 recibo_ref text NOT NULL UNIQUE,
 clave_idempotencia uuid NOT NULL,
 material_huella_sha256 text NOT NULL,
 version_expediente_esperada numeric(20,0) NOT NULL,
 analisis_version numeric(20,0) NOT NULL,
 analisis_recibo_ref text NOT NULL,
 analisis_huella_sha256 text NOT NULL,
 categoria_ref text NOT NULL,
 catalogo_id text NOT NULL,
 modulo_id text NOT NULL,
 catalogo_version integer NOT NULL,
 catalogo_huella_sha256 text NOT NULL,
 categoria_id text NOT NULL,
 fuente_ref text NOT NULL,
 motivo_ref text NOT NULL,
 aprobacion_ref text NOT NULL,
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 rpt_perfil_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL,
 rpt_decision_ref text NOT NULL,
 rpt_auditoria_ref text NOT NULL,
 rpt_consumo_huella_sha256 text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 recibo_json jsonb NOT NULL,
 PRIMARY KEY(expediente_ref,revision),
 UNIQUE(organizacion_ref,clave_idempotencia),
 UNIQUE(expediente_ref,recibo_ref),
 FOREIGN KEY(expediente_ref) REFERENCES vec_contratacion_temporal.expediente_alta(expediente_ref),
 FOREIGN KEY(expediente_ref,analisis_version) REFERENCES vec_contratacion_temporal.expediente_version_integral(expediente_ref,version),
 FOREIGN KEY(expediente_ref,anterior_recibo_ref) REFERENCES vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1(expediente_ref,recibo_ref),
 CHECK(revision BETWEEN 1 AND 1000000),
 CHECK(version_expediente_esperada BETWEEN 2 AND 9007199254740991::numeric),
 CHECK(analisis_version BETWEEN 2 AND version_expediente_esperada),
 CHECK(revision=1 AND anterior_recibo_ref IS NULL OR revision>1 AND anterior_recibo_ref IS NOT NULL),
 CHECK(analisis_huella_sha256 ~ '^[0-9a-f]{64}$' AND catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'
   AND material_huella_sha256 ~ '^[0-9a-f]{64}$'
   AND consumo_huella_sha256 ~ '^[0-9a-f]{64}$' AND rpt_consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 CHECK(categoria_ref=categoria_id),
 CHECK(registrada_en=date_trunc('microseconds',registrada_en)),
 CHECK(jsonb_typeof(recibo_json)='object')
);
CREATE INDEX vinculo_categoria_rpt_ct_v1_ultimo ON vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1(expediente_ref,revision DESC);
CREATE TRIGGER vinculo_categoria_rpt_ct_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER vinculo_categoria_rpt_ct_v1_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 FROM PUBLIC;

-- El evento solo describe una confirmacion ya consumida. No es un borrador
-- de reserva ni produce un mensaje hacia RPT.
CREATE TABLE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1(
 evento_ref text PRIMARY KEY,
 expediente_ref text NOT NULL,
 revision bigint NOT NULL,
 recibo_ref text NOT NULL UNIQUE,
 tipo text NOT NULL DEFAULT 'contratacion_temporal.categoria_rpt.vinculo.confirmado',
 confirmada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(expediente_ref,revision) REFERENCES vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1(expediente_ref,revision),
 CHECK(tipo='contratacion_temporal.categoria_rpt.vinculo.confirmado')
);
CREATE TRIGGER evento_vinculo_categoria_rpt_ct_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER evento_vinculo_categoria_rpt_ct_v1_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1 FROM PUBLIC;

-- Interna: lee exclusivamente CT. El recibo de analisis ha de proceder de
-- una confirmacion durable ligada a su propia version y al agregado exacto.
CREATE FUNCTION vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(
 p_organizacion_ref text,p_expediente_ref text)
RETURNS TABLE(version_expediente numeric,analisis_version numeric,analisis_recibo_ref text,
 analisis_huella_sha256 text,categoria_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$
 SELECT a.version,v.version,r.recibo_ref,
        vec_contratacion_temporal.huella_analisis_derivado_v2(v.agregado_json->'analisis'),
        v.agregado_json#>>'{analisis,categoria_ref}'
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral vigente
   ON vigente.expediente_ref=a.expediente_ref AND vigente.version=a.version
 JOIN vec_contratacion_temporal.expediente_version_integral v ON v.expediente_ref=a.expediente_ref
 JOIN vec_contratacion_temporal.actuacion_expediente_integral x
   ON x.expediente_ref=v.expediente_ref AND x.version_expediente=v.version AND x.secuencia=v.version
 JOIN vec_contratacion_temporal.reserva_operacion_analisis r
   ON r.expediente_ref=v.expediente_ref AND r.organizacion_ref=e.organizacion_ref
  AND r.version_expediente+1=v.version AND r.recibo_ref=x.recibo_ref
 JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c ON c.ambito_raiz_hmac=r.ambito_raiz_hmac
 WHERE e.organizacion_ref=p_organizacion_ref AND e.expediente_ref=p_expediente_ref
   AND v.version<=a.version AND v.origen_version='analisis_o3'
   AND vigente.agregado_json->'analisis'=v.agregado_json->'analisis'
   AND v.agregado_json->>'organizacion_ref'=e.organizacion_ref
   AND v.agregado_json->>'referencia'=e.expediente_ref
   AND vec_contratacion_temporal.analisis_rrhh_valido_v3(v.agregado_json->'analisis') IS TRUE
   AND v.agregado_json#>>'{analisis,actuacion_registro,recibo_ref}'=r.recibo_ref
   AND x.actuacion_json->>'recibo_ref'=r.recibo_ref
   AND ((r.operacion='registrar' AND x.actuacion_json->>'accion_clave'='contratacion_temporal.analisis.registrar')
     OR (r.operacion='rectificar' AND x.actuacion_json->>'accion_clave'='contratacion_temporal.analisis.rectificar'))
   AND c.recibo_json->>'recibo_ref'=r.recibo_ref
   AND c.recibo_json->>'expediente_ref'=e.expediente_ref
   AND c.recibo_json->>'organizacion_ref'=e.organizacion_ref
   AND c.recibo_json->>'version_resultante'=v.version::text
   AND c.confirmada_en=x.registrada_en
   AND encode(sha256(convert_to(c.recibo_json::text,'UTF8')),'hex')=c.recibo_huella_sha256
 ORDER BY v.version DESC LIMIT 1
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; a record; v record; consumo record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-154: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 32 AND 1024
 THEN RAISE EXCEPTION 'CT-154: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-154: JSON inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM json_each(p_material::json))<>3
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>3
    OR (SELECT array_agg(key ORDER BY orden) FROM json_each(p_material::json) WITH ORDINALITY AS j(key,valor,orden))
       IS DISTINCT FROM ARRAY['esquema','organizacion_ref','expediente_ref']
    OR jsonb_typeof(s->'esquema') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'organizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'expediente_ref') IS DISTINCT FROM 'string'
    OR s->>'esquema'<>'vec.ct.vinculo-categoria-rpt.consulta.v1'
    OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
 THEN RAISE EXCEPTION 'CT-154: consulta inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO a FROM vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(
  s->>'organizacion_ref',s->>'expediente_ref');
 IF NOT FOUND THEN RETURN jsonb_build_object('encontrado',false,'version_expediente',null,'analisis',null,'vinculo',null); END IF;
 SELECT * INTO v FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
  WHERE expediente_ref=s->>'expediente_ref' AND organizacion_ref=s->>'organizacion_ref'
  ORDER BY revision DESC LIMIT 1;
 RETURN jsonb_build_object('encontrado',true,'version_expediente',a.version_expediente,
   'analisis',jsonb_build_object('version',a.analisis_version,'recibo_ref',a.analisis_recibo_ref,
      'huella_sha256',a.analisis_huella_sha256,'categoria_ref',a.categoria_ref),
   'vinculo',CASE WHEN v.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object(
      'revision',v.revision,'recibo_ref',v.recibo_ref,'catalogo_id',v.catalogo_id,'modulo_id',v.modulo_id,
      'catalogo_version',v.catalogo_version,'catalogo_huella_sha256',v.catalogo_huella_sha256,
      'categoria_id',v.categoria_id,'fuente_ref',v.fuente_ref,
      'motivo_ref',v.motivo_ref,'aprobacion_ref',v.aprobacion_ref,
      'prospectivo',true,'acredita_procedencia_historica',false) END);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
 p_material text,
 p_ct_capacidad bytea,p_ct_decision bytea,p_ct_motivo bytea,p_ct_contexto bytea,
 p_ct_persona_version numeric,p_ct_perfil_version numeric,p_ct_payload bytea,p_ct_sobre bytea,p_ct_evidencia bytea,p_ct_raiz bytea,
 p_rpt_capacidad bytea,p_rpt_decision bytea,p_rpt_motivo bytea,p_rpt_contexto bytea,
 p_rpt_persona_version numeric,p_rpt_perfil_version numeric,p_rpt_payload bytea,p_rpt_sobre bytea,p_rpt_evidencia bytea,p_rpt_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; d jsonb; rd jsonb; ct record; rpt jsonb; anterior record; previa record; a record;
 rpt_material jsonb; recibo jsonb; vinculo jsonb; recibo_ref text; instante timestamptz(6);
 material_h text; revision_nueva bigint; version_esperada numeric; analisis_version numeric;
 revision_esperada bigint; clave uuid; catalogo_version integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-154: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 8192
 THEN RAISE EXCEPTION 'CT-154: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  s:=p_material::jsonb;
  d:=convert_from(p_ct_decision,'UTF8')::jsonb;
  rd:=convert_from(p_rpt_decision,'UTF8')::jsonb;
  clave:=(s->>'clave_idempotencia')::uuid;
  version_esperada:=(s->>'version_expediente_esperada')::numeric;
  analisis_version:=(s->>'analisis_version')::numeric;
  revision_esperada:=(s->>'revision_esperada')::bigint;
  catalogo_version:=(s->>'catalogo_version')::integer;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-154: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM json_each(p_material::json))<>19
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>19
    OR (SELECT array_agg(key ORDER BY orden) FROM json_each(p_material::json) WITH ORDINALITY AS j(key,valor,orden))
       IS DISTINCT FROM ARRAY['esquema','organizacion_ref','expediente_ref','version_expediente_esperada',
         'analisis_version','analisis_recibo_ref','analisis_huella_sha256','categoria_ref','catalogo_id',
         'modulo_id','catalogo_version','catalogo_huella_sha256','categoria_id','fuente_ref','motivo_ref',
         'aprobacion_ref','revision_esperada','anterior_recibo_ref','clave_idempotencia']
    OR s->>'esquema'<>'vec.ct.vinculo-categoria-rpt.registro.v1'
    OR (s-ARRAY['esquema','organizacion_ref','expediente_ref','version_expediente_esperada',
      'analisis_version','analisis_recibo_ref','analisis_huella_sha256','categoria_ref','catalogo_id',
      'modulo_id','catalogo_version','catalogo_huella_sha256','categoria_id','fuente_ref','motivo_ref',
      'aprobacion_ref','revision_esperada','anterior_recibo_ref','clave_idempotencia'])<>'{}'::jsonb
    OR jsonb_typeof(s->'esquema') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'organizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'expediente_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'version_expediente_esperada') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'analisis_version') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'analisis_recibo_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'analisis_huella_sha256') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'categoria_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'catalogo_id') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'modulo_id') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'catalogo_version') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'categoria_id') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'fuente_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'motivo_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'aprobacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'revision_esperada') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'clave_idempotencia') IS DISTINCT FROM 'string'
    OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR version_esperada NOT BETWEEN 2 AND 9007199254740991::numeric OR version_esperada<>trunc(version_esperada)
    OR analisis_version NOT BETWEEN 2 AND version_esperada OR analisis_version<>trunc(analisis_version)
    OR revision_esperada NOT BETWEEN 0 AND 999999
    OR catalogo_version NOT BETWEEN 1 AND 2147483647
    OR s->>'clave_idempotencia' IS DISTINCT FROM clave::text
    OR s->>'analisis_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR s->>'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR s->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
    OR s->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
    OR s->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
    OR s->>'categoria_ref' IS DISTINCT FROM s->>'categoria_id'
    OR s->>'analisis_recibo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'motivo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (revision_esperada=0 AND jsonb_typeof(s->'anterior_recibo_ref')<>'null')
    OR (revision_esperada>0 AND s->>'anterior_recibo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
 THEN RAISE EXCEPTION 'CT-154: petición inválida' USING ERRCODE='22023'; END IF;
 IF d->>'principal_id' IS NULL OR d->>'perfil_activo_ref' IS NULL
    OR rd->>'perfil_activo_ref' IS NULL
    OR d->>'principal_id' IS DISTINCT FROM rd->>'principal_id'
    OR d->>'perfil_activo_ref' IS NOT DISTINCT FROM rd->>'perfil_activo_ref'
 THEN RAISE EXCEPTION 'CT-154: identidades divergentes' USING ERRCODE='42501'; END IF;
 material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 rpt_material:=jsonb_build_object('catalogo_id',s->>'catalogo_id','modulo_id',s->>'modulo_id',
  'version',catalogo_version,'huella_sha256',s->>'catalogo_huella_sha256','categoria_id',s->>'categoria_id');
 -- Ambos consumos son frescos, en esta misma transacción. Ningún resultado
 -- local se confirma si falta la publicación RPT exacta o cualquiera de ellos.
 SELECT * INTO STRICT ct FROM vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(
  p_material,p_ct_capacidad,p_ct_decision,p_ct_motivo,p_ct_contexto,p_ct_persona_version,p_ct_perfil_version,
  p_ct_payload,p_ct_sobre,p_ct_evidencia,p_ct_raiz);
 rpt:=vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
  rpt_material,p_rpt_capacidad,p_rpt_decision,p_rpt_motivo,p_rpt_contexto,p_rpt_persona_version,p_rpt_perfil_version,
  p_rpt_payload,p_rpt_sobre,p_rpt_evidencia,p_rpt_raiz);
 IF rpt->>'consumo_nuevo' IS DISTINCT FROM 'true'
 THEN RAISE EXCEPTION 'CT-154: lectura RPT sin consumo fresco' USING ERRCODE='42501'; END IF;
 IF rpt->>'encontrado' IS DISTINCT FROM 'true'
    OR rpt#>>'{datos,publicacion,catalogo_id}' IS DISTINCT FROM s->>'catalogo_id'
    OR rpt#>>'{datos,publicacion,version}' IS DISTINCT FROM s->>'catalogo_version'
    OR rpt#>>'{datos,publicacion,huella_sha256}' IS DISTINCT FROM s->>'catalogo_huella_sha256'
    OR rpt#>>'{datos,entrada,clave}' IS DISTINCT FROM s->>'categoria_id'
 THEN RAISE EXCEPTION 'CT-154: publicación RPT ajena o ausente' USING ERRCODE='55000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:vinculo_categoria_rpt:'||(s->>'expediente_ref'),0));
 SELECT * INTO previa FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
  WHERE organizacion_ref=s->>'organizacion_ref' AND clave_idempotencia=clave;
 IF FOUND THEN
  IF previa.actor_ref IS DISTINCT FROM d->>'principal_id'
     OR previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
     OR previa.rpt_perfil_ref IS DISTINCT FROM rd->>'perfil_activo_ref'
     OR previa.material_huella_sha256 IS DISTINCT FROM material_h
     OR previa.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
  THEN RAISE EXCEPTION 'CT-154: clave reutilizada con otro material' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('replay',true,'recibo',previa.recibo_json,'vinculo',previa.recibo_json->'vinculo');
 END IF;
 SELECT * INTO a FROM vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(
  s->>'organizacion_ref',s->>'expediente_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'CT-154: análisis confirmado ausente' USING ERRCODE='55000'; END IF;
 IF a.version_expediente IS DISTINCT FROM version_esperada
    OR a.analisis_version IS DISTINCT FROM analisis_version
    OR a.analisis_recibo_ref IS DISTINCT FROM s->>'analisis_recibo_ref'
    OR a.analisis_huella_sha256 IS DISTINCT FROM s->>'analisis_huella_sha256'
    OR a.categoria_ref IS DISTINCT FROM s->>'categoria_ref'
 THEN RAISE EXCEPTION 'CT-154: anclaje o versión divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO anterior FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1
  WHERE expediente_ref=s->>'expediente_ref' ORDER BY revision DESC LIMIT 1;
 IF coalesce(anterior.revision,0)<>revision_esperada
    OR anterior.recibo_ref IS DISTINCT FROM s->>'anterior_recibo_ref'
 THEN RAISE EXCEPTION 'CT-154: revisión de vínculo divergente' USING ERRCODE='55000'; END IF;
 revision_nueva:=revision_esperada+1;
 instante:=date_trunc('microseconds',clock_timestamp());
 recibo_ref:='recibo:'||gen_random_uuid()::text;
 vinculo:=jsonb_build_object('revision',revision_nueva,'recibo_ref',recibo_ref,
  'catalogo_id',s->>'catalogo_id','modulo_id',s->>'modulo_id','catalogo_version',catalogo_version,
  'catalogo_huella_sha256',s->>'catalogo_huella_sha256','categoria_id',s->>'categoria_id',
  'fuente_ref',s->>'fuente_ref','motivo_ref',s->>'motivo_ref','aprobacion_ref',s->>'aprobacion_ref',
  'prospectivo',true,'acredita_procedencia_historica',false);
 recibo:=jsonb_build_object('recibo_ref',recibo_ref,'registrado_en',to_char(instante,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revision',revision_nueva,'material_huella_sha256',material_h,'prospectivo',true,
  'acredita_procedencia_historica',false,'decision_ref',ct.decision_ref,'auditoria_ref',ct.auditoria_ref,
  'consumo_huella_sha256',ct.consumo_huella_sha256,'rpt_decision_ref',rpt->>'decision_ref',
  'rpt_auditoria_ref',rpt->>'auditoria_ref','rpt_consumo_huella_sha256',rpt->>'consumo_huella_sha256',
  'rpt_perfil_ref',rd->>'perfil_activo_ref',
  'vinculo',vinculo);
 INSERT INTO vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1(
  organizacion_ref,expediente_ref,revision,anterior_recibo_ref,recibo_ref,clave_idempotencia,
  material_huella_sha256,version_expediente_esperada,analisis_version,analisis_recibo_ref,
  analisis_huella_sha256,categoria_ref,catalogo_id,modulo_id,catalogo_version,catalogo_huella_sha256,
  categoria_id,fuente_ref,motivo_ref,aprobacion_ref,actor_ref,perfil_ref,rpt_perfil_ref,decision_ref,auditoria_ref,
  consumo_huella_sha256,rpt_decision_ref,rpt_auditoria_ref,rpt_consumo_huella_sha256,registrada_en,recibo_json)
 VALUES(s->>'organizacion_ref',s->>'expediente_ref',revision_nueva,s->>'anterior_recibo_ref',recibo_ref,clave,
  material_h,version_esperada,analisis_version,s->>'analisis_recibo_ref',s->>'analisis_huella_sha256',
  s->>'categoria_ref',s->>'catalogo_id',s->>'modulo_id',catalogo_version,s->>'catalogo_huella_sha256',
  s->>'categoria_id',s->>'fuente_ref',s->>'motivo_ref',s->>'aprobacion_ref',d->>'principal_id',
  d->>'perfil_activo_ref',rd->>'perfil_activo_ref',ct.decision_ref,ct.auditoria_ref,ct.consumo_huella_sha256,
  rpt->>'decision_ref',rpt->>'auditoria_ref',rpt->>'consumo_huella_sha256',instante,recibo);
 INSERT INTO vec_contratacion_temporal.evento_vinculo_categoria_rpt_ct_v1(evento_ref,expediente_ref,revision,recibo_ref,confirmada_en)
 VALUES('evento:'||gen_random_uuid()::text,s->>'expediente_ref',revision_nueva,recibo_ref,instante);
 RETURN jsonb_build_object('replay',false,'recibo',recibo,'vinculo',vinculo);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure; nombre text; r record; permitido oid;
BEGIN
 FOREACH nombre IN ARRAY ARRAY[
  'vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text)',
  'vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
 ] LOOP
  f:=nombre::regprocedure;
  permitido:=CASE WHEN nombre LIKE '%anclaje_vinculo%' THEN NULL
   ELSE 'vec_contratacion_temporal_ejecutor'::regrole::oid END;
  FOR r IN SELECT DISTINCT a.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>p.proowner AND a.grantee IS DISTINCT FROM permitido LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN r.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(r.grantee)) END);
  END LOOP;
  IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_contratacion_temporal_propietario'::regrole
     OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee<>p.proowner AND a.grantee IS DISTINCT FROM permitido
                         OR a.privilege_type<>'EXECUTE'
                         OR a.grantee=permitido AND a.is_grantable))
  THEN RAISE EXCEPTION 'CT-154: ACL de función incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['vinculo_categoria_rpt_ct_v1','evento_vinculo_categoria_rpt_ct_v1'] LOOP
  FOR r IN SELECT DISTINCT a.grantee FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
    LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE n.nspname='vec_contratacion_temporal' AND c.relname=nombre AND a.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %s',nombre,
    CASE WHEN r.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(r.grantee)) END);
  END LOOP;
  FOR r IN SELECT DISTINCT a.grantee FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace,
    LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
   WHERE n.nspname='vec_contratacion_temporal' AND t.typname=nombre AND a.grantee<>t.typowner LOOP
   EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM %s',nombre,
    CASE WHEN r.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(r.grantee)) END);
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
