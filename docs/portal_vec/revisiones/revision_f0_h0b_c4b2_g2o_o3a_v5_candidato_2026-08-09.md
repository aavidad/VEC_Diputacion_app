# Revisión del candidato O3a V5 — 9 de agosto de 2026

Estado: **GO técnico de la proyección; NO-GO de integración por R**.

## Alcance

La revisión abarca los seis ficheros Go de O3a V5, el conductor durable
C01..C21, el analizador AST/DAG, 195 mutantes atómicos y cuatro mutantes de
autoridad. No modifica ni autoriza R, Orquesta, O3b ni producción.

## Resultado reproducido

- Funcional V13: `GO`, `P0=0,P1=0,P2=0`.
- Seguridad V13: `GO`, `P0=0,P1=0,P2=0`.
- Go 1.26.5, `gofmt`, `go vet`, builds normal y `-race`, y autopruebas:
  verdes.
- Conductor: 14/14 bloques, 74 registros, normal y carrera real, FD 5→5,
  residuos cero.
- AST: tipado verde y una única llamada productiva a `Start`.
- Mutantes: BASE verde, M001..M195 muertos; cuatro SEC muertos; cero
  supervivientes o no compilables.
- Procedencia: diez fuentes `1e4bb36b…`, GOROOT completo `b53ebeab…`,
  GOTOOLDIR `1061bd99…`, runner `b15792c9…`, validador `917b94b2…`.

## Commits candidatos

- `4882d46`: seis fuentes de arranque/custodia;
- `af82f9e`: AST, manifiestos y evidencia V17;
- `0f90902`: conductor durable y evidencia r3.

## Ledger posterior exacto

| Alias | Líneas | SHA-256 |
| --- | ---: | --- |
| R, intacto | 702 | `7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487` |
| G1 | 698 | `e1f8406adda1cdf4db11da780d9bb59fc121d84b09689e95bea1adc649da984a` |
| G6a | 559 | `9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8` |
| G6b | 500 | `f38dd287682916d97230ca18fc8c04b54377c09010e8e0ed5459d6fae29c65af` |
| G6c | 548 | `a8eb81a65fd2ef7183fa2a4d6c3630f4ed733034ca680f64d2b2eb3a9789974a` |
| G7a | 626 | `5f12c5b817282d22304ffc4b940718224df6aa8e64a088402619c8733eaf0dbb` |
| G7b | 675 | `08385aae1ef8b9d6dcc2f7958357e28bc02bc9f71ece69ffbbb4067b1a02cc33` |

La base es `de9b98e9dae14c6a9812adda35694bac5838b191`. El ledger
de fuentes del conductor es `6e1600fb…`; el ledger M001..M195 es
`1425fb74…`, SEC `8711c457…`, GOROOT `b53ebeab…` y GOTOOLDIR `1061bd99…`.

## Condición de parada

R permanece exactamente en 702 líneas y SHA-256 `7ad65a66…`. El contrato
exige que R capture las cinco fuentes nuevas y amplíe su manifiesto antes de
declarar material autónomo. La orden vigente prohíbe usar R. Por tanto este
acta acredita la proyección y permite publicar la rama de trabajo para
trazabilidad, pero no autoriza CI como puerta de integración, materializar en
`integracion/ct-o4-04e-20260726`, desplegar ni cambiar métricas.

Siguiente decisión necesaria: autorizar explícitamente la minitarea R o aprobar
una enmienda contractual independiente con doble revisión. Hasta entonces,
O3a, `Start`, mapa FD, O3b y producción continúan `NO-GO`.
