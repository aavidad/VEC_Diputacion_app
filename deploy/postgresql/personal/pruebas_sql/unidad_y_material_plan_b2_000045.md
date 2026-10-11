# Personal45: unidad y material del plan B2

Personal45 exige que una alta nueva use una unidad publicada de su organismo en Organización de Personal. Elige la última revisión del nodo antes de comprobar retirada, pertenencia y vigencia en la fecha del alta. Si no hay una unidad admisible o hay ambigüedad, deniega.

La autorización V3 se consume antes de recuperar el recibo existente. Después del replay se toma FOR SHARE sobre la barrera de publicación existente de Personal31, antes de leer los nodos. Un publicador modifica esa misma fila por el trigger BEFORE INSERT. La migración comprueba la dependencia, propietario, RLS, fila de control, trigger activo y definición del trigger. La guarda nueva se ejecuta después del bloqueo; retirar una unidad no invalida por sí solo un alta histórica ya registrada. Esta posición se conserva en el ensayo del cuerpo instalado. La recuperación nominal completa sigue pendiente de material V3 válido. El orden conservado es consumo V3 → replay → barrera → lectura de unidad → efecto.

En los actos del plan se cotejan organismo, unidad y persona o empleado antes de llamar al alta o hecho original. La ocupación debe conservar empleado, relación y revisión del recibo previo. La comprobación posterior ya revertía las divergencias; este parche no acredita una escritura cruzada persistida.

## Dependencia y cambio

Base: PR #1004, commit `bf2abb8ce76dbee567bed56c5c27dfb32ac09143`. Personal41–44 deben estar instaladas antes de Personal45. Una migración nueva, sin DOWN; no cambia tablas, ACL, firmas, canon de recursos ni recibos existentes. `unidad_ref` es una clave textual y `nodo_ref` un UUID: no son intercambiables.

La carga de RPT 2026 crea plazas y puestos, pero no nodos. El recorrido nominal requiere una unidad publicada con la misma clave en Organización de Personal; no se deduce de la RPT.

| Función | Preimagen SHA256 | Postimagen SHA256 |
| --- | --- | --- |
| `registrar_acto_empleado_b2_interna` | `0b0efafebb10439c61a4b5401238ae4efec028923be3f97e313265514bf2c1a6` | `9ba1f3c54a9140a2dfe6dca5ee6a55637f6a0deafb4e8dbbff88a4e82be2d258` |
| `registrar_acto_plan_incorporacion_ct_v1` | `7ce49537c80a3b9821b5a9c4d8322d51aad722e7966250b8e81bca4d94949455` | `ed3701c97406915c0cef13161a861fe44d2dd133aa981b4354ffc4ac6881a485` |

Huellas calculadas sobre `pg_get_functiondef` de PostgreSQL 18.4. Los cuerpos literales proceden de Personal42/44 instaladas en el clon, no de las migraciones antiguas 19/23.

## Ensayo del 11 de octubre de 2026

Clon propio PostgreSQL 18.4, sin red externa, contenedor `--rm --restart=no --memory=2g --cpus=2 --pids-limit=128`; datos en disco, fuera de `/dev/shm`. Origen físico HZ19 conservado, SHA256 `bb1d63c2cddbdd0dda0dbba8cad71136402da4e67e71e4ec34296c9fb4c0bf06`. Orden: HZ20 causal, Personal40, datos RPT 2026 y Personal41–44, después Personal45. Solo se aplicaron sobre el clon nuevo.

Resultado de la candidata R2: `ENSAYO-OK`; ambas postimágenes, propietario, ACL y configuración coinciden. El ensayo final R2 partió de una copia física nueva anterior a Personal45. Los clones anteriores conservan sus versiones; no se reaplicó una versión modificada sobre una instalada.

Prueba propia, como DBA exclusivamente en el clon desechable:

```sh
docker exec -i vec-codexh-p45-r2-pg18 psql -X -q \
  -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < deploy/postgresql/personal/pruebas_sql/unidad_y_material_plan_b2_000045.sql
```

Resultado: `P45_FOCAL_OK`.

- Diez casos del predicado de unidad extraído del cuerpo instalado: válida, ajena, inexistente, retirada, futura, vencida, no conocida, ambigua, cambio de organismo y retirada posterior. No se ejecuta una copia independiente de ese predicado.
- Cinco denegaciones en la fachada real del plan, por organismo, unidad, persona, empleado o ausencia de relación. El material compatible llega al original y se rechaza allí.
- Tres denegaciones de ocupación por sustitución de empleado, relación o revisión. Usan una relación y un recibo sintéticos insertados por DBA en el fixture; no acreditan un alta autorizada.
- Seleccionar, preparar, ejecutar y confirmar atraviesan las guardas de Personal y llegan al núcleo V3 real, que rechaza el material deliberadamente incompleto y sin firma con `22023: entrada VEC-AD-3 inválida`. Son pruebas de frontera, no operaciones positivas.
- El núcleo y sus consumidores no se sustituyen. Las denegaciones no añaden efectos. ROLLBACK deja cero nodos, planes y roles del fixture, y conserva la historia previa.

La prueba de concurrencia usa dos sesiones PostgreSQL reales y ejecuta el segmento de bloqueo extraído de la función instalada, cuya postimagen comprueba previamente. No sustituye el consumidor V3 ni acredita un alta nominal:

```sh
python3 deploy/postgresql/personal/pruebas_sql/unidad_y_material_plan_b2_000045_concurrencia.py \
  vec-codexh-p45-r2-pg18
```

Resultado: `P45_CONCURRENCIA_OK`.

- El lector fija una instantánea SERIALIZABLE; el publicador confirma un INSERT que avanza el trigger real; el bloqueo posterior del lector termina con SQLSTATE `40001`.
- El lector obtiene FOR SHARE primero; `pg_blocking_pids` acredita que el INSERT del publicador está esperando a ese lector. Después del COMMIT del lector, el publicador confirma.

Estas dos publicaciones sintéticas quedan en el clon desechable como historia. El runner exige un clon que todavía no conserve su ensayo; no las borra ni permite repetirlo sobre ellas. La prueba focal anterior sí termina en ROLLBACK sin fixtures.

Semgrep local, métricas y comprobación de versión desactivadas: dos reglas genéricas de SQL existentes en `unidad_bootstrap_admin_000031_semgrep_local.yml`, dos archivos analizados (migración y runner), cero hallazgos y cero errores. Su alcance es detectar concesiones a PUBLIC; no sustituye la revisión SQL independiente. Gosec y puerta Go no aplican: no hay cambios Go propios. `git diff --check` correcto.

No se acreditan todavía el alta positiva con V3 válido, la cadena nominal completa ni su recuperación tras reinicio. Las dos revisiones independientes del hash candidato, la CI, la integración y el despliegue quedan pendientes de Dirección. No se ha actuado sobre cidonia ni sobre servicios de otros encargos.
