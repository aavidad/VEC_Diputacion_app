# O4-05: integración web de la decisión de cobertura

**Fecha:** 26 de julio de 2026
**Estado:** preparado; bloqueado hasta el cierre verificable de O4-04E

## Resultado buscado

Conectar la pantalla comparativa de cobertura ya existente en el portal
interno de RRHH con el caso de uso y la persistencia productivos:

`navegador RRHH → API interna → aplicación O4-03 → sesión O4-04E → PostgreSQL → recibo`

La misma aplicación y los mismos contratos neutrales servirán a web,
escritorio, CLI y MCP. La interfaz web no aportará autoridad, no conservará
estado de negocio y no usará cookies ni almacenamiento del navegador.

## Inventario reutilizable

| Pieza | Estado | Decisión |
| --- | --- | --- |
| Propuesta, decisión y rectificación O4-03 | Real | Reutilizar los servicios de aplicación. |
| Adaptador PostgreSQL O4-04E | Real en Go | Usar sólo después del cierre SQL y su revisión independiente. |
| Pantalla RRHH de expedientes y cobertura | Reutilizable | Mantener contrato, presentador, vista, componentes, CSS e i18n. |
| Datos y adaptador de presentación | DEMO aislada | Conservar únicamente bajo el modo explícito de presentación; nunca incluir en manifiestos productivos. |
| API interna de cobertura | No existe | Crear un adaptador de entrada neutral y cerrado. |
| Composición interna de Contratación temporal | No existe | Crear una raíz productiva separada del bootstrap DEMO. |
| Capturas actuales | DEMO | Crear un E2E y un capturador productivos distintos. |

No se reescribirá la interfaz. Se sustituirá su fuente sintética por un
cliente HTTP productivo que proyecte las respuestas gobernadas sobre el
contrato visual existente.

## Condiciones de seguridad y arquitectura

- Arquitectura hexagonal: HTTP y PostgreSQL son adaptadores sustituibles.
- La intención del cliente contiene sólo expediente, versión esperada,
  idempotencia, identidad semántica, vía y motivo.
- Identidad, perfil, organización y permisos proceden de la frontera interna
  confiable y nunca de cuerpo, consulta, cabeceras libres o almacenamiento
  web.
- Lista positiva de rutas, campos, métodos, tamaños y tipos.
- Rechazo de `Cookie`, `Set-Cookie`, credenciales en URL y autoridad declarada
  por el cliente.
- La raíz de composición acepta y normaliza `Forwarded`/`X-Forwarded-*` solo
  desde el proxy corporativo autenticado. Antes de entregar la petición al
  adaptador elimina esas cabeceras, `Via`, `Connection`, `Keep-Alive`,
  `Upgrade` y cada cabecera nominada dinámicamente por `Connection`. El
  adaptador las rechaza siempre.
- Un resultado ambiguo se reconcilia contra el primario; nunca autoriza un
  reintento ciego.
- Las claves funcionales se localizan mediante i18n o catálogo gobernado. La
  web no inventa etiquetas.
- Contratación temporal consume capacidades de Bolsa por sus puertos; ni la
  API ni la vista consultan tablas de Bolsa.

## Secuencia de commits verificables

### 1. Contrato neutral y adaptador HTTP

Crear los manejadores internos de consulta de propuesta y
decisión/rectificación:

- DTO cerrados, límites y errores localizables;
- proyección minimizada de vía recomendada, evaluaciones, identidad semántica
  y recibo;
- pruebas de forma, autoridad, cancelación y cabeceras prohibidas.

### 2. Composición interna productiva

Construir una raíz de composición de Contratación temporal que:

- ensamble O4-03 con la sesión y el reconciliador O4-04E;
- registre únicamente las rutas internas permitidas;
- falle cerrada sin identidad corporativa, PDP/VEC, reloj, PostgreSQL,
  catálogo, gobierno o lectura de análisis;
- incorpore la política de proxy confiable anterior, sin trasladar cabeceras
  de reenvío a los manejadores;
- no reutilice la composición DEMO de Bolsa o del portal.

### 3. Cliente web y proyección RRHH

Añadir una fuente HTTP productiva y conectarla al coordinador de módulos:

- reutilizar contrato, presentador, vista, componentes y estilos existentes;
- mantener el adaptador sintético sólo para el modo explícito de presentación;
- incorporar el cliente productivo a los manifiestos;
- añadir las claves i18n de estados y errores sin fijar etiquetas de catálogo.

### 4. Recibo y recuperación

Cerrar la conducta ante pérdida de respuesta:

- recibo válido prevalece;
- sin recibo no se repite la escritura;
- consulta mínima protegida de estado/recibo para recuperar tras un refresco,
  sin `localStorage`;
- pruebas de replay, CAS, recibo cruzado o adulterado, denegación y
  reconciliación primaria.

### 5. E2E, accesibilidad y revisión visual

Probar sobre PostgreSQL 18 efímero:

- navegador → API → aplicación → O4-04E → recibo;
- concurrencia y pérdida de respuesta;
- teclado, lector, foco, estados de carga y errores;
- capturas de la pantalla comparativa en tres resoluciones mediante un runner
  distinto del DEMO;
- acta de alcance y aceptación de RRHH.

## Bloqueos explícitos

1. O4-04E debe quedar publicado, probado en PostgreSQL 18.4 y con `GO`
   independiente.
2. Sistemas debe proporcionar la frontera que convierta mTLS y Kerberos en un
   contexto interno confiable. No se sustituirá por cookies, `Authorization`
   del navegador ni cabeceras `X-*` aceptadas libremente.
3. Debe aprobarse la lectura mínima protegida de estado/recibo necesaria para
   sobrevivir a una recarga sin persistencia del navegador.
4. Las vías y evaluaciones necesitan claves i18n o catálogo gobernado; los
   textos del adaptador DEMO no son autoridad productiva.

## Estimación

Después del cierre de O4-04E y de resolver identidad y recuperación:

- pantalla comparativa conectada: 2–3 días efectivos;
- O4-05 completo, incluido E2E durable y revisión: 7–11 días efectivos.

La estimación no convierte las dependencias de Sistemas en datos ficticios ni
autoriza a degradar el modelo de seguridad para acelerar una demostración.
