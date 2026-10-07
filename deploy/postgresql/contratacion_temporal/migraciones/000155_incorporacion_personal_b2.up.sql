\set ON_ERROR_STOP on
-- CT155: incorporación, paso RRHH. Plan prospectivo autorizado antes de Personal,
-- origen final separado del ejercicio CT75; no declara firma ni eficacia legal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000155',0));
DO $pre$
BEGIN
 IF to_regclass('vec_contratacion_temporal.plan_incorporacion_personal_b2_v1') IS NOT NULL
 OR to_regprocedure('vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb)') IS NULL
 OR NOT has_function_privilege(current_user,'vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'CT155: dependencias incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

-- Rellenar SOLO con la preimagen final posterior a P23/CT130. Los NULL
-- impiden instalar esta migración con las huellas provisionales postAD131.
DO $preimagen_consumidores$
DECLARE x record;p record;acl_sha text;efectiva_sha text;dep_sha text;shdep_sha text;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('incorporacion_expediente_ct115(text,text)',false,'d9c103e638c97217803fac97c63a3f851b97032ecd26a01a4c175fb7a04d5060','6ce41dac491c4bad54fe6e71cd2f8a1ea21a1ae8b731e98a15c54dda9c09d8b9',ARRAY['search_path=pg_catalog']::text[],'bb292b36c6a0704ee9476dc9d0170093420eba24a503c6bfd30fc3c15a5512de','a80b2ec0aa2d2df2d426481d95e8f665a8f388deecdc74def71011f3d8bb0566','6d286e28cd336d2af53e945281f00406184134175ede49c5de1d2bfdd5eb31bb','e587f58f8c5335b5bdb34243a22749449549371082cf2d9aecaac3c90f09f165'),
  ('resultado_cese_ct115(vec_contratacion_temporal.cese_nombramiento_v1)',false,'bf55468cbbf940aa5fc12390b7714d0e523c655317fa555bb949f3e0fd9900fb','e750349774fd31a65adc070f7fd96016b453ba5b21069e73842931e652a74d60',ARRAY['search_path=pg_catalog']::text[],'bb292b36c6a0704ee9476dc9d0170093420eba24a503c6bfd30fc3c15a5512de','a80b2ec0aa2d2df2d426481d95e8f665a8f388deecdc74def71011f3d8bb0566','53cbfc730df7ec4b4dcecd4c43318c463307461cf112a1da02c6802906e6f123','ee929f1324ae183a649b219aa4fc76349ef908f7a7c98c028522ea77b2a86ed2'),
  ('resultado_ginpix_ct124(vec_contratacion_temporal.confirmacion_ginpix_v1)',false,'cafcf1a91da1a003405a32ce1ea1d1c1c9c48866ad9b9f2e1ff238591a4c2b4e','deb6c92426528c6a73c7cddf0f27ad534e4c89628a311c30633616b39f821dc1',ARRAY['search_path=pg_catalog']::text[],'bb292b36c6a0704ee9476dc9d0170093420eba24a503c6bfd30fc3c15a5512de','a80b2ec0aa2d2df2d426481d95e8f665a8f388deecdc74def71011f3d8bb0566','ee6f0f055918fc4e7a8d681273d5622053e6f2a471c7a4a4571deb38eb35c463','303a8f0a5f0a13d71351cd97f5b8beccdfa0d875546ddcc4fd64a0ac2703c884'),
  ('ginpix_confirmado_ct124(text,text)',false,'631a2573ac303016501357966f4b32451d6c9a02d3b031d72c705f8100b93b3f','72bde6ac566ad7a784995c2c389dddc8f9c5c4e07d568c8af9d77597bc444fab',ARRAY['search_path=pg_catalog']::text[],'bb292b36c6a0704ee9476dc9d0170093420eba24a503c6bfd30fc3c15a5512de','a80b2ec0aa2d2df2d426481d95e8f665a8f388deecdc74def71011f3d8bb0566','96d38bbd46d954c567683858a03b9e1c8ffebb46895e8e1b2e22787cab919f98','3dab34881c7a31e8e7916c756d73c75f9dc6f141388f41d801902f561ba17983'),
  ('confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',true,'8d237b59e7dd2dee3ad0b151ac20dece8536d9f76fd6ac2e7ca1db18155e8071','5fa395b4d8a86c3c6a783d7ab2f6d64a1cbba480a0770ebf7294c4f65256a6a6',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[],'dcf5f8d75fd8215c95882934ba547ceac7633a0bfa6abbbf529168f90c162cda','7ad94585991db40b728799f4c3730af4f1ee6479c77324644660f0f6a7d27b8f','1ee57b386daf022129098a812cc635623eb1cdf70de1e0891d77c4de7c2d7a25','32bc04bcfd957b06c3ded20c1f5e30167d3538fca2e5ccdc917bc0066cf4f433'),
  ('origen_reincorporacion_ct130(jsonb)',false,'e4e7aac037419fce5dc5687c023af1208969576db995c7b30c70a68d88f67380','cdb0337060700bab4fcca32cd4750c0f3de2b9514323ca21457a0a738cb23e99',ARRAY['search_path=pg_catalog','row_security=on']::text[],'bb292b36c6a0704ee9476dc9d0170093420eba24a503c6bfd30fc3c15a5512de','a80b2ec0aa2d2df2d426481d95e8f665a8f388deecdc74def71011f3d8bb0566','1f6d6babd22cecc3ce89846226943509659486392ef7dfe212f4a75d05ac91f3','7347dfcb20e9485dcbf41538187f8d4ec73f0ced74e011492ac8eb71ba97afc3'),
  ('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',true,'923baa922f986747e03eac73816bff10709267575319edaf9ff00b96b363c49e','570e9652dc6ff181230b08a708f7535053ccd59316e447bc8610b3d71b0aef88',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[],'dcf5f8d75fd8215c95882934ba547ceac7633a0bfa6abbbf529168f90c162cda','7ad94585991db40b728799f4c3730af4f1ee6479c77324644660f0f6a7d27b8f','a7f0e1dec1f34ff0c90c9cf8e31bcae470293bb66985e4eb4b0e6e2dd7419e00','85f6351729eaba3ccbcc285808347975dc19f7e5045d972d4356f1d8e8c3918f')
 ) v(firma,es_definer,def_sha,src_sha,config,acl_sha,efectiva_sha,dep_sha,shdep_sha) LOOP
  IF x.def_sha IS NULL OR x.src_sha IS NULL OR x.config IS NULL OR x.acl_sha IS NULL
     OR x.efectiva_sha IS NULL OR x.dep_sha IS NULL OR x.shdep_sha IS NULL THEN
   RAISE EXCEPTION 'CT155: falta preimagen final de %',x.firma USING ERRCODE='55000';
  END IF;
  SELECT q.oid,q.pronamespace,q.proowner,q.prosecdef,q.proconfig,
         pg_get_functiondef(q.oid) AS def,q.prosrc INTO STRICT p
  FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object(
   'grantee',CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,
   'grantor',pg_get_userbyid(a.grantor),'privilege_type',a.privilege_type,
   'is_grantable',a.is_grantable) ORDER BY a.grantee,a.grantor,a.privilege_type,a.is_grantable),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO acl_sha FROM aclexplode(coalesce((SELECT proacl FROM pg_proc WHERE oid=p.oid),
                                         acldefault('f',p.proowner))) a;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object(
   'rolname',r.rolname,'execute',has_function_privilege(r.oid,p.oid,'EXECUTE'),
   'schema_usage',has_schema_privilege(r.oid,p.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO efectiva_sha FROM pg_roles r;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(d) ORDER BY
   d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO dep_sha FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(d) ORDER BY
   d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO shdep_sha FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=p.oid;
  IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
     OR p.prosecdef IS DISTINCT FROM x.es_definer OR p.proconfig IS DISTINCT FROM x.config
     OR encode(sha256(convert_to(p.def,'UTF8')),'hex') IS DISTINCT FROM x.def_sha
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM x.src_sha
     OR acl_sha IS DISTINCT FROM x.acl_sha OR efectiva_sha IS DISTINCT FROM x.efectiva_sha
     OR dep_sha IS DISTINCT FROM x.dep_sha OR shdep_sha IS DISTINCT FROM x.shdep_sha THEN
   RAISE EXCEPTION 'CT155: preimagen incompatible: %',x.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $preimagen_consumidores$;

-- Gramática JSON de material propio: claves ordenadas recursivamente, cadenas
-- escapadas como Go. No confundir jsonb::text con la huella de la intención.
CREATE FUNCTION vec_contratacion_temporal.canon_plan_personal_ct155(v jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
DECLARE r text;
BEGIN
 CASE jsonb_typeof(v)
 WHEN 'object' THEN SELECT '{'||coalesce(string_agg(vec_contratacion_temporal.texto_json_incorporacion_go_v2(e.key)||':'||vec_contratacion_temporal.canon_plan_personal_ct155(e.value),',' ORDER BY e.key COLLATE "C"),'')||'}' INTO r FROM jsonb_each(v) e;
 WHEN 'string' THEN r:=vec_contratacion_temporal.texto_json_incorporacion_go_v2(v#>>'{}');
 WHEN 'number' THEN IF (v#>>'{}')!~'^(0|[1-9][0-9]{0,15})$' OR (v#>>'{}')::numeric>9007199254740991 THEN RAISE EXCEPTION 'CT155: entero inválido' USING ERRCODE='22023'; END IF; r:=v::text;
 WHEN 'boolean' THEN r:=v::text;
 ELSE RAISE EXCEPTION 'CT155: nodo inválido' USING ERRCODE='22023';
 END CASE;RETURN r;
END $f$;
CREATE FUNCTION vec_contratacion_temporal.sesion_plan_personal_ct155()
RETURNS void LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
 OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT155: sesión denegada' USING ERRCODE='42501'; END IF;
END $f$;

CREATE TABLE vec_contratacion_temporal.plan_incorporacion_personal_b2_v1(
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 version_expediente numeric(20,0) NOT NULL CHECK(version_expediente BETWEEN 1 AND 9007199254740991),
 clave_idempotencia uuid NOT NULL,
 plan_ref text NOT NULL UNIQUE,
 plan_version bigint NOT NULL DEFAULT 1 CHECK(plan_version=1),
 recibo_ref text NOT NULL UNIQUE,
 intencion_ref text NOT NULL UNIQUE,
 intencion_recibo_ref text NOT NULL UNIQUE,
 solicitud_personal_ref text NOT NULL UNIQUE,
 idempotencia_personal_uuid uuid NOT NULL UNIQUE,
 material text NOT NULL CHECK(octet_length(material) BETWEEN 2 AND 65536),
 material_json jsonb NOT NULL CHECK(material_json=material::jsonb AND material=vec_contratacion_temporal.canon_plan_personal_ct155(material_json)),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
 actor_ref text NOT NULL,perfil_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE CHECK(consumo_sha256~'^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 outbox_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.outbox_expediente_integral(evento_ref) DEFERRABLE INITIALLY DEFERRED,
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 contrato_json jsonb NOT NULL CHECK(jsonb_typeof(contrato_json)='object'),
 PRIMARY KEY(organizacion_ref,expediente_ref),UNIQUE(organizacion_ref,clave_idempotencia),
 UNIQUE(organizacion_ref,expediente_ref,plan_ref),
 FOREIGN KEY(expediente_ref,version_expediente) REFERENCES vec_contratacion_temporal.expediente_version_integral(expediente_ref,version)
);
CREATE TABLE vec_contratacion_temporal.origen_incorporacion_personal_b2_v1(
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,
 plan_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 protocolo text NOT NULL DEFAULT 'personal_b2_v1' CHECK(protocolo='personal_b2_v1'),
 material text NOT NULL CHECK(octet_length(material) BETWEEN 2 AND 65536),
 material_json jsonb NOT NULL CHECK(material_json=material::jsonb AND material=vec_contratacion_temporal.canon_plan_personal_ct155(material_json)),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
 relacion_ref text NOT NULL,ocupacion_ref text NOT NULL,
 actor_ref text NOT NULL,perfil_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE CHECK(consumo_sha256~'^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 outbox_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.outbox_expediente_integral(evento_ref) DEFERRABLE INITIALLY DEFERRED,
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 recibo_json jsonb NOT NULL CHECK(jsonb_typeof(recibo_json)='object'),
 PRIMARY KEY(organizacion_ref,expediente_ref),UNIQUE(organizacion_ref,expediente_ref,recibo_ref),
 FOREIGN KEY(organizacion_ref,expediente_ref,plan_ref) REFERENCES vec_contratacion_temporal.plan_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,plan_ref)
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_incorporacion_personal_b2_v1','origen_incorporacion_personal_b2_v1'] LOOP
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
 EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true)',t);
 EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',t);
 EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC,vec_contratacion_temporal_ejecutor',t);
 EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;

-- Un solo evento del outbox existente en la transacción del acto CT. No crea
-- tablas de progreso ni duplica el estado administrativo del expediente.
CREATE FUNCTION vec_contratacion_temporal.evento_plan_personal_ct155(org text,exp text,ver numeric,op text,eventoref text,tipo text,recibo text,instante timestamptz)
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE sec bigint; anterior text; payload bytea; huella text;
BEGIN
 SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT sec,anterior FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
 IF sec>=9007199254740991 THEN RAISE EXCEPTION 'CT155: límite outbox' USING ERRCODE='22003'; END IF;
 payload:=convert_to(jsonb_build_object('esquema','vec.ct.incorporacion-personal-b2.evento.v1','protocolo','personal_b2_v1','organizacion_ref',org,'expediente_ref',exp,'plan_ref',op,'recibo_ref',recibo,'registrada_en',instante)::text,'UTF8');
 huella:=encode(sha256(anterior::bytea||payload),'hex');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
 VALUES(eventoref,sec+1,CASE WHEN tipo='ct.incorporacion-personal.v1' THEN 'origen:'||op ELSE op END,exp,ver,tipo,payload,encode(sha256(payload),'hex'),anterior,huella,instante);
 UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral SET secuencia_outbox=sec+1,cabeza_outbox_sha256=huella WHERE control_id;
END $f$;

-- Referencias CT de la propuesta vigente. Para la apertura inicial CT no
-- almacena el hash del registro de Bolsa: se debe acreditar en la fuente Bolsa,
-- nunca sustituirlo por la huella de un sobre de selección.
CREATE FUNCTION vec_contratacion_temporal.antecedentes_plan_personal_ct155(org text,exp text)
RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE a record; p record; v record; r record; e record; rf record; apertura_sha text;propuesta_bolsa text;
BEGIN
 SELECT * INTO a FROM vec_contratacion_temporal.anclaje_vinculo_categoria_rpt_ct154(org,exp);
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT pf.* INTO p FROM vec_contratacion_temporal.aceptacion_vigente_ct124(org,exp) av
 JOIN vec_contratacion_temporal.propuesta_formalizacion pf ON pf.propuesta_ref=av.propuesta_ref;
 IF NOT FOUND OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n WHERE n.organizacion_ref=org AND n.expediente_ref=exp AND n.aceptacion_resolucion_ref=p.resolucion_ref) THEN RETURN NULL; END IF;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE resolucion_ref=p.resolucion_ref;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 WHERE clave_idempotencia=r.seleccion_clave;
 SELECT * INTO v FROM vec_contratacion_temporal.vinculo_categoria_rpt_ct_v1 WHERE organizacion_ref=org AND expediente_ref=exp ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT coalesce(x.continuacion_recibo#>>'{ReciboBolsa,RegistroSHA256}',''),x.continuacion_recibo#>>'{ReciboBolsa,PropuestaRef}' INTO apertura_sha,propuesta_bolsa FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh x
 WHERE x.organizacion_ref=org AND x.expediente_ref=exp AND x.continuacion_recibo#>>'{ReciboBolsa,OperacionRef}'=p.aceptacion_bolsa_json->>'AperturaOperacionRef' LIMIT 1;
 SELECT * INTO rf FROM vec_contratacion_temporal.resolucion_formalizacion WHERE organizacion_ref=org AND expediente_ref=exp AND propuesta_ref=p.propuesta_ref;
 RETURN jsonb_build_object('organizacion_ref',org,'expediente_ref',exp,'documento_ref',coalesce(rf.recibo_json->>'DocumentoRef',''),'documento_sha256',coalesce(rf.recibo_json->>'DocumentoSHA256',''),'documento_version',coalesce((rf.recibo_json->>'DocumentoVersion')::numeric,0),'version_expediente',a.version_expediente,
 'analisis_version',a.analisis_version,'analisis_recibo_ref',a.analisis_recibo_ref,'analisis_sha256',a.analisis_huella_sha256,'categoria_ref',a.categoria_ref,
 'propuesta_recibo_ref',p.recibo_ref,'aceptacion_ref',p.resolucion_ref,'aceptacion_recibo_ref',r.recibo_ref,
 'bolsa',jsonb_build_object('unidad_ref',(SELECT vv.agregado_json#>>'{asignacion,unidad_ref}' FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_version_integral vv ON vv.expediente_ref=aa.expediente_ref AND vv.version=aa.version WHERE aa.expediente_ref=exp),'categoria_ref',a.categoria_ref,
 'necesidad_ref',e.recibo_json#>>'{necesidad,referencia}','aceptacion_operacion_ref',p.aceptacion_bolsa_json->>'OperacionRef','aceptacion_registro_sha256',p.aceptacion_bolsa_json->>'RegistroSHA256',
 'apertura_operacion_ref',p.aceptacion_bolsa_json->>'AperturaOperacionRef','apertura_registro_sha256',coalesce(apertura_sha,''),'llamamiento_ref',p.llamamiento_ref,'propuesta_ref',coalesce(propuesta_bolsa,e.recibo_json#>>'{propuesta,referencia}')),
 'vinculo',v.recibo_json->'vinculo');
END $f$;

CREATE FUNCTION vec_contratacion_temporal.validar_plan_personal_ct155(p jsonb,s jsonb)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; nodo jsonb; desde date; hasta date;
BEGIN
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','bolsa','persona_ref','persona_version','persona_fuente','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','fuente_organizacion','fuente_plantilla','fuente_rpt','version_plaza_ref','version_puesto_ref','revision_plaza','revision_puesto','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','catalogo_rpt_version','catalogo_rpt_sha256','categoria_ref','vinculo_revision','vinculo_recibo_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256','ejercicio_sintetico']) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT155: plan incompleto' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','analisis_recibo_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','persona_ref','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','version_plaza_ref','version_puesto_ref','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','categoria_ref','vinculo_recibo_ref','documento_ref'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'CT155: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['version_expediente','analisis_version','persona_version','revision_plaza','revision_puesto','catalogo_rpt_version','vinculo_revision'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'number' OR p->>k!~'^[1-9][0-9]{0,15}$' OR (p->>k)::numeric>9007199254740991 THEN RAISE EXCEPTION 'CT155: versión inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['analisis_sha256','catalogo_rpt_sha256','documento_sha256'] LOOP
 IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' OR p->>k!~'^[0-9a-f]{64}$' OR p->>k=repeat('0',64) THEN RAISE EXCEPTION 'CT155: huella inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['persona_fuente'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version','sha256']) IS NOT TRUE
 OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,15}$'
 OR (nodo->>'version')::numeric>9007199254740991 OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64)
 THEN RAISE EXCEPTION 'CT155: fuente inválida' USING ERRCODE='22023'; END IF;
 END LOOP;

 nodo:=p->'fuente_organizacion';
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'sha256'!~'^[0-9a-f]{64}$' OR nodo->>'sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: fuente organización inválida' USING ERRCODE='22023';END IF;
 FOREACH k IN ARRAY ARRAY['fuente_plantilla','fuente_rpt'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','revision','fuente_ref','fuente_sha256']) IS NOT TRUE OR nodo->>'ref'!~'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' OR nodo->>'revision'!~'^[1-9][0-9]{0,15}$' OR (nodo->>'revision')::numeric>9007199254740991 OR nodo->>'fuente_ref'!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR nodo->>'fuente_sha256'!~'^[0-9a-f]{64}$' OR nodo->>'fuente_sha256'=repeat('0',64) THEN RAISE EXCEPTION 'CT155: instrumento inválido' USING ERRCODE='22023';END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['regimen','modalidad'] LOOP
 nodo:=p->k;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(nodo,ARRAY['ref','version']) IS NOT TRUE OR nodo->>'ref'!~'^[a-z][a-z0-9_:-]{2,159}$' OR nodo->>'version'!~'^[1-9][0-9]{0,9}$' OR (nodo->>'version')::numeric>2147483647
 THEN RAISE EXCEPTION 'CT155: catálogo inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'bolsa',ARRAY['unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','apertura_registro_sha256','llamamiento_ref','propuesta_ref']) IS NOT TRUE
 OR EXISTS(SELECT 1 FROM jsonb_each_text(p->'bolsa') e WHERE (CASE WHEN e.key LIKE '%sha256' THEN e.value!~'^[0-9a-f]{64}$' OR e.value=repeat('0',64) ELSE e.value!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' END))
 OR p->>'categoria_ref' IS DISTINCT FROM p#>>'{bolsa,categoria_ref}'
 OR jsonb_typeof(p->'ejercicio_sintetico') IS DISTINCT FROM 'boolean'
 OR p->>'motivo_clave'!~'^[a-z][a-z0-9_.-]{1,79}$'
 OR jsonb_typeof(p->'clase_ocupacion') IS DISTINCT FROM 'string'
 OR p->>'clase_ocupacion' !~ '^[a-z][a-z0-9_]{0,63}$'
 OR p->>'desde'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR (p->>'hasta'<>'' AND p->>'hasta'!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
 THEN RAISE EXCEPTION 'CT155: intención inválida' USING ERRCODE='22023'; END IF;
 BEGIN desde:=(p->>'desde')::date;hasta:=nullif(p->>'hasta','')::date;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT155: fecha inválida' USING ERRCODE='22023'; END;
 IF NOT isfinite(desde) OR to_char(desde,'YYYY-MM-DD')<>p->>'desde' OR (hasta IS NOT NULL AND (NOT isfinite(hasta) OR hasta<=desde OR to_char(hasta,'YYYY-MM-DD')<>p->>'hasta'))
 OR (p->>'analisis_version')::numeric>(p->>'version_expediente')::numeric
 OR (SELECT jsonb_object_agg(x.clave,p->x.clave) FROM unnest(ARRAY['organizacion_ref','expediente_ref','version_expediente','puesto_ref','plaza_ref','regimen','modalidad','clase_ocupacion','desde','hasta','motivo_clave','documento_ref','documento_sha256']) AS x(clave))
 IS DISTINCT FROM s-ARRAY['clave_idempotencia','version_plantilla_ref','version_rpt_ref']
 OR p#>>'{fuente_plantilla,ref}' IS DISTINCT FROM s->>'version_plantilla_ref' OR p#>>'{fuente_rpt,ref}' IS DISTINCT FROM s->>'version_rpt_ref'
 THEN RAISE EXCEPTION 'CT155: selección divergente' USING ERRCODE='22023'; END IF;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.operacion_plan_personal_ct155(op text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $f$
DECLARE m jsonb;s jsonb;p jsonb;h jsonb;a jsonb; ct record; d jsonb; previa record; plan record; version_actual numeric;
 org text;exp text;unidad text;actual_unidad text;accion text;sha text;ahora timestamptz;recibo text;eventoref text;planref text;intencion text;intrec text;solpersonal text;idem uuid;contrato jsonb;salida jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.sesion_plan_personal_ct155();
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536 THEN RAISE EXCEPTION 'CT155: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT155: material inválido' USING ERRCODE='22023'; END;
 IF p_material IS DISTINCT FROM vec_contratacion_temporal.canon_plan_personal_ct155(m) THEN RAISE EXCEPTION 'CT155: material no canónico' USING ERRCODE='22023'; END IF;
 s:=CASE WHEN op='registrar' THEN m->'solicitud' ELSE m END;org:=s->>'organizacion_ref';exp:=s->>'expediente_ref';
 unidad:=CASE op WHEN 'registrar' THEN m#>>'{material,unidad_ct_ref}' WHEN 'confirmar' THEN m->>'unidad_ct_ref' ELSE m->>'unidad_ref' END;
 accion:=CASE op WHEN 'registrar' THEN 'contratacion_temporal.incorporacion_personal.plan.registrar' WHEN 'confirmar' THEN 'contratacion_temporal.incorporacion_personal.origen.confirmar' WHEN 'leer' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' WHEN 'origen' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' WHEN 'antecedentes' THEN 'contratacion_temporal.incorporacion_personal.plan.consultar' END;
 IF accion IS NULL OR d->>'accion' IS DISTINCT FROM accion THEN RAISE EXCEPTION 'CT155: operación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT ct FROM vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT vv.agregado_json#>>'{asignacion,unidad_ref}' INTO actual_unidad FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_version_integral vv ON vv.expediente_ref=aa.expediente_ref AND vv.version=aa.version JOIN vec_contratacion_temporal.expediente_alta ea ON ea.expediente_ref=aa.expediente_ref WHERE aa.expediente_ref=exp AND ea.organizacion_ref=org;
 IF actual_unidad IS NULL OR unidad IS DISTINCT FROM actual_unidad THEN RAISE EXCEPTION 'CT155: unidad del expediente divergente' USING ERRCODE='42501';END IF;
 -- El permiso nuevo y su auditoría se consumen también para recuperar.
 PERFORM set_config('vec.ct115.organizacion_ref',org,true);PERFORM set_config('vec.ct115.expediente_ref',exp,true);
 IF op IN ('leer','origen','antecedentes') THEN
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','unidad_ref']) IS NOT TRUE THEN RAISE EXCEPTION 'CT155: selector inválido' USING ERRCODE='22023'; END IF;
 IF op='antecedentes' THEN RETURN vec_contratacion_temporal.antecedentes_plan_personal_ct155(org,exp); END IF;
 IF op='leer' THEN SELECT contrato_json INTO salida FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 ELSE SELECT recibo_json INTO salida FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;END IF;
 RETURN salida;END IF;
 -- Un bloqueo por expediente serializa plan y origen; las restricciones UUID
 -- impiden reutilizar la misma intención para otro expediente de organización.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec.ct155:'||org||':'||exp,0));
 SELECT aa.version INTO version_actual FROM vec_contratacion_temporal.expediente_integral_actual aa JOIN vec_contratacion_temporal.expediente_alta ea ON ea.expediente_ref=aa.expediente_ref WHERE ea.organizacion_ref=org AND aa.expediente_ref=exp FOR UPDATE OF aa;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT155: expediente ausente' USING ERRCODE='55000'; END IF;
 sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');ahora:=date_trunc('microseconds',clock_timestamp());
 IF op='registrar' THEN
 p:=m->'material';
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['solicitud','material']) IS NOT TRUE OR s->>'clave_idempotencia'!~'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' OR s->>'clave_idempotencia'='00000000-0000-0000-0000-000000000000'
 THEN RAISE EXCEPTION 'CT155: intención inválida' USING ERRCODE='22023'; END IF;
 PERFORM vec_contratacion_temporal.validar_plan_personal_ct155(p,s);
 SELECT * INTO previa FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND clave_idempotencia=(s->>'clave_idempotencia')::uuid;
 IF FOUND THEN IF previa.material_sha256 IS DISTINCT FROM sha OR previa.expediente_ref IS DISTINCT FROM exp THEN RAISE EXCEPTION 'CT155: idempotencia reutilizada' USING ERRCODE='55000'; END IF;RETURN previa.contrato_json; END IF;
 IF version_actual IS DISTINCT FROM (p->>'version_expediente')::numeric OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 THEN RAISE EXCEPTION 'CT155: versión u origen en conflicto' USING ERRCODE='55000'; END IF;
 a:=vec_contratacion_temporal.antecedentes_plan_personal_ct155(org,exp);
 IF a IS NULL OR (SELECT jsonb_object_agg(k,p->k) FROM unnest(ARRAY['version_expediente','analisis_version','analisis_recibo_ref','analisis_sha256','categoria_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref']) k)
 IS DISTINCT FROM a-ARRAY['organizacion_ref','expediente_ref','bolsa','vinculo','documento_ref','documento_sha256','documento_version']
 OR p->'bolsa'-'apertura_registro_sha256' IS DISTINCT FROM a->'bolsa'-'apertura_registro_sha256'
 OR (a#>>'{bolsa,apertura_registro_sha256}'<>'' AND p#>>'{bolsa,apertura_registro_sha256}' IS DISTINCT FROM a#>>'{bolsa,apertura_registro_sha256}')
 OR p->>'documento_ref' IS DISTINCT FROM a->>'documento_ref' OR p->>'documento_sha256' IS DISTINCT FROM a->>'documento_sha256'
 OR p->>'vinculo_revision' IS DISTINCT FROM a#>>'{vinculo,revision}' OR p->>'vinculo_recibo_ref' IS DISTINCT FROM a#>>'{vinculo,recibo_ref}'
 OR p->>'catalogo_rpt_id' IS DISTINCT FROM a#>>'{vinculo,catalogo_id}' OR p->>'catalogo_rpt_modulo' IS DISTINCT FROM a#>>'{vinculo,modulo_id}' OR p->>'catalogo_rpt_version' IS DISTINCT FROM a#>>'{vinculo,catalogo_version}' OR p->>'catalogo_rpt_sha256' IS DISTINCT FROM a#>>'{vinculo,catalogo_huella_sha256}'
 THEN RAISE EXCEPTION 'CT155: antecedentes divergentes' USING ERRCODE='55000'; END IF;
 planref:='plan_ct_b2_'||replace(gen_random_uuid()::text,'-','');recibo:='recibo_ct_b2_'||replace(gen_random_uuid()::text,'-','');intencion:='intencion_ct_b2_'||replace(gen_random_uuid()::text,'-','');intrec:='recibo_intencion_ct_b2_'||replace(gen_random_uuid()::text,'-','');solpersonal:='solicitud_ct_b2_'||replace(gen_random_uuid()::text,'-','');idem:=gen_random_uuid();eventoref:='evento:plan-personal-b2:'||gen_random_uuid()::text;
 contrato:=jsonb_build_object('protocolo','personal_b2_v1','plan_ref',planref,'plan_version',1,'plan_recibo_ref',recibo,'plan_sha256',sha,'intencion_ref',intencion,'intencion_recibo_ref',intrec,'intencion_version',1,'solicitud_personal_ref',solpersonal,'idempotencia_personal_uuid',idem::text,'material',p,'solicitud',s,'registrado_en',to_char(ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(org,exp,version_actual,planref,eventoref,'ct.plan-incorporacion-personal.v1',recibo,ahora);
 INSERT INTO vec_contratacion_temporal.plan_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,version_expediente,clave_idempotencia,plan_ref,recibo_ref,intencion_ref,intencion_recibo_ref,solicitud_personal_ref,idempotencia_personal_uuid,material,material_json,material_sha256,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,contrato_json)
 VALUES(org,exp,version_actual,(s->>'clave_idempotencia')::uuid,planref,recibo,intencion,intrec,solpersonal,idem,p_material,m,sha,d->>'principal_id',d->>'perfil_activo_ref',ct.decision_ref,ct.consumo_huella_sha256,ct.auditoria_ref,eventoref,ahora,contrato);
 RETURN contrato;
 END IF;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','plan_ref','plan_version','plan_sha256','hechos']) IS NOT TRUE THEN RAISE EXCEPTION 'CT155: confirmación inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO plan FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 IF NOT FOUND OR plan.plan_ref IS DISTINCT FROM m->>'plan_ref' OR m->>'plan_version'<>'1' OR plan.material_sha256 IS DISTINCT FROM m->>'plan_sha256' THEN RAISE EXCEPTION 'CT155: plan divergente' USING ERRCODE='55000'; END IF;
 h:=vec_personal.probar_origen_incorporacion_plan_v1(org,exp,plan.plan_ref,1,plan.material_sha256,m->'hechos');
 IF h IS NULL OR h IS DISTINCT FROM m->'hechos' THEN RAISE EXCEPTION 'CT155: Personal no acredita el origen' USING ERRCODE='55000'; END IF;
 SELECT * INTO previa FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE organizacion_ref=org AND expediente_ref=exp;
 IF FOUND THEN IF previa.material_sha256 IS DISTINCT FROM sha THEN RAISE EXCEPTION 'CT155: origen reutilizado' USING ERRCODE='55000'; END IF;RETURN previa.recibo_json;END IF;
 IF version_actual IS DISTINCT FROM plan.version_expediente OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp)
 THEN RAISE EXCEPTION 'CT155: versión u origen en conflicto' USING ERRCODE='55000'; END IF;
 recibo:='recibo:incorporacion-personal-b2:'||gen_random_uuid()::text;eventoref:='evento:incorporacion-personal-b2:'||gen_random_uuid()::text;
 salida:=jsonb_build_object('protocolo','personal_b2_v1','confirmacion',m,'recibo_ref',recibo,'auditoria_ref',ct.auditoria_ref,'outbox_ref',eventoref,'registrado_en',to_char(ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'firma_oficial',false,'eficacia_administrativa',false);
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(org,exp,version_actual,plan.plan_ref,eventoref,'ct.incorporacion-personal.v1',recibo,ahora);
 INSERT INTO vec_contratacion_temporal.origen_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,plan_ref,recibo_ref,material,material_json,material_sha256,relacion_ref,ocupacion_ref,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,recibo_json)
 VALUES(org,exp,plan.plan_ref,recibo,p_material,m,sha,h->>'relacion_ref',h->>'ocupacion_ref',d->>'principal_id',d->>'perfil_activo_ref',ct.decision_ref,ct.consumo_huella_sha256,ct.auditoria_ref,eventoref,ahora,salida);
 RETURN salida;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.registrar_plan_nominal_b2_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$ SELECT vec_contratacion_temporal.operacion_plan_personal_ct155('registrar',p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;

CREATE FUNCTION vec_contratacion_temporal.leer_plan_nominal_b2_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$ SELECT vec_contratacion_temporal.operacion_plan_personal_ct155('leer',p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_origen_incorporacion_b2_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$ SELECT vec_contratacion_temporal.operacion_plan_personal_ct155('confirmar',p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;

CREATE FUNCTION vec_contratacion_temporal.leer_origen_incorporacion_b2_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$ SELECT vec_contratacion_temporal.operacion_plan_personal_ct155('origen',p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;

CREATE FUNCTION vec_contratacion_temporal.leer_antecedentes_plan_b2_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security='on' AS $f$ SELECT vec_contratacion_temporal.operacion_plan_personal_ct155('antecedentes',p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) $f$;

-- Proyección neutral del origen exacto. Los dos protocolos mantienen sus
-- recibos originales; B2 no crea CT75 ni una raíz de seguimiento de ejercicio.
CREATE FUNCTION vec_contratacion_temporal.origen_incorporacion_neutral_ct155(org text,exp text,recibo text)
RETURNS TABLE(protocolo text,organizacion_ref text,expediente_ref text,recibo_ref text,inicio date,hasta date,llamamiento_ref text,relacion_ref text,seguimiento_ref text)
LANGUAGE plpgsql STABLE SET search_path=pg_catalog SET row_security='on' AS $f$
DECLARE total integer;
BEGIN
 SELECT count(*) INTO total FROM (
 SELECT i.recibo_ref FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
 UNION ALL SELECT i.recibo_ref FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)) x;
 IF total>1 THEN RAISE EXCEPTION 'CT155: origen ambiguo' USING ERRCODE='55000'; END IF;
 RETURN QUERY
 SELECT 'ejercicio_v2'::text,i.organizacion_ref,i.expediente_ref,i.recibo_ref,vec_contratacion_temporal.inicio_incorporacion_ct115(i.material_json),
 ((i.material_json#>>'{Confirmacion,PeriodoIncorporacion,hasta}')::timestamptz AT TIME ZONE 'Europe/Madrid')::date,
 (SELECT p.llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion p WHERE p.organizacion_ref=org AND p.expediente_ref=exp ORDER BY p.confirmada_en DESC,p.propuesta_ref COLLATE "C" DESC LIMIT 1),
 r.relacion_ref,i.seguimiento_ref
 FROM vec_contratacion_temporal.incorporacion_registro_v2 i
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref AND r.organizacion_ref=i.organizacion_ref AND r.expediente_ref=i.expediente_ref AND r.relacion_ref=i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'
 WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo)
 UNION ALL
 SELECT i.protocolo,i.organizacion_ref,i.expediente_ref,i.recibo_ref,(p.material_json#>>'{material,desde}')::date,nullif(p.material_json#>>'{material,hasta}','')::date,
 p.material_json#>>'{material,bolsa,llamamiento_ref}',i.relacion_ref,NULL::text
 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
 JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
 WHERE i.organizacion_ref=org AND i.expediente_ref=exp AND (recibo IS NULL OR i.recibo_ref=recibo);
END $f$;

-- FKs nativas por protocolo: se conserva la FK de CT75 y cada fila antigua.
-- Las columnas nuevas no reescriben la historia. El XOR hace imposible enlazar
-- dos orígenes o guardar un origen sin padre.
DO $fk$
DECLARE t text; col text;
BEGIN
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','confirmacion_ginpix_v1','reincorporacion_titular_v1'] LOOP
 col:=CASE WHEN t='reincorporacion_titular_v1' THEN 'incorporacion_recibo_ref' ELSE 'incorporacion_ref' END;
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ALTER COLUMN %I DROP NOT NULL, ADD COLUMN incorporacion_b2_ref text, ADD COLUMN incorporacion_protocolo text NOT NULL DEFAULT ''ejercicio_v2''',t,col);
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ADD CONSTRAINT origen_b2_ct155_fk FOREIGN KEY(organizacion_ref,expediente_ref,incorporacion_b2_ref) REFERENCES vec_contratacion_temporal.origen_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,recibo_ref) MATCH SIMPLE',t);
 EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ADD CONSTRAINT origen_xor_ct155 CHECK((%I IS NOT NULL)<>(incorporacion_b2_ref IS NOT NULL) AND ((incorporacion_protocolo=''ejercicio_v2'' AND %I IS NOT NULL) OR (incorporacion_protocolo=''personal_b2_v1'' AND incorporacion_b2_ref IS NOT NULL)))',t,col,col);
 END LOOP;
 ALTER TABLE vec_contratacion_temporal.confirmacion_ginpix_v1 ADD CONSTRAINT ginpix_origen_b2_ct155_unico UNIQUE(incorporacion_b2_ref);
END $fk$;
CREATE FUNCTION vec_contratacion_temporal.enlazar_origen_personal_ct155()
RETURNS trigger LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE ref text;
BEGIN
 IF TG_TABLE_NAME='reincorporacion_titular_v1' THEN ref:=NEW.incorporacion_recibo_ref;ELSE ref:=NEW.incorporacion_ref;END IF;
 -- El recibo B2 tiene un espacio propio. La FK compuesta valida el padre y
 -- organización/expediente: el prefijo sólo selecciona la columna, no autoriza.
 IF ref LIKE 'recibo:incorporacion-personal-b2:%' THEN
 IF NEW.incorporacion_b2_ref IS NOT NULL THEN RAISE EXCEPTION 'CT155: dos orígenes' USING ERRCODE='23514'; END IF;
 NEW.incorporacion_b2_ref:=ref;NEW.incorporacion_protocolo:='personal_b2_v1';
 IF TG_TABLE_NAME='reincorporacion_titular_v1' THEN NEW.incorporacion_recibo_ref:=NULL;ELSE NEW.incorporacion_ref:=NULL;END IF;
 END IF;RETURN NEW;
END $f$;
DO $triggers$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','confirmacion_ginpix_v1','reincorporacion_titular_v1'] LOOP
 EXECUTE format('CREATE TRIGGER enlazar_origen_personal_ct155 BEFORE INSERT ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.enlazar_origen_personal_ct155()',t);
 END LOOP;
END $triggers$;

DO $consumidores$
DECLARE x record;p record;def text;meta jsonb;paso jsonb;config_esperada text[];
        dependencias jsonb;compartidas jsonb;permisos_efectivos jsonb;
BEGIN
 FOR x IN SELECT firma,jsonb_agg(jsonb_build_object('anterior',anterior,'nuevo',nuevo)) AS pasos FROM (VALUES
('incorporacion_expediente_ct115(text,text)',$old$    SELECT r.recibo_ref, vec_contratacion_temporal.inicio_incorporacion_ct115(r.material_json),
           (SELECT pf.llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion pf
             WHERE pf.organizacion_ref=p_organizacion AND pf.expediente_ref=p_expediente
             ORDER BY pf.confirmada_en DESC, pf.propuesta_ref COLLATE "C" DESC LIMIT 1)
      FROM vec_contratacion_temporal.incorporacion_registro_v2 r
     WHERE r.organizacion_ref=p_organizacion AND r.expediente_ref=p_expediente
     ORDER BY r.registrada_en DESC, r.recibo_ref COLLATE "C" DESC LIMIT 1$old$,$new$    SELECT n.recibo_ref,n.inicio,n.llamamiento_ref FROM vec_contratacion_temporal.origen_incorporacion_neutral_ct155(p_organizacion,p_expediente,NULL) n$new$),
('resultado_cese_ct115(vec_contratacion_temporal.cese_nombramiento_v1)',$old$r.incorporacion_ref,'inicio'$old$,$new$coalesce(r.incorporacion_ref,r.incorporacion_b2_ref),'inicio'$new$),
('resultado_ginpix_ct124(vec_contratacion_temporal.confirmacion_ginpix_v1)',$old$r.incorporacion_ref,'inicio'$old$,$new$coalesce(r.incorporacion_ref,r.incorporacion_b2_ref),'inicio'$new$),
('ginpix_confirmado_ct124(text,text)',$old$g.incorporacion_ref=(SELECT$old$,$new$coalesce(g.incorporacion_ref,g.incorporacion_b2_ref)=(SELECT$new$),
('confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$WHERE incorporacion_ref=v_inc.recibo_ref$old$,$new$WHERE coalesce(incorporacion_ref,incorporacion_b2_ref)=v_inc.recibo_ref$new$),
('origen_reincorporacion_ct130(jsonb)',$old$DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; i vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;$old$,$new$DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; i record;$new$),
('origen_reincorporacion_ct130(jsonb)',$old$ SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE recibo_ref=c.incorporacion_ref;
 IF NOT FOUND THEN estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 SELECT * INTO r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=i.seguimiento_ref;
 IF NOT FOUND OR i.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR i.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR r.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR r.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}' IS DISTINCT FROM r.relacion_ref
    OR m->>'relacion_ref' IS DISTINCT FROM r.relacion_ref
    OR r.relacion_ref IS NULL THEN
  estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 relacion_ref:=r.relacion_ref; incorporacion_recibo_ref:=i.recibo_ref;$old$,$new$ SELECT * INTO i FROM vec_contratacion_temporal.origen_incorporacion_neutral_ct155(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref));
 IF NOT FOUND OR i.protocolo IS DISTINCT FROM c.incorporacion_protocolo OR i.relacion_ref IS DISTINCT FROM m->>'relacion_ref' OR i.relacion_ref IS NULL THEN
  estado:='cese_no_coincide';RETURN NEXT;RETURN;END IF;
 relacion_ref:=i.relacion_ref;incorporacion_recibo_ref:=i.recibo_ref;$new$),
('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$ JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref$old$,$new$ CROSS JOIN LATERAL vec_contratacion_temporal.origen_incorporacion_neutral_ct155(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref)) i$new$),
('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',$old$   AND r.organizacion_ref=c.organizacion_ref AND r.expediente_ref=c.expediente_ref
   AND i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'=r.relacion_ref
   AND r.relacion_ref=p_material->>'relacion_ref'$old$,$new$   AND i.protocolo=c.incorporacion_protocolo
   AND i.relacion_ref=p_material->>'relacion_ref'$new$)
 ) v(firma,anterior,nuevo) GROUP BY firma LOOP
 SELECT pg_get_functiondef(oid) AS def,to_jsonb(q)-'prosrc' AS meta,
        q.proconfig,q.prosecdef INTO STRICT p FROM pg_proc q
 WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma) AND proowner=current_user::regrole;
 meta:=p.meta;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
  d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO dependencias FROM pg_depend d WHERE d.classid='pg_proc'::regclass
  AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
  d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass
  AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'rolname',r.rolname,'execute',has_function_privilege(r.oid,q.oid,'EXECUTE'),
  'schema_usage',has_schema_privilege(r.oid,q.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)
 INTO permisos_efectivos FROM pg_roles r CROSS JOIN pg_proc q
 WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 IF x.firma IN (
  'confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') THEN
  IF NOT p.prosecdef OR p.proconfig IS NULL
     OR (SELECT count(*) FROM unnest(p.proconfig) c WHERE c LIKE 'search_path=%')<>1
     OR NOT p.proconfig @> ARRAY['search_path=pg_catalog'] THEN
   RAISE EXCEPTION 'CT155: entorno heredado incompatible: %',x.firma USING ERRCODE='55000';
  END IF;
  config_esperada:=array_replace(p.proconfig,'search_path=pg_catalog','search_path=pg_catalog, pg_temp');
  meta:=jsonb_set(meta,'{proconfig}',to_jsonb(config_esperada));
  EXECUTE format('ALTER FUNCTION %s SET search_path TO pg_catalog, pg_temp',
                 to_regprocedure('vec_contratacion_temporal.'||x.firma));
  SELECT pg_get_functiondef(q.oid) INTO STRICT def FROM pg_proc q
   WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)
     AND q.proconfig IS NOT DISTINCT FROM config_esperada;
 ELSE def:=p.def;
 END IF;
 FOR paso IN SELECT value FROM jsonb_array_elements(x.pasos) LOOP
 IF length(def)-length(replace(def,paso->>'anterior',''))<>length(paso->>'anterior') THEN RAISE EXCEPTION 'CT155: preimagen incompatible: %',x.firma USING ERRCODE='55000';END IF;
 def:=replace(def,paso->>'anterior',paso->>'nuevo');
 END LOOP;
 EXECUTE def;
 IF (SELECT pg_get_functiondef(oid) FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM def
 OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
     d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass
       AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM dependencias
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
     d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass
       AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM compartidas
 OR (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'rolname',r.rolname,'execute',has_function_privilege(r.oid,q.oid,'EXECUTE'),
     'schema_usage',has_schema_privilege(r.oid,q.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)
     FROM pg_roles r CROSS JOIN pg_proc q
     WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM permisos_efectivos
 THEN RAISE EXCEPTION 'CT155: metadatos/dependencias alterados: %',x.firma USING ERRCODE='55000';END IF;
 END LOOP;
END $consumidores$;

-- Blindaje ACL de todas las funciones nuevas, incluso ante ACL por defecto.
DO $acl$
DECLARE f record;g record;dest text;
BEGIN
 FOR f IN SELECT p.oid,p.proname,p.proowner FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE '%ct155' OR p.proname IN ('registrar_plan_nominal_b2_v1','leer_plan_nominal_b2_v1','confirmar_origen_incorporacion_b2_v1','leer_origen_incorporacion_b2_v1','leer_antecedentes_plan_b2_v1')) LOOP
 FOR g IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f.oid AND x.grantee<>f.proowner LOOP
 dest:=CASE WHEN g.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(g.grantee)) END;
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f.oid::regprocedure,dest);
 END LOOP;
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
 IF f.proname IN ('registrar_plan_nominal_b2_v1','leer_plan_nominal_b2_v1','confirmar_origen_incorporacion_b2_v1','leer_origen_incorporacion_b2_v1','leer_antecedentes_plan_b2_v1') THEN EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f.oid::regprocedure);END IF;
 END LOOP;
END $acl$;
COMMIT;
