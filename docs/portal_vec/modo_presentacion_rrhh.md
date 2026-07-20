# Modo de presentación RRHH

## Decisión

La presentación del martes se entrega como un artefacto separado y desechable,
no como una rama provisional del servidor productivo. Las pantallas, sus
componentes accesibles y los contratos que consumen son parte del producto. Lo
intercambiable son los adaptadores que suministran datos o ejecutan órdenes.

Esto permite enseñar los recorridos completos sin fingir que existe una firma,
un registro o una notificación reales y sin obligar a reescribir la web si el
proyecto continúa.

| Elemento | Presentación | Producción |
| --- | --- | --- |
| Artefactos VEC | `runtime-presentacion`/`vec-presentacion` para el portal y `runtime-cartografia-presentacion`/`vec-cartografia-presentacion` para mediar las rutas | `runtime`/`vec-server`; la composición productiva completa sigue pendiente |
| Perfil | `presentacion_rrhh` | `produccion` |
| Datos privados | Prohibidos | Solo mediante conectores autorizados |
| Datos de la muestra | Metadatos BOP públicos reales; personas, expedientes, actos y resultados privados sintéticos y marcados | El adaptador DEMO se excluye físicamente de la imagen |
| Escrituras durables o de servidor | Ninguna; solo cambia memoria volátil de la pestaña | Casos de uso, autorización y recibos reales |
| API | Consulta pública local de Bolsa y una operación cartográfica exacta, sin identidad ni efectos administrativos | API pública/interna según frontera |
| Firma, registro, pagos y mensajes | Simulación visual | Adaptadores productivos aún por integrar |
| Cartografía | PBF, grafo OSRM y MBTiles reales, locales y versionados; expedientes e importes siguen siendo sintéticos | Conector cartográfico aprobado, versionado y sujeto a auditoría de Sistemas |
| Red saliente | Redes Docker segmentadas y sin salida a Internet; solo el mediador puede alcanzar OSRM en su red exclusiva | Según allowlist y decisión de Sistemas |

## Activación cerrada

La presentación parte deshabilitada. El servidor exige simultáneamente:

1. binario específico `cmd/vec-presentacion`;
2. perfil `VEC_EXECUTION_PROFILE=presentacion_rrhh`;
3. selector `VEC_RRHH_PRESENTATION_ENABLED=true`;
4. primera guarda literal
   `ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO`;
5. segunda guarda literal
   `CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA`;
6. autenticación deshabilitada, almacenamiento en memoria y catálogo personal
   ausente;
7. fuentes cuyos nombres terminen en `.demo.json`;
8. listener con IP literal loopback, privada o link-local y allowlist compuesta
   únicamente por redes locales enumeradas.

El mediador cartográfico se construye y ejecuta como proceso distinto. Repite
las guardas de presentación y, además, exige la URL exacta de OSRM, su ámbito,
la lista positiva de redes de destino y una versión gobernada del grafo. El
portal no conoce ni puede alcanzar la dirección de OSRM.

La comprobación se hace en configuración, bootstrap y servidor. No depende de
un `build tag`. Los binarios normales rechazan incluso un selector parcial de
presentación para evitar que una variable copiada por error exponga datos
sintéticos.

## Superficie servida

Las superficies usan listas positivas y solo los métodos enumerados:

- `/presentacion/`: selector de recorridos;
- `/bolsa/`: consulta pública;
- `/area-personal/`: punto de vista de la persona candidata;
- `/portal-empleado/`: punto de vista técnico de RRHH;
- `/api/publico/`: consulta pública local de convocatorias y categorías;
- `POST /api/presentacion/cartografia/rutas`: cálculo no autoritativo de una
  ruta de Dietas mediante el mediador aislado;
- `GET|HEAD /tiles/osm/{z}/{x}/{y}.png`: teselas OSM locales del mismo origen,
  limitadas a los niveles admitidos;
- `/healthz`, `/styles.css` y `/favicon.svg`.

