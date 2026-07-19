# Inventario y retirada incremental de la presentación RRHH

Fecha de corte: 19 de julio de 2026.

## Objetivo y regla de decisión

Este inventario permite retirar todo el material operativo de demostración
cuando RRHH acepte los recorridos, sin perder la web candidata a producto. La
retirada no se hará de una vez ni mediante una búsqueda y borrado indiscriminado
de las palabras `demo` o `presentacion`.

La frontera es esta:

- se conservan las vistas, contratos, validadores, tema, componentes, ayudas,
  accesibilidad, presentadores, router y coordinación entre módulos;
- se sustituyen los datos, identidades, efectos, recibos y documentos locales
  por adaptadores productivos detrás de sus puertos;
- se eliminan el lanzador, el binario, el perfil, las guardas, las rutas y el
  artefacto exclusivo de presentación cuando ninguna superficie dependa ya de
  ellos;
- una capacidad solo deja de usar DEMO después de acreditar su vertical real;
- nunca se migran a producción estados, referencias o recibos `DEMO-*`.

El estado actual no acredita que existan todos los conectores reales. En las
tablas, «sustituto productivo» describe el puerto o la integración que debe
existir; cuando todavía no está integrada se indica expresamente como
**pendiente**.

Los antecedentes son el [modo de presentación RRHH](modo_presentacion_rrhh.md),
el [entregable funcional](entregable_rrhh_bolsa_2026-07-17.md), la
[matriz de aceptación](matriz_aceptacion_web_bolsa_2026-07-18.md), la
[revisión visual automatizada](revision_web_presentacion.md) y la
[integración del Portal, Cronos y Dietas](integracion_portal_cronos_dietas_2026-07-19.md).

## Significado de las decisiones

- **Eliminar al sustituir**: borrar el componente operativo y su prueba
  exclusiva cuando la vertical productiva cumpla su aceptación.
- **Eliminar al cierre**: mantenerlo mientras sirva para presentar alguna
  vertical; borrarlo cuando todas estén sustituidas.
- **Conservar**: forma parte de la web final. Se eliminan únicamente sus ramas
  o textos específicos de presentación, si los contiene.
- **Conservar como prueba negativa**: no es un proveedor DEMO; impide que una
  composición normal acepte material DEMO. Debe seguir existiendo aunque se
  retire la muestra.
- **Mover a `testdata` si aporta valor**: los datos sintéticos mínimos pueden
  sobrevivir como fixtures no desplegables. No deben conservarse como rutas,
  adaptadores operativos o valores predeterminados de producción.

## Inventario por ruta y fichero

