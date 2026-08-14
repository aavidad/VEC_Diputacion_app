# Enmienda O3a V5 CND V3: inventario FD exacto de C21

Fecha: 14 de agosto de 2026.

Tarea: `O3A-V5-CND-V3-C21-INVENTARIO-EXACTO`.

Estado: candidato técnico local. Requiere revisión funcional y de seguridad
independientes. No acredita estabilidad, toolchain, O4, publicación ni CI.

## Base y criterio único

La base exacta es
`ec02febaa3080cb8e7894340075beb850ea54348`, con padre
`a997acbb9a51c02168f1648225c0e69c84e648d4` y árbol
`e7c5dfaaa96b76ed9d25733456d56c0261ac98db`.

El contrato O3a exige que C21 reproduzca inventarios FD inicial y final
idénticos. La base solo contaba descriptores vivos y comparaba dos enteros.
Ese oráculo detectaba altas o bajas netas, pero podía aceptar que un FD se
cerrase y otro recurso ocupase el mismo número, o que una baja y un alta
distintas conservasen la cardinalidad.

El criterio único de V3 es comparar un inventario estructurado por cada número
de FD vivo. No pretende atribuir retroactivamente el estado agregado 66 del
rojo histórico ni corregir una causa aún desconocida.

## Write-set exacto

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go
tools/o3a_v5_conductor/fuentes_v5.tsv
tools/o3b_p7_conductor/fuentes.tsv
tools/o3c_p6_conductor/fuentes.tsv
docs/portal_vec/enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
```

G7a es test-only por `//go:build ignore`. Los tres ledgers son autoridades
vivas que consumen exactamente ese fuente y su actualización es mecánica. G7b,
producción, conductores, workflow, evidencias históricas, PostgreSQL,
credenciales, estado transversal y métricas permanecen byte a byte.

## Inventario cerrado

`inventarioFDVivosPruebaO3aM38` enumera `/proc/self/fd`, convierte cada nombre
a número y comprueba que el descriptor siga vivo con `F_GETFD`. Una entrada
obsoleta que ya devuelve `EBADF` se omite; cualquier otro fallo cierra el caso
como error de snapshot.

Para cada FD vivo conserva una tupla comparable de diez campos:

```text
FD -> {FD flags, status flags, device, inode, rdev, mode,
       uid, gid, link count, size}
```

La igualdad usa número y tupla completa. Por tanto una sustitución con la misma
cardinalidad, incluidos un cierre y una reapertura sobre el mismo número pero
con otra identidad física, termina en el estado test-only 104. Los estados 99,
100, 101, 102, 103 y 105, el orden de limpieza y la exigencia de hijos cero no
cambian. Ningún estado negativo se acepta como éxito.

El inventario no abre ni duplica los FD examinados, no los convierte en
`*os.File`, no cambia sus flags y no relaja `CLOEXEC`, lease, custodia ni
cardinalidad. El descriptor efímero usado por `os.ReadDir` ya está cerrado al
comprobar las entradas y solo se omite si `F_GETFD` acredita `EBADF`.

## Presupuesto y huellas

G7a queda en exactamente 750 líneas, en la parada local y por debajo del tope duro 800,
con SHA-256
`2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3`.
G7b permanece en 744 líneas y SHA-256
`5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e`.
Los tres ledgers vivos fijan la nueva huella de G7a y conservan el resto de sus
entradas.

## Puertas requeridas

1. identidad, genealogía, write-set, modos, líneas y hashes exactos;
2. `gofmt`, `go vet` y builds normal/race de los diez fuentes;
3. mutante efímero de igual cardinalidad que sustituye un FD vivo por
   `/dev/null` sobre el mismo número y exige estado 104, normal y race, con
   stdout y stderr vacíos;
4. una única corrida contractual O3a normal/race en clon limpio propiedad de
   `orquesta`, sin retry, mayoría, `SKIP`, sleep o tolerancia;
5. conductores O3b P7 y O3c P6 con su toolchain autorizada y `flock --close`,
   porque comparten G7a y sus ledgers vivos;
6. calidad global proporcional, `git diff --check`, Gitleaks y residuos cero;
7. revisión funcional y de seguridad independientes del SHA exacto.

Una puerta roja se conserva con su primer estado y no se repite para buscar
verde. Una corrida verde solo demuestra compatibilidad del inventario nuevo;
no revoca el P1 C21 sellado sobre `74f2495`.

## Límites y relevo

V3 cierra únicamente el falso verde estático de igualdad por cardinalidad. La
causa histórica sigue necesitando el primer negativo natural de una ejecución
canónica instrumentada con estados 99--105. No se implementa una corrección de
cleanup, hijos, netpoll, resultado o runtime sin ese dato.

No se autoriza integración, push, despliegue, producción, credenciales,
publicación, CI remota ni cambio de porcentajes.
