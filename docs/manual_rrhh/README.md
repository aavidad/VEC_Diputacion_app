# Manual funcional del técnico de Recursos Humanos

Contratación temporal · VEC Diputación · Corte: 5 de septiembre de 2026.

**Cierre ya publicado:** `b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`. La bandeja
real muestra 50 expedientes y su detalle en el recorrido local (`8443` / base
`55433`). El caso verificado encadena solicitud `v1` a análisis `201`/`v2` y
recupera el recibo único tras reinicio, mediante lectura independiente de
PostgreSQL y navegador.
Se mantienen cinco pasos y parte del sexto; este cierre no incrementa el
contador. La [guía canónica](../../GUIA_RECORRIDO_ALBERTO.md) contiene el
recorrido y comandos exactos.

## Qué puede hacer hoy

**Cinco de los ocho pasos están demostrados en desarrollo, más la selección
y el aviso local del sexto.** Se usa la aplicación conectada a PostgreSQL:
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
| 6. Llamamiento | Parcial: selección, registro de aviso local y recuperación de los mismos recibos tras reiniciar. Sin envío, entrega, aceptación ni plazo acreditados. |
| 7. Nombramiento / formalización | No recorrible de extremo a extremo: pendientes aceptación, modelos y circuito de firmas. |
| 8. Incorporación, GINPIX y seguimiento | No recorrible de extremo a extremo: pendientes confirmaciones y conexión funcional de la salida. |

La **bandeja de expedientes está cerrada para este alcance de desarrollo**:
lista 50 expedientes y abre el detalle real en `8443`/`55433`. Esto no convierte
los datos sintéticos en expedientes reales ni amplía los cinco pasos y parte
del sexto. Para recuperar el llamamiento se usan las referencias conservadas
en la guía.

Este manual explica el trabajo funcional. Los comandos de arranque,
certificados, datos sintéticos de ejemplo y comprobaciones de persistencia
están en la [guía del recorrido manual](../../GUIA_RECORRIDO_ALBERTO.md).
El [manual de usuario de Bolsa](../manual_usuario/manual_portal_bolsas.md)
complementa la navegación de ese módulo; sus pantallas de demostración no
acreditan funciones reales de Contratación temporal.

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

### 6. Seleccionar y registrar el aviso local

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
completo de envío, entrega, aceptación, renuncia o vencimiento. No use las
opciones de una pantalla de demostración para simular esas decisiones.

### 7. Nombramiento: límite actual

El objetivo exige una propuesta aceptada y los documentos y firmas que
correspondan. El conjunto previsto incluye **informe definitivo, resolución,
diligencia, toma de posesión, notificación y comunicación al centro**.

No hay un recorrido completo demostrado de esos seis documentos y su firma.
Los modelos de desarrollo o un catálogo de demostración no son modelos
oficiales aprobados por RRHH. El informe sin firma del paso 5 no sustituye
al paquete de formalización. No marque aceptación, firma o notificación para
desbloquear artificialmente este paso.

### 8. Incorporación, GINPIX y seguimiento: límite actual

El objetivo es confirmar la incorporación efectiva, registrar la relación en
Personal y conservar la respuesta de GINPIX, además de los hechos posteriores
y el cierre. No basta con una fecha prevista ni con que exista un exportador.

La ficha descargable para carga manual es la salida mínima prevista, pero
todavía no se acredita aquí como recorrido completo disponible. Un fichero
generado no demuestra una carga ni un alta en GINPIX. Faltan las conexiones,
confirmaciones y evidencias del recorrido; no registre incorporaciones o
cargas ficticias para darlo por terminado.

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
| Servicio o bandeja no disponible (`503`) | No significa «no hay expedientes». La bandeja sigue pendiente de cierre; comunique la incidencia sin volver a registrar el expediente. |
| Fiscalización desfavorable | Tramite la devolución a la unidad competente; no continúe al llamamiento. |
| Documento sin firma, aviso local o entrega pendiente | Conserve el alcance indicado. No los convierta en firma, aceptación ni inicio de plazo. |

Al comunicar una incidencia, indique entorno de desarrollo, pantalla, acción,
mensaje, momento, expediente y recibo si existe. No remita cuerpos completos
con datos personales ni material de autenticación.

## Fuentes y ayuda

- [Procedimiento normalizado de contratación temporal](../portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Petición de RRHH: transcripción y lectura funcional](../estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md).
- [Requisitos de acceso interno y separación de perfiles](../estudio_requisitos/acceso_interno_tecnicos_administracion.md).
- [Guía técnica para arrancar, recorrer y recuperar el caso sintético](../../GUIA_RECORRIDO_ALBERTO.md).
- [Manual de usuario del módulo Bolsa](../manual_usuario/manual_portal_bolsas.md).

El siguiente avance funcional pendiente es completar la respuesta del sexto
paso. Siguen faltando
la comunicación con entrega acreditada y la respuesta de la persona candidata
con sus reglas de plazo. Este manual no redefine esas reglas ni declara
aceptación por RRHH.