| Pieza DEMO | Por qué existe | Eliminar o conservar | Sustituto productivo/puerto | Condición de aceptación | Prueba de ausencia en artefacto |
| --- | --- | --- | --- | --- | --- |
| `cmd/vec-presentacion/main.go` | Punto de entrada exclusivo, no autoritativo y local. | **Eliminar al cierre** junto con su construcción. | Raíces públicas e internas reales, separadas y compuestas por los binarios autorizados. No se declara aquí que esa composición esté terminada. | Todas las superficies aceptadas arrancan sin el binario DEMO y fallan cerradas si falta una dependencia. | El inventario de la imagen no contiene `usr/local/bin/vec-presentacion`; lo comprueba `scripts/verificar_contenido_artefactos_presentacion.sh`. |
| `scripts/arrancar_presentacion_rrhh.sh` | Fija perfil, memoria, fuentes sintéticas y dos guardas para la revisión local. | **Eliminar al cierre**. | Manual y automatización de despliegue de cada superficie real. **Pendiente de decisión de Sistemas**. | Existe arranque reproducible del producto con identidad y proveedores reales, sin variables DEMO. | `rg --files` y el inventario del paquete de despliegue no contienen el lanzador; no se copia a la imagen actual, pero debe comprobarse también en el paquete final. |
| `scripts/smoke_presentacion_rrhh.sh` | Prueba el artefacto DEMO, sus rutas y sus barreras. | **Eliminar al cierre** o archivar fuera del circuito de despliegue; migrar sus aserciones reutilizables al E2E productivo. | Smoke de superficies pública, personal e interna con dependencias de prueba equivalentes a las reales. **Pendiente**. | El nuevo smoke acredita rutas, autenticación/autorización, persistencia y fallos cerrados; no espera textos ni recibos DEMO. | `rg -n 'presentacion_rrhh|DEMO-' scripts` solo devuelve documentación histórica o fixtures de prueba expresamente permitidos. El script actual tiene expectativas de catálogo que pueden quedar obsoletas y no debe usarse como evidencia productiva. |
| `scripts/capturar_presentacion_web.py`, `scripts/revision_web/` y `scripts/tests/test_capturar_presentacion_web.py` | Recorren visualmente la muestra, exigen su cabecera y generan capturas sin persistencia del navegador. | **Conservar la infraestructura**, retirar el manifiesto/cabecera exclusivos y renombrarla como revisión web productiva; eliminar pruebas solo DEMO. | Capturador contra las URL reales, con identidades sintéticas autorizadas de entorno de pruebas y matriz de roles. | Recorre las vistas aprobadas en escritorio, tableta y móvil, sin errores, fugas, desbordamientos ni acciones fuera de permiso. | El código productivo de revisión no contiene `X-VEC-Modo-Presentacion` ni `?presentacion=rrhh`; los fixtures quedan fuera de imagen. |
| `data/demo/convocatorias_publicas.demo.json` | Fuente local de metadatos públicos y plazos/estado sintéticos. | **Eliminar al sustituir**; si se reutiliza una muestra mínima, mover a `testdata/`. | Puerto de consulta pública alimentado por el repositorio/publicador gobernado o importador autorizado. **Integración productiva pendiente**. | Procedencia, versión, vigencia, publicación y documentos oficiales están contrastados; no se inventan plazos. | La imagen no contiene `data/demo/` ni `.demo.json`; el verificador actual lo exige. |
| `data/catalogos/categorias-profesionales/v1.demo.json` | Catálogo local para que la consulta pública funcione sin repositorio real. | **Eliminar al sustituir**; fixture mínimo solo en `testdata/` si se necesita. | Puerto de catálogos versionados y gobernados. **Adaptador productivo pendiente de acreditar**. | Catálogo institucional aceptado, versionado, con integridad y ciclo de publicación. | Inventario de imagen sin `.demo.json`; añadir además una comprobación explícita de esta ruta al verificador. |
| `docs/demo/fuentes_publicas_bolsa.md` | Conserva la procedencia y explica qué parte de la muestra deriva del BOP. | **Conservar como evidencia documental**, no desplegar; actualizar o trasladar a un registro de procedencia definitivo. | Registro de fuentes de autoridad y evidencias del importador/publicador. | Cada registro público real puede trazarse a fuente oficial, revisión y versión. | La documentación no debe estar en la imagen; la copia cerrada del `Dockerfile` ya no copia `docs/`. |
| `web/static/presentacion/index.html`, `presentacion.css` y `presentacion.test.mjs` | Selector de puntos de vista exclusivo de la muestra. | **Eliminar al cierre**. | Portadas pública e interna definitivas; la elección de superficie se resuelve por frontera de red y autenticación, no por un selector DEMO. | RRHH acepta accesos y navegación definitivos, incluidos retorno por logotipo y separación público/interno. | No existe `app/web/static/presentacion/`; el script de contenido ya lo comprueba. |
| `web/static/area-personal/adaptador-presentacion.js` y su prueba | Mantienen en memoria perfil, méritos, solicitud, autobaremo, llamamientos, alegaciones y recibos del aspirante. | **Eliminar al sustituir**; trasladar fixtures mínimos a `testdata/` si conservan valor. | `cliente-http.js` contra los casos de uso privados de área personal, con identidad, autorización, idempotencia, persistencia, auditoría y recibos. Varias verticales siguen **pendientes**. | Cada acción visible devuelve un resultado real autorizado o falla cerrada; recargar conserva el estado confirmado; las pruebas cubren titularidad. | La imagen no contiene `area-personal/adaptador-presentacion.js`; el verificador actual detecta `presentacion` y debe mantener la ruta explícita. |
| `web/static/area-personal/arranque.js` | Selecciona por `?presentacion=rrhh` entre el adaptador local y el cliente HTTP. | **Conservar el fichero**, retirar la rama e importación DEMO cuando toda el área personal esté cableada. | Composición única con `crearClienteHTTPAreaPersonal()` y puerto documental real. | Ninguna URL o parámetro puede activar un proveedor alternativo; el cliente normal falla cerrado. | `rg -n 'presentacion=rrhh|adaptador-presentacion|descarga-recibos-presentacion' web/static/area-personal/arranque.js` no devuelve coincidencias. |
| `web/static/area-personal/aplicacion.js`, `index.html` y `vistas/*.js` | Son la aplicación y vistas reutilizables, pero contienen avisos, etiquetas y bifurcaciones visuales DEMO. | **Conservar**. Retirar solo ramas sintéticas, campos `recorrido_demo` y textos DEMO después de sustituir sus puertos. | Los mismos contratos y renderizadores alimentados por proyecciones reales; servicio documental para recibos. | Las 14 vistas aceptadas siguen disponibles; no se puede forzar plazo, firma, pago o registro desde el navegador. | Pruebas de superficie sin `?presentacion`; `rg` no encuentra importaciones DEMO ni acciones «simular». No se exige borrar validaciones de seguridad. |
| `web/static/area-personal/contrato.js` y `cliente-http.js` | Definen el contrato común y el cliente normal; contienen el indicador que distingue una respuesta sintética. | **Conservar**, incluida la validación que rechaza orígenes inesperados. | Ya son el límite frontend; deben apuntar a endpoints reales cuando sus verticales existan. | Pruebas de contrato y API reales pasan con `presentacion=false` y rechazan respuestas DEMO. | No es correcto exigir cero menciones a `presentacion`: la prueba negativa puede permanecer. La evidencia es una prueba que rechaza `presentacion=true`. |
| `web/static/portal-empleado/datos-presentacion.js` | Agregado interno sintético para todas las pantallas de Bolsa. | **Eliminar por grupos de capacidad**; no conservar como respaldo automático. | Proyecciones autorizadas de gobierno de convocatorias, bases, revisión, llamamientos, documentos, comunicaciones, roles y auditoría. **No todas existen hoy**. | Cada lista/capacidad procede de su caso de uso real, con ámbito, finalidad y fecha de corte; el panel no inventa una respuesta ante error. | Ausente de la imagen y sin importación dinámica en `portal.js`; ambas comprobaciones deben pasar. |
| `web/static/portal-empleado/portal-presentacion-adaptador.js` | Simula operaciones internas y emite recibos volátiles. | **Eliminar al sustituir**. | Puertos de comando de cada vertical, autorizados en servidor, transaccionales, idempotentes y auditados. **Pendientes según capacidad**. | Cada botón tiene una vertical real probada; el éxito solo se muestra tras recibo durable. | Ausente de la imagen; `rg -n 'portal-presentacion-adaptador' web/static/portal-empleado/portal.js` sin coincidencias. |
| `web/static/portal-empleado/portal-borradores-demo-cliente.js` | Simula el cliente de borradores detrás del contrato de borradores. | **Eliminar al sustituir**; conservar el contrato y la UI. | `portal-borradores-api.js` y el adaptador servidor del repositorio de borradores. La mera presencia de código PostgreSQL no acredita despliegue. | Crear, leer y actualizar borrador con control de versión, identidad, autorización y persistencia tras reinicio. | Ausente de imagen y sin importación desde `portal.js`; pruebas del cliente HTTP no admiten fallback local. |
| `web/static/portal-empleado/portal-borradores-fixtures.test-helper.mjs` | Fixture sintético exclusivo de pruebas de contrato/UI. | **Conservar solo como prueba** si nunca se despliega y no imita credenciales reales. | No necesita sustituto de runtime. | La fixture valida contratos sin convertirse en proveedor operativo. | Su extensión de prueba no entra en un paquete web mínimo; el verificador debe comprobar el manifiesto real del runtime, no solo el nombre. |
| `web/static/portal-empleado/identidad/presentacion.js` | Fabrica un `ContextoActor` a partir del perfil sintético de la muestra. | **Eliminar al sustituir**. | Proveedor de `ContextoActor` procedente de la frontera autenticada interna; identidad real aún depende de Sistemas. | Actor, mecanismo, capacidades y ámbito se resuelven y revalidan en servidor; no se aceptan perfil o rol de query. | Ausente de imagen y sin importación desde el coordinador. |
| `web/static/portal-empleado/identidad/contexto-actor.js` | Contrato inmutable compartido por Bolsa, Cronos y Dietas. | **Conservar**. | El proveedor productivo entrega el mismo contrato. | Una única identidad de sesión se comparte por referencia y toda capacidad ausente se deniega. | Pruebas de identidad compartida pasan; no debe buscarse su eliminación. Véase su `README.md`. |
| `web/static/portal-empleado/portal-catalogo-presentacion.js` | Hace visibles Bolsa, Cronos y Dietas sin consultar el registro real. | **Eliminar al sustituir**. | `portal-catalogo-modulos.js` mediante `/api/vec/modules` o el puerto de catálogo de módulos autorizado. La disponibilidad productiva debe verificarse. | Catálogo real devuelve solo módulos habilitados para actor, entorno y despliegue; falla cerrado. | Ausente de imagen y de `portal-modulos-coordinador.js`; prueba de navegación contra catálogo real. |
| `web/static/portal-empleado/modulos/cronos/datos-presentacion.js` y `adaptador-presentacion.js` | Proporcionan fichajes, horario, permisos, efectos y recibos volátiles. | **Eliminar al sustituir**. | Consulta y comandos de Cronos descritos en `modulos/cronos/INTEGRACION.md`. **No existe aún aquí un adaptador productivo acreditado**. | Datos confinados a la red corporativa, identidad compartida, capacidades mínimas, persistencia, auditoría e idempotencia probadas. | Ausentes de imagen y sin importaciones en el coordinador; prueba de que `#cronos` falla cerrado o se oculta si el módulo no está conectado. |
| `web/static/portal-empleado/modulos/dietas/datos-presentacion.js` y `adaptador-presentacion.js` | Proporcionan comisiones, rutas, gastos y transiciones volátiles. | **Eliminar al sustituir**. | Puertos `obtenerDatos()` y `ejecutar(comando)` documentados en `modulos/dietas/INTEGRACION.md`. **Adaptador productivo pendiente**. | Validaciones monetarias y de ruta en servidor, autorización por ámbito, versión/idempotencia, recibo y auditoría. | Ausentes de imagen y sin importaciones en el coordinador; `#dietas` no usa datos locales. |
| `web/static/portal-empleado/documentos/recibo-pdf-presentacion.js` y `descarga-recibos-presentacion.js`, con sus pruebas | Generan y descargan en el navegador un PDF institucional con marca DEMO y QR local. | **Eliminar al sustituir**; conservar los descriptores documentales de los módulos. | Puerto documental común hacia generación, firma/sello, custodia, CSV/QR y cotejo autorizados. **Conector productivo no acreditado todavía**. | El servidor entrega documento inmutable, huella, firma/sello, recibo de custodia y referencia opaca verificable; autorización de descarga probada. | Ausentes de imagen y sin importaciones desde área personal/coordinador; CSP y pruebas impiden generador local en producción. |
| `web/static/verificar/adaptador-presentacion.js` | Reconoce localmente referencias `DEMO-REC-*`, `DEMO-DIE-REC-*` y `DEMO-CRONOS-REC-*`. | **Eliminar al sustituir**. | `POST /api/publico/documentos/cotejo` respaldado por el registro documental real. **La llamada frontend existe; eso no acredita el servicio productivo**. | Cotejo real resuelve vigencia/revocación sin exponer datos personales ni aceptar referencias DEMO. | Ausente de imagen y sin importación dinámica desde `verificar.js`; prueba negativa para `DEMO-*`. |
| `web/static/verificar/verificar.js`, `index.html` y `verificar.css` | Interfaz reutilizable de cotejo; `verificar.js` contiene una rama por query para DEMO. | **Conservar**, retirar parámetro y rama de presentación. | Misma interfaz, exclusivamente contra el puerto público de cotejo. | Respuestas, errores, accesibilidad, limitación de datos y revocación aprobados. | `rg -n 'presentacion|adaptador-presentacion' web/static/verificar/verificar.js` sin coincidencias; pruebas contra API de ensayo. |
| `web/static/bolsa/documentos/bases-*-demo.html`, `bases-*-demo.pdf` y `bases-demo.css` | Reproducciones adaptadas para enseñar bases realistas sin servirlas como actos oficiales. | **Eliminar al sustituir**. Conservar, fuera del runtime, la procedencia pública si aporta evidencia. | Repositorio documental/publicador oficial: documento original o copia auténtica, metadatos, versión, firma, vigencia y descarga. **Pendiente de conexión productiva**. | La ficha de convocatoria enlaza el documento oficial correcto y su historial; autorización y caché no mezclan borrador/publicado. | Ninguna ruta `app/web/static/bolsa/documentos/*demo*` en imagen; ampliar la lista negativa del script para cubrir explícitamente HTML, PDF y CSS. |
| `scripts/generar_bases_demo_pdf.py` y su prueba | Regeneran los PDF adaptados de la muestra a partir de HTML. | **Eliminar al sustituir** o conservar solo como herramienta histórica fuera de distribución. | Pipeline documental institucional; no debe ser un generador local de actos administrativos. | PDFs oficiales se generan, firman, custodian y versionan por el servicio aprobado. | No aparece en imagen ni paquete de despliegue; `rg --files scripts \| rg 'generar_bases_demo'` vacío tras retirada. |
| `web/static/bolsa/index.html`, `bolsa.js`, `bolsa.css` y `bolsa_adaptable.css` | Portal público reutilizable; contiene banner, explicación, retorno al lanzador y estilos condicionales DEMO. | **Conservar**, retirar solo ramas y textos de muestra cuando las fuentes sean oficiales. | API pública real conserva su contrato de fuente y procedencia. | La fuente productiva declara explícitamente que no es demostración, las fechas tienen autoridad y los enlaces documentales son oficiales. | Pruebas públicas sin banner ni enlace `/presentacion/`; se conserva una prueba que rechace una fuente marcada DEMO en producción si corresponde. |
| `web/static/portal-empleado/portal.js` | Router/composición de Bolsa; importa datos, efectos y borradores DEMO cuando recibe query. | **Conservar**, retirar sus importaciones, perfil de query y ejecución local por capacidad a medida que se sustituyan. | Clientes HTTP y puertos de aplicación reales; sin fallback. | Las vistas aprobadas funcionan con proyección real y cada error deja la acción cerrada. | `rg -n 'datos-presentacion|portal-presentacion-adaptador|portal-borradores-demo|presentacion=rrhh' web/static/portal-empleado/portal.js` sin coincidencias al cierre. |
| `web/static/portal-empleado/portal-modulos-coordinador.js` | Coordinador definitivo que hoy compone identidad, catálogo, Cronos, Dietas y PDF de presentación. | **Conservar**, reemplazar `cargarPresentacion()` por composición interna real o eliminarla tras crear esa composición. | Proveedores reales del contexto, catálogo, Cronos, Dietas y documentos, inyectados por módulo. | `cargarInterno()` monta solo adaptadores autorizados; un módulo sin proveedor no aparece o falla cerrado. | Sin importaciones de `*/presentacion.js`; pruebas del coordinador con dobles de puerto situados solo en tests. |
| `web/static/portal-empleado/portal-contrato.js` | Contrato común que también valida el agregado DEMO y, en modo normal, rechaza `origen.demostracion != false`. | **Conservar la validación normal y su rechazo**; retirar el esquema/listas exclusivas DEMO cuando dejen de usarse. | Contrato canónico de panel interno. | Pruebas garantizan que el envelope real no puede ser raw ni estar marcado como demostración. | No se exige cero menciones a `demostracion`: la aserción negativa es una defensa. Se elimina solo `validarPanelBolsa(..., true)` y el esquema de presentación. |
| `web/static/portal-empleado/portal-llamamientos-flujo.js` y contrato asociado | Flujo reutilizable con una bifurcación que obtiene una propuesta local DEMO. | **Conservar**, retirar la rama local y el validador del esquema de presentación. | `portal-llamamientos-api.js` y `POST /api/vec/bolsa/propuestas-llamamiento`. La disponibilidad real debe probarse. | Elegibilidad y prelación se calculan en servidor, con versión, motivo, autorización e idempotencia. | Pruebas obligan a usar el cliente HTTP y rechazan `demostracion=true`; sin `obtenerPresentacion`. |
| `web/static/portal-empleado/portal-vistas-*.js`, `portal-eventos.js`, `portal-inicio.js`, `portal-menu-bolsa.*`, CSS y `index.html` | Vistas, shell y tema candidatos a producto; algunos textos, campos, identificadores y botones están rotulados DEMO. | **Conservar**. Sustituir texto y campos sintéticos por propiedades del contrato real, sin reescribir las pantallas. | Proyecciones y comandos de las verticales reales. | RRHH acepta contenido y jerarquía; todos los controles están cableados o deshabilitados de forma honesta, nunca simulados. | Revisión visual productiva y matriz pantalla/puerto; `rg` no encuentra operaciones «simular» ni IDs `DEMO-` fuera de pruebas negativas. |
| `web/static/portal-empleado/portal-demo-total.test.mjs` y pruebas exclusivas de adaptadores de presentación | Protegen la integridad de la muestra mientras existe. | **Eliminar al cierre**; migrar casos de accesibilidad, router y contratos a pruebas productivas antes de borrarlas. | Pruebas unitarias, de contrato e integración sobre las mismas vistas con dobles ubicados en tests. | La cobertura reutilizable no disminuye y las nuevas pruebas no importan runtime DEMO. | Manifiesto de pruebas sin suites de presentación; el runtime nunca debe incluir pruebas, con independencia del nombre. |
| `config/presentacion.go` y campos `RRHHPresentation*`/variables `VEC_RRHH_PRESENTATION_*` en `config/config.go` | Definen perfil y doble guarda para que la muestra no se active por error. | **Eliminar al cierre**, después del binario y rutas. | Ninguno: producción mantiene sus propios perfiles y configuración cerrada. | Ninguna raíz normal referencia esos campos y las configuraciones desconocidas fallan cerradas. | `rg -n 'RRHHPresentation|VEC_RRHH_PRESENTATION|presentacion_rrhh' --glob '*.go' --glob '*.yml' --glob '*.sh'` solo devuelve, en su caso, migraciones/documentación histórica. |
| `internal/app/bootstrap/presentacion.go` y sus pruebas | Composición mínima que prohíbe conectores, exige memoria y fuentes `.demo.json`; las composiciones normales rechazan sus selectores. | **Eliminar la composición al cierre**. Conservar o reubicar pruebas negativas generales que impidan mezclar perfiles. | Bootstraps públicos e internos reales. | Cada raíz declara sus proveedores, no admite selectores DEMO y cuenta con pruebas de mezcla imposible. | Sin `NewHTTPServerPresentacionWithConfig`; pruebas normales siguen fallando ante configuración desconocida o material sintético. |
| `internal/app/server/presentacion_marca.go`, ramas `NewHTTPServerPresentacion`/`NewHandlerPresentacionWithConfig`/`registrarDirectorioPresentacion` y sus pruebas | Allowlist, cabecera técnica, restricción local, solo lectura y rutas de la muestra. | **Eliminar al cierre**. Conservar en los handlers reales las cabeceras de seguridad, límites y listas positivas generales. | Handlers público e interno definitivos. | Cada superficie tiene lista positiva y frontera propias; ninguna sirve rutas DEMO. | `rg -n 'New.*Presentacion|Modo-Presentacion|registrarDirectorioPresentacion' internal/app/server` sin coincidencias, y las pruebas reales de rutas siguen pasando. |
| `staticHandler(false)` y `rutaMaterialExclusivoPresentacion` en `internal/app/server/server.go` | Impiden que servidores normales sirvan segmentos cuyo nombre contiene `presentacion` o `demo`. | **Conservar como prueba negativa durante la migración**; tras borrar todo DEMO puede sustituirse por un manifiesto estático positivo, que es más preciso. | Allowlist/manifiesto de recursos del artefacto web real. | El empaquetado enumera solo recursos aprobados y una prueba intenta pedir antiguas rutas DEMO obteniendo `404`. | Requests a la lista histórica devuelven `404`; inventario físico también limpio. No retirar la defensa antes que los archivos. |
| `Dockerfile`: construcción de `vec-presentacion`, copias `web-presentacion`/`web-produccion`, podas por nombre y destino `runtime-presentacion` | Produce dos artefactos físicamente separados. | **Eliminar al cierre** la construcción/destino DEMO. **Conservar y endurecer** la construcción cerrada del runtime real. | Una única construcción por superficie productiva con manifiesto positivo de archivos. | La imagen real no depende de poda por patrón para ser segura, no compila el binario DEMO y contiene solo recursos necesarios. | Inventario de imagen limpio y SBOM/manifiesto esperado; el script se adapta al único runtime real. |
| `docker-compose.yml`: servicios/perfil `presentacion`, red y variables DEMO | Ejecuta la muestra aislada tras proxy local. El perfil `local` también monta hoy los dos `.demo.json`. | **Eliminar al cierre** el perfil DEMO. En `local`, reemplazar esos montajes por fixtures de desarrollo bajo `testdata` o fuentes de ensayo nombradas sin ambigüedad; nunca tratarlas como producción. | Despliegues de desarrollo/ensayo y producción documentados por Sistemas. | Ningún servicio productivo monta `data/demo`, deshabilita autenticación ni usa memoria para actos. | `docker compose config` del perfil productivo sin guardas, rutas `.demo.json`, `AuthMode=disabled` ni `StorageMode=memory`. |
| `scripts/verificar_contenido_artefactos_presentacion.sh` | Demuestra separación física entre ambos artefactos. | **Conservar y transformar** en verificador de contenido productivo; retirar las aserciones positivas del artefacto DEMO al eliminarlo. | Puerta de CI sobre imagen, manifiesto/SBOM y lista negativa histórica. | Falla ante cualquier ruta antigua, `DEMO-*`, binario, guardas o datos sintéticos; se ejecuta en cada publicación. | El propio script devuelve cero y el `grep` negativo sobre el inventario no produce salida. |
| `internal/candidate/adapters/handler/demo.go` y `NewDemoAPI*` asociados | Demo heredada del procedimiento de candidato; no es el adaptador web de la presentación RRHH, pero contiene fixtures y ruta ejecutable sintética. | **Auditar y retirar por separado**; no debe olvidarse por estar fuera de `web/static`. Conservar dominio/casos de uso, no el runner ni sus fixtures. | Endpoints reales de procedimiento con repositorios y autorización; pruebas con fixtures en `*_test.go`/`testdata`. | Ninguna ruta de runtime ejecuta `handleDemo`; los casos de uso conservan cobertura sin generar 138 solicitudes sintéticas en servicio. | `rg -n 'NewProcedureDemoRunner|handleDemo|NewDemoAPI' cmd internal --glob '!**/*_test.go'` sin coincidencias y el inventario no incluye un binario que exponga esa ruta. |

