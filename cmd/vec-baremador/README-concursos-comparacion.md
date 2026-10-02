# Comparar configuraciones de Concursos

La [CLI existente](README.md) admite `--modo concursos --comparar-reglas`.
Recibe dos configuraciones sintéticas de Provisión, sin entrada personal ni
cálculo de puntuaciones. Comprueba sus contratos y las huellas SHA256 de los
archivos antes de compararlas.

Desde la raíz del repositorio:

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-baremador \
  --modo concursos --comparar-reglas \
  --reglas cmd/vec-baremador/testdata/comparacion_concursos_v1.json \
  --reglas-sha256 94e648c0c171ced6746ac34d671e07c273343c009e970b4b75c84c8f778bcb2b \
  --reglas-nuevas cmd/vec-baremador/testdata/comparacion_concursos_v2.json \
  --reglas-nuevas-sha256 205e24752921500f989c7ace139c615f677bff3a721517519fef345117cc245a
```

El ejemplo devuelve diez cambios: bases, tope total, coeficiente y tope de
antigüedad, conversión temporal, coeficiente de un tramo de grado y altas/bajas
por cambios de ID de otro tramo y de la regla de titulaciones. Los valores son
hipotéticos y no representan bases oficiales.

La salida usa `vec.provision.diferencia_configuracion.v1` y el alcance
`comparacion_reglas`. La cabecera conserva la convocatoria, las versiones y las
huellas canónicas de las configuraciones. Estas huellas corresponden al mismo
material canónico que utiliza el cálculo existente; pueden diferir de las
huellas de archivo si cambia la disposición del JSON.

Cada cambio identifica su ámbito, regla, tramo, campo y tipo (`alta`, `baja` o
`modificacion`). Conserva los valores anterior y nuevo mediante una variante
tipada: texto, entero, booleano, puntos, fecha, racional, tipos, conversión,
regla o tramo. Los puntos son cadenas de micropuntos exactos. El cero y `false`
se conservan; un lado ausente se representa como `null`.

El comparador contempla fechas, bases, tope total y todos los campos de reglas
y tramos del contrato actual. Identifica reglas y tramos por ID; renombrar un
ID produce baja y alta. Una regla añadida o retirada incluye sus tramos sin
repetirlos como cambios independientes. Reordenar reglas, tramos o tipos
mantiene la misma semántica. Los cambios se ordenan para reproducir la salida.

Ambas configuraciones deben pertenecer a la misma convocatoria. Sus versiones
son cadenas opacas: `revision:zeta` y `revision:alfa` no tienen orden numérico
ni legal. Comparar material idéntico devuelve `cambios: []`; usar el mismo token
de versión con material distinto se rechaza. El informe no acredita sucesión
administrativa, aprobación, firma, publicación ni auditoría institucional.

Cada archivo admite hasta 2 MiB. `--limite-bytes` puede reducir ese límite.
Se rechazan contratos incorrectos, claves desconocidas o repetidas, documentos
concatenados, configuraciones inválidas y huellas incorrectas. No se admiten
`--entrada`, `--entrada-sha256`, `--ejemplo` ni `--listar-ejemplos` durante la
comparación. Un fallo devuelve código 2 y un diagnóstico JSON sin rutas ni
contenido; una comparación correcta devuelve 0.

Las pruebas focales cubren los cambios, la reproducción y los rechazos:

```sh
GOCACHE=/dev/shm/go-build go test -race -p 8 \
  ./internal/modules/provision/domain ./cmd/vec-baremador
```

Este corte trabaja con archivos locales sintéticos y no guarda cambios ni
conecta con Personal, PostgreSQL o servicios externos.
