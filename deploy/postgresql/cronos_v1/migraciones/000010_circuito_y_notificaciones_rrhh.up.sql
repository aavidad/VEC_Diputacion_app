\set ON_ERROR_STOP on
-- Cronos 000010. Tres cambios que se instalan juntos después de 000009:
--
-- 1. Circuito de resolución (Alberto, 25/09/2026). Por regla general los
--    permisos los resuelve primero la jefatura y por último RRHH (J-A), sea
--    cual sea el circuito escrito en el catálogo. Sólo va directo a RRHH (A)
--    la persona con una marca expresa y vigente de su unidad publicada por
--    RRHH en permiso_circuito_directo (provisional, duda 47); nunca se deduce
--    de la falta de jefatura. Sin jefatura asignada y sin esa marca la
--    solicitud queda pendiente de asignación: RRHH la ve en su bandeja, pero
--    nadie puede resolverla (falla cerrado). Sustituye las dos funciones de
--    quien resuelve de 000009 y, de paso, trata como no competente (PC012)
--    una asignación que empieza entre el consumo y la comprobación.
-- 2. consumir_propio_v1 (000008) rechaza también un contexto de quien
--    resuelve ya fijado en la transacción, como ya hacía con el propio.
-- 3. Notificaciones de la persona empleada a RRHH (C9): tipo de un catálogo
--    versionado, fecha a la que se refiere, texto de hasta 512 caracteres y,
--    opcionalmente, la referencia y la huella SHA-256 de un documento que no
--    se sube. Con recibo, idempotencia e historia de solo adición. RRHH (quien
--    resuelve el paso de administración de esa persona en el circuito
--    publicado) las consulta en su bandeja y las marca atendidas con recibo;
--    la persona ve el estado de las suyas.
--
-- Autorización: cada función de notificaciones consume en su transacción una
-- decisión V3 nueva de su audiencia (AD3-58). Orden: AD3-53, AD3-70, AD3-57,
-- cronos_v1 000009, AD3-58 y esta migración. No crea cuentas LOGIN ni datos.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000010',0));
DO $pre$
DECLARE fachada text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_auditor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR to_regprocedure('vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_cronos_v1.consultar_bandeja_permisos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_resolucion') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_circuito_directo') IS NOT NULL
    OR to_regclass('vec_cronos_v1.notificacion') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_cronos_v1.permiso_resolucion'::regclass AND attname='circuito' AND NOT attisdropped) THEN
   RAISE EXCEPTION 'Cronos 000010: preimagen incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH fachada IN ARRAY ARRAY['registrar_y_consumir_cronos_notificacion_v3_atestada','consumir_cronos_notificaciones_propio_v3_atestada',
   'consumir_cronos_notificaciones_bandeja_v3_atestada','registrar_y_consumir_cronos_atencion_notificacion_v3_atestada'] LOOP
   IF to_regprocedure('vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
      OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
     RAISE EXCEPTION 'Cronos 000010: falta consumidor AD3-58 %',fachada USING ERRCODE='55000';
   END IF;
 END LOOP;
END $pre$;

-- ===================== 1. Circuito de resolución =====================

-- Marca expresa de circuito directo a RRHH para las personas de una unidad
-- sin jefatura. Solo adición: se retira con una fila en la tabla de retirada.
CREATE TABLE vec_cronos_v1.permiso_circuito_directo (
  marca_ref text PRIMARY KEY CHECK (marca_ref ~ '^circuito:cronos:[-A-Za-z0-9_.:]{1,128}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  unidad_ref text NOT NULL CHECK (unidad_ref ~ '^[-A-Za-z0-9_.:]{1,128}$'),
  unidad_etiqueta text NOT NULL CHECK (length(unidad_etiqueta) BETWEEN 1 AND 120 AND unidad_etiqueta !~ '[[:cntrl:]]'),
  vigente_desde timestamptz(6) NOT NULL,
  vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
  sintetico boolean NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
CREATE INDEX permiso_circuito_directo_empleado_idx ON vec_cronos_v1.permiso_circuito_directo(empleado_ref);
CREATE TABLE vec_cronos_v1.permiso_circuito_directo_retirada (
  marca_ref text PRIMARY KEY REFERENCES vec_cronos_v1.permiso_circuito_directo,
  retirada_en timestamptz(6) NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
-- Circuito aplicado en cada resolución. Las anteriores a 000010 quedan sin él.
ALTER TABLE vec_cronos_v1.permiso_resolucion
  ADD COLUMN circuito text CHECK (circuito IS NULL OR circuito IN ('A','J-A')),
  ADD COLUMN circuito_directo_ref text REFERENCES vec_cronos_v1.permiso_circuito_directo,
  ADD CONSTRAINT permiso_resolucion_circuito_directo_check CHECK ((circuito IS NOT DISTINCT FROM 'A')=(circuito_directo_ref IS NOT NULL)),
  ADD CONSTRAINT permiso_resolucion_circuito_paso_check CHECK (circuito IS DISTINCT FROM 'A' OR paso='administracion');

CREATE FUNCTION vec_cronos_v1.circuito_directo_vigente_v1(p_empleado text,p_ahora timestamptz)
RETURNS text LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT d.marca_ref FROM vec_cronos_v1.permiso_circuito_directo d
  WHERE d.empleado_ref=p_empleado AND d.vigente_desde<=p_ahora AND (d.vigente_hasta IS NULL OR p_ahora<d.vigente_hasta)
    AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_circuito_directo_retirada x WHERE x.marca_ref=d.marca_ref AND x.retirada_en<=p_ahora)
  ORDER BY d.publicada_en DESC,d.marca_ref LIMIT 1
$f$;
CREATE FUNCTION vec_cronos_v1.jefatura_asignada_v1(p_empleado text,p_ahora timestamptz)
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor r
   WHERE r.empleado_ref=p_empleado AND r.paso='responsable'
     AND r.vigente_desde<=p_ahora AND (r.vigente_hasta IS NULL OR p_ahora<r.vigente_hasta)
     AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=p_ahora))
$f$;

-- La existencia de jefatura se comprueba sólo dentro de las funciones
-- definidoras de quien resuelve (sin DML directo) y no se devuelve nunca.
CREATE POLICY lectura_jefatura_asignada ON vec_cronos_v1.permiso_resolutor FOR SELECT TO vec_cronos_v1_propietario
 USING (paso='responsable' AND nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL);

CREATE OR REPLACE FUNCTION vec_cronos_v1.consultar_bandeja_permisos_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; paso text; propio text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','paso','zona_horaria']);
 paso:=m->>'paso'; propio:=m->>'empleado_ref';
 IF paso NOT IN ('responsable','administracion') THEN
   RAISE EXCEPTION 'paso Cronos inválido' USING ERRCODE='PC001';
 END IF;
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('consumir_cronos_bandeja_permisos_v3_atestada','resolutor',
   'cronos.permisos.bandeja.consultar','bandeja_permisos','consultar_bandeja_permisos','bandeja:cronos:permisos:'||paso,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.bandeja_acceso(decision_ref,resolutor_ref,paso,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',paso,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('paso',paso,'pendientes',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'solicitud_ref',s.solicitud_ref,'empleado_ref',s.empleado_ref,'empleado_etiqueta',
      (SELECT r.empleado_etiqueta FROM vec_cronos_v1.permiso_resolutor r WHERE r.empleado_ref=s.empleado_ref AND r.paso=paso
         AND r.resolutor_ref=m->>'actor_ref' ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1),
      'permiso_ref',s.permiso_ref,'nombre',pc.nombre,'circuito',CASE WHEN e.estado='solicitado' AND s.directo IS NOT NULL THEN 'A' ELSE 'J-A' END,
      'pendiente_asignacion',(e.estado='solicitado' AND s.directo IS NULL AND NOT s.jefatura),'justificante_exigido',pc.justificante_exigido,
      'desde',to_char(s.desde,'YYYY-MM-DD'),'hasta',to_char(s.hasta,'YYYY-MM-DD'),'hora_inicio',s.hora_inicio,'hora_fin',s.hora_fin,
      'cantidad',s.cantidad,'unidad',s.unidad,'estado',e.estado,'version',e.version,'solicitada_en',s.solicitada_en)
      ORDER BY s.solicitada_en,s.solicitud_ref)
    FROM (SELECT s0.*,vec_cronos_v1.circuito_directo_vigente_v1(s0.empleado_ref,k.ahora) directo,
                 vec_cronos_v1.jefatura_asignada_v1(s0.empleado_ref,k.ahora) jefatura
            FROM vec_cronos_v1.permiso_solicitud s0
           WHERE vec_cronos_v1.resolutor_competente_v1(s0.empleado_ref,paso) AND s0.empleado_ref<>propio
           ORDER BY s0.solicitada_en,s0.solicitud_ref) s
    JOIN vec_cronos_v1.estado_permiso_visible_v1() e ON e.solicitud_ref=s.solicitud_ref
    JOIN vec_cronos_v1.permiso_catalogo pc ON pc.version_ref=s.catalogo_version_ref
   -- Jefatura: todo lo solicitado salvo marca directa. RRHH: lo que ya tiene
   -- la conformidad de la jefatura, lo de marca directa y, sin poder
   -- resolverlo, lo que espera la asignación de una jefatura.
   WHERE (paso='responsable' AND e.estado='solicitado' AND s.directo IS NULL)
      OR (paso='administracion' AND (e.estado='pendiente_administracion'
          OR (e.estado='solicitado' AND (s.directo IS NOT NULL OR NOT s.jefatura))))),'[]'::jsonb));
 IF jsonb_array_length(resultado->'pendientes')>500 THEN
   RAISE EXCEPTION 'bandeja Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE OR REPLACE FUNCTION vec_cronos_v1.resolver_permiso_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; ref text; paso text; decision text; motivo text; esperada integer;
 previa vec_cronos_v1.permiso_resolucion%ROWTYPE; sol vec_cronos_v1.permiso_solicitud%ROWTYPE; cat vec_cronos_v1.permiso_catalogo%ROWTYPE;
 asignacion text; actual record; nuevo text; recibo text; aviso text; directo text; circuito text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','solicitud_ref',
   'paso','decision','motivo','version_esperada','zona_horaria']);
 paso:=m->>'paso'; decision:=m->>'decision'; motivo:=nullif(m->>'motivo','');
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'solicitud_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR paso NOT IN ('responsable','administracion') OR decision NOT IN ('aprobar','denegar')
    OR m->>'version_esperada' !~ '^[1-9][0-9]{0,2}$'
    OR (motivo IS NOT NULL AND (char_length(motivo)>500 OR motivo ~ '[[:cntrl:]]' OR motivo<>btrim(motivo)))
    OR (decision='denegar' AND motivo IS NULL) THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 esperada:=(m->>'version_esperada')::integer; ref:='permiso:cronos:resolucion:'||(m->>'clave_operacion');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('registrar_y_consumir_cronos_resolucion_permiso_v3_atestada','resolutor',
   'cronos.permiso.resolver','resolucion_permiso','resolver_permiso',ref,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:solicitud:'||(m->>'solicitud_ref')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.permiso_resolucion WHERE resolucion_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.resolucion_replay(decision_ref,resolutor_ref,resolucion_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,m->>'actor_ref',ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('resolucion_ref',ref,'solicitud_ref',previa.solicitud_ref,'recibo_ref',previa.recibo_ref,
     'estado',previa.estado_resultante,'version',previa.version_resultante,'instante_utc',previa.registrada_en,'replay',true);
 END IF;
 -- Sin competencia vigente la solicitud no es visible: inexistente y ajena
 -- responden igual, sin revelar si existe.
 SELECT * INTO sol FROM vec_cronos_v1.permiso_solicitud s WHERE s.solicitud_ref=m->>'solicitud_ref' AND vec_cronos_v1.resolutor_competente_v1(s.empleado_ref,paso);
 IF NOT FOUND OR sol.empleado_ref=m->>'empleado_ref' THEN
   RAISE EXCEPTION 'resolución no competente' USING ERRCODE='PC012';
 END IF;
 -- La asignación se evalúa en el mismo instante del consumo: una que empieza
 -- entre ambos no acredita competencia (no competente, nunca error interno).
 SELECT r.asignacion_ref INTO asignacion FROM vec_cronos_v1.permiso_resolutor r
  WHERE r.empleado_ref=sol.empleado_ref AND r.paso=paso AND r.resolutor_ref=m->>'actor_ref'
    AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
    AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
  ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1;
 IF asignacion IS NULL THEN
   RAISE EXCEPTION 'resolución no competente' USING ERRCODE='PC012';
 END IF;
 SELECT * INTO STRICT cat FROM vec_cronos_v1.permiso_catalogo WHERE version_ref=sol.catalogo_version_ref;
 SELECT e.version,e.estado INTO STRICT actual FROM vec_cronos_v1.permiso_estado e WHERE e.solicitud_ref=sol.solicitud_ref ORDER BY e.version DESC LIMIT 1;
 IF actual.version<>esperada THEN
   RAISE EXCEPTION 'versión Cronos en conflicto' USING ERRCODE='PC011';
 END IF;
 -- El circuito no sale del catálogo: J-A salvo marca directa vigente.
 directo:=vec_cronos_v1.circuito_directo_vigente_v1(sol.empleado_ref,k.ahora);
 IF actual.estado='solicitado' AND directo IS NULL AND paso='responsable' THEN
   nuevo:=CASE decision WHEN 'aprobar' THEN 'pendiente_administracion' ELSE 'denegado' END;
   circuito:='J-A';
 ELSIF (actual.estado='pendiente_administracion' OR (actual.estado='solicitado' AND directo IS NOT NULL)) AND paso='administracion' THEN
   nuevo:=CASE decision WHEN 'aprobar' THEN 'concedido' ELSE 'denegado' END;
   IF actual.estado='solicitado' THEN circuito:='A'; ELSE circuito:='J-A'; directo:=NULL; END IF;
   -- Separación de funciones: quien resolvió como responsable no concede
   -- ni deniega después como RRHH la misma solicitud.
   IF EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolucion r WHERE r.solicitud_ref=sol.solicitud_ref AND r.actor_ref=m->>'actor_ref') THEN
     RAISE EXCEPTION 'resolución no competente' USING ERRCODE='PC012';
   END IF;
 ELSIF actual.estado='solicitado' AND paso='administracion' AND NOT vec_cronos_v1.jefatura_asignada_v1(sol.empleado_ref,k.ahora) THEN
   -- Sin jefatura ni marca directa: pendiente de asignación, no se resuelve.
   RAISE EXCEPTION 'solicitud Cronos pendiente de asignación' USING ERRCODE='PC014';
 ELSE
   RAISE EXCEPTION 'estado Cronos en conflicto' USING ERRCODE='PC011';
 END IF;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.permiso_resolucion(resolucion_ref,clave_operacion,solicitud_ref,empleado_ref,paso,decision,motivo,version_previa,
   version_resultante,estado_resultante,circuito,circuito_directo_ref,asignacion_ref,actor_ref,perfil_ref,material_sha256,decision_ref,auditoria_ref,
   consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',sol.solicitud_ref,sol.empleado_ref,paso,decision,motivo,actual.version,actual.version+1,nuevo,circuito,directo,asignacion,
   m->>'actor_ref',m->>'perfil_ref',k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.permiso_estado(estado_ref,solicitud_ref,empleado_ref,version,estado,pendiente_justificar,recibo_ref,registrada_en)
 VALUES('permiso:estado:'||gen_random_uuid()::text,sol.solicitud_ref,sol.empleado_ref,actual.version+1,nuevo,
   nuevo='concedido' AND cat.justificante_exigido,'recibo:cronos:'||gen_random_uuid()::text,k.ahora);
 IF nuevo IN ('concedido','denegado') THEN
   aviso:='aviso:cronos:'||gen_random_uuid()::text;
   INSERT INTO vec_cronos_v1.permiso_aviso(aviso_ref,resolucion_ref,solicitud_ref,empleado_ref,creado_en)
   VALUES(aviso,ref,sol.solicitud_ref,sol.empleado_ref,k.ahora);
 END IF;
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,sol.solicitud_ref,sol.empleado_ref,'cronos.permiso.resuelto',
   jsonb_build_object('solicitud_ref',sol.solicitud_ref,'resolucion_ref',ref,'recibo_ref',recibo,'paso',paso,'circuito',circuito,'estado',nuevo,
     'version',actual.version+1,'aviso_ref',aviso),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('resolucion_ref',ref,'solicitud_ref',sol.solicitud_ref,'recibo_ref',recibo,'estado',nuevo,
   'version',actual.version+1,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

-- ===================== 2. Consumo propio endurecido =====================

CREATE OR REPLACE FUNCTION vec_cronos_v1.consumir_propio_v1(
    p_fachada text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_bloqueos text[],
    m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF p_fachada NOT IN ('consumir_cronos_movimientos_propio_v3_atestada','registrar_y_consumir_cronos_correccion_v3_atestada',
     'consumir_cronos_permisos_propio_v3_atestada','registrar_y_consumir_cronos_permiso_v3_atestada') THEN
   RAISE EXCEPTION 'fachada Cronos desconocida' USING ERRCODE='PC003';
 END IF;
 -- Ni el contexto propio ni el de quien resuelve pueden venir ya fijados.
 IF nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL OR nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'contexto Cronos ya fijado' USING ERRCODE='PC003';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'autorización Cronos ilegible' USING ERRCODE='PC003';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',material_sha256);
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,p_accion,p_tipo,p_finalidad,p_recurso,m->>'actor_ref',m->>'perfil_ref',huella);
 FOREACH b IN ARRAY coalesce(p_bloqueos,ARRAY[]::text[]) LOOP
   PERFORM pg_advisory_xact_lock(hashtextextended(b,0));
 END LOOP;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',p_fachada) INTO STRICT consumo
   USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL
    OR nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 decision_ref:=consumo.decision_ref; auditoria_ref:=consumo.auditoria_ref; consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

-- ===================== 3. Notificaciones a RRHH =====================

-- Catálogo versionado de tipos. Lo publica el propietario (instalación
-- gobernada); los valores iniciales son sintéticos y provisionales.
CREATE TABLE vec_cronos_v1.notificacion_tipo (
  version_ref text PRIMARY KEY CHECK (version_ref ~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$'),
  tipo_ref text NOT NULL CHECK (tipo_ref ~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}$' AND starts_with(version_ref,tipo_ref||':')),
  nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 120 AND nombre !~ '[[:cntrl:]]'),
  orden integer NOT NULL CHECK (orden BETWEEN 1 AND 10000),
  vigente_desde timestamptz(6) NOT NULL,
  vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
  sintetico boolean NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
-- Declaración de la persona. El adjunto es la referencia de custodia y la
-- huella del documento, nunca sus bytes.
CREATE TABLE vec_cronos_v1.notificacion (
  notificacion_ref text PRIMARY KEY CHECK (notificacion_ref ~ '^notificacion:cronos:[0-9a-f-]{36}$'),
  clave_operacion text NOT NULL CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
  tipo_version_ref text NOT NULL REFERENCES vec_cronos_v1.notificacion_tipo,
  fecha_referida date NOT NULL,
  texto text NOT NULL CHECK (char_length(texto) BETWEEN 1 AND 512 AND translate(texto,E'\n\t','') !~ '[[:cntrl:]]' AND btrim(texto,E' \n\t')<>''),
  adjunto_ref text CHECK (adjunto_ref IS NULL OR adjunto_ref ~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'),
  adjunto_sha256 text CHECK (adjunto_sha256 IS NULL OR (adjunto_sha256 ~ '^[0-9a-f]{64}$' AND adjunto_sha256<>repeat('0',64))),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  registrada_en timestamptz(6) NOT NULL,
  UNIQUE (empleado_ref,clave_operacion),
  CHECK ((adjunto_ref IS NULL)=(adjunto_sha256 IS NULL))
);
CREATE INDEX notificacion_empleado_idx ON vec_cronos_v1.notificacion(empleado_ref,registrada_en);
-- Atención por RRHH: una sola por notificación, con recibo.
CREATE TABLE vec_cronos_v1.notificacion_atencion (
  atencion_ref text PRIMARY KEY CHECK (atencion_ref ~ '^notificacion:cronos:atencion:[0-9a-f-]{36}$'),
  clave_operacion text NOT NULL CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  notificacion_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.notificacion,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  asignacion_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_resolutor,
  actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  atendida_en timestamptz(6) NOT NULL,
  UNIQUE (actor_ref,clave_operacion)
);
CREATE INDEX notificacion_atencion_empleado_idx ON vec_cronos_v1.notificacion_atencion(empleado_ref);
-- Evidencia de cada acceso autorizado y de cada replay de RRHH.
CREATE TABLE vec_cronos_v1.notificaciones_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.notificaciones_bandeja_acceso (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.atencion_replay (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  atencion_ref text NOT NULL REFERENCES vec_cronos_v1.notificacion_atencion,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  registrada_en timestamptz(6) NOT NULL
);

DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['permiso_circuito_directo','permiso_circuito_directo_retirada','notificacion_tipo','notificacion',
   'notificacion_atencion','notificaciones_acceso','notificaciones_bandeja_acceso','atencion_replay'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
 -- Publicaciones gobernadas (marca de circuito y tipos): las inserta el
 -- propietario sin contexto de sesión; se leen enteras.
 FOREACH tabla IN ARRAY ARRAY['permiso_circuito_directo','permiso_circuito_directo_retirada','notificacion_tipo'] LOOP
  EXECUTE format('CREATE POLICY lectura_publicada ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (true)',tabla);
  EXECUTE format($p$CREATE POLICY publicacion ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario
    WITH CHECK (nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL)$p$,tabla);
 END LOOP;
END $seguridad$;
-- La persona escribe y lee lo suyo; RRHH competente lee y atiende.
CREATE POLICY lectura_propia ON vec_cronos_v1.notificacion FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'') OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_propia ON vec_cronos_v1.notificacion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY lectura ON vec_cronos_v1.notificacion_atencion FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'') OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.notificacion_atencion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion') AND actor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.notificaciones_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.notificaciones_bandeja_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.atencion_replay FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_atencion ON vec_cronos_v1.solicitud_outbox FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion') AND tipo='cronos.notificacion.atendida');

ALTER TABLE vec_cronos_v1.solicitud_outbox DROP CONSTRAINT solicitud_outbox_tipo_check;
ALTER TABLE vec_cronos_v1.solicitud_outbox ADD CONSTRAINT solicitud_outbox_tipo_check
 CHECK (tipo IN ('cronos.correccion.solicitada','cronos.permiso.solicitado','cronos.permiso.resuelto','cronos.aviso.archivado',
   'cronos.notificacion.registrada','cronos.notificacion.atendida'));
-- La frontera audita también las cuatro rutas nuevas. Sólo amplía la lista.
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
  '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
  '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
  '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
  '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
  '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
  '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos',
  '/api/interna/cronos/notificaciones/propio','/api/interna/cronos/notificaciones/envios',
  '/api/interna/cronos/notificaciones/bandeja','/api/interna/cronos/notificaciones/atenciones','otra'));

