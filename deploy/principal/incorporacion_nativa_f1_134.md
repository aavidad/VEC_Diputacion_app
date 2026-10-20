# Incorporación y cierre del nombramiento nativo

F1-134 continúa el mismo expediente del caso 4.0 de
[`FLUJO_APP.md`](../../docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación),
desde la resolución de nombramiento. Los pasos y sus recibos siguen pendientes
de comprobar en la candidata conjunta; este documento no acredita su ejecución.

## Antecedente del plan de Personal

El lector del plan usaba la aceptación vigente del circuito de correo, que
exige un justificante. La resolución de Mi Bolsa conserva su respuesta y su
recibo originales, y exige que ese justificante esté vacío. CT255 añade a la
fuente existente de antecedentes B2 una selección específica para esa pareja
de resolución y recibo. Conserva primero el camino de correo.

La propuesta debe pertenecer al expediente y a la organización consultados,
conservar el mismo llamamiento y el contenido nativo de aceptación. Las
referencias de respuesta y recibo tienen que coincidir con la resolución
publicada. CT212 mantiene el cotejo posterior con la fuente propietaria de
Bolsa. No se fabrica un justificante ni se traslada un recibo de otra copia.
El lector general de aceptación y el circuito de no incorporación no cambian.

## Activación

CONFIG NUEVA: ninguna. Preparar Personal con la orden existente del
[apartado 12 de la fábrica](LEEME_copia_local.md#12-personal-incorporación-alta-ginpix-manual-y-cese-f1-119),
usando la configuración aprobada y las conexiones propias. Conservar su regla
`cese_fecha_efecto`: no tiene valor por omisión ni se decide en el navegador.

CT255 requiere CT249 y CT253, en el orden definitivo de la candidata. Su fila
en la lista única y su ensayo quedan pendientes hasta fijar esas dependencias.
No reaplicar migraciones ni presentar como activación los datos preparados
directamente en una copia. No crear por SQL la incorporación, relación,
ocupación, cese o cierre que deben registrar las pantallas.

## Recorrido que se debe comprobar

| Paso | Operación existente | Evidencia |
| --- | --- | --- |
| 17 | El centro acredita la incorporación | Recibo de centro, fecha y documento |
| 18 | RRHH prepara y confirma el plan B2 | Plan con intención durable; recibo de alta, relación y ocupación en Personal |
| 16a–16b | RRHH descarga la ficha y confirma la grabación manual en GINPIX | Ficha ligada al origen B2 y recibo local de número y fecha; sin transmisión automática |
| 19 | RRHH registra el cese | Recibo CT y fin de la relación y ocupación comprobado en la ficha nominal de Personal |
| 20 | Bolsa aplica su política de retorno | Estado Disponible o Disponible desde en Mi Bolsa o en la ficha del candidato |
| 21 | RRHH cierra con el cese CT y GINPIX confirmados | Expediente completado y recibo de cierre |

Si se pierde una respuesta, conservar la misma clave y recuperar la operación;
no crear otra para simular el reintento. Tras reiniciar aplicación y PostgreSQL,
comprobar los mismos recibos y fechas, y la ausencia de duplicados. La ficha de
Personal y el estado de Bolsa son lecturas de sus propietarios. El cierre no
añade una puerta nueva sobre sus recibos: esa ampliación queda en F1-135.

## Estado del corte

La función se ha preparado desde su definición instalada en una copia propia
de la principal. Faltan la prueba SQL con las dependencias definitivas, el
recorrido HTTP y en Chrome, el reinicio y las dos revisiones independientes.
No hay todavía una prueba positiva de alta o cese sobre el nombramiento nativo.
