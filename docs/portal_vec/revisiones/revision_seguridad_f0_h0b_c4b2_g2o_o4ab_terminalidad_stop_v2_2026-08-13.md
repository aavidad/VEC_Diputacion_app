# Revisión de seguridad O4AB-P0-ENMIENDA-TERMINALIDAD-STOP V2

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4ab-p0-v2-seguridad-20260813`, sin edición del worktree ni de la
rama productora.

Dictamen: **GO**, `P0=0`, `P1=0`, `P2=0`.

## Corte exacto y alcance

Se revisó exclusivamente el candidato documental
`9de3ba320dfa4beede58e5a1d34aa02a3459f073`, con padre V1
`a89a3228554f53b32f5d81fc8b0438835f35b0f6`, base
`5345d5d097b51ab3567983f048feabeceaf2957b` y árbol
`d44580e843d89a9311d9cbd322f3499f5f162b54`. Su delta V2 modifica únicamente
dos Markdown, con 72 inserciones y 28 borrados:

| Documento | Delta V2 | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+47/-17` | 264 | `f53ee8711eb64d7dc999e2f67154f874de7d80bfbc2e18a02c413febd04d1e15` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md` | `+25/-11` | 115 | `b380505c14de57fe0a8b1355212b55fd028971d12fb971000214b96da4ca89fe` |

Se releyeron completos la enmienda V2, su checkpoint y las decisiones
publicadas O4a y O4b:

| Autoridad | Líneas | SHA-256 |
| --- | ---: | --- |
| `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md` | 535 | `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc` |
| `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4b_senales_grupo_2026-08-12.md` | 443 | `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f` |

También se releyeron las cuatro actas `NO-GO` que abren esta V2:

| Objeto | Tipo | Commit | Líneas | SHA-256 | Resultado |
| --- | --- | --- | ---: | --- | --- |
| O4A-P4 `2b7eaf4` | funcional | `e20a5fe597d9c172d59b99e75878f8e99e196c76` | 176 | `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48` | `NO-GO`, `P0=1,P1=2,P2=0` |
| O4A-P4 `2b7eaf4` | seguridad | `a62ee60f70315a1fc0d290290b681c9c3f35227b` | 147 | `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB-P0 V1 `a89a322` | funcional | `cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` | 160 | `76ee47785d27a01f61c07e750e021a88abee701a713b5bd1516cb8f97b7466cf` | `NO-GO`, `P0=0,P1=1,P2=0` |
| O4AB-P0 V1 `a89a322` | seguridad | `e7e06423941807a41c40946e5a4af12e1b30bb11` | 133 | `e968004722ebd94a86c6a1450ee46baef811067a0372ecee99f979a4d2072a72` | `NO-GO`, `P0=0,P1=1,P2=0` |

La lectura obligatoria y las autoridades O3 se conservaron del mismo contexto
de revisión y se revalidaron. Código, pruebas, estado transversal, métricas y
candidatos históricos permanecen inmóviles. Este dictamen no acredita código
ni descendientes.

## Cierre del P1 de V1: deadline y linealización

V1 podía comprobar `ahora < finParadaFinal`, consumir tiempo en los sondeos de
primario, reserva e identidad y ejecutar después un STOP ya tardío. V2 elimina
esa ejecución admitida:

1. consume el permiso one-shot, valida custodia y hace el preflight físico con
   permisos lease separados;
2. solo después de consolidar toda la evidencia necesaria de presencia hace
   exactamente una lectura monotónica final;
3. igualdad o vencimiento respecto de `finParadaFinal` llevan a OBF directo;
4. únicamente una lectura estrictamente anterior fija conjuntamente presencia
   y vigencia en `ahoraFinal`; el sobre ya está preasignado, se prepara el
   permiso lease y STOP es la siguiente syscall literal.

La lectura final es parte del preflight, no una operación intercalada después
de que la presencia haya autorizado el efecto. Después de ella quedan
prohibidos otro reloj, sonda, validación, espera, log o asignación falible.
Por tanto V2 satisface simultáneamente la comprobación final y la inmediatez
exigida por O4b. Emplea el mismo criterio de linealización ya publicado para
CONT y KILL: lectura estrictamente verde y syscall siguiente, sin recrear ni
extender el límite.

