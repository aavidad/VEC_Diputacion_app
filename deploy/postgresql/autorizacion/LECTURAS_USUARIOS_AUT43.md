# Consulta administrativa de usuarios — AUT43

AUT43 lista y consulta las Personas vigentes en CA que tienen asignaciones
registradas para la organización y unidad autorizadas. Incluye asignaciones
caducadas y revocadas; no promete Personas nunca asignadas ni una consulta
histórica de Persona. No devuelve nombres, cuentas, etiquetas, actos ni historia.

Las acciones propias son `administracion.usuarios.listar` y
`administracion.usuarios.consultar`, módulo `administracion`, finalidad
`gestion_usuarios`, garantía `alto` y obligación `auditar`. Exigen Aplicación v5
publicada, asignación y controles vigentes, catálogo propio de AUT42 y ámbito
exacto. Sistemas y Aplicación v4 no habilitan estas operaciones.

## Puertos

Las fachadas `listar_usuarios_admin_v1` y `consultar_usuario_admin_v1` reciben
`material text, capacidad bytea, decision bytea, motivo bytea, contexto bytea,
persona_version numeric, perfil_version numeric, payload bytea, sobre bytea,
evidencia bytea, raiz bytea`. Devuelven `{datos,consumo}` con el acuse real V3
de siete campos. Sólo el grupo nuevo `vec_admin_usuarios_lector` tiene EXECUTE;
no se crea LOGIN ni se concede su pertenencia por petición.

`validar_administrador_usuarios_v1(decision jsonb,material jsonb)` es el gate
propietario para AD185, antes y después del consumo.
`canon_lectura_usuarios_admin_v1(jsonb)` y
`recurso_lectura_usuarios_admin_v1(text)` son fachadas puras de formato para AD.
AD185 usa `registrar_y_consumir_usuarios_admin_v3_atestada` con los mismos once
argumentos y el acuse V3 existente; su código pertenece a dirección.

## Material y salida

Ambas entradas llevan `esquema`, `organizacion_ref`, `unidad_ref` y `conjunto_ref`,
en ese orden. La lista añade `filtros` con `perfil_ref`, `unidad_ref`, `estado`,
seguido de `cursor` y `limite:50`. La ficha añade sólo `persona_ref`. No se omiten
campos vacíos ni se admiten campos ajenos o null. `perfil_ref` filtra una versión
de rol exacta, como el filtro existente de la API. `estado` admite vacío,
`vigente` o `caducado`; sin filtro aparecen también `revocado` y `pendiente`.

El conjunto usa SHA256 de UTF8 `vec.admin.conjunto-usuarios.v1\n` seguido del
JSON Go compacto `{organizacion_ref,unidad_ref}` y los primeros 32 hexadecimales,
con prefijo `conjunto_admin:`. El contexto firmado contiene los maps Go
`ambitos` y `atributos.material_sha256`, ligando todo el material.

Cada persona devuelve sólo `persona_ref`, `unidad_ref`, `denominacion_version`
y `perfiles`. Cada perfil contiene exactamente `perfil_ref`, `rol_version_ref`,
`version` (CA), `estado`, `vigente_desde`, `vigente_hasta`. Estado y ventana son
los de la asignación AUT; la versión del perfil procede del puerto CA34.
La versión de nombre procede del puerto CA32: NULL sólo por ausencia comprobada,
un fallo se propaga como error. El nombre requiere otra lectura V3 propia.

Los filtros de ámbito, rol y estado se aplican sobre la misma asignación actual,
antes de DISTINCT, orden por Persona con colación C y límite 51. Ese límite
se aplica a Personas, no a sus perfiles: se devuelve el conjunto completo de
perfiles del ámbito, sin truncarlo ni introducir un máximo adicional. La vigencia de
Persona se comprueba también antes de paginar por CA34. AUT no pagina CA primero
ni consulta tablas CA. El cursor `usuarios:<SHA64>:<persona>` liga conjunto y
filtros mediante `vec.admin.cursor-usuarios.v1\n<conjunto>\n<JSON filtros>`; no
otorga permisos y cada página necesita capacidad y consumo nuevos.

Consultar Persona ausente o ajena al conjunto devuelve `datos:null` tras consumir
y auditar la consulta; la respuesta no distingue ambos casos. La lista vacía
conserva igualmente consumo nominal. Un replay de capacidad no habilita lectura.
Después de proyectar se revalidan actor, perfil y vigencia; el consumidor Go
debe comprobar el acuse y confirmar COMMIT antes de responder. Los intentos
denegados/error tras rollback requieren el registrador común en su composición.

## Dependencias y comprobación

Orden estructural: AUT42 y CA32, CA34, AUT43 (gate y lector), AD185. El grupo
no se habilita hasta cerrar ese árbol. No se copian ni reaplican SQL ajenas.
CA34 y AUT43 llevan listas separadas de sus únicas migraciones nuevas.
AUT47 concede al grupo `CONNECT` sobre la base actual, que AUT43 no daba; sin él
el LOGIN lector no entra donde PUBLIC no tiene CONNECT. Sin DOWN.
Los vectores de bytes y SHA se calcularon independientemente; el ensayo SQL
comprueba formato, vínculo de filtros/cursor y cierre de helpers.
`aut43_persona_51_perfiles.sql` prepara una Persona sintética con 51 perfiles
y asignaciones ordinarias en un mismo ámbito, mediante INSERT normales en las
autoridades CA y AUT, y coteja la proyección privada de lista y ficha. Se
ejecuta sólo en clon y ROLLBACK: no fabrica decisión, sesión ni V3 favorable.

Estado: candidato preparado, sin ensayo PostgreSQL ni revisión independiente.
Faltan positivos V3 reales, negativas, página mayor de 50 y filtro cruzado entre
asignaciones; dirección conserva el único turno de PostgreSQL. No se ha
instalado nada en la principal ni se acredita todavía pantalla o lectura usable.
