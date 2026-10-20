# Instrucciones del repositorio para agentes

## Estado vigente — 20 de octubre de 2026

VEC es la aplicación de RRHH de la Diputación de Granada. La referencia de
código es `origin/main` del repositorio `aavidad/VEC_Diputacion_app`. Cada
encargo parte de esa referencia y se desarrolla en rama y carpeta propias.
Se busca primero lo que ya existe en el índice de código, `origin/main`,
ramas y PR abiertas; para Go se usa también `gopls`.

Desde el 15/10 se trabaja en local con copias propias, mediante
[`deploy/principal/copia_local.py`](deploy/principal/copia_local.py).
La excepción de la presentación del 18/10 no reabre cidonia: queda congelada
hasta nueva orden. La siguiente instalación se preparará para un servidor
nuevo; los equipos no despliegan ni cambian servicios compartidos.

El ciclo es PR pequeña, revisión independiente, CI y fusión por dirección.
Claude revisa e integra; el autor no fusiona su PR ni da por cerrada su tarea.
El cierre de la fase 1 exige el caso de [FLUJO_APP §4.0](docs/producto/FLUJO_APP.md),
sus variantes y las condiciones del [plan](docs/producto/PLAN_IMPLEMENTACION.md),
con reinicio de aplicación y PostgreSQL. Un arranque correcto o una actuación
sembrada no acreditan ese recorrido. Lo pendiente de RRHH usa un valor
provisional configurable, documentado como tal. Los datos son siempre ficticios.

La [instantánea anterior](docs/historia/agentes_hasta_20261020.md) conserva
las instrucciones tal como estaban antes de esta revisión. Las reglas que
siguen vigentes se mantienen a continuación y en los documentos de producto.

## Lectura obligatoria de los documentos de producto — 13 de octubre de 2026

Orden de Alberto: todo agente (Claude, subagente o Codex) lee, antes de
programar, revisar o encargar trabajo, los documentos de `docs/producto/` que
afecten a su tarea, y siempre `PRD.md` y `PLAN_IMPLEMENTACION.md`:

- [PRD.md](docs/producto/PRD.md): qué es VEC, para quién y con qué prioridad.
- [TRD.md](docs/producto/TRD.md): pila, arquitectura, seguridad y despliegue.
- [FLUJO_APP.md](docs/producto/FLUJO_APP.md): recorridos y pantallas por perfil.
- [DISENO_UI_UX.md](docs/producto/DISENO_UI_UX.md): reglas de diseño y usabilidad.
- [ESQUEMA_BACKEND.md](docs/producto/ESQUEMA_BACKEND.md): esquemas, roles, migraciones y API.
- [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md): qué falta y en qué orden.
- [TABLERO_TAREAS.md](docs/producto/TABLERO_TAREAS.md): qué falta, quién lo hace y en qué estado.

La skill `documentos-producto-vec` (`.agents/skills/documentos-producto-vec/`)
dice cuál leer según la tarea y cuál actualizar en la misma PR cuando una
entrega cambie lo que describen. Si un documento contradice el código o una
orden vigente de este fichero, mandan el código y la orden; la diferencia se
anota en la entrega.

Las reglas E01 a E13 de [ESPECIFICACIONES_AGENTES.md](ESPECIFICACIONES_AGENTES.md)
ya viven en esos seis documentos; ese fichero solo dice dónde está cada una.

## Cómo se trabaja: órdenes vigentes de Alberto

