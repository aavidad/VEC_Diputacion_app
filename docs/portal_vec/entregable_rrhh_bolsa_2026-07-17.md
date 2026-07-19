# Entregable urgente para RRHH: Portal del Empleado y Gestión de Bolsas

## Decisión y alcance

La interfaz se construye como tres superficies separadas y candidatas a
reutilizarse en producción:
consulta pública, área personal del aspirante y gestión interna de RRHH. Todas
reutilizan el lenguaje visual VEC ya incorporado al repositorio. No se crea un
segundo producto ni se copia la aplicación antigua a otra base de código.

El alcance visible pedido por RRHH es:

1. lanzador explícito de los tres puntos de vista;
2. consulta pública de convocatorias, categorías, plazos y ayuda;
3. área personal del aspirante con su expediente completo de Bolsa;
4. portada con varios módulos del Portal del Empleado y solo `Bolsas de
   trabajo` habilitado en la fase inicial;
5. cuadro de mando de Bolsa basado en la referencia visual `image006.png`
   facilitada por RRHH y deliberadamente no distribuida con el repositorio;
6. elaboración y gestión de bolsas;
7. asistente de llamamientos basado en la referencia visual `image004.png`,
   sometida a la misma política de no distribución;
8. navegación por convocatorias, solicitudes, méritos, baremación,
   alegaciones, importación, llamamientos, contratos, estadísticas, documentos,
   comunicaciones y auditoría;
9. configuración de roles y permisos con política de denegación por defecto;
10. separación visible y técnica entre la zona interna y la zona externa.

El HTML, el CSS adaptable, la navegación, los renderizadores, los estados
accesibles y los contratos son código de producto candidato a reutilizarse;
RRHH aún debe aceptar los recorridos. No hay una maqueta alternativa que haya
que desechar por completo. Los títulos, fechas, categorías y CVE/BOP de la
consulta son metadatos públicos reales contrastados. Los datos privados y
efectos sintéticos necesarios para enseñar los recorridos antes de disponer de
todas las APIs se aíslan en adaptadores de presentación intercambiables y no se
mezclan con los clientes normales.

El corte actual consta de **36 vistas navegables**:

| Superficie | Número | Contenido |
| --- | ---: | --- |
| Lanzador | 1 | Entrada controlada a cada recorrido. |
| Bolsa pública | 1 | Consulta anónima. |
| Área personal | 14 | Inicio; convocatorias; detalle; perfil; méritos; solicitud; autobaremación; seguimiento; llamamientos; subsanaciones; alegaciones; mensajes; certificados; ayuda. |
| Portal interno | 1 | Portada común y catálogo modular del Portal del Empleado. |
| Gestión interna de Bolsa | 17 | Capacidades agrupadas en las diez áreas visuales 1–10 solicitadas por RRHH. |
| Cronos | 1 | Jornada, fichajes y permisos con identidad común de presentación. |
| Dietas | 1 | Comisiones, gastos y liquidaciones con identidad común de presentación. |

El inventario anterior describe pantallas evaluables, no capacidades
productivas. Una vista marcada `DEMO` no acredita integración, una prueba E2E
productiva ni un acto administrativo válido.

La puerta automática de presentación ha recorrido **36 vistas y 22 estados de
interacción** representativos en tres resoluciones. El cierre del 19 de julio
de 2026 produjo **174 capturas, 174 escenarios correctos y cero hallazgos**.
Incluye los menús móviles abiertos, el perfil técnico restringido, el recorrido
de solicitud del aspirante, operaciones internas con recibo `DEMO-REC-*` y la
ruta OSM/OSRM real de Dietas. En todos los casos se trata de
evidencia de la muestra aislada, no de aceptación de RRHH ni de una prueba E2E
productiva.

Las superficies se dividen sin herramienta de compilación en piezas
cohesionadas:

- `index.html`: estructura semántica y zonas de navegación;
- `portal.css`: tema, carcasa y controles comunes;
- `portal-componentes.css`: paneles, tarjetas, tablas e indicadores;
- `portal-flujos.css`: formularios, asistente, accesibilidad adaptable e
  impresión;
