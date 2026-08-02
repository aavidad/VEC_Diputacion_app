# Enmienda F0-H0b M38: presupuesto real y topología H0

Fecha: 2 de agosto de 2026.

Estado: candidata; no autoriza reanudar la implementación hasta obtener dos
revisiones independientes con `P0=0`, `P1=0` y `P2=0`.

Prevalencia propuesta: modifica la
[enmienda del adaptador privado](enmienda_f0_h0b_m38_ciclo_privado_2026-08-02.md)
solo en el presupuesto local del adaptador, la topología interior declarada,
el reparto de la causal y el finalizador y la semántica de grupo/ausencia de
F02, F03 y F04. Permanecen vigentes el resto de sus controles, puertas y
límites, así como el máximo global de 800 líneas de
[DEC-051](registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).

## Alto observado

La implementación no confirmada de `agent/f0-h0b-funcional-20260802`, sobre
`25b17cf`, alcanzó esta foto exacta:

| Fichero | Líneas | SHA-256 observado |
| --- | ---: | --- |
| Runner | 725 | `f93080731afb7343bab0f1d04f50dc3d47ab41b8d12e645689a7ac43b263627a` |
| Auxiliar H0b | 580 | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| Adaptador privado | 361 | `a108981d6a7c5832b28a797b2579b7417c7a31bfe3e57f1f6ed24961372c68b1` |

El adaptador superó en once líneas el límite local de 350. El productor se
detuvo sin comprimir, eliminar controles ni confirmar el código.

Según el informe de sesión del productor —no incorporado como acta o salida
probatoria—, una primera preprueba falló cerrada antes de Docker porque el
manifiesto se comparaba en orden de argumentos y el capturador lo emitía en
orden canónico.
Corregido ese defecto, una segunda preprueba alcanzó H0b y falló cerrada al
tratar como basal el padre
`/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes`.
No es basal en el inventario H0. Ambas ejecuciones son observaciones de sesión
diagnósticas y no acreditan H0, M38 ni PostgreSQL.

El mismo informe declaró cero contenedores y temporales propios `vec-f0-h0-*`
después del fallo; esa declaración debe repetirse y persistirse en C4a. Las
auditorías sí comprobaron directamente sobre el WIP conservado las líneas y
huellas de la tabla, la invariancia de D2c, D2d y el capturador, el éxito de
`bash -n` y `git diff --check`, el rechazo SC2016 de ShellCheck y la huella
literal obsoleta del adaptador. El estado sigue siendo `NO-GO`.

## Por qué no basta con subir 350 a 370

Tres auditorías independientes y solo lectura encontraron trabajo funcional
que la estimación de 265..349 no había absorbido:

- registrar cada identidad física inmediatamente después de crear o copiar y
  antes del seam N03/E03, N05/E05 y N07/E07;
- rechazar tanto existencia como enlace simbólico colgante al acreditar una
  ausencia;
- validar el grupo completo en F02/F03/F04, incluidos los destinos conocidos
  que carezcan de identidad porque deben seguir exactamente ausentes;
- registrar una retirada parcial controlada si la segunda pasada se interrumpe;
- exigir topología exacta, no solo un árbol compuesto por directorios;
- hacer idempotentes la retirada interior y el epílogo exterior;
- diferir y drenar señales, conservar y restaurar `monitor` y la exportación de
  `SHELLOPTS`, y cerrar la ventana fork -> `$!`;
- ligar PID, PGID, PPID y tiempo de inicio, y acreditar la extinción del grupo;
- reconciliar CID, cidfile, nombre, etiqueta, imagen, estado y postausencia;
- hacer globales y exactos los plazos de readiness y del caso;
- endurecer los rechazos 64 y añadir los mutantes interiores y exteriores ya
  exigidos;
- descomprimir líneas opacas sin ocultar sentencias para caber.

Ampliar solo once o veinte líneas produciría otro falso margen y empujaría las
pruebas obligatorias fuera del presupuesto.

## Topología H0 exacta

El inventario H0 deja como padres basales dentro de `/repo_h0b`:

```text
/repo_h0b
└── deploy/postgresql/autorizacion_atestada_v3
    └── migraciones
        └── 000007_componentes
```

Son basales también los segmentos intermedios `deploy` y
`deploy/postgresql`. Ninguno se adopta ni se retira.

Solo los nodos marcados `[propio]` están inicialmente ausentes y pasan a ser
propios después de acreditar preausencia, crearlos sin `--parents` y registrar
su identidad:

