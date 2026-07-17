# Entregable urgente para RRHH: Portal del Empleado y Gestión de Bolsas

## Decisión y alcance

La interfaz se construye como superficie definitiva del Portal del Empleado,
reutilizando la carcasa y el lenguaje visual VEC ya incorporados al repositorio.
No se crea un segundo producto ni se copia la aplicación antigua a otra base de
código.

El alcance visible pedido por RRHH es:

1. portada con varios módulos del Portal del Empleado;
2. solo `Bolsas de trabajo` habilitado en la fase inicial;
3. cuadro de mando de Bolsa basado en `fotos/image006.png`;
4. elaboración y gestión de bolsas;
5. asistente de llamamientos basado en `fotos/image004.png`;
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
- `portal-eventos.js`: interacción sin decisiones de negocio;
- `datos-presentacion.js`: único adaptador sintético, cargado solo de forma
  explícita.

Cada fichero cumple el tope de 800 líneas de DEC-051.

## Rutas

### Aplicación normal

`/portal-empleado/`

- intenta cargar únicamente `GET /api/vec/bolsa/panel`;
- usa credenciales de la sesión interna del mismo origen;
- rechaza una respuesta marcada como `demostracion`;
- ante `401`, `403`, `404`, `501` o error de red muestra acceso cerrado;
- no sustituye el error por datos locales;
- no guarda datos de negocio en el navegador.

La API aún no está montada. Por tanto, esta ruta falla cerrada de forma
deliberada hasta que se complete la vertical real.

### Presentación explícita para RRHH

`/portal-empleado/?presentacion=rrhh#portal`

Accesos directos útiles:

- cuadro de mando: `#bolsa/resumen`;
- elaboración: `#bolsa/elaboracion`;
- llamamientos: `#bolsa/llamamientos`.

Solo el parámetro exacto `presentacion=rrhh` importa dinámicamente
`web/static/portal-empleado/datos-presentacion.js`. La pantalla mantiene en
todo momento un aviso visible de datos sintéticos y ausencia de validez
administrativa. Ninguna acción envía, firma, registra ni persiste.

Además, el servidor exige la activación explícita
`VEC_RRHH_PRESENTATION_ENABLED=true`. La opción parte deshabilitada y, sin
ella, `datos-presentacion.js` responde `404` incluso aunque alguien conozca la
URL. La ruta normal y el resto de recursos definitivos permanecen disponibles.

## Inventario exhaustivo de elementos temporales

| Elemento | Ubicación | Qué hace ahora | Sustitución obligatoria |
| --- | --- | --- | --- |
| Juego de datos sintéticos | `datos-presentacion.js` | Aporta bolsas, candidatos enmascarados, expedientes, próximos llamamientos y actividad | Eliminar del despliegue productivo o mantener solo en artefacto de demostración; la aplicación normal consume la API interna |
| Contexto «María Pérez» | `datos-presentacion.js.sesion` | Permite revisar el encabezado con un perfil ficticio | Proyección del principal autenticado, obtenida en servidor y limitada a nombre, iniciales y perfil efectivo |
| Indicadores y gráficos | mismo adaptador | Facilitan validar composición y jerarquía visual | Agregados calculados en servidor con fecha de corte, ámbito y origen; nunca sumar expedientes ajenos en el navegador |
| Botones de alta, firma, envío y exportación | `portal-eventos.js`, controlador creado por inyección | Navegan o explican la limitación; no ejecutan negocio | Comandos API autenticados, autorizados por recurso, idempotentes, con control de versión y recibo de auditoría |
| Selección de candidatos | estado efímero de `portal.js` | Comprueba el asistente de cuatro pasos | Propuesta del caso de uso de llamamientos, revalidada dentro de la transacción; el navegador nunca decide elegibilidad |
| Filtros de candidatos | `portal.js` | Filtran únicamente el conjunto sintético ya cargado | Consulta paginada y autorizada; los parámetros se validan y los resultados se minimizan en servidor |
| Configuración de bases/baremo | diálogo explicativo | Enseña las familias de configuración | Caso de uso de convocatoria gobernada y repositorio PostgreSQL, con versión, vigencia, fuente jurídica, firmas y publicación |
| Canales correo/Telegram | estado visible «no conectado» | Enseña el diseño previsto | Puertos de comunicación y adaptadores aprobados que devuelvan recibos verificables; ningún conector simulado concede efecto |
| Preferencias de texto y contraste | `localStorage` con prefijo `vec_portal_` | Guarda exclusivamente dos preferencias visuales | Puede permanecer; no contiene identidad, expediente, selección ni otro dato de negocio |

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
- selección de bolsa;
- resumen lateral;
- distribución de candidatos por estado;
- filtros por estado, puntuación y disponibilidad;
- orden de prelación;
- tabla de candidatos y selección;
- configuración de condiciones, plazo y canales;
- revisión final.

La versión final no aceptará como decisión la selección calculada en
JavaScript. El servidor propondrá y revalidará las personas elegibles con la
versión exacta de bolsa y reglas, autorización vigente y trazabilidad atómica.

## Cobertura funcional real a 17 de julio de 2026