- `portal.js`: composición, carga cerrada y navegación interna;
- `portal-contrato.js`: validación pura de envelopes, panel y propuestas;
- `portal-panel-interno.js`: presentación exclusiva del agregado seguro real;
- `portal-eventos.js`: interacción sin decisiones de negocio;
- `portal-vistas-*.js`: renderizadores compartidos por la presentación y la
  composición normal, candidatos a la futura composición productiva;
- `portal-presentacion-adaptador.js`, `portal-borradores-demo-cliente.js` y
  `datos-presentacion.js`: adaptadores y datos internos volátiles, cargados
  solo de forma explícita;
- `area-personal/aplicacion.js` y `area-personal/vistas/*.js`: navegación y 14
  renderizadores del aspirante compartidos por ambos modos;
- `area-personal/cliente-http.js`: cliente real que falla cerrado;
- `area-personal/adaptador-presentacion.js`: estado sintético del aspirante en
  memoria volátil;
- `presentacion/index.html`: lanzador exclusivo del artefacto de muestra;
- `ayuda-contenido.js`: pasos, FAQ, transcripción y referencia de audio
  sustituibles por catálogo;
- `assets/ayuda-llamamiento-bolsa.mp3`: guía local sin datos personales.

Cada fichero cumple el tope de 800 líneas de DEC-051.

## Rutas

### Aplicación normal

`/bolsa/`, `/area-personal/` y `/portal-empleado/`

La consulta pública no comparte identidad ni navegación con las dos
superficies privadas. El área personal y el portal interno seleccionan sus
clientes HTTP reales en la raíz de composición y fallan cerrados si faltan
identidad, permiso, capacidad o API. Nunca recurren de forma automática al
estado de presentación.

En particular, el agregado inicial de `/portal-empleado/`:

- en la carga inicial consulta únicamente `GET /api/vec/bolsa/panel`;
- exige el envelope canónico `{ "data": { ... } }` y rechaza una proyección
  raw situada en la raíz;
- ejecuta `fetch` con `credentials: "omit"`: no crea ni consume cookies y no
  guarda credenciales en JavaScript; la futura aplicación de escritorio y la
  frontera autenticada deberán aportar autorización explícita;
- rechaza una respuesta marcada como `demostracion`;
- ante `401`, `403`, `404`, `501` o error de red muestra acceso cerrado;
- no sustituye el error por datos locales;
- no guarda datos de negocio en el navegador.

El área personal aplica la misma regla a cada operación: usa
`credentials: "omit"`, no guarda credenciales y solo habilita una acción si el
servicio real anuncia la capacidad correspondiente.

Ya existe el adaptador HTTP estricto para `GET/HEAD` de esta ruta, pero aún no
está montado. Rechaza query, cuerpo, cookies, credenciales de proxy y cabeceras
heredadas de identidad o rol; tampoco interpreta `Authorization`. La frontera
preparadora deberá resolver del lado servidor actor, perfil, ámbito, motivo y
correlación antes de invocar el caso de uso. Por tanto, la ruta sigue fallando
cerrada hasta que se complete la vertical real.

### Presentación explícita para RRHH

La forma soportada de arrancar la muestra es exclusivamente Docker:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

Este lanzador construye y ejecuta portal, proxy, mediador, OSRM y teselas en
contenedores, espera su salud y lanza la prueba rápida interna. No instala ni ejecuta
Go, Playwright, Chromium, OSRM, TileServer GL o Nginx directamente en el
anfitrión.

El proxy es el único servicio que publica un acceso, en
`http://127.0.0.1:8081`; el punto de entrada es `/presentacion/`. Desde él se
abren:

- consulta pública: `/bolsa/`;
- área personal: `/area-personal/?presentacion=rrhh`;
- gestión interna: `/portal-empleado/?presentacion=rrhh&perfil=tecnico#portal`
  o `perfil=administrador`; el perfil es obligatorio y nunca se eleva por
  omisión.

Accesos directos útiles:

- cuadro de mando: `#bolsa/resumen`;
- elaboración: `#bolsa/elaboracion`;
- llamamientos: `#bolsa/llamamientos`.

