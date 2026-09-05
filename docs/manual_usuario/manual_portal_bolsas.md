# Manual de usuario · Portal VEC y Bolsas de trabajo

Diputación de Granada · Ventanilla Electrónica del Empleado Público

**Edición: 5 de septiembre de 2026.** Alcance de referencia: versión
`b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`. Este manual explica qué puede recorrer una persona desde el
portal, qué resultados debe esperar y qué opciones siguen pendientes.
Describe un entorno de desarrollo con datos sintéticos, no un servicio
autorizado para tramitar expedientes de personas reales.

**Disponible: cinco pasos completos de Contratación temporal y una parte
del sexto: selección, apertura de llamamiento y aviso local.** No están
completados el llamamiento corporativo, el nombramiento ni la incorporación.
La bandeja real lista 50 expedientes y abre su detalle en el recorrido local
`8443`/base `55433`; no incrementa el contador de pasos.

Para elegir la documentación adecuada:

- Este manual: acceso, navegación, resultados visibles y ayuda del portal.
- [Manual de Recursos Humanos](../manual_rrhh/README.md): procedimiento,
  responsabilidades y recorrido de tramitación por perfiles.
- [Guía de recorrido de Alberto](../../GUIA_RECORRIDO_ALBERTO.md): comandos
  exactos de preparación y arranque, datos sintéticos conservados y
  comprobación del recorrido después de reiniciar. La preparación del
  servidor corresponde a Sistemas; no necesita ejecutar sus comandos para
  utilizar un entorno que ya le hayan preparado.

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
| Contratación temporal | Recorrido real de desarrollo descrito en el apartado 4, con recibos y persistencia. |
| Cuadro y detalle de expedientes de Contratación temporal | Bandeja real de desarrollo: 50 expedientes y detalle consultable en `8443`/`55433`, con datos sintéticos. |
| Gestión interna de Bolsas | Pantallas de presentación y componentes reales todavía sin ensamblar como gestión completa. Consulte el estado de cada opción en el apartado 5. |
| Consulta pública de convocatorias | Listado, filtros, detalle y documentos cuando el servicio público esté habilitado. Compruebe el aviso de la fuente; no permite tramitar una candidatura personal. |
| Otros módulos del portal | Solo están disponibles si el servidor los habilita para su perfil. No se consideran terminados por aparecer en la portada. |

No utilice datos de personas reales, documentos de identidad, teléfonos,
correos ni expedientes reales en ninguno de estos recorridos de desarrollo.

## 2. Acceso y finalización de la sesión

### Entrar al portal interno

1. Pida a Sistemas la dirección del entorno, su certificado de desarrollo y
   el perfil de navegador preparado. No comparta certificados ni contraseñas.
2. Abra la dirección indicada y entre en `/portal-empleado/`, sin seleccionar
   el modo de presentación. En el entorno de la guía, después de preparar
   el túnel, la dirección es
   `https://localhost:18443/portal-empleado/`.
3. Utilice el certificado correspondiente a su función. Recursos Humanos e
   Intervención usan certificados y perfiles de navegador separados.
4. Espere a que el portal compruebe los módulos disponibles. Entre desde
   **Inicio del portal** en **Contratación temporal** y, para iniciar el
   recorrido, en **Nueva petición**.

La conexión exige un certificado de cliente válido. Si el navegador indica
que falta el certificado o no reconoce el servidor, pida ayuda a Sistemas:
no omita la advertencia de seguridad ni cambie la dirección para sortearla.

El perfil y los permisos los determina el servidor. Escribir otro nombre de
perfil en la dirección, cambiar una referencia o seleccionar un perfil de
presentación no concede permisos reales.

### Consultar la zona pública

Si Sistemas ha habilitado la consulta pública, abra `/bolsa/`. Esa zona
muestra información de convocatorias, no los expedientes internos de RRHH
ni la ficha privada de cada aspirante. Su disponibilidad es independiente
del recorrido interno de Contratación temporal.

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
  No confunda el llamamiento de Contratación temporal con el asistente de
  presentación de Bolsa: son accesos distintos.
