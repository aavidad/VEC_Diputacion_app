# AUT48: versión vigente del rol de Aplicación

## Problema

El perfil fijo de Aplicación (`rol:administracion_perfiles`) cambia de versión
cada vez que se le añade una concesión: v5 con AUT42 (usuarios y denominación)
y v6 con AUT45 (lote ordinario). Cuatro comprobaciones de PostgreSQL llevaban el
número de versión escrito en el código:

| Comprobación | Quién la usa |
| --- | --- |
| `validar_administrador_usuarios_v1` | lista y ficha de usuarios (AD185) |
| `validar_administrador_denominacion_persona_v1` | denominación de Persona (AD184) |
| `acreditar_perfil_aplicacion_nominal_v1` | capacidades, certificados y cargo competencial |
| `acreditar_ambito_certificado_nominal_v1` | ámbito de certificados nominales |

AUT45 resolvió la v6 copiando las cuatro funciones de la v5 y añadiendo una
rama en cada despachador. Una v7, por ejemplo con la exportación de auditoría,
habría dejado las cuatro operaciones sin acreditar hasta repetir esa copia.
Además, el emisor de vec-admin y el adaptador Go de usuarios aceptaban sólo
«v5 o v6».

En el clon con vec-admin real, la lectura de usuarios ya funcionaba en v6 tras
AUT45; el aviso del runbook («con v6 deja de acreditar») era de antes de AUT45.
Lo que sí fallaba con seguridad era cualquier versión posterior.

## Diseño

La versión aceptada es la que señala la **asignación actual** del perfil de la
persona. No hay «>= vN». Para que una decisión pase, en esa versión tienen que
cumplirse a la vez:

- el rol es `administracion_perfiles`, la referencia es `rol:administracion_perfiles:v<N>`
  con el mismo N que la fila, está publicado y su huella canónica coincide con la
  de la decisión;
- su control de vigencia actual está `habilitada` y su huella coincide;
- sus metadatos de perfil fijo son de Aplicación (`aplicacion`, `fijo_sistema`)
  y apuntan a esa versión y a esa huella;
- su entrada de catálogo para la acción existe, es de esa versión y esa huella,
  y su concesión es exactamente la esperada;
- la misma concesión está dentro del documento del rol;
- la asignación está activa, en plazo y cubre el ámbito pedido.

Sin la concesión exacta en esa versión, se deniega. Una versión antigua tampoco
sirve aunque siga habilitada, porque la asignación actual ya no la señala.

Cada función genérica es el cuerpo de la v6 de AUT45 cambiando sólo cuatro
cosas: la versión fija por la de la decisión, `r.version=6` por la coherencia
entre referencia y número, `fuente_version=6` por `r.version`, y se quita el
número de versión de la entrada de catálogo. Ese número no aporta nada porque
`(version_rol_ref, accion_ref)` es única y se comprueban su fuente y su huella.
La v4 conserva su función preservada. Las funciones por versión de AUT42 y AUT45
siguen instaladas, sin uso.

En Go, el emisor y el adaptador aceptan cualquier `rol:administracion_perfiles:v<N>`
bien formada; la concesión exacta la sigue exigiendo el emisor y después
PostgreSQL.

## Comparación con otros sistemas (sólo conceptos)

- **Keycloak**: los roles no tienen versión; el token lleva nombres de rol y la
  aplicación comprueba el permiso, no la versión. Aquí se comprueba también el
  permiso (la concesión), pero se conserva la versión porque el registro de
  autorización es histórico y cada decisión queda ligada a la huella exacta.
- **Zanzibar / SpiceDB**: la decisión lleva un testigo de consistencia («zookie»)
  y la comprobación exige un estado al menos así de reciente. Aquí la decisión
  lleva la versión y la huella del rol; la comprobación exige que sea la que
  señala la asignación actual.
- **Cedar / OPA**: las políticas se versionan en un almacén y se evalúa siempre
  la vigente; el registro de decisiones guarda la revisión. Es lo mismo que hace
  ahora VEC: versión vigente para decidir, versión y huella para auditar.

Ninguno de ellos fija la versión del rol en el código de la aplicación. Esa era
la causa raíz.

## Decisiones que deben validar Alberto o K

1. La versión aceptada se determina por la asignación actual, no por «la última
   versión publicada del rol». Con fechas prospectivas (AUT46) puede haber una
   versión nueva publicada antes de que empiecen sus asignaciones; así la lectura
   sigue con la versión que la persona tiene asignada en ese momento.
2. Se quita la comprobación del número de versión de la entrada de catálogo.
3. Las v5 y v6 pasan también por la comprobación genérica, para que el ensayo con
   vec-admin real pruebe ese camino. Las funciones de AUT42 y AUT45 quedan sin uso.

## Lo que queda fuera

- `acreditar_perfil_aplicacion_lote_ordinario_v1` (AUT45) sigue fijada a v6:
  una v7 dejaría sin acreditar el lote ordinario hasta generalizarla igual. La
  siguiente minitarea, que crea la v7, debe incluirla.
- La comprobación de ámbito de certificados exige ahora además metadatos de
  perfil fijo de Aplicación para la versión (v4 en adelante); antes, en v6, sólo
  miraba la asignación.
- No hay DOWN. Para volver atrás habría que reinstalar los despachadores de
  AUT45; las funciones por versión de AUT42 y AUT45 siguen instaladas.

## Instalación

`deploy/principal/lista_sql_claude_aut48_version_rol_20261005.txt`, una vez y
sin DOWN, después de AUT45. Se puede aplicar con vec-admin en marcha.
