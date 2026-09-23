\set ON_ERROR_STOP on
-- D2/D3/D4: instantánea orientativa de OSRM y tres grupos provisionales.
-- No atribuye grupo al empleado, ni constituye liquidación o envío.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000004:calculo-comision:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.borrador_comision') IS NULL
    OR to_regclass('vec_dietas.calculo_comision') IS NOT NULL
    OR to_regclass('vec_dietas.version_tarifa_provisional') IS NULL
    OR to_regprocedure('vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000004: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_dietas.calculo_comision (
 comision_ref text PRIMARY KEY REFERENCES vec_dietas.borrador_comision(referencia),
 calculo jsonb NOT NULL CHECK(jsonb_typeof(calculo)='object'),
 version_tarifa text NOT NULL REFERENCES vec_dietas.version_tarifa_provisional(version_ref),
 registrado_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_dietas.calculo_comision ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.calculo_comision FORCE ROW LEVEL SECURITY;
CREATE POLICY persona_contextual ON vec_dietas.calculo_comision FOR ALL TO vec_dietas_propietario
 USING (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=comision_ref AND b.persona_ref=current_setting('vec.dietas.persona_ref',true)))
 WITH CHECK (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=comision_ref AND b.persona_ref=current_setting('vec.dietas.persona_ref',true)));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_dietas.calculo_comision FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.calculo_comision FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.calculo_comision FROM PUBLIC,vec_dietas_ejecutor;

-- Preconsulta con la misma concesión nominal de crear. Consume AD3 y
-- revalida Personal antes de cualquier llamada a OSRM; no crea borrador.
CREATE FUNCTION vec_dietas.recuperar_comision_por_clave_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; c jsonb; cap jsonb; ctx jsonb; d jsonb; v record; b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.recibo_borrador_comision%ROWTYPE; calc jsonb; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando'; cap:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1' OR m->>'operacion'<>'crear' OR m->>'recurso_ref'<>'dietas:borradores:propios'
    OR c ? 'calculo' OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR c->>'hora_inicio' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$' OR c->>'hora_fin' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR jsonb_typeof(c->'codigos_ruta')<>'array' OR jsonb_array_length(c->'codigos_ruta') NOT BETWEEN 2 AND 12
    OR d->'campos_permitidos' IS DISTINCT FROM '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb
    OR m->>'huella_semantica' IS DISTINCT FROM vec_dietas.huella_semantica_crear_borrador_v1(p_material)
    OR vec_dietas.cotejar_recurso_dietas_borrador_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
 THEN RAISE EXCEPTION 'preconsulta Dietas inválida' USING ERRCODE='PD003'; END IF;
 IF p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric THEN RAISE EXCEPTION 'versiones ContextoActor Dietas incoherentes' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_dietas.consumir_ad3_borrador_v1(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref' OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR cap->>'huella_efecto_sha256' IS DISTINCT FROM v.huella_efecto_sha256 OR NOT vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto)
 THEN RAISE EXCEPTION 'AD3 no ligado a preconsulta Dietas' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);
 PERFORM vec_personal.revalidar_relacion_dietas_v1(i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),(i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',(i->>'fuente_version')::bigint,(i->>'fecha_referencia')::date);
 SELECT * INTO b FROM vec_dietas.borrador_comision WHERE persona_ref=i->>'persona_ref' AND clave_idempotencia=c->>'clave_idempotencia';
 IF NOT FOUND THEN
  INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||'preconsulta'||ahora::text),NULL,'dietas:borradores:propios','consultar',d->>'principal_id',i->>'persona_ref','no_encontrado',d->>'correlacion_ref',ahora);
  RETURN jsonb_build_object('encontrado',false);
 END IF;
 IF b.huella_semantica_sha256 IS DISTINCT FROM m->>'huella_semantica' THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
 SELECT * INTO STRICT r FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=b.referencia;
 SELECT calculo INTO STRICT calc FROM vec_dietas.calculo_comision WHERE comision_ref=b.referencia;
 INSERT INTO vec_dietas.auditoria_borrador_comision VALUES('adi_'||md5(v.auditoria_ref||b.referencia||ahora::text),b.referencia,b.referencia,'consultar',d->>'principal_id',i->>'persona_ref','concedido',d->>'correlacion_ref',ahora);
 RETURN jsonb_build_object('encontrado',true,'comision',jsonb_build_object('referencia',b.referencia,'estado','borrador','fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref,'calculo',calc),'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'repeticion',true));
END $$;

