# Decisión F0-H0b C4b-2: quinta captura para la supervisión exterior

Fecha: 4 de agosto de 2026.

Estado: propuesta de dirección pendiente de dos revisiones independientes con
`P0=0`, `P1=0` y `P2=0`. Mientras no obtenga ambos `GO`, no autoriza cambios de
código ni modifica el estado de C4b-2, H0b, C2, F0 o producción.

Prevalencia propuesta: desarrolla exclusivamente la cláusula de parada y
quinta captura de la
[enmienda de presupuesto y topología](enmienda_f0_h0b_m38_presupuesto_real_topologia_2026-08-02.md).
Conserva la semántica de señales de C4b-1, la topología H0b, los oráculos, las
causales y todas las puertas de las enmiendas anteriores.

## Motivo del alto

El corte estable `91cb804` conserva esta medida reproducible:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 769 | se acreditará de nuevo sobre el candidato |
| Adaptador privado de ciclo | 527 | `d9b61a183e5a32c321a3eeb48483ce40c83551bc7a700354ccc88e8206d9ee1f` |

El checkpoint fijado para el adaptador al cerrar C4b es 540 líneas. Quedan
trece, pero C4b-2 todavía necesita, como mínimo legible:

- máquina `ninguno → provisional → capturado → acreditado → terminal`;
- lectura única `PID|estado|PPID|PGID|inicio` y comparación contra reciclado;
- recuperación exacta de cero o un trabajo en la ventana `fork → $!`, con
  incidente ante cualquier otra cardinalidad;
- acreditación y revalidación antes de `CONT`, señal o espera;
- plazo absoluto de 180 segundos, `TERM`, gracia de dos segundos, `KILL`,
  `wait -f` del Bash directo y prueba de extinción del grupo;
- mutantes deterministas sin Docker para todas esas fronteras.

El análisis productor estima al menos 48 líneas netas después de sustituir la
lógica incompleta existente: el adaptador alcanzaría 575 como mínimo y dejaría
cero reserva real para C4b-3 y C4c. Comprimir sentencias, retirar mutantes o
elevar silenciosamente el checkpoint contradiría la enmienda.

También falta una autoridad única de limpieza exterior. El runner posee la
trampa y el adaptador actual mezcla supervisión de procesos, Docker, temporal
y recursos interiores. Ampliar esa mezcla haría más difícil demostrar que
todos los fallos convergen en un solo epílogo.

No se ha escrito código en el intento detenido. `bash -n`, ShellCheck,
`git diff --check` y las huellas de H0b, D2c, D2d y el capturador permanecen
verdes.

## Alternativas

### A. Encajar C4b-2 en trece líneas

Rechazada. Solo sería posible ocultando estados, agrupando controles o
trasladando pruebas fuera del contrato reproducible.

### B. Consumir las cuarenta líneas reservadas a C4c

Rechazada. Convertiría el checkpoint 540 en una cifra decorativa y obligaría a
otro rediseño inmediato.

### C. Trasladar la supervisión al runner

Rechazada. El runner está en 769 líneas, conserva política, oráculo, causal y
resultado, y tiene un máximo local de 775. Mezclarle mecanismo de procesos o
Docker quebraría además el reparto ya revisado.

### D. Quinta fuente privada de supervisión exterior

