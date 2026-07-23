# Decisión de autorización V3 para el alta de contratación temporal

Fecha: 23 de julio de 2026.

Estado: decisión de arquitectura; frontera de aplicación O2-04 implementada en
la rama de trabajo y pendiente de revisión independiente. El efecto atómico
O2-05 sigue pendiente.

## Corte implementado en O2-04

La aplicación de contratación temporal ya no acepta ni construye la identidad
o la autorización provisionales del módulo. Usa directamente:

- `ResolutorContextoAutorizacionAltaV3`, que entrega el
  `VinculoAutenticacionActorV2` y el
  `ResultadoContextoActorRegistradoV2` comunes;
- `ResolutorMotivoAutorizacionAltaV3`, que resuelve la referencia exacta del
  catálogo publicado;
- `AutorizadorSolicitudLigadaV3`, que devuelve decisión y confirmación durable
  V3;
- `SelladorAmbitoIdempotencia`, cuyo alias activo es la referencia del recurso
  que se autoriza antes de preparar una reserva.

La secuencia de aplicación revalida vínculo, decisión y confirmación antes de
preparar y de nuevo antes de devolver un recibo o solicitar el efecto. Un
reintento ya confirmado no evita el PDP. Los contratos no reciben HTTP,
cookies ni estado de navegador y son los mismos para web, escritorio, CLI y
MCP.

Este corte no afirma atomicidad: hasta cerrar O2-05 puede existir una reserva
interna autorizada que no llegue a efecto si la concesión vence o falla una
dependencia posterior. La reserva no se expone como endpoint y no equivale a
un expediente, una auditoría ni un evento. O2-05 debe trasladar preparación,
consumo de concesión y confirmación del efecto al único `COMMIT` descrito en
esta decisión.

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

La referencia autorizable de cada invocación será el alias HMAC activo del
ámbito lógico de la operación, derivado de:

- clave de idempotencia;
- organización;
- actor;
- perfil.

Nunca se persiste ni se expone la clave idempotente original. El recurso V3
incluirá, como ámbitos o atributos resueltos por servidor, la organización, el
centro, la categoría y la definición/versionado del flujo. La acción y
finalidad son valores cerrados de la política publicada, no texto aportado por
el cliente.

La identidad durable no usará una única HMAC como clave primaria lógica. Un
conector de llavero calculará el alias activo y los alias de las generaciones
anteriores admitidas para verificación. PostgreSQL resolverá cualquiera de
ellos a una única reserva, rechazará que dos alias converjan en reservas
distintas y añadirá el alias activo al reintentar. La huella de petición
aplicará el mismo patrón: una coincidencia con cualquier generación retenida
acredita igualdad; el valor de la generación activa se incorpora a la historia.

Las claves anteriores se conservarán solo para verificación durante, como
mínimo, todo el plazo de retención idempotente. La matriz de generaciones
publicada indicará activación, fin de emisión, fin de verificación y retirada.
Rotar de v1 a v2 deberá conservar el mismo expediente y recibo. La sintaxis
versionada sin esta convivencia no se considera rotación resuelta.

## Secuencia objetivo

```text
credencial breve ligada al emisor
  -> revalidación de autenticación y ContextoActor V2 registrado
  -> vínculo autenticación-actor V2
  -> resolución de flujo, motivo catalogado y alias HMAC activos/retirados
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
  -> recibo o reconciliación por cualquier alias admitido del mismo ámbito
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

Un resultado de `COMMIT` indeterminado se reconcilia por el conjunto de alias
HMAC admitidos y las huellas de petición de generaciones retenidas. No se
repite a ciegas.

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
- rotación v1→v2 con convivencia, sin segunda reserva ni conflicto falso;
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
