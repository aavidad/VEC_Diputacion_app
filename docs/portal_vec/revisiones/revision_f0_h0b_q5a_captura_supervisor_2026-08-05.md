# Revisión F0-H0b Q5a: quinta captura y autoprueba del supervisor

Fecha: 5 de agosto de 2026.

## Resultado

**GO** sobre `649ee46fe6f2c4911a6f0bf83eed44b54f9844f0`, confirmado
por tres revisiones independientes:

| Revisión | Resultado | Severidades |
| --- | --- | --- |
| Seguridad | `GO` | `P0=0`, `P1=0`, `P2=0` |
| Funcional y ABI | `GO` | `P0=0`, `P1=0`, `P2=0` |
| Capacidad y contrato | `GO` | `P0=0`, `P1=0`, `P2=0` |

Q5a queda aceptada como prerrequisito de C4b-2. No acredita todavía el
supervisor operativo, el árbol Bash, plazos, espera, extinción, Docker,
PostgreSQL, H0b, C2, F0 ni producción.

## Historial vinculante

`4f36a9a` incorporó la fuente Go y la autoprueba ABI. Se detuvo porque el
entorno Go no estaba completamente aislado, faltaba la huella literal del
binario y las postcondiciones de carga podían quedar enmascaradas.

`6b6cb73` cerró build, fuente, binario y cargas, pero las revisiones detectaron
que una función `type` heredada eludía la puerta y que los nuevos controles se
habían compactado para conservar un ledger ya insuficiente.

La enmienda aceptada en `1b2f5ad` autorizó Bash `-p`, entorno crudo acreditado,
formato legible y traslado de las funciones de snapshot a D2d. `d52d5c2`
ejecutó el traslado y `2bd0311` aplicó el arranque protegido.

La revisión de `2bd0311` volvió a detener la integración: Bash 5.3 traza
`exec 9<&-` como `+ exec`, que la allowlist no admitía, y D2d había añadido una
supresión `SC2154` contraria a la puerta documental. `649ee46` limitó la
alternativa a `exec( |$)` y sustituyó la supresión por la dependencia explícita
`declare -g temporales`. Los tres revisores trabajaron sobre copias inmutables
del nuevo SHA.

## Evidencia final de seguridad

La revisión reprodujo:

- shebang y `/usr/bin/bash -p` aceptados; Bash sin `-p` rechazado con 65;
- `/usr/bin/bash`, `/usr/bin/env` y `/usr/bin/grep` físicos, ejecutables y no
  escribibles; ruta escribible mutante rechazada;
- `PIPESTATUS=(0 1)` como única continuación;
- productor, consumidor, cardinalidad, pareja, `BASH_FUNC_*` y `LD_*`
  adversos rechazados con 65;
- funciones `env` y `type` no importadas, no ejecutadas y no publicadas;
- `BASH_ENV` y `ENV` ignorados;
- `PS4`, `SHELLOPTS` y `BASHOPTS` retirados por el launcher de las sondas;
- cinco defectos de `probar_rechazos_m38_f0` con hijo 64, allowlist verde,
  agregado 0 y cero órdenes no admitidas;
- cuatro marcas privadas consumidas y seis destinos exactos `readonly`;
- entorno Go permitido, `GOAMD64=v3`, fuente/binario sustituidos y modos
  adversos cerrados;
- cero temporales nuevos y árbol inmutable.

La frontera confiable sigue empezando en launcher, EUID, kernel, cargador y
binarios administrados. El runner rechaza `LD_*`, pero no promete revertir una
biblioteca ya cargada por un proceso padre comprometido.

## Evidencia funcional y ABI

Se reprodujeron exactamente cinco fuentes y cuatro cargas Shell. El manifiesto
canónico contiene cinco pares ruta/SHA y las copias son regulares, 0600, del
usuario efectivo y de enlace único.

Huellas finales:

| Artefacto | SHA-256 |
| --- | --- |
| D2c | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| D2d | `9b137f1302c5672e9fd5c0c8df169810cbc7e57a11fa2129bf79a777e92c5e81` |
| H0b | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| Adaptador M38 | `98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7` |
| Supervisor Go | `c18622edb1ad5168752918ab6f9062b7cf06fbda490f68d1da2ff1ff90032a1e` |
| Capturador Go | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |
| Binario supervisor | `eb0764c58c7eb2d954abdecc65769a10cfb1f1b4080e5f8c23e417af2df78d86` |

El build cerrado usa Go 1.26.5, Linux/amd64, GOAMD64 v1, CGO desactivado para
el supervisor, proxy y red desactivados y `-trimpath`. El binario queda 0700.

La autoprueba real acreditó `pidfd_open` 434, `pidfd_send_signal` 424, flag 4,
señal cero, mutante `EINVAL`, continuidad de la referencia y `EBADF` tras el
cierre. El modo desconocido devuelve 64. La fuente Go no crea hijos,
goroutines, canales, SQL, Docker ni recursos de caso.

## Capacidad, calidad y alcance

| Fichero | Líneas | Límite local |
| --- | ---: | ---: |
| Runner | 789 | 790 |
| D2d | 145 | 155 |
| Adaptador | 527 | 527 en Q5a |
| Supervisor Go | 131 | 190 |

Las reservas siguientes permanecen en 6 líneas para C4b-2, 3 para C4b-3 y 2
para C4c. El máximo global DEC-051 continúa en 800. La allowlist solo amplió
`exec ` a `exec( |$)`. D2d conserva cinco supresiones históricas: base y final
coinciden, sin una nueva.

Puertas verdes:

- `bash -n` y ShellCheck `-x`;
- `gofmt`, `go vet` y builds aislados;
- autopruebas de capturador y supervisor;
- `git diff --check` y control de tamaños;
- hashes, invariantes, write-set y documentación;
- worktrees limpios y residuos cero.

Por instrucción no se ejecutaron Docker real, red, PostgreSQL ni la matriz/E2E
completa. Estas omisiones son correctas para Q5a y siguen siendo obligatorias
en C4b-2 y su integración posterior.

## Decisión de dirección

Se acepta Q5a sobre `649ee46`. Los contadores funcionales no aumentan: F0
permanece `10/23`, O4-05 `3/5`, Contratación temporal `24/46` y Bolsa
productiva `1/14`. Producción continúa en `NO-GO`.

La siguiente tarea única es C4b-2 operativo, después de publicar este corte y
observar su CI completa. Productor y revisores vuelven a ser distintos.