-- Consumo nominal de las notificaciones. Modo 'propio': la persona sobre sí
-- misma. Modo 'rrhh': quien resuelve el paso de administración, con el mismo
-- recurso {persona_ref, paso_resolucion='administracion'} que su bandeja.
CREATE FUNCTION vec_cronos_v1.consumir_notificacion_v1(
    p_fachada text,p_modo text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_bloqueos text[],
    m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF NOT ((p_modo='propio' AND p_fachada IN ('registrar_y_consumir_cronos_notificacion_v3_atestada','consumir_cronos_notificaciones_propio_v3_atestada'))
      OR (p_modo='rrhh' AND p_fachada IN ('consumir_cronos_notificaciones_bandeja_v3_atestada','registrar_y_consumir_cronos_atencion_notificacion_v3_atestada'))) THEN
   RAISE EXCEPTION 'fachada Cronos desconocida' USING ERRCODE='PC003';
 END IF;
 IF nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL OR nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'contexto Cronos ya fijado' USING ERRCODE='PC003';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'autorización Cronos ilegible' USING ERRCODE='PC003';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=CASE p_modo WHEN 'propio' THEN vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',material_sha256)
   ELSE vec_cronos_v1.huella_contexto_resolutor_v1(m->>'actor_ref','administracion',material_sha256) END;
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,p_accion,p_tipo,p_finalidad,p_recurso,m->>'actor_ref',m->>'perfil_ref',huella);
 FOREACH b IN ARRAY coalesce(p_bloqueos,ARRAY[]::text[]) LOOP
   PERFORM pg_advisory_xact_lock(hashtextextended(b,0));
 END LOOP;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',p_fachada) INTO STRICT consumo
   USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos no válida' USING ERRCODE='PC003';
 END IF;
 IF p_modo='propio' THEN
   PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 ELSE
   PERFORM set_config('vec.cronos.resolutor_ref',m->>'actor_ref',true);
 END IF;
 decision_ref:=consumo.decision_ref; auditoria_ref:=consumo.auditoria_ref; consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

