# Revisión de seguridad O3a V5: estados causales con Go 1.26.6

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-ESTADOS-CAUSALES-GO1.26.6`.

Dictamen: **NO-GO de seguridad**, `P0=0`, `P1=1`, `P2=0`.

Este dictamen juzga solo la fidelidad y el cierre seguro de la instrumentación
diagnóstica. No acredita la estabilidad de O3a V5, el parche de toolchain, O4,
publicación, CI remota ni producción.

## Identidad y alcance

La revisión se hizo sobre el candidato exacto
`a997acbb9a51c02168f1648225c0e69c84e648d4`, con padre
`74f249587d5c0da41092a2c017ccd6cb54817248` y árbol
`38473c53cd1838113e549bf53233136a7e1da3cd`. La ascendencia, los tipos de
objeto, los modos y `git diff --check` son correctos. El worktree productor no
se modificó.

El delta candidato contiene `+211/-25` en seis rutas:

| Ruta | Estado | Líneas | SHA-256 |
| --- | :---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go` | M | 742 | `4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7` |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go` | M | 741 | `084e5363ec969ef705caed7dcc213b5d7a54483574cf18bc2b0382fd6cccadf5` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_go1_26_6_2026-08-14.md` | A | 150 | `2c3688fce80014027a5b3192ff0fd944579e3cd963843e9669d3ad151c6b8064` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | M | 11 | `96907cac6ef96c9c7bc4770385cfd054c638039739d459c9ea51d5988b5b5ceb` |
| `tools/o3b_p7_conductor/fuentes.tsv` | M | 23 | `2191a7b06e50da65df36e58bd8403a6bec6110f4b5dd229a272f5cb6172c29b5` |
| `tools/o3c_p6_conductor/fuentes.tsv` | M | 33 | `4f60e1ec8e3c09a928f56eba2fce0a6d26c5fb331776117b19bd858562f4e7ef` |

Los seis objetos son blobs ordinarios `100644`. Los dos fuentes modificados
son test-only por `//go:build ignore && linux && amd64`; no cambia fuente
productivo, conductor, workflow, oráculo esperado, plazo, cardinalidad, mapa
FD, PostgreSQL, credenciales, estado transversal ni métricas. El único
write-set revisor es esta acta.

## Auditoría causal y de privilegio

La constante `99 + iota` produce exactamente quince estados distintos,
`99..113`. Todos son no cero y ninguno es aceptado como `GO` por los
conductores. Las ramas quedan trazadas así:

- TUPLA: netpoll 99, snapshot inicial 100, clase/origen 101, limpieza 102,
  snapshot final 103, delta FD 104 e hijo restante 105;
- C16: alias preparado 106 y alias retirado 107;
- C17: testigos/TID 108;
- C18: barrera parcial 109, barrera de plazo 110 y vuelta tardía 111;
- selector lineal ajeno 112 e hijo lineal restante 113.

Los estados previos 77, 78 y 93..98 continúan atribuyendo preparación común;
0, 65 y 72..76 no cambian. Los scripts mantienen los valores esperados
anteriores y no aceptan `99..113`, `SKIP`, retry ni tolerancia adicional.

En TUPLA se conserva el cortocircuito histórico: si clase u origen son
inválidos se devuelve 101 antes de ejecutar la limpieza, igual que el primer
operando verdadero del `||` anterior. Si el resultado es válido, snapshot,
comparación FD y comprobación de hijos conservan orden y cardinalidad. C16,
C17 y C18 conservan asimismo su orden; el cambio observable pretendido es
solo el estado del primer predicado fallido.

La corrección en `probarAliasRetiradaExternaO3aM38` sí elimina un falso
verde: asigna a `err` el resultado de `escribirControlPruebaO3aM38` antes de
retornarlo. No se añaden impresiones, logs, stdout, stderr ni otro canal
lateral. Las fuentes siguen debajo de la parada local de 750 y del tope duro
de 800; G7a queda en 742 líneas y G7b en 741. Los tres ledgers contienen las
dos huellas exactas anteriores y solo esas dos entradas cambian.

## Hallazgo P1: C18 conserva un falso verde de escritura

