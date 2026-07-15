# Decisión de trabajo: espacios de acceso y persistencia intercambiable

Estado: **base aceptada para el estudio; pendiente de aprobación en las
especificaciones definitivas**.

Fecha: 14 de julio de 2026.

## 1. Una persona común, varios contextos sin suma de privilegios

Existirá un registro canónico de persona identificado mediante un `id_persona`
interno, inmutable y sin significado externo. El DNI o NIE será un atributo de
identidad protegido, no la clave primaria común de la aplicación.

Una misma persona podrá tener varias vinculaciones:

- aspirante o integrante de una bolsa;
- personal empleado;
- responsable de una unidad;
- representante de otra persona;
- personal tramitador, técnico o resolutor de RRHH;
- personal de soporte, auditoría o administración.

Estas vinculaciones no producirán una unión automática de permisos. Toda sesión
tendrá un único perfil activo, un ámbito y una vigencia. El cambio a un perfil de
mayor riesgo será explícito, quedará auditado y podrá exigir autenticación
reforzada.

Ejemplo: una técnica de RRHH que entre como empleada solo podrá consultar sus
propias nóminas, solicitudes, permisos y expediente. Para tramitar una bolsa
deberá entrar en el contexto de RRHH; aun así, el sistema bloqueará cualquier
actuación sobre su propio expediente o sobre otro en el que exista conflicto de
interés.

## 2. Espacios de producto diferenciados

Compartirán núcleo y datos autorizados, pero no serán un único frontal que se
limite a ocultar opciones de menú.

| Espacio | Destinatarios | Información y operaciones |
| --- | --- | --- |
| Portal público de empleo | Cualquier persona, incluido el bot público | Convocatorias, bases, requisitos, plazos, ayudas, anuncios y resultados legalmente publicables y minimizados. |
| Área personal del aspirante | Persona identificada o su representante acreditado | Solo sus datos, solicitudes, méritos, documentos, autobaremaciones, subsanaciones, alegaciones, llamamientos y justificantes. |
| Portal del personal empleado | Personal con vinculación vigente o histórica cuando proceda | Autoservicio propio: expediente personal, certificados, nóminas, dietas, permisos, formación, movilidad, notificaciones y solicitudes internas. |
| Espacio del responsable | Persona con responsabilidad o delegación vigente | Tareas y datos mínimos de las unidades y personas sobre las que tenga competencia: autorizaciones, cobertura y calendario; no acceso general a nómina, salud o expedientes. |
| Backoffice de RRHH | Personal tramitador, técnico, tribunal, resolutor, firmante y publicación | Expedientes y operaciones limitados por procedimiento, convocatoria, fase, unidad, tarea, tipo documental y periodo. |
| Administración funcional | Personal autorizado para configurar la aplicación | Catálogos, formularios, flujos, baremos, calendarios, plantillas, políticas y roles; con versión, simulación, revisión y doble aprobación. |
| Administración técnica y seguridad | Sistemas, IAM, seguridad y auditoría | Infraestructura, identidades, integraciones, observabilidad y evidencias, sin acceso ordinario al contenido de los expedientes. |

Un empleado también podrá presentarse a un proceso selectivo. Lo hará en el
contexto de aspirante, sin heredar sus permisos internos ni recibir ventajas de
acceso a información no publicada.

## 3. Fronteras técnicas mínimas

- Pasarelas y audiencias de credenciales distintas para portal público, área
  personal, portal del empleado, backoffice y administración.
- Cookies, políticas de contenido, límites, sesiones y claves separadas por
  superficie.
- La API pública leerá una proyección de publicación separada; nunca consultará
  directamente expedientes, personas o el almacén documental privado.
- El backoffice y la administración no se expondrán mediante la pasarela
  pública. Su acceso requerirá red o acceso remoto corporativo protegido y
  autenticación reforzada según el riesgo.
- Web, aplicación de escritorio, API, CLI y MCP invocarán los mismos casos de
  uso y el mismo motor de autorización. Ningún adaptador tendrá permisos
  implícitos por el canal utilizado.
- Se aplicará denegación por defecto y autorización por acción, objeto, campo,
  relación, finalidad, ámbito, vigencia y nivel de autenticación.
