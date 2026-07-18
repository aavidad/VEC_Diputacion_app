# Estado real y brecha funcional de Bolsa — verificación de 18 de julio de 2026

## Finalidad y alcance

Esta memoria sustituye la fotografía funcional redactada el 17 de julio por una
verificación más estricta del código y de la composición existente el 18 de
julio de 2026. Se conserva el nombre del fichero para no romper sus referencias.

El objetivo es impedir que una pantalla visible, un conjunto de datos
sintéticos, un dominio bien probado o una migración cerrada se contabilicen como
un recorrido administrativo terminado. La revisión abarca el módulo Bolsa, sus
portales público e interno, su raíz de composición, los adaptadores y las pruebas
asociadas. No acredita el despliegue de infraestructura externa.

Los términos de esta memoria significan:

| Estado | Significado verificable |
| --- | --- |
| **Presentación** | Interfaz preparada para enseñar el recorrido. Puede contener datos sintéticos y controles sin efecto administrativo. |
| **Demostración conectada** | Navegador, API y fuente se comunican de extremo a extremo, pero la fuente declara expresamente que carece de validez administrativa. |
| **Contrato probado** | Dominio, caso de uso, puerto o adaptador pasa pruebas aisladas. No implica que una raíz de composición lo haga alcanzable. |
| **Integrado** | La ruta está registrada con dependencias reales y puede recorrerse sin dobles ni semillas locales. Aún puede tener barreras operativas para producción. |
| **Productivo E2E** | Recorrido integrado, durable, autorizado, auditable, recuperable tras reinicio y probado bajo concurrencia con todos sus conectores legales. |

Solo el último estado permite afirmar que una capacidad administrativa está
terminada. Un manifiesto, un botón o una migración instalada pero cerrada no
conceden por sí mismos una capacidad productiva.

## Resultado ejecutivo

La situación observada es:

- **0 recorridos administrativos productivos E2E** de Bolsa;
- **1 recorrido público de demostración conectado**: consulta anónima de
  convocatorias, detalle y categorías;
- **1 vertical real de lectura interna contratada y probada**, el panel agregado
  de Bolsa, pero no registrada en la composición ni abierta para producción;
- **1 vertical real de elaboración contratada y probada hasta el núcleo**:
  bandeja/editor web, API interna, fachada, servicio durable, alias HMAC
  multigeneración y cifrado de sobre KMS; todavía sin adaptador Go PostgreSQL,
  identidad ni composición productivos;
- **10 apartados visibles** en el portal RRHH de presentación;
- **2 de esos 10 apartados** pueden representar el contrato interno real en modo
  de solo lectura: resumen y convocatorias agregadas;
- **0 de esos 10 apartados** resulta alcanzable actualmente en modo normal,
  porque la ruta interna no está compuesta;
- **8 de los 10 apartados** muestran de forma expresa «Funcionalidad no
  conectada» cuando reciben el contrato interno real;
- **3 recursos públicos GET/HEAD** están registrados: listado, detalle y
  directorio de categorías;
- la fuente pública de serie contiene **36 procesos históricos construidos a
  partir de 37 publicaciones BOP, con título, categoría, fecha y CVE públicos
  reales**; tres fichas incorporan además
  **3 PDF, 3 HTML accesibles y 16 requisitos** adaptados. No se inventan plazos
  estructurados para los demás procesos;
- el catálogo profesional contiene **68 categorías históricas** (5 de
  Administración general, 60 de Administración especial y 3 de organismos
  dependientes), contrastadas
  con OPES y publicaciones del BOP, pero declara ser de demostración y estar
  pendiente de validación funcional por RRHH.

Estas cifras expresan cobertura de recorridos, no calidad del código ni volumen
de trabajo. La cimentación reutilizable es considerable, pero no se suma como
funcionalidad productiva mientras permanezca aislada o cerrada.

## Composición y rutas observadas

### Superficie pública

