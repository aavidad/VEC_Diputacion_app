# VEC: requisitos técnicos (TRD)

Fuentes absorbidas el 13/10/2026: `docs/portal_vec/arquitectura_tecnica.md`
(retirado), `docs/portal_vec/cumplimiento_y_seguridad.md` y
`docs/portal_vec/contrato_modulos_vec.md` (reducidos a una nota), los conectores
de `docs/portal_vec/expediente_contratacion_temporal_rrhh.md` (retirado), las
secciones técnicas de `AGENTS.md`, E03 a E10 y E12 de
`ESPECIFICACIONES_AGENTES.md` y las decisiones de identidad, servidores,
auditoría, rendimiento y firma de la memoria de dirección.

TRD quiere decir «documento de requisitos técnicos». Explica cómo está construida
VEC, la aplicación de Recursos Humanos de la Diputación de Granada: con qué
tecnología, cómo se reparte el código, cómo arranca, cómo se identifica y autoriza
a cada persona, qué exige la seguridad y la protección de datos, cómo se prueba y
cómo se sube al servidor.

Comprobado sobre `origin/main@2fbb15342` el 13 de octubre de 2026; la candidata
#1099 se indica aparte en los apartados 9 y 16.

Es la fuente única de las reglas técnicas. Reúne lo vigente de
`docs/portal_vec/arquitectura_tecnica.md`, `docs/portal_vec/cumplimiento_y_seguridad.md`,
`docs/portal_vec/contrato_modulos_vec.md`, los apartados técnicos de `AGENTS.md`
y de `ESPECIFICACIONES_AGENTES.md` (E03 a E10 y E12) y la guía mantenida de
`docs/manual_programador/README.md`. Lo histórico de esos documentos no se ha
copiado.

| Para… | Leer |
| --- | --- |
| Qué es VEC, para quién y con qué prioridad | `docs/producto/PRD.md` |
| Quién hace qué y en qué orden | `docs/producto/FLUJO_APP.md` |
| Cómo se ven y se usan las pantallas | `docs/producto/DISENO_UI_UX.md` |
| Esquemas, tablas, roles de base de datos y rutas de la API | `docs/producto/ESQUEMA_BACKEND.md` |
| Qué falta y en qué orden | `docs/producto/PLAN_IMPLEMENTACION.md` |
| Perfiles, acciones y ámbitos concretos | `docs/producto/FLUJO_APP.md`, apartado 6 |
| Reglas detalladas de PostgreSQL | `docs/portal_vec/seguridad_persistencia_postgresql.md` |
| Índice generado de funciones Go | `docs/manual_programador/LEEME.md` (se regenera con `scripts/generar_manual_programador.py`) |

Si este documento y el código no coinciden, manda el código y, después, las
órdenes vigentes de `AGENTS.md`. Quien encuentre la diferencia la anota en su
entrega y corrige este documento en la misma PR (petición de integración en GitHub).

## 1. Pila tecnológica

| Pieza | Qué se usa | Dónde se fija |
| --- | --- | --- |
| Servidor | Go. `go.mod` declara `go 1.25.12` como mínimo de lenguaje y `toolchain go1.26.9` como compilador exacto | `go.mod`, `Dockerfile` (imagen `golang:1.26.9-bookworm` fijada por huella) |
| Interfaz web | HTML, CSS y JavaScript sin frameworks ni paso de compilación. Única biblioteca de terceros: Leaflet 1.9.4 para mapas | `web/static/`, `web/static/portal-empleado/vendor/leaflet-1.9.4/` |
| Pruebas web | Node 20 con su ejecutor integrado (`node --test`), sin dependencias; unos 480 ficheros `*.test.mjs` | `scripts/verificar_calidad.sh` |
| Base de datos | PostgreSQL 18 (en CI, `postgres:18.4-bookworm` fijada por huella). Conexión desde Go con `pgx/v5` | `.github/workflows/ci.yml`, `deploy/postgresql/` |
| Documentos | PDF con `go-pdf/fpdf`; plantillas Word versionadas por catálogo | `go.mod`, `config/` |
| Almacén de ficheros | Cliente S3 (`aws-sdk-go-v2`) contra un almacén compatible (Ceph) | `deploy/almacen_s3_ceph/` |
| Análisis de ficheros subidos | Proceso aparte con ClamAV en caja aislada y sin red | `cmd/vec-analizador`, `deploy/analizador/README.md` |
| Servidor | cidonia, con los procesos en contenedores podman sin privilegios de administrador (rootless). Solo datos sintéticos | fuera de Git; `docs/manual_sistemas/README.md` |

Una dependencia nueva solo entra si está mantenida, tiene licencia compatible y
reduce un riesgo real. Las imágenes y acciones de CI se fijan por huella, nunca
por etiquetas móviles como `latest`.

### Ejecutables

Todo vive en un solo módulo Go. `cmd/` tiene unos ochenta ejecutables. Los
importantes son:

| Ejecutable | Para qué sirve |
| --- | --- |
| `cmd/vec-server` | Servidor principal: portal de RRHH, portal del empleado, Área personal del aspirante y consulta pública. Lleva subórdenes de operación (`importar-convoca`, `constituir-bolsa`, `publicar-proyeccion-publica`, `comprobar-dietas`, `rellenar-vinculos-bolsa`) |
| `cmd/vec-admin` | Administración en su propio proceso y subdominio, con certificado de cliente obligatorio |
| `cmd/vec-publico`, `cmd/vec-interno` | Composición separada de las superficies pública e interna |
| `cmd/vec-analizador` | Análisis de ficheros subidos (ClamAV y política de PDF) |
| `cmd/vec-auditoria-verificar`, `cmd/vec-auditoria-checkpoint` | Comprobar la cadena de auditoría y sellar puntos de control |
| `cmd/vec-copias-*` | Copias de seguridad, ensayo y restauración |
| Resto de `cmd/vec-*` | Preparación y actos de gobierno: catálogos, perfiles, firmas, provisión de identidades sintéticas |

`cmd/` no es hoy un cliente de línea de órdenes general ni hay cliente de
escritorio. Si se crean, usarán los mismos casos de uso que la web.

`VEC_PORTAL_PROCESO` decide qué portal atiende un proceso de `vec-server`: vacío
para los dos, `interno` (RRHH y empleados) o `externo` (Área personal y consulta
pública). Un valor mal escrito impide arrancar (`config/portal_proceso.go`,
`internal/app/separacionportales/`). El `Dockerfile` construye dos imágenes,
`runtime-publico` y `runtime-interno`, y la CI comprueba que cada una lleva solo
lo suyo.

En HZ37, el proceso combinado conserva el montaje interno de «Mi Bolsa» de
HZ36 cuando faltan `VEC_EXTERNO_BOLSA_DATABASE_URL` y
`VEC_MATERIAL_PORTAL_EXTERNO_DIR` y la inscripción está apagada. Conserva sus
identidades, permisos, ámbitos, rutas y auditoría; una identidad externa no
recibe acceso interno. Si se configura cualquiera de esos dos valores o se
activa la inscripción, se exige el montaje externo completo, incluida su
confianza V3. Una configuración parcial, inválida o sin confianza produce un
error de arranque del componente, sin sustituirlo por el interno. Es una
medida temporal hasta HZ38, cuando P publique la confianza externa propia.
La selección se comparte entre rutas, fronteras y material en
`internal/app/bootstrap/bolsa_portal_combinado.go`.

## 2. Arquitectura hexagonal

VEC sigue la arquitectura hexagonal de forma estricta. Las reglas del negocio
están en el centro. Todo lo que habla con el exterior (HTTP, PostgreSQL, correo,
firma, ficheros) queda en el borde y se puede cambiar sin tocar el centro.

