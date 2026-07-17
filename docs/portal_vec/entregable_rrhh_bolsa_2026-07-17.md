# Entregable urgente para RRHH: Portal del Empleado y Gestión de Bolsas

## Decisión y alcance

La interfaz se construye como superficie definitiva del Portal del Empleado,
reutilizando la carcasa y el lenguaje visual VEC ya incorporados al repositorio.
No se crea un segundo producto ni se copia la aplicación antigua a otra base de
código.

El alcance visible pedido por RRHH es:

1. portada con varios módulos del Portal del Empleado;
2. solo `Bolsas de trabajo` habilitado en la fase inicial;
3. cuadro de mando de Bolsa basado en la referencia visual `image006.png`
   facilitada por RRHH y deliberadamente no distribuida con el repositorio;
4. elaboración y gestión de bolsas;
5. asistente de llamamientos basado en la referencia visual `image004.png`,
   sometida a la misma política de no distribución;
6. navegación por contratos, reglas, consulta, estadísticas, documentos,
   comunicaciones y auditoría;
7. separación visible y técnica entre la zona interna y la zona externa.

El HTML, el CSS adaptable, la navegación, los estados accesibles y el contrato
de lectura pertenecen a la aplicación final. No hay una maqueta alternativa
que después haya que reescribir. Los datos sintéticos necesarios para enseñarla
antes de disponer de la API se aíslan en un único adaptador eliminable y no se
mezclan con el cliente normal.

La superficie se divide sin herramienta de compilación en piezas cohesionadas:

- `index.html`: estructura semántica y zonas de navegación;
- `portal.css`: tema, carcasa y controles comunes;
- `portal-componentes.css`: paneles, tarjetas, tablas e indicadores;
- `portal-flujos.css`: formularios, asistente, accesibilidad adaptable e
  impresión;
- `portal.js`: contrato, carga cerrada y vistas;
- `portal-contrato.js`: validación pura de envelopes, panel y propuestas;
- `portal-panel-interno.js`: presentación exclusiva del agregado seguro real;
- `portal-eventos.js`: interacción sin decisiones de negocio;
- `datos-presentacion.js`: único adaptador sintético, cargado solo de forma
  explícita;
- `ayuda-contenido.js`: pasos, FAQ, transcripción y referencia de audio
  sustituibles por catálogo;
- `assets/ayuda-llamamiento-bolsa.mp3`: guía local sin datos personales.

Cada fichero cumple el tope de 800 líneas de DEC-051.

## Rutas

### Aplicación normal

`/portal-empleado/`

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

Ya existe el adaptador HTTP estricto para `GET/HEAD` de esta ruta, pero aún no
está montado. Rechaza query, cuerpo, cookies, credenciales de proxy y cabeceras
heredadas de identidad o rol; tampoco interpreta `Authorization`. La frontera
preparadora deberá resolver del lado servidor actor, perfil, ámbito, motivo y
correlación antes de invocar el caso de uso. Por tanto, la ruta sigue fallando
cerrada hasta que se complete la vertical real.

### Presentación explícita para RRHH

`/portal-empleado/?presentacion=rrhh#portal`

Accesos directos útiles:

- cuadro de mando: `#bolsa/resumen`;
- elaboración: `#bolsa/elaboracion`;
- llamamientos: `#bolsa/llamamientos`.

Solo el parámetro exacto `presentacion=rrhh` importa dinámicamente
`web/static/portal-empleado/datos-presentacion.js`. La pantalla mantiene en
todo momento un aviso visible de datos sintéticos y ausencia de validez
administrativa. Ninguna acción de negocio envía, firma, registra ni persiste.

Además, el servidor exige la activación explícita
`VEC_RRHH_PRESENTATION_ENABLED=true`. La opción parte deshabilitada y, sin
ella, `datos-presentacion.js` responde `404` incluso aunque alguien conozca la
URL. La ruta normal y el resto de recursos definitivos permanecen disponibles.

### Bolsa pública

`/bolsa/`

La zona anónima conserva una navegación propia, pero no hereda enlaces del
Portal del Empleado ni del backoffice. El menú lateral solo permite acceder a
convocatorias, búsqueda, categorías profesionales y ayuda pública. En
pantallas estrechas se transforma en una cabecera compacta de dos columnas; no
desaparece. Esta separación evita presentar accesos internos sin sacrificar la
orientación ni la coherencia visual.

La consulta sigue identificada como demostración mientras su fuente no sea una
publicación oficial. El menú, el logotipo institucional, la adaptación móvil,
el alto contraste y la ayuda son componentes definitivos. El cambio se
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

