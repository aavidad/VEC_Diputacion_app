# Gobierno técnico de usuarios ADMIN (AD188)

La preparación reutiliza el cargador de material HMAC, derivador y firmante
COSE existentes. `bootstrap.PrepararMaterialUsuariosAdmin` exige una raíz pública
fijada y exactamente dos audiencias de usuarios, con claves distintas. No crea
material maestro, raíces, sesiones, perfiles ni decisiones. Devuelve material
privado en memoria y lo borra al cerrar. El JSON habitual oculta los secretos;
`EscribirMaterialPrivado` sirve exclusivamente al mantenimiento privado.

`AplicarGobiernoUsuariosAdmin` consume plan aprobado, material original y un pool
LOGIN técnico separado. Sólo devuelve el acuse tras COMMIT. Un error de conexión
o COMMIT indeterminado requiere recuperar el mismo plan, nunca regenerar claves.

AD188 instala estructura privada y funciones, sin publicación favorable. El DBA
provisiona fuera de Git el LOGIN del grupo técnico y la configuración aprobada:
SHA de bytes originales de plan y material, preimagen JSONB::text y ventana.
Plan V1: `version`, `operacion_ref` gcu_, `preparado_en`, `caduca_en`,
`preimagen_sha256`, `configuracion` (revision/secuencia/huella_sha256/
publicada_en/expira_en) y dos `clave_ordenes` CAS.
Material: `claves` en orden listar/consultar con las diez coordenadas que entrega
el proveedor. Contiene dos secretos HMAC ya derivados; nunca se imprime ni se
incluye en el plan, auditoría, documentación o Git.

La operación valida LOGIN/configuración, SHA externos, CAS y vigencia, publica
dos claves y punteros propios y añade una configuración diaria nueva ligada a la
misma raíz. Las filas anteriores permanecen intactas. La confirmación técnica y
cada intento se registran en auditoría común en la misma transacción. El efecto
usa una subtransacción: denegación/error revierte el efecto y conserva el intento;
si falla el registro común, todo debe revertirse. El replay no repone claves
revocadas ni renueva ventanas.

La corrección de coexistencia selecciona la última clave CT dentro de su catálogo
cerrado antes de validar actos y material. Conserva la selección global de
configuración/raíz y sus controles. ADMIN tiene actos propios; no se añade a CT
para eludir su comprobación. La renovación diaria sigue la autoridad y protocolo
existentes CT y queda expresamente ligada al plan técnico aprobado.

Orden medido: POST185 -> AD186 -> AD187 -> AD188. El clon K conserva AUD6250,
políticas/preservaciones vacías y frío anterior a186. CHECK de familias posterior
a187: `97d754fef3d60b4799f82fac0d09417133f4caa5b25e8f2bcba005fb1f72a09f`.
No se ha instalado188. La instalación requiere dos revisiones del hash final.

