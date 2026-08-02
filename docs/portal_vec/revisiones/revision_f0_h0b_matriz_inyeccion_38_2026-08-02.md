# Revisión F0-H0b: matriz de inyección de 38 casos

Fecha: 2 de agosto de 2026.

## Resultado

La [enmienda de matriz de 38 casos](../enmienda_f0_h0b_matriz_inyeccion_38_2026-08-02.md),
confirmada finalmente en `7a10acc`, obtuvo dos revisiones documentales finales
independientes `GO`, ambas con `P0=P1=P2=0`.

Este dictamen aprueba el contrato documental del checkpoint 4. No acredita su
implementación, una ejecución PostgreSQL, el cierre funcional de H0b, C2 o F0
ni producción.

## Primer candidato y NO-GO del revisor 1

El objeto candidato local `37245eaf` recibió `NO-GO`, con
`P0=0, P1=1, P2=1`:

- `P1`: el oráculo fijaba solo el último centinela. No exigía para cada caso
  `F01` a `F15` la secuencia ordenada completa desde la acción seleccionada
  hasta el centinela terminal, independiente del bucle o array ejecutor;
- `P2`: los selectores vacíos en modo inyección, desconocidos, repetidos o no
  emitidos por el conductor se rechazaban sin fijar estado exacto ni demostrar
  que el rechazo ocurría antes de temporales, contenedores, Docker y `psql`.

El amend produjo el objeto candidato local `168df6ce`. Añadió las quince
secuencias literales completas, los mutantes de centinela y el rechazo exacto
`64` anterior a cualquier recurso. Reservó `79` para las fronteras `A/N/E` y
`65` para los fallos `F`.

`37245eaf` y `168df6ce` son identificadores observados durante la cadena local
de revisión. Los amend posteriores sustituyeron el commit de la rama; esta
evidencia no presupone que esos objetos intermedios conserven una referencia
Git ni depende de su accesibilidad futura.

## Segundo NO-GO

El revisor 2 evaluó `168df6ce` y emitió `NO-GO`, con
`P0=0, P1=1, P2=3`:

- `P1`: el intervalo atribuido al runner no estaba desglosado y no podía
  reproducirse sin suponer ahorros;
- `P2`: el contrato no separaba de forma inequívoca que el auxiliar almacena
  expectativas literales mientras solo el runner calcula y propaga la causal
  real;
- `P2`: el proceso nominal no fijaba conjuntamente *seam* cerrado, estado `0`,
  secuencia completa de quince centinelas más terminal y recuperación exacta;
- `P2`: las autopruebas no mutaban de forma independiente la condición de
  recuperación esperada.

La corrección final `7a10acc` cerró los cuatro hallazgos sin minificar, retirar
controles, mover fronteras ni añadir un cuarto auxiliar.

## Medición reproducible y límites

El presupuesto final del runner no descuenta ahorros:

| Bloque | Mínimo | Máximo conservador |
| --- | ---: | ---: |
| Base del checkpoint 3 | 630 | 630 |
| Selector, *seam* y traza | +8 | +12 |
| Cruce de 23 fronteras | +10 | +14 |
| Finalizador, idempotencia y centinelas | +6 | +10 |
| Driver de 39 procesos y recuperación exacta | +14 | +20 |
| Propagación, nominal y rechazos | +5 | +8 |
| **Delta** | **+43** | **+64** |
| **Total** | **673** | **694** |

Por ello quedan fijados objetivo `690` y límite duro `700`. Las 100 líneas
hasta el límite global de 800 no acreditan ni desbloquean I0. Antes de I0 se
exige replanificación y decisión separadas.

La medición independiente del auxiliar H0b es:

```text
base 460 + catálogo 43 + oráculo completo 43 + estructura/mutantes 10..14
= 556..560 líneas
```

Su objetivo queda en `560` y su límite duro en `580`. El margen solo protege
la legibilidad. El auxiliar conserva catálogo y resultados esperados literales
puros; no recibe procesos, Docker, `psql`, estados, finalización o causal real.
D2c, D2d y el capturador permanecen byte a byte inmutables.

## Dictámenes finales

| Revisión | Dictamen | Recuento | Verificación independiente |
| --- | --- | --- | --- |
| Revisor 1 | `GO` | `P0=P1=P2=0` | Contrastó las 23 fronteras, las 15 acciones, las quince secuencias `F` completas, `TERMINAL`, mutantes de centinela, estados `64/79/65` y rechazo anterior a todo recurso. |
| Revisor 2 | `GO` | `P0=P1=P2=0` | Reprodujo ambas sumas sin ahorros, límites y reserva; comprobó el nominal `0`, la recuperación exacta, la mutación de su condición y la separación entre oráculo literal y causal del runner. |

Ambos revisores comprobaron además que los 38 fallos más el nominal usan
procesos y contenedores nuevos; la trampa exterior de cada hijo retira sus
recursos y el conductor acredita por sus identidades exactas cero residuos
antes del siguiente caso; driver, Docker, `psql`, *seam*, finalizador y estados
permanecen en el runner.

## Puertas documentales

- enlaces Markdown locales de la enmienda y su nota de prevalencia: válidos;
- cardinalidades literales: 23 fronteras, 15 acciones y 15 secuencias del
  finalizador;
- `git diff --check` y `git show --check`: limpios;
- enmienda, documento de límites y esta revisión por debajo de 800 líneas;
- Gitleaks del commit final: cero fugas;
- autor y confirmador: `aavidad <avidad@dipgra.es>`;
- *write-set* documental, sin código, Word de RRHH, tablero ni relevo.

No se ejecutaron Bash, ShellCheck ni PostgreSQL porque el corte es
exclusivamente documental. Esas puertas pertenecen a la implementación del
checkpoint 4 y no se heredan de este doble `GO`.

## Estado

La revisión no cierra H0b, C2 ni F0 y no aumenta métricas. F0 permanece en
`10/23`, O4-05 en `3/5`, Contratación en `24/46`, Bolsa productiva en `1/14`
y producción en `NO-GO`.
