-- Solo para la copia ficticia HZ37 del recorrido completo. El preparador
-- sustituye los dos marcadores tras cotejar el volcado y la copia.
-- Una segunda ejecución se rechaza por la preimagen exacta.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

DO $guarda$
BEGIN
  IF current_user <> 'postgres' OR current_database() <> 'postgres'
     OR (SELECT system_identifier::text FROM pg_control_system()) <> '__SID_COPIA__'
     OR '__SID_COPIA__' !~ '^[0-9]{15,20}$'
     OR '__HUELLA_FICHERO__' !~ '^[0-9a-f]{64}$' THEN
    RAISE EXCEPTION 'fixture HZ37: identidad de copia o parámetros incorrectos';
  END IF;
  IF (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote
      WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
        AND acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
        AND huella_fichero_sha256 = '__HUELLA_FICHERO__'
        AND categoria_ref = 'categoria:rpt:administrativo'
        AND bolsa_ref = 'bolsa:administrativo:2026-09-17'
        AND estado_conciliacion = 'pendiente' AND estado_staging = 'disponible'
        AND version = 1 AND secuencia_historia = 1) <> 1 THEN
    RAISE EXCEPTION 'fixture HZ37: lote de CONVOCA distinto o ausente';
  END IF;
  IF (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion
      WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
        AND bolsa_ref = 'bolsa:administrativo:2026-09-17'
        AND instantanea_ref = 'instantanea:constitucion:klbedadnegieljnnecgmokbpkhjojcdd'
        AND categoria_ref = 'categoria:rpt:administrativo') <> 1
     OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.recibo_carga_convoca
                WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84') THEN
    RAISE EXCEPTION 'fixture HZ37: constitución o recibo distintos';
  END IF;
  IF (SELECT count(*) FROM vec_bolsa_llamamientos.sustitucion_bolsa
      WHERE bolsa_ref_sustituida = 'bolsa:administrativo:2026-09-17'
        AND bolsa_ref_nueva = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk') <> 1 THEN
    RAISE EXCEPTION 'fixture HZ37: sustitución distinta';
  END IF;
END $guarda$;

CREATE TEMP TABLE vec_fixture_participaciones ON COMMIT DROP AS
SELECT participacion_ref FROM vec_bolsa_llamamientos.vinculo_candidato
WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84';
CREATE TEMP TABLE vec_fixture_llamamientos ON COMMIT DROP AS
SELECT llamamiento_ref FROM vec_bolsa_llamamientos.llamamiento_emitido
WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17';

CREATE TEMP VIEW vec_fixture_cuentas AS
SELECT 'fila_staging' entidad, count(*) filas FROM vec_bolsa_importacion_convoca.fila_staging WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'historia_estado', count(*) FROM vec_bolsa_importacion_convoca.historia_estado WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'lote', count(*) FROM vec_bolsa_importacion_convoca.lote WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'conciliacion', count(*) FROM vec_bolsa_importacion_convoca.conciliacion WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'decision_retencion', count(*) FROM vec_bolsa_importacion_convoca.decision_retencion WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'outbox', count(*) FROM vec_bolsa_importacion_convoca.outbox WHERE importacion_ref = 'importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'original', count(*) FROM vec_bolsa_importacion_convoca.original_exportacion_cifrada WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'bolsa', count(*) FROM vec_bolsa_llamamientos.bolsa_constituida WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'ambito', count(*) FROM vec_bolsa_llamamientos.ambito_bolsa WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'constitucion', count(*) FROM vec_bolsa_llamamientos.constitucion WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'instantanea', count(*) FROM vec_bolsa_llamamientos.instantanea_orden_bolsa WHERE instantanea_ref = 'instantanea:constitucion:klbedadnegieljnnecgmokbpkhjojcdd'
UNION ALL SELECT 'entradas', count(*) FROM vec_bolsa_llamamientos.constitucion_entrada WHERE instantanea_ref = 'instantanea:constitucion:klbedadnegieljnnecgmokbpkhjojcdd'
UNION ALL SELECT 'vinculos', count(*) FROM vec_bolsa_llamamientos.vinculo_candidato WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'datos_contacto', count(*) FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones)
UNION ALL SELECT 'situaciones', count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones)
UNION ALL SELECT 'operaciones_situacion', count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones)
UNION ALL SELECT 'trazas', count(*) FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones)
UNION ALL SELECT 'reposiciones', count(*) FROM vec_bolsa_llamamientos.reposicion_orden_bolsa WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'politica_orden', count(*) FROM vec_bolsa_llamamientos.politica_orden_bolsa WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'politica_lectura', count(*) FROM vec_bolsa_llamamientos.politica_ofertas_lectura_v3 WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'llamamientos', count(*) FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'cuerpos', count(*) FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'completitud', count(*) FROM vec_bolsa_llamamientos.completitud_correo_llamamiento WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'coordinacion', count(*) FROM vec_bolsa_llamamientos.coordinacion_completitud_correo WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'contactos', count(*) FROM vec_bolsa_llamamientos.contacto_participacion WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17' OR participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones) OR llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'respuestas', count(*) FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos)
UNION ALL SELECT 'sustitucion', count(*) FROM vec_bolsa_llamamientos.sustitucion_bolsa WHERE bolsa_ref_sustituida = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'origen_llamamiento_ct', count(*) FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'origen_oferta_ct', count(*) FROM vec_bolsa_llamamientos.origen_oferta_ct_v1 WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'inscripcion_asociada', count(*) FROM vec_bolsa_llamamientos.inscripcion_acta_asociada WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'inscripcion_incorporada', count(*) FROM vec_bolsa_llamamientos.inscripcion_incorporacion WHERE acta_ref = 'acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84'
UNION ALL SELECT 'ofertas', count(*) FROM vec_bolsa_llamamientos.oferta_publicada WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'confirmaciones', count(*) FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa WHERE bolsa_ref = 'bolsa:administrativo:2026-09-17'
UNION ALL SELECT 'justificantes', count(*) FROM vec_bolsa_llamamientos.respuesta_justificante_documental_b119 WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);

