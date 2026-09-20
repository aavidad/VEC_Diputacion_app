# Estado y plan de ataque del proyecto

## Bolsa B10 operativa en runtime aislado de servidor — 20 de septiembre de 2026

La publicación y la consulta pública B10 están recorridas en un entorno nuevo
y segregado del servidor. PostgreSQL 18.4 usa volumen, roles y TLS propios; el
proceso `vec-publico` escucha únicamente en `127.0.0.1:18082`. No se modificó
Caddy, no se publicó la ruta en Internet y no se tocó ninguna base conservada
de Contratación temporal.

La primera ejecución falló cerrada por dos precondiciones reales. La cuenta
lectora detectó un privilegio `TEMPORARY` residual de `PUBLIC` en la base
`postgres`; la reparación privada revocó todos los privilegios de `PUBLIC` en
`postgres` y `template1`. Después, el lector rechazó una proyección sintética
que solo contenía un catálogo genérico. La segunda publicación incorporó los
cinco catálogos exigidos por el contrato y generó una nueva ancla V3:
`249c28a2cae6f83658fc3172d80551e41b7a54a4a79f5f10557e0a9f6eaf051e`.
Los dos cambios recibieron revisión independiente de SQL e identidad con
`GO`, `P0=P1=P2=0`. Los scripts, credenciales, certificados y material
sintético permanecen fuera de Git.

`/livez`, `/readyz`, la relación de bolsas y el detalle de lista respondieron
`200`. La API devuelve **12 bolsas y 390 posiciones**, y PostgreSQL conserva
los mismos recuentos, cinco catálogos y la misma ancla. El primer listado
recorrido devolvió sus 41 posiciones completas. El reinicio de la aplicación
y el reinicio conjunto de PostgreSQL y aplicación conservaron ancla, instante
y recuentos. Una segunda ejecución del despliegue terminó correctamente sin
republicar ni duplicar datos.

Si PostgreSQL se reinicia mientras la aplicación permanece viva, la consulta
puede responder temporalmente `503` aunque `/readyz` ya devuelva `200`; el
reinicio posterior de la aplicación recuperó inmediatamente el recorrido.
Este desacoplamiento entre disponibilidad de la fuente y `readyz` queda como
deuda operativa antes de exponer B10 mediante el acceso autenticado previsto.
El runtime aislado acredita publicación y lectura sintéticas recuperables; no
acredita identidad institucional, datos reales, Internet ni producción.

## Bolsa B10 dispone de publicación operativa validada — 20 de septiembre de 2026

El subcomando `vec-server publicar-proyeccion-publica` recibe por fichero la
proyección SQL V2 completa, su manifiesto canónico V2 y la lista pública B10.
Antes de escribir deriva el manifiesto desde la proyección, recalcula las
huellas de convocatoria, exige la misma ancla y prepara una única publicación
V3 mediante el adaptador PostgreSQL existente. El DSN procede exclusivamente
de configuración y los errores no revelan rutas, material ni credenciales.

La validación rechaza claves duplicadas, aliases, campos ausentes, `null`
incompatible, fuentes divergentes y límites excedidos. El transporte conserva
`bolsas:[]` y `posiciones:[]`, y usa exactamente el literal temporal que la
frontera SQL compara. Pruebas focales, `go vet`, compilación y `git diff --check`
terminaron correctamente. Dos revisiones independientes dieron `GO`, con
`P0=P1=P2=0` pendientes sobre los cinco archivos del corte.

Este corte incorpora la herramienta de publicación; todavía no acredita una
publicación real. La siguiente comprobación se hará en una PostgreSQL pública
nueva y aislada, con sus LOGIN, TLS y proceso `vec-publico`, sin instalar ni
revertir migraciones en las bases conservadas ni exponer rutas en Internet
antes de disponer del acceso autenticado.

## Bolsa B10 conectada a la composición pública — 20 de septiembre de 2026

El corte B10 incorpora una proyección PostgreSQL separada para bolsas vigentes
y posiciones con documento enmascarado, un manifiesto V3 común con las
convocatorias y el lector que monta las rutas existentes de consulta pública
en `cmd/vec-publico`. La publicación es atómica, el lector enumera doce vistas
y cualquier escritura lateral invalida el mismo testigo. No se publican
nombres, contactos ni referencias internas de persona o candidatura.

La prueba focal instaló las migraciones `000001` y `000002` en PostgreSQL 18.4
efímero, publicó una bolsa con dos posiciones, ejecutó el lector Go y comprobó
ACL, redacción de errores, invalidación y el ciclo UP→DOWN→UP. Las pruebas Go
de canónico, adaptador, HTTP y composición también terminaron correctamente.
Dos revisiones independientes dieron `GO`, con `P0=P1=P2=0` pendientes.

El corte todavía no está desplegado: `vec.cidonia.cloud` conserva la portada
cerrada y el servidor no dispone aún de la PostgreSQL pública dedicada, los
LOGIN segregados, TLS ni el proceso `vec-publico`. La prueba efímera del lector
no sustituye el recorrido conjunto TLS/ACL/manifiesto/HTTP previo a publicarlo.

## Bolsa RRHH consulta datos constituidos — 20 de septiembre de 2026

El corte canónico `081cc0bc9abfc8ba138d50f28bee5198565d5c6b` retira de
B12 y B5 la lectura y mezcla del fichero DEMO. Las dos rutas internas consultan
ahora exclusivamente la constitución durable y el lote protegido asociado; si
falta la fuente, el lote o una fila exacta, responden `503` sin fabricar una
identidad ni devolver una lista parcial. Dos revisiones independientes dieron
`GO`, sin hallazgos P0, P1 o P2 nuevos, y la prueba focal del montaje terminó
correctamente.

La instancia sintética principal ejecuta el binario SHA256
`4619f0d02309de97db55e15bb3c7306b63f390e227a0ddb5e2daa8e316177fc8`.
Después de sustituir el artefacto y reiniciar la aplicación, B12 respondió
`200` con esquema `vec.bolsa.rrhh.bolsas.v1` y **12 bolsas**; B5 respondió
`200` con cinco candidatos en la página solicitada, documento enmascarado y
cursor de continuación. No se aplicó ni revirtió SQL y se conserva el binario
anterior para recuperación.

Esta evidencia acredita las dos lecturas internas en el entorno de desarrollo
con mTLS y datos sintéticos. No acredita identidad corporativa, producción ni
la consulta pública B10: esa ruta sigue requiriendo una proyección pública
separada de los datos protegidos.

## Bolsa completa su superficie de presentación — 20 de septiembre de 2026

El corte canónico `a899983bf766ceba46ac7f04416dca8985fb1dd2` permite
renderizar las **34/34 superficies inventariadas de Bolsa**: 14 del área
personal, 18 de gestión interna y 2 públicas. Esta cifra mide cobertura de
interfaz. Las 14 superficies personales y las operaciones administrativas sin
composición siguen rotuladas como **DEMO** y no producen efectos.

Las lecturas ya conectadas conservan su alcance: B12 presenta el cuadro interno
de bolsas, B5 la lista y ficha interna de aspirantes, y B10 la consulta pública
de bolsas y posiciones mediante su contrato público `v1`. Son consultas de
solo lectura con los campos minimizados previstos; no convierten el resto de
las pantallas en capacidades reales ni acreditan identidad, persistencia o
tramitación completa.

El catálogo i18n común ya gobierna parte de la navegación y los formatos de
Bolsa en el portal, el área personal y la consulta pública. La integración es
parcial: los códigos de negocio permanecen separados de los textos, pero aún no
se declara localizada toda la superficie. Las pruebas focales Node sobre los
archivos afectados terminaron con **151/151 casos correctos**. No se ejecutó
recorrido de navegador porque este entorno no dispone de Chrome ni Playwright;
por tanto no se afirma validación visual, E2E ni publicación productiva.

Personal incorpora una vista informativa **DEMO**, con datos sintéticos y
efímeros, sin cálculos de derechos, importes, bases, pagos ni operaciones. Los
módulos existentes de Cronos y Dietas se conservan y muestran sus límites DEMO
o pendientes sin atribuirles integración institucional. `dudas.md` reúne ahora
las preguntas **1–36** para Contratación, Bolsa, Cronos, Dietas, Personal,
Administración y Usuarios/Contacto. Este corte no cambia los porcentajes
históricos de Contratación ni permite declarar VEC productiva.

## Runtime principal alineado y auditoría de frontera instalada — 19 de septiembre de 2026

La instancia principal sintética ejecuta el corte canónico
`d1307f61283933530c935c5e1e1f5fd9fa15ed09`. El binario servido tiene SHA256
`f17b164b49dd9a31513da8582c0f5ba641230ce3129ccdfbc94cdcd38135eba8` y el
inventario de contenido web SHA256
`ad4c439bd1c9c97a75ee82f51c43a5391dcbba8f4b6a8c3eecb547d951cf12e0`.
El runtime anterior permanece conservado para recuperación. El primer reinicio
tropezó con la gestión `cgroupfs`; se corrigió el arranque con
`cgroup-manager=systemd`, sin volver al binario anterior. Las instancias aisladas
de prueba no se modificaron.

El corte incorpora la conservación de la auditoría cuando el cliente cancela, un diagnóstico interno clasificable sin filtrar causas y etiquetas funcionales traducidas en seguimiento.

Las mismas cinco consultas de solo lectura —incorporación, ficha GINPIX,
anotación administrativa, preparación de cierre y seguimiento— respondieron `200`
en la línea base, después de sustituir el runtime y después de reiniciarlo. No se
ejecutaron `POST`, no se emitieron cookies y no aparecieron duplicados. Esta
comprobación acredita continuidad del recorrido sintético de consulta; no
acredita efectos institucionales, producción ni validez legal.

La migración de auditoría de frontera `000108` está instalada únicamente en la
base sintética principal. Las revisiones independientes de SQL e identidad dieron
`GO`. Las cinco operaciones SQL directas sobre su tabla —`SELECT`, `INSERT`,
`UPDATE`, `DELETE` y `TRUNCATE`— quedaron denegadas con `42501`; la tabla conserva
cero filas. No se instaló la migración en bases aisladas ni se ejecutó `DOWN`.

Contratación temporal conserva **19/19 superficies visibles**, pero su estado
funcional es **15/19 completas, tres parciales y una bloqueada por la integración
real con Portafirmas**. La cobertura de pantallas no equivale a producción ni
cierra SMTP, plazos, firma, identidad u otros efectos aún pendientes.

## Portada pública cerrada — 19 de septiembre de 2026

La secuencia `15e437bb`, `d86fe653` y `cd3df5ab` publica en
`https://vec.cidonia.cloud/` una portada de acceso cerrada. Cl@ve, certificado
digital y DNIe aparecen como opciones previstas, visibles y deshabilitadas. No
existe todavía login operativo, sesión web ni acceso privado desde Internet:
faltan integrar los proveedores, el verificador y el gateway que establecerán
esas autoridades. `admin.cidonia.cloud` continúa cerrado.

La prueba de runtime confirmó navegación privada con `303` hacia `/`, API con
`401`, ausencia de cookies, ruta lateral inexistente con `404`, CSP estricta y
los demás vhosts atendidos con `200`. No se expusieron rutas privadas ni datos
personales. Esta portada informa del acceso futuro; no acredita autenticación,
autorización, publicación del portal privado ni producción.

## Corte web primero de Contratación — 19 de septiembre de 2026

La preparación para la presentación prioriza pantallas completas y recorribles,
con datos sintéticos identificados, antes de conectar los efectos institucionales
pendientes. Los contratos actuales se conservan para sustituir el adaptador de
presentación por HTTP sin rehacer las vistas.

Los cortes `5fc0d429`, `2e5810b7` y `01f2bb68` dejan la rectificación en consulta
cuando no hay motivos publicados, muestran Llamamiento y Formalización solo en
su fase acreditada y separan el resultado manual sintético del vencimiento no
evaluable. Los cortes `0b9d0a7e`, `6efd48b3`, `4a5a4a9e` y `daab1278`
presentan firma y formalización sin inventar firmantes, orden, envío o recibo,
y dejan plazos, canal y reglas pendientes donde RRHH todavía debe decidir.

La web de presentación tiene **19/19 superficies recorribles**: cuadro y
dieciocho tareas. El servidor aislado de presentación respondió `200` y el
recorrido Playwright pasó en 1440 × 1000, 1024 × 900 y 390 × 844, con las
dieciocho tareas presentes, cero errores JavaScript y cero desbordamiento
global. Firma, formalización, llamamiento, resultado y preparación previa
mantienen sus límites visibles. Los activos servidos coinciden con el corte.

El contador funcional se mantiene en **15/19 (79 %)**: la cobertura visual no
sustituye Portafirmas, SMTP, la política de inicio/plazo ni la identidad de la
reserva histórica pendiente. Las pantallas, contratos e i18n se conservan para
conectar esas autoridades sin rehacer la interfaz.

## Consulta de incorporación sin preparación — 19 de septiembre de 2026

El corte `dcf89da1549ab7173c9a8cbf23da82c4b040fb9b`, integrado en la línea
remota, separa la autorización para consultar el detalle organizativo de la
existencia del plan durable de incorporación. Un expediente sintético sin plan
responde `409 preparacion_pendiente`; uno que ya tiene plan responde `200`. El
actor conserva el permiso nominal de detalle en su organización, pero confirmar
la incorporación y los demás efectos siguen exigiendo un plan válido y fallan
cerrados cuando falta.

Dos revisiones independientes de identidad dieron `GO`, con `P0=P1=P2=P3=0`.
El centinela previo pasó en dos paquetes Go y `6/6` casos Node; la prueba focal
`go test -count=1 ./internal/app/bootstrap -run
'^TestIncorporacionV2PermisosNominales'` terminó en `ok` con Go 1.26.6. Cubre
tanto el detalle sin plan como el rechazo previo a publicación/PDP de la
confirmación sin plan. El runtime privado se reinició para observar ambos casos,
sin `POST`, SQL ni escritura en la base. Es código de desarrollo con datos
sintéticos: no acredita producción, incorporación efectiva ni validez legal.

La comprobación posterior en navegador abrió un expediente sintético `v9` sin
plan: portal, cuadro y detalle respondieron `200`, y la consulta de incorporación
`409`. La pantalla mostró la preparación no disponible, un botón **Actualizar**
y cero formularios. Ancho de viewport/contenido: `1440/1440` y `390/390`; cero
errores JavaScript, cookies, `localStorage` y `sessionStorage`. Los artefactos se
conservan en custodia privada.

## Bolsa: inventario y consulta pública — 19 de septiembre de 2026

El inventario actual de Bolsa reúne **34 pantallas**: 18 de gestión interna,
14 del área personal y 2 públicas. Las 14 del área personal son recorribles
visualmente solo en modo **DEMO**; ninguna está conectada de extremo a extremo
como capacidad productiva y tres muestran información parcial del entorno de
desarrollo. Esta medida separa cobertura visual de integración real.

El corte `2e3b0cca`, integrado y publicado, elimina dos consultas anticipadas a
endpoints de Bolsa que todavía no están compuestos. No suprime ni maquilla los
fallos de una operación solicitada por la persona usuaria: esos errores siguen
mostrándose en su contexto.

El corte `fbb573b0`, también integrado y publicado, conecta las convocatorias
públicas B13 al contrato vivo `v1`. La API servida respondió con esquema `v1`
y 12 registros. `ad520906` incorpora ese contrato `v1` a los manifiestos web
público y de producción para que el activo forme parte de ambos empaquetados.

El corte `94223751`, integrado y publicado, añade desde B5 una ficha de
participación de solo lectura. Reutiliza los datos ya disponibles del candidato
y no introduce edición, efectos administrativos ni otro origen de verdad.

Tras reiniciar la aplicación, las comprobaciones HTTP devolvieron `200` para
`/bolsa/`, `contrato-v1.js` y `portal.js`; el activo servido incluye la ficha y
ya no inicia la consulta anticipada a `/api/vec/bolsa/panel`. La API pública
mantiene el esquema `v1` con 12 registros. El `HEAD` canónico y la rama remota
coinciden en `94223751ae38124d52e767d6fcca2548237a5d2f`. Estas observaciones
acreditan publicación de activos y respuestas HTTP tras reinicio; no acreditan
todavía un recorrido E2E en navegador, identidad real, persistencia del área
personal ni producción.

### Corte visible desplegado — 19-09-2026

Los commits canónicos `d6ccd629` y `05458427`, integrados y publicados en la
rama remota de GitHub en `05458427`, mantienen visibles y deshabilitados cuatro
accesos de menú a capacidades aún no compuestas y presentan las vigencias B12 como
fecha civil o como instante localizado, según el dato recibido. El runtime
privado recibió únicamente los cuatro archivos web afectados, con copia de
respaldo fuera de Git.

El recorrido en navegador a 1440 y 390 px mostró 12 vigencias B12 y los cuatro
controles deshabilitados, sin desbordamiento, errores JavaScript, respuestas
no-2xx, cookies ni almacenamiento web; el POST automático fue solo de lectura.
La URL pública `vec.cidonia.cloud` continúa inaccesible porque Caddy en 443 no
tiene aún el vhost ni el passthrough mTLS; el listener privado
`127.0.0.1:8443` sí respondió `200`. Este corte no acredita Mi Bolsa ni
producción y no cambia las métricas de Contratación temporal.

## Cartografía de Dietas conectada al runtime privado — 19 de septiembre de 2026

OSRM usa en el mismo pod privado de VEC el grafo local de Granada, fijado por
imagen y huella, sin publicar otro puerto ni usar servicios externos. La
aplicación y el sidecar se reiniciaron sin tocar PostgreSQL; desde el contenedor
VEC una ruta sintética devolvió `200` y `code: Ok`. El portal mTLS y `livez`
respondieron `200` después del cambio. `readyz` conserva el `503` previsto por
la composición de desarrollo incompleta.

La identidad sintética disponible no posee `dietas.ruta.read`: el endpoint
`POST /api/vec/dietas/road-route` respondió `403`, por lo que no se amplió su
permiso ni se presenta el recorrido web de Dietas como cerrado. El contenedor
Docker usado para la preparación quedó detenido y conservado como rollback;
el sidecar rootless es el único OSRM activo de este runtime.

## Renovación diaria de confianza de Contratación — 19 de septiembre de 2026

