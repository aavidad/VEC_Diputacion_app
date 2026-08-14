# Enmienda O3c P6: primer fallo durable del conductor

Fecha: 14 de agosto de 2026.

Tarea: `O3C-P6-CONDUCTOR-FALLO-DURABLE-P0`.

Estado: candidato técnico local. Requiere revisión funcional y de seguridad
independientes. No corrige ni acredita O3a V3, C21, toolchain, O4, publicación
o CI.

## Base y causa del corte

La base exacta es
`fea52f3ddf796991c93c85cae992ca695db1ae63`, con padre
`ec02febaa3080cb8e7894340075beb850ea54348` y árbol
`c9d5a393f39d48a5015848d5fd49a0b69959c622`.

La única corrida O3c P6 exigida a esa base terminó en:

```text
CAP_NORMAL_021 normal estado=1 stdout=197 stderr=0 grupo=si
inventario=5/5,0/0,0/0,0/0,0/0
```

No se repitió. Las revisiones funcional `4bfb7b43069e20a0ccb1eaf6e2a80b2ff462a14f`
y de seguridad `29430b4ce8592e6e2bde74b69e37e44437091a77`
conservaron el resultado como `NO-GO P1`. El conductor anterior eliminó el
staging al salir y solo dejó los tamaños en su mensaje terminal; por tanto los
197 bytes no son reconstruibles ni se les atribuye contenido.

Una sonda posterior ejecutó una sola vez cada uno de los siete selectores. Sus
estados y salidas fueron los esperados, pero la sonda tuvo su propio `NO-GO`
por temporales del runner. No compensa el rojo ni permite inferir su causa.

## Criterio único

Ante el primer caso ordinario o BF que no cumpla su oráculo, el conductor:

1. conserva la fila `NO-GO` ya añadida al TSV parcial;
2. copia stdout y stderr íntegros sin interpretarlos;
3. fija ID, modo, estado, inventarios anterior y posterior y ausencia de PGID;
4. liga HEAD, Go, conductor, publicador, matriz, ledger, fuentes y binarios;
5. genera y verifica `SHA256SUMS` antes de publicar;
6. renombra dentro del directorio padre un paquete privado completo;
7. devuelve el mismo fallo y nunca escribe un resumen `GO`.

El destino debe no existir. Una publicación previa no se sobrescribe. El
paquete usa directorio `0700`, ficheros `0600` y se crea primero con `mktemp`
en el mismo padre para que el último `mv` sea un rename atómico local.

## Write-set exacto

```text
tools/o3c_p6_conductor/conductor.sh
tools/o3c_p6_conductor/fallo_durable.sh
docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md
```

No cambian G7a/G7b, fuentes Go productivas o test-only, casos, ledgers,
evidencias históricas, workflow, Docker, PostgreSQL, estado transversal ni
métricas. El nuevo helper pertenece exclusivamente al conductor O3c.

## Contenido del paquete

El paquete negativo contiene, como mínimo:

```text
fuentes.tsv
binarios.tsv
contexto.tsv
casos.tsv
bf_directos.tsv
fallo.tsv
fallo.stdout
fallo.stderr
resumen.txt
SHA256SUMS
```

`contexto.tsv` enlaza el candidato y todas las autoridades ejecutables. En una
salida verde también queda incluido y el resumen liga explícitamente el SHA del
publicador. La evidencia verde conserva el criterio anterior: solo se mueve
después de 244 casos, seis BF, residuos cero y checksums válidos.

Los bytes crudos permanecen fuera de Git y con permisos privados. Pueden
contener rutas diagnósticas locales; las actas solo reproducirán datos
minimizados y hashes. Nunca contienen credenciales, datos personales reales ni
autorización para producción.

## Presupuesto y huellas

El conductor ocupa 182 líneas y SHA-256
`6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538`.
El publicador ocupa 94 líneas y SHA-256
`8827cd2953aaf0787897062fd1652ef716eaef969717f803715df85573ba7641`.

## Puertas requeridas

1. identidad, genealogía, único write-set, modos, líneas y hashes;
2. `bash -n` y ShellCheck sin exclusiones nuevas;
3. autoprueba que acredita copia byte a byte, `NO-GO`, checksum y rechazo de
   sobrescritura;
4. mutantes que omiten stdout, falsean el resumen, permiten sobrescritura o
   excluyen los raw de `SHA256SUMS`, todos muertos;
5. integración sintética con un binario que devuelve uno: primer caso rojo,
   paquete completo, E/S conservadas, inventarios exactos y exit no cero;
6. una única corrida canónica O3c normal/race sobre clon limpio propiedad de
   `orquesta`, Go 1.26.5 y `flock --close`, sin retry ni mayoría;
7. `git diff --check`, Gitleaks, residuos cero y doble revisión independiente.

Si la corrida canónica vuelve a fallar, su paquete es el único dato causal
nuevo y no se repite. Si termina verde, solo acredita compatibilidad del
publicador y tampoco revoca `CAP_NORMAL_021` ni el P1 C21 histórico.

## Límites

No se autoriza push, despliegue, CI remota, producción, credenciales, cambio de
porcentajes, integración del candidato rechazado ni corrección de código O3c
sin una causa preservada. Este corte mejora la evidencia de fallo; no transforma
un fallo en éxito.