| Elemento | Ubicación | Qué hace ahora | Sustitución obligatoria |
| --- | --- | --- | --- |
| Juego de datos sintéticos | `datos-presentacion.js` | Aporta bolsas, necesidades, expedientes, agregados y una propuesta sintética separada sin identidad ni contacto | Eliminar del despliegue productivo o mantener solo en artefacto de demostración; la aplicación normal consume la API interna |
| Contexto «María Pérez» | `datos-presentacion.js.sesion` | Permite revisar el encabezado con un perfil ficticio | Proyección del principal autenticado, obtenida en servidor y limitada a nombre, iniciales y perfil efectivo |
| Indicadores y gráficos | mismo adaptador | Facilitan validar composición y jerarquía visual | Agregados calculados en servidor con fecha de corte, ámbito y origen; nunca sumar expedientes ajenos en el navegador |
| Botones de alta, firma, envío y exportación | `portal-eventos.js`, controlador creado por inyección | Navegan o explican la limitación; no ejecutan negocio | Comandos API autenticados, autorizados por recurso, idempotentes, con control de versión y recibo de auditoría |
| Propuesta de llamamiento | función aislada `obtenerPropuestaPresentacion` | Permite revisar evaluaciones sintéticas sin realizar una petición | `POST /api/vec/bolsa/propuestas-llamamiento`, autorizado e idempotente; el servidor decide elegibilidad y prelación |
| Evaluaciones visibles | tabla de propuesta | Muestra secuencia, resultado, puntuación, regla y fundamento sintéticos | Proyección del comando específico sin nombre, documento, contacto ni identificador individual reutilizable |
| Configuración de bases/baremo | diálogo explicativo | Enseña las familias de configuración | Caso de uso de convocatoria gobernada y repositorio PostgreSQL, con versión, vigencia, fuente jurídica, firmas y publicación |
| Canales correo/Telegram | estado visible «no conectado» | Enseña el diseño previsto | Puertos de comunicación y adaptadores aprobados que devuelvan recibos verificables; ningún conector simulado concede efecto |
| Preferencias de texto y contraste | `localStorage` con prefijo `vec_portal_` | Guarda exclusivamente dos preferencias visuales | Puede permanecer; no contiene identidad, expediente, selección ni otro dato de negocio |
| Ayuda y audio | `ayuda-contenido.js` y MP3 local | Pasos, FAQ, audio y transcripción accesibles | Catálogo de ayuda versionado y conector de formatos/audio; la interfaz conserva el mismo contrato |

No hay `localStorage` de bolsas, llamamientos, candidatos o expedientes en esta
nueva superficie. Si en el futuro aparece uno, una prueba deberá impedir su
integración.

## Correspondencia con las fotografías

### Cuadro de mando (`image006.png`)

Se conservan:

- navegación funcional de los diez bloques;
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

## Cobertura funcional real a 17 de julio de 2026

| Área | Estado comprobado | No se debe afirmar todavía |
| --- | --- | --- |
| Consulta pública de convocatorias y categorías | Operativa extremo a extremo con fuente sintética y aviso DEMO en `/bolsa/` | Publicación oficial desde un expediente interno |
| Portal del Empleado antiguo | Carcasa, perfiles, menús y numerosas vistas reutilizables | Portal privado estable: `/api/vec/workspace` falla cerrado sin ámbito resuelto |
| Elaboración de bolsa histórica | Formulario visual heredado; dominio nuevo de convocatoria gobernada con organización/unidad inmutables, huellas canónicas y autorización ligada a la versión confirmada | Resolvedor autoritativo de ámbito desde expediente, persistencia, API, firma y publicación oficiales |
| Panel interno de Bolsa | Dominio, servicio de aplicación, contrato agregado sin datos personales, consulta PostgreSQL y pruebas de integración | Endpoint compuesto, identidad interna real, autoridad COSE de ejecución y productor de la proyección |
| Llamamientos | Dominio y caso de uso probados; el comando transporta la instantánea completa y un adaptador genera referencias opacas con 256 bits de aleatoriedad criptográfica; el esquema PostgreSQL V1 conserva auditoría y outbox | El adaptador PostgreSQL permanece cerrado hasta disponer de contrato SQL atómico V2, fuente autoritativa, motor publicado, autoridad COSE de efecto, API y confirmación/envío real |
| Integrantes/candidatos | Flujo propio del aspirante en API `fake`; datos de catálogo disponibles | Listado administrativo productivo y orden durable de bolsa |
| Autobaremación | Flujo de demostración heredado y núcleo nuevo de baremación avanzado | Reglas configurables de RRHH conectadas extremo a extremo |
| Revisión firmada de baremación | Dominio, aplicación y PostgreSQL avanzados | Composición en servidor, API e interfaz operativa |
| Documentos de aspirantes | Puertos S3, cuarentena y firma modelados | Subida y firma de Bolsa operativas; el endpoint actual responde `503` |
| Alegaciones | Lectura parcial en demostración | Presentación probatoria; el `POST` falla cerrado |
| Comunicaciones | Creación/listado parcial de avisos en modo `fake` | Entrega, acuse, Telegram, correo o notificación fehaciente conectados |
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

El cliente definitivo no contiene nombres, DNI, expedientes, categorías,
fechas ni cifras del juego de presentación. Una prueba automatizada impide
reintroducir los literales conocidos fuera del adaptador aislado.

## Mapa de sustitución por pantalla

| Pantalla | Base reutilizable existente | Corte real siguiente |
| --- | --- | --- |
| Portada | Registro de módulos y manifiestos VEC | Habilitación administrable por despliegue y rol, no lista fija del navegador |
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

## Criterio para retirar el adaptador de presentación

`datos-presentacion.js` se podrá retirar del artefacto productivo cuando:

1. `GET /api/vec/bolsa/panel` esté montado y probado de extremo a extremo con
   PostgreSQL e identidad real;
2. la sesión interna proyecte el perfil y ámbito efectivos;
3. elaboración y llamamientos dispongan de consultas reales;
4. las pruebas de seguridad verifiquen que otro ámbito no puede consultar la
   misma bolsa, expediente o persona;
5. las pruebas del navegador cubran carga, vacío, error, permiso denegado,
   paginación y datos reales;
6. Sistemas disponga de un artefacto separado si decide conservar una demo.

No basta con ocultar el parámetro de URL. El despliegue productivo deberá poder
excluir físicamente el adaptador sintético o bloquear su servicio mediante una
política de empaquetado probada.

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
