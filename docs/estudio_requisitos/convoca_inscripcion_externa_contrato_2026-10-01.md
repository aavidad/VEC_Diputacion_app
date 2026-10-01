# Inscripción externa de CONVOCA: contrato pendiente

Estado: preparación de la solicitud, sin presentación disponible. Este contrato
orienta la pantalla de preparación y el futuro caso de uso de Bolsa. No implementa
interfaces Go, HTTP, SQL, firma, registro electrónico ni recibos.

## Fuentes y límites

- [Ficha CONVOCA](ficha_adaptacion_bolsa_convoca_2026-09-16.md), S2, S4 y B15:
  inscripción, documentación, tasas, consulta propia y requisitos de acceso.
- [Portal público observado](referencia_convoca_publica_2026-09-23.md): acceso
  identificado, interesado o representante y «Mis inscripciones».
- [Estudio de datos personales](datos_personales_inscripciones_rgpd_2026-09-29.md),
  apartados 2 y 4: minimización y separación entre población externa e interna.
- [Especificaciones](../../ESPECIFICACIONES_AGENTES.md), E02–E07 y E10–E12:
  autoridades únicas, autorización central, historia y evidencia verificable.

Falta confirmar el formulario y los anexos exactos de una convocatoria de Granada,
su versión publicada y los campos y declaraciones que exige. Los manuales de Cádiz
y Mérida sirven de referencia; sus campos, plazos y reglas no se adoptan como norma
de Granada. Tampoco está acreditado aquí el circuito de Sede, pago o representación.

## Piezas existentes y responsabilidades

| Pieza | Encaje y límite |
| --- | --- |
| `internal/candidate/ports/procedure.go`: `SolicitudRecord` y `SolicitudRepository` | Contrato del núcleo existente; contiene candidatura, estado y baremación. No acredita inscripción externa. |
| `internal/candidate/usecases/procedure.go`: `RegistrarSolicitud` y `buildSolicitudRecord` | Recibe candidato y méritos, calcula baremación y guarda. No se monta como presentación propia del aspirante. |
| `internal/modules/bolsa/ports/convocatorias_publicas.go` | Consulta del detalle público canónico; su lectura no autoriza la inscripción. |
| `internal/modules/bolsa/domain/convocatoria_gobernada.go` | Conserva referencias de configuración con identidad, versión y huella. Se reutilizará la publicación gobernada exacta. |
| Bolsa | Posee la inscripción y su estado e historia; consulta datos por puertos autorizados. |
| Aspirantes | Posee la ficha externa propuesta en el estudio; su información se obtiene por finalidad. Este contrato no acredita su implementación. |
| Documentos | Posee la custodia y la recuperación autorizada de evidencias. Bolsa conserva referencias. |

La inscripción no exige vínculo de empleado ni copia la ficha de Personal. El
navegador no elige actor, aspirante, perfil, permisos, finalidad o población.

## Operaciones futuras

Los siguientes nombres describen un contrato propuesto, sin rutas HTTP asignadas:

| Operación | Entrada neutral | Resultado previsto |
| --- | --- | --- |
| `Presentar` | Contexto acreditado del servidor y `ComandoPresentarInscripcion` | `ResultadoPresentacion`, solo tras confirmar el efecto duradero y las evidencias exigidas por el circuito admitido. |
| `ConsultarPropia` | Contexto acreditado del servidor y referencia opaca de inscripción | `InscripcionPropia`, limitada a campos autorizados de esa persona. La referencia por sí sola no da acceso. |

El contexto procede de identidad y autorización centrales, con superficie externa,
vínculo acreditado y concesión positiva, exacta y vigente para cada operación.
Se revalida en consulta y recuperación. Nunca se deserializa de campos del formulario.
La representación necesitará su acreditación específica antes de habilitarse.

| Tipo propuesto | Campos mínimos y procedencia |
| --- | --- |
| `ComandoPresentarInscripcion` | Publicación exacta, clave de idempotencia, versión esperada de la solicitud o precondición de ausencia, referencia versionada del formulario confirmado y referencias de evidencias custodiadas. |
| `ReferenciaPublicacion` | `identificador_publico` canónico, referencia de publicación, versión y huella de su contenido. Bolsa resuelve y verifica estos valores; no acepta una revisión enviada como autoridad por el cliente. |
| `ReferenciaEvidencia` | Referencia opaca de documento, versión, representación y huella, con evidencia de custodia verificable en Documentos. Sin bytes, rutas, nombres de fichero o datos personales en este contrato. |
| `ResultadoPresentacion` | Referencia de inscripción, versión, instante acreditado, publicación exacta y referencia del recibo duradero existente; recuperación de la misma operación sin generar otro recibo. |
| `InscripcionPropia` | Referencia, estado, versión, publicación exacta y referencias autorizadas de evidencias y recibo, cuando existan. Estado de inscripción y admisión administrativa son distintos. |

Los campos y declaraciones del formulario quedan pendientes de su fuente exacta;
no se sustituyen por un mapa libre de datos personales. Los hechos de Aspirantes y
las evidencias de Documentos se resuelven mediante sus autoridades y permisos.
Una evidencia custodiada no acredita por sí sola firma, pago ni asiento registral.

## Condiciones de implementación y pantalla

1. Verificar publicación, versión, huella, plazo y formulario admitido con el instante
   del servidor. Una fuente de demostración no habilita una presentación real.
2. Evaluar los requisitos según las bases versionadas. Un dato pendiente no equivale
   a incumplimiento ni a admisión; no bloquea automáticamente si las bases permiten
   declarar o completar ese requisito. La previsión de titulación exige la excepción
   explícita de OPE y su hito; no habilita una bolsa de incorporación inmediata.
3. Vincular la clave de idempotencia al contenido canónico y al contexto acreditado.
   Misma clave y contenido recuperan el mismo efecto; un contenido distinto da
   conflicto. La versión esperada evita sobrescribir cambios posteriores.
4. Confirmar estado, autorización consumida, versión, historia, auditoría y salida de
   eventos requerida en la transacción duradera. No crear eventos de borrador ni
   fabricar firma, número de registro, pago o recibo para mostrar éxito.
5. Mantener «Presentar» deshabilitado mientras falten las dependencias del circuito.
   La pantalla puede preparar la solicitud, explicar qué falta y volver al detalle
   público; no anuncia inscripción presentada ni conserva datos en almacenamiento web.

Se reserva `bolsa_solicitudes 000001` para el borrador de persistencia externa
segregada (B15, S2/S4), en el registro de reservas fuera de Git. Todavía no hay SQL
ejecutable. El diseño e instalación siguen pendientes del formulario gobernado,
revisión independiente y ensayo autorizado. La cadena va después de H6 y de los
pasos 0→B→A→M de `ORDEN_SQL_NUCLEO.md`; un consumidor nuevo de autorización requiere
su propia reserva y turno de Dirección/D. Este corte no cambia H6 ni trata datos
reales. La aceptación futura exige navegador hasta recibo y recuperación tras
reinicio, con historia conservada y sin duplicar la inscripción.
