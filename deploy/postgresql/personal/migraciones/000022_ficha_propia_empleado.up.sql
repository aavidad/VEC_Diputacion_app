\set ON_ERROR_STOP on
-- Ficha propia de la persona empleada («mis datos»): relaciones de servicio
-- y servicios reconocidos del empleado canónico de la propia persona, con las
-- denominaciones publicadas (régimen, modalidad, situación, clase de servicio,
-- unidad y puesto). Lectura nominal con consumo V3 (AD3-74) en la misma
-- transacción, recibo propio y comprobación, con la proyección gobernada
-- persona→empleado de Personal 000016, de que el empleado pedido es el de la
-- persona que consulta. Añade además el registro segregado de denegaciones de
-- su frontera HTTP. Instalar tras Personal 000021 y AD3-74. Sin DOWN tras
-- historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000022:ficha-propia',0));
SET LOCAL ROLE vec_personal_propietario;
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_personal.org_nodo_historia') IS NULL
    OR to_regclass('vec_personal.puesto_tipo_historia') IS NULL
    OR (SELECT count(*) FROM pg_catalog.pg_attribute a
        WHERE a.attrelid IN ('vec_personal.relacion_servicio_historia'::regclass,
          'vec_personal.situacion_empleado_historia'::regclass,
          'vec_personal.servicio_reconocido_historia'::regclass)
          AND a.attname='catalogo_snapshot' AND NOT a.attisdropped
          AND a.atttypid='jsonb'::regtype AND a.attnotnull)<>3
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_personal_registrador_frontera' AND NOT rolcanlogin)
    OR to_regclass('vec_personal.recibo_ficha_propia_empleado') IS NOT NULL
    OR to_regclass('vec_personal.denegacion_frontera_ficha_propia') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 000022: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El recibo conserva el empleado consultado (referencia opaca), huellas y
-- cardinalidad; nunca los datos de la ficha.
CREATE TABLE vec_personal.recibo_ficha_propia_empleado (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^fichapropia:[0-9a-f-]{36}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 400),
 consultada_en timestamptz(6) NOT NULL
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_ficha_propia_empleado
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_ficha_propia_empleado
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_ficha_propia_empleado ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_ficha_propia_empleado FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_ficha_propia_empleado
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.recibo_ficha_propia_empleado
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_registrador_frontera;

