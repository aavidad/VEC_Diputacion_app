-- La reversión de funciones que firman recibos con historia no es segura.
DO $rechazo$
BEGIN
  RAISE EXCEPTION 'ContextoActor 000007 conserva historia: DOWN prohibido' USING ERRCODE='55000';
END
$rechazo$;
