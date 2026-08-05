# Enmienda G2-O/O1b: ledger correctivo hasta 790 líneas

Fecha: 5 de agosto de 2026.

Estado: **propuesta para revisión independiente; NO-GO para reanudar código**.

Base integrada: `fac31c939c30715d30bd5273a9ab578c27fb4e16`.

Estado candidato detenido, sin commit:

- worktree `.worktrees/f0-h0b-c4b2-o1b-20260805`;
- G2 678 líneas, SHA-256
  `d44f0cc3c011d09d95dc1f1b56f69382c61bfac45de12376feb0dee0788568f5`;
- delta G2 `+278` sobre 400;
- runner, G1, capturador, adaptador, D2d, D2c y H0b invariantes;
- runner conserva correctamente las huellas anteriores;
- ningún commit candidato y ninguna integración.

## Motivo único

El contrato O1b autorizó delta máximo +265, total previsto 585..665 y parada
680. El primer candidato material midió producción +136..137 y autopruebas más
enganche aproximadamente +141..142: G2 alcanzó 678 y delta +278. Dirección
detuvo el productor antes de confirmar o tocar el runner.

El árbol detenido demuestra que el lector cabe en G2, pero el ledger previo no
cubre una matriz completa y legible. No se eliminan pruebas ni se minifican
controles para recuperar trece líneas.

Esta enmienda cambia únicamente el presupuesto y enumera la corrección
obligatoria. No modifica firma, estados, resultados, errores, gramática,
propiedad, límites, régimen mono/multitrama, write-set o prohibición de O2.

Cuando obtenga el GO documental, sustituirá exclusivamente las tablas de
delta y totales y los umbrales de parada `+265/680` del apartado «Ledger
físico de O1b» del contrato original. Permanecen vigentes sin modificación la
firma, los estados, resultados, errores, precedencias, matriz funcional,
write-set, invariantes, puertas y prohibiciones de aquel contrato.

## Resultado de las revisiones del árbol detenido

Dos revisiones independientes mantienen el candidato en NO-GO:

| Revisión | P0 | P1 | P2 | Resultado |
| --- | ---: | ---: | ---: | --- |
| Funcional y seguridad | 0 | 3 grupos | 1 grupo | NO-GO |
| Ledger y cobertura | 0 | 4 | 2 | NO-GO |

Bloqueos comunes:

1. delta `+278 > +265`;
2. `prevalidarCodificacionM38` llama indebidamente a la autoprueba O1b;
3. faltan vectores contractuales materiales;
4. las huellas del runner aún no corresponden al candidato.

El punto 4 es una garantía correcta mientras el árbol está detenido. Los dos
literales solo se sustituyen después de estabilizar y revisar G2.

## Corrección funcional obligatoria

La prevalidación del encoder O1a vuelve a terminar en `nil`. O1b se invoca una
sola vez al final de `autoprobarTramasM38`, después de toda la matriz O1a. El
cambio puede ser de cero líneas netas y no convierte una operación de codec en
ejecución de pruebas.

La matriz O1b debe añadir o reforzar, sin pruebas vacuas:

- construcción comprobada; ningún `lector, _ :=` oculta errores;
- muestras en orden determinista y diagnóstico reproducible;
- máximos canónicos reales, incluidos LF: `SOBRE` 2212, `CONTROL` 100,
  `TERMINAL` 179 y `TICKET` 2060;
- L1 más fragmento vacío sin EOF, conservando parcial;
- L2 más fragmento vacío sin EOF, y dato posterior en llamada separada;
- copia defensiva de parcial, monoframa L2 y trama entregada;
- EOF parcial en la misma llamada y después de varios fragmentos;
- fragmento enorme con LF temprano y enorme sin LF, sin reserva proporcional;
- precedencia de byte inválido sobre exceso en la misma frontera;
- buffer y longitud cero después de éxito, EOF y cada familia de fallo;
- tupla completa de retornos cero para todos los errores;
- error terminal pegajoso desde byte, exceso, gramática, EOF parcial,
  monoframa ausente, cola monoframa y uso posterior a EOF;
