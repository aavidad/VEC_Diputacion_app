# Unidad del puesto y unidad gestora del llamamiento

F1-21 corrige la lectura del puesto que hace la ficha de Contratación antes de
ofrecerlo desde una bolsa constituida. Antes enviaba a Personal la unidad que
gestiona el expediente. Ahora toma la unidad del puesto de la RPT publicada
validada y conserva la unidad gestora para la autorización de Contratación y
el origen del llamamiento en Bolsa. No cambia la pantalla ni los permisos.

Reutiliza el contrato de la carga RPT de Personal: el código de centro publicado
se corresponde con `unidad:centro:<codigo_en_minusculas>`. La petición debe
conservar la referencia y huella de esa publicación. La consulta histórica
comprueba además el puesto individual, su unidad, vigencia, unicidad y todas
las páginas del mismo corte. Un centro no mapeable o un puesto ajeno o ambiguo
se rechazan. No se amplía la consulta a todo el organismo.

## Activación

No hay SQL ni configuración nueva. Instalar primero la consulta histórica de
la PR #1150 con su material nominal y el cliente mTLS existente de Contratación.
Preparar el binario y conservar el anterior para la vuelta atrás. Instalar con
la aplicación detenida y vigilar el reinicio. Los datos del puesto y la unidad
se conservan: no se modifican para hacer coincidir puesto y unidad gestora.

## Comprobación

Usar la misma petición del caso de `FLUJO_APP.md`, apartado 4.0, vinculada a la
necesidad de la RPT. La unidad gestora de RRHH y la unidad del puesto del centro
son distintas. Desde la ficha, abrir «Llamamiento y comunicación» y realizar la
selección por la operación existente. La lectura de Personal debe usar la
unidad del puesto y producir su recibo real; Contratación y Bolsa conservan la
unidad gestora y sus permisos. La continuidad o parada de la selección depende
del orden y de las reglas de la bolsa; este corte no modifica esas reglas.

Con la consulta histórica desactivada o su configuración ausente, la aplicación
sigue arrancando y la selección queda no disponible por la vía existente. Un
puesto ajeno o ambiguo y un centro no mapeable no crean un llamamiento. Conservar
recibos y comprobar el mismo expediente después de reiniciar.

## Evidencia de la composición local

Los dos archivos Go mantienen exactamente los bytes de la cabeza `62db5b475`
revisada; integrar la principal no cambió su lógica. La focal Go de unidades
distintas y rechazos, `gopls`, Semgrep y gosec fueron favorables. La CI ejecuta
la puerta completa. Este corte no modifica SQL, permisos ni configuración.

El binario aislado anterior arrancó y la ficha respondió 200. Su intento nativo
se detuvo antes de reservar: la copia tenía la concesión de origen ampliada a
40 campos por R1a, mientras aquel montaje exigía la anterior de 38. Fue un
rechazo de contratos incompatibles, no una prueba de ofrecimiento exitoso.
La fuente y la concesión publicadas deben corresponder a la misma versión;
no se rebaja esa comprobación para activar la lectura del puesto.

El consumidor se comprobó en la composición de R1a con CT230 y CT231. Desde
la ficha del caso 2026/94011 se confirmó la selección nativa, se conservó su
justificante tras reiniciar aplicación y PostgreSQL y se anotó el aviso con
recibo real. Personal consultó la unidad del puesto; Contratación y Bolsa
mantuvieron la gestora.

En el montaje final `e065ac39b`, binario normal SHA256
`6162f2dbff0c308639c605f7dfd5080b42ef90af3d93ce107c4df9d128b89d75`,
se recuperó la confirmación con el cliente web real y la misma solicitud y clave.
El resultado fue idéntico al anterior, estado confirmado, un origen y un
justificante nativos, con los bytes conservados. Fue recuperación idempotente,
no una selección nueva. Los ajustes del recibo y la fuente ampliada pertenecen
a #1144/#1170; la recuperación de ficha, a #1178. La activación conjunta está
descrita en la guía `habilitacion_criterios_ofrecimiento_20261015.md` de la
[PR #1144](https://github.com/aavidad/VEC_Diputacion_app/pull/1144).

La lectura del puesto también precedió al rechazo seguro por comprobación
pendiente en la copia desechable R59: decisión, atestación, consumo y auditoría
nominales de Personal, sin crear otro llamamiento. El dato del expediente se
conservó durante esa prueba. La atribución de esos asientos es por recurso y
ventana temporal, no por la correlación HTTP del rechazo.

El ensayo usa datos ficticios y solo acredita este consumidor de la consulta.
No se atribuye al binario aislado de este corte la confirmación nativa de la
composición posterior ni el recorrido completo de FLUJO_APP §4.0. Dirección
integra y prepara la candidata conjunta antes de cualquier instalación.

## Contraste tras recibir los justificantes de la principal

La integración `b4d21b426` conserva los dos archivos Go de este corte. En la
composición `70a29941d`, con el binario normal SHA256
`6d26fc8910751b60f7f70a8b393772e9f410514ebda5aacf67f9affc1659e685`,
Chrome recuperó de nuevo la selección con la solicitud y clave originales,
antes y después de reiniciar aplicación y PostgreSQL. Resultado, justificante,
artefacto y único origen se conservaron. La ficha mostró el llamamiento.
La copia negativa mantuvo el rechazo pendiente sin efectos y abrió la bolsa
real con teclado. Es un contraste del mismo consumidor en la composición;
no atribuye una selección nueva ni completa el recorrido del apartado 4.0.