El portal `vec-presentacion` exige el perfil, el selector y las dos guardas
literales documentadas. El mediador independiente
`vec-cartografia-presentacion` repite ese cierre y solo permite calcular rutas
contra el OSRM interno autorizado. Solo el parámetro exacto
`presentacion=rrhh` selecciona en las superficies privadas sus adaptadores
volátiles. Las pantallas distinguen de forma visible las referencias públicas
reales de los datos privados sintéticos y advierten de la ausencia de validez
administrativa. Las acciones pueden modificar el escenario
durante la visita y emitir un recibo `DEMO-`, pero el estado se pierde al
recargar y nunca firma, registra, paga, envía ni persiste fuera de la memoria.

La muestra no usa cookies, `localStorage`, `sessionStorage` ni volumen durable.
La única comunicación entre componentes es la cartografía interna de Dietas:
el navegador usa el mismo origen, el mediador consulta OSRM en una red Docker
exclusiva y las teselas OSM se sirven localmente. No existe salida a Internet.
El destino Docker productivo elimina físicamente el
lanzador, los ficheros `.demo.json`, el binario y cualquier ruta cuyo nombre
contenga `presentacion` o `demo`. La explicación reproducible está en el
[modo de presentación RRHH](modo_presentacion_rrhh.md) y la correspondencia
pantalla/contrato/adaptador en la
[matriz de aceptación](matriz_aceptacion_web_bolsa_2026-07-18.md).

La revisión visual también se ejecuta en Docker, mediante el servicio
`revision-web-presentacion` del perfil `herramientas-presentacion`. Las capturas
y los informes quedan bajo `var/` y no forman parte del repositorio.

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

### Bolsa pública

`/bolsa/`

La zona anónima conserva una navegación propia, pero no hereda enlaces del
Portal del Empleado ni del backoffice. El menú lateral solo permite acceder a
convocatorias, búsqueda, categorías profesionales y ayuda pública. En
pantallas estrechas se transforma en una cabecera compacta de dos columnas; no
desaparece. Esta separación evita presentar accesos internos sin sacrificar la
orientación ni la coherencia visual.

La consulta sigue identificada como demostración porque sus documentos son
reproducciones adaptadas y no una publicación administrativa del sistema. Sus
títulos, fechas, categorías y CVE proceden de publicaciones oficiales del BOP.
El menú, el logotipo institucional, la adaptación móvil,
el alto contraste y la ayuda son componentes reutilizables pendientes de
aceptación. El cambio se
incorporó en `29afe4d` y cuenta con una prueba específica que impide introducir
en esta superficie cualquier destino ajeno a la lista positiva de cinco rutas
públicas y evita que una regla CSS de pantalla retire el menú.

La composición `cmd/vec-publico` permite arrancar esta zona como proceso
independiente. Su lista positiva no incluye `/`, `/portal-empleado`,
`/api/vec`, `/api/demo` ni la API heredada, y no carga Personal, credenciales
de demostración o almacenes privados. Existe además una raíz HTTP interna
separada, todavía sin proceso productivo, que solo admite Portal del Empleado y
`/api/vec` y prohíbe cookies en ambos sentidos.

## Inventario exhaustivo de elementos temporales

Esta sección conserva el resumen funcional del entregable. El inventario
operativo por ruta, con decisión de retirada, sustituto, condición de
aceptación, prueba de ausencia y marcha atrás, se mantiene en
[Inventario y retirada incremental de la presentación RRHH](inventario_retirada_presentacion_2026-07-19.md).

