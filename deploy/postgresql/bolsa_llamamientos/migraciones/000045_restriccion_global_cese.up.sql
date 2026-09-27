\set ON_ERROR_STOP on
-- Bolsa 000045. El cese acreditado por CT produce una restricción por candidato,
-- nunca una copia del estado en cada participación. Requiere CT129 y Bolsa 000008,
-- 000018, 000024, 000037. La política inicial es solo para datos sintéticos.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000045',0));

DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'Bolsa 000045: instalación DBA requerida' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_relevo_cese') THEN
  RAISE EXCEPTION 'Bolsa 000045 ya instalada' USING ERRCODE='55000';
 END IF;
 CREATE ROLE vec_bolsa_llamamientos_relevo_cese NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_relevo_cese',current_database());
END $rol$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.integracion_desarrollo') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)','EXECUTE') THEN
  RAISE EXCEPTION 'Bolsa 000045: dependencias incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_relevo_cese;

-- El mapeo distingue modalidad CT y causa CT: "modalidad|causa" prevalece
-- sobre "modalidad". Sin entrada exacta no se aplica una regla implícita.
CREATE TABLE vec_bolsa_llamamientos.politica_cese_bolsa (
 version bigint PRIMARY KEY CHECK (version>=1),
 catalogo_ref text NOT NULL CHECK (octet_length(catalogo_ref) BETWEEN 1 AND 512),
 catalogo_sha256 text NOT NULL CHECK (catalogo_sha256 ~ '^[a-f0-9]{64}$'),
 mapeo jsonb NOT NULL CHECK (jsonb_typeof(mapeo)='object' AND octet_length(mapeo::text)<=16384),
 meses_general integer NOT NULL CHECK (meses_general BETWEEN 0 AND 120),
 meses_acumulacion integer NOT NULL CHECK (meses_acumulacion BETWEEN 0 AND 120),
 computo text NOT NULL CHECK (computo='fecha_cese_meses_calendario_ajuste_fin_mes'),
 estado text NOT NULL CHECK (estado IN ('ejemplo_sintetico','vigente')),
 publicada_en timestamptz(6) NOT NULL,
 publicada_por text NOT NULL DEFAULT session_user CHECK (octet_length(publicada_por) BETWEEN 1 AND 128)
);
CREATE TABLE vec_bolsa_llamamientos.restriccion_cese_bolsa (
 evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento:ct:contrato-bolsa:[a-f0-9]{64}$'),
 origen_ref text NOT NULL UNIQUE CHECK (octet_length(origen_ref) BETWEEN 1 AND 512),
 origen_huella_sha256 text NOT NULL CHECK (origen_huella_sha256 ~ '^[a-f0-9]{64}$'),
 origen_posicion bigint NOT NULL CHECK (origen_posicion>=0),
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 llamamiento_ref text NOT NULL,
 relacion_ref text NOT NULL,
 expediente_ref text NOT NULL,
 organizacion_ref text NOT NULL,
 modalidad_clave text NOT NULL,
 causa_contrato_clave text,
 clase_bolsa text NOT NULL CHECK (clase_bolsa IN ('general','acumulacion_tareas')),
 fecha_efecto date NOT NULL CHECK (isfinite(fecha_efecto)),
 disponible_desde date NOT NULL CHECK (isfinite(disponible_desde) AND disponible_desde>=fecha_efecto),
 politica_version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_cese_bolsa(version),
 fuente_tipo text NOT NULL,
 fuente_ref text NOT NULL,
 fuente_sha256 text NOT NULL CHECK (fuente_sha256 ~ '^[a-f0-9]{64}$'),
 recibo_ct_ref text NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:bolsa:cese:[a-f0-9]{64}$'),
 recibida_en timestamptz(6) NOT NULL,
 recibida_por text NOT NULL DEFAULT session_user CHECK (octet_length(recibida_por) BETWEEN 1 AND 128),
 CHECK (evento_ref='evento:ct:contrato-bolsa:' || encode(sha256(convert_to('cese'||chr(31)||origen_ref,'UTF8')),'hex'))
);
CREATE TABLE vec_bolsa_llamamientos.auditoria_cese_bolsa (
 auditoria_ref text PRIMARY KEY CHECK (auditoria_ref ~ '^auditoria:bolsa:cese:[a-f0-9]{64}$'),
 evento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.restriccion_cese_bolsa(evento_ref),
 accion text NOT NULL CHECK (accion='restriccion_cese_aplicada'),
 resultado text NOT NULL CHECK (resultado='aplicada'),
 actor text NOT NULL CHECK (octet_length(actor) BETWEEN 1 AND 128),
 correlacion_ref text NOT NULL,
 politica_version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_cese_bolsa(version),
 recibo_ref text NOT NULL,
 registro jsonb NOT NULL CHECK (jsonb_typeof(registro)='object'),
 registro_sha256 text NOT NULL CHECK (registro_sha256=encode(sha256(convert_to(registro::text,'UTF8')),'hex')),
 registrada_en timestamptz(6) NOT NULL,
 CHECK (auditoria_ref='auditoria:bolsa:cese:'||encode(sha256(convert_to(evento_ref,'UTF8')),'hex'))
);
CREATE INDEX restriccion_cese_bolsa_candidato ON vec_bolsa_llamamientos.restriccion_cese_bolsa(candidato_ref,disponible_desde DESC);
DO $seguridad$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['politica_cese_bolsa','restriccion_cese_bolsa','auditoria_cese_bolsa'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user=''vec_bolsa_llamamientos_propietario'')',t||'_solo_propietario',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t||'_inmutable',t);
 END LOOP;
