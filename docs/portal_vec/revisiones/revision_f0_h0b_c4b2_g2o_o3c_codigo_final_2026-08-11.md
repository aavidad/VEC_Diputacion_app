# Revisión material final F0-H0b/C4b-2/G2-O/O3c

Fecha: 11 de agosto de 2026.

Identificador: `O3C-P7-EVIDENCIA`.

Estado: **GO documental material, P0=P1=P2=0; pendiente de publicación y CI**.
Este documento no autoriza merge, integración, producción, despliegue, cambio
de métricas ni apertura de O4a--O6.

## Alcance y autoridad

La revisión contrasta requisito por requisito la
[decisión O3c](../decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md)
contra la cabeza publicada P6
`bc926891654e9800a71d6039b389afb10bacb3df`. El contrato tiene SHA-256
`f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6`.
La toolchain sellada es `go1.26.5 linux/amd64`.

P7 añade únicamente este ledger y sus dos revisiones materiales. Código
productivo P1--P5 y herramientas/evidencias P6 permanecen byte a byte
inmutables.

## DAG exacto P0--P7

La cadena es lineal, sin merges:

| Lote | Commit final | Padre | Rama publicada | Responsabilidad |
| --- | --- | --- | --- | --- |
| P0 | `edc4a4d207a80a9520411865bcd2afdb77655fc3` | `2761eca4914160dd41546d60c3c91898d42cc7e7` | `trabajo/o3c-p0-contrato-20260811` | Contrato ejecutable y cierre de publicación; el material nace en `2761eca4914160dd41546d60c3c91898d42cc7e7` sobre O3b `124aa60e19d8daf31906098d4d60b4a4d2c6e281`. |
| P1 | `254b7364cab48d5e4770e470bcb1e2d5e6d01778` | `edc4a4d207a80a9520411865bcd2afdb77655fc3` | `trabajo/o3c-p1-autoridad-20260811` | Autoridad C0--CF y consumo lineal. |
| P2 | `19694995d09fa881c99521416e0587f63121d71c` | `254b7364cab48d5e4770e470bcb1e2d5e6d01778` | `trabajo/o3c-p2-revalidacion-20260811` | Revalidación final y permiso CONT. |
| P3 | `da53a224e3dfec2e014e480c9bdd4333085aac85` | `19694995d09fa881c99521416e0587f63121d71c` | `trabajo/o3c-p3-cont-20260811` | Marca de 180 s y CONT único. |
| P4 | `29ca98b88c9a4644b4676c9705bfd80b2f723c37` | `da53a224e3dfec2e014e480c9bdd4333085aac85` | `trabajo/o3c-p4-observacion-20260811` | Primera observación raw inmediata. |
| P5 | `c0f2a9945ed2fc5648980ee48b91424a04977655` | `29ca98b88c9a4644b4676c9705bfd80b2f723c37` | `trabajo/o3c-p5-handoff-20260811` | Handoff O4a opaco y retirada pre-CONT. |
| P6 | `bc926891654e9800a71d6039b389afb10bacb3df` | `dbe038e70060ccb89826a80565cc3bee18fb8ee2` | `trabajo/o3c-p6-conductor-20260811` | Conductor, AST/tipos/DAG y 144 mutantes; el material inicial es `dbe038e70060ccb89826a80565cc3bee18fb8ee2`. |
| P7 | pendiente | `bc926891654e9800a71d6039b389afb10bacb3df` | `trabajo/o3c-p7-evidencia-20260811` | Ledger, revisiones y cierre documental. |

Aristas: `P0→P1→P2→P3→P4→P5→P6→P7`. O4a-P0 permanece
bloqueada hasta la publicación y CI 5/5 de P7.

## CI publicada P0--P6

Cada enlace fue contrastado en GitHub con la rama y el SHA remoto exactos. Las
siete ejecuciones muestran cinco jobs y los cinco concluyeron correctamente.
En P6, los cinco jobs públicos están `completed/success`; a la hora de esta
auditoría la cabecera HTML del run aún mostraba `In progress`. Dirección
designó expresamente esa CI como 5/5. Este ledger acredita los cinco jobs y no
reinterpreta la cabecera del proveedor como `completed`:

| Lote | CI | SHA final |
| --- | --- | --- |
| P0 | [31455895954](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31455895954) | `edc4a4d207a80a9520411865bcd2afdb77655fc3` |
| P1 | [31459027283](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31459027283) | `254b7364cab48d5e4770e470bcb1e2d5e6d01778` |
| P2 | [31463581591](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31463581591) | `19694995d09fa881c99521416e0587f63121d71c` |
| P3 | [31466890405](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31466890405) | `da53a224e3dfec2e014e480c9bdd4333085aac85` |
| P4 | [31470363744](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31470363744) | `29ca98b88c9a4644b4676c9705bfd80b2f723c37` |
| P5 | [31476338742](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31476338742) | `c0f2a9945ed2fc5648980ee48b91424a04977655` |
| P6 | [31504085619](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31504085619) | `bc926891654e9800a71d6039b389afb10bacb3df` |

