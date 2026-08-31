# Checkpoint F0-H0b/C4b-2/G2-O — O4C-P0-CONTRATO

Fecha: 13 de agosto de 2026.

Estado: **CANDIDATO RECONCILIADO A REVISIÓN**. No autoriza implementación
O4c, O4A-P5, O5/O6, integración, producción, despliegue ni cambio de métricas.

## Corte exacto

- Base de reconciliación: `de2c9be8ea25c25a4e4173d1fdf6f5dcdfb769c8`.
- Genealogía fuente: candidata `de4ee5a11c3611e56ab09899da860ef88647ea0c`
  sobre `64351a469df4e7bba4850ef1500d9e2c4bf378de`.
- Rama: `trabajo/o4c-p0-reconciliacion-20260831`.
- Dependencias contractuales conservadas: contratos O4a y O4b con doble GO;
  O4b publicado con CI `31546649383`, cinco de cinco puertas verdes.
- Decisión: [O4c terminalidad y limpieza](../decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md).
- Write-set reconciliado: decisión y este checkpoint, ambos Markdown y ambos
  altas nuevas respecto de `de2c9be8ea25c25a4e4173d1fdf6f5dcdfb769c8`.
- Código, pruebas, herramientas, runner, workflows, SQL, O3, O4a, O4b,
  `AGENTS.md`, handoffs, roadmap, ledger transversal y métricas no se modifican.

La cadena O4A-P4 histórica y sustituida es
`1e75c829215c43b4472908e9e00acc255aa016d9` ->
`4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9` ->
`1f8186cf0705043fea638db4ba1ab4ff086455a3` ->
`6a7a83b252a24971d1255f19c0e30ae7f4e4eb90` ->
`2b7eaf498f8f68a90b66c004f166a62f070b5064`. La cadena vigente es
`99992cf44afb362a34f346ee2a5a5a8408ea7afd` ->
`6c56135b0eee69cff460e3df6562e0b557e0c49d`, integrada mediante
`01c5c1fb19458dff3794cae69c4777a5399dd60f`; este último commit es ancestro
de la base de reconciliación. Este documento no reabre ni acredita O4A-P4.

La decisión candidata reconciliada tiene 572 líneas y SHA-256
`13fe516ed9e1e27693226833e0b1c79ff46e7fe4500704e42fdfe840da00c404`.
Las nuevas revisiones independientes deberán fijar exactamente esos bytes o
emitir NO-GO.

## Corrección P1 del inventario

La revisión funcional independiente del padre fuente emitió `NO-GO`,
`P0=0, P1=1, P2=0`, en
`109f11244d8a794f016961b59ac70e3ba03b1496`; la revisión de seguridad emitió
`GO`, `P0=P1=P2=0`, en
`f14afce00f07bb12d6d3b10a8fceb77b4cbd8707`. La candidata fuente corregida
`de4ee5a11c3611e56ab09899da860ef88647ea0c` recibió después GO funcional en
`bebeff77d92ddfd8bec926c7a473557039b70ce2` y GO de seguridad en
`3839c8e9d330ca4584f91a399478cb15834e22c9`, ambos exclusivamente sobre sus
bytes. Este candidato reconciliado no reescribe ni integra esas actas y no
hereda sus GO.

La corrección sustituye la cardinalidad contradictoria por el conjunto
cerrado de cinco FD
`{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}` y sella el
mismo conjunto en OC02, OC16 y sus mutantes. `Cmd` y `Process` se anulan
después como referencias lógicas, pero no añaden un sexto FD al inventario.
Ninguna otra regla O4c cambia.

## Contrato fijado

O4c es el único propietario de terminalidad final, Wait funcional, drenaje,
`ECHILD`, `ESRCH`, TERMINAL y liberación. Consume owners O4C transferidos por
O4A-P5, conserva estados subyacentes observador 2/lease 3 y nunca señala,
decide causa, recrea tiempo o expone autoridad.

Orden positivo:

```text
terminalidad < finDrenaje
  -> cmd.Wait único
  -> Wait4 WNOHANG hasta ECHILD
  -> sonda de grupo ESRCH
  -> cerrar pidfd primario/reserva y CONTROL
  -> emitir/cerrar TERMINAL canónico
  -> inventario exacto
  -> liberar observador
  -> liberar lease como última capacidad
```

La trama reutiliza el codec O1a, usa nonce sellado, `os.Getpid()`, fase S3,
causa primaria O4a, identidad O3b y `1|1|0|1`. `SALIDA` conserva el estado
real 0/64/65/79; las demás causas usan su estado canónico. Incidente de cierre
previo o nuevo fuerza 65/cuarentena y no sustituye la causa ni fabrica un
terminal normal.

## DAG y bloqueos

```text
O4C-P0 -> P1-AUTORIDAD -> P2-TERMINAL-WAIT -> P3-DRENAJE
         -> P4-CIERRES-PREVIOS -> P5-TERMINAL -> P6-LIBERACION
         -> P7-CONDUCTOR -> P8-EVIDENCIA

O4A-P4 -> O4B-P1 -> ... -> O4B-P5
O4A-P4 + O4B-P5 + O4C-P0 -> O4A-P5 -> O4C-P1
O4C-P8 -> O5a
```

O4C-P1 no se abre con este candidato: necesita doble revisión, publicación y
CI de P0, además de O4A-P5 material acreditado. O4A-P5 sigue esperando el
cierre material de O4B-P5 y el cierre revisado de este O4C-P0. O4A-P4 vigente
ya está integrado y no constituye el bloqueo actual.

## Puertas del productor

Antes del commit local se deben reproducir:

1. SHA-256 y líneas de decisión/checkpoint y autoridades citadas;
2. existencia de todos los enlaces Markdown locales;
3. base exacta, rama exclusiva y write-set de solo dos altas;
4. búsqueda de secretos, tokens, rutas privadas y datos personales;
5. Gitleaks sobre el rango exacto y `git diff --check`.

No hay paquete Go afectado; focal normal/race, gofmt y vet no aportan evidencia
a este corte Markdown y se declaran no aplicables. Las puertas de código,
conductor y mutantes pertenecen a P1--P7.

## Revisión exigida

Se requieren una revisión funcional y otra de seguridad, nuevas e
independientes, sobre los mismos bytes y el nuevo hash, ambas `GO`,
`P0=P1=P2=0`. Los GO históricos de `de4ee5a` no se trasladan. El productor no
crea esas actas, no se autoaprueba, no integra, no hace push ni cambia
porcentajes.

Siguiente acción: relevo a dos revisores independientes. Hasta entonces el
estado permanece candidato y la ruta autoritativa no está completa.