```text
autorizacion_atestada_v3/                                      [basal]
├── migraciones/                                               [basal]
│   └── 000007_componentes/                                    [basal]
│       ├── 080_consumidor_nominal.sql                          [propio]
│       └── __h0b/                                              [propio]
│           ├── sin-r0.sql                                      [propio]
│           ├── nominal/                                        [propio]
│           │   └── ensayo.sql                                  [propio]
│           └── error/                                          [propio]
│               └── ensayo.sql                                  [propio]
└── pruebas_sql/                                                [propio]
    └── 000007_componentes/                                    [propio]
        └── 080_consumidor_nominal.sql                          [propio]
```

La ubicación `migraciones/000007_componentes/__h0b` sustituye expresamente la
ruta de wrappers de la enmienda anterior. Mantiene F04 bajo un padre basal
conocido y separa sus recursos de los dos padres propios de la rama T.

La topología de este arnés es cerrada. Si una evolución del inventario convierte
`pruebas_sql` o su `000007_componentes` en basales, la prueba devuelve 65 y
exige revisar el contrato; no intenta adoptar dinámicamente una forma distinta.
Se elimina el protocolo provisional `B/O0/O1`.

## Propiedad de F02, F03 y F04

F02 recibe el conjunto literal de los tres wrappers. Antes de retirar ninguno,
cada ruta debe ser una de estas dos cosas:

1. el fichero exacto cuya identidad física y contenido se registraron; o
2. una ausencia exacta, incluida la ausencia de enlace simbólico.

Cada miembro literal conserva uno de cuatro estados: `nunca_creado`,
`registrado_presente`, `retirado_para_sustitucion` o
`retirado_por_la_accion`. Una mezcla de estos estados es esperable ante
inyección y sustituye expresamente la regla anterior que trataba toda mezcla
presente/ausente como error. `nunca_creado` exige ausencia exacta;
`registrado_presente` exige la identidad registrada; y los dos estados de
retirada exigen ausencia exacta. `retirado_para_sustitucion` solo puede volver
a `registrado_presente` después de registrar la nueva identidad;
`retirado_por_la_accion` permite la segunda ejecución idempotente. La
desaparición externa de un miembro `registrado_presente`, una ruta presente sin
identidad o una identidad discrepante fuerza 65 antes de retirar nada. Este
ledger se aplica por igual a F02, F03 y F04.

F03 valida y retira exclusivamente M080, T080 y los dos directorios propios de
la rama T. Reacredita el padre basal de M080, retira hojas antes que padres y
nunca retira `migraciones/000007_componentes`. Si no existe una hoja registrada,
debe acreditar su ausencia exacta; no puede devolver éxito solo porque la lista
de identidades esté vacía.

F04 valida y retira exclusivamente `__h0b` y sus directorios propios
`nominal/error`. Su primera pasada exige el conjunto exacto esperado para el
estado alcanzado, identidades exactas y directorios hoja vacíos. Reacredita
después el padre basal de migraciones.

Las tres acciones conservan el registro después de una primera retirada para
que una segunda ejecución pueda acreditar que todo el grupo está ausente y
devolver éxito. Si una revalidación de segunda pasada falla después de retirar
un miembro, registran cuáles se retiraron y detienen el grupo restante; el
contenedor exterior se elimina después por su identidad exacta.

## Orden de identidad, sustitución y seams

Para N03 y N05 el orden obligatorio es:

```text
preausencia exacta -> copia -> stat/huella/enlaces -> identidad registrada
                   -> seam de copia -> cotejo de contenido -> seam de huella
```

E03 y E05 reutilizan las rutas M080 y T080 del tramo nominal. No sobrescriben
un fichero vivo. Primero acreditan y revalidan la identidad nominal, retiran
esa versión exacta, acreditan postausencia y marcan la transición. Después
copian la versión de error, registran inmediatamente su nueva identidad y solo
entonces activan E03/E05; E04/E06 cotejan el nuevo contenido. Un fallo entre la
retirada nominal y la copia conserva un estado de transición cuya única forma
válida es la ausencia exacta.

N07 y E07 operan sobre wrappers distintos y aplican preausencia exacta, copia,
registro inmediato de identidad y seam. N08/E08 cotejan su contenido.

Un fallo entre cualquier copia y su registro no se simula mediante N03/E03,
N05/E05 o N07/E07. Pertenece a los mutantes internos de creación incompleta y
fuerza 65 con retirada exterior del contenedor. Así los estados 79 del oráculo
nunca dejan un objeto creado pero desconocido para F02/F03/F04.

