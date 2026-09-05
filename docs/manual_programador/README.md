# Manual mantenido del programador de VEC

Guía práctica para continuar el desarrollo sin reconstruir piezas existentes.
Se mantiene a mano; no es el catálogo de firmas ni un certificado de despliegue.
Estado funcional de referencia: 5 de septiembre de 2026.

## Qué leer y qué mantener

Este manual puede consultarse desde un clon del repositorio, sin acceder a
archivos privados de otra máquina. Lea las instrucciones aplicables al
directorio y las instrucciones vigentes del operador cuando se aporten.
La evidencia del recorrido y las órdenes actuales prevalecen sobre cifras
o bloqueos históricos. Las reglas operativas de este corte son:

- Priorizar avances observables de Contratación temporal, conectando las
  piezas existentes antes de perfeccionarlas o reconstruirlas.
- Mantener una línea canónica por capacidad y archivos de edición disjuntos;
  conservar el trabajo ajeno, sin copiar implementaciones entre ramas.
- Usar solo datos sintéticos, autorización de servidor y efectos persistentes
  reales; no presentar un adaptador DEMO como éxito ni usar cookies o
  almacenamiento web para el recorrido.
- Probar al completar cada hito, con alcance proporcionado. Exigir dos
  revisiones independientes en identidad, autorización, criptografía, SQL y
  fronteras de datos personales; no repetir suites globales por cada edición.
- No abrir el runner ni nuevas cadenas de documentos de decisión por inercia.
  Documentar el avance en los manuales y la guía existentes, sin convertir
  una mejora menor en una reescritura.

Este README explica arquitectura, conexión y trabajo cotidiano. El
[catálogo autogenerado LEEME.md](LEEME.md) sirve para localizar firmas y
comentarios Go; no debe editarse manualmente. Su advertencia histórica de no
editar este directorio se refiere al material generado, no a este README
manual expresamente separado.

Los cuatro perfiles documentales tienen finalidades distintas:

| Perfil | Entrada | Qué debe resolver |
| --- | --- | --- |
| Usuario | [Manual de usuario](../manual_usuario/manual_portal_bolsas.md) | Acceder, navegar, realizar acciones y entender estados. |
| Recursos Humanos | [Manual de RRHH](../manual_rrhh/README.md) | Recorrer el procedimiento, responsabilidades y límites funcionales. |
| Programador | Este README | Localizar, modificar y conectar las piezas reales. |
| Operador | [Manual de sistemas](../manual_sistemas/README.md) | Preparar, arrancar, parar, respaldar y recuperar el entorno. |

Los manuales distinguen funciones demostradas y pendientes; su existencia
no acredita por sí misma una entrega funcional.
La [guía del recorrido demostrado](../../GUIA_RECORRIDO_ALBERTO.md)
conserva los comandos operativos y los recibos sintéticos. No duplicar aquí
sus credenciales, recetas de instalación ni registros de cada ejecución.

### Cómo consultar el catálogo generado

El [generador](../../scripts/generar_manual_programador.py) ejecuta
`go list ./...` y `go doc -all`, y sobrescribe `LEEME.md` y nueve archivos
de áreas enumerados en `AREAS`. No escribe este `README.md`.

El índice conservado no incluye Contratación temporal. Además, el generador
agrupa los módulos distintos de Bolsa en `modulos_personal_cronos_dietas.md`,
aunque el título no los enumere todos, y omite un paquete si falla su
`go doc`. Por tanto, ausencia en el catálogo no prueba ausencia de código;
presencia tampoco prueba que esté conectado o probado en navegador.

Para una consulta puntual, desde la raíz del worktree:

```bash
rg -n 'NuevasRutas|NuevoManejadorSeleccionLlamamiento' internal/app internal/modules/contrataciontemporal
go doc vec-diputacion-granada/internal/modules/contrataciontemporal/ports
```

Solo si el hito incluye actualizar la referencia generada:

```bash
python3 scripts/generar_manual_programador.py
git diff --stat -- docs/manual_programador
```

Revisar el resultado completo: no ejecutar el generador por cada edición de
este manual ni confiar en sus resúmenes históricos como estado del producto.

## Estado funcional: cinco pasos y parte del sexto

El recorrido de desarrollo acreditado usa navegador, API interna, autorización,
PostgreSQL y recibos reales, con datos sintéticos y conservación tras reinicio:

| Paso del recorrido mantenido | Alcance demostrado |
| --- | --- |
| 1. Solicitud | Alta de una petición real. |
| 2. Análisis | Registro del análisis de Recursos Humanos. |
| 3. Bolsa | Propuesta y decisión de cobertura del expediente. No toda la aplicación Bolsa. |
| 4. Asignación | Envío del expediente a la unidad. |
| 5. Informe jurídico y fiscalización | Registro durable y resultados de fiscalización; devolución a unidad cuando corresponde. |
| 6. Llamamiento, parcial | Selección e inicio del llamamiento y registro de un aviso local; mismos recibos recuperados tras reinicio. |
| 7 y 8 | Nombramiento e incorporación, GINPIX y seguimiento: no declarados completos de extremo a extremo. |

El aviso local no demuestra correo enviado, entrega al destinatario, aceptación,
renuncia ni inicio de plazo. Una intención pendiente de salida (`outbox`) no
es un acuse del sistema externo. El contador es **cinco pasos completos más
un tramo del sexto**, no un porcentaje global de aplicación terminada.

La bandeja y el detalle de expedientes tienen código y una conexión candidata
en trabajo. Su instalación de consultas, configuración nominal de identidad
y recorrido real todavía no se dan por entregados en este manual.
Un `503` debe explicarse como dependencia no disponible, no sustituirse por
datos de presentación. Un `404` del panel de Bolsa tampoco demuestra que
haya fallado la identidad de Contratación temporal.

No confundir la instalación parcial de migraciones, una compilación correcta,
el menú visible o `/livez` con un recorrido completo. El cierre exige evidencia
de la versión que realmente se arrancó.

## Arquitectura real y propiedad

El ejecutable del recorrido es [cmd/vec-server](../../cmd/vec-server/main.go).
Carga `config`, llama a `bootstrap.NewHTTPServerWithConfig` y, con el perfil
de desarrollo explícito, entra en la composición de desarrollo.

```text
Navegador: shell + módulo + cliente HTTP
  → servidor de desarrollo con certificado de cliente
  → lista de rutas y autorización de servidor
  → adaptador HTTP → caso de uso → dominio y puertos
  → adaptadores PostgreSQL / Bolsa / salida local
  → estado, historia y salida pendiente durables → recibo minimizado
```

No existe una base de negocio común que cualquier módulo pueda modificar:

- `internal/modules/contrataciontemporal/` posee el expediente y su procedimiento.
- `internal/modules/bolsa/` posee sus convocatorias, propuestas, órdenes y
  llamamientos. Contratación consume sus puertos y recibos; no escribe sus tablas.
- `internal/vec/` aporta contratos y autoridades compartidas: identidad,
  autorización, contexto del actor, documentos y evidencias.
- `internal/app/` conecta dependencias concretas y superficies.
- `internal/candidate/` es código heredado; no es un camino alternativo para
  duplicar el flujo de Contratación.

Dentro de un módulo: `domain` contiene reglas sin infraestructura;
`ports` declara contratos; `application` coordina dominio y puertos;
`adapters` traduce HTTP, SQL y proveedores. Un caso de uso no debe importar
PostgreSQL ni leer el DOM. El contrato de otro módulo se consume por su frontera,
no se reproduce con otra implementación equivalente.

La identidad sintética de desarrollo procede del canal validado y de sus
autoridades explícitas. No acredita Kerberos corporativo ni habilita producción.
No admitir cuenta, perfil o permisos declarados por el navegador.

## Dónde modificar y dónde conectar

