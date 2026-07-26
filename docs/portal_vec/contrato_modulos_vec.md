# Contrato de modulos VEC

## Objetivo

Definir una forma unica de enganchar modulos al portal VEC principal. La meta es
que Bolsa, concursos, nominas, permisos, accion social, formacion o cualquier
otro bloque funcional se conecten igual: mismo login, mismo shell, mismo modelo
de menu, mismas reglas de permisos, misma auditoria, mismo sistema de
notificaciones, mismo i18n y mismos puertos transversales.

Este documento es una propuesta para discutir con otros agentes antes de
programar el portal modular.

## Fuentes revisadas

- Portal del Empleo Publico de la Junta de Andalucia:
  `https://portalempleopublico.juntadeandalucia.es/`
- Sede Electronica del Empleo Publico:
  `https://portalempleopublico.juntadeandalucia.es/sede`
- Web del Empleado Publico Andaluz, mapa del sitio:
  `https://ws045.juntadeandalucia.es/empleadopublico/emp-mapasitio-.html?p=%2FCategorias_Principales%2F`
- Catalogo de procedimientos y servicios para personal empleado:
  `https://portalempleopublico.juntadeandalucia.es/sede/acceso-tramites/rps-personal-empleado`
- Preguntas frecuentes IAAP sobre Web del Empleado Publico y procesos
  selectivos:
  `https://www.juntadeandalucia.es/organismos/iaap/areas/empleo-publico/preguntas-frecuentes.html`

## Decision base

VEC no debe ser una aplicacion monolitica de Bolsa. Debe ser un portal modular
con una zona unica de identificacion para personal interno mediante Kerberos/AD.
Una vez autenticado, el usuario puede saltar de un modulo a otro sin repetir
login.

Reglas:

- El shell VEC es propietario de identidad, sesion, layout, menu, permisos
  globales, busqueda global, notificaciones, auditoria transversal e i18n base.
- Cada modulo aporta dominio, casos de uso, rutas, pantallas, permisos propios,
  widgets, eventos, tipos documentales y acciones.
- Ningun modulo implementa su propio login.
- Ningun modulo decide por strings de UI si el usuario puede operar; usa claims
  y permisos del contexto VEC.
- Bolsa entra como `vec.module.bolsa`, igual que entrarian Nominas, CAP,
  Permisos o Accion Social.

## Arquitectura objetivo

```text
vec-shell
  identidad Kerberos/AD
  sesion y permisos
  menu modular
  tablero
  busqueda global
  notificaciones
  expediente/documentos comunes
  auditoria comun
  i18n comun
  registro de modulos

vec-core
  contratos de modulo
  contratos de identidad
  contratos de navegacion
  contratos de documentos
  contratos de notificacion
  contratos de auditoria
  contratos de eventos

vec-module-bolsa
  dominio bolsa
  solicitudes
  meritos/RUM
  autobaremacion
  listados
  alegaciones

vec-module-*
  cualquier otro modulo con el mismo contrato
```

## Contrato unico de modulo

Todo modulo debe publicar un manifiesto estable. El shell no debe conocer
detalles internos del modulo; solo consume este contrato.

```json
{
  "module_ref": "vec.module.bolsa",
  "version": "v1",
  "title_i18n_key": "module.bolsa.title",
  "description_i18n_key": "module.bolsa.description",
  "category_ref": "seleccion_y_bolsas",
  "base_route": "/bolsa",
  "api_prefix": "/api/modules/bolsa",
  "required_roles": ["employee", "candidate_manager"],
  "menu_entries": [
    {
      "entry_ref": "bolsa.solicitudes",
      "label_i18n_key": "module.bolsa.menu.solicitudes",
      "route": "/bolsa/solicitudes",
      "required_permissions": ["bolsa.solicitud.read"]
    }
  ],
  "widgets": [
    {
      "widget_ref": "bolsa.pending_actions",
      "slot": "dashboard.summary",
      "required_permissions": ["bolsa.dashboard.read"]
    }
  ],
  "capabilities": [
    "expediente",
    "documentos",
    "notificaciones",
    "auditoria",
    "tramites"
  ],
  "events_published": [
    "bolsa.solicitud_registrada",
    "bolsa.autobaremo_calculado",
    "bolsa.alegacion_presentada"
  ],
  "events_subscribed": [
    "documento.validado",
    "notificacion.leida"
  ],
  "health_route": "/api/modules/bolsa/healthz"
}
```