CREATE FUNCTION vec_dietas.crear_o_recuperar_comision_calculada_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; c jsonb; calc jsonb; tr jsonb; op jsonb; dato jsonb; salida jsonb; anterior jsonb; decision jsonb; ref text; n int; grupo_actual int; total numeric(12,4):=0; tarifa numeric(8,4); man numeric(9,2); alo numeric(9,2); suma_man bigint; suma_alo bigint; importe bigint;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; c:=m->'comando'; calc:=c->'calculo'; decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'cálculo Dietas inválido' USING ERRCODE='22023'; END;
 IF decision->'campos_permitidos' IS DISTINCT FROM '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb THEN RAISE EXCEPTION 'proyección de cálculo no autorizada' USING ERRCODE='PD003'; END IF;
 IF m->>'operacion'<>'crear' OR calc IS NULL OR jsonb_typeof(calc)<>'object'
    OR calc->>'procedencia'<>'osrm_interno' OR calc->>'motor'<>'OSRM'
    OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160
    OR calc->>'rotulo'<>'PROVISIONAL · pendiente de confirmación por RRHH'
    OR calc->>'version_tarifa' !~ '^provisional:[a-z0-9:-]{8,120}$'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$'
    OR calc->>'hora_inicio' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR calc->>'hora_fin' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR c->>'hora_inicio' IS DISTINCT FROM calc->>'hora_inicio' OR c->>'hora_fin' IS DISTINCT FROM calc->>'hora_fin'
    OR jsonb_typeof(c->'codigos_ruta')<>'array' OR jsonb_typeof(calc->'tramos_ruta')<>'array'
    OR jsonb_typeof(calc->'opciones_dieta')<>'array'
    OR jsonb_array_length(c->'codigos_ruta') NOT BETWEEN 2 AND 12
    OR jsonb_array_length(calc->'tramos_ruta')<>jsonb_array_length(c->'codigos_ruta')-1
    OR jsonb_array_length(calc->'opciones_dieta')<>3
 THEN RAISE EXCEPTION 'cálculo Dietas inválido' USING ERRCODE='22023'; END IF;
 SELECT k.eur_por_km INTO STRICT tarifa FROM vec_dietas.importe_km_provisional k
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=k.version_ref
 WHERE k.version_ref=calc->>'version_tarifa' AND k.vehiculo='automovil'
   AND v.vigente_desde<=(c->>'fecha_inicio')::date AND (v.vigente_hasta IS NULL OR (c->>'fecha_fin')::date<v.vigente_hasta);
 IF tarifa IS DISTINCT FROM (calc->>'eur_por_km')::numeric THEN RAISE EXCEPTION 'tarifa Dietas incompatible' USING ERRCODE='22023'; END IF;
 FOR n IN 0..jsonb_array_length(calc->'tramos_ruta')-1 LOOP
  tr:=calc->'tramos_ruta'->n;
  IF tr->>'origen_codigo' IS DISTINCT FROM c->'codigos_ruta'->>n
     OR tr->>'destino_codigo' IS DISTINCT FROM c->'codigos_ruta'->>(n+1)
     OR tr->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
     OR (tr->>'kilometros')::numeric<=0 THEN RAISE EXCEPTION 'tramo OSRM inválido' USING ERRCODE='22023'; END IF;
  total:=total+(tr->>'kilometros')::numeric;
 END LOOP;
 IF total IS DISTINCT FROM (calc->>'kilometros')::numeric OR total>10000
    OR (calc->>'importe_kilometraje_centimos')::bigint IS DISTINCT FROM round(total*tarifa*100)::bigint
 THEN RAISE EXCEPTION 'importe ruta incompatible' USING ERRCODE='22023'; END IF;
 FOR grupo_actual IN 1..3 LOOP
  op:=calc->'opciones_dieta'->(grupo_actual-1);
  SELECT d.manutencion_eur,d.alojamiento_eur INTO STRICT man,alo FROM vec_dietas.importe_dieta_provisional d
  WHERE version_ref=calc->>'version_tarifa' AND pais_iso2='ES' AND d.grupo=grupo_actual;
  IF (op->>'grupo')::int IS DISTINCT FROM grupo_actual OR op->'calculo'->>'version_tarifa_ref' IS DISTINCT FROM calc->>'version_tarifa'
     OR op->'calculo'->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
     OR jsonb_typeof(op->'calculo'->'tramos')<>'array' OR jsonb_array_length(op->'calculo'->'tramos')>62
  THEN RAISE EXCEPTION 'grupo Dietas incompatible' USING ERRCODE='22023'; END IF;
  suma_man:=0; suma_alo:=0;
  FOR dato IN SELECT value FROM jsonb_array_elements(op->'calculo'->'tramos') LOOP
   IF dato->>'version_tarifa_ref' IS DISTINCT FROM calc->>'version_tarifa' OR dato->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
      OR dato->>'fecha' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$' THEN RAISE EXCEPTION 'tramo Dietas incompatible' USING ERRCODE='22023'; END IF;
   importe:=(dato->>'importe_centimos')::bigint;
   IF dato->>'tipo'='manutencion' AND (dato->>'porcentaje')::int IN (50,100)
      AND importe=(CASE WHEN (dato->>'porcentaje')::int=50 THEN ceil(man*100/2)::bigint ELSE (man*100)::bigint END) THEN suma_man:=suma_man+importe;
   ELSIF dato->>'tipo'='alojamiento_tope_pendiente_justificante' AND (dato->>'porcentaje')::int=100 AND importe=(alo*100)::bigint THEN suma_alo:=suma_alo+importe;
   ELSE RAISE EXCEPTION 'importe tramo incompatible' USING ERRCODE='22023'; END IF;
  END LOOP;
  IF suma_man IS DISTINCT FROM (op->'calculo'->>'manutencion_centimos')::bigint
     OR suma_alo IS DISTINCT FROM (op->'calculo'->>'alojamiento_tope_centimos')::bigint
     OR suma_man+suma_alo IS DISTINCT FROM (op->'calculo'->>'total_maximo_orientativo_centimos')::bigint
  THEN RAISE EXCEPTION 'desglose tramo incompatible' USING ERRCODE='22023'; END IF;
 END LOOP;
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ref:=salida->'comision'->>'referencia';
 IF ref !~ '^dco_[A-Za-z0-9_-]{22,128}$' THEN RAISE EXCEPTION 'recibo Dietas incompatible' USING ERRCODE='PD003'; END IF;
 IF (salida->'recibo'->>'repeticion')::boolean IS FALSE THEN
  INSERT INTO vec_dietas.calculo_comision VALUES(ref,calc,calc->>'version_tarifa',clock_timestamp());
 END IF;
 SELECT calculo INTO STRICT anterior FROM vec_dietas.calculo_comision WHERE comision_ref=ref;
 -- La huella semántica ya ligó fechas, horas, motivo y ruta. En replay se
 -- conserva la instantánea original aunque OSRM haya publicado otro grafo.
 IF (salida->'recibo'->>'repeticion')::boolean IS FALSE AND anterior IS DISTINCT FROM calc THEN RAISE EXCEPTION 'cálculo Dietas incompatible' USING ERRCODE='PD003'; END IF;
 RETURN jsonb_set(salida,'{comision,calculo}',anterior,true);
