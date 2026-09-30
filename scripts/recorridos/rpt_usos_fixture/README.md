# Ejercicio sintético de usos RPT con V3

Base del encargo: `b8bd755b70399fab6627d12858e4dc90ade6b7d3` (PR #222).
La plantilla conserva datos de entrada pendientes. No contiene credenciales ni
material criptográfico y no autoriza ninguna operación.

La entrega incluye guardas, una composición optativa de V3 y un coordinador
del recorrido. El resultado operativo sigue siendo **NO-GO** hasta obtener
las dos revisiones sensibles del candidato y el READY del clon. No se ha
ejecutado provisión, COSE de consumo real ni un positivo PostgreSQL.

El ejercicio se limita al clon local en loopback. Su DSN y puerto proceden de
la configuración privada y se cotejan con la huella del clon H6; el puerto
del servidor MCP de consultas no identifica por sí solo el clon de recorridos.
Su ejecución espera
la confirmación de H6, AD3-132 y las seis fachadas RPT por el responsable del
clon, además de la revisión de la composición del fixture. No se instala SQL,
no se inicia una aplicación y no se genera una identidad al cargar el fixture.

## Dependencias comprobables antes de provisionar

El responsable del clon debe acreditar estas seis funciones, su firma de once
argumentos y sus ACL efectivas. Un nombre presente no acredita la instalación
correcta, su gobierno criptográfico ni su disponibilidad para el LOGIN elegido.

| Fachada en `vec_autorizacion_atestada_v3` | Contrato |
| --- | --- |
| `listar_categorias_habilitadas_rpt_v3_atestada` | Lectura AD3-117 |
| `leer_publicacion_categoria_rpt_v3_atestada` | Lectura AD3-117 |
| `consultar_uso_categoria_rpt_v3_atestada` | Lectura AD3-117 |
| `reservar_uso_categoria_rpt_v3_atestada` | Uso AD3-126 |
| `confirmar_uso_categoria_rpt_v3_atestada` | Uso AD3-126 |
| `cancelar_uso_categoria_rpt_v3_atestada` | Uso AD3-126 |

Firma común: `(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)`.
La comprobación de H6/AD3-132 es adicional. Este documento no acredita que esas
migraciones estén instaladas y no incluye instrucciones para reaplicarlas.

## Identidad y concesiones

Una CA de desarrollo y su identidad sintética deben estar ya provisionadas en
un directorio privado ajeno a Git. La composición debe verificar la asociación
entre certificado, identidad, cuenta, persona, perfil y sesión mediante la
autoridad existente. Las claves y la CA nunca se copian desde el perfil CT.

El LOGIN del ejercicio pertenece directamente a exactamente una familia:
`vec_contratacion_temporal_ejecutor` si el consumidor fijado es
`contratacion_temporal`. AD3-126 exige una sola pertenencia, sin `ADMIN OPTION`,
con `INHERIT` y sin `SET`, y rechaza otras pertenencias VEC o familias heredadas.
La provisión de identidad, el gobierno de claves y la ejecución funcional usan
capacidades separadas. El LOGIN ejecutor no publica su propia concesión.

El perfil fijo `usos_categorias` debe publicarse mediante la autoridad AD3,
con tres acciones exactas, sin ampliar los perfiles CT. Una sustitución requiere
aprobación nominal de la huella de la asignación vigente y CAS bajo bloqueo.
Una revocación, restricción ajena, caducidad o retirada del rol deniega. Cada
operación consume la asignación publicada; no publica permisos por petición.

| Campo de la operación | Valor fijado por composición |
| --- | --- |
| Audiencia de consumo | `vec_catalogos_configurables.usos_categorias.v1` |
| Acciones | `vec.catalogos.categorias.reservar_uso`, `vec.catalogos.categorias.confirmar_uso`, `vec.catalogos.categorias.cancelar_uso` |
| Tipo de recurso | `uso_categoria` |
| Finalidad | `vincular_categoria_a_operacion` |
| Campos permitidos | `recibo`, `uso` |
| Obligaciones | Lista vacía |
| Ámbitos exactos | `catalogo_id`, `modulo_id`, `consumidor` |
| Atributo exacto | `material_sha256`, obtenido mediante `PreparadorUsosCategoriaRPT` |

La audiencia de despliegue del sobre COSE procede del gobierno de confianza.
Es un compromiso distinto de la audiencia de consumo anterior. El fixture no
construye bytes de decisión, capacidades, evidencia de confianza ni material
exportado a partir de JSON.

## APIs que se reutilizan

- `ports.PreparadorUsosCategoriaRPT` obtiene el recurso del material exacto
  mediante PostgreSQL. El cliente no imita `jsonb::text` ni calcula otra huella.
- `domain.NuevaSolicitudAutorizacionLigadaV3` liga el vínculo registrado,
  motivo publicado, correlación, acción, finalidad y recurso preparados.
- `confianzaatestacion.NuevoEmisorMaterialAutorizacionAtestadaV3` encadena
  autorización central, registro durable, atestación COSE, verificación de
  confianza, capacidad breve y exportación nominal. Un fallo de cualquiera
  de estas fases impide el efecto RPT.
- `postgres.NuevoGestorUsosCategoriaRPTPostgreSQL` fija descriptor y consumidor
  desde composición; sus tres métodos vuelven a comprobar el material y
  consumen V3 en la transacción del recibo.

La provisión fija existente en
`internal/app/bootstrap/contratacion_temporal_perfiles_fijos_rrhh_desarrollo.go:282`
está ligada al soporte y los actos de CT. La autoridad común de publicación con
CAS está en `internal/app/bootstrap/autoridad_postgresql_desarrollo.go:179`.
La composición RPT debe reutilizar esa autoridad con actos propios y una
plantilla fija. El fixture usa `autoridadPostgreSQLDesarrollo` con actos RPT
propios, sin llamar al asegurador de perfiles CT.

La modificación de emisión se limita a
`internal/app/bootstrap/contratacion_temporal_postgresql_gobierno_desarrollo.go`:
`audienciasConsumoGobiernoCTDesarrollo()` admite la audiencia exacta de usos RPT.
La misma lista protege el publicador y el reconocimiento del puntero de gobierno
propio. El archivo se modificó tras su cesión exclusiva; la revisión sensible
del candidato final sigue pendiente. No se llama al helper
inferior `publicarGobiernoAtestacionCTEnTxDesarrollo` para eludir la guarda.

La composición optativa está en los archivos nuevos
`catalogos_rpt_usos_fixture_*`: derivación nominal con dominio y prefijo RPT
propios, publicación mediante el publicador completo, contexto con discriminador
RPT propio y publicación real por `publicarResultadoContextoPostgreSQLDesarrollo`,
perfil fijo con actos RPT propios y CAS, y tres pools separados de gobierno,
registro de decisiones y ejecución. La verificación del certificado contra la
CA de desarrollo es una comprobación separada; el contexto sintético no
acredita autenticación mTLS de una persona. La cuenta y persona conservan su
procedencia común; perfil, vínculo, registro, autenticación y sesión quedan
separados mediante el discriminador propio del fixture.

`PlanificarPerfilRPTUsosFixtureV3(cfg, descriptor)` obtiene los compromisos de
la plantilla inicial sin abrir PostgreSQL ni publicar. El operador aprueba su
huella antes de provisionar. Para sustituir una asignación ya publicada, la
huella aprobada debe ser la de esa preimagen vigente, obtenida por su autoridad.

`NuevoRPTUsosFixtureV3(ctx, cfg, configuracion, validadorMotivos)` crea la
composición privada. Recibe el validador nominal de motivos ya configurado,
igual que CT usa `NuevoValidadorReferenciaMotivoPostgreSQLV2` con el pool
`motivos_autorizacion` en su composición nominal. Si ese puerto o su configuración
aprobada faltan, la construcción deniega; este fixture no crea un LOGIN nuevo
para suplirlos. Los métodos `Preparador()`, `Gestor()` y `Emitir(...)` conectan
con `rptusosfixture.NuevoRecorrido(...)`; `Cerrar()` libera los tres pools propios.
No hay montaje HTTP ni ampliación de permisos por una ruta pública.

La composición comprueba el destino y los tres LOGIN antes de escribir. Lee
la asignación vigente y admite su preimagen antes de publicar gobierno/contexto;
repite la admisión al provisionar y conserva el CAS del publicador común.
El preflight del ejecutor sólo lee metadata de PostgreSQL y exige su familia
única, las seis firmas RPT y sus permisos de ejecución. El resto de READY
procede de las evidencias privadas aprobadas del clon, no de esos nombres SQL.

## Recorrido previsto cuando la composición esté revisada y el clon esté READY

1. Conservar fuera de Git la aprobación, preimagen de la asignación y evidencia
   de las dependencias. Acreditar una categoría habilitada, la publicación exacta
   y el descriptor gobernado. Inventariar la historia anterior del clon.
2. Preparar la configuración privada a partir de `plantilla.json` y los DTO
   tipados de bootstrap. La plantilla es una lista de datos pendientes, no un
   codec de capacidades V3 ni una orden ejecutable. Construir el fixture con
   el validador de motivos existente y provisionar el perfil mediante su autoridad.
   operativa. En las operaciones posteriores, consumirlo sin republicarlo.
3. Preparar la primera reserva, solicitar V3 real y reservar. Conservar el uso,
   recibo de reserva, versión, estado y fecha devueltos por RPT.
4. Conservar fuera de Git un documento sintético con referencia
   `fixture:rpt:terminal:...` y su SHA256, rotulado «entrada de ensayo».
   Preparar el material terminal, emitir una V3 nueva y confirmar.
   Conservar el recibo terminal y las dos fechas. Este testigo prueba la ligadura
   del material; no acredita un efecto de CT, Personal o Bolsa.
5. Preparar otra reserva con otro `uso_ref` y recibo. Conservar la entrada de
   ensayo de cancelación, emitir una V3 nueva y cancelar.
6. Repetir cada operación con el mismo material y una V3 fresca. El consumo de
   autorización y su auditoría son nuevos; el uso, recibos, fechas y revisión
   de negocio permanecen idénticos. No reutilizar una capacidad consumida.
7. El responsable reinicia únicamente los servicios autorizados del clon.
   Repetir el mismo recorrido con otra V3 fresca y cotejar los recibos, fechas,
   publicación, estado y revisión. No aumentar la historia de negocio ni borrar
   la auditoría nueva de los accesos.
8. El responsable de persistencia compara la historia de negocio y las huellas
   anteriores mediante su canal de lectura autorizado. El ejecutor no consulta
   tablas de otros módulos ni dispone de SQL de reparación.

La recuperación por `LectorCategoriasRPT` usa la audiencia distinta
`vec_catalogos_configurables.lectura_categorias.v1`, sus tres acciones y un
perfil fijo de lectura que ya esté provisionado. Este ejercicio no añade ese
perfil: si falta, el responsable de persistencia coteja los usos y la historia
por su canal autorizado de ensayo. `usos_categorias` no concede lectura implícita.
La evidencia terminal sintética no acredita una operación
real de CT, una firma legal ni una entrega administrativa.

El retiro del ejercicio conserva la historia: revocar su identidad y sus
asignaciones mediante sus autoridades, retirar el material privado y el código
del fixture. Nunca ejecutar `DOWN` ni borrar usos o decisiones para retirarlo.

## Validación del subcorte de guardas

Con Go 1.26.6, las siguientes comprobaciones terminaron correctamente:

```text
gofmt -l internal/app/bootstrap/catalogos_rpt_usos_fixture_guardas*.go
go test -p 32 ./internal/app/bootstrap -run '^TestRPTUsosFixture' -count=1
go test -race -p 32 ./internal/app/bootstrap -run '^TestRPTUsosFixture' -count=1
go vet -p 32 ./internal/app/bootstrap
gopls check catalogos_rpt_usos_fixture_guardas.go catalogos_rpt_usos_fixture_guardas_test.go
git diff --cached --check
```

`gopls check` se ejecutó desde `internal/app/bootstrap`. Go y las herramientas
de análisis se ejecutaron en un sandbox bwrap sin red, con fuentes y módulos
de sólo lectura, entorno vacío con variables permitidas y escritura limitada
al scratch del encargo. Se fijaron límites de CPU, memoria, procesos, tamaño de
ficheros y tiempo; gopls necesitó limitar el heap tras agotar su primer intento.

Semgrep pasó cinco reglas Go locales de `semgrep-local.yml` sobre los dos
archivos nuevos, con métricas desactivadas, dos trabajos y sin red: cero
hallazgos. Son reglas focales de este ejercicio, no la campaña completa
`p/golang`. Gosec examinó el paquete y no encontró incidencias en los archivos
nuevos; conserva 55 avisos de código anterior y no notificó errores de análisis.
Las pruebas con PostgreSQL, provisión real, COSE real, replay y recuperación
tras reinicio siguen pendientes. No se ejecutó la puerta global.
