# VEC: documento de requisitos de producto

Fuentes absorbidas el 13/10/2026:
`docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md` (retirado),
`docs/estudio_requisitos/analisis_integral_rrhh.md` y
`docs/estudio_requisitos/README.md` (reducidos a una nota), la parte de producto
de `ESPECIFICACIONES_AGENTES.md` y las decisiones de alcance, datos de personas
y administración de la memoria de dirección.

Fecha de corte del estado: 20 de octubre de 2026, sobre `origin/main@3ef7dee0b`.
Los requisitos absorbidos el 13/10 se conservan; este corte actualiza estado,
entorno de trabajo y aceptación sin ampliar el alcance del producto.

Este documento dice qué es VEC, para quién se hace, qué tiene que hacer cada
módulo, en qué orden se construye y cómo se sabe que una parte está terminada.
Reúne en un solo sitio los requisitos de producto que antes estaban repartidos
entre el análisis integral, el catálogo funcional, el índice del estudio de
requisitos, las especificaciones para agentes y los planes de módulo (ver el
apartado 20). Si algo de aquí choca con una orden más reciente de Alberto o con
el código, mandan la orden o el código, y se corrige este documento.

Lo leen todos los agentes (Claude y Codex) antes de programar, revisar o encargar
trabajo, y cualquier persona de RRHH que quiera saber qué se está construyendo.

## 1. Los seis documentos de producto

| Documento | Qué responde |
| --- | --- |
| `docs/producto/PRD.md` (este) | Qué es VEC, para quién, qué módulos, qué debe hacer cada uno, con qué prioridad y cuándo está terminado |
| `docs/producto/TRD.md` | Cómo está construido: pila, arquitectura, seguridad, servidores y despliegue |
| `docs/producto/FLUJO_APP.md` | Quién hace qué y en qué orden, por perfil y por recorrido |
| `docs/producto/DISENO_UI_UX.md` | Cómo se ven y se usan las pantallas |
| `docs/producto/ESQUEMA_BACKEND.md` | Esquemas, tablas, roles de base de datos, migraciones y API |
| `docs/producto/PLAN_IMPLEMENTACION.md` | Qué falta, en qué orden y quién lo hace |

Siguen siendo fuente propia, y este documento solo los resume:

- `docs/plan_modulos/<módulo>.md`: inventario, huecos y tareas de cada módulo.
- `docs/estudio_requisitos/decisiones_rrhh_2026-10-02.md`: lo que RRHH ya respondió.
- `dudas.md`: lo que sigue preguntado a RRHH, al DPD o a Informática.
- Las fichas de módulo y las matrices normativas de `docs/estudio_requisitos/` y
  `docs/portal_vec/`, con las citas legales completas.

## 2. Qué es VEC

VEC es la aplicación de Recursos Humanos de la Diputación de Granada. No es una
aplicación de nómina ni una suma de formularios sueltos: es una plataforma que
comparte identidad, persona, permisos, auditoría, documentos, firma,
calendarios, comunicaciones y procedencia de los datos, mientras cada materia
conserva sus propias reglas, permisos y ciclo de vida.

Tiene tres partes que trabajan sobre los mismos hechos, cada una con sus permisos:

- **La parte interna de RRHH**: contratación temporal, bolsas de trabajo, procesos
  selectivos, provisión de puestos, Personal, organización y RPT, control horario
  (Cronos), dietas, administración y auditoría.
- **El área del empleado**: su ficha, sus trámites, sus fichajes y sus dietas.
- **La parte pública y el área del aspirante**: convocatorias, bolsas, inscripción,
  méritos y la consulta propia de cada persona («Mi Bolsa»).

La persona es el centro. Una misma persona puede ser aspirante, empleada o las
dos cosas a la vez, sin duplicarse. Candidato, empleado o representante son
vínculos de esa persona, con fechas y revocables. Cada módulo guarda sus datos y
se relaciona con los demás por referencias y avisos, nunca leyendo tablas ajenas.

Separar así las materias permite cuatro cosas a la vez:

1. reutilizar los datos acreditados de una persona sin volver a pedirlos;
2. que Bolsa, Personal, RPT, Cronos o Nómina no interpreten el mismo hecho de
   formas distintas;
3. que un permiso para una finalidad no abra datos de otra;
4. que una norma, un convenio, unas bases o un módulo nuevos entren sin reescribir
   el núcleo ni alterar expedientes cerrados.

VEC no decide derechos por parecido, por una constante en el código ni por lo que
hace otra Administración. Toda decisión funcional tiene que poder contestar: qué
norma, convenio, acuerdo, bases o resolución estaba vigente; a qué personal,
puesto, centro y periodo se aplicaba; qué hechos y documentos se usaron; qué
regla versionada dio el resultado; quién lo revisó, aprobó o firmó; y cómo se
rectificó sin borrar el resultado anterior.

Hoy todos los datos son sintéticos (inventados pero verosímiles). El código es
de producción: no hay código de demostración que se tire después. VEC no está en
producción y no trata datos reales hasta que el Delegado de Protección de Datos
(DPD), Seguridad y RRHH completen la evaluación de impacto, la categoría del
Esquema Nacional de Seguridad (ENS), el análisis de riesgos y el registro de
actividades de tratamiento.

## 3. Mandato institucional

### Estrategia de modernización de 2025

La Estrategia integral de modernización, simplificación y optimización de la
gestión del personal, aprobada por la Junta de Gobierno el 18 de febrero de 2025,
es la fuente institucional prioritaria. Detecta más de 175 categorías o
especialidades, solapamientos, bolsas insuficientes, certificados de servicios
como cuello de botella, gestión reactiva de las necesidades y comunicaciones
informales. VEC tiene que ayudar de forma comprobable a:

- inventariar escalas, subescalas, categorías, especialidades, personal activo,
  vacantes, funciones y competencias, conservando la correspondencia histórica
  cuando se agrupen o extingan categorías;
- normalizar protocolos, impresos, responsables, canales y plazos;
- automatizar certificados sin perder revisión, firma y expediente;
- planificar necesidades y crédito antes de que llegue la urgencia;
- medir tiempos de procesos selectivos y de solicitudes de personal, y la
  cobertura de cada categoría mediante bolsas;
- ampliar y flexibilizar las bolsas conforme al reglamento vigente.

### Registro de actividades de tratamiento

El apartado 2.17 del registro de actividades de la Diputación incluye en Recursos
Humanos: toma de posesión, contratos, altas y bajas; nóminas, trienios, dietas y
anticipos; permisos, vacaciones, incompatibilidades y régimen disciplinario;
órganos de representación; selección y provisión; control horario y
geolocalización; formación, planes de pensiones y acción social; y prevención y
vigilancia de la salud. Eso acredita el alcance, pero no autoriza un acceso común:
salud, geolocalización, biometría, denuncias y afiliación sindical van en
compartimentos con perfiles y registros de acceso propios. El registro tendrá que
actualizarse para cada finalidad nueva.

### Sistemas actuales

VEC convive con GINPIX 7/SAVIA, CONVOCA, WCRONOS, Portale, MOAD, la sede
electrónica, el registro, el portafirmas, Active Directory/Kerberos y el correo
corporativo. Antes de integrar datos reales de cualquiera de ellos se hace una
ficha con su propietario funcional y técnico, el dato maestro, la interfaz
disponible, los identificadores, el comportamiento ante fallo, la base jurídica
del intercambio y el plan de retirada. Ningún módulo escribe directamente en
GINPIX, nómina, WCRONOS o Active Directory.

## 4. Qué problema resuelve

| Hoy | Con VEC |
| --- | --- |
| Las bolsas y la inscripción se gestionan en CONVOCA, que ya no se mantiene ni se usa al día | VEC sustituye a CONVOCA entero: convocatorias, inscripción, baremación, bolsas y, más adelante, procesos selectivos y provisión. Las bolsas existentes se cargan desde los Excel que exporta RRHH |
| La contratación temporal circula por papel, correo y varias aplicaciones, sin un expediente común | Un expediente único desde la petición del centro hasta el cese, con firma, fiscalización, historia y recibos |
| Control horario en WCRONOS, una aplicación PHP 5 obsoleta | Cronos se reescribe |
| Dietas sin una aplicación que una a empleado, jefatura y RRHH | Dietas se crea de nuevo, con esas tres miradas |
| Cada aplicación tiene su acceso y sus permisos | Una sola identidad por certificado, permisos centralizados y auditoría de todo |
| Es difícil demostrar quién hizo qué | Cada operación, también las consultas, queda registrada con valor de prueba |

Lo que VEC no sustituye: GINPIX, SICAL (contabilidad), MOAD (que da el número de
expediente), el portafirmas, la sede ni el registro oficial de entrada. VEC se
conecta con ellos o prepara el dato para grabarlo a mano mientras no haya
integración (apartado 15).

## 5. Para quién

### Perfiles

| Perfil | Quién es | Por dónde entra |
| --- | --- | --- |
| Visitante | Cualquier ciudadano | Parte pública, sin identificarse |
| Aspirante o candidato | Persona que se inscribe en una bolsa o convocatoria; puede ser externa o empleada | Área del aspirante, con DNIe o certificado digital (nunca DNI y clave) |
| Representante | Quien actúa por otra persona con representación verificada | Área del aspirante, dentro del trámite y periodo representados |
| Empleado | Persona con relación de servicio con la Diputación | Área del empleado, con certificado |
| Responsable de unidad | Jefatura que organiza su servicio | Espacio de responsables (futuro): cobertura y tareas, nunca el expediente completo |
| Centro solicitante | Técnico de una Delegación o centro que pide personal | Portal interno, «Peticiones del centro» |
| Ratificador del centro | Diputado o Diputada de la Delegación | Portal interno, la misma petición |
| Técnico de RRHH | Quien tramita expedientes y bolsas; en los recorridos usa su perfil fijo de técnico de RRHH | Portal interno de RRHH |
| Jefatura de Servicio y Dirección de RRHH | Autorizan, informan, dan el visto bueno y firman | Portal interno de RRHH |
| Intervención | Fiscaliza el gasto: conforme, reparo y nueva fiscalización tras subsanar | Portal interno, desde la ficha del expediente |
| Firmantes | Jefatura, Diputada y quienes indiquen las delegaciones de firma | Firma con GrxFirma desde el navegador |
| Órganos de selección | Presidencia, secretaría, vocales y suplentes de un tribunal | Solo los procesos en que tienen nombramiento vigente (futuro, con Selectivos) |
| Administrador | Personas nominales que reparten perfiles y gobiernan catálogos | Administración, separada del portal de RRHH |
| Sistemas | Informática: arranque, copias, registros técnicos | Herramientas de operación, sin acceso funcional a expedientes |

La matriz de perfiles, ámbitos y segregaciones está en
`docs/producto/FLUJO_APP.md`, apartado 6. Quién hace cada paso y en qué orden
se cuenta en `docs/producto/FLUJO_APP.md`.

### Superficies

VEC separa las zonas por las que se entra, aunque compartan núcleo:

1. portal público, sin identificarse;
2. área del aspirante y de su representante;
3. área del empleado;
4. espacio de responsables de unidad;
5. tramitación interna de RRHH;
6. administración funcional;
7. administración técnica, seguridad y auditoría privilegiada.

