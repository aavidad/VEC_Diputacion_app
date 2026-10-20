# Manual mantenido del programador de VEC

Guía vigente, contrastada con `origin/main@3c643ce34` el 20/10/2026.
El estado de las tareas está en el [tablero](../producto/TABLERO_TAREAS.md);
las decisiones técnicas, en el [TRD](../producto/TRD.md). Se desarrolla y
recorre en copias locales. cidonia permanece congelada; la siguiente
instalación corresponde a dirección en un servidor nuevo.

## Antes de escribir

Lea [AGENTS.md](../../AGENTS.md), [PRD](../producto/PRD.md) y
[plan de implementación](../producto/PLAN_IMPLEMENTACION.md), además de los
documentos que afecte el encargo. El caso de aceptación es
[FLUJO_APP §4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).
Busque la capacidad en el índice de código, `origin/main`, las ramas y las PR
abiertas. Para Go use `gopls` desde terminal. Si no dispone del índice,
indíquelo y contraste el árbol y sus referencias antes de añadir otra pieza.

Cada tarea parte de `origin/main`, tiene rama y carpeta propias y declara sus
ficheros. No se edita un fichero ocupado por otra PR. Una PR tiene una sola
responsabilidad, objetivo de 400 líneas manuales y máximo de 1.500.
Los títulos siguen `Tarea: <ID> — …`. Dirección revisa e integra.

## Arquitectura y propietarios

| Pieza | Ruta | Responsabilidad |
| --- | --- | --- |
| Dominio | `internal/modules/<modulo>/domain`, `internal/vec/domain` | Invariantes, estados y cálculos; sin HTTP, SQL ni proveedores |
| Puertos | `internal/modules/<modulo>/ports`, `internal/vec/ports` | Contratos mínimos que necesita la aplicación |
| Aplicación | `internal/modules/<modulo>/application`, `internal/vec/application` | Casos de uso que coordinan dominio y puertos |
| Adaptadores | `internal/modules/<modulo>/adapters`, `internal/vec/adapters` | PostgreSQL, HTTP, correo y otros proveedores |
| Composición | `internal/app/bootstrap`, `internal/app/composicion` | Selección de adaptadores, configuración y montaje de superficies |
| Servidor común | `internal/app/server` | Estáticos, salud, cabeceras y límites HTTP |
| Interfaz | `web/static/portal-empleado`, `web/static/area-personal`, `web/static/bolsa` | Pantallas y clientes de los contratos montados |
| Utilidades | `internal/shared` | i18n, telemetría y herramientas comunes sin negocio |

Las dependencias apuntan hacia los contratos y las reglas. El adaptador
implementa el puerto y la composición lo inyecta; el dominio no importa
aplicación ni adaptadores. Compruebe la ubicación real antes de mover código:
el árbol conserva paquetes con organización anterior.

Hay quince módulos en `internal/modules/`: `administracion`, `aspirantes`,
`bolsa`, `calendarios`, `carrera`, `certificados`, `contrataciontemporal`,
`cronos`, `dietas`, `formacion`, `meritos`, `personal`, `provision`, `seleccion`
y `usuarios`. Su presencia no acredita que estén completos ni activados.
El manifiesto común está en [types.go](../../internal/vec/domain/types.go),
y el registro en [service.go](../../internal/vec/application/service.go).
El menú consulta el catálogo de módulos; declararlo no monta sus rutas ni
concede permisos. No hay un instalador dinámico de módulos.

Persona es común. Cada módulo conserva sus datos y obtiene proyecciones
ajenas mediante puertos autorizados. Contratación temporal coordina; Bolsa
conserva convocatorias, candidaturas, orden y llamamientos; Personal conserva
relaciones de servicio y ocupaciones. Ningún módulo consulta tablas de otro.
Una cuenta o un certificado no prueban una relación de servicio.

## Composición y consumidores

[bootstrap.go](../../internal/app/bootstrap/bootstrap.go) monta los servidores
y registra módulos. Los ficheros de capacidad de `internal/app/bootstrap`
construyen dependencias concretas; `internal/app/composicion/interna` separa
la raíz interna y su configuración. Por ejemplo,
[contrataciontemporal/rutas.go](../../internal/app/composicion/interna/contrataciontemporal/rutas.go)
recibe casos de uso y autoridades ya constituidos y forma las rutas.
No se añade una conexión o una comprobación de entorno al dominio.

`cmd/vec-server` conserva el montaje combinado de desarrollo;
`cmd/vec-admin` atiende Administración por separado. `cmd/vec-interno` y
`cmd/vec-publico` tienen composiciones y superficies separadas. Una herramienta
de `cmd/` prepara o ejecuta su operación concreta: no es un cliente general
del expediente ni una autorización para publicar otras rutas.

Para añadir una capacidad:

1. Identifique su propietario y el paso del recorrido que la consume.
   Reutilice reglas y contratos existentes; un puerto nuevo necesita un
   consumidor concreto.
2. Añada la regla en dominio y su coordinación en aplicación. Conserve la
   versión esperada, la clave de idempotencia y el recibo del resultado.
3. Implemente los puertos con adaptadores. Si necesita datos ajenos, pida al
   módulo dueño una proyección autorizada. Pruebe persistencia y denegación
   en PostgreSQL real cuando la garantía dependa de él.
