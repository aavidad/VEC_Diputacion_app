# Enmienda O3c P6 CAP_NORMAL_021: runtime temporal aislado V3

Fecha: 21 de agosto de 2026.

Tarea: `O3C-P6-CAP-NORMAL-021-RUNTIME-TMP-V3`.

Estado: candidato técnico local. Requiere pruebas focales, una única corrida
canónica y revisión funcional y de seguridad independientes sobre el SHA
exacto. No autoriza integración, publicación, CI remota, despliegue,
producción ni cambio de métricas.

## Base y autoridad

La base exacta es
`2cbcbae157a18eaf62e3de7f989b68be1e915941`, con árbol
`d9d22407568670ddce2e631302f02162f25f2310`. Conserva el publicador durable
V2 de `9391243f98d232a8eb78ed20ba5306e31b864bad` y sus dos revisiones
independientes de alcance filesystem:

- funcional `19ed5f03ed7905f56dabaed883d771fbfb2bd7e7`;
- seguridad `d9629d787c351ac5592ea091535407ca8bdad75f`.

Esos GO no revocaron el rojo histórico `CAP_NORMAL_021`, los dos NO-GO de
O3A-V5-CND-V3 ni el P1 C21/toolchain. Esta enmienda tampoco los convierte en
GO: corrige únicamente la causa demostrada por la sonda de siete selectores,
que obtuvo los estados esperados y salida cero, pero acumuló una entrada en
el runtime temporal por selector.

Son autoridad funcional directa:

- la decisión O3c, en particular C21 y C22;
- el conductor O3c P6 y su ledger de 32 fuentes;
- las dos revisiones NO-GO de O3A-V5-CND-V3;
- la enmienda y las dos revisiones del publicador durable V2.

## Capability y criterio único

Cada selector aislado de `TestHandoffO3cP5CasosAislados` recibe un directorio
runtime privado, atribuible al selector y creado dentro del `TMPDIR` privado
de su ejecución. El proceso padre contiene descendientes y retira ese
directorio después de esperar al hijo, también cuando el hijo termina con
estado fatal 65.

El conductor crea una raíz runtime separada del temporal de compilación,
ejecuta cada caso en un subdirectorio privado y mide exactamente esa raíz.
El subdirectorio de ejecución se retira después de acreditar la ausencia del
grupo. El inventario antes y después observa por tanto el mismo `TMPDIR` que
usan el binario, sus subprocesos y `os.MkdirTemp("", ...)`.

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
2. temporales antes=después=0 en la raíz privada;
3. ningún hijo, zombi o grupo residual;
4. la limpieza la realiza el padre y no depende de `defer` en el hijo fatal;
5. los recorridos verdes retornan normalmente para ejecutar `testing.Cleanup`;
6. ningún estado se reintenta, tolera, reclasifica, salta o decide por mayoría.

La corrida canónica conserva además 244 casos, seis BF directos, cien
capturas normal y cien race, FD/hijos/zombis/grupos/temporales delta cero y
el toolchain literal `go version go1.26.5 linux/amd64`.

## Diseño acotado

### Arnés Go

El padre valida que `TMPDIR` sea un directorio absoluto real y vacío. Antes de
cada selector crea una raíz `0700`, reemplaza de forma explícita `HOME`,
`TMPDIR`, `GOTMPDIR` y `O3C_P5_CASO` en el entorno hijo, espera su estado,
contiene descendientes y retira la raíz. Solo entonces comprueba estado y
salidas.

Los dos puntos de éxito que antes llamaban `os.Exit(0)` retornan normalmente.
Así se ejecutan las limpiezas registradas por el arnés; los cuatro caminos
fatales conservan 65 y siguen siendo limpiados desde el padre.

### Conductor

Compilación y ejecución tienen temporales distintos. Cada invocación recibe
un entorno vacío y cerrado con `HOME`, `TMPDIR`, `GOTMPDIR`, `GOROOT`,
`GOENV=off` y `GOTOOLCHAIN=local` explícitos. El lock del conductor se cierra
antes del `exec` del target.

La raíz runtime se inventaría antes y después de cada fila. El conductor
retira el subdirectorio propio solo después de contener el grupo y marca la
fila NO-GO si la retirada o la ausencia no se acreditan. El publicador V2
recibe sin cambios el primer fallo y lo publica en el destino nuevo exacto.

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

Antes del commit:

1. identidad, genealogía, limpieza, write-set y hashes;
2. `gofmt -d` del único Go modificado;
3. `bash -n` y ShellCheck del conductor, si la herramienta ya está presente;
4. build normal y race de las 32 fuentes del ledger;
5. una única ejecución focal normal de los siete selectores, cada selector
   una vez, estados exactos, salida cero y cinco inventarios sin delta;
6. validación mecánica de las 32 huellas del ledger;
7. `go vet` focal, `git diff --check` y barrido de write-set.

Solo después del commit candidato se permite una única corrida canónica del
conductor O3c, propiedad de `orquesta`, bajo lock cerrado, a un destino de
evidencia nuevo. Si termina roja, el publicador V2 conserva el primer paquete
y no se repite. Si termina verde, el mismo SHA recibe dos revisiones
independientes; el productor no emite su propio GO.

## Límites

El corte no interpreta stdout/stderr históricos, no borra residuos o
evidencias previas y no relaja cardinalidades, plazos, oráculos o contención.
No acredita la estabilidad C21 ni el parche de toolchain, no abre O4 y no
autoriza datos reales, red, SQL, Docker, PostgreSQL, push, despliegue,
producción o credenciales.
