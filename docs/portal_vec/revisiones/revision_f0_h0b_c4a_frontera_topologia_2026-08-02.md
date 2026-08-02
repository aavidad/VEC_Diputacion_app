# Revisión F0-H0b C4a: frontera, topología y H0 nominal

Fecha: 2 de agosto de 2026.

## Resultado

El checkpoint C4a finaliza en `bb5e668` con revisiones independientes finales
`GO` y `P0=0`, `P1=0`, `P2=0` dentro de su alcance.

C4a acredita frontera runner/adaptador, topología H0, identidades anteriores a
los seams, sustitución nominal→error, captura privada de cuatro, H0 nominal
PostgreSQL 18.4 y residuos cero. No acredita C4b, C4c, C4d, la matriz M38, H0b
global, C2, F0 ni producción.

## Cadena de código

El código acumulado desde `11b237a` se conserva en seis commits acotados:

1. `5c68c0d`: primitivas retornables;
2. `ab5cc95`: plantillas y wrapper privado;
3. `a8db21a`: oráculo independiente del wrapper;
4. `25b17cf`: flujo R0 funcional;
5. `e60dc2b`: frontera y topología H0;
6. `bb5e668`: activación M38 posterior al preámbulo.

`e60dc2b` consolida el trabajo en curso previamente preservado bajo la decisión
de presupuesto; no se presenta como una minitarea pequeña.

El write-set acumulado es exactamente:

- runner PostgreSQL ContextoActor V3;
- auxiliar privado H0b;
- adaptador privado de recursos M38/H0b.

D2c, D2d y el capturador Go permanecen byte a byte inmutables.

## Revisión del primer candidato C4a

`e60dc2b` obtuvo dos revisiones C4a verdes, incluida una H0 independiente, pero
la revisión de seguridad emitió `NO-GO`, `P0=0 P1=1 P2=0`.

El hijo activaba traza e inyección antes del preámbulo sin R0. Su primer
finalizador consumía la inyección, contaminaba traza/recuperación e impedía que
A/N/E/F alcanzasen su seam real. El candidato no se integró.

`bb5e668` mueve la activación al punto posterior al finalizador preliminar y al
reinicio del ledger. Limpia recuperación, traza y observación antes del caso
principal. También evita que un seam F posterior sobrescriba `INVALIDO` cuando
ya hubo un fallo real anterior.

El cambio corrector es focal: un fichero, siete líneas añadidas y seis
retiradas. No añade alcance ni modifica el adaptador.

## Garantías C4a revisadas

Las revisiones finales aceptaron:

- guardia de carga privada consumible y ejecución directa 64;
- captura canónica exacta de cuatro auxiliares y SHA-256 literales coincidentes;
- política, secuencia, causal, finalizador, oráculo y `RESULTADO` en el runner;
- adaptador limitado a mecanismos;
- padre basal M acreditado y nunca retirado;
- padres T inicialmente ausentes, creados sin `--parents`, identificados y
  retirados como propios;
- wrappers bajo `migraciones/000007_componentes/__h0b`;
- ledger físico de cuatro estados;
- ausencia exacta mediante `! -e && ! -L`;
- N03/N05 con preausencia, copia, identidad y huella anteriores a los seams;
- E03/E05 con retirada exacta de la versión nominal, postausencia y nueva
  identidad antes de los seams;
- N07/E07 sobre rutas distintas con preausencia e identidad;
- F02 con conjunto literal de tres wrappers;
- persistencia del ledger porque el finalizador ejecuta las acciones en el
  shell actual, sin subshell;
- preámbulo sin R0 fuera de la instrumentación M38;
- F esperado no sobrescribe un `INVALIDO` previo.

## Presupuestos y huellas

| Fichero | Líneas | Objetivo | Límite |
| --- | ---: | ---: | ---: |
| Runner | 756 | 760 | 775 |
| Auxiliar H0b | 580 | 580 | 580 |
| Adaptador privado | 482 | 520 | 580 |

Huellas verificadas:

| Artefacto | SHA-256 |
| --- | --- |
| D2c | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| Adaptador | `7cff227b5f97315fd5693a6901ee4bfc55640d93181d4a9f15fd99d2780646e8` |
| D2d | `8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075` |
| Capturador | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |

## Pruebas finales

Sobre el HEAD exacto `bb5e6685ac1843a9654d997a4efc2e89537c11d3`
quedaron verdes:

- `bash -n`, ShellCheck `-x`, `git diff --check` y `git show --check`;
- oráculo puro y prueba focal de preservación F02/`INVALIDO`/N03;
- ejecución directa de cada uno de los cuatro auxiliares: estado 64;
- Gitleaks focal y acumulado: cero fugas;
- H0 nominal mediante `--etapa H0`, imagen PostgreSQL 18.4 fijada por digest:
  estado 0 en 20,73 segundos.

La ejecución independiente observó 42501 exacto sin R0, R0 canónico,
integración nominal C2, error posterior 22012, ambos finalizadores, restauración
exacta de las dos raíces y clasificador contra errores PostgreSQL reales.

Después de la ejecución se acreditaron cero:

- contenedores por nombre propio;
- contenedores por etiqueta propietaria;
- rutas `/tmp/vec-f0-h0-*`;
- procesos del runner o con prefijo propio.

El worktree permaneció limpio antes y después.

## Pendientes expresos

C4a no corrige ni oculta:

- C4b: señales, `monitor`/`SHELLOPTS`, trabajo provisional, PGID, plazos,
  daemon y epílogo exterior idempotente;
- C4c: completar y acreditar dos pasadas, ledger e idempotencia de
  F02/F03/F04, incluidos mutantes de identidad/topología; en particular,
  siguen incompletas las segundas ejecuciones de F03/F04;
- C4d: rechazos 64 endurecidos, matriz de 39 procesos, H0 × 3, A1, C1,
  mutante A1, Gitleaks y doble revisión funcional final.

La ruta de matriz continúa en fallo cerrado y no imprime una acreditación
falsa. Ningún pendiente se computa como cerrado por la H0 nominal.

## Estado

C4a puede integrarse como checkpoint no productivo. Las métricas no aumentan:
F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
`1/14`; producción continúa en `NO-GO`.
