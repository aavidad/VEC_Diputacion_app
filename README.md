# VEC Diputacion Granada

Copyright (c) 2026 Alberto Avidad (avidad@dipgra.es), para la Diputacion
Provincial de Granada. Publicado bajo la
[Licencia Publica de la Union Europea v1.2 (EUPL-1.2)](LICENSE).

Prototipo Go de VEC, Ventanilla Electronica del empleado publico para la
Diputacion de Granada. La aplicacion raiz es un shell modular con identidad,
menu, permisos, auditoria, eventos e i18n comunes. Personal/Nominas, Cronos,
Dietas y Bolsa son modulos independientes con `ModuleID`, permisos y menus
propios; VEC solo los agrega y permite relacionarlos por empleado, expediente,
justificante o auditoria.

Fecha de corte de este estado: **23 de julio de 2026**. El repositorio es una
base de desarrollo y demostracion verificable; no acredita por si solo
conformidad ENS, ENI o RGPD ni esta autorizado para tratar datos reales.

## Seguimiento sencillo

El estado comprensible, el frente actual y el orden de trabajo se mantienen en
[ESTADO_PROYECTO.md](ESTADO_PROYECTO.md). Esta es la tabla de dirección que se
actualiza cada vez que se cierra una capacidad o cambia el frente principal;
los códigos internos `Txx` quedan relegados a la documentación técnica.

Los agentes paralelos, incluidos los que se arranquen manualmente, deben leer
[ORQUESTACION_AGENTES.md](ORQUESTACION_AGENTES.md) antes de tomar una tarea.

### Frente activo: contratación temporal solicitada por RRHH

RRHH ha definido un procedimiento integral que parte de la petición de un
centro y termina con la incorporación, el envío a GINPIX y el seguimiento. No
reemplaza el módulo Bolsa: se implementa como el módulo coordinador
`contrataciontemporal`, que usa Bolsa para la vía de cobertura y los
llamamientos.

- [Especificación del expediente de contratación temporal](docs/portal_vec/expediente_contratacion_temporal_rrhh.md)
- [Objetivos y hoja de ruta del frente RRHH](docs/portal_vec/objetivos_y_hoja_ruta_rrhh_2026-07-23.md)
- [Tareas verificables y commits del frente RRHH](docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md)
- [Mapa visual de objetivos, dependencias y paralelización](docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md)
- [Matriz normativa europea, española y andaluza](docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md)
- [Relevo técnico del frente activo](docs/portal_vec/relevo_contratacion_temporal_2026-07-23.md)

Estado del primer corte: especificación, manifiesto, dominio, puertos de alta y
primer caso de uso implementados sobre la convergencia C3/C4 `eeac3a2`. El
nuevo registro de solicitud exige identidad interna de garantía alta, flujo
gobernado, HMAC, idempotencia, autorización de efecto y una confirmación
transaccional de expediente, auditoría y outbox. El adaptador PostgreSQL, la
API y la web siguen pendientes; ninguna pantalla o contrato aislado se
contabiliza como flujo administrativo terminado.

## Estado honesto

Probado:

- Proceso público independiente `cmd/vec-publico`, con vida en `/livez` y
  disponibilidad en `/readyz` (`/healthz` conserva el alias de readiness),
  portal anónimo en `/bolsa/` y consulta minimizada de convocatorias en
  `/api/publico/bolsa/convocatorias`, junto con el directorio gobernado de
  categorias en `/api/publico/bolsa/categorias`. Su raíz de composición no
  carga Personal, credenciales `fake`, almacenes privados ni la API heredada;
  usa por defecto fuentes de demostracion sin datos personales.
- Raíces HTTP de lista positiva separadas para la superficie pública y la
  interna. La primera no sirve el Portal del Empleado ni `/api/vec`; la segunda
  no sirve la Bolsa pública, rechaza cookies, `Proxy-Authorization` y
  cabeceras heredadas de identidad, y nunca emite `Set-Cookie`. Ninguna raíz
  entrega trailers de petición a la aplicación: los materializa con un límite
  previo al caso de uso y rechaza tanto los declarados como los tardíos.
- UI estatica en `http://127.0.0.1:8080/` como carcasa del tablero VEC: modulos,
  expedientes, filtros, cola, detalle y flujo de acciones. Sus datos y acciones
  privadas permanecen cerrados hasta conectar identidad y autorizacion reales.