| Capa | Contiene | No puede importar |
| --- | --- | --- |
| `domain` | Estados, invariantes, validaciones y cálculos puros | aplicación, puertos, adaptadores, HTTP, SQL, proveedores |
| `application` | Casos de uso que coordinan dominio y puertos | HTTP, PostgreSQL, Docker, SDK concretos |
| `ports` | Contratos mínimos (interfaces Go y estructuras de datos sencillas) | implementaciones, validación o serialización de negocio |
| `adapters` | Traducción a HTTP, PostgreSQL, SMTP, S3, GrxFirma… | reglas de negocio propias |

Fuera de las cuatro capas está la composición (`internal/app/`, `cmd/`). Es el
único sitio que elige qué adaptador concreto se conecta a cada puerto. La
dependencia apunta siempre hacia dentro: el adaptador implementa el puerto que
pide la aplicación y la composición lo inyecta.

### Dónde está cada cosa

| Ruta | Qué hay |
| --- | --- |
| `internal/vec/` | Núcleo común: identidad, autorización, auditoría, documentos, datos personales, registro de módulos |
| `internal/modules/<módulo>/` | Un directorio por módulo con sus cuatro capas: `administracion`, `aspirantes`, `bolsa`, `calendarios`, `carrera`, `certificados`, `contrataciontemporal`, `cronos`, `dietas`, `formacion`, `meritos`, `personal`, `provision`, `seleccion`, `usuarios` |
| `internal/modules/<módulo>/manifest.go` | Manifiesto del módulo: identificador, menú y permisos que declara |
| `internal/shared/` | Utilidades sin negocio: `i18n`, `telemetria`, `postgresql`, `limiteshttp`, `plazoarranque`, `baremacion` |
| `internal/app/bootstrap/` | Montaje de cada capacidad a partir de la configuración |
| `internal/app/composicion/` | Composición de las superficies `interna`, `publica` y de administración |
| `internal/app/server/` | Servidor HTTP común: estáticos, salud, cabeceras de seguridad, límites, rechazo de cookies |
| `config/` | Variables de entorno y catálogos versionados embebidos (`*_v1.json`, `*_v2.json`) |
| `deploy/postgresql/<esquema>/migraciones/` | Migraciones SQL por propietario |
| `web/static/` | Interfaz web y catálogos de textos |

Ejemplo real, Contratación temporal: `internal/modules/contrataciontemporal/domain/`
tiene el expediente y sus estados; `application/` los casos de uso
(`cancelacion_expediente.go`, `cese_cierre_modificacion.go`, `confirmacion_ginpix.go`);
`ports/` lo que esos casos piden; `adapters/` las piezas `postgres`, `httpapi`,
`smtp`, `ginpixapi`, `ginpixfichero`, `firmaemisorv2`, `auditoria` y otras.

### Reglas de dependencia

1. **Cada módulo es dueño de sus datos.** Ningún módulo lee ni escribe tablas de
   otro. Se intercambian referencias opacas, órdenes, eventos y registros de
   salida (outbox). Contratación temporal coordina; Bolsa conserva convocatorias,
   candidaturas, orden y llamamientos; Personal conserva relaciones de servicio,
   ocupación e incorporación. Quien necesita un dato ajeno pide una proyección
   autorizada a su dueño.
2. **Persona es la entidad común.** Todo empleado es persona; un aspirante
   externo no necesita ser empleado. Una cuenta, un certificado o un DNI no
   prueban una relación de servicio. El `ContextoActor` enlaza la persona con sus
   referencias de candidato o empleado, con versión y vigencia.
3. **El núcleo no depende de los módulos.** Lo comprueban pruebas que leen los
   `import`: `internal/vec/application/arquitectura_dependencias_test.go`,
   `internal/vec/adapters/httpapi/composicion_arquitectura_test.go`.
4. **Una regla, todos los canales.** Web, escritorio, línea de órdenes, MCP
   (protocolo con el que un asistente de IA usa herramientas), tareas programadas
   y llamadas entre servicios usan los mismos casos de uso. Ninguna regla vive en
   HTTP, en el DOM ni en el navegador.
5. **Autoridades únicas.** Identidad, perfiles, permisos, auditoría, i18n y tema
   tienen una sola autoridad común. Ningún módulo crea la suya, ni su propio
   inicio de sesión, ni su propia ficha del empleado.
6. **Adaptadores intercambiables.** Base de datos, almacén, firma, antivirus,
   identidad, correo, documentos, calendarios y sistemas externos son
   adaptadores. La composición real nunca conecta un doble de prueba ni un
   adaptador ficticio.
7. **Lo no decidido se configura.** Fases, plazos, plantillas, formatos,
   categorías, motivos y reglas van en catálogos versionados. Solo las
   invariantes técnicas se escriben en el código. Un expediente conserva la
   versión del catálogo que se le aplicó; una versión nueva no reinterpreta su
   historia.
8. **Lo justo, sin capas de más.** Cada capa separa una responsabilidad real. Se
   añade un puerto solo cuando hay un consumidor concreto. No se reestructura el
   código para parecerse a ejemplos antiguos.

## 3. Módulos

VEC crece como un sistema de módulos registrados, sin cargar código en caliente
ni exigir microservicios. Añadir un módulo es incorporar un paquete revisado,
registrarlo en la composición, compilar y, si hace falta, instalar sus
migraciones y activar su configuración. Separar un módulo en otro despliegue solo
se hace si lo justifica la red, el aislamiento de fallos o su ciclo de vida.

### Lo que existe

| Mecanismo | Dónde | Límite |
| --- | --- | --- |
| Manifiesto | `internal/vec/domain/types.go` (`ModuleManifest`) y `manifest.go` de cada módulo | Describe menú y permisos que necesita. No concede nada |
| Registro | `internal/vec/application/service.go` (`RegisterModule`, `Modules`, `BuildMenu`) | No es instalación durable ni descubrimiento automático |
| Composición | `internal/app/bootstrap/bootstrap.go` y montajes por capacidad | Registrar un manifiesto no conecta rutas, autoridad ni almacenamiento |
| Catálogo web | `web/static/portal-empleado/portal-catalogo-modulos.js` lee `/api/vec/modules` | Construye la navegación; no da permisos |
| Montaje web | `portal-modulos-coordinador.js` con imports locales revisados | No se ejecuta JavaScript ni SQL recibido de un catálogo o un formulario |

### Lo que debe traer un módulo nuevo

| Parte | Condición |
| --- | --- |
| Identidad y versión | Identificador estable; versión del paquete y del contrato separadas; rutas y claves de texto sin colisiones |
| Dominio y aplicación | Casos de uso propios, puertos pequeños; no importa adaptadores ni tablas de otro módulo |
| Montaje en el servidor | Constructor en composición con dependencias explícitas; rutas en la autoridad de rutas existente; nada de autorregistro global oculto |
| Montaje web | Entrada local de montar y desmontar, con limpieza de escuchadores y foco; cliente HTTP inyectado |
| Autorización | Acciones, ámbitos, finalidad y campos concretos, concedidos por el sistema común |
| Persistencia | Esquema, roles y migraciones propios; recibos, historia y repetición segura. La aplicación no migra al arrancar |
| Dependencias | Obligatorias u opcionales por capacidad, con su conducta ante fallo |
| Intercambio | API o puerto; evento versionado solo si hay consumidor. Datos mínimos y referencias |
| Textos y aspecto | Catálogos comunes, tema común, estados accesibles; ni menú propio ni textos sueltos |
| Operación | Configuración privada, salud por capacidad, registros sin datos sensibles |
| Retirada | Desactivar rutas y tareas conservando datos, documentos y recibos. Desinstalar nunca ejecuta un DOWN destructivo |

### Aislamiento de fallos

Un fallo solo alcanza a las capacidades que declaran depender de la que falla. La
caída de Formación no impide consultar un contrato; la caída de identidad o
autorización sí bloquea lo que las necesita. Un módulo ausente no se sondea en
bucle ni deja botones activos sin servidor detrás. Detalle en
`docs/portal_vec/aislamiento_modular_y_dependencias_2026-07-25.md`.

