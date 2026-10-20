# Portal y Bolsa: manual de septiembre de 2026

Histórico: instrucciones y límites de septiembre de 2026. No describen el acceso ni el estado actuales. Consulte los [manuales por perfil](../manuales/README.md). Se conservan los antecedentes funcionales; se han retirado relatos de sesiones y contadores de operaciones.

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

## 4. Antecedentes de contratación temporal

El alcance de septiembre comprendía formularios de petición, análisis,
cobertura, asignación, informe, fiscalización y registros manuales de
llamamiento y respuesta. Eran comprobaciones parciales: no acreditaban el
recorrido único completo ni un nombramiento eficaz.

| Actuación | Garantía funcional | Límite del corte histórico |
| --- | --- | --- |
| Petición y análisis | Revisión de datos, confirmación expresa y recibo | La petición no sustituía el análisis de crédito. |
| Cobertura y asignación | Conservación de la decisión y de la unidad responsable | No acreditaban selección de una persona ni fiscalización. |
| Informe | Preparación del documento y registro de la actuación | Un informe preparado no estaba firmado por el mero hecho de guardarlo. |
| Fiscalización | Conformidad o reparo y subsanación separados | La subsanación no convertía el reparo en favorable. |
| Llamamiento | Selección ligada al expediente y recibo propio | El aviso local no acreditaba envío, entrega ni apertura de plazo. |
| Declaración de respuesta | Referencia, fecha y código de comprobación del correo recibido | El contenido del correo no se custodiaba ni se verificaban su origen o firma. |
| Resolución manual de aceptación o renuncia | Revisión expresa de la respuesta y su justificante | El criterio provisional no acreditaba un plazo legal ni una respuesta propia en el portal. |
| Continuación tras renuncia | Nuevo llamamiento con su recibo, conservando la renuncia anterior | No avisaba ni aceptaba automáticamente por la siguiente persona. |
| Respuesta de la siguiente persona | Declaración y resolución separadas, con antecedentes conservados | No modificaba la respuesta inicial ni iniciaba una propuesta automáticamente. |
| Propuesta de nombramiento | Antecedentes derivados y confirmación con recibo propio | Seguía siendo un borrador, sin firma, envío ni incorporación. |

### Consultar borradores

El corte ofrecía el informe definitivo, la resolución, la diligencia, la toma
de posesión, la notificación y la comunicación al centro como borradores.
Una descarga recuperaba el documento asociado a la propuesta; no creaba otra
propuesta ni repetía el llamamiento. La notificación descargada no acreditaba
su entrega y la comunicación al centro no confirmaba una incorporación.

La consulta de un borrador y la recuperación de un PDF firmado son distintas.
Las instrucciones actuales de descarga están en
[consulta del expediente y documentos](../manuales/responsable_rrhh.md).

### Recuperar después de una interrupción

La recuperación autorizada de una actuación ya registrada debía devolver el
mismo recibo y su fecha original, sin crear otra actuación ni modificar sus
antecedentes. Una duda sobre el resultado no autorizaba a cambiar la clave o
los datos para forzar otro registro.

Las comprobaciones históricas de recuperación eran parciales. No acreditaban
la conservación de todo el recorrido vigente después de reiniciar.
El formulario que aún no se había confirmado no se guardaba automáticamente
en el navegador; al cerrar o recargar podía ser necesario volver a introducir
sus datos. No correspondía reiniciar ni recrear el servicio para resolver una
duda de registro: primero había que consultar el resultado existente.

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