Los jobs son `puerta-secretos`, `puerta-calidad`,
`puerta-artefactos-productivos`, `puerta-postgresql-contexto-actor-v3` y
`puerta-postgresql-bolsa-publica`. Las puertas PostgreSQL son globales del
repositorio y no convierten PostgreSQL en parte funcional de O3c.

## Ledger productivo P1--P5

| Fuente | Líneas | SHA-256 |
| --- | ---: | --- |
| `continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go` | 256 | `150c46ebeef8b6d2850d735b1f679701d620f0ef54850ab01f20fca986c9a599` |
| `continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go` | 293 | `35409603d803a6d74288a391bea93239d2246fbe3c0d35eecf2063c0da1fe1aa` |
| `continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go` | 100 | `447d5779ea90731b3b53a46870861b79bc95a478bd0fa7540b717ea279bd94be` |
| `continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go` | 146 | `bf2a5814608479cfe03628be31e951087a6da29f21f8a88a053108fa0d6620b0` |
| `continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go` | 269 | `66fb9c71e8c5d5e03cd7a32380986c23a0139720673a9b2348e1bcf03d3ec4cf` |

Las cinco fuentes productivas suman 1064 líneas; cada fichero queda por debajo
de la parada contractual de 650 y del tope DEC-051 de 800. Las pruebas focales
selladas tienen, respectivamente, SHA-256 `0a2e6da9…`, `85ee7784…`,
`4f805712…`, `44b0c6f8…` y `13bac7a3…`.

## Conductor durable P6

Autoridades:

- conductor `5a0fd5515c8909b438ca718a47bd0dc7a630b27b0900be64888328eb157ae580`;
- matriz `1e2d93e4f24d53c470699fc2751deaba5ef5132aa173e41a02709c03ef7d547f`;
- fuentes `a26bda20e5ba7ad8f5015b54b491cd4ded0ed3b756e5efcfa7e86f61f382b9b1`;
- toolchain `go1.26.5 linux/amd64`.

La evidencia canónica contiene 244/244 filas `GO`: 122 normal y 122 race.
C01--C22 aparecen una vez por modo y C22 aporta cien capturas por modo. Son
exactamente 100 capturas normales y 100 race. Los inventarios de FD, hijos,
zombis, grupos y temporales tienen delta cero; el runner exige la desaparición
del PGID hasta `ESRCH` y `residuos.txt` está vacío.

Seis procesos BF directos —tres normal y tres race— acreditan estado 65,
EOF/no retorno, `stdout=0` y `stderr=0`. No se usa el `PASS` del harness como
sustituto. No hay `SKIP` ni reintento que oculte fallos.

## AST, tipado, DAG y límites

| Artefacto | Líneas | SHA-256 |
| --- | ---: | --- |
| `tools/o3c_p6_ast/main.go` | 397 | `0b8bdc3bc121754884dfd1435143f952700026741bccf874b13c5057c2ba3984` |
| `tools/o3c_p6_ast/invariantes.go` | 786 | `6b64a5c327a8f84019229082caf33bb7ff00946a74910ba129f0451fc4c3b181` |
| `tools/o3c_p6_ast/retirada.go` | 82 | `55500193cce25d83b912f0b6298994864ff970cbd1718ef2dd1f0441ca6461cd` |
| `tools/o3c_p6_ast/seguridad.go` | 78 | `39299179b51b7263da92b4275b40463f0b8c13af12923b4a32ac2dfa8b8f9669` |

El analizador tipa las cinco fuentes, construye el DAG de llamadas y ownership
y acredita diez grupos: entrada/máquina, autoridad conjunta, revalidación,
marca/CONT, observación, handoff, retirada C7, fatalidad, fronteras O4 y APIs
prohibidas. Ningún fichero supera 800 líneas.

## Expansión mutante

La evidencia final sella:

- base productiva `c0f2a9945ed2fc5648980ee48b91424a04977655`;
- digest estable de fuentes objetivo
  `d32f321453553eea76724e91d02032e4f68ea537e93255dbe58dc2f31c1fa1fe`;
- runner `232ef62922300433d28a7851c48713cd161fb256bfa672503447adc235429ff1`;
- fusión `661862dfa62906db3e60ca46fd7cb534da91845cc559b99008dd8bc0b6f7f578`;
- binario canónico `ae46aab541bdf38073a56f2d21741f211e3c08fdc51b9fe4adb8434c888dd225`;
- `SHA256SUMS` `279de79a111b6c16879f5655d95e1c4cc5501d48b767a528f7d7cf35d2b679c5`;
- manifiesto `c76bf9cde83ebb82f2d979fc4ef426bc439d56d03afc30130d60a2d3762c8b0c`;
- resultados `ba9fde987b8356a548945e2ed609514834532f87ca5b3aa297b520a66ea1adec`.

