# O4-03 — diseño de propuesta y decisión de vía de cobertura

Fecha: 24 de julio de 2026.

Estado: diseño previo a implementación. O4-03 continúa abierta.

## Objetivo

Construir una propuesta automática reproducible a partir del catálogo
gobernado y de comprobaciones atestadas, y permitir que RRHH adopte o
rectifique una decisión humana motivada. La propuesta informa; nunca produce
por sí sola un efecto jurídico ni modifica el expediente.

O4-04 persistirá después la decisión, su autorización, auditoría, outbox y
recibo en una sola transacción. O4-05 conectará API y pantalla.

## Brechas confirmadas

O4-01 identifica vías, orden, obligatoriedad y procedencia, pero su canon V1 no
declara qué resultados habilitan cada vía ni cómo tratar una ausencia. Ese
canon ya publicado no se modifica.

O4-02 autentica y consume cada observación, pero su método público devuelve
solo `ComprobacionCobertura`. Para decidir con prueba suficiente se debe
conservar también petición, catálogo, atestación, verificación, orden de
consumo y recibo durable.

`DecisionViaCobertura` todavía no liga catálogo, propuesta, conjunto de
evidencias, política ni actuación. Tampoco existe rectificación segregada.

## Microcortes

| Corte | Resultado verificable |
| --- | --- |
| O4-03A | Política versionada y propuesta canónica, sin alterar O4-01 V1. |
| O4-03B | Evidencia completa y redactada de O4-02, con huella de conjunto. |
| O4-03C | Caso de uso de proponer, decidir y rectificar con autorización VEC V3, idempotencia y CAS contractual. |
| O4-03D | Pruebas adversariales, carrera y revisión independiente. |

No se crearán ficheros nuevos en `*/ports` mientras siga vigente DEC-092. Los
contratos nuevos nacerán en un subpaquete funcional de Contratación Temporal;
solo se ampliarán de forma compatible los contratos O4-02 ya existentes cuando
sea imprescindible conservar su evidencia.

## Política gobernada

La política de decisión debe quedar ligada a:

- organización y finalidad;
- identidad exacta del catálogo: referencia, versión y huella;
- referencia, versión, huella y vigencia de la propia política;
- por cada vía, comprobaciones previstas y resultados habilitantes;
- tratamiento explícito de la ausencia;
- prioridad para producir una recomendación determinista;
- motivo gobernado cuando la decisión humana se aparte de la recomendación.

La vía y sus condiciones son datos administrables. El núcleo solo conoce
estados técnicos cerrados y no contiene listas compiladas de Bolsa, SAE,
convocatoria u otras opciones futuras.

## Estados técnicos de propuesta

| Estado | Significado | Puede decidirse |
| --- | --- | --- |
| `viable` | Todas las condiciones exigidas se acreditan y al menos una vía es viable. | Sí, sobre una vía viable. |
| `incompleta` | Falta una comprobación cuyo tratamiento no permite ausencia. | No. |
| `conflictiva` | Existen observaciones incompatibles para la misma condición o procedencia. | No. |
| `sin_via` | El conjunto es completo y coherente, pero ninguna vía resulta habilitada. | No por el flujo ordinario. |

`no_aplica` es un resultado acreditado; no equivale a falta de datos.
Indisponibilidad, timeout y error privado tampoco equivalen a ausencia,
resultado negativo ni éxito.

## Propuesta

La propuesta canónica contiene, como mínimo:

- organización, expediente y versión analizada;
- referencia y huella del análisis durable de origen;
- identidades exactas de catálogo y política;
- huella global ordenada de los conjuntos de evidencias O4-02 de todas las
  vías, ligada al análisis durable y sin PII;
- evaluación de cada vía;
- vía recomendada cuando el estado es `viable`;
- referencia y huella reproducible de la propuesta;
- instante de generación y vigencia.

O4-02 liga cada petición, respuesta y orden a una vía concreta. Por tanto, una
comprobación compartida se consulta una vez por cada identidad O4-02 ligada a
vía; no se reutiliza una orden entre vías. Dentro de una misma vía, el replay
exacto se normaliza y una renovación con el mismo resultado funcional conserva
un único representante determinista. Claves con resultados funcionales
distintos producen conflicto.

