# O4-04D — Informe de cierre técnico del consumo C1

Fecha: 2026-07-26

Rama: `correccion/ct-o4-04-20260726`

Estado: **GO técnico provisional; pendiente de revisión independiente**

Commit: no creado por instrucción de coordinación

## Alcance congelado

El entregable implementa exclusivamente O4-04D:

- persistencia durable del lote C1 y de sus evidencias;
- canónicos y huellas verificables;
- prevalidación, bloqueo, CAS, persistencia e idempotencia;
- límites de 1 a 512 evidencias y límite lógico de 64 MiB por lote;
- unicidad de posición, petición y respuesta;
- RLS forzada, ACL mínima e inmutabilidad;
- canon de recurso resuelto en servidor que liga rama, organización,
  expediente, versión, reserva, orden y lote con la huella de la decisión;
- wrapper interno VEC de concesión/denegación con prueba criptográfica;
- proyección `p_resultado_vec` estrictamente privada y sin `EXECUTE` runtime;
- migraciones `up` y `down` con barrera y dependencias explícitas.

No forma parte de este entregable el ejecutor externo E. El único camino
productivo futuro permitido es O4-04E: wrapper VEC y persistencia en una sola
transacción serializable, sin aceptar un resultado VEC aportado externamente.

## Write-set

1. `deploy/postgresql/contratacion_temporal/migraciones/`
   `000023_consumo_c1_esquema_canones_o4_04d.{up,down}.sql`
2. `deploy/postgresql/contratacion_temporal/migraciones/`
   `000024_consumo_c1_operaciones_o4_04d.{up,down}.sql`
3. `deploy/postgresql/contratacion_temporal/migraciones_autorizacion/`
   `000004_wrapper_vec_cobertura_o4_04d.{up,down}.sql`
4. `internal/modules/contrataciontemporal/adapters/postgres/`
   `consumo_c1_cobertura_o4_04d_fixture_test.go`
5. `internal/modules/contrataciontemporal/adapters/postgres/`
   `consumo_c1_cobertura_o4_04d_integracion_test.go`
6. `internal/modules/contrataciontemporal/adapters/postgres/`
   `consumo_c1_cobertura_o4_04d_test.go`
7. `internal/modules/contrataciontemporal/adapters/postgres/`
   `consumo_c1_cobertura_o4_04d_wrapper_integracion_test.go`

Este informe es el undécimo fichero del write-set.

## Validación ejecutada

PostgreSQL real:

```text
Imagen: postgres:18.4-bookworm
Resultado: PASS
```

La suite real cubre ContextoActor y Autorización reales, concesión versión 10,
denegación versión 100, canónicos, ACL, dependencias, ciclo `down/up`, lotes de
1 y 512 evidencias, rechazo de 0 y 513, rechazo de lote lógico mayor de 64 MiB,
rollback, replay exacto, replay mutado con `23505`, posiciones desordenadas,
peticiones y respuestas duplicadas, concurrencia y reapertura de la conexión.
Con una decisión válida muta de uno en uno organización, expediente, versión,
reserva, orden, lote y huella de contexto; las siete variantes se deniegan.
También comprueba por catálogo que ninguna función D tiene `EXECUTE` para los
roles runtime y que una sesión LOGIN no propietaria recibe `42501` al intentar
fabricar y persistir un `p_resultado_vec`.

Comandos con resultado satisfactorio:

```bash
VEC_O404D_ADMIN_DSN='<DSN efímera fuera de Git>' \
VEC_O404D_PG18_AISLADO=1 \
go test ./internal/modules/contrataciontemporal/adapters/postgres \
  -run '^TestConsumoC1CoberturaO404DPostgreSQL18Real$' -count=1 -v

go test ./internal/modules/contrataciontemporal/... -count=1
VEC_O404D_ADMIN_DSN='<DSN efímera fuera de Git>' \
VEC_O404D_PG18_AISLADO=1 \
go test -race ./internal/modules/contrataciontemporal/adapters/postgres -count=1
go vet ./internal/modules/contrataciontemporal/...
git diff --no-index --check /dev/null '<cada fichero del write-set>'
```

Los contenedores efímeros utilizados para la validación fueron eliminados.

## SHA-256 del código congelado

```text
da8b9734d99368db21807d83f02d418f69c8e8356a57fb3ba41485bf18388646  000023...down.sql
43405380ef1f92d58225b94f491ffb7ceb50e4897d960939382392dd6b2fd063  000023...up.sql
2452cd39603f9f2db79e1b518fbd3d450876e215547568e399aff01f0c4374fe  000024...down.sql
d91ce7c315b72324132e4a85b580ae156d262776e5c21accbd4294b3a6036c93  000024...up.sql
108bd99510176cd1bbb5fee7aa3b1abe74e46c5044f6c01b4069e3e5a4bc088d  wrapper...down.sql
e5b0fe58a47377e50a1fe8f3f4431c1f35450f4ce3c6419b87afbf75f1dbb132  wrapper...up.sql
c4ecd54741da09c00cb470102e91808ef4df3d7d41f4d818be3e1632475cb85c  fixture_test.go
961f6b22b3b17cb8121b499fef3fbe6a8024a065274b3bb325b0eec703ce8a45  integracion_test.go
f2bcfb4cadfc21b4081dd1e2d60d051baf5ac1a479b3d07489951ab1289c0a9f  test.go
3b24fbeec839d304622d1f0c30cc05bd87cd718452da7fb511c527e6993dbff5  wrapper_integracion_test.go
```

Los nombres abreviados de este bloque se corresponden, en el mismo orden, con
el inventario del write-set. El SHA-256 de este propio informe se registra
externamente para evitar una referencia circular.

## Riesgo residual

No se conoce un bloqueo técnico en O4-04D. Las funciones D siguen existiendo
como prerrequisitos privados del propietario técnico, sin concesiones runtime;
no constituyen una API productiva ni autorizan a aceptar `p_resultado_vec`
externo. El estado sigue siendo provisional hasta que otra persona o agente
complete la revisión independiente del write-set congelado y O4-04E materialice
la composición atómica futura.