CREATE TEMP TABLE vec_fixture_esperado(entidad text PRIMARY KEY, filas bigint) ON COMMIT DROP;
INSERT INTO vec_fixture_esperado VALUES
('fila_staging',41),('historia_estado',1),('lote',1),('conciliacion',0),
('decision_retencion',0),('outbox',0),('original',0),('bolsa',1),('ambito',1),
('constitucion',1),('instantanea',1),('entradas',41),('vinculos',41),
('datos_contacto',41),('situaciones',65),('operaciones_situacion',3),
('trazas',108),('reposiciones',2),('politica_orden',1),('politica_lectura',1),
('llamamientos',3),('cuerpos',3),('completitud',3),('coordinacion',3),
('contactos',6),('respuestas',2),('sustitucion',1),('origen_llamamiento_ct',0),
('origen_oferta_ct',0),('inscripcion_asociada',0),('inscripcion_incorporada',0),
('ofertas',0),('confirmaciones',0),('justificantes',0);

-- La bolsa activa se guarda como multiconjunto de huellas de filas; incluye
-- sus entradas y datos de sus propias participaciones, incluso si están vacíos.
CREATE TEMP VIEW vec_fixture_activa AS
SELECT 'bolsa' entidad, count(*) filas, md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) huella FROM vec_bolsa_llamamientos.bolsa_constituida t WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'
UNION ALL SELECT 'ambito', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.ambito_bolsa t WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'
UNION ALL SELECT 'constitucion', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.constitucion t WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'
UNION ALL SELECT 'instantanea', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.instantanea_orden_bolsa t WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'
UNION ALL SELECT 'entradas', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.constitucion_entrada t WHERE instantanea_ref IN (SELECT instantanea_ref FROM vec_bolsa_llamamientos.instantanea_orden_bolsa WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk')
UNION ALL SELECT 'vinculos', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.vinculo_candidato t WHERE acta_ref IN (SELECT acta_ref FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk')
UNION ALL SELECT 'datos_contacto', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.datos_contacto_participacion t WHERE participacion_ref IN (SELECT participacion_ref FROM vec_bolsa_llamamientos.vinculo_candidato WHERE acta_ref IN (SELECT acta_ref FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'))
UNION ALL SELECT 'situaciones', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.situacion_participacion t WHERE participacion_ref IN (SELECT participacion_ref FROM vec_bolsa_llamamientos.vinculo_candidato WHERE acta_ref IN (SELECT acta_ref FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk'))
UNION ALL SELECT 'contactos', count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), ',' ORDER BY md5(to_jsonb(t)::text)),'')) FROM vec_bolsa_llamamientos.contacto_participacion t WHERE bolsa_ref = 'bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
CREATE TEMP TABLE vec_fixture_activa_antes ON COMMIT DROP AS SELECT * FROM vec_fixture_activa;

CREATE TEMP TABLE vec_fixture_triggers(tabla regclass, nombre text, PRIMARY KEY(tabla,nombre)) ON COMMIT DROP;
INSERT INTO vec_fixture_triggers VALUES
('vec_bolsa_importacion_convoca.historia_estado','historia_estado_inmutable'),
('vec_bolsa_importacion_convoca.lote','lote_transicion_cerrada'),
('vec_bolsa_llamamientos.ambito_bolsa','negar_mutacion'),
('vec_bolsa_llamamientos.bolsa_constituida','negar_mutacion'),
('vec_bolsa_llamamientos.constitucion','negar_mutacion'),
('vec_bolsa_llamamientos.constitucion_entrada','negar_mutacion'),
('vec_bolsa_llamamientos.contacto_participacion','contacto_participacion_inmutable'),
('vec_bolsa_llamamientos.cuerpo_llamamiento_contacto','cuerpo_llamamiento_contacto_inmutable'),
('vec_bolsa_llamamientos.datos_contacto_participacion','datos_contacto_participacion_inmutable'),
('vec_bolsa_llamamientos.instantanea_orden_bolsa','negar_mutacion'),
('vec_bolsa_llamamientos.llamamiento_emitido','llamamiento_emitido_inmutable'),
('vec_bolsa_llamamientos.operacion_situacion_participacion','operacion_situacion_participacion_inmutable'),
('vec_bolsa_llamamientos.politica_ofertas_lectura_v3','politica_ofertas_lectura_inmutable'),
('vec_bolsa_llamamientos.politica_orden_bolsa','politica_orden_bolsa_inmutable'),
('vec_bolsa_llamamientos.reposicion_orden_bolsa','reposicion_orden_bolsa_inmutable'),
('vec_bolsa_llamamientos.respuesta_portal_llamamiento','respuesta_portal_llamamiento_inmutable'),
('vec_bolsa_llamamientos.situacion_participacion','situacion_participacion_inmutable'),
('vec_bolsa_llamamientos.sustitucion_bolsa','sustitucion_bolsa_inmutable'),
('vec_bolsa_llamamientos.traza_valor_participacion','traza_valor_participacion_inmutable'),
('vec_bolsa_llamamientos.vinculo_candidato','negar_mutacion');

DO $preimagen$
BEGIN
  IF EXISTS (SELECT 1 FROM vec_fixture_esperado e FULL JOIN vec_fixture_cuentas c USING(entidad)
             WHERE e.filas IS DISTINCT FROM c.filas) THEN
    RAISE EXCEPTION 'fixture HZ37: cantidades anteriores distintas';
  END IF;
  IF (SELECT filas FROM vec_fixture_activa_antes WHERE entidad='bolsa') <> 1
     OR (SELECT filas FROM vec_fixture_activa_antes WHERE entidad='constitucion') <> 1
     OR (SELECT filas FROM vec_fixture_activa_antes WHERE entidad='entradas') <> 12
     OR (SELECT filas FROM vec_fixture_activa_antes WHERE entidad='vinculos') <> 12 THEN
    RAISE EXCEPTION 'fixture HZ37: bolsa activa distinta';
  END IF;
  IF EXISTS (SELECT 1 FROM vec_fixture_triggers e LEFT JOIN pg_trigger t
             ON t.tgrelid=e.tabla AND t.tgname=e.nombre
             WHERE t.oid IS NULL OR t.tgisinternal OR t.tgenabled <> 'O') THEN
    RAISE EXCEPTION 'fixture HZ37: triggers de inmutabilidad distintos';
  END IF;
END $preimagen$;

-- Solo se suspenden guardas de DELETE inmutable; las FK quedan habilitadas.
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado DISABLE TRIGGER historia_estado_inmutable;
ALTER TABLE vec_bolsa_importacion_convoca.lote DISABLE TRIGGER lote_transicion_cerrada;
ALTER TABLE vec_bolsa_llamamientos.ambito_bolsa DISABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.bolsa_constituida DISABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.constitucion DISABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.constitucion_entrada DISABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DISABLE TRIGGER contacto_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto DISABLE TRIGGER cuerpo_llamamiento_contacto_inmutable;
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion DISABLE TRIGGER datos_contacto_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.instantanea_orden_bolsa DISABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido DISABLE TRIGGER llamamiento_emitido_inmutable;
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion DISABLE TRIGGER operacion_situacion_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.politica_ofertas_lectura_v3 DISABLE TRIGGER politica_ofertas_lectura_inmutable;
ALTER TABLE vec_bolsa_llamamientos.politica_orden_bolsa DISABLE TRIGGER politica_orden_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa DISABLE TRIGGER reposicion_orden_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento DISABLE TRIGGER respuesta_portal_llamamiento_inmutable;
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion DISABLE TRIGGER situacion_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.sustitucion_bolsa DISABLE TRIGGER sustitucion_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.traza_valor_participacion DISABLE TRIGGER traza_valor_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.vinculo_candidato DISABLE TRIGGER negar_mutacion;

DELETE FROM vec_bolsa_llamamientos.completitud_correo_llamamiento WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.coordinacion_completitud_correo WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.cuerpo_llamamiento_contacto WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.contacto_participacion WHERE bolsa_ref='bolsa:administrativo:2026-09-17' OR participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones) OR llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE llamamiento_ref IN (SELECT llamamiento_ref FROM vec_fixture_llamamientos);
DELETE FROM vec_bolsa_llamamientos.politica_ofertas_lectura_v3 WHERE bolsa_ref='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_llamamientos.reposicion_orden_bolsa WHERE bolsa_ref='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones);
DELETE FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones);
DELETE FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones);
DELETE FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones);
DELETE FROM vec_bolsa_llamamientos.politica_orden_bolsa WHERE bolsa_ref='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_llamamientos.sustitucion_bolsa WHERE bolsa_ref_sustituida='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_llamamientos.vinculo_candidato WHERE participacion_ref IN (SELECT participacion_ref FROM vec_fixture_participaciones);
DELETE FROM vec_bolsa_llamamientos.constitucion_entrada WHERE instantanea_ref='instantanea:constitucion:klbedadnegieljnnecgmokbpkhjojcdd';
DELETE FROM vec_bolsa_llamamientos.constitucion WHERE acta_ref='acta:importacion-convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84';
DELETE FROM vec_bolsa_llamamientos.instantanea_orden_bolsa WHERE instantanea_ref='instantanea:constitucion:klbedadnegieljnnecgmokbpkhjojcdd';
DELETE FROM vec_bolsa_llamamientos.ambito_bolsa WHERE bolsa_ref='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_llamamientos.bolsa_constituida WHERE bolsa_ref='bolsa:administrativo:2026-09-17';
DELETE FROM vec_bolsa_importacion_convoca.fila_staging WHERE importacion_ref='importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84';
DELETE FROM vec_bolsa_importacion_convoca.historia_estado WHERE importacion_ref='importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84';
DELETE FROM vec_bolsa_importacion_convoca.lote WHERE importacion_ref='importacion:convoca:fc6a18e71a1d654708eaf45d1c774818cd95b388dd7ad13e53a241a0b6f6ba84' AND huella_fichero_sha256='__HUELLA_FICHERO__';

