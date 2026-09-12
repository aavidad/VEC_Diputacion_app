# VEC Diputación de Granada

Copyright (c) 2026 Alberto Avidad (avidad@dipgra.es), para la Diputacion
Provincial de Granada. Publicado bajo la
[Licencia Publica de la Union Europea v1.2 (EUPL-1.2)](LICENSE).

## Incorporación, GINPIX y Word comprobados en navegador — 12 de septiembre de 2026

El código publicado avanzó a `7211ac973306b067525a45d05d7914a9a4a710f5`.
El runtime continúa en `e4ffa72de593b9500ee155e8dd3256474984d831`, con el binario
`1b21f31d299b94133e1e5b14b313d2e8fff0aef9909824813abfbd4a609d766e`.
Chrome obtuvo `200` al recuperar la incorporación y la ficha GINPIX, sin
repetir el `POST`: se conservaron el recibo `ref:2bc3d281…`, la fecha
`2026-09-10T13:07:06.614186Z` y los seis campos cotejados. La ficha GINPIX
descargada tiene SHA256
`4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`.

Los seis borradores Word se descargaron con HTTP `200` y son ZIP válidos. El
recorrido no produjo errores JavaScript ni datos en cookies o almacenamiento;
la incorporación no mostró desbordamiento a 1440, 1024 o 390 px. Son
borradores de desarrollo: no acreditan firma, eficacia, envío ni transmisión.

La agrupación a ancho completo de los seis pares PDF/Word está integrada en
código y revisada, pero aún no está desplegada. Faltan comprobar esa agrupación
en runtime y la recuperación después de reiniciar PostgreSQL. Auth13 está
instalada una sola vez; CT86/87 y AD3-30/31 siguen sin ensayo ni instalación.

El siguiente corte frontend integra fechas civiles UTC legibles sin desplazar
el día, paginación mediante cursor opaco de un solo uso y validación accesible
del filtro de hasta 80 caracteres. Los errores recuperables conservan el
cuadro y no lanzan otra consulta por una entrada inválida. Sus dos revisiones
estáticas dieron `GO` y la suite web terminó 351/351. El despliegue y el E2E de
paginación siguen pendientes.

## Historia del estado funcional anterior — 12 de septiembre de 2026

Este apartado y los bloques cronológicos inferiores se conservan como historia;
el estado vivo es el descrito al inicio de este documento.

Base del cierre funcional: `00558603dbd3040eacb03cb511f3b840be241b10`, rama
`integracion/ct-producto-ligero-20260821`. El desarrollo activo y la base
sintética están en el servidor; las instrucciones antiguas de arranque local
no describen la instancia actual.
Este entorno usa exclusivamente datos sintéticos y no está autorizado para
producción ni para tratar datos reales.

Se pueden enseñar **cinco pasos completos y partes del sexto, séptimo y octavo**.
El 10 de septiembre se registró de forma real en desarrollo una
incorporación sintética: formulario `GET 200`/`POST 200`, dos confirmaciones
expresas, recibo registrado y alta sintética en Personal. Esta capacidad pertenece al
octavo paso, pero todavía no puede enseñarse como ciclo recuperado completo.

La resolución manual de ejercicio se comprobó con Chromium, certificado de
pruebas, autorización del servidor y PostgreSQL reales: alta `201`,
recuperación y repetición con la misma clave `200`, mismo recibo y sin duplicado. El caso avanza de versión
`7` a `8`; eso **no significa ocho pasos terminados**. Se conservan 52
expedientes y una nueva resolución de ejercicio; el caso original en versión
`7` permanece disponible.

**No hay firma oficial, eficacia administrativa ni envío.** La incorporación
ya está integrada y tuvo escritura visible en navegador, pero su recuperación
después del reinicio aún no está acreditada: la lectura devolvió `GET 503` por
la restauración histórica de Auth12. Auth13 conserva dos dictámenes `GO` y
regresión verde en `ef6704a6`, pero sigue pendiente de instalación. Por eso no
se declara todavía recorrible de extremo a extremo ni se eleva la métrica.

CT70–85, trece pools y las tres capacidades con sus catálogos y definición ya
están provisionados; no son trabajo pendiente ni deben reaplicarse. No hubo
precargas de negocio. GINPIX ya tiene preparación, montaje y descarga V2
integrados en código; falta comprobarlos desde el recibo recuperado en el
servidor. La anotación y el cierre también tienen piezas de dominio y
persistencia integradas, pendientes de su recorrido conjunto.

