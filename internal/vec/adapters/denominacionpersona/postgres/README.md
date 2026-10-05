# Adaptador PostgreSQL de la denominación de Persona

`Nuevo(ctx, pool, protector)` acredita el runtime mediante CA32. La ausencia de
la fachada AD nominal o del gate AUT42 cierra el constructor. No publica permisos,
no siembra fuentes y no descifra datos.

La composición prepara primero el sobre con el protector existente, resuelve
una procedencia concreta y llama a `MaterialPublicacion`. La lectura usa
`MaterialLectura`. Ambos helpers devuelven el material y su recurso; el emisor
central liga ese recurso a una capacidad V3 nominal. La orden del puerto conserva
el material original, la evidencia V2 y su vínculo, los ámbitos y la correlación.

La publicación sólo entrega el recibo tras COMMIT confirmado. La lectura sólo
entrega el sobre y el acuse después de confirmar el consumo común. Un fallo o
COMMIT incierto devuelve `ErrNoDisponible`, sin contenido ni reintento automático.
El lector existente vuelve a cotejar el acuse y el acceso antes de llegar al KMS.

Implementa los puertos de registro y fuente autorizada existentes. La búsqueda
permanece cerrada en este corte. No hay composición productiva ni positivo SQL
acreditado; el [contrato CA32](../../../../../deploy/postgresql/contexto_actor_v1/DENOMINACION_PERSONA_000032.md)
detalla dependencias, bytes canónicos y límites de entrega.
