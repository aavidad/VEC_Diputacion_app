# Revisión F0-H0b: límite 775 y descriptores del runner M38

Fecha: 2 de agosto de 2026.

## Resultado

La
[enmienda de límite 775 y ejecución por descriptor](../enmienda_f0_h0b_matriz_limite_runner_775_2026-08-02.md),
confirmada en su candidato final `803533b`, obtuvo dos revisiones documentales
independientes `GO`, ambas con `P0=P1=P2=0`.

Este dictamen aprueba exclusivamente el presupuesto y el protocolo documental
del checkpoint 4. No acredita su implementación, Bash, PostgreSQL, los 39
procesos, H0b, C2, F0 ni producción.

## Cadena exacta de revisión

| Candidato | Revisor 1 | Revisor 2 | Resultado |
| --- | --- | --- | --- |
| `2ceab4c` | `NO-GO`, `P0=0 P1=3 P2=0` | `NO-GO`, `P0=0 P1=4 P2=3` | Rehacer presupuesto y protocolo. |
| `6bf183a` | `NO-GO`, `P0=0 P1=1 P2=0` | `NO-GO`, `P0=0 P1=1 P2=1` | Corregir nombre y rechazos previos a recursos. |
| `803533b` | `GO`, `P0=P1=P2=0` | `GO`, `P0=P1=P2=0` | Candidato documental final. |

`2ceab4c` y `6bf183a` son identificadores observados durante la cadena de
amend. No se presupone que conserven una referencia Git ni que su contenido
rechazado sea válido. `803533b` identifica el árbol exacto que recibió los dos
dictámenes finales.

## Primer candidato rechazado

Los dos dictámenes sobre `2ceab4c` coincidieron en que el presupuesto y el
protocolo no eran implementables con las garantías declaradas. La síntesis
conjunta de sus hallazgos, sin atribuir un hallazgo particular a uno de los
revisores, fue:

- el intervalo `727..739` omitía parte del protocolo seguro y del orden de
  observación del resultado;
- una copia privada conservada por nombre seguía expuesta a sustitución de
  ruta y no justificaba llamarla inmutable frente a procesos del mismo UID;
- la raíz no podía derivarse desde `BASH_SOURCE` de una copia temporal y no
  procedía prometer una SHA-256 de contenido para un directorio vivo;
- el conductor debía conocer la identidad física del temporal antes de lanzar
  al hijo, incluso si este moría antes de informar;
- el contenedor no podía descubrirse o retirarse solo por nombre: eran
  necesarios ID completo, nombre exacto y etiqueta propietaria;
- un proceso intermedio podía romper la ligadura `PPID|caso|identidad`;
- un `RESULTADO` emitido antes de `EXIT` no acreditaba el estado real después
  de la trampa exterior;
- N01/E01 locales no acreditaban un `mkdir` real dentro del contenedor y
  dejaban ambiguos los recursos agrupados de F02/F04.

La corrección `6bf183a` recalculó el presupuesto, mantuvo el conductor dentro
del runner y fijó Linux/`procfs`, una copia abierta y desvinculada, un FD
canónico del runner, otro FD para la raíz, temporales preasignados, contenedores
ligados a tres datos, lanzamiento directo y comparación entre resultado previo
y estado posterior a `EXIT`. También concretó N01/E01 como hojas distintas del
contenedor creadas después de acreditar R0, con wrappers y retirada agrupada.

## Segundo candidato rechazado

Las revisiones de `6bf183a` detectaron todavía dos ambigüedades materiales y
una documental:

- el nombre del documento conservaba el sufijo del límite anterior pese a
  fijar un límite duro de 775;
- el texto no excluía inequívocamente las autopruebas inválidas de la
  preasignación de recursos;
- faltaba acreditar sin consultar Docker que selector vacío, desconocido,
  repetido, sin ticket o discrepante no creaba temporal, contenedor ni ejecutaba
  Docker o `psql` antes de devolver 64.

