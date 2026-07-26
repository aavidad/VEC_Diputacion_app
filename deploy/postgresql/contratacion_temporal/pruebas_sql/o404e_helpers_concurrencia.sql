\set ON_ERROR_STOP on
CREATE FUNCTION vec_o404e_concedida.preparar(p_caso text,p_cantidad integer)
RETURNS jsonb
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog
AS $$
  SELECT vec_o404e_concedida.preparar_ortogonal(
    p_caso,p_cantidad,NULL,NULL)
$$;
REVOKE ALL ON FUNCTION vec_o404e_concedida.preparar(text,integer)
  FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_concedida.preparar(text,integer)
  TO vec_o404e_tcb;

CREATE FUNCTION vec_o404e_concedida.carga_retirada_gobierno()
RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  IF session_user<>'vec_o404e_gob' THEN
    RAISE EXCEPTION 'lectura de retirada O4-04E fuera de contrato';
  END IF;
  RETURN (SELECT carga FROM vec_o404e_concedida.retiro_gobierno);
END
$$;
REVOKE ALL ON FUNCTION
  vec_o404e_concedida.carga_retirada_gobierno() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
  vec_o404e_concedida.carga_retirada_gobierno() TO vec_o404e_gob;