La preparación global exige exactamente un conjunto por cada vía publicada,
rechaza vías omitidas, añadidas o duplicadas y liga:

- organización, expediente y versión;
- análisis durable y su huella;
- catálogo, política, finalidad, categoría y periodo;
- cada par ordenado `vía + huella del conjunto`;
- los resultados minimizados exactos utilizados por la propuesta.

Cambiar un byte de catálogo, política, análisis, resultado o evidencia cambia
la huella o invalida la preparación y la propuesta.

## Decisión humana

El cliente solo aporta la intención, referencias de autenticación, expediente,
versión, clave idempotente, propuesta, vía elegida y motivo funcional
gobernado cuando proceda. Actor, perfil, roles, unidad, catálogo, política,
resultados, evidencias, recibos y autoridad se resuelven en el servidor.

La decisión ordinaria exige:

- propuesta vigente y estado `viable`;
- vía elegida presente y viable;
- actor decisor distinto del proponente automático o humano cuando la política
  lo exija, sin posibilidad de rebajar la segregación;
- motivación obligatoria y motivo gobernado si se aparta de la recomendación;
- autorización VEC V3 exacta para
  `contratacion_temporal.cobertura.decidir`;
- CAS sobre la versión esperada;
- una orden opaca que O4-04 pueda confirmar sin reinterpretar la política.

Con los contratos VEC V3 existentes, O4-03C construye la solicitud ligada y
ejecuta `ExigirSolicitudLigadaV3` antes de fabricar la orden opaca. Esa
operación registra durablemente una concesión, igual que en el patrón O3. La
orden conserva solicitud, decisión y confirmación completas. O4-04 no abre una
transacción VEC anidada: consume esa concesión una sola vez dentro de la misma
transacción serializable que consume las evidencias y aplica el CAS y el
efecto de cobertura.

La idempotencia de O4-03C es contractual. La garantía durable frente a
reinicios, carreras, respuesta perdida y colisiones solo se cierra en O4-04.

## Rectificación

Rectificar crea una nueva versión y conserva la anterior. Exige propuesta nueva
o todavía vigente, motivo publicado, referencia a la decisión sustituida y un
actor distinto del decisor anterior. Se prohíbe cuando ya existe asignación o
un efecto posterior, salvo un procedimiento explícito de retroacción que queda
fuera de O4-03.

La autorización usa una acción distinta:
`contratacion_temporal.cobertura.rectificar`. El replay exacto devuelve el
mismo recibo; la misma clave con otra semántica es conflicto.

## Seguridad y minimización

- Denegación predeterminada y privilegio mínimo por operación.
- Ningún nombre, DNI, contacto, candidatura o posición viaja por estos
  contratos.
- Las referencias a Bolsa o procedimiento proceden de evidencia gobernada, no
  del cliente.
- Texto de proveedor y causas privadas no se guardan ni se devuelven.
- Los errores públicos distinguen solicitud inválida, denegación, conflicto,
  resultado no confiable e indisponibilidad sin filtrar detalles internos.
- Contexto, catálogo, política, preparación global, evidencias y concesión VEC
  se revalidan justo antes de fabricar la orden; O4-04 repetirá la revalidación
  y consumirá las capacidades dentro del `COMMIT`.
- El motivo conservado por el dominio compromete catálogo, versión, huella,
  entrada y clave i18n, pero no acredita autoridad externa por sí solo. O4-03C
  debe reconsultar la publicación exacta y comprobar que la entrada está
  vigente y declara esa clave i18n; O4-04 debe repetirlo antes de cualquier
  efecto durable. Una referencia rellenada por el llamador nunca basta.
- Web, escritorio, CLI y MCP invocarán el mismo caso de uso, sin cookies como
  autoridad.

## Puerta de cierre O4-03

Se exige probar propuesta determinista, ausencia, contradicción, catálogo y
política adulterados, decisión alternativa motivada, rectificación segregada,
replay, colisión semántica, CAS, cancelación, concurrencia, redacción y copia
defensiva. Deben quedar verdes pruebas unitarias, carrera, `go vet`, pruebas
globales, límites, secretos y revisión independiente.

El cierre técnico no autoriza producción. Permanecen O4-04, O4-05 y las
conformidades formales de RRHH, Jurídico, DPD, ENS, ENI y EIPD.
