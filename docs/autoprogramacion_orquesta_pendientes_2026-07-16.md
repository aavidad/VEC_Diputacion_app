# Autoprogramacion Orquesta pendientes 2026-07-16

## Alcance

- Origen: [auditoria de diseno y seguridad 2026-07-16](portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md)
  y decisiones DEC-050 a DEC-053 adoptadas en el
  [registro de decisiones](portal_vec/registro_decisiones.md).
- Este backlog sigue el contrato del scanner documental: cada pendiente Txx
  lleva origen, estado, area hexagonal, accion y evidencia. Ninguna entrada
  abre codigo desde texto narrativo: todas remiten a una DEC adoptada o a un
  hallazgo verificado con comandos reproducibles.
- Orden recomendado: T01 y T06 son documentales y desbloquean al resto;
  T02 tiene evidencia de fallo real en CI y precede a cualquier ampliacion
  de `internal/vec/ports`.
- Prioridad de negocio (responsable, 2026-07-17): **Bolsa primero**. El
  objetivo inmediato es una Bolsa operable de punta a punta: levantar el
  NO-GO de la migracion V2, conectores productivos de la saga de firma,
  porte del ranking/desempate del heredado (hallazgo 4 de la brecha) y su
  API privada con UI conectada. Los demas frentes no se detienen, pero
  ceden el paso ante conflicto de recursos o de carril.
- **Orden de ataque vigente (direccion, 18/07/2026). Sustituye al del 17/07.**
  El criterio ha cambiado: la prioridad es **terminar la funcionalidad de la
  aplicacion**, no pulir su estructura interna. El refactor estructural se
  salda en v2.

  1. **T21 + T20 juntas**: el perfil `desarrollo` con credenciales propias y
     el adaptador Go PostgreSQL de borradores. Van primero y en paralelo
     porque entre las dos convierten la primera vertical en algo que funciona
     de verdad de punta a punta. Nada de esto espera a Sistemas.
  2. **T12** (durabilidad probatoria) y **T13** (registro de accesos con
     finalidad). Van a continuacion porque **no son mejoras: la EIPD los
     declara condicion previa para el piloto con datos reales**. Sin ellos la
     aplicacion solo puede demostrarse con datos sinteticos.

  3. **T17** (importador gobernado de Convoca): es lo que permite que la
     aplicacion coma datos reales de bolsas.
  4. **T07** (coherencia frontend-API verificada en ejecucion) y el resto de
     **T03**.
  5. **T14 a T16** (conservacion, derechos RGPD, cifrado en reposo), del
     paquete `docs/cumplimiento/`, redactadas para programarse sin esperar a
     las mesas: los plazos y politicas que solo las mesas pueden fijar entran
     como configuracion, no como codigo.
  6. **T18** (Formacion) y **T19** (adaptador TSA), este ultimo cuando haya
     decision de proveedor.

  Criterio general que corrige el patron observado el 18/07: **cuando un
  camino a lo operativo este bloqueado, no se sustituye por mas contrato de
  nucleo probado en memoria.** Construir contratos verificables mientras la
  infraestructura falta es razonable, pero produce paquetes verdes sin flujo
  utilizable. Ante un bloqueo, la conducta correcta es informar del bloqueo y
  pasar a la siguiente tarea desbloqueada de la cola, no profundizar en la
  bloqueada. Y si una tarea necesaria no esta en esta cola, el agente debe
  senalarlo a direccion en vez de asumir que no es prioritaria: T20 existio
  como trabajo propuesto en un documento de brecha durante dias sin que nadie
  lo encolara, y esa omision fue de direccion.

  **T02 queda aparcado para v2** (ver su nota de aparcamiento): no se abren
  mas tandas, pero sus reglas de contencion siguen siendo obligatorias.

  **S-03 cambia de naturaleza el 18/07**: sigue siendo cierto que no hay
  **produccion** hasta el visto bueno de Sistemas, pero ya **no bloquea el
  desarrollo**. La identidad, el KMS y el TLS se componen contra proveedores
  propios bajo el perfil `desarrollo` de T21, y los autoritativos entran
  despues por configuracion. Lo que se aparca es el despliegue real, no el
  trabajo. Sigue vigente que nada debe disenarse de forma que estorbe la
  llegada de la identidad definitiva.

  Nota de criterio para el agente: aparcar T02 **no** autoriza a bajar el
  liston de calidad en lo que si se programa. Sigue exigido el mismo rigor
  de pruebas, `go vet`, carrera donde toque y evidencia medida en cada
  entrega. Lo que se aparca es un refactor estructural, no la disciplina.

## Pendientes Txx

### T01 — Analisis de brecha del nucleo heredado (DEC-050)

- `origen`: DEC-050 del registro de decisiones.
- `estado`: completado 2026-07-16 por direccion: [analisis de brecha](portal_vec/brecha_nucleo_heredado_bolsa.md) fusionado; la decision de alcance quedo adoptada como DEC-079 (sede existente, sin autobaremo declarativo): el porte del legado puede abrirse conforme a ella.
- `area_hexagonal`: docs primero; nucleo y composicion despues.
- `accion`: documentar el inventario de capacidades de `internal/candidate`
  y el analisis de brecha contra `internal/modules/bolsa`, dejando en el
  registro que se porta, que ya esta cubierto y que se descarta. Sin ese
  documento no se abre codigo de porte ni de borrado.
- `evidencia`: `go list -f` verifica que solo `internal/app/bootstrap`
  importa `internal/candidate`; la API heredada solo se monta en `fake`.

### T02 — Extraer la logica canonica de `internal/vec/ports` (H-01)

