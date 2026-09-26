# Ficha de adaptación de VEC Bolsa a CONVOCA — 16 de septiembre de 2026

Decisión de Alberto (16/09/2026): VEC sustituye a CONVOCA, con importación desde CONVOCA
al menos para la carga inicial. Esta ficha traduce las fuentes disponibles en requisitos
numerados, dice qué hay en VEC para cada uno y qué cambio haría falta. No es una orden de
trabajo: Bolsa está aparcada hasta cerrar el orden de correcciones de Contratación.

## Estado al 26 de septiembre de 2026

Bolsa queda cerrada para la presentación a RRHH del 28/09/2026: integrada en `main`
(PR #66 a #72) y desplegada en la principal de cidonia, con «Mi bolsa» y el portal del
candidato encendidos (`VEC_BOLSA_PORTAL_CANDIDATO_ENABLED`). La tabla de requisitos de abajo
conserva su foto del 16/09; el detalle del cierre, la evidencia y los pendientes están en
`ESTADO_PROYECTO.md`, «Bolsa y Contratación temporal cerradas para la presentación — 26 de
septiembre de 2026». Las reglas que RRHH aún no ha decidido van en el paquete de ejemplo
modificable `data/demo/reglas/*.demo.json` (marca `paquete:ejemplo:vec:v1`). Producción sigue
en `NO-GO`.

## Fuentes

- **P** — Pliego de Prescripciones Técnicas SE 15/2020, §1 a) y c), páginas 2–3
  (`docs/convoca_dipgra/01_Pliegos_Oficiales/`, capturas en `06_Capturas_Pliego_Oficial/`).
  Es la descripción oficial de lo que CONVOCA hace en la Diputación. CONVOCA forma parte de
  Ginpix 7 (SAVIA): el mismo proveedor y la misma suite que el GINPIX al que RRHH quiere
  trasladar los nombramientos.
- **M** — Manual de bolsas de empleo de Mérida (`04_Manuales/Manual_Bolsas_Empleo_Convoca_Merida.pdf`).
  Misma plataforma; describe pantallas y operaciones de listas de sustitución. Sus **reglas**
  (3 llamadas, 24 horas, larga/corta duración, +6 meses) son de Mérida, no de Granada: sirven
  como ejemplo de parametrización, no como norma.
- **C** — Guía de solicitudes de Cádiz (`04_Manuales/Manual_Presentacion_Solicitudes_Convoca_Cadiz.pdf`):
  inscripción con tasas, documentación y registro.
- **R** — `Peticion.pdf` del Servicio de selección externa de la Diputación.
- **W** — Word de RRHH (paso 5, llamamiento).

## Requisitos de gestión de bolsas (fase 1)

