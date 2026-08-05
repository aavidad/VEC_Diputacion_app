# Enmienda G2-O/O1a: ledger correctivo G2 hasta 400 líneas

Fecha: 5 de agosto de 2026.

Estado: **propuesta pendiente de doble revisión independiente**. No autoriza
código hasta obtener `P0=P1=P2=0` de dos revisores.

## Motivo único

El candidato O1a `b250d38` quedó en NO-GO por dos defectos funcionales, una
matriz de mutantes insuficiente y una reserva previa a validación. G2 ocupa 299
líneas y su parada aprobada era 320. Corregir con legibilidad y pruebas reales
requiere 60..80 líneas adicionales.

Esta enmienda cambia solo el presupuesto de G2 para corregir O1a. No modifica
gramática, topología, autoridad, framing, O1b ni operación real.

## Alternativas

Se rechaza añadir una séptima fuente de autoprueba: obligaría a ampliar
manifiesto, captura y hashes en un runner que ya ocupa 800 líneas, aumentando
la superficie crítica solo para repartir tests.

Se adopta conservar implementación y autoprueba cohesionadas en G2 y elevar su
parada a 400. Sigue muy por debajo del límite DEC-051 de 800 y conserva una
reserva positiva sin minificación.

## Base y ledger

Base candidata no integrable: `b250d38aadd165aa80471dbbf1200d56bd4165bb`.
Base integradora sin código O1a: `ef565cd`.

| Unidad | Base candidata | Corrección prevista | Total previsto | Parada |
| --- | ---: | ---: | ---: | ---: |
| Runner R | 800 | 0 líneas; tres SHA sustituidos | 800 | 800 |
| G1 | 686 | 0 | 686 | 690 |
| G2 | 299 | +60..+80 | 359..379 | 400 |
| Capturador | 799 | 0 | 799 | 799 |
| Adaptador M38 | 527 | 0 | 527 | 527 |
| D2d | 145 | 0 | 145 | 145 |
| D2c | 588 | 0 | 588 | 588 |
| H0b | 580 | 0 | 580 | 580 |

Desglose G2:

| Corrección | Delta conservador |
| --- | ---: |
| Prevalidar encoder antes de concatenar | +10..+15 |
| Rechazar sin Bash/S4 y cruces asociados | +5..+8 |
| Sustituir matriz insuficiente por cobertura contractual | +45..+57 |
| **Total** | **+60..+80** |

La reserva final mínima es 21 líneas. No se transfiere a otra fase.

## Write-set y parada

La corrección puede tocar únicamente:

1. G2 para los tres hallazgos y sus mutantes;
2. runner para sustituir SHA de G2 y del binario; el SHA G1 solo cambia si G1
   cambia, lo cual esta enmienda no autoriza.

G1 y todo componente restante deben quedar byte a byte invariantes respecto de
`b250d38`. Se detiene si:

- G2 supera 400 o la corrección supera +80;
- cambia G1 o una ruta no autorizada;
- el runner deja de tener 800 líneas o cambia algo distinto de SHA literales;
- se minifican sentencias, fusionan controles o eliminan pruebas;
- `--supervisar-m38` deja de devolver 64;
- aparece O1b, FD, proceso, señal, pidfd, Bash operativo, Docker, PostgreSQL,
  SQL, red o dependencia nueva.

## Criterio de corrección

El nuevo candidato debe:

- rechazar sin Bash/S4;
- validar cardinalidad y tamaños antes de construir cualquier salida;
- probar cada causa/estado, S1..S4, bloque con/sin Bash, cardinalidad,
  versión/tipo, selector, máximos y overflow decimal, ticket 2048/2049,
  vacío, TAB, NUL, no ASCII y fronteras de clase no vacuas;
- conservar dos builds privados reproducibles, autoprueba verde, hashes
  invariantes y modo 64 cerrado;
- recibir al menos dos revisiones independientes posteriores al productor.

La aceptación de este ledger no integra `b250d38`, no cierra O1a y no modifica
las métricas F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` ni el NO-GO de producción.
