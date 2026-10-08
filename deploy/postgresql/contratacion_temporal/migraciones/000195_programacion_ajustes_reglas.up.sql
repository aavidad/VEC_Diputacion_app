\set ON_ERROR_STOP on
-- CT195: fecha de efecto opcional, cabeza completa y preimagen de replay.
-- CT190 y CT191 preceden este cambio. No ejecutar DOWN sobre historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000195',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.regla_base_activacion_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.fase_regla_instantanea_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.leer_activacion_regla_base_v1()') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
      'vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole
      AND p.prosecdef AND md5(p.prosrc)='4aa294ec7d98b1431615bdd27a6c8c89')
 THEN RAISE EXCEPTION 'CT195: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
ALTER TABLE vec_contratacion_temporal.regla_ajuste_version_v1
 ADD COLUMN publicada_en timestamptz(6),
 ADD COLUMN material_original jsonb;
ALTER TABLE vec_contratacion_temporal.regla_ajuste_version_v1
 ADD CONSTRAINT regla_ajuste_publicacion_fecha_v1 CHECK (
  publicada_en IS NULL OR
  (publicada_en=date_trunc('microseconds',publicada_en) AND publicada_en<=vigente_desde)),
 ADD CONSTRAINT regla_ajuste_material_original_v1 CHECK (
  material_original IS NULL OR (jsonb_typeof(material_original)='object'
    AND octet_length(material_original::text)<=65536));