-- Comprueba también las FK diferidas antes de restaurar las guardas.
SET CONSTRAINTS ALL IMMEDIATE;

ALTER TABLE vec_bolsa_importacion_convoca.historia_estado ENABLE TRIGGER historia_estado_inmutable;
ALTER TABLE vec_bolsa_importacion_convoca.lote ENABLE TRIGGER lote_transicion_cerrada;
ALTER TABLE vec_bolsa_llamamientos.ambito_bolsa ENABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.bolsa_constituida ENABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.constitucion ENABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.constitucion_entrada ENABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ENABLE TRIGGER contacto_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.cuerpo_llamamiento_contacto ENABLE TRIGGER cuerpo_llamamiento_contacto_inmutable;
ALTER TABLE vec_bolsa_llamamientos.datos_contacto_participacion ENABLE TRIGGER datos_contacto_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.instantanea_orden_bolsa ENABLE TRIGGER negar_mutacion;
ALTER TABLE vec_bolsa_llamamientos.llamamiento_emitido ENABLE TRIGGER llamamiento_emitido_inmutable;
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion ENABLE TRIGGER operacion_situacion_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.politica_ofertas_lectura_v3 ENABLE TRIGGER politica_ofertas_lectura_inmutable;
ALTER TABLE vec_bolsa_llamamientos.politica_orden_bolsa ENABLE TRIGGER politica_orden_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa ENABLE TRIGGER reposicion_orden_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento ENABLE TRIGGER respuesta_portal_llamamiento_inmutable;
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion ENABLE TRIGGER situacion_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.sustitucion_bolsa ENABLE TRIGGER sustitucion_bolsa_inmutable;
ALTER TABLE vec_bolsa_llamamientos.traza_valor_participacion ENABLE TRIGGER traza_valor_participacion_inmutable;
ALTER TABLE vec_bolsa_llamamientos.vinculo_candidato ENABLE TRIGGER negar_mutacion;

