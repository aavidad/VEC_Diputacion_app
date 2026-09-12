\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000075:registro:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $dependencias$
DECLARE firma text; f oid; nombre text;
BEGIN
 IF current_setting('server_encoding')<>'UTF8' OR NOT EXISTS (SELECT 1 FROM pg_roles
  WHERE rolname=current_user AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=current_user::regrole)
 OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE 'incorporacion75_%' OR p.proname='registrar_incorporacion_ejercicio_v2')) THEN
  RAISE EXCEPTION 'CT75: instalación incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH nombre IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2'] LOOP
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_contratacion_temporal' AND c.relname=nombre)
  OR EXISTS (SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='vec_contratacion_temporal' AND t.typname=nombre) THEN
   RAISE EXCEPTION 'CT75: objeto preexistente' USING ERRCODE='55000';
  END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2','expediente_integral_actual','expediente_version_integral'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_contratacion_temporal' AND c.relname=nombre AND c.relkind='r'
    AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CT75: almacén propietario requerido' USING ERRCODE='55000';
  END IF;
 END LOOP;
 FOREACH firma IN ARRAY ARRAY[
  'vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(jsonb)',
  'vec_contratacion_temporal.contexto_incorporacion_ejercicio_sha256_v2(jsonb)',
  'vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(jsonb)',
  'vec_contratacion_temporal.intencion_registro_incorporacion_sha256_v2(jsonb)',
  'vec_contratacion_temporal.estado_seguimiento_canonico_v1(jsonb,jsonb)',
  'vec_contratacion_temporal.seguimiento73_nodo(jsonb,text)',
  'vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(jsonb,jsonb,jsonb,text,text,text)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole AND provolatile='i' AND NOT prosecdef)
  OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner) THEN
   RAISE EXCEPTION 'CT75: codec propietario requerido' USING ERRCODE='55000';
  END IF;
 END LOOP;
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_personal.acreditar_alta_ejercicio_v1(text,text,text,bigint,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') OR NOT EXISTS (
   SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.oid=f
    AND p.proowner=n.nspowner AND p.prosecdef AND p.provolatile='v') THEN
   RAISE EXCEPTION 'CT75: fachada nominal requerida' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $dependencias$;

-- Solo historia CT; la actuación real está en estado CT72 y evento outbox.
CREATE TABLE vec_contratacion_temporal.incorporacion_registro_v2 (
 recibo_ref text PRIMARY KEY CHECK (recibo_ref ~ '^ref:[0-9a-f]{64}$'),
 seguimiento_ref text NOT NULL REFERENCES vec_contratacion_temporal.seguimiento_raiz_v2,
 organizacion_ref text NOT NULL, idempotencia_ref text NOT NULL, solicitud_ref text NOT NULL,
 expediente_ref text NOT NULL, version_expediente numeric(20,0) NOT NULL,
 version_anterior numeric(20,0) NOT NULL, version_resultante numeric(20,0) NOT NULL,
 estado_anterior_sha256 text NOT NULL, estado_resultante_sha256 text NOT NULL,
 material_json jsonb NOT NULL, material_canonico bytea NOT NULL, material_sha256 text NOT NULL,
 intencion_canonica bytea NOT NULL, intencion_sha256 text NOT NULL,
 exportacion_ct bytea[] NOT NULL CHECK (array_ndims(exportacion_ct)=1 AND array_lower(exportacion_ct,1)=1 AND cardinality(exportacion_ct)=8 AND array_position(exportacion_ct,NULL) IS NULL),
 persona_version numeric(20,0) NOT NULL CHECK (persona_version BETWEEN 1 AND 9007199254740991),
 perfil_version numeric(20,0) NOT NULL CHECK (perfil_version BETWEEN 1 AND 9007199254740991),
 recibo_json jsonb NOT NULL, auditoria_ref text NOT NULL UNIQUE, outbox_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
 evidencia_orden_json jsonb NOT NULL CHECK (jsonb_typeof(evidencia_orden_json)='object'),
 UNIQUE (organizacion_ref,idempotencia_ref), UNIQUE (seguimiento_ref,version_resultante),
 FOREIGN KEY (expediente_ref,version_expediente) REFERENCES vec_contratacion_temporal.expediente_version_integral,
 FOREIGN KEY (seguimiento_ref,version_anterior,estado_anterior_sha256) REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento,estado_sha256),
 FOREIGN KEY (seguimiento_ref,version_resultante,estado_resultante_sha256) REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento,estado_sha256),
 CHECK (version_anterior BETWEEN 0 AND 9007199254740990 AND version_resultante=version_anterior+1),
 CHECK (octet_length(material_canonico) BETWEEN 1 AND 1048576 AND material_sha256=encode(sha256(material_canonico),'hex')),
 CHECK (octet_length(intencion_canonica) BETWEEN 1 AND 1048576 AND intencion_sha256=encode(sha256(intencion_canonica),'hex')),
 CHECK (jsonb_typeof(material_json)='object' AND jsonb_typeof(recibo_json)='object'),
 CHECK ((recibo_json->>'MaterialOriginalSHA256'=material_sha256 AND recibo_json->>'IntencionSHA256'=intencion_sha256
  AND recibo_json->>'SeguimientoRef'=seguimiento_ref AND recibo_json->>'AuditoriaCTRef'=auditoria_ref
  AND recibo_json->>'OutboxCTRef'=outbox_ref AND recibo_json#>>'{Transicion,recibo_ref}'=recibo_ref
  AND recibo_json->'EjercicioSintetico'='true'::jsonb AND recibo_json->'FirmaOficial'='false'::jsonb AND recibo_json->'EficaciaAdministrativa'='false'::jsonb) IS TRUE)
);
CREATE TABLE vec_contratacion_temporal.incorporacion_auditoria_v2 (
 auditoria_ref text PRIMARY KEY, recibo_ref text NOT NULL REFERENCES vec_contratacion_temporal.incorporacion_registro_v2 DEFERRABLE INITIALLY DEFERRED,
 recuperado boolean NOT NULL, decision_ct_ref text NOT NULL UNIQUE, consumo_ct_sha256 text NOT NULL UNIQUE,
 decision_lectura_ref text NOT NULL UNIQUE, consumo_lectura_sha256 text NOT NULL UNIQUE,
 consumos_json jsonb NOT NULL, registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
 CHECK (consumo_ct_sha256 ~ '^[0-9a-f]{64}$' AND consumo_lectura_sha256 ~ '^[0-9a-f]{64}$'),
 CHECK ((consumos_json->>'DecisionCTRef'=decision_ct_ref AND consumos_json->>'ConsumoCTSHA256'=consumo_ct_sha256
  AND consumos_json->>'DecisionLecturaRef'=decision_lectura_ref AND consumos_json->>'ConsumoLecturaSHA256'=consumo_lectura_sha256) IS TRUE)
);
CREATE TABLE vec_contratacion_temporal.incorporacion_outbox_v2 (
 outbox_ref text PRIMARY KEY, recibo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.incorporacion_registro_v2 DEFERRABLE INITIALLY DEFERRED,
 evento_json jsonb NOT NULL CHECK (jsonb_typeof(evento_json)='object'),
 estado_sha256 text NOT NULL CHECK (estado_sha256 ~ '^[0-9a-f]{64}$'),
 creada_en timestamptz(6) NOT NULL CHECK (isfinite(creada_en)),
 CHECK ((evento_json->>'recibo_ref'=recibo_ref) IS TRUE)
);
ALTER TABLE vec_contratacion_temporal.incorporacion_registro_v2
 ADD CONSTRAINT incorporacion_registro_v2_auditoria_fk FOREIGN KEY (auditoria_ref) REFERENCES vec_contratacion_temporal.incorporacion_auditoria_v2 DEFERRABLE INITIALLY DEFERRED,
 ADD CONSTRAINT incorporacion_registro_v2_outbox_fk FOREIGN KEY (outbox_ref) REFERENCES vec_contratacion_temporal.incorporacion_outbox_v2 DEFERRABLE INITIALLY DEFERRED;
