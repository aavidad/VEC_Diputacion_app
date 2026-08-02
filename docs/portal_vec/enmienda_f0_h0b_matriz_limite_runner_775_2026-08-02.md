# Enmienda F0-H0b: límite 775 y ejecución por descriptor del runner M38

Fecha: 2 de agosto de 2026.

Estado: **aprobada por doble `GO` documental independiente**; véase la
[evidencia de revisión](revisiones/revision_f0_h0b_matriz_limite_runner_775_2026-08-02.md).

## Motivo medido

La implementación legible del contrato M38 alcanzó 688 líneas en el runner
después de incorporar el selector temprano, las 23 fronteras separadas, el
finalizador de 15 acciones y la propagación inicial. Se detuvo antes de añadir
el conductor y antes de superar el límite entonces vigente de 700 líneas.

La estimación anterior no incluía íntegramente las garantías exigidas después
por las dos revisiones: identidad exterior conocida aunque un hijo muera antes
de informar, ticket por descriptor anónimo, rechazos anteriores a todo recurso,
recuperación exacta caso a caso, orden de observación del resultado y ejecución
desde un único inodo privado ya desvinculado. La medición reproducible y sin
descontar ahorros es:

| Bloque del runner | Mínimo | Máximo conservador |
| --- | ---: | ---: |
| Implementación parcial observada | 688 | 688 |
| Núcleo del conductor, rechazos, recursos y *bootstrap* | +35 | +45 |
| Protocolo seguro por descriptores y orden de resultado | +14 | +24 |
| **Incremento pendiente** | **+49** | **+69** |
| **Runner M38 completo** | **737** | **757** |

