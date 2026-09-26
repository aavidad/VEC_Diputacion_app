-- Recorrido de Selección 000001 con el LOGIN de ensayo (miembro solo de
-- vec_seleccion_ejecutor) y los dobles de AD3-89/AD3-90. Cada comprobación
-- imprime «ok»; cualquier desviación aborta con «FALLO».
\set ON_ERROR_STOP on
SET default_transaction_isolation = 'serializable';
SET timezone = 'UTC';

CREATE FUNCTION pg_temp.cap(p_op text, p_recurso text, p_extra jsonb DEFAULT '{}') RETURNS bytea LANGUAGE sql AS
$$ SELECT convert_to((jsonb_build_object('operacion', p_op, 'efecto_ref', p_recurso) || p_extra)::text, 'UTF8') $$;
CREATE FUNCTION pg_temp.dec(p_op text, p_recurso text) RETURNS bytea LANGUAGE sql AS
$$ SELECT convert_to(jsonb_build_object('accion', p_op, 'recurso_ref', p_recurso)::text, 'UTF8') $$;
CREATE FUNCTION pg_temp.ctx(p_persona text) RETURNS bytea LANGUAGE sql AS
$$ SELECT convert_to(jsonb_build_object('persona_ref', p_persona)::text, 'UTF8') $$;
CREATE FUNCTION pg_temp.huella(p text) RETURNS text LANGUAGE sql AS $$ SELECT encode(sha256(convert_to(p, 'UTF8')), 'hex') $$;