**Prioridad (08/10).** Bolsa y Contratación temporal primero, con las bases que
necesitan; los demás módulos, después. Ya no rige «todos los módulos en
paralelo» (07/10). Orden exacto y bases en
[PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados 1 y 3.

**Papeles.**

- Claude es director, integrador y programador jefe: fija los encargos, revisa
  cada PR, fusiona, lo comprueba en Chrome y programa piezas propias en su
  carpeta de trabajo. Desde el 15/10 todo se trabaja en local y cidonia queda
  fuera hasta nueva orden: mientras tanto, el ensayo y el recorrido se hacen en
  una copia local. La siguiente instalación corresponde a dirección en el servidor nuevo.
- Los equipos Codex (una letra cada uno) programan en local y en GitHub, cada
  uno en su rama. Claude los lanza y los vigila por lo que hacen de verdad, no
  solo por si siguen vivos.
- Fable y Astra asesoran a Claude en solo lectura: no programan ni entran en un
  equipo.
- Cada encargo termina en algo que se ve en la web. Lo visible que no funciona
  se encarga a terminar; no se quita del menú salvo que Alberto lo pida.

**Decisiones y preguntas (13/10).**

- A Alberto no se le preguntan opciones. Claude decide por consenso escrito con
  Fable y Astra (desde el 15/10), lo anota en el canal y le informa del
  resultado.
- Solo van a Alberto: gastar dinero o créditos, datos reales, producción o
  sistemas corporativos, y lo que el control de permisos bloquee pidiendo su
  visto bueno expreso. Eso no se deriva a otro agente para saltarlo.
- Ante un fallo, nada de apaños que salten la seguridad (quitar comprobaciones,
  falsear firmas, actos o recibos): solo vías del producto, como actos de
  administración, migraciones revisadas o pantallas.
- **Los datos son ficticios (14/10).** Si un bloqueo es de datos (falta un
  puesto, una fila, un estado), se usa otro dato existente o se recarga con la
  semilla o el guion de datos; no se programa una función para fabricarlo.
- En dudas de diseño o de proceso se mira antes cómo lo resuelven tres a cinco
  aplicaciones o administraciones de referencia, con fuentes públicas.
- Lo que depende de RRHH: primero se busca en lo publicado (BOE, BOP de Granada,
  Transparencia de la Diputación, el propio repositorio) y se aplica citando la
  norma como valor configurable. Solo lo interno va a `dudas.md`, numerado tras
  la última, en lenguaje de RRHH y por PR; lo resuelto se quita sin renumerar.
  Lo de Informática va a
  `docs/estudio_requisitos/peticion_informatica_integraciones_2026-09-23.md`.

**Coordinación.**

- Canal: `CANAL_CLAUDE_CODEX.md`, en la carpeta principal y fuera de Git. Los
  equipos escriben ahí PREGUNTA, BLOQUEO, ESPERO o CESIÓN y Claude responde en
  minutos.
- Nada en cola: un encargo nuevo empieza ya, con un subagente o un equipo libre.
- Antes de crear una migración se reserva su número en
  `RESERVAS_MIGRACIONES.md` (carpeta principal, fuera de Git). Nunca se reutiliza
  un número reservado por otra rama.
- No se fusionan, rebasan ni limpian ramas ajenas. Quien crea una carpeta de
  trabajo, una rama, un contenedor o temporales los borra cuando su trabajo está
  integrado.

**Encargos.** Cada encargo dice:

- una sola tarea, con su identificador, rama y carpeta de trabajo propias fuera
  de `/tmp` y la lista de ficheros que va a tocar;
- que se busque primero lo que ya existe (índice de código, `gopls`,
  `origin/main`, ramas y PR abiertas) y se reutilice: no se repite código hecho;
- los documentos de `docs/producto/` y los apartados que debe leer;
- las skills que tocan, que se usan siempre: `usabilidad-vec`, `aspecto-vec`,
  `disenar-sistema-visual-vec` e `impeccable` para pantallas; `humanizer` para
  textos que lee una persona; `programar-backend-vec`, `persistir-autorizar-vec`
  y `programar-interfaz-vec` para programar; `revisar-sql-vec` y `ensayar-sql`
  para SQL; `security-audit` y Semgrep local para autenticación, permisos, SQL o
  entradas HTTP; `revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec`
  para cerrar; `probar-recorridos-vec` para recorridos;
- el modelo y el esfuerzo justos: el más barato que baste. Esfuerzo alto solo
  para programar SQL, permisos, identidad, firma y datos personales, y para las
  revisiones finales; nunca para buscar ni para repetir revisiones. Como mucho
  seis subagentes vivos por equipo.

**Atascos.** Tres fallos iguales o 20 minutos sin avanzar: se escribe BLOQUEO
en el canal y se espera respuesta. No se sube el esfuerzo para salir del
atasco.

**Cuota.** Se trabaja hasta el 100 % de la cuota semanal y nunca se gastan
créditos; al llegar al 100 % se avisa a Alberto. Bolsa y CT nunca se quedan sin
equipo: si falta cuota, se para antes el equipo de menor prioridad.

**Entrega y subida: reglas fijas de Alberto (14/10, «a fuego»).**

1. **A cidonia solo sube una candidata congelada y ensayada entera** en una copia
   de cidonia: SQL en su orden, configuración, actos de administración,
   arranque, recorrido en Chrome y reinicio. Se instala exactamente lo
   ensayado. No hay subidas sueltas para probar. Mientras cidonia esté fuera
   (desde el 15/10), ese ensayo y el recorrido se hacen en una copia local.
2. **Una PR está lista solo si trae código, guion de activación** (configuración,
   actos, datos) **y la prueba de que la app arranca con esa configuración y
   sin ella.** Lo que necesite algo que nadie ha probado junto no está listo.
3. **Un único caso de aceptación del recorrido**, escrito en
   [FLUJO_APP.md](docs/producto/FLUJO_APP.md), apartado 4.0. Lo que lo
   contradiga se decide antes de programar.
4. **Ninguna configuración nueva tumba la app**: falla solo su función. Ningún
   fichero que la app lee al arrancar se cambia sin reinicio vigilado con
   vuelta atrás. cidonia nunca queda apagada (vale para cuando vuelva; ahora
   está fuera por orden del 15/10).
5. **Datos ficticios:** si falta un dato, se ajusta el dato, no se programa.

Pruebas: en local solo las focales; la CI pasa la puerta completa y nunca se
repite una prueba sobre el mismo commit. El SQL se ensaya antes en una copia
local de la principal. Revisiones según la regla 5 de simplificación: dos
independientes para lógica de negocio, permisos, identidad, cifrado o SQL con
permisos; una para lo demás. Se fusiona al pasar las puertas rápidas de la CI. Ciclo y formato de entrega
en [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados
10.3 y 10.4. El agente no da su tarea por cerrada: lo hace dirección al
integrar.

**Simplificación: reglas de dirección (16/10, aprobadas por Alberto).**
Consenso con Fable y Astra; detalle en
[PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartado 10.5.

1. **Código muerto.** Lo que un cambio deja sin uso (función, variante, versión
   de rol, fichero o prueba) se retira en la misma PR. El que se vea en lo que
   se toca, también, demostrando que su protección sigue cubierta. Mover
   ficheros no cuenta como reducción.
2. **Tamaño de PR.** Tope de 1.500 líneas; objetivo de 400 escritas a mano, sin
   contar lo generado ni los datos de prueba.
3. **Núcleo de autorización.** Las migraciones nuevas de
   `deploy/postgresql/autorizacion_atestada_v3` solo las escribe el equipo R. Las
   que ya están en PR siguen el orden fijo del tablero. Como mucho una
   reconstrucción de la función central por semana.
4. **Lecturas.** Una lectura nueva se añade como fila de la lista existente, no
   como función ni versión nueva.
5. **Revisiones.** Dos independientes solo para lógica de negocio, permisos,
   identidad, cifrado o SQL con permisos. Lo demás, una.
6. **Controles.** Cada control nuevo dice de qué protege y qué prueba lo
   comprueba. Cada excepción temporal tiene responsable y condición de
   retirada. Ninguna capa nueva sin un consumidor del recorrido vigente.
7. **Lo opcional** falla cerrado sin tumbar el arranque, con un error de
   catálogo que dice qué falta.
8. **Objetivos medibles.** Copia local en menos de 15 minutos con un guion
   único; CI con todas las migraciones y pruebas SQL del recorrido de la fase 1;
   lecturas por debajo de 300 ms en el percentil 95.

**Commits y textos.**

- Commits y PR solo con autoría de aavidad: sin `Co-Authored-By`, sin «Generated
  with» y sin ninguna mención a IA (orden del 16/07).
- Todo lo que lee Alberto, en castellano llano y sin siglas internas: se dice
  qué hace cada cosa («revisión independiente», no «E10»).
- Ningún secreto, certificado, ruta privada ni dato personal en Git. Nunca se
  envía código ni datos a servicios externos para auditarlos.

## Reglas técnicas y de operación que se conservan

- Arquitectura hexagonal y módulos enchufables: dominio y aplicación dependen
  de puertos; los adaptadores implementan esos contratos. La composición vive
  en `internal/app/bootstrap`. Persona es común; cada módulo conserva sus
  datos y no consulta tablas de otro. Se conservan auditoría, privacidad,
  recibos, idempotencia e historia.
- Las lecturas usan la sesión ligada al certificado y dejan constancia de
  auditoría. Los actos consumen una decisión de autorización estricta en la
  misma transacción que su efecto. Identificarse no firma documentos y un
  recibo local no confirma una operación externa.
- Toda función nueva o reconstruida `SECURITY DEFINER` lleva
  `SET search_path = pg_catalog, pg_temp`. No se rebajan comprobaciones para
  resolver un bloqueo. Se revisa la seguridad de lo cambiado con
  `security-audit`, Semgrep local (`--metrics=off`) y GoSec cuando corresponda;
  nunca se envía código ni datos a servicios externos para auditar.
- Ningún texto ni idioma se incrusta en el código. Los textos visibles viven
  en `web/static/textos/<idioma>/`; se conserva el sistema común de i18n y
  diseño. Escritorio sin desplazamiento de página, con adaptación móvil,
  teclado y foco accesibles. La ayuda se abre con «?».
- SQL: se reserva el número antes de escribir en `RESERVAS_MIGRACIONES.md`,
  fuera de Git. La lista común de instalación es
  `deploy/principal/lista_sql_main_tras_hz37.txt`. El ensayo SQL del recorrido
  usa `deploy/postgresql/fase1/ensayar.py`; la copia funcional usa
  `deploy/principal/copia_local.py`. No se confunden ambos ensayos.
  Se parte del cuerpo instalado en la principal al reconstruir una función.
  Nunca se reaplica ni se modifica una migración instalada, ni se usa DOWN
  sobre historia. El ensayo se hace una vez sobre la cabeza final en una
  copia propia de PostgreSQL 18; revisión SQL independiente antes de integrar.
- Autoridad, admisiones y asignaciones de desarrollo duran entre 60 y 90 días.
  La aclaración de dirección del 20/10 conserva los certificados del kit:
  el máximo de 90 días no se aplica a esos certificados. Una vigencia fija
  incompatible se comunica a dirección.

## Ritmo y material de trabajo — órdenes del 18 al 20 de octubre

- Carpetas de trabajo y PGDATA desechables en `/dev/shm/vec`; guiones,
  preparadores, fuentes y evidencias privadas en la carpeta de estado propia,
  fuera de Git. El material común de candidata se lee sin modificarlo.
- Antes de restaurar o compilar se miran `free -g` y `df -h /dev/shm`.
  Con menos de 40 GB disponibles se retiran copias y temporales propios que
  sobren y se avisa en el canal. Se conservan 32 GB libres para el sistema.
  PostgreSQL propio: `--rm --restart=no --memory 2g`. No se borra material ajeno.
- Commit y push al menos cada 30 minutos mientras haya trabajo nuevo;
  ESTADO de una línea en el canal. Aclaración de dirección del 20/10 a las
  12:22: si la cabeza está congelada esperando revisión o CI, basta el ESTADO;
  no se añade un commit vacío que reinicie las comprobaciones.
- Cada tarea tiene el nombre del tablero; títulos de PR: `Tarea: <ID> — …`.
  Los encargos de documentación usan el identificador DOC de su área.
  Se avisa TOCO antes de editar un fichero compartido y LIBERO al subirlo.
  No se toca un documento ocupado por otra PR abierta.
- Una PR documental pide `REVISAR #N` en el canal y recibe una revisión.
  Activación y SQL se declaran «ninguna» si solo cambian documentos; no se
  atribuyen a esa PR ensayos funcionales que no se han ejecutado.

## Dónde están las reglas que antes vivían aquí

| Sección antigua | Dónde vive ahora |
| --- | --- |
| Prioridad vigente | [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados 1 y 3 |
| Trabajo paralelo y granularidad | [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados 10.1 y 10.2 |
| Ciclo obligatorio de entrega y formato de entrega | [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados 10.3 y 10.4 |
| Arquitectura no negociable | [TRD.md](docs/producto/TRD.md), apartados 2 y 3; datos entre módulos en [ESQUEMA_BACKEND.md](docs/producto/ESQUEMA_BACKEND.md), apartado 3 |
| Seguridad y protección de datos | [TRD.md](docs/producto/TRD.md), apartados 5 a 8 y 11 |
| i18n, idioma y presentación | [DISENO_UI_UX.md](docs/producto/DISENO_UI_UX.md), apartados 4, 8 y 9; parte técnica en [TRD.md](docs/producto/TRD.md), apartado 12 |
| Calidad y tamaño de ficheros | [TRD.md](docs/producto/TRD.md), apartados 13 y 14; [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartado 10.2 |
| Relevos de agosto y septiembre, checkpoint del 31/08, bloques OpenClaw | [docs/historico/AGENTS_historico_hasta_20261013.md](docs/historico/AGENTS_historico_hasta_20261013.md) |

Las reglas de los relevos antiguos que siguen en vigor (no reaplicar migraciones
con historia, identificarse no es firmar, un recibo local no confirma un sistema
externo…) están en [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md),
apartado 8.
