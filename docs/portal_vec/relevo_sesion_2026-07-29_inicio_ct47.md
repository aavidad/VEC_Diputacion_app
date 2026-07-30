# Relevo de sesión: inicio de CT-000047

Fecha: 29 de julio de 2026.

## Motivo

Después de cerrar CT-000046, dos inventarios independientes contrastaron el
camino crítico documental con el código no de prueba. La composición raíz no
puede ser el siguiente commit aislado: antes faltan dos fronteras que el estado
anterior daba por disponibles demasiado pronto.

No se ha perdido trabajo ni se revierte CT-000046. El hallazgo corrige el orden
de integración y evita construir una raíz con dobles o autoridad ficticia.

## Estado confirmado

Sí existen y se reutilizan:

- las fachadas PostgreSQL CT-000045;
- el adaptador y pool nominal CT-000046;
- los casos de uso de cuadro y detalle;
- el dispatcher de rutas exactas;
- la cápsula mTLS/TLS 1.3;
- el ciclo de vida con propiedad exclusiva y cierre inverso;
- las piezas comunes de sesión, contexto actor y PDP V3.

No existen todavía en código productivo:

1. los manejadores HTTP de cuadro y detalle;
2. una implementación de `AutoridadContextoConsultaRRHH`;
3. una implementación de `AutorizadorConsultaRRHH`;
4. el ensamblado de esas piezas en la raíz y en la web.

Además, Sistemas debe proporcionar los conectores y materiales reales de
verificación de aserciones corporativas, política de garantía, KMS/firma,
certificados, roles y secretos. Su ausencia mantiene el arranque cerrado.

## Orden corregido

```text
CT-000047A: HTTP protegido de cuadro/detalle
→ CT-000047B: puentes de autoridad y PDP reales
→ composición raíz y propiedad de recursos
→ matriz TLS/mTLS viva
→ fuente HTTP de la misma web definitiva
→ E2E HTTP completo contra PostgreSQL 18
→ conformidades RRHH, DPD y Sistemas
```

## CT-000047A integrado

Write-set exclusivo:

```text
internal/modules/contrataciontemporal/adapters/httpinterno/consulta_rrhh*.go
```

Alcance:

- dos rutas `POST` exactas ya fijadas por el diseño;
- cuerpos JSON cerrados que solo transportan intención;
- límites, tipos y cabeceras en lista positiva;
- rechazo de cookies, credenciales y cabeceras de identidad libres;
- respuestas `no-store`, cero `Set-Cookie` y errores opacos localizables;
- reutilización de los casos de uso y constructores existentes;
- pruebas de contrato, límites, cancelación y no oráculo.

El conjunto `c430785`–`b00d2ec` obtuvo `GO` independiente con P0=0 y P1=0.
La revisión inicial obligó a corregir ruta opaca, prioridad de cancelación y
validación pública del resultado antes de integrar. Los P2 restantes quedaron
cerrados después en dos minitareas revisadas de forma independiente:
`cd82caa` comprueba cancelación tras decodificar y `fc039c2` completa la
matriz negativa de componentes de `URL`. CT-000047A queda con P0=P1=P2=0.
Véase [la evidencia de revisión](revisiones/o4_05_revision_http_consultas_ct_000047a_2026-07-29.md).

El corte no toca SQL, PostgreSQL, raíz, web, estilos ni adaptadores de
presentación. Añade validadores neutrales en `ports` para impedir que HTTP
publique una proyección no coherente con la solicitud; las garantías privadas
de capacidad, ámbito y recibo permanecen en sus validadores internos.

## CT-000047B

Antes de programarlo debe quedar probado que cada conversión reutiliza las
autoridades VEC existentes y no fabrica material de confianza. Los proveedores
corporativos ausentes se inyectarán por puertos; no se sustituirán por memoria,
cabeceras, cookies, datos DEMO o configuración de desarrollo.

El primer prerrequisito mínimo está integrado:

- `700d72a` devuelve vínculo y resultado registrado desde una única
  resolución;
- `f49afd0` conserva el contrato y los errores del constructor compatible;
- `f186ce9` cierra las copias defensivas de todos los contenedores mutables.

El conjunto obtuvo `GO` independiente con P0=0, P1=0 y P2=0. La
[evidencia](revisiones/o4_05_revision_vinculo_resultado_ct_000047b_a_2026-07-29.md)
no acredita todavía autoridad de canal, PDP, raíz ni E2E.

