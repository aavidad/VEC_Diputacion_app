# Enmienda F0-H0b Q5a: arranque protegido y formato auditable

Fecha: 4 de agosto de 2026.

Estado: segunda propuesta de dirección **aceptada** sobre `4c0f3b1` tras dos
contrarrevisiones independientes con `P0=0`, `P1=0` y `P2=0`. La primera,
`935bb28`, recibió `GO` de capacidad y dos `NO-GO P1` por el entorno de las
sondas con traza y por no distinguir el estado completo de la tubería. Autoriza
solo los dos commits Q5a definidos en este documento. No cierra Q5a, C4b-2,
H0b, C2, F0 ni habilita producción.

## Motivo

El primer candidato Q5a, `4f36a9a`, incorporó la quinta fuente y la autoprueba
ABI. Sus revisiones detuvieron la integración porque el build heredaba parte
del entorno, no aislaba todas las llamadas a Go y no acreditaba literalmente
el binario construido.

La corrección `6b6cb73` cerró esos defectos técnicos con build privado,
entorno permitido, huellas anterior y posterior y binario determinista. La
segunda contrarrevisión confirmó esas propiedades, pero emitió dos `NO-GO P1`:

1. `type -t env` podía resolver y ejecutar una función `type` exportada por un
   entorno adverso antes de la frontera Go;
2. el candidato añadió un cuarto literal de huella y nuevos controles sin
   ampliar el ledger, por lo que mantuvo artificialmente 773 líneas juntando
   sentencias y comprobaciones independientes.

Ambos dictámenes son vinculantes. Esta enmienda sustituye solo el arranque, el
write-set y el presupuesto Q5a de la
[decisión del supervisor](decision_f0_h0b_c4b2_captura_quinta_supervision_exterior_2026-08-04.md).
El resto de aquella decisión permanece vigente.

## Arranque protegido obligatorio

El runner es Linux específico y su entrada controlada usa el Bash físico
`/usr/bin/bash` en modo privilegiado de interpretación, que no importa
funciones del entorno ni procesa `BASH_ENV` o `ENV`:

```text
#!/usr/bin/bash -p
```

El runner exige inmediatamente que la bandera `p` esté activa. La ejecución
admitida es el fichero ejecutable mediante su shebang o, cuando un componente
acreditado debe abrir una copia por descriptor, `/usr/bin/bash -p`. Invocarlo
mediante `bash fichero` sin `-p` no forma parte del contrato y termina en 65
antes de la lógica del runner.

Después de comprobar el modo protegido y antes de exportar, borrar variables,
crear temporales o seleccionar herramientas, el runner inspecciona el entorno
crudo con `/usr/bin/env -0` y `/usr/bin/grep -zE`. La salida de `grep` se dirige
a `/dev/null`: un cuerpo de función nunca llega a stdout, stderr o log. El
patrón anclado `^(BASH_FUNC_|LD_)` rechaza cualquier función exportada, no solo
`env` o `type`, y cualquier variable del cargador dinámico.

La puerta conserva inmediatamente los dos elementos de `PIPESTATUS`. La única
pareja de éxito es `env=0, grep=1`, que significa productor correcto y ausencia
de coincidencias:

- `0,0`: existe al menos una función exportada, estado 65;
- `0,1`: no existe ninguna, se puede continuar;
- cualquier otra pareja, cardinalidad o valor: fallo de la puerta, estado 65.

La comprobación se expresa como una tubería usada directamente por `if`, para
que `errexit` no cierre antes de guardar `PIPESTATUS`. En la rama de
coincidencia termina en 65. En la rama sin coincidencia copia primero el array
y exige cardinalidad dos y valores `0,1`; no reduce la pareja a `$?` ni al
estado de `grep`. Así, un productor fallido seguido de `grep=1` tampoco avanza.

