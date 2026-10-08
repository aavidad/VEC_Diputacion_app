# Manual de usuario · Portal VEC y Bolsas de trabajo

## Bolsa en la presentación a RRHH — 28 de septiembre de 2026

Esta actualización describe lo integrado en `main@7247682cb`, a partir del
[corte de presentación del 26 de septiembre](https://github.com/aavidad/VEC_Diputacion_app/blob/033fda6ed/ESTADO_PROYECTO.md#bolsa-y-contratación-temporal-cerradas-para-la-presentación--26-de-septiembre-de-2026).
«Mi bolsa» y el portal del candidato figuran en ese corte como desplegados y
habilitados en la instancia principal de presentación. Use solo identidades y
datos **sintéticos** facilitados por Sistemas. La presentación no autoriza
tramitación de personas reales: faltan decisiones de RRHH y la puerta de
producción sigue cerrada.

VEC reúne en una misma navegación la bolsa, sus candidaturas y la continuación
de Peticiones de personal temporal. RRHH puede consultar orden, situación, contactos y
actuaciones con su contexto; la persona candidata puede ver **sus propias**
participaciones y el resultado de las acciones que el servidor admita. Los
recibos y el historial facilitan comprobar una actuación y recuperar su
resultado tras una interrupción. Las reglas de ejemplo se cambian en su
catálogo gobernado; no hay que inventar una regla jurídica para enseñar el
recorrido.

El [apartado 5](#5-bolsas-de-trabajo-recorrido-de-presentación) explica el
recorrido de Bolsa y sus límites. Las referencias de Peticiones de personal temporal
anteriores a este corte se conservan abajo como historial de sus pruebas.

## Disponible para enseñar — 10 de septiembre de 2026

Cinco pasos completos y partes del sexto y séptimo, con datos inventados
guardados de verdad. También se ha registrado una incorporación sintética real
del octavo paso, pero su recuperación posterior al reinicio sigue pendiente;
por tanto, el ciclo completo no se presenta aún como disponible.
El cierre funcional de referencia corresponde a `00558603`; no identifica
por sí solo la aplicación servida actualmente. Se conservan 52 expedientes
sintéticos, sin crear otros para presentar el recorrido.

Sistemas debe preparar el acceso privado al servidor y el certificado de
pruebas. No hay enlace público; `localhost:8443` no apunta al servidor desde
su ordenador. Consulte [acceso actual](../manual_sistemas/README.md#entorno-privado-vigente).

### Consultar la validación y los documentos

1. Entre en **Peticiones de personal temporal**, localice el expediente sintético
   indicado por quien presenta y abra su detalle.
2. En un caso con propuesta pendiente de validación, revise el formulario de
   resolución y marque sus dos comprobaciones solo después de leerlas.
   Confirme únicamente si desea registrar una nueva actuación del ejercicio.
3. Espere al recibo. Si el caso ya está validado, consulte el recibo existente;
   no cree otra petición para enseñarlo.
4. Los seis botones de descarga siguen disponibles después de validar.
   Descargan los borradores de la propuesta anterior, no documentos firmados.

El caso comprobado conserva el mismo recibo después de reiniciar aplicación
y base. Una conexión interrumpida no demuestra que se haya perdido el registro:
mantenga los datos y la clave originales y consulte al operador antes de repetir.
El certificado de acceso no firma; la validación manual no es nombramiento
eficaz ni envía notificaciones. La incorporación registrada conserva
`firma_oficial=false` y `eficacia_administrativa=false`. No se debe repetir:
la consulta de recuperación devolvió `503` y Sistemas debe cerrar primero la
restauración de autorización. La integración con GINPIX tampoco está cerrada
como evidencia visible.

## Historial funcional conservado hasta el 6 de septiembre

Las cifras y ubicaciones de este historial describen aquellos cortes, no el
acceso actual. Para operar prevalece el apartado anterior.

Diputación de Granada · Ventanilla Electrónica del Empleado Público

**Edición: 6 de septiembre de 2026.** Cierre anterior publicado: versión
`b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`. Este manual explica qué puede recorrer una persona desde el
portal, qué resultados debe esperar y qué opciones siguen pendientes.
Describe un entorno de desarrollo con datos sintéticos, no un servicio
autorizado para tramitar expedientes de personas reales.

**Centros y organización:** en Peticiones de personal temporal, el enlace
«Centros y organización de referencia» permite buscar unidades y consultar
su procedencia. Los controles de edición solo aparecen cuando Sistemas
habilita el guardado persistente. Un cambio exige revisar unidad, adscripción
y motivo antes de confirmarlo; el resultado debe mostrar un recibo.
Si la conexión falla, no cierre ni recargue la pestaña: reintente la misma
operación cuando la pantalla lo permita. Los borradores sin confirmar solo
viven en esa pestaña. Añadir un cargo no asigna una persona ni le concede
permisos para solicitar o ratificar. Consulte la
[guía vigente de organización](../../GUIA_RECORRIDO_ALBERTO.md#centros-y-organización-de-referencia).

**Disponible: cinco pasos completos de Peticiones de personal temporal y partes del sexto
y séptimo: llamamiento sintético y propuesta de nombramiento de desarrollo.** No están
completados el llamamiento corporativo, el nombramiento ni la incorporación.
La base principal conserva 51 solicitudes, con bandeja y detalle consultables
en el recorrido local `8443`/base `55433`; no incrementa el contador de pasos.

**Incluido en esta entrega; recuperación demostrada:**
el registro de declaración RRHH obtuvo `HTTP 201`. Corregida la comparación
de fechas con ceros finales, dirección confirmó en navegador `200/200/200`
para recuperar selección, comunicación y respuesta tras el segundo reinicio
de aplicación y PostgreSQL: mismos recibo, justificante y fecha, sin nuevo
registro. Sin errores JavaScript, cookies, almacenamiento web ni desbordamiento
horizontal en ese recorrido. También se confirmó un conflicto real `409` en
navegador. La entrega 3 queda cerrada funcionalmente en desarrollo.

**Corte 4 publicado en `17ea874`:** aceptación manual sintética `201`;
tras reiniciar aplicación y PostgreSQL principal, navegador `200/200/200/200`,
mismo recibo y fecha, sin duplicados. Cierre técnico con servicios y permisos reales,
no aprobación de un plazo legal. Aquel corte mantenía **5/8 pasos completos más parte del sexto**.

**Corte 5 incluido en esta entrega:** renuncia manual sintética registrada desde
el mismo formulario: navegador real `200/201/201/201` y recuperación
`200/200/200/200` tras reiniciar aplicación y PostgreSQL principal, con los mismos
recibo, resolución, auditoría, fecha e intención pendiente, sin duplicados.
Sin errores JS, cookies, almacenamiento web ni desbordamiento. Objetivo 5 cerrado
funcionalmente en desarrollo; criterio provisional sin aval legal ni del operador.

**Objetivo 7 incluido en esta entrega:** quinta operación `201` real tras renuncia;
recuperación `200/200/200/200/200` tras reiniciar app/PostgreSQL principal, mismos
recibos y fecha. Cerrado funcionalmente solo en ese ejercicio sintético, sin
duplicados, errores JS, cookies, almacenamiento web ni desbordamiento.

**Objetivo 8 cerrado funcionalmente en desarrollo:** propuesta visible en Chrome
con `201`; tras reiniciar app/PostgreSQL principal, cuatro antecedentes `200` y
propuesta `200`, mismos identificadores, recibo, fecha y versión `7`, sin duplicados.
Cero errores JS, cookies, almacenamiento web y desbordamiento. No acredita firma,
nombramiento eficaz ni correo real. Esta revisión incorpora el cierre funcional;
el hash publicado se comprueba en Git.

**Objetivo 9: primer informe disponible como borrador de desarrollo.** Chrome `200`,
29267 bytes, PDF y pantalla inspeccionados por dirección. Rectificación: el primer
fallo navegador no capturó estado HTTP; el `502` era curl con límite 50, paginación
separada sin corrección acreditada. Navegador límite 100: sonda `200`, vista `404`
por fechas equivalentes comparadas como texto. AD3-21 corrige las dos consultas y
está instalada en ambas bases, no reaplicar; AD3-20/CT61 se conservan.
Tras el parche, cinco POST de bandeja/detalle/PDF `200`, mismo PDF y cero errores JS,
cookies y almacenamiento web. Tras reiniciar app/PostgreSQL principal: cinco POST `200`,
sin `404`, mismo PDF e historia CT/Bolsa conservada; no cierra la paginación con límite 50.
Dirección observó la pantalla estable de 390 px sin obstrucción del detalle ni del botón;
la captura previa era una transición de 180 ms, sin cambio de UI ni validación de usabilidad global.
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
Siguiente objetivo 10, circuito de firma/evidencia admitida. Sin SQL nuevo, firma, envío,
entrega, plazo legal ni orden de incorporación; evidencia central en la guía.

Para elegir la documentación adecuada:

- Este manual: acceso, navegación, resultados visibles y ayuda del portal.
- [Manual de Recursos Humanos](../manual_rrhh/README.md): procedimiento,
  responsabilidades y recorrido de tramitación por perfiles.
- [Guía de recorrido de Alberto](../../GUIA_RECORRIDO_ALBERTO.md): comandos
  y datos sintéticos conservados, con las pruebas de recuperación de cada
  corte. Sus arranques locales antiguos son históricos. El acceso actual
  corresponde a [Sistemas](../manual_sistemas/README.md#entorno-privado-vigente);
  no necesita ejecutar comandos para utilizar el entorno preparado.

## 1. Antes de empezar: real, demostración y pendiente

En este manual, **real en desarrollo** significa que el servidor comprueba el
permiso, guarda el resultado en la base de datos y devuelve un recibo. Los datos
siguen siendo sintéticos. No significa firma jurídica, envío corporativo ni
autorización para producción.

**DEMO o presentación** permite conocer pantallas con ejemplos. Sus cambios
son simulados; un recibo de presentación no acredita una actuación guardada
en el expediente. Puede perder esos cambios al recargar o cerrar la página.

**Pendiente o no disponible** significa que no hay un recorrido completo
habilitado en esta versión, aunque exista el menú, el formulario o una parte
del programa. Un botón visible no garantiza que el servicio esté conectado.

| Zona | Qué puede esperar en esta edición |
|---|---|
| Peticiones de personal temporal | Recorrido real de desarrollo descrito en el apartado 4, con recibos y persistencia. |
| Cuadro y detalle de expedientes de Peticiones de personal temporal | La base del servidor conserva 52 expedientes sintéticos; bandeja y detalle consultables mediante el acceso privado preparado por Sistemas. |
| Gestión interna de Bolsas | Cuadro, ficha de candidatos y acciones de la presentación integrada en `main`. El menú solo ofrece las capacidades habilitadas para la sesión. Consulte el apartado 5 antes de registrar una actuación sintética. |
| Mi bolsa, área personal | Consulta de participaciones propias, situación y último llamamiento; algunas acciones muestran recibo si el servidor las confirma. Acceso con identidad de candidato habilitada para la presentación. |
| Consulta pública de bolsas y convocatorias | Listados y detalles minimizados sin identidad. Un documento de ejemplo no son unas bases aprobadas ni permite tramitar una candidatura personal. |
| Otros módulos del portal | Solo están disponibles si el servidor los habilita para su perfil. No se consideran terminados por aparecer en la portada. |

No utilice datos de personas reales, documentos de identidad, teléfonos,
correos ni expedientes reales en ninguno de estos recorridos de desarrollo.

## 2. Acceso y finalización de la sesión

### Entrar al portal interno

1. Pida a Sistemas la dirección del entorno, su certificado de desarrollo y
   el perfil de navegador preparado. No comparta certificados ni contraseñas.
2. Abra la dirección indicada y entre en `/portal-empleado/`, sin seleccionar
   el modo de presentación. La instancia está en el servidor privado;
   [Sistemas prepara el acceso](../manual_sistemas/README.md#entorno-privado-vigente)
   antes de abrirla desde otro equipo.
3. Utilice el certificado correspondiente a su función. Recursos Humanos e
   Intervención usan certificados y perfiles de navegador separados.
4. Espere a que el portal compruebe los módulos disponibles. Para Bolsa,
   abra **Gestión de Bolsas → Bolsas y candidatos**. Para Peticiones de personal temporal,
   entre desde **Inicio del portal** en ese módulo y consulte el expediente
   sintético indicado por quien presenta.

La conexión exige un certificado de cliente válido. Si el navegador indica
que falta el certificado o no reconoce el servidor, pida ayuda a Sistemas:
no omita la advertencia de seguridad ni cambie la dirección para sortearla.

El perfil y los permisos los determina el servidor. Escribir otro nombre de
perfil en la dirección, cambiar una referencia o seleccionar un perfil de
presentación no concede permisos reales.

La persona candidata utiliza su propio acceso a **Mi área personal**. No use
el certificado ni la sesión de RRHH para enseñar «Mi bolsa»: se mostrarían
competencias distintas y no se verificaría el aislamiento de la ficha propia.

### Consultar la zona pública

Si Sistemas ha habilitado la consulta pública, abra `/bolsa/`. Esa zona
muestra información de convocatorias, no los expedientes internos de RRHH
ni la ficha privada de cada aspirante. Su disponibilidad es independiente
del recorrido interno de Peticiones de personal temporal.

### Conocer la presentación

Use el selector de `/presentacion/` cuando quiera explorar las pantallas
DEMO. La franja **Presentación para RRHH** indica que las personas,
expedientes y actuaciones internas son sintéticos y sin efectos reales.

Algunas referencias de convocatorias o del Boletín Oficial de la Provincia
pueden proceder de publicaciones públicas. Eso no convierte en reales los
plazos, documentos o actuaciones rotulados DEMO. No mezcle sus referencias
con un formulario del recorrido real.

### Al terminar

Conserve por el canal autorizado las referencias necesarias para continuar
y cierre el perfil temporal de navegador. La retirada del material de
desarrollo se realiza según las instrucciones de Sistemas y la guía.
Cerrar una pestaña no revoca por sí solo un certificado.

## 3. Cómo orientarse en la pantalla

- **Inicio del portal** muestra los módulos y su disponibilidad para el
  perfil activo. Si aparece **No disponible**, no hay una entrada operativa
  concedida para ese acceso.
- El **menú lateral** cambia según el módulo. En pantallas pequeñas se abre
  con el botón de navegación. En Bolsa, los grupos desplegables reúnen
  opciones relacionadas; el apartado activo queda señalado.
- Las **migas de navegación** y el título de la cabecera indican dónde está.
  No confunda el llamamiento de Peticiones de personal temporal con el asistente de
  presentación de Bolsa: son accesos distintos.
- **A+** aumenta o restablece el texto y **Contraste** activa o desactiva el
  alto contraste. Estos ajustes afectan a la página abierta; no se guardan
  como preferencias para otra sesión.
- **Avisos** muestra los avisos accesibles. Un guion o un mensaje de fuente
  no disponible no equivale a «cero asuntos pendientes».
- El botón **«?»** del portal RRHH abre la ayuda de la vista actual. **Cerrar**
  devuelve a la pantalla sin registrar una actuación; no hay un bloque de
  texto de ayuda permanente en esa superficie.

Si el panel de Bolsa no carga, no suponga que ha perdido todos los permisos
del portal: la disponibilidad de Peticiones de personal temporal se comprueba por
separado.

## 4. Recorrido real de Peticiones de personal temporal

El orden disponible se resume a continuación. Los campos concretos, las
responsabilidades y las alternativas de tramitación se desarrollan en el
[manual de RRHH](../manual_rrhh/README.md). Para reproducir el ejemplo
conservado, use los datos exactos de la
[guía de recorrido](../../GUIA_RECORRIDO_ALBERTO.md).

| Paso del recorrido | Acción visible disponible | Resultado y límite |
|---|---|---|
| 1. Solicitud | Revisar y registrar una nueva petición con datos de catálogo. | Expediente y primer recibo guardados. |
| 2. Análisis | Registrar el análisis por Recursos Humanos. | Mismo expediente, nueva versión y recibo. Hay cinco modalidades; el ejemplo recorrido utiliza Sustitución. |
| 3. Bolsa: vía de cobertura | Revisar la propuesta y confirmar **Bolsa vigente**. | Decisión de cobertura guardada. No crea por sí sola una bolsa ni publica una convocatoria. |
| 4. Asignación | Confirmar la unidad y la persona responsable referenciada. | Asignación y recibo guardados. |
| 5. Informe jurídico y Fiscalización | Preparar el informe de desarrollo; Intervención registra el resultado. | Documento sin firma ni validez jurídica y resultado de fiscalización guardados. El desfavorable registra la devolución a la unidad. |
| 6. Llamamiento, parcialmente disponible | Recorridos sintéticos, aviso CT62, declaración CT63 y aceptación manual del sucesor CT64 recuperables tras reinicio principal. | Faltan vencimiento, envío corporativo y plazo; no acredita entrega ni plazo legal. |
| 7. Nombramiento, parcialmente disponible | Registrar y recuperar la propuesta desde aceptación sintética; expediente `6→7`; descargar seis borradores y consultar la validación manual sintética con su recibo. | Recibo recuperado tras reinicio principal. No hay nombramiento eficaz, posesión real, firma, envío ni entrega; validación manual sintética disponible, no firma oficial. |
| 8. Incorporación y seguimiento | Pendiente como recorrido completo. | No se acredita incorporación, integración con GINPIX ni cierre del seguimiento. |

El número de recibos o la versión del expediente no es el número de pasos
completados: el paso 5 contiene más de una actuación.

### Iniciar y continuar una petición

1. En **Peticiones de personal temporal → Nueva petición**, complete el formulario con
   las entradas sintéticas de los catálogos. Revise fechas y campos
   obligatorios.
2. Pulse **Revisar solicitud** y después **Confirmar y registrar** una sola
   vez. Espere a ver **Solicitud registrada** y su recibo.
3. Continúe por los formularios que aparecen tras cada resultado: análisis,
   decisión de cobertura, asignación e informe. Compruebe que mantienen la
   misma referencia de expediente.
4. Lea los avisos antes de confirmar. El informe muestra expresamente
   **DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA**.
5. Para fiscalizar, use el perfil separado de Intervención. En
   **Peticiones de personal temporal**, introduzca la referencia del expediente y su
   versión remitida `5`; pulse **Abrir fiscalización**.

Fiscalización admite **Favorable**, **Favorable con observaciones** y
**Desfavorable**. Los dos últimos requieren observaciones. Un resultado
desfavorable devuelve el expediente a la unidad con estado de incidencia;
no habilita el llamamiento favorable ni constituye un error de guardado.

No registre otra solicitud solo para volver a abrir la anterior. La bandeja
permite localizar el expediente y abrir su detalle; use las referencias
conservadas y las entradas indicadas en la guía.

### Iniciar o recuperar el llamamiento y su aviso local

1. Con el perfil de RRHH, abra **Peticiones de personal temporal → Nueva petición →
   Llamamiento y comunicación**.
2. Para recuperar el ejemplo ya existente, tome de la guía la referencia
   del expediente fiscalizado, la versión de entrada `6` y la clave de
   selección original. No cree otra solicitud ni repita la fiscalización.
3. Pulse **Revisar e iniciar llamamiento** y confirme. El servidor comprueba
   los permisos y el expediente; no elija ni invente una persona candidata.
4. Espere al **Recibo verificado**. Los datos de **Registrar comunicación**
   se rellenan desde ese resultado. No sustituya la organización, la
   referencia del llamamiento ni el recibo antecedente.
5. Para recuperar también el aviso existente, introduzca su clave de
   comunicación original y pulse **Revisar y registrar comunicación**.
6. Compruebe el mensaje **Registrada localmente · Sin entrega acreditada** o,
   si ya existía, **Registro local recuperado · Sin entrega acreditada**.

El aviso se conserva como un archivo en el servidor de desarrollo. **No se
envía correo corporativo ni se acredita que lo haya recibido una persona.**
Una referencia de intención de envío o un plazo mostrado no demuestra que
haya comenzado un plazo legal de respuesta.

Para una operación genuinamente nueva, siga la guía y conserve su nueva
clave antes de enviar. Para recuperar una anterior, no pulse **Preparar
clave nueva**: mantenga la clave y todos los datos originales.

### Registrar una respuesta declarada por RRHH

1. Recupere la comunicación confirmada en versión `2` mediante el recorrido
   anterior. En el **mismo formulario**, la operación **3. Registrar respuesta
   recibida por correo** toma sus antecedentes del recibo; no los invente.
   El servidor exige el permiso propio
   `contratacion_temporal.llamamiento.respuesta.registrar`.
2. Elija **Aceptación declarada en el correo** o **Renuncia declarada en el
   correo**, indique una referencia opaca sin datos personales y la **fecha de
   recepción en UTC**, no la hora local. Seleccione un `.eml` no vacío de hasta
   **2 MiB**. WebCrypto calcula SHA-256 en su equipo: el contenido no se envía
   ni se guarda en VEC y la huella no se edita manualmente.
3. Espere a **Huella calculada**, revise los datos y confirme expresamente el
   registro. El selector puede quedar vacío al actualizar la pantalla aunque
   la huella siga calculada; el contenido del archivo no se conserva.
4. Conserve el recibo, el justificante, la huella, las fechas y la clave. Ante
   un resultado ambiguo o un acceso denegado, no prepare otra
   clave ni cambie los datos para forzar el registro: avise al operador.

Para practicar, use el [correo sintético existente](../manual_rrhh/ejemplos/respuesta_sintetica.eml)
y los datos del caso de aceptación de la [guía](../../GUIA_RECORRIDO_ALBERTO.md). Ese archivo
**nunca se envió y no corresponde a una persona real**.
El caso de renuncia tiene otro correo y otras claves: use su apartado de la guía,
no cambie la respuesta ni los bytes del ejemplo anterior para recuperar la renuncia.

El original sigue en el sistema de correo; VEC registra la declaración, su
referencia y su huella, **sin verificar origen, firma ni custodia**. No acredita
envío o entrega ni resuelve aceptación o renuncia. La comunicación conserva
la versión `2`, el expediente observado sigue en `6` y no cambia la candidatura
ni su estado en Bolsa. No aumenta el contador de pasos.

### Registrar aceptación o renuncia manual: solo ejercicio sintético

1. Recupere la declaración **Aceptación declarada en el correo** o **Renuncia
   declarada en el correo** con los datos
   originales de la [guía](../../GUIA_RECORRIDO_ALBERTO.md). Aparece **4. Solicitar
   resolución de respuesta** en el mismo formulario. La respuesta y el justificante
   proceden del recibo de declaración confirmado, enlazado a comunicación `v2`;
   no puede elegir otra respuesta desde la resolución.
2. Use una clave propia para resolución, distinta de las operaciones anteriores.
   Para recuperar una resolución registrada, conserve su clave y datos originales.
3. Solo tras comprobar el caso sintético, marque las dos casillas, inicialmente
   vacías: **He comprobado la respuesta y su justificante** y **Para este ejercicio
   sintético, he comprobado que la respuesta llegó dentro del plazo del ejercicio**.
   El criterio de desarrollo es de solo lectura. No se aporta otro `.eml`.
4. Pulse **Revisar y solicitar resolución** y confirme expresamente. Solo tras
   confirmar CT y Bolsa aparece **Aceptación registrada · ejercicio sintético**
   o **Renuncia registrada · ejercicio sintético**, según la declaración original.
   La renuncia muestra referencia, fecha UTC y **Siguiente candidato pendiente**:
   no se ha seleccionado ni avisado a otra persona.

**Validación manual de desarrollo: no acredita entrega de correo ni plazo legal real**.
Sin ambas casillas no se envía. La petición antigua sin revisión manual conserva
el `409` de validación pendiente, sin efectos: puede corregir casillas con la misma
clave. Ante resultado ambiguo, clave y material quedan congelados, sin reintento automático.
Dirección confirmó ambas resoluciones `201` y recuperación `200` tras reinicio,
con sus respectivos recibos y fechas originales; la intención del recibo de renuncia conserva su `pendiente` histórico.
La propuesta de desarrollo se describe abajo; no hay política
legal aprobada, vencimiento ni correo corporativo completos.
El criterio manual es provisional y exclusivo de desarrollo sintético, no aval del operador.

### Quinta operación: continuar tras la renuncia sintética

Tras recuperar la renuncia, el mismo panel ofrece continuar; expediente, resolución
e intención vienen de sus recibos, no se editan. Use la quinta clave original de la
[guía](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-7-recuperar-la-continuación-tras-renuncia)
y confirme expresamente la apertura de un único nuevo llamamiento, sin otro `.eml`.
**Siguiente llamamiento abierto · ejercicio sintético** confirma la continuidad
posterior: no reescribe el recibo de renuncia, ni acredita aviso enviado, entrega
o aceptación. No sustituya el llamamiento de la primera comunicación.
Una vez recibido, otro envío queda desactivado; ante ambigüedad conserve clave y
material, reintento solo explícito. Un `409` no autoriza cambiar de clave.

### Sexta operación: aviso local al sucesor

Recupere las cinco operaciones previas y use **6. Registrar aviso local al sucesor**.
Solo introduzca su sexta clave original de la [guía](../../GUIA_RECORRIDO_ALBERTO.md#aviso-local-al-sucesor-ct62);
el panel deriva las referencias y versión del recibo de continuación. Confirme expresamente.
`201` muestra **Aviso local al sucesor registrado · No enviado**; tras reinicio principal,
`200` recupera el mismo recibo, fecha, versión `2` e intención local, sin duplicados.
No envía correo ni abre plazo; habilita la declaración manual del paso 7, no una resolución. Los recibos previos
se conservan; ante ambigüedad mantenga clave/material, sin reintento automático ni eludir `409`.

### Séptima operación: declaración de respuesta del sucesor

Tras el aviso local validado `v2`, **7. Registrar respuesta recibida del sucesor** deriva sus referencias.
Declare aceptación o renuncia, referencia opaca de correo y fecha UTC; seleccione el `.eml` hasta 2 MiB,
compruebe la huella calculada localmente y confirme con su clave propia. El contenido no se sube ni guarda.
[Datos y archivo exactos para recuperar](../../GUIA_RECORRIDO_ALBERTO.md#declaración-de-respuesta-del-sucesor-ct63).
`201` real y siete operaciones `200` tras reinicio principal, mismos justificante/recibo/auditoría/fecha, sin duplicados.
**Declaración de respuesta del sucesor registrada · Sin resolución**
no acredita origen, firma, entrega ni plazo; tampoco cambia la resolución original. Conserve clave/material
ante ambigüedad. La resolución del sucesor requiere la octava operación separada, no es automática.

### Octava operación: resolución manual sintética del sucesor

Recupere las siete operaciones anteriores y use la octava con la clave original de la
[guía](../../GUIA_RECORRIDO_ALBERTO.md#resolución-manual-del-sucesor-ct64).
El panel deriva la respuesta y el justificante; no los edite. Marque las dos revisiones explícitas,
inicialmente vacías, de respuesta/justificante y plazo del ejercicio sintético, y confirme sin otro `.eml`.
La aceptación solo se confirma tras CT y Bolsa: en CT64, versión resultante `3`, expediente conservado en `6`.
El primer `503` se recuperó con la misma clave, sin regenerar evaluación: ocho POST `200` antes y después
del reinicio principal, mismos recibo/fecha/evaluación/auditoría, sin duplicados.
Ante ambigüedad conserve clave/material; el `409` pendiente conocido, sin ambigüedad previa, permite corregir casillas, no cambiar clave.
No acredita envío, plazo legal ni firma; no registra propuesta ni tercer llamamiento automáticamente.
CT65 permite la propuesta separada descrita debajo. La métrica sigue en **5/8 más partes del sexto y séptimo**.

### Propuesta de nombramiento tras aceptación

Para el sucesor aceptado, recupere las ocho operaciones previas y use **Propuesta de nombramiento · desarrollo**,
el mismo formulario, novena operación del panel. [Clave y datos CT65](../../GUIA_RECORRIDO_ALBERTO.md#propuesta-desde-la-aceptación-del-sucesor-ct65).
Revise los antecedentes derivados y las publicaciones; confirme con su clave original y versión esperada `6`,
también al recuperar desde `7`. Ocho antecedentes `200` y propuesta `201`; tras reinicio principal,
nueve `200`, mismos recibo/fecha/v7 e historia, sin duplicados. Propuesta original conservada.
Mantenga clave/material ante ambigüedad; no eluda conflictos con otra clave. Sin firma, envío, plazo legal ni incorporación;
no es un noveno paso RRHH ni acredita nuevas descargas de los seis PDF ya cerrados.

Para el caso original de la guía, el mismo panel ofrece **Propuesta de nombramiento · desarrollo** solo tras
aceptación confirmada, nunca renuncia. Referencias y publicaciones no son editables;
la solicitud conserva versión esperada `6`, también para recuperar el expediente ya en `7`.
Use la clave original de la [guía](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-8-recuperar-la-propuesta-de-nombramiento)
y confirme expresamente, sin otro `.eml`. Si las publicaciones no cargan, no envíe.
El recibo **Propuesta registrada · ejercicio sintético** conserva referencia y
fecha; no es un nombramiento firmado. Ante ambigüedad mantenga clave/material:
sin reintento automático ni otra clave para eludir un `409`.

### Descargar los seis borradores de desarrollo

En el caso ya validado `v8` también están disponibles: los botones recuperan
los borradores de su propuesta `v7`, no documentos nuevos ni firmados.
El ejemplo original siguiente permanece conservado sin esa validación.

En **Peticiones de personal temporal → Cuadro de mando**, busque
`2026/CT-f5a5578760afec875187195d4108606a`, aplique el filtro y pulse **Abrir expediente**.
En el detalle real `v7`, `nombramiento/en_curso`, elija un botón de cabecera:
**Descargar informe · borrador de desarrollo** (`informe-definitivo-borrador.pdf`),
**Descargar resolución · borrador de desarrollo** (`resolucion-borrador.pdf`),
**Descargar diligencia · borrador de desarrollo** (`diligencia-borrador.pdf`),
**Descargar toma de posesión · borrador de desarrollo** (`toma-posesion-borrador.pdf`),
**Descargar notificación · borrador de desarrollo** (`notificacion-borrador.pdf`) o
**Descargar comunicación al centro · borrador de desarrollo** (`comunicacion-centro-borrador.pdf`).
No recupere la propuesta por POST ni repita las operaciones de llamamiento para descargar.
Ante error se conserva el detalle. [Recorrido y huella comprobada](../../GUIA_RECORRIDO_ALBERTO.md#objetivo-9-descargar-el-primer-informe-borrador).
Son borradores sin firma: no certifican hechos, comparecencia, posesión real ni nombramiento eficaz;
la notificación tampoco acredita envío, entrega ni apertura de plazo legal;
la comunicación al centro no envía un aviso ni ordena la incorporación;
la métrica sigue en **5/8 más partes del sexto y séptimo**, sin otro paso RRHH completo.

### Qué ocurre después de un reinicio

En el cierre previamente publicado, la solicitud `v1` llevó al análisis `201`/`v2` y,
tras reiniciar, se recuperó un único recibo, mediante lectura independiente
de PostgreSQL y navegador.
Con la misma instalación, datos y material de seguridad, una recuperación
autorizada del llamamiento o del aviso devuelve el mismo recibo y la fecha
original, sin crear otro efecto.
La declaración de respuesta también se recuperó tras corregir la comparación
temporal y repetir el reinicio de aplicación y PostgreSQL. El navegador devolvió
`200`, con los mismos recibo, justificante y fecha original, sin nuevo registro.
Los identificadores exactos constan en el [manual de RRHH](../manual_rrhh/README.md).
La aceptación manual sintética también se recuperó tras reiniciar aplicación y
PostgreSQL principal: `200/200/200/200`, mismo recibo CT y fecha, sin duplicados.
La renuncia manual sintética también: `200/200/200/200`, mismos recibo, resolución,
auditoría, fecha e intención pendiente con su carga real. La aceptación anterior
y su declaración siguen intactas. La [guía](../../GUIA_RECORRIDO_ALBERTO.md)
separa las claves de ambos casos. La quinta operación también se recuperó tras
reiniciar app/PostgreSQL principal: cinco `200`, mismos 14 campos salvo
`estado_local: replay_confirmado`, sin duplicados; objetivo 7 cerrado solo tras renuncia sintética.
La propuesta del objetivo 8 también se recuperó tras reiniciar app/PostgreSQL
principal: `200`, mismos propuesta/recibo/fecha y versión `7`, sin duplicados.

El formulario sin confirmar no se guarda automáticamente en el navegador.
Después de cerrar o recargar puede tener que introducir de nuevo sus datos.
La guía contiene el ejemplo y la comprobación del reinicio; no reinicie ni
recree el servidor por su cuenta.

## 5. Bolsas de trabajo: recorrido de presentación

### RRHH: consultar una bolsa y actuar sobre una candidatura

1. Entre en **Gestión de Bolsas → Bolsas y candidatos** y abra el **Cuadro de
   bolsas**. La tabla reúne categoría, vigencia, totales por situación y
   llamamientos en curso. Una fila sin datos no significa que se hayan borrado
   candidaturas: compruebe el estado de carga y los filtros.
2. Elija una bolsa para abrir **Candidatos**. Consulte el orden calculado, el
   estado y, cuando exista, el último llamamiento. El filtro por situación y la
   búsqueda sirven para localizar la candidatura sin alterar su posición. El
   documento se muestra enmascarado cuando corresponde; no copie datos
   personales a una nota libre.
3. Abra la ficha de la candidatura. La pestaña **Histórico** y los bloques de
   contactos y situación distinguen qué se registró, cuándo y por quién. Si la
   pantalla ofrece **Pausar**, **Reactivar** o **Excluir**, revise la causa, el
   justificante y la persona validadora exigidos antes de confirmar. Una
   solicitud o un cambio solo existe cuando el servidor devuelve su recibo.
4. Para una necesidad de cobertura, use **Nuevo llamamiento** desde la bolsa o
   la vista de llamamientos que ofrezca su sesión. Revise puesto, destino,
   jornada, propuesta de orden, versiones y plazo de la regla mostrada. El
   servidor calcula la prelación: cambiar una fila de la tabla no selecciona a
   otra persona. Distinga propuesta, borrador, contacto, respuesta y
   resolución; cada una requiere su propia confirmación.
5. Si se muestran **Ofertas publicadas**, revise centro, fechas y plazo de
   disposición antes de publicar una oferta sintética. Al vencer el plazo
   configurado, vuelva a consultar la propuesta vigente antes de confirmar una
   adjudicación o el paso a llamamiento directo. El estado **Pendiente de
   resolver** significa que RRHH aún debe decidir.

La pantalla agrupa información y evita reconstruir el orden y los contactos en
hojas separadas. Una tabla permite ver varios casos a la vez y la ficha
conserva el detalle de cada uno. Los contadores, los estados escritos y el
historial facilitan localizar la siguiente acción, pero no reemplazan la
lectura de las bases ni la competencia de RRHH. Las entradas del menú dependen
del permiso y de la capacidad realmente conectada; una entrada visible o una
pantalla de presentación no acreditan por sí solas una actuación guardada.

**Otras opciones del menú.** Elaboración, convocatorias, solicitudes, méritos,
alegaciones, importación, reglas, baremación, estadísticas, documentos,
comunicaciones, auditoría y configuración reúnen consultas y pantallas con
alcances distintos. Siga el estado y el recibo de cada operación concreta. Ver
una pantalla no significa que se hayan publicado bases, importado un lote
corporativo, firmado un documento, enviado una comunicación o cambiado un
permiso. La vista de auditoría tampoco sustituye el registro protegido del
servidor.

### Persona candidata: consultar «Mi bolsa»

Con la identidad sintética de candidato preparada por Sistemas, entre en **Mi
área personal → Mi bolsa**. La consulta muestra únicamente las participaciones
vinculadas a esa identidad. Cada ficha puede mostrar categoría, versión y
vigencia de la bolsa, orden inicial dentro de su instantánea y última situación
registrada. **Orden inicial** no promete la posición futura: la prelación
depende de la versión y de las reglas aplicables. El catálogo de campos puede
ocultar un dato; no interprete su ausencia como cero o como una decisión
administrativa.

| Lo que aparece | Cómo leerlo |
| --- | --- |
| Disponible, No disponible o Disponible desde fecha | Situación de esa participación en Bolsa; la fecha indicada no equivale a un nuevo llamamiento. |
| Trabajando o Pendiente de incorporación | Estado comunicado a Bolsa; «pendiente» todavía no acredita relación de servicio. |
| Renuncia o Excluido | Situación registrada para esa participación; consulte su fecha y el cauce de revisión aplicable. |
| Último llamamiento por correo | Fecha, canal y resultado registrado. «Enviado» no prueba por sí solo entrega, lectura o aceptación. |
| Llamamiento abierto | La pantalla indica el vencimiento y el modo: **RRHH confirmará la respuesta** o **Respuesta firme**. Elija la respuesta admitida y conserve el recibo; una renuncia justificada exige la causa y el justificante previstos. |
| Solicitud de pausa o reactivación | Puede quedar **pendiente de RRHH**. El recibo confirma el registro de la solicitud, no su aprobación. |
| Oferta publicada | **Me ofrezco** manifiesta disposición para esa oferta. Su recibo no adjudica el puesto; consulte después si está abierta, pendiente de RRHH, resuelta o adjudicada a usted. |

Cuando se ofrece confirmar un contacto propio, revise primero la información
mostrada y confirme solo si es correcta. Una incidencia de conexión después de
pulsar una acción exige consultar el resultado o reintentar **la misma**
operación según indique la pantalla; cambiar datos o abrir otra solicitud puede
producir un conflicto. Los formularios sin confirmar no se conservan al cerrar
la pestaña.

### Consulta pública y reglas del ejercicio

En la consulta pública de bolsas se muestran solo campos minimizados, como
orden y documento enmascarado; no se abre la ficha privada del candidato. La
consulta de convocatorias permite filtrar, abrir detalles y examinar documentos
ofrecidos por la propia ficha. Un estado vacío puede deberse a filtros; un
error de servicio no demuestra que no existan convocatorias.

Las reglas todavía pendientes de decisión de RRHH están en un **catálogo de
ejemplo**, identificado como tal. Entre sus ejemplos están el orden por
puntuación y acta, la lista cerrada o rotatoria, la reposición tras un
contrato, las causas de situación y los plazos de disposición. Son parámetros
revisables y versionados; el rótulo **Regla de ejemplo** no les da validez
administrativa. El personal usuario no debe cambiar una fecha o una causa para
convertir una simulación en una regla aprobada.

El recorrido de presentación tampoco acredita firma legal, envío y entrega
corporativos, plazo legal, nombramiento eficaz ni incorporación. La
autenticación con certificado identifica a la persona ante la aplicación; no
firma documentos. Las capturas antiguas de este manual muestran cortes
anteriores y no prueban las funciones descritas en esta actualización.

## 6. Recibos y mensajes: cómo actuar

Un recibo identifica el resultado confirmado de una operación: referencia,
fecha y, según la actuación, expediente, versión y otras referencias
relacionadas. Conserve esos datos por el canal autorizado si necesita
continuar o pedir ayuda. No invente referencias ni confunda la versión del
llamamiento con la del expediente.

Un recibo de selección acredita selección; un recibo de aviso local acredita
registro local. Ninguno acredita por sí solo firma jurídica, correo,
entrega, aceptación o nombramiento.

| Mensaje o situación | Qué hacer |
|---|---|
| **Registrando. Conserve los datos y espere la respuesta.** | Espere. No pulse varias veces ni cambie la clave de operación. |
| **Recuperado sin repetir el efecto** | Es el resultado anterior recuperado. Compare su referencia y fecha; no espere un recibo nuevo. |
| **Registro local recuperado · Sin entrega acreditada** | El aviso local ya existía. No se ha demostrado un envío ni una entrega. |
| Campos inválidos o **Petición rechazada** | Revise campos, referencias y fechas. No cambie datos para aparentar una autorización. |
| Solicitud duplicada o conflicto | No cree otra clave ni altere el detalle para forzar el registro de la misma petición. Conserve las referencias y solicite revisión. |
| **Esta operación requiere revisión del servidor. No la repita ni cambie la clave para forzarla.** | Deténgase y avise a soporte. No inicie una operación sustitutiva. |
| Resultado no confirmado o conexión interrumpida | No suponga que nada se guardó. Conserve los datos originales; use la recuperación indicada por la pantalla y la guía, o consulte a soporte. |
| Acceso denegado | Compruebe el perfil y certificado con Sistemas. RRHH no puede fiscalizar usando su certificado. |
| Servicio, módulo o fuente no disponibles | No significa lista vacía ni éxito. Puede volver a comprobar una consulta de lectura; no repita a ciegas una operación de escritura. |
| **No hay avisos accesibles** | No hay avisos que esta vista pueda mostrar; no equivale a demostrar que toda la tramitación está al día. |

Los cambios locales de un formulario y las confirmaciones de presentación
no son recibos del servidor. Si no hay resultado confirmado, no comunique
que la actuación ha finalizado.

## 7. Seguridad y privacidad durante el uso

- Use únicamente datos sintéticos autorizados en desarrollo y el certificado
  asignado a su perfil. No intercambie el de RRHH con el de Intervención.
- No comparta contraseñas, certificados, claves privadas ni configuraciones
  de conexión. No los adjunte a incidencias ni los publique en GitHub.
- El portal no utiliza cookies ni almacenamiento web para conservar
  sesiones, expedientes o preferencias. Los datos confirmados permanecen en
  el servidor, no en una copia local del navegador.
- El servidor comprueba el permiso y el ámbito de cada consulta o cambio;
  acceder con certificado no concede por sí solo la función de RRHH. Los
  cambios confirmados conservan recibo e historia, y la auditoría registra
  quién actuó y cuándo según su autoridad. Use la ficha y los recibos para
  revisar el resultado; no trate una captura de pantalla como auditoría.
- La falta de nombres o contactos en un recibo de selección es intencionada.
  No intente reconstruir identidades desde referencias ni consultar
  expedientes ajenos.
- No pegue referencias privadas en buscadores, servicios externos o
  incidencias públicas. Incluso las referencias sin nombre deben tratarse
  por el canal de soporte autorizado.
- Si una acción está bloqueada, no modifique direcciones, campos técnicos o
  perfiles de presentación para sortearlo.

## 8. Ayuda y accesibilidad

En el portal de RRHH, pulse el botón **«?»** para abrir la explicación
contextual en un diálogo. El botón tiene un nombre accesible para lectores de
pantalla; el texto de ayuda se muestra al abrirlo, sin ocupar espacio fijo en
la vista de trabajo. La ayuda de Bolsa incluye preguntas frecuentes y una
transcripción que puede leerse sin reproducir audio.

Ese contenido explica el recorrido visible de Bolsa; no confirma por sí solo
que cada operación esté habilitada para la sesión actual.
Para el recorrido disponible de Peticiones de personal temporal, consulte el apartado
4, el [manual de RRHH](../manual_rrhh/README.md) y la
[guía de recorrido](../../GUIA_RECORRIDO_ALBERTO.md).

Para manejar la pantalla:

- Use **Tabulador** y **Mayús + Tabulador** para recorrer los controles, y
  **Entrar** o **Espacio** para activar un botón.
- El enlace **Saltar al contenido principal** evita recorrer todo el menú.
- En pantalla pequeña, abra y cierre el menú de navegación; **Escape**
  permite cerrarlo.
- Use **A+**, **Contraste** y el zoom del navegador según sus necesidades.
  Las tablas anchas pueden desplazarse dentro de su marco.
- Lea los mensajes junto al formulario y cierre los diálogos con
  **Cerrar**. Si un control no es accesible, notifíquelo; no se declara una
  certificación de accesibilidad por disponer de estas ayudas.

### Pedir ayuda sin perder el trabajo

Indique a soporte el entorno, la fecha y hora, su perfil funcional, la
pantalla y el texto exacto del mensaje. Añada por canal privado la referencia
de expediente, recibo y clave de operación si resultan necesarias para
localizar el intento. No incluya certificados, contraseñas ni datos reales.

Si aporta una captura, oculte cualquier dato sensible. Distinga entre
«pulsé confirmar», «recibí confirmación» y «recuperé un recibo existente»:
son situaciones diferentes.

Este manual en Markdown recoge el alcance de esta edición. Las capturas o
exportaciones anteriores no acreditan funcionalidades nuevas; utilice la
documentación de la versión que Sistemas le haya indicado.
