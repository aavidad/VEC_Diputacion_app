# Consumidor PostgreSQL de autorización atestada VEC-AD-3

Estado: base O2-05 y consumidores nominales de consultas RRHH probados en
PostgreSQL 18.4 y revisados de forma independiente. Permanecen cerradas las
puertas externas de producción.

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
3. `000003_consumidor_consulta_cuadro_rrhh_v3` añade una fachada nominal para
   el cuadro interno de Contratación temporal y amplía la audiencia gobernada
   únicamente con ese uso.
4. `000004_consumidor_consulta_detalle_rrhh_v3` añade la fachada nominal del
   detalle de expediente y completa la segunda ampliación de audiencia.
5. `000005_revalidacion_final_consultas_rrhh_v3` relee gobierno, consumo y
   revocaciones y repite la revalidación viva justo antes del cierre
   transaccional.
6. `000006_prueba_consumo_consultas_rrhh_v3` compone esa revalidación sin
   duplicarla y devuelve la prueba completa de cuadro o detalle tras releer
   bajo locks las tablas propias de atestación, consumo y auditoría. Además,
   ordena las tres clases de revocación con el checkpoint de gobierno antes
   de que una prueba final pueda considerarse confirmable.

La capacidad es un objeto plano de 37 campos. Antes de convertirla a `jsonb`
se rechazan campos ausentes, repetidos o desconocidos. Después se reconstruye
la secuencia exacta de bytes de Go. MAC, hashes, audiencia, operación, efecto,
reloj, clave, gobierno, configuración, raíz y revocaciones se cotejan dentro
de la transacción.

La decisión nominal se registra mediante el CAS V3 existente. El consumidor
no sustituye RBAC, ContextoActor ni el catálogo de motivos. Su resultado
interno incluye `consumo_nuevo`: `true` solo tras insertar consumo y auditoría;
`false` solo cuando recupera exactamente esos bytes. La función exterior usa
esta señal para impedir que una capacidad ya consumida complete piezas tarde.
Las fachadas de consulta devuelven además la huella real del eslabón de
auditoría. Un replay sirve solo para conciliación: la futura función exterior
de Contratación temporal deberá rechazar `consumo_nuevo=false` antes de leer o
entregar datos.

Las dos funciones nominales de `000006` no consumen ni escriben otra vez.
Exigen un consumo previo, llaman primero a la revalidación final de `000005` y
releen en la misma transacción las tres filas autoritativas con locks. Devuelven,
en orden, `decision_ref`, `efecto_ref`, `huella_efecto_sha256`,
`consumo_huella_sha256`, `auditoria_ref`, `auditoria_huella_sha256`,
`consumida_en` y `revalidada_en`. Las diez piezas originales siguen siendo la
entrada obligatoria y se cotejan de nuevo; no se acepta una huella o referencia
probatoria aportada por Contratación temporal.