Campos obligatorios:

| Campo | Uso |
| --- | --- |
| `module_ref` | Identidad opaca, estable y unica del modulo. |
| `version` | Version del contrato del modulo. |
| `category_ref` | Agrupacion del menu principal. |
| `base_route` | Ruta frontend propiedad del modulo. |
| `api_prefix` | Prefijo API del modulo. |
| `required_roles` | Roles minimos para ver el modulo. |
| `menu_entries` | Entradas de menu que el shell dibuja. |
| `widgets` | Resumenes que el shell puede colocar en tablero. |
| `capabilities` | Capacidades transversales que consume. |
| `events_published` | Eventos que emite al bus VEC. |
| `events_subscribed` | Eventos que escucha. |
| `health_route` | Comprobacion de salud del modulo. |

## Backend: interfaz comun

El shell debe depender de una interfaz comun, no de paquetes concretos como
Bolsa.

```go
type VECModuleProvider interface {
    Manifest(ctx context.Context) (VECModuleManifest, error)
    Menu(ctx context.Context, principal VECPrincipal) ([]VECMenuEntry, error)
    DashboardWidgets(ctx context.Context, principal VECPrincipal) ([]VECWidget, error)
    Routes() []VECHTTPRoute
    Health(ctx context.Context) VECModuleHealth
}
```

El siguiente `VECPrincipal` es una proyeccion historica de presentacion y
traza. Su modelo de autorizacion ha sido sustituido por DEC-009, DEC-020 y
DEC-037: aunque el shell lo cree desde Kerberos/AD, ninguno de sus campos es
una concesion ejecutable.

```go
type VECPrincipal struct {
    SubjectRef     string
    EmployeeRef    string
    DisplayName    string
    Units          []string
    Roles          []string
    Permissions    []string
    AuthMethod     string // kerberos_ad
    SessionRef     string
    CorrelationRef string
}
```

Reglas:

- `VECPrincipal` llega autenticado al modulo, pero autenticacion no equivale a
  autorizacion.
- El modulo nunca autoriza usando roles, permisos, unidades o `claims`
  transportados por el cliente o por el shell. Solicita al PDP central una
  decision positiva y exacta para cada caso de uso; menu y manifiesto solo
  controlan presentacion.
- Un dato ausente, desconocido, ambiguo, caducado o no verificable deniega. No
  hay suma de perfiles, herencia, comodines positivos ni administrador
  universal.
- Las reglas de dominio viven en casos de uso del modulo.
- Los handlers HTTP del modulo solo adaptan transporte, DTOs e i18n.
- Las acciones de politica se expresan como claves concretas:
  `bolsa.solicitud.read`, `nomina.recibo.download`,
  `permisos.solicitud.submit`, etc.; una clave no concede por existir en el
  manifiesto.

## Frontend: interfaz comun

El shell renderiza estructura comun. El modulo registra vistas y acciones.

```ts
type VECFrontendModule = {
  moduleRef: string;
  mount(base: VECMountContext): void;
  unmount(): void;
  routes: VECRoute[];
  widgets: VECWidgetDescriptor[];
};

type VECMountContext = {
  principal: VECPrincipalView;
  api: VECAPIClient;
  i18n: VECI18n;
  notify: VECNotificationClient;
  audit: VECAuditClient;
  documents: VECDocumentClient;
  navigate: (route: string) => void;
};
```

Reglas:

- El modulo no dibuja sidebar global ni topbar global.
- El modulo no guarda tokens ni implementa login.
- El modulo no calcula baremos, nominas, permisos ni resoluciones legales en
  JavaScript; pide estados y resultados al backend.
- El modulo puede tener componentes propios, pero usa controles base del shell
  para tabla, filtros, detalle, timeline, documentos, notificaciones y recibos.

## Eventos estandar

Todos los modulos publican eventos con envelope comun.

