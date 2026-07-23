# Decisión propuesta — idempotencia neutral de canal para el alta

Fecha: 23 de julio de 2026.

Estado: propuesta de Dirección pendiente de las revisiones independientes
O2-08A/O2-09A. No autoriza composición ni producción.

## Problema detectado

El contrato de `ports` exige que cada cliente genere una clave UUIDv4 con
CSPRNG y la conserve durante un reintento. O2-09A cumple esa regla y entrega
`clave_idempotencia` al ejecutor neutral.

O2-08A, en cambio:

- admite en el cuerpo únicamente `SolicitudCentro`;
- rechaza la clave tanto en JSON como en `Idempotency-Key`;
- espera que una autoridad de servidor incorpore otra clave al contexto.

No existe todavía un mecanismo que permita a esa autoridad recuperar la misma
clave después de que una petición web se cancele o pierda su respuesta. Una
clave nueva en cada reintento puede crear un segundo efecto; confiar en una
cookie o en almacenamiento del navegador incumple las decisiones del
proyecto.

## Clasificación de la clave

La clave de idempotencia no es:

- identidad;
- autenticación;
- autorización;
- rol, permiso o capacidad;
- secreto;
- prueba de autoría.

Es un identificador aleatorio no confiable que expresa «este reintento
pertenece a la misma intención». El servidor nunca concede por su valor. La
seguridad procede de:

1. contexto de autenticación, sesión, perfil y organización resuelto por una
   frontera confiable;
2. autorización revalidada por el caso de uso;
3. alias HMAC del ámbito, derivado en servidor;
4. huella HMAC del contenido normalizado;
5. transacción durable que liga ambos valores y rechaza divergencias.

Aceptar el identificador como dato no convierte al cliente en autoridad.

## Decisión propuesta

La petición O2-08B utilizará un envelope cerrado:

```json
{
  "clave_idempotencia": "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
  "solicitud": {
    "centro_ref": "referencia-opaca",
    "contacto_ref": "referencia-opaca",
    "categoria_ref": "referencia-opaca",
    "grupo_subgrupo": "A2",
    "motivo_clave": "clave-gobernada",
    "detalle": "texto",
    "periodo": {
      "inicio": "2026-08-01T00:00:00Z",
      "fin": "2026-08-31T00:00:00Z"
    },
    "rc": {
      "existe": false
    },
    "documentos_adjuntos": [],
    "observaciones": ""
  }
}
```

Reglas:

- el cliente genera una UUIDv4 canónica con CSPRNG;
- la mantiene únicamente en memoria durante revisión, envío y reintento de la
  misma intención;
- no se envía en query ni `Idempotency-Key`, para evitar registros
  automáticos de intermediarios;
- no aparece en respuesta, DOM, estado renderizable, telemetría, log, auditoría
  ni documento;
- el manejador la valida como dato y la copia defensivamente;
- `AutoridadContextoCanal` aporta autenticación, sesión, perfil y organización,
  pero no inventa ni sustituye la clave;
- aplicación revalida el contexto, calcula mediante conectores de llavero los
  alias HMAC activos/retenidos y deriva la huella del contenido normalizado;
- la clave original se borra del material efímero cuando el caso de uso ya ha
  generado sus sellos;
- PostgreSQL solo conserva alias y huellas; nunca la UUID en claro.

## Semántica

| Situación | Resultado |
| --- | --- |
| Misma identidad efectiva, clave y contenido | Mismo expediente y recibo. |
| Misma identidad efectiva y clave, contenido distinto | Conflicto, cero segundo efecto. |
| Misma clave bajo otra organización, actor o perfil | Ámbito HMAC distinto; no permite consultar ni recuperar el primer efecto. |
| Clave inválida, ausente o centinela | Rechazo anterior al PDP y a persistencia. |
| Cancelación antes del efecto | Reintento con la misma clave y contenido. |
| Cancelación o respuesta perdida tras `COMMIT` | Reconciliación O2-06 y mismo recibo; nunca clave nueva automática. |
| Rotación HMAC | Alias retenidos convergen en la misma reserva y se añade el activo. |

La titularidad del recibo se revalida en cada recuperación. Conocer una UUID de
otra persona no permite recuperar su expediente.

## Neutralidad

Web, escritorio, CLI y MCP construyen el mismo comando:

```text
clave aleatoria de intención + solicitud funcional
```

Cada adaptador decide cómo mantenerla durante la interacción, sin convertirla
en estado durable:

- web: memoria de la página mientras la intención no cambie;
- escritorio: memoria del proceso o diario local cifrado si el producto exige
  recuperación tras cierre, mediante un conector futuro;
- CLI/MCP: argumento o sobre de operación gobernado, nunca variable de
  autoridad.

Todos invocan el mismo caso de uso y reciben la misma proyección minimizada.

## Alternativas descartadas

### Clave exclusivamente generada por el servidor en cada `POST`

No permite ligar un reintento después de perder la respuesta que contenía la
primera clave. Puede duplicar el efecto.

### Cookie, `localStorage` o `sessionStorage`

Ata la operación a la web, aumenta exposición y contradice la decisión de
clientes neutrales sin cookies.

### Idempotencia derivada solo del contenido

Colapsa dos intenciones legítimas con el mismo contenido y no expresa la
voluntad de reintento. Tampoco sustituye la clave aleatoria.

### Clave libre como autoridad o índice global

Permitiría colisiones entre identidades y recuperación cruzada. La clave solo
puede usarse después de ligarla a contexto autoritativo y contenido mediante
HMAC.

### `Idempotency-Key`

Es un patrón interoperable habitual, pero proxies y registros de acceso suelen
capturar cabeceras. Mientras la infraestructura corporativa no garantice su
redacción extremo a extremo, el envelope cerrado minimiza el riesgo y mantiene
el mismo contrato lógico.

## Cambios requeridos si las revisiones emiten GO

O2-08B deberá:

1. cambiar el DTO HTTP al envelope cerrado;
2. trasladar la clave validada a `SolicitudRegistrarExpediente`;
3. exigir que el contexto confiable devuelva `ClaveIdempotencia` y `Solicitud`
   vacías, rechazando contaminación;
4. mantener identidad y organización exclusivamente en el contexto confiable;
5. actualizar OpenAPI, límites, pruebas de duplicados, Unicode, redacción,
   replay y reutilización con otro contenido;
6. demostrar que ninguna capa registra cuerpo o clave;
7. conectar O2-09A sin traducción semántica ni clave sustituta.

O2-09A deberá conservar la misma clave solo mientras el borrador normalizado no
cambie. Volver a edición o alterar cualquier campo crea una nueva intención y,
por tanto, una nueva clave al regresar a revisión.

O2-10 probará la semántica completa con respuesta perdida, reinicio,
concurrencia, rotación de HMAC y dos identidades que usen la misma UUID.

## Condición de adopción

La propuesta se convierte en decisión únicamente después de:

- dictamen independiente O2-08A/O2-09A;
- confirmación de que O2-05 liga alias, huella y actor como se documenta;
- revisión de seguridad de logs del proxy y de aplicación;
- pruebas O2-10 de extremo a extremo.

Hasta entonces la API y la vista siguen siendo candidatos aislados y no deben
conectarse mediante un adaptador provisional.
