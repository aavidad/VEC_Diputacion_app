# Decisión F0-H0b/C4b-2/G2-O/O2a-P0: preparación para G3

Fecha: 5 de agosto de 2026.

Estado: **integrada, publicada y con CI verde**. La autorización original y la
enmienda de ShellCheck obtuvieron, cada una, dos revisiones documentales
independientes con `P0=P1=P2=0`; el código posterior obtuvo otros dos GO
independientes y la ejecución `31007941124` superó cinco de cinco puertas.

Acta:
[revisión O2a-P0](revisiones/revision_f0_h0b_c4b2_g2o_o2a_p0_preparacion_g3_2026-08-05.md).

Resultado de código:
[revisión final O2a-P0](revisiones/revision_f0_h0b_c4b2_g2o_o2a_p0_codigo_final_2026-08-05.md).

Base exacta de implementación:
`ec530091e6f157baa54ff50e9c70f21c7a014e94`. No se crea el candidato desde
otro corte aunque su código material coincida con `5581821`.

## Motivo

O1b está publicada en `5581821` y acreditada por la ejecución GitHub
`31002229666`, con cinco de cinco puertas verdes. El árbol de código vigente
ocupa:

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 800 | `e37d8a51a5b961ef9175833ba78c47aab4e8e29180db6ff9d771498fa3d16d87` |
| G1 | 686 | `9fab2cae4edd0b5cf8cd5d67fd7a1f9643b81085c815b0c10cb477f67a7e1afe` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| D2d | 145 | `9b137f1302c5672e9fd5c0c8df169810cbc7e57a11fa2129bf79a777e92c5e81` |
| Capturador | 799 | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |
| Adaptador M38 | 527 | `98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7` |
| D2c | 588 | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | 580 | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |

O2a necesita una máquina S0 pura, retención defensiva y su matriz de
autoprueba. Dos estimaciones independientes la sitúan entre 193 y 340 líneas.
No cabe en G2 ni en el espacio agregado de G1 y G2 sin infringir DEC-051,
mezclar responsabilidades o retirar evidencia. El manifiesto privado contiene
exactamente seis fuentes y el runner no dispone de líneas para capturar una
séptima sin preparar antes espacio legible.

Esta decisión crea una minitarea estructural real, O2a-P0, previa al contrato
funcional y a la fuente G3. No crea un fichero vacío ni implementa O2a.

## Hallazgo del primer intento material

El primer intento partió de la base exacta fijada y se detuvo sin commit tras
realizar el traslado literal. El runner quedó en 783 líneas, D2d en 163 y el
bloque conservó su huella, pero ShellCheck rechazó D2d de forma aislada con
`SC2154`: la función trasladada usa `contenedor`, que el runner inicializa
antes de capturar y cargar D2d, aunque el análisis estático del auxiliar no
puede observar ese contrato externo.

No se acepta declarar o inicializar `contenedor` en D2d porque modificaría el
entorno de ejecución para satisfacer una herramienta. Tampoco se acepta una
exclusión global que pudiera ocultar futuros usos sin contrato.

La corrección propuesta añade inmediatamente antes de la definición trasladada
una única directiva local, documental y sin efecto en ejecución:

```bash
# shellcheck disable=SC2154  # Aportado por el runner acreditado.
derivar_repo_base_h0_f0() {
```

La directiva no forma parte del cuerpo de 17 líneas y no cambia su huella. El
runner ya asigna `contenedor` antes de `capturar_auxiliares_privados_f0`; D2d
se carga dentro de esa operación y su ejecución directa sale en 64 antes de
alcanzar la directiva. Con la corrección, D2d debe ocupar 164 líneas. Sus dos
dictámenes documentales independientes dieron `GO`, `P0=P1=P2=0`; el candidato
puede reanudarse dentro del mismo write-set.

La materialización exacta de la directiva sobre el árbol detenido debe producir
estas huellas antes de cualquier otra prueba:

- D2d: `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e`;
- runner, tras acreditar esa huella:
  `da6871ca174890c85eb93ee4cfac15f32ecd1ac046d84d24fa68170ac34c52e9`.

## Prevalencia y corrección de trazabilidad

La
[corrección O1a vigente](enmienda_f0_h0b_c4b2_g2o0_correccion_o1a_2026-08-05.md)
prevalece sobre la especificación G2-O y el primer O0, ambos rechazados:

1. Go consume y valida el `SOBRE` antes de aceptar `ARMAR`;
2. `ACK_LISTO` y `ACK_CASO` desaparecieron del protocolo;
3. O2a será únicamente `S0 ESPERAR_SOBRE` hasta `S1 ESPERAR_ARMAR`;
4. O2b conserva «ARMAR y cancelación sin Bash» como roadmap no autorizado;
5. L0–L4 son estados privados del lector O1b y no equivalen a S0–S6.

