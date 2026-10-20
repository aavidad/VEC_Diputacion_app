# Instrucciones del repositorio para agentes

## Estado vigente — 20 de octubre de 2026

VEC es la aplicación de RRHH de la Diputación de Granada. La referencia de
código es `origin/main` del repositorio `aavidad/VEC_Diputacion_app`. Cada
encargo parte de esa referencia y se desarrolla en rama y carpeta propias.
Antes de escribir se busca en el índice de código, `origin/main`, ramas y PR
abiertas para reutilizar lo que existe; para Go se usa `gopls`.

Se trabaja en copias locales propias con
[`copia_local.py`](deploy/principal/copia_local.py), porque cidonia permanece
congelada. La siguiente instalación se preparará para un servidor nuevo, sin
fecha ni máquina decididas. Los equipos no despliegan ni cambian servicios
compartidos. La excepción de la presentación del 18/10 no reabre el servidor.

Bolsa y Contratación temporal tienen prioridad, con las bases que necesitan;
los demás módulos siguen después. El cierre de fase 1 exige el caso único de
[FLUJO_APP §4.0](docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación),
sus variantes y las condiciones del plan, con reinicio de aplicación y
PostgreSQL. Un arranque correcto o un acto sembrado no acreditan ese recorrido.
Los datos son ficticios. Lo pendiente de RRHH usa un valor provisional
configurable y documentado; no se fija como constante en el código.

## Lectura y fuentes

Antes de programar, revisar o encargar trabajo se leen los documentos de
producto afectados, siempre PRD y plan de implementación:

| Documento | Contenido |
| --- | --- |
| [PRD](docs/producto/PRD.md) | Alcance, personas usuarias y prioridades |
| [TRD](docs/producto/TRD.md) | Pila, arquitectura, seguridad y despliegue |
| [FLUJO_APP](docs/producto/FLUJO_APP.md) | Recorridos y perfiles; §4.0 es el único caso de aceptación |
| [DISENO_UI_UX](docs/producto/DISENO_UI_UX.md) | Presentación, accesibilidad e idiomas |
| [ESQUEMA_BACKEND](docs/producto/ESQUEMA_BACKEND.md) | Datos, roles, migraciones y API |
| [PLAN_IMPLEMENTACION](docs/producto/PLAN_IMPLEMENTACION.md) | Orden, condiciones de cierre y ciclo de trabajo |
| [TABLERO_TAREAS](docs/producto/TABLERO_TAREAS.md) | Tareas, responsables y estado |

La skill `documentos-producto-vec` indica qué leer y qué actualizar en la misma
PR cuando cambie lo descrito. Ante una contradicción mandan el código y las
órdenes vigentes; se anota y corrige la diferencia. El estado del tablero lo
actualiza dirección. [ESPECIFICACIONES_AGENTES.md](ESPECIFICACIONES_AGENTES.md)
conserva el índice de las trece reglas técnicas hacia estos documentos.

Las dudas funcionales se contrastan primero con BOE, BOP de Granada,
Transparencia de la Diputación y el repositorio. Se cita la fuente normativa
y se mantiene configurable el valor pendiente de confirmación interna.
Solo lo interno va a `dudas.md`, tras el último número y sin renumerar al
resolverlo. Las integraciones pendientes se llevan en
[la petición a Informática](docs/estudio_requisitos/peticion_informatica_integraciones_2026-09-23.md).
Las dudas de diseño y proceso se contrastan con tres a cinco referencias
públicas antes de decidir.

## Arquitectura, seguridad e historia

- Se conserva la arquitectura hexagonal: dominio y aplicación dependen de
  puertos; los adaptadores los implementan. La composición está en
  `internal/app/bootstrap` e `internal/app/composicion`.
- Persona es común. Cada módulo conserva sus datos y no consulta tablas de
  otro; obtiene proyecciones autorizadas mediante contratos. Los módulos son
  enchufables sin duplicar identidad, autorización, auditoría ni i18n.
- Las lecturas usan la sesión ligada al certificado y dejan constancia de
  auditoría. Los actos consumen una decisión de autorización positiva y
  estricta en la misma transacción que el efecto. El perfil o el botón visible
  no conceden autoridad.
- Se conservan privacidad, recibos, idempotencia, versión e historia de solo
  adición. Identificarse no firma documentos. Un recibo local no confirma un
  sistema externo; una indisponibilidad nunca se interpreta como éxito.
- No se saltan comprobaciones ni se falsean firmas, actos o recibos ante un
  fallo. Si falta un dato ficticio se usa otro existente o se recarga con la
  semilla o guion de datos; no se programa una función para fabricarlo.