| Necesidad | Fuente o directorio que debe inspeccionarse |
| --- | --- |
| Configuración y separación de conexiones | [config/postgresql_contratacion_temporal.go](../../config/postgresql_contratacion_temporal.go) y [config/config.go](../../config/config.go). |
| Arranque y selección del perfil | [bootstrap.go](../../internal/app/bootstrap/bootstrap.go) y [composicion_desarrollo.go](../../internal/app/bootstrap/composicion_desarrollo.go). |
| Conectar el procedimiento real | [contratacion_temporal_desarrollo.go](../../internal/app/bootstrap/contratacion_temporal_desarrollo.go); los archivos `contratacion_temporal_*_desarrollo.go` aportan cada dependencia. |
| Registrar rutas internas del módulo | [composicion/interna/contrataciontemporal/rutas.go](../../internal/app/composicion/interna/contrataciontemporal/rutas.go), función `NuevasRutas`, y su llamada en bootstrap. |
| Contrato HTTP y respuesta pública | [adapters/httpinterno](../../internal/modules/contrataciontemporal/adapters/httpinterno/): manejador, contrato cerrado y proyección del recibo. |
| Regla o coordinación funcional | [domain](../../internal/modules/contrataciontemporal/domain/), [application](../../internal/modules/contrataciontemporal/application/) y [ports](../../internal/modules/contrataciontemporal/ports/). |
| Persistencia y transacción | [adapters/postgres](../../internal/modules/contrataciontemporal/adapters/postgres/) y [deploy/postgresql/contratacion_temporal](../../deploy/postgresql/contratacion_temporal/). |
| Navegación y montaje web | [portal.js](../../web/static/portal-empleado/portal.js) y [portal-modulos-coordinador.js](../../web/static/portal-empleado/portal-modulos-coordinador.js). |
| Formulario, vista, contrato y cliente HTTP | [modulos/contratacion-temporal](../../web/static/portal-empleado/modulos/contratacion-temporal/): `formulario-*.js`, `vista*.js`, `contrato-*.js` y `cliente-http-*.js`. |
| Textos visibles | [portal-i18n.js](../../web/static/portal-empleado/portal-i18n.js); en el módulo, `i18n.js`, `i18n-expedientes.js` e `i18n-llamamiento.js`; backend compartido en [internal/shared/i18n](../../internal/shared/i18n/). |
| Publicar un recurso estático del recorrido | [web/produccion.manifest](../../web/produccion.manifest), leído por [estaticos_produccion.go](../../internal/app/server/estaticos_produccion.go). Revisar los otros manifiestos solo si cambia su superficie. |

Para ampliar una acción existente, seguir su cadena completa: contrato y
manejador → caso de uso y puertos → adaptador real → dependencia de bootstrap
→ ruta → cliente y formulario. Añadir una función o una ruta a `NuevasRutas`
no basta si la raíz no le entrega una implementación autorizada.

Ejemplo ya existente: selección llama a
`POST /api/vec/contratacion-temporal/llamamientos/seleccion` con
`expediente_ref`, `version_esperada` y `clave_idempotencia`.
El formulario de comunicación toma organización, llamamiento, versión y recibo
antecedente de la respuesta autenticada, no de referencias inventadas.

Cuando un nuevo import JavaScript devuelve `404`, comprobar el archivo y
todos sus imports transitivos contra el manifiesto. El servidor carga la lista
al arrancar: modificarla exige reiniciar el proceso para validar el resultado.
No quitar guardas que separan recursos reales y de presentación.

Conservar el shell y el tema compartidos: contexto del expediente, etiquetas,
confirmación explícita, estados de espera/error y recibo. No duplicar helpers,
CSS estructural ni reglas funcionales en la vista.

## Preparar y arrancar desarrollo

Desde la raíz del worktree canónico asignado, no desde la raíz histórica ni
desde el producto publicado mientras otra persona lo está validando.

Requisitos verificables en las fuentes:

- Go: [go.mod](../../go.mod) declara mínimo `1.25.12` y toolchain `1.26.5`.
  Usar la toolchain de entrega fijada, sin degradarla para salvar una compilación.
- Bash, Python 3, `curl`, OpenSSL y utilidades de sistema para el lanzador y
  el generador de credenciales. Git para inventario y revisión.
- Node.js compatible con `node --test` para pruebas web focales.
- PostgreSQL **18.4** para este recorrido y sus migraciones exactas; Docker
  solo si se utiliza la instancia aislada existente. No levantar otra por inercia.
- Certificados y material persistente fuera de Git; conexiones con
  `sslmode=verify-full` y autoridades de certificación correctas.

El [lanzador](../../scripts/arrancar_vec_desarrollo.sh) no instala PostgreSQL,
roles ni migraciones. Genera o verifica material de desarrollo, compila un
binario temporal y escucha en loopback con TLS 1.3 y certificado de cliente.
No usar `arrancar_presentacion_rrhh.sh` para demostrar efectos persistentes.

Para el recorrido ya conservado, preparar las seis conexiones según el
[apartado de arranque de la guía](../../GUIA_RECORRIDO_ALBERTO.md#2-arrancar-vec-con-postgresql-real).
Son cuentas separadas, no seis alias del mismo usuario:

- `VEC_CT_DATABASE_URL`: ejecución del módulo.
- `VEC_CT_GOBIERNO_DATABASE_URL`: gobierno.
- `VEC_CT_CONFIRMADOR_DATABASE_URL`: confirmación de cobertura.
- `VEC_CT_LECTOR_RESULTADO_DATABASE_URL`: lectura de resultado.
- `VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL`: registro de autorización.
- `VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL`: ejecución de Bolsa.

Las conexiones de consultas y motivos RRHH, registro/revalidación de identidad
y contexto del actor que aparecen en la configuración candidata pertenecen al
trabajo de bandeja aún pendiente. No activarlas parcialmente ni atribuirles
éxito por existir sus variables.

Con esas conexiones preparadas y la variable `material_vec` apuntando al
directorio persistente aprobado:

```bash
go version
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material "$material_vec"
```

Abrir `https://localhost:8443/portal-empleado/` con el certificado de RRHH;
Intervención usa su certificado y perfil de navegador separados.
Si el proceso está en el remoto, abrir desde el equipo del navegador:

```bash
ssh -N -L 18443:127.0.0.1:8443 root@cidonia.cloud
```

En ese caso la URL es `https://localhost:18443/portal-empleado/`.
La importación protegida de certificados se describe en la guía; no publicar
claves, contraseñas ni cadenas de conexión en Git, capturas o mensajes.

La guía distingue la base del navegador en `55433` y la base de pruebas
en `55432`. Dirección puede reservar esta última para validar una candidata:
no lanzar pruebas en paralelo contra ella ni cambiar el destino del producto.
Nunca reconstruir o restaurar una base para resolver un fallo de arranque sin
una orden explícita.

Para parar, `Ctrl-C` en la terminal del lanzador. Para reiniciar, repetir el
mismo comando conservando conexiones, base y directorio de material. No
regenerar claves ni crear UUID nuevos para recuperar una operación anterior.
Una interrupción de transporte puede dejar un efecto confirmado: recuperar
con la misma intención y comparar recibos antes de ordenar otra operación.

## Ciclo corto: un hito observable, pruebas al terminar

1. Inventariar ramas, worktrees y cambios. Buscar primero la capacidad y sus
   llamadas; comparar deltas y patch-id antes de editar o integrar.
2. Acordar una única línea canónica y archivos de edición disjuntos. Si dos
   personas necesitan el mismo archivo, coordinar la edición, no crear copias.
3. Implementar el mínimo que permita avanzar a un usuario: una acción,
   conexión o recuperación completa. No rehacer piezas por mejoras cosméticas.
4. Al completar el hito, ejecutar las pruebas focales y revisar el recorrido.
   No repetir suites completas por cada línea ni abrir un nuevo ciclo documental.
5. Revisar seguridad según el riesgo y entregar archivos, evidencia y límites.
   Actualizar el manual afectado cuando cambie lo que su lector puede hacer.
6. Dirección integra y publica el cambio autorizado; solo después se limpia
   lo propio, integrado y sin modificaciones pendientes.

Comprobaciones iniciales de solo lectura en la línea actual:

```bash
git status --short
git worktree list --porcelain
git branch -vv
git cherry integracion/ct-producto-ligero-20260821 HEAD
git diff integracion/ct-producto-ligero-20260821...HEAD | git patch-id --stable
```

La última orden identifica el delta conjunto; `git cherry` compara equivalencia
de parches por commit. Ninguna sustituye inspeccionar cambios sin commit.
No copiar código manualmente entre ramas ni abrir worktrees sustitutos.

Elegir las pruebas del paquete o flujo modificado. Ejemplos existentes para
un hito de llamamiento, no una lista que deba repetirse para cualquier cambio:

```bash
go test ./internal/modules/contrataciontemporal/application -run Seleccion -count=1
go test ./internal/modules/contrataciontemporal/adapters/httpinterno -run Seleccion -count=1
node --test web/static/portal-empleado/modulos/contratacion-temporal/formulario-llamamiento.test.mjs
git diff --check
```

Para documentación sola: revisar contenido, enlaces y espacios; no arrancar
bases ni ejecutar las pruebas funcionales. Para interfaz: prueba focal y
revisión de navegador a 1440, 1024 y 390 píxeles, sin declarar evidencia visual
si solo se ejecutó Node.

Identidad, autorización, criptografía, SQL y datos personales requieren dos
revisiones independientes y comprobaciones específicas del riesgo. Los efectos
durables se demuestran con PostgreSQL real y un reinicio en el hito pertinente.
Las pruebas globales `go test ./...`, `go vet ./...` y la puerta de calidad
se reservan al cierre de integración que dirección programe; no son un paso
por cada edición ni sustituyen el recorrido humano.

### Commit, publicación y limpieza

Cada commit debe ser coherente, compilable y mostrar un avance o una
documentación utilizable. Añadir al índice solo las rutas revisadas; nunca
`git add -A` sobre un worktree compartido. Si la asignación exige entrega sin
commit, entregar el delta y dejar la confirmación a dirección.

Antes de integrar: repetir inventario/equivalencias, revisar el hash exacto
cuando corresponda y preservar el trabajo ajeno. La autoría configurada es
`aavidad`, sin atribuciones de IA. No cambiar la configuración global de Git.

Después del commit e integración autorizados, dirección hace push de la rama
canónica y verifica que el hash local, el de seguimiento y el remoto coinciden.
Sin push comprobado no decir «publicado». No forzar push ni borrar referencias
para ocultar divergencias. Producto limpio significa ausencia de cambios
pendientes allí; no exige borrar el trabajo de las candidatas.

Retirar exclusivamente ramas/worktrees propios, integrados y limpios, tras
confirmar que ninguna persona los usa. Preservar trabajo sin commit, ramas con
revisión pendiente y evidencias. En particular, los tres archivos históricos
sin seguimiento de `deploy/postgresql/autorizacion/` no forman parte de este
hito y no se añaden, borran ni reconstruyen.

## Cuándo actualizar este manual

Modificarlo en el mismo hito cuando cambien el arranque, la conexión de una
capacidad, sus límites o las rutas de trabajo del programador. Mantener las
firmas detalladas en Go y su catálogo generado; los pasos humanos en los
manuales de cada perfil; los recibos y comandos del recorrido en la guía.

Una entrega debe decir qué puede hacer ahora una persona, en qué entorno se
comprobó, qué prueba se ejecutó y qué falta para la acción siguiente.
Ni el número de archivos ni el volumen de pruebas son una medida de avance.