El corte `18bd0ebb` renueva bajo demanda la configuración diaria de confianza
de Contratación sin exigir reiniciar la aplicación a medianoche. Reutiliza el
gobierno, la raíz y las claves existentes; conserva las audiencias de escritura,
Bolsa, cuadro, detalle, incorporación y continuidad, y falla cerrado ante una
revocación, una raíz distinta o un error de publicación. No añade migraciones.

Dos revisiones independientes del mismo parche resultaron favorables: SQL y
concurrencia, e identidad/criptografía. Pruebas focales de concurrencia y una
base PostgreSQL 18.4 desechable comprobaron publicación única, adopción,
rollback y revocaciones. La campaña Go final dejó `go vet ./...` y compilación
verdes; el único fallo global fue un test de inventario ejecutado inicialmente
sin metadatos Git, y su paquete pasó después con el entorno correcto.

Binario instalado con SHA-256
`edff82d89425a0c6e914ece5618e664cc3decfa175009073f42a6517f3324f15`.
Tras reiniciar solo la aplicación, el portal mTLS y dos consultas de cuadro
respondieron `200`, con 71 expedientes, sin errores JavaScript ni desbordamiento
en 1440/390. Las 130 tablas comparadas son idénticas antes y después. Falta la
observación natural del primer cruce de medianoche; las pruebas con reloj
controlado cubren ese límite sin alterar el reloj ni la base conservada.

## Apertura y lectura de expedientes — 19 de septiembre de 2026

Abrir un expediente sitúa el foco en su cabecera; conserva error/denegación
si la consulta falla y descarta respuestas de selecciones anteriores.
La carga automática de Cobertura ya no roba ese foco; el reintento explícito
conserva su aviso. Comprobado en PC 1440 y móvil 390 con consultas reales.
El raíl completa Obtención del candidato cuando el historial acredita propuesta
posterior a aceptación; no adelanta firma, incorporación ni seguimiento.
Datos de fase con texto neutro y raíl móvil de ocho pasos compacto (312,5 px);
sin cambio de disposición de escritorio ni desbordamiento a 320/390.
53 pruebas focales correctas y comprobación navegador sin errores JavaScript.
No se registraron efectos de negocio ni se cambiaron APIs, SQL o recibos.
El contador sigue 16/19; retoques menores posteriores a la presentación.
PC tiene prioridad. Bolsa incluye externos: persona no implica empleado.

## Navegación contextual de Contratación — 19 de septiembre de 2026

El corte `53f108d6` deja de anunciar Documentos y Auditoría cuando el adaptador
HTTP no compone esas consultas. La barra, los atajos de incidencia y el panel
de fase usan la misma disponibilidad, derivada de capacidad concedida y método
real; la presentación que sí aporta ambos conserva sus accesos. No se añadieron
rutas, permisos, SQL ni efectos.

La revisión independiente quedó en `GO`: 51 pruebas focales correctas. En el
runtime privado, bandeja, detalle y estadísticas respondieron `200` a 1440 y
390 px, sin denegación engañosa, errores JavaScript ni desbordamiento global.
La validación fue solo de lectura y no ejecutó descargas ni mutaciones.

## Estudio integral y extensibilidad — 19 de septiembre de 2026

Actualizados el [estudio integral](docs/estudio_requisitos/analisis_integral_rrhh.md#15-síntesis-vigente-para-los-agentes)
y el [contrato modular](docs/portal_vec/contrato_modulos_vec.md#criterio-vigente-de-ampliación--19-de-septiembre-de-2026),
enlazados desde las especificaciones obligatorias. Incluyen ficha personal de
22 bloques, autoridades y relaciones, Formación futura, servicios/trienios,
cotización, pagos de Dietas y exposición por módulo. Registro tipo plugin sobre
lo existente; no nueva implementación ni otra ficha maestra. Revisión de fuentes
y código en da708d63, sin nueva validación normativa ni pruebas de runtime.
No aumenta el contador de pantallas. Contratación mantiene prioridad y los
manuales definitivos esperan conformidad de RRHH.

El historial general y el panel de cada fase muestran fecha y hora reales de
las actuaciones en horario peninsular, conservando las fechas civiles de los
períodos. Comprobados dos hitos reales en navegador a1440/390, sin erroresJS
ni desbordamiento; 22 pruebas focales correctas. No se calculan plazos legales.

## Análisis y rectificación recorridos — 19 de septiembre de 2026

El contador funcional se mantiene en **15/19 (79 %)**. Análisis incluye la
rectificación existente, pero conserva una reserva histórica sin identidad
funcional separada. Siguen parciales Llamamiento y Resultado, y pendiente el
circuito de Firma; no equivale a producción.

La corrección 2be6b461, publicada y servida, permite cargar las 151 categorías.
En navegador se verifica la equivalencia de jornada a 1440/390 px.
La rectificación precarga también las observaciones autorizadas del detalle y
las conserva si RRHH cambia otro campo; ausencia sigue siendo válida. Textos
de análisis simplificados, sin alterar contratos HTTP ni autorización.
Dirección revisó el diff; 61 pruebas focales de contrato, adaptación y montaje
correctas, incluida conservación de observaciones hasta el POST.

Caso sintético 2026/CT-8c17ba0b2be0fa7d84131e1dc93db150: otra identidad RRHH
rectifica v2→v3 desde navegador, HTTP201, recibo
rec_ct_an_f4dc768baead35e3e98902cde64646d9, confirmado
2026-09-19T13:41:39.987015Z. Tras reiniciar los mismos app/PostgreSQL, las
130 tablas comprobadas permanecen idénticas. El detalle devuelve200, tres
hitos con los dos anteriores idénticos y las observaciones nuevas; existe
una sola actuación v3 con ese recibo. No se volvió a ejecutar la operación.
Formulario recuperado comprobado a 1440/390, sin errores JS ni desbordamiento.

También recuperadas incorporación, seguimiento original y cierre existente;
ficha GINPIX de 1969 bytes idéntica a la huella previamente conservada.
No hay transmisión GINPIX, envío corporativo ni firma acreditados. El motivo
de rectificación está publicado para el ejercicio sintético; el catálogo
corporativo y la renovación diaria duradera de confianza siguen pendientes.

## Recuperación del catálogo de Análisis — 19 de septiembre de 2026

La configuración real contiene 151 categorías. Contrato y formulario admiten
hasta 1000 categorías, conservando los límites de las demás listas y la
validación de cada entrada. Corrige el rechazo completo del formulario por
el antiguo máximo de 100. Revisión independiente favorable y 41 pruebas
focales correctas, incluido montaje y limpieza del listener de jornada.
Comprobación en navegador tras desplegar pendiente; no cierra rectificación.

## Conservación del trabajo Go revisado — 19 de septiembre de 2026

Integradas las extracciones previas de bootstrap y seguimiento y sus pruebas:
27 archivos conservados, sin cambios de comportamiento según comparación de
funciones y dos revisiones sensibles independientes del conjunto de autoridad.
Se mantienen SQL, contratos, claves, serialización y recibos; C23 queda excluido.
Pruebas focales y compilación correctas. La campaña global sólo falló inicialmente
en el inventario Git de nueve casos de cobertura porque la candidata era un
archivo sin .git; ese paquete pasa con metadatos Git de solo lectura. Vet global
sin diagnósticos. No se ejecutó integración PostgreSQL ni se instaló otro binario.
Este mantenimiento no añade pantallas ni capacidades de producto.

Bandeja actual comprobada en navegador autenticado: 71 filas, consultas 200,
sin errores JavaScript ni desbordamiento a 1440/390. Análisis tiene un defecto
detectado en vivo: su catálogo devuelve 151 categorías y la web limita a 100;
corrección acotada en curso. No se da por comprobada la nueva ayuda de jornada
en el formulario real hasta resolver ese montaje.

## Jornada comprensible en Análisis — 19 de septiembre de 2026

El campo conserva su entero canónico, pero muestra al escribir la equivalencia
porcentual localizada: 5000 → 50,00 % y 10000 → 100,00 %. Los valores inválidos
limpian esa ayuda; el DTO y la validación no cambian. Usa i18n y anuncio accesible,
sin perder el foco. Revisión de dirección y 18 pruebas focales correctas.
No cierra la recuperación histórica pendiente de Análisis.

## Recuperación de incorporación y disponibilidad — 19 de septiembre de 2026

La vista RRHH recibe explícitamente el cliente de continuidad: puede recuperar
la incorporación existente cuando el catálogo de alta responde pero falla la
configuración de análisis. Intervención conserva su entrada de fiscalización.
Sin API nueva, POST adicional ni cambio de autorización. Revisión independiente
favorable; 28 pruebas focales correctas, incluida la combinación real de clientes.

La bandeja devolvía 503 porque la confianza de desarrollo del proceso anterior
caducó a medianoche UTC. Tras copia privada y reinicio de la misma aplicación:
consultas autorizadas de límites 1/5/100 responden 200, con 1/5/71 expedientes.
De 130 tablas CT/Personal/Bolsa, 122 conservan exactamente su contenido; las ocho
variaciones corresponden a auditoría y cursores de las tres consultas. No se
reaplicaron migraciones ni se reconstruyeron operaciones o recibos. La confianza
renovada caduca de nuevo a medianoche: pendiente renovación duradera con revisión
sensible. Esto recupera disponibilidad, no cierra pantallas ni producción.

## Corte funcional de interfaz — 19 de septiembre de 2026

Corte 259c75cc integrado, publicado en main y servido en el servidor sobre d7982bb4:
el resumen de llamamiento muestra las fechas confirmadas; el raíl respeta las
fases acreditadas y los colores RRHH; la consulta de una fase conserva foco,
encabezados de tabla y las traducciones del montaje. El detalle distingue la
firma pendiente de los documentos preparatorios disponibles, sin atribuir envío
al portafirmas. Se mantienen las seis descargas y sus contratos.

Comprobaciones focales del candidato: 96 casos de llamamiento/documentos/fases,
19 del adaptador, 3 de fases tras la conexión i18n y 24 de presentación. Son
grupos parcialmente solapados, no una cifra de pantallas ni pruebas de producción.
Revisión de código cerrada; comprobación visual sobre assets desplegados pendiente.
El contador histórico 15/19 no aumenta por esta mejora de pantallas existentes.

Se conservan separados el WIP Go, la portada y C23. La ficha manual GINPIX,
anotaciones y cierre disponen de conexiones existentes: no se reimplementan.
El manual distingue esos recorridos del portafirmas, entrega de correo, plazos
legales y transmisión GINPIX pendientes. No se han aplicado migraciones.


## Hoja de ruta vigente — 16 de septiembre de 2026

Alineada con los tres documentos que fijan lo que se pide: el **Word de RRHH**
(«Pantalla de procedimiento de gestión de contratación y gestión de bolsas», 8
fases), **Peticion.pdf** (Servicio de selección externa: bolsas, estados,
llamamientos, portal, cuadro de control, correos, plantillas, trazabilidad) y el
**pliego SE 15/2020** (`docs/convoca_dipgra/`, lo que CONVOCA hace hoy en la
Diputación). Sustituye al «Plan de ataque funcional» de julio, que queda abajo
como histórico. Las reglas y el orden detallado están en
`INSTRUCCIONES_DESATASCO.md`; el consenso técnico, en `comunicacion.md`.

| Fase | Entregable | Fuente | Estado | Cuándo está terminado |
| --- | --- | --- | --- | --- |
| 0 | **Contratación temporal presentable**: bandeja que pagina, log en todo 5xx, raíl de 8 fases con 5 estados, número visible, observaciones, nombres de catálogo, coste con fuente o «sin calcular», correo de llamamiento en buzón de prueba, guion de demo. | Word, pasos 1–8 | En curso (orden de correcciones 1–8) | RRHH recorre los 8 pasos en el navegador con datos sintéticos sin ver errores ni restos de desarrollo. |
| 1 | **Contratación temporal útil**: composición de producción, identidad y roles reales, portafirmas, SMTP corporativo, RC y coste, GINPIX, instalación en intranet, copias. | Word; `dudas.md` 1–12 y 15 | Pendiente; depende de RRHH, Sistemas e Informática | Un expediente real recorre las 8 fases con efectos administrativos y aceptación formal de RRHH. |
| 2 | **Bolsa, gestión** (B1–B14 de la ficha): importador de CONVOCA invocable, estados y pantalla, orden según reglamento, histórico de contactos, llamamiento real, pausas y bajas, consulta pública, portal del candidato con DNIe/certificado, cuadro de control, avisos. | Pliego §1 c); Peticion.pdf; `dudas.md` 13–18 | Aparcada hasta cerrar la fase 0 | Las bolsas reales importadas de CONVOCA se gestionan en VEC y Contratación llama desde ellas. |
| 3 | **Bolsa, proceso selectivo** (S1–S7): alta y publicación, inscripción con Sede y pasarela, autobaremación, validación y subsanación, resultados de pruebas, baremación y listado definitivo. | Pliego §1 a) | Aparcada; código escrito sin componer | Una convocatoria nueva se tramita de principio a fin en VEC y CONVOCA deja de usarse. |
| 4 | **Cronos** reescrito (sustituye a la aplicación PHP 5). | Aplicación actual + normativa de jornada | Aparcado; requiere ficha de requisitos aprobada | Ficha aprobada y recorrido de fichaje, jornada y permisos en navegador. |
| 5 | **Dietas** nuevo, con los puntos de vista del empleado, del jefe que autoriza y de RRHH (sustituye a la aplicación PHP 5). | Aplicación actual + cuantías vigentes | Aparcado; requiere ficha de requisitos aprobada | Ficha aprobada y recorrido solicitud → autorización → liquidación en navegador. |

Métrica única que se reporta: «pasos del flujo de RRHH recorribles de extremo a
extremo por un humano» (fase 0) y, después, la fila de esta tabla que se cierra
con su criterio comprobado. Cronos, Dietas y Bolsa no se reportan como avance
mientras estén aparcados. Lo que sigue en este fichero es historial de cortes.


### Continuidad real tras subsanar y fiscalizar de nuevo

El expediente sintético `9511d16d…` continúa desde la fiscalización favorable con
observaciones `v8`: selección `200`, aviso local `201`, declaración `201`, aceptación
manual `201` y propuesta `201`, versión `9`. La propuesta es
`propuesta:a26304cb-8f29-46b8-beaf-82b5cf9a66a9`, recibo
`recibo:c0d83684-ae43-44a2-baa0-857d1b5cb750`, fecha
`2026-09-13T07:43:15.57655Z`. Se conservan las claves y peticiones exactas fuera de Git.
Recuperación confirmada tras reiniciar los mismos contenedores de aplicación y
PostgreSQL: cinco respuestas `200`, mismos cinco recibos, referencias, fechas y
versiones; solo cambian los estados de replay previstos. Los 122 objetos de
Contratación, Personal y Bolsa permanecen idénticos durante el reinicio. Tras las
consultas, los 117 objetos de negocio siguen iguales; únicamente cambian las cinco
auditorías de lectura esperadas. No hay nuevas operaciones ni duplicados.

La consulta autorizada recupera el detalle `v9` con nueve actuaciones y ofrece los
seis pares PDF/DOCX. El informe DOCX de esta nueva propuesta tiene 3379 bytes y
SHA256 `45fdd2b3543aee4d6b9d58aa7b8b425ea359cf01c488165f744c972202081611`;
el documento identifica versión y actuación de propuesta `9`. No se han repetido
los documentos del caso original. Navegador a 1440/390 px sin errores JavaScript
ni desbordamiento. El aviso es local y la aceptación es manual sintética: sin
SMTP, entrega externa, plazo legal, firma ni nombramiento eficaz acreditados.

El contador permanece en **15/19**. La composición de Rectificación de Análisis
está publicada en `f0a0b7db`, con dos revisiones independientes y comprobación focal
de publicación y vigencia del motivo antes del commit. No está activada: falta el
paquete privado de motivos publicado y acreditar un caso con actor distinto al del
análisis anterior. La lectura de historia encuentra siete candidatos sintéticos v2
con autor distinto; todavía no se ha ejecutado su rectificación. No se han creado
identidades ni relajado la separación de actores. CT93–96 siguen instaladas una
sola vez; no reaplicar migraciones.


Traslado queda recorrido y el contador pasa a **15/19 pantallas terminadas en
desarrollo (79 %), tres parciales y una pendiente de pantalla**. Desde el acceso
manual existente se recuperan selección, comunicación, declaración, aceptación y
propuesta con cinco respuestas `200` y las claves originales. El enlace de revisión
conserva la navegación y enfoca el título de propuesta, también fuera de los datos
registrados plegados; corrección publicada en `8bdf550a`, con 73 pruebas focales.

La propuesta conserva `propuesta:2dd1c999-44c3-4fdc-b68e-e0adde592c81`, el recibo
`recibo:3335969d-3bb5-4258-afcc-1af26b7f7207`, versión `7` y fecha
`2026-09-06T01:28:30.697897Z`, con estado `replay_confirmado`. Cuatro capturas a
1440/390 px acreditan aceptación y propuesta sin errores JavaScript ni desbordamiento.
No se creó otra propuesta ni se reiniciaron servicios en este corte. No acredita
traslado externo, envío de correo, firma, plazo legal ni nombramiento eficaz.
Resultado conserva pendiente el vencimiento con inicio y plazo gobernados; Análisis
mantiene la rectificación sin catálogo admitido y Firma espera su circuito.

### Antecedentes del corte de catorce pantallas

Cobertura queda recorrida y el contador pasa a **14/19 pantallas terminadas en
desarrollo (74 %), cuatro parciales y una pendiente de pantalla**. La consulta
sintética del período exacto del expediente existente responde200; RRHH confirma
Bolsa vigente mediante una única decisión201 v2→v3. La consulta de resultado200
recupera los mismos seis campos del recibo y el detalle200 ofrece asignación.
Sin errores JS, almacenamiento web ni desbordamiento a1440/390. No representa
una comprobación corporativa: la fuente de desarrollo declara solo ese período,
sin ampliar rangos ni modificar SQL, permisos o evaluadores.

Análisis normal también registra201 sobre una solicitud existente del centro de
desarrollo: v1→v2, cinco modalidades disponibles, fechas originales enero-marzo2027,
RC validada y dos actuaciones consultadas después con200. No se ha creado otra
solicitud. La fila3 conserva su parte pendiente de rectificación sin catálogo de
motivos; no se habilita mediante una elección inventada. La cabecera distingue
«Período solicitado» y «Período analizado por RRHH», con datos autorizados y
traducciones; el caso de cobertura conserva ambos períodos diferentes.