| Elemento | Ubicación | Qué hace ahora | Sustitución obligatoria |
| --- | --- | --- | --- |
| Datos sintéticos internos | `portal-empleado/datos-presentacion.js` | Aporta bolsas, expedientes, reglas, lotes y agregados con referencias `DEMO-` | Se excluye físicamente de producción; los renderizadores reciben proyecciones autorizadas de la API interna |
| Adaptador interno volátil | `portal-presentacion-adaptador.js` y `portal-borradores-demo-cliente.js` | Aplica durante la visita las operaciones permitidas y emite recibos `DEMO-` | Casos de uso reales, autorizados por operación, recurso, ámbito y finalidad, con transacción e idempotencia |
| Perfiles internos sintéticos | `datos-presentacion.js.sesion` | Se elige explícitamente técnico revisor o administrador; cada uno tiene actor opaco, vistas y operaciones propias | Proyección mínima del principal autenticado, resuelta y revalidada en servidor |
| Datos y perfil del aspirante | `area-personal/adaptador-presentacion.js` | Muestra exclusivamente `Persona Aspirante de Demostración`, referencias `DEMO-` y correo reservado `.test` | API privada que solo proyecte información propia tras identidad y autorización reales |
| Indicadores y gráficos | adaptadores de presentación | Facilitan validar composición y jerarquía visual | Agregados calculados en servidor con fecha de corte, ámbito y origen; nunca sumar expedientes ajenos en el navegador |
| Ruta y mapa de Dietas | `vec-cartografia-presentacion`, OSRM y TileServer GL | Calcula una ruta real sobre un grafo gobernado y presenta teselas OSM locales, sin identidad ni efectos administrativos | Conector productivo autorizado, persistencia de la versión del grafo y auditoría de la liquidación; la vista y los puertos se conservan |
| Botones de alta, revisión, firma, registro, pago, envío y exportación | controladores inyectados de ambas superficies | Exigen confirmación, cambian solo memoria y muestran actor, objetivo y recibo sin efectos reales | Comandos API autenticados, autorizados, idempotentes, con control de versión y recibo durable |
| Propuesta de llamamiento | `obtenerPropuestaPresentacion` | Permite revisar evaluaciones sintéticas sin realizar una petición | `POST /api/vec/bolsa/propuestas-llamamiento`, autorizado e idempotente; el servidor decide elegibilidad y prelación |
| Ficheros seleccionados | formularios del aspirante | Conservan como máximo el nombre durante la interacción; no leen ni envían el contenido | Carga directa al almacén autorizado, cuarentena, antivirus, huella, cifrado, autorización y recibo |
| Configuración de bases, baremo, roles y calendarios | vistas internas compartidas | Versiona únicamente el escenario efímero para evaluar el flujo | Casos de uso gobernados y repositorios autorizados, con vigencia, fuente jurídica, segregación, firmas y publicación |
| Canales correo/Telegram | estado visible «no conectado» | Enseña el diseño y simula el cambio de estado sin destinatario real | Puertos de comunicación y adaptadores aprobados que devuelvan recibos verificables; ningún conector simulado concede efecto |
| Preferencias de texto y contraste | estado de la instancia de la página | Permite evaluar accesibilidad durante la visita | Preferencias del usuario mediante un contrato aprobado; la presentación no usa almacenamiento del navegador |
| Ayuda y audio | `ayuda-contenido.js`, vistas de ayuda y MP3 local | Pasos, FAQ, audio y transcripción accesibles | Catálogo de ayuda versionado y conector de formatos/audio; la interfaz conserva el mismo contrato |

No existe uso de `localStorage`, `sessionStorage` ni cookies en las superficies
de presentación. Una recarga restaura el escenario inicial completo. Las
pruebas automáticas impiden reintroducir almacenamiento de navegador, tráfico
externo o identidades con apariencia real.

## Correspondencia con las fotografías

### Cuadro de mando (`image006.png`)

Se conservan:

- navegación funcional del portal y las 17 secciones internas de Bolsa,
  agrupadas en las diez áreas visuales 1–10;
- cinco indicadores principales;
- bolsas destacadas con cobertura;
- próximos llamamientos;
- actividad reciente;
- indicadores agregados;
- accesos rápidos;
- contexto de usuario, avisos y ayuda.

Se mejora:

- aviso inequívoco del origen sintético;
- semántica HTML y navegación por teclado;
- foco visible, salto al contenido y región de anuncios;
- tablas desplazables y composición móvil;
- alto contraste y aumento de texto;
- separación entre estado, tendencia y acciones;
- identificadores personales enmascarados.

### Nuevo llamamiento (`image004.png`)

Se conservan:

- cuatro pasos;
- elección de una necesidad de cobertura y su bolsa aplicable;
- resumen lateral;
- petición explícita de propuesta;
- orden de prelación aplicado por el servidor;
- tabla de evaluaciones minimizadas, sin identidad ni contacto;
- configuración de condiciones, plazo y canales;
- revisión final.

La versión final no descarga el listado de integrantes ni ofrece casillas de
selección, búsqueda por nombre o búsqueda por documento. El servidor propondrá
y revalidará las personas elegibles con la versión exacta de bolsa y reglas,
autorización vigente y trazabilidad atómica.