## Elementos que forman la web definitiva

Estos elementos no se deben borrar al retirar la presentación:

| Capa reutilizable | Rutas principales | Tratamiento |
| --- | --- | --- |
| Tema y carcasa | `web/static/styles.css`, `web/static/portal-empleado/portal*.css`, `web/static/area-personal/area-personal.css`, `web/static/bolsa/bolsa*.css` | Conservar variables, diseño adaptable, alto contraste, impresión y colores semánticos. Retirar solo selectores exclusivos de avisos DEMO que queden sin uso. |
| Shell y router interno | `portal-empleado/index.html`, `portal.js`, `portal-eventos.js`, `portal-inicio.js`, `portal-menu-bolsa.js`, `portal-modulos-coordinador.js` | Conservar navegación y montaje. Sustituir en la raíz de composición las dependencias, no las vistas. |
| Vistas administrativas de Bolsa | `portal-vistas-baremacion.js`, `portal-vistas-convocatorias.js`, `portal-vistas-gobierno.js`, `portal-vistas-operaciones.js`, `portal-vistas-utilidades.js`, `portal-panel-interno.js` | Conservar lo aceptado por RRHH. Cada acción debe asociarse a un puerto real o permanecer cerrada; eliminar rótulos y valores DEMO, no la estructura útil. |
| Área aspirante | `area-personal/aplicacion.js`, `contrato.js`, `cliente-http.js`, `calculo-autobaremo.js`, `flujo-solicitud.js`, `vistas/*.js` | Conservar contratos, flujo y renderizadores. El cálculo de navegador es estimativo; el resultado autoritativo debe venir del servidor. |
| Consulta pública | `bolsa/index.html`, `bolsa.js`, navegación, estilos y clientes públicos | Conservar separación anónima, filtros, categorías, ayuda y accesibilidad. Cambiar fuente y documentos. |
| Cotejo | `verificar/index.html`, `verificar.js`, `verificar.css` | Conservar formulario y presentación del resultado; eliminar exclusivamente el adaptador y query DEMO. |
| Contratos y denegación | `portal-contrato.js`, contratos de borradores y llamamientos, `identidad/contexto-actor.js`, contratos Cronos/Dietas | Conservar validadores, capacidades explícitas y rechazos de origen sintético. Una mención a `demostracion` en una prueba negativa no es «paja». |
| Cronos | `modulos/cronos/{contrato,presentador,vista,i18n,documentos}.js` y `cronos.css` | Conservar. Sustituir datos y ejecutor sintéticos conforme a su [guía de integración](../../web/static/portal-empleado/modulos/cronos/INTEGRACION.md). |
| Dietas | `modulos/dietas/{contrato,presentador,vista,i18n}.js` y `dietas.css` | Conservar. Sustituir solo el adaptador conforme a su [guía de integración](../../web/static/portal-empleado/modulos/dietas/INTEGRACION.md). |
| Ayuda | `portal-empleado/ayuda-contenido.js`, vistas de ayuda, transcripciones y activos aprobados | Conservar la experiencia y conectar un catálogo versionado. Verificar licencia, vigencia y accesibilidad de cada activo. |