-- Guardado con el material de la persona; p_actor permite suplantar el
-- contexto atestado para los negativos.
CREATE FUNCTION pg_temp.guardar(p_persona text, p_conv text, p_conv_version int, p_esperada int, p_clave text, p_material text,
  p_requisitos jsonb DEFAULT '[{"clave":"nacionalidad","estado":"cumple"},{"clave":"titulacion","estado":"pendiente"}]',
  p_actor text DEFAULT NULL, p_extra jsonb DEFAULT '{}', p_completos boolean DEFAULT true)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, version integer, puntuacion_micropuntos bigint, datos_completos boolean) LANGUAGE sql AS $$
 SELECT * FROM vec_seleccion.guardar_borrador_propio_v1(p_persona, p_conv, p_conv_version, p_esperada, p_clave, pg_temp.huella(p_material),
  'sol_' || substr(pg_temp.huella(p_persona || p_conv), 1, 30), 'libre', 'clave:ensayo', '\x000102030405060708090a0b'::bytea,
  convert_to('cifrado-sintetico-' || p_material, 'UTF8'), CASE WHEN p_completos THEN pg_temp.huella('doc' || p_persona) END,
  CASE WHEN p_completos THEN '***5678*' END, p_completos, p_requisitos,
  '[{"clave_grupo":"experiencia","clave_merito":"meses","cantidad":"14"}]'::jsonb, 1400000,
  pg_temp.cap('seleccion.solicitudes_propias.guardar_borrador', 'mis-solicitudes:' || p_persona, p_extra),
  pg_temp.dec('seleccion.solicitudes_propias.guardar_borrador', 'mis-solicitudes:' || p_persona),
  '\x00'::bytea, pg_temp.ctx(coalesce(p_actor, p_persona)), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.presentar(p_persona text, p_sol text, p_esperada int, p_clave text, p_material text)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, numero_justificante text, presentada_en timestamptz, recibo_ref text, puntuacion_micropuntos bigint) LANGUAGE sql AS $$
 SELECT * FROM vec_seleccion.presentar_solicitud_propia_v1(p_persona, p_sol, p_esperada, p_clave, pg_temp.huella(p_material),
  'recibo:seleccion-presentacion:' || pg_temp.huella(p_sol || p_clave),
  pg_temp.cap('seleccion.solicitudes_propias.presentar', 'mis-solicitudes:' || p_persona),
  pg_temp.dec('seleccion.solicitudes_propias.presentar', 'mis-solicitudes:' || p_persona),
  '\x00'::bytea, pg_temp.ctx(p_persona), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.listar_propias(p_persona text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT vec_seleccion.listar_solicitudes_propias_v1(p_persona,
  pg_temp.cap('seleccion.solicitudes_propias.consultar', 'mis-solicitudes:' || p_persona),
  pg_temp.dec('seleccion.solicitudes_propias.consultar', 'mis-solicitudes:' || p_persona),
  '\x00'::bytea, pg_temp.ctx(p_persona), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.borrador(p_persona text, p_conv text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT vec_seleccion.leer_borrador_propio_v1(p_persona, p_conv,
  pg_temp.cap('seleccion.solicitudes_propias.consultar', 'mis-solicitudes:' || p_persona),
  pg_temp.dec('seleccion.solicitudes_propias.consultar', 'mis-solicitudes:' || p_persona),
  '\x00'::bytea, pg_temp.ctx(p_persona), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.listar_rrhh(p_conv text, p_cursor bigint DEFAULT 0, p_op text DEFAULT 'seleccion.solicitudes.consultar') RETURNS jsonb LANGUAGE sql AS $$
 SELECT vec_seleccion.listar_solicitudes_convocatoria_v1(p_conv, p_cursor, 50,
  pg_temp.cap(p_op, 'solicitudes-convocatoria:' || pg_temp.huella(p_conv)),
  pg_temp.dec(p_op, 'solicitudes-convocatoria:' || pg_temp.huella(p_conv)),
  '\x00'::bytea, pg_temp.ctx('per_RRRRRRRRRRRRRRRRRRRRRR'), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.detalle_rrhh(p_sol text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT vec_seleccion.leer_detalle_solicitud_v1(p_sol,
  pg_temp.cap('seleccion.solicitudes.consultar_detalle', 'solicitud-seleccion:' || p_sol),
  pg_temp.dec('seleccion.solicitudes.consultar_detalle', 'solicitud-seleccion:' || p_sol),
  '\x00'::bytea, pg_temp.ctx('per_RRRRRRRRRRRRRRRRRRRRRR'), 1, 1, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea, '\x00'::bytea)
$$;
CREATE FUNCTION pg_temp.espera(p_sql text, p_codigo text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN
 BEGIN EXECUTE p_sql;
 EXCEPTION WHEN others THEN
  IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'FALLO: % dio % (%), se esperaba %', p_sql, SQLSTATE, SQLERRM, p_codigo; END IF;
  RETURN 'ok';
 END;
 RAISE EXCEPTION 'FALLO: % se aceptó; se esperaba %', p_sql, p_codigo;
END $$;
CREATE FUNCTION pg_temp.cierto(p boolean, p_que text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF p IS NOT TRUE THEN RAISE EXCEPTION 'FALLO: %', p_que; END IF; RETURN 'ok'; END $$;
-- Contenido con la forma exacta que publica la aplicación (el adaptador Go
-- rechaza cualquier otra al leer).
CREATE FUNCTION pg_temp.contenido(p_patron text DEFAULT '{anio}/SOL-{numero}') RETURNS jsonb LANGUAGE sql AS $$
 SELECT jsonb_build_object('turnos', '[{"clave":"libre","etiqueta":"Turno libre"}]'::jsonb,
  'requisitos', '[{"clave":"nacionalidad","titulo":"Nacionalidad","descripcion":"","obligatorio":true,"impide_presentar":true},{"clave":"titulacion","titulo":"Titulación","descripcion":"","obligatorio":true,"impide_presentar":false}]'::jsonb,
  'baremo', '{"maximo":"20","redondeo":"mitad_arriba","grupos":[]}'::jsonb, 'numeracion', jsonb_build_object('patron', p_patron, 'ancho', 6),
  'fecha_referencia', '2026-10-30', 'marca_ejemplo', true)
$$;

\set abierta 'proceso:publico:ensayo-seleccion-abierta'
\set cerrada 'proceso:publico:ensayo-seleccion-cerrada'
\set A 'per_AAAAAAAAAAAAAAAAAAAAAA'
\set B 'per_BBBBBBBBBBBBBBBBBBBBBB'

-- 1. Publicación gobernada e idempotente.
SELECT (now() - interval '1 day')::text AS abre, (now() + interval '10 days')::text AS cierra \gset
SELECT pg_temp.cierto((SELECT nueva AND version = 1 FROM vec_seleccion.publicar_convocatoria_v1(:'abierta', 'Bolsa de ensayo', :'abre', :'cierra', pg_temp.contenido(), pg_temp.huella('c1'), '2026-09-01T00:00:00Z')), 'primera publicación');
SELECT pg_temp.cierto((SELECT NOT nueva AND version = 1 FROM vec_seleccion.publicar_convocatoria_v1(:'abierta', 'Bolsa de ensayo', :'abre', :'cierra', pg_temp.contenido(), pg_temp.huella('c1'), '2026-09-01T00:00:00Z')), 'republicar igual reutiliza la versión');
SELECT pg_temp.espera($$SELECT * FROM vec_seleccion.publicar_convocatoria_v1('proceso:publico:ensayo-seleccion-abierta', 'Otro título', now(), now() + interval '1 day', '{}'::jsonb, encode(sha256(convert_to('c1','UTF8')),'hex'), '2026-09-01T00:00:00Z')$$, '22023');
SELECT pg_temp.cierto((SELECT nueva FROM vec_seleccion.publicar_convocatoria_v1(:'cerrada', 'Bolsa cerrada', now() - interval '10 days', now() - interval '1 day', pg_temp.contenido(), pg_temp.huella('cerrada'), '2026-09-01T00:00:00Z')), 'convocatoria cerrada publicada');
SELECT pg_temp.cierto((SELECT count(*) = 2 AND bool_or((c->>'abierta')::boolean) AND NOT bool_and((c->>'abierta')::boolean) FROM jsonb_array_elements(vec_seleccion.convocatorias_publicadas_v1()) c WHERE c->>'convocatoria_ref' LIKE 'proceso:publico:ensayo-seleccion-%'), 'plazo evaluado con el reloj de la base');

-- 2. Borrador versionado e idempotente.
SELECT pg_temp.cierto((SELECT NOT reutilizada AND version = 1 FROM pg_temp.guardar(:'A', :'abierta', 1, 0, 'clave-borrador-1', 'm1')), 'primer borrador');
SELECT pg_temp.cierto((SELECT reutilizada AND version = 1 FROM pg_temp.guardar(:'A', :'abierta', 1, 0, 'clave-borrador-1', 'm1')), 'repetición idempotente');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 0, 'clave-borrador-1', 'otro')$$, :'A', :'abierta'), 'VSL02');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 0, 'clave-borrador-2', 'm2')$$, :'A', :'abierta'), 'VSL03');
SELECT pg_temp.cierto((SELECT version = 2 FROM pg_temp.guardar(:'A', :'abierta', 1, 1, 'clave-borrador-3', 'm3')), 'segunda versión');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 1, 'clave-borrador-4', 'm4')$$, :'A', :'abierta'), 'VSL03');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 0, 'clave-cerrada-1', 'mc')$$, :'A', :'cerrada'), 'VSL01');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 0, 'clave-nada-01', 'mn')$$, :'A', 'proceso:publico:inexistente'), 'VSL07');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 0, 'clave-ajena-1', 'mx', p_actor => %L)$$, :'A', :'abierta', :'B'), '42501');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 2, 'clave-repe-01', 'mr', p_extra => '{"repetida":true}')$$, :'A', :'abierta'), '42501');

