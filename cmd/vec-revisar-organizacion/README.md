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

## Obtener el paquete revisado

Añada `--preparar` para incluir `paquete` en la salida cuando supera la revisión.
Puede combinarlo con `--idioma` en cualquier orden:

```sh
go run ./cmd/vec-revisar-organizacion --preparar < cmd/vec-revisar-organizacion/ejemplo.sintetico.json > revision.json
go run ./cmd/vec-revisar-organizacion --preparar --idioma en < cmd/vec-revisar-organizacion/ejemplo.sintetico.json > revision.en.json
```

Compruebe primero el estado del proceso. Extraiga el paquete con `jq`, si está
instalado, y vuelva a revisarlo con el mismo comando:

```sh
jq -cje '.paquete' revision.json > paquete.json
go run ./cmd/vec-revisar-organizacion < paquete.json
```

El paquete conserva manifiesto, hechos y decisiones, en el orden usado para
calcular `informe.paquete_huella_sha256`. La huella corresponde a la serialización
compacta de ese objeto mediante `json.Marshal`, no a los bytes del archivo
indentado ni al informe traducido. Extraerlo y repetir la revisión conserva la
misma huella. Un paquete rechazado no aparece en la salida; el informe sitúa
el fallo y el comando termina con estado `1`.

El paquete usa el contrato de preparación de Personal existente y no contiene
actor, concesión ni recibo. El circuito autorizado debe resolver la identidad,
verificar catálogos y fuente, conciliar y aprobar antes de publicar con Personal
000011. Esta opción tampoco conecta ese circuito ni acredita su instalación.
Las decisiones pendientes pueden conservarse como declaraciones para revisión;
no permiten publicar.

La exportación comprueba también que el JSON compacto normalizado no supera
8 MiB. Algunos caracteres, como `<`, se escapan al normalizar y aumentan su
tamaño. Si se supera el límite, el informe devuelve `paquete_excede_limite`,
sin paquete ni huellas. La salida indica el tamaño real en `bytes_paquete` y
el máximo en `limite_bytes_paquete`, sin devolver el material recibido.
El proceso termina con estado `1`. La revisión sin
`--preparar` conserva su comportamiento. Extraiga el paquete en formato
compacto, sin salto final, como en el comando `jq -cje` anterior.
