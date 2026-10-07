\set ON_ERROR_STOP on
-- CT172 añade revisiones PDF a cada acto V2; conserva CT118/170 y su cabeza global.
-- Requiere CT170, AD162/AD170, CT174 y AUT32/AUT35 con sus fuentes nominales. Solo UP.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000172',0));
DO $pre$
DECLARE nombre text; f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
   OR getdatabaseencoding()<>'UTF8'
   OR to_regclass('vec_contratacion_temporal.firma_documento_revision_pdf_v2') IS NOT NULL THEN
  RAISE EXCEPTION 'CT172 preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['firma_documento_v1','firma_documento_custodia_v1',
  'firma_documento_auditoria_v1','firma_documento_outbox_v1','firma_historia_cabeza_v1'] LOOP
  IF to_regclass('vec_contratacion_temporal.'||nombre) IS NULL THEN
   RAISE EXCEPTION 'CT172 preimagen incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 -- Resolver primero el OID mantiene el rechazo nominal 55000 si falta una
 -- dependencia; has_function_privilege(text) daría undefined_function.
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(text,text,text,numeric)',
  'vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)',
  'vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)',
  'vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)'] LOOP
  f:=to_regprocedure(nombre);
  IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
   RAISE EXCEPTION 'CT172 dependencia no disponible: %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
DO $politica_pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
  AND c.conname='firma_documento_v1_politica_verificacion_check' AND c.contype='c' AND c.convalidated
  AND pg_get_constraintdef(c.oid,true)='CHECK (politica_verificacion = ''politica:vec:firma:verificacion-autonoma:v1''::text)') THEN
  RAISE EXCEPTION 'CT172 política previa incompatible' USING ERRCODE='55000'; END IF;
END $politica_pre$;
ALTER TABLE vec_contratacion_temporal.firma_documento_v1
 DROP CONSTRAINT firma_documento_v1_politica_verificacion_check,
 ADD CONSTRAINT firma_documento_v1_politica_verificacion_check CHECK(politica_verificacion IN(
  'politica:vec:firma:verificacion-autonoma:v1','politica:vec:firma:verificacion-autonoma:v2'));
