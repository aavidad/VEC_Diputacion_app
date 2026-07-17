# Decisiones del calculador de experiencia V1

## Estado y naturaleza del documento

| Campo | Valor |
| --- | --- |
| Fecha de corte | 17 de julio de 2026. |
| Naturaleza | Decisión de implementación V1 del motor determinista de experiencia. |
| Alcance | Cálculo de experiencia a partir de un plan compilado y una instantánea de tramos. |
| Fuera de alcance | Autobaremo completo, interpretación autónoma de bases y decisión administrativa sobre méritos. |
| Política de seguridad | Denegación por defecto y fallo cerrado. |
| Validación jurídica | Obligatoria por RRHH y, cuando proceda, Secretaría antes de utilizar una configuración en una convocatoria. |

Este documento fija semántica técnica para que dos ejecuciones con las mismas
entradas produzcan el mismo resultado explicable. **No interpreta por sí mismo
las bases de una convocatoria ni decide qué fórmula es jurídicamente
aplicable.** La traducción de unas bases a catálogos, criterios, coeficientes,
topes, jornadas, unidades temporales, restos y redondeos debe quedar motivada,
versionada y validada por RRHH y, cuando proceda, Secretaría.

Una configuración técnicamente válida puede ser jurídicamente incorrecta para
una convocatoria concreta. Superar el compilador o el calculador no sustituye
la aprobación funcional y jurídica de esa configuración.

## 1. Alcance V1

La V1 calcula exclusivamente la puntuación correspondiente a experiencia. Su
entrada está formada por:

- un plan de experiencia ya compilado y vinculado a un conjunto de reglas
  versionado;
- una instantánea canónica de tramos de servicio, atributos catalogados,
  jornada y atestaciones opacas;
- una fecha de corte inclusiva fijada por el plan.

La V1 no calcula por sí sola el autobaremo íntegro. No contiene todavía la
semántica administrativa de titulaciones, cursos u otros tipos de mérito; el
límite total del autobaremo; los modos orientativo, propuesta, provisional o
definitivo; ni la aceptación o rechazo administrativo de evidencias. El motor
podrá reutilizarse en esas fases, pero recibirá en cada una una instantánea
autorizada y congelada por el caso de uso correspondiente.

Tampoco decide si una experiencia es válida jurídicamente. Esa decisión se
expresa mediante las reglas publicadas y las evidencias previamente admitidas,
siempre bajo la responsabilidad del órgano competente.

## 2. Semántica cerrada de criterios

### 2.1 Operadores booleanos

La V1 adopta estas reglas, sin inferencias adicionales:

- todos los criterios de una regla se combinan mediante **AND**;
- los valores admitidos dentro de un mismo criterio se combinan mediante
  **OR**;
- los atributos adicionales del tramo que una regla no consulta no alteran el
  resultado;
- la ausencia de un atributo exigido hace que la regla no coincida;
- si ninguna regla coincide, el cálculo puede finalizar con cero puntos y debe
  explicar ese resultado.

### 2.2 Identidad exacta del catálogo

La evaluación exige coincidencia exacta de la clave, referencia, versión y
huella del catálogo empleado por la regla. Si el tramo presenta la clave que la
regla necesita pero su catálogo, versión o huella no coincide con el esperado,
el cálculo queda bloqueado con una incidencia `catalogo_incompatible`. No se
transforma silenciosamente en «no cumple» ni se equiparan valores por su texto.

Esta barrera evita que una instantánea antigua o manipulada obtenga cero puntos
aparentemente válido. La migración o equivalencia entre versiones de catálogo
es una operación administrativa explícita y queda fuera del calculador V1.

## 3. Periodo computable, coincidencias y solapes

### 3.1 Definición de periodo

Para restos y redondeos, un **periodo V1** es una pareja seleccionada
`(tramo, regla)` una vez evaluados los criterios, resuelta la coincidencia y
recortado el intervalo por la fecha de corte. No es un día, un mes natural ni
una fragmentación oculta generada por el calculador.

Una misma entrada puede originar varios periodos si la política de coincidencia
permite que su tramo puntúe en varias reglas. Esa multiplicidad debe aparecer
en la explicación.

### 3.2 Coincidencia de reglas sobre el mismo tramo

Cuando un mismo tramo satisface varias reglas:

- si pertenecen a grupos distintos, la V1 bloquea el cálculo; no inventa una
  prioridad entre grupos;
- si pertenecen al mismo grupo, se aplica exactamente la política compilada de
  coincidencia;
- varias reglas conservadas para el mismo tramo son una coincidencia, no un
  solape temporal del tramo consigo mismo.

La prioridad numérica es ascendente: `1` representa la prioridad más alta. Las
prioridades deben ser únicas dentro del grupo.

### 3.3 Solapes de tramos distintos

