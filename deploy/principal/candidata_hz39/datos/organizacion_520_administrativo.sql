\set ON_ERROR_STOP on
-- Derivada de deploy/postgresql/personal/datos/organizacion_rpt_2026_v1.sql; añade Administrativo 9/01, centro 520.
-- No migra funciones ni borra datos. Organismo por parámetro, una transacción.
\if :{?organismo}
\else
DO $falta$ BEGIN RAISE EXCEPTION 'organización RPT 2026: falta -v organismo=<organismo_ref>' USING ERRCODE='22023'; END $falta$;
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='60s';
SET LOCAL lock_timeout='5s';
SET LOCAL timezone='UTC';
-- El bloqueo es la primera consulta: la instantánea SERIALIZABLE se toma con él.
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:datos:organizacion-rpt-2026:centro:520:v1',0));
SELECT pg_catalog.set_config('vec.organismo', :'organismo', true);

CREATE TEMP TABLE rpt_2026_fila(
 n integer PRIMARY KEY, codigo text NOT NULL, denominacion text NOT NULL,
 centro_codigo text NOT NULL, centro text NOT NULL, categoria text NOT NULL,
 grupo text NOT NULL, nivel integer, dotacion integer NOT NULL, provision text NOT NULL
) ON COMMIT DROP;
INSERT INTO rpt_2026_fila VALUES
 (33,'9','ADMINISTRATIVO','520','TRANSFORMACIÓN DIGITAL','administrativo','C1',17,1,'C');
GRANT SELECT ON rpt_2026_fila TO vec_personal_propietario;

