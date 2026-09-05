# VEC Diputación de Granada

Copyright (c) 2026 Alberto Avidad (avidad@dipgra.es), para la Diputacion
Provincial de Granada. Publicado bajo la
[Licencia Publica de la Union Europea v1.2 (EUPL-1.2)](LICENSE).

**Estado funcional: 5 de septiembre de 2026. Base publicada del corte actual: `4034157`.**

Cierre de bandeja y análisis: `b2effba`. El desarrollo y las bases sintéticas
se han trasladado al equipo local del operador. La
[guía de arranque vigente](GUIA_RECORRIDO_ALBERTO.md) permite abrir expedientes
en la bandeja principal y registrar el análisis de una solicitud existente,
con recibo conservado tras reiniciar. El servidor remoto queda detenido y conservado.

**Corte 3 incluido en esta entrega; recuperación demostrada:** el navegador registró
una respuesta declarada por RRHH (`201`), con referencia y huella del correo,
actor y justificante persistentes. Tras el parche y segundo reinicio de
aplicación y PostgreSQL, selección, comunicación y respuesta se recuperaron
con `200/200/200`, conservando el mismo justificante, recibo y fecha.
Los rechazos intermitentes por comparar como texto fechas equivalentes están
corregidos mediante comparación de instantes; el diagnóstico se retiró.
También se comprobó un conflicto `409` desde el navegador, sin duplicado.

**Corte actual: solicitud de resolución, no aceptación confirmada.** El cuarto
formulario reutiliza las referencias originales y el justificante RRHH.
Navegador real: `200/200/200/409`; muestra «Pendiente de validar respuesta y
plazo por RRHH. No se ha confirmado la aceptación.», sin recibo terminal ni
duplicados. Conserva el intento para reintento manual con la misma clave.

VEC, Ventanilla Electrónica del Empleado Público, es un portal modular en Go
para la gestión de Recursos Humanos de la Diputación de Granada. Contratación
temporal coordina la petición de personal y su tramitación; Bolsa conserva
las capacidades de convocatorias, baremación, orden y llamamientos. Personal,
Nóminas, Cronos y Dietas son ámbitos diferenciados, no una aplicación ya
terminada por aparecer en el menú.

**Se pueden recorrer cinco pasos completos de Contratación temporal y una
parte del sexto, con datos sintéticos, autorización, PostgreSQL y recibos
persistentes.** El aviso del llamamiento es local: no acredita envío de correo
corporativo ni entrega a una persona.

Este repositorio no está autorizado para producción ni para tratar datos
reales. Publicar código, superar pruebas o mostrar una pantalla no certifica
cumplimiento normativo ni sustituye la aceptación de Recursos Humanos.

## Empiece por su perfil

| Perfil | Manual | Para qué sirve |
|---|---|---|
| Usuario del portal | [Manual de usuario](docs/manual_usuario/manual_portal_bolsas.md) | Acceso, navegación, opciones reales frente a DEMO, recibos, mensajes y ayuda. |
| Recursos Humanos | [Manual de RRHH](docs/manual_rrhh/README.md) | Tramitación, responsabilidades, resultados esperados y límites de cada paso. |
| Programación | [Manual del programador](docs/manual_programador/README.md) | Composición, código reutilizable, contratos, desarrollo y comprobaciones por hito. |
| Sistemas | [Manual de Sistemas](docs/manual_sistemas/README.md) | Preparación del entorno, configuración, certificados, persistencia y operación. |

La [Guía de recorrido de Alberto](GUIA_RECORRIDO_ALBERTO.md) es la referencia
canónica para los **comandos exactos de arranque**, el acceso desde el
navegador, los datos sintéticos conservados y la repetición después de
reiniciar. No hay una segunda receta de instalación en este README.

## Qué funciona de extremo a extremo

El recorrido acreditado utiliza navegador con certificado de cliente,
servicios reales de aplicación, autorización de servidor y PostgreSQL.
No utiliza el adaptador DEMO para afirmar un guardado.

| Paso | Recorrido disponible | Límite |
|---|---|---|
| 1. Solicitud | Alta desde formulario y primer recibo del expediente. | Datos y catálogos de desarrollo. |
| 2. Análisis | Registro del análisis por RRHH y nueva versión del expediente. | El ejemplo recorrido utiliza Sustitución; el formulario ofrece cinco modalidades. |
| 3. Bolsa | Propuesta y decisión de cobertura por **Bolsa vigente**. | No equivale a gestionar de principio a fin una convocatoria de Bolsa. |
| 4. Asignación | Registro de unidad y persona responsable referenciada. | Destino sintético configurado. |
| 5. Informe jurídico y Fiscalización | Documento de desarrollo y resultado favorable, favorable con observaciones o desfavorable; este último registra devolución a la unidad. | El documento no tiene firma ni validez jurídica. Fiscalización corresponde al perfil de Intervención. |
| 6. Llamamiento, parcial | Selección, aviso local y declaración RRHH persistentes y recuperables. Cuarta operación: solicitar resolución; recorrido real `200/200/200/409`, pendiente de validación. | No confirma aceptación ni duplica registros. Faltan validación competente de respuesta/plazo, permiso nominal y activación; tampoco acredita envío o entrega corporativos. |
| 7. Nombramiento | Pendiente como recorrido completo. | No se declaran terminados sus seis documentos, incluida la Diligencia. |
| 8. Incorporación y seguimiento | Pendiente como recorrido completo. | No se acredita incorporación, integración con GINPIX ni cierre del seguimiento. |

