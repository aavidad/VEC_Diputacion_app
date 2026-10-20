# Portal del empleado: inventario y continuación

## Vigencia al 20 de octubre de 2026

Este módulo conserva su alcance y sigue después de Bolsa y Contratación
según el [plan general](../producto/PLAN_IMPLEMENTACION.md). El estado de cada
tarea lo fija el [tablero](../producto/TABLERO_TAREAS.md). Se trabaja en
copias locales con datos ficticios; cidonia está congelada.

El inventario de abajo conserva sus fechas, contratos y requisitos. Sus
ramas, responsables, órdenes de reanudación, horas y cifras pertenecen a esos
cortes: no son un encargo actual ni acreditan una instalación o un recorrido.
Antes de reanudar, dirección debe contrastar qué se ha integrado en `main`
y fijar una tarea con consumidor visible y comprobación propia. Lo pendiente
de RRHH sigue en [dudas.md](../../dudas.md), con valor provisional configurable
cuando corresponda; ninguna prueba con datos ficticios autoriza datos reales.

## Inventario histórico anterior al 20 de octubre de 2026

## Continuación del 4 de octubre de 2026

Este apartado actualiza el inventario del 1 de octubre conservado debajo.
Base comprobada: `origin/main@77a4e7470`.

- I-01 e I-03 ya están montadas: Inicio ofrece accesos propios y «Mis trámites»
  reúne las lecturas existentes de Cronos y Dietas. La PR #364 incorporó las
  hojas que el plan inicial conservaba como WIP; no hay que recuperarlas otra vez.
- Este corte corrige los accesos desde «Mi ficha»: Cronos y Dietas pueden abrirse
  en una sesión nueva, sin visitar antes sus pantallas. El catálogo y el estado
  de carga deciden la navegación; cada operación conserva su autorización en
  servidor. Un destino ausente o cuya carga falló sigue deshabilitado. Al entrar,
  el foco permanece en el contenido principal.
- I-04 ya consume referencias de actos en las historias propias de Personal;
  los documentos necesitan el vínculo autorizado con su versión. I-05
  no dispone de plazos administrativos en las lecturas actuales; las fechas de
  permiso o comisión no permiten deducirlos. I-06 conserva el recibo de Dietas,
  sin convertirlo en registro oficial o notificación legal.
- La parte de catálogos ES/EN de I-09 ya usa el lector común y `solicitudes.json`.
  El componente genérico de Solicitudes carece de consumidor productivo; esta
  comprobación no da por cerrado el trámite gobernado de I-08.

La continuación añade tres mejoras en «Mis trámites»: al caducar la sesión,
retira los datos de ambos paneles; el justificante de Dietas muestra la versión
exacta del recibo; y una devolución permite consultar motivo, etapa, versión y
fecha del hecho, con acceso a «Mis dietas» para revisarla. Esa fecha no fija un
plazo administrativo. No se añade una descarga ni se repite una escritura.

Validación de los accesos: pruebas focales de coordinación, composición y ficha;
Chrome del sistema en ES/EN, 1440/390 y ampliación al 200 %, con teclado y foco.
El navegador monta coordinador y Personal reales con respuestas HTTP sintéticas;
no acredita mTLS, PostgreSQL nominal ni instalación en la principal.

Estado a 1 de octubre de 2026. Equipo I. Base comprobada:
`origin/main@0a62a3ea68e58fbf890885a2f59cc80343107e18`.
La orden de dirección de las 16:50 y 16:55 es conservar el trabajo, entregar
este plan y parar el módulo. No se han modificado padres, manifiestos, permisos,
SQL ni servicios. Ninguna de las tres capacidades EMP está cerrada. EMP-004 (trabajo colaborativo) y EMP-005 (agenda compartida) quedan fuera de este encargo y de la estimación.

Fuentes: [catálogo de capacidades (PRD, apartado 9)](../producto/PRD.md),
[análisis integral, apartados 17–21](../estudio_requisitos/analisis_integral_rrhh.md),
[petición de RRHH](../estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md),
[matriz normativa](../estudio_requisitos/matriz_normativa_rrhh_2026.md) y
[modelo histórico de organización y RPT](../estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md).