- Portal anonimo en `http://127.0.0.1:8080/bolsa/` con menu propio y estable de
  convocatorias, busqueda, categorias y ayuda. Conserva tema y logotipo
  institucionales, pero no presenta enlaces a Cronos, Nominas, Dietas,
  Administracion ni Auditoria interna; escritorio y movil tienen pruebas de
  regresion especificas.
- Registro de modulos Personal/Nominas (`vec.module.personal`), Cronos
  (`vec.module.cronos`), Dietas (`vec.module.dietas`) y Bolsa
  (`vec.module.bolsa`) via manifiestos. Menu, permisos y acciones demo con
  recibo auditable estan probados en casos de uso y adaptadores de prueba, no
  expuestos por el despliegue predeterminado.
- Catalogo profesional comun y versionado, con 68 categorias de demostracion,
  compartido por Personal y Bolsa. El arranque coteja que las categorias de
  cada convocatoria existan en la version exacta del catalogo; la UI ya no
  mantiene una lista de dos categorias escrita en el codigo.
- Dominio, servicio y frontera web de borradores de convocatorias gobernadas:
  alta y actualizacion conservan flujo, bases, catalogos, autorizacion, motivo,
  CAS e idempotencia exactos. El Portal del Empleado ya contiene una bandeja y
  un editor que consumen la API interna real, sin caer a datos sinteticos si
  falta identidad. La ruta y el adaptador HTTP estan probados, pero todavia no
  se montan en la composicion productiva ni disponen de un adaptador Go
  PostgreSQL autorizado; por ello la pantalla falla cerrada fuera de pruebas.
- Contrato durable de esos borradores: un compromiso HMAC no consumible fija
  accion, version, actor, motivo y correlacion antes del PDP; la reserva solo
  nace tras la concesion. El diario distingue alias HMAC de distintas
  generaciones y conserva una unica identidad primaria, por lo que un reintento
  solapado no duplica el efecto. Antes de confirmar, el agregado se liga a un
  perfil criptografico gobernado, una AAD canonica, un sobre AEAD, una clave de
  datos envuelta y una atestacion KMS que la persistencia debera revalidar dentro
  de la transaccion. El material durable no conserva el motivo ni su SHA
  semantico crudo. Es un GO del contrato de nucleo, no un flujo RRHH operativo:
  la confirmacion PostgreSQL permanece deliberadamente cerrada hasta almacenar
  y comprobar todo el contrato KMS, y faltan composicion e identidad reales.
- Contrato reservado para un futuro espacio de trabajo interno. Su endpoint
  permanece cerrado (`503`) hasta resolver en servidor persona, relacion,
  ambito, finalidad y campos exactos; el antiguo agregado sintetico fue
  eliminado y no se entrega una instantanea transversal por un permiso
  grueso.
- Adaptador HTTP interno de solo lectura para
  `GET/HEAD /api/vec/bolsa/panel`, con envelope cerrado, fechas opcionales
  reales, listas canónicas, errores no filtrados y rechazo de toda identidad o
  selector declarado en cabeceras, query o cuerpo. Está deliberadamente sin
  montar: no devuelve datos hasta que la composición resuelva identidad,
  perfil, ámbito, motivo y autoridad VEC-AD-2 del lado servidor.
- Con `fake` habilitado expresamente, API Bolsa heredada con
  `GET /api/portal`, `POST /api/demo`, candidatos, manifiesto operacional,
  documentos, alegaciones, avisos, auditoria y persistencia local opt-in. Esa
  API no se registra en `disabled`; `trusted_headers` se rechaza ya durante el
  arranque de cualquier composicion integrada.
- Casos de dominio, puertos, repositorios en memoria y handlers HTTP.
- Nucleo RBAC+ABAC de lista positiva cerrada: sin comodines positivos,
  asignacion/rol/politicas versionados, CAS y decisiones breves. El primer
  adaptador PostgreSQL con RLS y privilegios minimos pasa integracion real,
  pero no esta conectado a ninguna superficie.
- Reglas de baremo configurables y versionadas con gobierno de publicacion y
  activacion: una version pasa por borrador, revision, publicacion y
  activacion sin pisar versiones previas, con huella e historial. El calculo
  de experiencia usa aritmetica racional exacta (sin coma flotante) y
  redondeos configurados por las bases. La persistencia oficial de los
  resultados de experiencia tiene esquema PostgreSQL con RLS, roles de
  privilegio minimo, migraciones y pruebas de integracion mecanica; aun no
  se monta en produccion.
