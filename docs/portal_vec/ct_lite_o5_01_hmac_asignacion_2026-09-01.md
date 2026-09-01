# CT-LITE-O5-01-HMAC-A-R1 — ABI y cancelación de sellos de asignación

Fecha: 1 de septiembre de 2026.

Estado del productor: candidato correctivo local de un único commit,
pendiente de doble revisión independiente del hash exacto. Este corte no
concede `GO`, no cierra `O5-01` y no autoriza integración, publicación ni
producción.

## Preflight y NO-GO corregidos

- Worktree exclusivo:
  `.worktrees/ct-o5-01-hmac-a-20260901`.
- Rama conservada: `trabajo/ct-o5-01-hmac-a-20260901`.
- Base y `HEAD` exactos antes de editar:
  `f1c515511c43ad0474f2c89eabff41fb31b8dd46`.
- El padre del candidato original es
  `c8cc4312b063ddca7294dfe27dc673ad1a4676d0`; la rama local de producto,
  su seguimiento local y la referencia remota ya disponible continúan
  coincidiendo en ese hash. No se actualizó ninguna referencia por red.
- El dictamen independiente conservado en
  `/tmp/vec-review-o5-01-hmac-a-20260901.log` declaró `NO-GO`,
  `P0=0, P1=1, P2=0`: ambos métodos podían propagar éxito cuando el conector
  cancelaba el contexto dentro de la última llamada y devolvía un sello
  válido.
- El NO-GO local adicional identificó un segundo P1: las preimágenes carecían
  de marcador de esquema versionado y las pruebas no fijaban su ABI byte a
  byte ni el HMAC resultante.
- Toolchain: `go1.26.5 linux/amd64`. El árbol estaba limpio antes del corte.

No se usaron Git de red, otros worktrees, Docker, PostgreSQL, despliegue,
producción ni datos o credenciales reales.

## Capability, invariante, write-set y siguiente corte

Capability: `AutoridadSellosAsignacionHMAC` continúa implementando
conjuntamente `ports.SelladorAmbitoAsignacion` y
`ports.DerivadorHuellaAsignacion`, ahora con ABI V1 fijada y cancelación
tardía cerrada.

Invariante: esquemas de preimagen, dominios HMAC y generaciones de clave son
coordenadas distintas; ámbito y petición permanecen separados, con las mismas
generaciones activas/retenidas y orden. Ningún éxito se publica si el contexto
queda cancelado o vence antes del retorno.

Write-set exacto:

- `internal/modules/contrataciontemporal/adapters/seguridad/sellos_asignacion.go`;
- `internal/modules/contrataciontemporal/adapters/seguridad/sellos_asignacion_test.go`;
- `docs/portal_vec/ct_lite_o5_01_hmac_asignacion_2026-09-01.md`.

Siguiente corte: doble revisión independiente del hash exacto R1. El consultor
durable PostgreSQL continúa fuera de alcance y condicionado por
`R3B`/numeración.

## ABI HMAC V1

La autoridad conserva los dos `llaveroHMAC` existentes y se construye solo
con `ConfiguracionSelladorHMAC`; no replica criptografía, recibe claves ni
crea un llavero alternativo. Cada preimagen empieza por un esquema interno V1
distinto tanto del dominio HMAC como de la referencia generacional:

| Uso | Esquema de preimagen | Dominio HMAC | Generación de clave |
| --- | --- | --- | --- |
| Ámbito | `vec.contratacion-temporal.asignacion.ambito.v1` | `vec.contratacion-temporal.asignacion.ambito` | `vec.contratacion-temporal.asignacion.ambito/vN` |
| Petición | `vec.contratacion-temporal.asignacion.peticion.v1` | `vec.contratacion-temporal.asignacion.peticion` | `vec.contratacion-temporal.asignacion.peticion/vN` |