- `origen`: auditoria H-01 y fallo real de CI.
- `estado`: **APARCADO PARA v2 por decision de direccion el 18/07/2026. No
  abrir mas tandas de T02.** La limpieza se pospone; la contencion NO. Ver
  la nota de aparcamiento mas abajo, que es de cumplimiento obligatorio.

  Historico: el 17/07 se detecto la desviacion de +6.563 lineas en
  `internal/vec/ports` y +10.336 lineas con 45 ficheros nuevos en
  `internal/modules/bolsa/ports` desde el ancla de la auditoria. Tres tandas
  (`2cf445f..65acc4c`) redujeron 1.818 lineas netas y dejaron
  `internal/vec/canonico` con cuatro subpaquetes utiles, que se conservan.
- `area_hexagonal`: puerto hacia nucleo/subpaquetes.
- `accion`: **ninguna hasta v2.** Cuando se retome, el troceo por capacidad
  (autorizacion, documental, almacen, auditoria) mueve derivaciones canonicas
  a subpaquetes; `ports` conserva interfaces y tipos de intercambio.

#### Nota de aparcamiento T02 — 18/07/2026 (vigente, no opcional)

Motivo del aparcamiento, con los numeros que lo sostienen: tres tandas
movieron 1.818 lineas netas, y el tercer corte —el mas caro, con carrera
`-race` de 101 s y pruebas de equivalencia byte a byte— movio 34 lineas.
Los tres ficheros que de verdad pesan siguen intactos
(`ejecuciones_documentales_v3.go` 3.975, `idempotencia_semantica_baremacion.go`
3.864, `baremacion.go` 2.055). A ese ritmo el cierre de T02 esta a cientos de
tandas, y compite por presupuesto con trabajo que desbloquea el piloto. La
prioridad pasa a terminar funcionalidad; la deuda de `ports` se asume
conscientemente y se salda en v2.

Lo que **sigue vigente pese al aparcamiento**:

1. **Ningun fichero nuevo en `*/ports`.** Los contratos nuevos nacen en
   subpaquetes de dominio (`reglasbaremo`, `calculoexperiencia`,
   `internal/vec/canonico` son el patron correcto ya usado); `ports` solo
   reexporta interfaces. Esta regla congela la deuda donde esta; sin ella la
   v2 hereda un problema peor que el actual.
2. **DEC-051 sin excepciones**: ningun fichero supera el tope, y el control
   ratcheado de `scripts/tamano_ficheros_base.txt` solo baja.
3. **`internal/vec/canonico` se conserva y se usa.** Lo ya extraido no se
   revierte, y si una capacidad nueva encaja de forma natural en un
   subpaquete canonico existente, va ahi y no al puerto.

Correccion de tecnica para cuando se retome en v2, porque la tecnica elegida
fue la principal causa del coste: la consolidacion primitiva a primitiva con
prueba de equivalencia byte a byte **no es el metodo por defecto**. Si un
fichero es autocontenido, se mueve entero (cambio de paquete) y la
equivalencia la demuestra el compilador. La prueba byte a byte se reserva a
funciones que participan en preimagenes de firma, huella o serializacion
canonica, donde un cambio de un solo byte invalidaria evidencia. Aplicada
asi, estaba bien empleada en `65acc4c`; como metodo general, no.
- `evidencia`: run de CI 29462846251 fallo por timeout de `go test -race`
  (600 s) en `internal/vec/ports`; el paquete tiene 19.733 lineas de fuente
  en el ancla de la auditoria. La linea base inmediatamente anterior a T02,
  `8ac2aa2`, ya habia crecido hasta 23.525 lineas de produccion y conservaba
  un fichero de 4.122 lineas (`ejecuciones_documentales_v3.go`). El cierre
  medido de la primera tanda queda a continuacion.

#### Cierre medido de la primera tanda T02 — 18/07/2026

El corte comprende `2cf445f..5ed064e`, tomando `8ac2aa2` (`2cf445f^`) como
linea base previa. Es un cierre de tanda, no el cierre de T02.

| Commit | Resultado incorporado |
|---|---|
| `2cf445f` | Extrae a `canonico/pagos` validacion, auditoria, entrega, operacion remota y derivaciones; `ports/pagos.go` conserva el contrato y las reexportaciones compatibles. |
| `35cadc5` | Funda `canonico/almacen` con capacidades, requisitos, objetos, instrucciones, resultados, recibos y seudonimizacion. |
| `98209b7` | Extrae primitivas, intercambio, atestaciones y despacho documental V3 a `canonico/documental`. |
| `842aaf9` | Funda `canonico/recibomaterial` con perfil, plan, instantanea, recibo, atestacion y reglas primitivas. |
| `02bf26b` | Extrae el contexto canonico de almacenamiento y liga su validacion a cargas documentales. |
| `e49ff6b` | Extrae capacidades del recibo material mediante proyecciones explicitas y copias defensivas, sin convertir capacidades nominales en alias de tipo. |
| `bbff751` | Extrae sellos de evidencia, sobre de prueba y reconciliacion documental V3. |
| `9bbe338` | Endurece la nominalidad criptografica: la forma de una atestacion no se presenta como autoridad y los DTO internos sensibles quedan redactados, con JSON, texto y binario denegados. |
| `5ed064e` | Sustituye tres expresiones regulares calientes por escaneres ASCII con equivalencia exhaustiva frente a las expresiones anteriores. |

##### Tamano antes y despues

Los conteos son lineas fisicas de ficheros Go versionados. Se separan
produccion y pruebas para no esconder el coste de la evidencia nueva.

