# Vínculo de ContextoActor V2 y autorización PDP V3

**Fecha de decisión:** 21 de julio de 2026
**Estado:** diseño aprobado para implementación; todavía no compuesto ni apto
para datos personales reales.

## Problema que resuelve

El ContextoActor durable compromete desde V2 la versión exacta de la cuenta.
El vínculo de autenticación V1 y la solicitud/decisión PDP V2 son contratos
históricos cerrados: su huella no contiene `cuenta_version`. No se pueden
reinterpretar, completar ni degradar sin invalidar sus bytes y su evidencia.

Además, `vec_autorizacion.contexto_actor_v1` solo admite una huella por
`(vca_, versión)`, mientras dos resoluciones legítimas del mismo vínculo pueden
tener distinto método, garantía o instante y, por tanto, distinta huella. Una
copia o sincronización eventual crearía dos autoridades y podría invalidar
sesiones entre sí.

## Decisión

Se congelan sin cambios:

- el canon ContextoActor V1;
- `VinculoAutenticacionActorV1`;
- `SolicitudAutorizacionLigadaV2`;
- la solicitud efectiva y la decisión reforzada V2;
- las evidencias históricas ya emitidas con esos esquemas.

La vía nueva será nominal y aditiva:

1. ContextoActor canónico V2 con `cuenta_version > 0`;
2. resultado registrado V2 que conserva el recibo `rca_`, las huellas y la
   autoridad, no solo la proyección de actor;
3. `VinculoAutenticacionActorV2`;
4. `SolicitudAutorizacionLigadaV3`;
5. solicitud efectiva
   `vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2`;
6. decisión
   `vec.autorizacion.decision.reforzada.v3.solicitud-ligada`;
7. wrappers PostgreSQL de Bolsa que solo aceptan decisión V3.

No habrá conversión automática V1→V2 ni V2→V3.

## Contenido mínimo del vínculo V2

El vínculo V2 conserva los datos cerrados de autenticación y actor que ya
existían en V1 y añade exclusivamente:

- esquema de vínculo V2 y bloque de versión 2;
- `registro_contexto_ref` (`rca_`);
- esquema exacto `vec.contexto-actor.vinculado.v2`;
- `contexto_actor_cuenta_version`;
- huella SHA-256 del manifiesto de procedencia;
- autoridad efectiva, que para uso real debe ser exactamente
  `autoridad_maestra_acreditada`.

También conserva `contexto_actor_ref`, `contexto_actor_version` y la huella de
los bytes V2. No transporta `OperacionRef`, bytes completos, DNI, nombres,
correo, roles, permisos ni claims libres.

El vínculo se fabrica desde el resultado registrado V2. Recibir únicamente un
`ContextoActor` es insuficiente porque perdería el recibo y su procedencia.

## Una única autoridad PostgreSQL

`vec_contexto_actor_v1` es la única autoridad de contexto. El sufijo del
esquema identifica la versión del módulo, no la del documento canónico.

El PDP no copiará filas ni mantendrá un puntero paralelo. En la misma
transacción de autorización invocará una función cerrada del módulo de contexto
que:

- localiza el `rca_` exacto;
- coteja bytes V2, huella, manifiesto y autoridad;
- coteja cuenta y versión, persona y versión, perfil y versión, `vca_` y
  versión, y todos los vínculos de módulo;
- bloquea y relee sus punteros actuales en un orden único;
- aplica estados y ventanas half-open con el instante autoritativo de la
  transacción;
- devuelve solo un resultado cerrado, sin exponer tablas.

La operación exige que `vec_contexto_actor_v1` y `vec_autorizacion` residan en
la misma base PostgreSQL, con propietarios y credenciales separados. Si se
separan en dos bases, la composición queda en NO-GO porque se pierde la
atomicidad. Un outbox puede servir para telemetría, nunca para decidir si un
contexto revocado sigue siendo utilizable.

## Privilegios

- La función de acreditación de uso pertenece al propietario del módulo de
  contexto y fija `search_path = pg_catalog`.
- Solo el propietario de autorización recibe `USAGE` del esquema y `EXECUTE`
  sobre esa función exacta.
- El runtime de autorización solo invoca su función exterior
  `SECURITY DEFINER`.
- Los runtimes de autorización, contexto y Bolsa no reciben lectura ni
  escritura cruzada de tablas, membresías entre propietarios o `SET ROLE`.
- Las tablas duplicadas históricas de autorización quedan fuera de las
  decisiones V3 y se conservan únicamente mientras las evidencias V1/V2 lo
  exijan.

## Recorrido T20E objetivo

```text
mTLS/Kerberos
  -> resolver y registrar ContextoActor V2
  -> vínculo autenticación-actor V2
  -> solicitud y decisión PDP V3
  -> acreditación transaccional del rca_ en la autoridad única
  -> preparación/confirmación PostgreSQL + KMS de Bolsa
  -> COMMIT
  -> relectura y verificación poscommit
  -> respuesta web y recibo
```

## Pruebas bloqueantes

- V1 conserva sus vectores históricos y rechaza un actor con
  `cuenta_version > 0`.
- V2 rechaza cuenta versión cero, esquema o huella V1 y cualquier campo extra.
- Alterar `rca_`, versión de cuenta, huella del contexto, huella del manifiesto
  o autoridad deniega la operación.
- Una revocación o avance concurrente de cualquier puntero nunca produce un
  permiso obsoleto ni un interbloqueo.
- Una expiración mientras se espera un bloqueo termina en denegación.
- Fallar la acreditación revierte la decisión completa; no queda fila parcial
  ni compensación posterior.
- La decisión V3 cambia ante cualquier campo nuevo del vínculo y PostgreSQL
  consume los mismos vectores canónicos que Go.
- Los runtimes no pueden consultar directamente las tablas ni ejecutar
  funciones auxiliares.
- El E2E sobrevive reinicios, reconcilia un `COMMIT` incierto y no usa memoria,
  datos de presentación ni las tablas duplicadas V1.

## Puertas pendientes

Este diseño no acredita por sí solo una fuente corporativa. Producción seguirá
bloqueada hasta disponer de la carga maestra gobernada, identidad y Kerberos
corporativos, KMS/TSA autorizados, T12, T13 y las conformidades de Sistemas,
DPD y RRHH que correspondan.