Como referencia pública, Keycloak distingue claves activas y pasivas
([Realm keys](https://www.keycloak.org/docs/26.8.0/server_admin/)); Vault Transit
versiona claves y conserva el descifrado histórico
([Transit](https://developer.hashicorp.com/vault/docs/secrets/transit)); AWS KMS
conserva material anterior y registra la rotación en CloudTrail
([Rotation](https://docs.aws.amazon.com/kms/latest/developerguide/rotate-keys.html));
Authentik importa parejas certificado/clave
([Certificates](https://docs.goauthentik.io/sys-mgmt/certificates)). El encaje VEC
es una inferencia: conservar historia/SPKI, renovar por adición y reutilizar el
KMS. Ninguna ventana o regla externa concede permisos automáticamente.

Pendiente: revisiones SQL/seguridad, ensayo positivo con proveedor real,
replay/reinicio, rollback obligatorio de auditoría y regresión CT. Las pruebas
focales de preparación/verificación no acreditan ese recorrido ni producción.

## Ensayo terminado y límite actual

AD188 de `8d213c010` se instaló una sola vez, después de dos GO. Los vectores
SQL de estructura/ACL/formato terminaron0. Un LOGIN técnico real registró cinco
intentos denegados: configuración ausente, CONNECT/USAGE/EXECUTE con facultad de
delegación y plan incompleto. Auditoría común6250→6255, cero claves nuevas. Las
ACL se restauraron después de cada caso. Los tres ejercicios de ACL usaron un
plan inválido: acreditan denegación y recibo real; no aíslan su causa frente al
rechazo del plan. El verificador recalculó los cinco materiales y eslabones reales.

Asignaciones, punteros, Persona/perfil CA, Rol5 y cabeza histórica6248 conservaron
sus recuentos y huellas. No se ampliaron vigencias de los administradores caducados.
El reinicio se comprueba sobre esos mismos cinco recibos, sin otro intento.
Actas privadas en `ensayo188/` del estado K; ninguna contiene claves.

La publicación favorable, su replay y la regresión CT tras dos claves ADMIN siguen
pendientes. El material HMAC del fixture de fuentes deriva SPKI
`e7a8529fd7e7959640a03bcfe325d65214b3943980f6e9227b22d22dd95ab89e`;
el gobierno requiere `9c11fc59ca30e845be2684b21a045d4882c77544e7d4d4d73f7f1ccdbbeb7f12`.
Son hashes públicos de DER. El candidato D6 también fue rechazado. El preparador
paró antes de escribir. Dirección respondió a la pregunta11:31 el 04/10 a las
11:34: el firmante principal9c11 vive sólo allí y no se copia a local. Autoriza
un firmante DEV propio exclusivamente en el clon desechable, por el circuito de
gobierno existente, con cambio documentado. No se ha ejecutado aún ese productor.
La entrega queda en borrador hasta cerrar el recorrido positivo del clon.

La API exige ahora `ArchivoSemillaRaiz` explícito y utiliza
`NuevoFirmanteAtestacionV3DesdeArchivo`, cotejando su pública fijada. El material
HMAC puede proceder de otro proveedor; nunca selecciona el firmante por defecto.
La semilla se lee del archivo privado existente y el cierre invalida el firmante.
No se copió ninguna clave de la principal. La prueba de fuente HMAC independiente
y la negativa de archivo ausente pasan con race.

El test PostgreSQL es opt-in mediante `VEC_GOBIERNO_USUARIOS_ENSAYO_CONFIG`:
JSON0600 fuera de Git, con fase/rutas/DSN privados. Normalmente se omite. Preparar
sólo escribe plan/material privados; aplicar/replay exige aprobación externa y el
LOGIN configurado; verificar lee la cadena técnica real. No crea perfiles, fuentes
ni sesiones. El kit no modifica las filas históricas para obtener un positivo.

## Comprobación focal de código y seguridad

Normal y race de `bootstrap`/`auditoria`, vet y diff verdes. Tras corregir los
acuses, la tanda focal race pasó de nuevo. Gopls resolvió el derivador existente.
Semgrep local: siete reglas, nueve archivos y cero hallazgos. El primer uso de la
regla AD183 de verificador sobre pruebas del preparador señaló dos WriteFile:
son los archivos0600 del fixture, no escrituras del verificador. AD188 limita esa
regla al paquete de auditoría y conserva las demás sin excluir código operativo.

Gosec se ejecutó sólo sobre los dos paquetes tocados. Sus 17 avisos están en
archivos heredados, sin cambios en esta rama. No señaló los archivos nuevos.
El cargador además reportó metadatos incompletos de dependencias aun con Go1.26:
no se presenta esta pasada como cobertura completa ni como resultado verde.
No se repitió una campaña global para ocultar ese límite.

- G101: `dietas_postgresql.go:241` es una consulta de acreditación, sin contraseña.
- G304: `material_desarrollo.go:446,680`, `fake_credentials.go:73`,
  `contratacion_temporal_subsanacion_politica_desarrollo.go:131`,
  `contratacion_temporal_propuesta_publicaciones_desarrollo.go:22`,
  `contratacion_temporal_centros_anteriores.go:41`, `catalogo_rpt_desarrollo.go:40`
  y `bolsa_importacion_convoca_custodia.go:53,73`: rutas existentes de configuración,
  catálogo o custodia, sin entrada HTTP nueva. El proveedor reutiliza la lectura
  privada/acotada del cargador; no amplía a peticiones la selección de ruta.
- G302: `portal_externo_material_v3.go:222` aplica0700 a directorios privados;
  el bit de recorrido del directorio es necesario, los archivos siguen0600.
- G104: `portal_externo_material_v3.go:294,298`,
  `contratacion_temporal_incorporacion_configuracion.go:92` y
  `contratacion_temporal_comunicacion_llamamiento_desarrollo.go:534,539,540`:
  cierres de limpieza en ramas ya fallidas. No convierten el fallo previo en éxito.

No se añadieron supresiones ni se cambiaron estos componentes ajenos. Las dos
revisiones de seguridad/SQL acreditan el código y sus límites, no una autorización
favorable ni producción.