La entrada dedicada `cmd/vec-publico/main.go` construye exclusivamente la API y
el servidor públicos. La lista positiva de
`internal/app/server/server.go` limita la superficie a `/bolsa`, sus recursos,
`/api/publico` y salud; no expone el portal interno y rechaza credenciales
ambientales.

| Ruta registrada | Estado | Evidencia principal |
| --- | --- | --- |
| `GET/HEAD /api/publico/bolsa/convocatorias` | Demostración conectada | `internal/modules/bolsa/adapters/httppublico/handler.go`, `internal/app/bootstrap/bootstrap.go` |
| `GET/HEAD /api/publico/bolsa/convocatorias/{identificador}` | Demostración conectada | `internal/modules/bolsa/adapters/httppublico/handler.go` |
| `GET/HEAD /api/publico/bolsa/categorias` | Demostración conectada | `internal/modules/bolsa/adapters/httppublico/handler.go`, `internal/modules/bolsa/adapters/catalogosvec/categorias.go` |
| `/bolsa/` | Interfaz integrada con la API anterior | `web/static/bolsa/index.html`, `web/static/bolsa/bolsa.js` |

La interfaz pública incorpora búsqueda, filtros por tipo, categoría, estado y
plazo, paginación, estados de carga/error/vacío, directorio de categorías,
detalle de plazos, requisitos, documentos y ayuda, y preferencias de texto
grande y alto contraste.

No obstante, no es una publicación administrativa real:

- `config/config.go` selecciona por defecto
  `data/demo/convocatorias_publicas.demo.json` y
  `data/catalogos/categorias-profesionales/v1.demo.json`;
- `internal/modules/bolsa/adapters/fichero/convocatorias.go` exige que la fuente
  indique `demostracion=true` y contenga un aviso de demostración; rechaza una
  fuente que pretenda utilizar el mismo adaptador como publicación oficial;
- los tres documentos visibles son páginas HTML sintéticas ubicadas en
  `web/static/bolsa/documentos/`;
- la propia interfaz informa de que no tramita solicitudes.

Por tanto, existe un recorrido técnico completo y reutilizable, pero aún no una
proyección pública durable de convocatorias aprobadas por RRHH.

### Superficie interna

El portal normal consulta únicamente:

```text
GET /api/vec/bolsa/panel
```

`web/static/portal-empleado/portal.js` usa `credentials: "omit"`, exige el
envelope canónico `vec.bolsa.panel.interno.v1` y falla cerrado ante ausencia de
sesión, autorización, ruta o servicio. No sustituye un fallo por datos
sintéticos.

La vertical real existe en piezas:

- frontera HTTP estricta en
  `internal/modules/bolsa/adapters/httpinterno/handler.go`;
- PEP y exigencia de autorización PDP ligada V2, garantía alta y superficie
  interna en `internal/modules/bolsa/application/panel_interno.go`;
- consulta PostgreSQL serializable y validación del recibo en
  `internal/modules/bolsa/adapters/postgres/panel_interno.go`;
- esquema, funciones cerradas, auditoría y proyección en
  `deploy/postgresql/bolsa_panel/`;
- presentador de contrato real en
  `web/static/portal-empleado/portal-panel-interno.js`.

Sin embargo:

- no hay importación ni registro productivo del adaptador `httpinterno` en la
  raíz de composición;
- no hay un proceso dedicado que componga el servidor interno con esa ruta;
- falta el preparador de órdenes que resuelva identidad, sesión revalidada,
  perfil, ámbito, motivo y correlación desde una frontera confiable;
- `deploy/postgresql/bolsa_panel/README.md` mantiene la función de consulta
  cerrada sin `EXECUTE` hasta disponer de verificación COSE, claves y
  revocaciones productivas.

La ruta preparada por el navegador para propuestas de llamamiento tampoco
existe en la composición:

```text
POST /api/vec/bolsa/propuestas-llamamiento
```

El navegador solo intentaría usarla si el servidor concediese expresamente esa
capacidad. El contrato real actual no la concede.

