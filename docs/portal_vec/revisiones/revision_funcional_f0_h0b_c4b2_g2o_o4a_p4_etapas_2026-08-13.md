# Revisión funcional independiente O4A-P4-ETAPAS

Fecha: 13 de agosto de 2026.

Identificador: `O4A-P4-ETAPAS-FUNCIONAL`.

Estado: **NO-GO, P0=1, P1=2, P2=0**.

Este dictamen revisa exclusivamente el candidato
`2b7eaf498f8f68a90b66c004f166a62f070b5064` sobre la base exacta
`5345d5d097b51ab3567983f048feabeceaf2957b`. No revisa ni acredita ningún
descendiente, no corrige producto y no autoriza integración, publicación,
despliegue, producción ni cambio de métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4a-p4-funcional-20260813`, rama
`revision/o4a-p4-funcional-20260813`, sin editar ni mover la rama productora.
Su único write-set es esta acta.

Antes de revisar se leyeron completos `AGENTS.md`, el relevo de sesión del 29
de julio, mapa/roadmap, tablero, relevo de contratación temporal, matriz
normativa, expediente RRHH y hoja de ruta RRHH. También se releyeron completos
los contratos O1a, O3a, O3b, O3c, O4a y O4b, sus enmiendas y checkpoints
aplicables. Para esta frontera prevalecen la decisión O4a de causa, tiempo y
etapas y la delegación limitada de efectos a O4b.

## Snapshot exacto

La cadena lineal revisada es:

```text
5345d5d097b51ab3567983f048feabeceaf2957b
  -> 1e75c829215c43b4472908e9e00acc255aa016d9
  -> 4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9
  -> 1f8186cf0705043fea638db4ba1ab4ff086455a3
  -> 6a7a83b252a24971d1255f19c0e30ae7f4e4eb90
  -> 2b7eaf498f8f68a90b66c004f166a62f070b5064
```

El delta contra la base contiene solo dos altas:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go` | 547 | `42ad068a627e003b6d356b5488d1df8f464e55ade83fa566b0c7ee046be5ef7f` |
| `causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go` | 572 | `579b69e5adbf7498b992badc0af968344b60547d3167a942727c7354e61a6561` |

No cambiaron O3, O4a-P1--P3, O4b, runner, herramientas, workflows, SQL,
documentación transversal ni métricas. Ambos ficheros quedan por debajo del
tope duro de 800 líneas.

## Resultado requisito por requisito

| Requisito vigente | Resultado funcional |
| --- | --- |
| Consumo lineal, autoidentidad, owners O4A, estado 3/2, sellos y recursos exactos | Conforme en código y focales de autoridad/replay/carrera. |
| `SALIDA` o terminalidad natural llega A7 sin señal y crea drenaje natural | Conforme y cubierto. |
| Causa de extinción crea parada inicial/drenaje rápido y emite un STOP opaco | Conforme y cubierto para las seis causas cerradas. |
| STOP inicial estable crea gracia, parada final y drenaje cooperativo | Conforme y cubierto. |
| STOP inicial no estable/error converge a KILL rápido con incidente | El error está cubierto; `NO_ESTABLE` no tiene prueba focal exacta. |
| TERM/CONT conservan cardinalidad y raws; error converge a KILL cooperativo | Conforme y cubierto para error TERM y error CONT. |
| Terminalidad tras TERM/CONT y antes de gracia llega A7 sin otra señal | Conforme y cubierto. |
| Grupo vivo al vencer gracia autoriza `PARADA_FINAL` | No conforme en el borde exacto: el resultado observado en igualdad se rechaza fatalmente. |
| Terminalidad observada por `PARADA_FINAL` llega A7 con cero KILL | **No conforme**: O4b la borra y P4 no admite ese resultado. |
| STOP final estable con grupo presente autoriza KILL cooperativo | Conforme para `ESTABLE`; la prueba existente cubre esa rama. |
| STOP final no estable/error autoriza KILL e incidente | Implementado, pero `NO_ESTABLE` no tiene prueba focal exacta. |
| KILL intentado una vez llega A7, conserva causa/raw y no reintenta | Conforme y cubierto, incluido raw no cero. |
| Resultados forjados, replay, plazo/cardenalidad/etapa ajenos y borde de KILL fallan cerrados | Conforme en los casos ejercitados. |
| Cero syscall, señal, Wait, FD, red, SQL o API pública dentro de O4a-P4 | Conforme por inspección estructural del delta. |

## Hallazgos

### P0-01 — terminalidad en PARADA_FINAL se transforma en KILL

La decisión O4a es explícita: su tabla de etapas, fila 308, ordena
`PARADA_FINAL observa terminalidad -> A7`, con drenaje cooperativo y cero KILL.
O4b contradice esa autoridad en sus líneas 217--224 y 263--267: normaliza la
terminalidad de cualquier STOP a `NO_ESTABLE`, afirma erróneamente que O4a no
define esa arista y hace converger el resultado a KILL, aunque el propio O4b
declara que O4a conserva la autoridad sobre causa y etapa.