No se usa `grep -q`, porque con `pipefail` su cierre anticipado podría convertir
una coincidencia en un falso error de tubería. No se invocan `type`, `command`,
`which`, una función heredada ni una ruta resuelta por `PATH` para esta puerta.
Las rutas físicas `/usr/bin/bash`, `/usr/bin/env` y `/usr/bin/grep`, siguiendo
enlaces de una instalación `usr-merge`, deben resolver a ficheros regulares del
sistema no escribibles por el usuario efectivo. Una instalación que no cumpla
esa precondición es incompatible y falla cerrada.

El entorno crudo puede conservar las cadenas `BASH_FUNC_*` aunque Bash `-p` no
las importe. Precisamente por eso la detección se hace sobre `/usr/bin/env -0`
y no sobre `declare -F`. Una combinación adversa de funciones exportadas
`env` y `type` debe terminar en 65 sin ejecutar ninguna de las dos.

La frontera Go conserva `/usr/bin/env -i` y su lista permitida. También sigue
rechazando `GOAMD64` heredado. El modo protegido no reemplaza el entorno vacío:
son defensas distintas y acumulativas.

La frontera confiable empieza antes de cargar el ejecutable: identidad/EUID,
kernel, cargador dinámico y binarios del sistema pertenecen al launcher
administrado. Bash `-p` no puede neutralizar una biblioteca ya ejecutada por
`LD_PRELOAD`, ni defenderse de `ptrace` o de un actor que controle el mismo
EUID. El launcher de CI debe entregar un entorno de cargador permitido, sin
`LD_*`; el runner rechaza su presencia, pero no afirma revertir un efecto
anterior a su primera instrucción. Esta limitación no debilita la puerta contra
funciones o configuración Go heredadas: evita presentar una garantía imposible
frente a un proceso padre o anfitrión ya comprometido.

## Invocaciones coherentes

Q5a puede modificar una sola línea del adaptador privado para que el Bash de
caso provisional se abra con `/usr/bin/bash -p`. Las dos pruebas adversas que
abren el runner por FD usan este launcher cerrado:

```text
/usr/bin/env -i
  PATH=/ruta-no-resoluble
  LC_ALL=C
  BASH_XTRACEFD=6
  /usr/bin/bash -p -x /proc/self/fd/8 ...
```

El entorno vacío evita que `PS4`, `SHELLOPTS`, `BASHOPTS`, `BASH_ENV`, `ENV`,
`CDPATH`, `GLOBIGNORE` o una función lleguen al Bash con traza. Solo esas dos
sondas necesitan `-x`; el FD 6 ya acreditado sigue siendo su único destino. No
cambian sus tickets, descriptores, estados ni semántica.

La forma exacta futura de C4b-2 queda corregida a:

```text
Cmd.Path = /usr/bin/bash
Cmd.Args = [/usr/bin/bash, -p, /proc/self/fd/8,
            --caso-inyeccion-h0b, SELECTOR_LITERAL]
```

El supervisor construirá además un entorno permitido sin `BASH_ENV`, `ENV` ni
`BASH_FUNC_*`. Esta enmienda sustituye solo la lista `Cmd.Args` anterior; no
autoriza todavía la implementación operativa C4b-2.

## Separación estructural sin sexta fuente

Para restaurar formato legible sin superar DEC-051 se trasladan, sin cambiar
su semántica, estas tres funciones ya existentes del runner al auxiliar D2d
`operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`:

- `acreditar_snapshot_contenedor_f0`;
- `rechazar_snapshot_adverso_f0`;
- `probar_snapshot_adverso_f0`.

D2d ya posee el descubrimiento, acreditación, huella y retirada del contenedor;
por ello las comprobaciones de su snapshot pertenecen a la misma autoridad. Se
cargan antes de su primer uso y pueden seguir llamando a las funciones base del
runner. El traslado no crea API pública, dependencia circular, sexto source ni
cambio funcional. D2d conserva el rechazo directo 64 y el consumo único de
`VEC_F0_CARGA_PRIVADA`.

La nueva huella de D2d se fija literalmente en el runner y en el manifiesto de
cinco. La línea `-p` del adaptador también obliga a actualizar su huella
literal. D2c, H0b y capturador permanecen byte a byte invariantes.

