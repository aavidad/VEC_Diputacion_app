# Contratos HTTP documentados

Corte contrastado con `origin/main@3ef7dee0b` el 20/10/2026.
Esta carpeta contiene contratos acotados; no es un inventario completo de rutas.
Los prefijos, errores y recibos comunes están en
[ESQUEMA_BACKEND §11](../producto/ESQUEMA_BACKEND.md#11-api-http).
La composición efectiva, sus selectores y la autorización determinan la
exposición de cada operación. Un contrato no concede acceso ni acredita que
una ruta esté montada en una copia.

| Contrato | Alcance actual | Límite documental |
| --- | --- | --- |
| [Inscripciones de Bolsa](bolsa_inscripciones_v1.md) | Consultas y actos del aspirante y RRHH, con sesión ligada al certificado | Revisar las modalidades y fuentes vigentes antes de implementar un consumidor; el documento no sustituye las pruebas HTTP ni el catálogo publicado |
| [Alta interna de Contratación temporal, versión 1](contratacion_temporal_alta_interna_v1.yaml) | Sobre legado de `POST /api/vec/contratacion-temporal/solicitudes`, aún aceptado por el manejador | No describe las variantes posteriores basadas en una necesidad catalogada |
| [Alta con necesidad, especificación versión 3](contratacion_temporal_alta_necesidad_v3.yaml) | Misma ruta, con necesidad del centro y puesto publicado cuando corresponde | El cuerpo documentado declara `vec.ct.alta_necesidad.v1`; el manejador también acepta el sobre `vec.ct.alta_necesidad.v2`, fuera de esta especificación |

## Fuentes para comprobar un contrato

- Alta de Contratación temporal:
  `internal/modules/contrataciontemporal/adapters/httpinterno/alta.go`,
  `alta_contrato.go` y `internal/modules/contrataciontemporal/ports/necesidades_alta.go`.
  Las rutas se montan en
  `internal/app/composicion/interna/contrataciontemporal/rutas.go`.
- Inscripciones: `internal/app/bootstrap/bolsa_inscripcion_composicion.go`,
  sus pruebas y los adaptadores HTTP de Bolsa. El portal externo y el proceso
  interno tienen composiciones e identidades distintas.
- Clientes reales: `web/static/portal-empleado/` y `web/static/area-personal/`.
  Las respuestas y errores se traducen con los catálogos de `web/static/textos/`.

Los actos se prueban con rechazo de autoridad ajena, versión vieja, repetición
y clave reutilizada con contenido distinto. Las lecturas de datos personales
usan sesión ligada al certificado y constancia de auditoría. Ni el actor ni
el perfil llegan como autoridad desde un cuerpo o una cabecera del navegador.

## Diferencias conocidas con las reglas de arranque

La regla vigente exige que una función opcional mal configurada falle cerrada
sin tumbar la aplicación. En el corte consultado, activar Inscripciones sin
sus dependencias obligatorias aún puede devolver un error de composición
que impide arrancar el proceso. Está señalado en el contrato; no se documenta
como un aislamiento ya conseguido. La corrección requiere código y pruebas
específicas en otro encargo.

Las variantes aceptadas pero no recogidas por un esquema se contrastan con el
manejador y sus pruebas antes de usarlas. No se retira un contrato legado
mientras tenga un consumidor admitido.
