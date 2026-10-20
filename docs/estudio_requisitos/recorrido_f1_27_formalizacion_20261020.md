# Propuesta y resolución del caso 4.0: comprobación del 20/10/2026

Tarea F1-27. Ensayo en una copia local nueva de `main@fb5200b68`, con
datos ficticios y Chrome del sistema. No se ha tocado cidonia. El caso de
aceptación sigue siendo [FLUJO_APP, apartado 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).

El recorrido sigue abierto: el expediente 2026/93001 ha llegado a
Fiscalización, versión 6, con las dos firmas del informe y un llamamiento
de Antonio emitido y vinculado. Falta registrar y resolver su aceptación
antes de comprobar los pasos 12 a 15 en ese mismo expediente.

## Preparación reproducida

Se usó `deploy/principal/copia_local.py crear`, con módulos, la identidad
original de RRHH y la lista completa
`deploy/principal/lista_sql_main_tras_hz37.txt`. La preparación tardó
101,43 segundos. Después, la orden `firma` completó sus siete etapas
privadas: Administración, firmantes, actos, plan y configuración. Se aprobó
la provisión de los dos perfiles fijos de RRHH y se reinició la copia.
Los certificados, conexiones, cuerpos de petición y recibos completos
permanecen fuera de Git.

Se reutilizaron la fábrica de copias, sus datos del caso 4.0 y el circuito
de firma de F1-32/F1-94. GrxFirma usado: 0.0.111. No se han añadido
migraciones, permisos ni código de producto.

## Expediente 2026/93001

| Paso | Pantalla y operación | Respuesta | Resultado |
| --- | --- | --- | --- |
| 7b, técnico | Ficha, firma del informe definitivo | `POST /firmas-documento/registro-vec` 201 | Firma registrada a las 9:45:56 |
| 7b, Jefatura | Ficha, segunda firma del mismo informe | La misma ruta, 201 | Firma registrada a las 9:48:17 |
| 8 | Intervención, resultado favorable | `POST /fiscalizaciones/resultados` 201 | Fiscalización, versión 6, a las 9:48:44 |
| 9 | Abrir llamamiento en Bolsa; seleccionar Antonio y emitir | `POST /api/vec/bolsa/llamamientos/emisiones` 201 | Emisión registrada; la ficha muestra «Correo enviado» |
| 9, vínculo | Volver a la ficha y vincular el llamamiento | `POST /expedientes/vinculos-bolsa` 201 | Antonio y la fecha de emisión aparecen en la ficha |
| 10 por otro canal | Buscar el registro manual indicado por dirección | No hay formulario en la ficha | No se envió una aceptación |
| 11–15 | Continuación desde la aceptación | No ejecutados | Pendientes de resolver el bloqueo anterior |

Las rutas abreviadas cuelgan de `/api/vec/contratacion-temporal`.
Las firmas tienen garantía sustancial y `firma_eficaz=false`, como
corresponde a los certificados de desarrollo. No acreditan firma oficial.

La primera consulta del original para Jefatura se interrumpió con
`ERR_NETWORK_CHANGED`, antes del registro. Una consulta posterior permitió
firmar y registrar con 201. Hay dos filas de firma, una por persona. No se
ha reproducido en este expediente el rechazo 403 investigado en F1-117.

## Bloqueo antes de la propuesta

Dirección indicó el 20/10 a las 9:51 registrar desde RRHH la respuesta
recibida por teléfono o correo. Tras vincular la emisión de Bolsa, la ficha
muestra «Sin respuesta registrada» y ofrece «Ver seguimiento en Bolsa»,
pero no ofrece registrar ni resolver una respuesta manual. El bloque
«Llamamientos de este expediente» dice que todavía no tiene llamamientos.

Antes del vínculo, la consulta de comunicaciones respondió 403. Después,
el bloque de selección ofrece «Llamar a la siguiente persona», cuyo montaje
de Organización histórica depende de F1-103. Pulsar «Ver seguimiento en
Bolsa» conserva la misma dirección y la misma ficha, sin abrir otra pestaña.

