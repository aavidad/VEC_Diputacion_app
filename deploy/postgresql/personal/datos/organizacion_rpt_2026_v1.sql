\set ON_ERROR_STOP on
-- Organización de Personal 2026 a partir de la RPT publicada de la Diputación
-- de Granada (revisión 2026-05-07, data/catalogos/rpt/v1.rpt-2026.json).
--
-- Carga en las tablas de historia de Personal10 una versión de RPT publicada,
-- una plantilla 2026 publicada, 32 puestos tipo (dos por cada categoría del
-- catálogo categorias_rpt v1), su dotación, un puesto y una plaza por puesto
-- tipo con su vínculo confirmado, y la cobertura de ocupaciones de esa
-- plantilla (Personal40). Con ello la consulta de vacantes y la selección de
-- plaza y puesto del plan de incorporación B2 tienen datos.
--
-- Lo ejecuta el superusuario de la base:
--   psql -X -d <base> -v organismo=<organismo_ref de personal-b2/servidor.json> -f organizacion_rpt_2026_v1.sql
-- Una transacción. Repetirlo no duplica nada. Si ya hay otra versión con la
-- misma referencia y distinto contenido, se para sin tocarla. Para cambiar
-- algo se publica otra versión (revisión nueva o version_ref nueva); este
-- fichero no se edita.
\if :{?organismo}
\else
\echo 'Falta -v organismo=<organismo_ref>'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='60s';
SET LOCAL lock_timeout='5s';
SET LOCAL timezone='UTC';
SELECT pg_catalog.set_config('vec.organismo', :'organismo', true);
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:datos:organizacion-rpt-2026:v1',0));

CREATE TEMP TABLE rpt_2026_fila(
 n integer PRIMARY KEY, codigo text NOT NULL, denominacion text NOT NULL,
 centro_codigo text NOT NULL, centro text NOT NULL, categoria text NOT NULL,
 grupo text NOT NULL, nivel integer, dotacion integer NOT NULL, provision text NOT NULL
) ON COMMIT DROP;
INSERT INTO rpt_2026_fila VALUES
 (1,'10','ADMINISTRATIVO','810A','GESTIÓN Y ADMINISTRACIÓN DE OBRAS PÚBLICAS Y VIVIENDA','administrativo','C1',17,1,'C'),
 (2,'11','ADMINISTRATIVO','712','CENTRO PROVINCIAL DE DROGODEPENDENCIAS','administrativo','C1',17,1,'C'),
 (3,'687','AUXILIAR ADMINISTRATIVO','711','SERVICIOS SOCIALES COMUNITARIOS','auxiliar-administrativo','C2',16,1,'2A'),
 (4,'77-102-001','AUXILIAR ADMINISTRATIVO','102','SECRETARÍA GENERAL','auxiliar-administrativo','C2',16,5,'C'),
 (5,'482','TECNICO DE GESTION','301','INTERVENCIÓN - CONTROL Y FISCALIZACIÓN','tecnico-de-gestion','A2',20,5,'C'),
 (6,'483-103-005','TECNICO DE GESTION','103','CONTRATACIÓN ADMINISTRATIVA - CENTRAL PROV. CONTRAT.','tecnico-de-gestion','A2',20,3,'C'),
 (7,'471','TECNICO ADMINISTRACION GENERAL','200','EMPLEO Y DESARROLLO PROVINCIAL','tecnico-administracion-general','A1',24,1,'C'),
 (8,'473-112-005','TECNICO ADMINISTRACION GENERAL','112','SERVICIO DEL CICLO URBANO DEL AGUA','tecnico-administracion-general','A1',24,1,'C'),
 (9,'521','TRABAJADOR SOCIAL','751','RESIDENCIA DE MAYORES "LA MILAGROSA"','trabajador-social','A2',20,2,'C'),
 (10,'522-752-001','TRABAJADOR SOCIAL','752','RESIDENCIA "RODRÍGUEZ PENALVA"','trabajador-social','A2',20,1,'C'),
 (11,'150','EDUCADOR','713','IGUALDAD Y JUVENTUD','educador','A2',20,1,'C'),
 (12,'152-711-001','EDUCADOR','711','SERVICIOS SOCIALES COMUNITARIOS','educador','A2',20,1,'C'),
 (13,'158-711-001','EDUCADOR SOCIAL','711','SERVICIOS SOCIALES COMUNITARIOS','educador-social','A2',20,1,'C'),
 (14,'705-713-002','EDUCADOR SOCIAL','713','IGUALDAD Y JUVENTUD','educador-social','A2',20,1,'C'),
 (15,'400-711-002','PSICOLOGO','711','SERVICIOS SOCIALES COMUNITARIOS','psicologo','A1',24,1,'C'),
 (16,'402-752-001','PSICOLOGO','752','RESIDENCIA "RODRÍGUEZ PENALVA"','psicologo','A1',24,1,'C'),
 (17,'1','ENFERMERA/O','754','SERVICIOS GENERALES DE LOS CENTROS SOCIALES','enfermera-o','A2',20,17,'C'),
 (18,'2','ENFERMERA/O','752','RESIDENCIA "RODRÍGUEZ PENALVA"','enfermera-o','A2',20,7,'C'),
 (19,'115','CUIDADOR TEC. PERSONAS DEPENDIENTES','755','CENTRO OCUPACIONAL "REINA SOFÍA"','ctpd-auxilar-de-enfermaria','C1',17,4,'C'),
 (20,'116-751-001','CUIDADOR TEC. PERSONAS DEPENDIENTES','751','RESIDENCIA DE MAYORES "LA MILAGROSA"','ctpd-auxilar-de-enfermaria','C1',17,45,'C'),
 (21,'376-600-001','OPERARIO','600','DEPORTES','operario','AP',13,3,'C'),
 (22,'376-752-003','OPERARIO','752','RESIDENCIA "RODRÍGUEZ PENALVA"','operario','AP',13,22,'C'),
 (23,'357','OFICIAL DE SERVICIOS MULTIPLES','530','RECURSOS HUMANOS','oficial-de-servicios-multiples','C2',16,1,'C'),
 (24,'358','OFICIAL DE SERVICIOS MULTIPLES','503','SUBALTERNOS Y COMUNICACIÓN','oficial-de-servicios-multiples','C2',16,1,'C'),
 (25,'52','AUXILIAR SERVICIOS GENERALES','503','SUBALTERNOS Y COMUNICACIÓN','auxiliar-servicios-generales','C2',16,2,'C'),
 (26,'53-530-003','AUXILIAR SERVICIOS GENERALES','530','RECURSOS HUMANOS','auxiliar-servicios-generales','C2',16,1,'C'),
 (27,'160','ENCARGADO','840','SERVICIO DE CARRETERAS','encargado','C1',17,1,'C'),
 (28,'162-210-002','ENCARGADO','210','CULTURA - SERVICIOS GENERALES','encargado','C1',17,1,'C'),
 (29,'21','ANALISTA-PROGRAMADOR','520','TRANSFORMACIÓN DIGITAL','analista-programador','A2',20,1,'C'),
 (30,'22-520-001','ANALISTA-PROGRAMADOR','520','TRANSFORMACIÓN DIGITAL','analista-programador','A2',20,2,'C'),
 (31,'31-810-001','ARQUITECTO TECNICO','810','SERVICIO DE PROYECTOS Y OBRAS','arquitecto-tecnico','A2',20,2,'C'),
 (32,'31-820-002','ARQUITECTO TECNICO','820','SERVICIO DE PLANIFICACIÓN','arquitecto-tecnico','A2',20,2,'C');
