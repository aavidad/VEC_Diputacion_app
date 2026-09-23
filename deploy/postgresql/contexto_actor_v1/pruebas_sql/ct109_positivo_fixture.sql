\set ON_ERROR_STOP on
-- Exclusivamente para la base vacía y desechable del runner CA6/CT109.
-- Siembra bajo cada propietario; conserva RLS, constraints y triggers activos.
-- Las huellas y referencias de esta semilla son sintéticas. La decisión y COSE
-- de consulta se emiten después mediante el helper Go real.
DO $precondiciones$
BEGIN
 IF current_setting('server_version_num')::integer<>180004
 OR current_setting('session_replication_role')<>'origin'
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta)
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.registro_acceso_rrhh)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1)
 THEN RAISE EXCEPTION 'CT109 positivo exige preimagen desechable vacía PostgreSQL 18.4'; END IF;
END $precondiciones$;
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $ca6$
DECLARE v vec_contexto_actor_v1.vinculo_corporativo_versiones%ROWTYPE;
BEGIN
 SELECT h.* INTO STRICT v
 FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones h
   ON h.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND h.version=a.version
 WHERE a.cuenta_ref='cta_corporativa_rrhh_000000000001';
 IF v.version<>5 THEN RAISE EXCEPTION 'CT109 positivo exige historia CA6 v5'; END IF;
 v.version:=6;
 v.estado:='activo';
 v.vigente_desde:=clock_timestamp()-interval '1 minute';
 v.vigente_hasta:=clock_timestamp()+interval '1 hour';
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones SELECT v.*;
 UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=6
 WHERE cuenta_ref=v.cuenta_ref;
END $ca6$;
COMMIT;
CREATE TABLE public.ct109_positivo_comprobante AS
 SELECT vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
 'cta_corporativa_rrhh_000000000001',1,'per_corporativa_rrhh_000000000001',1,
 'prf_corporativo_rrhh_000000000001',1,'vca_corporativo_rrhh_000000000001',1) comprobante;
REVOKE ALL ON public.ct109_positivo_comprobante FROM PUBLIC;
DO $comprobante$
BEGIN
 IF (SELECT comprobante->>'vinculo_corporativo_version'
 FROM public.ct109_positivo_comprobante) IS DISTINCT FROM '6'
 THEN RAISE EXCEPTION 'CT109 positivo sin comprobante CA6 nuevo'; END IF;
END $comprobante$;

-- Identidad durable: cuenta activa, sesión, control vigente y consumo de aserción.
-- El helper Go lee estas filas y usa exactamente sus tiempos y huellas.
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
INSERT INTO vec_identidad_sesiones_v1.cuenta VALUES
 ('cta_corporativa_rrhh_000000000001',false,NULL,clock_timestamp(),
  'opr_ct109_positivo_00000000000001');
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta VALUES
 ('cta_corporativa_rrhh_000000000001',1,'activa',clock_timestamp(),
  'opr_ct109_positivo_00000000000001');
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual VALUES
 ('cta_corporativa_rrhh_000000000001',1,clock_timestamp(),
  'opr_ct109_positivo_00000000000001');
DO $sesion$
DECLARE t timestamptz:=clock_timestamp()-interval '2 seconds';
BEGIN
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1
 (sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
 cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,
 metodo_observado,garantia_observada,politica_garantia_ref,
 politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en)
 VALUES ('ses_ct109_positivo_00000000000001','aut_ct109_positivo_00000000000001',
 repeat('1',64),'ase_ct109_positivo_00000000000001',
 'cta_corporativa_rrhh_000000000001','cta_corporativa_rrhh_000000000001',false,
 'interna_corporativa','kerberos_ad','alto','pga_ct109_positivo_00000000000001',
 repeat('2',64),t,t+interval '1 second');
 INSERT INTO vec_autorizacion.control_sesion_v1
 (control_sesion_ref,revision,sesion_ref,estado,huella_sha256,
 sesion_revalidada_en,sesion_valida_hasta)
 VALUES ('cse_ct109_positivo_00000000000001',1,'ses_ct109_positivo_00000000000001',
 'activa',repeat('3',64),t+interval '1 second',t+interval '10 minutes');
 INSERT INTO vec_autorizacion.control_sesion_actual_v1 VALUES
 ('ses_ct109_positivo_00000000000001','cse_ct109_positivo_00000000000001',1,
 t+interval '1 second','opr_ct109_positivo_00000000000001');
 INSERT INTO vec_identidad_sesiones_v1.consumo_asercion VALUES
 ('opr_ct109_positivo_00000000000002','vec.identidad.hmac-sha256.v1',
 'idh_ct109_positivo_00000000000001','clave:ct109:prueba',1,
 decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),
 decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),NULL,
 'aut_ct109_positivo_00000000000001',repeat('1',64),
 'ase_ct109_positivo_00000000000001','ses_ct109_positivo_00000000000001',
 'cse_ct109_positivo_00000000000001',1,'cta_corporativa_rrhh_000000000001',1,
 'cta_corporativa_rrhh_000000000001',1,t+interval '1 second');