Una capacidad se describe separando código, composición, instalación, recorrido
comprobado y aprobación funcional. Tener cuatro capas, un manifiesto o pruebas no
acredita que esté disponible.

## 4. Composición, configuración y arranque

### Cómo arranca `vec-server`

`cmd/vec-server/main.go` hace, en orden:

1. Atiende las subórdenes de operación, si se pidió alguna, y termina.
2. Lee la configuración (`config.Load()` en `config/config.go`).
3. Crea el emisor de incidencias técnicas. Si no puede, usa uno nulo: la
   supervisión nunca impide arrancar (`cmd/vec-server/incidencias_arranque.go`).
4. Compone el servidor (`bootstrap.NuevoServidorHTTPSupervisado`). Cada
   capacidad se monta en su fichero de `internal/app/bootstrap/`.
5. Monta el registro técnico de acceso (apartado 10) y la supervisión de
   errores 5xx.
6. Escucha con TLS si hay certificado y clave.

Cada fase deja una línea de arranque con fase, resultado y causa, sin rutas, DSN
(cadenas de conexión a la base) ni datos de configuración.

### Configuración básica

| Variable | Por defecto | Uso |
| --- | --- | --- |
| `VEC_HTTP_ADDR` | `127.0.0.1:8080` | Dirección de escucha; solo local por defecto |
| `VEC_HTTP_ALLOWED_CIDRS` | solo local (IPv4 e IPv6) | Redes desde las que se aceptan peticiones |
| `VEC_AUTH_MODE` | `disabled` | Identidad apagada hasta configurar una frontera admitida |
| `VEC_TLS_CERT_FILE`, `VEC_TLS_KEY_FILE` | vacías | Van juntas; si solo hay una, no arranca |

Cada variable se define y normaliza en `config/`. La configuración privada
(conexiones, claves, listas de personas) vive fuera de Git, en ficheros del
servidor con permisos `0600`.

Salud del proceso: `GET /livez` (el proceso vive) y `GET /readyz` (sus
dependencias responden); `/healthz` es un alias de `/readyz`
(`internal/app/server/server.go`).

La confianza del portal externo se prepara administrativamente antes de arrancar
la web. `preparar-portal-externo --publicador-confianza-dsn <fichero privado>`
publica una raíz y claves derivadas del material externo, mediante un LOGIN
nominal con un único grupo y sin acceso al propietario AD3. Conserva el inventario
`externo/v3` y la auditoría común; la web no recibe ese DSN. Una tarea administrativa
separada comprueba cada quince minutos y renueva antes de caducar, conservando
las claves del portal ya arrancado. Orden de instalación
y uso: `deploy/principal/confianza_externa_hz38.md`.

### Selectores `VEC_*_ENABLED`