Adoptada de forma condicionada. Se crea un único fuente Shell:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  supervision_exterior_m38_h0b_fuente_corporativa_contexto_actor_v1.sh
```

No es un proceso, conductor, supervisor ejecutable ni nueva autoridad de
negocio. Se copia dentro del mismo snapshot coherente, se valida por ruta y
SHA-256 literal y se carga con `source` en el proceso del runner. Su ejecución
directa devuelve 64 antes de cualquier efecto.

## Fronteras de propiedad

### Runner

Conserva exclusivamente:

- parser, selector, catálogo, oráculo y causal `0/64/65/79`;
- ticket, conductor de casos, `RESULTADO` y validación final;
- trampa `EXIT/INT/TERM`, que invoca una sola primitiva pública del supervisor;
- orden del finalizador interior F01..F15;
- captura, validación y carga exactas de las cinco fuentes privadas.

La trampa no implementa limpieza: solo conserva el estado de salida, solicita
la finalización interior cuando procede e invoca el epílogo exterior único.

### Adaptador privado de ciclo

`ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh` queda
restringido al ciclo interior del contenedor:

- topología H0b y formas físicas interiores;
- materialización y cotejo de M080, T080 y wrappers;
- acciones F02, F03 y F04 con ledger e idempotencia;
- mutantes interiores de C4c.

No conserva régimen shell, señales, trabajos, PID/PGID, temporales exteriores,
creación o retirada Docker ni epílogo exterior.

### Quinta fuente de supervisión exterior

Es la única propietaria mecánica de:

- régimen `monitor`, exportación de `SHELLOPTS` y espera terminal de clientes;
- identidad activa, temporal exterior y contenedor exacto;
- máquina de estados del trabajo y tupla física del hijo;
- lanzamiento directo, barrera `STOP/CONT`, plazo y extinción del PGID;
- creación, readiness, reconciliación y retirada Docker de C4b-3;
- una sola primitiva idempotente de epílogo exterior.

El epílogo puede llamar primitivas interiores del adaptador, pero ninguna otra
función puede iniciar una segunda limpieza, desarmar la identidad o lanzar el
siguiente caso. `finalizar_h0b_f0` sigue siendo el finalizador funcional
interior y no compite por procesos, Docker o temporales.

H0b, D2c, D2d y el capturador Go permanecen byte a byte inmóviles.

## Captura privada exacta de cinco

El capturador continúa ejecutándose una sola vez sobre el árbol acreditado. El
runner debe:

1. declarar las cinco rutas y cinco huellas literales;
2. capturarlas en una sola operación;
3. exigir cinco entradas exactas, sin ausencia, duplicado o extra;
4. cotejar cada par ruta/huella antes de cargar cualquiera;
5. comprobar que las cinco ejecuciones directas devuelven 64;
6. ejecutar Bash y ShellCheck sobre runner y cinco copias;
7. cargar cada fuente desde el snapshot con una marca privada efímera que se
   consume al entrar.

La validación puede refactorizarse mediante arrays y bucles literales para no
duplicar código, pero no puede aceptar rutas configurables, descubrir fuentes
por glob ni rebajar la cardinalidad exacta. Un mutante sustituye la quinta ruta
o su huella y debe fallar antes de temporal, Docker, proceso o SQL.

## Contrato C4b-2 conservado

La separación no cambia el resultado exigido:

1. el estado se arma como `provisional` antes del fork;
2. la recuperación admite exactamente cero o un trabajo y trata más de uno
   como incidente 65 sin escoger por conveniencia;
3. la identidad acreditada es
   `PID|estado|PPID|PGID|inicio`, con `estado=T`, `PID=PGID`, padre directo y
   tiempo de inicio estable antes de `CONT`;
4. PID, PPID, PGID e inicio se revalidan antes de cada señal y espera; una
   discrepancia nunca autoriza a señalizar un número reciclado;
5. un único plazo absoluto de 180 segundos se arma antes de `CONT` y nunca se
   reinicia tras una señal o espera interrumpida;
6. la retirada hace `CONT` si procede, `TERM`, gracia máxima de dos segundos,
   `KILL` solo si persiste, `wait -f` únicamente al Bash directo y acredita
   cero miembros del grupo y cero trabajos;
7. no comienza otro caso hasta alcanzar `terminal` y completar el epílogo;
8. C4b-1 conserva exactamente su semántica: señal única 130/143, primera señal
   observada enclavada y ráfaga previa al marco en `{130,143}`.

## Presupuestos propuestos

La quinta captura no convierte el máximo DEC-051 de 800 en reserva:

| Fichero al cerrar C4b | Objetivo | Máximo local |
| --- | ---: | ---: |
| Runner | 770 | 775 |
| Auxiliar H0b | 580 exactas | 580 |
| Adaptador interior de ciclo | 400 | 440 |
| Supervisor exterior | 260 | 340 |

El traslado debe ser mecánico y revisable. No se permite duplicar funciones
entre los dos auxiliares ni mantener fachadas muertas «por compatibilidad». Si
el supervisor proyecta más de 260 al cerrar C4b se realiza un nuevo checkpoint
de capacidad; superar 340 exige otra decisión. C4c no modifica el supervisor.

## Write-set y secuencia

Tras dos `GO` documentales, el trabajo se divide sin commits rotos:

1. **Q5a — captura y frontera:** nuevo auxiliar con guardia privada; traslado
   mecánico de la supervisión ya existente; manifiesto exacto de cinco;
   comportamiento H0 nominal sin cambios.
2. **C4b-2 — hijo y grupo:** máquina de estados, identidad física, recuperación,
   plazo, extinción y mutantes sin Docker.
3. **C4b-3 — Docker y epílogo:** completa la autoridad única de contenedor,
   temporal y daemon.

Write-set material permitido para Q5a/C4b-2:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
  pruebas_sql/ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh
  pruebas_sql/supervision_exterior_m38_h0b_fuente_corporativa_contexto_actor_v1.sh
```

La documentación y las actas se integran en commits separados por dirección.
El productor no aprueba ni integra su propio candidato.

## Puertas de aceptación

Además de las puertas previas, Q5a y C4b-2 deben demostrar:

- diff de traslado sin pérdida ni duplicación de autoridad;
- manifiesto y snapshot de cinco exactos;
- rechazo 64 de ejecución directa y de ruta/huella quinta mutada;
- invariancia byte a byte de H0b, D2c, D2d y capturador;
- `bash -n`, ShellCheck, `git diff --check`, Gitleaks y presupuestos;
- mutantes `antes del fork`, `fork→$!`, `después de $!`, cardinalidad 0/1/>1,
  PID/inicio reciclado, PPID/PGID/estado adversos, espera interrumpida, plazo,
  hijo resistente a TERM y grupo ya vacío;
- hijo directo recolectado, PGID extinguido y tabla de trabajos vacía después
  de cada mutante;
- H0 PostgreSQL 18.4 real, C4b-1 no regresado y residuos cero;
- dos revisiones independientes del commit funcional con
  `P0=0`, `P1=0`, `P2=0`.

Q5a no cierra C4b-2. C4b-2 no cierra C4b ni H0b. Ningún checkpoint modifica
F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
`1/14` ni el `NO-GO` de producción.

## Criterio para aprobar esta decisión

Dos revisores distintos deben confirmar, antes de programar:

1. que la quinta fuente no abre una ruta viva después de la captura;
2. que no crea un proceso intermedio ni una segunda autoridad;
3. que el reparto deja una sola primitiva de epílogo exterior;
4. que los presupuestos dejan reserva real para C4b-3 y C4c;
5. que ninguna garantía de C4b-1, C4b-2, H0b o las fuentes inmóviles se rebaja.

Un solo `NO-GO` mantiene detenido el código y obliga a corregir esta decisión.