CREATE TABLE vec_contratacion_temporal.firma_documento_revision_pdf_v2(
 firma_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.firma_documento_v1(firma_ref),
 catalogo_version numeric(20,0) NOT NULL CHECK(catalogo_version BETWEEN 1 AND 9007199254740991),
 firma_anterior_ref text REFERENCES vec_contratacion_temporal.firma_documento_revision_pdf_v2(firma_ref),
 recibo_anterior_ref text REFERENCES vec_contratacion_temporal.firma_documento_v1(recibo_ref),
 entrada_documento_ref text NOT NULL CHECK(entrada_documento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 entrada_documento_version numeric(20,0) NOT NULL CHECK(entrada_documento_version BETWEEN 1 AND 9007199254740991),
 entrada_huella_sha256 text NOT NULL CHECK(entrada_huella_sha256 ~ '^[0-9a-f]{64}$'),
 entrada_longitud numeric(20,0) NOT NULL CHECK(entrada_longitud>0),
 orden_pdf integer NOT NULL CHECK(orden_pdf BETWEEN 1 AND 2),
 byte_range bigint[] NOT NULL CHECK(array_length(byte_range,1)=4 AND array_lower(byte_range,1)=1
  AND byte_range[1]=0 AND byte_range[2]>=entrada_longitud AND byte_range[3]>byte_range[2] AND byte_range[4]>0),
 revision_huella_sha256 text NOT NULL CHECK(revision_huella_sha256 ~ '^[0-9a-f]{64}$'),
 contenido_firmado_huella_sha256 text NOT NULL CHECK(contenido_firmado_huella_sha256 ~ '^[0-9a-f]{64}$'),
 revision_longitud numeric(20,0) NOT NULL CHECK(revision_longitud BETWEEN 1 AND 33554432 AND revision_longitud>entrada_longitud
  AND revision_longitud=byte_range[3]+byte_range[4]),
 evidencia_firmas_canonica jsonb NOT NULL CHECK(jsonb_typeof(evidencia_firmas_canonica)='array' AND jsonb_array_length(evidencia_firmas_canonica)=orden_pdf),
 evidencia_firmas_texto text NOT NULL CHECK(octet_length(evidencia_firmas_texto) BETWEEN 2 AND 32768 AND evidencia_firmas_texto::jsonb=evidencia_firmas_canonica),
 evidencia_firmas_huella_sha256 text NOT NULL CHECK(evidencia_firmas_huella_sha256=encode(sha256(convert_to(evidencia_firmas_texto,'UTF8')),'hex')),
 comprobada_en timestamptz(6) NOT NULL CHECK(isfinite(comprobada_en)),
 rol_id_firmante text NOT NULL CHECK(rol_id_firmante ~ '^ct_cargo_[a-z0-9_]{2,80}$'),
 cuenta_firmante_ref text NOT NULL CHECK(cuenta_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 vinculo_credencial_firmante_ref text NOT NULL CHECK(vinculo_credencial_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 vinculo_credencial_firmante_revision numeric(20,0) NOT NULL CHECK(vinculo_credencial_firmante_revision BETWEEN 1 AND 9007199254740991),
 vinculo_credencial_firmante_huella text NOT NULL CHECK(vinculo_credencial_firmante_huella ~ '^[0-9a-f]{64}$'),
 competencia_evidencia_ref text NOT NULL CHECK(competencia_evidencia_ref ~ '^evidencia:competencia-firmante-ct:[0-9a-f]{64}$'),
 competencia_evidencia_huella_sha256 text NOT NULL CHECK(competencia_evidencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
 competencia_evidencia_esquema text NOT NULL CHECK(competencia_evidencia_esquema='vec.competencia-firmante.historica.v1'),
 competencia_consumo_decision_ref text NOT NULL,
 competencia_consumo_huella_sha256 text NOT NULL CHECK(competencia_consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 competencia_auditoria_consumo_ref text NOT NULL,
 descriptor_firma_original bytea NOT NULL CHECK(octet_length(descriptor_firma_original) BETWEEN 512 AND 32768),
 descriptor_firma_huella_sha256 text NOT NULL CHECK(descriptor_firma_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND descriptor_firma_huella_sha256=encode(sha256(descriptor_firma_original),'hex')),
 CHECK((orden_pdf=1 AND firma_anterior_ref IS NULL AND recibo_anterior_ref IS NULL)
   OR (orden_pdf=2 AND firma_anterior_ref IS NOT NULL AND recibo_anterior_ref IS NOT NULL AND firma_anterior_ref<>firma_ref))
);
ALTER TABLE vec_contratacion_temporal.firma_documento_revision_pdf_v2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.firma_documento_revision_pdf_v2 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.firma_documento_revision_pdf_v2
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_revision_pdf_v2 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_revision_pdf_v2 FROM PUBLIC;
CREATE TRIGGER revision_pdf_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.firma_documento_revision_pdf_v2
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
-- Las funciones V1 permanecen instaladas. Este cerrojo impide continuar una
-- ronda V2 mediante un contrato de una sola firma, incluso desde CT118/145.
CREATE FUNCTION vec_contratacion_temporal.impedir_degradacion_firma_pdf_v2() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF TG_OP<>'INSERT' OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal' OR TG_TABLE_NAME<>'firma_documento_v1' THEN
  RAISE EXCEPTION 'guarda de firma PDF denegada' USING ERRCODE='42501'; END IF;
 -- El bloqueo comparte la cabeza global usada por CT170 y serializa también legado.
 PERFORM 1 FROM vec_contratacion_temporal.firma_historia_cabeza_v1 h
  WHERE h.organizacion_ref=NEW.organizacion_ref AND h.expediente_ref=NEW.expediente_ref FOR UPDATE;
 IF NEW.resultado='firmado' AND NEW.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
   AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
    WHERE f.organizacion_ref=NEW.organizacion_ref AND f.expediente_ref=NEW.expediente_ref
      AND f.documento=NEW.documento AND f.catalogo_huella_sha256=NEW.catalogo_huella_sha256
      AND f.politica_verificacion='politica:vec:firma:verificacion-autonoma:v2'
      AND (NEW.original_documento_ref IS NULL OR
       ROW(f.original_documento_ref,f.original_documento_version,f.original_huella_sha256)
       IS NOT DISTINCT FROM ROW(NEW.original_documento_ref,NEW.original_documento_version,NEW.original_huella_sha256))) THEN
  RAISE EXCEPTION 'ronda PDF V2 no admite degradación V1' USING ERRCODE='P1184'; END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER impedir_degradacion_pdf_bi BEFORE INSERT ON vec_contratacion_temporal.firma_documento_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.impedir_degradacion_firma_pdf_v2();
CREATE FUNCTION vec_contratacion_temporal.comprobar_hija_firma_pdf_v2() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF TG_OP<>'INSERT' OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal' OR TG_TABLE_NAME<>'firma_documento_v1' THEN
  RAISE EXCEPTION 'guarda de revisión PDF denegada' USING ERRCODE='42501'; END IF;
 IF NEW.politica_verificacion='politica:vec:firma:verificacion-autonoma:v2' AND NOT EXISTS(
   SELECT 1 FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 r WHERE r.firma_ref=NEW.firma_ref
    AND r.orden_pdf=NEW.paso_orden AND r.revision_huella_sha256=NEW.firmado_huella_sha256) THEN
  RAISE EXCEPTION 'acto V2 sin revisión PDF exacta' USING ERRCODE='23514'; END IF;
 RETURN NULL;
END $f$;
CREATE CONSTRAINT TRIGGER hija_pdf_v2_ai AFTER INSERT ON vec_contratacion_temporal.firma_documento_v1
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.comprobar_hija_firma_pdf_v2();
CREATE FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(
 p_solicitud text,p_comprobada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_descriptor_nominal bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE s jsonb; d jsonb; descriptor jsonb; descriptor_texto text; contexto_nominal bytea; canon jsonb; relacion jsonb; competencia jsonb; consumo record; previa record; enlace record; cabeza record; anterior record; revision_anterior record; evidencia_firma jsonb; br jsonb;
 h text; descriptor_h text; contexto_h text; recurso text; v numeric; org text; siguiente integer;
 ahora timestamptz(6); firma text; recibo text; auditoria text; evento text;
 k text; fecha text; accion text; tipo_recurso text; operacion_auditoria text; tipo_evento text;
 vacia text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user = current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
 THEN RAISE EXCEPTION 'registro de firma verificada denegado' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536 THEN
    RAISE EXCEPTION 'material de firma verificada inválido' USING ERRCODE='22023'; END IF;
 IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
    RAISE EXCEPTION 'decisión de firma verificada inválida' USING ERRCODE='42501'; END IF;
 s := p_solicitud::jsonb;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_solicitud::json)) <> (SELECT count(*) FROM jsonb_each(s))
    OR s->>'Via' NOT IN ('certificado_vec','portafirmas_registro_rrhh')
 THEN RAISE EXCEPTION 'material de firma verificada inválido' USING ERRCODE='22023'; END IF;
 -- El canon nominal actual no contiene fuentes contrastables de estos tres
 -- metadatos. Mantener NULL hasta que su propietario publique esa capacidad.
 IF s->'PuestoFirmanteRef' IS DISTINCT FROM 'null'::jsonb
  OR s->'AmbitoFirmanteRef' IS DISTINCT FROM 'null'::jsonb
  OR s->'ActoCompetenciaRef' IS DISTINCT FROM 'null'::jsonb THEN
  RAISE EXCEPTION 'metadatos nominales no acreditados' USING ERRCODE='22023'; END IF;
 IF (s->>'Via'='portafirmas_registro_rrhh' AND
     vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
      'Via','OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
      'PasoRef','PasoOrden','Secuencia','HistoriaRevision','HistoriaHuella',
      'OriginalRef','OriginalVersion','OriginalHuella','FirmadoHuella',
      'CertificadoHuella','FirmanteRef','FirmantePrincipalRef','PerfilFirmanteRef','CargoFirmante',
      'UnidadFirmanteRef','PerfilActivoFirmanteRef','PuestoFirmanteRef','AmbitoFirmanteRef','AsignacionFirmanteRef',
      'AsignacionFirmanteVersion','AsignacionFirmanteHuella','VersionRolFirmanteRef',
      'VersionRolFirmanteHuella','ControlVigenciaFirmanteRef','ControlVigenciaFirmanteRevision',
      'ControlVigenciaFirmanteHuella','AsignacionVigenteDesde','AsignacionVigenteHasta',
      'ActoCompetenciaRef','DelegacionRef','PoliticaVerificacion',
      'RevocacionEstado','SelloTiempoEstado','ReferenciaPortafirmasDeclarada','FechaPortafirmasDeclarada',
      'ClaveIdempotencia','DocumentoCustodiaRef','DocumentoCustodiaVersion','CatalogoVersion','RolIDFirmante','PerfilActivoOperadorRef','CuentaFirmanteRef','VinculoCredencialFirmanteRef','VinculoCredencialFirmanteRevision','VinculoCredencialFirmanteHuella','FirmaAnteriorRef','ReciboAnteriorRef','EntradaDocumentoRef','EntradaDocumentoVersion','EntradaDocumentoLongitud','EntradaDocumentoHuella','OrdenFirmaPDF','ByteRange','RevisionHuellaSHA256','ContenidoFirmadoHuellaSHA256','RevisionLongitud','EvidenciaFirmasCanonica','EvidenciaFirmasHuellaSHA256']) IS NOT TRUE)
    OR (s->>'Via'='certificado_vec' AND
     vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
      'Via','OrganizacionRef','ExpedienteRef','VersionExpediente','Documento','CatalogoRef','CatalogoHuella',
      'PasoRef','PasoOrden','Secuencia','HistoriaRevision','HistoriaHuella',
      'OriginalRef','OriginalVersion','OriginalHuella','FirmadoHuella',
      'CertificadoHuella','FirmanteRef','FirmantePrincipalRef','PerfilFirmanteRef','CargoFirmante',
      'UnidadFirmanteRef','PerfilActivoFirmanteRef','PuestoFirmanteRef','AmbitoFirmanteRef','AsignacionFirmanteRef',
      'AsignacionFirmanteVersion','AsignacionFirmanteHuella','VersionRolFirmanteRef',
      'VersionRolFirmanteHuella','ControlVigenciaFirmanteRef','ControlVigenciaFirmanteRevision',
      'ControlVigenciaFirmanteHuella','AsignacionVigenteDesde','AsignacionVigenteHasta',
      'ActoCompetenciaRef','DelegacionRef','PoliticaVerificacion',
      'RevocacionEstado','SelloTiempoEstado','ClaveIdempotencia','DocumentoCustodiaRef',
      'DocumentoCustodiaVersion','CatalogoVersion','RolIDFirmante','PerfilActivoOperadorRef','CuentaFirmanteRef','VinculoCredencialFirmanteRef','VinculoCredencialFirmanteRevision','VinculoCredencialFirmanteHuella','FirmaAnteriorRef','ReciboAnteriorRef','EntradaDocumentoRef','EntradaDocumentoVersion','EntradaDocumentoLongitud','EntradaDocumentoHuella','OrdenFirmaPDF','ByteRange','RevisionHuellaSHA256','ContenidoFirmadoHuellaSHA256','RevisionLongitud','EvidenciaFirmasCanonica','EvidenciaFirmasHuellaSHA256']) IS NOT TRUE)
 THEN RAISE EXCEPTION 'material de firma verificada inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['Via','OrganizacionRef','ExpedienteRef','Documento','CatalogoRef','CatalogoHuella',
  'PasoRef','HistoriaHuella','OriginalRef','OriginalHuella','FirmadoHuella','CertificadoHuella','FirmanteRef',
  'FirmantePrincipalRef','PerfilFirmanteRef','CargoFirmante','UnidadFirmanteRef','PerfilActivoFirmanteRef',
  'AsignacionFirmanteRef','AsignacionFirmanteHuella','VersionRolFirmanteRef','VersionRolFirmanteHuella',
  'ControlVigenciaFirmanteRef','ControlVigenciaFirmanteHuella',
  'AsignacionVigenteDesde','AsignacionVigenteHasta',
  'PoliticaVerificacion','RevocacionEstado','SelloTiempoEstado','ClaveIdempotencia',
  'DocumentoCustodiaRef'] LOOP
   IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' THEN
      RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF jsonb_typeof(s->'HistoriaRevision') IS DISTINCT FROM 'number'
    OR (s->>'HistoriaRevision') !~ '^(0|[1-9][0-9]{0,15})$'
    OR (s->>'HistoriaRevision')::numeric > 9007199254740991::numeric
    OR (s->>'HistoriaHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'HistoriaHuella') = repeat('0',64) THEN
    RAISE EXCEPTION 'cabeza de historia inválida' USING ERRCODE='22023'; END IF;
 IF s->>'Via'='portafirmas_registro_rrhh' THEN
   FOREACH k IN ARRAY ARRAY['ReferenciaPortafirmasDeclarada','FechaPortafirmasDeclarada'] LOOP
     IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' THEN
       RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
   END LOOP;
 END IF;
 IF jsonb_typeof(s->'DelegacionRef') IS NULL OR jsonb_typeof(s->'DelegacionRef') NOT IN ('string','null') THEN
  RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['VersionExpediente','PasoOrden','Secuencia','OriginalVersion',
  'AsignacionFirmanteVersion','ControlVigenciaFirmanteRevision','DocumentoCustodiaVersion'] LOOP
   IF jsonb_typeof(s->k) IS DISTINCT FROM 'number' OR (s->>k) !~ '^[1-9][0-9]{0,15}$'
      OR (s->>k)::numeric > 9007199254740991::numeric THEN
      RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF s->>'PoliticaVerificacion' <> 'politica:vec:firma:verificacion-autonoma:v2'
    OR (s->>'OrganizacionRef') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'ExpedienteRef') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'Documento') !~ '^[a-z][a-z0-9_]{1,63}$'
    OR s->>'RevocacionEstado' <> 'vigente'
    OR s->>'SelloTiempoEstado' NOT IN ('no_presente','valido','no_comprobado')
    OR (s->>'PasoOrden')::integer NOT BETWEEN 1 AND 16
    OR (s->>'Secuencia')::integer NOT BETWEEN 1 AND 100000
    OR (s->>'ClaveIdempotencia') !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
    OR (s->>'CatalogoHuella') !~ '^[0-9a-f]{64}$' OR (s->>'CatalogoHuella') = repeat('0',64)
    OR (s->>'OriginalHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'FirmadoHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'CertificadoHuella') !~ '^[0-9a-f]{64}$'
    OR s->>'FirmanteRef' IS DISTINCT FROM 'ref:'||(s->>'CertificadoHuella')
    OR (s->>'AsignacionFirmanteHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'VersionRolFirmanteHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'ControlVigenciaFirmanteHuella') !~ '^[0-9a-f]{64}$'
    OR s->>'ControlVigenciaFirmanteRef' IS DISTINCT FROM s->>'VersionRolFirmanteRef'
    OR (s->>'OriginalHuella') = (s->>'FirmadoHuella')
 THEN RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 -- Las fechas son declaraciones o evidencia de una fuente. Se conservan en
 -- texto canónico exacto; se rechaza toda precisión superior al microsegundo.
 FOREACH k IN ARRAY ARRAY['AsignacionVigenteDesde','AsignacionVigenteHasta'] LOOP
   fecha := s->>k;
   IF fecha !~ '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]{0,5}[1-9])?Z$'
      OR NOT isfinite(fecha::timestamptz) THEN
      RAISE EXCEPTION 'fecha de firma externa inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF s->>'Via'='portafirmas_registro_rrhh' THEN
   fecha:=s->>'FechaPortafirmasDeclarada';
   IF fecha !~ '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]{0,5}[1-9])?Z$'
      OR NOT isfinite(fecha::timestamptz) THEN
      RAISE EXCEPTION 'fecha declarada inválida' USING ERRCODE='22023'; END IF;
 END IF;
 IF (s->>'AsignacionVigenteDesde')::timestamptz >= (s->>'AsignacionVigenteHasta')::timestamptz THEN
    RAISE EXCEPTION 'ventana de competencia inválida' USING ERRCODE='22023'; END IF;

 FOREACH k IN ARRAY ARRAY['CuentaFirmanteRef','VinculoCredencialFirmanteRef'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR s->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
   RAISE EXCEPTION 'vínculo credencial inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF jsonb_typeof(s->'PerfilActivoOperadorRef') IS DISTINCT FROM 'string' OR s->>'PerfilActivoOperadorRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
  RAISE EXCEPTION 'perfil operacional inválido' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(s->'RolIDFirmante') IS DISTINCT FROM 'string' OR s->>'RolIDFirmante' IS DISTINCT FROM s->>'CargoFirmante'
  OR s->>'RolIDFirmante' !~ '^ct_cargo_[a-z0-9_]{2,80}$' THEN
  RAISE EXCEPTION 'rol nominal del firmante inválido' USING ERRCODE='22023'; END IF;
 IF p_comprobada_en IS NULL OR NOT isfinite(p_comprobada_en) THEN
  RAISE EXCEPTION 'observación de verificación inválida' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['CatalogoVersion','VinculoCredencialFirmanteRevision','EntradaDocumentoVersion','EntradaDocumentoLongitud','OrdenFirmaPDF','RevisionLongitud'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'number' OR s->>k !~ '^[1-9][0-9]{0,15}$'
    OR (s->>k)::numeric>9007199254740991 THEN RAISE EXCEPTION 'revisión PDF inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (s->>'OrdenFirmaPDF')::integer IS DISTINCT FROM (s->>'PasoOrden')::integer
   OR (s->>'OrdenFirmaPDF')::integer NOT BETWEEN 1 AND 2
   OR (s->>'RevisionLongitud')::numeric>33554432
   OR (s->>'EntradaDocumentoLongitud')::numeric >= (s->>'RevisionLongitud')::numeric
   OR s->>'EntradaDocumentoRef' IS NULL OR s->>'EntradaDocumentoRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
   OR jsonb_typeof(s->'ByteRange') IS DISTINCT FROM 'array' OR jsonb_array_length(s->'ByteRange')<>4
   OR jsonb_typeof(s->'EvidenciaFirmasCanonica') IS DISTINCT FROM 'array'
   OR jsonb_array_length(s->'EvidenciaFirmasCanonica')<>(s->>'OrdenFirmaPDF')::integer
   OR octet_length((p_solicitud::json->'EvidenciaFirmasCanonica')::text)>32768
 THEN RAISE EXCEPTION 'revisión PDF inválida' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['VinculoCredencialFirmanteHuella','EntradaDocumentoHuella','RevisionHuellaSHA256','ContenidoFirmadoHuellaSHA256','EvidenciaFirmasHuellaSHA256'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR s->>k !~ '^[0-9a-f]{64}$' OR s->>k=repeat('0',64)
  THEN RAISE EXCEPTION 'huella revisión PDF inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF s->>'RevisionHuellaSHA256' IS DISTINCT FROM s->>'FirmadoHuella'
   OR s->>'EvidenciaFirmasHuellaSHA256' IS DISTINCT FROM encode(sha256(convert_to(
     (p_solicitud::json->'EvidenciaFirmasCanonica')::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'evidencia revisión PDF divergente' USING ERRCODE='22023'; END IF;
 FOR br IN SELECT value FROM jsonb_array_elements(s->'ByteRange') LOOP
  IF jsonb_typeof(br) IS DISTINCT FROM 'number' OR br::text !~ '^(0|[1-9][0-9]{0,15})$'
    OR br::text::numeric>33554432 THEN RAISE EXCEPTION 'ByteRange inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (s#>>'{ByteRange,0}')::numeric<>0
   OR (s#>>'{ByteRange,1}')::numeric<(s->>'EntradaDocumentoLongitud')::numeric
   OR (s#>>'{ByteRange,1}')::numeric>=(s#>>'{ByteRange,2}')::numeric
   OR (s#>>'{ByteRange,3}')::numeric<=0
   OR (s#>>'{ByteRange,2}')::numeric+(s#>>'{ByteRange,3}')::numeric<>(s->>'RevisionLongitud')::numeric THEN
  RAISE EXCEPTION 'ByteRange no cubre revisión PDF' USING ERRCODE='22023'; END IF;
 FOR evidencia_firma IN SELECT value FROM jsonb_array_elements(s->'EvidenciaFirmasCanonica') LOOP
  IF jsonb_typeof(evidencia_firma) IS DISTINCT FROM 'object'
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(evidencia_firma,ARRAY[
     'Orden','ByteRange','RevisionHuellaSHA256','ContenidoFirmadoHuellaSHA256','RevisionLongitud',
     'CubreDocumentoCompletoHastaAqui','FirmanteRef','CertificadoHuellaSHA256','IntegridadEstado',
     'CadenaEstado','CertificadoEstado','RevocacionEstado','SelloTiempoEstado','TipoFirma','NivelDocMDP','CambiosDesdeAnterior']) IS NOT TRUE
    OR evidencia_firma->'CubreDocumentoCompletoHastaAqui' IS DISTINCT FROM 'true'::jsonb
    OR evidencia_firma->>'IntegridadEstado' IS DISTINCT FROM 'valida'
    OR evidencia_firma->>'CadenaEstado' IS DISTINCT FROM 'valida'
    OR evidencia_firma->>'CertificadoEstado' IS DISTINCT FROM 'vigente'
    OR evidencia_firma->>'RevocacionEstado' IS DISTINCT FROM 'vigente'
    OR evidencia_firma->>'TipoFirma' IS DISTINCT FROM 'aprobacion'
    OR evidencia_firma->'NivelDocMDP' IS DISTINCT FROM 'null'::jsonb
    OR evidencia_firma->'CambiosDesdeAnterior' IS DISTINCT FROM '{"Estado":"permitidos","Detalle":["firma_anadida"]}'::jsonb THEN
   RAISE EXCEPTION 'firma de revisión PDF inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 evidencia_firma:=s->'EvidenciaFirmasCanonica'->((s->>'OrdenFirmaPDF')::integer-1);
 IF evidencia_firma->'Orden' IS DISTINCT FROM s->'OrdenFirmaPDF'
   OR evidencia_firma->'ByteRange' IS DISTINCT FROM s->'ByteRange'
   OR evidencia_firma->'RevisionLongitud' IS DISTINCT FROM s->'RevisionLongitud'
   OR evidencia_firma->>'RevisionHuellaSHA256' IS DISTINCT FROM s->>'RevisionHuellaSHA256'
   OR evidencia_firma->>'ContenidoFirmadoHuellaSHA256' IS DISTINCT FROM s->>'ContenidoFirmadoHuellaSHA256'
   OR evidencia_firma->>'FirmanteRef' IS DISTINCT FROM s->>'FirmanteRef'
   OR evidencia_firma->>'CertificadoHuellaSHA256' IS DISTINCT FROM s->>'CertificadoHuella'
   OR evidencia_firma->>'SelloTiempoEstado' IS DISTINCT FROM s->>'SelloTiempoEstado' THEN
  RAISE EXCEPTION 'firma nueva no corresponde al acto' USING ERRCODE='22023'; END IF;
 IF (s->>'OrdenFirmaPDF')::integer=1 THEN
  IF s->'FirmaAnteriorRef' IS DISTINCT FROM 'null'::jsonb OR s->'ReciboAnteriorRef' IS DISTINCT FROM 'null'::jsonb
    OR s->>'EntradaDocumentoRef' IS DISTINCT FROM s->>'OriginalRef'
    OR s->'EntradaDocumentoVersion' IS DISTINCT FROM s->'OriginalVersion'
    OR s->>'EntradaDocumentoHuella' IS DISTINCT FROM s->>'OriginalHuella' THEN
   RAISE EXCEPTION 'primer paso PDF no parte del original' USING ERRCODE='22023'; END IF;
 ELSE
  IF s->>'FirmaAnteriorRef' IS NULL OR s->>'FirmaAnteriorRef' !~ '^firma-ct:[0-9a-f-]{36}$'
    OR s->>'ReciboAnteriorRef' IS NULL OR s->>'ReciboAnteriorRef' !~ '^recibo-firma-ct:[0-9a-f-]{36}$' THEN
   RAISE EXCEPTION 'antecedente PDF inválido' USING ERRCODE='22023'; END IF;
 END IF;
 IF p_descriptor_nominal IS NULL OR octet_length(p_descriptor_nominal) NOT BETWEEN 512 AND 32768 THEN
  RAISE EXCEPTION 'descriptor nominal de firma no disponible' USING ERRCODE='42501'; END IF;
 h := encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 descriptor_h:=encode(sha256(p_descriptor_nominal),'hex');
 IF s->>'Via'='certificado_vec' THEN
   recurso := 'operacion-firma-vec-ct:'||(s->>'ClaveIdempotencia');
   accion := 'contratacion_temporal.documento.firma_vec.registrar';
   tipo_recurso := 'firma_vec_documento_contratacion_temporal';
   operacion_auditoria := accion;
   tipo_evento := 'contratacion_temporal.documento.firma_vec_registrada';
 ELSE
   recurso := 'operacion-firma-externa-ct:'||(s->>'ClaveIdempotencia');
   accion := 'contratacion_temporal.documento.firma_externa.registrar';
   tipo_recurso := 'firma_externa_documento_contratacion_temporal';
   operacion_auditoria := accion;
   tipo_evento := 'contratacion_temporal.documento.firma_externa_registrada';
 END IF;
 contexto_h := encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
   '"},"atributos":{"descriptor_firma_sha256":"'||descriptor_h||'","material_sha256":"'||h||'"}}','UTF8')),'hex');
 BEGIN d := convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'decisión de firma verificada inválida' USING ERRCODE='42501'; END;
 IF d->>'accion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM tipo_recurso
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM recurso
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'principal_id' IS NULL OR d->>'principal_id' !~ '^per_[A-Za-z0-9_-]{2,159}$'
    OR (s->>'Via'='certificado_vec' AND d->>'principal_id' IS DISTINCT FROM s->>'FirmantePrincipalRef')
    OR (s->>'Via'='portafirmas_registro_rrhh' AND d->>'principal_id' IS NOT DISTINCT FROM s->>'FirmantePrincipalRef')
    OR (s->>'Via'='portafirmas_registro_rrhh' AND d->>'version_rol_ref' IS DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1')
    OR d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoOperadorRef'
 THEN RAISE EXCEPTION 'autorización de firma verificada divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(
  p_solicitud,p_descriptor_nominal,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR consumo.consumo_nuevo IS NOT TRUE THEN
    RAISE EXCEPTION 'consumo de firma verificada divergente' USING ERRCODE='42501'; END IF;
 -- El consumo/auditoría del registrador precede a toda revalidación.
 -- CT174 protege el puntero con FOR SHARE hasta COMMIT, también al recuperar.
 relacion := vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
  s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'UnidadFirmanteRef',
  (s->>'VersionExpediente')::numeric);
 BEGIN
  descriptor_texto:=convert_from(p_descriptor_nominal,'UTF8');
  descriptor:=descriptor_texto::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'descriptor nominal de firma no disponible' USING ERRCODE='42501'; END;
 -- Selección del plan gobernado en servidor; estos datos no conceden permiso.
 -- AUT35 obtiene identidad, fuentes Personal y asignación desde propietarios.
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor,ARRAY[
   'esquema','certificado_der_sha256','seleccion','recurso','accion','finalidad',
   'motivo','circuito','paso_ref','paso_orden','fecha_historica']) IS NOT TRUE
  OR (SELECT count(*) FROM json_each(descriptor_texto::json))<>(SELECT count(*) FROM jsonb_each(descriptor))
  OR descriptor->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.constructor-ct.v1'
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor->'seleccion',ARRAY[
   'perfil_esperado_ref','perfil_activo_ref','rol_id','cargo_ref','enlace_ejercicio_ref']) IS NOT TRUE
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor->'recurso',ARRAY[
   'organizacion_ref','unidad_ref','expediente_ref','documento_ref','recurso_autorizable_ref',
   'modulo_id','tipo_recurso','recurso_contexto_sha256','original','pdf_raiz_sha256',
   'firmado','pdf_firmado_sha256','numero_firmas','entrada_revision']) IS NOT TRUE
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor#>'{recurso,original}',ARRAY[
   'referencia','version','huella_sha256']) IS NOT TRUE
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor#>'{recurso,firmado}',ARRAY[
   'referencia','version','huella_sha256']) IS NOT TRUE
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor->'circuito',ARRAY[
   'referencia','version','huella_sha256']) IS NOT TRUE
  OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor->'motivo',ARRAY[
   'catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']) IS NOT TRUE
  OR descriptor->>'certificado_der_sha256' IS DISTINCT FROM s->>'CertificadoHuella'
  OR descriptor#>>'{seleccion,perfil_esperado_ref}' IS DISTINCT FROM s->>'PerfilFirmanteRef'
  OR descriptor#>>'{seleccion,perfil_activo_ref}' IS DISTINCT FROM s->>'PerfilActivoFirmanteRef'
  OR descriptor#>>'{seleccion,rol_id}' IS DISTINCT FROM s->>'RolIDFirmante'
  OR descriptor#>>'{recurso,organizacion_ref}' IS DISTINCT FROM s->>'OrganizacionRef'
  OR descriptor#>>'{recurso,unidad_ref}' IS DISTINCT FROM s->>'UnidadFirmanteRef'
  OR descriptor#>>'{recurso,expediente_ref}' IS DISTINCT FROM s->>'ExpedienteRef'
  OR descriptor#>>'{recurso,documento_ref}' IS DISTINCT FROM s->>'OriginalRef'
  OR descriptor#>>'{recurso,recurso_autorizable_ref}' IS DISTINCT FROM s->>'OriginalRef'
  OR descriptor#>>'{recurso,modulo_id}' IS DISTINCT FROM 'contratacion_temporal'
  OR descriptor#>>'{recurso,original,referencia}' IS DISTINCT FROM s->>'OriginalRef'
  OR descriptor#>'{recurso,original,version}' IS DISTINCT FROM s->'OriginalVersion'
  OR descriptor#>>'{recurso,original,huella_sha256}' IS DISTINCT FROM s->>'OriginalHuella'
  OR descriptor#>>'{recurso,pdf_raiz_sha256}' IS DISTINCT FROM s->>'OriginalHuella'
  OR descriptor#>>'{recurso,firmado,referencia}' IS DISTINCT FROM s->>'DocumentoCustodiaRef'
  OR descriptor#>'{recurso,firmado,version}' IS DISTINCT FROM s->'DocumentoCustodiaVersion'
  OR descriptor#>>'{recurso,firmado,huella_sha256}' IS DISTINCT FROM s->>'FirmadoHuella'
  OR descriptor#>>'{recurso,pdf_firmado_sha256}' IS DISTINCT FROM s->>'FirmadoHuella'
  OR descriptor#>'{recurso,numero_firmas}' IS DISTINCT FROM s->'OrdenFirmaPDF'
  OR descriptor#>>'{circuito,referencia}' IS DISTINCT FROM s->>'CatalogoRef'
  OR descriptor#>'{circuito,version}' IS DISTINCT FROM s->'CatalogoVersion'
  OR descriptor#>>'{circuito,huella_sha256}' IS DISTINCT FROM s->>'CatalogoHuella'
  OR descriptor->>'paso_ref' IS DISTINCT FROM s->>'PasoRef'
  OR descriptor->'paso_orden' IS DISTINCT FROM s->'PasoOrden'
  OR descriptor->'fecha_historica' IS DISTINCT FROM 'null'::jsonb
  OR ((s->>'OrdenFirmaPDF')::integer=1 AND descriptor#>'{recurso,entrada_revision}' IS DISTINCT FROM 'null'::jsonb)
  OR ((s->>'OrdenFirmaPDF')::integer=2 AND (
     vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(descriptor#>'{recurso,entrada_revision}',ARRAY[
      'referencia','version','huella_sha256']) IS NOT TRUE
     OR descriptor#>>'{recurso,entrada_revision,referencia}' IS DISTINCT FROM s->>'EntradaDocumentoRef'
     OR descriptor#>'{recurso,entrada_revision,version}' IS DISTINCT FROM s->'EntradaDocumentoVersion'
     OR descriptor#>>'{recurso,entrada_revision,huella_sha256}' IS DISTINCT FROM s->>'EntradaDocumentoHuella')) THEN
  RAISE EXCEPTION 'descriptor nominal de firma divergente' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['seleccion','recurso','motivo','circuito'] LOOP
  IF (SELECT count(*) FROM json_each((descriptor_texto::json)->k))
    <> (SELECT count(*) FROM jsonb_each(descriptor->k)) THEN
   RAISE EXCEPTION 'descriptor nominal de firma divergente' USING ERRCODE='42501'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['original','firmado','entrada_revision'] LOOP
  IF jsonb_typeof(descriptor#>ARRAY['recurso',k])='object'
   AND (SELECT count(*) FROM json_each((descriptor_texto::json)#>ARRAY['recurso',k]))
    <> (SELECT count(*) FROM jsonb_each(descriptor#>ARRAY['recurso',k])) THEN
   RAISE EXCEPTION 'descriptor nominal de firma divergente' USING ERRCODE='42501'; END IF;
 END LOOP;
 -- La fecha histórica pertenece al efecto CT, no al PDF ni al verificador.
 -- Se lee sólo para construir el contexto; no se devuelve antes de autorizar.
 SELECT registrada_en INTO ahora FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
   AND clave_idempotencia=s->>'ClaveIdempotencia';
 IF NOT FOUND THEN ahora:=date_trunc('microseconds',clock_timestamp()); END IF;
 descriptor:=descriptor||jsonb_build_object('fecha_historica',vec_contratacion_temporal.instante_utc_v1(ahora));
 contexto_nominal:=vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(
  descriptor,relacion,to_jsonb(consumo));
 IF contexto_nominal IS NULL OR octet_length(contexto_nominal) NOT BETWEEN 512 AND 32768 THEN
  RAISE EXCEPTION 'contexto nominal de firma no disponible' USING ERRCODE='42501'; END IF;
 canon:=convert_from(contexto_nominal,'UTF8')::jsonb;
 IF canon->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
  OR canon->'recurso' IS DISTINCT FROM descriptor->'recurso'
  OR canon->>'accion' IS DISTINCT FROM descriptor->>'accion'
  OR canon->>'finalidad' IS DISTINCT FROM descriptor->>'finalidad'
  OR canon->'motivo' IS DISTINCT FROM descriptor->'motivo'
  OR canon->'circuito' IS DISTINCT FROM descriptor->'circuito'
  OR canon->>'paso_ref' IS DISTINCT FROM descriptor->>'paso_ref'
  OR canon->'paso_orden' IS DISTINCT FROM descriptor->'paso_orden'
  OR canon->>'fecha_historica' IS DISTINCT FROM descriptor->>'fecha_historica'
  OR canon#>>'{identidad,persona_ref}' IS DISTINCT FROM s->>'FirmantePrincipalRef'
  OR canon#>>'{identidad,certificado_der_sha256}' IS DISTINCT FROM s->>'CertificadoHuella'
  OR canon#>>'{identidad,cuenta,referencia}' IS DISTINCT FROM s->>'CuentaFirmanteRef'
  OR canon#>>'{identidad,vinculo_certificado,referencia}' IS DISTINCT FROM s->>'VinculoCredencialFirmanteRef'
  OR canon#>'{identidad,vinculo_certificado,version}' IS DISTINCT FROM s->'VinculoCredencialFirmanteRevision'
  OR canon#>>'{identidad,vinculo_certificado,huella_sha256}' IS DISTINCT FROM s->>'VinculoCredencialFirmanteHuella'
  OR canon#>>'{competencia,rol_id}' IS DISTINCT FROM s->>'RolIDFirmante'
  OR canon#>>'{competencia,perfil_esperado_ref}' IS DISTINCT FROM s->>'PerfilFirmanteRef'
  OR canon#>>'{competencia,perfil_activo_ref}' IS DISTINCT FROM s->>'PerfilActivoFirmanteRef'
  OR canon#>>'{competencia,asignacion,referencia}' IS DISTINCT FROM s->>'AsignacionFirmanteRef'
  OR canon#>'{competencia,asignacion,version}' IS DISTINCT FROM s->'AsignacionFirmanteVersion'
  OR canon#>>'{competencia,asignacion,huella_sha256}' IS DISTINCT FROM s->>'AsignacionFirmanteHuella'
  OR canon#>>'{competencia,rol,referencia}' IS DISTINCT FROM s->>'VersionRolFirmanteRef'
  OR canon#>>'{competencia,rol,huella_sha256}' IS DISTINCT FROM s->>'VersionRolFirmanteHuella'
  OR canon#>>'{competencia,control_rol,referencia}' IS DISTINCT FROM s->>'ControlVigenciaFirmanteRef'
  OR canon#>'{competencia,control_rol,version}' IS DISTINCT FROM s->'ControlVigenciaFirmanteRevision'
  OR canon#>>'{competencia,control_rol,huella_sha256}' IS DISTINCT FROM s->>'ControlVigenciaFirmanteHuella'
  OR canon#>>'{competencia,vigente_desde}' IS DISTINCT FROM s->>'AsignacionVigenteDesde'
  OR canon#>>'{competencia,vigente_hasta}' IS DISTINCT FROM s->>'AsignacionVigenteHasta'
  OR canon#>>'{personal,delegacion,acto,referencia}' IS DISTINCT FROM s->>'DelegacionRef' THEN
  RAISE EXCEPTION 'contexto nominal de firma divergente' USING ERRCODE='42501'; END IF;
 competencia := vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(
  contexto_nominal,relacion,to_jsonb(consumo));
 IF competencia->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
  OR competencia->>'evidencia_ref' !~ '^evidencia:competencia-firmante-ct:[0-9a-f]{64}$'
  OR competencia->>'huella_sha256' !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'evidencia de competencia no disponible' USING ERRCODE='42501'; END IF;
 -- Un consumo V3 nuevo precede también al replay. El cerrojo y los UNIQUE
 -- de CT118 cubren ambas vías; SERIALIZABLE obliga a repetir una carrera.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:firma_documento:'||
   (s->>'ExpedienteRef')||':'||(s->>'Documento'),0));
 SELECT * INTO previa FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
    AND clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
   IF previa.via_registro IS DISTINCT FROM s->>'Via'
      OR previa.solicitud_huella_sha256 IS DISTINCT FROM h THEN
      RAISE EXCEPTION 'clave de firma reutilizada con otro material' USING ERRCODE='P1181'; END IF;
   IF previa.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
      OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 r WHERE r.firma_ref=previa.firma_ref) THEN
    RAISE EXCEPTION 'replay no tiene evidencia V2' USING ERRCODE='55000'; END IF;
   SELECT * INTO enlace FROM vec_contratacion_temporal.firma_documento_custodia_v1
    WHERE firma_ref=previa.firma_ref;
   IF NOT FOUND OR enlace.documento_ref IS DISTINCT FROM s->>'DocumentoCustodiaRef'
      OR enlace.documento_version IS DISTINCT FROM (s->>'DocumentoCustodiaVersion')::numeric THEN
      RAISE EXCEPTION 'enlace de custodia de firma inconsistente' USING ERRCODE='55000'; END IF;
   SELECT * INTO revision_anterior FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2
    WHERE firma_ref=previa.firma_ref;
   IF NOT FOUND OR revision_anterior.descriptor_firma_huella_sha256 IS DISTINCT FROM
     encode(sha256(revision_anterior.descriptor_firma_original),'hex') THEN
    RAISE EXCEPTION 'descriptor nominal histórico inconsistente' USING ERRCODE='55000'; END IF;
   IF revision_anterior.descriptor_firma_original IS DISTINCT FROM p_descriptor_nominal
    OR revision_anterior.descriptor_firma_huella_sha256 IS DISTINCT FROM descriptor_h THEN
    RAISE EXCEPTION 'clave de firma reutilizada con otro descriptor' USING ERRCODE='P1181'; END IF;
   IF revision_anterior.competencia_evidencia_ref IS DISTINCT FROM competencia->>'evidencia_ref'
    OR revision_anterior.competencia_evidencia_huella_sha256 IS DISTINCT FROM competencia->>'huella_sha256'
    OR revision_anterior.competencia_consumo_decision_ref IS DISTINCT FROM previa.decision_ref
    OR revision_anterior.competencia_consumo_huella_sha256 IS DISTINCT FROM previa.consumo_huella_sha256
    OR revision_anterior.competencia_auditoria_consumo_ref IS DISTINCT FROM previa.auditoria_consumo_ref THEN
    RAISE EXCEPTION 'evidencia nominal histórica de firma inconsistente' USING ERRCODE='55000'; END IF;
   RETURN jsonb_build_object('FirmaRef',previa.firma_ref,'ReciboRef',previa.recibo_ref,
      'Secuencia',previa.secuencia,'Resultado',previa.resultado,'ExpedienteVersion',previa.expediente_version,
      'ActorRef',previa.actor_ref,'PerfilRef',previa.perfil_ref,'RegistradaEn',previa.registrada_en,
      'SolicitudHuella',previa.solicitud_huella_sha256,'YaRegistrada',true,
      'DocumentoCustodiaRef',enlace.documento_ref,'DocumentoCustodiaVersion',enlace.documento_version,
      'CompetenciaEvidenciaRef',revision_anterior.competencia_evidencia_ref,
      'CompetenciaEvidenciaHuellaSHA256',revision_anterior.competencia_evidencia_huella_sha256);
 END IF;
 SELECT a.version,vv.agregado_json->>'organizacion_ref' INTO v,org
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral vv
   ON vv.expediente_ref=a.expediente_ref AND vv.version=a.version
  WHERE a.expediente_ref=s->>'ExpedienteRef' FOR KEY SHARE OF a;
 IF NOT FOUND OR org IS DISTINCT FROM s->>'OrganizacionRef' THEN
    RAISE EXCEPTION 'expediente de firma no disponible' USING ERRCODE='42501'; END IF;
 IF v IS DISTINCT FROM (s->>'VersionExpediente')::numeric THEN
    RAISE EXCEPTION 'versión del expediente en conflicto' USING ERRCODE='P1182'; END IF;
 INSERT INTO vec_contratacion_temporal.firma_historia_cabeza_v1 VALUES
  (s->>'OrganizacionRef',s->>'ExpedienteRef',0,vacia)
 ON CONFLICT (organizacion_ref,expediente_ref) DO NOTHING;
 SELECT revision,huella_sha256 INTO STRICT cabeza FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef' FOR UPDATE;
 IF cabeza.revision IS DISTINCT FROM (s->>'HistoriaRevision')::numeric
    OR cabeza.huella_sha256 IS DISTINCT FROM s->>'HistoriaHuella' THEN
    RAISE EXCEPTION 'historia de firmas modificada desde la consulta' USING ERRCODE='P1701'; END IF;
 SELECT coalesce(max(secuencia),0)+1 INTO siguiente FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
    AND documento=s->>'Documento';
 IF siguiente IS DISTINCT FROM (s->>'Secuencia')::integer THEN
    RAISE EXCEPTION 'secuencia de firma en conflicto' USING ERRCODE='P1183'; END IF;

 IF (s->>'OrdenFirmaPDF')::integer=2 THEN
  SELECT * INTO anterior FROM vec_contratacion_temporal.firma_documento_v1 f
   WHERE f.firma_ref=s->>'FirmaAnteriorRef' FOR KEY SHARE;
  IF NOT FOUND OR anterior.recibo_ref IS DISTINCT FROM s->>'ReciboAnteriorRef'
    OR anterior.organizacion_ref IS DISTINCT FROM s->>'OrganizacionRef'
    OR anterior.expediente_ref IS DISTINCT FROM s->>'ExpedienteRef'
    OR anterior.expediente_version IS DISTINCT FROM v
    OR anterior.documento IS DISTINCT FROM s->>'Documento'
    OR anterior.catalogo_ref IS DISTINCT FROM s->>'CatalogoRef'
    OR anterior.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella'
    OR anterior.paso_orden<>1 OR anterior.resultado<>'firmado'
    OR anterior.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
    OR anterior.original_documento_ref IS DISTINCT FROM s->>'OriginalRef'
    OR anterior.original_documento_version IS DISTINCT FROM (s->>'OriginalVersion')::numeric
    OR anterior.original_huella_sha256 IS DISTINCT FROM s->>'OriginalHuella'
    OR anterior.firmado_huella_sha256 IS DISTINCT FROM s->>'EntradaDocumentoHuella'
    OR anterior.secuencia<>siguiente-1
    OR anterior.firmante_principal_ref IS NOT DISTINCT FROM s->>'FirmantePrincipalRef'
    OR anterior.certificado_huella_sha256 IS NOT DISTINCT FROM s->>'CertificadoHuella'
  THEN RAISE EXCEPTION 'antecedente PDF no es primer paso activo exacto' USING ERRCODE='P1184'; END IF;
  SELECT * INTO revision_anterior FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 WHERE firma_ref=anterior.firma_ref;
  IF NOT FOUND OR revision_anterior.revision_longitud IS DISTINCT FROM (s->>'EntradaDocumentoLongitud')::numeric
    OR revision_anterior.catalogo_version IS DISTINCT FROM (s->>'CatalogoVersion')::numeric
    OR revision_anterior.evidencia_firmas_canonica->0 IS DISTINCT FROM s->'EvidenciaFirmasCanonica'->0
    OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_custodia_v1 c
     WHERE c.firma_ref=anterior.firma_ref AND c.recibo_ref=anterior.recibo_ref
       AND c.documento_ref=s->>'EntradaDocumentoRef'
       AND c.documento_version=(s->>'EntradaDocumentoVersion')::numeric
       AND c.documento_huella_sha256=s->>'EntradaDocumentoHuella') THEN
   RAISE EXCEPTION 'revisión de entrada no es custodia del antecedente exacto' USING ERRCODE='P1184'; END IF;
 ELSE
  IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
    WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
      AND f.documento=s->>'Documento' AND f.resultado='firmado'
      AND (f.original_documento_ref IS NULL OR (f.original_documento_ref=s->>'OriginalRef' AND f.original_documento_version=(s->>'OriginalVersion')::numeric))
      AND f.original_huella_sha256=s->>'OriginalHuella') THEN
   RAISE EXCEPTION 'original ya tiene firma en otra cadena' USING ERRCODE='P1184'; END IF;
 END IF;
 firma := 'firma-ct:'||gen_random_uuid()::text;
 recibo := 'recibo-firma-ct:'||gen_random_uuid()::text;
 auditoria := 'auditoria-firma-ct:'||gen_random_uuid()::text;
 evento := 'evento-firma-ct:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.firma_documento_v1 (
   firma_ref,organizacion_ref,expediente_ref,expediente_version,documento,secuencia,clave_idempotencia,
   solicitud_huella_sha256,catalogo_ref,catalogo_huella_sha256,paso_ref,paso_orden,resultado,
   original_huella_sha256,firmado_huella_sha256,certificado_huella_sha256,firmante_ref,
   politica_verificacion,verificacion_estado,verificacion_motivo,revocacion_estado,sello_tiempo_estado,
   actor_ref,perfil_ref,decision_ref,consumo_huella_sha256,auditoria_consumo_ref,recibo_ref,registrada_en,
   via_registro,historia_revision_observada,historia_huella_observada,
   original_documento_ref,original_documento_version,firmante_principal_ref,
   perfil_firmante_ref,cargo_firmante,unidad_firmante_ref,perfil_activo_firmante_ref,
   puesto_firmante_ref,ambito_firmante_ref,
   asignacion_firmante_ref,asignacion_firmante_version,asignacion_firmante_huella_sha256,
   version_rol_firmante_ref,version_rol_firmante_huella_sha256,
   control_vigencia_firmante_ref,control_vigencia_firmante_revision,control_vigencia_firmante_huella_sha256,
   asignacion_vigente_desde,asignacion_vigente_hasta,acto_competencia_ref,
   delegacion_ref,referencia_portafirmas_declarada,fecha_portafirmas_declarada)
 VALUES (firma,s->>'OrganizacionRef',s->>'ExpedienteRef',v,s->>'Documento',siguiente,
   s->>'ClaveIdempotencia',h,s->>'CatalogoRef',s->>'CatalogoHuella',s->>'PasoRef',(s->>'PasoOrden')::integer,
   'firmado',s->>'OriginalHuella',s->>'FirmadoHuella',s->>'CertificadoHuella',s->>'FirmanteRef',
   s->>'PoliticaVerificacion','valida','verificada',s->>'RevocacionEstado',s->>'SelloTiempoEstado',
   d->>'principal_id',d->>'perfil_activo_ref',consumo.decision_ref,consumo.consumo_huella_sha256,
   consumo.auditoria_ref,recibo,ahora,s->>'Via',(s->>'HistoriaRevision')::numeric,s->>'HistoriaHuella',
   s->>'OriginalRef',(s->>'OriginalVersion')::numeric,
   s->>'FirmantePrincipalRef',s->>'PerfilFirmanteRef',s->>'CargoFirmante',s->>'UnidadFirmanteRef',
   s->>'PerfilActivoFirmanteRef',
   s->>'PuestoFirmanteRef',s->>'AmbitoFirmanteRef',s->>'AsignacionFirmanteRef',
   (s->>'AsignacionFirmanteVersion')::numeric,s->>'AsignacionFirmanteHuella',
   s->>'VersionRolFirmanteRef',s->>'VersionRolFirmanteHuella',s->>'ControlVigenciaFirmanteRef',
   (s->>'ControlVigenciaFirmanteRevision')::numeric,s->>'ControlVigenciaFirmanteHuella',
   s->>'AsignacionVigenteDesde',s->>'AsignacionVigenteHasta',
   s->>'ActoCompetenciaRef',s->>'DelegacionRef',s->>'ReferenciaPortafirmasDeclarada',
   s->>'FechaPortafirmasDeclarada');
 INSERT INTO vec_contratacion_temporal.firma_documento_revision_pdf_v2 VALUES(
  firma,(s->>'CatalogoVersion')::numeric,s->>'FirmaAnteriorRef',s->>'ReciboAnteriorRef',
  s->>'EntradaDocumentoRef',(s->>'EntradaDocumentoVersion')::numeric,s->>'EntradaDocumentoHuella',
  (s->>'EntradaDocumentoLongitud')::numeric,(s->>'OrdenFirmaPDF')::integer,
  ARRAY[(s#>>'{ByteRange,0}')::bigint,(s#>>'{ByteRange,1}')::bigint,(s#>>'{ByteRange,2}')::bigint,(s#>>'{ByteRange,3}')::bigint],
  s->>'RevisionHuellaSHA256',s->>'ContenidoFirmadoHuellaSHA256',(s->>'RevisionLongitud')::numeric,
  s->'EvidenciaFirmasCanonica',(p_solicitud::json->'EvidenciaFirmasCanonica')::text,s->>'EvidenciaFirmasHuellaSHA256',
  date_trunc('microseconds',p_comprobada_en),s->>'RolIDFirmante',s->>'CuentaFirmanteRef',s->>'VinculoCredencialFirmanteRef',
  (s->>'VinculoCredencialFirmanteRevision')::numeric,s->>'VinculoCredencialFirmanteHuella',
  competencia->>'evidencia_ref',competencia->>'huella_sha256',competencia->>'esquema',
  consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,
  p_descriptor_nominal,descriptor_h);
 INSERT INTO vec_contratacion_temporal.firma_documento_custodia_v1 VALUES
   (firma,recibo,ahora,s->>'DocumentoCustodiaRef',(s->>'DocumentoCustodiaVersion')::numeric,s->>'FirmadoHuella');
 INSERT INTO vec_contratacion_temporal.firma_documento_auditoria_v1 VALUES
   (auditoria,firma,recibo,operacion_auditoria,'firmado',
    consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora);
 INSERT INTO vec_contratacion_temporal.firma_documento_outbox_v1 VALUES
   (evento,firma,recibo,tipo_evento,'pendiente',ahora);
 RETURN jsonb_build_object('FirmaRef',firma,'ReciboRef',recibo,'Secuencia',siguiente,'Resultado','firmado',
   'ExpedienteVersion',v,'ActorRef',d->>'principal_id','PerfilRef',d->>'perfil_activo_ref',
   'RegistradaEn',ahora,'SolicitudHuella',h,'YaRegistrada',false,
   'DocumentoCustodiaRef',s->>'DocumentoCustodiaRef',
   'DocumentoCustodiaVersion',(s->>'DocumentoCustodiaVersion')::numeric,
   'CompetenciaEvidenciaRef',competencia->>'evidencia_ref',
   'CompetenciaEvidenciaHuellaSHA256',competencia->>'huella_sha256');
EXCEPTION WHEN lock_not_available THEN
 RAISE EXCEPTION 'registro de firma externa no disponible' USING ERRCODE='P1185';
WHEN data_exception THEN
 RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; consumo record; exacta record; cabeza record; firmas jsonb; revisiones jsonb;
 h text; contexto_h text; v numeric; org text; coincide boolean; separacion boolean;
 vacia text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'lectura V2 denegada' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 4096
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'lectura V2 inválida' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY['OrganizacionRef','ExpedienteRef','VersionExpediente',
   'Documento','FirmantePrincipalCandidatoRef','ClaveIdempotencia','PasoOrden','CatalogoHuella','Via','UnidadRef']) IS NOT TRUE
  OR (SELECT count(*) FROM json_each(p_solicitud::json))<>(SELECT count(*) FROM jsonb_each(s))
  OR jsonb_typeof(s->'Via') IS DISTINCT FROM 'string' OR s->>'Via' NOT IN('certificado_vec','portafirmas_registro_rrhh')
  OR s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb
  OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
  OR s->>'OrganizacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
  OR s->>'ExpedienteRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR jsonb_typeof(s->'Documento') IS DISTINCT FROM 'string' OR s->>'Documento' !~ '^[a-z][a-z0-9_]{1,63}$'
  OR jsonb_typeof(s->'FirmantePrincipalCandidatoRef') IS DISTINCT FROM 'string'
  OR s->>'FirmantePrincipalCandidatoRef' !~ '^per_[A-Za-z0-9_-]{2,159}$'
  OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
  OR s->>'ClaveIdempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
  OR jsonb_typeof(s->'CatalogoHuella') IS DISTINCT FROM 'string' OR s->>'CatalogoHuella' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(s->'VersionExpediente') IS DISTINCT FROM 'number' OR s->>'VersionExpediente' !~ '^[1-9][0-9]{0,15}$'
  OR (s->>'VersionExpediente')::numeric>9007199254740991
  OR jsonb_typeof(s->'PasoOrden') IS DISTINCT FROM 'number' OR s->>'PasoOrden' !~ '^[12]$' THEN
  RAISE EXCEPTION 'lectura V2 inválida' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
  OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
  RAISE EXCEPTION 'lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v2_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
  RAISE EXCEPTION 'consumo lectura V2 divergente' USING ERRCODE='42501'; END IF;
 SELECT a.version,vv.agregado_json->>'organizacion_ref' INTO v,org
  FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral vv
   ON vv.expediente_ref=a.expediente_ref AND vv.version=a.version WHERE a.expediente_ref=s->>'ExpedienteRef' FOR KEY SHARE OF a;
 IF NOT FOUND OR org IS DISTINCT FROM s->>'OrganizacionRef' THEN
  RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,
   'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 SELECT * INTO exacta FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
  IF exacta.documento IS DISTINCT FROM s->>'Documento' OR exacta.expediente_version IS DISTINCT FROM (s->>'VersionExpediente')::numeric
   OR exacta.paso_orden IS DISTINCT FROM (s->>'PasoOrden')::integer OR exacta.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella'
   OR exacta.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2' THEN
   RAISE EXCEPTION 'recuperación lectura V2 divergente' USING ERRCODE='P1181'; END IF;
 ELSE
  IF v IS DISTINCT FROM (s->>'VersionExpediente')::numeric THEN
   RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,'RevisionesPDF','[]'::jsonb,
    'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
 END IF;
 SELECT revision,huella_sha256 INTO cabeza FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef';
 IF NOT FOUND THEN cabeza.revision:=0; cabeza.huella_sha256:=vacia; END IF;
 IF exacta.firma_ref IS NOT NULL THEN cabeza.revision:=exacta.historia_revision_observada; cabeza.huella_sha256:=exacta.historia_huella_observada; END IF;
 IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 f WHERE f.organizacion_ref=s->>'OrganizacionRef'
   AND f.expediente_ref=s->>'ExpedienteRef' AND f.documento=s->>'Documento')>1000 THEN
  RAISE EXCEPTION 'historia V2 no representable' USING ERRCODE='P1525'; END IF;
 -- El lector privado común produce la proyección mínima, sin identidad de tercero.
 firmas:=vec_contratacion_temporal.consultar_firmas_documento_v3(s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'Documento',s->>'FirmantePrincipalCandidatoRef',NULL);
 -- Recuperar una operación histórica no revela actos posteriores al propio recibo.
 SELECT coalesce(jsonb_agg(x.value ORDER BY f.secuencia),'[]'::jsonb) INTO firmas
 FROM jsonb_array_elements(firmas) x JOIN vec_contratacion_temporal.firma_documento_v1 f ON f.firma_ref=x.value->>'FirmaRef'
 WHERE f.expediente_version<=(s->>'VersionExpediente')::numeric
   AND (exacta.firma_ref IS NULL OR f.secuencia<=exacta.secuencia);
 SELECT coalesce(jsonb_agg(x.value||jsonb_build_object('FirmanteRef',f.firmante_ref,'CertificadoHuella',f.certificado_huella_sha256,
   'FirmaAnteriorRef',r.firma_anterior_ref,'ReciboAnteriorRef',r.recibo_anterior_ref,
   'EntradaDocumentoRef',r.entrada_documento_ref,'EntradaDocumentoVersion',r.entrada_documento_version,
   'EntradaDocumentoHuella',r.entrada_huella_sha256,'EntradaDocumentoLongitud',r.entrada_longitud,
   'OrdenFirmaPDF',r.orden_pdf,'ByteRange',r.byte_range,'RevisionHuellaSHA256',r.revision_huella_sha256,
   'ContenidoFirmadoHuellaSHA256',r.contenido_firmado_huella_sha256,'RevisionLongitud',r.revision_longitud,
   'EvidenciaFirmasCanonica',r.evidencia_firmas_texto,'EvidenciaFirmasHuellaSHA256',r.evidencia_firmas_huella_sha256)
   ORDER BY f.secuencia),'[]'::jsonb) INTO revisiones
 FROM jsonb_array_elements(firmas) x JOIN vec_contratacion_temporal.firma_documento_v1 f ON f.firma_ref=x.value->>'FirmaRef'
 JOIN vec_contratacion_temporal.firma_documento_revision_pdf_v2 r ON r.firma_ref=f.firma_ref;
 SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND f.firmante_principal_ref=s->>'FirmantePrincipalCandidatoRef'),
  NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef' AND f.resultado='firmado'
   AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256) IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
   AND (f.firmante_principal_ref IS NULL OR f.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella')) INTO coincide,separacion;
 RETURN jsonb_build_object('Encontrado',true,'ExpedienteRef',s->>'ExpedienteRef','Firmas',firmas,'RevisionesPDF',revisiones,
  'HistoriaRevision',cabeza.revision,'HistoriaHuella',cabeza.huella_sha256,
  'CoincideFirmanteEnOtroPaso',coincide,'HistoriaSeparacionAcreditada',separacion);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'consulta V2 transitoria' USING ERRCODE='P1525';
WHEN data_exception THEN
 RAISE EXCEPTION 'material de consulta V2 inválido' USING ERRCODE='22023';
END $f$;

DO $acl$
DECLARE f regprocedure; a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
  WHERE c.oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_revision_pdf_v2 FROM %I',pg_get_userbyid(a.grantee)); END LOOP;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) x
  WHERE t.oid=(SELECT reltype FROM pg_class WHERE oid='vec_contratacion_temporal.firma_documento_revision_pdf_v2'::regclass)
   AND x.grantee<>0 AND x.grantee<>t.typowner LOOP
  EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_revision_pdf_v2 FROM %I',pg_get_userbyid(a.grantee)); END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.impedir_degradacion_firma_pdf_v2()'::regprocedure,
  'vec_contratacion_temporal.comprobar_hija_firma_pdf_v2()'::regprocedure] LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee)); END LOOP;
  IF f::text LIKE 'vec_contratacion_temporal.registrar_firma_verificada_v2(%' OR f::text LIKE 'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(%' THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f); END IF;
 END LOOP;
END $acl$;
COMMIT;
