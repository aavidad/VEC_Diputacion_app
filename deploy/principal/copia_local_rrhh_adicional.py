"""Semilla declarada de la segunda persona técnica en una copia de desarrollo."""

import hashlib
import json
import os
import re
import secrets
from pathlib import Path

from preparar_rrhh_adicional_copia import preparar


def preparar_copia(copia):
    from copia_local import guardar, sha256

    estado = copia.leer()
    if estado['preparada'] or not copia.existe():
        raise ValueError('La segunda persona requiere una copia nueva, antes del arranque')
    sujeto = 'desarrollo:rrhh:adicional:' + secrets.token_hex(16)
    manifiesto = preparar(copia.material, sujeto, 'Antonio Ruiz Lozano')
    persona, perfil = manifiesto['persona_ref'], manifiesto['perfil_ref']
    if any(not re.fullmatch(prefijo + '[a-z0-9]{32}', valor)
           for prefijo, valor in [('per_', persona), ('prf_', perfil)]):
        raise ValueError('Referencias del segundo técnico incompatibles')
    asignacion = 'asg_' + secrets.token_hex(16)
    semilla = {'tipo': 'semilla_sintetica_de_copia', 'gobierno_real_acreditado': False,
               'persona_ref': persona, 'perfil_ref': perfil, 'asignacion_id': asignacion,
               'rol_ref': 'rol:seguimiento_cese_ct_desarrollo:v2',
               'organizacion_ref': 'organizacion:desarrollo:dipgra'}
    # Retenido antes del efecto; crear nunca reejecuta una instancia existente.
    ruta = copia.raiz / 'rrhh-adicional-semilla.json'
    with ruta.open('x') as salida:
        json.dump(semilla, salida, ensure_ascii=False, indent=2)
        salida.write('\n')
        salida.flush()
        os.fsync(salida.fileno())
    ruta.chmod(0o600)
    # Gemela de la asignación operativa de la técnica original: mismo emisor y
    # mismo acto del circuito de desarrollo, para que cada operación pueda
    # ceñirla al expediente exacto como hace con ella.
    sql = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $semilla$
DECLARE r record; c record; doc jsonb; ahora timestamptz := clock_timestamp();
BEGIN
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:seguimiento_cese_ct_desarrollo:v2';
 SELECT v.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual a
  JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision)
  WHERE a.version_rol_ref=r.version_rol_ref;
 IF r.documento->>'estado' IS DISTINCT FROM 'publicada' OR c.estado IS DISTINCT FROM 'habilitada'
  OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=r.rol_id AND version>r.version)
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') p
   WHERE p->>'accion'='contratacion_temporal.incorporacion.no_incorporacion'
    AND p->>'modulo_id'='contratacion_temporal'
    AND p->>'tipo_recurso'='no_incorporacion_contratacion_temporal'
    AND p->'finalidades'='["registrar_no_incorporacion_contratacion_temporal"]'::jsonb
    AND p->>'garantia_minima'='alto')
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref='{perfil}')
 THEN RAISE EXCEPTION 'semilla segundo tecnico: preimagen incompatible' USING ERRCODE='55000'; END IF;
 doc:=jsonb_build_object('asignacion_id','{asignacion}','version',1,
  'perfil_activo_ref','{perfil}','principal_id','{persona}','version_rol_ref',r.version_rol_ref,
  'estado','activa','ambitos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref',
   'valores',jsonb_build_array('organizacion:desarrollo:dipgra'))),
  'emitida_por','identidad:desarrollo:no-autoritativa',
  'emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',to_char((ahora+interval '89 days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revocada_en','0001-01-01T00:00:00Z');
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:{asignacion}:v1','{asignacion}',1,'{perfil}','{persona}',r.version_rol_ref,
  encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),ahora,doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
 VALUES('{perfil}','asignacion:{asignacion}:v1',ahora,'identidad:desarrollo:no-autoritativa','acto:ct:desarrollo:asignacion:v1');
END $semilla$;
COMMIT;
"""
    copia.sql_bytes(sql.encode(), 'rrhh-adicional-semilla.log')
    fuentes = {
        'VEC_CT_REGLAS_SOURCE_PATH': 'data/catalogos/contratacion-temporal/reglas.v3.json',
        'VEC_CT_CAUSAS_CESE_SOURCE_PATH': 'data/demo/reglas/ct_causas_cese.demo.json',
        'VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH': 'data/demo/reglas/ct_motivos_rectificacion.demo.json',
    }
    entorno = copia.raiz / 'entorno_modulos.json'
    if sha256(entorno) != estado['entorno_modulos_sha256']:
        raise ValueError('El entorno de módulos cambió durante la semilla')
    nuevo = json.loads(entorno.read_text())
    for variable, relativa in fuentes.items():
        fuente = Path(estado['fuente']) / relativa
        if not fuente.is_file():
            raise ValueError('Falta el catálogo versionado de seguimiento')
        nuevo[variable] = str(fuente)
    nuevo.update(VEC_CT_SEGUIMIENTO_CESE_ENABLED='true',
                 VEC_CT_INCORPORACION_ACREDITADA_ENABLED='true')
    guardar(entorno, nuevo)
    estado['entorno_modulos_sha256'] = sha256(entorno)
    guardar(copia.raiz / 'estado.json', estado)
    return hashlib.sha256(ruta.read_bytes()).hexdigest()
