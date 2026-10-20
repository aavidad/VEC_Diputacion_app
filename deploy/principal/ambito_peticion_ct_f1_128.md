# Organización de las peticiones del centro

F1-128 conserva la organización de cada revisión de petición y la coteja en
la lectura, la ratificación y la entrega a RRHH. La reserva, su confirmación y
el alta deben pertenecer a esa misma organización. El servidor entrega el
valor de su instancia a SQL; la decisión de autorización queda ligada a él
por la huella del contexto. El navegador no elige ese ámbito.

Las revisiones anteriores mantienen su referencia original. No se modifican
documentos, huellas de material, recibos ni identidades. Una repetición de una
operación de otra organización se deniega. Ratificar una petición ajena y
ratificar una inexistente conservan la misma respuesta de conflicto.

## Instalación

Instalar CT252 al final de `lista_sql_main_tras_hz37.txt` junto con el binario
de este corte. Cambian las firmas de las cuatro funciones de petición y
entrega, y la de proyección previa a la entrega. Se retiran las anteriores;
no hay un periodo compatible entre binarios ni se reaplica la migración.

La migración comprueba la definición instalada y los permisos antes de
cambiar la tabla. Conserva propietario y concesiones. No publica roles ni
requiere actos adicionales de Administración para el montaje vigente.

Este corte no añade configuración ni activa una organización maestra. El
montaje conserva su referencia vigente hasta el corte de activación. Ajustes,
plantillas y las demás capacidades fuera del recorrido de petición y alta
siguen pendientes de su publicación con el ámbito correspondiente.

## Comprobación pendiente

La focal SQL registrada comprueba firmas, permisos, forma del ámbito e
inmutabilidad. Las pruebas del adaptador comprueban el argumento enviado y
el rechazo de un recurso ajeno antes de abrir la transacción.

Antes de declarar este corte listo faltan el ensayo de SQL en copia propia,
el recorrido HTTP de petición y entrega, su recuperación tras reiniciar y
las dos revisiones independientes. El recorrido con la organización maestra
requiere además el corte de activación; no queda acreditado por estas pruebas.
