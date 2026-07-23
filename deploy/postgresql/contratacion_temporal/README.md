# PostgreSQL de contratación temporal

Estado: preparación idempotente implementada; confirmación de efectos cerrada
hasta integrar la autorización VEC durable.

## Alcance del primer corte

La migración `000001_preparacion_altas` crea:

- identidad inmutable de cada reserva;
- versiones de solo adición;
- puntero actual separado;
- función cerrada `preparar_alta_v1(jsonb)`;
- idempotencia semántica por HMAC;
- referencias estables ante reintentos y concurrencia;
- privilegio runtime exclusivamente `EXECUTE`.

La clave idempotente original no cruza hacia PostgreSQL. El adaptador solicita
a `SelladorAmbitoIdempotencia` una HMAC ligada a organización, actor y perfil,
y solo persiste esa HMAC. Reutilizar el ámbito con otra huella de petición,
organización, actor o perfil devuelve conflicto y no crea otro expediente.

La reserva no concede permisos ni confirma un alta. La próxima migración
añadirá agregado, primera actuación, consumo de autorización VEC, auditoría y
outbox en un único `COMMIT`. No se abrirá esa función a la cuenta runtime
antes de disponer del contrato durable de autorización.

## Instalación

1. Ejecutar `roles_up.sql` como DBA.
2. Aprovisionar una cuenta nominativa de migración miembro únicamente de
   `vec_contratacion_temporal_migrador`.
3. Ejecutar la migración ascendente con dicha cuenta.
4. Aprovisionar la cuenta de la aplicación como miembro únicamente de
   `vec_contratacion_temporal_ejecutor`.

Ejemplo sin credenciales incrustadas:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql
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
privilegio mínimo del ejecutor, reintento, conflicto semántico sin divulgación
de la identidad persistida, ocho sesiones concurrentes, inmutabilidad, rechazo
del rollback ordinario con historia y limpieza destructiva autorizada. Las
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