En
`supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go`,
la función `probarVueltaTardiaExternaO3aM38` conserva esta forma en las líneas
727--729 del candidato:

```go
if err = prepararFixtureO3aM38(f); err != nil || escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|") != nil {
	return err
}
```

Si la preparación termina sin error y falla esa primera escritura, el segundo
operando hace verdadera la condición, pero su error no se asigna. `return err`
devuelve `nil` si la limpieza diferida también termina bien, aunque se omita el
resto del protocolo de vuelta tardía. El llamador continúa y C18 puede devolver
0 en vez del estado causal 111. Es el mismo patrón de falso verde que el
candidato corrige en C16, todavía presente en otra rama que declara cubrir.

El defecto es fail-open respecto del oráculo de prueba: una indisponibilidad
de escritura puede presentarse como éxito y ocultar precisamente la causa que
la minitarea pretende hacer observable. Es **P1**, no P0, porque vive en
fuente test-only y no concede una autoridad productiva; tampoco es P2 porque
invalida directamente el criterio único del corte.

La corrección exigida es separar preparación y escritura y asignar el error de
la escritura antes de retornarlo, igual que en C16; después deben actualizarse
la huella de G7b, los tres ledgers y la enmienda, y presentarse un nuevo SHA a
doble revisión. Esta acta no corrige el productor.

## Rectificación expresa de la sonda C16

Se retira por completo del razonamiento de esta revisión el C16 estado 66
obtenido al ejecutar directamente `conductor_c15_c21.sh` bajo `flock`. Esa
sonda heredó como FD 3 el descriptor del candado sin `CLOEXEC`; el conductor
completo cierra expresamente todos los FD mayores o iguales que 3 en
`ejecutar_bloque_aislado`. Por tanto aquel C16 fue ambiental y no es evidencia
contra `74f2495` ni contra este candidato. No se reescribe el acta histórica,
pero su inferencia C16 no se reutiliza aquí.

Permanece válido el rojo durable del conductor completo original: C21 race,
índice 44, `TUPLA_C`, esperado 0, observado 66, stdout y stderr vacíos,
FD 5→5 y residuos cero. El propio candidato dice que solo diagnostica, no
estabiliza. Una corrida verde posterior, incluida la de esta revisión, no
revoca ese P1 de estabilidad ni satisface una prueba proporcional de
determinismo.

## Evidencia y puertas

La evidencia productora O3a en
`/srv/fabrica/revisiones/evidencia-o3a-v5-estados-causales-a997acb-r1`
verifica su `SHA256SUMS` completo y registra GO 14/14, 74 casos, Go 1.26.6,
FD 5→5 y residuos cero. Sus huellas principales son:

| Artefacto | SHA-256 |
| --- | --- |
| `resumen.txt` | `c6a6a8547e910e80f501175e8b92b07d2bc6ace63f13a05fdec4b9719ec87e1f` |
| `manifiesto.tsv` | `7245d0c499083bfedec0b93c48d46214798a445c9c6e8d4874e0754210e4bfc4` |
| `c15_c21_race_c21_indices.tsv` | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `5825ffc2286691d4cb3fd5c3a0ccc37ce7f40b1dcd9015ca8eedaf31b814da9f` |

La reproducción de seguridad se ejecutó una sola vez, sin retry, sobre el
clon limpio exacto propiedad de `orquesta`, con el candado cerrado antes del
proceso. Terminó GO 14/14, 74 casos, normal y race, FD 4→4 y residuos cero;
`SHA256SUMS` es válido. La evidencia está en
`/srv/fabrica/revisiones/evidencia-o3a-v5-estados-causales-seguridad-a997acb-r1`:

| Artefacto | SHA-256 |
| --- | --- |
| `resumen.txt` | `63dcd4fc4fcb65e6caf63ea644a83f863e462094dd761ac29156f438a5501543` |
| `manifiesto.tsv` | `f6edee984da87caee1301f6c3151ad341d58667d87d12d38c8ad50a8cbb697ee` |
| `c15_c21_race_c21_indices.tsv` | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `d2df205561e018f5ff6a7a345a20e6a9ebf825993541d5a19876a110b52affd2` |

