# Confianza y capacidad breve VEC-AD-3 de O2-05

Fecha: 23 de julio de 2026.

Estado: corte nominal implementado, pendiente de revisión independiente.
`NO-GO` para producción y para cerrar O2-05.

## Alcance

Este corte añade, sin modificar el significado de VEC-AD-2:

- configuración de confianza V3 con lista positiva Ed25519, audiencia exacta,
  ventanas, secuencia monotónica, versión explícita de raíz, rotación por
  identificador y revocación;
- verificación de un `COSE_Sign1` con payload separado sobre la preimagen
  exacta VEC-AD-3 y AAD V3;
- prueba nominal que liga decisión, motivo catalogado, contexto de actor V2,
  payload, sobre, raíz y revisión de confianza;
- emisor HMAC separado que acuña una capacidad con vigencia máxima de cinco
  segundos;
- verificador de capacidad independiente, con su propia copia gobernada de
  claves activas o retenidas;
- una única exportación canónica y defensiva para el futuro consumidor SQL.

No se ha añadido HTTP, cookies, almacenamiento de navegador ni reglas
dependientes de web. El mismo contrato puede ser invocado desde web,
escritorio, CLI o MCP.

## Separación de autoridades

Los tres componentes tienen tipos y responsabilidades diferentes:

1. `ServicioConfianzaAtestacionAutorizacionV3` contiene únicamente claves
   públicas y verifica COSE. No puede emitir capacidades.
2. `EmisorCapacidadesAtestacionAutorizacionV3` contiene únicamente la clave
   HMAC de emisión. Recibe una prueba V3 ya validada y no posee credenciales
   SQL.
3. `VerificadorCapacidadesAtestacionAutorizacionV3` representa al consumidor.
   Verifica una exportación canónica con una copia gobernada de claves y no
   puede emitir.

La composición productiva deberá ejecutar el emisor en un broker segregado.
El constructor con material en memoria permite probar el protocolo, pero no
es un HSM/KMS ni habilita producción.

## Ligadura de operación y efecto

La operación no se recibe como texto libre: se extrae de
`SolicitudAutorizacionLigadaV3.Datos().Accion`.

La confirmación SQL O2-05 eleva el perfil de alta a exactamente tres ámbitos
y cinco atributos: `efecto_huella_sha256`, `flujo_ref`, `flujo_version`,
`flujo_huella_sha256` y `huella_peticion_hmac_activa`. La nueva huella
compromete los bytes del efecto completo `efecto-alta.v2`; por transitividad,
la capacidad compromete también referencias, solicitud, RC, documentos,
fechas y actuación. O2-06 deberá construir ese mismo canon antes de pedir la
decisión, sin alterar el núcleo ni usar mapas abiertos fuera del DTO
canónico.

`efecto_ref` se deriva de `Recurso.Referencia`, que en O2-04 es el ámbito de
idempotencia HMAC resuelto por el servidor. `huella_efecto_sha256` se deriva
con `Recurso.HuellaContextoAutorizacionSHA256()`, que compromete el conjunto
canónico completo de ámbitos y atributos del recurso. Acción, referencia de
recurso y huella de contexto ya están, además, dentro de la solicitud ligada,
la decisión y el mensaje VEC-AD-3 firmado.

No se acepta operación, referencia de efecto ni huella por un canal lateral.
El consumidor SQL coteja esos valores con la reserva O2-04 y el efecto V2 que
materializa, y consume la decisión en esa misma transacción.
Una reserva que no corresponda al mismo ámbito HMAC y contexto firmado debe
fallar cerrada.

## Exportación canónica

El puerto público
`ports.ExportadorCapacidadAtestacionAutorizacionV3` solo expone:

```go
ExportacionCanonicaParaConsumidor() ([]byte, error)
```

El valor concreto bloquea JSON, texto, binario, gob, CBOR, YAML y XML
genéricos, y redacta `fmt` y `slog`. La salida deliberada es un objeto JSON
plano, canónico, sin claves desconocidas o duplicadas, con 37 campos y esquema:

```text
vec.autorizacion.capacidad-registro-consumo-atestado.v3
```

La MAC HMAC-SHA-256 cubre en orden cerrado:

- esquema y versión;
- clave, versión, revisión y huella de gobierno;
- emisor y audiencia del consumidor;
- nonce, emisión y expiración;
- decisión y motivo;
- payload VEC-AD-3, sobre COSE y prueba de confianza;
- referencia y huella del contexto de actor;
- audiencia de despliegue;
- operación, referencia y huella del efecto;
- límite de la decisión;
- revisión, secuencia, ventana, identificador y versión de raíz de confianza;
- suite VEC-AD-3.

La preimagen usa para cada valor el encuadre:

```text
numero_de_bytes_UTF8:valor\n
```

El consumidor PostgreSQL deberá reproducir exactamente este orden y no deberá
aceptar un DTO reconstruido, una serialización alternativa o un campo
adicional.

## Tiempo, rotación y revocación

- La ventana es *half-open*: `emitida_en <= ahora < expira_en`.
- La duración nunca supera cinco segundos y se recorta por la decisión, la
  raíz, la configuración de confianza y la clave HMAC.
- Solo una clave en estado `emision` puede acuñar.
- Una clave `verificacion` puede comprobar emisiones previas durante su
  retención.
- Una clave `revocada` no emite ni verifica.
- El consumidor coteja identificador, versión, revisión/huella de gobierno,
  emisor, audiencia y ventana además de la MAC.
- `configuracion_secuencia` y `raiz_version` se autentican en la MAC; la
  secuencia también forma parte de la huella canónica de configuración.

La revocación y rotación productivas deberán proceder de la autoridad durable
y releerse/bloquearse dentro de la transacción SQL. El consumidor deberá
bloquear los punteros vigentes y exigir que secuencia, revisión, huella,
identificador, versión y huella SPKI coincidan exactamente. Una instantánea Go
no sustituye esa comprobación final.

La secuencia monotónica evita que una revisión opaca parezca más reciente por
su nombre, pero no resuelve una restauración completa y coordinada de base de
datos y configuración. Producción sigue necesitando un ancla externa
monotónica/WORM que detecte ese retroceso.

## Matriz cubierta

Las pruebas focales cubren:

- firma, clave, `kid`, suite, audiencia, payload y AAD exactos;
- raíz/configuración vigentes y revocadas;
- solape de dos raíces durante rotación y rechazo de `kid` o SPKI duplicados;
- decisión, motivo y contexto cruzados;
- cancelación antes y después de dependencias;
- emisión y verificación con componentes separados;
- salto entre procesos mediante la exportación;
- evolución cerrada del perfil O2-04 al efecto completo O2-05;
- operación y recurso/efecto derivados solo de contenido firmado;
- expiración a cinco segundos;
- rotación a clave retenida, otra clave y clave revocada;
- secuencia de configuración y versión de raíz nulas, cruzadas o alteradas;
- alteración de decisión, motivo, payload, contexto, audiencia, operación,
  efecto, huella o MAC;
- JSON duplicado, abierto, reordenado o con contenido posterior;
- copia defensiva, bloqueo de codecs y redacción de logs.

## Pendiente y puertas externas

Este corte todavía no implementa:

- el adaptador Go/pgx O2-06 que proyectará `efecto-alta.v2`;
- la composición productiva que lo conectará con la aplicación;
- broker Unix/mTLS y separación física de identidades;
- HSM/KMS, custodia, ceremonia, rotación y destrucción reales;
- recarga autoritativa de revocaciones durante bloqueo;
- ancla WORM/monotónica externa ante restauración completa de toda la base.

Hasta completar esas piezas y obtener revisión de Sistemas, Seguridad, DPD y
responsable funcional, la capacidad es un contrato probado, no una
autorización productiva.
