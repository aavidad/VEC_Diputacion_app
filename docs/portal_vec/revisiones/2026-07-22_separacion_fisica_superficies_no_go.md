# Revisión de separación física de superficies

**Fecha:** 22 de julio de 2026
**Resultado:** **NO-GO para producción**
**Alcance:** composición de procesos, artefactos, despliegue y aislamiento entre la superficie pública, el área personal externa, el portal interno, la administración privilegiada y los clientes de automatización.

## Resumen ejecutivo

La aplicación dispone de contratos de seguridad, filtros HTTP, componentes de identidad y manejadores funcionales que constituyen una base válida. Sin embargo, todavía no existe una composición productiva que materialice las fronteras de seguridad exigidas.

La separación actual es principalmente lógica: el servidor público limita sus rutas y rechaza credenciales impropias, pero su binario continúa dependiendo de la composición monolítica y arrastra paquetes de dominios internos. El despliegue, además, expone un único servicio de aplicación y no produce imágenes independientes para cada superficie.

Por tanto, no debe autorizarse el paso a producción hasta completar el plan C1-C10 y superar las pruebas de aislamiento de este documento. Este veredicto no impide continuar el desarrollo ni utilizar los artefactos de presentación en entornos expresamente identificados como demostración.

## Evidencia exacta

### 1. Puntos de entrada y composición

- [`cmd/vec-server/main.go`](../../../cmd/vec-server/main.go) llama a `bootstrap.NewHTTPServerWithConfig`. Es el servidor integrado de desarrollo y su propia composición rechaza el modo productivo.
- [`cmd/vec-publico/main.go`](../../../cmd/vec-publico/main.go) llama a `bootstrap.NewHTTPServerPublicoWithConfig`. Aplica aislamiento de rutas, pero la composición productiva termina en `ErrComposicionProductivaNoDisponible`.
- `cmd/vec-presentacion` y `cmd/vec-cartografia-presentacion` son artefactos de demostración; no son raíces de composición productivas.
- `cmd/vec-emisor-capacidad-v4` sí ofrece un patrón reutilizable de proceso aislado: socket Unix, DSN propio y cierre ordenado.
- `cmd/bolsa-server` está retirado y falla de forma cerrada.

La composición sigue concentrada en [`internal/app/bootstrap/bootstrap.go`](../../../internal/app/bootstrap/bootstrap.go), especialmente en:

- `NewHTTPServerWithConfig`;
- `NewHTTPServerPublicoWithConfig`;
- `NewAPIPublicaBolsaWithConfig`;
- `newVECShellAPICompuestaConIdentidad`;
- `rechazarComposicionProductivaNoDisponible`.

Esa raíz importa y compone capacidades de candidatura, Personal, Cronos, Dietas y otros dominios. El grafo obtenido con `go list -deps ./cmd/vec-publico` incluye paquetes de `internal/candidate` y de los módulos Cronos, Dietas y Personal. En Go, la unidad de importación es el paquete y pueden existir efectos de inicialización; una lista blanca de rutas no sustituye una frontera de dependencia ni una separación física de proceso.

### 2. Aislamiento HTTP ya disponible

[`internal/app/server/server.go`](../../../internal/app/server/server.go) contiene controles valiosos:

- `NewHandlerPublicoWithConfig` limita la superficie a `/healthz`, `/bolsa`, `/verificar`, `/api/publico` y recursos estáticos seleccionados.
- `NewHandlerInternoWithConfig` limita la superficie a `/healthz`, `/portal-empleado`, `/locales` y `/api/vec`.
- Ambas superficies rechazan cookies, `Authorization`, `Proxy-Authorization`, cabeceras de identidad heredada (`X-VEC-*`, `X-Auth-*`, `X-Forwarded-*`, `Forwarded` y `Via`) y tráileres, y eliminan `Set-Cookie` de la respuesta.

Estos controles deben conservarse, pero [`internal/app/server/superficie_integrada.go`](../../../internal/app/server/superficie_integrada.go) sigue siendo una superficie integrada heredada y no debe formar parte de los artefactos productivos.

