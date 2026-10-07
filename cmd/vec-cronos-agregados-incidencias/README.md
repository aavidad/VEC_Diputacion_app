# Ensayo de incidencias por periodo

La CLI cuenta incidencias explícitas de una instantánea sintética y devuelve un
agregado por código y estado. Incluye personas, días y unidades persona-día
distintos afectados, además de la cobertura del periodo. Aporta una pieza
reutilizable de C10/C11 para RRHH; el acceso real y los informes siguen pendientes.

Desde la raíz del repositorio, compile y ejecute con los archivos revisados:

```sh
task_bin_dir=$(mktemp -d)
trap 'rm -rf "$task_bin_dir"' EXIT
GOCACHE=/dev/shm/go-build GOPROXY=off go build -o "$task_bin_dir/agregados" ./cmd/vec-cronos-agregados-incidencias
"$task_bin_dir/agregados" \
  --snapshot data/demo/cronos/agregados-incidencias-periodo.json \
  --snapshot-sha256 5478724dba05a98ec0325a71e5189d95672b0c65e78f937cf633a099eeb40d1b \
  --textos web/static/textos/es/cronos-agregados-incidencias-ensayo.json \
  --textos-sha256 fb54703202e3571b27971f9834b9b8d1981a764baf68d97ddc2753384173698c \
  --idioma es
```

Para inglés, use `web/static/textos/en/cronos-agregados-incidencias-ensayo.json`,
huella `2cb71b94be419794383acaf4e54823a840b2592aef9765c6506b8314c132f03b`
y `--idioma en`. El idioma debe coincidir con el catálogo cargado. Para recibir
la instantánea por stdin, use `--snapshot -` y conserve su huella exacta.
La lectura de stdin espera EOF; el proceso termina con código 2 a los diez segundos
si una lectura o escritura sigue bloqueada. Ctrl+C conserva su comportamiento habitual.

El ejemplo abarca los días 28 y 29 de septiembre de 2026, en Europe/Madrid.
Devuelve este fragmento:

```json
{
  "estado": "incompleto",
  "personas_seleccionadas": 3,
  "dias_periodo": 2,
  "personas_dias_esperados": 6,
  "personas_dias_completos": 5,
  "personas_dias_incompletos": 1,
  "personas_dias_desconocidos": 0,
  "incidencias_observadas": 3,
  "personas_afectadas_observadas": 2,
  "dias_afectados_observados": 2,
  "personas_dias_afectados_observados": 2,
  "total_incidencias": null
}
```

Hay dos incidencias pendientes y una resuelta. Dos corresponden a la misma
persona y día: aumentan el número de incidencias, pero cuentan una sola unidad
persona-día afectada. Los estados proceden de hechos declarados; el ensayo
no modifica ni resuelve incidencias.

## Contrato de la instantánea

- `demo: true`, esquema 1 y referencias `demo:`. La selección contiene de 1 a
  100 personas únicas; una lista vacía, duplicada o con comodines se rechaza.
  La selección delimita datos sintéticos y no concede permisos.
- `desde` es inclusiva y `hasta_exclusiva` queda fuera. Se admiten hasta 366
  días completos, con zona horaria explícita; `Local` se rechaza. El corte UTC
  debe alcanzar el final del periodo. Cada incidencia tiene fecha incluida y
  un instante de registro UTC entre el inicio de esa fecha y el corte, ambos
  incluidos, con precisión máxima de microsegundos.
- El catálogo cerrado admite únicamente `registro_incompleto`, con estados
  `pendiente` y `resuelta`, en ese orden. Este código representa un hecho
  declarado; aquí no se deriva de fichajes, minutos o saldos. La huella del
  catálogo usa su referencia, versión decimal, código y dos estados, cada uno
  seguido por LF. Cambiar estos valores cambia la huella.
- Cada fuente tiene referencia, versión, huella y estado `disponible` o
  `no_disponible`. Se admiten hasta cuatro fuentes, todas necesarias para
  completar la cobertura. Sus huellas son metadatos declarados: no se consultan
  ni verifican archivos externos.
- La cobertura declara `completa`, `incompleta` o `desconocida` para cada
  persona, fecha y fuente. Falta de fila o fuente no disponible produce cobertura
  desconocida. Las filas duplicadas, ajenas a la selección o con otra versión
  se rechazan. Los recuentos de cobertura cuentan unidades persona-día, aunque
  haya varias fuentes.
- Cada incidencia tiene referencia única, versión, persona, fecha, código,
  estado, fuente y versión de fuente. Una segunda fila con la misma referencia
  se rechaza: la instantánea contiene estados al corte, sin historial de eventos.
  Los hechos ya declarados siguen observados aunque su fuente figure caída;
  esos hechos no completan la cobertura.
- Se admiten hasta 10.000 incidencias y 10.000 combinaciones persona-día-fuente.
  Los archivos son regulares, de hasta 1 MiB; se rechazan enlaces en el fichero
  final, FIFO, directorios, UTF-8 inválido, campos desconocidos, claves duplicadas,
  tipos incorrectos y JSON adicional. Stdin tiene el mismo límite de bytes.

`total_incidencias` solo contiene un número cuando todas las unidades tienen
cobertura completa. En ese caso, una lista explícita vacía produce cero. Con
cobertura incompleta o desconocida, el total y los totales por grupo son `null`;
los números observados describen únicamente los hechos recibidos. Un fallo de
lectura o validación no produce un agregado: termina con código 2 y este
diagnóstico técnico, sin rutas ni datos de entrada:

```json
{"error":"entrada_invalida","estado":"desconocido","total_incidencias":null}
```

El diagnóstico requiere que stderr admita escritura. El límite de tiempo puede
terminar el proceso sin diagnóstico. Un fallo del escritor puede dejar una salida
parcial; el consumidor debe aceptar resultados solo con salida completa y código 0.

## Límites y siguiente dependencia

La salida conserva periodo, corte, zona, versiones y huellas de instantánea,
catálogo, fuentes y textos. No enumera personas ni incidencias individuales.
Los agregados de grupos pequeños pueden permitir identificar personas: este
ensayo no acredita anonimización ni fija un umbral de privacidad para uso real.

No se evalúan fichajes ni se calculan ausencias, porcentajes de absentismo,
causas de salud o datos sindicales. `Absentismo` en `ConsultaMovimientos`
conserva su significado de permiso concedido; este agregado no lo reinterpreta
ni sustituye la consulta de presencia de PR #287.

Los textos están en catálogos por idioma. No hay HTTP, SQL, red, documentos,
autoridad activa, datos reales ni cambios en la composición del portal. Para
consumir una fuente real hacen falta relaciones vigentes de Personal, concesión
nominal con finalidad, campos y ámbito exactos, auditoría de lectura y denegación,
y el enclave Cronos admitido. Las huellas identifican bytes y configuración;
no son autorización, firma ni evidencia de instalación o publicación.
