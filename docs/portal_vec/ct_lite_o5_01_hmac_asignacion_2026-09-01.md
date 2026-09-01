# CT-LITE-O5-01-HMAC-A — sellos HMAC de asignación

Fecha: 1 de septiembre de 2026.

Estado del productor: candidato local de un único commit, pendiente de
revisión independiente del hash exacto. Este corte no concede `GO`, no cierra
`O5-01` y no autoriza integración, publicación ni producción.

## Preflight acreditado

- Worktree exclusivo:
  `.worktrees/ct-o5-01-hmac-a-20260901`.
- Rama nueva: `trabajo/ct-o5-01-hmac-a-20260901`.
- Base y `HEAD` exactos antes de editar:
  `c8cc4312b063ddca7294dfe27dc673ad1a4676d0`.
- La rama local de producto
  `integracion/ct-producto-ligero-20260821`, su seguimiento local y
  `origin/integracion/ct-producto-ligero-20260821` coincidían exactamente en
  esa misma base; el árbol de trabajo estaba limpio.
- No existía implementación productiva de `SellarAmbitoAsignacion` ni de
  `DerivarHuellaAsignacion`; solo estaban los contratos y dobles de prueba.
- Blobs de autoridad verificados:
  `infraestructura_alta.go` =
  `e567334fcfcc74199438fd03ef63ca8c927a9752` y `sellos_alta.go` =
  `d7ee9f4118f2e484d9713472d4c816a1fa7b9c4c`.
- Toolchain: `go1.26.5 linux/amd64`.
- La prueba previa `GOPROXY=off go test -count=1` del paquete
  `adapters/seguridad` terminó verde.

No se usaron Git de red, otros worktrees, Docker, PostgreSQL, despliegue,
producción ni datos o credenciales reales.

## Capability, invariante, write-set y siguiente corte

Capability: `AutoridadSellosAsignacionHMAC` implementa conjuntamente
`ports.SelladorAmbitoAsignacion` y `ports.DerivadorHuellaAsignacion`.

Invariante: dominios de ámbito y petición estrictamente separados, mismas
generaciones activas/retenidas y orden, asignación/reasignación no colisionan,
toda coordenada funcional cambia la huella, cero material sensible fuera del
HMAC.

Write-set exacto:

- `internal/modules/contrataciontemporal/adapters/seguridad/sellos_asignacion.go`;
- `internal/modules/contrataciontemporal/adapters/seguridad/sellos_asignacion_test.go`;
- `docs/portal_vec/ct_lite_o5_01_hmac_asignacion_2026-09-01.md`.

Siguiente corte: consultor durable PostgreSQL solo tras resolver
`R3B`/numeración.

## Diseño implementado

La nueva autoridad es inmutable y posee conjuntamente dos instancias del
`llaveroHMAC` existente. Se construye exclusivamente con
`ConfiguracionSelladorHMAC`; no replica HMAC, crea llaveros alternativos ni
recibe claves. El constructor exige que activo y retenidos tengan las mismas
generaciones y el mismo orden en ambos dominios:

```text
vec.contratacion-temporal.asignacion.ambito
vec.contratacion-temporal.asignacion.peticion
```

La preimagen de ámbito contiene el dominio, la UUID de idempotencia,
organización, actor y perfil. La preimagen de petición contiene el segundo
dominio y operación, organización, expediente, versión, actor, perfil, unidad,
responsable, motivo y observaciones. Ambas son canónicas, se entregan solo al
conector HMAC y se borran al regresar. El resultado expone exclusivamente
sellos HMAC generacionales.

La autoridad no incorpora caché, estado global mutable, goroutines,
persistencia, proveedor de desarrollo, registro ni trazas. Contexto nulo o
cancelado, material inválido, configuración incoherente, dependencia nula
tipada, fallo privado y sello devuelto bajo otro dominio fallan cerrados con
errores opacos. Dos aserciones de compilación acreditan los puertos exactos.

## Matriz de pruebas

Las pruebas focales acreditan:

- rotación `v3 → v2 → v1`, retenidas y orden idénticos en ambos resultados;
- separación de dominios en llavero, preimagen y sello;
- invocación de todas las generaciones en cada llamada, sin caché;
- compromisos distintos para asignar y reasignar;
- cambio de huella al variar operación, organización, expediente, versión,
  actor, perfil, unidad, responsable, motivo u observaciones;
- ausencia de UUID y contenido funcional en resultados y errores;
- borrado de la preimagen prestada al conector al terminar;
- cancelación, receptor/contexto nulos, `nil` tipado, historias generacionales
  divergentes y sello de dominio incorrecto;
- estabilidad concurrente de una misma autoridad con 32 parejas de llamadas.

## Puertas del productor

Con Go 1.26.5 y `GOPROXY=off` terminaron en verde:

```text
gofmt sellos_asignacion.go sellos_asignacion_test.go
go test -count=1 ./internal/modules/contrataciontemporal/adapters/seguridad
go test -count=50 -run 'Asignacion|Reasignacion' ./internal/modules/contrataciontemporal/adapters/seguridad
go test -race -count=2 -run 'Asignacion|Reasignacion' ./internal/modules/contrataciontemporal/adapters/seguridad
go vet ./internal/modules/contrataciontemporal/adapters/seguridad
git diff --cached --check
/tmp/vec-gitleaks-20260831 protect --staged --redact --no-banner --log-level warn
```

El write-set preparado contiene solo los tres ficheros declarados. El
ejecutable focal de Gitleaks se verificó antes de usarlo con SHA-256
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d` y el
escaneo terminó sin hallazgos. Después del commit se ejecuta el `merge-tree`
de solo lectura contra el producto local actual y se informa su resultado
junto al hash candidato; esta nota no puede autoacreditar el hash que la
contiene.

## Seguridad, privacidad, i18n y accesibilidad

Los datos funcionales solo forman preimágenes efímeras del HMAC. La autoridad
no devuelve ni registra UUID, referencias u observaciones, y descarta el texto
de errores privados. Las referencias de prueba son sintéticas; no se añaden
secretos, datos personales, conectores reales ni material de clave al
repositorio.

No hay texto visible, HTTP, web ni presentación. Por ello este corte no cambia
i18n o accesibilidad; tampoco las acredita para la vertical.

## Límites y revisión requerida

Quedan fuera puertos, aplicación, `sellos_alta.go`, PostgreSQL, HTTP, rutas,
manifiesto, web, composición, E2E y documentos transversales. La autoridad no
está compuesta y no demuestra persistencia, notificación durable, aplicación
arrancable, vertical productiva ni cumplimiento formal.

Un agente distinto debe revisar el hash exacto, reproducir las puertas y
emitir `GO` o `NO-GO`. El productor no integra, publica ni se autoaprueba.
