\set ON_ERROR_STOP on
-- AUT36: una concesión real de lectura técnica, fijada por catálogo de datos.
-- No abre rutas, no asigna personas y no añade facultades de gestión de perfiles.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.catalogo_accion_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.catalogo_fijo_sistemas_admin_v1') IS NOT NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id='operador_plataforma')
 THEN RAISE EXCEPTION 'AUT36: PARO clave=preimagen actual=incompatible esperado=AUT33_sin_Sistemas' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
-- Amplía sólo los dos dominios categoriales; todas las filas previas se conservan.
DO $dominios$
DECLARE tabla regclass;columna smallint;nombre text;definicion text;item record;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_autorizacion.perfil_fijo_categoria_nominal_v1','categoria_administrativa','aplicacion','sistemas','perfil_fijo_categoria_nominal_categoria_v2'),
  ('vec_autorizacion.catalogo_accion_nominal_v1','clase_control','administrador_aplicacion','administrador_sistemas','catalogo_accion_nominal_clase_control_v2')
 ) d(tabla,columna,anterior,nueva,constraint_nueva) LOOP
  tabla:=pg_catalog.to_regclass(item.tabla);
  SELECT attnum INTO STRICT columna FROM pg_catalog.pg_attribute WHERE attrelid=tabla AND attname=item.columna AND NOT attisdropped;
  SELECT c.conname,pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT nombre,definicion
  FROM pg_catalog.pg_constraint c WHERE c.conrelid=tabla AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[columna];
  IF definicion IS DISTINCT FROM pg_catalog.format('CHECK ((%I = %L::text))',item.columna,item.anterior)
  THEN RAISE EXCEPTION 'AUT36: PARO clave=dominio actual=incompatible esperado=CHECK_AUT33_exacta' USING ERRCODE='55000'; END IF;
  EXECUTE pg_catalog.format('ALTER TABLE %s DROP CONSTRAINT %I',tabla,nombre);
  EXECUTE pg_catalog.format('ALTER TABLE %s ADD CONSTRAINT %I CHECK(%I IN(%L,%L))',tabla,item.constraint_nueva,item.columna,item.anterior,item.nueva);
 END LOOP;
END $dominios$;
CREATE TABLE vec_autorizacion.catalogo_fijo_sistemas_admin_v1(
 catalogo_ref text PRIMARY KEY,
 version numeric(20,0) NOT NULL CHECK(version=1),
 material bytea NOT NULL,
 huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(material),'hex')),
 documento jsonb NOT NULL CHECK(documento=pg_catalog.convert_from(material,'UTF8')::jsonb),
 publicada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_autorizacion.catalogo_fijo_sistemas_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.catalogo_fijo_sistemas_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.catalogo_fijo_sistemas_admin_v1 FOR ALL TO vec_autorizacion_propietario
 USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.catalogo_fijo_sistemas_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.catalogo_fijo_sistemas_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.catalogo_fijo_sistemas_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.catalogo_fijo_sistemas_admin_v1 FROM PUBLIC;
DO $publicar$
DECLARE material text:=$catalogo${
  "esquema": "vec.administracion.acciones_sistemas.v1",
  "catalogo_ref": "catalogo:administracion:acciones_sistemas:v1",
  "version": 1,
  "perfil_fijo": {
    "rol_id": "operador_plataforma",
    "categoria_administrativa": "sistemas",
    "tipo_perfil": "fijo_sistema"
  },
  "acciones": [
    {
      "accion_ref": "accion:administracion.registros_tecnicos.consultar",
      "accion": "administracion.registros_tecnicos.consultar",
      "modulo_id": "administracion",
      "tipo_recurso": "registro_tecnico",
      "finalidades": [
        "operacion_tecnica"
      ],
      "garantia_minima": "alto",
      "campos_permitidos": [
        "codigo",
        "componente",
        "correlacion",
        "correlacion_ref",
        "entorno",
        "esquema",
        "etapa",
        "instante",
        "nivel",
        "recuento",
        "resultado",
        "severidad",
        "version_binario"
      ],
      "obligaciones": [],
      "dimensiones_ambito": [
        "organizacion_ref"
      ],
      "clase_control": "administrador_sistemas"
    }
  ],
  "familias": [
    "vec.incidencia_tecnica.v1",
    "vec.resultado_tecnico.v1"
  ],
  "campos_excluidos": [
    "identidad",
    "mensaje",
    "paths",
    "recurso",
    "sql",
    "texto_error_libre"
  ],
  "recurso": {
    "referencia": "dataset_opaco",
    "propiedad": [
      "organizacion_ref",
      "proceso"
    ],
    "resolucion": "configuracion_privada_propietaria",
    "rutas_aportadas_cliente": false,
    "organizacion_aportada_cliente": false
  },
  "filtros": [
    "desde",
    "hasta",
    "familia",
    "codigo",
    "resultado",
    "nivel",
    "severidad",
    "componente",
    "etapa",
    "entorno",
    "correlacion"
  ],
  "fechas_filtro": "UTC",
  "paginacion": {
    "limites_configuracion_ejemplo": {
      "por_defecto": 32,
      "maximo": 100
    },
    "cursor": "opaco",
    "cursor_vinculado": [
      "dataset",
      "fuente",
      "filtros",
      "organizacion_ref"
    ]
  },
  "operaciones": [
    "consultar"
  ],
  "operaciones_excluidas": [
    "buscar_texto_libre",
    "exportar",
    "escribir",
    "gestionar_roles",
    "consultar_auditoria_nominal"
  ],
  "precondiciones_lector": [
    "sesion_sistemas",
    "decision_V3_exacta",
    "lector_propietario",
    "auditoria_transaccional",
    "confirmacion_commit"
  ],
  "auditoria_transaccional": "invariante_kernel",
  "fuente_codigo": [
    {
      "archivo": "internal/vec/adapters/observabilidad/recolector.go",
      "huella_sha256": "a0495cbe9c57f3f224d68d8a934c9df970f1f95b6704662b87058ed4cf2c1201"
    },
    {
      "archivo": "internal/vec/adapters/observabilidad/resultado_jsonl.go",
      "huella_sha256": "c55a7c7a6ea8bd51256517e1c1c1a687a0ea6368e56038e9678be4eb5a89ee3c"
    }
  ]
}
$catalogo$;c jsonb;accion jsonb;
 d jsonb;control jsonb;ahora timestamptz(6):=pg_catalog.clock_timestamp();fecha text;sha text;
 fuente_sha constant text:='c079851725613a0aa019addb2fdc803c58f7a10b3c889e4fa205227409c31961';