Una sesión tiene un solo perfil activo y los permisos no se suman. Un empleado
que se inscribe en una bolsa entra como aspirante; un técnico de RRHH que mira su
propio recibo entra como empleado. Pertenecer a una unidad no permite ver todo su
expediente: una jefatura ve solo lo necesario para organizar el servicio
(disponibilidad, saldo suficiente, efecto operativo), nunca diagnósticos,
afiliación, nómina, méritos de bolsa ni denuncias. Cada módulo puede exponerse
por separado (por ejemplo, Cronos accesible desde fuera y Personal o Dietas solo
en la red interna); cómo se configura está en `docs/producto/TRD.md`.

### Qué necesita cada perfil

- **Aspirante.** Ver las convocatorias abiertas y saber si cumple los requisitos.
  Inscribirse con sus datos, títulos y méritos, que aporta una sola vez y
  reutiliza. Consultar en «Mi Bolsa» su estado en cada bolsa, su historial de
  llamamientos y las ofertas, y aceptar. Solo puede cambiar sus datos personales;
  cualquier otra cosa la pide a RRHH, que la valida (decisión de RRHH del 02/10).
- **Empleado.** Consultar su ficha, relaciones, servicios, documentos y trámites;
  más adelante, fichar, pedir permisos, gestionar dietas y ver recibos.
- **Centro.** Pedir personal eligiendo la necesidad de la RPT, ver en qué punto
  está su petición y confirmar la incorporación.
- **Ratificador.** Ratificar o devolver la petición de su Delegación.
- **Técnico de RRHH.** Una bandeja con lo pendiente. En Contratación: alta,
  análisis del crédito, llamamiento desde la bolsa real, informe, resolución,
  GINPIX, paso a Personal y cese. En Bolsa: cargar bolsas, revisar inscripciones
  y méritos, baremar, ver candidatos y su contacto completo en la ficha,
  configurar reglas y plazos.
- **Intervención.** Ver lo que le llega para fiscalizar, poner un reparo y volver
  a fiscalizar tras la subsanación.
- **Firmantes.** Firmar el documento correcto, en el orden correcto, sabiendo qué
  firman, y verificar la firma después.
- **Administrador.** Dar y quitar perfiles con constancia de quién y por qué;
  activar o desactivar módulos; copias de seguridad desde la pantalla.

## 6. Principios de producto

Son decisiones de Alberto y del estudio de requisitos que valen para todo VEC.
Cómo se consiguen técnicamente está en `docs/producto/TRD.md`.

1. **Una sola aplicación y una sola persona.** Los módulos se añaden registrándolos
   en la composición existente, sin otro acceso, otros permisos ni otra ficha.
   «Módulo» no obliga a carga dinámica ni a microservicios.
2. **Base primero.** Lo que otros módulos consumen se hace antes: identidad y
   perfiles, auditoría, Personal y Organización/RPT, firma, y después cada módulo.
   Dentro de eso, van delante las bases que necesitan Bolsa y Contratación.
3. **Lo justo, sin capas.** La solución más sencilla que funcione y sea rápida,
   sin rebajar la seguridad. Lo que no hace falta ya va a
   `docs/estudio_requisitos/pendientes_v2.md`.
4. **Configurable lo que RRHH no ha decidido.** Plazos, reglas, causas, plantillas,
   canales, formularios y categorías de la RPT están en catálogos versionados que
   RRHH cambia desde la pantalla, con historia y auditoría. Nunca una constante en
   el código. Solo las garantías técnicas quedan fijas.
5. **Una configuración nueva nunca impide arrancar.** Si falta, falla solo esa
   función y la pantalla explica qué falta.
6. **Terminar, no ocultar.** Lo que no funciona se termina; no se quita del menú.
   Cada botón funciona o explica en pantalla por qué no está disponible.
7. **Todo se ve en la web.** Subidas, firmas, descargas, PDF, Word y recibos se
   usan desde pantalla, no solo con herramientas de terminal.
8. **Datos sintéticos sin marcas.** Nombres verosímiles; nada visible dice
   «ejemplo», «prueba» ni «demo».
9. **No se inventa lo jurídico.** Un borrador no es un documento firmado,
   identificarse no es firmar, un aviso no es una notificación, una declaración
   no es una resolución y una referencia a un correo no es su custodia. No se
   inventan plazos, nombramientos eficaces ni incorporaciones.
10. **Los mismos casos de uso para todos los canales.** Web, línea de órdenes,
    escritorio y asistentes usan las mismas capacidades y permisos; ninguno se
    salta la autorización, la validación ni el registro.

### De dónde salen las reglas

El orden de autoridad es: derecho de la Unión Europea y legislación estatal;
legislación de Andalucía; BOP, acuerdos, convenio, reglamentos, resoluciones,
circulares, RPT, plantilla, bases y procedimientos de la Diputación; instrucciones
y funcionamiento real contrastado con la unidad; y, solo como referencia para
mejorar el diseño, lo que hacen otras Administraciones o productos. Una fuente
posterior no se presume consolidada con las anteriores. Una contradicción produce
un aviso jurídico o un resultado «no determinable», nunca una elección silenciosa
del texto más cómodo.

Antes de programar una regla funcional se hace la puerta institucional:

1. localizar la documentación oficial de la Diputación que se aplica;
2. comprobar modificaciones, derogaciones, fecha de efectos y ámbito;
3. identificar el órgano y el sistema dueños de cada dato;
4. registrar documento, versión, fecha de consulta, apartado y dudas;
5. conseguir casos públicos o anonimizados para las pruebas de aceptación;
6. validar con la unidad competente cualquier vacío o contradicción.

Lo que ya está publicado (BOE, BOP, Transparencia) se usa sin parar a preguntar;
a RRHH solo se le pregunta lo que no está publicado. Lo que falta se configura
con un valor provisional y la duda va a `dudas.md`.

### Cómo vive una regla configurable

Una regla publicada declara su identificador, versión, estado y ámbito; sus
fuentes y fecha de efectos; a quién se aplica y a quién no; qué hechos usa y de
dónde salen; operadores, unidades, redondeos, topes y prioridades; calendario;
documentos y revisiones exigidos; casos de prueba aprobados; y quién la prepara,
revisa, publica, suspende o sustituye. RRHH no escribe código, SQL ni fórmulas
libres.

```text
borrador → simulada → revisada → aprobada → publicada → vigente → sustituida o anulada
```

Cada expediente guarda la versión exacta de las reglas con que se tramitó.
Corregir una regla no recalcula en silencio actos firmes: genera un análisis de
impacto o una rectificación.

## 7. Modelo de datos que no se simplifica

### Persona, cuenta y relación

Son cosas distintas: la persona; sus identificadores legales protegidos; sus
cuentas de acceso; la representación de otra persona; su perfil de aspirante; su
relación de servicio; el régimen jurídico; el vínculo y la causa temporal; la
situación administrativa; el cargo o mandato; y la adscripción y ocupación de una
plaza o puesto. VEC distingue, como mínimo, funcionario de carrera e interino;
laboral fijo, indefinido y temporal; eventual; y personal directivo, alto cargo o
cargo electo. No hay un único campo «tipo de personal», y el vínculo real nunca se
deduce de una cuenta, un certificado, un DNI ni de la columna de la RPT.

### Categoría, plaza, puesto y ocupación

- **Categoría** (cuerpo, escala, subescala, especialidad): clasificación de la
  persona o de la plaza.
- **Plaza**: dotación de plantilla con presupuesto.
- **Puesto tipo** y **puesto individual** de la RPT.
- **Ocupación**: relación con fechas entre una persona y una plaza o puesto.
- **Reserva**: lo que impide tratar una ausencia como vacante libre.

Su historia guarda dos fechas: la de efectos administrativos y la de cuándo lo
supo el sistema. Vacante presupuestaria, puesto sin ocupante y necesidad cubrible
son tres cosas distintas. Igual de separados van el nivel del puesto, el grado
personal consolidado, el grado o nivel laboral del convenio y los grupos
funcionariales y laborales. No hay un examen periódico de «subida de nivel»: el
grado solo cambia por resolución e inscripción en el Registro de Personal. El
detalle está en `docs/estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md`.

### Quién es dueño de qué

| Área | Es dueña de | No puede |
| --- | --- | --- |
| Identidad y acceso | Cuentas, autenticación, sesiones, perfil activo, permisos y delegaciones de acceso | Convertir un grupo de Active Directory o un menú en permiso |
| Personas y representación | Persona, contacto, preferencias, representación y procedencia | Declarar una relación de servicio por ver una cuenta |
| Organización, plantilla y RPT | Organismos, unidades, centros, categorías, plazas, puestos y sus versiones | Decidir quién ocupa o cubre un puesto |
| Registro de Personal | Relaciones, nombramientos, contratos, ceses, situaciones, ocupaciones, antigüedad y grados reconocidos | Calcular nómina o imponer reglas de Bolsa |
| Planificación | Necesidades, escenarios, coste, crédito, oferta de empleo y cobertura | Crear una contratación o tocar el presupuesto sin expediente |
| Selección | Convocatorias, solicitudes, admisión, pruebas, tribunales, listas y alegaciones | Gestionar la disponibilidad de una bolsa ya constituida |
| Bolsa | Constitución, orden, disponibilidad, preferencias, llamamientos, renuncias y propuesta de incorporación | Inventar contratos, ceses o servicios |
| Registro de méritos | Títulos, experiencia y cursos, con evidencias, verificación y vigencia | Trasladar puntos de una convocatoria a otra |
| Baremación | Reglas publicadas, cálculo, revisión, rectificación, firma y explicación | Admitir código o fórmulas libres de RRHH |
| Provisión | Concursos, libre designación, movilidad y adjudicación | Reutilizar sin más el orden de una bolsa |
| Carrera | Grado, carrera, promoción interna y evaluación | Cambiar un grado o retribución sin resolución |
| Calendarios | Días naturales, hábiles, festivos y apertura de centros, con versiones | Decidir la jornada de una persona |
| Cronos | Cuadrantes, jornada, fichajes, teletrabajo, ausencias, permisos y saldos | Valorar importes de nómina o revelar causas médicas |
| Nómina | Conceptos, tablas, cálculo, cierre, cotización, IRPF y recibos | Reabrir una nómina cerrada |
| Dietas | Comisión de servicio, itinerario, anticipo, justificantes y liquidación | Usar la posición GPS como justificante del gasto |
| Formación | Planes, acciones, plazas, asistencia y certificados | Reconocer un grado o un mérito fuera de su procedimiento |
| Acción social | Convocatorias de ayuda, requisitos, presupuesto, comisión y concesión | Exponer datos familiares o de salud a otros módulos |
| Prevención | Riesgos, aptitud, vigilancia de la salud y accidentes | Entregar diagnósticos a RRHH o a una jefatura |
| Igualdad y protocolos reservados | Planes, indicadores y procedimientos confidenciales | Mezclar denuncias o víctimas con el expediente ordinario |
| Relaciones laborales | Negociación, representación, crédito sindical, incompatibilidades y disciplina | Convertir afiliación o denuncias en un dato general del empleado |
| Documentos y archivo | Documentos, expedientes, series, acceso y conservación | Dar acceso por compartir almacenamiento |
| Comunicaciones | Plantillas, destinatarios, canales, entregas y recibos | Tratar un correo o un Telegram como notificación administrativa |
| Analítica y transparencia | Indicadores, cuadros de mando y publicaciones minimizadas | Consultar tablas de trabajo sin una vista y finalidad aprobadas |

