# C5: recuperación propia y precondiciones durables

Estado: borrador bloqueado para revisión. No instala objetos ni activa C5.
Fuente exacta: `0d54c4d64c3d26de9ab331a5f305dd21d76f7ef7`, rama
`origin/trabajo/codexe-cronos-c5-retoma-20261001`.
El [contrato de CRN11](../README.md) conserva sus bytes y sus dos revisiones
anteriores. Su SHA-256 es
`9ddfd49e8ea7e7020c3e875abc9fd88f35715a3d789938666623195c50b1b61e`.
Las revisiones de ese contrato no acreditan este borrador nuevo.

CRN11 `000011` está reservado para recuperar el recibo inicial propio.
AD138 está reservado para consumidores nominales separados. Su orden es
posterior al paso 3 de M, sobre la postimagen publicada y ensayada que D
identifique. No se añade a H6 ni se presume esa postimagen instalada.
Decisión, resolución y aplicación necesitan un corte Cronos posterior cuyo
número sigue pendiente; este borrador no amplía la reserva CRN11.

## Dependencias que mantienen cerrada la operación

| Dependencia | Entrega necesaria |
| --- | --- |
| AD138, autoridad de D | Audiencia, perfil de consumo, fachada nominal, catálogo/motivo, emisor, reconocimiento, dimensiones exactas, validación de payload/sobre y permisos técnicos. No basta el número reservado. |
| Base causal | Hash de la postimagen tras el paso 3 de M y preimagen real de catálogo, ACL, propietarios, RLS, triggers y funciones en el clon autorizado. Las huellas de archivos no son esa preimagen. |
| Enclave sintético F1 | Identificación expresa por D, con frontera y cuentas admitidas. Este borrador no elige un entorno ni abre un pool. |
| Atomicidad | Consumo V3, evidencia central, lectura y evidencia Cronos en la misma transacción. Si hay bases separadas, hace falta el puente durable acordado; dos commits no bastan. |
| Adaptador y composición | Proveedor nominal de lectura propia, serializer canónico y llamada durable después de revisión. El proveedor genérico por paso no prueba ese contrato. |

Mientras falte cualquiera de ellas, `RecuperarRecibo` conserva
`ErrDependenciaNoDisponible`. No se crea una fachada ficticia, una concesión
runtime ni un nombre de función de consumo para completar el SQL.

## Piezas existentes contrastadas

| Fuente relativa a la raíz del repositorio | Hecho relevante |
| --- | --- |
| `internal/modules/cronos/ports/correccion.go` | `ClaveRecuperacionCorreccion` exige solicitud, clave y paso; `ReciboCorreccion` define los siete campos de respuesta. |
| `internal/modules/cronos/domain/correccion.go` | Valida el comando, omite el instante de recepción de la huella semántica y fija las cinco transiciones posteriores admitidas. |
| `internal/modules/cronos/application/correccion.go` | `ServicioCorrecciones` reconoce recuperación histórica por paso y revalida el contexto de actor. La audiencia general no publica por sí sola lectura propia. |
| `internal/modules/cronos/adapters/postgres/empleado_solicitudes.go` | Sólo solicita el olvido por `solicitar_correccion_propia_v1`; `RegistrarActuacion` y `RecuperarRecibo` fallan cerrados. |
| `deploy/postgresql/cronos_v1/migraciones/000008_movimientos_y_permisos_empleado.up.sql` | Guarda solicitud, actuación inicial y evento juntos. La repetición de escritura compara la huella del material de solicitud, consume V3 nuevo y conserva el recibo v1. |
| `deploy/postgresql/cronos_v1/migraciones/000004_programacion_y_libro_saldo.up.sql` | `programacion_jornada` conserva versiones, zona, política y fuente. `consultar_libro_saldo_interno_v1` deriva el libro desde programación y marcajes originales; no hay proyección de correcciones aplicadas. |
| `deploy/postgresql/cronos_v1/migraciones/000001_esquema_marcajes.up.sql` | Marcajes e historia son de solo adición. El trigger `rechazar_mutacion_historia` conserva esa protección. |

Las fuentes funcionales son C5 de
`docs/estudio_requisitos/ficha_cronos_2026-09-23.md`, identidad/autorización,
auditoría y enclave de `docs/estudio_requisitos/seguridad_y_despliegue_cronos.md`,
y las secciones 1 a 8 de
`docs/portal_vec/seguridad_persistencia_postgresql.md`. Se aplican E03 a E07,
E10 y E12: autoridad común, denegación por defecto, privacidad, atomicidad,
historia y revisión exacta. No se interpreta ni se cambia una regla legal.

## Identidad, material y recibo inicial

El servidor resuelve actor y perfil desde ContextoActor y exactamente un
empleado vigente desde Personal. La titularidad no concede autorización.
No obtiene el empleado leyendo primero la solicitud ni lo toma del navegador.
Admite sólo `paso=solicitud`; deriva y coteja
`solicitud_ref = 'correccion:cronos:' + clave_operacion` antes de pedir V3.

