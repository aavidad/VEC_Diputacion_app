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
