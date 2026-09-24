-- La reversión de funciones sobre recibos con historia no es segura.
DO $rechazo$
BEGIN
  RAISE EXCEPTION 'ContextoActor 000006 conserva historia: DOWN prohibido' USING ERRCODE='55000';
END
$rechazo$;
