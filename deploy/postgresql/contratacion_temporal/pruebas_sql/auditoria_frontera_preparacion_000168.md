# Auditoría de frontera de preparación

CT168 amplía el registrador común de CT108/CT162. No crea tablas, roles ni
permisos. Su única dependencia SQL es la postimagen real de CT162; no usa
objetos de CT165–167 ni cambia el núcleo AD3.

La lista acepta dos rutas POST de Selección (`preparacion-bases/guardar` y
`consultar`) y tres de Bolsa (`reglas-baremo/borradores/alta`,
`versiones/consultar` y `recibos/recuperar`). Cada familia tiene su superficie
propia. Selección registra `acceso_denegado`; Bolsa conserva los dos motivos
fijos `autenticacion_requerida` y `acceso_denegado`.

Estos rechazos preceden a la resolución del contexto personal. El actor queda
vacío. No se trasladan JSON, cabeceras, certificados, nombres, IP ni mensajes
de error a la auditoría. La composición debe validar antes el canal mTLS y
el catálogo sellado; la lista de auditoría no concede acceso a las rutas.

El montador de Selección inyecta su callback
`RegistrarRechazoFrontera(*http.Request) error` sobre
`ports.RegistradorAuditoriaFronteraRutaExacta`. Forma una orden con la
superficie de Selección, la ruta nominal, causa `acceso_denegado` y una
correlación confiable; utiliza `corr_no_disponible` si aún no existe. El
callback no lee el cuerpo ni crea identidad. Una auditoría confirmada permite
responder 403; su ausencia o fallo exige 503. Ambos casos terminan antes del
material V3 y del repositorio de negocio.

El dispatcher común aplica la misma regla de fallo a las nuevas familias,
exige POST sin query y conserva las reglas previas de las otras rutas.
Las denegaciones posteriores de sesión o PDP mantienen sus autoridades;
este corte no observa ni recodifica respuestas de los manejadores.

Dirección ensaya el UP nuevo y la prueba SQL en una transacción del clon,
con dos revisiones sobre el mismo SHA antes de ejecutarlo. La prueba usa
RESTART transaccional y termina mediante ROLLBACK del conductor, conservando
la secuencia y la historia anteriores. DOWN rechaza su ejecución para
proteger los registros de solo adición; la recuperación de la aplicación usa
un artefacto compatible y conserva la auditoría.