El candidato materializa la contradicción. En la rama `PARADA_FINAL`, líneas
478--490 del productivo, solo acepta `ESTABLE`, `NO_ESTABLE` o raw de error;
no acepta `TERMINAL`. `NO_ESTABLE` enclava incidente y, estable o no, siempre
emite `MATAR_GRUPO`. Por tanto, una terminalidad observada durante la parada
final produce un KILL no autorizado y altera además la evidencia de cierre.

No es una mera ausencia de test. Es una violación causal observable de la
autoridad O4a y bloquea el candidato.

Corrección accionable, fuera de esta revisión:

1. dirección debe fijar mediante una minitarea contractual separada la
   interfaz O4a/O4b, manteniendo distinta la semántica de STOP inicial y STOP
   final;
2. O4b debe poder preservar `TERMINAL` para `PARADA_FINAL` sin ampliar su
   autoridad de causa o etapa;
3. O4A-P4 debe aceptar ese resultado exacto solo para la parada final,
   consolidar A5→A7, fijar terminalidad y emitir cero KILL/incidente;
4. el material corregido requiere revisión funcional y de seguridad nuevas.

### P1-01 — el borde de gracia válido cae en fatalidad

O4a, fila 307, ordena que el grupo vivo **al vencer** gracia autorice
`PARADA_FINAL`. O4b comprueba el tiempo antes de CONT, pero sus líneas 229--233
no restringen la evidencia posterior `GRUPO_PRESENTE` a una observación
estrictamente anterior a `finGracia`.

P4 usa `marcaResultadoO4aM38(..., true)` tanto para `TERMINAL` como para
`GRUPO_PRESENTE` (líneas 457--458); esa función exige
`observado.Before(limite)` (líneas 382--386). Una presencia acreditada justo
en `finGracia`, que es la condición textual que abre la parada final, se trata
como resultado incompatible y entra en fatalidad en vez de autorizar STOP.

La corrección contractual debe fijar la clasificación de la evidencia
post-CONT que cruza o iguala el borde, y el código debe aplicar por separado el
borde estricto de terminalidad y el borde de presencia que abre
`PARADA_FINAL`.

### P1-02 — faltan oráculos focales para ramas obligatorias

La focal exacta contiene siete funciones `TestEtapasO4aP4*`, pero no construye
ningún resultado `evidenciaNoEstableO4bM38`. Tampoco contiene el oráculo
positivo de terminalidad en `PARADA_FINAL` ni presencia observada en la
igualdad de gracia. El caso terminal existente solo cubre TERM→CONT antes de
gracia.

Tras corregir el contrato y el producto deben añadirse oráculos y mutantes
específicos para: `NO_ESTABLE` inicial/final, terminalidad final con cero KILL,
presencia al borde de gracia y rechazo de `TERMINAL` en la parada inicial. Un
descendiente fuera del SHA objetivo no subsana ni acredita este candidato.

## Puertas reproducidas

Se ejecutaron sobre el SHA exacto:

```bash
mapfile -t fuentes < <(find deploy/postgresql/autorizacion_atestada_v3/pruebas_sql \
  -maxdepth 1 -type f -name '*contexto_actor_v1*.go' \
  ! -name '*_test.go' ! -name 'capturar_snapshot_*' -print | sort)
fuentes+=(deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go)
go test -count=20 -run '^TestEtapasO4aP4' "${fuentes[@]}"
go test -race -count=5 -run '^TestEtapasO4aP4' "${fuentes[@]}"
go vet "${fuentes[@]}"
gofmt -d deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go \
  deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..2b7eaf498f8f68a90b66c004f166a62f070b5064
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..2b7eaf498f8f68a90b66c004f166a62f070b5064'
```

Resultados: normal 20/20 verde; race 5/5 verde; vet, gofmt sin delta,
`git diff --check` y Gitleaks verdes. Gitleaks recorrió cinco commits y 48,82
KB sin hallazgos. También se comprobaron base, ancestry, write-set, líneas y
SHA-256.

Una primera ejecución global hecha como `root` no se considera evidencia:
falló en pruebas que exigen identidad no privilegiada y en `go list` con
sellado VCS. La repetición como el usuario de pruebas, con `safe.directory`
efímero, se detuvo manualmente tras una sucesión de paquetes verdes una vez
confirmado el P0; no se declara como puerta. Global race, calidad, PostgreSQL,
Docker, HTTP, E2E y despliegue no se ejecutaron porque no son proporcionales a
un candidato ya bloqueado y no sustituyen las puertas focales anteriores.

## Seguridad, privacidad y relevo

No se encontraron secretos, credenciales, datos personales, rutas privadas ni
salidas libres nuevas. El producto permanece interno, no serializable y sin
API. No se tocó master, integración, producción, porcentajes ni ramas ajenas.

El relevo inequívoco es **NO-GO**. `2b7eaf4` debe conservarse como candidato
histórico sin autoaprobación. La siguiente acción no es integrar: es acordar y
revisar independientemente la corrección contractual O4a/O4b de terminalidad y
borde de gracia; después, una minitarea productora O4A-P4 corregida con sus
focales y dos revisiones independientes.