No se sirven `/api/vec/`, `/api/demo`, `/candidates`, el árbol de datos, la
documentación del repositorio ni rutas estáticas generales. Las rutas no
canónicas y los escapes de directorio se rechazan. Todas las superficies
descartan en el borde cookies, `Authorization`, credenciales de proxy y
cabeceras de identidad ambiental; nunca las entregan a los servicios ni emiten
`Set-Cookie`. Conservan CSP, `nosniff`, `DENY`, política de referente y el resto
de cabeceras del servidor común.

## Arranque para la revisión

La presentación se ejecuta exclusivamente en Docker. El equipo anfitrión solo
necesita Docker Engine y Docker Compose v2; no se instala ni se ejecuta en él
Go, OSRM, TileServer GL, Playwright, Chromium ni un proxy auxiliar. El arranque
soportado es:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

Después se abre `http://127.0.0.1:8081/presentacion/`. El script construye la
composición, espera la salud de portal, proxy, mediador, OSRM y teselas, y
ejecuta la prueba rápida cartográfica dentro de un contenedor efímero. Fija las dos
guardas, memoria y las fuentes `.demo.json`; estas incorporan únicamente
metadatos públicos reales del BOP y estado privado sintético. No carga
credenciales.

El arranque y las fronteras HTTP pueden comprobarse en un proyecto Docker
efímero y autocontenido mediante el comando siguiente. Está pensado para CI o
un entorno limpio y no debe ejecutarse a la vez que la composición principal,
porque ambas reservan las mismas subredes privadas:

```bash
scripts/smoke_presentacion_rrhh.sh
```

Con la composición principal activa, la prueba rápida repetible es:

```bash
scripts/smoke_cartografia_presentacion.sh
```

Con la composición ya levantada, la revisión funcional y visual se ejecuta
también dentro de Docker:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

El manifiesto actual contiene 36 vistas y 25 estados de interacción en
escritorio, portátil y móvil. La ejecución cerrada el 20 de julio de 2026
obtuvo **183/183 escenarios correctos, 183 capturas y cero hallazgos**, incluidos
los cuatro puntos de vista DEMO, su selector, el cálculo de una ruta real de
Dietas y la carga de su mapa interno. La línea
base archivada de 159 escenarios es anterior a Cronos, Dietas, dos vistas
adicionales de Bolsa y la integración cartográfica. La herramienta exige la cabecera
técnica exclusiva de presentación, comprueba menús y recibos DEMO, y falla ante
almacenamiento del navegador, errores, recursos fallidos, controles sin nombre
o desbordamientos. El detalle y los informes generados se describen en
[captura y revisión de la presentación](revision_web_presentacion.md).

El contenedor aislado se arranca con:

```bash
docker compose --profile presentacion up --detach --build --wait
```

El proxy es el único servicio que publica un acceso, siempre en
`127.0.0.1:8081`. Portal, mediador, OSRM y renderizador de teselas no publican
puertos. Las redes `presentacion-portal`, `presentacion-cartografia` y
`presentacion-osrm` separan cada salto; la red de borde tiene el enmascaramiento
deshabilitado. Los contenedores usan sistema de ficheros de solo lectura,
límites de procesos/memoria/CPU, prohibición de nuevos privilegios y
capacidades Linux retiradas.

Cuando la revisión se sirve mediante un proxy corporativo situado en otro
equipo, no se amplía ese bind ni se usa `0.0.0.0`. El perfil opcional
`presentacion-remota` añade una entrada de borde independiente, ligada a una IP
interna concreta y con ACL de origen montada fuera del repositorio. El puerto
convencional es `18081`; el procedimiento, la configuración del frontal, las
pruebas y la retirada están en
[Acceso desde un proxy corporativo](acceso_proxy_presentacion.md).

La composición exige un grafo OSRM y una versión de teselas ya preparados. El
PBF fuente, su huella, la versión lógica del grafo y la versión activa del
MBTiles se gobiernan fuera del código; actualizar el dato no requiere recompilar
la web. La generación y activación se realizan con los scripts Docker de
`deploy/osrm-granada` y `deploy/osm-tiles-granada`. El procedimiento completo,
incluida la reversión, se describe en
[Cartografía interna de Dietas](cartografia_interna_dietas_2026-07-19.md).