El vector dorado de ámbito fija el JSON completo de esquema, UUID de
idempotencia, organización, actor y perfil. El vector de petición fija el JSON
completo de esquema, operación, organización, expediente, versión, actor,
perfil, unidad, responsable, motivo y observaciones. Ambos comparan los bytes
íntegros y el HMAC SHA-256 esperado con material sintético.

El orden de campos, sus nombres, tipos y reglas de serialización forman parte
del ABI V1. Reordenar o renombrar un campo, o añadir o retirar cualquiera,
exige introducir un esquema nuevo; no se permite alterar silenciosamente V1.

## Cancelación tardía y borrado

`SellarAmbitoAsignacion` y `DerivarHuellaAsignacion` capturan la colección y
el error del llavero compartido. Solo después de un sellado correcto vuelven a
consultar `ctx.Err()` y, si observan `context.Canceled` o
`context.DeadlineExceeded`, devuelven la colección cero y el error exacto del
contexto.

Las regresiones usan una única generación: el último conector calcula y
devuelve un sello válido, pero cancela el contexto dentro de esa misma llamada.
Los dos métodos rechazan el éxito y las copias prestadas acreditan que ambas
preimágenes se borraron antes de regresar. El `defer` de borrado se conserva
en todas las salidas.

## Matriz de pruebas

Además de la matriz original, las pruebas focales acreditan:

- dos vectores ABI V1 con comparación byte a byte del JSON completo y del
  HMAC esperado;
- separación explícita entre esquema, dominio HMAC y referencia de generación;
- cancelación dentro del último conector para ámbito y petición, con colección
  cero, `context.Canceled` y preimagen borrada;
- rotación `v3 → v2 → v1`, retenidas y orden idénticos;
- separación de dominios, invocación sin caché y estabilidad concurrente;
- asignación y reasignación distintas y sensibilidad a todas las coordenadas;
- fallos opacos ante configuración, dependencia nula tipada, contexto,
  material o sello de dominio inválidos, sin exposición de datos.

## Puertas del productor R1

Con Go 1.26.5 y `GOPROXY=off` terminaron en verde antes del commit:

```text
gofmt sellos_asignacion.go sellos_asignacion_test.go
go test -count=1 ./internal/modules/contrataciontemporal/adapters/seguridad
go test -count=50 -run 'Asignacion|Reasignacion' ./internal/modules/contrataciontemporal/adapters/seguridad
go test -race -count=2 -run 'Asignacion|Reasignacion' ./internal/modules/contrataciontemporal/adapters/seguridad
go vet ./internal/modules/contrataciontemporal/adapters/seguridad
git diff --check
/tmp/vec-gitleaks-20260831 protect --staged --redact --no-banner --log-level warn
```

El ejecutable focal de Gitleaks se verificó antes de usarlo con SHA-256
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`.
La lista focal confirma que el patrón repetido incluye expresamente
`FijaABIV1ConVectoresDorados` y `CancelaDentroDelUltimoConector`. El
`merge-tree` contra el producto local actual, el hash candidato y el resto de
resultados poscommit se informan tras crear el commit; el documento no puede
autoacreditar el hash que lo contiene.

## Seguridad, privacidad, i18n y accesibilidad

Los datos funcionales solo forman preimágenes efímeras del HMAC. La autoridad
no devuelve ni registra UUID, referencias u observaciones y descarta los
errores privados. Los vectores son sintéticos; no se añaden secretos, datos
personales, conectores reales ni material de clave al repositorio.

No hay texto visible, HTTP, web ni presentación. Por ello este corte no cambia
i18n o accesibilidad ni las acredita para la vertical.

## Límites y revisión requerida

Quedan fuera `sellos_alta.go`, puertos, aplicación, PostgreSQL, HTTP, rutas,
manifiesto, web, composición, E2E y documentos transversales. La autoridad no
está compuesta y no demuestra persistencia, notificación durable, aplicación
arrancable, vertical productiva ni cumplimiento formal.

Dos agentes distintos del productor deben revisar el hash exacto, reproducir
las puertas y emitir sus dictámenes. El productor no integra, publica ni se
autoaprueba.
