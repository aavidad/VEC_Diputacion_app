# Administración de perfiles: estudio y diseño

**Estado: BORRADOR incompleto.** Se cerró por orden de Alberto el 30/09/2026
antes de terminar. Falta, y no debe darse por hecho:

- la comparación con otros productos y administraciones (paso 1): no se ha
  consultado ninguna fuente, así que este borrador no afirma nada sobre cómo lo
  resuelven Workday, SAP SuccessFactors, Oracle HCM, Odoo, Keycloak, Entra ID
  PIM ni ninguna administración española;
- la revisión independiente de seguridad (opus-alto con `security-audit`);
- el repaso de pantallas con `usabilidad-vec` y el de textos con `humanizer`;
- la reserva real de números de migración en `RESERVAS_MIGRACIONES.md`.

Lo que sí contiene está sacado del código de `main` en `5cc7ca123`,
localizado con el índice de código.

## 1. Requisitos de Alberto

1. Una persona puede tener varios perfiles (técnico de RRHH, Intervención,
   centro solicitante, ratificador, empleado, candidato…). En la cabecera elige
   el perfil activo, y con él cambian la vista, los menús y los permisos.
2. Hay un perfil «administrador», el único que pone y quita perfiles.
3. La administración va en un subdominio propio (p. ej.
   `admin.vec.cidonia.cloud`), con mTLS de una CA propia de administración. En
   producción solo se entra desde la red corporativa (rangos en configuración,
   aplicados en Caddy y en la aplicación). En desarrollo y en cidonia, abierto.
4. El administrador no ve expedientes ni Bolsa (separación de funciones).
5. Dar o quitar el perfil de administrador exige doble control, y el sistema
   nunca puede quedarse sin ningún administrador.
6. El primer administrador se provisiona una sola vez, con aprobación del
   operador.
7. Todo queda auditado: quién, cuándo, antes, después y motivo.
8. Una revocación nunca revive.

## 2. Lo que ya existe y hay que reutilizar

No se crea una segunda autoridad de permisos. Todo lo necesario ya tiene dueño:

| Pieza | Dónde está | Qué aporta |
| --- | --- | --- |
| Perfil y vínculo cuenta-persona-perfil | `deploy/postgresql/contexto_actor_v1` (`perfil_versiones`/`perfil_actual`, `vinculo_contexto_versiones`/`_actual`) | Cada perfil es una referencia `prf_…` de una persona `per_…`, con estado `activo`/`revocado` y vigencia. Una persona con varios perfiles ya se representa como varios vínculos. |
| ContextoActor | `internal/vec/domain/contexto_actor.go` | `SolicitudContextoActor` exige un `PerfilActivoRef` concreto: un perfil vacío no significa «el habitual» ni se elige el primero. |
| Roles fijos y asignaciones V3 | `deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql` (`version_rol`, `control_vigencia_version_rol`, `asignacion_perfil`, `asignacion_perfil_actual`) | Rol versionado sin comodines; asignación versionada por `perfil_activo_ref`, con puntero actual que solo avanza (disparador `validar_avance_asignacion_actual`) y guarda `actualizada_por` y `acto_ref`. |
| Provisión por huella y CAS | `config/ct_provision_perfiles_centro.go`, `cmd/vec-publicar-permiso-interno` | El operador aprueba sustituir una asignación exacta por su huella SHA-256; nunca actúa sobre una revocada, restringida o retirada. No se publican permisos por petición ni al arrancar. |
| Superficie de administración | `internal/vec/adapters/httpseguridad/superficie.go` | Ya existen `SuperficieAdministracionPrivilegiada` y `ZonaRedAdministracion`. Exigen Kerberos + certificado, dos grupos criptográficos, garantía alta, cuenta privilegiada separada y autenticación de hace menos de 5 minutos. Una superficie no interna no puede escuchar en todas las interfaces. |
| Sesión | `internal/vec/adapters/httpseguridad/sesion_durable.go` (`AltaSesionAtomica`, con `CuentaPrivilegiada` y `Superficie`) | Alta de sesión atómica y durable, ligada a la superficie. |
| Separación de procesos | `cmd/vec-publico`, `cmd/vec-interno`, `cmd/vec-server`; PR #143/#147/#192 | Un proceso por portal, con su propio listener y su propio LOGIN de base de datos. |
| Pantalla de administración | `web/static/portal-empleado/modulos/administracion/` | Pestañas ya dibujadas sin autoridad («roles» con acciones bloqueadas). Textos en `web/static/textos/es/administracion.json`. |