END $seguridad$;

CREATE FUNCTION vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(
 p_catalogo_ref text,p_mapeo jsonb,p_meses_general integer,p_meses_acumulacion integer,p_estado text)
RETURNS TABLE(version bigint,reutilizada boolean,catalogo_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE v_sha text; v_actual vec_bolsa_llamamientos.politica_cese_bolsa;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'publicación denegada' USING ERRCODE='42501';
 END IF;
 IF p_catalogo_ref IS NULL OR octet_length(p_catalogo_ref) NOT BETWEEN 1 AND 512 OR p_catalogo_ref<>btrim(p_catalogo_ref)
    OR p_mapeo IS NULL OR jsonb_typeof(p_mapeo)<>'object' OR octet_length(p_mapeo::text)>16384
    OR (SELECT count(*) FROM jsonb_object_keys(p_mapeo)) NOT BETWEEN 1 AND 100
    OR EXISTS (SELECT 1 FROM jsonb_each(p_mapeo) e WHERE e.key !~ '^[a-z][a-z0-9._-]{1,79}(\|[a-z][a-z0-9._-]{1,79})?$'
                 OR jsonb_typeof(e.value)<>'string' OR e.value #>> '{}' NOT IN ('general','acumulacion_tareas'))
    OR p_meses_general NOT BETWEEN 0 AND 120 OR p_meses_acumulacion NOT BETWEEN 0 AND 120
    OR p_estado NOT IN ('ejemplo_sintetico','vigente') THEN
  RAISE EXCEPTION 'política de cese inválida' USING ERRCODE='22023';
 END IF;
 v_sha:=encode(sha256(convert_to(jsonb_build_object('mapeo',p_mapeo,'meses_general',p_meses_general,
   'meses_acumulacion',p_meses_acumulacion,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado',p_estado)::text,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:politica-cese',0));
 SELECT * INTO v_actual FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 IF v_actual.version IS NOT NULL AND v_actual.catalogo_ref=p_catalogo_ref AND v_actual.catalogo_sha256=v_sha THEN
  RETURN QUERY SELECT v_actual.version,true,v_sha; RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_cese_bolsa(version,catalogo_ref,catalogo_sha256,mapeo,meses_general,
   meses_acumulacion,computo,estado,publicada_en)
 VALUES(coalesce(v_actual.version,0)+1,p_catalogo_ref,v_sha,p_mapeo,p_meses_general,p_meses_acumulacion,
   'fecha_cese_meses_calendario_ajuste_fin_mes',p_estado,clock_timestamp());
 RETURN QUERY SELECT coalesce(v_actual.version,0)+1,false,v_sha;
END $f$;

-- Versión inicial explícita para el ejercicio sintético. Una organización
-- distinta exige otra política publicada como vigente. La regla se revisará
-- cuando RRHH conteste la duda 64; no declara norma ratificada.
INSERT INTO vec_bolsa_llamamientos.politica_cese_bolsa(version,catalogo_ref,catalogo_sha256,mapeo,meses_general,
 meses_acumulacion,computo,estado,publicada_en)
SELECT 1,'catalogo:bolsa:cese:ejemplo-sintetico:v1',
 encode(sha256(convert_to(jsonb_build_object('mapeo',m,'meses_general',5,'meses_acumulacion',9,
   'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex'),
 m,5,9,'fecha_cese_meses_calendario_ajuste_fin_mes','ejemplo_sintetico',clock_timestamp()
FROM (VALUES ('{"interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb)) x(m);

-- CT valida publicación y contenido antes de que Bolsa resuelva el vínculo
-- propio; el rol del relevo solo puede invocar esta función. Replay verifica
-- de nuevo el origen y devuelve exactamente el recibo y la fecha originales.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(
 p_origen_ref text,p_huella_sha256 text,p_posicion bigint)
RETURNS TABLE(reutilizada boolean,recibo_ref text,candidato_ref text,disponible_desde date,politica_version bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE v_ct record; v_vinculo record; v_previa vec_bolsa_llamamientos.restriccion_cese_bolsa;
 v_politica vec_bolsa_llamamientos.politica_cese_bolsa; v_clase text; v_meses integer;
 v_evento_ref text; v_recibo_ref text; v_desde date; v_auditoria jsonb; v_ahora timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_origen_ref IS NULL
    OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512 OR p_origen_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]*$'
    OR p_huella_sha256 !~ '^[a-f0-9]{64}$' OR p_posicion IS NULL OR p_posicion<0 THEN
  RAISE EXCEPTION 'cese Bolsa inválido' USING ERRCODE='22023';
 END IF;
 SELECT * INTO v_ct FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(p_origen_ref,p_huella_sha256,p_posicion);
 IF NOT FOUND OR v_ct.fecha_efecto IS NULL OR v_ct.llamamiento_ref IS NULL OR v_ct.relacion_ref IS NULL
    OR v_ct.modalidad_clave IS NULL OR v_ct.expediente_ref IS NULL OR v_ct.organizacion_ref IS NULL
    OR v_ct.fuente_tipo IS NULL OR v_ct.fuente_ref IS NULL OR v_ct.recibo_ref IS NULL
    OR v_ct.fuente_sha256 !~ '^[a-f0-9]{64}$' THEN
  RAISE EXCEPTION 'cese CT no acreditado' USING ERRCODE='42501';
 END IF;
 v_evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||p_origen_ref,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:restriccion-cese:'||v_evento_ref,0));
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r WHERE r.evento_ref=v_evento_ref;
 IF FOUND THEN
  IF v_previa.origen_huella_sha256<>p_huella_sha256 OR v_previa.origen_posicion<>p_posicion
     OR v_previa.relacion_ref<>v_ct.relacion_ref OR v_previa.fecha_efecto<>v_ct.fecha_efecto THEN
   RAISE EXCEPTION 'cese divergente' USING ERRCODE='VBC01';
  END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.candidato_ref,v_previa.disponible_desde,v_previa.politica_version; RETURN;
 END IF;
 SELECT vc.candidato_ref,l.bolsa_ref INTO v_vinculo
 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
 JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref=l.operacion_ref
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=convert_from(i.registro_canonico,'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}'
 WHERE l.llamamiento_ref=v_ct.llamamiento_ref;
 IF NOT FOUND OR v_vinculo.candidato_ref IS NULL THEN RAISE EXCEPTION 'candidato de cese no resuelto' USING ERRCODE='23503'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('bolsa:politica-cese',0));
 SELECT * INTO v_politica FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR (v_politica.estado='ejemplo_sintetico' AND v_ct.organizacion_ref !~ '^organizacion:desarrollo:') THEN
  RAISE EXCEPTION 'política de cese no aplicable' USING ERRCODE='55000';
 END IF;
 IF v_politica.catalogo_sha256 IS DISTINCT FROM encode(sha256(convert_to(jsonb_build_object('mapeo',v_politica.mapeo,
   'meses_general',v_politica.meses_general,'meses_acumulacion',v_politica.meses_acumulacion,
   'computo',v_politica.computo,'estado',v_politica.estado)::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'política de cese corrupta' USING ERRCODE='55000';
 END IF;
 v_clase:=coalesce(v_politica.mapeo->>(v_ct.modalidad_clave||'|'||v_ct.causa_contrato_clave),v_politica.mapeo->>v_ct.modalidad_clave);
 IF v_clase NOT IN ('general','acumulacion_tareas') OR v_clase IS NULL THEN
  RAISE EXCEPTION 'modalidad CT sin mapeo Bolsa' USING ERRCODE='55000';
 END IF;
 v_meses:=CASE v_clase WHEN 'acumulacion_tareas' THEN v_politica.meses_acumulacion ELSE v_politica.meses_general END;
 -- PostgreSQL suma meses calendario y ajusta el día si el mes final es más
 -- corto: 31/01 + 1 mes = 28/02 (o 29/02 en bisiesto).
 v_desde:=(v_ct.fecha_efecto + make_interval(months=>v_meses))::date;
 v_recibo_ref:='recibo:bolsa:cese:'||encode(sha256(convert_to(v_evento_ref,'UTF8')),'hex');
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa(evento_ref,origen_ref,origen_huella_sha256,origen_posicion,
  candidato_ref,llamamiento_ref,relacion_ref,expediente_ref,organizacion_ref,modalidad_clave,causa_contrato_clave,clase_bolsa,
  fecha_efecto,disponible_desde,politica_version,fuente_tipo,fuente_ref,fuente_sha256,recibo_ct_ref,recibo_ref,recibida_en)
 VALUES(v_evento_ref,p_origen_ref,p_huella_sha256,p_posicion,v_vinculo.candidato_ref,v_ct.llamamiento_ref,
  v_ct.relacion_ref,v_ct.expediente_ref,v_ct.organizacion_ref,v_ct.modalidad_clave,v_ct.causa_contrato_clave,v_clase,
  v_ct.fecha_efecto,v_desde,v_politica.version,v_ct.fuente_tipo,v_ct.fuente_ref,v_ct.fuente_sha256,v_ct.recibo_ref,
  v_recibo_ref,v_ahora);
 v_auditoria:=jsonb_build_object('esquema','vec.bolsa.cese.auditoria.v1','evento_ref',v_evento_ref,
   'accion','restriccion_cese_aplicada','resultado','aplicada','actor',session_user,'correlacion_ref',p_origen_ref,
   'politica_version',v_politica.version,'recibo_ref',v_recibo_ref,'registrada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_bolsa_llamamientos.auditoria_cese_bolsa(auditoria_ref,evento_ref,accion,resultado,actor,correlacion_ref,
  politica_version,recibo_ref,registro,registro_sha256,registrada_en)
 VALUES('auditoria:bolsa:cese:'||encode(sha256(convert_to(v_evento_ref,'UTF8')),'hex'),v_evento_ref,
  'restriccion_cese_aplicada','aplicada',session_user,p_origen_ref,v_politica.version,v_recibo_ref,v_auditoria,
  encode(sha256(convert_to(v_auditoria::text,'UTF8')),'hex'),v_ahora);
 RETURN QUERY SELECT false,v_recibo_ref,v_vinculo.candidato_ref,v_desde,v_politica.version;
END $f$;

-- Lectura mínima para las proyecciones autorizadas de RRHH, Mi Bolsa y
-- publicación: ninguna participación adquiere una copia de la restricción.
-- Sin fila significa que el cese no restringe en el corte solicitado.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(
 p_participacion_ref text,p_corte timestamptz)
RETURNS TABLE(disponible_desde date,fecha_efecto date,recibo_ref text,politica_version bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT r.disponible_desde,r.fecha_efecto,r.recibo_ref,r.politica_version
 FROM vec_bolsa_llamamientos.vinculo_candidato vc
 JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa r ON r.candidato_ref=vc.candidato_ref
 WHERE vc.participacion_ref=p_participacion_ref
   AND r.fecha_efecto<=(p_corte AT TIME ZONE 'Europe/Madrid')::date
   AND r.disponible_desde>(p_corte AT TIME ZONE 'Europe/Madrid')::date
 ORDER BY r.disponible_desde DESC,r.fecha_efecto DESC,r.evento_ref DESC
 LIMIT 1
$f$;

-- La lectura B6 usa la restricción por candidato en todas las bolsas. Este
-- cambio tiene guardas exactas para evitar sustituir otra versión del lector.
DO $orden$
DECLARE v_oid oid:='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure;
 v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
 WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 FOR v_cambio IN SELECT * FROM (VALUES
  ($a$SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,$a$,
   $b$SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         coalesce(rc.disponible_desde>(p_en AT TIME ZONE 'Europe/Madrid')::date,false) AS cese_restringido,$b$),
  ($a$(s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)) AS ocupa_turno,$a$,
   $b$((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en))
           AND NOT coalesce(rc.disponible_desde>(p_en AT TIME ZONE 'Europe/Madrid')::date,false)) AS ocupa_turno,$b$),
  ($a$s ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en$a$,
   $b$s ON true
    LEFT JOIN LATERAL (SELECT max(r.disponible_desde) AS disponible_desde
      FROM vec_bolsa_llamamientos.vinculo_candidato vc
      JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa r ON r.candidato_ref=vc.candidato_ref
     WHERE vc.participacion_ref=e.participacion_ref AND r.fecha_efecto<=(p_en AT TIME ZONE 'Europe/Madrid')::date) rc ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en$b$),
  ($a$b.participacion_ref,b.orden_acta,e.orden_vigente,b.situacion,$a$,
   $b$b.participacion_ref,b.orden_acta,e.orden_vigente,
        CASE WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'disponible_desde' ELSE b.situacion END AS situacion,$b$),
  ($a$CASE WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'$a$,
   $b$CASE WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde') THEN 'restriccion_cese'
             WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'$b$)
 ) AS x(antes,despues) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.antes,''))<>length(v_cambio.antes) THEN
   RAISE EXCEPTION 'Bolsa 000045: lector de orden incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.antes,v_cambio.despues);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000045: lector de orden alterado' USING ERRCODE='55000';
 END IF;
END $orden$;

-- Mi Bolsa v1 y su envoltura de portal (000029) mantienen la autorización
-- vigente y proyectan la restricción desde la misma transacción. El formato
-- existente es instante UTC; se representa la medianoche de Granada como
-- instante UTC sin perder la fecha local de disponibilidad.
DO $mi_bolsa$
DECLARE v_oid oid:='vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
  WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 FOR v_cambio IN SELECT * FROM (VALUES
  ($a$'estado',situacion.situacion,$a$,
   $b$'estado',CASE WHEN cese.disponible_desde IS NOT NULL
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN 'no_disponible' ELSE situacion.situacion END,$b$),
  ($a$'desde',to_char(situacion.desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$a$,
   $b$'desde',to_char((CASE WHEN cese.fecha_efecto IS NOT NULL
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN cese.fecha_efecto::timestamp AT TIME ZONE 'Europe/Madrid' ELSE situacion.desde END)
       AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$b$),
  ($a$'fecha_disponible',CASE WHEN situacion.fecha_disponible IS NULL THEN NULL ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END$a$,
   $b$'fecha_disponible',CASE WHEN cese.disponible_desde IS NOT NULL
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN to_char((cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid')
         AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       WHEN situacion.fecha_disponible IS NULL THEN NULL
       ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END$b$),
  ($a$) situacion ON true
 LEFT JOIN LATERAL ($a$,
   $b$) situacion ON true
 LEFT JOIN LATERAL vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(
   participacion.participacion_ref,p_consultada_en) cese ON true
 LEFT JOIN LATERAL ($b$)
 ) AS x(antes,despues) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.antes,''))<>length(v_cambio.antes) THEN
   RAISE EXCEPTION 'Bolsa 000045: Mi Bolsa incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.antes,v_cambio.despues);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000045: Mi Bolsa alterada' USING ERRCODE='55000';
 END IF;
END $mi_bolsa$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(text,jsonb,integer,integer,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(text,jsonb,integer,integer,text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint) TO vec_bolsa_llamamientos_relevo_cese;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;
COMMENT ON TABLE vec_bolsa_llamamientos.restriccion_cese_bolsa IS 'Restricción global por candidato derivada de cese CT verificado; el recibo, regla y origen son inmutables.';
COMMIT;