-- 3. Otra persona no ve ni toca la solicitud ajena.
SELECT pg_temp.cierto(jsonb_array_length(pg_temp.listar_propias(:'B')) = 0, 'B no ve solicitudes ajenas');
SELECT pg_temp.cierto(pg_temp.borrador(:'B', :'abierta') IS NULL, 'B no lee el borrador ajeno');
SELECT pg_temp.cierto((pg_temp.borrador(:'A', :'abierta')->>'version')::int = 2, 'A lee su última versión');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 2, 'clave-pres-B1', 'pb')$$, :'B', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref')), 'VSL08');

-- 4. Bases actualizadas: hay que volver a guardar.
SELECT pg_temp.cierto((SELECT nueva AND version = 2 FROM vec_seleccion.publicar_convocatoria_v1(:'abierta', 'Bolsa de ensayo', :'abre', :'cierra', pg_temp.contenido(), pg_temp.huella('c2'), '2026-09-02T00:00:00Z')), 'versión 2 de las bases');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 2, 'clave-pres-A1', 'p2')$$, :'A', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref')), 'VSL06');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 1, 2, 'clave-borrador-5', 'm5')$$, :'A', :'abierta'), 'VSL06');

-- 5. Requisito obligatorio que impide presentar en «no_cumple».
SELECT pg_temp.cierto((SELECT version = 3 FROM pg_temp.guardar(:'A', :'abierta', 2, 2, 'clave-borrador-6', 'm6', '[{"clave":"nacionalidad","estado":"no_cumple"}]')), 'borrador con requisito no cumplido');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 3, 'clave-pres-A2', 'p3')$$, :'A', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref')), 'VSL05');
-- «pendiente» en un requisito obligatorio no excluye; «no_cumple» en uno que no impide presentar, tampoco.
SELECT pg_temp.cierto((SELECT version = 4 FROM pg_temp.guardar(:'A', :'abierta', 2, 3, 'clave-borrador-7', 'm7', '[{"clave":"nacionalidad","estado":"pendiente"},{"clave":"titulacion","estado":"no_cumple"}]')), 'borrador con pendiente');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 3, 'clave-pres-A3', 'p4')$$, :'A', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref')), 'VSL03');