`803533b` renombró el documento con el límite 775 y retiró la ruta heredada.
Reservó
identidad, temporal, nombre y etiqueta exclusivamente para `NOMINAL` y los 38
casos válidos. Los rechazos se ejecutan con `/usr/bin/bash -x`, un FD de traza
propiedad del bootstrap y un `PATH` no resoluble; su estado exacto 64 y una
traza formada solo por operaciones internas acreditan cero órdenes externas.
Esas autopruebas no consultan Docker y no se cuentan entre los 39 hijos.

## Presupuesto y propiedad aprobados

La suma final no descuenta ahorros:

| Bloque | Mínimo | Máximo conservador |
| --- | ---: | ---: |
| Runner parcial observado | 688 | 688 |
| Núcleo del conductor, rechazos, recursos y bootstrap | +35 | +45 |
| Protocolo por descriptores y orden del resultado | +14 | +24 |
| **Runner completo previsto** | **737** | **757** |

El objetivo local queda en 760 líneas y el límite duro en 775. Permanecen 25
líneas hasta el máximo global 800 de DEC-051; no constituyen reserva y no
desbloquean I0. El auxiliar H0b conserva objetivo 560, límite duro 580 y queda
congelado en 580. Su SHA-256 literal se fija solo tras cerrar el código.

El write-set de código sigue limitado al runner y al auxiliar H0b. D2c, D2d y
el capturador permanecen byte a byte inmutables. Se rechaza un cuarto artefacto
porque dividiría propiedad, carga privada, acreditación y trampa exterior y
abriría otra ventana TOCTOU. Superar 775 es un hard-stop; no se autoriza
minificar, retirar controles o trasladar el conductor al auxiliar.

## Garantías finales contrastadas

Ambos revisores finales aceptaron conjuntamente:

- exactamente 38 fallos más `NOMINAL`, sin recursión;
- inodo canónico abierto, desvinculado y revalidado antes y después de cada
  hijo mediante `/proc/self/fd`;
- pruebas adversas de sustitución, eliminación y restauración del antiguo
  nombre sin confundir nombre con inodo;
- FD de raíz con metadatos físicos, sin SHA-256 del directorio;
- temporal 0700 precreado y adoptado solo por dispositivo e inodo exactos;
- contenedor retirado únicamente tras validar ID, nombre y etiqueta;
- ticket por FD ligado al PPID directo, sin presentarlo como frontera
  criptográfica entre procesos del mismo UID;
- causal declarada contrastada con el estado posterior a `EXIT`, donde un
  fallo exterior sustituye 79 por 65;
- F01..F15 armadas solo después de acreditar R0;
- N01/E01 como `mkdir` reales sobre hojas distintas del contenedor, sin añadir
  una frontera a las 23 existentes;
- F02 y F04 agrupadas, exactas, idempotentes ante ausencia segura y sin glob o
  prefijo;
- rechazos 64 sin preasignación ni llamada externa;
- catálogo literal 38, oráculo literal 39 y auxiliar congelado.

## Puertas documentales

- enlaces Markdown locales y ancla DEC-051: válidos;
- documento de decisión y enmienda M38 por debajo de 800 líneas;
- ausencia de la ruta heredada en el árbol candidato;
- `git diff --check` y `git show --check`: limpios;
- Gitleaks del candidato final: cero fugas;
- autor y confirmador: `aavidad <avidad@dipgra.es>`;
- solo dos documentos, sin código, Word de RRHH, tablero ni relevo.

No se ejecutaron Bash, ShellCheck ni PostgreSQL porque la cadena revisada es
exclusivamente documental. Esas puertas pertenecen al checkpoint 4 y no se
heredan de estos `GO`.

## Estado

La revisión no cierra H0b, C2 ni F0 y no aumenta métricas. F0 permanece en
`10/23`, O4-05 en `3/5`, Contratación temporal en `24/46`, Bolsa productiva en
`1/14` y producción en `NO-GO`.
