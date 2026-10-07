# Recorrido preparado del historial de Cronos

El guion consulta permisos propios en un servidor autorizado y recorre su
historial con teclado. Comprueba páginas anterior/siguiente, filtro de concedidos,
nombres accesibles calculados por Chrome y foco visible sin obstrucción.
Visita español e inglés a 1440 y 390 px, y a 1440 px con zoom nativo al 200 %.
Reutiliza la auditoría, zoom y guardas del runner F, sin modificar Cronos.

```sh
node scripts/recorridos/cronos_historial/recorrer.mjs --plan
```

El plan no abre navegador, lee certificados ni contacta con VEC. La ejecución
requiere un servidor interno instalado y autorizado con datos sintéticos,
certificado cliente sintético existente, TLS de confianza y concesión nominal de
consulta de permisos propios. No genera certificados ni publica permisos.
El backend deriva la identidad de mTLS y decide la autorización, también para
las lecturas existentes del shell indicadas abajo. El guion no las concede.

Dirección prepara un JSON privado fuera de Git con esta forma:

```json
{
  "version": 1,
  "sintetico": true,
  "entorno_controlado": true,
  "origen": "https://127.0.0.1:PUERTO_INTERNO",
  "commit_servido": "HASH_COMPLETO_VERIFICADO_POR_DIRECCION",
  "anio": 2026,
  "identidad": {
    "certificado": "/RUTA_PRIVADA/empleado-sintetico.crt",
    "clave": "/RUTA_PRIVADA/empleado-sintetico.key"
  }
}
```

El año debe coincidir con el que consulta inicialmente la página, normalmente el
año actual de Madrid. Los datos propios deben contener más de veinte solicitudes
y al menos una concedida. Cada solicitud debe tener una proyección visible
distinguible (nombre, fecha, duración y estado). Veinte es el tamaño de página vigente de la interfaz,
no un límite de solicitudes ni una regla de permisos. El guion no crea esos datos.

Configuración, certificado y clave: archivos propios, regulares, sin enlaces,
privados (0600) y fuera de Git. La salida debe ser nueva y su carpeta padre propia
y privada (0700). Se mantiene la verificación TLS y el certificado cliente se
aplica únicamente al origen HTTPS numérico de loopback indicado.

```sh
export VEC_PLAYWRIGHT_MODULE=/RUTA_PLAYWRIGHT_EXISTENTE/index.mjs
node scripts/recorridos/cronos_historial/recorrer.mjs \
  --config /RUTA_PRIVADA/cronos.json --salida /RUTA_PRIVADA/SALIDA_NUEVA
```

Usa Playwright local y `/usr/bin/google-chrome`. Solo permite GET del mismo
origen. En Cronos, solo admite `/api/interna/cronos/permisos/propio` con la consulta
exacta `?anio=...`. El resto de API se deniega salvo estas seis lecturas existentes,
sin parámetros, verificadas en el arranque de la fuente vigente:

| Ruta GET exacta | Consumidor que la pide al abrir el portal |
| --- | --- |
| `/api/vec/modules` | `portal-catalogo-modulos.js`, catálogo de módulos |
| `/api/vec/session` | `portal-catalogo-modulos.js`, sesión visible |
| `/api/vec/usuarios/mis-preferencias` | `portal.js` → carga de `portal-preferencias.js` |
| `/api/vec/usuarios/mi-imagen` | `portal-preferencias.js` → `cargarCorreos`, imagen de cabecera |
| `/api/vec/usuarios/mis-correos` | `portal-preferencias.js` → `cargarCorreos` |
| `/api/vec/bolsa/bolsas` | `portal.js` → `cargarFuenteDatos`, sondeo inicial de Bolsa |

Solo admite recursos estáticos en `/portal-empleado/`, `/comun/`, `/textos/`,
`/assets/` y `/locales/`. Rechaza rutas con codificación porcentual o dobles barras,
otras operaciones, redirecciones, Set-Cookie y WebSocket. No envía perfil,
actor, permiso ni cabeceras de identidad elegidos por el operador.

Valida la respuesta con `validarPermisosPropiosCronos` importado desde la URL del
módulo que ya cargó la página, conservando su versión. Una respuesta 200 sin ese
módulo o con datos incompatibles corta el recorrido. El cambio de páginas y filtro
debe permanecer local: exige una sola consulta de permisos por visita.
Compara las referencias exactas y su orden al avanzar, volver y filtrar, sin
guardarlas en el informe. Como la vista no expone referencias, usa su renderer
real para vincular cada proyección visible a una única referencia del DTO.
Si dos referencias resultan indistinguibles, corta con `escenario_insuficiente`;
no atribuye esa ambigüedad a un fallo del servidor ni añade identificadores a la UI.
No activa Solicitar ni otras acciones de negocio. Las lecturas pueden añadir
auditoría de acceso en el servidor.

El perfil temporal de ajustes de Chrome se elimina al cerrar. El informe solo
conserva contadores, estados, versión del módulo y commit declarado; omite
certificados, rutas, URL, personas, cuerpos y textos. Verifica métricas del zoom
nativo para distinguirlo de CSS o emulación. Código 0: comprobaciones terminadas;
1: configuración inválida o recorrido cortado.

Son comprobaciones parciales: no prueban contraste, lector de pantalla, orden
lógico, controles con flechas ajenos al filtro ni `summary`. No acreditan
autenticación, concesiones, persistencia, reinicio ni cumplimiento WCAG global.
Una lectura HTTP autorizada y las evidencias de Dirección se valoran juntas;
el guion no certifica por sí solo esa frontera.

Para ensayar la herramienta sin servidor, use aislamiento de red y scratch propio:

```sh
node --test scripts/recorridos/cronos_historial/recorrer.test.mjs
VEC_F_CRONOS_SALIDA=/RUTA_PRIVADA/NUEVO_ENSAYO \
  node scripts/recorridos/cronos_historial/recorrer.test.mjs --ensayar-chrome
```

El ensayo monta la vista y cliente reales sobre HTML mínimo y 41 solicitudes sintéticas del
catálogo de casos. Playwright entrega bytes directamente, sin servidor ni conexión
HTTP. Su espía de navegador comprueba las opciones mTLS que produciría el runner y
retira solo las rutas ficticias antes de crear el contexto offline. No usa PKI,
no ensaya TLS ni autentica una persona. Esta instrumentación está únicamente en
la prueba: la CLI real conserva certificado, clave y validación TLS obligatorios.
El informe lo distingue como `prueba_guion` y mantiene las acreditaciones en falso.
Además rechaza un mutante que conserva las veinte filas de la primera página
al avanzar a la segunda: el mismo recuento no basta para dar la página por válida.
