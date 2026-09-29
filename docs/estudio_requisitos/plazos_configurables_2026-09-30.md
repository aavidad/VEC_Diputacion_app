# Plazos configurables desde la aplicación

Fecha: 30/09/2026. Origen: orden de Alberto del 29/09/2026.

> Todas las dudas de tiempos y demás debemos poder configurarlas en la app, en
> vez de esperar a RRHH. Como además pueden cambiar, conviene que todas las
> dudas de fechas se pongan como campos desde configuración para modificarlo.
> Todo tiene que quedar registrado: quién hizo el cambio.

Este estudio tiene tres partes: el inventario de plazos que hoy son dudas o
constantes, el diseño de la configuración y el plan de cortes. Empieza por
Contratación temporal, por los plazos que usa la lista de expedientes.

## 1. Qué hay hoy

### 1.1 De dónde salen los plazos

Los plazos de Contratación temporal y de Bolsa ya no están escritos en el
código. Salen de dos **catálogos de reglas** versionados:

- `vec.contratacion_temporal.reglas` (32 reglas), fichero
  `data/demo/reglas/ct_reglas.ejemplo.demo.json`;
- `vec.bolsa.reglas` (45 reglas), fichero
  `data/demo/reglas/bolsa_reglas.ejemplo.demo.json`.

El paquete común `internal/vec/reglas` los lee con un `Resolutor`: elige la
versión vigente, valida cada regla (unidad, cantidad, cómputo, origen, norma,
duda) y calcula vencimientos con Calendarios (días hábiles con festivos, o
cómputo civil de fecha a fecha). Cada regla lleva su referencia exacta
`catálogo:versión:entrada` y la huella SHA-256 del catálogo.

