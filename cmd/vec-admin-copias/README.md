# API de copias en ADMIN

Este corte prepara un manejador inyectable. No añade otro servidor, perfil ni sesión.
El punto de composición es `administracion.NuevoHandlerCopias`. Debe montarse después
de la comprobación vigente de CA, CRL, red y sesión del listener ADMIN existente.

La sesión procede del resolvedor ADMIN. Cada operación consulta de nuevo la autoridad
central con actor, perfil, acción y recurso exactos. Los adaptadores de negocio deben
revalidar y consumir esa autoridad junto al efecto, auditoría y recibo. Un callback
que devuelve `nil` no demuestra consumo V3. Las capacidades solo orientan la vista.

La raíz de la API es `/api/admin/copias/v1`:

| Método y sufijo | Entrada o salida |
| --- | --- |
| GET vacío | `cursor` y `limite` opcionales; página de copias y versión actual de la fuente. |
| GET `/capacidades` | Capacidades nominales; no contiene identidad ni concesiones. |
| GET `/{copia_ref}` | Estado, compatibilidad y metadatos del conjunto. |
| GET `/opciones-restauracion` | Destinos, motivos y ventanas de configuración, por referencia y clave i18n. |
| GET `/calendario` o `/retencion` | `{configuracion:{version,politica}}`. |
| GET `/propuestas` | Propuestas y datos necesarios para revisarlas. |
| POST `/lanzamientos` | `operacion_ref`, `version_esperada`, `tipo: "completa"`; devuelve recibo. |
| POST `/calendario` o `/retencion` | `operacion_ref`, `version_esperada` y política completa; devuelve recibo. |
| POST `/propuestas` | Referencias de conjunto, destino, motivo y ventana, operación y fechas UTC; devuelve propuesta. |
| POST `/propuestas/{ref}/revision` o `/ejecucion` | Operación, destino, huella de propuesta y versión esperada. |

Los modelos completos están en `internal/modules/administracion/ports/httpcopias`.
Calendario y retención comparten una política atómica. El adaptador de configuración
debe exigir las concesiones de todos los bloques que cambien; el permiso de un
bloque no permite modificar el otro. La versión de lanzamiento procede de su fuente
real y el adaptador la contrasta antes de reservar. No se deduce del número de filas
ni se reemplaza por una constante.

Las propuestas muestran la preimagen y las políticas selladas. La copia previa se
exige antes de sustituir; no se presenta como ya hecha al proponer. Los intervalos de
pérdida solo aparecen cuando una fuente los acredita. El servicio CS10 comprueba el
cercado, pero no ejecuta sustituciones. Sin ejecutor real, `/ejecucion` devuelve 503.

Solo se acepta JSON de hasta 16 KiB, con campos requeridos explícitos y sin claves
duplicadas, campos desconocidos, valores nulos ni documentos adicionales. Los POST
exigen origen y contexto de navegación del mismo sitio. Se rechazan cookies,
Authorization y cabeceras libres de identidad. Las respuestas no generan cookies,
redirecciones ni CORS y llevan `no-store`. Los fallos contienen código y clave i18n;
no incluyen certificados, secretos, rutas o causas internas.

La comprobación focal usa `go test -race` y `go vet` en los paquetes afectados.
Una prueba HTTP con mTLS local verifica denegación sin sesión o permiso, rechazo de
entradas falsificadas, autorización nominal y retirada de permiso en una conexión.
Son fixtures sintéticos: no acreditan montaje, asignaciones administrativas,
consumo V3, persistencia de una copia ni un recorrido de navegador.

La revisión exige `propuesta.metadatos_revision` con fecha de copia, comienzo de la
pérdida de cambios, versiones observadas de aplicación/PostgreSQL/esquema en el
destino y en el conjunto resultante, huellas de sus descriptores y razones de
compatibilidad. Estos datos se vinculan a la propuesta, conjunto, destino y
preimagen por referencias y SHA256. Proceden del manifiesto autenticado y de la
observación actual del destino; no se rellenan con fechas o versiones por defecto.

La composición inyecta `FuenteRevision.PropuestaParaRevision`. Cada revisión vuelve
a consultar esa fuente, exige permiso de lectura, contrasta versión y sello y
comprueba que la información permita decidir. La ventana y la caducidad siguen
siendo las de la propuesta sellada. Sin fuente, con datos desconocidos o con
compatibilidad sin acreditar, la capacidad de revisar queda desactivada y el POST
se deniega antes de llamar al control. Los consumidores anteriores pueden seguir
mostrando historia sin esos metadatos, pero no aprobarla. El cuerpo del POST sigue
limitado a operación, destino, sello y versión; rechaza metadatos aportados por el
cliente. El control conserva la revalidación de preimagen y autoridad en su efecto.

