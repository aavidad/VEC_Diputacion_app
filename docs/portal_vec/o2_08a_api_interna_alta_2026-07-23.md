# O2-08A — contrato y adaptador HTTP interno del alta

Fecha: 23 de julio de 2026.

Estado: candidato aislado. No registra la ruta ni compone dependencias.

## Resultado

El paquete
`internal/modules/contrataciontemporal/adapters/httpinterno` traduce
exclusivamente:

```text
POST /api/interno/v1/contratacion-temporal/solicitudes
→ application.SolicitudRegistrarExpediente
→ ports.ReciboAlta
```

La entrada JSON contiene solo los campos de `domain.SolicitudCentro`. La
salida proyecta únicamente referencia y número de expediente, versión,
referencia de recibo e instante de confirmación. Las referencias de auditoría
y evento que valida el caso de uso no salen por HTTP.

El contrato normativo y neutral al cliente está en
`docs/api/contratacion_temporal_alta_interna_v1.yaml`.

## Frontera de confianza

`AutoridadContextoCanal` recibe únicamente `context.Context`. O2-07 deberá
inyectar una implementación que resuelva desde el contexto preparado por la
superficie interna:

- referencia de autenticación;
- referencia de sesión;
- referencia de perfil activo;
- referencia de organización;
- clave UUIDv4 opaca de idempotencia ligada al cliente y petición.

La autoridad devuelve esos campos en
`application.SolicitudRegistrarExpediente` con `Solicitud` vacía. El manejador
valida la forma de las referencias, exige que la parte funcional esté vacía y
añade una copia defensiva del JSON ya validado. No existe constructor público
de contexto autenticado ni se entrega `*http.Request` a la autoridad.

O2-07 deberá componer la superficie corporativa de garantía alta y las
dependencias reales. Este corte no registra el manejador en ningún servidor,
no resuelve certificado, Kerberos o credencial de cliente y no sustituye al
PDP del caso de uso.

## Contrato de entrada

El DTO HTTP es propio del adaptador. Rechaza antes de llamar a la autoridad:

- campos desconocidos, duplicados o con caja alternativa;
- `null`, segundo documento JSON, UTF-8 inválido y tipos incorrectos;
- cuerpo superior a 262144 bytes, profundidad superior a 8 o más de 1024
  tokens JSON;
- fechas que no sean fechas civiles a medianoche con sufijo `Z`;
- periodos negativos o superiores a 100 años;
- importes no enteros, no positivos, superiores a 922337203685477 céntimos o
  en moneda distinta de `EUR`;
- referencias fuera de 3–160 caracteres, adjuntos repetidos o más de 64;
- detalle u observaciones sin NFC, con espacios exteriores, controles no
  admitidos o más de 4000 caracteres.

La declaración de retención de crédito es cerrada: `existe=false` no admite
los demás campos; `existe=true` exige número, fecha, importe y documento.

No se acepta clave de idempotencia, identidad, sesión, perfil, actor,
organización, rol, permiso, decisión, capacidad ni evidencia de seguridad en
el JSON.

## HTTP y errores públicos

Solo se admite `POST`, `application/json` en UTF-8 y un `Accept` compatible
con JSON. No se admiten query, cookies, `Authorization`, cabeceras heredadas
de identidad o rol, `Idempotency-Key`, compresión de entrada ni override de
método.

Los errores usan:

```json
{
  "error": {
    "codigo": "peticion_no_valida",
    "clave_i18n": "api.contratacion_temporal.alta.error.peticion_no_valida",
    "correlacion_ref": "corr_00000000000000000000000000000000"
  }
}
```

La correlación la genera el servidor y no contiene identidad ni clave de
idempotencia. El manejador no registra errores, cuerpos, cabeceras ni datos
personales. Los errores privados se clasifican con `errors.Is` y nunca se
reflejan.

Todas las respuestas eliminan `Set-Cookie`, CORS, `Content-Encoding` y
`Retry-After`; fijan JSON UTF-8, `no-store, no-transform`, `nosniff`, CSP
cerrada, `no-referrer`, política de permisos, CORP y protección de marcos. No
se duplica HSTS ni la terminación TLS.

## Confirmación, replay y resultado pendiente

El manejador valida el `ports.ReciboAlta` completo, incluidas las referencias
internas que después omite, limita la versión al entero seguro JSON y rechaza
instantes futuros usando un reloj inyectado. La ligadura fuerte del recibo al
expediente pertenece al caso de uso y a la transacción; HTTP no crea una
segunda autoridad para reconstruirla.

Si el ejecutor devuelve un recibo válido junto con una cancelación observada
después del efecto, prevalece el recibo confirmado. O2-06 deberá expresar el
resultado indeterminado mediante un contrato neutral propio, sin importar este
adaptador HTTP. La composición O2-07 traducirá ese error neutral envolviéndolo
con `ErrResultadoAltaIndeterminado`; entonces HTTP responderá
`503 operacion_pendiente`, sin `Retry-After`. No hay reconciliación ni
reintento automático en O2-08A.

## Validación del OpenAPI

La prueba automatizada parsea el YAML con `gopkg.in/yaml.v3` —dependencia ya
presente— y comprueba estructura, objetos cerrados, DTO, límites técnicos y
catálogo estado–código. El repositorio no incorpora un validador semántico de
OpenAPI 3.1 y este encargo no añade dependencias. Queda pendiente, antes de
integrar, validar externamente
`docs/api/contratacion_temporal_alta_interna_v1.yaml` con una herramienta
OpenAPI 3.1 aprobada por el proyecto.

## Límites del corte

- No hay servicio falso, repositorio en memoria ni éxito simulado.
- No se modifica aplicación, dominio, puertos, PostgreSQL ni Bolsa.
- No se registra la ruta ni se toca la composición global.
- La validación funcional autoritativa y el consumo de permisos siguen en el
  caso de uso y la transacción durable.
- La recuperación de un resultado indeterminado pertenece a O2-06.
- La composición real pertenece a O2-07.

**Candidato O2-08A; no registrado, no compuesto, no integrado y no productivo
hasta O2-07/O2-06.**
