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

Orden ensayado: POST185 → AD186 → AD187 → AD188 → AD189 → AD191. Cada
migración se instaló una sola vez en el clon, tras las dos revisiones de su
contenido exacto. CHECK de familias previo a AD188:
`97d754fef3d60b4799f82fac0d09417133f4caa5b25e8f2bcba005fb1f72a09f`.
AD191 conserva la función de efecto y corrige tres referencias de su acuse;
el archivo AD188 instalado permanece intacto.

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

## Recorrido comprobado en el clon

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

Dirección respondió el 04/10 a las 11:34: el firmante principal permanece en
la principal. Autorizó un firmante DEV propio exclusivamente en el clon, por
el circuito de gobierno existente. Ese setup se ejecutó con plan/configuración
aprobados y CAS, conservando la raíz pública original como historia. No es una
rotación de producción ni una confirmación ficticia AD188. Acta privada:
`lab-root-dev/acta-setup-root-dev.json`.

El primer positivo encontró SQLSTATE `55000`: la variable PL/pgSQL `a` ocultaba
un alias SQL al leer la configuración. AD191 corrigió sólo su declaración y
dos usos, manteniendo firma, OID, propietario, ACL y demás metadata. Después:

- LOGIN técnico real: permitido, dos claves, dos punteros y una confirmación.
- Replay real: mismo recibo, fecha y huella; únicamente un nuevo intento.
- Reinicio de PostgreSQL y otro replay real: mismo recibo, sin otra publicación.
- Plan aprobado con preimagen obsoleta, mediante otro LOGIN técnico: denegado,
  recibo nulo e intento durable.
- Comprobación real del helper CT: reconoce su gobierno tras las dos claves ADMIN.

Auditoría común final: 6262 registros. La confirmación se conserva en 6258;
los intentos de alta, replay, CAS denegado y replay tras reinicio son 6259–6262.
El tramo contiguo 6257–6262 incluye el error anterior y permite recalcular
materiales y eslabones sin omitir la confirmación intermedia. La exportación
opt-in declara únicamente ese tramo, no cobertura global de la auditoría.

Dos diagnósticos adicionales terminaron en ROLLBACK: retirada de la raíz DEV
rechazada con recibo nulo y fallo obligatorio de auditoría durante replay
rechazado con `42501`, sin acuse de éxito. Usaron sesión técnica simulada sólo
para el diagnóstico. El segundo acredita rollback obligatorio sobre replay;
no se presenta como ensayo de fallo durante una primera publicación.

Recuentos y huellas de roles, asignaciones y Persona/perfil CA permanecieron
idénticos antes y después del reinicio. Los perfiles APP vencidos siguen
vencidos: el recorrido técnico de gobierno no acredita acceso nominal actual
ni una sesión HIGH.

Actas privadas: `ensayo191/reinicio-final/acta-reinicio.json`,
`acta-root-retirada-rollback.json`, `acta-auditoria-obligatoria-rollback.json` y
`ensayo188positivo/acuse-aplicar-post191.json`. Copia fría previa a AD191:
`estado-pre191-rootDEV6257-frio.tgz`, SHA256
`e6bd65d9c09b16140eadbaed0081944c234faea3f2e7fd38c36234d2b2ad7d62`.
Copia fría posterior al positivo/CAS, antes del último replay:
`estado-post191-gobierno-real6261-frio.tgz`, SHA256
`38dbbdecf1640e226f71c2a4db3c0ebbd039e52a810bfd810a20c02532c21514`.

El producto prepara y aplica gobierno técnico en el clon. El montaje ADMIN y
los recorridos nominales requieren sus fuentes y vigencias propias. No se ha
desplegado esta entrega en servidores.

La API exige ahora `ArchivoSemillaRaiz` explícito y utiliza
`NuevoFirmanteAtestacionV3DesdeArchivo`, cotejando su pública fijada. El material
HMAC puede proceder de otro proveedor; nunca selecciona el firmante por defecto.
La semilla se lee del archivo privado existente y el cierre invalida el firmante.
No se copió ninguna clave de la principal. La prueba de fuente HMAC independiente
y la negativa de archivo ausente pasan con race.

El test PostgreSQL es opt-in mediante `VEC_GOBIERNO_USUARIOS_ENSAYO_CONFIG`:
JSON0600 fuera de Git, con fase/rutas/DSN privados. Normalmente se omite. Preparar
sólo escribe plan/material privados; aplicar/replay exige aprobación externa y el
LOGIN configurado; verificar lee un tramo contiguo de la cadena técnica real. Los tests normales
no abren PostgreSQL. `VEC_GOBIERNO_USUARIOS_LAB_CONFIG` activa por separado el
setup DEV, con autorización explícita y SHA de plan/configuración;
`TestGobiernoUsuariosCoexistenciaCTPrivado` sólo consulta el helper CT.
El setup reserva semilla y acta con O_EXCL antes de abrir PostgreSQL; una salida
existente se rechaza sin alterar el gobierno. La semilla se escribe y sincroniza
antes del efecto; el descriptor del acta reservado se completa tras COMMIT.
Las negativas de colisión se comprueban sin PostgreSQL.
Las filas históricas se conservan.

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
revisiones de seguridad/SQL acreditan el código y sus límites, sin acreditar una sesión nominal o producción. El recorrido técnico favorable
es el que se describe arriba, con LOGIN y auditoría reales.