-- Denegaciones de la frontera HTTP de la ficha propia: correlación, motivo
-- cerrado, estado y, si la identidad ya estaba acreditada, el actor.
CREATE TABLE vec_personal.denegacion_frontera_ficha_propia (
 evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK(correlacion_ref='corr_no_disponible' OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'),
 motivo text NOT NULL,
 estado_http smallint NOT NULL,
 actor_ref text CHECK(actor_ref IS NULL OR (length(actor_ref) BETWEEN 1 AND 512 AND actor_ref ~ '^[A-Za-z0-9:_-]+$')),
 registrada_en timestamptz(6) NOT NULL,
 CHECK((motivo='peticion_invalida' AND estado_http=400)
    OR (motivo='autenticacion_requerida' AND estado_http=401 AND actor_ref IS NULL)
    OR (motivo='acceso_denegado' AND estado_http=403)
    OR (motivo='sin_empleado' AND estado_http=403)
    OR (motivo='empleado_ambiguo' AND estado_http=403)
    OR (motivo='no_encontrada' AND estado_http=404)
    OR (motivo='metodo_no_permitido' AND estado_http=405)
    OR (motivo='dependencia_no_disponible' AND estado_http=503))
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.denegacion_frontera_ficha_propia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.denegacion_frontera_ficha_propia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.denegacion_frontera_ficha_propia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.denegacion_frontera_ficha_propia FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.denegacion_frontera_ficha_propia
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.denegacion_frontera_ficha_propia
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_registrador_frontera;
REVOKE ALL ON SEQUENCE vec_personal.denegacion_frontera_ficha_propia_evento_id_seq
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_registrador_frontera;

-- Solo un LOGIN nominal miembro exclusivo de vec_personal_registrador_frontera
-- (sin SET ni ADMIN) registra; no lee ni modifica lo registrado.
CREATE FUNCTION vec_personal.registrar_denegacion_ficha_propia_v1(
 p_correlacion_ref text,p_motivo text,p_estado_http smallint,p_actor_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
 SET lock_timeout='1s' SET statement_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
       JOIN pg_catalog.pg_roles g ON g.oid=m.roleid AND g.rolname='vec_personal_registrador_frontera'
       JOIN pg_catalog.pg_roles s ON s.oid=m.member AND s.rolname=session_user
       WHERE m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles r
       WHERE r.rolname<>session_user AND r.rolname<>'vec_personal_registrador_frontera'
         AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER')) THEN
  RAISE EXCEPTION 'registrador de frontera Personal invalido' USING ERRCODE='42501';
 END IF;
 IF p_correlacion_ref IS NULL OR p_motivo IS NULL OR p_estado_http IS NULL THEN
  RAISE EXCEPTION 'denegacion de ficha propia invalida' USING ERRCODE='22023';
 END IF;
 BEGIN
  INSERT INTO vec_personal.denegacion_frontera_ficha_propia(correlacion_ref,motivo,estado_http,actor_ref,registrada_en)
  VALUES(p_correlacion_ref,p_motivo,p_estado_http,nullif(p_actor_ref,''),date_trunc('microseconds',clock_timestamp()));
 EXCEPTION WHEN check_violation THEN
  RAISE EXCEPTION 'denegacion de ficha propia invalida' USING ERRCODE='22023';
 END;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)
 TO vec_personal_registrador_frontera;

CREATE FUNCTION vec_personal.consultar_ficha_propia_empleado_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; consumo record; proyeccion record;
 fecha date; conocido timestamptz(6); empleado text; persona text;
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 relaciones jsonb; servicios jsonb; ajenas integer; ahora timestamptz(6); recibo text;
 campos constant jsonb:='["corte","evidencia","relaciones","servicios"]';
BEGIN
 IF session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR length(p_material)>4096 OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL
    OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'ficha propia denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de ficha propia inválido' USING ERRCODE='22023'; END;
 empleado:=m->>'empleado_ref'; persona:=m->>'persona_ref';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
      'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
      'empleado_ref','esquema','perfil_ref','perfil_version','persona_ref','persona_version','vigente_en']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.ficha-propia.consulta.v1'
    OR empleado IS NULL OR empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido)
    OR conocido>transaction_timestamp()
    OR fecha::text IS DISTINCT FROM m->>'vigente_en'
    OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR m->>'actor_ref' IS NULL OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' IS NULL OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'cuenta_ref' IS NULL OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' IS NULL OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR persona IS NULL OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'cuenta_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'perfil_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR m->>'actor_ref' IS DISTINCT FROM persona
    OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'material de ficha propia incompatible' USING ERRCODE='42501';
 END IF;
 -- json.Marshal de la estructura Go, en orden de campos y bytes UTF-8 exactos.
 material_canon:='{"esquema":"vec.personal.ficha-propia.consulta.v1","empleado_ref":'||to_jsonb(empleado)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(persona)::text||
  ',"persona_version":'||(m->>'persona_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de ficha propia no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||
  '},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"material_sha256":"'||material_sha||'","operacion":"ficha_propia"'||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.registro_empleado.ficha_propia.consultar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'ficha_propia_empleado'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_ficha_propia' OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->>'recurso_ref' IS DISTINCT FROM empleado
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM 'personal.registro_empleado.ficha_propia.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM empleado OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión de ficha propia divergente' USING ERRCODE='42501'; END IF;
 -- Autoridad propia de Personal: el empleado pedido debe ser, ahora, el único
 -- empleado canónico de la persona que consulta. Ausencia, ambigüedad,
 -- revocación o un empleado ajeno se deniegan igual.
 SELECT * INTO STRICT proyeccion
  FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,transaction_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado' OR proyeccion.empleado_ref IS DISTINCT FROM empleado
    OR proyeccion.persona_ref IS DISTINCT FROM persona THEN
  RAISE EXCEPTION 'empleado ajeno a la persona' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM empleado
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'consumo de ficha propia divergente' USING ERRCODE='42501'; END IF;
 -- La última revisión conocida de cada hecho manda. Una relación del empleado
 -- inscrita para otra persona es una incoherencia: no se muestra nada.
 SELECT count(*) INTO ajenas
 FROM (SELECT DISTINCT ON (relacion_ref) persona_ref FROM vec_personal.relacion_servicio_historia
       WHERE empleado_ref=empleado AND conocido_desde<=conocido
       ORDER BY relacion_ref,conocido_desde DESC,revision DESC) r
 WHERE r.persona_ref<>persona;
 IF ajenas<>0 THEN RAISE EXCEPTION 'ficha propia incoherente' USING ERRCODE='55000'; END IF;
 -- Cada relación se describe en su fecha de referencia: la de la consulta si
 -- sigue abierta, su último día si terminó antes o su inicio si es futura.
 -- El fin es el último día inclusive ([desde,hasta) semiabierto en origen).
 WITH rel AS (SELECT DISTINCT ON (relacion_ref) * FROM vec_personal.relacion_servicio_historia
     WHERE empleado_ref=empleado AND conocido_desde<=conocido
     ORDER BY relacion_ref,conocido_desde DESC,revision DESC),
 ref AS (SELECT r.*, CASE WHEN r.vigente_hasta IS NOT NULL AND r.vigente_hasta<=fecha THEN r.vigente_hasta-1
                          WHEN r.vigente_desde>fecha THEN r.vigente_desde ELSE fecha END AS en
         FROM rel r)
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'inicio',x.vigente_desde::text,
   'fin',coalesce((x.vigente_hasta-1)::text,''),
   'estado',x.estado,
   'regimen',coalesce(x.catalogo_snapshot #>> '{regimen,denominacion}',''),
   'modalidad',coalesce(x.catalogo_snapshot #>> '{modalidad,denominacion}',''),
   'unidad',coalesce((SELECT n.denominacion FROM (SELECT DISTINCT ON (nodo_ref) * FROM vec_personal.org_nodo_historia
       WHERE organismo_ref=x.organismo_ref AND unidad_ref=x.unidad_ref AND conocido_desde<=conocido
       ORDER BY nodo_ref,conocido_desde DESC,revision DESC) n
     WHERE NOT n.retirado AND n.vigente_desde<=x.en AND (n.vigente_hasta IS NULL OR x.en<n.vigente_hasta)
     ORDER BY n.nodo_ref LIMIT 1),''),
   'puesto',coalesce((SELECT pt.denominacion FROM (SELECT DISTINCT ON (ocupacion_ref) * FROM vec_personal.ocupacion_empleado_historia
       WHERE empleado_ref=empleado AND relacion_ref=x.relacion_ref AND conocido_desde<=conocido
       ORDER BY ocupacion_ref,conocido_desde DESC,revision DESC) o
     JOIN LATERAL (SELECT DISTINCT ON (puesto_ref) * FROM vec_personal.puesto_rpt_historia
       WHERE puesto_ref=o.puesto_ref AND conocido_desde<=conocido
       ORDER BY puesto_ref,conocido_desde DESC,revision DESC) pu ON true
     JOIN vec_personal.puesto_tipo_historia pt ON pt.tipo_ref=pu.tipo_ref AND pt.revision=pu.tipo_revision
     WHERE o.vigente_desde<=x.en AND (o.vigente_hasta IS NULL OR x.en<o.vigente_hasta)
     ORDER BY o.ocupacion_ref LIMIT 1),''),
   'situacion',coalesce((SELECT s.catalogo_snapshot #>> '{situacion,denominacion}' FROM (SELECT DISTINCT ON (situacion_ref) * FROM vec_personal.situacion_empleado_historia
       WHERE empleado_ref=empleado AND relacion_ref=x.relacion_ref AND conocido_desde<=conocido
       ORDER BY situacion_ref,conocido_desde DESC,revision DESC) s
     WHERE s.estado='vigente' AND s.vigente_desde<=x.en AND (s.vigente_hasta IS NULL OR x.en<s.vigente_hasta)
     ORDER BY s.vigente_desde DESC,s.situacion_ref LIMIT 1),''))
   ORDER BY x.vigente_desde DESC,x.relacion_ref),'[]'::jsonb)
 INTO relaciones FROM ref x;
 WITH srv AS (SELECT DISTINCT ON (servicio_ref) * FROM vec_personal.servicio_reconocido_historia
     WHERE empleado_ref=empleado AND conocido_desde<=conocido
     ORDER BY servicio_ref,conocido_desde DESC,revision DESC)
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'inicio',s.periodo_desde::text,'fin',s.periodo_hasta::text,
   'clase',coalesce(s.catalogo_snapshot #>> '{clase_servicio,denominacion}',''),
   'dias',s.dias_reconocidos,'estado',s.estado)
   ORDER BY s.periodo_desde DESC,s.servicio_ref),'[]'::jsonb)
 INTO servicios FROM srv s
 WHERE s.vigente_desde<=fecha AND (s.vigente_hasta IS NULL OR fecha<s.vigente_hasta);
 IF jsonb_array_length(relaciones)>200 OR jsonb_array_length(servicios)>200 THEN
  RAISE EXCEPTION 'ficha propia excede límite' USING ERRCODE='54000'; END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'ficha propia caducada' USING ERRCODE='42501'; END IF;
 recibo:='fichapropia:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_ficha_propia_empleado
  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora);
 RETURN jsonb_build_object('ficha',jsonb_build_object(
   'corte',jsonb_build_object('vigente_en',fecha::text,'conocido_en',m->>'conocido_en'),
   'relaciones',relaciones,'servicios',servicios),
   'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
   'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
