# Enmienda G2-O/O1a: ledger correctivo G2 hasta 420 líneas

Fecha: 5 de agosto de 2026.

Estado: **consumida y cerrada por O1a integrada hasta `52c8852`**. La parada
400 fue insuficiente; la evidencia real midió G2=391. La corrección `6e0b985`
obtuvo tres GO independientes y el resultado final quedó en G2=400, dentro de
la parada 420, también con triple GO. Nunca autoriza O1b.

## Motivo único

El candidato O1a `b250d38` quedó en NO-GO por dos defectos funcionales, una
matriz de mutantes insuficiente y una reserva previa a validación. G2 ocupa 299
líneas y su parada aprobada era 320. Corregir con legibilidad y pruebas reales
requiere 90..101 líneas adicionales; la medición detenida fue +92, G2=391.

Esta enmienda cambia solo el presupuesto de G2 para corregir O1a. No modifica
gramática, topología, autoridad, framing, O1b ni operación real.

## Alternativas

Se rechaza añadir una séptima fuente de autoprueba: obligaría a ampliar
manifiesto, captura y hashes en un runner que ya ocupa 800 líneas, aumentando
la superficie crítica solo para repartir tests.

Se adopta conservar implementación y autoprueba cohesionadas en G2 y elevar su
parada a 420. Sigue muy por debajo del límite DEC-051 de 800 y conserva una
reserva positiva sin minificación.

## Base y ledger

El candidato correctivo será hijo directo de la base candidata no integrable
`b250d38aadd165aa80471dbbf1200d56bd4165bb`. La rama integradora permanece en
su línea documental; `4e3bc6d` y `b250d38` son ramas hermanas nacidas de
`ef565cd`, no una secuencia de código.

| Unidad | Base candidata | Corrección prevista | Total previsto | Parada |
| --- | ---: | ---: | ---: | ---: |
| Runner R | 800 | 0 líneas; SHA de G2 y binario sustituidos | 800 | 800 |
| G1 | 686 | 0 | 686 | 690 |
| G2 | 299 | +90..+101 | 389..400 | 420 |
| Capturador | 799 | 0 | 799 | 799 |
| Adaptador M38 | 527 | 0 | 527 | 527 |
| D2d | 145 | 0 | 145 | 145 |
| D2c | 588 | 0 | 588 | 588 |
| H0b | 580 | 0 | 580 | 580 |

Desglose G2:

| Corrección | Delta conservador |
| --- | ---: |
| Prevalidar encoder antes de concatenar, incluido control vacío | +20..+28 |
| Rechazar sin Bash/S4 y cruces asociados | +1..+4 |
| Sustituir matriz insuficiente por cobertura contractual | +69..+69 |
| **Total** | **+90..+101** |

La medición real conservada sin commit es G2=391, dentro del rango. La reserva
final mínima es 20 líneas. No se transfiere a otra fase.

## Write-set y parada

La corrección puede tocar únicamente:

1. G2 para los tres hallazgos y sus mutantes;
2. runner para sustituir SHA de G2 y del binario; el SHA G1 solo cambia si G1
   cambia, lo cual esta enmienda no autoriza. Por tanto, fuente G1 y su literal
   SHA quedan byte a byte invariantes respecto de `b250d38`.

G1 y todo componente restante deben quedar byte a byte invariantes respecto de
`b250d38`. Se detiene si:

- G2 supera 420 o la corrección supera +101;
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
