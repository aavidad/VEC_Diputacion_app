# Consumidor PostgreSQL de autorización atestada VEC-AD-3

Estado: candidato O2-05 probado en PostgreSQL 18; pendiente de revisión
independiente y de las puertas externas de producción.

## Responsabilidad

Este adaptador consume una capacidad breve VEC-AD-3 y conserva la evidencia
necesaria para demostrar qué decisión autorizó qué efecto. No verifica COSE ni
emite capacidades: esas autoridades pertenecen al verificador y al broker
separados.

Las migraciones son aditivas y no cambian VEC-AD-2:

1. `000001_gobierno_y_registro_v3` crea el gobierno versionado de claves HMAC,
   configuraciones y raíces; revocaciones; checkpoints anti-retroceso;
   atestaciones, consumos y cadena de auditoría durables.
2. `000002_consumidor_capacidad_v3` añade el parser estricto, la
   reconstrucción canónica compatible con `encoding/json`, la comprobación de
   MAC y la función cerrada
   `registrar_y_consumir_decision_v3_atestada`.

La capacidad es un objeto plano de 37 campos. Antes de convertirla a `jsonb`
se rechazan campos ausentes, repetidos o desconocidos. Después se reconstruye
la secuencia exacta de bytes de Go. MAC, hashes, audiencia, operación, efecto,
reloj, clave, gobierno, configuración, raíz y revocaciones se cotejan dentro
de la transacción.

La decisión nominal se registra mediante el CAS V3 existente. El consumidor
no sustituye RBAC, ContextoActor ni el catálogo de motivos.

## Privilegios

`roles_up.sql` crea únicamente grupos `NOLOGIN`:

- `vec_autorizacion_atestada_v3_propietario`;
- `vec_autorizacion_atestada_v3_migrador`;
- `vec_autorizacion_atestada_v3_emisor`;
- `vec_autorizacion_atestada_v3_consumidor`.

La vertical de contratación no recibe lectura ni DML. Su propietario obtiene
solo `USAGE`, la referencia nominal necesaria y `EXECUTE` sobre el consumidor
cerrado. El LOGIN de aplicación tampoco ejecuta ese consumidor directamente:
solo la función final de contratación temporal.

El secreto HMAC se provisiona fuera del repositorio. El emisor no obtiene por
estas migraciones una lectura directa del secreto almacenado. La implantación
debe resolver su uso mediante el broker y el HSM/KMS aprobados.

## Orden de instalación

En la misma base PostgreSQL deben existir ContextoActor V2 registrado,
autorización nominal V3 y contratación temporal hasta `000002`.

```text
roles_up.sql                              DBA
000001_gobierno_y_registro_v3.up.sql      migrador VEC-AD-3
000002_consumidor_capacidad_v3.up.sql     migrador VEC-AD-3
000003_expediente_confirmacion_atestada   migrador contratación
000004_funcion_confirmar_alta_atestada    migrador contratación
```

La transacción cliente debe ser `SERIALIZABLE`, de escritura y UTC, con
`statement_timeout` entre 1 ms y 15 s e
`idle_in_transaction_session_timeout` entre 1 ms y 20 s. La función instala
un `lock_timeout` propio de 2 s.

Los reintentos completos se limitan a `40001` y `40P01`, siempre con una
transacción nueva. Un resultado de `COMMIT` perdido es indeterminado y se
reconcilia; nunca se transforma en cancelación.

## Prueba reproducible

```bash
go test ./internal/vec/adapters/seguridad/confianzaatestacion
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh
```

El segundo mandato usa PostgreSQL 18 por resumen criptográfico, sin red, sin
puertos publicados y con datos en `tmpfs`. Comprueba:

- vector canónico compartido Go↔SQL y rechazo estructural temprano;
- alta completa, repetición exacta y efecto único;
- cruces de decisión/efecto, caducidad y alias incoherentes;
- rollback integral ante un fallo posterior al consumo;
- cuatro sesiones concurrentes sobre la misma capacidad;
- rotación, retención y revocación de HMAC;
- rotación, anti-retroceso y revocación de configuración y raíz;
- mínimo privilegio, denegación predeterminada y funciones históricas cerradas;
- `down` protegido y destrucción explícita con inventario final.

Todos los datos, claves y roles LOGIN del ensayo son sintéticos y viven solo
en el contenedor efímero.

## Reversión

La reversión ordinaria falla si existe historia. La destrucción exige el valor
explícito:

```text
vec.confirmar_destruccion_autorizacion_atestada_v3=
DESTRUIR_AUTORIZACION_ATESTADA_V3_IRREVERSIBLE
```

Debe ejecutarse en orden inverso y solo tras copia restaurable, autorización
formal y retirada previa de las referencias de contratación. El `down` no usa
`CASCADE`.

## Puertas que siguen cerradas

Estas migraciones no autorizan producción. Siguen siendo obligatorios:

- broker separado y verificación COSE interoperable;
- HSM/KMS, doble control y ceremonia de rotación/revocación;
- ancla monotónica externa contra restauraciones atrasadas;
- copias, restauración, retención y destrucción ensayadas;
- TLS/mTLS y credenciales nominativas aprovisionadas por Sistemas;
- revisión independiente y aprobación de Sistemas, Seguridad, DPD y RRHH.

No se han usado cookies ni autoridad procedente del cliente web. Web,
escritorio, CLI y MCP deben llegar por los mismos puertos de aplicación.