## Lo que ya está en main

| Pieza y rutas | Estado real |
| --- | --- |
| Personal: `internal/modules/personal/adapters/httpinterno/ficha_propia.go`, `internal/app/bootstrap/personal_empleado.go`; web `modulos/personal/cliente-http-ficha-propia.js` y `vista-ficha-integral.js` | Consulta propia `GET /api/interna/personal/mi-ficha`, sin referencia de persona en la petición. Relaciones, puesto, unidad y servicios con procedencia. Identidad, autorización y auditoría en servidor. Código compuesto; la configuración y el recorrido del empleado concreto requieren validación. |
| Shell: `portal-composicion-empleado.js` y `portal-modulos-coordinador.js` en `web/static/portal-empleado/` | Monta Personal, Cronos y Dietas y conserva sus rutas directas. Los tres se cargan al abrirlos, pero están ocultos en menú e Inicio por la presentación histórica de Bolsa/Contratación. La ficha personal ya enlaza Cronos y Dietas. |
| Cronos: `modulos/cronos/cliente-solicitudes-http.js`, `vista-permisos-propios.js`, `vista-avisos-propios.js` | Solicitudes, historial, circuito, justificación pendiente y avisos propios. Consulta de permisos por año, sin cursor; las escrituras devuelven recibo, pero el listado no lo incluye. No duplicar sus formularios ni presentar un aviso como notificación legal. |
| Dietas: `modulos/dietas/cliente-borradores-http.js`, `vista-borradores-propios.js`, `vista-recorridos.js` | Comisiones propias paginadas, borrador, envío y corrección; la lista devuelve comisión y recibo. Una relación ambigua exige resolverla en el recorrido existente. Fiscalizada no significa pagada. |
| Solicitudes genéricas: `modulos/solicitudes/vista.js`, `i18n.js`, `solicitudes.css` | Componente sin consumidor productivo ni API común. Nueva solicitud, aportar documentación y certificados permanecen deshabilitados. Su i18n y fechas todavía necesitan adaptación al sistema común. No conectarlo a `datos-presentacion.js` como fuente real. |
| Responsables: `personal/application/competencias_asignacion_dietas.go`, `dietas/ports/circuito_comision.go`, `cronos/application/resolucion_permiso.go` | Existen asignaciones y circuitos específicos. La asignación de responsable en Personal no concede competencia. Dietas usa `FuenteCompetenciaCircuitoSinCatalogo`; Cronos conserva resolutores con vigencia para permisos. No hay autoridad general de delegación temporal ni cobertura real de unidad. |