### Área personal del aspirante

Las 14 vistas cubren el recorrido desde la consulta de una convocatoria hasta
el seguimiento del expediente: perfil propio, inventario reutilizable de
méritos y documentos, solicitud por pasos, autobaremación, pago o exención,
firma y registro, disponibilidad y llamamientos, subsanaciones, alegaciones,
mensajes, certificados y ayuda accesible. Cada acción de presentación exige
confirmación y devuelve un recibo sintético; la misma acción en modo normal
permanece deshabilitada si el cliente real no recibe una capacidad positiva.

## Cobertura funcional real a 19 de julio de 2026

| Área | Estado comprobado | No se debe afirmar todavía |
| --- | --- | --- |
| Artefacto de presentación | Portal y mediador cartográfico separados, único acceso por el proxy en `127.0.0.1:8081`; 36 vistas y 22 flujos obtuvieron 174/174 en tres resoluciones, con 174 capturas y cero hallazgos | Integración productiva, E2E productivo, aceptación formal o validez administrativa |
| Consulta pública de convocatorias y categorías | Recorrido local con 36 procesos basados en 37 publicaciones BOP reales, documentos adaptados y aviso DEMO en `/bolsa/` | Publicación oficial desde un expediente interno |
| Área personal del aspirante | 14 vistas reutilizables pendientes de aceptación, con cliente HTTP real cerrado y adaptador de presentación seleccionado en composición | Identidad real, autorización sobre datos propios, persistencia, pago, firma, registro, carga o descarga efectiva |
| Gestión interna de Bolsa | Portal más 17 secciones reutilizables agrupadas en el menú visual 1–10; operaciones volátiles con confirmación y recibo `DEMO-` | Identidad interna reforzada, permisos reales, transacciones durables, firma, publicación, comunicación o integración corporativa |
| Cronos y Dietas en el portal | Dos verticales iniciales reutilizables, con identidad compartida y adaptadores de presentación independientes; la ruta OSRM y las teselas OSM internas versionadas de Dietas superaron la matriz visual integral | Integración productiva, equivalencia funcional completa, confinamiento de Cronos a la red corporativa y aceptación de RRHH/Sistemas |
| Portal VEC heredado | Carcasa, perfiles, menús y numerosas vistas reutilizables fuera del corte de Bolsa | Portal privado estable: `/api/vec/workspace` falla cerrado sin ámbito resuelto |
| Elaboración de bolsa histórica | Formulario visual heredado; dominio de convocatoria gobernada y contrato HMAC V2 probado, sin sellado durable previo al PDP ni huella semántica cruda durable | Resolvedor autoritativo de ámbito, diario de recuperación, persistencia, API, firma, dependencias y publicación oficiales |
| Panel interno de Bolsa | Dominio, servicio de aplicación, contrato agregado sin datos personales, consulta PostgreSQL y pruebas de integración | Endpoint compuesto, identidad interna real, autoridad COSE de ejecución y productor de la proyección |
| Llamamientos | Dominio y caso de uso probados; el comando transporta la instantánea completa y un adaptador genera referencias opacas con 256 bits de aleatoriedad criptográfica; el esquema PostgreSQL V1 conserva auditoría y outbox | El adaptador PostgreSQL permanece cerrado hasta disponer de contrato SQL atómico V2, fuente autoritativa, motor publicado, autoridad COSE de efecto, API y confirmación/envío real |
| Integrantes/candidatos | Área personal completa en presentación y datos de catálogo disponibles | Listado administrativo productivo y orden durable de bolsa |
| Autobaremación | Flujo evaluable y candidato a producción con adaptador volátil y núcleo nuevo de baremación avanzado | Aceptación de RRHH y reglas configurables conectadas extremo a extremo |
| Revisión firmada de baremación | Dominio, aplicación y PostgreSQL avanzados | Composición en servidor, API e interfaz operativa |
| Documentos de aspirantes | Puertos S3, cuarentena y firma modelados | Subida y firma de Bolsa operativas; el endpoint actual responde `503` |
| Alegaciones | Presentación completa del aspirante y resolución interna volátil | Persistencia probatoria y resolución real; el cliente normal falla cerrado sin capacidad |
| Comunicaciones | Preparación y envío simulados con referencias sintéticas y sin tráfico externo | Entrega, acuse, Telegram, correo o notificación fehaciente conectados |
| Auditoría | Auditoría heredada parcial y registro probatorio en baremación | Trazabilidad unificada de elaboración, llamamientos y comunicaciones |