## Migración incremental, sin gran salto

### 0. Congelar la referencia aprobada

1. Etiquetar el commit y la imagen que RRHH ha revisado.
2. Guardar la matriz pantalla → contrato → puerto → responsable.
3. Ejecutar y archivar la revisión visual y el inventario físico.
4. Registrar como defecto cualquier control que parezca real sin estar
   conectado; no aumentar la muestra mientras se migra.

### 1. Sustituir una vertical cada vez

El orden concreto depende de la prioridad funcional, pero cada vertical sigue
siempre la misma secuencia:

1. cerrar el contrato canónico, errores, capacidades y recibo;
2. probar el caso de uso sin HTTP;
3. implementar el adaptador productivo detrás del puerto;
4. probar repositorio, autorización, concurrencia, idempotencia y auditoría;
5. montar el adaptador solo en el perfil real correspondiente;
6. ejecutar pruebas verticales con datos sintéticos controlados en un entorno
   de ensayo, no con el runtime DEMO;
7. comparar la vista con la referencia aceptada;
8. retirar la rama DEMO de esa capacidad y actualizar este inventario.

No se cambia una pantalla para ocultar que el puerto falta. Hasta que la
vertical exista, el control se mantiene explícitamente no disponible o la
capacidad no se publica.

### 2. Orden recomendado de superficies

