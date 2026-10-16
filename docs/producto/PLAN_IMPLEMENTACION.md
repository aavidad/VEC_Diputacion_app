# Plan de implementación de VEC

Fuentes absorbidas el 13/10/2026: `docs/plan_modulos/OBJETIVOS.md` (retirado),
la hoja de ruta de
`docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md` (retirado),
`ESTADO_PROYECTO.md` (reducido a una nota), la condición de terminado de
`docs/portal_vec/expediente_contratacion_temporal_rrhh.md` (retirado), las
reglas de trabajo de `AGENTS.md`, E11 y E13 de `ESPECIFICACIONES_AGENTES.md` y
las decisiones de orden de módulos y baremador de la memoria de dirección.

Revisado el 16 de octubre de 2026 para la candidata A+B+C sobre `origin/main@3216104e4`;
los cortes siguen pendientes de integración y ensayo conjunto.

Este documento es la única fuente de qué falta, en qué orden se construye,
cómo se sabe que cada fase está terminada y cómo se trabaja. Es lectura
obligatoria antes de elegir o encargar la siguiente tarea. Reúne lo vigente de
`docs/plan_modulos/OBJETIVOS.md`, `ESTADO_PROYECTO.md`, la hoja de ruta de
`docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md`, las reglas de
trabajo de `AGENTS.md` y los relevos de dirección; la lista exacta está en el
apartado 13.

No repite lo que cuentan los otros documentos de producto:

| Para saber… | Lee |
| --- | --- |
| Qué es VEC, para quién y qué módulos tiene | [`PRD.md`](PRD.md) |
| Cómo está construido, cómo arranca y cómo se despliega | [`TRD.md`](TRD.md) |
| Quién hace qué en cada recorrido y con qué perfil | [`FLUJO_APP.md`](FLUJO_APP.md) |
| Cómo se ven y se usan las pantallas | [`DISENO_UI_UX.md`](DISENO_UI_UX.md) |
| Esquemas, tablas, roles, migraciones y rutas de la API | [`ESQUEMA_BACKEND.md`](ESQUEMA_BACKEND.md) |

El detalle de cada módulo sigue en `docs/plan_modulos/<modulo>.md` (apartado
9). Las preguntas a RRHH y al DPD siguen en `dudas.md`. Si este plan y un plan
de módulo no coinciden, manda el más reciente: anota la diferencia y corrige
el otro en la misma entrega.

## Vocabulario mínimo

- **cidonia**: el servidor donde se instala VEC para que RRHH lo vea, con datos
  sintéticos. No es producción.
- **HZ*n***: número de cada subida a cidonia. Cada una tiene copia fría de la
  base y un guion de vuelta atrás.
- **Migración SQL**: cambio de la base. Su código empieza por las letras del
  dueño: `AD` administración e identidad, `AUT` autorización, `CT`
  Contratación temporal, `B` Bolsa, `P` Personal, `CA`, `CC` e `IS` cadenas de
  identidad y catálogos, `D` Documentos, `RUM` y `M` Méritos. Los números se
  reservan antes de escribirlas en el registro de reservas, que está fuera de
  Git. Nunca se reutiliza un número reservado por otra rama.
- **Selector**: variable de entorno que enciende una función, por ejemplo
  `VEC_BOLSA_CESE_CT_ENABLED`. Apagada, la función explica en pantalla por qué
  no está; nunca impide que arranque la aplicación.
- **Equipo**: un agente Codex con una letra (K, L, M, H, O, J…) que programa
  una tarea en su propia rama. Claude dirige, programa también, revisa,
  fusiona y sube a cidonia.
- **V3**: el motor de autorización. Cada decisión queda ligada a acción,
  recurso, finalidad y actor. Se describe en `TRD.md`.

## 1. Prioridad vigente

- **Ahora**: cerrar Bolsa y Contratación temporal (CT), con la firma de dos
  personas. Contratación temporal sigue el procedimiento que remitió RRHH
  (`docs/estudio_requisitos/decisiones_rrhh_2026-10-02.md`).
- **Bolsa no se borra ni se subordina**: mantiene convocatorias, candidaturas,
  posiciones, reglas y llamamientos. CT no tiene llamamiento propio: usa el
  llamamiento real de Bolsa sobre una bolsa constituida.
- **Después**: Cronos y luego Dietas (orden de Alberto del 07/10/2026), cada
  uno detrás de la base de Personal que necesite.
- **Aparcado**: el resto de módulos, hasta cerrar Bolsa y CT. Sus planes se
  conservan y se retoman uno a uno.

Cuando una tarea de Bolsa o CT necesita una capacidad común (VEC, Personal,
Documentos, firma, identidad):

1. se define una tarea dependiente y acotada;
2. se implementa en el módulo dueño de esa capacidad;
3. se prueba e integra;
4. se vuelve en seguida a Bolsa y CT.

No se amplía otro módulo por conveniencia ni se cambia la prioridad sin orden
de dirección. Las preguntas pendientes no detienen las partes que no dependen
de ellas; lo que RRHH no ha decidido se configura en un catálogo y se pregunta
en `dudas.md`.

Que haya código en `main` no significa que esté instalado en cidonia ni que se
haya recorrido en el navegador. Los datos son sintéticos y nada de esto es
producción.

## 2. Estado actual

Lo marcado *pendiente de verificar* no se ha podido comprobar al escribir.

### 2.1 Instalado en cidonia

Según el registro de dirección del 13/10/2026 a las 15:15 (*pendiente de
verificar contra el servidor*):

- **Subida HZ36.** También está arrancado el servidor de administración
  (`vec-admin`, en su subdominio con certificado), con la carga de bolsas desde
  el Excel de CONVOCA (B1), la fuente de respuestas B103 y la cuenta externa.
- **Reglas y catálogos**: reglas de CT v2, reglas de Bolsa v6, circuito v2,
  plantillas de documentos v3, catálogo de acciones v5, rol RRHH v10.
