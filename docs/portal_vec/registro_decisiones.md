# Registro de decisiones y mejoras

Este documento recoge alternativas detectadas durante el desarrollo. Las
decisiones reversibles se aplican en segundo plano; las que alteren alcance,
riesgo juridico, datos reales, coste o despliegue se consensuan antes.

## DEC-001 — DOCX en lugar de DOC

- Estado: aplicada.
- Necesidad conservada: RRHH obtiene un documento Word editable.
- Mejora: generar `.docx` Open XML, no `.doc` binario historico.
- Motivo: interoperabilidad, inspeccion, ausencia de macros por construccion y
  mantenimiento con herramientas actuales.
- Reversibilidad: un adaptador futuro podria importar/exportar otro formato sin
  tocar el nucleo, aunque `.doc` no se recomienda.

## DEC-002 — PDF funcional separado del PDF administrativo final

- Estado: aplicada como frontera arquitectonica.
- Salida actual: PDF determinista, Unicode y sin acciones activas.
- Mejora pendiente de consenso tecnico con Sistemas/Archivo: seleccionar y
  certificar el perfil PDF/A y de accesibilidad PDF/UA que corresponda, incluida
  su herramienta de validacion.
- Motivo: «ser PDF» no acredita preservacion, accesibilidad, firma ni registro.

## DEC-003 — Contenido fuera de la base de datos y referencia opaca

- Estado: contrato aplicado; adaptador productivo pendiente.
- Mejora: contenido en almacenamiento de objetos S3 compatible local, con
  subida directa prefirmada para aportaciones y metadatos en PostgreSQL.
- Motivo: absorber picos sin hacer pasar cada byte por el proceso web, permitir
  cuarentena/antivirus y cambiar a nube o red interna sin tocar el dominio.

## DEC-004 — Datos de referencia fuera de Git y Docker

- Estado: aplicada sin borrar material local.
- Hallazgo: `Baremador` contiene nombres y DNI y los productos OSRM ocupan unos
  560 MiB; `COPY . .` los introducia en el contexto de construccion.
- Mejora: exclusiones explicitas y Dockerfile con lista cerrada de directorios.
- Motivo: minimizacion, privacidad, cadena de suministro y tamano de imagen.
- Limite de producto: `Baremador`, `Bolsa_Diputacion`,
  `Bolsa_Diputacion_app`, `cegid_peoplenet_aapp`, `convoca_dipgra`, los PDF de
  entrada y los datos OSRM son fuentes de estudio, prototipos o datos locales;
  no forman parte del binario, imagen, despliegue ni frontera de seguridad de
  VEC. Algunos contienen otro `go.mod`, por lo que tampoco quedan cubiertos por
  `go test ./...` desde la raiz. No se pueden ejecutar ni publicar como si
  fueran el portal actual.
- Retencion: la exclusion no autoriza a borrar originales, datos de trabajo o
  prototipos. Cualquier deduplicacion o eliminacion requiere inventario,
  clasificacion, copia verificada y aprobacion del responsable del dato; hasta
  entonces permanecen aislados y fuera de los artefactos entregables.

## DEC-005 — Construccion interior-exterior con cortes verticales

- Estado: aplicada.
- Regla: dominio -> casos de uso -> puertos -> adaptadores -> API/CLI/MCP -> UI.
- Mejora metodologica: cada capacidad se valida pronto mediante un corte
  vertical pequeno; no se espera a terminar un «nucleo total» sin integracion.

## DEC-006 — Suelo de capacidad para agentes de programacion

- Estado: aplicada por indicacion del responsable del proyecto.
- Trabajo basico o mecanico: `gpt-5.6-terra` con razonamiento `medium` como
  minimo.
- Dominio, concurrencia e integraciones no criticas: se asignan a
  `gpt-5.6-luna` o a un nivel superior.
- Seguridad, proteccion de datos, autorizacion, criptografia, aislamiento de
  red y revision de amenazas: `gpt-5.6-sol` o un nivel superior.
- Arquitectura y revision critica: modelo director o nivel superior disponible.
- Alcance: cualquier tarea que produzca o modifique codigo, incluidas pruebas.
  No se permiten modelos `mini` ni inferiores para programacion.
- Control: el director revisa los cambios, conserva conjuntos de escritura
  separados y exige `go test`, `go vet` y pruebas de carrera proporcionales al
  riesgo antes de integrar.

## DEC-007 — Documento logico separado de sus formatos

- Estado: aplicada en dominio, caso de uso y adaptador de memoria.
- Decision: DOCX, PDF y futuras salidas no son actos administrativos distintos;
  son representaciones de una version de `DocumentoLogico`.
- Motivo: evita estados contradictorios por formato, permite derivar firma y
  preservacion de bytes exactos y conserva una sola historia administrativa.
- Invariante: cada representacion tiene SHA-256 propio y todas comparten la
  huella HMAC de la fuente semantica, plantilla exacta y relaciones tipadas.

## DEC-008 — Idempotencia fuerte antes de renderizar

- Estado: contrato y adaptador concurrente de memoria aplicados; PostgreSQL
  pendiente.
- Decision: reservar por principal y clave opaca antes de crear identificadores
  o renderizar. La clave queda vinculada a una huella HMAC de todos los datos
  con efecto.
- Resultado: un reintento confirmado devuelve el mismo agregado sin nueva
  auditoria; una peticion concurrente informa «en curso»; cambiar datos con la
  misma clave es conflicto; un fallo libera la reserva y permite reintentar.
- Produccion: indice unico, transaccion, arrendamiento renovable, recuperacion
  de procesos caidos y recoleccion controlada de objetos huerfanos.

## DEC-009 — Autorizacion obtenida dentro del caso de uso

- Estado: aplicada a gobierno y generacion documental.
- Decision: se elimina `AutorizacionRef` de las ordenes. El caso de uso llama al
  puerto RBAC+ABAC y valida la decision devuelta antes de efectuar trabajo.
- Motivo: una referencia aportada por el navegador no demuestra que exista,
  este vigente, corresponda al actor ni autorice ese recurso.
- Defensa adicional: `Principal.Roles` y `Principal.Permissions` quedan como
  datos informativos para presentacion/traza y nunca conceden acceso.

## DEC-010 — Claves HMAC separadas por finalidad

- Estado: frontera aplicada; politica KMS productiva pendiente.
- Decision: la huella de idempotencia usa un puerto y una clave distintos de la
  huella de datos/fuente documental. Su entrada canonica incluye directamente
  plantilla, datos, relaciones, representaciones y contexto de la operacion.
- Motivo: rotar la clave de datos no debe convertir un reintento identico en un
  conflicto ni mezclar usos criptograficos.
- Pendiente: fijar con Sistemas el periodo de convivencia/verificacion de las
  claves de idempotencia durante rotaciones y restauraciones.

## DEC-011 — Catalogos de negocio como datos gobernados

- Estado: dominio, caso de uso, puertos y adaptador concurrente de memoria
  aplicados; interfaz web y PostgreSQL pendientes.
- Decision: una opcion nueva crea una version de catalogo desde la aplicacion;
  no se agrega a una constante ni obliga a recompilar el nucleo o el modulo.
- Controles: version inmutable al publicar, revision optimista durante edicion,
  vigencia temporal, fuente, doble control, huella canonica, RBAC+ABAC,
  auditoria y outbox atomicos.
- Limite: las invariantes tecnicas de seguridad y consistencia siguen en codigo.
  Las reglas y transiciones de negocio se incorporaran mediante flujos y motores
  versionados, no ocultandolas en atributos de texto del catalogo.

## DEC-012 — Almacen documental cifrado como conector compuesto

- Estado: puerto de objetos, fachada documental, registro de conectores, carga
  directa segura, recibo atestado de confirmacion y adaptador criptografico
  local aplicados; cifrado por sobre digital, repositorio durable, proceso
  recuperable y adaptador productivo pendientes.
- Decision: PostgreSQL no almacena los binarios. El nucleo utiliza referencias
  opacas y puertos; un decorador de cifrado por sobre digital permite sustituir
  filesystem, S3 compatible o gestor documental sin rebajar la politica.
- Claves: claves de datos por objeto, envueltas por una clave de KMS/HSM; ni
  claves maestras ni claves privadas de firma se guardan en la aplicacion.
- Zonas: carga directa a cuarentena y promocion a almacen admitido solo tras
  verificar tamano, huella, tipo real, antivirus y reglas documentales. Ambas
  zonas usan identidades y permisos distintos.
- Firma: original, representacion firmada y evidencias de validacion son
  objetos inmutables relacionados; una firma nunca sobrescribe los bytes de
  origen.
- Gobierno: autorizacion por objeto, ambito y finalidad; auditoria de toda
  lectura o mutacion; retencion, bloqueo legal, borrado y copia restaurable
  segun cuadro de clasificacion y calendario aprobados.
- Especificacion: `docs/portal_vec/almacen_documental_seguro.md`.

## DEC-013 — QR de cotejo sobre CSV y versiones firmadas inmutables

- Estado: decision y especificacion aplicadas; caso de uso y conectores
  productivos pendientes.
- Decision: el documento muestra CSV y URL legibles y un QR que facilita abrir
  el cotejo. Un codigo lineal, si un legado lo exige, representa el mismo CSV y
  no crea otra identidad.
- Limite: el QR no acredita autenticidad. El cotejo resuelve el CSV opaco a los
  bytes exactos, SHA-256, firmas, sellos, validacion y auditoria de una unica
  version emitida.
- Firma multiple: la franja se incorpora antes de firmar; cada cofirma PAdES es
  una revision incremental inmutable. No se estampa ni reimprime el PDF despues
  de firmado. Una firma posterior a la emision crea nueva version y nuevo CSV.
- Reutilizacion: AutofirmaV2 aporta firma en cliente y apariencia QR; DSS de la
  Comision Europea se evaluara como conector local para validacion PAdES,
  longevidad y deteccion de modificaciones maliciosas.
- AutofirmaV3: debe fallar cerrado cuando la apariencia sea obligatoria,
  devolver huellas y evidencia por revision y permitir que el portal valide y
  almacene cada paso; el cliente de escritorio no se convierte en archivo.
- Acceso: la politica diferencia documentos publicos, protegidos e internos;
  poseer el CSV no publica datos personales ni evita controles adicionales.
- Especificacion: `docs/portal_vec/firma_csv_qr_y_cotejo.md`.

## DEC-014 — Separar autobaremo ciudadano y baremo administrativo

- Estado: contencion aplicada al prototipo heredado; reemplazo por el nuevo
  dominio de Bolsa en curso.
- Hallazgo: la API ciudadana aceptaba el estado enviado por el navegador y
  permitia crear un merito como `Validado`. El calculador sumaba tambien
  meritos rechazados, en subsanacion o borrador sin distinguir la finalidad.
- Decision inmediata: el candidato solo crea borradores y no puede asignar un
  estado administrativo. El autobaremo provisional excluye rechazados y
  subsanaciones; el baremo oficial solo computa meritos validados.
- Motivo: la declaracion del interesado y la decision del organo son hechos
  distintos, con autoridades, efectos y evidencias distintas.
- Limite conocido: el paquete heredado aun usa estados mutables, `float64` y
  reglas compiladas. Se mantiene solo como compatibilidad temporal.
- Verificacion: pruebas negativas de autovalidacion y de filtrado por estado,
  ademas del conjunto completo de pruebas Go y comprobacion sintactica de JS.

## DEC-015 — Decision de baremacion firmada y rectificacion append-only

- Estado: dominio, puertos, caso de uso, adaptadores de prueba y pruebas
  adversarias aplicados; persistencia y conectores productivos pendientes. No
  existen rutas HTTP, CLI ni MCP para este corte.
- Decision: cada merito conserva puntos declarados, calculados y reconocidos,
  resultado y motivacion; cada documento/evidencia tiene su propia conclusion.
  La decision eficaz debe estar firmada y validada.
- Rectificacion: un inspector o supervisor puede revocar una aceptacion o
  rehabilitar un rechazo. No modifica la fila anterior: crea una decision que
  referencia la version y huella exactas sustituidas, registra antes/despues y
  provoca un nuevo calculo versionado.
- Controles: motivo tipificado y razonado, pruebas, norma/base, autorizacion
  reforzada, conflicto optimista, firma, sello de tiempo, auditoria y outbox en
  la misma confirmacion. Los cambios sin efecto se rechazan.
- Documentos: retirar la aptitud administrativa no borra el fichero. Original,
  evaluaciones y decisiones permanecen inmutables con sus retenciones.
- Efectos: si cambia un resultado ya publicado se abre un flujo gobernado de
  rectificacion, notificacion, alegacion/recurso y eventual republicacion; no
  se altera silenciosamente un listado.
- Alternativa descartada: un booleano `aceptado` editable con un log lateral.
  No demuestra que el estado, su auditoria y la firma se confirmasen juntos y
  permite perder decisiones anteriores.
- Firma individual: es un requisito reforzado del proyecto. No se presenta
  como obligacion general de todas las administraciones y no sustituye el acta
  colegiada firmada y verificable.
- Corte aplicado: decision inicial, rectificacion, revocacion y rehabilitacion
  son cuatro acciones exactas. El binario PDF/PAdES se recupera en flujo, se
  comprueban tamano y SHA-256 y se custodia cifrado y seudonimizado antes de una
  reserva OCC breve. La confirmacion liga un manifiesto tipado y sellado con
  agregado, auditoria y outbox. Un conflicto conserva la referencia al
  documento firmado huerfano para reconciliacion; nunca repite ni oculta la
  firma.
- Puerta productiva: repositorio PostgreSQL, archivo durable del manifiesto,
  reconciliador de huerfanos, firma/validacion/almacen/KMS reales y circuito
  maker-checker persistente cuando la politica exija personas distintas. La
  vertical actual exige que adopcion y firma pertenezcan al mismo actor y no se
  presenta como doble control.

## DEC-016 — Composicion de patrones de otras administraciones

- Estado: estudio oficial incorporado a la especificacion.
- Decision: no clonar un portal concreto. Se adopta de VEC/SAS el merito como
  unidad y la comparacion entre puntuacion solicitada y validada; de BAPE/SACYL
  el historico y la consulta de aceptados/rechazados con causa; del INAP el
  desglose, aprobacion colegiada y acta firmada; y de bases del BOP de Granada
  el cotejo local, alegaciones y limites de lo solicitado.
- Mejoras propias: firma por decision, historial append-only, aritmetica decimal
  fija, doble control configurable por riesgo y catalogos/reglas gobernados sin
  recompilar.
- Doble revision: no se ha identificado una obligacion general de dos tecnicos
  por merito. Se configura para rechazos, rectificaciones, excepciones, gran
  impacto, empates/corte, conflicto de interes y muestreo.
- Subsanacion: corrige acreditacion por defecto; no introduce meritos nuevos
  fuera de plazo salvo habilitacion expresa de las bases versionadas.
- Fuentes y contraste: `docs/portal_vec/dominio_y_autobaremacion.md` y
  `docs/referencias_portales_aapp/README.md`.

## DEC-017 — Paralelizacion condicionada al valor de Orquesta

- Estado: revisada el 15 de julio de 2026 por indicacion del responsable del
  proyecto; Orquesta vuelve a estar autorizada, pero no es obligatoria.
- Decision: usar Orquesta cuando reduzca el tiempo total, permita repartir
  unidades realmente independientes o produzca evidencia reproducible que el
  director pueda revisar. Si su coste de preparacion, contexto o integracion es
  mayor que el trabajo acotado, se emplean subagentes directos en tandas, con el
  maximo de concurrencia disponible y mas tandas sucesivas cuando se liberan.
- Separacion de trabajo: cada subagente recibe un directorio o una tarea de
  solo lectura independiente; el director integra, revisa y ejecuta las pruebas
  globales para evitar escrituras cruzadas.
- Motivo: Orquesta es un medio de coordinacion, no una dependencia del producto
  ni una meta de uso. La seleccion se hace por rendimiento y trazabilidad de
  cada bloque, sin sacrificar revision independiente ni puertas globales.

## DEC-018 — Puertas separadas y autenticacion reforzada para personal interno

- Estado: decision aplicada a la especificacion; despliegue e identidad
  corporativa pendientes.
- Decision: portal ciudadano, portal interno corporativo y administracion de
  sistema tendran entradas, audiencias, sesiones y fronteras de red separadas.
  Las rutas internas no existiran en el proxy publico.
- Personal, tecnicos y RRHH: acceso solo desde Mulhacen o VPN autorizada, identidad AD por
  Kerberos/SPNEGO y certificado protegido por PIN/biometria o segundo factor
  equivalente aprobado. Las decisiones probatorias se firman aparte.
- Administradores: cuenta nominativa privilegiada distinta de la ordinaria,
  puesto/bastion de gestion, privilegio temporal, sesiones cortas,
  reautenticacion y doble control en tareas criticas.
- Precaucion: si Kerberos se obtuvo por PKINIT usando la misma tarjeta, contar
  otra comprobacion del mismo certificado como factor independiente puede ser
  incorrecto. Se registran autenticadores reales y nivel de garantia, y se
  aplica la matriz resultante de categorizacion y riesgos.
- Motivo: ENS exige identificadores singulares por perfil, minimo privilegio,
  segregacion, autenticacion proporcionada al riesgo, registro y refuerzos para
  accesos internos/privilegiados. Separar la red reduce ademas la superficie que
  un fallo de autorizacion de aplicacion podria exponer.
- Especificacion: `docs/estudio_requisitos/acceso_interno_tecnicos_administracion.md`.

## DEC-019 — Cobro de tasas mediante conector y confirmacion de servidor

- Estado: arquitectura aprobada; simulacion insegura desactivada, contratos de
  dominio y puertos y primer caso de uso seguro de alta endurecidos y probados.
  Modulo **no exponible**; adaptador corporativo y persistencia productiva
  pendientes de identificar e implementar.
- Hallazgo: la maqueta de Bolsa permitia que un boton del navegador cambiase el
  estado local a `Pagada` y fabricase un recibo. No existia pasarela, evidencia
  remota ni conciliacion. Este comportamiento queda prohibido incluso para una
  futura API real.
- Decision: VEC no captura datos de tarjeta ni acepta importes o estados de
  pago del cliente. Crea una orden exacta e idempotente y usa un puerto tipado
  para redirigir a la pasarela corporativa o recaudatoria aprobada.
- Autoridad: el retorno del navegador nunca confirma el cobro. Solo una
  notificacion autenticada servidor a servidor o una consulta al proveedor,
  con coincidencia exacta y persistencia probatoria, permite avanzar; la
  politica puede exigir ademas conciliacion.
- Consistencia: resultado desconocido se consulta antes de reintentar; pagos,
  devoluciones y anulaciones forman historia de solo adicion. Estado,
  evidencia, auditoria y outbox se confirman atomicamente.
- Separacion: cobros ciudadanos y pagos salientes de nomina, dietas o Tesoreria
  usan puertos y permisos distintos. Un rol de tramitacion no hereda capacidad
  para conciliar o devolver.