### Antecedentes del corte de trece pantallas

Corte comprobado del 13 de septiembre: **13/19 pantallas terminadas en desarrollo
(68 %), cinco parciales y una pendiente de pantalla**. Se cierran Inicio, Unidad,
fiscalización, documentación, resumen GINPIX y generación documental mediante
sus recorridos de navegador; no se cuentan commits ni conexiones externas.

La bandeja recupera 52 filas autorizadas y los filtros vacío/completados responden
200 sin acciones de continuación ajenas. El ajuste `253c4d63` permite leer los
identificadores largos a 390 px y conserva el scroll interno de la tabla.
Unidad registra `201` v3 → v4 y se reabre por consulta `200` con la unidad confirmada,
sin otra asignación y con el informe habilitado. La fiscalización v7 → v8 conserva
el mismo recibo tras reinicio, según la evidencia detallada a continuación.
Documentos muestra seis pares PDF/DOCX y el estado de cada descarga; un DOCX real
conserva su huella. Cancelación y reintento pasan en navegador con transporte
retenido de forma controlada, sin enviar ese PDF al servidor. GINPIX recupera el
recibo original y el seguimiento, separando ficha manual y envío externo pendiente.
Todo lo inspeccionado en 1440/390 px queda sin errores JS ni desbordamiento.

Siguen pendientes Análisis, Cobertura, Llamamiento, Resultado y Traslado; Firma
carece de circuito admitido. El 503 de Cobertura está localizado en la consulta
de la fuente sintética para el período del caso existente; todavía no está corregido
en runtime. CT94–96 están instaladas, pero este corte no declara recorrida su
cadena posterior v8. La fiscalización manual informa que no ha consultado
antecedentes: no infiere un informe ni una fase de la versión remitida.

CT93–96 están instaladas una sola vez en la principal. La comprobación final
reconcilió únicamente la grafía `TimeZone=UTC` del catálogo PostgreSQL, sin DDL
adicional; los siete cuerpos SQL, permisos e historia quedaron comprobados.
El binario de continuidad y las mejoras web hasta `93e77d1d` están activos.

Intervención registró una fiscalización favorable con observaciones sintéticas:
`201`, v7 → v8, fase `fiscalizacion`, estado `en_curso`, el 13 de septiembre
de 2026 a las 06:38:42.032945 UTC. Tras respaldar y reiniciar la misma aplicación
y PostgreSQL, recuperó con la petición y clave originales el mismo recibo de
12 campos (`201`). La consulta RRHH respondió `200` con ocho actuaciones.
Los 118 objetos de CT/Personal se conservaron en el reinicio; las consultas
posteriores solo cambiaron cinco objetos de auditoría de acceso. Sin duplicados.

El navegador se comprobó a 1440/390 px, sin errores JS, cookies, almacenamiento
web ni desbordamiento. El acceso manual de Intervención no consulta antecedentes:
su corrección de presentación evita deducir fase o informe de la versión escrita.
No acredita firma, vencimiento legal, correo ni transmisión externa. El contador
se conserva hasta cerrar la revisión visual del ajuste y de las páginas restantes.

### Antecedentes de preparación del corte

La fiscalización conserva su recibo confirmado mientras refresca el detalle y
recompone la continuación ofrecida por el servidor. Si falla la consulta o llega
una versión distinta, mantiene el recibo y avisa de la actualización pendiente;
no repite la fiscalización. Las 16 pruebas focales del montaje y las recuperaciones
son satisfactorias. El recorrido principal posterior a subsanación sigue pendiente
de instalar CT93–96 y activar el código; este cambio no aumenta el contador.

El análisis exige las cinco modalidades del contrato RRHH vigente también al
montar el formulario. La vista Documentos agrupa los seis borradores PDF/DOCX,
comprueba que su índice autorizado corresponde al expediente y versión actuales
y permite cancelar la descarga. Conserva el canal de consulta y los límites de
borrador de desarrollo. Las 45 pruebas web focales pasan; la activación y el
recorrido real de estas mejoras están pendientes. El contador continúa en 7/19.

La primera tanda de páginas conecta la bandeja a los expedientes de la página
autorizada, conserva el recibo de asignación mientras recupera el detalle y ofrece
el informe solo cuando coinciden la unidad y la versión confirmadas. La candidatura
aceptada agrupa sus antecedentes y enlaza con la propuesta existente. Los textos
usan el catálogo común; no se registran tareas propias ni efectos adicionales.

La integración supera 32 pruebas web focales, incluida la página sin expedientes
pendientes y las recuperaciones de asignación y propuesta. Estos cambios de interfaz
están preparados para activación y recorrido real; mantienen el contador 7/19.

La pantalla de subsanación queda terminada en desarrollo: registro y recuperación
tras reinicio con el mismo recibo, más reapertura del expediente v7 mostrando
«Subsanación registrada» sin otro formulario de envío. El recibo recién registrado
se conserva al actualizar el detalle; un reparo posterior puede ofrecer de nuevo
la corrección. El contador de pantallas pasa a 7/19 (37 %), con 16 superficies
visibles (84 %); no equivale a cerrar la fiscalización posterior ni pasos legales.

La comprobación final usó únicamente consultas: detalle 200, siete actuaciones,
cero formularios de subsanación y estado registrado visible. En 1440/390 no hubo
desbordamiento, errores JavaScript, cookies ni almacenamiento web. El runtime
necesitó además los dos metadatos del historial ya integrados en `f081df9d`;
se activaron sus assets de adaptador/contrato, sin SQL ni otro registro.
Once pruebas web del corte, tres casos de renderizado sobre la base runtime y
ocho pruebas del adaptador fueron satisfactorias. Los assets del estado registrado
proceden de `770b72ea` y conservan la base compatible con CT92.

El ensayo aislado de CT93→CT96 terminó satisfactoriamente con DDL, ACL, rechazos,
historia y reinicio del clon; el clon quedó detenido y conservado. La instalación
principal sigue pendiente de su instalador revisado. No se ha registrado una
nueva fiscalización ni una propuesta posterior en la base principal.

La fuente publicada `1362affd` compila correctamente. Su binario y once assets
web están preparados, sin activar: la aplicación conserva el binario compatible
con CT92. El acceso actual de Intervención todavía limita la versión remitida a
v5; la fuente integrada permite la versión posterior. La comprobación de lectura
no envió ninguna fiscalización. Activar la continuidad exige antes instalar
CT93–96 con el procedimiento revisado y un respaldo privado fresco.

## Antecedentes de los cortes

Los apartados siguientes conservan las entregas y los límites de cada corte
anterior. Las cifras, instalaciones y comprobaciones vigentes son las indicadas
al comienzo de este documento; los pendientes antiguos no los sustituyen.

Los seis borradores conservan una propuesta posterior a la subsanación cuando
el detalle autorizado añade su resolución y la anotación administrativa siguiente.
La consulta solicita la versión actual y el renderer identifica la propuesta por
la cadena de actuaciones: por ejemplo, propuesta v9 desde detalle v10 o v11.
El documento indica la versión consultada; no promete bytes idénticos al emitido
en v9. La recuperación original de propuesta v7 conserva su fachada existente.

Los seis archivos coinciden con la entrega revisada `563c7aa4`; pasaron 18 pruebas
web y las pruebas focales y vet del renderer. No añade SQL ni permisos. El recorrido
de una nueva propuesta persistida continúa pendiente de CT93–96; no se atribuye
otra descarga real a esta integración.

Subsanación demostrada el 13 de septiembre de 2026: formulario real, registro
HTTP 201 y expediente `9511d16d…` de v6 a v7, con siete actuaciones y recibo visible.
Se conservan `subsanacion_unidad` e `incidencia`; la corrección no aprueba la
fiscalización. Tras reiniciar la misma aplicación y PostgreSQL, la petición
original recuperó los once campos del mismo recibo y fecha, sin duplicados.
El contrato HTTP actual devuelve 201 también al recuperar esa operación.

Antes y después del reinicio coincidieron 118 tablas/secuencias de Contratación
y Personal. Tras las nuevas consultas de navegador, permanecieron idénticos los
113 objetos de negocio y una sola reserva de subsanación; se añadieron tres
registros de acceso auditados y dos alcances. En 1440/390 no hubo desbordamiento
ni errores JavaScript. El primer arnés de recuperación falló al leer `status`;
se conservó su intento y se corrigió sin cambiar la petición ni su clave.

AD3-38 y CT92 están instaladas con historia: no reaplicar UP ni DOWN. La causa
de indisponibilidad era la clave del motivo: el contrato PostgreSQL exige el
perfil V2. La configuración corregida obtuvo dos revisiones y validación SQL
de solo lectura antes de publicarse por el arranque existente. El cargador
valida ahora ese perfil y registra únicamente etapas y códigos SQLSTATE acotados.
Código integrado en `cd1861f0`; pruebas focales y vet de bootstrap satisfactorios.
El binario activo `31c8aa9b…` conserva la base funcional `6f36d8e5` con este parche,
y su configuración devuelve 200 con subsanación disponible.

CT93–96 están integradas como fuente revisada, pendientes de ensayo e instalación
para continuar la fiscalización y las operaciones posteriores. El contador
mantiene 6/19 pantallas terminadas: el circuito completo de reparos sigue parcial.

Tras registrar una propuesta, «Ver expediente actualizado» recupera el detalle
mediante las consultas autorizadas existentes. Si falla la lectura, conserva el
recibo y permite reintentar sin repetir la propuesta. La prueba focal reproduce
el avance del cuadro, el primer fallo del detalle y su recuperación posterior;
el formulario y esa regresión suman 72 pruebas web satisfactorias.

AD3-38 y CT92 quedaron instaladas una sola vez el 13 de septiembre, con respaldo
privado y comparación de historia, ACL y roles. El intento previo se detuvo antes
del respaldo y del DDL por una tabla mal nombrada en el capturador; se conservó y
se reanudó con dos revisiones del parche. No reaplicar estas migraciones.
El runtime utiliza el corte de subsanación 6f36d8e5; su formulario sigue pendiente
de habilitación efectiva: configuración y detalle responden 200, sin nuevo POST.
CT93–96 continúan pendientes de ensayo e instalación. El contador no aumenta.

La propuesta de formalización admite la versión fiscalizada posterior a una
subsanación y conserva ese antecedente al recuperar el mismo recibo. El formulario
usa la versión de la selección confirmada; no sustituye las versiones propias
de la resolución de llamamiento o de Bolsa. Las operaciones originales v6→v7
conservan su función SQL.

La fuente CT96 y sus 13 archivos Go/web coinciden íntegramente con la entrega
539247a6 revisada por dos revisores. Pasaron 98 pruebas web de formulario y
cliente y la campaña conjunta `go test ./...` y `go vet ./...`.
CT96 permanece sin ensayo PostgreSQL ni instalación; el recorrido
N8→propuesta9 está probado con repositorios dobles, no acredita una propuesta
nueva en la base principal. Siguen pendientes la continuación tras renuncia
con versiones posteriores y la lectura histórica de esas nuevas propuestas.

Aviso, declaración de respuesta y consulta del justificante pueden recuperar
la versión fiscalizada que originó su selección confirmada. El lector deriva
ese vínculo del llamamiento persistido y conserva autorización nueva en cada
operación; la versión no se toma de la cabecera actual ni de un campo del usuario.

CT95 y su código mantienen la entrega con dos revisiones favorables. Pruebas
Go focales y vet del ámbito afectados completados. No está instalada CT95;
el binario que la utiliza no debe desplegarse antes de su ensayo e instalación,
pues el nuevo lector también atiende operaciones del recorrido original.
Este corte no acredita envío de correo ni otro registro real de comunicación.

Los seis borradores pueden representar una propuesta actual posterior a una
subsanación, usando el hito y la versión reales del detalle autorizado. Se
conserva la recuperación de la propuesta original v7 desde resolución v8 o
anotación v9. No se habilita la lectura histórica de propuestas posteriores
a través de versiones nuevas sin su circuito autorizado.

La integración mantiene el cliente y el canal de consulta existentes. Pasaron
27 pruebas web y las pruebas focales del renderer. La lectura desde una nueva
propuesta persistida sigue pendiente; las descargas reales ya acreditadas de
la propuesta original permanecen como evidencia del corte anterior.

La selección del llamamiento se prepara ahora con la versión CT fiscalizada
vigente, conservando la versión independiente de Bolsa y las funciones originales
para v6. CT94 añade lectores para versiones posteriores; permanece pendiente
de ensayo PostgreSQL e instalación, y no se ha abierto otro llamamiento real.

La prueba de montaje confirma que un resultado favorable v8 rellena la selección
con v8 sin ejecutar selección, comunicación ni respuesta. Pasaron 97 pruebas
web del recorrido existente y 10 de fiscalización, las pruebas Go focales, vet
y compilación. La comunicación y la propuesta posteriores a nuevas versiones
conservan sus dependencias; esta integración no acredita ese circuito completo.

La continuidad de fiscalización tras subsanar reutiliza el mismo formulario,
API y permiso de Intervención. Exige la corrección ligada al retorno vigente,
conserva las instantáneas anteriores y permite registrar un nuevo resultado;
no lo presupone favorable. La fuente CT93 está revisada, pero su ensayo
PostgreSQL e instalación todavía no se han realizado. La primera fiscalización
conserva sus funciones v1 y el circuito original.

Go, SQL y formulario coinciden con la entrega revisada del apoyo. En integración
se mantuvieron las guardas actuales de la vista y se actualizó una prueba que
omitía la asignación real exigida antes de preparar informe. Los 33 casos web
focales quedaron verdes tras ese ajuste; pruebas Go, vet del ámbito y compilación
completados. Esta evidencia no acredita una refiscalización de navegador ni
la continuación posterior del llamamiento desde nuevas versiones.

Subsanación de reparos queda compuesta como capacidad opcional: formulario,
cliente y `POST /api/vec/contratacion-temporal/subsanacion-reparos`, sobre el
retorno existente. El recibo confirmado se conserva mientras se recuperan
el detalle y su historial; la corrección mantiene la incidencia y no acredita
una nueva fiscalización favorable. La disponibilidad procede del servidor.

CT92 y AD3-38 conservan las huellas con dos revisiones favorables. Su ensayo
PostgreSQL e instalación principal siguen pendientes en este corte; no reaplicar
migraciones históricas. La política privada se configura con
`VEC_CT_SUBSANACION_POLITICA_FILE`, sin actor ni perfil suministrados por la web.
El catálogo opcional de rectificación se conecta mediante
`VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH`: exige publicación vigente;
sin fuente mantiene la indisponibilidad. No se han aprobado motivos legales.

Comprobaciones de integración: 31 pruebas web focales, pruebas Go de los
paquetes afectados, vet del ámbito CT/composición y compilación de aplicación.
Revisión independiente de montaje favorable. No acredita todavía un registro
de subsanación desde navegador. Cobertura continúa en `503`; el candidato de
concurrencia se descartó de producto tras medir el mismo fallo en 626 ms.

El detalle muestra ocho fases del procedimiento y el historial de actuaciones
ya registrado. Solo destaca la fase actual; las demás permanecen **Sin confirmar**.
El rail enlaza la fuente RRHH y el flujo declarado mediante sus huellas, sin
crear estados ni dar por completadas fases anteriores.

La comprobación real de Unidad y del expediente `fe493…v9` pasó a 1440 y
390 px, sin desbordamiento ni errores JavaScript. Las consultas respondieron
`200`: Gestión de bolsa y Nombramiento fueron sus respectivas fases activas;
el historial abierto mostró 3 y 9 actuaciones. El CSS móvil corrige los mínimos
de las cuadrículas y del formulario, conservando el desplazamiento interno del
rail. No hubo nuevas escrituras de negocio ni descargas documentales en esta
comprobación; tampoco se mostró un recibo que permitiera volver a cotejarlo.

Propuesta publicada en `1ce29f875df764f47b9144ab55ee96c27c68109f` y rail en
`da17727bfbdbb4d74afd6f2a57bf2c418da58fdf`. El binario de esta prueba fue
`a1a721201edccfa9711bb145f18377d8bc0cfaabda6b880b790d764b26c9a009`;
se conservaron la misma aplicación y PostgreSQL, sin reinicio de la base.
El resumen de aceptación reutiliza CT65; asset disponible, recorrido completo
pendiente. Las revisiones y pruebas focales del rail, propuesta e historial
pasaron. El ensayo real descrito no repitió operaciones de incorporación,
anotación o cierre. El contador conserva 6 de 19 pantallas terminadas en
el ejercicio de desarrollo; estos commits no incrementan esa cifra.

La apertura de formularios exige ahora la asignación que proyecta el detalle:
el caso Unidad v3 presenta Asignación y no ofrece todavía Informe jurídico.
La prueba real confirmó esa secuencia y la legibilidad del aviso de rectificación
a 390 px, sin escrituras de negocio. Con el catálogo actual la interfaz indica
«Rectificación no disponible» y no monta controles de envío.

El diagnóstico revisado identifica la indisponibilidad de cobertura en
`cobertura.presentacion.preparador`. El binario observado tras reiniciar solo
la aplicación tiene SHA256 `783c051a6ee1d81fb4925aed99615dbe3cabbc39ee3ed53c316e2620de9ea66f`.
No registra causas privadas ni cambia el 503 público; la corrección del
preparador sigue pendiente de comprobación real.

El detalle reúne ahora el resumen final para GINPIX: centro y categoría del expediente consultado, periodo, fecha y recibo original de incorporación. La ficha de carga manual se descarga por GET 200 con la misma huella `4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`; el seguimiento responde 200. El envío externo figura como no conectado. Comprobado a 1440/390 px sin errores JavaScript ni desbordamiento. Los indicadores del cuadro precisan que cuentan los expedientes de la página actual.

La búsqueda del cuadro indica su criterio real: inicio del número de expediente. El botón de cierre amplía su área a 44 × 44 px. La lectura nominal de la bandeja conserva las mismas 52 referencias y fases, sin errores JavaScript ni desbordamiento en 1440/390 px.