1. consulta pública, catálogo y documentos publicados;
2. identidad y contexto del área personal, seguido de lectura del expediente;
3. borrador, carga documental, firma, pago y registro de la solicitud;
4. panel interno y gobierno de convocatorias/bases;
5. revisión, baremación, alegaciones y publicación;
6. llamamientos, comunicaciones, contratos/ceses y recibos;
7. cotejo documental real;
8. Cronos y Dietas solo en la frontera interna correspondiente.

El orden no supone que esas integraciones estén disponibles ahora. Permite
mantener la muestra para las verticales pendientes mientras las ya migradas se
prueban exclusivamente por el camino real.

### 3. Retirada final del modo de presentación

Cuando ninguna vista dependa de proveedores sintéticos:

1. borrar adaptadores, datos, PDFs y selector DEMO;
2. retirar queries, perfiles y ramas de composición DEMO de los ficheros
   compartidos;
3. borrar binario, bootstrap, handler, cabecera, guardas y perfil Compose;
4. simplificar el `Dockerfile` para que no compile ni copie la muestra;
5. transformar el verificador en una puerta permanente del artefacto real;
6. conservar documentación de decisiones y procedencia, pero no servirla;
7. repetir unidad, integración, E2E, revisión visual, inventario físico y
   auditoría de secretos/datos antes de publicar.