Las referencias recientes a `ACK_LISTO` eran obsoletas y se corrigen junto a
esta decisión. Ni esta corrección documental ni O2a-P0 autorizan O2a u O2b.

## Responsabilidad única de O2a-P0

Mover, sin alterar su cuerpo ni su punto de llamada, la función Shell
`derivar_repo_base_h0_f0` desde:

`deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`

hacia el auxiliar privado ya capturado:

`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`.

La función es una operación Docker del runner: valida y retira siete
componentes de la raíz derivada H0. D2d ya posee operaciones Docker del runner,
se carga desde la copia privada antes de la única llamada a esa función y
dispone de la autoridad mínima necesaria. No se introduce una dependencia
circular ni se carga una ruta viva.

Se conservan literalmente:

- nombre y firma;
- cuerpo `docker exec ... bash -ceu`;
- lista y orden de los siete componentes;
- modos, enlaces, vaciado y `rmdir`;
- punto y orden de la llamada;
- mensajes, estados de salida y fallo cerrado.

## Write-set y ledger

Únicamente pueden cambiar:

1. runner, retirando las 17 líneas de la función y actualizando el literal SHA
   de D2d;
2. D2d, incorporando esas 17 líneas, una separación visual legible y la única
   directiva local de ShellCheck fijada por la enmienda;
3. documentación y evidencia de la propia minitarea.

| Unidad | Base | Previsión | Parada |
| --- | ---: | ---: | ---: |
| Runner | 800 | 783 | 783 |
| D2d | 145 | 164 | 164 |
| G1 | 686 | 686 | 686 |
| G2 | 798 | 798 | 798 |
| Manifiesto | 6 fuentes | 6 | 6 |

Capturador, G1, G2, adaptador M38, D2c y H0b son invariantes byte a byte. El
SHA del binario Go compuesto no cambia. No se añade G3, fuente, ruta, proceso,
FD, modo, dependencia o entrada de manifiesto.

Se detiene sin commit si:

- el runner no queda exactamente en 783 líneas o D2d en 164;
- se declara o inicializa `contenedor` en D2d, o se desactiva `SC2154` fuera de
  la única definición trasladada;
- la función cambia semántica o posición relativa de ejecución;
- cambia cualquier unidad invariante o el binario Go;
- el manifiesto deja de contener las mismas seis fuentes;
- se comprime, fusiona o retira otro control para obtener espacio.

## Verificación de O2a-P0

El candidato deberá acreditar:

- `git diff --numstat` y rangos movidos;
- cuerpo completo de 17 líneas trasladado byte a byte, incluida su nueva línea
  final, con SHA-256
  `aae98945ae26e7b4f2637e662157bdaf26a414d3100b046d2c91c4cf1fa59d74`;
- cero definiciones de `derivar_repo_base_h0_f0` en el runner, exactamente una
  en la copia privada capturada de D2d y exactamente una llamada en el runner;
- carga completa de D2d anterior a esa llamada;
- `bash -n` y ShellCheck del runner y D2d;
- ejecución directa de D2d cerrada en 64;
- captura privada exacta y SHA nuevo de D2d;
- ausencia, duplicado, ruta o huella adversos rechazados;
- H0 nominal completo en PostgreSQL 18.4, incluida la derivación de raíz;
- autoprueba G1/O1a/O1b y modos `--supervisar-m38`/desconocido en 64;
- fuentes Go y binario compuesto invariantes;
- residuos cero, `git diff --check`, Gitleaks y puertas globales;
- doble revisión independiente de código, `P0=P1=P2=0`.

O2a-P0 no autoriza código hasta obtener dos revisiones documentales
independientes con `P0=P1=P2=0`. El candidato posterior requerirá otras dos
revisiones independientes de implementación.

## Trabajo posterior todavía cerrado

Solo después de integrar y medir O2a-P0 se redactará el contrato funcional y
ledger definitivo de O2a. Ese documento deberá decidir una API alimentada por
fragmentos, sin propiedad real de FD 9, y una G3 capturada con implementación
real; no se admitirá un cascarón.

O2a retendrá una única carga opaca para uso futuro, sin interpretar, exponer,
registrar ni entregar el ticket, y no abrirá FD, Bash, procesos, goroutines,
señales, reloj o red. Estas son restricciones para el siguiente contrato, no
autorización de código en este corte.

O2a-P0 no cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`,
Bolsa productiva `1/14` ni el `NO-GO` de producción.

## Resultado material

El candidato `df9a422` obtuvo doble GO independiente de código,
`P0=P1=P2=0`, y quedó integrado como `48a46a3`. Dirección reprodujo H0
PostgreSQL 18.4, residuos cero, huellas, Gitleaks y la puerta global. El cierre
se publicó en `ef027cf` y la CI `31007941124` terminó completamente verde.