- Dinero: entero en unidad menor y codigo de moneda, nunca `float64`.
- Cierre del contrato: permisos, campos, transiciones y procedencias usan
  listas positivas exactas; entradas no canonicas deniegan; referencias son
  opacas; alta, peticion y devolucion tienen HMAC y reservas separadas; el
  origen y handoff quedan ligados al comando; hechos y auditoria conservan la
  autorizacion y las atestaciones verificadas.
- Primer caso: la aplicacion obtiene exactamente una liquidacion autoritativa,
  admite solo pago propio, cruza actor, perfil, vinculo V1, sesion y atestacion,
  exige campos exactos y ninguna obligacion desconocida y reserva una clave
  idempotente antes de confirmar atomicamente orden, auditoria y outbox. No
  recibe importe, tarifa, estado, sujeto o concepto desde el iniciador.
- Cierre TOCTOU: `ConfirmarCreacion` obtiene el instante de la transaccion y
  relee bajo esa misma transaccion la decision V1, la sesion y contexto de actor,
  la asignacion, la version y control de vigencia del rol, el catalogo completo
  y la liquidacion autoritativa. Compara referencias, revisiones, huellas,
  estados y vigencias exactos. Caducidad, retirada, revocacion, ambiguedad o CAS
  perdido revierten orden, auditoria y outbox; la evidencia previa no reemplaza
  estas lecturas.
- Consumo: el mismo `COMMIT` registra la relacion unica
  `DecisionRef -> (OrdenRef, HuellaEfectoSHA256)`. Solo el reintento del efecto
  exacto ya existente se resuelve idempotentemente y sin reescribir; reutilizar
  la decision para otra orden o mutacion deniega sin alterar el primer efecto.
- Alcance demostrado: el puerto expresa estas obligaciones y el adaptador de
  pruebas las simula bajo un mutex, incluidos cambios entre PDP y `COMMIT`. No
  acredita independencia criptografica ni sustituye el repositorio productivo.
  Una autoridad externa solo es admisible si ofrece bloqueo, testigo de
  exclusion (`fence`) o CAS que permanezca valido hasta confirmar la
  transaccion.
- Puerta productiva: no se montaran rutas HTTP, CLI ni MCP hasta disponer de
  identidad/PDP criptograficos reales, pasarela autenticada, saga durable,
  repositorio transaccional con CAS sobre la liquidacion autoritativa,
  conciliacion operativa y auditoria inmutable con claves gestionadas y
  rotables. El verificador de atestacion debera admitir cancelacion explicita.
- Especificacion: `docs/portal_vec/pagos_tasas_y_conciliacion.md`.

## DEC-020 — Denegacion por defecto y alcance expreso por operacion

- Estado: aceptada; nucleo RBAC+ABAC y contrato de identidad/superficies
  aplicados; conexion del arranque productivo pendiente.
- Decision: la politica de permisos evoluciona de menos a mas. Si no se
  autorizan expresamente modulo, accion, tipo y referencia/contexto del
  recurso, conjunto exacto de dimensiones de ambito, finalidad, campos y
  obligaciones, se deniega. Una accion o modulo nuevo no entra en roles
  existentes por comodin.
- Semantica: se usa catalogo positivo cerrado. Ausencia, lista vacia,
  ambiguedad, codigo desconocido, caducidad, error o dependencia no disponible
  deniegan; nunca se busca una concesion parecida ni se aplica una reserva
  permisiva.
- Precedencia: ante varias reglas aplicables gana el resultado mas restrictivo.
  ABAC solo reduce una concesion RBAC previa, las restricciones de campos se
  intersectan, una prohibicion expresa prevalece y los perfiles no suman sus
  permisos. Ampliar autoridad exige publicar y asignar otra concesion exacta.
- Canonicalidad: el punto de autorizacion no recorta, corrige ni traduce una
  accion, perfil, recurso, finalidad o correlacion para hacerlos coincidir. Si
  no llegan ya en su representacion canonica exacta, se deniegan antes de
  consultar al autorizador.
- Consultas: un filtro ausente, vacio, repetido, no canonico o con comodines
  nunca significa «todos». La consulta global, cuando sea necesaria, sera un
  caso de uso diferente con accion, concesion, limites, finalidad y auditoria
  propios; no se obtiene omitiendo el alcance de una consulta ordinaria.
- Instantanea: asignacion, version exacta de rol, control de vigencia global de
  esa version y catalogo completo de politicas actuales se leen como una sola
  instantanea coherente. La decision conserva revision y huella del catalogo,
  referencias y huellas de todas las politicas evaluadas, y separa el
  subconjunto que resulto aplicable.
- Registro: una concesion solo adquiere validez cuando el registro de solo
  adicion completa un CAS atomico que vuelve a comprobar asignacion actual,
  revision/huella del control de rol y revision/huella del catalogo. Si
  publicacion, retirada o revocacion gana la carrera, no se inserta la
  concesion. Si la fuente no permite obtener una instantanea fiable, se deniega
  sin inventar una falsa evidencia.
- Vigencia: el vencimiento de una concesion es el menor entre el TTL tecnico,
  el fin de su asignacion y la siguiente frontera temporal conocida de
  cualquier politica publicada —inicio o fin—. Una decision calculada antes de
  una politica futura no puede atravesar su entrada en vigor.
- Ambitos: no existe ningun comodin positivo, tampoco el heredado
  `global=["*"]`. Las dimensiones de la asignacion y del recurso coinciden
  exactamente y cada valor necesita coincidencia positiva. Una dimension o
  valor adicional, ausente, futuro o desconocido deniega. Una operacion
  transversal usa otra accion y una referencia de conjunto gobernada y
  versionada; ampliar el conjunto exige una nueva publicacion y concesion.
- Efecto: acciones que producen efectos administrativos diferentes se
  autorizan por separado. Baremar inicialmente, rectificar, revocar y
  rehabilitar no comparten permiso; lo mismo rige para preparar, confirmar,
  analizar o promover una carga y para iniciar, conciliar o devolver un cobro.
- Consumo: antes del `COMMIT` del efecto, el repositorio de la operacion vuelve
  a comprobar dentro de esa misma transaccion la decision, su uso, la
  asignacion, el control de rol y el catalogo actuales. Un cambio entre la
  decision y el efecto revierte todo; esta conexion duradera es puerta de
  produccion y no se sustituye con un TTL corto.
- Campos y obligaciones: cada caso de uso declara expresamente si consume una
  lista de campos. Una restriccion de campos ignorada o una obligacion sin
  implementacion comprobable deniega toda la operacion y deja rastro; no se
  ejecuta parcialmente. En el cotejo protegido, `estado` y `codigo_ref` son
  campos base obligatorios; los opcionales se omiten sin concesion exacta y
  `permite_descarga` solo puede ser verdadero con su campo visible, la
  capacidad `descarga` y la politica documental expresamente favorables.
- Exterior: solo visitante, titular/aspirante y representante verificado. La
  zona anonima muestra informacion publicable y el area autenticada limita el
  acceso a recursos propios o representados.
- Interior: RBAC concede una funcion general y los ambitos restringen unidad,
  convocatoria, fase, tarea, expediente, relacion, periodo, clasificacion y
  campos. Se selecciona un unico perfil por sesion.
- Administracion: las cuentas tecnicas y privilegiadas no adquieren acceso
  funcional a expedientes por administrar plataforma, base, claves o roles.
- Gobierno: versiones y asignaciones son configuracion gobernada y revocable;
  cabeceras, menus y roles declarados por el cliente no son autoridad.
- Retirada de rol: se gobierna mediante un control versionado que nombra la
  version exacta, actor, fecha UTC canonica, acto y motivo tipificado. Publicar
  una `v2` retirada nunca revoca implicitamente una asignacion que apunta a
  `v1`.
- Manifiestos: cada entrada de menu declara al menos un permiso concreto,
  previamente incluido en el catalogo del mismo modulo. Se rechazan listas
  vacias, comodines, permisos desconocidos, modulos cruzados, rutas externas y
  duplicados. Un manifiesto invalido recuperado del repositorio cierra todo el
  menu en vez de intentar corregirlo o mostrarlo parcialmente.
- Identidad: una asercion protegida autentica sujeto, cuenta, sesion y factores
  vinculados al mismo sujeto, pero no transporta roles autoritativos. Cada caso
  de uso obtiene la asignacion exacta y vigente del repositorio de acceso.
- Finalidades: se expresan mediante claves canonicas gobernadas, no mediante
  frases libres ni textos normalizados por el servidor. El motivo humano se
  registra aparte y nunca amplia la concesion.
- Compatibilidad: el traductor heredado de roles a permisos solo existe en el
  modo de demostracion. En modos reales, incluso una cabecera procedente de un
  proxy admitido autentica como maximo la identidad; no concede por si sola
  ninguna operacion funcional.
- Contencion heredada: la API antigua de Bolsa, que todavia decide con roles
  gruesos, no acepta identidades por cabeceras en modos reales. Permanece
  cerrada hasta sustituir cada comprobacion por autorizacion RBAC+ABAC de
  accion, recurso, ambito, finalidad y campos. La consulta de sesion y el
  calculo de rutas de Dietas tambien exigen permisos concretos.
- Arranque seguro: sin conector configurado la autenticacion queda
  deshabilitada. Los modos `fake` y `trusted_headers` solo pueden habilitarse
  expresamente para pruebas heredadas; la imagen y Compose parten cerrados. El
  servidor impide arrancar `fake` si el acceso HTTP incluye una red distinta de
  loopback.
- Certificados: presentar `PeerCertificates` no autentica. La terminacion mTLS
  directa exige cadena verificada y coincidencia exacta con el certificado
  par. Un PEM reenviado y una cabecera `SUCCESS` no cuentan, ni siquiera desde
  una CIDR confiable, hasta disponer de asercion criptografica con audiencia y
  antirrepeticion. Rol y garantia tampoco se deducen del sujeto o del nombre
  del mecanismo. Una identidad Kerberos y otra de certificado no se combinan
  hasta que una atestacion confiable demuestre su enlace exacto.
- Cache: las respuestas parten con `Cache-Control: no-store`; solo los recursos
  estaticos versionados sustituyen deliberadamente esa politica. Sesiones y API
  nunca se declaran cacheables.
- Especificacion: `docs/portal_vec/matriz_roles_y_ambitos.md`.

## DEC-021 — Cierre del alta documental heredada hasta disponer de confirmacion fiable

- Estado: contencion aplicada; dominio, puertos y caso de uso seguro
  implementados y probados; adaptadores productivos e integracion HTTP
  pendientes.
- Hallazgo: el endpoint heredado aceptaba del navegador el CSV, SHA-256,
  referencia de almacenamiento, sello de tiempo, firma y fecha, y construia
  con ellos una evidencia. El cliente podia fabricar todos esos datos.
- Decision inmediata: el `POST` heredado de documentos responde con servicio
  no disponible y no persiste nada. La consulta permanece para compatibilidad
  y el caso de uso interno se conserva hasta migrar datos/pruebas.
- Extension de la contencion: la presentacion heredada de alegaciones y las
  transiciones de envio/lectura de notificaciones tambien quedan cerradas,
  porque recibian el CSV probatorio del mismo cliente que solicitaba la
  operacion. La creacion de un borrador de aviso no se presenta como entrega.
- Reemplazo: el servidor reserva una carga directa a cuarentena, recibe del
  almacen la version y huella calculadas, verifica tipo/tamano/antivirus,
  obtiene firma y validacion de conectores confiables y confirma documento,
  auditoria y outbox atomicamente. El navegador solo aporta el fichero y los
  datos declarativos permitidos.
- Autorizacion: preparar, confirmar, analizar y promover son acciones
  independientes de lista positiva. La ausencia o fallo de cualquiera deniega
  la fase correspondiente. El antivirus actua con una identidad tecnica
  explicita y no hereda la identidad ni los permisos del ciudadano.
- Estado tecnico: reserva, preparacion, recepcion, analisis y promocion forman
  una maquina de estados versionada. Idempotencia, control optimista,
  agregado, auditoria y outbox se confirman mediante contratos tipados; una
  respuesta parcial o no concluyente queda retenida.
- Motivo: marcar el antivirus como pendiente no vuelve fiables el objeto, la
  huella, el CSV o la firma declarados. La ausencia temporal de carga es
  preferible a generar evidencia administrativa falsa.

## DEC-022 — Concesiones tecnicas exactas y recibo de confirmacion de un uso

- Estado: contrato del nucleo, caso de uso, adaptador de pruebas y pruebas
  adversarias aplicados. El adaptador criptografico de recibos esta aplicado;
  repositorio durable, intencion recuperable y conector productivo de objetos
  siguen pendientes. La confirmacion es **NO EXPONIBLE**.
- Hallazgo: una sesion de carga que el navegador vuelve a presentar no basta
  para acreditar que la confirmacion pertenece a la preparacion exacta ni
  impide reenvios. Del mismo modo, una finalidad generica no distingue leer,
  analizar, confirmar o promover el objeto.
- Decision: el puerto de almacen usa una lista positiva cerrada de acciones
  tecnicas. Cada solicitud y evidencia incluye exactamente accion, carga,
  sujeto seudonimizado HMAC, recurso, modulo y huella HMAC de solicitud, ademas
  de operacion, correlacion, autorizacion, finalidad y clasificacion. Falta o
  discordancia de cualquier vinculo deniega.
- Privacidad: los conectores no reciben el identificador real del interesado.
  El seudonimo se calcula en un puerto local confiable, por carga y con clave
  propia; no se reutilizan claves de sesion, idempotencia o documentos.
- Reenvio: preparar emite un recibo opaco, no serializable por mecanismos
  genericos, ligado a la sesion y a la solicitud. Confirmar exige que un
  consumidor confiable compruebe MAC, caducidad y consumo atomico de un uso.
  El repositorio fija las fechas durables de alta y consumo. `RegistradoEn` se
  incorpora al recibo mediante una atestacion post-alta y debe coincidir al
  consumir; el reloj del proceso nunca se presenta como evidencia. Una clave
  HMAC de atestacion distinta liga el comprobante al contexto, autorizacion,
  accion, sesion y evidencia
  exactos, incluidas las fechas de alta, consumo y expiracion y el menor
  limite de autorizacion/sesion. El consumo crea atomicamente una intencion
  durable con referencia y huella HMAC. Solo tras verificar esa atestacion
  puede construirse la llamada al conector. Un fallo remoto no autoriza a
  reutilizar el recibo.
- Reemision: el grupo opaco se deriva exclusivamente de carga y sesion. Un
  alta posterior sustituye atomicamente el recibo activo anterior del grupo,
  aunque cambien otros invariantes; estos siguen protegidos por el vinculo
  HMAC y no se relajan. El resultado conserva el predecesor, su mismo grupo
  HMAC, su autorizacion y el instante de sustitucion; un grupo discordante
  deniega. La nueva emision queda ligada a su propia
  autorizacion y nunca hereda la anterior.
- Frontera: emisor, consumidor y verificador deben ser la misma raiz
  criptografica y ninguna dependencia puede ser nula ni nula tipada. Los
  errores del conector se traducen a una respuesta publica cerrada para no
  revelar infraestructura o material probatorio.
- Limite distribuido: consumir el recibo y confirmar en un almacen remoto no
  forman una sola transaccion. HMAC autentica, pero no aporta atomicidad. Se
  exige intencion durable, idempotencia garantizada por el conector y
  reconciliacion antes de habilitar el caso de uso en HTTP, CLI o MCP.
- Evidencia: la confirmacion del almacen debe referenciar el consumo exacto;
  promocion, retencion, inmovilizacion, levantamiento y eliminacion deben
  declarar su fundamento especifico. La creacion acredita un unico instante
  coherente entre objeto y evidencia.
- Alcance productivo: el adaptador en memoria sigue siendo solo de pruebas.
  Aunque el contrato ya modela la intencion, la exposicion HTTP permanece
  cerrada hasta disponer de repositorio PostgreSQL y trabajador recuperable,
  idempotencia remota exacta, consumo durable y atomico, cifrado, cuarentena,
  antivirus, auditoria durable y reconciliacion idempotente.
- Especificacion: `docs/portal_vec/almacen_documental_seguro.md`.

## DEC-023 — Privilegios positivos y seguridad por filas en PostgreSQL

- Estado: primer corte de autorizacion aplicado y probado con PostgreSQL real;
  no montable en produccion hasta atestacion criptografica y consumo atomico
  en la transaccion del efecto.
- Decision: PostgreSQL aplica una segunda barrera de denegacion por defecto.
  Cada modulo, superficie y trabajador usa una identidad tecnica distinta con
  concesiones exactas; las cuentas de ejecucion no poseen objetos, no son
  superusuario y carecen de `BYPASSRLS` y roles globales de lectura o escritura.
- Filas: las tablas sensibles habilitan y fuerzan seguridad por filas. No
  disponer de una politica aplicable deniega. Lectura y escritura tienen
  politicas separadas, con `USING` y `WITH CHECK` expresos.
- Contexto: sujeto, perfil, ambito, finalidad, decision, sesion y correlacion
  proceden de la autorizacion confiable y solo viven dentro de la transaccion;
  nunca se conservan como estado reutilizable de una conexion del pool.
- Atomicidad: agregado, control optimista, hecho, auditoria y bandeja de salida
  se confirman juntos. Los efectos remotos requieren intencion durable,
  idempotencia, consulta y reconciliacion.
- Separacion: portal exterior, gestion interna y trabajadores no comparten
  credencial. Cronos conserva base, red y credenciales exclusivas dentro de
  Mulhacen.
- Limite verificado: el CAS fija frescura pero no duplica el PDP en SQL. La
  identidad `vec_autorizacion_registro` es exclusiva del PDP aislado y nunca
  se entrega a portales o modulos. Falta sellar criptograficamente la tupla
  completa y revalidarla/consumirla junto con agregado, auditoria y outbox;
  hasta entonces el adaptador no se conecta a ninguna superficie.
- Especificacion: `docs/portal_vec/seguridad_persistencia_postgresql.md`.

## DEC-024 — Cadena Go mantenida y dependencias reproducibles

- Estado: aplicada y pendiente de automatizar en la integracion continua.
- Hallazgo: el modulo y el contenedor seguian declarando Go 1.22, rama que ya
  no recibe correcciones de seguridad, y las bibliotecas PDF e imagen estaban
  por detras de sus versiones vigentes.
- Decision: el modulo exige como minimo la revision corregida Go 1.25.12, fija
  la herramienta preferida en Go 1.26.5 y construye con la imagen oficial
  1.26.5 identificada por resumen criptografico. Declarar `go 1.25.12` impide
  que `GOTOOLCHAIN=local` rebaje silenciosamente la compilacion a una revision
  1.25 vulnerable. La imagen de ejecucion tambien queda fijada por resumen.