| Superficie | `8ac2aa2` | `5ed064e` | Variacion |
|---|---:|---:|---:|
| `internal/vec/ports`, produccion | 42 ficheros / 23.525 lineas | 42 / 21.704 | -1.821 lineas (-7,7 %) |
| `internal/vec/ports`, pruebas | 30 / 14.270 | 30 / 14.270 | sin cambio |
| `internal/vec/ports`, total | 72 / 37.795 | 72 / 35.974 | -1.821 lineas (-4,8 %) |
| `internal/vec/canonico`, produccion | 0 / 0 | 30 / 4.373 | +30 / +4.373 |
| `internal/vec/canonico`, pruebas | 0 / 0 | 12 / 1.664 | +12 / +1.664 |
| `ports` + `canonico`, produccion | 42 / 23.525 | 72 / 26.077 | +30 / +2.552 |
| `ports` + `canonico`, total | 72 / 37.795 | 114 / 42.011 | +42 / +4.216 |
| `internal/modules/bolsa/ports`, produccion | 34 / 14.424 | 34 / 14.424 | sin cambio |
| `internal/modules/bolsa/ports`, pruebas | 29 / 12.342 | 29 / 12.342 | sin cambio |

El crecimiento de `canonico` no se interpreta como ahorro global: contiene la
logica extraida, proyecciones de compatibilidad y 1.664 lineas de pruebas
nuevas. La metrica de arquitectura relevante en esta tanda es que la
derivacion deja de vivir en el paquete de contratos.

| Fichero de produccion de `internal/vec/ports` | Antes | Actual | Delta |
|---|---:|---:|---:|
| `pagos.go` | 1.571 | 719 | -852 |
| `almacen_objetos.go` | 1.716 | 1.206 | -510 |
| `recibo_escritura_objeto_material_v2.go` | 1.754 | 1.474 | -280 |
| `ejecuciones_documentales_v3.go` | 4.122 | 3.975 | -147 |
| `autorizacion_almacen.go` | 797 | 743 | -54 |
| `ejecucion_componentes_documentales_atestada.go` | 1.160 | 1.154 | -6 |
| `cargas_documentales.go` | 757 | 779 | +22 |
| `formatos_documentales_gobernados.go` | 745 | 751 | +6 |

##### Medidas reproducibles

La medida completa compara la linea base pre-T02 `8ac2aa2` con `5ed064e`:

```bash
/usr/bin/time -f 'elapsed=%e user=%U sys=%S maxrss=%M' \
  go test -race -count=1 ./internal/vec/ports
```

| Carrera completa de `internal/vec/ports` | Base `8ac2aa2` | Despues `5ed064e` | Mejora |
|---|---:|---:|---:|
| Tiempo comunicado por `go test` | `582.884 s` | `485.223 s` | `-97.661 s` (-16,8 %) |
| Tiempo de pared | `586.29 s` | `485.51 s` | `-100.78 s` (-17,2 %) |
| CPU usuario / sistema | no conservado | `1042.88 s` / `10.43 s` | no comparable |
| RSS maximo | `365664 KiB` | `184860 KiB` | `-180804 KiB` (-49,4 %) |

El escaner ASCII se midio ademas con un A/B causal aislado: worktree
`bbff751` frente a `5ed064e`. El commit intermedio `9bbe338` solo endurece
`recibomaterial` y no esta en el camino ejecutado por la prueba focal.

```bash
/usr/bin/time -f 'elapsed=%e user=%U sys=%S maxrss=%M' \
  go test -count=1 ./internal/vec/ports \
  -run '^TestPreparacionAlmacenV4DetectaCadaCampoDeContextoCapacidadesYPolitica$'

/usr/bin/time -f 'elapsed=%e user=%U sys=%S maxrss=%M' \
  go test -race -count=1 ./internal/vec/ports \
  -run '^TestPreparacionAlmacenV4DetectaCadaCampoDeContextoCapacidadesYPolitica$'
```

| Prueba focal | Base `bbff751` | Despues `5ed064e` | Mejora |
|---|---:|---:|---:|
| Normal, tiempo de paquete | `5.912 s` | `3.915 s` | -33,8 % |
| Normal, pared | `6.12 s` | `4.12 s` | -32,7 % |
| Normal, CPU usuario / sistema | no conservado | `4.63 s` / `0.28 s` | no comparable |
| Normal, RSS maximo | `144800 KiB` | `144812 KiB` | sin variacion material |
| Race, tiempo de paquete | `85.784 s` | `50.100 s` | -41,6 % |
| Race, pared | `90.14 s` | `50.38 s` | -44,1 % |
| Race, CPU usuario / sistema | `93.88 s` / `1.19 s` | `51.35 s` / `0.53 s` | -45,3 % / -55,5 % |
| Race, RSS maximo | `340300 KiB` | `161552 KiB` | `-178748 KiB` (-52,5 %) |

La carrera completa local vuelve a terminar por debajo del timeout de 600 s
que origino T02, pero `485.51 s` todavia consume el 80,9 % de ese margen. La
medida corrige localmente el sintoma observado; no sustituye una nueva
ejecucion verde de CI ni justifica cerrar el riesgo de rendimiento o T02.

##### Decisiones de seguridad conservadas

- Las capacidades nominales con campos privados permanecen en `ports`. No se
  sustituyen por alias de tipo reflectivos ni se reconstruye autoridad con
  `reflect`; su paso al nucleo usa conversores explicitos hacia proyecciones
  `Datos*`. Los alias se reservan a vocabulario o datos pasivos cuya
  construccion no concede autoridad.
- `DatosSolicitudAtestacion`, `DatosAtestacion`, `DatosPerfilPublicado` y
  `DatosResultadoReferencia` redactan `String`, `GoString`, `Format` y
  `LogValue`, y deniegan serializacion y deserializacion JSON, texto y binaria.
  No son DTO HTTP ni prueba criptografica; solo transportan forma nominal hacia
  una autoridad homologada y una relectura durable.
- Los mensajes, codigos y canonicos cruzan las fronteras con copias defensivas;
  las huellas sensibles se comparan en tiempo constante. La extraccion no
  introduce un conector productivo ni una autoridad por defecto.