Hoy esos catálogos **solo se pueden cambiar editando el fichero y
rearrancando**. La pantalla «Reglas vigentes» (PR #162, abierta) solo los
muestra.

Hay una excepción que ya se edita por pantalla: la **política de ofertas de
Bolsa** (`vec_bolsa_llamamientos.politica_ofertas_version`, migración B47).
Cada guardado crea una versión nueva inmutable, consume una autorización V3 en
la misma transacción y deja recibo, auditoría y outbox. Las ofertas ya
publicadas conservan la versión que las gobernó. Es el modelo a seguir.

### 1.2 Inventario de Contratación temporal

| Regla | Valor de ejemplo | Origen | Quién la usa | Duda |
|---|---|---|---|---|
| `c01.plazo_analisis` | 5 días hábiles desde la solicitud | ejemplo | **Nadie.** No declara `fases`, así que la lista no la aplica | Pendiente de RRHH |
| `c02.plazo_informes` | 10 días hábiles | ejemplo (Ley 39/2015, art. 80.2) | Lista de expedientes, fases `asignacion_unidad` e `informe_juridico` | Si se aplica y a qué fases |
| `c03.plazo_fiscalizacion` | 10 días hábiles; 5 si es urgente | ejemplo | Lista de expedientes, fase `fiscalizacion` | Cotejar con Intervención (duda 63) |
| `c04.plazo_subsanacion` | 10 días hábiles | ejemplo | Lista de expedientes, fase `subsanacion_unidad` | Pendiente de RRHH |
| `c07.jornada_completa` | 2.250 minutos semanales (37 h 30 min) | ejemplo | Análisis: fracción de jornada | Duda 38 |
| `c08.acumulacion_tareas` | 9 meses en 18 | ejemplo | Análisis: aviso al superar la duración | Duración por modalidad |
| `c08.programas_temporales` | 3 años (+12 meses) | ejemplo | Análisis | Íd. |
| `c08.vacante` | 3 años | ejemplo | Análisis | Íd. |
| `c08.circunstancias_produccion` | 6 meses (+12) | ejemplo | Análisis | Íd. |

El cálculo de la lista está en `internal/app/bootstrap/plazos_fase_ct.go`: busca
la regla cuyo atributo `fases` contiene la fase actual, calcula el vencimiento
desde la fecha de entrada en la fase (`FasesDesde`, CT-000110) y devuelve
«en plazo», «vence hoy», «vencido» o «no calculado». La urgencia sale de
CT-000125.

**Problema de fondo:** el cálculo usa siempre las reglas vigentes **ahora**. Si
mañana c03 pasa de 10 a 7 días, todos los expedientes que ya están en
fiscalización cambian su fecha de vencimiento de golpe, y algunos pasarían a
«vencido» sin que nadie haya hecho nada. Esto hay que corregirlo antes de
permitir cambios (apartado 2.5).

### 1.3 Inventario de Bolsa

| Regla | Valor | Origen | Quién la usa |
|---|---|---|---|
| `b02.intentos_contacto` / `b02.separacion_intentos` | 2 intentos, 2 horas | reglamento | Registro de intentos de contacto |
| `b03.procesos_sin_contacto` | 2 procesos | reglamento | Íd. |
| `b04.franja_llamadas` | 09:00–14:00, días hábiles | ejemplo | Íd. |
| `b05.plazo_respuesta` | 1 día hábil desde el contacto efectivo | ejemplo | Llamamiento de Contratación temporal, portal del candidato, asistente de llamamiento de Bolsa |
| `b10.plazo_publicacion` | 2 días hábiles | reglamento | Ofertas publicadas de Bolsa |
| `b11.acreditar_renuncia_justificada` | 10 días hábiles | reglamento | Portal del candidato |
| `b11.periodo_matrimonio` | 15 días naturales | reglamento | **Nadie** |
| `b14.reposicion_general` | 5 meses | reglamento | Situaciones de Bolsa |
| `b14.reposicion_acumulacion_tareas` | 9 meses | reglamento | Solo por el mapa de modalidades |
| `b17.aviso_encadenamiento` | 18 meses en 24 | ejemplo | Avisos de RRHH (copia en PostgreSQL, B41) |
| `b18.pausa_voluntaria` | 12 meses | ejemplo | Portal del candidato |
| `b19.vacante_duracion_maxima` | 3 años, aviso 30 días antes | reglamento | Avisos de RRHH (copia en PostgreSQL, B41) |
| `b20.sae_duracion_maxima` | 9 meses | reglamento | Avisos de vía de cobertura (Contratación) |
| `b21.plazo_documentacion` | 3 días hábiles desde la aceptación | ejemplo | Documentación de la formalización (Contratación) |
| `b23.plazo_incorporacion` | 1 día hábil desde la aceptación | reglamento | Íd. |
| `b24.consecuencias` / `b24.sancion.suspension` | 1 mes / 6 meses | reglamento / ejemplo | Sanciones |
| `b25.vigencia_bolsa` | 5 años, aviso 6 meses antes | reglamento | Avisos de vía de cobertura |
| `b29.contacto_origen_convoca` | 12 meses | ejemplo | Contacto de aspirantes importados |
| `b30.plazas_plazo_respuesta` | 24 horas | ejemplo | **Nadie** (ver duplicidades) |
| Política de ofertas (PostgreSQL, por bolsa) | 48 horas naturales | ejemplo | Ofertas y plazas de oferta (B47, B54, B58) |

### 1.4 Duplicidades que hay que resolver

Hay dos autoridades para el mismo plazo, y dan valores distintos:

1. **Plazo de respuesta al llamamiento.** `b05.plazo_respuesta` dice 1 día
   hábil; la política de ofertas dice 48 horas naturales; `dudas.md` (dudas 1
   y 43) dice 48 horas. El llamamiento de Contratación y el portal del
   candidato usan b05; las ofertas de Bolsa usan la política.
2. **Plazo de respuesta por plaza.** `b30.plazas_plazo_respuesta` (24 h) no lo
   lee nadie; lo que se ejecuta es `politica.plazas.respuesta_horas`.

Propuesta: la política de ofertas es la autoridad de las ofertas publicadas
(ya es editable y versionada) y b05 lo es del llamamiento directo. Se retira
b30 del catálogo (o se marca como «valor inicial de la política») y se deja
escrito en la ayuda que son dos plazos distintos. Lo decide Alberto antes del
corte de Bolsa; no bloquea el de Contratación.

### 1.5 Valores fijos en el código

| Dónde | Qué | Tratamiento |
|---|---|---|
| `web/static/portal-empleado/modulos/bolsa/rrhh-plazos-ui.js`, líneas 11 y 222 | 48 horas al crear la política; 48 o 1 al cambiar de unidad | Es una decisión inventada. Debe salir del catálogo (valor inicial de la política) y llegar en la respuesta de la API |
| `bolsa/domain/politica_ofertas.go` y B47/B54 | Límites 1–30 días, 1–720 horas | Límites técnicos de seguridad. Se quedan, documentados |
| `bolsa/domain/intentos_contacto.go` | Separación máxima de 7 días | Límite técnico |
| `bolsa_llamamientos/000028` | Vencimiento de oferta como mucho a 120 días | Límite técnico |
| `bolsa_llamamientos/000020` | «3 years» y «30 days» en los avisos | Ya sustituido por los parámetros de B41, que salen de b19 |
| CT: `MaximoAniosPeriodoAlta`, `maximoAniosPeriodoAnalisis` | Tope de años de un periodo | Límite técnico de validación |
| CT: estadísticas | Ventana por defecto de 11 semanas | Presentación, no es un plazo |
| `firma-autofirma.js` | 5 minutos para completar la firma | Técnico |

Un límite técnico no es un plazo administrativo: impide valores absurdos (un
plazo de 10.000 días) y protege la base. Si RRHH necesita un plazo que lo
supere, se sube el límite con una PR; no se hace configurable.

### 1.6 Otros módulos (fuera de esta fase)

Cronos (permisos, duda 41), Dietas (duda 23), conservación documental (dudas
60 y 61) y jornada (duda 38) también tienen plazos pendientes. Se tratarán con
el mismo mecanismo cuando llegue su turno, por el orden de la hoja de ruta.

## 2. Diseño

### 2.1 Una sola autoridad: el catálogo de reglas, con una capa de ajustes

No se crea otra fuente de plazos. La autoridad sigue siendo el catálogo de
reglas y el mismo `Resolutor`. Lo que se añade es una **capa de ajustes**
guardada en PostgreSQL:

- el **catálogo base** es el fichero de hoy (en producción, el catálogo que
  apruebe RRHH). No cambia desde la aplicación;
- los **ajustes** son los valores que RRHH fija desde la pantalla. Forman su
  propio catálogo versionado, `vec.contratacion_temporal.reglas.ajustes`, con
  versiones 1, 2, 3… en PostgreSQL, de solo adición;
- el `Resolutor` devuelve la regla base con el ajuste aplicado encima.

Por qué una capa y no copias completas del catálogo en la base: el catálogo
base seguirá cambiando (se añaden reglas nuevas con cada corte). Con copias
completas, un catálogo base nuevo borraría lo que RRHH hubiera cambiado, o
habría que fusionarlo a mano. Con la capa, el ajuste de RRHH se mantiene
aunque cambie la base, y la pantalla avisa si la regla base cambió después
del ajuste («la regla de partida ha cambiado; revise su valor»).

Referencia exacta: una regla ajustada cita el ajuste
(`vec.contratacion_temporal.reglas.ajustes:<n>:c03.plazo_fiscalizacion`, con la
huella de esa versión de ajustes). Una regla sin ajustar sigue citando la
base. Así cualquier hecho guardado dice qué valor lo gobernó.

### 2.2 Qué se puede cambiar

Cada regla del catálogo base declara qué campos admiten ajuste y con qué
opciones. Son atributos de datos, no código:

```json
"editable": "cantidad,cantidad_urgente,unidad",
"opciones_unidad": "dias_habiles,dias_naturales",
"cantidad_minima": "1",
"cantidad_maxima": "60"
```

Reglas:

- solo son ajustables las reglas con `editable`. En el primer corte, c01–c04
  (cantidad, cantidad urgente de c03 y unidad días hábiles o naturales);
- el cómputo (administrativo o civil) y la unidad deben seguir siendo
  coherentes: días hábiles solo con cómputo administrativo. El `Resolutor`
  valida la regla ajustada con las mismas comprobaciones que la base; un
  ajuste que produzca una regla no válida se rechaza al guardar;
- **no se ajustan en esta fase** los atributos estructurales: `fases`,
  `inicio`, listas, catálogos enlazados ni `c23.fase_operacion.*`. Cambiar
  `fases` altera qué perfil fijo de RRHH se necesita (PR #154) y exige la
  provisión aprobada por el operador. Se verá aparte;
- una regla de origen **reglamento** no se ajusta desde la pantalla: su valor
  lo fija una norma publicada. Si la norma cambia, se cambia el catálogo base
  citando el nuevo BOP. Sí se ajusta su parte de ejemplo cuando la tenga.

Tras el ajuste, la regla pasa a origen **`configuracion`**: «valor fijado en
VEC por RRHH». La pantalla lo muestra con quién, cuándo y por qué. En un
entorno de ejemplo (paquete de demostración), la regla sigue rotulada como de
ejemplo: quien la ajusta es una identidad sintética y no hay aprobación real
de RRHH detrás.

La duda de la regla no se borra: queda como antecedente («Duda: …; resuelta
en VEC el 30/09/2026»). La duda de `dudas.md` se quita cuando RRHH la
confirme, no cuando alguien ajusta el valor en desarrollo.

### 2.3 Qué queda registrado

Tabla `vec_contratacion_temporal.regla_ajuste_version`, de solo adición (un
disparador rechaza UPDATE y DELETE):

| Campo | Contenido |
|---|---|
| `catalogo_id`, `version` | Catálogo de ajustes y versión (1, 2, 3…) |
| `ajustes` | Todos los ajustes vigentes en esa versión (no solo el cambio), en JSON canónico |
| `huella_sha256` | Huella de `ajustes` |
| `base_version`, `base_huella_sha256` | Catálogo base que vio quien guardó |
| `version_esperada` | Control optimista: debe ser la versión anterior |
| `actor_ref` | Persona que hizo el cambio (referencia opaca `per_…`) |
| `motivo` | Texto obligatorio, 10–500 caracteres, sin datos personales |
| `vigente_desde` | Instante de publicación (en el primer corte, inmediato) |
| `clave_idempotencia` | Repetir la misma petición no crea otra versión |
| `recibo_ref`, `decision_ref`, `auditoria_ref` | Recibo y consumo de la autorización V3 |

Tabla `regla_ajuste_cambio`: una fila por campo cambiado, con clave de la
regla, campo, **valor anterior** y **valor nuevo**. El valor anterior de un
campo sin ajuste previo es el de la base; lo aporta la aplicación y la base lo
guarda junto a `base_huella_sha256` para poder comprobarlo después.

Tabla `regla_ajuste_outbox`: un evento por versión, para que otros
consumidores (por ejemplo, la copia de parámetros de avisos de Bolsa, B41) se
enteren sin leer tablas ajenas.

Todo en una transacción: autorización consumida, versión, cambios, recibo,
auditoría y outbox. Si falla una parte, no se guarda nada.

### 2.4 Quién puede cambiarlo

- Un **perfil fijo de administración de reglas** de la persona de RRHH, con el
  mismo mecanismo de los perfiles fijos de los cortes 2 y 3 (PR #141, #154,
  #161): la asignación se publica una sola vez al arrancar y después solo se
  consume. **No se publica un permiso por petición.**
- Acción `contratacion_temporal.reglas.ajustar`, recurso
  `catalogo_reglas` = `vec.contratacion_temporal.reglas`, finalidad
  `gobierno_reglas_contratacion_temporal`.
- La función SQL consume la autorización (consumidor AD3 nuevo) y comprueba
  acción, recurso, finalidad y actor, como B47.
- Leer las reglas y su historial sigue siendo lectura de RRHH (la frontera
  común de lectura que ya usa «Reglas vigentes»).
- La respuesta de lectura dice si quien consulta puede ajustar (`ajustable`),
  para que la pantalla no ofrezca un botón que después dará 403. El servidor
  vuelve a comprobarlo al guardar.

### 2.5 Los plazos que ya están corriendo

**Decisión propuesta: el plazo que ya corre se queda con el valor con el que
empezó.** Un cambio se aplica a las fases que empiecen después de guardarlo.

Cómo se hace sin guardar nada por expediente: las versiones de ajustes tienen
`vigente_desde` y no se borran nunca. Para cada expediente, el cálculo pide las
reglas vigentes **en el instante en que entró en la fase** (`FasesDesde`), no
las de ahora. El `Resolutor` gana un método `ReglasEn(ctx, instante)`.

Por qué:

- seguridad jurídica: el plazo que se comunicó o empezó a correr no cambia a
  mitad (principio *tempus regit actum*);
- evita que un cambio convierta de golpe decenas de expedientes en «vencido»;
- es lo mismo que ya hace la política de ofertas de Bolsa: cada oferta guarda
  la versión que la gobernó.

Si RRHH quiere dar más tiempo a un expediente concreto que ya está en plazo,
eso es una **ampliación de plazo** (Ley 39/2015, art. 32: antes de que venza y
como mucho la mitad del plazo), un acto sobre ese expediente con su propio
motivo. No se hace cambiando el catálogo. Queda fuera de esta fase.

La lista de expedientes lo explica: en el detalle del plazo, «Calculado con la
regla vigente cuando el expediente entró en esta fase (ajuste 2, 30/09/2026)».

Límite técnico que hay que resolver: hoy el `Resolutor` lee como mucho 16
versiones de un catálogo. Con ajustes de solo adición, la consulta de ajustes
pide **una** versión, la vigente en el instante pedido
(`WHERE vigente_desde <= instante ORDER BY version DESC LIMIT 1`), y no la
lista entera.

### 2.6 Pantalla

Va dentro de «Reglas vigentes» (PR #162), no en otra pantalla:

- cada regla ajustable muestra su valor y un botón **Cambiar** (solo si
  `ajustable`);
- el formulario muestra los campos permitidos: número con su mínimo y máximo,
  y desplegables con las opciones de la regla (textos por clave i18n, nunca en
  el código);
- motivo obligatorio;
- antes de guardar, un resumen: «Plazo de fiscalización: 10 días hábiles →
  7 días hábiles. Se aplicará a los expedientes que entren en fiscalización
  desde ahora. Los que ya están en fiscalización mantienen su plazo.»;
- al guardar, recibo con versión, fecha y hora;
- **Historial** de cada regla: fecha, persona, antes → después y motivo;
- si otra persona guardó antes (versión desfasada), se avisa y se recarga sin
  perder lo escrito.

La ayuda va en el botón «?» de la pantalla, no en el texto.

### 2.7 Contrato

- Lectura: `GET /api/vec/reglas/vigentes` pasa a `vec.reglas.vigentes.v2`:
  añade el origen `configuracion`, los campos `ajuste` (versión, fecha,
  persona, motivo), `editable` con opciones y límites, y `ajustable` por
  catálogo. La v1 de la PR #162 exige `duda` no vacía y origen `reglamento` o
  `ejemplo`; la v2 lo amplía sin romperla.
- Historial: `GET /api/contratacion-temporal/rrhh/reglas/ajustes/historial`
  (paginado, 50 por página).
- Guardar: `POST /api/contratacion-temporal/rrhh/reglas/ajustes` con
  `version_esperada`, `clave_idempotencia`, `motivo` y la lista de cambios.
  Respuestas: 201 (versión nueva), 200 (repetición idéntica), 409 (versión
  desfasada o clave reutilizada con otro contenido), 422 (valor fuera de las
  opciones o regla no válida), 403 (sin perfil), 503 (base no disponible;
  nunca se da por guardado).

### 2.8 Producción

- El catálogo base de producción lo aprueba RRHH; el paquete de ejemplo se
  retira. Los ajustes hechos en desarrollo no se llevan a producción: la tabla
  de ajustes de la principal empieza vacía.
- Antes de pasar a producción, ninguna regla que se use puede quedar con
  origen `ejemplo`.

## 3. Cortes

Cada corte es una PR pequeña que compila, se prueba y se revisa sola.

1. **Go, sin SQL: el plazo en curso se queda.** `Resolutor.ReglasEn`; la
   calculadora de la lista usa `FasesDesde`; contrato `editable` y origen
   `configuracion` en `internal/vec/reglas`; una sola consulta del catálogo de
   Contratación para todos los resolutores (hoy se crean cuatro a partir del
   mismo fichero). Pruebas del cambio de valor con expedientes en curso.
2. **SQL de Contratación (CT-000148 y AD3-000114).** Tablas de ajustes,
   cambios y outbox; función de publicación con consumo V3; función de lectura
   de la versión vigente en un instante. Ensayo en el clon de la principal y
   revisión con `revisor-sql-vec`.
3. **Go: guardar y leer ajustes.** Adaptador PostgreSQL, caso de uso, ruta
   HTTP, perfil fijo de administración de reglas, composición. Revisión de
   seguridad focal y Semgrep.
4. **Pantalla.** Edición e historial en «Reglas vigentes» (tras fusionar la
   PR #162). `usabilidad-vec`, `aspecto-vec`, `impeccable` y revisión con
   `revisor-usabilidad-vec`.
5. **Bolsa.** Mismo mecanismo en `vec_bolsa_llamamientos`; resolver las dos
   duplicidades del apartado 1.4; el valor inicial de la política de ofertas
   sale del catálogo (quitar las 48 h de `rrhh-plazos-ui.js`); admitir el
   catálogo de ajustes en `politica_llamamiento_admitida` (CT111), que hoy
   solo admite entradas de `vec.bolsa.reglas`.

## 4. Preguntas abiertas para Alberto

1. ¿Se aplica un plazo de análisis a la fase «solicitud» (c01)? Hoy existe la
   regla pero no se usa. Activarla hace que la lista muestre plazo también en
   solicitudes nuevas.
2. Plazo de respuesta al llamamiento: ¿se confirma que b05 (llamamiento
   directo) y la política de ofertas (ofertas publicadas) son dos plazos
   distintos?
3. ¿Hace falta programar un cambio para una fecha futura («a partir del 1 de
   enero»)? El diseño lo admite (`vigente_desde`), pero el primer corte lo
   aplica al guardar.
