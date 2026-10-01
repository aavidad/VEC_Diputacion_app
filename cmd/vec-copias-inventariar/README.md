# Inventario local antes de preparar una copia

La CLI CS02 compara un descriptor de release con un inventario declarado y comprueba
los bytes de sus archivos bajo una raíz local. Detecta archivos ausentes, tamaños o
huellas distintos, migraciones omitidas y diferencias de esquema.

Desde la raíz del repositorio puede probarse con los ejemplos sintéticos:

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-copias-inventariar \
  -descriptor cmd/vec-copias-inventariar/testdata/descriptor.json \
  -observado cmd/vec-copias-inventariar/testdata/observado.json \
  -raiz cmd/vec-copias-inventariar/testdata/instalacion
```

Los archivos de ese ejemplo son texto sintético; no contienen un binario VEC ni una
observación real de PostgreSQL. La salida JSON identifica su alcance como
`inventario_offline_declarado` y la procedencia PostgreSQL como
`declaracion_offline_no_autenticada`. `autoriza_copia` y `autoriza_restauracion`
siempre son `false`.

Para un inventario autorizado, se aportan tres entradas:

- `-descriptor`: JSON con `inventario` previsto y `rutas`, cada una con `id` lógico
  y `ruta_relativa`. Debe describir todos los binarios, activos web, catálogos,
  material, configuración y ficheros previstos. Cada artefacto declara SHA256 y tamaño.
- `-observado`: JSON del tipo `copias.Inventario`, obtenido por el procedimiento
  autorizado que observe versiones y módulos instalados. Un archivo SQL presente o
  el mayor número de migración no demuestra instalación.
- `-raiz`: directorio local autorizado que contiene los archivos declarados. Las
  rutas se resuelven dentro de esa raíz; no se admiten rutas absolutas ni `..`.

El descriptor completo y el observado se mantienen separados. Una ruta desconocida,
repetida o ausente impide comprobar el conjunto. Los JSON rechazan campos desconocidos,
claves repetidas y contenido adicional, y tienen un límite de 4 MiB.

La CLI escribe razones con clave, valores comparados y acción requerida. Para binarios
y activos públicos muestra las huellas exactas; para material, configuración y ficheros
privados contrasta las huellas del inventario entero. No imprime contenido, rutas
locales ni huellas aisladas de secretos. La salida sigue siendo JSON aunque
fallen la lectura o los argumentos. Los códigos de salida son:

| Código | Resultado |
| --- | --- |
| 0 | Coinciden los bytes y las versiones declaradas. |
| 1 | Hay diferencias o faltan datos/archivos para comprobarlas. |
| 2 | No se pueden leer o interpretar las entradas o los argumentos son incorrectos. |
| 4 | No se puede escribir la salida. |

Esta pieza no autentica descriptores, no demuestra la relación entre commit y binario,
no consulta PostgreSQL ni compara roles, ACL o contenido de tablas. Tampoco crea o
restaura copias. Esas comprobaciones pertenecen a las siguientes piezas del plan.
Una coincidencia offline no sustituye la restauración aislada exigida para cada copia.