Las inserciones de revocación de clave, configuración o raíz avanzan
incondicionalmente el checkpoint dentro de su propia transacción. Si la prueba
ha bloqueado primero el checkpoint, la revocación espera hasta su `COMMIT`. Si
la revocación crea primero la nueva versión MVCC, la prueba que partió de un
snapshot anterior falla con `40001` tras confirmarse aquella. El rollback de
una revocación no altera el checkpoint ni provoca una denegación espuria.

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
000003_consumidor_consulta_cuadro_rrhh_v3 migrador VEC-AD-3
000004_consumidor_consulta_detalle_rrhh_v3 migrador VEC-AD-3
000005_revalidacion_final_consultas_rrhh_v3 migrador VEC-AD-3
000006_prueba_consumo_consultas_rrhh_v3   migrador VEC-AD-3
000003_expediente_confirmacion_atestada   migrador contratación
000004_integridad_agregado_alta           migrador contratación
000005_funcion_confirmar_alta_atestada    migrador contratación
```

La transacción cliente debe ser `SERIALIZABLE`, de escritura y UTC, con
`statement_timeout` entre 1 ms y 15 s e
`idle_in_transaction_session_timeout` entre 1 ms y 20 s. La función instala
un `lock_timeout` propio de 2 s para consumir y de 1 s para revalidar y
construir la prueba.

Los reintentos completos se limitan a `40001` y `40P01`, siempre con una
transacción nueva. Un resultado de `COMMIT` perdido es indeterminado y se
reconcilia; nunca se transforma en cancelación.

## Prueba reproducible

```bash
go test ./internal/vec/adapters/seguridad/confianzaatestacion
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh
./deploy/postgresql/autorizacion_atestada_v3/probar_consultas_rrhh_v3_pg18_4.sh
./deploy/postgresql/autorizacion_atestada_v3/probar_prueba_consumo_consultas_rrhh_v3_pg18_4.sh
```

El último runner carga como componente obligatorio
`pruebas_sql/probar_serializacion_revocaciones_rrhh_v3.sh`. Su ausencia o
error impide ejecutar el ensayo; no constituye una prueba independiente ni
una ruta de producción.

El segundo mandato usa PostgreSQL 18 por resumen criptográfico, sin red, sin
puertos publicados y con datos en `tmpfs`. Comprueba:

- vector canónico compartido Go↔SQL y rechazo estructural temprano;
- alta completa, repetición exacta y efecto único;
- ocho piezas ligadas: consumo, reserva confirmada, expediente, versión,
  actuación, auditoría, outbox y marcador portable;
- cruces de decisión/efecto, caducidad y alias incoherentes;
- rollback integral ante un fallo posterior al consumo;
- cuatro sesiones concurrentes sobre la misma capacidad;
- replay expirado válido solo con agregado íntegro y rechazo sin reparación
  ante ausencia o deriva de cada pieza, refs, canones, huellas y cadenas;
- rotación, retención y revocación de HMAC;
- rotación, anti-retroceso y revocación de configuración y raíz;
- mínimo privilegio, denegación predeterminada y funciones históricas cerradas;
- `down` protegido y destrucción explícita con inventario final.

Todos los datos, claves y roles LOGIN del ensayo son sintéticos y viven solo
en el contenedor efímero.

El tercer mandato fija PostgreSQL 18.4 por resumen, no publica puertos y
comprueba de forma específica:

- cuatro fotografías exactas de la restricción de audiencias y reversión
  protegida frente a adulteración;
- regresión funcional del alta existente;
- aislamiento A/B de cuadro y detalle, ACL nominativa y rol mínimo;
- rollback atómico, recibo de ocho campos y replay sin crecimiento;
- ligadura independiente de las diez piezas del conjunto probatorio;
- DER-SPKI Ed25519 canónico y rechazo de X25519, RSA, nulos y DER no canónico;
- colisión múltiple determinista, caducidad durante RBAC y ambos órdenes
  serializables entre consumo y revocación;
- bloqueo absoluto del `down` ante claves, punteros o historia.

El cuarto mandato fija también PostgreSQL 18.4 por resumen, sin red ni puertos,
y comprueba:

- firmas nominales y retorno exacto de las ocho piezas probatorias;
- ACL exclusiva para el propietario de Contratación temporal, sin ampliar el
  LOGIN runtime ni los grupos VEC;
- ausencia de lecturas de tablas de Contratación temporal desde la autoridad
  VEC-AD-3;
- consumo ausente, cuadro, detalle, replay de conciliación y cruces nominales;
- nueva ligadura de cada una de las diez piezas originales;
- relectura bloqueada de atestación, consumo y auditoría, incluido el cruce de
  una auditoría válida perteneciente a otra decisión y efecto;
- revocación previa de clave y matriz causal de las tres clases: rollback de
  raíz, prueba anterior a revocación de clave y revocación de configuración
  anterior a la prueba;
- rechazo causal exacto con `40001`, sin confundir `lock_timeout` o deadlock
  con una revocación observada;
- reentrada, `up → down → up`, catálogo y `down RESTRICT` ante una dependencia
  futura simulada de Contratación temporal.

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

`000006` no borra historia: retira sus tres triggers de serialización y sus
cuatro funciones con `RESTRICT`. Una función futura que dependa de cualquiera
de las dos fachadas impide la reversión. Después se retira `000005`; las
migraciones de audiencias conservan además sus barreras históricas propias.

## Puertas que siguen cerradas

Estas migraciones no autorizan producción. Siguen siendo obligatorios:

- broker separado y verificación COSE interoperable;
- HSM/KMS, doble control y ceremonia de rotación/revocación;
- ancla monotónica externa contra restauraciones atrasadas;
- copias, restauración, retención y destrucción ensayadas;
- TLS/mTLS y credenciales nominativas aprovisionadas por Sistemas;
- aprobación de Sistemas, Seguridad, DPD y RRHH.

No se han usado cookies ni autoridad procedente del cliente web. Web,
escritorio, CLI y MCP deben llegar por los mismos puertos de aplicación.
