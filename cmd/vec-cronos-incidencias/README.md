# Incidencias de registro por periodo

Este ensayo permite consultar los registros sintéticos de una persona durante un periodo.
Lee una sola entrada JSON por stdin y devuelve JSON. No consulta ni modifica una base de datos.

El periodo incluye `desde` y excluye `hasta`, en la zona declarada (Madrid en este ejemplo).
El corte UTC es exclusivo: los hechos de ese instante no forman parte del snapshot.
El resultado conserva el corte UTC, la fuente, sus versiones y las huellas de los archivos de ejemplo.
La huella identifica bytes: no acredita autenticidad, autorización ni firma.

La cobertura y la secuencia se muestran por separado. Una fecha con cobertura completa
puede tener registros, una secuencia incompleta o ningún registro. La ausencia de registros
no acredita una ausencia del empleado. Si falta cobertura en cualquier fecha evaluable o información anterior necesaria,
la secuencia de todas las fechas evaluables queda indeterminada y el recuento es `null`. Las fechas posteriores al corte
quedan sin evaluar. Una entrada abierta no se extiende más allá del corte.

La cobertura declarada se refiere a los datos disponibles hasta el corte. La consulta no
decide absentismo, incumplimiento, permisos, jornada prevista, presencia física ni correcciones.
No sustituye las consultas de movimientos y permisos concedidos existentes.

## Ejecutar el ejemplo

Desde la raíz de este worktree, con Go y jq ya instalados:

```sh
directorio_binario=$(mktemp -d)
binario=$directorio_binario/vec-cronos-incidencias
trap 'unlink "$binario"; rmdir "$directorio_binario"' EXIT
go build -o "$binario" ./cmd/vec-cronos-incidencias
idioma=es
snapshot=data/demo/cronos/incidencias-periodo.json
textos=web/static/textos/$idioma/cronos-incidencias-ensayo.json
snapshot_sha=$(sha256sum "$snapshot" | cut -d' ' -f1)
textos_sha=$(sha256sum "$textos" | cut -d' ' -f1)
jq -n --rawfile snapshot "$snapshot" --rawfile textos "$textos" \
  --arg idioma "$idioma" --arg snapshot_sha "$snapshot_sha" --arg textos_sha "$textos_sha" \
  '{snapshot:$snapshot,textos:$textos,idioma:$idioma,snapshot_sha256:$snapshot_sha,textos_sha256:$textos_sha}' \
  | "$binario"
```

Cambie `idioma=es` por `idioma=en` para consultar el mismo ejemplo con textos ingleses.
El ejemplo incluye un día con entrada y salida, una salida sin entrada, un día sin registros,
una entrada abierta y dos fechas posteriores al corte.

La envoltura contiene los archivos como cadenas para conservar sus bytes y huellas exactos.
Se rechazan argumentos, claves duplicadas o desconocidas, versiones no admitidas,
UTF-8 inválido, huellas incorrectas y entradas superiores a 1 MiB.
El periodo admite hasta 366 fechas y el snapshot hasta 10.000 hechos.
Las referencias del ensayo empiezan por `demo:`.
Los instantes deben ser UTC, con precisión máxima de microsegundos.
Dos hechos con la misma referencia o instante se rechazan; el ensayo no decide su orden.
Se rechazan hechos desde el fin exclusivo del periodo, incluso si son anteriores al corte.
Un error de entrada, lectura o salida termina con código 2. El diagnóstico contiene un código
estable cuando el propio catálogo de textos no se puede leer.
El proceso termina con código 2 al alcanzar diez segundos, también si stdin o stdout
están bloqueados. Ctrl-C conserva su comportamiento normal.

## Alcance comprobable

Esta pieza prepara una consulta y una CLI de ensayo. No está conectada al portal, a permisos,
auditoría ni persistencia. No cierra C10 ni acredita el enclave interno de Cronos.
Solo admite ejemplos sintéticos y carece de conectores de red.
La consulta futura necesita una autorización positiva de la persona y el periodo exactos,
la fuente interna autorizada y la auditoría central de la lectura.
