# Revisión F0-H0b M38: presupuesto real y topología H0

Fecha: 2 de agosto de 2026.

## Resultado

La
[enmienda de presupuesto real y topología](../enmienda_f0_h0b_m38_presupuesto_real_topologia_2026-08-02.md),
confirmada en `aadc6db`, obtuvo tres revisiones documentales independientes
`GO`, todas con `P0=0`, `P1=0` y `P2=0`.

El dictamen autoriza a reanudar las minitareas C4a–C4d bajo sus nuevos límites.
No acredita el código WIP, H0, PostgreSQL, la matriz M38, H0b, C2, F0 ni
producción.

## Alto funcional que motivó la decisión

El WIP no confirmado quedó medido en runner 725, auxiliar H0b 580 y adaptador
361. El adaptador superaba el límite anterior de 350 antes de completar
señales, idempotencia, grupos y mutantes. Las auditorías reprodujeron las
huellas de los tres ficheros y comprobaron que D2c, D2d y el capturador seguían
inmutables.

Las prepruebas y los residuos cero fueron observaciones de sesión del productor,
sin acta persistente, y la enmienda los califica expresamente como diagnóstico.
ShellCheck seguía rechazando SC2016, la huella literal del adaptador estaba
obsoleta y H0 permanecía en `NO-GO`.

## Cadena exacta

| Candidato | Dictamen | Corrección |
| --- | --- | --- |
| `936c417` | `NO-GO`, con `P1=2` y `P2=3` en la revisión más exigente | Distinguir N de E; sustituir expresamente la semántica anterior de grupos; marcar nodos propios; desglosar ahorros; atribuir diagnósticos. |
| `aadc6db` | tres `GO`, todos `P0=P1=P2=0` | Candidato documental final. |

`936c417` fue un objeto intermedio de `commit --amend`: no se considera
aprobado ni se presupone que conserve una referencia Git.

## Presupuesto aprobado

El ledger parte de 361 líneas y añade todos los controles pendientes. Los tres
revisores aceptaron la proyección reproducible 465..564 y el sobre conjunto
450..571.

| Fichero | Objetivo | Límite local | DEC-051 |
| --- | ---: | ---: | ---: |
| Runner | 760 o menos | 775 | 800 |
| Auxiliar H0b | 580 exactas | 580 | 800 |
| Adaptador privado | 520 o menos | 580 | 800 |

El adaptador tiene checkpoint obligatorio en 560. Superar 580 exige una nueva
decisión; no permite minificar, retirar controles o mutantes, mover política ni
crear un quinto auxiliar. La captura privada sigue teniendo cuatro fuentes.

## Topología y estados aprobados

La revisión acepta como basales `/repo_h0b`, sus segmentos hasta
`autorizacion_atestada_v3`, `migraciones` y
`migraciones/000007_componentes`. Nunca se adoptan ni retiran.

Son propios después de preausencia, creación sin `--parents` e identidad:

- M080 y T080;
- `pruebas_sql` y su `000007_componentes`;
- `migraciones/000007_componentes/__h0b`;
- sus hojas `nominal/error` y los tres wrappers.

El cambio de los wrappers al padre basal de migraciones está expresamente
enmendado. Si los dos padres de la rama T pasan a ser basales, la prueba falla
con 65 y exige otra decisión.

Cada miembro de F02/F03/F04 conserva uno de cuatro estados:
`nunca_creado`, `registrado_presente`, `retirado_para_sustitucion` o
`retirado_por_la_accion`. La ausencia solo es válida en el estado que la
autoriza; la desaparición externa de una identidad presente fuerza 65.

N03/N05 parten de ausencia. E03/E05 revalidan y retiran la versión nominal,
acreditan postausencia y registran la nueva versión antes del seam. N07/E07 usan
rutas distintas y parten también de ausencia. F02/F03/F04 validan el conjunto
literal completo antes de retirar, conservan ledger de retirada parcial y son
idempotentes en su segunda ejecución.

## Fronteras y supervisión aprobadas

El runner mantiene parser, selector, secuencia de tramos, causal, finalizador,
oráculo y `RESULTADO`. El adaptador solo expone mecanismos. La supervisión debe
completar estado previo de `monitor`/`SHELLOPTS`, señales diferidas, trabajo
provisional, PID/PGID/PPID/inicio, plazos, extinción, espera directa, CID,
cidfile, imagen, readiness, daemon, temporal y epílogo idempotente.

Las minitareas aprobadas son:

1. C4a: frontera, topología, identidades, ShellCheck, huellas y H0 nominal;
2. C4b: supervisión exterior y mutantes de proceso/Docker;
3. C4c: grupos interiores y mutantes de identidad/topología;
4. C4d: rechazos, matriz 39, H0 × 3, A1, C1, mutante A1, Gitleaks y
   doble revisión.

Ninguna minitarea intermedia aumenta métricas ni autoriza producción.

## Puertas documentales

- objeto exacto: `aadc6dbe473bec082036bc31dda34edd1ec52d93`;
- worktree documental limpio;
- `git show --check` y `git diff --check`: verdes;
- enlaces locales y ancla DEC-051: válidos;
- Gitleaks `ae43932..aadc6db`: un commit, 15,24 kB y cero fugas;
- único cambio: este diseño Markdown de 298 líneas;
- revisiones solo lectura, sin PostgreSQL, Docker ni cambios de código.

## Estado

La implementación puede continuar por C4a, pero H0b, C2 y F0 siguen abiertos.
F0 permanece en `10/23`, O4-05 en `3/5`, Contratación temporal en `24/46`,
Bolsa productiva en `1/14` y producción en `NO-GO`.