DO $postimagen$
BEGIN
  IF EXISTS (SELECT 1 FROM vec_fixture_cuentas WHERE filas <> 0) THEN
    RAISE EXCEPTION 'fixture HZ37: quedan filas del lote demostrativo';
  END IF;
  IF EXISTS (SELECT 1 FROM vec_fixture_activa_antes a FULL JOIN vec_fixture_activa b USING(entidad)
             WHERE a.filas IS DISTINCT FROM b.filas OR a.huella IS DISTINCT FROM b.huella) THEN
    RAISE EXCEPTION 'fixture HZ37: cambió la bolsa activa';
  END IF;
  IF EXISTS (SELECT 1 FROM vec_fixture_triggers e LEFT JOIN pg_trigger t
             ON t.tgrelid=e.tabla AND t.tgname=e.nombre
             WHERE t.oid IS NULL OR t.tgisinternal OR t.tgenabled <> 'O') THEN
    RAISE EXCEPTION 'fixture HZ37: trigger sin restaurar';
  END IF;
END $postimagen$;
SELECT 'fixture_hz37_lote_retirado' AS resultado,
       (SELECT count(*) FROM vec_fixture_cuentas WHERE filas=0) AS comprobaciones_vacias,
       (SELECT filas FROM vec_fixture_activa WHERE entidad='bolsa') AS bolsas_activas_conservadas,
       (SELECT filas FROM vec_fixture_activa WHERE entidad='entradas') AS entradas_activas_conservadas;
DROP VIEW vec_fixture_activa;
DROP VIEW vec_fixture_cuentas;
COMMIT;