### Modo de presentación RRHH

El conjunto visual ampliado solo se carga cuando coinciden dos condiciones:

1. parámetro exacto `?presentacion=rrhh`;
2. configuración de servidor `VEC_RRHH_PRESENTATION_ENABLED=true`.

`web/static/portal-empleado/datos-presentacion.js` se identifica como adaptador
exclusivo de presentación, declara `demostracion=true`, no es una API ni una
persistencia y mantiene deshabilitadas las capacidades de proponer y confirmar
llamamientos. Contiene seis bolsas sintéticas, tres necesidades, tres
elaboraciones, cuatro contratos, tres reglas, cuatro documentos, cuatro canales
y cuatro eventos de actividad.

`web/static/portal-empleado/portal-eventos.js` limita las acciones a navegación,
impresión o diálogos explicativos. «Validar recorrido» no confirma, firma,
comunica ni modifica un expediente. Este modo es adecuado para enseñar la
distribución del futuro puesto de trabajo, pero no demuestra actos de negocio.

## Matriz funcional estricta

| Capacidad | Presentación o demo | Contrato probado reutilizable | Integrado sin demo | Productivo E2E | Brecha principal |
| --- | --- | --- | --- | --- | --- |
| Consulta pública | Sí: interfaz y fuente DEMO conectadas | Sí: servicio, filtros, minimización y handlers | No | No | Sustituir la fuente DEMO por una proyección durable de versiones aprobadas. |
| Panel interno agregado | Sí: cuadro ampliado sintético | Sí: dominio, aplicación, HTTP y PostgreSQL de lectura | No | No | Identidad interna, autoridad COSE, publicador de proyección y raíz de composición. |
| Gobierno de convocatorias | Sí: expedientes y botones sintéticos; bandeja/editor real de borradores | Sí: API interna, fachada y servicio durable para alta/actualización; publicación, sustitución y retirada en dominio y puertos | No | No | Adaptador Go PostgreSQL completo, identidad/PDP/KMS productivos, publicación/retirada y composición. |
| Bases y reglas de baremo | Sí: pantalla y valores de ejemplo | Sí: versiones, topes, jornada, redondeo, cálculo exacto y planes de gobierno | No | No | Adaptador Go productivo, broker de capacidades, fuente autoritativa, editor y simulador RRHH. |
| Autobaremación de aspirante | Sí, en el runtime heredado `fake` | Sí: recorrido heredado E2E y motor oficial moderno aislado | No | No | Preservar la capacidad y migrarla a identidad, fuentes, persistencia y API modernas; no eliminarla ni confundirla con la baremación oficial RRHH. |
| Revisión técnica de baremación | Sí: concepto visual | Sí: aceptación, desestimación, subsanación, rectificación, revocación, rehabilitación y firma | No | No | Bandeja RRHH, autorización, evidencia documental, firma/custodia y persistencia compuestas. |
| Llamamientos | Sí: asistente sintético de cuatro pasos | Sí: selección del primer elegible, seguridad e idempotencia en dominio/aplicación | No | No | Persistencia completa, fuentes autoritativas, confirmación y ciclo de comunicación/respuesta. |
| Contratos, ceses y reincorporaciones | Sí: tabla sintética | Parcial: conceptos relacionados con llamamientos | No | No | Modelo y casos de uso propios, integración con RRHH/nómina y trazabilidad del ciclo completo. |
| Documentación y firma | Sí: plantillas y estados sintéticos | Sí: puertos PAdES, sello de tiempo, validación, custodia y flujo de firma de baremación | No | No | Subida, análisis, generación, firma, registro, custodia y descarga conectados al expediente. |
| Comunicaciones | Sí: canales previstos | Parcial: contratos generales y outbox en algunos agregados | No | No | Conectores reales, notificación fehaciente, reintentos, recibos y preferencias. |
| Ayuda | Sí: contenido real estático, FAQ, audio local y transcripción | Sí: contrato de contenido comprobado en web | Sí, como recurso estático | No como sistema administrable | Catálogo gobernado, contexto por perfil/expediente, cobertura exhaustiva y bot. |
| Roles y permisos | Sí: contexto sintético de presentación | Sí: 7 permisos gruesos, 11 entradas de menú y políticas PDP de alta garantía | No para Bolsa interna | No | Identidad productiva, roles publicados, asignaciones, segregación y preparador de órdenes. |
| Auditoría y trazabilidad | Sí: cronología sintética | Sí: huellas, CAS, idempotencia, outbox, recibos y auditoría en varias piezas | No en un acto web completo | No | Enlazar cada acción web con autorización, persistencia, recibo, recuperación y consulta. |
| Candidatura, alegaciones y registro | Parcial en legado y presentación | Parcial en modelos heredados | No | No | Recorrido personal completo, firma, registro, subsanación, resolución y notificación. |

