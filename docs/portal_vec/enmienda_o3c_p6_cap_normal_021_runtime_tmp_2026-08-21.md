# Enmienda O3c P6 CAP_NORMAL_021: attestation del runtime temporal V4

Fecha: 21 de agosto de 2026.

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-ATTESTATION`.

Estado: candidato técnico local. Requiere pruebas focales, una única corrida
canónica y revisión funcional y de seguridad independientes sobre el SHA
exacto. No autoriza integración, publicación, CI remota, despliegue,
producción ni cambio de métricas.

## Base y autoridad

La base exacta es
`14c1f31079e466a82b8e1d390168078973cc6e05`, con árbol
`1c736bcb555326841485a1f0c0486626ad6a5038`. Conserva el publicador durable
V2 de `9391243f98d232a8eb78ed20ba5306e31b864bad` y sus dos revisiones
independientes de alcance filesystem:

- funcional `19ed5f03ed7905f56dabaed883d771fbfb2bd7e7`;
- seguridad `d9629d787c351ac5592ea091535407ca8bdad75f`.

Esos GO no revocaron el rojo histórico `CAP_NORMAL_021`, los dos NO-GO de
O3A-V5-CND-V3 ni el P1 C21/toolchain. Esta enmienda tampoco los convierte en
GO. V4 acredita únicamente el vínculo entre las 32 fuentes compiladas, el
checkout, el `TMPDIR` exacto de cada ejecución y el paquete local publicado.

## V3: GO productor revocado por revisión

La base V4 `14c1f31` obtuvo una única corrida productora verde y durable:
244 casos, seis BF, cien capturas normal y cien race, Go 1.26.5 y residuos
cero. Su `SHA256SUMS` es
`f1edf1a274843221a12c56f5cfc1abe9efd4bb4b460888f8948aa9699e26877b`.
Ese GO productor no es un dictamen y no se repetirá.

Las revisiones funcional y de seguridad independientes emitieron `NO-GO`
sobre ese SHA exacto. Al iniciar V4 sus actas todavía no estaban presentes
como objetos Git locales; los hallazgos transmitidos por dirección son:

- P1 funcional: la fila medía el contenedor `runtime_tmp` después de borrarlo,
  pero no el `TMPDIR` efectivo `runtime_aislado/tmp` antes de retirar el
  contenedor;
- P1 seguridad: la publicación verde usaba `mv` simple, sin no-replace ni
  identidad física;
- P1 seguridad: existía una ventana entre el hash de las fuentes del target y
  su reapertura posterior para compilar;
- P2 seguridad: el helper no acreditaba UID efectivo y modo 0700;
- P2 seguridad: utilidades y FD ambientales no quedaban acreditados.

V4 no convierte esos NO-GO en GO ni abre O3A-V4, C21, toolchain u O4.

## Candidatos rojos preservados y atribución corregida

El primer candidato `4d2951f83057490390c4e4e25927c5b06e0f868c` recibió una
única corrida canónica. Terminó `NO-GO` en `C17_OWNERS`, normal, estado 1,
stdout 200, stderr 0, grupo ausente e inventarios
`6/6,0/0,0/0,0/0,0/0`. El publicador V2 conservó el paquete exacto bajo la
raíz privada de evidencias del usuario `orquesta`; `SHA256SUMS` es
`a5c4434ee7e04c3877641ed1b1d6d2c63e3e66bfc43bb77ca8708249419fda64`.
No se repitió esa corrida.

El paquete sellado conserva solamente el fallo exterior y la línea que mide
la ejecución interior: `err=exit status 2`, stdout 48 y stderr 1927. No
conserva los 48 o 1927 bytes interiores, por lo que no acredita su contenido.
La primera atribución a una doble liberación del hilo fue una inferencia del
código —`defer runtime.UnlockOSThread()` local junto al `testing.Cleanup` de
`autoridadRealBarreraO3bPruebaM38`—, no una conclusión contenida en el raw.

El segundo candidato `6407fad5feb91330be3425c8feaea3fd31886460`
retiró el `defer` y recibió una sola focal agregada de los siete selectores.
Terminó también `NO-GO`, estado 1, stdout 200, stderr 0, con la misma medición
interior 48/1927. Su paquete tiene `SHA256SUMS`
`94e5fdca7f0de9297032f389e87c286042ec74ad4a7954293e69d04aacce42f2`.
No recibió corrida canónica y no se repitió la focal. Este resultado refuta la
atribución a la doble liberación y obliga a retirarla.

La causa estructural visible es el `testing.Cleanup` heredado:
`consolidarHandoffO3bM38` entrega la custodia y fija
`autoridadCapturaO3bM38.custodia=nil`, mientras el cleanup registrado por
`autoridadRealBarreraO3bPruebaM38` dereferencia después esa custodia. El
retorno normal activa ese cleanup fuera de su precondición. Este corte no
modifica el helper O3b ajeno: restaura los dos `os.Exit(0)` originales del
hijo y el `defer runtime.UnlockOSThread()` original. La nueva capacidad de
retirada temporal pertenece al padre y ocurre después de `cmd.Run`, con
independencia de que el hijo termine mediante `os.Exit`.

Una sonda anterior, ejecutada fuera del aislamiento canónico, heredó un FD
ambiental y falló antes de alcanzar los selectores. Se conserva como intento
inválido y no cuenta como GO, NO-GO ni evidencia causal.

Son autoridad funcional directa:

- la decisión O3c, en particular C21 y C22;
- el conductor O3c P6 y su ledger de 32 fuentes;
- las dos revisiones NO-GO de O3A-V5-CND-V3;
- la enmienda y las dos revisiones del publicador durable V2.

## Capability y criterio único

La única capability de V4 es una attestation no suplantable de la ejecución:
las fuentes compiladas son un snapshot privado de las 32 rutas exactas; el
checkout mantiene HEAD, árbol y limpieza; cada selector usa un `TMPDIR`
privado de UID efectivo y modo 0700; y el paquete GO se publica localmente sin
reemplazo, conservando identidad dev:inode.

Cada selector aislado de `TestHandoffO3cP5CasosAislados` recibe un directorio
runtime privado. El proceso padre mide las entradas del `TMPDIR` efectivo
antes y después del hijo, registra también los residuos que existen antes de
la limpieza, y retira solamente esas entradas conservando el directorio y su
identidad física. La atestación TSV se precrea fuera del `TMPDIR` del hijo,
congela su huella dev:inode/UID/modo y se filtra de su entorno; cualquier
sustitución de esa ruta invalida el agregado. Después de `Lstat=ENOENT` del selector, el padre
acredita que la raíz exterior queda vacía; la retirada del contenedor se
acredita en una columna separada, también cuando el hijo termina con estado
fatal 65.

## Invariantes

Para cada uno de los siete selectores:

| Selector | Estado esperado |
| --- | ---: |
| `positivo` | 0 |
| `retirada` | 0 |
| `retirada_terminal` | 0 |
| `reuso` | 65 |
| `particion` | 65 |
| `retirada_sin_ref` | 65 |
| `retirada_plazo` | 65 |

Se exige conjuntamente:

1. stdout y stderr de cero bytes;
2. entradas del `TMPDIR` efectivo parten de cero; el preconteo de residuos se
   conserva por selector, el padre los retira, y el conteo final es cero antes
   de retirar el contenedor;
3. ningún hijo, zombi o grupo residual;
4. la limpieza temporal la realiza el padre después de `cmd.Run` y no depende
   de `defer` o `testing.Cleanup` en el hijo;
5. los recorridos verdes conservan sus `os.Exit(0)` originales para no activar
   un cleanup O3b cuya autoridad ya fue transferida;
6. el contenedor runtime queda retirado y acreditado separadamente;
7. cada selector acredita dev/inode/UID/modo, entradas iniciales, residuos
   pre-limpieza, `Lstat=ENOENT` y raíz exterior vacía en `tmpdir_selectores.tsv`;
8. las fuentes compiladas son regulares, no symlink, byte-exactas al ledger y
   se leen únicamente desde el snapshot privado;
9. HEAD, árbol y limpieza del target son iguales antes y después del snapshot;
10. cada hijo hereda solo stdin, stdout y stderr; todo FD ambiental `>=3` se
   cierra antes de `exec`;
11. la publicación GO no reemplaza un destino, elimina el origen de
   publicación y conserva dev:inode, UID y modo 0700;
12. ningún estado se reintenta, tolera, reclasifica, salta o decide por
   mayoría.

La corrida canónica conserva además 244 casos, seis BF directos, cien
capturas normal y cien race, 1.472 filas de atestación de selectores (cinco
agregados y cien capturas por modo, más C17 por modo), FD/hijos/zombis/grupos/temporales delta cero y
el toolchain literal `go version go1.26.5 linux/amd64`.

## Diseño acotado

### Arnés Go

El padre valida que `TMPDIR` sea un directorio absoluto real, vacío, propiedad
del UID efectivo y de modo 0700. Antes de cada selector crea una raíz con la
misma autoridad, reemplaza de forma explícita `HOME`,
`TMPDIR`, `GOTMPDIR` y `O3C_P5_CASO` en el entorno hijo, espera su estado,
contiene descendientes y retira la raíz. Solo entonces comprueba estado y
salidas.

Los dos puntos de éxito conservan sus `os.Exit(0)` originales y la fijación de
hilo conserva su `defer` original. El proceso padre no delega en esas salidas
la limpieza temporal: espera el estado, contiene descendientes y retira la
raíz privada por igual en éxito y en fatal 65.

El BF directo de partición usa un padre test-only que ejecuta una sola vez el
selector fatal, acredita y retira su temporal privado y solo entonces termina
él mismo en 65. La variable del BF no se hereda al selector, evitando recursión
y conservando estado 65, EOF y salida 0/0.

### Conductor

Compilación y ejecución tienen temporales distintos. El conductor copia las
32 fuentes a un snapshot 0700 conservando sus rutas relativas. Rechaza ruta
absoluta o con ascenso, origen o copia no regular/symlink y cualquier hash
distinto. Revalida HEAD, árbol y limpieza después de copiar y compila normal y
race exclusivamente desde el snapshot.

Cada invocación recibe un entorno vacío y cerrado con `HOME`, `TMPDIR`,
`GOTMPDIR`, `GOROOT`, `GOENV=off` y `GOTOOLCHAIN=local` explícitos. El proceso
intermedio cierra todos los FD `>=3` antes de `exec`; el lock nunca llega al
target. Las rutas y huellas SHA-256 de las utilidades externas se conservan en
la evidencia.

La fila registra tanto el inventario del contenedor como el del `TMPDIR`
efectivo antes/después, el preconteo de residuos por selector, su acreditación,
la retirada del contenedor y el cierre de FD ambientales. El publicador V2
recibe sin cambios el primer fallo. Un GO se prepara en un temporal 0700 hermano
del destino, se sella allí, se retira explícitamente el staging y solo después
se revalida el target; finalmente se mueve con `mv -n -T` tras acreditar origen
ausente, destino real y la misma huella dev:inode, UID y modo.

## Write-set exacto

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go
tools/o3c_p6_conductor/conductor.sh
tools/o3c_p6_conductor/fuentes.tsv
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
```

