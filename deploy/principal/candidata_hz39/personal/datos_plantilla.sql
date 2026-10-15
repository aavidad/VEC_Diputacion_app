\set ON_ERROR_STOP on
-- Datos ficticios de activación; no acredita una aprobación oficial de Plantilla.
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SELECT pg_advisory_xact_lock(hashtextextended('datos:hz39:personal:plantilla',0));
DO $fixture$
DECLARE pl vec_personal.version_plantilla_historia%ROWTYPE;
        l vec_personal.plaza_plantilla_historia%ROWTYPE;
        instante timestamptz(6);
BEGIN
  SELECT * INTO STRICT pl FROM vec_personal.version_plantilla_historia
  WHERE organismo_ref='org_0e4b1d56ba2e24fb8e4808cbc7abb660'
    AND codigo_version_fuente='Plantilla VEC 2026 desde RPT 2026'
    AND revision=1 AND estado='publicada';
  SELECT * INTO STRICT l FROM vec_personal.plaza_plantilla_historia
  WHERE organismo_ref=pl.organismo_ref AND codigo_plaza_fuente='2026/0033'
    AND revision=1 AND plantilla_version_ref=pl.version_ref AND plantilla_revision=1;
  IF EXISTS(SELECT 1 FROM vec_personal.version_plantilla_historia p
      WHERE p.version_ref=pl.version_ref AND p.revision=2
        AND (to_jsonb(p)-'conocido_desde') =
            (to_jsonb(pl)-'conocido_desde')||jsonb_build_object(
              'revision',2,'aprobada_en','2026-05-07','publicada_en','2026-05-07'))
     AND EXISTS(SELECT 1 FROM vec_personal.plaza_plantilla_historia p
      WHERE p.plaza_ref=l.plaza_ref AND p.revision=2
        AND (to_jsonb(p)-'conocido_desde') =
            (to_jsonb(l)-'conocido_desde')||jsonb_build_object('revision',2,'plantilla_revision',2))
  THEN RETURN; END IF;
  IF pl.aprobada_en IS NOT NULL OR pl.publicada_en IS NOT NULL THEN
    RAISE EXCEPTION 'datos de plantilla: fecha de partida incompatible' USING ERRCODE='55000'; END IF;
  IF (SELECT count(*) FROM vec_personal.version_plantilla_historia WHERE version_ref=pl.version_ref)<>1
     OR (SELECT count(*) FROM vec_personal.plaza_plantilla_historia WHERE plaza_ref=l.plaza_ref)<>1
     OR (SELECT count(*) FROM vec_personal.version_rpt_historia
         WHERE organismo_ref=pl.organismo_ref AND codigo_version_fuente='RPT 2026 rev. 2026-05-07'
           AND estado='publicada' AND aprobada_en=DATE '2026-05-07'
           AND publicada_en=DATE '2026-05-07')<>1
  THEN RAISE EXCEPTION 'preimagen ficticia de plantilla/plaza incompatible' USING ERRCODE='55000'; END IF;
  instante:=date_trunc('microseconds',clock_timestamp());
  IF instante<=pl.conocido_desde OR instante<=l.conocido_desde THEN
    RAISE EXCEPTION 'instante ficticio no posterior' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_personal.version_plantilla_historia
   (version_ref,revision,organismo_ref,ejercicio,codigo_version_fuente,estado,
    aprobada_en,publicada_en,retirado,vigente_desde,vigente_hasta,conocido_desde,
    version_previa_ref,fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
  VALUES(pl.version_ref,2,pl.organismo_ref,pl.ejercicio,pl.codigo_version_fuente,pl.estado,
    DATE '2026-05-07',DATE '2026-05-07',pl.retirado,pl.vigente_desde,pl.vigente_hasta,instante,
    pl.version_previa_ref,pl.fuente_ref,pl.documento_ref,pl.acto_ref,pl.huella_fuente_sha256);
  INSERT INTO vec_personal.plaza_plantilla_historia
   (plaza_ref,revision,organismo_ref,unidad_ref,plantilla_version_ref,plantilla_revision,
    codigo_plaza_fuente,clasificacion_ref,estado_estructural,dotacion_presupuestaria,
    retirado,vigente_desde,vigente_hasta,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
  VALUES(l.plaza_ref,2,l.organismo_ref,l.unidad_ref,l.plantilla_version_ref,2,
    l.codigo_plaza_fuente,l.clasificacion_ref,l.estado_estructural,l.dotacion_presupuestaria,
    l.retirado,l.vigente_desde,l.vigente_hasta,instante,l.fuente_ref,l.acto_ref,l.huella_fuente_sha256);
END $fixture$;
DO $cobertura$
DECLARE c vec_personal.cobertura_ocupaciones_historia%ROWTYPE;
BEGIN
  SELECT h.* INTO STRICT c FROM vec_personal.cobertura_ocupaciones_historia h
  JOIN vec_personal.version_plantilla_historia p
    ON p.version_ref=h.plantilla_version_ref AND p.organismo_ref=h.organismo_ref
  WHERE h.organismo_ref='org_0e4b1d56ba2e24fb8e4808cbc7abb660'
    AND h.revision=1 AND h.plantilla_revision=1 AND h.estado='completa'
    AND p.codigo_version_fuente='Plantilla VEC 2026 desde RPT 2026'
    AND p.revision=1 AND p.estado='publicada';
  IF EXISTS(SELECT 1 FROM vec_personal.cobertura_ocupaciones_historia h
      WHERE h.cobertura_ref=c.cobertura_ref AND h.revision=2
        AND (to_jsonb(h)-'conocido_desde') =
            (to_jsonb(c)-'conocido_desde')||jsonb_build_object('revision',2,'plantilla_revision',2))
  THEN RETURN; END IF;
  IF (SELECT count(*) FROM vec_personal.cobertura_ocupaciones_historia WHERE cobertura_ref=c.cobertura_ref)<>1
     OR (SELECT count(*) FROM vec_personal.version_plantilla_historia
         WHERE version_ref=c.plantilla_version_ref AND revision=2 AND estado='publicada'
           AND aprobada_en=DATE '2026-05-07' AND publicada_en=DATE '2026-05-07')<>1
  THEN RAISE EXCEPTION 'preimagen de cobertura ficticia incompatible' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_personal.cobertura_ocupaciones_historia
    (cobertura_ref,revision,organismo_ref,plantilla_version_ref,plantilla_revision,
     estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,
     fuente_version,fuente_huella_sha256)
  VALUES(c.cobertura_ref,2,c.organismo_ref,c.plantilla_version_ref,2,
     c.estado,c.vigente_desde,c.vigente_hasta,clock_timestamp(),c.acto_ref,c.fuente_ref,
     c.fuente_version,c.fuente_huella_sha256);
END $cobertura$;
COMMIT;