## Contrato de lectura seguro implementado

La pantalla normal espera el envelope `{ "data": { ... } }`; dentro de `data`
debe existir la proyección interna mínima
`vec.bolsa.panel.interno.v1`. Una raíz raw se rechaza aunque contenga campos
aparentemente válidos. El contrato Go ya implementado contiene:

- `selector`: organización o unidad de gestión exacta autorizada, nunca un
  comodín;
- `origen`: revisión, fecha de actualización y `demostracion: false`;
- `prueba_lectura`: referencias opacas de lectura, decisión y auditoría
  confirmadas;
- doce `indicadores` agregados de convocatorias, bolsas, llamamientos,
  documentos e incidencias;
- `convocatorias`: referencias y claves gobernadas, sin datos de personas;
- `actuaciones_pendientes`: trabajo administrativo agregado, sin actor ni
  interesado.

Las colecciones más amplias de la presentación —bolsas destacadas, contratos,
reglas, documentos, canales, series y actividad— no forman parte de esta
primera proyección segura. La aplicación real debe marcarlas como no disponibles
hasta disponer de consultas autorizadas; no debe convertir su ausencia en
ceros ni listas vacías que aparenten datos reales. La identidad visible de la
sesión se proyectará por una frontera separada y no se mezclará con el agregado
de Bolsa. Su contrato ya liga referencias autoritativas `aut_`, `ase_`,
`ses_`, `cse_` y `cta_`, distingue cuenta de acceso y cuenta ordinaria y vuelve
a cotejarlas antes de cada proyección. No se montará hasta disponer del
registro durable real que las emita y revoque.

El servidor debe devolver `demostracion: false`. La interfaz rechaza de forma
explícita una fuente que intente mezclar datos de demostración en la ruta
normal.

El contrato global prohíbe expresamente la propiedad `candidatos`. La propuesta
no forma parte del `GET`: se obtiene mediante un comando separado y su tabla
solo admite `secuencia`, `resultado`, `puntuacion`, `regla` y `fundamento`. El
validador rechaza nombre, DNI/documento, teléfono, correo, contacto o
identificadores de persona/candidato.

Antes de estabilizar el contrato se añadirán:

- paginación y límites máximos;
- filtros tipados;
- referencias opacas, no claves de base de datos;
- fecha de corte y zona horaria;
- versión/ETag de cada agregado mutable;
- capacidades granulares por acción, recurso y transición, además de la
  capacidad inicial de solicitar propuesta;
- política de minimización por rol y finalidad.

Los clientes y renderizadores reutilizables no contienen nombres, DNI,
expedientes, categorías, fechas ni cifras del juego de presentación. Las
pruebas automatizadas impiden reintroducir identidades con apariencia real y
mantienen los fixtures dentro de los adaptadores aislados.

## Mapa de sustitución por pantalla

| Pantalla | Base reutilizable existente | Corte real siguiente |
| --- | --- | --- |
| Lanzador | Artefacto exclusivo `vec-presentacion` | No se incorpora a producción; solo se conserva para formación o validación visual local |
| Portada | Registro de módulos y manifiestos VEC | Habilitación administrable por despliegue y rol, no lista fija del navegador |
| Área personal | 14 renderizadores, contrato validado y cliente HTTP de fallo cerrado | Identidad de aspirante, autorización sobre información propia y composición de cada API real |
| Cuadro de mando | Dominio, aplicación, consulta PostgreSQL y adaptador HTTP no montado de `vec.bolsa.panel.interno.v1` | Identidad interna, autoridad COSE, publicador de proyección y composición productiva dedicados |
| Elaboración | `domain/convocatoria_gobernada*.go` | Repositorio PostgreSQL, aplicación, HTTP y composición |
| Llamamientos | `application/llamamientos.go`, comando indivisible con instantánea completa y esquema PostgreSQL V1 cerrado | Fuente y motor autoritativos, guardado SQL completo V2, fachada HTTP interna y composición |
| Contratos/ceses | Estados de elegibilidad del núcleo de llamamientos | Agregado de relación temporal y eventos que recalculan disponibilidad |
| Reglas | Estudios de baremación configurable y dominio de convocatoria | Catálogo versionado, editor, simulación, aprobación y publicación firmada |
| Consulta candidato | `/bolsa/` y API pública | Zona privada propia separada de la gestión interna |
| Estadísticas | Estructura visual de agregados | Consultas anonimizadas con fecha de corte y control de inferencia |
| Documentos | Puertos de formato, S3, firma y cotejo | Composición específica de Bolsa, plantilla gobernada y circuito de firmas |
| Comunicaciones | Puertos de notificación | Adaptadores reales y recibos fehacientes |
| Auditoría | Auditoría VEC y outbox probatoria de baremación | Proyección unificada con integridad y retención aprobadas |
| Ruta y mapa de Dietas | Vista y puertos reutilizables, Leaflet local, mediador aislado, OSRM y teselas OSM versionadas | Composición interna autenticada, autorización de Dietas, persistencia de la versión de cálculo y auditoría del expediente |

