# O4-05: mapa de cableado productivo

Fecha: 26 de julio de 2026.

## Resultado

La interfaz administrativa de diecisiete pantallas se conserva. No se
reescriben HTML, componentes, presentadores, estilos ni traducciones. La
integración sustituirá las fuentes de presentación por adaptadores HTTP
neutrales que consuman los mismos casos de uso disponibles para web,
escritorio, CLI y MCP.

El recorrido productivo será:

```text
canal corporativo autenticado
  → identidad y sesión revalidadas
  → PDP, capacidad y ámbito exactos
  → HTTP protegido
  → aplicación
  → PostgreSQL con lectura y auditoría atómicas
  → adaptador visual
  → interfaz existente
```

La ausencia de cualquier dependencia provoca denegación o fallo de arranque;
nunca activa una fuente de presentación.

## Decisiones de dirección

### Prefijo de rutas

Se conserva la lista positiva ya probada bajo `/api/vec/`. Las dos consultas
previstas son:

```text
POST /api/vec/contratacion-temporal/cuadro/consultas
POST /api/vec/contratacion-temporal/expedientes/consultas
```

No se crea el prefijo alternativo `/api/interno/v1`: duplicaría dispatcher,
montaje, pruebas de superficie y configuración de proxy sin aportar una
capacidad funcional distinta.

### Identidad

`Remote-User`, `X-Forwarded-User` y cualquier cabecera de identidad libre no
son autoridades. El proxy deberá autenticarse ante VEC mediante mTLS y
transportar una aserción corporativa breve, protegida, ligada al canal y de un
solo uso. La aplicación:

1. valida el canal;
2. consume la aserción;
3. revalida sesión, cuenta, organización, perfil y garantía;
4. crea un contexto opaco de servidor;
5. retira el material de autenticación antes de invocar el manejador.

Se mantienen el rechazo de cookies y credenciales persistidas por el navegador
y la prohibición de `Set-Cookie`.

### Proyección y auditoría

La lectura de cuadro o detalle y el registro durable de acceso ocurren en una
única transacción:

```text
revalidar capacidad
  → fijar ámbito/RLS
  → leer proyección
  → registrar acceso
  → confirmar
  → publicar resultado
```

Un `SELECT` seguido de una auditoría independiente no es aceptable: podría
entregar información sin evidencia durable.

### Composición visual

La respuesta operativa minimizada no incorpora etiquetas, textos, permisos ni
definiciones de flujo. El adaptador visual combinará:

```text
proyección operativa
  + definición gobernada y versionada del flujo
  + catálogo gobernado
  + claves i18n
  + capacidades visuales concedidas
  = contrato visual existente
```

Las capacidades visuales solo controlan la presentación. Cada consulta o
efecto vuelve a ser autorizado en servidor. Una actuación sin soporte durable
se omite o aparece como no disponible; nunca se simula en producción.

## Cortes verificables

### C1 — puente de identidad interna

Entregable:

- adaptación estrecha del servicio común de identidad;
- contexto opaco de servidor;
- autoridades de ruta, alta, cobertura y consulta RRHH;
- retirada del material de autenticación.

Pruebas:

- canal sin mTLS;
- aserción ausente, forjada, repetida o caducada;
- organización, perfil o garantía incompatibles;
- cabeceras libres y cookies;
- nulos tipados y concurrencia;
- ausencia de `Set-Cookie`.

### C2 — proyección PostgreSQL de cuadro y detalle

Entregable:

- rol `NOLOGIN` exclusivo;
- login técnico con pertenencia mínima;
- fachadas SQL de cuadro y detalle;
- implementación de `SesionConsultaRRHH`;
- cursor autenticado y ligado a filtros y ámbito;
- acceso durable en la misma transacción.

Pruebas:

- PostgreSQL efímero real;
- organización, centro y unidad;
- cursor manipulado, caducado o trasladado;
- versión obsoleta;
- expediente inexistente, ajeno y denegado indistinguibles;
- ACL, RLS, concurrencia, reinicio y reversión protegida.

### C3 — HTTP protegido

Entregable:

- dos manejadores `POST`;
- JSON estricto sin duplicados ni campos desconocidos;
- límite de 4 KiB para la consulta de cuadro;
- `Accept` y `Content-Type` exactos;
- respuestas `no-store`;
- errores con código y clave i18n sin causas privadas;
- contrato OpenAPI.

### C4 — raíz interna

Entregable:

- construcción atómica de servicios y rutas;
- manifiesto de contratación temporal registrado;
- servidor creado exclusivamente por la cápsula de aplicación;
- propiedad y cierre inverso de pools y proveedores;
- `cmd` con escucha, apagado y cierre;
- arranque cerrado si falta identidad, PDP, PostgreSQL, catálogo o auditoría.