Si el reloj final cruza el borde, no se sella ni devuelve resultado parcial:
la autorización permanece consumida; no existen cardinalidad, raw, marca o
incidente ordinarios; O4a no toma otra transición desde A5; y rige la fatalidad
65/EOF/stdout=0/stderr=0, sin señal, cierre, log, limpieza ni efecto posterior.
Esto conserva denegación predeterminada y evita convertir demora o
indisponibilidad en permiso. Los oráculos E05/E06 y los mutantes de omisión,
adelanto, inversión, falseo, igualdad, resultado vencido y operación
interpuesta hacen observable la corrección.

## Autoridad mínima, terminalidad y cardinalidad

- O4a sigue poseyendo causa, precedencia, límites, etapa y autorización; O4b
  acredita condición física y ejecuta únicamente el efecto autorizado. La
  cláusula de prevalencia sustituye solo las frases O4b incompatibles con las
  filas terminales ya existentes en O4a.
- Una presencia observada antes de gracia no se reutiliza al llegar al borde.
  `PARADA_FINAL` vuelve a acreditar primario, reserva e identidad bajo lease;
  duda, discordancia, flags o consolidación inciertos son OBF, nunca presencia.
- Terminalidad durante el preflight produce cero STOP, cero KILL, cero
  incidente y cardinalidad real 0. Los raws cero son campos canónicos ausentes,
  no un éxito fabricado. Una marca fuera de `finParadaFinal` sigue siendo
  rechazada por O4a, sin habilitar señal.
- Presencia y lectura final verde permiten como máximo un STOP. Tras STOP,
  terminalidad acreditada produce cardinalidad 1 y A7 sin KILL ni incidente.
  Estabilidad, no estabilidad y raw no cero conservan sus ramas publicadas.
- `cardinalidadMaxima=1` permanece en la autorización y se distingue de la
  cardinalidad real 0/1. Alias, replay, etapa, deadline, cardinalidad o raw
  forjados fallan cerrados; no hay clon, rollback ni serialización del permiso.
- `PARADA_INICIAL` conserva su asimetría: intenta un STOP, cardinalidad 1,
  terminalidad controlable normalizada a `NO_ESTABLE` y posterior convergencia
  a KILL rápido. No acepta `TERMINAL` ni cardinalidad 0.
- No se introducen Wait/waitid/wait4, señal cero, PID/PGID, `pidfd_open`,
  duplicación, fallback, reintento, bucle, parser, FD, goroutine, cierre,
  limpieza, log, dato personal, secreto, HTTP, SQL, red o producción.

El DAG permanece cerrado y secuencial: esta V2 requiere doble revisión,
publicación y CI 5/5; después puede existir un nuevo candidato O4A-P4; O4B-P1,
O4A-P5, O4C-P1 y O5/O6 siguen bloqueados. La corrección O4C-P0 es independiente
y no acredita este corte.

## Puertas reproducidas

- `git rev-parse HEAD HEAD^ 'HEAD^{tree}'`: candidato, padre y árbol exactos;
- `git merge-base --is-ancestor` para base→V2 y V1→V2: verde;
- `git diff --name-status`, `--stat` y `git show --name-only`: dos Markdown,
  `+72/-28`, sin otro fichero;
- `sha256sum` y `wc -l`: huellas y líneas de enmienda, checkpoint, O4a, O4b y
  las cuatro actas `NO-GO` exactas;
- enlaces Markdown locales de enmienda/checkpoint: verdes;
- `git diff --check` en
  `a89a3228554f53b32f5d81fc8b0438835f35b0f6..9de3ba320dfa4beede58e5a1d34aa02a3459f073`
  y en
  `5345d5d097b51ab3567983f048feabeceaf2957b..9de3ba320dfa4beede58e5a1d34aa02a3459f073`:
  verde;
- búsqueda focal de secretos, credenciales, DSN y datos personales: solo
  menciones normativas;
- Gitleaks V1→V2: un commit, 5,28 KB, cero filtraciones;
- Gitleaks base→V2: dos commits, 21,85 KB, cero filtraciones;
- worktree candidato limpio antes de añadir esta acta.

Go normal/race, gofmt, vet, PostgreSQL y E2E no aplican a un delta exclusivo
de documentación. No se declaran ejecutados ni se sustituyen por las puertas
documentales.

## Cierre

El candidato documental exacto recibe **GO**, `P0=P1=P2=0`. Este es solo uno
de los dos dictámenes independientes requeridos. Dirección debe verificar el
otro dictamen sobre los mismos bytes antes de publicación y CI 5/5. No se
autoriza por esta acta código O4A-P4, O4B-P1, O4A-P5, O4c, O5/O6, integración,
push, despliegue, producción ni cambio de métricas.
