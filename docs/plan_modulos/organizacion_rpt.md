# Organización y RPT: inventario y trabajo siguiente

La consulta publicada permite quitar la búsqueda aplicada y volver a la primera
página, conservando la pestaña. El buscador mantiene el texto aún sin enviar y
el foco durante la carga. Esta mejora no cambia la fuente de la RPT ni acredita
ocupación, vacantes o vigencia administrativa.

Fecha: 1 de octubre de 2026. Base contrastada: `origin/main@0a62a3ea68e58fbf890885a2f59cc80343107e18`.
Este plan recoge código publicado y candidatas separadas; no declara ORG/RPT completos.
El inventario se hizo sobre esa base. Main avanzó a `f49e01089` con CT159;
el código de Organización y RPT conserva los mismos bytes.
Se remata el comparador ya producido y se conserva lo pendiente para mañana.

Fuentes: [especificaciones](../../ESPECIFICACIONES_AGENTES.md),
[catálogo funcional, apartado 4](../estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md#4-personal-organización-rpt-y-planificación)
y [modelo histórico](../estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md).

## Qué existe en main

En las rutas abreviadas, `personal/` empieza en `internal/modules/`,
`vec/` en `internal/` y `bootstrap/` en `internal/app/`.
Las vistas están en `web/static/portal-empleado/`.

| Capacidad | Implementación y consumidor | Alcance comprobado |
| --- | --- | --- |
| ORG-001, estructura | `personal/domain/organizacion.go`, `ports/estructura_organizativa.go`, adaptadores `catalogosvec` y `organizacionpublica`; vista `vista-estructura-organizativa-publica.js`. | Consulta de unidades publicadas de referencia. No acredita cadena de mando, delegaciones de competencia ni suplencias. |
| ORG-001, edición | `ports/edicion_organizacion.go`, `adapters/postgres/organizacion.go`, frontera de Organización y pantalla `/portal-empleado/organizacion/`. | Revisión y huella esperadas, cambio y recibo. Se conserva el circuito existente; no extender su permiso a otras capacidades. |
| RPT-001, categorías | [PR #222](https://github.com/aavidad/VEC_Diputacion_app/pull/222) y [#244](https://github.com/aavidad/VEC_Diputacion_app/pull/244), ambas fusionadas; `vec/ports/catalogos_rpt_{lectura,usos}.go`, adaptadores PostgreSQL y `/portal-empleado/categorias-rpt/`. | Lectura y usos con autorización V3; pantalla para consultar y preparar gestión. El gobierno final no queda cerrado por estas PR. |
| RPT-003, lectura publicada | `bootstrap/personal_consultas_publicas.go` → `personal/adapters/rptpublica/fuente.go` → `application/consulta_rpt_publica.go` → `/api/vec/personal/rpt-publica` → `vista-rpt-publica.js`. | 842 filas de puesto tipo, 1.714 dotaciones agregadas y 145 categorías. No son 1.714 puestos individuales reconciliados. |
| RPT-002/003/004, historia | `personal/domain/organizacion_historica.go`, puerto y aplicación homónimos, adaptador PostgreSQL y `vec/adapters/httpapi/personal_organizacion_historica.go`. | Separa unidades, puestos tipo, dotaciones, plazas, puestos individuales y vínculos, con efectos y conocimiento. Código existente sin montaje raíz. |
| RPT-004, importación | `domain/importacion_organizacion_historica.go`, aplicación, puertos, adaptadores y `personal_importacion_organizacion_historica.go`. | Preparar, conciliar y publicar con fuente, huella, revisión y predecesora. No está compuesto en la raíz. |
| RPT-005/006, Personal B2 | `registro_empleado_b2.go`, aplicación y PostgreSQL; `interna/personal_b2_montaje.go` y `registro-b2.js`. | Ficha, ocupaciones y consulta `/api/vec/personal/vacantes`. Vacante exige cobertura completa y estado `vacante_sin_ocupacion`; no prueba cubribilidad. |

`organizacion/historico.js:130` mantiene consulta e importación desactivadas.
El constructor de cada servicio histórico solo tiene consumidores en pruebas,
según gopls. La prueba de montaje conserva esa correspondencia con la web.
El antiguo `CatalogService`/`RPTPosition` y su importación `Replace` sustituyen
fotografías: no reutilizarlos como autoridad histórica. El importador PDF sigue
siendo preparación, sin fuente corporativa reconciliada ni autorización para
provisión o informes de vacantes. No modificar originales de RRHH.

## Huecos que quedan

| Código | Resultado pendiente |
| --- | --- |
| ORG-001 | Historia recorrible por RRHH y relaciones organizativas acreditadas. Un nombre de cargo no concede competencia. |
| RPT-002 | Plantilla por ejercicio, dotación presupuestaria, actos de creación/amortización/reserva y vínculo OEP, con historia. |
| RPT-003 | Inventario individual reconciliado y condiciones completas del puesto: requisitos, funciones, provisión, nivel, complementos y vigencia. |
| RPT-004 | Montaje de consulta/importación existentes, fuentes y aprobaciones reales; comparación conectada a lecturas autorizadas. |
| RPT-005 | Circuito completo persona–relación–plaza–puesto, titularidad, fechas, actos y reserva. Propietario de RPT-005: M; B aporta la relación de la persona por puerto. |
| RPT-006 | Proyecciones distintas de dotación vacante, puesto sin ocupante y necesidad cubrible, con cobertura e incertidumbre. Propietario de RPT-006: M; coordinar el contrato de relación con B. |

El informe sintético de diferencias descrito abajo permite revisar dos cortes.
No cubre por sí solo importación, autorización, ocupación ni vacantes reales.

## Minitareas en orden de ejecución

### 1. Cerrar la búsqueda de estructura ya producida

- Propietario: M; hoja `vista-estructura-organizativa-publica.js` y su prueba.
- Dependencia: turno compartido F→M para importadores, padres y versiones de caché.
- SQL/servidor: ninguno. No cambiar fuentes, permisos ni estructura de datos.
- Cierre: misma búsqueda por nombre/adscripción desde el portal, caché anterior
  superada, teclado y Chrome PC/móvil; GO del conjunto y PR de producto aparte.

### 2. Comprobar precondiciones y después montar la consulta histórica

- Propietarios: D, perfiles/contexto/SQL y entorno; M, caso de uso, frontera
  y composición en su turno de archivos compartidos; Dirección, revisión e integración.
- Archivos: `personal/application/consulta_organizacion_historica.go`,
  `personal/ports/organizacion_historica.go`, `personal/adapters/postgres/organizacion_historica.go`,
  `vec/adapters/httpapi/personal_organizacion_historica.go`, composición interna
  existente y `organizacion/historico.js`. Reservar la ruta raíz concreta antes de editar.
- Dependencia: concesión nominal propia para `personal.organizacion_historica.consultar`,
  fuente publicada y puerto autorizador real. Primero se comprueban esas
  precondiciones. Si falta la fuente, se completa la preparación/publicación
  mínima de la tarea 4 antes del montaje. Los perfiles de categorías no
  conceden por sí solos lectura histórica. Mantener la pantalla desactivada hasta entonces.
- SQL/servidor: comprobar postimagen de Personal 000010 y AD3 000051 existentes;
  D acredita instalación compatible. No reaplicar migraciones ni crear tablas sustitutas.
- Cierre: navegador → contexto/perfil fijo → V3 → consulta PostgreSQL → evidencia;
  cortes y cobertura idénticos tras reinicio, denegación y revocación. Revisión sensible independiente.

### 3. Retomar el gobierno de categorías conservado

- Propietarios: M, rama de gobierno; D, convergencia B→A y dependencias nominales.
- Archivos: candidata Cat4/AD134, README y pruebas de la rama `a9001c097`.
  No reconstruir lo aportado por #222/#244 ni dar por integradas #249/#256.
- SQL/servidor: fijar postimagen B→A, CA21/AUT25/IS10 y orden exacto; ensayo final
  PostgreSQL 18 y dos revisiones del mismo hash antes de instalación por D.
- Cierre: separación editor/revisor, confirmación V3, CAS, auditoría, recibo
  recuperable y deshabilitación con historia. Los ensayos antiguos no cierran el circuito final.
  La numeración expresa prioridad; este gobierno no bloquea tareas independientes
  que consuman categorías ya publicadas.

### 4. Conectar fuentes y completar plantilla/puestos, sin otro importador

- Propietario: M, importación histórica existente; RRHH, diccionario y actos;
  Documentos, custodia; D, perfiles y persistencia; B, uso en ocupaciones.
- Archivos: `personal/*/importacion_organizacion_historica.go` y adaptadores actuales.
- Dependencias: definición de códigos, identidades estables, reconciliación de
  dotaciones con unidades y aprobación de fechas/actos. Activar por fases verificadas.
- SQL/servidor: verificar Personal 000011 y sus dependencias en la postimagen
  acordada. Si hace falta migración nueva, reservar número y justificarla antes de crearla.
- Cierre: preparación no autoritativa → conciliación → aprobación/publicación,
  sin sobrescribir versiones, y lectura recuperable del mismo corte.
  Verificar ejercicio y presupuesto, creación/amortización y vínculo con OEP;
  requisitos, funciones, provisión, nivel y complementos del puesto; acuerdos,
  BOP, incidencias y comparación de versiones desde lecturas autorizadas.

### 5. Completar ocupación, reserva y vacantes de RPT

- Propietario: M, RPT-005 y RPT-006. B aporta persona y relación de servicio por
  puerto: referencia, versión, vigencia y procedencia mínima. Sin tablas cruzadas.
- Archivos previstos de M: `rpt_ocupaciones`, `rpt_reservas` y `rpt_vacantes` en
  dominio, puertos, aplicación y adaptadores de Personal; consumidor RPT propio.
  Reutilizar o extraer las operaciones B2 actuales antes de activar el reemplazo:
  una sola autoridad de ocupación, con historia y recibos anteriores consultables.
- Dependencias: contrato de B, vínculos plaza–puesto, actos de ocupación/reserva,
  presupuesto y cobertura. No inferir la relación desde nómina o certificado.
- SQL/servidor: M produce su candidata; D valida composición e instalación.
  Reservar números y ordenar en `ORDEN_SQL_NUCLEO.md` si hacen falta cambios.
  Ensayo y dos revisiones exactas para SQL/permisos/datos personales.
- Cierre: ocupación y reserva con recibo recuperable; tres proyecciones separadas
  de dotación vacante, puesto sin ocupante y necesidad cubrible. Revalidar tras
  reinicio; una fuente ausente conserva incertidumbre y no declara vacante.

### 6. Completar tipos y relaciones de organización

- Propietario: M, tipos y relaciones históricas; B, referencias personales y actos
  por puertos; D, autorización y persistencia; RRHH, fuente y alcance.
- Archivos previstos: dominio, puertos y aplicación de organización de Personal;
  adaptadores actuales. Reservar contratos y archivos concretos antes de escribir.
- Dependencias: organismos, áreas, servicios, centros, unidades y jerarquías;
  significado, vigencia y alcance de delegaciones de competencia y suplencias.
- SQL/servidor: aprovechar historia existente; si requiere otro modelo o circuito,
  reservar migración y reestimar antes de ampliarlo. No conceder permiso por cargo.
- Cierre: consultar una relación y su rectificación por efectos y conocimiento,
  con acto, procedencia, permiso propio y acceso minimizado a referencias de personas.

## Reparto de archivos con Personal

La ubicación física actual es `internal/modules/personal/`; la propiedad de
capacidad fijada por Dirección no autoriza dos escritores del mismo archivo.

| Propietario | Archivos y responsabilidad |
| --- | --- |
| M | `domain/organizacion*.go`, `domain/importacion_organizacion_historica.go`, `domain/estructura_organizativa_publica.go`, `domain/rpt_publica.go`, `domain/comparacion_organizacion_historica.go` y sus pruebas; puertos/aplicación/adaptadores específicos de esos recorridos; `adapters/organizacionpublica`, `adapters/rptpublica`; vistas de organización/RPT y CLI del comparador. |
| M, nuevos | `rpt_ocupaciones`, `rpt_reservas`, `rpt_vacantes` por capas y consumidor propio: RPT-005/006. Antes de crear, localizar y reutilizar el comportamiento B2 existente. |
| B | Persona, relación de servicio, ficha, servicios y actos personales; `relacion_empleado*`, `ficha_propia*`, registro e incorporación del empleado. Publica su contrato mínimo para M. |
| Mixtos, escritor acordado | `registro_empleado_b2*`, `interna/personal_b2_montaje.go`, `registro-b2*` y manifiesto Personal. B conserva la edición actual; M acuerda con B la extracción/transición de ocupaciones y vacantes antes de tocar estos archivos. |
| D/común | Identidad, contexto y autorización; consume la fuente histórica de responsables, delegaciones y suplencias de M. El Portal del empleado I solo consume esa capacidad. |
| En turno M | Padres, manifiestos web, importadores y `dudas.md`, tras LIBERO F→M y con un solo escritor. Dirección revisa e integra. |

Propuesta posterior: estudiar un módulo enchufable `organizacion` al estabilizar
los puertos con Personal. Hoy se conservan ubicación, contratos e historia;
no duplicar dominio, tablas ni autorización para preparar esa separación.

## Fase posterior, fuera de esta estimación

PLA-001 (necesidades, escenarios y crédito) y PLA-002 (escenarios del capítulo I
conciliados con presupuesto/nómina) se abordarán después. No hay programación
ni estimación para ellos en este corte; no producen decisiones automáticas.

## Decisiones pendientes de RRHH y Sistemas

Propuestas pendientes de numerar en `dudas.md` al recibir el turno M:

- Fuente maestra, diccionario y significado de códigos de plaza, puesto y datos de reserva.
- Identidades estables, cambios/reutilización de códigos y conciliación plaza–puesto–dotación.
- Actos y fechas que acreditan efectos; qué documento/custodia debe conservar cada versión.
- Autoridades para conciliar/aprobar, ámbitos de lectura y separación de funciones.
- Reserva, cobertura temporal/definitiva y criterio para declarar una necesidad cubrible.
- Fuentes suficientes para cada proyección histórica y tratamiento de datos en conflicto.

Contrastar primero las preguntas existentes de Personal B; no duplicar ni
inventar números. Una decisión pendiente bloquea su efecto concreto.

## Por dónde empezar mañana y trabajo en curso

Primera minitarea: retomar `trabajo/codexm-org-busqueda-20261001@a6b9eae35`,
comprobar si F ha liberado el turno y cerrar su cadena de caché. Si el turno
sigue pendiente, hacer solo la comprobación de precondiciones de la tarea 2;
no activar el histórico ni abrir otro desarrollo. Después se continúa por la tarea 2.

| Fuente conservada | Estado y siguiente dependencia |
| --- | --- |
| `trabajo/codexm-org-busqueda-20261001@a6b9eae35` | WIP de hoja, GO independiente, Node 9/9 y Chrome focal; caché/importadores pendientes F→M, sin PR. |
| `trabajo/codexm-rpt-informe-20261001@c02852f30` | [PR #305](https://github.com/aavidad/VEC_Diputacion_app/pull/305), borrador. Incluye núcleo `ae87d7cb8` y CLI/informe `cmd/vec-comparar-organizacion`; GO independiente, focales y Chrome verdes. Calidad global y CI en curso; no publicado en main. |
| `origin/trabajo/codexm-rpt-escritura-retoma-20260930@a9001c097` | Gobierno de categorías pendiente B→A, dependencias nominales y ensayo final; preservar. |
| [#249 IS10](https://github.com/aavidad/VEC_Diputacion_app/pull/249), [#256 CA21](https://github.com/aavidad/VEC_Diputacion_app/pull/256) | Abiertas; postimágenes de D, no capacidades instaladas. |
| [#290](https://github.com/aavidad/VEC_Diputacion_app/pull/290) `d957f0ec9` | Borrador abierto, CI 5/5; preparación reproducible de clon H6 hasta SQL62. No manual ni instalación. |
| `trabajo/codexm-clon-preimagenes-nominales-20261001@e2c5b0388` | NO-GO preservado deliberadamente como evidencia; no borrar, instalar ni usar como base aprobada. |

D prepara la intervención del servidor mediante copia fría; ese trabajo no
es el clon H6b. AD132 queda aparte. Este plan no acredita instalación nueva.
Solo Dirección integra en main; se conservan las ramas pendientes/rechazadas.

## Estimación

Horquilla de trabajo con un equipo Codex y varios subagentes, cortes de PR de
1–3 horas, revisión independiente y CI. Incluye recorridos y recuperación;
las revisiones de SQL o permisos requieren dos lectores.

| Minitarea | Horas de un equipo |
| --- | ---: |
| 1. Cerrar búsqueda, caché y PR | 1–3 |
| 2. Precondiciones, autorización y consulta histórica montada | 16–28 |
| 3. Gobierno de categorías conservado, convergencia y ensayo | 20–36 |
| 4. Fuente/importación y recorrido de plantilla y puestos | 48–80 |
| 5. Ocupación, reserva y las tres proyecciones; contrato de B | 40–64 |
| 6. Tipos, jerarquías, delegaciones y suplencias | 20–36 |
| **Total técnico estimado** | **145–247** |

A ocho horas por jornada: **19–31 días de un equipo**. Con dos equipos y
archivos disjuntos: **14–23 días**, porque fuentes, autorización y publicación
siguen condicionando el orden. Esta horquilla es planificación; no acredita
que las capacidades estén cerradas ni promete una fecha de entrega.

Fuera del equipo M: D/Sistemas aporta instalación y perfiles; Personal B,
persona y relación de servicio por puerto; RRHH, diccionario, fuentes y decisiones de
cobertura. Si entregan esos contratos y datos al comenzar, prever **3–8 jornadas
adicionales de coordinación y validación**. Pueden solaparse con tareas
independientes; no se suman automáticamente al total. Sin fuentes o respuestas
no puede fecharse la espera institucional. Una migración o un conector no
previstos exige revisar esta estimación antes de ampliar el corte.

## Consenso Astra

Astra dio GO arquitectónico al reparto y orden del borrador `eb7f7739`.
Se incorporaron sus precisiones: precondiciones antes del montaje; publicación
mínima de fuente antes de consulta si falta; IS10 explícita; ruta correcta de
aplicación y tareas independientes de categorías separadas del gobierno pendiente.
La segunda ronda añadió la tarea 6 y su margen: las cinco tareas iniciales
no cubrían los tipos, delegaciones de competencia y suplencias de ORG-001.
La estimación queda condicionada a las fuentes, contratos y circuito acordados.

Acuerdos: cerrar primero la búsqueda conservada; reutilizar historia e
importación con concesión V3 propia; M conserva ocupaciones y reservas RPT;
B aporta persona y relación de servicio por puerto. El comparador sintético no acredita vacantes
ni historia corporativa. Las candidatas SQL y la evidencia NO-GO se preservan.
No se abre la PR del plan hasta recibir el GO de Dirección sobre su SHA exacto.

La ronda posterior aplica la decisión de Dirección de las 17:40: RPT-005/006
pertenecen a M; B aporta relación/persona por puerto y conserva los archivos
mixtos hasta acordar su transición. Se evita una segunda autoridad de ocupación.
La tarea 5 se estima una sola vez en M. PLA-001/002 quedan para después.


## Consulta de plazas: búsqueda en la página — 4 de octubre de 2026

Inventario sobre `origin/main@7141dedb5`: la consulta B2 de vacantes está montada
con autorización nominal y lectura PostgreSQL. Aporta plazas de plantilla sin
ocupación registrada y conserva corte, versión estructural, acto y fuente.
La hoja ya distingue ese resultado de la ocupación del puesto y de la necesidad
cubrible, que siguen pendientes de sus fuentes y criterios.

La hoja añade búsqueda por código literal de plaza, denominación de unidad o
puesto, dentro de la página recibida. Indica coincidencias y total de esa página;
permite limpiar la búsqueda y distingue filtro sin coincidencias de consulta
sin registros. La búsqueda no cambia ámbito, corte, origen ni autorización y
no consulta más páginas. Cada hoja nueva parte sin filtro; los estados de fallo
no reutilizan la lista anterior. Los padres y sus versiones de caché los actualiza
Dirección en su turno de integración.

Esta mejora usa el consumidor existente. No añade descarga ni publica fuentes,
ocupaciones o reservas. Para obtener puestos sin ocupante y necesidades cubribles
faltan lecturas nominales propias, cobertura de ocupaciones/reservas, vínculos y
criterios acreditados. El contrato `LectorRelacionParaRPTV1` sigue preparado:
una concesión B2 no lo convierte en una lectura autorizada para otro consumidor.
La política de acreditación de fuentes de Personal 000011 continúa pendiente de
una autoridad admitida; el revisor local ya integrado no la sustituye.

## Preparación reutilizable de fuente — 4 de octubre de 2026

Inventario sobre `origin/main@66233cc3e`: Personal 000011 conserva los contratos,
el servicio de preparación/conciliación/publicación y el repositorio PostgreSQL.
`vec-revisar-organizacion` revisa el paquete en memoria; `vec-organizacion-semilla`
genera la semilla del catálogo preparatorio de Contratación, sin publicar historia
Personal. La política de acreditación de fuentes sigue siendo un puerto: este
corte no aporta una autoridad institucional ni verifica la instalación SQL.
La corrección de decisiones de clase ajena de [#585](https://github.com/aavidad/VEC_Diputacion_app/pull/585)
ya está integrada en esta base y se conserva.

Se amplía el [revisor existente](../../cmd/vec-revisar-organizacion/README.md)
con `--preparar`: además del informe entrega el paquete válido en el orden de
su huella, reutilizable por el mismo lector. El rechazo conserva el informe y
omite el paquete. Se reutilizan reglas, límites y pendientes de publicación;
no se fabrica actor ni se ejecuta otro importador.

Este corte aporta preparación local a la tarea 4. Para publicar y montar la
consulta histórica siguen pendientes la fuente admitida, diccionario, actos,
custodia, catálogos y concesiones nominales, con aprobación separada y consumo
transaccional de autorización/auditoría. No cierra las tareas de plantilla,
ocupación, reserva o vacantes; la ausencia de una fuente mantiene incertidumbre.


Comprobación local: pruebas normales y de carrera, y `go vet`, verdes en
`cmd/vec-revisar-organizacion` y `personal/application`. El ejemplo se exportó
con `--preparar`, se extrajo y se volvió a leer con el mismo comando: conservó
la huella `900a156e5ea86103a52ff065a9b56d716553257f2f6c6fae02e2b62b525815b6`.
Semgrep con cuatro reglas locales y gosec focal no comunicaron hallazgos.
La revisión independiente y la integración corresponden a Dirección.


## Contratos de gobierno para la fuente común — 7 de octubre de 2026

Se recuperan cinco archivos Go de `a9001c097`: dominio, puertos y aplicación
de gobierno de categorías. El consumidor inmediato es la fuente nominal
común que prepara V; no se importa el handler, PostgreSQL ni Cat4/AD134
como parte de este corte. Las rutas y acciones de escritura siguen cerradas.

La aplicación exige actor, vínculo y contexto iguales, garantía High y
versión de rol esperada. La revisión corrigió dos defectos: los errores
al aprobar y confirmar se normalizan igual que al proponer, y se comprueban
contexto y tamaños antes de copiar el mapa o calcular la huella documental.
Pruebas focales normales y de carrera, gopls y diff correctos; dos revisiones
independientes favorables del código `0772604ff`.

V conserva identidad, sesión, perfil fijo, instantánea y emisor comunes.
T conserva el adaptador RPT, handler y composición. La separación efectiva
editor–aprobador, CAS, recibo durable y auditoría transaccional necesitan
la conciliación SQL posterior; estos tipos no los acreditan por sí solos.
AUT25, CA21 e IS10 se preservan hasta su decisión, sin duplicar fuentes ni
activar una garantía sintética como producción.
