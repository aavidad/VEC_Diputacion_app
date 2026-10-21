# F1-149 — Fecha histórica de la función anterior de firma

La función anterior de registro de firma conserva la comparación de fechas como
texto que CT259 corrigió en la versión actual. CT260 aplicará el mismo cotejo
como instante a `registrar_firma_verificada_v2`. No concede ejecución a nadie:
la versión anterior conserva sus restricciones actuales.

## Activación preparada

Usar una copia local nueva de la principal y la lista SQL canónica de esta rama.
CT260 debe quedar después de CT259 y del resto de las migraciones de main.
La preimagen instalada de la función anterior tiene SHA256
`49cf7bf45eb803baf87abee9556fe1417c61de4cf6daa8b0865f208e25ee6bd0`.
La reconstrucción requiere su cuerpo instalado, propietario, permisos y
configuración cotejados; esos metadatos están pedidos a dirección.

No necesita configuración, actos ni datos nuevos. El arranque conserva la
configuración existente. Dirección organiza el ensayo y dos revisiones
independientes sobre la cabeza final; la prueba de comparación aislada no
sustituye a una firma completa.

## Reconocimiento de la migración instalada

CT260 reconocerá tanto su preimagen como su postimagen exactas, con los mismos
permisos y configuración. Ante la postimagen omitirá la reconstrucción y avisará
que ya está instalada. Cualquier otra definición seguirá rechazándose.

CT259 ya está instalada. Añadirle ese reconocimiento requeriría cambiar sus
bytes y su huella en la lista canónica. Se conserva íntegra y se deja anotado
este límite, como permite el encargo. No se reaplica CT259 ni se modifica
historia.

La prueba histórica de CT219 se ajustará a las postimágenes vigentes de ambas
funciones. Conserva el resto de los cotejos de permisos y del plan de firma.
