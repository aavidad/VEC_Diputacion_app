# Revisión funcional O4C-P0-CORRECCIÓN-INVENTARIO

Fecha: 13 de agosto de 2026.

Identificador: `O4C-P0-CORRECCIÓN-INVENTARIO-FUNCIONAL`.

Estado: **GO, P0=0, P1=0, P2=0**.

Este dictamen independiente revisa exclusivamente el candidato
`de4ee5a11c3611e56ab09899da860ef88647ea0c`, padre exacto
`64351a469df4e7bba4850ef1500d9e2c4bf378de` y árbol
`6be07800da0f90f1bd61f370ecfe57c9a9f6d108`. No revisa ni acredita
descendientes, O4A-P4, O4A-P5, O4C-P1, O5/O6, integración, publicación,
producción, despliegue ni cambio de métricas.

## Independencia, lectura y write-set

La revisión se realizó desde el worktree exclusivo
`o4c-p0-correccion-inventario-funcional-20260813`, rama
`revision/o4c-p0-correccion-inventario-funcional-20260813`, creada en el SHA
objetivo. No se editó ni movió la rama productora
`trabajo/o4c-p0-correccion-inventario-20260813`. El único write-set de esta
revisión es la presente acta.

Antes de editar se releyó `AGENTS.md` completo. Se comprobaron byte a byte y
se conservaron las lecturas obligatorias ya completadas en esta misma sesión:
relevo del 29 de julio, mapa/roadmap, tablero, relevo de contratación temporal,
matriz normativa, expediente RRHH y hoja de ruta RRHH. También se releyeron
la decisión O4c corregida y su checkpoint completos y se recontrastaron O1a,
O3a, O3b, O3c, O4a, O4b y sus evidencias aplicables.

## Identidad y delta exactos

El commit tiene un único padre y el productor permanece limpio. El rango
incremental contiene exactamente dos modificaciones Markdown, 32 inserciones
y 11 borrados:

| Fichero modificado respecto de `64351a4` | Delta | Líneas finales | SHA-256 final |
| --- | ---: | ---: | --- |
| `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md` | `+9/-6` | 555 | `55e970da04933d7eb1287ec06d7b8f24767c25712d04d5111eedf6eb52b687bc` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md` | `+23/-5` | 115 | `9132d26c3fd2cb1e346ca6d67bef97535472ab3b628af273d7d0402301275746` |

No cambian código, pruebas, herramientas, runner, workflows, SQL, O3, O4a,
O4b, `AGENTS.md`, handoffs, roadmap, ledger transversal ni métricas. En el
rango acumulado desde la base normativa
`5345d5d097b51ab3567983f048feabeceaf2957b` siguen siendo exactamente dos
altas; en el rango de corrección son exactamente las dos modificaciones
anteriores. Las dos afirmaciones del checkpoint son por ello compatibles.

Los siete hashes de autoridad declarados coinciden:

```text
O4a             ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc
checkpoint O4a  0ba877cd831f7957d36cab885644dd6f8e5069d80e1629669b92d282e99d1513
O4b             675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f
checkpoint O4b  1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45
O3c             f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6
ledger O3c      a1de39d1c80492cae8b6b858a096d8f3051b346913064fe51a83dd6573dbb3b1
O1a             29c1520f6aab91d832ec6ddb41efd75df587d240b8fb1e7dc51f107df831a659
```

También existen y corresponden al padre exacto las actas citadas: funcional
`109f11244d8a794f016961b59ac70e3ba03b1496`, con `NO-GO P1=1`, y seguridad
`f14afce00f07bb12d6d3b10a8fceb77b4cbd8707`, con `GO P0=P1=P2=0`. Esta
revisión no las integra ni convierte la revisión de seguridad del padre en una
revisión de los bytes corregidos.

## Resolución del hallazgo previo

El P1 del padre queda resuelto sin ambigüedad:

| Punto exigido | Evidencia en el candidato corregido |
| --- | --- |
| Cardinalidad física | La secuencia 328--330 exige exactamente cinco FD. |
| Conjunto cerrado | `{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}`. |
| Consumo/cierre | Wait consume solo `pidfdOpaco`; los otros cuatro se cierran una vez en el orden fijado. |
| Referencias lógicas | `Cmd` y `Process` se anulan después y no cuentan como sexto FD. |
| Entrada OC02 | Exige owners, estados, sellos y los cinco FD exactos. |
| Salida OC16 | El snapshot elimina exactamente esos cinco, sin FD nuevo o ajeno. |
| Mutantes OC02/OC16 | Deben morir omisión de cualquiera de los cinco, duplicación, reordenación, cuenta de seis o cambio ajeno. |

La corrección no altera orden de Wait, ECHILD, ESRCH, cierres, inventario,
observador y lease; tampoco cambia causa, estado, tiempos, codec, cuarentena ni
resultado privado.

## Auditoría funcional completa

| Requisito del contrato O4c completo | Resultado |
| --- | --- |
| O4c posee solo terminalidad final, Wait, drenaje, cierres, TERMINAL e inventario | Conforme; no invade causa/tiempo/etapas O4a ni señales O4b. |
| Agregado O4A-P5 one-shot, A8 y owners O4C con estados observador 2/lease 3 | Total, opaco y fail-closed. |
| Límite absoluto de una única rama, monotónico, no recreado, con borde estricto | Conforme. |
| Primario y reserva concuerdan terminalidad antes del único `cmd.Wait` | Conforme; reserva no sustituye ni recibe señal. |
| Wait acepta solo `nil`/0 o `ExitError` propio con ProcessState y conserva estado real | Conforme con la tabla O1a de causa/estado. |
| Wait4 WNOHANG hasta `ECHILD` y luego sonda de grupo solo `ESRCH` | Conforme; PID cero, EPERM, EINTR y duda no acreditan cierre. |
| Secuencia de cinco consumos/cierres e inventario físico exacto | Conforme tras esta corrección. |
| TERMINAL canónico único; parciales/EINTR acotados; incidente implica cuarentena | Conforme, sin sustituir la causa histórica. |
| Observador se libera antes y lease es la última capacidad | Conforme; no hay efecto posterior ni rollback. |
| Resultado OC7 privado y one-shot sin PID/pidfd/nonce/ticket/error libre | Conforme. |
| DAG P0→P8 y write-sets acotados | Conforme; O4C-P1 sigue bloqueado hasta O4A-P5 y cierre documental. |
| Sin señal, pidfd nuevo, fallback PID/PGID, segundo Wait, API, SQL, HTTP o red | Conforme y explícito. |
| Matriz OC01--OC22 y mutantes OC01--OC23 cubren causalidad, bordes y residuos | Conforme como contrato para P1--P7. |

La contradicción O4a/O4b detectada en la revisión separada de O4A-P4 sigue
siendo un bloqueo aguas arriba. No es autoridad de O4c corregirla y este GO no
la oculta, no acredita `2b7eaf4` y no desbloquea O4A-P5 ni O4C-P1.

## Puertas reproducidas

```bash
git show -s --format='%H%n%P%n%T%n%s' de4ee5a11c3611e56ab09899da860ef88647ea0c
git diff --name-status 64351a469df4e7bba4850ef1500d9e2c4bf378de..de4ee5a11c3611e56ab09899da860ef88647ea0c
git diff --numstat 64351a469df4e7bba4850ef1500d9e2c4bf378de..de4ee5a11c3611e56ab09899da860ef88647ea0c
git diff --name-status 5345d5d097b51ab3567983f048feabeceaf2957b..de4ee5a11c3611e56ab09899da860ef88647ea0c
wc -l docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md
sha256sum docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md \
  docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md
git diff --check 64351a469df4e7bba4850ef1500d9e2c4bf378de..de4ee5a11c3611e56ab09899da860ef88647ea0c
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..de4ee5a11c3611e56ab09899da860ef88647ea0c
gitleaks git --no-banner --redact \
  --log-opts='64351a469df4e7bba4850ef1500d9e2c4bf378de..de4ee5a11c3611e56ab09899da860ef88647ea0c'
gitleaks git --no-banner --redact \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..de4ee5a11c3611e56ab09899da860ef88647ea0c'
```

Resultados: identidad, padre, árbol, rama, productor limpio, delta incremental
`32/11`, write-set incremental `M/M`, write-set acumulado `A/A`, líneas,
hashes, siete autoridades, actas citadas, enlaces locales y ambos
`git diff --check` verdes. Gitleaks recorrió el commit corrector (1,84 KB) y el
rango acumulado de cinco commits (34,44 KB), sin hallazgos. La búsqueda focal
de tokens, claves, DSN y credenciales tampoco encontró material sensible.

No existe paquete Go afectado: focal normal/race, gofmt, vet, mutantes,
PostgreSQL, Docker, HTTP y E2E no aplican a esta corrección Markdown. No se
declaran ejecutados ni se usan para ampliar el GO.

## Cierre y relevo

La corrección funcional del inventario es completa y no queda hallazgo P0, P1
o P2 en los bytes exactos revisados. Este GO es una sola revisión independiente;
el candidato necesita todavía revisión de seguridad sobre `de4ee5a`,
integración/publicación por dirección y CI 5/5 conforme a su contrato.

No se hizo push, deploy, cambio de credenciales, estado transversal,
porcentajes ni producción. O4C-P1 continúa bloqueado por O4A-P5 incluso
después de un eventual doble GO documental de este P0.