Cada capacidad que necesita algo más que el núcleo se enciende con un selector de
entorno `VEC_<CAPACIDAD>_ENABLED=true`. Hay unos treinta. Apagado, la capacidad
no se compone y sus rutas responden cerradas. Ejemplos:
`VEC_DOCUMENTOS_ENABLED`, `VEC_BOLSA_BORRADORES_ENABLED`,
`VEC_BOLSA_PORTAL_CANDIDATO_ENABLED`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED`,
`VEC_FIRMA_VERIFICACION_ENABLED`, `VEC_RRHH_AUDITORIA_ENABLED`.

Las capacidades de Bolsa y Contratación temporal se componen solo con la «doble
llave de desarrollo»: `VEC_EXECUTION_PROFILE`, `VEC_AUTH_MODE` y su
reconocimiento expreso `VEC_DEVELOPMENT_GUARD` (`config/desarrollo.go`). cidonia
funciona así porque todos sus datos son sintéticos. La composición productiva
sin esa llave está pendiente.

El inventario de variables de Bolsa y Contratación temporal, con lo que cada una
necesita, está en `config/selectores_dependencias_bolsa_ct_v1.json` (unas cien:
selectores, conexiones, catálogos y ajustes). Al arrancar,
`internal/app/bootstrap/selectores_dependencias_aviso.go` deja un aviso por cada
dependencia que falte. Solo avisa: no detiene nada y nunca escribe valores.

Selector y permiso son cosas distintas. El selector decide qué se compone; el
permiso de cada persona se decide en cada petición. `VEC_PORTAL_MODULOS_VISIBLES`
limita qué módulos ofrece el menú y tampoco concede acceso
(`config/portal_modulos.go`). Alberto decidió el 25/09 que activar o desactivar
un módulo se haga desde Administración, auditado. Ese interruptor aún no existe y
hoy se usa la variable.

### Regla: ninguna configuración nueva tumba el arranque

Si falta o está mal la configuración de una función, falla solo esa función. La
aplicación arranca y el resto sigue funcionando. La función afectada responde
cerrada (404 o 503) y, en pantalla, explica que no está disponible.

- Una capacidad nueva lee su configuración de un fichero privado propio (por
  ejemplo `VEC_ADMIN_VERSION_BOLSA_CONFIG_FILE`). Sin él, sus rutas quedan
  cerradas (`deploy/principal/habilitacion_b1_convoca_20261009.md`).
- Una dependencia opcional nunca se convierte en condición de arranque global.
- Lo que exija configuración o actos previos se prepara y se ensaya en una copia
  antes de subirlo, no a ciegas en la principal.

Cronos comprueba su configuración y conexiones antes de añadir sus audiencias
al gobierno común. Si falla una dependencia propia, cierra sus rutas con 503
y conserva el montaje del resto de la aplicación. Su material se publica en
una transacción, al final de la preparación de Contratación, con plazo propio.
El selector apagado no lee el fichero privado ni abre conexiones de Cronos
(`internal/app/bootstrap/cronos_arranque.go`).

**Lo que falta.** El código no cumple la regla en todos los casos. `main.go`
termina el proceso (`log.Fatalf`) si falla la composición de cualquier
componente. El 13/10, `main` no arrancaba en cidonia porque fallaba el componente
`inscripcion_bolsa_externa`. La subida volvió atrás sola, pero la regla se
incumplió. Falta revisar cada montaje para que un error propio cierre solo su
función.

## 5. Identidad y sesión

### Quién entra y cómo

| Quién | Cómo se identifica hoy | Previsto en producción |
| --- | --- | --- |
| Personal interno (RRHH, centros, Intervención, empleados) | Certificado personal como único factor. Excepción aprobada por Alberto el 24/09, solo en desarrollo y para la presentación | Active Directory corporativo (Kerberos) más certificado. Lo montado para pruebas en cidonia se retira |
| Administración (`vec-admin`) | mTLS (TLS mutuo: el navegador también presenta un certificado), en su propio subdominio, con lista privada de personas autorizadas | Solo desde la intranet, con DNIe o certificado admitido y permiso administrativo explícito |
| Aspirantes externos | Cuenta, LOGIN de base de datos y autoridad del portal externo, separados de los internos | Certificado, DNIe o Cl@ve, según lo admita la sede |
| Consulta pública | Sin identificación; solo datos públicos gobernados | Igual |

Un certificado válido no concede permiso por sí solo. SSH es acceso al servidor,
no un factor de entrada al portal.

La excepción del certificado como único factor (24/09) usa una autoridad de
certificación de desarrollo propia, guardada fuera de Git, con un certificado
por persona inventada. Se declara con su garantía real, con su alcance y su fecha
de retirada, y no se extiende a producción sin volver a preguntar a Alberto. Al
pasar a producción se retiran la autenticación básica del proxy de entrada y los
certificados de desarrollo por perfil. Una misma persona no comparte cuenta entre los
portales interno y externo.

### Garantía de la identidad

Cada identidad lleva su método y su nivel de garantía (`AuthAssurance` en
`internal/vec/domain`). Las acciones con efecto jurídico (presentar una
solicitud, aceptar o renunciar, firmar) exigen garantía alta, y la autorización
lo comprueba. Se declara la garantía real del factor usado, no una «alta» por
equivalencia. **Identificarse no es firmar**: la firma de un documento es un acto
aparte (apartado 9).

### Cómo llega la identidad al código

1. El certificado se comprueba en el canal TLS. El proxy de entrada pasa una
   aserción de identidad atestada y la aplicación la verifica
   (`internal/app/composicion/interna/certificado_personal_temporal.go`). Un
   registro privado liga la huella del certificado a una cuenta, se coteja con la
   lista de revocación (CRL) y se relee en cada petición: una retirada surte
   efecto sin reiniciar.
2. Con la cuenta se construye el `ContextoActor`
   (`internal/vec/domain/contexto_actor.go`): persona, perfil activo y una
   instantánea con vínculos y versiones para auditar y revalidar. La persona y sus
   ámbitos salen de PostgreSQL (`deploy/postgresql/contexto_actor_v1/`,
   `identidad_sesiones_v1/`), nunca de lo que envía el navegador.
3. Cada persona actúa con un solo perfil activo. Los perfiles no se suman.

### Sin cookies ni almacenamiento del navegador

- La API rechaza toda petición con `Cookie` o credenciales guardadas por el
  navegador, y ninguna respuesta emite `Set-Cookie`
  (`internal/app/server/server.go`, `prohibirCookiesYAutorizacionProxyConLimite`).
- `localStorage`, `sessionStorage` e IndexedDB no guardan sesión, credenciales ni
  permisos. No van tokens en la URL.
- Cabeceras libres, parámetros, cookies o campos JSON no identifican a nadie.

**Decidido, pendiente de comprobar.** El 07/10 se acordó acelerar las lecturas
reutilizando la sesión durable por huella del certificado, sin token ni cookie
(Alberto: «no quiero cookies, que son inseguras»). Se reutiliza la sesión durable
que ya existe, más una caché en memoria de las sesiones recientes indexada por la
huella del certificado. La revocación se garantiza cotejando versiones en la
transacción de lectura. Las lecturas comprueban permisos y dejan apunte de
auditoría nominal; los actos con efecto siguen con decisión completa en la misma
transacción. No se crean fachadas de permisos nuevas para cada lectura. Su estado está en
`docs/producto/PLAN_IMPLEMENTACION.md`.

## 6. Autorización

Se deniega por defecto. Solo autoriza una concesión central positiva, exacta y
vigente para actor, perfil, acción, recurso, ámbito, finalidad, campos y
obligaciones. Ausencia, lista vacía, ambigüedad, valor desconocido, caducidad,
error o un servicio caído deniegan. La indisponibilidad nunca se interpreta como
permiso.

- No hay administrador universal, comodines positivos, permisos implícitos ni
  suma de perfiles.
- Menú, ruta, manifiesto, rol, cabecera, titularidad o prefijo de un
  identificador sirven para presentar y dar contexto. Nunca conceden acceso.
- El caso de uso revalida la autorización justo antes del efecto, también al
  repetir una operación. Una concesión histórica o una referencia opaca no dan
  acceso por sí solas.
- Se conserva la separación de funciones: preparar, revisar, publicar, firmar y
  auditar requieren las personas y competencias de cada circuito. Una misma
  persona no cubre un doble control cambiando de perfil.

| Término | Qué es |
| --- | --- |
| RBAC | Control por roles: el perfil debe tener la concesión. Es condición necesaria |
| ABAC | Control por atributos (unidad, centro, fase…). Solo puede restringir lo que da el rol |
| PDP | Punto de decisión de políticas: el componente central que decide sí o no |
| V3 | Tercera versión de la autorización atestada. El PDP emite una capacidad breve, protegida con MAC (código de autenticación del mensaje) y ligada a la operación exacta; PostgreSQL la verifica y la consume (`deploy/postgresql/autorizacion_atestada_v3/README.md`) |
| Consumidor nominal | Función SQL por operación que acepta solo su audiencia y su acción y registra el consumo |
| LOGIN | Rol de PostgreSQL que puede conectarse. Cada capacidad usa el suyo, con permisos mínimos |

### Perfiles y concesiones

Los perfiles (RRHH, centro, ratificador, Intervención, aspirante, ADMIN…) son
versiones de rol con una lista cerrada de acciones. Las acciones son claves con
espacio de nombres (por ejemplo `bolsa.carga_convoca.confirmar`); una clave
publicada no cambia de significado: se publica otra. Asignar o retirar un perfil
deja historia y auditoría. Cambiar las acciones de un rol publica una versión
nueva, con plan y aprobación ligados a una huella (AUT45, AUT48, AUT64 en
`deploy/postgresql/autorizacion/`). Aprobar o liquidar lo hace un perfil fijo
asignado por Administración, no un acto por persona. El gobierno de perfiles está
en `docs/plan_modulos/administracion.md`.

La administración de sistema (cuentas, claves, conectores, despliegue) y el
gobierno funcional de RRHH (catálogos, reglas, plantillas) son
responsabilidades distintas. Operar la plataforma no da acceso a expedientes ni a
decisiones de RRHH. La superficie de administración está segregada del portal de
RRHH y del público; ocultar un menú o devolver 403 no prueba esa segregación.

### Actos ADMIN con dos administradores

Los cambios de gobierno sensibles exigen dos personas ADMIN distintas. La primera
presenta una propuesta. La segunda la aprueba con la referencia y la huella de
esa propuesta. Dos cuentas o dos certificados de la misma persona no valen.
Ejemplo real: la versión del rol de Bolsa (`/api/admin/perfiles/v1/gobierno-version-bolsa/propuestas`
y `/cierres`, en `deploy/principal/habilitacion_b1_convoca_20261009.md`). La
restauración de copias sigue el mismo patrón (`docs/plan_modulos/copias_seguridad.md`).

### Lecturas y actos

- **Actos con efecto.** Decisión V3 consumida en la misma transacción que escribe
  el estado (apartado 7).
- **Lecturas.** Hoy también pasan por fachadas V3 nominales y eso las encarece.
  El objetivo acordado es comprobarlas contra la sesión, con su apunte de
  auditoría (apartado 5).

La consulta histórica de firmas originales de Contratación temporal (F1-36,
en preparación) parte de `expediente.consultar` para la ficha. La versión
ampliada de la decisión concede exactamente el campo
`selectores_recuperacion_firmas`; `[]` no es un comodín. Además comprueba una
asignación propia y vigente de un cargo de firma, con organización y unidad
compatibles con las de la firma original. Esa segunda comprobación solo reduce
el resultado: no concede acceso a la ficha, no cambia el perfil activo y no
exige que consulte quien firmó entonces. La ausencia de unidad histórica
deniega la recuperación. No se usa el plan de firma actual para reinterpretar
la firma original.

## 7. Efectos duraderos: auditoría, outbox, idempotencia y versión

Todo efecto escribe, en **una sola transacción** de PostgreSQL:

1. el consumo de la autorización;
2. el cambio de estado, con su versión nueva;
3. la historia, solo por adición: nunca se borra ni se reescribe;
4. el apunte de auditoría común, encadenado con el anterior;
5. el registro de salida (outbox) para avisar a otros módulos o sistemas;
6. el recibo que vuelve a la pantalla.

Si algo falla, no queda nada a medias. Una tabla local de cambios no sustituye a
la auditoría común.

| Mecanismo | Qué garantiza | Cómo se usa |
| --- | --- | --- |
| Auditoría común | Quién hizo qué, sobre qué, cuándo, con qué resultado y versión, también en lecturas e intentos denegados. Encadenada y sellada para que valga como prueba ante un juzgado | `internal/vec/auditoria/`, `cmd/vec-auditoria-verificar`, `docs/plan_modulos/auditoria.md` |
| Outbox | Lo que hay que comunicar fuera queda escrito con el efecto y sale después, sin perderse ni duplicarse | Por consumidor: Bolsa reserva el envío antes de llamar al SMTP y registra el resultado (`deploy/principal/README.md`). No hay un repartidor genérico único |
| Idempotencia | Repetir la misma operación devuelve el mismo recibo sin otro efecto | Campo `clave_idempotencia` u `operacion_ref`, o cabecera `Idempotency-Key`. Misma clave con otro contenido: 409 |
| Versión optimista | Nadie pisa un cambio ajeno sin enterarse | Campo `version_esperada`. Si ya cambió: 409 y se relee |

Cada apunte de auditoría lleva actor, perfil, método de identificación, acción,
recurso, instante, correlación, versión y resultado (aceptado, rechazado, error
funcional, error técnico). Lleva la huella o la referencia de lo afectado, nunca
secretos ni documentos personales completos. Se auditan como mínimo: altas y
cambios de solicitudes y méritos; carga, validación o rechazo de documentos;
cálculo de baremo con la versión de reglas; publicación de listados;
consultas de expedientes y de datos de contacto; notificaciones; ejercicio de
derechos de protección de datos; cambios de permisos, configuración y claves. El
destino de la auditoría está separado y quien configura una política no puede
alterar su evidencia.

La orden de Alberto (03/10) es guardar todo lo que hace cada usuario, también
consultas y descargas de datos personales, por si hay que revisarlo en un
juzgado. La cadena de huellas se sella periódicamente con una autoridad de
sellado de tiempo (TSA). La exportación sale firmada y sellada, con el método
para comprobarla, y la propia exportación queda auditada. La conservación va por
serie documental (duda 84). Lo lleva el equipo L; el estado está en
`docs/plan_modulos/auditoria.md`.

Una recuperación reutiliza la operación original; no se crea otra para aparentar
éxito. Tras reiniciar la aplicación y PostgreSQL, se deben ver los mismos
recibos y fechas y nada duplicado. Una repetición o un reinicio nunca duplican
efectos, recibos ni comunicaciones.

En F1-36, la lectura ampliada del detalle deja un anexo privado, inmutable y
uno a uno con el acceso y el recibo de la ficha. El anexo conserva el conjunto
exacto de selectores ofrecidos, su decisión y la referencia y huella de la
asignación comprobada. Se verifican el sello del detalle y el del anexo antes
de confirmar la transacción. Un recibo anterior de ficha acredita aquella
lectura, pero no prueba los selectores nuevos. Esta ampliación sigue pendiente
de ensayo SQL, revisión independiente e integración.

## 8. Protección de datos

- **Solo datos sintéticos**, con nombres verosímiles, hasta cerrar la EIPD
  (evaluación de impacto en protección de datos), la categorización del ENS
  (Esquema Nacional de Seguridad), el análisis de riesgos, el registro de
  actividades de tratamiento y las autorizaciones formales. Los datos reales no se
  copian a desarrollo.
- **Minimización.** Listados, auditoría y eventos usan referencias opacas y los
  datos justos. Los datos especialmente protegidos (por ejemplo, discapacidad)
  solo si la convocatoria lo exige y con medidas reforzadas.
- **Contacto de candidatos**: enmascarado en listados y completo en la ficha, con
  constancia de quién lo consultó.
- **Internos y externos separados.** Datos de trabajadores y de aspirantes no se
  mezclan; los traspasos (de bolsa a interinidad, de selectivo a plaza) son
  operaciones gobernadas.
- **Registros técnicos seudonimizados**: sin DNI completo, teléfonos, correos ni
  contenido de documentos.
- **Derechos y conservación.** Acceso, rectificación, oposición, limitación y
  supresión con circuito trazable; plazos de conservación por fase; borrado o
  anonimización solo tras cumplir la normativa de archivo. Estos circuitos aún no
  están construidos.
- **IA cerrada** en selección, empleo, baremación, evaluación o asignación hasta
  cumplir el Reglamento europeo de IA. El asistente público solo usa contenido
  público gobernado.

Las decisiones pendientes del DPD (delegado de protección de datos) están en
`dudas.md` y en `docs/producto/PLAN_IMPLEMENTACION.md`, apartado 11.

## 9. Documentos, firma y expediente electrónico

| Requisito | Estado |
| --- | --- |
| Cada documento tiene identificador estable, huella SHA-256, origen, fecha, actor, formato y metadatos ENI (Esquema Nacional de Interoperabilidad) | Modelo en `internal/vec/domain/documentos.go` (`MetadatosENI`) y servicio en `internal/vec/documentos/` |
| Un fichero subido pasa por cuarentena y análisis; si no sale limpio, no se usa ni se exporta | `cmd/vec-analizador` con ClamAV y política de PDF (`deploy/analizador/README.md`) |
| Una versión nueva del contenido es otro documento; la anterior se conserva | Referencia por versión en `internal/vec/domain/documentos_logicos.go` |
| La firma registra firmante, mecanismo, referencia y fecha, y solo cuenta tras verificación positiva | Circuito con GrxFirma (apartado 15) |
| La versión de baremo o regla aplicada queda congelada en el expediente | Patrón general de catálogos versionados |
| Documentos publicados o entregados, accesibles | Pendiente de comprobar en los PDF generados |
| CSV (código seguro de verificación), sello de tiempo, registro electrónico de entrada, notificación electrónica, índice del expediente firmado y exportación ENI completa | **Pendientes.** No hay puertos de registro ni de notificación electrónica |

La candidata #1099 genera los PDF nuevos con cabecera 1.4 y dos elementos
`/ID` estables en el productor común de Documentos. Conserva el contenido y los
desplazamientos de los objetos; los documentos ya custodiados se recuperan con
sus bytes originales. El plan nominal exige la referencia completa del circuito
vigente, con versión y huella. El guion
`scripts/operacion/firma_ct_tres_pasos/publicar_plan_v2.sh` prepara el acto con
`vec-plan-firma-validar` y usa el gobierno ADMIN existente para crear y publicar
con personas distintas; `--comprobar` no conecta con ADMIN. Dirección publicará
el plan y hará la comprobación final con los certificados que conserva en
cidonia. La prueba local de firma y sus límites se documentan en
`deploy/principal/candidata_hz39/firma/LEEME.md`; no acredita su instalación
en cidonia.

En el registro múltiple, cada PDF firmado tiene su propio identificador. Las
altas del contrato V2 usan la secuencia global del documento como versión de
custodia, sin reiniciarla tras una devolución. Un reintento conserva exactamente
la versión ya guardada, incluida la versión 1 de operaciones antiguas. El
contrato V1 mantiene la versión 1. Dos documentos del circuito no pueden
resolver al mismo tipo documental canónico.

El guion de certificados de desarrollo prepara por defecto certificados de
400 días y una lista de revocación de 90 días. Los parámetros `--cert-dias` y
`--crl-dias` permiten cambiar esos plazos. La CA debe cubrirlos; la lista nueva
conserva sus revocaciones y la reutilización comprueba el material privado.
Los cargos del kit están vigentes hasta el 31 de diciembre de 2027. La ventana
entre preparar y aprobar su publicación sigue limitada a un día.

Un borrador no es un documento firmado. Autenticarse no es firmar. Un aviso no es
una entrega acreditada. Una referencia a un correo no es su custodia. No se
inventan plazos, nombramientos eficaces ni incorporaciones.

## 10. Rendimiento

| Objetivo | Medida |
| --- | --- |
| Lecturas de pantallas (listas, fichas, cuadros) | Menos de 300 ms en el percentil 95 (p95: el 95 % de las peticiones tarda menos), medido por HTTP en cidonia con volumen real |
| Base común (sesión, módulos, preferencias) | Menos de 100 ms |
| Consultas por petición | Sin N+1: nunca una consulta por cada fila de una lista. Se lee en bloque y se pagina |
| Volumen | Miles de registros, con carga progresiva |
| Pantalla útil | Menos de 1 s. El portal pinta primero y cada módulo carga sin bloquear a los demás |

Regla para todo módulo (Alberto, 06/10: «tiene que ser instantáneo»; «se van a
meter miles de personas en un momento»):

- Toda PR que añada o cambie una ruta mide con volumen realista (`EXPLAIN
  ANALYZE` y tiempo) o fija el número de consultas que hace.
- Nada de N+1, tampoco en el consumo de permisos por fila: lotes e índices.
- Prueba de carga en local a cientos y miles de usuarios antes de cada hito.
- Nunca se gana velocidad quitando permisos ni auditoría.
- Cada error técnico deja en el registro su causa interna y la etapa en que
  ocurrió; nunca en la respuesta. Métricas y perfiles solo en superficie interna,
  para Sistemas. PostgreSQL con `pg_stat_statements` y
  `log_min_duration_statement`.

Telemetría por petición (`internal/shared/telemetria/`):

- Una línea JSON por petición con ruta sin valores, estado, duración y bytes. No
  guarda identidad, cuerpos ni cabeceras. Es distinta de la auditoría de datos.
- `VEC_TELEMETRIA_LENTA_MS` marca como lenta una petición a partir de 300 ms por
  defecto.
- `VEC_TELEMETRIA_LENTA_CONSULTAS` avisa cuando una petición pasa de 20
  consultas, señal típica de un N+1.
- El trazador de `pgx` cuenta consultas y tiempo de base de datos de cada
  petición (`telemetria/bd.go`) si la consulta usa el contexto de la petición.
- Consultas para Sistemas: `deploy/principal/consultas_observabilidad.sql`.
  Pruebas de carga: `scripts/carga/`.

**Lo que falta.** El cuadro de Contratación temporal medía entre 363 y 372 ms de
p95 el 08/10, por encima del objetivo. Faltaba medir todas las lecturas en
cidonia (`docs/producto/PLAN_IMPLEMENTACION.md`, fase 1, condición 4).

## 11. Seguridad

### Reglas del código

- Límites de tamaño, tiempo, profundidad y cardinalidad **antes** de reservar
  memoria o llamar a un conector.
- Contexto y cancelación en toda frontera lenta (base de datos, red, ficheros).
- Copias defensivas al cruzar límites con datos mutables.
- Los errores que ve el cliente no revelan detalles internos. Los registros
  tampoco llevan rutas, DSN ni datos de configuración.
- Ningún secreto, certificado, clave, token, DSN, dato personal real ni ruta
  privada entra en Git, pruebas, errores o registros.
- Cifrado en tránsito (TLS) y en reposo para documentos, copias y secretos.
  Claves fuera del proceso y rotables. No se regeneran identidades ni claves al
  arrancar.
- Ninguna regla ordinaria acepta código o SQL enviado desde un formulario.

### Defensas que ya están

- **HTTP** (`internal/app/server/`): límite de cuerpo, rechazo de rutas
  escapadas, de trailers y de cookies, política de seguridad de contenido (CSP)
  en los estáticos, escucha solo local por defecto y lista de redes admitidas.
- **PostgreSQL**: toda función `SECURITY DEFINER` nueva o rehecha fija
  `SET search_path = pg_catalog, pg_temp`, comprobado por
  `scripts/verificar_search_path_definer.py`. Cada capacidad usa su LOGIN con lo
  mínimo. Detalle en `docs/portal_vec/seguridad_persistencia_postgresql.md`.
- **Fallos silenciosos**: `tools/vecsilencio` impide descartar un error sin
  registrarlo, fuera de una lista base que solo puede bajar.

### Herramientas

| Herramienta | Uso |
| --- | --- |
| Gitleaks | Busca secretos en cada envío; primera puerta de la CI (`.gitleaks.toml`). En local, `scripts/verificar_secretos_git.sh` |
| govulncheck | Vulnerabilidades conocidas en dependencias Go, dentro de la puerta de calidad |
| Semgrep | Análisis estático con reglas locales (`semgrep --config auto`, o `p/golang` y `p/javascript`) sobre lo cambiado antes de cada PR. Nunca se sube código a su nube. No hay configuración propia en el repositorio |
| Skill `security-audit` | Revisión centrada en lo cambiado antes de cada PR que toque autenticación, permisos, SQL, datos personales, ficheros subidos, red o entradas HTTP. Auditoría completa solo cuando la pida Alberto o antes de un hito grande |
| ZAP, Nuclei, Trivy y similares | Reservadas para cuando la aplicación esté terminada |

Los cambios de identidad, autorización, criptografía, SQL y fronteras de datos
personales necesitan dos revisiones independientes de la versión final. Un cambio
posterior reabre la parte afectada.

## 12. Textos, idiomas y accesibilidad (parte técnica)

i18n significa internacionalización: preparar la aplicación para varios idiomas.

- Ningún texto visible vive en el código, ni en Go ni en JavaScript: tampoco
  mensajes de validación, estados, ayudas, documentos ni notificaciones.
- Los textos están en catálogos JSON por idioma: `web/static/textos/es/` y
  `web/static/textos/en/` (unos 165 ficheros por idioma). `web/static/textos/idiomas.json`
  fija el idioma por defecto (`es`) y sigue el idioma del navegador.
- Go carga los catálogos con `internal/shared/i18n` y busca por idioma y clave.
- Fechas, números, moneda, zonas horarias y plurales se formatean según el
  idioma. Los datos se escapan al pintarlos.
- La ayuda solo aparece tras un botón «?», nunca como texto fijo.
- Puertas: `scripts/verificar_i18n.py` y `scripts/verificar_i18n_catalogos.mjs`.
- Accesibilidad desde el diseño, según WCAG y la norma EN 301 549: teclado, foco
  visible, contraste, zoom, lector de pantalla, errores junto al campo, mensajes
  que no dependan solo del color, y estados vacío, carga, error y denegación.
- El tema común es la única autoridad visual; ningún módulo duplica CSS
  estructural. Las reglas de pantalla están en `docs/producto/DISENO_UI_UX.md`.

Los nombres del código van en castellano coherente, salvo términos técnicos
universales y convenciones de Go. No se mezcla castellano e inglés en el mismo
vocabulario.

## 13. Calidad del código

- Comentarios de documentación donde la intención o el contrato no sean obvios.
- Sin funciones monolíticas ni estado global mutable.
- Tamaño de fichero: objetivo de 500 líneas y aviso a partir de 800. No es un
  tope: un fichero cohesionado de 1.000 líneas es mejor que tres de 400 que solo
  se entienden juntos. Al pasar del aviso se congela el tamaño en
  `scripts/tamano_ficheros_base.txt` y se justifica en una línea del commit
  (`scripts/comprobar_tamano_ficheros.sh`).
- Antes de escribir, buscar lo que ya existe (índice de código, `gopls`, ramas
  abiertas) y reutilizarlo.
- Todo es código de producción. Solo los datos son sintéticos. No hay código de
  demostración ni marcas de «ejemplo», «prueba» o «demo» en nada nuevo.
- Fechas civiles (plazos, vigencias, nacimientos) se guardan como fecha, no
  como medianoche UTC, y cada intervalo declara si su fin es inclusivo o
  exclusivo. Dinero y distancias se calculan con aritmética exacta, nunca con coma
  flotante.
- Commits pequeños y coherentes, con autor `aavidad`; cada uno compila y pasa sus
  pruebas focales. No se deja una API pública a medias, una migración sin
  consumidor o una ruta sin autoridad.

## 14. Pruebas e integración continua

### Las cinco puertas de la CI

La CI (integración continua) es la batería automática que GitHub ejecuta en cada
PR y en cada envío a `main` (`.github/workflows/ci.yml`):

| Puerta | Qué comprueba | Duración aproximada |
| --- | --- | --- |
| `puerta-secretos` | Gitleaks sobre los commits nuevos | 1 min |
| `puerta-calidad` | `scripts/verificar_calidad.sh` completo (abajo) | 35 min |
| `puerta-artefactos-productivos` | Construye las imágenes pública e interna y comprueba su contenido y su fallo cerrado | 2 a 5 min |
| `puerta-postgresql-contexto-actor-v3` | Identidad y autorización V3 en un PostgreSQL 18 real | 2 min |
| `puerta-postgresql-bolsa-publica` | Proyección pública de Bolsa en PostgreSQL 18 con TLS | 2 min |

`scripts/verificar_calidad.sh` ejecuta: comprobación de `search_path`, `gofmt`,
`go mod verify`, `go test ./...`, `go test -race ./...`, `go vet ./...`,
`vecsilencio`, `go build ./cmd/...`, las pruebas web con Node, las de i18n, las
de dependencias de las superficies pública e interna, los manifiestos web y la
cartografía, `govulncheck`, el tamaño de ficheros y `git diff --check`.

### Qué se prueba en local

- Solo lo focal: `gofmt`, y `go test`, `go test -race` y `go vet` del paquete
  tocado; `node --test` del fichero web tocado; `git diff --check`.
- La puerta completa la pasa la CI. No se repite en local sobre el mismo commit.
- Pruebas proporcionales al riesgo, con un tope orientativo de 30 minutos.
- Por frontera: el dominio sin dobles; los casos de uso con dobles de sus
  puertos; HTTP con estado, cuerpos estrictos, rutas y autorización; en
  escrituras, también denegación, versión vieja, repetición, misma clave con
  otro contenido e historia conservada.
- SQL: ensayo en el clon local de la principal (PostgreSQL 18 en
  `127.0.0.1:55441`) con la skill `ensayar-sql` y revisión independiente
  (`revisar-sql-vec`). Una garantía de permisos, transacción, concurrencia o
  recuperación se prueba en PostgreSQL real. Nunca se ensaya contra cidonia.
- Pantallas: Playwright con el Chrome del sistema a 1440 y 390 px.

### Qué significa «funciona»

Una capacidad es recorrible cuando se comprueba navegador → API → autorización →
aplicación → PostgreSQL → recibo, y, si se afirma que persiste, también tras
reiniciar aplicación y base. Se distingue siempre código escrito, revisado,
probado, instalado, recorrible y publicado. Pruebas verdes no equivalen a
conformidad funcional de RRHH.

Desde el 13/10, una PR se puede fusionar cuando pasan las puertas rápidas y
dirección ha leído el cambio. La puerta de calidad sigue en `main` y, si falla,
se arregla encima.

## 15. Despliegue

### Subidas HZ

Cada subida a cidonia se llama «HZ» seguida de un número correlativo (HZ36 el
13/10). La hace dirección con guiones que están fuera de Git:

1. Compila `origin/main` y marca la revisión en el binario.
2. Hace una **copia fría** de la base y del binario anterior.
3. Instala las migraciones nuevas, si las hay, en orden.
4. Sustituye el binario y la web, y arranca.
5. Comprueba que la web responde (`/portal-empleado/` en 200).
6. Si falla el SQL o el arranque, **vuelve atrás sola** a la versión anterior.

cidonia nunca puede quedar apagada por un fallo de arranque. Si la versión nueva
exige configuración o actos previos, se preparan y se ensayan antes en una copia.
El registro de cada HZ (revisión, copia y vuelta atrás) se guarda fuera de Git.
El procedimiento con ensayo en `ROLLBACK` está en `deploy/principal/desplegar.sh`
y `deploy/principal/README.md`; cada habilitación con pasos manuales tiene su
guía `deploy/principal/habilitacion_*.md`.

### Migraciones

- Viven en `deploy/postgresql/<esquema>/migraciones/NNNNNN_nombre.up.sql`, una
  serie numerada por esquema. Abreviaturas de trabajo: CT
  (`contratacion_temporal`), B (`bolsa_llamamientos`), AD o AD3
  (`autorizacion_atestada_v3`), AUT (`autorizacion`), CA (`contexto_actor_v1`),
  IS (`identidad_sesiones_v1`), P (`personal`), BIC (`bolsa_importacion_convoca`),
  D (`dietas_borradores`). AD218, por ejemplo, es
  `autorizacion_atestada_v3/migraciones/000218_…`.
- **Antes de crear una, se reserva su número** en `RESERVAS_MIGRACIONES.md`, en la
  raíz del directorio de trabajo principal y fuera de Git. Nunca se reutiliza un
  número reservado por otra rama.
- Se instalan en **orden causal**: primero lo que otra necesita. Listas de
  instalación: `deploy/principal/lista_sql_*.txt`.
- **Nunca DOWN con historia.** Una migración instalada no se reaplica, no se edita
  y no se revierte con su `.down.sql` si ya tiene datos o historia encima. Se
  corrige con una migración nueva. Las migraciones son aditivas: un cambio
  incompatible exige versión nueva y mantener a los consumidores anteriores
  mientras se adaptan.
- Instalar una migración no la enciende. Puede faltar un acto de gobierno, un
  LOGIN o un selector. Un fichero SQL en Git no prueba que la base lo tenga.
- Restaurar una copia no es una forma de volver atrás sobre operaciones
  posteriores, y no deshace correos, firmas ni altas en sistemas externos.

### Servidores hasta producción y en producción

Orden de Alberto del 06/10: hasta producción todo corre en `vec-server` en
cidonia (interno, aspirantes y público juntos); lo que importa es que lo que ve
RRHH vaya muy rápido. `vec-admin` corre en cidonia desde el 10/10 solo para los
actos de administración, con su certificado y un lector sin superusuario.
`vec-interno` no se despliega; `vec-publico` existe pero no se publica.

En producción se separan `vec-interno`, `vec-publico` y `vec-admin`, se decide
dónde va el área del aspirante (separada del portal interno, para que los picos
de aspirantes no frenen a RRHH) y se retira `vec-server`. La base de cidonia se
retira entera (12/10): producción se instala desde cero de forma reproducible
(esquema, roles, catálogos y provisiones en una base vacía), y los dos
administradores inventados se sustituyen por personas reales.

## 16. Integraciones externas

| Sistema | Qué es | Cómo se conecta | Estado |
| --- | --- | --- | --- |
| GrxFirma | Firmador de la Diputación (antes AutofirmaV2), compatible con el protocolo de AutoFirma | La persona firma en su equipo (`afirma://` desde el navegador); la clave nunca sale de allí y el servidor nunca firma por su cuenta. VEC verifica con el validador de GrxFirma, desplegado como servicio aparte y consumido por HTTP (`internal/vec/documentos/adapters/validadorautofirma`). Política `politica:vec:firma:verificacion-autonoma:v1` (`deploy/firma/README.md`) | Circuito de dos firmantes en curso. En la copia local, la candidata #1099 produce PDF 1.4 con /ID; la firma CLI y el verificador GrxFirma aeec0e5 superan el rechazo anterior. Falta publicar el plan nominal con la referencia completa y recorrer el registro desde la ficha; dirección lo comprobará en cidonia. Nunca se marca «firmado» sin verificación positiva |
| Correo | Avisos a aspirantes y comunicaciones del llamamiento | Adaptadores SMTP (`internal/modules/contrataciontemporal/adapters/smtp`, avisos de Bolsa, «Mis correos» de Usuarios), con reserva previa para no duplicar envíos. La dirección sale del alta propia de VEC | Sale por un buzón de pruebas. Falta el servidor de la Diputación (duda 88) y su configuración desde Administración, con secreto de solo escritura, cifrado y auditado |
| GINPIX | Aplicación de gestión de personal que usa hoy la Diputación, donde se da de alta a la persona contratada | `ginpixfichero` genera la ficha de carga y `ginpixapi` prepara un envío HTTP. RRHH carga la ficha a mano y registra en VEC el número de alta (`application/confirmacion_ginpix.go`) | Ficha y confirmación manual funcionan. Envío automático no conectado: faltan datos de Informática (duda 127) |
| CONVOCA | Aplicación actual de convocatorias, que VEC sustituye | Importación del Excel de CONVOCA desde la pantalla de Bolsa (vista previa y confirmación) o con `vec-server importar-convoca`; formato `.xls` con `nkiri/xls` | Funciona en cidonia con datos sintéticos. El uso de datos reales espera al DPD (duda 45) |
| OSRM | Motor de rutas por carretera (Open Source Routing Machine) para los kilómetros de Dietas | Contenedor propio con el mapa de Granada y 15 km alrededor, sin servicio público. Cinco variables `VEC_OSRM_*` que van juntas; si falta una, el cálculo no se habilita (`deploy/osrm-granada/README.md`) | Preparado. Dietas está aparcado hasta cerrar Bolsa y Contratación temporal |
| Almacén S3 | Custodia de documentos | `deploy/almacen_s3_ceph/` | En uso con `VEC_DOCUMENTOS_ENABLED` |

