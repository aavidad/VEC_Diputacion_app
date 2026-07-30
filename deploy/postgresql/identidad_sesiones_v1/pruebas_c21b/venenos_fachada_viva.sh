#!/usr/bin/env bash

probar_venenos_fachada_viva() {
  local up=$1
  local down=$2
  local fachada=$3
  local consumidor=$4
  local autenticacion=$5
  local sesion=$6
  local salidas=$7
  local respaldo=revalidar_contexto_corporativo_rrhh_v1_ct92

  paso 'SECURITY INVOKER atraviesa la fachada y falla cerrado'
  psql_valor "ALTER FUNCTION $fachada(text,text) SECURITY INVOKER" \
    >/dev/null
  [[ $(psql_valor "SELECT (NOT prosecdef)::text FROM pg_proc
WHERE oid='$fachada(text,text)'::regprocedure") == true ]]
  if actor_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM vec_contexto_actor_v1.c21b_proxy_identidad(
'$autenticacion','$sesion'); COMMIT" \
      >"$salidas/security_invoker" 2>&1; then
    echo 'SECURITY INVOKER no fallo cerrado dentro de C2.1b' >&2
    return 1
  fi
  grep -Fq '42501' "$salidas/security_invoker"
  grep -Fq \
    'SQL function "revalidar_contexto_corporativo_rrhh_v1"' \
    "$salidas/security_invoker"
  psql_valor "ALTER FUNCTION $fachada(text,text) SECURITY DEFINER" \
    >/dev/null
  [[ $(psql_valor "SELECT prosecdef::text FROM pg_proc
WHERE oid='$fachada(text,text)'::regprocedure") == true ]]
  exigir_uno "$autenticacion" "$sesion" 'SECURITY DEFINER restaurado'

  paso 'lenguaje hostil atraviesa la inspeccion interna de la fachada real'
  eliminar_proxy
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
ALTER FUNCTION $fachada(text,text) RENAME TO $respaldo;
CREATE FUNCTION $fachada(
  p_autenticacion_ref text,p_sesion_ref text
)
RETURNS TABLE(
  cuenta_ref text,metodo_observado text,garantia_observada text,
  identidad_valida_hasta timestamptz
)
LANGUAGE plpgsql VOLATILE CALLED ON NULL INPUT SECURITY DEFINER
PARALLEL UNSAFE SET search_path=pg_catalog SET lock_timeout='1s'
AS \$funcion\$
BEGIN
  RETURN QUERY
  SELECT r.cuenta_ref,r.metodo_observado,r.garantia_observada,
         r.identidad_valida_hasta
  FROM vec_identidad_sesiones_v1.$respaldo(
    p_autenticacion_ref,p_sesion_ref
  ) AS r;
END
\$funcion\$;
REVOKE ALL ON FUNCTION $fachada(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION $fachada(text,text) TO $consumidor;
SQL
  [[ $(psql_valor "SELECT l.lanname FROM pg_proc p
JOIN pg_language l ON l.oid=p.prolang
WHERE p.oid='$fachada(text,text)'::regprocedure") == plpgsql ]]
  crear_proxy
  exigir_cero "$autenticacion" "$sesion" \
    'la fachada real detecta el lenguaje plpgsql del nombre nominal'
  eliminar_proxy
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION $fachada(text,text);
ALTER FUNCTION vec_identidad_sesiones_v1.$respaldo(text,text)
  RENAME TO revalidar_contexto_corporativo_rrhh_v1;
SQL
  [[ $(psql_valor "SELECT l.lanname FROM pg_proc p
JOIN pg_language l ON l.oid=p.prolang
WHERE p.oid='$fachada(text,text)'::regprocedure") == sql ]]
  crear_proxy
  exigir_uno "$autenticacion" "$sesion" 'lenguaje SQL restaurado'

  paso 'retorno hostil bloquea la reinstalacion en un ciclo aislado'
  eliminar_proxy
  psql_archivo "$down" >/dev/null
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION $fachada(text,text)
RETURNS TABLE(
  cuenta_ref text,metodo_observado text,garantia_observada text
)
LANGUAGE sql VOLATILE CALLED ON NULL INPUT SECURITY DEFINER
PARALLEL UNSAFE SET search_path=pg_catalog SET lock_timeout='1s'
BEGIN ATOMIC
  SELECT NULL::text,NULL::text,NULL::text WHERE false;
END;
REVOKE ALL ON FUNCTION $fachada(text,text) FROM PUBLIC;
SQL
  [[ $(psql_valor "SELECT (pg_get_function_result(
'$fachada(text,text)'::regprocedure)=
'TABLE(cuenta_ref text, metodo_observado text, garantia_observada text)')::text") == true ]]
  esperar_fallo_archivo "$up" 'retorno hostil de tres columnas'
  grep -Fq '55000' "$salidas/fallo_archivo"
  psql_valor "SET ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION $fachada(text,text)" >/dev/null
  psql_archivo "$up" >/dev/null
  crear_proxy
  exigir_uno "$autenticacion" "$sesion" \
    'retorno nominal de cuatro columnas restaurado'
}
