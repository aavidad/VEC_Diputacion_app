#!/usr/bin/env python3
"""Solo fixture sintético para el PG18 desechable de probar_pg18.sh."""

import hashlib
import json


def literal(value):
    return "'" + value.replace("'", "''") + "'"


def canon(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


issued = "2026-09-23T20:00:00Z"
expires = "2027-03-22T20:00:00Z"
profile = "prf_0123456789abcdef0123456789abcdef"
person = "per_0123456789abcdef0123456789abcdef"
employee = "emp_0123456789abcdef0123456789abcdef"
role_id = "dietas_r1d_provisional"
role_ref = f"rol:{role_id}:v1"
assignment_id = "dietas_r1d_0123456789abcdef"
assignment_ref = f"asignacion:{assignment_id}:v1"
actor = "desarrollo:dietas-r1d:provisional"
role = dict(rol_id=role_id, version=1, nombre="Dietas R1 sintético provisional",
            estado="publicada", concesiones=[dict(accion="dietas.borrador.propio.crear",
            modulo_id="dietas", tipo_recurso="comision_borrador",
            finalidades=["crear_borrador_propio"], garantia_minima="alto",
            campos_permitidos=["comision.estado"])], publicada_por=actor,
            publicada_en=issued, retirada_en="0001-01-01T00:00:00Z")
assignment = dict(asignacion_id=assignment_id, version=1, perfil_activo_ref=profile,
                  principal_id=person, version_rol_ref=role_ref, estado="activa",
                  ambitos=[{"clave": "empleado_ref", "valores": [employee]},
                           {"clave": "persona_ref", "valores": [person]}],
                  vigente_desde=issued, vigente_hasta=expires,
                  emitida_por=actor, emitida_en=issued,
                  revocada_en="0001-01-01T00:00:00Z")
hash_assignment = hashlib.sha256(canon(assignment).encode()).hexdigest()

groups = ["vec_identidad_sesiones_v1_registrador", "vec_identidad_sesiones_v1_revalidador",
          "vec_contexto_actor_v1_runtime", "vec_autorizacion_motivos_evaluador",
          "vec_dietas_ejecutor"]
for group in groups:
    print(f"CREATE ROLE {group} NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;")
members = [
    ("registro_identidad", "vec_identidad_sesiones_v1_registrador"),
    ("revalidacion_identidad", "vec_identidad_sesiones_v1_revalidador"),
    ("contexto", "vec_contexto_actor_v1_runtime"),
    ("fuente_autorizacion", "vec_autorizacion_fuente"),
    ("registro_autorizacion", "vec_autorizacion_registro"),
    ("motivos", "vec_autorizacion_motivos_evaluador"),
    ("dietas", "vec_dietas_ejecutor"),
    ("personal", "vec_dietas_ejecutor"),
]
for name, group in members:
    login = "vec_dietas_r1d_" + name + "_desarrollo"
    print(f"CREATE ROLE {login} LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;")
    print(f"GRANT {group} TO {login} WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;")
    print(f"GRANT CONNECT ON DATABASE postgres TO {login};")
print("CREATE ROLE p6_ruta_intermedia NOLOGIN;")
print("CREATE ROLE p6_ct_shared_login LOGIN;")
print("GRANT vec_autorizacion_fuente TO p6_ruta_intermedia WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;")
print("GRANT p6_ruta_intermedia TO p6_ct_shared_login WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;")
print("BEGIN; SET LOCAL ROLE vec_autorizacion_propietario;")
print("INSERT INTO vec_autorizacion.version_rol (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES (")
print(f"{literal(role_ref)},{literal(role_id)},1,{literal('0'*64)},{literal(issued)}::timestamptz,{literal(canon(role))}::jsonb);")
print("INSERT INTO vec_autorizacion.asignacion_perfil (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento) VALUES (")
print(f"{literal(assignment_ref)},{literal(assignment_id)},1,{literal(profile)},{literal(person)},{literal(role_ref)},{literal(hash_assignment)},{literal(issued)}::timestamptz,{literal(canon(assignment))}::jsonb);")
print("INSERT INTO vec_autorizacion.asignacion_perfil_actual (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref) VALUES (")
print(f"{literal(profile)},{literal(assignment_ref)},{literal(issued)}::timestamptz,{literal(actor)},'acto:fixture:p6');")
print("COMMIT;")
print("CREATE SCHEMA vec_ct_sentinel; CREATE TABLE vec_ct_sentinel.control (n integer); INSERT INTO vec_ct_sentinel.control VALUES (7);")
print("CREATE SCHEMA vec_bolsa_sentinel; CREATE TABLE vec_bolsa_sentinel.control (n integer); INSERT INTO vec_bolsa_sentinel.control VALUES (11);")