- Toda función nueva o reconstruida `SECURITY DEFINER` lleva
  `SET search_path = pg_catalog, pg_temp`. No se envía código ni datos a
  servicios externos para auditar. No se incluyen secretos, certificados,
  rutas privadas ni datos personales reales en Git.
- Los cambios de autenticación, permisos, datos personales, SQL, ficheros
  subidos, red o entradas HTTP pasan revisión de seguridad focal con
  `security-audit`, Semgrep local (`--metrics=off`) y GoSec sobre lo cambiado
  cuando corresponda. La auditoría completa se reserva a un encargo expreso
  o al hito que la exija.
- Autoridad, admisiones y asignaciones de desarrollo duran entre 60 y 90 días.
  Se conservan los certificados del kit: ese máximo no se les aplica. Una
  vigencia fija incompatible se comunica para corregirla.

## SQL y copias de ensayo

Antes de crear una migración se reserva su número en
`RESERVAS_MIGRACIONES.md`, en la carpeta principal y fuera de Git. No se usa
un número reservado por otra rama. Las migraciones nuevas de
`deploy/postgresql/autorizacion_atestada_v3` corresponden al propietario del
núcleo indicado en el tablero; las pendientes respetan su orden. Como máximo
se reconstruye la función central una vez por semana.

La lista común es
[`lista_sql_main_tras_hz37.txt`](deploy/principal/lista_sql_main_tras_hz37.txt).
Se respeta el orden causal y se parte del cuerpo instalado en la principal
al reconstruir una función. No se reaplica ni modifica una migración
instalada, ni se usa DOWN sobre historia. Las lecturas nuevas se añaden como
filas de la lista existente, no como funciones o versiones nuevas.

El ensayo SQL usa
[`ensayar.py`](deploy/postgresql/fase1/ensayar.py); la copia funcional usa
[`copia_local.py`](deploy/principal/copia_local.py). Son comprobaciones
complementarias. El SQL se ensaya una vez sobre la cabeza final en una copia
propia de PostgreSQL 18, con revisión SQL independiente antes de integrar.
No se ensaya contra cidonia ni se reaplican migraciones para repetir un ensayo.

## Interfaz, textos y documentación pública

Ningún texto ni idioma se incrusta en el código. Los textos visibles viven en
`web/static/textos/<idioma>/` y usan el sistema común de i18n y diseño.
Se prioriza escritorio sin desplazamiento de página, con adaptación móvil,
teclado, foco y contraste accesibles. La ayuda se abre con «?».
Cada capacidad tiene una pantalla consumidora real. Lo visible que falla se
termina; no se quita del menú para ocultarlo.

Se escribe en castellano llano, en impersonal y explicando qué se hace y por
qué. Se evitan atribuciones personales de las reglas, historia de sesiones,
relevos, nombres de equipos y cifras de cuota en los documentos vigentes.
Los antecedentes técnicos útiles van a `docs/historico/`; lo que no aporta se
retira. Los manuales de usuario explican tareas con vocabulario de RRHH y sin
identificadores internos. Los textos pasan `humanizer`.

El repositorio es público: no incluye manuales, pliegos, capturas ni PDF de
otros productos o administraciones. Las referencias públicas se citan sin
redistribuir sus adjuntos. VEC importa datos de CONVOCA y se integra mediante
contratos; no se describe como copia de otro producto. Los datos ficticios
propios de importación se conservan.

## Encargos, revisión e integración

Cada encargo declara una tarea del tablero, rama, carpeta y ficheros propios,
sus fuentes, dependencias, consumidor, invariantes y prueba de cierre. No se
amplía el alcance por iniciativa propia. Los títulos de PR siguen
`Tarea: <ID> — …`; la documentación usa el identificador DOC de su área.
Una responsabilidad por PR, objetivo de 400 líneas manuales y máximo de
1.500, sin contar generado ni datos de prueba. Se repasa el encargo cada
30 minutos para evitar desviaciones.

Se nombran y usan las skills aplicables: `documentos-producto-vec`;
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e `impeccable`
para pantallas; `humanizer` para textos; `programar-backend-vec`,
`persistir-autorizar-vec` y `programar-interfaz-vec` para programar;
`revisar-sql-vec` y `ensayar-sql` para SQL; `probar-recorridos-vec` para
recorridos; `revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec` para
entregar. Se usa el modelo más barato suficiente; esfuerzo alto solo en
programación sensible y revisión final. Como máximo seis ayudantes, o el
límite menor del encargo; no se sube esfuerzo para resolver un atasco.