Existe solape cuando los intervalos efectivos de dos **tramos distintos** se
intersectan dentro del mismo grupo de concurrencia. En V1 cualquier solape de
este tipo bloquea, aunque ambos tramos compartan el mismo `servicioRef`. El
identificador de servicio no permite deduplicar, fusionar ni tolerar un solape.

Los tramos contiguos no se solapan. Los intervalos se comparan después de
resolver criterios, coincidencias, extremos y fecha de corte. Los grupos
separan ámbitos de concurrencia: la V1 no inventa interacción entre tramos que
solo resultan aplicables a grupos distintos.

La detección se realizará con barrido de eventos sobre intervalos, ordenando
los finales antes que los inicios en una misma fecha. Debe mantener activos los
tramos por su identidad y tener complejidad `O(a log a)`, siendo `a` el número
de aplicaciones seleccionadas. No recorrerá el calendario día a día.

## 4. Matriz V1 de fallo cerrado

### 4.1 Concurrencia

| Dimensión | Política | V1 |
| --- | --- | --- |
| Coincidencia | Rechazar | Admitida; más de una regla coincidente bloquea. |
| Coincidencia | Elegir prioridad | Admitida; gana el menor número de prioridad. |
| Coincidencia | Acumular | Admitida; se conservan todas las reglas coincidentes del grupo. |
| Coincidencia | Elegir mayor puntuación | Rechazada por el compilador. |
| Solape | Rechazar | Única política admitida. |
| Solape | Acumular con límite | Rechazada por el compilador. |
| Solape | Elegir mayor puntuación | Rechazada por el compilador. |
| Solape | Elegir mayor dedicación | Rechazada por el compilador. |

No se sustituye una política no soportada por la opción aparentemente más
parecida. Su mera presencia impide compilar el plan.

### 4.2 Restos, frontera y momento de redondeo

| Política de restos | Frontera V1 | Tratamiento |
| --- | --- | --- |
| Conservar exactos | Exacta | Conserva la fracción exacta. |
| Acumular por regla | Regla | Suma fracciones por regla y conserva el total exacto. |
| Descartar por periodo | Periodo | Trunca cada `(tramo, regla)` y explica su resto. |
| Descartar por regla | Regla | Suma por regla, trunca una vez y explica el resto final. |

Son incompatibilidades V1 y bloquean la compilación:

- frontera de restos por regla con redondeo por periodo;
- `maximo_unidades` con redondeo por periodo;
- mínimo de apartado distinto de cero;
- una unidad temporal base distinta del día;
- cualquier modelo de jornada no soportado expresamente por la V1.

## 5. Normalización temporal

Todos los intervalos internos son semiabiertos: `[inicio, fin_exclusivo)`. La
fecha de corte del plan es inclusiva y se transforma una sola vez en
`corte_exclusivo = fecha_corte + 1 día`.

Para cada pareja tramo-regla:

1. `inicio` es la fecha inicial informada.
2. Un tramo en curso termina inicialmente en `corte_exclusivo`.
3. En un tramo cerrado con extremo final exclusivo,
   `fin_exclusivo = fin_informado`.
4. En un tramo cerrado con extremo final inclusivo,
   `fin_exclusivo = fin_informado + 1 día`.
5. El final efectivo es el mínimo entre ese final y
   `corte_exclusivo`.
6. Si `fin_exclusivo <= inicio`, el periodo aporta cero y se explica como
   intervalo vacío por la semántica del extremo.
7. Si el inicio está después del corte, el tramo queda excluido con motivo
   explícito.

La implementación evitará sumar un día a una fecha final cuando ya pueda
recortarla directamente por el corte, con el fin de no introducir un
desbordamiento artificial. El número de días es la longitud exacta del
intervalo semiabierto resultante. Años bisiestos y cambios de mes se resuelven
mediante fechas civiles, nunca mediante duraciones de reloj ni zonas horarias.

## 6. Jornada y atestación protegida

La V1 admite:

| Política | Factor aplicado |
| --- | --- |
| Proporcional | Fracción exacta de jornada del tramo. |
| Íntegra | `1/1`. |
| Íntegra desde umbral | `1/1` si la jornada alcanza el umbral; fracción real en otro caso. |
| Protegida íntegra | `1/1` con atestación válida; fracción real sin ella. |

El modelo por horas no se admite en V1. La comparación con un umbral es exacta
y el propio umbral se considera incluido.

Una atestación de protección cubre **todo el tramo** al que está asociada. La
fuente responsable de construir la instantánea debe dividir el servicio cuando
empiece, termine o cambie la situación protegida, al igual que debe dividirlo
ante cambios de jornada, categoría u otro atributo relevante. El calculador no
deduce fechas parciales ni recibe la causa protegida.

