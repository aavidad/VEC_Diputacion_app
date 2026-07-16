# VEC Diputacion Granada

Prototipo Go de VEC, Ventanilla Electronica del empleado publico para la
Diputacion de Granada. La aplicacion raiz es un shell modular con identidad,
menu, permisos, auditoria, eventos e i18n comunes. Personal/Nominas, Cronos,
Dietas y Bolsa son modulos independientes con `ModuleID`, permisos y menus
propios; VEC solo los agrega y permite relacionarlos por empleado, expediente,
justificante o auditoria.

## Estado honesto

Probado:

- Servidor HTTP con salud en `/healthz`, carcasa estatica en `/` y consulta
  publica minimizada de convocatorias en
  `/api/publico/bolsa/convocatorias`. Esta consulta se monta en todos los modos
  de autenticacion y usa por defecto una fuente sintetica sin datos personales.
- UI estatica en `http://127.0.0.1:8080/` como carcasa del tablero VEC: modulos,
  expedientes, filtros, cola, detalle y flujo de acciones. Sus datos y acciones
  privadas permanecen cerrados hasta conectar identidad y autorizacion reales.
- Registro de modulos Personal/Nominas (`vec.module.personal`), Cronos
  (`vec.module.cronos`), Dietas (`vec.module.dietas`) y Bolsa
  (`vec.module.bolsa`) via manifiestos. Menu, permisos y acciones demo con
  recibo auditable estan probados en casos de uso y adaptadores de prueba, no
  expuestos por el despliegue predeterminado.
- Modelo de workspace VEC con expediente de empleado, puestos, nomina,
  antiguedad, certificados, Cronos, Dietas y Bolsa. Su endpoint permanece
  cerrado (`503`) hasta resolver en servidor persona, relacion, ambito,
  finalidad y campos exactos; no se entrega una instantanea agregada por un
  permiso grueso.
- Con `fake` habilitado expresamente, API Bolsa heredada con
  `GET /api/portal`, `POST /api/demo`, candidatos, manifiesto operacional,
  documentos, alegaciones, avisos, auditoria y persistencia local opt-in. Esa
  API no se registra en `disabled` ni en `trusted_headers`.
- Casos de dominio, puertos, repositorios en memoria y handlers HTTP.
- Nucleo RBAC+ABAC de lista positiva cerrada: sin comodines positivos,
  asignacion/rol/politicas versionados, CAS y decisiones breves. El primer
  adaptador PostgreSQL con RLS y privilegios minimos pasa integracion real,
  pero no esta conectado a ninguna superficie.