El corte D conecta la lectura de la propuesta original v7 para reutilizar sus
seis borradores desde un expediente que ya está en v9. CT91, SHA256
`8759adf01dfe2d14a583ee051527367423ff5fe6ad147953be2b0dc3a0181a34`, está
instalada una sola vez y obtuvo dos `GO`. El catálogo pasó un ensayo PostgreSQL
DDL en `ROLLBACK`, su comprobación instalada terminó en `PASS` y las pruebas
focales Go de HTTP, adaptador PostgreSQL y composición pasaron. El runtime
descargó dos representaciones del informe definitivo: PDF de 29280 bytes y SHA256
`f6bd9fd9f61dc266621e9a72ab4ec0f5e24f0c55073a1de403d59cd7d9531810`.
El DOCX es OOXML válido, tiene 3378 bytes y SHA256
`7141cbc60e586086494cb4bc609ef9748c6f09675cb1f172ae4649f3f1944577`.
Dos lecturas de cuadro, detalle y PDF dieron cuatro `POST 200`; incorporación y
preparación de cierre dieron `GET 200`, y el DOCX respondió `200`. Es una prueba
representativa de un PDF y un DOCX entre seis borradores, no doce descargas. La
comparación de solo lectura conservó sin cambios la historia CT en 12 tablas,
su estado y las cinco tablas de Personal, sin escrituras de negocio. No hubo errores
JavaScript ni desbordamiento a 1440/390 px; los dos `404` heredados de Bolsa no
pertenecen al corte. El parser admite `menu: null` sin habilitar acciones y las
siete claves locales canónicas están desplegadas. No reaplique CT91. El binario
compilado desacopla la preparación E06 de sus proveedores y restaura la
publicación de los dos motivos del catálogo de llamamiento. Conserva los
auxiliares de correo, cuyo circuito sigue pendiente;
para arrancarlo no se instaló AD3-32 ni se cambiaron permisos.

Corrección de recuperación de Cobertura: el análisis registrado conserva la fase Solicitud. La vista exige resultado RC, la misma referencia y versión en cuadro/detalle, y ausencia de cobertura o asignación. El expediente existente `b50fa719…`, v2 y RC validada, abre ahora el formulario sin repetir el análisis. Cuadro y detalle responden 200; su consulta automática de propuesta devuelve 503 y sigue en corrección del backend. Esta comprobación acredita el montaje recuperado, no una decisión de cobertura ni el cierre de la pantalla 4.

El montaje normal de RRHH recibe ahora el cliente existente de Fiscalización para continuar desde Informe jurídico. La bandeja real respondió 200 sin errores JavaScript; no contiene actualmente expedientes en esa fase, por lo que este corte acredita la conexión y sus comprobaciones focales, no un nuevo registro de fiscalización.

Bandeja y detalle muestran los nombres del catálogo de centros y categorías ya cargado; conservan las referencias originales si falta una etiqueta. Comprobación real: «Centro solicitante» y «Categoría C2», sin errores JavaScript ni desbordamiento a 1440/390 px. El 404 del lector histórico y la descarga pendiente desde v9 describen un corte anterior; el corte D posterior cerró la recuperación desde la propuesta original v7 con descargas representativas PDF y DOCX.

El detalle muestra la jornada como porcentaje (10000 → 100 %). La pantalla anterior se comprobó a 1440/390 px sin desbordamiento ni errores JavaScript y usaba dos columnas en móvil. El CSS vigente la cambia a una columna por debajo de 620 px y conserva botones de navegación de al menos 44 px; el QA actual detectó un ancho desplazable de 788 px en viewport de 390 px, por lo que el ajuste móvil sigue abierto.

Si falla la apertura de Análisis o del siguiente formulario de Cobertura, Asignación o Informe jurídico, aparece un aviso con reintento de apertura. El reintento conserva el recibo y no repite el POST confirmado; la prueba focal cubre dos fallos consecutivos sin duplicar avisos. Bandeja y detalle reales comprobados a 1440/390 px, sin errores JavaScript ni desbordamiento.

## Incorporación, GINPIX y Word comprobados en navegador — 12 de septiembre de 2026

La base de este corte es `2eb94c1c8277a7bb93393d7ed41e66580590182f` y sus
cambios son `f1512fdbe85a891d425a1d453993c512a5f420b4` y
`cfa4d70fa14a23827f8cd15420975fe5d5d82fd3`. El runtime principal comprobado
usa el binario SHA256
`bc65b1d6213cf1da31d58fd23c96acb27b4b54c8f99ef9cbc7948d30c21b4122`, el
asset SHA256 `f0fa62246a8dbeb5b3dcc8e56e68e7e3d9ace4511f1ba813619764130c63ceb4`
y el árbol web SHA256
`b3a18ba8556946dd7b33c1e9cccbfe95c4a37f6a2c219db815a5e8ab43dd1215`.
Chrome recuperó la
incorporación y la ficha GINPIX con HTTP `200`, sin repetir el `POST`.
Coincidieron los seis campos cotejados, el recibo
`ref:2bc3d281…` y la fecha `2026-09-10T13:07:06.614186Z`; la ficha descargada
tiene SHA256
`4f56c3dd607a1495852ac5e88a3b15340a50481c87a46a6fe078e3350afd4057`.

El corte B presenta en la ficha GINPIX el recibo de incorporación y el nombre
del archivo, y muestra las fechas del seguimiento en formato legible sin perder
su valor ISO. La descarga real y la consulta de seguimiento respondieron
`GET 200`; la ficha conservó el mismo SHA256 y no hubo errores JavaScript. La
revisión visual terminó sin desbordamiento a 1440 ni 390 px.

En el runtime anterior `75157434…`, los seis DOCX se descargaron con HTTP `200`
y validación ZIP correcta. El
recorrido registró cero errores JavaScript y cero datos en cookies o
almacenamiento web. La pantalla de incorporación no tuvo desbordamiento a
1440, 1024 ni 390 px. Los Word conservan su carácter de borradores: no
constituyen firma, eficacia, envío, entrega o transmisión.

La trazabilidad mantuvo visibles recibo y fecha y permitió abrir y cerrar el
detalle. No hubo errores JavaScript, cookies, almacenamiento ni desbordamiento
a 1440, 1024 o 390 px. No se repitieron las seis descargas Word. La evidencia
privada está custodiada. En aquel recorrido, la preparación devolvió `GET 404`;
la comprobación vigente aparece debajo y tampoco acredita un cierre.

La agrupación a ancho completo de los seis pares PDF/Word está desplegada y
visible. Tras el reinicio, los seis DOCX devolvieron `200` con nombres, bytes y
SHA256 idénticos a la evidencia anterior. Dirección inspeccionó 1440 y 390 px;
la incorporación mantuvo desbordamiento cero a 1440, 1024 y 390 px. El informe
de incorporación registró cero errores JavaScript, cookies y almacenamiento;
el informe Word acredita por separado cero errores JavaScript.

La comparación privada custodiada confirma cinco huellas de Personal idénticas
y seis contadores más la fila de
incorporación CT idénticos, sin `POST` de incorporación. Se conservan respaldos
privados de Auth13 de las 13:42:35 y 13:51:46 UTC. Los objetivos 11,
recuperación, y 12, ficha manual GINPIX, quedan funcionalmente acreditados tras
reinicio; no completan Contratación ni convierten los ocho hitos en completos.
AD3-30/CT86, AD3-31/CT87 y CT90 se instalaron después una sola vez en principal;
la recuperación del recorrido tras reinicio ya está acreditada.

El frontend publicado integra seis JS y dos pruebas a
partir de tres hunks revisados, sin conflictos y conservando DOCX y la
agrupación. Presenta fechas civiles UTC sin desplazar el día; consume una sola
vez cada cursor opaco con controles **Siguiente** y **Reiniciar**; mantiene el
cuadro ante errores recuperables; y aclara que el chip cuenta esta página, no
el total. El filtro admite hasta 80 caracteres y una entrada inválida muestra
un aviso accesible sin `TypeError` ni nueva consulta. Dos revisiones estáticas
dieron `GO`. La validación terminó 351/351 en Contratación temporal y 23/23 en
el coordinador, 374 comprobaciones en total; los manifiestos 11 públicos, 111
internos, 3 compartidos y 1 traducción también pasaron. No se ejecutó una nueva
campaña Go. En navegador, la página única mostró 52 filas con límite 100 y
**Siguiente** deshabilitado; el filtro válido devolvió HTTP `200` y una fila.
Una entrada de 81 caracteres mostró el aviso accesible, mantuvo esa fila y no
envió otra consulta; limpiar el filtro volvió a obtener HTTP `200`. No hubo
errores JavaScript. El detalle respondió `200` y mostró doce botones sin
pulsarlos. No se acredita avanzar con cursor porque el conjunto no supera una
página; la prueba controlada sigue cubriendo ese contrato.

También se incorporan únicamente al versionado 60 fuentes SQL históricas
exactas: manifiesto 60/60 y 30/30 scripts `UP` cotejados con el registro privado
de instalación. Dos revisiones de apoyo dieron `GO`. No se ejecutaron SQL ni
`DOWN`; CT70–85 y Auth13 conservan su historia instalada. Los cuatro avisos de
fin de fichero de originales CT70/80 se mantienen intencionadamente para
preservar igualdad byte a byte. Dos clústeres aislados se restauraron con
`PASS`. Los ensayos posteriores de CT86/87 y la instalación principal de CT90
se detallan debajo.

La recuperación del cierre administrativo está integrada en nueve archivos de
producto y pruebas. Los cinco de recuperación recibieron dos revisiones `GO`;
los otros cuatro, de trazabilidad, tuvieron revisión proporcional de dirección.
**Preparar cierre** crea y muestra una solicitud inmutable y ofrece guardar su
JSON sin ejecutar el `POST`; **Guardar datos de recuperación** permite conservar
el archivo antes de confirmar. Solo la segunda acción, confirmada expresamente,
envía el cierre. La importación coteja expediente y seguimiento contra el
recibo obtenido por el `GET` original. Recuperar o completar reutiliza la
solicitud exacta y puede concluir una operación pendiente, sin repetir el
`POST` de incorporación. Los detalles de trazabilidad plegados mantienen
visibles recibo, fecha, estado, límites y período.

La validación global terminó 378/378 pruebas web `PASS` y los manifiestos 111/11/3/1
pasaron. Esta UI ya estaba visible en el runtime `037b4226…`; allí el `GET` daba
`404`. El recorrido posterior descrito debajo obtuvo `200`, pero no llegó al
`POST` de cierre. Los objetivos 11 y 12 conservan la
acreditación del reinicio anterior. Los cuatro `UP` de CT86/87 pasaron primero
en dos clones separados y después se instalaron una vez en principal junto con
CT90. La instrumentación separada de CT87 fue bloqueada y quedó congelada.
No acredita un ensayo funcional.

La nueva bandeja nominal consta de 16 archivos Go y no añade SQL. Exige técnico
explícito y aporta lectores nominales a las dos consultas con intersección de
organización y unidad, sin fallback. Dos revisiones estáticas dieron `GO` sobre
el manifiesto `9e65…`; las pruebas focales de puertos y bootstrap, `go vet` y
compilación y `go test ./...` global pasaron. El E2E con dos lectores sigue
pendiente, por lo que aún no hay `GO` funcional.

El operador resolvió la fuente SMTP como una cuenta remitente todavía por crear
sobre una IP interna de la Diputación. El destino es el correo obligatorio de
un alta VEC existente. Ya no está pendiente elegir el canal; faltan crear la
cuenta remitente y fijar servidor, puerto, TLS y credencial reales.

El candidato de Cobertura reúne 20 fuentes de producto y pruebas con dos `GO`.
La decisión humana es obligatoria y cerrada a **Bolsa**, **SAE** o **Nueva
convocatoria**. La recomendación y su motivo ligado se muestran como ayuda, sin
elegir automáticamente. Se conserva la v1 histórica; la v2 corrige las
secuencias 3/4 y alinea diccionario, DTO y traductor reales. Después de recargar,
la vista recupera el formulario de asignación existente y no crea una nueva
reasignación.

Este corte todavía no está en runtime ni tiene E2E de las tres vías. La campaña
web terminó 385/385 `PASS` y los manifiestos 111/11/3/1 pasaron. Go global
`77665`, vet `98315` y build `31874` terminaron con código 0. El binario
candidato SHA256
`a86116def448a1bae7193bcc86cc7bf3a482c78ec98d3b8f4ddae9450fc29e06`
está preparado y no desplegado. El único lector legítimo actual
combina unidad RRHH y técnico de la organización; no se han comprobado dos
unidades ni la modificación del centro. El material adicional para ese alcance
continúa en revisión.

En el clon normal, CT87 llegó a HTTP `200`, pero la publicación técnica falló
por un P1 de precedencia JSON en su función. El lector SQL cotejó el preflight
posterior `69115647718ec3287262c320887e8d3ff660a566dd1fc6099e0e7f28d4f743af`
y cinco snapshots que cotejan contenido, tres tablas, roles, ACL y esquema,
idénticos byte a byte al estado inmediatamente anterior al intento. No se persistieron los
tres `INSERT`; esta comparación no se atribuye al preflight anterior del
arranque de las 15:18. CT90 se instaló primero únicamente en los dos clones y
después en principal, como se detalla debajo. CT87 está
instalada y no deben reaplicarse sus cuatro `UP`; la instrumentación bloqueada
quedó congelada y no acredita un ensayo funcional. No hay rollback `PASS`
acreditado para este correctivo.

CT90 `581c69…`, focal final `c731…`, recibió dos `GO` estáticos y operativos y
se instaló una sola vez en ambos clones, no en la principal. La focal real pasó
un caso positivo, nueve negativos y la sucesora operativa. El cambio exacto son
tres paréntesis en `cierre87_validar_sucesora`: modifica su cuerpo solo en esos
puntos y conserva OID, firma, roles, ACL, las otras funciones y el resto de la
historia. Los recibos CT86 y CT87 mantienen SHA256
`54a437cf11aefa525b135896e7daf837962819ae0e49ea3486fa676f829f00b7` y
`27379a8fd633455ccf7878935bed791a50c04e6ea02a5e490ea1e82793663675`.
CT86/87/90 tienen historia y no deben reaplicarse. La recuperación `e359…` dejó
sus tres filas correctas. La campaña global Go y vet de continuidad terminó en
`PASS`; la evidencia privada está custodiada sin exponer rutas.

Chrome registró después una anotación administrativa `201` a las
`16:33:10.102225Z`, recibo
`recibo:56756272-1842-4778-b857-e0ae59b322db`, expediente v8→v9 y seguimiento
original v1. El `GET` de recuperación respondió `200` con el mismo recibo. La
comparación SQL de solo lectura confirmó una anotación, una
versión, una actuación y un outbox, con toda la historia anterior como
subconjunto; Personal en cinco tablas, incorporación, raíz y estado de
seguimiento quedaron exactos. La preparación de cierre respondió `GET 200`.
Chrome quedó sin errores JS, cookies, almacenamiento ni desbordamiento a
1440/1024/390, con inspección de dirección a 390 px. En el intento ordinario
posterior, incorporación y preparación devolvieron `GET 200`, se guardó el JSON
y un único `POST` de cierre devolvió `400`. La solicitud original con clave
`aed453da…` se recuperó después de corregir la allowlist HTTP. El replay exacto
respondió una sola vez `201` a las `17:35:57.825562Z`, recibo
`ref:2db8cfe02f7f97bc99b183ac66d579b83698981e78220f301e39f873f8b36f7c`,
seguimiento resultante 2 y expediente v9 preservado; `fuente_recibo` es
`respuesta_json`. SQL de solo lectura confirmó las filas previas de las 12 tablas de historia
conservadas y solo una preparación, un registro, una auditoría y un outbox nuevos
de CT87; las cinco tablas de Personal quedaron exactas. Chrome terminó sin JS,
cookies, almacenamiento ni overflow a 1440/1024/390.

La corrección HTTP consta de los dos archivos Go `7231…` y `d2bfc1d8…` con `GO`;
Go global y vet pasaron. El binario mínimo `bc65…`, sobre la base e4 y esos dos
parches, reinició el mismo clon87 con salud `200`, sin tocar principal ni
material web. Tras reiniciar la app, la anotación recuperó `GET 200` con el
mismo recibo, fecha y v9. Después se reiniciaron conjuntamente esa app y
PostgreSQL del clon87: Chrome recuperó la anotación con `GET 200` y el mismo
recibo/v9, y el cierre con replay `POST 200`, mismo recibo y seguimiento 2. La
comparación conservó 15 líneas CT y estado, salvo una auditoría adicional
prevista de recuperación (`recuperado=true`, `17:51:33.617742`), sin sustituir
la auditoría original. Registro, preparación y outbox quedaron únicos; las
cinco tablas de Personal conservaron SHA256 `35c081…`. No hubo duplicados.
El resultado no acredita firma, cese, eficacia, envío ni GINPIX externo. Los
`404` observados en módulos Bolsa no relacionados quedan fuera de este cierre.

E08 incorpora traducciones comunes de anotación y cierre en siete archivos JS
y pruebas revisados. Las 21 focales pasaron. El motivo visible para la persona
no altera el payload; los mensajes configurados se propagan escapados, y se
conservan las dos fases y el replay. E08 todavía no está desplegado en runtime.
La campaña global terminó 365/365 pruebas CT más 23/23 del coordinador, 388 en
total, y manifiestos 11/111/3/1 en `PASS`. E08 no cambió Go ni repitió la
campaña global Go y vet ya cerrada.

UI5 `f39…`/`37da…` corrige en fuente el P2 del estado mostrado al refrescar un
`GET`; recibió `GO` y quedó aplicada en integración. La raíz terminó 395 pruebas
web y manifiestos 11/111/3/1 en `PASS`. Aún no está en runtime: la evidencia de
navegador usa la web `037b…` y el binario mínimo `bc65…`.

La prueba de CT86 recibió `NO-GO`: su guarda exige 10 s frente a una capacidad
real de 5 s. El arnés SQL no se ejecutó y el negocio permaneció intacto.

El lector DER se limita a dos archivos Go, `1d574869…` y prueba `a9cb…`, con dos
`GO`. Bootstrap completo terminó en 23,254 s y vet pasó. Rechaza antes de
bootstrap el perfil PEM calculado y no introduce una autoridad nueva. La
activación del binario `7d90…` en principal falló a las 16:00 por ese perfil;
el rollback del binario y material propios devolvió salud `200` sin restaurar
la base. El material corregido de stage DER usa un único campo técnico y pasó
11 pruebas más el cargador con contexto real. El reintento está pausado por
compartir material con normal87 y el lector sigue sin activar.