BEGIN
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(material,'UTF8')),'hex') IS DISTINCT FROM fuente_sha
 THEN RAISE EXCEPTION 'AUT36: PARO clave=catalogo_datos actual=bytes_divergentes esperado=SHA_publicada' USING ERRCODE='55000'; END IF;
 c:=material::jsonb;
 accion:=c->'acciones'->0;
 IF pg_catalog.jsonb_array_length(c->'acciones')<>1
 OR accion->>'accion' IS DISTINCT FROM 'administracion.registros_tecnicos.consultar'
 OR accion->>'modulo_id' IS DISTINCT FROM 'administracion' OR accion->>'tipo_recurso' IS DISTINCT FROM 'registro_tecnico'
 OR accion->'finalidades' IS DISTINCT FROM '["operacion_tecnica"]'::jsonb
 OR accion->'campos_permitidos' IS DISTINCT FROM '["codigo","componente","correlacion","correlacion_ref","entorno","esquema","etapa","instante","nivel","recuento","resultado","severidad","version_binario"]'::jsonb
 OR accion->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR accion->'dimensiones_ambito' IS DISTINCT FROM '["organizacion_ref"]'::jsonb
 OR accion->>'clase_control' IS DISTINCT FROM 'administrador_sistemas'
 THEN RAISE EXCEPTION 'AUT36: PARO clave=concesion actual=divergente esperado=lectura_tecnica_cerrada' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_autorizacion.catalogo_fijo_sistemas_admin_v1 VALUES(c->>'catalogo_ref',1,pg_catalog.convert_to(material,'UTF8'),fuente_sha,c,ahora);
 fecha:=pg_catalog.to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 d:=pg_catalog.jsonb_build_object('rol_id',c#>>'{perfil_fijo,rol_id}','version',1,'nombre',c#>>'{perfil_fijo,rol_id}',
  'estado','publicada','concesiones',pg_catalog.jsonb_build_array(accion-ARRAY['accion_ref','dimensiones_ambito','clase_control']),
  'publicada_por','migracion:autorizacion:000036','publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
 IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT36: PARO clave=rol actual=invalido esperado=concesion_positiva_real' USING ERRCODE='23514'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex');
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 VALUES('rol:operador_plataforma:v1',c#>>'{perfil_fijo,rol_id}',1,sha,ahora,d);
 control:=pg_catalog.jsonb_build_object('version_rol_ref','rol:operador_plataforma:v1','revision',1,'estado','habilitada',
  'actualizado_por','migracion:autorizacion:000036','actualizado_en',fecha);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento,creada_en)
 VALUES('rol:operador_plataforma:v1',1,'habilitada',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex'),ahora,control,ahora);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES('rol:operador_plataforma:v1',1,ahora,'migracion:autorizacion:000036','migracion:autorizacion:000036');
 INSERT INTO vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase,huella_sha256) VALUES('rol:operador_plataforma:v1','administrador',sha);
 INSERT INTO vec_autorizacion.perfil_fijo_categoria_nominal_v1 VALUES('rol:operador_plataforma:v1',sha,'sistemas','fijo_sistema',c->>'catalogo_ref',1,fuente_sha,ahora);
 INSERT INTO vec_autorizacion.catalogo_accion_nominal_v1 VALUES(accion->>'accion_ref',1,c->>'catalogo_ref',1,fuente_sha,
  'rol:operador_plataforma:v1',accion-ARRAY['accion_ref','dimensiones_ambito','clase_control'],
  accion->'dimensiones_ambito','administrador_sistemas',ahora,NULL);
END $publicar$;
COMMIT;
