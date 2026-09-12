# Manual funcional del técnico de Recursos Humanos

## Incorporación, GINPIX y borradores Word comprobados — 12 de septiembre de 2026

La base actual de este corte es `5ccfb967b75ad8ee604cd62f6648c61ab48b5f9b`.
El último runtime comprobado usa la web
`037b4226e979c755d47cb28591857ec03ca4d63c` y el binario
`1b21f31d299b94133e1e5b14b313d2e8fff0aef9909824813abfbd4a609d766e`.
Chrome recuperó la incorporación
y la ficha GINPIX con HTTP `200`.
No se repitió el alta: coinciden el recibo `ref:2bc3d281…`, la fecha
`2026-09-10T13:07:06.614186Z` y los seis campos cotejados. La ficha GINPIX tiene
SHA256 `4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`.

El detalle de trazabilidad conserva recibo y fecha visibles y puede abrirse y
cerrarse. El recorrido no tuvo errores JavaScript, cookies, almacenamiento ni
desbordamiento a 1440, 1024 o 390 px. No se repitieron los seis Word. En aquel
recorrido, el `GET` de preparación devolvió `404`; la comprobación vigente
aparece debajo y tampoco acredita que el cierre esté disponible.

En el runtime anterior `75157434…`, los seis documentos disponibles como PDF
se descargaron también en Word con HTTP `200`: informe definitivo, resolución, diligencia, toma de posesión,
notificación y comunicación al centro. Los seis ficheros DOCX son ZIP válidos y
su informe registró cero errores JavaScript. El recorrido de incorporación
registró además cero cookies y almacenamiento, y no mostró desbordamiento a
1440, 1024 o 390 px.

Los PDF y Word son borradores de desarrollo, sin firma, eficacia
administrativa, envío, entrega, transmisión o modelo oficial aprobado. La
agrupación a ancho completo de los seis pares está desplegada y visible. La
recuperación conservó los seis nombres, tamaños y SHA256, además de las huellas
de Personal y CT, sin repetir el `POST` de incorporación. Los objetivos 11 y
12 quedan acreditados funcionalmente tras reinicio, sin completar Contratación
ni los ocho hitos. Auth13 está instalada una sola vez; CT86/87 y AD3-30/31 se
instalaron después solo en los dos clones. La principal no recibió ese SQL.

El frontend publicado añade controles de paginación. **Siguiente** usa una vez el
cursor opaco recibido; **Reiniciar** vuelve a la primera página. El chip indica
los expedientes de esta página, no un total general. Si una lectura recuperable
falla, el cuadro visible se conserva. Las fechas civiles UTC se muestran sin
cambiar de día. El filtro admite como máximo 80 caracteres; si la entrada no
es válida aparece un aviso accesible y no se envía otra consulta. Esta mejora
superó 351 pruebas web y dos revisiones estáticas. En navegador, las 52 filas
cabían en el límite 100 y **Siguiente** quedó correctamente deshabilitado. El
filtro válido dejó una fila; una entrada de 81 caracteres mostró el aviso sin
otra consulta ni perder el cuadro, y limpiar volvió a responder `200`. No se ha
demostrado todavía el avance con cursor porque este conjunto ocupa una sola
página.

Las 60 fuentes SQL históricas añadidas en este corte quedan solo versionadas;
no se han ejecutado ni se debe aplicar `DOWN`. CT70–85 y Auth13 conservan su
historia instalada. Los ensayos posteriores de CT86/87 y la instalación de
CT90 se detallan debajo; la principal permaneció fuera de ese SQL.

La recuperación de cierre está integrada, revisada y visible. En el runtime
`037b…`, el `GET` de preparación daba `404`; la comprobación posterior descrita
debajo obtuvo `200` sin llegar al `POST`. **Preparar cierre** crea y muestra una solicitud inmutable y
ofrece descargar el archivo JSON sin enviar `POST`. Pulse **Guardar datos de
recuperación** para conservarlo antes de confirmar. Solo la segunda acción,
confirmada expresamente, registra el cierre. Para reanudar, importe el mismo
JSON: la pantalla coteja expediente y seguimiento contra el recibo del `GET`
original. **Recuperar** o **Completar** reutilizan exactamente la solicitud y
pueden terminar una operación pendiente. Nunca repiten la incorporación.

Los detalles de trazabilidad plegados mantienen visibles recibo, fecha, estado,
límites y período. Los cinco archivos de recuperación recibieron dos `GO`; los
otros cuatro, de trazabilidad, tuvieron revisión proporcional de dirección.
Las 378/378 pruebas web y los manifiestos 111/11/3/1 están verdes. CT86/87 se
instalaron solo en dos clones aislados; la base principal sigue pendiente y
la instrumentación CT87 bloqueada quedó congelada, sin acreditar un ensayo de
cierre.

La bandeja nominal en código exige identificar al técnico y limita sus dos
consultas a la intersección de organización y unidad, sin fallback. Sus 16
archivos recibieron dos `GO` estáticos; las focales de puertos/bootstrap,
`go vet ./...`, compilación y `go test ./...` global pasaron. Falta un E2E
con dos lectores. No añade SQL. Para el futuro correo se usará una cuenta remitente aún
por crear sobre una IP interna de la Diputación; el destino es el correo
obligatorio de un alta VEC existente. Faltan configurar servidor, puerto, TLS y
credencial reales.

El corte de Cobertura aún no está disponible en runtime. Su formulario exige
que RRHH elija **Bolsa**, **SAE** o **Nueva convocatoria**. La recomendación y
su motivo se muestran para apoyar la decisión, pero no eligen por el técnico.
Después de recargar, continúe con el formulario de asignación recuperado; no
registre una nueva reasignación. La v1 se conserva como historia y la v2 corrige
las secuencias 3/4 con el diccionario, DTO y traductor reales.

Las 20 fuentes recibieron dos `GO`; la web terminó 385/385 `PASS` y los
manifiestos 111/11/3/1 pasaron; Go global, vet y build terminaron con código 0.
El binario candidato no está desplegado y el E2E de las tres vías sigue
pendiente. El lector actual solo admite la unidad RRHH y al
técnico de su organización, sin prueba de dos unidades ni cambio de centro.
CT87 ya está instalada y no se reaplica. Su publicación en clon falló por
precedencia JSON; cinco snapshots posteriores quedaron idénticos al estado
inmediatamente anterior al intento, sin tres `INSERT` persistidos. CT90 se
instaló después únicamente en los dos clones, como se detalla debajo. El bloqueo de instrumentación
fue una comprobación separada.