La métrica es **cinco pasos completos más un tramo del sexto**, no un
porcentaje global ni un recuento de pantallas, contratos o pruebas.
La bandeja y el detalle ya están conectados en `b2effba`: se demostraron 50
solicitudes conservadas y un análisis desde una de sus filas, sin otra alta.
Esto mejora la continuidad del trabajo; no cierra por sí solo otro paso del flujo.
El registro de respuesta conserva una respuesta, un asiento y un evento;
no cambia Bolsa, no avanza el expediente y mantiene comunicación versión `2`.
El `.eml` sintético se lee y resume con SHA256 en el navegador: no se sube ni
se custodia. El aviso sigue siendo local, no correo corporativo entregado.
Bolsa Go y SQL AD3 `000015` / Bolsa `000004` están preparados para aceptación
RRHH, no activados ni instalados permanentemente en BD. La dinámica SQL aislada
comprobó solo almacenamiento con un doble de autorización estrictamente privado
y transaccional, sin datos ni migraciones persistidos; no acredita
criptografía ni aceptación de extremo a extremo. Falta validación de negocio
competente y proveedor de permiso nominal. Las preguntas pendientes no detienen
la programación independiente ni autorizan a inventar plazo o autoridad.

## Probar el recorrido disponible

1. Lea el manual de su perfil y la [guía canónica](GUIA_RECORRIDO_ALBERTO.md).
   Sistemas prepara el servidor, la base y el acceso del navegador.
2. Abra `/portal-empleado/` en el entorno autorizado, con el certificado de
   desarrollo correspondiente. RRHH e Intervención usan perfiles separados.
3. Entre en **Contratación temporal → Nueva petición**. Use exclusivamente
   las entradas sintéticas de los catálogos y los ejemplos de la guía.
4. Confirme una sola vez cada actuación y conserve la referencia del
   expediente, su versión, la clave de operación cuando corresponda y el
   recibo. Un error de conexión no demuestra que no se haya guardado nada.
5. Para continuar el llamamiento existente, utilice sus datos y claves
   originales. Recuperar no significa crear otra solicitud o preparar una
   clave nueva. La guía conserva el ejemplo exacto de la respuesta y enlaza
   el `.eml` sintético que debe cargarse sin cambios. La recuperación ya está
   comprobada; no eluda un rechazo cambiando claves o repitiendo el registro.
6. Para verificar un reinicio, siga la guía conservando PostgreSQL y el
   material de seguridad. El resultado recuperado mantiene recibo y fecha;
   no debe duplicar el efecto.

**Registrada localmente · Sin entrega acreditada** significa que se ha
guardado un aviso en el servidor de desarrollo. No es un correo enviado,
una notificación recibida ni una aceptación de candidatura.

## Módulos y superficies: estado honesto

| Área | Qué existe | Qué no debe darse por terminado |
|---|---|---|
| Contratación temporal | Recorrido real descrito arriba y pantallas de tramitación. | Bandeja completa en la base de referencia, resto del paso 6 y pasos 7–8. |
| Bolsa interna | Dominio, servicios, persistencia y pantallas reutilizables; proveedor durable conectado al llamamiento de Contratación temporal. | Gestión completa de convocatorias, borradores, méritos, alegaciones, contratos, firma y notificaciones desde el portal. |
| Bolsa pública | Consulta de convocatorias, categorías, detalle y documentos; composición pública separada. | Inscripción personal, consulta privada de posición y tramitación administrativa completas. Su disponibilidad depende del entorno configurado. |
| Personal y Nóminas | Módulo, contratos y material funcional de desarrollo/presentación. | Maestro de personal, nómina y procedimientos corporativos completos. |
| Cronos | Módulo y pantallas/capacidades de desarrollo y presentación. | Gestión corporativa completa de jornada, fichajes y aprobaciones. |
| Dietas | Módulo, presentación y conector cartográfico interno para cálculo de rutas. | Liquidación oficial, aprobación y pago completos; una ruta calculada no acredita una dieta aprobada. |
| Capacidades comunes | Identidad, permisos, catálogos, documentos, auditoría y eventos con distintos grados de conexión. | Disponibilidad automática de todas las capacidades en todos los módulos. |