## Comandos que debe implementar la API

Los botones finales se conectarán a comandos separados. Como mínimo:

- crear borrador de convocatoria;
- versionar bases y reglas;
- validar, firmar y publicar una versión;
- solicitar una propuesta de llamamiento;
- revisar la propuesta;
- confirmar el llamamiento con revalidación atómica;
- registrar respuesta, renuncia y justificación;
- registrar nombramiento/contrato, cese y reincorporación;
- generar y enviar documentos;
- consultar recibos y trazabilidad.

Todos exigirán:

- identidad interna reforzada;
- permiso positivo y ámbito sobre el recurso;
- transición de estado válida;
- clave de idempotencia;
- control de versión concurrente;
- motivo cuando proceda;
- evento de auditoría y recibo;
- ausencia de decisiones confiadas al cliente.

### Contrato preparado para solicitar una propuesta

La interfaz queda preparada para:

```http
POST /api/vec/bolsa/propuestas-llamamiento
Content-Type: application/json
Accept: application/json
Idempotency-Key: <UUID aleatorio y estable para el intento>
```

```json
{
  "data": {
    "esquema": "vec.bolsa.propuesta-llamamiento.solicitud.v1",
    "necesidad_id": "<referencia opaca autorizada>"
  }
}
```

La respuesta también debe usar `{ "data": { ... } }` y el esquema
`vec.bolsa.propuesta-llamamiento.v1`. El navegador solo ejecuta este `POST` si
el panel real ha devuelto `capacidades.solicitar_propuesta_llamamiento=true`.
Con capacidad ausente o `false` no se invoca `fetch`. En la presentación la
capacidad permanece a `false`: el botón carga exclusivamente la propuesta
sintética local, lo anuncia y no realiza tráfico de escritura.

Cambiar de necesidad invalida la propuesta visible y la clave de idempotencia
del intento anterior. La confirmación definitiva será otro comando, con nueva
capacidad, nueva clave y revalidación transaccional; nunca reutilizará como
decisión el estado del navegador.

## Ayuda accesible y audio local

La ayuda del menú abre un componente reutilizable con cuatro pasos, preguntas
frecuentes, reproductor HTML nativo y transcripción completa. El contenido
vive en `ayuda-contenido.js`, no disperso entre controladores, para poder
sustituirlo por un catálogo versionado o un conector de ayuda sin rehacer la
vista.

El MP3 se generó localmente siguiendo el mecanismo ya usado en OPES:
`edge-tts`, voz `es-ES-AlvaroNeural`. Se validó con Whisper `medium`; la
transcripción obtenida coincide en significado y secuencia con la transcripción
visible, dura aproximadamente 46 segundos y no contiene datos personales. El
portal no usa red, hotlink ni servicio TTS durante la reproducción.

Referencia de control del activo: 281232 bytes; SHA-256
`a0b0eb8d0eb1111cf32a2ad0f94929f3d6211a128290d10efd3b1e422f5bfa25`.

## Identidad visual institucional

Se reutiliza el activo institucional local ya empleado por el portal de la
Diputación, copiado dentro del artefacto como
`web/static/portal-empleado/assets/logo-diputacion-granada.svg`.
El fichero incorporado tiene la huella SHA-256:
`99ea04b463a34dbc9399e91eaee085dda6d22554278a4a115517ab4318014098`.