- Los escaneres ASCII mantienen exactamente los lenguajes previos, incluidos
  limites, UTF-8 no ASCII y bytes invalidos. Las pruebas comparan las funciones
  nuevas con las expresiones regulares originales en fronteras, todos los
  octetos, mutaciones completas de DNI/NIE, 100.000 entradas deterministas y
  semillas de fuzzing.

##### Siguiente corte incremental T02 — 18/07/2026

El commit `af1a79c` extrae la canonizacion y la comprobacion de la preimagen de
la entrada neutral documental V1 desde el puerto a
`internal/vec/canonico/documental/entrada_neutral_v1.go`. El fichero canonico
tiene 94 lineas de produccion y su prueba dedicada, 113. Como resultado,
`materializacion_documental_v4.go` baja de 1.590 a 1.532 lineas.

Los limites del codec permanecen privados en el canonico. El puerto conserva
sin cambios su superficie publica, sus tipos, sus errores observables y la
preimagen historica; solo traduce el resultado cerrado del canonico a su error
nominal. El corte supero las pruebas normales y con detector de carreras de
los paquetes afectados, `go vet` y el control ratcheado de tamanos.

La carrera completa de `internal/vec/ports` con `go test -race -count=1`
comunico `485.657 s`. No acredita una mejora del coste global frente a los
`485.223 s` del corte anterior, pero mantiene la linea base local bajo el
timeout de 600 s. Se conserva por ello como medida de no regresion, no como
motivo para declarar resuelto el riesgo de rendimiento ni para cerrar T02.

##### Tercer corte incremental T02 — 18/07/2026

El commit `65acc4c` elimina de `materializacion_documental_v4.go` las copias
locales de serializacion, huella y validacion de referencias, y hace que el
puerto consuma `SerializarCamposV3`, `HuellaBytesSHA256` y
`ReferenciaEjecucionV3Valida` desde `internal/vec/canonico/documental`. La
comprobacion de forma canonica y no reutilizacion de huellas se incorpora al
mismo nucleo como la nueva `HuellasSHA256Distintas`. El fichero del puerto
baja de 1.532 a 1.498 lineas, 34 menos.

La prueba explicita fija que las colecciones `nil` y vacia producen una
preimagen `nil`; conserva byte a byte Unicode valido, UTF-8 malformado, NUL y
delimitadores; comprueba la SHA-256 de la preimagen vacia; y cubre forma,
cardinalidad y duplicados de huellas. Desde la raiz quedaron verdes las
pruebas normales de `canonico/documental` y `ports`, la carrera completa del
canonico y la carrera focal del puerto (`101.679 s`), ademas de `go vet`, el
control ratcheado de tamanos y el control de diff.

Una revision GO independiente confirmo equivalencia byte a byte con las
implementaciones retiradas y ausencia de cambios en API, conjuntos de metodos
y errores observables.

##### Trabajo restante para cerrar T02

1. Continuar el troceo por capacidad de
   `ejecuciones_documentales_v3.go` (3.975 lineas),
   `materializacion_documental_v4.go` (1.498),
   `recibo_escritura_objeto_material_v2.go` (1.474),
   `almacen_objetos.go` (1.206) y
   `ejecucion_componentes_documentales_atestada.go` (1.154), sin ampliar sus
   superficies publicas ni usar alias para capacidades nominales.
2. Inventariar y extraer la logica de `internal/modules/bolsa/ports`: esta
   tanda no modifica sus 14.424 lineas de produccion ni sus 12.342 de pruebas.
3. Reducir de nuevo el coste de la carrera completa y confirmar el resultado
   en CI: aunque la medida local ya no supera 600 s, el margen del 19,1 % es
   insuficiente frente a variacion y crecimiento futuro. Mantener medidas A/B
   con el mismo comando, revision exacta y estado de cache documentado.
4. Mantener congelada la creacion de ficheros en ambos `*/ports` y ratchear la
   base de tamanos solo a la baja. T02 se cerrara cuando `ports` sea contrato y
   proyeccion fina, no por el mero hecho de que la prueba quede bajo el timeout.

### T03 — Cableado de modulos fuera de `httpapi` (H-02)

- `origen`: auditoria H-02.
- `estado`: en curso. Primera rebanada terminada el 17/07/2026: la raiz de
  composicion construye e inyecta el servicio de catalogo Personal y
  `httpapi` ya no importa ni decide entre sus adaptadores `file`/`memory`;
  tambien se retiro el servicio Cronos sintetico que solo alimentaba codigo
  muerto. Queda extraer los handlers/contratos de ruta para eliminar los
  imports funcionales restantes de `internal/modules/*`.
- `area_hexagonal`: adaptador hacia composicion.
- `accion`: programar el traslado del montaje de modulos de
  `internal/vec/adapters/httpapi` (`workspace.go`, `cronos.go`,
  `personal_rpt.go`) a `internal/app/bootstrap`; `httpapi` recibe handlers o
  interfaces ya compuestos y pierde todos los imports de
  `internal/modules/*`.
- `evidencia`: `nuevoServicioCatalogoPersonal` vive en
  `internal/app/bootstrap`; `HandlerOptions.PersonalCatalog` recibe el caso
  de uso compuesto. `go list` sigue mostrando imports funcionales de modulo,
  por lo que T03 no se declara cerrado todavia.

### T04 — Frontend en modulos ES con tokens (DEC-052, H-03)

- `origen`: DEC-052 del registro de decisiones.
- `estado`: en_curso, cedido al agente Orquesta el 2026-07-16: el subagente de direccion cayo sin commitear y el agente inicio la extraccion conforme a DEC-052 (`81a54d1` modulo cronos con tests JS, `5135eb4` arranque cerrado). La direccion no relanza subagente en este carril para evitar colision; revisa y sube.
- `area_hexagonal`: adaptador (frontend estatico).
- `accion`: programar la particion de `web/static/app.js` en modulos ES
  nativos por dominio funcional, eliminando en cada modulo migrado los
  estilos inline y colores literales; temas solo como redefinicion de tokens
  bajo atributo del documento. Criterio de terminado por modulo segun
  DEC-052.