## Montaje en la frontera existente

`administracion.NuevoServidorCopias(cfg, dependencias, publicos, auditorLecturas)`
monta `/admin/copias/` y `/api/admin/copias/v1` dentro de la verificación F de red,
CA y CRL. `NuevoServidor` conserva su comportamiento de prueba de vida. No se
crea otro portal ni se sirve el árbol web completo: los recursos del módulo,
sus dos catálogos y el tema común tienen una lista positiva exacta.

La sesión y el permiso `copias_consultar` se comprueban también al cargar los
recursos de pantalla. Los accesos usan la auditoría central; una denegación
exterior que no pueda registrarse responde con servicio no disponible.
La configuración `config.ejemplo.json` contiene solo rutas sintéticas y
referencias. No configura identidad, claves, roles ni permisos.

`AbrirLecturasCopias` abre el diario CS07 y la política versionada en sus
ubicaciones privadas fuera del conjunto restaurable. Recibe el destino CS03
ya construido por su propietario con KMS y catálogo. La lectura recupera y
autentica el conjunto completo, coteja conjunto y operación del diario y
minimiza la respuesta. El estado declarado en CS07 no acredita verificación.
La compatibilidad se calcula con CS01 solo cuando la fuente de instalación
actual aporta la observación y la preimagen; en otro caso queda no comprobable.
El detalle busca el conjunto en un máximo de 64 páginas del diario. Superado
ese límite se bloquea; no se afirma que la copia no exista.

## Dependencias que siguen pendientes

No hay un arranque operativo de `vec-admin-copias` ni una sesión alternativa.
El proveedor central debe implementar la resolución ADMIN de F, incluyendo
vínculo nominal certificado/cuenta privilegiada, sesión y perfil vigentes.
La fachada del portal interno no sirve: tiene otra superficie de confianza.
La gestión de perfiles no concede competencias de copias.

La materialización V3 de CS08 requiere el consumidor nominal AD143, reservado
por su propietario, y el SQL de `administracion_copias` todavía borrador. El
contrato que debe implementar la autoridad central es:

```go
ComprometerOrden(context.Context, ordenescopias.Orden,
    domain.SolicitudAutorizacionLigadaV3,
    domain.DecisionAutorizacionLigadaV3,
    ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) error
```

Debe revalidar persona, sesión, perfil, política, acción, recurso, campos y
obligaciones; consumir la concesión exacta con unicidad y CAS; y conservar
orden, auditoría y outbox en la misma transacción. Registrar una candidata
V3 no equivale a consumirla. No se genera SQL ni se ejecuta una migración en
este montaje.

El anclaje externo también debe implementar
`ValidarActual(context.Context, ordenescopias.Orden, time.Time) error`, con
revocación, época y cercado actuales fuera del rollback. Durante mantenimiento
no basta la sesión histórica ni una firma previa. La composición no habilita
lanzamiento, cambios de calendario/retención, propuesta, revisión o ejecución
sin sus proveedores reales. Las propuestas y los metadatos de revisión
siguen bloqueados hasta disponer del registro y la observación actual
correspondientes; no se devuelven listas vacías inventadas.

Las pruebas locales comprueban el montaje, la segregación de recursos, red y
revocación, retirada de sesión/permiso, fallo de auditoría y lectura minimizada
con diario real y manifiesto sintético. No acreditan provisión central,
instalación SQL, copias reales, despliegue ni recorrido de navegador.

El listado conserva además las reservas y capturas en progreso del diario,
con su fecha de reserva original y compatibilidad no comprobable. Estas filas
no acreditan publicación: omiten huella, release, tamaño y fecha de fin. Un
abandono previo a la publicación se muestra como fallido solo cuando la
historia conserva el motivo de captura admitido y no confirma publicación.
Las fases publicadas y los abandonos por fallo de verificación deben recuperar
el conjunto autenticado; un error de integridad no se degrada a progreso.
Cada conjunto incluido tiene su propio recibo de acceso, además del listado;
un fallo de auditoría bloquea la página completa.

Una política todavía vacía o inválida queda no disponible. La respuesta no
presenta una versión cero como configuración válida. El ensayo del grafo lee
el HTML y sus dependencias del árbol real a través del manejador ADMIN; no
sustituye los estilos comunes por recursos de prueba.