- Dependencias: `go-pdf/fpdf` se actualiza a 0.12.0 y `x/image` a 0.44.0. Sus
  pruebas de PDF, DOCX, QR/cotejo y seguridad se ejecutan normales y con
  detector de carreras; `govulncheck` no encuentra vulnerabilidades conocidas
  alcanzables al ejecutar el analisis completo con Go 1.26.5 a la fecha de
  corte. El mismo analisis bajo Go 1.25.0 alcanzaba vulnerabilidades de la
  biblioteca estandar y motivo elevar el minimo a 1.25.12.
- Operacion: fijar un resumen no congela actualizaciones. Cada nueva revision
  de seguridad debe proponer resumen, SBOM, pruebas, analisis y aprobacion antes
  de sustituir la imagen reproducible.
- Puerta: `scripts/verificar_calidad.sh` unifica formato, verificacion de
  modulos, pruebas, carreras, analisis estatico, compilacion, vulnerabilidades y
  comprobacion del diff.

## DEC-025 — Fake local sin autoridad aportada por el cliente

- Estado: aplicada en configuracion, composicion, frontera HTTP, cliente
  estatico y pruebas adversarias. No habilita ninguna superficie productiva.
- Hallazgo: la demo precargaba identidades y tokens predecibles, aceptaba
  sujeto, rol y mecanismo enviados por el navegador y podia limitar el par
  remoto a loopback sin ligar realmente el listener a esa interfaz.
- Decision: `fake` exige un fichero local explicito y seguro que almacena solo
  SHA-256 de tokens opacos de alta entropia. Cada registro declara sujeto,
  nombre, un unico rol VEC, rol heredado, mecanismo y garantia exactos; no se
  infiere ninguno. Una tabla positiva solo admite `ciudadano/candidate`,
  `administrativo/validator_l1`, `tecnico_rrhh/validator_l2`,
  `jefatura_rrhh/validator_l2` y `administrador/system_admin`. Cualquier rol
  desconocido, alias, mezcla o pluralidad invalida el fichero completo antes
  de arrancar; un empleado ordinario no se convierte en tramitador de Bolsa.
  JSON, duplicados, principal y permisos Unix se validan tambien antes de
  arrancar.
- Peticion: solo un Bearer registrado selecciona la entrada en servidor.
  `X-Auth-*`, `X-VEC-*` y `X-Auth-Token` no son autoridad. El JavaScript no
  contiene tokens, identidades simuladas ni selector de roles.
- Red: ademas de CIDR exclusivamente local, `VEC_HTTP_ADDR` debe contener una IP
  loopback literal. En fake se rechazan `Forwarded`, `Via` y `X-Forwarded-*`.
  El resolvedor vuelve a exigir una IP loopback y puerto numerico desde
  `RemoteAddr`, de modo que usar por error el manejador crudo no elude la
  frontera del servidor.
- Cierre: `NewDemoAPI` sin fichero seguro falla; ausencia o discordancia de
  credencial devuelve `401`. Un fichero ausente, permisivo, enlazado o
  incorrecto impide el arranque completo.
- Limite: el binario demo aun no esta separado fisicamente del productivo; se
  conserva como mejora pendiente y no se simula su cumplimiento.
- Especificacion:
  `docs/portal_vec/autenticacion_fake_local_segura.md`.

## DEC-026 — Cierre del HTTP heredado de Cronos y del workspace agregado

- Estado: contencion aplicada y probada; superficie interna productiva
  pendiente.
- Hallazgo: los `GET` heredados entregaban una instantanea completa de Cronos
  con un permiso grueso; los `POST` aceptaban el identificador de empleado del
  navegador. El workspace mezclaba datos de varias personas y convertia fallos
  de Cronos en `200` con colecciones vacias. No existe todavia un resolver
  servidor de persona, organigrama, delegacion, finalidad y campos capaz de
  justificar cada recurso.
- Decision: `/api/vec/workspace`, `/api/vec/cronos/timecards` y
  `/api/vec/cronos/leave-requests` fallan cerrados. Una identidad sin el permiso
  preliminar recibe `403`; incluso con el permiso recibe `503` y ningun dato o
  efecto. Los `POST` no decodifican ni persisten identificadores aportados por
  el cliente.
- Errores: una dependencia ausente o un fallo del backend se propaga; nunca se
  sustituye por una respuesta satisfactoria vacia. Los auxiliares no interpretan
  identificador o lista vacios como «todos» ni sustituyen una persona no
  encontrada por la primera disponible. El puerto de jornadas exige fecha y
  lista positiva no vacia de personas, y saldos y permisos exigen persona y
  ejercicio exactos; valores vacios, repetidos, no canonicos o con `*` deniegan.
- Separacion: acceso propio, consulta de subordinados, aprobacion por jefatura,
  competencia de RRHH, justificantes sensibles y GPS son operaciones positivas
  independientes. El rol tecnico o de administrador no concede datos
  funcionales por defecto.
- Reapertura: solo en el binario y la red internos, con persona vinculada a la
  identidad por fuente corporativa, lista positiva exacta de recursos,
  relacion/delegacion vigente, finalidad, campos, autenticacion reforzada,
  consultas acotadas, auditoria y revalidacion en la transaccion. Las
  cabeceras del cliente y las convenciones de nombres no son autoridad.
- Especificacion:
  `docs/estudio_requisitos/seguridad_y_despliegue_cronos.md`.

## DEC-027 — Ninguna autoridad por omision, inferencia o prioridad de alias

- Estado: aplicado a la identidad VEC y al ambito de convocatoria del servicio
  candidato heredado; las superficies heredadas siguen sin ser productivas.
- Convocatoria: crear un candidato exige una referencia explicita, canonica,
  sin comodines y exactamente igual a la configurada para las reglas activas.
  Ya no existe `convocatoria-default`. La vinculacion candidato-convocatoria se
  recupera del repositorio durable; un reinicio, una vinculacion ausente o una
  convocatoria distinta deniegan en vez de reconstruir un valor de reserva.
- Calculo: el resultado de autobaremo debe nombrar exactamente la convocatoria
  y version de reglas configuradas. Un calculador que devuelve otro proceso no
  produce expediente.
- Identidad: sujeto autenticado no implica rol `ciudadano`, y el nombre
  declarado del mecanismo no implica nivel de garantia. Rol, perfil y garantia
  deben proceder de evidencia positiva del conector y del repositorio de
  acceso. Dos alias autoritativos contradictorios invalidan toda la asercion;
  el orden de las cabeceras nunca decide cual prevalece.
- Certificado: solo cuenta el par de un handshake TLS directo con cadena
  verificada. Un PEM reenviado mas `SUCCESS` se ignora hasta disponer de una
  asercion criptografica breve con audiencia, enlace de canal y
  antirrepeticion. Confiar en una CIDR no crea esa prueba.
- Compatibilidad: los atajos de identidad que necesitan fixtures antiguos
  viven exclusivamente en el resolvedor compilado dentro de pruebas. No se
  comparten con el codigo productivo ni con el fichero fake seguro.
- Motivo: un valor predeterminado puede transformar ausencia de evidencia en
  acceso. La politica del proyecto es monotona de menos a mas: solo una
  concesion exacta y demostrable amplia capacidades.

## DEC-028 — Evidencia opaca y consumo unico junto con el efecto

- Estado: contrato general y primer efecto documental aplicados y probados en
  memoria; PostgreSQL y los restantes efectos siguen pendientes y cerrados.
- Decision: despues de la ultima revalidacion, el caso de uso crea una
  capacidad opaca sobre la decision reforzada. El repositorio deriva por si
  mismo la identidad y huella del efecto, compara la tupla funcional completa y
  consume la decision bajo la misma transaccion o seccion critica que confirma
  agregado, auditoria y outbox.
- Alcance cerrado: principal, perfil, accion de la version publicada de la
  plantilla, modulo, tipo y referencia de recurso, huella de contexto,
  finalidad, correlacion y garantia deben coincidir exactamente. Una decision
  con campos que el caso no consume, obligaciones no acreditadas, comodines,
  caducidad o cualquier discordancia se deniega sin efecto parcial.
- Reintentos: repetir exactamente el mismo efecto es idempotente y no duplica
  auditoria ni eventos. Reutilizar la decision para otro identificador,
  contenido o metadato se deniega. Las pruebas concurrentes confirman una sola
  escritura y un solo ganador cuando dos efectos compiten por la misma
  decision.
- Limite criptografico: la huella canonica evita ambiguedades y mutacion
  accidental, pero no es una firma del PDP. La composicion productiva exige
  atestacion versionada verificable y revalidacion de asignacion, rol y
  politicas dentro de la misma transaccion PostgreSQL del efecto.
- Limite distribuido: el binario se guarda de forma idempotente antes de la
  confirmacion; si la autorizacion caduca se conserva un huerfano recuperable.
  Produccion necesita intencion durable, promocion/recoleccion y reconciliacion,
  no una falsa transaccion distribuida con el almacen de objetos.

## DEC-029 — Conector OSRM cerrado por configuracion positiva completa

- Estado: aplicado en configuracion, composicion, transporte HTTP, pruebas y
  manual de implantacion; la red de produccion sigue pendiente de aprobacion por
  Sistemas y no se inventa en codigo.
- Decision: URL base, nombre de ambito, limites geograficos y CIDR de destino son
  cuatro valores obligatorios y atomicos. Si no se configura ninguno, Dietas
  permanece sin motor; si se configura solo una parte o un valor no es canonico,
  el adaptador no se construye.
- Destino: la URL aprueba un esquema `http` o `https`, un host y un puerto
  exactos. Se rechazan credenciales, ruta, consulta, fragmento, hostname
  ambiguo, puerto no canonico y el servicio publico de demostracion.
- Red: `VEC_OSRM_ALLOWED_CIDRS` es una lista positiva sin valor predeterminado;
  no admite `/0`, duplicados ni prefijos no canonicos. El host se resuelve en el
  momento de conectar y solo se marca una IP incluida. No se heredan proxies del
  entorno.
- Redirecciones: el cliente dedicado no sigue ninguna. Una respuesta `3xx` es un
  fallo del conector y no puede cambiar host, puerto, esquema o red.
- Privacidad: la respuesta al portal ya no publica la URL interna del motor; solo
  identifica el tipo de motor y el ambito funcional autorizado.
- Contrato: la peticion rechaza campos desconocidos, un segundo valor JSON y un
  numero de alternativas fuera del minimo documentado. La respuesta queda
  limitada a 20 MiB y solo se reenvia si contiene `code="Ok"` y una lista de
  rutas; un cuerpo sobredimensionado, malformado o semanticamente distinto
  falla cerrado.
- Pruebas: configuracion ausente conserva `503`; configuracion parcial,
  malformada, universal o no canonica falla al construir; una IP fuera de CIDR no
  recibe trafico; una redireccion no alcanza el segundo servidor; la
  configuracion explicita valida calcula la ruta.

## DEC-030 — Perfiles fake independientes sin administrador universal

- Estado: aplicado al traductor temporal de la carcasa, a los fixtures HTTP y
  a su matriz de regresion. No sustituye el PDP productivo.
- Hallazgo: `administrador`, `system_admin` y `jefatura_rrhh` compartian un
  paquete privilegiado con todos los permisos de Personal, Cronos, Dietas y
  Bolsa. Tecnicos, administrativos, empleados y jefaturas operativas heredaban
  a su vez listas y acciones globales sin persona, unidad, relacion,
  convocatoria, tarea ni expediente resueltos. Las pruebas usaban ese
  administrador universal para validar funcionalidades no administrativas.
- Decision: el traductor exige exactamente un perfil canonico y despacha a una
  lista positiva independiente por perfil; no combina roles, no recorta
  espacios, no corrige mayusculas ni aplica precedencia. Ausencia, repeticion,
  mezcla, alias adicional o valor desconocido dejan la lista vacia.
- Administracion tecnica: `administrador/system_admin` solo recibe carcasa y
  las capacidades tecnicas expresas de roles, catalogos, integraciones y
  monitorizacion. No recibe Personal, Cronos, Dietas ni Bolsa. Tampoco recibe
  la auditoria generica, porque todavia puede revelar referencias funcionales;
  la vista tecnica minimizada sera otro caso de uso.
- RRHH: `jefatura_rrhh` no recibe roles, integraciones ni seguridad.
  `tecnico_rrhh/validator_l2` y `administrativo/validator_l1` no reciben por
  defecto Cronos, nominas, Dietas ni Administracion. Ninguno recibe listas o
  expedientes funcionales hasta que el servidor aporte asignacion y recurso
  exactos.
- Personal y jefaturas operativas: `personal_interno`, `jefe_servicio` y
  `jefe_seccion` solo reciben la carcasa. Titularidad, subordinacion,
  delegacion, unidad y periodo no se deducen del nombre del rol.
- Demo ciudadana: conserva solo las entradas necesarias para representar el
  area propia de Bolsa. Esos permisos filtran el menu y no autorizan por si
  solos una futura operacion; cada ruta necesitara titular o representacion y
  recurso resueltos en servidor.
- Pruebas: existe una matriz exacta y otra negativa para cada perfil frente a
  todos los permisos publicados por Administracion, Personal, Cronos, Dietas
  y Bolsa, incluido el workspace. Los tests de conectores y catalogos usan un
  principal con la unica capacidad que ejercitan directamente en el manejador;
  los tests del modo fake verifican `403` y ausencia de menus funcionales. No
  se ha creado otro superusuario de prueba.
- Especificacion:
  `docs/portal_vec/autenticacion_fake_local_segura.md` y
  `docs/portal_vec/matriz_roles_y_ambitos.md`.

## DEC-031 — Contencion por lista blanca de la API Bolsa heredada

- Estado: aplicada como barrera temporal; la API heredada sigue siendo un
  prototipo y no sustituye al PDP RBAC+ABAC del nucleo.
- Identidad: el adaptador de cabeceras confiables y el manejador admiten un
  unico perfil canonico. Si llegan dos perfiles, incluso validos, no se elige
  uno por orden ni se suman capacidades: la peticion queda sin autoridad. Un
  perfil desconocido adicional tampoco se descarta para conservar el valido;
  cabeceras autoritativas repetidas o valores con espacios no canonicos se
  rechazan completos.
- Separacion: `system_admin` administra estado y capacidades tecnicas, pero no
  se considera personal de seleccion. No puede ejecutar la demo funcional,
  abrir el portal de tramitacion ni consultar candidatos, baremos,
  notificaciones o auditoria funcional. Los validadores tampoco heredan las
  rutas tecnicas de administracion.
- Descubrimiento: la raiz y el portal anuncian exclusivamente las rutas de su
  perfil. Un ciudadano no recibe el inventario interno y un administrador
  tecnico no recibe rutas de expedientes. Publicar una ruta no la autoriza; el
  control del caso de uso sigue siendo obligatorio.
- Alcances: las consultas heredadas de notificaciones y auditoria exigen una
  unica referencia de candidato exacta. Ausencia, repeticion, mezcla de
  parametros, espacios, comodines, controles o ambito de otro tipo se rechazan
  antes del repositorio. El identificador de la URL debe coincidir exactamente
  con el sujeto para las operaciones propias.
- Entradas: el JSON rechaza campos desconocidos y cualquier segundo valor; no
  se corrige silenciosamente una peticion ambigua.
- Limite: `validator_l1` y `validator_l2` siguen siendo perfiles gruesos del
  prototipo. Antes de produccion cada accion debe migrarse al autorizador
  central y resolver convocatoria, asignacion, recurso, finalidad, campos y
  obligaciones dentro de la operacion correspondiente.

## DEC-032 — Perfil candidato para atestar decisiones; puerta aun cerrada

- Nota de vigencia: el mensaje provisional de 30 campos descrito en esta
  decision historica fue sustituido por DEC-037. No es una version aceptable,
  no se reconstruye y no dispone de fallback.
- Estado: borrador parcialmente implementado y no aprobado para produccion.
  Existen serializador Go, pruebas adversariales y puerto estructural de firma;
  faltan ligadura de autenticacion real, perfil aprobado, verificador, catalogo
  de claves, PostgreSQL y consumo atomico.
- Decision: el borrador firma una representacion binaria versionada de los 30
  campos actuales de la decision mas suite, clave y audiencia. Esos campos no
  se congelan como V1 hasta ligar metodo, garantia observada, sesion/asercion y
  huella probatoria de autenticacion. La clave privada vive
  fuera de PostgreSQL y de la credencial de registro; la base conserva material
  publico, gobierno y revocacion.
- Perfil a evaluar: Ed25519 verificado mediante una version fijada y auditada de
  `pgsodium`. No se presupone soporte de firma en `pgcrypto`, no se fija
  RSA-PSS sin verificador concreto y no existe fallback HMAC automatico.
- Efecto: verificar al registrar no basta. Cada repositorio de negocio vuelve a
  verificar y consume la decision dentro de la misma transaccion que escribe
  agregado, hecho, auditoria y outbox.
- Aprobacion: algoritmo, extension, HSM/KMS, cadena de suministro, privilegios,
  rotacion, revocacion, recuperacion y vectores Go/PostgreSQL requieren dictamen
  de Sistemas y Seguridad. Cualquier ausencia mantiene el adaptador sin montar.
- Especificacion:
  `docs/portal_vec/atestacion_criptografica_decisiones.md`.

## DEC-033 — Concesiones ejecutables y denegaciones probatorias separadas

- Estado: contrato, servicio y adaptador de memoria aplicados y probados;
  persistencia append-only PostgreSQL de denegaciones pendiente.
- Hallazgo: el puerto unico trataba concesiones y denegaciones como si tuvieran
  la misma semantica. PostgreSQL rechazaba correctamente una denegacion en la
  tabla de capacidades, pero el servicio sustituia entonces su causa funcional
  por `registro_no_disponible`.
- Decision: `RegistroDecisionesAutorizacion` acepta exclusivamente concesiones
  ejecutables y conserva su CAS. `RegistroDenegacionesAutorizacion` registra
  exclusivamente resultados negativos y nunca puede producir una capacidad.
  El servicio elige uno u otro despues de validar la decision; no existe
  fallback entre ambos.
- Fallo cerrado: si la traza negativa no esta disponible, la operacion sigue
  denegada, conserva su codigo funcional y propaga ademas
  `ErrRegistroDenegacionNoDisponible` para que la perdida de evidencia sea
  observable. Nunca se reintenta como concesion.
- Separacion: el adaptador de memoria usa colecciones distintas y una
  denegacion no aparece en la consulta de concesiones. El adaptador productivo
  necesitara tabla, rol y funcion append-only propios, sin lectura ni consumo
  desde los repositorios de negocio.
- Limite: este corte mejora la trazabilidad negativa, pero no cierra el P0 de
  atestacion criptografica ni autoriza a montar PostgreSQL en produccion.

## DEC-034 — Contexto canonico de actor con perfil expreso y denegacion por defecto

- Estado: primer corte de dominio, puerto, servicio de aplicacion y adaptador
  en memoria implementado y probado. No esta conectado a HTTP, al arranque
  productivo ni al workspace.