DO $carga$
DECLARE
 org text:=pg_catalog.current_setting('vec.organismo');
 fuente_rpt constant text:='fuente:rpt:diputacion-granada:2026-05-07';
 huella_rpt constant text:='b0685beb5c02b8a30d5e0d6d3d9bceca11ddf76ad4987f4bcb1aa60ac7ebe9a8';
 acto_rpt constant text:='acto:publicacion-rpt:diputacion-granada:2026-05-07';
 fuente_pla constant text:='fuente:plantilla-vec:rpt-2026:centro:520';
 acto_pla constant text:='acto:carga-plantilla-vec:rpt-2026:centro:520';
 huella_pla_esperada constant text:='018822e9d4eb947b6b643aae122a04dd184346801c5ebdb23256e7196c3e8fdf';
 huella_pla text; ns text; v_rpt uuid; v_pla uuid; v_cob uuid; ahora timestamptz(6);
 previas integer;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'organización RPT 2026: ejecútalo como superusuario' USING ERRCODE='42501'; END IF;
 IF org !~ '^[a-z][a-z0-9_:-]{2,127}$' THEN
  RAISE EXCEPTION 'organización RPT 2026: organismo inválido' USING ERRCODE='22023'; END IF;
 IF pg_catalog.to_regclass('vec_personal.cobertura_ocupaciones_historia') IS NULL
  OR pg_catalog.strpos(pg_catalog.pg_get_functiondef('vec_personal.validar_revision_cobertura_ocupaciones_v1()'::regprocedure),
     'p.fuente_ref=NEW.fuente_ref AND p.huella_fuente_sha256=NEW.fuente_huella_sha256')=0 THEN
  RAISE EXCEPTION 'organización RPT 2026: falta Personal40' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM rpt_2026_fila)<>1 THEN
  RAISE EXCEPTION 'organización RPT 2026: material incompleto' USING ERRCODE='22023'; END IF;
 -- La huella de la plantilla es la de sus plazas en forma canónica.
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.string_agg(
   pg_catalog.concat_ws('|','2026/'||pg_catalog.lpad(n::text,4,'0'),codigo,categoria,grupo,'unidad:centro:'||pg_catalog.lower(centro_codigo)),
   E'\n' ORDER BY n),'UTF8')),'hex') INTO huella_pla FROM rpt_2026_fila;
 IF huella_pla IS DISTINCT FROM huella_pla_esperada THEN
  RAISE EXCEPTION 'organización RPT 2026: huella de plantilla % distinta de la fijada', huella_pla USING ERRCODE='22023'; END IF;
 ns:='vec:personal:organizacion-rpt-2026:v1:'||org||':';
 v_rpt:=pg_catalog.md5(ns||'rpt')::uuid; v_pla:=pg_catalog.md5(ns||'plantilla')::uuid; v_cob:=pg_catalog.md5(ns||'cobertura')::uuid;

 SELECT count(*) INTO previas FROM vec_personal.plaza_plantilla_historia WHERE plaza_ref=pg_catalog.md5(ns||'plaza:9')::uuid;
 IF previas>0 THEN
  IF NOT EXISTS(SELECT 1 FROM vec_personal.version_rpt_historia WHERE version_ref=v_rpt AND revision=1
      AND organismo_ref=org AND huella_fuente_sha256=huella_rpt)
   OR NOT EXISTS(SELECT 1 FROM vec_personal.version_plantilla_historia WHERE version_ref=v_pla AND revision=1
      AND organismo_ref=org AND estado='publicada')
   OR (SELECT count(*) FROM vec_personal.plaza_plantilla_historia WHERE plantilla_version_ref=v_pla AND revision=1 AND codigo_plaza_fuente='2026/0033' AND huella_fuente_sha256=huella_pla)<>1
   OR (SELECT count(*) FROM vec_personal.puesto_tipo_historia WHERE rpt_version_ref=v_rpt AND revision=1 AND codigo_fila_fuente='9')<>1
   OR NOT EXISTS(SELECT 1 FROM vec_personal.dotacion_rpt_historia
        WHERE dotacion_ref=pg_catalog.md5(ns||'dotacion:9')::uuid AND revision=1
          AND organismo_ref=org AND tipo_ref=pg_catalog.md5(ns||'tipo:9')::uuid
          AND tipo_revision=1 AND unidad_ref='unidad:centro:520' AND cantidad=1)
   OR NOT EXISTS(SELECT 1 FROM vec_personal.puesto_rpt_historia
        WHERE puesto_ref=pg_catalog.md5(ns||'puesto:9')::uuid AND revision=1
          AND organismo_ref=org AND tipo_ref=pg_catalog.md5(ns||'tipo:9')::uuid
          AND tipo_revision=1 AND unidad_ref='unidad:centro:520'
          AND codigo_puesto_fuente='9/01' AND estado_estructural='vigente' AND NOT retirado)
   OR (SELECT count(*) FROM vec_personal.vinculo_plaza_puesto_historia v JOIN vec_personal.plaza_plantilla_historia pl
        ON pl.plaza_ref=v.plaza_ref AND pl.revision=v.plaza_revision
        WHERE pl.plantilla_version_ref=v_pla AND pl.codigo_plaza_fuente='2026/0033'
          AND v.vinculo_ref=pg_catalog.md5(ns||'vinculo:9')::uuid AND v.revision=1
          AND v.plaza_ref=pg_catalog.md5(ns||'plaza:9')::uuid
          AND v.puesto_ref=pg_catalog.md5(ns||'puesto:9')::uuid AND v.puesto_revision=1
          AND v.organismo_ref=org AND v.estado='confirmado' AND NOT v.retirado)<>1
   OR NOT EXISTS(SELECT 1 FROM vec_personal.cobertura_ocupaciones_historia WHERE cobertura_ref=v_cob AND revision=1
        AND estado='completa' AND plantilla_version_ref=v_pla AND plantilla_revision=1) THEN
   RAISE EXCEPTION 'datos centro 520: PARO clave=carga_previa actual=incompleta_o_distinta esperado=cinco_filas_del_guion' USING ERRCODE='55000'; END IF;
  RAISE NOTICE 'organización RPT 2026: ya estaba cargada; sin cambios';
  RETURN;
 END IF;
 -- Reutiliza la plantilla y cobertura publicadas; no crea una segunda plantilla.
 IF NOT EXISTS(SELECT 1 FROM vec_personal.version_rpt_historia WHERE version_ref=v_rpt AND revision=1 AND organismo_ref=org AND estado='publicada' AND huella_fuente_sha256=huella_rpt) THEN
  RAISE EXCEPTION 'datos centro 520: falta la RPT publicada esperada, organismo=%', org; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_personal.version_plantilla_historia WHERE version_ref=v_pla AND revision=1
      AND organismo_ref=org AND estado='publicada' AND NOT retirado)
   OR NOT EXISTS(SELECT 1 FROM vec_personal.cobertura_ocupaciones_historia WHERE cobertura_ref=v_cob AND revision=1
      AND organismo_ref=org AND plantilla_version_ref=v_pla AND plantilla_revision=1 AND estado='completa') THEN
  RAISE EXCEPTION 'datos centro 520: PARO clave=plantilla_y_cobertura actual=ausente esperado=publicada_y_completa'; END IF;
 SET LOCAL ROLE vec_personal_propietario;
 ahora:=pg_catalog.clock_timestamp();
 INSERT INTO vec_personal.puesto_tipo_historia(tipo_ref,revision,organismo_ref,unidad_ref,rpt_version_ref,rpt_revision,
   codigo_fila_fuente,denominacion,clasificacion_ref,forma_provision_ref,nivel_destino,vigente_desde,conocido_desde,
   fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'tipo:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),v_rpt,1,
   codigo,denominacion,'categoria:rpt:'||categoria,provision,nivel,'2026-01-01',ahora,fuente_rpt,acto_rpt,huella_rpt
 FROM rpt_2026_fila;
 INSERT INTO vec_personal.dotacion_rpt_historia(dotacion_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,
   cantidad,reconciliacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'dotacion:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),
   pg_catalog.md5(ns||'tipo:'||codigo)::uuid,1,dotacion,'parcial',
   '2026-01-01',ahora,fuente_rpt,acto_rpt,huella_rpt
 FROM rpt_2026_fila;
 INSERT INTO vec_personal.puesto_rpt_historia(puesto_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,
   codigo_puesto_fuente,estado_estructural,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'puesto:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),
   pg_catalog.md5(ns||'tipo:'||codigo)::uuid,1,codigo||'/01','vigente','2026-01-01',ahora,fuente_rpt,acto_rpt,huella_rpt
 FROM rpt_2026_fila;
 INSERT INTO vec_personal.plaza_plantilla_historia(plaza_ref,revision,organismo_ref,unidad_ref,plantilla_version_ref,
   plantilla_revision,codigo_plaza_fuente,clasificacion_ref,estado_estructural,dotacion_presupuestaria,vigente_desde,
   conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'plaza:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),v_pla,1,
   '2026/'||pg_catalog.lpad(n::text,4,'0'),'categoria:rpt:'||categoria,'vigente','acreditada','2026-01-01',ahora,
   fuente_pla,acto_pla,huella_pla
 FROM rpt_2026_fila;
 INSERT INTO vec_personal.vinculo_plaza_puesto_historia(vinculo_ref,revision,organismo_ref,unidad_ref,plaza_ref,
   plaza_revision,puesto_ref,puesto_revision,estado,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'vinculo:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),
   pg_catalog.md5(ns||'plaza:'||codigo)::uuid,1,pg_catalog.md5(ns||'puesto:'||codigo)::uuid,1,'confirmado',
   '2026-01-01',ahora,fuente_pla,acto_pla,huella_pla
 FROM rpt_2026_fila;
 RESET ROLE;
 RAISE NOTICE 'organización RPT 2026: añadidos 1 puesto tipo, 1 dotación, 1 puesto, 1 plaza y 1 vínculo; plantilla y cobertura previas conservadas';
END $carga$;
COMMIT;
