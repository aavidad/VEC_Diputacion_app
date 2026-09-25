# Permiso temporal de consulta CT para vec-interno

**Condición de uso:** instalar primero AUTH-14 y conservar el cotejo F1 exacto
de referencia, huella y retirada de la política antes del PDP de vec-interno.
AUTH-14 limita la lectura de esta versión de rol al LOGIN nominal de la fuente
de vec-interno; el PDP V3 genérico compara el nivel `sustancial` y por sí mismo
no coteja la política de garantía. La comprobación F1 de la CLI es una
precondición de aprovisionamiento; AUTH-14 no acredita por sí sola que las
funciones de registro y consumo revaliden esa política.

Ejecutar **antes** de arrancar vec-interno, solo por la administración del entorno
de desarrollo. La persona, cuenta y perfil deben estar vinculados y activos en
F1. El perfil es propio del lector interno y aparece en el material privado de
`contextos.json`; no se reutiliza el perfil de vec-server.

`VEC_PERMISO_INTERNO_DSN` contiene el DSN administrativo PostgreSQL y se aporta
fuera de Git. Ejecutar `go run ./cmd/vec-publicar-permiso-interno` con `--cuenta`,
`--persona`, `--perfil`, `--organizacion`, `--organizacion-corporativa`,
`--politica` y `--politica-huella`, tomados de los registros gobernados
privados. No pasar el DSN por argumento. `--organizacion` es el ámbito CT del
permiso (el mismo que el selector nominal de vec-interno, p. ej.
`organizacion:…`); `--organizacion-corporativa` es la referencia `org_…` del
vínculo corporativo de ContextoActor. No son intercambiables: el cotejo F1 usa
la segunda y el ámbito V3 la primera.

La herramienta coteja la política activa de Identidad, el vínculo F1 de
cuenta/persona/perfil y el vínculo corporativo vigente de esa misma cuenta con
la organización solicitada, la superficie interna y el uso `consulta_rrhh`,
dentro de una transacción serializable. Publica una asignación V3 estable para la consulta
de detalle CT, con garantía sustancial y vencimiento igual al de la política,
como máximo el 31 de octubre de 2026. Un reintento exacto solo coteja la
publicación; una asignación diferente o retirada falla. No concede alta,
lectura Personal ni confirmación de incorporación. vec-interno y vec-server no
publican este permiso al arrancar ni por operación.

La concesión V3 comprueba el nivel `sustancial`; el factor concreto queda
ligado en la frontera F1 de vec-interno, que coteja referencia, huella y
retirada de la política por petición **antes de llamar al PDP**. El perfil no
debe aparecer en la configuración de otro canal. AUTH-14 deniega la lectura
de su rol a otros LOGIN de fuente; todo consumidor adicional requiere un guard
equivalente sobre el vínculo F1, la política de garantía y su retirada.
La política restrictiva V3 actual solo filtra recurso, no el identificador de
política de autenticación, por lo que no cerraría ese cruce. El procedimiento
administrativo verifica que la misma persona conserve otro perfil RRHH de
garantía alta y rechaza cualquier asignación previa distinta en el perfil
lector.

**Condición transitoria observada en el clon (25/09/2026).** Esa comprobación
lee la asignación *actual* (`asignacion_perfil_actual`) del otro perfil de la
persona y exige que su versión de rol conceda
`contratacion_temporal.expediente.consultar` con garantía `alto`. En el
perfil RRHH de vec-server esa asignación no es estable: vec-server la reescribe
por operación. Solo contiene el rol `consulta_detalle_rrhh_desarrollo`
inmediatamente después de una consulta de detalle CT de esa persona en
vec-server; tras arrancar vec-server u otra operación vuelve a la asignación
de técnico RRHH y la herramienta termina en «publicación denegada». Por tanto:

1. con vec-server en marcha, abrir como esa persona el detalle de un
   expediente CT (consulta de detalle RRHH);
2. ejecutar inmediatamente la herramienta, sin otra operación ni reinicio de
   vec-server entre medias;
3. si falla, repetir desde el paso 1; un reintento exacto posterior solo
   coteja la publicación ya hecha.

La herramienta no relaja la comprobación: falla cerrada si el estado no se da.

La lista `campos_permitidos` del rol enumera los campos de seguimiento
autorizados. La consulta CT interna todavía usa el detalle RRHH para verificar
la incorporación original; la proyección HTTP de seguimiento es la frontera
que limita los campos entregados al navegador. Esa lista V3 por sí sola no
filtra el resultado SQL del detalle.

Prueba local: `testdata/pg18_esquema.sql` crea una preimagen sintética en una
base PostgreSQL 18 desechable; se ejecuta la CLI dos veces con las mismas
referencias y después `testdata/pg18_casos.sql`. Se ensayan también persona,
perfil, asignación y política divergentes y una política caducada. Este ensayo
comprueba publicación y replay SQL, pero no sustituye RLS/ACL de la migración
canónica ni acredita una decisión PDP V3 sobre el recorrido completo.
