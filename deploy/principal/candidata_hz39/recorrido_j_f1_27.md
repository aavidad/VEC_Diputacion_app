# Recorrido de J — F1-27

En curso, 21 de octubre de 2026. Fuente de ejecución:
`5abbbcaae5c47c65dc6954daf92015b7aad08aeb`, main con el diagnóstico de
selección nativa de F1-27. Dirección autorizó esta copia a las 09:37,
tras integrar el registro de errores fijos.

La [receta completa](../recorrido_copia/LEEME.md) crea una copia nueva con
material e identidades propios. I se ha parado con su gestor y se conserva
sin cambiar su fuente, binario, petición ni clave. H permaneció parada hasta comprobar las dos firmas de J y se retiró después
con su gestor para liberar memoria, conforme a la orden anterior de limpieza.
Sus evidencias privadas y su hoja se conservan; el expediente 2026/95032
no se reintentó. No se añaden ramas.

| Tramo | Estado |
| --- | --- |
| Creación completa y carga nativa de Bolsa | Completada en 256,57 segundos; 146 migraciones |
| Petición, ratificación y alta (1–3) | 200, 200 y 201; expediente 2026/95034, versión 1 |
| Análisis, vía, asignación e informe (4–7) | Actuaciones 201 hasta versión 5 |
| Dos firmas del informe y fiscalización (7b–8) | Dos firmas 201 y fiscalización favorable 201; versión 6 |
| Selección, aviso, contacto y aceptación (9–11) | Selección 200; aviso, contacto, aceptación, vínculo y resolución portal 201 |
| Propuesta, resolución, documentos y tres firmas (12–15) | Paso 12 bloqueado en interfaz; no se envía la propuesta |
| Incorporación, Personal, GINPIX, cese, vuelta a Bolsa y cierre (17–21) | Pendiente |
| Reinicio y recuperación de lecturas y recibos | Pendiente |

La causa del 503 de I sigue sin identificarse. El diagnóstico integrado solo
amplía el registro con errores fijos reconocidos; conserva los desconocidos
como tipo. No modifica el error devuelto, la autorización ni las reglas.
Si J confirma la selección, el recorrido continuará; si falla, se conservarán
su petición, correlación y causa observada antes de proponer una corrección.
Los materiales privados quedan fuera de Git. Esta hoja no cierra F1-27 ni F1-28.

La primera apertura del programa de firma se detuvo en la conexión TLS, antes
de mostrar el selector y sin enviar registros de firma a VEC. El navegador
se había iniciado antes de preparar las cuatro autoridades locales de J.
Se cerraron navegador y programa de firma de esa sesión y se abrió el
navegador con esas autoridades ya preparadas. Solo se modifica el entorno
privado del navegador; se conservan las comprobaciones TLS y los permisos.

Tras reabrir el navegador, ambas firmas del informe devolvieron 201,
verificación técnica válida y `firma_eficaz=false`. Hubo un registro por
firmante. El PDF descargado desde la ficha contiene 49.314 bytes y su SHA-256,
`86b8551684eaac338c3c9a7fa5c87b8508baa029b609e73254d8a576a566af99`,
coincide con el resultado de la segunda firma. Intervención confirmó después
el resultado favorable con su recibo y la versión resultante 6. Hasta este
tramo no se ha solicitado selección ni registrado contacto en J.

## Selección, contacto y aceptación

La selección devolvió 200 en su único envío, a las 10:02:08 de Madrid, con
recibo confirmado. El aviso se anotó después con respuesta 201. El resultado
de J no permite atribuir la causa del 503 de I al código ni darla por resuelta.

«Abrir llamamiento en Bolsa» llevó al asistente de otro llamamiento; no se
confirmó nada allí. Al volver a la ficha, el bloque de contacto de CT no se
muestra para una comunicación recuperada. La vía existente que sí se usó es
el seguimiento del llamamiento en Bolsa: «Ver ficha y llamar» abre la ficha de
Yago y el registro telefónico. Se confirmó «Contactado» una sola vez, con
respuesta 201 a las 10:08:51 y la hora automática del servidor. No hubo otra
selección ni un llamamiento nuevo.

Mi Bolsa mostró ese contacto y el plazo hasta el 23 de octubre a las 00:00,
hora de Madrid. Yago aceptó con su certificado; la respuesta firme devolvió
201 a las 10:09:34. RRHH vinculó por pantalla la emisión ofrecida como propia
del expediente (201), revisó la aceptación y su plazo y la confirmó (201 a
las 10:11:55). CT permanece en versión 6. No se aportó un correo ficticio
para sustituir esa respuesta ni se modificaron permisos.

## Paso 12: propuesta sin petición

La ficha muestra la candidatura aceptada y el formulario «Guardar la
propuesta», con las cuatro publicaciones cargadas. Pulsarlo no envía una
petición ni cambia el mensaje «Documentos cargados. Revise y guarde la
propuesta». Seleccionar después la comunicación existente en su lista hace
desaparecer ese formulario. No se ha enviado ningún POST de propuesta.

El inventario de código y una comprobación independiente localizan una
contradicción: `formulario-llamamiento.js:411` muestra la propuesta de Mi Bolsa
separada cuando no hay comunicación de CT seleccionada; el envío exige esa
selección en las líneas 801–805. La guarda de consulta de respuesta de las
líneas 806–807 también depende de un antecedente de comunicación de CT,
aunque la aceptación procede de Mi Bolsa. El código de preparación ya
conserva la resolución y su recibo como antecedente nominal del portal.

Se ha comunicado a dirección para asignar la corrección mínima. No se han
fabricado solicitudes ni alterado la selección del navegador. Las capturas a
1440 y 390 px y el registro de peticiones quedan en el material privado;
a 390 px el contenido mide 375 px y no desborda. Los pasos 12–21 y el
reinicio final siguen pendientes.
