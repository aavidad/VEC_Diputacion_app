# Instrucciones del repositorio para agentes

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
  cada PR, fusiona, sube a cidonia, lo comprueba en Chrome y programa piezas
  propias en su carpeta de trabajo. Solo Claude despliega en cidonia.
- Los equipos Codex (una letra cada uno) programan en local y en GitHub, cada
  uno en su rama. Claude los lanza y los vigila por lo que hacen de verdad, no
  solo por si siguen vivos.
- Astra (Codex de arquitectura, solo lectura) no programa ni entra en un equipo:
  solo debate con Claude.
- Cada encargo termina en algo que se ve en la web. Lo visible que no funciona
  se encarga a terminar; no se quita del menú salvo que Alberto lo pida.

**Decisiones y preguntas (13/10).**

- A Alberto no se le preguntan opciones. Claude decide por consenso escrito con
  Astra, lo anota en el canal y le informa del resultado.
- Solo van a Alberto: gastar dinero o créditos, datos reales, producción o
  sistemas corporativos, y lo que el control de permisos bloquee pidiendo su
  visto bueno expreso. Eso no se deriva a otro agente para saltarlo.
- Ante un fallo, nada de apaños (filas SQL a mano, quitar comprobaciones, datos
  inventados para tapar un hueco): solo vías del producto, como actos de
  administración, migraciones revisadas o pantallas.
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
- el modelo y el esfuerzo justos: el más barato que baste para lo mecánico y
  esfuerzo alto para SQL, seguridad, firma, datos personales y revisiones. Como
  mucho seis subagentes vivos por equipo.

**Cuota.** Se trabaja hasta el 100 % de la cuota semanal y nunca se gastan
créditos; al llegar al 100 % se avisa a Alberto. Bolsa y CT nunca se quedan sin
equipo: si falta cuota, se para antes el equipo de menor prioridad.

**Entrega (13/10).** Menos pruebas y menos esperas: en local solo pruebas
focales; la CI pasa la puerta completa y nunca se repite una prueba sobre el
mismo commit. SQL y seguridad conservan su ensayo en copia de la principal y un
revisor; lo demás, lectura del diff por Claude. Se fusiona al pasar las puertas
rápidas de la CI y se sube a cidonia tras cada fusión con algo visible. cidonia
nunca puede quedar apagada por un fallo de arranque. Ciclo y formato de entrega
en [PLAN_IMPLEMENTACION.md](docs/producto/PLAN_IMPLEMENTACION.md), apartados
10.3 y 10.4. El agente no da su tarea por cerrada: lo hace dirección al
integrar.

**Commits y textos.**

- Commits y PR solo con autoría de aavidad: sin `Co-Authored-By`, sin «Generated
  with» y sin ninguna mención a IA (orden del 16/07).
- Todo lo que lee Alberto, en castellano llano y sin siglas internas: se dice
  qué hace cada cosa («revisión independiente», no «E10»).
- Ningún secreto, certificado, ruta privada ni dato personal en Git. Nunca se
  envía código ni datos a servicios externos para auditarlos.

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
