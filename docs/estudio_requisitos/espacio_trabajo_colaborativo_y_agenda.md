# Espacio de trabajo colaborativo y agenda compartida

Fecha de registro: **17 de julio de 2026**.

Estado: **capacidad futura; fuera del alcance de la entrega prioritaria de Bolsa**.

## 1. Objetivo

El portal incorporará en una fase posterior un espacio de trabajo compartido
entre departamentos y unidades administrativas. Permitirá organizar tareas,
responsables, vencimientos, reuniones, avisos y seguimiento desde el mismo
núcleo del portal, sin reactivar el antiguo `workspace` agregado ni exponer una
instantánea transversal de RRHH.

El calendario colaborativo es distinto de:

- el calendario hábil y sus fuentes normativas;
- el calendario laboral, turnos, permisos y presencia de Cronos;
- los plazos jurídicos de un expediente;
- la agenda personal mantenida por un proveedor externo.

Podrán relacionarse mediante referencias y eventos autorizados, pero ninguna
de esas fuentes se copiará o reinterpretará como otra.

## 2. Capacidades previstas

- espacios por organización, unidad, equipo, proyecto o expediente;
- tareas y listas con título, descripción, responsable, participantes,
  prioridad, estado, fecha objetivo, dependencias y lista de comprobación;
- comentarios, menciones y adjuntos mediante el almacén documental común;
- agenda compartida con reuniones, hitos, recordatorios y disponibilidad
  publicada deliberadamente;
- vistas de lista, tablero, agenda y cronología sobre la misma información;
- avisos mediante los conectores comunes del núcleo;
- delegaciones y suplencias con fecha de inicio y fin;
- búsqueda, exportación e informes limitados por ámbito y finalidad;
- automatizaciones declarativas y versionadas, sin scripts arbitrarios;
- API, web, CLI y MCP sobre los mismos casos de uso y permisos.

## 3. Interoperabilidad

La integración se hará mediante puertos sustituibles. Como mínimo se estudiarán:

- iCalendar para importación y exportación;
- CalDAV cuando el servidor corporativo lo permita;
- Google Calendar mediante un conector OAuth separado;
- Microsoft 365/Exchange mediante un conector independiente;
- webhooks o bandeja de salida para sincronización asíncrona y reintentos.

El núcleo no conocerá tokens, URL ni modelos propios de Google o Microsoft.
Cada conector declarará alcance, cuenta técnica, consentimiento o base jurídica,
sentido de sincronización, política de conflictos, reintento y revocación. La
compatibilidad no implicará activar una salida a Internet desde zonas internas:
Sistemas deberá aprobar la pasarela o servicio de integración correspondiente.

## 4. Seguridad y protección de datos

- denegación por defecto y autorización por espacio, recurso, acción y campo;
- una persona solo ve espacios y tareas concedidos expresamente;
- la administración técnica no obtiene por defecto el contenido funcional;
- clasificación de cada espacio y adjunto, con compartimentos para materias
  reservadas;
- notificaciones minimizadas: el canal puede recibir un aviso y un enlace sin
  reproducir información sensible;
- registro de alta, lectura sensible, cambio, asignación, exportación y borrado
  lógico, separado de los registros técnicos;
- historial append-only para cambios relevantes; una edición no borra la
  responsabilidad o el estado anteriores;
- retención y archivo por serie o finalidad, no una conservación ilimitada;
- protección frente a enumeración, menciones masivas y exportaciones amplias;
- conectores externos sin acceso a módulos o campos no incluidos en su alcance.

No se sincronizarán con proveedores externos expedientes, datos de Cronos,
méritos, nóminas, geolocalización ni categorías especiales por el mero hecho de
estar relacionados con una tarea. Solo podrá salir la proyección mínima que
aprueben el responsable funcional, Sistemas, Seguridad y el DPD.

## 5. Modelo arquitectónico

Se implementará como módulo hexagonal acoplado únicamente a capacidades del
núcleo: identidad, organización, autorización, catálogos, documentos,
notificaciones, auditoría, eventos y búsqueda. Sus puertos iniciales serán:

- repositorio de espacios, tareas y agenda;
- directorio de participantes y ámbitos;
- agenda externa;
- notificaciones;
- almacenamiento documental;
- búsqueda y exportación;
- reloj y trabajos programados.

La fuente maestra de la tarea será el portal. La sincronización de agenda se
modelará como una proyección reconciliable, con identificadores de ambos lados,
versiones, recibos y estado de conflicto. Un fallo externo no confirmará una
operación local como sincronizada ni perderá el cambio pendiente.

## 6. Puertas antes de programar

1. Taller con RRHH y unidades administrativas para delimitar casos reales.
2. Inventario de calendarios corporativos y políticas de uso existentes.
3. Decisión sobre información admisible en servicios externos.
4. Matriz de espacios, roles, delegaciones, visibilidad y segregación.
5. Reglas de conservación, archivo, exportación y derecho de acceso.
6. Diseño de sincronización, conflictos, zonas de red y custodia de tokens.
7. Pruebas de accesibilidad, concurrencia, reintento, revocación y fuga entre
   espacios.

Hasta superar esas puertas, `/api/vec/workspace` seguirá fallando cerrado. La
capacidad futura no condiciona ni retrasa el núcleo y la Bolsa actuales.
