# Preparar una propuesta de efectos de permisos C3

La muestra `data/demo/reglas/cronos-efectos-permisos.configurable.json` contiene tres reglas editables y avisos en los catálogos de español e inglés. Son valores de ejemplo pendientes de revisión y aprobación por RRHH. Cada regla se refiere a un permiso C6, una versión de ese permiso, un colectivo y una vigencia. El colectivo aplicable a una persona y una fecha debe llegar de Personal.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-cronos-preparar-catalogo data/demo/reglas/cronos-efectos-permisos.configurable.json
```

La salida JSON devuelve la referencia y versión de la política, el número de reglas y el SHA-256 de los bytes exactos del archivo. `estado: propuesta_sin_aprobar` indica que la validación local no publica el catálogo, no activa efectos horarios ni concede a nadie permiso para adoptarlo. Una edición, incluso de espacios, cambia la huella y exige nueva revisión de la versión.

El validador rechaza claves duplicadas o desconocidas, referencias y fechas inválidas, más de 100 reglas y reglas duplicadas para el mismo permiso, versión C6 y colectivo. La autoridad común debe acreditar publicación, aprobación separada y vínculo con la huella; Cronos debe recibir las fuentes verificadas y confirmar la adopción con autorización nominal, comparación de la versión anterior, auditoría y recibo durable. La instalación de ese tramo sigue pendiente.
