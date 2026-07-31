# Revisión final C2.2-B: vínculo corporativo RRHH versionado

Fecha: 31 de julio de 2026.

## Veredicto

**GO técnico independiente: P0=0, P1=0 y P2=0.**

C2.2-B supera la matriz técnica compuesta en el corte estable
`de6e7df0e377c6e53b4ff1209b994bf3c006c536`. Acredita un modelo durable y
reversible del vínculo corporativo RRHH; no acredita una capacidad funcional
productiva. No autoriza datos reales, publicación, selección, API, web ni
despliegue. Producción VEP continúa en **NO-GO**.

## Objetivo, alcance y fuente de verdad

El corte conserva, dentro de ContextoActor, una historia de solo adición y un
puntero actual que ligan versiones exactas de cuenta, persona, perfil, vínculo
ContextoActor y organización para la superficie `interna_corporativa` y el
uso `consulta_rrhh`.

La fuente técnica es la
[decisión C2.2](../decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md)
y su concreción verificable es el
[contrato C2.2-B](../coordinacion_c2_2_b_vinculo_corporativo_2026-07-31.md).
La revisión se hizo sobre PostgreSQL 18.4 real fijado por digest.

Quedan fuera: publicación y revocación; selección, cardinalidades y recibo
privado; fachada, reconciliación, API, CLI, MCP, HTTP, web o escritorio;
autorización PDP; efectos de Contratación y Bolsa; AD, nómina, RPT o datos
reales; composición productiva, TLS/mTLS productivos y E2E funcional; y las
aprobaciones ENS, EIPD y de protección de datos.

## Corte estable revisado

| Minitarea | Commit o serie estable | Resultado final |
| --- | --- | --- |
| B0 | `a8e7500797035f6495dc9d589e94b19a812a9e99`, `63c4d2b9fadeeacdd7fdab491ec1140d47ddc7dd` | Contrato y errata sobre consumidores dinámicos externos. |
| B-S0 | `4dcd648e73cf1fd0ac47aba7d551490d285a4af4` | Runner base extraído sin cambio semántico. |
| B-S1 | `69489cdfb7c7e4af5b9efaa2829cfd35dfad0a75` | Primario PostgreSQL 18.4 acreditado por TCP estable. |
| B1 | `f050a6180b748a11fedd39b2bf738d17e8751ac8`, `965ad78b71bb05df9548f05884196ca89aba9d81`, `e49f3622e6030f378e14ed385f2728ba4500d682`, `6fc21c159c5f8c1fc5270f15fafca7bab882c3ff` | Migración reversible y casos 1–10 y 13, tras dos reaperturas. |
| B2 | `ff4e929beae4d329b31e9142a0f3ebf8a317bbdc`, `8c3393c5c37d2e638b0dde80356975ab1a823eb3`, `ea41c967b3bbcbbe0955f36f9b7255f0542b52ea` | Concurrencia, caso 11 y 13, tras dos correcciones. |
| B3 | `d712cd048e377174b110f92f6498dfc9bb3bd894`, `e3d60cc7b7d3897c654ac40dc624cf37a22670aa` | Ejecución literal `pgx`, caso 12 y 13. |
| B4 | `de6e7df0e377c6e53b4ff1209b994bf3c006c536` | Composición directa y exacta de la matriz completa. |

Los SHA anteriores son los commits de la cadena integrada, no identificadores
equivalentes de ramas de trabajo.

## NO-GO y correcciones reales

### B1

`f050a61` no se aceptó: faltaban cobertura exacta de RLS, manifiestos,
membresías, postcondiciones y ramas negativas. `965ad78` siguió en
**NO-GO, P1=1** porque serializaba `pg_policy.polroles` como OID numérico en
un manifiesto que debía ser simbólico e independiente de OID.

`e49f362` corrigió esa portabilidad, pero una prueba adversarial cambió una
tabla nueva a `REPLICA IDENTITY FULL` y el `up` todavía aceptó la
postcondición. Faltaba acreditar completamente `relreplident`, índices y
atributos físicos de TOAST: **NO-GO, P1=1**.