- Hallazgo: la cuenta tecnica autenticada no equivale por si sola a una persona
  canonica ni determina un perfil. Elegir el primer enlace, inferir un perfil o
  mezclar identificadores de candidato y empleado permitiria confundir sujetos
  y ampliar autoridad entre modulos.
- Entrada exacta: la resolucion exige una cuenta autenticada canonica y una
  unica referencia de perfil solicitada expresamente. No se recortan ni
  normalizan valores, no se admiten comodines, alias o perfiles implicitos y no
  existe un perfil predeterminado.
- Coincidencia unica: la fuente debe devolver todas las instantaneas que
  coincidan exactamente con cuenta y perfil, sin `LIMIT 1`, prioridad ni
  precedencia. Solo una coincidencia produce contexto. Cero o dos o mas
  coincidencias obtienen la misma denegacion y nunca se selecciona la primera,
  aunque varias apunten a la misma persona.
- Denegacion por defecto: solicitud no canonica, perfil ausente, ajeno,
  revocado, futuro o caducado, enlace no vigente, dependencia no disponible,
  contexto cancelado o resultado ambiguo fallan cerrado. Ninguna ausencia se
  interpreta como autorizacion: toda operacion sin concesion positiva, concreta
  y exacta del PDP permanece denegada.
- Minimizacion: la resolucion no recibe, consulta ni conserva DNI, nombre,
  correo u otros datos personales. Cuenta, persona, perfil, candidato y
  empleado se representan mediante referencias opacas de tipo diferenciado.
- Enlaces versionados: la instantanea conserva identidad y version del enlace
  cuenta-persona-perfil, versiones de persona y perfil, estado y ventana de
  vigencia. Cada referencia opaca de candidato o empleado dispone asimismo de
  identidad de enlace, version, estado y vigencia propios. Duplicados y
  combinaciones de tipo y prefijo incompatibles se rechazan, no se corrigen.
- Salida sin autoridad fabricada: `Principal.ID` usa la persona canonica y
  conserva exclusivamente metodo y nivel de garantia autenticados. El contexto
  no incorpora nombre, correo, atributos, roles ni permisos y no sustituye la
  autorizacion RBAC+ABAC de cada operacion. `PerfilActivoRef` y las referencias
  opacas de candidato o empleado quedan disponibles para resolver despues el
  recurso y ambito exactos.
- Implementado: tipos y validacion de dominio, contrato de consulta completa,
  resolucion de coincidencia unica, copias defensivas, orden canonico y almacen
  inmutable en memoria protegido para lecturas concurrentes. Las pruebas cubren
  ambiguedad, perfiles y enlaces no vigentes, entradas no canonicas, fallos de
  dependencia, ausencia de filtracion de su causa, copias defensivas y carreras.
- Cerrado: no existe todavia adaptador productivo PostgreSQL u Oracle, esquema
  de unicidad ni politica RLS para estos enlaces. HTTP y `bootstrap` no aceptan
  ni construyen este contexto y el workspace agregado continua en `503`; nunca
  se confia en un contexto aportado por el cliente. Tampoco esta implementada la
  revalidacion autoritativa de versiones, vigencias y revocaciones dentro de la
  misma transaccion que ejecute cada efecto de negocio. Hasta cerrar esas tres
  piezas no se abre ninguna superficie nueva ni Cotejo.
- Relacion: concreta para la identidad canonica las reglas de denegacion por
  defecto de DEC-020 y de ausencia de autoridad por omision, inferencia o
  prioridad de DEC-027.

## DEC-035 — Llamamiento determinista sobre lista constituida y primer elegible

- Estado: dominio puro VS9, caso de uso, puertos y adaptador concurrente de
  memoria para pruebas implementados con pruebas adversariales. No existen
  repositorio productivo, atestacion criptografica del motor ni API; por tanto,
  este corte no abre ninguna operacion de llamamiento.
- Decision: un llamamiento solo puede proponerse desde una bolsa constituida,
  una necesidad de cobertura, una instantanea exacta del orden y una politica
  gobernada, todas versionadas y ligadas por referencias y huellas SHA-256. La
  necesidad conserva ademas la version y huella exactas de la bolsa para que no
  pueda reutilizarse con otra constitucion o revision del listado.
- Primer elegible: la propuesta conserva el prefijo completo y contiguo de la
  instantanea desde la posicion 1 hasta la persona seleccionada. Cada posicion
  anterior debe contener un recibo valido con resultado `no_elegible` y la
  ultima debe ser la primera con resultado `elegible`. No se admite omitir una
  posicion, empezar en otra, incluir otra elegible antes ni continuar evaluando
  personas posteriores. La cardinalidad total de la instantanea se conserva
  para poder demostrar el alcance examinado sin exponer el resto del orden.
- Ligaduras probatorias: cada evaluacion identifica participacion, sujeto
  opaco, orden y situacion vigente; liga referencia, version y huella de la
  necesidad, instantanea y politica; y conserva referencias y huellas distintas
  para entrada y resultado del motor. La propuesta liga asimismo bolsa,
  necesidad, instantanea, politica, seleccion y todas las evaluaciones mediante
  una huella de contenido determinista. Las decisiones de cambio de situacion y
  los recibos no pueden reutilizarse dentro del agregado.
- Cronologia: los constructores canonizan UTC con precision de microsegundo y
  rechazan instantes no serializables. Se exige
  `referencia <= generacion de instantanea <= evaluacion <= generacion de propuesta`.
  Bolsa y politica deben estar vigentes tanto en el instante de referencia como
  al generar la propuesta, y esta debe producirse antes del fin semicerrado de
  la necesidad. Una caducidad intermedia no se interpreta como permiso.
- Privacidad: el dominio no recibe DNI, NIE, nombre, correo ni otros atributos
  personales; trabaja con referencias internas opacas. Como defensa adicional
  rechaza formatos evidentes de DNI/NIE, incluidos separadores, y etiquetas de
  DNI, NIE, NIF o pasaporte tanto en referencias como en claves gobernadas.
  Esta defensa sintactica no desclasifica los identificadores seudonimos: la
  propuesta completa seguira siendo informacion personal de acceso interno y
  una futura vista publica debera proyectar solo datos expresamente publicables.
- Denegacion por defecto: ausencia de evaluaciones, resultado desconocido,
  hueco o duplicado en el orden, mezcla de bolsa, necesidad, instantanea,
  politica, estado o sujeto, huella cruzada, recibo repetido, cronologia
  imposible, caducidad o cardinalidad fuera de limite impiden crear la
  propuesta. No hay estado, causa ni regla favorable predeterminados: estados,
  causas, requisitos y politicas son datos gobernados y versionados.
- Aplicacion: el iniciador no aporta participantes. El caso de uso resuelve un
  unico recurso autorizable, crea el vinculo V1, exige la decision central,
  obtiene necesidad, listado y politica de fuentes autoritativas, evalua en
  orden hasta el primer elegible y confirma propuesta, auditoria y outbox en
  una mutacion atomica. La unicidad por necesidad y huella impide que dos
  solicitudes concurrentes creen propuestas distintas; una repeticion exacta
  es idempotente.
- Puertas que permanecen cerradas: la futura aplicacion debera obtener las
  entradas desde una fuente durable y autoritativa del listado definitivo; no
  aceptara desde HTTP, CLI o MCP una coleccion de participantes como prueba de
  pertenencia. La persistencia productiva debera imponer unicidad global e
  inmutabilidad de recibos y decisiones, control de version e idempotencia. Las
  huellas aportan integridad referencial, pero no autenticidad: los recibos del
  motor y la propuesta necesitaran atestacion o firma verificable y gobierno de
  claves. Hasta implementar y probar repositorio durable, atestacion y API
  minimizada, no se montara ninguna ruta ni adaptador productivo.
- Escala pendiente: la instantanea actual conserva historias completas para
  poder verificar la situacion vigente. Antes de produccion se medira con
  volumen real y se valorara una proyeccion inmutable de situacion vigente,
  ligada a la historia por huella, junto con limites de peticion y de coste
  agregado.
- Relacion: aplica la denegacion por defecto de DEC-020, la ausencia de
  autoridad inferida de DEC-027 y la futura atestacion y consumo transaccional
  descritos en DEC-032.

## DEC-036 — Unico arranque canonico y presentacion sin permisos implicitos

- Estado: aplicadas las barreras inmediatas; separacion fisica completa de las
  superficies web y autorizacion de todos los casos de uso siguen pendientes.
- Inventario historico: `Baremador`, `Bolsa_Diputacion` y
  `Bolsa_Diputacion_app` son material de referencia no desplegable. Pueden
  carecer de identidad, autorizacion o auditoria central y contener datos de
  referencia; el contexto Docker y la copia cerrada del `Dockerfile` los
  excluyen de todas las capas. No se reutiliza ninguno como ejecutable.
- Entrada unica: solo `cmd/vec-server` puede escuchar. El alias
  `cmd/bolsa-server` es un centinela retirado que termina con error y no puede
  omitir TLS ni crear otra composicion. La direccion predeterminada es
  `127.0.0.1:8080` y las redes cliente tambien parten limitadas a loopback.
- Presentacion cerrada: un rol ausente, desconocido o ambiguo obtiene el perfil
  visual `sin_acceso`, sin modulos ni filas. Una fila sin modulo declarado no
  se interpreta como global. El administrador tecnico solo ve carcasa y
  administracion tecnica; no hereda Personal, Cronos, Dietas o Bolsa. Los
  perfiles de ciudadano y autoservicio tampoco reciben la cola operativa: la
  interfaz no intenta deducir titularidad buscando nombres, DNI, CSV ni otros
  fragmentos en el contenido. Solo una proyeccion del servidor ligada al sujeto
  autenticado podra mostrar expedientes propios.
- Aserciones exactas: sujeto, roles, metodo y garantia de autenticacion no se
  corrigen para conceder. Mayusculas, espacios, separadores alternativos,
  duplicados o combinaciones no publicadas fallan cerrados antes de resolver
  permisos. Nombre, correo y DNI auxiliares siguen la misma regla: una
  repeticion, alias contradictorio o cabecera explicitamente vacia invalida la
  identidad completa, aunque esos datos no concedan permisos. La confianza de
  red en un proxy no convierte una cabecera ambigua en autoridad.
- Autoridad: estas barreras del navegador son defensa y claridad, nunca una
  autorizacion. El servidor sigue decidiendo por permiso positivo exacto y el
  objetivo obligatorio es que cada caso de uso exija identidad canonica,
  perfil, accion, recurso, finalidad y evidencia de autorizacion, cualquiera
  que sea el adaptador HTTP, CLI o MCP.
- Contexto de ejecucion: los casos heredados de Bolsa ya no convierten un
  contexto nulo en `context.Background()`. Nulo, cancelado o vencido detienen la
  operacion antes de puertos y se vuelven a comprobar entre escrituras y antes
  de la auditoria. Esto evita efectos posteriores a perder la sesion, pero no
  sustituye la autorizacion funcional pendiente dentro de esos casos de uso.
- Puerta pendiente: antes de produccion se generan paquetes, dominios, sesiones
  y redes separados para portal exterior, portal interno y administracion
  privilegiada. La auditoria durable y el consumo atomico deben sustituir las
  mutaciones heredadas que hoy registran su traza despues del cambio.

## DEC-037 — Vinculo obligatorio autenticacion, sesion y actor en autorizacion

- Estado: implementado en dominio, serializacion Go y capacidad de uso; la
  composicion de identidad y PostgreSQL permanecen cerradas.
- Decision: ninguna `DecisionAutorizacion` es valida sin un
  `VinculoAutenticacionActorV1`. El bloque contiene exactamente 25 datos
  obligatorios: version, documentos y huellas de autenticacion, asercion,
  sesion y control; cuenta ordinaria/privilegiada; persona canonica y perfil
  activo explicitos; superficie, metodo y garantia observados; politica de
  garantia y tiempos; documento, version y huella del contexto de actor.
- Autoridad: el vinculo es una capacidad opaca, no rellenable mediante literal
  ni reconstruible desde JSON o texto. Su fabrica invoca una fuente de sesion
  autoritativa y cruza el resultado con un `ContextoActor` ya resuelto. Cuenta,
  metodo y garantia deben coincidir; la huella del actor liga persona, perfil,
  versiones, vigencias y referencias de modulo. Un DTO nunca sustituye esa
  revalidacion.
- Menor privilegio: superficie desconocida o anonima, metodo `demo`, valor cero,
  campo ausente, referencia no opaca, revision cero, huella no SHA-256, tiempo
  no canonico o mezcla entre sesion, cuenta, perfil y actor deniegan. Una cuenta
  ordinaria coincide con `cuenta_ordinaria_ref`; una privilegiada exige cuentas
  distintas y solo existe en `administracion_privilegiada`.
- Cierre de mezcla: `principal_id` y `perfil_activo_ref` del bloque deben
  coincidir exactamente con el `ContextoActor` al emitir y con los campos de la
  decision al validar, registrar, serializar y consumir. Una huella valida de
  otro actor no autoriza a combinar vinculo A con decision B. Dos vinculos solo
  coinciden cuando ambos son validos y sus 25 datos son exactamente iguales,
  sin normalizacion ni igualdad positiva de valores cero.
- Tiempo: se exige `autenticacion_verificada_en <= sesion_emitida_en <=
  sesion_revalidada_en < sesion_valida_hasta`. La decision se emite despues de
  revalidar y su caducidad es el minimo positivo entre configuracion, sesion,
  contexto de actor y sus referencias, asignacion y fronteras de politicas.
- Frescura pendiente: que la fabrica consulte sincronicamente la fuente no
  demuestra por si solo que `sesion_revalidada_en` sea suficientemente reciente.
  Antes de una superficie interna o privilegiada debe existir una antiguedad
  maxima gobernada y comprobada, o una revalidacion/fence autoritativa dentro de
  la transaccion del efecto. Un TTL corto de la decision no sustituye este
  control y, mientras falte, esas composiciones permanecen cerradas.
- Multifactor pendiente: V1 conserva un unico `MetodoObservado` y un nivel de
  garantia; eso no prueba que certificado y Kerberos hayan sido dos factores
  independientes ni que pertenezcan a la misma persona y sesion. El conector de
  identidad o una atestacion aprobada debe aportar el conjunto real de factores,
  su enlace e independencia. No se deduce doble factor por una garantia alta.
- Criptografia: `VEC-AD-1` cambia su separador a
  `VEC-AUTORIZACION-ATESTACION-V1-AUTENTICACION-ACTOR`, expande los 25 datos en
  orden fijo y publica un vector nuevo. La huella canonica de evidencia de uso
  cambia tambien de dominio. El borrador anterior de 30 campos no tiene
  fallback ni se acepta como V1 actual.
- Persistencia cerrada: serializar el bloque permite firmar y almacenar
  evidencia, pero no reconstituye la capacidad. PostgreSQL no se abre hasta
  disponer de migracion, deteccion de campos duplicados, reconstruccion binaria
  independiente, verificacion de firma y revalidacion transaccional de sesion,
  cuentas, actor y consumo junto al efecto.
- Pruebas: reflexion de los 25 datos y de la decision, vector binario fijo,
  decodificador independiente, mutacion del vinculo, cada ausencia, cruces de
  cuenta/perfil/superficie, cronologia, microsegundos, anos `0001..9999`, valor
  cero, JSON no rehidratable, cancelacion y limites temporales. Se ejecutan
  tambien en `linux/386` para detectar conversiones dependientes de palabra.
- API temporal: `DecisionAutorizacion.VigenteEn` solo devuelve verdadero si la
  evidencia instantanea completa sigue siendo valida; una estructura meramente
  concedida y dentro de fechas no se presenta como vigente.

## DEC-038 — Evidencia V1/PDP en baremacion y barrera transaccional pendiente

- Estado: transporte de autorizacion implementado en aplicacion y puertos. Esta
  implementada y probada unicamente la simulacion defensiva en memoria del
  consumo de reserva, confirmacion y abandono. Adaptadores productivos y
  exposicion HTTP, CLI y MCP denegados.
- Decision: cada operacion de esta vertical obtiene exactamente una sesion de
  la fuente confiable, cruza `ContextoActor` y `VinculoAutenticacionActorV1`, llama
  al PDP central y exige principal, perfil, accion, recurso, finalidad,
  correlacion y campos exactos, sin obligaciones desconocidas. Cero o varias
  sesiones, cualquier mezcla o una decision incompleta deniegan antes del
  repositorio.
- Capacidad: `ContextoOperacionBaremacion` conserva una
  `EvidenciaUsoDecisionAutorizacion` opaca, inmutable y no rehidratable. La
  reserva y la confirmacion deben compartir el vinculo V1 exacto, aunque sus
  acciones y decisiones sean distintas. Adoptar, rectificar, revocar,
  rehabilitar, firmar, custodiar y cada lectura son concesiones positivas
  independientes; una no hereda otra.
- Memoria: el adaptador exclusivo de pruebas valida vigencia dentro de su mutex,
  exige el vinculo exacto reserva-confirmacion y simula para reservar, confirmar
  y abandonar el consumo unico
  `DecisionRef -> (huella de decision, huella de efecto)`. Al materializar sus
  controles desde la propia evidencia no demuestra independencia frente al PDP
  ni autoriza despliegue.
- TOCTOU productivo: el repositorio durable debe volver a leer y comparar, en la
  misma transaccion que el efecto, decision, asignacion, version/control de rol,
  catalogo, sesion y contexto de actor actuales; consumir la `DecisionRef`; y
  confirmar agregado, auditoria y outbox o ninguno. PostgreSQL/Oracle, IdP,
  registro de sesiones y verificador criptografico aun no implementan esta
  composicion. Validar antes de la transaccion o confiar en el TTL falla
  cerrado. Este CAS y esta relectura autoritativa independiente no estan
  resueltos por la simulacion en memoria.
- Lecturas sensibles: obtener baremacion, versiones, evidencias, representacion
  o artefactos firmados exige revalidar autoridad y alcance de fila en la misma
  instantanea fiable de lectura. Los filtros o comprobaciones anteriores al
  repositorio no bastan; mientras falte esta barrera, no se exponen.
- Firma y custodia: firma, sello de tiempo, aumento y custodia pueden ser
  irreversibles y ocurren antes del OCC final. Produccion requiere intencion
  durable previa, claves idempotentes, manifiesto archivado, validacion y
  custodia reales, y reconciliacion de huerfanos sin repetir ni ocultar una
  firma. El recurso de almacen ya se enriquece antes del PDP con operacion,
  carga, clasificacion, seudonimo HMAC, huella del plan y `EffectRef`; queda
  copiado dentro de la capacidad, y tres puentes nominales separan custodia
  canonica, custodia firmada y retencion de objeto+version. La fabrica anterior
  sin esos atributos deniega. Recuperacion, escritura y retencion usan
  decisiones distintas con el mismo vinculo V1. Como la decision de retencion
  exacta solo puede emitirse despues de conocer el resultado de escritura,
  sigue pendiente la intencion durable y el reconciliador entre ambos pasos.
  Los dobles actuales solo prueban el contrato y no existe cableado productivo.