## Reparto de política y mecanismo

El runner conserva y ejecuta:

- parser, selector, catálogo, oráculo y causal 0/64/65/79;
- secuencia de los tramos sin R0, nominal y error;
- invocación y decisión del finalizador F01..F15;
- construcción, análisis y validación de `RESULTADO`;
- decisión de detener la matriz y estado final del caso.

El adaptador expone primitivas mecánicas; no llama a `finalizar_h0b_f0`, no
decide la causal y no emite ni interpreta `RESULTADO`. Conserva las identidades,
materializa y ejecuta wrappers, administra contenedor/temporal/grupo de procesos
y ejecuta las operaciones exactas que el runner ordena.

Esta devolución de política al runner no añade un proceso ni un quinto
auxiliar. La captura privada sigue teniendo exactamente cuatro fuentes. Un
quinto auxiliar queda prohibido en este corte; solo puede estudiarse mediante
otra decisión si el adaptador legible supera 580 o no puede aislar su estado.

## Supervisión exterior que debe completarse

La identidad del caso se arma antes de su primera comprobación o efecto. Desde
ese punto todos los retornos y señales convergen en una sola primitiva
idempotente. La implementación debe:

1. guardar el estado previo de `monitor` y del atributo de exportación de
   `SHELLOPTS`, exigir tabla de trabajos inicialmente vacía y restaurarlos;
2. diferir `INT`/`TERM` durante cada transición crítica y drenar la señal
   pendiente hacia el epílogo;
3. precrear el contenedor con cliente Docker paterno acotado y acreditar CID,
   cidfile físico, nombre, etiqueta, imagen, estado y readiness en 60 segundos
   totales antes del ticket;
4. armar un trabajo provisional antes del fork y recuperar por cardinalidad
   exacta cero/uno si una señal llega antes de copiar `$!`;
5. acreditar estado `T`, `PID=PGID`, PPID directo y tiempo de inicio antes de
   `CONT`, y revalidarlos al señalizar y esperar;
6. aplicar 180 segundos absolutos desde `CONT`, extinguir el PGID completo,
   esperar con `wait -f` solo al Bash directo y acreditar cero miembros;
7. retirar el contenedor solo tras reconciliar su identidad y esperar al daemon;
8. retirar al final el temporal por identidad exacta y acreditar postausencia;
9. conservar la identidad armada y devolver incidente 65 si el daemon impide
   conocer la postausencia, sin lanzar el caso siguiente.

La [decisión de semántica INT/TERM de C4b-1](decision_f0_h0b_c4b1_semantica_senales_2026-08-02.md)
precisa el punto 2: se enclava la primera señal entregada y observada al
iniciarse un manejador, no el primer `kill`. Una ráfaga anterior a ese marco
admite exactamente uno de `{130, 143}`, con una sola cancelación, cero efectos
o trabajos nuevos y limpieza convergente.

Un cidfile ausente, malformado o sustituido no permite borrar por nombre ni
desarmar la identidad. Una ruta temporal ya ausente o un contenedor ya retirado
son éxito idempotente solo cuando el resto de evidencias concuerda.

## Presupuesto reproducible corregido

El ledger parte del adaptador observado de 361 líneas:

| Trabajo neto pendiente | Mínimo | Máximo |
| --- | ---: | ---: |
| Literales, orden de identidad, imagen y readiness | +4 | +10 |
| Señales y restauración de `monitor`/`SHELLOPTS` | +15 | +24 |
| Trabajo provisional y PID/PGID/PPID/inicio | +12 | +20 |
| Plazos, `wait -f` y extinción | +8 | +16 |
| Epílogo idempotente, CID, temporal y daemon | +14 | +24 |
| F02/F03/F04, dos pasadas, ausencia y ledger | +16 | +30 |
| Mutantes interiores y exteriores | +45 | +70 |
| Descompresión de líneas opacas | +12 | +20 |
| Unificar espera, plazo y cancelación | -9 | -5 |
| Factorizar reconciliación exacta del contenedor | -6 | -3 |
| Factorizar validaciones físicas interiores | -7 | -3 |
| **Proyección final del adaptador** | **465** | **564** |

La estimación independiente conjunta quedó en 450..571. Se adopta el sobre
conservador común:

| Fichero | Objetivo | Límite local | DEC-051 |
| --- | ---: | ---: | ---: |
| Runner | 760 o menos | 775 | 800 |
| Auxiliar H0b | 580 exactas | 580 | 800 |
| Adaptador privado | 520 o menos | 580 | 800 |

