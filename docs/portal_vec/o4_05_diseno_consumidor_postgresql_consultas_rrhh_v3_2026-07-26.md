# O4-05: diseño del consumidor PostgreSQL de consultas RRHH V3

## Estado y alcance

Este documento fija el diseño del siguiente corte del camino crítico de
Contratación temporal. El objetivo es implementar `SesionConsultaRRHH` con
PostgreSQL real para el cuadro operativo y el detalle de expediente.

El corte deberá verificar y consumir VEC-AD-3, leer la proyección autorizada y
registrar el acceso en una única transacción. No incluye todavía identidad
corporativa, HTTP, composición raíz, TLS productivo ni aceptación E2E.

## Decisiones no negociables

1. La función de confirmación de alta no se reutiliza ni se generaliza.
2. Cuadro y detalle tienen consumidores y funciones exteriores nominales
   separados.
3. El adaptador Go recibe una orden nominal; nunca capacidad o material
   aislados.
4. La autoridad final es PostgreSQL. El resumen Go solo sirve para cotejos
   defensivos.
5. La autorización dura como máximo cinco segundos y se vuelve a comprobar
   antes de terminar la transacción.
6. Un replay interno puede reconociliar un resultado ambiguo, pero nunca
   autoriza una segunda entrega de datos.
7. Denegado, ausente y ajeno no forman un oráculo de existencia.

## Vocabulario cerrado

| Campo | Cuadro | Detalle |
|---|---|---|
| Acción | `contratacion_temporal.cuadro.consultar` | `contratacion_temporal.expediente.consultar` |
| Finalidad | `gestion_operativa_contratacion_temporal` | `tramitacion_expediente_contratacion_temporal` |
| Tipo de recurso | `cuadro_rrhh_contratacion_temporal` | `expediente_contratacion_temporal` |
| Audiencia | `vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1` | `vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1` |
| Ligadura | ámbito, filtros, límite y cursor | expediente y versión observada |

Ambas decisiones deben fijar además `modulo_id=contratacion_temporal`. No se
aceptan comodines ni parámetros con los que el llamador pueda elegir acción,
finalidad, audiencia, módulo o tipo.

## Evolución del gobierno común VEC-AD-3

La tabla `clave_capacidad_version` nació cerrada a la audiencia de alta. Una
migración ascendente nueva sustituirá su restricción por una lista positiva de
exactamente tres audiencias:

- confirmación de alta;
- consulta de cuadro RRHH;
- consulta de detalle RRHH.

No se modifica la migración histórica `000001`. La reversión solo restaurará la
restricción original cuando no existan claves, punteros, consumos ni funciones
dependientes de las audiencias de consulta.

Se crearán dos consumidores comunes:

```text
vec_autorizacion_atestada_v3.
  registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(10 piezas)

vec_autorizacion_atestada_v3.
  registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(10 piezas)
```

Cada consumidor fija sus constantes en el cuerpo SQL. Puede reutilizar los
validadores criptográficos privados ya instalados, pero no expondrá una
función genérica parametrizable.

## Rol y separación de responsabilidades

El grupo técnico `vec_contratacion_temporal_consultor_rrhh` pertenece al módulo
Contratación temporal. Se crea mediante un delta propio del módulo, no mediante
el bootstrap de autorización atestada.

El grupo es `NOLOGIN`, sin privilegios administrativos ni bypass de RLS. Cada
cuenta productiva:

- es un `LOGIN` nominativo;
- tiene una única membresía directa al grupo consultor;
- no puede usar `SET ROLE`;
- no pertenece a ejecutor, migrador, propietario, gobernador, confirmador ni
  lectores anteriores;
- solo puede conectar, usar el esquema y ejecutar las dos funciones exteriores
  del módulo.

No recibe acceso directo a tablas, secuencias, consumidores VEC ni funciones
internas.

## Transacción exterior

Habrá una función exterior de cuadro y otra de detalle. Cada una realizará:

1. acreditar `session_user`, función, propietario, ACL, primario, TLS,
   aislamiento serializable, UTC y límites;
2. reconstruir el canon exacto de la solicitud Go y su SHA-256;
3. reconstruir y cotejar organización, ámbito, acción, finalidad, módulo,
   recurso y huella de consulta;
4. consumir las diez piezas mediante el consumidor nominal correspondiente;
5. rechazar una capacidad ya consumida para impedir una segunda lectura;
6. leer una instantánea coherente y aplicar el ámbito;
7. registrar el acceso en una cadena inmutable;
8. revalidar vigencia y revocaciones;
9. devolver proyección minimizada y recibo.

Cualquier error revierte consumo, lectura, cursor y auditoría. Un `COMMIT`
ambiguo no se reintenta a ciegas: se reconcilia sin volver a entregar datos o
se solicita una autorización nueva.

## Registro durable de acceso