-- Tipos vigentes en un instante, sin repetir tipo (la versión más reciente).
CREATE FUNCTION vec_cronos_v1.notificacion_tipos_vigentes_v1(p_instante timestamptz)
RETURNS SETOF vec_cronos_v1.notificacion_tipo LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT DISTINCT ON (t.tipo_ref) t.* FROM vec_cronos_v1.notificacion_tipo t
  WHERE t.vigente_desde<=p_instante AND (t.vigente_hasta IS NULL OR p_instante<t.vigente_hasta)
  ORDER BY t.tipo_ref,t.vigente_desde DESC,t.publicada_en DESC
$f$;

CREATE FUNCTION vec_cronos_v1.registrar_notificacion_propia_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; fecha date; texto text; adjunto text; huella text; tipo vec_cronos_v1.notificacion_tipo%ROWTYPE;
 previa vec_cronos_v1.notificacion%ROWTYPE; ref text; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','tipo_version_ref',
   'fecha_referida','texto','adjunto_ref','adjunto_sha256','zona_horaria']);
 texto:=m->>'texto'; adjunto:=nullif(m->>'adjunto_ref',''); huella:=nullif(m->>'adjunto_sha256','');
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'tipo_version_ref' !~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$'
    OR char_length(texto) NOT BETWEEN 1 AND 512 OR translate(texto,E'\n\t','') ~ '[[:cntrl:]]' OR btrim(texto,E' \n\t')=''
    OR (adjunto IS NULL)<>(huella IS NULL)
    OR (adjunto IS NOT NULL AND (adjunto !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$' OR huella !~ '^[0-9a-f]{64}$' OR huella=repeat('0',64))) THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 fecha:=vec_cronos_v1.fecha_material_v1(m->>'fecha_referida');
 emp:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('registrar_y_consumir_cronos_notificacion_v3_atestada','propio',
   'cronos.notificacion.propia.registrar','notificacion_propia','comunicar_incidencia_rrhh','notificacion:cronos:'||(m->>'clave_operacion'),
   ARRAY['vec_cronos_v1:notificacion:'||emp||':'||(m->>'clave_operacion')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- La clave es de la persona: la de otra persona no se ve ni se revela.
 SELECT * INTO previa FROM vec_cronos_v1.notificacion n WHERE n.empleado_ref=emp AND n.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.solicitud_replay(decision_ref,empleado_ref,agregado_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,emp,previa.notificacion_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('notificacion_ref',previa.notificacion_ref,'recibo_ref',previa.recibo_ref,'instante_utc',previa.registrada_en,'replay',true);
 END IF;
 SELECT * INTO tipo FROM vec_cronos_v1.notificacion_tipos_vigentes_v1(k.ahora) t WHERE t.version_ref=m->>'tipo_version_ref';
 IF NOT FOUND THEN
   RAISE EXCEPTION 'tipo de notificación Cronos no vigente' USING ERRCODE='PC011';
 END IF;
 ref:='notificacion:cronos:'||gen_random_uuid()::text; recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.notificacion(notificacion_ref,clave_operacion,empleado_ref,actor_ref,perfil_ref,tipo_version_ref,fecha_referida,texto,
   adjunto_ref,adjunto_sha256,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',emp,m->>'actor_ref',m->>'perfil_ref',tipo.version_ref,fecha,texto,adjunto,huella,
   k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 -- El evento no lleva el texto ni el adjunto: sólo referencias.
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,ref,emp,'cronos.notificacion.registrada',
   jsonb_build_object('notificacion_ref',ref,'recibo_ref',recibo,'tipo_version_ref',tipo.version_ref),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('notificacion_ref',ref,'recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_notificaciones_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','zona_horaria']);
 emp:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('consumir_cronos_notificaciones_propio_v3_atestada','propio',
   'cronos.notificaciones.propio.consultar','notificaciones_propio','consultar_notificaciones_propio','notificaciones:cronos:'||emp,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.notificaciones_acceso(decision_ref,empleado_ref,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,emp,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('empleado_ref',emp,
   'tipos',coalesce((SELECT jsonb_agg(jsonb_build_object('tipo_version_ref',t.version_ref,'tipo_ref',t.tipo_ref,'nombre',t.nombre) ORDER BY t.orden,t.tipo_ref)
      FROM vec_cronos_v1.notificacion_tipos_vigentes_v1(k.ahora) t),'[]'::jsonb),
   'notificaciones',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'notificacion_ref',n.notificacion_ref,'tipo_ref',t.tipo_ref,'tipo_nombre',t.nombre,'fecha_referida',to_char(n.fecha_referida,'YYYY-MM-DD'),
      'texto',n.texto,'adjunto_ref',n.adjunto_ref,'adjunto_sha256',n.adjunto_sha256,'registrada_en',n.registrada_en,
      'estado',CASE WHEN a.atencion_ref IS NULL THEN 'registrada' ELSE 'atendida' END,'atendida_en',a.atendida_en)
      ORDER BY n.registrada_en DESC,n.notificacion_ref)
    FROM (SELECT * FROM vec_cronos_v1.notificacion n0 WHERE n0.empleado_ref=emp ORDER BY n0.registrada_en DESC,n0.notificacion_ref LIMIT 500) n
    JOIN vec_cronos_v1.notificacion_tipo t ON t.version_ref=n.tipo_version_ref
    LEFT JOIN vec_cronos_v1.notificacion_atencion a ON a.notificacion_ref=n.notificacion_ref),'[]'::jsonb));
 IF jsonb_array_length(resultado->'tipos')>100 THEN
   RAISE EXCEPTION 'catálogo Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_bandeja_notificaciones_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; propio text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','zona_horaria']);
 propio:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('consumir_cronos_notificaciones_bandeja_v3_atestada','rrhh',
   'cronos.notificaciones.bandeja.consultar','bandeja_notificaciones','consultar_bandeja_notificaciones','bandeja:cronos:notificaciones',NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.notificaciones_bandeja_acceso(decision_ref,resolutor_ref,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 -- Primero lo pendiente de atender y, después, lo más reciente; nunca lo propio.
 resultado:=jsonb_build_object('notificaciones',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'notificacion_ref',n.notificacion_ref,'empleado_ref',n.empleado_ref,'empleado_etiqueta',
      (SELECT r.empleado_etiqueta FROM vec_cronos_v1.permiso_resolutor r WHERE r.empleado_ref=n.empleado_ref AND r.paso='administracion'
         AND r.resolutor_ref=m->>'actor_ref' ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1),
      'tipo_ref',t.tipo_ref,'tipo_nombre',t.nombre,'fecha_referida',to_char(n.fecha_referida,'YYYY-MM-DD'),'texto',n.texto,
      'adjunto_ref',n.adjunto_ref,'adjunto_sha256',n.adjunto_sha256,'registrada_en',n.registrada_en,
      'atendida',n.atendida_en IS NOT NULL,'atendida_en',n.atendida_en) ORDER BY n.atendida_en IS NOT NULL,n.registrada_en DESC,n.notificacion_ref)
    FROM (SELECT n0.*,a.atendida_en FROM vec_cronos_v1.notificacion n0
            LEFT JOIN vec_cronos_v1.notificacion_atencion a ON a.notificacion_ref=n0.notificacion_ref
           WHERE vec_cronos_v1.resolutor_competente_v1(n0.empleado_ref,'administracion') AND n0.empleado_ref<>propio
           ORDER BY a.atendida_en IS NOT NULL,n0.registrada_en DESC,n0.notificacion_ref LIMIT 500) n
    JOIN vec_cronos_v1.notificacion_tipo t ON t.version_ref=n.tipo_version_ref),'[]'::jsonb));
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.atender_notificacion_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; actor text; previa vec_cronos_v1.notificacion_atencion%ROWTYPE; n vec_cronos_v1.notificacion%ROWTYPE;
 asignacion text; ref text; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','notificacion_ref','zona_horaria']);
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$' OR m->>'notificacion_ref' !~ '^notificacion:cronos:[0-9a-f-]{36}$' THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 actor:=m->>'actor_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('registrar_y_consumir_cronos_atencion_notificacion_v3_atestada','rrhh',
   'cronos.notificacion.atender','atencion_notificacion','atender_notificacion','notificacion:cronos:atencion:'||(m->>'clave_operacion'),
   ARRAY['vec_cronos_v1:atencion:'||actor||':'||(m->>'clave_operacion'),'vec_cronos_v1:notificacion:'||(m->>'notificacion_ref')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.notificacion_atencion x WHERE x.actor_ref=actor AND x.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.atencion_replay(decision_ref,resolutor_ref,atencion_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,actor,previa.atencion_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('atencion_ref',previa.atencion_ref,'notificacion_ref',previa.notificacion_ref,'recibo_ref',previa.recibo_ref,
     'instante_utc',previa.atendida_en,'replay',true);
 END IF;
 -- Sin competencia la notificación no es visible: ajena e inexistente igual.
 SELECT * INTO n FROM vec_cronos_v1.notificacion x WHERE x.notificacion_ref=m->>'notificacion_ref'
   AND vec_cronos_v1.resolutor_competente_v1(x.empleado_ref,'administracion');
 IF NOT FOUND OR n.empleado_ref=m->>'empleado_ref' THEN
   RAISE EXCEPTION 'atención no competente' USING ERRCODE='PC012';
 END IF;
 SELECT r.asignacion_ref INTO asignacion FROM vec_cronos_v1.permiso_resolutor r
  WHERE r.empleado_ref=n.empleado_ref AND r.paso='administracion' AND r.resolutor_ref=actor
    AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
    AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
  ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1;
 IF asignacion IS NULL THEN
   RAISE EXCEPTION 'atención no competente' USING ERRCODE='PC012';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_cronos_v1.notificacion_atencion x WHERE x.notificacion_ref=n.notificacion_ref) THEN
   RAISE EXCEPTION 'notificación ya atendida' USING ERRCODE='PC011';
 END IF;
 ref:='notificacion:cronos:atencion:'||gen_random_uuid()::text; recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.notificacion_atencion(atencion_ref,clave_operacion,notificacion_ref,empleado_ref,asignacion_ref,actor_ref,perfil_ref,
   material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,atendida_en)
 VALUES(ref,m->>'clave_operacion',n.notificacion_ref,n.empleado_ref,asignacion,actor,m->>'perfil_ref',
   k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,n.notificacion_ref,n.empleado_ref,'cronos.notificacion.atendida',
   jsonb_build_object('notificacion_ref',n.notificacion_ref,'atencion_ref',ref,'recibo_ref',recibo),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('atencion_ref',ref,'notificacion_ref',n.notificacion_ref,'recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

DO $acl$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.circuito_directo_vigente_v1(text,timestamptz)',
   'vec_cronos_v1.jefatura_asignada_v1(text,timestamptz)',
   'vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consumir_notificacion_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.notificacion_tipos_vigentes_v1(timestamptz)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.consultar_bandeja_permisos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.registrar_notificacion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_notificaciones_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_bandeja_notificaciones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.atender_notificacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f);
 END LOOP;
END $acl$;
COMMIT;
