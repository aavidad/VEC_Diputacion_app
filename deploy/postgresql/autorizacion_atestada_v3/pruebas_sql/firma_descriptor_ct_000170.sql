\set ON_ERROR_STOP on
-- AD170: estructura y negativas reales; no concede ni sustituye V3.
-- Los sobres incompletos sólo aíslan rechazo ANTES del consumidor común.
-- Favorable/replay requieren otra puerta con sesión y capacidad acreditadas.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario AND p.prosecdef AND p.provolatile='v'
   AND p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'])
  OR NOT has_function_privilege(ct,f,'EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(propietario,ct) OR a.is_grantable)) THEN
  RAISE EXCEPTION 'AD170 ACL/configuración divergente' USING ERRCODE='55000'; END IF;
END $acl$;
CREATE TEMP TABLE ad170_preimagen AS SELECT
 pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) AS nucleo,
 pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) AS ad162,
 (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) AS metadatos;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SELECT set_config('ad170.auditorias',(SELECT count(*)::text FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),true);
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $negativas$
DECLARE descriptor jsonb; original text; cambiado text; solicitud text; decision bytea; capacidad bytea;
 ctx text; material_h text; descriptor_h text; via text; accion text; audiencia text; tipo text;
 recurso text; actor text; caso record; codigo text; diagnostico text;