- `evidencia`: `app.js` tiene 13.211 lineas, ~300 asignaciones `.style` y 67
  colores literales; `styles.css` ya define 49 tokens.

### T05 — Contrato API por modulo para clientes equivalentes (DEC-053)

- `origen`: DEC-053 del registro de decisiones; Ola 2 del plan de
  implantacion.
- `area_hexagonal`: puerto y adaptador HTTP.
- `estado`: completado 2026-07-16 la parte documental por direccion: [contratos API](portal_vec/contratos_api_modulos.md) fusionado; los endpoints de Ola 2 se abriran contra ese contrato.
- `accion`: documentar el contrato de endpoints por modulo (ruta, metodo,
  envelope, errores, version) y programar los endpoints finos de la Ola 2
  contra ese contrato; la web consume la API como un cliente mas y ningun
  cliente incorpora logica de negocio.
- `evidencia`: el frontend actual solo hace `fetch` real a la API heredada y
  `/api/demo`; las pantallas privadas operan con datos sinteticos locales.

### T06 — Nivel de madurez por modulo en el contrato (H-05)

- `origen`: auditoria H-05.
- `estado`: completado 2026-07-16 por direccion: seccion de niveles de madurez fusionada en el contrato de modulos.
- `area_hexagonal`: docs.
- `accion`: documentar en el
  [contrato de modulos](portal_vec/contrato_modulos_vec.md) el nivel que
  cumple cada modulo (completo, parcial, solo manifiesto) para que el shell
  no asuma capacidades inexistentes.
- `evidencia`: Bolsa tiene las cuatro capas; Cronos y Personal parciales;
  Dietas y Administracion solo manifiesto (arbol de `internal/modules/`).

### T07 — Coherencia frontend-API verificada en ejecucion

- `origen`: inconsistencias detectadas y verificadas en vivo por T05 en
  [contratos API por modulo](portal_vec/contratos_api_modulos.md), seccion
  "Inconsistencias detectadas".
- `estado`: nuevo. No ejecutar hasta fusionar el carril T04 (mismo write_set
  `web/static/**`).
- `area_hexagonal`: adaptador (frontend estatico) y composicion.
- `accion`: programar tres correcciones: (1) `staffHeaders()` y
  `candidateHeaders()` de `app.js` no envian `Authorization: Bearer`, que el
  modo `fake` exige — 401 verificado en `/api/vec/session`; (2) ningun perfil
  `fake` de serie tiene permisos funcionales de Cronos/Dietas/Personal — 403
  verificado con Bearer valido, y `loadPortal()` sin `.catch` por llamada
  deja la carcasa siempre en estado de error contra un despliegue demo
  limpio; (3) `app.js` llama a `GET /api` (solo existe en la API heredada
  fake) cuando el equivalente real de la carcasa es `GET /api/vec`.
- `evidencia`: curls documentados en la seccion de inconsistencias del
  contrato; codigos 401/403 reproducidos contra `cmd/vec-server` en fake.

### T08 — Codigo muerto de workspace en httpapi

- `origen`: hallazgo de lectura de T05.
- `estado`: completado el 17/07/2026. Se conserva la ruta fail-closed y se
  elimino todo el grafo sintetico inalcanzable.
- `area_hexagonal`: adaptador.
- `accion`: cerrada por eliminacion; el futuro workspace interno se
  construira como caso de uso acotado por sujeto, finalidad y campos, no
  reactivando el agregado demo.
- `evidencia`: `workspace.go` contiene solo `handleWorkspace`; no quedan
  simbolos `workspaceSnapshot*`, semillas Cronos ni datos RPT/nomina/dietas.

### T09 — Regresion de tamano en la ola de autorizacion del 17/07

- `origen`: puerta `scripts/comprobar_tamano_ficheros.sh` ejecutada por
  direccion antes del push de la ola COSE/VEC-AD-2.
- `estado`: corregido por direccion el 17/07/2026 (`nucleo: dividir
  autorizacion para cumplir el tope de tamano`): `domain/autorizacion.go`
  quedo en 434 lineas mas `autorizacion_politicas_decision.go` (600), y
  `memory/autorizacion_test.go` en 667 mas `autorizacion_aislamiento_test.go`
  (540). Linea base ratificada a la baja.
- `area_hexagonal`: nucleo y adaptador de memoria.
- `accion` (preventiva, para el agente): ejecutar la puerta de tamanos
  antes de cada commit, no solo `go test`. Dos ficheros crecieron por
  encima del tope duro (1024 y 1190 lineas frente a bases de 933 y 860)
  a lo largo de 24 commits sin que ninguna puerta local lo detectara.
- `evidencia`: salida de la puerta el 17/07 a las 11:14; el tope duro es
  800 (DEC-051) y la linea base solo puede menguar.

### T10 — Estilo de mensajes de commit desviado

- `origen`: revision de direccion del 17/07 por la tarde.
- `estado`: nuevo (correccion de conducta, no de codigo; el historico no se reescribe).
- `area_hexagonal`: proceso.
- `accion`: volver al estilo del repositorio `area: verbo en infinitivo`.
  Los prefijos convencionales estan prohibidos por la directriz 5 de la
  auditoria, incluida su variante con parentesis: `fix(web):`, `test(web):`,
  `test(seguridad):`.
- `evidencia`: commits `8e95568`, `1448bc4`, `65d4bee`, `28721f2` del
  17/07/2026 usan prefijos `fix()`/`test()`.

### T11 — Retirar cookies del portal interno conforme a DEC-053

- `origen`: DEC-053 del registro de decisiones ("la web consume la API como
  un cliente mas"; sin cookies de sesion, para que un cliente de escritorio
  sea equivalente byte a byte). El cliente nuevo del portal la contradice.
