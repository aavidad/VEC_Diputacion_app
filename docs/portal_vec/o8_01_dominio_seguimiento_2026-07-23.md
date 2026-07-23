# O8-01 — Dominio gobernado de seguimiento laboral

Fecha: 23 de julio de 2026.

Estado del corte: modelo de dominio aislado, sujeto a revisión independiente.

## Alcance

O8-01 incorpora un agregado `Seguimiento` separado del expediente coordinador.
Queda ligado exclusivamente mediante referencias opacas a:

- organización;
- expediente de contratación temporal;
- relación o nombramiento.

No contiene nombre, DNI, correo, teléfono, ubicación, fichajes, causas médicas
ni texto libre. Actor, unidad, documentos, recibo y correlación también se
representan mediante referencias opacas.

El contrato técnico de referencia opaca es `ref:` seguido de 64 dígitos
hexadecimales en minúsculas y distintos de cero. Por tanto, valores legibles
como un DNI, nombre o teléfono no atraviesan el agregado. El futuro adaptador
de referencias debe obtener el material mediante generación aleatoria o HMAC
con secreto externo; no debe convertir un dato personal de baja entropía en
referencia mediante un hash simple.

Este corte cubre:

- incorporación;
- prórroga;
- incidencia;
- suspensión o espera cuando una definición publicada la configure;
- cese previsto mediante el periodo proyectado;
- cese efectivo;
- rectificación compensatoria;
- reapertura expresamente publicada.

## Autoridad funcional

Los estados, motivos, transiciones, documentos exigibles, ámbitos y resultados
de calendario proceden de una `DefinicionSeguimiento` publicada, versionada,
fechada e inmutable. La publicación contiene referencia, versión, huella
SHA-256, vigencia semiabierta `[desde, hasta)` y canon cerrado.

Las claves son administrables. Añadir una transición con operaciones técnicas
ya soportadas no requiere modificar ni recompilar el núcleo.

La publicación valida antes de calcular su huella:

1. límites de estados, motivos, transiciones y requisitos;
2. unicidad de claves;
3. existencia y carácter no final del estado inicial;
4. existencia de al menos un estado final;
5. origen y destino de cada transición;
6. motivos y documentos permitidos u obligatorios;
7. ámbitos y resultados admitidos para la evidencia de calendario;
8. coherencia del requisito de periodo con su efecto técnico;
9. salida desde un estado final solo por rectificación o reapertura publicada;
10. ausencia de ciclos silenciosos cuando la publicación los prohíbe.

No se han compilado causas laborales, modalidades, festivos, plazos ni efectos
jurídicos de una incidencia. Una incidencia solo cambia lo que declare la
transición publicada.

## Operaciones técnicas sobre periodos

El dominio admite cuatro operaciones cerradas de consistencia:

- `ninguno`: registra la actuación sin alterar la proyección temporal;
- `abrir`: materializa el periodo inicial previsto;
- `ampliar`: añade un tramo posterior no solapado;
- `cerrar`: registra un cese efectivo separado.
- `rectificar_tramo`: sustituye la proyección de un tramo mediante un acto
  compensatorio;
- `rectificar_cese`: sustituye la proyección del cese mediante un acto
  compensatorio;
- `reabrir`: retira el cese de la proyección vigente sin retirarlo de la
  historia.

Estas operaciones no son causas ni reglas laborales. La definición gobernada
decide qué transición las utiliza.

El periodo previsto inicial no se sobrescribe ni obliga a fingir que lo
previsto fue lo efectivamente materializado. La incorporación aporta el primer
tramo resultante y cada prórroga añade otro. Una ampliación:

- no puede solaparse;
- debe comenzar al terminar el último tramo o después;
- no borra ni reduce los tramos anteriores.

El cese efectivo queda enlazado a su actuación. No puede preceder a la
incorporación ni al instante de registro de su propio acto.

## Actuaciones e historia

Cada actuación conserva:

- secuencia y versión CAS;
- definición exacta aplicada;
- transición y clase;
- estado origen y destino;
- motivo catalogado;
- actor y unidad por referencia;
- instante efectivo y de registro;
- documentos tipados por clave y referencia;
- periodo, si la publicación lo exige;
- referencia, versión, huella, ámbito y resultado del calendario, si lo exige;
- actuación, recibo y correlación;
- actuación compensada, en una rectificación;
- huella de petición, huella anterior y huella propia.

La historia es de solo adición. Una rectificación añade una actuación
compensatoria, conserva intacta la actuación anterior y recalcula únicamente
la proyección vigente del tramo o cese corregido. Cuando la definición exige
segregación, el actor de rectificación debe ser distinto del actor del acto
rectificado. La reapertura compensa la proyección del cese y permite que una
transición posterior publique un nuevo cese; el cese anterior permanece en la
cadena.

Un estado final rechaza transiciones ordinarias. La definición puede publicar:

- una rectificación desde final hacia otro estado final;
- una reapertura desde final hacia un estado no final.