| Nº | Requisito | Fuente | Qué hay en VEC (`a1bc66b7`) | Cambio necesario |
| --- | --- | --- | --- | --- |
| B1 | Registro de todos los candidatos que pasan a una bolsa desde una OEP o desde una convocatoria específica de bolsa. | P c.1 | `BolsaConstituida` y `ParticipacionBolsa` en `bolsa/domain/llamamientos.go`; importador de CONVOCA (`adapters/xlsconvoca`, `postgresimportacionconvoca`, 2.289 líneas) sin ningún servidor ni CLI que lo invoque. | Comando `cmd/vec-bolsa-importar` que ejecute el importador sobre los XLS de CONVOCA con recibo y resumen; alta de participaciones desde el listado definitivo de una convocatoria propia (fase 2). |
| B2 | Estado o situación del candidato en cada momento. | P c.2; M §8; R p.2 | `SituacionParticipacionBolsa` con `estado_clave` de catálogo (`disponible`, `ocupado`, `no_disponible`, `excluido`, `renuncia_pendiente`), `desde`/`hasta`, y tres SHA-256 por situación. Sin pantalla. | Catálogo cerrado y visible: Disponible, No disponible (pausa), Trabajando, Pendiente de incorporación, Renuncia, Excluido, Disponible el dd/mm/aaaa. Pantalla de gestión. Retirar las huellas de situación que no sustenten evidencia (consenso, cargo 4). |
| B3 | Histórico de contactos realizados por correo y por teléfono, con anotaciones de cada contacto. | P c.2; M §4 | No existe. | Entidad Contacto ligada al llamamiento: canal, fecha y hora, usuario, resultado, anotación. Es lo que RRHH llama «histórico» en R. |
| B4 | Datos de contacto: correo electrónico y teléfonos. | P c.2; R p.1 | Correo obligatorio del alta VEC (`contacto_usuario_vec`, cifrado). Teléfonos: no. | Dos teléfonos en el contacto del candidato, con el mismo tratamiento que el correo. |
| B5 | Filtrar disponibles y no disponibles; buscar por nombre, apellidos o DNI. | P c.3, c.4 | `/api/vec/bolsa/panel` declarado y sin montar; sin búsqueda. | Pantalla de bolsa: lista ordenada, filtro por estado, búsqueda; DNI enmascarado salvo permiso. |
| B6 | Lista ordenada según los criterios del reglamento de aplicación; listas cerradas y rotatorias; reposicionamiento tras contrato. | P c.5; M §4–5; R p.2 | `InstantaneaOrdenBolsa` que se le entrega al servicio; `ProponerPrimerLlamamiento` / `ProponerSiguienteLlamamiento`. El orden no se calcula desde la bolsa. | Orden calculado desde la bolsa por regla parametrizada (puntuación, tipo de lista, reposición tras contrato: misma posición / fin de lista / no disponible hasta fecha). Parámetros: respuesta de RRHH (dudas 13 y 14). |
| B7 | Llamamiento con constancia: canal, intentos, horario, qué vale como renuncia, plazo de respuesta. | M §4; W paso 5; R p.3 | Llamamiento con «aviso local»; aceptación/renuncia manual sintética; SMTP escrito y no compuesto. | Registro de intentos por canal (B3) + correo real + SMS si Sistemas lo admite; plazo y efectos: respuesta de RRHH (dudas 1–3). |
| B8 | Suspensión temporal voluntaria (pausa) manteniendo la posición, reactivación y causas de baja con justificante. | M §6–7 | Estados sí; operaciones no. | Operaciones pausar / reactivar / excluir con motivo, justificante y quién lo valida. |
| B9 | Vigencia: la bolsa entra en vigor con el listado definitivo y sustituye a las anteriores de la misma categoría. | M §2 | `VigenteDesde` / `VigenteHasta` en `BolsaConstituida`. | Sustitución automática de la bolsa anterior de la categoría al constituir la nueva. |
| B10 | Consulta pública de la lista de cada bolsa, sin identificarse: orden, estado y DNI enmascarado. | M §8, §10; R p.2 | Público solo convocatorias y categorías (`cmd/vec-publico`, no arrancado). | Ruta pública por bolsa con los campos que apruebe RRHH (dudas 17). |
| B11 | Portal personal del candidato: su posición, estado, contratos, último llamamiento, disponibilidad; pausar / reactivar. | R p.2; M §7 | Dominio, caso de uso V3, adaptador PostgreSQL y handler `GET /api/vec/bolsa/mi-bolsa` hechos (23/09); **sin componer**: `validarOrden` exige `SuperficieAutenticacionExternaPersonalV1` y un vínculo de candidato, y esa frontera no está construida. | Construir el acceso del candidato con DNIe o certificado (decisión acordada 4: es desarrollo nuestro, no una duda de RRHH), ficha propia y las operaciones de B8 desde el portal. |
| B12 | Cuadro de control: bolsas activas, candidatos por bolsa y por estado. | R p.3 | No. | Contadores sobre B2, sin más mecanismo. |
| B13 | Envío de correo por estado y avisos automáticos (candidato saltado; tres años trabajando). | R p.3; fabricante: «aviso cuando llega el turno» | No. | Después de B7; parámetros: RRHH (dudas 13). |
| B14 | Documentación y plazo para formalizar tras aceptar (M: 24 horas). | M §9; W paso 6 | Enlaza con Contratación (nombramiento). | Plazo y lista de documentos: RRHH (dudas 9 y 18). |
| B15 | Evaluación personal de oportunidades y acceso directo a la solicitud. | Decisión del operador, 20/09/2026 | El área personal enumera convocatorias DEMO y la web pública muestra su detalle y requisitos, pero los requisitos son texto libre y no hay cotejo con el expediente de méritos ni enlace personalizado real. | Estructurar y versionar requisitos de acceso; contrastarlos con datos declarados o acreditados y mostrar `cumple`, `no_cumple` o `pendiente`. En una bolsa inmediata, la titulación futura no cumple. En cada requisito de titulación de una OPE, RRHH puede activar antes de publicar «Permitir inscripción con titulación pendiente», indicando obligatoriamente hito y evidencia; queda desactivado por defecto y su cambio posterior requiere nueva versión o rectificación. Abrir el detalle por `identificador_publico`, iniciar la solicitud precompletada y revalidar el requisito. |

## Requisitos del proceso selectivo (fase 2, para dejar de usar CONVOCA)

| Nº | Requisito | Fuente | Qué hay en VEC | Cambio necesario |
| --- | --- | --- | --- | --- |
| S1 | Alta del proceso selectivo y publicación. | P a) | Gobierno de convocatorias (borrador, versión, publicación) probado, no compuesto. | Componer en el servidor interno y en el público. |
| S2 | Inscripción telemática integrada con la Sede electrónica de la Diputación y con pasarela de pago. | P a); C | Contratos de tasas y pagos probados; ninguna integración con Sede ni pasarela. | Integración con la Sede (MOAD) y la pasarela: Informática. |
| S3 | Autobaremación de méritos. | P a) | Existe (legado y presentación). | Componer sobre la solicitud real. |
| S4 | Gestión de inscripciones: validación, subsanación, trámites del aspirante con certificado electrónico. | P a) | Parcial (subsanaciones y alegaciones en presentación). | Componer con identidad por certificado. |
| S5 | Carga de resultados de pruebas. | P a) | No existe. | Nuevo, pequeño: resultados por prueba y aspirante, con acta. |
| S6 | Comunicación con candidatos por SMS y correo electrónico. | P a) | No. | Compartido con B7. |
| S7 | Baremación y listado definitivo que constituye la bolsa. | P a); M §4.1 | Baremación firmada, cálculo de experiencia, reglas de baremo: completos y sin componer. | Componer; su salida alimenta B1. |

## Orden propuesto cuando Bolsa se retome

1. B1 importador invocable con los XLS reales (es la condición de Alberto y lo que alimenta a Contratación).
2. B2 + B5: estados y pantalla de bolsa. 3. B6 orden calculado con los parámetros de RRHH. 4. B3 + B7 + B8: contactos, llamamiento real, pausas. 5. B10 + B11: público y portal. 6. B15 sobre el portal y detalle público ya conectados. 7. B12 + B13. 8. Fase 2 (S1–S7) solo cuando haya una convocatoria nueva que tramitar.

## Lo que no se decide aquí

Reglas de Granada (reglamento de bolsas, tipos de lista, reposición, límites de encadenamiento, plazos y canal del llamamiento, campos públicos, documentación de formalización): `dudas.md`, preguntas 13–18. Exportación de CONVOCA (qué bolsas, formato, fecha de corte): pregunta 16.