```json
{
  "event_ref": "evt-...",
  "event_type": "bolsa.autobaremo_calculado",
  "module_ref": "vec.module.bolsa",
  "subject_ref": "employee-or-candidate-ref",
  "actor_ref": "employee-ref",
  "occurred_at": "2026-06-19T12:00:00Z",
  "correlation_ref": "corr-...",
  "payload_ref": "opaque-payload-ref",
  "audit_level": "administrative"
}
```

Eventos transversales recomendados:

| Evento | Productor | Consumidores |
| --- | --- | --- |
| `documento.aportado` | Cualquier modulo | Expediente, auditoria, notificaciones |
| `documento.validado` | Documentos | Modulos con evidencias |
| `notificacion.emitida` | Cualquier modulo | Buzon, auditoria |
| `tramite.borrador_creado` | Cualquier modulo | Tablero, auditoria |
| `tramite.presentado` | Cualquier modulo | Expediente, registro, notificaciones |
| `alegacion.presentada` | Cualquier modulo | Unidad revisora, auditoria |
| `resolucion.publicada` | Cualquier modulo | Notificaciones, expediente |

## Capacidades transversales del shell

Estas capacidades no deben duplicarse por modulo:

| Capacidad | Propietario | Uso por modulos |
| --- | --- | --- |
| Identidad Kerberos/AD | `vec-shell` | Reciben `VECPrincipal`. |
| Sesion | `vec-shell` | No hay login por modulo. |
| Menu | `vec-shell` | Consume `menu_entries`. |
| Permisos | `vec-core`/shell | Evalua visibilidad; modulo valida accion. |
| i18n | `vec-core` | Catalogo por modulo, claves sin texto hardcodeado. |
| Notificaciones | `vec-core` | Buzon unico y eventos por modulo. |
| Documentos/expediente | `vec-core` | Adjuntos, CSV, ENI, firma, versionado. |
| Auditoria | `vec-core` | Timeline unico por sujeto/tramite/modulo. |
| Busqueda global | `vec-shell` | Indices aportados por modulos. |
| Salud | `vec-shell` | Consulta `health_route` de cada modulo. |

## Mapa de modulos inicial

Basado en el mapa del sitio y las paginas oficiales revisadas, estas opciones
de menu pueden modelarse como modulos o familias de modulos.

| Categoria VEC | Modulo candidato | Opciones fuente |
| --- | --- | --- |
| Seleccion y bolsas | `vec.module.procesos_selectivos` | Acceso funcionarios, acceso laborales, promocion interna, modelos, consulta plazas. |
| Seleccion y bolsas | `vec.module.bolsa` | Bolsa de trabajo, bolsa unica comun, interinos, laborales, vista expediente, solicitud, alegaciones. |
| Provision | `vec.module.cap` | Concurso abierto y permanente funcionario/laboral, participacion, consulta puestos, alegaciones, desistimiento, opcion, vista expediente. |
| Provision | `vec.module.concursos` | Concursos de meritos, concurso funcionarios, concurso laborales, PLD, SNL, permutas. |
| Tramites laborales | `vec.module.accion_social` | Ayudas, anticipos reintegrables, alegaciones, ayudas discapacidad. |
| Tramites laborales | `vec.module.derechos_deberes` | Compatibilidad, horarios, salud laboral, vacaciones, permisos y licencias. |
| Puesto y carrera | `vec.module.puesto_trabajo` | RPT, caracteristicas de puesto, justicia, plantillas. |
| Puesto y carrera | `vec.module.carrera_horizontal` | Evaluacion del desempeno, carrera horizontal, consolidacion de grado. |
| Retribuciones | `vec.module.nominas` | Nomina, certificados de retenciones IRPF, certificados personales. |
| Formacion | `vec.module.formacion` | Catalogo formacion, SAFO, acciones formativas. |
| Tramitacion electronica | `vec.module.tramites` | Consulta de tramites, presentaciones, desistimientos, peticion destino. |
| Datos personales | `vec.module.mis_datos` | Historial administrativo, hoja acreditacion, vida administrativa, curriculum, domicilio, IRPF, haberes. |
| Servicios | `vec.module.servicios_empleado` | Calendarios oficiales, direcciones, junta de personal, informacion DGRRHHFP. |
| Seguridad documental | `vec.module.verificacion` | Verificador de firma, formatos admisibles, codigos de organos, registros. |

## Bolsa como primer modulo

Bolsa debe engancharse al VEC asi:

```text
module_ref: vec.module.bolsa
category_ref: seleccion_y_bolsas
base_route: /bolsa
api_prefix: /api/modules/bolsa
uses:
  - identidad Kerberos/AD del shell
  - expediente/documentos comun
  - notificaciones comun
  - auditoria comun
  - i18n comun + catalogo propio
owns:
  - convocatoria bolsa
  - solicitud bolsa
  - meritos/RUM aplicados a bolsa
  - autobaremacion
  - listados provisional/definitivo
  - alegaciones de bolsa
```

Bolsa no debe ser especial. Si un segundo modulo, por ejemplo `nominas`, no
puede enganchar usando el mismo manifiesto y las mismas interfaces, el contrato
esta mal disenado.

## Ciclo de vida de instalacion

1. El modulo registra su `VECModuleManifest`.
2. El shell valida version, rutas, permisos y colisiones.
3. El shell monta entradas de menu segun permisos del usuario.
4. El shell llama `DashboardWidgets` para el tablero.
5. Al navegar, el shell monta la vista del modulo en la zona de contenido.
6. Las acciones del modulo llaman su API bajo `api_prefix`.
7. El modulo publica eventos administrativos.
8. Shell/core actualizan auditoria, expediente y notificaciones.
9. `health_route` permite saber si el modulo esta operativo.

## Gates para aceptar un modulo

Un modulo VEC solo se acepta si cumple:

- `gate-manifest`: manifiesto valido, versionado y sin colision de rutas.
- `gate-sso`: no tiene login propio; usa `VECPrincipal`.
- `gate-hexagonal`: dominio y casos de uso no importan shell, HTTP, DB,
  Orquesta, rutas locales ni proveedor externo.
- `gate-i18n`: textos visibles con claves del catalogo.
- `gate-permissions`: permisos declarados en manifiesto y validados en backend.
- `gate-documents`: documentos por puerto comun, no filesystem interno.
- `gate-notifications`: avisos por bus/notificacion comun, no canal propio.
- `gate-audit`: acciones administrativas emiten evento auditable.
- `gate-health`: expone salud del modulo.
- `gate-tests`: pruebas focales del modulo y prueba de integracion con shell.

## Preguntas para discutir con otros agentes

1. El manifiesto debe ser JSON estatico, Go struct registrado en bootstrap o
   ambos?
2. El shell debe montar frontend por imports locales, web components o bundle
   por modulo?
3. Conviene un bus de eventos en memoria primero y adaptador persistente
   despues?
4. `Documentos`, `Notificaciones` y `Auditoria` son modulos visibles propios o
   capacidades internas del shell con pantallas globales?
5. Bolsa debe exponer RUM/meritos como parte propia o consumir un modulo comun
   `vec.module.rum`?
6. Como versionar permisos para que un modulo nuevo no rompa perfiles
   existentes?
7. Que prueba minima debe demostrar que un modulo se puede instalar, ver en
   menu, ejecutar una accion, emitir evento y auditarse?

## Propuesta de primera entrega

Antes de programar mas funcionalidad VEC, crear una vertical minima:

```text
vec-core/module_manifest.go
vec-core/principal.go
vec-core/menu.go
vec-core/events.go
vec-shell/module_registry.go
vec-module-bolsa/manifest.go
tests:
  - registra Bolsa
  - menu aparece con permisos correctos
  - usuario Kerberos salta de Dashboard a Bolsa sin nuevo login
  - accion Bolsa emite evento auditable
  - modulo sin permiso no aparece
```

Cuando esto este verde, cada opcion del menu real puede convertirse en modulo
sin volver a debatir como se engancha al VEC principal.

---

# Revision y mejoras (contraste con administraciones reales)

Esta seccion la anade una revision posterior. No reemplaza la propuesta de
arriba; la complementa contrastandola con como resuelven esto portales reales
del empleado publico (Junta de Andalucia, Punto de Acceso General, sedes
autonomicas) y con la arquitectura de referencia de microfrontend que la propia
Junta publica. Las fuentes estan al final.

## Resumen de la revision