- **A+** aumenta o restablece el texto y **Contraste** activa o desactiva el
  alto contraste. Estos ajustes afectan a la página abierta; no se guardan
  como preferencias para otra sesión.
- **Avisos** muestra los avisos accesibles. Un guion o un mensaje de fuente
  no disponible no equivale a «cero asuntos pendientes».
- **Ayuda** abre una explicación del portal. **Cerrar** permite volver a la
  pantalla sin realizar una actuación.

Si el panel de Bolsa no carga, no suponga que ha perdido todos los permisos
del portal: la disponibilidad de Contratación temporal se comprueba por
separado.

## 4. Recorrido real de Contratación temporal

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
| 6. Llamamiento, parcialmente disponible | Iniciar la selección desde un expediente fiscalizado favorablemente y registrar su comunicación local. | Orden, propuesta, llamamiento abierto y aviso local persistentes. No acredita correo enviado, entrega, aceptación, renuncia ni gestión completa del plazo. |
| 7. Nombramiento | Pendiente como recorrido completo. | No se ofrece una formalización terminada ni sus seis documentos completos. |
| 8. Incorporación y seguimiento | Pendiente como recorrido completo. | No se acredita incorporación, integración con GINPIX ni cierre del seguimiento. |

El número de recibos o la versión del expediente no es el número de pasos
completados: el paso 5 contiene más de una actuación.

### Iniciar y continuar una petición

1. En **Contratación temporal → Nueva petición**, complete el formulario con
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
   **Contratación temporal**, introduzca la referencia del expediente y su
   versión remitida `5`; pulse **Abrir fiscalización**.

Fiscalización admite **Favorable**, **Favorable con observaciones** y
**Desfavorable**. Los dos últimos requieren observaciones. Un resultado
desfavorable devuelve el expediente a la unidad con estado de incidencia;
no habilita el llamamiento favorable ni constituye un error de guardado.

No registre otra solicitud solo para volver a abrir la anterior. La bandeja
permite localizar el expediente y abrir su detalle; use las referencias
conservadas y las entradas indicadas en la guía.

### Iniciar o recuperar el llamamiento y su aviso local

1. Con el perfil de RRHH, abra **Contratación temporal → Nueva petición →
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

### Qué ocurre después de un reinicio

Los resultados confirmados de este recorrido se conservan en el servidor. En
la comprobación de cierre, la solicitud `v1` llevó al análisis `201`/`v2` y,
tras reiniciar, se recuperó un único recibo, mediante lectura independiente
de PostgreSQL y navegador.
Con la misma instalación, datos y material de seguridad, una recuperación
autorizada del llamamiento o del aviso devuelve el mismo recibo y la fecha
original, sin crear otro efecto.

El formulario sin confirmar no se guarda automáticamente en el navegador.
Después de cerrar o recargar puede tener que introducir de nuevo sus datos.
La guía contiene el ejemplo y la comprobación del reinicio; no reinicie ni
recree el servidor por su cuenta.

## 5. Bolsas de trabajo: estado de todas las opciones

El menú organiza 17 vistas en diez grupos. La tabla distingue la pantalla
escrita de la función realmente utilizable en la versión de referencia.
Las opciones de presentación no sustituyen al recorrido real del apartado 4.