- cola monoframa con `fin=false` y `fin=true`;
- L3 ya enclavado antes de devolver un `CONTROL` coincidente con EOF;
- dato directo desde L3, sin pasar primero por otra llamada inválida;
- consumo del segundo control después de parcial previa más dos controles
  coalescidos;
- monoframa y CONTROL con sobrante exacto y contador de la llamada actual.

Cada frontera física comprueba la identidad de error: tamaño exacto llega a
O1a y un byte superior es exceso. No basta con observar cualquier error.

## Ledger correctivo

Las dos estimaciones independientes del trabajo restante fueron +35..+55 y
+70..+105 líneas. Se adopta el rango envolvente global, sin tratarlo como
objetivo. Los rangos observados por subparte están correlacionados y no se
suman de forma independiente; el único presupuesto vinculante es el global:

| Magnitud global G2 | Delta desde base | Líneas de G2 |
| --- | ---: | ---: |
| Candidato detenido | +278 | 678 |
| Corrección total restante | +35..+105 | — |
| **Final previsto** | **+313..+383** | **713..783** |
| **Parada de revisión** | **+390** | **790** |

| Unidad | Base | Candidato detenido | Total previsto | Parada O1b |
| --- | ---: | ---: | ---: | ---: |
| Runner | 800 | 800 | 800 | 800 |
| G1 | 686 | 686 | 686 | 686 |
| G2 | 400 | 678 | 713..783 | 790 |
| Capturador | 799 | 799 | 799 | 799 |
| Adaptador M38 | 527 | 527 | 527 | 527 |
| D2d | 145 | 145 | 145 | 145 |
| D2c | 588 | 588 | 588 | 588 |
| H0b | 580 | 580 | 580 | 580 |

La parada 790 queda diez líneas por debajo del tope absoluto DEC-051 de 800.
Si a 790 falta un solo vector, hay que detenerse y revisar la separación; no se
usa el margen de diez líneas sin otra decisión. El objetivo es cerrar cerca del
menor extremo compatible con legibilidad, no consumir la parada.

## Write-set e invariantes

Código permitido tras el GO documental:

1. G2, únicamente para corregir el enganche y completar O1b;
2. runner, al final, únicamente para sustituir el SHA-256 de G2 y el SHA-256
   del binario compuesto, sin cambiar sus 800 líneas.

G1, su literal SHA, capturador, adaptador M38, D2d, D2c y H0b quedan byte a
byte invariantes. El manifiesto conserva seis fuentes. Se prohíben séptimo
fichero, traslado de pruebas a G1, retirada de cobertura y combinación de
controles independientes para caber.

El candidato se vuelve a detener si:

- G2 supera 790 o delta +390;
- cambia una unidad invariante;
- el runner cambia algo distinto de los dos literales autorizados;
- `--supervisar-m38` deja de devolver 64;
- aparece FD, proceso, señal, pidfd operativo, Bash de caso, Docker,
  PostgreSQL, SQL, red, reloj o dependencia nueva;
- una puerta exige relajar el contrato.

## Puertas posteriores

Solo tras corregir G2 se calculan sus huellas y las del binario para el runner.
Después se reproducen:

- `gofmt` y `go vet` sobre G1+G2;
- dos builds privados con raíces, `HOME`, `TMPDIR` y `GOCACHE` disjuntos;
- SHA binaria reproducible e igual al literal;
- fuentes invariantes antes y después del build;
- autoprueba completa, modo operativo/desconocido 64, FD e hijos invariantes;
- runner 800, Bash, ShellCheck, `git diff --check` y Gitleaks;
- puertas globales después de doble GO de código.

Docker, PostgreSQL, red y E2E no corresponden al lector puro. La aceptación de
este ledger no integra el árbol detenido, no autoriza O2 y no cambia F0
`10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14`
ni producción `NO-GO`.