### Cómo se relacionan los módulos

```text
Cuenta autenticada → Persona
                       ├─ Candidatura(s) → Bolsa
                       └─ Empleado → Relación(es) de servicio
                                      └─ Adscripción u ocupación con fechas
                                           ├─ Comisión → Dietas
                                           └─ Fichaje o solicitud → Cronos
```

Las flechas son contratos, no consultas a tablas ajenas. Cambiar de contrato no
crea otra persona ni otro empleado. Cuando un trámite tiene que elegir relación,
el servidor la valida contra Personal; nunca se toma la primera fila.

| De → a | Qué se pasa | Qué no se confunde |
| --- | --- | --- |
| Contratación → Personal | Acto admitido, persona, relación, fechas y evidencia | Una propuesta no es un nombramiento eficaz |
| Personal → Bolsa | Servicios, relación o cese, jornada y fecha de efectos | Bolsa decide la disponibilidad con su regla; Contratación no cambia el orden |
| Personal y méritos → área del aspirante | Datos declarados o acreditados para evaluar requisitos, con fuente y vigencia | Requisito de acceso y puntuación son decisiones distintas |
| Personal y Organización → Cronos y Dietas | Empleado, relación, adscripción y responsable competente | No se deduce el empleado del DNI ni el aprobador del nombre del cargo |
| Cronos → Personal y Nómina | Incidencia de tiempo aprobada | Horas fichadas no son servicios reconocidos ni devengo |
| Personal → antigüedad y Nómina | Servicios reconocidos, resolución y efectos | No se calculan trienios sumando contratos |
| Formación → méritos y Carrera | Curso acreditado, horas, resultado y certificado | Inscrito, asistente y superado son tres estados; curso no es puntuación |
| Dietas → circuito económico | Liquidación aprobada y confirmación del pago | Solicitado, aprobado, liquidado y pagado son cuatro estados |
| Prevención → Personal y Cronos | Conclusión funcional (apto, apto con limitaciones, no apto) | Nunca el diagnóstico |
| Todos → archivo y auditoría | Referencias, metadatos y actuación | Compartir almacén no da acceso a documentos de otra finalidad |
| Módulos → analítica | Vista autorizada con definición y periodo | Un indicador no consulta libremente datos personales |

Un fallo de integración nunca se interpreta como cero, ausencia, vacante, permiso
o cumplimiento. La fuente maestra de cada dato se acuerda con RRHH y Sistemas;
no se presupone que las interfaces de otros sistemas existan.

## 8. Módulos: alcance, estado y prioridad

Cada módulo tiene un plan en `docs/plan_modulos/` con su estado real y lo que
falta; el orden de trabajo detallado está en `docs/producto/PLAN_IMPLEMENTACION.md`.
«Código en main» no significa activado en una copia ni recorrido en el
navegador. Se trabaja en local con datos ficticios; cidonia está congelada
y la próxima instalación irá a un servidor nuevo. La receta vigente es
[`LEEME_copia_local.md`](../../deploy/principal/LEEME_copia_local.md).

### Prioridad 1: Bolsa y Contratación temporal, con sus bases

| Módulo | Qué cubre | Estado al 20/10 | Plan |
| --- | --- | --- | --- |
| Contratación temporal | Expediente desde petición hasta cese | Integrados el llamamiento desde bolsa constituida, preparación de firma, incorporación y cese. Falta acreditar el caso único completo y sus variantes con reinicio | `docs/plan_modulos/contratacion_temporal.md` |
| Bolsa de trabajo | Bolsas, candidaturas, disponibilidad, ofertas, llamamientos, Mi Bolsa e importación de CONVOCA | Dos plazas aceptadas y adjudicadas por orden con recuperación tras reinicio en ensayo parcial. La continuación en CT y la vuelta a Bolsa del caso único siguen pendientes | `docs/plan_modulos/bolsa.md` |
| Inscripción, méritos y baremación | Solicitud, cartera personal, revisión, alegaciones e incorporación a bolsa | Código integrado y activación por copia. La entrada externa ensayada el 20/10 era de una cuenta sin participaciones; no acredita todos sus actos. La puntuación del baremador común queda fuera de este corte | `docs/plan_modulos/bolsa.md`, `docs/plan_modulos/selectivos.md` |
| Firmas | GrxFirma, verificación y recuperación de originales | Informe firmado y paso a fiscalización ensayados en copia local; consulta histórica integrada. Pendiente el expediente completo, incluida su resolución en el mismo recorrido | `docs/plan_modulos/firmas.md` |
| Identidad, perfiles y administración | Acceso por certificado y gobierno de perfiles | La copia compone Administración; cada capacidad exige su gobierno y configuración. El material ficticio de partida no acredita por sí solo actos administrativos | `docs/plan_modulos/administracion.md` |
| Auditoría común | Registro de lecturas y actos con cadena de huellas | Integrada en las operaciones conectadas. Activación y consulta de RRHH deben comprobarse en la copia del recorrido; exportación judicial aparcada | `docs/plan_modulos/auditoria.md` |
| Personal | Relaciones, ocupaciones, situaciones e incorporación | La copia prepara incorporación, alta, GINPIX manual y cese. Los ensayos parciales no cierran el tramo del caso único | `docs/plan_modulos/personal.md` |
| Organización y RPT | Unidades, centros, puestos y categorías | El caso de aceptación usa necesidad y puesto publicados. Consulta histórica preparada en la copia; el gobierno completo y los datos sembrados conservan sus límites declarados | `docs/plan_modulos/organizacion_rpt.md` |
| Documentos | Custodia, PDF, Word y originales firmados | Descargas desde la ficha comprobadas en ensayos parciales; cuatro documentos del caso siguen sin firma. Las plantillas oficiales esperan a RRHH | `docs/plan_modulos/contratacion_temporal.md` |

Evidencia y límites en [`ESTADO_PROYECTO.md`](../../ESTADO_PROYECTO.md). El
[`TABLERO_TAREAS.md`](TABLERO_TAREAS.md) conserva el estado de cada tarea;
las PR abiertas no se cuentan como integradas en este corte.

### Prioridad 2 y 3: Cronos, después Dietas

| Módulo | Qué cubre | Estado | Plan |
| --- | --- | --- | --- |
| Cronos | Fichajes, jornada, saldos, permisos, incidencias, equipo e informes | Aparcado hasta cerrar Bolsa y CT. Hay piezas y SQL sin instalar; ninguna de las doce capacidades está cerrada formalmente | `docs/plan_modulos/cronos.md` |
| Dietas | Solicitud y justificación, autorización y liquidación | Aparcado. La liquidación del alojamiento está bloqueada hasta que RRHH fije la regla (duda 104) | `docs/plan_modulos/dietas.md` |

### Después, uno a uno y cada uno tras sus bases

Aparcados hasta cerrar Bolsa y CT (orden de Alberto del 07/10). El orden previsto
es: Selectivos y Provisión (adelantando méritos, baremación y certificados);
Formación y Carrera; Certificados; Acción social; Nóminas (primero consulta,
luego cálculo); y Prevención, Igualdad, Incompatibilidades, Régimen disciplinario,
Relaciones sindicales y Canal interno.

| Módulo | Plan |
| --- | --- |
| Procesos selectivos | `docs/plan_modulos/selectivos.md` |
| Provisión | `docs/plan_modulos/provision.md` |
| Formación y carrera | `docs/plan_modulos/formacion.md`, `docs/plan_modulos/carrera_formacion.md` |
| Certificados y analítica | `docs/plan_modulos/certificados_analitica.md` |
| Portal del empleado | `docs/plan_modulos/portal_empleado.md` |
| Copias de seguridad | `docs/plan_modulos/copias_seguridad.md` |
| Acción social | `docs/plan_modulos/accion_social.md` |
| Nóminas | `docs/plan_modulos/nominas.md` |
| Prevención de riesgos laborales | `docs/plan_modulos/prevencion_riesgos_laborales.md` |
| Igualdad | `docs/plan_modulos/igualdad.md` |
| Incompatibilidades | `docs/plan_modulos/incompatibilidades.md` |
| Régimen disciplinario | `docs/plan_modulos/regimen_disciplinario.md` |
| Relaciones sindicales | `docs/plan_modulos/relaciones_sindicales.md` |
| Canal interno de información | `docs/plan_modulos/canal_interno_informacion.md` |

`docs/plan_modulos/comun.md` no es un módulo: es una regla común de funciones
PostgreSQL (va en `docs/producto/ESQUEMA_BACKEND.md`).

## 9. Requisitos funcionales

Los códigos entre paréntesis (NUC-001, BOL-005…) identifican cada capacidad del
catálogo funcional y se usan en planes, tareas y PR. Cada tabla dice el resultado
que se exige; el estado de cada uno está en su plan de módulo.

### 9.1 Núcleo común

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| NUC-001 | Persona | Identificador opaco común, coincidencia asistida, procedencia por dato, fusiones y separaciones trazadas; el DNI no es clave técnica ni campo de búsqueda general |
| NUC-002 | Identidades y cuentas | Varias identidades por persona, proveedores sustituibles, un perfil por sesión sin sumar permisos |
| NUC-003 | Representación | Poder, tutoría o representación con alcance, vigencia, evidencia y revocación; cada actuación identifica a representado y representante |
| NUC-004 | Organismos, unidades y centros | Organización histórica, jerarquías versionadas y relaciones de dependencia con fechas |
| NUC-005 | Relaciones jurídicas | Nombramiento, contrato, régimen, causa, situación y periodo como hechos separados, con las dos fechas |
| NUC-006 | Registro de fuentes de autoridad | Fuente, artículo, documento, huella, ámbito, vigencia, órgano, firmas, parámetros y casos de prueba |
| NUC-007 | Catálogos gobernados | Código estable, versión, traducción, procedencia, vigencia y sustitución; nunca listas legales en el código |
| NUC-008 | Formularios gobernados | Esquemas versionados, validación en servidor, accesibilidad, borrador y migración explícita |
| NUC-009 | Flujo de expedientes | Estados, tareas, plazos, responsables, suplencias, bloqueos, doble control, rectificaciones y cierre |
| NUC-010 | Documento | Original, representaciones, metadatos, huellas, procedencia y relaciones sin duplicar contenido |
| NUC-011 | Almacenamiento seguro | Cifrado, cuarentena, antivirus, versiones, retención, copia y restauración comprobadas |
| NUC-012 | Firma, sello y verificación | Firma de persona y sello distintos, cofirmas, tiempo fiable, código seguro de verificación (CSV) y QR de cotejo, con GrxFirma |
| NUC-013 | Generación de documentos | PDF, PDF/A, Word y otros desde un modelo común; se distingue el original jurídico de la copia accesible |
| NUC-014 | Registro administrativo | Conexión con el registro oficial; VEC no simula un registro con una tabla propia |
| NUC-015 | Notificaciones y avisos | Notificación administrativa trazable y avisos separados (correo; otros canales cuando se aprueben) sin datos sensibles |
| NUC-016 | Autorización | Permisos por perfil y por unidad, expediente, finalidad, estado, tiempo y delegación; se deniega lo no concedido |
| NUC-017 | Segregación de funciones | Incompatibilidades y doble intervención en configuración, baremación, firma, pago, seguridad y expedientes reservados |
| NUC-018 | Auditoría con valor de prueba | Actor, sujeto, finalidad, acción, expediente, regla, antes y después, tiempo fiable y resultado, en registro que no se puede manipular |
| NUC-019 | Registros técnicos | Separados de la auditoría funcional, minimizados y con su vista para Sistemas |
| NUC-020 | Calendarios | Día natural, hábil, festivo, apertura y jornada separados, con fuentes y versiones |
| NUC-021 | Comunicaciones | Preferencias, plantillas, canales, entregas y bajas, sin convertir un aviso en acto |
| NUC-022 | Búsqueda | Por permiso, sin permitir recorrer datos ajenos, con filtros auditados y resultados mínimos |
| NUC-023 | Exportación e informes | Vistas aprobadas, finalidad, límites, cifrado, caducidad y auditoría; nunca SQL libre |
| NUC-024 | Ayuda y accesibilidad | Una fuente de ayuda versionada para web, manual, audio y ayuda en pantalla tras el botón «?»; el asistente solo usa ese contenido |
| NUC-025 | Otros canales | API, línea de órdenes y escritorio sobre los mismos casos de uso y permisos |
| NUC-026 | Eventos y bandeja de salida | Avisos entre módulos versionados, sin duplicados, con reintento durable |
| NUC-027 | Importación | Preparación, simulación, incidencias, aprobación y reversión; nunca escritura directa en tablas de otro sistema |
| NUC-028 | Conservación y archivo | Serie, expediente, índice, transferencia, bloqueo, valoración y expurgo autorizados |