| Área | Estado comprobado | No se debe afirmar todavía |
| --- | --- | --- |
| Consulta pública de convocatorias y categorías | Operativa extremo a extremo con fuente sintética y aviso DEMO en `/bolsa/` | Publicación oficial desde un expediente interno |
| Portal del Empleado antiguo | Carcasa, perfiles, menús y numerosas vistas reutilizables | Portal privado estable: `/api/vec/workspace` falla cerrado sin ámbito resuelto |
| Elaboración de bolsa histórica | Formulario visual que guardaba en `localStorage`; dominio nuevo de convocatoria gobernada | Persistencia, API, firma y publicación oficiales |
| Llamamientos | Dominio y caso de uso avanzados, con orden, elegibilidad, autorización y pruebas | PostgreSQL, API, interfaz conectada y envío real |
| Integrantes/candidatos | Flujo propio del aspirante en API `fake`; datos de catálogo disponibles | Listado administrativo productivo y orden durable de bolsa |
| Autobaremación | Flujo de demostración heredado y núcleo nuevo de baremación avanzado | Reglas configurables de RRHH conectadas extremo a extremo |
| Revisión firmada de baremación | Dominio, aplicación y PostgreSQL avanzados | Composición en servidor, API e interfaz operativa |
| Documentos de aspirantes | Puertos S3, cuarentena y firma modelados | Subida y firma de Bolsa operativas; el endpoint actual responde `503` |
| Alegaciones | Lectura parcial en demostración | Presentación probatoria; el `POST` falla cerrado |
| Comunicaciones | Creación/listado parcial de avisos en modo `fake` | Entrega, acuse, Telegram, correo o notificación fehaciente conectados |
| Auditoría | Auditoría heredada parcial y registro probatorio en baremación | Trazabilidad unificada de elaboración, llamamientos y comunicaciones |

## Contrato de lectura definitivo previsto

La pantalla normal espera una proyección interna con esquema
`vec.bolsa.panel.v1`. La respuesta debe contener:

- `sesion`: nombre visible, iniciales y perfil ya resuelto;
- `indicadores`, `estados_candidatos`, `distribucion_global` y `series`:
  agregados ya calculados y limitados al ámbito autorizado;
- `avisos`: proyección mínima de avisos que la sesión puede consultar;
- `configuracion_llamamiento` y `catalogos_llamamiento`: valores gobernados por
  la versión de bases y catálogos, nunca listas decisorias incrustadas en el
  cliente;
- `bolsas`: solo bolsas accesibles al ámbito del técnico;
- `candidatos`: proyección mínima y enmascarada, solo cuando la operación y el
  expediente concreto lo permitan;
- `elaboraciones`: expedientes de convocatoria asignados;
- `proximos`: llamamientos programados dentro del ámbito;
- `actividad`: eventos funcionales minimizados que el rol pueda consultar;
- `contratos`, `reglas`, `documentos` y `canales`: proyecciones autorizadas de
  las bandejas auxiliares;
- `auditoria`: referencia opaca de la proyección trazable.

El servidor debe devolver `demostracion: false`. La interfaz rechaza de forma
explícita una fuente que intente mezclar datos de demostración en la ruta
normal.

Antes de estabilizar el contrato se añadirán:

- paginación y límites máximos;
- filtros tipados;
- referencias opacas, no claves de base de datos;
- fecha de corte y zona horaria;
- versión/ETag de cada agregado mutable;
- capacidades de acción calculadas en servidor;
- política de minimización por rol y finalidad.

El cliente definitivo no contiene nombres, DNI, expedientes, categorías,
fechas ni cifras del juego de presentación. Una prueba automatizada impide
reintroducir los literales conocidos fuera del adaptador aislado.

## Mapa de sustitución por pantalla

| Pantalla | Base reutilizable existente | Corte real siguiente |
| --- | --- | --- |
| Portada | Registro de módulos y manifiestos VEC | Habilitación administrable por despliegue y rol, no lista fija del navegador |
| Cuadro de mando | Catálogo profesional y consulta pública | Proyección interna paginada y autorizada |
| Elaboración | `domain/convocatoria_gobernada*.go` | Repositorio PostgreSQL, aplicación, HTTP y composición |
| Llamamientos | `application/llamamientos.go` y sus pruebas | Adaptador PostgreSQL, fachada HTTP interna y composición |
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

## Criterio para retirar el adaptador de presentación

`datos-presentacion.js` se podrá retirar del artefacto productivo cuando:

1. `GET /api/vec/bolsa/panel` esté compuesto y probado con PostgreSQL;
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

1. Proyección de lectura real del cuadro de mando.
2. Persistencia y API de elaboración de convocatorias.
3. Persistencia y API de llamamientos, empezando por propuesta de solo lectura.
4. Confirmación atómica del llamamiento y trazabilidad.
5. Respuestas, renuncias, contratos, ceses y reincorporaciones.
6. Documentos, firmas, publicación y comunicaciones fehacientes.
7. Estadísticas y explotación con anonimización aprobada.

La línea de confianza/atestación no prioritaria quedó preservada, sin activarse,
en el commit `fd6e0ef` y en
`docs/portal_vec/relevo_confianza_atestacion_v2_2026-07-17.md`.
