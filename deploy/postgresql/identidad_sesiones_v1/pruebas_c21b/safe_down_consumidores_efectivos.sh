#!/usr/bin/env bash

probar_safe_down_consumidores_efectivos() {
  local up=$1
  local down=$2
  local fachada=$3
  local consumidor=$4
  local oid_inicial=$5
  local oid_final=

  paso 'safe-down rechaza una sobrecarga de la fachada propia'
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION
vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text)
RETURNS boolean LANGUAGE sql AS 'SELECT false';
REVOKE ALL ON FUNCTION
vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text)
FROM PUBLIC;
SQL
  esperar_fallo_archivo "$down" 'sobrecarga de la fachada propia'
  psql_valor "SET ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION
vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text)" \
    >/dev/null

  paso 'down conserva USAGE para consumidor efectivo directo'
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_directo()
RETURNS boolean LANGUAGE sql AS 'SELECT true';
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_directo()
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_directo() TO $consumidor;
SQL
  psql_archivo "$down" >/dev/null
  [[ $(psql_valor "SELECT has_schema_privilege('$consumidor',
'vec_identidad_sesiones_v1','USAGE')::text") == true ]]
  [[ $(psql_valor "SELECT (to_regprocedure('$fachada(text,text)')
IS NULL)::text") == true ]]
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
REVOKE EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_directo() FROM $consumidor;
DROP FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_directo();
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM $consumidor;
SQL
  psql_archivo "$up" >/dev/null

  paso 'down conserva USAGE para consumidor efectivo por herencia'
  psql_admin <<SQL
CREATE ROLE vec_c21b_consumidor_heredado NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
ALTER ROLE $consumidor INHERIT;
GRANT vec_c21b_consumidor_heredado TO $consumidor
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_heredado()
RETURNS boolean LANGUAGE sql AS 'SELECT true';
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_heredado()
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_heredado()
TO vec_c21b_consumidor_heredado;
SQL
  psql_archivo "$down" >/dev/null
  [[ $(psql_valor "SELECT has_schema_privilege('$consumidor',
'vec_identidad_sesiones_v1','USAGE')::text") == true ]]
  [[ $(psql_valor "SELECT (to_regprocedure('$fachada(text,text)')
IS NULL)::text") == true ]]
  psql_admin <<SQL
REVOKE vec_c21b_consumidor_heredado FROM $consumidor;
ALTER ROLE $consumidor NOINHERIT;
SET ROLE vec_identidad_sesiones_v1_propietario;
REVOKE EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_heredado()
FROM vec_c21b_consumidor_heredado;
DROP FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_heredado();
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM $consumidor;
RESET ROLE;
DROP ROLE vec_c21b_consumidor_heredado;
SQL
  psql_archivo "$up" >/dev/null

  paso 'down conserva USAGE para consumidor efectivo mediante PUBLIC'
  psql_admin <<'SQL'
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_public()
RETURNS boolean LANGUAGE sql AS 'SELECT true';
GRANT EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_public() TO PUBLIC;
SQL
  psql_archivo "$down" >/dev/null
  [[ $(psql_valor "SELECT has_schema_privilege('$consumidor',
'vec_identidad_sesiones_v1','USAGE')::text") == true ]]
  [[ $(psql_valor "SELECT (to_regprocedure('$fachada(text,text)')
IS NULL)::text") == true ]]
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
REVOKE EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.c21b_consumidor_public() FROM PUBLIC;
DROP FUNCTION vec_identidad_sesiones_v1.c21b_consumidor_public();
REVOKE USAGE ON SCHEMA vec_identidad_sesiones_v1 FROM $consumidor;
SQL
  psql_archivo "$up" >/dev/null

  oid_final=$(psql_valor "SELECT '$fachada(text,text)'::regprocedure::oid")
  [[ $oid_final != "$oid_inicial" ]]
}