Dos inventarios están descomponiendo el resto en unidades mínimas compilables.
No se abrirá una tarea genérica de «autoridad/PDP»: retención nominal, recursos
cerrados, decisión, material atestado, cronología, rutas y ciclo de vida se
confirmarán y revisarán por separado.

La retención nominal quedó integrada en `be59d58` con
[GO independiente](revisiones/o4_05_revision_retencion_nominal_ct_000047b1_2026-07-29.md)
y P0=P1=P2=0. `ContextoConsultaRRHH` conserva ahora el par exacto mediante un
clon privado, sin getter ni serialización, y vuelve a comprobarlo en cada uso.

La [decisión CT-000047B](decision_puentes_autoridad_consulta_rrhh_ct_000047b_2026-07-29.md)
elige guardianes de Contratación temporal y una fachada VEC de alto nivel. Se
descarta una tercera capacidad común sin segundo consumidor real. El siguiente
corte D0 quedó integrado en `6a18b43` con
[GO independiente](revisiones/o4_05_revision_recurso_opaco_ct_000047d0_2026-07-29.md)
y P0=P1=P2=0. D1 y D2 quedan desbloqueados para construir por separado los
recursos cerrados de cuadro y detalle, sin getters ni mapas aportados por el
cliente.

D1 quedó integrado en `c908ce6` con
[GO independiente](revisiones/o4_05_revision_recurso_cuadro_ct_000047d1_2026-07-29.md)
y P0=P1=P2=0. Organización, referencia, módulo, tipo, ámbitos, dominio y
huella se derivan exclusivamente del contexto y la solicitud tipada. D2
quedó integrado después en `c174644` con
[GO independiente](revisiones/o4_05_revision_recurso_detalle_ct_000047d2_2026-07-29.md)
y las mismas garantías cerradas para el expediente y su versión observada.

A2 se dividió para no crear un commit monolítico. A2.1 quedó integrado en
`d69a744` con
[GO independiente](revisiones/o4_05_revision_raiz_nominal_ct_000047a21_2026-07-29.md)
y P0=P1=P2=0. El servicio de confianza conserva y recupera internamente la
raíz pública nominal ligada a una prueba exacta; A2.2 puede ahora construir la
fachada de emisión sin exponer el catálogo o una clave.

A2.2 quedó integrado en `a677b14` con
[GO independiente](revisiones/o4_05_revision_fachada_emision_vec_ct_000047a22_2026-07-29.md)
y P0=P1=P2=0. La fachada encadena concesión durable, atestación, confianza,
capacidad, raíz y material sin aceptar selectores libres. A4 puede empezar a
componer los dos emisores nominales y guardianes de cuadro y detalle.

A4.1 quedó integrado en `05d6767` con
[GO independiente](revisiones/o4_05_revision_par_emisores_ct_000047a41_2026-07-29.md).
El par privado rechaza nulos, implementaciones sin identidad física y la misma
instancia en ambas posiciones.

A4.2 quedó integrado en `841f90a` con
[GO independiente](revisiones/o4_05_revision_preparacion_nominal_ct_000047a42_2026-07-30.md)
y P0=P1=P2=0. Las preparaciones privadas de cuadro y detalle derivan D1/D2,
retienen un solo clon del resultado exacto y ligan acción, finalidad, recurso,
motivo, correlación y vínculo a la solicitud VEC. No resuelven motivos,
generan correlaciones, emiten material ni publican API. A4.3 queda
desbloqueado para componer esas autoridades nominales sin exponer el resultado
registrado.

A4.3 quedó integrado en `fa304c4` con
[GO independiente](revisiones/o4_05_revision_guardian_nominal_ct_000047a43_2026-07-30.md)
y P0=P1=P2=0. El guardián público tiene métodos nominales separados, resuelve
M0 y correlación una vez, reutiliza A4.2, selecciona el emisor A2.2 exacto y
valida un segundo reloj no retrógrado antes de construir el material. Un error
posterior a evidencia durable devuelve material cero. A5 debe ahora migrar los
dos casos de uso y retirar la vía copiable anterior antes de componer la raíz.

A5.1 quedó integrado en `bea0b4b` con
[GO independiente](revisiones/o4_05_revision_servicios_emisor_ct_000047a51_2026-07-30.md),
P0=P1=0 y un P2 no bloqueante. Cuadro y detalle usan ya el emisor A4.3 exacto
y dos relojes posteriores para capacidad y orden. A5.2 debe retirar el
andamiaje de prueba legado, migrar los fixtures restantes y cerrar la fábrica
cruda antes de eliminar `AutorizadorConsultaRRHH`.