`fuentes.tsv` cambia solo la huella de `handoff_test.go`. No se modifican
fuentes productivas, G7a, `fallo_durable.sh`, casos, ledgers O3a/O3b,
workflows, estado transversal o métricas.

## Puertas y secuencia sin reintentos

Después del commit candidato y antes de ejecutar conducta:

1. identidad, genealogía, limpieza, write-set y hashes;
2. `gofmt -d` del único Go modificado;
3. `bash -n` y ShellCheck del conductor, si la herramienta ya está presente;
4. build normal y race de las 32 fuentes del ledger, sin ejecutar binarios;
5. validación mecánica de las 32 huellas del ledger;
6. `go vet` focal, `git diff --check` y barrido de write-set.

Después de esas puertas no conductuales se permite una única corrida canónica
del conductor O3c, propiedad de `orquesta`, bajo lock cerrado, a un destino de
evidencia nuevo. No se ejecuta una sonda ad hoc previa. Si termina roja, el
publicador V2 conserva el primer paquete y ese SHA no se repite. Si termina
verde, el mismo SHA recibe dos revisiones independientes; el productor no
emite su propio GO.

## Límites

El corte no interpreta stdout/stderr históricos, no borra residuos o
evidencias previas y no relaja cardinalidades, plazos, oráculos o contención.
No acredita la estabilidad C21 ni el parche de toolchain, no abre O4 y no
autoriza datos reales, red, SQL, Docker, PostgreSQL, push, despliegue,
producción o credenciales.
