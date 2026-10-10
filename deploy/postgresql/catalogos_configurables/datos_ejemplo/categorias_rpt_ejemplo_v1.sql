\set ON_ERROR_STOP on
-- Categorías RPT de EJEMPLO para el catálogo `categorias_rpt` del módulo
-- `personal` (el que espera la configuración B2 de incorporación a Personal).
--
-- Es un paquete de ejemplo retirable: 16 categorías tomadas de la RPT pública
-- de la Diputación de Granada (revisión 2026-05-07, data/catalogos/rpt/).
-- NO es la publicación jurídica de la RPT ni lleva aprobación de RRHH; así
-- consta en fuente_ref, aprobación y motivo del documento.
--
-- Usa la función de publicar existente (CC1), sin SQL nuevo. Solo la puede
-- ejecutar su propietario: lo lanza el superusuario de la base (postgres),
-- que lee las tablas del esquema y hace SET ROLE al propietario de AD3 V3.
--
-- Repetirlo no duplica nada: publicar devuelve el mismo recibo si la versión 1
-- ya existe con el mismo documento. Si existe con otro contenido, se para.
-- Para cambiar las categorías se publica otra versión (v2) con sus preimágenes;
-- este fichero no se edita.
--
-- El documento va en bytes canónicos del tipo Go CatalogoConfigurable
-- (json.Marshal), para que su huella coincida con la que calcula la aplicación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout = '30s';
SET LOCAL lock_timeout = '5s';

DO $previo$
DECLARE previa text;
BEGIN
 IF NOT (SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname = current_user) THEN
  RAISE EXCEPTION 'categorias_rpt ejemplo: ejecútalo como superusuario' USING ERRCODE = '42501';
 END IF;
 IF to_regprocedure('vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)') IS NULL THEN
  RAISE EXCEPTION 'categorias_rpt ejemplo: falta la autoridad de catálogos (CC1)' USING ERRCODE = '55000';
 END IF;
 SELECT huella_sha256 INTO previa FROM vec_catalogos_configurables.publicacion
  WHERE catalogo_id = 'categorias_rpt' AND version = 1;
 IF previa IS NOT NULL AND previa <> 'd0092b72fac479cd86d06008aacea794c9e2bb16d4b573ef3ad5fbd5ff344b9b' THEN
  RAISE EXCEPTION 'categorias_rpt ejemplo: ya hay una versión 1 distinta (%); no se toca', previa
   USING ERRCODE = '55000';
 END IF;
 IF previa IS NOT NULL THEN
  RAISE NOTICE 'categorias_rpt ejemplo: la versión 1 ya estaba publicada; se repite sin cambios';
 END IF;