## 3. Encaje propuesto en VEC

### 3.1 Perfil activo

- El perfil activo va **en la sesión del servidor**, nunca en cookies,
  `localStorage` ni cabeceras que mande el navegador.
- Cambiar de perfil es abrir una sesión nueva ligada al otro perfil. El
  navegador pide el cambio con una petición `POST` que solo lleva la referencia
  opaca `prf_…` elegida. El servidor:
  1. comprueba que ese perfil pertenece a la misma persona y cuenta de la
     sesión actual y que su vínculo está activo y vigente;
  2. resuelve de nuevo el `ContextoActor` con esa `SolicitudContextoActor`;
  3. da de alta una sesión nueva (misma autenticación, perfil distinto) y
     cierra la anterior;
  4. deja auditado el cambio (perfil anterior, perfil nuevo, sesión).
- Cada decisión V3 ya comprueba el perfil activo de la sesión. Cambiar el menú
  en el navegador no concede nada.
- La lista de perfiles de la cabecera sale de una lectura del servidor que
  devuelve solo las referencias y su nombre visible (clave i18n del rol), no
  los permisos.
- El perfil «administrador» no aparece en ese selector: solo existe en el
  subdominio de administración, con su propia cuenta privilegiada.

### 3.2 Perfil de administrador y separación de funciones

- Nuevo rol fijo `administracion_perfiles` en `version_rol`, cuyas concesiones
  son solo: listar personas y perfiles (datos mínimos), dar perfil, quitar
  perfil, leer la auditoría de perfiles y proponer o aprobar cambios de
  administrador.
- Ninguna concesión de expedientes, Contratación, Bolsa, Personal ni
  documentos. Una prueba lo verifica leyendo la versión publicada del rol.
- La cuenta de administrador es la cuenta privilegiada separada que ya exige
  la superficie de administración (`RequiereCuentaPrivilegiada`). Aunque la
  misma persona tenga perfiles de RRHH, no los usa desde ese proceso y el
  proceso de administración no tiene LOGIN de base de datos sobre esquemas de
  negocio.

### 3.3 Actos, doble control y mínimo de administradores

Cada cambio es un **acto** con referencia propia, idempotente y con CAS sobre
la versión actual:

| Acto | Quién | Control |
| --- | --- | --- |
| Dar perfil ordinario | un administrador | CAS sobre el vínculo y la asignación; motivo obligatorio |
| Quitar perfil ordinario | un administrador | CAS; motivo; pasa a `revocado` y ya no vuelve |
| Proponer dar o quitar administrador | un administrador | queda pendiente, con caducidad |
| Aprobar la propuesta | **otro** administrador distinto | persona distinta a quien propuso y a la afectada; misma huella de propuesta |
| Provisión del primer administrador | el operador, fuera de la web | una sola vez; aprobación por referencia y huella, como la provisión actual |

Reglas en la base de datos, no solo en Go:

- **Nunca cero administradores.** La función que aplica una revocación de
  administrador cuenta, con bloqueo, los administradores activos y vigentes y
  rechaza si quedaría ninguno. Recomendación a confirmar: exigir al menos dos
  antes de permitir una revocación.
- **Una revocación no revive.** Tras `revocado`, una nueva versión del mismo
  vínculo o asignación en `activo` se rechaza. Volver a dar el perfil crea un
  vínculo nuevo con otra referencia; la historia anterior se conserva.
- **Primer administrador, una sola vez.** Una fila de control que solo admite
  un único alta de arranque; si ya existe cualquier administrador (activo o
  revocado), la provisión se rechaza.
- Nadie aprueba su propia propuesta ni se revoca a sí mismo sin segundo
  administrador.

### 3.4 Auditoría

Cada acto guarda en una tabla de solo adición: referencia del acto, quién
(persona y cuenta opacas), cuándo (UTC), perfil afectado, estado y huella
**antes**, estado y huella **después**, motivo (texto libre limitado, sin datos
personales de terceros), propuesta y aprobación si hubo doble control, y
correlación de la petición. Se reutiliza `acto_ref`/`actualizada_por` de
`asignacion_perfil_actual`. La auditoría de cambios de perfil activo (3.1) va
en el registro de sesiones.

### 3.5 Proceso, subdominio, CA y red