| Opción del menú | Uso y estado en esta edición |
|---|---|
| Elaboración y borradores | Tiene un formulario de guardado real separado de la DEMO, pero su recorrido completo no está habilitado en el entorno descrito. No dé un borrador por guardado sin confirmación del servicio. |
| Convocatorias, bases y calendario | Presentación de configuración y publicación. No acredita aprobación, firma ni publicación administrativa de unas bases. |
| Solicitudes y admisión | Presentación de revisión, admisión y subsanación de candidaturas a Bolsa. No es el alta real de una petición de personal de Contratación temporal. |
| Revisión de méritos | Presentación de méritos sintéticos. No registra una valoración administrativa real de una persona. |
| Alegaciones | Presentación de revisión y resolución. No presenta ni notifica una alegación real. |
| Importación Convoca | Simula lotes ya incluidos en la presentación. No procesa un archivo del equipo ni concilia personas con un sistema corporativo. |
| Llamamientos automáticos según bases y Reglamento | Asistente general de presentación en cuatro pasos. La vía real escrita bloquea la continuación tras confirmar la propuesta; no ofrece el llamamiento corporativo completo. |
| Contratos, ceses y reincorporaciones | Presentación. No registra un contrato, un cese o una reincorporación reales desde esta pantalla. |
| Reglas y versiones | Presentación de reglas. No cambia las reglas autorizadas del recorrido real. |
| Baremación y ranking | Presentación de cálculo y ordenación. No publica una lista oficial ni una puntuación administrativa. |
| Portal de consulta para candidatos | Presentación de una consulta minimizada. No es una sesión personal operativa para consultar la posición real o datos privados. |
| Cuadro de mando para dirección | Pantalla y consulta de datos escritas, pero todavía sin conectar en el entorno descrito. Sus indicadores DEMO no representan actividad real. |
| Estadísticas y explotación de datos | Presentación de agregados. No se ofrece como informe real sobre personas o expedientes. |
| Generación y firma de documentos | Presentación de plantillas y circuito de firma. No acredita firma, cotejo ni custodia jurídica. El informe real de desarrollo se obtiene en Contratación temporal y lleva su aviso sin validez jurídica. |
| Correo y mensajería | Presentación de canales y envíos. No envía correo ni acredita recepción. Es distinta del registro real de aviso local de Contratación temporal. |
| Auditoría y trazabilidad | Pantalla de presentación. Los hechos persistentes del recorrido real no convierten esta vista en un historial operativo completo. |
| Configuración, roles y permisos | Presentación de administración. Cambiar un ejemplo no concede permisos reales ni modifica la seguridad del servidor. |

### Explorar la presentación de llamamientos

Desde la presentación, entre en **Bolsas de trabajo → Llamamientos** y siga
el asistente: elegir necesidad, revisar la propuesta, configurar y revisar
la preparación. Sirve para conocer la disposición y el vocabulario de la
pantalla. Sus recibos y efectos siguen siendo DEMO.

En acceso real, un mensaje de «detalle no disponible» tras confirmar una
propuesta no autoriza a avanzar a configuración o comunicación. No cambie
al modo DEMO para completar aparentemente esa operación.

### Buscar una convocatoria pública

Cuando esté habilitada la zona `/bolsa/`:

1. Revise el aviso de la fuente para saber si está consultando una
   presentación.
2. Busque por texto y use los filtros de tipo, categoría, estado o plazo.
   Puede limpiar los filtros y recorrer las páginas de resultados.
3. Abra el detalle de la convocatoria y consulte su descripción, requisitos,
   plazos, publicación y documentos asociados.
4. Abra solo los documentos que la propia ficha ofrezca. Un archivo rotulado
   DEMO no es una base aprobada.
5. Si no hay resultados, compruebe los filtros. Si hay un error de servicio,
   la consulta no ha confirmado que no existan convocatorias.

Este listado no registra inscripciones, no comunica una selección y no
muestra la posición privada de una persona candidata.

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
- La falta de nombres o contactos en un recibo de selección es intencionada.
  No intente reconstruir identidades desde referencias ni consultar
  expedientes ajenos.
- No pegue referencias privadas en buscadores, servicios externos o
  incidencias públicas. Incluso las referencias sin nombre deben tratarse
  por el canal de soporte autorizado.
- Si una acción está bloqueada, no modifique direcciones, campos técnicos o
  perfiles de presentación para sortearlo.

## 8. Ayuda y accesibilidad

Use **Ayuda** en el menú o en el pie del portal para abrir la explicación
contextual. La ayuda de Bolsa incluye preguntas frecuentes, un audio servido
por el propio portal y su transcripción. Puede leer el texto sin reproducir
el audio.

Ese contenido explica el asistente general de Bolsa; no constituye una
confirmación de que sus cuatro pasos estén habilitados en operación real.
Para el recorrido disponible de Contratación temporal, consulte el apartado
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