### 3. Contrato de seguridad e identidad reutilizable

[`internal/vec/adapters/httpseguridad/superficie.go`](../../../internal/vec/adapters/httpseguridad/superficie.go) ya modela las cuatro superficies humanas mediante `ConfiguracionSuperficie`, `ValidarArquitecturaCompleta` y `ValidarConjuntoSuperficies`:

1. pública anónima;
2. personal externa;
3. corporativa interna;
4. administración privilegiada.

El contrato permite que pública anónima y personal externa compartan un listener exterior, siempre con puertas de entrada distintas. Exige listeners diferentes para la superficie interna y la administración.

[`internal/vec/adapters/httpseguridad/identidad.go`](../../../internal/vec/adapters/httpseguridad/identidad.go) aporta `ServicioIdentidad`, `AutenticarCanalTLSMutuo`, `Resolver` y `ProyectarCuentaAutenticada`. La persistencia duradera de sesión y revalidación ya dispone de:

- [`RegistroSesionesPostgreSQL`](../../../internal/vec/adapters/httpseguridad/postgres/registro.go);
- [`RevalidadorAutenticacionActorPostgreSQL`](../../../internal/vec/adapters/httpseguridad/postgres/revalidador_autenticacion_actor.go).

### 4. Manejadores internos de Bolsa pendientes de montaje

Existen componentes aprovechables y probados, pero todavía no están montados en una raíz productiva:

- [`httpinterno.NuevoHandler`](../../../internal/modules/bolsa/adapters/httpinterno/handler.go);
- [`NuevoHandlerBorradores`](../../../internal/modules/bolsa/adapters/httpinterno/borradores.go);
- [`NuevoHandlerPropuestasLlamamiento`](../../../internal/modules/bolsa/adapters/httpinterno/llamamientos_http.go).

Estos manejadores esperan que el servidor haya preparado un contexto de actor autenticado y autorizado. No deben aceptar identidad o rol declarados por cabeceras HTTP del cliente.

### 5. Artefactos y despliegue

[`Dockerfile`](../../../Dockerfile) compila `vec-server`, `vec-presentacion` y el componente cartográfico, pero no compila ni copia `vec-publico` en el runtime de producto. El stage `runtime` contiene `vec-server` y el último stage del fichero es `herramientas-revision-web`.

El servicio `vec-api` de [`docker-compose.yml`](../../../docker-compose.yml) no fija un `target` de construcción. En consecuencia, Compose selecciona el último stage, que corresponde a las herramientas Playwright y no al runtime de producto.

**Fijar únicamente `target: runtime` no resuelve la producción.** Corregiría la selección accidental del stage, pero ese runtime contiene `vec-server`, conserva la composición integrada y falla de forma deliberada en producción mediante `ErrComposicionProductivaNoDisponible`. Tampoco crearía procesos, redes, cuentas, secretos ni artefactos independientes por superficie.

Además:

- [`web/produccion.manifest`](../../../web/produccion.manifest) mezcla área personal, Bolsa pública, portal del empleado, Cronos y Dietas.
- El logotipo público se sirve desde una ruta de la superficie interna (`/portal-empleado/assets/...`), creando acoplamiento entre artefactos.
- Compose mantiene un solo `vec-api` detrás de `proxy-local` y un backend compartido.
- La carga global de configuración lee parámetros de dominios internos incluso para procesos que no deberían conocerlos.
- Los adaptadores de fichero de Bolsa y VEC se identifican y validan expresamente como demostración; no pueden convertirse en fuente productiva cambiándoles el nombre.

## Decisión de procesos

Se adoptan cuatro unidades de despliegue con raíces de composición y artefactos distintos:

| Proceso | Superficie | Exposición y restricciones |
|---|---|---|
| `vec-exterior` | Pública anónima y área personal externa | Único proceso exterior permitido. Mantiene puertas de entrada, sesiones, audiencias y políticas separadas dentro del listener. No importa dominios internos ni recibe secretos de PostgreSQL interno o KMS corporativo. |
| `vec-interno` | Portal del empleado y gestión de RRHH | Accesible exclusivamente desde la red Mulhacén. Requiere identidad corporativa, canal autenticado y PDP por operación. No publica puertos hacia el exterior. |
| `vec-administracion` | Administración privilegiada | Listener, red, cuenta de servicio, audiencia y material criptográfico propios. No se combina con RRHH: hacerlo contradiría `ValidarConjuntoSuperficies` y el principio de mínimo privilegio. |
| `vec-automatizacion` | MCP público y, más adelante, MCP interno; CLI como cliente | El MCP público solo consulta información pública y ayuda. El MCP interno será otro despliegue dentro de la red de gestión. La CLI no es un listener: será un cliente fino de las API y nunca accederá directamente a PostgreSQL, KMS ni almacenes de efectos. |

Cada binario debe fijar su superficie en código. Una variable como `VEC_AUTH_MODE`, un perfil o un argumento de arranque no podrá transformar un binario público en interno o administrativo. `vec-server` permanecerá reservado al desarrollo y quedará excluido de las imágenes productivas.

## Qué se reutiliza

- Listas blancas de rutas y defensas frente a cookies, credenciales, cabeceras de proxy, tráileres y `Set-Cookie` de [`server.go`](../../../internal/app/server/server.go).
- Contrato completo de superficies, identidad, sesiones PostgreSQL y revalidación de `httpseguridad`.
- `ContextActor`, PDP V3 y la política de denegación por defecto.
- Puertos y manejadores internos de Bolsa, incluida la semántica V3 de borradores y llamamientos.
- Patrón de proceso aislado de `vec-emisor-capacidad-v4`.
- Manifiestos positivos, inventario exacto de imágenes y endurecimiento de contenedores: usuario no privilegiado, raíz de solo lectura, eliminación de capacidades y `no-new-privileges`.

## Qué no se reutiliza en producción

- La raíz monolítica de `bootstrap` como composición de producto.
- `vec-server` y `superficie_integrada.go` como servidores productivos.
- La carga global de configuración para todas las superficies.
- Los adaptadores de fichero marcados como DEMO.
- Identidad basada en cabeceras confiadas, `Principal` heredado o el `Bearer` opcional del cliente web.
- Un único manifiesto web, proxy, red backend o conjunto de secretos para todas las superficies.
- `/healthz` como prueba simultánea de vida y disponibilidad operativa.

## Plan de cierre C1-C10

### C1. Frontera de dependencias pública

- Crear `internal/app/composicion/publica/configuracion.go` y `raiz.go`, con pruebas unitarias de composición.
- Trasladar desde `bootstrap` exclusivamente la composición pública.
- Crear un cargador de configuración pública que no lea secretos internos.
- Adaptar `cmd/vec-publico/main.go`; la función heredada de `bootstrap` podrá delegar temporalmente para compatibilidad.
- Añadir `scripts/verificar_dependencias_superficies.sh` y una puerta CI que rechace importaciones de dominios internos en el binario público.
- Mantener cerrada la composición productiva hasta disponer de una fuente pública autoritativa.

### C2. Artefactos físicos independientes

- Añadir al [`Dockerfile`](../../../Dockerfile) stages explícitos `runtime-publico` y `runtime-interno`.
- Crear `web/publico.manifest` y `web/interno.manifest`.
- Mover recursos compartibles, como el logotipo, a `/assets/...` sin dependencia del portal interno.
- Añadir `scripts/verificar_contenido_artefactos_productivos.sh` para inventariar y negar cruces de binarios, interfaces, configuración y recursos DEMO.

### C3. Proyección pública real

- Implementar un adaptador PostgreSQL de solo lectura para `ConsultaConvocatoriasPublicas` y el catálogo público, bajo `internal/modules/bolsa/adapters/postgrespublico`.
- Crear migraciones y vistas explícitas en `deploy/postgresql/bolsa_publica`.
- Usar una cuenta `LOGIN` exclusiva con permisos únicamente sobre una proyección sin datos personales.
- Desbloquear la raíz pública productiva solo después de probar esa proyección; nunca mediante JSON DEMO.