Las PR históricas [#44](https://github.com/aavidad/VEC_Diputacion_app/pull/44),
[#48](https://github.com/aavidad/VEC_Diputacion_app/pull/48) y
[#31](https://github.com/aavidad/VEC_Diputacion_app/pull/31) están incorporadas.
Las ramas históricas de solicitudes y `demo-jefes-20260929` no aportan una
implementación pendiente que deba reconstruirse.

## Huecos del catálogo

| Capacidad | Lo que falta |
| --- | --- |
| EMP-001 · Carpeta personal | Hacer localizable la ficha propia desde Inicio. Actos, documentos y solicitudes necesitan proyecciones propias de sus módulos, con procedencia y permisos; no reutilizar la lectura RRHH. Tiempo, formación y economía sin fuente no se ofrecen como datos disponibles. |
| EMP-002 · Bandeja y solicitudes | Una consulta propia localizable que reutilice Cronos/Dietas. La bandeja común no tiene todavía formularios gobernados, plazos administrativos, tareas, subsanaciones, decisiones ni notificaciones generales. El recibo interno de operación no sustituye un asiento de registro oficial. |
| EMP-003 · Responsable de unidad | Fuente acreditada de competencia por unidad y procedimiento, titular y suplente temporal; después tareas y cobertura mínimas. Su implementación permanece detenida: Organización produce la fuente de responsables/delegaciones/suplencias (ORG-001), D aplica la autorización común e El portal solo consume. Nunca expediente completo, diagnóstico, nómina ni méritos de Bolsa. |

## Minitareas, por orden de valor

| Orden | Responsabilidad y archivos previstos | Dependencia y criterio de cierre |
| --- | --- | --- |
| I-01 | Montar la hoja de accesos propios ya preparada. `portal-accesos-empleado.js`, `portal-inicio.js`, `portal-modulos-coordinador.js`, `portal.js`, `index.html` y manifiestos internos. | Separar ruta navegable registrada/diferida de vista montada y de operación autorizada. Lista explícita de autoservicio: no usar `VISTAS_MODULOS_PERSONALES`, que incluye gestión. Al abrir Cronos, omitir bandejas de RRHH mientras no haya señal positiva específica del servidor, coordinado con E. Cierre: enlaces desde una sesión nueva, sin consultas propias al abrir Inicio, carga al clicar, errores/denegación, volver e historial, ES/EN, teclado y escritorio/móvil. Renovar `?v=` de toda la cadena afectada. |
| I-02 | Verificar y corregir la vista WIP de trámites. `modulos/solicitudes/vista-tramites-propios.js`, su prueba y `textos/{es,en}/tramites-empleado.json`. | Fuente WIP `2668e0bd1c7348050e950b785624bc6c822d567c` ya revisada, sin montaje; antes de ampliar, ejecutar las 13 pruebas ya escritas y comprobar foco del año frente a una respuesta tardía. Cierre: dos paneles independientes, error parcial, relación ambigua, año, páginas/cursor, cancelación y Chrome. No formularios ni datos sintéticos en composición real. |
| I-03 | Componer fuente y vista de trámites, solo al abrir la nueva vista. `fuente-tramites-propios.js`, vista anterior y padres del portal. | I-01/I-02 y turno de compartidos. Inyectar únicamente lecturas propias existentes: Cronos `consultarPermisos({anio},{signal})`; Dietas `listar({limit:20,cursor},{signal})`. Cronos pagina en pantalla el resultado anual; Dietas conserva su cursor. Sin total ni cronología conjunta. Ante relación ambigua, remitir a Mis dietas sin seleccionar la primera. Cierre: montaje alcanzable, solo GET propios, recibo Dietas idéntico al recibido, denegación sin datos y dos revisiones independientes del hash final por datos personales. |
| I-04 | Consumir actos y documentos propios producidos por Personal. Previsión propia: adaptador web `portal-fuentes-carpeta.js` y montaje `portal-composicion-empleado.js`; Personal conserva los contratos, proyecciones y cambios de su ficha existente. | Personal produce actos/documentos propios; El portal solo consume su contrato autorizado, coordinando el montaje en la ficha de Personal sin escribir sus hojas por cuenta propia. Las PR de Personal [#298](https://github.com/aavidad/VEC_Diputacion_app/pull/298)/[#299](https://github.com/aavidad/VEC_Diputacion_app/pull/299) siguen abiertas en borrador, sin fusionar, y preparan ficha propia y trazas RRHH; [#300](https://github.com/aavidad/VEC_Diputacion_app/pull/300) también está abierta en borrador, no es una proyección propia montada. Cierre: documentos/actos del actor autorizado con fuente, versión y descarga real, sin campos RRHH prestados. |
| I-05 | Mostrar tarea y plazo administrativo de cada solicitud cuando su módulo los publique. Previsión: fuente/vista de trámites y puertos propietarios. | Fuentes de cada módulo y reglas aprobadas por RRHH. No deducir vencimiento de fechas de permiso o comisión. Cierre: cada fecha tiene hecho inicial, calendario, regla/versiones y siguiente actuación definidos; un dato ausente se muestra como no disponible. |
| I-06 | Recuperar justificantes conservados y separar avisos de notificaciones. Previsión: lectura propia de recibos/documentos en el módulo dueño y enlaces desde trámites. | Cronos/Dietas, Documentos, Registro y Notificaciones comunes. Cierre: consulta posterior devuelve bytes o referencia acreditada, sin repetir una escritura; no llamar registro, firma, notificación o entrega a lo que no lo acredita. |
| I-07 | Consumir la fuente de Organización y la autorización de los servicios comunes de autorización en el espacio de responsable. Previsión propia: vista/adaptador de lectura del portal; los puertos y adaptadores propietarios permanecen en Organización, Autorización y el módulo de cada tarea. | Organización produce responsables, delegaciones y suplencias históricas (ORG-001); Autorización valida el acceso; cada módulo produce sus tareas y Personal la relación laboral cuando proceda. El portal no fabrica la fuente ni concede permisos. Sin esta fuente no se ofrece gestión por etiqueta de puesto ni existencia de una función. Cierre: unidad, procedimiento, acciones, acto, vigencia y revocación acreditados; delegación expirada o retirada denegada, auditoría y campos mínimos; dos revisiones sensibles. |
| I-08 | Montar el primer trámite gobernado distinto de Cronos/Dietas en la superficie de solicitudes existente. Previsión: `modulos/solicitudes/vista.js`, i18n común, esquema/catálogo recibido y cliente del procedimiento propietario. | I-09 adapta antes los textos existentes. RRHH identifica el trámite; NUC-008/NUC-014 y módulo dueño aportan esquema, validación, borrador, firma/registro y permisos. El portal no inventa otro motor ni duplica formularios. Cierre: completar, corregir, conservar/recuperar borrador y presentar con justificante real; vuelta a bandeja con estados, plazos y decisión de I-05/I-06. La primera entrega lleva un consumidor aprobado, no un formulario genérico sin procedimiento. |
| I-09 | Pasar `modulos/solicitudes/i18n.js` a catálogos por idioma y fechas comunes. Previsión: esa hoja, `vista.js`/pruebas de Solicitudes y `textos/{es,en}/solicitudes-empleado.json`. | Lector/idioma comunes e integración de la superficie existente. Se ejecuta antes de I-08 y conserva los textos de la superficie existente. Cierre: sin diccionarios ni idioma compilado, claves ES/EN completas, fechas localizadas y consumidor existente de solicitudes con carga/errores verificados. |

Los cambios previstos en varios padres de I-01/I-03 son la cadena de montaje y
caché de una misma función visible; no abren módulos paralelos. No hay SQL nuevo
del portal del empleado. Si Personal o Autorización requieren SQL para proyecciones o competencia, su dueño reserva el
número y fija su dependencia en `ORDEN_SQL_NUCLEO.md`. Solo borrador; ensayo en
clon y revisiones antes de instalar. El portal no consulta tablas ajenas.

## Decisiones pendientes de RRHH

Se conservan aquí para llevarlas a `dudas.md` si todavía no están recogidas. Su presencia aquí no equivale a una aprobación
ni autoriza duplicar preguntas.

- Fuente de responsables y suplentes: quién acredita unidad/procedimiento,
  acciones, titular, sustituto, inicio, fin, acto y retirada. Organización produce la fuente histórica y los servicios comunes de autorización mantienen la autoridad de acceso.
- Qué primer trámite gobernado debe ofrecerse sin duplicar Cronos/Dietas, y qué actos y documentos oficiales puede consultar o descargar cada empleado,
  y cómo solicita una corrección sin reescribir el dato histórico.
- Para cada tipo de solicitud: quién actúa, qué hecho inicia el plazo, qué
  calendario/regla aplica y dónde se conservan decisión, justificante y notificación.
- Qué cobertura y tareas necesita la jefatura y qué campos admite el DPD.
  No se habilita una ficha completa por ocupar un puesto de responsabilidad.

## Condiciones funcionales conservadas

El portal consume fuentes autorizadas de cada módulo, con carga diferida y una lista explícita de autoservicio. Cronos conserva su consulta anual y Dietas su cursor; no se fabrica una cronología ni un total conjunto. Inicio no consulta datos propios innecesarios. La autoridad de responsables, delegaciones y suplencias procede de Organización y de sus permisos; no nace del cargo mostrado en una ficha.
