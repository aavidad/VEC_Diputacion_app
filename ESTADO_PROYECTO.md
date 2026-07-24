# Estado y plan de ataque del proyecto

**Última actualización:** 24 de julio de 2026

**Frente principal:** primera vertical real del procedimiento de contratación
temporal solicitado por RRHH, complementaria a Bolsa.

**Frente conservado:** completar Bolsa conforme a esta misma tabla; el nuevo
módulo no elimina ni sustituye sus capacidades.

**Objetivo actual:** recorrer de forma real el alta de una solicitud de
contratación temporal desde el portal interno hasta PostgreSQL, con identidad,
autorización, idempotencia, auditoría, outbox y recibo.

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
creado `contrataciontemporal` sin borrar Bolsa. El primer hito lleva **3 de 7
puertas cerradas (43 %)**. Confirmación, API y web tienen piezas reales
implementadas; O2-06 y O2-07 son candidatos verificados sobre PostgreSQL 18,
pero no se contabilizan como cerrados hasta superar revisión independiente y
el E2E. La tabla histórica de Bolsa que sigue a continuación se conserva como
referencia honesta de sus capacidades y no se da por completada.

Los agentes adicionales deben seguir
[ORQUESTACION_AGENTES.md](ORQUESTACION_AGENTES.md); allí se indican tareas
ocupadas, trabajos libres, dependencias, límites de archivos y formato de
entrega.

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
| 2. Datos heredados | Importador de Convoca | Sí, exclusivamente con hojas sintéticas | Endurecer el parser XLS, las invariantes y la huella antes de PostgreSQL. |
| 3. Calidad y seguridad | Supervisión independiente de cada entrega | Sí; el revisor no modifica el código revisado | GO o defectos reproducibles antes de cada commit. |
| 4. Orquestación ampliada | Ensayo de Orquesta antigua cerrado en `NO-GO` | No se amplía; la entrega rechazada quedó fuera del árbol principal | Corregir Convoca con agentes directos y repetir las puertas independientes. |
| 5. Dirección e integración | Tablero, pruebas globales y commits acotados | Sí | Integrar solo entregas revisadas y mantener una única verdad. |

**Siguiente carril que se abrirá al quedar uno libre:** durabilidad probatoria y
registro de accesos, necesarios antes de introducir datos personales reales.

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
| Consulta pública de convocatorias | ✅ | 🧪 DEMO | 🧪 DEMO | ✅ DEMO | ❌ | ❌ |
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
| Auditoría, recibos y registro de accesos | ✅ | ❌ completo | 🟡 Piezas aisladas | 🧪 Cronología | ❌ | ❌ |
| Catálogos, configuración y plazos administrables | 🟡 Parcial | 🧪 Consulta | 🧪 Consulta | 🧪 Parcial | ❌ | ❌ |
| Importación de datos de Convoca | 🟡 Base sintética; auditoría `NO-GO` | ❌ | ❌ | 🧪 Presentación | ❌ | ❌ |
| API pública | ✅ | 🧪 DEMO | 🧪 DEMO | ✅ DEMO | ❌ | ❌ |
| API interna completa | 🟡 Parcial | ❌ | ❌ | ❌ | ❌ | ❌ |
| CLI, MCP y acceso gobernado para IA | 🟡 Contratos | ❌ | ❌ | ❌ | ❌ | ❌ |
| Protección de datos, conservación y expurgo | ✅ Diseño/núcleo | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Copias, recuperación, observabilidad y operación | 🟡 Parcial | ❌ integral | 🟡 Pruebas parciales | ❌ | ❌ | ❌ |
| Accesibilidad, tema y preferencias visuales | ✅ | 🟡 Vistas actuales | 🟡 Web | ✅ Parcial | ❌ formal | ❌ |

La tabla distingue trabajo reutilizable de funcionalidad utilizable. Por eso
puede haber `✅` en «Contrato probado» y `❌` en «Integrado» sin que exista una
contradicción.

## Plan de ataque funcional

| Orden | Entregable comprensible | Estado | Cuándo se considera terminado |
| --- | --- | --- | --- |
| 1 | Crear y editar convocatorias reales | 🚧 **Ahora** | RRHH puede crear, listar, abrir y modificar un borrador desde la web; persiste tras reinicio y deja autorización, cifrado, auditoría y recibo. |
| 2 | Poder usar datos reales en un piloto seguro | ⬜ Después | Toda prueba y acceso queda durable; se registra quién accede, para qué y cuándo. |
| 3 | Importar la información existente de Convoca | 🚧 **Corrección de seguridad**, sin datos reales | Parser aislado y estricto, importación repetible, validada, con incidencias, procedencia y sin duplicados. |
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