### Precisión sobre los adaptadores durables

La presencia de SQL o de un tipo PostgreSQL no equivale a disponibilidad
productiva:

- `deploy/postgresql/bolsa_reglas_baremo/README.md` declara la instalación
  cerrada y pendiente del adaptador que restaure el canon Go y de las fronteras
  confiables;
- `deploy/postgresql/bolsa_calculo_experiencia/README.md` enumera barreras
  `NO-GO`, incluido el repositorio autoritativo de reglas y el adaptador Go;
- `internal/modules/bolsa/adapters/postgres/llamamientos_transaccion.go` valida
  el comando pero devuelve deliberadamente
  `ErrPersistenciaPropuestaNoDisponible` hasta que un nuevo contrato confirme la
  instantánea completa, autorización, atestación, auditoría y outbox en un solo
  `COMMIT`;
- `deploy/postgresql/bolsa_baremacion/README.md` mantiene bloqueos explícitos de
  producción y exige una prueba Go–PostgreSQL completa;
- `internal/modules/bolsa/operational_contract.go` fija
  `LegalProductionReady=false` y declara como no configuradas la firma
  electrónica, el registro electrónico, la notificación fehaciente y la
  auditoría probatoria externa.

Tampoco se deben contabilizar como productivas las capacidades denominadas
`real` en el manifiesto heredado de candidato: la raíz de composición solo monta
esa API cuando la autenticación es `fake`.

## Paneles RRHH: visible frente a operativo

| Apartado visible | Presentación RRHH | Contrato interno real | Alcanzable ahora en modo normal |
| --- | --- | --- | --- |
| Cuadro de mando | Completo con datos sintéticos | Resumen agregado, 12 indicadores, convocatorias, actuaciones y prueba de lectura | No |
| Elaboración y gestión | Expedientes sintéticos y controles informativos | Lista agregada de convocatorias en solo lectura | No |
| Llamamientos | Asistente sintético sin confirmación | «Funcionalidad no conectada» | No |
| Contratos, ceses y reincorporaciones | Tabla sintética | «Funcionalidad no conectada» | No |
| Motor de reglas | Tabla y diálogos sintéticos | «Funcionalidad no conectada» | No |
| Consulta segura | Explicación de minimización | «Funcionalidad no conectada» | No |
| Estadísticas | Indicadores sintéticos | «Funcionalidad no conectada» | No |
| Generación de documentos | Plantillas sintéticas | «Funcionalidad no conectada» | No |
| Comunicaciones | Canales sintéticos | «Funcionalidad no conectada» | No |
| Auditoría | Cronología sintética | «Funcionalidad no conectada» | No |

El contrato real del panel está deliberadamente minimizado: admite hasta 40
resúmenes de convocatoria y 80 actuaciones pendientes, pero no contiene nombres,
documentos identificativos, correos, teléfonos ni una colección global de
aspirantes. La prueba de lectura incluye referencias de lectura, auditoría,
decisión y correlación, secuencia y fecha confirmada. Esta buena base de
trazabilidad sigue sin ser un recorrido operativo mientras no se componga.

