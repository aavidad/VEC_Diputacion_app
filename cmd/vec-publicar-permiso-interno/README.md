# Permiso temporal de consulta CT para vec-interno

Ejecutar **antes** de arrancar vec-interno, solo por la administración del entorno
de desarrollo. La persona, cuenta y perfil deben estar vinculados y activos en
F1. El perfil es propio del lector interno y aparece en el material privado de
`contextos.json`; no se reutiliza el perfil de vec-server.

`VEC_PERMISO_INTERNO_DSN` contiene el DSN administrativo PostgreSQL y se aporta
fuera de Git. Ejecutar `go run ./cmd/vec-publicar-permiso-interno` con `--cuenta`,
`--persona`, `--perfil`, `--organizacion`, `--politica` y `--politica-huella`,
tomados de los registros gobernados privados. No pasar el DSN por argumento.

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
retirada de la política por petición **antes de llamar al PDP**. El perfil no debe aparecer en la
configuración de ningún otro canal: un canal que reutilizara ese perfil y
presentara otra autenticación sustancial podría invocar la concesión V3.
Todo consumidor adicional necesita un guard equivalente sobre el vínculo F1,
la política de garantía y su retirada.
La política restrictiva V3 actual solo filtra recurso, no el identificador de
política de autenticación, por lo que no cerraría ese cruce. El procedimiento
administrativo verifica que la misma persona conserve otro perfil RRHH de
garantía alta y rechaza cualquier asignación previa distinta en el perfil
lector.

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
