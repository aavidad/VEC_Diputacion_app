# F1-148 — Comparación de la fecha histórica de la firma

La firma de un documento de Contratación temporal puede rechazarse cuando la
fecha histórica termina en un cero de microsegundos. La función de firma genera
seis decimales y el constructor de competencia elimina los ceros finales; ambos
textos representan el mismo instante. La corrección compara el instante y
conserva el rechazo de una fecha diferente. La consume la pantalla de firma del
documento, incluida la segunda firma del informe.

## Activación prevista

1. Usar una copia local nueva de la principal con la lista SQL canónica completa
   de esta rama. CT259 debe quedar al final, después de las migraciones de main.
2. Comprobar la preimagen instalada de
   `vec_contratacion_temporal.registrar_firma_verificada_v3`: SHA256
   `8f58548d20ef55314fa78e631cec10d0dd3a918e46e45de5b5b527758b6afb1a`.
3. Aplicar CT259 una sola vez, con su comprobación de postimagen y los mismos
   permisos y configuración. No necesita variables, concesiones ni datos nuevos.
4. Ejecutar la prueba SQL de regresión y repetirla tras reiniciar PostgreSQL.
   Dirección organiza el ensayo y las dos revisiones independientes.
5. Arrancar la aplicación con la configuración existente y comprobar la firma
   desde su pantalla. El caso anterior de P se conserva sin reintentar; el nuevo
   recorrido se hará en otra copia, tras la integración.

Este documento describe la activación preparada. Todavía no acredita la
instalación, las revisiones ni el recorrido de firma de esta corrección. No se
cambian opciones de arranque; la aplicación sin CT259 conserva el comportamiento
anterior. No se reaplica una migración instalada ni se revierte historia.

## Alcance del cambio

Solo se reconstruye la función de firma verificada de la versión 3. Conserva
propietario, permisos, búsqueda de esquemas fija y el resto de las guardas.
Las pruebas comprobarán las fracciones `.123450Z`, `.120000Z`, `.000000Z` y
`.123456Z`, y el rechazo de un instante diferente. La prueba aislada de la
comparación no sustituye a un acto de firma completo.

La inspección de las funciones instaladas de firma y plan encontró también el
mismo patrón en `registrar_firma_verificada_v2`, SHA256
`49cf7bf45eb803baf87abee9556fe1417c61de4cf6daa8b0865f208e25ee6bd0`.
Se comunica a dirección para otro corte; no se reconstruye en esta entrega.
Los antecedentes de código están en CT172 y CT209; CT219 contiene la función
actual de la versión 3.