El preflight compiló 144/144 mutantes. Seis lotes secuenciales y su fusión
acreditan 144/144 compilables y muertos en 24/24 familias: 73 por AST/tipos/DAG,
55 por prueba causal, cinco meta-conductor y once meta-evidencia. No hay huecos,
duplicados, supervivientes, timeout aceptado como muerte, ruta privada ni
proceso/PGID residual. `go run` falla cerrado; dos builds `-trimpath` producen
el mismo binario. El runner funciona en checkout de un solo commit y después
de un commit posterior porque la autoridad es el digest de las fuentes, no un
`HEAD` circular.

## Auditoría requisito por requisito

| Requisito contractual | Evidencia final |
| --- | --- |
| Consumo B5 lineal, C0--CF, CAS 1/1→2/3 y anti-replay | C01--C03, AST y mutantes C01--C03. |
| Revalidación CONTROL→observador→TID/PPID/PDEATHSIG→bootstrap→pidfd/inventario→`/proc`→segunda ronda | C03--C08, AST tipado y familias C03--C08. |
| Marca monotónica única de 180 s y CONT inmediato | C09--C13, AST y familias C09--C13. |
| Primario, `SIGCONT`, `NULL`, flag `1<<2`, sin reserva/reintento | C10--C13 y mutantes de argumentos/orden. |
| Unión cerrada de cinco observaciones y precedencia | C14--C16, AST y familias C14--C16. |
| Handoff C3→C4T→C5, owners observador→lease y agregado opaco | C17--C18, AST ownership y familias C17--C18/C23. |
| Retirada C7→C8/CF, deadline 3 s, KILL individual, Wait→ECHILD→ESRCH→cierres/liberación | C19, AST retirada y familias C19--C21. |
| Post-CONT sin retirada local, causa ni O4; CF 65/EOF/0/0 | C20--C21 y seis BF directos. |
| APIs prohibidas, TERMINAL y ownership acíclico | AST seguridad/DAG y familias C20--C24. |
| Matriz normal/race y residuos cero | C22, 100+100 capturas y cinco meta-mutantes de conductor. |

## Reproducción desde checkout limpio

```bash
(cd tools/o3c_p6_conductor/evidencia && sha256sum -c SHA256SUMS)
sha256sum -c tools/o3c_p6_mutantes/evidencia/SHA256SUMS
go test ./tools/o3c_p6_ast ./tools/o3c_p6_mutantes
go test -race ./tools/o3c_p6_ast ./tools/o3c_p6_mutantes
go vet ./tools/o3c_p6_ast ./tools/o3c_p6_mutantes
go run ./tools/o3c_p6_ast \
  -dir deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
destino=$(mktemp -d); rmdir "$destino"
tools/o3c_p6_conductor/conductor.sh . "$destino"
go test ./...
go test -race ./...
go vet ./...
scripts/verificar_calidad.sh
git diff --check
```

Las puertas focales normal/race, conductor completo, AST/tipos/DAG, mutantes,
checksums, gofmt, ShellCheck, vet y calidad global quedaron verdes. Gitleaks se
acreditó en la CI P6 `31504085619` mediante `puerta-secretos`.

## Seguridad, datos y fronteras

- No se trataron datos personales, secretos, PID/pidfd/ticket serializados ni
  rutas privadas.
- No hubo Orquesta, Firecracker, Jailer, Docker funcional O3c, SQL, HTTP,
  producción o despliegue.
- O4a, O4b, O4c, O5 y O6 permanecen cerrados.
- No se modifican master, integración ni porcentajes oficiales.
- F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
  `1/14` y producción `NO-GO` no cambian por este documento.

## Revisiones independientes

La [revisión funcional](revision_funcional_f0_h0b_c4b2_g2o_o3c_2026-08-11.md)
y la [revisión de seguridad](revision_seguridad_f0_h0b_c4b2_g2o_o3c_2026-08-11.md)
releyeron el mismo snapshot corregido y emitieron `GO`, `P0=P1=P2=0`.
Funcional reprodujo hashes, cadena, 244 casos, seis BF, AST y 144 mutantes.
Seguridad volvió a comprobar fronteras, refs remotos, residuos, secretos y
ausencia de cambios fuera del ledger.

## Condición de cierre

Solo el doble GO permite commit documental, push normal y comprobación de la
CI P7 exacta 5/5. Dirección conserva en exclusiva la decisión de integrar y de
asignar O4A-P0-CONTRATO.