- Perfil de autorizacion V2 nominal: la solicitud efectiva ya no admite
  principal declarado ni motivo libre, liga la referencia exacta de un
  catalogo publicado y separa evidencias y registros V1/V2. El adaptador de
  memoria, los generadores CSPRNG y la capacidad nominal de correlacion estan
  probados de extremo a extremo. La proyeccion historica de motivos PostgreSQL
  V2, su adaptador Go con identidad evaluadora minima y las representaciones
  binarias separadas `VEC-AD-2` para concesiones y `VEC-AD-D-1` para
  denegaciones tambien estan implementados y probados. Sus parsers estrictos
  solo producen proyecciones nominales minimizadas, con limites previos a la
  reserva y canonicalidad byte a byte; nunca reconstruyen autoridad. No se
  montan en produccion. La inspeccion COSE Sign1 comun sigue siendo una
  primitiva sin autoridad, pero el servicio estricto VEC-AD-2, el catalogo
  PostgreSQL de confianza publica y su cargador Go ya gobiernan raices,
  ventanas y revocaciones y pasan integracion conjunta sobre PostgreSQL 18.4.
  No contienen claves privadas ni conceden efectos. El firmante aislado, el
  manifiesto criptografico de gobierno, el anclaje externo anti-restauracion,
  los registradores por consumidor y el consumo atomico del efecto permanecen
  cerrados.