- `estado`: completado el 17/07/2026 en `1a32b1c`. La regresion queda
  protegida por pruebas del cliente y del contrato de superficies.
- `area_hexagonal`: adaptador (frontend) y composicion.
- `accion`: el cliente del portal interno no debe usar
  `credentials: "same-origin"` ni asumir sesion de cookie: la autenticacion
  viaja en cabecera `Authorization` (token de sesion emitido por el
  adaptador de identidad; para tecnicos, derivado de mTLS+Kerberos segun
  DEC-053), y el cliente web usa `credentials: "omit"`. Bajo ese contrato web
  no hay credencial ambiental que el navegador adjunte entre sitios. La app de
  administracion sera de escritorio y no tiene contexto web entre sitios: la
  API no puede exigir nada que un cliente no navegador no pueda enviar. Si un
  futuro navegador negociase Kerberos/SPNEGO o mTLS automaticamente, habria que
  reevaluar CSRF y origen antes de habilitarlo.
- `evidencia`: `web/static/portal-empleado/portal.js`,
  `web/static/bolsa/bolsa.js` y `web/static/app.js` usan
  `credentials: "omit"`; `web/static/app.seguridad.test.mjs` impide
  reintroducir cookies, credenciales ambientales o identidad persistida en
  almacenamiento web. El contrato de superficies ya no contiene
  configuracion de cookies y el proxy local elimina cualquier cabecera
  `Cookie` entrante.

### T12 — Durabilidad probatoria productiva de Bolsa

- `origen`: prioridad de negocio "Bolsa primero" y revision de direccion del
  17/07: la semantica durable (saga de firma, auditoria encadenada,
  expediente probatorio) esta probada solo en memoria.
- `estado`: nuevo.
- `area_hexagonal`: adaptadores de persistencia y composicion.
- `accion`: llevar a PostgreSQL productivo, por este orden, la auditoria
  encadenada, el registro de decisiones de autorizacion ya migrado y los
  puntos de control de la saga de firma; incluir copias de seguridad
  ensayadas y prueba de recuperacion tras reinicio completo. Sin esto un
  incidente no pierde seguridad pero si trazabilidad.
- `evidencia`: adaptadores en memoria en `internal/modules/bolsa` y
  `internal/vec`; el flujo durable documenta sus conectores pendientes.

### T20 — Adaptador Go PostgreSQL de borradores y diario de convocatorias

- `origen`: direccion, 18/07/2026. Correccion de un descuido de la propia
  direccion: este trabajo estaba descrito como «Trabajo propuesto, punto 1»
  dentro de [brecha funcional](portal_vec/brecha_funcional_bolsa_2026-07-17.md)
  pero **nunca se convirtio en tarea numerada**, por lo que la cola jamas
  apunto a el y el agente no tenia como saber que era prioritario.
- `estado`: nuevo. **Cabecera de cola junto a T12 y T13.**
- `area_hexagonal`: adaptador de persistencia.
- `accion`: implementar el adaptador Go PostgreSQL del diario y del agregado
  cifrado de borradores de convocatorias: restaurar la identidad primaria y
  todos sus alias HMAC multigeneracion, y revalidar la atestacion KMS dentro
  de la transaccion de confirmacion. Corregir antes el defecto alto de
  idempotencia multigeneracion que la revision cruzada dejo en NO-GO (ventanas
  de rotacion solapadas tipo `[g3,g2]` y `[g2,g1]`).
- **Por que si se puede hacer ahora**: no depende de Sistemas. El proveedor
  KMS entra **inyectado tras una interfaz**, igual que se decidio para la TSA
  en T19: en pruebas, un KMS simulado determinista; en produccion, el real
  cuando Sistemas lo provea. Construir el adaptador contra la interfaz no
  requiere infraestructura productiva y es exactamente lo que hoy falta para
  que la pantalla de borradores deje de fallar cerrada.
- **Alcance ampliado el 18/07 por decision del responsable**: T20 ya no se
  detiene ante la falta de identidad, KMS o TLS reales. Se compone entero
  contra el perfil `desarrollo` de **T21**, que aporta CA, certificados, KMS
  e identidad propios. La sustitucion por los proveedores autoritativos sera
  un cambio de configuracion, no un rediseno, porque todos entran inyectados
  tras interfaz. La pantalla de borradores debe quedar **funcionando de punta
  a punta** bajo ese perfil.
- **Lo unico que sigue fuera de T20**: publicacion y retirada de
  convocatorias, que tienen la barrera adicional de DEC-091 (aprobacion
  firmada y dependencias como hechos autoritativos preexistentes). Esa
  barrera es de diseno juridico, no de infraestructura, y no la levanta un
  certificado de pega. T20 cubre alta y actualizacion.
- `evidencia`: migraciones `000003` de borradores durables y pruebas SQL ya
  presentes en `deploy/postgresql/bolsa_convocatorias/`; revisiones cruzadas
  en NO-GO del cliente web, del servicio Go y de la migracion SQL.

### T21 — Perfil `desarrollo` con credenciales propias para desbloquear todo

- `origen`: direccion, 18/07/2026, por decision expresa del responsable: no se
  detiene ningun frente esperando a Sistemas. Se generan credenciales propias
  de pega y se sustituyen por las autoritativas cuando esten disponibles.
- `estado`: nuevo. **Habilitador transversal: desbloquea T20 y levanta las
  barreras de composicion. Se hace junto con T20, no despues.**
