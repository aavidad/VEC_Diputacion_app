# CT-LITE-O6-03-B2 — agregado de llamamiento abierto en Bolsa

Fecha: 31 de agosto de 2026.

## Capability

Este corte incorpora al dominio puro de Bolsa un valor `LlamamientoAbierto`
que enlaza por referencias opacas un llamamiento con su bolsa, necesidad y
propuesta. El valor nace en estado técnico `abierto` y admite una única
transición terminal a los resultados ya autorizados por el contrato O6:
`aceptacion`, `renuncia` o `expiracion_gobernada`.

La transición aplica compare-and-swap sobre una versión dentro del entero
seguro común de JSON. Un replay con la misma versión esperada, el mismo estado
terminal y la misma referencia opaca de operación devuelve exactamente el
valor ya producido. Otra terminal o una versión divergente fallan sin mutar el
receptor.

## Invariantes

- El agregado es un valor puro e inmutable: cada transición devuelve una copia.
- Las referencias de llamamiento, bolsa, necesidad, propuesta y operación son
  opacas y reutilizan el validador endurecido del dominio de llamamientos.
- El estado abierto no contiene terminal; un estado terminal contiene
  exactamente el terminal del mismo tipo.
- La versión es positiva, no desborda el entero seguro interoperable y aumenta
  una sola vez al cerrar.
- La idempotencia se comprueba antes de intentar un nuevo efecto terminal, pero
  solo acepta la repetición exacta del comando original.
- No existen campos de persona, sujeto, participación, candidato, selección,
  contacto o sucesor.
- Los errores son centinelas opacos y no incluyen referencias ni entradas.

## Write-set

- `internal/modules/bolsa/domain/llamamiento_abierto.go`
- `internal/modules/bolsa/domain/llamamiento_abierto_test.go`
- `docs/portal_vec/ct_lite_o6_03_b2_agregado_llamamiento_abierto_2026-08-31.md`

Los tres ficheros son nuevos. El corte no modifica aplicación, puertos,
adaptadores, SQL, composición ni documentos transversales de estado.

## Pruebas del corte

La batería focal cubre construcción válida e inválida, valor cero y terminal
nulo, referencias endurecidas, copias defensivas, los cuatro estados exactos,
CAS sin efectos, replay exacto, terminal incompatible, borde del entero seguro,
errores opacos, ausencia estructural de referencias personales o selección y
derivaciones concurrentes de un mismo valor puro.

Puertas exigidas para el candidato:

```text
gofmt
Go 1.26.5 offline: go test ./internal/modules/bolsa/domain
Go 1.26.5 offline: go test -race ./internal/modules/bolsa/domain
Go 1.26.5 offline: go vet ./internal/modules/bolsa/domain
git diff --check
write-set, modos y límites
Gitleaks local con binario de huella acreditada
merge-tree de solo lectura si producto conserva la base exacta
```

Resultados reproducidos antes del commit:

- `go version`: `go1.26.5 linux/amd64`, con `GOTOOLCHAIN=go1.26.5` y
  `GOPROXY=off`;
- `go test ./internal/modules/bolsa/domain`: verde;
- `go test -race ./internal/modules/bolsa/domain`: verde;
- `go vet ./internal/modules/bolsa/domain`: verde;
- `git diff --check`: verde;
- tamaños: 180 líneas productivas, 332 de prueba y 100 documentales, todos por
  debajo de 500/800 según corresponda;
- modos: los tres ficheros nuevos son `0644`;
- Gitleaks: binario `/tmp/vec-gitleaks-20260831` verificado con SHA-256
  `c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`;
  el escaneo individual de cada fichero del write-set terminó sin hallazgos.

Un escaneo adicional `--no-git` de todo el árbol señaló 60 coincidencias
preexistentes fuera de este write-set. No se atribuyen al candidato ni se
presentan como una puerta global verde. El merge-tree se ejecuta únicamente
sobre el hash final y se informa en la entrega, sin modificar producto.

## Limitaciones

El agregado no elige ni llama a ninguna persona; tampoco selecciona sucesor,
comunica, calcula plazos, usa reloj o aleatoriedad, persiste, autentica una
orden, decide competencia administrativa, escribe auditoría/outbox ni compone
una ruta. La referencia terminal es identidad semántica, no prueba por sí sola
de autorización, durabilidad o efecto jurídico.

Este corte aislado no cierra O6-03, no acredita integración con el expediente,
API, web, E2E o producción y no cambia métricas.

## Siguiente corte

Un corte dependiente deberá consumir este valor desde la aplicación de Bolsa
con una orden opaca ya autenticada, dejando autorización, persistencia,
auditoría, inbox/outbox y reconciliación en sus autoridades correspondientes.
Solo una revisión independiente del hash exacto podrá recomendar su integración.
