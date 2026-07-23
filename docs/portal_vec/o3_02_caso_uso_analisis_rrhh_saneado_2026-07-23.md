# O3-02 — caso de uso saneado de análisis RRHH

Fecha: 23 de julio de 2026.

Estado: candidato técnico corregido tras el segundo `NO-GO`, pendiente de una
nueva revisión independiente e integración. No cierra O3-02, no habilita datos
reales y no acredita producción, ENS, ENI ni conformidad jurídica.

## Alcance

Este corte implementa la coordinación de aplicación para registrar y
rectificar el análisis de un expediente de contratación temporal. Conserva la
arquitectura hexagonal y no introduce HTTP, cookies, SQL, SDK de proveedor ni
almacenamiento del navegador.

Incluye:

- comandos neutrales para web, escritorio, CLI y MCP;
- contexto de identidad de garantía alta resuelto por la frontera VEC;
- artefacto interno opaco acuñado primero desde pruebas O3-03 verificadas y
  consumido únicamente después de reserva, política y autorización;
- recuperación idempotente temprana de una confirmación exacta, incluso si las
  fuentes O3-03 ya no están disponibles;
- orden indivisible y puerto único para consumir conjuntamente RC y coste;
- idempotencia semántica con HMAC rotatorio;
- política gobernada, versionada y ligada al estado previo;
- autorización VEC V3 exacta;
- control optimista de versión y rectificación motivada;
- segregación obligatoria entre quien registró y quien rectifica;
- orden opaca de persistencia atómica;
- recibo verificable, cancelación y errores públicos sin causas privadas.

No incluye:

- adaptador PostgreSQL de preparación o confirmación;
- adaptadores reales de RC, coste, verificación TCB, catálogo, credenciales ni
  consumo durable;
- adaptador durable del consumo conjunto RC+coste y de la consulta idempotente
  temprana; ambos pertenecen a O3-04;
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
confirmaciones, credenciales, raíces, recibos, actor, unidad ni actuación.
Tampoco existe ya un constructor público que acepte
`DatosArtefactoAnalisis`: una proyección nominal, aunque sea internamente
coherente, no puede acuñar autoridad.

`CapacidadPrepararArtefactoAnalisisO3` obtiene las peticiones selladas desde
infraestructura interna, invoca los puertos O3-03, contrasta credenciales y
pruebas de posesión contra la confianza fijada por la composición, verifica
respuesta, TCB y catálogo, presenta un desafío nuevo a cada autoridad y acuña
un artefacto verificado todavía sin consumo. El caso de uso solo solicita el
consumo conjunto después de reservar la intención, validar la política y
obtener la concesión VEC V3 aplicable.

El constructor se llama
`NuevaCapacidadPrepararArtefactoAnalisisO3ParaComposicionInterna`. Debe
invocarse únicamente desde la raíz de composición confiable. Está exportado
porque esa raíz vive en otro paquete Go y el árbol `internal` impide su uso
desde otros módulos, pero Go no impide por sí solo que otro paquete de este
mismo módulo lo invoque. Por ello la restricción interna se completa con
revisión de dependencias y con la prohibición probada de que transportes, DTO,
cabeceras, cookies o comandos acepten raíces, confianza o credenciales. No se
afirma una separación que el lenguaje no pueda imponer.

El artefacto conserva y liga:

- organización, expediente, versión y campos funcionales exactos;
- petición y HMAC de petición de RC y, cuando existe, coste;
- respuesta, huella, atestación, generación y ventana de vigencia;
- confirmaciones TCB y, en RC negativa, publicación gobernada;
- fuente, verificador y publicador con raíz, autoridad, backend, rol, serie,
  generación, huella de clave y vigencia de credencial;
- orden y recibo de consumo conjunto, con RC y coste confirmados en un mismo
  instante atómico;
- instante de preparación y huella SHA-256 determinista del contenido.

Esta huella es una dirección de contenido, no una segunda autoridad
criptográfica. La autoridad procede de O3-03. Los motivos negativos de RC
conservan por separado la entrada del catálogo y la clave i18n; no se exige
que sean el mismo identificador ni se aceptan textos libres como autoridad.
La huella estable del artefacto no cambia al incorporar el recibo de consumo:
el recibo queda ligado estructuralmente por la prueba opaca y la orden final,
no se acepta desde el cliente y no actúa como autoridad independiente.

## Secuencia de la operación

1. Validar el comando y reservar margen para el incremento CAS.
2. Resolver y revalidar el contexto de actor desde la frontera confiable.
3. Consultar una confirmación durable mediante la identidad semántica aportada
   por el cliente y el actor ya resuelto. Un replay exacto devuelve el recibo
   antes de releer O3-03; la misma clave con otra semántica produce conflicto.
4. Si no existe confirmación, resolver internamente las peticiones selladas de
   RC y coste.
5. Verificar credenciales, prueba de posesión, respuesta, TCB y publicación,
   sin consumir todavía ninguna respuesta.
6. Revalidar las autoridades con desafíos nuevos, comprobar vigencia y acuñar
   el artefacto verificado sin recibos de consumo.
7. Construir y sellar las preimágenes de ámbito y semántica.
8. Reservar o recuperar la intención idempotente. Una recuperación confirmada
   en esta segunda barrera tampoco repite el consumo.
