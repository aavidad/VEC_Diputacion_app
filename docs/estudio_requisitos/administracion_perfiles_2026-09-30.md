# Administración de perfiles: requisitos y orden de implantación

**Estado: estudio de diseño, sin administración funcional.** La PR #211 abrió
este estudio. La [PR #232](https://github.com/aavidad/VEC_Diputacion_app/pull/232)
prepara la frontera y contratos, pero no da de alta administradores ni expone
la pantalla de gestión. Ningún corte citado aquí acredita instalación en
cidonia, firma legal o acceso a expedientes.

## Decisiones vigentes

1. Una persona puede tener varios perfiles. Cada petición utiliza uno solo,
   elegido expresamente y ligado a la identidad y al canal. Sus permisos no
   se suman.
2. Hay un único tipo de rol para administrar perfiles. Cada administrador
   tiene su propia cuenta privilegiada y asignación nominal vigente. El rol
   común no convierte `PerfilActivoRef` en una referencia compartida ni da
   acceso a Contratación, Bolsa, Personal o documentos.
3. El acto inicial de instalación provisiona **a la vez dos personas
   administradoras nominativas ya acreditadas**. Alberto aprueba expresamente
   la preimagen y su huella; el aplicador usa CAS y confirma las dos altas en
   una sola transacción. La decisión consta en el canal del 30/09/2026 a las
   21:25 y 22:30. Preparar un plan y su huella no efectúa el alta.
4. Después del arranque, dar o quitar el rol administrador exige propuesta y
   aprobación de otra persona administradora, distinta de la proponente y de
   la afectada. El mismo doble control rige las altas y bajas de Intervención.
   Las asignaciones nominales y la decisión V3 se revalidan al aplicar.
5. La guarda de una baja conserva al menos una persona administradora activa.
   Se recomienda mantener dos o más para sostener el doble control. Si se
   pierde una credencial o una persona, se bloquean los actos sin aprobador
   válido aunque aún quede una persona administradora activa.
   La recuperación necesita procedimiento excepcional, aprobación expresa del
   operador, evidencia y revisión antes de implementarse. No abre un segundo
   bootstrap ni permite autoalta.
6. ADMIN usa proceso, subdominio, cuenta técnica y CA propios. En cidonia el
   TLS cliente llega directo a Go por passthrough de capa 4; Go verifica la
   cadena mTLS de la CA ADMIN. Un PEM reenviado en una cabecera no acredita
   `VerifiedChains`.
7. Solo en la frontera ADMIN de desarrollo y cidonia, un certificado ADMIN
   verificado puede aportar garantía HIGH sin Kerberos. Aun así se exige la
   concesión administrativa exacta y vigente. Producción exige Kerberos,
   certificado y CIDR corporativas; cualquier falta deniega.
8. Los actos, consultas y denegaciones quedan auditados con datos mínimos.
   Una revocación conserva su historia y no vuelve a `activo` por otra versión.

Estas decisiones concretan la separación de autoridades y la denegación por
defecto de [ESPECIFICACIONES_AGENTES.md](../../ESPECIFICACIONES_AGENTES.md)
(E04–E07 y E10). La matriz general distingue otras funciones técnicas
privilegiadas; el rol de este estudio gobierna perfiles sin absorberlas.

## Contraste con productos existentes

Las fuentes describen sus productos. El encaje de la última columna es una
decisión de VEC; ninguna fuente acredita una práctica de la Diputación.

| Fuente primaria | Hecho comprobado | Encaje en VEC |
| --- | --- | --- |
| [Keycloak: hostname de administración](https://www.keycloak.org/server/hostname) | Permite otra URL para la consola; advierte que `hostname-admin` por sí solo no cierra la API administrativa en la URL principal. | Subdominio, proceso y rutas ADMIN separados; la entrada pública no enruta su API. |
| [Microsoft Entra PIM: aprobación](https://learn.microsoft.com/en-us/entra/id-governance/privileged-identity-management/pim-approval-workflow) | Admite aprobación delegada para activar roles e impide al solicitante aprobar su propia activación. | Dos personas distintas para cambios de administrador e Intervención; VEC revalida la asignación al ejecutar. |
| [Teleport: solicitudes de acceso](https://goteleport.com/docs/identity-governance/access-requests/access-request-configuration/) | Configura roles solicitables y revisores; por defecto no se puede solicitar elevación. | Lista positiva cerrada y doble control. Una solicitud nunca publica permisos por sí sola. |
| [GitHub: continuidad de propietarios](https://docs.github.com/en/organizations/managing-peoples-access-to-your-organization-with-roles/maintaining-ownership-continuity-for-your-organization) | Recomienda dos personas propietarias para evitar perder acceso y no permite cambiar el rol propio. | Arranque obligatorio con dos personas, recomendación de mantener dos y guarda posterior de una activa; recuperación excepcional con contrato propio. |

## Autoridades que se reutilizan

| Pieza existente | Contrato para este estudio |
| --- | --- |
| `contexto_actor_v1`: `perfil_versiones`, `perfil_actual`, `vinculo_contexto_versiones` y `vinculo_contexto_actual` | Cada vínculo de persona y perfil tiene referencia, vigencia y estado. El perfil activo procede de un vínculo vivo de esa persona. |
| `internal/vec/domain/contexto_actor.go` | `SolicitudContextoActor` exige un `PerfilActivoRef` concreto; no elige un perfil por defecto. |
| `autorizacion`: `version_rol`, `asignacion_perfil` y `asignacion_perfil_actual` | La versión del rol se publica mediante el circuito gobernado. Cada administrador conserva una asignación nominal distinta, con versión, ámbito, vigencia y acto. |
| Provisión existente por huella y CAS | La preimagen aprobada identifica exactamente dos personas acreditadas, dos asignaciones y la versión del rol. Una petición web no es aprobación. |
| `SuperficieAdministracionPrivilegiada` y sesiones durables | ADMIN usa cuenta privilegiada, audiencia y superficie propias. La excepción HIGH en desarrollo/cidonia se limita a esta frontera. |
| Auditoría V3 y registro de sesiones | Se conserva quién hizo cada lectura, denegación, propuesta, aprobación y cambio, y su resultado. El administrador funcional no altera el destino de auditoría. |

En desarrollo existe además una provisión autorizada de doble llave, preimagen
y huella CAS. Se debe cotejar con el acto inicial antes de instalar, sin
interpretar su presencia como alta ejecutada ni copiar identidades o claves a
Git. El rol SQL técnico del pool no equivale al rol humano publicado ni a sus
dos asignaciones nominales.

## Contrato funcional pendiente

### Perfil activo y sesión

La cabecera muestra los perfiles que el servidor resolvió para la misma
persona y cuenta, con nombre del catálogo i18n. Elegir otro perfil abre un
contexto de sesión ligado a su referencia nominal; el servidor comprueba
titularidad, vigencia, revocación, audiencia y superficie. Invalida la sesión
anterior y audita el cambio. El identificador enviado por el navegador nunca
concede el perfil. Pestañas que usen la sesión antigua reciben denegación o
piden recarga; no heredan el nuevo perfil por compartir navegador. Cabecera y
acciones siguen la sesión confirmada, sin guardar permisos o identidad en
cookies ni almacenamiento web.

El administrador usa su perfil solo en el subdominio ADMIN y con cuenta
privilegiada. No aparece en el selector del portal ordinario. El selector y
su sesión requieren una pieza posterior; la PR #232 no los entrega.

### Asignaciones, doble control y recuperación

Cada acto conserva actor, persona afectada, perfil nominal, versión publicada
del rol, ámbito, vigencia, versión y huella anteriores, versión y huella
nuevas, motivo, correlación e instante. Compara la preimagen exacta por CAS y
vuelve a comprobar la decisión V3 en la misma transacción que escribe estado,
auditoría y recibo. Un replay de la misma clave recupera el recibo anterior
sin duplicar efectos; una preimagen diferente se rechaza. No hay provisión
implícita por petición, menú o arranque. En desarrollo permanece la provisión
expresamente autorizada con doble llave, preimagen y huella CAS, distinta del
acto inicial ADMIN aquí definido.

Una revocación crea historia y retira la asignación vigente. Una futura
concesión autorizada usa otra referencia; no reactiva el vínculo ni la
asignación revocados. Intervención necesita una referencia de rol publicada y
exacta antes de abrir sus actos. Hasta entonces su alta o baja se deniega.
Gestionar esa asignación no permite al administrador fiscalizar expedientes.

Las propuestas de administrador e Intervención tienen referencia, contenido,
huella inmutable, caducidad gobernada, estado e historia. El aprobador ve la
preimagen y el efecto, aporta motivo y no puede ser proponente ni afectado.
Se cotejan dos personas y dos asignaciones vivas, nunca dos sesiones o dos
perfiles de la misma persona. Para una baja ordinaria se bloquea el conjunto
relevante y se comprueba dentro de la transacción que permanece al menos una
persona administradora activa. Dos bajas concurrentes no pueden dejar cero: SQL
debe usar bloqueo/CAS o aislamiento serializable con reintento completo de
`40001`, y probarlo en PostgreSQL 18. Véanse el
[aislamiento](https://www.postgresql.org/docs/18/transaction-iso.html) y los
[bloqueos](https://www.postgresql.org/docs/18/explicit-locking.html).

El acto inicial único valida aprobación expresa, dos personas nominativas ya
acreditadas y distintas, sus cuentas privilegiadas, versión del rol y preimagen
por huella/CAS. Si falta algo, no aplica ninguna asignación. La señal de
bootstrap consumida y los dos recibos se escriben juntos; la repetición exacta
recupera el resultado y una segunda preimagen distinta se rechaza. No se crea
primero una persona administradora para que apruebe a la segunda.

La pérdida sobrevenida de ambos administradores es un bloqueo operativo. El
estudio no define aún un mecanismo de recuperación extraordinaria: debe
autorizarse y revisarse con su propia preimagen, actor y rastro antes de
implementarlo. No se reutiliza la operación inicial ya consumida.

### Auditoría, frontera y pantalla

Se auditan también búsqueda y consulta de personas, perfiles y propuestas,
incluidas denegaciones, sin volcar DNI, certificados, documentos o resultados
completos en logs. La historia es de solo adición y reconstruye preimagen,
decisión, aprobador, resultado y recibo. La vista devuelve solo campos
necesarios para gestionar perfiles; el administrador no edita esa auditoría.

`vec-admin` escucha en listener propio, con CA ADMIN distinta de la del portal
interno. En cidonia, Caddy enruta TLS por passthrough de capa 4 hasta Go; se
comprueban cadena y revocación del certificado cliente. La política nominal
liga certificado y cuenta privilegiada, además de exigir asignación V3.
Producción añade Kerberos y CIDR corporativas en entrada y aplicación. Red,
CA, certificados y cuentas quedan fuera de Git. No se infiere identidad de
`Host`, `X-Forwarded-*` ni de un PEM enviado por el cliente.

La futura pantalla permitirá buscar una persona por campos mínimos, consultar
sus perfiles, proponer altas y bajas, revisar propuestas ajenas y leer el
historial permitido. Mostrará preimagen, efecto y motivo antes de confirmar;
explicará caducidad, conflictos y denegaciones sin revelar datos ajenos. Los
textos estarán en `web/static/textos/<idioma>/*.json`. Requiere revisión de
usabilidad independiente antes de publicarse. Las pestañas dibujadas hoy no
son autoridad funcional.

## Orden real y límite de la PR #232

| Orden | Entrega o dependencia | Estado que puede afirmarse |
| --- | --- | --- |
| A1/A2: CA19 y AUT22 | Historia que no revive y rol ADMIN lector sin negocio. | Preparados y ensayados en clon; sin asignación ni administración funcional. |
| CA20 → AUT23 → IS9 | Fachadas y actos con doble control; política de certificado y vínculo nominal. | Preparados y ensayados, sin aplicar plan inicial ni abrir permisos runtime. |
| Paso 3 de M en el núcleo V3 | Postimagen y orden que fija Dirección en `ORDEN_SQL_NUCLEO.md`, fuera de Git. | Dependencia de AD137; no se adelanta. |
| AD137 → AUT24 | Consumidor V3 y aplicación atómica del plan aprobado de dos personas. | Pendientes; requieren ensayo, dos revisiones sensibles e instalación dirigida. |
| API, sesión de perfil y pantalla | Gestión y selector con autorización, auditoría y recorrido real. | Pendientes; la PR #232 solo expone `GET /livez` en ADMIN. |

La secuencia causal del material preparado por la PR #232 es
CA19 → AUT22 → CA20 → AUT23 → IS9, sobre la preimagen autorizada por Dirección.
Su CLI prepara y coteja un plan privado de dos personas; **no lo aplica**.
AUT24 deberá consumir la decisión V3 de AD137 y confirmar ambas altas en una
transacción. Ensayar en clon no instala SQL en cidonia. Dirección comprueba
por separado integración, instalación, API, navegador y recuperación.

## Comprobaciones de aceptación pendientes

- Instalación con dos personas sintéticas distintas y preimagen aprobada:
  dos altas y recibo recuperable. Preimagen distinta, aprobación ausente o
  persona repetida: cero efectos.
- Dos sesiones de una persona no satisfacen el doble control. Una persona
  afectada, revocada o sin asignación nominal vigente tampoco aprueba.
- Dos bajas concurrentes que dejarían cero administradores activos: no pueden
  confirmar ambas. Cada una exige proponente y aprobador válidos, con aprobador
  distinto del proponente y de la persona afectada; sin ellos, ninguna confirma.
- La gestión del perfil de Intervención exige rol publicado, propuesta y otra
  persona administradora aprobadora; una petición sola se deniega. Para
  fiscalizar, el rol ADMIN no basta: se exige un perfil activo de Intervención.
- TLS sin cadena verificada, CA equivocada, certificado revocado, identidad
  discordante o falta de CIDR/Kerberos en producción: denegación.
- Consulta autorizada y denegada auditadas; revocación sin reactivación;
  pestañas con sesión anterior sin perfil nuevo.
- Recorrido navegador → API → V3 → PostgreSQL → recibo, con recuperación tras
  reinicio, antes de declarar la administración utilizable. Autenticarse con
  certificado no firma una resolución ni acredita entrega legal.