END $sesion$;
COMMIT;
CREATE TABLE public.ct109_positivo_metadatos AS
 SELECT jsonb_build_object(
 'expediente_ref','expediente:ct109:positivo','organizacion_ref','org_diputaciondemo0001',
 'cuenta_ref',s.cuenta_ref,'persona_ref','per_corporativa_rrhh_000000000001',
 'perfil_ref','prf_corporativo_rrhh_000000000001',
 'contexto_ref','vca_corporativo_rrhh_000000000001',
 'sesion_ref',s.sesion_ref,'autenticacion_ref',s.autenticacion_ref,
 'sesion',to_jsonb(s),'control',to_jsonb(c)) datos
 FROM vec_autorizacion.sesion_autenticacion_v1 s
 JOIN vec_autorizacion.control_sesion_v1 c USING (sesion_ref)
 WHERE s.sesion_ref='ses_ct109_positivo_00000000000001';
REVOKE ALL ON public.ct109_positivo_metadatos FROM PUBLIC;

-- Alta sintética completa. Sus FK diferidas se comprueban al COMMIT.
-- Los triggers materializan el agregado y la publicación RRHH reales.
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.control_publicacion_rrhh
 VALUES(true,0,0,clock_timestamp(),clock_timestamp());
-- Génesis CT39, derivada del punto cero de accesos ya sembrado por el runner.
INSERT INTO vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
 (control,version_esquema,secuencia_base,cabeza_base_sha256,creada_en,
 prueba_canonica,prueba_huella_sha256)
 SELECT true,1,b.ultima_secuencia,b.cabeza_sha256,r.t,p.canon,
 encode(sha256(p.canon),'hex')
 FROM vec_contratacion_temporal.control_cadena_accesos_rrhh b
 CROSS JOIN LATERAL (SELECT clock_timestamp() t) r
 CROSS JOIN LATERAL (SELECT convert_to(
 'VEC-CT-CONTROL-REGISTRADOR-ACCESO-RRHH-V2'||chr(10),'UTF8')
 ||vec_contratacion_temporal.encuadrar_texto_v1('1')
 ||vec_contratacion_temporal.encuadrar_texto_v1(b.ultima_secuencia::text)
 ||vec_contratacion_temporal.encuadrar_texto_v1(b.cabeza_sha256)
 ||vec_contratacion_temporal.encuadrar_texto_v1(
 vec_contratacion_temporal.instante_utc_v1(r.t)) canon) p
 WHERE b.control AND b.ultima_secuencia=0 AND b.cabeza_sha256=repeat('0',64);
