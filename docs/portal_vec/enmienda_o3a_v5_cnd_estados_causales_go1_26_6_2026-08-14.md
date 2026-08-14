# Enmienda O3a V5: estados causales del conductor con Go 1.26.6

Fecha: 14 de agosto de 2026.

Tarea: `O3A-V5-CND-ESTADOS-CAUSALES-GO1.26.6`.

Estado: candidato técnico local; requiere revisión funcional y de seguridad
independientes. No acredita todavía el parche de toolchain, O4, publicación ni
CI remota.

## Base, antecedente y criterio único

La base exacta es el candidato de toolchain
`74f249587d5c0da41092a2c017ccd6cb54817248`, con padre
`d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c` y árbol
`bdbd0b2b05c43d7c14230a5c9b0a3114f44a4bb6`.

Ese candidato recibió dos `NO-GO`, `P0=0`, `P1=1`, `P2=1`:

- revisión funcional `ac1a95c25a0793e8ff0387fd5a186d54d9011b1b`;
- revisión de seguridad `ef40b1fd404eec0c2f6096f7d5d513c46a5d18dc`.

El P1 conserva una ejecución completa y sellada del conductor O3a V5 que
falló en `c15_c21` race, C21 índice 44, `TUPLA_C`, esperado 0 y observado 66,
con stdout y stderr vacíos, FD 5→5, residuos cero y checksums válidos. Dos
ejecuciones funcionales exactas fueron verdes. Esa divergencia demuestra
intermitencia y no se compensa por mayoría ni reintento.

El estado 66 agregaba siete causas de `TUPLA_C` y varias causas de los casos
lineales C16--C18. El criterio único de esta minitarea es que cada bifurcación
externa que antes terminaba en ese estado genérico devuelva un estado test-only
causal distinto, sin cambiar el oráculo, el orden, la cardinalidad, los cien
procesos C21, los plazos o la conducta productiva.

Esta minitarea no pretende estabilizar la causa todavía. Hace observable el
primer predicado que falla para que una corrección posterior tenga propietario
y prueba exactos.

## Corrección de una sonda ambiental descartada

Una reproducción diagnóstica posterior ejecutó
`conductor_c15_c21.sh` directamente bajo `flock`, sin pasar por
`conductor.sh`. Produjo C16 estado 66. La instrumentación efímera lo redujo a
`todosCLOEXEC=false` por FD 3: era el descriptor heredado del propio candado.

Ese resultado C16 queda descartado. La autoridad real
`conductor.sh` enumera `/proc/self/fd/*`, cierra todo FD mayor o igual que 3 y
solo entonces hace `exec` del bloque. El `NO-GO` C21 de la ejecución completa
no heredó ese descriptor y permanece válido.

Una muestra full-equivalent posterior cerró los FD de igual forma: seis
secuencias C21 terminaron verdes y la séptima cortó antes en `C18_BORDES`,
estado agregado 66, con salida vacía. Es otro síntoma de que el agregado
impedía localizar la causa; no es un pase ni una ráfaga usada para aprobar.

## Write-set exacto

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go
tools/o3a_v5_conductor/fuentes_v5.tsv
tools/o3b_p7_conductor/fuentes.tsv
tools/o3c_p6_conductor/fuentes.tsv
docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_go1_26_6_2026-08-14.md
```

Los dos fuentes son test-only por `//go:build ignore`. Los tres ledgers son
autoridades vivas que fijan exactamente sus hashes; su actualización es
mecánica e inseparable del cambio. No se modifican fuentes productivas,
conductores, casos, evidencias históricas, workflow, PostgreSQL, Bash
operativo, credenciales, estado transversal ni métricas.

## Estados causales

Los estados 99--113 son negativos test-only; ninguno es éxito ni se acepta
como sustituto del estado esperado:

| Estado | Causa exterior exacta |
| ---: | --- |
| 99 | preparación de netpoll previa a TUPLA |
| 100 | snapshot/conteo FD inicial de TUPLA |
| 101 | clase u origen de resultado TUPLA discrepante |
| 102 | limpieza de la fixture TUPLA |
| 103 | snapshot/conteo FD final de TUPLA |
| 104 | cardinal FD inicial/final discrepante |
| 105 | hijo restante después de TUPLA |
| 106 | caso C16: alias de preparado |
| 107 | caso C16: alias de retirada |
| 108 | caso C17: testigos/TID |
| 109 | caso C18: barrera parcial |
| 110 | caso C18: barrera de plazo |
| 111 | caso C18: vuelta tardía |
| 112 | selector lineal no enumerado |
| 113 | hijo restante después de C16--C18 |

Los estados de preparación 77, 78 y 93--98 ya eran causales y se conservan.
Los códigos contractuales 0, 65 y 72--76 tampoco cambian.

En `probarAliasRetiradaExternaO3aM38`, el fallo de
`escribirControlPruebaO3aM38` se asigna ahora a `err` antes de devolver. La
forma anterior podía devolver nil si la preparación había sido verde y la
escritura fallaba. Cerrar ese falso verde es parte de conservar la atribución
causal, no una relajación del caso.

## Presupuestos y ledgers

Los tamaños finales quedan por debajo de la parada local 750 y del tope duro
800 de DEC-051:

| Alias | Líneas | SHA-256 |
| --- | ---: | --- |
| G7a, pruebas de arranque | 742 | `4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7` |
| G7b, pruebas adversas | 741 | `084e5363ec969ef705caed7dcc213b5d7a54483574cf18bc2b0382fd6cccadf5` |

`fuentes_v5.tsv`, `tools/o3b_p7_conductor/fuentes.tsv` y
`tools/o3c_p6_conductor/fuentes.tsv` apuntan a esas dos huellas. No se
recalculan ni reescriben evidencias históricas que acreditaron otras huellas y
otras toolchains.

## Puertas requeridas

Antes de revisión independiente se exige:

1. identidad exacta, write-set, hashes, líneas y modos;
2. `gofmt`, `go vet` y builds normal/race de los diez fuentes;
3. una reproducción O3a V5 normal/race en clon limpio propiedad de
   `orquesta`, sin reintento y con cierre de FD equivalente al workflow;
4. conductores O3b P7 y O3c P6 con su toolchain histórica exacta 1.26.5,
   porque comparten los dos fuentes y sus ledgers vivos;
5. calidad global, `git diff --check` y Gitleaks por rango;
6. revisión funcional y de seguridad sobre el SHA exacto.

Una puerta roja conserva su estado causal y no se repite para buscar verde.
No se acepta `SKIP`, retry, sleep, mayor tolerancia, exclusión de FD o cambio
del estado esperado.

## Límites y relevo

Este corte no corrige aún el P2 documental del conteo dinámico de
vulnerabilidades: la base vigente encuentra seis, incluida
`GO-2026-5026`, y Go 1.26.6 elimina las seis. Esa corrección permanece como
minitarea separada.

Tampoco acredita la estabilidad del conductor: su objetivo es obtener la
causa exacta en la próxima ejecución canónica. El estado observado decidirá el
único propietario del siguiente parche. El candidato de toolchain y este
descendiente requieren nueva doble revisión antes de cualquier integración.

No se autoriza push, publicación, CI remota, despliegue, producción,
credenciales, cambio de seguridad ni actualización de porcentajes.
