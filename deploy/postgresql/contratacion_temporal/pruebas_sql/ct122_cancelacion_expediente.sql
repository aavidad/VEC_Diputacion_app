\set ON_ERROR_STOP on
-- CT122 sobre la estructura real restaurada (volcado de la principal) con
-- AD3-87 y CT122 instaladas. Base desechable: los datos se confirman.
-- La fachada AD3-87 se sustituye por un doble explícito que devuelve el
-- consumo ligado a la decisión: prueba la transacción CT (versión, actuación,
-- outbox, publicación al cuadro, recibo, idempotencia, RLS y negativos), no
-- la criptografía V3.
-- Requiere antes ct122_fixture_pg18.sql en la misma sesión y las variables
-- exp_asignacion (v3), exp_solicitud (v1) y exp_fiscalizado (v7).

-- Construye la entrada de confirmación igual que el adaptador Go.
CREATE FUNCTION pg_temp.entrada(p_exp text, p_version numeric, p_canal text, p_motivo text, p_fases jsonb, p_obs text, p_sufijo text,
    p_accion text DEFAULT 'contratacion_temporal.expediente.cancelar')
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; inst text; m jsonb; act jsonb; amb jsonb; atr jsonb; h text; dec jsonb; amb_hmac text; hue_hmac text; org text; fase text;
BEGIN
 SELECT v.agregado_json INTO STRICT ag FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=p_exp AND v.version=p_version;
 -- La versión 1 guarda la forma del alta; Go la recibe normalizada.
 IF p_version=1 THEN ag:=vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(ag); END IF;
 org:=ag->>'organizacion_ref'; fase:=ag->>'fase_actual';
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.cancelacion.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.cancelacion.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo||p_motivo,'UTF8')),'hex');
 m:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'version_esperada',p_version,'actor_ref','per_ct122_actor','perfil_ref','prf_ct122',
   'canal',p_canal,'motivo_clave',p_motivo,'fases_admitidas',p_fases,'observaciones',p_obs);
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,'accion_clave','contratacion_temporal.expediente.cancelar',
   'actor_ref','per_ct122_actor','unidad_ref',ag->'actuaciones'->-1->>'unidad_ref','recibo_ref','recibo:ct122:'||p_sufijo,'realizada_en',inst,
   'fase_origen',fase,'fase_destino',fase,'estado_origen','en_curso','estado_destino','cancelado');
 IF p_obs<>'' THEN act:=act||jsonb_build_object('observaciones',p_obs); END IF;
 amb:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'fase_previa',fase,'estado_previo','en_curso');
 IF p_canal='centro' THEN amb:=amb||jsonb_build_object('centro_ref',ag#>>'{solicitud,centro_ref}'); END IF;
 atr:=jsonb_build_object('version_expediente',p_version::text,'canal',p_canal,'motivo_clave',p_motivo,
   'fases_admitidas',(SELECT string_agg(e #>> '{}',',' ORDER BY o) FROM jsonb_array_elements(p_fases) WITH ORDINALITY x(e,o)),
   'observaciones_huella_sha256',encode(sha256(convert_to(p_obs,'UTF8')),'hex'),
   'politica_ref','motivos_cancelacion_contratacion_temporal','politica_version','1','politica_huella_sha256',repeat('c',64),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac);
 h:=vec_contratacion_temporal.huella_contexto_go_ct122(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct122:'||p_sufijo,'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso','cancelacion_contratacion_temporal',
   'finalidad','cancelar_expediente_contratacion_temporal','recurso_ref',p_exp,'principal_id','per_ct122_actor','perfil_activo_ref','prf_ct122','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-cancelacion.v1','operacion','cancelar_expediente','material',m,
   'referencias',jsonb_build_object('reserva_ref','reserva:ct122:'||p_sufijo,'recibo_ref','recibo:ct122:'||p_sufijo,'evento_ref','evento:ct122:'||p_sufijo),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,'expediente_anterior',ag,
   'expediente_siguiente',ag||jsonb_build_object('version',p_version+1,'actualizado_en',inst,'estado_actual','cancelado','actuaciones',(ag->'actuaciones')||jsonb_build_array(act)),
   'actuacion',act,'politica',jsonb_build_object('definicion_ref','motivos_cancelacion_contratacion_temporal','definicion_version',1,'definicion_huella_sha256',repeat('c',64),
     'accion','contratacion_temporal.expediente.cancelar','finalidad','cancelar_expediente_contratacion_temporal','evaluada_en',inst,
     'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   'autorizacion',jsonb_build_object('accion',p_accion,'finalidad','cancelar_expediente_contratacion_temporal','recurso_ref',p_exp,'principal_id','per_ct122_actor',
     'perfil_activo_ref','prf_ct122','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),
     'persona_version',1,'perfil_version',1,'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),'decision_ref','decision:ct122:'||p_sufijo,
     'contexto_recurso_huella_sha256',h),
   'instante_efecto',inst,'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),
   '_decision',dec);
END $$;

-- Utilidades de aserción: las variables de psql no se sustituyen dentro de
-- cuerpos entre dólares, así que las comprobaciones reciben los valores.
CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;
CREATE FUNCTION pg_temp.confirmar(e jsonb, p_decision_extra text DEFAULT '') RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(e-'_decision','\x6361706163696461640a',convert_to((e->'_decision')::text||p_decision_extra,'UTF8'),
   '\x6d6f7469766f','\x01',1,1,'\x01','\x01','\x01','\x01');
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
CREATE FUNCTION pg_temp.preparar(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.preparar_cancelacion_expediente_v1(jsonb_build_object('esquema','vec.contratacion-temporal.preparar-cancelacion.v1',
  'operacion','cancelar_expediente','material',e->'material','sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',e->>'ambito_idempotencia_hmac',
  'generacion',1,'huella_peticion_hmac',e->>'huella_peticion_hmac'),'retenidos','[]'::jsonb),'referencias_candidatas',e->'referencias'));
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
CREATE FUNCTION pg_temp.consultar(p_org text, p_exp text) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.consultar_cancelacion_expediente_v1(p_org,p_exp);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct122_runtime',pg_my_temp_schema()::regnamespace); END $$;

\set fases '["solicitud","asignacion_unidad","informe_juridico"]'
SELECT pg_temp.entrada(:'exp_asignacion',3,'rrhh','necesidad_desaparecida',:'fases'::jsonb,'El centro ya no necesita el refuerzo','uno')::text AS entrada_uno \gset
SELECT pg_temp.entrada(:'exp_solicitud',1,'centro','desistimiento_centro',:'fases'::jsonb,'','centro')::text AS entrada_centro \gset
SELECT pg_temp.entrada(:'exp_solicitud',1,'rrhh','error_solicitud','["informe_juridico"]'::jsonb,'','fase')::text AS entrada_fase \gset
SELECT pg_temp.entrada(:'exp_fiscalizado',7,'rrhh','error_solicitud','["fiscalizacion","nombramiento"]'::jsonb,'','fisc')::text AS entrada_fisc \gset
SELECT pg_temp.entrada(:'exp_asignacion',3,'rrhh','necesidad_desaparecida',:'fases'::jsonb,'','accion','contratacion_temporal.expediente.cerrar')::text AS entrada_accion \gset
SELECT agregado_json->>'organizacion_ref' AS org FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_asignacion' AND version=3 \gset

-- Vector de compatibilidad con Go (mismo valor que CT115 y que el dominio VEC).
SELECT pg_temp.exigir(vec_contratacion_temporal.huella_contexto_go_ct122('{"b":"2","a":"x:1"}','{"k":"hmac-sha256:vec.x/v1:ab","c":"solicitud,asignacion_unidad"}')
    =encode(sha256(convert_to('{"ambitos":{"a":"x:1","b":"2"},"atributos":{"c":"solicitud,asignacion_unidad","k":"hmac-sha256:vec.x/v1:ab"}}','UTF8')),'hex'),'vector de contexto');

-- Fuera de una sesión del ejecutor de CT la fachada se niega.
SELECT pg_temp.exigir((pg_temp.preparar(:'entrada_uno'::jsonb))->>'error'='42501','superusuario sin sesión de ejecutor');

SELECT pg_temp.exigir(NOT has_function_privilege('vec_ct122_runtime','vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),'fachada AD3 invocable por el ejecutor');
SET SESSION AUTHORIZATION vec_ct122_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_contratacion_temporal.cancelacion_expediente_v1','SELECT'),'lectura directa de cancelaciones');
SELECT pg_temp.exigir(NOT has_function_privilege('vec_contratacion_temporal.texto_valido_ct122(text,integer,boolean)','EXECUTE'),'auxiliar ejecutable');
SELECT pg_temp.exigir(NOT has_function_privilege('vec_contratacion_temporal.admision_cancelacion_ct122(jsonb,numeric,jsonb)','EXECUTE'),'admisión ejecutable');

BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(:'entrada_uno'::jsonb)::text AS prep \gset
SELECT pg_temp.preparar(:'entrada_fase'::jsonb)->>'resultado' AS prep_fase \gset
SELECT pg_temp.preparar(:'entrada_fisc'::jsonb)->>'resultado' AS prep_fisc \gset
SELECT pg_temp.preparar(jsonb_set(:'entrada_uno'::jsonb,'{material,version_esperada}','2'))->>'resultado' AS prep_version \gset
SELECT pg_temp.preparar(jsonb_set(:'entrada_uno'::jsonb,'{material,canal}','"intervencion"'))->>'error' AS prep_canal \gset
SELECT pg_temp.preparar(jsonb_set(:'entrada_uno'::jsonb,'{material,fases_admitidas}','["solicitud","solicitud"]'))->>'error' AS prep_repetidas \gset
SELECT pg_temp.consultar(:'org',:'exp_asignacion')::text AS consulta_vacia \gset
COMMIT;
SELECT pg_temp.exigir((:'prep'::jsonb)->>'resultado'='preparada' AND (:'prep'::jsonb)#>>'{expediente,version}'='3','preparación');
SELECT pg_temp.exigir(:'prep_fase'='fase_no_admitida','fase no admitida por la regla');
SELECT pg_temp.exigir(:'prep_fisc'='tras_fiscalizacion','preparación tras fiscalización');
SELECT pg_temp.exigir(:'prep_version'='version_en_conflicto','versión desfasada');
SELECT pg_temp.exigir(:'prep_canal'='22023','canal desconocido');
SELECT pg_temp.exigir(:'prep_repetidas'='22023','fases repetidas');
SELECT pg_temp.exigir((:'consulta_vacia'::jsonb)->'cancelacion'='null'::jsonb,'consulta sin cancelación');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir((pg_temp.preparar(:'entrada_uno'::jsonb))->>'error'='42501','preparación en transacción de escritura');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir((pg_temp.confirmar(:'entrada_uno'::jsonb))->>'error'='42501','confirmación en solo lectura');
ROLLBACK;

-- Negativos de confirmación: ninguno escribe.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_fisc'::jsonb)->>'resultado'='tras_fiscalizacion','confirmación tras fiscalización');
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_fase'::jsonb)->>'resultado'='fase_no_admitida','confirmación en fase no admitida');
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_accion'::jsonb)->>'error'='42501','acción ajena');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{expediente_siguiente,estado_actual}','"completado"'))->>'error'='22023','proyección manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{actuacion,fase_destino}','"informe_juridico"'))->>'error'='22023','actuación manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{contexto,atributos,motivo_clave}','"error_solicitud"'))->>'error'='42501','contexto manipulado');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{contexto,ambitos,fase_previa}','"solicitud"'))->>'error'='42501','fase previa manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_uno'::jsonb,' ')->>'error'='42501','decisión divergente');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{material,observaciones}','" con espacio"'))->>'error'='22023','texto con espacio de borde');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_uno'::jsonb,'{material,actor_ref}','"per_otro"'))->>'error'='42501','actor distinto del autorizado');
ROLLBACK;