O8-01 no decide las tareas pendientes ni autoriza el cierre administrativo;
esa coordinación corresponde a O8-02.

## CAS e idempotencia

La creación parte de versión cero. Cada transición nueva exige la versión
esperada y aumenta versión y secuencia en una unidad.

- versión incorrecta: `ErrVersionEnConflicto`, sin mutación;
- misma referencia y misma huella semántica: repetición exacta, sin nueva
  actuación;
- misma referencia con contenido distinto:
  `ErrActuacionSeguimientoEnConflicto`, sin mutación.

El agregado se trata como valor inmutable. Aplicar devuelve otro agregado y
todos los accesores de slices o punteros entregan copias defensivas.

## Rehidratación e integridad

`EstadoPersistidoSeguimiento` es el contrato de estado futuro para un adaptador,
pero este corte no lo persiste.

La rehidratación:

1. vuelve a validar y resumir la definición;
2. reconstruye la raíz desde referencias, definición, estado inicial, periodo
   previsto e instante de creación;
3. reproduce cada evento en orden mediante las mismas reglas;
4. coteja secuencia, versión, definición, cadena de huellas y huellas de
   petición y actuación;
5. compara la serialización canónica de la proyección reconstruida con la
   recibida.

La reproducción mantiene un índice local de referencias duplicadas y no clona
el historial completo en cada evento. Las colecciones se limitan antes de
recorrerlas.

La huella SHA-256 detecta alteraciones y colisiones semánticas dentro del
modelo, pero no acredita por sí sola origen, competencia, autorización ni
custodia. El futuro adaptador durable deberá añadir autenticidad, ACL,
auditoría y persistencia de solo adición en una transacción autorizada.

## Canon V1

El canon usa un formato binario explícito:

- dominio de separación;
- versión de esquema `1`;
- algoritmo `sha-256`;
- cadenas UTF-8 precedidas por longitud `uint32`;
- enteros big-endian;
- instantes UTC en microsegundos;
- presencia explícita para valores opcionales;
- colecciones normalizadas y ordenadas;
- ausencia de mapas, reflexión, etiquetas JSON u `omitempty`.

Hay dominios separados para definición, raíz, petición, actuación y estado.
Esquemas o dominios desconocidos se rechazan. Antes de recorrer o serializar
se validan referencias, definición, estado, periodos, cese, instantes,
secuencia, cadena de huellas y cardinalidades.

## Calendario

El dominio no conoce festivos ni calcula días hábiles. Solo conserva una
evidencia ya calculada:

- referencia y versión de calendario;
- huella SHA-256;
- clave de ámbito territorial;
- clave de resultado;
- instante del cálculo.

La definición limita ámbitos y resultados admitidos. La capa de aplicación
deberá obtener la evidencia desde un conector de calendario intercambiable y
autorizado.

## Seguridad, protección de datos e i18n

- Denegación predeterminada: toda clave o requisito no publicado se rechaza.
- Minimización: no hay datos identificativos ni categorías especiales.
- Exactitud: definición, actor, unidad, fuente temporal, recibo y huellas
  quedan inmovilizados.
- Trazabilidad: secuencia, CAS, correlación y cadena append-only.
- i18n: estados, motivos, transiciones, documentos, ámbitos y resultados son
  claves administrables; el dominio no contiene etiquetas visibles.
- Neutralidad: no hay dependencia de HTTP, web, escritorio, CLI, MCP, SQL ni
  proveedor.
- Errores: mensajes generales en castellano, sin referencias privadas.
- IA: no existe decisión automatizada ni tratamiento mediante IA.

## Evidencia de pruebas

La familia `seguimiento*_test.go` cubre:

- incorporación, prórroga, incidencia y cese;
- transición añadida solo por definición;
- transición, estado, motivo, documento y calendario incorrectos;
- CAS, repetición exacta y colisión semántica;
- solapamiento, UTC, precisión, orden y extremos transportables;
- cese temporalmente incoherente y operación ordinaria tras final;
- rectificación append-only y segregación de actor;
- corrección de tramo y cese, reapertura y nuevo cese;
- rehidratación exacta y adulteración de todos los eventos;
- copias defensivas y derivación concurrente sin carrera;
- canon determinista, esquema cerrado y límites;
- rechazo de valores personales como referencias opacas.

## Límites y siguiente frontera

Este corte no incluye:

- autorización ni segregación organizativa ejecutable;
- persistencia, auditoría durable u outbox;
- API, aplicación, web, escritorio, CLI o MCP;
- consulta o cálculo de calendario;
- integración con Personal, nómina o GINPIX;
- firma, registro, notificación, conservación o expurgo;
- normativa, causas, plazos o calendarios concretos;
- habilitación de datos reales, efectos jurídicos o producción.

Las aprobaciones CT-CUM-02 a CT-CUM-10 continúan pendientes. La categorización
ENS, EIPD, competencia, política ENI, conservación y validación formal
corresponden a los responsables indicados en la matriz normativa.

**O8-01 modela dominio; no persiste, expone API ni habilita producción.**