- Proceso nuevo `cmd/vec-admin`, con la superficie
  `administracion_privilegiada` y zona `administracion` ya definidas, su
  propio listener y su propio LOGIN de base de datos con permisos solo sobre
  las funciones de administración de perfiles.
- Subdominio propio (p. ej. `admin.vec.cidonia.cloud`) en Caddy, con mTLS
  contra una **CA de administración distinta** de la interna. La aplicación
  vuelve a comprobar la huella del certificado de cliente.
- Rangos de red permitidos en configuración (`RedesPermitidas`), aplicados dos
  veces: en Caddy (`remote_ip`) y en la aplicación (`NuevaPoliticaRed`).
  Producción: solo redes corporativas. Desarrollo y cidonia: abierto, con la
  excepción declarada y fechada, como la política interna temporal.
- Pendiente de decidir: en cidonia no hay Kerberos, y la superficie exige
  Kerberos + certificado. Hará falta una excepción de desarrollo fechada,
  parecida a `PoliticaInternaDesarrolloCertificadoPersonal`, que hoy la
  superficie de administración prohíbe expresamente.

### 3.6 Pantallas del administrador (pendiente de `usabilidad-vec`)

Esbozo inicial, sin revisar:

1. **Buscar persona**: un buscador; resultado con nombre y unidad.
2. **Ficha de la persona**: sus perfiles con estado y fechas; botones «Dar
   perfil» y «Quitar perfil». Quitar pide motivo y avisa de que no se puede
   deshacer.
3. **Pendientes de aprobar**: propuestas de administrador de otros; aprobar o
   rechazar con motivo.
4. **Historial**: la auditoría filtrable por persona, perfil o fecha.

Todos los textos en `web/static/textos/<idioma>/administracion.json`. Las
pestañas actuales de `modulos/administracion` sirven de base.

### 3.7 SQL y números a reservar

Números orientativos, **no reservados todavía**. Reservarlos en
`RESERVAS_MIGRACIONES.md` al empezar cada corte, cogiendo el siguiente libre:

| Esquema | Número orientativo | Contenido |
| --- | --- | --- |
| `contexto_actor_v1` | siguiente libre tras 000018 | vínculo sin revivir tras revocación; lectura de perfiles de una persona |
| `autorizacion` | siguiente libre tras 000021 | rol `administracion_perfiles`, actos, propuestas, aprobación, mínimo de administradores, provisión única, auditoría |
| `autorizacion_atestada_v3` | siguiente libre tras 000128 | consumidor V3 de los actos de administración |
| `identidad_sesiones_v1` | siguiente libre tras 000008 | cambio de perfil activo con sesión nueva y su auditoría |

Cada migración: revisión con `revisar-sql-vec`, ensayo con `ensayar-sql` en el
clon de la principal y revisor SQL independiente.

## 4. Cortes propuestos

| Corte | Qué entrega | Depende de |
| --- | --- | --- |
| A1 | SQL: una revocación no revive (vínculo y asignación) | — |
| A2 | SQL: rol fijo `administracion_perfiles` sin acceso a negocio y prueba de ello | — |
| A3 | SQL: actos dar/quitar perfil ordinario con CAS, motivo y auditoría | A1, A2 |
| A4 | SQL: propuesta y aprobación de administrador, mínimo de administradores | A3 |
| A5 | CLI de operador: primer administrador una sola vez | A2, A4 |
| B1 | Proceso `cmd/vec-admin` con superficie de administración, mTLS de CA propia y rangos de red | — |
| B2 | Caddy: subdominio, CA de administración y `remote_ip` | B1 |
| B3 | Casos de uso y API de administración sobre A3–A4 | A4, B1 |
| B4 | Pantallas del administrador (con `usabilidad-vec` y revisión independiente) | B3 |
| C1 | SQL + backend: cambio de perfil activo con sesión nueva | A1 |
| C2 | Selector de perfil en la cabecera del portal interno | C1 |
| D1 | Recorrido completo en navegador y revisión de seguridad | B4, C2, A5 |

## 5. Preguntas abiertas

1. ¿Mínimo de uno o de dos administradores activos?
2. ¿Cómo se entra a `vec-admin` en cidonia sin Kerberos?
3. ¿El perfil de candidato se gestiona también desde aquí o sigue en su
   proceso externo separado (#143/#192)? Este borrador supone que sigue fuera.
4. ¿Caducan los perfiles ordinarios o duran hasta que se quitan?