El código que quede sin uso se retira en la misma PR, demostrando que su
protección sigue cubierta. Mover ficheros no cuenta como reducción.
Cada control nuevo declara de qué protege y qué prueba lo comprueba; cada
excepción temporal tiene responsable y condición de retirada. Ninguna capa
nueva se añade sin consumidor del recorrido vigente.

La PR incluye código, guion de activación, configuración, actos y datos
necesarios, SQL ordenado y arranque probado con la configuración nueva y sin
ella. Lo opcional falla cerrado sin tumbar el arranque, con error catalogado
que explique qué falta. No se cambia un fichero leído al arrancar sin
reinicio vigilado y recuperación. Una PR documental declara configuración,
SQL y activación «ninguna» cuando corresponda.

En local solo se ejecutan pruebas focales proporcionales y `-race` si hay
concurrencia. La CI pasa la puerta completa, incluida
`scripts/verificar_calidad.sh`; no se repite sobre el mismo commit.
Se requieren dos revisiones independientes para lógica de negocio, permisos,
identidad, cifrado o SQL con permisos; una para lo demás. Pantallas llevan
revisión de usabilidad y recorrido con Playwright y Chrome del sistema a
1440 y 390 px; SQL lleva su revisión especializada. La revisión identifica
la cabeza exacta; un cambio posterior reabre solo la parte afectada.

Dirección lee y fusiona al pasar las puertas rápidas; la calidad completa se
vigila y sus fallos se corrigen. El autor no aprueba ni fusiona su propia PR,
ni declara cerrada la tarea. Dirección cierra al integrar. Antes de instalar se
ensaya una candidata congelada completa, con configuración, actos, SQL en
orden, recorrido en Chrome y reinicio; se instala exactamente lo ensayado.

Objetivos medibles: copia local en menos de 15 minutos con un guion único;
CI con migraciones y pruebas SQL del recorrido de fase 1; lecturas por debajo
de 300 ms en el percentil 95. El detalle de entrega y simplificación está en
[PLAN_IMPLEMENTACION §10](docs/producto/PLAN_IMPLEMENTACION.md#10-cómo-se-trabaja).

## Coordinación y material de trabajo

La coordinación va en `CANAL_CLAUDE_CODEX.md`, en la carpeta principal y fuera
de Git. Se registra APERTURA, ESTADO, PREGUNTA, BLOQUEO, TOCO, LIBERO,
ESTORBA, LISTA PARA FUSIONAR o FIN. Se avisa antes de tocar un fichero
compartido y se libera al subir. No se edita un fichero ocupado por otra PR,
ni se fusionan, rebasan o limpian ramas ajenas.

Las decisiones ordinarias corresponden a dirección y se anotan en el canal.
Solo se eleva al operador gasto, datos reales, producción, sistemas
corporativos o una acción que requiera su aprobación expresa. No se deriva a
otro agente para saltar un control. Tres fallos iguales o 20 minutos sin
avance requieren BLOQUEO con el error concreto; se sigue con lo independiente.
Los encargos nuevos se asignan a quien pueda empezarlos sin solapar ficheros.
No se gastan créditos y se conserva la prioridad de Bolsa y Contratación.

Carpetas de trabajo y PGDATA desechables van en `/dev/shm/vec`; guiones,
preparadores y evidencias privadas, en la carpeta de estado propia en disco,
fuera de Git. El material común de candidata solo se lee. Antes de restaurar
o compilar se miran `free -g`, `df -h /dev/shm` y `df -h /home`.
Con menos de 40 GB disponibles se retira material propio sin uso y se avisa;
se conservan 32 GB para el sistema. No se trabaja con menos de 10 GB libres
en disco. PostgreSQL propio usa `--rm --restart=no --memory 2g`.

Se hace commit y push al menos cada 30 minutos mientras haya trabajo nuevo,
y se escribe ESTADO. Una cabeza congelada esperando revisión o CI solo exige
ESTADO, para no reiniciar comprobaciones con commits vacíos. Commits pequeños
con autoría `aavidad`, sin coautorías, atribuciones automáticas ni menciones a
herramientas de generación. Se retira únicamente material propio tras la
integración, con árbol limpio y sin usuarios de la carpeta o servicio.

## Antecedentes y reglas absorbidas

El [extracto anterior](docs/historico/AGENTS_hasta_20261020.md) y el
[histórico hasta el 13/10](docs/historico/AGENTS_historico_hasta_20261013.md)
conservan contexto. Las reglas de arquitectura, datos, seguridad, calidad e
i18n viven en TRD, ESQUEMA_BACKEND y DISENO_UI_UX; las de orden, granularidad,
entrega y relevos anteriores, en PLAN_IMPLEMENTACION §8 y §10.