El contrato esta bien planteado: shell propietario de identidad/sesion/menu,
modulos hexagonales que solo aportan dominio y vistas, manifiesto estable,
gates de aceptacion y vertical minima antes de escalar. Eso coincide con la
practica real. Hay, sin embargo, **cinco huecos importantes** que conviene
cerrar antes de programar, porque cambian contratos de datos y son caros de
retro-encajar:

1. Identidad: asumir solo Kerberos/AD es insuficiente para una administracion.
2. Falta firma electronica como capacidad transversal (no es lo mismo que login).
3. Falta registro electronico y asiento registral como capacidad transversal.
4. El frontend "imports locales vs web components" ya tiene respuesta de referencia.
5. Faltan transversales de cumplimiento: ENS, RGPD/consentimiento, accesibilidad.

## 1. Identidad: Kerberos/AD no basta (CRITICO)

El documento fija `AuthMethod = kerberos_ad` como unica via. Los portales reales
del empleado publico **no** usan una sola via:

- **SSOWeb / SSO corporativo** para personal interno desde la red (este es el
  caso de uso que cubre tu Kerberos/AD).
- **Certificado electronico** (FNMT, DNIe) para acceso desde fuera de la red.
- **Cl@ve** (Cl@ve PIN, Cl@ve Permanente) como identidad estatal.

Un empleado a veces entra desde su puesto interno (SSO) y a veces desde casa con
certificado o Cl@ve. Si el contrato solo entiende Kerberos, el dia que haya que
abrir teletrabajo o tramites fuera de la red corporativa hay que rehacer el
contrato de identidad y todos los modulos que asuman `kerberos_ad`.

Cambio propuesto en `VECPrincipal`:

```go
type VECPrincipal struct {
    SubjectRef       string
    EmployeeRef      string
    DisplayName      string
    Units            []string
    Roles            []string
    Permissions      []string
    AuthMethod       string // sso_corporativo | certificado | clave
    AuthAssurance    string // bajo | sustancial | alto  (nivel eIDAS/ENS)
    IdentityProvider string // ref del IdP que autentico
    SessionRef       string
    CorrelationRef   string
}
```

Regla nueva: ciertos tramites (presentar solicitud, alegacion, aceptar plaza)
deben exigir un `AuthAssurance` minimo. Eso lo valida el shell/core, no el
modulo, pero el manifiesto del modulo debe poder declarar el nivel requerido por
accion. Sin esto, un modulo no puede expresar "esto se firma, no basta con estar
logueado".

### Decision tomada: certificado como metodo preferente

Decision del responsable del proyecto: el **certificado electronico (DNIe/FNMT)**
es el metodo preferente por ser el mas seguro (nivel de garantia alto eIDAS,
criptografico, resistente a phishing).

Reglas concretas de esta decision:

- **Certificado es obligatorio** para cualquier accion que genere un acto
  administrativo firmado: presentar solicitud, alegacion, aceptar/renunciar
  plaza, desistir. Estas acciones exigen `AuthAssurance = alto`.
- **SSO corporativo se admite** solo para navegacion y consulta del empleado
  interno desde la red (comodidad de uso diario), nunca para firmar. Equivale a
  `AuthAssurance = sustancial` como maximo.
- **Cl@ve** queda como via alternativa de identificacion para acceso fuera de la
  red cuando no haya certificado a mano, sujeta al mismo limite: no firma actos
  que requieran `alto`.

Por que no "solo certificado para todo": obligar al certificado en cada login
diario del personal interno genera friccion alta sin ganancia real de seguridad
en acciones de solo lectura. La seguridad se concentra donde importa (firma de
actos) via `AuthAssurance`, no bloqueando la navegacion. Si mas adelante se
decide endurecer y exigir certificado tambien para entrar, basta subir el
`AuthAssurance` minimo del shell; el contrato ya lo soporta sin reescritura.

## 2. Firma electronica como capacidad transversal (CRITICO)

El documento menciona "firma" de pasada dentro de `documentos`, pero firmar un
tramite es una capacidad distinta de adjuntar un documento. En una
administracion, presentar una solicitud o una alegacion produce un **acto
firmado** con sello de tiempo, no solo un registro en BD.

Propuesta: anadir capacidad `firma` al shell/core, con su puerto:

```go
type VECSignaturePort interface {
    Sign(ctx context.Context, principal VECPrincipal, req VECSignRequest) (VECSignReceipt, error)
    Verify(ctx context.Context, ref string) (VECSignVerification, error)
}
```