### 9.2 Contratación temporal

Para la presentación vale el caso de aceptación de `FLUJO_APP.md`, apartado 4.0,
que empieza con la inscripción del aspirante y termina con su cese.

El circuito lo describió RRHH (`docs/estudio_requisitos/decisiones_rrhh_2026-10-02.md`
y la especificación de julio, hoy en `docs/producto/FLUJO_APP.md`, apartado 4.3):

1. Petición del centro, firmada por el técnico y ratificada por el Diputado o la
   Diputada de la Delegación. Elige la necesidad publicada en la RPT.
2. Autorización de la Dirección de RRHH y alta del expediente según la circular
   de peticiones de personal de mayo de 2026, con número de personas y puesto.
3. Comprobación de crédito. Sin crédito no se tramita nada. La retención la hace
   cada Delegación en SICAL; si no llega, se transmite el estado de las partidas
   y el coste aproximado.
4. Ofrecimiento desde la bolsa constituida de la categoría. Lo hace Bolsa; CT no
   tiene llamamiento propio. La correspondencia entre categoría y bolsa está en
   un catálogo editable. Si no hay nadie disponible, se valora la causa y se abre
   una nueva convocatoria de bolsa; si es muy urgente, oferta genérica al SAE.
5. Adjudicación a quien corresponda entre quienes aceptaron.
6. Informe de la Jefatura de Servicio para Intervención.
7. Fiscalización: conforme o reparo. El reparo se subsana con un informe nuevo y
   se fiscaliza otra vez, con un cómputo de plazo nuevo (RD 424/2017, arts. 10.2
   y 12.4).
8. Resolución firmada por la Diputada con el visto bueno de la Dirección o de la
   Jefatura de RRHH.
9. Grabación en GINPIX, a mano mientras no haya integración.
10. Firma de la toma de posesión o del contrato por la persona; recibe el documento
    de incorporación y lo entrega en su centro, que confirma la incorporación;
    paso a Personal.
11. Seguimiento, cese y vuelta de la persona a su bolsa.

Si la persona aún no ha empezado, una devolución o sustitución repite los trámites
(salvo Intervención si el contrato no cambia); si ya ha empezado, se reinicia todo.

Requisitos que no se negocian: las modalidades y los seis documentos que exige la
fuente (contratos, informes, resoluciones, modificaciones, comunicados de cese y
documento de incorporación); el expediente guarda la versión y la huella de las
reglas con que se tramitó; cada paso deja recibo, historia y auditoría; un
reintento o un reinicio no duplica nada. El número de expediente lo da MOAD y
VEC lo guarda. Los plazos de CT se editan desde «Reglas vigentes». No se igualan
pantallas con fases ni se cierra la ambigüedad de las ocho fases sin validación
de RRHH.

### 9.3 Bolsa de trabajo

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| BOL-001 | Constitución | Bolsa ligada a su proceso u origen, categoría, ámbito, versión normativa, vigencia, lista y criterios de orden |
| BOL-002 | Preferencias | Centro, territorio, duración, jornada y clase de oferta, según lo que permita la configuración |
| BOL-003 | Disponibilidad | Estado voluntario o reglado, causa, periodo, documentos y decisión |
| BOL-004 | Restricción tras el cese | Política versionada sobre el cese oficial, una sola vez por persona y no copiada en cada bolsa |
| BOL-005 | Llamamiento | Propuesta reproducible, orden, exclusiones, intentos, canales, plazos, respuestas y renuncias |
| BOL-006 | Posición propia | La persona ve su orden, estado, última actuación y siguiente paso, sin datos de terceros |
| BOL-007 | Publicaciones | Listas mínimas, aprobadas y versionadas; mejor la consulta individual cuando basta |
| BOL-008 | Incorporación | Propuesta, documentación, aptitud, nombramiento o contrato, alta en Personal y disponibilidad resultante |
| BOL-009 | Indicadores | Cobertura por categoría, agotamiento, tiempos, intentos y renuncias, con definición y fuente |

Reglas decididas por RRHH y por el [Reglamento de bolsas](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Reglamento_seleccion_de_personal_y_bolsas_bop_16_01_2026.pdf)
(BOP de 16/01/2026, modificado el 14/05/2026):

- La oferta publica un extracto del puesto (descripción, duración estimada y
  requisitos). La política de cada bolsa fija si el plazo empieza con la
  publicación efectiva en VEC (art. 8.1.a y b del Reglamento) o con el correo
  declarado por RRHH (decisión del 02/10, duda 153 abierta). La configuración
  inicial propone publicación; una política ya publicada conserva su valor.
  La persona manifiesta interés dentro del plazo; no contestar no tiene
  consecuencia. Un aviso pendiente de envío no abre el plazo.
- Se adjudica por posición entre quienes aceptaron en plazo y estén disponibles
  (art. 9). Si nadie acepta, llamamiento directo: acepta, renuncia justificada o
  exclusión. Con justificante válido vuelve a estar disponible.
- El turno de discapacidad va en bolsa separada (art. 4.2).
- Estados: «en revisión» (se espera un justificante), renuncia justificada (la
  persona aporta el documento cuando termina la causa y RRHH lo valida); no existe
  la suspensión temporal.
- Tras una relación temporal, no disponibilidad de cinco meses, o nueve si fue
  acumulación de tareas (art. 9), como política configurable con la norma citada.
  El aviso de tres años seguidos y los máximos por acumulación vienen de la ley.
- Los integrantes introducen y cambian sus datos de contacto con su certificado;
  RRHH no los valida. Si un correo no llega, cuenta como no contestado. Se guarda
  el historial de todos los ofrecimientos y de a quién llegaron.
- Carga inicial desde los Excel que exporte RRHH de CONVOCA, con vista previa y
  confirmación.
- Contacto enmascarado en listados y completo en la ficha, con constancia de la
  consulta. Quien no tiene permiso para el contacto completo lo ve enmascarado,
  con un aviso claro. La persona ve el suyo completo en «Mi Bolsa». Las cifras
  agrupadas llevan con un clic a su listado filtrado.
- En «Mi Bolsa» cada persona ve su propio nombre, con su sesión; nadie más ve
  nombres ajenos (09/10). Todo el personal de RRHH ve todas las bolsas, y cada
  consulta queda en la auditoría (12/10).
- La carga desde CONVOCA lee dos formatos `.xls`: el resumen (DNI enmascarado,
  apellidos, nombre, turno, experiencia, formación y total) y el detallado «con
  claves» (una fila por mérito, con grupo, orden, puntos de autobaremación frente
  a los del tribunal y motivo). El DNI llega enmascarado, así que no basta para
  ningún acto con efectos hasta confirmar la identidad de la persona.
- La primera versión de la inscripción en línea es para aspirantes externos
  (09/10). El canal propio del empleado, con la misma persona y la experiencia
  acreditada desde Personal, está en `docs/estudio_requisitos/pendientes_v2.md`.

### 9.4 Inscripción, requisitos, méritos y baremación

Es la parte de CONVOCA que se integra ya, porque es la base de las bolsas.

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| SEL-002 | Convocatoria gobernada | Bases, anexos, plazas, requisitos, fases, calendario, tasa, baremo, tribunal y circuito como versión firmada que no cambia |
| SEL-003 | Publicidad | Página pública accesible con versiones, plazos, documentos y fuente oficial; retirada sin borrar historia |
| SEL-004 | Solicitud | Borrador recuperable, representación, requisitos, anexos, firma, registro, justificante y modificación dentro de plazo |
| SEL-005 | Tasas | Cálculo, exención, pasarela intercambiable, conciliación y devolución |
| SEL-006 | Admisión | Comprobación de requisitos, datos de oficio, exclusión motivada, subsanación y listas provisional y definitiva |
| SEL-009 | Alegaciones y recursos | Presentación, informe, resolución y efecto sin borrar el acto anterior |
| RUM-001 | Registro de méritos | Experiencia, títulos, cursos y otras evidencias reutilizables, con procedencia, vigencia y estado |
| RUM-002 | Datos de oficio | Consulta autorizada a Personal y fuentes externas, con alternativa humana si la respuesta no es clara |
| RUM-003 | Validez del documento | Autenticidad separada del reconocimiento del mérito y de si aplica en una convocatoria |
| BAR-001 | Editor de baremo | Reglas tipadas para periodos, jornada, experiencia, títulos, cursos, topes, solapes, equivalencias y desempates |
| BAR-002 | Simulación | Casos de prueba, muestras anonimizadas, comparación entre versiones y bloqueo ante resultados no determinables |
| BAR-003 | Autobaremo | Cálculo orientativo explicable; la persona no escribe puntos ni se autovalida |
| BAR-004 | Revisión técnica | Decisión por mérito, motivo, evidencia, revisor, doble control según riesgo y firma |
| BAR-005 | Rectificación | Revocación, rehabilitación o corrección con un acto nuevo firmado y recálculo |
| BAR-006 | Resultado | Puntuación declarada, calculada, validada y definitiva, con desglose, topes, orden y huella reproducible |

Reglas de requisitos de acceso (de `ESPECIFICACIONES_AGENTES.md`, E02):

- Bolsa admite candidaturas de externos y de empleados; no exige ser empleado.
  Una persona empleada se inscribe con su misma persona, sin duplicarse, y su
  experiencia en la casa se toma de Personal.
- Cada dato que aporta la persona se registra una vez, con fuente, evidencia,
  vigencia y estado: declarado, pendiente, acreditado o rechazado. Comprobar un
  requisito y puntuar un mérito son decisiones distintas: un título puede habilitar
  y puntuar, solo habilitar o no aplicar. No se deducen requisitos de los puntos ni
  se convierte una valoración antigua en un hecho universal.
