# O3-02 — caso de uso saneado de análisis RRHH

Fecha: 23 de julio de 2026.

Estado: candidato técnico pendiente de revisión independiente e integración.
No cierra O3-02, no habilita datos reales y no acredita producción, ENS, ENI
ni conformidad jurídica.

## Alcance

Este corte implementa la coordinación de aplicación para registrar y
rectificar el análisis de un expediente de contratación temporal. Conserva la
arquitectura hexagonal y no introduce HTTP, cookies, SQL, SDK de proveedor ni
almacenamiento del navegador.

Incluye:

- comandos neutrales para web, escritorio, CLI y MCP;
- contexto de identidad de garantía alta resuelto por la frontera VEC;
- artefacto interno opaco preparado por el puerto O3-03;
- idempotencia semántica con HMAC rotatorio;
- política gobernada, versionada y ligada al estado previo;
- autorización VEC V3 exacta;
- control optimista de versión y rectificación motivada;
- segregación configurable entre quien registró y quien rectifica;
- orden opaca de persistencia atómica;
- recibo verificable, cancelación y errores públicos sin causas privadas.

No incluye:

- implementación ni aprobación de O3-03;
- adaptador PostgreSQL de preparación o confirmación;
- auditoría, outbox e historia durable reales;
- composición, API, interfaz o pruebas E2E;
- decisión de cumplimiento ni autorización para tratar datos personales.

## Frontera de autoridad

El cliente solo aporta referencias de autenticación, sesión y perfil, las
coordenadas del expediente, la clave de idempotencia, la referencia del
artefacto y los campos funcionales mínimos:

- modalidad;
- categoría y grupo/subgrupo;
- causa;
- periodo;
- jornada en diezmilésimas;
- referencia y huella de la entrada de retención de crédito.

El cliente no puede construir `AnalisisRRHH`, `ValidacionRC`, coste, fuentes,
recibos, actor, unidad ni actuación. El puerto `PreparadorArtefactoAnalisisO3`
entrega un valor opaco no serializable. La aplicación deriva desde él la
proyección autoritativa y el dominio vuelve a validar sus invariantes.

El artefacto queda ligado a organización, expediente, versión, entrada
funcional, resultado de RC, fuentes, recibos, instantes, coste y huella. Los
motivos negativos de RC y de rectificación usan entradas de catálogo
versionadas y claves i18n coincidentes; no se aceptan textos libres como
autoridad.

## Secuencia de la operación

1. Validar el comando y reservar margen para el incremento CAS.
2. Resolver y revalidar el contexto de actor desde la frontera confiable.
3. Obtener el artefacto opaco O3-03 para las coordenadas exactas.
4. Construir preimágenes separadas de ámbito idempotente y semántica.
5. Sellarlas mediante un puerto HMAC con generaciones alineadas.
6. Reservar o recuperar la operación idempotente.
7. Resolver la política gobernada para fase y estado previos exactos.
8. Comprobar segregación de funciones.
9. Obtener y revalidar una concesión VEC V3 ligada al recurso exacto.
10. Reproducir `RegistrarAnalisis` o `RectificarAnalisis` sobre una copia del
    agregado anterior.
11. Construir una orden que vuelve a ejecutar la transición y exige igualdad
    completa del expediente resultante.
12. Delegar el único efecto en una transacción durable y validar su recibo.

La futura transacción debe ejecutar conjuntamente el CAS del agregado y de la
política, el consumo único del artefacto y de la concesión VEC, la
idempotencia, el historial de solo adición, auditoría, recibo y outbox.

## Controles incorporados

### Minimización y privacidad

- Los DTO no aceptan el agregado autoritativo ni datos identificativos.
- Las causas privadas de conectores nunca aparecen en errores o valores de
  log.
- Las pruebas usan exclusivamente referencias y contenido sintéticos.
- Los tipos de evidencia interna muestran una representación opaca.

### Integridad y trazabilidad

- La referencia y huella del artefacto participan en la semántica HMAC, en la
  preparación, en la política, en el recurso VEC y en el recibo.
- La fábrica de la orden reproduce la operación de dominio sobre el estado
  previo y compara el agregado completo, no una selección de campos.
- Se rechaza rectificar cuando ya existe cobertura o asignación materializada.
- Las versiones, secuencias y enteros canónicos se limitan a `2^53-1`; una
  versión de expediente que deba incrementarse debe ser estrictamente menor.

### Denegación y disponibilidad

- Una denegación válida se distingue de la caída de una dependencia.
- Un resultado mal formado se clasifica como no confiable, nunca como
  denegación ni éxito.
- La indisponibilidad no produce RC válida, coste, autorización ni recibo.
- La cancelación previa impide efectos; una cancelación posterior a un
  `COMMIT` confirmado no convierte el éxito durable en resultado ambiguo.

### Encapsulación

Artefacto, preparación, orden y evidencia rechazan codificación y
decodificación JSON, XML, texto, binario, gob, CBOR y YAML. Esta barrera evita
que un adaptador reconstruya autoridad interna a partir de una carga externa.

## Evidencia automatizada

Las pruebas del corte cubren:

- registro y rectificación correctos;
- derivación interna de RC y coste;
- segregación y motivo gobernado;
- ligaduras del artefacto y del recurso VEC V3;
- repetición idempotente y conflicto semántico;
- adulteración de un campo adicional del expediente resultante;
- cobertura previa ya materializada;
- denegación frente a indisponibilidad;
- redacción de causas internas;
- resultados opacos o políticas no confiables;
- cancelación antes y después del efecto;
- dependencias con nulo tipado;
- límites de versión y catálogo;
- bloqueo de todos los codecs previstos.

## Trazado normativo

Este diseño aporta evidencia técnica parcial para:

- RGPD y LOPDGDD: minimización, integridad, exactitud y responsabilidad
  proactiva;
- ENS: privilegio mínimo, denegación predeterminada, autenticidad,
  trazabilidad y segregación;
- ENI, leyes 39/2015 y 40/2015: expediente versionado, actor, instante, motivo,
  recibo e historia;
- normativa de igualdad y empleo público: revisión humana y rectificación
  motivada;
- Reglamento de IA: este corte no usa IA ni adopta decisiones automatizadas.

Las aprobaciones CT-CUM-02 a CT-CUM-10 siguen pendientes. La revisión de código
no sustituye las decisiones del DPD, Seguridad, Sistemas, Jurídico, Archivo,
Intervención o RRHH.

## Siguiente trabajo

1. Revisión independiente del corte saneado.
2. Cierre independiente de O3-03.
3. O3-04: preparación y confirmación PostgreSQL atómicas, con roles y ACL,
   concurrencia, reintento, reinicio y resultado indeterminado.
4. Composición real y, después, API, interfaz y E2E de O3-05.