- `area_hexagonal`: composicion, adaptadores de seguridad y configuracion.
- `accion`: crear un perfil de ejecucion `desarrollo` que componga la vertical
  completa con proveedores propios tras las interfaces ya existentes:
  1. **CA propia y certificados**: script que genere una CA local, certificado
     de servidor y certificados de cliente para mTLS. Extiende el `AuthMode`
     ya existente en `config`; no se crea un mecanismo paralelo.
  2. **KMS de desarrollo**: proveedor respaldado por fichero tras la interfaz
     KMS, con envoltura de clave y atestacion reales en forma pero emitidas
     localmente. Habilita el adaptador de T20 sin esperar a HSM.
  3. **Identidad de alta garantia simulada**: resolutor que entregue un actor
     con la misma forma que dara el certificado/Kerberos real, para que la
     composicion productiva se pueda cablear y probar entera.
  4. **TSA de desarrollo**: sello determinista para pruebas, sujeto a la
     restriccion de marcado del punto 3 de guardarrailes.
  5. **TLS**: certificado autofirmado de la CA propia para el listener interno.
- **Guardarrailes de cumplimiento obligatorio.** El riesgo de este atajo no es
  tecnico sino de trasvase: que material de pega o actos firmados con el
  acaben tratados como autoritativos. Se contiene asi:
  1. **Ningun material criptografico se commitea jamas.** El repositorio es
     **publico**. Claves, certificados y semillas se generan con el script en
     local y van a `.gitignore`. Ni siquiera los de pega: un certificado en el
     historico publico es un hallazgo de seguridad aunque no valga nada.
  2. **El perfil `desarrollo` no puede ser el valor por defecto** y no puede
     seleccionarse desde el perfil productivo. Se replica el patron anti-fuga
     ya usado para los datos sinteticos de demostracion (doble llave y prueba
     que impide su activacion accidental), que es el precedente correcto ya
     existente en el codigo.
  3. **Todo acto producido bajo el perfil `desarrollo` queda marcado de forma
     estructural e imborrable como no autoritativo**, en el propio dato
     persistido y no solo en la configuracion. Un sello emitido por la TSA de
     desarrollo nunca debe poder confundirse con uno cualificado, ni sobrevivir
     a una migracion a produccion. Los datos generados en desarrollo **no se
     migran**: se descartan al cambiar de perfil.
  4. **Arranque ruidoso**: el proceso declara en el log, en cada arranque, que
     corre con credenciales no autoritativas y cuales.
  5. **Prueba de conmutacion**: una prueba verifica que el perfil productivo
     rechaza arrancar si algun proveedor de desarrollo esta compuesto. El
     fallo cerrado se conserva; lo que cambia es que en `desarrollo` hay un
     proveedor que responde, no que se elimine la exigencia.
- `evidencia`: `config.AuthMode` y los modos `fake`/`trusted`/`disabled` ya
  existentes en `internal/app/bootstrap`; patron anti-fuga de la demo
  sintetica; interfaz KMS de `gobiernoconvocatorias/cifrado_borradores.go`,
  que hoy rechaza texto cifrado sintetico y debera admitir el proveedor de
  desarrollo solo bajo el perfil correspondiente.

### T13 — Registro durable de accesos a datos personales con finalidad

- `origen`: paquete de cumplimiento (`docs/cumplimiento/`): ENS op.exp.8-10 y
  responsabilidad proactiva RGPD.
- `estado`: nuevo. Tras T12, misma zona de persistencia.
- `area_hexagonal`: nucleo (contrato) y adaptadores de persistencia.
- `accion`: registrar de forma durable y consultable cada acceso a datos
  personales: actor, rol, expediente afectado, finalidad declarada, base de
  la autorizacion, momento y version de reglas. Consulta interna autorizada
  para control y para atender derechos; sin volcar datos personales en el
  propio registro mas alla de la referencia opaca del expediente.
- `evidencia`: la EIPD lo fija como condicion de piloto; la cadena de
  auditoria existe en memoria y la autorizacion ya declara finalidad.

### T14 — Conservacion y expurgo programados

- `origen`: paquete de cumplimiento: RAT (plazos por actividad) y ENS
  mp.info.6; estudio de archivo documental de RRHH.
- `estado`: nuevo. Los plazos concretos los fija la mesa; programar el
  mecanismo con plazos configurables por tipo documental.
- `area_hexagonal`: nucleo y persistencia.
- `accion`: ciclo de vida por tipo de dato/documento: vigencia, bloqueo
  cautelar (recursos), supresion o anonimizacion irreversible con acta de
  expurgo auditada. Nada se borra sin acta; nada se conserva sin plazo.
- `evidencia`: `docs/estudio_requisitos/archivo_documental_rrhh_relacionado.md`.

### T15 — Atencion de derechos RGPD con evidencia

- `origen`: paquete de cumplimiento: RAT y EIPD (acceso, rectificacion,
  supresion, oposicion, limitacion).
- `estado`: nuevo.
- `area_hexagonal`: nucleo (casos de uso internos) y API interna.
- `accion`: casos de uso internos para: extraer todo lo tratado de una
  persona (acceso/portabilidad), rectificar con trazabilidad, suprimir o
  limitar con bloqueo cautelar, y oponerse; cada atencion genera recibo
  auditable. La supresion respeta las obligaciones de conservacion (T14).
- `evidencia`: EIPD seccion 6; sin estos casos de uso la atencion seria
  manual sobre la base de datos, sin evidencia.

### T16 — Cifrado en reposo de datos personales en PostgreSQL

- `origen`: paquete de cumplimiento: ENS mp.info.3.
- `estado`: nuevo. La gestion de claves productiva depende de KMS/HSM
  (Sistemas); disenar con inyeccion de proveedor como el resto.
- `area_hexagonal`: adaptadores de persistencia.
- `accion`: cifrar en reposo las columnas de datos identificativos y
  especiales (sobres AEAD con AAD canonico, como los flujos ya existentes),
  con anillo de claves historicas y rotacion sin parada; las columnas de
  busqueda usan derivaciones ciegas (HMAC) como ya hace la idempotencia.
