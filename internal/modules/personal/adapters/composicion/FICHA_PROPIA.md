# Auditoría de Mi ficha

La consulta propia conserva su autorización, lectura y recibo en la misma
transacción serializable. Denegaciones y errores posteriores a una identidad
registrada se anotan mediante `RegistradorIntentosAuditoria`, después de cerrar
la transacción de Personal. No se escribe en el antiguo registro de denegaciones.
Sus tablas e historia instalada se conservan.

La frontera captura una copia del contexto, vínculo, perfil y correlación antes
de validar la entrada. Un reintento del registro conserva la misma orden y no
vuelve a consultar la ficha. Si el registro no devuelve un acuse válido, la
respuesta es indisponibilidad y no contiene datos personales.

## Montaje interno

`identidad/personal-empleado.json` pasa a versión 2. Retirar
`dsn_personal_frontera`; conservar las siete conexiones nominales restantes.
Añadir `limite_auditoria_segundos`, entre 1 y 30, y `intentos_auditoria`:

| Campo | Fuente exigida |
| --- | --- |
| `proceso` | El proceso gobernado de `auditoria-intentos.json`. |
| `canal` | Superficie interna corporativa. |
| `recurso_entrada_invalida` | Referencia técnica opaca configurada, sin contenido de la petición. |
| `motivo_denegado` | Referencia vigente del catálogo común de motivos. |
| `motivo_entrada_invalida` | Referencia vigente del catálogo común de motivos. |
| `motivo_no_disponible` | Referencia vigente del catálogo común de motivos. |

El archivo común `auditoria-intentos.json` selecciona una cuenta dedicada,
proceso, canal y límite. El montaje coteja base y cuentas segregadas, exige el
preflight común y rechaza materiales antiguos o incompletos. No publica
concesiones por petición ni crea cuentas o permisos.

## Alcance pendiente

Cuando la resolución de sesión no entrega contexto registrado, incluido empleado
ausente o ambiguo, no se fabrica una atribución nominal: la ruta permanece
cerrada con respuesta genérica. Los rechazos anteriores a ese contexto requieren
su contrato de auditoría común de frontera; este corte no acredita su cobertura.
Las peticiones sin autenticación y las rutas inexistentes conservan su respuesta
temprana. No se atribuye un perfil a partir de un certificado presentado.

El CSV generado en el navegador sigue siendo una copia local de la consulta.
No constituye una exportación auditada por el servidor. Su cierre necesita una
acción nominal específica, ligada al recibo y al corte exactos; una consulta
nueva cambia el instante de conocimiento y no registra una descarga.

Las pruebas de aplicación, composición y HTTP usan datos y dependencias
sintéticos. No acreditan instalación, PostgreSQL nominal, navegador con sesión
institucional ni despliegue. Dirección integra y habilita el paquete coherente.