El material propuesto conserva los siete campos del contrato anterior:
`actor_ref`, `perfil_ref`, `empleado_ref`, `clave_operacion`, `solicitud_ref`,
`paso` y `version_recibo`. Todos son cadenas; paso y versión son exactamente
`solicitud` y `1`. El serializer del servidor produce claves ordenadas, sin
espacios, en UTF-8, hasta 4096 bytes. SQL detecta duplicados antes de convertir
a `jsonb`, comprueba las siete claves y calcula SHA-256 de los bytes recibidos.
No sustituye esos bytes por una reserialización de `jsonb`.

La ecuación de recuperación propuesta es:

```text
K = (empleado acreditado, solicitud_ref, clave original, solicitud, 1)
H = SHA256(material UTF-8 exacto)
R(K) = (solicitud_ref, actuacion_ref, recibo_ref,
        pendiente_responsable, 1, registrada_en original, true)

Recuperar(K, concesión nueva ligada a K y H) = R(K) + un acceso nuevo
Recuperar(K, la misma concesión ya consumida) = rechazo
Delta(solicitudes, actuaciones, recibos de negocio, outbox, marcajes) = 0
```

Dos accesos con decisiones nuevas devuelven el mismo recibo y añaden dos
evidencias. No son replays de una concesión. La versión actual del agregado
puede ser posterior: el recibo recuperado sigue siendo v1 y no informa del
estado actual. No exige repetir la declaración del olvido, su hora o su motivo.

El resultado usa exactamente los siete campos de `ReciboCorreccion`:
`solicitud_ref`, `actuacion_ref`, `recibo_ref`, `estado`, `version`,
`instante_utc` y `replay`. El instante procede de `a.registrada_en`, con
precisión de microsegundo; el adaptador lo presenta en UTC y valida las
referencias UUID v4. No usa la hora nueva del acceso como fecha del recibo.

## Unidad de confirmación de la recuperación

1. Validar entradas y límites de los once argumentos antes de interpretar
   sobres. Rechazar contexto previo de empleado o resolutor en la transacción.
2. Acreditar el vínculo propio y verificar/consumir V3 nominal nuevo. Ligar
   actor, perfil, empleado, recurso, paso, versión y huella exactos. La firma,
   raíz, emisor, audiencia y vigencia se verifican por la autoridad existente.
3. Exigir consumo nuevo y referencias de decisión, auditoría y consumo
   canónicas. Revalidar sesión, asignación, catálogo y vigencia con reloj vivo
   tras cualquier espera. Fijar sólo el contexto ya acreditado, local al commit.
4. Obtener una sola fila con el join del borrador SQL. Cero filas produce
   rechazo uniforme; más de una indica dependencia inconsistente. No leer
   primero por referencia libre para confirmar que la solicitud existe.
5. Añadir `correccion_recibo_acceso` con la fila leída, H y las referencias
   provenientes exclusivamente de AD138. Las FK individuales no aseguran el
   vínculo conjunto: el INSERT sólo admite la fila del join comprobado.
6. Revalidar antes del retorno y confirmar consumo, auditoría central y acceso
   Cronos juntos. El adaptador entrega el recibo sólo después de commit.

La lectura de v1 inmutable no necesita bloquear ni actualizar la solicitud.
El consumo central sí debe proteger su uso único y las autoridades vigentes
según AD138. No se usa `ON CONFLICT DO NOTHING`. Un fallo de consumo,
auditoría, INSERT, cardinalidad o commit revierte la unidad completa.
La denegación se audita por la frontera existente: un rollback no conserva
por sí solo esa evidencia. El recorrido posterior debe comprobar ambos caminos.

La futura función propuesta en el contrato anterior será SECURITY DEFINER,
con propietario NOLOGIN/NOBYPASSRLS, `search_path=pg_catalog,pg_temp`,
`row_security=on`, `timezone=UTC`, `lock_timeout=2s` y `statement_timeout=5s`.
Todos los objetos se cualifican. FORCE RLS, políticas propias, trigger
inmutable y revocación de tabla, tipo de fila y función se comprobarán en la
postimagen. La candidata inicial no concede ejecución runtime; cualquier
activación posterior necesita consumidor visible y entrega expresa.

## Precondiciones para decisión, resolución y aplicación

| Paso nuevo | Versión y estado anteriores | Resultado y recibo nuevos |
| --- | --- | --- |
| `decision_responsable` | v1, `pendiente_responsable` | `favorable` produce v2 `pendiente_rrhh`; `desfavorable`, v2 `denegada_responsable`. |
| `resolucion_rrhh` | v2, `pendiente_rrhh` | `favorable` produce v3 `pendiente_aplicacion`; `desfavorable`, v3 `denegada_rrhh`. |
| `aplicacion` | v3, `pendiente_aplicacion` | Resultado vacío; produce v4 `aplicada` y una compensación. |

Las denegaciones son terminales en el dominio existente. No se omite la
decisión del responsable ni se aplica directamente desde la solicitud.
Las constantes de acción ya existen en aplicación; audiencia, recurso,
finalidad, campos, obligaciones y fachadas nominales siguen por acordar con D.
No se recicla el permiso propio de solicitud ni el de recuperación.

Para escribir hará falta, además:

- Resolver sujeto gobernado por autoridad vigente y mínima, sin inferirlo del
  empleado del decisor ni de un ID libre del navegador. Acordar cómo se obtiene
  esa relación antes del acceso funcional a las tablas protegidas.
- Revalidar relación jerárquica/delegación para el responsable y competencia,
  unidad y periodo para RRHH. Publicar la separación de funciones y su
  comprobación exacta, sin inventar quién puede aprobar a quién.
- Un registro nuevo de solo adición por operación, con clave, paso, solicitud,
  actor/perfil, sujeto acreditado, versión esperada, resultado, huella y vínculo
  al recibo/consumo. `correccion_actuacion` no guarda clave, resultado, actor,
  perfil ni huella de los pasos v2 a v4 y no basta para su idempotencia.
- Una restricción única y un orden de bloqueo acordados para clave y agregado.
  Dentro de la transacción se comprueba primero un replay autorizado exacto;
  sólo un comando nuevo exige que la última versión sea la esperada. Dos claves
  distintas para el mismo estado no pueden añadir la misma siguiente versión.

La ecuación propuesta para esas escrituras posteriores es:

```text
I = (paso, clave nueva de ese paso), con unicidad final pendiente de contrato
S = (solicitud, actor, perfil, sujeto acreditado, paso, clave,
     versión esperada, resultado); el instante de recepción no pertenece a S
H_S = SHA256(codificación exacta acordada de S)

I existente y S idéntico + autorización actual => mismo recibo histórico
I existente y S distinto => conflicto, sin otra actuación
I nuevo + estado/versión válidos + autorización actual
  => consumo + registro de operación + actuación v+1 + recibo + auditoría + outbox
```

La huella Go `ActuacionCorreccion.HuellaSemantica` usa `json.Marshal` de un
struct con nombres de campo Go; no es la misma codificación que el material
JSON de recuperación ni que `material_sha256` de 000008. El futuro adaptador
debe acordar y probar los bytes exactos con SQL y V3, incluyendo el sujeto
acreditado en el vínculo de autorización. No puede comparar esas huellas
como si fueran intercambiables.

Aplicar requiere una compensación nueva ligada a solicitud, resolución,
operación y recibo, más fuente y versión de la política usada. Original y
corrección se conservan. Falta definir la conversión de fecha/hora civil a
instante, zona, ambigüedad DST y el tratamiento de hueco declarado frente a
original existente. La programación vigente no puede elegirse implícitamente
para reescribir un hecho histórico. Falta también la proyección derivada que
consuma una sola vez la compensación y reproduzca saldo e incidencias con sus
fuentes. No se inserta un marcaje original para simular esa aplicación.

Sin esos contratos, la aplicación sigue cerrada. Registrar `aplicada` con una
intención de proyección pendiente no acredita efecto completo; si se admite
un puente durable deberá definir fases, reconciliación y cuándo se puede
emitir ese recibo. No se escribe en tablas de Personal ni de otro módulo.

## Comprobaciones que faltan antes de activar

| Caso sintético | Resultado exigido |
| --- | --- |
| Material duplicado, nulo, extra, tipo incorrecto, tamaño excesivo o paso distinto | Rechazo antes de consumir; sin recibo. |
| Otra persona, solicitud inexistente o clave cruzada | Respuesta uniforme, sin confirmar existencia ni datos. |
| Cambio de un campo o bytes con el mismo sobre; concesión histórica de solicitud | Rechazo; no reutilizar consumo. |
| Dos lecturas con decisiones nuevas, también tras actuaciones posteriores | Recibo inicial idéntico; una evidencia por lectura; cero negocio nuevo. |
| Sobres repetidos, revocación durante espera o auditoría caída | Sin éxito ni efecto parcial; denegación auditable por frontera. |
| Reinicio de aplicación y PostgreSQL | Mismo recibo y fecha; nueva autorización para recuperar. |
| ACL, tipo de fila, roles y FORCE RLS | Sin ejecución ni lectura directa runtime; propietario sin salto de RLS. |
| Escrituras posteriores concurrentes y replay exacto tras avance | Un único paso nuevo o conflicto; replay conserva recibo histórico. |
| Aplicación, una vez acordada | Una compensación efectiva, originales intactos y proyección reproducible. |

## Comprobación de este corte

Se ha contrastado estáticamente el contrato con Go y las fuentes SQL existentes.
El archivo `recuperacion_propia_v1.sql.borrador` contiene sólo comentarios,
incluidos los fragmentos de validación, consulta y tabla propuesta; no contiene
una función operativa ni una llamada ficticia a AD138. No es `.up.sql`, no
entra en listas SQL y no se ejecuta para validarlo.

No se ha ejecutado SQL, contenedor, servicio, prueba Go ni recorrido. `gopls`
no está en PATH; no hay cambio Go. El ensayo de `revisar-sql-vec`/`ensayar-sql`
queda para la futura candidata ejecutable y Dirección. Las dos revisiones
SQL/identidad sobre su hash exacto siguen siendo necesarias; este corte no
emite un GO independiente sobre sí mismo ni declara C5 cerrado.