El adaptador se somete a un checkpoint obligatorio al alcanzar 560 líneas. Las
veinte restantes solo permiten cerrar una desviación medida, no añadir alcance.
Superar 580 exige parar y decidir de nuevo. Las 220 líneas hasta DEC-051 no son
reserva. No se permite agrupar sentencias, retirar controles, mensajes o
mutantes, ni trasladar política al adaptador para cumplir el límite.

## Minitareas confirmables

La implementación se reanuda en cortes pequeños sobre la rama funcional:

1. **C4a — frontera y topología:** devolver causal/finalizador al runner,
   aplicar la topología cerrada, registrar identidades antes de seams, arreglar
   ausencias y dejar Bash, ShellCheck, huellas y H0 nominal verdes;
2. **C4b — supervisión exterior:** señales, trabajo provisional, plazos,
   epílogo idempotente y mutantes de proceso/Docker, con residuos cero;
3. **C4c — grupos interiores:** F02/F03/F04 de dos pasadas, ledger e
   idempotencia, más mutantes de identidad/topología;
4. **C4d — evidencia completa:** rechazos 64, 39 procesos, H0 tres veces, A1,
   C1, mutante A1, Gitleaks y dos revisiones independientes.

### Descomposición obligatoria de C4b

La inspección posterior a C4a confirma que C4b comparte estado entre runner y
adaptador y no puede paralelizarse sin carreras de edición ni completarse como
un único commit revisable. Se divide en tres minitareas secuenciales; ninguna
se presenta por separado como cierre de C4b:

1. **C4b-1 — régimen shell y señal diferida:** conservar y restaurar
   `monitor` y el atributo de exportación de `SHELLOPTS`, exigir cero trabajos
   iniciales, diferir `INT`/`TERM` durante la sección crítica y unificar la
   espera terminal del cliente y del reloj;
2. **C4b-2 — hijo directo y grupo:** armar el trabajo provisional antes del
   fork, resolver la cardinalidad cero/uno, acreditar y revalidar PID, PGID,
   PPID y tiempo de inicio, aplicar el plazo absoluto y demostrar la extinción
   completa del grupo;
3. **C4b-3 — Docker, temporal y epílogo único:** acotar `run`, readiness y
   `rm`, reconciliar CID, nombre, etiqueta, imagen y estado, resolver las
   terminaciones tardías y dejar un solo propietario idempotente de la
   limpieza exterior.

Cada minitarea modifica exclusivamente el runner y el adaptador, actualiza la
huella literal del segundo y conserva byte a byte H0b, D2c, D2d y el
capturador. El acta agregada se añade solo después del verde de las tres.

Se fija un checkpoint anticipado de 540 líneas para el adaptador al cerrar
C4b, de modo que C4c conserve cuarenta líneas hasta el límite 580. El runner
debe quedar en 770 o menos al cerrar C4b y nunca superar 775. Si el adaptador
proyecta más de 540 o no puede sostener una sola autoridad de limpieza, se
detiene la implementación y se somete a revisión una nueva decisión de captura
quinta que separe supervisor exterior y ciclo interior. No se comprime código
ni se modifica D2d para evitar esa decisión.

Un corte solo puede confirmarse si sus pruebas son verdes y no deja una ruta
que la rama integradora pueda interpretar como cierre funcional. Ningún
checkpoint intermedio aumenta métricas ni se publica como producción.

## Puertas finales

Además de las doce puertas de la enmienda anterior, el commit funcional debe
acreditar:

- SHA literal actual del adaptador y captura canónica exacta de cuatro;
- cero SC2016 sin justificar y cero líneas opacas creadas para caber;
- topología H0 exacta y fallo 65 si los padres propios aparecen como basales;
- orden de identidad anterior a cada seam de copia;
- segunda ejecución idempotente de F02/F03/F04;
- señal en fork -> `$!`, `SHELLOPTS` exportado, `wait -f` interrumpido, cliente
  Docker no terminante y daemon no reconciliable;
- rechazos 64 con traza no vacía, exactamente un `+ exit 64` final y cero
  identidad, temporal, Docker o `psql`;
- runner de 775 o menos, H0b en 580 exactas y adaptador de 580 o menos;
- write-set material de tres ficheros e invariancia byte a byte de D2c, D2d y
  capturador.

Producción, H0b, C2 y F0 permanecen en `NO-GO`. Esta enmienda no aumenta F0
`10/23`, O4-05 `3/5`, Contratación temporal `24/46` ni Bolsa productiva `1/14`.
