# Enmienda F0-H0b/C4b-2 G2-S: separación capturada

Fecha: 5 de agosto de 2026.

Estado: **aceptada** tras doble revisión independiente con `P0=P1=P2=0`.
Autoriza únicamente G2-S; no cierra C4b-2 ni modifica métricas.

## Motivo y alcance único

C4b-2/G1 quedó integrado en `f3d928d`–`d28d37d` con doble `GO`,
`P0=P1=P2=0`, 100/100 autopruebas de dirección y puerta completa verde. Su
fuente Go ocupa 754 líneas y no admite G2 bajo el tope 800 sin una separación.

Esta enmienda autoriza únicamente preparar esa separación. Sustituye la
prohibición del sexto fichero y la cardinalidad cinco de la decisión C4b-2.
No fija ni implementa canales, FD operativos, Bash de caso, plazos o recibos;
eso pertenece a G2-O y a su especificación independiente.

El capturador actual ocupa 799 líneas y ya admite 1..512 rutas explícitas,
elimina duplicados, ordena canónicamente, copia por descriptor y publica un
manifiesto privado. Permanece byte a byte invariante.

## Composición cerrada de seis

El manifiesto contiene exactamente, en orden lexicográfico:

1. `arnes_fuente_corporativa_contexto_actor_v1.sh`;
2. `arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh`;
3. `ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh`;
4. `operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`;
5. `supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go`;
6. `supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go`.

Las cuatro cargas Shell siguen en orden funcional D2c, D2d, H0b y adaptador.
Ambos Go usan `//go:build ignore && linux && amd64`; solo el primero contiene
`main`. Las compilaciones ordinarias conservan el capturador en `GoFiles` y
los dos supervisores en `IgnoredGoFiles`. Se prohíbe `-tags=ignore`.

El runner acredita ambos Go antes y después del build: ruta y SHA-256
literales, `0600`, EUID, fichero regular, no simbólico y un enlace. Ejecuta
`go vet fuente_g1 fuente_g2` y `go build -trimpath fuente_g1 fuente_g2` sobre
las copias privadas con el entorno Go cerrado vigente. El binario compuesto
queda `0700`, regular, con un enlace y SHA-256 reproducible literal. No reabre
una ruta viva.

## Separación funcional mínima

G1 conserva `main`, la autoprueba aceptada y su comportamiento. Reconoce
además el modo literal `--supervisar-m38`, sin argumentos adicionales, y lo
delega a una única función privada definida por el segundo fichero.

Durante G2-S esa función es un cierre seguro: devuelve uso 64 sin abrir,
duplicar o cerrar FD, crear procesos, instalar señales ni cambiar el número de
hijos. G2-O sustituirá solo su cuerpo después de revisión propia; no volverá a
modificar el dispatcher G1.

Para que el segundo fichero sea una separación real y no un cascarón, G2-S
puede trasladarle mecánicamente las primitivas Linux compartidas que utilizarán
G1 y G2 —duplicación de pidfd, `poll`/terminalidad, `prctl`, conteo de FD,
`ESRCH` y envío pidfd—. No cambia firmas, conducta ni pruebas G1.

## Ledger físico previo

Estado de partida: runner 789, Go G1 754, capturador 799 y adaptador 527.

El runner no añade controles paralelos. Sustituye sus bloques escalares:

| Cambio | Variación máxima de líneas |
| --- | ---: |
| ruta y SHA del sexto fichero | +2 |
| sexta entrada en captura y manifiesto | +1 |
| segunda ruta Go privada | +1 |
| validación común de forma y dos hashes | +2 netas, reemplazando el bloque de un fichero |
| `vet`/build y reacreditación de dos rutas | +2 netas, reemplazando las llamadas escalares |
| **Runner máximo** | **797** |

El commit debe adjuntar `git diff --numstat`, rangos sustituidos y `wc -l`
reales. Si el runner supera 800 o necesita comprimir controles independientes,
G2-S se detiene y requiere otra separación.

Topes del corte: runner 800, G1 790, G2 160, capturador 799 exacto y adaptador
527 exactas. Son máximos, no objetivos ni reserva automática.

## Write-set y cierre

Código permitido: runner, fuente G1 y nueva fuente G2. Documentación y pruebas
propias acompañan el corte. Capturador, adaptador, D2c, D2d y H0b son
invariantes.

G2-S exige:

- manifiesto exacto seis; 5/7, ausencia, duplicado, ruta y SHA mutados fallan;
- sustitución de cada copia privada y del binario detectada;
- `vet` y dos builds aislados de ambos Go con SHA binaria idéntica;
- paquete ordinario con un `GoFiles` y dos `IgnoredGoFiles`;
- `--supervisar-m38` devuelve 64 sin cambios de FD, procesos o hijos;
- G1 100/100, desconocido/ayudante inválido 64 y residuos cero;
- `gofmt`, Bash, ShellCheck, focales y puertas globales verdes;
- huellas invariantes del capturador, adaptador, D2c, D2d y H0b;
- doble revisión independiente `P0=P1=P2=0`.

Q5a y G1 conservan su alcance histórico de cinco fuentes y el binario anterior.
G2-S no los reescribe ni habilita Docker, PostgreSQL, red, SQL, datos reales o
producción.
