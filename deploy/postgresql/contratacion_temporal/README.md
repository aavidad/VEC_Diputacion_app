# PostgreSQL de contratación temporal

Estado: preparación idempotente implementada; confirmación de efectos cerrada
hasta integrar la autorización VEC durable.

## Alcance del corte

La migración `000001_preparacion_altas` crea:

- identidad inmutable de cada reserva;
- versiones de solo adición;
- puntero actual separado;
- función cerrada `preparar_alta_v1(jsonb)`;
- idempotencia semántica por HMAC;
- referencias estables ante reintentos y concurrencia;
- privilegio runtime exclusivamente `EXECUTE`.

La migración aditiva `000002_rotacion_hmac` conserva la historia v1 y añade:

- alias inmutables de ámbito y huella por reserva canónica;
- función cerrada `preparar_alta_v2(jsonb)`;
- política explícita de generaciones obligatorias;
- restricciones genéricas `/vN`;
- revocación runtime de v1 y concesión exclusiva de v2;
- rechazo de colisiones o convergencia entre dos reservas.
- límites propios de bloqueo, sentencia e inactividad en la función v2.

La clave idempotente original no cruza hacia PostgreSQL. El adaptador solicita
a `SelladorAmbitoIdempotencia` una HMAC ligada a organización, actor y perfil,
y solo persiste esa HMAC. Reutilizar el ámbito con otra huella de petición,
organización, actor o perfil devuelve conflicto y no crea otro expediente.

## Matriz de generaciones

La política instalada por `000002` es:

| Posición | Generación | Estado | Obligación durante la ventana |
| --- | ---: | --- | --- |
| 0 | 2 | activa | Siempre presente y primera. |
| 1 | 1 | retenida | Siempre presente hasta cerrar formalmente la ventana. |

El llavero de aplicación admite una activa y hasta tres retenidas, ordenadas
de mayor a menor y sin repetir generación, referencia ni sello. Los dos
dominios —ámbito idempotente y huella de petición— deben presentar la misma
matriz. Si el conector no puede usar una generación retenida, la operación
falla antes de PostgreSQL. Si un cliente omite una generación exigida, la
función v2 falla cerrada.

El adaptador reutiliza los mismos sellos y referencias candidatas, pero abre
una transacción nueva hasta un máximo de tres intentos exclusivamente ante
`40001` —fallo de serialización— o `40P01` —interbloqueo—. Cualquier otro
SQLSTATE falla sin reintento. La cancelación o el vencimiento del contexto
detienen la secuencia, mientras que un `COMMIT` confirmado se devuelve como
éxito durable sin una comprobación tardía que lo convierta en fallo.

Retirar v1 requiere una migración posterior, inventario de vida máxima de las
claves idempotentes, agotamiento probado de esa vida, copia y restauración
ensayadas, y aprobación de Seguridad. Nunca se cambia la tabla de política a
mano ni se elimina la clave histórica durante la ventana.

La reserva no concede permisos ni confirma un alta. La próxima migración
añadirá agregado, primera actuación, consumo de autorización VEC, auditoría y
outbox en un único `COMMIT`. No se abrirá esa función a la cuenta runtime
antes de disponer del contrato durable de autorización.

## Instalación

1. Ejecutar `roles_up.sql` como DBA.
2. Aprovisionar una cuenta nominativa de migración miembro únicamente de
   `vec_contratacion_temporal_migrador`.
3. Ejecutar, por orden, `000001_preparacion_altas.up.sql` y
   `000002_rotacion_hmac.up.sql` con dicha cuenta.
4. Aprovisionar la cuenta de la aplicación como miembro únicamente de
   `vec_contratacion_temporal_ejecutor`.

Ejemplo sin credenciales incrustadas:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql
```

El repositorio no contiene DSN, usuarios LOGIN, contraseñas ni secretos HMAC.
Estos se suministran mediante el gestor de secretos y la configuración del
entorno.

## Prueba reproducible

La prueba de integración no instala PostgreSQL en el equipo. Usa un contenedor
efímero, sin puertos publicados, sin red y con el directorio de datos en
`tmpfs`. La imagen está fijada por resumen criptográfico:

```bash
./deploy/postgresql/contratacion_temporal/probar_integracion_docker.sh
```

El ensayo valida instalación con cuenta de migración, separación de roles,
privilegio mínimo del ejecutor, alta v1, rotación v2→v1 con el mismo
expediente, recibo, versión, auditoría, evento e instante de confirmación, alta
nativa v2, conflicto semántico, ausencia de una generación histórica, bloqueo
directo limitado por la propia función, y ocho sesiones concurrentes con ambas
generaciones. El ensayo conserva y valida el resumen de `pgbench`: exige
`16/16` transacciones procesadas y cero fallidas. También comprueba
inmutabilidad, rechazo del rollback ordinario con historia y limpieza
destructiva autorizada. Las
identidades LOGIN y la clave usadas existen únicamente dentro del contenedor
efímero y no son credenciales de ningún entorno.

## Reversión protegida

La reversión normal solo funciona sin historia. Destruir historia exige un
procedimiento formal, copia verificada y la confirmación explícita:

```text
vec.confirmar_destruccion_contratacion_temporal=
DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE
```

Esa variable nunca debe formar parte de una receta ordinaria de despliegue.