- Identidad pendiente: se aplican tambien las puertas de frescura y multifactor
  de DEC-037. No se abre el portal interno por observar Kerberos, certificado o
  una garantia alta sin atestacion independiente y revalidacion autoritativa.
- Relacion: concreta para baremacion la lista positiva y el consumo de DEC-020,
  la evidencia opaca de DEC-028 y el vinculo V1 de DEC-037.

## DEC-039 — Cierre preventivo de superficies heredadas sin evidencia V1

- Estado: auditoria adversaria realizada el 15 de julio de 2026. Las
  superficies afectadas permanecen sin adaptador productivo y sin exposicion
  HTTP, CLI o MCP. No forman parte de la base segura aceptada para despliegue.
- Hallazgo inicial: `ContextoOperacionAlmacen` y el antiguo
  `ContextoProteccionCodigoCotejo` transportaban referencias declarativas en
  estructuras publicas; por si solas no demostraban una concesion vigente. Los
  servicios heredados de catalogos, flujos, cargas, documentos logicos y
  cotejo tampoco consumen aun `EvidenciaUsoDecisionAutorizacion` junto con cada
  efecto. Una referencia con formato valido nunca equivale a permiso.
- Identidad: el auxiliar de autorizacion heredado no aporta todavia
  `ContextoActor` y `VinculoAutenticacionActorV1` a la solicitud central. Con
  el PDP real falla cerrado; no se permite sustituirlo por un autorizador mas
  permisivo para hacer funcionar una integracion.
- Puerta de producto: ninguna de esas capacidades se cableara en el arranque
  hasta que use una capacidad opaca derivada de evidencia V1, declare un mapeo
  positivo entre accion de negocio y accion tecnica, relea todos los controles
  autoritativos y consuma `DecisionRef -> efecto` en la misma transaccion. Cada
  operacion de proteger, recuperar y eliminar custodia exige autoridad propia;
  la limpieza de huerfanos usa una identidad tecnica expresa y limitada.
- Adaptadores futuros: CLI, MCP, bots y tareas no reciben repositorios ni el
  servicio transversal generico. Solo reciben casos de uso protegidos. Registro
  de modulos de arranque, auditoria y eventos internos se separan de consultas
  y mutaciones invocables por usuario.
- HTTP actual: los permisos estaticos solo existen en el modo demostracion. En
  composicion real la lista queda vacia, por lo que las rutas funcionales
  deniegan. Antes de abrirlas resolveran perfil y recurso en servidor y llamaran
  al PDP por accion, ambito, finalidad, campos y sesion exactos.
- Correcciones inmediatas: las tres lecturas del repositorio de baremacion ya
  releen el reloj y revalidan dentro de `RLock`; la fabrica que habilita su
  adaptador efimero existe solo en ficheros `_test.go`. Esto no reduce los
  bloqueos productivos de DEC-038.
- Cotejo cerrado: el contexto publico anterior se ha eliminado. Proteger,
  recuperar y eliminar un huerfano usan tres capacidades opacas e
  incompatibles, derivadas de decisiones V1 distintas para accion, recurso,
  huella, finalidad y parametros exactos. El flujo interactivo solo puede
  proteger o recuperar; ante un fallo posterior abandona la reserva pero no
  inventa autoridad para borrar la custodia. La eliminacion queda bloqueada
  hasta disponer de worker interno, cuenta privilegiada, superficie
  administrativa, PDP exacto, intencion durable y reconciliacion.
- Superficie transversal cerrada: `Service` solo expone `Modules` y
  `BuildMenu`, y ambos vuelven a exigir dentro de aplicacion sus permisos
  positivos exactos. El catalogo se valida y copia en profundidad al entrar y
  salir para que una lectura no pueda mutar permisos. Registro de modulos,
  escritura/publicacion y consulta de auditoria viven en operaciones internas
  separadas; CLI y MCP no las reciben. La ayuda generica de auditoria del
  prototipo no es una API productiva: sigue cerrada hasta sustituirla por casos
  de uso con mapeo gobernado, evidencia V1, consumo unico, transaccion y outbox
  durables.
- Referencias de auditoria: la aplicacion, no solo HTTP, rechaza referencia
  vacia, no canonica, con espacios, controles, UTF-8 invalido, mas de 512 bytes
  o cualquier `*`. Una consulta global futura sera otro caso de uso con permiso,
  finalidad, limites y traza propios.
- Comodin restrictivo: `*` solo se admite como selector de una politica ABAC
  que reduce o deniega una concesion RBAC exacta previa. Nunca concede acceso.
  Es invalido en solicitudes, concesiones, permisos, acciones, recursos,
  ambitos, campos, finalidades y obligaciones positivas; su unico lugar
  posible es la expresion interna de una regla exclusivamente reductora.
  Si la organizacion prohibe tambien esa notacion de denegacion, se migrara a
  enumeracion sin cambiar el principio: ante ausencia o duda siempre gana la
  denegacion.

## DEC-040 — Capacidad opaca V1 y plan cerrado para el almacen documental

- Estado: contrato del nucleo y adaptadores de referencia implantados el 15 de
  julio de 2026. La preparacion de cargas ya conserva su manifiesto V1 en la
  misma transaccion que agregado, auditoria y outbox, y la generacion
  documental dispone de manifiesto compuesto y contrato de efectos durables.
  Intenciones remotas, reconciliacion y adaptadores productivos aun pendientes
  mantienen cerradas las operaciones que dependen de ellos.
- Decision: `ContextoOperacionAlmacen` deja de ser un conjunto publico de
  referencias. Es una capacidad opaca, inmutable, no serializable y derivada
  exclusivamente de una `DecisionAutorizacion` V1 positiva mediante fabricas
  especificas de lista cerrada.
- Vinculo previo al PDP: operacion, carga, clasificacion, sujeto
  seudonimizado, huella HMAC de solicitud, efecto y, cuando exista, objeto y
  version exactos forman parte del `RecursoAutorizable`. La fabrica vuelve a
  calcular su huella y exige coincidencia exacta con la decision. Anadir esos
  datos despues de autorizar no es valido.
- Semantica: accion de negocio, conjunto exacto de campos, finalidad, recurso,
  modulo, tipo, actor y sesion V1, vigencia y ausencia de obligaciones no
  implementadas se comprueban sin alias, herencia, union, comodines ni valores
  predeterminados. Una decision habilita un unico plan y solo sus pasos
  declarados. Derivar un paso selecciona; nunca amplia.
- Punto de efecto: los conectores usan una proyeccion defensiva solo para
  construir trazabilidad y revalidan la capacidad con su propio reloj
  inmediatamente antes de cada lectura o mutacion. La proyeccion no permite
  reconstruir autoridad.
- Operaciones cerradas: no existe fabrica positiva general. Leer para analisis,
  analizar, preparar/abandonar/confirmar carga, promover, custodiar decisiones
  de baremacion y retener una version firmada tienen puentes separados. Listar,
  inmovilizar, levantar, eliminar y cualquier accion nueva deniegan hasta que
  se especifiquen su caso de uso, decision, segregacion, plan, pruebas y
  reconciliacion propios; tampoco las habilita un administrador.
- Preparacion durable: reserva, agregado preparado, auditoria, outbox y
  `ManifiestoPreparacionCargaDirectaV1` se confirman atomicamente. El manifiesto
  conserva MIME, tamano, huella, clasificacion, sesion HMAC, conector, recurso,
  decision V1 y plan exactos; su rehidratacion no completa datos parciales. La
  confirmacion usa otra decision y el recibo atestado, y nunca reconstituye la
  capacidad de preparar. La reemision permanece denegada hasta poder ligar
  manifiesto y recibo en una unica frontera transaccional.
- Reclamacion previa: `DecisionRef` y su tupla exacta de efecto, plan, esquema,
  huella, indice, token, carga y arrendamiento se reclaman durablemente dentro
  de la reserva y antes de abrir una sesion remota. Una segunda carrera se
  corta antes del almacen. Los estados activa, consumida, abandonada y expirada
  son terminales respecto de esa decision: abandono o caducidad exigen una
  decision nueva y nunca reciclan la anterior. El consumo final ocurre en el
  mismo commit que manifiesto, agregado, auditoria y outbox.
- Instantaneas defensivas: preparacion y transiciones posteriores clonan una
  sola vez carga, auditoria, evento y manifiesto; validan y persisten exactamente
  esa copia. Volver a leer mapas o slices del argumento tras validar reabriria
  un TOCTOU y esta cubierto por pruebas de alias y carrera.
- Efecto durable pendiente: el adaptador productivo registrara antes del efecto
  `DecisionRef -> (EfectoRef, huellaDecisionV1, huellaPlan)` y por cada paso un
  resultado tecnico exacto. Reintentos identicos devolveran el mismo recibo;
  cualquier reutilizacion cruzada denegara. Una respuesta remota ambigua queda
  pendiente de reconciliacion y nunca autoriza una compensacion de borrado.
- Generacion configurable: una plantilla puede declarar permisos de negocio
  sin recompilar, pero eso no convierte cualquier decision en permiso de
  escritura. El puerto ya modela la generacion simple como `N=1` y la de varias
  representaciones como un unico manifiesto completo, versionado y ligado al
  recurso evaluado, con pasos exactos y estados reservado, confirmado o
  indeterminado. La composicion productiva solo se abrira cuando el caso de uso
  consuma ese registro durable y exista su adaptador transaccional real; no se
  sustituye por un registro en memoria.

## DEC-041 — Arranque de catalogos sin siembra ni importacion implicita

- Estado: aplicado al catalogo RPT/personal y cubierto por pruebas adversarias.
- Decision: construir o reiniciar el servicio es una operacion exclusivamente
  de lectura. Si la ruta durable no contiene un snapshot, el catalogo queda
  vacio y no se crea ningun fichero, copia de respaldo, categoria ni puesto.
- Fuentes externas: una variable de entorno, un JSON disponible en disco o un
  juego de datos demostrativo nunca desencadenan una importacion durante el
  arranque. Un snapshot preexistente se conserva byte a byte aunque
  `VEC_RPT_IMPORT_JSON` apunte a una orden de sustitucion valida u hostil.
- Administracion futura: sembrar, importar o reemplazar datos sera un caso de
  uso administrativo expreso, con permiso positivo exacto, origen y huella
  autorizados, validacion previa, escritura atomica, auditoria y recibo. Hasta
  implementar toda esa cadena, la composicion productiva no lo ejecuta.
- Pruebas: los 842 puestos, categorias y catalogos demostrativos se cargan
  unicamente desde un auxiliar compilado en `_test.go`; no forman parte del
  constructor ni de un comportamiento predeterminado de produccion.

## DEC-042 — Formatos gobernados y metadatos institucionales de procedencia

- Estado: decision arquitectonica adoptada como contratos V2/V3 en sombra el
  15 de julio de 2026. El ejecutor V2 de borradores fue retirado del paquete
  activo el 16 de julio de 2026 al quedar superado por V3/V4; se conservan los
  contratos de dominio y puertos que V3/V4 reutilizan y sus hallazgos como
  requisitos de regresion. Cercado, recibos y pruebas adversarias permanecen
  sin cableado productivo. PDF y DOCX siguen siendo por ahora los unicos
  formatos ejecutables del prototipo heredado.
- Formatos: el nucleo no mantendra una lista compilada de PDF, DOCX, ODT, CSV,
  TXT, JSON u otros. Separara `FormatoID`, perfil inmutable y versionado,
  revision publicada del catalogo y conector instalado y homologado. Una
  plantilla fija referencias exactas; nunca adopta implicitamente la ultima
  version. Ausencia, ambiguedad, retirada o discrepancia de MIME, extension,
  esquema, capacidades o digest deniegan.
- Limite de configuracion: desde la aplicacion se podran crear y publicar
  perfiles cubiertos por esquemas cerrados y capacidades de conectores ya
  instalados. Un motor, parser, validador, clase de firma, CDR o cifrado nuevo
  exige instalar y homologar otro conector, pero no modificar el nucleo. El
  catalogo nunca contendra comandos, scripts, ejecutables, clases, imagenes de
  contenedor ni URL de descarga.
- Procedencia institucional: un perfil puede exigir una marca de procedencia
  no personal. La lista positiva inicial admite identificador de la entidad y
  del organo emisor, UUID aleatorio del documento, perfil y version, instante
  de creacion, URI de verificacion cuando la politica de acceso la permita y
  referencia opaca preasignada del manifiesto de integridad. No admite DNI,
  persona, destinatario, usuario interno, IP,
  ruta, nombre de servidor, detalle de infraestructura, version vulnerable,
  credencial, clave ni otro secreto.
- Momento criptografico: la marca se incorpora a los bytes antes de firma,
  sello o sello de tiempo. Un original firmado y custodiado es inmutable y no
  se reescribe para anadir, corregir o retirar metadatos. Una representacion
  posterior es otro artefacto derivado, con identidad, huella, relacion y, si
  procede, firma propias.
- Ausencia de autorreferencia: la marca embebida no contiene el SHA-256 de sus
  propios bytes ni el digest de un manifiesto que a su vez comprometa ese
  SHA-256. La referencia opaca del manifiesto puede reservarse antes; la huella
  final se calcula despues de marcar y queda ligada en el manifiesto, auditoria
  y firma externos. Asi se evita una dependencia circular imposible de
  reproducir.
- Representacion: PDF, ODT y DOCX usaran metadatos estandar cuando el perfil los
  soporte y el validador compruebe la salida. JSON o XML solo incluiran un
  bloque explicito definido en su esquema. CSV, TXT y cualquier formato sin un
  mecanismo fiable conservaran la procedencia en un manifiesto lateral
  firmado, ligado por SHA-256. No se insertaran espacios invisibles, caracteres
  de ancho cero ni alteraciones semanticas para ocultar una marca.
- Alcance probatorio: estos metadatos permiten reconocer el origen, pero pueden
  perderse durante una conversion y no sustituyen firma o sello electronico,
  hash, CSV/servicio de cotejo, custodia ni auditoria. Su ausencia en una copia
  no prueba que el documento sea ajeno; la verificacion autoritativa se realiza
  contra el artefacto canonico y su manifiesto.
- Privacidad y gobierno: al no individualizar destinatarios, la marca prevista
  no se usa para seguimiento personal. La finalidad, conjunto exacto de campos,
  conservacion y acceso se documentan por perfil y se someten a DPD y
  responsables juridico, funcional y de seguridad antes de produccion. Cualquier
  futura huella por destinatario seria otro tratamiento, desactivado por
  defecto, y requeriria una decision y evaluacion especificas.
- Corte implantado: identidad de sintaxis sin MIME ni extension; perfil
  historico inmutable con esquema, dialecto, canonicalizacion, reglas, politica
  y limite exactos; revision de catalogo con huella; y publicacion operativa
  separada, secuencial y revocable. Los componentes se atestiguan por funcion
  con identidad/version, homologacion, huella de artefacto, broker, dominio de
  confianza y techo. Renderizador y verificador deben ser independientes y
  ninguna interfaz ejecutable sale del caso de uso.
- Ejecucion V2 retirada: el prototipo demostro cardinalidad exactamente uno,
  techo igual al minimo institucional/perfil/componentes, escritor limitado,
  aislamiento de copias y relecturas contra TOCTOU. Su artefacto opaco y su
  evidencia restaurable no bastaban para acreditar atestacion, atomicidad ni
  recuperacion. Esas condiciones se mantienen como pruebas de regresion que
  V3/V4 deben satisfacer sin reintroducir el camino competidor.
- Migracion: el contrato V1 permanece solo para compatibilidad de compilacion.
  Su constructor de perfil no concede autoridad y sus servicios fallan cerrado
  antes de alcanzar renderizadores, marcadores o verificadores legacy.
- Puerta productiva: faltan catalogo y evidencias durables, broker que compruebe
  criptograficamente las atestaciones, reserva idempotente de referencias,
  anclaje autenticado de evidencia, reconciliacion, ejecucion en flujo hacia el
  almacen, verificacion independiente de equivalencia entre contenido neutral y
  salida, adaptadores de formatos y los contratos gobernados de
  marcado/extraccion y manifiesto lateral, pendientes de integrar en V4. El
  contrato nuevo no esta cableado al generador ni
  expuesto por HTTP, CLI o MCP; una resolucion previa nunca sera autoridad
  permanente.
- Auditoria adversaria: una revision independiente posterior a las pruebas
  confirmo que las relecturas reducen, pero no eliminan, el TOCTOU; el cierre
  definitivo exige reserva y confirmacion atomicas con token de cercado. Tambien
  constato que pasar un descriptor a un ejecutor no demuestra que el binario
  ejecutado sea el atestado, que el SHA restaurable no autentica por si solo la
  evidencia y que el corte en memoria puede multiplicar varias veces el techo
  nominal. Estos hallazgos se consideran bloqueos expresos de produccion, no
  deuda aceptada silenciosamente.
- Desarrollo del cierre: broker, recibos, verificacion semantica, reserva,
  cercado, reconciliacion, evidencia autenticada y ejecucion por flujo se
  especifican en `ejecucion_documental_atestada_v3.md`.
- Auditoria V3 y apertura de V4: aunque las pruebas normales, de carrera y
  analisis estatico son verdes, se comprobo que la API publica aun permite
  fabricar por forma resultados denominados verificados, que el consumo de
  decision no usa la capacidad opaca completa ya existente y que entrada,
  salida y persistencia no estan atestadas de extremo a extremo. Tambien faltan
  CAS y retos frescos en reconciliacion, cronologia durable completa, estados
  negativos/indeterminados y recuperacion segura tras reinicio. V4 migra a
  pruebas criptograficas crudas verificadas localmente, autorizacion exacta,
  entrada/salida inmutables y recibo de almacen; hasta entonces ninguna puerta
  publica o interna puede consumir estos contratos.
- Evidencia de Orquesta: Goal
  `goal:d415dddc4ef32c43c9d3f16e390f2dba`, AppSpec generacion 1 con SHA-256
  `580746d07a407e84a79a8da4f66fb4ef81be55c33d7fd4ffd05511f57a741745`.
  Artefactos inmutables `catalogo-formatos`
  (`cc0e630680c96812b45c5fc6358134d27c7380e57f004774290fbe707df1ec69`)
  y `seguridad-formatos`
  (`8076c1d7e7912d9fcd0908320fb4d5aa39e5409c1119948c055cbcf092c44bae`).
  El estado `succeeded` acredita ejecucion y evidencia, no integracion de codigo.

## DEC-043 — Emision cerrada y reverificacion del registro de atestacion PDP V4

- Estado: decision experimental anterior, complementada por DEC-044. La
  remediacion V4 actual esta validada tecnicamente y cerrada a produccion; el
  resultado aplicable y sus limites se registran en DEC-044.
