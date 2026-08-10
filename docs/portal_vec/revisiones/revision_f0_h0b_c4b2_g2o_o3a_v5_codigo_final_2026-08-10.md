# Revisión final O3a V5 — 10 de agosto de 2026

Estado: **GO técnico final, P0=0, P1=0, P2=0**.

## Alcance y procedencia

El corte publicado es `12a8bc578fa4adfdd2d349463dbec46da182a93d` en
`trabajo/o3a-v5-20260809` e `integracion/ct-o4-04e-20260726`. Desde la base
`0fb80ac689d5803db8f168497bb61210937f1408` añade una cadena lineal, sin merges:

- `3864bde`: aislamiento y atribución causal de C03;
- `ef183be`: conductor C21 aislado y evidencia durable V26;
- `11360eb`: reproducción mutante hermética V25;
- `12a8bc5`: enmienda de autoridad y límites del corte.

El fast-forward de la integradora partió del remoto
`de9b98e9dae14c6a9812adda35694bac5838b191`. No se aplicó el checkpoint, no se
mezclaron worktrees y los cuatro ficheros del WIP transversal quedaron fuera del
índice y conservaron sus hashes.

## Ledger material

El ledger completo de diez fuentes está en
`tools/o3a_v5_conductor/fuentes_v5.tsv`, SHA-256
`fc654291ffda8c9dae2ef690267b842568082644cfe2d384d718238ac85d492a`.
Las seis fuentes propias de O3a son:

| Alias | Líneas | SHA-256 |
| --- | ---: | --- |
| G1 | 698 | `e1f8406adda1cdf4db11da780d9bb59fc121d84b09689e95bea1adc649da984a` |
| G6a | 559 | `9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8` |
| G6b | 500 | `f38dd287682916d97230ca18fc8c04b54377c09010e8e0ed5459d6fae29c65af` |
| G6c | 548 | `a8eb81a65fd2ef7183fa2a4d6c3630f4ed733034ca680f64d2b2eb3a9789974a` |
| G7a | 715 | `d8bad20d409af9a4062d9a9fc73b9b0f014e4827fcbb0967e7263fee884971f3` |
| G7b | 732 | `7182defc49463a2da9a5cc4ef8f59440d0e90046ef33db8a308c557d2af18d6e` |

G7a y G7b respetan las paradas locales 750/750 y el tope duro 800 de DEC-051.
R permanece intacto: 702 líneas y SHA-256
`7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487`.

## Evidencia reproducida

- Conductor `bb65400…`: 14/14 bloques, 74 casos, normal y `-race`, FD 6→6,
  residuos cero.
- C21 `edf6a719…`: cien procesos aislados por modo, exactamente una ejecución
  por índice, sidecar 1..100 y corte inmediato sin reintento.
- Evidencia `evidencia-cnd-v5-26-final-r1`: 21/21 entradas válidas, SHA de
  `SHA256SUMS` `84af39aa9c89bb81a50add06e9c048c0eccc2c30a237a19b583e6b91feaf7bb1`.
- Mutantes V25: BASE verde, M001..M195 muertos, cero supervivientes o no
  compilables; SEC 4/4 muertos. Ledgers `ed840cd3…` y `d3005f88…`.
- Analizador AST tipado: una llamada productiva a `Start`.
- `go test ./...`, `go test -race ./...`, `go vet ./...` y `git diff --check`
  verdes en la candidata y en la integradora.

Los dos carriles independientes —funcional y seguridad— terminaron con `GO`,
`P0=0,P1=0,P2=0` en cada hito V39, V40, V41 y V42. V39 cerró el código; V40
autorizó la materialización tras la CI candidata; V41 autorizó el push
integrador; V42 verificó el cierre publicado.

## Publicación y CI

- Rama de trabajo: CI `31381886800`, cinco de cinco jobs verdes sobre
  `12a8bc578fa4adfdd2d349463dbec46da182a93d`.
- Rama integradora: CI `31385725239`, intento uno y cinco de cinco jobs verdes
  sobre el mismo SHA. En `puerta-calidad` pasaron la calidad global y el paso
  `Probar el conductor durable O3a V5 sin R`.
- Las puertas de secretos, artefactos productivos, PostgreSQL contexto actor y
  PostgreSQL Bolsa pública terminaron verdes.

No se usó force-push, no se ejecutó R y no se arrancaron Orquesta, Firecracker
ni Jailer. No quedaron procesos, hijos, FD o temporales gobernados por el corte.

## Límites y siguiente frontera

Este GO cierra únicamente el corte técnico O3a V5. No cambia métricas, no abre
producción, despliegue, datos reales, O3b ni fases posteriores. La siguiente
frontera del mapa es O3b, pero requiere un identificador y contrato autorizados;
esta revisión no los crea ni los sustituye.