Parada y retirada de contenedores:

```bash
docker compose --profile presentacion down
```

## Separación física de artefactos

El `Dockerfile` contiene tres destinos VEC separados:

- `runtime-presentacion`, que incluye el launcher, los adaptadores de
  presentación, los ficheros `.demo.json` y solo `vec-presentacion`; antes de
  empaquetar elimina pruebas, guías de integración, la SPA y módulos
  históricos, y no copia el directorio de configuración de trabajo;
- `runtime-cartografia-presentacion`, que contiene solo
  `vec-cartografia-presentacion`; no incorpora el portal, datos, documentación,
  fixtures, `vec-presentacion` ni `vec-server`;
- `runtime`, que instala exclusivamente las rutas enumeradas en
  `web/produccion.manifest`: no copia la SPA histórica de raíz (`index.html` y
  `app.js`), documentación, pruebas, fixtures, rutas de presentación o demo,
  `data/demo` ni `vec-presentacion`; solo incorpora `vec-server`.

La inspección física reproducible es:

```bash
scripts/verificar_contenido_artefactos_presentacion.sh
```

La prueba construye los tres destinos, exporta sus sistemas de ficheros y compara
el árbol web productivo con el manifiesto exacto. También falla ante una ruta
añadida aunque su nombre sea neutro, almacenamiento del navegador, firmas de
contenido de la SPA histórica, documentación, datos plausibles o cualquier
adaptador, launcher, dato o binario de presentación. El verificador unitario
del manifiesto se ejecuta con
`scripts/tests/test_verificar_web_produccion.sh`.

Las raíces exportadas —servidores y handlers embebibles— rechazan perfiles,
modos de autenticación o almacenamiento desconocidos sin reflejar su valor
ambiental. En perfil `produccion` fallan cerradas con
`ErrComposicionProductivaNoDisponible`: todavía no existen en esta composición
identidad ni repositorios autoritativos para el shell VEC, y las consultas
públicas aún dependen de adaptadores de fichero que validan fuentes DEMO.
Renombrar un JSON o seleccionar globalmente `local_durable` no elimina esas
limitaciones. El cierre alcanza también al listener público anónimo hasta que
reciba su repositorio público autoritativo; los perfiles cerrados de desarrollo
y presentación conservan sus guardas específicas.

Los handlers normal, público e interno rechazan cualquier parámetro de consulta
llamado `presentacion`, con independencia de su valor. Algunas ramas
condicionales de presentación aún permanecen en JavaScript compartido y se
retirarán vertical por vertical según el inventario; la imagen productiva no
incluye sus adaptadores, datos ni artefactos exclusivos y esas ramas no pueden
activarse por URL.

## Modelo de amenaza acotado

Este modo reduce, pero no convierte la muestra en producción:

- evita introducir datos personales: solo reutiliza metadatos públicos del BOP
  y fixtures sintéticos para identidad, expedientes y actuaciones;
- impide que un botón visual produzca un acto administrativo;
- evita que cookies o cabeceras del navegador se interpreten como identidad;
- impide publicar por accidente otros directorios del repositorio;
- impide seleccionar la muestra desde el binario normal;
- el portal no compone clientes para PostgreSQL, S3, Autofirma, registro,
  pasarela de pago o comunicaciones; el único cliente de red de la muestra es
  el mediador cartográfico, limitado por red y lista positiva al OSRM interno;
- el navegador obtiene rutas y teselas por el mismo proxy y nunca consulta
  servicios cartográficos públicos;
- muestra siempre avisos de “demostración” y “sin validez administrativa”.

No acredita autenticación, autorización nominal, persistencia, firma, registro,
notificación fehaciente ni cumplimiento productivo. Tampoco debe recibir una
copia de datos personales o expedientes reales “para que la demo parezca más
completa”; los datos públicos verificables se incorporan con su procedencia.

## Selector descartable de perfiles de presentación