## Pruebas ejecutadas en esta verificación

Se ejecutaron, sin modificar datos del proyecto:

```text
go test ./internal/modules/bolsa/domain/... \
  ./internal/modules/bolsa/application/... \
  ./internal/modules/bolsa/ports/... \
  ./internal/modules/bolsa/adapters/httppublico \
  ./internal/modules/bolsa/adapters/httpinterno \
  ./internal/modules/bolsa/adapters/fichero

go test ./internal/modules/bolsa/adapters/postgres

go test ./internal/app/bootstrap ./internal/app/server ./config

node --test web/static/bolsa/*.test.mjs \
  web/static/portal-empleado/*.test.mjs
```

Resultado: todos los paquetes Go anteriores finalizaron correctamente. La
repetición ampliada de 18 de julio terminó con **51 de 51** pruebas Node sin
fallos: 5 del portal público y 46 del Portal del Empleado, incluidas las del
editor real de borradores.

Esta evidencia acredita pruebas unitarias y de contrato, incluidas pruebas de
adaptadores con dobles. En esta revisión no se ejecutaron:

- los arneses PostgreSQL efímeros de `deploy/postgresql/bolsa_*/`;
- un recorrido con navegador real;
- una prueba de reinicio completo;
- los conectores externos de identidad, firma, registro, custodia, antivirus o
  notificación.

Por ello el resultado verde no debe describirse como despliegue productivo ni
como prueba E2E legal.

## Bloqueos ordenados por camino crítico

1. **Identidad y autoridad interna**: falta componer autenticación de alta
   garantía, revalidación, perfil activo, ámbito, motivo, correlación, PDP y
   verificación COSE. La política continúa siendo denegar por defecto. La base
   actual conserva cuentas `cta_` y una proyección de contexto, pero no los
   maestros versionados de persona, perfil y vínculos `cta_` → `per_` → `prf_`;
   no se abrirá el portal RRHH usando las asignaciones RBAC como sustituto de
   esa autoridad de identidad.
2. **Composición durable de convocatorias**: ya existen fachada, servicio de
   alta/actualización, diario y recuperación con cercado, identidad HMAC
   multigeneración, perfil criptográfico gobernado, sobre AEAD, DEK envuelta y
   contrato de revalidación KMS. Faltan el adaptador Go PostgreSQL que restaure
   y persista esos tipos sin degradarlos, las fuentes autoritativas de
   aprobación/dependencias y la raíz autenticada. La confirmación SQL debe
   conservarse sin privilegio de ejecución mientras no almacene AAD, perfil,
   envoltura y atestación KMS completas. Por ello la composición sigue en
   `NO-GO`.
3. **Gobierno ejecutable del baremo**: faltan el adaptador Go y las fuentes
   autoritativas que permitan usar las reglas versionadas y el cálculo oficial
   sin riesgos de incoherencia o TOCTOU.
4. **Cadena documental legal**: faltan la subida directa, cuarentena, análisis,
   custodia, generación, firma, sello de tiempo, registro y descarga conectados
   al expediente.
5. **Proyección pública oficial**: la API pública solo puede leer su fuente DEMO;
   falta publicar la misma versión aprobada y auditable de la convocatoria.
6. **Candidatura y revisión RRHH**: faltan los recorridos personales, la fuente
   de méritos, la autobaremación oficial, la bandeja técnica y sus decisiones
   firmadas.
7. **Ciclo completo de llamamiento**: faltan persistencia productiva,
   confirmación, comunicación, respuesta, renuncia, no comparecencia,
   contratación, cese y reincorporación.
8. **Prueba operacional**: faltan navegador, PostgreSQL real, concurrencia,
   reinicio, reconciliación y conectores legales en un mismo corredor.

Las dependencias no deben invertirse: una bolsa constituida y sus llamamientos
dependen de convocatoria, bases, candidaturas, baremación, listas y alegaciones
previamente gobernadas.