- Ruta de autoridad: se elimina el constructor libre de solicitudes de registro
  y cualquier exportador desde la autoridad. La fachada neutral es
  `application.EjecutorDocumentalAtestadoV4`; recibe un puerto fijado en la raiz
  de composicion y no recibe DTO persistible, verificador, raiz, repositorio o
  instante elegido por el llamador. El servicio de confianza es detalle del
  adaptador PostgreSQL y no una dependencia de `application`.
- Reverificacion: inmediatamente antes de registrar se interpreta de nuevo el
  `VEC-AD-1` estricto y se verifica criptograficamente el COSE Sign1 completo
  con EdDSA, AAD externo, audiencia y `kid` cerrados y la raiz/configuracion
  actualmente vigente y no revocada. No se confia en una prueba historica ni
  en metadatos nominales aportados por el llamador.
- Enlace al adaptador: una pareja HMAC aleatoria y exclusiva de la instancia
  autentica las 25 claves de aplicacion, metadatos de confianza y todos los
  bloques binarios. El servicio conserva el emisor y solo entrega al constructor
  del registrador el verificador opaco; otra pareja deniega antes de `BeginTx`.
  Esta capacidad protege el paso interno, pero no sustituye la firma COSE.
- Igualdad exacta: el payload firmado se coteja contra los 30 campos superiores
  y los 25 datos de autenticacion-actor de la decision viva. Listas conservan
  orden, mapas exigen las mismas parejas, instantes son identicos y la preimagen
  canonica completa debe coincidir byte a byte con el recurso autorizado.
- Obligaciones: hasta implementar persistencia del mapa, prueba tipada de cada
  cumplimiento y revocacion, solo se aceptan obligaciones y cumplimientos
  vacios con sus huellas canonicas. No se interpretan como cumplidas por omision.
- Persistencia: PostgreSQL conserva su revalidacion dentro de la transaccion de
  decision, asignacion, rol, catalogo, sesion y contexto de actor, junto con
  unicidades y cercado. La reverificacion COSE del servicio y el `COMMIT` no son
  una sola transaccion; este limite TOCTOU queda documentado como puerta
  productiva pendiente, no como riesgo aceptado.
- Matriz de validacion pendiente: mutacion individual 30+25 y de las 25 claves
  persistibles, cambio de raiz, pareja incorrecta antes de transaccion,
  obligaciones no vacias,
  serializacion/cabeceras/CBOR adversarios, repeticion, carrera, analisis estatico
  e integracion PostgreSQL efimera con migraciones, privilegios e identidad
  autoritativa actual.

## DEC-044 — Puerto hexagonal y cierre operacional de ejecucion documental V4

- Estado: remediacion tecnica validada en corte limpio. **NO-GO productivo**
  hasta completar los controles operacionales de esta decision y obtener
  aprobacion de Sistemas y Seguridad.
- Nucleo: `application.EjecutorDocumentalAtestadoV4` depende unicamente de
  `ports.ConectorEjecucionDocumentalAtestadaV4`. El puerto transporta solicitud
  vinculada y sobre opacos y devuelve una confirmacion opaca sin secreto,
  credencial, COSE ni capacidad reutilizable.
- Adaptador: `pgx`, SQL, verificacion COSE, emision HMAC, socket Unix y
  repositorios residen en
  `internal/vec/adapters/postgres/confianzadocumental`. El constructor permite
  que la futura raiz de composicion productiva inyecte el ejecutor en
  `application`, pero este corte no lo conecta a HTTP, CLI ni MCP;
  `cmd/vec-emisor-capacidad-v4` compone directamente el emisor aislado.
- Sustitucion: Oracle u otro motor homologado implementara el mismo puerto sin
  modificar dominio ni caso de uso. El nuevo adaptador debe conservar consumo,
  efecto, auditoria y outbox en una frontera atomica; la intercambiabilidad no
  autoriza una version funcionalmente mas debil.
- ACL de tipos: la migracion revoca `USAGE`, fija los privilegios por defecto y
  una guarda DDL propiedad del DBA cierra tipos fila implicitos futuros. La
  aceptacion crea una tabla posterior al `up` y consulta privilegios efectivos.
- Membresias: `roles_down.sql` debe comparar los miembros reales con una lista
  positiva y fallar antes de revocar nada si encuentra uno inesperado. Un
  desmontaje aparentemente correcto nunca puede ocultar una membresia ajena.
- `pgcrypto`: se inventarian funciones y privilegios efectivos, se elimina la
  ejecucion general que no sea necesaria y el runtime solo alcanza la operacion
  HMAC minima por el envoltorio propietario. En una base compartida, cualquier
  revocacion global exige analisis de impacto y procedimiento aprobado.
- Desmontaje: eliminar el esquema o datos esta siempre denegado sin opcion
  explicita. El `down` elimina objetos conocidos con `RESTRICT`; dependencias
  externas u objetos futuros no soportados abortan sin mutacion persistente.
- Puerta operacional: nunca se reunen credenciales emisora y ejecutora, cuenta
  de servicio, volumen de secretos ni volcados. El directorio y socket Unix
  tienen ACL e identidades segregadas; la clave se custodia y rota mediante
  HSM/KMS o gestor homologado. Copia, restauracion, observabilidad sin secretos,
  respuesta a incidentes y operacion deben aprobarse antes de exponer el flujo.
- Aceptacion tecnica: pruebas unitarias, de carrera, analisis estatico,
  arquitectura, vulnerabilidades e integracion PostgreSQL 18.4 con identidades
  reales, privilegios, replay, concurrencia, revocacion y `up/down` terminaron
  correctamente el 15 de julio de 2026. No acredita las puertas operacionales.

## DEC-045 — Idempotencia semantica y reanudacion con autoridad vigente

- Estado: decision transversal adoptada el 15 de julio de 2026; implantacion
  pendiente. Los cortes de baremacion, generacion documental y carga directa
  permanecen cerrados a produccion hasta cumplirla.
- Problema: una clave idempotente no basta si el repositorio compara tambien
  autorizacion, sesion, referencias aleatorias o ventanas del intento. Tras un
  `COMMIT` cuya respuesta se pierde, una sesion nueva no recupera el resultado
  y puede provocar conflicto falso, documento huerfano o repeticion de firma,
  custodia, renderizado u otro efecto remoto.
- Indice estable: cada operacion usa un indice HMAC versionado sobre modulo,
  accion, principal estable y clave del cliente. La clave no se persiste en
  claro. La rotacion conserva un llavero de indices aun vigentes durante la
  retencion necesaria para recuperar operaciones historicas.
- Intencion estable: otra HMAC compromete todos los datos materiales exactos
  del efecto y del resultado esperado. Incluye agregado/version base,
  decisiones, politicas, huellas, objetos, retencion y motivo cuando procedan;
  excluye autenticacion, sesion, autorizacion del intento, token de reserva,
  correlaciones tecnicas, auditoria/outbox aleatorios y tiempos efimeros.
  El manifiesto probatorio V1 de baremacion no cumple esta regla porque cubre
  autorizaciones efimeras de reserva y confirmacion: se sustituira por un
  manifiesto material V2. Cada recibo remoto tendra esquema y huella canonica
  propios; la huella del documento nunca sustituye la del recibo de custodia o
  retencion.
- Catalogos materiales: el formato documental y la clasificacion se fijan en
  la intencion mediante snapshots administrables con referencia, version y
  huella. La clave de formato se vincula al MIME canonico, al perfil y politica
  de firma y a las capacidades del conector; la clasificacion se vincula a las
  politicas de custodia y retencion. PDF/PAdES puede ser la configuracion
  inicial, pero no se compila como unica alternativa ni se cambia el significado
  historico de una clave de catalogo ya utilizada.
- Frontera de confianza en proceso: un tipo opaco evita promociones y
  manipulaciones accidentales, pero por si solo no demuestra autoridad si su
  fabrica acepta puertos sustituibles. Identidades, criptografia, catalogos y
  verificadores son parte de la base de computacion confiable y se fijan una
  sola vez en la raiz de composicion homologada. Ningun DTO, handler, modulo o
  conector elegido por una peticion puede suministrarlos o sustituirlos. El
  flujo permanece cerrado hasta que un servicio de aplicacion con dependencias
  privadas controle esas fabricas y existan pruebas que recorran el ensamblado
  real; un conector no confiable se aisla fuera de proceso y se autentica, no se
  carga como codigo con los mismos privilegios.
- Autoridad actual: cada intento, incluida una consulta o recuperacion
  idempotente, obtiene y consume una concesion RBAC/ABAC positiva, exacta y
  vigente. Esta concesion no cambia la intencion ni se compara con la original.
  Una baja, revocacion o cambio de ambito deniega antes de revelar el resultado
  ya confirmado.
- Doble corte documental: antes de leer datos personales, fusionarlos,
  sellarlos o entregarlos a un renderizador se exige una preautorizacion sobre
  expediente, sujeto, plantilla, finalidad y clasificacion. Tras obtener los
  bytes y huellas se exige otra concesion ligada al manifiesto exacto justo
  antes del efecto. Identidad, pertenencia, relaciones y clasificacion proceden
  de fronteras internas vinculadas a la sesion, nunca del DTO del cliente.
- Estados: ausente crea una operacion; mismo indice y otra intencion deniega
  reutilizacion; misma intencion en curso informa sin repetir efectos; una
  operacion confirmada devuelve la version y evidencia transaccional completas.
  La aplicacion coteja el resultado persistido contra la intencion exacta antes
  de aceptarlo.
- Efectos: identificadores, manifiestos y claves tecnicas necesarios para
  repetir o reconciliar se fijan durablemente antes del primer efecto externo.
  Una respuesta ambigua nunca causa borrado compensatorio ni genera una segunda
  identidad. El reconciliador continua desde el ultimo paso probado. La
  confirmacion final recibe evidencia tipada de decision, reserva, plan y cada
  objeto; una referencia textual suelta no acredita el efecto.
- Pruebas de apertura: `COMMIT` mas `DeadlineExceeded`, reinicio del proceso,
  sesion y autorizacion nuevas del mismo actor, dos reintentos concurrentes,
  autorizacion revocada, cambio individual de cada campo de intencion, rotacion
  de clave e inyeccion de evidencia persistida incompleta o contradictoria.
  Version, efecto, auditoria y outbox deben conservar cardinalidad uno. El
  corredor obligatorio atraviesa el caso de uso Go, el adaptador y PostgreSQL
  real; una prueba SQL que reutilice HMAC sinteticos no acredita compatibilidad
  con las preimagenes distintas de reserva y confirmacion.

## DEC-046 — Preparacion y entrega recuperable de carga directa

- Estado: decision adoptada el 15 de julio de 2026; contrato y adaptador
  pendientes. La carga directa a almacenamiento compatible con S3 continua
  como objetivo, pero el flujo actual no se expone como productivo.
- Problema: confirmar carga y manifiesto antes de emitir el recibo evita
  aceptar una sesion no persistida, pero abre otro fallo. Si la emision o la
  respuesta HTTP se pierde despues del `COMMIT`, el reintento encuentra la
  preparacion ya procesada y el usuario no puede recuperar una credencial
  valida para completar la subida.
- Intencion de entrega: el mismo commit de preparacion persiste una intencion
  estable ligada al indice idempotente, carga, vinculo HMAC de la sesion,
  manifiesto, decision, expiracion y huellas. Sus estados cerrados son
  `pendiente`, `emitida`, `consumida` y `expirada`.
- Emision: el emisor registra de forma idempotente el recibo exacto o un
  identificador opaco de entrega de un solo uso. Ni la URL temporal ni una
  credencial de objeto se escriben en auditoria, outbox o logs generales.
- Recuperacion: una repeticion autorizada devuelve una proyeccion de la
  preparacion confirmada y un identificador de recuperacion. Una operacion
  separada, con identidad, sesion y permiso actuales revalidados, recupera el
  mismo recibo aun vigente o sustituye atomicamente el anterior, conservando
  predecesor, causa y expiracion. Nunca reconstruye autoridad desde el DTO.
- Reconciliacion: un outbox o trabajador durable resuelve la ventana entre
  commit, emisor y entrega. Una respuesta ambigua deja el estado recuperable;
  no abandona la reserva, no revoca a ciegas una sesion confirmada y no activa
  dos recibos simultaneos.
- Pruebas de apertura: commit correcto seguido de fallo del emisor, perdida de
  respuesta HTTP y reinicio; reintento desde otra sesion valida; carreras de
  recuperacion; caducidad y sustitucion; revocacion de permiso; y recibo,
  manifiesto o vinculo manipulados. Deben quedar una preparacion, como maximo
  un recibo activo y una cadena completa de auditoria y evidencias.

## DEC-047 — Capacidades TCB privadas y conectores nominales

- Estado: decision transversal adoptada el 15 de julio de 2026; servicios de
  aplicacion, composicion y adaptadores productivos pendientes. Ningun contrato
  nominal de idempotencia o despacho autoriza por si solo un efecto.
- Problema de lenguaje: Go no ofrece paquetes amigos. Una fabrica exportada que
  acepte verificadores sustituibles puede ser invocada tambien por un handler y
  autocertificar una capacidad mediante dobles. Sellar una interfaz con metodos
  privados evita ese abuso, pero tambien impide que un adaptador KMS, registro u
  Oracle/PostgreSQL de otro paquete la implemente y rompe la arquitectura
  hexagonal.
- Puertos publicos: solicitudes canonicas, resultados de conectores y pruebas
  transportables se denominan expresamente `Crudo` o `Nominal`. Pueden tener
  constructores publicos y ser implementados por adaptadores, pero su validacion
  solo acredita forma, vinculo y redaccion; nunca autenticidad, completitud,
  vigencia, consumo ni autoridad.
- Servicio privado: la promocion se realiza dentro de un servicio de aplicacion
  con dependencias privadas de identidad, KMS, registro, catalogos y
  persistencia, fijadas exclusivamente por la raiz de composicion homologada.
  Su constructor devuelve solo el puerto del caso de uso. No devuelve
  verificadores, repositorios, promotores ni capacidades.
- Capacidad efimera: el valor comprobado tiene campos privados, no se serializa,
  no se persiste, no se registra y no aparece en firmas HTTP, CLI, MCP ni en
  eventos. Se crea y consume dentro de la misma llamada del servicio. Un
  handler solo recibe comandos funcionales y resultados minimizados.
- Completitud de llaveros: idempotencia exige un testimonio combinado y
  atestado que inmovilice referencia, revision, cantidad, orden y topologia
  completa de los llaveros de identidades e indices junto con toda la matriz.
  El productor y el verificador de confianza son fronteras distintas. Una
  matriz coherente cuya fuente haya omitido historicos y recalculado sus propias
  huellas sigue siendo nominal y se deniega.
- Consumo documental: verificar o releer una orden no basta. El registro debe
  releer el inicio durable, adopcion V3/V4, secuencia y reclamacion y consumir
  la orden mediante CAS en la misma transaccion que confirma auditoria y
  outbox. Se prohibe el patron leer-verificar-usar con una ventana TOCTOU.
- Composicion: HTTP, CLI, MCP, modulos funcionales y DTO nunca reciben ni
  seleccionan los conectores TCB. La raiz construye una vez los servicios con
  implementaciones homologadas y entrega a cada adaptador de entrada solamente
  su caso de uso de minimo privilegio.
- Guardas de arquitectura: pruebas AST/importaciones fallan si una frontera
  externa importa adaptadores TCB, referencia fabricas o tipos comprobados,
  incluye capacidades en peticiones/respuestas o usa constructores de
  composicion fuera de `bootstrap`, aplicacion o pruebas autorizadas.
- Aislamiento reforzado: si el modelo de amenaza incluye un handler o proceso
  web comprometido, la opacidad de tipos Go no constituye una barrera. KMS,
  registro y trabajador de efectos se ejecutan en proceso interno separado con
  identidad de servicio, mTLS, red restringida y un protocolo nominal
  autenticado.
- Puerta tecnica: no hay GO hasta probar el ensamblado real desde el caso de uso
  hasta KMS y PostgreSQL, llamada unica donde proceda, consumo CAS, cancelacion,
  reinicio, carreras, denegacion por defecto y ausencia de capacidades en todas
  las fronteras externas. Dobles unitarios no acreditan esta puerta.

## DEC-048 — Ambito y formula verificable de idempotencia de baremacion

- Estado: refinamiento de DEC-045 adoptado el 15 de julio de 2026. Los
  contratos nominales pueden implantarse, pero el flujo productivo permanece
  cerrado hasta que el servicio privado, KMS/HSM y persistencia prueben esta
  formula de extremo a extremo.
- Sujeto autoritativo: la clave aportada por el cliente nunca identifica a la
  persona. El servicio resuelve el sujeto desde sesion, expediente y relaciones
  internas. El identificador interno estable se entrega solo de forma efimera a
  las fronteras de identidad y KMS/HSM; no se persiste, registra ni expone. Este
  origina de manera independiente el seudonimo con el dominio `sujeto` y los
  candidatos con el dominio `principal`. DNI, nombre, correo y referencias
  libres del cliente no entran en el indice. El testimonio liga el seudonimo y
  las referencias opacas de proceso, solicitud y baremacion utilizadas para
  resolver la misma identidad.
- Principal estable: por cada generacion vigente del llavero `principal` se
  deriva
  `HMAC(clave_principal, esquema + identificador_interno_estable_canonico)`.
  No se deriva del seudonimo: la rotacion del llavero `sujeto` cambiara ese
  seudonimo, pero no los candidatos de principal ni los indices recuperables.
  Tampoco incluye la clave idempotente del cliente, sesion, autorizacion,
  intento ni expediente; por ello el mismo sujeto conserva principal durante
  reintentos y las rotaciones pueden cotejar historicos sin confundir personas.
- Indice de operacion: por cada combinacion de principal e historico vigente
  del llavero `indice` se deriva
  `HMAC(clave_indice, esquema + despliegue + modulo + accion + principal +
  clave_cliente)`. La clave de cliente solo admite UUIDv4 canonico o material
  binario opaco de alta entropia, se entrega de forma efimera y nunca se guarda
  en claro. Misma persona y clave recuperan la misma operacion; otra persona o
  accion produce indices distintos.
- Formula homologada: el testimonio fija esquema, version, referencia y revision
  de politica de derivacion, instantaneas completas y ordenadas de ambos
  llaveros, matriz cartesiana completa, preimagen contextual comprometida y
  evidencia de atestacion. Una firma valida sobre hashes arbitrarios no prueba
  la formula. La raiz independiente debe rederivar y comparar o verificar una
  atestacion HSM que identifique de forma inequívoca operacion, politica y
  material comprometido.
- Fuentes efimeras: productor y verificador reciben copias de un solo uso del
  minimo material necesario, incluida la identidad interna estable y la clave
  de cliente cuando corresponda, creadas dentro del servicio y destruidas al
  finalizar. Son implementaciones distintas fijadas por composicion; ni HTTP,
  CLI, MCP ni un modulo seleccionan una de ellas. El resultado publico sigue
  siendo nominal y no habilita persistencia ni efectos. La limpieza de memoria
  reduce exposicion accidental, pero no aisla de codigo malicioso con los mismos
  privilegios; ese riesgo exige el proceso separado previsto en DEC-047.