A5.2 quedó integrado en `12aaaf6` con
[GO independiente](revisiones/o4_05_revision_retirada_vias_crudas_ct_000047a52_2026-07-30.md)
y P0=P1=P2=0. Ya no se exportan el autorizador legado ni la fábrica cruda de
material; A4.3 conserva la única construcción privada. Los dos fixtures usan
el recorrido real, quedan por debajo de 500 líneas y mantienen los vectores
PostgreSQL byte a byte. A5 queda cerrado; el siguiente eslabón funcional es
M1/M2 y después la composición raíz.

El contrato M0 de motivos quedó integrado en `c1bb5ec` con
[GO independiente](revisiones/o4_05_revision_contrato_motivos_ct_000047m0_2026-07-29.md).
Cuadro y detalle tienen métodos nominales separados, sin selectores libres.
La
[decisión M1/M2](decision_motivos_nominales_consulta_rrhh_ct_000047_2026-07-30.md)
reserva `000008`–`000010` para fundamento, publicación y resolución, seguidas
de un adaptador PostgreSQL con pool exclusivo. No se usarán `MAX(version)`,
configuración libre ni referencias fijadas en código. La activación real sigue
sujeta al catálogo aprobado y a los materiales de Sistemas.

El primer candidato M1.1, `99c1e1b`, obtuvo
[NO-GO independiente](revisiones/o4_05_revision_fundamento_motivos_ct_000047m11_nogo_2026-07-30.md)
con P0=0, P1=1 y P2=0. Una reentrada aceptaba triggers y políticas RLS
homónimos degradados y una FK ausente; el `down` tampoco distinguía la
procedencia de una restricción en una tabla anterior. M1.1a corrige la
adopción exacta y añade esos estados envenenados a PostgreSQL 18.4.

La cadena M1.1 quedó finalmente integrada hasta `047e52c` con
[GO técnico independiente](revisiones/o4_05_revision_fundamento_motivos_ct_000047m11_go_2026-07-30.md)
y P0=P1=P2=0. Los NO-GO posteriores detectaron tablas no permanentes,
disparadores propios e internos degradados, ACL, reglas, colaciones, índices y
una retirada que no verificaba todas sus marcas. Esos casos forman ahora parte
del arnés PostgreSQL 18.4. `000008` queda cerrado; el siguiente tramo es M1.2,
reservado exclusivamente para `000009`.

M1.2 quedó integrado hasta `0a564e2` con
[doble GO independiente](revisiones/o4_05_revision_publicacion_retirada_motivos_ct_000047m12_2026-07-30.md),
P0=0 y P1=0. `000009` aporta publicación y retirada nominal de cuadro y
detalle, historia de solo adición, replay, cronología, exclusión temporal,
actor derivado del `session_user`, ACL/RLS cerradas y safe-down semántico. Las
revisiones corrigieron una FK fundamental alterable con `ON DELETE CASCADE` y
un orden de locks no garantizado por una lista `SELECT`; ambos casos forman
parte del arnés PostgreSQL 18.4. M1.3/`000010` queda diseñada para resolver
las dos referencias sin selector libre. Su
[coordinación acotada](coordinacion_ct_000047m13_resolucion_motivos_000010_2026-07-30.md)
fija tres ficheros, dos fachadas, matriz PostgreSQL 18.4 y revisión
independiente. Una revisión preventiva detectó que el evaluador V2 conserva
una función histórica con selectores libres. M1.3 no lo reutilizará: antes se
integra el
[rol exclusivo M1.R](coordinacion_ct_000047m1r_rol_resolutor_motivos_rrhh_2026-07-30.md).

M1.R quedó integrado en `231648b` tras
[GO independiente](revisiones/o4_05_revision_rol_resolutor_motivos_ct_000047m1r_2026-07-30.md),
P0=P1=P2=0 y reproducciones reales sobre PostgreSQL 18.4. El rol nace sin
acceso funcional y con una única ACL `CONNECT`; M1.3 se reanuda sobre esta
base y será la única pieza que conceda las dos fachadas nominales.