GRANT SELECT ON rpt_2026_fila TO vec_personal_propietario;

DO $carga$
DECLARE
 org text:=pg_catalog.current_setting('vec.organismo');
 fuente_rpt constant text:='fuente:rpt:diputacion-granada:2026-05-07';
 huella_rpt constant text:='b0685beb5c02b8a30d5e0d6d3d9bceca11ddf76ad4987f4bcb1aa60ac7ebe9a8';
 acto_rpt constant text:='acto:publicacion-rpt:diputacion-granada:2026-05-07';
 fuente_pla constant text:='fuente:plantilla:diputacion-granada:2026';
 acto_pla constant text:='acto:plantilla:diputacion-granada:2026';
 huella_pla_esperada constant text:='f5ec895dcd1d0282676e81668d4d09047bd200e1e2f29d232597140ee741f524';
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
 IF (SELECT count(*) FROM rpt_2026_fila)<>32 THEN
  RAISE EXCEPTION 'organización RPT 2026: material incompleto' USING ERRCODE='22023'; END IF;
 -- La huella de la plantilla es la de sus plazas en forma canónica.
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.string_agg(
   pg_catalog.concat_ws('|','2026/'||pg_catalog.lpad(n::text,4,'0'),codigo,categoria,grupo,'unidad:centro:'||pg_catalog.lower(centro_codigo)),
   E'\n' ORDER BY n),'UTF8')),'hex') INTO huella_pla FROM rpt_2026_fila;
 IF huella_pla IS DISTINCT FROM huella_pla_esperada THEN
  RAISE EXCEPTION 'organización RPT 2026: huella de plantilla % distinta de la fijada', huella_pla USING ERRCODE='22023'; END IF;
 ns:='vec:personal:organizacion-rpt-2026:v1:'||org||':';
 v_rpt:=pg_catalog.md5(ns||'rpt')::uuid; v_pla:=pg_catalog.md5(ns||'plantilla')::uuid; v_cob:=pg_catalog.md5(ns||'cobertura')::uuid;

 SELECT count(*) INTO previas FROM vec_personal.version_rpt_historia WHERE version_ref=v_rpt;
 IF previas>0 THEN
  IF NOT EXISTS(SELECT 1 FROM vec_personal.version_rpt_historia WHERE version_ref=v_rpt AND revision=1
      AND organismo_ref=org AND huella_fuente_sha256=huella_rpt)
   OR NOT EXISTS(SELECT 1 FROM vec_personal.version_plantilla_historia WHERE version_ref=v_pla AND revision=1
      AND organismo_ref=org AND huella_fuente_sha256=huella_pla)
   OR (SELECT count(*) FROM vec_personal.plaza_plantilla_historia WHERE plantilla_version_ref=v_pla AND revision=1)<>32
   OR NOT EXISTS(SELECT 1 FROM vec_personal.cobertura_ocupaciones_historia WHERE cobertura_ref=v_cob AND revision=1) THEN
   RAISE EXCEPTION 'organización RPT 2026: ya hay una carga distinta con estas referencias; no se toca' USING ERRCODE='55000'; END IF;
  RAISE NOTICE 'organización RPT 2026: ya estaba cargada; sin cambios';
  RETURN;
 END IF;
 -- Una sola RPT y una sola plantilla publicadas por organismo y fecha.
 IF EXISTS(SELECT 1 FROM vec_personal.version_rpt_historia WHERE organismo_ref=org)
  OR EXISTS(SELECT 1 FROM vec_personal.version_plantilla_historia WHERE organismo_ref=org) THEN
  RAISE EXCEPTION 'organización RPT 2026: el organismo ya tiene otra RPT o plantilla; publicar como versión nueva' USING ERRCODE='55000'; END IF;

 SET LOCAL ROLE vec_personal_propietario;
 ahora:=pg_catalog.clock_timestamp();
 INSERT INTO vec_personal.version_rpt_historia(version_ref,revision,organismo_ref,codigo_version_fuente,estado,
   aprobada_en,publicada_en,vigente_desde,conocido_desde,fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
 VALUES(v_rpt,1,org,'RPT 2026 rev. 2026-05-07','publicada','2026-05-07','2026-05-07','2026-01-01',ahora,
   fuente_rpt,'Relación de Puestos de Trabajo de la Diputación de Granada 2026, revisión 2026-05-07',acto_rpt,huella_rpt);
 INSERT INTO vec_personal.version_plantilla_historia(version_ref,revision,organismo_ref,ejercicio,codigo_version_fuente,estado,
   aprobada_en,publicada_en,vigente_desde,conocido_desde,fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
 VALUES(v_pla,1,org,2026,'Plantilla 2026','publicada','2026-01-01','2026-01-01','2026-01-01',ahora,
   fuente_pla,'Plantilla de personal de la Diputación de Granada 2026',acto_pla,huella_pla);
 INSERT INTO vec_personal.puesto_tipo_historia(tipo_ref,revision,organismo_ref,unidad_ref,rpt_version_ref,rpt_revision,
   codigo_fila_fuente,denominacion,clasificacion_ref,forma_provision_ref,nivel_destino,vigente_desde,conocido_desde,
   fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'tipo:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),v_rpt,1,
   codigo,denominacion,'categoria:rpt:'||categoria,provision,nivel,'2026-01-01',ahora,fuente_rpt,acto_rpt,huella_rpt
 FROM rpt_2026_fila;
 INSERT INTO vec_personal.dotacion_rpt_historia(dotacion_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,
   cantidad,reconciliacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 SELECT pg_catalog.md5(ns||'dotacion:'||codigo)::uuid,1,org,'unidad:centro:'||pg_catalog.lower(centro_codigo),
   pg_catalog.md5(ns||'tipo:'||codigo)::uuid,1,dotacion,CASE WHEN dotacion=1 THEN 'reconciliada' ELSE 'parcial' END,
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
 -- La misma fuente que publica la plantilla declara completas sus ocupaciones:
 -- al cargarla ninguna plaza está ocupada; las altas B2 posteriores las ocupan.
 INSERT INTO vec_personal.cobertura_ocupaciones_historia(cobertura_ref,revision,organismo_ref,plantilla_version_ref,
   plantilla_revision,estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,fuente_huella_sha256)
 VALUES(v_cob,1,org,v_pla,1,'completa','2026-01-01','2028-01-01',ahora,acto_pla,fuente_pla,1,huella_pla);
 RESET ROLE;
 RAISE NOTICE 'organización RPT 2026: cargadas 1 RPT, 1 plantilla, 32 puestos tipo, 32 puestos, 32 plazas, 32 vínculos y la cobertura';
END $carga$;
COMMIT;