### C5 — fuente HTTP de la interfaz

Se conservan:

- contrato, presentador, vista y componentes de expedientes;
- estilos y traducciones comunes.

Se incorporan:

- contrato de proyecciones RRHH;
- cliente HTTP;
- fuente HTTP que implemente el puerto visual existente;
- selección de fuente desde el coordinador modular;
- manifiestos productivos sin adaptadores ni datos de presentación.

`listar` y `obtener` usan las nuevas proyecciones. `ejecutar` solo ofrece
efectos productivos realmente cerrados. Documentos y auditoría solo aparecen
cuando sus puertos reales estén integrados.

### C6 — E2E y revisión visual

La evidencia final incluye:

- cliente → identidad corporativa → API → aplicación → PostgreSQL → pantalla;
- cero cookies, `Set-Cookie` o autoridad procedente del navegador;
- denegación antes de PostgreSQL sin capacidad;
- reinicio, concurrencia y cierre de recursos;
- caída de un módulo sin paralizar los demás;
- manifiestos sin fuentes de demostración;
- capturas revisadas de las diecisiete pantallas con datos reales o estados
  honestamente no disponibles.

## Orden y paralelización

```text
C1 identidad ───────────────┐
                            ├→ C3 HTTP → C4 raíz → C5 web → C6 E2E
C2 PostgreSQL ──────────────┘
catálogos/composición visual ┘
```

C1, C2 y el contrato de composición visual pueden avanzar en paralelo con
write-sets disjuntos. C3, C4, C5 y C6 se integran en ese orden.

## Puertas que siguen abiertas

- identidad corporativa y acuerdo de integración con Sistemas;
- PDP y asignaciones reales aprobadas;
- funciones exteriores de emisión/consumo y adaptador Go de consultas;
- adaptador físico independiente para publicaciones visuales gobernadas;
- matriz TLS real con CA, SAN/hostname, errores de confianza y downgrade;
- EIPD, categorización ENS y aprobaciones organizativas de la matriz normativa;
- E2E y aceptación formal de RRHH.

La recuperación propia O4-05 y su pool acreditado obtuvieron `GO` técnico en
`d307b42` y `d3d6a04`, incluida una única sentencia de acreditación e
invocación y ataques reales sobre función, ACL y membresías. Esos cortes no
sustituyen las proyecciones de cuadro/detalle ni acreditan TLS productivo.

`2a9ddb1` cierra el fundamento tipado para cruzar una capacidad VEC-AD-3 sin
reinterpretar su JSON privado. La
[migración del contrato de consulta RRHH](o4_05_migracion_capacidad_consulta_rrhh_v3_2026-07-26.md)
liga después cuadro y detalle a órdenes, audiencias y recursos nominales
separados. Sigue pendiente consumir la exportación original dentro de la misma
transacción que lee y audita; el resumen Go no es una autoridad.

El [conjunto probatorio completo VEC-AD-3](o4_05_conjunto_probatorio_vec_ad3_2026-07-26.md)
añade después una única instantánea opaca con las diez entradas exigidas por
PostgreSQL y el resumen derivado del mismo nominal. El consumidor de alta no
se generaliza: cuadro y detalle necesitarán fachadas nominales separadas y
listas positivas propias.

`a0d39c1` cierra técnicamente esas dos fachadas nominales comunes, incluida la
huella real de auditoría, el replay de conciliación y las carreras de
revocación. `2820759` añade el rol nominativo y el registro durable, minimizado
y encadenado de accesos, con barrera global 16. Ambos cortes tienen revisión
independiente y PostgreSQL 18.4 verde.

La migración `000037` cierra C2-C con
[GO técnico independiente](revisiones/o4_05_revision_publicacion_global_rrhh_2026-07-26.md):
backfill base reproducible, singleton retenido hasta `COMMIT`, ordinal global
único, proyección 1:1 indexable y safe-down `17→16`. No crea cursor ni fachada
y no convierte el orden del backfill en orden histórico.

`2f014c4` cierra C2-D1 con
[GO técnico doble](revisiones/o4_05_revision_cursores_rrhh_postgresql_2026-07-26.md):
persistencia minimizada del cursor, privilegio CSPRNG exacto, ligadura de
página 2 al origen y de cada hijo al consumo padre, revocación y safe-down
semántico del catálogo.

Siguen pendientes las funciones exteriores que ejecuten emisión, consumo,
lectura y registro en una sola transacción. Después deben cerrarse el
adaptador Go, la matriz TLS y el E2E completo.

Este mapa no declara producción. Evita que una pantalla o una respuesta
aislada se presenten como recorrido terminado.