El corte fuente de correo incorpora 18 archivos Go y un catálogo de traducción con manifiesto `cd5642…` y
ocho SQL; ambos grupos recibieron dos `GO`. Implementa E03/E06/E08: reserva
antes de SMTP, auditoría HMAC común, autorización V3 final fresca y outbox. El
JSON contiene diez campos canónicos y el estado admite cuatro literales SQL;
el replay no devuelve el secreto de finalización ni provoca otro SMTP; los
dos accesores del parser existente no crean autoridad. CT88, AD3-32 y T13/5 siguen
sin instalar. Faltan el contacto VEC candidato y acreditado aportado por apoyo,
la cuenta remitente y configuración SMTP concretas en la IP interna, la
composición real y el ensayo PostgreSQL. No hay runtime ni envío. La campaña
global Go y vet terminó en `PASS`; no se repitió la suite web: no cambió JavaScript; catálogo y manifiestos
verificados. El nuevo autorizador de bootstrap sigue pendiente y estos hechos de
fuente no acreditan conformidad.

Cursor9 queda integrado en fuente mediante seis Go (`3e189e10…`) con dos `GO`
de apoyo y un `GO` independiente, más tres SQL CT89 (`9b9b559…`; `UP
1f27d896…`, `DOWN 85853ed…`, focal `89b2be…`) con dos `GO`. Resuelve dos causas:
separa el SHA ASCII del localizador del SHA raw32 de la evidencia, y mantiene la
sesión solo para el mismo TLS, certificado y lector, con autoridad fresca
revalidada y consumo único antes de delegar, bajo mapa de hasta 64 cursores y TTL límite. No añade
cookies ni relaja el TTL SQL. Un cambio TLS, expulsión, reinicio o fallo tras la
reserva obliga a solicitar otra primera página. CT89 no está instalada; no se
han acreditado PostgreSQL 50→2 ni recuperación del cursor, así que no hay E2E.
Go global y vet terminaron en `PASS`. La autoridad y roles normales
no cambian. Contacto13 se aplicó después, como se detalla debajo.

Contacto13 (`3191e8c3…`) queda aplicado en fuente mediante hunks revisados, con
dos `GO` nativos adicionales y dos de apoyo. La API interna Go de VEC, todavía
sin exposición HTTP, ofrece alta, cambio y consulta del contacto usando emisor
V3 real. Cifra con subclave
KMS, AES-GCM, AAD por persona y versión y buffer efímero. El nuevo módulo
`usuarios` declara permisos sin concederlos. Las pruebas V3/PDP/COSE usan
material real; gobierno y persistencia usan dobles. El catálogo incorpora cinco
claves y conserva las dos de correo. `roles_up.sql` es solo fuente y no debe
instalarse. Tampoco hay SMTP ni dirección heredada inferida. Quedan pendientes
el preparador HMAC central y su composición, almacén AD3-35/T13-6, gobierno,
alta web, vínculo Bolsa y replay durable. Go global y vet terminaron en `PASS`.
JavaScript quedó intacto; los manifiestos 11/111/3/1 pasaron y el catálogo de
dos claves de correo más cinco de contacto conserva SHA256 `d22f70a…`. No hay
runtime de este corte.

Correo14 añade cambios en 14 archivos Go en fuente. El manifiesto final
`d23a5d307014b791bdca2dfdbf109ec955f64bc663f4ca749b9d5a8db71d3cc8`
recibió dos `GO`. El rol nuevo de correo tiene exactamente dos concesiones; la
autoridad real PDP/COSE/HMAC está probada, conserva la correlación nominal de
audit17 y deja CT54 sin interceptar. Go global y vet terminaron con código 0.
Este corte no compone HTTP, PostgreSQL o SMTP y no acredita runtime ni otro E2E.

La instalación principal conservó Auth13 y aplicó una sola vez AD3-30/CT86,
AD3-31/CT87 y CT90 (`a65ba4…`). Las tres filas CT conservaron huella `8c6ce…` y
las dos capacidades de gobierno `0c925…`. La misma app arrancó y su `GET` mTLS
respondió `200` (`142ec…`); la incorporación original y la ficha GINPIX se
consultaron solo por `GET`, con recibo y SHA256 `4f56…` intactos.

En Chrome principal, la anotación respondió `201` a las
`2026-09-12T19:22:55.043368Z`, recibo
`430b3ba1-da78-4743-951c-bf5a89737f10`, expediente v8→v9. El cierre respondió
`201`, recibo `cbbc4406…2a3f6`, clave
`3a9924a3-1944-4fc8-b510-2566f6767504`; el seguimiento queda
`cerrado_administrativamente/v2` y el agregado del expediente conserva
`nombramiento/en_curso/v9`. Se conservaron las filas anteriores en las 12
tablas, una incorporación y una raíz; las cinco tablas de Personal
quedaron idénticas byte a byte. Versiones pasaron 108→109 y actuación/outbox
56→57; quedaron una anotación y una preparación, registro, auditoría y outbox
de cierre. Chrome registró cero JS, cookies, almacenamiento y overflow a
1440/1024/390; dirección inspeccionó 390 px. Dos `404` de módulos Bolsa ajenos
impiden resumir la sesión como «todo 200».

El cambio UI `cfa4d70fa14a23827f8cd15420975fe5d5d82fd3`, dos archivos con
focal 14 y `GO`, refleja `cerrado_administrativamente` como estado del
seguimiento y fue revisado sin
secretos. La web terminó 396/396 y los
manifiestos 11/111/3/1 pasaron. No se repitieron Go global ni vet por esta
etiqueta JS; el corte Correo14 ya los había cerrado con código 0. El runtime
usó binario `bc65…` y asset `f0fa…`. Normal87 pertenece a otro entorno: tras
dos replays web conserva las demás tablas y estados y suma solo dos auditorías
de recuperación, cuatro en total. La misma app y PostgreSQL principal reiniciaron
con código 0; antes de HTTP, historia `56c9…` y Personal `35c081…` fueron exactos.
Dos revisiones finales de identidad y SQL dieron `GO` al delta de arranque.
Chrome recuperó la anotación con `GET 200`, mismo recibo `430b3ba1…`, fecha y
v9. El replay exacto del cierre devolvió `POST 200`, mismo recibo `cbbc44…` y
seguimiento 2; el `GET` posterior devolvió `200` y la UI mostró **Cerrado**.
Ambas sesiones conservaron incorporación y ficha GINPIX `200`, recibo original,
fecha y SHA `4f56c3…`. La comparación final SHA256
`9c88e0216b7d8582edaf73ba72b7d409e00d22a36b1049c7270645fd1ad9e850`
conservó once tablas CT, estados y Personal. Auditoría de cierre pasó 1→2 con
exactamente una nueva fila `recuperado=true`, mismo recibo y original intacta;
preparación, registro y outbox permanecieron en uno. Los informes de anotación
y cierre conservan SHA256 `128e050c79cc916dd9bc28cc293acb1f8ef52489be6d2957872e616a4ac86b6a`
y `203ed05dcb2bd875a4f010357442127d831408061c8e1222fc20db8bb3e21e86`.
No se acredita firma, cese, eficacia,
envío, GINPIX externo, efecto legal, cierre jurídico del expediente ni 8/8.

Un clon SQL creado en el mismo clúster compartió dependencias y activó la
guarda global, por lo que no sirvió como prueba aislada. Tras dos revisiones
`GO`, se archivó su evidencia y se retiró solamente el clon propio. La base
principal quedó intacta y el runtime se recuperó. Las próximas pruebas de este
tipo usarán un clúster aislado.

El contraste con RRHH mantiene ocho hitos; las 17 pantallas son maquetas, no
otras fases. Siguen pendientes el correo efectivo, la bandeja destinataria y
las decisiones sobre firma y modelos oficiales.

## Historia de la recuperación del trabajo — 12 de septiembre de 2026

Este apartado y los bloques cronológicos inferiores se conservan como historia;
el estado vivo es el descrito al inicio de este documento.

Orden vigente del operador: revisar y conservar el trabajo, ordenar las ramas y
publicar lo válido y reactivar el director remoto sobre ramas limpias. Este bloque sustituye
la pausa operativa del 10 de septiembre que se conserva debajo como historia.
El apoyo integra esta recuperación; el director remoto ya trabaja en las entregas
siguientes en ramas propias y toma la integración tras recibir la base publicada.

La raíz remota se llevó a la canónica limpia `1b28079c`. Sus 268 diferencias
pendientes quedaron preservadas en `refs/rescate/wip-20260912`
(`stash 7aa50c3c`) y en una copia privada; no se descartó trabajo por estar
sin confirmar. El cotejo de 319 archivos locales encontró 309 idénticos a lo
conservado en remoto, diez divergentes y ninguno ausente. Las divergencias
requieren revisión; no justifican reconstruir ni sobrescribir la canónica.

Los veinte archivos de backend de anotación, cierre y composición nominal
tienen dos revisiones independientes. Se corrigió el P2 de observaciones: la entrada
reutiliza la validación del servicio y devuelve `422` antes del ejecutor,
conservando el texto para corregirlo. La regresión focal, carrera y `go vet`
de HTTP están verdes. La candidata conjunta supera `go test ./...`, `go vet ./...`,
las 314 pruebas web de Contratación y el verificador de manifiestos. El rango
canónico anterior y los 29 archivos seleccionados superan la detección de secretos.
La publicación se comprueba por el hash de la referencia remota después del envío.

También se corrigen cuatro pruebas desactualizadas: tres fixtures y la
comprobación de arquitectura que omitía las tres llamadas a constructores TCB
ya integradas en la función privada de composición desde `28b14bba`.
La excepción queda limitada a esos constructores, archivo y función; no
autoriza reexportaciones ni modifica código productivo. `go vet` también detectó
dos literales de prueba sin nombres de campos; se corrigieron conservando sus valores.

El frontend conserva un `NO-GO` con hallazgos concretos pendientes de corrección;
no se presenta como integrado por las pruebas aisladas de sus componentes.
Faltan su revisión final y el caso DOM conjunto `v8→v9`, incluida la nueva
preparación del cierre después de anotar. El acceso Codex nativo con el perfil
`aavidad` está comprobado y se corrigió la confianza del proyecto que impedía
cargar su configuración. El director y sus agentes ya trabajan en programación,
pruebas, revisión, documentación y preparación operativa. El techo es 24 sesiones
en todo el árbol, incluido el director; no supone 24 sesiones siempre ocupadas.
Terra/medium para código y pruebas, Sol/medium para documentación, Luna/low
para tareas mecánicas y Astra/high para dirección o revisión sensible. `xhigh`
requiere una dificultad concreta justificada, nunca una tarea mecánica.

Este corte no ha instalado SQL ni cambiado la aplicación servida, la base o los
recibos. Auth13 sigue pendiente de instalación; CT86/AD3-30 y CT87/AD3-31
requieren validación e instalación propias. No reaplicar CT70–85 ni repetir
el POST de incorporación. Tras integrar y publicar lo válido, continúan la
recuperación por GET, GINPIX, anotación/cierre y recorrido conjunto
(objetivos 11–14); el vencimiento sigue condicionado a inicio y política
acreditados. La métrica funcional no aumenta.

Para comprobar el estado de código desde el repositorio remoto:

```sh
git status --short
git rev-parse trabajo/ct-app-llamamiento-b4a-20260905
git show --no-patch --oneline refs/rescate/wip-20260912
```

## Cierre por cuota — orden de Alberto, 10 de septiembre de 2026

Sesión cerrada por orden expresa del operador. No reactivar agentes ni continuar
el desarrollo automáticamente. Al comprobar el runtime sólo figuraba el director;
no había subagentes ni descendientes activos disponibles. No se iniciaron nuevas
campañas, instalaciones, migraciones ni cambios de cuenta para este cierre.

Quedan confirmados en la canónica los cortes `76a2b7c1` (consulta original),
`1385b134` (panel), `8b3d1ee2` (continuidad del cierre), `52d4862f` (dominio de
anotación), `a063eac1` (referencias móviles), `2ba4b86e` (persistencia de anotación)
y `9fb12560` (persistencia de cierre). Este corte añade únicamente los dos
archivos de sellado HMAC de anotación ya revisados por dos especialistas y
comprobados con pruebas focales; conserva generaciones, dominios separados y
rechazo de cancelación después del último conector. No se repitieron pruebas
para cerrar. Ningún commit de esta sesión se presenta como publicado.

Pendientes conservados, sin integrar como terminados:

- HTTP de anotación: última reparación Astra en cuatro archivos, diez pruebas
  focales aprobadas; faltan los dos dictámenes sobre sus hashes finales. Los
  NO-GO anteriores no se convierten automáticamente en GO por el parche.
- Composición nominal: doce archivos propios y cuatro dependencias congelados
  en `composicion-nominal/manifest-review16.json`; consta GO estático de identidad.
  Falta recoger/cerrar la segunda revisión y validar la candidata conjunta.
- Interfaz: formularios y cliente conservados; pruebas aisladas de anotación y
  cierre con transporte sintético. El montaje tras anotación incorpora un callback
  y una guarda de vigencia corregida, pero falta el caso DOM conjunto v8→v9 que
  demuestre el nuevo GET de preparación y el cierre habilitado. No atribuir al
  montaje final las pruebas de versiones anteriores de los componentes.
- Los cambios de paginación/foco y las bajas ajenas del manifiesto siguen como
  WIP separado. Para inventariar usar bytes de `git show` de la canónica frente
  a archivos reales: el índice ajeno no representa lo pendiente de esta rama.

La copia privada está en
`/root/.local/state/vec-codex-director-20260910/cierre-cuota/`: inventario de 179
archivos sobre `9fb12560`, copia del WIP, parche y entregas temporales. Incluye
material ajeno conservado dentro del alcance; no atribuye su autoría. Las demás
candidatas, manifiestos y evidencias permanecen en el directorio privado padre.
No se copiaron credenciales ni bases y no se modificaron documentos WIP del árbol.

Punto de reanudación: inventariar esa copia y los hashes finales; cerrar revisión
HTTP de anotación y segunda revisión de composición; comprobar únicamente la
candidata conjunta y el caso DOM pendiente; integrar hunks propios sin paginación,
foco o bajas ajenas. Después, con canal operativo autorizado, resolver Auth13 y
la configuración nominal/publicaciones conservadas, comprobar GET de incorporación,
GINPIX y el recorrido anotación→cierre con recuperación e historia sin duplicados.

Auth13 `ef6704a6` sigue revisada pero no instalada; CT70–85 ya tienen historia y
no deben reaplicarse. CT86/AD3-30 y CT87/AD3-31 están integradas como scripts,
sin ejecución SQL ni instalación acreditadas. No repetir el POST de incorporación.
Los objetivos 11–14 permanecen pendientes de cierre conjunto en runtime; el
objetivo 6 aún requiere inicio y política acreditados. No hay aplicación terminada.

HEAD ajeno conservado: `d25861ff34670498b1c7825e753c36de4ca859d0`. SHA256 del
índice real: `b46876aa1fa9d138e1dae28f4c406c4712f9eb9e44bcda98c7df334558ae62db`.
Sin stash, reset, limpieza, reinicios ni cambios en bases o recibos.

## Cierre administrativo: persistencia integrada en código

La candidata de dieciséis archivos conserva la raíz y la fundación v1 al
adoptar la definición sucesora v2 para el cierre. El cierre es nominal: no
registra cese ni altera periodos. El inventario exige exactamente las dos tareas
antecedentes —incorporación CT75 y primera anotación CT86— y no inventa un
`Documento`. La autorización V3 fresca se exige al crear y en replay.
Actuación, snapshot, recibo y outbox se escriben en una sola transacción.

Dos revisiones independientes emitieron `GO` sobre los dieciséis hashes
finales. La candidata aislada verificó esos dieciséis hashes y pasó las pruebas
Go del adaptador PostgreSQL, con dobles transaccionales, en diez casos
principales y siete subcasos, `0.034 s`. Las cuatro pruebas de dominio y
puertos también pasaron, en `0.019 s` y `0.022 s`.

CT87 y AD3-31 no se han ejecutado ni instalado en la base conservada. La
configuración, el `GET` HTTP y el montaje principal de runtime siguen
pendientes. El formulario no está integrado, no existe cierre visible y el
objetivo 13 continúa abierto.


La anotación administrativa ya está integrada en código en servicio, puertos y
adaptador PostgreSQL, con los scripts CT86 y AD3-30 como artefactos. El primer
asiento es neutro: conserva fase, estado, raíz y contexto completo, incluidas
observaciones Unicode. La autorización nueva se exige al crear y recuperar; el
servicio exige una fuente de recuperación autorizada. El puente mediante
`ConsultaDetalleRRHH` pertenece al montaje pendiente.

Dos revisiones independientes emitieron `GO` sobre los siete archivos finales
de producto —identidad y SQL—, y la revisión del harness SQL emitió `GO` sobre
su alcance potencial de 22 tablas listadas. La candidata aislada, formada por
la canónica `a063eac1` y diez archivos exactos, pasó las pruebas focales de
aplicación y adaptador PostgreSQL, sin dependencias ausentes. Los scripts SQL
no se han ejecutado ni instalado.

API, HTTP, frontend y montaje nominal quedan fuera de este corte. La dependencia
raíz y fuente nominal y su configuración siguen pendientes, igual que Auth13
en runtime. No existe anotación visible ni está completo el objetivo 13.

La continuidad de incorporación ya está integrada en dominio y servicio para una
única adopción sucesora publicada, conservando fundación v1, raíz y cadena y
permitiendo cierre nominal sin cese ni cambio de periodos; dos revisiones
independientes emitieron `GO` y la prueba focal aislada quedó verde en dominio,
aplicación y puertos, incluido el golden V1, mientras SQL, adaptador, HTTP,
composición, autoridad, publicación y libro de runtime continúan pendientes,
por lo que no existe cierre visible ni está completo el objetivo 13.


## Backend y panel integrados en código; runtime pendiente