Y un evento estandar `tramite.firmado` en la tabla de eventos. El gate
`gate-tramites` deberia exigir que toda accion que cree un acto administrativo
pase por el puerto de firma + sello de tiempo, nunca por logica propia del
modulo.

### Decision: usar AutofirmaV2 (Dipgra) como motor de firma y de auth por certificado

Tenemos software propio: **AutofirmaV2** (Go, hexagonal, GPLv3, Oficina de
Software Libre de la Diputacion de Granada), corriendo como cliente local del
empleado en loopback (`127.0.0.1:8080`). Expone REST con, entre otros:

- `POST /sign` y `POST /sign-batch` (firma trifasica de lotes).
- `POST /verify` (verificacion de firma).
- `GET /certificates` (certificados disponibles del empleado).
- `POST /auth/challenge` + `POST /auth/verify` (autenticacion reto/respuesta por
  certificado).

Decision: **el VEC no reimplementa firma ni identidad por certificado.**
`vec-core` define los puertos (`VECSignaturePort`, `VECCertAuthPort`) y el shell
los satisface con un adaptador delgado que invoca AutofirmaV2 local. AutofirmaV2
es un adaptador externo intercambiable; el dominio del VEC no lo conoce.

Asi, las dos decisiones de identidad y firma se cubren con software propio ya
probado:

- Identidad por certificado (`AuthMethod=certificado`, `AuthAssurance=alto`) via
  `/auth/challenge` + `/auth/verify`.
- Firma de actos administrativos via `/sign` (y `/sign-batch` para resoluciones
  masivas), verificacion via `/verify`.

Salvedad arquitectonica importante: AutofirmaV2 vive en el **puesto del
empleado** (loopback), no es un servicio central. La clave privada nunca sale del
puesto. Por tanto el flujo es **frontend del shell -> AutofirmaV2 del puesto**
(patron tipo `afirma://`, como la AGE con AutoFirma), no servidor-a-servidor. El
adaptador de firma del VEC es principalmente de **frontend**: el shell orquesta
la llamada al AutofirmaV2 local y recibe el resultado firmado + justificante,
que luego el backend registra y audita. El backend nunca firma por su cuenta.

## 3. Registro electronico / asiento registral (ALTO)

Relacionado con lo anterior pero separado: cuando un tramite se "presenta", la
administracion genera un **asiento de registro** (numero de registro, fecha,
justificante). Tu tabla de eventos ya tiene `tramite.presentado`, pero no hay
capacidad ni puerto de registro. Propuesta: capacidad `registro` + puerto
`VECRegistryPort` que devuelva numero de asiento y justificante (PDF/CSV), y que
`tramite.presentado` lleve el `registro_ref` en el envelope. Sin esto, los
modulos acabaran inventando su propia numeracion, que es justo lo que el contrato
quiere evitar.

## 4. Frontend: ya hay respuesta de referencia (resuelve tu pregunta abierta #2)

Tu pregunta abierta era "imports locales, web components o bundle por modulo".
La arquitectura de referencia de microfrontend de la Junta de Andalucia ya fija
un patron que encaja exactamente con tu shell:

- **Webpack Module Federation** para cargar modulos dinamicamente en el shell
  (patron Application Shell: el contenedor orquesta carga/descarga de modulos).
- **Web Components (Lit)** para los controles base compartidos (tabla, filtros,
  detalle, timeline, documentos) que tu shell debe ofrecer.
- **Event Bus** pub/sub para comunicacion entre modulos (coincide con tu bus de
  eventos) y estado compartido centralizado en el shell.

Recomendacion concreta: adoptar **Module Federation + contratos de Web Component
para los controles base**, en vez de imports locales. Imports locales atan el
despliegue de todos los modulos al del shell; Module Federation permite desplegar
un modulo sin redeployar el portal entero, que es el objetivo real de la
modularidad. Tu `VECFrontendModule.mount/unmount` ya es compatible con esto.

## 5. Transversales de cumplimiento que faltan (ALTO)

Una administracion tiene obligaciones que el contrato debe modelar como gates,
no dejarlas a cada modulo:

- **ENS (Esquema Nacional de Seguridad)**: trazabilidad, niveles de seguridad
  por dato. Tu auditoria ayuda, pero anade `gate-ens` (registro de accesos a
  datos personales, no solo de acciones).
- **RGPD / consentimiento y minimizacion**: un `gate-rgpd` que exija base
  juridica declarada por tratamiento y que los modulos no expongan datos
  personales que no necesitan.
- **Accesibilidad (EN 301 549 / WCAG, obligatoria por ley en sector publico)**:
  `gate-accesibilidad` sobre los controles base del shell. Si los controles base
  cumplen, los modulos heredan cumplimiento; por eso debe vivir en el shell.
- **Interoperabilidad ENI**: los documentos del expediente deben seguir el
  Esquema Nacional de Interoperabilidad (metadatos ENI, formatos admisibles,
  CSV). Tu capacidad `documentos` ya menciona ENI; conviene convertirlo en
  `gate-eni` explicito.

## Respuestas sugeridas a tus preguntas abiertas

1. **Manifiesto JSON vs Go struct**: ambos. Go struct como fuente de verdad en
   bootstrap (tipado, validable en compilacion) y export JSON para el frontend y
   para validacion de colisiones. Una sola fuente, dos representaciones generadas.
2. **Frontend**: Module Federation + Web Components base (ver punto 4).
3. **Bus en memoria primero**: si, correcto. Empezar in-memory con la interfaz del
   envelope ya definida y meter adaptador persistente despues sin tocar modulos.
4. **Documentos/Notificaciones/Auditoria, modulos o capacidades**: capacidades
   del shell con pantallas globales, no modulos instalables. Son transversales;
   si fueran modulos, podrian "no instalarse" y romper a todos los demas.
5. **RUM/meritos**: modulo comun `vec.module.rum` consumido por Bolsa y por
   Concursos/CAP. Los meritos se reutilizan entre procesos; duplicarlos en cada
   modulo lleva a baremos divergentes.
6. **Versionado de permisos**: permisos como claves namespaced e inmutables
   (`bolsa.solicitud.read`); nunca se renombran, se deprecan. Perfiles agrupan
   claves. Un modulo nuevo solo anade claves nuevas, jamas cambia el significado
   de una existente.
7. **Prueba minima de modulo**: la que ya propones en "primera entrega" es la
   correcta; anade un caso de "accion que requiere firma exige AuthAssurance
   minimo" y "tramite presentado genera asiento de registro".

## Gates anadidos (sobre tu lista existente)

- `gate-identidad`: soporta SSO + certificado + Cl@ve; declara assurance por accion.
- `gate-firma`: actos administrativos pasan por puerto de firma + sello de tiempo.
- `gate-registro`: tramites presentados generan asiento registral con justificante.
- `gate-ens`: registro de accesos a datos personales.
- `gate-rgpd`: base juridica por tratamiento, minimizacion de datos.
- `gate-accesibilidad`: controles base cumplen EN 301 549 / WCAG.
- `gate-eni`: documentos con metadatos y formatos ENI.

## Prioridad sugerida para la vertical minima

Mantener tu vertical minima, pero anadir al alcance desde el dia uno el contrato
de identidad multi-metodo (punto 1) y el puerto de firma (punto 2), porque son
los dos que mas contaminan el resto si llegan tarde. El registro, ENS, RGPD,
accesibilidad y ENI pueden entrar como gates definidos ahora y cumplidos de forma
incremental, pero **definidos ya** para que ningun modulo nazca incumpliendolos.

## Aislamiento y propagacion de fallos

La regla transversal queda desarrollada en
[`aislamiento_modular_y_dependencias_2026-07-25.md`](aislamiento_modular_y_dependencias_2026-07-25.md):
un fallo solo se propaga a capacidades que declaren depender de la capacidad
fallida. El shell, los despliegues y las raices de composicion no pueden
convertir una dependencia opcional en condicion de arranque global.

## Fuentes revisadas en esta revision

- Portal del Empleo Publico de la Junta de Andalucia y su zona personal /
  metodos de acceso (SSOWeb para empleados, certificado, Cl@ve):
  `https://portalempleopublico.juntadeandalucia.es/acceso/zona-personal`
- Punto de Acceso General de la AGE:
  `https://administracion.gob.es/`
