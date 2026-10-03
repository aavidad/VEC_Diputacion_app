\set ON_ERROR_STOP on
-- CT172: estructura y rechazo con funciones reales; no sustituye V3 ni K.
-- La cadena favorable requiere fuentes nominales reales de CA/Personal/AUT.
-- Un fixture externo autorizado debe entrar por el LOGIN CT, en otra puerta.
-- No ejecutar esta prueba sobre principal ni sobre historia de producción.
\if :{?ct172_fixture_sql}
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $runtime$
BEGIN
 IF current_setting('role')<>'none' OR current_user<>session_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'CT172 fixture requiere LOGIN runtime CT sin SET ROLE'
   USING ERRCODE='42501'; END IF;
END $runtime$;
-- El fixture no contiene BEGIN/COMMIT/ROLLBACK, SET ROLE ni funciones dobles.
-- Debe comprobar alta/CAS, raíz/entrada exacta, dos firmas, denegaciones sin
-- efecto y recuperación nominal vigente con recibo/fecha/historia idénticos.
-- El descriptor exterior original y su SHA se conservan sin reescritura;
-- repetir una clave con otros bytes debe fallar con P1181, aunque el JSON sea equivalente.
-- Negativa real del fixture: tras el favorable, mantener solicitud/clave, añadir
-- un espacio al descriptor ORIGINAL y obtener otra decisión/capacidad V3 para
-- su nueva huella. Exigir P1181 y recibo/fecha/historia anteriores intactos.
-- No reutilizar la capacidad previa: sólo demostraría el rechazo AD170 de binding.
-- Replay con decisión V3 nueva exige otra transacción; reinicio exige otro
-- recorrido sobre un clon desechable, nunca se atribuyen a este ROLLBACK.
\i :ct172_fixture_sql
ROLLBACK;
\else
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $estructura$
DECLARE f regprocedure; propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 ejecutor oid:='vec_contratacion_temporal_ejecutor'::regrole;
BEGIN
 IF to_regclass('vec_contratacion_temporal.firma_documento_revision_pdf_v2') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'CT172 ausente o ABI incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
  AND c.relowner=propietario AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND p.polname='propietario' AND p.polroles=ARRAY[propietario])
  OR EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND p.polroles<>ARRAY[propietario])
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass AND a.grantee<>propietario)
  OR EXISTS(SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
   WHERE t.oid=(SELECT reltype FROM pg_class WHERE oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass)
    AND a.grantee<>propietario)
  OR (SELECT count(*) FROM pg_attribute a WHERE a.attrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND a.attname IN('descriptor_firma_original','descriptor_firma_huella_sha256') AND a.attnotnull AND NOT a.attisdropped)<>2
  OR has_table_privilege(ejecutor,'vec_contratacion_temporal.firma_documento_revision_pdf_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN
  RAISE EXCEPTION 'CT172 ACL/RLS abierta' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario
    AND p.prosecdef AND p.provolatile='v'
    AND p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'])
   OR NOT has_function_privilege(ejecutor,f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(propietario,ejecutor) OR a.is_grantable)) THEN
   RAISE EXCEPTION 'CT172 fachada abierta o configuración divergente: %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.impedir_degradacion_firma_pdf_v2()'::regprocedure,
  'vec_contratacion_temporal.comprobar_hija_firma_pdf_v2()'::regprocedure] LOOP
  IF has_function_privilege(ejecutor,f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee<>propietario) THEN
   RAISE EXCEPTION 'CT172 guarda ejecutable desde fuera' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
   AND t.tgname='hija_pdf_v2_ai' AND t.tgdeferrable AND t.tginitdeferred AND t.tgenabled='O')
  OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
   AND t.tgname='impedir_degradacion_pdf_bi' AND t.tgenabled='O')
  OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass
   AND t.tgname='revision_pdf_inmutable' AND t.tgenabled='O') THEN
  RAISE EXCEPTION 'CT172 guardas ausentes' USING ERRCODE='55000'; END IF;