M1.3 quedó integrada en `281f52b` tras
[doble GO independiente](revisiones/o4_05_revision_resolucion_motivos_ct_000047m13_2026-07-30.md),
P0=P1=P2=0 y siete ejecuciones completas acumuladas sobre PostgreSQL 18.4
entre productor, revisor y dirección. `000010` expone solo las dos fachadas
nominales, valida la topología del resolutor, la vigencia y las retiradas,
retiene los bloqueos causales y posee `down` seguro. Una contrarrevisión
detectó una cobertura RLS ausente en el primer runner; se corrigió y repitió
la matriz antes del commit. El frente activo pasa a M2.1, pool y acreditación
exclusivos.

M2.1 quedó integrada hasta `c6c0028` tras
[doble revisión independiente](revisiones/o4_05_revision_pool_motivos_ct_000047m21_2026-07-30.md),
con P0=P1=0. El pool PostgreSQL es exclusivo, no expone `pgxpool`, sella pool,
conexión y transacción, conserva los OID de ambas fachadas y reacredita en cada
operación identidad, TLS, topología, ACL global, `PUBLIC`, autoridad efectiva,
dependencias y manifiestos. PostgreSQL 18.4 rechazó estados hostiles de tablas,
tipos, esquemas, ACL predeterminadas, políticas y catálogos de tipos. M2.2,
adaptador de las dos resoluciones nominales, queda desbloqueada.

M2.2 quedó integrada en `45f985c` y documentada en `9981012` tras doble
[revisión independiente](revisiones/o4_05_revision_adaptador_motivos_ct_000047m22_2026-07-30.md),
con P0=P1=P2=0. Implementa las dos consultas nominales literales, cardinalidad
exacta, transacción serializable, reacreditación viva, reversión acotada y
error opaco. La revisión corrigió antes del commit un contexto nulo tipado que
podía provocar pánico durante la normalización del error.

M2.3 quedó integrada en `bd22d05` tras doble
[GO funcional y de seguridad](revisiones/o4_05_revision_adaptador_motivos_ct_000047m23_2026-07-30.md),
P0=P1=P2=0. PostgreSQL 18.4 real superó tres ejecuciones limpias con ausencia,
referencias distintas, retiradas, caducidad, carrera causal, ACL/DML, sesión
hostil, derivas, reconexión y reinicio del mismo pool. M2 queda cerrado; el
frente activo pasa a las autoridades corporativas/PDP necesarias para la
composición raíz.

CT-000047C1 quedó integrada técnicamente en `4471e3a` y cerrada
documentalmente en `c7b587c` tras
[doble GO independiente](revisiones/o4_05_revision_capsula_identidad_ct_000047c1_2026-07-30.md)
y P0=P1=P2=0. Dos NO-GO previos evitaron integrar una cápsula cruzable,
reutilizable entre peticiones o incompletamente cerrada frente a
serializadores. El corte final liga una sola petición mediante estado atómico,
revalida la sesión y el canal al consumir y conserva una API basada solo en
`context.Context`.

La
[decisión C2](decision_contexto_corporativo_rrhh_ct_000047c2_2026-07-30.md)
prohíbe usar el PDP como selector y obliga a resolver perfil y organización
junto con el registro ContextoActor en una única transacción `SERIALIZABLE`.

CT-000047C2.1a quedó integrada técnicamente en `e8c950c` y cerrada
documentalmente en `808522d` tras
[doble GO independiente](revisiones/o4_05_revision_rol_selector_contexto_rrhh_ct_000047c21a_2026-07-30.md)
y P0=P1=P2=0. Crea el rol selector `NOLOGIN` con solo `CONNECT` y sin acceso
funcional. Tres ciclos de revisión corrigieron la autoridad implícita de
`PUBLIC` sobre tipos y sustituyeron una heurística evadible por la relación
inversa exacta del catálogo para arrays automáticos. La ejecución GitHub
`30527303065` terminó completamente verde.

C2.1b quedó integrada en `ea91b30`–`d768007`. Su contrato recibe
exclusivamente las referencias de autenticación y sesión y devuelve cuenta,
método, garantía y caducidad observados; no acepta ni devuelve perfil,
organización o candidatos. La revisión final obtuvo P0=P1=P2=0 después de
cerrar cobertura adversarial, réplica física y una prueba no tautológica del
retorno. El siguiente corte es C2.2-S0.1.

El emisor HMAC actual permanece limitado a pruebas. Antes de producción falta
el puerto de MAC no exportable y el adaptador KMS/HSM, además del firmante COSE
V3 y el conjunto de confianza proporcionados por Sistemas.

