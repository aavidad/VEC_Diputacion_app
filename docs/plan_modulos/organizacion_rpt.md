# Organización y RPT: inventario y trabajo siguiente

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
| RPT-005 | Circuito completo persona–relación–plaza–puesto, titularidad, fechas, actos y reserva. Propietario: Personal B. |
| RPT-006 | Proyecciones distintas de dotación vacante, puesto sin ocupante y necesidad cubrible, con cobertura e incertidumbre. Coordinar con B. |

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

### 5. Completar ocupación, reserva y vacantes con Personal B

- Propietario: B; M aporta referencias/versiones estructurales por puertos.
- Archivos B: `registro_empleado_b2*`, montaje B2 y `registro-b2*`; M no los edita.
- Dependencias: relaciones y actos acreditados, vínculos estructurales exactos
  y cobertura de las fuentes; nunca deducir ocupación desde nómina o certificado.
- SQL/servidor: revisión y ensayo por B/D sobre su propia candidata; sin tablas cruzadas.
- Cierre: historia privada preservada y las tres proyecciones separadas,
  sin convertir ausencia de datos en plaza vacante ni en necesidad cubrible.

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
| `trabajo/codexm-rpt-informe-20261001@aaa8fa072` | [PR #305](https://github.com/aavidad/VEC_Diputacion_app/pull/305), borrador. Incluye núcleo `ae87d7cb8` y CLI/informe `cmd/vec-comparar-organizacion`; GO independiente, focales y Chrome verdes. Calidad global y CI en curso; no publicado en main. |
| `origin/trabajo/codexm-rpt-escritura-retoma-20260930@a9001c097` | Gobierno de categorías pendiente B→A, dependencias nominales y ensayo final; preservar. |
| [#249 IS10](https://github.com/aavidad/VEC_Diputacion_app/pull/249), [#256 CA21](https://github.com/aavidad/VEC_Diputacion_app/pull/256) | Abiertas; postimágenes de D, no capacidades instaladas. |
| [#290](https://github.com/aavidad/VEC_Diputacion_app/pull/290) `d957f0ec9` | Borrador abierto, CI 5/5; preparación reproducible de clon H6 hasta SQL62. No manual ni instalación. |
| `trabajo/codexm-clon-preimagenes-nominales-20261001@e2c5b0388` | NO-GO preservado deliberadamente como evidencia; no borrar, instalar ni usar como base aprobada. |

D prepara la intervención del servidor mediante copia fría; ese trabajo no
es el clon H6b. AD132 queda aparte. Este plan no acredita instalación nueva.
Solo Dirección integra en main; se conservan las ramas pendientes/rechazadas.

## Consenso Astra

Astra dio GO arquitectónico al reparto y orden del borrador `eb7f7739`.
Se incorporaron sus precisiones: precondiciones antes del montaje; publicación
mínima de fuente antes de consulta si falta; IS10 explícita; ruta correcta de
aplicación y tareas independientes de categorías separadas del gobierno pendiente.

Acuerdos: cerrar primero la búsqueda conservada; reutilizar historia e
importación con concesión V3 propia; Personal B conserva ocupaciones y reservas;
M aporta estructura por puertos. El comparador sintético no acredita vacantes
ni historia corporativa. Las candidatas SQL y la evidencia NO-GO se preservan.
No se abre la PR del plan hasta recibir el GO de Dirección sobre su SHA exacto.
