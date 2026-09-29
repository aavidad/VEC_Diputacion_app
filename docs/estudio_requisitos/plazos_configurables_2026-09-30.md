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

**Problema de fondo:** hasta este estudio, el cálculo usaba las reglas
vigentes **ahora**. Si mañana c03 pasa de 10 a 7 días, todos los expedientes
que ya están en fiscalización cambiarían su vencimiento de golpe, y algunos
pasarían a «vencido» sin que nadie haya hecho nada. El corte 1 lo corrige
(apartado 2.5).

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

### 1.7 Otras cosas que hay que saber

- **Respaldo fijo de jornada.** Si falta `c07.jornada_completa`, el análisis
  usa 37 h 30 min escritos en el código
  (`contratacion_temporal_reglas_analisis_desarrollo.go`). Debe fallar cerrado
  o salir del catálogo.
- **Valores leídos solo al arrancar.** Las duraciones de c08 (opciones del
  análisis) y los parámetros de avisos de Bolsa (B41) se leen una vez al
  arrancar. Un ajuste de esas reglas no se aplicaría hasta reiniciar. Por eso
  no son ajustables en el primer corte.
- **Cinco resolutores del mismo fichero.** El catálogo de Contratación se abre
  cinco veces: `reglas_ejemplo.go`, `contratacion_temporal_reglas_analisis`,
  `_cancelacion`, `_seguimiento_cese` e `_incorporacion_centro`. Antes de
  componer los ajustes hay que dejar uno solo, compartido.
- **Campos que el primer corte no cubre.** La franja de llamadas (b04), la
  ventana de encadenamiento (b17) y los días de aviso (b19, b25) son atributos
  y no la cantidad principal. Se añadirán como campos ajustables cuando se
  haga Bolsa.
- **Quién guarda el vencimiento y quién lo recalcula.** El llamamiento de
  Contratación (CT111) guarda el plazo con su referencia: un ajuste posterior
  no le afecta. Recalculan al leer la lista de expedientes, las sanciones, las
  situaciones de Bolsa, el portal del candidato, la documentación de la
  formalización y los avisos de vía de cobertura. Para todos ellos rige lo del
  apartado 2.5: el cálculo usa los ajustes vigentes en el inicio del plazo.

## 2. Diseño

### 2.1 Una sola autoridad: el catálogo de reglas, con una capa de ajustes

No se crea otra fuente de plazos. La autoridad sigue siendo el catálogo de
reglas y el mismo `Resolutor`. Lo que se añade es una **capa de ajustes**
guardada en PostgreSQL:

- el **catálogo base** es el fichero de hoy (en producción, el catálogo que
  apruebe RRHH). No cambia desde la aplicación y declara qué se puede ajustar;
- los **ajustes** son los valores que RRHH fija desde la pantalla. Forman su
  propio catálogo versionado, `vec.contratacion_temporal.reglas.ajustes`, con
  versiones 1, 2, 3… en PostgreSQL, de solo adición;
- el `Resolutor` devuelve la regla base con el ajuste aplicado encima. Nadie
  más lee la tabla de ajustes.

Por qué una capa y no copias completas del catálogo en la base: el catálogo
base seguirá cambiando (se añaden reglas con cada corte). Con copias
completas, un catálogo base nuevo borraría lo que RRHH hubiera cambiado, o
habría que fusionarlo a mano. Con la capa, el ajuste de RRHH se mantiene
aunque cambie la base.

Referencia exacta: una regla ajustada cita el ajuste
(`vec.contratacion_temporal.reglas.ajustes:<n>:c03.plazo_fiscalizacion`). Su
huella combina la huella de la base y la de esa versión de ajustes, de modo
que identifica exactamente el valor que la gobernó. La regla lleva además la
referencia de la entrada base (`BaseReferencia`): con solo la del ajuste no se
podría reproducir la regla si la base cambió después. Una regla sin ajustar
sigue citando la base.

**Esquema por módulo.** Los ajustes de Contratación viven en
`vec_contratacion_temporal`, y los de Bolsa vivirán en
`vec_bolsa_llamamientos`. Se descarta un esquema común `vec_reglas` por tres
motivos: los catálogos son de cada módulo (`modulo_id`); la guarda de sesión
del núcleo AD3 y los roles ejecutores son por módulo, y un esquema común
necesitaría una familia de roles nueva con acceso desde varios módulos; y es
lo que ya hace la política de ofertas (B47). La duplicación queda en el SQL:
el `Resolutor`, el adaptador y el caso de uso de Go son comunes.

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