BEGIN
 descriptor:=jsonb_build_object('esquema','vec.competencia-firmante.constructor-ct.v1',
  'certificado_der_sha256',repeat('a',64),
  'seleccion',jsonb_build_object('perfil_esperado_ref','perfil:ct:jefatura',
    'perfil_activo_ref','prf_ad170_firmante','rol_id','ct_cargo_jefatura',
    'cargo_ref','cargo:ad170:prueba','enlace_ejercicio_ref','enlace:ad170:prueba'),
  'recurso',jsonb_build_object('organizacion_ref','organizacion:ad170:prueba','unidad_ref','unidad:ad170:prueba',
    'expediente_ref','expediente:ad170:prueba','documento_ref','documento:ad170:original',
    'recurso_autorizable_ref','documento:ad170:original','modulo_id','contratacion_temporal','tipo_recurso','documento',
    'recurso_contexto_sha256',repeat('b',64),
    'original',jsonb_build_object('referencia','documento:ad170:original','version',1,'huella_sha256',repeat('c',64)),
    'pdf_raiz_sha256',repeat('c',64),
    'firmado',jsonb_build_object('referencia','documento:ad170:firmado','version',1,'huella_sha256',repeat('d',64)),
    'pdf_firmado_sha256',repeat('d',64),'numero_firmas',1,'entrada_revision',NULL),
  'accion','contratacion_temporal.documento.firmar','finalidad','formalizar',
  'motivo',jsonb_build_object('catalogo_id','vec.motivos','catalogo_version',1,'catalogo_huella_sha256',repeat('e',64),'entrada_clave','firma'),
  'circuito',jsonb_build_object('referencia','vec.circuito:1','version',1,'huella_sha256',repeat('f',64)),
  'paso_ref','vec.circuito:1:paso1','paso_orden',1,'fecha_historica',NULL);
 original:=descriptor::text;
 FOREACH via IN ARRAY ARRAY['certificado_vec','portafirmas_registro_rrhh'] LOOP
  IF via='certificado_vec' THEN
   accion:='contratacion_temporal.documento.firma_vec.registrar'; audiencia:='vec_contratacion_temporal.firma_vec.v2';
   tipo:='firma_vec_documento_contratacion_temporal'; recurso:='operacion-firma-vec-ct:clave-ad170-00000001';
   actor:='per_ad170_firmante';
  ELSE
   accion:='contratacion_temporal.documento.firma_externa.registrar'; audiencia:='vec_contratacion_temporal.firma_externa.v2';
   tipo:='firma_externa_documento_contratacion_temporal'; recurso:='operacion-firma-externa-ct:clave-ad170-00000001';
   actor:='per_ad170_registrador';
  END IF;
  solicitud:=jsonb_build_object('Via',via,'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v2',
   'OrganizacionRef','organizacion:ad170:prueba','ClaveIdempotencia','clave-ad170-00000001',
   'FirmantePrincipalRef','per_ad170_firmante','CertificadoHuella',repeat('a',64),'PerfilActivoOperadorRef','prf_ad170_operador')::text;
  material_h:=encode(sha256(convert_to(solicitud,'UTF8')),'hex');
  descriptor_h:=encode(sha256(convert_to(original,'UTF8')),'hex');
  ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:ad170:prueba"},"atributos":{"descriptor_firma_sha256":"'||descriptor_h||'","material_sha256":"'||material_h||'"}}','UTF8')),'hex');
  decision:=convert_to(jsonb_build_object('accion',accion,'modulo_id','contratacion_temporal','tipo_recurso',tipo,
   'finalidad','gestionar_contratacion_temporal','recurso_ref',recurso,'contexto_recurso_huella_sha256',ctx,
   'principal_id',actor,'perfil_activo_ref','prf_ad170_operador','version_rol_ref','rol:firma_externa_registro_ct_desarrollo:v1',
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),
   'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8');
  capacidad:=convert_to(jsonb_build_object('operacion',accion,'audiencia_consumo',audiencia,'efecto_ref',recurso,
   'huella_efecto_sha256',ctx,'huella_decision_sha256',encode(sha256(decision),'hex'))::text,'UTF8');
  -- La decisión/capacidad no cambian. Otra validación no puede hacer pasar
  -- estas regresiones: se exige el rechazo concreto del wrapper AD170.
  FOR caso IN SELECT * FROM (VALUES
   ('contexto documental',jsonb_set(descriptor,'{recurso,recurso_contexto_sha256}',to_jsonb(repeat('1',64)))::text,'42501'),
   ('acción competencial',jsonb_set(descriptor,'{accion}','"otra_accion"'::jsonb)::text,'42501'),
   ('selector cargo',jsonb_set(descriptor,'{seleccion,cargo_ref}','"cargo:ad170:otro"'::jsonb)::text,'42501'),
   ('bytes distintos',original||E'\n','42501'),
   ('fecha externa',jsonb_set(descriptor,'{fecha_historica}','"2026-10-03T00:00:00Z"'::jsonb)::text,'22023'),
   ('clave extra',(descriptor||jsonb_build_object('extra',true))::text,'22023'),
   ('clave repetida',replace(original,'"rol_id": "ct_cargo_jefatura"','"rol_id": "ct_cargo_jefatura", "rol_id": "ct_cargo_jefatura"'),'22023')
  ) v(nombre,bytes,estado) LOOP
   IF caso.bytes=original THEN RAISE EXCEPTION 'AD170 fixture no cambió: %',caso.nombre USING ERRCODE='55000'; END IF;
   BEGIN
    PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(
     solicitud,convert_to(caso.bytes,'UTF8'),capacidad,decision,'\x','\x',1,1,'\x','\x','\x','\x');
    RAISE EXCEPTION 'AD170 aceptó sustitución: %',caso.nombre USING ERRCODE='55000';
   EXCEPTION WHEN others THEN
    GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,diagnostico=MESSAGE_TEXT;
    IF codigo IS DISTINCT FROM caso.estado
     OR (caso.estado='42501' AND diagnostico IS DISTINCT FROM 'AD170 firma descriptor divergente') THEN
     RAISE EXCEPTION 'AD170 no comprobó rechazo de % en %',caso.nombre,via USING ERRCODE='55000'; END IF;
   END;
  END LOOP;
 END LOOP;
END $negativas$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $sin_efectos$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)::text IS DISTINCT FROM current_setting('ad170.auditorias') THEN
  RAISE EXCEPTION 'AD170 negativa consumió auditoría' USING ERRCODE='55000'; END IF;
END $sin_efectos$;
RESET ROLE;
DO $preimagen$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_temp.ad170_preimagen p WHERE p.nucleo IS DISTINCT FROM
  pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
  OR p.ad162 IS DISTINCT FROM pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
  OR p.metadatos IS DISTINCT FROM (SELECT to_jsonb(x)-'prosrc' FROM pg_proc x WHERE x.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)) THEN
  RAISE EXCEPTION 'AD170 alteró núcleo o AD162' USING ERRCODE='55000'; END IF;
END $preimagen$;
ROLLBACK;
\echo 'AD170 estructura/negativas; favorable y replay nominal NO EJECUTADOS.'