### C4. Esqueleto interno cerrado

- Crear `cmd/vec-interno/main.go` e `internal/app/composicion/interna/{configuracion.go,raiz.go}`.
- Incorporar `server.NewHTTPServerInterno`.
- Fallar antes de abrir el socket si falta TLS mutuo, identidad, sesiones, `ContextActor`, PDP, KMS/TSA o cualquiera de los tres pools PostgreSQL exigidos por T20.
- No publicar un proceso vacío que responda como saludable.

### C5. Identidad interna

- Crear `internal/app/composicion/interna/identidad.go`.
- Validar TLS mutuo y una aserción criptográfica ligada al canal mediante `ServicioIdentidad`.
- Tratar la aserción como credencial verificada, nunca como usuario o rol confiado por cabecera; eliminarla antes de llegar al manejador.
- Obtener perfil, roles y ámbito desde `ContextActor` y PDP.
- Configurar TLS con `RequireAndVerifyClientCert`.
- Eliminar la rama `Bearer` opcional de `portal-borradores-api.js` en producción y mantener `credentials: "omit"`.

### C6. Cableado T20 de Bolsa

- Crear `internal/app/composicion/interna/bolsa_borradores.go`.
- Montar las rutas exactas de `httpinterno`.
- Encadenar identidad, registro de `ContextActor`, PDP V3, persistencia PostgreSQL V3 y la fachada T20 con KMS y recibo probatorio.
- Prohibir cualquier degradación silenciosa de V3 a V1.

### C7. Administración privilegiada

- Crear `cmd/vec-administracion` e `internal/app/composicion/administracion`.
- Añadir `server.NewHandlerAdministracionWithConfig` con lista blanca `/api/admin`.
- Asignar red, audiencia, SAN, pines y cuenta privilegiada independientes.
- Probar que un técnico de RRHH no puede alcanzar el listener administrativo.

### C8. Despliegue productivo

- Crear `deploy/compose/produccion.yml` y configuraciones `nginx-publico.conf`, `nginx-interno.conf` y `nginx-administracion.conf`.
- Fijar targets Docker explícitos y redes backend diferentes.
- No entregar secretos PostgreSQL/KMS al proceso público; no publicar el proceso interno; aislar administración.
- Montar secretos mediante ficheros (`*_FILE`), nunca en argumentos ni variables visibles.
- No confiar en `X-Forwarded-For` para identidad o autorización.
- Autenticar el salto proxy-aplicación mediante TLS mutuo o socket Unix autenticado.

### C9. MCP y CLI

- Crear `cmd/vec-cli` y `cmd/vec-mcp-publico` como clientes finos de las API, sin importar adaptadores PostgreSQL o KMS.
- Limitar MCP público a consultas públicas y ayuda.
- Desplegar el futuro MCP interno en la red de gestión, con TLS mutuo, identidad delegada y PDP en cada operación.
- Impedir acceso directo a despachadores o almacenes de efectos.

### C10. Retirada y documentación

- Eliminar del despliegue productivo el runtime ambiguo y `vec-server`.
- Actualizar Compose, README y el manual de Sistemas con la topología real.
- Mantener la presentación como artefacto desechable, separado y rotulado como DEMO.

## Pruebas de aislamiento obligatorias

### Dependencias y compilación

- Ejecutar `go list -deps` y `go build` por binario.
- Rechazar en CI cualquier importación cruzada: público hacia dominios internos; CLI/MCP hacia PostgreSQL, KMS o almacenes de efectos.

### Contenido de artefactos

- Exportar el inventario exacto de cada imagen.
- Comprobar ausencia de binarios, recursos web, configuración, secretos y adaptadores DEMO pertenecientes a otra superficie.
- Validar cada artefacto contra su manifiesto positivo específico.

### Matriz HTTP