El backend confirmado es
`76a2b7c13309bc0526e45d585c2fdf7ca77aecf6`. Este corte reúne sobre la base
`3375d527` las entregas de consulta, proyección, HTTP y panel `ORIGINAL`.

El contrato de lectura es
`GET /api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=<referencia>`.
La respuesta `original_incorporacion` representa la incorporación histórica
que originó el seguimiento; no representa el estado actual del expediente.

Dos revisiones independientes con Astra/high emitieron `GO` para el backend y
sus dos dependencias. Seis pruebas focales Go están verdes en cinco paquetes.
Node está verde `7/7`, incluido el montaje principal.

El primer arnés Chromium usó CSS inventado y no acreditaba el CSS de producto.
El arnés siguiente cargó el índice candidato de `1385b134`, los doce CSS
reales y referencias de 64 caracteres: detectó `scrollWidth=834` en 390 px,
con escritorio correcto. Tras el hunk CSS único, `cssfix3` midió
`1440/1440` y `390/390`, pero superponía una navegación inventada y no
acreditaba ausencia de obstrucción.

La repetición final `cssfix4` usa el wrapper mínimo real del componente CT,
sin `nav` ni `aside`, los doce CSS reales y transporte interceptado. Dio
`PASS`: ancho de documento igual al viewport a 1440 y 390 px; una consulta;
referencias completas; cero errores JavaScript, consola, almacenamiento o hijos
tras destruir la raíz. En ambos anchos, `elementFromPoint` devolvió `H3`
para el título y `BUTTON` para el botón, ambos visibles y sin obstrucción.
Dirección inspeccionó la captura móvil, legible y sin superposición, con el
botón completo. El resultado tiene SHA256
`149ae2ab956667d624834169b66dd987f18b2f6d60dc821dfff69dea49e937e2`.
Esta evidencia acredita sólo el componente con transporte sintético
interceptado; no prueba el shell completo ni un E2E de runtime. La campaña global `go test ./...` no se ejecutó porque la revisión
automática rechazó su alcance masivo.

Auth13 `ef6704a6` no está instalado y la recuperación `GET` continúa bloqueada
en runtime. GINPIX `e2831250` está integrado, pero no se ha probado sobre un
recibo recuperado; no hay un resultado nuevo de GINPIX que comunicar. Siguen
abiertos el objetivo 11 (recuperación), el 12 (GINPIX en runtime), el 13
(anotación y cierre) y el 14 (recorrido conjunto). En el objetivo 6, tanto el
inicio como la política siguen pendientes de elección y acreditación.

**Estado comprobado y recuperación pendiente — 10 de septiembre de 2026.** El formulario de
incorporación registró una operación real de desarrollo a las
`2026-09-10T13:07:06.614186Z`: `GET 200` y `POST 200`, dos confirmaciones
expresas, sin errores JavaScript ni desbordamiento en 1440/390 px. Conserva
recibo `ref:2bc3d281379b33ed532825768a29001d19cc7af96c58cead9badb4d13fa6290b`,
solicitud `7`, expediente `8`, seguimiento `0→1` y periodo `2027Q1`, con
`firma_oficial=false` y `eficacia_administrativa=false`.

La escritura produjo una alta de Personal, una relación, una ocupación, una
auditoría y un outbox; en Contratación temporal, una incorporación, una
auditoría, un outbox, una raíz y dos estados. No hubo precargas de negocio.
CT70–85, trece pools y tres capacidades con catálogos y definición constan ya
provisionados; no reaplicar. Tras reiniciar aplicación y PostgreSQL, la lectura
de recuperación devolvió `GET 503` por la restauración histórica Auth12. Auth13
tiene dos dictámenes `GO` y regresión verde en `ef6704a6`, pero sigue pendiente
de instalación. Por tanto, no hay ciclo recuperado completo ni debe repetirse
la escritura. Se mantienen cinco pasos completos y partes de 6/7/8 únicamente
como avance parcial; GINPIX es el siguiente corte existente aún sin cierre
visible. Integración escrita y evidencia visible deben comunicarse por separado.

**Cierre operativo del 6 de septiembre: puente centro→RRHH demostrado.** La
única copia canónica está en OpenClaw, rama
`trabajo/ct-app-llamamiento-b4a-20260905`, sobre la base GitHub `85cb47ce`; el
entorno local anterior queda congelado y el incremento queda documentado en
este corte sobre esa base. Se mantiene la misma rama; antes de presentarlo como
publicado hay que verificar con Git el hash y el estado de publicación.
Dirección está autorizada a confirmar y publicar tras revisar el corte, sin que
este estado afirme que el `push` ya se haya realizado. En Chrome, tras reiniciar
aplicación y PostgreSQL privados, la
recuperación obtuvo `GET 200`/`POST 200`, recibo completo idéntico y ninguna
alta nueva: el primer `POST 503` había creado el alta y el reintento la enlazó
sin duplicar. Se conservan 52 expedientes, 52 versiones, 2 revisiones de
petición, 1 reserva, 1 confirmación y 1 evento de entrega, con huellas previas
intactas. El recibo original
`recibo:ct-alta:9e45c28d21cadddc4f0bfd548249c555b0e0e3c55ce04b7d14052a65eec46467`
se creó `2026-09-06 17:30:26.404646Z` para el expediente
`expediente:ct:4ff4285d7ae5d6c4fb34d199a942d7c7d66ba4274ebbaab44fc8296089e8c5cf`,
número `2026/CT-d06f98d5506ded3ee3b8a7d34d867b1e`, versión 1. Las vistas
1440/1024/390 no tienen overflow global, la tabla desplaza internamente y no
hubo errores JS, cookies ni almacenamiento web.

AD3-24/CT68 ya están instaladas con historia: no reaplicar `DOWN`/`UP`. La
definición SQL estructural se ajustó una vez preservando tablas. Restaurar
requiere también las ACL de base y 43 tipos de fila además del dump; el detalle
privado está fuera de Git. CT68, manifiesto estructural y CSS acreditados por
los SHA256 consignados en `AGENTS.md`, con dos `GO` estáticos vigentes. Esto no
es producción ni firma legal, no cambia la métrica —cinco pasos completos más
partes del sexto/séptimo— y deja como siguiente cola el objetivo 10: resolución
y evidencia conforme a la fuente o circuito admitido. Dirección está autorizada
a continuar las partes reutilizables para presentación; nunca se inventará una
firma legal ni se presentará autenticación o evidencia sintética como tal firma.

**Plan vigente: 6 de septiembre de 2026. Prioridad exclusiva: Contratación temporal.**

**Entrega solicitada por el operador: aplicación funcional para presentación,
no puesta en producción.** El cierre de esta entrega exige recorrer los ocho
pasos desde el navegador con datos sintéticos, persistencia y recibos reales,
centro/personas responsables identificados y manual de demostración. No basta
una pantalla de muestra. Correo corporativo, firma oficial, datos reales y
conexión automática a GINPIX no bloquean esta entrega: se emplean avisos locales
visibles, borradores claramente marcados, actuaciones sintéticas explícitas y
ficha descargable. Nunca se presentan como envío, firma o nombramiento legal.
Las decisiones configuradas para el ejercicio son de desarrollo y sustituibles;
no se inventan reglas jurídicas. La autorización productiva queda fuera de este
cierre, sin perder su lista de dependencias. Se mantienen los ocho pasos y la
cola funcional existente; no se reabren capacidades terminadas.

La presentación se construye sobre la aplicación final: mismo dominio,
formularios, casos de uso, persistencia e historial. No se crea una maqueta
desechable ni una segunda implementación. Identidades sintéticas, avisos locales
y modelos documentales de desarrollo quedan en configuración/adaptadores
sustituibles; los conectores corporativos se incorporarán por los mismos puertos.

Cada sustitución de presentación debe explicarse en el paso afectado y en los
manuales: qué se ha realizado, qué no, y qué conector o actuación se requiere
para uso real. Ejemplo: «Documento generado y guardado, sin firma electrónica;
para uso real debe incorporarse la firma mediante certificado o servicio
corporativo que se establezca». La autenticación con certificado no constituye
la firma de un documento. No indicar como aprobada una alternativa todavía
pendiente de concretar con la Diputación.

Petición funcional posterior del operador: centros y categorías de la RPT,
organización editable y ratificación previa según cada unidad, sin niveles
obligatorios inexistentes. Primer corte: consulta real de los 41 centros de
la fuente publicada, sus agrupaciones y once puestos cotejados de Transformación
Digital. API y pantalla conectadas al catálogo versionado de Personal,
en preparación; no es un organigrama funcional completo ni habilita altas.
Edición durable demostrada: formulario real, autorización, PostgreSQL y recibo;
alta y edición de un cargo sintético, recuperación tras reiniciar aplicación
y base, sin duplicados. Conservadas3revisiones/2cambios/2eventos y51expedientes.
Siguiente: vinculación de responsables autorizados y ratificación antes de RRHH. No
deducir permisos ni dependencias por el orden de las filas. La categoría
profesional y el código individual de puesto no son la denominación RPT.
Este corte no añade un sexto paso completo. Se mantiene la cola inferior,
incluidas las dependencias de firma y nombramiento, sin reabrir lo cerrado.

Primer PDF del objetivo 9 publicado en `5c57b29f`.
AD3-21 cerrada en `975f0c16`, resolución publicada en `9e692f80`.
Objetivo 9, seis PDF borradores, publicado en `c82a3068`. Esta revisión incorpora
la propuesta desde la aceptación del sucesor CT65, tras la aceptación CT64 del corte `117c9fb9`;
el hash publicado se comprueba en Git.

Este es el único plan operativo. El historial inferior se conserva como
referencia; sus porcentajes, carriles y órdenes antiguos no dirigen el trabajo.
Bolsa y los demás módulos quedan fuera, salvo la pieza mínima que necesite
contratación. No se reabren los cinco primeros pasos ya recorridos.

## Punto de partida comprobado

- Circuito previo del centro demostrado: presentación y ratificación con dos
  certificados sintéticos distintos, formulario real, autorización, PostgreSQL
  y recibos. Tras reiniciar aplicación y base principal, ambos reintentos200
  conservan recibo/fecha: una petición, dos revisiones/eventos, 51 expedientes
  previos intactos. Autorización23/Contratación67 ya tienen historia; no revertir.
  Dos revisiones independientes de las zonas sensibles cerradas sin bloqueos.
  Siguiente incremento observable: entregar la petición ratificada al alta
  existente de RRHH. No crear otro alta ni duplicar código entre ramas.
  Métrica sin cambio: cinco pasos completos y partes del sexto/séptimo.
- Publicado: cinco pasos del procedimiento de RRHH y parte del sexto
  (selección, apertura de llamamiento y aviso local), con datos sintéticos.
- Manuales de usuario, RRHH, programación y Sistemas publicados en
  `287751ad4042167a5920ba79f26815241718af29`.
- Bandeja, detalle y acceso al análisis publicados en `b2effba`, con corrección
  de instalación `13f7a92`. Desarrollo local, misma rama, ambas bases y material
  conservados; instancias remotas detenidas. Consultas instaladas en ambas bases.
  Recorrido principal 8443/55433: lista de 50 solicitudes → abrir una existente
  v1 → formulario de análisis → HTTP 201 y recibo v2. Tras reiniciar aplicación
  y PostgreSQL conserva 50 solicitudes y un único recibo/asiento de análisis.
  La base secundaria 8444/55432 conserva sus 21 expedientes; no se mezclan datos.
- Corte 3 incluido en esta entrega, cerrado técnicamente: respuesta declarada por RRHH registrada desde el
  navegador con HTTP `201`, actor, referencia y huella del correo y
  justificante persistentes. Lectura confirmada: una respuesta, un asiento
  y un evento. Migraciones CT `000056` y autorización `000014` instaladas en
  ambas bases; no reaplicar. Los rechazos intermitentes `403` se debían a
  comparación textual de fechas equivalentes, no al reinicio.
  Comparación temporal corregida en ambas bases mediante el bloque literal
  `DO $fechas$` de autorización `000014`, con sus tres comprobaciones correctas;
  diagnóstico temporal retirado. **Recuperación confirmada después del parche
  y segundo reinicio de aplicación y PostgreSQL principal: navegador
  `200/200/200` para selección, comunicación y respuesta**, mismas claves,
  justificante, recibo y fecha originales. Sin errores de JavaScript, cookies,
  almacenamiento web ni desbordamiento horizontal en móvil.
  Conflicto `409` comprobado desde el navegador, sin duplicado.
  No cambia Bolsa ni expediente; comunicación sigue en versión `2`.
- Objetivo 4 cerrado técnicamente para aceptación manual sintética desde el cuarto formulario.
  Navegador real `200/200/200/201`, con antecedentes originales, API, V3, CT58 y
  Bolsa4 reales; cero errores JS, cookies, almacenamiento web y desbordamiento.
  Recibo CT `recibo:d6bdcc7b-e22e-4fe9-8aac-a1eb554a4103`, resolución
  `7a3e4a2e-d142-4562-ae5c-59c95b011e0c`, fecha `2026-09-05T22:27:02.861379Z`.
  Tras reiniciar app/PostgreSQL principal: `200/200/200/200`, mismo recibo/fecha.
  Base: una resolución CT58 y una aceptación Bolsa, tres historias y tres
  eventos, sin duplicados. No se atribuye este recorrido a la secundaria.
  Dos revisiones RRHH expresas y política fija
  `politica:ct:revision-manual-sintetica:20260906`; no aprobación legal real.
  Éxito solo tras CT y Bolsa; consulta/escritura/Bolsa conservan permisos propios.
  La petición antigua sin revisión manual sigue en `409` pendiente, sin efectos.
- Objetivo 5 cerrado funcionalmente: renuncia manual sintética desde el mismo formulario,
  navegador real `200/201/201/201`, sin errores JS, cookies, almacenamiento web
  ni desbordamiento. Nuevo expediente `fe4934a1…`, fiscalizado `v6`; la base
  principal conserva 51 solicitudes, con bandeja y detalle consultables.
  Recibo `recibo:408fda57-638d-4a3b-a441-4ef56396e23a`, fecha
  `2026-09-05T23:00:45.289468Z`; intención
  `intencion:f4bd0049-8b96-410b-8144-4384bdb47ed0` pendiente, con carga real
  durable en la misma fila CT, sin siguiente candidato ejecutado.
  Dirección confirma dos resoluciones CT (aceptación y renuncia); Bolsa conserva
  dos órdenes, dos propuestas, una aceptación y una renuncia.
  Tras reiniciar app/PostgreSQL principal, dirección confirmó `200/200/200/200`,
  mismos recibo, resolución, auditoría, fecha e intención con su carga real.
  Dos filas CT, seis registros Bolsa, seis historias y seis eventos; sin duplicados.
  Aceptación y declaración anteriores intactas. Cierre solo del ejercicio sintético.
  AD3-18/Bolsa5/CT59 instaladas en ambas bases; UP/DOWN con ACL, funciones y
  comprobaciones correctas según dirección. No reaplicar ni revertir estos datos.
- Objetivo 7 recorrido desde la quinta operación del mismo formulario: HTTP `201`
  real tras la renuncia `0c5fdea4…`, mismo expediente `fe4934a1…`; un nuevo
  llamamiento abierto y recibo CT `recibo:b5bb611f-0126-4806-90c5-85b9f9b63778`,
  fecha `2026-09-05T23:57:11.037866Z`. Bolsa conserva siete registros (dos órdenes,
  tres propuestas, aceptación y renuncia), siete historias y siete eventos.
  Tras reiniciar app/PostgreSQL principal, navegador `200/200/200/200/200`:
  mismos 14 campos salvo `estado_local: replay_confirmado`, sin duplicados.
  Cero errores JS, cookies, almacenamiento web y desbordamiento. Objetivo 7
  cerrado funcionalmente solo tras renuncia sintética.
  La intención CT original se conserva; el recibo de renuncia permanece histórico.
  No acredita aviso enviado, aceptación del nuevo llamamiento ni plazo legal.
- Objetivo 8 cerrado funcionalmente en desarrollo: Chrome `201` y recibo visible
  de propuesta `propuesta:2dd1c999-44c3-4fdc-b68e-e0adde592c81`, recibo
  `recibo:3335969d-3bb5-4258-afcc-1af26b7f7207`, fecha `2026-09-06T01:28:30.697897Z`.
  Tras reiniciar app/PostgreSQL principal: cuatro antecedentes `200` y propuesta
  `200`, mismos identificadores/fecha/v7. Agregado `nombramiento/en_curso/v7`, una
  actuación v7 y un outbox nuevos; historia previa intacta. Cero errores JS,
  cookies, almacenamiento web y desbordamiento. AD3-20/CT61 instaladas en ambas
  bases, no reaplicar; navegador acreditado solo en principal. Sin firma ni
  nombramiento eficaz. El detalle RRHH por API confirma v7, nombramiento y siete hitos.
  Esta revisión incorpora el cierre funcional; el hash publicado se comprueba en Git.
- Consulta de justificante conectada con permiso propio V3 real y fresco,
  misma respuesta/recibo y auditoría de acceso. Es interna: no crea DTO HTTP ni expone
  `Seleccion`. AD3 `000016` / CT `000057` instaladas en ambas bases locales,
  `55433` y `55432`; consulta repetida tras reiniciar app/PostgreSQL principal.
  No concede por sí sola aceptación ni valida un plazo legal.
- AD3 `000015` / Bolsa `000004` y AD3 `000017` / CT `000058` están **instaladas
  en ambas bases**, con ambas apps en la compilación corregida; no reaplicar.
  La prueba aislada anterior de Bolsa4 (`8197db3`) usó un doble privado
  transaccional: acreditó almacenamiento/replay/conflictos, no criptografía.
  El roundtrip de aceptación UP/DOWN de las cuatro migraciones comprobó reversión
  exacta en ROLLBACK, sin modificar autorización ni usar dobles. El navegador
  actual sí utilizó criptografía real; son comprobaciones distintas.
- Métrica: **5 de 8 pasos completos más partes del sexto y séptimo**, no un 100 %.
  Registrar una declaración de aceptación no resuelve la aceptación ni
  verifica origen, firma o custodia del correo. El `.eml` se lee y resume
  localmente en el navegador; no se sube.