CT90 corrige únicamente tres paréntesis de `cierre87_validar_sucesora`. Sus dos
revisiones y la prueba real —un positivo, nueve negativos y la sucesora
operativa— pasaron. Está instalada una sola vez en cada clon, no en la base
principal. Los recibos CT86 y CT87 conservan sus SHA256; no reaplique CT86,
CT87 ni CT90. La recuperación `e359…` dejó tres filas correctas. Go global y vet
de continuidad terminaron en `PASS`; la evidencia privada queda custodiada sin
exponer rutas.

En la comprobación posterior, la anotación administrativa respondió `201`,
mostró su recibo y actualizó el expediente de v8 a v9 conservando el seguimiento
original v1. **Recuperar** respondió `GET 200` con el mismo recibo. La lectura
SQL confirmó una anotación, una versión, una actuación y
un outbox, sin pérdida de historia ni cambios en Personal o incorporación.
**Preparar cierre** respondió `GET 200`; se guardó el JSON y un único `POST` de
cierre devolvió `400`. Conserve la solicitud original y la clave `aed453da…`
para un eventual replay exacto; no repita el envío mientras continúa el
diagnóstico. La lectura SQL encontró las 14 filas de negocio iguales al estado
posterior a la anotación y cero cierres. La allowlist HTTP admite solo `Accept`
y `Content-Type` y rechazó el `User-Agent` de Chrome; los campos y la clave son
válidos, pero la corrección espera dos revisiones. El reinicio posterior sigue
pendiente; el `503` anterior queda solo como antecedente.

E08 aporta traducciones comunes de anotación y cierre. El motivo comprensible
se muestra sin cambiar el payload, y los mensajes configurados se propagan
escapados. Las dos fases y el replay se mantienen. Sus siete archivos JS y
pruebas fueron revisados y 21 focales pasaron, pero esta mejora todavía no está
desplegada. La campaña global terminó 365/365 pruebas CT más 23/23 del
coordinador, 388 en total, y manifiestos 11/111/3/1 en `PASS`. No hubo cambios
Go ni se repitió su campaña global.

La prueba CT86 quedó en `NO-GO`: la guarda exige 10 s y la capacidad real es 5
s. No se ejecutó el arnés SQL y el negocio permaneció intacto.

El lector DER también permanece desactivado. Su bootstrap completo y vet
pasaron, pero la activación principal de las 16:00 rechazó el perfil PEM previo
al bootstrap. Sistemas restauró solo el binario y material propios y recuperó
salud `200`, sin restaurar la base. El material corregido pasó 11 pruebas y el
cargador real; el reintento está pausado por material compartido. El correo
`78c209…` permanece privado por `NO-GO`. Siguen pendientes roles completos,
auditoría, hexagonalidad e i18n; estos verdes no acreditan conformidad ni E2E.

## Historia del corte para presentación — 10 de septiembre de 2026

Este apartado y los bloques cronológicos inferiores se conservan como historia;
el recorrido vigente es el descrito al inicio de este manual.

### Panel integrado en código; runtime pendiente

El panel `ORIGINAL` integra la consulta por referencia de expediente sobre el
backend confirmado
`76a2b7c13309bc0526e45d585c2fdf7ca77aecf6`. Su contrato es
`GET /api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=<referencia>`.
La respuesta muestra `original_incorporacion`, que es la incorporación
histórica de origen. No indica por sí sola el estado actual del expediente.

Cuando la ruta esté disponible en runtime, RRHH abrirá el panel con su perfil
autorizado, introducirá una referencia sintética existente y consultará el
seguimiento. No debe crear otra incorporación para recuperar el caso. Si el
servicio devuelve indisponibilidad, conserve la referencia y comuníquelo a
Sistemas; no cambie datos o permisos para forzar la consulta.

La evidencia actual es focal: seis pruebas Go verdes en cinco paquetes y Node
`7/7`, incluido el montaje principal.

La prueba aislada del componente de seguimiento, con los estilos de producto y
transporte sintético interceptado, mostró las referencias completas y el botón
visible sin desbordamiento a 1440 y 390 px. Dirección inspeccionó la captura
móvil, legible y sin superposición. Esta comprobación no abarca el shell
completo ni el runtime. La campaña global
`go test ./...` no se ejecutó porque la revisión automática rechazó su alcance
masivo.

Auth13 `ef6704a6` no está instalado y la recuperación `GET` sigue bloqueada
en runtime. GINPIX `e2831250` está integrado, sin prueba sobre un recibo
recuperado. No hay nueva respuesta de GINPIX, anotación, cierre ni recorrido
conjunto acreditados.

Siguen abiertos los objetivos 11 (recuperación), 12 (GINPIX en runtime), 13
(anotación y cierre) y 14 (conjunto). En el objetivo 6, el inicio y la política
siguen pendientes de elección y acreditación. Esta copia parte de `3375d527`.