-- Confirmación de RRHH y efectos en la misma transacción.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar(:'entrada_uno'::jsonb)::text AS conf \gset
COMMIT;
SELECT pg_temp.exigir((:'conf'::jsonb)->>'resultado'='confirmada' AND (:'conf'::jsonb)#>>'{recibo,estado_resultante}'='cancelado'
  AND (:'conf'::jsonb)#>>'{recibo,fase_resultante}'='asignacion_unidad' AND (:'conf'::jsonb)#>>'{recibo,version_resultante}'='4'
  AND (:'conf'::jsonb)#>>'{recibo,motivo_clave}'='necesidad_desaparecida','recibo de cancelación');
-- Repetición directa con la misma evidencia: mismo recibo, sin otra escritura.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_uno'::jsonb)#>>'{recibo,recibo_ref}'=(:'conf'::jsonb)#>>'{recibo,recibo_ref}','repetición directa');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir(pg_temp.preparar(:'entrada_uno'::jsonb)->>'resultado'='confirmada','recuperación por la preparación');
SELECT pg_temp.exigir(pg_temp.preparar(jsonb_set(:'entrada_uno'::jsonb,'{material,motivo_clave}','"error_solicitud"'))->>'resultado'='idempotencia_reutilizada','clave reutilizada con otra intención');
SELECT pg_temp.exigir(pg_temp.consultar(:'org',:'exp_asignacion')#>>'{cancelacion,motivo_clave}'='necesidad_desaparecida','consulta de la cancelación');
COMMIT;
-- Otra intención sobre el expediente ya cancelado.
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada(:'exp_asignacion',4,'rrhh','error_solicitud',:'fases'::jsonb,'','dos')::text AS entrada_dos \gset
SET SESSION AUTHORIZATION vec_ct122_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_dos'::jsonb)->>'resultado'='cancelacion_existente','segunda cancelación');
ROLLBACK;

-- Cancelación del centro: el ámbito autorizado lleva su centro.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'entrada_centro'::jsonb,'{contexto,ambitos,centro_ref}','"centro:otro"'))->>'error'='42501','centro ajeno');
SELECT pg_temp.exigir(pg_temp.confirmar(:'entrada_centro'::jsonb)#>>'{recibo,fase_resultante}'='solicitud','cancelación del centro');
COMMIT;
RESET SESSION AUTHORIZATION;

-- Historia de solo adición y publicación al cuadro.
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral
   WHERE expediente_ref=:'exp_asignacion' AND version=4 AND estado='cancelado' AND fase_clave='asignacion_unidad' AND origen_version='cancelacion_expediente_ct122')=1,'versión cancelada');
SELECT pg_temp.exigir((SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_asignacion')=4,'cabeza del expediente');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.actuacion_expediente_integral
   WHERE expediente_ref=:'exp_asignacion' AND secuencia=4 AND actuacion_json->>'accion_clave'='contratacion_temporal.expediente.cancelar')=1,'actuación');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral
   WHERE expediente_ref=:'exp_asignacion' AND tipo_evento='ct.expediente_cancelado.v1')=1,'evento en el outbox');
SELECT pg_temp.exigir((SELECT estado_clave FROM vec_contratacion_temporal.publicacion_version_rrhh
   WHERE expediente_ref=:'exp_asignacion' ORDER BY corte_global DESC LIMIT 1)='cancelado','publicación al cuadro RRHH');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.cancelacion_expediente_v1)=2,'dos cancelaciones');
SELECT pg_temp.exigir((SELECT centro_ref FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE canal='centro') IS NOT NULL,'centro registrado');
DO $$
BEGIN
    UPDATE vec_contratacion_temporal.cancelacion_expediente_v1 SET observaciones='x';
    RAISE EXCEPTION 'FALLO: la cancelación es mutable';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF;
END $$;
SELECT 'CT122 OK';
