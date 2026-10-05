# Informe CSV de fichajes de ejemplo

Esta herramienta local convierte el mismo ejemplo sintético del informe PDF en una tabla CSV. El archivo empieza con una fila de contexto que indica el periodo, la zona horaria, si la fuente se declara completa y que el resultado carece de validez administrativa. Las filas siguientes enumeran los fichajes aportados, ordenados por instante. La hora incluye el desfase UTC para distinguir las dos horas iguales del cambio de horario.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-cronos-informe-movimientos-csv \
  web/static/textos/es/cronos-informe-movimientos-csv.json \
  cmd/vec-cronos-informe-saldo/testdata/movimientos.json > /tmp/cronos-movimientos-ejemplo.csv
```

Para obtener los rótulos ingleses, use `web/static/textos/en/cronos-informe-movimientos-csv.json`. El ejemplo no consulta el registro real. La exportación de datos de empleados requiere autorización nominal de descarga y auditoría duradera; esta herramienta no implementa esas capacidades.
