# PostgreSQL de contratación temporal

Estado: O2-05 probado; O3-04 dispone de historia integral y preparación
durables, pero su confirmación atómica continúa en desarrollo. No es una
composición autorizada para producción.

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
- límite propio de bloqueo en la función v2;
- precondiciones ambientales de sentencia e inactividad transaccional.

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

Antes de cada invocación, la misma transacción debe configurar
`statement_timeout` entre 1 ms y 15 s e
`idle_in_transaction_session_timeout` entre 1 ms y 20 s. La función valida
ambos límites al comenzar y deniega valores ausentes, nulos, mal formados o
superiores. No los instala mediante `proconfig`: el límite de sentencia debe
estar armado antes de iniciar el `SELECT`, y el límite de inactividad debe
seguir vigente después de que la función retorne. `lock_timeout=2s` sí es un
límite propio de `preparar_alta_v2`.

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

La reserva no concede permisos ni confirma un alta. Las migraciones aditivas
`000003_expediente_confirmacion_atestada` y
`000004_integridad_agregado_alta` y
`000005_funcion_confirmar_alta_atestada` añaden el agregado durable, versión y
actuación iniciales, consumo VEC-AD-3, auditoría y outbox en un único
`COMMIT`. La identidad `confirmacion_agregado_alta`, su huella portable y las
FK diferibles ligan reserva confirmada, expediente, versión, actuación,
auditoría y outbox. No dependen de `txid`, WAL ni estado de la conexión.

El consumidor informa internamente si acaba de consumir la capacidad o si la
recuperó. Un replay nunca inserta ni completa piezas: solo devuelve el recibo
si el reconciliador acredita todas las filas, refs, canones, huellas,
instantes, eslabones y marcador. Si el efecto es posterior a otros efectos,
valida predecesor, sucesor inmediato y cabeza vigente; el marcador conserva
además la prueba durable del eslabón creado originalmente. Al instalar
`000005`, el runtime pierde `preparar_alta_v2` y recibe solo
`confirmar_alta_atestada_v1`. La autorización genérica tampoco queda expuesta
al LOGIN de la aplicación.

La entrada final usa bytes canónicos de esquemas explícitos y versionados; no
serializa por reflexión el agregado. El resultado contiene solo referencias,
número visible, versión, instante y huellas. Ninguna clave idempotente,
secreto HMAC ni dato personal se devuelve.

Las migraciones aditivas `000006_expediente_integral_versionado` y
`000007_preparacion_operaciones_analisis` preparan O3-04:

- materializan el alta O2 como versión 1 de una historia integral única y de
  solo adición;
- separan esa historia del puntero actual empleado por CAS;
- reservan las operaciones de análisis con HMAC generacional sin guardar la
  clave idempotente original;
- permiten replay exacto y consulta de un recibo ya confirmado;
- mantienen RLS, ACL de mínimo privilegio, filas inmutables y reversión
  destructiva protegida.

Estas migraciones no confirman aún un análisis. La confirmación final debe
consumir fuentes y autorización, publicar la versión 2 y registrar actuación,
auditoría, recibo y outbox en un solo `COMMIT`.

## Instalación

1. Ejecutar `roles_up.sql` como DBA.
2. Aprovisionar una cuenta nominativa de migración miembro únicamente de
   `vec_contratacion_temporal_migrador`.
3. Ejecutar, por orden, `000001_preparacion_altas.up.sql` y
   `000002_rotacion_hmac.up.sql` con dicha cuenta.
4. Instalar el consumidor
   `deploy/postgresql/autorizacion_atestada_v3` y su gobierno.
5. Instalar como propietario de autorización la frontera estrecha
   `migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql`.
6. Ejecutar `000003_expediente_confirmacion_atestada.up.sql`,
   `000004_integridad_agregado_alta.up.sql` y
   `000005_funcion_confirmar_alta_atestada.up.sql`, por ese orden.
7. Ejecutar `000006_expediente_integral_versionado.up.sql` y
   `000007_preparacion_operaciones_analisis.up.sql`, por ese orden.
8. Aprovisionar la cuenta de la aplicación como miembro únicamente de
   `vec_contratacion_temporal_ejecutor`.

Ejemplo sin credenciales incrustadas:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000007_preparacion_operaciones_analisis.up.sql
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
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh
```

El ensayo valida instalación con cuenta de migración, separación de roles,
privilegio mínimo del ejecutor, alta v1, rotación v1→v2 con el mismo
expediente, recibo, versión, auditoría, evento e instante de confirmación, alta
nativa v2, conflicto semántico, ausencia de una generación histórica, bloqueo
directo limitado por la propia función y límites conductuales de sentencia e
inactividad transaccional, además de ocho sesiones concurrentes con ambas
generaciones. El ensayo conserva y valida el resumen de `pgbench`: exige
`16/16` transacciones procesadas y cero fallidas. También comprueba
inmutabilidad, rechazo del rollback ordinario con historia y limpieza
destructiva autorizada. Las
identidades LOGIN y la clave usadas existen únicamente dentro del contenedor
efímero y no son credenciales de ningún entorno.

El segundo runner integra ContextoActor, autorización nominal V3, gobierno de
confianza, VEC-AD-3 y contratación. Comprueba el `COMMIT` único, repetición,
concurrencia, rotaciones y revocaciones, fallo inyectado, ACL, retirada
protegida y reconciliación negativa de cada pieza. Retira o altera de forma
aislada reserva, expediente, versión, actuación, auditoría, outbox, marcador,
refs, huellas y cadenas; cada replay debe fallar sin escribir y el runner
restaura exactamente los casos antes de continuar. La guía detallada está en
`deploy/postgresql/autorizacion_atestada_v3/README.md`.

Ese mismo runner comprueba además la instalación aditiva de `000006` y
`000007`, el materializado de la versión integral inicial, la preparación
reservada/reutilizada/conflictiva de análisis, la imposibilidad de leer o
escribir directamente sus tablas y su retirada protegida.

## Reversión protegida

La reversión normal solo funciona sin historia. Destruir historia exige un
procedimiento formal, copia verificada y la confirmación explícita:

```text
vec.confirmar_destruccion_contratacion_temporal=
DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE
```

Esa variable nunca debe formar parte de una receta ordinaria de despliegue.
