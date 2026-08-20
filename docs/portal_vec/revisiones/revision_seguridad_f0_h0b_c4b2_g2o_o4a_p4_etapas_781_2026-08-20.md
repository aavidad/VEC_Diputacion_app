# Revisión de seguridad O4A-P4-REVISION-781

Fecha: 20 de agosto de 2026.

Identificador: `O4A-P4-REVISION-781-SEGURIDAD`.

Dictamen: **NO-GO**, `P0=0`, `P1=2`, `P2=0`.

Esta revisión independiente cubre exclusivamente el candidato
`781bb5891ba3304bfed9d104e71d48846a5b6679`, con padre
`2b7eaf498f8f68a90b66c004f166a62f070b5064` y árbol
`707065f5e2d8a19fd8d37e79af889be52e9bf4b5`. No acredita descendientes,
O4b material, O4c, integración, publicación, CI, producción ni despliegue.

## Independencia, lectura y write-set

La revisión se realizó en el worktree exclusivo
`/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o4a-p4-seguridad-781-20260820`,
rama `revision/o4a-p4-seguridad-781-20260820`, sin editar la rama productora.
El único write-set revisor es esta acta.

Antes de editar se leyeron completos `/srv/fabrica/AGENTS.md`, `AGENTS.md`,
el relevo de sesión, mapa/roadmap, tablero y relevo de contratación temporal,
la matriz normativa, la especificación normalizada y la hoja de ruta RRHH.
También se releyeron completas las decisiones O4a, P1B, P1D y O4b, los dos
NO-GO con padre directo `2b7eaf4`, la enmienda O4AB-P0 V3 `2abba91`, su
checkpoint y sus dos GO directos.

V3 conserva condición local documental: no consta publicada ni existe CI 5/5
acreditada para ella. Ninguna rama remota local contiene `2abba91`. Sus dos GO
no autorizan usarla como autoridad material ni acreditan este candidato.

## Identidad, genealogía y alcance material

La genealogía desde la base publicada `5345d5d097b51ab3567983f048feabeceaf2957b`
es lineal y contiene seis commits:

```text
1e75c829215c43b447290290b681c9c3f35227b
4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9
1f8186cf0705043fea638db4ba1ab4ff086455a3
6a7a83b252a24971d1255f19c0e30ae7f4e4eb90
2b7eaf498f8f68a90b66c004f166a62f070b5064
781bb5891ba3304bfed9d104e71d48846a5b6679
```

El rango acumulado contiene exclusivamente dos altas Go del arnés:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go` | 547 | `42ad068a627e003b6d356b5488d1df8f464e55ade83fa566b0c7ee046be5ef7f` |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go` | 600 | `c6d3782e011885048f74b41f998906b7a8a8f3f7ccf9d64602b677c97a22912c` |

El delta `2b7eaf4..781bb58` modifica solo la focal: `+29/-1`. Añade las ramas
`NO_ESTABLE` inicial/final y los fatales de `TERMINAL` forjado en STOP inicial
y muestra estable exactamente en el borde. El productivo conserva byte a byte
el SHA revisado y rechazado en `2b7eaf4`.

## Hallazgos

### P1-01 — permanece el permiso de STOP/KILL tras terminalidad final

O4a, autoridad publicada de decisión, exige que terminalidad durante
`PARADA_FINAL` llegue A7 con cero KILL. El candidato sigue aceptando en esa
etapa únicamente cardinalidad 1 y las clases `ESTABLE`, `NO_ESTABLE` o error
raw (`etapas.go`, líneas 478--490). No admite `TERMINAL`, cardinalidad real 0
ni terminalidad posterior a STOP con cardinalidad 1. Toda salida controlable
de esa rama termina autorizando KILL; `NO_ESTABLE` y raw no cero enclavan
además incidente.

El nuevo caso `parada_final_no_estable_a_kill`, líneas 251--263 de la focal,
solo prueba esa semántica antigua. No cubre ni implementa la arista terminal
sin KILL. Un verde focal demuestra coherencia interna con el comportamiento
rechazado, no privilegio mínimo.

Impacto: O4a puede conceder un efecto posterior que su autoridad decisora
prohíbe después de terminalidad acreditable, y puede convertir el cierre
normal en incidente/cuarentena. Se mantiene P1: pidfd, identidad y lease
siguen cerrados y no se acreditó señal contra un proceso ajeno.

### P1-02 — borde de gracia y revalidación física siguen ausentes