El traslado forma un commit de refactorización autónomo. El arranque protegido,
las nuevas huellas, el reflujo y los mutantes forman un commit de corrección
posterior. Ninguno se integra sin revisar el estado final conjunto.

## Formato y presupuesto vinculantes

Se autoriza la huella literal del binario del supervisor como cuarto valor de
integridad añadido por Q5a. Los controles independientes no se unen con `;`,
`&&` ni en una sola condición para simular que caben. Se restauran saltos y
bloques legibles, especialmente en:

- declaraciones nuevas y helper de entorno;
- `vet` y los dos builds privados;
- invocación del capturador y comparación de las cinco entradas;
- ausencia, carga y postcondiciones de los seis destinos del adaptador;
- forma y huellas posterior de fuente y binario;
- rechazo del modo desconocido.

Las cifras son líneas físicas `wc -l` y el máximo global de DEC-051 sigue
siendo 800:

| Corte | Runner | D2d | Adaptador | Supervisor Go |
| --- | ---: | ---: | ---: | ---: |
| Base corregida `6b6cb73` | 773 | 96 | 527 | 131 |
| Q5a tras traslado y reflujo | 755..790 | 140..155 | 527 | 130..190 |
| C4b-2 | hasta 795 | sin crecimiento material | 543..555 | 330..450 |
| C4b-3 | hasta 798 | sin crecimiento material | 551..569 | 365..490 |
| C4c | hasta 800 | sin crecimiento material | hasta 600 | inmutable |

Q5a se detiene si el runner supera 790, D2d supera 155, el adaptador cambia su
número de líneas o Go supera 190. C4b-2 y siguientes se detienen en sus límites
locales. Llegar a un límite no crea reserva; cualquier necesidad adicional
exige separar otra responsabilidad o una nueva decisión. No se relaja el
objetivo general de 500 ni el tope duro 800.

## Write-set excepcional de Q5a

| Commit | Ficheros de código autorizados |
| --- | --- |
| Refactorización | runner y D2d |
| Corrección | runner y adaptador privado |

La documentación Q5a acompaña ambos commits. La fuente Go puede permanecer
inmutable si reproduce la misma huella. H0b, D2c y capturador no se modifican.

## Puertas de aceptación

Además de las puertas Q5a ya aprobadas técnicamente, el candidato debe probar:

1. entrada ejecutable y `/usr/bin/bash -p`: aceptadas;
2. Bash sin `-p`: 65 antes de la lógica del runner;
3. `BASH_ENV` y `ENV` adversos bajo `-p`: ninguno se procesa y sus marcadores
   permanecen ausentes;
4. funciones exportadas `env`, `type` y ambas combinadas: 65, sin ejecutar sus
   cuerpos;
5. ausencia de funciones exportadas: `PIPESTATUS=(0 1)` y el flujo continúa;
6. productor o `grep` fallidos, otra cardinalidad o cualquier otra pareja: 65,
   nunca ausencia falsa;
7. sondas `-x` bajo `PS4`, `SHELLOPTS` y `BASHOPTS` adversos: el cuerpo de `PS4`
   no se ejecuta y la traza usa el prefijo literal predeterminado;
8. `GOAMD64=v3`, cualquier `LD_*`, binario sustituido, marca previa y destino
   `readonly`: 65;
9. cinco fuentes, cuatro cargas, seis postcondiciones y ambas huellas: exactas;
10. funciones trasladadas disponibles solo después de cargar D2d y equivalencia
   de sus mutantes de snapshot;
11. `bash -n`, ShellCheck sin supresiones nuevas, `gofmt`, `go vet`, build
    reproducible, autoprueba ABI, `git diff --check` y residuos cero.

Q5a no ejecuta Docker, PostgreSQL, red ni el runner E2E. Esas pruebas pertenecen
a la integración operativa posterior. Un productor distinto de los revisores
implementa; seguridad y capacidad emiten al menos dos dictámenes independientes
sobre el mismo SHA final.