Por ello el objetivo local de revisión pasa a **760 líneas o menos** y el
límite duro a **775 líneas o menos**. Este ajuste no sustituye el límite global
de 800 de
[DEC-051](registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
Solo quedan 25 líneas no asignadas hasta ese máximo global; no son una reserva
funcional y no acreditan ni desbloquean I0. I0 permanece bloqueado y exige una
replanificación y decisión separadas.

El auxiliar H0b conserva objetivo de 560 y límite duro de 580 líneas. La
implementación medida ocupa exactamente 580 y queda **congelada**: no recibe
procesos, Docker, `psql`, estados causales, orden de ejecución, finalización,
recuperación ni más lógica para hacer caber el runner. Su SHA-256 definitivo
permanece pendiente hasta cerrar y revisar el código completo; entonces se
fijará literalmente en el runner antes de ejecutar PostgreSQL.

## Propiedad y *write-set*

La implementación conserva exactamente dos ficheros de código:

1. `probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`, propietario del
   conductor, procesos, tickets, contenedores, sesiones, causal, *seam*,
   finalizador, trampa exterior y copia privada del propio runner;
2. `arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh`,
   propietario únicamente del catálogo literal, el oráculo literal puro, su
   comparador y los mutantes estructurales.

El auxiliar SQL D2c, el auxiliar operativo D2d y el capturador permanecen byte
a byte inmutables. No se añade un cuarto artefacto ni se reparte el conductor
en un fichero nuevo. Esa alternativa queda rechazada porque duplicaría la
propiedad del ciclo de vida, ampliaría la carga privada y su acreditación,
crearía otra ventana TOCTOU y obligaría a coordinar dos autoridades sobre
procesos, recursos y trampa exterior.

No se autoriza minificar, agrupar sentencias sin razón de diseño, retirar
mensajes, huellas, acreditaciones, mutantes o regresiones ni trasladar
responsabilidades al auxiliar congelado. Superar 775 líneas es un *hard-stop*
y exige una nueva decisión antes de editar más código.

## Inodo canónico del runner y raíz por descriptor

El checkpoint exige Linux con `procfs` montado y accesible. Si `/proc/self/fd`
no ofrece las semánticas acreditadas, el runner falla cerrado con estado 65;
no existe una vía alternativa por nombre de fichero.

El conductor no invoca 39 veces una ruta viva. Antes del primer hijo debe:

1. acreditar el runner original como fichero regular, no enlace, de un solo
   enlace duro y con modo, UID y tamaño admitidos;
2. registrar sus metadatos físicos y SHA-256 antes de copiar;
3. crear una única copia privada con exclusión, `nofollow`, límite anterior a
   la lectura y permisos privados;
4. abrir esa copia en un descriptor canónico fijo, acreditar por
   `/proc/self/fd` dispositivo, inodo, tipo, modo, UID, enlaces, tamaño y
   SHA-256, y mantener el descriptor abierto durante toda la matriz;
5. desvincular inmediatamente la ruta de la copia y acreditar que el nombre ya
   no existe mientras el descriptor conserva dispositivo e inodo, pasa de un
   enlace a cero y mantiene tipo, modo, UID, tamaño y SHA-256;
6. volver a acreditar el runner original y exigir los mismos metadatos y
   SHA-256 observados antes de copiar;
7. revalidar el inodo canónico y su SHA-256 por descriptor inmediatamente antes
   y después de cada hijo;
8. detenerse con estado 65 ante cualquier divergencia, sin ejecutar el caso
   siguiente.

Cada hijo recibe un duplicado heredado de ese descriptor y ejecuta directamente
`/usr/bin/bash` sobre `/proc/self/fd/<fd>`. No abre la ruta original ni una ruta
temporal del runner. El conductor acredita con pruebas adversas que sustituir,
eliminar y restaurar el antiguo nombre de la copia no cambia el inodo ejecutado.
La restauración crea bajo ese nombre otro fichero byte a byte igual pero con
inodo distinto; nunca vuelve a enlazar ni reemplaza el descriptor canónico.
Después de cada mutación se revalidan sus metadatos y SHA-256 y al terminar se
retira por identidad exacta el fichero adverso o restaurado.

Este protocolo evita la sustitución por ruta y detecta divergencias del inodo
canónico; no promete inmutabilidad criptográfica frente a otro proceso del
mismo UID capaz de escribir mediante un descriptor ya abierto. El entorno de
prueba y los permisos siguen siendo parte de la frontera operativa.

El conductor abre también un descriptor del directorio raíz físico original,
registra dispositivo, inodo, tipo, modo, UID y enlaces y lo mantiene abierto.
El ticket liga cada caso a esos metadatos esperados. El hijo acredita el FD de
raíz y deriva su trabajo desde `/proc/self/fd/<fd-raiz>`; no calcula la raíz
desde `BASH_SOURCE` ni desde el nombre temporal ya desvinculado.

El runner original se acredita mediante metadatos y SHA-256. La raíz, por ser
un directorio, se acredita mediante sus metadatos físicos, no mediante una
SHA-256 de contenido. Ambos descriptores y sus objetos se comprueban antes y
después de la matriz. La retirada solo alcanza recursos privados por identidad
exacta; nunca usa glob, prefijo ni limpieza global.

## Identidad y recuperación de cada hijo

Solo los 39 hijos válidos —`NOMINAL` más los 38 casos del catálogo— reciben
identidad de recursos. Para cada uno, antes de iniciar el proceso, el conductor
genera una identidad de 64 hexadecimales que determina de forma inequívoca la
ruta temporal, el nombre del contenedor y su etiqueta propietaria. El conductor
reserva el directorio temporal real con modo 0700 antes de arrancar y registra
dispositivo, inodo, UID, tipo, modo y número de enlaces. El hijo solo lo adopta
si su lectura coincide exactamente; nunca lo vuelve a crear ni acepta una ruta
equivalente.

Las autopruebas de selector vacío, desconocido, repetido, sin ticket o con
ticket discrepante se lanzan sin identidad, ruta temporal, nombre ni etiqueta
de contenedor. Deben devolver 64 mediante operaciones internas de Bash antes
de ejecutar cualquier orden externa. El conductor acredita para cada rechazo
cero temporales propios, cero contenedores propios y cero invocaciones o
consultas a Docker y `psql`; ni siquiera consulta Docker para inferir esa
ausencia. Estas autopruebas no forman parte de los 39 hijos de la matriz.

La acreditación usa `/usr/bin/bash -x`, un FD de traza preabierto y propiedad
del *bootstrap* del conductor y un `PATH` deliberadamente no resoluble. El
estado debe ser 64 y la traza cerrada debe contener solo las operaciones
internas del parser y el `exit 64`; cualquier intento de `mkdir`, `mktemp`,
Docker, `psql` u otra orden externa invalida la autoprueba. El FD de observación
pertenece al conductor y no constituye un temporal ni una identidad del hijo.

El ticket viaja por un descriptor anónimo y liga `PPID|caso|identidad` con los
metadatos esperados del temporal, la raíz y el runner canónico. El conductor
lanza `/usr/bin/bash` directamente, sin subshell, `env` ni otro proceso
intermedio que cambie el PPID observado. El ticket no es una frontera
criptográfica entre procesos del mismo UID: es una ligadura probatoria interna
contra selectores accidentales o no emitidos por este conductor.

`NOMINAL` es un selector interno explícito, mantiene el *seam* cerrado y entra
directamente en el recorrido hijo. Ningún selector hijo vuelve a despachar el
conductor ni puede iniciar recursión.

`BASH_ENV` y `ENV` quedan eliminados en padre e hijo. El parser de casos válidos
también falla cerrado antes de adoptar el temporal, Docker y `psql` ante
cualquier discrepancia posterior del ticket o de los metadatos ligados.

Cada hijo dispone de proceso, contenedor PostgreSQL 18.4, base, roles,
filesystem, temporales y estado de shell nuevos. Si muere antes de informar, el
conductor reacredita dispositivo e inodo del temporal preasignado y retira solo
esa identidad. Para el contenedor no basta el nombre: si aparece, el conductor
descubre su ID completo y valida conjuntamente ID, nombre exacto y etiqueta
propietaria antes de retirarlo. No usa coincidencias por prefijo.

El hijo emite `RESULTADO` antes de ejecutar su trampa `EXIT`. El conductor
conserva ese estado declarado, espera al proceso y lo contrasta con el estado
real observado **después** de la trampa y de la retirada. Un `79` declarado
solo permanece `79` si finalizador y limpieza exterior terminan verdes;
cualquier fallo exterior lo convierte en 65. La línea previa no puede ocultar
ni sustituir el estado final. Antes del siguiente caso deben estar ausentes el
inodo temporal exacto y el contenedor con la identidad preasignada.

## Concreción de N01/E01 sin cambiar la cardinalidad

Se mantienen las 23 fronteras y las 15 acciones publicadas. En particular:

- F01–F15 solo se arman después de crear y acreditar el R0 canónico; nunca en
  el recorrido `sin-r0`;
- los directorios locales donde se materializarán nominal y error se crean
  antes del `COMMIT` R0 y no constituyen N01/E01;
- antes de crear R0 se prepara en el contenedor únicamente el padre canónico de
  dos hojas de wrapper todavía ausentes;
- después del `COMMIT` y de acreditar R0, `N01` es un
  `docker exec mkdir --mode=0700` real sobre la hoja nominal; `E01` hace lo
  mismo sobre otra hoja de error que sigue ausente. No se usa `--parents`, no
  se sustituye la creación por una validación y ningún `mkdir` post-COMMIT
  queda fuera de una frontera;
- N02/E02 materializan cada modo después de crear su hoja; N07/E07 copian su
  wrapper a esa hoja exacta y N09/E09 lo ejecutan desde allí;
- creación, materialización, cada copia y cada huella permanecen fronteras
  diferentes;
- el orden continúa siendo `A01`, `N01..N10`, `A02`, `E01..E10`, `A03` y el
  finalizador `F01..F15 → TERMINAL`;
- F02 retira de forma agrupada los dos wrappers exactos; F04 retira de forma
  agrupada ambas hojas y solo los padres canónicos que correspondan. Ambas
  acciones son idempotentes ante ausencia exacta, exigen identidad y vacío
  antes de `rmdir` y no usan glob ni prefijo;
- ningún nivel transforma la causal 79 en 65 o 1 cuando toda la limpieza
  termina verde; los fallos de finalización o retirada exterior devuelven 65.

Esta concreción reemplaza la interpretación local anterior de N01/E01, pero no
añade ni fusiona casos: siguen siendo exactamente 23 fronteras, 15 acciones y
38 fallos aislados más el nominal.

El catálogo conserva 38 identificadores y el oráculo 39 entradas literales,
incluido `NOMINAL`. Conductor y catálogo son autoridades separadas; el estado,
la secuencia completa y la condición de recuperación esperados proceden solo
del oráculo.

## Puertas y cierre

Antes de reanudar código, esta enmienda requiere dos revisiones documentales
independientes finales `GO`, ambas con `P0=P1=P2=0`. Después, el checkpoint 4
debe superar Bash, ShellCheck, rechazos 64 anteriores a recursos, mutantes,
límites, huellas, `git diff --check`, Gitleaks, PostgreSQL 18.4, 39 procesos y
contenedores nuevos, Linux/`procfs`, descriptores canónicos de runner y raíz,
mutaciones adversas de la antigua ruta, revalidación del inodo antes y después
de cada hijo, adopción exacta del temporal preasignado, cotejo entre
`RESULTADO` previo y estado posterior a `EXIT`, regresiones H0/A1/C1 y cero
residuos exactos.

Esta decisión no acredita código, PostgreSQL, H0b, C2, F0 ni producción. No
cambia porcentajes: F0 continúa `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