El primer candidato del puerto MAC, `dd97e3a`, obtuvo
[NO-GO independiente](revisiones/o4_05_revision_puerto_mac_ct_000047macp1_nogo_2026-07-30.md)
con P0=0, P1=1 y P2=0. Clonaba y recorría una preimagen antes de aplicar su
límite de 32 KiB. No se integró ni publicó por sí solo; solo podía incorporarse
junto con una corrección revisada que aplicara el límite antes de cualquier
reserva o SHA-256 y lo demostrara con una regresión de asignaciones.

La corrección `cfa1bc6` cerró el hallazgo y obtuvo
[GO independiente](revisiones/o4_05_revision_puerto_mac_ct_000047macp11_2026-07-30.md)
con P0=P1=P2=0. Se integró en `3f63599` y `eb6fbfd`: el límite se aplica
antes de clonar, recorrer o calcular SHA-256, y una regresión acredita cero
asignaciones adicionales. Esto cierra el contrato del puerto, no el adaptador
KMS/HSM ni la verificación transaccional de PostgreSQL.

La revisión detectó además que el SQL actual necesita el secreto HMAC dentro
de PostgreSQL. La
[decisión MAC/PostgreSQL](decision_mac_capacidad_y_postgresql_ct_000047_2026-07-29.md)
mantiene `NO-GO` productivo hasta que Sistemas y DBA elijan HSM accesible desde
PostgreSQL o una nueva capacidad asimétrica. Verificar solo en Go queda
descartado porque rompería el consumo autoritativo dentro de la transacción.

La cronología de detalle quedó corregida en `e3e12d5`: contexto, autorización
y orden ya no comparten un instante capturado antes de las fronteras lentas.
La minitarea obtuvo `GO` independiente sin hallazgos. Véase la
[revisión CT-000047C2](revisiones/o4_05_revision_cronologia_detalle_ct_000047c2_2026-07-29.md).

La misma garantía quedó cerrada para el cuadro en `ce77db3`, también con `GO`
independiente y P0=P1=P2=0. Véase la
[revisión CT-000047C1](revisiones/o4_05_revision_cronologia_cuadro_ct_000047c1_2026-07-29.md).

## Métricas

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `24/46`, 52 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

El inventario no reduce el avance funcional: hace visible una dependencia de
integración que ya estaba abierta. M1.3 está confirmada y revisada, pero no
aumenta la métrica oficial por ser una pieza interna: CT-000047 solo contará
cuando cierre su recorrido funcional y el CI publicado quede verde.

## Autoridad de trabajo

Rama estable:

```text
integracion/ct-o4-04e-20260726
```

Worktree estable:

```text
.worktrees/ct-stable-docs
```

Trabajo activo: C2.2-S0.1 en un worktree aislado. C2.1b ya no es trabajo
local pendiente. No se programa en el directorio raíz histórico ni se
modifica el Word de RRHH sin seguimiento.

El candidato inicial de C2.2-S0.1 (`240568d`) recibió `NO-GO` independiente
porque no detectaba ACL y propiedades de columna, disparadores internos,
publicaciones ni todos los atributos relevantes de los roles. La corrección
`26b7d61` permanece aislada y pendiente de revisión independiente; no debe
integrarse ni publicarse antes del `GO`.

C2.2-S0.2 dispone de inventario, pero el primer intento se detuvo sin commit:
el contrato completo superaba los límites de tamaño de un único `down` y un
único runner. El siguiente paso es aprobar una división ejecutable por el
migrador real antes de retomar la implementación; no se deben comprimir ni
omitir garantías para encajarla.

## Optimización probatoria CT88

El corte `f662302` optimiza dos validadores documentales sin cambiar su
contrato. Una revisión independiente obtuvo P0=P1=P2=0 y reprodujo mejoras
del 37,5 % al 53,0 % en las pruebas afectadas. Dirección repitió las pruebas
normales, la carrera focal y `go vet` sobre la rama estable. La
[evidencia reproducible](revisiones/revision_optimizacion_validacion_documental_ct88_2026-07-30.md)
queda separada del camino funcional: no aumenta métricas ni habilita
producción.

GitHub detectó después una dependencia C4 no aprobada introducida por el
refactor. CT104/CT108 la eliminaron sin relajar la barrera y unificaron la
primitiva SHA-256 del dominio. CT109 dio `GO`, P0=P1=P2=0; los commits estables
son `7760de1` y `71ac897`.
