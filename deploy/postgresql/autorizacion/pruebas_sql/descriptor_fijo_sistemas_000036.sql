\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE r record;m record;c record;cat record;esperadas jsonb;
BEGIN
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:operador_plataforma:v1';
 SELECT * INTO STRICT m FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref=r.version_rol_ref;
 SELECT * INTO STRICT c FROM vec_autorizacion.catalogo_accion_nominal_v1 WHERE version_rol_ref=r.version_rol_ref;
 SELECT * INTO STRICT cat FROM vec_autorizacion.catalogo_fijo_sistemas_admin_v1 WHERE catalogo_ref=m.fuente_ref;
 esperadas:='["codigo","componente","correlacion","correlacion_ref","entorno","esquema","etapa","instante","nivel","recuento","resultado","severidad","version_binario"]'::jsonb;
 IF r.documento->>'estado' IS DISTINCT FROM 'publicada' OR jsonb_array_length(r.documento->'concesiones')<>1
 OR r.documento#>>'{concesiones,0,accion}' IS DISTINCT FROM 'administracion.registros_tecnicos.consultar'
 OR r.documento#>'{concesiones,0,campos_permitidos}' IS DISTINCT FROM esperadas
 OR m.categoria_administrativa IS DISTINCT FROM 'sistemas' OR m.tipo_perfil IS DISTINCT FROM 'fijo_sistema'
 OR m.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR m.fuente_huella_sha256 IS DISTINCT FROM cat.huella_sha256
 OR c.fuente_huella_sha256 IS DISTINCT FROM cat.huella_sha256
 OR c.concesion IS DISTINCT FROM r.documento->'concesiones'->0
 OR c.dimensiones_ambito IS DISTINCT FROM '["organizacion_ref"]'::jsonb
 OR c.clase_control IS DISTINCT FROM 'administrador_sistemas'
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE version_rol_ref=r.version_rol_ref)
 THEN RAISE EXCEPTION 'AUT36 prueba: descriptor concede fuera de lectura técnica o crea asignaciones'; END IF;
 IF cat.huella_sha256 IS DISTINCT FROM 'c079851725613a0aa019addb2fdc803c58f7a10b3c889e4fa205227409c31961'
 THEN RAISE EXCEPTION 'AUT36 prueba: catálogo diverge de datos publicados'; END IF;
END $prueba$;
ROLLBACK;
