# Checkpoint F0-H0b/C4b-2/G2-O — O4C-P0-CONTRATO

Fecha: 13 de agosto de 2026.

Estado: **CANDIDATO CORREGIDO A REVISIÓN**. No autoriza implementación O4c, O4A-P5,
O5/O6, integración, producción, despliegue ni cambio de métricas.

## Corte exacto

- Base: `5345d5d097b51ab3567983f048feabeceaf2957b`.
- Padre corregido: `64351a469df4e7bba4850ef1500d9e2c4bf378de`.
- Rama: `trabajo/o4c-p0-correccion-inventario-20260813`.
- Dependencias acreditadas: contratos O4a y O4b con doble GO; O4b publicado
  con CI `31546649383`, cinco de cinco puertas verdes.
- Decisión: [O4c terminalidad y limpieza](../decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md).
- Write-set corrector: decisión y este checkpoint, ambos Markdown; ningún otro
  fichero cambia respecto de `64351a469df4e7bba4850ef1500d9e2c4bf378de`.
- Código, pruebas, herramientas, runner, workflows, SQL, O3, O4a, O4b,
  `AGENTS.md`, handoffs, roadmap, ledger transversal y métricas: byte-inmutables.

La cadena candidata O4A-P4
`1e75c829215c43b4472908e9e00acc255aa016d9` ->
`4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9` ->
`1f8186cf0705043fea638db4ba1ab4ff086455a3` ->
`6a7a83b252a24971d1255f19c0e30ae7f4e4eb90` ->
`2b7eaf498f8f68a90b66c004f166a62f070b5064` no pertenece a esta base y
continúa pendiente de revisión independiente. El último SHA es el objetivo
exacto vigente de esa revisión; este documento no lo acredita.

La decisión candidata corregida tiene 555 líneas y SHA-256
`55e970da04933d7eb1287ec06d7b8f24767c25712d04d5111eedf6eb52b687bc`.
Las revisiones independientes deberán fijar exactamente esos bytes o emitir
NO-GO.

## Corrección P1 del inventario

La revisión funcional independiente del padre exacto emitió `NO-GO`,
`P0=0, P1=1, P2=0`, en
`109f11244d8a794f016961b59ac70e3ba03b1496`; la revisión de seguridad emitió
`GO`, `P0=P1=P2=0`, en
`f14afce00f07bb12d6d3b10a8fceb77b4cbd8707`. Este candidato no reescribe ni
integra esas actas.

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
cierre independiente de O4A-P4 y O4B-P5.

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

Se requieren revisión funcional y de seguridad independientes sobre los mismos
bytes, ambas `GO`, `P0=P1=P2=0`. El productor no crea esas actas, no se
autoaprueba, no integra, no hace push ni cambia porcentajes.

Siguiente acción: relevo a dos revisores independientes. Hasta entonces el
estado permanece candidato y la ruta autoritativa no está completa.