El modo de presentación incorpora un selector accesible desde el bloque de
identidad de la cabecera. Es una ayuda **exclusiva de la demostración** para que
RRHH pueda recorrer cuatro puntos de vista sintéticos; no autentica, no concede
roles y no forma parte del futuro sistema productivo de identidad.

| Punto de vista DEMO | Superficie | Alcance visible |
| --- | --- | --- |
| Usuario externo | Área personal del aspirante | Procesos selectivos, candidaturas, documentos y datos propios |
| Funcionario | Portal del Empleado | Cronos y Dietas de autoservicio, sin gestión técnica de Bolsa |
| Técnico de RRHH | Portal interno | Revisión técnica de Bolsa expresamente enumerada |
| Administrador | Portal interno | Gobierno, configuración y operaciones de Bolsa expresamente enumeradas |

Las opciones no se acumulan: técnico y administrador no heredan Cronos o
Dietas, y funcionario no hereda la gestión de Bolsa. Cada cambio navega a una
URL canónica y recarga por completo la superficie, por lo que descarta el estado
volátil anterior. El selector no usa cookies, `localStorage`, `sessionStorage`,
IndexedDB ni Cache Storage. Un perfil ausente, repetido o desconocido falla
cerrado y nunca selecciona al administrador por defecto.

La implementación descartable vive en
`web/static/presentacion/selector-perfiles.js` y
`web/static/presentacion/selector-perfiles.css`. Los dos únicos puntos de carga
están protegidos por el modo de presentación en
`web/static/area-personal/arranque.js` y
`web/static/portal-empleado/portal.js`. El manifiesto productivo no incorpora
los dos archivos del selector y el verificador anti-fuga lo comprueba.

Si se retira la demostración, se eliminan esos dos archivos, sus dos importaciones
condicionales y los accesos sintéticos de `web/static/presentacion/index.html`.
Las pantallas, contratos, adaptadores productivos y controles reales de
autorización no se sustituyen ni se borran. En producción, identidad, perfiles y
capacidades los resolverá siempre el servidor; una URL nunca cambiará permisos.

## Sustitución incremental de adaptadores

Cuando se autorice continuar, no se cambia la web completa. Para cada
capacidad se sigue este orden:

1. cerrar el contrato de entrada, salida, errores y recibos de la capacidad;
2. implementar el adaptador real detrás del puerto de aplicación ya definido;
3. probar dominio, adaptador, autorización y vertical E2E con datos sintéticos;
4. habilitar la composición real solo en el perfil correspondiente;
5. hacer que la pantalla prefiera el contrato real y falle cerrada si no está
   disponible;
6. retirar de la presentación únicamente el adaptador sintético de esa
   capacidad cuando ya no sea necesario.

Así pueden cambiarse, de uno en uno, consulta pública, perfil, expedientes,
documentos, baremación, revisión, llamamientos, firma, registro, pagos y
comunicaciones. Nunca se conectará un adaptador de presentación a un puerto de
producción ni se migrarán sus estados: los actos simulados son descartables y
no autoritativos.

## Retirada completa

La decisión exacta por fichero y por vertical, con sustituto productivo,
condición de aceptación, prueba de ausencia y marcha atrás, se encuentra en el
[inventario de retirada incremental](inventario_retirada_presentacion_2026-07-19.md).

Si no se aprueba el proyecto, basta con retirar el perfil Compose
`presentacion`, los destinos `runtime-presentacion` y
`runtime-cartografia-presentacion`, `cmd/vec-presentacion`,
`cmd/vec-cartografia-presentacion`, `web/static/presentacion`, los
`datos-presentacion.js`, los ficheros `.demo.json` exclusivos y los scripts y
configuraciones de la muestra cartográfica. El destino normal sin material DEMO
no depende de ellos, aunque todavía no está autorizado para producción.

Si se aprueba, el artefacto puede mantenerse solo para formación y pruebas
visuales, siempre en red local, con metadatos públicos verificables y datos
privados sintéticos, fuera del inventario de servicios productivos. Cada
publicación debe volver a ejecutar las pruebas de contenido y seguridad.

`skill_ref: admin-data-web`
