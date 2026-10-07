\set ON_ERROR_STOP on
-- Preparada para el clon sintético, únicamente DESPUÉS de AD178 revisada.
-- No acredita consumo firmado ni recuperación histórica positiva de K.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $contrato$
DECLARE f regprocedure;meta record;
BEGIN
 f:='vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT * INTO STRICT meta FROM pg_proc WHERE oid=f;
 IF meta.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR NOT meta.prosecdef OR meta.pronargs<>10 OR meta.provolatile<>'v' OR meta.proparallel<>'u'
 OR NOT ('search_path=pg_catalog'=ANY(meta.proconfig))
 OR NOT ('row_security=on'=ANY(meta.proconfig))
 OR meta.proargnames IS DISTINCT FROM ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz','decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'] THEN
  RAISE EXCEPTION 'AD178 prueba: PARO clave=ABI actual=incompatible esperado=10_argumentos_7_resultados_owner_AUT';
 END IF;
 IF (SELECT count(*) FROM aclexplode(coalesce(meta.proacl,acldefault('f',meta.proowner))))<>2
 OR EXISTS(SELECT 1 FROM aclexplode(coalesce(meta.proacl,acldefault('f',meta.proowner))) a
 WHERE a.grantee NOT IN (meta.proowner,'vec_contratacion_temporal_propietario'::regrole)
 OR a.grantor<>meta.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
  RAISE EXCEPTION 'AD178 prueba: PARO clave=ACL actual=incompatible esperado=solo_owner_AUT_CT';
 END IF;
END $contrato$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $rechazos$
DECLARE c jsonb;d jsonb;campos jsonb:='["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb;campo text;v jsonb;rechazos integer:=0;
BEGIN
 -- Sólo entradas negativas; no se fabrica material V3 para acreditar un efecto.
 c:=jsonb_build_object('audiencia_consumo','vec_contratacion_temporal.firmas_r5.recuperar.v2',
 'operacion','contratacion_temporal.documento.firmas_r5_v2.recuperar','efecto_ref','expediente:ad178:prueba','huella_efecto_sha256',repeat('a',64));
 d:=jsonb_build_object('accion',c->>'operacion','modulo_id','contratacion_temporal','tipo_recurso','expediente_contratacion_temporal',
 'finalidad','gestionar_contratacion_temporal','recurso_ref',c->>'efecto_ref','contexto_recurso_huella_sha256',repeat('a',64),
 'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),'campos_permitidos',campos,'obligaciones','[]'::jsonb);
 FOREACH campo IN ARRAY ARRAY['CanonNominal','CanonNominalRef','CanonNominalSHA256','MaterialRootSHA256'] LOOP
  v:=d||jsonb_build_object('campos_permitidos',campos-campo);
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
    convert_to(c::text,'UTF8'),convert_to(v::text,'UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL);
   RAISE EXCEPTION 'AD178 prueba: PARO clave=campo48_ausente actual=permitido esperado=42501';
  EXCEPTION WHEN insufficient_privilege THEN rechazos:=rechazos+1; END;
 END LOOP;
 v:=d||jsonb_build_object('campos_permitidos',campos-ARRAY['CanonNominal','CanonNominalRef','CanonNominalSHA256','MaterialRootSHA256']);
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
   convert_to(c::text,'UTF8'),convert_to(v::text,'UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD178 prueba: PARO clave=consulta44 actual=permitida esperado=42501';
 EXCEPTION WHEN insufficient_privilege THEN rechazos:=rechazos+1; END;
 IF rechazos<>5 THEN RAISE EXCEPTION 'AD178 prueba: PARO clave=rechazos actual=% esperado=5',rechazos; END IF;
END $rechazos$;
ROLLBACK;
SELECT 'AD178-ABI-ACL-RECHAZOS48-OK; sin consumo positivo';