- Probar todas las rutas válidas y todos los cruces entre superficies.
- Incluir rutas escapadas, dobles barras, segmentos no canónicos y codificaciones alternativas.
- Verificar rechazo de cookies, `Authorization`, cabeceras de proxy e identidad, tráileres y `Set-Cookie`, también con respuestas anticipadas, `Flush` y códigos informativos 1xx.

### Identidad y autorización

- Rechazar firma, audiencia, superficie o vínculo de canal inválidos.
- Rechazar repetición, sesión revocada, autenticación de un solo factor cuando se requieran dos y reutilización indebida de grupos criptográficos.
- Verificar que administración utiliza cuenta privilegiada propia y que RRHH carece de acceso al listener.

### Arranque y observabilidad

- Confirmar salida no nula antes de abrir sockets cuando falte configuración, TLS o dependencias obligatorias.
- Verificar que los errores están redactados y no revelan secretos.
- Separar `/livez` de `/readyz`; disponibilidad solo será positiva cuando las dependencias necesarias estén operativas.

### Red y despliegue renderizado

- Validar el Compose ya renderizado, no solo su plantilla.
- Probar que el público no puede resolver ni conectar con PostgreSQL interno o KMS.
- Probar que el exterior no alcanza interno o administración, y que RRHH no alcanza administración.
- Revisar puertos publicados, DNS, rutas del proxy, cuentas, volúmenes y secretos por contenedor.

### Clientes web, MCP y CLI

- Verificar que el JavaScript productivo no usa cookies, almacenamiento local ni `Bearer`, y que todas las peticiones usan `credentials: "omit"`.
- Probar listas blancas de operaciones MCP/CLI y ausencia de dependencias directas a infraestructura.

## Riesgos que permanecen abiertos

| Riesgo | Consecuencia | Tratamiento obligatorio |
|---|---|---|
| Un proxy único decide la zona solo por URL | Reunión accidental de fronteras y escalada por error de enrutado | Listeners, configuraciones y backends separados; pruebas negativas de rutas cruzadas. |
| TLS mutuo termina solo en el proxy | La aplicación no puede acreditar el canal original | TLS mutuo proxy-aplicación o socket Unix autenticado y protegido. |
| Confianza en `X-Forwarded-For` o cabeceras de identidad | Suplantación de origen, usuario o rol | Autorizar solo con identidad criptográfica y `RemoteAddr` del canal autenticado cuando proceda. |
| Redes o secretos compartidos | La separación de procesos queda anulada | Redes, cuentas y secretos mínimos por superficie. |
| `/healthz` devuelve éxito sin dependencias | Falso positivo operativo y despliegue inseguro | Separar vida y disponibilidad; comprobaciones reales de dependencias. |
| Renombrar un adaptador DEMO | Datos no autoritativos aceptados como reales | Adaptador PostgreSQL productivo y puerta CI contra recursos DEMO. |
| Rama `Bearer` inactiva pero presente | Regresión futura contraria a la política sin cookies/tokens del portal | Eliminarla del artefacto productivo y probar su ausencia. |
| Colisión de puertos, DNS o reglas de proxy | Exposición involuntaria de superficies internas | Pruebas sobre despliegue renderizado y escaneo desde cada zona. |
| Carga global de configuración | Exposición de nombres, credenciales o dependencias ajenas | Cargadores mínimos por proceso. |
| Añadir solo `target: runtime` | Sensación falsa de cierre: se ejecutaría el servidor integrado y seguiría sin existir separación física | Crear y seleccionar los runtimes específicos C2; mantener NO-GO hasta C1-C10. |

## Condición de salida del NO-GO

El veredicto podrá revisarse únicamente cuando existan los cuatro artefactos decididos, las raíces pública, interna y administrativa fallen de forma cerrada, las fuentes de datos sean autoritativas, el despliegue materialice redes y secretos separados y todas las pruebas de aislamiento anteriores estén automatizadas y en verde. Una demostración funcional o una lista blanca de rutas, por sí solas, no satisfacen esta condición.
