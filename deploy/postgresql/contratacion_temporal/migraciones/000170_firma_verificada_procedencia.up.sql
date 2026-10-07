\set ON_ERROR_STOP on
-- CT170: registro de PDF firmado y verificado en servidor por dos vías:
-- certificado de VEC (el actor firma) o Portafirmas declarado por RRHH (otro
-- actor registra). La referencia y fecha de Portafirmas son DECLARADAS: este
-- registro no acredita envío ni respuesta de su servicio corporativo.
-- CT118/CT145 conservan su historia y sus funciones. La tabla común y sus
-- UNIQUE mantienen clave y secuencia únicas entre ambas vías.
-- Requiere CT145, AD3-156/157 y revalidación transaccional AUT30. Solo UP;
-- nunca retroetiquetar filas anteriores. AUT30 debe permanecer cerrada hasta
-- que la autoridad nominal de cargo/acto/unidad esté publicada.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000170', 0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
DECLARE f oid;
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario' OR getdatabaseencoding() <> 'UTF8'
    OR to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_custodia_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_auditoria_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_outbox_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_historia_cabeza_v1') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.registrar_firma_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
       'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)','EXECUTE')
    OR EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='vec_contratacion_temporal'
               AND table_name='firma_documento_v1' AND column_name='via_registro')
    OR to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.nueva_huella_firma_historia_v1(text,text,text,text,integer,text)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.avanzar_cabeza_firma_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'CT170: preimagen incompatible' USING ERRCODE='55000'; END IF;
 f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
    RAISE EXCEPTION 'CT170: AD3-156 requerido' USING ERRCODE='55000';
 END IF;
 f := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
    RAISE EXCEPTION 'CT170: AD3-157 requerido' USING ERRCODE='55000';
 END IF;
 f := to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
    RAISE EXCEPTION 'CT170: AD3-159 requerido' USING ERRCODE='55000';
 END IF;
 f := to_regprocedure('vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
    RAISE EXCEPTION 'CT170: AUT30 requerido' USING ERRCODE='55000';
 END IF;
END $pre$;
ALTER TABLE vec_contratacion_temporal.firma_documento_v1
 ADD COLUMN via_registro text,
 ADD COLUMN historia_revision_observada numeric(20,0),
 ADD COLUMN historia_huella_observada text,
 ADD COLUMN original_documento_ref text,
 ADD COLUMN original_documento_version numeric(20,0),
 ADD COLUMN firmante_principal_ref text,
 ADD COLUMN perfil_firmante_ref text,
 ADD COLUMN cargo_firmante text,
 ADD COLUMN unidad_firmante_ref text,
 ADD COLUMN perfil_activo_firmante_ref text,
 ADD COLUMN puesto_firmante_ref text,
 ADD COLUMN ambito_firmante_ref text,
 ADD COLUMN asignacion_firmante_ref text,
 ADD COLUMN asignacion_firmante_version numeric(20,0),
 ADD COLUMN asignacion_firmante_huella_sha256 text,
 ADD COLUMN version_rol_firmante_ref text,
 ADD COLUMN version_rol_firmante_huella_sha256 text,
 ADD COLUMN control_vigencia_firmante_ref text,
 ADD COLUMN control_vigencia_firmante_revision numeric(20,0),
 ADD COLUMN control_vigencia_firmante_huella_sha256 text,
 ADD COLUMN asignacion_vigente_desde text,
 ADD COLUMN asignacion_vigente_hasta text,
 ADD COLUMN acto_competencia_ref text,
 ADD COLUMN delegacion_ref text,
 ADD COLUMN referencia_portafirmas_declarada text,
 ADD COLUMN fecha_portafirmas_declarada text;
COMMENT ON COLUMN vec_contratacion_temporal.firma_documento_v1.via_registro IS
 'NULL histórico: firma de prueba CT118/145, sin eficacia administrativa. certificado_vec: firma del actor validada y ligada al original. portafirmas_registro_rrhh: PDF externo verificado y registrado por otro actor RRHH, con referencia/fecha declaradas.';
COMMENT ON COLUMN vec_contratacion_temporal.firma_documento_v1.referencia_portafirmas_declarada IS
 'Referencia declarada por RRHH; no prueba envío, recepción ni estado de Portafirmas.';
COMMENT ON COLUMN vec_contratacion_temporal.firma_documento_v1.fecha_portafirmas_declarada IS
 'Fecha declarada por RRHH, conservada literalmente en el material canónico; no es sello de tiempo acreditado.';
ALTER TABLE vec_contratacion_temporal.firma_documento_v1
 ADD CONSTRAINT firma_documento_v1_procedencia_externa_check CHECK (
 (via_registro IS NULL AND historia_revision_observada IS NULL AND historia_huella_observada IS NULL
  AND original_documento_ref IS NULL AND original_documento_version IS NULL
  AND firmante_principal_ref IS NULL AND perfil_firmante_ref IS NULL AND cargo_firmante IS NULL
  AND unidad_firmante_ref IS NULL AND perfil_activo_firmante_ref IS NULL
  AND puesto_firmante_ref IS NULL AND ambito_firmante_ref IS NULL
  AND asignacion_firmante_ref IS NULL AND asignacion_firmante_version IS NULL
  AND asignacion_firmante_huella_sha256 IS NULL AND version_rol_firmante_ref IS NULL
  AND version_rol_firmante_huella_sha256 IS NULL AND control_vigencia_firmante_ref IS NULL
  AND control_vigencia_firmante_revision IS NULL AND control_vigencia_firmante_huella_sha256 IS NULL
  AND asignacion_vigente_desde IS NULL
  AND asignacion_vigente_hasta IS NULL
  AND acto_competencia_ref IS NULL AND delegacion_ref IS NULL
  AND referencia_portafirmas_declarada IS NULL AND fecha_portafirmas_declarada IS NULL)
 OR
 (via_registro IS NOT NULL AND via_registro IN ('certificado_vec','portafirmas_registro_rrhh')
  AND resultado = 'firmado'
  AND historia_revision_observada IS NOT NULL AND historia_huella_observada IS NOT NULL
  AND historia_revision_observada BETWEEN 0 AND 9007199254740991::numeric
  AND historia_huella_observada ~ '^[0-9a-f]{64}$'
  AND historia_huella_observada <> repeat('0',64)
  AND num_nonnulls(original_documento_ref,original_documento_version,firmante_principal_ref,
      perfil_firmante_ref,cargo_firmante,unidad_firmante_ref,perfil_activo_firmante_ref,
      asignacion_firmante_ref,asignacion_firmante_version,asignacion_firmante_huella_sha256,
      version_rol_firmante_ref,version_rol_firmante_huella_sha256,control_vigencia_firmante_ref,
      control_vigencia_firmante_revision,control_vigencia_firmante_huella_sha256,
      asignacion_vigente_desde,asignacion_vigente_hasta) = 17
  AND original_documento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND original_documento_version BETWEEN 1 AND 9007199254740991::numeric
  AND firmante_ref ~ '^ref:[0-9a-f]{64}$'
  AND firmante_ref = 'ref:'||certificado_huella_sha256
  AND firmante_principal_ref ~ '^per_[A-Za-z0-9_-]{2,159}$'
  AND actor_ref ~ '^per_[A-Za-z0-9_-]{2,159}$'
  AND perfil_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND cargo_firmante IS NOT NULL AND char_length(cargo_firmante) BETWEEN 1 AND 256
  AND cargo_firmante !~ '[[:cntrl:]]' AND btrim(cargo_firmante) = cargo_firmante
  AND unidad_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND perfil_activo_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND (puesto_firmante_ref IS NULL OR puesto_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
  AND (ambito_firmante_ref IS NULL OR ambito_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
  AND asignacion_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND asignacion_firmante_version BETWEEN 1 AND 9007199254740991::numeric
  AND asignacion_firmante_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND asignacion_firmante_huella_sha256 <> repeat('0',64)
  AND version_rol_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND version_rol_firmante_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND version_rol_firmante_huella_sha256 <> repeat('0',64)
  AND control_vigencia_firmante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND control_vigencia_firmante_ref = version_rol_firmante_ref
  AND control_vigencia_firmante_revision BETWEEN 1 AND 9007199254740991::numeric
  AND control_vigencia_firmante_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND control_vigencia_firmante_huella_sha256 <> repeat('0',64)
  AND asignacion_vigente_desde IS NOT NULL AND asignacion_vigente_hasta IS NOT NULL
  AND (acto_competencia_ref IS NULL OR acto_competencia_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
  AND (delegacion_ref IS NULL OR delegacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
  AND ((via_registro = 'certificado_vec' AND actor_ref = firmante_principal_ref
        AND referencia_portafirmas_declarada IS NULL AND fecha_portafirmas_declarada IS NULL)
       OR (via_registro = 'portafirmas_registro_rrhh' AND actor_ref <> firmante_principal_ref
        AND referencia_portafirmas_declarada IS NOT NULL
        AND char_length(referencia_portafirmas_declarada) BETWEEN 1 AND 256
        AND referencia_portafirmas_declarada !~ '[[:cntrl:]]'
        AND btrim(referencia_portafirmas_declarada) = referencia_portafirmas_declarada
        AND fecha_portafirmas_declarada IS NOT NULL))));
-- Cabeza global del expediente: abarca todos los documentos y también los
-- INSERT legados CT118/145. El hash vacío es fijo; cada fila añade a la
-- cadena su firma/recibo/documento/secuencia/huella de material. El trigger
-- avanza la revisión en la misma transacción que cualquier registro.
CREATE TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 (
 organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 expediente_ref text NOT NULL CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 revision numeric(20,0) NOT NULL CHECK (revision BETWEEN 0 AND 9007199254740991::numeric),
 huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$' AND huella_sha256 <> repeat('0',64)),
 PRIMARY KEY (organizacion_ref,expediente_ref)
);
COMMENT ON TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 IS
 'CAS global del historial de firmas del expediente, todas las vías y documentos. No sustituye la historia inmutable.';
ALTER TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.firma_historia_cabeza_v1
 TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.firma_historia_cabeza_v1 FROM PUBLIC;
DO $acl_cabeza$
DECLARE a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,
  LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
  WHERE c.oid='vec_contratacion_temporal.firma_historia_cabeza_v1'::regclass
    AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.firma_historia_cabeza_v1 FROM %I',pg_get_userbyid(a.grantee));
 END LOOP;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_type t,
  LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) x
  WHERE t.oid=(SELECT c.reltype FROM pg_class c
    WHERE c.oid='vec_contratacion_temporal.firma_historia_cabeza_v1'::regclass)
    AND x.grantee<>0 AND x.grantee<>t.typowner LOOP
   EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.firma_historia_cabeza_v1 FROM %I',pg_get_userbyid(a.grantee));
 END LOOP;
END $acl_cabeza$;
CREATE FUNCTION vec_contratacion_temporal.nueva_huella_firma_historia_v1(
 p_anterior text,p_firma_ref text,p_recibo_ref text,p_documento text,p_secuencia integer,p_material_huella text
) RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(decode(p_anterior,'hex')||convert_to(
  jsonb_build_array(p_firma_ref,p_recibo_ref,p_documento,p_secuencia,p_material_huella)::text,'UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.nueva_huella_firma_historia_v1(text,text,text,text,integer,text) FROM PUBLIC;
-- CT118/145 ya puede tener historia. Se deriva la cabeza sin cambiar esas
-- filas y bajo el ACCESS EXCLUSIVE que mantiene el ALTER de la misma tabla.
DO $preexistente$
DECLARE f record; vacia text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
BEGIN
 FOR f IN SELECT organizacion_ref,expediente_ref,firma_ref,recibo_ref,documento,secuencia,solicitud_huella_sha256
   FROM vec_contratacion_temporal.firma_documento_v1
   ORDER BY organizacion_ref,expediente_ref,registrada_en,firma_ref LOOP
   INSERT INTO vec_contratacion_temporal.firma_historia_cabeza_v1 VALUES
    (f.organizacion_ref,f.expediente_ref,0,vacia)
   ON CONFLICT (organizacion_ref,expediente_ref) DO NOTHING;
   UPDATE vec_contratacion_temporal.firma_historia_cabeza_v1 h SET
     revision=h.revision+1,
     huella_sha256=vec_contratacion_temporal.nueva_huella_firma_historia_v1(
       h.huella_sha256,f.firma_ref,f.recibo_ref,f.documento,f.secuencia,f.solicitud_huella_sha256)
   WHERE h.organizacion_ref=f.organizacion_ref AND h.expediente_ref=f.expediente_ref;
 END LOOP;
END $preexistente$;
CREATE FUNCTION vec_contratacion_temporal.avanzar_cabeza_firma_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $f$
DECLARE vacia text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
BEGIN
 IF TG_OP <> 'INSERT' OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
    OR TG_TABLE_NAME <> 'firma_documento_v1' THEN
   RAISE EXCEPTION 'avance de cabeza de firma denegado' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_contratacion_temporal.firma_historia_cabeza_v1 VALUES
  (NEW.organizacion_ref,NEW.expediente_ref,0,vacia)
 ON CONFLICT (organizacion_ref,expediente_ref) DO NOTHING;
 UPDATE vec_contratacion_temporal.firma_historia_cabeza_v1 h SET
   revision=h.revision+1,
   huella_sha256=vec_contratacion_temporal.nueva_huella_firma_historia_v1(
     h.huella_sha256,NEW.firma_ref,NEW.recibo_ref,NEW.documento,NEW.secuencia,NEW.solicitud_huella_sha256)
 WHERE h.organizacion_ref=NEW.organizacion_ref AND h.expediente_ref=NEW.expediente_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'cabeza de firma no avanzada' USING ERRCODE='55000'; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.avanzar_cabeza_firma_v1() FROM PUBLIC;
CREATE TRIGGER cabeza_global_ai AFTER INSERT ON vec_contratacion_temporal.firma_documento_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.avanzar_cabeza_firma_v1();
-- El evento antiguo permanece para firmas de prueba. Una firma externa lleva
-- su propia acción de auditoría y evento, inequívocos para consumidores.
ALTER TABLE vec_contratacion_temporal.firma_documento_auditoria_v1
 DROP CONSTRAINT firma_documento_auditoria_v1_operacion_check,
 ADD CONSTRAINT firma_documento_auditoria_v1_operacion_check CHECK
 (operacion IN ('contratacion_temporal.documento.firmar','contratacion_temporal.documento.firma_externa.registrar',
                'contratacion_temporal.documento.firma_vec.registrar'));
ALTER TABLE vec_contratacion_temporal.firma_documento_outbox_v1
 DROP CONSTRAINT firma_documento_outbox_v1_tipo_check,
 ADD CONSTRAINT firma_documento_outbox_v1_tipo_check CHECK
 (tipo IN ('contratacion_temporal.documento.firmado','contratacion_temporal.documento.devuelto',
           'contratacion_temporal.documento.firma_externa_registrada',
           'contratacion_temporal.documento.firma_vec_registrada'));
CREATE FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v1(
 p_solicitud text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE s jsonb; d jsonb; consumo record; previa record; enlace record; cabeza record;
 h text; contexto_h text; recurso text; v numeric; org text; siguiente integer;
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
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 16384 THEN
    RAISE EXCEPTION 'material de firma verificada inválido' USING ERRCODE='22023'; END IF;
 IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
    RAISE EXCEPTION 'decisión de firma verificada inválida' USING ERRCODE='42501'; END IF;
 s := p_solicitud::jsonb;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_solicitud::json)) <> (SELECT count(*) FROM jsonb_each(s))
    OR s->>'Via' NOT IN ('certificado_vec','portafirmas_registro_rrhh')
 THEN RAISE EXCEPTION 'material de firma verificada inválido' USING ERRCODE='22023'; END IF;
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
      'ClaveIdempotencia','DocumentoCustodiaRef','DocumentoCustodiaVersion']) IS NOT TRUE)
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
      'DocumentoCustodiaVersion']) IS NOT TRUE)
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
 FOREACH k IN ARRAY ARRAY['PuestoFirmanteRef','AmbitoFirmanteRef','ActoCompetenciaRef','DelegacionRef'] LOOP
   IF jsonb_typeof(s->k) IS NULL OR jsonb_typeof(s->k) NOT IN ('string','null') THEN
      RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['VersionExpediente','PasoOrden','Secuencia','OriginalVersion',
  'AsignacionFirmanteVersion','ControlVigenciaFirmanteRevision','DocumentoCustodiaVersion'] LOOP
   IF jsonb_typeof(s->k) IS DISTINCT FROM 'number' OR (s->>k) !~ '^[1-9][0-9]{0,15}$'
      OR (s->>k)::numeric > 9007199254740991::numeric THEN
      RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF s->>'PoliticaVerificacion' <> 'politica:vec:firma:verificacion-autonoma:v1'
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
 h := encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
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
   '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
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
    OR (s->>'Via'='certificado_vec' AND d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoFirmanteRef')
    OR (s->>'Via'='certificado_vec' AND d->>'version_rol_ref' IS DISTINCT FROM s->>'VersionRolFirmanteRef')
    OR (s->>'Via'='portafirmas_registro_rrhh' AND d->>'principal_id' IS NOT DISTINCT FROM s->>'FirmantePrincipalRef')
    OR (s->>'Via'='portafirmas_registro_rrhh' AND d->>'version_rol_ref' IS DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1')
    OR coalesce(d->>'perfil_activo_ref','') = ''
 THEN RAISE EXCEPTION 'autorización de firma verificada divergente' USING ERRCODE='42501'; END IF;
 IF s->>'Via'='certificado_vec' THEN
   SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
     p_solicitud,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
     p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
   SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_firma_externa_ct_v3_atestada(
     p_solicitud,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
     p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF consumo.efecto_ref IS DISTINCT FROM recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
    OR consumo.consumo_nuevo IS NOT TRUE THEN
    RAISE EXCEPTION 'consumo de firma verificada divergente' USING ERRCODE='42501'; END IF;
 -- AUT30 comprueba la asignación, el control de vigencia y la competencia
 -- nominal actual bajo bloqueo. Una instantánea leída antes de custodiar el
 -- PDF no autoriza un COMMIT. La misma guarda se ejecuta en un replay.
 IF vec_autorizacion.revalidar_competencia_firmante_ct_v1(p_solicitud) IS NOT TRUE THEN
    RAISE EXCEPTION 'competencia del firmante no acreditada al confirmar' USING ERRCODE='42501'; END IF;
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
   SELECT * INTO enlace FROM vec_contratacion_temporal.firma_documento_custodia_v1
    WHERE firma_ref=previa.firma_ref;
   IF NOT FOUND OR enlace.documento_ref IS DISTINCT FROM s->>'DocumentoCustodiaRef'
      OR enlace.documento_version IS DISTINCT FROM (s->>'DocumentoCustodiaVersion')::numeric THEN
      RAISE EXCEPTION 'enlace de custodia de firma inconsistente' USING ERRCODE='55000'; END IF;
   RETURN jsonb_build_object('FirmaRef',previa.firma_ref,'ReciboRef',previa.recibo_ref,
      'Secuencia',previa.secuencia,'Resultado',previa.resultado,'ExpedienteVersion',previa.expediente_version,
      'ActorRef',previa.actor_ref,'PerfilRef',previa.perfil_ref,'RegistradaEn',previa.registrada_en,
      'SolicitudHuella',previa.solicitud_huella_sha256,'YaRegistrada',true,
      'DocumentoCustodiaRef',enlace.documento_ref,'DocumentoCustodiaVersion',enlace.documento_version);
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
 IF (s->>'PasoOrden')::integer > 1 AND NOT EXISTS (
   SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
   WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
     AND f.documento=s->>'Documento' AND f.resultado='firmado'
     AND f.via_registro IN ('certificado_vec','portafirmas_registro_rrhh')
     AND f.firmante_principal_ref IS NOT NULL
     AND f.catalogo_huella_sha256=s->>'CatalogoHuella'
     AND f.paso_orden=(s->>'PasoOrden')::integer-1
     AND f.original_documento_ref=s->>'OriginalRef'
     AND f.original_documento_version=(s->>'OriginalVersion')::numeric
     AND f.original_huella_sha256=s->>'OriginalHuella') THEN
    RAISE EXCEPTION 'la firma no continúa el paso anterior' USING ERRCODE='P1184'; END IF;
 ahora := date_trunc('microseconds',clock_timestamp());
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
   'DocumentoCustodiaVersion',(s->>'DocumentoCustodiaVersion')::numeric);
EXCEPTION WHEN lock_not_available THEN
 RAISE EXCEPTION 'registro de firma externa no disponible' USING ERRCODE='P1185';
WHEN data_exception THEN
 RAISE EXCEPTION 'material de firma externa inválido' USING ERRCODE='22023';
END $f$;
-- CT152 mantiene v2 sin acceso técnico directo. La proyección v3 es solo
-- interna y agrega procedencia explícita: NULL histórico nunca implica
-- intervención de Portafirmas ni competencia acreditada.
CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_documento_v3(
 p_organizacion_ref text,p_expediente_ref text,p_documento text,p_firmante_candidato_ref text,
 p_firma_ref text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
AS $f$
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
    RAISE EXCEPTION 'consulta de firmas denegada' USING ERRCODE='42501'; END IF;
 IF p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_documento IS NULL OR p_documento !~ '^[a-z][a-z0-9_]{1,63}$'
    OR p_firmante_candidato_ref IS NULL
    OR p_firmante_candidato_ref !~ '^per_[A-Za-z0-9_-]{2,159}$'
    OR (p_firma_ref IS NOT NULL AND p_firma_ref !~ '^firma-ct:[0-9a-f-]{36}$') THEN
    RAISE EXCEPTION 'consulta de firmas inválida' USING ERRCODE='22023'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
   JOIN vec_contratacion_temporal.expediente_version_integral v
     ON v.expediente_ref=a.expediente_ref AND v.version=a.version
   WHERE a.expediente_ref=p_expediente_ref
     AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref) THEN
    RAISE EXCEPTION 'consulta de firmas denegada' USING ERRCODE='42501'; END IF;
 RETURN coalesce((SELECT jsonb_agg(jsonb_build_object(
   'FirmaRef',f.firma_ref,'ReciboRef',f.recibo_ref,'Documento',f.documento,'Secuencia',f.secuencia,
   'ExpedienteVersion',f.expediente_version,'CatalogoRef',f.catalogo_ref,'CatalogoHuella',f.catalogo_huella_sha256,
   'PasoRef',f.paso_ref,'PasoOrden',f.paso_orden,'Resultado',f.resultado,
   'ConMotivoDevolucion',f.motivo_devolucion IS NOT NULL,'OriginalHuella',f.original_huella_sha256,
   'FirmadoHuella',f.firmado_huella_sha256,'SelloTiempoEstado',f.sello_tiempo_estado,
   'RegistradaEn',f.registrada_en,'ClaveIdempotencia',f.clave_idempotencia,
   'FirmantePrincipalAcreditado',f.firmante_principal_ref IS NOT NULL,
   'CoincideFirmanteCandidato',f.firmante_principal_ref IS NOT NULL
      AND f.firmante_principal_ref=p_firmante_candidato_ref,
   'HistoriaRevision',f.historia_revision_observada,'HistoriaHuella',f.historia_huella_observada,
   'DocumentoCustodiaRef',c.documento_ref,'DocumentoCustodiaVersion',c.documento_version,
   'Via',f.via_registro,'OriginalRef',f.original_documento_ref,
   'OriginalVersion',f.original_documento_version,
   'ReferenciaPortafirmasDeclarada',f.referencia_portafirmas_declarada,
   'FechaPortafirmasDeclarada',f.fecha_portafirmas_declarada)
   ORDER BY f.documento,f.secuencia)
   FROM (SELECT * FROM vec_contratacion_temporal.firma_documento_v1
     WHERE organizacion_ref=p_organizacion_ref AND expediente_ref=p_expediente_ref
       AND documento=p_documento
       AND (p_firma_ref IS NULL OR firma_ref=p_firma_ref)
     ORDER BY secuencia LIMIT 1000) f
   LEFT JOIN vec_contratacion_temporal.firma_documento_custodia_v1 c ON c.firma_ref=f.firma_ref),'[]'::jsonb);
END $f$;
-- CT170: lector R5 nominal. La función interna v3 permanece privada; AD159
-- consume una concesión nueva y audita cada lectura, incluso la ausencia.
CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE s jsonb; c jsonb; d jsonb; consumo record; firmas jsonb; exacta record;
 h text; contexto_h text; version_actual numeric; org text; total bigint;
 cabeza_revision numeric; cabeza_huella text; filtro_firma_ref text;
 coincide_otro boolean:=false; separacion_acreditada boolean:=false;
 vacia constant text:=encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex');
 campos_fila constant jsonb:='["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb;
 campos_permiso constant jsonb:='["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb;
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off' THEN
    RAISE EXCEPTION 'consulta R5 denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 1024 THEN
    RAISE EXCEPTION 'material de consulta R5 inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'material de consulta R5 inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_material::json)) <> 8
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s,ARRAY[
       'OrganizacionRef','ExpedienteRef','VersionExpediente','Documento',
       'FirmantePrincipalCandidatoRef','ClaveIdempotencia','PasoOrden','CatalogoHuella']) IS NOT TRUE
    OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'VersionExpediente') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'Documento') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'FirmantePrincipalCandidatoRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'PasoOrden') IS DISTINCT FROM 'number'
    OR jsonb_typeof(s->'CatalogoHuella') IS DISTINCT FROM 'string'
    OR (s->>'OrganizacionRef') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'ExpedienteRef') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'VersionExpediente') !~ '^[1-9][0-9]{0,15}$'
    OR (s->>'VersionExpediente')::numeric > 9007199254740991::numeric
    OR (s->>'Documento') !~ '^[a-z][a-z0-9_]{1,63}$'
    OR (s->>'FirmantePrincipalCandidatoRef') !~ '^per_[A-Za-z0-9_-]{2,159}$'
    OR (s->>'ClaveIdempotencia') !~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
    OR (s->>'PasoOrden') !~ '^([1-9]|1[0-6])$'
    OR (s->>'CatalogoHuella') !~ '^[0-9a-f]{64}$'
    OR (s->>'CatalogoHuella') = repeat('0',64)
    OR p_material IS DISTINCT FROM '{"OrganizacionRef":'||to_json(s->>'OrganizacionRef')::text||
      ',"ExpedienteRef":'||to_json(s->>'ExpedienteRef')::text||
      ',"VersionExpediente":'||(s->>'VersionExpediente')||
      ',"Documento":'||to_json(s->>'Documento')::text||
      ',"FirmantePrincipalCandidatoRef":'||to_json(s->>'FirmantePrincipalCandidatoRef')::text||
      ',"ClaveIdempotencia":'||to_json(s->>'ClaveIdempotencia')::text||
      ',"PasoOrden":'||(s->>'PasoOrden')||
      ',"CatalogoHuella":'||to_json(s->>'CatalogoHuella')::text||'}' THEN
    RAISE EXCEPTION 'material de consulta R5 no canónico' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
    OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR octet_length(p_raiz) <> 44
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
    OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version <> trunc(p_persona_version)
    OR p_perfil_version <> trunc(p_perfil_version) THEN
    RAISE EXCEPTION 'capacidad de consulta R5 inválida' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'capacidad de consulta R5 inválida' USING ERRCODE='42501'; END;
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR d->'campos_permitidos' IS DISTINCT FROM campos_permiso
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
    RAISE EXCEPTION 'autorización de consulta R5 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h THEN
    RAISE EXCEPTION 'consumo de consulta R5 divergente' USING ERRCODE='42501'; END IF;
 SELECT a.version,v.agregado_json->>'organizacion_ref' INTO version_actual,org
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=s->>'ExpedienteRef';
 IF NOT FOUND OR org IS DISTINCT FROM s->>'OrganizacionRef' THEN
   RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb,
     'HistoriaRevision',0,'HistoriaHuella',vacia,'CoincideFirmanteEnOtroPaso',false,
     'HistoriaSeparacionAcreditada',false); END IF;
 IF (s->>'VersionExpediente')::numeric > version_actual OR NOT EXISTS (
    SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral v
    WHERE v.expediente_ref=s->>'ExpedienteRef' AND v.version=(s->>'VersionExpediente')::numeric
      AND v.agregado_json->>'organizacion_ref'=s->>'OrganizacionRef') THEN
   RAISE EXCEPTION 'versión del expediente en conflicto' USING ERRCODE='P1182'; END IF;
 SELECT * INTO exacta FROM vec_contratacion_temporal.firma_documento_v1
 WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
   AND clave_idempotencia=s->>'ClaveIdempotencia';
 IF FOUND THEN
   IF exacta.documento IS DISTINCT FROM s->>'Documento'
      OR exacta.expediente_version IS DISTINCT FROM (s->>'VersionExpediente')::numeric
      OR exacta.via_registro IS NULL
      OR exacta.via_registro NOT IN ('certificado_vec','portafirmas_registro_rrhh') THEN
     RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef',
       'Firmas','[]'::jsonb,'HistoriaRevision',0,'HistoriaHuella',vacia,
       'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
   filtro_firma_ref:=exacta.firma_ref;
   cabeza_revision:=exacta.historia_revision_observada;
   cabeza_huella:=exacta.historia_huella_observada;
 ELSE
   IF (s->>'VersionExpediente')::numeric<>version_actual THEN
     RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef',
       'Firmas','[]'::jsonb,'HistoriaRevision',0,'HistoriaHuella',vacia,
       'CoincideFirmanteEnOtroPaso',false,'HistoriaSeparacionAcreditada',false); END IF;
   SELECT revision,huella_sha256 INTO cabeza_revision,cabeza_huella
   FROM vec_contratacion_temporal.firma_historia_cabeza_v1
   WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef';
   IF NOT FOUND THEN cabeza_revision:=0; cabeza_huella:=vacia; END IF;
   SELECT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
     WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
       AND f.resultado='firmado' AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256)
         IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
       AND f.firmante_principal_ref=s->>'FirmantePrincipalCandidatoRef'),
     NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
     WHERE f.organizacion_ref=s->>'OrganizacionRef' AND f.expediente_ref=s->>'ExpedienteRef'
       AND f.resultado='firmado' AND ROW(f.documento,f.paso_orden,f.catalogo_huella_sha256)
         IS DISTINCT FROM ROW(s->>'Documento',(s->>'PasoOrden')::integer,s->>'CatalogoHuella')
       AND (f.via_registro IS NULL OR f.firmante_principal_ref IS NULL
         OR f.catalogo_huella_sha256 IS DISTINCT FROM s->>'CatalogoHuella'))
   INTO coincide_otro,separacion_acreditada;
 END IF;
 SELECT count(*) INTO total FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
    AND documento=s->>'Documento' AND (filtro_firma_ref IS NULL OR firma_ref=filtro_firma_ref);
 IF total>1000 THEN RAISE EXCEPTION 'historia de firmas no representable' USING ERRCODE='P1525'; END IF;
 firmas:=vec_contratacion_temporal.consultar_firmas_documento_v3(
  s->>'OrganizacionRef',s->>'ExpedienteRef',s->>'Documento',s->>'FirmantePrincipalCandidatoRef',
  filtro_firma_ref);
 IF jsonb_typeof(firmas) IS DISTINCT FROM 'array' OR jsonb_array_length(firmas)<>total
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(firmas) x
      WHERE jsonb_typeof(x) IS DISTINCT FROM 'object'
         OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(x,ARRAY(
            SELECT jsonb_array_elements_text(campos_fila))) IS NOT TRUE) THEN
    RAISE EXCEPTION 'proyección de firmas R5 incompatible' USING ERRCODE='P1525'; END IF;
 RETURN jsonb_build_object('Encontrado',true,'ExpedienteRef',s->>'ExpedienteRef','Firmas',firmas,
   'HistoriaRevision',cabeza_revision,'HistoriaHuella',cabeza_huella,
   'CoincideFirmanteEnOtroPaso',coincide_otro,'HistoriaSeparacionAcreditada',separacion_acreditada);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'consulta R5 transitoria' USING ERRCODE='P1525';
END $f$;
DO $acl$
DECLARE f regprocedure; a record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.nueva_huella_firma_historia_v1(text,text,text,text,integer,text)'::regprocedure,
  'vec_contratacion_temporal.avanzar_cabeza_firma_v1()'::regprocedure
 ] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
   FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
    LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
    WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
      EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
   END LOOP;
   IF f IN (
     'vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
     'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) THEN
     EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f);
   END IF;
 END LOOP;
 -- CT152 ya revocó la lectura técnica. v3 queda privada al propietario;
 -- solo el lector R5 nominal anterior recibe EXECUTE técnico.
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.consultar_firmas_documento_v3(text,text,text,text,text)','EXECUTE') THEN
   RAISE EXCEPTION 'CT170: lectura de historia sin V3 nominal' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