El portal está servido de forma **privada**, no en una URL pública.
`https://localhost:8443/portal-empleado/` corresponde al servidor remoto;
no funciona directamente en el equipo del visitante sin acceso preparado.
Consulte [acceso y operación actuales](docs/manual_sistemas/README.md#entorno-privado-vigente).
El certificado identifica al usuario de pruebas: no firma los documentos.

El 12 de septiembre se inventarió y conservó el trabajo local y remoto. El
backend pendiente de anotación y cierre está revisado y se ha corregido el error
que presentaba observaciones inválidas como indisponibilidad. La base conjunta
supera las pruebas Go, `go vet`, 314 pruebas web y los manifiestos. El director
remoto y sus agentes continúan la corrección del frontend, la preparación
operativa y la documentación; todavía no se acredita un nuevo recorrido
en la aplicación servida. El
[estado y plan vigentes](ESTADO_PROYECTO.md) conservan la continuación exacta.

## Empiece por su perfil

| Perfil | Manual | Para qué sirve |
|---|---|---|
| Usuario del portal | [Manual de usuario](docs/manual_usuario/manual_portal_bolsas.md) | Acceso, navegación, opciones reales frente a DEMO, recibos, mensajes y ayuda. |
| Recursos Humanos | [Manual de RRHH](docs/manual_rrhh/README.md) | Tramitación, responsabilidades, resultados esperados y límites de cada paso. |
| Programación | [Manual del programador](docs/manual_programador/README.md) | Composición, código reutilizable, contratos, desarrollo y comprobaciones por hito. |
| Sistemas | [Manual de Sistemas](docs/manual_sistemas/README.md) | Preparación del entorno, configuración, certificados, persistencia y operación. |

La [Guía de recorrido de Alberto](GUIA_RECORRIDO_ALBERTO.md) es la referencia
de los datos sintéticos conservados y de los recorridos anteriores. Para
el arranque y acceso al servidor actual, use el manual de Sistemas; no ejecute
las recetas locales históricas como si describieran esta instalación.

## Qué funciona de extremo a extremo

El recorrido acreditado utiliza navegador con certificado de cliente,
servicios reales de aplicación, autorización de servidor y PostgreSQL.
No utiliza el adaptador DEMO para afirmar un guardado.

La [organización de referencia](GUIA_RECORRIDO_ALBERTO.md#centros-y-organización-de-referencia)
permite consultar centros y preparar altas o cambios de unidades con motivo,
revisión y recibo persistentes. Alta y edición sintéticas comprobadas tras
reinicio, sin duplicados. No asigna ocupantes, concede permisos ni habilita
todavía la ratificación multicientro.

El [circuito previo del centro](GUIA_RECORRIDO_ALBERTO.md#petición-del-centro-y-ratificación)
permite presentar y ratificar una petición con dos identidades sintéticas
configuradas, formularios reales y recibos persistentes tras reinicio.
La entrega posterior al alta de RRHH ya se comprobó en el entorno privado,
con recibo conservado tras reinicio. No es firma documental ni habilita
automáticamente a todos los centros del catálogo.

| Paso | Recorrido disponible | Límite |
|---|---|---|
| 1. Solicitud | Alta desde formulario y primer recibo del expediente. | Datos y catálogos de desarrollo. |
| 2. Análisis | Registro del análisis por RRHH y nueva versión del expediente. | El ejemplo recorrido utiliza Sustitución; el formulario ofrece cinco modalidades. |
| 3. Bolsa | Propuesta y decisión de cobertura por **Bolsa vigente**. | No equivale a gestionar de principio a fin una convocatoria de Bolsa. |
| 4. Asignación | Registro de unidad y persona responsable referenciada. | Destino sintético configurado. |
| 5. Informe jurídico y Fiscalización | Documento de desarrollo y resultado favorable, favorable con observaciones o desfavorable; este último registra devolución a la unidad. | El documento no tiene firma ni validez jurídica. Fiscalización corresponde al perfil de Intervención. |
| 6. Llamamiento, parcial | Recorridos sintéticos, aviso CT62, declaración CT63 y aceptación manual del sucesor CT64 recuperables tras reinicio principal, sin duplicados. | Faltan vencimiento, envío corporativo y plazo. No acredita entrega ni plazo legal aprobado. |
| 7. Nombramiento, parcial | Propuesta desde aceptación sintética, `201` y recuperación `200` tras reinicio, expediente `6→7`; seis borradores descargables y validación manual sintética con recibo `7→8`. | Sin nombramiento eficaz, posesión real, firma, envío ni entrega; validación manual sintética disponible, no firma oficial. |
| 8. Incorporación y seguimiento | Pendiente como recorrido completo. | No se acredita incorporación, integración con GINPIX ni cierre del seguimiento. |

La métrica es **cinco pasos completos más partes del sexto y séptimo**, no un
porcentaje global ni un recuento de pantallas, contratos o pruebas.

### Antecedentes de los recorridos, no inventario de la instancia actual

Los párrafos siguientes conservan cifras y pruebas de sus cortes originales.
El inventario actual es de 52 expedientes sintéticos en el servidor privado;
las referencias a dos bases y aplicaciones corresponden al entorno anterior.

La bandeja y el detalle ya están conectados en `b2effba`: se demostraron 50
solicitudes conservadas y un análisis desde una de sus filas, sin otra alta.
Esto mejora la continuidad del trabajo; no cierra por sí solo otro paso del flujo.
El registro de respuesta conserva una respuesta, un asiento y un evento;
no cambia Bolsa, no avanza el expediente y mantiene comunicación versión `2`.
El `.eml` sintético se lee y resume con SHA256 en el navegador: no se sube ni
se custodia. El aviso sigue siendo local, no correo corporativo entregado.
AD3 `000015` / Bolsa `000004` y AD3 `000017` / CT `000058` están instaladas en
ambas bases, con ambas apps en la compilación corregida; AD3-16/CT57 ya estaban.
No reaplicar. La prueba aislada anterior de Bolsa4 (`8197db3`) usó un doble
privado transaccional. El roundtrip de aceptación UP/DOWN de esas cuatro migraciones
verificó reversión exacta en ROLLBACK, sin modificar autorización ni usar dobles.
El navegador actual sí usó criptografía real; no confundir estas comprobaciones.
AD3 `000018` / Bolsa `000005` / CT `000059` también están instaladas en ambas
bases: dirección confirmó UP/DOWN con ACL, funciones y comprobaciones conservadas.
No reaplicar ni ejecutar DOWN sobre los registros guardados.
Las preguntas pendientes no detienen la programación independiente ni autorizan
a inventar plazo o autoridad; continúa **5/8 más partes del sexto y séptimo**.

## Probar el recorrido disponible

1. Lea el manual de su perfil y la [guía canónica](GUIA_RECORRIDO_ALBERTO.md).
   Sistemas prepara el servidor, la base y el acceso del navegador.
2. Abra `/portal-empleado/` en el entorno autorizado, con el certificado de
   desarrollo correspondiente. RRHH e Intervención usan perfiles separados.
3. Para enseñar el trabajo conservado, entre en **Contratación temporal**,
   localice el expediente indicado por el operador y abra su detalle.
   Use **Nueva petición** solo si se ha acordado crear otro caso sintético.
4. Confirme una sola vez cada actuación y conserve la referencia del
   expediente, su versión, la clave de operación cuando corresponda y el
   recibo. Un error de conexión no demuestra que no se haya guardado nada.
5. Para continuar el llamamiento existente, utilice sus datos y claves
   originales. Recuperar no significa crear otra solicitud o preparar una
   clave nueva. La guía conserva el ejemplo exacto de la respuesta y enlaza
   el `.eml` sintético de aceptación que debe cargarse sin cambios. El caso de
   renuncia usa su propio material y claves; su recuperación tras reinicio también está confirmada.
   No eluda un rechazo cambiando claves o repitiendo el registro.
6. La recuperación tras reinicio ya está acreditada. No la repita para leer
   esta documentación o presentar el caso. Si Sistemas acuerda una nueva
   comprobación, conservará base, material, clave, recibo y fecha originales.

**Registrada localmente · Sin entrega acreditada** significa que se ha
guardado un aviso en el servidor de desarrollo. No es un correo enviado,
una notificación recibida ni una aceptación de candidatura.

## Módulos y superficies: estado honesto

| Área | Qué existe | Qué no debe darse por terminado |
|---|---|---|
| Contratación temporal | Bandeja, detalle y recorrido real descrito arriba. | Resto del paso 6, formalización oficial e incorporación/seguimiento completos. |
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
- La instancia y el material conservados se operan según el
  [entorno privado vigente](docs/manual_sistemas/README.md#entorno-privado-vigente).
  Las recetas Docker locales de la guía son históricas; no crean otra instancia.
- La referencia de [procesos y configuración](docs/manual_programador/cmd_y_configuracion.md)
  complementa el [código de configuración](config). Use los valores del
  entorno autorizado; no copie secretos ni conexiones a Git.

No se incluyen aquí contraseñas, certificados, conexiones privadas ni órdenes
de despliegue. El arranque de desarrollo no constituye una puesta en
producción.

## Documentación de referencia

Para uso actual, empiece por los cuatro manuales anteriores. En GitHub,
la rama de producto es `integracion/ct-producto-ligero-20260821`; la rama
predeterminada `vec-orquesta-20260619` puede mostrar el corte del 31 de julio.
Esta actualización no cambia la rama predeterminada ni acredita su publicación.
La verificación y cualquier cambio de esa selección corresponden a dirección.

- [Especificación del expediente remitido por RRHH](docs/portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Arquitectura técnica modular](docs/portal_vec/arquitectura_tecnica.md).
- [Catálogo de contratos de API por módulo](docs/portal_vec/contratos_api_modulos.md).
  Describe contratos; para saber qué está conectado consulte la guía y los
  manuales de esta edición.
- [Historial de decisiones](docs/portal_vec/registro_decisiones.md).
  Conserva antecedentes con su fecha y alcance, no una orden de trabajo
  vigente por el mero hecho de estar enlazado.
- [Índice de requisitos históricos](docs/estudio_requisitos/README.md):
  estudio de julio, no guía de arranque ni plan operativo actual.
- [Referencias externas archivadas](docs/referencias_portales_aapp/README.md).
- [Paquete de cumplimiento pendiente de validación](docs/cumplimiento/LEEME.md)
  e [informe histórico del Comité de Seguridad](docs/comite_seguridad/LEEME.md).
- [Documentación del proyecto](docs/): archivo técnico; sus actas, revisiones,
  decisiones y relevos fechados mantienen su alcance original, no son el
  punto de arranque actual. No se reescriben ni se renuevan sus fechas.

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