END $estructura$;
-- La entrada se rechaza por la fachada real antes de consumir V3. Esta
-- comprobación no acredita una decisión favorable ni la competencia nominal.
CREATE ROLE vec_ct172_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
-- Identidad efímera mixta: el núcleo debe denegar la mezcla de autoridades.
-- CREATE/GRANT viven en esta transacción y desaparecen al ROLLBACK.
GRANT vec_contratacion_temporal_ejecutor,vec_personal_ejecutor TO vec_ct172_prueba;
CREATE TEMP TABLE ct172_preimagen AS
 SELECT 'firma'::text AS tipo,count(*) AS filas FROM vec_contratacion_temporal.firma_documento_v1
 UNION ALL SELECT 'revision',count(*) FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2
 UNION ALL SELECT 'custodia',count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1
 UNION ALL SELECT 'auditoria',count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1
 UNION ALL SELECT 'outbox',count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1;
SET SESSION AUTHORIZATION vec_ct172_prueba;
DO $negativas$
DECLARE campo text; material jsonb; base jsonb; evidencia jsonb; descriptor jsonb;
 bytes_descriptor bytea; decision bytea; capacidad bytea; contexto_h text; traza text;
BEGIN
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_verificada_v2(
   '{}','2026-10-03T00:00:00Z','\x','\x','\x','\x',1,1,'\x','\x','\x','\x','\x');
  RAISE EXCEPTION 'CT172 aceptó material vacío' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN NULL;
 END;
 -- Entrada completa sólo de formato: no es un PDF ni una autoridad acreditada.
 -- El núcleo REAL deniega la identidad técnica mixta antes de criptografía.
 -- Los sobres no acreditados nunca se usan como una autorización favorable.
 evidencia:=jsonb_build_array(jsonb_build_object('Orden',1,'ByteRange',jsonb_build_array(0,1050,1150,50),
  'RevisionHuellaSHA256',repeat('d',64),'ContenidoFirmadoHuellaSHA256',repeat('4',64),'RevisionLongitud',1200,
  'CubreDocumentoCompletoHastaAqui',true,'FirmanteRef','ref:'||repeat('e',64),'CertificadoHuellaSHA256',repeat('e',64),
  'IntegridadEstado','valida','CadenaEstado','valida','CertificadoEstado','vigente','RevocacionEstado','vigente',
  'SelloTiempoEstado','no_presente','TipoFirma','aprobacion','NivelDocMDP',NULL,
  'CambiosDesdeAnterior',jsonb_build_object('Estado','permitidos','Detalle',jsonb_build_array('firma_anadida'))));
 base:=jsonb_build_object(
   'Via','certificado_vec',
   'OrganizacionRef','organizacion:ct172:negativas',
   'ExpedienteRef','expediente:ct172:negativas',
   'VersionExpediente',1,
   'Documento','informe_definitivo',
   'CatalogoRef','vec.circuito:1',
   'CatalogoHuella',repeat('a',64),
   'PasoRef','vec.circuito:1:informe.p1',
   'PasoOrden',1,
   'Secuencia',1,
   'HistoriaRevision',0,
   'HistoriaHuella',repeat('b',64),
   'OriginalRef','documento:ct172:original',
   'OriginalVersion',1,
   'OriginalHuella',repeat('c',64),
   'FirmadoHuella',repeat('d',64),
   'CertificadoHuella',repeat('e',64),
   'FirmanteRef','ref:'||repeat('e',64),
   'FirmantePrincipalRef','per_ct172_firmante',
   'PerfilFirmanteRef','perfil:ct:jefatura',
   'CargoFirmante','ct_cargo_jefatura',
   'UnidadFirmanteRef','unidad:ct172:negativas',
   'PerfilActivoFirmanteRef','prf_ct172_firmante',
   'PuestoFirmanteRef',NULL,
   'AmbitoFirmanteRef',NULL)||
  jsonb_build_object(
   'AsignacionFirmanteRef','asignacion:ct172:negativas',
   'AsignacionFirmanteVersion',1,
   'AsignacionFirmanteHuella',repeat('f',64),
   'VersionRolFirmanteRef','rol:ct_cargo_jefatura:v1',
   'VersionRolFirmanteHuella',repeat('1',64),
   'ControlVigenciaFirmanteRef','rol:ct_cargo_jefatura:v1',
   'ControlVigenciaFirmanteRevision',1,
   'ControlVigenciaFirmanteHuella',repeat('2',64),
   'AsignacionVigenteDesde','2026-01-01T00:00:00Z',
   'AsignacionVigenteHasta','2027-01-01T00:00:00Z',
   'ActoCompetenciaRef',NULL,
   'DelegacionRef',NULL,
   'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v2',
   'RevocacionEstado','vigente',
   'SelloTiempoEstado','no_presente',
   'ClaveIdempotencia','clave-ct172-negativa-0001',
   'DocumentoCustodiaRef','documento:ct172:firmado',
   'DocumentoCustodiaVersion',1,
   'CatalogoVersion',1,
   'RolIDFirmante','ct_cargo_jefatura',
   'PerfilActivoOperadorRef','prf_ct172_firmante',
   'CuentaFirmanteRef','cuenta:ct172:negativas',
   'VinculoCredencialFirmanteRef','vinculo:ct172:negativas',
   'VinculoCredencialFirmanteRevision',1,
   'VinculoCredencialFirmanteHuella',repeat('3',64))||
  jsonb_build_object(
   'FirmaAnteriorRef',NULL,
   'ReciboAnteriorRef',NULL,
   'EntradaDocumentoRef','documento:ct172:original',
   'EntradaDocumentoVersion',1,
   'EntradaDocumentoLongitud',1000,
   'EntradaDocumentoHuella',repeat('c',64),
   'OrdenFirmaPDF',1,
   'ByteRange',jsonb_build_array(0,1050,1150,50),
   'RevisionHuellaSHA256',repeat('d',64),
   'ContenidoFirmadoHuellaSHA256',repeat('4',64),
   'RevisionLongitud',1200,
   'EvidenciaFirmasCanonica',evidencia,
   'EvidenciaFirmasHuellaSHA256',encode(sha256(convert_to(evidencia::text,'UTF8')),'hex'));
 descriptor:=jsonb_build_object('esquema','vec.competencia-firmante.constructor-ct.v1',
  'certificado_der_sha256',base->'CertificadoHuella',
  'seleccion',jsonb_build_object('perfil_esperado_ref',base->'PerfilFirmanteRef','perfil_activo_ref',base->'PerfilActivoFirmanteRef',
   'rol_id',base->'RolIDFirmante','cargo_ref','cargo:ct172:negativas','enlace_ejercicio_ref','enlace:ct172:negativas'),
  'recurso',jsonb_build_object('organizacion_ref',base->'OrganizacionRef','unidad_ref',base->'UnidadFirmanteRef',
   'expediente_ref',base->'ExpedienteRef','documento_ref',base->'OriginalRef','recurso_autorizable_ref',base->'OriginalRef',
   'modulo_id','contratacion_temporal','tipo_recurso','documento','recurso_contexto_sha256',repeat('5',64),
   'original',jsonb_build_object('referencia',base->'OriginalRef','version',base->'OriginalVersion','huella_sha256',base->'OriginalHuella'),
   'pdf_raiz_sha256',base->'OriginalHuella',
   'firmado',jsonb_build_object('referencia',base->'DocumentoCustodiaRef','version',base->'DocumentoCustodiaVersion','huella_sha256',base->'FirmadoHuella'),
   'pdf_firmado_sha256',base->'FirmadoHuella','numero_firmas',1,'entrada_revision',NULL),
  'accion','contratacion_temporal.documento.firmar','finalidad','formalizar',
  'motivo',jsonb_build_object('catalogo_id','vec.motivos','catalogo_version',1,'catalogo_huella_sha256',repeat('6',64),'entrada_clave','firma'),
  'circuito',jsonb_build_object('referencia',base->'CatalogoRef','version',base->'CatalogoVersion','huella_sha256',base->'CatalogoHuella'),
  'paso_ref',base->'PasoRef','paso_orden',1,'fecha_historica',NULL);
 bytes_descriptor:=convert_to(descriptor::text,'UTF8');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:ct172:negativas"},"atributos":{"descriptor_firma_sha256":"'||
  encode(sha256(bytes_descriptor),'hex')||'","material_sha256":"'||encode(sha256(convert_to(base::text,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 decision:=convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firma_vec.registrar',
  'modulo_id','contratacion_temporal','tipo_recurso','firma_vec_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
  'recurso_ref','operacion-firma-vec-ct:'||(base->>'ClaveIdempotencia'),'contexto_recurso_huella_sha256',contexto_h,
  'principal_id',base->'FirmantePrincipalRef','perfil_activo_ref',base->'PerfilActivoOperadorRef',
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('suite','no_acreditada','operacion','contratacion_temporal.documento.firma_vec.registrar',
  'audiencia_consumo','vec_contratacion_temporal.firma_vec.v2','efecto_ref','operacion-firma-vec-ct:'||(base->>'ClaveIdempotencia'),
  'huella_efecto_sha256',contexto_h,'huella_decision_sha256',encode(sha256(decision),'hex'))::text,'UTF8');
 BEGIN
  PERFORM vec_contratacion_temporal.registrar_firma_verificada_v2(base::text,'2026-10-03T00:00:00Z',capacidad,decision,
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),bytes_descriptor);
  RAISE EXCEPTION 'CT172 aceptó mezcla de autoridades técnicas' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  GET STACKED DIAGNOSTICS traza=PG_EXCEPTION_CONTEXT;
  IF strpos(traza,'consumir_decision_mutacion_v3_interna')=0 THEN
   RAISE EXCEPTION 'CT172 control no alcanzó consumidor V3 común' USING ERRCODE='55000'; END IF;
 END;
 -- Sólo cambia el campo bajo prueba. Si falta la guarda, debe llegar al
 -- mismo rechazo V3 42501; por eso un 22023 acredita su rechazo temprano.
 FOREACH campo IN ARRAY ARRAY['PuestoFirmanteRef','AmbitoFirmanteRef','ActoCompetenciaRef'] LOOP
  material:=base||jsonb_build_object(campo,'ref:ct172:no-acreditada');
  BEGIN
   PERFORM vec_contratacion_temporal.registrar_firma_verificada_v2(material::text,'2026-10-03T00:00:00Z',capacidad,decision,
    convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),bytes_descriptor);
   RAISE EXCEPTION 'CT172 aceptó metadato sin fuente: %',campo USING ERRCODE='55000';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
  END;
 END LOOP;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2('{}','\x','\x','\x','\x',1,1,'\x','\x','\x','\x');
  RAISE EXCEPTION 'CT172 aceptó consulta vacía' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN NULL;
 END;
END $negativas$;
RESET SESSION AUTHORIZATION;
DO $sin_efectos$
BEGIN
 IF EXISTS(SELECT tipo,filas FROM pg_temp.ct172_preimagen EXCEPT
  (SELECT 'firma',count(*) FROM vec_contratacion_temporal.firma_documento_v1
   UNION ALL SELECT 'revision',count(*) FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2
   UNION ALL SELECT 'custodia',count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1
   UNION ALL SELECT 'auditoria',count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1
   UNION ALL SELECT 'outbox',count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1)) THEN
  RAISE EXCEPTION 'CT172 rechazo produjo efecto' USING ERRCODE='55000'; END IF;
END $sin_efectos$;
ROLLBACK;
\echo 'CT172: estructura/frontera técnica/guardas; favorable V3/cripto/CAS/replay/reinicio y descriptor distinto con V3 nueva NO EJECUTADOS.'
\endif