Conectores previstos que todavía no tienen puerto: fuente presupuestaria y
validación de la retención de crédito (RC), cálculo del coste de personal, oferta
al SAE, registro electrónico y notificación, calendario de días hábiles y sello
de tiempo. GINPIX sigue siendo un puerto: la ficha de carga manual y el envío
automático usan el mismo modelo y una versión explícita del mapeo de campos.

Sobre GrxFirma (decisiones de Alberto de septiembre y octubre): VEC no importa código de
GrxFirma; la verificación entra por el puerto `VerificadorFirma` de la base de
documentos, de modo que el proveedor se puede cambiar (también vale la AutoFirma
oficial). En los textos se usa el nombre GrxFirma; el protocolo `afirma://` se
mantiene. Se usa la versión de GrxFirma más avanzada que arregle los fallos; si no
la hay, se arregla lo mínimo, con copia de seguridad. La maqueta de firma
aprobada el 09/10 vive aparte del código principal y se puede borrar entera.

Una integración caída nunca se interpreta como éxito, firma, entrega ni
autorización. La tarea queda «esperando a otro departamento o sistema», con un
reintento que se puede recuperar.

## 17. Lo que falta para producción

Producción con datos reales sigue en **NO-GO**. Estos son los criterios técnicos
que deben cumplirse antes:

| Área | Criterio | Estado |
| --- | --- | --- |
| Identidad | Active Directory para personal; certificado, DNIe o Cl@ve para aspirantes; retirada del montaje de pruebas | Pendiente de Informática |
| Composición | Arranque productivo sin la doble llave de desarrollo; ninguna función que tumbe el arranque | Pendiente |
| Autorización | Matriz completa por caso de uso y canal; interruptor de módulos en Administración | Parcial |
| Datos | EIPD, registro de actividades, información por capas, circuito de derechos y conservación | Pendiente (DPD y RRHH) |
| ENS | Categorización, análisis de riesgos, política, medidas y evidencias de conformidad | Pendiente |
| Expediente | CSV, sello de tiempo, registro electrónico, notificación electrónica, índice firmado y exportación ENI | Pendiente |
| Correo | Servidor corporativo configurado desde Administración | Pendiente (duda 88) |
| Auditoría | Cadena íntegra, consulta controlada, exportación con valor de prueba | Parcial (`docs/plan_modulos/auditoria.md`) |
| Exposición por módulo | Decidir qué módulos se abren fuera de la red corporativa (Cronos) y cuáles solo dentro (Dietas, Personal) | Pendiente de estudio |
| Plataforma | Decisiones de julio de 2026 para datos reales (tres máquinas virtuales con Podman Quadlet o Compose, Kubernetes aplazado, preparación para ENS ALTA) y lo que falta (volumetría, RTO y RPO, inventario de la Diputación) | En `docs/estudio_requisitos/README.md`; las valida Sistemas |
| Accesibilidad | Auditoría WCAG/EN 301 549 y declaración de accesibilidad | Pendiente |
| Operación | Copias y restauración ensayadas desde Administración, respuesta a incidentes, auditoría de seguridad completa | Parcial (`docs/plan_modulos/copias_seguridad.md`) |
| Rendimiento | Todas las lecturas por debajo de 300 ms p95 con volumen real | Parcial |