- Primer corte de [contexto canonico de actor](docs/portal_vec/registro_decisiones.md#dec-034--contexto-canonico-de-actor-con-perfil-expreso-y-denegacion-por-defecto):
  cuenta autenticada y perfil expreso resuelven exactamente una persona y sus
  enlaces opacos versionados, sin DNI ni autoridad inferida. Persistencia,
  conexion HTTP y revalidacion transaccional permanecen cerradas.
- Dominio VS9 de [llamamientos por primer elegible](docs/portal_vec/registro_decisiones.md#dec-035--llamamiento-determinista-sobre-lista-constituida-y-primer-elegible):
  conserva el prefijo completo del orden, liga bolsa, necesidad, instantanea,
  politica y recibos por version y huella, y falla cerrado ante cualquier hueco
  o caducidad. El adaptador disponible genera referencias con 256 bits de
  aleatoriedad criptografica y espacios de nombres separados, pero aun no esta
  cableado al flujo productivo. Fuente autoritativa, firma, persistencia
  completa y API siguen cerradas.
- La autobaremacion existente se conserva como capacidad funcional. Su recorrido
  completo sigue perteneciendo al runtime heredado habilitado solo con
  autenticacion `fake`; no se confunde con el motor oficial moderno ni se
  elimina. La migracion pendiente debe mantener paridad funcional y sustituir
  identidad, fuentes y persistencia de demostracion por adaptadores
  autoritativos antes de abrirla en el portal real.
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
- Valores exactos compartidos para los futuros motores de baremacion
  (`Puntos`, racionales, fraccion de jornada, fecha e intervalo civil), sin
  `float64`, zonas horarias ni redondeos implicitos. Son infraestructura de
  dominio; aun no interpretan unas bases concretas.
- Primer corte `NUC-006` del registro de fuentes de autoridad: dominio
  versionado e inmutable, doble reloj juridico/tecnico, solicitudes de firma
  durables, linaje verificable, historia encadenada y formato canonico V1.
  Incluye ya el caso de uso de consulta interna exacta con autorización V2 y
  recibo ligado a lectura, auditoría firmada y resultado confirmado. Permanece
  en **NO-GO productivo** hasta incorporar repositorio transaccional,
  comprobacion criptografica y de competencia, segregacion y anclaje externo
  WORM, segun
  [su especificacion](docs/portal_vec/registro_fuentes_autoridad.md).
- Analisis integral de RRHH, matriz normativa, catalogo funcional y limites de
  materias reservadas documentados antes de ampliar Personal, Cronos o
  Nominas. Son especificaciones de trabajo y no reglas activas en Bolsa.
- Integracion continua en GitHub Actions: cada push y pull request ejecuta
  la puerta canonica completa (`gofmt`, tests, `-race`, `vet`, `build`,
  `govulncheck` y tamano de ficheros conforme a DEC-051).
- Test obligatorio local: `go test ./...`; puerta completa en
  `scripts/verificar_calidad.sh`.

Simulado:

- La autenticacion parte deshabilitada. Solo `fake` puede habilitarse de forma
  expresa para pruebas locales; no es apto para produccion ni concede
  autoridad en la arquitectura nueva. El servidor rechaza el modo `fake` si
  alguna red HTTP permitida no es loopback. El valor heredado
  `trusted_headers` permanece reconocible en configuracion para fallar con un
  diagnostico explicito, pero ninguna composicion integrada lo admite.
- Persistencia en memoria por defecto. `file` y su alias `local_durable` solo
  afectan ahora a la API Bolsa heredada cuando se habilita `fake`; no convierten
  el despliegue `disabled` en una aplicacion privada durable.
- Integraciones AAPP como puertos/stubs iniciales, sin clientes reales SCSP, SIR,
  Notific@, InSiDe ni AutofirmaV2 cableado en runtime.
- El registro `NUC-006` verifica hoy invariantes y huellas estructurales, no una
  firma real ni la competencia del firmante. Su caso de consulta usa un puerto
  transaccional, pero el adaptador durable y los casos de mutacion siguen
  pendientes; no debe exponerse como autoridad por HTTP.

Pendiente productivo:

- Autenticacion real por certificado/AutofirmaV2, Kerberos AD/SSO y autorizacion
  operativa.
- Atestacion criptografica del PDP y consumo/revalidacion PostgreSQL dentro de
  la misma transaccion de cada efecto; repositorios de negocio, auditoria
  probatoria durable y backups/restauracion ensayada.
- TLS y proxy productivo, observabilidad centralizada, gestion de secretos,
  limites acordados con Sistemas y hardening del entorno final. El proxy local
  incluido no es una terminacion TLS ni un frontal de produccion.
- Cierre de los flujos completos de cada modulo real: Personal/Nominas con maestro de
  empleados, puestos, situaciones, trienios, servicios prestados y certificados;
  Cronos con cuadrantes y normativa de jornada; Dietas con calculo oficial de
  kilometraje/dietas; Bolsa con configurador administrativo de bases y baremo,
  listados, alegaciones, firma, notificaciones y persistencia duradera. Existen
  piezas de dominio y contratos de estos flujos, pero no el recorrido
  productivo completo.

## Documentacion

### Manuales

- [Manual de usuario del Portal del Empleado · Bolsas](docs/manual_usuario/manual_portal_bolsas.md):
  recorrido completo por pantallas con capturas; disponible tambien en
  [PDF](docs/manual_usuario/manual_portal_bolsas.pdf).

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
- [Gobierno y publicación de reglas de experiencia](docs/portal_vec/gobierno_publicacion_reglas_experiencia.md):
  secuencia administrativa, autorización, idempotencia, transacción y pruebas
  exigidas para que un baremo pase de borrador a proyección pública.
- [Cálculo oficial de experiencia](docs/portal_vec/calculo_oficial_experiencia.md):
  integración del motor puro con fuente exacta, doble autorización,
  idempotencia semántica, persistencia y auditoría transaccional.
- [Persistencia PostgreSQL del cálculo oficial de experiencia](deploy/postgresql/bolsa_calculo_experiencia/README.md):
  migraciones de resultados oficiales, frontera V2, roles de privilegio
  mínimo y pruebas SQL de integración, ACL, frontera y rectificaciones.
- [Estudio profesional de pantallas VEC](docs/portal_vec/estudio_pantallas_profesionales.md):
  flujos completos por modulo/menu, datos visibles, acciones, estados,
  integraciones, validaciones y criterio de terminado.
- [Entregable de presentación de Bolsa para RRHH](docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md):
  inventario del corte navegable, elementos temporales y sustitución por
  conectores autorizados.
- [Modo de presentación RRHH](docs/portal_vec/modo_presentacion_rrhh.md):
  arranque del artefacto aislado, guardas, red y exclusión física de
  presentación en producción.
- [Inventario y retirada incremental de la presentación](docs/portal_vec/inventario_retirada_presentacion_2026-07-19.md):
  decisión por fichero sobre qué conservar, sustituir o eliminar, condición
  de aceptación, prueba de ausencia, migración progresiva y marcha atrás.
- [Matriz de aceptación de la web de Bolsa](docs/portal_vec/matriz_aceptacion_web_bolsa_2026-07-18.md):
  correspondencia entre pantallas reutilizables, contratos y adaptadores de
  presentación o producción.
- [Captura y revisión de la presentación](docs/portal_vec/revision_web_presentacion.md):
  puerta Docker reproducible de 36 vistas y 25 flujos en tres resoluciones.
- [UX portal empleado tipo VEC](docs/portal_vec/ux_portal_empleado.md)
- [Matriz inicial de perfiles, roles y ambitos](docs/portal_vec/matriz_roles_y_ambitos.md)
- [Catalogos configurables y gobernados](docs/portal_vec/catalogos_configurables.md)
- [Catalogo comun de categorias profesionales](docs/portal_vec/catalogo_categorias_profesionales.md)
- [Registro versionado de fuentes de autoridad](docs/portal_vec/registro_fuentes_autoridad.md)

### Cumplimiento normativo

- [Paquete de cumplimiento para las mesas de validacion](docs/cumplimiento/LEEME.md):
  borradores de categorizacion ENS con declaracion de aplicabilidad, RAT y
  EIPD del modulo de Bolsas, con version imprimible conjunta en PDF.

### Portal VEC: seguridad y autorizacion

- [Auditoria de diseno, estructura y seguridad 2026-07-16](docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md):
  auditoria tecnica provisional, riesgos y plan de remediacion pendiente de
  validacion por los responsables del proyecto.
- [Cumplimiento, seguridad y expediente electronico](docs/portal_vec/cumplimiento_y_seguridad.md)
- [Atestacion criptografica de decisiones de autorizacion](docs/portal_vec/atestacion_criptografica_decisiones.md)
- [Catalogo PostgreSQL de confianza publica VEC-AD-2](deploy/postgresql/confianza_atestacion_v2/README.md)
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
- [Sellado de tiempo y sincronizacion horaria para firmas](docs/portal_vec/sellado_de_tiempo_y_sincronizacion.md):
  sello cualificado externo (TS@/FNMT) como hora legal frente al reloj del
  servidor (NTP/ROA), lo que el dominio ya exige y el adaptador T19 pendiente.
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
- [Brecha funcional verificada de Bolsa a 17 de julio de 2026](docs/portal_vec/brecha_funcional_bolsa_2026-07-17.md):
  separa modelos, demostraciones, integraciones y capacidades productivas, y
  fija el siguiente corte vertical definitivo.
- [Referencias para modulos Cronos y Dietas](docs/portal_vec/referencias_cronos_dietas.md)
- [Dietas: matriz provincial de distancias](docs/portal_vec/dietas_matriz_distancias.md)
- [Cartografía interna de Dietas](docs/portal_vec/cartografia_interna_dietas_2026-07-19.md):
  rutas OSRM y teselas OSM locales, versionadas y confinadas en Docker.
- [Pagos, tasas y conciliacion](docs/portal_vec/pagos_tasas_y_conciliacion.md)

### Portal VEC: planes de desarrollo

- [Orden de desarrollo Orquesta](docs/portal_vec/desarrollo_vec_orquesta.md):
  alcance, gates y plan de tareas para evolucionar el shell.
- [Plan de implantacion con Orquesta](docs/portal_vec/plan_implantacion_orquesta.md):
  olas, runs, dependencias y agentes recomendados.

### Estudio de requisitos

- [Estudio de requisitos del portal transversal de RRHH](docs/estudio_requisitos/README.md)
- [Analisis integral del portal de Recursos Humanos](docs/estudio_requisitos/analisis_integral_rrhh.md)
- [Matriz normativa de Recursos Humanos a julio de 2026](docs/estudio_requisitos/matriz_normativa_rrhh_2026.md)
- [Catalogo funcional y hoja de ruta integral de RRHH](docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md)
- [Peticion de RRHH: transcripcion accesible y lectura tecnica inicial](docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md)
- [Baremacion configurable, jornada y servicios obtenidos de oficio](docs/estudio_requisitos/baremacion_configurable_jornada_y_datos_de_oficio.md)
- [Decisiones del calculador exacto de experiencia V1](docs/portal_vec/decisiones_calculador_experiencia_v1.md)
- [Integracion del Baremador de puestos singularizados y otros procesos](docs/estudio_requisitos/integracion_baremador_concursos_provision.md)
- [Modelo historico de RPT, plazas, puestos, ocupaciones y vacantes](docs/estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md)
- [Calendario habil, laboral y de jornada historico](docs/estudio_requisitos/calendario_habil_laboral_historico.md)
- [Turnos, festivos y compensaciones](docs/estudio_requisitos/turnos_festivos_y_compensaciones.md)
- [Archivo documental relacionado de RRHH](docs/estudio_requisitos/archivo_documental_rrhh_relacionado.md)
- [Materias economicas, reservadas y relaciones laborales](docs/estudio_requisitos/materias_reservadas_economicas_y_relaciones_laborales.md)
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
| `VEC_AUTH_MODE` | `disabled` | `disabled` o `fake` local. `trusted_headers` hace fallar el arranque integrado; `cmd/vec-publico` lo ignora porque no compone autenticacion. |
| `VEC_FAKE_CREDENTIALS_FILE` | vacio | Fichero JSON local obligatorio en `fake`; debe ser regular, `0600` o mas restrictivo y guardar solo SHA-256 de tokens opacos. |
| `VEC_HTTP_ALLOWED_CIDRS` | `127.0.0.1/32,::1/128` | Lista positiva de redes remotas que pueden alcanzar el servidor HTTP. Una entrada invalida cierra el acceso. |
| `VEC_BOLSA_STORAGE_MODE` | `memory` | `memory`, `file` o `local_durable` para datos Bolsa. |
| `VEC_BOLSA_DATA_DIR` | `var/bolsa` | Directorio durable del modulo Bolsa. |
| `VEC_BOLSA_DATA_PATH` | `var/bolsa/bolsa_store.json` | Fichero exacto del adaptador durable heredado; prevalece sobre el directorio. |
| `VEC_BOLSA_PUBLIC_SOURCE_PATH` | `data/demo/convocatorias_publicas.demo.json` | Fuente de solo lectura de la consulta publica; el arranque falla si no existe o no es un fichero. |
| `VEC_TRUSTED_PROXY_CIDRS` | `127.0.0.1/32,::1/128` | Parametro heredado conservado para compatibilidad de configuracion y pruebas aisladas; ninguna raiz integrada lo usa como origen de identidad. |
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

La superficie anónima aislada se arranca con:

```bash
VEC_HTTP_ADDR=127.0.0.1:8080 \
  VEC_AUTH_MODE=disabled \
  VEC_HTTP_ALLOWED_CIDRS=127.0.0.1/32,::1/128 \
  VEC_BOLSA_PUBLIC_SOURCE_PATH=data/demo/convocatorias_publicas.demo.json \
  go run ./cmd/vec-publico
```

Este proceso solo acepta `/livez`, `/readyz`, `/healthz`, `/bolsa/`, los recursos públicos
enumerados y `/api/publico/`; `/`, `/api/vec`, `/api/demo` y el Portal del
Empleado responden `404`. La fuente predeterminada combina metadatos públicos
reales contrastados —títulos, categorías, fechas y CVE del BOP— con un
envoltorio de demostración sin validez administrativa. No importa expedientes
ni datos personales.

### Entregable del Portal del Empleado y Bolsa

La presentación completa se ejecuta en Docker y mantiene separados el portal,
el mediador cartográfico y el destino normal sin material DEMO. Se arranca deliberadamente
con:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

El `Dockerfile` produce tres destinos VEC: `runtime-presentacion` contiene solo
el portal sintético, `runtime-cartografia-presentacion` contiene solo el
mediador de rutas y `runtime` contiene el servidor normal sin material de
presentación. `scripts/verificar_contenido_artefactos_presentacion.sh` construye
e inspecciona los tres árboles.

El proxy es el único componente que publica un acceso, en
`http://127.0.0.1:8081`; el lanzador queda en `/presentacion/`. El manifiesto
actual contiene 36 vistas: un lanzador, una consulta pública, 14 vistas del área
personal del aspirante y 20 vistas internas —portal, 17 secciones de Bolsa,
Cronos y Dietas—. El lateral de Bolsa conserva las diez áreas visuales 1–10
validadas en las referencias de RRHH y agrupa bajo ellas las capacidades nuevas
sin perder rutas. Las acciones de firma, registro, pago, carga documental, comunicación y
demás operaciones se representan mediante adaptadores volátiles: exigen
confirmación, devuelven recibos `DEMO-` y se pierden al recargar.

Para una presentación a través de un frontal corporativo se conserva el acceso
privado anterior y se activa una entrada Docker adicional con lista blanca de
origen, bind a una única IP interna y perfil `presentacion-remota`. No se abre
`0.0.0.0` ni se aceptan cabeceras de identidad. Véase
[acceso desde un proxy corporativo](docs/portal_vec/acceso_proxy_presentacion.md).

La consulta pública incluye dos recorridos con plazo abierto, derivados de
convocatorias y bases públicas reales pero con intervalos sintéticos señalados
de forma inequívoca como `DEMO`. Los recibos descargables son PDF A4 reales de
presentación, con identidad institucional, texto de constancia o certificación,
referencia opaca y QR hacia la pantalla local de cotejo. No constituyen firma,
registro ni certificado administrativo.

La muestra no utiliza cookies, `localStorage`, `sessionStorage` ni volúmenes
duraderos. Su único conector funcional real es el cartográfico, confinado a
OSRM en una red Docker exclusiva; no consulta Internet. No contiene datos
personales reales: las
referencias BOP son públicas y reales, mientras personas, expedientes, actos y
resultados privados son sintéticos y se identifican de forma visible. El destino
normal excluye físicamente el lanzador, los datos y los adaptadores de
presentación, aunque todavía no está autorizado ni compuesto para producción.

Dietas calcula rutas reales con el grafo OSRM interno y muestra teselas OSM
renderizadas localmente. El PBF, el grafo y el MBTiles se versionan y se montan
en solo lectura. El navegador usa exclusivamente las rutas del mismo origen
`POST /api/presentacion/cartografia/rutas` y
`/tiles/osm/{z}/{x}/{y}.png`; no conoce las direcciones internas. Portal,
mediador, OSRM y renderizador no publican puertos propios.

Las pantallas, los componentes, los contratos y los renderizadores se han
diseñado como candidatos reutilizables en producción. RRHH deberá validarlos y
la composición productiva aún no existe. La raíz de composición selecciona un
adaptador volátil para la muestra y permitirá incorporar conectores
autorizados; la ruta normal nunca cae automáticamente al adaptador de
presentación. Por tanto, disponer de las 36 vistas navegables acredita el
recorrido visual y funcional de la demo, pero **no** acredita integración con
PostgreSQL, identidad, firma, registro u otros servicios, ni una prueba E2E
productiva ni validez administrativa.

La revisión automática completa se ejecuta, con la composición anterior activa,
en un contenedor efímero que ya contiene Playwright y Chromium:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

La revisión integral actual obtuvo **183/183 escenarios correctos, 183
capturas y cero hallazgos**: 36 vistas y 25 flujos en tres resoluciones,
incluidos los cuatro puntos de vista DEMO, su separación por mínimo privilegio
y la ruta real de Dietas. La herramienta exige la marca técnica del servidor de presentación,
rechaza cualquier destino no autorizado y revisa menús,
recibos DEMO, accesibilidad básica, almacenamiento del navegador, errores y
desbordamientos. En 1024 y 1440 px también comprueba por fila que Estado,
Acciones, sus controles y sus etiquetas no queden recortados ni solapados. Una
revisión humana adicional confirmó las pantallas corregidas y el QR del PDF se
decodificó con un lector independiente. Esta puerta no sustituye la aceptación
humana de RRHH.

El equipo anfitrión solo requiere Docker Engine y Docker Compose v2 para este
recorrido. No se instala ni se ejecuta directamente Playwright, Chromium, OSRM,
TileServer GL, Go o Nginx. La topología, la gobernanza de los datos
cartográficos y la operación se detallan en
[Cartografía interna de Dietas](docs/portal_vec/cartografia_interna_dietas_2026-07-19.md).

La aplicación normal mantiene las fronteras separadas: `/bolsa/` para consulta
anónima, `/area-personal/` para la persona aspirante y `/portal-empleado/` para
gestión interna. Sin identidad, autorización y conectores reales, las dos
superficies privadas fallan cerradas; no sustituyen el error por datos locales.
El detalle operativo está en el
[modo de presentación RRHH](docs/portal_vec/modo_presentacion_rrhh.md), la
[matriz de aceptación](docs/portal_vec/matriz_aceptacion_web_bolsa_2026-07-18.md)
y el
[inventario del entregable](docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md).

La puerta completa y reproducible para desarrollo y CI se ejecuta con:

```bash
scripts/verificar_calidad.sh
```

`cmd/vec-presentacion` sirve el portal no autoritativo y
`cmd/vec-cartografia-presentacion` media de forma aislada el único cálculo de
rutas admitido por la muestra; ninguno es un servidor productivo.
`cmd/vec-publico` conserva la
superficie pública aislada. `cmd/vec-server` se mantiene como composición
integrada heredada para desarrollo; no representa la separación de superficies
de producción y rechaza `VEC_AUTH_MODE=trusted_headers` antes de construir sus
handlers. `cmd/bolsa-server` está retirado y falla cerrado. El proceso interno
productivo no se habilitará hasta componer identidad reforzada, autorización y
persistencia reales.

La frontera de identidad ya exige que el registro autoritativo devuelva, en
la misma operación atómica que consume la aserción, las referencias opacas de
autenticación, aserción, sesión, control y cuentas (`aut_`, `ase_`, `ses_`,
`cse_` y `cta_`). Se revalidan completas antes de proyectar el principal y no
pueden proceder de identificadores IdP o cabeceras. El contrato está cerrado;
el adaptador PostgreSQL durable sigue siendo obligatorio antes del GO.

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
curl -fsS http://127.0.0.1:8080/livez
curl -fsS \
  'http://127.0.0.1:8080/api/publico/bolsa/convocatorias?plazo=abierto'
curl -fsS http://127.0.0.1:8080/api/publico/bolsa/categorias
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
- `internal/vec/adapters/postgres`: barreras durables de autorizacion y
  catalogo publico VEC-AD-2 con cargador de confianza real; permanecen aislados
  y sin cablear a efectos hasta completar firma, anclaje y consumo atomico.
- `internal/modules/personal`: manifiesto de Personal/Nominas: expediente de
  empleado, puestos, situaciones administrativas, antiguedad, servicios
  prestados, certificados, nomina e incidencias retributivas.
- `internal/modules/cronos`: manifiesto de Cronos: fichajes, horarios,
  incidencias, permisos, vacaciones, reducciones 63/64, saldos y aprobaciones.
- `internal/modules/dietas`: manifiesto de Dietas: comisiones de servicio,
  kilometraje, mapa provincial, justificantes, aprobaciones y liquidaciones.
- `internal/modules/bolsa`: modulo hexagonal de Bolsa: manifiesto, consulta
  publica, convocatorias gobernadas, baremacion, firma y adaptadores. Varias
  capacidades siguen siendo contratos o dobles de prueba y no superficies
  productivas. Cada convocatoria gobernada incorpora una organización
  obligatoria y una unidad gestora opcional, inmutables en toda su cadena de
  versiones; las mutaciones derivan su ámbito de la versión confirmada.
- `internal/candidate`: nucleo heredado de Bolsa, usado por el primer modulo.
- `cmd/vec-publico`: composición mínima de la superficie anónima de Bolsa.
- `cmd/vec-server`: composición integrada heredada para desarrollo y
  presentación, no frontera productiva.
- `config`: configuración común de los procesos.
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

## Licencia y autoria

Este software es obra de Alberto Avidad (avidad@dipgra.es), desarrollado para
la Diputacion Provincial de Granada, y se publica bajo la
[EUPL-1.2](LICENSE) para que cualquier administracion publica u organizacion
pueda reutilizarlo, adaptarlo y redistribuirlo.

Condiciones esenciales de la reutilizacion (articulo 5 de la EUPL-1.2):

- Mantener intactos los avisos de autoria y de licencia, incluido el nombre
  del autor original, en el codigo y en las obras derivadas.
- Distribuir las obras derivadas bajo la EUPL o una licencia compatible de
  las enumeradas en su apendice.
- Indicar los cambios realizados sobre la obra original.

Todas las versiones linguisticas oficiales de la EUPL publicadas por la
Comision Europea tienen identico valor juridico. El fichero `LICENSE` incluye
primero el texto oficial en español y, a continuacion, el texto oficial en
ingles; ninguno prevalece sobre el otro.