## Siguiente corte: expediente gobernado de convocatoria

El corte de mayor valor para RRHH es sustituir una cadena completa de
presentación por el expediente real de convocatoria, desde el borrador hasta su
proyección pública. Debe construirse de dentro hacia fuera y permanecer sin
rutas internas de escritura registradas mientras falte identidad segura.

### Estado de seguridad del ciclo sellado–autorización

El contrato V2 implantado en `5d9fa80` ya elimina el efecto durable del sellado
de motivo previo a la autorización. El corte entre lo implantado y lo aún
decidido es:

```text
versión canónica y motivo local
  → compromiso HMAC determinista no durable
  → material V2 exacto, sin motivo ni SHA semántico crudo
  → solicitud de concesión PDP exacta
  [contratos y pruebas implantados]
  → concesión PDP exacta
  → reserva durable por CAS
  → atestación HSM consumible
  → confirmación transaccional
  [contrato de núcleo, diario y recuperación implantados]
  → persistencia de AAD, perfil, DEK envuelta y sobre AEAD
  → revalidación KMS autoritativa dentro del COMMIT
  [adaptador Go, SQL equivalente, identidad y composición pendientes]
```

Acción, versión, motivo aplicable, actor y correlación forman una única preimagen
canónica. El contrato del futuro adaptador solo permite enviar al HSM/KMS
dominio y huella; al material durable solo la representación HMAC con clave. La
huella semántica cruda no aparece en el material, la atestación final ni la
reconciliación, y los valores efímeros fallan cerrados ante JSON, XML, Gob,
texto, binario, CBOR y YAML. El esquema de intención V1 se rechaza sin
reinterpretación; el dominio criptográfico del motivo conserva su propia
versión nominal V1. El motivo administrativo sí permanece en el expediente
gobernado y sujeto a sus permisos; no se replica en el material HMAC.

El contrato ha superado pruebas normales, de carrera, manipulación, formatos
de serialización reales y vectores de referencia. El núcleo ya dispone del
diario, recuperación, cercado y orden de confirmación atómica. Esto es un GO
técnico del contrato, no un GO productivo: aún no existe un adaptador Go y una
transacción PostgreSQL equivalentes que unan reserva, decisión, sellado, KMS,
CAS de negocio, auditoría, bandeja transaccional y recibo en un solo commit.

La secuencia decidida consulta el localizador y la huella HMAC de
idempotencia antes del PDP, pero solo reserva después de la concesión. Una
denegación deja cero reservas de negocio y cero sellados. Una respuesta perdida
se recuperará desde el diario antes de repetir PDP/HSM; entregar el recibo o
datos actuales exigirá una autorización nueva de lectura.

Publicar y retirar tienen además otra barrera: aprobación firmada y
dependencias deben ser hechos autoritativos, inmutables y versionados que ya
existan antes del PDP y se relean localmente durante el commit. Las
atestaciones reconstruibles actuales no levantan esa barrera.

### Trabajo propuesto

1. Implementar el adaptador Go PostgreSQL del diario y del agregado cifrado,
   restaurando la identidad primaria y todos sus alias, y revalidando la
   atestación KMS dentro de la transacción de confirmación.
2. Componer el servicio de borradores y su API ya existentes con identidad de
   alta garantía, perfil activo, motivo catalogado, PDP y conectores KMS reales.
3. Migrar aprobación y dependencias a hechos autoritativos preexistentes e
   implementar el adaptador PostgreSQL de gobierno con transacción
   `SERIALIZABLE`, CAS, idempotencia, auditoría y bandeja transaccional. La
   función de escritura debe permanecer sin privilegios de ejecución hasta
   abrir la autoridad productiva.
4. Completar el adaptador Go de reglas de baremo y el simulador de experiencia;
   la convocatoria fijará siempre referencia, versión y huella exactas, nunca
   «la última versión».
