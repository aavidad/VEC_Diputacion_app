# Recorrido de F — F1-27

En curso, 21 de octubre de 2026. Esta hoja registra el ensayo; no acredita
los pasos que aún no se han recorrido.

La copia parte de main `502a6890f4a8f964694841d9c11b2f2726242bbb`,
con la receta completa y la carga nativa de la bolsa ya integradas. No se han
añadido ramas pendientes. Go declarado: 1.26.9; la actualización a 1.26.10
aún no estaba integrada al empezar. Dirección autorizó este turno a las 03:07.

Se reutilizan `copia_local.py crear --recorrido-completo` y el preparador
versionado de los cuatro entornos de firma. El aspirante es Yago, primera
persona del XLS de Administrativo; cada referencia procede de la nueva carga.
Los materiales de E, sus claves de operación y sus actos no se trasladan a F.

## Activación

La orden completa está en [la receta del recorrido](../recorrido_copia/LEEME.md).
Se declaran los cinco puertos locales y los dos binarios de firma con sus
huellas. El volcado se lee del material privado de referencia; certificados,
conexiones, recibos y registros permanecen fuera de Git.

## Datos y actuaciones del tramo recorrido

En una copia nueva, el centro selecciona Administrativo de la RPT, grupo C1,
puesto 9 y plaza 2026/0033. La necesidad es una vacante para una persona,
con jornada de 37:30 y periodo del 2 de noviembre al 31 de diciembre de 2026.
La clasificación orgánica es 151, la funcional 920, el proyecto P-1 y la
financiación el 100 %. La petición declara la operación de crédito
220260004524, de 21 de octubre de 2026, por 8.450,32 euros.

Otra persona del centro revisa y ratifica. RRHH crea el expediente con un
número MOAD libre; en F fue 2026/95030. En el análisis conserva los datos de
la petición, elige «Necesidad temporal» y «Retención de crédito 001», e indica
37 horas y 30 minutos. Después confirma bolsa vigente, asignación e informe.

La técnica y Jefatura firman el informe desde sus paneles y confirman cada
firma en GrxFirma. Se comprueba el recibo y la verificación técnica antes de
pasar a la siguiente persona. Intervención abre el expediente en versión 5 y
registra «Favorable»; esa opción no lleva observaciones. RRHH pulsa «Llamar a
la siguiente persona» y, confirmado el llamamiento, «Anotar el aviso».
Cada copia genera sus propias claves y recibos desde los formularios.

## Resultado

| Tramo | Estado |
| --- | --- |
| Creación completa y carga de Bolsa | Correcta, 254,43 segundos; Yago disponible, 1 de 41 |
| Petición, ratificación y alta (1–3) | 200, 200 y 201; expediente 2026/95030, versión 1 |
| Análisis, vía, asignación e informe (4–7) | Cuatro respuestas 201; versiones 2, 3, 4 y 5 |
| Dos firmas del informe y fiscalización (7b–8) | Dos firmas 201 con verificación técnica válida; fiscalización favorable 201, versión 6 |
| Llamamiento, contacto, aceptación y resolución (9–11) | Selección 200 confirmada y aviso 201; contacto bloqueado por conversión horaria |
| Propuesta, resolución, documentos y tres firmas (12–15) | Pendiente |
| Reinicio y recuperación de las mismas lecturas y recibos | Pendiente |

E conserva su evidencia hasta el paso 8 y sigue parada. Este ensayo continúa
el caso único de FLUJO_APP.md, apartado 4.0, con la persona y la bolsa de F.

Antes de abrir Chrome, se prepara la confianza local de los cuatro firmantes
con el lanzador de su entorno y `--install-local-tls-trust`. Después se inicia
el navegador con la confianza limitada a los certificados propios de la copia.
La primera firma conserva `firma_eficaz=false`; la verificación técnica válida
no acredita por sí sola eficacia jurídica.

## Contacto pendiente en el paso 9

El llamamiento quedó confirmado y el buzón propio recibió un mensaje. Mi Bolsa
muestra el correo enviado; todavía no ofrece la respuesta del aspirante.

El contacto efectivo devolvió 422, `contenido_no_valido`. La entrada de las
03:26 de Madrid se envió como `03:26:00Z`, futura frente al reloj del servidor.
`formulario-llamamiento.js` añade `Z` al campo `instante_en`; el campo
`recibida_en` ya emplea la conversión de Madrid a UTC. La persistencia rechaza
correctamente un contacto futuro. Se comunicó la causa a dirección.

La ficha nativa de Bolsa confirma el llamamiento, pero el intento telefónico
está fuera de la franja 09:00–14:00. No se registró otro canal ni se creó otro
llamamiento para evitar ese límite. El enlace desde CT abre el asistente de
nuevo llamamiento, del que se salió sin crear nada. Los pasos 10–15 siguen
pendientes.

## Estado al esperar la corrección

Dirección asignó el fallo horario a F1-143 y ordenó esperar con la copia
parada. El gestor detuvo aplicación, Administración, PostgreSQL, buzón y
verificador; también se cerraron Chrome y la pantalla privada. Se conservan
la copia y la fuente de ejecución originales, sin modificar su sello.

Antes de parar, la técnica descargó desde Chrome el PDF con las dos firmas:
49.203 bytes, SHA-256
`3fe35c6a6eab71253f41576dce97bf82fb41e37ee8381996fd2eb2ba1562cb82`.
Coincide con la huella del documento firmado del segundo recibo. La instantánea
privada conserva la versión 6, los dos recibos, sus fechas y los recuentos.
Esto acredita la lectura anterior a la parada; aún no se ha hecho el reinicio
de aceptación de F.