- El área personal evalúa las convocatorias con plazo abierto frente a los
  requisitos estructurados y versionados de sus bases. Por cada requisito da
  `cumple`, `no_cumple` o `pendiente`, con motivo y procedencia. Lo que solo
  está en texto, una equivalencia no aprobada o un dato que falta queda pendiente:
  no se interpreta con IA ni se da por cumplido. El resultado global distingue si
  puede solicitar ya, si puede preparar la solicitud pero debe completar o
  acreditar datos, si hay un incumplimiento comprobado (que se explica) o si el
  plazo está cerrado o ya existe solicitud.
- Cada requisito temporal fija cuándo debe cumplirse según las bases (solicitud,
  fin de plazo, inicio del proceso, prueba, incorporación u otra fecha). En bolsas
  de incorporación inmediata el título debe estar aportado en ese momento; un
  título futuro no habilita.
- Solo en ofertas de empleo público y procesos selectivos cuyas bases lo permitan,
  se admite el «cumplimiento previsto»: fecha estimada y evidencia admitida, con
  revalidación al llegar el hito. Una previsión nunca es una acreditación ni se
  reutiliza en otra convocatoria. En la edición de cada requisito de titulación
  de una oferta de empleo público hay un control «Permitir inscripción con
  titulación pendiente», desactivado por defecto; al activarlo son obligatorios la
  fecha límite y la evidencia. En bolsas de incorporación inmediata está
  deshabilitado. Forma parte de la versión de las bases: cambiarlo después exige
  versión nueva o rectificación.
- La evaluación no sustituye la admisión. Un «pendiente» no impide presentar la
  solicitud si las bases permiten aportar el dato durante el trámite. El aviso
  abre el detalle público de la convocatoria y, desde ahí, la solicitud
  precompletada, previa confirmación de la persona.

Más principios de baremación: el autobaremo y el baremo de RRHH son resultados
distintos; aceptar o rechazar un mérito es una decisión firmada que no se
sobrescribe (una revisión crea una rectificación y conserva todas las versiones);
las bases de cada convocatoria fijan la versión exacta del baremo; jornada,
solapes, conversiones y redondeos no tienen valores ocultos por defecto; los
servicios internos se obtienen de oficio de Personal y una falta de respuesta
nunca se convierte en cero puntos. El baremador es un motor común: Bolsa primero,
después Selectivos y Provisión con sus propias reglas.

### 9.5 Firmas

Firma con GrxFirma por el protocolo `afirma://` desde el navegador y verificación
con su validador como servicio aparte. Dos o más firmantes en el orden del
circuito, el mismo PDF verificado, justificante y auditoría de cada descarga. Las
delegaciones y suplencias las comunica Secretaría General y no siempre firma la
misma persona (dudas 122, 128 y 143). VEC nunca inventa una firma legal.

### 9.6 Identidad, perfiles y administración

- Acceso con DNIe o certificado digital; Kerberos corporativo para el personal
  interno más adelante. Un certificado válido no da permisos por sí solo.
- Perfiles fijos asignados por el administrador con historia y motivo; quien
  aprueba o liquida lo hace por su perfil, no por un acto suelto cada vez. Un
  perfil fijo no crea por sí solo la competencia administrativa.
- No hay administrador universal, comodines ni suma de perfiles. Nadie resuelve
  su propia solicitud ni hace dos pasos que deban hacer personas distintas.
- Administración en superficie propia: reparto de perfiles, roles y permisos,
  interruptor de módulos auditado, catálogos y copias de seguridad.
- En producción, Administración solo desde la intranet, con DNIe o certificado y
  autorización expresa.
- Hay dos perfiles de administrador (02/10). **Administrador de Sistemas**:
  instala, actualiza, hace copias y propone restauraciones; no ve datos
  personales ni reparte perfiles. **Administrador de Aplicación**: perfiles y
  cargos, catálogos, configuración funcional, módulos y auditoría. Ninguno ve
  expedientes ni bolsas. Puede haber un perfil auditor de solo lectura.
- Nadie se asigna un perfil a sí mismo. El primer administrador de cada tipo se
  crea en la instalación; los siguientes necesitan dos personas, y nunca puede
  quedar ninguno.
- Gestión completa de perfiles (03/10): perfiles versionados sobre el catálogo
  cerrado de acciones de cada módulo (una acción en uso se deshabilita, no se
  borra); asignaciones con ámbito, vigencia y, si hace falta, el acto o la
  delegación que las respalda (suplencias); consulta de quién tiene qué;
  retirada de un perfil o bloqueo de una persona con efecto inmediato. La
  pantalla de Usuarios (lista, ficha con perfiles e historial, asignar o retirar
  varios) tiene una orden de consola equivalente y usa la autoridad central
  existente, sin crear otra.
- Los cargos de los circuitos (Jefatura de Servicio, Dirección de RRHH,
  Diputado o Diputada delegada, Intervención…) son perfiles a los que se asigna
  una persona con vigencia. Circuitos, plantillas y textos nombran el perfil,
  nunca a la persona.
- El interruptor de cada módulo se maneja desde Administración, persiste, queda
  auditado y deniega por defecto; el catálogo del portal y las rutas lo
  respetan. Los selectores de entorno siguen como capacidad técnica.
- Copias de base, configuración, material y ficheros, programadas y
  comprobadas restaurándolas en una base desechable, con retención, cifrado y
  destino configurables. Restaurar exige dos personas (propone Sistemas, aprueba
  Aplicación) y antes comprueba las versiones de PostgreSQL, del programa y del
  esquema; si algo no casa, no se ejecuta y se explica por qué.

### 9.7 Auditoría

Se registra todo lo que hace cada usuario, también lo que consulta o descarga:
quién (con su perfil), qué, sobre qué, para qué, cuándo, con qué resultado y por
qué canal; también los intentos denegados. Solo se añade, en la misma operación
que el efecto, encadenado con huellas y con sellado de tiempo. Se puede exportar
firmado para un juzgado y la exportación también queda auditada. Ni el
administrador funcional puede alterarla. No guarda secretos ni documentos
personales completos. La conservación por serie documental espera al Archivo y
al DPD (duda 84). Consultas y exportaciones se registran antes de entregar el
resultado (ANA-005).

### 9.8 Personal y Organización/RPT

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| PER-001 | Registro de Personal | Relaciones, actos, antigüedad, trienios, situaciones, grados, ocupaciones y procedencia |
| PER-002 | Alta y toma de posesión | Comprobaciones, acto firmado, registro, identidad corporativa, ocupación, nómina y comunicaciones conciliadas |
| PER-003 | Contrato, nombramiento y cese | Causa, periodo, documentos, competencia, efectos, reserva y avisos a Bolsa y Nómina |
| PER-004 | Situaciones | Catálogo por régimen y periodo, con sus efectos en ocupación, reserva, servicios, jornada y nómina |
| PER-005 | Rectificación | Dato oficial de solo lectura; propuesta, evidencia, revisión, decisión y sustitución histórica |
| ORG-001 | Estructura organizativa | Organismos, áreas, servicios, centros y unidades con su historia, delegaciones y suplencias |
| RPT-001 | Categorías | Cuerpo, escala, subescala, clase, especialidad y categoría laboral separados, con equivalencias |
| RPT-002 | Plantilla y plazas | Dotaciones por ejercicio, creación, amortización, reserva y vínculo con la oferta de empleo |
| RPT-003 | Puestos | Puesto tipo e individual, unidad, requisitos, funciones, provisión, nivel, complementos y vigencia |
| RPT-004 | Versiones de la RPT | Importación, acuerdo, BOP, diferencias, aprobación y consulta histórica sin sobrescribir |
| RPT-005 | Ocupación y reserva | Persona, relación, plaza y puesto con fechas, acto y reserva, sin deducirla de la nómina |
| RPT-006 | Vacantes | Vistas separadas de dotación vacante, puesto sin ocupante y necesidad cubrible |
| PLA-001 | Planificación | Necesidades, escenarios, coste, jubilaciones, vacantes, bolsas y crédito, sin decisiones automáticas |
| PLA-002 | Capítulo I | Escenarios conciliados con presupuesto y nómina, separados del acto aprobado |
| SEL-001 | Plan y oferta de empleo | Relación entre necesidad, crédito, plaza, oferta, convocatoria y su ejecución |

Contratación tramita el expediente temporal y aporta el acto admitido; Personal
registra la relación, la incorporación y el cese. RRHH añade o deshabilita
categorías de la RPT; las que están en uso no se borran. Un nombre de cargo no da
competencia: la da la relación organizativa acreditada.

### 9.9 Área del empleado

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| EMP-001 | Carpeta personal | Datos oficiales propios, actos, puesto, servicios, documentos, solicitudes y su procedencia |
| EMP-002 | Bandeja y solicitudes | Formularios, borradores, estado, plazo, justificante, subsanación, decisión y notificación |
| EMP-003 | Espacio del responsable | Cobertura y tareas de su unidad con delegación temporal; nunca el expediente completo |
| EMP-004 | Trabajo en equipo | Espacios por equipo o expediente con tareas, comentarios y adjuntos, con ámbito y auditoría |
| EMP-005 | Agenda compartida | Reuniones e hitos sincronizables con calendarios externos mediante conectores sustituibles |

La ficha integral del empleado reúne, con una cabecera común y pestañas (Empleo,
Tiempo, Formación, Economía, Trámites y Documentos), lo que cada módulo dueño
publica de esa persona: identificación y contacto; cuentas y representación;
relación de servicio y situaciones; unidad, centro, plaza y puesto con su
historia; servicios prestados (declarados, comprobados y reconocidos por
separado); antigüedad y trienios; grado y carrera; jornada, fichajes y saldos;
permisos y teletrabajo; títulos y méritos; formación; nóminas, bases de
cotización y certificados fiscales; dietas y pagos; ayudas; prevención (solo lo
que el titular puede ver); incompatibilidades; documentos y certificados;
solicitudes y avisos; y la trazabilidad de correcciones.

Reglas de la ficha: cada bloque se pide al módulo dueño y se carga por separado,
paginado y con periodo acotado; no se manda toda la ficha al navegador para
ocultar campos. Cada dato muestra fuente, fecha de efectos, fecha de actualización
y si es declarado, provisional o confirmado. Un bloque caído no tumba los demás y
se muestra como no disponible, no como cero. Un dato oficial se corrige en su
módulo con una solicitud. Las jefaturas y RRHH solo ven las tareas y campos de su
competencia.

### 9.10 Cronos

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| CRO-001 | Perfil de jornada | Calendario, colectivo, centro, ciclo, horario, flexibilidad y reducciones desde reglas aprobadas |
| CRO-002 | Cuadrantes y turnos | Planificación, publicación, cambios, cobertura y descanso, con reglas por centro |
| CRO-003 | Fichajes | Fichaje original que no se modifica, origen, corrección aparte y auditoría |
| CRO-004 | Incidencias | Detección explicable, alegación, aprobación y cierre; una anomalía no es una sanción automática |
| CRO-005 | Permisos y vacaciones | Catálogo por régimen, hecho causante, saldo, justificación mínima, resolución y recurso |
| CRO-006 | Teletrabajo | Solicitud, días, objetivos, medios, revisión y revocación |
| CRO-007 | Horas y compensaciones | Autorización, tiempo real, saldo, disfrute o pago, con paso trazable a Nómina |
| CRO-008 | Geolocalización de vehículos | Finalidad, horario, ruta, acceso, retención y evaluación de impacto, solo en red interna |