La compilación focal independiente de los diez fuentes pasó `gofmt`, `go
vet`, build normal y build `-race`. Los binarios efímeros tuvieron SHA-256
`1c1e9431e225af10424d498990cfad7b89d15adfc67d1aa04f4f4d0db9c1207c`
y `f4a8369728cc644544fa712b8aadedaf8134e1854dabff9a4242d4056f9da215`.

Las evidencias productoras compartidas se validaron sin reinterpretarlas:

- O3b P7, Go 1.26.5: 234 filas y seis O17 directos, todo GO, residuos cero;
  hashes `0d018ec1c8a6f73259c87e912b8892fdfc46f41b1cab5ad4459abbecddc69784`,
  `bac12a9285022beda3a9e1e74db2289cca60c2ee935f2968a3fb3cdfa711edf2`,
  `602341bc67e52d66ea27cd3323589b045dd7f60432d96416917d395fe0faab1b`
  y `d40a0f3750fbec83fd51e6ae5b2abc1fefb0c5520df648ca9287dcbf71d4f6cc`;
- O3c P6, Go 1.26.5: 244 filas y seis BF directos, todo GO, residuos cero;
  hashes `75398451e0dc30fd915d068c62e3244671778b73b1363bc0a7b8d1b0f4d91a82`,
  `873887779c51884074115a554f526ac092430af709faa3058e2657fbbf17a849`,
  `73d34dab8e9145a9b681ffb0b5e188d889bcc67e24bb1978e35ce43b06f2bf2e`
  y `e7c666bd491ddda59166eeeef015e4cbe824f17bbeacef968d6a5549e1ae3009`.

No se repitieron esos dos conductores pesados ni la calidad global ya verde:
la validación proporcional de sus evidencias selladas basta para comprobar
los ledgers compartidos y no puede subsanar el falso verde determinista de
C18.

Gitleaks v8.30.0 recorrió el delta `74f2495..a997acb` (1 commit, 9,72 KB) y
el rango acumulado `5345d5d..a997acb` (22 commits, 178,28 KB), sin fugas. La
búsqueda focal tampoco encontró clave, token, DSN o credencial. No hay enlaces
Markdown locales en la enmienda candidata que puedan quedar rotos.

Comandos principales reproducidos:

```text
git rev-parse HEAD HEAD^ HEAD^{tree}
git diff --stat HEAD^ HEAD
git diff --name-status HEAD^ HEAD
git diff --check HEAD^ HEAD
git ls-tree -r HEAD -- RUTAS_CANDIDATAS
wc -l RUTAS_CANDIDATAS
sha256sum RUTAS_CANDIDATAS
gofmt -d FUENTES_O3A
go vet FUENTES_O3A
CGO_ENABLED=0 go build -trimpath -o BINARIO_NORMAL FUENTES_O3A
CGO_ENABLED=1 go build -race -trimpath -o BINARIO_RACE FUENTES_O3A
flock --close .sec-toolchain-review-gates.lock runuser -u orquesta -- tools/o3a_v5_conductor/conductor.sh CLON_EXACTO EVIDENCIA_NUEVA
(cd EVIDENCIA && sha256sum -c SHA256SUMS)
go run github.com/zricethezav/gitleaks/v8@v8.30.0 git . --no-banner --redact --no-color --log-opts=RANGO
```

## Cierre y relevo

Resultado de esta minitarea: **NO-GO de seguridad**, `P0=0`, `P1=1`,
`P2=0`. El P2 documental del inventario de vulnerabilidades pertenece al
ancestro de toolchain y sigue abierto fuera de este dictamen; `P2=0` aquí no
lo acredita ni lo cierra.

Siguiente trabajo causal: corregir exclusivamente la propagación del error de
la primera escritura en C18, actualizar sus tres ledgers y obtener revisión
funcional y de seguridad sobre el nuevo SHA. El P1 histórico de estabilidad
O3a permanece abierto hasta evidencia proporcional posterior. No se autoriza
integración, push, despliegue, credenciales, producción ni cambio de métricas.
