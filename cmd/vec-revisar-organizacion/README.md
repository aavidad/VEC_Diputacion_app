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

`informe.cobertura_conciliacion` permite localizar las decisiones que faltan.
Contiene una entrada por hecho, con su identidad, clase y fila de origen. En
`clase_decision` aparece la clase esperada: un `nodo` necesita una decisión de
`unidad`; las otras clases necesitan una decisión de su misma clase.

El resultado de cada hecho es `sin_decision`, `pendiente`, `descartada` o
`vinculada`. `recuentos_hechos` cuenta estos resultados. La clase y la fila
se comprueban juntas: una decisión de puesto no cubre la unidad que procede
de la misma fila. Las decisiones de `clasificacion` figuran en
`decisiones_adicionales` y no sustituyen la decisión de ningún hecho.

`completa` solo indica que todos los hechos tienen una decisión declarada
`vinculada` y que las decisiones adicionales también están vinculadas.
Una decisión pendiente o descartada conserva la cobertura incompleta.
La revisión sigue en `preparacion_no_autoritativa`, con acreditación de fuente,
conciliación autorizada, aprobación y publicación pendientes. El indicador
no certifica el contenido de la fuente ni las correspondencias declaradas.

La cobertura usa el orden del paquete normalizado y conserva el mismo
contenido al cambiar el orden de entrada o el idioma. Aparece tanto en la
revisión como con `--preparar`; el paquete exportado y sus huellas conservan
su formato. Se omite si la entrada o el paquete resultan rechazados, incluido
el rechazo por tamaño durante la exportación.

## Comprobar que la preparación está completa

Use `--comprobar-completo` para comprobar los campos obligatorios del manifiesto
y las decisiones declaradas de todos los hechos. El comando devuelve
`comprobacion_completitud.completa` y una lista de `faltantes` con `clave`,
`esperado` y `actual`. Estos son códigos traducidos en `mensajes`; la lista no
incluye filas, identificadores ni el contenido recibido.

```sh
go run ./cmd/vec-revisar-organizacion --comprobar-completo < cmd/vec-revisar-organizacion/ejemplo.sintetico.json
```

El estado de salida es `0` si esta comprobación local está completa y `1` si
falta algo o el paquete es inválido. Se puede añadir `--preparar` para obtener
el paquete normalizado solo cuando esté completo. `--idioma` cambia los mensajes,
sin cambiar los códigos ni las huellas. Las opciones se admiten una vez cada una
y en cualquier orden.

«Completa» describe exclusivamente el material preparatorio. El informe mantiene
`preparacion_no_autoritativa` y enumera las verificaciones institucionales
pendientes. No acredita fuente, catálogos, acto, conciliación duradera,
aprobación ni publicación. Personal 000011 sigue denegando la publicación.

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