La cabecera lo referencia desde el mismo origen, con texto alternativo
`Diputación de Granada`, dimensiones intrínsecas `250 × 84` y una regla
adaptable que conserva la proporción. El fondo blanco garantiza que los
trazados azul oscuro y verde sean visibles sobre la navegación corporativa.
No se usa hotlink ni se presenta el favicon genérico de VEC como marca oficial.
La prueba de interfaz comprueba además que el SVG no contiene `script`,
`foreignObject` ni manejadores `onload`.

## Autoridad criptográfica y motivo del cierre actual

El formato canónico VEC-AD-2, el servicio de atestación, el COSE Sign1 de
payload separado y el verificador Ed25519 estricto ya están implementados y
probados. También existen ya el catálogo PostgreSQL versionado de confianza
pública y su cargador Go real. El runner conjunto sobre PostgreSQL 18.4 prueba
el contrato de dieciséis columnas, la huella cruzada Go/PostgreSQL, identidad y
ACL mínimas, RLS, revocación concurrente y caducidad bajo bloqueo.

Este catálogo no contiene claves privadas ni concede autoridad de negocio. Lo
que falta para una ejecución productiva es el firmante HSM/KMS o proceso
aislado, el manifiesto criptográfico de los actos de gobierno, un anclaje
externo que detecte la restauración de una copia antigua, el broker por socket
Unix, la capacidad efímera de un solo uso, los registradores específicos para
panel y llamamientos y la revalidación de revocación dentro del mismo `COMMIT`
que produce el efecto. Hasta completar esa cadena, no se conceden permisos de
negocio y los endpoints internos no se montan. Una verificación local acredita
integridad criptográfica, pero por sí sola no concede autoridad para alterar o
consultar datos administrativos.

## Criterio para sustituir los adaptadores de presentación

Los adaptadores, el lanzador, los datos `.demo.json`, `vec-presentacion` y
`vec-cartografia-presentacion` **ya están excluidos físicamente** del destino
Docker productivo.
No se espera a terminar las APIs para obtener esa separación. La prueba
`scripts/verificar_contenido_artefactos_presentacion.sh` construye e inspecciona
los tres destinos VEC —producción, portal de presentación y mediador
cartográfico— y falla si producción contiene una pieza de muestra o si el
mediador incorpora el portal, datos o un binario ajeno.

Cada adaptador volátil dejará de ser necesario para una capacidad cuando:

1. su contrato y caso de uso estén cerrados;
2. la API esté montada con identidad, ámbito y permisos reales;
3. la operación sea durable, idempotente y devuelva un recibo verificable;
4. las pruebas de seguridad impidan consultar o modificar otro ámbito,
   expediente o persona;
5. exista una prueba E2E técnica con los conectores autorizados y los casos de
   carga, vacío, error, denegación, concurrencia y recuperación;
6. RRHH haya aceptado formalmente el recorrido correspondiente.

La sustitución se realiza en la raíz de composición, no reescribiendo la
pantalla. La presentación puede conservar su adaptador para formación local,
pero producción nunca lo importa ni cae a él si falla un servicio real.

## Prioridad de implementación después de la presentación

1. Completar sobre el catálogo COSE V2 el firmante aislado, el manifiesto de
   gobierno, el anclaje externo y los registradores de consumidores; mantener
   todo cerrado por defecto.
2. Componer la proyección PostgreSQL real del cuadro de mando con identidad
   interna, PDP y motivo catalogado.
3. Completar fuente, motor y transacción autoritativos de llamamientos.
4. Resolvedor autoritativo del ámbito desde el expediente, persistencia y API
   de elaboración de convocatorias.
5. Confirmación atómica del llamamiento y trazabilidad.
6. Respuestas, renuncias, contratos, ceses y reincorporaciones.
7. Documentos, firmas, publicación y comunicaciones fehacientes.
8. Estadísticas y explotación con anonimización aprobada.

El diseño técnico de confianza se conserva en
`docs/portal_vec/atestacion_criptografica_decisiones.md`; debe leerse junto con
el estado actualizado de este entregable y no como autorización para abrir las
funciones SQL.