`6fc21c1` cerró esas postcondiciones, los cinco índices, las dos TOAST y sus
índices, y la línea base de cero etiquetas de seguridad. La mutación
adversarial pasa a rollback íntegro. Resultado final:
**GO, P0=P1=P2=0**.

### B2

`ff4e929` observaba sesiones bloqueadas y fallos, pero no demostraba el lock,
el bloqueador ni el error contractual exactos. `8c3393c` añadió PID y
`pg_blocking_pids` exactos, locks concretos, SQLSTATE y mensaje esperados,
ausencia de interbloqueos y preservación estructural. Aun así quedó un
**P1**: en la carrera del puntero, la huella base se capturaba después de
confirmar el cambio y podía compararse contra sí misma.

`ea41c96` elimina ese TOCTOU: captura antes una huella de filas estables,
excluye las tablas de B y el contador que debe cambiar, y acredita después por
separado puntero, historia, generación y filas base. Resultado final:
**GO, P0=P1=P2=0**.

### B3

`d712cd0` recibió **NO-GO, P0=0, P1=2, P2=0**. La cancelación observaba un
`AccessExclusiveLock` no concedido, pero no probaba su bloqueador mediante
`pg_blocking_pids`; además, el saneador sólo rechazaba que sobreviviera el
opt-in exacto y podía aceptar otro valor residual del GUC.

`e3d60cc` exige conjuntamente lock y bloqueador exactos, y que
`current_setting(..., true)` quede vacío. Cualquier residuo o conexión que no
pueda sanearse provoca su destrucción. Acredita `TxStatus E → I`, `ROLLBACK`,
`RESET`, éxito confirmado y cierre irrecuperable. Resultado final:
**GO, P0=P1=P2=0**.

## Matriz de aceptación 1–13

| Caso | Garantía reproducida | Autoridad |
| ---: | --- | --- |
| 1 | Cadena `000001 → 000004`, atomicidad y rechazo de reentrada o parcial. | B1 |
| 2 | Columnas, restricciones, FKs, índices, propiedad, RLS, ACL, triggers, TOAST y claves base exactos. | B1 |
| 3 | Gramática `vcr_`, límites, controles, superficie y uso literales. | B1 |
| 4 | Versiones, cruces de actor, procedencias, autoridad, estados, tiempos y ventanas. | B1 |
| 5 | Versiones activa/revocada, puntero coherente y generación común. | B1 |
| 6 | Historia append-only y no truncable; puntero no truncable. | B1 |
| 7 | Cero acceso directo o efectivo para todo rol no propietario probado. | B1 |
| 8 | `000002 down` y `000003 down` fallan cerrados mientras B existe. | B1/B2 |
| 9 | Retirada hostil rechazada, opt-in exacto y función externa inocua preservada. | B1 |
| 10 | Ciclo `down/up` vacío preserva filas, generación y catálogo predecesores. | B1 |
| 11 | Carreras de alta, retirada, DDL, inserción y puntero, sin deadlock ni deriva. | B2 |
| 12 | Bytes completos del `down` por `pgx.Conn.Exec`, cancelación y saneamiento. | B3 |
| 13 | Cero contenedores, procesos, conexiones, FIFO o temporales residuales. | B1/B2/B3/B4 |

B4 no duplica cobertura: llama cada runner focal exactamente una vez, en el
orden contractual, y propaga el primer fallo.

## Pruebas reproducidas

Productores y revisores ejecutaron las puertas focales reales:

```text
bash deploy/postgresql/contexto_actor_v1/probar_contexto_actor_v1_base_pg18_4.sh
bash deploy/postgresql/contexto_actor_v1/probar_organizacion_corporativa_v1_pg18_4.sh
bash deploy/postgresql/contexto_actor_v1/probar_vinculo_corporativo_rrhh_v1_estructura_adversarial_pg18_4.sh
bash deploy/postgresql/contexto_actor_v1/probar_vinculo_corporativo_rrhh_v1_concurrencia_preservacion_pg18_4.sh
bash deploy/postgresql/contexto_actor_v1/probar_vinculo_corporativo_rrhh_v1_pgx_pg18_4.sh
```