END $previo$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $publicar$
DECLARE
 doc constant text := $doc${"id":"categorias_rpt","version":1,"revision":1,"modulo_id":"personal","nombre":"Categorías RPT (paquete de ejemplo)","descripcion":"Paquete de ejemplo retirable con 16 categorías tomadas de la RPT pública de la Diputación de Granada (revisión 2026-05-07). Sirve para desarrollo y presentación. No es la publicación jurídica de la RPT.","fuente_ref":"ejemplo:rpt-publica-diputacion-granada-2026-05-07:no-es-publicacion-juridica","motivo_creacion":"Paquete de ejemplo para que Contratación temporal pueda vincular expedientes a una categoría RPT publicada mientras RRHH no aprueba la fuente oficial.","entradas":[{"clave":"categoria:rpt:administrativo","etiqueta":"Administrativo","orden":10,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"ADMINISTRATIVO","escalas":"AG","etiqueta_en":"Administrative officer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C1,C2","paquete":"ejemplo"}},{"clave":"categoria:rpt:auxiliar-administrativo","etiqueta":"Auxiliar administrativo","orden":20,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"AUXILIAR ADMINISTRATIVO","escalas":"AG","etiqueta_en":"Administrative assistant","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C2","paquete":"ejemplo"}},{"clave":"categoria:rpt:tecnico-de-gestion","etiqueta":"Técnico de gestión","orden":30,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"TECNICO DE GESTION","escalas":"AG","etiqueta_en":"Management officer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}},{"clave":"categoria:rpt:tecnico-administracion-general","etiqueta":"Técnico de administración general","orden":40,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"TECNICO ADMINISTRACION GENERAL","escalas":"AE,AG","etiqueta_en":"General administration officer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A1","paquete":"ejemplo"}},{"clave":"categoria:rpt:trabajador-social","etiqueta":"Trabajador social","orden":50,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"TRABAJADOR SOCIAL","escalas":"AE","etiqueta_en":"Social worker","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}},{"clave":"categoria:rpt:educador","etiqueta":"Educador","orden":60,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"EDUCADOR","escalas":"AE","etiqueta_en":"Educator","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2,C1","paquete":"ejemplo"}},{"clave":"categoria:rpt:educador-social","etiqueta":"Educador social","orden":70,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"EDUCADOR SOCIAL","escalas":"AE","etiqueta_en":"Social educator","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}},{"clave":"categoria:rpt:psicologo","etiqueta":"Psicólogo","orden":80,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"PSICOLOGO","escalas":"AE","etiqueta_en":"Psychologist","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A1","paquete":"ejemplo"}},{"clave":"categoria:rpt:enfermera-o","etiqueta":"Enfermero o enfermera","orden":90,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"ENFERMERA/O","escalas":"AE","etiqueta_en":"Nurse","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}},{"clave":"categoria:rpt:ctpd-auxilar-de-enfermaria","etiqueta":"Auxiliar de enfermería","orden":100,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"CTPD/AUXILAR DE ENFERMARÍA","escalas":"AE","etiqueta_en":"Nursing assistant","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C1,C2","paquete":"ejemplo"}},{"clave":"categoria:rpt:operario","etiqueta":"Operario","orden":110,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"OPERARIO","escalas":"AE","etiqueta_en":"Operative","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"AP","paquete":"ejemplo"}},{"clave":"categoria:rpt:oficial-de-servicios-multiples","etiqueta":"Oficial de servicios múltiples","orden":120,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"OFICIAL DE SERVICIOS MULTIPLES","escalas":"AE","etiqueta_en":"Multi-service officer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C2","paquete":"ejemplo"}},{"clave":"categoria:rpt:auxiliar-servicios-generales","etiqueta":"Auxiliar de servicios generales","orden":130,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"AUXILIAR SERVICIOS GENERALES","escalas":"AE","etiqueta_en":"General services assistant","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C2","paquete":"ejemplo"}},{"clave":"categoria:rpt:encargado","etiqueta":"Encargado","orden":140,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"ENCARGADO","escalas":"AE","etiqueta_en":"Supervisor","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"C1","paquete":"ejemplo"}},{"clave":"categoria:rpt:analista-programador","etiqueta":"Analista programador","orden":150,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"ANALISTA-PROGRAMADOR","escalas":"AE","etiqueta_en":"Analyst programmer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}},{"clave":"categoria:rpt:arquitecto-tecnico","etiqueta":"Arquitecto técnico","orden":160,"vigente_desde":"2026-10-10T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z","atributos":{"denominacion_rpt":"ARQUITECTO TECNICO","escalas":"AE","etiqueta_en":"Building engineer","fuente_rpt":"rpt-diputacion-granada-2026-05-07","grupos":"A2","paquete":"ejemplo"}}],"estado":"publicado","creado_por":"sistema:ejemplo:preparador-categorias-rpt","creado_en":"2026-10-10T00:00:00Z","ultima_modificacion_en":"0001-01-01T00:00:00Z","publicado_por":"sistema:ejemplo:publicador-categorias-rpt","publicado_en":"2026-10-10T00:00:00Z","aprobacion_ref":"ejemplo:sin-aprobacion-juridica","motivo_publicacion":"Publicación de ejemplo, retirable y sustituible por otra versión. No es la publicación jurídica de la RPT.","retirado_en":"0001-01-01T00:00:00Z"}$doc$;
 huella constant text := 'd0092b72fac479cd86d06008aacea794c9e2bb16d4b573ef3ad5fbd5ff344b9b';
 vacia constant text := encode(sha256(convert_to('{}', 'UTF8')), 'hex');
 recibo text;
BEGIN
 IF encode(sha256(convert_to(doc, 'UTF8')), 'hex') <> huella THEN
  RAISE EXCEPTION 'categorias_rpt ejemplo: el documento no coincide con su huella' USING ERRCODE = '22023';
 END IF;
 recibo := vec_catalogos_configurables.publicar(
  'categorias_rpt', 1, huella, doc, '{}'::jsonb, vacia,
  'ejemplo:aprobacion-a:sin-valor-juridico', 'ejemplo:aprobacion-b:sin-valor-juridico',
  'sistema:ejemplo:publicador-categorias-rpt', 'ejemplo:decision:categorias_rpt:v1',
  'recibo:ejemplo:categorias_rpt:v1', 'motivos_catalogos:1:paquete_ejemplo');
 IF recibo IS DISTINCT FROM 'recibo:ejemplo:categorias_rpt:v1' THEN
  RAISE EXCEPTION 'categorias_rpt ejemplo: recibo inesperado' USING ERRCODE = '55000';
 END IF;
END $publicar$;
RESET ROLE;

DO $comprobar$
DECLARE n integer; lista jsonb;
BEGIN
 SELECT count(*) INTO n FROM vec_catalogos_configurables.publicacion
  WHERE catalogo_id = 'categorias_rpt' AND version = 1 AND huella_sha256 = 'd0092b72fac479cd86d06008aacea794c9e2bb16d4b573ef3ad5fbd5ff344b9b'
    AND recibo_ref = 'recibo:ejemplo:categorias_rpt:v1';
 IF n <> 1 THEN RAISE EXCEPTION 'categorias_rpt ejemplo: publicación ausente' USING ERRCODE = '55000'; END IF;
 SELECT count(*) INTO n FROM vec_catalogos_configurables.entrada_publicada
  WHERE catalogo_id = 'categorias_rpt' AND version = 1;
 IF n <> 16 THEN RAISE EXCEPTION 'categorias_rpt ejemplo: % entradas, se esperaban 16', n USING ERRCODE = '55000'; END IF;
 SELECT count(*) INTO n FROM vec_catalogos_configurables.categoria_control c
   JOIN vec_catalogos_configurables.entrada_publicada e
     ON e.catalogo_id = 'categorias_rpt' AND e.version = 1 AND e.categoria_id = c.categoria_id
  WHERE c.catalogo_id = 'categorias_rpt';
 IF n <> 16 THEN RAISE EXCEPTION 'categorias_rpt ejemplo: control de categorías incompleto' USING ERRCODE = '55000'; END IF;
 lista := vec_catalogos_configurables.listar_habilitadas('categorias_rpt', NULL, 100);
 RAISE NOTICE 'categorias_rpt ejemplo: % categorías habilitadas para uso nuevo',
  jsonb_array_length(lista#>'{datos,items}');
END $comprobar$;
COMMIT;
