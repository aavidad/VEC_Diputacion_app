# Informe de permisos de Cronos: ejemplo local

Desde la raíz del repositorio, genere el PDF con datos sintéticos:

```sh
go run -p 8 ./cmd/vec-cronos-informe-permisos \
  web/static/textos/es/cronos-informe-permisos.json \
  cmd/vec-cronos-informe-permisos/testdata/ejemplo.json > /tmp/cronos-permisos-ejemplo.pdf
```

El archivo de ejemplo declara todos los campos. Para probar un informe más
breve, copie ese JSON y cambie solo `campos_permitidos` a esta lista:

```json
["etiqueta", "unidad", "concedido"]
```

Ejecute la misma orden con la ruta de la copia como segundo argumento.

Para abrir el mismo ejemplo en una hoja de cálculo, genere el CSV:

```sh
go run -p 8 ./cmd/vec-cronos-informe-permisos --formato=csv \
  web/static/textos/es/cronos-informe-permisos-csv.json \
  cmd/vec-cronos-informe-permisos/testdata/ejemplo.json > /tmp/cronos-permisos-ejemplo.csv
```

El CSV incluye una fila de contexto con persona sintética, ejercicio, fecha
de corte y aviso de ejemplo, aunque no haya permisos. Solo aparecen las
columnas elegidas en `campos_permitidos`. Las cantidades de días son enteros
de días; las de horas
son enteros de minutos y la columna de unidad lo indica. `null` se escribe
«No disponible» y cero se conserva como `0`. No se suman unidades distintas.
Las celdas de texto que podrían interpretarse como fórmulas llevan un
apóstrofo inicial.

`campos_permitidos` es obligatorio y no tiene valor por defecto. Admite una
lista no vacía de los siete campos conocidos, sin repetidos. Si incluye
`pendiente_resolver`, `concedido` o `restante`, debe incluir también `unidad`.
Una cantidad `null` incluida se muestra como dato no disponible; un campo
excluido no aparece en el PDF ni en el CSV.

La herramienta exige `demo: true` y solo produce un ejemplo local. No consulta
permisos de personas empleadas, no confirma una exportación y no habilita una
descarga en el portal. Los tipos y campos que podrán exportarse siguen pendientes
de RRHH y del DPD en la [pregunta 118](../../dudas.md).
