# Revisión material final F0-H0b/C4b-2/G2-O/O3b

Fecha: 11 de agosto de 2026.

Identificador: `O3B-P8-EVIDENCIA-V2`.

Estado: **GO documental, P0=P1=P2=0**. Dos revisiones independientes sobre el
mismo snapshot acreditaron el cierre material. Este documento no autoriza
merge, producción, despliegue, O3c ni O4a--O6.

## Alcance y autoridad

La revisión contrasta requisito por requisito la
[decisión O3b](../decision_f0_h0b_c4b2_g2o_o3b_ticket_stop_identidad_2026-08-10.md)
contra la cabeza publicada
`d1de33fc3246b5ebcabe2728deb181da849527c6`. El contrato tiene SHA-256
`d9aa33eddf90da2fb0e7f1aac239a18797e70b8afe7f9fe3024f1a9e5f401ada`.
La toolchain efectiva es `go1.26.5 linux/amd64`.

P8 V2 añade solo este ledger y sus dos revisiones. Código productivo P1--P6,
pruebas y herramientas P7/P7B permanecen byte a byte inmutables.

## Cadena exacta P0--P7B

| Lote | Commit | Padre | Responsabilidad |
| --- | --- | --- | --- |
| P0 | `a6a0b40f09d74445465ecfb9b411baf55d8f9266` | `672b67102be91d33c2a60ca7dfa4d45d6dbd643d` | Contrato O3b. |
| P1a | `24b62169051395e01babda5bae02dfd6600645cb` | `a6a0b40f09d74445465ecfb9b411baf55d8f9266` | Autoridad B0--BF y consumo CAS. |
| P1b | `7fb44cc6c8bbfdd352476a7866c33fc03c2a72e1` | `24b62169051395e01babda5bae02dfd6600645cb` | Separación del ledger O3a. |
| P2 | `26db4bb4761448fe925e4fd3a68727ea46246c6d` | `7fb44cc6c8bbfdd352476a7866c33fc03c2a72e1` | Última barrera. |
| P3 | `e6f67ec62c9dee8ea4635d95e63640bd2278a3ae` | `26db4bb4761448fe925e4fd3a68727ea46246c6d` | Ticket y cierre únicos. |
| P4 | `e75dd5dbdf07faef405fd121fca3e58454ab9e4f` | `e6f67ec62c9dee8ea4635d95e63640bd2278a3ae` | Auto-STOP estable. |
| P5 | `ca9758987c46e022c7e21797a46675b8b18d8011` | `e75dd5dbdf07faef405fd121fca3e58454ab9e4f` | Identidad `/proc` exacta. |
| P6 | `d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28` | `ca9758987c46e022c7e21797a46675b8b18d8011` | Handoff conjunto opaco. |
| P7a | `c69a858b034208e824687abf23d2f4ebb7f4d496` | `d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28` | Conductor durable. |
| P7b | `c007678fbe1cce6f9ae9e0f61e8991214b113577` | `c69a858b034208e824687abf23d2f4ebb7f4d496` | AST, tipado y DAG. |
| P7c | `dccf6d9c2fca9ad079e69f5d7240cfb0d1292bdd` | `c007678fbe1cce6f9ae9e0f61e8991214b113577` | 131 mutantes. |
| P7B | `d1de33fc3246b5ebcabe2728deb181da849527c6` | `dccf6d9c2fca9ad079e69f5d7240cfb0d1292bdd` | O17 directo 65/EOF/0/0. |

La cadena es lineal, sin merges. P8 V2 no la integra.

## CI publicada

Las ejecuciones públicas siguientes corresponden al SHA exacto, `attempt=1`,
evento `push`, estado `completed/success` y cinco jobs `success`:

| Lote | CI | SHA |
| --- | --- | --- |
| P0 | [31392221513](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31392221513) | `a6a0b40f09d74445465ecfb9b411baf55d8f9266` |
| P1 | [31398536128](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31398536128) | `7fb44cc6c8bbfdd352476a7866c33fc03c2a72e1` |
| P2 | [31404754893](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31404754893) | `26db4bb4761448fe925e4fd3a68727ea46246c6d` |
| P3 | [31410901527](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31410901527) | `e6f67ec62c9dee8ea4635d95e63640bd2278a3ae` |
| P4 | [31416968307](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31416968307) | `e75dd5dbdf07faef405fd121fca3e58454ab9e4f` |
| P5 | [31422488306](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31422488306) | `ca9758987c46e022c7e21797a46675b8b18d8011` |
| P6 | [31428834333](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31428834333) | `d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28` |
| P7 | [31443778659](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31443778659) | `dccf6d9c2fca9ad079e69f5d7240cfb0d1292bdd` |
| P7B | [31448775349](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31448775349) | `d1de33fc3246b5ebcabe2728deb181da849527c6` |

Los jobs comunes son calidad, secretos, artefactos productivos y las puertas
PostgreSQL de contexto actor y Bolsa pública. No convierten PostgreSQL en
parte funcional de O3b.

## Ledger productivo P1--P6

