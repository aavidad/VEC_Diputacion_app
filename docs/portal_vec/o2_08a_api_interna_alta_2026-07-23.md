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

## Idioma, i18n y neutralidad

El contrato no incorpora mensajes visibles ni decide una localización. Cada
error expone un código público en castellano y una `clave_i18n` estable; el
cliente autorizado resuelve esa clave mediante la autoridad i18n común. Los
nombres propios de HTTP, JSON, OpenAPI, Go y los tipos ya publicados por
aplicación, dominio y puertos se conservan como términos técnicos.

Web, escritorio, CLI y MCP consumen el mismo caso de uso y el mismo JSON. El
manejador no conoce sesión de navegador, certificado, Kerberos ni mecanismo
de credencial concreto. Esas superficies solo pueden preparar el contexto
confiable que O2-07 entregará a la autoridad inyectada.

## Trazabilidad normativa y organizativa

La matriz normativa de contratación temporal exige que cada tarea funcional
declare siete aspectos. Para este corte aislado son:

1. **Normas y requisitos aplicables.** Minimización y protección desde el
   diseño del RGPD; hipótesis de categoría alta y privilegio mínimo del ENS;
   separación de actuaciones y expediente de las Leyes 39/2015 y 40/2015 y
   del ENI; neutralidad de canal, denegación predeterminada e i18n estable de
   las reglas del proyecto. Este adaptador aporta controles técnicos, pero no
   autocertifica cumplimiento.
2. **Datos y documentos tratados.** El cuerpo contiene las referencias y
   datos funcionales de `domain.SolicitudCentro`, incluidos detalle y
   observaciones. El contacto y los documentos viajan solo por referencias
   opacas; no se recibe contenido documental. El manejador no registra ni
   refleja el cuerpo.
3. **Finalidad y autoridad.** La única finalidad es solicitar el alta inicial
   de un expediente de contratación temporal. El contexto de autenticación,
   perfil, organización e idempotencia procede de la autoridad confiable del
   servidor; el caso de uso y la transacción conservan la autoridad sobre la
   autorización y el efecto.
4. **Control preventivo.** DTO cerrado, lista positiva, límites previos,
   rechazo de autoridad aportada por el cliente, cancelación propagada,
   recibo completo validado y respuesta minimizada. La indisponibilidad y el
   resultado indeterminado nunca se convierten en autorización o éxito.
5. **Evidencia y pruebas.** `alta_test.go`, `alta_seguridad_test.go` y
   `alta_limites_test.go` cubren contrato, límites, inyección de autoridad,
   errores, recibos, cancelación y carrera. El OpenAPI tiene comprobación
   estructural automatizada y la entrega requiere revisión independiente de
   seguridad y contrato.
6. **Conservación y acceso.** HTTP no conserva estado, no emite cookies y
   ordena `no-store`. La conservación durable pertenece a O2-05/O2-06 y a la
   política documental aprobada. Este corte no ofrece consulta, descarga ni
   acceso posterior.
7. **Responsables de validación.** Dirección e integración deben verificar el
   contrato técnico; RRHH, Sistemas, Seguridad, DPD, Jurídico y Archivo deben
   validar sus ámbitos antes de datos reales, efectos administrativos o
   producción. La validación semántica externa de OpenAPI también permanece
   como puerta previa a integración.

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