B-S0 comprobó además el runner principal completo y su variante
`VEC_CONTEXTO_ACTOR_OMITIR_GO=1`. B-S1 exigió tres respuestas consecutivas
`180004|false` por TCP `127.0.0.1` con contraseña y espera acotada, sin
`pg_isready` ni aceptación del servidor temporal de `initdb`.

Dirección reprodujo en `de6e7df`:

```text
timeout 900s bash deploy/postgresql/contexto_actor_v1/probar_integracion.sh
go test ./...
go vet ./...
```

La composición predeterminada, con Go habilitado, terminó con código cero en
**2 min 30 s**. Ejecutó ocho runners y dejó el mensaje global sólo después de
todos ellos. Después, `go test ./...` y `go vet ./...` quedaron verdes.

También quedaron verdes `bash -n`, ShellCheck, `gofmt`, `git diff --check`,
Gitleaks focal, modos y write-sets. No quedaron contenedores, procesos,
conexiones ni temporales residuales. Los artefactos de B quedan bajo 800
líneas: `up` 728, `down` 601, runners B1/B2/B3 637/652/119, prueba Go B3 433
y orquestador principal 15.

Huellas principales del corte: `up`
`f1be3123b1286e7fe8ffae99073179b90b08303c31c8ee8c3783315a6078f368`,
`down` `08ce3d19e175e38cc62a7ed82b052e7aaddf4a6835e8d71e8a4f5092c029c177`,
runner B1 `57c03f4c239f8fabf29c810d04257fc6392c8380868e57839c90e788a895958f`,
B2 `47fdc6c61400acd13ee3b5a432754137f1aee7df071354d1c6359a919c4c899f`
y B3 `b6b0bb7f3fba7465014d973d947de69ea8862d2781974aaf0c317f5b208ae767`.

## Seguridad, privacidad y frontera

Sólo se usan datos sintéticos. Las referencias opacas siguen siendo datos
personales seudonimizados: la minimización no equivale a anonimización ni
sustituye EIPD, ENS, análisis de riesgos o conservación aprobada.

Las tablas tienen RLS activada y forzada, política única del propietario y
ACL cerradas. `PUBLIC`, runtime, selector, LOGIN sintéticos, PDP,
Autorización, Contratación y Bolsa no obtienen acceso efectivo. La política es
de denegación por defecto.

Se detectan dependencias catalogales y consumidores gobernados dentro del
esquema. Un consumidor dinámico externo sin dependencia catalogal no es
descubrible de forma general: necesita registro explícito y puerta de
despliegue futuros. Hasta entonces queda prohibido y producción sigue en
NO-GO. No se presenta una búsqueda global de `pg_proc.prosrc` como garantía.

La comprobación de cero `pg_seclabel` es sólo una línea base de integridad:
**no acredita SELinux ni SE-PostgreSQL**. `allow_system_table_mods=on` se usa
únicamente dentro del contenedor efímero de prueba para inyectar etiquetas
sintéticas adversariales; nunca es una opción de despliegue.

Un superusuario o DBA comprometido que escriba directamente en `pg_catalog`
queda fuera del contrato. Su control pertenece a la frontera ENS y operativa:
separación de funciones, bastionado, privilegios, monitorización, respaldo y
respuesta.

## Estado y continuación

| Ámbito | Estado conservado |
| --- | ---: |
| Contratación temporal | `24/46` (`52 %`) |
| Objetivo O4-05 | `3/5` |
| Bolsa productiva | `1/14` (`7 %`) |
| Producción VEP | **NO-GO** |

El cierre de B no incrementa métricas funcionales. La siguiente minitarea es
**C2.3**, publicación y revocación gobernadas, con la reserva `000005`.
Después siguen C2.4 y C2.5–C2.11. No debe declararse capacidad funcional, E2E
o aptitud productiva antes de esos cierres y de las aprobaciones externas.
