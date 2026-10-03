\set ON_ERROR_STOP on
-- Solo sobre clon PostgreSQL 18 con AD149 -> AD151 -> AD156 -> AD157.
-- Comprueba la fachada preparatoria. No consume autorización ni firma nada.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 s jsonb; d jsonb; c jsonb; material text; decision bytea; capacidad bytea;
 d_otro jsonb; c_otro jsonb; decision_otra bytea;
 huella text; contexto text; err text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-157 prueba: falta fachada'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
    AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'AD3-157 prueba: entorno de fachada abierto'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable
      OR a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole))) THEN
  RAISE EXCEPTION 'AD3-157 prueba: ACL abierta'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
     'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
     AND strpos(p.prosrc,'firma_vec_documento_ct')<>0)
    OR EXISTS (SELECT 1 FROM pg_constraint q WHERE q.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
     AND q.conname='clave_capacidad_version_audiencia_consumo_check'
     AND strpos(pg_get_constraintdef(q.oid,true),'vec_contratacion_temporal.firma_vec.v1')<>0) THEN
  RAISE EXCEPTION 'AD3-157 prueba: se abrió el núcleo o la audiencia'; END IF;
 s:=jsonb_build_object(
  'Via','certificado_vec','OrganizacionRef','organizacion:ct:prueba','ExpedienteRef','expediente:ct:prueba',
  'VersionExpediente',7,'Documento','informe_definitivo','CatalogoRef','catalogo:prueba',
  'CatalogoHuella',repeat('a',64),'PasoRef','paso:prueba','PasoOrden',1,'Secuencia',1,
  'HistoriaRevision',0,'HistoriaHuella',repeat('a',64),
  'OriginalRef','ref:'||repeat('b',64),'OriginalVersion',1,'OriginalHuella',repeat('b',64),
  'FirmadoHuella',repeat('c',64),'CertificadoHuella',repeat('d',64),'FirmanteRef','ref:'||repeat('d',64),
  'FirmantePrincipalRef','per_prueba_firmante','PerfilFirmanteRef','perfil:ct:jefatura',
  'CargoFirmante','Jefatura','UnidadFirmanteRef','unidad:prueba',
  'PerfilActivoFirmanteRef','perfil-activo:prueba','PuestoFirmanteRef',NULL,
  'AmbitoFirmanteRef',NULL,'AsignacionFirmanteRef','asignacion:prueba','AsignacionFirmanteVersion',1,
  'AsignacionFirmanteHuella',repeat('e',64),
  'VersionRolFirmanteRef','rol:publicado:prueba','VersionRolFirmanteHuella',repeat('f',64),
  'ControlVigenciaFirmanteRef','rol:publicado:prueba','ControlVigenciaFirmanteRevision',1,
  'ControlVigenciaFirmanteHuella',repeat('1',64),
  'AsignacionVigenteDesde','2026-10-02T00:00:00Z','AsignacionVigenteHasta','2026-10-03T00:00:00Z',
  'ActoCompetenciaRef',NULL,'DelegacionRef',NULL,
  'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v1',
  'RevocacionEstado','vigente','SelloTiempoEstado','no_presente',
  'ClaveIdempotencia','clave_prueba_firma_vec_0001','DocumentoCustodiaRef','custodia:prueba',
  'DocumentoCustodiaVersion',1);
 IF (SELECT count(*) FROM jsonb_object_keys(s))<>43 THEN RAISE EXCEPTION 'AD3-157 prueba: canon incompleto'; END IF;
 material:=s::text;
 huella:=encode(sha256(convert_to(material,'UTF8')),'hex');
 contexto:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:ct:prueba"},"atributos":{"material_sha256":"'||huella||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('accion','contratacion_temporal.documento.firma_vec.registrar',
  'modulo_id','contratacion_temporal','tipo_recurso','firma_vec_documento_contratacion_temporal',
  'finalidad','gestionar_contratacion_temporal','recurso_ref','operacion-firma-vec-ct:clave_prueba_firma_vec_0001',
  'contexto_recurso_huella_sha256',contexto,'principal_id','per_prueba_firmante',
  'perfil_activo_ref','perfil-activo:prueba','version_rol_ref','rol:publicado:prueba',
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'));
 decision:=convert_to(d::text,'UTF8');
 c:=jsonb_build_object('operacion','contratacion_temporal.documento.firma_vec.registrar',
  'audiencia_consumo','vec_contratacion_temporal.firma_vec.v1',
  'efecto_ref','operacion-firma-vec-ct:clave_prueba_firma_vec_0001',
  'huella_efecto_sha256',contexto,'huella_decision_sha256',encode(sha256(decision),'hex'));
 capacidad:=convert_to(c::text,'UTF8');
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   material,capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: la firma VEC se abrió';
 EXCEPTION WHEN SQLSTATE '55000' THEN
  GET STACKED DIAGNOSTICS err=MESSAGE_TEXT;
  IF err IS DISTINCT FROM 'AD3-157: PARO clave=vinculo_rol_cargo_publicado actual=ausente esperado=atestacion_exacta' THEN
   RAISE EXCEPTION 'AD3-157 prueba: PARO inesperado'; END IF;
 END;
 d_otro:=jsonb_set(d,'{perfil_activo_ref}','"perfil-activo:otro"');
 decision_otra:=convert_to(d_otro::text,'UTF8');
 c_otro:=jsonb_set(c,'{huella_decision_sha256}',to_jsonb(encode(sha256(decision_otra),'hex')));
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   material,convert_to(c_otro::text,'UTF8'),decision_otra,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: perfil activo ajeno admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 d_otro:=jsonb_set(d,'{version_rol_ref}','"rol:publicado:otro"');
 decision_otra:=convert_to(d_otro::text,'UTF8');
 c_otro:=jsonb_set(c,'{huella_decision_sha256}',to_jsonb(encode(sha256(decision_otra),'hex')));
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   material,convert_to(c_otro::text,'UTF8'),decision_otra,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: rol ajeno admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 c:=jsonb_set(c,'{audiencia_consumo}','"vec_contratacion_temporal.firma_externa.v1"');
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   material,convert_to(c::text,'UTF8'),decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: se aceptó capacidad externa';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 s:=jsonb_set(s,'{OriginalHuella}',to_jsonb(repeat('f',64)));
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   s::text,capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: se aceptó material posterior a decisión';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 s:=jsonb_set(s,'{OriginalHuella}',to_jsonb(repeat('b',64)));
 s:=jsonb_set(s,'{CertificadoHuella}',to_jsonb(repeat('f',64)));
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
   s::text,capacidad,decision,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD3-157 prueba: cambió el certificado del material';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
 END;
END $prueba$;
ROLLBACK;