La atestación no se aplica a políticas que no la necesitan. Su presencia puede
quedar indicada de forma neutra en la explicación, pero la causa, diagnóstico
o circunstancia personal nunca forma parte de la entrada ni de la salida.

## 7. Pipeline exacto de cálculo

La ejecución seguirá este orden y no podrá reordenarlo un adaptador:

1. Validar estructura, versión, referencias, huellas y límites del plan y de la
   entrada.
2. Construir índices de reglas y atributos para evitar el producto cartesiano
   entre todos los tramos y todas las reglas.
3. Evaluar criterios con semántica AND/OR y catálogo exacto.
4. Resolver coincidencias dentro de cada grupo y bloquear coincidencias entre
   grupos.
5. Normalizar y recortar cada periodo a un intervalo semiabierto.
6. Detectar solapes entre tramos distintos dentro de cada grupo.
7. Obtener días exactos del intervalo efectivo.
8. Aplicar el factor exacto de jornada.
9. Dividir por las unidades base exigidas por cada unidad puntuable.
10. Aplicar la política de restos en su frontera declarada.
11. Cuando el redondeo sea por regla, aplicar `maximo_unidades`, si existe,
    después del tratamiento de restos y antes del coeficiente.
12. Multiplicar las unidades resultantes por `puntos_por_unidad` mediante
    aritmética racional exacta.
13. Redondear en el momento configurado:
    - por periodo: redondear cada `(tramo, regla)` y sumar después;
    - por regla: agregar primero y redondear una sola vez.
14. Aplicar el máximo de puntos de la regla.
15. Sumar reglas por apartado y aplicar el máximo del apartado.
16. Sumar los apartados ya limitados para obtener el total.
17. Ordenar la explicación canónicamente, serializarla y calcular su huella.

`RedondeoExacto` exige un número entero de micropuntos. Si queda fracción, no
trunca: produce un bloqueo explicable.

Los cálculos intermedios usarán racionales e enteros sin signo de precisión
controlada; no usarán `float32`, `float64` ni aproximaciones decimales. Un
entero redondeado no debe convertirse prematuramente al tipo limitado
`Puntos`: primero se aplican los topes jurídicos de regla y apartado. Así, un
intermedio grande que un tope válido reduzca no genera un falso desbordamiento.
Si después de los topes el total no cabe en los límites técnicos, la ejecución
falla cerrada.

La implementación impondrá presupuestos máximos de aplicaciones, eventos de
intervalo y operaciones exactas. Superarlos es un error técnico, nunca un
resultado parcial.

## 8. Resultado y explicación

### 8.1 Estados separados

El contrato distinguirá al menos:

- **completado**: existe resultado exacto, aunque sea cero;
- **bloqueado por negocio**: la entrada es procesable, pero una regla V1 impide
  concluir, por ejemplo coincidencia rechazada, solape, catálogo incompatible o
  redondeo exacto imposible;
- **error técnico**: manipulación o corrupción de huellas, estructura inválida,
  presupuesto agotado, fallo aritmético o desbordamiento no rescatado por un
  tope.

Un bloqueo de negocio debe ser una salida de dominio canónica y explicable, no
un error opaco. Un error técnico tampoco se transforma en cero puntos ni en un
resultado incompleto.

### 8.2 Contenido mínimo

El resultado incluirá:

- referencia y huella del plan, del conjunto de reglas y de la instantánea;
- fecha de corte;
- por tramo y regla: referencias opacas, criterios evaluados, selección o
  descarte, intervalo original y efectivo, política de extremo, días, jornada
  de origen y aplicada, uso neutro de la atestación, unidades exactas, resto y
  motivo de exclusión o bloqueo;
- por regla: unidades antes y después de topes, coeficiente, puntuación racional
  anterior al redondeo, modo y momento de redondeo, máximo y puntuación final;
- por apartado: suma anterior al máximo, máximo aplicado y resultado;
- total final e incidencias ordenadas canónicamente;
- huella de la representación canónica.

La explicación nunca expresará una equivalencia jurídica que no esté contenida
en la regla validada. Debe indicar qué hizo el motor, no inventar por qué las
bases eligieron esa política.

## 9. Privacidad, registros y auditoría

El calculador no necesita DNI, nombre, correo, dirección, causa médica ni texto
libre. Usará referencias opacas a persona, servicio, evidencia, atestación,
instantánea y regla. La minimización se aplica tanto a entradas como a salidas e
incidencias.

No se registrarán en logs:

- la instantánea completa;
- el resultado o la explicación completos;
- atributos laborales detallados;
- documentos o atestaciones;
- causas protegidas ni datos personales.

Los logs técnicos solo podrán contener identificadores de correlación opacos,
códigos de error seguros, versión del contrato, duración y contadores
agregados. El resultado explicable es información protegida del expediente y
se custodia mediante el puerto correspondiente, con autorización previa a cada
lectura. Auditoría no equivale a volcado indiscriminado de datos.