Cierre funcional `00558603`: 5/8 pasos completos más partes del sexto y séptimo.
Se conservan 52 expedientes sintéticos en el servidor privado. El
[manual de Sistemas](../manual_sistemas/README.md#entorno-privado-vigente)
describe el acceso; los puertos locales del historial inferior no son una
dirección operativa en el equipo del visitante.

La resolución tiene ahora **validación manual de ejercicio con recibo**.
Desde el detalle del expediente con propuesta, RRHH revisa el material,
marca las dos comprobaciones y confirma expresamente. El caso demostrado
pasa de propuesta `v7` a resolución de ejercicio `v8`; recuperación tras
reinicio y repetición con la misma clave conservan recibo, fecha y actuación.

En un expediente ya validado, enseñe su recibo y descargue los seis documentos
desde el detalle. Son los borradores históricos de la propuesta, que se
mantienen accesibles; la validación no los transforma en documentos oficiales.
El expediente original sin esa validación se ha conservado.

Para la presentación distinga siempre:

- Identificación por certificado de pruebas: permite acceder según permisos;
  no equivale a firma documental.
- Validación manual: registra una actuación sintética; no constituye firma,
  nombramiento eficaz, envío ni orden de incorporación.
- Incorporación: registro real de desarrollo ya logrado mediante formulario,
  con dos confirmaciones, recibo y persistencia en CT y Personal; la recuperación
  tras reinicio sigue pendiente y todavía no cierra el octavo paso.

El caso sintético registrado corresponde a solicitud `7`, expediente `8`,
periodo `2027Q1` y seguimiento `0→1`; no tiene firma oficial ni eficacia
administrativa. Tras reiniciar aplicación y PostgreSQL, la lectura quedó en
`GET 503` durante la restauración histórica de autorización, por lo que no se
debe crear otra incorporación para probarla. GINPIX sigue como siguiente corte
sin cierre visible.

Para uso real siguen pendientes los modelos y circuito de firma admitidos,
la comunicación corporativa, las reglas de plazo y las autorizaciones de
datos y operación. No se asignan esos efectos a una casilla de la presentación.

## Antecedentes hasta el 6 de septiembre

Las cifras e instalaciones siguientes son históricas. El corte anterior
describe lo vigente; no hay que repetir actuaciones para actualizarlo.

Contratación temporal · VEC Diputación · Corte: 6 de septiembre de 2026.

**Consulta de centros:** desde la bandeja, **Centros y organización de
referencia** abre una tabla con búsqueda, tipo, adscripción y página de la RPT.
La [guía](../../GUIA_RECORRIDO_ALBERTO.md#centros-y-organización-de-referencia)
detalla su cobertura: 41 centros, agrupaciones de la fuente y once puestos de
Transformación Digital, todavía sin las dependencias funcionales completas.
Una adscripción no identifica al ocupante ni a quien puede ratificar.
Consultar este catálogo no amplía los centros admitidos por el formulario
actual; la ratificación multicientro sigue pendiente.
En la instancia principal, **Nueva unidad** y **Editar** guardan denominación,
tipo y adscripción con motivo, revisión y recibo. Puede preparar niveles
distintos según el área; no se exige que exista director, subdirector o jefe
de servicio en todas. Un cargo configurado no identifica a su ocupante ni
autoriza una petición. La guía recoge el alta/edición sintética y la recuperación
del mismo recibo tras reiniciar, sin duplicados.

**Cierre de bandeja y análisis publicado:** `b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`.
La base principal conserva 51 solicitudes; bandeja y detalle consultables (`8443` / base
`55433`). El caso verificado encadena solicitud `v1` a análisis `201`/`v2` y
recupera el recibo único tras reinicio, mediante lectura independiente de
PostgreSQL y navegador.
El cierre de bandeja no incrementa el contador de pasos.
La [guía canónica](../../GUIA_RECORRIDO_ALBERTO.md) contiene el
recorrido y comandos exactos.

**Incluido en esta entrega; recuperación demostrada:**
declaración RRHH registrada mediante `HTTP 201`. Corregida la comparación de
fechas con ceros finales, dirección confirmó en navegador `200/200/200` para
recuperar selección, comunicación y respuesta tras el segundo reinicio de
aplicación y PostgreSQL. Mismos recibo, justificante y fecha, sin nuevo registro.
También se confirmó un conflicto real `409` en navegador. La entrega 3 queda
cerrada funcionalmente en desarrollo; no aumenta el contador.

**Corte 4 publicado en `17ea874`:** aceptación manual sintética `201`;
tras reiniciar aplicación y PostgreSQL principal, navegador `200/200/200/200`,
mismo recibo y fecha, sin duplicados. Cierre técnico, no aprobación de política
legal ni del procedimiento por RRHH. Aquel corte mantenía **5/8 pasos más parte del 6**.

**Corte 5 incluido en esta entrega:** renuncia manual sintética registrada;
navegador real `200/201/201/201` y, tras reiniciar aplicación y PostgreSQL principal,
`200/200/200/200`, mismos recibo, resolución, auditoría, fecha e intención pendiente.
Sin duplicados, errores JS, cookies, almacenamiento web ni desbordamiento.
Objetivo 5 cerrado funcionalmente solo en desarrollo, sin aval legal ni del operador.

**Objetivo 7 incluido en esta entrega:** continuación tras esa renuncia, quinta
operación `201` real y cinco `200` tras reiniciar app/PostgreSQL principal; mismos
recibos/fecha, sin duplicados, errores JS, cookies, almacenamiento web ni desbordamiento.
En el corte `9ccef45` todavía no se registraba el aviso local al sucesor.

**Objetivo 8 cerrado funcionalmente en desarrollo:** propuesta desde aceptación,
Chrome `201` y, tras reiniciar app/PostgreSQL principal, cuatro antecedentes `200`
y propuesta `200`, mismos propuesta/recibo/fecha/v7, sin duplicados. Cero errores
JS, cookies, almacenamiento web y desbordamiento. Sin firma ni nombramiento eficaz.
Esta revisión incorpora el cierre funcional; el hash publicado se comprueba en Git.
[Caso y claves](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-8-recuperar-la-propuesta-de-nombramiento).

**Objetivo 9: primer informe borrador descargado**, Chrome `200`, 29267 bytes,
PDF y pantalla inspeccionados por dirección. Rectificación: primer fallo navegador
sin estado HTTP capturado; el `502` era curl con límite 50, paginación separada sin
corrección acreditada. Navegador límite 100: sonda `200`, vista `404`, por comparación
textual de fechas equivalentes. AD3-21 corrige esas dos lecturas, instalada en ambas bases.
Tras el parche, cinco POST de bandeja/detalle/PDF `200`, mismo PDF y cero errores JS,
cookies y almacenamiento web. Tras reiniciar app/PostgreSQL principal: cinco POST `200`,
sin `404`, mismo PDF e historia CT/Bolsa conservada; no cierra la paginación con límite 50.
Pantalla estable de 390 px observada por dirección sin obstrucción del detalle ni botón;
captura previa durante transición de 180 ms, sin modificar UI ni validar usabilidad global.
Esta revisión añade resolución borrador: Chrome `200`, 29770 bytes, informe original
idéntico en la misma sesión; seis POST `200`, cero errores JS, cookies y almacenamiento web.
Dirección inspeccionó PDF y pantalla estable de 390 px con dos botones. Tras reiniciar
aplicación/PostgreSQL principal: otros seis POST `200`, ambos PDF idénticos en tamaño
y SHA256, historial y recibos anteriores conservados; cero errores JS, cookies y
almacenamiento web. Tercer borrador, diligencia: 28366 bytes; siete POST `200` antes
y después de reiniciar aplicación/PostgreSQL principal, los tres PDF e historial previo
idénticos, cero errores JS, cookies y almacenamiento web. Dirección inspeccionó PDF
y pantalla estable de 390 px con tres botones. Toma de posesión comprobada también tras
reinicio principal: ocho POST `200`, cuatro PDF e historial idénticos; evidencia en la guía.
Disponibles **6/6**: comunicación al centro comprobada también tras reinicio principal.
Siguiente objetivo 10, circuito de firma/evidencia admitida. Sin envío, entrega, plazo legal
ni orden de incorporación; evidencia central en la guía.
Las descargas no añaden SQL propio ni otro paso completo.

## Qué puede hacer hoy

**El último recuento formal conserva cinco de los ocho pasos demostrados en
desarrollo, más partes del sexto y séptimo.** Las partes acreditadas del octavo
se describen debajo sin recalcular ese recuento. Se usa la aplicación
conectada a PostgreSQL:
los recibos descritos son persistentes, pero los datos, catálogos y fuentes
del recorrido son sintéticos. No es una habilitación para tramitar datos
reales ni una aceptación funcional de Recursos Humanos.

| Paso del procedimiento | Situación del recorrido |
| --- | --- |
| 1. Solicitud | Demostrado: formulario y recibo de alta. |
| 2. Análisis de RRHH | Demostrado con fuentes sintéticas y recibo. |
| 3. Bolsa / vía de cobertura | Demostrado para la decisión de usar Bolsa vigente; no cierra toda la gestión de Bolsa. |
| 4. Asignación | Demostrado: unidad, responsable y recibo. |
| 5. Informe jurídico y Fiscalización | Demostrado: documento de desarrollo sin firma, resultado de Intervención y devolución a la unidad cuando es desfavorable. |
| 6. Llamamiento | Parcial: recorridos sintéticos, aviso CT62, declaración CT63 y aceptación manual del sucesor CT64 recuperables tras reinicio principal. Faltan vencimiento, envío corporativo y plazo; no acredita entrega ni plazo legal. |
| 7. Nombramiento / formalización | Parcial: propuesta desde aceptación sintética `201` y recuperación `200` tras reinicio, expediente `6→7`; seis borradores descargables y validación manual sintética `7→8` con recibo recuperable. Sigue pendiente el circuito de firma oficial; no posesión real, nombramiento eficaz, envío ni entrega. |
| 8. Incorporación, GINPIX y seguimiento | Parcial comprobado: la incorporación y Personal se recuperan por `GET` tras reiniciar aplicación y PostgreSQL, con recibo y fecha originales; la ficha manual GINPIX responde `200` y conserva seis campos y SHA256. No acredita transmisión o confirmación del destino; anotación y cierre siguen pendientes de instalar y recorrer. |

La **bandeja de expedientes está cerrada para este alcance de desarrollo**:
la base del servidor conserva 52 expedientes sintéticos, con bandeja y detalle
consultables por el acceso privado preparado por Sistemas. Esto no convierte
los datos sintéticos en expedientes reales ni añade otro paso completo.
Para recuperar el llamamiento se usan las referencias conservadas
en la guía.

Este manual explica el trabajo funcional. El
[manual de Sistemas](../manual_sistemas/README.md#entorno-privado-vigente)
describe el acceso y operación actuales. La
[guía del recorrido manual](../../GUIA_RECORRIDO_ALBERTO.md) conserva los
datos sintéticos, recibos y pruebas anteriores; sus recetas locales no son
instrucciones de arranque del servidor.
El [manual de usuario de Bolsa](../manual_usuario/manual_portal_bolsas.md)
complementa la navegación de ese módulo; sus pantallas de demostración no
acreditan funciones reales de Contratación temporal.

## Presentación funcional y límites de uso real

La presentación se recorre en la misma aplicación final, con perfiles y datos sintéticos; no se bloquea por correo corporativo, firma o GINPIX.

| Operación de presentación | Qué hace realmente | Qué falta para uso real |
| --- | --- | --- |
| Acceso con certificado | Autentica el perfil de desarrollo frente a la aplicación; **no es firma de documento**. | Identidad corporativa y permisos del personal real. |
| Aviso local | Persiste un aviso e intención recuperables en VEC; no envía correo ni acredita entrega o plazo. | Canal corporativo admitido, entrega acreditada y reglas de plazo. |
| Descarga de borradores | Genera documentos preparatorios para revisar; no son documentos oficiales ni nombramiento eficaz. | Plantillas oficiales y firma con certificado o servicio corporativo admitido, según el circuito que se determine. |
| Salida GINPIX | Hay piezas de exportación, pero no se acredita aquí una salida completa desde el portal ni una transmisión. | Conexión o carga autorizada, confirmación del sistema destino y seguimiento. |

La [petición y ratificación del centro](../../GUIA_RECORRIDO_ALBERTO.md#petición-del-centro-y-ratificación)
ya se recorre con dos identidades sintéticas y recibos recuperados tras reinicio.
La [entrega al expediente de RRHH](../../GUIA_RECORRIDO_ALBERTO.md#entregar-la-petición-ratificada-a-rrhh)
también está demostrada, con el mismo recibo tras reiniciar aplicación y base,
sin otra alta. Ratificar no equivale a firmar. Si consta **Expediente creado**,
consulte ese recibo; no repita la entrega para presentar el caso.

## Antes de actuar

1. Pida al operador el entorno de desarrollo preparado según la guía. Entre
   en **Portal del Empleado → Contratación temporal** con el certificado
   asignado al perfil de prueba. No use un entorno o certificado de producción.
2. Compruebe el perfil efectivo y el expediente. El perfil RRHH no permite
   actuar como Intervención; un campo de formulario no cambia los permisos.
3. Use exclusivamente datos sintéticos y opciones de los catálogos mostrados.
   No introduzca nombres, documentos identificativos, correos ni teléfonos reales.
4. Antes de confirmar, revise expediente, versión, decisión y referencias.
   Pulse una sola vez y conserve el recibo. Un mensaje de espera, un documento
   preparado o un color verde no sustituyen al recibo de confirmación.

El acceso de desarrollo mediante certificado no acredita identidad
corporativa ni firma administrativa. No se guardan credenciales ni datos de
trabajo en cookies o almacenamiento web; cerrar una pestaña puede perder lo
introducido que aún no esté confirmado.

## El procedimiento objetivo y quién decide

Esta tabla describe el objetivo administrativo, no funcionalidades ya
disponibles. Se conserva la agrupación de ocho pasos de la guía; la
numeración del documento original respecto a GINPIX tiene una ambigüedad
pendiente de aclaración con RRHH.

| Paso | Responsable y entradas | Decisión y salida esperada |
| --- | --- | --- |
| 1. Solicitud | Centro solicitante; necesidad, categoría, fechas, contacto y documentación presupuestaria disponible. | Solicitar cobertura. Expediente identificado y recibo de alta. En la prueba registra el perfil RRHH. |
| 2. Análisis | RRHH; solicitud y fuente de retención de crédito. | Validar modalidad, causa, fechas, jornada y crédito. Análisis motivado y recibo. |
| 3. Bolsa / cobertura | RRHH; análisis y comprobaciones procedentes de Bolsa u otras fuentes competentes. | Elegir y motivar la vía viable. Decisión con referencias de fuente y recibo. |
| 4. Asignación | RRHH; expediente con cobertura decidida. | Asignar unidad y responsable competentes. Recibo de asignación y trazabilidad del destino. |
| 5. Informe y Fiscalización | Unidad gestora para el informe; Intervención para fiscalizar. Expediente asignado y documentación. | Informe y resultado favorable, favorable con observaciones o desfavorable. Recibos separados; devolución motivada cuando corresponda. |
| 6. Llamamiento | RRHH y Bolsa; expediente habilitado, lista y reglas vigentes. La persona candidata responde. | Propuesta ordenada, llamamiento, comunicación y respuesta acreditada. Recibos diferenciados de selección, entrega y aceptación o renuncia. |
| 7. Nombramiento | Unidad de formalización y firmantes competentes; propuesta aceptada y documentación exigible. | Preparar, revisar, firmar y registrar los documentos que correspondan. Documentos versionados y evidencias de firma y registro. |
| 8. Incorporación y seguimiento | Centro, Personal y operación autorizada de GINPIX; formalización válida y hechos de incorporación. | Confirmar incorporación, transmitir o cargar la ficha y seguir la relación hasta su cierre. Confirmaciones de cada sistema, no solo un fichero generado. |

Bolsa conserva la autoridad sobre sus integrantes, disponibilidad, orden y
reglas. RRHH no reordena personas desde este formulario ni calcula un baremo
alternativo. Desempates, exclusiones y plazos dependen de las bases y fuentes
aplicables: este manual no fija puntuaciones, sanciones ni horas de respuesta.
Si falta una fuente, el resultado queda pendiente o indeterminado; no se
interpreta como bolsa agotada, cero puntos o autorización para saltar un orden.

## Recorrer los formularios disponibles

### 1. Registrar la solicitud

1. Abra **Nueva petición**. Seleccione centro, contacto referenciado,
   categoría, grupo o subgrupo y motivo; complete detalle y fechas.
2. Declare la situación de la retención de crédito, es decir, la reserva
   presupuestaria. Use el ejemplo sintético de la guía, sin adjuntar documentos reales.
3. Pulse **Revisar solicitud → Confirmar y registrar**.

Salida: **Solicitud registrada**, número visible, referencia de expediente,
versión inicial `1`, referencia de recibo y fecha. La aplicación abre el
análisis. Un duplicado no se resuelve creando otra clave: consulte primero el
resultado existente. El alta confirmada no significa crédito validado.

### 2. Registrar el análisis de RRHH

1. En **Análisis por Recursos Humanos**, revise modalidad, categoría,
   grupo, causa, fechas y jornada; elija la fuente sintética de crédito.
2. Pulse **Registrar análisis**.

Salida: mismo expediente, versión resultante `2`, recibo y fecha. La guía
acredita el ejemplo de **Sustitución**; que aparezcan otras modalidades en el
catálogo no acredita todos sus recorridos. Si faltan datos o la fuente no es
válida, no avance suponiendo crédito disponible.

### 3. Decidir la vía de cobertura

1. Espere la propuesta en **Decidir la vía de cobertura**. Compruebe su
   viabilidad y la vía recomendada; el ejemplo demostrado usa **Bolsa vigente**.
2. Revise la decisión y pulse **Confirmar vía de cobertura**; confirme el diálogo.

Salida: recibo de decisión, mismo expediente y versión `3`. Todavía no hay
persona aceptante ni comunicación entregada. Si la propuesta no puede
determinarse, no elija una vía inventada para continuar.

### 4. Asignar a la unidad responsable

1. En **Asignar expediente a la unidad responsable**, compruebe unidad y
   persona responsable propuestas.
2. Marque la comprobación y pulse **Confirmar asignación**.

Salida: recibo, fecha y versión `4` del mismo expediente. El recibo acredita
la asignación persistida, no la apertura de la bandeja por su destinatario.
No sustituya referencias para sortear una denegación o un ámbito incorrecto.

### 5. Preparar el informe y remitir a Intervención

1. En **Preparar informe jurídico**, compruebe expediente y versión `4`.
   Acepte expresamente que se generará un documento de desarrollo sin firma
   y pulse **Confirmar y preparar informe**.
2. Conserve el recibo, la referencia del documento y la versión `5`.
   El contenido indica **DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA**.
   No lo use como informe administrativo firmado.
3. Intervención accede con su perfil separado, introduce la referencia del
   expediente y versión `5`, y pulsa **Abrir fiscalización**. Selecciona el
   resultado en **Registrar resultado de Fiscalización** y confirma mediante
   **Registrar resultado**.

Salida: recibo de Fiscalización y versión `6`. **Favorable con
observaciones** y **Desfavorable** exigen observaciones. El desfavorable
devuelve el expediente a la unidad en incidencia, conservando el historial:
no permite continuar al llamamiento ni demuestra que toda la subsanación
posterior esté disponible. Para el tramo demostrado del paso 6 se parte de
un expediente con resultado **Favorable**.

### 6. Seleccionar, registrar el aviso local y declarar la respuesta

Entrada: expediente fiscalizado favorablemente en versión `6`. Para
recuperar el caso ya demostrado, use los datos exactos del apartado 6 de la
guía; no registre otra solicitud ni otra fiscalización.

1. Con perfil RRHH abra **Nueva petición → Llamamiento y comunicación**.
   Compruebe expediente, versión y clave de operación, y pulse
   **Revisar e iniciar llamamiento**.
2. Conserve el recibo de selección. Organización, llamamiento, versión `1`
   y recibo antecedente de comunicación se rellenan desde ese resultado real:
   no los invente ni los sustituya.
3. En **Registrar comunicación**, use la clave correspondiente a esa
   comunicación y pulse **Revisar y registrar comunicación**.

Salida: recibo de selección y recibo de comunicación local, con fecha y
referencia de intención de aviso. La fuente de Bolsa del ejemplo es sintética
firmada; la orden, el llamamiento y el registro local sí quedan persistidos.

| Lo que muestra o conserva | Lo que acredita — y lo que no |
| --- | --- |
| Recibo de selección | La selección confirmada y su llamamiento. Es el antecedente del aviso, no prueba de entrega. |
| `registrada_localmente` | Registro local confirmado. No correo enviado ni recibido. |
| `replay_registrada_localmente` | Recuperación del mismo registro, recibo y fecha; no un segundo aviso. |
| Intención de aviso | Registro persistente pendiente de salida. No certifica entrega. |
| Fichero en el subdirectorio `comunicaciones` del material de desarrollo | Aviso sintético recuperable: «Aviso de desarrollo, no enviado, no abre plazo». Lo comprueba el operador según la guía; no se presenta como descarga de la interfaz. |

La versión local resultante `2` de comunicación no es la versión del
expediente. **Registrar el aviso local no abre plazo legal, no acredita
aceptación y no permite dar por realizado el nombramiento.** No hay recorrido
completo de envío, entrega o vencimiento. La aceptación y la renuncia manuales se
limitan al ejercicio sintético descrito debajo. No use las
opciones de una pantalla de demostración para simular esas decisiones.

#### Operación 3: registrar la respuesta declarada por RRHH

En el **mismo formulario**, tras recuperar el recibo confirmado de comunicación
en versión `2`, aparece **3. Registrar respuesta recibida por correo**. Sus
referencias proceden de ese recibo. Requiere el permiso específico
`contratacion_temporal.llamamiento.respuesta.registrar`; poder registrar un
aviso no concede este permiso.

1. Seleccione **Aceptación declarada en el correo** (`aceptacion`) o **Renuncia
   declarada en el correo** (`renuncia`). Indique una referencia opaca del correo,
   sin direcciones ni datos personales, y la fecha de recepción **en UTC**.
2. Seleccione un `.eml` no vacío de hasta **2 MiB**. WebCrypto calcula SHA-256
   localmente; el contenido nunca se envía ni se guarda en VEC. La huella no
   admite edición manual. **Huella calculada** sigue siendo válido aunque el
   selector quede vacío al actualizarse la pantalla; no se muestra el nombre.
3. Revise y confirme expresamente la declaración. Conserve datos, clave y
   recibo; ante resultado ambiguo o acceso denegado, no
   cambie la clave ni el material para forzar otro registro. Avise al operador.

Ejemplo reproducible: [respuesta_sintetica.eml](ejemplos/respuesta_sintetica.eml),
**nunca enviado ni correspondiente a una persona real**. SHA-256 del archivo:
`7984edfd3ba13c87b0c04160dbfa8b338b356ead70d80df04066e67e4ed419b9`.
Para recuperar la declaración existente use los datos y la clave de la
[guía canónica](../../GUIA_RECORRIDO_ALBERTO.md), no una operación nueva.
Este archivo corresponde al caso de aceptación. La renuncia usa su propio
correo sintético conservado por el operador, con la huella y claves de la guía;
no sustituya un correo por el otro ni reconstruya su contenido para recuperar.

El servidor conserva actor, declaración y recibo. El justificante enlaza la
referencia y la huella declaradas: **no conserva el correo ni verifica origen,
firma o custodia**. El original sigue en el sistema de correo. No acredita
envío, entrega ni aceptación o renuncia terminal; no cambia candidatura ni
estado Bolsa, y no avanza expediente.

Registro sintético observado por dirección (`HTTP 201` y navegador):

- Recibo: `9e14599d-2edc-42aa-afde-170420c838aa`.
- Justificante: `84727d1d-31ef-4fde-92c8-d3a8e2953931`.
- Registrada en UTC: `2026-09-05T18:09:06.065542Z`.
- Misma comunicación `v2` y expediente `v6`, sin transición en Bolsa.

Tras corregir el defecto temporal y repetir el reinicio de aplicación y
PostgreSQL, el navegador recuperó selección, comunicación y respuesta con
`200/200/200`. **Mismos recibo, justificante y fecha anteriores, sin nuevo
registro**. El recorrido quedó comprobado sin errores JavaScript, cookies,
almacenamiento web ni desbordamiento horizontal. Cierra esta entrega de
declaración, no la aceptación terminal ni todo el paso 6.

#### Operación 4: aceptación o renuncia manual del ejercicio sintético

Después de recuperar una declaración `aceptacion` o `renuncia`, aparece **4. Solicitar
resolución de respuesta** en el mismo formulario. Reutiliza las referencias
originales, recibo de comunicación `v2` y justificante; no permite cambiar sus
antecedentes, elegir otra respuesta ni conceder identidad. Use una clave propia;
si recupera, la original.
Solo tras comprobar el caso, marque las dos casillas inicialmente vacías:

- **He comprobado la respuesta y su justificante**.
- **Para este ejercicio sintético, he comprobado que la respuesta llegó dentro del plazo del ejercicio**.

El criterio fijo, de solo lectura, es `politica:ct:revision-manual-sintetica:20260906`.
**Validación manual de desarrollo: no acredita entrega de correo ni plazo legal real**.
Revise y confirme expresamente, sin otro `.eml`. El servidor conserva declaración,
actor y política con permisos propios; solo tras CT y Bolsa muestra
**Aceptación registrada · ejercicio sintético** o **Renuncia registrada · ejercicio
sintético**, según el recibo antecedente. La declaración del corte 3 no basta.
La renuncia añade referencia, fecha UTC y **Siguiente candidato pendiente**;
no confirma selección ni aviso a otra persona.

Dirección confirmó `201` con API/V3/CT58/Bolsa4 reales y `200/200/200/200`
tras reiniciar aplicación y PostgreSQL principal, sin duplicados:

- Recibo CT: `recibo:d6bdcc7b-e22e-4fe9-8aac-a1eb554a4103`.
- Resolución: `7a3e4a2e-d142-4562-ae5c-59c95b011e0c`.
- Fecha UTC conservada: `2026-09-05T22:27:02.861379Z`.

Para renuncia, dirección confirmó navegador real `200/201/201/201` y recuperación
`200/200/200/200` tras reiniciar app/PostgreSQL principal. Otro expediente,
`fe4934a1…`, fiscalizado `v6`, conserva:

- Recibo CT: `recibo:408fda57-638d-4a3b-a441-4ef56396e23a`.
- Resolución: `resolucion:0c5fdea4-be11-4bdd-bd0b-5bc035dd9ae0`.
- Fecha UTC: `2026-09-05T23:00:45.289468Z`.
- Intención: `intencion:f4bd0049-8b96-410b-8144-4384bdb47ed0`, pendiente.

Tras reinicio se conservan esos datos, auditoría y carga de intención en la misma
fila CT, sin duplicados ni siguiente ejecutado. Aceptación y declaración anteriores
intactas. La [guía, caso de renuncia](../../GUIA_RECORRIDO_ALBERTO.md#corte-5-recuperar-la-renuncia-manual-sintética)
contiene expediente completo, cuatro claves y correo/huella/fecha originales.
El método es manual provisional solo para desarrollo sintético, no aval del operador.

Sin ambas casillas no se envía. La petición antigua sin revisión manual sigue en
`409` pendiente, sin efectos: permite corregir casillas conservando la clave.
Ante resultado ambiguo, conserve congelados clave y material; no hay reintentos automáticos.
Faltan vencimiento, envío corporativo y plazo legal. La propuesta de desarrollo
se describe a continuación; no hay política legal aprobada.
La [guía canónica](../../GUIA_RECORRIDO_ALBERTO.md) conserva el recorrido exacto.

#### Operación 5: continuar tras la renuncia sintética

Recupere primero la renuncia. El mismo panel deriva expediente, resolución e
intención; use la quinta clave original de la [guía](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-7-recuperar-la-continuación-tras-renuncia).
Confirme expresamente que abrirá un único nuevo llamamiento, sin otro `.eml`.
El `201` muestra **Siguiente llamamiento abierto · ejercicio sintético**:
recibo CT `recibo:b5bb611f-0126-4806-90c5-85b9f9b63778`, fecha UTC
`2026-09-05T23:57:11.037866Z`. Tras reiniciar app/PostgreSQL principal, cinco `200`;
mismos 14 campos salvo `estado_local: replay_confirmado`, sin duplicados.
La intención `despachada` del nuevo recibo acredita continuidad posterior;
el `pendiente` del recibo de renuncia queda histórico, sin modificar sus bytes.
No acredita envío, entrega ni aceptación; no sustituya la comunicación antecedente
por este nuevo llamamiento. El éxito desactiva otro envío. Ante resultado ambiguo
conserve clave/material, sin reintentos automáticos ni otra clave para eludir `409`.

#### Operación 6: registrar el aviso local al sucesor

Tras recuperar la continuación, use **6. Registrar aviso local al sucesor** en el mismo panel.
Referencias y versión `1` vienen de su recibo; solo introduzca la sexta clave original
de la [guía](../../GUIA_RECORRIDO_ALBERTO.md#aviso-local-al-sucesor-ct62) y confirme expresamente.
`201` muestra **Aviso local al sucesor registrado · No enviado**; tras reinicio principal,
`200` recupera la misma comunicación, recibo, fecha, versión `2` e intención local, sin duplicados.
No cambia los recibos previos; habilita la declaración manual del paso 7, no una resolución. No envía ni abre plazo.
Conserve clave/material ante ambigüedad; no eluda un `409` con otra clave.

#### Operación 7: declarar la respuesta del sucesor, sin resolverla

Tras recuperar el aviso local `v2`, use **7. Registrar respuesta recibida del sucesor**.
El panel deriva sus antecedentes; introduzca clave propia, respuesta declarada, referencia opaca de correo,
fecha UTC y `.eml` hasta 2 MiB para huella local; confirme expresamente. No se conserva el contenido.
[Clave, archivo exacto y recibo](../../GUIA_RECORRIDO_ALBERTO.md#declaración-de-respuesta-del-sucesor-ct63).
Registro real `201`; siete operaciones `200` tras reinicio principal, mismos justificante/recibo/auditoría/fecha,
sin duplicados. Aceptación solo declarada, no resuelta ni verificada en origen/firma/plazo.
No cambia los recibos ni la resolución anterior. Mantenga clave/material ante ambigüedad; `409` no autoriza otra clave.
La resolución requiere la octava operación separada, no es automática.

#### Operación 8: resolver manualmente al sucesor, solo ejercicio sintético

Recupere las siete operaciones previas y use la octava con la clave original de la
[guía](../../GUIA_RECORRIDO_ALBERTO.md#resolución-manual-del-sucesor-ct64).
Respuesta, justificante y referencias se derivan, no se editan. Marque expresamente las dos revisiones
de respuesta/justificante y plazo del ejercicio, inicialmente vacías, y confirme; sin otro `.eml`.
Solo tras confirmar CT y Bolsa se muestra la aceptación sintética. En CT64, versión resultante `3`, expediente `6`.
El primer `503` dejó CT guardado y Bolsa pendiente; se recuperó con la misma clave, sin regenerar evaluación.
Ocho POST `200` antes y después del reinicio principal, mismos recibo/fecha/evaluación/auditoría, sin duplicados.
Ante ambigüedad conserve clave/material; el `409` pendiente conocido, sin ambigüedad previa, permite corregir casillas, no cambiar clave.
No acredita envío, plazo legal ni firma; no registra propuesta ni tercer llamamiento automáticamente.
CT65 permite la propuesta separada descrita debajo. Continúan cinco pasos completos más tramos del sexto y séptimo.

### 7. Nombramiento: propuesta de desarrollo y límites

Para el sucesor aceptado, recupere las ocho operaciones anteriores y use el mismo formulario de propuesta,
novena operación del panel: clave original y [recorrido CT65](../../GUIA_RECORRIDO_ALBERTO.md#propuesta-desde-la-aceptación-del-sucesor-ct65).
Referencias/publicaciones derivadas, confirmación expresa y versión esperada `6` incluso al recuperar desde `7`.
Navegador: ocho antecedentes `200` y propuesta `201`; tras reiniciar app/PostgreSQL principal, nueve `200`,
mismos recibo/fecha/v7 e historia. El expediente sucesor pasa a nombramiento/en_curso; propuesta original intacta.
No cambie clave/material ante ambigüedad o conflicto. Sin otro `.eml`, firma, envío, plazo legal ni incorporación;
no aumenta la métrica RRHH ni acredita nuevas descargas de PDF.

En el caso original de aceptación confirmada de la guía, el mismo panel ofrece registrar la
propuesta, nunca desde renuncia. Referencias y cuatro publicaciones se derivan;
no se editan. Conserve versión esperada `6`, aunque el agregado ya sea `7`, y la
clave de propuesta `018f47a6-5d2b-4c10-8a11-123456789008` del caso de la guía.
Confirme expresamente, sin otro `.eml`; ante ambigüedad mantenga clave/material.
Dirección confirmó `201` y recuperación `200` tras reinicio principal, mismo
recibo y fecha `2026-09-06T01:28:30.697897Z`. Es una propuesta sin firma ni eficacia
de nombramiento. Objetivo 9: los seis borradores disponibles, incluida comunicación al centro.
El objetivo 10 ya permite validación manual sintética de resolución; el circuito
de firma oficial sigue pendiente.

Para descargar, vaya a **Cuadro de mando**, busque
`2026/CT-f5a5578760afec875187195d4108606a`, aplique el filtro y pulse **Abrir expediente**.
Desde el detalle real `v7`, `nombramiento/en_curso`, use en la cabecera
**Descargar informe · borrador de desarrollo** (`informe-definitivo-borrador.pdf`),
**Descargar resolución · borrador de desarrollo** (`resolucion-borrador.pdf`),
**Descargar diligencia · borrador de desarrollo** (`diligencia-borrador.pdf`),
**Descargar toma de posesión · borrador de desarrollo** (`toma-posesion-borrador.pdf`),
**Descargar notificación · borrador de desarrollo** (`notificacion-borrador.pdf`) o
**Descargar comunicación al centro · borrador de desarrollo** (`comunicacion-centro-borrador.pdf`).
Sin firmas ni eficacia administrativa; no certifican hechos, comparecencia, posesión real ni nombramiento.
La notificación es solo un borrador: no acredita envío, entrega ni apertura de plazo legal.
La comunicación al centro no envía un aviso ni ordena la incorporación.
En el caso validado `v8`, los mismos botones descargan los borradores de la
propuesta anterior `v7`: no hay que crear otra propuesta ni cambiar su versión.
No recupere la propuesta por POST ni repita el llamamiento para esta lectura.
Ante error se conserva el detalle. [Instrucciones y SHA256](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-9-descargar-el-primer-informe-borrador).
AD3-20/CT61 y AD3-21 instaladas en ambas bases: no reaplicar; el PDF no añade SQL propio.

El nombramiento completo exige la propuesta y los documentos y firmas que
correspondan. El conjunto previsto incluye **informe definitivo, resolución,
diligencia, toma de posesión, notificación y comunicación al centro**.

La descarga de los seis borradores no completa su circuito de firma.
Los modelos de desarrollo o un catálogo de demostración no son modelos
oficiales aprobados por RRHH. El informe sin firma del paso 5 no sustituye
al paquete de formalización. No marque aceptación, firma o notificación para
desbloquear artificialmente este paso.

#### Validación manual de la resolución, solo ejercicio sintético

Desde el detalle de un caso con propuesta, revise número, fecha y motivo de
la resolución. Lea y marque las dos comprobaciones del formulario; pulse
**Registrar validación manual** y confirme expresamente solo si procede
crear esa actuación sintética. Espere el recibo antes de darla por guardada.

En el caso ya demostrado `v8`, consulte el recibo existente. La recuperación
tras reiniciar aplicación y PostgreSQL conservó recibo, fecha y actuación;
no se necesita repetir esa prueba ni validar el caso original `v7` para
enseñarlo. Una respuesta incierta obliga a conservar los datos y la misma
clave, no a crear otra resolución. No acredita firma ni eficacia administrativa.

### 8. Incorporación, GINPIX y seguimiento: límite actual

Para presentar el caso ya conservado, abra su detalle `v8` y use **Recibo
original**. Compruebe y guarde la referencia del recibo y su fecha antes de
continuar. **Descargar ficha GINPIX** obtiene por `GET` la ficha manual admitida;
**Consultar seguimiento** recupera el vínculo original. Estas consultas nunca
deben sustituirse por otro `POST` de incorporación. Si un `GET` falla, conserve
las referencias y comunique el error sin crear otra alta.

La recuperación de incorporación, el asiento de Personal y la ficha GINPIX se
comprobaron después de reiniciar aplicación y PostgreSQL, con recibo, fecha,
seis campos y SHA256 conservados. La ficha es una salida para carga manual: no
demuestra transmisión, alta ni confirmación del sistema GINPIX. La primera
anotación y el cierre administrativo continúan pendientes del ensayo e
instalación de CT86/87 y AD3-30/31; estos botones de consulta no registran esos
efectos ni completan el octavo paso.

## Recibos, interrupciones y reanudación

Un recibo identifica una actuación, no todo el procedimiento. Conserve
expediente, operación, versión, recibo y fecha; cuando exista, conserve también
la clave de la petición mediante el procedimiento seguro de la guía. No
incluya certificados o secretos en una captura o incidencia.

| Situación | Qué hacer |
| --- | --- |
| Hay recibo confirmado | Continúe desde ese resultado. No repita el alta para «volver a entrar». |
| Se perdió la respuesta o figura resultado indeterminado | Conserve datos y clave. Pida comprobar el resultado; puede haber efectos persistidos aunque la pantalla no los haya recibido. |
| Reintento autorizado de la misma operación | Misma clave y mismos datos, con autorización vigente. No cambie el contenido bajo esa clave ni reutilice credenciales caducadas. |
| Reinicio de VEC | El operador conserva PostgreSQL y el material del entorno. La guía indica cómo consultar o recuperar el resultado; no se recrea la base. |
| Alta o análisis tras cerrar la página | La persistencia existe, pero la guía advierte que no tienen aún una vista de recarga de recibos cerrados. Pida comprobación al operador, no cree un duplicado. |
| Recuperación de selección y comunicación | La guía acredita mismos recibos y fechas tras reiniciar, usando las claves originales. No pulse **Preparar clave nueva** para recuperar ese caso. |
| Recuperación de la declaración RRHH | Comprobada tras el segundo reinicio de aplicación y PostgreSQL: mismos recibo, justificante y fecha, sin nuevo registro. Conserve la misma clave y todos los datos originales. |
| Recuperación de la renuncia manual sintética | Comprobada tras reiniciar aplicación y PostgreSQL principal: mismos recibo, resolución, auditoría, fecha e intención pendiente. Use sus cuatro claves originales, no las de aceptación. |

La recuperación de una selección interrumpida está acotada al estado
admitido por el servidor. No autoriza a borrar historia, reiniciar estados o
reintentar cualquier fallo. Si falla la escritura del fichero de aviso, no
lo considere completado: el operador comprobará el registro y el reintento
expreso con la misma clave, sin repetir la selección.

## Errores y avisos que requieren detener la acción

| Mensaje o situación | Actuación del técnico |
| --- | --- |
| Datos incompletos o inválidos | Revise los campos señalados antes de confirmar. Use los catálogos disponibles. |
| Acceso denegado | Compruebe perfil, certificado y ámbito con el operador. RRHH no sustituye a Intervención ni a un firmante. |
| Conflicto de versión o duplicado (`409`) | No fuerce otra versión ni otra clave. Conserve la referencia y pida consultar el estado actual; un conflicto no implica siempre el mismo motivo. |
| Operación ocupada o resultado indeterminado | No dé por hecho éxito ni ausencia de efectos. Evite envíos repetidos y siga la recuperación anterior. |
| Servicio o bandeja no disponible (`503`) | No significa «no hay expedientes». Comuníquelo a Sistemas sin volver a registrar el expediente; no convierta un fallo de consulta en una nueva alta. |
| Fiscalización desfavorable | Tramite la devolución a la unidad competente; no continúe al llamamiento. |
| Documento sin firma, aviso local o entrega pendiente | Conserve el alcance indicado. No los convierta en firma, aceptación ni inicio de plazo. |

Al comunicar una incidencia, indique entorno de desarrollo, pantalla, acción,
mensaje, momento, expediente y recibo si existe. No remita cuerpos completos
con datos personales ni material de autenticación.

## Fuentes y ayuda

- [Procedimiento normalizado de contratación temporal](../portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Petición de RRHH: transcripción y lectura funcional](../estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md).
- [Requisitos de acceso interno y separación de perfiles](../estudio_requisitos/acceso_interno_tecnicos_administracion.md).
- [Acceso y operación actuales por Sistemas](../manual_sistemas/README.md#entorno-privado-vigente).
- [Guía de casos sintéticos y recibos conservados](../../GUIA_RECORRIDO_ALBERTO.md).
- [Manual de usuario del módulo Bolsa](../manual_usuario/manual_portal_bolsas.md).

El registro de la declaración no completa la resolución de respuesta del sexto
paso. Siguen faltando
la comunicación con entrega acreditada y la respuesta de la persona candidata
con sus reglas de plazo. Este manual no redefine esas reglas ni declara
aceptación por RRHH.