## Condiciones mínimas por tipo de puerto

La palabra «real» no basta como criterio de sustitución. Como mínimo:

| Puerto | Condición para retirar su DEMO |
| --- | --- |
| Identidad | Principal proveniente de frontera autorizada, sesión/canal protegido, capacidades y ámbito revalidados en servidor, sin rol elegido por URL. |
| Consulta | Proyección mínima, autorización por objeto y finalidad, procedencia y fecha de corte, sin fallback local. |
| Comando | Validación de estado y versión, autorización, idempotencia, transacción, auditoría y recibo antes de comunicar éxito. |
| Persistencia | Durabilidad tras reinicio, concurrencia probada, migraciones y restauración; los datos DEMO no se importan. |
| Documentos | Carga segura, cuarentena/análisis, huella, cifrado, custodia, autorización, versionado, firma/sello y descarga trazable según el caso. |
| Cotejo | Referencia opaca, respuesta pública mínima, estado de vigencia/revocación y protección contra enumeración. |
| Comunicaciones | Plantilla/versionado, consentimiento o base jurídica, destinatario resuelto en servidor, reintentos, baja y recibo del proveedor. |
| Catálogos | Gobierno, versión, vigencia, integridad, publicación y compatibilidad; nunca listas ocultas fijadas en una vista. |

