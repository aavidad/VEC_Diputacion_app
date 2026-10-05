\set ON_ERROR_STOP on
-- CT175 tras AD178, AD177 y AUT41. Se ejecuta conectado con un LOGIN del runtime
-- CT (miembro de vec_contratacion_temporal_ejecutor, sin propietario/migrador).
-- Comprueba composición y rechazos; no hay consumo positivo ni recorrido causal.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE f regprocedure:='vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 solicitud text; h text; contexto text; c bytea; d bytea; rechazo text;
BEGIN
 IF NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') THEN
  RAISE EXCEPTION 'CT175 prueba: conéctate con un LOGIN del runtime CT'; END IF;
 -- El runtime sólo ve la fachada CT; ni la de AD178 ni la lectura AUT41.
 IF NOT has_function_privilege(session_user,f,'EXECUTE')
  OR has_function_privilege(session_user,'vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
  OR has_function_privilege(session_user,'vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)'::regprocedure,'EXECUTE')
  OR has_function_privilege(session_user,'vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)'::regprocedure,'EXECUTE') THEN
  RAISE EXCEPTION 'CT175 prueba: ACL del runtime divergente'; END IF;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2 THEN
  RAISE EXCEPTION 'CT175 prueba: ACL de la fachada divergente'; END IF;
 -- 1. Material con una clave de más: 22023 antes de consumir.
 BEGIN
  PERFORM vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2('{"Via":"certificado_vec","Extra":1}',
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT175 prueba: material inválido aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 -- 2. Material válido con capacidad de otro expediente: 42501 antes de consumir.
 solicitud:='{"OrganizacionRef":"org-prueba","ExpedienteRef":"exp-prueba-1","VersionExpediente":1,"Documento":"contrato","FirmantePrincipalCandidatoRef":"per_prueba","ClaveIdempotencia":"prueba-ct175-000001","PasoOrden":1,"CatalogoHuella":"'||repeat('a',64)||'","Via":"certificado_vec","UnidadRef":null}';
 h:=encode(sha256(convert_to(solicitud,'UTF8')),'hex');
 contexto:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"org-prueba"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 d:=convert_to('{"recurso_ref":"exp-prueba-1","contexto_recurso_huella_sha256":"'||contexto||'"}','UTF8');
 c:=convert_to('{"efecto_ref":"exp-otro","huella_efecto_sha256":"'||contexto||'","huella_decision_sha256":"'||encode(sha256(d),'hex')||'"}','UTF8');
 BEGIN
  PERFORM vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(solicitud,c,d,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT175 prueba: capacidad ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM<>'lectura V2 divergente' THEN RAISE EXCEPTION 'CT175 prueba: causa inesperada 2: %',SQLERRM; END IF;
 END;
 -- 3. Todo ligado y con la forma de AD178, pero sin atestación: llega al núcleo y se deniega.
 d:=convert_to(jsonb_build_object('recurso_ref','exp-prueba-1','contexto_recurso_huella_sha256',contexto,
  'accion','contratacion_temporal.documento.firmas_r5_v2.recuperar','modulo_id','contratacion_temporal',
  'tipo_recurso','expediente_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),'obligaciones','[]'::jsonb,
  'campos_permitidos','["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb)::text,'UTF8');
 c:=convert_to(jsonb_build_object('efecto_ref','exp-prueba-1','huella_efecto_sha256',contexto,
  'huella_decision_sha256',encode(sha256(d),'hex'),'operacion','contratacion_temporal.documento.firmas_r5_v2.recuperar',
  'audiencia_consumo','vec_contratacion_temporal.firmas_r5.recuperar.v2')::text,'UTF8');
 BEGIN
  PERFORM vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(solicitud,c,d,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT175 prueba: decisión sin atestar aceptada';
 EXCEPTION WHEN OTHERS THEN
  -- El núcleo rechaza por su propia validación (límites, firma, vigencia).
  GET STACKED DIAGNOSTICS rechazo=PG_EXCEPTION_CONTEXT;
  IF strpos(rechazo,'consumir_decision_mutacion_v3_interna')=0 THEN
   RAISE EXCEPTION 'CT175 prueba: el rechazo no vino del núcleo: % / %',SQLERRM,rechazo; END IF;
 END;
END $prueba$;
ROLLBACK;
SELECT 'CT175-COMPOSICION-Y-3-RECHAZOS-OK; sin consumo positivo';