-- Un borrador incompleto se guarda, pero no se presenta.
SELECT pg_temp.cierto((SELECT version = 5 AND NOT datos_completos FROM pg_temp.guardar(:'A', :'abierta', 2, 4, 'clave-borrador-9', 'm9', p_completos => false)), 'borrador incompleto admitido');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 5, 'clave-pres-A6', 'p6')$$, :'A', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref')), 'VSL09');
SELECT pg_temp.cierto((SELECT version = 6 AND datos_completos FROM pg_temp.guardar(:'A', :'abierta', 2, 5, 'clave-borrador-10', 'm10', '[{"clave":"nacionalidad","estado":"pendiente"},{"clave":"titulacion","estado":"no_cumple"}]')), 'borrador completado');

-- 6. Presentación, justificante y repetición.
CREATE TEMP TABLE recibo AS SELECT * FROM pg_temp.presentar(:'A', (pg_temp.borrador(:'A', :'abierta')->>'solicitud_ref'), 6, 'clave-pres-A4', 'p5');
SELECT pg_temp.cierto((SELECT NOT reutilizada AND numero_justificante = extract(year FROM presentada_en AT TIME ZONE 'Europe/Madrid')::int || '/SOL-000001' AND puntuacion_micropuntos = 1400000 FROM recibo), 'justificante interno AAAA/SOL-000001');
SELECT pg_temp.cierto((SELECT p.reutilizada AND p.numero_justificante = r.numero_justificante AND p.presentada_en = r.presentada_en AND p.recibo_ref = r.recibo_ref
  FROM recibo r, pg_temp.presentar(:'A', r.solicitud_ref, 6, 'clave-pres-A4', 'p5') p), 'repetición: mismo recibo, número e instante');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 6, 'clave-pres-A4', 'otro')$$, :'A', (SELECT solicitud_ref FROM recibo)), 'VSL02');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.presentar(%L, %L, 6, 'clave-pres-A5', 'p5')$$, :'A', (SELECT solicitud_ref FROM recibo)), 'VSL04');
SELECT pg_temp.espera(format($$SELECT * FROM pg_temp.guardar(%L, %L, 2, 6, 'clave-borrador-8', 'm8')$$, :'A', :'abierta'), 'VSL04');
SELECT pg_temp.cierto((SELECT l->>'estado' = 'presentada' AND l->>'numero_justificante' = (SELECT numero_justificante FROM recibo) FROM jsonb_array_elements(pg_temp.listar_propias(:'A')) l), 'A ve su solicitud presentada');
-- Segunda persona: numeración correlativa.
SELECT pg_temp.cierto((SELECT version = 1 FROM pg_temp.guardar(:'B', :'abierta', 2, 0, 'clave-borrador-B', 'mb')), 'borrador de B');
SELECT pg_temp.cierto((SELECT numero_justificante LIKE '%/SOL-000002' FROM pg_temp.presentar(:'B', (pg_temp.borrador(:'B', :'abierta')->>'solicitud_ref'), 1, 'clave-pres-B2', 'pb2')), 'segundo justificante correlativo');

-- 7. RRHH: listado minimizado y ficha auditada.
SELECT pg_temp.cierto(jsonb_array_length(pg_temp.listar_rrhh(:'abierta')) = 2, 'RRHH ve las dos presentadas');
SELECT pg_temp.cierto(jsonb_array_length(pg_temp.listar_rrhh(:'abierta', (pg_temp.listar_rrhh(:'abierta')->0->>'presentacion_id')::bigint)) = 1, 'paginación por cursor');
SELECT pg_temp.cierto((SELECT d->>'numero_justificante' = (SELECT numero_justificante FROM recibo) AND jsonb_array_length(d->'historia') = 7
  FROM pg_temp.detalle_rrhh((SELECT solicitud_ref FROM recibo)) d), 'ficha con historia de solo adición');
SELECT pg_temp.espera(format($$SELECT pg_temp.detalle_rrhh(%L)$$, 'sol_' || repeat('z', 30)), 'VSL08');
SELECT pg_temp.espera(format($$SELECT pg_temp.listar_rrhh(%L, 0, 'seleccion.solicitudes_propias.consultar')$$, :'abierta'), '42501');
SELECT 'fin_recorrido';