Se crearán tablas propias para control de cadena, registros de acceso y, si
resulta necesario, cursores de cuadro. No constituyen una segunda fuente de
verdad del expediente.

El registro conserva referencias opacas, decisión, sesión, perfil,
organización, ámbito, acción, finalidad, dominio y huella de consulta,
referencia y versión cuando proceda, huellas de capacidad/consumo/resultado,
auditoría VEC, total publicado, resultado genérico e instante.

No conserva texto de búsqueda, cursor en claro, documentos, observaciones,
nombres ni otros datos personales. Las tablas usan RLS forzada, propietario
exacto, inmutabilidad y cadena `anterior_sha256 → huella_sha256`.

## Cuadro y paginación

El orden estable es:

```text
actualizado_en DESC, expediente_ref DESC
```

Se leen como máximo `limite+1` filas para determinar `hay_mas`, con límite
absoluto de cien resultados.

Si se necesita cursor, será opaco y estará ligado a organización, ámbito,
filtros, límite, corte global y última fila. Solo se almacenará su huella,
nunca el token en claro. La consulta siguiente deberá usar una autorización
nueva ligada a ese cursor.

`expediente_version_integral.registrada_en` no sirve por sí sola como corte
global: su precisión admite empates entre expedientes y dos transacciones
pueden confirmar en un orden distinto al de sus relojes. C2-C cerró esa
precondición con `000037`: el backfill define una base reproducible y cada
versión posterior bloquea un singleton hasta `COMMIT` antes de recibir un
ordinal global único. Esto no activa aún la paginación; faltan cursor
autenticado, fachada y transacción exterior de consulta.

## Detalle

`version_observada=0` solicita la versión actual. Una versión no nula exige
coincidencia exacta. Inexistente, fuera de ámbito y no autorizado producen el
mismo resultado exterior.

El agregado bruto permanece dentro del adaptador acreditado. Go lo reduce y
valida mediante el contrato de detalle antes de exponerlo.

## Límites iniciales

| Límite | Valor máximo |
|---|---:|
| Duración de capacidad | 5 s |
| `statement_timeout` | 4 s |
| `lock_timeout` | 1 s |
| Inactividad en transacción | 6 s |
| Filas del cuadro | 100 |
| Respuesta de cuadro | 256 KiB |
| Detalle/agregado | 256 KiB |
| Profundidad JSON | 24 |
| Elementos JSON | 16.384 |

Los límites se aplican antes de reservar memoria o realizar trabajo costoso.

## Cortes verificables

### C2-A — consumidores comunes VEC

Estado: GO técnico independiente en `a0d39c1`.

- evolución de la restricción de audiencias;
- consumidor nominal de cuadro;
- consumidor nominal de detalle;
- migraciones ascendentes y reversión protegida;
- pruebas A/B, diez piezas, replay, ACL y revocación.

### C2-B — rol y persistencia de acceso

Estado: GO técnico independiente en `2820759`.

- rol consultor RRHH;
- cadena de auditoría y registros minimizados;
- ausencia deliberada de cursor hasta disponer de corte global alineado con
  `COMMIT`;
- RLS, inmutabilidad y reversión.

### C2-C — publicación global estable

Estado: GO técnico independiente en `3cb17ca`, documentado en
[la revisión de `000037`](revisiones/o4_05_revision_publicacion_global_rrhh_2026-07-26.md).

- singleton transaccional y proyección 1:1 de solo adición;
- backfill bajo bloqueo, ordenado por referencia `COLLATE "C"` y versión;
- `corte_base` explícito que no afirma orden histórico;
- ordinal posterior único y limitado a `2^53-1`, retenido hasta `COMMIT`;
- extracción indexable y cotejada del resumen RRHH;
- RLS, ACL, inmutabilidad y reversión `17→16` protegida;
- PostgreSQL 18.4 real, rollback, concurrencia y safe-down.

### Pendiente tras C2-C — funciones exteriores de Contratación

- cuadro y detalle en transacción única;
- canon compartido con Go;
- lectura por ámbito, versión y filtros;
- cursor opaco ligado al corte global;
- recibo durable y límites.

### C2-D — adaptador Go

- implementación de `SesionConsultaRRHH`;
- pool y función acreditados;
- decodificación estricta y copia defensiva;
- normalización de errores sin oráculo;
- cierre, cancelación y resultado ambiguo.

### C2-E — evidencia real

- PostgreSQL 18.4 efímero;
- concurrencia, replay, revocación y fallos;
- ACL y sustitución hostiles;
- matriz TLS;
- revisión independiente del TCB completo.

## Puerta de producción

C2 solo queda cerrado cuando consumo, lectura y auditoría sobreviven juntos a
pruebas reales de reinicio, concurrencia, revocación y reversión, y el adaptador
Go devuelve únicamente resultados validados después de `COMMIT`.

Este diseño no autoriza producción por sí solo.
