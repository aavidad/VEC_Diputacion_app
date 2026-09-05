# Estado y plan de ataque del proyecto

**Plan vigente: 5 de septiembre de 2026. Prioridad exclusiva: Contratación temporal.**

Base publicada del corte actual: `4034157b0e09ce9e1827c503cacd85ea7c894c01`.

Este es el único plan operativo. El historial inferior se conserva como
referencia; sus porcentajes, carriles y órdenes antiguos no dirigen el trabajo.
Bolsa y los demás módulos quedan fuera, salvo la pieza mínima que necesite
contratación. No se reabren los cinco primeros pasos ya recorridos.

## Punto de partida comprobado

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
- Objetivo 4 en curso: el cuarto formulario solicita resolución de aceptación
  con las referencias originales y el justificante de la declaración RRHH.
  Navegador real: recuperación `200/200/200` y solicitud `409`, con el texto
  «Pendiente de validar respuesta y plazo por RRHH. No se ha confirmado la
  aceptación.». Sin recibo terminal, duplicados ni cambio de estado; se conserva
  el mismo intento para reintento manual, sin nueva clave ni reintento automático.
- Bolsa Go y las migraciones AD3 `000015` / Bolsa `000004` están preparadas para
  aceptación RRHH, **no activadas ni instaladas permanentemente en las bases**.
  Falta validación de negocio competente y proveedor de permiso nominal.
  Dinámica focal PostgreSQL PASS: una aceptación almacenada, un historial y un
  evento; replay con mismo recibo/fecha, material divergente y segundo terminal
  rechazados. UP/DOWN restauraron SHA y ACL exactos; cero datos y cero migraciones
  persistidos. El doble de autorización fue estrictamente privado y transaccional:
  acredita almacenamiento aislado, no criptografía ni aceptación funcional E2E.
- Métrica sin incremento: **5 de 8 pasos completos más parte del sexto**.
  Registrar una declaración de aceptación no resuelve la aceptación ni
  verifica origen, firma o custodia del correo. El `.eml` se lee y resume
  localmente en el navegador; no se sube.
- No se acredita correo entregado, aceptación, nombramiento, incorporación
  ni producción por tener pantallas o contratos escritos.

## Cómo trabajamos desde ahora

1. Un objetivo funcional en curso por línea canónica. Dirección integra;
   tres subagentes ayudan con archivos separados, componentes necesarios o
   documentación. No se crean ramas equivalentes ni se copia código entre ellas.
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
| 4 | Registrar una aceptación válida | Permiso específico, comprobación competente de respuesta, plazo, justificante y estado, resolución y mismo recibo recuperable; la declaración del corte 3 no sustituye esa resolución. Habilita la propuesta de nombramiento solo tras aceptación válida. | En curso: formulario real hasta `409` pendiente; Bolsa Go/SQL preparados, no activados. Faltan validación de negocio y proveedor de permiso nominal |
| 5 | Registrar una renuncia válida | Respuesta y motivo conservados; deja de ofrecerse la aceptación de ese llamamiento. | Después de 3 |
| 6 | Resolver un vencimiento cuando corresponda | Solo con inicio y política de plazo acreditados; no calcula un plazo legal desde el aviso local. | Depende de evidencia y política |
| 7 | Continuar con la siguiente persona tras renuncia o vencimiento | Reutiliza el orden entregado por Bolsa y abre un único nuevo llamamiento; no crea otro motor de selección. | Después de 5 o 6 |
| 8 | Guardar y recuperar una propuesta de nombramiento | Parte de la aceptación real registrada; muestra datos, estado de propuesta y recibo, sin fingir nombramiento firmado. | Después de 4 |
| 9 | Descargar los documentos de la propuesta | Un documento por corte, usando el generador existente; campos del expediente y descarga real. Véase desglose siguiente. | Después de 8 |
| 10 | Incorporar la resolución y su evidencia de firma o validación | Documento y estado vinculados al expediente según la autoridad admitida. Una firma pendiente no se presenta como completada. | Depende del circuito admitido |
| 11 | Confirmar la incorporación | Fecha, centro y relación de personal conservados y recuperables; solo la integración mínima de Personal necesaria para contratación. | Después de nombramiento válido |
| 12 | Descargar la ficha para GINPIX | Fichero de incorporación utilizable para la grabación manual prevista; no exige construir la conexión automática. | Después de 11 |
| 13 | Registrar seguimiento y cerrar el expediente | Una anotación y después el cierre, en cortes separados; historial y estado final conservados. | Después de 11 |
| 14 | Entregar el recorrido completo a Alberto y RRHH | Arranque reproducible, ocho pasos recorribles, manuales al día y lista explícita de dependencias productivas. Una comprobación conjunta final. | Después de los anteriores |

**Documentos del objetivo 9: seis cortes, no un generador nuevo.**
Informe definitivo, resolución, diligencia, toma de posesión, notificación y
comunicación al centro. Se reutilizan las piezas escritas. Sin modelo oficial
se entrega un borrador de desarrollo claramente marcado, no una redacción
jurídica validada ni un documento firmado.

El paso 6 no se declara completo por cerrar solo la aceptación: también se
comprueban sus alternativas y la continuidad con la selección siguiente.
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

## Plan de ataque funcional

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