5. Relacionar las bases con documento lógico, representación, huella, firma
   validada y recibo de custodia. El dominio ya exige esa correspondencia uno a
   uno antes de publicar.
6. Exponer primero un DTO interno de expediente en solo lectura y adaptar
   `web/static/portal-empleado/portal-panel-interno.js`. No mostrar controles de
   edición si el servidor no entrega una capacidad positiva.
7. Crear una consulta PostgreSQL pública que solo proyecte versiones publicadas
   y aprobadas. El adaptador de fichero continuará reservado a demostración.
8. Montar las rutas mutantes únicamente después de disponer del preparador de
   identidad y de la autorización productiva.

Archivos principales del corte y adaptadores todavía pendientes, sin mezclar
responsabilidades:

```text
internal/modules/bolsa/application/gobiernoconvocatorias/servicio_borradores.go
internal/modules/bolsa/application/gobiernoconvocatorias/recuperacion_borradores.go
internal/modules/bolsa/application/gobiernoconvocatorias/cifrado_borradores.go
internal/modules/bolsa/adapters/httpinterno/borradores.go
internal/modules/bolsa/adapters/postgres/convocatorias_gobierno.go
internal/modules/bolsa/adapters/postgres/convocatorias_gobierno_test.go
internal/modules/bolsa/adapters/postgres/convocatorias_publicas.go
deploy/postgresql/bolsa_convocatorias/pruebas_sql/gobierno_y_publicacion.sql
```

Rutas internas objetivo, no registrables antes de resolver identidad y
autorización:

```text
GET   /api/vec/bolsa/convocatorias
GET   /api/vec/bolsa/convocatorias/{id}/versiones/{secuencia}
POST  /api/vec/bolsa/convocatorias/borradores
PATCH /api/vec/bolsa/convocatorias/{id}/borradores/{revision}
POST  /api/vec/bolsa/convocatorias/{id}/publicaciones
```

Las rutas públicas existentes no necesitan cambiar de forma; debe cambiar su
adaptador de lectura una vez exista la proyección oficial.

### Criterios de aceptación del corte

- un borrador sobrevive a reinicio completo;
- dos cambios concurrentes sobre la misma revisión no pueden confirmarse ambos;
- un reintento materialmente idéntico devuelve el mismo recibo sin duplicar
  auditoría ni eventos;
- una regla, calendario, flujo, catálogo o documento con versión o huella
  distinta se rechaza;
- no aparece ningún contenido público antes de aprobación y publicación;
- las bases publicadas tienen firma validada y recibo de custodia;
- la proyección pública corresponde a la misma versión y huella aprobadas;
- la denegación por defecto funciona sin identidad, con identidad caducada y sin
  concesión exacta;
- PostgreSQL, API y navegador se prueban conjuntamente, incluida concurrencia,
  reinicio y reconciliación;
- la UI muestra estado, versión, plazo, evidencia, recibo y siguiente acción sin
  inventar datos ausentes.

Este corte aún no migra a producción candidaturas, la autobaremación preservada
ni llamamientos. Sí crea la autoridad de convocatoria y reglas que todos ellos
necesitan. La fuente DEMO del recorrido público ya emplea metadatos BOP reales,
pero tampoco se elimina hasta que exista una proyección oficial de la misma
versión aprobada; las reproducciones documentales adaptadas no se
reinterpretarán como publicación administrativa ni se sustituirán por una vía
de administración insegura.

## Regla de comunicación del avance

Los informes, demostraciones y porcentajes futuros deben acompañar cada
capacidad de uno de los estados definidos al comienzo. En particular:

- «visible» no significa «conectado»;
- «probado» no significa «compuesto»;
- «PostgreSQL» no significa «abierto para producción»;
- «auditable en el dominio» no significa «trazabilidad E2E»;
- «datos sintéticos completos» no significa «gestión RRHH terminada».

No se afirmará que Bolsa está completa ni productiva hasta superar los criterios
E2E con identidad, persistencia, documentación y conectores legales reales.