- Primer corte de [contexto canonico de actor](docs/portal_vec/registro_decisiones.md#dec-034--contexto-canonico-de-actor-con-perfil-expreso-y-denegacion-por-defecto):
  cuenta autenticada y perfil expreso resuelven exactamente una persona y sus
  enlaces opacos versionados, sin DNI ni autoridad inferida. Persistencia,
  conexion HTTP y revalidacion transaccional permanecen cerradas.
- Dominio VS9 de [llamamientos por primer elegible](docs/portal_vec/registro_decisiones.md#dec-035--llamamiento-determinista-sobre-lista-constituida-y-primer-elegible):
  conserva el prefijo completo del orden, liga bolsa, necesidad, instantanea,
  politica y recibos por version y huella, y falla cerrado ante cualquier hueco
  o caducidad. Fuente autoritativa, firma, persistencia y API siguen cerradas.
- Concesiones ejecutables y denegaciones probatorias usan puertos separados:
  una denegacion nunca entra en el almacen de capacidades ni pierde su causa
  funcional si falla su traza. El registro durable de denegaciones sigue
  pendiente y cerrado.
- Generacion documental con evidencia opaca y consumo unico junto con
  documento, auditoria y outbox en memoria; una decision no se reutiliza para
  otro efecto. Este corte aun no tiene API HTTP ni persistencia productiva.
- Compose local con proxy inverso como unica entrada en loopback, API sin puerto
  publicado y aislada en una red interna. Ambos contenedores se ejecutan sin
  privilegios, con raiz de solo lectura, capacidades eliminadas y limites de
  recursos; sigue siendo un perfil local sin TLS.
- Idempotencia semantica nominal V2 de baremacion con denegacion cerrada de
  serializacion de valores sensibles. La proyeccion persistible (DDL) esta
  bajo dictamen NO-GO hasta cerrar las brechas contractuales listadas en
  [la migracion V2](docs/portal_vec/migracion_baremacion_idempotencia_v2.md).
- Saga durable de firma de baremaciones: contratos, expediente probatorio,
  fachada reanudable y adaptador en memoria con arrendamiento, cercado y
  AES-256-GCM, probados adversariamente (reintentos ambiguos, reanudacion
  entre replicas, claves cruzadas). Los conectores productivos siguen
  pendientes segun
  [el flujo durable](docs/portal_vec/flujo_firma_baremacion_durable.md).
- Integracion continua en GitHub Actions: cada push y pull request ejecuta
  la puerta canonica completa (`gofmt`, tests, `-race`, `vet`, `build`,
  `govulncheck` y tamano de ficheros conforme a DEC-051).
- Test obligatorio local: `go test ./...`; puerta completa en
  `scripts/verificar_calidad.sh`.

Simulado:

- La autenticacion parte deshabilitada. `fake` y `trusted_headers` solo se
  habilitan de forma expresa para pruebas locales; ninguno es apto para
  produccion ni concede autoridad en la arquitectura nueva. El servidor
  rechaza el modo `fake` si alguna red HTTP permitida no es loopback.
- Persistencia en memoria por defecto. `file` y su alias `local_durable` solo
  afectan ahora a la API Bolsa heredada cuando se habilita `fake`; no convierten
  el despliegue `disabled` en una aplicacion privada durable.
- Integraciones AAPP como puertos/stubs iniciales, sin clientes reales SCSP, SIR,
  Notific@, InSiDe ni AutofirmaV2 cableado en runtime.

Pendiente productivo:

- Autenticacion real por certificado/AutofirmaV2, Kerberos AD/SSO y autorizacion
  operativa.
- Atestacion criptografica del PDP y consumo/revalidacion PostgreSQL dentro de
  la misma transaccion de cada efecto; repositorios de negocio, auditoria
  probatoria durable y backups/restauracion ensayada.
- TLS y proxy productivo, observabilidad centralizada, gestion de secretos,
  limites acordados con Sistemas y hardening del entorno final. El proxy local
  incluido no es una terminacion TLS ni un frontal de produccion.
- Desarrollo completo de cada modulo real: Personal/Nominas con maestro de
  empleados, puestos, situaciones, trienios, servicios prestados y certificados;
  Cronos con cuadrantes y normativa de jornada; Dietas con calculo oficial de
  kilometraje/dietas; Bolsa con baremo, listados, alegaciones, firma,
  notificaciones y persistencia duradera.

## Documentacion

### Manual del programador

- [Manual del programador](docs/manual_programador/LEEME.md): arquitectura por
  capas y referencia de todas las funciones y tipos exportados, para que
  sirven y como se usan. Se regenera con
  `scripts/generar_manual_programador.py`.

### Portal VEC: arquitectura y contratos

- [Arquitectura tecnica modular del portal](docs/portal_vec/arquitectura_tecnica.md)
- [Contrato de modulos VEC](docs/portal_vec/contrato_modulos_vec.md): contrato
  para enchufar Bolsa, nominas, concursos u otros modulos, con los niveles de
  madurez declarados por modulo.
- [Contratos API por modulo](docs/portal_vec/contratos_api_modulos.md):
  endpoints reales con envelope, errores y version, huecos de la Ola 2 e
  inconsistencias verificadas en ejecucion; base para clientes web y de
  escritorio equivalentes (DEC-053).
- [Inventario funcional VEC/VEPA para portal empleado](docs/portal_vec/inventario_vec.md)
- [Registro de decisiones y mejoras](docs/portal_vec/registro_decisiones.md)
- [Dominio del portal VEC y autobaremacion](docs/portal_vec/dominio_y_autobaremacion.md)
- [Versiones gobernadas de convocatoria de Bolsa](docs/portal_vec/convocatorias_gobernadas.md)
- [Estudio profesional de pantallas VEC](docs/portal_vec/estudio_pantallas_profesionales.md):
  flujos completos por modulo/menu, datos visibles, acciones, estados,
  integraciones, validaciones y criterio de terminado.
- [UX portal empleado tipo VEC](docs/portal_vec/ux_portal_empleado.md)
- [Matriz inicial de perfiles, roles y ambitos](docs/portal_vec/matriz_roles_y_ambitos.md)
- [Catalogos configurables y gobernados](docs/portal_vec/catalogos_configurables.md)

### Portal VEC: seguridad y autorizacion

- [Auditoria de diseno, estructura y seguridad 2026-07-16](docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md):
  auditoria tecnica provisional, riesgos y plan de remediacion pendiente de
  validacion por los responsables del proyecto.
- [Cumplimiento, seguridad y expediente electronico](docs/portal_vec/cumplimiento_y_seguridad.md)
- [Atestacion criptografica de decisiones de autorizacion](docs/portal_vec/atestacion_criptografica_decisiones.md)
- [Autenticacion fake local segura](docs/portal_vec/autenticacion_fake_local_segura.md)
- [Seguridad y permisos de PostgreSQL](docs/portal_vec/seguridad_persistencia_postgresql.md)
- [Integracion de autorizacion en la firma de baremaciones V1](docs/portal_vec/integracion_autorizacion_firma_baremacion_v1.md)
- [Auditoria adversaria del flujo durable de firma de baremacion V1](docs/portal_vec/auditoria_adversaria_flujo_firma_baremacion_v1.md)
- [Auditoria adversaria de ejecucion documental V3](docs/portal_vec/auditoria_adversaria_ejecucion_documental_v3.md)

### Portal VEC: gestion documental

- [Capacidad documental transversal del Portal VEC](docs/portal_vec/capacidad_documental.md)
- [Almacen documental seguro, cifrado e intercambiable](docs/portal_vec/almacen_documental_seguro.md)
- [Ejecucion documental atestada V3](docs/portal_vec/ejecucion_documental_atestada_v3.md)
- [Firma multiple, CSV, QR y servicio de cotejo](docs/portal_vec/firma_csv_qr_y_cotejo.md)
- [Metadatos institucionales de procedencia documental](docs/portal_vec/metadatos_institucionales_procedencia.md)
- [Recibos materiales de almacenamiento V2](docs/portal_vec/recibos_materiales_almacenamiento_v2.md)
- [Flujo durable de firma de baremaciones](docs/portal_vec/flujo_firma_baremacion_durable.md)
- [Migracion de baremacion a idempotencia semantica V2](docs/portal_vec/migracion_baremacion_idempotencia_v2.md)

### Portal VEC: modulos e integraciones

- [Modulo Personal/Nominas publico](docs/portal_vec/nominas_personal_publico.md):
  normativa, referencias profesionales, modelo funcional y gates de calidad.
- [Brecha del nucleo heredado de Bolsa](docs/portal_vec/brecha_nucleo_heredado_bolsa.md):
  inventario y analisis de brecha de `internal/candidate` para la retirada
  por porte (DEC-050).
- [Referencias para modulos Cronos y Dietas](docs/portal_vec/referencias_cronos_dietas.md)
- [Dietas: matriz provincial de distancias](docs/portal_vec/dietas_matriz_distancias.md)
- [Pagos, tasas y conciliacion](docs/portal_vec/pagos_tasas_y_conciliacion.md)

### Portal VEC: planes de desarrollo

- [Orden de desarrollo Orquesta](docs/portal_vec/desarrollo_vec_orquesta.md):
  alcance, gates y plan de tareas para evolucionar el shell.
- [Plan de implantacion con Orquesta](docs/portal_vec/plan_implantacion_orquesta.md):
  olas, runs, dependencias y agentes recomendados.

### Estudio de requisitos

- [Estudio de requisitos del portal transversal de RRHH](docs/estudio_requisitos/README.md)
- [Peticion de RRHH: transcripcion accesible y lectura tecnica inicial](docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md)
- [Baremacion configurable, jornada y servicios obtenidos de oficio](docs/estudio_requisitos/baremacion_configurable_jornada_y_datos_de_oficio.md)
- [Acceso interno de tecnicos, RRHH y administracion](docs/estudio_requisitos/acceso_interno_tecnicos_administracion.md)
- [Accesibilidad personalizable, ayuda, audio y asistente](docs/estudio_requisitos/ayuda_documentacion_audio_y_asistente.md)
- [Brechas para un producto profesional, seguro y trazable](docs/estudio_requisitos/brechas_para_producto_profesional.md)
- [Arquitectura de despliegue, contenedores y escalado](docs/estudio_requisitos/despliegue_contenedores_y_escalado.md)
- [Manual 00 para Sistemas: preparacion e instalacion de la plataforma](docs/estudio_requisitos/manual_sistemas_preparacion_plataforma.md)
- [Metodo de desarrollo asistido con Orquesta V06](docs/estudio_requisitos/metodo_desarrollo_orquestado_v06.md)
- [Seguridad y despliegue del modulo Cronos](docs/estudio_requisitos/seguridad_y_despliegue_cronos.md)
- [Sistema de diseno, plantillas y temas visuales](docs/estudio_requisitos/sistema_diseno_y_temas.md)

### Comite de seguridad

- [Entregables para el Comite de Seguridad](docs/comite_seguridad/LEEME.md)
- [Informe para la validacion de la arquitectura y del enfoque de seguridad](docs/comite_seguridad/informe_validacion_arquitectura_seguridad.md)

### Referencias de portales AAPP

- [Referencias de portales de empleo publico y del personal](docs/referencias_portales_aapp/README.md)
- [Comparativa y composicion recomendada de portales publicos de RRHH](docs/referencias_portales_aapp/comparativa_y_composicion_recomendada.md)
- [Decision de trabajo: espacios de acceso y persistencia intercambiable](docs/referencias_portales_aapp/decision_espacios_y_persistencia.md)

### Notas de desarrollo

- [OSRM interno de Granada para Dietas/VEC](deploy/osrm-granada/README.md)
- [Vertical slice juridico-administrativo](docs/vertical_slice_juridico.md)
- [Autoprogramacion Orquesta pendientes 2026-07-16](docs/autoprogramacion_orquesta_pendientes_2026-07-16.md):
  cola T01-T08 derivada de la auditoria y las DEC-050 a DEC-053, con estados
  de reserva por carril.
- [Autoprogramacion Orquesta pendientes 2026-05-23](docs/autoprogramacion_orquesta_pendientes_2026-05-23.md)
- [Duplicaciones railes pendientes 2026-05-24](docs/duplicaciones_railes_pendientes_2026-05-24.md)
- [Rail errors observados 2026-05-23](docs/rail_errors_observados_2026-05-23.md)

## Requisitos

- Go 1.25.12 o superior. Las compilaciones de entrega usan una revision
  mantenida y corregida; a fecha de corte, Go 1.26.5. El minimo exacto evita
  compilar con revisiones 1.25 anteriores que conservan vulnerabilidades
  conocidas aunque se fuerce `GOTOOLCHAIN=local`.
- Docker y Docker Compose para arranque containerizado.
- `curl` para smoke checks.
- Git para la puerta completa `scripts/verificar_calidad.sh`; la primera
  ejecucion tambien necesita resolver los modulos Go y `govulncheck`.

## Configuracion

| Variable | Default | Uso |
| --- | --- | --- |
| `VEC_HTTP_ADDR` | `127.0.0.1:8080` | Direccion de escucha HTTP canonica; parte cerrada en loopback. |
| `BOLSA_HTTP_ADDR` | vacio | Alias legado, usado solo si `VEC_HTTP_ADDR` no existe. |
| `VEC_AUTH_MODE` | `disabled` | `disabled`; `fake` o `trusted_headers` unicamente para pruebas locales heredadas. |
| `VEC_FAKE_CREDENTIALS_FILE` | vacio | Fichero JSON local obligatorio en `fake`; debe ser regular, `0600` o mas restrictivo y guardar solo SHA-256 de tokens opacos. |
| `VEC_HTTP_ALLOWED_CIDRS` | `127.0.0.1/32,::1/128` | Lista positiva de redes remotas que pueden alcanzar el servidor HTTP. Una entrada invalida cierra el acceso. |
| `VEC_BOLSA_STORAGE_MODE` | `memory` | `memory`, `file` o `local_durable` para datos Bolsa. |
| `VEC_BOLSA_DATA_DIR` | `var/bolsa` | Directorio durable del modulo Bolsa. |
| `VEC_BOLSA_DATA_PATH` | `var/bolsa/bolsa_store.json` | Fichero exacto del adaptador durable heredado; prevalece sobre el directorio. |
| `VEC_BOLSA_PUBLIC_SOURCE_PATH` | `data/demo/convocatorias_publicas.demo.json` | Fuente de solo lectura de la consulta publica; el arranque falla si no existe o no es un fichero. |
| `VEC_TRUSTED_PROXY_CIDRS` | `127.0.0.1/32,::1/128` | Proxies admitidos como origen de cabeceras solo cuando se activa el prototipo `trusted_headers`. |
| `VEC_OSRM_BASE_URL` | vacio | URL exacta del OSRM interno; vacio mantiene Dietas sin motor de rutas. |
| `VEC_OSRM_SCOPE_NAME` | vacio | Nombre explicito del ambito geografico autorizado. |
| `VEC_OSRM_SCOPE_BOUNDS` | vacio | Limites canonicos `lat_min,lon_min,lat_max,lon_max`. |
| `VEC_OSRM_ALLOWED_CIDRS` | vacio | Redes de destino positivas del conector OSRM; no se infieren. |

La ruta raiz `GET /` sirve la UI estatica. El shell VEC vive en `/api/vec`; sus
rutas privadas exigen identidad. La API Bolsa heredada bajo `/api` solo existe
en `fake`. La consulta publica de Bolsa usa el prefijo separado
`/api/publico/bolsa` y no lee el almacen privado.

El modo `fake` no tiene usuarios ni tokens incorporados. Ademas del fichero,
exige `VEC_HTTP_ADDR` con IP loopback literal (por ejemplo,
`127.0.0.1:8080`) y CIDR permitida exclusivamente local. La preparacion y
rotacion se describen en
[Autenticacion fake local segura](docs/portal_vec/autenticacion_fake_local_segura.md).

En `fake`, cada token resuelve un unico sujeto, rol VEC y perfil heredado. Un
token ciudadano no sirve para tramitar y uno tecnico no puede actuar como el
candidato. En altas de candidato, el `id` debe coincidir exactamente con el
sujeto autenticado y `call_id` debe ser la convocatoria configurada
`convocatoria-demostracion`: no hay valor predeterminado, comodin ni inferencia.
El README no publica un token generico ni mezcla perfiles en un mismo ejemplo.

### Red del perfil Compose

Compose permite cambiar sus rangos antes de crear las redes:

| Variable Compose | Default | Uso |
| --- | --- | --- |
| `VEC_HTTP_PUBLISHED_PORT` | `8080` | Puerto del proxy publicado solo en `127.0.0.1`. |
| `VEC_DOCKER_SUBNET` | `192.168.255.240/29` | Red interna de API y proxy. |
| `VEC_DOCKER_GATEWAY` | `192.168.255.241` | Pasarela de la red interna. |
| `VEC_PROXY_INTERNAL_ADDRESS` | `192.168.255.242` | IP fija del proxy y unico CIDR admitido por la API. |
| `VEC_API_INTERNAL_ADDRESS` | `192.168.255.243` | IP fija de la API, sin publicacion al host. |
| `VEC_DOCKER_EDGE_SUBNET` | `192.168.255.248/29` | Red de borde del proxy. |
| `VEC_DOCKER_EDGE_GATEWAY` | `192.168.255.249` | Pasarela de la red de borde. |

Los dos rangos deben ser distintos entre si y no solaparse con redes Docker,
VPN o corporativas existentes. Si Sistemas asigna otros rangos, debe cambiar de
forma coordinada subred, pasarela e IP fijas; no se debe ampliar
`VEC_HTTP_ALLOWED_CIDRS` a toda la red interna.

## Arranque local con Go

```bash
VEC_HTTP_ADDR=127.0.0.1:8080 \
  VEC_AUTH_MODE=disabled \
  VEC_HTTP_ALLOWED_CIDRS=127.0.0.1/32,::1/128 \
  VEC_BOLSA_PUBLIC_SOURCE_PATH=data/demo/convocatorias_publicas.demo.json \
  go run ./cmd/vec-server
```

La puerta completa y reproducible para desarrollo y CI se ejecuta con:

```bash
scripts/verificar_calidad.sh
```

`cmd/bolsa-server` esta retirado y falla cerrado. No existe una segunda via de
arranque: cualquier ejecucion soportada usa `cmd/vec-server` y su configuracion
canonica, incluido TLS cuando corresponda.

## Arranque local con Docker Compose

```bash
docker compose --profile local up --build -d
```

El proxy publica `http://127.0.0.1:8080`; `vec-api` solo declara `expose` en la
red interna y no tiene un puerto del host. El perfil arranca con autenticacion
`disabled`, elimina antes del salto las cabeceras de identidad aportadas por el
cliente y solo expone la consulta publica. El volumen nombrado queda reservado
para el adaptador heredado, pero en este modo no demuestra persistencia privada.

Parar:

```bash
docker compose --profile local down
```

## Smoke checks

Con el arranque Go o Compose anterior, estas comprobaciones son reproducibles
sin secretos ni credenciales locales:

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS \
  'http://127.0.0.1:8080/api/publico/bolsa/convocatorias?plazo=abierto'
test "$(curl -sS -o /dev/null -w '%{http_code}' \
  http://127.0.0.1:8080/api/vec/modules)" = 401
test "$(curl -sS -o /dev/null -w '%{http_code}' \
  http://127.0.0.1:8080/api/portal)" = 404
```

Los dos ultimos checks prueban la frontera cerrada: el shell privado solicita
identidad y la API heredada ni siquiera esta montada. Las pruebas `fake` deben
seguir el manual enlazado y usar credenciales locales separadas por perfil; no
forman parte del smoke predeterminado.

Los endpoints actuales son de prototipo. No sustituyen registro electronico,
firma, notificacion fehaciente, archivo ENI ni persistencia duradera.

## Arquitectura

- `internal/vec/domain`: tipos del shell VEC sin HTTP ni persistencia concreta.
- `internal/vec/ports`: contratos de registro de modulos, auditoria, eventos e
  interoperabilidad AAPP.
- `internal/vec/application`: casos de uso del shell probados contra memoria.
- `internal/vec/adapters`: HTTP y memoria.
- `internal/vec/adapters/postgres`: primera barrera durable de autorizacion,
  aislada y no cableada hasta completar atestacion y consumo atomico.
- `internal/modules/personal`: manifiesto de Personal/Nominas: expediente de
  empleado, puestos, situaciones administrativas, antiguedad, servicios
  prestados, certificados, nomina e incidencias retributivas.
- `internal/modules/cronos`: manifiesto de Cronos: fichajes, horarios,
  incidencias, permisos, vacaciones, reducciones 63/64, saldos y aprobaciones.
- `internal/modules/dietas`: manifiesto de Dietas: comisiones de servicio,
  kilometraje, mapa provincial, justificantes, aprobaciones y liquidaciones.
- `internal/modules/bolsa`: manifiesto del modulo Bolsa para registrarlo en VEC.
- `internal/candidate`: nucleo heredado de Bolsa, usado por el primer modulo.
- `cmd/vec-server` y `config`: composicion y configuracion canonica.
- `cmd/bolsa-server`: centinela retirado; no arranca ningun servidor.

Los directorios `Baremador`, `Bolsa_Diputacion`, `Bolsa_Diputacion_app`,
`convoca_dipgra` y otros materiales locales son fuentes de estudio o
prototipos independientes: no forman parte del binario, la imagen, las pruebas
del modulo raiz ni una superficie desplegable. No deben ejecutarse ni
publicarse: algunos carecen de la identidad, autorizacion y auditoria del
nucleo y pueden contener datos personales de referencia. `.dockerignore` y el
`Dockerfile` canonico los excluyen de todas las capas de imagen.

La i18n se centraliza en `internal/shared/i18n`; el prototipo usa catalogo
espanol con fallback si no hay ficheros externos.