El resultado post-CONT usa `marcaResultadoO4aM38(..., true)` tanto para
`TERMINAL` como para `GRUPO_PRESENTE` (líneas 454--474), por lo que exige
`observado < finGracia`; la igualdad continúa fatal en vez de admitir la
clasificación contractual correspondiente. Después de una presencia anterior,
`continuarEtapasEnO4aM38` (líneas 526--542) solo compara el reloj y emite
`PARADA_FINAL`: no consume una evidencia física nueva ni distingue
terminalidad sobrevenida.

La enmienda V3 describe el preflight condicional, cardinalidad máxima frente a
real 0/1, terminalidad sin STOP y la lectura final estricta antes del efecto.
Pero V3 no es antepasado del candidato, no está publicada y carece de CI 5/5
acreditada; además, el productivo `781bb58` no materializa ninguno de esos
puntos. El candidato no puede heredar documentalmente ni técnicamente su
corrección.

Impacto: una presencia antigua basta para emitir el permiso final aunque el
grupo haya terminado antes del borde. También se rechaza la terminalidad
válida en igualdad. Ambos resultados incumplen causalidad exacta y mínima
autoridad de señal.

## Controles conformes que no compensan los P1

- Consumo lineal de entrada, autorización y resultado; autoidentidad, vínculo,
  generación, etapa, cardinalidad, límite y rol pidfd cotejados.
- CAS one-shot, sin rollback, con ganador único en carreras de alias/replay.
- Owners O4A, lease 3, observador 2, TID, registro, generaciones, `pending`,
  cinco FD sellados e identidad completa verificados en fallo cerrado.
- Causa primaria inmutable e incidente de cierre separado mediante CAS único.
- Deadlines monotónicos absolutos, aritmética exacta 1/2/1/5 y bordes de KILL
  estrictos; no hay reinicio civil, timer ni espera.
- Raws separados, no negativos, historial acotado a cuatro y resultado opaco
  no serializable.
- Ausencia en el productivo de syscall de señal, señal cero, `Wait`, `waitid`,
  `wait4`, cierre/escritura, `/proc`, `pidfd_open`, duplicación, goroutine,
  canal, API exportada, log, HTTP, SQL o red.
- No aparecen secretos, credenciales, identidad humana ni datos personales en
  el delta. Gitleaks no está disponible y, por mandato, no se instaló ni se
  sustituyó por otra declaración.

## Gates reproducidos

La focal explícita se construyó con las 25 fuentes Go productivas `ignore` que
coinciden con `*contexto_actor_v1*.go`, excluyendo `capturar_snapshot_*`, más
la prueba `etapas_test.go`: 26 fuentes totales.

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
git diff --check 2b7eaf498f8f68a90b66c004f166a62f070b5064..781bb5891ba3304bfed9d104e71d48846a5b6679
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..781bb5891ba3304bfed9d104e71d48846a5b6679
```

Resultados: normal 20/20 y race 5/5 verdes; `go vet` focal verde; `gofmt -d`
sin salida; ambos `git diff --check` verdes. Identidad, padre, árbol, ancestry,
write-set, líneas, hashes y ausencia local de una rama remota que contenga V3
se reprodujeron.

No se ejecutaron gates globales, PostgreSQL, Docker, HTTP, E2E ni despliegue:
están fuera del alcance autorizado. Gitleaks se omite porque no está
disponible y no se autoriza instalarlo.

## Relevo corrector propuesto

El candidato permanece **NO-GO** y no debe integrarse. Primero dirección debe
publicar la enmienda V3 con sus dos GO sobre los mismos bytes, acreditar CI
5/5 y comunicar el SHA base publicado exacto. Sin esa dependencia no se abre
código corrector.

Después, la siguiente minitarea canónica propuesta es
`O4A-P4-V2-TERMINALIDAD-STOP`, con write-set máximo y exclusivo:

1. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go`;
2. su prueba existente `..._etapas_test.go`.

Debe sustituir, no acumular, los oráculos rechazados para mantener ambos
ficheros en la parada de 650 líneas. Su cierre observable es: resultado final
`TERMINAL` cardinal 0/1 llega A7 sin incidente, STOP ni KILL; terminalidad
post-CONT hasta igualdad gana; presencia anterior no autoriza por sí sola el
efecto del borde; las forjas de etapa/cardinal/raw/marca fallan cerradas; y
`PARADA_INICIAL` conserva su asimetría. No implementa O4b, syscall, Wait,
cierre, limpieza ni otra frontera.

La corrección material requerirá revisión funcional y de seguridad nuevas. El
productor no puede autoaprobarla. Esta acta no cambia estado transversal,
métricas, master, credenciales, publicación, producción o despliegue.
