# Decisión de autorización V3 para el alta de contratación temporal

Fecha: 23 de julio de 2026.

Estado: decisión de arquitectura; implementación O2-04/O2-05 pendiente de
revisión independiente.

## Problema

El primer corte del caso de uso separa la preparación idempotente de la
confirmación. Esa separación permite probar concurrencia y persistencia, pero
no es todavía una frontera de producción:

- la identidad y autorización locales son provisionales y no deben convertirse
  en una segunda autoridad frente a VEC;
- una reserva no debe quedar expuesta como operación autónoma sin autorización;
- una repetición confirmada no debe devolver un recibo basándose únicamente en
  estado de navegador o en una clave idempotente;
- la concesión, el expediente, la actuación, la auditoría y el evento deben
  quedar ligados por una única transacción autoritativa.

La solución tampoco puede depender de cookies, de almacenamiento del
navegador, de HTTP ni de una aplicación concreta. Web, escritorio, CLI y MCP
deben invocar la misma capacidad.

## Decisión

El módulo reutilizará sin reinterpretar las capacidades comunes:

- `ResultadoContextoActorRegistradoV2`;
- `VinculoAutenticacionActorV2`;
- `SolicitudAutorizacionLigadaV3`;
- `DecisionAutorizacionLigadaV3`;
- `ConfirmacionRegistroConcesionAutorizacionLigadaV3`.

Se retirarán del camino real los constructores locales de identidad y
autorización. No habrá conversión desde esos tipos provisionales ni un modo de
aceptarlos por compatibilidad.

La referencia autorizable será el ámbito lógico estable de la operación,
derivado mediante HMAC versionado de:

- clave de idempotencia;
- organización;
- actor;
- perfil.

Nunca se persiste ni se expone la clave idempotente original. El recurso V3
incluirá, como ámbitos o atributos resueltos por servidor, la organización, el
centro, la categoría y la definición/versionado del flujo. La acción y
finalidad son valores cerrados de la política publicada, no texto aportado por
el cliente.

## Secuencia objetivo

```text
credencial breve ligada al emisor
  -> revalidación de autenticación y ContextoActor V2 registrado
  -> vínculo autenticación-actor V2
  -> resolución de flujo, motivo catalogado y ámbito HMAC
  -> solicitud/decisión PDP V3
  -> registro durable de la concesión candidata
  -> transacción PostgreSQL de efecto
       - releer y consumir la concesión V3
       - cotejar ámbito, organización, actor, perfil, acción, recurso,
         finalidad, motivo, correlación, contexto y vigencia
       - resolver la idempotencia y sus referencias ganadoras
       - confirmar expediente y actuación
       - registrar auditoría e inbox/outbox
  -> COMMIT
  -> recibo o reconciliación por el mismo ámbito
```

La preparación O2-02/O2-03 queda como pieza interna reutilizable de esa
transacción y nunca como endpoint ni como autoridad. O2-05 decidirá si la
función se integra físicamente en la confirmación o se invoca como auxiliar
privado dentro de la misma transacción; en ambos casos el runtime exterior solo
recibirá `EXECUTE` sobre una función cerrada.

## Repeticiones

Cada invocación revalida la identidad y obtiene una decisión V3 vigente. La
transacción aplica estas reglas:

1. una decisión no consumida puede ligarse a un único efecto;
2. repetir la misma operación exacta devuelve el mismo recibo;
3. una misma decisión para otro efecto se rechaza;
4. el mismo ámbito con otra huella de petición se rechaza;
5. recuperar un recibo nunca evita la revalidación de identidad/autorización;
6. una expiración o revocación observada antes del efecto deniega;
7. una cancelación posterior a un `COMMIT` confirmado no convierte el éxito en
   fallo.

Un resultado de `COMMIT` indeterminado se reconcilia por ámbito HMAC y huella
de petición. No se repite a ciegas.

## Base de datos y privilegios

Autorización V3 y contratación temporal deben residir en la misma base
PostgreSQL para el efecto atómico. Conservan propietarios, migradores y
ejecutores distintos.

La capacidad cruzada será mínima:

- el propietario de contratación temporal recibe `USAGE` del esquema de
  autorización y `EXECUTE` sobre una función exacta de cotejo/consumo;
- ningún runtime obtiene `SELECT`, `INSERT`, `UPDATE`, `DELETE`, membresía de
  propietarios ni `SET ROLE`;
- las funciones fijan `search_path = pg_catalog`, tiempo UTC, límites de
  bloqueo/sentencia y orden de bloqueos;
- consumo, expediente, historia, auditoría y outbox son solo adición;
- una reversión destructiva requiere una autorización operativa explícita y no
  forma parte del runtime.

Separar las dos autoridades en bases diferentes deja O2-05 en `NO-GO`: un
outbox no sustituye la atomicidad de una autorización revocable.

## Clientes y sesiones

El caso de uso no recibe cookies ni cabeceras libres de identidad.

- Web mantiene una credencial breve solo en memoria y no usa `Cookie`,
  `Set-Cookie`, `localStorage` ni `sessionStorage` como autoridad.
- Escritorio usa conectores de certificado/mTLS y, en la red corporativa,
  Kerberos cuando corresponda.
- CLI y MCP emplean credenciales ligadas al emisor y capacidades permitidas
  por operación.

Todos terminan en el mismo puerto de aplicación. Ninguna regla funcional o de
permiso se implementa en JavaScript, DOM, sesión HTTP o cliente de escritorio.

## Puertas de cierre

O2-04/O2-05 no se cierran hasta demostrar:

- imposibilidad de fabricar o serializar solicitud, vínculo, decisión y
  confirmación nominales;
- denegación por actor, perfil, organización, finalidad, acción, recurso,
  motivo, flujo, contexto, versión o correlación distintos;
- consumo único e idempotencia bajo sesiones concurrentes;
- expiración y revocación mientras se espera un bloqueo;
- ausencia de reserva/expediente/auditoría/evento parcial ante cualquier fallo;
- replay confirmado con revalidación vigente;
- timeout antes y después de `COMMIT`, reinicio y reconciliación;
- ACL negativas y ausencia de lectura cruzada de tablas;
- igualdad de comportamiento desde dobles contractuales de web y escritorio,
  sin cookies.

La aprobación funcional de RRHH y las conformidades de Sistemas, Seguridad y
DPD siguen siendo puertas externas; el software y sus pruebas no las
sustituyen.
