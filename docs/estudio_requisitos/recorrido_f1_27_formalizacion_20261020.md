# Propuesta y resolución del caso 4.0: comprobación del 20/10/2026

Tarea F1-27. Ensayo en una copia local nueva de `main@fb5200b68`, con
datos ficticios y Chrome del sistema. No se ha tocado cidonia. El caso de
aceptación sigue siendo [FLUJO_APP, apartado 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).

El recorrido sigue abierto: el expediente 2026/93001 ha llegado a
Fiscalización, versión 6, con las dos firmas del informe y un llamamiento
de Antonio emitido y vinculado. Antonio ha aceptado desde Mi Bolsa, con
respuesta firme y recibo conservados tras reiniciar. Falta resolver esa
aceptación antes de comprobar los pasos 12 a 15 en el mismo expediente.

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
| 9, contacto | Ficha de Antonio, Registrar intento telefónico | 201 | Contacto telefónico ficticio declarado, con recibo, a las 11:00; abre el plazo de respuesta |
| 10 | Mi Bolsa de Antonio, Acepto y Enviar respuesta | `POST /api/vec/bolsa/mi-bolsa/respuestas` 201 | Respuesta firme a las 11:01:19, con recibo |
| 11 | Volver al expediente tras la aceptación | Lecturas 200 | Muestra Aceptación confirmada, pero no ofrece resolver ni preparar propuesta |
| 12–15 | Propuesta, resolución, documentos y firmas | No ejecutados en este expediente | Pendientes del paso 11 |

Las rutas abreviadas cuelgan de `/api/vec/contratacion-temporal`.
Las firmas tienen garantía sustancial y `firma_eficaz=false`, como
corresponde a los certificados de desarrollo. No acreditan firma oficial.

La primera consulta del original para Jefatura se interrumpió con
`ERR_NETWORK_CHANGED`, antes del registro. Una consulta posterior permitió
firmar y registrar con 201. Hay dos filas de firma, una por persona. No se
ha reproducido en este expediente el rechazo 403 investigado en F1-117.

## Bloqueo antes de la propuesta

El correo enviado no basta para abrir la respuesta en Mi Bolsa. La consulta
inicial de Antonio mostraba el envío de las 9:55, sin botón de aceptación.
Tras registrar el contacto telefónico ficticio declarado mediante «Registrar
intento telefónico», la pantalla ofreció aceptar o renunciar. Antonio
aceptó desde su área personal: respuesta firme, sin repetir el acto.
Este contacto declarado no acredita una llamada a una persona externa.

La operación genérica «Registrar contacto» había respondido 409,
`contacto_no_valido`, con el error de PostgreSQL «control de intentos
inválido». El mensaje visible «La clave corresponde a otro contacto» no
explicaba esa causa. Su botón queda tapado por el resumen lateral a 1440
píxeles; se envió mediante Tab y Enter. No se guardó ese contacto.
Dirección asignó su corrección a F1-123; la operación específica de teléfono
sí devolvió 201 y permitió continuar sin cambiar permisos ni reglas.

La ficha del expediente muestra ahora «Aceptación confirmada», el contacto
de las 11:00 y el recibo de respuesta de las 11:01. Sin embargo, sigue en
Fiscalización, versión 6, y no ofrece resolver la respuesta ni preparar
propuesta. «Llamamientos de este expediente» permanece vacío. La consulta
de comunicaciones tuvo un 503 transitorio y terminó con 200 y lista vacía.

El vínculo de la emisión externa no crea los antecedentes de selección,
comunicación y respuesta que exige la resolución de CT. Dirección indicó
continuar por «Llamar a la siguiente persona», cuyo montaje depende de
F1-103. Se ha pedido componer la siguiente candidata con esa entrega y la
activación de Personal de F1-119. No se han fabricado antecedentes ni recibos.

La resolución manual exige dos comprobaciones expresas de la técnica de
RRHH; no exige una segunda persona. La aceptación de Mi Bolsa se logró con
el perfil interno ya disponible, sin la activación externa de F1-115.
Los fallos de preferencias e imagen del área personal siguen separados.

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

Después de registrar la aceptación se hizo un segundo reinicio de la
aplicación y PostgreSQL. Las instantáneas completas de firmas y expediente
siguen idénticas. La tabla de respuestas conserva una única respuesta de
Antonio, con el mismo recibo, llamamiento, modo firme y fechas. Chrome
vuelve a mostrar «Acepto», «Respuesta firme» y el recibo de las 11:01.
No se repitió el envío de la respuesta.

En la comprobación anterior a 1440 y 390 píxeles el ancho del documento
coincidía con el de la ventana; no había cookies ni contenido en los
almacenes local y de sesión. El botón del contacto genérico presenta el
solapamiento descrito y asignado a F1-123.

Este reinicio parcial no sustituye la comprobación final después del cierre.

## Pendiente

Resolver la aceptación de Antonio; propuesta, resolución,
descargas PDF/Word y tres firmas de la resolución en 2026/93001. Después,
incorporación del centro, alta en Personal, GINPIX manual, cese, vuelta a
Bolsa y cierre. Falta el reinicio final de aplicación y PostgreSQL y el
cotejo de los mismos recibos, fechas y recuentos.

## Capturas

- [Llamamiento vinculado y correo enviado](recorrido_f1_27_formalizacion_20261020/09_vinculo.png).
- [Mi Bolsa antes de registrar el contacto efectivo](recorrido_f1_27_formalizacion_20261020/10_mi_bolsa.png).
- [Aceptación firme de Antonio y su recibo](recorrido_f1_27_formalizacion_20261020/10_aceptacion_firme.png).
- [Llamamiento tras reiniciar, a 390 píxeles](recorrido_f1_27_formalizacion_20261020/reinicio_390.png).
- [Resolución registrada en la semilla histórica distinta](recorrido_f1_27_formalizacion_20261020/13_semilla_distinta.png).