El orden de trabajo está en `docs/producto/PLAN_IMPLEMENTACION.md`.

## 18. Contradicciones detectadas en documentos anteriores

Los documentos absorbidos tienen afirmaciones que ya no valen. Manda este TRD:

| Documento | Afirmación superada | Lo vigente |
| --- | --- | --- |
| `docs/portal_vec/arquitectura_tecnica.md` | No hay migraciones ni base de datos; repositorios en memoria y autenticación ficticia; catálogos en `web/static/locales`; ficheros de menos de 300 líneas; ningún adaptador lee `os.Getenv` | PostgreSQL 18 con cientos de migraciones; catálogos en `web/static/textos/<idioma>/`; aviso a partir de 800 líneas; varias piezas de `internal/app/bootstrap` y `internal/app/composicion` leen el entorno directamente (ninguna capa de dominio lo hace) |
| `docs/portal_vec/cumplimiento_y_seguridad.md` | Prototipo con autenticador ficticio y memoria; doble factor para todo el personal interno | Identidad por certificado y `ContextoActor`; certificado como único factor solo en desarrollo, hasta Active Directory |
| `docs/portal_vec/contrato_modulos_vec.md` (propuestas históricas) | Identidad solo Kerberos; Webpack Module Federation y Web Components con Lit; GrxFirma también para iniciar sesión (`/auth/challenge`); bus de eventos y búsqueda global del shell | JavaScript sin frameworks ni compilación, con imports locales; identidad por certificado en el canal TLS; GrxFirma solo para firmar y verificar; no hay bus de eventos general ni búsqueda global |
| `docs/manual_programador/LEEME.md` | Lista de módulos (Personal/Nóminas, Cronos, Dietas, Bolsa, Administración) | Quince módulos en `internal/modules/` |

