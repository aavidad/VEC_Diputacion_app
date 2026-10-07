# Ensayo de calendario histórico de Cronos

Este comando consulta el calendario de centro que corresponde a Lucía Moreno
Salas, una persona ficticia, durante un periodo con cambio de adscripción.
Muestra las fechas de cada tramo, el centro, las versiones de calendario y las
carencias que impiden determinar algún tramo.

Desde la raíz del repositorio, con la versión de Go indicada en `go.mod`:

```sh
go run ./cmd/vec-cronos-calendario < cmd/vec-cronos-calendario/testdata/cambio-centro.json
```

Recibe un único JSON por la entrada estándar, sin argumentos. Escribe el
resultado JSON en la salida estándar y los errores en la salida de diagnóstico.
El código de salida es `0` cuando produce un resultado, incluso si hay tramos
indeterminados; es `1` cuando la entrada es inválida o no puede reproducir la
consulta. Devuelve `2` si no puede cargar los textos o escribir el resultado o el diagnóstico.
No escribe archivos ni requiere servicios.

## Casos disponibles

| Archivo en `testdata/` | Resultado esperado |
| --- | --- |
| `cambio-centro.json` | Dos tramos determinados. Cambio de Granada a Motril el 5 de junio; cada tramo aplica el festivo local sintético de su centro. |
| `hueco.json` | Falta la adscripción del 4 de junio. Ese tramo queda indeterminado. |
| `solape.json` | Dos adscripciones cubren el 5 de junio. Ese tramo queda indeterminado. |
| `falta-calendario.json` | Falta la capa local de Motril. El segundo tramo queda indeterminado y enumera el ámbito ausente. |
| `cruce-anio.json` | Divide el periodo entre 2026 y 2027 y usa las versiones del año correspondiente. |
| `conocimiento-corregido.json` | Un segundo snapshot, conocido el 1 de octubre, usa la versión 2 del calendario local de Granada; cambia el día señalado del 4 al 3 de junio. |

Son datos ficticios, incluidos los días señalados y sus fuentes. No representan
festivos oficiales, una adscripción de Personal ni una regla aprobada por RRHH.
Para comparar los dos momentos de conocimiento, repite el comando con
`cambio-centro.json` y con `conocimiento-corregido.json`. Cada archivo produce
una salida idéntica al repetirlo.

## Entrada y trazabilidad

El periodo y las adscripciones usan intervalos `[desde, hasta)`: incluyen la
primera fecha y excluyen la última. El snapshot y las versiones preseleccionadas
deben declarar exactamente el mismo `conocido_en` que la solicitud. Este
instante es obligatorio; no se reemplaza por la hora actual.

El caso de uso consume el puerto existente `ConsultaCalendarios`. La CLI
conecta `Calendarios.application.Servicio.CalendarioCentro`, que combina las
capas nacional, autonómica, local y del centro. Su repositorio de ensayo acepta
solo las versiones preseleccionadas para el instante indicado; no busca una
última versión ni implementa historia bitemporal.

La salida conserva:

- referencias y versiones de todas las adscripciones candidatas de cada tramo;
- versiones de Calendarios con la procedencia sintética declarada;
- referencias y SHA-256 de fuente, aportadas como metadatos del ensayo;
- SHA-256 de los bytes de entrada, del snapshot normalizado, de cada versión
  con sus días y del resultado anual de Calendarios;
- las fechas del calendario recortadas al tramo y los ámbitos sin cobertura.

La huella de entrada cambia si cambian sus bytes, incluso los espacios. La
huella del snapshot ordena las adscripciones por referencia y expresa el
instante en UTC. Las demás huellas usan la representación JSON de los DTO de
Go, sin añadir fechas de ejecución. Estas huellas permiten comparar este
ensayo; no son firmas ni acreditan custodia de los originales.

Las referencias son opacas. El adaptador no abre archivos, URLs ni documentos
referenciados. Los SHA-256 de fuente son declaraciones del fixture: no hay
archivo oficial ni verificación documental. La entrada admite hasta 2 MiB,
256 adscripciones, 128 versiones y un periodo de hasta siete cambios de año,
entre 2000 y 2100. Rechaza campos desconocidos, claves duplicadas y más de una
versión del mismo ámbito y año.

Los textos legibles están en
`web/static/textos/<idioma>/cronos-calendario-ensayo.json`. Los idiomas y el
predeterminado proceden de `web/static/textos/idiomas.json`. Si omites `idioma`,
el comando usa ese predeterminado. Para usar inglés, cambia `idioma` a `en` en
una copia del JSON.

## Qué significa el resultado

`estado_determinacion` describe exclusivamente la cobertura del calendario de
centro: `determinado` si todos los tramos tienen una adscripción única y las
cuatro capas; `parcial` si solo algunos la tienen; `indeterminado` si ninguno.
No se elige la primera adscripción cuando hay varias.

`laborable` pertenece al calendario de centro. No acredita apertura efectiva
del servicio ni que la persona tuviera trabajo programado. `persona_programada`
y `minutos_teoricos` permanecen en `null`; faltan perfil horario y cuadrante.
No calcula jornada, fichajes, presencia, saldo, permisos ni vencimientos.

Es una demostración local y sintética, señalada con `demostracion: true`.
No activa el módulo, rutas HTTP, manifiestos ni la composición del portal.
No demuestra autorización, auditoría durable, persistencia, segregación del
enclave de Cronos ni uso con datos reales. Esas puertas siguen pendientes.

## Comprobación focal

```sh
go test -p 32 ./cmd/vec-cronos-calendario/... ./internal/modules/cronos/application/... ./internal/modules/cronos/ports/...
go test -race -p 32 ./cmd/vec-cronos-calendario/... ./internal/modules/cronos/application/... ./internal/modules/cronos/ports/...
go vet -p 32 ./cmd/vec-cronos-calendario/... ./internal/modules/cronos/application/... ./internal/modules/cronos/ports/...
```

Las pruebas recorren los seis fixtures con el servicio real de Calendarios y
comprueban repetibilidad, cobertura, fronteras, ausencia de doble conteo y
rechazo de entradas ambiguas. Son comprobaciones CLI y de aplicación; no son
un recorrido del portal ni una instalación productiva.