DO $alta$
DECLARE
 t timestamptz:=clock_timestamp()-interval '1 minute';
 z text:=to_char(t,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 h text:=encode(sha256(convert_to('ct109:semilla:alta','UTF8')),'hex');
 cero text:=repeat('0',64);
 ambito text:='hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||h;
 peticion text:='hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'||h;
 cnf text:='cnf_ct_'||substr(h,1,32);
 a jsonb; canon bytea; canon_h text; solicitud_h text; payload bytea;
BEGIN
 a:=jsonb_build_object(
  'esquema','vec.contratacion_temporal.alta.v2',
  'reserva_ref','reserva:ct109:positivo','expediente_ref','expediente:ct109:positivo',
  'numero_visible','2026/CT109-POS','recibo_ref','recibo:ct109:positivo',
  'organizacion_ref','org_diputaciondemo0001',
  'actor_ref','per_corporativa_rrhh_000000000001',
  'perfil_ref','prf_corporativo_rrhh_000000000001','version',1,
  'flujo',jsonb_build_object('definicion_ref','flujo:ct109:positivo','version',1,'huella_sha256',h),
  'fase_actual','solicitud','estado_actual','en_curso',
  'solicitud',jsonb_build_object('centro_ref','centro:ct109:positivo',
   'contacto_ref','contacto:ct109:positivo','categoria_ref','categoria:ct109:positivo',
   'grupo_subgrupo','C2','motivo_clave','sustitucion','detalle','Solicitud sintética CT109',
   'periodo',jsonb_build_object('inicio','2026-10-01T00:00:00.000000Z','fin','2026-11-01T00:00:00.000000Z'),
   'rc',jsonb_build_object('existe',false,'numero','','fecha','',
    'importe',jsonb_build_object('centimos',0,'moneda','EUR'),'documento_ref',''),
   'documentos_adjuntos','[]'::jsonb,'observaciones',''),
  'creado_en',z,'actualizado_en',z,
  'actuacion',jsonb_build_object('secuencia',1,'version_expediente',1,
   'accion_clave','solicitud.registrar','actor_ref','per_corporativa_rrhh_000000000001',
   'unidad_ref','unidad:ct109:positivo','recibo_ref','recibo:ct109:positivo',
   'realizada_en',z,'fase_origen','','fase_destino','solicitud',
   'estado_origen','pendiente','estado_destino','en_curso','observaciones','',
   'documentos_ref','[]'::jsonb));
 canon:=vec_contratacion_temporal.reconstruir_efecto_alta_v2(a);
 canon_h:=encode(sha256(canon),'hex');
 solicitud_h:=encode(sha256(convert_to(
  vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(a->'solicitud'),'UTF8')),'hex');
 payload:=convert_to(jsonb_build_object('expediente_ref','expediente:ct109:positivo',
  'version',1,'tipo','semilla_sintetica_ct109')::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.identidad_reserva_alta VALUES
 (ambito,'reserva:ct109:positivo','expediente:ct109:positivo','2026/CT109-POS',
 'recibo:ct109:positivo',peticion,'org_diputaciondemo0001',
 'per_corporativa_rrhh_000000000001','prf_corporativo_rrhh_000000000001',t);
 INSERT INTO vec_contratacion_temporal.reserva_alta_version VALUES
 (ambito,1,'reservada',NULL,NULL,NULL,NULL,t,NULL),
 (ambito,2,'confirmada',1,'auditoria:ct109:alta','evento:ct109:alta',t,t,cnf);
 INSERT INTO vec_contratacion_temporal.reserva_alta_actual VALUES (ambito,2);
 INSERT INTO vec_contratacion_temporal.expediente_alta VALUES
 ('expediente:ct109:positivo','reserva:ct109:positivo','2026/CT109-POS',
 'org_diputaciondemo0001','per_corporativa_rrhh_000000000001',
 'prf_corporativo_rrhh_000000000001','decision:ct109:alta',
 'efecto:ct109:alta',h,t,cnf);
 INSERT INTO vec_contratacion_temporal.expediente_alta_version VALUES
 ('expediente:ct109:positivo',1,canon,canon_h,'flujo:ct109:positivo',1,h,
 'solicitud','en_curso',solicitud_h,t,cnf);
 INSERT INTO vec_contratacion_temporal.actuacion_alta VALUES
 ('expediente:ct109:positivo',1,1,'solicitud.registrar',
 'per_corporativa_rrhh_000000000001','unidad:ct109:positivo',
 'recibo:ct109:positivo','solicitud','en_curso',t,h,cnf);
 INSERT INTO vec_contratacion_temporal.auditoria_alta VALUES
 ('auditoria:ct109:alta',1,'expediente:ct109:positivo','decision:ct109:alta',cero,h,t,cnf,h);
 INSERT INTO vec_contratacion_temporal.outbox_alta VALUES
 ('evento:ct109:alta',1,'expediente:ct109:positivo',
 'contratacion_temporal.expediente.registrado.v1',payload,encode(sha256(payload),'hex'),cero,h,t,NULL,cnf);
 INSERT INTO vec_contratacion_temporal.confirmacion_agregado_alta VALUES
 (cnf,h,ambito,2,'reserva:ct109:positivo','expediente:ct109:positivo','2026/CT109-POS',
 'recibo:ct109:positivo','decision:ct109:alta','efecto:ct109:alta',h,h,1,canon_h,
 1,h,'auditoria:ct109:alta',1,cero,h,'evento:ct109:alta',1,
 encode(sha256(payload),'hex'),cero,h,t,h,t);
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh
 WHERE expediente_ref='expediente:ct109:positivo' AND version=1 AND corte_global=1)
 THEN RAISE EXCEPTION 'CT109: trigger de publicación no produjo la semilla'; END IF;
END $alta$;
COMMIT;

-- Génesis autorización: catálogo vacío calculado con el canon Go SHA256([]).
-- Los motivos se publican después desde una conexión del proyector nominal.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.control_catalogo_politicas
 (control_id,revision,huella_sha256,actualizado_en,actualizado_por,acto_ref)
 VALUES(true,1,encode(sha256(convert_to('[]','UTF8')),'hex'),clock_timestamp(),
 'semilla:ct109:positivo','acto:ct109:catalogo');
DO $genesis_motivos$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_checkpoint_origen
 WHERE control_id AND ultima_secuencia=0 AND ultimo_evento_ref IS NULL
 AND ultima_huella_evento_sha256 IS NULL)
 OR (SELECT count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
 WHERE clase_consulta IN ('cuadro','detalle') AND ultima_publicacion_version=0
 AND ultima_publicacion_ref IS NULL AND ultima_publicacion_huella_sha256 IS NULL)<>2
 THEN RAISE EXCEPTION 'CT109: falta génesis restaurada de motivos'; END IF;
END $genesis_motivos$;
COMMIT;
CREATE ROLE ct109_motivos_proyector LOGIN INHERIT NOSUPERUSER NOCREATEDB
 NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_motivos_proyector TO ct109_motivos_proyector
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO ct109_motivos_proyector;