## 10. Pruebas obligatorias antes de integrar

### 10.1 Casos jurídicos dorados

RRHH debe aprobar un catálogo de casos asociado a la definición y huella
exactas de las bases. Como mínimo:

- jornada completa, media y otras fracciones;
- jornada protegida con y sin atestación, sin conservar la causa;
- umbral alcanzado exactamente y valor inmediatamente inferior;
- servicio dividido por cambios de jornada o protección;
- conversión de días a meses o años según la fórmula exacta publicada;
- restos por periodo y por regla;
- topes de unidades, regla y apartado.

Estos casos prueban que la configuración representa la interpretación aprobada;
el equipo de programación no determina esa interpretación.

### 10.2 Límites temporales y concurrencia

- extremos inclusivo y exclusivo;
- tramo de un solo día e intervalo vacío;
- fecha de corte, tramo abierto y tramo completamente posterior;
- 29 de febrero y límites de mes y año;
- tramos contiguos que no se solapan;
- solape real entre tramos diferentes, incluido igual `servicioRef`;
- varias reglas acumuladas sobre el mismo tramo sin falso autosolape;
- coincidencia en los tres modos admitidos, prioridad `1` y cruce de grupos.

### 10.3 Criterios, aritmética y topes

- AND entre criterios y OR entre valores;
- atributo ausente, atributo adicional y ninguna regla coincidente;
- versión o huella de catálogo incompatible;
- fracciones no representables como micropuntos bajo redondeo exacto;
- diferencias deliberadas entre redondeo por periodo y por regla;
- diferencias deliberadas entre descarte por periodo y por regla;
- máximo de unidades, máximo de regla y máximo de apartado;
- intermedio grande reducido legalmente por un tope;
- desbordamiento final, presupuesto excedido y entrada manipulada.

### 10.4 Propiedades y robustez

- invarianza ante permutar el orden de tramos, reglas y atributos;
- misma entrada canónica produce mismos bytes y misma huella;
- añadir un tramo posterior al corte o no coincidente no cambia los puntos;
- invariancia al dividir tramos solo en las políticas en las que deba
  conservarse, y prueba explícita de la no invariancia cuando el redondeo o
  descarte por periodo la haga intencionada;
- comparación del barrido de intervalos con un oráculo diario limitado a tests;
- fuzzing de fechas, racionales, coincidencias y solapes;
- límites de complejidad, carreras y ausencia de PII o causas protegidas en
  errores, explicaciones técnicas y logs.

Las políticas que el compilador V1 rechaza deben tener pruebas negativas. No
basta con que carezcan de pruebas de cálculo.

## 11. Decisiones pendientes para una V2

La ampliación deberá ser explícita, versionada y acompañada de nueva validación
jurídica y pruebas. Quedan fuera de V1:

1. Coincidencia mediante elección de la regla con mayor puntuación.
2. Solapes mediante acumulación limitada, mayor puntuación o mayor dedicación.
3. Modelos de jornada basados en horas y calendarios laborales.
4. Reglas entre grupos, prioridades globales o compatibilidades cruzadas.
5. Mínimos de apartado y un máximo jurídico global del resultado.
6. Equivalencias o migraciones administradas entre versiones de catálogo.
7. Atestaciones parciales dentro de un tramo sin división previa por la fuente.
8. Cálculo integral de otros méritos y composición del autobaremo completo.
9. Modos orientativo, propuesta, provisional y definitivo, incluidos sus
   límites de declaración y efectos administrativos.
10. Estado de revisión, aceptación, rechazo, rectificación y firma de las
    evidencias; pertenece al flujo administrativo, no a la aritmética V1.
11. Cualquier nueva unidad temporal, regla de redondeo o semántica de restos que
    no figure en la matriz V1.

Ninguna opción V2 se simulará en V1 mediante valores especiales, reglas
duplicadas o comportamiento implícito. Mientras no exista contrato aprobado,
el compilador y el calculador deben rechazarla.

## 12. Puerta de uso en una convocatoria

Antes de usar el calculador en una convocatoria real deben existir, como
mínimo:

1. traducción documentada de cada cláusula de las bases a una regla concreta;
2. revisión funcional de RRHH;
3. validación de Secretaría cuando proceda;
4. versión y huella congeladas del conjunto de reglas y catálogos;
5. casos jurídicos dorados aprobados para esa versión;
6. acreditación técnica de todas las pruebas de este documento;
7. procedimiento de rectificación que no reescriba resultados o reglas
   históricos.

La decisión de implementación V1 establece **cómo** ejecuta el sistema una
regla ya aprobada. RRHH y Secretaría determinan **qué** interpretación de las
bases debe configurarse. Ambas responsabilidades son necesarias y no se
sustituyen entre sí.
