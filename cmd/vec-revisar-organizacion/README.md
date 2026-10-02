# Revisar un paquete de Organización

El comando lee de la entrada estándar el paquete de preparación existente:
`{ "manifiesto": { ... }, "hechos": [ ... ] }`. Admite `decisiones` opcionales
con el contrato de conciliación existente. No necesita servicios ni credenciales.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-revisar-organizacion < cmd/vec-revisar-organizacion/ejemplo.sintetico.json
go run ./cmd/vec-revisar-organizacion --idioma en < cmd/vec-revisar-organizacion/ejemplo.sintetico.json
```

El ejemplo contiene una unidad ficticia y referencias sintéticas. Use solo
material sintético en desarrollo. Conserve los originales de RRHH fuera de Git.

La salida JSON contiene `informe` y los `mensajes` del catálogo del idioma:

- `valido` indica si el paquete supera esta revisión local.
- `clave_error`, `seccion` y `fila_fallida` sitúan el primer fallo. Las filas
  empiezan en 1 dentro de `hechos` o `decisiones`.
- `hechos` y `decisiones` cuentan las entradas recibidas. Los recuentos por clase
  y resultado incluyen las filas válidas anteriores al fallo.
- `manifiesto_huella_sha256` usa la huella del contrato existente.
- `paquete_huella_sha256` identifica el contenido normalizado: los hechos se
  ordenan por clase e identidad y las decisiones por clase y fila de fuente.
  Cambiar el orden de las filas o el idioma no cambia esa huella.
- `pendientes_publicacion` conserva campos que faltan y comprobaciones que
  requieren el circuito autorizado, aunque el paquete resulte válido.

Las huellas solo aparecen cuando todo el paquete supera la revisión.
El límite es 8 MiB, 3.000 hechos y 1.000 decisiones. Se rechazan claves
repetidas o desconocidas, valores nulos, datos ajenos al contrato, UTF-8 inválido
y varios documentos JSON concatenados.

Estado del proceso: `0` para un paquete válido en esta revisión; `1` para
entrada o paquete rechazados; `2` si no se puede cargar el catálogo o escribir
el informe. `go run` comunica un estado de error propio al fallar el comando.

Esta comprobación solo trabaja en memoria. No acredita autenticidad, permisos,
auditoría, aprobación ni persistencia. Las referencias y fechas aportadas son
declaraciones; la publicación sigue pendiente del circuito existente.
