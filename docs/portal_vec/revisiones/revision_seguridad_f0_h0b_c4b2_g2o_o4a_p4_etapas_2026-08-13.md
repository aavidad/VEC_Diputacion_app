# Revisión de seguridad O4A-P4-ETAPAS

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4a-p4-seguridad-20260813`, sin edición del worktree ni de las ramas
productoras.

Dictamen: **NO-GO**, `P0=0`, `P1=1`, `P2=0`.

## Corte exacto y alcance

Se revisó exclusivamente el candidato
`2b7eaf498f8f68a90b66c004f166a62f070b5064`, sobre la base publicada
`5345d5d097b51ab3567983f048feabeceaf2957b`. Su árbol es
`0456fb928390d587de46e9854d5a525601d985cb4` y la cadena candidata exacta es:

```text
1e75c829215c43b4472908e9e00acc255aa016d9
4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9
1f8186cf0705043fea638db4ba1ab4ff086455a3
6a7a83b252a24971d1255f19c0e30ae7f4e4eb90
2b7eaf498f8f68a90b66c004f166a62f070b5064
```

El rango añade únicamente:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go` | 547 | `42ad068a627e003b6d356b5488d1df8f464e55ade83fa566b0c7ee046be5ef7f` |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go` | 572 | `579b69e5adbf7498b992badc0af968344b60547d3167a942727c7354e61a6561` |

Ambos respetan la parada de 650 líneas y el máximo DEC-051 de 800. Ningún
descendiente del SHA objetivo forma parte del objeto, evidencia o dictamen y
no se corrigió código de producción.

## Hallazgo P1 — las aristas terminales autoritativas quedan inejecutables

O4a es el propietario exclusivo de causa, precedencias, etapas, subplazos y
autorizaciones (líneas 38--45). Su tabla total, líneas 295--312, exige
simultáneamente:

- TERM y CONT consolidados con terminalidad antes de gracia: A7 y cero señal;
- solo grupo vivo al vencer gracia: autorización de `PARADA_FINAL`;
- terminalidad observada durante `PARADA_FINAL`: A7 y cero KILL.

El contrato O4b reconoce en sus líneas 41--45 que, ante contradicción,
prevalece O4a para toda decisión. Sin embargo, sus líneas 217--224 y 263--267
afirman que O4a no define la segunda arista, normalizan terminalidad de STOP a
`NO_ESTABLE` y obligan convergencia a KILL. La afirmación es incompatible con
la tabla O4a vigente. El documento de efecto O4b no puede suprimir una
transición del documento propietario de decisión.

El candidato materializa esa contradicción en dos puntos:

1. tras un resultado TERM/CONT `GRUPO_PRESENTE` (productivo, líneas 454--475),
   conserva A3, pero `continuarEtapasEnO4aM38` (líneas 526--542) solo compara
   el reloj con `finGracia` y autoriza `PARADA_FINAL`; no existe nueva evidencia
   no recolectora que demuestre que el grupo continúa vivo en el instante de
   gracia. Una terminalidad ocurrida entre la observación inmediata de O4b y
   el vencimiento no puede tomar A7;
2. el consumidor de `PARADA_FINAL` (líneas 478--490) solo admite `ESTABLE`,
   `NO_ESTABLE` o raw de error y, para los tres, autoriza KILL. No admite
   `TERMINAL` ni puede tomar la arista A7 sin KILL exigida por O4a.

La focal, líneas 176--188, fija precisamente el camino con evidencia de
presencia anterior a gracia y emite STOP al llegar al borde. Sus líneas
233--240 solo prueban terminalidad recibida inmediatamente después de
TERM/CONT; no prueban terminalidad sobrevenida antes de gracia ni terminalidad
observada por `PARADA_FINAL`.

### Impacto de seguridad

El flujo puede otorgar permisos one-shot para STOP y después KILL cuando la
autoridad decisora exigía entrega a O4c sin otra señal. También convierte una
terminación normal de esa ventana en `INCIDENTE_DE_CIERRE`/cuarentena mediante
la normalización `NO_ESTABLE`. Se vulneran privilegio mínimo, causalidad exacta
y la política de cero señal posterior a terminalidad. Se clasifica P1, no P0,
porque O4b mantiene pidfd primario, identidad y lease acreditados y la duda
física sigue fallando cerrada; no se acreditó señalización de un proceso ajeno
ni fuga de autoridad.

### Corrección requerida

Antes de volver a revisar código debe reconciliarse el contrato aplicando la
prevalencia ya fijada. O4b debe poder devolver terminalidad de STOP, o debe
definirse explícitamente una observación no señalizadora nueva que acredite
presencia al vencer gracia; O4a-P4 debe consumir terminalidad en ambas ventanas
y converger A7 sin STOP/KILL. La corrección necesita focales y mutantes para
los dos bordes. Si dirección pretendiera retirar esas aristas de O4a, haría
falta una decisión normativa nueva y revisada: no basta con que O4b afirme que
no existen.

## Controles que sí quedan acreditados

- denegación predeterminada ante nulo, clon, alias, replay, sello, owner,
  generación, TID, recurso, huella, estado o límite adversos;
- autorizaciones/resultados privados preasignados, opacos y one-shot, con CAS
  de consumo y carreras focales de ganador único;
- causa primaria inmutable, latch de incidente separado, deadlines absolutos
  monotónicos y bordes estrictos sin recomputación civil;
- ausencia en el productivo P4 de syscall, señal, Wait/wait4/waitid, cierre,
  escritura, `/proc`, goroutine, canal, timer, log o API exportada;
- CONTROL, TERMINAL, tres referencias pidfd, custodia, identidad y
  `finBootstrap` cotejados; el análisis AST impide mutar este último en P1--P4;
- raws no negativos, cardinalidades/etapas/clases cerradas, historial máximo
  cuatro, permisos pendientes únicos y cero secreto, credencial o dato humano;
- Gitleaks sin hallazgos en los cinco commits del rango exacto.

Estos controles no compensan la transición P1 ausente.

## Reproducción

Se construyó la unidad focal con los 40 ficheros Go `ignore` del arnés, dejando
fuera únicamente el capturador probatorio con build normal. Resultados:

- `gofmt -d` sobre los dos ficheros del candidato: verde, salida vacía;
- focal `^TestEtapasO4aP4`, `-count=100`: verde;
- la misma focal, `-race -count=100`: verde, sin carrera reportada;
- `go vet` sobre las 40 fuentes explícitas: verde;
- `go test ./... -count=1 -timeout 20m` y `go vet ./...`: verdes;
- `git diff --check`: verde;
- Gitleaks sobre
  `5345d5d097b51ab3567983f048feabeceaf2957b..2b7eaf498f8f68a90b66c004f166a62f070b5064`:
  5 commits, 48,82 KB, cero filtraciones.

La puerta completa `scripts/verificar_calidad.sh` no terminó verde: después de
`go mod verify` y del `go test ./...` normal verdes, su `go test -race ./...`
falló en un paquete ajeno al write-set,
`internal/modules/contrataciontemporal/application`, caso
`TestDecidirCoberturaSQLStatePrimarioNoReintentaNiPublicaExito/40001`, con
`SQLSTATE 40001 publicó un resultado`. Por el `set -e`, no alcanzó las fases
posteriores a la carrera. El caso ajeno pasó aislado con `-race -count=1` y
también `-race -count=20`; no se modificó ni se clasifica desde esta minitarea.
El `go vet ./...` ya se había ejecutado por separado y quedó verde.

Las pruebas verdes demuestran consistencia interna con la semántica
implementada, no conformidad con las dos aristas normativas omitidas. El fallo
global queda registrado como gate no verde adicional y no se usa para aumentar
ni reducir la severidad del P1 propio del candidato.

## Cierre

El candidato exacto permanece **NO-GO**. No se autoriza O4B-P1, O4A-P5,
integración, publicación, despliegue, producción ni cambio de métricas. La
siguiente tarea causal es una corrección contractual explícita O4a/O4b y,
después, un nuevo candidato O4A-P4 revisado de forma independiente.
