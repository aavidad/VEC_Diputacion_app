\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000074:transicion:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $dependencias$
DECLARE firma text; f oid;
BEGIN
 IF current_setting('server_encoding')<>'UTF8'
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=current_user::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND NOT rolcanlogin AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolbypassrls AND NOT rolreplication)
 OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN
    ('aplicar_incorporacion_seguimiento_v2','normalizar_salida_transicion_incorporacion_v2')) THEN
  RAISE EXCEPTION 'CT74: precondición incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH firma IN ARRAY ARRAY[
  'vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(jsonb)',
  'vec_contratacion_temporal.estado_seguimiento_canonico_v1(jsonb,jsonb)',
  'vec_contratacion_temporal.seguimiento73_nodo(jsonb,text)',
  'vec_contratacion_temporal.seguimiento73_micro(jsonb,boolean)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
    AND NOT prosecdef AND provolatile='i' AND proconfig @> ARRAY['search_path=pg_catalog']
    AND prorettype=CASE WHEN firma LIKE '%seguimiento73_micro(%' THEN 'bigint'::regtype ELSE 'bytea'::regtype END)
  OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner)) THEN
   RAISE EXCEPTION 'CT74: codec propietario requerido' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $dependencias$;

-- Normalización de representación DESPUÉS de validar el estado con CT73.
-- No calcula huellas ni reglas: refleja time.Time JSON/omitempty/slices de Go.
-- Helper privado acotado a los DTO de salida validados, no parser de documentos.
CREATE FUNCTION vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(j jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
AS $normalizar$
DECLARE r jsonb; k text; v jsonb; s text;
BEGIN
 IF j IS NULL OR pg_column_size(j)>8388608 OR jsonb_typeof(j) NOT IN ('object','array','string','number','boolean','null') THEN
  RAISE EXCEPTION 'CT74: salida inválida' USING ERRCODE='22023';
 END IF;
 IF jsonb_typeof(j)='array' THEN
  SELECT COALESCE(jsonb_agg(vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(value) ORDER BY ord),'[]'::jsonb)
   INTO r FROM jsonb_array_elements(j) WITH ORDINALITY x(value,ord);
  RETURN r;
 ELSIF jsonb_typeof(j)<>'object' THEN RETURN j;
 END IF;
 r:='{}'::jsonb;
 FOR k,v IN SELECT key,value FROM jsonb_each(j) LOOP
  IF k IN ('motivo_clave','rectifica_actuacion_ref') AND v='""'::jsonb THEN CONTINUE; END IF;
  IF k IN ('creado_en','actualizado_en','efectivo_en','registrada_en','calculado_en','desde','hasta') THEN
   PERFORM vec_contratacion_temporal.seguimiento73_micro(v,false);
   s:=v#>>'{}';
   s:=regexp_replace(regexp_replace(s,'(\.[0-9]*[1-9])0+Z$','\1Z'),'\.0+Z$','Z');
   v:=to_jsonb(s);
  ELSE v:=vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(v);
  END IF;
  IF k IN ('periodos_resultantes','documentos') AND v='[]'::jsonb THEN v:='null'::jsonb; END IF;
  IF k='documentos' AND jsonb_typeof(v)='array' THEN
   SELECT jsonb_agg(value ORDER BY (value->>'tipo_clave') COLLATE "C",(value->>'referencia') COLLATE "C") INTO v FROM jsonb_array_elements(v);
  END IF;
  r:=r||jsonb_build_object(k,v);
 END LOOP;
 RETURN r;
END $normalizar$;

-- ÚNICA API pública del corte, accesible sólo al propietario CT.
-- Es pura: no INSERT/UPDATE, no consultas a almacenes, no autoridad V3/Personal.
-- Estado anterior es entrada a validar, nunca un posterior aportado por caller.
CREATE FUNCTION vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(
 publicacion jsonb, anterior jsonb, material jsonb,
 actuacion_ref text, recibo_ref text, registrada_en text
) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
AS $aplicar$
DECLARE a jsonb; c jsonb; p jsonb; t jsonb; d jsonb; ev jsonb; vieja jsonb; documentos jsonb;
 version_actual numeric; version_esperada numeric; n integer; instante bigint; actualizado bigint;
 limite bigint; inicio bigint; fin bigint; canon bytea; hp text; ha text; anterior_hash text;
BEGIN
 IF publicacion IS NULL OR anterior IS NULL OR material IS NULL OR actuacion_ref IS NULL
 OR recibo_ref IS NULL OR registrada_en IS NULL
 OR pg_column_size(publicacion)>8388608 OR pg_column_size(anterior)>8388608 OR pg_column_size(material)>1048576
 OR length(registrada_en)>64 OR length(actuacion_ref)<>68 OR length(recibo_ref)<>68 THEN
  RAISE EXCEPTION 'CT74: entrada inválida' USING ERRCODE='22023';
 END IF;
 -- Contrato material completo (incluye original Personal y contexto); NO permiso.
 PERFORM vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(material);
 canon:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(publicacion,anterior);
 IF canon IS NULL OR octet_length(canon) NOT BETWEEN 1 AND 8388608 THEN
  RAISE EXCEPTION 'CT74: estado inválido' USING ERRCODE='22023';
 END IF;
 a:=vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(anterior);
 c:=material->'Confirmacion'; p:=material->'Preparacion';
 version_actual:=(a->>'version')::numeric;
 version_esperada:=(c->>'VersionSeguimientoEsperada')::numeric;
 -- Como RestaurarSeguimientoPersistido: la versión se coteja ANTES de Aplicar.
 IF version_actual IS DISTINCT FROM version_esperada
 OR a->'organizacion_ref' IS DISTINCT FROM p->'OrganizacionRef'
 OR a->'expediente_ref' IS DISTINCT FROM c->'SolicitudPersonal'->'expediente_ref'
 OR a->'relacion_ref' IS DISTINCT FROM c->'ResultadoPersonal'->'relacion_ref' THEN
  RAISE EXCEPTION 'CT74: vínculos o versión incompatibles' USING ERRCODE='22023';
 END IF;
 instante:=vec_contratacion_temporal.seguimiento73_micro(to_jsonb(registrada_en),false);
 registrada_en:=regexp_replace(regexp_replace(registrada_en,'(\.[0-9]*[1-9])0+Z$','\1Z'),'\.0+Z$','Z');
 documentos:=c->'Documentos';
 d:=jsonb_build_object('actuacion_ref',actuacion_ref,'transicion_clave','confirmar_incorporacion',
   'motivo_clave',c->'MotivoClave','actor_ref',p->'ActorRef','unidad_ref',p->'UnidadRef',
   'efectivo_en',c->'PeriodoIncorporacion'->'desde','registrada_en',registrada_en,
   'documentos',documentos,'periodo',c->'PeriodoIncorporacion',
   'recibo_ref',recibo_ref,'correlacion_ref',p->'CorrelacionRef');
 -- Nodo CT73 produce cabecera/petición BINARIA V1 y valida referencias/instantes.
 hp:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(d,'peticion')),'hex');
 n:=jsonb_array_length(a->'actuaciones');
 FOR vieja IN SELECT value FROM jsonb_array_elements(a->'actuaciones') LOOP
  IF vieja->>'actuacion_ref'=actuacion_ref THEN
   IF vieja->>'huella_peticion_sha256'<>hp THEN
    RAISE EXCEPTION 'CT74: actuación en conflicto' USING ERRCODE='22023';
   END IF;
   -- El puente devuelve la ÚLTIMA actuación, no cambia historia ni fecha.
   ev:=a->'actuaciones'->(n-1);
   RETURN jsonb_build_object('estado',a,'evento',ev,
     'estado_canonico_base64',replace(encode(canon,'base64'),chr(10),''),
     'estado_sha256',encode(sha256(canon),'hex'));
  END IF;
 END LOOP;
 actualizado:=vec_contratacion_temporal.seguimiento73_micro(a->'actualizado_en',false);
 inicio:=vec_contratacion_temporal.seguimiento73_micro(publicacion->'vigencia'->'desde',false);
 limite:=vec_contratacion_temporal.seguimiento73_micro(publicacion->'vigencia'->'hasta',true);
 IF n>=10000 OR instante<actualizado OR instante<inicio
 OR (limite<>-62135596800000000 AND instante>=limite) THEN
  RAISE EXCEPTION 'CT74: transición fuera de vigencia' USING ERRCODE='22023';
 END IF;
 SELECT value INTO STRICT t FROM jsonb_array_elements(publicacion->'transiciones')
  WHERE value->>'clave'='confirmar_incorporacion';
 -- Material SIEMPRE aporta período y nunca rectifica/calendario. Por dominio,
 -- sólo ordinaria abrir/ampliar puede satisfacer esta petición; nada hardcoded
 -- sobre nombres o carácter final del estado destino de la definición.
 IF t->'origen' IS DISTINCT FROM a->'estado_actual'
 OR t->>'clase'<>'ordinaria' OR t->'requiere_periodo' IS DISTINCT FROM 'true'::jsonb
 OR t ? 'calendario' OR t->>'efecto_periodo' NOT IN ('abrir','ampliar')
 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(t->'motivos_permitidos') x WHERE x.value=c->'MotivoClave')
 OR EXISTS (SELECT 1 FROM jsonb_array_elements(documentos) x WHERE NOT EXISTS
     (SELECT 1 FROM jsonb_array_elements(t->'documentos') r WHERE r.value->'tipo_clave'=x.value->'tipo_clave'))
 OR EXISTS (SELECT 1 FROM jsonb_array_elements(t->'documentos') r WHERE r.value->'obligatorio'='true'::jsonb
     AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(documentos) x WHERE x.value->'tipo_clave'=r.value->'tipo_clave')) THEN
  RAISE EXCEPTION 'CT74: requisitos de definición incumplidos' USING ERRCODE='22023';
 END IF;
 inicio:=vec_contratacion_temporal.seguimiento73_micro(c->'PeriodoIncorporacion'->'desde',false);
 fin:=vec_contratacion_temporal.seguimiento73_micro(c->'PeriodoIncorporacion'->'hasta',false);
 IF fin<=inicio OR a ? 'cese_efectivo' THEN
  RAISE EXCEPTION 'CT74: período incompatible' USING ERRCODE='22023';
 END IF;
 IF t->>'efecto_periodo'='abrir' THEN
  IF a->'periodos_resultantes' NOT IN ('null'::jsonb,'[]'::jsonb) THEN
   RAISE EXCEPTION 'CT74: período ya abierto' USING ERRCODE='22023';
  END IF;
 ELSE
  IF jsonb_typeof(a->'periodos_resultantes') IS DISTINCT FROM 'array'
   OR jsonb_array_length(a->'periodos_resultantes')=0 THEN
   RAISE EXCEPTION 'CT74: período ausente' USING ERRCODE='22023';
  END IF;
  vieja:=a->'periodos_resultantes'->(jsonb_array_length(a->'periodos_resultantes')-1);
  IF inicio<vec_contratacion_temporal.seguimiento73_micro(vieja->'intervalo'->'hasta',false) THEN
   RAISE EXCEPTION 'CT74: período solapado' USING ERRCODE='22023';
  END IF;
 END IF;
 anterior_hash:=CASE WHEN n=0 THEN a->>'huella_raiz_sha256'
  ELSE a->'actuaciones'->(n-1)->>'huella_actuacion_sha256' END;
 ev:=d||jsonb_build_object('secuencia',version_actual+1,'version_seguimiento',version_actual+1,
   'definicion',a->'definicion','clase',t->'clase','estado_origen',t->'origen','estado_destino',t->'destino',
   'huella_peticion_sha256',hp,'huella_anterior_sha256',anterior_hash,'huella_actuacion_sha256','');
 -- El nodo no incluye la huella propia en su preimagen (no autorreferencia).
 ha:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(ev,'actuacion')),'hex');
 ev:=ev||jsonb_build_object('huella_actuacion_sha256',ha);
 a:=a||jsonb_build_object('version',version_actual+1,'estado_actual',t->'destino',
   'actualizado_en',registrada_en,'actuaciones',(a->'actuaciones')||jsonb_build_array(ev),
   'periodos_resultantes',CASE WHEN a->'periodos_resultantes'='null'::jsonb THEN '[]'::jsonb
      ELSE a->'periodos_resultantes' END ||jsonb_build_array(jsonb_build_object('intervalo',c->'PeriodoIncorporacion','actuacion_ref',actuacion_ref)));
 -- Barrera de dominio completa sobre lo calculado. Reproduce toda la historia
 -- y compara sus huellas; no basta el escritor binario del evento.
 canon:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(publicacion,a);
 IF canon IS NULL OR octet_length(canon) NOT BETWEEN 1 AND 8388608 THEN
  RAISE EXCEPTION 'CT74: posterior incompatible' USING ERRCODE='22023';
 END IF;
 RETURN jsonb_build_object('estado',a,'evento',ev,
   'estado_canonico_base64',replace(encode(canon,'base64'),chr(10),''),
   'estado_sha256',encode(sha256(canon),'hex'));
EXCEPTION WHEN OTHERS THEN
 -- QUERY_CANCELED no es capturada por OTHERS. Ningún detalle de material sale.
 RAISE EXCEPTION 'CT74: incorporación de seguimiento inválida' USING ERRCODE='22023';
END $aplicar$;

DO $acl$
DECLARE f record; r record;
BEGIN
 FOR f IN SELECT p.oid,p.oid::regprocedure AS firma FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN
     ('aplicar_incorporacion_seguimiento_v2','normalizar_salida_transicion_incorporacion_v2') LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.firma);
  FOR r IN SELECT DISTINCT x.rolname FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
      JOIN pg_roles x ON x.oid=a.grantee WHERE p.oid=f.oid AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.firma,r.rolname);
  END LOOP;
  EXECUTE format('COMMENT ON FUNCTION %s IS %L',f.firma,'CT74:transicion-pura-incorporacion-v2;sin-autoridad-ni-commit');
 END LOOP;
END $acl$;
COMMIT;