- solo son ajustables las reglas con `editable`. En el primer corte, c01–c04:
  cantidad, cantidad urgente de c03 y unidad (días hábiles o naturales);
- el `Resolutor` valida la regla ajustada con las mismas comprobaciones que la
  base (unidad, cómputo, cantidad urgente no mayor que la ordinaria) y dentro
  de las opciones y límites que ella misma declara;
- **no se ajustan** los atributos estructurales: `fases`, `inicio`, listas,
  catálogos enlazados ni `c23.fase_operacion.*`. Cambiar `fases` altera qué
  perfil fijo de RRHH se necesita (PR #154) y exige la provisión aprobada por
  el operador;
- una regla de origen **reglamento**, o con parte de ejemplo, no se ajusta
  desde la pantalla: su valor lo fija una norma publicada. Si la norma cambia,
  se cambia el catálogo base citando el nuevo BOP.

El **origen** de la regla (reglamento o ejemplo) es su procedencia normativa y
no cambia con un ajuste. El ajuste va en un campo aparte (`Ajuste`: versión,
desde cuándo rige y campos ajustados). Una regla de ejemplo ajustada deja de
rotularse como «regla de ejemplo» salvo que todo el paquete sea de
demostración: en desarrollo quien ajusta es una identidad sintética y no hay
aprobación real de RRHH detrás.

La duda de la regla no se borra: queda como antecedente. La duda de `dudas.md`
se quita cuando RRHH la confirme, no cuando alguien ajusta el valor en
desarrollo.

**Si la base cambia y el ajuste deja de encajar** (por ejemplo, la base baja
`cantidad_maxima` o quita `editable`), solo esa regla queda fuera de uso: su
plazo sale como «no calculado» y la pantalla pide revisar el ajuste. El resto
del catálogo y el arranque siguen funcionando. Nunca se vuelve en silencio al
valor base. El ensayo sobre el clon de la principal comprueba los ajustes
guardados contra la base nueva antes de desplegar.

### 2.3 Qué queda registrado

Tabla `vec_contratacion_temporal.regla_ajuste_version`, de solo adición (un
disparador rechaza UPDATE y DELETE):

| Campo | Contenido |
|---|---|
| `catalogo_id`, `version` | Catálogo de ajustes y versión (1, 2, 3…) |
| `ajustes_canonico` | Todos los ajustes vigentes en esa versión (no solo el cambio), en la forma canónica que calcula Go; como mucho 64 reglas y 16 KiB |
| `huella_sha256` | SHA-256 de `ajustes_canonico`, comprobada por la base |
| `base_version`, `base_huella_sha256` | Catálogo base que vio quien guardó |
| `version_esperada` | Control optimista: debe ser la versión anterior |
| `actor_ref` | Persona que hizo el cambio (referencia opaca) |
| `motivo_clave` | Motivo de un catálogo cerrado: respuesta de RRHH a una duda, acuerdo o instrucción, cambio normativo, corrección de un error |
| `referencia` | Opcional, hasta 120 caracteres: número de acuerdo, BOP o duda |
| `nota` | Opcional, hasta 500 caracteres |
| `vigente_desde` | `clock_timestamp()` al publicar; nunca anterior al de la versión previa |
| `clave_idempotencia` | Repetir la misma petición no crea otra versión |
| `recibo_ref`, `decision_ref`, `auditoria_ref` | Recibo y consumo de la autorización V3 |

Tabla `regla_ajuste_cambio`: una fila por campo cambiado, con clave de la
regla, campo, **valor anterior** y **valor nuevo**. El valor anterior de un
campo sin ajuste previo es el de la base: lo aporta la aplicación y la base lo
guarda junto a `base_huella_sha256`. Solo Go puede validar un ajuste contra la
base, porque la base no está en PostgreSQL; la base sí comprueba forma,
límites, encadenamiento de versiones y huella.

Tabla `regla_ajuste_outbox`: un evento por versión, con catálogo, versión y
huella, para que otros consumidores se enteren sin leer tablas ajenas.

Todo en una transacción: autorización consumida, versión, cambios, recibo,
auditoría y outbox. Si falla una parte, no se guarda nada.

**Datos personales.** El motivo es una clave de catálogo, no texto libre. La
referencia y la nota son libres y la pantalla avisa de que no deben llevar
datos de personas; solo se muestran en el historial. Ni el motivo, ni la nota,
ni la persona viajan con la regla a sus consumidores, al outbox ni a la
lectura común «Reglas vigentes». El historial muestra el nombre de quien
cambió el valor mediante una consulta de identidad gobernada; mientras esa
consulta no esté compuesta, dice «persona de RRHH» y nunca enseña la
referencia opaca.

### 2.4 Quién puede cambiarlo

- Un **perfil fijo de administración de reglas** de la persona de RRHH, con el
  mismo mecanismo de los perfiles fijos de los cortes 2 y 3 (PR #141, #154,
  #161): la asignación se publica una sola vez al arrancar y después solo se
  consume. **No se publica un permiso por petición.**
- Acción `contratacion_temporal.reglas.ajustar`, tipo de recurso
  `catalogo_reglas`, recurso `vec.contratacion_temporal.reglas`, finalidad
  `gobierno_reglas_contratacion_temporal`, sin campos ni obligaciones.
- La decisión queda ligada al contenido exacto:
  `contexto_recurso_huella_sha256` es la huella del efecto, SHA-256 de
  (catálogo, `version_esperada`, `base_huella`, ajustes canónicos, motivo,
  referencia y nota). La función SQL recalcula esa huella y la compara.
- Como en B47, la autorización se consume antes de mirar la idempotencia: cada
  repetición necesita una decisión nueva, que emite el caso de uso.
- Consumidor AD3 nuevo (AD3-114), con la guarda de sesión del ejecutor de
  Contratación. Parchea el núcleo por preimagen de texto, así que se ensaya en
  el clon después de las AD3-110 a 113 que siguen abiertas.
- La lectura común `GET /api/vec/reglas/vigentes` no cambia de contrato: es
  neutral respecto al módulo y no deduce identidad. Lo que es de la edición
  (qué se puede ajustar, con qué opciones, si quien consulta puede hacerlo, e
  historial) va en una ruta propia de Contratación con la misma frontera y el
  mismo perfil fijo que el guardado.

### 2.5 Los plazos que ya están corriendo

**Decisión propuesta: el plazo que ya corre se queda con el valor con que
empezó.** Un ajuste se aplica a los plazos que empiecen después de guardarlo.

Cómo se hace sin guardar nada por expediente:

- el catálogo base se resuelve siempre con la hora actual (el fichero no tiene
  historia en la aplicación);
- los ajustes, que no se borran nunca y tienen `vigente_desde`, se resuelven
  en el **instante en que empezó el plazo**. `Vencimiento` y
  `VencimientoUrgente` ya lo hacen por defecto con su `inicio`, así que todos
  los consumidores que calculan un plazo cumplen la regla sin tocarlos. Para
  la lista de expedientes, el inicio es la entrada en la fase (`FasesDesde`,
  CT-000110).

Consecuencia que hay que tener clara: **un cambio del catálogo base sí afecta
a los plazos en curso**. Lo que se congela son los ajustes de RRHH. En
producción el catálogo base solo cambia con una PR y un despliegue.

Por qué:

- seguridad jurídica: el plazo que se comunicó o empezó a correr no cambia a
  mitad (*tempus regit actum*);
- evita que un cambio convierta de golpe decenas de expedientes en «vencido»;
- es lo que ya hace la política de ofertas de Bolsa.

Casos límite:

- **Vuelta a una fase** (por ejemplo, de subsanación a fiscalización): CT-000110
  abre un tramo nuevo; el plazo empieza entero con la regla de ese momento. Si
  la subsanación fuera una suspensión del plazo (Ley 39/2015, art. 22.1.a), lo
  correcto sería reanudar el plazo, no reiniciarlo. Viene de antes y se
  pregunta a RRHH (apartado 4).
- **Urgencia declarada después de entrar en la fase:** se usa la cantidad
  urgente de la versión vigente al entrar en la fase.
- **Ajuste deshecho:** se publica otra versión con el valor anterior; los
  plazos que empezaron entre medias conservan el suyo.
- **Carrera de milisegundos** entre `vigente_desde` y la confirmación de la
  transacción: un plazo que empiece en ese intervalo puede calcularse con la
  versión anterior. Es aceptable y queda dicho.

Si RRHH quiere dar más tiempo a un expediente concreto que ya está en plazo,
eso es una **ampliación de plazo** (Ley 39/2015, art. 32: antes de que venza y
como mucho la mitad del plazo), un acto sobre ese expediente con su propio
motivo. No se hace cambiando el catálogo. Queda fuera de esta fase.

**Rendimiento.** La lista agrupa por (fase, entrada en fase) y haría casi una
consulta de ajustes por fila. Como las versiones son inmutables, el adaptador
guarda en memoria las versiones ya leídas por número y solo pregunta a la base
cuál es la última.

### 2.6 Pantalla

Va dentro de «Reglas vigentes» (PR #162), no en otra pantalla:

- cada regla ajustable muestra su valor y un botón **Cambiar** (solo si quien
  consulta puede ajustar);
- el formulario muestra los campos permitidos: número con su mínimo y máximo,
  y desplegables con las opciones de la regla (textos por clave i18n, nunca en
  el código);
- motivo (desplegable) obligatorio, referencia y nota opcionales;
- antes de guardar, un resumen: «Plazo de fiscalización: 10 días hábiles →
  7 días hábiles. Se aplicará a los expedientes que entren en fiscalización
  desde ahora. Los que ya están en fiscalización mantienen su plazo.»;
- al guardar, recibo con versión, fecha y hora;
- **Historial** de cada regla: fecha, persona, antes → después y motivo;
- si otra persona guardó antes (versión desfasada), se avisa y se recarga sin
  perder lo escrito;
- una regla con el ajuste fuera de uso lo dice y ofrece revisarlo.

La ayuda va en el botón «?» de la pantalla, no en el texto.

### 2.7 Contrato

- Lectura común: `GET /api/vec/reglas/vigentes` sigue en
  `vec.reglas.vigentes.v1`. Una regla ajustada sale con su valor ajustado, su
  origen normativo y la versión de la base; no rompe la pantalla de la PR #162.
- Edición: `GET /api/contratacion-temporal/rrhh/reglas/ajustes` devuelve las
  reglas ajustables con sus opciones, el ajuste vigente, si quien consulta
  puede ajustar y el historial (paginado, 50 por página).
- Guardar: `POST /api/contratacion-temporal/rrhh/reglas/ajustes` con
  `version_esperada`, `clave_idempotencia`, motivo, referencia, nota y la lista
  de cambios. Respuestas: 201 (versión nueva), 200 (repetición idéntica), 409
  (versión desfasada o clave reutilizada con otro contenido), 422 (valor fuera
  de las opciones o regla no válida), 403 (sin perfil), 503 (base no
  disponible; nunca se da por guardado).
- Hasta que la pantalla del corte 4 esté fusionada, la ruta de guardado queda
  apagada con el interruptor de módulos del panel de administración.

### 2.8 Producción

- Hoy toda la composición de reglas depende de la doble llave de desarrollo y
  exige un paquete de ejemplo (`reglas_ejemplo.go`). No existe todavía una
  composición del catálogo base real. Es una dependencia para producción, no
  de esta fase.
- El catálogo base de producción lo aprueba RRHH; el paquete de ejemplo se
  retira. Los ajustes hechos en desarrollo no se llevan a producción: la tabla
  de ajustes de la principal empieza vacía.
- Antes de pasar a producción, ninguna regla que se use puede quedar con
  origen `ejemplo` sin ajuste aprobado.

## 3. Cortes

Cada corte es una PR pequeña que compila, se prueba y se revisa sola.

1. **Resolutor con ajustes (Go, sin SQL).** Puerto de ajustes, contrato
   `editable`, aplicación del ajuste con fallo aislado por regla, plazos con
   los ajustes vigentes en su inicio, lectura común que admite reglas
   ajustadas, y c01–c04 declaradas ajustables en el paquete de ejemplo. Sin
   almacén de ajustes compuesto, la conducta no cambia. Pruebas con un almacén
   en memoria.
2. **Un solo resolutor de Contratación.** Los cinco sitios que abren el
   fichero pasan a compartir el mismo `Resolutor`; el respaldo fijo de 37 h
   30 min falla cerrado.
3. **SQL de Contratación (CT-000148 y AD3-000114).** Tablas de ajustes,
   cambios y outbox; función de publicación con consumo V3 y huella del
   efecto; lectura de la versión vigente en un instante. Ensayo en el clon de
   la principal y revisión con `revisor-sql-vec`.
4. **Go: almacén y caso de uso.** Adaptador PostgreSQL con caché por versión y
   caso de uso de publicar (validación contra la base, cálculo de cambios y de
   la huella del efecto).
5. **Go: HTTP y perfil fijo.** Rutas de lectura de edición y de guardado,
   perfil fijo de administración de reglas y composición, con la ruta de
   guardado apagada. Revisión de seguridad focal y Semgrep.
6. **Pantalla.** Edición e historial en «Reglas vigentes» (tras fusionar la
   PR #162); se enciende la ruta de guardado. `usabilidad-vec`, `aspecto-vec`,
   `impeccable` y revisión con `revisor-usabilidad-vec`.
7. **Bolsa.** Mismo mecanismo en `vec_bolsa_llamamientos`; resolver las dos
   duplicidades del apartado 1.4; el valor inicial de la política de ofertas
   sale del catálogo (quitar las 48 h de `rrhh-plazos-ui.js`); campos
   ajustables de franja, ventana y días de aviso; admitir
   `vec.bolsa.reglas.ajustes` en `politica_llamamiento_admitida` (CT111), que
   exige otra migración de Contratación porque la tabla es solo de su
   propietario.

## 4. Preguntas abiertas

Para Alberto:

1. ¿Se aplica un plazo de análisis a la fase «solicitud» (c01)? Hoy existe la
   regla pero no se usa. Activarla hace que la lista muestre plazo también en
   solicitudes nuevas.
2. Plazo de respuesta al llamamiento: ¿se confirma que b05 (llamamiento
   directo) y la política de ofertas (ofertas publicadas) son dos plazos
   distintos?
3. ¿Hace falta programar un cambio para una fecha futura («a partir del 1 de
   enero»)? El diseño lo admite (`vigente_desde`), pero el primer corte lo
   aplica al guardar.

Para RRHH (duda 95 de `dudas.md`):

4. Cuando Intervención devuelve el expediente para subsanar y vuelve después a
   fiscalización, ¿el plazo de fiscalización empieza de nuevo o se reanuda
   donde se quedó?

## 5. Revisión independiente

Revisión de diseño (Opus alto, 30/09/2026): **GO con cambios**. Este texto ya
recoge sus hallazgos: la base se resuelve con la hora actual (P0); la lectura
común no se rompe con un ajuste y la edición va en ruta propia (P1); un ajuste
roto deja fuera solo su regla (P1); motivo de catálogo y sin datos personales
en las reglas (P1); decisión AD3 ligada a la huella del efecto (P1); y las
omisiones del inventario, los casos límite, el esquema por módulo y la
división de los cortes (P2).

## 6. Revisión SQL de AD3-114 y CT-148 (pendiente de aplicar)

Revisión independiente (revisor-sql-vec) del commit 927e0dee8 de la rama
`trabajo/plazos-configurables-sql-20260930`: **GO, ENSAYO-OK**. También dio
ENSAYO-OK y `CT148-PRUEBAS-OK` sobre el WIP 369e3ad3b, cuya lógica no revisó
a fondo. Pendiente de aplicar antes de abrir la PR:

1. Validar en la función, con 22023, la longitud, los espacios y los
   caracteres de control de `referencia` y `nota`. Hoy salta el CHECK de la
   tabla con 23514, y Go lo tomaría por un fallo de persistencia.
2. Carrera entre `vigente_desde` (reloj de la sentencia) y la confirmación:
   dejarlo escrito o que el consumidor guarde la versión con la que calculó.
3. Añadir a la prueba un caso con la fachada AD3 real: una decisión de
   consultar sobre material de ajustar debe dar 42501.
4. Corregir el comentario de serialización. El conflicto concurrente lo
   detienen la PK o el UNIQUE con 40001, no la lectura de la cabeza.
5. En el WIP, `solicitud_h` incluye `version_esperada`: comprobar que el
   cliente la repite sin recalcularla y cubrir la repetición en la prueba.

Menores: exigir que `limite` sea número; valorar un patrón sin ceros a la
izquierda para `cantidad`; el relé del outbox necesitará una tabla de
entregas aparte, como CT-131.

La huella del contexto de AD3-114 coincide con la de Go. Condición: el
adaptador calcula la huella del material con `$1::jsonb::text` en
PostgreSQL, como `plantillascatalogo/repositorio.go`.