Las 17 vistas del menú de Bolsa **no son 17 funciones administrativas
terminadas**. El [manual de usuario](docs/manual_usuario/manual_portal_bolsas.md)
clasifica cada opción sin confundir componentes escritos con recorridos
habilitados.

La presentación se identifica como DEMO y permite explorar pantallas y
resultados sintéticos. Sus adaptadores volátiles no sustituyen un servicio
real no disponible. La ayuda, los audios y las capturas de presentación
tampoco acreditan un envío, una firma o una actuación administrativa.

## Arquitectura y orientación en el código

Arquitectura hexagonal: el dominio expresa las reglas, la aplicación coordina
los casos de uso, los puertos fijan los intercambios y los adaptadores
conectan HTTP, PostgreSQL y otros proveedores. La raíz conecta esas piezas;
la existencia de un adaptador no garantiza que esté expuesto.

| Directorio | Responsabilidad |
|---|---|
| [internal/modules/contrataciontemporal](internal/modules/contrataciontemporal) | Procedimiento de contratación temporal y coordinación con otros módulos. |
| [internal/modules/bolsa](internal/modules/bolsa) | Reglas y capacidades propietarias de Bolsa. |
| [internal/vec](internal/vec) | Capacidades comunes del portal. |
| [internal/app/bootstrap](internal/app/bootstrap) | Ensamblaje y dependencias del entorno de desarrollo. |
| [internal/app/composicion](internal/app/composicion) | Raíces y superficies separadas de la aplicación. |
| [web/static](web/static) | Portal, formularios, recursos públicos y presentación. |
| [deploy/postgresql](deploy/postgresql) | Persistencia, funciones y roles por capacidad. |
| [config](config) | Configuración de los procesos y conexiones. |

Cada módulo conserva su autoridad: otro módulo no copia sus reglas ni accede
directamente a sus tablas. Las integraciones reutilizan sus puertos.
Las superficies pública, interna y de presentación tienen alcances distintos;
no deben intercambiarse como atajo para habilitar operaciones.

## Requisitos y configuración

- [go.mod](go.mod) declara Go `1.25.12` como mínimo y `go1.26.5` como
  herramienta de referencia del proyecto.
- El recorrido real requiere PostgreSQL, conexiones nominales separadas y
  certificados de desarrollo; su preparación está en la guía y el manual
  de Sistemas. No basta con arrancar una pantalla estática.
- Docker, las instancias de base y los materiales ya conservados se operan
  según la [guía de recorrido](GUIA_RECORRIDO_ALBERTO.md), sin recrearlos
  para repetir una operación.
- La referencia de [procesos y configuración](docs/manual_programador/cmd_y_configuracion.md)
  complementa el [código de configuración](config). Use los valores del
  entorno autorizado; no copie secretos ni conexiones a Git.

No se incluyen aquí contraseñas, certificados, conexiones privadas ni órdenes
de despliegue. El arranque de desarrollo no constituye una puesta en
producción.

## Documentación de referencia

- [Especificación del expediente remitido por RRHH](docs/portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Arquitectura técnica modular](docs/portal_vec/arquitectura_tecnica.md).
- [Catálogo de contratos de API por módulo](docs/portal_vec/contratos_api_modulos.md).
  Describe contratos; para saber qué está conectado consulte la guía y los
  manuales de esta edición.
- [Historial de decisiones](docs/portal_vec/registro_decisiones.md).
  Conserva antecedentes con su fecha y alcance, no una orden de trabajo
  vigente por el mero hecho de estar enlazado.
- [Documentación del proyecto](docs/): requisitos, referencias y material
  técnico conservado.

Los manuales explican el uso; la guía conserva los comandos y datos del
recorrido. Los documentos históricos y las exportaciones anteriores deben
leerse con su fecha, sin convertir resultados antiguos en estado actual.

## Desarrollo, seguridad y ayuda

Antes de trabajar, lea completas las [instrucciones del repositorio](AGENTS.md)
y las instrucciones de desatasco vigentes facilitadas por dirección. Reutilice
la línea canónica de cada capacidad: inventar una implementación equivalente
en otra rama no es avance.

Los cambios se agrupan en hitos observables y se comprueban según su riesgo;
la documentación no exige ejecutar suites de producto. Las zonas sensibles
conservan las revisiones requeridas. El manual del programador recoge el
procedimiento técnico, sin imponerlo a quien solo utiliza el portal.

Se mantienen denegación por defecto, autorización en servidor, datos
minimizados y ausencia de cookies y almacenamiento web. No se usan datos
reales sin autorización expresa ni se interpreta una indisponibilidad como
éxito. Un recibo confirma únicamente el efecto que describe.

Para ayuda de uso, consulte el manual de usuario y **Ayuda** en el portal.
Para incidencias, conserve el mensaje, fecha, perfil y referencia de operación
por el canal autorizado. No publique datos privados ni credenciales en GitHub.

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