## Marcha atrás segura

La retirada será reversible por versión, no reactivando DEMO dentro de
producción:

1. conservar la imagen/commit anterior aprobado por cada hito;
2. desplegar cambios de esquema compatibles hacia atrás antes de cambiar la
   aplicación;
3. activar una vertical real mediante composición del entorno, no mediante un
   selector enviado por el navegador;
4. si falla, volver a la imagen productiva anterior o deshabilitar el módulo;
5. nunca usar el adaptador DEMO como fallback, ni copiar sus datos a la base
   real, ni habilitar las guardas de presentación en producción;
6. no borrar una fixture operativa hasta cerrar el periodo de observación de su
   sustituto, pero mantenerla físicamente fuera de todo artefacto productivo;
7. borrar definitivamente el modo de presentación solo tras un release estable
   y una copia etiquetada de su evidencia documental.

## Verificación reproducible

### Inventario físico existente

Mientras coexistan ambos artefactos:

```bash
scripts/verificar_contenido_artefactos_presentacion.sh
```

El script actual construye `runtime` y `runtime-presentacion`, exporta ambos y
falla si producción contiene:

```text
app/web/.*presentacion
data/demo/
.demo.json
usr/local/bin/vec-presentacion
```

La puerta debe ampliarse con una lista negativa histórica antes de la retirada,
porque un nombre nuevo podría ser sintético sin contener esas palabras. Como
mínimo debe cubrir los adaptadores, PDFs, rutas, guardas, referencias `DEMO-*`
y el runner heredado inventariados en este documento.