## Cómo mantener este documento

- **Quién.** Quien cambie la pila, el arranque, la configuración, la identidad, la
  autorización, la seguridad, la protección de datos, las pruebas, el despliegue
  o una integración lo actualiza en la misma PR. Dirección (Claude) lo revisa al
  fusionar.
- **Cuándo.** En la misma PR que el cambio; cuando una decisión de Alberto cambie
  una regla técnica; y cuando cambie el estado de una integración (apartado 16)
  o de un criterio de producción (apartado 17).
- **Cómo.** Cambios pequeños, en castellano llano (skill `humanizer`,
  `.agents/skills/humanizer/VEC-USO.md`), con la ruta de la fuente y sin
  secretos, rutas privadas ni datos personales. Actualizar la línea
  «Comprobado sobre» al revisar el documento entero.
- **Qué no va aquí.** El estado de cada tarea (`docs/producto/PLAN_IMPLEMENTACION.md`),
  las tablas y rutas concretas (`docs/producto/ESQUEMA_BACKEND.md`), las reglas de
  pantalla (`docs/producto/DISENO_UI_UX.md`), el proceso de trabajo entre agentes
  (`AGENTS.md`) y el procedimiento de cada habilitación (`deploy/principal/`).
- Las reglas de lectura y mantenimiento de los seis documentos están en
  `.agents/skills/documentos-producto-vec/SKILL.md`.