- **Bolsa**: candidatos, importación desde CONVOCA, correo (buzón de pruebas),
  baremación, documentos de Bolsa, convocatorias (vacía hasta abrir la
  inscripción), configuración, «Mi Bolsa» del aspirante, cifras globales
  pulsables (#921), cese con vuelta a la bolsa (selector encendido desde HZ10).
  «Mi perfil» de RRHH responde 403. Hay tres convocatorias de inscripción
  cargadas.
- **CT**: desde la petición del centro hasta la resolución, con el alta según
  la circular de mayo (CT193 y CT194), edición de plazos en «Reglas vigentes»
  (#919, CT190 y CT191), recuperación de una selección a medias entre CT y
  Bolsa (incidencia ORQ-1, #918), incorporación acreditada y plantillas PDF y
  Word.
- **Paso a Personal**: encendido desde HZ18. Ningún expediente antiguo puede
  llegar a él porque usan una categoría de desarrollo; hace falta uno con
  categoría de la RPT publicada (#1070, en curso).
- **Auditoría de RRHH**: instalada pero apagada; le falta configuración.
- **Contacto completo del candidato** en la ficha: responde 200.

La historia de subidas, con sus SQL, copias y guiones de vuelta atrás, la
conserva dirección fuera de Git.

### 2.2 Fusionado en `main` y todavía sin subir

El registro del 13/10 a las 15:15 cita #1093, #1096, #1097, #1100 y #1103.
Desde la madrugada del 13/10 se fusionaron además #1088 (seleccionar desde la
bolsa constituida para el llamamiento CT) y #1112 (cuenta externa propia del
aspirante). *Pendiente de verificar* cuáles están ya en cidonia.

La subida estaba bloqueada por la migración AD250 (#1081), que no encajaba con
la base real de cidonia. #1102 la sustituye por AD262, ensayada en una copia de
la principal del 13/10 (`deploy/principal/habilitacion_ad262_actual_20261013.md`).
*Pendiente de verificar* si AD262 está instalada.

### 2.3 En curso

**F1-37, ajustes de firma.** El código ya está fusionado en `main` (#1163):
límite PDF de 2 MiB en firma VEC, cotas de transporte calculadas aparte,
cierre de auditoría seguro y sonda de plantillas por organización (CT226,
sobre CT218). Reutiliza las pantallas de plantillas y la ficha, sin cambiar
la verificación ni los permisos de firma. El guion está en
`deploy/principal/firma_ajustes_f1_37_20261015.md`; el ensayo conjunto con
la consulta histórica de F1-36 sigue pendiente.

**F0-02, cierre del arranque heredado de Administración.** Se prepara la
exigencia del fichero privado de usuarios antes de abrir conexiones y se
retira la alternativa que pedía un grupo de auditoría inexistente. Se conserva
la lista y ficha de usuarios con la configuración aprobada. No añade SQL ni
perfiles. Activación y vuelta atrás en
`deploy/principal/administracion_arranque_f0_02_20261015.md`; pendiente de
integración e instalación por dirección.

Corte H de vigencia de gobierno, orden de dirección del 15/10 a las 19:36:
preparación y aplicación con 24 horas por defecto, hasta 90 días solo en
desarrollo y rechazo de plazos mayores de 24 horas en producción. AD282
conserva publicaciones y recibos. La entrega exige revisión independiente y
ensayo local de preparación, aplicación y reinicio antes de incorporarla a la
candidata de dirección.
Guion: `deploy/principal/vigencia_gobierno_admin_20261015.md`.

**HZ37, O: Mi Bolsa interna en combinado mientras no se seleccione el modo
externo.** Recupera el montaje de HZ36 si no hay DSN ni material externo y la
inscripción está apagada. La selección externa exige configuración completa
y confianza vigente; si falla, devuelve error y no monta la interna. Conserva
las autoridades y la separación de identidades de HZ36. Es temporal hasta
HZ38: P prepara la publicación nominal de la confianza externa (opción B).
El criterio de configuración está en `TRD.md`, apartado 1. La integración y
el arranque en cidonia corresponden a dirección.

PR abiertas (el estado de borrador se comprueba en GitHub):

| PR | Qué | Equipo | Fase |
| --- | --- | --- | --- |
| #1099 | CT: preparar originales y conectar la firma nominal en VEC | H | 1 |
| #1070 | Comprobar el alta en Personal (B2) y detenerla sin bolsa constituida | K | 1 |
| #1008 | Textos: cómo corregir o publicar la categoría antes de vincularla | J | 1 |
| #882 | Formación: recuperar el idioma sin perder el borrador | T | 6, aparcado |
| #881 | Provisión: agrupar experiencia y separar el corrector provisional | T | 5, aparcado |

#1099 incorporó los originales y la consulta de firma desde la ficha. #1137
registra en la copia H dos firmas del informe y tres de la resolución con
GrxFirma, y recupera los dos PDF y los mismos recibos después de reiniciar.
Son dos expedientes distintos: P conserva la comprobación del caso único de
`FLUJO_APP.md` 4.0. La fase 1 y la instalación en cidonia siguen pendientes.
El guion de activación, la lista SQL y los límites están en
`deploy/principal/candidata_hz39/firma/LEEME.md`.

**F1-36A, H: base de recuperación histórica.** En rama: AD286, CT228,
AUT103, montaje nominal del consumidor existente de recuperación V2 y
publicador de cargos con su permiso positivo de recuperación. Las fuentes SQL
se ensayaron en copia propia con dos revisiones; arranque con configuración
anterior y nueva, reinicio y recuperación de originales comprobados. Falta CI
antes de integrar. Este
corte conserva las asignaciones de consulta vigentes y no ofrece un panel nuevo;
la recuperación exige renovar por Administración el cargo de firma. B conectará
la fuente autorizada y el anexo de la ficha; C montará la pantalla y gobernará
las asignaciones. Hasta C sigue abierto F1-36. Guion A:
`deploy/principal/firma_historica_base_f1_36a_20261016.md`.

**F1-36B, H: fuente de ficha y justificante.** En rama: AUT102, CT229 y el
lector interno que forma los selectores de las firmas originales y su anexo
sellado. Depende de AD286, CT228 y AUT103 del corte A. En la copia H se
ensayaron las SQL una vez, con dos revisiones; la ficha ampliada y el original
histórico rechazan al confirmar una decisión que caducó después de leer,
sin dejar efectos parciales. Los cuatro cargos de consulta se publicaron por
el CLI existente; sus asignaciones se prepararon como datos ficticios
expresamente autorizados, sin atribuirles actos de Administración. C añade
la respuesta HTTP y el panel; quedan Chrome y reinicio conjunto. F1-36 sigue
abierto. Guion:
`deploy/principal/firma_historica_fuente_f1_36b_20261016.md`.

**F1-36C, H: ficha de firmas anteriores.** En rama: respuesta ampliada,
cliente y panel que ofrece un botón por firma original autorizada. Depende
de A (recuperación y concesión) y B (fuente de ficha y anexo); no añade SQL.
Faltan integración conjunta, configuración y actos gobernados, arranque con
y sin complemento, ensayo focal, revisión independiente de usabilidad,
recorrido en Chrome y reinicio. F1-36 permanece abierto. Guion:
`deploy/principal/firma_historica_ficha_f1_36c_20261016.md`.

Ramas sin PR conservadas en el relevo de Claude del 12/10 (fichero
`RELEVO_CLAUDE_20261012.md` de la carpeta principal, fuera de Git). Ninguna
está instalada; verificar su hash antes de reanudar:

| Corte | Falta para usarlo |
| --- | --- |
| Firmantes de méritos (B7) | Una puerta completa; sin SQL, HTTP ni firma real |
| Cálculo de formación por categoría | Una puerta completa sobre el último hash; API y web |
| Analizador de PDF con ClamAV | Probar el lanzador controlado con un clamd real; montaje de cuarentena |
| Vista de cartera en «Mi Bolsa» | API real y montaje; no mostrar cifras ficticias |
| Documentos y S3 (D16 a D18) | Autorización nominal del aspirante externo y composición |
| Cliente del gestor documental Mayan | TLS del destino y configuración de Sistemas |
| Preferencias entre bolsas | PostgreSQL integrado, lectura autorizada y efecto en la adjudicación |
| Inscripción por varias categorías (B105) | **No instalar**: falta la modalidad conjunta configurable (apartado 7) |

La candidata de R conecta la consulta histórica de puestos de Personal con el
proceso combinado, dependencia del ofrecimiento desde una bolsa constituida.
Conserva la autoridad nominal existente y añade configuración opcional. En la
copia local se han comprobado lectura con permiso y recibo, denegación de otra
unidad y la misma página tras reiniciar aplicación y PostgreSQL. La selección
completa sigue pendiente de la necesidad RPT de la petición y de separar su
unidad gestora de la unidad del puesto; no se da por resuelto ese recorrido.

### 2.4 Lo que todavía no funciona de punta a punta

- **El recorrido completo** de la presentación (aspirante → centro →
  ratificador → RRHH → Intervención → llamamiento desde la bolsa → aceptación
  → resolución firmada → incorporación → Personal → cese → vuelta a la bolsa).
  El único acta escrita es la del 05/10
  (`docs/estudio_requisitos/acta_recorrido_ct_bolsa_20261005.md`) y se paró en
  la resolución.
- **La firma en el recorrido completo.** En la copia H, informe con dos firmas
  y resolución con tres, descargados desde la ficha después de reiniciar.
  Intervención rechaza la fiscalización sin la segunda firma del informe y
  la admite después. Son dos expedientes distintos. P acredita el caso único
  del apartado 4.0 de `FLUJO_APP.md`; este ensayo parcial no acredita cidonia.
- **El correo de la Diputación.** Se usa un buzón de pruebas; vale para la
  presentación, no para uso real.
- **La lectura rápida medida en cidonia.** El último dato del cuadro de CT en
  cidonia era de 363 a 372 ms en el percentil 95; las mejoras posteriores
  (#916, CT190) se midieron solo en local (unos 209 a 283 ms).
- **Los demás módulos** tienen código parcial en `main`, pero están aparcados
  y sin recorrido.

### 2.5 Lista del 8 de octubre, revisada

`OBJETIVOS.md` (08/10) fijaba once puntos. Así están el 13/10:

| Punto del 08/10 | Situación a 13/10 |
| --- | --- |
| 1. Encender en cidonia lo que existe | Cese con vuelta a la bolsa (HZ10), incorporación acreditada (HZ12) y paso a Personal (HZ18) encendidos. Faltan auditoría de RRHH y verificación de firma |
| 2. Fusionar PR abiertas de Bolsa y CT | #774, #905, #906, #907, #910 y #540 fusionadas; #909 cerrada sin fusionar |
| 3. ORQ-1, selección a medias | Resuelta con #918 (AD225, CT198, AD226, CT199), en cidonia desde HZ12 |
| 4. Repetir el recorrido completo | Pendiente (F1-31 del tablero) |
| 5. Carga de bolsas desde CONVOCA | #751 y #752 fusionadas; la pantalla #759 se cerró y la carga se hizo en `vec-admin` con dos administradores (10/10) |
| 6. Firma de dos personas | Informe y resolución registrados y recuperados tras reinicio en H (#1137); falta el recorrido único de P |
| 7. Plazos de CT editables y tarjetas de plazo | Plazos con #919 (en cidonia desde HZ11). Tarjetas de plazo en Inicio (CT192): *pendiente de verificar* |
| 8. Velocidad | Pendiente de medir en cidonia |
| 9. Correo de la Diputación | Bloqueado por datos de Informática (duda 88) |
| 10. Cifras globales de Bolsa pulsables | Hecho con #921 (B94) |
| 11. Limpieza menor | Rutas muertas de cierre retiradas con #915; la duda 147 se resolvió por el Reglamento; *pendiente de verificar* el cambio de catálogo de reglas |

## 3. Orden de construcción: base primero

Regla de Alberto (03/10/2026): si un paquete es base de otros, se hace antes;
si no, se construye un tejado para una casa que no se sabe cuánto ocupará.
Dentro de esa regla van delante las bases que necesitan Bolsa y CT
(05/10/2026). Si un equipo se bloquea esperando una base, no se le busca
trabajo «encima»: se adelanta la base.

```mermaid
flowchart TD
    ID["Identidad y ContextoActor<br/>internal/vec/domain/contexto_actor.go"]
    AUT["Autorización V3<br/>denegación por defecto"]
    ADM["Administración de perfiles<br/>cmd/vec-admin"]
    CAT["Catálogos versionados e i18n<br/>data/catalogos"]
    AUD["Auditoría común<br/>internal/vec/auditoria"]
    DOC["Documentos y custodia<br/>internal/vec/documentos"]
    ORG["Organización y RPT"]
    BOL["Bolsa<br/>internal/modules/bolsa"]
    CT["Contratación temporal<br/>internal/modules/contrataciontemporal"]
    FIR["Firma GrxFirma<br/>y verificación"]
    PER["Personal<br/>internal/modules/personal"]
    RESTO["Cronos, Dietas, Selectivos, Provisión,<br/>Méritos, Formación y resto"]

    ID --> AUT --> ADM
    AUT --> CAT
    AUT --> AUD
    CAT --> DOC
    AUD --> DOC
    CAT --> ORG
    DOC --> BOL
    ORG --> BOL
    BOL --> CT
    DOC --> FIR
    FIR --> CT
    CT --> PER
    ORG --> PER
    PER --> RESTO
    BOL --> RESTO
```

Una flecha de A a B significa «B usa A». Nada se construye antes que sus
flechas de entrada.

La orden del 08/10 (01:30) pone todo el esfuerzo en Bolsa, Contratación
temporal y lo que dependa de ellas. Sustituye a «todos los módulos en
paralelo» (07/10) y al orden general de la hoja de ruta del 24/09 mientras dure
esta prioridad. Primero va el recorrido Bolsa → CT → Personal → cese y las
bases que consume: identidad y administración, firma real de CT, auditoría,
Personal en lo que recibe del nombramiento y el paso de un proceso selectivo a
una bolsa.

Orden general de bases de VEC: identidad, usuarios, perfiles y
administradores → auditoría común → Personal y Organización/RPT → firmas →
selectivos, baremo y méritos → resto. Hoja de ruta tras Bolsa y CT: Selección,
Provisión, Cronos, Dietas, Formación, Méritos, Certificados, Acción social,
Nóminas y, al final, Prevención, Igualdad, Incompatibilidades, Régimen
disciplinario y Relaciones sindicales. El 07/10 Alberto adelantó Cronos y
Dietas al primer puesto tras Bolsa y CT (apartado 1).

Módulos visibles en cidonia hasta cerrar Bolsa y CT: Bolsa y Contratación, más
Documentos, Personal, Administración y Usuarios. Se limitan con
`VEC_PORTAL_MODULOS_VISIBLES` hasta que exista el interruptor de módulos en
Administración. Cronos sigue apagado.

| Capa | Qué aporta a los demás | Dónde está | Estado |
| --- | --- | --- | --- |
| Identidad y ContextoActor | Quién es la persona y con qué perfil actúa, desde el certificado; nunca desde cookies ni datos del navegador | `internal/vec/domain/contexto_actor.go`, `internal/vec/adapters` | En cidonia. Kerberos, pendiente |
| Autorización V3 | Decisión por acción, recurso, finalidad y actor, consumida en la misma transacción que el efecto | `internal/vec`, migraciones `AUT` y `AD` | En cidonia |
| Administración de perfiles | Asignar y retirar perfiles con dos personas administradoras | `cmd/vec-admin` | Arrancado en cidonia; asignación desde pantalla, parcial |
| Catálogos e i18n | Reglas, plazos, plantillas y textos por idioma, versionados | `data/catalogos/`, `internal/vec/reglas` | En uso por Bolsa y CT |
| Auditoría común | Registro encadenado de todo, también lecturas | `internal/vec/auditoria` | Instalada; la de RRHH, apagada |
| Documentos | Custodia, descarga auditada, plantillas PDF y Word | `internal/vec/documentos` | Encendido; antivirus y Mayan en ramas |
| Organización y RPT | Centros, puestos y categorías publicadas | `internal/vec/domain/catalogos_rpt_gobierno.go`, `data/catalogos/rpt/`, `data/catalogos/estructura-organizativa/` | Categorías de la RPT en el portal; la necesidad RPT en la petición del centro falta (duda 162) |
| Bolsa | Bolsas, candidaturas, posiciones, llamamientos | `internal/modules/bolsa` | Fase 1 |
| CT | Expediente de contratación temporal | `internal/modules/contrataciontemporal` | Fase 1 |
| Firma | GrxFirma por `afirma://` y su validador | `internal/modules/contrataciontemporal/adapters` | Fase 1, sin cerrar |
| Personal | Relación laboral que nace del nombramiento | `internal/modules/personal` | Recibe de CT en fase 1; el resto, fase 2 |

Personal recibe el alta desde CT por su puerto, sin duplicar el procedimiento
de CT. Ningún módulo lee ni escribe tablas de otro; el detalle está en
`ESQUEMA_BACKEND.md`.

## 4. Hoja de ruta funcional

El catálogo funcional del 16/07/2026 (retirado el 13/10) era el inventario de
capacidades de cada materia; ese inventario está ahora en `PRD.md`, apartado 9.
Su orden de implantación queda absorbido aquí y ajustado a las órdenes
posteriores de Alberto:

| Bloque del catálogo | Qué contenía | Dónde queda en este plan |
| --- | --- | --- |
| F0. Fundamento | Persona, cuenta, relación, organización y autoridad; política versionada, aritmética exacta y tiempo civil; autorización, segregación, auditoría y eventos; documentos, firma y archivo; portales externo e interno separados | Fase 0 |
| F1. Núcleo y Bolsa | Convocatoria y bases gobernadas; portal público y área del aspirante; solicitud, tasa, firma, registro y justificante; méritos reutilizables; baremo con autobaremo y revisión firmada; listas, disponibilidad y llamamiento; documentos y avisos; datos ficticios sin atajos desechables | Fase 1, más la parte de convocatorias y baremo de la fase 5 |
| F2. Selección y Bolsa completas | Órgano de selección, pruebas, admisión, subsanación, alegaciones, recursos, incorporación, indicadores e integraciones reales | Fase 5 |
| F3. Columna interna | Personal, RPT histórica, ocupaciones, vacantes, planificación, provisión, carrera, certificados, portal del empleado | Fases 2, 5 y 6 |
| F4. Tiempo y operación | Formación, espacio de responsables, Cronos, centros y turnos, teletrabajo, horas, dietas, geolocalización | Fases 3, 4 y 6. Cronos y Dietas se adelantan por orden del 07/10 |
| F5. Materias económicas y reservadas | Nómina, acción social, PRL, igualdad, protocolos, incompatibilidades, disciplina, relaciones laborales | Fase 6 |

Del catálogo se mantienen dos ideas: lo que no se desarrolla ahora no se
simula con reglas arbitrarias para aparentar cobertura, y la protección frente
a refactorizaciones futuras está en fijar límites, identificadores, versiones,
autoridad y contratos neutrales, no en programar todos los módulos por
adelantado.

## 5. Fases y criterios de cierre

Una fase está cerrada cuando su criterio se cumple **en cidonia, con datos
sintéticos y en el navegador**, no en un clon ni en una prueba aislada.

### 5.1 Puertas de cualquier capacidad

Antes de programar una capacidad funcional:

- fuente normativa y provincial identificada, con vigencia y ámbito;
- dueño del dato y órgano competente conocidos;
- procedimiento real y excepciones levantados, o configurados en catálogo si
  RRHH aún no ha decidido;
- perfiles, segregación, conservación y riesgos definidos.

Para darla por terminada:

- dominio y puertos no dependen del adaptador elegido;
- autorización probada, también las denegaciones y la separación entre zona
  pública, del empleado e interna;
- reglas, fuentes y versiones aparecen en resultados y auditoría;
- reintento, concurrencia, idempotencia y fallo parcial probados;
- recorrido real con PostgreSQL, casos negativos y reinicio;
- pantallas y documentos accesibles; ayuda tras el botón «?» actualizada;
- migración ensayada sobre copia de la principal, con copia y vuelta atrás;
- documentación al día en la misma PR (manual de usuario y el documento de
  `docs/producto/` que cambie);
- PR fusionada, subida a cidonia y prueba final en Chrome;
- RRHH ha visto la fase funcionando y la da por buena.

### Fase 0. Base común

Identidad, autorización, administración de perfiles, catálogos, auditoría y
documentos, en lo que Bolsa y CT consumen. Criterio:

1. Cada perfil del recorrido (aspirante, centro, ratificador, RRHH,
   Intervención, firmantes) entra con su certificado sintético y ve solo lo
   suyo; un perfil ajeno recibe 403 con apunte de denegación.
2. Las personas administradoras asignan y retiran esos perfiles desde
   `vec-admin`, con auditoría, sin SQL a mano.
3. La auditoría de RRHH está encendida (`VEC_RRHH_AUDITORIA_ENABLED`) y la
   consulta por persona devuelve sus lecturas y actos.
4. Cada catálogo que el recorrido usa está publicado con versión y huella.

Estado: 1 y 4 casi cumplidos; 2 parcial; 3 pendiente (equipo L).

### Fase 1. Bolsa y Contratación temporal (fase actual)

Se cierra cuando se cumplen estas cuatro condiciones en cidonia:

1. **Recorrido completo de RRHH en Chrome, con reinicio.** Certificados
   sintéticos de personas distintas; petición del centro, ratificación, alta,
   análisis con crédito, oferta desde la bolsa con varias plazas, aceptación,
   informe, fiscalización con un reparo y su subsanación, resolución firmada
   por dos personas, GINPIX, paso a Personal, cese y vuelta de la persona a su
   bolsa. Después se reinician aplicación y PostgreSQL y se repiten las
   consultas: mismos recibos, mismas fechas, ningún duplicado. Queda un acta
   con la respuesta HTTP y una captura por paso.
2. **Selectores encendidos**: `VEC_BOLSA_BORRADORES_ENABLED`,
   `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED`, `VEC_DOCUMENTOS_ENABLED`,
   `VEC_BOLSA_CESE_CT_ENABLED`, `VEC_PERSONAL_B2_GOBIERNO_ENABLED`,
   `VEC_CT_INCORPORACION_ACREDITADA_ENABLED`, `VEC_FIRMA_VERIFICACION_ENABLED`
   y `VEC_RRHH_AUDITORIA_ENABLED`. El aviso de dependencias al arrancar no
   muestra ninguna ausencia.
3. **Ningún «no disponible» sin motivo.** Cada botón o enlace que ve RRHH
   funciona o explica en pantalla qué falta. Sin respuestas 404 ni 503 de rutas
   de Bolsa o CT en el registro durante el recorrido.
4. **Lecturas rápidas**: cuadro de CT, lista de expedientes, ficha, bolsas,
   candidatos y ficha del candidato por debajo de 300 ms en el percentil 95,
   medido por HTTP en cidonia con el volumen de la principal. La base común
   (sesión, módulos, preferencias) por debajo de 100 ms.

Desde el 13/10 el recorrido empieza antes: la persona aspirante se inscribe y
aporta títulos y méritos en «Mi Bolsa», RRHH la admite y constituye la bolsa,
y solo entonces el centro pide. Ese tramo forma parte de la condición 1
(detalle en `FLUJO_APP.md`).

Estado a 13/10: ninguna condición acreditada con acta. En la 2 faltan la
auditoría de RRHH y la verificación de firma (*pendiente de verificar* su
valor actual en cidonia).

### Fase 2. Personal y Organización como base del resto

Lo que Personal y la RPT deben dar a Cronos, Dietas y los demás: ficha del
empleado con historia propia, situaciones, ocupación del puesto y documentos
ligados a cada hecho. Criterio: un alta que llega desde CT aparece en Personal
con su relación, puesto de la RPT y documentos; el cese la cierra; la consulta
histórica responde en menos de 300 ms; y Cronos y Dietas leen al empleado por
un puerto propio. Solo se construye lo que necesiten las fases 3 y 4.

### Fase 3. Cronos

Reescritura de la aplicación PHP actual de control horario, capacidades C1 a
C12 de `docs/estudio_requisitos/ficha_cronos_2026-09-23.md`. Hoy, 0 de 12
cerradas formalmente; apagado en cidonia. Criterio: las doce capacidades con
recorrido real en cidonia. Necesita las respuestas a las dudas 19 a 21, 41,
42, 47, 48 y 100 a 103.

### Fase 4. Dietas

Comisiones de servicio: solicitud, autorización, justificación y liquidación
(`docs/estudio_requisitos/ficha_dietas_2026-09-23.md`). Criterio: una comisión
sintética recorre las tres perspectivas (persona, jefatura que autoriza, RRHH
que liquida) con tarifas de un catálogo versionado. Dudas 22 a 26, 40, 44, 46,
49 a 51 y 104 a 108.

### Fase 5. Sustituir Convoca entero

**Baremador** (orden de Alberto del 30/09): un motor común, primero para
Bolsa y sin olvidar Concursos de provisión, en su propia pestaña o módulo; las
bases son casi iguales. Lo imprescindible es configurar las puntuaciones
(topes, tablas, fecha de referencia) con versión y auditoría. Reglas versionadas
por convocatoria, méritos de oficio desde Personal y la RPT y precisión fija. Del
prototipo de baremador que existe aparte se reutilizan reglas y casos como
pruebas de conformidad, no su código ni su almacenamiento, y nunca sus datos
reales. Anonimizar no es solo cambiar nombres y DNI: también se desplazan
fechas, se alteran cifras y se generan identificadores inventados, fuera de Git
y consultándolo con el DPD. Estudios:
`docs/estudio_requisitos/integracion_baremador_concursos_provision.md` y
`docs/estudio_requisitos/baremacion_configurable_jornada_y_datos_de_oficio.md`.

Procesos selectivos, Provisión y Méritos: publicación de convocatorias,
inscripción, baremación con firma, alegaciones y recursos. Criterio: una
convocatoria sintética se publica, recibe inscripciones con méritos, se bareman
con firma de las personas designadas, admite alegaciones y termina en una
bolsa constituida o una adjudicación, sin usar Convoca.

### Fase 6. Resto de módulos

Por este orden orientativo, cada uno detrás de las bases que necesita:
Formación, Certificados, Acción social, Nóminas (consulta y luego cálculo),
Prevención de riesgos, Igualdad, Incompatibilidades, Régimen disciplinario,
Relaciones sindicales y Canal interno de información. Criterio: el general del
apartado 5.1.

### Fase 7. Paso a uso real

No es programación. Antes de tratar datos reales hacen falta evaluación de
impacto (EIPD), categorización del Esquema Nacional de Seguridad (ENS),
análisis de riesgos, registro de actividades y autorizaciones formales;
clasificación según el Reglamento europeo de IA antes de usar IA en
selección o baremación; Kerberos; correo de la Diputación; portafirmas y
GINPIX reales; y la auditoría de seguridad con herramientas de ataque, que se
reserva para cuando la aplicación esté terminada. Ninguna fase anterior
autoriza datos reales.

## 6. Tareas pendientes

Las tareas de todas las fases, con su equipo, su estado y cómo se comprueban,
están en [`TABLERO_TAREAS.md`](TABLERO_TAREAS.md); los criterios de cierre de
cada fase siguen en el apartado 5.

## 7. Decisiones de producto pendientes de construir

Decisiones tomadas con Alberto el 12/10 que todavía no están programadas
enteras. Cómo las ve cada perfil se describe en `FLUJO_APP.md`.

**Cartera de méritos y puntuación por bolsa.**

- La persona incorpora una vez sus títulos y cursos a una cartera personal en
  «Mi Bolsa», aunque aún no se apunte a ninguna bolsa. Puede pedir retirar un
  mérito con motivo; la retirada no borra lo ya decidido.
- Méritos es dueño del hecho (tipo, fecha, horas, origen, versión, evidencia);
  Documentos, del fichero; Bolsa, de la relación del mérito con cada categoría
  y de los puntos. Un mérito no tiene puntos universales: la misma persona
  puede tener 6,5 en una bolsa y 5,9 en otra.
- Al apuntarse, «Mi Bolsa» enseña por categoría si cada mérito cuenta, no
  cuenta o está pendiente, con motivo, regla, fecha de corte, puntos y total
  provisional.
- La ventana temporal de los méritos sale de las bases publicadas de cada
  convocatoria. No se fija «cinco años» en el código ni se toma por defecto la
  fecha de publicación.
- La persona ordena sus bolsas por preferencia. Ese orden no decide todavía
  adjudicaciones ni renuncias.

**Acreditación de títulos en PDF.**

- No se exige firma electrónica para subir un documento. Hay dos vías: original
  electrónico del emisor (firma, sello o código de verificación) o copia
  escaneada de un título en papel.
- La copia escaneada se recibe como pendiente de acreditación; puede contar en
  el autobaremo provisional, rotulada así, pero no da puntos definitivos. La
  firma del propio aspirante sobre la copia no acredita el título.
- Cuando haga falta cotejar, lo hace una persona habilitada para expedir
  copias auténticas. El cotejo se hace una vez por persona y documento y se
  reutiliza en todas sus bolsas; cada bolsa vuelve a aplicar sus reglas de
  puntos y fecha.

**Inscripción por categorías.** RRHH elige en cada convocatoria, antes de
publicarla, si hay una solicitud por categoría o una conjunta. Cada modalidad
tiene su recibo y su reintento. El borrador B105 solo cubre la separada.

**Revisión, firmas y cambios en una bolsa activa.**

- RRHH es el único editor de baremaciones. Designa la lista de firmantes; cada
  cambio crea versión nueva y obliga a firmar otra vez a todos.
- La pantalla distingue «preparado, pendiente de envío a firma», «enviado a
  firma» (aceptado por el portafirmas) y «firmado» (firma verificada y
  custodiada). Identificarse con certificado no es firmar.
- Si una decisión firmada cambia puntos de una bolsa activa, Bolsa recalcula el
  orden, avisa a RRHH de todas las personas y llamamientos afectados y para los
  llamamientos automáticos de ese tramo hasta que RRHH lo revise.
- El aviso por correo no sustituye la publicación o notificación formal que
  fijen las bases.

**Documentos y antivirus.** Gestor documental Mayan; los ficheros esperan en
cuarentena hasta pasar antivirus (ClamAV) y análisis de PDF. Al principio solo
se admite PDF. Ningún fichero dudoso llega a Mayan ni a RRHH. Informática
opera esos conectores con el perfil de administrador de sistemas.

## 8. Reglas que siguen vigentes de los relevos anteriores

De los relevos de septiembre, guardados ahora en
`docs/historico/AGENTS_historico_hasta_20261013.md`, siguen en vigor estas
reglas, aunque sus rutas, ramas y cifras sean historia:

- Una migración ya instalada con historia no se reaplica ni se revierte.
- El Word original del procedimiento que remitió RRHH no se edita: se cita y se
  trabaja sobre copias o extractos.
- Identificarse con certificado no firma documentos. Ninguna autenticación,
  borrador o evidencia sintética se presenta como firma legal.
- Toda sustitución provisional explica en pantalla qué se hizo y qué falta
  para el uso real, sin palabras como «prueba» o «demo».
- No se asignan permisos por nombre de cargo ni se mezclan solicitante,
  ratificador y RRHH en la misma identidad.
- Un recibo local no acredita la confirmación de un sistema externo (GINPIX,
  portafirmas, correo). La indisponibilidad nunca se interpreta como éxito.

## 9. Planes por módulo

Cada plan tiene arriba su estado y después lo que falta. La fecha es la del
último estado escrito en el plan; varios están desfasados respecto a este
documento (apartado 13).

| Módulo | Plan | Estado en una línea | Fase |
| --- | --- | --- | --- |
| Bolsa | `bolsa.md` (08/10) | En cidonia; falta recorrido completo y acreditación de méritos | 1 |
| Contratación temporal | `contratacion_temporal.md` (08/10) | En cidonia hasta la resolución; falta firma, Personal y recorrido con reinicio | 1 |
| Firmas | `firmas.md` (08/10) | Cinco firmas verificadas en copia H; recorrido único pendiente | 1 |
| Regla común de funciones PostgreSQL | `comun.md` | Vigente: `search_path` fijo en toda función con privilegios del propietario | 0 |
| Administración de perfiles | `administracion.md` (05/10) | Asignación desde pantalla, parcial | 0 |
| Auditoría | `auditoria.md` (05/10) | Instalada; RRHH apagada; exportación judicial aparcada | 0 |
| Copias de seguridad | `copias_seguridad.md` (01/10) | Plan; ejecución desde Administración pendiente | 0, aparcado |
| Organización y RPT | `organizacion_rpt.md` (01/10) | Consulta publicada; ocupaciones y vacantes pendientes | 2 |
| Personal | `personal.md` | Ficha propia y actos de RRHH parciales; recibe de CT | 1 y 2 |
| Portal del empleado | `portal_empleado.md` (04/10) | Inicio y «Mis trámites» montados | 2 |
| Cronos | `cronos.md` (07/10) | Aparcado; 0 de 12 capacidades | 3 |
| Dietas | `dietas.md` (07/10) | Aparcado | 4 |
| Procesos selectivos | `selectivos.md` (07/10) | Aparcado; preparadores locales sin acta firmada | 5 |
| Provisión | `provision.md` (07/10) | Aparcado; bases transcritas | 5 |
| Carrera y Formación | `carrera_formacion.md` (02/10) | Aparcado | 5 y 6 |
| Formación | `formacion.md` (07/10) | Aparcado | 6 |
| Certificados y analítica | `certificados_analitica.md` (01/10) | Aparcado; ningún certificado emitido | 6 |
| Acción social | `accion_social.md` (04/10) | Aparcado; solo plan | 6 |
| Nóminas | `nominas.md` (04/10) | Aparcado; solo plan | 6 |
| Prevención de riesgos | `prevencion_riesgos_laborales.md` (04/10) | Aparcado; solo plan | 6 |
| Igualdad | `igualdad.md` (04/10) | Aparcado; solo plan | 6 |
| Incompatibilidades | `incompatibilidades.md` (04/10) | Aparcado; solo plan | 6 |
| Régimen disciplinario | `regimen_disciplinario.md` (04/10) | Aparcado; solo plan | 6 |
| Relaciones sindicales | `relaciones_sindicales.md` (04/10) | Aparcado; solo plan | 6 |
| Canal interno de información | `canal_interno_informacion.md` (07/10) | Aparcado; entrada pública en rama sin integrar | 6 |

## 10. Cómo se trabaja

### 10.0 Reglas fijas de entrega y subida (14/10)

Alberto las fijó el 14/10 y mandan sobre lo que diga en contra el resto de este
apartado, también el paso 5 de 10.3. El texto completo está en `AGENTS.md`,
sección «Entrega y subida»; aquí va solo el resumen:

1. A cidonia solo sube una candidata congelada y ensayada entera en una copia
   de cidonia, y se instala exactamente lo ensayado. Desde el 15/10 cidonia
   está fuera hasta nueva orden: el ensayo y el recorrido se hacen en una copia
   local.
2. Una PR está lista solo si trae código, guion de activación y la prueba de
   que la app arranca con esa configuración y sin ella.
3. Hay un único caso de aceptación del recorrido:
   [FLUJO_APP.md, apartado 4.0](FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).
4. Ninguna configuración nueva tumba la app; cidonia nunca queda apagada
   (para cuando vuelva).
5. Con datos ficticios, si falta un dato se ajusta el dato, no se programa.

### 10.1 Encargo y trabajo paralelo

- **Un identificador de tarea por agente.** Nadie toma otra tarea por
  iniciativa propia ni amplía la suya.
- **Rama y carpeta de trabajo propias**, fuera de `/tmp`, con la lista de
  ficheros que va a tocar declarada y sin solaparse con otro equipo. Si dos
  tareas necesitan el mismo fichero, van una detrás de otra, o primero se
  separa una pieza propia; nunca se edita el mismo fichero a la vez.
- **Quien programa no integra ni aprueba su propio trabajo.** Otro agente
  revisa y da su visto bueno (`GO`) o lo rechaza (`NO-GO`).
- **No se fusionan, rebasan ni limpian ramas ajenas.**
- **No repetir código hecho.** Antes de escribir se busca en el índice de
  código, en `main` y en las ramas y PR abiertas.
- **Cada encargo nombra las skills** que tocan: `usabilidad-vec`,
  `aspecto-vec`, `disenar-sistema-visual-vec` e `impeccable` para pantallas;
  `programar-backend-vec`, `persistir-autorizar-vec` y
  `programar-interfaz-vec` para programar; `revisar-sql-vec` y `ensayar-sql`
  para SQL; `humanizer` para textos que lee una persona; `security-audit` para
  autenticación, permisos, SQL o fronteras HTTP; `revisar-cambios-vec`,
  `documentar-entregar-vec` y `pr-vec` para cerrar. Y los documentos de
  `docs/producto/` que el equipo debe leer.
- **Modelo justo**: el más barato que baste. Esfuerzo alto solo para programar
  SQL, permisos, identidad, firma y datos personales, y para las revisiones
  finales; nunca para buscar ni para repetir revisiones. Como mucho seis
  subagentes vivos por equipo.
- **Atascos**: tres fallos iguales o 20 minutos sin avanzar, BLOQUEO en el
  canal, sin subir el esfuerzo.
- **Nada en cola**: un encargo nuevo empieza ya con un equipo o subagente
  libre.
- **Plantilla del encargo**: objetivo y paso de RRHH al que sirve, ficheros
  propios y ficheros ajenos que no se tocan, invariantes que hay que conservar y
  pruebas que lo cierran. En trabajos largos, el agente vuelve a leer sus
  instrucciones cada 30 minutos para no desviarse.

### 10.2 Tamaño de cada tarea (minitareas)

- Una minitarea tiene una sola responsabilidad visible y un solo criterio de
  cierre.
- Código y pruebas forman un commit que compila y pasa sus pruebas focales; no
  se admiten cortes que dejen una API a medias, una migración sin quien la use
  o una ruta registrada sin autoridad.
- Tamaño de la PR: tope de 1.500 líneas y objetivo de 400 escritas a mano (sin
  contar lo generado ni los datos de prueba). Sustituye el umbral anterior de
  tres ficheros o doscientas líneas. Si una tarea necesita dos motivos
  distintos en el mensaje, se divide.
- Una capacidad grande se expresa como un grafo de minitareas con
  dependencias, no como un identificador que acumula contrato, adaptador,
  composición, pantalla y recorrido.
- Dividir no significa entregar cambios cosméticos: la unidad mínima es una
  capacidad que compila, está protegida y se puede comprobar.
- Tamaño de fichero: objetivo de 500 líneas, aviso a partir de 800. No es un
  tope: un fichero coherente de 1.000 líneas es mejor que tres de 400 que solo
  se entienden juntos. Al pasar el aviso se congela el tamaño en
  `scripts/tamano_ficheros_base.txt` y se justifica en el mensaje del commit.
- Lo nuevo de CT y Bolsa va en ficheros nuevos; la lista de ficheros que no
  conviene engordar está en los planes de CT y Bolsa.
- **Lo que RRHH no ha decidido se configura** en un catálogo versionado y se
  anota como pregunta numerada en `dudas.md`, nunca como constante.
- **Ninguna configuración nueva puede impedir que la aplicación arranque.** Si
  falta, falla solo esa función y lo explica en pantalla.

### 10.3 Ciclo de entrega

Orden vigente de Alberto del 13/10: menos esperas y una mejora visible en cada
subida.

1. **PR pequeña** con código, pruebas focales y documentación juntos, la línea
   `CONFIG NUEVA:` (o «ninguna») y la lista de SQL a instalar si las hay.
2. **Pruebas en local, solo focales**: `gofmt`, `go test`, `go test -race` y
   `go vet` del paquete afectado, `git diff --check`, la batería web
   (`node --test` sobre los `*.test.mjs`) si toca la web, y ensayo en
   PostgreSQL 18 si hay migración.
3. **Revisión independiente**: dos para lógica de negocio, permisos,
   identidad, cifrado o SQL con permisos; una para lo demás (apartado 10.5,
   regla 5). Las pantallas pasan por el revisor de usabilidad; el SQL, por el
   revisor SQL con ensayo en el clon local de la principal; lo que toca
   autenticación, permisos, SQL o fronteras HTTP, por `security-audit` y
   Semgrep con reglas locales.
4. **Fusión**: Claude lee el diff y fusiona cuando pasan las puertas rápidas de
   la CI (secretos, PostgreSQL y artefactos, unos dos minutos;
   `.github/workflows/ci.yml`). La puerta de calidad completa
   (`scripts/verificar_calidad.sh`, unos 35 minutos) corre en `main`; si
   falla, se arregla encima. No se repite la puerta completa sobre el mismo
   commit.
5. **Subida a cidonia** tras cada fusión con algo visible, sin esperar a
   juntar. Desde el 15/10 cidonia está fuera: este paso se sustituye por el
   recorrido en una copia local, y lo que sigue vale para cuando vuelva. Cada subida hace copia fría de la base, instala y deja una vuelta
   atrás automática; si la comprobación falla, se vuelve a la subida anterior.
   Las SQL se ensayan antes sobre una copia de la principal y se instalan una
   vez, en el orden de su lista causal (`deploy/principal/lista_sql_*.txt`).
   Las tareas de un solo uso dentro del pod llevan `--restart=no`.
6. **Prueba final** en Chrome (Playwright con el Chrome del sistema) a 1440 y
   390 px, sin errores de JavaScript y sin cookies ni almacenamiento del
   navegador usados como sesión.
7. **Limpieza**: quien creó carpetas de trabajo y ramas temporales las borra
   tras la fusión. Una rama con SQL pendiente de ensayo o con `NO-GO` se
   conserva.

Esto sustituye el ciclo anterior de `AGENTS.md`, que exigía la puerta completa
antes de cada integración. Lo que no funciona se termina con un equipo; no se
quita del menú para ocultarlo.

### 10.4 Formato de la entrega

```text
Tarea:
Estado: GO / NO-GO / bloqueada
Commit(s):
Archivos modificados:
Resultado:
Pruebas ejecutadas:
Pruebas omitidas y motivo:
Seguridad, privacidad, i18n y accesibilidad:
CONFIG NUEVA:
SQL a instalar y orden:
Limitaciones:
Riesgos:
Siguiente tarea desbloqueada:
Revisión independiente:
```

El agente no da su tarea por cerrada ni cambia porcentajes; lo hace dirección
al integrar.

### 10.5 Reglas de simplificación (16/10)

Consenso de dirección con Fable y Astra, aprobado por Alberto el 16/10. Valen
para toda PR desde ese día; el resumen está en `AGENTS.md`.

1. **Código muerto.** Si un cambio deja sin uso una función, una variante, una
   versión de rol, un fichero o una prueba, se retira en la misma PR. El código
   muerto que se vea en lo que se toca se retira también, demostrando que la
   protección que daba sigue cubierta. Mover ficheros no cuenta como reducción.
2. **Tamaño de PR.** Tope de 1.500 líneas y objetivo de 400 escritas a mano,
   sin contar lo generado ni los datos de prueba (apartado 10.2).
3. **Núcleo de autorización.** Las migraciones nuevas de
   `deploy/postgresql/autorizacion_atestada_v3` solo las escribe su equipo
   dueño, el R. Las que ya están en PR siguen el orden fijo del tablero
   (`TABLERO_TAREAS.md`, apartado 3). Como mucho se reconstruye la función
   central una vez por semana.
4. **Lecturas.** Una lectura nueva se añade como fila de la lista existente, no
   como función ni versión nueva.
5. **Revisiones.** Dos independientes solo para lógica de negocio, permisos,
   identidad, cifrado o SQL con permisos. Para lo demás basta una.
6. **Controles y excepciones.** Cada control nuevo dice de qué protege y qué
   prueba lo comprueba. Cada excepción temporal tiene responsable y condición
   de retirada. No se añade ninguna capa sin un consumidor del recorrido
   vigente.
7. **Lo opcional** falla cerrado sin tumbar el arranque, con un error de
   catálogo que dice qué falta.
8. **Objetivos medibles.** Copia local en menos de 15 minutos con un guion
   único (F0-28); CI que instala todas las migraciones del recorrido de la
   fase 1 y ejecuta sus pruebas SQL (F0-29); lecturas por debajo de 300 ms en
   el percentil 95.

## 11. Riesgos y bloqueos

| Riesgo o bloqueo | Qué frena | Quién lo resuelve | Mientras tanto |
| --- | --- | --- | --- |
| Plantillas Word de contratos, informes y resoluciones (duda 124) | Documentos definitivos | RRHH | Plantillas de catálogo cambiables |
| Oferta al SAE: datos, canal, criterios de Mesa General y documento que acredita la selección (72, 73, 126) | Oferta al SAE | RRHH | Acceso al SAE retirado de Inicio hasta tener gestión |
| Delegaciones de firma y suplencias; perfiles fijos (122, 128, 143) | Quién firma cada paso | RRHH y Secretaría General | Perfiles fijos asignados por administración |
| Portafirmas Firmadoc o AutoFirma (74) | Envío real a firma | RRHH e Informática | Firma con GrxFirma; el conector de Firmadoc existe apagado |
| SICAL y GINPIX (125, 127) | Confirmación de los sistemas externos | Informática | Registro de la operación en VEC, sin confirmación externa |
| Servidor de correo (88) | Avisos reales | Informática | Buzón de pruebas |
| Toma de posesión eficaz y fecha de efectos del cese (140) | Paso a Personal y cese definitivos | RRHH | Valor de catálogo |
| Plazo para aportar documentos tras aceptar; audiencia antes de la baja (18, 62) | Reglas de baja | RRHH | Plazo en el catálogo de cada bolsa |
| Urgencia, coste sin fecha de fin, plazo tras llamar por teléfono (63, 146, 151) | Reglas de CT y Bolsa | RRHH | Valores de catálogo |
| Que el correo valga como publicación (art. 8.1 del Reglamento de bolsas) | Plazos de la oferta | RRHH | Correo como aviso, sin efecto de publicación |
| Lista pública de cada bolsa (17, 83) | Qué ve cada persona | DPD | Solo integrantes identificados, documento parcialmente oculto |
| Uso del correo y teléfono de CONVOCA (45); conservación de actuaciones de aspirantes (97) | Contacto y archivo | DPD | Contacto enmascarado en listados, completo en la ficha con constancia |
| Firma: recorrido único e instalación pendientes | Condición 1 de la fase 1 | Equipo H y P | Cinco firmas verificadas en dos expedientes; falta el caso único |
| Kerberos | Entrada del personal interno | Informática | Certificado personal como único factor, solo desarrollo y presentación |
| Datos reales tras EIPD y ENS | Uso real | Diputación | Datos sintéticos con nombres verosímiles |
| SQL que no encaja con la base real (caso AD250) | Todo lo fusionado detrás | Dueño de la SQL y Claude | Ensayar siempre sobre una copia actual; sustituir, nunca reaplicar |
| Cuota de los equipos | Ritmo de trabajo | Alberto y dirección | Pocos subagentes, esfuerzo medio por defecto, cierre ordenado al 95 % |
| Lectura del cuadro de CT por encima de 300 ms en cidonia | Condición 4 de la fase 1 | Sin dueño actual | Medir tras la próxima subida |
| Disco lleno por la caché de Go | Compilaciones y pruebas | Cada equipo | Vigilar espacio y limpiar la caché |

Nada de esto impide cumplir las cuatro condiciones de la fase 1 con valores de
catálogo, que se cambian sin tocar código. Las preguntas que una norma
publicada ya responde se retiran citando la norma.

## 12. Calendario orientativo por hitos

Sin fechas, salvo la que fijó Alberto. Cada hito empieza cuando el anterior
cumple su condición.

| Hito | Condición de llegada | Depende de |
| --- | --- | --- |
| H1. Subida desbloqueada | AD262 y lo fusionado sin subir instalados en cidonia, sin 404 ni 503 nuevos | Ensayo de AD262, hecho en copia |
| H2. Presentación a RRHH | Alberto recorre él mismo, cambiando de perfil, el caso de `FLUJO_APP.md` §4.0: empieza con la inscripción del aspirante en la bolsa y el alta de sus títulos, y termina con el cese y la vuelta a la bolsa. Prevista para el 13 o el 14 de octubre, según Alberto | H1 y las tareas de fase 1 de O, M, K y H |
| H3. Bolsa y CT cerradas | Las cuatro condiciones de la fase 1, con acta | H2 y la firma de punta a punta |
| H4. Base de Personal y RPT | Criterio de la fase 2 | H3 |
| H5. Cronos | Doce capacidades en cidonia | H4 y respuestas de RRHH |
| H6. Dietas | Comisión sintética en las tres perspectivas | H4 y respuestas de RRHH |
| H7. Convoca sustituido | Criterio de la fase 5 | H3, Méritos y firma |
| H8. Resto de módulos | Cada uno con su criterio | Las bases que necesita |
| H9. Uso real | Fase 7 | Decisiones de la Diputación |

Si un hito se retrasa por una respuesta pendiente, se sigue con las partes que
no dependen de ella y se anota en el plan del módulo; no se abre una línea de
trabajo paralela que la sustituya.

## 13. Cómo mantener este documento

**Quién**: Claude, como dirección. El estado de cada tarea no se lleva aquí,
sino en `TABLERO_TAREAS.md`, que también actualiza solo dirección.

**Cuándo**:

- tras cada subida a cidonia: se actualiza el apartado 2.1 con el número de
  subida y lo que entró, y se vacía lo que ya no esté en 2.2;
- al cumplirse un criterio de fase o un hito;
- cuando Alberto cambie el orden o la prioridad (apartado 1).

**Cómo**: cambios pequeños, no reescrituras. La fecha y el commit de base de la
primera línea se actualizan siempre. Lo no comprobado se marca *pendiente de
verificar*. El detalle técnico de cada tarea (hashes, ensayos, cifras de
medición) va en el plan del módulo, no aquí. No se pegan secretos, rutas
privadas ni datos personales.

**Documentos que este plan sustituye.** Este plan es la única referencia de su
contenido. `OBJETIVOS.md` y el catálogo funcional se retiraron el 13/10 (su
versión anterior sigue en la historia de Git); `ESTADO_PROYECTO.md` quedó como
una nota que remite aquí:

- `docs/plan_modulos/OBJETIVOS.md`: absorbido entero.
- `ESTADO_PROYECTO.md`: absorbido el estado actual; la historia sigue en Git.
- `docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md`: absorbidos
  el orden de implantación (apartado 11), las puertas (12) y el alcance
  inmediato (13). El inventario de capacidades por materia está en `PRD.md`.
- `AGENTS.md`: absorbidos «Prioridad vigente», «Trabajo paralelo»,
  «Granularidad obligatoria», «Ciclo obligatorio de entrega» y «Entrega», y
  las reglas aún vigentes de sus relevos de septiembre (apartado 8). La
  arquitectura, la seguridad, la i18n y la calidad de `AGENTS.md` van en
  `TRD.md` y `DISENO_UI_UX.md`.
- `RELEVO_CLAUDE_20261012.md` (fuera de Git): absorbidas las decisiones de
  producto (apartado 7) y la tabla de ramas conservadas (apartado 2.3). Los
  hashes exactos y las rutas privadas siguen en ese relevo.
