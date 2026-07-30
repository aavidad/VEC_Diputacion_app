#!/usr/bin/env bash

probar_venenos_acl_topologia() {
  local fachada=$1
  local login=$2
  local selector=$3
  local consumidor=$4
  local autenticacion=$5
  local sesion=$6
  local rol=
  local roles_acl=(
    PUBLIC
    "$login"
    "$selector"
    vec_contexto_actor_v1_runtime
    vec_identidad_sesiones_v1_revalidador
    vec_contratacion_temporal_ejecutor
  )

  paso 'ACL adicionales sobre la fachada: PUBLIC, login y runtimes'
  psql_admin <<'SQL'
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
SQL
  for rol in "${roles_acl[@]}"; do
    psql_valor "GRANT EXECUTE ON FUNCTION $fachada(text,text) TO $rol" \
      >/dev/null
    exigir_cero "$autenticacion" "$sesion" "ACL adicional para $rol"
    psql_valor "REVOKE EXECUTE ON FUNCTION $fachada(text,text) FROM $rol" \
      >/dev/null
    exigir_uno "$autenticacion" "$sesion" "ACL restaurada para $rol"
  done
  psql_valor 'DROP ROLE vec_contratacion_temporal_ejecutor' >/dev/null

  paso 'ACL de esquema, columna y funcion base no nominales'
  psql_valor 'GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO PUBLIC' \
    >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'USAGE de esquema para PUBLIC'
  psql_valor 'REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC' \
    >/dev/null

  psql_valor "GRANT CREATE ON SCHEMA vec_identidad_sesiones_v1
TO $consumidor" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'CREATE de esquema del consumidor'
  psql_valor "REVOKE CREATE ON SCHEMA vec_identidad_sesiones_v1
FROM $consumidor" >/dev/null

  psql_valor "GRANT SELECT(cuenta_ref)
ON vec_identidad_sesiones_v1.cuenta TO $consumidor" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'ACL de columna del consumidor'
  psql_valor "REVOKE SELECT(cuenta_ref)
ON vec_identidad_sesiones_v1.cuenta FROM $consumidor" >/dev/null

  psql_valor "GRANT EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)
TO $consumidor" >/dev/null
  exigir_cero "$autenticacion" "$sesion" \
    'ejecucion de la fachada base por el consumidor'
  psql_valor "REVOKE EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)
FROM $consumidor" >/dev/null
  exigir_uno "$autenticacion" "$sesion" 'ACL nominal restaurada'

  paso 'atributos hostiles de login, selector, consumidor y fachada'
  psql_valor "ALTER ROLE $login CREATEDB" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'LOGIN con CREATEDB'
  psql_valor "ALTER ROLE $login NOCREATEDB" >/dev/null

  psql_valor "ALTER ROLE $selector INHERIT" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'selector con INHERIT'
  psql_valor "ALTER ROLE $selector NOINHERIT" >/dev/null

  psql_valor "ALTER ROLE $selector CONNECTION LIMIT 0" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'selector con limite de conexion'
  psql_valor "ALTER ROLE $selector CONNECTION LIMIT -1" >/dev/null

  psql_valor "ALTER ROLE $consumidor INHERIT" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'consumidor con INHERIT'
  psql_valor "ALTER ROLE $consumidor NOINHERIT" >/dev/null

  psql_valor "ALTER FUNCTION $fachada(text,text) LEAKPROOF" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'fachada LEAKPROOF'
  psql_valor "ALTER FUNCTION $fachada(text,text) NOT LEAKPROOF" >/dev/null

  psql_valor "ALTER FUNCTION $fachada(text,text) PARALLEL SAFE" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'fachada paralela segura alterada'
  psql_valor "ALTER FUNCTION $fachada(text,text) PARALLEL UNSAFE" >/dev/null
  exigir_uno "$autenticacion" "$sesion" 'atributos nominales restaurados'

  paso 'opciones hostiles de la membresia unica'
  psql_valor "GRANT $selector TO $login
WITH ADMIN TRUE,INHERIT TRUE,SET FALSE" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'membresia con ADMIN'
  psql_valor "GRANT $selector TO $login
WITH ADMIN FALSE,INHERIT TRUE,SET FALSE" >/dev/null

  psql_valor "GRANT $selector TO $login
WITH ADMIN FALSE,INHERIT TRUE,SET TRUE" >/dev/null
  exigir_cero "$autenticacion" "$sesion" 'membresia con SET'
  psql_valor "GRANT $selector TO $login
WITH ADMIN FALSE,INHERIT TRUE,SET FALSE" >/dev/null

  psql_valor "GRANT $selector TO $login
WITH ADMIN FALSE,INHERIT FALSE,SET FALSE" >/dev/null
  esperar_fallo_actor \
    "SELECT count(*) FROM vec_contexto_actor_v1.c21b_proxy_identidad(
'$autenticacion','$sesion')" 'membresia sin INHERIT'
  psql_valor "GRANT $selector TO $login
WITH ADMIN FALSE,INHERIT TRUE,SET FALSE" >/dev/null
  exigir_uno "$autenticacion" "$sesion" 'opciones de membresia restauradas'
}
