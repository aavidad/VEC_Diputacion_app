# Reglas y circuito de firma: ensayo local del 12 de octubre

Candidata operativa `fe57878adf433fe07c107447d3f361aeacd248fd`, después de
`438bacab7523c8bf985f69ee146736c4201830f6`. Revisión independiente SQL,
seguridad y usabilidad estática GO, P0/P1/P2=0; el segundo dictamen revisó
únicamente el parche del lector. La integración posterior de main conserva
estos cambios y añade sólo producto ya integrado.

Las fuentes vigentes privadas se copiaron en versiones nuevas: CT v2 con 36
entradas, Bolsa v6 con 50 y circuito v2 con 11. Comparación estructural:
mismas claves, valores, plazos y referencias; los nombres y descripciones
visibles están limpios. El origen `configuracion` no atribuye procedencia
normativa ni una aprobación de RRHH. El circuito conserva la firma pendiente.

PostgreSQL 18.4, contenedor propio `--rm --restart=no --memory 2g`, datos en
disco y copia de la base final de K, apagada. Base CT191 inicial v1/secuencia1,
huella `5530995222b852169848d7729a024cdc30b631fbde48608b1329e517f1d50572`.
La copia original no tenía ajustes. Para comprobar su conservación se añadió
solo en el clon un ajuste v1, `c04.plazo_subsanacion`, cantidad10→8, con su
fila de cambio y evento. No se atribuye a esos datos una decisión V3.

El preparador CT190 real validó 36 entradas y emitió canónico v2 con huella
`6ecf24d17b889c8bbd4400580977ac8b1c2c863c824f09ac5498d70a3d4a7141`.
`--comprobar` devolvió `preparado_con_rebase`, secuencia1/ajustes1 mediante
READ ONLY, sin crear archivos. `--aplicar` confirmó base v2/secuencia2 y creó
ajustes v2/base2 con los mismos valores, canónico y SHA que v1. La operación
registra recibo, instalador y referencia operativa; no simula una decisión
V3. El reintento devuelve `replay`, sin otro evento ni versión.

El primer GET de ajustes dio503: el repositorio exigía al menos un cambio en
cada versión histórica, mientras que el traslado de base no cambia ningún
valor. El parche permite una lista vacía únicamente con el marcador, base y
referencia específicos de esta operación. Una versión normal vacía sigue
rechazada. La prueba cubrió el traslado y nueve casos negativos. La fixture
v1 se completó por INSERT en el clon; el código no normalizó ni reescribió
ninguna fila anterior.

Chrome del sistema, mediante playwright-cli y mTLS nominal del material
local: `/portal-empleado/reglas/` muestra 86 reglas; origen configurable sin
pastilla. Capturas [es1440](reglas-es-1440.png) y [en390](reglas-en-390.png):
sin desborde horizontal ni marcas visibles. Las 19 traducciones cambiadas y
los textos de pantalla usan el catálogo del idioma activo; los datos
anteriores conservan sus etiquetas y atributo de idioma.

GET `/api/vec/reglas/vigentes`:200. GET `/api/vec/contratacion-temporal/reglas/ajustes?limite=20`:
200, versión2, base2, cantidad8, ambos recibos originales y cambio10→8 en v1.
GET `/api/vec/contratacion-temporal/circuito-firma`:200, catálogo2, 11 pasos,
Portafirmas desconectado y firmas pendientes. El GET de ajustes final dio
108,6ms de servidor en una muestra; no es una medición p95 de gran volumen.

Tras reiniciar aplicación y PostgreSQL, la misma consulta de ajustes dio200
y conservó valores, versiones y recibos. La recuperación del guion devolvió
replay/secuencia2/ajustes2. El recibo del rebase es
`recibo:8dfc2c90-a7bf-44cc-a6c8-92a4e070fe06`; v1 conserva
`recibo:2f851d5b-405c-4c91-80a0-9366fc8b6bae`. No duplicados.

Pruebas focales: Go de bootstrap/reglas/preparador/guion; 66 Node de los
recorridos y27 de caché; nueve negativos y lectura V3 del lector de ajustes;
bash/shellcheck, gopls/diff-check. Semgrep local sin métricas:0 hallazgos.
Gosec de los tres paquetes cambiados:0 avisos en líneas propias;87 avisos y
20 errores de carga de los paquetes anteriores, conservados en salida
privada. La puerta completa corresponde a la CI de la PR.

Límite de presentación: el historial por regla filtra por cambios de valor,
por lo que el traslado v2 consta en API/base pero no añade una línea a ese
historial. El delta de dirección añade Configuradas al resumen y al filtro; los cuatro
indicadores abren su filtro y conservan el origen en la URL. No se afirma validación global WCAG ni recorrido
completo de otros módulos.

Configuración exacta y orden operativo en LEEME y lista SQL. Dirección
fusiona y ejecuta en cidonia; este ensayo no tocó ese servidor.

## Corrección solicitada por dirección

El envoltorio de `psql` puede ejecutarse dentro de un contenedor, sin acceso
a la ruta del guion en el host. El rebase se entrega ahora por entrada
estándar. Una prueba focal con un envoltorio recibió los 6.105 bytes de SQL
y no recibió `-f`. El cuerpo SQL no cambia; no se repitió el ensayo de datos.
Dirección también lo ensayó en su copia HZ21 con un ajuste previo: preparado,
registrado y replay, con v1 conservada.

La primera CI detectó tres fallos web: metadatos EN sin sus pares ES y dos
expectativas anteriores al catálogo y al grafo de caché nuevos. Se corrigió
la paridad de claves y se contrastaron los originales con las fuentes
conocidas, sin retirar las guardas. Pasaron 44 pruebas Node focales y la
comprobación de tamaños; portal-i18n.test.mjs queda en 1.102 líneas.

Chrome del sistema comprobó exclusivamente el delta visual en un arnés
local, con los 86 datos del catálogo y la respuesta AJ2 registrada. No es
otro ensayo SQL ni de autorización. En escritorio: 86 = 27 del Reglamento
+ 0 propuestas + 59 configuradas. Pulsar Configuradas abre 59 filas y
conserva `origen=configuracion` en la URL; al abrir esa URL se restaura el
filtro. A 390 px no hay desborde horizontal y sólo se pide el catálogo
`textos/en/reglas.json` para la pantalla inglesa. Capturas del delta:
[es1440](configuradas-es-1440.png) y [en390](configuradas-en-390.png).

El LEEME usa el montaje final `/vec-incorporacion/` y aclara que el guion
comprueba esta publicación inicial: un ajuste posterior real hace que
`--comprobar` devuelva `ajustes_incompatibles`, sin revertir ese ajuste.
