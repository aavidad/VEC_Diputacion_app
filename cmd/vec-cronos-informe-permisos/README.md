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

`campos_permitidos` es obligatorio y no tiene valor por defecto. Admite una
lista no vacía de los siete campos conocidos, sin repetidos. Si incluye
`pendiente_resolver`, `concedido` o `restante`, debe incluir también `unidad`.
Una cantidad `null` incluida se muestra como dato no disponible; un campo
excluido no aparece en el PDF.

La herramienta exige `demo: true` y solo produce un ejemplo local. No consulta
permisos de personas empleadas, no confirma una exportación y no habilita una
descarga en el portal. Los tipos y campos que podrán exportarse siguen pendientes
de RRHH y del DPD en la [pregunta 118](../../dudas.md).