Fuentes: TREBEP (arts. 47 a 51), Convenio colectivo de 2006 con sus
modificaciones, Reglamento de tiempo de trabajo de 2010 (su art. 17.4 lo declaró
nulo el Pleno del 2/09/2026) y Resolución de jornada y permisos de 2015. Calendarios
da festivos y apertura; Personal da relación, colectivo y adscripción; Cronos
guarda programación, fichajes, correcciones, permisos y saldos. El saldo es un
libro de movimientos explicables, no un número que se sobrescribe. Solicitud,
decisión del responsable, resolución de RRHH y aplicación al saldo son hechos
distintos. Nadie resuelve su propia solicitud ni hace a la vez el paso de
jefatura y el de RRHH. Los permisos actuales del catálogo son provisionales hasta
que RRHH los confirme (dudas 19 a 21, 38, 41, 42, 47, 48 y 100 a 103).

### 9.11 Dietas

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| DIE-001 | Comisión de servicio | Orden, persona, objeto, ruta, fechas, medio, anticipo, autorizadores y documentos |
| DIE-002 | Itinerario y kilometraje | Ruta de la cartografía interna con versión, excepciones motivadas y cifras exactas; el GPS no acredita el derecho |
| DIE-003 | Liquidación | Tarifas por vigencia, justificantes, cálculo, fiscalización, pago y rectificación |

El recorrido que se debe terminar: la persona abre una comisión ligada a su
relación y unidad; la preparación calcula manutención por tramos, alojamiento
cuando RRHH fije su regla y kilometraje por la ruta interna, mostrando vigencia,
procedencia y versión de cada tarifa (un dato que falta deja la propuesta
pendiente, con motivo); adjunta los justificantes con antivirus y custodia; el
envío pasa por revisión, autorización, liquidación de RRHH y fiscalización según
el circuito aprobado; la liquidación usa la versión exacta de comisión,
justificantes y tarifas, y genera su PDF; la exportación al sistema económico
tiene su permiso y su acuse; solo la confirmación de Tesorería permite mostrar
«pagado». Fuentes: Real Decreto 462/2002, Orden HFP/793/2023, bases de ejecución
del presupuesto de 2026 y art. 47 del Convenio. RRHH debe fijar la regla por
colectivo y fecha (dudas 22 a 26, 40, 44, 46, 49 a 51 y 104 a 108).

### 9.12 Provisión, carrera, formación y certificados

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| PRO-001 | Catálogo de procedimientos | Concurso, libre designación, adscripción provisional, comisión y movilidad, con su fuente |
| PRO-002 | Concurso y baremo | Convocatoria, puestos, requisitos, méritos, preferencias, valoración, alegaciones y adjudicación reproducibles |
| PRO-003 | Movilidad por salud | Conclusión funcional de Prevención, puestos compatibles, prioridad y resolución, sin datos clínicos |
| CAR-001 | Grado funcionarial | Nivel, grado personal, periodos, vía de reconocimiento, acto e inscripción separados |
| CAR-002 | Progresión laboral | Según convenio, convocatoria, curso o prueba y resolución; la antigüedad no da progresión automática |
| CAR-003 | Promoción interna | Elegibilidad, convocatoria, pruebas y toma de posesión, sin confundirla con carrera |
| CAR-004 | Desempeño | Sistema negociado y transparente, sin vigilancia algorítmica oculta |
| FOR-001 | Plan de formación | Necesidades, presupuesto, acciones, ediciones, plazas y publicación |
| FOR-002 | Solicitud y selección | Solicitud, autorización, criterios, sustituciones, lista y notificación |
| FOR-003 | Ejecución y certificado | Asistencia, aprovechamiento, evaluación, certificado firmado y paso a méritos, sin dar puntos ni grado |
| CER-001 | Certificados | Plantillas versionadas, datos de oficio de Personal, revisión, firma, registro, CSV, entrega y conservación |

En Provisión, la libre designación conserva su circuito de idoneidad y no tiene
orden automático. Provisión usa el baremador común con sus propias reglas y lee
de Personal, RPT y el registro de méritos por contrato; no reconstruye la RPT ni
declara vacantes por falta de un dato. Formación no reconoce grados: el curso de
consolidación de grado es fuente para Carrera.

### 9.13 Selectivos

Además de lo de 9.4, cuando se retome: SEL-007 (tribunal: composición,
recusación, sustitución, acceso aislado por proceso y fase, sesiones, actas y
firmas) y SEL-008 (pruebas: sedes, llamamientos, anonimato cuando proceda,
calificaciones, mínimos, incidencias y publicación). La composición, asistencia,
abstención, sustitución, quórum y orden de firma del tribunal se congelan en el
acta. Un grupo genérico no equivale a nombramiento.

### 9.14 Nóminas, acción social y materias reservadas

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| NOM-001 | Conceptos y tablas | Definiciones, fórmulas tipadas, fuentes, colectivos, vigencia y casos de prueba |
| NOM-002 | Devengo | Relación, puesto, jornada, incidencias, atrasos y prorrateos con procedencia |
| NOM-003 | Seguridad Social | Altas, bajas, bases, tipos, liquidaciones y conciliación |
| NOM-004 | IRPF | Retenciones, regularizaciones y certificados, en compartimento fiscal |
| NOM-005 | Cálculo y cierre | Prenómina, diferencias, doble revisión, cierre que no cambia y rectificación formal |
| NOM-006 | Fiscalización y contabilidad | Lotes, reparos, aprobación, asientos, pagos y conciliación con Intervención y Tesorería |
| NOM-007 | Recibo y certificados | Acceso propio, firma, CSV, conservación y datos mínimos |
| NOM-008 | Migración | Datos conciliados y varios ciclos en paralelo antes de retirar el sistema actual |
| ASO-001 | Convocatoria de ayuda | Modalidad, colectivo, requisitos, presupuesto, plazo, comisión y cuantía versionados |
| ASO-002 | Solicitud y unidad familiar | Datos familiares mínimos, documentación, subsanación y compartimento propio |
| ASO-003 | Presupuesto y resolución | Reserva, prelación, propuesta, resolución, pago, reintegro y cierre |
| PRL-001 | Riesgos y medidas | Evaluación por puesto y centro, medidas, responsables y revisión |
| PRL-002 | Vigilancia de la salud | Datos clínicos solo para su personal sanitario; hacia Personal, solo la aptitud |
| PRL-003 | Accidentes | Parte, investigación, medidas y comunicaciones, con acceso restringido |
| IGU-001 | Plan de igualdad | Diagnóstico, negociación, medidas, responsables, calendario e indicadores |
| IGU-002 | Igualdad en selección | Informe, composición equilibrada, lenguaje inclusivo y adaptaciones |
| RES-001 | Protocolos reservados | Denuncia, protección, instrucción y resolución en expediente aislado |
| RES-002 | Sistema interno de información | Enlace al canal institucional existente; no se crea un segundo canal |
| RLA-001 | Negociación | Mesas, actas, acuerdos y su paso a reglas |
| RLA-002 | Crédito sindical | Representante, saldo, solicitud y justificación; afiliación separada |
| RLA-003 | Incompatibilidades | Declaración, actividad, informes, resolución y publicación legal |
| RLA-004 | Disciplina | Hechos, tipificación, prescripción, instructor, audiencia, resolución y recurso, segregados |

Reglas que ya se saben: Nóminas empieza por la consulta de recibos y no fija
fórmulas ni tablas sin acto aprobado (GINPIX no se presume su fuente; duda 129).
Acción social empieza por una modalidad con convocatoria vigente verificada
(duda 130). Prevención no ofrece a RRHH ninguna forma de abrir diagnósticos; la
aptitud se trata también como dato de salud (duda 131). Igualdad recomienda; no
cambia admisiones, baremos ni salarios ni declara discriminación automáticamente
(duda 132). Incompatibilidades distingue segunda actividad pública, privada y
exceptuada; no hay renovación anual universal (duda 133). En disciplina, los doce
meses de la Ley andaluza 5/2023 y los seis del convenio no son opciones
intercambiables; prescripción de falta, de sanción y caducidad son cómputos
distintos (duda 134). Las cifras sindicales de 2020 no se heredan (duda 135). El
canal interno respeta la Ley 2/2023 sin convertir sus garantías en opciones de
RRHH (duda 136).

### 9.15 Analítica, transparencia y copias

| Código | Capacidad | Resultado exigido |
| --- | --- | --- |
| ANA-001 | Indicadores gobernados | Definición, fórmula, fuente, periodo, responsable, calidad y versión |
| ANA-002 | Cuadros de mando | Datos agregados o seudonimizados y filtrados por ámbito |
| ANA-003 | Transparencia | Publicación aprobada, mínima, accesible y versionada |
| ANA-004 | Informes obligatorios | Plantillas para ISPA y otras obligaciones, con revisión y firma |
| ANA-005 | Acceso y exportación | Registro nominal de consulta y descarga antes de entregar el resultado |

Copias de seguridad: desde Administración se ve qué copias están completas y
cuáles se han restaurado bien; se pide una copia y se fija su calendario y
retención. Cada copia reúne base de datos, configuración, documentos y binarios,
con manifiesto, cifrado y huellas. Antes de copiar y de restaurar se comprueban
versiones; para restaurar, otra persona administradora aprueba, y antes se hace
una copia verificada del estado actual.

## 10. Flujos que cruzan módulos

### De aspirante a empleado

```text
aspirante → solicitud registrada → resultado y bolsa → llamamiento
→ propuesta de nombramiento o contrato → comprobaciones → acto firmado
→ relación nueva en Personal → ocupación
→ identidad corporativa, Cronos y Nómina mediante avisos conciliados
```

La persona no se duplica; nace una relación nueva. Bolsa no escribe en Personal,
Active Directory, Cronos ni Nómina. El paso a plaza fija por un proceso selectivo
usa el mismo mecanismo, con sus diferencias, y al terminar una relación temporal
la persona vuelve a su bolsa.

### Servicio prestado reutilizado como mérito

```text
nombramiento y cese en Personal → periodo oficial de servicios
→ vista mínima al registro de méritos → instantánea de la convocatoria
→ aplicación de sus bases → revisión y resultado
```

La nómina puede conciliar, pero no es la única fuente. Una falta de respuesta no
se convierte en cero meses.

### Permiso con efecto económico

```text
solicitud → responsable → RRHH cuando proceda → resolución
→ incidencia aprobada de jornada → concepto para Nómina → conciliación → recibo y auditoría
```

Solo se pasa el efecto necesario, nunca el diagnóstico o el documento que lo
justifica.

## 11. Requisitos no funcionales de producto

Son requisitos que nota quien usa VEC. Cómo se consiguen está en
`docs/producto/TRD.md`.

### Rapidez

- Toda lectura responde por debajo de 300 ms en el percentil 95 con volumen real.
  La base común (sesión, módulos y preferencias) baja de 100 ms.
- La pantalla útil aparece en menos de un segundo. El portal pinta primero y
  carga cada módulo después, sin que uno lento bloquee a los demás.
- Sin consultas repetidas por fila. Se mide con miles de registros y, antes de
  cada hito, se prueba con cientos y miles de personas a la vez.
- Nunca se gana velocidad quitando permisos ni auditoría.

### Auditoría con valor de prueba

Ver 9.7. VEC es de una Administración pública: tiene que poder demostrar ante un
juzgado qué hizo cada persona y qué datos vio.