| Fuente | Líneas | SHA-256 |
| --- | ---: | --- |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go` | 146 | `0b95fe0cfda784089c904941f56a3ad52a0eb519ef43b2057af22d24498c0e53` |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go` | 428 | `0499ede483615d57d5438d579f32580649b40fbf34b3be6bc26d31fa6c86c02d` |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go` | 230 | `4c3e7b888107a20b0ec2ae2479aa9a30378aad3a23b92ffc398597e6ce3dc484` |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go` | 157 | `99ae6fc4267062863f7e5adbccbe9e3698f08f114f107fd07f8cb6ed2f4af5f2` |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go` | 179 | `803b9a343934e5423e0d3f5f5f4cbfee86f7321790d979e017254a924ac2be7a` |
| `captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go` | 191 | `4b8b856e506b188a750922b4e320502b597a4aacf930507376eec9feccf4a534` |

Todas quedan por debajo de la parada local 750 y del tope DEC-051 de 800.

## Conductor y evidencia P7B

Autoridades:

- conductor `08e462544d4ca12b32ff6c766a201a9a325615542f5a6c5e9e38db2047932311`;
- matriz de casos `4b77892033510062cc869325195c6514816d08b83f92dd40dea911f4d038edca`;
- matriz de 22 fuentes `f09d79b014abc432bd02d5ee8062286cbf9957d0a4e2df40f632624700526b4d`;
- `SHA256SUMS` `0706dc1a5c6437ba5b93e4703fe402368b17f42765fbbf3a015b004a8e2fcf4f`;
- casos `76c7ac933ab1f7122a2bbe044cc2c89e77f5f2f710bcf3de77fc9d4fc007fb33`;
- ledger directo O17 `3ea7d82f814f56d84c41dc892e2c27a9a616adc7c1e314acc2f620a38f9f743a`;
- resumen `145afc7491ae79e3e58bf81b44f38afef17d5dbc840fd7d6c72bb371bb240cf5`.

La evidencia contiene 234/234 filas `GO`: 117 normal y 117 race. O01--O17
aparecen una vez por modo; O18 aporta 100 capturas por modo. Los cinco
inventarios —FD, hijos, zombis, grupos y temporales— tienen delta cero;
`residuos.txt` tiene cero bytes.

### Cierre causal de O17

P7B añade seis comprobaciones directas con IDs estables:

| ID | Modos | Resultado exigido |
| --- | --- | --- |
| `O17_BF_PIDFD` | normal/race | Sin referencia pidfd fiable: 65, EOF, no retorno, stdout=0, stderr=0. |
| `O17_BF_PARTICION` | normal/race | Partición irreversible: 65, EOF, no retorno, stdout=0, stderr=0. |
| `O17_BF_LEASE` | normal/race | Lease perdida: 65, EOF, no retorno, stdout=0, stderr=0. |

El conductor ejecuta directamente cada proceso hijo con su variable de entorno
y patrón exactos, redirige ambos flujos por separado y exige estado 65 y cero
bytes. El timeout no puede dar falso verde porque retorna 137. Los sentinelas
posteriores al BF retornan 3, de modo que 65 acredita ausencia de retorno. La
terminación y cierre de ambas redirecciones acreditan EOF. Las pruebas focales,
además, conectan dos `bytes.Buffer` antes de `Run` y verifican sus longitudes.
El `PASS\n` del harness exterior no se usa como prueba O17.

## Los 18 oráculos

| ID | Invariante |
| --- | --- |
| O01 | Entrada A6 lineal; nulos, aliases, clones y reuso cerrados. |
| O02 | Baseline, signo, registro, TID y contador final. |
| O03 | CONTROL: cancelar, EOF, parcial, framing, presupuesto, EINTR y error. |
| O04 | Borde de un segundo y tiempo positivo hasta bootstrap. |
| O05 | Tres referencias pidfd; pérdida y fatalidad sin referencia fiable. |
| O06 | Sin `F_DUPFD`, `pidfd_open` ni reapertura pidfd. |
| O07 | Trama exacta `PID_SUPERVISOR|TICKET\n` y escritor único. |
| O08 | Parciales, EINTR, cero, EPIPE y cierre único. |
| O09 | Auto-STOP T estable sin STOP emitido por Go. |
| O10 | `/proc/<PID>/stat` adversarial, acotado y exacto. |
| O11 | Señal pidfd cero con flag de grupo. |
| O12 | Precedencias CONTROL/señal/bootstrap/pidfd/STOP/identidad/inventario. |
| O13 | Terminalidad o cambio entre dos T nunca entrega. |
| O14 | Retirada post-ticket sin reemisión ni segundo cierre. |
| O15 | Handoff conjunto, sin writer y origen consumido. |
| O16 | O3c sin CONT, plazo 180 s, TERMINAL ni efecto. |
| O17 | BF 65, EOF/no retorno, cero stdout/stderr y sin E/S posterior. |
| O18 | Cien capturas por modo con cinco inventarios sin delta. |

Cada caso conserva ID, modo, comando, SHA del target, estado, tamaños de salida,
duración, inventarios, oráculo y resultado. No hay `SKIP` ni reintento.

## AST, tipado y DAG

El analizador tiene SHA
`68ce62adfe2f8f769e58ec7e5c865ee38a92e0babd5d908a290eb735f6bf686e`.
Su JSON vigente y el regenerado coinciden en
`c85cd1f94b636b6d4e9c95c6458ef2d621f8424939246a524fc743b5b1294aaf`:
seis fuentes, tipado `GO`, 55 definiciones, 161 aristas y once aserciones.

Las aserciones cubren entrada única/CAS, máquina total, ticket/cierre únicos,
Wait solo en retirada terminal, señal pidfd cero de grupo, `/proc stat` único y
acotado, dos T e identidad completa, handoff conjunto, O3c sin efectos, APIs
prohibidas y ownership acíclico sin dependencia de pruebas.

## Expansión mutante

El validador conjunto
`ecfb0a3567a7f081a66f83ec97173a20b4d4ac8e44f1e20fc8a1c19569f56ac9`
confirma 131 alternativas, 32 familias y cero supervivientes.

- B01--B18: runner `8e3095f45793ac3d64e2f8cfeebae291b26a6aa01515971fe1538e2edcc44714`,
  resultados `394521aab8277c7b497b82866dfdde51356c0e0fff8f43ec9717962f5d725cc4`,
  81/81 muertos —71 conductuales y diez AST de señalización—.
- B19--B30, B21A y B25A: runner
  `e415b818e1fe9652ef4c33e271127e3f7d7742f1c045d03c9dfe78811e2cda2d`,
  catálogo `d57529a9c3111d3cc0df9f201ab027e408e743db7b7f5562daa8ea1d861e85a4`,
  resultados `20a9e2d5f208337774b3d67d6e3def4c5247ed8c87215c299e18947827091b13`,
  50/50 muertos —14 conductuales, 33 AST/CFG y tres meta-B30—.

B25A muta la retirada O3a en copia efímera. B30 mata falsear residuos,
reutilizar un caso fallido y aceptar `SKIP`. Los ejecutores usan PGID exclusivo,
exigen `ESRCH` y no publican caches, rutas privadas o supervivientes.

## Auditoría requisito por requisito

| Requisito | Fuente autoritativa | Estado final |
| --- | --- | --- |
| B0--BF, CAS y consumo lineal | O01, AST, B01--B03 | Acreditado. |
| Lease/observador/contador/TID/PPID | O02/O12/O15, B02--B04/B21A | Acreditado. |
| CONTROL y límites 1024/4/4096/8 | O03/O12, B04--B05 | Acreditado. |
| Bootstrap y plazos no reiniciados | O04, B06/B23/B25A | Acreditado. |
| Tres pidfd sin duplicación | O05/O06/O11, B07/B08/B15 | Acreditado. |
| Ticket exacto y cierre único | O07/O08/O14, B09--B13 | Acreditado. |
| Auto-STOP sin señal Go | O09/O11, B14--B16 | Acreditado. |
| Parser e identidad `/proc` | O10/O13, B17--B20 | Acreditado. |
| Handoff conjunto B4T | O15, B21/B21A/B22 | Acreditado. |
| O3c sin efectos | O16, B23/B28 | Acreditado. |
| Retirada, Wait terminal y BF | O14/O17, B24--B26/B29 | Acreditado. |
| BF 65/EOF/no-retorno/0/0 | Seis filas directas P7B | Acreditado sin usar `PASS`. |
| Sin segundo owner/hook/goroutine | AST ownership, B27 | Acreditado. |
| Conductor y residuos | O18, B30 | Acreditado. |

## Reproducción

Desde checkout limpio en `d1de33fc3246b5ebcabe2728deb181da849527c6`:

```bash
(cd tools/o3b_p7_conductor/evidencia-p7b-o17-final-r1 && sha256sum -c SHA256SUMS)
go run ./tools/o3b_p7_ast -dir deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
tools/o3b_p7_mutantes/validar.sh
destino=$(mktemp -d); rmdir "$destino"
tools/o3b_p7_conductor/conductor.sh . "$destino"
go test ./...
go test -race ./...
go vet ./...
git diff --check
```

Las puertas focales normal/race, conductor completo, AST/DAG, mutantes,
checksums, gofmt, ShellCheck, vet y globales normal/race quedaron verdes. La
calidad local alcanzó las puertas Go y quedó limitada por el host Docker
(`exec /prueba: permission denied`); la CI P7B acreditó calidad y secretos 5/5.

## Fronteras cerradas

- O3c contiene solo el agregado opaco y no tiene consumidor operativo.
- O3b no emite CONT, no crea `finCaso=180s` ni produce efecto TERMINAL.
- O4a--O6, C4b-2, C4b, H0b, C2 y F0 siguen cerrados.
- No hay producción, despliegue, dato real ni decisión sobre personas.
- Las métricas oficiales no se incrementan.
- P8 V2 no modifica master, no fusiona y no toca la integradora.

## Revisiones independientes

La [revisión funcional](revision_funcional_f0_h0b_c4b2_g2o_o3b_2026-08-11.md)
y la [revisión de seguridad](revision_seguridad_f0_h0b_c4b2_g2o_o3b_2026-08-11.md)
releyeron el mismo snapshot y emitieron `GO`, `P0=P1=P2=0`. Ambas reprodujeron
los tres BF en normal y race: seis estados 65, `stdout=0`, `stderr=0`, EOF y
no retorno. Seguridad repitió además el conductor completo con 234 casos,
100+100 capturas, seis O17 directos, cinco inventarios sin delta y residuos
cero. Funcional regeneró AST/DAG y validó 131/131 mutantes en 32 familias.

La consulta anónima posterior de GitHub quedó limitada por cuota HTTP 403; no
se reinterpretó como verificación. Los enlaces y resultados exactos de CI de
la tabla proceden de la evidencia publicada de cada lote y el SHA remoto P7B
se reconfirmó. La CI propia de P8 V2 se comprobará después del push.

## Condición de publicación

El doble GO permite solo un commit documental pequeño, push normal de la rama
P8 V2 y comprobación de su CI 5/5. La integración y cualquier siguiente fase
quedan condicionadas a dirección.