- Separacion de dominios: la comprobacion comprende todos los historicos
  vigentes de `sujeto`, `principal`, `indice`, `motivo`, `manifiesto` e
  `intencion`, no una referencia representativa por dominio. Cada dominio
  conserva de una a ocho generaciones ordenadas. El KMS resuelve alias a
  identidades fisicas y politicas y deniega cualquier reutilizacion cruzada,
  aunque las referencias textuales sean diferentes.
- Casos de aceptacion: mismo sujeto con dos claves de cliente mantiene principal
  y cambia indices; dos sujetos con la misma clave cambian ambos; cruzar
  solicitud, sujeto, accion, politica, topologia, fila, columna o atestacion
  falla cerrado. Rotar el dominio `sujeto` para la misma identidad cambia el
  seudonimo sin cambiar los principales ni los indices historicos. Tambien se
  prueban omisiones autoconsistentes primera, intermedia y ultima, matriz 1x1
  frente a historicos 8x8, alias fisico entre dominios, cancelacion, `typed nil`,
  redaccion y adaptadores desde otro paquete.

## DEC-049 — Identidad atomica y fingerprint estable frente a rotaciones

- Estado: decision adoptada el 15 de julio de 2026 como cierre de DEC-045 y
  DEC-048. El contrato nominal y el futuro servicio privado deben aplicarla
  antes de abrir la idempotencia de baremacion a produccion.
- Problema: seudonimo de sujeto, sello de manifiesto y HMAC de motivo contienen
  version, referencia de clave y MAC. Volver a derivarlos tras una rotacion
  produce sobres distintos aunque la persona, el documento, el motivo y la
  operacion de negocio sean los mismos. Compararlos como parte del fingerprint
  causa conflictos falsos y rompe la recuperacion historica.
- Resolucion atomica: la frontera de identidad recibe conjuntamente referencias
  opacas de proceso, solicitud y baremacion y el seudonimo esperado. Entrega una
  sola vez un ancla interna aleatoria, inmutable y binaria de 256 bits junto con
  referencia, revision, huella y atestacion del snapshot. No acepta referencias
  de una persona y seudonimo de otra.
- Snapshot ligado: su huella se calcula canonicamente sobre esquema, ambito,
  seudonimo esperado, referencia y revision del snapshot y ancla interna. La
  fabrica la recomputa antes de usarla. La raiz independiente recibe una copia
  efimera de la misma resolucion, reconstruye el material y verifica la
  atestacion; una huella o firma meramente declarada no acredita la relacion.
  La frontera se consulta una vez y el lote se clona internamente para productor
  y verificador, evitando TOCTOU y ABA entre dos resoluciones.
- Ancla interna: se genera con CSPRNG en el servicio de identidad, se cifra en
  reposo, se incluye en copias y restauracion y no cambia por rotar llaveros. No
  es DNI, nombre, UUID textual, seudonimo ni HMAC con clave rotatoria. Nunca se
  persiste en el modulo, DTO, contexto, log, auditoria ni evento.
- Fingerprint semantico: compromete el indice idempotente ya verificado por la
  raiz TCB, ambito, versiones, decisiones, referencias y huellas estables de
  negocio, politicas, catalogos, objetos, recibos y resultado esperado. No
  incorpora version, referencia ni valor de sobres criptograficos rotatorios.
  El indice liga sujeto, despliegue, modulo, accion y clave de operacion sin
  publicar un principal o hash global correlacionable.
- Pruebas rotatorias: el seudonimo exacto, HMAC de motivo, sello de manifiesto y
  demas sobres se verifican con sus claves historicas antes de aceptar igualdad.
  El manifiesto queda ligado por su referencia, version y huella material. Para
  un motivo libre de baja entropia se verifica el texto efimero contra el HMAC
  historico almacenado, o se introduce solo dentro del HMAC de intencion; nunca
  se persiste el texto ni un SHA publico susceptible de diccionario.
- Sobre probatorio: cada intento conserva y firma por separado el seudonimo,
  snapshot de identidad, referencias y revisiones de clave, HMAC y evidencias
  exactas que se verificaron. Rotar una clave cambia este sobre y genera otro
  asiento de auditoria, pero no convierte la misma intencion en otra operacion.
- Orden del servicio: revalida identidad, sesion y permiso actuales; resuelve y
  atesta identidad; deriva candidatos de indice; localiza la operacion; verifica
  pruebas actuales e historicas; compara el fingerprint estable; y solo entonces
  recupera el resultado o crea/continua efectos mediante CAS. Ningun dato
  persistido se revela antes de esta secuencia.
- Casos de aceptacion: misma ancla y clave cliente con rotacion independiente de
  `sujeto`, `manifiesto` o `motivo` mantiene indice y fingerprint y crea nueva
  evidencia de intento; cambiar identidad, ambito, contenido del manifiesto o
  motivo cambia o deniega la operacion. Cruces A/B, snapshot manipulado, frontera
  cambiante, firma falsa, historico omitido y callback asincrono fallan cerrados.

## DEC-050 — Retirada por porte del nucleo heredado de Bolsa

- Estado: decision adoptada el 16 de julio de 2026 por el responsable del
  proyecto. Recogida tambien como H-04 en la
  [auditoria de diseno y seguridad](auditoria_diseno_y_seguridad_2026-07-16.md).
- Problema: `internal/candidate` (heredado, en ingles) convive con
  `internal/modules/bolsa` (nuevo, en espanol). Verificado con el grafo de
  imports: su unico consumidor restante es `internal/app/bootstrap`, que solo
  monta la API heredada en modo `fake`. El doble nucleo duplica dominio,
  adaptadores y mantenimiento.
- Decision: retirar el nucleo heredado portando primero al modulo nuevo lo
  que siga haciendo falta. Retirar no es borrar sin mas; la secuencia
  obligatoria es: inventario de capacidades del heredado; analisis de brecha
  contra `internal/modules/bolsa` dejando constancia en este registro de lo
  que se descarta; porte de lo necesario al formato nuevo (espanol,
  hexagonal, fallo cerrado, autorizacion por caso de uso y limites de
  DEC-051) con sus tests; y solo entonces borrado de `internal/candidate`,
  de su cableado en `bootstrap` y de la configuracion residual de la API
  heredada.
- Mientras tanto: el heredado queda en solo-mantenimiento. No se admite
  codigo, tests ni documentacion nuevos que dependan de `internal/candidate`.
- Reversibilidad: alta hasta el borrado final, que conserva el historico Git
  como via de recuperacion.

## DEC-051 — Limite de tamano de los ficheros de codigo

- Estado: decision adoptada el 16 de julio de 2026 por el responsable del
  proyecto, que delego el umbral concreto en la auditoria tecnica. En vigor,
  aplicada por la puerta de calidad y el CI.
- Problema: los ficheros de miles de lineas agotan el contexto de los
  agentes que desarrollan el proyecto e impiden revisarlos con seguridad
  (peores casos en la fecha: `web/static/app.js` con 13.211 lineas e
  `idempotencia_semantica_baremacion.go` con 4.215).
- Decision: objetivo de diseno de 500 lineas por fichero de codigo y tope
  duro de 800 lineas. El tope lo exige
  `scripts/comprobar_tamano_ficheros.sh`, integrado en
  `scripts/verificar_calidad.sh` y en el workflow `ci`. El margen entre
  objetivo y tope existe para no partir por burocracia un fichero
  cohesionado de 501 lineas: hasta ~800 lineas un agente lo lee entero sin
  hipotecar su contexto; por encima debe trocearse (en Go, dividir en varios
  ficheros del mismo paquete conserva API y comportamiento).
- Linea base: `scripts/tamano_ficheros_base.txt` congela unicamente los
  ficheros ya versionados que superaban el tope: no pueden crecer y la linea
  base solo puede menguar. Los ficheros sin versionar no quedan exceptuados;
  deben trocearse antes de incorporarse.
- Reversibilidad: ajustar el umbral es un cambio de una linea mas la
  regeneracion de la linea base, consensuado en este registro.

## DEC-052 — Frontend en modulos ES nativos con estilos por tokens

- Estado: decision adoptada el 16 de julio de 2026 por el responsable del
  proyecto. Ejecuta la remediacion H-03 de la
  [auditoria de diseno y seguridad](auditoria_diseno_y_seguridad_2026-07-16.md).
- Problema: `web/static/app.js` concentra 13.211 lineas sin modulos ni build:
  ningun agente puede revisarlo entero. Ademas contiene ~300 asignaciones de
  estilo inline desde JS, 67 de ellas con colores literales que se saltan el
  sistema de tokens CSS y romperian cualquier cambio de tema o estilo.
- Decision: partir el frontend en modulos ES nativos
  (`<script type="module">`) por dominio funcional: bolsa, dietas, cronos,
  personal, workspace y componentes comunes. Sin framework, sin bundler, sin
  npm y sin build step: el navegador resuelve los `import` directamente, en
  coherencia con el diseno autocontenido y sin dependencias externas del
  portal. Cada modulo resultante cumple los limites de DEC-051.
- Estilos y temas: colores, tipografias y espaciados viven unicamente como
  tokens CSS (variables `--*` en `styles.css`, hoy 49). Un tema es una
  redefinicion de tokens bajo un atributo del documento (por ejemplo
  `:root[data-theme="oscuro"]`), y cambiar de tema es cambiar ese atributo
  desde JS: una linea. El JS manipula clases semanticas, nunca estilos
  inline. Prohibido introducir colores literales o `.style` nuevos en JS;
  los existentes se eliminan al migrar cada modulo.
- Criterio de terminado por modulo migrado: funcionalidad intacta, sin
  variables globales nuevas, sin colores literales, fichero dentro del tope
  de DEC-051 y tema conmutable sin retocar el modulo.
- Reversibilidad: media. Los modulos pueden reagruparse si hiciera falta,
  pero no se contempla volver al monolito.

## DEC-053 — API primero con clientes web y de escritorio equivalentes

- Estado: decision adoptada el 16 de julio de 2026 por el responsable del
  proyecto. Condiciona la construccion de la Ola 2 (endpoints finos).
- Necesidad: los tecnicos de RRHH y administracion acceden a la parte
  interna y delicada del portal. El responsable contempla darles una
  aplicacion de escritorio con seguridad reforzada en lugar de acceso web,
  segun las puertas separadas ya descritas en
  [acceso interno de tecnicos](../estudio_requisitos/acceso_interno_tecnicos_administracion.md).
- Decision: toda funcionalidad nace como endpoint API JSON versionado bajo
  `/api/vec`; la web estatica es un cliente mas, sin privilegios de acceso
  distintos de los de cualquier otro cliente autorizado. Prohibido
  incorporar logica de negocio, calculos normativos o datos de negocio
  incrustados en los clientes: lo que hoy hace `app.js` con datos sinteticos
  locales se considera deuda a extinguir con la Ola 2, no un patron a
  repetir.
- Sesion y autenticacion: sin cookies de sesion en ninguna superficie; la
  autenticacion viaja por cabeceras o token en cada peticion, como ya ocurre.
  La puerta de tecnicos podra montarse como superficie separada con
  autenticacion reforzada (mTLS de dispositivo y empleado, Kerberos/AD,
  allowlist de puestos de administracion via `VEC_HTTP_ALLOWED_CIDRS`), y
  reutiliza sin cambios la autorizacion por caso de uso del servidor: un
  cliente distinto jamas implica una autorizacion distinta.
- Contrato: cada modulo documenta sus endpoints (ruta, metodo, envelope,
  errores y version) en su documentacion de contrato, de forma que un
  cliente de escritorio pueda construirse contra el contrato sin leer el
  codigo del servidor.
- Reversibilidad: alta para el cliente (web y escritorio son
  intercambiables por diseno); baja para el principio API primero, que es
  precisamente la garantia de esa intercambiabilidad.

## DEC-054 — Abandono acotado de reservas antes del COMMIT

- Estado: decision aplicada al corte interno de baremacion; la persistencia
  durable del resultado indeterminado y el reconciliador productivo siguen
  pendientes.
- Limite seguro: una reserva solo puede abandonarse antes de invocar la
  frontera que puede enviar `COMMIT`. Desde que se invoca
  `ConfirmarCambio`, cualquier error, cancelacion o perdida de respuesta
  puede ocultar un commit aplicado: queda prohibido abandonar, compensar o
  repetir el efecto a ciegas.
- Autoridad: el abandono obtiene una concesion RBAC/ABAC nueva, exacta y
  vigente, distinta de las usadas para reservar o confirmar.
- Reintento: ante una respuesta ambigua del abandono se permite un unico
  reintento sincronico con la misma concesion, token, recurso y solicitud
  exacta, dentro de un plazo independiente y nunca mas alla de la vigencia
  conocida de la reserva.
- Resultado no acreditado: agotado el plazo o el reintento sin confirmacion
  autoritativa, no se presume abandono. Se conserva una clasificacion fija y
  expurgada; la evidencia durable y su reconciliacion forman parte del cierre
  productivo pendiente. Tras invocar `ConfirmarCambio`, todo desenlace no
  autoritativo debe clasificarse como transaccionalmente indeterminado y
  entregarse a reconciliacion, sin ampliar autoridad ni intentos.

## DEC-055 — Sobre probatorio nominal V2 ligado a la operacion

- Estado: contrato nominal y normalizacion fail-closed implantados el 16 de
  julio de 2026; flujo productivo, persistencia y DDL V2 siguen en NO-GO.
- Separacion: V1 permanece experimental e inalterado. V2 usa la finalidad
  criptografica independiente
  `sobre_probatorio_confirmacion_baremacion_v2`; su representacion canonica
  cubre la autorizacion exacta y, dentro del dominio V2, situa referencia opaca
  e indice HMAC antes del token, agregado, manifiesto, trazabilidad y tiempo del
  intento.
- Alcance nominal: `IntentoNominalConfirmacionBaremacionV2` y
  `ResultadoNominalConfirmacionBaremacionV2` solo validan forma. El eco exacto
  del identificador no demuestra persistencia, autenticidad ni atomicidad de
  version, auditoria y outbox; esa garantia queda reservada al futuro resultado
  canonico autenticado y al corredor Go/PostgreSQL real.
- Fallo cerrado: un error tipado solo se conserva, mediante copia expurgada, si
  es unico, valido y su identificador coincide exactamente con el esperado.
  Ante error generico, identificador ajeno, ramas multiples, `typed nil`, ciclo
  o desenvoltura hostil se crea un indeterminado con el identificador esperado.
  Nunca se conserva la causa tecnica, se abandona la reserva ni se concede un
  reintento.
- Vectores: la prueba congela por separado SHA-256 del canonico V1 y del sobre
  V2 y verifica que cambiar referencia, indice o efecto cambia el material,
  mientras el propio HMAC queda fuera de su preimagen para evitar circularidad.
- Siguiente puerta: definir `RepositorioOperacionesBaremacion`, preparacion
  durable, capacidad AEAD, resultado canonico, consumo de autorizacion y
  reconciliacion. Hasta entonces ningun adaptador productivo puede presentar
  estos tipos nominales como autoridad o prueba de `COMMIT`.

## DEC-056 — Autenticidad y conservacion del manifiesto probatorio de baremacion

- Estado: decision adoptada el 16 de julio de 2026 para el corte previo a
  produccion. El sistema continua limitado a datos sinteticos y no existe
  historia V2 productiva que migrar; esta excepcion permite completar y
  congelar ahora el canonico de version 2 sin reinterpretar datos reales.
- Canonico congelado: la representacion del manifiesto incluye esquema,
  finalidad funcional, version, identidades y referencias, version base,
  instante UTC, cardinalidad explicita y secuencia de autorizaciones y
  evidencias, huella del contenido y una envoltura criptografica exclusiva.
  Los conteos evitan que dos particiones de colecciones compartan una misma
  secuencia de campos. Cualquier cambio futuro de estos bytes exige una nueva
  version de esquema y vectores de migracion; no se modificara V2 en silencio.
- Contrato productor-verificador: el productor declara una
  `FinalidadSelloBaremacion` cerrada y el conector calcula
  `HMAC(K_finalidad, finalidad || 0x00 || representacion_canonica)`. El
  verificador reconstruye exactamente el mismo material. La finalidad
  selecciona un dominio y llavero historico; una carga sellada para reserva,
  confirmacion, sobre nominal o manifiesto no puede reutilizarse en otro.
- Historia probatoria: cada decision incorporada conserva el manifiesto
  completo, inmutable y enlazado uno a uno con baremacion, decision y numero de
  version. Antes de anexar una nueva decision y antes de toda lectura o
  recuperacion se reconstruyen y verifican todos los manifiestos que sostienen
  la version solicitada. Una clave desconocida, retirada, revocada o no
  disponible falla cerrada y no produce efectos.
- Rotacion: el sello conserva una referencia de clave. El conector productivo
  debera resolver la clave activa al firmar y las claves historicas admitidas
  al verificar; retirar una clave con historia viva requiere una operacion
  administrativa aprobada y evidencia de resellado o cierre de retencion, no
  una eliminacion silenciosa.
- Persistencia: el adaptador PostgreSQL y la migracion `000005` implantan el
  almacen append-only, la reconstruccion exacta del archivo, la
  reverificacion y la atomicidad con version, auditoria y outbox. El corredor
  oficial valida esa puerta de archivo desde una base PostgreSQL 18.4 vacia,
  tambien tras reinicio, bajo concurrencia y con un `LOGIN` real sometido a
  RLS. No acredita por si solo la recuperacion semantica completa de una
  operacion cuya respuesta a `COMMIT` se pierda. La funcionalidad continua en
  **NO-GO productivo** hasta disponer del conector KMS/HSM auditado y cerrar la
  identificacion y reconciliacion nominal exigidas por DEC-045.

## DEC-057 — Prevalidacion autorizada del archivo probatorio de baremacion

- Estado: contrato de puertos, aplicacion, memoria y PostgreSQL implantado y
  validado el 16 de julio de 2026. La version 2 no alcanzo produccion ni
  contiene datos reales que migrar; se conserva como formato historico
  congelado y la nueva escritura se eleva de forma completa a V3.
- Autoridad minima: antes de confirmar una decision, el servicio solicita una
  concesion nueva para la accion exacta
  `bolsa.baremacion.archivo.prevalidar`, sobre recurso `baremacion` y solo el
  campo `archivo_probatorio`. La concesion debe estar vigente inmediatamente
  antes de construir el manifiesto. Se rechazan campos ampliados, accion o
  recurso distintos y cualquier divergencia de baremacion, principal, sujeto,
  perfil activo, finalidad, correlacion, autenticacion o sesion respecto de la
  confirmacion.
- Separacion de deberes: prevalidar y confirmar requieren referencias de
  autorizacion distintas. Ambas concesiones completas forman parte del
  manifiesto V3 y de la representacion canonica HMAC de confirmacion; no basta
  con guardar sus identificadores. En un alta sin archivo la prevalidacion debe
  estar materialmente ausente, no meramente caducada o invalida.