### Auditoría del árbol fuente

Esta búsqueda localiza candidatos; no decide por sí sola qué borrar:

```bash
rg -n -i \
  'presentacion_rrhh|VEC_RRHH_PRESENTATION|\?presentacion=rrhh|adaptador-presentacion|\.demo\.json|DEMO-' \
  cmd config internal scripts web Dockerfile docker-compose.yml
```

Cada coincidencia se clasifica como proveedor operativo, rama a retirar,
fixture de prueba o rechazo de seguridad. Las dos últimas categorías no se
borran automáticamente.

### Evidencia final de ausencia

La retirada se acepta solo si se cumplen simultáneamente:

1. el inventario de la imagen productiva no contiene ninguna ruta de la tabla
   marcada «eliminar»;
2. ninguna raíz de composición importa o construye un proveedor DEMO;
3. pedir cada antigua ruta devuelve `404` sin redirección a una muestra;
4. ninguna variable o query puede activar el modo retirado;
5. los tests productivos rechazan datos con `demostracion=true` y referencias
   `DEMO-*`;
6. la revisión visual confirma que las vistas conservadas no han perdido menú,
   tema, accesibilidad ni acciones reales;
7. cada acción visible aparece en la matriz pantalla/puerto y cuenta con prueba
   de aceptación o permanece explícitamente deshabilitada;
8. el paquete no contiene documentación, capturas, fixtures, secretos ni datos
   personales.

`scripts/smoke_local_productizable.sh` puede ayudar durante el desarrollo, pero
es una prueba local heredada y no acredita por sí sola ninguna condición
productiva de esta retirada.

## Registro de avance

La migración debe actualizar esta tabla en cada integración:

| Vertical | Adaptador DEMO activo | Adaptador productivo integrado | Aceptación funcional | Ausencia física comprobada | Fecha/evidencia |
| --- | --- | --- | --- | --- | --- |
| Consulta pública y catálogos | Sí | No acreditado en este documento | Pendiente | Sí en `runtime` actual por poda, no retirada del fuente | — |
| Área personal | Sí | Cliente HTTP preparado; verticales completas no acreditadas | Pendiente | Sí en `runtime` actual por poda | — |
| Bolsa interna | Sí | Parcial; no todas las capacidades están integradas | Pendiente | Sí en `runtime` actual por poda | — |
| Documentos/recibos/cotejo | Sí, local | No acreditado | Pendiente | Sí en `runtime` actual por poda | — |
| Cronos | Sí | No acreditado | Pendiente | Sí en `runtime` actual por poda | — |
| Dietas | Sí | No acreditado | Pendiente | Sí en `runtime` actual por poda | — |

«Ausente del `runtime` actual» significa únicamente separación física del
artefacto de presentación. No significa que la vertical productiva exista ni
que pueda declararse terminada.
