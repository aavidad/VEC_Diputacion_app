\set ON_ERROR_STOP on
-- Vector focal prospectivo AUT61. Sólo ejecutar después del UP nuevo en la
-- base desechable acordada; todos los casos terminan en ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
DECLARE canon text:=$json$[{"referencia":"accion:sintetica","version":1,"fuente_ref":"fuente:modulo","fuente_version":2,"concesion":{"accion":"sintetico.consultar","modulo_id":"sintetico","tipo_recurso":"expediente","finalidades":["revision"],"garantia_minima":"alto"},"dimensiones_ambito":["unidad"],"clase_control":"consulta_auditada","vigente_desde":"2026-10-01T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z"}]$json$;
 huella text:='5fdabf454a5f15834a80e70cef1aef929648805dc481119064206654bc4c94cb';
 entrada jsonb;fuente jsonb;paquete jsonb;otro jsonb;ordinaria jsonb;central jsonb;
 nuevo_sha text;rol_existente text;
BEGIN
 IF pg_catalog.octet_length(canon)<>390
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(canon,'UTF8')),'hex') IS DISTINCT FROM huella THEN
  RAISE EXCEPTION 'AUT61 prueba: vector Go divergente';END IF;
 entrada:=pg_catalog.jsonb_set((canon::jsonb)->0,'{fuente_huella_sha256}',pg_catalog.to_jsonb(huella));
 IF vec_autorizacion.canon_entradas_fuente_catalogo_acciones_v2(pg_catalog.jsonb_build_array(entrada)) IS DISTINCT FROM canon THEN
  RAISE EXCEPTION 'AUT61 prueba: recanon Go divergente';END IF;
 fuente:=pg_catalog.jsonb_build_object('modulo_id','sintetico','referencia','fuente:modulo',
  'version',2,'huella_sha256',huella,'entradas_canon',canon,'entradas',pg_catalog.jsonb_build_array(entrada));
 paquete:=pg_catalog.jsonb_build_object('esquema','vec.admin.catalogo-acciones.paquete.v2',
  'referencia','paquete:sintetico','version',1,'fuentes',pg_catalog.jsonb_build_array(fuente),'perfiles','[]'::jsonb);
 PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(paquete);

 -- Misma estructura JSON, bytes con espacio y SHA recalculado: el canon Go
 -- sigue siendo obligatorio; una huella autofabricada no lo reemplaza.
 nuevo_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(canon||' ','UTF8')),'hex');
 otro:=pg_catalog.jsonb_set(paquete,'{fuentes,0,entradas_canon}',pg_catalog.to_jsonb(canon||' '));
 otro:=pg_catalog.jsonb_set(otro,'{fuentes,0,huella_sha256}',pg_catalog.to_jsonb(nuevo_sha));
 otro:=pg_catalog.jsonb_set(otro,'{fuentes,0,entradas,0,fuente_huella_sha256}',pg_catalog.to_jsonb(nuevo_sha));
 BEGIN
  PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(otro);
  RAISE EXCEPTION 'AUT61 prueba: acepto canon con espacio';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 otro:=pg_catalog.jsonb_set(paquete,'{fuentes,0,huella_sha256}',pg_catalog.to_jsonb(pg_catalog.repeat('f',64)));
 BEGIN
  PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(otro);
  RAISE EXCEPTION 'AUT61 prueba: acepto huella de fuente inventada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 otro:=pg_catalog.jsonb_set(paquete,'{fuentes,0,entradas,0,concesion,accion}',pg_catalog.to_jsonb('sintetico.otro'::text));
 BEGIN
  PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(otro);
  RAISE EXCEPTION 'AUT61 prueba: acepto entrada alterada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 otro:=pg_catalog.jsonb_set(paquete,'{fuentes}',pg_catalog.jsonb_build_array(fuente,fuente));
 BEGIN
  PERFORM vec_autorizacion.exigir_fuentes_catalogo_acciones_admin_v2(otro);
  RAISE EXCEPTION 'AUT61 prueba: acepto fuente duplicada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 PERFORM vec_autorizacion.verificar_catalogo_central_rol_nuevo_v1();
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id='aut61_prueba_ordinaria') THEN
  RAISE EXCEPTION 'AUT61 prueba: RolID sintetico ocupado';END IF;
 ordinaria:=pg_catalog.jsonb_set(entrada,'{clase_control}',pg_catalog.to_jsonb('ordinario'::text));
 PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(ordinaria,'aut61_prueba_ordinaria','Rol de prueba');
 -- El cierre revalida este descriptor después de insertar la versión. La
 -- unicidad del RolID corresponde al aplicar, bajo su bloqueo de escritura.
 SELECT rol_id INTO rol_existente FROM vec_autorizacion.version_rol
  WHERE rol_id ~ '^[a-z][a-z0-9_]{2,63}$'
   AND rol_id NOT IN ('administracion_perfiles','operador_plataforma')
   AND rol_id !~ '(^candidato_|extern|^intervencion)'
  ORDER BY rol_id LIMIT 1;
 IF rol_existente IS NULL THEN RAISE EXCEPTION 'AUT61 prueba: falta RolID previo ordinario';END IF;
 PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(ordinaria,rol_existente,'Rol de prueba');
 BEGIN
  PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(entrada,'aut61_prueba_ordinaria','Rol de prueba');
  RAISE EXCEPTION 'AUT61 prueba: acepto clase desconocida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 otro:=pg_catalog.jsonb_set(ordinaria,'{concesion,modulo_id}',pg_catalog.to_jsonb('administracion'::text));
 otro:=pg_catalog.jsonb_set(otro,'{concesion,accion}',pg_catalog.to_jsonb('administracion.perfiles.definicion.aprobar'::text));
 BEGIN
  PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(otro,'aut61_prueba_ordinaria','Rol de prueba');
  RAISE EXCEPTION 'AUT61 prueba: acepto accion ADMIN caso 23';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 SELECT n.concesion INTO central FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  WHERE n.concesion->>'accion'='personal.cargo_competencial.publicar' LIMIT 1;
 IF central IS NULL THEN RAISE EXCEPTION 'AUT61 prueba: accion central de Personal ausente';END IF;
 otro:=pg_catalog.jsonb_set(ordinaria,'{concesion}',
  pg_catalog.jsonb_set(central,'{finalidades}','["revision"]'::jsonb));
 BEGIN
  PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(otro,'aut61_prueba_ordinaria','Rol de prueba');
  RAISE EXCEPTION 'AUT61 prueba: acepte accion central con finalidad alterada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;

 SELECT n.concesion INTO central FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  WHERE n.clase_control='administrador_sistemas' LIMIT 1;
 IF central IS NULL THEN RAISE EXCEPTION 'AUT61 prueba: accion Sistemas ausente';END IF;
 otro:=pg_catalog.jsonb_set(ordinaria,'{concesion}',central);
 BEGIN
  PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(otro,'aut61_prueba_ordinaria','Rol de prueba');
  RAISE EXCEPTION 'AUT61 prueba: acepte accion Sistemas';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
END $prueba$;
ROLLBACK;