- Objetivo 9, primer documento demostrado: informe definitivo, borrador de desarrollo,
  Chrome `200`, 29267 bytes; PDF y pantalla inspeccionados por dirección.
  Rectificación: primer fallo navegador sin estado HTTP capturado; el `502` era curl
  con límite 50, paginación separada sin corrección acreditada. Navegador límite 100:
  sonda `200`, vista `404`, rechazo PostgreSQL `42501` por fechas equivalentes comparadas
  como texto. AD3-21 corrige las dos funciones de lectura y está instalada en ambas bases,
  sin reaplicar AD3-20/CT61; UP/DOWN exacto en ROLLBACK en secundaria.
  Tras el parche, cinco POST de bandeja/detalle/PDF `200`, mismo tamaño y SHA256,
  cero errores JS, cookies y almacenamiento web. Tras reiniciar app/PostgreSQL principal:
  dos cuadros iniciales 100, filtrado 100, detalle v7 y PDF, los cinco `200`, sin `404`;
  mismo PDF e historia CT/Bolsa conservada. Dirección observó pantalla estable de 390 px sin obstrucción
  del detalle ni botón; captura previa durante transición de 180 ms, sin modificar UI
  ni validar usabilidad global. Sin SQL propio del PDF ni replay
  de propuesta para descargar.
  Segundo borrador: resolución, Chrome `200`, 29770 bytes; misma sesión con informe
  original idéntico, seis POST `200`, cero errores JS, cookies y almacenamiento web.
  Dirección inspeccionó PDF y pantalla estable de 390 px con dos botones.
  Tras reiniciar aplicación/PostgreSQL principal: otros seis POST `200`, ambos PDF
  idénticos en tamaño y SHA256, historial y recibos anteriores conservados;
  cero errores JS, cookies y almacenamiento web. E2E acreditado solo en principal.
  Tercer borrador: diligencia, Chrome `200`, 28366 bytes; siete POST `200`, informe
  y resolución idénticos, cero errores JS, cookies y almacenamiento web.
  Dirección inspeccionó PDF y pantalla estable de 390 px con tres botones.
  Tras reiniciar aplicación/PostgreSQL principal: otros siete POST `200`, tres PDF idénticos;
  historial de dos resoluciones CT, siete registros Bolsa y versiones 1..6 igual al previo.
  Cero errores JS, cookies y almacenamiento web; E2E acreditado solo en principal.
  Comunicación al centro: diez POST `200` antes y después del reinicio principal;
  seis PDF e historial conservados. Evidencia y huella en la guía.
  Objetivo 9 cerrado funcionalmente en desarrollo: **6/6 borradores**; siguiente 10, fuente/circuito de firma admitido.
  Sin SQL nuevo, firma, envío, entrega, plazo legal ni orden de incorporación.
  Corrección temporal confirmada tras reinicio; no se atribuye una carrera ni se cierra la paginación 50.
  Aviso local al sucesor cerrado: `201` y seis operaciones `200` tras reinicio principal,
  mismo recibo/fecha/v2/outbox. CT62 instalada en ambas bases; no reaplicar ni ejecutar DOWN.
  Tres comunicaciones/historias/outbox, anteriores intactos; evidencia central en la guía.
  Declaración del sucesor CT63: seis antecedentes `200` y registro `201` real; cruces `409` sin efectos.
  Tras reiniciar app/PostgreSQL principal, siete operaciones `200`, mismo justificante/recibo/auditoría/fecha;
  tres declaraciones/historias/outbox y antecedentes CT/Bolsa intactos, sin duplicados. Declaración cerrada funcionalmente.
  CT63 instalada en ambas bases; no reaplicar ni DOWN con historial. Datos exactos en la guía.
  CT64: aceptación manual sintética del sucesor confirmada CT+Bolsa mediante recuperación con la misma clave;
  primer `503` había conservado CT, no completado Bolsa. Ocho POST `200` antes y después del reinicio principal,
  mismos recibo/fecha/evaluación/auditoría; tres resoluciones CT y ocho operaciones/historias/outbox Bolsa,
  antecedentes intactos. En aquel corte, versión resultante `3`, expediente `6`. CT64 instalada en ambas bases:
  no reaplicar ni DOWN con resolución sucesora, sin SQL remoto.
  [Octava operación y evidencia](GUIA_RECORRIDO_ALBERTO.md#resolución-manual-del-sucesor-ct64).
  CT65: ocho antecedentes `200` y propuesta `201`; tras reiniciar app/PostgreSQL principal, nueve `200`,
  mismos recibo/fecha/v7 e historia. Expediente `fe4934a1…`, `6→7`, nombramiento/en_curso;
  versión esperada de solicitud siempre `6`. Dos propuestas, 55 actuaciones y 55 outbox globales (+1 cada uno).
  Tres respuestas/comunicaciones/resoluciones, ocho Bolsa, propuesta original y trece versiones previas intactas.
  CT65 instalada en ambas bases; no reaplicar ni DOWN con propuesta sucesora, sin SQL remoto.
  [Novena operación del panel, no noveno paso RRHH](GUIA_RECORRIDO_ALBERTO.md#propuesta-desde-la-aceptación-del-sucesor-ct65).
  Faltan vencimiento, envío corporativo y plazo legal; seis PDF cerrados, no repetidos en este corte.
  La aceptación manual sintética no acredita entrega ni plazo legal aprobado,
  nombramiento, incorporación o producción.
  El criterio manual de desarrollo sigue provisional, no aprobado por el operador.

## Cómo trabajamos desde ahora

Configuración de equipo preparada el 10 de septiembre de 2026 en
`.codex/config.toml`: director Astra/high, doce perfiles iniciales con modelo y
esfuerzo explícitos, y siete skills de proyecto bajo `.agents/skills/`.
Se autoriza delegación recursiva dentro del encargo y los permisos recibidos.
El director puede elevar modelo o esfuerzo ante dificultad o fallo, sin nueva
confirmación del operador, conservando el trabajo y verificando la sesión de relevo.
Carga comprobada en Codex 0.147 con `initialize`, `config/read` y `skills/list`:
doce perfiles sin error y siete skills detectadas. Arranque nativo solicitado
el 10 de septiembre: Codex 0.147 fue rechazado por versión insuficiente para Astra.
Instalada copia independiente 0.154 en `/opt/vec-codex/0.154.0`, conservando la anterior.
La sesión de aavidad@gmail.com terminó por cuota agotada; el error indicaba
renovación el 15 de septiembre a las 14:30, sin zona horaria declarada.
El operador ha elegido expresamente alberto@avidad.com y completado OAuth.
Autenticación confirmada con `login status` y correo del ID token, sin copiar
ni mostrar credenciales. Cada cuenta conserva su CODEX_HOME independiente.
Selector instalado en `/opt/vec-codex/bin/vec-codex`, versionado en
`.codex/bin/vec-codex`; acepta `aavidad` o `alberto`, sin alternancia automática.
Comprobar con `/opt/vec-codex/bin/vec-codex alberto login status` y
`/opt/vec-codex/bin/vec-codex aavidad login status` como root.
El nuevo acceso no acredita cuota ni agentes activos por sí mismo. El operador
ha ordenado arrancar el director con alberto; se verificará el proceso y sus
sesiones reales antes de declarar el equipo trabajando.
La unidad `vec-codex-director-20260910.service` y su encargo están preparados;
el arranque es directo con Codex CLI y no utiliza el coordinador OpenClaw.
Estado consultable con `systemctl show vec-codex-director-20260910.service
-p ActiveState -p SubState -p MainPID -p ExecMainStatus`.
Las cuatro raíces anteriores de OpenClaw están archivadas, con código y WIP
conservados. Este corte cambia la coordinación, no añade un paso RRHH recorrido.

1. Un objetivo funcional en curso por línea canónica. Un director Codex nativo
   integra; los agentes crean y gestionan subagentes a demanda, también anidados,
   con archivos exclusivos y responsabilidad de cada padre sobre sus entregas.
   No hay árbol fijo ni cuota total de especialistas. La concurrencia inicial
   configurada es doce; se respeta el límite efectivo del runtime. Los perfiles
   pueden ampliarse cuando una tarea requiera otra especialidad.
2. Cada corte debe dejar algo que se pueda enseñar desde el navegador.
   Se busca un tamaño de **una sesión corta, unas 1–4 horas de trabajo**:
   es un límite para dividir el trabajo, no una promesa de duración.
   Si aparece una dependencia mayor, se identifica el siguiente resultado
   mínimo y se divide antes de continuar; no se prolonga un hito durante semanas.
3. Cada cierre tiene código, comprobación focal y recorrido visible.
   Cuando escribe datos, se comprueba persistencia y recuperación sin duplicar.
   Las pruebas se agrupan al terminar el corte, no por línea modificada.
   Se conservan las revisiones independientes exigidas para identidad,
   permisos, criptografía y SQL; no se añaden campañas preventivas generales.
4. Commit pequeño y coherente; revisión aplicable, inventario y equivalencias
   antes de integrar, publicación y comprobación del estado remoto. No se
   mezcla WIP ajeno ni se retiran ramas con trabajo pendiente.
5. Se actualiza esta tabla y el manual afectado en el mismo cierre. La bitácora
   existente recoge avances y fallos concretos, sin crear tableros, contratos
   de gobierno ni documentos de decisión adicionales.
6. Se corrige lo que impide el recorrido o induce a error. Mejoras no necesarias
   no abren una reescritura ni desplazan el siguiente objetivo.

## Cola de entregas pequeñas

El orden expresa dependencias, no autoriza a dar por existente una pieza.
Antes de cada edición se comprueba qué implementación ya está disponible.

| Orden | Entrega observable | Comprobación de cierre | Estado |
| --- | --- | --- | --- |
| 1 | Abrir un expediente desde la bandeja real | Lista y detalle desde navegador, tras reinicio; misma referencia y versión, sin otra alta. Publicado con instrucciones de arranque. | Cerrado: `b2effba` |
| 2 | Retomar la actuación pendiente desde ese expediente | El formulario existente recibe automáticamente expediente y versión; se conserva la recuperación manual ya publicada. Una actuación por corte. | Cerrado para análisis de solicitud v1: `b2effba`; no afirma recuperación de todas las actuaciones |
| 3 | Registrar la declaración de una respuesta recibida por RRHH | Actor, referencia y SHA256 declarados, vinculados al llamamiento y con justificante persistente; no verifica origen, firma o custodia, entrega de correo ni aceptación terminal. Recuperación con la misma clave y autorización vigente sin duplicados. | Incluido en esta entrega: `201` y recuperación `200` tras parche y segundo reinicio; conflicto `409` sin duplicado. Cerrado técnicamente |
| 4 | Registrar una aceptación válida | Permiso específico, comprobación competente de respuesta, plazo, justificante y estado, resolución y mismo recibo recuperable; la declaración del corte 3 no sustituye esa resolución. Habilita la propuesta de nombramiento solo tras aceptación válida. | Cerrado técnicamente solo para ejercicio manual sintético: `201` y replay `200` tras reinicio con API/V3/CT/Bolsa reales, sin duplicados. No política legal aprobada ni habilitación productiva |
| 5 | Registrar una renuncia válida | Respuesta y motivo conservados; deja de ofrecerse la aceptación de ese llamamiento. | Cerrado funcionalmente solo en ejercicio manual sintético: `201` y recuperación `200` tras reinicio, mismos recibo e intención pendiente, sin duplicados. No política legal aprobada |
| 6 | Resolver un vencimiento cuando corresponda | Solo con inicio y política de plazo acreditados; no calcula un plazo legal desde el aviso local. | Depende de evidencia y política |
| 7 | Continuar con la siguiente persona tras renuncia o vencimiento | Reutiliza el orden entregado por Bolsa y abre un único nuevo llamamiento; no crea otro motor de selección. | Continuación, aviso CT62, declaración CT63 y aceptación manual sintética CT64 recuperados tras reinicio principal: ocho `200`, recibos e historia intactos. Sin tercer llamamiento, envío, entrega ni vencimiento |
| 8 | Guardar y recuperar una propuesta de nombramiento | Parte de la aceptación real registrada; muestra datos, estado de propuesta y recibo, sin fingir nombramiento firmado. | Caso original y sucesor CT65 recorridos: `201` y replay `200` tras reinicio principal, misma propuesta/recibo/fecha/v7 en cada caso; solicitud conserva versión esperada `6`. Hash publicado comprobable en Git |
| 9 | Descargar los documentos de la propuesta | Un documento por corte, usando el generador existente; campos del expediente y descarga real. Véase desglose siguiente. | Cerrado funcionalmente en desarrollo: 6/6 borradores; diez POST `200` antes/después del reinicio principal, PDF e historial idénticos. Siguiente 10, sujeto a fuente/circuito de firma admitido. Sin firmas, envío, entrega, plazo legal, incorporación ni otro paso RRHH completo |
| 10 | Incorporar la resolución y su evidencia de firma o validación | Documento y estado vinculados al expediente según la autoridad admitida. Una firma pendiente no se presenta como completada. | Los seis borradores no acreditan firma o validación; depende del circuito admitido |
| 11 | Confirmar la incorporación | Fecha, centro y relación de personal conservados y recuperables; solo la integración mínima de Personal necesaria para contratación. | Acreditado funcionalmente tras reinicio, sin repetir el `POST`; no firma ni completa Contratación |
| 12 | Descargar la ficha para GINPIX | Fichero de incorporación utilizable para la grabación manual prevista; no exige construir la conexión automática. | Acreditado como ficha manual recuperada tras reinicio; no confirma GINPIX externo |
| 13 | Registrar seguimiento y cerrar el expediente | Una anotación y después el cierre, en cortes separados; historial y estado final conservados. | Cierre administrativo del seguimiento acreditado en principal: anotación y cierre recuperados tras reiniciar app/PostgreSQL, mismos recibos y una sola auditoría nueva. No es cese ni cierre jurídico del expediente |
| 14 | Entregar el recorrido completo a Alberto y RRHH | Arranque reproducible, ocho pasos recorribles, manuales al día y lista explícita de dependencias productivas. Una comprobación conjunta final. | Después de los anteriores |

**Documentos del objetivo 9: seis cortes, no un generador nuevo.**
Informe definitivo, resolución, diligencia, toma de posesión, notificación y comunicación al centro
ya descargables como borradores. Siguiente: circuito de firma/evidencia admitida (10). Sin modelo oficial
se entrega un borrador de desarrollo claramente marcado, no una redacción
jurídica validada ni un documento firmado.

El paso 6 no se declara completo por registrar aceptación y renuncia sintéticas:
faltan vencimiento y correo corporativo; la continuación tras renuncia sintética
tiene `201` real y recuperación `200` tras reinicio; su aviso local posterior también es recuperable,
la declaración del sucesor tiene `201` y recuperación `200` tras reinicio principal confirmada.
La octava operación confirma separadamente la aceptación manual sintética del sucesor con CT y Bolsa;
no hay envío corporativo ni plazo legal. La declaración no resuelve automáticamente.
La numeración de esta cola no sustituye los ocho pasos del procedimiento.

## Dependencias externas sin detener todo el desarrollo

Orden del operador: las preguntas pendientes no detienen la programación de
partes independientes. No autorizan a inventar plazo, identidad o permiso, ni
a convertir una solicitud de resolución pendiente en aceptación confirmada.

Correo corporativo, modelos oficiales, circuito de firma y conexión automática
a GINPIX requieren información o autorización externas. Se continúa únicamente
por alternativas reales permitidas: declaraciones registradas con sus límites
explícitos, documentos de desarrollo identificados y ficha de grabación manual. Si falta una
autoridad imprescindible, se señala la entrega exacta bloqueada y se trabaja
en otra entrega de contratación que no dependa de ella. No se simula el éxito.

La aplicación de desarrollo con datos sintéticos y la autorización para usar
datos reales en producción son cierres distintos.

## Historial conservado — no es el plan activo

<details>
<summary>Tablero anterior a este plan (julio–agosto de 2026)</summary>

**Última actualización:** 9 de agosto de 2026

**Frente principal:** completar las verticales reales de Bolsa y la adaptación
de Contratación temporal solicitada por RRHH.

**Frente técnico activo:** integración O4-05 de Contratación temporal. El
checkpoint interno O2b queda cerrado en `d86aea8`–`4b39265`, con doble `GO`,
`P0=P1=P2=0`, 31/31 mutantes, H0 PostgreSQL 18.4 y puerta global verdes. No
abre Bash ni modo operativo y no suma una capacidad funcional; está publicado
en `c1ca5aa`, CI `31287803830` 5/5. O3a-P1 queda publicado en la cadena
`758e66f` → `a1aeab7` → `f3a1e96` → `ce0848e`; la CI `31298943127`
terminó 5/5. Conserva material R-only de 702 líneas/SHA `7ad65a66…` y seis
evidencias durables.
La doble revisión final dio `GO`, `P0=P1=P2=0`, con matriz
100/600 + 10/10 + 1, H0 PostgreSQL 18.4, calidad global, Gitleaks y residuos
cero. El siguiente único trabajo es corregir y revisar el contrato O3a
completo sobre `ce0848e`; O3a,
`Start` y mapa FD siguen bloqueados hasta doble GO, publicación y CI 5/5 del
contrato.

Solo se usan datos sintéticos hasta cerrar las puertas de
autorización, trazabilidad y protección de datos.

**Objetivo actual:** cerrar el primer recorrido productivo de Contratación
temporal sin falsos adaptadores: HTTP protegido → C1/C2 corporativo → PDP →
servicios nominales → PostgreSQL → recibo. Bolsa conserva su prioridad
funcional y su trabajo ya verificado, pero no se infla su avance con contratos
aislados.

Este es el tablero de seguimiento para dirección. Se actualiza antes del commit
que cierre una capacidad o siempre que cambie el frente principal. Los códigos
internos como `T20` se conservan en la documentación técnica, pero no dirigen
este tablero.

Los objetivos, las siete puertas del primer hito y la relación sin
solapamientos con Bolsa se detallan en
[Objetivos y hoja de ruta del frente RRHH](docs/portal_vec/objetivos_y_hoja_ruta_rrhh_2026-07-23.md).
El orden, las dependencias y los carriles simultáneos están en el
[mapa revisable de objetivos y tareas](docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md).

### Cambio de frente del 23/07/2026

El documento recibido de RRHH exige un expediente coordinador nuevo. Se ha
creado `contrataciontemporal` sin borrar Bolsa. El primer hito lleva **2 de 7
puertas publicadas (29 %)**: dominio y caso de uso. La preparación idempotente
PostgreSQL ya está validada localmente, pero no contará como tercera puerta
hasta publicar sus commits. Después faltan confirmación con autorización VEC
durable, API, pantalla y E2E/aceptación. La tabla histórica de Bolsa que sigue
a continuación se conserva como referencia honesta de sus capacidades y no se
da por completada.

Los agentes adicionales deben seguir
[ORQUESTACION_AGENTES.md](ORQUESTACION_AGENTES.md); allí se indican tareas
ocupadas, trabajos libres, dependencias, límites de archivos y formato de
entrega.

### Corte operativo verificable del 30 de julio

Los porcentajes oficiales no incorporan código local, contratos aislados ni
pruebas parciales. Un corte solo aumenta el avance cuando supera PostgreSQL 18,
revisión independiente, pruebas aplicables, commit y conexión al recorrido
productivo.

| Ámbito | Avance oficial | Trabajo activo todavía no computado |
| --- | ---: | --- |
| Bolsa productiva de extremo a extremo | **1 de 14 capacidades (7 %)** | B1 Convoca y B2/T13 tienen GO técnico aislado; falta composición API/web y conectores productivos. |
| Primera vertical de contratación | **5 de 10 tareas (50 %)** | O2-06 permanece aparcada hasta terminar O4-05. |
| Contratación temporal | **24 de 46 tareas (52 %)** | O4-05 lleva 3/5 hitos. CT-000039 a CT-000047A, M1/M2, C1, C2.1a y C2.1b están cerrados aisladamente. C2.1b no se contabiliza por ser una frontera interna; faltan C2.2, PDP, composición, TLS/mTLS y E2E. |
| Presentación web | **Aproximadamente 90 % presentable** | No equivale a integración, aceptación de RRHH ni producción. |

La única capacidad de Bolsa contada de extremo a extremo es la consulta
pública: contrato, PostgreSQL 18, composición productiva y prueba técnica. La
raíz interna continúa cerrada por diseño hasta recibir todas sus dependencias
reales; no se sustituirán con indicadores booleanos ni adaptadores DEMO.

## Dónde estamos ahora

### Corte de presentación de Bolsa

La composición Docker de presentación permite enseñar **36 vistas** desde el
único acceso publicado por el proxy, `http://127.0.0.1:8081/presentacion/`.
Separa el portal `vec-presentacion`, el mediador
`vec-cartografia-presentacion` y el destino normal sin material DEMO, todavía
no autorizado ni compuesto para producción:

| Punto de vista | Vistas navegables | Alcance |
| --- | ---: | --- |
| Lanzador | 1 | Selección inequívoca de los cuatro puntos de vista DEMO. |
| Consulta pública | 1 | Convocatorias, categorías, plazos y ayuda sin identidad. |
| Área personal del aspirante | 14 | Inicio, convocatorias y detalle, perfil, méritos, solicitud, autobaremación, expediente, llamamientos, subsanaciones, alegaciones, mensajes, certificados y ayuda. |
| Gestión interna | 20 | Portal del Empleado, 17 secciones de gestión de Bolsa, Cronos y Dietas. |

Las pantallas, contratos y renderizadores se han diseñado como candidatos a
reutilización, sujetos a aceptación de RRHH y Sistemas; el
perfil de presentación inyecta adaptadores en memoria volátil. No emplea
cookies, `localStorage`, `sessionStorage` ni volumen durable. Dietas conecta un
mediador aislado a un grafo OSRM y teselas OSM internos, reales y versionados;
no hay salida a servicios cartográficos públicos. Esta entrega acredita una
**presentación navegable**, no integración
productiva, E2E productivo, identidad real, persistencia, firma, registro,
pagos o comunicaciones reales.

Existe además un perfil descartable `presentacion-remota` para enlazar la demo
con un frontal corporativo de otro equipo. Mantiene el acceso anterior en
loopback y añade una entrada aislada sobre una única IP interna, con ACL cerrada
por defecto y configuración operativa custodiada fuera de Git. No supone una
publicación productiva; requiere lista blanca coincidente en la red y en el
cortafuegos Docker del servidor.

La última línea base cerrada recorrió 36 vistas y 25 flujos en 1440×1000,
1024×900 y 390×844: **183 de 183 escenarios correctos, 183 capturas y cero
hallazgos**. Incluye los cuatro puntos de vista por mínimo privilegio, la ruta
OSM/OSRM real de Dietas, la carga efectiva de teselas servidas en la red
interna y recibos de las operaciones representativas. Los certificados de la
persona aspirante generan un PDF binario real de demostración, con identidad
institucional, referencia opaca y QR comprobado mediante un lector
independiente. Ninguna
de estas puertas sustituye la revisión humana ni la aceptación formal de RRHH.
Véanse la
[revisión automatizada](docs/portal_vec/revision_web_presentacion.md), el
[modo de presentación](docs/portal_vec/modo_presentacion_rrhh.md) y la
[matriz de aceptación](docs/portal_vec/matriz_aceptacion_web_bolsa_2026-07-18.md).
La operación cartográfica se documenta en
[Cartografía interna de Dietas](docs/portal_vec/cartografia_interna_dietas_2026-07-19.md).

**Primera funcionalidad real: paso 3 de 6.**

| Paso del objetivo actual | Estado | Evidencia o condición de cierre |
| --- | --- | --- |
| 1. Entorno seguro de desarrollo | ✅ Terminado | mTLS 1.3, identidad local de alta garantía, cifrado KMS y sello de tiempo de desarrollo, sin claves en Git. |
| 2. Diario durable y cifrado de borradores | ✅ Terminado | PostgreSQL, reintentos, recuperación, alias de distintas generaciones y borrado de memoria probados. |
| 3. Guardado transaccional e identificadores seguros | 🚧 Integración Go en curso | PostgreSQL/KMS y el identificador HMAC ya tienen `GO` independiente. Falta recorrer desde Go la transacción A/B real y verificar el recibo después del commit. |
| 4. Autorización y lectura gobernada | ⬜ Pendiente inmediato | Migrar la lectura a autorización V2 real y devolver el sobre cifrado completo sin acceso directo a tablas. |
| 5. Conexión definitiva con la web | ⬜ Pendiente | Registrar servicios y rutas reales bajo identidad mTLS, sin cookies, fixtures ni adaptadores falsos. |
| 6. Prueba completa y entrega manual | ⬜ Pendiente | Web → API → autorización → PostgreSQL/KMS → auditoría → respuesta; reinicio, concurrencia y fallos; después guía para que Alberto/RRHH lo prueben. |

## Carriles paralelos activos

| Carril | Trabajo actual | Puede avanzar sin bloquear al resto | Siguiente entrega |
| --- | --- | --- | --- |
| 1. Camino crítico | Adaptador Go→PostgreSQL y verificación poscommit | Sí; consume los contratos y funciones SQL ya revisados | Recorrido A/B real, reinicio y recibo verificado desde Go. |
| 2. Datos heredados | Composición productiva del importador de Convoca | Sí, exclusivamente con hojas sintéticas | Envolver recuperación con VEC/T13, custodia externa y protector KMS/HSM. |
| 3. Calidad y seguridad | Extender B2/T13 a cada wrapper | Sí; el revisor no modifica el código revisado | Registrar accesos permitidos y denegados de cada vertical. |
| 4. Orquestación ampliada | Ensayo de Orquesta antigua cerrado en `NO-GO` | No se amplía; la entrega rechazada quedó fuera del árbol principal | Corregir Convoca con agentes directos y repetir las puertas independientes. |
| 5. Dirección e integración | Tablero, pruebas globales y commits acotados | Sí | Integrar solo entregas revisadas y mantener una única verdad. |

**Siguiente carril que se abrirá al quedar uno libre:** composición API/web del
primer recorrido productivo, manteniendo cerrada la entrada de datos reales.

## Significado de las columnas

- **Contrato probado:** existe código real probado de forma aislada.
- **Integrado:** el servidor soportado registra la capacidad con sus
  dependencias reales; una pantalla o un test aislado no cuentan.
- **E2E técnico:** se ha probado el recorrido técnico de extremo a extremo.
- **Probable ahora:** Alberto o RRHH pueden recorrerlo manualmente. La marca
  `Presentación` o `DEMO` indica adaptadores sintéticos y ausencia de validez
  administrativa; una marca sin esos términos exige conectores reales.
- **Aceptado RRHH:** existe una prueba de aceptación formal registrada.
- **Producción:** está desplegado con infraestructura y conectores autorizados.

`E2E` significa *end to end* o «de extremo a extremo». Un E2E marcado `DEMO`
solo prueba el artefacto local y no cuenta como E2E productivo. `UAT` significa
pruebas de aceptación de usuario; aquí se escribe siempre **Aceptado RRHH**
para evitar la sigla.

## Tabla principal de capacidades

| Capacidad | Contrato probado | Integrado | E2E técnico | Probable ahora | Aceptado RRHH | Producción |
| --- | --- | --- | --- | --- | --- | --- |
| Consulta pública de convocatorias | ✅ | ✅ PostgreSQL | ✅ Técnico | ✅ DEMO y prueba técnica | ❌ | ❌ No desplegada |
| Panel interno agregado de Bolsa | ✅ | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Creación y edición de convocatorias | ✅ | 🚧 En curso | ❌ | 🧪 Presentación | ❌ | ❌ |
| Publicación, sustitución y retirada | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Bases y reglas de baremo | ✅ | ❌ | 🟡 Núcleo/BD | 🧪 Presentación | ❌ | ❌ |
| Autobaremación del aspirante | ✅ | 🧪 Legado | 🧪 Legado | 🧪 Presentación | ❌ | ❌ |
| Revisión técnica y rectificación firmada | ✅ | ❌ | 🟡 Aplicación/BD | 🧪 Presentación | ❌ | ❌ |
| Listas, ranking y desempates | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Llamamientos | ✅ | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Contratos, ceses y reincorporaciones | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Candidatura, solicitud y registro | 🟡 Parcial | 🧪 Legado | 🧪 Legado | 🧪 Presentación | ❌ | ❌ |
| Subsanaciones y alegaciones | 🟡 Parcial | 🧪 Legado | ❌ | 🧪 Presentación | ❌ | ❌ |
| Documentos, carga, cuarentena y antivirus | ✅ | ❌ | 🟡 Piezas aisladas | 🧪 Presentación | ❌ | ❌ |
| Firma, sello de tiempo, CSV/QR y cotejo | ✅ | ❌ | 🟡 Piezas aisladas | 🧪 Presentación | ❌ | ❌ |
| Generación y descarga de PDF/DOCX/ODT/CSV/JSON/etc. | ✅ | ❌ | 🟡 Renderizadores | 🧪 Presentación | ❌ | ❌ |
| Comunicaciones, correo, Telegram y notificación | 🟡 Parcial | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Tasas, pagos, devoluciones y conciliación | ✅ | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| Ayuda, audio y transcripción | ✅ | ✅ Estática | ✅ Estática | ✅ | ❌ formal | ❌ administrable |
| Bot de ayuda pública | 🟡 Diseño | ❌ | ❌ | ❌ | ❌ | ❌ |
| Identidad y separación público/interno | ✅ | 🟡 Desarrollo | ✅ Desarrollo | 🧪 Presentación | ❌ | ❌ |
| Roles, permisos y autorización por operación | ✅ | ❌ en Bolsa | 🟡 Piezas aisladas | 🧪 Presentación | ❌ | ❌ |
| Auditoría, recibos y registro de accesos | ✅ | 🟡 T13 aislado | 🟡 PostgreSQL 18 aislado | 🧪 Cronología | ❌ | ❌ |
| Catálogos, configuración y plazos administrables | 🟡 Parcial | 🧪 Consulta | 🧪 Consulta | 🧪 Parcial | ❌ | ❌ |
| Importación de datos de Convoca | ✅ | 🟡 PostgreSQL aislado | ✅ PostgreSQL 18 aislado | 🧪 Presentación | ❌ | ❌ |
| API pública | ✅ | 🧪 DEMO | 🧪 DEMO | ✅ DEMO | ❌ | ❌ |
| API interna completa | 🟡 Parcial | ❌ | ❌ | ❌ | ❌ | ❌ |
| CLI, MCP y acceso gobernado para IA | 🟡 Contratos | ❌ | ❌ | ❌ | ❌ | ❌ |
| Protección de datos, conservación y expurgo | ✅ Diseño/núcleo | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Copias, recuperación, observabilidad y operación | 🟡 Parcial | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Accesibilidad, tema y preferencias visuales | ✅ | 🟡 Vistas actuales | 🟡 Web | ✅ Parcial | ❌ formal | ❌ |

La tabla distingue trabajo reutilizable de funcionalidad utilizable. Por eso
puede haber `✅` en «Contrato probado» y `❌` en «Integrado» sin que exista una
contradicción.

## Plan de ataque funcional (julio de 2026, histórico: sustituido por la hoja de ruta del 16/09/2026)

| Orden | Entregable comprensible | Estado | Cuándo se considera terminado |
| --- | --- | --- | --- |
| 1 | Crear y editar convocatorias reales | 🚧 **Ahora** | RRHH puede crear, listar, abrir y modificar un borrador desde la web; persiste tras reinicio y deja autorización, cifrado, auditoría y recibo. |
| 2 | Poder usar datos reales en un piloto seguro | 🚧 T13 aislado, sin datos reales | Toda prueba y acceso queda durable; se registra quién accede, para qué y cuándo. |
| 3 | Importar la información existente de Convoca | 🚧 **Composición productiva**, sin datos reales | Parser y PostgreSQL 18 están cerrados; faltan VEC/T13, custodia externa, KMS/HSM y API/web. |
| 4 | Gestionar bases, reglas, puntuaciones y publicación | ⬜ Pendiente | RRHH configura las bases sin programar; se calculan puntuaciones y se publican actos aprobados y firmados. |
| 5 | Completar el expediente del aspirante | ⬜ Pendiente | Perfil, documentos, solicitud, autobaremación, registro, subsanación y alegaciones funcionan juntos. |
| 6 | Completar revisión técnica, listas y llamamientos | ⬜ Pendiente | RRHH revisa, firma, rectifica, genera listas y realiza llamamientos trazables. |
| 7 | Completar contratos, comunicaciones y pagos | ⬜ Pendiente | Ciclo posterior, notificaciones, respuestas, tasas, devoluciones y conciliación quedan conectados. |
| 8 | Validación formal y producción | ⬜ Pendiente | Alberto/RRHH aceptan una versión; Sistemas y seguridad autorizan conectores, despliegue, copias y operación. |

## Regla de actualización

Cada cierre debe actualizar, en este orden:

1. «Dónde estamos ahora».
2. La fila afectada de la tabla principal.
3. El plan de ataque si cambia el frente.
4. La fecha y el historial inferior.
5. Solo después, el commit de la funcionalidad.

## Historial de cambios de este tablero

| Fecha | Cambio |
| --- | --- |
| 26/07/2026 | O4-05 cierra su tercer hito interno en `023b890`: cliente HTTP productivo para alta y cobertura, manifiestos cerrados, DTO alineados con Go y bloqueo de reenvío ante resultado indeterminado. `cdea9cf` divide la prueba que superaba el tope y amplía la puerta de tamaño a `.mjs`. Supera 378/378 pruebas web, suite Go, `go vet`, carrera focal, manifiestos, Gitleaks y revisión independiente. La métrica permanece en Contratación 19/46 y Bolsa 1/14 hasta composición, recuperación y E2E. |
| 26/07/2026 | B1 Convoca y B2/T13 obtienen GO técnico independiente, se integran en la rama principal y pasan la regresión Go conjunta. Convoca queda probado en PostgreSQL 18/TLS con cifrado opaco, RLS, conservación y reversión; T13 registra consultas permitidas con finalidad y filtro exacto. No aumentan el porcentaje productivo hasta su composición API/web y conectores reales. O4-04D queda cerrado y O4-04E pasa a ser el siguiente corte de Contratación. |
| 26/07/2026 | Se fija la medida oficial: Bolsa 1/14 capacidades productivas E2E y Contratación temporal 18/46 tareas. Continúan sin computar B1 Convoca, B2/T13 y O4-04D hasta PostgreSQL 18, revisión cruzada, commit e integración. |
| 20/07/2026 | Presentación RRHH pulida y revisada: 183/183 escenarios, 183 capturas y cero hallazgos. Corregidos directorio público, foco del recibo de llamamiento, tablas operativas, huellas sintéticas, separación Reglas/Baremación y composición de Reglas a 1024 px. Certificados PDF DEMO reales con QR opaco verificable y selector de cuatro perfiles probado. |
| 19/07/2026 | Puerta cartográfica y visual cerrada: 174/174 escenarios correctos, 174 capturas y cero hallazgos sobre 36 vistas, 22 flujos y tres resoluciones; incluye ruta OSRM real y carga efectiva de teselas OSM internas. |
| 19/07/2026 | Composición Docker de presentación ampliada a portal, mediador cartográfico, OSRM y teselas OSM internas: un único acceso por `127.0.0.1:8081` y datos cartográficos versionados. |
| 19/07/2026 | Línea base anterior de la puerta automática: 159/159 escenarios correctos, 159 capturas y cero hallazgos sobre 32 vistas, 21 flujos y tres resoluciones. Queda pendiente la aceptación humana de RRHH y no cambia las columnas de integración, E2E productivo o producción. |
| 19/07/2026 | Presentación aislada de Bolsa cerrada: 32 vistas (1 lanzador + 1 pública + 14 del aspirante + 16 internas), adaptadores volátiles y cero cookies o almacenamiento de navegador. |
| 18/07/2026 | Piloto de Orquesta para Convoca cerrado en `NO-GO`: un proceso, cero revisiones y código no compilable. La auditoría bloquea datos reales hasta endurecer parser XLS, huella e invariantes. |
| 18/07/2026 | PostgreSQL/KMS y HMAC rotatorio obtienen `GO` independiente; comienza el recorrido durable Go→PostgreSQL y el ensayo aislado de Orquesta. |
| 18/07/2026 | Creación del tablero único; separación entre código probado, integración, E2E técnico, prueba manual, aceptación RRHH y producción. |

Para el detalle técnico y la brecha de cada fila se mantiene la
[matriz ampliada](docs/portal_vec/matriz_estado_operativo_bolsa_2026-07-18.md).

</details>
