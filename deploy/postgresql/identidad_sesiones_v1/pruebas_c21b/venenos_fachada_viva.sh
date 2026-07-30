#!/usr/bin/env bash

probar_venenos_fachada_viva() {
  local down=$1
  local fachada=$2
  local consumidor=$3
  local autenticacion=$4
  local sesion=$5
  local salidas=$6
  local respaldo=revalidar_contexto_corporativo_rrhh_v1_ct92
  local definicion_original definicion_hostil oid_original

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

  paso 'retorno hostil aislado bloquea el safe-down por su manifiesto'
  definicion_original=$(psql_valor \
    "SELECT pg_get_functiondef('$fachada(text,text)'::regprocedure)")
  [[ $(grep -Fo \
    'identidad_valida_hasta timestamp with time zone' \
    <<<"$definicion_original" | wc -l) == 1 ]]
  definicion_hostil=${definicion_original/identidad_valida_hasta timestamp with time zone/identidad_caduca_en timestamp with time zone}
  [[ $definicion_hostil != "$definicion_original" ]]
  [[ ${definicion_hostil/identidad_caduca_en/identidad_valida_hasta} == \
    "$definicion_original" ]]
  oid_original=$(psql_valor \
    "SELECT '$fachada(text,text)'::regprocedure::oid")
  eliminar_proxy
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
ALTER FUNCTION $fachada(text,text) RENAME TO $respaldo;
$definicion_hostil;
REVOKE ALL ON FUNCTION $fachada(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION $fachada(text,text) TO $consumidor;
SQL
  [[ $(psql_valor "WITH
o AS (SELECT * FROM pg_proc WHERE oid=
'vec_identidad_sesiones_v1.$respaldo(text,text)'::regprocedure),
h AS (SELECT * FROM pg_proc WHERE oid='$fachada(text,text)'::regprocedure)
SELECT (
  o.oid<>h.oid AND o.proowner=h.proowner AND o.prolang=h.prolang
  AND o.prokind=h.prokind AND o.provolatile=h.provolatile
  AND o.proparallel=h.proparallel AND o.prosecdef=h.prosecdef
  AND o.proleakproof=h.proleakproof AND o.proisstrict=h.proisstrict
  AND o.proretset=h.proretset AND o.pronargs=h.pronargs
  AND o.pronargdefaults=h.pronargdefaults
  AND o.proargtypes=h.proargtypes AND o.prorettype=h.prorettype
  AND o.proallargtypes=h.proallargtypes AND o.proargmodes=h.proargmodes
  AND o.proconfig=h.proconfig AND o.prosrc=h.prosrc
  AND o.proargnames[1:5]=h.proargnames[1:5]
  AND o.proargnames[6]='identidad_valida_hasta'
  AND h.proargnames[6]='identidad_caduca_en'
  AND replace(pg_get_functiondef(o.oid),'$respaldo',
      'revalidar_contexto_corporativo_rrhh_v1')
      =replace(pg_get_functiondef(h.oid),'identidad_caduca_en',
      'identidad_valida_hasta')
)::text FROM o CROSS JOIN h") == true ]] || {
    echo 'la copia con OUT hostil alteró metadatos o cuerpo SQL' >&2
    return 1
  }
  [[ $(psql_valor "WITH
o AS (SELECT oid,proacl FROM pg_proc WHERE oid=
'vec_identidad_sesiones_v1.$respaldo(text,text)'::regprocedure),
h AS (SELECT oid,proacl FROM pg_proc
      WHERE oid='$fachada(text,text)'::regprocedure)
SELECT (NOT EXISTS (
    SELECT * FROM (
      (SELECT grantor,grantee,privilege_type,is_grantable
       FROM aclexplode(o.proacl)
       EXCEPT
       SELECT grantor,grantee,privilege_type,is_grantable
       FROM aclexplode(h.proacl))
      UNION ALL
      (SELECT grantor,grantee,privilege_type,is_grantable
       FROM aclexplode(h.proacl)
       EXCEPT
       SELECT grantor,grantee,privilege_type,is_grantable
       FROM aclexplode(o.proacl))
    ) AS diferencia_acl
))::text FROM o CROSS JOIN h") == true ]] || {
    echo 'la copia con OUT hostil alteró la ACL nominal' >&2
    return 1
  }
  [[ $(psql_valor "WITH
o AS (SELECT oid FROM pg_proc WHERE oid=
'vec_identidad_sesiones_v1.$respaldo(text,text)'::regprocedure),
h AS (SELECT oid FROM pg_proc
      WHERE oid='$fachada(text,text)'::regprocedure)
SELECT (NOT EXISTS (
    SELECT * FROM (
      (SELECT refclassid,refobjid,refobjsubid,deptype
       FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=o.oid
       EXCEPT
       SELECT refclassid,refobjid,refobjsubid,deptype
       FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=h.oid)
      UNION ALL
      (SELECT refclassid,refobjid,refobjsubid,deptype
       FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=h.oid
       EXCEPT
       SELECT refclassid,refobjid,refobjsubid,deptype
       FROM pg_depend WHERE classid='pg_proc'::regclass AND objid=o.oid)
    ) AS diferencia_dependencias
  )
  AND (SELECT count(*) FROM pg_depend
       WHERE classid='pg_proc'::regclass
       AND objid=h.oid AND refclassid='pg_proc'::regclass
       AND refobjid=
       'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure
       AND deptype='n')=1)::text FROM o CROSS JOIN h") == true ]] || {
    echo 'la copia con OUT hostil alteró dependencias nominales' >&2
    return 1
  }
  [[ $(psql_valor "WITH
o AS (SELECT oid FROM pg_proc WHERE oid=
'vec_identidad_sesiones_v1.$respaldo(text,text)'::regprocedure),
h AS (SELECT oid FROM pg_proc
      WHERE oid='$fachada(text,text)'::regprocedure)
SELECT (pg_get_function_result(o.oid)=
      'TABLE(cuenta_ref text, metodo_observado text, garantia_observada text, identidad_valida_hasta timestamp with time zone)'
  AND pg_get_function_result(h.oid)=
      'TABLE(cuenta_ref text, metodo_observado text, garantia_observada text, identidad_caduca_en timestamp with time zone)'
)::text FROM o CROSS JOIN h") == true ]]
  esperar_fallo_archivo "$down" \
    'retorno hostil con único nombre OUT distinto'
  grep -Fq '55000' "$salidas/fallo_archivo"
  psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
DROP FUNCTION $fachada(text,text);
ALTER FUNCTION vec_identidad_sesiones_v1.$respaldo(text,text)
  RENAME TO revalidar_contexto_corporativo_rrhh_v1;
SQL
  [[ $(psql_valor \
    "SELECT '$fachada(text,text)'::regprocedure::oid") == "$oid_original" ]]
  crear_proxy
  exigir_uno "$autenticacion" "$sesion" \
    'retorno nominal de cuatro columnas restaurado'
}