- Dominios criptograficos: la confirmacion corriente usa
  `confirmacion_baremacion_v2` y el efecto
  `efecto-confirmacion-baremacion-v2`. El sobre nominal y el manifiesto pasan a
  `sobre_probatorio_confirmacion_baremacion_v3` y
  `manifiesto_probatorio_baremacion_v3`. El efecto previo conocido antes de
  acceder al almacen se deriva como
  `SHA-256(canonico("efecto-prevalidacion-archivo-probatorio-baremacion-v3",
  huella_confirmacion_v2))`; no se confunde con el recibo posterior obtenido
  tras leer y validar el archivo. El puerto de sellado rechaza expresamente
  finalidades V1/V2 retiradas; solo el puerto de verificacion puede aceptarlas
  para comprobar artefactos historicos ya existentes.
- Protocolo durable e idempotencia: la autorizacion de prevalidacion se consume
  en una transaccion corta y queda ligada al efecto exacto de confirmacion y a
  la huella del archivo devuelto. Esa transaccion se cierra antes de consultar
  el KMS/HSM, por lo que nunca se retienen bloqueos PostgreSQL durante E/S
  criptografica. Si la verificacion falla, no se consume la autorizacion de
  confirmacion ni se modifica version, decision, auditoria u outbox. Mientras
  la capacidad temporal de reserva permanezca exclusivamente en memoria, el
  reintento exacto reutiliza de forma idempotente el consumo previo y puede
  completar la confirmacion; reutilizar cualquiera de las dos concesiones para
  otro efecto falla cerrado. La capacidad no se serializa ni se incluye en
  auditoria, outbox o estado general. Si el proceso termina antes de confirmar,
  este corte no puede reanudar ese intento: debe fallar cerrado y esperar la
  capacidad recuperable y el reconciliador nominal previstos desde `000006`.
  Antes del consumo, PostgreSQL coteja tambien el vinculo completo de
  autenticacion de la reserva con el de la nueva concesion. Una autorizacion
  valida del mismo principal pero procedente de otra sesion o revision de
  `ContextoActor` devuelve `reserva_invalida` y no deja uso ni prevalidacion;
  el corredor incluye esa regresion funcional adversaria.
  El adaptador en memoria conserva una aplicacion atomica como referencia no
  productiva porque no realiza E/S criptografica externa dentro del
  repositorio.
- Puerta productiva: los vectores V2 historicos permanecen inalterados y los
  nuevos vectores V3 congelan los bytes actuales. PostgreSQL 18.4 supera la
  puerta durable de archivo, la migracion reversible y el corredor
  Go/PostgreSQL; no supera todavia la recuperacion nominal completa. El
  **NO-GO productivo** permanece hasta que el almacenamiento probatorio y sus
  claves procedan de conectores productivos auditados y se complete la
  reconciliacion nominal de DEC-045; ninguno de esos requisitos se sustituye
  por los sellos sinteticos del corredor.

## DEC-058 — Perfil temporal canonico unico para decisiones de autorizacion

- Estado: defecto transversal detectado y corregido el 16 de julio de 2026;
  los corredores PostgreSQL de autorizacion, ejecucion documental V4 y Bolsa
  V3 forman parte de su puerta de regresion. El sistema sigue sin datos
  productivos y en **NO-GO**, por lo que no existe historia real que transformar.
- Hallazgo: `encoding/json` aplica RFC3339Nano a `time.Time` y elimina ceros
  fraccionales finales. La representacion criptografica de la decision y la
  reconstruccion SQL exigen siempre UTC con seis digitos de microsegundo. Como
  los seis instantes ligados comparten la misma fraccion, aproximadamente una
  de cada diez decisiones podia quedar registrada con un documento valido pero
  no coincidir despues con su canon probatorio.
- Decision: `RepresentacionCanonicaDecisionAutorizacionReforzadaV1` es la unica
  fuente de la proyeccion comprometida por la huella. El adaptador PostgreSQL
  deriva de ella su documento de 31 claves, sustituyendo exclusivamente los
  dos manifiestos canonicos por sus listas de referencias y mapas de huellas.
  No vuelve a serializar `time.Time`, no inventa un instante de verificacion y
  conserva la posibilidad de registrar decisiones validas con obligaciones;
  registrar no equivale a acreditar que esas obligaciones se hayan cumplido.
- Cobertura: se prueban segundo exacto, limites de microsegundo, de cero a seis
  ceros finales, los dos instantes de la decision y los cuatro del vinculo de
  autenticacion, obligaciones presentes y rechazo submicrosegundo. Bolsa fuerza
  deliberadamente un cero final y ejecucion documental usa segundo exacto; ya
  no se permite elegir una fraccion que oculte la divergencia.
- Historia y despliegue: nunca se reescribe una decision append-only. Antes del
  GO se publicara una migracion de endurecimiento que conserve la forma legacy
  de 30 claves, exija `.ddddddZ` en los seis instantes de toda decision nueva de
  31 claves y aborte en el preflight si detecta historia actual no canonica.
  Una base exclusivamente sintetica se reconstruye; si apareciera historia real
  se versionaria el lector en lugar de mutar documentos probatorios.

## DEC-059 — Capacidades opacas, nominales y de un solo efecto

- Estado: decision implantada el 16 de julio de 2026 para las reservas de
  generacion documental, emision de codigo de cotejo, alta y devolucion de
  cobro y para el arrendamiento del flujo de firma de baremaciones. No concede
  por si misma un GO productivo a los adaptadores en memoria ni a pagos.
- Problema: representar autoridad temporal mediante cadenas o slices permite
  copiarla, persistirla o recuperarla accidentalmente mediante serializadores,
  registros y reflexion segura, incluso aunque el campo no sea exportado. Un
  identificador de reserva no debe convertirse en una credencial reutilizable.
- Decision de tipo: cada capacidad es nominal, tiene valor cero invalido y
  contiene un unico cierre privado e inmutable que captura 256 bits obtenidos
  del CSPRNG. No existe metodo para revelar, reconstruir o aceptar el material.
  En documentos, cotejo y pagos el cierre nace ligado al dominio criptografico
  exacto y solo calcula o compara una huella SHA-256 canonica. En firma solo
  calcula o verifica el HMAC de finalidad fija que le solicita el adaptador.
  Las comparaciones de huellas validas se realizan en tiempo constante.
- Superficies genericas: JSON, texto, binario, gob y XML fallan de forma
  cerrada, tanto al codificar como al decodificar. `fmt` y `slog` producen
  exclusivamente marcadores redactados. La API de reflexion segura solo ve un
  campo funcion privado, que no puede convertir, sustituir ni invocar. Esta
  defensa reduce fugas accidentales y abuso dentro del proceso; no pretende
  resistir ejecucion arbitraria de codigo, `unsafe`, un volcado de memoria o un
  host comprometido, que corresponden a la frontera de ejecucion y secretos.
- Persistencia y consumo: los adaptadores en memoria de documentos y cotejo
  guardan unicamente la huella; el de firma guarda unicamente HMAC y captura su
  clave efimera en otro cierre no reflectible. Confirmar o abandonar consume y
  elimina el selector. Una liberacion repetida del mismo arrendamiento ya
  ausente es idempotente; un token nulo, ajeno u obsoleto frente a un
  arrendamiento vigente falla sin mutacion. El contrato de pagos usa los tipos
  nominales y su doble de prueba consume la huella, pero aun no existe un
  adaptador productivo de cobros que acredite esta semantica.
- Frontera productiva: el repositorio de firma real debe delegar el HMAC en un
  KMS/HSM, con clave versionada y rotacion; los almacenes durables deben aplicar
  transaccion, caducidad autoritativa, unicidad y borrado de la huella al
  consumir. Los relojes de los adaptadores en memoria no acreditan tiempo
  autoritativo de base de datos. El metodo actual de firma recibe la clave HMAC
  como bytes y solo acredita el adaptador en memoria: antes de construir el
  adaptador productivo se definira un puerto de sellado KMS que opere con una
  clave no exportable, o se sellara en KMS el compromiso SHA-256 del token. No
  se trasladara una clave HSM al proceso para reutilizar este doble de prueba.
- Alcance pendiente: esta decision no transforma todavia
  `TokenReservaCargaDocumental` ni `TokenReservaBaremacion`, cuyos contratos
  historicos permiten revelar material al adaptador. Deben migrarse en un corte
  posterior, con sus repositorios y corredores, antes de afirmar que todas las
  capacidades del portal son opacas.

## DEC-060 — Cierre de capacidades historicas de carga y baremacion

- Estado: decision implantada y probada el 16 de julio de 2026. Cierra el
  alcance pendiente de DEC-059 para los tipos publicos y los adaptadores en
  memoria, pero no concede un GO productivo: carga documental sigue sin
  repositorio durable y baremacion necesita KMS/HSM y reconciliacion nominal.
- Carga documental: cada reserva genera 256 bits con el CSPRNG y los captura en
  un cierre privado ligado a `vec:token-reserva-carga-documental:v1`. El tipo no
  acepta ni devuelve una cadena; solo calcula o coteja su huella SHA-256. El
  repositorio en memoria conserva esa huella, la elimina al confirmar,
  abandonar o expirar y usa su reloj interno para la caducidad. JSON, texto,
  binario, gob y XML fallan cerrados; `fmt`, `slog` y reflexion segura no
  recuperan el material.
- Baremacion historica: se conserva exactamente la formula durable
  `SHA-256(Base64URL)` y la disposicion canonica V2/V3 para no romper reservas,
  migraciones ni vectores congelados. El valor importado se copia a un cierre
  privado y el tipo deja de publicar `Revelar`; solo ofrece huella y comparacion
  constante. La escritura del canonico ocurre dentro del paquete de puertos,
  sin devolver la cadena al llamador.
- Limite explicito de baremacion: los canonicos historicos incluyen por contrato
  los bytes Base64URL y `CargaProtegida` permite revelarlos al sellador. Por
  tanto se ha cerrado la exposicion del token como capacidad publica, pero no se
  afirma opacidad absoluta de su preimagen dentro de la frontera criptografica.
  El conector productivo debe limitar esa carga al sellador KMS/HSM y una futura
  version podra comprometer el token sin incluirlo, con migracion y vectores
  propios; V2/V3 no se reinterpretan.
- Linealizacion y cancelacion: confirmar frente a abandonar tiene un unico
  ganador. Los mutadores revalidan el contexto despues de adquirir el bloqueo,
  de modo que una cancelacion mientras se espera no crea, consume, abandona ni
  transforma estado. La respuesta y todos los asientos de una reserva se
  construyen y validan antes del punto de publicacion logica.
- Evidencia: permanecen congelados los cinco vectores V2/V3; se prueban
  serializacion hostil, reflexion, dominios cruzados, repeticion, caducidad,
  carreras y cancelacion determinista. El corredor PostgreSQL 18.4 V3 conserva
  los vectores, los casos 4096+4096, reinicios, RLS/ACL y concurrencia
  `SERIALIZABLE`. Esa puerta acredita compatibilidad, no un adaptador durable de
  carga ni la custodia productiva de claves.

## DEC-061 — Autoridad central y proyecciones de categorias profesionales para Bolsa y Personal

- Estado: implantada el 16 de julio de 2026 para las consultas de Bolsa y
  Personal y su demostracion local. No declara oficial el contenido, no
  sustituye la aprobacion de RRHH ni concede GO productivo al catalogo.
- Hallazgo: coexistian 58 categorias hardcodeadas en un workspace heredado, un
  snapshot mutable de Personal y solo dos entradas distintas dentro del JSON
  publico. Las dos ultimas usaban guiones bajos, mientras el maestro historico
  usaba claves kebab-case. La interfaz se limitaba a representar el catalogo de
  dos elementos que recibia.
- Decision: `categorias-profesionales` es un `CatalogoConfigurable` del nucleo,
  con ID, version y huella exactas. El bootstrap carga una sola instantanea
  inmutable y la comparte con Bolsa y Personal mediante puertos y adaptadores;
  ninguno puede resolver implicitamente «la ultima version» ni volver a
  embeber el catalogo. Las convocatorias conservan las claves y una referencia
  inmutable al ID, version y huella, incluida en su propia huella publica. Ruta,
  ID, version y huella esperada son configurables; el bootstrap coteja las
  referencias antes de montar las rutas y una seleccion incompatible impide el
  arranque.
- Proyecciones: la faceta de convocatorias contiene solo categorias con
  resultados y su conteo, calculado aplicando los demas filtros. El directorio
  `/api/publico/bolsa/categorias` devuelve todas las entradas vigentes y
  publicables, con area configurada y conteos derivados. Los `GET` heredados de
  Personal conservan rutas, paginacion, filtros y alias de campos compatibles,
  pero se proyectan desde la misma referencia ID/version/huella. Ninguna de
  estas respuestas expone `source_path`, rutas locales, alias internos,
  actores, motivos ni aprobaciones.
- Datos iniciales: el paquete demo recupera 58 categorias del corpus historico
  OPES, 5 de Administracion general y 53 de Administracion especial. Su fuente
  candidata queda identificada por SHA-256 y el paquete se marca, en datos y
  en pantalla, como
  pendiente de validacion y aprobacion formal por RRHH. No se copiaron bases de
  candidatos ni otros ficheros con datos personales.
- Frontera de despliegue: el adaptador de fichero es estricto, acotado,
  inmutable y de solo lectura. No importa ni siembra el gobierno al arrancar.
  Produccion debe sustituirlo por el repositorio publicado PostgreSQL u Oracle,
  manteniendo el mismo puerto y publicando cada version con autenticacion alta,
  doble control, auditoria y recibo.
- Compatibilidad y legado: `POST`, `PUT` y `DELETE` de categorias en Personal
  ya no mutan un segundo almacen; tras comprobar el permiso responden `409`
  con `catalogo_gobernado_requiere_borrador`, sin asiento de exito. Las
  categorias presentes en snapshots historicos de Personal se validan mediante
  una proyeccion tipada y su subarbol JSON se conserva opaco, incluidas
  extensiones desconocidas, al persistir RPT. No se cargan como autoridad
  consultable. La lista hardcodeada y sus alias se retiran del workspace.
- Administracion pendiente: una modificacion real requerira un caso de uso
  distinto, con version base, borrador, motivo y fuente, validaciones, doble
  aprobacion por actores diferentes, firma/publicacion y recibo auditable. No se
  reinterpreta el `409` como publicacion ni se afirma que ese flujo este
  implementado o productivo.

## DEC-062 — Versiones gobernadas y reproduccion exacta de convocatorias de Bolsa

- Estado: dominio implantado y endurecido mediante revision adversaria el 16 de
  julio de 2026. El contrato durable, PostgreSQL y la composicion interna siguen
  en construccion; ninguna ruta administrativa de este corte queda habilitada
  en produccion y se mantiene el **NO-GO**.
- Separacion conceptual: la version conserva por separado contenido semantico,
  gobierno administrativo y fase procedimental. Borrador, publicada,
  sustituida y retirada son estados de gobierno. Inscripcion, subsanacion,
  alegaciones o cierre proceden exclusivamente de una instancia del flujo
  exacto; cerrar un plazo no equivale a retirar una publicacion.
- Reproduccion: cada version fija ID, numero de secuencia, predecesora inmediata,
  expediente, catalogos, calendario, reglas de baremacion, flujo del proceso,
  flujo de solicitud y documentos oficiales por identidad, version y huella.
  La cadena reserva ademas una unica identidad de instancia del flujo de
  proceso, incluida tanto en la huella semantica como en la huella de estado.
  Una solicitud futura se ligara a esta referencia y huella, nunca a «las reglas
  actuales» ni a «la ultima version».
- Documentos: cada publicacion documental exige correspondencia uno a uno con
  documento logico y version, representacion, SHA-256 de contenido, firma
  validada, recibo de custodia y URL publica no reutilizada. La convocatoria no
  guarda los bytes ni sustituye al almacen documental; conserva referencias
  opacas que el verificador productivo debe releer y atestar.
- Gobierno y doble control: solo se editan borradores mediante CAS. Aprobacion y
  comprobacion de dependencias se ligan a referencia, revision, huella de
  contenido y huella completa del estado para impedir la repeticion A-B-A. El
  creador, ultimo editor, aprobador y publicador quedan separados; aprobador y
  ejecutor de retirada tambien. Una correccion crea una sucesora y publicar la
  nueva y sustituir la anterior es un unico efecto atomico.
- Proyeccion publica: solo una version en estado publicada puede proyectarse y
  requiere la instancia reservada del tipo `convocatoria_bolsa`, ligada al ID
  estable, y la definicion publicada de version y huella exactas. El estado debe
  pertenecer a esa definicion y la instancia inicial no puede preceder a la
  publicacion. Borradores, sustituidas y retiradas no atraviesan esa frontera.
  El lector exterior revalidara ademas el puntero activo actual, usara una
  proyeccion minimizada y un pool propio; no leera agregados aportados ni
  instantaneas antiguas desde cache.
- Canonico y limites: el agregado y su contenido ofrecen representaciones
  canonicas reproducibles y versionadas mediante los esquemas de contenido v2 y
  estado v1, con techo de 8 MiB y vectores golden. Los textos deben ser UTF-8,
  NFC y canonicos; se rechazan controles, URLs y colecciones desproporcionadas
  antes de interpretarlas, copiarlas u ordenarlas. Las fases heredadas quedan
  alineadas con las claves gobernadas en minusculas. El maximo de categorias
  permite el catalogo inicial de 58 entradas y futuras ampliaciones razonables
  sin volver a compilar.
- Persistencia: se crea un esquema propio `vec_bolsa`; no se amplian tablas,
  roles ni fachadas de `vec_bolsa_baremacion`. Se reutilizan sus patrones de
  transaccion serializable, RLS, privilegio minimo, append-only, auditoria y
  outbox, no su autoridad ni su estado. Como publicar, sustituir y retirar no
  incrementan la revision del borrador, PostgreSQL mantendra ademas un
  `numero_estado` monotono y aplicara CAS sobre referencia, revision, numero y
  huella completa.
- Frontera interna: toda gestion exige superficie corporativa o administracion
  privilegiada coherente, garantia alta, Kerberos y certificado acreditados por
  el vinculo de autenticacion, decision PDP exacta, campos positivos exactos y
  ninguna obligacion ignorada. La base releera y consumira la decision y su
  atestacion PDP dentro de la misma transaccion. El navegador nunca construye
  actor, accion, recurso, evidencia, aprobacion, verificacion ni instante.
- Puerta productiva: permanecen pendientes el revalidador productivo de
  autenticacion, sellador HMAC de motivos, registros atestados de aprobacion y
  dependencias, migraciones/roles/RLS, concurrencia y recuperacion reales y
  listener interno separado. Hasta cerrar y probar todos esos puntos, solo se
  sirven datos publicos explicitamente marcados como demostracion.