DO $tablas$
DECLARE n text; a record;
BEGIN
 FOREACH n IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2'] LOOP
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true)',n);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',n);
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',n);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace,
   LATERAL aclexplode(c.relacl) x WHERE ns.nspname='vec_contratacion_temporal' AND c.relname=n AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',n,pg_get_userbyid(a.grantee));
  END LOOP;
  EXECUTE format('COMMENT ON TABLE vec_contratacion_temporal.%I IS %L',n,'CT75:registro-incorporacion-v2:inmutable');
 END LOOP;
END $tablas$;

CREATE FUNCTION vec_contratacion_temporal.incorporacion75_instante(t timestamptz) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog
BEGIN ATOMIC
 SELECT regexp_replace(regexp_replace(to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'(\.[0-9]*[1-9])0+Z$','\1Z'),'\.0+Z$','Z');
END;

-- Ocho piezas y dos versiones exactas; no MAC/criptografía duplicada.
CREATE FUNCTION vec_contratacion_temporal.incorporacion75_piezas(p bytea[], pv numeric, fv numeric) RETURNS void
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE i int; maximos int[]:=ARRAY[32768,524288,65536,262144,1048576,1048576,262144,44];
BEGIN
 IF p IS NULL OR array_ndims(p) IS DISTINCT FROM 1 OR array_lower(p,1) IS DISTINCT FROM 1 OR cardinality(p)<>8
 OR pv IS NULL OR fv IS NULL OR pv NOT BETWEEN 1 AND 9007199254740991 OR fv NOT BETWEEN 1 AND 9007199254740991
 OR scale(pv)<>0 OR scale(fv)<>0 THEN RAISE EXCEPTION 'CT75: exportación inválida' USING ERRCODE='22023'; END IF;
 FOR i IN 1..8 LOOP
  IF p[i] IS NULL OR octet_length(p[i]) NOT BETWEEN 1 AND maximos[i] THEN
   RAISE EXCEPTION 'CT75: exportación inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF octet_length(p[1])<512 OR octet_length(p[8])<>44 THEN RAISE EXCEPTION 'CT75: exportación inválida' USING ERRCODE='22023'; END IF;
END $$;

CREATE FUNCTION vec_contratacion_temporal.incorporacion75_ventana(c jsonb,d jsonb,ahora timestamptz) RETURNS timestamptz
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE s text; ci timestamptz; cf timestamptz; di timestamptz; df timestamptz; limite timestamptz;
BEGIN
 FOREACH s IN ARRAY ARRAY[c->>'emitida_en',c->>'expira_en',d->>'emitida_en',d->>'valida_hasta',c->>'configuracion_expira_en',c->>'raiz_valida_hasta'] LOOP
  IF vec_contratacion_temporal.instante_incorporacion_go_v2(s,false) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT75: ventana inválida' USING ERRCODE='42501'; END IF;
 END LOOP;
 ci:=(c->>'emitida_en')::timestamptz; cf:=(c->>'expira_en')::timestamptz;
 di:=(d->>'emitida_en')::timestamptz; df:=(d->>'valida_hasta')::timestamptz;
 limite:=least(cf,df,(c->>'configuracion_expira_en')::timestamptz,(c->>'raiz_valida_hasta')::timestamptz);
 IF ahora IS NULL OR NOT isfinite(ahora) OR ahora<ci OR ahora<di OR ahora>=limite OR ci<di OR cf>df OR cf<=ci OR cf-ci>interval '5 seconds' THEN
  RAISE EXCEPTION 'CT75: permiso vencido' USING ERRCODE='42501'; END IF;
 RETURN limite;
END $$;