END $$;

CREATE FUNCTION vec_dietas.consultar_comisiones_calculadas_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE salida jsonb; item jsonb; items jsonb:='[]'::jsonb; calc jsonb; decision jsonb; material jsonb; ref text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 BEGIN material:=p_material::jsonb; decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'consulta Dietas inválida' USING ERRCODE='22023'; END;
 IF (material->>'operacion'='detalle' AND decision->'campos_permitidos' IS DISTINCT FROM '["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.registrado_en","recibo.referencia","recibo.repeticion","recibo.version"]'::jsonb)
    OR (material->>'operacion'='lista' AND decision->'campos_permitidos' IS DISTINCT FROM '["items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.registrado_en","items.recibo.referencia","items.recibo.repeticion","items.recibo.version","siguiente_cursor"]'::jsonb)
    OR material->>'operacion' NOT IN ('detalle','lista') THEN RAISE EXCEPTION 'proyección de cálculo no autorizada' USING ERRCODE='PD003'; END IF;
 salida:=vec_dietas.consultar_borradores_propios_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF salida->>'resultado'='no_encontrado' THEN RETURN salida; END IF;
 IF jsonb_typeof(salida->'items')='array' THEN
  FOR item IN SELECT value FROM jsonb_array_elements(salida->'items') LOOP
   ref:=item->'comision'->>'referencia';
   SELECT calculo INTO STRICT calc FROM vec_dietas.calculo_comision WHERE comision_ref=ref;
   items:=items||jsonb_build_array(jsonb_set(item,'{comision,calculo}',calc,true));
  END LOOP;
  RETURN jsonb_set(salida,'{items}',items,true);
 END IF;
 ref:=salida->'comision'->>'referencia';
 SELECT calculo INTO STRICT calc FROM vec_dietas.calculo_comision WHERE comision_ref=ref;
 RETURN jsonb_set(salida,'{comision,calculo}',calc,true);
END $$;
ALTER FUNCTION vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
ALTER FUNCTION vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_dietas_ejecutor;
REVOKE ALL ON FUNCTION vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_ejecutor;
COMMIT;