- Arquitectura de referencia Microfrontend (Junta de Andalucia):
  `https://desarrollo.juntadeandalucia.es/recursos/reglas-pautas/arquitectura-referencia-microfrontend`
- Esquema Nacional de Interoperabilidad (ENI), RD 4/2010:
  `https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331`
- Cl@ve, sistema de identificacion y firma del Estado (ejemplo de sede):
  `https://sede.carm.es/web/pagina?IDCONTENIDO=2522&IDTIPO=240`

---

# Niveles de madurez de modulo (H-05)

Esta seccion cierra el hallazgo H-05 de
`docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md`: Bolsa tiene las
cuatro capas hexagonales, Cronos y Personal son parciales, Dietas y
Administracion son solo manifiesto. El contrato debe declarar el nivel de
cada modulo para que el shell VEC no asuma capacidades que el modulo todavia
no implementa.

## Definicion de niveles

- **completo**: existen las cuatro capas (`domain`, `ports`, `application`,
  `adapters`) y cada capa tiene tests propios. El modulo puede ejecutar casos
  de uso reales, persistir estado y exponer efectos auditables.
- **parcial**: existen las cuatro capas, pero al menos una esta debil (sin
  test propio, un unico fichero) o cubre solo un subconjunto de lo que el
  manifiesto anuncia.
- **solo-manifiesto**: unicamente publica `ModuleID`, `Manifest()` con menu y
  permisos. No hay `domain`, `ports`, `application` ni `adapters`; cualquier
  dato adicional que exponga (catalogos, referencias) es estatico, sin caso
  de uso que lo transforme ni puerto que lo persista.

## Que puede asumir el shell segun el nivel declarado

| Nivel | El shell puede | El shell NO puede |
| --- | --- | --- |
| completo | Montar rutas de `api_prefix`, invocar acciones con efectos (crear, aprobar, firmar), suscribir `events_published` y auditarlas, confiar en `health_route`. | — |
| parcial | Mostrar menu y datos de solo lectura de las capas ya implementadas. | Asumir que toda accion listada en el manifiesto tiene caso de uso detras; debe verificarse capacidad por capacidad, no por modulo completo. |
| solo-manifiesto | Mostrar menu, icono, grupo y permisos declarados; reservar hueco en el dashboard. | Enrutar acciones con efectos, invocar `api_prefix`, esperar `events_published` reales, ni dar por buena `health_route`: no hay handler, dominio ni persistencia detras. |

## Clasificacion actual (evidencia de arbol de paquetes)

Evidencia tomada de `internal/modules/*` en este commit: capas presentes y
numero de ficheros `_test.go` por capa.

| Modulo | Nivel | Capas presentes | Tests por capa | Tests totales |
| --- | --- | --- | --- | --- |
| `bolsa` | completo | domain, ports, application, adapters (+ `internal/transaccion`) | domain 3, ports 9, application 15, adapters 9, internal 1 | 39 |
| `personal` | parcial | domain, ports, application, adapters | domain 2, ports 0, application 1, adapters 2 | 6 |
| `cronos` | parcial | domain, ports, application, adapters | domain 2, ports 0, application 1, adapters 1 | 5 |
| `dietas` | solo-manifiesto | ninguna (`manifest.go` + `routes.go` con dataset estatico de municipios/rutas) | — | 2 (`manifest_test.go`, `routes_test.go`) |
| `administracion` | solo-manifiesto | ninguna (solo `manifest.go`) | — | 1 (`manifest_test.go`) |

Notas sobre la evidencia:

- En `personal` y `cronos` la capa `ports` tiene un unico fichero de
  interfaces y ningun test propio: es la capa mas debil de ambos y el motivo
  de clasificarlos como parciales, no completos.
- `dietas/routes.go`, pese al nombre, no es un enrutador HTTP: es un dataset
  de municipios, puntos y distancias de la provincia de Granada pensado para
  una futura funcion de kilometraje. No hay caso de uso que lo consuma
  todavia, asi que no cuenta como capa `application`.
- `administracion` no tiene ni dataset propio: es manifiesto puro.
- Un modulo solo sube de nivel cuando su arbol de paquetes lo respalda; esta
  tabla se actualiza en el mismo commit que anada o complete una capa.