- El perfil activo será inequívoco en pantalla. No existirá suplantación
  silenciosa por soporte ni un rol permanente de superadministración universal.
- Se registrarán tanto accesos permitidos como denegados a información sensible,
  cambios de perfil, delegaciones, exportaciones y accesos excepcionales.

## 4. Persistencia: PostgreSQL inicial y Oracle futuro

### Requisito

PostgreSQL será el primer motor. Oracle se considera una alternativa futura de
primer nivel y la arquitectura no podrá impedir su incorporación.

El núcleo de dominio y los casos de uso no conocerán:

- SQL ni nombres de tablas;
- secuencias, columnas de identidad o procedimientos del fabricante;
- tipos JSON, fechas o grandes objetos propios de un motor;
- paginación, bloqueos, índices o extensiones específicas;
- bibliotecas de conexión ni códigos de error del controlador.

Los puertos de salida expresarán operaciones de negocio y garantías necesarias,
por ejemplo obtener una convocatoria vigente, guardar una presentación de forma
atómica o reservar el siguiente llamamiento. No se creará un puerto genérico que
acepte SQL arbitrario.

### Contenido de cada adaptador

Cada adaptador PostgreSQL u Oracle incluirá, como una unidad versionada:

- implementación de los repositorios;
- esquema físico y migraciones propias;
- traducción de errores y restricciones;
- estrategia de transacciones, concurrencia y bloqueos;
- bandeja de salida transaccional e idempotencia;
- paginación, ordenación y búsquedas;
- comprobaciones de salud y versión;
- métricas sin datos personales;
- copia, restauración y procedimientos operativos específicos;
- optimizaciones necesarias sin contaminar el núcleo.

Ambos deberán superar el mismo conjunto de pruebas de contrato funcional,
transaccional y de concurrencia. También se probarán migraciones, carga,
restauración, recuperación ante fallo y reproducción exacta de baremos y
llamamientos.

### Selección y cambio

Una vez instalado y certificado un adaptador, un despliegue nuevo podrá elegirlo
mediante configuración, por ejemplo `motor_persistencia=postgresql` u
`motor_persistencia=oracle`, sin modificar ni recompilar el núcleo.

Esto no significa que trasladar un sistema ya poblado sea un simple cambio de
una variable. Una migración en producción entre PostgreSQL y Oracle exigirá:

1. crear y validar el esquema de destino;
2. transformar y transferir los datos y documentos referenciados;
3. conciliar recuentos, restricciones, huellas y expedientes;
4. ensayar rendimiento, copias y recuperación;
5. ejecutar un plan de corte y retorno;
6. conservar evidencias de la migración.

La aplicación podrá cambiar de motor sin reprogramar el negocio, pero los datos
existentes nunca se moverán sin un proyecto de migración controlado.

### Condiciones para considerar Oracle soportado

Oracle solo figurará como motor soportado cuando existan:

- adaptador y migraciones completos;
- controlador y licencias aprobados;
- pruebas de contrato y seguridad superadas;
- prueba representativa de carga y concurrencia;
- procedimientos de alta disponibilidad, copia, restauración y parcheo;
- documentación de operación y migración;
- personal o servicio responsable de su administración;
- validación del alcance ENS y de la configuración de auditoría y cifrado.

Hasta entonces será una capacidad prevista, no una compatibilidad declarada.

## 5. Consecuencias sobre la composición modular

- Los módulos consumirán capacidades comunes mediante puertos: persona,
  identidad, autorización, expediente, documento, firma, notificación,
  calendario, auditoría y persistencia.
- Un módulo no podrá acceder a las tablas de otro ni compartirlas como contrato.
- Los cambios de esquema serán propiedad del adaptador y del módulo responsable,
  con compatibilidad y migración versionadas.
- Añadir un tipo de certificado, formulario, estado permitido, baremo, plantilla
  o flujo ordinario se realizará mediante configuración funcional publicada, no
  mediante una migración de código.
- Añadir una capacidad de negocio genuinamente nueva sí requerirá un módulo o una
  nueva versión contractual; la configuración no sustituirá al diseño y las
  pruebas de software.