4. Conecte HTTP y composición con identidad validada, autorización positiva,
   límites y errores de catálogo. El actor no procede del cuerpo HTTP.
5. Conecte la pantalla al contrato real, con carga, vacío, error y reintento.
   Los textos van en `web/static/textos/<idioma>/`; la ayuda se abre con «?».
6. Recorra el paso en la copia propia y compruebe el resultado tras reiniciar
   aplicación y PostgreSQL. La PR dice qué pantalla usa la capacidad.

La ausencia de configuración opcional debe cerrar solo su función y explicar
qué falta mediante un error de catálogo. Las garantías se comprueban en el
montaje concreto: no se dan por cumplidas porque exista una interfaz.

## Identidad, autorización e historia

La lectura usa la sesión ligada al certificado y deja constancia de auditoría.
Los actos requieren una decisión central positiva para actor, acción, recurso,
ámbito y finalidad, consumida en la misma transacción que el efecto. Un perfil,
un menú o un selector no conceden autoridad.

Una escritura conserva auditoría, recibo e historia. El reintento con la misma
clave y contenido recupera el resultado; otro contenido con la misma clave
se rechaza. Una dependencia caída no abre acceso ni confirma una entrega.
Identificarse no firma documentos; un recibo local no confirma GINPIX,
correo ni un portafirmas externo. Use datos ficticios y deje claves,
certificados y configuración privada fuera de Git.

## Migraciones y ensayo SQL

La estructura y los permisos se describen en
[ESQUEMA_BACKEND](../producto/ESQUEMA_BACKEND.md). Antes de crear una migración,
reserve su número en `RESERVAS_MIGRACIONES.md`, fuera de Git. Las nuevas
migraciones del núcleo de autorización son del equipo R.

Las migraciones viven en `deploy/postgresql/<esquema>/migraciones/`.
La lista común posterior al esquema de referencia es
[`lista_sql_main_tras_hz37.txt`](../../deploy/principal/lista_sql_main_tras_hz37.txt):
respete el orden causal y sus huellas. No se modifica ni reaplica una SQL
instalada; se corrige con otra migración. No se usa DOWN sobre historia.
Al reconstruir una función, parta de su cuerpo instalado en la principal.
Toda función nueva o reconstruida `SECURITY DEFINER` lleva
`SET search_path = pg_catalog, pg_temp`.

El [ensayador de fase 1](../../deploy/postgresql/fase1/LEEME.md) restaura el
esquema sin datos y ejecuta instalación, pruebas SQL y reinicio:

```sh
python3 deploy/postgresql/fase1/ensayar.py --pruebas
```

Se ejecuta una vez sobre la cabeza final, con PostgreSQL 18 propio y memoria
disponible. Este ensayo de esquema y SQL no acredita el recorrido web.
La copia con datos y servicios se prepara con
[`copia_local.py`](../../deploy/principal/copia_local.py), según su
[receta](../../deploy/principal/LEEME_copia_local.md). No se ensaya en cidonia.

## Pruebas y entrega

En local se ejecutan solo las pruebas focales: formato, `go test -p 6`,
`go vet` del paquete y `-race` si hay concurrencia; Node sobre las pruebas
web afectadas y `git diff --check`. En escrituras cubra denegación, versión
vieja, repetición, clave con contenido distinto y conservación de historia.
Las pantallas se recorren con Playwright y el Chrome del sistema a 1440 y
390 px. Las reglas y los valores pendientes de RRHH se mantienen configurables.

La [CI](../../.github/workflows/ci.yml) ejecuta secretos, calidad, artefactos,
identidad y autorización PostgreSQL, Bolsa pública y recorrido SQL de fase 1.
La puerta completa es [`scripts/verificar_calidad.sh`](../../scripts/verificar_calidad.sh);
no se repite en local sobre el mismo commit. Una PR documental comprueba
referencias y espacios y deja las puertas automáticas a la CI.

Antes de entregar se usan `revisar-cambios-vec`, `documentar-entregar-vec`
y `pr-vec`. SQL lleva `revisar-sql-vec` y su ensayo. Los cambios de
identidad, permisos, SQL, HTTP o ficheros subidos pasan revisión de seguridad
focal, Semgrep local sin métricas y GoSec sobre lo cambiado cuando corresponda.
Dos revisiones independientes para lógica de negocio, permisos, identidad,
cifrado o SQL con permisos; una para lo demás. Los textos pasan `humanizer`.

La PR incluye resultado, pruebas, límites, configuración y SQL en orden.
Si cambia comportamiento, aporta guion de activación y arranque con la
configuración nueva y sin ella. La documentación sola declara activación
«ninguna». No se equiparan código integrado, SQL instalado, recorrido
comprobado y publicación. Dirección da por cerrada la tarea al integrar.

## Inventario generado e historia

[LEEME.md](LEEME.md) y los otros nueve índices Go son una instantánea generada
anterior al corte actual. Sus firmas se consultan contra el código vigente y
`gopls`. El generador `scripts/generar_manual_programador.py` conserva una
introducción desfasada; regenerar requiere corregirlo en una tarea de código.
Esta entrega solo documental no cambia el generador.

El [mapa técnico de septiembre](../historico/manual_programador_arquitectura_202609.md)
conserva el antecedente de arquitectura. Las instrucciones de operación se
consultan en la receta de copia local; los relevos y relatos de sesiones se
retiran del manual vigente y permanecen en la historia de Git.