La consulta de Mi Bolsa de Antonio responde 200 y muestra el envío del
correo de las 9:55, pero no presenta una acción para aceptar ese llamamiento.
Esta lectura no acredita su aceptación. También aparecen fallos de
preferencias e imagen del área personal; no se consideran permisos nuevos
ni se han alterado para continuar.

La resolución manual de CT exige dos comprobaciones expresas de la técnica
de RRHH. No exige una segunda persona: esa condición pertenece al circuito
distinto de no incorporación. No se ha rehecho la copia ni añadido otro
perfil para suplir el formulario ausente.

## Comprobación histórica separada

Para localizar los paneles posteriores se abrió también 2026/CT-000132,
que ya tenía una propuesta histórica, versión 7. Registrar su validación
manual de resolución devolvió 201 y dejó la versión 8. Es un expediente
de Auxiliar Administrativo distinto del caso de Antonio: ese resultado no
acredita el paso 13 del caso 4.0.

Desde esa ficha se descargaron la resolución, diligencia, toma de posesión,
notificación y comunicación al centro en PDF y Word: diez ficheros. La
resolución conserva el logotipo y el número 2026/CT-000132; los cinco Word
contienen su XML y una imagen. Es una comprobación separada de descargas,
sin firmas de resolución y sin aceptación del caso de Antonio.

En esa ficha, las consultas de plan de alta en Personal y vínculo de
categoría RPT responden 404. La fábrica no prepara por sí sola
`VEC_CT_INCORPORACION_V2_FILE` ni la configuración de Personal y relevo de
cese. Su activación se está preparando por la receta existente; no se
han fabricado capacidades, firmas ni recibos para suplirla.

## Reinicio parcial

Se reiniciaron la aplicación y PostgreSQL con la revisión de código
ensayada. Antes y después son idénticos la versión 6, los dos recibos de
firma, sus fechas y las huellas de los PDF firmados. También coinciden
estos ocho recuentos:

| Registro | Antes y después |
| --- | --- |
| Firmas | 2 |
| Auditorías de firma | 2 |
| Custodias de firma | 2 |
| Salidas de firma | 2 |
| Revisiones de PDF | 2 |
| Reservas de fiscalización | 10 |
| Terminales de fiscalización | 10 |
| Vínculos de emisión de Bolsa | 1 |

La ficha conserva Antonio, la emisión de las 9:55 y «Correo enviado».
La lectura de comunicaciones respondió 503 en una carga posterior al
reinicio; no se considera resuelta ni se presenta el arranque como libre
de errores. El bloque vinculado sigue mostrando «Sin respuesta registrada».
En 1440 y 390 píxeles el ancho del documento coincide con el de la ventana;
no hay cookies ni contenido en los almacenes local y de sesión.

Este reinicio parcial no sustituye la comprobación final después del cierre.

## Pendiente

Registrar y resolver la aceptación de Antonio; propuesta, resolución,
descargas PDF/Word y tres firmas de la resolución en 2026/93001. Después,
incorporación del centro, alta en Personal, GINPIX manual, cese, vuelta a
Bolsa y cierre. Falta el reinicio final de aplicación y PostgreSQL y el
cotejo de los mismos recibos, fechas y recuentos.

## Capturas

- [Llamamiento vinculado y correo enviado](recorrido_f1_27_formalizacion_20261020/09_vinculo.png).
- [Mi Bolsa sin acción de aceptación](recorrido_f1_27_formalizacion_20261020/10_mi_bolsa.png).
- [Llamamiento tras reiniciar, a 390 píxeles](recorrido_f1_27_formalizacion_20261020/reinicio_390.png).
- [Resolución registrada en la semilla histórica distinta](recorrido_f1_27_formalizacion_20261020/13_semilla_distinta.png).