-- Las filas anteriores son inmutables: NULL señala que el material exacto de
-- la petición no quedó guardado. Su fecha de publicación fue vigente_desde.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.operar_ajustes_reglas_v1(
 p_material jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 operacion text; d jsonb; consumo record; previa record; anterior record;
 actor text; clave uuid; esperada bigint; nueva bigint;
 ajustes jsonb; canonico text; huella text; previos jsonb; cambio jsonb; r record; c record; solicitud_h text;
 limite integer; antes_de bigint; historial jsonb; hay_mas boolean; vigente jsonb;
 cabeza jsonb; v_consultada_en timestamptz(6);
 ahora timestamptz(6); efecto timestamptz(6); efecto_solicitado text;
 recibo text; n integer; base_activa record; solicitud_h_legacy text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-148: ejecución denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object' OR octet_length(p_material::text)>65536
 THEN RAISE EXCEPTION 'CT-148: material inválido' USING ERRCODE='22023'; END IF;
 operacion:=p_material->>'operacion';
 IF operacion IS NULL OR operacion<>ALL(ARRAY['consultar','ajustar'])
    OR p_material->>'catalogo_id' IS DISTINCT FROM 'vec.contratacion_temporal.reglas.ajustes'
    OR p_material->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
    OR (operacion='consultar' AND (p_material-ARRAY['operacion','organizacion_ref','catalogo_id','limite','antes_de_version'])<>'{}'::jsonb)
    OR (operacion='ajustar' AND (p_material-ARRAY['operacion','organizacion_ref','catalogo_id','clave_idempotencia',
         'version_esperada','base_version','base_huella_sha256','ajustes_canonico','ajustes_huella_sha256',
         'cambios','motivo_clave','referencia','nota','vigente_desde'])<>'{}'::jsonb)
 THEN RAISE EXCEPTION 'CT-148: operación inválida' USING ERRCODE='22023'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-148: decisión inválida' USING ERRCODE='22023'; END;
 actor:=d->>'principal_id';
 IF actor IS NULL OR actor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR d->>'accion' IS DISTINCT FROM (CASE WHEN operacion='consultar'
         THEN 'contratacion_temporal.reglas.consultar_ajustes' ELSE 'contratacion_temporal.reglas.ajustar' END)
 THEN RAISE EXCEPTION 'CT-148: decisión divergente' USING ERRCODE='42501'; END IF;
 -- AD3-114 calcula la huella del contexto desde este mismo material.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_ct_v3_atestada(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM 'vec.contratacion_temporal.reglas'
 THEN RAISE EXCEPTION 'CT-148: consumo divergente' USING ERRCODE='42501'; END IF;

 IF operacion='consultar' THEN
  v_consultada_en:=date_trunc('microseconds',clock_timestamp());
  IF p_material ? 'limite' AND jsonb_typeof(p_material->'limite')<>'number'
  THEN RAISE EXCEPTION 'CT-148: consulta inválida' USING ERRCODE='22023'; END IF;
  BEGIN
   limite:=coalesce((p_material->>'limite')::integer,50);
   antes_de:=(p_material->>'antes_de_version')::bigint;
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-148: consulta inválida' USING ERRCODE='22023'; END;
  IF limite NOT BETWEEN 1 AND 50 OR (antes_de IS NOT NULL AND antes_de NOT BETWEEN 2 AND 10000000)
  THEN RAISE EXCEPTION 'CT-148: consulta inválida' USING ERRCODE='22023'; END IF;
  SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,'ajustes',v.ajustes,
    'vigente_desde',v.vigente_desde,'publicada_en',coalesce(v.publicada_en,v.vigente_desde),
    'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
   INTO cabeza FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
   WHERE v.catalogo_id=p_material->>'catalogo_id' ORDER BY v.version DESC LIMIT 1;
  SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,'ajustes',v.ajustes,
    'vigente_desde',v.vigente_desde,'publicada_en',coalesce(v.publicada_en,v.vigente_desde),
    'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
   INTO vigente FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
   WHERE v.catalogo_id=p_material->>'catalogo_id' AND v.vigente_desde<=v_consultada_en
   ORDER BY v.vigente_desde DESC,v.version DESC LIMIT 1;
  SELECT coalesce(jsonb_agg(h.fila ORDER BY h.version DESC),'[]'::jsonb) INTO historial FROM (
    SELECT v.version,
     jsonb_build_object('version',v.version,'vigente_desde',v.vigente_desde,
      'publicada_en',coalesce(v.publicada_en,v.vigente_desde),'actor_ref',v.actor_ref,
      'motivo_clave',v.motivo_clave,'referencia',v.referencia,'nota',v.nota,
      'base_version',v.base_version,'recibo_ref',v.recibo_ref,
      'cambios',(SELECT coalesce(jsonb_agg(jsonb_build_object('regla_clave',x.regla_clave,'campo',x.campo,
        'anterior',x.anterior,'nuevo',x.nuevo) ORDER BY x.regla_clave,x.campo),'[]'::jsonb)
        FROM vec_contratacion_temporal.regla_ajuste_cambio_v1 x
        WHERE x.catalogo_id=v.catalogo_id AND x.version=v.version)) AS fila
     FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
    WHERE v.catalogo_id=p_material->>'catalogo_id' AND (antes_de IS NULL OR v.version<antes_de)
    ORDER BY v.version DESC LIMIT limite) h;
  SELECT count(*)>limite INTO hay_mas FROM (
   SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
    WHERE v.catalogo_id=p_material->>'catalogo_id' AND (antes_de IS NULL OR v.version<antes_de)
    LIMIT limite+1) t;
  RETURN jsonb_build_object('consultada_en',v_consultada_en,
   'cabeza',cabeza,'vigente',vigente,'vigente_hoy',vigente,
   'historial',historial,'hay_mas',hay_mas);
 END IF;

 -- Ajustar.
 BEGIN
  clave:=(p_material->>'clave_idempotencia')::uuid;
  esperada:=(p_material->>'version_esperada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-148: versión o clave inválida' USING ERRCODE='22023'; END;
 IF p_material ? 'vigente_desde' THEN
  efecto_solicitado:=p_material->>'vigente_desde';
  IF jsonb_typeof(p_material->'vigente_desde')<>'string'
     OR coalesce(efecto_solicitado,'') !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
  THEN RAISE EXCEPTION 'CT-195: fecha de efecto inválida' USING ERRCODE='22023'; END IF;
  BEGIN efecto:=efecto_solicitado::timestamptz(6);
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-195: fecha de efecto inválida' USING ERRCODE='22023'; END;
 END IF;
 IF clave IS NULL OR p_material->>'clave_idempotencia'<>clave::text
    OR jsonb_typeof(p_material->'version_esperada')<>'number' OR esperada IS NULL OR esperada NOT BETWEEN 0 AND 9999998
    OR jsonb_typeof(p_material->'base_version')<>'number'
    OR coalesce(p_material->>'base_version','') !~ '^[1-9][0-9]{0,6}$'
    OR coalesce(p_material->>'base_huella_sha256','') !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_material->'ajustes_canonico')<>'string'
    OR coalesce(p_material->>'ajustes_huella_sha256','') !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_material->'cambios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_material->'cambios') NOT BETWEEN 1 AND 256
    OR coalesce(p_material->>'motivo_clave','') !~ '^[a-z][a-z0-9_]{2,63}$'
    OR (p_material ? 'referencia' AND jsonb_typeof(p_material->'referencia')<>'string')
    OR (p_material ? 'nota' AND jsonb_typeof(p_material->'nota')<>'string')
    OR (p_material ? 'referencia' AND (char_length(p_material->>'referencia') NOT BETWEEN 1 AND 120
         OR p_material->>'referencia' IS DISTINCT FROM btrim(p_material->>'referencia')
         OR p_material->>'referencia' ~ '[[:cntrl:]]'))
    OR (p_material ? 'nota' AND (char_length(p_material->>'nota') NOT BETWEEN 1 AND 500
         OR p_material->>'nota' IS DISTINCT FROM btrim(p_material->>'nota')
         OR p_material->>'nota' ~ '[[:cntrl:]]'))
 THEN RAISE EXCEPTION 'CT-148: ajuste inválido' USING ERRCODE='22023'; END IF;
 canonico:=p_material->>'ajustes_canonico';
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 IF octet_length(canonico)>16384 OR huella<>p_material->>'ajustes_huella_sha256'
 THEN RAISE EXCEPTION 'CT-148: huella de ajustes divergente' USING ERRCODE='22023'; END IF;
 BEGIN ajustes:=canonico::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-148: ajustes ilegibles' USING ERRCODE='22023'; END;
 IF NOT vec_contratacion_temporal.ajustes_reglas_forma_valida_v1(ajustes)
 THEN RAISE EXCEPTION 'CT-148: forma de ajustes inválida' USING ERRCODE='22023'; END IF;

 IF EXISTS (SELECT 1 FROM jsonb_array_elements(p_material->'cambios') e
             WHERE jsonb_typeof(e)<>'object' OR (e-ARRAY['regla_clave','campo','anterior','nuevo'])<>'{}'::jsonb
                OR jsonb_typeof(e->'regla_clave')<>'string' OR jsonb_typeof(e->'campo')<>'string'
                OR jsonb_typeof(e->'anterior')<>'string' OR jsonb_typeof(e->'nuevo')<>'string')
 THEN RAISE EXCEPTION 'CT-148: cambio mal formado' USING ERRCODE='22023'; END IF;
 -- La repetición se reconoce por lo que pidió el cliente (clave, versión
 -- esperada, valores nuevos y motivo), no por el material entero: el valor
 -- anterior y la instantánea se recalculan sobre la cabeza, que puede haber
 -- avanzado precisamente por la primera ejecución de esta misma petición.
 solicitud_h:=encode(sha256(convert_to(jsonb_build_object(
   'organizacion_ref',p_material->>'organizacion_ref','clave_idempotencia',clave::text,'version_esperada',esperada,
   'cambios',(SELECT jsonb_agg(jsonb_build_object('regla_clave',e->>'regla_clave','campo',e->>'campo','nuevo',e->>'nuevo')
               ORDER BY e->>'regla_clave',e->>'campo') FROM jsonb_array_elements(p_material->'cambios') e),
   'motivo_clave',p_material->>'motivo_clave','referencia',p_material->>'referencia',
   'nota',p_material->>'nota','vigente_desde_solicitado',efecto_solicitado)::text,'UTF8')),'hex');
 solicitud_h_legacy:=encode(sha256(convert_to(jsonb_build_object(
   'organizacion_ref',p_material->>'organizacion_ref','clave_idempotencia',clave::text,'version_esperada',esperada,
   'cambios',(SELECT jsonb_agg(jsonb_build_object('regla_clave',e->>'regla_clave','campo',e->>'campo','nuevo',e->>'nuevo')
               ORDER BY e->>'regla_clave',e->>'campo') FROM jsonb_array_elements(p_material->'cambios') e),
   'motivo_clave',p_material->>'motivo_clave','referencia',p_material->>'referencia',
   'nota',p_material->>'nota')::text,'UTF8')),'hex');

 -- Serializa escritores cooperantes. La lectura de la cabeza no bloquea la
 -- instantánea de otro escritor serializable: la PK o UNIQUE detecta el
 -- conflicto concurrente con 40001 en el cliente que debe reintentar.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:ajustes_reglas',0));
 SELECT * INTO previa FROM vec_contratacion_temporal.regla_ajuste_version_v1 v WHERE v.clave_idempotencia=clave;
 IF FOUND THEN
  IF (previa.material_original IS NULL AND efecto_solicitado IS NOT NULL)
     OR previa.solicitud_huella_sha256 IS DISTINCT FROM
     (CASE WHEN previa.material_original IS NULL THEN solicitud_h_legacy ELSE solicitud_h END)
     OR previa.actor_ref IS DISTINCT FROM actor
  THEN RAISE EXCEPTION 'CT-148: clave reutilizada con otra solicitud' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('ajustes',previa.ajustes,'replay',true,'recibo',jsonb_build_object(
   'recibo_ref',previa.recibo_ref,'clave_idempotencia',clave,'version',previa.version,
   'huella_sha256',previa.huella_sha256,'vigente_desde',previa.vigente_desde,
   'publicada_en',coalesce(previa.publicada_en,previa.vigente_desde),
   'decision_ref',previa.decision_ref,'auditoria_ref',previa.auditoria_ref,
   'consumo_huella_sha256',previa.consumo_huella_sha256));
 END IF;
 -- Una clave nueva solo puede publicar contra la base activada y aprobada.
 -- La comparación va después del replay para conservar su recibo histórico.
 SELECT a.version,a.huella_sha256 INTO base_activa
 FROM (SELECT * FROM vec_contratacion_temporal.regla_base_activacion_v1
       ORDER BY secuencia DESC LIMIT 1) a
 JOIN vec_contratacion_temporal.regla_base_publicada_v1 b
   ON b.catalogo_id=a.catalogo_id AND b.version=a.version
  AND b.huella_sha256=a.huella_sha256 AND b.aprobacion_ref=a.aprobacion_ref
 WHERE a.activa AND a.catalogo_id='vec.contratacion_temporal.reglas';
 IF NOT FOUND THEN
  RAISE EXCEPTION 'CT-191: base activa aprobada no disponible' USING ERRCODE='55000';
 END IF;
 IF base_activa.version IS DISTINCT FROM (p_material->>'base_version')::bigint
    OR base_activa.huella_sha256 IS DISTINCT FROM p_material->>'base_huella_sha256' THEN
  RAISE EXCEPTION 'CT-191: base del ajuste desfasada' USING ERRCODE='40001';
 END IF;
 SELECT * INTO anterior FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
  WHERE v.catalogo_id=p_material->>'catalogo_id' ORDER BY v.version DESC LIMIT 1;
 IF coalesce(anterior.version,0)<>esperada
 THEN RAISE EXCEPTION 'CT-148: conflicto de versión' USING ERRCODE='40001'; END IF;
 previos:=coalesce(anterior.ajustes,'{}'::jsonb);
 nueva:=esperada+1;

 -- Cada cambio: campo único, valor nuevo en la versión nueva y, si ya estaba
 -- ajustado, el anterior coincide con la versión previa.
 n:=0;
 FOR cambio IN SELECT e FROM jsonb_array_elements(p_material->'cambios') e LOOP
  IF cambio->>'anterior'=cambio->>'nuevo'
     OR ajustes #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS DISTINCT FROM cambio->>'nuevo'
     OR (previos #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS NOT NULL
         AND previos #>> ARRAY[cambio->>'regla_clave',cambio->>'campo'] IS DISTINCT FROM cambio->>'anterior')
  THEN RAISE EXCEPTION 'CT-148: cambio incoherente con los ajustes' USING ERRCODE='22023'; END IF;
  n:=n+1;
 END LOOP;
 IF (SELECT count(DISTINCT (e->>'regla_clave',e->>'campo')) FROM jsonb_array_elements(p_material->'cambios') e)<>n
 THEN RAISE EXCEPTION 'CT-148: cambio repetido' USING ERRCODE='22023'; END IF;
 -- Nada cambia sin su fila de cambio y ningún ajuste previo desaparece.
 FOR r IN SELECT x.key AS regla,y.key AS campo,y.value#>>'{}' AS valor
           FROM jsonb_each(ajustes) x,LATERAL jsonb_each(x.value) y LOOP
  IF previos #>> ARRAY[r.regla,r.campo] IS DISTINCT FROM r.valor AND NOT EXISTS (
     SELECT 1 FROM jsonb_array_elements(p_material->'cambios') e
      WHERE e->>'regla_clave'=r.regla AND e->>'campo'=r.campo)
  THEN RAISE EXCEPTION 'CT-148: ajuste sin cambio declarado' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR c IN SELECT x.key AS regla,y.key AS campo FROM jsonb_each(previos) x,LATERAL jsonb_each(x.value) y LOOP
  IF ajustes #>> ARRAY[c.regla,c.campo] IS NULL
  THEN RAISE EXCEPTION 'CT-148: un ajuste previo no puede desaparecer' USING ERRCODE='22023'; END IF;
 END LOOP;

 -- Un efecto nuevo no puede anteceder a la cabeza completa, aunque esta
 -- permanezca programada. El replay anterior ya salió antes de este CAS.
 ahora:=date_trunc('microseconds',clock_timestamp());
 efecto:=coalesce(efecto,ahora);
 IF efecto<ahora THEN
  RAISE EXCEPTION 'CT-195: fecha de efecto anterior a publicación' USING ERRCODE='22023'; END IF;
 IF anterior.version IS NOT NULL AND efecto<anterior.vigente_desde THEN
  RAISE EXCEPTION 'CT-195: efecto anterior a cabeza' USING ERRCODE='40001'; END IF;
 recibo:='recibo:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.regla_ajuste_version_v1(
  catalogo_id,version,version_esperada,organizacion_ref,ajustes,ajustes_canonico,huella_sha256,
  base_version,base_huella_sha256,actor_ref,motivo_clave,referencia,nota,vigente_desde,
  publicada_en,material_original,
  clave_idempotencia,solicitud_huella_sha256,recibo_ref,decision_ref,consumo_huella_sha256,auditoria_ref)
 VALUES(p_material->>'catalogo_id',nueva,esperada,p_material->>'organizacion_ref',ajustes,canonico,huella,
  (p_material->>'base_version')::bigint,p_material->>'base_huella_sha256',actor,p_material->>'motivo_clave',
  p_material->>'referencia',p_material->>'nota',efecto,ahora,p_material,
  clave,solicitud_h,recibo,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref);
 INSERT INTO vec_contratacion_temporal.regla_ajuste_cambio_v1(catalogo_id,version,regla_clave,campo,anterior,nuevo)
 SELECT p_material->>'catalogo_id',nueva,e->>'regla_clave',e->>'campo',e->>'anterior',e->>'nuevo'
   FROM jsonb_array_elements(p_material->'cambios') e;
 INSERT INTO vec_contratacion_temporal.regla_ajuste_outbox_v1(evento_ref,catalogo_id,version,huella_sha256,tipo,estado,creada_en)
 VALUES('evento:'||gen_random_uuid()::text,p_material->>'catalogo_id',nueva,huella,
  'contratacion_temporal.reglas.ajustadas','pendiente',ahora);
 RETURN jsonb_build_object('ajustes',ajustes,'replay',false,'recibo',jsonb_build_object(
  'recibo_ref',recibo,'clave_idempotencia',clave,'version',nueva,'huella_sha256',huella,
  'vigente_desde',efecto,'publicada_en',ahora,
  'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumo_huella_sha256',consumo.consumo_huella_sha256));
END $funcion$;

-- Material privado para un wrapper nominal de sesión y auditoría comunes.
-- La función queda ejecutable solo por el propietario del módulo.
CREATE FUNCTION vec_contratacion_temporal.leer_gobierno_ajustes_reglas_v1(
 p_limite integer,p_antes_de bigint)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='5s'
AS $leer$
DECLARE v_corte timestamptz(6); v_cabeza jsonb; v_vigente jsonb;
 v_programados jsonb; v_historial jsonb; v_hay_mas boolean; v_hay_mas_programados boolean;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 50
    OR (p_antes_de IS NOT NULL AND p_antes_de NOT BETWEEN 2 AND 10000000)
 THEN RAISE EXCEPTION 'CT-195: lectura privada denegada' USING ERRCODE='42501'; END IF;
 v_corte:=date_trunc('microseconds',clock_timestamp());
 SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,
  'ajustes',v.ajustes,'vigente_desde',v.vigente_desde,
  'publicada_en',coalesce(v.publicada_en,v.vigente_desde),
  'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
 INTO v_cabeza FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
 WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
 ORDER BY v.version DESC LIMIT 1;
 SELECT jsonb_build_object('version',v.version,'huella_sha256',v.huella_sha256,
  'ajustes',v.ajustes,'vigente_desde',v.vigente_desde,
  'publicada_en',coalesce(v.publicada_en,v.vigente_desde),
  'base_version',v.base_version,'base_huella_sha256',v.base_huella_sha256)
 INTO v_vigente FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
 WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
   AND v.vigente_desde<=v_corte
 ORDER BY v.vigente_desde DESC,v.version DESC LIMIT 1;
 SELECT coalesce(jsonb_agg(jsonb_build_object('version',x.version,
  'huella_sha256',x.huella_sha256,'ajustes',x.ajustes,
  'vigente_desde',x.vigente_desde,
  'publicada_en',coalesce(x.publicada_en,x.vigente_desde),
  'base_version',x.base_version,'base_huella_sha256',x.base_huella_sha256)
  ORDER BY x.vigente_desde,x.version),'[]'::jsonb)
 INTO v_programados FROM (
  SELECT * FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
  WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
    AND v.vigente_desde>v_corte
  ORDER BY v.vigente_desde,v.version LIMIT 50) x;
 SELECT count(*)>50 INTO v_hay_mas_programados FROM (
  SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
  WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
    AND v.vigente_desde>v_corte LIMIT 51) x;
 SELECT coalesce(jsonb_agg(h.fila ORDER BY h.version DESC),'[]'::jsonb)
 INTO v_historial FROM (
  SELECT v.version,jsonb_build_object('version',v.version,
    'vigente_desde',v.vigente_desde,
    'publicada_en',coalesce(v.publicada_en,v.vigente_desde),
    'actor_ref',v.actor_ref,'motivo_clave',v.motivo_clave,
    'referencia',v.referencia,'nota',v.nota,'base_version',v.base_version,
    'base_huella_sha256',v.base_huella_sha256,'recibo_ref',v.recibo_ref,
    'huella_sha256',v.huella_sha256,
    'cambios',(SELECT coalesce(jsonb_agg(jsonb_build_object(
      'regla_clave',x.regla_clave,'campo',x.campo,'anterior',x.anterior,
      'nuevo',x.nuevo) ORDER BY x.regla_clave,x.campo),'[]'::jsonb)
      FROM vec_contratacion_temporal.regla_ajuste_cambio_v1 x
      WHERE x.catalogo_id=v.catalogo_id AND x.version=v.version)) AS fila
  FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
  WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
    AND (p_antes_de IS NULL OR v.version<p_antes_de)
  ORDER BY v.version DESC LIMIT p_limite) h;
 SELECT count(*)>p_limite INTO v_hay_mas FROM (
  SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
  WHERE v.catalogo_id='vec.contratacion_temporal.reglas.ajustes'
    AND (p_antes_de IS NULL OR v.version<p_antes_de)
  LIMIT p_limite+1) x;
 RETURN jsonb_build_object('consultada_en',v_corte,'cabeza',v_cabeza,'vigente_hoy',v_vigente,
  'programados',v_programados,'hay_mas_programados',v_hay_mas_programados,
  'historial',v_historial,'hay_mas',v_hay_mas);
END $leer$;

CREATE FUNCTION vec_contratacion_temporal.leer_preimagen_ajuste_por_clave_v1(p_clave uuid)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='5s'
AS $leer$
DECLARE v_material jsonb; v_version bigint;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR p_clave IS NULL
 THEN RAISE EXCEPTION 'CT-195: preimagen privada denegada' USING ERRCODE='42501'; END IF;
 SELECT v.version,v.material_original INTO v_version,v_material
 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v
 WHERE v.clave_idempotencia=p_clave;
 IF NOT FOUND THEN RETURN jsonb_build_object('encontrada',false); END IF;
 RETURN jsonb_build_object('encontrada',true,'version',v_version,
  'material_original',v_material);
END $leer$;
ALTER FUNCTION vec_contratacion_temporal.leer_gobierno_ajustes_reglas_v1(integer,bigint)
 OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.leer_preimagen_ajuste_por_clave_v1(uuid)
 OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_gobierno_ajustes_reglas_v1(integer,bigint),
 vec_contratacion_temporal.leer_preimagen_ajuste_por_clave_v1(uuid) FROM PUBLIC;

DO $post$
DECLARE acto regprocedure:=
 'vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 privado1 regprocedure:='vec_contratacion_temporal.leer_gobierno_ajustes_reglas_v1(integer,bigint)'::regprocedure;
 privado2 regprocedure:='vec_contratacion_temporal.leer_preimagen_ajuste_por_clave_v1(uuid)'::regprocedure;
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (acto,privado1,privado2)
   AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole OR NOT p.prosecdef
    OR NOT EXISTS (SELECT 1 FROM unnest(p.proconfig) c
      WHERE replace(c,' ','')='search_path=pg_catalog,pg_temp')))
 OR EXISTS (SELECT 1 FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid IN (privado1,privado2) AND x.grantee<>p.proowner)
 OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',acto,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_consultor_rrhh',privado1,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',privado2,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_class c,
   LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
   WHERE c.oid='vec_contratacion_temporal.regla_ajuste_version_v1'::regclass
     AND x.grantee<>c.relowner)
 THEN RAISE EXCEPTION 'CT-195: postcondición ACL incumplida' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
