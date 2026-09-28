# Manual técnico de sistemas de VEC

**Corte de fuente:** `main@7247682cb` (28 de septiembre de 2026). Este manual describe código integrado y el último despliegue **documentado** en [ESTADO_PROYECTO.md](../../ESTADO_PROYECTO.md), no certifica el estado instantáneo de un servidor. La principal de cidonia sirvió un ejercicio de Bolsa y Contratación temporal con datos sintéticos en el corte del 26/09. La producción con datos reales permanece en **NO-GO**: faltan, entre otras validaciones, la EIPD y la categorización ENS. En la principal sigue documentado `pg_hba` con `trust` para conexiones TCP internas; cerrar ese punto requiere un cambio operativo comprobado.

## Arquitectura y ventajas comprobables

VEC separa reglas (`domain`), casos de uso (`application` o `usecases`), contratos (`ports`), adaptadores HTTP/PostgreSQL y composición (`internal/app`). El registro de módulos y sus manifiestos conectan navegación y capacidades; el manifiesto por sí solo no instala SQL, registra una ruta funcional ni concede permisos. El cliente web estático consume la API; `cmd/vec-server` también contiene operaciones CLI concretas de importación, constitución y comprobación. Esta separación permite reutilizar casos de uso desde otros clientes cuando se compongan sus adaptadores; **no acredita hoy una aplicación de escritorio completa ni una CLI para todo VEC**. Véanse el [contrato modular vigente](../portal_vec/contrato_modulos_vec.md#criterio-vigente-de-ampliación--19-de-septiembre-de-2026), [módulos](../../internal/modules/) y [composición](../../internal/app/bootstrap/).

| Autoridad | Datos y frontera visible en este corte |
| --- | --- |
| Contratación temporal | Expediente, versiones, actuaciones, recibos y seguimiento; coordina por puertos y referencias opacas. No escribe las tablas de Bolsa o Personal. |
| Bolsa | Convocatorias, candidaturas, posiciones, disponibilidad, llamamientos y su historia. Sus reglas de ejemplo son catálogos modificables y están rotuladas como provisionales. |
| Personal | Persona, vínculo de empleado, relaciones y ocupaciones según sus contratos; una cuenta o certificado no demuestra una relación de servicio. |
| Documentos, Calendarios, Cronos y Dietas | Conservan sus contratos y datos propios. La presencia de código, manifiesto o migración no demuestra que todas sus capacidades estén activadas o recorridas en cidonia. |
| Núcleo VEC | Identidad/contexto, autorización V3, auditoría, catálogo de módulos, i18n y tema comunes. Los módulos no crean un segundo login o punto de decisión. |

Las ventajas comprobables son reglas independientes del transporte, ampliación por módulos registrados, catálogos versionados, recibos recuperables e historia de solo adición en los cortes que la implementan, y denegación ante falta de autoridad. Cada ventaja se evalúa por **capacidad compuesta y probada**, no por la presencia de una carpeta. La [fuente RRHH](../portal_vec/expediente_contratacion_temporal_rrhh.md) conserva el recorrido completo y el [estado](../../ESTADO_PROYECTO.md) separa integrado, instalado, recorrido y pendiente.

## Fronteras de seguridad y datos

- La autorización funcional V3 exige concesión central positiva, exacta y vigente para actor, acción, recurso, ámbito, finalidad, campos y obligaciones. RBAC es necesario; ABAC solo restringe. Menú, URL, cabecera, certificado, manifiesto o rol PostgreSQL no conceden por sí mismos una operación. El efecto revalida la decisión en su transacción. Véanse la [matriz de roles](../portal_vec/matriz_roles_y_ambitos.md) y [E04/E06](../../ESPECIFICACIONES_AGENTES.md).
- PostgreSQL añade cuentas técnicas separadas, privilegios y ACL específicos, funciones propietarias y controles de transacción. Comparar privilegios **efectivos**, incluidos `PUBLIC` y membresías, antes y después de una migración. Las migraciones instaladas tienen historia: no repetir `UP` ni usar `DOWN` sobre datos conservados. La [política SQL](../portal_vec/seguridad_persistencia_postgresql.md) define la frontera; cada esquema debe comprobarse contra su postimagen real.
- Los secretos, DSN, certificados, claves de cifrado y configuración de identidad se custodian fuera de Git, con permisos restringidos y rotación por la autoridad correspondiente. Los nombres de variables en [config](../../config/) son contratos, no valores de despliegue. No volcar material privado en argumentos, logs, tickets ni esta documentación. La web de VEC no guarda credenciales en cookies o almacenamiento local.
- El acceso público, el interno y la administración tienen fronteras distintas. El perfil de presentación sintético no sustituye identidad institucional ni convierte autenticación en firma. La verificación con AutofirmaV2 descrita en el [estado](../../ESTADO_PROYECTO.md) usó una autoridad de desarrollo; no otorga validez legal. La raíz de [vec-interno](../../cmd/vec-interno/main.go) exige dependencias institucionales concretas; la existencia del binario no demuestra que estén aprovisionadas.

## Artefacto web, arranque y comprobación

1. Antes de intervenir, registrar commit fuente (`git rev-parse HEAD`), cambios (`git status --short --branch`), huellas del binario y activos **servidos**, versión de PostgreSQL, migraciones instaladas y estado de servicios. El commit del checkout no identifica por sí solo el binario en memoria. Conservar inventario y preimagen por el canal privado.
2. Construir el binario de la revisión aprobada con la versión de Go de [go.mod](../../go.mod); empaquetar `web/` de **esa misma revisión**. [Producción](../../web/produccion.manifest), [interno](../../web/interno.manifest) y [público](../../web/publico.manifest) son listas de activos por superficie. El servidor normal usa una lista positiva derivada del manifiesto de producción: si falta o es inválido, cierra los estáticos. El artefacto de presentación tiene composición propia. Verificar cada import HTML/JS/CSS y que no se mezclan activos de otro commit.
3. Verificar fuera de Git identidades técnicas, certificados, CA, conexión TLS, política V3 vigente, catálogos publicados y dependencias obligatorias del perfil. Encender un selector solo cuando sus datos, SQL y autoridad estén instalados. Una variable presente o un `200` de portada no acreditan un caso de uso.
4. Arrancar únicamente perfil y servicios autorizados para el entorno. Comprobar `/livez` y `/readyz` (`/healthz` es alias histórico), después lectura autorizada por módulo y denegaciones. Para afirmar persistencia, recuperar el **mismo** recibo y versión tras reiniciar aplicación y PostgreSQL, sin duplicar historia. Los `POST` no son sondas: solo repetir con la clave original de un contrato idempotente acreditado.

[deploy/principal](../../deploy/principal/) conserva ensambladores y guiones de cortes de la réplica sintética. En particular, [desplegar.sh](../../deploy/principal/desplegar.sh) cambia a `main`, hace `pull`, detiene la aplicación y ejecuta una cadena SQL de preimagen histórica. **No es un actualizador genérico** para una base con migraciones posteriores: inventariar y comparar la preimagen exacta antes de usar cualquier tramo. [verificar.sh](../../deploy/principal/verificar.sh) hace lecturas mTLS de casos sintéticos existentes; sus respuestas no demuestran escrituras, entrega de correo ni recuperación. Los procedimientos concretos de operación y configuración permanecen fuera de Git.

## Copia y restauración ensayada

Antes de cambiar binario, activos o esquema, preparar una copia consistente fuera de Git con el responsable de la base. Registrar punto temporal y huellas; incluir el volcado de la base, roles globales, ACL de base y objetos, tipos de fila, extensiones, historia de migraciones y referencias a material criptográfico. Custodiar por separado credenciales, claves y certificados necesarios: `pg_dumpall --globals-only --no-role-passwords` no los incluye. Comprobar legibilidad del archivo con `pg_restore --list` y conservar sus SHA-256.

Ensayar restauración en un clúster PostgreSQL **nuevo y aislado**, con versión compatible: restaurar primero roles globales y después la base, detenerse ante cualquier error y cotejar roles, membresías, ACL efectivas, tipos, extensiones, migraciones, recuentos e historia. Reponer secretos solo por el canal privado. Arrancar allí el artefacto compatible y recuperar recibos sintéticos con pruebas de denegación; impedir emisiones externas durante el ensayo. Una copia de tablas no restituye por sí sola las autoridades V3. No sobrescribir una base viva con un volcado anterior si contiene efectos posteriores; volver a un binario compatible tampoco revierte SQL confirmado.

## Observabilidad y diagnóstico

`cmd/vec-server` compone un emisor de incidencias técnicas JSON Lines y supervisión de respuestas HTTP 5xx, pánicos y fallo de arranque. Los códigos y la correlación son opacos; el middleware sanea el registro de `net/http` y omite URL, consultas, cabeceras y cuerpos. Ese flujo técnico **no sustituye** la auditoría funcional y de accesos. Revisar además recibo, versión, auditoría y outbox del módulo afectado con permisos adecuados. Véanse [incidencias_arranque.go](../../cmd/vec-server/incidencias_arranque.go) y [supervision.go](../../internal/app/server/supervision/supervision.go).

| Síntoma | Primera comprobación | Acción segura |
| --- | --- | --- |
| No arranca o `readyz` devuelve 503 | Código de incidencia, perfil, selector, postimagen SQL y dependencia obligatoria. | Corregir causa; conservar binario y base previos. No regenerar claves ni repetir migraciones. |
| `401` o fallo TLS | Canal, certificado, emisor, vigencia y revocación de identidad; configuración mTLS. | Verificar frontera de confianza. No aceptar cabeceras libres para eludirla. |
| `403` | Actor, perfil, concesión V3, ámbito, finalidad, campos, obligaciones y ACL efectiva. | Publicar concesión correcta por su circuito; no ampliar comodín. |
| `409` o reintento | Versión esperada, clave semántica y recibo previo. | Recuperar operación original; no fabricar otra clave para el mismo acto. |
| `5xx` o lectura ausente | Correlación saneada, conexión nominal, postimagen SQL y estado de auditoría/outbox. | Aislar capa y comparar preimagen; no usar un `POST` de prueba ni revertir historia. |

La instancia de cidonia es una **presentación sintética**. Correo corporativo, firma legal, plazos normativos, custodia externa, activación de módulos y acceso institucional se declaran solo donde exista su recorrido y aprobación específicos. Consultar el [seguimiento único](../../ESTADO_PROYECTO.md) antes de afirmar un despliegue nuevo.