- `evidencia`: patron AEAD ya implantado en la saga de firma; falta
  extenderlo a las columnas personales del DDL nuevo.

### T17 — Importador gobernado de exportaciones Convoca

- `origen`: decision del responsable (17/07): hasta que exista un modulo
  propio que sustituya a Convoca, las bolsas se alimentan con sus
  exportaciones de hoja de calculo. Encaja con DEC-079 (el VEC importa) y
  con las fuentes de autoridad verificables del nucleo.
- `estado`: nuevo. Tras el motor de reglas en curso; antes del porte del
  ciclo de Solicitud.
- `area_hexagonal`: adaptador de importacion + nucleo (fuente de autoridad).
- `accion`: caso de uso de importacion por lotes con dos formatos reales de
  Convoca (.xls):
  1. **Resumen por persona** (8 columnas): DNI/NIE enmascarado, Primer
     Apellido, Segundo Apellido, Nombre, Turno, Experiencia, Formacion,
     Total.
  2. **Detallado "con claves"** (12 columnas, una fila por merito):
     las cuatro de identidad mas Turno, Grupo, Descripcion del grupo,
     Orden grupo, Descripcion del merito, Puntos autobaremacion, Puntos
     tribunal, Motivo.
  Requisitos: zona de ensayo (staging) con validacion antes de cargar;
  huella SHA-256 del fichero e idempotencia (reimportar el mismo fichero no
  duplica); acta de importacion auditada (quien, cuando, fichero, filas
  aceptadas/rechazadas y motivo); la carga declara su fuente de autoridad
  "Convoca (exportacion enmascarada)" con su nivel: **no habilita por si
  sola actos con efectos** (la identidad completa debera confirmarse desde
  el registro corporativo antes de un llamamiento o contrato); los "Puntos
  autobaremacion" se importan como dato historico de contraste, nunca como
  puntuacion oficial (DEC-079: puntua el motor); los ficheros reales jamas
  entran en Git, en la imagen ni en fixtures (regla 8 de la auditoria):
  los tests usan ficheros sinteticos con las mismas cabeceras.
- `evidencia`: exportaciones reales inspeccionadas el 17/07 (52 filas
  resumen; 307 filas detalladas) con las cabeceras literales citadas; el
  DNI llega enmascarado desde origen (`***NNNN**`). Utilidad previa de
  referencia en `Trabajo/Emilio/main.go` (cruce con certificados PDF).

### T18 — Modulo de Formacion continua (DEC-080, fase informativa)

- `origen`: DEC-080 del registro de decisiones.
- `estado`: nuevo. Tras T17; comparte patron de importacion gobernada.
- `area_hexagonal`: modulo nuevo completo (`internal/modules/formacion`).
- `accion`: por cortes: (1) manifiesto `vec.module.formacion` con menu,
  permisos `formacion.*` y nivel de madurez declarado; (2) dominio de
  catalogo (curso, version, huella, estado editorial) con importador
  gobernado de paquetes OPES (HTML canonico + metadatos + SHA-256; staging,
  acta, revision editorial obligatoria antes de publicar); (3) progreso
  personal minimizado (el empleado solo ve el suyo; nueva actividad RAT);
  (4) tarjeta y vistas en el Portal del Empleado reutilizando la carcasa
  actual. Sin certificados ni computo de meritos en esta fase (DEC-080).
  El ciclo editorial nace preparado para homologacion (adenda DEC-080):
  estados borrador/revisado/publicado/homologado con firma de persona
  habilitada en el ultimo; en fase 1 el estado homologado queda cerrado por
  fallo cerrado (sin capacidad concedida no es alcanzable) y las vistas lo
  muestran como previsto, no disponible.
  El contrato del paquete de curso se congela por version; los ficheros de
  cursos reales no entran en Git: los tests usan paquetes sinteticos.
- `evidencia`: DEC-080; fabrica OPES verificada (salida HTML canonica,
  politica de fuentes, proveedor de audio); patron de importacion ya
  definido en T17.

### T19 — Adaptador de sellado de tiempo cualificado (TSA) para firmas

- `origen`: estudio `docs/portal_vec/sellado_de_tiempo_y_sincronizacion.md`
  (18/07). El dominio ya exige sello de tiempo y falla cerrado sin el; falta
  el conector productivo con una TSA real.
- `estado`: nuevo. Depende de la decision de proveedor (seccion 3 del
  estudio) y de credenciales de Sistemas; el adaptador puede construirse y
  probarse antes con una TSA simulada determinista.
- `area_hexagonal`: adaptador (sellado de tiempo) sobre el contrato ya
  existente `FirmaDecisionTecnica`.
- `accion`: cliente RFC 3161 detras de interfaz inyectable (TS@ de la
  Administracion recomendado; FNMT-RCM como alternativa): envia la huella de
  la revision, recibe y valida el token (cadena de la TSA, politica, huella
  coincidente) y rellena `SelloTiempoRef`, `PoliticaSelloTiempoRef/Version`,
  `ValidacionSelloTiempoRef` y `SelladaEn`; nunca fija la fecha desde el
  reloj local. Politica de sellado versionada; fallo cerrado si la TSA no
  responde o el token no valida (la firma queda pendiente, jamas sellada con
  hora local); soporte de aumento de longevidad
  (`RequiereAumentoLongevidad`, resellado antes de caducar la cadena). El
  reloj operativo sigue entrando por el puerto `Reloj` ya existente,
  alimentado del sistema sincronizado por NTP (ROA/RedIRIS, competencia de
  Sistemas). Sin endpoints ni credenciales reales en Git; pruebas con TSA
  simulada.
- `evidencia`: `internal/modules/bolsa/domain/firma_decision_validacion.go`
  ya modela sello, politica versionada, validacion, vinculo a revision
  sellada y longevidad; solo existen adaptadores de memoria.
