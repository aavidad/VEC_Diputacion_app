# Revisar una política de grado

Este catálogo prepara la revisión documental de Carrera. El ejemplo es
sintético y se puede cambiar o retirar. Sus fechas y referencias no son una
regla de la Diputación. No contiene umbrales ni fórmulas de reconocimiento.

La versión del catálogo y la versión de su fuente se conservan por separado.
Cada vía, condición de periodo, límite y evidencia declara su referencia,
fuente y versión. Las colecciones vacías siguen pendientes; VEC no las completa.
La vigencia tiene fecha inicial y puede dejar abierta la fecha final.

Para revisar el ejemplo junto con la preparación existente:

```sh
go run ./cmd/vec-carrera-preparar --politica-grado-sintetica \
  data/catalogos/carrera/politica_grado_ejemplo.json \
  < cmd/vec-carrera-preparar/testdata/entrada.json
```

La salida conserva la política aportada, sus comprobaciones y pendientes,
junto con la preparación del expediente. Una referencia de aprobación solo
queda anotada: no acredita aprobación competente. La fuente admitida, esa
aprobación y el circuito autorizado permanecen pendientes en todos los casos.
La preparación mantiene separados nivel del puesto y grado personal.

El comando no calcula periodos computables ni reconoce derechos. No modifica
Personal, genera resoluciones, firma documentos o inscribe actos. Para usar una
política real se necesitan la fuente provincial aprobada, la evidencia de su
aprobación y el acceso autorizado; este modo solo acepta datos sintéticos.