9. Resolver y validar la política gobernada para fase y estado previos.
10. Exigir siempre actor distinto en una rectificación.
11. Obtener y revalidar la concesión VEC V3 ligada al recurso exacto.
12. Consumir RC y coste mediante una sola
    `OrdenConsumoConjuntoFuentesAnalisisO3` y una sola invocación a
    `ConsumidorConjuntoFuentesAnalisisO3`.
13. Revalidar la ventana de la evidencia e incorporar al artefacto el recibo
    conjunto completo.
14. Reproducir `RegistrarAnalisis` o `RectificarAnalisis` sobre una copia del
    agregado anterior.
15. Construir una orden que vuelve a ejecutar la transición y exige igualdad
    completa del expediente resultante.
16. Delegar el efecto en la transacción durable y validar que
    `ConfirmadaEn` siga dentro del contexto y de la concesión VEC V3; un recibo
    posterior, aunque sea estructuralmente correcto, se rechaza.

El puerto conjunto exige una sola transacción durable: no existe éxito parcial
solo para RC o solo para coste. El replay de la misma referencia y huella debe
devolver exactamente el mismo recibo completo; la misma referencia con otra
huella debe devolver `ErrConjuntoFuentesAnalisisYaConsumido`. O3-02 define y
prueba el contrato, pero no implementa su persistencia PostgreSQL.

La futura transacción O3-04 debe cerrar el CAS del agregado y de la política,
la idempotencia, el historial de solo adición, auditoría, recibo y outbox, y
dar recuperación inequívoca tras un resultado indeterminado. Este corte no
afirma atomicidad entre el consumo conjunto O3-03 y el `COMMIT` final O3-04:
esa garantía sigue pendiente del adaptador durable y de sus pruebas de caída,
reinicio y concurrencia.

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
- La huella del artefacto cubre cada coordenada funcional y probatoria; la
  preimagen semántica cubre además esa huella y el contenido completo.
- Una credencial válida que cambia serie, generación, backend, clave o ventana
  entre verificación y consumo se rechaza antes de producir efectos.
- Una denegación de política, segregación o VEC V3 no consume RC ni coste.
- El replay exacto confirmado se recupera antes de consultar fuentes; las
  fuentes caídas no impiden devolver su recibo.
- El consumo conjunto no puede confirmar solo RC cuando la orden incluye
  coste; el replay exacto recupera el mismo recibo y la misma referencia con
  otra respuesta produce conflicto explícito.
- La fábrica de la orden reproduce la operación de dominio sobre el estado
  previo y compara el agregado completo, no una selección de campos.
- El instante del recibo final debe permanecer dentro de la vigencia del
  contexto y de la concesión VEC V3.
- La rectificación exige siempre un actor diferente; ninguna política puede
  rebajar esa segregación.
- Se rechaza rectificar cuando ya existe cobertura o asignación materializada.
- Las versiones, secuencias y enteros canónicos se limitan a `2^53-1`; una
  versión de expediente que deba incrementarse debe ser estrictamente menor.

### Denegación y disponibilidad

- Una denegación válida se distingue de la caída de una dependencia.
- Un resultado mal formado se clasifica como no confiable, nunca como
  denegación ni éxito.
- La indisponibilidad no produce RC válida, coste, autorización ni recibo.
- El caso de uso completo tiene un presupuesto global máximo de cinco
  segundos; los presupuestos locales quedan subordinados al contexto padre.
- La cancelación previa impide efectos; una cancelación posterior a un
  `COMMIT` confirmado no convierte el éxito durable en resultado ambiguo.

### Encapsulación

Artefacto, pruebas O3-03, preparación, orden y evidencia rechazan codificación
y decodificación JSON, XML, texto, binario, gob, CBOR y YAML. Esta barrera
evita que un adaptador reconstruya autoridad interna a partir de una carga
externa.

## Evidencia automatizada

Las pruebas del corte cubren:

- registro y rectificación correctos;
- integración real del caso de uso con peticiones, fuentes, credenciales,
  atestaciones, TCB, consumo y derivación de RC y coste;
- ausencia del constructor nominal público y rechazo de una proyección
  nominal autoconsistente;
- revalidación de credenciales con desafío nuevo;
- adulteración de atestación, confirmación, credencial, catálogo y recibo;
- replay exacto, conflicto por respuesta distinta y caducidad tras consumo;
- recuperación temprana con O3-03 caído;
- orden e invocación únicas para RC y coste, rechazo del recibo parcial y
  timestamp atómico común;
- copia profunda de la confirmación de publicación al exportar las pruebas;
- segregación y motivo gobernado;
- imposibilidad de desactivar la segregación desde la política;
- ligaduras del artefacto y del recurso VEC V3;
- repetición idempotente y conflicto semántico;
- adulteración de un campo adicional del expediente resultante;
- cobertura previa ya materializada;
- denegación frente a indisponibilidad;
- redacción de causas internas;
- resultados opacos o políticas no confiables;
- cancelación antes y después del efecto;
- recibo final fuera de la ventana del contexto y la concesión;
- presupuesto global de cinco segundos;
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

1. Revisión independiente del puente O3-02/O3-03.
2. O3-04: preparación y confirmación PostgreSQL atómicas, con roles y ACL,
   concurrencia, reintento, reinicio y resultado indeterminado.
3. Composición real y, después, API, interfaz y E2E de O3-05.