-- Validación de ligaduras; el permiso REAL se consume exclusivamente por AD3.
CREATE FUNCTION vec_contratacion_temporal.incorporacion75_autoridad(m jsonb,p bytea[],pv numeric,fv numeric,ahora timestamptz) RETURNS text
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE c jsonb; d jsonb; v jsonb; x jsonb; h text; motivo bytea; k text; bloque bytea; b bytea:=''::bytea; bloques bytea[];
BEGIN
 PERFORM vec_contratacion_temporal.incorporacion75_piezas(p,pv,fv);
 PERFORM vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(m);
 c:=convert_from(p[1],'UTF8')::jsonb; d:=convert_from(p[2],'UTF8')::jsonb; x:=convert_from(p[4],'UTF8')::jsonb; v:=m->'Vinculo';
 h:=vec_contratacion_temporal.contexto_incorporacion_ejercicio_sha256_v2(m);
 motivo:=convert_to('{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":'||
  vec_contratacion_temporal.nodo_incorporacion_canonico_v2(m->'MotivoV3','motivo')||'}','UTF8');
 -- Decisión V3 fija seis decimales; MaterialCanonico usa RFC3339Nano.
 FOREACH k IN ARRAY ARRAY['autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en'] LOOP
  v:=jsonb_set(v,ARRAY[k],to_jsonb(to_char((v->>k)::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 END LOOP;
 IF d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
 OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.incorporacion.confirmar'
 OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'confirmacion_incorporacion_ejercicio_v2'
 OR d->>'finalidad' IS DISTINCT FROM 'registrar_incorporacion_confirmada_por_personal'
 OR d->>'recurso_ref' IS DISTINCT FROM m#>>'{Confirmacion,SolicitudPersonal,expediente_ref}'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR d->'vinculo_autenticacion_actor' IS DISTINCT FROM v
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'CorrelacionV3'
 OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id' OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
 OR p[3] IS DISTINCT FROM motivo OR p[4] IS DISTINCT FROM vec_contratacion_temporal.bytes_incorporacion_v2(m->'ContextoCanonico')
 OR (x->>'persona_version')::numeric IS DISTINCT FROM pv OR (x->>'perfil_version')::numeric IS DISTINCT FROM fv
 OR c->>'contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref' OR c->>'huella_contexto_sha256' IS DISTINCT FROM v->>'contexto_actor_huella_sha256'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.incorporacion_ejercicio.v2'
 OR c->>'operacion' IS DISTINCT FROM d->>'accion' OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref'
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM h OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
 OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p[2]),'hex')
 OR c->>'huella_motivo_sha256' IS DISTINCT FROM encode(sha256(p[3]),'hex') THEN
  RAISE EXCEPTION 'CT75: autoridad no ligada' USING ERRCODE='42501'; END IF;
 PERFORM vec_contratacion_temporal.incorporacion75_ventana(c,d,ahora);
 -- HuellaConjuntoSHA256 Go: longitud uint64BE antes de CADA bloque.
 bloques:=ARRAY[p[1]];
 FOREACH k IN ARRAY ARRAY['decision_ref','huella_decision_sha256','huella_motivo_sha256','contexto_ref','huella_contexto_sha256','operacion','efecto_ref','huella_efecto_sha256','audiencia_consumo'] LOOP
  bloques:=array_append(bloques,convert_to(c->>k,'UTF8'));
 END LOOP;
 bloques:=bloques||ARRAY[convert_to(vec_contratacion_temporal.incorporacion75_instante((c->>'emitida_en')::timestamptz),'UTF8'),
  convert_to(vec_contratacion_temporal.incorporacion75_instante((c->>'expira_en')::timestamptz),'UTF8'),p[2],p[3],p[4],int8send(pv::bigint),int8send(fv::bigint),p[5],p[6],p[7],p[8]];
 FOREACH bloque IN ARRAY bloques LOOP
  IF bloque IS NULL THEN RAISE EXCEPTION 'CT75: huella inválida' USING ERRCODE='42501'; END IF;
  b:=b||int8send(octet_length(bloque)::bigint)||bloque;
 END LOOP;
 RETURN encode(sha256(b),'hex');
END $$;

-- Canon público Go de solicitud V3: no serializa tipos opacos ni concede permiso.
CREATE FUNCTION vec_contratacion_temporal.incorporacion75_solicitud_sha256(m jsonb) RETURNS text
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE v jsonb; r jsonb; k text; t text; s text; partes text[]; mapas text[]:='{}';
BEGIN
 r:=vec_contratacion_temporal.recurso_incorporacion_ejercicio_v2(m); v:=m->'Vinculo'; partes:='{}';
 FOREACH k IN ARRAY ARRAY['esquema','bloque_version','autenticacion_ref','autenticacion_huella_sha256','asercion_ref','sesion_ref','control_sesion_ref','control_sesion_revision','control_sesion_huella_sha256','cuenta_ref','cuenta_ordinaria_ref','principal_id','perfil_activo_ref','cuenta_privilegiada','superficie','metodo_observado','garantia_observada','politica_garantia_ref','politica_garantia_huella_sha256','autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en','registro_contexto_ref','contexto_actor_esquema','contexto_actor_ref','contexto_actor_version','contexto_actor_cuenta_version','contexto_actor_huella_sha256','manifiesto_procedencia_huella_sha256','autoridad_efectiva'] LOOP
  IF k=ANY(ARRAY['autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en']) THEN
   t:=vec_contratacion_temporal.texto_json_incorporacion_go_v2(to_char((v->>k)::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
  ELSIF jsonb_typeof(v->k) IN ('number','boolean') THEN t:=v->>k;
  ELSE t:=vec_contratacion_temporal.texto_json_incorporacion_go_v2(v->>k); END IF;
  IF t IS NULL THEN RAISE EXCEPTION 'CT75: solicitud incompleta' USING ERRCODE='22023'; END IF;
  partes:=array_append(partes,vec_contratacion_temporal.texto_json_incorporacion_go_v2(k)||':'||t);
 END LOOP;
 s:='{"esquema":"vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2","vinculo_autenticacion_actor":{'||array_to_string(partes,',')||'},"accion":"contratacion_temporal.incorporacion.confirmar","recurso":';
 FOREACH k IN ARRAY ARRAY['ambitos','atributos'] LOOP
  SELECT '['||coalesce(string_agg('{"clave":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(e.key)||',"valor":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(e.value)||'}',',' ORDER BY e.key COLLATE "C"),'')||']'
   INTO t FROM jsonb_each_text(r->k) e;
  mapas:=array_append(mapas,t);
 END LOOP;
 s:=s||'{"referencia":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(r->>'referencia')||',"modulo_id":"contratacion_temporal","tipo":"confirmacion_incorporacion_ejercicio_v2","ambitos":'||mapas[1]||',"atributos":'||mapas[2]||'},"finalidad":"registrar_incorporacion_confirmada_por_personal","correlacion_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(m->>'CorrelacionV3')||',"referencia_motivo":'||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(m->'MotivoV3','motivo')||'}';
 RETURN encode(sha256(convert_to(s,'UTF8')),'hex');
END $$;

-- Captura del DTO de B: integridad/ligadura, NO fábrica nominal ni lectura RBAC.
-- El propietario de autenticación/contexto/RBAC/concesión debe releer originales.
CREATE FUNCTION vec_contratacion_temporal.incorporacion75_evidencia(e jsonb,m jsonb,p bytea[],pv numeric,fv numeric,ahora timestamptz) RETURNS void
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE esperado jsonb; d jsonb; x jsonb; k text; mc bytea; he text; hs text;
 preparado timestamptz; evaluada timestamptz; concedida timestamptz;
BEGIN
 IF e IS NULL OR pg_column_size(e)>16777216 THEN RAISE EXCEPTION 'CT75: evidencia ausente' USING ERRCODE='22023'; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(e,ARRAY['Esquema','PreparadoEn','EvaluadaEn','ConcesionRegistradaEn','MaterialCanonico','MaterialSHA256','IntencionSHA256','SolicitudSHA256','ContextoOriginalRef','CapacidadCanonica','DecisionCanonica','MotivoCanonico','ContextoActorCanonico','PersonaVersion','PerfilVersion','PayloadVECAD3','SobreCOSESign1','EvidenciaVerificacion','RaizPublicaSPKI','HuellaExportacionSHA256']);
 FOREACH k IN ARRAY ARRAY['PreparadoEn','EvaluadaEn','ConcesionRegistradaEn'] LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_micro(e->k,false);
 END LOOP;
 preparado:=(e->>'PreparadoEn')::timestamptz; evaluada:=(e->>'EvaluadaEn')::timestamptz; concedida:=(e->>'ConcesionRegistradaEn')::timestamptz;
 he:=vec_contratacion_temporal.incorporacion75_autoridad(m,p,pv,fv,evaluada);
 PERFORM vec_contratacion_temporal.incorporacion75_autoridad(m,p,pv,fv,ahora);
 d:=convert_from(p[2],'UTF8')::jsonb; x:=convert_from(p[4],'UTF8')::jsonb;
 IF evaluada<preparado OR evaluada>ahora OR concedida>evaluada
 OR concedida<(d->>'emitida_en')::timestamptz OR concedida>=(d->>'valida_hasta')::timestamptz
 OR preparado<(m#>>'{Vinculo,autenticacion_verificada_en}')::timestamptz
 OR preparado<(m#>>'{Vinculo,sesion_revalidada_en}')::timestamptz
 OR preparado>=(m#>>'{Vinculo,sesion_valida_hasta}')::timestamptz
 OR preparado<(x->>'resuelto_en')::timestamptz OR preparado<(x->>'vigente_desde')::timestamptz
 OR preparado>=(x->>'vigente_hasta')::timestamptz OR preparado<(m#>>'{Personal,RegistradoEn}')::timestamptz THEN
  RAISE EXCEPTION 'CT75: tiempos originales incompatibles' USING ERRCODE='22023'; END IF;
 mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(m);
 hs:=vec_contratacion_temporal.incorporacion75_solicitud_sha256(m);
 IF d->>'esquema_huella_solicitud' IS DISTINCT FROM 'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2'
 OR d->>'solicitud_huella_sha256' IS DISTINCT FROM hs THEN RAISE EXCEPTION 'CT75: solicitud no ligada' USING ERRCODE='42501'; END IF;
 esperado:=jsonb_build_object('Esquema','vec.contratacion-temporal.incorporacion.ejercicio.orden-original.v2',
  'PreparadoEn',vec_contratacion_temporal.incorporacion75_instante(preparado),'EvaluadaEn',vec_contratacion_temporal.incorporacion75_instante(evaluada),'ConcesionRegistradaEn',vec_contratacion_temporal.incorporacion75_instante(concedida),
  'MaterialCanonico',replace(encode(mc,'base64'),chr(10),''),'MaterialSHA256',encode(sha256(mc),'hex'),
  'IntencionSHA256',vec_contratacion_temporal.intencion_registro_incorporacion_sha256_v2(m),'SolicitudSHA256',hs,'ContextoOriginalRef',m#>'{Vinculo,registro_contexto_ref}',
  'CapacidadCanonica',replace(encode(p[1],'base64'),chr(10),''),'DecisionCanonica',replace(encode(p[2],'base64'),chr(10),''),
  'MotivoCanonico',replace(encode(p[3],'base64'),chr(10),''),'ContextoActorCanonico',replace(encode(p[4],'base64'),chr(10),''),
  'PersonaVersion',pv,'PerfilVersion',fv,'PayloadVECAD3',replace(encode(p[5],'base64'),chr(10),''),'SobreCOSESign1',replace(encode(p[6],'base64'),chr(10),''),
  'EvidenciaVerificacion',replace(encode(p[7],'base64'),chr(10),''),'RaizPublicaSPKI',replace(encode(p[8],'base64'),chr(10),''),'HuellaExportacionSHA256',he);
 IF e IS DISTINCT FROM esperado THEN RAISE EXCEPTION 'CT75: captura original incompatible' USING ERRCODE='22023'; END IF;
END $$;

CREATE FUNCTION vec_contratacion_temporal.incorporacion75_recibo(m jsonb,seg text,pub jsonb,a jsonb,p jsonb,ev jsonb,consumos jsonb,aud text,ob text) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE mc bytea; ca bytea; cp bytea; t jsonb;
BEGIN
 mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(m);
 ca:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,a); cp:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,p);
 t:=ev-ARRAY['secuencia','version_seguimiento','definicion','clase','estado_origen','estado_destino','huella_peticion_sha256','huella_anterior_sha256','huella_actuacion_sha256'];
 RETURN jsonb_build_object('Esquema','vec.contratacion-temporal.incorporacion.ejercicio.recibo.v2',
  'MaterialOriginalCanonico',replace(encode(mc,'base64'),chr(10),''),'MaterialOriginalSHA256',encode(sha256(mc),'hex'),
  'IntencionSHA256',vec_contratacion_temporal.intencion_registro_incorporacion_sha256_v2(m),'ContextoOriginalRef',m#>'{Vinculo,registro_contexto_ref}',
  'VersionSolicitudPersonal',m#>'{Confirmacion,SolicitudPersonal,version_expediente}','VersionActualExpediente',m->'VersionActualExpediente',
  'SeguimientoRef',seg,'DefinicionSeguimiento',a->'definicion','HuellaRaizSeguimiento',a->'huella_raiz_sha256',
  'HuellaEstadoAnterior',encode(sha256(ca),'hex'),'HuellaEstadoResultante',encode(sha256(cp),'hex'),
  'VersionSeguimientoAnterior',a->'version','VersionSeguimientoResultante',p->'version','Transicion',t,
  'AuditoriaCTRef',aud,'OutboxCTRef',ob,'ConsumosOriginales',consumos,'EjercicioSintetico',true,'FirmaOficial',false,'EficaciaAdministrativa',false);
END $$;

CREATE FUNCTION vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(
 material jsonb, seguimiento_ref text,
 ct_capacidad bytea,ct_decision bytea,ct_motivo bytea,ct_contexto bytea,ct_persona_version numeric,ct_perfil_version numeric,
 ct_payload bytea,ct_sobre bytea,ct_evidencia bytea,ct_raiz bytea,
 lector_capacidad bytea,lector_decision bytea,lector_motivo bytea,lector_contexto bytea,lector_persona_version bigint,lector_perfil_version bigint,
 lector_payload bytea,lector_sobre bytea,lector_evidencia bytea,lector_raiz bytea, evidencia_orden jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s'
AS $registrar$
DECLARE
 pc bytea[]:=ARRAY[ct_capacidad,ct_decision,ct_motivo,ct_contexto,ct_payload,ct_sobre,ct_evidencia,ct_raiz];
 pl bytea[]:=ARRAY[lector_capacidad,lector_decision,lector_motivo,lector_contexto,lector_payload,lector_sobre,lector_evidencia,lector_raiz];
 c jsonb; d jsonb; lc jsonb; ld jsonb; s jsonb; resultado_personal jsonb; preparacion jsonb; original_personal jsonb;
 vinculo_actual jsonb; clave_vinculo text;
 lectura jsonb; registro_personal jsonb; esperado_personal jsonb; consumos jsonb; consumos_originales jsonb; respuesta jsonb;
 recibo jsonb; pub jsonb; def jsonb; anterior jsonb; posterior jsonb; aplicado jsonb; evento jsonb; raiz_json jsonb;
 r vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 aud vec_contratacion_temporal.incorporacion_auditoria_v2%ROWTYPE;
 ob vec_contratacion_temporal.incorporacion_outbox_v2%ROWTYPE;
 raiz vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 definicion vec_contratacion_temporal.seguimiento_definicion_v2%ROWTYPE;
 estado_a vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 estado_p vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 consumo record; version_actual numeric; ultima_version numeric; esperada numeric;
 mc bytea; ic bytea; ca bytea; cp bytea; hmaterial text; hintencion text; hcontexto text; hexportacion text; horiginal text;
 inicio timestamptz; ahora timestamptz; fin timestamptz; leida timestamptz; registrada timestamptz; limite_ct timestamptz; limite_lector timestamptz;
 ref_recibo text; ref_actuacion text; ref_auditoria text; ref_outbox text;
 recuperado boolean; conflicto boolean:=false; mensaje text; codigo text;
BEGIN
 -- Runtime único CT, directo y no delegable; migrador/owner/roles mixtos fuera.
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR (SELECT count(*) FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member WHERE u.rolname=session_user)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member JOIN pg_roles e ON e.oid=m.roleid
  WHERE u.rolname=session_user AND e.rolname='vec_contratacion_temporal_ejecutor'
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
   AND NOT e.rolcanlogin AND e.rolinherit AND NOT e.rolsuper AND NOT e.rolcreatedb AND NOT e.rolcreaterole AND NOT e.rolreplication AND NOT e.rolbypassrls
   AND NOT EXISTS (SELECT 1 FROM pg_auth_members padre WHERE padre.member=e.oid))
 OR EXISTS (SELECT 1 FROM pg_roles WHERE left(rolname,4)='vec_' AND rolname NOT IN (session_user,'vec_contratacion_temporal_ejecutor') AND pg_has_role(session_user,oid,'MEMBER')) THEN
  RAISE EXCEPTION 'CT75: llamador denegado' USING ERRCODE='42501'; END IF;
 -- Compartidos de dependencia antes de cualquier lock de negocio/AD3.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
 inicio:=clock_timestamp();
 IF material IS NULL OR pg_column_size(material)>1048576 OR seguimiento_ref IS NULL
 OR seguimiento_ref !~ '^ref:[0-9a-f]{64}$' OR seguimiento_ref='ref:'||repeat('0',64) THEN
  RAISE EXCEPTION 'CT75: entrada inválida' USING ERRCODE='22023'; END IF;
 mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(material);
 ic:=vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(material);
 hmaterial:=encode(sha256(mc),'hex'); hintencion:=encode(sha256(ic),'hex');
 hcontexto:=vec_contratacion_temporal.contexto_incorporacion_ejercicio_sha256_v2(material);
 hexportacion:=vec_contratacion_temporal.incorporacion75_autoridad(material,pc,ct_persona_version,ct_perfil_version,inicio);
 PERFORM vec_contratacion_temporal.incorporacion75_evidencia(evidencia_orden,material,pc,ct_persona_version,ct_perfil_version,inicio);
 PERFORM vec_contratacion_temporal.incorporacion75_piezas(pl,lector_persona_version::numeric,lector_perfil_version::numeric);
 c:=convert_from(ct_capacidad,'UTF8')::jsonb; d:=convert_from(ct_decision,'UTF8')::jsonb;
 lc:=convert_from(lector_capacidad,'UTF8')::jsonb; ld:=convert_from(lector_decision,'UTF8')::jsonb;
 -- P1: dos permisos válidos de actores/contextos distintos NO forman un intento.
 -- Vinculo completo del MATERIAL ACTUAL, nunca el guardado en r durante replay.
 -- Wire de decisión V3 fija seis decimales (igual a incorporacion75_autoridad).
 vinculo_actual:=material->'Vinculo';
 FOREACH clave_vinculo IN ARRAY ARRAY['autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en'] LOOP
  vinculo_actual:=jsonb_set(vinculo_actual,ARRAY[clave_vinculo],to_jsonb(to_char((vinculo_actual->>clave_vinculo)::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 END LOOP;
 IF ld->'vinculo_autenticacion_actor' IS DISTINCT FROM vinculo_actual
 OR ld->>'principal_id' IS DISTINCT FROM vinculo_actual->>'principal_id'
 OR ld->>'perfil_activo_ref' IS DISTINCT FROM vinculo_actual->>'perfil_activo_ref'
 OR lector_contexto IS DISTINCT FROM ct_contexto
 OR lector_contexto IS DISTINCT FROM vec_contratacion_temporal.bytes_incorporacion_v2(material->'ContextoCanonico')
 OR lc->>'contexto_ref' IS DISTINCT FROM vinculo_actual->>'registro_contexto_ref'
 OR lc->>'huella_contexto_sha256' IS DISTINCT FROM vinculo_actual->>'contexto_actor_huella_sha256'
 OR lector_persona_version::numeric IS DISTINCT FROM ct_persona_version
 OR lector_perfil_version::numeric IS DISTINCT FROM ct_perfil_version THEN
  RAISE EXCEPTION 'CT75: lector ajeno al intento actual' USING ERRCODE='42501'; END IF;
 -- Audiencia, acción, motivo y recurso lectores siguen siendo propios de Personal;
 -- su verificación/consumo corresponde a la fachada, no se sustituyen por los CT.
 limite_ct:=vec_contratacion_temporal.incorporacion75_ventana(c,d,inicio);
 limite_lector:=vec_contratacion_temporal.incorporacion75_ventana(lc,ld,inicio);
 s:=material#>'{Confirmacion,SolicitudPersonal}'; resultado_personal:=material#>'{Confirmacion,ResultadoPersonal}';
 preparacion:=material->'Preparacion'; original_personal:=material->'Personal';
 esperada:=(material#>>'{Confirmacion,VersionSeguimientoEsperada}')::numeric;
 IF c->>'decision_ref' IS NOT DISTINCT FROM ld->>'decision_ref'
 OR c->>'decision_ref' IS NOT DISTINCT FROM original_personal->>'DecisionOriginalRef'
 OR ld->>'decision_ref' IS NOT DISTINCT FROM original_personal->>'DecisionOriginalRef' THEN
  RAISE EXCEPTION 'CT75: permisos no independientes' USING ERRCODE='42501'; END IF;
 -- No se consulta negocio antes de consumir CT y la autorización lectora propia.
 BEGIN
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(
   ct_capacidad,ct_decision,ct_motivo,ct_contexto,ct_persona_version,ct_perfil_version,ct_payload,ct_sobre,ct_evidencia,ct_raiz);
 EXCEPTION WHEN SQLSTATE 'P1102' THEN RAISE EXCEPTION 'CT75: permiso ya consumido' USING ERRCODE='42501'; END;
 ahora:=clock_timestamp();
 IF consumo.consumo_nuevo IS DISTINCT FROM true OR consumo.decision_ref IS DISTINCT FROM c->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM s->>'expediente_ref' OR consumo.huella_efecto_sha256 IS DISTINCT FROM hcontexto
 OR consumo.consumo_huella_sha256 IS NULL OR consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR consumo.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(consumo.consumo_huella_sha256,1,32)
 OR consumo.consumida_en IS NULL OR NOT isfinite(consumo.consumida_en) OR consumo.consumida_en<inicio OR consumo.consumida_en>ahora
 OR ahora<inicio OR ahora>=least(limite_ct,limite_lector) THEN RAISE EXCEPTION 'CT75: consumo CT inválido' USING ERRCODE='42501'; END IF;
 BEGIN
  lectura:=vec_personal.acreditar_alta_ejercicio_v1(preparacion->>'OrganizacionRef',s->>'solicitud_ref',s->>'expediente_ref',(s->>'version_expediente')::bigint,
   resultado_personal->>'resultado_ref',resultado_personal->>'recibo_ref',resultado_personal->>'relacion_ref',resultado_personal->>'ocupacion_ref',original_personal->>'MaterialSHA256',
   lector_capacidad,lector_decision,lector_motivo,lector_contexto,lector_persona_version,lector_perfil_version,lector_payload,lector_sobre,lector_evidencia,lector_raiz);
 EXCEPTION WHEN SQLSTATE 'P1102' THEN conflicto:=true; RAISE; END;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(lectura,ARRAY['registro','decision_lectura_ref','consumo_huella_sha256','auditoria_lectura_ref','leida_en']);
 registro_personal:=lectura->'registro';
 esperado_personal:=jsonb_build_object('solicitud',s,'resultado',resultado_personal,'material_canonico',original_personal->'MaterialCanonico',
  'material_sha256',original_personal->'MaterialSHA256','registrado_en',registro_personal->'registrado_en',
  'decision_original_ref',original_personal->'DecisionOriginalRef','auditoria_ref',original_personal->'AuditoriaRef','outbox_ref',original_personal->'OutboxRef',
  'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false);
 PERFORM vec_contratacion_temporal.seguimiento73_micro(registro_personal->'registrado_en',false);
 PERFORM vec_contratacion_temporal.seguimiento73_micro(lectura->'leida_en',false);
 leida:=(lectura->>'leida_en')::timestamptz;
 IF registro_personal IS DISTINCT FROM esperado_personal
 OR (registro_personal->>'registrado_en')::timestamptz IS DISTINCT FROM (original_personal->>'RegistradoEn')::timestamptz
 OR (registro_personal->>'registrado_en')::timestamptz>inicio
 OR lectura->>'decision_lectura_ref' IS DISTINCT FROM ld->>'decision_ref'
 OR lectura->>'consumo_huella_sha256' IS NULL OR lectura->>'consumo_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR lectura->>'consumo_huella_sha256'=consumo.consumo_huella_sha256
 OR lectura->>'auditoria_lectura_ref' IS NULL OR lectura->>'auditoria_lectura_ref' !~ '^auditoria:personal:lectura:[0-9a-f-]{36}$'
 OR lectura->>'auditoria_lectura_ref'=original_personal->>'AuditoriaRef'
 OR leida<consumo.consumida_en OR leida<ahora OR leida>=least(limite_ct,limite_lector) THEN
  RAISE EXCEPTION 'CT75: acreditación propietaria incoherente' USING ERRCODE='55000'; END IF;
 consumos:=jsonb_build_object('DecisionCTRef',consumo.decision_ref,'HuellaExportacionCT',hexportacion,
  'ConsumoCTSHA256',consumo.consumo_huella_sha256,'AuditoriaAD3CTRef',consumo.auditoria_ref,
  'ConsumidaCTEn',vec_contratacion_temporal.incorporacion75_instante(consumo.consumida_en),
  'DecisionLecturaRef',lectura->'decision_lectura_ref','ConsumoLecturaSHA256',lectura->'consumo_huella_sha256',
  -- AD3-10:514–515, referencia determinista del consumo REAL devuelto por Personal.
  'AuditoriaAD3LecturaRef','aud_v3_'||substr(lectura->>'consumo_huella_sha256',1,32),
  'AuditoriaPersonalLecturaRef',lectura->'auditoria_lectura_ref','LeidaPersonalEn',vec_contratacion_temporal.incorporacion75_instante(leida));
 -- Serialización estable por idempotencia; no reintento ante 40001/40P01.
 PERFORM pg_advisory_xact_lock(hashtextextended('CT75:'||(preparacion->>'OrganizacionRef')||':'||(s->>'idempotencia_ref'),0));
 SELECT a.version INTO version_actual FROM vec_contratacion_temporal.expediente_integral_actual a WHERE a.expediente_ref=s->>'expediente_ref' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT75: expediente no disponible' USING ERRCODE='55000'; END IF;
 SELECT z.* INTO raiz FROM vec_contratacion_temporal.seguimiento_raiz_v2 z WHERE z.seguimiento_ref=registrar_incorporacion_ejercicio_v2.seguimiento_ref FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT75: raíz gobernada no disponible' USING ERRCODE='55000'; END IF;
 IF raiz.organizacion_ref IS DISTINCT FROM preparacion->>'OrganizacionRef' OR raiz.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
 OR raiz.relacion_ref IS DISTINCT FROM resultado_personal->>'relacion_ref' OR raiz.version_expediente_observada>version_actual THEN
  conflicto:=true; RAISE EXCEPTION 'CT75: raíz incompatible' USING ERRCODE='P1102'; END IF;
 SELECT f.* INTO STRICT definicion FROM vec_contratacion_temporal.seguimiento_definicion_v2 f
  WHERE f.definicion_ref=raiz.definicion_ref AND f.definicion_version=raiz.definicion_version AND f.definicion_sha256=raiz.definicion_sha256 FOR SHARE;
 pub:=definicion.publicacion_json; def:=vec_contratacion_temporal.seguimiento73_definicion(pub);
 IF definicion.definicion_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(def,'publicacion') THEN
  RAISE EXCEPTION 'CT75: publicación no reproducible' USING ERRCODE='55000'; END IF;
 SELECT v.* INTO r FROM vec_contratacion_temporal.incorporacion_registro_v2 v
  WHERE v.organizacion_ref=preparacion->>'OrganizacionRef' AND v.idempotencia_ref=s->>'idempotencia_ref' FOR SHARE;
 recuperado:=FOUND;
 SELECT max(e.version_seguimiento) INTO ultima_version FROM vec_contratacion_temporal.seguimiento_estado_v2 e WHERE e.seguimiento_ref=raiz.seguimiento_ref;
 IF recuperado THEN
  IF r.intencion_canonica IS DISTINCT FROM ic OR r.intencion_sha256 IS DISTINCT FROM hintencion OR r.seguimiento_ref IS DISTINCT FROM raiz.seguimiento_ref
  OR r.solicitud_ref IS DISTINCT FROM s->>'solicitud_ref' OR r.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
  OR r.version_expediente IS DISTINCT FROM (material->>'VersionActualExpediente')::numeric OR version_actual<r.version_expediente THEN
   conflicto:=true; RAISE EXCEPTION 'CT75: intención en conflicto' USING ERRCODE='P1102'; END IF;
  SELECT e.* INTO STRICT estado_a FROM vec_contratacion_temporal.seguimiento_estado_v2 e WHERE e.seguimiento_ref=raiz.seguimiento_ref AND e.version_seguimiento=r.version_anterior FOR SHARE;
  SELECT e.* INTO STRICT estado_p FROM vec_contratacion_temporal.seguimiento_estado_v2 e WHERE e.seguimiento_ref=raiz.seguimiento_ref AND e.version_seguimiento=r.version_resultante FOR SHARE;
 ELSE
  IF ultima_version IS NULL THEN RAISE EXCEPTION 'CT75: estado gobernado no disponible' USING ERRCODE='55000'; END IF;
  IF version_actual IS DISTINCT FROM (material->>'VersionActualExpediente')::numeric OR ultima_version<>esperada THEN
   conflicto:=true; RAISE EXCEPTION 'CT75: versión en conflicto' USING ERRCODE='P1102'; END IF;
  SELECT e.* INTO STRICT estado_a FROM vec_contratacion_temporal.seguimiento_estado_v2 e WHERE e.seguimiento_ref=raiz.seguimiento_ref AND e.version_seguimiento=esperada FOR SHARE;
 END IF;
 anterior:=estado_a.estado_json; ca:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,anterior);
 raiz_json:=jsonb_build_object('referencia',raiz.seguimiento_ref,'organizacion_ref',raiz.organizacion_ref,'expediente_ref',raiz.expediente_ref,'relacion_ref',raiz.relacion_ref,
  'definicion',anterior->'definicion','estado_actual',def->'estado_inicial','periodo_previsto',anterior->'periodo_previsto','creado_en',anterior->'creado_en');
 IF ca IS DISTINCT FROM estado_a.estado_canonico OR encode(sha256(ca),'hex') IS DISTINCT FROM estado_a.estado_sha256
 OR raiz.raiz_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(raiz_json,'raiz')
 OR raiz.raiz_sha256 IS DISTINCT FROM anterior->>'huella_raiz_sha256' THEN
  RAISE EXCEPTION 'CT75: historia anterior no reproducible' USING ERRCODE='55000'; END IF;
 ahora:=clock_timestamp();
 IF ahora<leida OR ahora>=least(limite_ct,limite_lector) THEN RAISE EXCEPTION 'CT75: autorización agotada' USING ERRCODE='42501'; END IF;
 IF recuperado THEN
  consumos_originales:=r.recibo_json->'ConsumosOriginales';
  PERFORM vec_contratacion_temporal.incorporacion75_evidencia(r.evidencia_orden_json,r.material_json,r.exportacion_ct,r.persona_version,r.perfil_version,r.registrada_en);
  IF (r.evidencia_orden_json->>'EvaluadaEn')::timestamptz>(consumos_originales->>'ConsumidaCTEn')::timestamptz THEN
   RAISE EXCEPTION 'CT75: evaluación histórica posterior al consumo' USING ERRCODE='55000'; END IF;
  horiginal:=vec_contratacion_temporal.incorporacion75_autoridad(r.material_json,r.exportacion_ct,r.persona_version,r.perfil_version,(consumos_originales->>'ConsumidaCTEn')::timestamptz);
  PERFORM vec_contratacion_temporal.incorporacion75_autoridad(r.material_json,r.exportacion_ct,r.persona_version,r.perfil_version,r.registrada_en);
  IF r.material_canonico IS DISTINCT FROM vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(r.material_json)
  OR r.intencion_canonica IS DISTINCT FROM vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(r.material_json)
  OR horiginal IS DISTINCT FROM consumos_originales->>'HuellaExportacionCT'
  OR consumos_originales->>'DecisionCTRef' IS DISTINCT FROM (convert_from(r.exportacion_ct[1],'UTF8')::jsonb)->>'decision_ref'
  OR consumos_originales->>'AuditoriaAD3CTRef' IS DISTINCT FROM 'aud_v3_'||substr(consumos_originales->>'ConsumoCTSHA256',1,32)
  OR consumos_originales->>'AuditoriaAD3LecturaRef' IS DISTINCT FROM 'aud_v3_'||substr(consumos_originales->>'ConsumoLecturaSHA256',1,32)
  OR (consumos_originales->>'LeidaPersonalEn')::timestamptz<(consumos_originales->>'ConsumidaCTEn')::timestamptz
  OR r.registrada_en<(consumos_originales->>'LeidaPersonalEn')::timestamptz OR r.registrada_en>inicio
  OR (c->>'emitida_en')::timestamptz<r.registrada_en THEN RAISE EXCEPTION 'CT75: historia nominal inválida' USING ERRCODE='55000'; END IF;
  FOREACH mensaje IN ARRAY ARRAY['DecisionCTRef','ConsumoCTSHA256','AuditoriaAD3CTRef','DecisionLecturaRef','ConsumoLecturaSHA256','AuditoriaAD3LecturaRef','AuditoriaPersonalLecturaRef'] LOOP
   IF consumos->>mensaje IS NOT DISTINCT FROM consumos_originales->>mensaje THEN
    RAISE EXCEPTION 'CT75: recuperación requiere permisos nuevos' USING ERRCODE='42501'; END IF;
  END LOOP;
  aplicado:=vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(pub,anterior,r.material_json,r.recibo_json#>>'{Transicion,actuacion_ref}',r.recibo_ref,vec_contratacion_temporal.incorporacion75_instante(r.registrada_en));
  posterior:=estado_p.estado_json; cp:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,posterior);
  evento:=aplicado->'evento';
  SELECT a.* INTO STRICT aud FROM vec_contratacion_temporal.incorporacion_auditoria_v2 a WHERE a.auditoria_ref=r.auditoria_ref FOR SHARE;
  SELECT o.* INTO STRICT ob FROM vec_contratacion_temporal.incorporacion_outbox_v2 o WHERE o.outbox_ref=r.outbox_ref FOR SHARE;
  recibo:=vec_contratacion_temporal.incorporacion75_recibo(r.material_json,raiz.seguimiento_ref,pub,anterior,posterior,evento,consumos_originales,r.auditoria_ref,r.outbox_ref);
  IF cp IS DISTINCT FROM estado_p.estado_canonico OR encode(sha256(cp),'hex') IS DISTINCT FROM estado_p.estado_sha256
  OR cp IS DISTINCT FROM decode(aplicado->>'estado_canonico_base64','base64') OR r.recibo_json IS DISTINCT FROM recibo
  OR r.estado_anterior_sha256 IS DISTINCT FROM estado_a.estado_sha256 OR r.estado_resultante_sha256 IS DISTINCT FROM estado_p.estado_sha256
  OR aud.recibo_ref<>r.recibo_ref OR aud.recuperado OR aud.consumos_json IS DISTINCT FROM consumos_originales OR aud.registrada_en<>r.registrada_en
  OR ob.recibo_ref<>r.recibo_ref OR ob.evento_json IS DISTINCT FROM evento OR ob.estado_sha256<>estado_p.estado_sha256 OR ob.creada_en<>r.registrada_en THEN
   RAISE EXCEPTION 'CT75: historia durable incoherente' USING ERRCODE='55000'; END IF;
  ref_recibo:=r.recibo_ref;
 ELSE
  registrada:=ahora;
  ref_recibo:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
  ref_actuacion:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
  ref_auditoria:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
  ref_outbox:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
  aplicado:=vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(pub,anterior,material,ref_actuacion,ref_recibo,vec_contratacion_temporal.incorporacion75_instante(registrada));
  posterior:=aplicado->'estado'; evento:=aplicado->'evento'; cp:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,posterior);
  IF (posterior->>'version')::numeric IS DISTINCT FROM esperada+1 OR cp IS DISTINCT FROM decode(aplicado->>'estado_canonico_base64','base64')
  OR encode(sha256(cp),'hex') IS DISTINCT FROM aplicado->>'estado_sha256' THEN
   RAISE EXCEPTION 'CT75: transición no reproducible' USING ERRCODE='55000'; END IF;
  recibo:=vec_contratacion_temporal.incorporacion75_recibo(material,raiz.seguimiento_ref,pub,anterior,posterior,evento,consumos,ref_auditoria,ref_outbox);
  INSERT INTO vec_contratacion_temporal.seguimiento_estado_v2 VALUES(raiz.seguimiento_ref,raiz.organizacion_ref,raiz.expediente_ref,raiz.relacion_ref,
   raiz.definicion_ref,raiz.definicion_version,raiz.definicion_sha256,raiz.raiz_sha256,esperada+1,esperada,estado_a.estado_sha256,posterior,cp,encode(sha256(cp),'hex'));
  INSERT INTO vec_contratacion_temporal.incorporacion_registro_v2 VALUES(ref_recibo,raiz.seguimiento_ref,raiz.organizacion_ref,s->>'idempotencia_ref',s->>'solicitud_ref',
   raiz.expediente_ref,version_actual,esperada,esperada+1,estado_a.estado_sha256,encode(sha256(cp),'hex'),material,mc,hmaterial,ic,hintencion,pc,ct_persona_version,ct_perfil_version,recibo,ref_auditoria,ref_outbox,registrada,evidencia_orden)
   RETURNING * INTO r;
  INSERT INTO vec_contratacion_temporal.incorporacion_outbox_v2 VALUES(ref_outbox,ref_recibo,evento,encode(sha256(cp),'hex'),registrada);
 END IF;
 IF recuperado THEN ref_auditoria:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'); registrada:=ahora; END IF;
 INSERT INTO vec_contratacion_temporal.incorporacion_auditoria_v2 VALUES(ref_auditoria,ref_recibo,recuperado,
  consumos->>'DecisionCTRef',consumos->>'ConsumoCTSHA256',consumos->>'DecisionLecturaRef',consumos->>'ConsumoLecturaSHA256',consumos,registrada);
 respuesta:=jsonb_build_object('recibo',r.recibo_json,'historia',jsonb_build_object('Esquema','vec.contratacion-temporal.incorporacion.ejercicio.historia-original.v2',
  'Evidencia',r.evidencia_orden_json,'Recibo',r.recibo_json,'Publicacion',pub,'Anterior',anterior,'Posterior',posterior),
  'recuperado',recuperado,'consumos_actuales',consumos);
 ahora:=clock_timestamp();
 IF ahora<registrada OR ahora<leida OR ahora>=least(limite_ct,limite_lector) OR pg_column_size(respuesta)>33554432 THEN
  RAISE EXCEPTION 'CT75: retorno no disponible' USING ERRCODE='55000'; END IF;
 PERFORM vec_contratacion_temporal.incorporacion75_autoridad(material,pc,ct_persona_version,ct_perfil_version,ahora);
 PERFORM vec_contratacion_temporal.incorporacion75_ventana(lc,ld,ahora);
 fin:=clock_timestamp();
 IF fin<ahora OR fin>=least(limite_ct,limite_lector) THEN RAISE EXCEPTION 'CT75: autorización agotada al retornar' USING ERRCODE='42501'; END IF;
 -- COMMIT pertenece a la TX exterior; no exponer antes de confirmarlo.
 RETURN respuesta;
EXCEPTION WHEN OTHERS THEN
 codigo:=SQLSTATE;
 IF codigo='P1102' AND NOT conflicto THEN codigo:='55000'; END IF;
 IF codigo='23505' THEN codigo:='40001'; END IF;
 IF codigo NOT IN ('P1102','42501','22023','40001','40P01','55P03','55000') THEN codigo:='55000'; END IF;
 RAISE EXCEPTION 'CT75: registro de incorporación no disponible' USING ERRCODE=codigo;
END $registrar$;

DO $acl$
DECLARE f record; a record;
BEGIN
 FOR f IN SELECT p.oid,p.proowner,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE 'incorporacion75_%' OR p.proname='registrar_incorporacion_ejercicio_v2') LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
  FOR a IN SELECT DISTINCT grantee FROM aclexplode((SELECT proacl FROM pg_proc WHERE oid=f.oid)) WHERE grantee<>f.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.oid::regprocedure,pg_get_userbyid(a.grantee));
  END LOOP;
  IF f.proname='registrar_incorporacion_ejercicio_v2' THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f.oid::regprocedure);
  END IF;
  EXECUTE format('COMMENT ON FUNCTION %s IS %L',f.oid::regprocedure,'CT75:registro-incorporacion-v2');
 END LOOP;
END $acl$;
COMMIT;