### Seguridad y acceso

- Lo que no está permitido expresamente se deniega.
- Sin cookies ni almacenamiento del navegador como sesión.
- La identidad viene del certificado comprobado por el servidor, nunca de datos
  que envía el navegador.
- Los actos con efecto se autorizan en la misma operación que los guarda.
- El diseño apunta a la categoría ALTA del ENS, sin declarar conformidad hasta
  tener categorización, análisis de riesgos, implantación y auditoría.

### Accesibilidad, lenguaje y presentación

- El ordenador de mesa es la vista prioritaria; el móvil (390 px) sigue siendo
  usable. Tablas, formularios y espacio de escritorio se optimizan para PC.
- Teclado, foco visible, contraste, zoom al 200 %, lector de pantalla y estados de
  vacío, carga, error y denegación. Documentos descargables accesibles.
- Lenguaje claro de la Administración: frases cortas, voz activa y el vocabulario
  del Departamento, sin siglas internas ni identificadores técnicos como dato
  principal.
- La ayuda está solo tras un botón «?»; nunca texto de ayuda fijo en pantalla.
- Una persona de RRHH sin formación previa debe poder usarlo, porque la plantilla
  rota. Aspecto limpio y profesional con el tema común, sin otra paleta.
- Las reglas de pantalla están en `docs/producto/DISENO_UI_UX.md` y en las skills
  `usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e `impeccable`.

### Idiomas

Castellano e inglés. Ningún texto visible está en el código: todos están en
catálogos por idioma, y los idiomas disponibles salen de
`web/static/textos/idiomas.json`. Añadir un idioma es copiar una carpeta y
traducirla. Fechas, números, moneda y plurales se formatean según el idioma.

### Protección de datos

- Solo datos sintéticos hasta las autorizaciones formales.
- Mínimos datos en listados, auditoría y avisos; referencias opacas.
- Contacto de candidatos: enmascarado en listados y completo en la ficha, con
  constancia de quién lo consulta.
- Los datos de trabajadores y de aspirantes no se mezclan. Una persona puede
  pasar de un lado a otro o estar en los dos, mediante traspasos gobernados con
  solo los datos necesarios y su base jurídica. Hay un módulo Aspirantes separado
  de Personal (`internal/modules/aspirantes/`). El estudio completo está en
  `docs/estudio_requisitos/datos_personales_inscripciones_rgpd_2026-09-29.md`; su
  segunda fase (base de datos aparte, categorías especiales) espera al DPD
  (dudas 77 a 79 y 85). Mientras no se decida, no se añaden datos personales
  nuevos en tablas compartidas por internos y externos.
- La IA no interviene en selección, baremación, evaluación ni asignación hasta
  completar las obligaciones del Reglamento europeo de IA. Un asistente público
  solo usa contenido público gobernado.

Clasificación de la información, con su frontera mínima:

| Clase | Ejemplos | Frontera |
| --- | --- | --- |
| Pública aprobada | Convocatorias, bases, RPT publicada, resultados autorizados | Vista específica, con retirada y conservación aprobadas |
| Personal ordinaria | Contacto, puesto, solicitudes propias, formación, servicios | Titular, o competencia y finalidad exactas |
| Económica reservada | Nóminas, cuentas, retenciones, embargos, ayudas | Red interna, cifrado, acceso nominativo y exportación controlada |
| Laboral sensible | Fichajes, ausencias, geolocalización, disciplina | Red interna, ámbito organizativo y registro reforzado |
| Categoría especial | Salud, discapacidad, afiliación, violencia, adaptaciones | Compartimento independiente y lo mínimo imprescindible |
| Confidencial de investigación | Acoso, denuncias, testigos, medidas cautelares | Equipo autorizado, seudónimos y barrera frente al expediente ordinario |
| Seguridad y privilegio | Roles, claves, sesiones, auditoría | Administración separada y registros que no se alteran |

### Correo

El correo del destinatario sale del alta en VEC, donde es obligatorio. El servidor
de correo (dirección interna, puerto, cifrado y cuenta emisora) se configura desde
Administración; la contraseña se escribe pero no se vuelve a mostrar. Un reintento
no envía dos veces. Que el envío funcione no demuestra que el correo se entregó.

### Fiabilidad

Un reintento, una caída o un reinicio no duplican recibos, registros ni correos.
Si una dependencia no responde, VEC no lo interpreta como permiso, firma o entrega:
lo dice y deja reintentar.

## 12. Fuera de alcance

### No se hará

- Inventar firma legal, plazos jurídicos, nombramientos eficaces o incorporaciones.
- Tratar el aviso por correo como notificación administrativa, ni las alegaciones
  registradas en VEC como asiento oficial de entrada.
- Generar el número de expediente (lo da MOAD).
- Usar IA para decidir sobre personas.
- Acceso del aspirante con DNI y clave.
- Copiar reglas, plazos, importes o sanciones de otra Administración.

### No se hace ahora

- Integración automática con GINPIX, SICAL, Firmadoc y el SAE (dudas 74, 125, 126
  y 127). Mientras tanto, el dato se prepara y se graba a mano.
- Correo de la Diputación configurable desde Administración: hoy sale por un buzón
  de pruebas, válido para la presentación (duda 88).
- SMS y Telegram para avisar de llamamientos (duda 152).
- Cálculo de nómina; primero solo consulta de recibos. Personal, Cronos, Dietas
  y Contratación temporal alimentarán Nóminas por sus puertos.
- Buscar en la plantilla a personas que cumplan unas características (por
  ejemplo, un C2 con titulación de C1) para avisarles de una necesidad o
  convocatoria. Es una idea para más adelante: por puertos, nunca con asignación
  automática, con base jurídica (mejor una suscripción voluntaria), datos
  mínimos, registro de las búsquedas y revisión del Reglamento europeo de IA si
  hubiera puntuación.
- Exportación judicial de la auditoría (aparcada en `docs/plan_modulos/auditoria.md`).
- Separar los servidores interno, público y de administración: hasta producción
  todo corre en un solo servidor; ver `docs/producto/TRD.md`.
- Los módulos aparcados del apartado 8.

## 13. Alertas de vigencia

Hallazgos sobre normas que no se resuelven eligiendo la más nueva ni la más
favorable: se registra la cadena, se pide criterio al órgano competente y se
bloquea solo la regla afectada.

| Materia | Hallazgo | Tratamiento |
| --- | --- | --- |
| Nivel, grado y promoción | No hay examen universal cada tres años; promoción, grado y progresión laboral son figuras distintas | Modelos separados; acto e inscripción obligatorios |
| Centros sociales | El reglamento común remite jornada, festivos y vacaciones a normas propias no localizadas | No activar reglas hasta recibir calendarios y acuerdos de centro |
| Vacaciones al cese | El Pleno del 2/09/2026 declaró nulo el art. 17.4 del Reglamento de tiempo de trabajo | Contrastar ese acto antes de publicar reglas de vacaciones |
| Plan de igualdad | El plan de 27/01/2022 declaró cuatro años; no consta sustitución pública | Acreditar plan vigente antes de activar decisiones |
| Protocolo de acoso | El acuerdo de 2020 es anterior a las leyes de 2022 a 2024 | Conservar garantías y pedir actualización |
| Disciplina | El Convenio de 2006 cita seis meses y normas derogadas; la Ley andaluza 5/2023, doce | Bloqueado hasta criterio jurídico por vínculo |
| Acción social | El texto refundido de 2021 cita la Ley 30/1992 | Usar los actos auténticos, la Ley 39/2015 y la convocatoria vigente |
| Crédito sindical | El acuerdo de 2020 liga cifras a aquel mandato | Pedir el acuerdo posterior a las elecciones |

## 14. Decisiones pendientes

Las preguntas abiertas están numeradas en `dudas.md`. Cuando una se responde, se
quita de allí y la decisión se apunta en `docs/estudio_requisitos/`. Mientras no
haya respuesta, el valor provisional va en un catálogo que se cambia sin tocar
código, y nada de esto impide cumplir las condiciones de cierre del apartado 16.

| Quién decide | Qué | Dudas |
| --- | --- | --- |
| RRHH | Plantillas Word de contratos, informes y resoluciones | 124 |
| RRHH | Selección por el SAE: datos, canal y documento que la acredita | 72, 73, 126 |
| RRHH y Secretaría General | Delegaciones de firma, suplencias y cargos con que firma cada persona | 122, 128, 143, 148 |
| RRHH e Informática | Firmadoc o AutoFirma; SICAL; GINPIX; servidor de correo | 74, 125, 127, 88 |
| RRHH | Toma de posesión eficaz y fecha de efectos del cese | 140 |
| RRHH | Plazo para aportar documentos tras aceptar; audiencia antes de la baja | 18, 62 |
| RRHH | Urgencia; coste sin fecha de fin; plazo tras llamar por teléfono | 63, 146, 151 |
| RRHH | Inicio de los dos días de la oferta; llamadas en festivo; respuesta en firme desde «Mi Bolsa» | 153, 156, 157 |
| RRHH | Jornada en la petición del centro; categorías genéricas de la RPT | 154, 149 |
| RRHH | Inscripción desde el portal, requisitos de acceso y retirada de méritos | 155, 159, 160 |
| RRHH | Vuelta a la bolsa tras cesar en una vacante | 158 |
| RRHH | Que el correo valga como publicación (art. 8.1 del Reglamento de bolsas) | sin número |
| RRHH | Procesos de RRHH y aplicaciones actuales (base de los módulos aparcados) | 39 y 129 a 138 |
| DPD | Quién ve la lista de cada bolsa y con qué campos | 17, 83 |
| DPD | Uso del correo y teléfono que vienen de CONVOCA | 45 |
| DPD | Conservación de actuaciones de aspirantes externos | 97 |
| DPD y Seguridad | Separación de datos, evaluación de impacto, registro de actividades, categoría ENS | 77–79, 85, 86, 89 |
| Archivo y DPD | Conservación por serie documental | 60, 61, 84 |
| RRHH | Fuentes de requisitos de Cronos y Dietas | 19–26, 38, 40–51, 100–108 |
| RRHH | Personal, usuarios y perfiles que reparte Administración | 27–36, 141, 142 |

Pendiente de producto, no de RRHH: que la petición del centro elija la necesidad
publicada de la RPT en lugar de un texto libre (anotado en `dudas.md` con #1088).

Antes de usar datos reales en cada bloque hará falta además, de forma general:
el texto consolidado del Acuerdo de funcionarios y del Convenio; el inventario y
diccionario de GINPIX, WCRONOS, Portale, nómina, MOAD y Active Directory, con el
dueño de cada dato; los reglamentos de turnos y centros sociales; el circuito real
de altas, ceses y toma de posesión; las tablas de valoración documental; y el
catálogo de órganos, delegaciones, suplencias y firmas.

## 15. Integraciones externas

| Sistema | Para qué | Situación |
| --- | --- | --- |
| GrxFirma (antes AutofirmaV2) y su validador | Firmar y verificar | Elegido; composición real en curso (`docs/plan_modulos/firmas.md`) |
| Portafirmas de la Diputación (Firmadoc) | Circuito de firma corporativo | Conector existente, apagado; duda 74 |
| GINPIX | Gestión de personal y nómina | Sin integración; grabación manual al final; duda 127 |
| SICAL | Retención de crédito | La hace cada Delegación; integración en duda 125 |
| MOAD | Número de expediente | VEC lo guarda, no lo genera |
| SAE | Oferta genérica cuando no hay bolsa | Pendiente de reunión; dudas 72, 73, 126 |
| CONVOCA | Origen de las bolsas existentes | Importación de sus Excel; después se retira |
| Correo corporativo | Ofertas y avisos | Buzón de pruebas hasta tener los datos de Informática; duda 88 |
| WCRONOS | Control horario actual | Se sustituye por Cronos; inventario pedido en duda 19 |
| Active Directory/Kerberos | Identidad corporativa | Futuro acceso del personal interno |
| Calendarios externos | Agenda compartida (EMP-005) | Conectores sustituibles; no se priorizan |

## 16. Métricas de éxito

### Bolsa y Contratación temporal cerradas

Son las cuatro condiciones de la fase 1 de `docs/producto/PLAN_IMPLEMENTACION.md`
(apartado 5). Se comprueban en
la copia local con datos sintéticos, no en una prueba aislada (cidonia está
congelada desde el 16/10):

1. **Recorrido completo de RRHH en el navegador, con reinicio.** Con Chrome y
   certificados de personas distintas: petición del centro, ratificación, alta,
   análisis con crédito, informe firmado por técnica y Jefatura, fiscalización
   con un reparo y su subsanación, llamamiento desde la bolsa constituida,
   aceptación y resolución de la aceptación, propuesta, resolución firmada por
   Jefatura, Diputación y Secretaría General, incorporación, alta en Personal,
   confirmación manual de GINPIX, cese, vuelta a la bolsa y cierre. Se sigue
   el caso único de `FLUJO_APP.md` §4.0 y sus variantes. Después se
   reinician la aplicación y PostgreSQL y se repiten las consultas: mismos
   recibos, mismas fechas y ningún duplicado. Queda un acta con la respuesta HTTP
   y una captura de cada paso.
2. **Selectores encendidos.** Los interruptores de configuración que necesita ese
   recorrido están activos y el aviso de dependencias al arrancar no muestra
   ausencias. La lista exacta está en `PLAN_IMPLEMENTACION.md`, fase 1.
3. **Ningún «no disponible» sin motivo.** Cada botón funciona o explica por qué no
   está. El registro no tiene respuestas 404 ni 503 de Bolsa o CT durante el
   recorrido.
4. **Lecturas rápidas.** Cuadro de CT, lista y ficha de expedientes, bolsas,
   candidatos y ficha del candidato por debajo de 300 ms en el percentil 95,
   medido por HTTP en la copia local; la base común por debajo de 100 ms.

Al 20/10 la fase sigue abierta. Hay ensayos parciales de firma, varias plazas,
cuenta externa y cese; no sustituyen el acta conjunta del caso 4.0. El
[acta de rendimiento](acta_rendimiento_lecturas_f1_34_20261016.md) conserva
fallos y tiempos por encima del objetivo con peticiones simultáneas.

Dirección confirma el 20/10 a las 12:27 el hito: recorrido completo de un
expediente del caso 4.0 con reinicio antes del 6 de noviembre. Si no se llega,
se recalcula el fin de fase 1, estimado para la segunda quincena de noviembre.
Las decisiones pendientes de RRHH siguen con valores provisionales
configurables; no se presentan como aprobadas ni se usan datos reales.

### Para cualquier capacidad

Antes de programar una regla funcional, la puerta institucional del apartado 6.
Para darla por terminada:

- se puede recorrer: navegador, API, autorización, aplicación, PostgreSQL y
  recibo, con recuperación tras reinicio, recorrida en la copia local;
- los permisos están probados, también las denegaciones y la separación de zonas;
- reglas, fuentes y versiones aparecen en el resultado y en la auditoría;
- concurrencia, reintento y fallo parcial están probados;
- pantallas y documentos son accesibles;
- la ayuda y los manuales están al día;
- la responsable funcional, Seguridad y el órgano jurídico competente han
  aceptado su parte antes de usarla con datos reales.

Código escrito, revisado, probado, instalado, recorrible y publicado son estados
distintos y se dicen por separado.

## 17. Recorrido de la presentación

Alberto presentará VEC a RRHH haciendo el proceso entero y cambiando de perfil en
cada paso. Ningún paso puede estar roto ni mostrar un «no disponible». Usa los
personas ficticias de una copia local (aspirante, centro, ratificador, RRHH e
Intervención), cada uno con su certificado. Todo encargo se prioriza por este
recorrido. El detalle de pantallas y llamadas está en `docs/producto/FLUJO_APP.md`.

| # | Perfil | Paso | Dónde se ve |
| --- | --- | --- | --- |
| 1 | Aspirante | Se identifica con certificado, se inscribe y aporta títulos y méritos | Área del aspirante (`web/static/area-personal/`) |
| 2 | Técnico de RRHH | Revisa la inscripción y los méritos; la bolsa queda constituida | Portal de RRHH, Bolsas |
| 3 | Centro | Pide una persona eligiendo la necesidad de la RPT | «Peticiones del centro» (`web/static/portal-empleado/peticiones-centro/`) |
| 4 | Ratificador | Ratifica la petición | «Peticiones del centro» |
| 5 | Técnico de RRHH | Da de alta el expediente y analiza el crédito | Contratación temporal |
| 6 | Técnica de RRHH y Jefatura | Preparan y firman el informe para Intervención | Ficha del expediente |
| 7 | Intervención | Fiscaliza; pone un reparo; RRHH subsana y vuelve a fiscalizar | Ficha del expediente |
| 8 | Técnico de RRHH | Lanza el llamamiento desde la bolsa real | Contratación temporal y Bolsa |
| 9 | Aspirante | Ve la oferta y acepta | «Mi Bolsa» |
| 10 | RRHH y firmantes | Propuesta y resolución con las tres firmas del catálogo, verificadas | Ficha del expediente, GrxFirma |
| 11 | Centro | Confirma la incorporación | «Peticiones del centro» |
| 12 | Técnico de RRHH | Alta en Personal con su relación y ocupación | Personal |
| 13 | Técnico de RRHH | Descarga la ficha desde el alta confirmada y registra la grabación manual en GINPIX | Ficha del expediente |
| 14 | Técnico de RRHH | Cese | Contratación temporal |
| 15 | Aspirante | Vuelve a estar disponible en su bolsa | «Mi Bolsa» |

El recorrido principal empieza en el paso 3, la petición del centro. Los pasos 1
y 2 (inscripción del aspirante y constitución de la bolsa) se enseñan aparte y
no forman parte de la condición 1 de la fase 1: el 14/10 Alberto aclaró que el
recorrido puede empezar donde convenga. Se recorre en la copia local, con cada
paso obtenido desde su pantalla. Los datos ficticios de partida pueden
prepararse en la copia; el alta, la ocupación y el cese no se precargan.

Qué falta para cada paso en la copia local: ver `TABLERO_TAREAS.md`. El
orden exacto y los resultados de aceptación los fija `FLUJO_APP.md` §4.0;
esta tabla es un resumen.

## 18. Palabras que conviene conocer

| Palabra | Significado |
| --- | --- |
| cidonia | Servidor de ensayo anterior, congelado; el avance actual se comprueba en copias locales |
| Principal | Base de referencia de la que se obtiene el respaldo para crear una copia local propia |
| CT | Contratación temporal |
| RPT | Relación de puestos de trabajo |
| GINPIX | Aplicación externa de gestión de personal y nómina |
| Recibo | Constancia que VEC devuelve tras cada acto, con fecha y referencia; un reintento devuelve el mismo |
| Perfil fijo | Conjunto de permisos que el administrador asigna a una persona |
| Catálogo | Datos configurables y versionados (reglas, plazos, causas, plantillas, textos) |
| Selector | Variable de configuración que enciende o apaga una función |
| CSV | Código seguro de verificación de un documento firmado |

## 19. Diferencias conocidas entre fuentes

- Resuelta el 13/10: el antiguo `AGENTS.md` decía «todos los módulos en
  paralelo» (07/10); manda la orden del 08/10, Bolsa y CT primero con sus bases
  y los demás módulos después (`PLAN_IMPLEMENTACION.md`, apartados 1 y 3).
- El catálogo funcional (julio) ordenaba por fases F0 a F5 y nombraba un
  «conector de AutoFirma» y avisos por Telegram. Las decisiones vigentes son
  «base primero» con Bolsa y CT delante y GrxFirma. El consenso de dirección
  del 15/10 admite publicación y notificación como inicios configurables
  del plazo de las ofertas (art. 8.1 del Reglamento, duda 153 abierta).
  Telegram sigue en la duda 152.
- El análisis integral daba como «pendiente de resolución» el art. 17.4 del
  Reglamento de tiempo de trabajo; el Pleno del 2/09/2026 lo declaró nulo.
- El análisis integral contaba siete módulos en `internal/modules/` y decía que
  Formación no tenía módulo. Hoy hay quince, Formación incluida.
- La página raíz histórica (`web/static/index.html` con `web/static/app.js`)
  conserva una demostración que guarda datos en el almacenamiento del navegador.
  No forma parte del producto y choca con los apartados 6 y 11.

## 20. De dónde sale este documento

Este documento absorbe lo vigente de:

- `docs/estudio_requisitos/analisis_integral_rrhh.md` (apartados 1 a 10, 12, 13,
  15, 17, 18, 21 y la parte de modelo de 22);
- `docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md` (catálogo de
  capacidades, flujos y puertas de entrada y salida; las fases F0 a F5 se dejan
  por obsoletas);
- `docs/estudio_requisitos/README.md` (orden de fuentes, puerta institucional y
  decisiones de producto; las de despliegue van a `docs/producto/TRD.md`);
- la parte de producto de `ESPECIFICACIONES_AGENTES.md` (E02 completo y lo
  funcional de E03, E04, E05, E07, E08 y E09);
- el alcance de los planes de `docs/plan_modulos/` (sus inventarios, estados y
  tareas siguen en cada plan y en `docs/producto/PLAN_IMPLEMENTACION.md`).

## 21. Cómo mantener este documento

- **Quién:** quien cambie el alcance, los perfiles, una regla de producto o la
  prioridad, en la misma PR (skill `documentos-producto-vec`). Dirección (Claude)
  revisa que siga cuadrando con `PLAN_IMPLEMENTACION.md` y con los planes de módulo al
  integrar.
- **Cuándo:**
  - al cerrar o aparcar un módulo, o cambiar su prioridad (apartado 8);
  - al aparecer o desaparecer un perfil o una superficie (apartado 5);
  - al decidir RRHH o el DPD algo que cambie una regla (apartados 9 y 14);
  - al cambiar las condiciones de cierre de la fase 1 del plan (apartado 16);
  - al cambiar el recorrido de la presentación (apartado 17).
- **Cómo:** un cambio pequeño basta; no se reescribe entero por una línea. Se
  actualiza la fecha de corte y el hash de `main`. El estado de cada módulo se
  resume aquí en una frase; el detalle vive en su plan. Una capacidad nueva entra
  con su código en el apartado 9. No se pegan secretos, rutas privadas, nombres de
  servidores internos ni datos personales.
- **Si no cuadra:** si este documento contradice el código o una orden vigente,
  mandan el código o la orden; se corrige aquí y se anota en la entrega.
